// Example: Webhook verification with Lettermint
//
// This example demonstrates how to verify and handle webhooks from Lettermint.
// It starts an HTTP server that listens for webhook events.
//
// Usage:
//
//	export LETTERMINT_WEBHOOK_SECRET="your-webhook-secret"
//	go run main.go
//
// Then configure your Lettermint webhook to point to http://your-server:8080/webhook
package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"

	"go.lettermint.co/sdk"
)

func main() {
	webhookSecret := os.Getenv("LETTERMINT_WEBHOOK_SECRET")
	if webhookSecret == "" {
		log.Fatal("LETTERMINT_WEBHOOK_SECRET environment variable is required")
	}

	// Create webhook handler
	http.HandleFunc("/webhook", webhookHandler(webhookSecret))

	// Health check endpoint
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	addr := ":8080"
	log.Printf("Starting webhook server on %s", addr)
	log.Printf("Webhook endpoint: http://localhost%s/webhook", addr)

	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

// webhookHandler returns an HTTP handler function for processing webhooks.
func webhookHandler(webhookSecret string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Only accept POST requests
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Verify the webhook signature
		event, err := lettermint.VerifyWebhookFromRequest(r, webhookSecret, lettermint.DefaultWebhookTolerance)
		if err != nil {
			log.Printf("Webhook verification failed: %v", err)

			// Return appropriate status codes based on error type
			if errors.Is(err, lettermint.ErrInvalidWebhookSignature) {
				http.Error(w, "Invalid signature", http.StatusUnauthorized)
				return
			}
			if errors.Is(err, lettermint.ErrWebhookTimestampExpired) {
				http.Error(w, "Webhook expired", http.StatusUnauthorized)
				return
			}

			http.Error(w, "Verification failed", http.StatusBadRequest)
			return
		}

		// Process the webhook event
		log.Printf("Received webhook event: %s", event.Event)
		log.Printf("Message ID: %s", event.Data.MessageID)
		log.Printf("Recipient: %s", event.Data.Recipient)

		// Handle different event types
		switch event.Event {
		case "message.delivered":
			handleDelivered(event)
		case "message.hard_bounced":
			handleHardBounce(event)
		case "message.soft_bounced":
			handleSoftBounce(event)
		case "message.opened":
			handleOpened(event)
		case "message.clicked":
			handleClicked(event)
		case "message.complained":
			handleComplained(event)
		default:
			log.Printf("Unknown event type: %s", event.Event)
		}

		// Respond with 200 OK to acknowledge receipt
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}
}

func handleDelivered(event *lettermint.WebhookEvent) {
	log.Printf("Email delivered to %s", event.Data.Recipient)

	// Example: Update delivery status in your database
	// db.UpdateEmailStatus(event.Data.MessageID, "delivered")
}

func handleHardBounce(event *lettermint.WebhookEvent) {
	log.Printf("Hard bounce for %s", event.Data.Recipient)

	// Example: Mark email as invalid in your database
	// db.MarkEmailAsInvalid(event.Data.Recipient)

	if event.Data.Response != nil {
		log.Printf("Bounce response: %d - %s", event.Data.Response.StatusCode, event.Data.Response.Message)
	}
}

func handleSoftBounce(event *lettermint.WebhookEvent) {
	log.Printf("Soft bounce for %s", event.Data.Recipient)

	// Example: Increment bounce counter, retry later
	// db.IncrementBounceCount(event.Data.Recipient)
}

func handleOpened(event *lettermint.WebhookEvent) {
	log.Printf("Email opened by %s", event.Data.Recipient)

	// Example: Track engagement
	// analytics.TrackOpen(event.Data.MessageID, event.Data.Recipient)
}

func handleClicked(event *lettermint.WebhookEvent) {
	log.Printf("Link clicked by %s", event.Data.Recipient)

	// Example: Track click engagement
	// analytics.TrackClick(event.Data.MessageID, event.Data.Recipient)
}

func handleComplained(event *lettermint.WebhookEvent) {
	log.Printf("Spam complaint from %s", event.Data.Recipient)

	// Example: Unsubscribe user to comply with regulations
	// db.UnsubscribeUser(event.Data.Recipient)
	// sendgrid.AddToSuppressionList(event.Data.Recipient)
}
