package lettermint

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func minimal() SendMailRequest {
	return SendMailRequest{From: "a@example.test", To: []string{"b@example.test"}, Subject: "x"}
}

func TestHTTPErrorMapping(t *testing.T) {
	cases := []struct {
		status int
		check  func(error) bool
	}{
		{400, func(err error) bool {
			var e *APIError
			return errors.As(err, &e) && fmt.Sprintf("%T", err) == "*lettermint.APIError"
		}},
		{401, func(err error) bool { var e *AuthenticationError; return errors.As(err, &e) }},
		{403, func(err error) bool { var e *PermissionError; return errors.As(err, &e) }},
		{404, func(err error) bool { var e *NotFoundError; return errors.As(err, &e) }},
		{409, func(err error) bool { var e *ConflictError; return errors.As(err, &e) }},
		{410, func(err error) bool { return fmt.Sprintf("%T", err) == "*lettermint.APIError" }},
		{422, func(err error) bool { var e *ValidationError; return errors.As(err, &e) }},
		{429, func(err error) bool { var e *RateLimitError; return errors.As(err, &e) }},
		{500, func(err error) bool { var e *ServerError; return errors.As(err, &e) }},
		{503, func(err error) bool { var e *ServerError; return errors.As(err, &e) }},
	}
	for _, c := range cases {
		client, _ := newTestClient(t, func(recorded, int) *http.Response {
			return jsonResponse(c.status, map[string]any{"error": map[string]any{"code": "SOME_CODE", "message": "Something failed", "details": map[string]any{"field": "x"}}})
		})
		_, err := client.Emails.Send(context.Background(), minimal())
		if !c.check(err) {
			t.Errorf("%d: wrong type %T", c.status, err)
		}
		apiErr := mustAs[*APIError](t, err)
		var sdkErr Error
		if !errors.As(err, &sdkErr) {
			t.Error("every SDK error is a lettermint.Error")
		}
		if apiErr.Status != c.status || apiErr.Code != "SOME_CODE" || apiErr.Message != "Something failed" || fmt.Sprint(apiErr.Details) != "map[field:x]" || len(apiErr.Body) == 0 {
			t.Errorf("%d: %+v", c.status, apiErr)
		}
		if err.Error() != fmt.Sprintf("lettermint: Something failed (HTTP %d, SOME_CODE)", c.status) {
			t.Errorf("%d: %s", c.status, err)
		}
	}
}

func TestLaravelValidationErrors(t *testing.T) {
	client, _ := newTestClient(t, func(recorded, int) *http.Response {
		return jsonResponse(422, map[string]any{"message": "The to field is required.", "errors": map[string]any{"to": []string{"The to field is required."}}})
	})
	_, err := client.Emails.Send(context.Background(), minimal())
	validation := mustAs[*ValidationError](t, err)
	if validation.Message != "The to field is required." || validation.Errors["to"][0] != "The to field is required." || validation.Code != "" {
		t.Fatalf("%+v", validation)
	}
	var body ValidationErrorBody
	if err := json.Unmarshal(validation.Body, &body); err != nil || body.Message != "The to field is required." {
		t.Fatal(body, err)
	}
}

func TestLegacyStringErrorCode(t *testing.T) {
	client, _ := newTestClient(t, func(recorded, int) *http.Response {
		return jsonResponse(403, map[string]any{"error": "sandbox_not_available", "message": "Sandbox is not available on the Free plan."})
	})
	_, err := client.Emails.Send(context.Background(), minimal())
	permission := mustAs[*PermissionError](t, err)
	if permission.Code != "sandbox_not_available" || permission.Message != "Sandbox is not available on the Free plan." {
		t.Fatalf("%+v", permission)
	}
}

func TestEmptyErrorBodyUsesTheStatusText(t *testing.T) {
	client, _ := newTestClient(t, func(recorded, int) *http.Response {
		r := rawResponse(404, "")
		r.Status = "404 Not Found"
		return r
	})
	_, err := client.Domains.Retrieve(context.Background(), "d", nil)
	notFound := mustAs[*NotFoundError](t, err)
	if notFound.Message != "Not Found" || notFound.Body != nil {
		t.Fatalf("%+v", notFound)
	}
}

func TestRetryAfter(t *testing.T) {
	now := time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)
	if d := parseRetryAfter("30", now); d == nil || *d != 30*time.Second {
		t.Fatal(d)
	}
	if d := parseRetryAfter(now.Add(90*time.Second+300*time.Millisecond).Format(http.TimeFormat), now); d == nil || *d != 90*time.Second {
		t.Fatal(d)
	}
	if d := parseRetryAfter(now.Add(-time.Hour).Format(http.TimeFormat), now); d == nil || *d != 0 {
		t.Fatal(d)
	}
	for _, value := range []string{"", "soon", "-1", "1.5"} {
		if parseRetryAfter(value, now) != nil {
			t.Fatal(value)
		}
	}
	client, fake := newTestClient(t, func(recorded, int) *http.Response {
		return jsonResponse(429, map[string]any{"message": "Too Many Attempts."}, "Retry-After", "12")
	})
	_, err := client.Emails.Send(context.Background(), minimal())
	rateLimit := mustAs[*RateLimitError](t, err)
	if rateLimit.RetryAfter == nil || *rateLimit.RetryAfter != 12*time.Second {
		t.Fatal(rateLimit.RetryAfter)
	}
	if len(fake.all()) != 1 {
		t.Fatal("the SDK never retries")
	}

	// A 5xx response can carry Retry-After too.
	client, _ = newTestClient(t, func(recorded, int) *http.Response {
		return jsonResponse(503, map[string]any{"message": "Service Unavailable"}, "Retry-After", "2")
	})
	_, err = client.Emails.Send(context.Background(), minimal())
	if server := mustAs[*ServerError](t, err); server.RetryAfter == nil || *server.RetryAfter != 2*time.Second {
		t.Fatal(server.RetryAfter)
	}
	client, _ = newTestClient(t, func(recorded, int) *http.Response {
		return jsonResponse(500, map[string]any{"message": "Server Error"})
	})
	_, err = client.Emails.Send(context.Background(), minimal())
	if server := mustAs[*ServerError](t, err); server.RetryAfter != nil {
		t.Fatal(*server.RetryAfter)
	}
}

func TestUnexpectedResponses(t *testing.T) {
	cases := map[string]struct {
		response *http.Response
		status   int
		message  string
	}{
		"empty 202":      {rawResponse(202, "", "Content-Type", "application/json"), 202, "empty body where JSON was expected"},
		"whitespace 200": {rawResponse(200, "  \n"), 200, "empty body"},
		"invalid JSON":   {rawResponse(200, "{not json"), 200, "not valid JSON"},
		"HTML 502":       {rawResponse(502, "<html><body><h1>502 Bad Gateway</h1></body></html>", "Content-Type", "text/html; charset=UTF-8"), 502, "not JSON (text/html)"},
		"wrong shape":    {jsonResponse(202, []string{"x"}), 202, "does not match Emails.Send"},
		"304":            {rawResponse(304, ""), 0, ""},
	}
	for label, c := range cases {
		client, _ := newTestClient(t, func(recorded, int) *http.Response { return c.response })
		_, err := client.Emails.Send(context.Background(), minimal())
		if c.status == 0 {
			mustAs[*RedirectError](t, err)
			continue
		}
		unexpected := mustAs[*UnexpectedResponseError](t, err)
		if unexpected.Status != c.status || !strings.Contains(unexpected.Message, c.message) {
			t.Errorf("%s: %+v", label, unexpected)
		}
		if label == "HTML 502" && !strings.Contains(unexpected.BodyExcerpt, "502 Bad Gateway") {
			t.Error(unexpected.BodyExcerpt)
		}
	}
	long := strings.Repeat("é", 300)
	if got := excerpt([]byte(long)); len([]rune(got)) != 201 || !strings.HasSuffix(got, "…") {
		t.Fatal(got)
	}
}

func TestUnknownEnumValuesAndFieldsDecode(t *testing.T) {
	client, _ := newTestClient(t, func(recorded, int) *http.Response {
		return jsonResponse(202, map[string]any{"message_id": "m", "status": "some_future_status", "some_future_field": map[string]any{"nested": []int{1, 2}}})
	})
	resp, err := client.Emails.Send(context.Background(), minimal())
	if err != nil || resp.Status != "some_future_status" || resp.MessageID != "m" {
		t.Fatal(resp, err)
	}
}

func TestEmptyAndTextResponses(t *testing.T) {
	client, fake := newTestClient(t, func(r recorded, _ int) *http.Response {
		if r.Method == "DELETE" {
			return rawResponse(204, "")
		}
		return rawResponse(200, "<p>Hello</p>", "Content-Type", "text/html; charset=UTF-8")
	})
	if err := client.Projects.ReportForwarding.Delete(context.Background(), "p"); err != nil {
		t.Fatal(err)
	}
	for _, read := range []func(context.Context, string, ...RequestOption) (string, error){client.Messages.HTML, client.Messages.Text, client.Messages.Source} {
		text, err := read(context.Background(), "m")
		if err != nil || text != "<p>Hello</p>" {
			t.Fatal(text, err)
		}
	}
	if fake.all()[0].Header.Get("Content-Type") != "" {
		t.Fatal("no body, no content type")
	}
}

// redirectServers starts an API that redirects every request to a foreign
// origin, and the foreign origin, which records what reaches it.
func redirectServers(t *testing.T, status int) (*httptest.Server, *atomic.Int32) {
	t.Helper()
	var foreignHits atomic.Int32
	foreign := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		foreignHits.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"message_id":"captured","status":"pending"}`))
	}))
	t.Cleanup(foreign.Close)
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, foreign.URL+r.URL.Path, status)
	}))
	t.Cleanup(api.Close)
	return api, &foreignHits
}

func TestRedirectsAreNeverFollowed(t *testing.T) {
	for _, status := range []int{301, 302, 303, 307, 308} {
		api, foreignHits := redirectServers(t, status)
		client, err := New(WithSendingToken(sendingFixture), WithTeamToken(teamFixture), WithBaseURL(api.URL+"/v1"))
		if err != nil {
			t.Fatal(err)
		}
		_, err = client.Emails.Send(context.Background(), minimal())
		if mustAs[*RedirectError](t, err).Status != status {
			t.Fatal(err)
		}
		_, err = client.Ping(context.Background())
		mustAs[*RedirectError](t, err)
		if foreignHits.Load() != 0 {
			t.Fatalf("%d: a request reached the foreign origin", status)
		}
	}
	// A caller's own client is copied, not changed, and still never follows.
	api, foreignHits := redirectServers(t, 307)
	custom := &http.Client{}
	client, _ := New(WithTeamToken(teamFixture), WithBaseURL(api.URL+"/v1"), WithHTTPClient(custom))
	_, err := client.Domains.List(context.Background(), nil)
	mustAs[*RedirectError](t, err)
	if foreignHits.Load() != 0 || custom.CheckRedirect != nil {
		t.Fatal("followed a redirect or changed the caller's client")
	}
}

func slowServer(t *testing.T, headerDelay, bodyDelay time.Duration) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-time.After(headerDelay):
		case <-r.Context().Done():
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(202)
		_, _ = w.Write([]byte(`{"message_id":`))
		w.(http.Flusher).Flush()
		select {
		case <-time.After(bodyDelay):
		case <-r.Context().Done():
			return
		}
		_, _ = w.Write([]byte(`"m","status":"pending"}`))
	}))
	t.Cleanup(server.Close)
	return server
}

func TestTimeouts(t *testing.T) {
	ctx := context.Background()
	headers := slowServer(t, 700*time.Millisecond, 0)
	client, _ := New(WithSendingToken(sendingFixture), WithBaseURL(headers.URL+"/v1"), WithTimeout(100*time.Millisecond))
	_, err := client.Emails.Send(ctx, minimal())
	if mustAs[*TimeoutError](t, err).Timeout != 100*time.Millisecond {
		t.Fatal(err)
	}

	body := slowServer(t, 0, 700*time.Millisecond)
	client, _ = New(WithSendingToken(sendingFixture), WithBaseURL(body.URL+"/v1"), WithTimeout(100*time.Millisecond))
	started := time.Now()
	_, err = client.Emails.Send(ctx, minimal())
	mustAs[*TimeoutError](t, err)
	if time.Since(started) > 500*time.Millisecond {
		t.Fatal("the timeout covers reading the body")
	}

	// A per-call timeout overrides the client's.
	client, _ = New(WithSendingToken(sendingFixture), WithBaseURL(headers.URL+"/v1"), WithTimeout(time.Minute))
	_, err = client.Emails.Send(ctx, minimal(), WithRequestTimeout(50*time.Millisecond))
	if mustAs[*TimeoutError](t, err).Timeout != 50*time.Millisecond {
		t.Fatal(err)
	}
	_, err = client.Emails.Send(ctx, minimal(), WithRequestTimeout(0))
	mustAs[*ConfigError](t, err)
}

func TestContextCancellationReturnsTheContextError(t *testing.T) {
	server := slowServer(t, 700*time.Millisecond, 0)
	client, _ := New(WithSendingToken(sendingFixture), WithBaseURL(server.URL+"/v1"))
	ctx, cancel := context.WithCancel(context.Background())
	time.AfterFunc(50*time.Millisecond, cancel)
	_, err := client.Emails.Send(ctx, minimal())
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("%T %v", err, err)
	}
	var sdkErr Error
	if errors.As(err, &sdkErr) {
		t.Fatal("user cancellation is not an SDK error")
	}

	deadline, stop := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer stop()
	_, err = client.Emails.Send(deadline, minimal())
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("the caller's deadline returns its error: %T %v", err, err)
	}

	cause := errors.New("shutting down")
	canceled, cancelCause := context.WithCancelCause(context.Background())
	cancelCause(cause)
	_, err = client.Emails.Send(canceled, minimal())
	if !errors.Is(err, cause) {
		t.Fatal(err)
	}
}

func TestConnectionErrors(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	url := server.URL
	server.Close()
	client, _ := New(WithSendingToken(sendingFixture), WithBaseURL(url+"/v1"))
	_, err := client.Emails.Send(context.Background(), minimal())
	connection := mustAs[*ConnectionError](t, err)
	if connection.Unwrap() == nil || strings.Contains(err.Error(), sendingFixture) {
		t.Fatal(err)
	}
}

func TestNilContextIsAConfigError(t *testing.T) {
	client, _ := newTestClient(t, nil)
	_, err := client.Emails.Send(nil, minimal())
	mustAs[*ConfigError](t, err)
}
