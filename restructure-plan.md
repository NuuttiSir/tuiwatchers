# Restructure Plan — Package Layout

## Goal

Keep `main.go` in root, move everything else into organized `internal/` packages.

---

## Recommended structure

```
tuiwatchers/
├── main.go                        # Entry point + program loop (package main)
├── go.mod
├── go.sum
├── tokens.json
├── README.md / TODO.txt / ...
└── internal/
    ├── types/
    │   └── types.go               # TokenFile, ChannelInfo, page enum, docStyle
    ├── twitch/
    │   └── api.go                 # getFollowedChannels, getAuthenticatedUser
    ├── auth/
    │   ├── auth.go                # OAuth flow (deviceToken, getUserToken, validateToken)
    │   ├── model.go               # AuthModel + messages + Init/Update/View
    │   └── commands.go            # authStartCommand, authPollCommand, authSuccessDelayCommand
    ├── streams/
    │   └── model.go               # StreamsModel + ChannelInfo + Init/Update/View + newChannelList
    ├── chat/
    │   ├── chat.go                # All chat types and functions
    │   └── terminal.go            # RawTerminal, render, input (split from chat.go if too big)
    ├── emotes/
    │   └── emotes.go              # EmoteCache, EmoteImage, FetchEmoteImage, kittyInlineImage
    ├── player/
    │   └── player.go              # startMPVWithStream
    └── storage/
        └── storage.go             # saveToken, tokenLoad, checkTokenFile
```

---

## Why this layout

- **`internal/`** — Go convention for packages not importable by external consumers. Compiler-enforced.
- **`types` package** — shared types (`TokenFile`, `ChannelInfo`, `page` enum, `docStyle`) need a non-main home so all `internal/` packages can import them. The only package with no business logic — just data definitions and lipgloss.
- **`auth` package** — combines current `auth.go` + `authPageUi.go` because they're tightly coupled: command closures produce messages the model consumes. One package avoids circular imports and cross-package message type headaches.
- **`streams` package** — owns `StreamsModel`, `ChannelInfo`, and the channel list widget.
- **Everything else** — one package per concern, all importing from `types` as needed.
- **`main.go` stays thin** — just sets up the program, runs it, loops on channel selection. Imports everything.

---

## Key file moves

| Current file | New location | Notes |
|-------------|--------------|-------|
| `main.go` (types) | `internal/types/types.go` | Move all shared struct definitions |
| `ui.go` | `internal/types/types.go` | `page` enum + `docStyle` merge into types |
| `auth.go` | `internal/auth/auth.go` | OAuth functions |
| `authPageUi.go` | `internal/auth/model.go` | AuthModel + messages + Init/Update/View |
| `streamsPageUi.go` | `internal/streams/model.go` | StreamsModel + Init/Update/View + newChannelList |
| `api.go` | `internal/twitch/api.go` | API client calls |
| `chat.go` | `internal/chat/chat.go` | All chat code |
| `emotes.go` | `internal/emotes/emotes.go` | Emote cache + rendering |
| `player.go` | `internal/player/player.go` | MPV launcher |
| `storage.go` | `internal/storage/storage.go` | Token file I/O |

---

## Main.go after restructure (conceptual)

```go
package main

import (
	"fmt"
	"os"

	"github.com/NuuttiSir/tuiwatchers/internal/auth"
	"github.com/NuuttiSir/tuiwatchers/internal/chat"
	"github.com/NuuttiSir/tuiwatchers/internal/player"
	"github.com/NuuttiSir/tuiwatchers/internal/streams"
	tea "charm.land/bubbletea/v2"
)

func main() {
	if len(os.Args) >= 2 {
		switch os.Args[1] {
		case "--chat":
			openChat()
			return
		}
	}

	model := auth.New()
	program := tea.NewProgram(model)
	selectedChannel, err := program.Run()
	if err != nil {
		fmt.Printf("Error: %v", err)
		os.Exit(1)
	}

	finalModel, ok := selectedChannel.(*streams.Model)
	if !ok {
		fmt.Println("Could not cast model")
		return
	}

	for {
		if finalModel.State == streams.PageQuitting || finalModel.SelectedChannel == "" {
			break
		}

		chatCmd, _ := chat.SpawnWindow(
			finalModel.BroadcasterIDs[finalModel.SelectedChannel],
			finalModel.TokenFile.UserID,
			finalModel.TokenFile.AccessToken,
		)

		mpvCmd, err := player.StartMPV(finalModel.SelectedChannel)
		if err == nil && mpvCmd != nil {
			mpvCmd.Wait()
		}

		if chatCmd != nil && chatCmd.Process != nil {
			chatCmd.Process.Kill()
		}

		prog2 := tea.NewProgram(streams.New(
			finalModel.Channels,
			finalModel.BroadcasterIDs,
			finalModel.TokenFile,
			0, 0,
		))
		res2, _ := prog2.Run()
		next, ok := res2.(*streams.Model)
		if !ok {
			break
		}
		finalModel = next
	}
}
```

---

## Import map (circular dependency check)

```
main ──→ types
main ──→ auth
main ──→ streams
main ──→ chat
main ──→ player

auth ──→ types
auth ──→ twitch
auth ──→ storage

streams ──→ types

twitch ──→ types
storage ──→ types
chat ──→ types
chat ──→ emotes
emotes ──→ (standalone, only stdlib)
player ──→ (standalone, only stdlib)
```

No circular dependencies.

---

## Alternative: Minimal split (if you want fewer packages)

Only move the I/O code. Keep bubbletea models in root `package main`.

```
tuiwatchers/
├── main.go                        # Entry point + loop
├── types.go                       # Shared types (package main)
├── ui.go                          # page enum, docStyle (package main)
├── authPageUi.go                  # AuthModel (package main)
├── streamsPageUi.go               # StreamsModel (package main)
├── go.mod
├── ...
└── internal/
    ├── twitch/
    │   └── api.go                 # API calls
    ├── auth/
    │   └── auth.go                # OAuth functions
    ├── chat/
    │   └── chat.go                # Chat code
    ├── emotes/
    │   └── emotes.go              # Emote cache
    ├── player/
    │   └── player.go              # MPV
    └── storage/
        └── storage.go             # Token I/O
```

**Pros:** No `types` package needed — shared types stay in `package main`. Fewer file moves.

**Cons:** Root dir still has 4+ `.go` files. The two model files (`authPageUi.go`, `streamsPageUi.go`) stay in root, which is what you said you wanted to move out of root.
