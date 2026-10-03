package lettermint

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestNewRequiresAToken(t *testing.T) {
	_, err := New()
	if mustAs[*ConfigError](t, err).Message != "pass WithSendingToken, WithTeamToken or both" {
		t.Fatal(err)
	}
	for _, option := range []Option{WithSendingToken(""), WithTeamToken("")} {
		if _, err := New(option); err == nil {
			t.Fatal("an empty token is a config error")
		}
	}
	_, err = New(WithSendingToken("lm_abc def"))
	if !strings.Contains(mustAs[*ConfigError](t, err).Message, "WithSendingToken") || strings.Contains(err.Error(), "lm_abc") {
		t.Fatalf("names the option, never the token: %v", err)
	}
}

func TestNewValidatesOptions(t *testing.T) {
	for _, baseURL := range []string{"api.lettermint.co", "ftp://api.lettermint.co", "https://user:pass@api.lettermint.co", "https://api.lettermint.co/v1?x=1", "https://api.lettermint.co/v1#f", ""} {
		if _, err := New(WithSendingToken(sendingFixture), WithBaseURL(baseURL)); err == nil {
			t.Errorf("%q: expected a config error", baseURL)
		}
	}
	for _, timeout := range []time.Duration{0, -time.Second} {
		_, err := New(WithSendingToken(sendingFixture), WithTimeout(timeout))
		mustAs[*ConfigError](t, err)
	}
}

func TestBaseURLTrailingSlashes(t *testing.T) {
	client, fake := newTestClient(t, func(recorded, int) *http.Response { return rawResponse(200, "pong") }, WithBaseURL("http://localhost:8080/v1//"))
	if _, err := client.Ping(context.Background()); err != nil {
		t.Fatal(err)
	}
	if got := fake.all()[0].URL.String(); got != "http://localhost:8080/v1/ping" {
		t.Fatal(got)
	}
}

func TestNewFromTokenDetectsTheTokenType(t *testing.T) {
	cases := map[string]string{
		"lm_team_abc123":                    "team",
		"lm_team_Conformance0Token1Fake2":   "team",
		"lm_Proj32Conformance0Token1Fake2V": "sending",
		"lm_abc":                            "sending",
	}
	for token, kind := range cases {
		fake := &fakeTransport{handle: func(recorded, int) *http.Response { return rawResponse(200, "pong") }}
		client, err := NewFromToken(token, WithHTTPClient(&http.Client{Transport: fake}))
		if err != nil {
			t.Fatalf("%s: %v", token, err)
		}
		if _, err := client.Ping(context.Background()); err != nil {
			t.Fatal(err)
		}
		header := fake.all()[0].Header
		if kind == "team" && (header.Get("Authorization") != "Bearer "+token || header.Get("x-lettermint-token") != "") {
			t.Errorf("%s should be a team token: %v", token, header)
		}
		if kind == "sending" && (header.Get("x-lettermint-token") != token || header.Get("Authorization") != "") {
			t.Errorf("%s should be a sending token: %v", token, header)
		}
	}
	for _, token := range []string{"", "lm_", "lm_team_", "lm_sso_abc123", "lm_with-dash", "eyJhbGciOiJIUzI1NiJ9.e30.x", "sk_live_abc", " lm_abc"} {
		_, err := NewFromToken(token)
		configErr := mustAs[*ConfigError](t, err)
		if !strings.Contains(configErr.Message, "unrecognised token format") || (len(token) > 3 && strings.Contains(err.Error(), token)) {
			t.Errorf("%q: %v", token, err)
		}
	}
	if _, err := NewFromToken("lm_abc", WithTeamToken(teamFixture)); err == nil {
		t.Fatal("token options cannot be combined with NewFromToken")
	}
}

func TestSingleTokenClientsNeverFallBack(t *testing.T) {
	ctx := context.Background()
	fake := &fakeTransport{handle: func(recorded, int) *http.Response { return jsonResponse(200, map[string]any{"data": []any{}}) }}
	sendingOnly, _ := New(WithSendingToken(sendingFixture), WithHTTPClient(&http.Client{Transport: fake}))
	_, err := sendingOnly.Domains.List(ctx, nil)
	if msg := mustAs[*ConfigError](t, err).Message; msg != "Domains.List needs a team token; create the client with lettermint.WithTeamToken" {
		t.Fatal(msg)
	}
	_, err = sendingOnly.Webhooks.Deliveries.Retrieve(ctx, "w", "d")
	if !strings.HasPrefix(mustAs[*ConfigError](t, err).Message, "Webhooks.Deliveries.Retrieve needs a team token") {
		t.Fatal(err)
	}
	teamOnly, _ := New(WithTeamToken(teamFixture), WithHTTPClient(&http.Client{Transport: fake}))
	_, err = teamOnly.Emails.Send(ctx, SendMailRequest{From: "a@example.test", To: []string{"b@example.test"}, Subject: "x"})
	if msg := mustAs[*ConfigError](t, err).Message; msg != "Emails.Send needs a sending token; create the client with lettermint.WithSendingToken" {
		t.Fatal(msg)
	}
	_, err = teamOnly.Emails.Compose().From("a@example.test").Send(ctx)
	mustAs[*ConfigError](t, err)
	_, err = teamOnly.Emails.Ping(ctx)
	mustAs[*ConfigError](t, err)
	if len(fake.all()) != 0 {
		t.Fatal("no request is sent without the right token")
	}
}

func TestPingAndSchedulingUseEitherToken(t *testing.T) {
	ctx := context.Background()
	handle := func(r recorded, _ int) *http.Response {
		if strings.HasSuffix(r.URL.Path, "/ping") {
			return rawResponse(200, " pong\n", "Content-Type", "text/html; charset=UTF-8")
		}
		return jsonResponse(200, map[string]any{"message_id": "m", "status": "canceled", "scheduled_at": nil})
	}
	both, fake := newTestClient(t, handle)
	if pong, err := both.Ping(ctx); err != nil || pong != "pong" {
		t.Fatal(pong, err)
	}
	if got := fake.all()[0].Header.Get("Authorization"); got != "Bearer "+teamFixture {
		t.Fatal(got)
	}
	if _, err := both.Emails.Ping(ctx); err != nil {
		t.Fatal(err)
	}
	if got := fake.all()[1].Header; got.Get("x-lettermint-token") != sendingFixture || got.Get("Authorization") != "" {
		t.Fatal(got)
	}

	fake2 := &fakeTransport{handle: handle}
	sendingOnly, _ := New(WithSendingToken(sendingFixture), WithHTTPClient(&http.Client{Transport: fake2}))
	if _, err := sendingOnly.Ping(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := sendingOnly.Messages.Cancel(ctx, "m"); err != nil {
		t.Fatal(err)
	}
	if _, err := sendingOnly.Messages.Reschedule(ctx, "m", RescheduleMessageRequest{ScheduledAt: "tomorrow 9am"}); err != nil {
		t.Fatal(err)
	}
	for _, r := range fake2.all() {
		if r.Header.Get("x-lettermint-token") != sendingFixture || r.Header.Get("Authorization") != "" {
			t.Fatalf("%s %s: %v", r.Method, r.URL, r.Header)
		}
	}
}

func TestWithHTTPClientIsNotChanged(t *testing.T) {
	custom := &http.Client{Timeout: 7 * time.Second}
	if _, err := New(WithSendingToken(sendingFixture), WithHTTPClient(custom), WithTimeout(time.Second)); err != nil {
		t.Fatal(err)
	}
	if custom.Timeout != 7*time.Second || custom.CheckRedirect != nil {
		t.Fatal("the caller's http.Client must not be changed")
	}
}

// renderings returns every common way a value ends up in logs.
func renderings(t *testing.T, value any) map[string]string {
	t.Helper()
	out := map[string]string{}
	for _, verb := range []string{"%v", "%+v", "%#v", "%s", "%q"} {
		out[verb] = fmt.Sprintf(verb, value)
		out["&"+verb] = fmt.Sprintf(verb, &value)
	}
	out["Sprint"] = fmt.Sprint(value)
	out["Println"] = fmt.Sprintln(value)
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	out["json"] = string(data)
	var buffer bytes.Buffer
	slog.New(slog.NewJSONHandler(&buffer, nil)).Info("value", "value", value)
	slog.New(slog.NewTextHandler(&buffer, nil)).Info("value", "value", value)
	out["slog"] = buffer.String()
	return out
}

func assertNoSecret(t *testing.T, label string, value any, secrets ...string) {
	t.Helper()
	for how, text := range renderings(t, value) {
		for _, s := range secrets {
			if strings.Contains(text, s) {
				t.Errorf("%s via %s shows a secret: %s", label, how, text)
			}
		}
	}
}

func TestTokensNeverShow(t *testing.T) {
	client, _ := newTestClient(t, nil)
	subjects := map[string]any{
		"client":             client,
		"client value":       *client,
		"emails":             client.Emails,
		"domains":            client.Domains,
		"messages":           client.Messages,
		"projects":           client.Projects,
		"report forwarding":  client.Projects.ReportForwarding,
		"routes":             client.Routes,
		"stats":              client.Stats,
		"suppressions":       client.Suppressions,
		"team":               client.Team,
		"team members":       client.Team.Members,
		"webhooks":           client.Webhooks,
		"webhook deliveries": client.Webhooks.Deliveries,
		"builder":            client.Emails.Compose().From("a@example.test").To("b@example.test"),
	}
	for label, value := range subjects {
		assertNoSecret(t, label, value, sendingFixture, teamFixture)
	}
	// Basic Auth credentials are request data: JSON keeps the password, the
	// fmt verbs and slog of the value itself do not show it.
	for label, value := range map[string]any{
		"basic auth":             WebhookBasicAuthData{Username: "user", Password: "hunter2-password"},
		"update with basic auth": UpdateWebhookData{BasicAuth: Value(WebhookBasicAuthData{Username: "user", Password: "hunter2-password"})},
		"store with basic auth":  &StoreWebhookData{BasicAuth: Value(WebhookBasicAuthData{Username: "user", Password: "hunter2-password"})},
	} {
		for how, text := range renderings(t, value) {
			if how != "json" && how != "slog" && strings.Contains(text, "hunter2-password") {
				t.Errorf("%s via %s shows the password: %s", label, how, text)
			}
		}
	}
	var buffer bytes.Buffer
	slog.New(slog.NewTextHandler(&buffer, nil)).Info("auth", "auth", WebhookBasicAuthData{Username: "user", Password: "hunter2-password"})
	if strings.Contains(buffer.String(), "hunter2-password") {
		t.Fatal(buffer.String())
	}
}

func TestClientShowsWhichTokensAreSet(t *testing.T) {
	client, _ := New(WithSendingToken(sendingFixture))
	got := fmt.Sprintf("%+v", client)
	want := "lettermint.Client{BaseURL: https://api.lettermint.co/v1, Timeout: 30s, SendingToken: [redacted], TeamToken: <not set>}"
	if got != want {
		t.Fatalf("%s\n%s", got, want)
	}
	data, _ := json.Marshal(client)
	if string(data) != `{"BaseURL":"https://api.lettermint.co/v1","SendingToken":"[redacted]","Timeout":"30s"}` {
		t.Fatal(string(data))
	}
	if fmt.Sprint(client.Domains) != "lettermint.DomainsService{}" {
		t.Fatal(fmt.Sprint(client.Domains))
	}
}
