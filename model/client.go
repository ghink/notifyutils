package model

import "context"

// Client is one configured channel. Implementations must be safe for concurrent
// use, and must not mutate the Message they are handed.
type Client interface {
	// Send delivers msg to the channel, honouring ctx for cancellation and deadlines.
	//
	// It returns an error derived from the errors package sentinels so callers can
	// match with errors.Is, or a transport error from the channel itself.
	Send(ctx context.Context, msg Message) error
}
