package lettermint

import (
	"context"
	"fmt"
	"regexp"
	"strings"
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
// its Build method.
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
	at := func(field string) string {
		if prefix == "" {
			return field
		}
		return prefix + "." + field
	}
	legacy, hasLegacy := message.Tag.Get()
	if err := validateTags(message.Tags, hasLegacy && legacy != "", at("tags")); err != nil {
		return err
	}
	for i, attachment := range message.Attachments {
		if attachment.Filename == "" {
			return &ClientValidationError{Field: fmt.Sprintf("%s[%d]", at("attachments"), i), Message: "an attachment needs a filename"}
		}
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
