package main

import (
	"os"
	"strings"

	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type ChatModel struct {
	TextInput textinput.Model
	Viewport  viewport.Model
	Messages  []ChatLine
	Status    string
	Width     int
	Height    int

	BroadcasterID string
	UserID        string
	AccessToken   string

	SupportsGraphics bool
}

type IncomingChatMessage struct {
	User  string
	Parts []MessagePart
	Text  string
}

type ChatLine struct {
	User  string
	Parts []MessagePart
}

type SendResultMessage struct {
	Ok  bool
	Err error
}

type ClearStatusMEssage struct{}

type EmoteLoadedMsg struct {
	ID      string
	Success bool
}

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
		TextInput:        ti,
		Viewport:         viewport,
		Messages:         []ChatLine{},
		BroadcasterID:    broadcasterID,
		UserID:           userID,
		AccessToken:      accessToken,
		SupportsGraphics: DetectGraphicsSupport(),
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
		var chatLine ChatLine
		chatLine.User = msg.User
		if len(msg.Parts) > 0 {
			chatLine.Parts = msg.Parts
		} else {
			chatLine.Parts = []MessagePart{{Kind: "text", Text: msg.Text}}
		}
		cm.Messages = append(cm.Messages, chatLine)
		cm.Viewport.SetContent(buildChatView(cm.Messages, cm.SupportsGraphics))
		cm.Viewport.GotoBottom()

		var cmds []tea.Cmd
		for _, part := range chatLine.Parts {
			if part.Kind == "emote" {
				if _, ok := EmoteImage(part.EmoteID); !ok {
					cmds = append(cmds, GetEmoteImage(part.EmoteID))
				}
			}
		}

		if len(cmds) > 0 {
			return cm, tea.Batch(cmds...)
		}
		return cm, nil

	case SendResultMessage:
		if msg.Err != nil || !msg.Ok {
			cm.Status = "send failed"
		} else {
			cm.Status = ""
		}
		return cm, nil
	case EmoteLoadedMsg:
		cm.Viewport.SetContent(buildChatView(cm.Messages, cm.SupportsGraphics))
		cm.Viewport.GotoBottom()
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

func formatChatLine(chatLine ChatLine, supportsGraphics bool) string {
	var builder strings.Builder
	builder.WriteString(chatLine.User)
	builder.WriteString(": ")
	for _, msgPart := range chatLine.Parts {
		if msgPart.Kind == "emote" {
			builder.WriteString("[")
			builder.WriteString(msgPart.Text)
			builder.WriteString("]")
			// 	if supportsGraphics {
			// 		if img, ok := EmoteImage(msgPart.EmoteID); ok && len(img) > 0 {
			// 			sequence := kittyInlineImage(img)
			// 			if sequence != "" {
			// 				builder.WriteString(sequence)
			// 				continue
			// 			}
			// 		}
			// 	}
			// 	// Fallback: emote name in brackets
			// 	builder.WriteString("[")
			// 	builder.WriteString(msgPart.Text)
			// 	builder.WriteString("]")
			// } else {
			// 	builder.WriteString(msgPart.Text)
			// }
		} else {
			builder.WriteString(msgPart.Text)
		}
	}
	return builder.String()
}

func buildChatView(chatLines []ChatLine, supportsGraphics bool) string {
	var builder strings.Builder
	for i, chatLine := range chatLines {
		if i > 0 {
			builder.WriteString("\n")
		}
		builder.WriteString(formatChatLine(chatLine, supportsGraphics))
	}
	return builder.String()
}

func DetectGraphicsSupport() bool {
	if os.Getenv("TERM") == "xterm-kitty" {
		return true
	}
	if os.Getenv("GHOSTTY_RESOURCES_DIR") != "" {
		return true
	}
	return false
}
