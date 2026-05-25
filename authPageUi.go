package main

import (
	"errors"
	"fmt"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type AuthModel struct {
	State        page
	Spinner      spinner.Model
	WindowWidth  int
	WindowHeight int
	AuthStatus   string
	TokenFile    TokenFile
	DeviceCode   DeviceCodeResponse
	PendingAuth  AuthSuccessMessage
	Err          error
}

type AuthSuccessMessage struct {
	ChannelList    []ChannelInfo
	BroadcasterIDs map[string]string
	TokenFile      TokenFile
}

type AuthErrorMessage struct {
	Err error
}

type AuthDeviceCodeMessage struct {
	DeviceCode DeviceCodeResponse
}

type AuthUserTokenMessage struct {
	UserToken AccessToken
	Err       error
}

type AuthDelayCompleteMessage struct{}

func initialAuthModel() AuthModel {
	Spinner := spinner.New()
	Spinner.Spinner = spinner.Dot
	Spinner.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	return AuthModel{
		State:   pageAuthentication,
		Spinner: Spinner,
	}
}

func (am AuthModel) Init() tea.Cmd {
	return tea.Batch(am.Spinner.Tick, authStartCommand())
}

func (am AuthModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case spinner.TickMsg:
		var cmd tea.Cmd
		am.Spinner, cmd = am.Spinner.Update(msg)
		return am, cmd
	case AuthSuccessMessage:
		am.PendingAuth = msg
		am.AuthStatus = "Successfull authentication"
		// TODO: Remove delay FUNC call
		return am, authSuccessDelayCommand()
	case AuthErrorMessage:
		am.State = pageQuitting
		am.Err = msg.Err
		return am, tea.Quit
	case AuthDeviceCodeMessage:
		am.DeviceCode = msg.DeviceCode
		am.AuthStatus = fmt.Sprintf("Go to %s and input code %s to authenticate\n", msg.DeviceCode.VerificationURI, msg.DeviceCode.UserCode)
		return am, authPollCommand(msg.DeviceCode)
	case AuthUserTokenMessage:
		if msg.Err != nil && msg.UserToken.AccessToken == " " {
			am.State = pageQuitting
			am.Err = msg.Err
			return am, tea.Quit
		}

		authUser := getAuthenticatedUser(clientID, msg.UserToken)
		if authUser.ID == "" {
			am.State = pageQuitting
			am.Err = errors.New("Could not fetch user data")
			return am, tea.Quit
		}

		if err := saveToken(tokenFilePath, msg.UserToken.AccessToken, authUser.ID); err != nil {
			am.State = pageQuitting
			am.Err = err
			return am, tea.Quit
		}
		tokenFile, err := tokenLoad(tokenFilePath)
		if err != nil {
			am.State = pageQuitting
			am.Err = err
			return am, tea.Quit
		}
		followDataList := getFollowedChannels(tokenFile.UserID, clientID, AccessToken{AccessToken: tokenFile.AccessToken})
		if len(followDataList.Data) == 0 {
			am.State = pageQuitting
			am.Err = errors.New("no followed channels found")
			return am, tea.Quit
		}
		channels := make([]ChannelInfo, 0, len(followDataList.Data))
		ids := make(map[string]string)
		for _, channel := range followDataList.Data {
			if channel.Type != "live" {
				continue
			}
			channels = append(channels, ChannelInfo{
				BroadcasterName: channel.UserName,
				GameName:        channel.GameName,
				ViewCount:       channel.ViewerCount,
			})
			ids[channel.UserName] = channel.UserID
		}

		if len(channels) == 0 {
			am.State = pageQuitting
			am.Err = errors.New("no live channels found")
			return am, tea.Quit
		}

		return am, func() tea.Msg {
			return AuthSuccessMessage{
				ChannelList:    channels,
				BroadcasterIDs: ids,
				TokenFile:      tokenFile,
			}
		}
	case AuthDelayCompleteMessage:
		return initialStreamsModel(
			am.PendingAuth.ChannelList,
			am.PendingAuth.BroadcasterIDs,
			am.PendingAuth.TokenFile,
			am.WindowWidth,
			am.WindowHeight,
		), tea.ClearScreen
	case tea.WindowSizeMsg:
		am.WindowWidth = msg.Width
		am.WindowHeight = msg.Height
	case tea.KeyPressMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			am.State = pageQuitting
			return am, tea.Quit
		}
	}
	return am, nil
}

func (am AuthModel) View() tea.View {
	var str string
	switch am.State {
	case pageAuthentication:
		status := "Authenticating..."
		if am.AuthStatus != "" {
			status = fmt.Sprintf("%s\n%s %s", status, am.Spinner.View(), am.AuthStatus)
		}
		str := fmt.Sprintf("%s %s", am.Spinner.View(), status)
		v := tea.NewView(docStyle.Render(str))
		v.AltScreen = true
		return v
	case pageQuitting:
		if am.Err != nil {
			str = fmt.Sprintf("Error: %v\n", am.Err)
			v := tea.NewView(docStyle.Render(str))
			v.AltScreen = true
			return v
		}
		str = "Goodbye.\n"
		v := tea.NewView(docStyle.Render(str))
		v.AltScreen = true
		return v
	default:
		str = "\n"
		v := tea.NewView(docStyle.Render(str))
		v.AltScreen = true
		return v
	}
}
