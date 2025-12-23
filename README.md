# Lettermint Go SDK

[![Go Reference](https://pkg.go.dev/badge/go.lettermint.co/sdk.svg)](https://pkg.go.dev/go.lettermint.co/sdk)
[![Go Report Card](https://goreportcard.com/badge/go.lettermint.co/sdk)](https://goreportcard.com/report/go.lettermint.co/sdk)
[![GitHub Tests Action Status](https://img.shields.io/github/actions/workflow/status/lettermint/lettermint-go/ci.yaml?branch=main&label=tests&style=flat-square)](https://github.com/lettermint/lettermint-go/actions?query=workflow%3ACI+branch%3Amain)

The official Go SDK for [Lettermint](https://lettermint.co).

## Requirements

- Go 1.21 or higher

## Installation

```bash
go get go.lettermint.co/sdk
```

## Usage

### Initialize the SDK

```go
import "go.lettermint.co/sdk"

client, err := lettermint.New("your-api-token")
if err != nil {
    log.Fatal(err)
}
```

### Sending Emails

The SDK provides a fluent interface for sending emails:

```go
ctx := context.Background()
resp, err := client.Email(ctx).
    From("sender@example.com").
    To("recipient@example.com").
    Subject("Hello from Lettermint").
    Text("This is a test email sent using the Lettermint Go SDK.").
    Send()

if err != nil {
    log.Fatal(err)
}

fmt.Printf("Email sent with ID: %s\n", resp.MessageID)
fmt.Printf("Status: %s\n", resp.Status)
```

#### Advanced Email Options

```go
resp, err := client.Email(ctx).
    From("John Doe <sender@example.com>").
    To("recipient1@example.com", "recipient2@example.com").
    CC("cc@example.com").
    BCC("bcc@example.com").
    ReplyTo("reply@example.com").
    Subject("Hello from Lettermint").
    HTML("<h1>Hello</h1><p>This is an HTML email.</p>").
    Text("This is a plain text version of the email.").
    Headers(map[string]string{
        "X-Custom-Header": "Custom Value",
    }).
    Attach("attachment.txt", base64EncodedContent).
    AttachWithContentID("logo.png", base64EncodedLogo, "logo"). // Inline attachment
    IdempotencyKey("unique-id-123").
    Metadata(map[string]string{
        "user_id": "12345",
    }).
    Tag("campaign-123").
    Send()
```

#### Inline Attachments

You can embed images and other content in your HTML emails using Content-IDs:

```go
resp, err := client.Email(ctx).
    From("sender@example.com").
    To("recipient@example.com").
    Subject("Email with inline image").
    HTML(`<p>Here is an image: <img src="cid:logo"></p>`).
    AttachWithContentID("logo.png", base64EncodedImage, "logo").
    Send()
```

### Idempotency

To ensure that duplicate requests are not processed, you can use an idempotency key:

```go
resp, err := client.Email(ctx).
    From("sender@example.com").
    To("recipient@example.com").
    Subject("Hello from Lettermint").
    Text("Hello! This is a test email.").
    IdempotencyKey("unique-request-id-123").
    Send()
```

The idempotency key should be a unique string that you generate for each unique email you want to send. If you make the same request with the same idempotency key, the API will return the same response without sending a duplicate email.

For more information, refer to the [documentation](https://docs.lettermint.co/platform/emails/idempotency).

### Webhook Verification

Verify webhook signatures to ensure the authenticity of webhook requests:

```go
func webhookHandler(w http.ResponseWriter, r *http.Request) {
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
        log.Printf("Email delivered to %s", event.Data.Recipient)
    case "message.hard_bounced":
        log.Printf("Hard bounce for %s", event.Data.Recipient)
    case "message.soft_bounced":
        log.Printf("Soft bounce for %s", event.Data.Recipient)
    }

    w.WriteHeader(http.StatusOK)
}
```

## API Reference

### Client Configuration

```go
client, err := lettermint.New("your-api-token",
    lettermint.WithBaseURL("https://api.lettermint.co/v1"), // Optional
    lettermint.WithTimeout(30*time.Second),                  // Optional
    lettermint.WithHTTPClient(customHTTPClient),             // Optional
)
```

### Email Builder Methods

- `From(email string)`: Set the sender email address
- `To(emails ...string)`: Set one or more recipient email addresses
- `Subject(subject string)`: Set the email subject
- `HTML(html string)`: Set the HTML body of the email
- `Text(text string)`: Set the plain text body of the email
- `CC(emails ...string)`: Set one or more CC email addresses
- `BCC(emails ...string)`: Set one or more BCC email addresses
- `ReplyTo(emails ...string)`: Set one or more Reply-To email addresses
- `Header(key, value string)`: Set a custom header
- `Headers(headers map[string]string)`: Set multiple custom headers
- `Attach(filename, base64Content string)`: Attach a file
- `AttachWithContentID(filename, content, contentID string)`: Attach an inline file
- `Route(route string)`: Set the routing key
- `IdempotencyKey(key string)`: Set an idempotency key
- `Metadata(metadata map[string]string)`: Set metadata
- `MetadataValue(key, value string)`: Set a single metadata value
- `Tag(tag string)`: Set a tag
- `Send() (*SendResponse, error)`: Send the email

### Error Handling

The SDK provides structured error types:

```go
resp, err := client.Email(ctx).From("...").To("...").Subject("...").HTML("...").Send()
if err != nil {
    // Check for specific error types
    var apiErr *lettermint.APIError
    if errors.As(err, &apiErr) {
        fmt.Printf("API Error (%d): %s\n", apiErr.StatusCode, apiErr.Message)
        fmt.Printf("Validation errors: %v\n", apiErr.Errors)
    }

    // Check for error categories using errors.Is()
    if errors.Is(err, lettermint.ErrValidation) {
        // Handle validation errors (422)
    } else if errors.Is(err, lettermint.ErrUnauthorized) {
        // Handle authentication errors (401)
    } else if errors.Is(err, lettermint.ErrTimeout) {
        // Handle timeout errors
    } else if errors.Is(err, lettermint.ErrRateLimited) {
        // Handle rate limit errors (429)
    }
}
```

## Testing

```bash
go test ./...
```

## Changelog

Please see [CHANGELOG](CHANGELOG.md) for more information on what has changed recently.

## Contributing

Please see [CONTRIBUTING](CONTRIBUTING.md) for details.

## Security Vulnerabilities

Please review [our security policy](../../security/policy) on how to report security vulnerabilities.

## Credits

- [Bjarn Bronsveld](https://github.com/bjarn)
- [All Contributors](../../contributors)

## License

The MIT License (MIT). Please see [License File](LICENSE) for more information.
