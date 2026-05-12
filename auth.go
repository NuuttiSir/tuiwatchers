// For funcs that call to twitch oauth2
package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

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

func validateToken(path, clientID string) (accessToken, userID string, err error) {
	tokenFile, err := tokenLoad(path)
	if err != nil {
		return "", "", err
	}

	fmt.Println(tokenFile.AccessToken)

	req, err := http.NewRequest("GET", "https://id.twitch.tv/oauth2/validate", nil)
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Authorization", "OAuth"+tokenFile.AccessToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		fmt.Println("OK")
		// token is valid
		fmt.Println("Token is still valid")
	} else {
		fmt.Println("EI OK")
		// toklen is invalid/expired
		fmt.Println("Token not valid")
		// Prompt to re-auth
		fmt.Println("lets reauth to get token")
		getUserToken(clientID)

	}
	fmt.Println("TOKENS OK")
	return tokenFile.AccessToken, tokenFile.UserID, nil
}
