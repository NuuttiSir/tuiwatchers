package twitch

import (
	"net/http"
	"time"
)

const ClientID string = "5kft01sjf8paema7idj04jakt7hlym"

var HTTPClient = &http.Client{Timeout: 10 * time.Second}
