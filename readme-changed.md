# TuiWatchers

[![Go](https://img.shields.io/badge/Go-1.26+-00ADD8?logo=go)](https://go.dev)
[![License](https://img.shields.io/badge/license-MIT-green)](LICENSE)

**TuiWatchers** is a terminal UI for browsing and watching your followed **Twitch** live streams. Pick a streamer from the list, and it launches **mpv** to play the stream and opens a **chat TUI** in a separate terminal window — all without leaving the command line.

## Features

- **Live stream browser** — lists all channels you follow that are currently live, sorted by game and viewer count
- **One-key playback** — press Enter to start streaming in mpv
- **Built-in chat** — spawns a dedicated terminal window with live Twitch chat (WebSocket via EventSub)
- **Send messages** — reply to chat directly from the terminal
- **OAuth login** — device-code flow; open a URL, enter a code, done
- **Session persistence** — your token is saved to `tokens.json` so you don't need to re-authenticate every time

## Requirements

| Dependency | Why you need it |
|---|---|
| [mpv](https://mpv.io) | Plays the stream |
| [yt-dlp](https://github.com/yt-dlp/yt-dlp) | mpv uses this under the hood to resolve Twitch URLs |
| A browser | To authenticate via `twitch.tv/activate` (works with Firefox, Chrome, etc.) |
| A supported terminal emulator | Needed for the separate chat window ([see below](#supported-terminals)) |

## Supported Terminals

The chat window auto-detects and spawns in one of these (must be on your `$PATH`):

- Ghostty
- Alacritty
- GNOME Terminal
- Konsole
- Kitty
- macOS Terminal (`open -a Terminal`)
- Windows Terminal (`wt`)

## Install

### Option A — Go toolchain (recommended)

```sh
go install github.com/NuuttiSir/tuiwatchers@latest
```

### Option B — Build from source

```sh
git clone https://github.com/NuuttiSir/tuiwatchers
cd tuiwatchers
go build
./tuiwatchers
```

See [Dependencies.md](Dependencies.md) for Windows-specific setup instructions.

## Usage

Run the program:

```sh
./tuiwatchers
```

On first launch, the app will display a URL and a code. Open `https://www.twitch.tv/activate` in your browser, enter the code, and authorize the app. Your session is saved to `tokens.json` for next time.

### Keybindings

| Key | Action |
|---|---|
| `↑` / `↓` | Navigate the stream list |
| `Enter` | Watch the selected stream (opens mpv + chat) |
| `q` / `Esc` / `Ctrl+C` | Quit |

### Chat window

When you select a stream, a separate terminal opens with a live chat TUI. You can:

- Scroll through messages with `↑`/`↓`
- Type a message and press `Enter` to send
- Press `Esc` to close the chat window

The app automatically kills the previous mpv and chat processes when you switch streams.

## How it works

1. **Auth** — Uses Twitch's OAuth Device Code flow (no local server needed).
2. **Fetch** — Queries the Twitch Helix API for your followed channels that are live.
3. **Watch** — Launches `mpv https://twitch.tv/<channel>` and a chat TUI subprocess.
4. **Chat** — Connects to Twitch EventSub over WebSocket for real-time messages.

## Motivation

I've always been drawn to terminal-first tools — Neovim, Yazi, you name it. I wanted to learn how APIs work, and building a Twitch TUI seemed like the perfect excuse. I couldn't find anything similar, so I made it myself.

This project made me fall in love with programming all over again after burning out in school. I hope you enjoy using it as much as I enjoyed building it.

## Contributing

Contributions are welcome! Fork the repo, make your changes, and open a pull request to the `main` branch.

```sh
git clone <your-fork-url>
cd tuiwatchers
go build
```

## License

[MIT](LICENSE)
