package twitch

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

const TwitchOauthURL = "https://id.twitch.tv/oauth2/"

// deviceToken returns the Device Token returned by Twitch device API
func DeviceToken() DeviceCodeResponse {
	resp, err := http.PostForm(TwitchOauthURL+"device", url.Values{
		"client_id": {ClientID},
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

func GetUserToken(deviceCode DeviceCodeResponse) AccessToken {
	for {
		time.Sleep(time.Duration(deviceCode.Interval) * time.Second)

		resp, err := http.PostForm(TwitchOauthURL+"token", url.Values{
			"client_id":   {ClientID},
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
		resp.Body.Close()
	}
}

func ValidateToken(accessTokenParam string) bool {
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
		return false
	}

	return true
}
