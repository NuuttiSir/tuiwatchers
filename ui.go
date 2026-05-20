// TODO: MAKE THE ITEMS BIGGER TO SEE
// TODO: MAKE PAGES
package main

import (
	"errors"
	"fmt"

	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type page int

const (
	pageAuthentication = iota
	pageStreams
	pageQuitting
)

var docStyle = lipgloss.NewStyle().Margin(1, 2)

type Model struct {
	State           page
	Spinner         spinner.Model
	Err             error
	ChannelList     list.Model
	SelectedChannel string
	BroadcasterIDs  map[string]string
	TokenFile       TokenFile
	WindowWidth     int
	WindowHeight    int
	AuthStatus      string
	DeviceCode      DeviceCodeResponse
}

type ChannelInfo struct {
	BroadcasterName string
	GameName        string
	ViewCount       int
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

func (chInfo ChannelInfo) FilterValue() string { return chInfo.BroadcasterName + " " + chInfo.GameName }
func (chInfo ChannelInfo) Title() string       { return chInfo.BroadcasterName }
func (chInfo ChannelInfo) Description() string {
	return fmt.Sprintf("%s - %d viewers", chInfo.GameName, chInfo.ViewCount)
}

func newChannelList(channels []ChannelInfo) list.Model {
	items := make([]list.Item, 0, len(channels))
	for _, channel := range channels {
		items = append(items, channel)
	}

	delegate := list.NewDefaultDelegate()
	delegate.SetSpacing(1)

	channelList := list.New(items, delegate, 0, 0)
	channelList.Title = "Live channels"
	channelList.SetFilteringEnabled(false)
	channelList.SetShowStatusBar(false)
	channelList.SetShowHelp(false)
	channelList.DisableQuitKeybindings()
	return channelList
}

func initialModel() Model {
	mySpinner := spinner.New()
	mySpinner.Spinner = spinner.Dot
	mySpinner.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	return Model{
		State:   pageAuthentication,
		Spinner: mySpinner,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(m.Spinner.Tick, authStartCommand())
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.Spinner, cmd = m.Spinner.Update(msg)
		return m, cmd
	case AuthSuccessMessage:
		m.State = pageStreams
		m.ChannelList = newChannelList(msg.ChannelList)
		m.BroadcasterIDs = msg.BroadcasterIDs
		m.TokenFile = msg.TokenFile
		h, v := docStyle.GetFrameSize()
		m.ChannelList.SetSize(m.WindowWidth-h, m.WindowHeight-v)
		return m, tea.ClearScreen
	case AuthErrorMessage:
		m.State = pageQuitting
		m.Err = msg.Err
		return m, tea.Quit
	case AuthDeviceCodeMessage:
		m.DeviceCode = msg.DeviceCode
		m.AuthStatus = fmt.Sprintf("Go to %s and input code %s to authenticate", msg.DeviceCode.VerificationURI, msg.DeviceCode.UserCode)
		return m, authPollCommand(msg.DeviceCode)
	case AuthUserTokenMessage:
		if msg.Err != nil && msg.UserToken.AccessToken == " " {
			m.State = pageQuitting
			m.Err = msg.Err
			return m, tea.Quit
		}

		authUser := getAuthenticatedUser(clientID, msg.UserToken)
		if authUser.ID == "" {
			m.State = pageQuitting
			m.Err = errors.New("Could not fetch user data")
			return m, tea.Quit
		}

		if err := saveToken(tokenFilePath, msg.UserToken.AccessToken, authUser.ID); err != nil {
			m.State = pageQuitting
			m.Err = err
			return m, tea.Quit
		}
		tokenFile, err := tokenLoad(tokenFilePath)
		if err != nil {
			m.State = pageQuitting
			m.Err = err
			return m, tea.Quit
		}
		followDataList := getFollowedChannels(tokenFile.UserID, clientID, AccessToken{AccessToken: tokenFile.AccessToken})
		if len(followDataList.Data) == 0 {
			m.State = pageQuitting
			m.Err = errors.New("no followed channels found")
			return m, tea.Quit
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
			m.State = pageQuitting
			m.Err = errors.New("no live channels found")
			return m, tea.Quit
		}

		m.State = pageStreams
		m.ChannelList = newChannelList(channels)
		m.BroadcasterIDs = ids
		m.TokenFile = tokenFile
		h, v := docStyle.GetFrameSize()
		m.ChannelList.SetSize(m.WindowWidth-h, m.WindowHeight-v)
		return m, tea.ClearScreen
	case tea.WindowSizeMsg:
		m.WindowWidth = msg.Width
		m.WindowHeight = msg.Height
		if m.State == pageStreams {
			h, v := docStyle.GetFrameSize()
			m.ChannelList.SetSize(m.WindowWidth-h, m.WindowHeight-v)
		}
	case tea.KeyPressMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			m.State = pageQuitting
			return m, tea.Quit
		case "enter":
			if m.State != pageStreams {
				return m, nil
			}
			if item, ok := m.ChannelList.SelectedItem().(ChannelInfo); ok {
				m.SelectedChannel = item.BroadcasterName
			}
			m.State = pageQuitting
			return m, tea.Quit
		}
	}

	if m.State == pageStreams {
		var cmd tea.Cmd
		m.ChannelList, cmd = m.ChannelList.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m Model) View() tea.View {
	var str string

	switch m.State {
	case pageAuthentication:
		status := "Authenticating..."
		if m.AuthStatus != ""{
			status = m.AuthStatus
		}
		str := fmt.Sprintf("%s %s", m.Spinner.View(), status)
		v := tea.NewView(docStyle.Render(str))
		v.AltScreen = true
		return v
	case pageStreams:
		content := m.ChannelList.View() + "\nUse arrow keys and Enter to select"
		v := tea.NewView(docStyle.Render(content))
		v.AltScreen = true
		return v
	case pageQuitting:
		if m.Err != nil {
			str = fmt.Sprintf("Error: %v\n", m.Err)
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
