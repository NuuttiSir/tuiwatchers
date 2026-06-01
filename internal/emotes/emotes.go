package emotes

import (
	"encoding/base64"
	"io"
	"net/http"
	"sync"
	"time"

	tea "charm.land/bubbletea/v2"
)

type EmoteCache struct {
	Emotes       map[string][]byte
	FailedEmotes map[string]bool
	RWM          sync.RWMutex
}

type EmoteLoadedMsg struct {
	ID      string
	Success bool
}

var cache = &EmoteCache{
	Emotes:       make(map[string][]byte),
	FailedEmotes: make(map[string]bool),
}

func EmoteImage(id string) ([]byte, bool) {
	cache.RWM.RLock()
	defer cache.RWM.RUnlock()
	img, ok := cache.Emotes[id]
	return img, ok
}

func FetchEmoteImage(id string) tea.Cmd {
	return func() tea.Msg {
		cache.RWM.RLock()
		_, failed := cache.FailedEmotes[id]
		_, ok := cache.Emotes[id]
		cache.RWM.RUnlock()

		if failed {
			return EmoteLoadedMsg{ID: id, Success: false}
		}
		if ok {
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

		cache.RWM.Lock()
		cache.Emotes[id] = data
		cache.RWM.Unlock()

		return EmoteLoadedMsg{ID: id, Success: true}
	}
}

// NOTE: This is ATM a signle-chunk transmission. Images larger than 4mb one
// would need to split into chunks with m=
// Twitch emotes as earlier stated with scale 1.0 are not that big
func KittyInlineImage(data []byte) string {
	encoded := base64.StdEncoding.EncodeToString(data)
	// f=100 = PNG, a=T = transmit+display, q=2 = suppress response
	// C=1    → advance cursor cell after image (inline placement)
	// c=2,r=1 → display size: 2 cols wide, 1 row tall (fits emote in a chatline)
	return "\x1b_Ga=T,f=100,C=1,c=2,r=1,q=2,m=0;" + encoded + "\x1b\\"
}
