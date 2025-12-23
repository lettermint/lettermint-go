package lettermint_test

import (
	"context"
	"fmt"
	"log"
	"net/http"

	lettermint "github.com/lettermint/lettermint-go"
)

func ExampleNew() {
	// Create a new Lettermint client
	client, err := lettermint.New("your-api-token")
	if err != nil {
		log.Fatal(err)
	}
	_ = client // Use client...
}

func ExampleNew_withOptions() {
	// Create a client with custom options
	client, err := lettermint.New("your-api-token",
		lettermint.WithBaseURL("https://api.lettermint.co/v1"),
		lettermint.WithTimeout(60_000_000_000), // 60 seconds in nanoseconds
	)
	if err != nil {
		log.Fatal(err)
	}
	_ = client // Use client...
}

func ExampleClient_Email() {
	client, err := lettermint.New("your-api-token")
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	resp, err := client.Email(ctx).
		From("sender@example.com").
		To("recipient@example.com").
		Subject("Hello from Lettermint").
		Text("This is a test email.").
		Send()

	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Email sent with ID: %s\n", resp.MessageID)
}

func ExampleClient_Email_advanced() {
	client, err := lettermint.New("your-api-token")
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	resp, err := client.Email(ctx).
		From("John Doe <john@example.com>").
		To("recipient@example.com").
		CC("manager@example.com").
		BCC("archive@example.com").
		ReplyTo("support@example.com").
		Subject("Monthly Report").
		HTML("<h1>Report</h1><p>Content here</p>").
		Text("Report\n\nContent here").
		Header("X-Campaign-ID", "dec-2024").
		Metadata(map[string]string{
			"user_id":     "12345",
			"campaign_id": "dec-2024",
		}).
		Tag("monthly-report").
		IdempotencyKey("report-dec-2024").
		Send()

	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Email sent with ID: %s, Status: %s\n", resp.MessageID, resp.Status)
}

func ExampleVerifyWebhookFromRequest() {
	http.HandleFunc("/webhook", func(w http.ResponseWriter, r *http.Request) {
		event, err := lettermint.VerifyWebhookFromRequest(
			r,
			"your-webhook-secret",
			lettermint.DefaultWebhookTolerance,
		)
		if err != nil {
			http.Error(w, "Invalid signature", http.StatusUnauthorized)
			return
		}

		// Process the event
		switch event.Event {
		case "message.delivered":
			fmt.Printf("Email delivered to %s\n", event.Data.Recipient)
		case "message.hard_bounced":
			fmt.Printf("Hard bounce for %s\n", event.Data.Recipient)
		}

		w.WriteHeader(http.StatusOK)
	})
}
