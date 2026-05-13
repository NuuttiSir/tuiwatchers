package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	tea "charm.land/bubbletea/v2"
)

func startMPVWithStream(channel tea.Model) error {
	selectedChannel := channel.(model).selectedChannel
	selectedChannelGame := channel.(model).gameName

	mpvInstance := exec.Command("/usr/bin/mpv", "https://twitch.tv/"+selectedChannel)
	fmt.Printf("Starting mpv instance watching channel %s who is streaming %s", selectedChannel, selectedChannelGame)

	output, err := mpvInstance.CombinedOutput()
	if err != nil {
		if strings.Contains(string(output), "ytdl") {
			fmt.Println("\nSeems like you dont have yt-dlp downloaded")
			fmt.Println("Do you want me to download it and try again")
			// TODO: download from github or provide it with the app
			// downloadYTDLP()
			os.Exit(1)
		}
	}
	return nil
}
