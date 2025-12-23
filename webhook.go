package lettermint

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	// DefaultWebhookTolerance is the default timestamp tolerance for webhook verification.
	// Webhooks with timestamps older than this will be rejected.
	DefaultWebhookTolerance = 5 * time.Minute

	// HeaderSignature is the webhook signature header name.
	HeaderSignature = "X-Lettermint-Signature"

	// HeaderDelivery is the webhook delivery timestamp header name.
	HeaderDelivery = "X-Lettermint-Delivery"
)

// VerifyWebhook verifies a webhook signature and returns the parsed event.
//
// The signature format is: t={timestamp},v1={hmac_sha256_hex}
// The HMAC is computed over: {timestamp}.{payload}
//
// Parameters:
//   - signature: The X-Lettermint-Signature header value
//   - payload: The raw request body
//   - deliveryTimestamp: The X-Lettermint-Delivery header value (Unix timestamp), or 0 to skip cross-validation
//   - signingSecret: Your webhook signing secret from the Lettermint dashboard
//   - tolerance: Maximum age of the webhook timestamp (use DefaultWebhookTolerance)
//
// Returns the parsed webhook event or an error if verification fails.
func VerifyWebhook(signature string, payload []byte, deliveryTimestamp int64, signingSecret string, tolerance time.Duration) (*WebhookEvent, error) {
	if signingSecret == "" {
		return nil, fmt.Errorf("%w: signing secret is required", ErrInvalidWebhookSignature)
	}

	if signature == "" {
		return nil, fmt.Errorf("%w: signature is required", ErrInvalidWebhookSignature)
	}

	// Parse signature: t={timestamp},v1={hash}
	sigTimestamp, sigHash, err := parseSignature(signature)
	if err != nil {
		return nil, err
	}

	// Cross-validate timestamp if provided
	if deliveryTimestamp != 0 && deliveryTimestamp != sigTimestamp {
		return nil, fmt.Errorf("%w: timestamp mismatch between signature and delivery headers", ErrInvalidWebhookSignature)
	}

	// Check timestamp tolerance
	now := time.Now().Unix()
	diff := now - sigTimestamp
	if diff < 0 {
		diff = -diff
	}

	if time.Duration(diff)*time.Second > tolerance {
		return nil, fmt.Errorf("%w: timestamp %d is %d seconds old (tolerance: %v)",
			ErrWebhookTimestampExpired, sigTimestamp, diff, tolerance)
	}

	// Compute expected signature
	signedPayload := fmt.Sprintf("%d.%s", sigTimestamp, string(payload))
	expectedHash := computeHMAC([]byte(signedPayload), signingSecret)

	// Constant-time comparison to prevent timing attacks
	if !secureCompare(sigHash, expectedHash) {
		return nil, fmt.Errorf("%w: signature verification failed", ErrInvalidWebhookSignature)
	}

	// Parse webhook event
	var event WebhookEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		return nil, fmt.Errorf("failed to parse webhook payload: %w", err)
	}

	event.RawPayload = payload

	return &event, nil
}

// VerifyWebhookFromRequest verifies a webhook from an HTTP request.
//
// This is a convenience function that extracts the signature and payload
// from the request and calls VerifyWebhook.
//
// Note: This function reads and closes the request body.
//
// Example:
//
//	func webhookHandler(w http.ResponseWriter, r *http.Request) {
//	    event, err := lettermint.VerifyWebhookFromRequest(r, "your-signing-secret", lettermint.DefaultWebhookTolerance)
//	    if err != nil {
//	        http.Error(w, "Invalid signature", http.StatusUnauthorized)
//	        return
//	    }
//	    // Process event...
//	}
func VerifyWebhookFromRequest(r *http.Request, signingSecret string, tolerance time.Duration) (*WebhookEvent, error) {
	signature := r.Header.Get(HeaderSignature)
	if signature == "" {
		return nil, fmt.Errorf("%w: missing %s header", ErrInvalidWebhookSignature, HeaderSignature)
	}

	deliveryHeader := r.Header.Get(HeaderDelivery)
	var deliveryTimestamp int64
	if deliveryHeader != "" {
		var err error
		deliveryTimestamp, err = strconv.ParseInt(deliveryHeader, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("%w: invalid %s header value", ErrInvalidWebhookSignature, HeaderDelivery)
		}
	}

	payload, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read request body: %w", err)
	}

	return VerifyWebhook(signature, payload, deliveryTimestamp, signingSecret, tolerance)
}

// parseSignature parses the signature header value.
// Expected format: t={timestamp},v1={hash}
func parseSignature(signature string) (timestamp int64, hash string, err error) {
	parts := strings.Split(signature, ",")
	if len(parts) < 2 {
		return 0, "", fmt.Errorf("%w: invalid signature format, expected t={timestamp},v1={hash}", ErrInvalidWebhookSignature)
	}

	for _, part := range parts {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			continue
		}

		key := strings.TrimSpace(kv[0])
		value := strings.TrimSpace(kv[1])

		switch key {
		case "t":
			timestamp, err = strconv.ParseInt(value, 10, 64)
			if err != nil {
				return 0, "", fmt.Errorf("%w: invalid timestamp in signature", ErrInvalidWebhookSignature)
			}
		case "v1":
			hash = value
		}
	}

	if timestamp == 0 {
		return 0, "", fmt.Errorf("%w: missing timestamp (t=) in signature", ErrInvalidWebhookSignature)
	}
	if hash == "" {
		return 0, "", fmt.Errorf("%w: missing hash (v1=) in signature", ErrInvalidWebhookSignature)
	}

	return timestamp, hash, nil
}

// computeHMAC computes HMAC-SHA256 and returns the hex-encoded string.
func computeHMAC(data []byte, secret string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil))
}

// secureCompare performs constant-time string comparison to prevent timing attacks.
func secureCompare(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}
