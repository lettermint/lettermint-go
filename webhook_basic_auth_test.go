package lettermint

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestWebhookBasicAuthStatesUseBearerAuth(t *testing.T) {
	credentials := &WebhookBasicAuthData{Username: " fixture user ", Password: ""}
	var clear *WebhookBasicAuthData
	for _, tc := range []struct {
		name    string
		auth    **WebhookBasicAuthData
		present bool
		value   any
	}{
		{"omit", nil, false, nil},
		{"set", &credentials, true, map[string]any{"username": " fixture user ", "password": ""}},
		{"remove", &clear, true, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				if r.Header.Get("Authorization") != "Bearer fixture-token" || r.Header.Get("x-lettermint-token") != "" {
					t.Error("wrong API authentication")
				}
				if r.Header.Get("Content-Type") != "application/json" {
					t.Error("wrong content type")
				}
				if calls == 1 && (r.Method != "POST" || r.URL.Path != "/webhooks") {
					t.Error("wrong create route")
				}
				if calls == 2 && (r.Method != "PUT" || r.URL.Path != "/webhooks/webhook-id") {
					t.Error("wrong update route")
				}
				var body map[string]json.RawMessage
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Fatal(err)
				}
				got, present := body["basic_auth"]
				if present != tc.present {
					t.Errorf("basic_auth present = %v", present)
				}
				if present {
					var value any
					if err := json.Unmarshal(got, &value); err != nil {
						t.Fatal(err)
					}
					if !reflect.DeepEqual(value, tc.value) {
						t.Errorf("unexpected credential state")
					}
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"data":{"id":"webhook-id","has_basic_auth":true},"message":"Updated"}`))
			}))
			defer server.Close()
			api, err := NewAPI("fixture-token", WithBaseURL(server.URL))
			if err != nil {
				t.Fatal(err)
			}
			created, err := api.Webhooks.Create(context.Background(), WebhookStoreRequest{Name: "Fixture", URL: "https://example.test/hook", Events: []APIWebhookEvent{APIWebhookEventMessageSent}, BasicAuth: tc.auth})
			if err != nil || !created.Data.HasBasicAuth {
				t.Fatalf("create result: %v", err)
			}
			updated, err := api.Webhooks.Update(context.Background(), "webhook-id", WebhookUpdateRequest{BasicAuth: tc.auth})
			if err != nil || !updated.Data.HasBasicAuth {
				t.Fatalf("update result: %v", err)
			}
		})
	}
}

func TestWebhookBasicAuthReadModels(t *testing.T) {
	payload := []byte(`{"has_basic_auth":true}`)
	var detail WebhookData
	var list WebhookListData
	var secret WebhookSecretData
	for _, value := range []any{&detail, &list, &secret} {
		if err := json.Unmarshal(payload, value); err != nil {
			t.Fatal(err)
		}
	}
	if !detail.HasBasicAuth || !list.HasBasicAuth || !secret.HasBasicAuth {
		t.Fatal("missing safe credential flag")
	}
}

func TestWebhookBasicAuthHelpersAndRedaction(t *testing.T) {
	request := WebhookUpdateRequest{BasicAuth: SetWebhookBasicAuth("fixture-user", "fixture-password")}
	encoded, err := json.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(encoded), "fixture-password") {
		t.Fatal("missing wire credentials")
	}
	for _, debug := range []string{fmt.Sprintf("%v", **request.BasicAuth), fmt.Sprintf("%#v", **request.BasicAuth)} {
		if strings.Contains(debug, "fixture-user") || strings.Contains(debug, "fixture-password") {
			t.Fatal("credential debug output is not redacted")
		}
	}
	request.BasicAuth = ClearWebhookBasicAuth()
	encoded, err = json.Marshal(request)
	if err != nil || string(encoded) != `{"basic_auth":null}` {
		t.Fatal("clear helper does not send null")
	}
}

func TestOldUnkeyedWebhookLiteralNeedsMigration(t *testing.T) {
	root, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	module := "module oldcaller\n\ngo 1.21\nrequire github.com/lettermint/lettermint-go/v2 v2.7.0\nreplace github.com/lettermint/lettermint-go/v2 => " + filepath.ToSlash(root) + "\n"
	source := `package oldcaller
import lm "github.com/lettermint/lettermint-go/v2"
var _ = lm.StoreWebhookData{"Fixture", "https://example.test/hook", nil, nil, nil, nil, nil, nil, nil, nil}
`
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(module), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "caller.go"), []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "test", "-mod=mod", ".")
	command.Dir = dir
	output, err := command.CombinedOutput()
	if err == nil || !strings.Contains(string(output), "too few values in struct literal") {
		t.Fatalf("expected an unkeyed-literal compile break, got %v: %s", err, output)
	}
}
