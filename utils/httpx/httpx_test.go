package httpx

import (
	"context"
	stderrors "errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// echoServer replies with a fixed status and body while recording the request it got.
func echoServer(t *testing.T, status int, body string) (*httptest.Server, *http.Request, *string) {
	t.Helper()
	var got http.Request
	var gotBody string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, _ := io.ReadAll(r.Body)
		got = *r.Clone(r.Context())
		gotBody = string(data)
		w.WriteHeader(status)
		_, _ = io.WriteString(w, body)
	}))
	t.Cleanup(srv.Close)
	return srv, &got, &gotBody
}

func TestDoPostsJSONByDefault(t *testing.T) {
	srv, got, gotBody := echoServer(t, 200, `{"ok":true}`)

	resp, err := Do(context.Background(), srv.Client(), Request{
		URL:  srv.URL,
		Body: map[string]string{"msgtype": "text"},
	})
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}

	if got.Method != http.MethodPost {
		t.Errorf("method = %s, want POST", got.Method)
	}
	if ct := got.Header.Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}
	if *gotBody != `{"msgtype":"text"}` {
		t.Errorf("request body = %q", *gotBody)
	}
	if resp.StatusCode != 200 {
		t.Errorf("StatusCode = %d, want 200", resp.StatusCode)
	}
	if string(resp.Body) != `{"ok":true}` {
		t.Errorf("response body = %q", resp.Body)
	}
}

// Raw bodies let a driver send a form encoding or a template-rendered payload
// without httpx second-guessing the content type.
func TestDoSendsRawBodiesVerbatim(t *testing.T) {
	srv, got, gotBody := echoServer(t, 200, "")

	if _, err := Do(context.Background(), srv.Client(), Request{
		Method: http.MethodPut,
		URL:    srv.URL,
		Body:   "title=hi&message=there",
		Header: map[string]string{"Content-Type": "application/x-www-form-urlencoded"},
	}); err != nil {
		t.Fatalf("Do() error = %v", err)
	}

	if got.Method != http.MethodPut {
		t.Errorf("method = %s, want PUT", got.Method)
	}
	if got.Header.Get("Content-Type") != "application/x-www-form-urlencoded" {
		t.Errorf("Content-Type = %q, want the declared form type", got.Header.Get("Content-Type"))
	}
	if *gotBody != "title=hi&message=there" {
		t.Errorf("request body = %q, want it unchanged", *gotBody)
	}
}

func TestDoWithNoBody(t *testing.T) {
	srv, got, gotBody := echoServer(t, 200, "")

	if _, err := Do(context.Background(), srv.Client(), Request{Method: http.MethodGet, URL: srv.URL}); err != nil {
		t.Fatalf("Do() error = %v", err)
	}
	if *gotBody != "" {
		t.Errorf("request body = %q, want empty", *gotBody)
	}
	if ct := got.Header.Get("Content-Type"); ct != "" {
		t.Errorf("Content-Type = %q, want none for a bodyless request", ct)
	}
}

// A driver signals failure through the returned status and body, never a Do error.
func TestDoReportsErrorStatusWithoutError(t *testing.T) {
	srv, _, _ := echoServer(t, 400, `{"error":"chat not found"}`)

	resp, err := Do(context.Background(), srv.Client(), Request{URL: srv.URL, Body: "x"})
	if err != nil {
		t.Fatalf("Do() error = %v, want nil for an HTTP-level failure", err)
	}
	if resp.StatusCode != 400 {
		t.Errorf("StatusCode = %d, want 400", resp.StatusCode)
	}
	if !strings.Contains(string(resp.Body), "chat not found") {
		t.Errorf("response body = %q, want the provider diagnostics preserved", resp.Body)
	}
}

func TestDoHeaderOverridesJSONContentType(t *testing.T) {
	srv, got, _ := echoServer(t, 200, "")

	if _, err := Do(context.Background(), srv.Client(), Request{
		URL:    srv.URL,
		Body:   map[string]string{"a": "b"},
		Header: map[string]string{"Content-Type": "application/json; charset=gbk"},
	}); err != nil {
		t.Fatalf("Do() error = %v", err)
	}
	if ct := got.Header.Get("Content-Type"); ct != "application/json; charset=gbk" {
		t.Errorf("Content-Type = %q, want the caller's override to win", ct)
	}
}

func TestDoHonoursContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// A server is not needed: an already-cancelled context must surface as an
	// error before anything reaches the transport.
	_, err := Do(ctx, http.DefaultClient, Request{URL: "http://127.0.0.1:1/", Body: "x"})
	if err == nil {
		t.Fatalf("Do() error = nil, want the cancelled context to abort the request")
	}
	if !stderrors.Is(err, context.Canceled) {
		t.Errorf("error = %v, want it to unwrap to context.Canceled", err)
	}
}

func TestDoRejectsInvalidBodyEncoding(t *testing.T) {
	// A channel value that cannot be marshalled must fail before the request goes out.
	if _, err := Do(context.Background(), http.DefaultClient, Request{
		URL:  "http://127.0.0.1:0/none",
		Body: map[string]any{"bad": make(chan int)},
	}); err == nil {
		t.Errorf("Do() error = nil, want the marshal failure")
	}
}
