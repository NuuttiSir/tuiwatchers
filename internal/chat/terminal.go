package chat

import (
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/NuuttiSir/tuiwatchers/internal/emotes"
	"github.com/NuuttiSir/tuiwatchers/internal/twitch"
	"golang.org/x/term"
)

type RawTerminal struct {
	FileDescriptor int
	Old            *term.State
	Width          int
	Height         int
	ChatRows       int
	mu             sync.Mutex
	inputLine      string
}

type ChatLine struct {
	User  string
	Parts []twitch.MessagePart
}

func NewRawTerm() (*RawTerminal, error) {
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
	return &RawTerminal{FileDescriptor: fileDescriptor, Old: old, Width: width, Height: height, ChatRows: height - 3}, nil
}

func (rt *RawTerminal) Restore() {
	fmt.Print("\x1b[r")
	fmt.Print("\x1b[?25h")
	fmt.Print("\x1b[2J\x1b[H")
	term.Restore(rt.FileDescriptor, rt.Old)
}

func (rt *RawTerminal) InitScreen() {
	fmt.Print("\x1b[2J")                  // Clear screen
	fmt.Printf("\x1b[1;%dr", rt.ChatRows) // scroll region = 1..ChatRows
	fmt.Printf("\x1b[%d;1H> ", rt.Height) // Cursor to input line on prompt, draw prompt carot as well
}

func (rt *RawTerminal) PrintMessage(line ChatLine) {
	for _, part := range line.Parts {
		if part.Kind == "emote" {
			emotes.FetchEmoteImage(part.EmoteID)
		}
	}
	rt.mu.Lock()

	defer rt.mu.Unlock()

	fmt.Print("\x1b[s")                       // save cursor (currently at input line)
	fmt.Printf("\x1b[%d;1H\r\n", rt.ChatRows) // go to last scroll line, emit newline → scrolls region
	fmt.Print("\x1b[2K")                      // clear the fresh blank line
	RenderLine(line)                          // print username + parts
	fmt.Print("\x1b[u")                       // restore cursor to input line
}

func (rt *RawTerminal) PrintSystem(text string) {
	rt.PrintMessage(ChatLine{User: "system",
		Parts: []twitch.MessagePart{{Kind: "text", Text: text}}})
}

func (rt *RawTerminal) WatchResize() {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGWINCH)
	go func() {
		for range ch {
			width, height, err := term.GetSize(rt.FileDescriptor)
			if err != nil {
				continue
			}
			rt.mu.Lock()
			rt.Width, rt.Height, rt.ChatRows = width, height, height-3
			fmt.Printf("\x1b[1;%dr", rt.ChatRows) // a new scroll region
			fmt.Printf("\x1b[%d;1H\x1b[2K> %s", rt.Height, rt.inputLine)
			rt.mu.Unlock()
		}
	}()
}
