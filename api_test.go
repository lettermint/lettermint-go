package lettermint

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestNewAPIUsesBearerAuthAndRawPing(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/ping" {
			t.Fatalf("path = %s, want /ping", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer api-token" {
			t.Fatalf("Authorization = %s, want bearer token", got)
		}
		if got := r.Header.Get("x-lettermint-token"); got != "" {
			t.Fatalf("x-lettermint-token = %s, want empty", got)
		}
		_, _ = w.Write([]byte("pong\n"))
	}))
	defer server.Close()

	api, err := NewAPI("api-token", WithBaseURL(server.URL))
	if err != nil {
		t.Fatalf("NewAPI() error = %v", err)
	}

	pong, err := api.Ping(context.Background())
	if err != nil {
		t.Fatalf("Ping() error = %v", err)
	}
	if pong != "pong" {
		t.Fatalf("Ping() = %q, want pong", pong)
	}
}

func TestAPIBlockedFileTypesUsesBearerAuth(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/blocked-file-types" {
			t.Fatalf("path = %s, want /blocked-file-types", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer api-token" {
			t.Fatalf("Authorization = %s, want bearer token", got)
		}
		if got := r.Header.Get("x-lettermint-token"); got != "" {
			t.Fatalf("x-lettermint-token = %s, want empty", got)
		}

		_ = json.NewEncoder(w).Encode(BlockedFileTypesResponse{
			Extensions: []string{"exe"},
			MimeTypes:  []string{"application/x-msdownload"},
		})
	}))
	defer server.Close()

	api, err := NewAPI("api-token", WithBaseURL(server.URL))
	if err != nil {
		t.Fatalf("NewAPI() error = %v", err)
	}

	response, err := api.BlockedFileTypes(context.Background())
	if err != nil {
		t.Fatalf("BlockedFileTypes() error = %v", err)
	}
	if len(response.Extensions) != 1 || response.Extensions[0] != "exe" {
		t.Fatalf("BlockedFileTypes().Extensions = %#v", response.Extensions)
	}
}

func TestClientPingAndSendBatchUseSendingAuth(t *testing.T) {
	seenPaths := map[string]bool{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenPaths[r.URL.Path] = true
		if got := r.Header.Get("x-lettermint-token"); got != "sending-token" {
			t.Fatalf("x-lettermint-token = %s, want sending token", got)
		}
		if got := r.Header.Get("Authorization"); got != "" {
			t.Fatalf("Authorization = %s, want empty", got)
		}

		switch r.URL.Path {
		case "/ping":
			_, _ = w.Write([]byte("pong"))
		case "/send/batch":
			var payload []SendMailRequest
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("decode payload: %v", err)
			}
			if len(payload) != 1 || payload[0].From != "from@example.com" {
				t.Fatalf("unexpected payload: %#v", payload)
			}
			_ = json.NewEncoder(w).Encode([]SendResponse{{MessageID: "msg_123", Status: "queued"}})
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	client, err := New("sending-token", WithBaseURL(server.URL))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	pong, err := client.Ping(context.Background())
	if err != nil {
		t.Fatalf("Ping() error = %v", err)
	}
	if pong != "pong" {
		t.Fatalf("Ping() = %q, want pong", pong)
	}

	_, err = client.SendBatch(context.Background(), []SendMailRequest{{
		From:    "from@example.com",
		To:      []string{"to@example.com"},
		Subject: "Hello",
	}})
	if err != nil {
		t.Fatalf("SendBatch() error = %v", err)
	}

	if !seenPaths["/ping"] || !seenPaths["/send/batch"] {
		t.Fatalf("missing expected paths: %#v", seenPaths)
	}
}

func TestAPIEndpointPathsAreTyped(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer api-token" {
			t.Fatalf("Authorization = %s, want bearer token", got)
		}

		switch r.URL.EscapedPath() {
		case "/domains/domain%2Fid", "/messages/msg%2Fid/html", "/routes/route%2Fid/verify-inbound-domain":
			_ = json.NewEncoder(w).Encode(map[string]any{"id": "ok"})
		default:
			t.Fatalf("unexpected path %s", r.URL.EscapedPath())
		}
	}))
	defer server.Close()

	api, err := NewAPI("api-token", WithBaseURL(server.URL))
	if err != nil {
		t.Fatalf("NewAPI() error = %v", err)
	}

	if _, err := api.Domains.Retrieve(context.Background(), "domain/id"); err != nil {
		t.Fatalf("Domains.Retrieve() error = %v", err)
	}
	if _, err := api.Messages.HTML(context.Background(), "msg/id"); err != nil {
		t.Fatalf("Messages.HTML() error = %v", err)
	}
	if _, err := api.Routes.VerifyInboundDomain(context.Background(), "route/id"); err != nil {
		t.Fatalf("Routes.VerifyInboundDomain() error = %v", err)
	}
}

func TestRawMessageEndpointsPreserveResponseBody(t *testing.T) {
	rawBody := "Subject: Test\r\n\r\nBody with trailing newline\n"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(rawBody))
	}))
	defer server.Close()

	api, err := NewAPI("api-token", WithBaseURL(server.URL))
	if err != nil {
		t.Fatalf("NewAPI() error = %v", err)
	}

	source, err := api.Messages.Source(context.Background(), "msg_123")
	if err != nil {
		t.Fatalf("Messages.Source() error = %v", err)
	}
	if source != rawBody {
		t.Fatalf("Messages.Source() = %q, want exact raw body %q", source, rawBody)
	}
}

func TestWebhookUpdateSerializesFalseValues(t *testing.T) {
	enabled := false
	includeMachineEvents := false

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode payload: %v", err)
		}

		if value, ok := payload["enabled"]; !ok || value != false {
			t.Fatalf("enabled = %#v, present = %v; want false", value, ok)
		}
		if value, ok := payload["include_machine_events"]; !ok || value != false {
			t.Fatalf("include_machine_events = %#v, present = %v; want false", value, ok)
		}

		_ = json.NewEncoder(w).Encode(map[string]any{"id": "webhook_123"})
	}))
	defer server.Close()

	api, err := NewAPI("api-token", WithBaseURL(server.URL))
	if err != nil {
		t.Fatalf("NewAPI() error = %v", err)
	}

	_, err = api.Webhooks.Update(context.Background(), "webhook_123", WebhookUpdateRequest{
		Enabled:              &enabled,
		IncludeMachineEvents: &includeMachineEvents,
	})
	if err != nil {
		t.Fatalf("Webhooks.Update() error = %v", err)
	}
}

func TestAPITypesMatchCurrentTeamSchema(t *testing.T) {
	if MessageEventTypeAutoReplied != MessageEventType("auto_replied") {
		t.Fatalf("MessageEventTypeAutoReplied = %q", MessageEventTypeAutoReplied)
	}
	if APIWebhookEventMessageAutoReplied != APIWebhookEvent("message.auto_replied") {
		t.Fatalf("APIWebhookEventMessageAutoReplied = %q", APIWebhookEventMessageAutoReplied)
	}
	if VolumeTier300000 != VolumeTier(300000) {
		t.Fatalf("VolumeTier300000 = %d", VolumeTier300000)
	}

	redact := false
	routeUpdate := UpdateRouteData{
		Settings: &UpdateRouteSettingsData{
			RedactEmailContent:         &redact,
			DisablePlaintextGeneration: &redact,
		},
		InboundSettings: &UpdateRouteInboundSettingsData{
			InboundSpamThreshold: floatPtr(3),
		},
	}
	projectUpdate := UpdateProjectData{RedactEmailContent: &redact}
	project := ProjectData{RedactEmailContent: true}
	projectCreate := StoreProjectData{Name: "Production", ShortToken: &redact}
	suppression := StoreSuppressionData{Reason: SuppressionReasonManual, Scope: SuppressionScopeGlobal}
	blockedFileTypes := BlockedFileTypesResponse{
		Extensions: []string{"exe"},
		MimeTypes:  []string{"application/x-msdownload"},
	}

	if routeUpdate.Settings.RedactEmailContent == nil ||
		routeUpdate.InboundSettings.InboundSpamThreshold == nil ||
		projectUpdate.RedactEmailContent == nil ||
		projectCreate.ShortToken == nil ||
		!project.RedactEmailContent ||
		suppression.Scope != SuppressionScopeGlobal ||
		blockedFileTypes.MimeTypes[0] != "application/x-msdownload" {
		t.Fatalf("generated API types do not expose current Team schema additions")
	}
}

func TestAPIExposesDocumentedOperations(t *testing.T) {
	api, err := NewAPI("api-token")
	if err != nil {
		t.Fatalf("NewAPI() error = %v", err)
	}

	methods := map[string]interface{}{
		"domain.index":                   api.Domains.List,
		"domain.store":                   api.Domains.Create,
		"domain.show":                    api.Domains.Retrieve,
		"domain.destroy":                 api.Domains.Delete,
		"domain.verifyDnsRecords":        api.Domains.VerifyDNSRecords,
		"domain.verifySpecificDnsRecord": api.Domains.VerifyDNSRecord,
		"domain.updateProjects":          api.Domains.UpdateProjects,
		"v1.ping":                        api.Ping,
		"v1.blockedFileTypes":            api.BlockedFileTypes,
		"message.index":                  api.Messages.List,
		"message.show":                   api.Messages.Retrieve,
		"message.events":                 api.Messages.Events,
		"message.source":                 api.Messages.Source,
		"message.html":                   api.Messages.HTML,
		"message.text":                   api.Messages.Text,
		"project.index":                  api.Projects.List,
		"project.store":                  api.Projects.Create,
		"project.show":                   api.Projects.Retrieve,
		"project.update":                 api.Projects.Update,
		"project.destroy":                api.Projects.Delete,
		"project.rotateToken":            api.Projects.RotateToken,
		"project.updateMembers":          api.Projects.UpdateMembers,
		"project.addMember":              api.Projects.AddMember,
		"project.removeMember":           api.Projects.RemoveMember,
		"route.index":                    api.Projects.Routes,
		"route.store":                    api.Projects.CreateRoute,
		"route.show":                     api.Routes.Retrieve,
		"route.update":                   api.Routes.Update,
		"route.destroy":                  api.Routes.Delete,
		"route.verifyInboundDomain":      api.Routes.VerifyInboundDomain,
		"stats.index":                    api.Stats.Retrieve,
		"suppression.index":              api.Suppressions.List,
		"suppression.store":              api.Suppressions.Create,
		"suppression.destroy":            api.Suppressions.Delete,
		"team.show":                      api.Team.Retrieve,
		"team.update":                    api.Team.Update,
		"team.usage":                     api.Team.Usage,
		"team.members":                   api.Team.Members,
		"webhook.index":                  api.Webhooks.List,
		"webhook.store":                  api.Webhooks.Create,
		"webhook.show":                   api.Webhooks.Retrieve,
		"webhook.update":                 api.Webhooks.Update,
		"webhook.destroy":                api.Webhooks.Delete,
		"webhook.test":                   api.Webhooks.Test,
		"webhook.regenerateSecret":       api.Webhooks.RegenerateSecret,
		"webhook.deliveries":             api.Webhooks.Deliveries,
		"webhook.showDelivery":           api.Webhooks.Delivery,
	}

	for operationID, method := range methods {
		if method == nil || reflect.ValueOf(method).Kind() != reflect.Func {
			t.Fatalf("missing SDK method for operation %s", operationID)
		}
	}
}

func floatPtr(value float64) *float64 {
	return &value
}
