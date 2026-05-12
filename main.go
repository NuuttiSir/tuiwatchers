package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/joho/godotenv"
)

const TwitchOauthURL = "https://id.twitch.tv/oauth2/"

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
	Total int `json:"total"`
	Data  []struct {
		BroadcasterID    string    `json:"broadcaster_id"`
		BroadcasterLogin string    `json:"broadcaster_login"`
		BroadcasterName  string    `json:"broadcaster_name"`
		FollowedAt       time.Time `json:"followed_at"`
	} `json:"data"`
	Pagination struct {
		Cursor string `json:"cursor"`
	}
}

type TokenFile struct {
	AccessToken string `json:"access_token"`
	UserID      string `json:"user_id"`
}

func saveToken(path, token, userID string) error {
	file := TokenFile{
		AccessToken: token,
		UserID:      userID,
	}
	bytesWrite, err := json.MarshalIndent(file, "", " ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, bytesWrite, 0666)
}

func tokenLoad(path string) (TokenFile, error) {
	bytesRead, err := os.ReadFile(path)
	if err != nil {
		return TokenFile{}, err
	}
	var tokenFile TokenFile
	if err := json.Unmarshal(bytesRead, &tokenFile); err != nil {
		return TokenFile{}, err
	}
	return tokenFile, nil
}

// TODO: Get pagination to work
// TODO: Get just live channels because who cares about channels  that are not live lol xddddd
func main() {
	godotenv.Load()
	clientID := os.Getenv("CLIENT_ID")
	tokenFilePath := "tokens.json"

	// TokenFile does not exist
	if _, err := os.Stat(tokenFilePath); errors.Is(err, os.ErrNotExist) {
		// tokens.json does not exist
		_, err := os.Create(tokenFilePath)
		if err != nil {
			fmt.Println("err: ", err)
		}
	}

	tokenFile, err := tokenLoad(tokenFilePath)
	if err != nil {
		fmt.Println("err: ", err)
	}
	if validateToken(tokenFile.AccessToken, clientID) {
		// Use the tokens and user id
	}

	userToken := getUserToken(clientID)
	// userTokenFile, err := json.Marshal(&userToken.AccessToken)
	// if err != nil {
	// 	fmt.Println("err: ", err)
	// }

	authUser := getAuthenticatedUser(clientID, userToken)
	// authTokenFile, err := json.Marshal(&authUser.ID)
	// if err != nil {
	// 	fmt.Println("err: ", err)
	// }

	followData := getFollowedChannels(authUser.ID, clientID, userToken)

	_, err := json.MarshalIndent(followData, "", " ")
	if err != nil {
		return
	}
}
