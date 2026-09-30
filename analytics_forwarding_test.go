package lettermint

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAnalyticsAndReportForwarding(t *testing.T) {
	methods := []string{"POST", "GET", "PUT", "POST", "POST", "DELETE"}
	paths := []string{"/analytics", "/projects/project%2Fid/report-forwarding", "/projects/project%2Fid/report-forwarding", "/projects/project%2Fid/report-forwarding/verify", "/projects/project%2Fid/report-forwarding/resend-code", "/projects/project%2Fid/report-forwarding"}
	index := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != methods[index] || r.URL.EscapedPath() != paths[index] {
			t.Errorf("request %d: %s %s", index, r.Method, r.URL.EscapedPath())
		}
		if r.Header.Get("Authorization") != "Bearer team-token" || r.Header.Get("x-lettermint-token") != "" {
			t.Error("wrong auth")
		}
		if index == 0 || index == 2 || index == 3 {
			var body map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Error(err)
			}
			if index == 0 && body["metrics"] == nil {
				t.Error("missing metrics")
			}
			if index == 2 && body["destination"] != "reports@example.com" {
				t.Error("wrong destination")
			}
			if index == 3 && body["code"] != "123456" {
				t.Error("wrong code")
			}
		}
		index++
		if index == 6 {
			w.WriteHeader(204)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if index == 1 {
			_, _ = w.Write([]byte(`{"data":{"summary":{"metrics":{"accepted":12,"delivery_rate":null}}},"meta":{"timezone":"UTC"},"pagination":{"total_groups":0,"returned_groups":0,"next_cursor":null,"truncated":false}}`))
			return
		}
		_, _ = w.Write([]byte(`{"data":{"destination":null,"verified":false,"verified_at":null}}`))
	}))
	defer server.Close()
	api, err := NewAPI("team-token", WithBaseURL(server.URL))
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	analytics, err := api.Analytics(ctx, AnalyticsRequest{Metrics: []string{"accepted"}})
	if err != nil {
		t.Fatal(err)
	}
	if analytics.Meta.Timezone != "UTC" || analytics.Data.Summary.Metrics.DeliveryRate != nil {
		t.Fatalf("unexpected analytics response: %+v", analytics)
	}
	result, err := api.Projects.RetrieveReportForwarding(ctx, "project/id")
	if err != nil || result.Data.Destination != nil || result.Data.Verified {
		t.Fatalf("result=%+v error=%v", result, err)
	}
	if _, err = api.Projects.UpdateReportForwarding(ctx, "project/id", UpdateReportForwardingRequest{Destination: "reports@example.com"}); err != nil {
		t.Fatal(err)
	}
	if _, err = api.Projects.VerifyReportForwarding(ctx, "project/id", VerifyReportForwardingRequest{Code: "123456"}); err != nil {
		t.Fatal(err)
	}
	if _, err = api.Projects.ResendReportForwardingCode(ctx, "project/id"); err != nil {
		t.Fatal(err)
	}
	if err = api.Projects.DeleteReportForwarding(ctx, "project/id"); err != nil {
		t.Fatal(err)
	}
	if index != 6 {
		t.Fatalf("got %d requests", index)
	}
}

func TestUpdatedResponseFields(t *testing.T) {
	name := "Team name"
	request := UpdateTeamData{Name: &name}
	encoded, err := json.Marshal(request)
	if err != nil || string(encoded) != `{"name":"Team name"}` {
		t.Fatalf("%s %v", encoded, err)
	}
	var page SuppressionIndexResponse
	if err := json.Unmarshal([]byte(`{"data":[],"path":"/suppressions","per_page":1}`), &page); err != nil || page.Path == nil || *page.Path != "/suppressions" {
		t.Fatalf("%+v %v", page, err)
	}
	var created ProjectStoreResponse
	if err := json.Unmarshal([]byte(`{"data":{},"message":"Created","api_token":"project-token"}`), &created); err != nil || created.APIToken != "project-token" {
		t.Fatalf("%+v %v", created, err)
	}
	var deleted SuppressionDestroyResponse
	if err := json.Unmarshal([]byte(`{"success":true,"message":"Review","status":"review_ticket_exists","ticket_identifier":"ticket-1","confidence":0.9}`), &deleted); err != nil || deleted.TicketIdentifier != "ticket-1" {
		t.Fatalf("%+v %v", deleted, err)
	}
}
