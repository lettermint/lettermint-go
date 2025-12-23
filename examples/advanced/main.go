// Example: Advanced email sending with Lettermint
//
// This example demonstrates advanced email features including:
// - HTML and text bodies
// - CC and BCC recipients
// - File attachments (including inline images)
// - Custom metadata and tags
// - Idempotency keys
// - Custom timeouts
//
// Usage:
//
//	export LETTERMINT_API_TOKEN="your-api-token"
//	go run main.go
package main

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	lettermint "github.com/lettermint/lettermint-go"
)

func main() {
	apiToken := os.Getenv("LETTERMINT_API_TOKEN")
	if apiToken == "" {
		log.Fatal("LETTERMINT_API_TOKEN environment variable is required")
	}

	// Create client with custom timeout
	client, err := lettermint.New(apiToken,
		lettermint.WithTimeout(60*time.Second),
	)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	// Create a context with timeout for this specific request
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Sample attachment content (in real usage, read from file)
	attachmentContent := base64.StdEncoding.EncodeToString([]byte("Hello, this is a text file attachment!"))

	// Sample inline image (1x1 red pixel PNG)
	inlineImageContent := base64.StdEncoding.EncodeToString([]byte{
		0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 0x00, 0x00, 0x00, 0x0D,
		0x49, 0x48, 0x44, 0x52, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x02, 0x00, 0x00, 0x00, 0x90, 0x77, 0x53, 0xDE, 0x00, 0x00, 0x00,
		0x0C, 0x49, 0x44, 0x41, 0x54, 0x08, 0xD7, 0x63, 0xF8, 0xCF, 0xC0, 0x00,
		0x00, 0x00, 0x03, 0x00, 0x01, 0x00, 0x05, 0xFE, 0xD4, 0xEF, 0x00, 0x00,
		0x00, 0x00, 0x49, 0x45, 0x4E, 0x44, 0xAE, 0x42, 0x60, 0x82,
	})

	// Send a full-featured email
	resp, err := client.Email(ctx).
		// Sender with display name (RFC 5322 format)
		From("John Doe <john@example.com>").
		// Multiple recipients
		To("recipient1@example.com", "recipient2@example.com").
		CC("manager@example.com").
		BCC("archive@example.com").
		ReplyTo("support@example.com").
		// Subject and body
		Subject("Monthly Report - December 2024").
		HTML(`
			<h1>Monthly Report</h1>
			<p>Please find the attached report for December 2024.</p>
			<p>Here's an inline image: <img src="cid:logo" alt="Logo"></p>
			<p>Best regards,<br>John Doe</p>
		`).
		Text("Monthly Report\n\nPlease find the attached report for December 2024.\n\nBest regards,\nJohn Doe").
		// Custom headers
		Header("X-Campaign-ID", "dec-2024-report").
		// Attachments
		Attach("report.txt", attachmentContent).
		AttachWithContentID("logo.png", inlineImageContent, "logo").
		// Metadata (included in webhooks)
		Metadata(map[string]string{
			"user_id":     "12345",
			"campaign_id": "dec-2024",
		}).
		MetadataValue("department", "sales").
		// Tag for categorization
		Tag("monthly-report").
		// Route (optional, for custom sending configuration)
		Route("transactional").
		// Idempotency key to prevent duplicate sends
		IdempotencyKey("report-dec-2024-12345").
		Send()

	if err != nil {
		// Handle specific error types
		var apiErr *lettermint.APIError
		if errors.As(err, &apiErr) {
			fmt.Printf("API Error (%d): %s\n", apiErr.StatusCode, apiErr.Message)
			if len(apiErr.Errors) > 0 {
				fmt.Printf("Validation errors: %v\n", apiErr.Errors)
			}
		}

		// Check for specific error categories
		if errors.Is(err, lettermint.ErrValidation) {
			fmt.Println("Validation failed - check your email parameters")
		} else if errors.Is(err, lettermint.ErrUnauthorized) {
			fmt.Println("Authentication failed - check your API token")
		} else if errors.Is(err, lettermint.ErrTimeout) {
			fmt.Println("Request timed out - try again later")
		} else if errors.Is(err, lettermint.ErrRateLimited) {
			fmt.Println("Rate limited - slow down your requests")
		}

		log.Fatalf("Failed to send email: %v", err)
	}

	fmt.Printf("Email sent successfully!\n")
	fmt.Printf("Message ID: %s\n", resp.MessageID)
	fmt.Printf("Status: %s\n", resp.Status)
}
