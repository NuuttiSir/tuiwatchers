package twitch

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/coder/websocket"
)

type IncomingChatMessage struct {
	User  string
	Parts []MessagePart
	Text  string
}
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
	ReconnectURL            string    `json:"reconnect_url"`
	RecoveryURL             any       `json:"recovery_url"`
}

type ServerMessage struct {
	Metadata       Metadata       `json:"metadata"`
	MessagePayload MessagePayload `json:"payload"`
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

func ConnectAndListen(ctx context.Context, out chan<- IncomingChatMessage, broadcasterID, userID, accessToken string) {
	// Open WebSocket connection
	wsURL := "wss://eventsub.wss.twitch.tv/ws"
	isReconnect := false
	for {
		nextURL, err := runSession(ctx, wsURL, isReconnect, out, broadcasterID, userID, accessToken)
		if err != nil {
			if ctx.Err() != nil {
				return // caller cancelled, exit quietly
			}
			// transient failure: retry the main endpoint with fresh subscribe
			wsURL = "wss://eventsub.wss.twitch.tv/ws"
			isReconnect = false
			time.Sleep(2 * time.Second)
			continue
		}
		wsURL = nextURL // session_reconenct gave new URL
		isReconnect = true
	}
}
func runSession(ctx context.Context, wsURL string, isReconnect bool, out chan<- IncomingChatMessage, broadcasterID, userID, accessToken string) (string, error) {
	conn, _, err := websocket.Dial(ctx, wsURL, nil)
	if err != nil {
		out <- IncomingChatMessage{User: "system", Text: fmt.Sprint(err)}
		return "Error: ", err
	}
	defer conn.CloseNow()

	// Start infinite read loop
	// We keep listening forever because Twitch will keep sending us messages
	for {
		// Read next message from Twitch
		// This BLOCKS meaning it waits until next message comes
		_, msg, err := conn.Read(ctx)
		if err != nil {
			out <- IncomingChatMessage{User: "system", Text: fmt.Sprint("read err: ", err)}
			return "Error: ", err
		}

		// Parse the raw message into usable struct
		var serverMessage ServerMessage
		err = json.Unmarshal(msg, &serverMessage)
		if err != nil {
			out <- IncomingChatMessage{User: "system", Text: fmt.Sprint("Unmarshaling err: ", err)}
			continue
		}

		// Check what TYPE of message Twitch sent
		switch serverMessage.Metadata.MessageType {

		case "session_welcome":
			// First message Twitch sends
			// Includes SESSION ID
			if serverMessage.MessagePayload.Session == nil {
				out <- IncomingChatMessage{User: "system", Text: "Welcome message has no session"}
				continue
			}
			sessionID := serverMessage.MessagePayload.Session.ID

			// SUBSCRIBE immediately with the SESSION ID if this is not a
			// reconnect so forst connect herp derp
			if !isReconnect {
				err := postSubscribe(ClientID, userID, broadcasterID, sessionID, accessToken)
				if err != nil {
					return "Error: ", err
				}
			}
			continue

		case "session_keepalive":
			// Twitch sends these periodically to inform its still here
			// ATM do nothing and keep looping
			// If these dont come we may have dropped connection
			continue

		case "notification":
			// This is the actual message, WE CARE ABOUT THIS
			if serverMessage.MessagePayload.Event == nil {
				out <- IncomingChatMessage{User: "system", Text: "Notification has no event"}
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
			return serverMessage.MessagePayload.Session.ReconnectURL, nil
		}
	}
}
