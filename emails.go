package lettermint

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"
)

// EmailsService sends email with the project sending token
// (x-lettermint-token). It holds no message state: every call sends exactly
// what it is given.
type EmailsService struct{ service }

// Send sends one email. The message is in the API's format; see Compose for a
// builder.
//
//	resp, err := client.Emails.Send(ctx, lettermint.SendMailRequest{
//		From:    "Acme <hello@acme.com>",
//		To:      []string{"jane@example.com"},
//		Subject: "Welcome",
//		HTML:    lettermint.Value("<p>Thanks for signing up.</p>"),
//	}, lettermint.WithIdempotencyKey("welcome-jane"))
func (s *EmailsService) Send(ctx context.Context, message SendMailRequest, options ...IdempotentOption) (*SendMailResponse, error) {
	if _, _, err := s.t.authHeader("Emails.Send", authSending); err != nil {
		return nil, err
	}
	if err := validateMessage(message, ""); err != nil {
		return nil, err
	}
	return callJSON[SendMailResponse](ctx, s.t, opSendMail, callArgs{label: "Emails.Send", body: message, options: idempotentOptions(options)})
}

// SendBatch sends up to 500 emails in one request. To include a builder, call
// its Build method. To give each email its own idempotency key, use
// SendBatchItems.
//
// When an email in the middle of a batch fails, the emails before it may
// already be accepted. The returned *APIError then lists them in
// Details["accepted"] (index and message_id).
func (s *EmailsService) SendBatch(ctx context.Context, messages []SendMailRequest, options ...IdempotentOption) (SendBatchMailResponse, error) {
	if _, _, err := s.t.authHeader("Emails.SendBatch", authSending); err != nil {
		return nil, err
	}
	for i, message := range messages {
		if err := validateMessage(message, fmt.Sprintf("messages[%d]", i)); err != nil {
			return nil, err
		}
	}
	body := messages
	if body == nil {
		body = []SendMailRequest{}
	}
	result, err := callJSON[SendBatchMailResponse](ctx, s.t, opSendBatchMail, callArgs{label: "Emails.SendBatch", body: body, options: idempotentOptions(options)})
	if err != nil {
		return nil, err
	}
	return *result, nil
}

// SendBatchItems sends up to 500 emails in one request, like SendBatch, and
// accepts a per-email IdempotencyKey (1-255 characters, unique within the
// batch). When an email with the same key and content was accepted in the
// last 24 hours, the API returns its original result and does not send it
// again. Reusing a key with different content, or while that email is still
// being processed, rejects the whole batch with a *ConflictError and sends
// nothing. To include a builder, call its BuildBatchItem method.
//
//	results, err := client.Emails.SendBatchItems(ctx, []lettermint.SendBatchMailItem{
//		{From: "hello@acme.com", To: []string{"jane@example.com"}, Subject: "Hi Jane", Text: lettermint.Value("Hello"), IdempotencyKey: lettermint.Ptr("welcome-jane")},
//	})
//
// The per-email keys are part of each email. WithIdempotencyKey still sets
// the Idempotency-Key header for the whole request. On a partial failure the
// returned *APIError lists the accepted emails in Details["accepted"], as
// for SendBatch.
func (s *EmailsService) SendBatchItems(ctx context.Context, items []SendBatchMailItem, options ...IdempotentOption) (SendBatchMailResponse, error) {
	if _, _, err := s.t.authHeader("Emails.SendBatchItems", authSending); err != nil {
		return nil, err
	}
	keys := make(map[string]int, len(items))
	for i, item := range items {
		prefix := fmt.Sprintf("items[%d]", i)
		if err := validateContent(item.Tag, item.Tags, item.Attachments, prefix); err != nil {
			return nil, err
		}
		if item.IdempotencyKey == nil {
			continue
		}
		key := *item.IdempotencyKey
		if err := validateBatchIdempotencyKey(key, prefix+".idempotency_key"); err != nil {
			return nil, err
		}
		if first, seen := keys[key]; seen {
			return nil, &ClientValidationError{Field: prefix + ".idempotency_key", Message: fmt.Sprintf("idempotency keys must be unique within a batch; items[%d] has the same key", first)}
		}
		keys[key] = i
	}
	body := items
	if body == nil {
		body = []SendBatchMailItem{}
	}
	result, err := callJSON[SendBatchMailResponse](ctx, s.t, opSendBatchMail, callArgs{label: "Emails.SendBatchItems", body: body, options: idempotentOptions(options)})
	if err != nil {
		return nil, err
	}
	return *result, nil
}

// Compose starts an email builder. The builder is a value: every setter
// returns a new builder and leaves the one it was called on unchanged, so a
// builder can be kept as a template and reused, also concurrently.
//
//	welcome := client.Emails.Compose().From("Acme <hello@acme.com>").Subject("Welcome")
//	_, err := welcome.To("jane@example.com").HTML("<p>Hi Jane</p>").Send(ctx)
func (s *EmailsService) Compose() EmailBuilder {
	return EmailBuilder{emails: s, message: SendMailRequest{To: []string{}}}
}

// ComposeFrom starts an email builder from a message. The builder copies the
// message, so later changes to it do not affect the builder.
func (s *EmailsService) ComposeFrom(message SendMailRequest) EmailBuilder {
	builder := EmailBuilder{emails: s, message: cloneMessage(message)}
	builder.err = validateMessage(builder.message, "")
	return builder
}

// Ping checks the sending token: GET /ping returns "pong".
func (s *EmailsService) Ping(ctx context.Context, options ...RequestOption) (string, error) {
	text, err := callText(ctx, s.t, opPing, callArgs{label: "Emails.Ping", auth: authSending, options: requestOptions(options)})
	return strings.TrimSpace(text), err
}

const maxTags = 20

var (
	tagNamePattern  = regexp.MustCompile(`^[A-Za-z0-9_-]{1,32}$`)
	tagValuePattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)
)

// validateMessage checks what the SDK can check before a request: tags and
// attachments. Emails.Send, Emails.SendBatch and the builder share it.
func validateMessage(message SendMailRequest, prefix string) error {
	return validateContent(message.Tag, message.Tags, message.Attachments, prefix)
}

// validateContent checks the tags and attachments of a message or batch item.
func validateContent(tag Nullable[string], tags []MessageTagInput, attachments []MessageAttachmentInput, prefix string) error {
	at := func(field string) string {
		if prefix == "" {
			return field
		}
		return prefix + "." + field
	}
	legacy, hasLegacy := tag.Get()
	if err := validateTags(tags, hasLegacy && legacy != "", at("tags")); err != nil {
		return err
	}
	for i, attachment := range attachments {
		if attachment.Filename == "" {
			return &ClientValidationError{Field: fmt.Sprintf("%s[%d]", at("attachments"), i), Message: "an attachment needs a filename"}
		}
	}
	return nil
}

const maxBatchIdempotencyKeyLength = 255

// validateBatchIdempotencyKey applies the API's rule for a per-email batch
// idempotency key: 1-255 characters.
func validateBatchIdempotencyKey(key, field string) error {
	if n := utf8.RuneCountInString(key); n == 0 || n > maxBatchIdempotencyKeyLength {
		return &ClientValidationError{Field: field, Message: fmt.Sprintf("an idempotency key must be 1-%d characters", maxBatchIdempotencyKeyLength)}
	}
	return nil
}

// validateTags applies the API's tag rules: at most 20 tags (19 with the
// legacy tag), names ^[A-Za-z0-9_-]{1,32}$ that do not start with
// __lettermint and are unique (case-sensitive), values ^[A-Za-z0-9_-]{1,64}$.
func validateTags(tags []MessageTagInput, hasLegacyTag bool, field string) error {
	maximum := maxTags
	if hasLegacyTag {
		maximum--
	}
	if len(tags) > maximum {
		if hasLegacyTag {
			return &ClientValidationError{Field: field, Message: fmt.Sprintf("a legacy tag and no more than %d message tags are permitted", maximum)}
		}
		return &ClientValidationError{Field: field, Message: fmt.Sprintf("no more than %d message tags are permitted", maximum)}
	}
	names := make(map[string]bool, len(tags))
	for _, tag := range tags {
		switch {
		case !tagNamePattern.MatchString(tag.Name):
			return &ClientValidationError{Field: field, Message: "message tag names must match ^[A-Za-z0-9_-]{1,32}$"}
		case strings.HasPrefix(strings.ToLower(tag.Name), "__lettermint"):
			return &ClientValidationError{Field: field, Message: "message tag names must not start with __lettermint"}
		case !tagValuePattern.MatchString(tag.Value):
			return &ClientValidationError{Field: field, Message: "message tag values must match ^[A-Za-z0-9_-]{1,64}$"}
		case names[tag.Name]:
			return &ClientValidationError{Field: field, Message: "message tag names must be unique (case-sensitive)"}
		}
		names[tag.Name] = true
	}
	return nil
}
