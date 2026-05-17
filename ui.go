// TODO: MAKE THE ITEMS BIGGER TO SEE
// TODO: MAKE PAGES
package main

import (
	"errors"
	"fmt"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

const (
	tokenFilePath = "tokens.json"
	clientID      = "5kft01sjf8paema7idj04jakt7hlym"
)

type page int

const (
	pageAuthentication = iota
	pageStreams
	pageQuitting
)

type Model struct {
	State           page
	Spinner         spinner.Model
	Err             error
	ChannelList     []ChannelInfo
	SelectedIndex   int
	SelectedChannel string
	BroadcasterIDs  map[string]string
	TokenFile       TokenFile
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
	return tea.Batch(m.Spinner.Tick, authCommand())
}

func authCommand() tea.Cmd {
	return func() tea.Msg {
		if err := checkTokenFile(tokenFilePath); err != nil {
			return AuthErrorMessage{Err: err}
		}

		tokenFile, err := tokenLoad(tokenFilePath)
		if err != nil {
			return AuthErrorMessage{Err: err}
		}

		if !validateToken(tokenFile.AccessToken) {
			userToken := getUserToken(clientID)
			if userToken.AccessToken == "" {
				return AuthErrorMessage{Err: errors.New("authentication failed")}
			}

			authUser := getAuthenticatedUser(clientID, userToken)
			if authUser.ID == "" {
				return AuthErrorMessage{Err: errors.New("could not fetch user data")}
			}

			if err := saveToken(tokenFilePath, userToken.AccessToken, authUser.ID); err != nil {
				return AuthErrorMessage{Err: err}
			}
		}

		tokenFile, err = tokenLoad(tokenFilePath)
		if err != nil {
			return AuthErrorMessage{Err: err}
		}

		followDataList := getFollowedChannels(tokenFile.UserID, clientID, AccessToken{AccessToken: tokenFile.AccessToken})
		if len(followDataList.Data) == 0 {
			return AuthErrorMessage{Err: errors.New("no followed channels found")}
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
			return AuthErrorMessage{Err: errors.New("no live channels found")}
		}

		return AuthSuccessMessage{
			ChannelList:    channels,
			BroadcasterIDs: ids,
			TokenFile:      tokenFile,
		}
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.Spinner, cmd = m.Spinner.Update(msg)
		return m, cmd
	case AuthSuccessMessage:
		m.State = pageStreams
		m.ChannelList = msg.ChannelList
		m.BroadcasterIDs = msg.BroadcasterIDs
		m.TokenFile = msg.TokenFile
		m.SelectedIndex = 0
		return m, nil
	case AuthErrorMessage:
		m.State = pageQuitting
		m.Err = msg.Err
		return m, tea.Quit
	case tea.KeyPressMsg:
		if m.State != pageStreams {
			return m, nil
		}
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			m.State = pageQuitting
			return m, tea.Quit
		case "k", "up":
			if m.SelectedIndex > 0 {
				m.SelectedIndex--
			}
			return m, nil
		case "j", "down":
			if m.SelectedIndex < len(m.ChannelList)-1 {
				m.SelectedIndex++
			}
			return m, nil
		case "enter":
			m.SelectedChannel = m.ChannelList[m.SelectedIndex].BroadcasterName
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m Model) View() tea.View {
	var str string

	switch m.State {
	case pageAuthentication:
		str := fmt.Sprintf("%s Authenticating...\n", m.Spinner.View())
		return tea.NewView(str)
	case pageStreams:
		str := m.renderStreamsPage()
		return tea.NewView(str)
	case pageQuitting:
		if m.Err != nil {
			str = fmt.Sprintf("Error: %v\n", m.Err)
			return tea.NewView(str)
		}
		str = "Goodbye.\n"
		return tea.NewView(str)
	default:
		str = "\n"
		return tea.NewView(str)
	}
}

func (m Model) renderStreamsPage() string {
	if len(m.ChannelList) == 0 {
		return "No channels are live that you follow SADGE :(\n"
	}

	body := "Live channels\n\n"
	for i, channel := range m.ChannelList {
		cursor := " "
		if i == m.SelectedIndex {
			cursor = ">"
		}
		body += fmt.Sprintf("%s %s - %s (%d viewers)\n", cursor, channel.BroadcasterName, channel.GameName, channel.ViewCount)
	}
	body += "\nUse arrow keys and Enter to select"
	return body
}
