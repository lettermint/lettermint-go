package lettermint

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// Error is implemented by every error the SDK returns. Use errors.As to check
// for it, or for one of the concrete types:
//
//	var apiErr *lettermint.APIError
//	if errors.As(err, &apiErr) {
//		log.Println(apiErr.Status, apiErr.Code, apiErr.Message)
//	}
//
// No SDK error contains request headers or tokens.
type Error interface {
	error
	lettermintError()
}

// ConfigError reports that the client was configured or called incorrectly:
// a missing or unrecognised token, a token the called method cannot use, an
// invalid option or an invalid path parameter. It is returned before any
// request is made.
type ConfigError struct {
	Message string
}

func (e *ConfigError) Error() string  { return "lettermint: " + e.Message }
func (*ConfigError) lettermintError() {}

// ClientValidationError reports that the SDK rejected a request before
// sending it, for example invalid message tags. Unlike ValidationError, the
// API never saw the request.
type ClientValidationError struct {
	// Field is the offending field, for example "tags" or "messages[2].tags".
	Field   string
	Message string
}

func (e *ClientValidationError) Error() string {
	if e.Field == "" {
		return "lettermint: " + e.Message
	}
	return fmt.Sprintf("lettermint: %s: %s", e.Field, e.Message)
}
func (*ClientValidationError) lettermintError() {}

// APIError reports that the API answered with an error status (4xx or 5xx)
// and a JSON or empty body. The common statuses have their own types, which
// all unwrap to *APIError: AuthenticationError (401), PermissionError (403),
// NotFoundError (404), ConflictError (409), ValidationError (422),
// RateLimitError (429) and ServerError (5xx).
type APIError struct {
	// Status is the HTTP status code.
	Status int
	// Code is the machine-readable error code from {"error": {"code"}}, or a
	// string "error" field, when the API sent one.
	Code string
	// Message is the API's error message, or the HTTP status text.
	Message string
	// Details is the decoded {"error": {"details"}} value, when the API sent one.
	Details any
	// Body is the raw JSON error body, or nil when it was empty. Decode it into
	// ApiErrorBody or ValidationErrorBody when you need more.
	Body json.RawMessage
}

func (e *APIError) Error() string {
	if e.Code != "" {
		return fmt.Sprintf("lettermint: %s (HTTP %d, %s)", e.Message, e.Status, e.Code)
	}
	return fmt.Sprintf("lettermint: %s (HTTP %d)", e.Message, e.Status)
}
func (*APIError) lettermintError() {}

// AuthenticationError is HTTP 401: the token is missing, invalid or revoked.
type AuthenticationError struct{ APIError }

// Unwrap returns the *APIError, so errors.As(err, &apiErr) matches.
func (e *AuthenticationError) Unwrap() error { return &e.APIError }

// PermissionError is HTTP 403: the token may not perform this action, or the
// plan lacks the feature.
type PermissionError struct{ APIError }

// Unwrap returns the *APIError, so errors.As(err, &apiErr) matches.
func (e *PermissionError) Unwrap() error { return &e.APIError }

// NotFoundError is HTTP 404: the resource does not exist or is not visible to
// the token.
type NotFoundError struct{ APIError }

// Unwrap returns the *APIError, so errors.As(err, &apiErr) matches.
func (e *NotFoundError) Unwrap() error { return &e.APIError }

// ConflictError is HTTP 409: the request conflicts with the current state, for
// example an Idempotency-Key reused with a different body.
type ConflictError struct{ APIError }

// Unwrap returns the *APIError, so errors.As(err, &apiErr) matches.
func (e *ConflictError) Unwrap() error { return &e.APIError }

// ValidationError is HTTP 422: the API rejected the request data.
type ValidationError struct {
	APIError
	// Errors holds the field errors from Laravel's {"message", "errors"} body,
	// when the API sent them.
	Errors map[string][]string
}

// Unwrap returns the *APIError, so errors.As(err, &apiErr) matches.
func (e *ValidationError) Unwrap() error { return &e.APIError }

// RateLimitError is HTTP 429: too many requests.
type RateLimitError struct {
	APIError
	// RetryAfter is how long to wait, from the Retry-After header (seconds or an
	// HTTP date). It is nil when the API did not send a usable value.
	RetryAfter *time.Duration
}

// Unwrap returns the *APIError, so errors.As(err, &apiErr) matches.
func (e *RateLimitError) Unwrap() error { return &e.APIError }

// ServerError is HTTP 5xx with a JSON or empty body.
type ServerError struct {
	APIError
	// RetryAfter is how long to wait, from the Retry-After header (seconds or an
	// HTTP date). It is nil when the API did not send a usable value.
	RetryAfter *time.Duration
}

// Unwrap returns the *APIError, so errors.As(err, &apiErr) matches.
func (e *ServerError) Unwrap() error { return &e.APIError }

// TimeoutError reports that the request did not complete within the timeout.
// The timeout covers the response headers and the body. The API may still
// have processed the request; retry a send with the same idempotency key.
//
// A deadline or cancellation of the caller's context is returned as the
// context's error instead.
type TimeoutError struct {
	// Timeout is the timeout that expired.
	Timeout time.Duration
}

func (e *TimeoutError) Error() string {
	return fmt.Sprintf("lettermint: the request to the Lettermint API timed out after %s", e.Timeout)
}
func (*TimeoutError) lettermintError() {}

// ConnectionError reports that the request could not be sent or the
// connection failed (DNS, TLS, refused, reset).
type ConnectionError struct {
	Err error
}

func (e *ConnectionError) Error() string {
	return "lettermint: could not reach the Lettermint API: " + e.Err.Error()
}

// Unwrap returns the underlying network error.
func (e *ConnectionError) Unwrap() error  { return e.Err }
func (*ConnectionError) lettermintError() {}

// UnexpectedResponseError reports a response the SDK could not decode: an
// empty or non-JSON body where JSON was expected, or an error status with a
// body that is not JSON, such as a proxy's HTML error page.
type UnexpectedResponseError struct {
	// Status is the HTTP status code.
	Status int
	// BodyExcerpt holds the first 200 characters of the response body.
	BodyExcerpt string
	Message     string
}

func (e *UnexpectedResponseError) Error() string  { return "lettermint: " + e.Message }
func (*UnexpectedResponseError) lettermintError() {}

// RedirectError reports that the API answered with a redirect (3xx). The SDK
// never follows redirects, so that tokens are not sent to another location.
type RedirectError struct {
	// Status is the HTTP status code.
	Status int
}

func (e *RedirectError) Error() string {
	return fmt.Sprintf("lettermint: the Lettermint API answered with a redirect (HTTP %d). Redirects are not followed; check the base URL", e.Status)
}
func (*RedirectError) lettermintError() {}

// WebhookVerificationReason says why a webhook delivery failed verification.
type WebhookVerificationReason string

// Reasons of a WebhookVerificationError.
const (
	WebhookSignatureHeaderMissing    WebhookVerificationReason = "signature_header_missing"
	WebhookSignatureHeaderMalformed  WebhookVerificationReason = "signature_header_malformed"
	WebhookDeliveryHeaderMissing     WebhookVerificationReason = "delivery_header_missing"
	WebhookDeliveryTimestampMismatch WebhookVerificationReason = "delivery_timestamp_mismatch"
	WebhookTimestampOutOfTolerance   WebhookVerificationReason = "timestamp_out_of_tolerance"
	WebhookSignatureMismatch         WebhookVerificationReason = "signature_mismatch"
	WebhookBodyInvalid               WebhookVerificationReason = "body_invalid"
	WebhookPayloadInvalid            WebhookVerificationReason = "payload_invalid"
)

// WebhookVerificationError reports that a webhook delivery could not be
// verified. Reject the request and do not process its payload.
type WebhookVerificationError struct {
	Reason  WebhookVerificationReason
	Message string
}

func (e *WebhookVerificationError) Error() string  { return "lettermint: " + e.Message }
func (*WebhookVerificationError) lettermintError() {}

func excerpt(body []byte) string {
	text := strings.ToValidUTF8(string(body), "�")
	runes := []rune(text)
	if len(runes) > 200 {
		return string(runes[:200]) + "…"
	}
	return text
}
