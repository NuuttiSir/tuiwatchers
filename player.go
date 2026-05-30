package main

import (
	"os"
	"os/exec"
)

func startMPVWithStream(channelName string) (*exec.Cmd, error) {
	mpvInstance := exec.Command("/usr/bin/mpv",
		"--profile=sw-fast",
		"--vo=kitty",
		"--vo-kitty-use-shm=yes",
		"--really-quiet",
		"https://twitch.tv/"+channelName,
	)

	mpvInstance.Stdin = os.Stdin
	mpvInstance.Stdout = os.Stdout
	mpvInstance.Stderr = os.Stderr

	err := mpvInstance.Start()
	if err != nil {
		return nil, err
	}
	return mpvInstance, err
}
