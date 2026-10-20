package lettermint

import (
	"context"
	"net/http"
	"net/url"
	"reflect"
	"sort"
	"strings"
	"testing"
)

var testIDs = map[string]string{
	"domainId":      "domain/1",
	"recordId":      "record 1",
	"messageId":     "message?1",
	"projectId":     "project#1",
	"routeId":       "route_1",
	"suppressionId": "suppression_1",
	"userId":        "user/id",
	"webhookId":     "webhook_1",
	"deliveryId":    "delivery_1",
	"inviteId":      "invite/1",
	"roleId":        "role/1",
}

func id(name string) string { return testIDs[name] }

// operationCalls calls every operation of the API through its public method.
func operationCalls() map[operationKey]func(context.Context, *Client) error {
	return map[operationKey]func(context.Context, *Client) error{
		opDeleteDomain: func(ctx context.Context, c *Client) error {
			_, err := c.Domains.Delete(ctx, id("domainId"))
			return err
		},
		opDeleteProject: func(ctx context.Context, c *Client) error {
			_, err := c.Projects.Delete(ctx, id("projectId"))
			return err
		},
		opDeleteReportForwarding: func(ctx context.Context, c *Client) error {
			return c.Projects.ReportForwarding.Delete(ctx, id("projectId"))
		},
		opDeleteRoute: func(ctx context.Context, c *Client) error { _, err := c.Routes.Delete(ctx, id("routeId")); return err },
		opDeleteSuppression: func(ctx context.Context, c *Client) error {
			_, err := c.Suppressions.Delete(ctx, id("suppressionId"))
			return err
		},
		opDeleteWebhook: func(ctx context.Context, c *Client) error {
			_, err := c.Webhooks.Delete(ctx, id("webhookId"))
			return err
		},
		opListBlockedFileTypes: func(ctx context.Context, c *Client) error { _, err := c.BlockedFileTypes(ctx); return err },
		opListDomains:          func(ctx context.Context, c *Client) error { _, err := c.Domains.List(ctx, nil); return err },
		opGetDomain: func(ctx context.Context, c *Client) error {
			_, err := c.Domains.Retrieve(ctx, id("domainId"), nil)
			return err
		},
		opListMessages: func(ctx context.Context, c *Client) error { _, err := c.Messages.List(ctx, nil); return err },
		opGetMessage: func(ctx context.Context, c *Client) error {
			_, err := c.Messages.Retrieve(ctx, id("messageId"))
			return err
		},
		opListMessageEvents: func(ctx context.Context, c *Client) error {
			_, err := c.Messages.Events(ctx, id("messageId"), nil)
			return err
		},
		opGetMessageHtml: func(ctx context.Context, c *Client) error {
			_, err := c.Messages.HTML(ctx, id("messageId"))
			return err
		},
		opGetMessageSource: func(ctx context.Context, c *Client) error {
			_, err := c.Messages.Source(ctx, id("messageId"))
			return err
		},
		opGetMessageText: func(ctx context.Context, c *Client) error {
			_, err := c.Messages.Text(ctx, id("messageId"))
			return err
		},
		opPing:         func(ctx context.Context, c *Client) error { _, err := c.Ping(ctx); return err },
		opListProjects: func(ctx context.Context, c *Client) error { _, err := c.Projects.List(ctx, nil); return err },
		opGetProject: func(ctx context.Context, c *Client) error {
			_, err := c.Projects.Retrieve(ctx, id("projectId"), nil)
			return err
		},
		opGetReportForwarding: func(ctx context.Context, c *Client) error {
			_, err := c.Projects.ReportForwarding.Retrieve(ctx, id("projectId"))
			return err
		},
		opListRoutes: func(ctx context.Context, c *Client) error {
			_, err := c.Routes.List(ctx, id("projectId"), nil)
			return err
		},
		opGetRoute: func(ctx context.Context, c *Client) error {
			_, err := c.Routes.Retrieve(ctx, id("routeId"), nil)
			return err
		},
		opGetStats: func(ctx context.Context, c *Client) error {
			_, err := c.Stats.Retrieve(ctx, GetStatsQuery{From: "2026-01-01", To: "2026-01-31"})
			return err
		},
		opListSuppressions: func(ctx context.Context, c *Client) error { _, err := c.Suppressions.List(ctx, nil); return err },
		opGetTeam:          func(ctx context.Context, c *Client) error { _, err := c.Team.Retrieve(ctx, nil); return err },
		opListTeamMembers:  func(ctx context.Context, c *Client) error { _, err := c.Team.Members.List(ctx, nil); return err },
		opGetTeamMember: func(ctx context.Context, c *Client) error {
			_, err := c.Team.Members.Retrieve(ctx, id("userId"))
			return err
		},
		opListTeamRoles: func(ctx context.Context, c *Client) error { _, err := c.Team.Roles(ctx); return err },
		opGetTeamUsage:  func(ctx context.Context, c *Client) error { _, err := c.Team.Usage(ctx); return err },
		opListWebhooks:  func(ctx context.Context, c *Client) error { _, err := c.Webhooks.List(ctx, nil); return err },
		opGetWebhook: func(ctx context.Context, c *Client) error {
			_, err := c.Webhooks.Retrieve(ctx, id("webhookId"))
			return err
		},
		opListWebhookDeliveries: func(ctx context.Context, c *Client) error {
			_, err := c.Webhooks.Deliveries.List(ctx, id("webhookId"), nil)
			return err
		},
		opGetWebhookDelivery: func(ctx context.Context, c *Client) error {
			_, err := c.Webhooks.Deliveries.Retrieve(ctx, id("webhookId"), id("deliveryId"))
			return err
		},
		opRetryWebhookDelivery: func(ctx context.Context, c *Client) error {
			_, err := c.Webhooks.Deliveries.Retry(ctx, id("webhookId"), id("deliveryId"))
			return err
		},
		opRescheduleMessage: func(ctx context.Context, c *Client) error {
			_, err := c.Messages.Reschedule(ctx, id("messageId"), RescheduleMessageRequest{ScheduledAt: "2026-10-01T09:00:00Z"})
			return err
		},
		opQueryAnalytics: func(ctx context.Context, c *Client) error {
			_, err := c.Analytics(ctx, AnalyticsQuery{Metrics: []AnalyticsMetric{AnalyticsMetricAccepted}})
			return err
		},
		opCreateDomain: func(ctx context.Context, c *Client) error {
			_, err := c.Domains.Create(ctx, StoreDomainData{Domain: "example.test"})
			return err
		},
		opVerifyDomainDnsRecords: func(ctx context.Context, c *Client) error {
			_, err := c.Domains.VerifyDNSRecords(ctx, id("domainId"))
			return err
		},
		opVerifyDomainDnsRecord: func(ctx context.Context, c *Client) error {
			_, err := c.Domains.VerifyDNSRecord(ctx, id("domainId"), id("recordId"))
			return err
		},
		opCancelScheduledMessage: func(ctx context.Context, c *Client) error {
			_, err := c.Messages.Cancel(ctx, id("messageId"))
			return err
		},
		opProcessInboundMessage: func(ctx context.Context, c *Client) error {
			_, err := c.Messages.Process(ctx, id("messageId"))
			return err
		},
		opResendMessage: func(ctx context.Context, c *Client) error {
			_, err := c.Messages.Resend(ctx, id("messageId"), ResendMessageRequest{})
			return err
		},
		opCreateProject: func(ctx context.Context, c *Client) error {
			_, err := c.Projects.Create(ctx, StoreProjectData{Name: "Production"})
			return err
		},
		opResendReportForwardingCode: func(ctx context.Context, c *Client) error {
			_, err := c.Projects.ReportForwarding.ResendCode(ctx, id("projectId"))
			return err
		},
		opVerifyReportForwarding: func(ctx context.Context, c *Client) error {
			_, err := c.Projects.ReportForwarding.Verify(ctx, id("projectId"), VerifyReportForwardingRequest{Code: "123456"})
			return err
		},
		opRotateProjectToken: func(ctx context.Context, c *Client) error {
			_, err := c.Projects.RotateToken(ctx, id("projectId")) //nolint:staticcheck // the deprecated endpoint is still callable
			return err
		},
		opCreateRoute: func(ctx context.Context, c *Client) error {
			_, err := c.Routes.Create(ctx, id("projectId"), StoreRouteData{Name: "Inbound", RouteType: RouteTypeInbound})
			return err
		},
		opVerifyRouteInboundDomain: func(ctx context.Context, c *Client) error {
			_, err := c.Routes.VerifyInboundDomain(ctx, id("routeId"))
			return err
		},
		opSendMail: func(ctx context.Context, c *Client) error { _, err := c.Emails.Send(ctx, minimal()); return err },
		opSendBatchMail: func(ctx context.Context, c *Client) error {
			_, err := c.Emails.SendBatch(ctx, []SendMailRequest{minimal()})
			return err
		},
		opCreateSuppressions: func(ctx context.Context, c *Client) error {
			_, err := c.Suppressions.Create(ctx, StoreSuppressionData{Reason: SuppressionCreateReasonManual, Scope: SuppressionCreateScopeTeam, Emails: Value([]string{"blocked@example.test"})})
			return err
		},
		opCreateWebhook: func(ctx context.Context, c *Client) error {
			_, err := c.Webhooks.Create(ctx, StoreWebhookData{Name: "Hook", URL: "https://example.test/hook", Events: []WebhookEvent{WebhookEventMessageSent}})
			return err
		},
		opRegenerateWebhookSecret: func(ctx context.Context, c *Client) error {
			_, err := c.Webhooks.RegenerateSecret(ctx, id("webhookId"))
			return err
		},
		opTestWebhook: func(ctx context.Context, c *Client) error {
			_, err := c.Webhooks.Test(ctx, id("webhookId"))
			return err
		},
		opUpdateDomainProjects: func(ctx context.Context, c *Client) error {
			_, err := c.Domains.UpdateProjects(ctx, id("domainId"), UpdateDomainProjectsData{ProjectIDs: []string{"p"}})
			return err
		},
		opUpdateProject: func(ctx context.Context, c *Client) error {
			_, err := c.Projects.Update(ctx, id("projectId"), UpdateProjectData{Name: Value("Renamed")})
			return err
		},
		opUpdateReportForwarding: func(ctx context.Context, c *Client) error {
			_, err := c.Projects.ReportForwarding.Update(ctx, id("projectId"), ReportForwardingRequest{Destination: "reports@example.test"})
			return err
		},
		opUpdateRoute: func(ctx context.Context, c *Client) error {
			_, err := c.Routes.Update(ctx, id("routeId"), UpdateRouteData{Name: Value("Renamed")})
			return err
		},
		opUpdateTeam: func(ctx context.Context, c *Client) error {
			_, err := c.Team.Update(ctx, UpdateTeamData{Name: Ptr("Acme")})
			return err
		},
		opUpdateTeamMemberAssignment: func(ctx context.Context, c *Client) error {
			_, err := c.Team.Members.UpdateAssignment(ctx, id("userId"), UpdateTeamMemberAssignmentData{RoleID: "role_1", ProjectAccess: UpdateTeamMemberAssignmentDataProjectAccess{Scope: ProjectAccessScopeAll}})
			return err
		},
		opListTeamInvites: func(ctx context.Context, c *Client) error { _, err := c.Team.Invites.List(ctx, nil); return err },
		opGetTeamInvite: func(ctx context.Context, c *Client) error {
			_, err := c.Team.Invites.Retrieve(ctx, id("inviteId"))
			return err
		},
		opCreateTeamInvite: func(ctx context.Context, c *Client) error {
			_, err := c.Team.Invites.Create(ctx, CreateTeamInviteData{Email: "jane@example.test", RoleID: "role_1", ProjectAccess: CreateTeamInviteDataProjectAccess{Scope: ProjectAccessScopeAll}})
			return err
		},
		opUpdateTeamInvite: func(ctx context.Context, c *Client) error {
			_, err := c.Team.Invites.Update(ctx, id("inviteId"), UpdateTeamInviteData{Email: "jane@example.test", RoleID: "role_1", ProjectAccess: UpdateTeamInviteDataProjectAccess{Scope: ProjectAccessScopeAll}})
			return err
		},
		opDeleteTeamInvite: func(ctx context.Context, c *Client) error {
			_, err := c.Team.Invites.Revoke(ctx, id("inviteId"))
			return err
		},
		opGetTeamRole: func(ctx context.Context, c *Client) error {
			_, err := c.Team.RetrieveRole(ctx, id("roleId"))
			return err
		},
		opCreateTeamRole: func(ctx context.Context, c *Client) error {
			_, err := c.Team.CreateRole(ctx, CreateTeamRoleData{Name: "Support", Permissions: []RbacPermission{RbacPermissionMembersRead}})
			return err
		},
		opUpdateTeamRole: func(ctx context.Context, c *Client) error {
			_, err := c.Team.UpdateRole(ctx, id("roleId"), UpdateTeamRoleData{Name: "Support", Permissions: []RbacPermission{RbacPermissionMembersRead}})
			return err
		},
		opDeleteTeamRole: func(ctx context.Context, c *Client) error {
			_, err := c.Team.DeleteRole(ctx, id("roleId"), nil)
			return err
		},
		opDuplicateTeamRole: func(ctx context.Context, c *Client) error {
			_, err := c.Team.DuplicateRole(ctx, id("roleId"), DuplicateTeamRoleData{Name: "Support copy"})
			return err
		},
		opUpdateWebhook: func(ctx context.Context, c *Client) error {
			_, err := c.Webhooks.Update(ctx, id("webhookId"), UpdateWebhookData{BasicAuth: Null[WebhookBasicAuthData]()})
			return err
		},
	}
}

func responseFor(key operationKey) *http.Response {
	op := operations[key]
	switch op.Response.Type {
	case "empty":
		return rawResponse(204, "")
	case "text":
		return rawResponse(200, "pong", "Content-Type", "text/plain")
	case "SendBatchMailResponse":
		return jsonResponse(op.Response.Status[0], []any{})
	}
	if op.Pagination != nil {
		return jsonResponse(op.Response.Status[0], map[string]any{"data": []any{}, "next_cursor": nil})
	}
	return jsonResponse(op.Response.Status[0], map[string]any{})
}

// pendingOperations have no SDK method yet. The member removal method is
// added by the next PR in this stack, which removes this list.
var pendingOperations = map[operationKey]bool{
	opDeleteTeamMember: true,
}

func TestEveryOperationHasAMethod(t *testing.T) {
	calls := operationCalls()
	if len(operations) != 71 || len(calls)+len(pendingOperations) != len(operations) {
		t.Fatalf("%d operations, %d calls, %d pending", len(operations), len(calls), len(pendingOperations))
	}
	keys := make([]string, 0, len(operations))
	for key := range operations {
		keys = append(keys, string(key))
	}
	sort.Strings(keys)
	for _, k := range keys {
		key := operationKey(k)
		call, ok := calls[key]
		if pendingOperations[key] {
			if ok {
				t.Errorf("%s has an SDK method; remove it from pendingOperations", key)
			}
			continue
		}
		if !ok {
			t.Errorf("%s has no SDK method", key)
			continue
		}
		t.Run(k, func(t *testing.T) {
			op := operations[key]
			client, fake := newTestClient(t, func(recorded, int) *http.Response { return responseFor(key) })
			if err := call(context.Background(), client); err != nil {
				t.Fatal(err)
			}
			requests := fake.all()
			if len(requests) != 1 {
				t.Fatalf("%d requests", len(requests))
			}
			r := requests[0]
			path := op.Path
			for _, name := range op.PathParams {
				path = strings.Replace(path, "{"+name+"}", encodeURIComponent(testIDs[name]), 1)
			}
			if r.Method != op.Method || r.Path() != path {
				t.Fatalf("%s %s, want %s %s", r.Method, r.Path(), op.Method, path)
			}
			if op.Auth == authSending {
				if r.Header.Get("x-lettermint-token") != sendingFixture || r.Header.Get("Authorization") != "" {
					t.Fatal(r.Header)
				}
			} else if r.Header.Get("Authorization") != "Bearer "+teamFixture || r.Header.Get("x-lettermint-token") != "" {
				t.Fatal(r.Header)
			}
			if (op.Request != nil) != (len(r.RawBody) > 0) {
				t.Fatalf("request body %q for %v", r.RawBody, op.Request)
			}
		})
	}
}

func TestRequestBodies(t *testing.T) {
	client, fake := newTestClient(t, func(recorded, int) *http.Response { return jsonResponse(200, map[string]any{}) })
	ctx := context.Background()
	if _, err := client.Webhooks.Update(ctx, "w", UpdateWebhookData{BasicAuth: Null[WebhookBasicAuthData](), ProjectIDs: []string{}, Enabled: Ptr(false)}); err != nil {
		t.Fatal(err)
	}
	if _, err := client.Webhooks.Update(ctx, "w", UpdateWebhookData{BasicAuth: Value(WebhookBasicAuthData{Username: "u", Password: "p"})}); err != nil {
		t.Fatal(err)
	}
	if _, err := client.Webhooks.Update(ctx, "w", UpdateWebhookData{Name: Ptr("Renamed")}); err != nil {
		t.Fatal(err)
	}
	if _, err := client.Messages.Process(ctx, "m", WithIdempotencyKey("process-1")); err != nil {
		t.Fatal(err)
	}
	requests := fake.all()
	for i, want := range []string{
		`{"enabled":false,"project_ids":[],"basic_auth":null}`,
		`{"basic_auth":{"username":"u","password":"p"}}`,
		`{"name":"Renamed"}`,
	} {
		if string(requests[i].RawBody) != want {
			t.Errorf("%d: %s, want %s", i, requests[i].RawBody, want)
		}
	}
	if requests[3].Header.Get("Idempotency-Key") != "process-1" || len(requests[3].RawBody) != 0 {
		t.Fatal(requests[3].Header)
	}
}

func TestMessagesResend(t *testing.T) {
	client, fake := newTestClient(t, func(recorded, int) *http.Response {
		return jsonResponse(202, map[string]any{"message_id": "msg_new", "status": "pending"})
	})
	ctx := context.Background()
	resp, err := client.Messages.Resend(ctx, "msg/1", ResendMessageRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if resp.MessageID != "msg_new" || resp.Status != "pending" {
		t.Fatalf("%+v", resp)
	}
	if _, err := client.Messages.Resend(ctx, "msg/1", ResendMessageRequest{Recipients: []string{"jane@example.test", "john@example.test"}}, WithIdempotencyKey("resend-1")); err != nil {
		t.Fatal(err)
	}
	requests := fake.all()
	for _, r := range requests {
		if r.Method != "POST" || r.URL.String() != "https://api.lettermint.co/v1/messages/msg%2F1/resend" {
			t.Fatal(r.Method, r.URL)
		}
		if r.Header.Get("Authorization") != "Bearer "+teamFixture || r.Header.Get("x-lettermint-token") != "" || r.Header.Get("Content-Type") != "application/json" {
			t.Fatal(r.Header)
		}
	}
	if string(requests[0].RawBody) != `{}` || requests[0].Header.Get("Idempotency-Key") != "" {
		t.Fatal(string(requests[0].RawBody), requests[0].Header)
	}
	if string(requests[1].RawBody) != `{"recipients":["jane@example.test","john@example.test"]}` || requests[1].Header.Get("Idempotency-Key") != "resend-1" {
		t.Fatal(string(requests[1].RawBody), requests[1].Header)
	}

	// Without a team token it uses the sending token.
	sendingFake := &fakeTransport{handle: func(recorded, int) *http.Response {
		return jsonResponse(202, map[string]any{"message_id": "msg_new", "status": "pending"})
	}}
	sendingOnly, _ := New(WithSendingToken(sendingFixture), WithHTTPClient(&http.Client{Transport: sendingFake}))
	if _, err := sendingOnly.Messages.Resend(ctx, "m", ResendMessageRequest{}); err != nil {
		t.Fatal(err)
	}
	if r := sendingFake.all()[0]; r.Header.Get("x-lettermint-token") != sendingFixture || r.Header.Get("Authorization") != "" {
		t.Fatal(r.Header)
	}

	// An empty ID is rejected before a request.
	_, err = client.Messages.Resend(ctx, "", ResendMessageRequest{})
	if !strings.Contains(mustAs[*ConfigError](t, err).Message, "Messages.Resend: messageId must be a non-empty string") {
		t.Fatal(err)
	}
	if len(fake.all()) != 2 {
		t.Fatal(len(fake.all()))
	}
}

func TestMessagesResendGoneIsAnAPIError(t *testing.T) {
	client, _ := newTestClient(t, func(recorded, int) *http.Response {
		return jsonResponse(410, map[string]any{"error": map[string]any{"code": "MESSAGE_CONTENT_UNAVAILABLE", "message": "The content of this message is no longer retained."}})
	})
	_, err := client.Messages.Resend(context.Background(), "m", ResendMessageRequest{})
	apiErr := mustAs[*APIError](t, err)
	if apiErr.Status != 410 || apiErr.Code != "MESSAGE_CONTENT_UNAVAILABLE" || apiErr.Message != "The content of this message is no longer retained." {
		t.Fatalf("%+v", apiErr)
	}
	if _, ok := err.(*APIError); !ok {
		t.Fatalf("%T, want *APIError", err)
	}
}

func TestWebhookDeliveriesRetry(t *testing.T) {
	client, fake := newTestClient(t, func(recorded, int) *http.Response {
		return jsonResponse(202, map[string]any{"message": "The delivery is being retried.", "delivery_id": "delivery/1"})
	})
	ctx := context.Background()
	resp, err := client.Webhooks.Deliveries.Retry(ctx, "hook/1", "delivery/1")
	if err != nil {
		t.Fatal(err)
	}
	if resp.Message != "The delivery is being retried." || resp.DeliveryID != "delivery/1" {
		t.Fatalf("%+v", resp)
	}
	r := fake.all()[0]
	if r.Method != "POST" || r.URL.String() != "https://api.lettermint.co/v1/webhooks/hook%2F1/deliveries/delivery%2F1/retry" {
		t.Fatal(r.Method, r.URL)
	}
	if r.Header.Get("Authorization") != "Bearer "+teamFixture || r.Header.Get("x-lettermint-token") != "" {
		t.Fatal(r.Header)
	}
	if len(r.RawBody) != 0 || r.Header.Get("Content-Type") != "" || r.Header.Get("Idempotency-Key") != "" {
		t.Fatal(string(r.RawBody), r.Header)
	}

	// Empty IDs are rejected before a request.
	_, err = client.Webhooks.Deliveries.Retry(ctx, "", "d")
	if !strings.Contains(mustAs[*ConfigError](t, err).Message, "Webhooks.Deliveries.Retry: webhookId must be a non-empty string") {
		t.Fatal(err)
	}
	_, err = client.Webhooks.Deliveries.Retry(ctx, "w", "")
	if !strings.Contains(mustAs[*ConfigError](t, err).Message, "Webhooks.Deliveries.Retry: deliveryId must be a non-empty string") {
		t.Fatal(err)
	}
	if len(fake.all()) != 1 {
		t.Fatal(len(fake.all()))
	}
}

func TestWebhookDeliveriesRetryConflict(t *testing.T) {
	client, _ := newTestClient(t, func(recorded, int) *http.Response {
		return jsonResponse(409, map[string]any{"error": map[string]any{"code": "WEBHOOK_DELIVERY_NOT_RETRYABLE", "message": "This delivery cannot be retried."}})
	})
	_, err := client.Webhooks.Deliveries.Retry(context.Background(), "w", "d")
	mustAs[*ConflictError](t, err)
	apiErr := mustAs[*APIError](t, err)
	if apiErr.Status != 409 || apiErr.Code != "WEBHOOK_DELIVERY_NOT_RETRYABLE" || apiErr.Message != "This delivery cannot be retried." {
		t.Fatalf("%+v", apiErr)
	}
}

func TestWebhookDeliveryNextRetryAt(t *testing.T) {
	bodies := []map[string]any{
		{"id": "d1", "next_retry_at": "2026-10-18T09:00:00Z"},
		{"id": "d2", "next_retry_at": nil},
		{"id": "d3"},
	}
	client, _ := newTestClient(t, func(_ recorded, i int) *http.Response { return jsonResponse(200, bodies[i]) })
	ctx := context.Background()
	scheduled, err := client.Webhooks.Deliveries.Retrieve(ctx, "w", "d1")
	if err != nil {
		t.Fatal(err)
	}
	if at, ok := scheduled.NextRetryAt.Get(); !ok || at != "2026-10-18T09:00:00Z" {
		t.Fatalf("%+v", scheduled.NextRetryAt)
	}
	none, err := client.Webhooks.Deliveries.Retrieve(ctx, "w", "d2")
	if err != nil {
		t.Fatal(err)
	}
	if !none.NextRetryAt.IsNull() {
		t.Fatalf("%+v", none.NextRetryAt)
	}
	absent, err := client.Webhooks.Deliveries.Retrieve(ctx, "w", "d3")
	if err != nil {
		t.Fatal(err)
	}
	if absent.NextRetryAt.IsSet() {
		t.Fatalf("%+v", absent.NextRetryAt)
	}
}

func TestTeamInvites(t *testing.T) {
	invite := map[string]any{
		"id":    "invite/1",
		"email": "jane@example.test",
		"role":  map[string]any{"id": "role_1", "name": "Developer"},
		"project_access": map[string]any{
			"scope":    "selected",
			"projects": []any{map[string]any{"id": "project_1", "name": "Production"}},
		},
		"created_at": "2026-10-20T09:00:00Z",
	}
	client, fake := newTestClient(t, func(r recorded, _ int) *http.Response {
		switch {
		case r.Method == "GET" && r.Path() == "/team/invites":
			return jsonResponse(200, map[string]any{"data": []any{invite}, "next_cursor": nil})
		case r.Method == "POST":
			return jsonResponse(201, invite)
		case r.Method == "DELETE":
			return jsonResponse(200, map[string]any{"message": "The invite has been revoked."})
		}
		return jsonResponse(200, invite)
	})
	ctx := context.Background()

	created, err := client.Team.Invites.Create(ctx, CreateTeamInviteData{
		Email:         "jane@example.test",
		RoleID:        "role_1",
		ProjectAccess: CreateTeamInviteDataProjectAccess{Scope: ProjectAccessScopeSelected, ProjectIDs: []string{"project_1"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if created.ID != "invite/1" || created.Email != "jane@example.test" || created.Role.ID != "role_1" || created.Role.Name != "Developer" || created.CreatedAt != "2026-10-20T09:00:00Z" {
		t.Fatalf("%+v", created)
	}
	if created.ProjectAccess.Scope != ProjectAccessScopeSelected || len(created.ProjectAccess.Projects) != 1 || created.ProjectAccess.Projects[0].Name != "Production" {
		t.Fatalf("%+v", created.ProjectAccess)
	}
	page, err := client.Team.Invites.List(ctx, &ListTeamInvitesQuery{PageSize: 10, PageCursor: "abc"})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Data) != 1 || page.Data[0].Email != "jane@example.test" || page.NextCursor != nil {
		t.Fatalf("%+v", page)
	}
	retrieved, err := client.Team.Invites.Retrieve(ctx, "invite/1")
	if err != nil {
		t.Fatal(err)
	}
	if retrieved.ID != "invite/1" {
		t.Fatalf("%+v", retrieved)
	}
	updated, err := client.Team.Invites.Update(ctx, "invite/1", UpdateTeamInviteData{
		Email:         "john@example.test",
		RoleID:        "role_2",
		ProjectAccess: UpdateTeamInviteDataProjectAccess{Scope: ProjectAccessScopeAll},
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Role.Name != "Developer" {
		t.Fatalf("%+v", updated)
	}
	revoked, err := client.Team.Invites.Revoke(ctx, "invite/1")
	if err != nil {
		t.Fatal(err)
	}
	if revoked.Message != "The invite has been revoked." {
		t.Fatalf("%+v", revoked)
	}

	requests := fake.all()
	want := []struct{ method, url, body string }{
		{"POST", "https://api.lettermint.co/v1/team/invites", `{"email":"jane@example.test","role_id":"role_1","project_access":{"scope":"selected","project_ids":["project_1"]}}`},
		{"GET", "https://api.lettermint.co/v1/team/invites?page%5Bsize%5D=10&page%5Bcursor%5D=abc", ""},
		{"GET", "https://api.lettermint.co/v1/team/invites/invite%2F1", ""},
		{"PUT", "https://api.lettermint.co/v1/team/invites/invite%2F1", `{"email":"john@example.test","role_id":"role_2","project_access":{"scope":"all"}}`},
		{"DELETE", "https://api.lettermint.co/v1/team/invites/invite%2F1", ""},
	}
	if len(requests) != len(want) {
		t.Fatal(len(requests))
	}
	for i, w := range want {
		r := requests[i]
		if r.Method != w.method || r.URL.String() != w.url || string(r.RawBody) != w.body {
			t.Errorf("%d: %s %s %s, want %s %s %s", i, r.Method, r.URL, r.RawBody, w.method, w.url, w.body)
		}
		if r.Header.Get("Authorization") != "Bearer "+teamFixture || r.Header.Get("x-lettermint-token") != "" || r.Header.Get("Idempotency-Key") != "" {
			t.Errorf("%d: %v", i, r.Header)
		}
	}

	// Empty IDs are rejected before a request.
	_, err = client.Team.Invites.Retrieve(ctx, "")
	if !strings.Contains(mustAs[*ConfigError](t, err).Message, "Team.Invites.Retrieve: inviteId must be a non-empty string") {
		t.Fatal(err)
	}
	_, err = client.Team.Invites.Update(ctx, "", UpdateTeamInviteData{})
	if !strings.Contains(mustAs[*ConfigError](t, err).Message, "Team.Invites.Update: inviteId must be a non-empty string") {
		t.Fatal(err)
	}
	_, err = client.Team.Invites.Revoke(ctx, "")
	if !strings.Contains(mustAs[*ConfigError](t, err).Message, "Team.Invites.Revoke: inviteId must be a non-empty string") {
		t.Fatal(err)
	}
	if len(fake.all()) != len(want) {
		t.Fatal(len(fake.all()))
	}

	// Without a team token, no request is sent.
	sendingOnly, _ := New(WithSendingToken(sendingFixture), WithHTTPClient(&http.Client{Transport: &fakeTransport{handle: func(recorded, int) *http.Response { return nil }}}))
	_, err = sendingOnly.Team.Invites.List(ctx, nil)
	if !strings.Contains(mustAs[*ConfigError](t, err).Message, "Team.Invites.List needs a team token") {
		t.Fatal(err)
	}
}

func TestTeamInvitesErrors(t *testing.T) {
	responses := []*http.Response{
		jsonResponse(409, map[string]any{"error": map[string]any{"code": "stale_resource", "message": "The team has no free seat."}}),
		jsonResponse(422, map[string]any{"message": "The email field must be a valid email address.", "errors": map[string]any{"email": []any{"The email field must be a valid email address."}}}),
	}
	client, _ := newTestClient(t, func(_ recorded, i int) *http.Response { return responses[i] })
	ctx := context.Background()
	body := CreateTeamInviteData{Email: "jane@example.test", RoleID: "role_1", ProjectAccess: CreateTeamInviteDataProjectAccess{Scope: ProjectAccessScopeAll}}
	_, err := client.Team.Invites.Create(ctx, body)
	conflict := mustAs[*ConflictError](t, err)
	if conflict.Status != 409 || conflict.Code != "stale_resource" || conflict.Message != "The team has no free seat." {
		t.Fatalf("%+v", conflict)
	}
	_, err = client.Team.Invites.Update(ctx, "invite_1", UpdateTeamInviteData{Email: "not-an-email", RoleID: "role_1", ProjectAccess: UpdateTeamInviteDataProjectAccess{Scope: ProjectAccessScopeAll}})
	validation := mustAs[*ValidationError](t, err)
	if validation.Status != 422 || len(validation.Errors["email"]) != 1 {
		t.Fatalf("%+v", validation)
	}
}

func TestTeamRoles(t *testing.T) {
	role := func(description any) map[string]any {
		body := map[string]any{"id": "role/1", "name": "Support", "system_key": nil, "permissions": []any{"members:read", "messages:read"}, "assignable": true}
		if description != "absent" {
			body["description"] = description
		}
		return body
	}
	client, fake := newTestClient(t, func(r recorded, i int) *http.Response {
		switch {
		case r.Method == "DELETE":
			return jsonResponse(200, map[string]any{"message": "The role has been deleted."})
		case r.Method == "POST":
			return jsonResponse(201, role("Handles support tickets"))
		case i == 0:
			return jsonResponse(200, role("Handles support tickets"))
		case i == 1:
			return jsonResponse(200, role(nil))
		}
		return jsonResponse(200, role("absent"))
	})
	ctx := context.Background()

	described, err := client.Team.RetrieveRole(ctx, "role/1")
	if err != nil {
		t.Fatal(err)
	}
	if description, ok := described.Description.Get(); !ok || description != "Handles support tickets" {
		t.Fatalf("%+v", described.Description)
	}
	if described.ID != "role/1" || described.Name != "Support" || described.SystemKey != nil || !described.Assignable ||
		!reflect.DeepEqual(described.Permissions, []RbacPermission{RbacPermissionMembersRead, RbacPermissionMessagesRead}) {
		t.Fatalf("%+v", described)
	}
	undescribed, err := client.Team.RetrieveRole(ctx, "role/1")
	if err != nil {
		t.Fatal(err)
	}
	if !undescribed.Description.IsNull() {
		t.Fatalf("%+v", undescribed.Description)
	}
	absent, err := client.Team.UpdateRole(ctx, "role/1", UpdateTeamRoleData{Name: "Support", Description: Null[string](), Permissions: []RbacPermission{RbacPermissionMembersRead}})
	if err != nil {
		t.Fatal(err)
	}
	if absent.Description.IsSet() {
		t.Fatalf("%+v", absent.Description)
	}
	created, err := client.Team.CreateRole(ctx, CreateTeamRoleData{Name: "Support", Description: Value("Handles support tickets"), Permissions: []RbacPermission{RbacPermissionMembersRead, RbacPermissionMessagesRead}})
	if err != nil {
		t.Fatal(err)
	}
	if created.ID != "role/1" {
		t.Fatalf("%+v", created)
	}
	if _, err := client.Team.CreateRole(ctx, CreateTeamRoleData{Name: "Viewer", Permissions: []RbacPermission{}}); err != nil {
		t.Fatal(err)
	}
	duplicate, err := client.Team.DuplicateRole(ctx, "role/1", DuplicateTeamRoleData{Name: "Support copy"})
	if err != nil {
		t.Fatal(err)
	}
	if duplicate.Name != "Support" {
		t.Fatalf("%+v", duplicate)
	}
	if _, err := client.Team.DuplicateRole(ctx, "role/1", DuplicateTeamRoleData{Name: "Support copy", Description: Value("Copy")}); err != nil {
		t.Fatal(err)
	}
	deleted, err := client.Team.DeleteRole(ctx, "role/1", nil)
	if err != nil {
		t.Fatal(err)
	}
	if deleted.Message != "The role has been deleted." {
		t.Fatalf("%+v", deleted)
	}
	if _, err := client.Team.DeleteRole(ctx, "role/1", &DeleteTeamRoleQuery{ReplacementRoleID: "role/2"}); err != nil {
		t.Fatal(err)
	}
	if _, err := client.Team.DeleteRole(ctx, "role/1", &DeleteTeamRoleQuery{}); err != nil {
		t.Fatal(err)
	}

	requests := fake.all()
	want := []struct{ method, url, body string }{
		{"GET", "https://api.lettermint.co/v1/team/roles/role%2F1", ""},
		{"GET", "https://api.lettermint.co/v1/team/roles/role%2F1", ""},
		{"PUT", "https://api.lettermint.co/v1/team/roles/role%2F1", `{"name":"Support","description":null,"permissions":["members:read"]}`},
		{"POST", "https://api.lettermint.co/v1/team/roles", `{"name":"Support","description":"Handles support tickets","permissions":["members:read","messages:read"]}`},
		{"POST", "https://api.lettermint.co/v1/team/roles", `{"name":"Viewer","permissions":[]}`},
		{"POST", "https://api.lettermint.co/v1/team/roles/role%2F1/duplicate", `{"name":"Support copy"}`},
		{"POST", "https://api.lettermint.co/v1/team/roles/role%2F1/duplicate", `{"name":"Support copy","description":"Copy"}`},
		{"DELETE", "https://api.lettermint.co/v1/team/roles/role%2F1", ""},
		{"DELETE", "https://api.lettermint.co/v1/team/roles/role%2F1?replacement_role_id=role%2F2", ""},
		{"DELETE", "https://api.lettermint.co/v1/team/roles/role%2F1", ""},
	}
	if len(requests) != len(want) {
		t.Fatal(len(requests))
	}
	for i, w := range want {
		r := requests[i]
		if r.Method != w.method || r.URL.String() != w.url || string(r.RawBody) != w.body {
			t.Errorf("%d: %s %s %s, want %s %s %s", i, r.Method, r.URL, r.RawBody, w.method, w.url, w.body)
		}
		if r.Header.Get("Authorization") != "Bearer "+teamFixture || r.Header.Get("x-lettermint-token") != "" || r.Header.Get("Idempotency-Key") != "" {
			t.Errorf("%d: %v", i, r.Header)
		}
	}

	// Empty IDs are rejected before a request.
	for label, call := range map[string]func() error{
		"Team.RetrieveRole":  func() error { _, err := client.Team.RetrieveRole(ctx, ""); return err },
		"Team.UpdateRole":    func() error { _, err := client.Team.UpdateRole(ctx, "", UpdateTeamRoleData{}); return err },
		"Team.DeleteRole":    func() error { _, err := client.Team.DeleteRole(ctx, "", nil); return err },
		"Team.DuplicateRole": func() error { _, err := client.Team.DuplicateRole(ctx, "", DuplicateTeamRoleData{}); return err },
	} {
		if err := call(); !strings.Contains(mustAs[*ConfigError](t, err).Message, label+": roleId must be a non-empty string") {
			t.Fatal(err)
		}
	}
	if len(fake.all()) != len(want) {
		t.Fatal(len(fake.all()))
	}
}

func TestTeamRolesConflict(t *testing.T) {
	client, _ := newTestClient(t, func(recorded, int) *http.Response {
		return jsonResponse(409, map[string]any{"error": map[string]any{"code": "built_in_role_immutable", "message": "Built-in roles cannot be changed."}})
	})
	_, err := client.Team.DeleteRole(context.Background(), "role_1", nil)
	conflict := mustAs[*ConflictError](t, err)
	if conflict.Status != 409 || conflict.Code != string(RbacConflictCodeBuiltInRoleImmutable) || conflict.Message != "Built-in roles cannot be changed." {
		t.Fatalf("%+v", conflict)
	}
}

func query(r recorded) map[string]string {
	result := map[string]string{}
	for key, values := range r.URL.Query() {
		result[key] = strings.Join(values, "|")
	}
	return result
}

func TestQueryParameters(t *testing.T) {
	client, fake := newTestClient(t, func(recorded, int) *http.Response {
		return jsonResponse(200, map[string]any{"data": []any{}, "next_cursor": nil})
	})
	ctx := context.Background()
	if _, err := client.Domains.List(ctx, &ListDomainsQuery{PageSize: 10, PageCursor: "abc", FilterStatus: DomainStatusVerified, FilterDomain: "acme", Sort: []ListDomainsQuerySortItem{ListDomainsQuerySortItemCreatedAtDesc, ListDomainsQuerySortItemDomain}}); err != nil {
		t.Fatal(err)
	}
	if _, err := client.Messages.List(ctx, &ListMessagesQuery{Filter: &ListMessagesQueryFilter{Tags: []ListMessagesQueryFilterTagsItem{{Name: "campaign", Value: "welcome"}, {Name: "tier", Value: "gold"}}}, FilterSearch: "hello world"}); err != nil {
		t.Fatal(err)
	}
	if _, err := client.Webhooks.List(ctx, &ListWebhooksQuery{FilterEnabled: Ptr(false), Cursor: "c1", PageSize: 5}); err != nil {
		t.Fatal(err)
	}
	if _, err := client.Messages.Events(ctx, "m", &ListMessageEventsQuery{IncludeMachineEvents: Ptr(true), PageSize: 2}); err != nil {
		t.Fatal(err)
	}
	if _, err := client.Domains.Retrieve(ctx, "d", &GetDomainQuery{Include: []GetDomainQueryIncludeItem{GetDomainQueryIncludeItemDNSRecords, GetDomainQueryIncludeItemProjects}}); err != nil {
		t.Fatal(err)
	}
	if _, err := client.Stats.Retrieve(ctx, GetStatsQuery{From: "2026-01-01", To: "2026-01-31"}); err != nil {
		t.Fatal(err)
	}
	requests := fake.all()
	want := []map[string]string{
		{"page[size]": "10", "page[cursor]": "abc", "sort": "-created_at,domain", "filter[status]": "verified", "filter[domain]": "acme"},
		{"filter[tags][0][name]": "campaign", "filter[tags][0][value]": "welcome", "filter[tags][1][name]": "tier", "filter[tags][1][value]": "gold", "filter[search]": "hello world"},
		{"filter[enabled]": "0", "cursor": "c1", "page[size]": "5"},
		{"include_machine_events": "1", "page[size]": "2"},
		{"include": "dnsRecords,projects"},
		{"from": "2026-01-01", "to": "2026-01-31"},
	}
	for i, w := range want {
		if got := query(requests[i]); !reflect.DeepEqual(got, w) {
			t.Errorf("%d: %v, want %v", i, got, w)
		}
	}
	if !strings.Contains(requests[1].URL.RawQuery, "filter%5Bsearch%5D=hello+world") {
		t.Fatal(requests[1].URL.RawQuery)
	}
	if requests[0].URL.RawQuery != "page%5Bsize%5D=10&page%5Bcursor%5D=abc&sort=-created_at%2Cdomain&filter%5Bstatus%5D=verified&filter%5Bdomain%5D=acme" {
		t.Fatal(requests[0].URL.RawQuery)
	}
}

func TestPaginationFollowsNextCursor(t *testing.T) {
	pages := map[string]any{
		"":   map[string]any{"data": []any{map[string]any{"id": "d1"}, map[string]any{"id": "d2"}}, "next_cursor": "c2"},
		"c2": map[string]any{"data": []any{map[string]any{"id": "d3"}}, "next_cursor": "c3"},
		"c3": map[string]any{"data": []any{}, "next_cursor": nil},
	}
	client, fake := newTestClient(t, func(r recorded, _ int) *http.Response {
		return jsonResponse(200, pages[r.URL.Query().Get("page[cursor]")])
	})
	var ids []string
	for domain, err := range client.Domains.Iterate(context.Background(), &ListDomainsQuery{PageSize: 2, FilterStatus: DomainStatusVerified}) {
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, domain.ID)
	}
	if strings.Join(ids, ",") != "d1,d2,d3" {
		t.Fatal(ids)
	}
	var queries []map[string]string
	for _, r := range fake.all() {
		queries = append(queries, query(r))
	}
	want := []map[string]string{
		{"page[size]": "2", "filter[status]": "verified"},
		{"page[size]": "2", "filter[status]": "verified", "page[cursor]": "c2"},
		{"page[size]": "2", "filter[status]": "verified", "page[cursor]": "c3"},
	}
	if !reflect.DeepEqual(queries, want) {
		t.Fatal(queries)
	}
}

func TestPaginationUsesTheTablesCursorParameter(t *testing.T) {
	client, fake := newTestClient(t, func(r recorded, _ int) *http.Response {
		if r.URL.Query().Get("cursor") != "" {
			return jsonResponse(200, map[string]any{"data": []any{map[string]any{"id": "w2"}}, "next_cursor": nil})
		}
		return jsonResponse(200, map[string]any{"data": []any{map[string]any{"id": "w1"}}, "next_cursor": "next"})
	})
	var ids []string
	for delivery, err := range client.Webhooks.Deliveries.Iterate(context.Background(), "hook/1", nil) {
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, delivery.ID)
	}
	if strings.Join(ids, ",") != "w1,w2" || fake.all()[1].URL.String() != "https://api.lettermint.co/v1/webhooks/hook%2F1/deliveries?cursor=next" {
		t.Fatal(ids, fake.all()[1].URL)
	}
}

func TestEveryListIterates(t *testing.T) {
	ctx := context.Background()
	iterators := map[string]func(*Client) func(func(any, error) bool){
		"domains":        func(c *Client) func(func(any, error) bool) { return anySeq(c.Domains.Iterate(ctx, nil)) },
		"messages":       func(c *Client) func(func(any, error) bool) { return anySeq(c.Messages.Iterate(ctx, nil)) },
		"message events": func(c *Client) func(func(any, error) bool) { return anySeq(c.Messages.IterateEvents(ctx, "m", nil)) },
		"projects":       func(c *Client) func(func(any, error) bool) { return anySeq(c.Projects.Iterate(ctx, nil)) },
		"routes":         func(c *Client) func(func(any, error) bool) { return anySeq(c.Routes.Iterate(ctx, "p", nil)) },
		"suppressions":   func(c *Client) func(func(any, error) bool) { return anySeq(c.Suppressions.Iterate(ctx, nil)) },
		"team members":   func(c *Client) func(func(any, error) bool) { return anySeq(c.Team.Members.Iterate(ctx, nil)) },
		"team invites":   func(c *Client) func(func(any, error) bool) { return anySeq(c.Team.Invites.Iterate(ctx, nil)) },
		"webhooks":       func(c *Client) func(func(any, error) bool) { return anySeq(c.Webhooks.Iterate(ctx, nil)) },
		"webhook deliveries": func(c *Client) func(func(any, error) bool) {
			return anySeq(c.Webhooks.Deliveries.Iterate(ctx, "w", nil))
		},
	}
	for label, iterator := range iterators {
		client, fake := newTestClient(t, func(r recorded, i int) *http.Response {
			if i == 0 {
				return jsonResponse(200, map[string]any{"data": []any{map[string]any{"id": "1"}}, "next_cursor": "two"})
			}
			return jsonResponse(200, map[string]any{"data": []any{map[string]any{"id": "2"}}, "next_cursor": nil})
		})
		count := 0
		for _, err := range iterator(client) {
			if err != nil {
				t.Fatal(label, err)
			}
			count++
		}
		requests := fake.all()
		cursor := requests[1].URL.Query().Get("page[cursor]") + requests[1].URL.Query().Get("cursor")
		if count != 2 || len(requests) != 2 || cursor != "two" {
			t.Errorf("%s: %d items, %d requests, cursor %q", label, count, len(requests), cursor)
		}
	}
}

func anySeq[T any](seq func(func(T, error) bool)) func(func(any, error) bool) {
	return func(yield func(any, error) bool) {
		seq(func(v T, err error) bool { return yield(v, err) })
	}
}

func TestPaginationStops(t *testing.T) {
	// On a repeated cursor.
	client, fake := newTestClient(t, func(recorded, int) *http.Response {
		return jsonResponse(200, map[string]any{"data": []any{map[string]any{"id": "x"}}, "next_cursor": "same"})
	})
	count := 0
	for _, err := range client.Domains.Iterate(context.Background(), nil) {
		if err != nil {
			t.Fatal(err)
		}
		count++
	}
	if count != 2 || len(fake.all()) != 2 {
		t.Fatal(count, len(fake.all()))
	}
	// When the caller breaks, without requesting the next page.
	client, fake = newTestClient(t, func(recorded, int) *http.Response {
		return jsonResponse(200, map[string]any{"data": []any{map[string]any{"id": "1"}, map[string]any{"id": "2"}}, "next_cursor": "more"})
	})
	for range client.Domains.Iterate(context.Background(), nil) {
		break
	}
	if len(fake.all()) != 1 {
		t.Fatal(len(fake.all()))
	}
	// At the first error, which is yielded once.
	client, _ = newTestClient(t, func(recorded, int) *http.Response { return jsonResponse(500, map[string]any{}) })
	errorsSeen := 0
	for _, err := range client.Domains.Iterate(context.Background(), nil) {
		mustAs[*ServerError](t, err)
		errorsSeen++
	}
	// On a page without a data array.
	client, _ = newTestClient(t, func(recorded, int) *http.Response { return jsonResponse(200, map[string]any{"next_cursor": nil}) })
	for _, err := range client.Domains.Iterate(context.Background(), nil) {
		mustAs[*UnexpectedResponseError](t, err)
		errorsSeen++
	}
	if errorsSeen != 2 {
		t.Fatal(errorsSeen)
	}
}

func TestPathParameters(t *testing.T) {
	client, fake := newTestClient(t, nil)
	if _, err := client.Domains.Retrieve(context.Background(), "a/b?c#d e%", nil); err != nil {
		t.Fatal(err)
	}
	if got := fake.all()[0].URL.EscapedPath(); got != "/v1/domains/a%2Fb%3Fc%23d%20e%25" {
		t.Fatal(got)
	}
	unescaped, _ := url.PathUnescape(strings.TrimPrefix(fake.all()[0].URL.EscapedPath(), "/v1/domains/"))
	if unescaped != "a/b?c#d e%" {
		t.Fatal(unescaped)
	}
	for _, value := range []string{"", ".", ".."} {
		_, err := client.Domains.Retrieve(context.Background(), value, nil)
		if !strings.Contains(mustAs[*ConfigError](t, err).Message, "Domains.Retrieve: domainId must be a non-empty string") {
			t.Fatal(err)
		}
		_, err = client.Webhooks.Deliveries.Retrieve(context.Background(), "w", value)
		mustAs[*ConfigError](t, err)
	}
	if len(fake.all()) != 1 {
		t.Fatal("rejected IDs send nothing")
	}
	if encodeURIComponent("AZaz09-_.!~*'()é/ ") != "AZaz09-_.!~*'()%C3%A9%2F%20" {
		t.Fatal(encodeURIComponent("AZaz09-_.!~*'()é/ "))
	}
}
