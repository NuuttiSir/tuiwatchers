package main

import (
	"os"
	"os/exec"
)

// TODO: MAKE STREAM WINDOW START ON THE LEFT SIDE OF THE MONITOR
// GPT SAID THIS: mpv --gpu-context=x11egl --geometry=50%x100%+0+0 <video>
// SEEMED TO WORK

// TODO: For memes start a soap carving video or subway surfers when stream is
// on ad break
func startMPVWithStream(channelName string) (*exec.Cmd, error) {
	// mpvInstance := exec.Command("/usr/bin/mpv", "https://twitch.tv/"+channelName)
	// err := mpvInstance.Start()
	// if err != nil {
	// 	os.Exit(1)
	// }
	//
	// return mpvInstance, err
	mpvInstance := exec.Command("/usr/bin/mpv",
		// "--msg-level=ytdl=debug",
		// "--log-file=/tmp/mpv.log",
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
