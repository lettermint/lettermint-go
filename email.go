package lettermint

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"runtime"
	"strings"
)

// EmailBuilder provides a fluent interface for composing and sending emails.
//
// Create a new EmailBuilder using Client.Email(ctx).
// The builder is NOT safe for concurrent use; create a new builder for each email.
type EmailBuilder struct {
	client         *Client
	ctx            context.Context
	payload        *emailPayload
	idempotencyKey string
}

// From sets the sender email address.
//
// Supports RFC 5322 format: "John Doe <john@example.com>" or plain "john@example.com".
func (b *EmailBuilder) From(email string) *EmailBuilder {
	b.payload.From = email
	return b
}

// To adds one or more recipient email addresses.
//
// Can be called multiple times to add more recipients.
func (b *EmailBuilder) To(emails ...string) *EmailBuilder {
	b.payload.To = append(b.payload.To, emails...)
	return b
}

// CC adds one or more CC recipient email addresses.
//
// Can be called multiple times to add more CC recipients.
func (b *EmailBuilder) CC(emails ...string) *EmailBuilder {
	b.payload.CC = append(b.payload.CC, emails...)
	return b
}

// BCC adds one or more BCC recipient email addresses.
//
// Can be called multiple times to add more BCC recipients.
func (b *EmailBuilder) BCC(emails ...string) *EmailBuilder {
	b.payload.BCC = append(b.payload.BCC, emails...)
	return b
}

// ReplyTo sets one or more Reply-To email addresses.
//
// Can be called multiple times to add more Reply-To addresses.
func (b *EmailBuilder) ReplyTo(emails ...string) *EmailBuilder {
	b.payload.ReplyTo = append(b.payload.ReplyTo, emails...)
	return b
}

// Subject sets the email subject line.
func (b *EmailBuilder) Subject(subject string) *EmailBuilder {
	b.payload.Subject = subject
	return b
}

// HTML sets the HTML body content.
//
// At least one of HTML or Text must be set before sending.
func (b *EmailBuilder) HTML(html string) *EmailBuilder {
	b.payload.HTML = html
	return b
}

// Text sets the plain text body content.
//
// At least one of HTML or Text must be set before sending.
func (b *EmailBuilder) Text(text string) *EmailBuilder {
	b.payload.Text = text
	return b
}

// Header adds a single custom email header.
//
// Can be called multiple times to add more headers.
func (b *EmailBuilder) Header(key, value string) *EmailBuilder {
	if b.payload.Headers == nil {
		b.payload.Headers = make(map[string]string)
	}
	b.payload.Headers[key] = value
	return b
}

// Headers sets multiple custom email headers at once.
//
// Merges with any headers already set via Header().
func (b *EmailBuilder) Headers(headers map[string]string) *EmailBuilder {
	if b.payload.Headers == nil {
		b.payload.Headers = make(map[string]string)
	}
	for k, v := range headers {
		b.payload.Headers[k] = v
	}
	return b
}

// Attach adds a file attachment to the email.
//
// The content must be base64-encoded.
func (b *EmailBuilder) Attach(filename, content string) *EmailBuilder {
	return b.AttachWithOptions(filename, content, AttachmentOptions{})
}

// AttachWithContentID adds a file attachment with a Content-ID for inline embedding.
//
// Use this for embedding images in HTML emails via cid: references.
// Example: <img src="cid:logo"> with contentID "logo".
func (b *EmailBuilder) AttachWithContentID(filename, content, contentID string) *EmailBuilder {
	return b.AttachWithOptions(filename, content, AttachmentOptions{ContentID: contentID})
}

// AttachWithContentType adds a file attachment with an explicit MIME type.
func (b *EmailBuilder) AttachWithContentType(filename, content, contentType string) *EmailBuilder {
	return b.AttachWithOptions(filename, content, AttachmentOptions{ContentType: contentType})
}

// AttachWithOptions adds a file attachment with optional Content-ID and MIME type values.
func (b *EmailBuilder) AttachWithOptions(filename, content string, options AttachmentOptions) *EmailBuilder {
	b.payload.Attachments = append(b.payload.Attachments, Attachment{
		Filename:    filename,
		Content:     content,
		ContentID:   options.ContentID,
		ContentType: options.ContentType,
	})
	return b
}

// Settings sets per-email route overrides.
func (b *EmailBuilder) Settings(settings EmailSettings) *EmailBuilder {
	b.payload.Settings = &settings
	return b
}

// Metadata sets custom metadata key-value pairs.
//
// Metadata is included in webhook payloads but not in email headers.
// Merges with any metadata already set.
func (b *EmailBuilder) Metadata(metadata map[string]string) *EmailBuilder {
	if b.payload.Metadata == nil {
		b.payload.Metadata = make(map[string]string)
	}
	for k, v := range metadata {
		b.payload.Metadata[k] = v
	}
	return b
}

// MetadataValue sets a single metadata key-value pair.
//
// Can be called multiple times to add more metadata.
func (b *EmailBuilder) MetadataValue(key, value string) *EmailBuilder {
	if b.payload.Metadata == nil {
		b.payload.Metadata = make(map[string]string)
	}
	b.payload.Metadata[key] = value
	return b
}

// Tag sets an email tag for categorization.
//
// Tags can be used to filter and group emails in the Lettermint dashboard.
func (b *EmailBuilder) Tag(tag string) *EmailBuilder {
	b.payload.Tag = tag
	return b
}

// Tags sets reusable name-value tags for the email.
//
// Each tag name and value must use the format required by the Sending API.
func (b *EmailBuilder) Tags(tags ...map[string]string) *EmailBuilder {
	b.payload.Tags = make([]MessageTag, 0, len(tags))
	for _, tag := range tags {
		b.payload.Tags = append(b.payload.Tags, MessageTag{Name: tag["name"], Value: tag["value"]})
	}
	return b
}

// MessageTags sets typed reusable name-value tags for the email.
func (b *EmailBuilder) MessageTags(tags ...MessageTag) *EmailBuilder {
	b.payload.Tags = append([]MessageTag(nil), tags...)
	return b
}

// Route sets the routing key for the email.
//
// Routes determine which sending configuration to use.
func (b *EmailBuilder) Route(route string) *EmailBuilder {
	b.payload.Route = route
	return b
}

// ScheduledAt sets the delivery time for the email.
// The Sending API accepts an ISO 8601 timestamp or a supported English time expression.
func (b *EmailBuilder) ScheduledAt(scheduledAt string) *EmailBuilder {
	b.payload.ScheduledAt = scheduledAt
	return b
}

// IdempotencyKey sets an idempotency key to prevent duplicate sends.
//
// If you provide the same idempotency key for multiple requests,
// only the first one will be processed. Use this when retrying failed requests.
func (b *EmailBuilder) IdempotencyKey(key string) *EmailBuilder {
	b.idempotencyKey = key
	return b
}

// Send sends the composed email via the Lettermint API.
//
// Returns the send response containing the message ID and status,
// or an error if the request fails.
//
// The context passed to Email() controls the request lifecycle.
// Use context.WithTimeout() or context.WithDeadline() for custom timeouts.
func (b *EmailBuilder) Send() (*SendResponse, error) {
	if err := b.validate(); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidRequest, err)
	}
	defer b.reset()

	jsonData, err := json.Marshal(b.payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal email payload: %w", err)
	}

	url := fmt.Sprintf("%s/send", strings.TrimSuffix(b.client.baseURL, "/"))
	req, err := http.NewRequestWithContext(b.ctx, http.MethodPost, url, bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("x-lettermint-token", b.client.apiToken)
	req.Header.Set("User-Agent", fmt.Sprintf("Lettermint/%s (Go; %s)", Version, runtime.Version()))

	if b.idempotencyKey != "" {
		req.Header.Set("Idempotency-Key", b.idempotencyKey)
	}

	resp, err := b.client.httpClient.Do(req)
	if err != nil {
		if b.ctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("%w: %v", ErrTimeout, err)
		}
		if b.ctx.Err() == context.Canceled {
			return nil, fmt.Errorf("request canceled: %w", err)
		}
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, parseAPIError(resp.StatusCode, body)
	}

	var sendResp SendResponse
	if err := json.Unmarshal(body, &sendResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &sendResp, nil
}

func (b *EmailBuilder) reset() {
	b.payload = &emailPayload{
		To:      []string{},
		CC:      []string{},
		BCC:     []string{},
		ReplyTo: []string{},
	}
	b.idempotencyKey = ""
}

// validate checks that all required fields are set.
func (b *EmailBuilder) validate() error {
	if b.payload.From == "" {
		return fmt.Errorf("from address is required")
	}
	if len(b.payload.To) == 0 {
		return fmt.Errorf("at least one recipient is required")
	}
	if b.payload.Subject == "" {
		return fmt.Errorf("subject is required")
	}
	if b.payload.HTML == "" && b.payload.Text == "" {
		return fmt.Errorf("either html or text body is required")
	}
	maximum := 20
	if b.payload.Tag != "" {
		maximum = 19
	}
	if len(b.payload.Tags) > maximum {
		return fmt.Errorf("no more than %d message tags are permitted", maximum)
	}
	namePattern := regexp.MustCompile(`^[A-Za-z0-9_-]{1,32}$`)
	valuePattern := regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)
	names := make(map[string]struct{}, len(b.payload.Tags))
	for _, tag := range b.payload.Tags {
		if !namePattern.MatchString(tag.Name) {
			return fmt.Errorf("message tag names must match ^[A-Za-z0-9_-]{1,32}$")
		}
		if strings.HasPrefix(strings.ToLower(tag.Name), "__lettermint") {
			return fmt.Errorf("message tag names must not start with __lettermint")
		}
		if !valuePattern.MatchString(tag.Value) {
			return fmt.Errorf("message tag values must match ^[A-Za-z0-9_-]{1,64}$")
		}
		if _, exists := names[tag.Name]; exists {
			return fmt.Errorf("message tag names must be unique and case-sensitive")
		}
		names[tag.Name] = struct{}{}
	}
	return nil
}

// parseAPIError converts an HTTP error response to an APIError.
func parseAPIError(statusCode int, body []byte) error {
	apiErr := &APIError{
		StatusCode:   statusCode,
		ResponseBody: string(body),
	}

	var errResp apiErrorResponse
	if err := json.Unmarshal(body, &errResp); err == nil {
		if errResp.Message != "" {
			apiErr.Message = errResp.Message
		} else if errResp.Error != "" {
			apiErr.Message = errResp.Error
		}
		apiErr.ErrorType = errResp.ErrorType
		apiErr.Errors = errResp.Errors
	} else {
		apiErr.Message = string(body)
	}

	if apiErr.Message == "" {
		apiErr.Message = http.StatusText(statusCode)
	}

	return apiErr
}
