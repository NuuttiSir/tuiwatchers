// NOTE: Cant use refresh tokens because they need client secret and cant give that publicly
package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	tea "charm.land/bubbletea/v2"
	"github.com/NuuttiSir/tuiwatchers/internal/chat"
	"github.com/NuuttiSir/tuiwatchers/internal/player"
	"github.com/NuuttiSir/tuiwatchers/internal/tui"
	"github.com/NuuttiSir/tuiwatchers/internal/twitch"
)

func spawnChatWindow(broadcasterID, userID, accessToken string) (*exec.Cmd, error) {
	executable, err := os.Executable()
	if err != nil {
		return nil, err
	}
	startScript := fmt.Sprintf("%q --chat %s %s; exec bash", executable, broadcasterID, userID)
	cmd := exec.Command("ghostty", "-e", "bash", "-c", startScript)
	cmd.Env = append(os.Environ(), "TUIWATCHERS_TOKEN="+accessToken)

	err = cmd.Start()
	if err != nil {
		return nil, err
	}

	return cmd, nil
}

func openChat(broadcasterID, userID string) {
	accessToken := os.Getenv("TUIWATCHERS_TOKEN")
	if accessToken == "" {
		fmt.Println("missing TUIWATCHERS_TOKEN")
		return
	}

	rt, err := chat.NewRawTerm()
	if err != nil {
		fmt.Println(err)
		return
	}
	defer rt.Restore()

	rt.InitScreen()
	rt.WatchResize()

	quit := make(chan struct{})
	incoming := make(chan twitch.IncomingChatMessage, 50)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go twitch.ConnectAndListen(ctx, incoming, broadcasterID, userID, accessToken)
	go func() {
		for msg := range incoming {
			parts := msg.Parts
			if len(parts) == 0 && msg.Text != "" {
				parts = []twitch.MessagePart{{Kind: "text", Text: msg.Text}}
			}
			line := chat.ChatLine{User: msg.User, Parts: parts}
			rt.PrintMessage(line)
		}
	}()

	rt.InputLoop(broadcasterID, userID, accessToken, quit)
	<-quit
}

func main() {
	if len(os.Args) >= 4 {
		switch os.Args[1] {
		case "--chat":
			openChat(os.Args[2], os.Args[3]) // 2 = broadcasterID, 3 = userID
			return
		}
	}

	model := tui.InitialAuthModel()
	program := tea.NewProgram(model)
	selectedChannel, err := program.Run()
	if err != nil {
		fmt.Printf("Whoops an error has occurred: %v", err)
		os.Exit(1)
	}

	var finalModel tui.StreamsModel
	switch model := selectedChannel.(type) {
	case tui.StreamsModel:
		finalModel = model
	case tui.AuthModel:
		if model.Err != nil {
			fmt.Printf("Authentication failed: %v\n", model.Err)
			os.Exit(1)
		}
		return // User quit during auth
	default:
		fmt.Printf("Unexpected model type returned from auth")
		return

	}

	for {
		if finalModel.State == tui.PageQuitting || finalModel.SelectedChannel == "" {
			break
		}

		broadcasterID := finalModel.BroadcasterIDs[finalModel.SelectedChannel]
		tokenFile := finalModel.TokenFile

		chatCmd, err := spawnChatWindow(broadcasterID, tokenFile.UserID, tokenFile.AccessToken)
		if err != nil {
			fmt.Printf("Failed to open chat window: %v\n", err)
		}

		mpvCmd, err := player.StartMPVWithStream(finalModel.SelectedChannel)
		if err != nil {
			fmt.Printf("Could not start mpv player: %v\n", err)
		} else if mpvCmd != nil {
			mpvCmd.Wait()
		}

		if chatCmd != nil && chatCmd.Process != nil {
			chatCmd.Process.Kill()
		}

		// re-show the streams list with the same data
		// If streamer goes offline while watching other streams, IDK if their streams show on the list and what happens if clicked
		followed, err := twitch.GetFollowedChannels(tokenFile.UserID, twitch.ClientID, twitch.AccessToken{AccessToken: tokenFile.AccessToken})
		channels, ids := finalModel.Channels, finalModel.BroadcasterIDs // fallback as in stale/old data
		if err == nil {
			// TODO: make into package level function so tui.BuildCahnnelsList when time
			channels, ids = tui.AuthModel{}.BuildChannelList(followed)
		}
		prog2 := tea.NewProgram(tui.InitialStreamsModel(
			channels,
			ids,
			tokenFile,
			0, 0,
		))
		res2, err := prog2.Run()
		if err != nil {
			break
		}
		next, ok := res2.(tui.StreamsModel)
		if !ok {
			break
		}
		finalModel = next
	}
}
