# Portability Roadmap — Windows Support & Streamlined Linux

Status of making TuiWatchers work beyond "Ghostty on Linux, run from the
repo directory". The first round of fixes landed in the `terminal` PR
(merge `3bd932d`, commit `930b9db`); this doc records what that covered
and what's still open, in detail.

**Current state in one line:** `GOOS=linux` builds and works;
`GOOS=windows` fails to compile with exactly one error
(`internal/chat/terminal.go:80: undefined: syscall.SIGWINCH`). Fixing
that makes it *compile* — the sections below are what it takes to make it
*work*.

---

## ✅ Done (landed in the `terminal` merge)

### 1. mpv resolved via PATH — `internal/player/player.go`

`/usr/bin/mpv` is gone. `StartMPVWithStream` now does
`exec.LookPath("mpv")` and returns a wrapped error if mpv isn't
installed. This covers Homebrew, NixOS, Flatpak-adjacent setups, and on
Windows `LookPath` will resolve `mpv.exe` automatically. `--really-quiet`
was also dropped so mpv's own errors are visible on stdout/stderr.

**Caveat that moved to the TODO list:** the command still hardcodes
`--vo=kitty` — see §B below. PATH lookup was only half of the mpv story.

### 2. Token file in the OS config directory — `internal/storage/storage.go`, `internal/tui/auth.go`

The `tokenFilePath = "tokens.json"` (CWD-relative) constant in `auth.go`
is gone. `storage.TokenFilePath()` now resolves:

- Linux: `~/.config/tuiwatchers/tokens.json` (respects `$XDG_CONFIG_HOME`)
- Windows: `%AppData%\tuiwatchers\tokens.json`
- macOS: `~/Library/Application Support/tuiwatchers/tokens.json`

The directory is created with `0o700` and the file is written with
`0o600` (was `0644`) since it holds an OAuth token. Both call sites in
`auth.go` (`AuthStartCommand`, `HandleAuthUserToken`) use it.

**Consequence:** running from a different directory no longer silently
logs you out. **Not done:** migrating an existing CWD `tokens.json` on
first run — current users (including this repo's working copy, which
still has one lying around) get one extra login prompt. Probably fine to
skip; delete the stray `tokens.json` in the repo dir instead.

### 3. Chat spawn: absolute binary path + token off the command line — `main.go`

Two of the three problems in `spawnChatWindow` are fixed:

- **`./tuiwatchers` → `os.Executable()`.** The child chat process is
  launched via the absolute path of the running binary (quoted with `%q`
  for spaces), so the app no longer has to be started from the directory
  containing the binary. `go install` works now.
- **Access token no longer in argv.** It's passed via the
  `TUIWATCHERS_TOKEN` environment variable on the child process instead
  of as a command-line argument visible in `ps`/Task Manager.
  `openChat()` reads the env var and errors clearly if it's missing.
  Accordingly the `--chat` arg check in `main()` went from `>= 6` to
  `>= 5` args.

**Not done:** it still launches literally
`ghostty -e bash -c "<exe> --chat …; exec bash"` — hardcoded terminal,
hardcoded shell. See §A.

### 4. Chat window ordering + resize handling — `main.go`, `internal/chat/terminal.go`

- The chat window is now spawned *before* `mpv` starts (it used to be
  spawned after `mpvCmd.Wait()` returned, i.e. after the stream ended,
  then immediately killed). Spawn failure prints a warning instead of
  being silently discarded.
- `RawTerminal.WatchResize()` listens for `SIGWINCH`, recomputes
  width/height/scroll region, and redraws the prompt. A mutex now guards
  all terminal writes (`PrintMessage`, `InputLoop`'s redraw, resize).
- Input handling was rewritten to parse UTF-8 properly (partial-rune
  buffering across reads), swallow arrow-key escape sequences, and
  handle Ctrl+C/Esc/Enter/Backspace explicitly.

**This introduced the current Windows compile blocker** — see §C.

### 5. Emote pipeline hardening — `internal/emotes/emotes.go`, `internal/chat/render.go`

- Emote fetching moved from `main.go`'s message loop into
  `PrintMessage` via `PrefetchEmoteImage`, which deduplicates concurrent
  fetches with an `inflight` map and remembers failures
  (`FailedEmotes`) so dead emotes aren't re-requested forever.
- `KittyInlineImage` gained multi-chunk transmission (`m=1`/`m=0`) for
  payloads over 4 KB instead of silently assuming single-chunk.
- `RenderLine` falls back to `[emoteName]` text while an image isn't
  cached yet — which is accidentally most of the §D fallback already.

**Bug to fix while in there:** `emotes.go:114` writes the final chunk
header as the literal string `"x1b_Gm=0;"` — missing the `\` in `\x1b`,
so any emote whose base64 exceeds 4096 bytes prints garbage and breaks
the escape stream. One-character fix.

**Data race to fix while in there:** `FetchEmoteImage` writes
`cache.FailedEmotes[id] = true` on its error paths (lines 55, 62)
without holding the write lock.

---

## 🔲 Remaining work, ordered

### A. Portable chat-window spawn (the main cross-platform blocker)

`spawnChatWindow` still assumes Ghostty and bash. Plan:

**Unix terminal detection**, in order:

1. `$TERMINAL` env var, if set (common Linux convention)
2. A candidate table probed with `exec.LookPath`: `ghostty`, `kitty`,
   `wezterm`, `alacritty`, `foot`, `gnome-terminal`, `konsole`, `xterm`
3. A `--terminal <cmd>` flag / config option as the escape hatch

Each terminal has a different "run this command" flag (`-e` for most,
`start --` for wezterm, `--` for gnome-terminal), so make it a table of
`{name, argsPrefix}` rather than string concatenation. Since
`os.Executable()` already landed, the child can be exec'd *directly*
(terminal → tuiwatchers, no `bash -c` shim) — the only thing `bash -c`
still buys is the `; exec bash` keep-window-open-after-crash trick.
Decide whether that's worth keeping; if the chat process handles its own
errors (it now prints and returns instead of dying instantly), letting
the window close is fine and the shell dependency disappears too.

**Windows spawn path** — per-OS files with build tags
(`spawn_windows.go` / `spawn_unix.go`) rather than a
`runtime.GOOS` branch:

```go
// Windows Terminal if available, else a plain new console window
if wt, err := exec.LookPath("wt.exe"); err == nil {
    cmd = exec.Command(wt, exe, "--chat", clientID, broadcasterID, userID)
} else {
    cmd = exec.Command("cmd", "/c", "start", "", exe, "--chat", ...)
}
cmd.Env = append(os.Environ(), "TUIWATCHERS_TOKEN="+accessToken)
```

Note `cmd /c start` detaches — `chatCmd.Process.Kill()` in `main()` will
kill the launcher, not the chat window. Either accept that the window
outlives the stream on Windows, or have the chat process poll its parent
/ watch a pipe and exit itself.

**Alternative worth keeping in mind:** render chat in a pane *inside*
the main TUI (Bubble Tea handles split layouts). That deletes the whole
problem class — no terminal detection, no per-OS spawn code, no separate
raw-mode implementation — at the cost of a layout refactor. Long-term
that's the most portable design; the spawn table is the short-term fix.

### B. mpv video output is still Kitty-only

`--vo=kitty --vo-kitty-use-shm=yes` means the *video itself* renders via
the Kitty graphics protocol, in the terminal. That works on
Kitty/Ghostty/WezTerm and nowhere else — on Windows Terminal, GNOME
Terminal, etc., mpv will fail or show nothing, even though the mpv
binary is found fine now.

- Detect Kitty-graphics support (same detection §D needs — build it
  once, share it) and only pass `--vo=kitty` when supported.
- Fallback: drop the `--vo` flags entirely and let mpv open its own
  native window (this is also the sanest default on Windows). Also drop
  `--vo-kitty-use-shm=yes` on Windows — POSIX shm doesn't exist there.
- Optional: a `--player-in-terminal=bool` flag/config so users can force
  either mode.
- Startup dependency check: `exec.LookPath` for `mpv` *and* `yt-dlp` at
  launch and print the per-OS install commands from `Dependencies.md`
  instead of failing mid-stream.

### C. Fix the Windows build: SIGWINCH

The one compile error. `syscall.SIGWINCH` doesn't exist on Windows.
Split `WatchResize` with build tags:

- `resize_unix.go` (`//go:build !windows`): current SIGWINCH
  implementation.
- `resize_windows.go` (`//go:build windows`): poll `term.GetSize` on a
  ticker (250–500 ms is plenty for a chat window), or use console
  window-buffer-size events via `golang.org/x/sys/windows` if polling
  feels dirty.

The rest of `terminal.go` (raw mode via `golang.org/x/term`, mutex,
scroll-region printing) is portable as-is *at the API level* — whether
the escape sequences actually render is §E.

### D. Emote rendering: capability detection

`RenderLine` already falls back to `[emoteName]` when an image isn't
cached — so the missing piece is smaller than originally scoped: a
`kittySupported` check that, when false, makes `RenderLine` *always*
take the text path and makes `PrefetchEmoteImage` a no-op (don't fetch
images that will never be shown).

- Detection: check `$TERM`/`$TERM_PROGRAM`/`$KITTY_WINDOW_ID` first;
  for rigor, send a Kitty graphics query escape and wait briefly for a
  response. Env sniffing alone is enough for v1.
- Share the result with §B's mpv decision.
- Optional later tier: Sixel (Windows Terminal ≥1.22 ships it), but text
  fallback alone is enough to call Windows "supported".
- While in the file: fix the `x1b` typo and the `FailedEmotes` race
  listed above.

### E. Verify raw-mode chat rendering on Windows

`golang.org/x/term` supports Windows consoles, but the manual escape
sequences (`InitScreen`'s scroll region `\x1b[r`, cursor save/restore
`\x1b[s`/`\x1b[u`, `\x1b[2K`) need virtual terminal processing enabled.
Windows Terminal enables VT by default — test there first. For plain
conhost, enable `ENABLE_VIRTUAL_TERMINAL_PROCESSING` on stdout via
`golang.org/x/sys/windows` in a build-tagged `terminal_windows.go`
(Bubble Tea does this automatically for the main UI; the raw chat window
must do it itself). Two Windows-specific input quirks to test in
`InputLoop`: legacy conhost sends Backspace as `8`, not `127` (handle
both — it's one extra case), and Enter arrives as `\r` (13), which is
already handled.

### F. Packaging & distribution (the "streamlined" part)

Unchanged from before — still entirely to do once A–E land:

- **GitHub Releases with prebuilt binaries** via
  [GoReleaser](https://goreleaser.com) in CI: one tag push produces
  `linux/amd64`, `linux/arm64`, `windows/amd64`, `darwin/arm64`
  archives. Pure-Go deps mean `CGO_ENABLED=0` cross-compiles for free
  (the Windows build already almost compiles today — see §C).
- **Linux:** AUR `tuiwatchers-bin` once releases exist; GoReleaser can
  also emit `.deb`/`.rpm` (nfpm) and a Homebrew tap. Nix flake as a
  nice-to-have.
- **Windows:** Scoop bucket first (right audience), winget manifest
  second. Note repo is on Codeberg — GoReleaser supports Gitea/Forgejo
  releases, or mirror tags to GitHub for the release automation.

### G. Docs to update alongside

- `README.md` Requirements: drop "Ghostty terminal, I have not gotten
  this to work on others" once §A lands; document support tiers
  (in-terminal video + image emotes on Kitty-protocol terminals; native
  mpv window + text emotes elsewhere).
- Windows Quick Start (Scoop/winget one-liners from `Dependencies.md`).
- Document `$TERMINAL` / `--terminal`, the new token file location
  (`~/.config/tuiwatchers/tokens.json` etc.), and that `TUIWATCHERS_TOKEN`
  is internal plumbing, not user configuration.

---

## Suggested order of attack

1. **§C** SIGWINCH build-tag split — one small file each, and suddenly
   `GOOS=windows go build ./...` passes; keep it passing from then on
   (worth a CI check).
2. **§A** portable spawn — Unix terminal table first (fixes "works on my
   distro"), then the Windows `wt.exe`/`cmd start` branch.
3. **§D + §B** shared Kitty-graphics detection, wired into both emote
   rendering and mpv's `--vo` choice, plus the two small emotes.go bug
   fixes.
4. **§E** hands-on Windows Terminal testing of the raw chat window
   (VT sequences, Backspace 8 vs 127).
5. **§F + §G** GoReleaser + Scoop/AUR + README — turns "clone and
   build" into "install and run".
