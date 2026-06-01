package chat

import (
	"fmt"
	"os"
	"strings"
	"unicode/utf8"

	"github.com/NuuttiSir/tuiwatchers/internal/twitch"
)

type SendResultMessage struct {
	Ok  bool
	Err error
}

type ClearStatusMessage struct{}

func (rt *RawTerminal) InputLoop(broadcasterID, userID, accessToken string,
	quit chan struct{}) {
	var buf []rune
	raw := make([]byte, 4) // enough for any UTF-8 rune + escape sequences

	redraw := func() {
		rt.mu.Lock()
		fmt.Printf("\x1b[%d;1H\x1b[2K> %s", rt.Height, string(buf))
		rt.mu.Unlock()
	}

	for {
		n, _ := os.Stdin.Read(raw)
		if n == 0 {
			continue
		}

		switch raw[0] {
		case 3, 27: // Ctrl+C or Esc
			close(quit)
			return
		case 13: // Enter
			msg := strings.TrimSpace(string(buf))
			buf = buf[:0]
			redraw()
			if msg != "" {
				go twitch.PostChatMessage(broadcasterID, userID, accessToken, msg)
			}
		case 127: // Backspace (DEL)
			if len(buf) > 0 {
				buf = buf[:len(buf)-1]
				redraw()
			}
		default:
			r, size := utf8.DecodeRune(raw[:n])
			if size > 0 && r != utf8.RuneError && r >= 32 {
				if len(buf) < 500 { // Twitch message limit
					buf = append(buf, r)
					redraw()
				}
			}
		}
	}
}
