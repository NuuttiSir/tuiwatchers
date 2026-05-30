// NOTE: Cant use refreshtokens because they need client secret and cant give that publicly
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	tea "charm.land/bubbletea/v2"
)

type DeviceCodeResponse struct {
	DeviceCode      string `json:"device_code"`
	Interval        int    `json:"interval"`
	UserCode        string `json:"user_code"`
	VerificationURI string `json:"verification_uri"`
}

type AccessToken struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
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
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	UserID       string `json:"user_id"`
}

func openChat() {
	broadcasterID := os.Args[3]
	userID := os.Args[4]
	accessToken := os.Args[5]

	rt, err := newRawTerm()
	if err != nil {
		fmt.Println(err)
		return
	}
	defer rt.restore()

	rt.initScreen()

	quit := make(chan struct{})
	incoming := make(chan IncomingChatMessage, 50)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go connectAndListen(ctx, incoming, broadcasterID, userID, accessToken)
	go func() {
		for msg := range incoming {
			// fetch emotes synchronously before printing (Phase 6, Option A)
			for _, part := range msg.Parts {
				if part.Kind == "emote" {
					fetchEmoteBlocking(part.EmoteID)
				}
			}
			line := ChatLine{User: msg.User, Parts: msg.Parts}
			rt.printMessage(line)
		}
	}()

	rt.inputLoop(broadcasterID, userID, accessToken, quit)
	<-quit
}

func main() {
	if len(os.Args) >= 2 {
		switch os.Args[1] {
		case "--chat":
			openChat()
			return
		}
	}

	model := initialAuthModel()
	program := tea.NewProgram(model)
	selectedChannel, err := program.Run()
	if err != nil {
		fmt.Printf("Whoops an error has occurred: %v", err)
		os.Exit(1)
	}

	finalModel, ok := selectedChannel.(StreamsModel)
	if !ok {
		fmt.Println("Could not cast model")
		return
	}

	for {
		if finalModel.State == pageQuitting || finalModel.SelectedChannel == "" {
			break
		}

		broadcasterID := finalModel.BroadcasterIDs[finalModel.SelectedChannel]
		tokenFile := finalModel.TokenFile

		chatCmd, _ := spawnChatWindow(broadcasterID, tokenFile.UserID, tokenFile.AccessToken)

		mpvCmd, err := startMPVWithStream(finalModel.SelectedChannel)
		if err == nil && mpvCmd != nil {
			mpvCmd.Wait()
		}

		if chatCmd != nil && chatCmd.Process != nil {
			chatCmd.Process.Kill()
		}

		// re-show the streams list with the same data
		prog2 := tea.NewProgram(initialStreamsModel(
			finalModel.Channels,
			finalModel.BroadcasterIDs,
			tokenFile,
			0, 0,
		))
		res2, err := prog2.Run()
		if err != nil {
			break
		}
		next, ok := res2.(StreamsModel)
		if !ok {
			break
		}
		finalModel = next
	}
}
