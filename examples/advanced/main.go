// Example: builder templates, attachments, tags, idempotency and errors.
//
// Usage:
//
//	export LETTERMINT_PROJECT_TOKEN="lm_..."
//	go run ./examples/advanced
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	lettermint "github.com/lettermint/lettermint-go/v3"
)

func main() {
	client, err := lettermint.New(
		lettermint.WithSendingToken(os.Getenv("LETTERMINT_PROJECT_TOKEN")),
		lettermint.WithTimeout(10*time.Second),
	)
	if err != nil {
		log.Fatal(err)
	}
	ctx := context.Background()

	// A builder is a value: keep a base and derive one email per recipient.
	invoice := client.Emails.Compose().
		From("Acme Billing <billing@acme.com>").
		Subject("Your invoice").
		Tags(lettermint.MessageTagInput{Name: "category", Value: "invoice"}).
		Metadata(map[string]string{"source": "example"})

	pdf := []byte("%PDF-1.4 ...")
	resp, err := invoice.
		To("jane@example.com").
		CC("accounts@example.com").
		HTML(`<p>Your invoice is attached.</p><img src="cid:logo">`).
		Attach(lettermint.Attachment{Filename: "invoice.pdf", Content: pdf, ContentType: "application/pdf"}).
		Attach(lettermint.Attachment{Filename: "logo.png", ContentBase64: "iVBORw0KGgo=", ContentID: "logo"}).
		Send(ctx, lettermint.WithIdempotencyKey("invoice-2026-10-jane"))

	var validation *lettermint.ValidationError
	var rateLimit *lettermint.RateLimitError
	var timeout *lettermint.TimeoutError
	switch {
	case errors.As(err, &validation):
		log.Fatalf("rejected: %s %v", validation.Message, validation.Errors)
	case errors.As(err, &rateLimit):
		log.Fatalf("rate limited, retry after %v with the same idempotency key", rateLimit.RetryAfter)
	case errors.As(err, &timeout):
		log.Fatal("the outcome is unknown; retry with the same idempotency key")
	case err != nil:
		log.Fatal(err)
	}
	fmt.Println("sent", resp.MessageID)

	// Schedule an email and cancel it again.
	scheduled, err := invoice.To("john@example.com").Text("Reminder").ScheduledAtTime(time.Now().Add(24 * time.Hour)).Send(ctx)
	if err != nil {
		log.Fatal(err)
	}
	if _, err := client.Messages.Cancel(ctx, scheduled.MessageID); err != nil {
		log.Fatal(err)
	}
}
