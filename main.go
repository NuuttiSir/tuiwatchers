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

//== CONSTS ==//
const TWITCH_OAUTH_URL = "https://id.twitch.tv/oauth2/"
const TWITCH_API_URL = "https://api.twitch.tv/helix/"

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
		"scopes":    {"user:read:follows"},
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

	q := req.URL.Query()
	q.Add("user_id", userID)
	req.URL.RawQuery = q.Encode()

	req.Header.Set("Authorization", "Bearer "+userToken.AccessToken)
	req.Header.Set("Client-Id", clientID)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("error:", err)
		return FollowData{}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		fmt.Println("follow error:", resp.Status, string(body))
		return FollowData{}
	}

	var followData FollowData
	if err := json.NewDecoder(resp.Body).Decode(&followData); err != nil {
		fmt.Println("error decoding:", err)
		return FollowData{}
	}
	return followData
}

func getAuthenticatedUser(clientID string, userToken AccessToken) UserData {
	req, err := http.NewRequest("GET", "https://api.twitch.tv/helix/users", nil)
	if err != nil {
		fmt.Println("error:", err)
		return UserData{}
	}

	req.Header.Set("Authorization", "Bearer "+userToken.AccessToken)
	req.Header.Set("Client-Id", clientID)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("error:", err)
		return UserData{}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		fmt.Println("users error:", resp.Status, string(body))
		return UserData{}
	}

	var userDataList UserDataList
	if err := json.NewDecoder(resp.Body).Decode(&userDataList); err != nil {
		fmt.Println("error decoding:", err)
		return UserData{}
	}
	if len(userDataList.Data) == 0 {
		fmt.Println("user not found")
		return UserData{}
	}
	return userDataList.Data[0]
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

func validateToken(path string) error {
	tokenFile, err := tokenLoad(path)
	if err != nil {
		return nil
	}

	fmt.Println(tokenFile.AccessToken)

	req, err := http.NewRequest("GET", "https://id.twitch.tv/oauth2/validate", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "OAuth"+tokenFile.AccessToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		fmt.Println("OK")
		// parse json file
		// token is valid
		// get scopes, userid, clientid
	} else {
		fmt.Println("EI OK")
		// toklen is invalid/expired
		// Prompt to re-auth
	}
	fmt.Println("TOKENS OK")
	return nil
}

// TODO: Get pagination to work
func main() {
	godotenv.Load()

	clientID := os.Getenv("CLIENT_ID")

	file, err := os.Create("tokens.json")
	if err != nil {
		fmt.Println("err: ", err)
	}

	validateToken("tokens.json")

	userToken := getUserToken(clientID)
	userTokenFile, err := json.Marshal(&userToken.AccessToken)
	if err != nil {
		fmt.Println("err: ", err)
	}
	file.Write(userTokenFile)

	authUser := getAuthenticatedUser(clientID, userToken)
	authTokenFile, err := json.Marshal(&authUser.ID)
	if err != nil {
		fmt.Println("err: ", err)
	}
	file.Write(authTokenFile)

	followData := getFollowedChannels(authUser.ID, clientID, userToken)

	_, err = json.MarshalIndent(followData, "", " ")
	if err != nil {
		return
	}
}
