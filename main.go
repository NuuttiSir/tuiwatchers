// NOTE: Cant use refresh tokens because they need client secret and cant give that publicly
package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	tea "charm.land/bubbletea/v2"
	"github.com/NuuttiSir/tuiwatchers/internal/chat"
	// "github.com/NuuttiSir/tuiwatchers/internal/emotes"
	"github.com/NuuttiSir/tuiwatchers/internal/player"
	"github.com/NuuttiSir/tuiwatchers/internal/tui"
	"github.com/NuuttiSir/tuiwatchers/internal/twitch"
)

func spawnChatWindow(broadcasterID, userID, accessToken string) (*exec.Cmd, error) {
	executable, err := os.Executable()
	if err != nil {
		return nil, err
	}
	startScript := fmt.Sprintf("%q --chat %s %s %s; exec bash", executable, twitch.ClientID, broadcasterID, userID)
	cmd := exec.Command("ghostty", "-e", "bash", "-c", startScript)
	cmd.Env = append(os.Environ(), "TUIWATCHERS_TOKEN="+accessToken)

	err = cmd.Start()
	if err != nil {
		return nil, err
	}

	return cmd, nil
}

func openChat() {
	broadcasterID := os.Args[3]
	userID := os.Args[4]
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
			line := chat.ChatLine{User: msg.User, Parts: msg.Parts}
			rt.PrintMessage(line)
		}
	}()

	rt.InputLoop(broadcasterID, userID, accessToken, quit)
	<-quit
}

func main() {
	if len(os.Args) >= 5 {
		switch os.Args[1] {
		case "--chat":
			openChat()
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

	finalModel, ok := selectedChannel.(tui.StreamsModel)
	if !ok {
		fmt.Println("Could not cast model")
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
		if err == nil && mpvCmd != nil {
			mpvCmd.Wait()
		}

		if chatCmd != nil && chatCmd.Process != nil {
			chatCmd.Process.Kill()
		}

		// re-show the streams list with the same data
		// If streamer goes offline while watching other streams, IDK if their streams show on the list and what happens if clicked
		prog2 := tea.NewProgram(tui.InitialStreamsModel(
			finalModel.Channels,
			finalModel.BroadcasterIDs,
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
