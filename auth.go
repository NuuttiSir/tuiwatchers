// For funcs that call to twitch oauth2
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	tea "charm.land/bubbletea/v2"
)

const (
	tokenFilePath  = "tokens.json"
	clientID       = "5kft01sjf8paema7idj04jakt7hlym"
	TwitchOauthURL = "https://id.twitch.tv/oauth2/"
)

// deviceToken returns the Device Token returned by Twitch device API
func deviceToken(clientID string) DeviceCodeResponse {
	resp, err := http.PostForm(TwitchOauthURL+"device", url.Values{
		"client_id": {clientID},
		"scopes":    {"user:read:follows user:write:chat user:read:chat"},
	})
	if err != nil {
		fmt.Println("error:", err)
		return DeviceCodeResponse{}
	}
	defer resp.Body.Close()

	var deviceCodeResponse DeviceCodeResponse
	if err := json.NewDecoder(resp.Body).Decode(&deviceCodeResponse); err != nil {
		fmt.Println("error decoding:", err)
		return DeviceCodeResponse{}
	}
	return deviceCodeResponse
}

func getUserToken(deviceCode DeviceCodeResponse) AccessToken {
	for {
		time.Sleep(time.Duration(deviceCode.Interval) * time.Second)

		resp, err := http.PostForm(TwitchOauthURL+"token", url.Values{
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
	}
}

func validateToken(accessTokenParam string) bool {
	req, err := http.NewRequest("GET", TwitchOauthURL+"validate", nil)
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
		// fmt.Println("token invalid or expired", resp.StatusCode)
		return false
	}

	return true
}

func authStartCommand() tea.Cmd {
	return func() tea.Msg {
		if err := checkTokenFile(tokenFilePath); err != nil {
			return AuthErrorMessage{Err: err}
		}

		tokenFile, err := tokenLoad(tokenFilePath)
		if err != nil {
			return AuthErrorMessage{Err: err}
		}

		if validateToken(tokenFile.AccessToken) {
			followDataList := getFollowedChannels(tokenFile.UserID, clientID, AccessToken{AccessToken: tokenFile.AccessToken})
			if len(followDataList.Data) == 0 {
				return AuthErrorMessage{Err: errors.New("no followed channels found")}
			}

			channels := make([]ChannelInfo, 0, len(followDataList.Data))
			ids := make(map[string]string)
			for _, channel := range followDataList.Data {
				if channel.Type != "live" {
					continue
				}
				channels = append(channels, ChannelInfo{
					BroadcasterName: channel.UserName,
					GameName:        channel.GameName,
					ViewCount:       channel.ViewerCount,
				})
				ids[channel.UserName] = channel.UserID
			}

			if len(channels) == 0 {
				return AuthErrorMessage{Err: errors.New("no live channels found")}
			}

			return AuthSuccessMessage{
				ChannelList:    channels,
				BroadcasterIDs: ids,
				TokenFile:      tokenFile,
			}
		}

		deviceCode := deviceToken(clientID)
		if deviceCode.DeviceCode == " " {
			return AuthErrorMessage{Err: errors.New("Could not get device code")}
		}
		return AuthDeviceCodeMessage{DeviceCode: deviceCode}
	}
}

func authPollCommand(deviceCode DeviceCodeResponse) tea.Cmd {
	return func() tea.Msg {
		userToken := getUserToken(deviceCode)
		if userToken.AccessToken == "" {
			return AuthErrorMessage{Err: errors.New("authentication failed")}
		}
		return AuthUserTokenMessage{UserToken: userToken}

	}
}
