// For funcs that call to twitch oauth2
package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

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

type validateResponse struct {
	ClientID  string `json:"client_id"`
	Login     string `json:"login"`
	UserID    string `json:"user_id"`
	ExpiresIn int    `json:"expires_in"`
}

func validateToken(accessTokenParam string) (accessToken, userID string, err error) {
	req, err := http.NewRequest("GET", "https://id.twitch.tv/oauth2/validate", nil)
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Authorization", "OAuth "+accessTokenParam)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("token invalid or expired (status %d)", resp.StatusCode)
	}

	var validated validateResponse
	if err := json.NewDecoder(resp.Body).Decode(&validated); err != nil {
		return "", "", fmt.Errorf("error decoding validate response: %w", err)
	}

	return accessTokenParam, validated.UserID, nil
}
