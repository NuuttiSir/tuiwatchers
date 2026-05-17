package main

import (
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
	AccessToken string `json:"access_token"`
	UserID      string `json:"user_id"`
}

// type validateResponse struct {
// 	ClientID  string `json:"client_id"`
// 	Login     string `json:"login"`
// 	UserID    string `json:"user_id"`
// 	ExpiresIn int    `json:"expires_in"`
// }

// func printFollowData(followDataList FollowDataList) {
// 	count := 0
// 	fmt.Println("Channels that are live")
// 	for _, ch := range followDataList.Data {
// 		if ch.Type == "live" {
// 			count += 1
// 			fmt.Printf("  - %s IS LIVE\n", ch.UserName)
// 		}
// 	}
// 	fmt.Println("Count of live streams: ", count)
// }

func openChat() {
	fmt.Println("CHAT")
	fmt.Println("Starting chat window")

	// Start the WebSocket listener in goroutine so it runs in background while MPV runs as well
	done := make(chan struct{})
	go func() {
		//ClientID, BroadcasterID, UserID, AccessToken
		connectAndListen(os.Args[2], os.Args[3], os.Args[4], os.Args[5])
		close(done)
	}()

	// Wait for the websocket goroutine to finish before exiting
	<-done
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

	spawnChatWindow(clientID, broadcasterID, finalModel.TokenFile.UserID, finalModel.TokenFile.AccessToken)
	startMPVWithStream(selectedChannel)
}
