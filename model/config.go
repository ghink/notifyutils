package model

import (
	"net/http"
	"time"
)

// HTTPTimeout is the request timeout the core puts on the default HTTP client it
// hands to drivers when Config.HTTPClient is nil.
const HTTPTimeout = 15 * time.Second

// Routes decides which channels receive a message, keyed by its Level.
// A level with no entry (or an empty one) is delivered to every configured driver.
type Routes = map[Level][]string

type Config struct {
	// Notifyutils
	Routes Routes
	// HTTP. When nil the core builds a client with HTTPTimeout applied.
	HTTPClient *http.Client
	// Channels' credentials
	Credentials C
	// JSON
	Unmarshal func(data []byte, v any) error
	Marshal   func(v any) ([]byte, error)
}
