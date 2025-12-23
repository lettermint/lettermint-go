// Example: Basic email sending with Lettermint
//
// This example demonstrates the simplest way to send an email using the Lettermint SDK.
// Set the LETTERMINT_API_TOKEN environment variable before running.
//
// Usage:
//
//	export LETTERMINT_API_TOKEN="your-api-token"
//	go run main.go
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"go.lettermint.co/sdk"
)

func main() {
	// Get API token from environment variable
	apiToken := os.Getenv("LETTERMINT_API_TOKEN")
	if apiToken == "" {
		log.Fatal("LETTERMINT_API_TOKEN environment variable is required")
	}

	// Create a new Lettermint client
	client, err := lettermint.New(apiToken)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	// Send an email using the fluent builder interface
	ctx := context.Background()
	resp, err := client.Email(ctx).
		From("sender@example.com").
		To("recipient@example.com").
		Subject("Hello from Lettermint Go SDK").
		Text("This is a test email sent using the Lettermint Go SDK.").
		Send()

	if err != nil {
		log.Fatalf("Failed to send email: %v", err)
	}

	fmt.Printf("Email sent successfully!\n")
	fmt.Printf("Message ID: %s\n", resp.MessageID)
	fmt.Printf("Status: %s\n", resp.Status)
}
