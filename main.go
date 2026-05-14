package main

import (
	"errors"
	"fmt"
	"os"
	"time"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"github.com/joho/godotenv"
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

type validateResponse struct {
	ClientID  string `json:"client_id"`
	Login     string `json:"login"`
	UserID    string `json:"user_id"`
	ExpiresIn int    `json:"expires_in"`
}

func printFollowData(followDataList FollowDataList) {
	count := 0
	fmt.Println("Channels that are live")
	for _, ch := range followDataList.Data {
		if ch.Type == "live" {
			count += 1
			fmt.Printf("  - %s IS LIVE\n", ch.UserName)
		}
	}
	fmt.Println("Count of live streams: ", count)
}

// TODO: Get pagination to work
func main() {
	godotenv.Load()
	clientID := os.Getenv("CLIENT_ID")
	tokenFilePath := "tokens.json"

	switch os.Args[1] {
	case "--chat":
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

		return
	}

	// Check if tokens.json exists and if not make the file
	if _, err := os.Stat(tokenFilePath); errors.Is(err, os.ErrNotExist) {
		fmt.Println("tokens.json not found... Creating")
		if err := saveToken(tokenFilePath, "", ""); err != nil {
			fmt.Println("err creating token file:", err)
		}
	}

	tokenFile, err := tokenLoad(tokenFilePath)
	if err != nil {
		fmt.Println("err loading token file:", err)
	}

	if !validateToken(tokenFile.AccessToken) {
		fmt.Println("Need to re-auth")

		userToken := getUserToken(clientID)
		if userToken.AccessToken == "" {
			fmt.Println("Authentication failed.")
			return
		}

		authUser := getAuthenticatedUser(clientID, userToken)
		if authUser.ID == "" {
			fmt.Println("Could not fetch user data.")
			return
		}

		if err := saveToken(tokenFilePath, userToken.AccessToken, authUser.ID); err != nil {
			fmt.Println("err saving token:", err)
		}
	}

	tokenFile, err = tokenLoad(tokenFilePath)
	if err != nil {
		fmt.Println("err loading token file:", err)
	}

	fmt.Println("Token is valid, reusing saved session.")
	followDataList := getFollowedChannels(tokenFile.UserID, clientID, AccessToken{AccessToken: tokenFile.AccessToken})

	var channels []list.Item
	for _, channel := range followDataList.Data {
		if channel.Type == "live" {
			channels = append(channels, item{
				title:    channel.UserName,
				gameName: channel.GameName,
				desc:     channel.Title,
			})
		}
	}

	ownStyles := newStyles()
	m := model{
		list: list.New(channels, itemDelegate{styles: ownStyles}, 0, 0),
	}
	m.list.Title = "Channels that are live"

	program := tea.NewProgram(m)
	selectedChannel, err := program.Run()
	if err != nil {
		fmt.Printf("Whoops an error has occurred: %v", err)
		os.Exit(1)
	}

	finalModel, ok := selectedChannel.(model)
	if !ok {
		fmt.Println("Could not cast model")
		return
	}

	// Get the broadcaster ID from the selected channel
	// We search followDataList to match the channel the user picked in the UI
	var broadcasterID string
	for _, followedChannel := range followDataList.Data {
		if followedChannel.UserName == finalModel.selectedChannel {
			broadcasterID = followedChannel.UserID
			break
		}
	}

	if broadcasterID == "" {
		fmt.Println("Could not find broadcaster ID for selected channel")
		return
	}

	spawnChatWindow(clientID, broadcasterID, tokenFile.UserID, tokenFile.AccessToken)
	startMPVWithStream(selectedChannel)
}
