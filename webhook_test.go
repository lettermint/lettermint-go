package lettermint

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// generateTestSignature creates a valid signature for testing
func generateTestSignature(payload string, secret string, timestamp int64) string {
	signedPayload := fmt.Sprintf("%d.%s", timestamp, payload)
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(signedPayload))
	hash := hex.EncodeToString(h.Sum(nil))
	return fmt.Sprintf("t=%d,v1=%s", timestamp, hash)
}

func TestVerifyWebhook_Success(t *testing.T) {
	payload := `{"id":"wh_123","event":"message.delivered","timestamp":1234567890,"data":{"message_id":"msg_123","recipient":"user@example.com"}}`
	secret := "test-secret"
	timestamp := time.Now().Unix()
	signature := generateTestSignature(payload, secret, timestamp)

	event, err := VerifyWebhook(signature, []byte(payload), 0, secret, DefaultWebhookTolerance)
	if err != nil {
		t.Fatalf("VerifyWebhook() error = %v", err)
	}

	if event == nil {
		t.Fatal("VerifyWebhook() returned nil event")
	}

	if event.ID != "wh_123" {
		t.Errorf("event.ID = %v, want wh_123", event.ID)
	}

	if event.Event != "message.delivered" {
		t.Errorf("event.Event = %v, want message.delivered", event.Event)
	}

	if event.Data.MessageID != "msg_123" {
		t.Errorf("event.Data.MessageID = %v, want msg_123", event.Data.MessageID)
	}

	if event.Data.Recipient != "user@example.com" {
		t.Errorf("event.Data.Recipient = %v, want user@example.com", event.Data.Recipient)
	}

	if event.RawPayload == nil {
		t.Error("event.RawPayload should not be nil")
	}
}

func TestVerifyWebhook_WithDeliveryTimestamp(t *testing.T) {
	payload := `{"id":"wh_123","event":"message.delivered"}`
	secret := "test-secret"
	timestamp := time.Now().Unix()
	signature := generateTestSignature(payload, secret, timestamp)

	// Matching delivery timestamp should work
	event, err := VerifyWebhook(signature, []byte(payload), timestamp, secret, DefaultWebhookTolerance)
	if err != nil {
		t.Fatalf("VerifyWebhook() error = %v", err)
	}
	if event == nil {
		t.Fatal("VerifyWebhook() returned nil event")
	}

	// Mismatched delivery timestamp should fail
	_, err = VerifyWebhook(signature, []byte(payload), timestamp+100, secret, DefaultWebhookTolerance)
	if err == nil {
		t.Fatal("VerifyWebhook() expected error for mismatched timestamp")
	}
	if !errors.Is(err, ErrInvalidWebhookSignature) {
		t.Errorf("VerifyWebhook() error should wrap ErrInvalidWebhookSignature, got %v", err)
	}
}

func TestVerifyWebhook_InvalidSignature(t *testing.T) {
	payload := `{"id":"wh_123","event":"message.delivered"}`
	secret := "test-secret"
	timestamp := time.Now().Unix()

	// Wrong signature
	signature := fmt.Sprintf("t=%d,v1=invalid_hash", timestamp)

	_, err := VerifyWebhook(signature, []byte(payload), 0, secret, DefaultWebhookTolerance)
	if err == nil {
		t.Fatal("VerifyWebhook() expected error for invalid signature")
	}

	if !errors.Is(err, ErrInvalidWebhookSignature) {
		t.Errorf("VerifyWebhook() error should wrap ErrInvalidWebhookSignature, got %v", err)
	}
}

func TestVerifyWebhook_WrongSecret(t *testing.T) {
	payload := `{"id":"wh_123","event":"message.delivered"}`
	timestamp := time.Now().Unix()
	signature := generateTestSignature(payload, "correct-secret", timestamp)

	_, err := VerifyWebhook(signature, []byte(payload), 0, "wrong-secret", DefaultWebhookTolerance)
	if err == nil {
		t.Fatal("VerifyWebhook() expected error for wrong secret")
	}

	if !errors.Is(err, ErrInvalidWebhookSignature) {
		t.Errorf("VerifyWebhook() error should wrap ErrInvalidWebhookSignature, got %v", err)
	}
}

func TestVerifyWebhook_ExpiredTimestamp(t *testing.T) {
	payload := `{"id":"wh_123","event":"message.delivered"}`
	secret := "test-secret"
	// Timestamp from 10 minutes ago
	timestamp := time.Now().Add(-10 * time.Minute).Unix()
	signature := generateTestSignature(payload, secret, timestamp)

	_, err := VerifyWebhook(signature, []byte(payload), 0, secret, DefaultWebhookTolerance)
	if err == nil {
		t.Fatal("VerifyWebhook() expected error for expired timestamp")
	}

	if !errors.Is(err, ErrWebhookTimestampExpired) {
		t.Errorf("VerifyWebhook() error should wrap ErrWebhookTimestampExpired, got %v", err)
	}
}

func TestVerifyWebhook_FutureTimestamp(t *testing.T) {
	payload := `{"id":"wh_123","event":"message.delivered"}`
	secret := "test-secret"
	// Timestamp from 10 minutes in the future
	timestamp := time.Now().Add(10 * time.Minute).Unix()
	signature := generateTestSignature(payload, secret, timestamp)

	_, err := VerifyWebhook(signature, []byte(payload), 0, secret, DefaultWebhookTolerance)
	if err == nil {
		t.Fatal("VerifyWebhook() expected error for future timestamp")
	}

	if !errors.Is(err, ErrWebhookTimestampExpired) {
		t.Errorf("VerifyWebhook() error should wrap ErrWebhookTimestampExpired, got %v", err)
	}
}

func TestVerifyWebhook_CustomTolerance(t *testing.T) {
	payload := `{"id":"wh_123","event":"message.delivered"}`
	secret := "test-secret"
	// Timestamp from 2 minutes ago
	timestamp := time.Now().Add(-2 * time.Minute).Unix()
	signature := generateTestSignature(payload, secret, timestamp)

	// Should fail with 1 minute tolerance
	_, err := VerifyWebhook(signature, []byte(payload), 0, secret, 1*time.Minute)
	if err == nil {
		t.Fatal("VerifyWebhook() expected error with 1 minute tolerance")
	}

	// Should succeed with 5 minute tolerance
	event, err := VerifyWebhook(signature, []byte(payload), 0, secret, 5*time.Minute)
	if err != nil {
		t.Fatalf("VerifyWebhook() error = %v with 5 minute tolerance", err)
	}
	if event == nil {
		t.Fatal("VerifyWebhook() returned nil event")
	}
}

func TestVerifyWebhook_MalformedSignature(t *testing.T) {
	tests := []struct {
		name      string
		signature string
	}{
		{"empty", ""},
		{"no parts", "invalid"},
		{"missing hash", "t=1234567890"},
		{"missing timestamp", "v1=abc123"},
		{"invalid timestamp", "t=invalid,v1=abc123"},
		{"wrong format", "timestamp=1234567890,hash=abc123"},
	}

	payload := `{"id":"wh_123"}`
	secret := "test-secret"

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := VerifyWebhook(tt.signature, []byte(payload), 0, secret, DefaultWebhookTolerance)
			if err == nil {
				t.Fatal("VerifyWebhook() expected error for malformed signature")
			}

			if !errors.Is(err, ErrInvalidWebhookSignature) {
				t.Errorf("VerifyWebhook() error should wrap ErrInvalidWebhookSignature, got %v", err)
			}
		})
	}
}

func TestVerifyWebhook_EmptySecret(t *testing.T) {
	payload := `{"id":"wh_123"}`
	timestamp := time.Now().Unix()
	signature := fmt.Sprintf("t=%d,v1=abc123", timestamp)

	_, err := VerifyWebhook(signature, []byte(payload), 0, "", DefaultWebhookTolerance)
	if err == nil {
		t.Fatal("VerifyWebhook() expected error for empty secret")
	}

	if !errors.Is(err, ErrInvalidWebhookSignature) {
		t.Errorf("VerifyWebhook() error should wrap ErrInvalidWebhookSignature, got %v", err)
	}
}

func TestVerifyWebhook_InvalidJSON(t *testing.T) {
	payload := `invalid json`
	secret := "test-secret"
	timestamp := time.Now().Unix()
	signature := generateTestSignature(payload, secret, timestamp)

	_, err := VerifyWebhook(signature, []byte(payload), 0, secret, DefaultWebhookTolerance)
	if err == nil {
		t.Fatal("VerifyWebhook() expected error for invalid JSON")
	}

	// Should not be a signature error, but a parsing error
	if errors.Is(err, ErrInvalidWebhookSignature) {
		t.Error("VerifyWebhook() error should not be ErrInvalidWebhookSignature for JSON error")
	}
}

func TestVerifyWebhookFromRequest_Success(t *testing.T) {
	payload := `{"id":"wh_123","event":"message.delivered","data":{"message_id":"msg_123"}}`
	secret := "test-secret"
	timestamp := time.Now().Unix()
	signature := generateTestSignature(payload, secret, timestamp)

	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(payload))
	req.Header.Set(HeaderSignature, signature)
	req.Header.Set(HeaderDelivery, fmt.Sprintf("%d", timestamp))

	event, err := VerifyWebhookFromRequest(req, secret, DefaultWebhookTolerance)
	if err != nil {
		t.Fatalf("VerifyWebhookFromRequest() error = %v", err)
	}

	if event == nil {
		t.Fatal("VerifyWebhookFromRequest() returned nil event")
	}

	if event.ID != "wh_123" {
		t.Errorf("event.ID = %v, want wh_123", event.ID)
	}
}

func TestVerifyWebhookFromRequest_MissingSignatureHeader(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(`{"id":"wh_123"}`))
	// No signature header

	_, err := VerifyWebhookFromRequest(req, "test-secret", DefaultWebhookTolerance)
	if err == nil {
		t.Fatal("VerifyWebhookFromRequest() expected error for missing signature header")
	}

	if !errors.Is(err, ErrInvalidWebhookSignature) {
		t.Errorf("VerifyWebhookFromRequest() error should wrap ErrInvalidWebhookSignature, got %v", err)
	}
}

func TestVerifyWebhookFromRequest_InvalidDeliveryHeader(t *testing.T) {
	payload := `{"id":"wh_123"}`
	secret := "test-secret"
	timestamp := time.Now().Unix()
	signature := generateTestSignature(payload, secret, timestamp)

	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(payload))
	req.Header.Set(HeaderSignature, signature)
	req.Header.Set(HeaderDelivery, "not-a-number")

	_, err := VerifyWebhookFromRequest(req, secret, DefaultWebhookTolerance)
	if err == nil {
		t.Fatal("VerifyWebhookFromRequest() expected error for invalid delivery header")
	}

	if !errors.Is(err, ErrInvalidWebhookSignature) {
		t.Errorf("VerifyWebhookFromRequest() error should wrap ErrInvalidWebhookSignature, got %v", err)
	}
}

func TestVerifyWebhookFromRequest_BodyReadError(t *testing.T) {
	secret := "test-secret"
	timestamp := time.Now().Unix()
	signature := fmt.Sprintf("t=%d,v1=abc123", timestamp)

	// Create a request with a body that will error on read
	req := httptest.NewRequest(http.MethodPost, "/webhook", &errorReader{})
	req.Header.Set(HeaderSignature, signature)

	_, err := VerifyWebhookFromRequest(req, secret, DefaultWebhookTolerance)
	if err == nil {
		t.Fatal("VerifyWebhookFromRequest() expected error for body read error")
	}
}

// errorReader is a reader that always returns an error
type errorReader struct{}

func (e *errorReader) Read(p []byte) (n int, err error) {
	return 0, io.ErrUnexpectedEOF
}

func TestSecureCompare(t *testing.T) {
	tests := []struct {
		name string
		a    string
		b    string
		want bool
	}{
		{"equal strings", "abc123", "abc123", true},
		{"different strings", "abc123", "abc456", false},
		{"different lengths", "abc", "abcdef", false},
		{"empty strings", "", "", true},
		{"one empty", "abc", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := secureCompare(tt.a, tt.b); got != tt.want {
				t.Errorf("secureCompare() = %v, want %v", got, tt.want)
			}
		})
	}
}
