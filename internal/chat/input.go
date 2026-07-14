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

func (rt *RawTerminal) InputLoop(broadcasterID, userID, accessToken string, quit chan struct{}) {
	var buf []rune
	raw := make([]byte, 4096) // enough for any UTF-8 rune + escape sequences
	var pending []byte

	redraw := func() {
		rt.mu.Lock()
		fmt.Printf("\x1b[%d;1H\x1b[2K> %s", rt.Height, string(buf))
		rt.mu.Unlock()
	}

	for {
		n, err := os.Stdin.Read(raw)
		if err != nil || n == 0 {
			continue
		}
		data := append(pending, raw[:n]...)
		pending = nil

		i := 0
		for i < len(data) {
			keyPress := data[i]
			switch keyPress {
			case 3: // Ctrl+C
				close(quit)
				return
			case 27:
				// a lone ESC is  n == 1, an arrow key is 3+ bytes ( ESC [ A )
				if n == len(data)-1 {
					close(quit)
					return
				}
				j := i + 1
				if data[j] == '[' {
					j++
					for j < len(data) && (data[j] < 0x40 || data[j] > 0x7e) {
						j++
					}
				}
				i = j + 1
				continue
			case 13: // Enter
				msg := strings.TrimSpace(string(buf))
				buf = buf[:0]
				redraw()
				if msg != "" {
					go func() {
						resp := twitch.PostChatMessage(broadcasterID, userID, accessToken, msg)
						if !resp.IsSent {
							dropReason := resp.DropReason.Message
							// if dropReason == "" {
							// 	dropReason = "Message not sent (Default reason)"
							// }
							rt.PrintSystem("Reason for dropping message: " + dropReason)
						}
					}()
				}
				i++
				continue
			case 127: // Backspace
				if len(buf) > 0 {
					buf = buf[:len(buf)-1]
				}
				i++
				redraw()
				continue
			}
			rune, size := utf8.DecodeRune(data[i:])
			if rune == utf8.RuneError && size <= 1 {
				if utf8.RuneStart(keyPress) && len(data)-i < utf8.UTFMax {
					pending = append(pending, data[i:]...)
					i = len(data)
					continue
				}
				i++
				continue
			}
			if rune >= 32 && len(buf) < 500 {
				buf = append(buf, rune)
			}
			i += size
		}
		redraw()
	}
}
