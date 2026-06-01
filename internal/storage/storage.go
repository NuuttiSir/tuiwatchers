package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
)

type TokenFile struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	UserID       string `json:"user_id"`
}

func SaveToken(path, accessToken, userID string) error {
	file := TokenFile{
		AccessToken: accessToken,
		UserID:      userID,
	}
	bytesWrite, err := json.MarshalIndent(file, "", "\t")
	if err != nil {
		return err
	}
	return os.WriteFile(path, bytesWrite, 0644)
}

func TokenLoad(path string) (TokenFile, error) {
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

func CheckTokenFile(tokenFilePath string) error {
	if _, err := os.Stat(tokenFilePath); errors.Is(err, os.ErrNotExist) {
		fmt.Println("tokens.json not found... Creating file")
		if err := SaveToken(tokenFilePath, "", ""); err != nil {
			return err
		}
	}
	return nil
}
