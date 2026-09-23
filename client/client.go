package client

import (
	"encoding/json"
	"net/http"

	"go.gh.ink/notifyutils/errors"
	"go.gh.ink/notifyutils/internal/state"
	"go.gh.ink/notifyutils/model"
)

type Client struct {
	Clients map[string]model.Client
	Config  model.Config
}

func NewClient(config model.Config) (client *Client, err error) {
	// Prepare JSON
	if config.Marshal == nil {
		config.Marshal = json.Marshal
	}
	if config.Unmarshal == nil {
		config.Unmarshal = json.Unmarshal
	}
	// Prepare HTTP: drivers never see a nil client, so a user's proxy and
	// timeout apply to every channel that talks to an upstream.
	if config.HTTPClient == nil {
		config.HTTPClient = &http.Client{Timeout: model.HTTPTimeout}
	}

	clients := make(map[string]model.Client)

	for driverName, driverCredential := range config.Credentials {
		// Check driver registered
		newDriverClient, ok := state.Drivers[driverName]
		if !ok {
			return nil, errors.ErrDriverNotRegistered.WithDriverName(driverName)
		}

		// Create driver client
		clients[driverName], err = newDriverClient.NewClient(model.DriverClientParam{
			// Channel credential
			Credential: driverCredential,
			// HTTP
			HTTPClient: config.HTTPClient,
			// JSON
			Marshal:   config.Marshal,
			Unmarshal: config.Unmarshal,
		})
		if err != nil {
			return nil, err
		}
	}

	return &Client{
		Clients: clients,
		Config:  config,
	}, nil
}
