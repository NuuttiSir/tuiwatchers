package tui

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type page int

const (
	PageDependencyCheck = iota
	PageAuthentication
	PageStreams
	PageQuitting
)

var docStyle = lipgloss.NewStyle().Margin(1, 2)

func renderView(str string) tea.View {
	v := tea.NewView(docStyle.Render(str))
	v.AltScreen = true
	return v
}
