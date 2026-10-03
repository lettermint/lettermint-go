# Upgrade guide

- [Upgrade from v2 to v3](#upgrade-from-v2-to-v3)
- [Upgrade from v1 to v2](#upgrade-to-v2)

# Upgrade from v2 to v3

v2 no longer receives updates, including fixes. Upgrade to v3 to keep getting them.

v3 is a new major version with the module path `github.com/lettermint/lettermint-go/v3`. It follows the shared design of the Lettermint SDK majors (the Node.js SDK 3.0 is the reference). The main changes are about safety:

- In v2, `EmailBuilder` was a pointer that `Send` reset afterwards. Setting fields over several statements, reusing a builder, or a validation error before `Send` (which skipped the reset) could carry recipients, content and the `Idempotency-Key` into the next email. In v3 the builder is a value: every setter returns a copy.
- v2 followed redirects and sent the `x-lettermint-token` header to the new location. v3 never follows redirects.
- `WithTimeout` changed the `Timeout` of the `http.Client` you passed with `WithHTTPClient`. v3 never changes your client.
- `fmt.Printf("%+v", client)` printed the API token. v3 shows tokens as `[redacted]` everywhere.
- Webhook verification checked only the last `v1` signature, rejected uppercase hex and treated `X-Lettermint-Delivery` as optional. v3 accepts any matching `v1`, accepts uppercase hex and requires the delivery header.
- Path IDs such as `..` could change the request path. v3 rejects empty IDs, `.` and `..`.
- The User-Agent reported version `1.0.0`. v3 reads the version from the build information.

## Highlights

- One client: `lettermint.New(lettermint.WithSendingToken(…), lettermint.WithTeamToken(…))`, or `lettermint.NewFromToken(token)`. It replaces `lettermint.New(token)` and `lettermint.NewAPI(token)`.
- Sending is stateless: `client.Emails.Send`, `client.Emails.SendBatch` and the value builder `client.Emails.Compose()`. The idempotency key is a per-call option (`WithIdempotencyKey`).
- Each part uses its own token: `client.Emails` the sending token, the Team API the team token. The SDK never falls back to the other token.
- Typed errors for every outcome, including empty or HTML responses, redirects, timeouts and network failures, all usable with `errors.As`.
- `context.Context` first on every call, including `EmailBuilder.Send`.
- Typed query structs (`&ListDomainsQuery{PageSize: 30}`) and `Iterate` methods (`iter.Seq2`) that follow `next_cursor`.
- Types are generated from the current API specification and use its names (see [Type names](#type-names)).
- Go 1.24 or newer. Go 1.21 to 1.23 are end of life; 1.24 adds the `omitzero` JSON tag that keeps "absent" and "empty" apart in requests.

## Upgrade with a coding agent

You can let a coding agent (Claude Code, Codex, Cursor, Copilot, …) do the upgrade. Copy this instruction into the agent from your module's root, then review its changes:

````text
Upgrade this Go module from the Lettermint Go SDK v2 (github.com/lettermint/lettermint-go/v2) to v3 (github.com/lettermint/lettermint-go/v3).

1. Run `go get github.com/lettermint/lettermint-go/v3@latest`. v3 needs Go 1.24 or newer: check the `go` directive in go.mod, CI workflows and Dockerfiles, and report anything older.
2. Read the upgrade guide before changing code: UPGRADE.md in the module cache (`$(go env GOMODCACHE)/github.com/lettermint/lettermint-go/v3@<version>/UPGRADE.md`, find the version with `go list -m github.com/lettermint/lettermint-go/v3`), or https://github.com/lettermint/lettermint-go/blob/main/UPGRADE.md. Treat it as the source of truth and don't guess APIs; when unsure, run `go doc github.com/lettermint/lettermint-go/v3 <Symbol>`.
3. Find every use of the SDK: imports of `github.com/lettermint/lettermint-go/v2`, `lettermint.New(`, `lettermint.NewAPI(`, `.Email(ctx)`, `.SendBatch(`, `SendBatchWithIdempotencyKey`, `.IdempotencyKey(`, `.Attach`, `VerifyWebhook`, `SetWebhookBasicAuth`, `ClearWebhookBasicAuth`, the `Err…` sentinel errors, `APIError` fields, and the v2 type names from the guide's type-name table.
4. Rewrite each use following the guide's before/after examples:
   - Change the import path to `github.com/lettermint/lettermint-go/v3`.
   - Create one client with `lettermint.New(lettermint.WithSendingToken(...))`, adding `lettermint.WithTeamToken(...)` only where the Team API is used. Keep the project's existing environment variable names.
   - Replace `client.Email(ctx)...Send()` with `client.Emails.Compose()...Send(ctx)`, or `client.Emails.Send(ctx, lettermint.SendMailRequest{...})`. Builders are values: when a builder is built over several statements, assign the result of each setter. Never keep a builder in a package variable that is changed later.
   - Move idempotency keys into `lettermint.WithIdempotencyKey(key)` on the `Send`/`SendBatch` call. Attachments become `lettermint.Attachment{Filename, Content (raw bytes) or ContentBase64, ContentType, ContentID}`.
   - Team API: use the same client (`client.Domains`, ...), typed query structs instead of `map[string]string{"page[size]": "10"}`, the renamed methods from the guide, and `Iterate` where code loops over pages.
   - Errors: replace `errors.Is(err, lettermint.ErrX)` with `errors.As` on the v3 error types (`*lettermint.ValidationError`, `*lettermint.RateLimitError`, ...), and the renamed `APIError` fields (`StatusCode` → `Status`, `ErrorType` → `Code`, `ResponseBody` → `Body`).
   - Webhooks: `webhook, err := lettermint.NewWebhook(secret)` once, then `webhook.VerifyRequest(r)` or `webhook.Verify(rawBody, r.Header)`. Keep passing the raw request body, keep the secret's `whsec_` prefix, and make sure the `X-Lettermint-Signature` and `X-Lettermint-Delivery` headers reach the handler.
   - Rename types using the guide's type-name table, and adapt optional fields (`*T`) and optional nullable fields (`lettermint.Nullable[T]`, set with `lettermint.Value(v)` or `lettermint.Null[T]()`).
5. Run `go build ./...`, `go vet ./...` and the tests, and fix every error. Don't send real email or call the live API while testing.
6. Finish with a summary: the files you changed, anything you could not migrate with certainty, and behaviour changes I should review.

Never print, log or commit API tokens or webhook secrets.
````

## Module path

```bash
go get github.com/lettermint/lettermint-go/v3
```

```go
// v2
import lettermint "github.com/lettermint/lettermint-go/v2"

// v3
import lettermint "github.com/lettermint/lettermint-go/v3"
```

## Create the client

`lettermint.New(token, ...)` and `lettermint.NewAPI(token, ...)` are replaced by one client. `APIClient` is removed.

```go
// v2
client, err := lettermint.New(os.Getenv("LETTERMINT_PROJECT_TOKEN"), lettermint.WithTimeout(10*time.Second))
api, err := lettermint.NewAPI(os.Getenv("LETTERMINT_TEAM_TOKEN"))

// v3
client, err := lettermint.New(
	lettermint.WithSendingToken(os.Getenv("LETTERMINT_PROJECT_TOKEN")), // for client.Emails
	lettermint.WithTeamToken(os.Getenv("LETTERMINT_TEAM_TOKEN")),       // for the Team API
	lettermint.WithTimeout(10*time.Second),
)
```

Pass one token or both. With only one token, a method that needs the other returns a `*ConfigError` (for example `Domains.List needs a team token; create the client with lettermint.WithTeamToken`) before any request.

You can also pass one token and let the SDK choose its type by the prefix:

```go
client, err := lettermint.NewFromToken("lm_team_...") // team token
client, err := lettermint.NewFromToken("lm_...")      // project sending token
```

Any other format (SSO tokens, OAuth tokens, an empty string) returns a `*ConfigError`. Use `New` with `WithSendingToken` or `WithTeamToken` for those.

`WithBaseURL`, `WithTimeout` and `WithHTTPClient` work as before, with two differences: `WithTimeout` no longer changes the `http.Client` you pass (the SDK copies it), and the timeout now also covers reading the response body.

## Send an email

```go
// v2
resp, err := client.Email(ctx).
	From("Acme <hello@acme.com>").
	To("jane@example.com").
	Subject("Welcome").
	HTML("<p>Hi Jane</p>").
	IdempotencyKey("welcome-jane").
	Send()

// v3: builder
resp, err := client.Emails.Compose().
	From("Acme <hello@acme.com>").
	To("jane@example.com").
	Subject("Welcome").
	HTML("<p>Hi Jane</p>").
	Send(ctx, lettermint.WithIdempotencyKey("welcome-jane"))

// v3: struct in the API's format
resp, err := client.Emails.Send(ctx, lettermint.SendMailRequest{
	From:    "Acme <hello@acme.com>",
	To:      []string{"jane@example.com"},
	Subject: "Welcome",
	HTML:    lettermint.Value("<p>Hi Jane</p>"),
}, lettermint.WithIdempotencyKey("welcome-jane"))
```

The builder is a value. Chaining works as before. If you built an email over several statements, assign the result of each setter:

```go
// v2: statements changed the shared builder
email := client.Email(ctx)
email.From("hello@acme.com")
email.To("jane@example.com")
if copy {
	email.CC("team@acme.com")
}
email.Subject("Hi").Text("Hi").Send()

// v3: keep the returned builder
email := client.Emails.Compose().From("hello@acme.com").To("jane@example.com")
if copy {
	email = email.CC("team@acme.com")
}
_, err := email.Subject("Hi").Text("Hi").Send(ctx)
```

A base builder can now be shared safely:

```go
welcome := client.Emails.Compose().From("Acme <hello@acme.com>").Subject("Welcome")
welcome.To("jane@example.com").HTML(janeHTML).Send(ctx)
welcome.To("john@example.com").HTML(johnHTML).Send(ctx, lettermint.WithIdempotencyKey("welcome-john"))
```

### Changed builder methods

| v2 | v3 |
| --- | --- |
| `client.Email(ctx)` | `client.Emails.Compose()`; the context moves to `Send(ctx)` |
| Setters change the builder and return the same pointer | Setters return a new `EmailBuilder` value |
| `To`, `CC`, `BCC`, `ReplyTo` append | They replace the list |
| `Header(key, value)` | `Headers(map)` (replaces all headers) |
| `Headers(map)` merged | `Headers(map)` replaces |
| `Metadata(map)` merged, `MetadataValue(key, value)` | `Metadata(map)` replaces |
| `Tags(...map[string]string)`, `MessageTags(...MessageTag)` | `Tags(...MessageTagInput)` |
| `Attach(filename, base64)`, `AttachWithContentID(…)`, `AttachWithContentType(…)`, `AttachWithOptions(filename, base64, AttachmentOptions)` | `Attach(lettermint.Attachment{Filename, Content or ContentBase64, ContentType, ContentID})`; `Content` takes raw bytes |
| `Settings(EmailSettings)` | `Settings(SendMailRequestSettings)` |
| `IdempotencyKey(key).Send()` | `Send(ctx, lettermint.WithIdempotencyKey(key))` |
| `Send() (*SendResponse, error)` | `Send(ctx, options...) (*SendMailResponse, error)`; the builder can be sent again |
| Invalid tags failed in `Send` with `ErrInvalidRequest`, and the builder kept its fields | The setter records a `*ClientValidationError` on the builder it returns (`Err()`); the builder it was called on is unchanged |
| `HTML("")`, `Text("")` sent nothing | Unchanged: an empty string removes the field, also for `Tag`, `Route` and `ScheduledAt` |
| `Send` required `from`, `to`, `subject` and a body before the request | The API validates the message and answers with a `*ValidationError` |
| — | `Route`, `SandboxResult`, `ScheduledAtTime(time.Time)`, `Build()`, `Err()`; `client.Emails.ComposeFrom(message)` |

Unchanged setters: `From`, `Subject`, `HTML`, `Text`, `Tag`, `Route`, `ScheduledAt`.

```go
// v2
email.Attach("invoice.pdf", base64.StdEncoding.EncodeToString(pdf)).
	AttachWithContentID("logo.png", logoBase64, "logo")

// v3
builder.
	Attach(lettermint.Attachment{Filename: "invoice.pdf", Content: pdf, ContentType: "application/pdf"}).
	Attach(lettermint.Attachment{Filename: "logo.png", ContentBase64: logoBase64, ContentID: "logo"})
```

## Batch sending and ping

```go
// v2
resp, err := client.SendBatch(ctx, messages)
resp, err := client.SendBatchWithIdempotencyKey(ctx, messages, "batch-1")
pong, err := client.Ping(ctx)
pong, err := api.Ping(ctx)

// v3
results, err := client.Emails.SendBatch(ctx, messages)
results, err := client.Emails.SendBatch(ctx, messages, lettermint.WithIdempotencyKey("batch-1"))
pong, err := client.Emails.Ping(ctx) // sending token
pong, err := client.Ping(ctx)        // team token if configured, otherwise the sending token
```

`messages` is a `[]lettermint.SendMailRequest` in both versions; include a builder with `builder.Build()`.

## Team API

The services move from `api := lettermint.NewAPI(token)` to the client. Query parameters are typed structs (pass `nil` for none) instead of `map[string]string` with bracket keys. Every list also has an `Iterate` method that follows `next_cursor`.

```go
// v2
api, _ := lettermint.NewAPI(token)
page, err := api.Domains.List(ctx, map[string]string{"page[size]": "10", "filter[status]": "verified"})

// v3
page, err := client.Domains.List(ctx, &lettermint.ListDomainsQuery{PageSize: 10, FilterStatus: lettermint.DomainStatusVerified})
for domain, err := range client.Domains.Iterate(ctx, &lettermint.ListDomainsQuery{FilterStatus: lettermint.DomainStatusVerified}) {
	if err != nil {
		return err
	}
	fmt.Println(domain.Domain)
}
```

Results are pointers to the generated types (`*DomainData`), lists are `*CursorPage[T]` aliases (`*ListDomainsResponse`), and text endpoints return `string`.

| v2 (`api := lettermint.NewAPI(token)`) | v3 (`client := lettermint.New(lettermint.WithTeamToken(token))`) |
| --- | --- |
| `api.Ping(ctx)` | `client.Ping(ctx)` |
| `api.BlockedFileTypes(ctx)` | `client.BlockedFileTypes(ctx)` |
| `api.Analytics(ctx, payload)` | `client.Analytics(ctx, query)` |
| `api.Domains.List(ctx, query)` | `client.Domains.List(ctx, *ListDomainsQuery)`, `client.Domains.Iterate(ctx, *ListDomainsQuery)` |
| `api.Domains.Create(ctx, payload)` | `client.Domains.Create(ctx, StoreDomainData)` |
| `api.Domains.Retrieve(ctx, id)` | `client.Domains.Retrieve(ctx, id, *GetDomainQuery)` (`Include`) |
| `api.Domains.Delete(ctx, id)` | `client.Domains.Delete(ctx, id)` |
| `api.Domains.VerifyDNSRecords(ctx, id)` | `client.Domains.VerifyDNSRecords(ctx, id)` |
| `api.Domains.VerifyDNSRecord(ctx, id, recordID)` | `client.Domains.VerifyDNSRecord(ctx, id, recordID)` |
| `api.Domains.UpdateProjects(ctx, id, payload)` | `client.Domains.UpdateProjects(ctx, id, UpdateDomainProjectsData)` |
| `api.Messages.List(ctx, query)` | `client.Messages.List(ctx, *ListMessagesQuery)`, `client.Messages.Iterate(…)` |
| `api.Messages.Retrieve(ctx, id)` | `client.Messages.Retrieve(ctx, id)` |
| `api.Messages.Events(ctx, id)` | `client.Messages.Events(ctx, id, *ListMessageEventsQuery)`, `client.Messages.IterateEvents(…)` |
| `api.Messages.Source` / `HTML` / `Text(ctx, id)` | unchanged, on `client.Messages` |
| `api.Messages.Reschedule(ctx, id, payload)` | `client.Messages.Reschedule(ctx, id, RescheduleMessageRequest)` |
| `api.Messages.Cancel(ctx, id)` | `client.Messages.Cancel(ctx, id)` |
| `api.Messages.Process(ctx, id)` | `client.Messages.Process(ctx, id, lettermint.WithIdempotencyKey(…)?)` |
| `api.Projects.List(ctx, query)` | `client.Projects.List(ctx, *ListProjectsQuery)`, `client.Projects.Iterate(…)` |
| `api.Projects.Create(ctx, payload)` | `client.Projects.Create(ctx, StoreProjectData)` |
| `api.Projects.Retrieve(ctx, id)` | `client.Projects.Retrieve(ctx, id, *GetProjectQuery)` |
| `api.Projects.Update(ctx, id, payload)` | `client.Projects.Update(ctx, id, UpdateProjectData)` |
| `api.Projects.Delete(ctx, id)` | `client.Projects.Delete(ctx, id)` |
| `api.Projects.RotateToken(ctx, id)` | `client.Projects.RotateToken(ctx, id)` (deprecated by the API) |
| `api.Projects.Routes(ctx, projectID, query)` | `client.Routes.List(ctx, projectID, *ListRoutesQuery)`, `client.Routes.Iterate(…)` |
| `api.Projects.CreateRoute(ctx, projectID, payload)` | `client.Routes.Create(ctx, projectID, StoreRouteData)` |
| `api.Projects.RetrieveReportForwarding(ctx, id)` | `client.Projects.ReportForwarding.Retrieve(ctx, id)` |
| `api.Projects.UpdateReportForwarding(ctx, id, payload)` | `client.Projects.ReportForwarding.Update(ctx, id, ReportForwardingRequest)` |
| `api.Projects.DeleteReportForwarding(ctx, id)` | `client.Projects.ReportForwarding.Delete(ctx, id)` |
| `api.Projects.VerifyReportForwarding(ctx, id, payload)` | `client.Projects.ReportForwarding.Verify(ctx, id, VerifyReportForwardingRequest)` |
| `api.Projects.ResendReportForwardingCode(ctx, id)` | `client.Projects.ReportForwarding.ResendCode(ctx, id)` |
| `api.Routes.Retrieve(ctx, id)` | `client.Routes.Retrieve(ctx, id, *GetRouteQuery)` |
| `api.Routes.Update(ctx, id, payload)` | `client.Routes.Update(ctx, id, UpdateRouteData)` |
| `api.Routes.Delete(ctx, id)` | `client.Routes.Delete(ctx, id)` |
| `api.Routes.VerifyInboundDomain(ctx, id)` | `client.Routes.VerifyInboundDomain(ctx, id)` |
| `api.Stats.Retrieve(ctx, query)` | `client.Stats.Retrieve(ctx, GetStatsQuery{From, To, ProjectID, IncludeMachine})` |
| `api.Suppressions.List(ctx, query)` | `client.Suppressions.List(ctx, *ListSuppressionsQuery)`, `client.Suppressions.Iterate(…)` |
| `api.Suppressions.Create(ctx, payload)` | `client.Suppressions.Create(ctx, StoreSuppressionData)` |
| `api.Suppressions.Delete(ctx, id)` | `client.Suppressions.Delete(ctx, id)` |
| `api.Team.Retrieve(ctx)` | `client.Team.Retrieve(ctx, *GetTeamQuery)` (`Include`) |
| `api.Team.Update(ctx, payload)` | `client.Team.Update(ctx, UpdateTeamData)` |
| `api.Team.Usage(ctx)` | `client.Team.Usage(ctx)` |
| `api.Team.Roles(ctx)` | `client.Team.Roles(ctx)` |
| `api.Team.Members(ctx, query)` | `client.Team.Members.List(ctx, *ListTeamMembersQuery)`, `client.Team.Members.Iterate(…)` |
| `api.Team.Member(ctx, userID)` | `client.Team.Members.Retrieve(ctx, userID)` |
| `api.Team.UpdateMemberAssignment(ctx, userID, payload)` | `client.Team.Members.UpdateAssignment(ctx, userID, UpdateTeamMemberAssignmentData)` |
| `api.Webhooks.List(ctx, query)` | `client.Webhooks.List(ctx, *ListWebhooksQuery)`, `client.Webhooks.Iterate(…)` |
| `api.Webhooks.Create(ctx, payload)` | `client.Webhooks.Create(ctx, StoreWebhookData)` |
| `api.Webhooks.Retrieve(ctx, id)` | `client.Webhooks.Retrieve(ctx, id)` |
| `api.Webhooks.Update(ctx, id, payload)` | `client.Webhooks.Update(ctx, id, UpdateWebhookData)` |
| `api.Webhooks.Delete(ctx, id)` | `client.Webhooks.Delete(ctx, id)` |
| `api.Webhooks.Test(ctx, id)` | `client.Webhooks.Test(ctx, id)` |
| `api.Webhooks.RegenerateSecret(ctx, id)` | `client.Webhooks.RegenerateSecret(ctx, id)` |
| `api.Webhooks.Deliveries(ctx, id, query)` | `client.Webhooks.Deliveries.List(ctx, id, *ListWebhookDeliveriesQuery)`, `client.Webhooks.Deliveries.Iterate(…)` |
| `api.Webhooks.Delivery(ctx, id, deliveryID)` | `client.Webhooks.Deliveries.Retrieve(ctx, id, deliveryID)` |

Every method also takes trailing options: `lettermint.WithRequestTimeout(d)` on every call, and `lettermint.WithIdempotencyKey(key)` on `Emails.Send`, `Emails.SendBatch`, `EmailBuilder.Send` and `Messages.Process`.

`Messages.Reschedule` and `Messages.Cancel` accept either token: the team token when configured, otherwise the sending token. A sending-only client can cancel the scheduled email it sent.

### Query parameters

Write bracketed names as struct fields. Arrays of values are joined with commas, arrays of objects are indexed, and booleans are sent as `1`/`0`. A zero value is not sent.

| v2 | v3 |
| --- | --- |
| `map[string]string{"page[size]": "30", "page[cursor]": c}` | `&ListDomainsQuery{PageSize: 30, PageCursor: c}` |
| `map[string]string{"filter[status]": "verified"}` | `&ListDomainsQuery{FilterStatus: lettermint.DomainStatusVerified}` |
| `map[string]string{"sort": "-created_at,domain"}` | `Sort: []ListDomainsQuerySortItem{ListDomainsQuerySortItemCreatedAtDesc, ListDomainsQuerySortItemDomain}` |
| `map[string]string{"filter[tags][0][name]": "a", "filter[tags][0][value]": "b"}` | `&ListMessagesQuery{Filter: &ListMessagesQueryFilter{Tags: []ListMessagesQueryFilterTagsItem{{Name: "a", Value: "b"}}}}` |
| `map[string]string{"filter[enabled]": "true"}` | `&ListWebhooksQuery{FilterEnabled: lettermint.Ptr(true)}` |
| webhooks: `map[string]string{"cursor": c}` | `&ListWebhooksQuery{Cursor: c}` (these lists use `cursor`, not `page[cursor]`) |

### Path parameters

IDs are still URL-encoded. An empty ID, `.` or `..` now returns a `*ConfigError` before the request.

### Message lists

The v2 types described message and event lists with a nested `meta` object. The API returns a flat cursor page, which v3 types as `CursorPage[T]`: read `page.NextCursor`, not `page.Meta.NextCursor`. Or use `Iterate`.

## Request and response fields

The generated structs follow one rule for optional and nullable properties (see the README's [Types](README.md#types)):

| Property | v2 | v3 |
| --- | --- | --- |
| optional | `T` with `omitempty`, or `*T` for booleans | `*T` with `omitzero`; slices and maps stay `[]T`/`map` and are sent as `[]` when empty but not nil |
| optional and nullable | `*T`, or `**T` for `basic_auth` | `Nullable[T]`: `lettermint.Value(v)`, `lettermint.Null[T]()` or absent (zero value). Read it with `Get()`, `IsNull()` and `IsSet()` |
| required and nullable | `*T` | `*T` (unchanged) |
| untyped objects | `map[string]interface{}` | named structs (`SendMailRequestSettings`, `MessageTagInput`, `MessageAttachmentInput`, `DomainDataProjectsItem`, `RouteDataSettings`, `TeamMemberDataRole`, …) |

```go
// v2
api.Webhooks.Update(ctx, id, lettermint.WebhookUpdateRequest{BasicAuth: lettermint.ClearWebhookBasicAuth()})
api.Webhooks.Update(ctx, id, lettermint.WebhookUpdateRequest{BasicAuth: lettermint.SetWebhookBasicAuth("user", "pass")})
api.Projects.Update(ctx, id, lettermint.ProjectUpdateRequest{Name: &name})

// v3
client.Webhooks.Update(ctx, id, lettermint.UpdateWebhookData{BasicAuth: lettermint.Null[lettermint.WebhookBasicAuthData]()})
client.Webhooks.Update(ctx, id, lettermint.UpdateWebhookData{BasicAuth: lettermint.Value(lettermint.WebhookBasicAuthData{Username: "user", Password: "pass"})})
client.Projects.Update(ctx, id, lettermint.UpdateProjectData{Name: lettermint.Value(name)})
```

Renamed fields: `Cc` → `CC`, `Bcc` → `BCC` (in `SendMailRequest`, `MessageData` and `MessageListData`), `Tls` → `TLS`, `HttpStatusCode` → `HTTPStatusCode`, `TokenLastUsedIp` → `TokenLastUsedIP`, `InboundMxHostname` → `InboundMXHostname`.

`StoreSuppressionData.Reason` and `.Scope` are `SuppressionCreateReason` and `SuppressionCreateScope` (the values a request may use).

## Errors

The sentinel errors (`ErrInvalidAPIToken`, `ErrInvalidRequest`, `ErrUnauthorized`, `ErrValidation`, `ErrRateLimited`, `ErrServerError`, `ErrTimeout`, `ErrInvalidWebhookSignature`, `ErrWebhookTimestampExpired`) are removed. Use `errors.As` with the error types. Every SDK error implements `lettermint.Error`.

| Situation | v2 | v3 |
| --- | --- | --- |
| HTTP 400 | `*APIError`, `errors.Is(err, ErrInvalidRequest)` | `*APIError` |
| HTTP 401 | `*APIError`, `ErrUnauthorized` | `*AuthenticationError` |
| HTTP 403 | `*APIError` | `*PermissionError` |
| HTTP 404 | `*APIError` | `*NotFoundError` |
| HTTP 409 | `*APIError` | `*ConflictError` |
| HTTP 422 | `*APIError`, `ErrValidation` | `*ValidationError` (`Errors`) |
| HTTP 429 | `*APIError`, `ErrRateLimited` | `*RateLimitError` (`RetryAfter`) |
| HTTP 5xx | `*APIError`, `ErrServerError` | `*ServerError` |
| Other 4xx | `*APIError` | `*APIError` |
| Empty or invalid JSON body, HTML error page | `failed to parse response: …` or an `*APIError` with the HTML as message | `*UnexpectedResponseError` (`Status`, `BodyExcerpt`) |
| Redirect (3xx) | followed, with the token | `*RedirectError` (`Status`); never followed |
| Timeout | `ErrTimeout` wrapped in a plain error | `*TimeoutError` (`Timeout`); covers headers and body |
| Network failure | `request failed: …` | `*ConnectionError` (`Err`) |
| Cancelled context | `request canceled: …` | the context's error (`context.Canceled`, or the cause) |
| Missing or invalid token | `ErrInvalidAPIToken` | `*ConfigError` |
| Invalid tags or message before the request | `ErrInvalidRequest` | `*ClientValidationError` (`Field`) |
| Webhook verification | `ErrInvalidWebhookSignature`, `ErrWebhookTimestampExpired` | `*WebhookVerificationError` (`Reason`) |

The status types embed `APIError` and unwrap to it, so `errors.As(err, &apiErr)` with `var apiErr *lettermint.APIError` matches all of them.

`APIError` field renames: `StatusCode` → `Status`, `ErrorType` → `Code`, `ResponseBody` (string) → `Body` (`json.RawMessage`; decode it into `ApiErrorBody` or `ValidationErrorBody`), `Errors` → `ValidationError.Errors`. `Code` comes from `{"error": {"code"}}`, or from a string `error` field; `Details` is new.

```go
// v2
if errors.Is(err, lettermint.ErrRateLimited) {
	retryLater()
}
var apiErr *lettermint.APIError
if errors.As(err, &apiErr) {
	log.Println(apiErr.StatusCode, apiErr.ErrorType, apiErr.Errors)
}

// v3
var rateLimit *lettermint.RateLimitError
if errors.As(err, &rateLimit) {
	retryLater(rateLimit.RetryAfter)
}
var validation *lettermint.ValidationError
if errors.As(err, &validation) {
	log.Println(validation.Status, validation.Code, validation.Errors)
}
```

The SDK does not retry requests. Pass `WithIdempotencyKey` when you retry a send.

## Webhooks

The functions are replaced by a verifier that you create once:

```go
// v2
event, err := lettermint.VerifyWebhookFromRequest(r, secret, lettermint.DefaultWebhookTolerance)
event, err := lettermint.VerifyWebhookFromRequestWithMaxBodyBytes(r, secret, tolerance, 10<<20)
event, err := lettermint.VerifyWebhook(signatureHeader, body, deliveryTimestamp, secret, tolerance)

// v3
webhook, err := lettermint.NewWebhook(secret) // lettermint.WithTolerance(d), lettermint.WithMaxBodyBytes(n)
event, err := webhook.VerifyRequest(r)
event, err := webhook.Verify(body, r.Header)
event, err := webhook.VerifySignature(body, signatureHeader, deliveryHeader)
```

- `X-Lettermint-Delivery` is required by `Verify` and `VerifyRequest`, and must equal the signed timestamp. `VerifySignature` checks it when it is not empty.
- Any matching `v1` signature is valid (key rotation), and uppercase hex is accepted.
- `VerifyRequest` puts the body back on `r.Body`.
- Every failure is a `*WebhookVerificationError` with a `Reason`: `signature_header_missing`, `signature_header_malformed`, `delivery_header_missing`, `delivery_timestamp_mismatch`, `timestamp_out_of_tolerance`, `signature_mismatch`, `body_invalid` or `payload_invalid`. An empty secret or a negative tolerance is a `*ConfigError`.
- The result is a `*WebhookPayload` (was `*WebhookEvent`): `ID`, `Event` (a `WebhookEvent`), `Timestamp` (an ISO 8601 string, was `time.Time`), `Data` (raw JSON, decode it with `DecodeData`; was the fixed `WebhookEventData` struct) and `Raw` (was `RawPayload`).
- `WebhookEvent` is now the open enum of event names (v2's `APIWebhookEvent`). The constants are `WebhookEventMessageDelivered` and so on (v2: `APIWebhookEventMessageDelivered`).

```go
// v2
switch event.Event {
case "message.delivered":
	log.Println(event.Data.Recipient)
}

// v3
switch event.Event {
case lettermint.WebhookEventMessageDelivered:
	var data struct {
		Recipient string `json:"recipient"`
	}
	_ = event.DecodeData(&data)
	log.Println(data.Recipient)
}
```

## Type names

The types are generated from the API specification of lettermint#2582 and use its names. Types that are not listed below keep their name. Some shapes also changed:

- Enums stay open string types; give `switch` statements a `default` branch.
- `SendMailResponse` has `MessageID`, `Status`, `ScheduledAt`, `Sandbox` and `SandboxResult`.
- `MessageTag` describes tags in responses; `MessageTagInput` describes tags you send.
- Message and event lists are `CursorPage[T]` (see [Message lists](#message-lists)).
- The API's error bodies are `ApiErrorBody` and `ValidationErrorBody`, because `APIError` and `ValidationError` are the SDK's error types.

| v2 | v3 |
| --- | --- |
| `APIWebhookEvent` | `WebhookEvent` |
| `AnalyticsRequest` | `AnalyticsQuery` |
| `AnalyticsRequestFiltersItem` | `AnalyticsFilter` |
| `AnalyticsRequestSort` | `AnalyticsSort` |
| `AnalyticsResponseMeta` | `AnalyticsMeta` |
| `AnalyticsResponseMetaComparison` | `AnalyticsMetaComparison` |
| `AnalyticsResponsePagination` | `AnalyticsPagination` |
| `AnalyticsResponsePayload` | `AnalyticsResults` |
| `AnalyticsResponsePayloadBreakdownItem` | `AnalyticsBreakdownRow` |
| `AnalyticsResponsePayloadBreakdownItemMetrics`, `AnalyticsResponsePayloadBreakdownItemPreviousMetrics`, `AnalyticsResponsePayloadBreakdownItemTrendItemMetrics`, `AnalyticsResponsePayloadBreakdownItemTrendItemPreviousMetrics`, `AnalyticsResponsePayloadSummaryMetrics`, `AnalyticsResponsePayloadSummaryPreviousMetrics`, `AnalyticsResponsePayloadTimeSeriesItemMetrics`, `AnalyticsResponsePayloadTimeSeriesItemPreviousMetrics` | `AnalyticsMetricValues` |
| `AnalyticsResponsePayloadBreakdownItemPrevious`, `AnalyticsResponsePayloadBreakdownItemTrendItemPrevious`, `AnalyticsResponsePayloadSummaryPrevious`, `AnalyticsResponsePayloadTimeSeriesItemPrevious` | `AnalyticsComparisonValues` |
| `AnalyticsResponsePayloadBreakdownItemPreviousRateBases`, `AnalyticsResponsePayloadBreakdownItemRateBases`, `AnalyticsResponsePayloadBreakdownItemTrendItemPreviousRateBases`, `AnalyticsResponsePayloadBreakdownItemTrendItemRateBases`, `AnalyticsResponsePayloadSummaryPreviousRateBases`, `AnalyticsResponsePayloadSummaryRateBases`, `AnalyticsResponsePayloadTimeSeriesItemPreviousRateBases`, `AnalyticsResponsePayloadTimeSeriesItemRateBases` | `AnalyticsRateBases` |
| `AnalyticsResponsePayloadBreakdownItemPreviousRateBasesBounceRate`, `AnalyticsResponsePayloadBreakdownItemPreviousRateBasesComplaintRate`, `AnalyticsResponsePayloadBreakdownItemPreviousRateBasesDeferralRate`, `AnalyticsResponsePayloadBreakdownItemPreviousRateBasesDeliveryRate`, `AnalyticsResponsePayloadBreakdownItemPreviousRateBasesEffectiveDeliveryRate`, `AnalyticsResponsePayloadBreakdownItemPreviousRateBasesHumanClickRate`, `AnalyticsResponsePayloadBreakdownItemPreviousRateBasesHumanOpenRate`, `AnalyticsResponsePayloadBreakdownItemRateBasesBounceRate`, `AnalyticsResponsePayloadBreakdownItemRateBasesComplaintRate`, `AnalyticsResponsePayloadBreakdownItemRateBasesDeferralRate`, `AnalyticsResponsePayloadBreakdownItemRateBasesDeliveryRate`, `AnalyticsResponsePayloadBreakdownItemRateBasesEffectiveDeliveryRate`, `AnalyticsResponsePayloadBreakdownItemRateBasesHumanClickRate`, `AnalyticsResponsePayloadBreakdownItemRateBasesHumanOpenRate`, `AnalyticsResponsePayloadBreakdownItemTrendItemPreviousRateBasesBounceRate`, `AnalyticsResponsePayloadBreakdownItemTrendItemPreviousRateBasesComplaintRate`, `AnalyticsResponsePayloadBreakdownItemTrendItemPreviousRateBasesDeferralRate`, `AnalyticsResponsePayloadBreakdownItemTrendItemPreviousRateBasesDeliveryRate`, `AnalyticsResponsePayloadBreakdownItemTrendItemPreviousRateBasesEffectiveDeliveryRate`, `AnalyticsResponsePayloadBreakdownItemTrendItemPreviousRateBasesHumanClickRate`, `AnalyticsResponsePayloadBreakdownItemTrendItemPreviousRateBasesHumanOpenRate`, `AnalyticsResponsePayloadBreakdownItemTrendItemRateBasesBounceRate`, `AnalyticsResponsePayloadBreakdownItemTrendItemRateBasesComplaintRate`, `AnalyticsResponsePayloadBreakdownItemTrendItemRateBasesDeferralRate`, `AnalyticsResponsePayloadBreakdownItemTrendItemRateBasesDeliveryRate`, `AnalyticsResponsePayloadBreakdownItemTrendItemRateBasesEffectiveDeliveryRate`, `AnalyticsResponsePayloadBreakdownItemTrendItemRateBasesHumanClickRate`, `AnalyticsResponsePayloadBreakdownItemTrendItemRateBasesHumanOpenRate`, `AnalyticsResponsePayloadSummaryPreviousRateBasesBounceRate`, `AnalyticsResponsePayloadSummaryPreviousRateBasesComplaintRate`, `AnalyticsResponsePayloadSummaryPreviousRateBasesDeferralRate`, `AnalyticsResponsePayloadSummaryPreviousRateBasesDeliveryRate`, `AnalyticsResponsePayloadSummaryPreviousRateBasesEffectiveDeliveryRate`, `AnalyticsResponsePayloadSummaryPreviousRateBasesHumanClickRate`, `AnalyticsResponsePayloadSummaryPreviousRateBasesHumanOpenRate`, `AnalyticsResponsePayloadSummaryRateBasesBounceRate`, `AnalyticsResponsePayloadSummaryRateBasesComplaintRate`, `AnalyticsResponsePayloadSummaryRateBasesDeferralRate`, `AnalyticsResponsePayloadSummaryRateBasesDeliveryRate`, `AnalyticsResponsePayloadSummaryRateBasesEffectiveDeliveryRate`, `AnalyticsResponsePayloadSummaryRateBasesHumanClickRate`, `AnalyticsResponsePayloadSummaryRateBasesHumanOpenRate`, `AnalyticsResponsePayloadTimeSeriesItemPreviousRateBasesBounceRate`, `AnalyticsResponsePayloadTimeSeriesItemPreviousRateBasesComplaintRate`, `AnalyticsResponsePayloadTimeSeriesItemPreviousRateBasesDeferralRate`, `AnalyticsResponsePayloadTimeSeriesItemPreviousRateBasesDeliveryRate`, `AnalyticsResponsePayloadTimeSeriesItemPreviousRateBasesEffectiveDeliveryRate`, `AnalyticsResponsePayloadTimeSeriesItemPreviousRateBasesHumanClickRate`, `AnalyticsResponsePayloadTimeSeriesItemPreviousRateBasesHumanOpenRate`, `AnalyticsResponsePayloadTimeSeriesItemRateBasesBounceRate`, `AnalyticsResponsePayloadTimeSeriesItemRateBasesComplaintRate`, `AnalyticsResponsePayloadTimeSeriesItemRateBasesDeferralRate`, `AnalyticsResponsePayloadTimeSeriesItemRateBasesDeliveryRate`, `AnalyticsResponsePayloadTimeSeriesItemRateBasesEffectiveDeliveryRate`, `AnalyticsResponsePayloadTimeSeriesItemRateBasesHumanClickRate`, `AnalyticsResponsePayloadTimeSeriesItemRateBasesHumanOpenRate` | `AnalyticsRateBase` |
| `AnalyticsResponsePayloadBreakdownItemTrendItem`, `AnalyticsResponsePayloadTimeSeriesItem` | `AnalyticsTimeSeriesPoint` |
| `AnalyticsResponsePayloadSummary` | `AnalyticsSummary` |
| `BlockedFileTypesResponse` | `BlockedFileTypes` |
| `CancelScheduledMessageResponse`, `MessageScheduleResponse`, `RescheduleMessageResponse` | `ScheduledMessage` |
| `CursorPaginator` | `CursorPage` |
| `DomainDestroyResponse`, `DomainVerifySpecificDNSRecordResponse`, `ProjectDestroyResponse`, `RouteDestroyResponse`, `WebhookDestroyResponse` | `MessageResponse` |
| `DomainIndexResponse` | `ListDomainsResponse` |
| `DomainShowResponse`, `DomainStoreResponse` | `DomainData` |
| `DomainStoreRequest` | `StoreDomainData` |
| `DomainUpdateProjectsRequest` | `UpdateDomainProjectsData` |
| `DomainUpdateProjectsResponse` | `DomainMutationResponse` |
| `DomainVerifyDNSRecordsResponse` | `DnsVerificationSuccessResponse` |
| `EmailPayload` | `SendMailRequest` |
| `MessageEventsResponse` | `ListMessageEventsResponse` |
| `MessageIndexResponse` | `ListMessagesResponse` |
| `MessageShowResponse` | `MessageData` |
| `ProjectIndexResponse` | `ListProjectsResponse` |
| `ProjectRotateTokenResponse` | `RotateProjectTokenResponse` |
| `ProjectShowResponse` | `ProjectData` |
| `ProjectStoreRequest` | `StoreProjectData` |
| `ProjectStoreResponse` | `ProjectCreatedData` |
| `ProjectUpdateRequest` | `UpdateProjectData` |
| `ProjectUpdateResponse` | `ProjectMutationResponse` |
| `RouteIndexResponse` | `ListRoutesResponse` |
| `RouteShowResponse` | `RouteData` |
| `RouteStoreRequest` | `StoreRouteData` |
| `RouteStoreResponse`, `RouteUpdateResponse` | `RouteMutationResponse` |
| `RouteUpdateRequest` | `UpdateRouteData` |
| `RouteVerifyInboundDomainResponse` | `InboundDomainVerificationResponse` |
| `SendBatchEmailResponse` | `SendBatchMailResponse` |
| `SendEmailResponse` | `SendMailResponse` |
| `StatsIndexResponse` | `StatsData` |
| `SuppressionDestroyResponse` | `DeleteSuppressionResponse` |
| `SuppressionIndexResponse` | `ListSuppressionsResponse` |
| `SuppressionStoreRequest` | `StoreSuppressionData` |
| `TeamMembersAssignmentUpdateRequest` | `UpdateTeamMemberAssignmentData` |
| `TeamMembersAssignmentUpdateResponse`, `TeamMembersShowResponse` | `TeamMemberData` |
| `TeamMembersResponse` | `ListTeamMembersResponse` |
| `TeamRolesResponse` | `TeamRoleListResponse` |
| `TeamShowResponse` | `TeamData` |
| `TeamUpdateRequest` | `UpdateTeamData` |
| `TeamUpdateResponse` | `TeamMutationResponse` |
| `TeamUsageResponse` | `TeamUsageDetailData` |
| `UpdateReportForwardingRequest` | `ReportForwardingRequest` |
| `WebhookDeliveriesResponse` | `ListWebhookDeliveriesResponse` |
| `WebhookIndexResponse` | `ListWebhooksResponse` |
| `WebhookRegenerateSecretResponse`, `WebhookStoreResponse` | `WebhookSecretResponse` |
| `WebhookShowDeliveryResponse` | `WebhookDeliveryData` |
| `WebhookShowResponse` | `WebhookData` |
| `WebhookStoreRequest` | `StoreWebhookData` |
| `WebhookTestResponse` | `TestWebhookResponse` |
| `WebhookUpdateRequest` | `UpdateWebhookData` |
| `WebhookUpdateResponse` | `WebhookMutationResponse` |
| `PingResponse` | Removed; `Ping` returns a `string` |

### Removed types

lettermint#2582 removed these schemas from the API specification:

| v2 | v3 |
| --- | --- |
| `AnalyticsResponseData` | Removed. Use `AnalyticsResponse` (`Data` is `AnalyticsResults`). |
| `StatsRequestData` | Removed. Use `GetStatsQuery`, the parameters of `Stats.Retrieve`. |
| Message list `meta` (`MessageIndexResponseMeta`) | Removed; not exported by v2. Lists are flat `CursorPage[T]`. |
| Message events `meta` (`MessageEventsResponseMeta`) | Removed; not exported by v2. |
| `SuppressionStoreResponseMessage1` | Removed; not exported by v2. `SuppressionStoreResponse.Message` is a `string`. |

These types are removed from the SDK:

| v2 | v3 |
| --- | --- |
| `PingResponse` | Removed. `Ping` returns a `string`. |
| `APIClient`, `NewAPI` | `Client` from `lettermint.New(lettermint.WithTeamToken(…))` |
| `Client.Email(ctx)` | `client.Emails.Compose()` |
| `Client.SendBatch`, `Client.SendBatchWithIdempotencyKey` | `client.Emails.SendBatch(ctx, messages, options...)` |
| `Client.Ping` (sending token) | `client.Emails.Ping(ctx)`; `client.Ping(ctx)` uses the team token when set |
| `SendResponse` | `SendMailResponse` |
| `EmailSettings` | `SendMailRequestSettings` |
| `AttachmentOptions` | `Attachment` (`ContentType`, `ContentID`) |
| `MessageTag` as a request type | `MessageTagInput` (`MessageTag` is the response type) |
| `WebhookEvent` (the verified payload struct) | `WebhookPayload` |
| `WebhookEventData`, `WebhookResponse` | Removed. Decode `WebhookPayload.Data` with `DecodeData`. |
| `VerifyWebhook`, `VerifyWebhookFromRequest`, `VerifyWebhookFromRequestWithMaxBodyBytes` | `NewWebhook(secret)` and its `Verify`, `VerifyRequest`, `VerifySignature` |
| `SetWebhookBasicAuth(user, pass)`, `ClearWebhookBasicAuth()` | `lettermint.Value(lettermint.WebhookBasicAuthData{…})`, `lettermint.Null[lettermint.WebhookBasicAuthData]()` |
| `ErrInvalidAPIToken`, `ErrInvalidRequest`, `ErrUnauthorized`, `ErrValidation`, `ErrRateLimited`, `ErrServerError`, `ErrTimeout`, `ErrInvalidWebhookSignature`, `ErrWebhookTimestampExpired` | The error types in [Errors](#errors) |
| `APIError.Unwrap` (to a sentinel) | Removed; the status types unwrap to `*APIError` |
| `EmailBuilder` methods `Header`, `MetadataValue`, `MessageTags`, `IdempotencyKey`, `AttachWithContentID`, `AttachWithContentType`, `AttachWithOptions` | See [Changed builder methods](#changed-builder-methods) |
| `DomainsService`, `MessagesService`, `ProjectsService`, `RoutesService`, `StatsService`, `SuppressionsService`, `TeamService`, `WebhooksService` methods `Projects.Routes`, `Projects.CreateRoute`, `Projects.*ReportForwarding*`, `Team.Members`, `Team.Member`, `Team.UpdateMemberAssignment`, `Webhooks.Deliveries`, `Webhooks.Delivery` | See the [Team API](#team-api) table. New: `ReportForwardingService`, `TeamMembersService`, `WebhookDeliveriesService`, `EmailsService` |
| `New(apiToken string, opts ...Option)` | `New(opts ...Option)` with `WithSendingToken` and/or `WithTeamToken`, or `NewFromToken(token, opts...)` |
| `const Version` | `func Version()` |

Unchanged: `DefaultBaseURL`, `DefaultTimeout`, `DefaultWebhookTolerance`, `DefaultWebhookMaxBodyBytes`, `HeaderSignature`, `HeaderDelivery`, `WithBaseURL`, `WithTimeout`, `WithHTTPClient`, and every enum constant except the `APIWebhookEvent…` constants.

# Upgrade to v2

This guide covers upgrading from the latest released v1 Go SDK to v2.

## Highlights

- The module path is `github.com/lettermint/lettermint-go/v2`.
- Sending email continues to use `lettermint.New(token)`.
- The full Lettermint API is available through `lettermint.NewAPI(token)`.
- Sending tokens use `x-lettermint-token`; full API tokens use `Authorization: Bearer`.
- `Ping` returns the raw trimmed `pong` response.
- API request and response structs are generated from the OpenAPI specs.

## Module path

```bash
go get github.com/lettermint/lettermint-go/v2
```

```go
import lettermint "github.com/lettermint/lettermint-go/v2"
```

## Sending

```go
client, err := lettermint.New("sending-token")
if err != nil {
    return err
}

pong, err := client.Ping(ctx)
```

## Full API

```go
api, err := lettermint.NewAPI("api-token")
if err != nil {
    return err
}

domains, err := api.Domains.List(ctx, nil)
```

## Batch Sending

```go
_, err = client.SendBatch(ctx, []lettermint.SendMailRequest{{
    From: "sender@example.com",
    To: []string{"user@example.com"},
    Subject: "Hello",
    Text: "Hi",
}})
```

