# Restructure Roadmap

## Final Structure Target

```
tuiwatchers/
├── main.go
├── go.mod
└── internal/
    ├── twitch/
    │   ├── client.go      # shared http.Client, clientID constant
    │   ├── oauth.go       # deviceToken, getUserToken, validateToken
    │   ├── api.go         # GetFollowedChannels, GetAuthenticatedUser, SendChatMessage, PostSubscribe
    │   └── eventsub.go    # ConnectAndListen + all WebSocket types
    ├── tui/
    │   ├── styles.go      # page enum, docStyle, renderView
    │   ├── auth.go        # AuthModel + all messages + Init/Update/View + commands
    │   └── streams.go     # StreamsModel + ChannelInfo + newChannelList
    ├── chat/
    │   ├── terminal.go    # RawTerminal, initScreen, printMessage, newRawTerm
    │   ├── render.go      # renderLine
    │   └── input.go       # inputLoop
    ├── emotes/
    │   └── emotes.go      # EmoteCache (with mutex), FetchEmoteBlocking, kittyInlineImage
    ├── player/
    │   └── player.go      # StartMPVWithStream
    └── storage/
        └── storage.go     # TokenFile, saveToken, tokenLoad, checkTokenFile
```

---

### `twitch/api.go`

Add pagination to `GetFollowedChannels` here (loop on `Pagination.Cursor`).

### `twitch/eventsub.go`

**Reconnect note** — wrap the connection logic in an outer retry loop:

```go
func ConnectAndListen(ctx context.Context, out chan<- IncomingChatMessage, ...) {
    for {
        if ctx.Err() != nil {
            return
        }
        listenOnce(ctx, out, ...)  // existing logic extracted here
        select {
        case <-ctx.Done():
            return
        case <-time.After(3 * time.Second):
            // retry
        }
    }
}
```

`go build ./...` after all four files.

---

## Phase 6 — Clean Up `main.go`

At this point the root files `api.go`, `auth.go`, `authPageUi.go`,
`streamsPageUi.go`, `ui.go`, `player.go`, `storage.go`, `emotes.go`,
`chat.go` should all be deleted or empty.

What stays in `main.go`:

- All type definitions that haven't moved (should be none left)
- `openChat()` function (already here)
- `spawnChatWindow()` — **move here from chat.go**
- `main()` function

`spawnChatWindow` in `main.go` becomes:
```go
func spawnChatWindow(broadcasterID, userID, accessToken string) (*exec.Cmd, error) {
    cmd := exec.Command(
        "/usr/bin/ghostty", "-e", "bash", "-c",
        "./tuiwatchers --chat "+twitch.ClientID+" "+broadcasterID+" "+userID+" "+accessToken+";exec bash",
    )
    if err := cmd.Start(); err != nil {
        return nil, err
    }
    return cmd, nil
}
```

`openChat()` in `main.go` updates its package-qualified calls:
```go
func openChat() {
    broadcasterID := os.Args[3]
    userID        := os.Args[4]
    accessToken   := os.Args[5]

    rt, err := chat.NewRawTerm()
    // ...
    go twitch.ConnectAndListen(ctx, incoming, broadcasterID, userID, accessToken)
    // ...
}
```

Final `main()` imports:
```go
import (
    "github.com/NuuttiSir/tuiwatchers/internal/chat"
    "github.com/NuuttiSir/tuiwatchers/internal/player"
    "github.com/NuuttiSir/tuiwatchers/internal/twitch"
    "github.com/NuuttiSir/tuiwatchers/internal/tui"
    tea "charm.land/bubbletea/v2"
)
```

`go build ./...` — everything should compile.

---

## Phase 7 — Post-Restructure Improvements

Now that the code is organized, these are easier to tackle cleanly:

- [ ] **`twitch/oauth.go`** — add `context.Context` parameter to `getUserToken`
  so the poll loop can be cancelled when the user quits
- [ ] **`twitch/api.go`** — verify pagination loop works end-to-end with a
  real account that follows many channels
- [ ] **`chat/terminal.go`** — expand the raw input buffer from 4 bytes to 16
  to safely handle longer escape sequences (arrow keys, function keys)
- [ ] **`main.go`** — make the Ghostty path (`/usr/bin/ghostty`) configurable,
  e.g. via an env var `TUIWATCHERS_TERMINAL`, with Ghostty as fallback
- [ ] **General** — run `go vet ./...` and `go test ./...` after Phase 6 passes

---

## Quick Reference — What Moves Where

| What | From | To |
|------|------|----|
| `TokenFile` + storage funcs | `storage.go` | `internal/storage` |
| `StartMPVWithStream` | `player.go` | `internal/player` |
| `EmoteCache` + emote funcs | `emotes.go` | `internal/emotes` |
| `ClientID` constant | `auth.go` | `internal/twitch/client.go` |
| OAuth funcs | `auth.go` | `internal/twitch/oauth.go` |
| Helix API funcs + send/subscribe | `api.go` + `chat.go` | `internal/twitch/api.go` |
| WebSocket listener + types | `chat.go` | `internal/twitch/eventsub.go` |
| `RawTerminal` + screen funcs | `chat.go` | `internal/chat/terminal.go` |
| `renderLine` | `chat.go` | `internal/chat/render.go` |
| `inputLoop` | `chat.go` | `internal/chat/input.go` |
| `page` enum + `docStyle` + `renderView` | `ui.go` + `authPageUi.go` | `internal/tui/styles.go` |
| `AuthModel` + commands + messages | `authPageUi.go` + `auth.go` | `internal/tui/auth.go` |
| `StreamsModel` + `ChannelInfo` | `streamsPageUi.go` | `internal/tui/streams.go` |
| `spawnChatWindow` | `chat.go` | `main.go` |
| Twitch API types | `main.go` | `internal/twitch/api.go` |
