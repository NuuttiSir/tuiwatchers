package main

import (
	"fmt"
	"os/exec"

	tea "charm.land/bubbletea/v2"
)

//NOTE: USE MPV
func startMPVWithStream(channel tea.Model) error {
	selectedChannel := channel.(model).selectedChannel
	selectedChannelGame := channel.(model).gameName

	mpvInstance := exec.Command("/usr/bin/mpv", "https://twitch.tv/"+selectedChannel)
	mpvInstance.Start()
	fmt.Printf("Starting mpv instance watching channel %v who is streaming %v", selectedChannel, selectedChannelGame)
	return nil
}
