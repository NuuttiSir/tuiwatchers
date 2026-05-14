// TODO: ADD A WAY TO SEND MESSAGES
// TODO: MAKE IT CLEANER WITH BUBBLETEA
// TODO: MAKE A INPUT FIELD AND MAKE IT KIND OF FULLSCREEN
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"time"

	"github.com/coder/websocket"
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
	Type string `json:"type"`
	Text string `json:"text"`
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

func connectAndListen(clientID, broadcasterID, userID, accessToken string) {
	// Open WebSocket connection
	ctx := context.Background()

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

			// SUBSCRIBE immidiately with the SESSION ID
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
			chatMessage := event.Message.Text
			username := event.ChatterUserName

			fmt.Printf("%s: %s\n", username, chatMessage)

		case "sessions_reconnect":
			// Twitch wants us to reconnect, log it for now for funsies
			fmt.Println("Twitch has requested reconnect")
		}
	}
}

func spawnChatWindow(clientID, broadcasterID, userID, accessToken string) {
	fmt.Println("In chat func")

	cmd := exec.Command("/usr/bin/ghostty", "-e", "bash", "-c", "./tuiwatchers --chat "+clientID+" "+broadcasterID+" "+userID+" "+accessToken+";exec bash")
	err := cmd.Start()
	if err != nil {
		fmt.Println("Terminal window opening error", err)
		return
	}
}
