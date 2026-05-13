// THIS FILE IS FOR REQUESTS TO THE api.twitch.tv URL
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const TwitchAPIURL = "https://api.twitch.tv/helix/"

func getUserData(username, clientID string, appToken AccessToken) UserData {
	req, err := http.NewRequest("GET", TwitchAPIURL+"users", nil)
	if err != nil {
		fmt.Println("error:", err)
		return UserData{}
	}

	if username != "" {
		q := req.URL.Query()
		q.Add("login", username)
		req.URL.RawQuery = q.Encode()
	}

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

func getFollowedChannels(userID, clientID string, userToken AccessToken) FollowDataList {
	req, err := http.NewRequest("GET", TwitchAPIURL+"streams/followed", nil)
	if err != nil {
		fmt.Println("error:", err)
		return FollowDataList{}
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
		return FollowDataList{}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		fmt.Println("follow error:", resp.Status, string(body))
		return FollowDataList{}
	}

	var followDataList FollowDataList
	if err := json.NewDecoder(resp.Body).Decode(&followDataList); err != nil {
		fmt.Println("error decoding:", err)
		return FollowDataList{}
	}
	return followDataList
}

func getAuthenticatedUser(clientID string, userToken AccessToken) UserData {
	req, err := http.NewRequest("GET", TwitchAPIURL+"users", nil)
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
