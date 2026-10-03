package lettermint

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"testing"
)

const (
	sendingFixture = "lm_sendingFixture0000000000000000"
	teamFixture    = "lm_team_teamFixture000000000000000000000000000"
	baseFixture    = "https://api.lettermint.co/v1"
)

// recorded is one request seen by the fake transport.
type recorded struct {
	Method  string
	URL     *url.URL
	Header  http.Header
	RawBody []byte
}

func (r recorded) Path() string { return strings.TrimPrefix(r.URL.EscapedPath(), "/v1") }

func (r recorded) JSON(t *testing.T) map[string]any {
	t.Helper()
	var body map[string]any
	if err := json.Unmarshal(r.RawBody, &body); err != nil {
		t.Fatalf("request body is not a JSON object: %v (%s)", err, r.RawBody)
	}
	return body
}

type handler func(r recorded, index int) *http.Response

// fakeTransport records every request and answers with its handler. It never
// touches the network.
type fakeTransport struct {
	mu       sync.Mutex
	requests []recorded
	handle   handler
}

func (f *fakeTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	var body []byte
	if request.Body != nil {
		body, _ = io.ReadAll(request.Body)
	}
	r := recorded{Method: request.Method, URL: request.URL, Header: request.Header.Clone(), RawBody: body}
	f.mu.Lock()
	f.requests = append(f.requests, r)
	index := len(f.requests) - 1
	f.mu.Unlock()
	response := f.handle(r, index)
	if response == nil {
		return nil, errors.New("fake transport: no response")
	}
	response.Request = request
	return response, nil
}

func (f *fakeTransport) all() []recorded {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]recorded(nil), f.requests...)
}

func jsonResponse(status int, body any, headers ...string) *http.Response {
	data, _ := json.Marshal(body)
	return rawResponse(status, string(data), append([]string{"Content-Type", "application/json"}, headers...)...)
}

func rawResponse(status int, body string, headers ...string) *http.Response {
	header := http.Header{}
	for i := 0; i+1 < len(headers); i += 2 {
		header.Set(headers[i], headers[i+1])
	}
	return &http.Response{
		StatusCode: status,
		Status:     http.StatusText(status),
		Header:     header,
		Body:       io.NopCloser(bytes.NewBufferString(body)),
	}
}

// newTestClient returns a client with both tokens and a fake transport.
func newTestClient(t *testing.T, handle handler, options ...Option) (*Client, *fakeTransport) {
	t.Helper()
	if handle == nil {
		handle = func(recorded, int) *http.Response { return jsonResponse(200, map[string]any{}) }
	}
	fake := &fakeTransport{handle: handle}
	all := append([]Option{WithSendingToken(sendingFixture), WithTeamToken(teamFixture), WithHTTPClient(&http.Client{Transport: fake})}, options...)
	client, err := New(all...)
	if err != nil {
		t.Fatal(err)
	}
	return client, fake
}

func mustAs[T error](t *testing.T, err error) T {
	t.Helper()
	var target T
	if !errors.As(err, &target) {
		t.Fatalf("expected %T, got %T: %v", target, err, err)
	}
	return target
}
