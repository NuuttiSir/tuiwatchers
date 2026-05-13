package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/coder/websocket"
)

type ServerMessage struct {
	Metadata struct {
		MessageID        string    `json:"message_id"`
		MessageType      string    `json:"message_type"`
		MessageTimestamp time.Time `json:"message_timestamp"`
	} `json:"metadata"`
	Payload struct {
		Session struct {
			ID                      string      `json:"id"`
			Status                  string      `json:"status"`
			ConnectedAt             time.Time   `json:"connected_at"`
			KeepaliveTimeoutSeconds int         `json:"keepalive_timeout_seconds"`
			ReconnectURL            interface{} `json:"reconnect_url"`
			RecoveryURL             interface{} `json:"recovery_url"`
		} `json:"session"`
	} `json:"payload"`
}

func welcomeMessage() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	conn, _, err := websocket.Dial(ctx, "wss://eventsub.wss.twitch.tv/ws", nil)
	if err != nil {
		fmt.Println(err)
	}
	defer conn.CloseNow()

	for {
		_, msg, err := conn.Read(ctx)
		if err != nil {
			fmt.Println(err)
		}

		var serverMessage ServerMessage
		err = json.Unmarshal(msg, &serverMessage)
		if err != nil {
			fmt.Println(err)
		}

		json, err := json.MarshalIndent(serverMessage, "", " ")
		if err != nil {
			fmt.Println(err)
		}

		if serverMessage.Metadata.MessageType == "session_welcome" {
			fmt.Printf("Got welcome message: %v", string(json))
			fmt.Println()
			conn.Close(websocket.StatusNormalClosure, "")
			return
		} else {
			fmt.Println("Did not get welcome message lol xd what the fuck")
			conn.Close(websocket.StatusNormalClosure, "")
			return
		}
	}
}

func postSubscribe(clientID, accessToken, userID, braodcasterID string) {
	req, err := http.NewRequest("POST", "https://api.twitch.tv/helix/eventsub/subscriptions", nil)
	if err != nil {
		fmt.Println(err)
	}
	req.Header.Set("Authorization", "OAuth "+accessToken)

}
