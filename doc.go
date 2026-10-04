// Package lettermint is the official Go SDK for Lettermint (https://lettermint.co).
//
// Create one client and share it:
//
//	client, err := lettermint.New(
//		lettermint.WithSendingToken(os.Getenv("LETTERMINT_PROJECT_TOKEN")), // for client.Emails
//		lettermint.WithTeamToken(os.Getenv("LETTERMINT_TEAM_TOKEN")),       // for the Team API
//	)
//
// NewFromToken creates a client from one token and detects its type by the
// prefix (lm_team_ or lm_).
//
// # Sending email
//
// client.Emails.Compose returns an EmailBuilder. The builder is a value: each
// setter returns a new builder, so a builder can be reused as a template.
//
//	resp, err := client.Emails.Compose().
//		From("Acme <hello@acme.com>").
//		To("jane@example.com").
//		Subject("Welcome").
//		HTML("<p>Thanks for signing up.</p>").
//		Send(ctx, lettermint.WithIdempotencyKey("welcome-jane"))
//
// client.Emails.Send and client.Emails.SendBatch take messages in the API's
// format (SendMailRequest).
//
// # Team API
//
// The other parts of the client (Domains, Messages, Projects, Routes, Stats,
// Suppressions, Team, Webhooks) use the team token. Lists return one
// CursorPage; Iterate methods follow next_cursor:
//
//	for domain, err := range client.Domains.Iterate(ctx, nil) {
//		if err != nil {
//			return err
//		}
//		fmt.Println(domain.Domain)
//	}
//
// # Errors
//
// Every SDK error implements Error. Check concrete types with errors.As:
// *APIError (and *AuthenticationError, *PermissionError, *NotFoundError,
// *ConflictError, *ValidationError, *RateLimitError, *ServerError, which
// unwrap to it), *TimeoutError, *ConnectionError, *UnexpectedResponseError,
// *RedirectError, *ConfigError, *ClientValidationError and
// *WebhookVerificationError. A cancelled or expired context returns the
// context's error.
//
// # Webhooks
//
//	webhook, err := lettermint.NewWebhook(os.Getenv("LETTERMINT_WEBHOOK_SECRET"))
//	event, err := webhook.VerifyRequest(r) // or webhook.Verify(rawBody, r.Header)
//
// # Generated code
//
// generated_types.go and generated_operations.go are generated from the
// Lettermint API specification by the SDK generator; do not edit them.
// Optional fields are pointers, optional fields that may also be null are
// Nullable values (Value, Null), and enums are open string types.
package lettermint

//go:generate sh scripts/generate.sh
