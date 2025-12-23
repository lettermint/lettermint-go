// Package lettermint provides the official Go SDK for the Lettermint email API.
//
// # Getting Started
//
// Create a client with your API token:
//
//	client, err := lettermint.New("your-api-token")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// # Sending Emails
//
// Use the fluent builder interface to compose and send emails:
//
//	ctx := context.Background()
//	resp, err := client.Email(ctx).
//	    From("sender@example.com").
//	    To("recipient@example.com").
//	    Subject("Hello from Lettermint").
//	    HTML("<p>Hello World</p>").
//	    Send()
//
// The builder supports all email features including CC, BCC, attachments,
// metadata, tags, and idempotency keys:
//
//	resp, err := client.Email(ctx).
//	    From("John Doe <john@example.com>").
//	    To("user@example.com").
//	    CC("manager@example.com").
//	    Subject("Monthly Report").
//	    HTML("<h1>Report</h1>").
//	    Text("Report (plain text)").
//	    Attach("report.pdf", base64Content).
//	    MetadataValue("user_id", "12345").
//	    Tag("monthly-report").
//	    IdempotencyKey("report-dec-2024").
//	    Send()
//
// # Client Configuration
//
// Use functional options to customize the client:
//
//	client, err := lettermint.New("your-api-token",
//	    lettermint.WithTimeout(60*time.Second),
//	    lettermint.WithBaseURL("https://custom-api.example.com"),
//	)
//
// # Error Handling
//
// The SDK provides structured errors for easy handling:
//
//	resp, err := client.Email(ctx).From("...").To("...").Subject("...").HTML("...").Send()
//	if err != nil {
//	    // Check for specific error types
//	    var apiErr *lettermint.APIError
//	    if errors.As(err, &apiErr) {
//	        fmt.Printf("API error %d: %s\n", apiErr.StatusCode, apiErr.Message)
//	    }
//
//	    // Check for error categories
//	    if errors.Is(err, lettermint.ErrValidation) {
//	        // Handle validation errors
//	    }
//	}
//
// # Webhook Verification
//
// Verify webhook signatures to ensure authenticity:
//
//	func webhookHandler(w http.ResponseWriter, r *http.Request) {
//	    event, err := lettermint.VerifyWebhookFromRequest(
//	        r,
//	        "your-webhook-secret",
//	        lettermint.DefaultWebhookTolerance,
//	    )
//	    if err != nil {
//	        http.Error(w, "Invalid signature", http.StatusUnauthorized)
//	        return
//	    }
//
//	    // Process event
//	    switch event.Event {
//	    case "message.delivered":
//	        // Handle delivery
//	    case "message.bounced":
//	        // Handle bounce
//	    }
//	}
//
// For more information, visit https://docs.lettermint.co
package lettermint
