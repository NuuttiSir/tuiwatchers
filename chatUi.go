package main

type IncomingChatMessage struct {
	User  string
	Parts []MessagePart
	Text  string
}

type ChatLine struct {
	User  string
	Parts []MessagePart
}

type SendResultMessage struct {
	Ok  bool
	Err error
}

type ClearStatusMessage struct{}

type EmoteLoadedMsg struct {
	ID      string
	Success bool
}
