package tui

import (
	"errors"
	"fmt"
	"time"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/NuuttiSir/tuiwatchers/internal/storage"
	"github.com/NuuttiSir/tuiwatchers/internal/twitch"
)

type AuthModel struct {
	State        page
	Spinner      spinner.Model
	WindowWidth  int
	WindowHeight int
	AuthStatus   string
	TokenFile    storage.TokenFile
	DeviceCode   twitch.DeviceCodeResponse
	PendingAuth  AuthSuccessMessage
	Err          error
}

type AuthSuccessMessage struct {
	ChannelList    []ChannelInfo
	BroadcasterIDs map[string]string
	TokenFile      storage.TokenFile
}

type AuthErrorMessage struct {
	Err error
}

type AuthDeviceCodeMessage struct {
	DeviceCode twitch.DeviceCodeResponse
}

type AuthUserTokenMessage struct {
	UserToken twitch.AccessToken
	Err       error
}

type AuthDelayCompleteMessage struct{}

func InitialAuthModel() AuthModel {
	Spinner := spinner.New()
	Spinner.Spinner = spinner.Dot
	Spinner.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	return AuthModel{
		State:   PageAuthentication,
		Spinner: Spinner,
	}
}

func (am AuthModel) Init() tea.Cmd {
	return tea.Batch(am.Spinner.Tick, AuthStartCommand())
}

func (am AuthModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case spinner.TickMsg:
		var cmd tea.Cmd
		am.Spinner, cmd = am.Spinner.Update(msg)
		return am, cmd
	case AuthSuccessMessage:
		am.PendingAuth = msg
		am.AuthStatus = "Successful authentication"
		// TODO: Remove delay FUNC call
		return am, AuthSuccessDelayCommand()
	case AuthErrorMessage:
		am.State = PageQuitting
		am.Err = msg.Err
		return am, tea.Quit
	case AuthDeviceCodeMessage:
		am.DeviceCode = msg.DeviceCode
		am.AuthStatus = fmt.Sprintf("Go to %s and input code %s to authenticate\n", msg.DeviceCode.VerificationURI, msg.DeviceCode.UserCode)
		return am, AuthPollCommand(msg.DeviceCode)
	case AuthUserTokenMessage:
		return am.HandleAuthUserToken(msg)
	case AuthDelayCompleteMessage:
		return InitialStreamsModel(
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
			am.State = PageQuitting
			return am, tea.Quit
		}
	}
	return am, nil
}

func (am AuthModel) View() tea.View {
	var str string
	switch am.State {
	case PageAuthentication:
		status := "Authenticating..."
		if am.AuthStatus != "" {
			status = fmt.Sprintf("%s\n%s", status, am.AuthStatus)
		}
		str = fmt.Sprintf("%s %s", am.Spinner.View(), status)
		return renderView(str)
	case PageQuitting:
		if am.Err != nil {
			str = fmt.Sprintf("Error: %v\n", am.Err)
			return renderView(str)
		}
		str = "Goodbye.\n"
		return renderView(str)
	default:
		str = "\n"
		return renderView(str)
	}
}

func (am AuthModel) HandleAuthUserToken(msg AuthUserTokenMessage) (AuthModel, tea.Cmd) {
	tokenFilePath, err := storage.TokenFilePath()
	if err != nil {
		am.State = PageQuitting
		am.Err = err
		return am, tea.Quit
	}

	if msg.Err != nil || msg.UserToken.AccessToken == "" {
		am.State = PageQuitting
		am.Err = msg.Err
		return am, tea.Quit
	}

	authUser, err := twitch.GetAuthenticatedUser(twitch.ClientID, msg.UserToken)
	if authUser.ID == "" || err != nil {
		am.State = PageQuitting
		am.Err = fmt.Errorf("could not fetch user data: %w", err)
		return am, tea.Quit
	}

	if err := storage.SaveToken(tokenFilePath, msg.UserToken.AccessToken, authUser.ID); err != nil {
		am.State = PageQuitting
		am.Err = err
		return am, tea.Quit
	}
	tokenFile, err := storage.TokenLoad(tokenFilePath)
	if err != nil {
		am.State = PageQuitting
		am.Err = err
		return am, tea.Quit
	}
	followDataList, err := twitch.GetFollowedChannels(tokenFile.UserID, twitch.ClientID, twitch.AccessToken{AccessToken: tokenFile.AccessToken})
	if err != nil {
		am.State = PageQuitting
		am.Err = fmt.Errorf("could not fetch followed channels: %w", err)
		return am, tea.Quit
	}
	// Same here as in getFollowedChannels
	if len(followDataList.Data) == 0 {
		am.State = PageQuitting
		am.Err = fmt.Errorf("none of the channels that you follow are live ATM: %w", err)
		return am, tea.Quit
	}

	channels, ids := am.BuildChannelList(followDataList)
	if len(channels) == 0 {
		am.State = PageQuitting
		am.Err = fmt.Errorf("No live channels found: %w", err)
		return am, tea.Quit
	}

	return am, func() tea.Msg {
		return AuthSuccessMessage{
			ChannelList:    channels,
			BroadcasterIDs: ids,
			TokenFile:      tokenFile,
		}
	}
}

func (am AuthModel) BuildChannelList(followDataList twitch.FollowDataList) ([]ChannelInfo, map[string]string) {
	channels := make([]ChannelInfo, 0, len(followDataList.Data))
	ids := make(map[string]string)
	for _, channel := range followDataList.Data {
		if channel.Type != "live" {
			continue
		}
		channels = append(channels, ChannelInfo{
			BroadcasterName: channel.UserName,
			Login:           channel.UserLogin,
			GameName:        channel.GameName,
			ViewCount:       channel.ViewerCount,
		})
		ids[channel.UserLogin] = channel.UserID
	}
	return channels, ids
}

func AuthStartCommand() tea.Cmd {
	return func() tea.Msg {
		tokenFilePath, err := storage.TokenFilePath()
		if err != nil {
			return AuthErrorMessage{Err: err}
		}

		if err := storage.CheckTokenFile(tokenFilePath); err != nil {
			return AuthErrorMessage{Err: err}
		}

		tokenFile, err := storage.TokenLoad(tokenFilePath)
		if err != nil {
			return AuthErrorMessage{Err: err}
		}

		validToken, err := twitch.ValidateToken(tokenFile.AccessToken)
		if err != nil {
			return AuthErrorMessage{Err: fmt.Errorf("Could not reach Twitch to validate token: %w", err)}
		}
		if validToken {
			followDataList, err := twitch.GetFollowedChannels(tokenFile.UserID, twitch.ClientID, twitch.AccessToken{AccessToken: tokenFile.AccessToken})
			if err != nil {
				return AuthErrorMessage{Err: err}
			}
			// Can't decide if i want to show this error or just give the user a empty list of channels
			if len(followDataList.Data) == 0 {
				return AuthErrorMessage{Err: errors.New("None of the channels that you follow are live ATM")}
			}

			channels, ids := InitialAuthModel().BuildChannelList(followDataList)
			if len(channels) == 0 {
				return AuthErrorMessage{Err: errors.New("no live channels found")}
			}

			return AuthSuccessMessage{
				ChannelList:    channels,
				BroadcasterIDs: ids,
				TokenFile:      tokenFile,
			}
		}

		deviceCode, err := twitch.DeviceToken()
		if err != nil {
			return AuthErrorMessage{Err: fmt.Errorf("could not get device code: %w", err)}
		}
		return AuthDeviceCodeMessage{DeviceCode: deviceCode}
	}
}

func AuthPollCommand(deviceCode twitch.DeviceCodeResponse) tea.Cmd {
	return func() tea.Msg {
		userToken, err := twitch.GetUserToken(deviceCode)
		if err != nil {
			return AuthErrorMessage{Err: err}
		}
		return AuthUserTokenMessage{UserToken: userToken}
	}
}

func AuthSuccessDelayCommand() tea.Cmd {
	return tea.Tick(2*time.Second, func(time.Time) tea.Msg {
		return AuthDelayCompleteMessage{}
	})
}
