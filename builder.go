package lettermint

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"maps"
	"slices"
	"time"
)

// Attachment is a file for EmailBuilder.Attach. Set Content to the raw bytes
// (the SDK base64-encodes them) or ContentBase64 to content that is already
// base64-encoded, not both.
type Attachment struct {
	Filename string
	// Content is the raw file content.
	Content []byte
	// ContentBase64 is the base64-encoded file content.
	ContentBase64 string
	// ContentType is the MIME type, for example application/pdf. The API
	// detects it when empty.
	ContentType string
	// ContentID makes the attachment inline: reference it as cid:<ContentID>
	// in the HTML body.
	ContentID string
}

// EmailBuilder composes an email. Create it with client.Emails.Compose().
//
// EmailBuilder is a value. Every setter returns a new builder and leaves the
// one it was called on unchanged, so a builder can be kept as a template and
// reused, also across goroutines. When you build an email over several
// statements, keep the returned builder:
//
//	email := client.Emails.Compose().From("hello@acme.com").To(user.Email).Subject("Your invoice")
//	if user.Accountant != "" {
//		email = email.CC(user.Accountant)
//	}
//	resp, err := email.HTML(invoiceHTML).Send(ctx)
//
// A setter that receives invalid input, such as an invalid tag, records the
// error on the builder it returns; Err, Build and Send return it, and Send
// makes no request. The builder it was called on stays valid.
//
// To, CC, BCC, ReplyTo, Headers, Metadata and Tags replace their value;
// Attach adds an attachment. An empty string removes HTML, Text, Tag, Route
// and ScheduledAt.
type EmailBuilder struct {
	emails  *EmailsService
	message SendMailRequest
	err     error
}

// with returns a copy of the builder with the change applied to a copy of the
// message, then validates the message.
func (b EmailBuilder) with(change func(*SendMailRequest)) EmailBuilder {
	next := EmailBuilder{emails: b.emails, message: b.message, err: b.err}
	change(&next.message)
	if next.err == nil {
		next.err = validateMessage(next.message, "")
	}
	return next
}

// From sets the sender, for example "Acme <hello@acme.com>".
func (b EmailBuilder) From(address string) EmailBuilder {
	return b.with(func(m *SendMailRequest) { m.From = address })
}

// To replaces the recipients.
func (b EmailBuilder) To(addresses ...string) EmailBuilder {
	return b.with(func(m *SendMailRequest) { m.To = append([]string{}, addresses...) })
}

// CC replaces the CC recipients.
func (b EmailBuilder) CC(addresses ...string) EmailBuilder {
	return b.with(func(m *SendMailRequest) { m.CC = slices.Clone(addresses) })
}

// BCC replaces the BCC recipients.
func (b EmailBuilder) BCC(addresses ...string) EmailBuilder {
	return b.with(func(m *SendMailRequest) { m.BCC = slices.Clone(addresses) })
}

// ReplyTo replaces the Reply-To addresses.
func (b EmailBuilder) ReplyTo(addresses ...string) EmailBuilder {
	return b.with(func(m *SendMailRequest) { m.ReplyTo = slices.Clone(addresses) })
}

// Subject sets the subject line.
func (b EmailBuilder) Subject(subject string) EmailBuilder {
	return b.with(func(m *SendMailRequest) { m.Subject = subject })
}

// HTML sets the HTML body. An empty string removes it.
func (b EmailBuilder) HTML(html string) EmailBuilder {
	return b.with(func(m *SendMailRequest) { m.HTML = optionalString(html) })
}

// Text sets the plain-text body. An empty string removes it.
func (b EmailBuilder) Text(text string) EmailBuilder {
	return b.with(func(m *SendMailRequest) { m.Text = optionalString(text) })
}

// Headers replaces the custom email headers.
func (b EmailBuilder) Headers(headers map[string]string) EmailBuilder {
	return b.with(func(m *SendMailRequest) { m.Headers = maps.Clone(headers) })
}

// Metadata replaces the metadata: data tracked with the email and included
// in webhooks, not added as headers.
func (b EmailBuilder) Metadata(metadata map[string]string) EmailBuilder {
	return b.with(func(m *SendMailRequest) { m.Metadata = maps.Clone(metadata) })
}

// Tag sets the legacy single tag. An empty string removes it.
func (b EmailBuilder) Tag(tag string) EmailBuilder {
	return b.with(func(m *SendMailRequest) { m.Tag = optionalString(tag) })
}

// Tags replaces the name/value tags: up to 20, or 19 with a legacy Tag.
// Names match ^[A-Za-z0-9_-]{1,32}$, must be unique and may not start with
// __lettermint; values match ^[A-Za-z0-9_-]{1,64}$.
func (b EmailBuilder) Tags(tags ...MessageTagInput) EmailBuilder {
	return b.with(func(m *SendMailRequest) { m.Tags = slices.Clone(tags) })
}

// Route sets the slug of the route to send through. An empty string removes it.
func (b EmailBuilder) Route(slug string) EmailBuilder {
	return b.with(func(m *SendMailRequest) { m.Route = optionalPointer(slug) })
}

// ScheduledAt schedules delivery: ISO 8601, or English such as "tomorrow
// 9am". An empty string removes the schedule.
func (b EmailBuilder) ScheduledAt(when string) EmailBuilder {
	return b.with(func(m *SendMailRequest) { m.ScheduledAt = optionalPointer(when) })
}

// ScheduledAtTime schedules delivery at a time, sent as ISO 8601 in UTC. The
// zero time removes the schedule.
func (b EmailBuilder) ScheduledAtTime(when time.Time) EmailBuilder {
	if when.IsZero() {
		return b.ScheduledAt("")
	}
	return b.ScheduledAt(when.UTC().Format("2006-01-02T15:04:05.000Z07:00"))
}

// Settings sets per-email settings that override the route settings.
func (b EmailBuilder) Settings(settings SendMailRequestSettings) EmailBuilder {
	return b.with(func(m *SendMailRequest) { m.Settings = &settings })
}

// SandboxResult sets the result a Sandbox project simulates for every recipient.
func (b EmailBuilder) SandboxResult(result SandboxResult) EmailBuilder {
	return b.with(func(m *SendMailRequest) { m.SandboxResult = &result })
}

// Attach adds an attachment.
func (b EmailBuilder) Attach(attachment Attachment) EmailBuilder {
	input := MessageAttachmentInput{Filename: attachment.Filename}
	var err error
	switch {
	case attachment.Content != nil && attachment.ContentBase64 != "":
		err = &ClientValidationError{Field: "attachments", Message: "set Content or ContentBase64 of an attachment, not both"}
	case attachment.Content != nil:
		input.Content = base64.StdEncoding.EncodeToString(attachment.Content)
	default:
		input.Content = attachment.ContentBase64
	}
	if attachment.ContentType != "" {
		input.ContentType = Value(attachment.ContentType)
	}
	if attachment.ContentID != "" {
		input.ContentID = Value(attachment.ContentID)
	}
	next := b.with(func(m *SendMailRequest) {
		m.Attachments = append(slices.Clip(m.Attachments), input)
	})
	if next.err == nil {
		next.err = err
	}
	return next
}

// Err returns the first error a setter recorded, or nil.
func (b EmailBuilder) Err() error { return b.err }

// Build returns a copy of the message in the API's format, or the error a
// setter recorded.
func (b EmailBuilder) Build() (SendMailRequest, error) {
	if b.err != nil {
		return SendMailRequest{}, b.err
	}
	return cloneMessage(b.message), nil
}

// Send sends the email. The builder stays unchanged and can be sent again.
func (b EmailBuilder) Send(ctx context.Context, options ...IdempotentOption) (*SendMailResponse, error) {
	if b.emails == nil {
		return nil, &ConfigError{Message: "create an EmailBuilder with client.Emails.Compose()"}
	}
	if b.err != nil {
		return nil, b.err
	}
	return b.emails.Send(ctx, cloneMessage(b.message), options...)
}

func (b EmailBuilder) view() view {
	return view{name: "EmailBuilder", fields: []viewField{{"Message", messageJSON(b.message)}}}
}

// String shows the message in the API's format. It never contains a token.
func (b EmailBuilder) String() string { return b.view().String() }

// GoString shows the message in the API's format. It never contains a token.
func (b EmailBuilder) GoString() string { return b.view().String() }

// Format shows the message for every verb. It never prints a token.
func (b EmailBuilder) Format(f fmt.State, verb rune) { b.view().format(f, verb) }

// LogValue implements slog.LogValuer without credentials.
func (b EmailBuilder) LogValue() slog.Value { return b.view().logValue() }

// MarshalJSON encodes the message in the API's format.
func (b EmailBuilder) MarshalJSON() ([]byte, error) { return json.Marshal(b.message) }

func messageJSON(message SendMailRequest) string {
	data, err := json.Marshal(message)
	if err != nil {
		return "{}"
	}
	return string(data)
}

func optionalString(value string) Nullable[string] {
	if value == "" {
		return Nullable[string]{}
	}
	return Value(value)
}

func optionalPointer[T ~string](value T) *T {
	if value == "" {
		return nil
	}
	return &value
}

// cloneMessage copies a message deeply enough that neither copy can change
// the other: slices and maps are copied.
func cloneMessage(message SendMailRequest) SendMailRequest {
	clone := message
	clone.To = slices.Clone(message.To)
	clone.CC = slices.Clone(message.CC)
	clone.BCC = slices.Clone(message.BCC)
	clone.ReplyTo = slices.Clone(message.ReplyTo)
	clone.Headers = maps.Clone(message.Headers)
	clone.Metadata = maps.Clone(message.Metadata)
	clone.Tags = slices.Clone(message.Tags)
	clone.Attachments = slices.Clone(message.Attachments)
	if message.Settings != nil {
		settings := *message.Settings
		clone.Settings = &settings
	}
	if message.Route != nil {
		clone.Route = Ptr(*message.Route)
	}
	if message.ScheduledAt != nil {
		clone.ScheduledAt = Ptr(*message.ScheduledAt)
	}
	if message.SandboxResult != nil {
		clone.SandboxResult = Ptr(*message.SandboxResult)
	}
	return clone
}

// Ptr returns a pointer to v, for optional fields of request types:
//
//	lettermint.UpdateProjectData{Name: lettermint.Ptr("Production")}
func Ptr[T any](v T) *T {
	return &v
}
