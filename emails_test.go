package lettermint

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
)

func accepted(r recorded, _ int) *http.Response {
	var body struct {
		Subject string `json:"subject"`
	}
	_ = json.Unmarshal(r.RawBody, &body)
	return jsonResponse(202, map[string]any{"message_id": "msg_" + body.Subject, "status": "pending"})
}

func TestSendPostsExactlyTheMessage(t *testing.T) {
	client, fake := newTestClient(t, accepted)
	resp, err := client.Emails.Send(context.Background(), SendMailRequest{
		From:    "Acme <hello@acme.test>",
		To:      []string{"jane@example.test"},
		Subject: "Welcome",
		HTML:    Value("<p>Hi & welcome</p>"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.MessageID != "msg_Welcome" || resp.Status != MessageStatusPending {
		t.Fatalf("%+v", resp)
	}
	r := fake.all()[0]
	if r.URL.String() != "https://api.lettermint.co/v1/send" || r.Method != "POST" {
		t.Fatal(r.Method, r.URL)
	}
	if string(r.RawBody) != `{"from":"Acme <hello@acme.test>","to":["jane@example.test"],"subject":"Welcome","html":"<p>Hi & welcome</p>"}` {
		t.Fatal(string(r.RawBody))
	}
	if r.Header.Get("x-lettermint-token") != sendingFixture || r.Header.Get("Content-Type") != "application/json" || r.Header.Get("Accept") != "application/json" {
		t.Fatal(r.Header)
	}
	if !strings.HasPrefix(r.Header.Get("User-Agent"), "lettermint-go/") || r.Header.Get("Authorization") != "" || r.Header.Get("Idempotency-Key") != "" {
		t.Fatal(r.Header)
	}
}

func TestIdempotencyKeyIsPerCall(t *testing.T) {
	client, fake := newTestClient(t, accepted)
	ctx := context.Background()
	message := SendMailRequest{From: "a@example.test", To: []string{"b@example.test"}, Subject: "One"}
	if _, err := client.Emails.Send(ctx, message, WithIdempotencyKey("order-123")); err != nil {
		t.Fatal(err)
	}
	message.Subject = "Two"
	if _, err := client.Emails.Send(ctx, message); err != nil {
		t.Fatal(err)
	}
	requests := fake.all()
	if requests[0].Header.Get("Idempotency-Key") != "order-123" || requests[1].Header.Get("Idempotency-Key") != "" {
		t.Fatal(requests[0].Header, requests[1].Header)
	}
	for _, key := range []string{"", "line\nbreak", "nul\x00"} {
		_, err := client.Emails.Send(ctx, message, WithIdempotencyKey(key))
		if mustAs[*ClientValidationError](t, err).Field != "IdempotencyKey" {
			t.Fatal(err)
		}
	}
	if len(fake.all()) != 2 {
		t.Fatal("an invalid key sends nothing")
	}
}

func TestSendValidatesTagsBeforeAnyRequest(t *testing.T) {
	client, fake := newTestClient(t, accepted)
	_, err := client.Emails.Send(context.Background(), SendMailRequest{From: "a", To: []string{"b"}, Subject: "x", Tags: []MessageTagInput{{Name: "not valid!", Value: "x"}}})
	if mustAs[*ClientValidationError](t, err).Field != "tags" {
		t.Fatal(err)
	}
	_, err = client.Emails.Send(context.Background(), SendMailRequest{From: "a", To: []string{"b"}, Subject: "x", Attachments: []MessageAttachmentInput{{Content: "SGk="}}})
	if mustAs[*ClientValidationError](t, err).Field != "attachments[0]" {
		t.Fatal(err)
	}
	if len(fake.all()) != 0 {
		t.Fatal("nothing is sent")
	}
}

func TestTagRules(t *testing.T) {
	tags := func(n int) []MessageTagInput {
		result := make([]MessageTagInput, n)
		for i := range result {
			result[i] = MessageTagInput{Name: fmt.Sprintf("t%d", i), Value: "v"}
		}
		return result
	}
	invalid := map[string][]MessageTagInput{
		"bad name":        {{Name: "has space", Value: "v"}},
		"long name":       {{Name: strings.Repeat("a", 33), Value: "v"}},
		"empty name":      {{Name: "", Value: "v"}},
		"reserved prefix": {{Name: "__LetterMint_x", Value: "v"}},
		"bad value":       {{Name: "a", Value: "v!"}},
		"long value":      {{Name: "a", Value: strings.Repeat("b", 65)}},
		"empty value":     {{Name: "a", Value: ""}},
		"duplicate":       {{Name: "a", Value: "1"}, {Name: "a", Value: "2"}},
		"too many":        tags(21),
	}
	for label, value := range invalid {
		if validateTags(value, false, "tags") == nil {
			t.Errorf("%s: expected an error", label)
		}
	}
	if err := validateTags(tags(20), false, "tags"); err != nil {
		t.Fatal(err)
	}
	if err := validateTags(tags(20), true, "tags"); err == nil || !strings.Contains(err.Error(), "legacy tag and no more than 19") {
		t.Fatal(err)
	}
	if err := validateTags(tags(19), true, "tags"); err != nil {
		t.Fatal(err)
	}
	if err := validateTags([]MessageTagInput{{Name: "a", Value: "1"}, {Name: "A", Value: "1"}, {Name: strings.Repeat("a", 32), Value: strings.Repeat("b", 64)}}, false, "tags"); err != nil {
		t.Fatal("names are case-sensitive and the limits are inclusive:", err)
	}
	client, _ := newTestClient(t, accepted)
	if client.Emails.Compose().Tags(tags(20)...).Err() != nil {
		t.Fatal("20 tags")
	}
	if client.Emails.Compose().Tags(tags(20)...).Tag("legacy").Err() == nil {
		t.Fatal("20 tags and a legacy tag")
	}
}

func TestSendBatch(t *testing.T) {
	client, fake := newTestClient(t, func(recorded, int) *http.Response {
		return jsonResponse(202, []any{
			map[string]any{"message_id": "m1", "status": "pending"},
			map[string]any{"message_id": "m2", "status": "scheduled", "scheduled_at": "2026-10-05T09:00:00Z"},
		})
	})
	built, err := client.Emails.Compose().From("a@example.test").To("c@example.test").Subject("Two").Text("Hi").ScheduledAt("2026-10-05T09:00:00Z").Build()
	if err != nil {
		t.Fatal(err)
	}
	results, err := client.Emails.SendBatch(context.Background(), []SendMailRequest{
		{From: "a@example.test", To: []string{"b@example.test"}, Subject: "One", Text: Value("Hi")},
		built,
	}, WithIdempotencyKey("batch-1"))
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 || results[1].Status != MessageStatusScheduled || results[1].ScheduledAt == nil || *results[1].ScheduledAt != "2026-10-05T09:00:00Z" {
		t.Fatalf("%+v", results)
	}
	r := fake.all()[0]
	if r.URL.Path != "/v1/send/batch" || r.Header.Get("Idempotency-Key") != "batch-1" {
		t.Fatal(r.URL, r.Header)
	}
	want := `[{"from":"a@example.test","to":["b@example.test"],"subject":"One","text":"Hi"},{"from":"a@example.test","to":["c@example.test"],"subject":"Two","scheduled_at":"2026-10-05T09:00:00Z","text":"Hi"}]`
	if string(r.RawBody) != want {
		t.Fatal(string(r.RawBody))
	}
	_, err = client.Emails.SendBatch(context.Background(), []SendMailRequest{{From: "a"}, {From: "b", Tags: []MessageTagInput{{Name: "!", Value: "x"}}}})
	if mustAs[*ClientValidationError](t, err).Field != "messages[1].tags" {
		t.Fatal(err)
	}
}

func TestBuilderSetsEveryField(t *testing.T) {
	client, fake := newTestClient(t, accepted)
	no := false
	_, err := client.Emails.Compose().
		From("Acme <hello@acme.test>").
		To("jane@example.test", "john@example.test").
		CC("cc@example.test").
		BCC("bcc@example.test").
		ReplyTo("support@acme.test").
		Subject("Everything").
		HTML("<p>Hi</p>").
		Text("Hi").
		Headers(map[string]string{"X-Campaign": "welcome"}).
		Metadata(map[string]string{"order_id": "1234"}).
		Tag("legacy").
		Tags(MessageTagInput{Name: "campaign", Value: "welcome"}).
		Route("transactional").
		ScheduledAtTime(time.Date(2026, 10, 5, 9, 0, 0, 0, time.FixedZone("CEST", 2*3600))).
		Settings(SendMailRequestSettings{TrackOpens: &no, TLS: Ptr(TlsPolicyEnforced)}).
		SandboxResult(SandboxResultHardBounced).
		Attach(Attachment{Filename: "a.txt", Content: []byte("Hello World")}).
		Attach(Attachment{Filename: "logo.png", ContentBase64: "SGk=", ContentType: "image/png", ContentID: "logo"}).
		Send(context.Background(), WithIdempotencyKey("everything"))
	if err != nil {
		t.Fatal(err)
	}
	got := fake.all()[0].JSON(t)
	want := map[string]any{
		"from": "Acme <hello@acme.test>", "to": []any{"jane@example.test", "john@example.test"},
		"cc": []any{"cc@example.test"}, "bcc": []any{"bcc@example.test"}, "reply_to": []any{"support@acme.test"},
		"subject": "Everything", "html": "<p>Hi</p>", "text": "Hi",
		"headers": map[string]any{"X-Campaign": "welcome"}, "metadata": map[string]any{"order_id": "1234"},
		"tag": "legacy", "tags": []any{map[string]any{"name": "campaign", "value": "welcome"}},
		"route": "transactional", "scheduled_at": "2026-10-05T07:00:00.000Z",
		"settings":       map[string]any{"track_opens": false, "tls": "enforced"},
		"sandbox_result": "hard_bounced",
		"attachments": []any{
			map[string]any{"filename": "a.txt", "content": "SGVsbG8gV29ybGQ="},
			map[string]any{"filename": "logo.png", "content": "SGk=", "content_type": "image/png", "content_id": "logo"},
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got  %v\nwant %v", got, want)
	}
	if fake.all()[0].Header.Get("Idempotency-Key") != "everything" {
		t.Fatal("key")
	}
}

func TestBuilderIsAReusableTemplate(t *testing.T) {
	client, fake := newTestClient(t, accepted)
	ctx := context.Background()
	base := client.Emails.Compose().From("hello@acme.test").Subject("Welcome").Tags(MessageTagInput{Name: "campaign", Value: "welcome"})
	if _, err := base.To("jane@example.test").HTML("<p>Hi Jane</p>").Send(ctx, WithIdempotencyKey("jane")); err != nil {
		t.Fatal(err)
	}
	if _, err := base.To("john@example.test").Send(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := base.Send(ctx); err != nil {
		t.Fatal(err)
	}
	requests := fake.all()
	john := requests[1].JSON(t)
	if john["to"].([]any)[0] != "john@example.test" || john["html"] != nil || requests[1].Header.Get("Idempotency-Key") != "" {
		t.Fatalf("%v %v", john, requests[1].Header)
	}
	if string(requests[2].RawBody) != `{"from":"hello@acme.test","to":[],"subject":"Welcome","tags":[{"name":"campaign","value":"welcome"}]}` {
		t.Fatalf("the base builder is unchanged: %s", requests[2].RawBody)
	}
}

func TestBuilderDoesNotShareSlicesOrMaps(t *testing.T) {
	client, _ := newTestClient(t, accepted)
	base := client.Emails.Compose().Attach(Attachment{Filename: "base.txt", ContentBase64: "YQ=="})
	one, _ := base.Attach(Attachment{Filename: "one.txt", ContentBase64: "Yg=="}).Build()
	two, _ := base.Attach(Attachment{Filename: "two.txt", ContentBase64: "Yw=="}).Build()
	again, _ := base.Build()
	if len(one.Attachments) != 2 || len(two.Attachments) != 2 || one.Attachments[1].Filename != "one.txt" || two.Attachments[1].Filename != "two.txt" || len(again.Attachments) != 1 {
		t.Fatalf("%v %v %v", one.Attachments, two.Attachments, again.Attachments)
	}

	recipients := []string{"a@example.test"}
	headers := map[string]string{"X-A": "1"}
	tags := []MessageTagInput{{Name: "a", Value: "1"}}
	builder := client.Emails.Compose().To(recipients...).Headers(headers).Tags(tags...)
	recipients[0] = "mallory@example.test"
	headers["X-A"] = "changed"
	tags[0].Name = "changed"
	built, _ := builder.Build()
	if built.To[0] != "a@example.test" || built.Headers["X-A"] != "1" || built.Tags[0].Name != "a" {
		t.Fatalf("%+v", built)
	}
	built.To[0] = "changed@example.test"
	built.Headers["X-A"] = "changed"
	rebuilt, _ := builder.Build()
	if rebuilt.To[0] != "a@example.test" || rebuilt.Headers["X-A"] != "1" {
		t.Fatal("Build returns a copy")
	}

	from := SendMailRequest{From: "a@example.test", To: []string{"b@example.test"}, Subject: "x"}
	composed := client.Emails.ComposeFrom(from)
	from.To[0] = "changed@example.test"
	if built, _ := composed.Build(); built.To[0] != "b@example.test" {
		t.Fatal("ComposeFrom copies the message")
	}
}

func TestInvalidSetterLeavesTheBuilderItWasCalledOnValid(t *testing.T) {
	client, fake := newTestClient(t, accepted)
	base := client.Emails.Compose().From("a@example.test").To("b@example.test").Subject("Valid").Tags(MessageTagInput{Name: "ok", Value: "1"})
	broken := base.Tags(MessageTagInput{Name: "not a valid tag name!", Value: "x"})
	if mustAs[*ClientValidationError](t, broken.Err()).Field != "tags" {
		t.Fatal(broken.Err())
	}
	if _, err := broken.Build(); err == nil {
		t.Fatal("Build returns the recorded error")
	}
	if _, err := broken.Text("still broken").Send(context.Background()); err == nil {
		t.Fatal("later setters keep the error")
	}
	if len(fake.all()) != 0 {
		t.Fatal("a broken builder sends nothing")
	}
	if base.Err() != nil {
		t.Fatal(base.Err())
	}
	if _, err := base.Send(context.Background()); err != nil {
		t.Fatal(err)
	}
	if got := fake.all()[0].JSON(t)["tags"]; fmt.Sprint(got) != "[map[name:ok value:1]]" {
		t.Fatal(got)
	}
	both := base.Attach(Attachment{Filename: "x", Content: []byte("a"), ContentBase64: "YQ=="})
	if mustAs[*ClientValidationError](t, both.Err()).Field != "attachments" {
		t.Fatal(both.Err())
	}
	var zero EmailBuilder
	_, err := zero.Send(context.Background())
	mustAs[*ConfigError](t, err)
}

func TestEmptyStringsRemoveOptionalFields(t *testing.T) {
	client, _ := newTestClient(t, accepted)
	full := client.Emails.Compose().HTML("<p>x</p>").Text("x").Tag("t").Route("r").ScheduledAt("tomorrow")
	cleared, _ := full.HTML("").Text("").Tag("").Route("").ScheduledAtTime(time.Time{}).Build()
	data, _ := json.Marshal(cleared)
	if string(data) != `{"from":"","to":[],"subject":""}` {
		t.Fatal(string(data))
	}
}

func TestBuilderPrintsTheMessageWithoutCredentials(t *testing.T) {
	client, _ := newTestClient(t, accepted)
	builder := client.Emails.Compose().From("a@example.test").Subject("Hi")
	got := fmt.Sprintf("%+v", builder)
	if got != `lettermint.EmailBuilder{Message: {"from":"a@example.test","to":[],"subject":"Hi"}}` {
		t.Fatal(got)
	}
	data, _ := json.Marshal(builder)
	if string(data) != `{"from":"a@example.test","to":[],"subject":"Hi"}` {
		t.Fatal(string(data))
	}
}

// Two emails composed at the same time on one client, with a scheduling
// point between setters, keep their own fields and keys.
func TestConcurrentComposeAndSend(t *testing.T) {
	client, fake := newTestClient(t, accepted)
	var wait sync.WaitGroup
	for i := range 20 {
		wait.Add(1)
		go func() {
			defer wait.Done()
			name := fmt.Sprintf("user%d", i)
			builder := client.Emails.Compose()
			for _, set := range []func(EmailBuilder) EmailBuilder{
				func(b EmailBuilder) EmailBuilder { return b.From(name + "@sender.test") },
				func(b EmailBuilder) EmailBuilder { return b.To(name + "@example.test") },
				func(b EmailBuilder) EmailBuilder { return b.Subject(name) },
				func(b EmailBuilder) EmailBuilder { return b.Text("Hi " + name) },
			} {
				builder = set(builder)
				time.Sleep(time.Millisecond)
			}
			if _, err := builder.Send(context.Background(), WithIdempotencyKey(name)); err != nil {
				t.Error(err)
			}
		}()
	}
	wait.Wait()
	for _, r := range fake.all() {
		body := r.JSON(t)
		name := body["subject"].(string)
		if body["from"] != name+"@sender.test" || body["to"].([]any)[0] != name+"@example.test" || body["text"] != "Hi "+name || r.Header.Get("Idempotency-Key") != name {
			t.Fatalf("mixed email: %v %v", body, r.Header)
		}
	}
	if len(fake.all()) != 20 {
		t.Fatal(len(fake.all()))
	}
}

func TestNothingIsLeftAfterAFailedRequest(t *testing.T) {
	client, fake := newTestClient(t, func(r recorded, i int) *http.Response {
		if i == 0 {
			return jsonResponse(500, map[string]any{"message": "Server Error"})
		}
		return accepted(r, i)
	})
	ctx := context.Background()
	_, err := client.Emails.Compose().From("a@example.test").To("b@example.test").CC("cc@example.test").Subject("A").Send(ctx, WithIdempotencyKey("a"))
	if mustAs[*ServerError](t, err).Status != 500 {
		t.Fatal(err)
	}
	if _, err := client.Emails.Compose().From("c@example.test").To("d@example.test").Subject("C").Send(ctx); err != nil {
		t.Fatal(err)
	}
	second := fake.all()[1]
	if second.Header.Get("Idempotency-Key") != "" || string(second.RawBody) != `{"from":"c@example.test","to":["d@example.test"],"subject":"C"}` {
		t.Fatal(second.Header, string(second.RawBody))
	}
}

func TestSendBatchItemsSendsPerEmailKeys(t *testing.T) {
	client, fake := newTestClient(t, func(recorded, int) *http.Response {
		return jsonResponse(202, []any{
			map[string]any{"message_id": "m1", "status": "pending"},
			map[string]any{"message_id": "m2", "status": "pending"},
			map[string]any{"message_id": "m3", "status": "pending"},
		})
	})
	built, err := client.Emails.Compose().From("a@example.test").To("c@example.test").Subject("Three").Text("Hi").BuildBatchItem("key-3")
	if err != nil {
		t.Fatal(err)
	}
	results, err := client.Emails.SendBatchItems(context.Background(), []SendBatchMailItem{
		{From: "a@example.test", To: []string{"b@example.test"}, Subject: "One", Text: Value("Hi"), IdempotencyKey: Ptr("key-1")},
		{From: "a@example.test", To: []string{"b@example.test"}, Subject: "Two", Text: Value("Hi")},
		built,
	}, WithIdempotencyKey("batch-1"))
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 3 || results[2].MessageID != "m3" {
		t.Fatalf("%+v", results)
	}
	r := fake.all()[0]
	if r.Method != "POST" || r.URL.String() != "https://api.lettermint.co/v1/send/batch" {
		t.Fatal(r.Method, r.URL)
	}
	if r.Header.Get("Idempotency-Key") != "batch-1" || r.Header.Get("x-lettermint-token") != sendingFixture || r.Header.Get("Authorization") != "" {
		t.Fatal(r.Header)
	}
	want := `[{"from":"a@example.test","to":["b@example.test"],"subject":"One","text":"Hi","idempotency_key":"key-1"},` +
		`{"from":"a@example.test","to":["b@example.test"],"subject":"Two","text":"Hi"},` +
		`{"from":"a@example.test","to":["c@example.test"],"subject":"Three","text":"Hi","idempotency_key":"key-3"}]`
	if string(r.RawBody) != want {
		t.Fatal(string(r.RawBody))
	}

	// Without the header option, no Idempotency-Key header is sent.
	if _, err := client.Emails.SendBatchItems(context.Background(), []SendBatchMailItem{{From: "a", To: []string{"b"}, Subject: "x", IdempotencyKey: Ptr("k")}}); err != nil {
		t.Fatal(err)
	}
	if got := fake.all()[1].Header.Get("Idempotency-Key"); got != "" {
		t.Fatal(got)
	}
	// A nil slice sends an empty array.
	if _, err := client.Emails.SendBatchItems(context.Background(), nil); err != nil {
		t.Fatal(err)
	}
	if got := string(fake.all()[2].RawBody); got != "[]" {
		t.Fatal(got)
	}
}

func TestSendBatchItemsValidatesBeforeSending(t *testing.T) {
	client, fake := newTestClient(t, func(recorded, int) *http.Response { return jsonResponse(202, []any{}) })
	ctx := context.Background()
	item := func(key *string) SendBatchMailItem {
		return SendBatchMailItem{From: "a", To: []string{"b"}, Subject: "x", IdempotencyKey: key}
	}
	cases := map[string]struct {
		items   []SendBatchMailItem
		field   string
		message string
	}{
		"empty key":     {[]SendBatchMailItem{item(nil), item(Ptr(""))}, "items[1].idempotency_key", "1-255 characters"},
		"long key":      {[]SendBatchMailItem{item(Ptr(strings.Repeat("k", 256)))}, "items[0].idempotency_key", "1-255 characters"},
		"duplicate key": {[]SendBatchMailItem{item(Ptr("same")), item(nil), item(Ptr("same"))}, "items[2].idempotency_key", "items[0] has the same key"},
		"invalid tag":   {[]SendBatchMailItem{item(nil), {From: "a", Tags: []MessageTagInput{{Name: "!", Value: "x"}}}}, "items[1].tags", "message tag names"},
		"attachment":    {[]SendBatchMailItem{{From: "a", Attachments: []MessageAttachmentInput{{Content: "SGk="}}}}, "items[0].attachments[0]", "filename"},
	}
	for name, c := range cases {
		_, err := client.Emails.SendBatchItems(ctx, c.items)
		validation := mustAs[*ClientValidationError](t, err)
		if validation.Field != c.field || !strings.Contains(validation.Message, c.message) {
			t.Errorf("%s: %+v", name, validation)
		}
	}
	if len(fake.all()) != 0 {
		t.Fatal("invalid batches send nothing")
	}
	// 255 characters, counted as characters rather than bytes, are allowed.
	if _, err := client.Emails.SendBatchItems(ctx, []SendBatchMailItem{item(Ptr(strings.Repeat("é", 255))), item(Ptr("other"))}); err != nil {
		t.Fatal(err)
	}
	teamOnly, _ := New(WithTeamToken(teamFixture), WithHTTPClient(&http.Client{Transport: fake}))
	_, err := teamOnly.Emails.SendBatchItems(ctx, []SendBatchMailItem{item(nil)})
	if msg := mustAs[*ConfigError](t, err).Message; msg != "Emails.SendBatchItems needs a sending token; create the client with lettermint.WithSendingToken" {
		t.Fatal(msg)
	}
}

func TestBuildBatchItem(t *testing.T) {
	client, fake := newTestClient(t, accepted)
	builder := client.Emails.Compose().
		From("Acme <hello@acme.test>").
		To("jane@example.test").
		CC("cc@example.test").
		Subject("Everything").
		HTML("<p>Hi</p>").
		Metadata(map[string]string{"order_id": "1234"}).
		Tags(MessageTagInput{Name: "campaign", Value: "welcome"}).
		Route("transactional").
		Attach(Attachment{Filename: "a.txt", Content: []byte("Hi")})
	message, err := builder.Build()
	if err != nil {
		t.Fatal(err)
	}
	item, err := builder.BuildBatchItem("order-1234")
	if err != nil {
		t.Fatal(err)
	}
	if item.IdempotencyKey == nil || *item.IdempotencyKey != "order-1234" {
		t.Fatal(item.IdempotencyKey)
	}
	itemJSON, _ := json.Marshal(item)
	messageJSON, _ := json.Marshal(message)
	if want := strings.TrimSuffix(string(messageJSON), "}") + `,"idempotency_key":"order-1234"}`; string(itemJSON) != want {
		t.Fatalf("%s\nwant %s", itemJSON, want)
	}
	// The item is a copy, and single sends never carry the per-email key.
	item.To[0] = "changed@example.test"
	if _, err := builder.Send(context.Background()); err != nil {
		t.Fatal(err)
	}
	body := fake.all()[0].JSON(t)
	if _, ok := body["idempotency_key"]; ok || body["to"].([]any)[0] != "jane@example.test" {
		t.Fatal(body)
	}

	if _, err := builder.BuildBatchItem(""); mustAs[*ClientValidationError](t, err).Field != "idempotency_key" {
		t.Fatal(err)
	}
	invalid := builder.Tags(MessageTagInput{Name: "!", Value: "x"})
	if _, err := invalid.BuildBatchItem("k"); err == nil || err != invalid.Err() {
		t.Fatal(err)
	}
}

// batchItem must copy every field of SendMailRequest, and SendBatchMailItem
// must have exactly those fields plus IdempotencyKey. When the types are
// regenerated with a new field, this test fails until batchItem copies it.
func TestBatchItemCopiesEveryField(t *testing.T) {
	messageType := reflect.TypeFor[SendMailRequest]()
	itemType := reflect.TypeFor[SendBatchMailItem]()
	if itemType.NumField() != messageType.NumField()+1 {
		t.Fatalf("SendBatchMailItem has %d fields, SendMailRequest %d", itemType.NumField(), messageType.NumField())
	}
	for i := range messageType.NumField() {
		field := messageType.Field(i)
		other, ok := itemType.FieldByName(field.Name)
		if !ok || other.Type != field.Type || other.Tag != field.Tag {
			t.Errorf("SendBatchMailItem.%s does not match SendMailRequest.%s", field.Name, field.Name)
		}
	}
	if key, ok := itemType.FieldByName("IdempotencyKey"); !ok || key.Tag.Get("json") != "idempotency_key,omitzero" {
		t.Fatal("SendBatchMailItem.IdempotencyKey")
	}
	// Fill every field of a message with a non-zero value and check the copy.
	var message SendMailRequest
	value := reflect.ValueOf(&message).Elem()
	for i := range value.NumField() {
		fillNonZero(t, value.Field(i))
	}
	item := reflect.ValueOf(batchItem(message))
	for i := range value.NumField() {
		name := messageType.Field(i).Name
		if !reflect.DeepEqual(item.FieldByName(name).Interface(), value.Field(i).Interface()) {
			t.Errorf("batchItem does not copy %s", name)
		}
	}
	if item.FieldByName("IdempotencyKey").Interface().(*string) != nil {
		t.Fatal("batchItem sets no key")
	}
}

func fillNonZero(t *testing.T, v reflect.Value) {
	t.Helper()
	switch v.Kind() {
	case reflect.String:
		v.SetString("x")
	case reflect.Bool:
		v.SetBool(true)
	case reflect.Pointer:
		v.Set(reflect.New(v.Type().Elem()))
		fillNonZero(t, v.Elem())
	case reflect.Slice:
		v.Set(reflect.MakeSlice(v.Type(), 1, 1))
		fillNonZero(t, v.Index(0))
	case reflect.Map:
		v.Set(reflect.MakeMap(v.Type()))
		v.SetMapIndex(reflect.ValueOf("k"), reflect.ValueOf("v"))
	case reflect.Struct:
		if v.Type() == reflect.TypeFor[Nullable[string]]() {
			v.Set(reflect.ValueOf(Value("x")))
			return
		}
		exported := 0
		for i := range v.NumField() {
			if v.Type().Field(i).IsExported() {
				fillNonZero(t, v.Field(i))
				exported++
			}
		}
		if exported == 0 {
			t.Fatalf("fillNonZero: %s", v.Type())
		}
	default:
		t.Fatalf("fillNonZero: %s", v.Type())
	}
}

func TestSendBatchErrorsListAcceptedEmails(t *testing.T) {
	client, _ := newTestClient(t, func(_ recorded, i int) *http.Response {
		if i == 0 {
			return jsonResponse(500, map[string]any{"error": map[string]any{
				"code":    "INTERNAL_ERROR",
				"message": "Server error",
				"details": map[string]any{"accepted": []any{map[string]any{"index": 0, "message_id": "m1"}}},
			}})
		}
		return jsonResponse(409, map[string]any{"error": map[string]any{"code": "IDEMPOTENCY_CONFLICT", "message": "The idempotency key was used with different content."}})
	})
	items := []SendBatchMailItem{{From: "a", To: []string{"b"}, Subject: "x", IdempotencyKey: Ptr("k1")}, {From: "a", To: []string{"c"}, Subject: "y", IdempotencyKey: Ptr("k2")}}
	_, err := client.Emails.SendBatchItems(context.Background(), items)
	details, _ := mustAs[*ServerError](t, err).Details.(map[string]any)
	acceptedEmails, _ := details["accepted"].([]any)
	if len(acceptedEmails) != 1 || acceptedEmails[0].(map[string]any)["message_id"] != "m1" {
		t.Fatalf("%#v", details)
	}
	_, err = client.Emails.SendBatchItems(context.Background(), items)
	if conflict := mustAs[*ConflictError](t, err); conflict.Code != "IDEMPOTENCY_CONFLICT" {
		t.Fatalf("%+v", conflict)
	}
}
