package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
)

func saveToken(path, token, userID string) error {
	file := TokenFile{
		AccessToken: token,
		UserID:      userID,
	}
	bytesWrite, err := json.MarshalIndent(file, "", " ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, bytesWrite, 0666)
}

func tokenLoad(path string) (TokenFile, error) {
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

func checkTokenFile(tokenFilePath string) error {
	if _, err := os.Stat(tokenFilePath); errors.Is(err, os.ErrNotExist) {
		fmt.Println("tokens.json not found... Creating")
		if err := saveToken(tokenFilePath, "", ""); err != nil {
			fmt.Println("err creating token file:", err)
		}
	}
	return nil
}
