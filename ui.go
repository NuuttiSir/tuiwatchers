package main

import (
	"charm.land/lipgloss/v2"
)

type page int

const (
	pageAuthentication = iota
	pageStreams
	pageQuitting
)

var docStyle = lipgloss.NewStyle().Margin(1, 2)
