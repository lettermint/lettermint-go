package lettermint

import (
	"context"
	"net/http"
	"os"
	"reflect"
	"testing"
	"time"
)

const (
	analyticsQueryJSON       = `{"metrics":["delivered","bounced","delivery_rate","delivery_latency_p50_ms"],"timezone":"Asia/Kolkata","include":["summary","time_series","breakdown"],"group_by":["recipient_domain"],"interval":"hour","compare":"previous_period","limit":2}`
	analyticsQueryCursorJSON = `{"metrics":["delivered","bounced","delivery_rate","delivery_latency_p50_ms"],"timezone":"Asia/Kolkata","include":["summary","time_series","breakdown"],"group_by":["recipient_domain"],"interval":"hour","compare":"previous_period","limit":2,"cursor":"cursor-page-2"}`
)

func analyticsQuery() AnalyticsQuery {
	return AnalyticsQuery{
		Metrics:  []AnalyticsMetric{AnalyticsMetricDelivered, AnalyticsMetricBounced, AnalyticsMetricDeliveryRate, AnalyticsMetricDeliveryLatencyP50Ms},
		Include:  []AnalyticsSection{AnalyticsSectionSummary, AnalyticsSectionTimeSeries, AnalyticsSectionBreakdown},
		GroupBy:  []AnalyticsGroupDimension{AnalyticsCatalogueGroupDimensionRecipientDomain},
		Interval: Ptr(AnalyticsIntervalHour),
		Timezone: Ptr("Asia/Kolkata"),
		Compare:  Ptr(AnalyticsComparisonPreviousPeriod),
		Limit:    Ptr(2),
	}
}

// analyticsFixture answers with testdata/analytics-<name>.json, a response
// like the API sends it.
func analyticsFixture(t *testing.T, name string) *http.Response {
	t.Helper()
	data, err := os.ReadFile("testdata/analytics-" + name + ".json")
	if err != nil {
		t.Fatal(err)
	}
	return rawResponse(200, string(data), "Content-Type", "application/json")
}

func instant(t *testing.T, value string) time.Time {
	t.Helper()
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		t.Fatal(err)
	}
	return parsed
}

func TestAnalyticsKeepsNullsEmptyRateBasesAndBothTimestampFormats(t *testing.T) {
	client, fake := newTestClient(t, func(recorded, int) *http.Response { return analyticsFixture(t, "page-1") })
	result, err := client.Analytics(context.Background(), analyticsQuery())
	if err != nil {
		t.Fatal(err)
	}
	if body := string(fake.all()[0].RawBody); body != analyticsQueryJSON {
		t.Fatal(body)
	}

	summary := result.Data.Summary
	metrics := AnalyticsMetricValues{Delivered: Value(1200), Bounced: Value(0), DeliveryRate: Value(0.9836), DeliveryLatencyP50Ms: Null[float64]()}
	if summary == nil || summary.Metrics != metrics {
		t.Fatalf("%+v", summary)
	}
	if bounced, ok := summary.Metrics.Bounced.Get(); !ok || bounced != 0 || summary.Metrics.Bounced.IsNull() {
		t.Fatal("a measured zero is a value")
	}
	if !summary.Metrics.DeliveryLatencyP50Ms.IsNull() || summary.Metrics.Accepted.IsSet() {
		t.Fatalf("%+v", summary.Metrics)
	}
	if base := summary.RateBases.DeliveryRate; base == nil || !reflect.DeepEqual(*base, AnalyticsRateBase{Numerator: Ptr(1200), Denominator: Ptr(1220)}) {
		t.Fatal(base)
	}
	if base := summary.Previous.RateBases.DeliveryRate; base == nil || base.Numerator != nil || base.Denominator != nil {
		t.Fatal(base)
	}
	if change := summary.Change["delivery_rate"]; !change.Absolute.IsNull() || !change.Relative.IsNull() || !change.PercentagePoints.IsNull() {
		t.Fatalf("%+v", change)
	}
	if change := summary.Change["delivered"]; change.Absolute != Value(100.0) || change.Relative != Value(0.0909) || change.PercentagePoints.IsSet() {
		t.Fatalf("%+v", change)
	}

	if len(result.Data.TimeSeries) != 2 {
		t.Fatal(len(result.Data.TimeSeries))
	}
	complete, unavailable := result.Data.TimeSeries[0], result.Data.TimeSeries[1]
	if complete.From != "2026-09-15T08:30:00+05:30" || !instant(t, complete.From).Equal(time.Date(2026, 9, 15, 3, 0, 0, 0, time.UTC)) {
		t.Fatal(complete.From)
	}
	if unavailable.Available || !unavailable.Partial || unavailable.RateBases != (AnalyticsRateBases{}) || !unavailable.Metrics.Delivered.IsNull() {
		t.Fatalf("%+v", unavailable)
	}

	meta := result.Meta
	if meta.GeneratedAt != "2026-09-15T04:12:30.482915Z" || !instant(t, meta.GeneratedAt).Equal(time.Date(2026, 9, 15, 4, 12, 30, 482915000, time.UTC)) {
		t.Fatal(meta.GeneratedAt)
	}
	if !instant(t, meta.From).Equal(time.Date(2026, 9, 15, 3, 0, 0, 0, time.UTC)) || meta.LastIngestedAt != nil {
		t.Fatalf("%+v", meta)
	}
	if meta.Comparison == nil || *meta.Comparison != (AnalyticsMetaComparison{From: "2026-09-15T01:00:00.000000Z", To: "2026-09-15T03:00:00.000000Z"}) {
		t.Fatal(meta.Comparison)
	}
	if want := (AnalyticsPagination{TotalGroups: 3, ReturnedGroups: 2, NextCursor: Ptr("cursor-page-2")}); !reflect.DeepEqual(result.Pagination, want) {
		t.Fatalf("%+v", result.Pagination)
	}
}

func TestAnalyticsLeavesOutWhatTheQueryDidNotAskFor(t *testing.T) {
	client, _ := newTestClient(t, func(recorded, int) *http.Response { return analyticsFixture(t, "summary") })
	result, err := client.Analytics(context.Background(), AnalyticsQuery{Metrics: []AnalyticsMetric{AnalyticsMetricDelivered}})
	if err != nil {
		t.Fatal(err)
	}
	summary := result.Data.Summary
	if summary == nil || summary.RateBases != (AnalyticsRateBases{}) || summary.Previous != nil || summary.Change != nil {
		t.Fatalf("%+v", summary)
	}
	if result.Data.TimeSeries != nil || result.Data.Breakdown != nil || result.Meta.Comparison != nil || result.Pagination.NextCursor != nil {
		t.Fatalf("%+v", result)
	}
}

func TestAnalyticsKeepsANullDimensionValue(t *testing.T) {
	client, fake := newTestClient(t, func(recorded, int) *http.Response { return analyticsFixture(t, "page-2") })
	query := analyticsQuery()
	query.Cursor = Ptr("cursor-page-2")
	result, err := client.Analytics(context.Background(), query)
	if err != nil {
		t.Fatal(err)
	}
	if body := string(fake.all()[0].RawBody); body != analyticsQueryCursorJSON {
		t.Fatal(body)
	}
	if len(result.Data.Breakdown) != 1 {
		t.Fatal(len(result.Data.Breakdown))
	}
	row := result.Data.Breakdown[0]
	if domain, ok := row.Dimensions["recipient_domain"]; !ok || domain != nil || len(row.Dimensions) != 1 || !row.Metrics.Bounced.IsNull() {
		t.Fatalf("%+v", row)
	}
}

func TestAnalyticsPagesFollowsNextCursor(t *testing.T) {
	client, fake := newTestClient(t, func(_ recorded, index int) *http.Response {
		if index == 0 {
			return analyticsFixture(t, "page-1")
		}
		return analyticsFixture(t, "page-2")
	})
	sent := analyticsQuery()
	var pages []*AnalyticsResponse
	for page, err := range client.AnalyticsPages(context.Background(), sent) {
		if err != nil {
			t.Fatal(err)
		}
		pages = append(pages, page)
	}
	requests := fake.all()
	if len(pages) != 2 || len(requests) != 2 {
		t.Fatal(len(pages), len(requests))
	}
	for _, r := range requests {
		if r.Method != "POST" || r.Path() != "/analytics" || r.Header.Get("Authorization") != "Bearer "+teamFixture {
			t.Fatal(r.Method, r.Path())
		}
	}
	if string(requests[0].RawBody) != analyticsQueryJSON || string(requests[1].RawBody) != analyticsQueryCursorJSON {
		t.Fatalf("%s\n%s", requests[0].RawBody, requests[1].RawBody)
	}
	if !reflect.DeepEqual(sent, analyticsQuery()) {
		t.Fatalf("the query was changed: %+v", sent)
	}

	var domains []*string
	for _, page := range pages {
		for _, row := range page.Data.Breakdown {
			domains = append(domains, row.Dimensions["recipient_domain"])
		}
	}
	if !reflect.DeepEqual(domains, []*string{Ptr("gmail.com"), Ptr("outlook.com"), nil}) {
		t.Fatal(domains)
	}
	if last := pages[1].Pagination; last.ReturnedGroups != 1 || last.NextCursor != nil {
		t.Fatalf("%+v", last)
	}
}

func TestAnalyticsPagesStops(t *testing.T) {
	ctx := context.Background()
	count := func(client *Client, query AnalyticsQuery) int {
		pages := 0
		for _, err := range client.AnalyticsPages(ctx, query) {
			if err != nil {
				t.Fatal(err)
			}
			pages++
		}
		return pages
	}
	// After one request for a query without more pages.
	client, fake := newTestClient(t, func(recorded, int) *http.Response { return analyticsFixture(t, "summary") })
	if pages := count(client, AnalyticsQuery{Metrics: []AnalyticsMetric{AnalyticsMetricDelivered}}); pages != 1 || len(fake.all()) != 1 {
		t.Fatal(pages, len(fake.all()))
	}
	// When the caller breaks, without requesting the next page.
	client, fake = newTestClient(t, func(recorded, int) *http.Response { return analyticsFixture(t, "page-1") })
	for page, err := range client.AnalyticsPages(ctx, analyticsQuery()) {
		if err != nil || *page.Pagination.NextCursor != "cursor-page-2" {
			t.Fatal(page, err)
		}
		break
	}
	if len(fake.all()) != 1 {
		t.Fatal(len(fake.all()))
	}
	// When the API repeats a cursor.
	client, fake = newTestClient(t, func(recorded, int) *http.Response { return analyticsFixture(t, "page-1") })
	if pages := count(client, analyticsQuery()); pages != 2 || len(fake.all()) != 2 {
		t.Fatal(pages, len(fake.all()))
	}
	// When the API returns the cursor the query started from.
	client, fake = newTestClient(t, func(recorded, int) *http.Response { return analyticsFixture(t, "page-1") })
	started := analyticsQuery()
	started.Cursor = Ptr("cursor-page-2")
	if pages := count(client, started); pages != 1 || len(fake.all()) != 1 || string(fake.all()[0].RawBody) != analyticsQueryCursorJSON {
		t.Fatal(pages, len(fake.all()))
	}
}

// deadlineTransport records the context of every request before the fake
// transport answers it.
type deadlineTransport struct {
	*fakeTransport
	contexts []context.Context
}

func (d *deadlineTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	d.contexts = append(d.contexts, request.Context())
	return d.fakeTransport.RoundTrip(request)
}

func TestAnalyticsPagesPassesOptionsAndSurfacesAnExpiredCursor(t *testing.T) {
	message := "The analytics cursor is invalid or expired. Submit a new query."
	transport := &deadlineTransport{fakeTransport: &fakeTransport{handle: func(_ recorded, index int) *http.Response {
		if index == 0 {
			return analyticsFixture(t, "page-1")
		}
		return jsonResponse(422, map[string]any{"message": message, "errors": map[string]any{"cursor": []string{message}}})
	}}}
	client, err := New(WithTeamToken(teamFixture), WithHTTPClient(&http.Client{Transport: transport}))
	if err != nil {
		t.Fatal(err)
	}
	type contextKey struct{}
	ctx := context.WithValue(context.Background(), contextKey{}, "caller")
	var pages []*AnalyticsResponse
	var errs []error
	for page, err := range client.AnalyticsPages(ctx, analyticsQuery(), WithRequestTimeout(5*time.Second)) {
		if err != nil {
			errs = append(errs, err)
			continue
		}
		pages = append(pages, page)
	}
	if len(pages) != 1 || len(errs) != 1 || len(transport.all()) != 2 {
		t.Fatal(len(pages), errs, len(transport.all()))
	}
	validation := mustAs[*ValidationError](t, errs[0])
	if !reflect.DeepEqual(validation.Errors, map[string][]string{"cursor": {message}}) {
		t.Fatalf("%+v", validation)
	}
	for _, requestCtx := range transport.contexts {
		deadline, ok := requestCtx.Deadline()
		if left := time.Until(deadline); !ok || left <= 0 || left > 5*time.Second || requestCtx.Value(contextKey{}) != "caller" {
			t.Fatalf("the context and the timeout apply to every page: %v", left)
		}
	}

	client, fake := newTestClient(t, nil)
	for _, err := range client.AnalyticsPages(ctx, analyticsQuery(), WithRequestTimeout(0)) {
		mustAs[*ConfigError](t, err)
	}
	if len(fake.all()) != 0 {
		t.Fatal(len(fake.all()))
	}
}

func TestAnalyticsErrors(t *testing.T) {
	ctx := context.Background()
	send := func(response *http.Response) error {
		client, _ := newTestClient(t, func(recorded, int) *http.Response { return response })
		_, err := client.Analytics(ctx, analyticsQuery())
		return err
	}

	// Field errors from a 422.
	message := "smtp_response_group can only be used in group_by."
	validation := mustAs[*ValidationError](t, send(jsonResponse(422, map[string]any{"message": message, "errors": map[string]any{"filters": []string{message}}})))
	if validation.Status != 422 || validation.Message != message || !reflect.DeepEqual(validation.Errors, map[string][]string{"filters": {message}}) {
		t.Fatalf("%+v", validation)
	}

	// Retry-After from a 503.
	server := mustAs[*ServerError](t, send(jsonResponse(503, map[string]any{"error": map[string]any{"code": "SERVICE_UNAVAILABLE", "message": "Try again shortly."}}, "Retry-After", "2")))
	if server.Status != 503 || server.Code != "SERVICE_UNAVAILABLE" || server.RetryAfter == nil || *server.RetryAfter != 2*time.Second {
		t.Fatalf("%+v", server)
	}

	// No RetryAfter for a 503 or 504 without the header.
	server = mustAs[*ServerError](t, send(jsonResponse(503, map[string]any{"message": "Analytics is unavailable."})))
	if server.RetryAfter != nil {
		t.Fatal(*server.RetryAfter)
	}
	message = "Analytics exceeded the query time limit. Retry with a shorter period or fewer dimensions."
	server = mustAs[*ServerError](t, send(jsonResponse(504, map[string]any{"message": message})))
	if server.Status != 504 || server.Message != message || server.RetryAfter != nil {
		t.Fatalf("%+v", server)
	}
}
