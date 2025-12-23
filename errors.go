package lettermint

import (
	"errors"
	"fmt"
)

// Sentinel errors for type checking with errors.Is()
var (
	// ErrInvalidAPIToken indicates the API token is missing or invalid.
	ErrInvalidAPIToken = errors.New("lettermint: invalid or missing API token")

	// ErrInvalidRequest indicates request validation failed before sending.
	ErrInvalidRequest = errors.New("lettermint: invalid request")

	// ErrUnauthorized indicates authentication failed (HTTP 401).
	ErrUnauthorized = errors.New("lettermint: unauthorized")

	// ErrValidation indicates validation error from API (HTTP 422).
	ErrValidation = errors.New("lettermint: validation error")

	// ErrRateLimited indicates rate limit exceeded (HTTP 429).
	ErrRateLimited = errors.New("lettermint: rate limit exceeded")

	// ErrServerError indicates server error (HTTP 5xx).
	ErrServerError = errors.New("lettermint: server error")

	// ErrTimeout indicates request timeout.
	ErrTimeout = errors.New("lettermint: request timeout")

	// ErrInvalidWebhookSignature indicates webhook signature verification failed.
	ErrInvalidWebhookSignature = errors.New("lettermint: invalid webhook signature")

	// ErrWebhookTimestampExpired indicates webhook timestamp is outside tolerance window.
	ErrWebhookTimestampExpired = errors.New("lettermint: webhook timestamp outside tolerance window")
)

// APIError represents an error response from the Lettermint API.
type APIError struct {
	// StatusCode is the HTTP status code returned by the API.
	StatusCode int

	// Message is the error message from the API.
	Message string

	// ErrorType is the specific error type (e.g., "validation_error").
	ErrorType string

	// Errors contains field-specific validation errors.
	Errors map[string][]string

	// ResponseBody is the raw response body for debugging.
	ResponseBody string
}

// Error implements the error interface.
func (e *APIError) Error() string {
	if e.ErrorType != "" {
		return fmt.Sprintf("lettermint: API error (%d): %s [%s]", e.StatusCode, e.Message, e.ErrorType)
	}
	return fmt.Sprintf("lettermint: API error (%d): %s", e.StatusCode, e.Message)
}

// Unwrap returns the underlying sentinel error for use with errors.Is().
func (e *APIError) Unwrap() error {
	switch e.StatusCode {
	case 400:
		return ErrInvalidRequest
	case 401:
		return ErrUnauthorized
	case 422:
		return ErrValidation
	case 429:
		return ErrRateLimited
	default:
		if e.StatusCode >= 500 {
			return ErrServerError
		}
		return nil
	}
}
