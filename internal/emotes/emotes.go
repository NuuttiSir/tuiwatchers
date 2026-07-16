package emotes

import (
	"encoding/base64"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

type EmoteCache struct {
	Emotes       map[string][]byte
	FailedEmotes map[string]bool
	// inflight     map[string]bool
	RWM          sync.RWMutex
}

// type EmoteLoadedMsg struct {
// 	ID      string
// 	Success bool
// }

var cache = &EmoteCache{
	Emotes:       make(map[string][]byte),
	FailedEmotes: make(map[string]bool),
	// inflight:     make(map[string]bool),
}

func EmoteImage(id string) ([]byte, bool) {
	cache.RWM.RLock()
	defer cache.RWM.RUnlock()
	img, ok := cache.Emotes[id]
	return img, ok
}

func FetchEmoteImage(id string) {
	cache.RWM.RLock()
	_, failed := cache.FailedEmotes[id]
	_, ok := cache.Emotes[id]
	cache.RWM.RUnlock()

	if failed {
		return
	}
	if ok {
		return
	}

	url := "https://static-cdn.jtvnw.net/emoticons/v2/" + id + "/static/light/1.0"
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		cache.RWM.Lock()
		cache.FailedEmotes[id] = true
		cache.RWM.Unlock()
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		cache.RWM.Lock()
		cache.FailedEmotes[id] = true
		cache.RWM.Unlock()
		return
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		cache.RWM.Lock()
		cache.FailedEmotes[id] = true
		cache.RWM.Unlock()
		return
	}

	cache.RWM.Lock()
	cache.Emotes[id] = data
	cache.RWM.Unlock()
}

// func PrefetchEmoteImage(id string) {
// 	cache.RWM.Lock()
// 	_, have := cache.Emotes[id]
// 	if have || cache.FailedEmotes[id] || cache.inflight[id] {
// 		cache.RWM.Unlock()
// 		return
// 	}
// 	cache.inflight[id] = true
// 	cache.RWM.Unlock()
//
// 	go func() {
// 		FetchEmoteImage(id)
// 		cache.RWM.Lock()
// 		delete(cache.inflight, id)
// 		cache.RWM.Unlock()
// }()
// }

func KittyInlineImage(data []byte) string {
	encoded := base64.StdEncoding.EncodeToString(data)
	const chunkSize = 4096

	if len(encoded) <= chunkSize {
		// f=100 = PNG, a=T = transmit+display, q=2 = suppress response
		// c=2,r=1 → display size: 2 cols wide, 1 row tall (fits emote in a chatline)
		// (no C key: default cursor policy advances the cursor past the image)
		return "\x1b_Ga=T,f=100,c=2,r=1,q=2;" + encoded + "\x1b\\"
	}

	var sb strings.Builder
	sb.WriteString("\x1b_Ga=T,f=100,c=2,r=1,q=2,m=1;")
	sb.WriteString(encoded[:chunkSize])
	sb.WriteString("\x1b\\")
	rest := encoded[chunkSize:]
	for len(rest) > chunkSize {
		sb.WriteString("\x1b_Gm=1;")
		sb.WriteString(rest[:chunkSize])
		sb.WriteString("\x1b\\")
		rest = rest[chunkSize:]
	}
	sb.WriteString("\x1b_Gm=0;")
	sb.WriteString(rest)
	sb.WriteString("\x1b\\")
	return sb.String()
}
