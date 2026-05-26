package main

import "os/exec"

type Terminal struct {
	Command string
	Args    []string
}

func checkTerminal() *Terminal {
	terminals := []Terminal{
		// Linux
		{"ghostty", []string{"-e"}},
		{"alacritty", []string{"-e"}},
		{"gnome-terminal", []string{"--"}},
		{"konsole", []string{"-e"}},
		{"kitty", []string{"-e"}},

		// Mac
		{"open", []string{"-a", "Terminal"}},

		// Windows
		{"wt", []string{}},
	}

	for _, terminal := range terminals {
		_, err := exec.LookPath(terminal.Command)
		if err == nil {
			return &terminal
		}
	}
	return nil
}
