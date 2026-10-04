// Example: verify Lettermint webhook deliveries in an HTTP handler.
//
// Usage:
//
//	export LETTERMINT_WEBHOOK_SECRET="whsec_..."
//	go run ./examples/webhook
//
// Then point a Lettermint webhook at http://your-server:8080/webhooks/lettermint.
package main

import (
	"errors"
	"log"
	"net/http"
	"os"

	lettermint "github.com/lettermint/lettermint-go/v3"
)

func main() {
	webhook, err := lettermint.NewWebhook(os.Getenv("LETTERMINT_WEBHOOK_SECRET"))
	if err != nil {
		log.Fatal(err)
	}

	http.HandleFunc("POST /webhooks/lettermint", func(w http.ResponseWriter, r *http.Request) {
		// VerifyRequest reads the raw body and checks X-Lettermint-Signature
		// and X-Lettermint-Delivery.
		event, err := webhook.VerifyRequest(r)
		var verification *lettermint.WebhookVerificationError
		if errors.As(err, &verification) {
			log.Printf("rejected webhook: %s", verification.Reason)
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
				Recipient string `json:"recipient"`
			}
			if err := event.DecodeData(&data); err == nil {
				log.Printf("delivered %s to %s", data.MessageID, data.Recipient)
			}
		case lettermint.WebhookEventMessageHardBounced:
			log.Printf("hard bounce: %s", event.Data)
		default:
			// Unknown events pass through; ignore the ones you do not handle.
			log.Printf("event %s", event.Event)
		}
		w.WriteHeader(http.StatusNoContent)
	})

	log.Println("listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
