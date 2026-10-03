// Example: send one email with the Lettermint Go SDK.
//
// Usage:
//
//	export LETTERMINT_PROJECT_TOKEN="lm_..."
//	go run ./examples/basic
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	lettermint "github.com/lettermint/lettermint-go/v3"
)

func main() {
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
		Send(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(resp.MessageID, resp.Status)
}
