package client

import (
	"context"
	stderrors "errors"
	"net/http"
	"strings"
	"testing"

	"go.gh.ink/notifyutils/driver"
	"go.gh.ink/notifyutils/errors"
	"go.gh.ink/notifyutils/model"
)

// recorder is a fake channel: it captures every message it was handed and replies
// with a canned error.
type recorder struct {
	name     string
	sent     []model.Message
	observed chan<- string
	fail     error
}

func (r *recorder) Send(_ context.Context, msg model.Message) error {
	r.sent = append(r.sent, msg)
	if r.observed != nil {
		r.observed <- r.name
	}
	return r.fail
}

// stubChannel is a driver that hands out one prepared recorder.
type stubChannel struct{ client model.Client }

func (d stubChannel) NewClient(_ model.DriverClientParam) (model.Client, error) {
	return d.client, nil
}

// spyDriver records the params it was constructed with.
type spyDriver struct {
	params *model.DriverClientParam
	err    error
}

func (d spyDriver) NewClient(params model.DriverClientParam) (model.Client, error) {
	if d.params != nil {
		*d.params = params
	}
	if d.err != nil {
		return nil, d.err
	}
	return &recorder{}, nil
}

// register wires a fake channel into the global registry under a unique name.
func register(t *testing.T, name string, c model.Client) {
	t.Helper()
	driver.Register(name, stubChannel{client: c})
}

func TestNewClientDriverNotRegistered(t *testing.T) {
	_, err := NewClient(model.Config{
		Credentials: model.C{"no-such-channel-xyz": {"url": "http://example.com"}},
	})

	if !stderrors.Is(err, errors.ErrDriverNotRegistered) {
		t.Fatalf("error = %v, want ErrDriverNotRegistered", err)
	}
	var typed *errors.NotifyutilsError
	if !stderrors.As(err, &typed) || typed.DriverName() != "no-such-channel-xyz" {
		t.Errorf("error should name the missing driver, got %v", err)
	}
}

func TestNewClientPassesParamsToDriver(t *testing.T) {
	var got model.DriverClientParam
	driver.Register("params-check", spyDriver{params: &got})

	httpClient := &http.Client{}
	if _, err := NewClient(model.Config{
		HTTPClient:  httpClient,
		Credentials: model.C{"params-check": {"token": "abc"}},
	}); err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	if got.Credential["token"] != "abc" {
		t.Errorf("Credential = %v, want token=abc", got.Credential)
	}
	if got.HTTPClient != httpClient {
		t.Errorf("HTTPClient = %v, want the injected client", got.HTTPClient)
	}
	if got.Marshal == nil || got.Unmarshal == nil {
		t.Errorf("Marshal/Unmarshal are nil, want the JSON defaults")
	}
}

// Drivers must never see a nil HTTP client: every channel that reaches an upstream
// is expected to honour the core's timeout.
func TestNewClientProvidesDefaultHTTPClient(t *testing.T) {
	var got model.DriverClientParam
	driver.Register("http-default-check", spyDriver{params: &got})

	c, err := NewClient(model.Config{Credentials: model.C{"http-default-check": {}}})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	if got.HTTPClient == nil {
		t.Fatalf("DriverClientParam.HTTPClient = nil, want a default client")
	}
	if got.HTTPClient.Timeout != model.HTTPTimeout {
		t.Errorf("Timeout = %v, want %v", got.HTTPClient.Timeout, model.HTTPTimeout)
	}
	if c.Config.HTTPClient != got.HTTPClient {
		t.Errorf("the resolved client on Config and on the driver params should be the same instance")
	}
}

func TestNewClientDriverConstructionFails(t *testing.T) {
	driver.Register("broken-driver", spyDriver{
		err: errors.ErrDriverCredentialInvalid.WithDriverName("broken"),
	})

	_, err := NewClient(model.Config{Credentials: model.C{"broken-driver": {}}})

	if !stderrors.Is(err, errors.ErrDriverCredentialInvalid) {
		t.Fatalf("error = %v, want ErrDriverCredentialInvalid", err)
	}
}

func TestSendUsesRouteForLevel(t *testing.T) {
	a, b := &recorder{name: "route-a"}, &recorder{name: "route-b"}
	register(t, "route-a", a)
	register(t, "route-b", b)

	c, err := NewClient(model.Config{
		Routes:      model.Routes{model.LevelCritical: []string{"route-a"}},
		Credentials: model.C{"route-a": {}, "route-b": {}},
	})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	if err := c.Send(context.Background(), model.Message{Level: model.LevelCritical, Text: "boom"}); err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	if len(a.sent) != 1 {
		t.Errorf("routed driver got %d sends, want 1", len(a.sent))
	}
	if len(b.sent) != 0 {
		t.Errorf("unrouted driver got %d sends, want 0", len(b.sent))
	}
}

// A level with no route reaches every configured channel, in a stable order.
func TestSendFallsBackToAllDrivers(t *testing.T) {
	observed := make(chan string, 4)
	c := &Client{Clients: map[string]model.Client{
		"zeta":  &recorder{name: "zeta", observed: observed},
		"alpha": &recorder{name: "alpha", observed: observed},
		"mid":   &recorder{name: "mid", observed: observed},
	}}

	if err := c.Send(context.Background(), model.Message{}); err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	close(observed)
	var got []string
	for name := range observed {
		got = append(got, name)
	}
	if strings.Join(got, ",") != "alpha,mid,zeta" {
		t.Errorf("delivery order = %v, want sorted [alpha mid zeta]", got)
	}
}

func TestSendFillsFormatAndLevelDefaults(t *testing.T) {
	rec := &recorder{}
	register(t, "defaults-check", rec)
	c, err := NewClient(model.Config{Credentials: model.C{"defaults-check": {}}})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	if err := c.Send(context.Background(), model.Message{Text: "hi"}); err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	if rec.sent[0].Format != model.FormatPlain {
		t.Errorf("Format = %q, drivers must see a concrete format", rec.sent[0].Format)
	}
	if rec.sent[0].Level != model.LevelInfo {
		t.Errorf("Level = %q, drivers must see a concrete level", rec.sent[0].Level)
	}
}

// Variable binding happens once, in the core, so that one message reaches every
// channel as the same text — and so a value that merely looks like a placeholder is
// never expanded a second time.
func TestSendRendersVarsBeforeDrivers(t *testing.T) {
	rec := &recorder{}
	register(t, "render-check", rec)
	c, err := NewClient(model.Config{Credentials: model.C{"render-check": {}}})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	vars := model.Vars{
		{Key: "host", Value: "web-01"},
		{Key: "usage", Value: "${host} 92%"}, // looks like a placeholder, must stay literal
	}
	err = c.Send(context.Background(), model.Message{
		Title: "${host} 告警",
		Text:  "使用率 ${usage}",
		Vars:  vars,
	})
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}

	got := rec.sent[0]
	if got.Title != "web-01 告警" {
		t.Errorf("Title = %q, want the placeholder bound", got.Title)
	}
	if got.Text != "使用率 ${host} 92%" {
		t.Errorf("Text = %q, want one pass with no re-expansion", got.Text)
	}
	// Vars travel on unchanged: channels that bind them as data still need them.
	if got.Vars[1].Value != "${host} 92%" {
		t.Errorf("Vars = %v, want them forwarded as given", got.Vars)
	}
}

// Rendering happens on a copy, so binding the variables never reaches back into the
// caller's own message.
func TestSendDoesNotMutateCallerMessage(t *testing.T) {
	rec := &recorder{}
	register(t, "nomutate", rec)
	c, err := NewClient(model.Config{Credentials: model.C{"nomutate": {}}})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	msg := model.Message{
		Title: "${host} 告警",
		Text:  "使用率 ${usage}%",
		Vars:  model.Vars{{Key: "host", Value: "web-01"}, {Key: "usage", Value: "92"}},
	}
	originalTitle, originalText := msg.Title, msg.Text

	if err := c.Send(context.Background(), msg); err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	if msg.Title != originalTitle || msg.Text != originalText {
		t.Errorf("Send mutated the caller's message to %q/%q", msg.Title, msg.Text)
	}
	if rec.sent[0].Text != "使用率 92%" {
		t.Errorf("driver received %q, want the rendered copy", rec.sent[0].Text)
	}
}

// An unknown placeholder stays visible rather than silently vanishing.
func TestSendLeavesUnknownPlaceholder(t *testing.T) {
	rec := &recorder{}
	register(t, "render-unknown", rec)
	c, err := NewClient(model.Config{Credentials: model.C{"render-unknown": {}}})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	if err := c.Send(context.Background(), model.Message{
		Text: "node ${missing} down",
		Vars: model.Vars{{Key: "other", Value: "x"}},
	}); err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	if rec.sent[0].Text != "node ${missing} down" {
		t.Errorf("Text = %q, want the unknown placeholder kept", rec.sent[0].Text)
	}
}

func TestSendToSkipsRouting(t *testing.T) {
	a, b := &recorder{}, &recorder{}
	register(t, "skip-a", a)
	register(t, "skip-b", b)
	c, err := NewClient(model.Config{
		Routes:      model.Routes{model.LevelInfo: []string{"skip-a"}},
		Credentials: model.C{"skip-a": {}, "skip-b": {}},
	})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	if err := c.SendTo(context.Background(), []string{"skip-b", "skip-b"}, model.Message{}); err != nil {
		t.Fatalf("SendTo() error = %v", err)
	}
	if len(b.sent) != 1 {
		t.Errorf("requested driver got %d sends, want 1 (a duplicate name must not double-send)", len(b.sent))
	}
	if len(a.sent) != 0 {
		t.Errorf("driver outside SendTo got %d sends, want 0", len(a.sent))
	}
}

func TestSendToUnknownDriver(t *testing.T) {
	c := &Client{Clients: map[string]model.Client{}}

	err := c.SendTo(context.Background(), []string{"ghost"}, model.Message{})
	if !stderrors.Is(err, errors.ErrDriverNotFound) {
		t.Fatalf("error = %v, want ErrDriverNotFound", err)
	}
	if !strings.Contains(err.Error(), "ghost") {
		t.Errorf("error %q should name the driver", err.Error())
	}
}

func TestSendWithoutAnyDriver(t *testing.T) {
	c, err := NewClient(model.Config{})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	if err := c.Send(context.Background(), model.Message{}); !stderrors.Is(err, errors.ErrNoDriverConfigured) {
		t.Fatalf("error = %v, want ErrNoDriverConfigured", err)
	}
}

// One dead channel must not stop the others, and every failure has to survive the
// aggregation both by sentinel and by attribution.
func TestSendAggregatesEveryFailure(t *testing.T) {
	transportErr := stderrors.New("dial tcp: connection refused")
	failing := &recorder{fail: errors.ErrDriverSendFailed.WithDriverName("failing").WithDriverCode("500")}
	broken := &recorder{fail: transportErr}
	ok := &recorder{}
	c := &Client{Clients: map[string]model.Client{
		"failing": failing,
		"broken":  broken,
		"ok":      ok,
	}}

	err := c.Send(context.Background(), model.Message{})
	if err == nil {
		t.Fatalf("Send() error = nil, want the combined failure")
	}
	if !stderrors.Is(err, errors.ErrDriverSendFailed) {
		t.Errorf("errors.Is(ErrDriverSendFailed) = false, want true")
	}
	if !stderrors.Is(err, transportErr) {
		t.Errorf("a plain transport error must stay matchable through the join")
	}
	for _, want := range []string{"failing", "broken"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("aggregated error %q should mention %q", err.Error(), want)
		}
	}
	// A failing channel must not stop the others.
	if len(ok.sent) != 1 {
		t.Errorf("healthy channel got %d sends, want 1", len(ok.sent))
	}
}

func TestNames(t *testing.T) {
	c := &Client{Clients: map[string]model.Client{"b": &recorder{}, "a": &recorder{}, "c": &recorder{}}}

	if got := strings.Join(c.Names(), ","); got != "a,b,c" {
		t.Errorf("Names() = %v, want sorted a,b,c", got)
	}
}
