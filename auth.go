// For funcs that call to twitch oauth2
package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

const TwitchOauthURL = "https://id.twitch.tv/oauth2/"

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
	if err := json.NewDecoder(resp.Body).Decode(&deviceCode); err != nil {
		fmt.Println("error decoding:", err)
		return AccessToken{}
	}

	fmt.Println("Go to:", deviceCode.VerificationURI)
	fmt.Println("Enter code:", deviceCode.UserCode)

	for {
		time.Sleep(time.Duration(deviceCode.Interval) * time.Second)

		resp, err := http.PostForm("https://id.twitch.tv/oauth2/token", url.Values{
			"client_id":   {clientID},
			"device_code": {deviceCode.DeviceCode},
			"grant_type":  {"urn:ietf:params:oauth:grant-type:device_code"},
		})
		if err != nil {
			fmt.Println("err:", err)
			return AccessToken{}
		}
		defer resp.Body.Close()

		var userToken AccessToken
		if err := json.NewDecoder(resp.Body).Decode(&userToken); err != nil {
			fmt.Println("err:", err)
			return AccessToken{}
		}

		if userToken.AccessToken != "" {
			return userToken
		}
		fmt.Println("Waiting for authorization...")
	}
}

func validateToken(accessTokenParam string) bool {
	req, err := http.NewRequest("GET", "https://id.twitch.tv/oauth2/validate", nil)
	if err != nil {
		fmt.Println("err: ", err)
		return false
	}
	req.Header.Set("Authorization", "OAuth "+accessTokenParam)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("err: ", err)
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Println("token invalid or expired", resp.StatusCode)
		return false
	}

	return true
}
