package main

import (
	"context"
	"fmt"
	"os"
	"time"

	_ "charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
)

type DeviceCodeResponse struct {
	DeviceCode      string `json:"device_code"`
	Interval        int    `json:"interval"`
	UserCode        string `json:"user_code"`
	VerificationURI string `json:"verification_uri"`
}

type AccessToken struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
	TokenType   string `json:"token_type"`
}

type UserData struct {
	BroadcasterType string    `json:"broadcaster_type"`
	CreatedAt       time.Time `json:"created_at"`
	Description     string    `json:"description"`
	DisplayName     string    `json:"display_name"`
	ID              string    `json:"id"`
	Login           string    `json:"login"`
	OfflineImageURL string    `json:"offline_image_url"`
	ProfileImageURL string    `json:"profile_image_url"`
	Type            string    `json:"type"`
	ViewCount       int       `json:"view_count"`
}

type UserDataList struct {
	Data []UserData `json:"data"`
}

type FollowData struct {
	ID           string    `json:"id"`
	UserID       string    `json:"user_id"`
	UserLogin    string    `json:"user_login"`
	UserName     string    `json:"user_name"`
	GameID       string    `json:"game_id"`
	GameName     string    `json:"game_name"`
	Type         string    `json:"type"`
	Title        string    `json:"title"`
	ViewerCount  int       `json:"viewer_count"`
	StartedAt    time.Time `json:"started_at"`
	Language     string    `json:"language"`
	ThumbnailURL string    `json:"thumbnail_url"`
	TagIds       []any     `json:"tag_ids"`
	Tags         []string  `json:"tags"`
}

type FollowDataList struct {
	Data       []FollowData `json:"data"`
	Pagination struct {
		Cursor string `json:"cursor"`
	} `json:"pagination"`
}

type TokenFile struct {
	ClientId     string
	AccessToken  string `json:"access_token"`
	UserID       string `json:"user_id"`
	RefreshToken string `json:"refresh_token"`
}

func openChat() {
	broadcasterID := os.Args[3]
	userID := os.Args[4]
	accessToken := os.Args[5]

	fmt.Println("CHAT")
	fmt.Println("Starting chat window")

	chatModel := InitialChatModel(broadcasterID, userID, accessToken)
	program := tea.NewProgram(chatModel)

	// channel for incoming messages
	incoming := make(chan IncomingChatMessage, 50)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	//websocket listener
	go connectAndListen(ctx, incoming, broadcasterID, userID, accessToken)

	go func() {
		for msg := range incoming {
			program.Send(msg)
		}
	}()

	_, err := program.Run()
	if err != nil {
		fmt.Println(err)
	}
	cancel()
}

func main() {
	if len(os.Args) >= 2 {
		switch os.Args[1] {
		case "--chat":
			openChat()
			return
		}
	}

	model := initialModel()
	program := tea.NewProgram(model)
	selectedChannel, err := program.Run()
	if err != nil {
		fmt.Printf("Whoops an error has occurred: %v", err)
		os.Exit(1)
	}

	finalModel, ok := selectedChannel.(Model)
	if !ok {
		fmt.Println("Could not cast model")
		return
	}

	if finalModel.Err != nil {
		fmt.Println(finalModel.Err)
		return
	}

	broadcasterID := finalModel.BroadcasterIDs[finalModel.SelectedChannel]
	if broadcasterID == "" {
		fmt.Println("Could not find broadcaster ID for selected channel")
		return
	}

	spawnChatWindow(broadcasterID, finalModel.TokenFile.UserID, finalModel.TokenFile.AccessToken)
	startMPVWithStream(selectedChannel)
}
