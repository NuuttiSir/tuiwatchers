// THIS FILE IS FOR REQUESTS TO THE api.twitch.tv URL
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const TwitchAPIURL = "https://api.twitch.tv/helix/"

func getFollowedChannels(userID, clientID string, userToken AccessToken) (FollowDataList, error) {
	req, err := http.NewRequest("GET", TwitchAPIURL+"streams/followed", nil)
	if err != nil {
		return FollowDataList{}, fmt.Errorf("Create request: %w", err)
	}

	q := req.URL.Query()
	q.Add("user_id", userID)
	req.URL.RawQuery = q.Encode()

	req.Header.Set("Authorization", "Bearer "+userToken.AccessToken)
	req.Header.Set("Client-Id", clientID)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return FollowDataList{}, fmt.Errorf("Do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return FollowDataList{}, fmt.Errorf("Api error: %d,: %s", resp.StatusCode, string(body))
	}

	var followDataList FollowDataList
	if err := json.NewDecoder(resp.Body).Decode(&followDataList); err != nil {
		return FollowDataList{}, fmt.Errorf("Decode error: %w", err)
	}
	return followDataList, nil
}

func getAuthenticatedUser(clientID string, userToken AccessToken) (UserData, error) {
	req, err := http.NewRequest("GET", TwitchAPIURL+"users", nil)
	if err != nil {
		return UserData{}, fmt.Errorf("Create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+userToken.AccessToken)
	req.Header.Set("Client-Id", clientID)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return UserData{}, fmt.Errorf("Do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return UserData{}, fmt.Errorf("Api error: %d: %s", resp.StatusCode, string(body))
	}

	var userDataList UserDataList
	if err := json.NewDecoder(resp.Body).Decode(&userDataList); err != nil {
		return UserData{}, fmt.Errorf("Decode error: %w", err)
	}
	if len(userDataList.Data) == 0 {
		return UserData{}, fmt.Errorf("User not found error")
	}
	return userDataList.Data[0], nil
}
