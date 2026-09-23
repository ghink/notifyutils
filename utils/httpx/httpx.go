package httpx

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
)

// Request describes one call to a channel's HTTP API.
type Request struct {
	// Method defaults to POST.
	Method string
	URL    string
	Header map[string]string
	// Body is encoded as follows: nil sends no body, []byte and string are sent
	// verbatim, and any other value is JSON-encoded as application/json.
	Body any
}

type Response struct {
	StatusCode int
	Header     http.Header
	Body       []byte
}

// Do performs the request and reads the whole response body.
//
// Only transport and read failures become errors: an HTTP status the channel
// treats as a business failure is reported through Response, because every
// provider signals failure in its own body shape.
func Do(ctx context.Context, client *http.Client, req Request) (*Response, error) {
	method := req.Method
	if method == "" {
		method = http.MethodPost
	}

	var (
		body   io.Reader
		isJSON bool
	)
	switch b := req.Body.(type) {
	case nil:
	case []byte:
		body = bytes.NewReader(b)
	case string:
		body = strings.NewReader(b)
	default:
		encoded, err := json.Marshal(b)
		if err != nil {
			return nil, err
		}
		body = bytes.NewReader(encoded)
		isJSON = true
	}

	r, err := http.NewRequestWithContext(ctx, method, req.URL, body)
	if err != nil {
		return nil, err
	}
	if isJSON {
		r.Header.Set("Content-Type", "application/json; charset=utf-8")
	}
	for key, value := range req.Header {
		r.Header.Set(key, value)
	}

	resp, err := client.Do(r)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return &Response{
		StatusCode: resp.StatusCode,
		Header:     resp.Header,
		Body:       data,
	}, nil
}
