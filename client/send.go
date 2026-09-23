package client

import (
	"context"
	"errors"
	"sort"

	nuerrors "go.gh.ink/notifyutils/errors"
	"go.gh.ink/notifyutils/model"
	"go.gh.ink/notifyutils/utils/text"
)

// Send delivers msg to the channels routed for its level, and reports the
// combined outcome. Every routed channel is attempted even if an earlier one
// fails, so one dead channel cannot swallow a notification.
func (c *Client) Send(ctx context.Context, msg model.Message) error {
	msg = prepare(msg)
	return c.send(ctx, c.routeFor(msg.Level), msg)
}

// SendTo delivers msg to the named channels only, bypassing routing.
func (c *Client) SendTo(ctx context.Context, drivers []string, msg model.Message) error {
	return c.send(ctx, dedupe(drivers), prepare(msg))
}

// Names lists the configured channels in sorted order.
func (c *Client) Names() []string {
	names := make([]string, 0, len(c.Clients))
	for name := range c.Clients {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func (c *Client) send(ctx context.Context, drivers []string, msg model.Message) error {
	if len(drivers) == 0 {
		return nuerrors.ErrNoDriverConfigured
	}

	var errs []error
	for _, name := range drivers {
		driver, ok := c.Clients[name]
		if !ok {
			// Wrapped like any other per-channel failure: the sentinel's own text
			// carries no driver name, so an aggregate would otherwise not say which
			// route entry was wrong.
			errs = append(errs, named(name, nuerrors.ErrDriverNotFound.WithDriverName(name)))
			continue
		}
		if err := driver.Send(ctx, msg); err != nil {
			errs = append(errs, named(name, err))
		}
	}

	return errors.Join(errs...)
}

// routeFor resolves the channels for a level: the configured route when it has
// entries, otherwise every configured channel in a stable order.
func (c *Client) routeFor(level model.Level) []string {
	if route, ok := c.Config.Routes[level]; ok && len(route) > 0 {
		return dedupe(route)
	}
	return c.Names()
}

// prepare fills the fields the core guarantees drivers always see set, and binds
// template variables into the body. Doing the substitution once here is what makes one
// message fan out as the same text to every channel: drivers receive rendered strings
// and must not run Subst again, since a value that itself contains "${...}" would then
// be expanded a second time.
func prepare(msg model.Message) model.Message {
	if msg.Format == "" {
		msg.Format = model.FormatPlain
	}
	if msg.Level == "" {
		msg.Level = model.LevelInfo
	}
	if len(msg.Vars) > 0 {
		msg.Title = text.Subst(msg.Title, msg.Vars)
		msg.Text = text.Subst(msg.Text, msg.Vars)
	}
	return msg
}

func dedupe(names []string) []string {
	seen := make(map[string]bool, len(names))
	list := make([]string, 0, len(names))
	for _, name := range names {
		if seen[name] {
			continue
		}
		seen[name] = true
		list = append(list, name)
	}
	return list
}

// nameError attributes a channel-side error to its driver, since a raw transport
// error carries no hint of which channel produced it. Sentinel matching still
// works through the unwrap chain.
type nameError struct {
	driver string
	err    error
}

func named(driver string, err error) error {
	return nameError{driver: driver, err: err}
}

func (e nameError) Error() string {
	return e.driver + ": " + e.err.Error()
}

func (e nameError) Unwrap() error {
	return e.err
}
