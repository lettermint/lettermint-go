package lettermint

// SendResponse represents the response from the send email API.
type SendResponse struct {
	// MessageID is the unique identifier for the sent message.
	MessageID string `json:"message_id"`

	// Status is the current status of the message.
	// Possible values: pending, queued, processed, delivered, soft_bounced, hard_bounced, failed
	Status string `json:"status"`
}

// Attachment represents an email attachment.
type Attachment struct {
	// Filename is the name of the attachment file.
	Filename string `json:"filename"`

	// Content is the base64-encoded content of the attachment.
	Content string `json:"content"`

	// ContentID is the Content-ID for inline attachments (optional).
	// Used for embedding images in HTML via cid: references.
	ContentID string `json:"content_id,omitempty"`
}

// emailPayload is the internal structure sent to the API.
type emailPayload struct {
	From        string            `json:"from"`
	To          []string          `json:"to"`
	Subject     string            `json:"subject"`
	HTML        string            `json:"html,omitempty"`
	Text        string            `json:"text,omitempty"`
	CC          []string          `json:"cc,omitempty"`
	BCC         []string          `json:"bcc,omitempty"`
	ReplyTo     []string          `json:"reply_to,omitempty"`
	Headers     map[string]string `json:"headers,omitempty"`
	Attachments []Attachment      `json:"attachments,omitempty"`
	Route       string            `json:"route,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	Tag         string            `json:"tag,omitempty"`
}

// WebhookEvent represents a parsed webhook payload from Lettermint.
type WebhookEvent struct {
	// ID is the unique webhook delivery ID.
	ID string `json:"id"`

	// Event is the event type (e.g., "message.delivered", "message.bounced").
	Event string `json:"event"`

	// Timestamp is the Unix timestamp when the event occurred.
	Timestamp int64 `json:"timestamp"`

	// Data contains the event-specific data.
	Data WebhookEventData `json:"data"`

	// RawPayload contains the original JSON payload for custom parsing.
	RawPayload []byte `json:"-"`
}

// WebhookEventData contains the event-specific data in a webhook payload.
type WebhookEventData struct {
	// MessageID is the unique identifier of the related message.
	MessageID string `json:"message_id"`

	// Recipient is the email address of the recipient.
	Recipient string `json:"recipient"`

	// Tag is the tag associated with the message (if set).
	Tag string `json:"tag,omitempty"`

	// Metadata contains the custom metadata associated with the message.
	Metadata map[string]string `json:"metadata,omitempty"`

	// Response contains delivery response details (for delivered/bounced events).
	Response *WebhookResponse `json:"response,omitempty"`
}

// WebhookResponse contains delivery response details.
type WebhookResponse struct {
	// StatusCode is the SMTP response status code.
	StatusCode int `json:"status_code,omitempty"`

	// Message is the SMTP response message.
	Message string `json:"message,omitempty"`
}

// apiErrorResponse is the structure of error responses from the API.
type apiErrorResponse struct {
	Message   string              `json:"message"`
	Error     string              `json:"error"`
	ErrorType string              `json:"error_type"`
	Errors    map[string][]string `json:"errors"`
}
