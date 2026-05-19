package main

import (
	"strings"

	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type ChatModel struct {
	TextInput textinput.Model
	Viewport  viewport.Model
	Messages  []string
	Status    string
	Width     int
	Height    int

	BroadcasterID string
	UserID        string
	AccessToken   string
}

type IncomingChatMessage struct {
	User string
	Text string
}

type SendResultMessage struct {
	Ok  bool
	Err error
}

type ClearStatusMEssage struct{}

func InitialChatModel(broadcasterID, userID, accessToken string) ChatModel {
	ti := textinput.New()
	ti.Placeholder = "Enter Chat Message Here"
	ti.SetVirtualCursor(false)
	ti.Focus()
	// Twitch does not allow sending messages > 500 chars
	ti.CharLimit = 500
	ti.SetWidth(1)

	viewport := viewport.New()

	return ChatModel{
		TextInput:     ti,
		Viewport:      viewport,
		Messages:      []string{},
		BroadcasterID: broadcasterID,
		UserID:        userID,
		AccessToken:   accessToken,
	}
}

func (cm ChatModel) Init() tea.Cmd {
	return tea.Batch(textinput.Blink)
}

func (cm ChatModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		cm.Width = msg.Width
		cm.Height = msg.Height

		headerHeight := lipgloss.Height(cm.headerView())
		footerHeight := lipgloss.Height(cm.footerView())
		inputHeight := lipgloss.Height(cm.TextInput.View())

		cm.TextInput.SetWidth(msg.Width - 2)
		cm.Viewport.SetWidth(msg.Width)
		cm.Viewport.SetHeight(msg.Height - headerHeight - inputHeight - footerHeight)

		return cm, nil

	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			return cm, tea.Quit
		case "enter":
			text := strings.TrimSpace(cm.TextInput.Value())
			if text == "" {
				return cm, nil
			}

			return cm, tea.Batch(sendChatCommand(cm.BroadcasterID, cm.UserID, cm.AccessToken, text))
		}
	case IncomingChatMessage:
		cm.Messages = append(cm.Messages, msg.User+": "+msg.Text)
		cm.Viewport.SetContent(strings.Join(cm.Messages, "\n"))
		cm.Viewport.GotoBottom()
	case SendResultMessage:
		if msg.Err != nil || !msg.Ok {
			cm.Status = "send failed"
		} else {
			cm.Status = ""
		}
		return cm, nil
	}

	var cmd tea.Cmd
	cm.TextInput, cmd = cm.TextInput.Update(msg)
	return cm, cmd
}

func (cm ChatModel) View() tea.View {
	// layout header viewport input footer
	content := lipgloss.JoinVertical(
		lipgloss.Top,
		cm.headerView(),
		cm.Viewport.View(),
		cm.TextInput.View(),
		cm.footerView(),
	)
	view := tea.NewView(content)
	return view
}

func (cm ChatModel) headerView() string { return "Chat\n" }
func (cm ChatModel) footerView() string {
	if cm.Status != "" {
		return "\n" + cm.Status
	}
	return "\nESC TO QUIT"
}
