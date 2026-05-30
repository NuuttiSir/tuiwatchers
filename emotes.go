package main

import (
	"encoding/base64"
	"io"
	"net/http"
	"time"

	tea "charm.land/bubbletea/v2"
)

type EmoteCache struct {
	Emotes       map[string][]byte
	FailedEmotes map[string]bool
}

var cache = &EmoteCache{
	Emotes:       make(map[string][]byte),
	FailedEmotes: make(map[string]bool),
}

func emoteImage(id string) ([]byte, bool) {
	img, ok := cache.Emotes[id]
	return img, ok
}

func FetchEmoteImage(id string) tea.Cmd {
	return func() tea.Msg {
		if _, failed := cache.FailedEmotes[id]; failed {
			return EmoteLoadedMsg{ID: id, Success: false}
		}
		if _, ok := cache.Emotes[id]; ok {
			return EmoteLoadedMsg{ID: id, Success: true}
		}

		// URLs last number tells how big the emote is, default is 28x28
		// So if its changed to 2.0 it is going to be 56x56 etc
		url := "https://static-cdn.jtvnw.net/emoticons/v2/" + id + "/static/light/1.0"
		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Get(url)
		if err != nil {
			cache.FailedEmotes[id] = true
			return EmoteLoadedMsg{ID: id, Success: false}
		}
		defer resp.Body.Close()

		data, err := io.ReadAll(resp.Body)
		if err != nil {
			cache.FailedEmotes[id] = true
			return EmoteLoadedMsg{ID: id, Success: false}
		}

		cache.Emotes[id] = data
		return EmoteLoadedMsg{ID: id, Success: true}
	}
}

// NOTE: This is ATM a signle-chunk transmission. Images larger than 4mb one
// would need to split into chunks with m=
// Twitch emotes as earlier stated with scale 1.0 are not that big
func kittyInlineImage(data []byte) string {
	encoded := base64.StdEncoding.EncodeToString(data)
	// f=100 = PNG, a=T = transmit+display, q=2 = suppress response
	// C=1    → advance cursor cell after image (inline placement)
	// c=2,r=1 → display size: 2 cols wide, 1 row tall (fits emote in a chatline)
	return "\x1b_Ga=T,f=100,C=1,c=2,r=1,q=2,m=0;" + encoded + "\x1b\\"
}
