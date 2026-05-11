package main

import (
	"encoding/json"
	"fmt"
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

func getAppToken(clientID, clientSecret string) AccessToken {
	resp, err := http.PostForm("https://id.twitch.tv/oauth2/token", url.Values{
		"client_id":     {clientID},
		"client_secret": {clientSecret},
		"grant_type":    {"client_credentials"},
	})
	if err != nil {
		fmt.Println("error:", err)
		return AccessToken{}
	}
	defer resp.Body.Close()

	var accessToken AccessToken
	err = json.NewDecoder(resp.Body).Decode(&accessToken)
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

func getUserToken(clientID string) AccessToken {
	resp, err := http.PostForm("https://id.twitch.tv/oauth2/device", url.Values{
		"client_id": {clientID},
		"scope":     {"user:read:follows"},
	})
	if err != nil {
		fmt.Println("error:", err)
		return AccessToken{}
	}
	defer resp.Body.Close()

	var deviceCode DeviceCodeResponse
	err = json.NewDecoder(resp.Body).Decode(&deviceCode)
	if err != nil {
		fmt.Println("error decoding:", err)
		return AccessToken{}
	}

	fmt.Println("Go to: ", deviceCode.VerificationURI)
	fmt.Println("Input code: ", deviceCode.UserCode)

	var userToken AccessToken
	for {
		time.Sleep(time.Duration(deviceCode.Interval) * time.Second)

		resp, err := http.PostForm("https://id.twitch.tv/oauth2/token", url.Values{
			"client_id":   {clientID},
			"device_code": {deviceCode.DeviceCode},
			"grant_type":  {"urn:ietf:params:oauth:grant-type:device_code"},
		})
		if err != nil {
			fmt.Println("err: ", err)
			return AccessToken{}
		}

		err = json.NewDecoder(resp.Body).Decode(&userToken)
		if err != nil {
			fmt.Println("err: ", err)
			return AccessToken{}
		}
		resp.Body.Close()

		if userToken.AccessToken != "" {
			break
		}
		fmt.Println("Waiting for authorization... Trying again in 5 seconds")
	}

	return userToken
}

func getUserData(username, clientID string, appToken AccessToken) UserData {
	req, err := http.NewRequest("GET", "https://api.twitch.tv/helix/users", nil)
	if err != nil {
		fmt.Println("error:", err)
		return UserData{}
	}

	if username != "" {
		q := req.URL.Query()
		q.Add("login", username)
		req.URL.RawQuery = q.Encode()
	}

	// Add query params
	// queryParams := req.URL.Query()
	// queryParams.Add("login", username)
	// req.URL.RawQuery = queryParams.Encode()

	// Add headers
	req.Header.Set("Authorization", "Bearer "+appToken.AccessToken)
	req.Header.Set("Client-Id", clientID)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("error:", err)
		return UserData{}
	}
	defer resp.Body.Close()

	var userDataList UserDataList
	err = json.NewDecoder(resp.Body).Decode(&userDataList)
	if err != nil {
		fmt.Println("error decoding:", err)
		return UserData{}
	}
	if len(userDataList.Data) == 0 {
		fmt.Println("user not found")
		return UserData{}
	}
	return userDataList.Data[0]
}

func getFollowedChannels(userID, clientID string, userToken AccessToken) FollowData {
	req, err := http.NewRequest("GET", "https://api.twitch.tv/helix/channels/followed", nil)
	if err != nil {
		fmt.Println("error:", err)
		return FollowData{}
	}

	// Add query params
	queryParams := req.URL.Query()
	queryParams.Add("user_id", userID)
	req.URL.RawQuery = queryParams.Encode()

	// Add headers
	req.Header.Set("Authorization", "Bearer "+userToken.AccessToken)
	req.Header.Set("Client-Id", clientID)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("error:", err)
		return FollowData{}
	}
	defer resp.Body.Close()

	var followData FollowData
	err = json.NewDecoder(resp.Body).Decode(&followData)
	if err != nil {
		fmt.Println("error decoding:", err)
		return FollowData{}
	}
	return followData
}

func main() {
	godotenv.Load()
	clientID := os.Getenv("CLIENT_ID")
	clientSecret := os.Getenv("CLIENT_SECRET")

	// 1. Check env vars loaded
	fmt.Println("clientID:", clientID)
	fmt.Println("clientSecret:", clientSecret)

	appToken := getAppToken(clientID, clientSecret)
	// 2. Check app token
	fmt.Println("appToken:", appToken)

	userToken := getUserToken(clientID)
	// 3. Check user token
	fmt.Println("userToken:", userToken)

	userData := getUserData("noobi3553", clientID, appToken)
	// 4. Check user data
	fmt.Println("userData:", userData)

	followData := getFollowedChannels(userData.ID, clientID, userToken)
	fmt.Println("followData:", followData)

	//TODO: Change token type to have first letter upper
	// fmt.Println("TokenType: ", accessToken.TokenType)

	//In the end store to tokens.json

}
