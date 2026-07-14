package player

import (
	"fmt"
	"os"
	"os/exec"
)

func StartMPVWithStream(channelName string) (*exec.Cmd, error) {
	mpvPath, err := exec.LookPath("mpv")
	if err != nil {
		return nil, fmt.Errorf("mpv not found in PATH: %w", err)
	}
	mpvInstance := exec.Command(mpvPath,
		"--profile=sw-fast",
		"--vo=kitty",
		"--vo-kitty-use-shm=yes",
		"https://twitch.tv/"+channelName,
	)

	mpvInstance.Stdin = os.Stdin
	mpvInstance.Stdout = os.Stdout
	mpvInstance.Stderr = os.Stderr

	err = mpvInstance.Start()
	if err != nil {
		return nil, err
	}
	return mpvInstance, err
}
