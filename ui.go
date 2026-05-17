// TODO: MAKE THE ITEMS BIGGER TO SEE
// TODO: MAKE PAGES
package main

import (
	"fmt"
	"strings"

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
		// if m.State != pageStreams {
		// 	return m, nil
		// }
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
		str := fmt.Sprintf("%s Authenticating...", m.Spinner.View())
		v := tea.NewView(str)
		v.AltScreen = true
		return v
	case pageStreams:
		str := m.renderStreamsPage()
		v := tea.NewView(str)
		v.AltScreen = true
		return v
	case pageQuitting:
		if m.Err != nil {
			str = fmt.Sprintf("Error: %v\n", m.Err)
			v := tea.NewView(str)
			v.AltScreen = true
			return v
		}
		str = "Goodbye.\n"
		v := tea.NewView(str)
		v.AltScreen = true
		return v
	default:
		str = "\n"
		v := tea.NewView(str)
		v.AltScreen = true
		return v
	}
}

func (m Model) renderStreamsPage() string {
	if len(m.ChannelList) == 0 {
		return "No channels are live that you follow SADGE :(\n"
	}

	var body strings.Builder
	body.WriteString("Live channels\n\n")
	for i, channel := range m.ChannelList {
		cursor := " "
		if i == m.SelectedIndex {
			cursor = ">"
		}
		fmt.Fprintf(&body, "%s %s - %s (%d viewers)\n", cursor, channel.BroadcasterName, channel.GameName, channel.ViewCount)
	}
	body.WriteString("\nUse arrow keys and Enter to select")
	return body.String()
}
