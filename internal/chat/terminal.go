package chat

import (
	"fmt"
	"os"
	"sync"

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
	fmt.Print("\x1b[2J/x1b[H")
	term.Restore(rt.FileDescriptor, rt.Old)
}

func (rt *RawTerminal) InitScreen() {
	fmt.Print("\x1b[2J")                  // Clear screen
	fmt.Printf("\x1b[1;%dr", rt.ChatRows) // scroll region = 1..ChatRows
	fmt.Printf("\x1b[%d;1H", rt.Height)   // Cursor to input line on prompt
}

func (rt *RawTerminal) PrintMessage(line ChatLine) {
	rt.mu.Lock()
	defer rt.mu.Unlock()

	fmt.Print("\x1b[s")                       // save cursor (currently at input line)
	fmt.Printf("\x1b[%d;1H\r\n", rt.ChatRows) // go to last scroll line, emit newline → scrolls region
	fmt.Print("\x1b[2K")                      // clear the fresh blank line
	RenderLine(line)                          // print username + parts
	fmt.Print("\x1b[u")                       // restore cursor to input line
}
