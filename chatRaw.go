package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"golang.org/x/term"
)

type RawTerm struct {
	FileDescriptor int
	Old            *term.State
	Width          int
	Height         int
	ChatRows       int
	mu             sync.Mutex
}

func newRawTerm() (*RawTerm, error) {
	fileDescriptor := int(os.Stdin.Fd())
	old, err := term.MakeRaw(fileDescriptor)
	if err != nil {
		return nil, err
	}
	width, height, err := term.GetSize(fileDescriptor)
	if err != nil {
		term.Restore(fileDescriptor, old)
		return nil, err
	}
	return &RawTerm{FileDescriptor: fileDescriptor, Old: old, Width: width, Height: height, ChatRows: height - 3}, nil

}

func (rt *RawTerm) restore() {
	fmt.Print("\x1b[r")
	fmt.Print("\x1b[?25h")
	fmt.Print("\x1b[2J/x1b[H")
	term.Restore(rt.FileDescriptor, rt.Old)
}

func (rt *RawTerm) initScreen() {
	fmt.Print("\x1b[2J")                  // Clear screen
	fmt.Printf("\x1b[1;%dr", rt.ChatRows) // scroll region = 1..ChatRows
	fmt.Printf("\x1b[%d;1H", rt.Height)   // Cursor to input line on prompt
}

func (rt *RawTerm) printMessage(line ChatLine) {
	rt.mu.Lock()
	defer rt.mu.Unlock()

	fmt.Print("\x1b[s")                       // save cursor (currently at input line)
	fmt.Printf("\x1b[%d;1H\r\n", rt.ChatRows) // go to last scroll line, emit newline → scrolls region
	fmt.Print("\x1b[2K")                      // clear the fresh blank line
	renderLine(line)                          // print username + parts
	fmt.Print("\x1b[u")                       // restore cursor to input line
}

func renderLine(line ChatLine) {
	fmt.Printf("\x1b[1m%s\x1b[0m: ", line.User) // bold username
	for _, part := range line.Parts {
		switch part.Kind {
		case "text":
			fmt.Print(part.Text)
		case "emote":
			if img, ok := EmoteImage(part.EmoteID); ok {
				fmt.Print(kittyInlineImage(img), " ")
			} else {
				fmt.Printf("[%s]", part.Text) // fallback until cached
			}
		}
	}
}

func (rt *RawTerm) inputLoop(broadcasterID, userID, accessToken string,
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
				go sendChatMessage(broadcasterID, userID, accessToken,
					msg)
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


func fetchEmoteBlocking(id string) {
	if _, ok := EmoteImage(id); ok {
		return
	}

	if cache.FailedEmotes[id] {
		return
	}

	url := "https://static-cdn.jtvnw.net/emoticons/v2/" + id + "/static/light/1.0"
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		cache.FailedEmotes[id] = true
		return
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		cache.FailedEmotes[id] = true
		return
	}
	cache.Emotes[id] = data
}
