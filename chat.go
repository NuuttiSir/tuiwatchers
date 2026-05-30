// TODO: MAKE THE NEW CHAT TERMINAL OPEN ON THE RIGHT SIDE OF THE MONITOR
// <CHECK player.go file for instructions>
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	tea "charm.land/bubbletea/v2"
	"github.com/coder/websocket"
	"golang.org/x/term"
)

type Metadata struct {
	MessageID        string    `json:"message_id"`
	MessageType      string    `json:"message_type"`
	MessageTimestamp time.Time `json:"message_timestamp"`
}

type MessagePayload struct {
	Session *Session   `json:"session"`
	Event   *ChatEvent `json:"event"`
}

type Session struct {
	ID                      string    `json:"id"`
	Status                  string    `json:"status"`
	ConnectedAt             time.Time `json:"connected_at"`
	KeepaliveTimeoutSeconds int       `json:"keepalive_timeout_seconds"`
	ReconnectURL            any       `json:"reconnect_url"`
	RecoveryURL             any       `json:"recovery_url"`
}

type ServerMessage struct {
	Metadata       Metadata       `json:"metadata"`
	MessagePayload MessagePayload `json:"payload"`
}

type Condition struct {
	BroadcasterUserID string `json:"broadcaster_user_id"`
	UserID            string `json:"user_id"`
}

type Transport struct {
	Method    string `json:"method"`
	SessionID string `json:"session_id"`
}

type SubscriptionRequest struct {
	Type      string    `json:"type"`
	Version   string    `json:"version"`
	Condition Condition `json:"condition"`
	Transport Transport `json:"transport"`
}

// Fragment is a fragment of a chat message
// Twitch splits messages into fragments
type Fragment struct {
	Type  string `json:"type"`
	Text  string `json:"text"`
	Emote Emote  `json:"emote"`
}

type Emote struct {
	ID         string `json:"id"`
	EmoteSetID string `json:"emote_set_id"`
	OwnerID    string `json:"owner_id"`
}

type MessagePart struct {
	Kind    string // Text or emote
	Text    string
	EmoteID string
}

type ChatMessage struct {
	Text      string     `json:"text"`
	Fragments []Fragment `json:"fragments"`
}

// ChatEvent is a event data for channel.chat.message notification
type ChatEvent struct {
	BroadcasterUserID    string      `json:"broadcaster_user_id"`
	BroadcasterUserLogin string      `json:"broadcaster_user_login"`
	BroadcasterUserName  string      `json:"broadcaster_user_name"`
	ChatterUserID        string      `json:"chatter_user_id"`
	ChatterUserLogin     string      `json:"chatter_user_login"`
	ChatterUserName      string      `json:"chatter_user_name"`
	MessageID            string      `json:"message_id"`
	Message              ChatMessage `json:"message"`
	Color                string      `json:"color"`
}

type SendChatMessage struct {
	BroadcasterID string `json:"broadcaster_id"`
	SenderID      string `json:"sender_id"`
	Message       string `json:"message"`
}

type ReceivedChatMessageAnswer struct {
	MessageID  string     `json:"message_id"`
	IsSent     bool       `json:"is_sent"`
	DropReason DropReason `json:"drop_reason"`
}

type DropReason struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type RawTerminal struct {
	FileDescriptor int
	Old            *term.State
	Width          int
	Height         int
	ChatRows       int
	mu             sync.Mutex
}

type IncomingChatMessage struct {
	User  string
	Parts []MessagePart
	Text  string
}

type ChatLine struct {
	User  string
	Parts []MessagePart
}

type SendResultMessage struct {
	Ok  bool
	Err error
}

type ClearStatusMessage struct{}

type EmoteLoadedMsg struct {
	ID      string
	Success bool
}

func sendChatMessage(broadcasterID, userID, accessToken, message string) ReceivedChatMessageAnswer {
	data := SendChatMessage{
		BroadcasterID: broadcasterID,
		SenderID:      userID,
		Message:       message,
	}
	body, err := json.Marshal(data)
	if err != nil {
		fmt.Println(err)
		return ReceivedChatMessageAnswer{}
	}

	req, err := http.NewRequest("POST", "https://api.twitch.tv/helix/chat/messages", bytes.NewBuffer(body))
	if err != nil {
		fmt.Println(err)
		return ReceivedChatMessageAnswer{}
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Client-Id", clientID)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		return ReceivedChatMessageAnswer{}
	}
	defer resp.Body.Close()

	var chatMessageAnswer ReceivedChatMessageAnswer
	dat, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
		return ReceivedChatMessageAnswer{}
	}
	err = json.Unmarshal(dat, &chatMessageAnswer)
	if err != nil {
		fmt.Println(err)
		return ReceivedChatMessageAnswer{}
	}

	return ReceivedChatMessageAnswer{
		MessageID: chatMessageAnswer.MessageID,
		IsSent:    chatMessageAnswer.IsSent,
	}

}

func sendChatCommand(broadcasterID, userID, accessToken, message string) tea.Cmd {
	return func() tea.Msg {
		resp := sendChatMessage(broadcasterID, userID, accessToken, message)
		if !resp.IsSent {
			return SendResultMessage{
				Ok:  false,
				Err: errors.New(resp.DropReason.Message),
			}
		}
		return SendResultMessage{Ok: true}

	}
}

func postSubscribe(clientID, userID, broadcasterID, sessionID, accessToken string) {
	data := SubscriptionRequest{
		Type:    "channel.chat.message",
		Version: "1",
		Condition: Condition{
			BroadcasterUserID: broadcasterID,
			UserID:            userID,
		},
		Transport: Transport{
			Method:    "websocket",
			SessionID: sessionID,
		},
	}

	body, err := json.Marshal(data)
	if err != nil {
		fmt.Println(err)
		return
	}

	req, err := http.NewRequest("POST", "https://api.twitch.tv/helix/eventsub/subscriptions", bytes.NewBuffer(body))
	if err != nil {
		fmt.Println(err)
		return
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Client-Id", clientID)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer resp.Body.Close()
}

func connectAndListen(ctx context.Context, out chan<- IncomingChatMessage, broadcasterID, userID, accessToken string) {
	// Open WebSocket connection
	conn, _, err := websocket.Dial(ctx, "wss://eventsub.wss.twitch.tv/ws", nil)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer conn.CloseNow()

	// Start infinite read loop
	// We keep listening forever because Twitch will keep sending us messages
	for {
		// Read next message from Twitch
		// This BLOCKS meaning it waits until next message comes
		_, msg, err := conn.Read(ctx)
		if err != nil {
			fmt.Println("read err: ", err)
			return
		}

		// Parse the raw message into usable struct
		var serverMessage ServerMessage
		err = json.Unmarshal(msg, &serverMessage)
		if err != nil {
			fmt.Println("Unmarshaling err: ", err)
			continue
		}

		// Check what TYPE of message Twitch sent
		switch serverMessage.Metadata.MessageType {

		case "session_welcome":
			// First message Twitch sends
			// Includes SESSION ID
			if serverMessage.MessagePayload.Session == nil {
				fmt.Println("welcome message has no session")
				continue
			}
			sessionID := serverMessage.MessagePayload.Session.ID

			// SUBSCRIBE immediately with the SESSION ID
			postSubscribe(clientID, userID, broadcasterID, sessionID, accessToken)

		case "session_keepalive":
			// Twitch sends these periodically to inform its still here
			// ATM do nothing and keep looping
			// If these dont come we may have dropped connection
			continue

		case "notification":
			// This is the actual message, WE CARE ABOUT THIS
			if serverMessage.MessagePayload.Event == nil {
				fmt.Println("Notification has no event")
				continue
			}
			event := serverMessage.MessagePayload.Event
			username := event.ChatterUserName

			// Twitch message is just many fragments
			var chatMessageParts []MessagePart
			for _, fragment := range event.Message.Fragments {
				switch fragment.Type {
				case "text":
					chatMessageParts = append(chatMessageParts, MessagePart{
						Kind: "text",
						Text: fragment.Text,
					})
				case "emote":
					chatMessageParts = append(chatMessageParts, MessagePart{
						Kind:    "emote",
						Text:    fragment.Text,
						EmoteID: fragment.Emote.ID,
					})

				}
			}

			// I dont think this == 0 ever fires as I dont think twitch chat message ever has no fragments
			if len(chatMessageParts) == 0 {
				out <- IncomingChatMessage{
					User: username,
					Text: event.Message.Text,
				}
			} else {
				out <- IncomingChatMessage{
					User:  username,
					Parts: chatMessageParts,
				}
			}

		case "session_reconnect":
			// Twitch wants us to reconnect, log it for now for funsies
			fmt.Println("Twitch has requested reconnect")
		}
	}
}

func spawnChatWindow(broadcasterID, userID, accessToken string) (*exec.Cmd, error) {
	cmd := exec.Command("/usr/bin/ghostty", "-e", "bash", "-c", "./tuiwatchers --chat "+clientID+" "+broadcasterID+" "+userID+" "+accessToken+";exec bash")

	err := cmd.Start()
	if err != nil {
		fmt.Println("Terminal window opening error", err)
		return nil, err
	}

	return cmd, err
}

func newRawTerm() (*RawTerminal, error) {
	fileDescriptor := int(os.Stdin.Fd())
	old, err := term.MakeRaw(fileDescriptor)
	if err != nil {
		return nil, err
	}
	width, height, err := term.GetSize(fileDescriptor)
	if err != nil {
		term.Restore(fileDescriptor, old)
		return nil, err
	}
	return &RawTerminal{FileDescriptor: fileDescriptor, Old: old, Width: width, Height: height, ChatRows: height - 3}, nil
}

func (rt *RawTerminal) restore() {
	fmt.Print("\x1b[r")
	fmt.Print("\x1b[?25h")
	fmt.Print("\x1b[2J/x1b[H")
	term.Restore(rt.FileDescriptor, rt.Old)
}

func (rt *RawTerminal) initScreen() {
	fmt.Print("\x1b[2J")                  // Clear screen
	fmt.Printf("\x1b[1;%dr", rt.ChatRows) // scroll region = 1..ChatRows
	fmt.Printf("\x1b[%d;1H", rt.Height)   // Cursor to input line on prompt
}

func (rt *RawTerminal) printMessage(line ChatLine) {
	rt.mu.Lock()
	defer rt.mu.Unlock()

	fmt.Print("\x1b[s")                       // save cursor (currently at input line)
	fmt.Printf("\x1b[%d;1H\r\n", rt.ChatRows) // go to last scroll line, emit newline → scrolls region
	fmt.Print("\x1b[2K")                      // clear the fresh blank line
	renderLine(line)                          // print username + parts
	fmt.Print("\x1b[u")                       // restore cursor to input line
}

func renderLine(line ChatLine) {
	fmt.Printf("\x1b[1m%s\x1b[0m: ", line.User) // bold username
	for _, part := range line.Parts {
		switch part.Kind {
		case "text":
			fmt.Print(part.Text)
		case "emote":
			if img, ok := emoteImage(part.EmoteID); ok {
				fmt.Print(kittyInlineImage(img), " ")
			} else {
				fmt.Printf("[%s]", part.Text) // fallback until cached
			}
		}
	}
}

func (rt *RawTerminal) inputLoop(broadcasterID, userID, accessToken string,
	quit chan struct{}) {
	var buf []rune
	raw := make([]byte, 4) // enough for any UTF-8 rune + escape sequences

	redraw := func() {
		rt.mu.Lock()
		fmt.Printf("\x1b[%d;1H\x1b[2K> %s", rt.Height, string(buf))
		rt.mu.Unlock()
	}

	for {
		n, _ := os.Stdin.Read(raw)
		if n == 0 {
			continue
		}

		switch raw[0] {
		case 3, 27: // Ctrl+C or Esc
			close(quit)
			return
		case 13: // Enter
			msg := strings.TrimSpace(string(buf))
			buf = buf[:0]
			redraw()
			if msg != "" {
				go sendChatMessage(broadcasterID, userID, accessToken,
					msg)
			}
		case 127: // Backspace (DEL)
			if len(buf) > 0 {
				buf = buf[:len(buf)-1]
				redraw()
			}
		default:
			r, size := utf8.DecodeRune(raw[:n])
			if size > 0 && r != utf8.RuneError && r >= 32 {
				if len(buf) < 500 { // Twitch message limit
					buf = append(buf, r)
					redraw()
				}
			}
		}
	}
}

func fetchEmoteBlocking(id string) {
	if _, ok := emoteImage(id); ok {
		return
	}

	if cache.FailedEmotes[id] {
		return
	}

	url := "https://static-cdn.jtvnw.net/emoticons/v2/" + id + "/static/light/1.0"
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		cache.FailedEmotes[id] = true
		return
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		cache.FailedEmotes[id] = true
		return
	}
	cache.Emotes[id] = data
}
