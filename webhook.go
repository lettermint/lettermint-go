package lettermint

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	// DefaultWebhookTolerance is the default maximum difference between the
	// signed timestamp and the current time, in either direction.
	DefaultWebhookTolerance = 5 * time.Minute

	// DefaultWebhookMaxBodyBytes is the largest body VerifyRequest reads.
	DefaultWebhookMaxBodyBytes int64 = 50 << 20

	// HeaderSignature is the webhook signature header: t=<unix seconds>,v1=<hex HMAC-SHA256>.
	HeaderSignature = "X-Lettermint-Signature"

	// HeaderDelivery is the webhook delivery header. It holds the signed
	// timestamp; it is not a delivery ID.
	HeaderDelivery = "X-Lettermint-Delivery"
)

var (
	timestampPattern = regexp.MustCompile(`^[0-9]+$`)
	sha256HexPattern = regexp.MustCompile(`^[0-9a-fA-F]{64}$`)
)

// maxSafeInteger is JavaScript's Number.MAX_SAFE_INTEGER, the largest
// timestamp every Lettermint SDK accepts.
const maxSafeInteger = 1<<53 - 1

// Webhook verifies Lettermint webhook deliveries: an HMAC-SHA256 signature
// over "<t>." + the raw body, keyed with the endpoint's signing secret
// (whsec_..., used as is), compared in constant time.
//
// Create it once with NewWebhook and share it; it is safe for concurrent use.
// Printing, logging or encoding a Webhook never shows the secret.
type Webhook struct {
	secret       secret
	tolerance    time.Duration
	now          func() time.Time
	maxBodyBytes int64
}

// WebhookOption configures a Webhook.
type WebhookOption func(*webhookConfig)

type webhookConfig struct {
	tolerance    time.Duration
	now          func() time.Time
	maxBodyBytes int64
}

// WithTolerance sets the maximum difference between the signed timestamp and
// the current time, in either direction, compared in whole seconds. Default
// DefaultWebhookTolerance. Zero accepts only the current second; it does not
// disable the check.
func WithTolerance(tolerance time.Duration) WebhookOption {
	return func(c *webhookConfig) { c.tolerance = tolerance }
}

// WithClock sets the clock used for the timestamp check, for tests.
func WithClock(now func() time.Time) WebhookOption {
	return func(c *webhookConfig) { c.now = now }
}

// WithMaxBodyBytes sets the largest body VerifyRequest reads. Default
// DefaultWebhookMaxBodyBytes.
func WithMaxBodyBytes(limit int64) WebhookOption {
	return func(c *webhookConfig) { c.maxBodyBytes = limit }
}

// NewWebhook creates a verifier for the endpoint's signing secret (whsec_...).
// Use the secret exactly as Lettermint shows it, including the whsec_ prefix.
// An empty secret or a negative tolerance is a *ConfigError.
func NewWebhook(signingSecret string, options ...WebhookOption) (*Webhook, error) {
	config := webhookConfig{tolerance: DefaultWebhookTolerance, now: time.Now, maxBodyBytes: DefaultWebhookMaxBodyBytes}
	for _, option := range options {
		if option != nil {
			option(&config)
		}
	}
	if signingSecret == "" {
		return nil, &ConfigError{Message: "the webhook signing secret must be a non-empty string"}
	}
	if config.tolerance < 0 {
		return nil, &ConfigError{Message: "WithTolerance needs a non-negative duration"}
	}
	if config.now == nil {
		return nil, &ConfigError{Message: "WithClock needs a non-nil clock"}
	}
	if config.maxBodyBytes <= 0 {
		return nil, &ConfigError{Message: "WithMaxBodyBytes needs a positive limit"}
	}
	return &Webhook{secret: secret(signingSecret), tolerance: config.tolerance, now: config.now, maxBodyBytes: config.maxBodyBytes}, nil
}

// WebhookPayload is a verified webhook delivery.
type WebhookPayload struct {
	// ID is the delivery ID.
	ID string `json:"id"`
	// Event is the event name, for example message.delivered. Unknown events
	// pass through unchanged.
	Event WebhookEvent `json:"event"`
	// Timestamp is when the event occurred, ISO 8601.
	Timestamp string `json:"timestamp"`
	// Data is the event data as raw JSON; see DecodeData.
	Data json.RawMessage `json:"data"`
	// Raw is the verified request body.
	Raw json.RawMessage `json:"-"`
}

// DecodeData decodes the event data into v.
func (p *WebhookPayload) DecodeData(v any) error {
	return json.Unmarshal(p.Data, v)
}

// Verify verifies a delivery from its raw body and request headers, and
// returns the decoded payload. It requires X-Lettermint-Signature and
// X-Lettermint-Delivery, which must equal the signed timestamp. Header names
// are matched case-insensitively, also in an http.Header built by hand.
//
// Pass the body exactly as received: the signature covers the bytes, so
// decoding and re-encoding the JSON breaks it. Every failure is a
// *WebhookVerificationError with a Reason.
func (w *Webhook) Verify(rawBody []byte, headers http.Header) (*WebhookPayload, error) {
	signature, count := readHeader(headers, HeaderSignature)
	if count == 0 {
		return nil, verificationError(WebhookSignatureHeaderMissing, "the X-Lettermint-Signature header is missing")
	}
	if count > 1 {
		return nil, verificationError(WebhookSignatureHeaderMalformed, "the request has more than one X-Lettermint-Signature header")
	}
	delivery, count := readHeader(headers, HeaderDelivery)
	if count == 0 {
		return nil, verificationError(WebhookDeliveryHeaderMissing, "the X-Lettermint-Delivery header is missing")
	}
	if count > 1 {
		return nil, verificationError(WebhookDeliveryTimestampMismatch, "the request has more than one X-Lettermint-Delivery header")
	}
	return w.verify(rawBody, signature, &delivery)
}

// VerifyRequest reads the request body (at most WithMaxBodyBytes) and
// verifies it with the request headers, like Verify. It replaces r.Body with
// a reader over the same bytes, so the handler can read the body again.
func (w *Webhook) VerifyRequest(r *http.Request) (*WebhookPayload, error) {
	if r == nil || r.Body == nil {
		return nil, verificationError(WebhookBodyInvalid, "the request has no body")
	}
	limited := io.LimitReader(r.Body, w.maxBodyBytes+1)
	body, err := io.ReadAll(limited)
	_ = r.Body.Close()
	r.Body = io.NopCloser(bytes.NewReader(body))
	if err != nil {
		return nil, verificationError(WebhookBodyInvalid, "the request body could not be read")
	}
	if int64(len(body)) > w.maxBodyBytes {
		return nil, verificationError(WebhookBodyInvalid, fmt.Sprintf("the request body is larger than %d bytes", w.maxBodyBytes))
	}
	return w.Verify(body, r.Header)
}

// VerifySignature verifies the raw body against an X-Lettermint-Signature
// value, for setups where the headers are not at hand. When timestamp (the
// X-Lettermint-Delivery value) is not empty, it must equal the signed
// timestamp.
func (w *Webhook) VerifySignature(rawBody []byte, signatureHeader, timestamp string) (*WebhookPayload, error) {
	if timestamp == "" {
		return w.verify(rawBody, signatureHeader, nil)
	}
	return w.verify(rawBody, signatureHeader, &timestamp)
}

func (w *Webhook) verify(rawBody []byte, signatureHeader string, delivery *string) (*WebhookPayload, error) {
	if w == nil {
		return nil, &ConfigError{Message: "create the Webhook with lettermint.NewWebhook"}
	}
	if strings.TrimSpace(signatureHeader) == "" {
		return nil, verificationError(WebhookSignatureHeaderMissing, "the X-Lettermint-Signature header is missing")
	}
	timestamp, signatures, err := parseSignatureHeader(signatureHeader)
	if err != nil {
		return nil, err
	}
	if delivery != nil && strings.TrimSpace(*delivery) != timestamp {
		return nil, verificationError(WebhookDeliveryTimestampMismatch, "the X-Lettermint-Delivery header does not match the signed timestamp")
	}
	if len(rawBody) == 0 {
		return nil, verificationError(WebhookBodyInvalid, "the raw request body is empty")
	}
	signed, _ := strconv.ParseInt(timestamp, 10, 64)
	difference := w.now().Unix() - signed
	if difference < 0 {
		difference = -difference
	}
	if time.Duration(difference)*time.Second > w.tolerance {
		return nil, verificationError(WebhookTimestampOutOfTolerance, "the signed timestamp is outside the allowed tolerance")
	}

	mac := hmac.New(sha256.New, []byte(w.secret.reveal()))
	mac.Write([]byte(timestamp + "."))
	mac.Write(rawBody)
	expected := mac.Sum(nil)
	matched := false
	for _, candidate := range signatures {
		if hmac.Equal(candidate, expected) {
			matched = true
		}
	}
	if !matched {
		return nil, verificationError(WebhookSignatureMismatch, "the webhook signature does not match")
	}

	trimmed := bytes.TrimSpace(rawBody)
	if !json.Valid(trimmed) {
		return nil, verificationError(WebhookPayloadInvalid, "the webhook payload is not valid JSON")
	}
	if !bytes.HasPrefix(trimmed, []byte("{")) {
		return nil, verificationError(WebhookPayloadInvalid, "the webhook payload is not a JSON object")
	}
	var payload WebhookPayload
	if err := json.Unmarshal(trimmed, &payload); err != nil {
		return nil, verificationError(WebhookPayloadInvalid, "the webhook payload does not have the expected fields")
	}
	payload.Raw = json.RawMessage(bytes.Clone(rawBody))
	return &payload, nil
}

// parseSignatureHeader reads t=<unix>,v1=<hex>[,v1=<hex>...]. Unknown schemes
// (v2=...) are ignored; any matching v1 is valid, which allows key rotation.
func parseSignatureHeader(header string) (string, [][]byte, error) {
	malformed := func(detail string) error {
		return verificationError(WebhookSignatureHeaderMalformed, "the signature header is malformed: "+detail)
	}
	for i := 0; i < len(header); i++ {
		if header[i] < 0x20 || header[i] > 0x7e {
			return "", nil, malformed("it contains non-ASCII or control characters")
		}
	}
	timestamp := ""
	var signatures [][]byte
	for _, part := range strings.Split(header, ",") {
		key, value, found := strings.Cut(strings.TrimSpace(part), "=")
		if !found {
			continue
		}
		switch {
		case key == "t":
			if timestamp != "" {
				return "", nil, malformed("it has more than one timestamp")
			}
			number, err := strconv.ParseInt(value, 10, 64)
			if !timestampPattern.MatchString(value) || err != nil || number > maxSafeInteger {
				return "", nil, malformed("the timestamp is not a number of seconds")
			}
			timestamp = value
		case key == "v1" && sha256HexPattern.MatchString(value):
			decoded, _ := hex.DecodeString(value)
			signatures = append(signatures, decoded)
		}
	}
	if timestamp == "" {
		return "", nil, malformed("the timestamp (t=) is missing")
	}
	if len(signatures) == 0 {
		return "", nil, malformed("no v1 signature is present")
	}
	return timestamp, signatures, nil
}

// readHeader returns the value of a header and how many values it has,
// matching names case-insensitively over every key of the map.
func readHeader(headers http.Header, name string) (string, int) {
	var values []string
	for key, list := range headers {
		if strings.EqualFold(key, name) {
			values = append(values, list...)
		}
	}
	if len(values) == 0 {
		return "", 0
	}
	return values[0], len(values)
}

func verificationError(reason WebhookVerificationReason, message string) error {
	return &WebhookVerificationError{Reason: reason, Message: message}
}

func (w Webhook) view() view {
	return view{name: "Webhook", fields: []viewField{{"Tolerance", w.tolerance.String()}, {"Secret", w.secret.shown()}}}
}

// String describes the verifier without the secret.
func (w Webhook) String() string { return w.view().String() }

// GoString describes the verifier without the secret.
func (w Webhook) GoString() string { return w.view().String() }

// Format prints the verifier without the secret for every verb.
func (w Webhook) Format(f fmt.State, verb rune) { w.view().format(f, verb) }

// LogValue implements slog.LogValuer without the secret.
func (w Webhook) LogValue() slog.Value { return w.view().logValue() }

// MarshalJSON encodes the tolerance only.
func (w Webhook) MarshalJSON() ([]byte, error) { return w.view().marshalJSON() }
