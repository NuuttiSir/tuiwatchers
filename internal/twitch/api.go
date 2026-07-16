package twitch

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

const TwitchAPIURL = "https://api.twitch.tv/helix/"

var ErrSubscribeRejected = errors.New("Subscribe rejected")

type AccessToken struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
}

type DeviceCodeResponse struct {
	DeviceCode      string `json:"device_code"`
	Interval        int    `json:"interval"`
	UserCode        string `json:"user_code"`
	VerificationURI string `json:"verification_uri"`
	ExpiresIn       int    `json:"expires_in"`
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
	ID           string    `json:"id"`
	UserID       string    `json:"user_id"`
	UserLogin    string    `json:"user_login"`
	UserName     string    `json:"user_name"`
	GameID       string    `json:"game_id"`
	GameName     string    `json:"game_name"`
	Type         string    `json:"type"`
	Title        string    `json:"title"`
	ViewerCount  int       `json:"viewer_count"`
	StartedAt    time.Time `json:"started_at"`
	Language     string    `json:"language"`
	ThumbnailURL string    `json:"thumbnail_url"`
	TagIds       []any     `json:"tag_ids"`
	Tags         []string  `json:"tags"`
}

type FollowDataList struct {
	Data       []FollowData `json:"data"`
	Pagination struct {
		Cursor string `json:"cursor"`
	} `json:"pagination"`
}

type Condition struct {
	BroadcasterUserID string `json:"broadcaster_user_id"`
	UserID            string `json:"user_id"`
}

type Transport struct {
	Method    string `json:"method"`
	SessionID string `json:"session_id"`
}

type SubscriptionRequest struct {
	Type      string    `json:"type"`
	Version   string    `json:"version"`
	Condition Condition `json:"condition"`
	Transport Transport `json:"transport"`
}

type SendChatMessage struct {
	BroadcasterID string `json:"broadcaster_id"`
	SenderID      string `json:"sender_id"`
	Message       string `json:"message"`
}

type ReceivedChatMessageAnswer struct {
	MessageID  string     `json:"message_id"`
	IsSent     bool       `json:"is_sent"`
	DropReason DropReason `json:"drop_reason"`
}

type ChatMessageResponse struct {
	Data []ReceivedChatMessageAnswer `json:"data"`
}

type DropReason struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func GetFollowedChannels(userID, clientID string, userToken AccessToken) (FollowDataList, error) {
	var followedChannelList FollowDataList
	cursor := ""

	for {
		req, err := http.NewRequest("GET", TwitchAPIURL+"streams/followed", nil)
		if err != nil {
			return FollowDataList{}, fmt.Errorf("Create request: %w", err)
		}
		q := req.URL.Query()
		q.Add("user_id", userID)
		q.Add("first", "100")
		if cursor != "" {
			q.Add("after", cursor)
		}

		req.URL.RawQuery = q.Encode()

		req.Header.Set("Authorization", "Bearer "+userToken.AccessToken)
		req.Header.Set("Client-Id", clientID)

		page, err := func() (FollowDataList, error) {
			resp, err := HTTPClient.Do(req)
			if err != nil {
				return FollowDataList{}, fmt.Errorf("Do request: %w", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				body, _ := io.ReadAll(resp.Body)
				return FollowDataList{}, fmt.Errorf("Api error: %d,: %s", resp.StatusCode, string(body))
			}

			var page FollowDataList
			err = json.NewDecoder(resp.Body).Decode(&page)
			if err != nil {
				return FollowDataList{}, fmt.Errorf("Decode error: %w", err)
			}
			return page, nil
		}()
		if err != nil {
			return FollowDataList{}, err
		}

		followedChannelList.Data = append(followedChannelList.Data, page.Data...)
		cursor = page.Pagination.Cursor
		if cursor == "" {
			return followedChannelList, nil
		}
	}
}

func GetAuthenticatedUser(clientID string, userToken AccessToken) (UserData, error) {
	req, err := http.NewRequest("GET", TwitchAPIURL+"users", nil)
	if err != nil {
		return UserData{}, fmt.Errorf("Create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+userToken.AccessToken)
	req.Header.Set("Client-Id", clientID)

	resp, err := HTTPClient.Do(req)
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

func PostChatMessage(broadcasterID, userID, accessToken, message string) (ReceivedChatMessageAnswer, error) {
	messageData := SendChatMessage{
		BroadcasterID: broadcasterID,
		SenderID:      userID,
		Message:       message,
	}
	body, err := json.Marshal(messageData)
	if err != nil {
		return ReceivedChatMessageAnswer{}, err
	}

	req, err := http.NewRequest("POST", "https://api.twitch.tv/helix/chat/messages", bytes.NewBuffer(body))
	if err != nil {
		return ReceivedChatMessageAnswer{}, err
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Client-Id", ClientID)
	req.Header.Set("Content-Type", "application/json")

	resp, err := HTTPClient.Do(req)
	if err != nil {
		return ReceivedChatMessageAnswer{}, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return ReceivedChatMessageAnswer{}, err
	}

	if resp.StatusCode != http.StatusOK {
		return ReceivedChatMessageAnswer{}, fmt.Errorf("Send failed: %d: %s", resp.StatusCode, data)
	}

	var chatMessageAnswer ChatMessageResponse
	err = json.Unmarshal(data, &chatMessageAnswer)
	if err != nil {
		return ReceivedChatMessageAnswer{}, err
	}
	if len(chatMessageAnswer.Data) == 0 {
		return ReceivedChatMessageAnswer{}, errors.New("Send response had no data")
	}

	return chatMessageAnswer.Data[0], nil
}

func postSubscribe(clientID, userID, broadcasterID, sessionID, accessToken string) error {
	data := SubscriptionRequest{
		Type:    "channel.chat.message",
		Version: "1",
		Condition: Condition{
			BroadcasterUserID: broadcasterID,
			UserID:            userID,
		},
		Transport: Transport{
			Method:    "websocket",
			SessionID: sessionID,
		},
	}

	body, err := json.Marshal(data)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", "https://api.twitch.tv/helix/eventsub/subscriptions", bytes.NewBuffer(body))
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Client-Id", clientID)
	req.Header.Set("Content-Type", "application/json")

	resp, err := HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("Subscribe request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 && resp.StatusCode < 500 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("%w: %d: %s", ErrSubscribeRejected, resp.StatusCode, respBody)
	}
	if resp.StatusCode != http.StatusAccepted {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Subscribe failed: %d: %s", resp.StatusCode, respBody)
	}

	return nil
}
