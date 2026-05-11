package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"

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
	Data []struct {
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
	} `json:"data"`
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

func getAccessToken(clientID, clientSecret string) AccessToken {

	// try to post into https://id.twitch.tv/oauth2/token
	resp2, err := http.PostForm("https://id.twitch.tv/oauth2/token", url.Values{
		// With clientID,devicveCode, and perms user:read:follows
		"client_id":     {clientID},
		"client_secret": {clientSecret},
		// "device_code":   {deviceCode.DeviceCode},
		// "scopes":        {"user:read:follows"},
		"grant_type": {"client_credentials"},
	})
	if err != nil {
		fmt.Println("error:", err)
		return AccessToken{}
	}
	defer resp2.Body.Close()

	var accessToken AccessToken
	err = json.NewDecoder(resp2.Body).Decode(&accessToken)
	if err != nil {
		fmt.Println("error decoding:", err)
		return AccessToken{}
	}
	return AccessToken{
		AccessToken: accessToken.AccessToken,
		ExpiresIn:   accessToken.ExpiresIn,
		TokenType:   accessToken.TokenType,
	}
}

func main() {
	godotenv.Load()
	clientID := os.Getenv("CLIENT_ID")
	clientSecret := os.Getenv("CLIENT_SECRET")

	resp, err := http.PostForm("https://id.twitch.tv/oauth2/device", url.Values{
		"client_id": {clientID},
		"scopes":    {"user:read:follows"},
	})
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	defer resp.Body.Close()

	var deviceCode DeviceCodeResponse
	err = json.NewDecoder(resp.Body).Decode(&deviceCode)
	if err != nil {
		fmt.Println("error decoding:", err)
		return
	}

	// try to post into https://id.twitch.tv/oauth2/token
	resp2, err := http.PostForm("https://id.twitch.tv/oauth2/token", url.Values{
		// With clientID,devicveCode, and perms user:read:follows
		"client_id":     {clientID},
		"client_secret": {clientSecret},
		// "device_code":   {deviceCode.DeviceCode},
		// "scopes":        {"user:read:follows"},
		"grant_type": {"client_credentials"},
	})
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	defer resp2.Body.Close()

	var accessToken AccessToken
	err = json.NewDecoder(resp2.Body).Decode(&accessToken)
	if err != nil {
		fmt.Println("error decoding:", err)
		return
	}
	//TODO: Change token type to have first letter upper
	// fmt.Println("TokenType: ", accessToken.TokenType)

	// check if gotten tokens
	// iffnot keep polling or return error
	// if yes Save to tokenResponse Struct

	//In the end store to tokens.json

	// GET USER
	req, err := http.NewRequest("GET", "https://api.twitch.tv/helix/users", nil)
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	// Add query params
	queryParams := req.URL.Query()
	queryParams.Add("login", "noobi3553")
	req.URL.RawQuery = queryParams.Encode()

	// Add headers
	req.Header.Set("Authorization", "Bearer "+accessToken.AccessToken)
	req.Header.Set("Client-Id", clientID)

	client := &http.Client{}
	resp3, err := client.Do(req)
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	defer resp3.Body.Close()

	var userData UserData
	err = json.NewDecoder(resp3.Body).Decode(&userData)
	if err != nil {
		fmt.Println("error decoding:", err)
		return
	}
	fmt.Println(userData.Data[0].ID)

	// GET followed streams
	req2, err := http.NewRequest("GET", "https://api.twitch.tv/helix/channels/followed", nil)
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	// Add query params
	queryParams2 := req2.URL.Query()
	queryParams2.Add("user_id", userData.Data[0].ID)
	req2.URL.RawQuery = queryParams2.Encode()

	// Add headers
	req2.Header.Set("Authorization", "Bearer "+accessToken.AccessToken)
	req2.Header.Set("Client-Id", clientID)

	client2 := &http.Client{}
	resp4, err := client2.Do(req2)
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	defer resp4.Body.Close()

	var followData FollowData
	err = json.NewDecoder(resp4.Body).Decode(&followData)
	if err != nil {
		fmt.Println("error decoding:", err)
		return
	}
	fmt.Println(followData)
}
