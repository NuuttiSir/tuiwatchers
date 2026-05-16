// TODO: MAKE THE ITEMS BIGGER TO SEE
// TODO: MAKE PAGES
package main

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

var (
	mainStyle = lipgloss.NewStyle().MarginLeft(2)
)

type model struct {
	channelList     []channelInfo
	selectedChannel string
	AuthComplete    bool
	Quitting        bool
	Choice          int
}

type channelInfo struct {
	title     string
	gameName  string
	viewCount int
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Make sure these keys always quit
	if msg, ok := msg.(tea.KeyPressMsg); ok {
		k := msg.String()
		if k == "q" || k == "esc" || k == "ctrl+c" {
			m.Quitting = true
			return m, tea.Quit
		}
	}

	// Hand off the message and model to the appropriate update function for the
	// appropriate view based on the current state.
	if !m.AuthComplete {
		return AuthUpdate(msg, m)
	}
	return StreamsUpdate(msg, m)
}

func (m model) View() tea.View {
	var s string
	if m.Quitting {
		return tea.NewView("\n  See you later!\n\n")
	}
	if !m.AuthComplete {
		s = AuthView(m)
	} else {
		s = StreamsView(m, 0)
	}
	return tea.NewView(mainStyle.Render("\n" + s + "\n"))
}

// SUB-UPDATES

func AuthUpdate(msg tea.Msg, m model) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "enter":
			m.AuthComplete = true
		}
	}
	return m, nil
}

func StreamsUpdate(msg tea.Msg, m model) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "j", "down":
			m.Choice++
			if m.Choice > 3 {
				m.Choice = 3
			}
		case "k", "up":
			m.Choice--
			if m.Choice < 0 {
				m.Choice = 0
			}
		}
	}
	return m, nil
}

// SUB-VIEWS

func AuthView(m model) string {
	return "AUTH"
}

func StreamsView(m model, index int) string {

	str := fmt.Sprintf("%d. Streamer: %s", index + 1, m.channelList[index].title)
	return str
}
