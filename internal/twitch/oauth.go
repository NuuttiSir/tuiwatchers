package twitch

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

const TwitchOauthURL = "https://id.twitch.tv/oauth2/"

// deviceToken returns the Device Token returned by Twitch device API
func DeviceToken() (DeviceCodeResponse, error) {
	resp, err := HTTPClient.PostForm(TwitchOauthURL+"device", url.Values{
		"client_id": {ClientID},
		"scopes":    {"user:read:follows user:write:chat user:read:chat"},
	})
	if err != nil {
		return DeviceCodeResponse{}, err
	}
	defer resp.Body.Close()

	var deviceCodeResponse DeviceCodeResponse
	if err := json.NewDecoder(resp.Body).Decode(&deviceCodeResponse); err != nil {
		return DeviceCodeResponse{}, err
	}
	return deviceCodeResponse, nil
}

func GetUserToken( /**ctx context.Context,**/ deviceCode DeviceCodeResponse) (AccessToken, error) {
	deadline := time.Now().Add(time.Duration(deviceCode.ExpiresIn) * time.Second)
	interval := time.Duration(deviceCode.Interval) * time.Second

	for time.Now().Before(deadline) {
		time.Sleep(interval)

		token, twitchErr, err := pollToken(deviceCode)
		if err != nil {
			return AccessToken{}, err
		}
		if token.AccessToken != "" {
			return token, nil
		}

		switch twitchErr {
		case "authorization_pending":
			continue
		case "slow_down":
			interval += time.Second
		default:
			return AccessToken{}, fmt.Errorf("Device AUTH failed: %s", twitchErr)
		}
	}
	return AccessToken{}, errors.New("Device code expired before authorization")
}

func pollToken(deviceCode DeviceCodeResponse) (AccessToken, string, error) {
	resp, err := HTTPClient.PostForm(TwitchOauthURL+"token", url.Values{
		"client_id":   {ClientID},
		"device_code": {deviceCode.DeviceCode},
		"grant_type":  {"urn:ietf:params:oauth:grant-type:device_code"},
	})
	if err != nil {
		return AccessToken{}, "", err
	}
	defer resp.Body.Close()

	var accessTokenWithMessage struct {
		AccessToken
		Message string `json:"message"`
	}
	err = json.NewDecoder(resp.Body).Decode(&accessTokenWithMessage)
	if err != nil {
		return AccessToken{}, "", err
	}
	return accessTokenWithMessage.AccessToken, accessTokenWithMessage.Message, nil

}

func ValidateToken(accessTokenParam string) (bool, error) {
	req, err := http.NewRequest("GET", TwitchOauthURL+"validate", nil)
	if err != nil {
		return false, err
	}
	req.Header.Set("Authorization", "OAuth "+accessTokenParam)

	resp, err := HTTPClient.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return false, nil
	}

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("Validate returned: %d", resp.StatusCode)
	}

	return true, nil
}
