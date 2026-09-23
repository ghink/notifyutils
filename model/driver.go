package model

import "net/http"

type Driver interface {
	NewClient(params DriverClientParam) (Client, error)
}

type DriverClientParam struct {
	// Channel credential
	Credential map[string]string
	// Shared HTTP client, always non-nil: the core substitutes a default with
	// HTTPTimeout applied when Config.HTTPClient is nil. Drivers that talk to an
	// upstream must use it so a user's proxy and timeout apply everywhere.
	HTTPClient *http.Client
	// JSON
	Unmarshal func(data []byte, v any) error
	Marshal   func(v any) ([]byte, error)
}
