# Emote Support Plan (Step-by-Step)

This plan describes how to add Twitch emote rendering in the chat window.
The goal is to turn EventSub message fragments into mixed text + emote output,
fetch emote images from the Twitch CDN, and render them inline when possible.

## 1) Parse emote fragments from EventSub

Goal: use fragments instead of raw `Message.Text` so we know where emotes are.

Steps:
- Update `Fragment` and related structs in `chat.go` to match the EventSub
  `channel.chat.message` payload. Each fragment can include `emote` data.
- Add an `Emote` struct with fields like `ID`, `SetID`, `OwnerID` (if present).
- In `connectAndListen`, build a slice of message parts from fragments:
  - For text fragments, add a text part.
  - For emote fragments, add an emote part containing the emote ID and text.
- Only fall back to `Message.Text` when fragments are missing or empty.

Notes:
- The payload uses `event.message.fragments`; do not infer emotes from text.
- Keep JSON tags consistent; missing fields should default cleanly.

## 2) Carry structured message parts into the UI

Goal: stop storing chat messages as plain strings.

Steps:
- Replace `Messages []string` in `ChatModel` (`chatUi.go`) with a structured
  slice, for example:

  ```go
  type MessagePart struct {
    Kind    string // "text" or "emote"
    Text    string
    EmoteID string
  }

  type ChatLine struct {
    User  string
    Parts []MessagePart
  }
  ```

- Update `IncomingChatMessage` to carry `Parts` instead of `Text`.
- When receiving a message in the UI, append a `ChatLine` and rebuild the
  viewport from structured content.

Notes:
- Keep a helper like `formatChatLine(ChatLine)` to centralize rendering.
- This makes later extensions (badges, mentions, etc.) easier.

## 3) Add emote cache + fetcher

Goal: resolve emote IDs to image bytes and avoid refetching.

Steps:
- Create `emotes.go` with an in-memory cache keyed by emote ID.
- Define a function `getEmoteImage(id string) (img []byte, ok bool)` and
  `fetchEmoteImage(id string) tea.Cmd` for async download.
- Use Twitch CDN URL:
  `https://static-cdn.jtvnw.net/emoticons/v2/{id}/default/dark/1.0`
- On message receive, enqueue fetch commands for missing emotes.

Notes:
- Add a small timeout to HTTP client.
- Store failed IDs to avoid tight retry loops.

## 4) Render emotes inline (graphics + fallback)

Goal: show the emote image where it appears in the message.

Steps:
- Implement a renderer that builds a single line string by concatenating
  text parts and emote parts.
- For emote parts:
  - If image bytes exist, render via Kitty graphics protocol (Ghostty
    supports it). This outputs an escape sequence inline.
  - If no image available or terminal unsupported, render a fallback like
    `:Kappa:` or `[Kappa]`.

Notes:
- Keep the rendered emote width small (1-2 cells) to avoid layout jumps.
- Always provide a fallback path so chat remains readable.

## 5) Trigger re-render on emote load

Goal: when images arrive, the chat view updates automatically.

Steps:
- Define a Tea message (e.g., `EmoteLoadedMsg{ID string}`) when a fetch
  succeeds or fails.
- In the chat model `Update`, on `EmoteLoadedMsg`, rebuild the viewport content
  from the structured lines.

Notes:
- Keep scroll pinned to bottom after re-render.
- Avoid appending duplicate lines; just regenerate the full viewport string.

## 6) Optional: terminal capability detection

Goal: choose graphics or fallback at runtime.

Steps:
- Check an env var or term info to decide if Kitty graphics are supported.
  For example, Ghostty usually sets `TERM=xterm-kitty`.
- Store a boolean in the chat model (e.g., `SupportsGraphics`).

Notes:
- If unsure, default to fallback to avoid broken output.

## 7) Testing checklist

- Start chat window and confirm messages display normally without emotes.
- Post a known emote (e.g., Kappa) and verify it renders as image or fallback.
- Confirm chat remains readable when emote download fails (network issues).
- Confirm scrolling behavior is unchanged.
