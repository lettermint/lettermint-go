# Lettermint Go SDK

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](https://opensource.org/licenses/MIT)
[![Build](https://github.com/lettermint/lettermint-go/actions/workflows/ci.yaml/badge.svg)](https://github.com/lettermint/lettermint-go/actions/workflows/ci.yaml)
[![Release](https://img.shields.io/github/release/lettermint/lettermint-go.svg?style=flat-square)](https://github.com/lettermint/lettermint-go/releases)
[![Go Reference](https://pkg.go.dev/badge/github.com/lettermint/lettermint-go/v3.svg)](https://pkg.go.dev/github.com/lettermint/lettermint-go/v3)
[![Join our Discord server](https://img.shields.io/discord/1305510095588819035?logo=discord&logoColor=eee&label=Discord&labelColor=464ce5&color=0D0E28&cacheSeconds=43200)](https://lettermint.co/r/discord)

The official Go SDK for [Lettermint](https://lettermint.co). It needs Go 1.24 or newer and has no dependencies.

Upgrading from v2? Read [UPGRADE.md](UPGRADE.md).

## Installation

```bash
go get github.com/lettermint/lettermint-go/v3
```

## Quick start

Create a client with a project sending token and send an email:

```go
import lettermint "github.com/lettermint/lettermint-go/v3"

client, err := lettermint.New(lettermint.WithSendingToken(os.Getenv("LETTERMINT_PROJECT_TOKEN")))
if err != nil {
	log.Fatal(err)
}

resp, err := client.Emails.Compose().
	From("Acme <hello@acme.com>").
	To("jane@example.com").
	Subject("Welcome to Acme").
	HTML("<p>Thanks for signing up.</p>").
	Text("Thanks for signing up.").
	Send(ctx)
if err != nil {
	log.Fatal(err)
}
fmt.Println(resp.MessageID, resp.Status) // "…", "pending"
```

Every method takes a `context.Context` first.

## Tokens

Lettermint has two kinds of API tokens:

| Option | Token | Used by | Sent as |
| --- | --- | --- | --- |
| `WithSendingToken` | Project sending token (`lm_…`) | `client.Emails` | `x-lettermint-token` header |
| `WithTeamToken` | Team API token (`lm_team_…`) | Every other part (`Domains`, `Messages`, `Projects`, …) | `Authorization: Bearer` header |

Pass one or both:

```go
client, err := lettermint.New(
	lettermint.WithSendingToken(os.Getenv("LETTERMINT_PROJECT_TOKEN")),
	lettermint.WithTeamToken(os.Getenv("LETTERMINT_TEAM_TOKEN")),
)
```

Each part uses its own token and never falls back to the other one. If the token a method needs is missing, the method returns a `*lettermint.ConfigError` that names the option (`Domains.List needs a team token; create the client with lettermint.WithTeamToken`), before any request. `client.Ping` uses the team token when it is set, otherwise the sending token. `Messages.Reschedule` and `Messages.Cancel` accept either token in the same way.

`NewFromToken` creates a client from one token and chooses its type by the format: `lm_team_` followed by letters and digits is a team token, and `lm_` followed by letters and digits is a sending token. Any other value, such as an SSO verification token (`lm_sso_…`), returns a `*ConfigError`; use `New` with `WithSendingToken` or `WithTeamToken` for it.

```go
client, err := lettermint.NewFromToken(os.Getenv("LETTERMINT_TOKEN"), lettermint.WithTimeout(10*time.Second))
```

Errors never contain a token, and printing or logging a client (`%v`, `%+v`, `%#v`, `slog`, `json.Marshal`) shows tokens as `[redacted]`.

### Options

| Option | Default | Description |
| --- | --- | --- |
| `WithSendingToken(token)` | | Project sending token. |
| `WithTeamToken(token)` | | Team API token. |
| `WithBaseURL(url)` | `https://api.lettermint.co/v1` | API base URL. |
| `WithTimeout(d)` | `30s` | Request timeout. It covers the response headers and the body. |
| `WithHTTPClient(c)` | a new `http.Client` | An HTTP client, for example with a proxy. The SDK copies it and never changes yours. |

The client holds no per-request state: create it once and share it between goroutines.

## Sending email

### The email builder

`client.Emails.Compose()` returns an `EmailBuilder`. The builder is a value: every setter returns a new builder and leaves the one it was called on unchanged, so you can keep a base builder and reuse it, also across goroutines:

```go
welcome := client.Emails.Compose().
	From("Acme <hello@acme.com>").
	Subject("Welcome to Acme").
	Tags(lettermint.MessageTagInput{Name: "campaign", Value: "welcome"})

_, err := welcome.To("jane@example.com").HTML("<p>Hi Jane</p>").Send(ctx)
_, err = welcome.To("john@example.com").HTML("<p>Hi John</p>").Send(ctx)
```

When you build an email over several statements, keep the returned builder:

```go
email := client.Emails.Compose().From("hello@acme.com").To(user.Email).Subject("Your invoice")
if user.Accountant != "" {
	email = email.CC(user.Accountant)
}
_, err := email.HTML(invoiceHTML).Send(ctx)
```

| Method | Description |
| --- | --- |
| `From(address)` | Sender, for example `Acme <hello@acme.com>`. |
| `To(...)`, `CC(...)`, `BCC(...)`, `ReplyTo(...)` | Replace the recipient list. |
| `Subject(text)` | Subject line. |
| `HTML(html)`, `Text(text)` | Bodies. An empty string removes one. |
| `Headers(map)` | Custom email headers. |
| `Metadata(map)` | Data stored with the message, not added as headers. |
| `Tags(...MessageTagInput)`, `Tag(name)` | Name/value tags, and the legacy single tag. |
| `Route(slug)` | The route to send through. |
| `ScheduledAt(when)`, `ScheduledAtTime(t)` | Delivery time: ISO 8601 or English such as `tomorrow 9am`, or a `time.Time`. |
| `Settings(SendMailRequestSettings)` | Per-email settings that override the route. |
| `SandboxResult(result)` | The result a Sandbox project simulates. |
| `Attach(Attachment)` | Adds an attachment. |
| `Send(ctx, options...)` | Sends the email. The builder can be sent again. |
| `Build()` | Returns the message in API format (`SendMailRequest`). |
| `Err()` | The error a setter recorded, if any. |

A setter that gets invalid input, such as an invalid tag, records the error on the builder it returns. `Err`, `Build` and `Send` return that error, and `Send` sends nothing. The builder it was called on stays valid.

`client.Emails.ComposeFrom(message)` starts a builder from a `SendMailRequest`.

### Plain structs

`client.Emails.Send` takes the message in the API's format, `SendMailRequest`:

```go
resp, err := client.Emails.Send(ctx, lettermint.SendMailRequest{
	From:    "Acme <hello@acme.com>",
	To:      []string{"jane@example.com"},
	ReplyTo: []string{"support@acme.com"},
	Subject: "Your order has shipped",
	HTML:    lettermint.Value(html),
	Metadata: map[string]string{"order_id": "1234"},
})
```

Optional fields are pointers (`lettermint.Ptr("transactional")`). Fields that are optional and may also be `null`, such as `HTML`, `Text` and `Tag`, are `lettermint.Nullable` values: `lettermint.Value(x)`, `lettermint.Null[T]()`, or the zero value to leave them out.

### Batch sending

Send up to 500 emails in one request. Use `Build` to include a builder:

```go
second, err := welcome.To("john@example.com").HTML("<p>Hi John</p>").Build()
results, err := client.Emails.SendBatch(ctx, []lettermint.SendMailRequest{
	{From: "hello@acme.com", To: []string{"jane@example.com"}, Subject: "Hi Jane", Text: lettermint.Value("Hello")},
	second,
})
```

### Idempotency

Pass an idempotency key to make retries safe. The API processes a key once, so a retry with the same key does not send the email again. The key applies only to the call it is passed to.

```go
client.Emails.Send(ctx, message, lettermint.WithIdempotencyKey("order-1234-confirmation"))
builder.Send(ctx, lettermint.WithIdempotencyKey("welcome-jane"))
client.Emails.SendBatch(ctx, messages, lettermint.WithIdempotencyKey("newsletter-2026-10"))
```

The SDK never retries on its own.

### Scheduling

```go
resp, err := client.Emails.Compose().
	From("hello@acme.com").
	To("jane@example.com").
	Subject("Your trial ends tomorrow").
	Text("…").
	ScheduledAtTime(time.Now().Add(24 * time.Hour)).
	Send(ctx)

if resp.Status == lettermint.MessageStatusScheduled {
	fmt.Println(*resp.ScheduledAt)
}

client.Messages.Reschedule(ctx, resp.MessageID, lettermint.RescheduleMessageRequest{ScheduledAt: "2026-10-20T09:00:00Z"})
client.Messages.Cancel(ctx, resp.MessageID)
```

### Sandbox

In a Sandbox project, nothing is delivered. Choose the simulated result per email:

```go
resp, err := client.Emails.Compose().From("hello@acme.com").To("jane@example.com").Subject("Test").Text("Test").
	SandboxResult(lettermint.SandboxResultHardBounced).
	Send(ctx)
```

### Tags

`Tags` accepts up to 20 case-sensitive name/value tags (19 when the legacy `Tag` is also set). Names match `^[A-Za-z0-9_-]{1,32}$`, may not start with `__lettermint` and must be unique. Values match `^[A-Za-z0-9_-]{1,64}$`. The SDK checks this before the request and returns a `*ClientValidationError` with the `Field`.

### Attachments

Set `Content` to raw bytes (the SDK base64-encodes them) or `ContentBase64` to content that is already encoded:

```go
pdf, _ := os.ReadFile("invoice.pdf")

client.Emails.Compose().
	From("billing@acme.com").
	To("jane@example.com").
	Subject("Your invoice").
	HTML(`<img src="cid:logo"> Your invoice is attached.`).
	Attach(lettermint.Attachment{Filename: "invoice.pdf", Content: pdf, ContentType: "application/pdf"}).
	Attach(lettermint.Attachment{Filename: "logo.png", ContentBase64: logoBase64, ContentID: "logo"}).
	Send(ctx)
```

`client.BlockedFileTypes(ctx)` lists the extensions and MIME types the API rejects.

## Team API

With a team token, the client manages domains, messages, projects, routes, statistics, suppressions, the team and webhooks:

```go
client, err := lettermint.New(lettermint.WithTeamToken(os.Getenv("LETTERMINT_TEAM_TOKEN")))

domain, err := client.Domains.Create(ctx, lettermint.StoreDomainData{Domain: "acme.com"})
_, err = client.Domains.VerifyDNSRecords(ctx, domain.ID)

project, err := client.Projects.Create(ctx, lettermint.StoreProjectData{Name: "Production"})
fmt.Println(*project.APIToken) // the new project's sending token, shown once

stats, err := client.Stats.Retrieve(ctx, lettermint.GetStatsQuery{From: "2026-10-01", To: "2026-10-31"})
html, err := client.Messages.HTML(ctx, "message-id")
```

| Field | Methods |
| --- | --- |
| `Domains` | `List`, `Iterate`, `Create`, `Retrieve`, `Delete`, `VerifyDNSRecords`, `VerifyDNSRecord`, `UpdateProjects` |
| `Messages` | `List`, `Iterate`, `Retrieve`, `Events`, `IterateEvents`, `Source`, `HTML`, `Text`, `Reschedule`, `Cancel`, `Process` |
| `Projects` | `List`, `Iterate`, `Create`, `Retrieve`, `Update`, `Delete`, `RotateToken` |
| `Projects.ReportForwarding` | `Retrieve`, `Update`, `Delete`, `Verify`, `ResendCode` |
| `Routes` | `List(ctx, projectID, …)`, `Iterate(ctx, projectID, …)`, `Create(ctx, projectID, …)`, `Retrieve`, `Update`, `Delete`, `VerifyInboundDomain` |
| `Stats` | `Retrieve` |
| `Suppressions` | `List`, `Iterate`, `Create`, `Delete` |
| `Team` | `Retrieve`, `Update`, `Usage`, `Roles` |
| `Team.Members` | `List`, `Iterate`, `Retrieve`, `UpdateAssignment` |
| `Webhooks` | `List`, `Iterate`, `Create`, `Retrieve`, `Update`, `Delete`, `Test`, `RegenerateSecret` |
| `Webhooks.Deliveries` | `List(ctx, webhookID, …)`, `Iterate(ctx, webhookID, …)`, `Retrieve(ctx, webhookID, deliveryID)` |
| (client) | `Ping`, `Analytics`, `AnalyticsPages`, `BlockedFileTypes` |

Update requests leave absent fields unchanged. A field that may be cleared is a `Nullable`: `lettermint.Null[T]()` sends `null`.

```go
client.Webhooks.Update(ctx, webhookID, lettermint.UpdateWebhookData{
	Enabled:   lettermint.Ptr(false),
	BasicAuth: lettermint.Null[lettermint.WebhookBasicAuthData](), // remove the credentials
})
```

### Query parameters and pagination

Query parameters are typed structs. Pass `nil` for none. The SDK sends them in the API's bracket syntax (`page[size]=30&filter[status]=verified&sort=-created_at`). A zero value is not sent; booleans are pointers so that `false` can be sent.

```go
page, err := client.Domains.List(ctx, &lettermint.ListDomainsQuery{
	PageSize:     30,
	FilterStatus: lettermint.DomainStatusVerified,
	Sort:         []lettermint.ListDomainsQuerySortItem{lettermint.ListDomainsQuerySortItemCreatedAtDesc},
})
fmt.Println(len(page.Data), page.NextCursor)
```

Every list has an `Iterate` method that returns an `iter.Seq2` and follows `NextCursor` until the last page:

```go
for message, err := range client.Messages.Iterate(ctx, &lettermint.ListMessagesQuery{FilterStatus: lettermint.MessageStatusHardBounced}) {
	if err != nil {
		return err
	}
	fmt.Println(message.ID, message.FromEmail)
}
```

Stop early with `break`. The SDK requests the next page only when the loop gets to it, and yields an error at most once.

### Analytics

`client.Analytics(ctx, query)` runs one analytics query. `Metrics` is the only required field; by default the API returns a summary of the last 30 days:

```go
result, err := client.Analytics(ctx, lettermint.AnalyticsQuery{
	Metrics:  []lettermint.AnalyticsMetric{lettermint.AnalyticsMetricDelivered, lettermint.AnalyticsMetricBounced, lettermint.AnalyticsMetricDeliveryRate},
	From:     lettermint.Ptr("2026-10-01"),
	To:       lettermint.Ptr("2026-10-31"),
	Timezone: lettermint.Ptr("Europe/Amsterdam"),
})
if err != nil {
	return err
}

if summary := result.Data.Summary; summary != nil {
	rate, ok := summary.Metrics.DeliveryRate.Get() // 0.9836 and true, or false when there is no data
	fmt.Println(rate, ok)
}
fmt.Println(result.Meta.Partial, result.Meta.EffectiveTo)
```

Add `Include` to ask for a `time_series` or a `breakdown`. A breakdown needs `GroupBy`, and the API returns its rows in pages of `Limit` (at most 200). `AnalyticsPages` follows `Pagination.NextCursor` for you. It returns an `iter.Seq2` that yields one whole response per request, so each page keeps its `Meta` and `Pagination`:

```go
query := lettermint.AnalyticsQuery{
	Metrics: []lettermint.AnalyticsMetric{lettermint.AnalyticsMetricDelivered, lettermint.AnalyticsMetricBounced},
	Include: []lettermint.AnalyticsSection{lettermint.AnalyticsSectionBreakdown},
	GroupBy: []lettermint.AnalyticsGroupDimension{lettermint.AnalyticsCatalogueGroupDimensionRecipientDomain},
	Sort:    &lettermint.AnalyticsSort{Metric: lettermint.AnalyticsMetricBounced, Direction: lettermint.AnalyticsSortDirectionDesc},
	Limit:   lettermint.Ptr(200),
}

var rows []lettermint.AnalyticsBreakdownRow
for page, err := range client.AnalyticsPages(ctx, query) {
	if err != nil {
		return err
	}
	rows = append(rows, page.Data.Breakdown...)
	if page.Pagination.Truncated {
		log.Println("More groups exist than the API ranks.")
	}
}
```

A cursor expires 60 seconds after its response, so read the next page promptly. An expired cursor yields a `*ValidationError` with `Errors["cursor"]`; run the query again to start over.

A few things to know when you read a response:

- A metric is `null` when the API cannot measure it for that row or bucket, and a rate is `null` when its denominator is zero. `0` means a measured zero. Metrics are `Nullable` values: `Get()` returns `false` for `null`.
- `Data.Summary`, `Data.TimeSeries` and `Data.Breakdown` are set only when `Include` asks for them. `Previous`, `Change` and `Meta.Comparison` are set only with `Compare`.
- `smtp_response_group` can be used in `GroupBy` but not as a filter dimension.
- Analytics can answer `503` or `504` when a query takes too long or the service is busy. Both return a `*ServerError`; see [Errors](#errors).

### Cancellation and timeouts

Every method takes a `context.Context`. Cancelling it, or its deadline, stops the request and returns the context's error. `lettermint.WithRequestTimeout(d)` overrides the client's timeout for one call:

```go
page, err := client.Messages.List(ctx, nil, lettermint.WithRequestTimeout(5*time.Second))
```

## Errors

Every error the SDK returns implements `lettermint.Error`. Check the types with `errors.As`:

| Type | When | Fields |
| --- | --- | --- |
| `*APIError` | Any 4xx or 5xx JSON (or empty) response | `Status`, `Code`, `Message`, `Details`, `Body` |
| `*AuthenticationError` | 401 | |
| `*PermissionError` | 403 | |
| `*NotFoundError` | 404 | |
| `*ConflictError` | 409 | |
| `*ValidationError` | 422 | `Errors` (field errors) |
| `*RateLimitError` | 429 | `RetryAfter` |
| `*ServerError` | 5xx | `RetryAfter` (when the API sent `Retry-After`) |
| `*TimeoutError` | No complete response within the timeout | `Timeout` |
| `*ConnectionError` | The request failed (DNS, TLS, refused, reset) | `Err` |
| `*UnexpectedResponseError` | An empty or non-JSON body where JSON was expected, or an error page such as a proxy's HTML 502 | `Status`, `BodyExcerpt` |
| `*RedirectError` | A 3xx response. Redirects are never followed, so tokens never go elsewhere. | `Status` |
| `*ConfigError` | A missing or unrecognised token, an invalid option or ID | `Message` |
| `*ClientValidationError` | The SDK rejected the request before sending it, such as invalid tags | `Field`, `Message` |
| `*WebhookVerificationError` | A webhook delivery is not genuine | `Reason` |

The status types embed `APIError` and unwrap to it, so `errors.As(err, &apiErr)` also matches a `*RateLimitError`. `Code` and `Message` come from the API's error body (`{"error": {"code", "message", "details"}}` or `{"message", "errors"}`). A cancelled context returns the context's error, not an SDK error.

```go
resp, err := client.Emails.Send(ctx, message, lettermint.WithIdempotencyKey(key))

var validation *lettermint.ValidationError
var rateLimit *lettermint.RateLimitError
var timeout *lettermint.TimeoutError
var apiErr *lettermint.APIError
switch {
case errors.As(err, &validation):
	log.Println(validation.Message, validation.Errors)
case errors.As(err, &rateLimit):
	// Wait for rateLimit.RetryAfter (when set), then retry with the same key.
case errors.As(err, &timeout):
	// The outcome is unknown. Retry with the same key.
case errors.As(err, &apiErr):
	log.Println(apiErr.Status, apiErr.Code, apiErr.Message)
case err != nil:
	return err
}
```

Errors never contain request headers or tokens.

## Webhooks

Verify each webhook delivery before you trust it. Use the webhook's signing secret (`whsec_…`), not an API token, and pass the **raw** request body: the signature covers the exact bytes, so decoding and re-encoding the JSON breaks it.

```go
webhook, err := lettermint.NewWebhook(os.Getenv("LETTERMINT_WEBHOOK_SECRET"))
if err != nil {
	log.Fatal(err)
}

http.HandleFunc("POST /webhooks/lettermint", func(w http.ResponseWriter, r *http.Request) {
	event, err := webhook.VerifyRequest(r)
	var verification *lettermint.WebhookVerificationError
	if errors.As(err, &verification) {
		http.Error(w, "invalid signature", http.StatusBadRequest)
		return
	}
	if err != nil {
		http.Error(w, "error", http.StatusInternalServerError)
		return
	}
	switch event.Event {
	case lettermint.WebhookEventMessageDelivered:
		var data struct {
			MessageID string `json:"message_id"`
		}
		_ = event.DecodeData(&data)
	}
	w.WriteHeader(http.StatusNoContent)
})
```

`VerifyRequest(r)` reads the body (at most 50 MiB, see `WithMaxBodyBytes`) and puts it back on `r.Body`. `Verify(rawBody, headers)` takes the body and an `http.Header`. Both require `X-Lettermint-Signature` and `X-Lettermint-Delivery` (header names are case-insensitive), check the HMAC-SHA256 signature in constant time against every `v1` value, check that the delivery timestamp equals the signed one and is within the tolerance, and return the payload. Otherwise they return a `*WebhookVerificationError` with a `Reason`: `signature_header_missing`, `signature_header_malformed`, `delivery_header_missing`, `delivery_timestamp_mismatch`, `timestamp_out_of_tolerance`, `signature_mismatch`, `body_invalid` or `payload_invalid`.

The default tolerance is 5 minutes in either direction. Change it with `lettermint.NewWebhook(secret, lettermint.WithTolerance(time.Minute))`. `0` accepts only the current second; it does not disable the check. A valid signature does not prevent a repeated delivery within the tolerance, so track `event.ID` if you must not process an event twice.

If the headers are not at hand, call `webhook.VerifySignature(rawBody, signatureHeader, deliveryHeader)`; an empty `deliveryHeader` skips the timestamp comparison.

`event.Event` is a `WebhookEvent` (`message.delivered`, `message.hard_bounced`, …). Unknown events pass through as their string. `event.Data` is the raw JSON of the event data; `event.Raw` is the whole verified body.

## Types

Request and response types are generated from the Lettermint API specification, for example `SendMailRequest`, `SendMailResponse`, `DomainData` and `ListDomainsResponse` (an alias of `CursorPage[DomainListData]`). Enums are open string types (`MessageStatus`, `WebhookEvent`, …) with a constant per known value; values the API adds later decode unchanged, so give `switch` statements a `default` branch. The API's error bodies are `ApiErrorBody` and `ValidationErrorBody`.

| Property in the API | Go field |
| --- | --- |
| required | `T` |
| required, may be `null` | `*T` (nil is `null`) |
| optional | `*T` (nil is absent); slices and maps are nil when absent |
| optional, may be `null` | `Nullable[T]`: absent (zero value), `Null[T]()` or `Value(v)`; read it with `Get()` |

## Versioning

The User-Agent header is `lettermint-go/<version> (<Go version>)`; `lettermint.Version()` reads the version from the build information. Releases are tags (`v3.x.y`); Go fetches them through the module proxy.

## Development

```bash
go vet ./...
go test -race ./...
sh scripts/generate.sh --check
```

`generated_types.go` and `generated_operations.go` are generated by the private [SDK generator](https://github.com/lettermint/sdk-generator). Do not edit them by hand. With a checkout of the generator, `go generate` (or `sh scripts/generate.sh`) regenerates them and `sh scripts/generate.sh --check` verifies them; set `LETTERMINT_SDK_GENERATOR` to the checkout (default `../sdk-generator`). Without the generator, as in CI, `--check` only verifies the generated headers.

## Changelog

Please see [CHANGELOG](CHANGELOG.md) for more information on what has changed recently.

## Security Vulnerabilities

Please review [our security policy](../../security/policy) on how to report security vulnerabilities.

## Credits

- [Bjarn Bronsveld](https://github.com/bjarn)
- [All Contributors](../../contributors)

## License

The MIT License (MIT). Please see [License File](LICENSE) for more information.
