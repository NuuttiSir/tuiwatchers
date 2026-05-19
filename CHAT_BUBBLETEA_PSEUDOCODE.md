# Bubble Tea Chat Pseudocode

Detailed pseudocode for chat UI and message flow with Bubble Tea. The text input stays at the bottom and chat messages scroll in the same window.

## Overview (how the parts fit together)

This design puts Bubble Tea in charge of both input and rendering. Network I/O runs in goroutines and pushes events into Bubble Tea, which keeps the UI state consistent and avoids mixed terminal writes.

Key ideas:

- Bubble Tea owns the terminal. You never print directly to stdout once the program is running.
- Incoming websocket events are turned into `tea.Msg` values and sent into the program with `program.Send(...)`.
- Outgoing messages are triggered by the Enter key in `Update` and sent via a `tea.Cmd` so it happens asynchronously.
- Layout is handled by a `viewport` for chat history plus a `textinput` pinned at the bottom. A `WindowSizeMsg` recomputes sizes so it works on resize.

Event flow summary:

1. User types into `textinput`.
2. On Enter, the model appends the message locally and returns a `sendChatCmd`.
3. The `sendChatCmd` makes the API call, then returns a `sendResultMsg` so the UI can show success or error.
4. Websocket goroutine listens for chat notifications and sends `incomingChatMsg` into the program.
5. The model appends incoming messages, updates the viewport content, and scrolls to the bottom.

Why this fixes your current behavior:

- In the current code, the Bubble Tea program ends before `readInput` and `sendLoop` start, so nothing sends.
- With this approach, all input and output stays inside Bubble Tea so there are no competing terminal readers.


## chatUi.go (UI model + layout + input handling)

This file defines the user interface state and layout. It keeps:

- `input`: the text input field where the user types.
- `viewport`: the scrollable area for chat history.
- `messages`: the list of lines rendered into the viewport.
- `status`: optional text like send errors.
- sizing: `width` and `height` so the UI can resize correctly.

The key behavior is in `Update`:

- `WindowSizeMsg`: recompute the viewport height so it fills the space between the header and the input.
- `KeyPressMsg` with Enter: append your own message, clear input, scroll to bottom, then trigger the send command.
- `incomingChatMsg`: append messages coming from websocket.
- `sendResultMsg`: update status if sending failed.

The `View` method builds the layout top-to-bottom:

1. Header (title)
2. Viewport (chat history)
3. Input (always at bottom)
4. Footer (status or help)

```go
// PSEUDOCODE

import (
  "charm.land/bubbles/v2/textinput"
  "charm.land/bubbles/v2/viewport"
  tea "charm.land/bubbletea/v2"
  "charm.land/lipgloss/v2"
)

type incomingChatMsg struct {
  user string
  text string
}
type sendResultMsg struct {
  ok bool
  err error
}
type clearStatusMsg struct{}

type ChatModel struct {
  input textinput.Model
  viewport viewport.Model
  messages []string        // rendered lines: "user: message"
  status string            // errors/sent confirmation
  width int
  height int

  // IDs + token for send cmd
  broadcasterID string
  userID string
  accessToken string
}

func InitialChatModel(broadcasterID, userID, accessToken string) ChatModel {
  ti := textinput.New()
  ti.Placeholder = "Type message..."
  ti.Focus()
  ti.CharLimit = 500
  ti.SetWidth(1) // will be set on WindowSizeMsg

  vp := viewport.New(0, 0)

  return ChatModel{
    input: ti,
    viewport: vp,
    messages: []string{},
    broadcasterID: broadcasterID,
    userID: userID,
    accessToken: accessToken,
  }
}

func (m ChatModel) Init() tea.Cmd {
  return tea.Batch(textinput.Blink)
}

func (m ChatModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
  switch msg := msg.(type) {
  case tea.WindowSizeMsg:
    m.width = msg.Width
    m.height = msg.Height

    headerH := lipgloss.Height(m.headerView())
    footerH := lipgloss.Height(m.footerView())
    inputH := lipgloss.Height(m.input.View())

    m.input.SetWidth(msg.Width - 2)
    m.viewport.Width = msg.Width
    m.viewport.Height = msg.Height - headerH - inputH - footerH

    return m, nil

  case tea.KeyPressMsg:
    switch msg.String() {
    case "ctrl+c", "esc":
      return m, tea.Quit
    case "enter":
      text := strings.TrimSpace(m.input.Value())
      if text == "" {
        return m, nil
      }

      // show your own message immediately
      m.messages = append(m.messages, "you: "+text)
      m.input.SetValue("")
      m.viewport.SetContent(strings.Join(m.messages, "\n"))
      m.viewport.GotoBottom()

      // fire async send cmd, report result back to model
      return m, tea.Batch(
        sendChatCmd(m.broadcasterID, m.userID, m.accessToken, text),
      )
    }
  case incomingChatMsg:
    m.messages = append(m.messages, msg.user+": "+msg.text)
    m.viewport.SetContent(strings.Join(m.messages, "\n"))
    m.viewport.GotoBottom()
    return m, nil

  case sendResultMsg:
    if msg.err != nil || !msg.ok {
      m.status = "send failed"
    } else {
      m.status = ""
    }
    return m, nil
  }

  var cmd tea.Cmd
  m.input, cmd = m.input.Update(msg)
  return m, cmd
}

func (m ChatModel) View() tea.View {
  // layout: header -> viewport -> input -> footer
  content := lipgloss.JoinVertical(
    lipgloss.Top,
    m.headerView(),
    m.viewport.View(),
    m.input.View(),
    m.footerView(),
  )

  v := tea.NewView(content)
  return v
}

func (m ChatModel) headerView() string { return "Chat\n" }
func (m ChatModel) footerView() string {
  if m.status != "" {
    return "\n" + m.status
  }
  return "\nESC to quit"
}
```

## chat.go (networking + glue for Bubble Tea commands)

This file provides the network actions as Bubble Tea commands and a way to feed incoming messages into the UI:

- `sendChatCmd` is a `tea.Cmd` that performs the Twitch API call and returns a `sendResultMsg` so the UI can show success or failure without blocking.
- `connectAndListen` becomes a long-running goroutine that reads websocket events and pushes them to a channel. It does not touch the UI directly.

Important detail: the websocket goroutine should not print to stdout, because Bubble Tea controls the screen. It should only send messages into the `incoming` channel.

```go
// PSEUDOCODE

type incomingChatMsg struct { user, text string }
type sendResultMsg struct { ok bool; err error }

func sendChatCmd(broadcasterID, userID, accessToken, text string) tea.Cmd {
  return func() tea.Msg {
    resp := sendChatMessage(broadcasterID, userID, accessToken, text)
    if !resp.IsSent {
      return sendResultMsg{ok: false, err: errors.New(resp.DropReason.Message)}
    }
    return sendResultMsg{ok: true}
  }
}

// adjust connectAndListen to push messages into a channel or callback
func connectAndListen(ctx context.Context, out chan<- incomingChatMsg, broadcasterID, userID, accessToken string) {
  // ... websocket setup
  for {
    // read event
    // if ctx cancelled -> return

    switch messageType {
    case "notification":
      out <- incomingChatMsg{user: username, text: chatMessage}
    }
  }
}
```

## main.go (launch Bubble Tea + wire listener into UI)

This file is the glue that wires the UI to the websocket listener:

- Create the Bubble Tea program with `InitialChatModel`.
- Create the `incoming` channel for websocket events.
- Start `connectAndListen` in a goroutine.
- Start a small relay goroutine that reads from `incoming` and calls `program.Send(msg)`.
- Run the Bubble Tea program. When it exits, cancel the context so the websocket goroutine stops.

This keeps all UI updates serialized through Bubble Tea, which prevents screen glitches and keeps input responsive.

```go
// PSEUDOCODE

func openChat() {
  broadcasterID := os.Args[3]
  userID := os.Args[4]
  accessToken := os.Args[5]

  model := InitialChatModel(broadcasterID, userID, accessToken)
  program := tea.NewProgram(model, tea.WithAltScreen())

  // channel for incoming websocket messages
  incoming := make(chan incomingChatMsg, 50)

  ctx, cancel := context.WithCancel(context.Background())
  defer cancel()

  // websocket listener -> send into Bubble Tea
  go connectAndListen(ctx, incoming, broadcasterID, userID, accessToken)

  // channel -> program.Send loop
  go func() {
    for msg := range incoming {
      program.Send(msg)
    }
  }()

  // run Bubble Tea; when user quits, cancel websocket and exit
  if _, err := program.Run(); err != nil {
    fmt.Println(err)
  }
  cancel()
}
```
