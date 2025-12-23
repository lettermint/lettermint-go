package lettermint

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestEmailBuilder_FluentAPI(t *testing.T) {
	client, _ := New("test-token")
	ctx := context.Background()

	builder := client.Email(ctx).
		From("sender@example.com").
		To("recipient@example.com").
		CC("cc@example.com").
		BCC("bcc@example.com").
		ReplyTo("reply@example.com").
		Subject("Test Subject").
		HTML("<p>HTML Body</p>").
		Text("Text Body").
		Header("X-Custom", "value").
		Attach("file.txt", "Y29udGVudA==").
		AttachWithContentID("logo.png", "aW1hZ2U=", "logo").
		MetadataValue("key", "value").
		Tag("test-tag").
		Route("test-route").
		IdempotencyKey("test-key")

	// Verify all values were set
	if builder.payload.From != "sender@example.com" {
		t.Errorf("From = %v, want sender@example.com", builder.payload.From)
	}
	if len(builder.payload.To) != 1 || builder.payload.To[0] != "recipient@example.com" {
		t.Errorf("To = %v, want [recipient@example.com]", builder.payload.To)
	}
	if len(builder.payload.CC) != 1 || builder.payload.CC[0] != "cc@example.com" {
		t.Errorf("CC = %v, want [cc@example.com]", builder.payload.CC)
	}
	if len(builder.payload.BCC) != 1 || builder.payload.BCC[0] != "bcc@example.com" {
		t.Errorf("BCC = %v, want [bcc@example.com]", builder.payload.BCC)
	}
	if len(builder.payload.ReplyTo) != 1 || builder.payload.ReplyTo[0] != "reply@example.com" {
		t.Errorf("ReplyTo = %v, want [reply@example.com]", builder.payload.ReplyTo)
	}
	if builder.payload.Subject != "Test Subject" {
		t.Errorf("Subject = %v, want Test Subject", builder.payload.Subject)
	}
	if builder.payload.HTML != "<p>HTML Body</p>" {
		t.Errorf("HTML = %v, want <p>HTML Body</p>", builder.payload.HTML)
	}
	if builder.payload.Text != "Text Body" {
		t.Errorf("Text = %v, want Text Body", builder.payload.Text)
	}
	if builder.payload.Headers["X-Custom"] != "value" {
		t.Errorf("Headers[X-Custom] = %v, want value", builder.payload.Headers["X-Custom"])
	}
	if len(builder.payload.Attachments) != 2 {
		t.Errorf("Attachments count = %v, want 2", len(builder.payload.Attachments))
	}
	if builder.payload.Metadata["key"] != "value" {
		t.Errorf("Metadata[key] = %v, want value", builder.payload.Metadata["key"])
	}
	if builder.payload.Tag != "test-tag" {
		t.Errorf("Tag = %v, want test-tag", builder.payload.Tag)
	}
	if builder.payload.Route != "test-route" {
		t.Errorf("Route = %v, want test-route", builder.payload.Route)
	}
	if builder.idempotencyKey != "test-key" {
		t.Errorf("idempotencyKey = %v, want test-key", builder.idempotencyKey)
	}
}

func TestEmailBuilder_MultipleRecipients(t *testing.T) {
	client, _ := New("test-token")
	ctx := context.Background()

	builder := client.Email(ctx).
		To("user1@example.com", "user2@example.com").
		To("user3@example.com")

	if len(builder.payload.To) != 3 {
		t.Errorf("To count = %v, want 3", len(builder.payload.To))
	}
}

func TestEmailBuilder_Headers(t *testing.T) {
	client, _ := New("test-token")
	ctx := context.Background()

	builder := client.Email(ctx).
		Header("X-First", "first").
		Headers(map[string]string{
			"X-Second": "second",
			"X-Third":  "third",
		})

	if len(builder.payload.Headers) != 3 {
		t.Errorf("Headers count = %v, want 3", len(builder.payload.Headers))
	}
}

func TestEmailBuilder_Metadata(t *testing.T) {
	client, _ := New("test-token")
	ctx := context.Background()

	builder := client.Email(ctx).
		MetadataValue("key1", "value1").
		Metadata(map[string]string{
			"key2": "value2",
			"key3": "value3",
		})

	if len(builder.payload.Metadata) != 3 {
		t.Errorf("Metadata count = %v, want 3", len(builder.payload.Metadata))
	}
}

func TestEmailBuilder_Validate(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*EmailBuilder)
		wantErr string
	}{
		{
			name:    "missing from",
			setup:   func(b *EmailBuilder) {},
			wantErr: "from address is required",
		},
		{
			name: "missing to",
			setup: func(b *EmailBuilder) {
				b.From("sender@example.com")
			},
			wantErr: "at least one recipient is required",
		},
		{
			name: "missing subject",
			setup: func(b *EmailBuilder) {
				b.From("sender@example.com").To("recipient@example.com")
			},
			wantErr: "subject is required",
		},
		{
			name: "missing body",
			setup: func(b *EmailBuilder) {
				b.From("sender@example.com").To("recipient@example.com").Subject("Test")
			},
			wantErr: "either html or text body is required",
		},
		{
			name: "valid with HTML",
			setup: func(b *EmailBuilder) {
				b.From("sender@example.com").To("recipient@example.com").Subject("Test").HTML("<p>Body</p>")
			},
			wantErr: "",
		},
		{
			name: "valid with Text",
			setup: func(b *EmailBuilder) {
				b.From("sender@example.com").To("recipient@example.com").Subject("Test").Text("Body")
			},
			wantErr: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, _ := New("test-token")
			builder := client.Email(context.Background())
			tt.setup(builder)

			err := builder.validate()

			if tt.wantErr == "" {
				if err != nil {
					t.Errorf("validate() unexpected error = %v", err)
				}
			} else {
				if err == nil {
					t.Error("validate() expected error, got nil")
				} else if err.Error() != tt.wantErr {
					t.Errorf("validate() error = %v, want %v", err.Error(), tt.wantErr)
				}
			}
		})
	}
}

func TestEmailBuilder_Send_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		if r.Method != http.MethodPost {
			t.Errorf("Method = %v, want POST", r.Method)
		}
		if r.URL.Path != "/send" {
			t.Errorf("Path = %v, want /send", r.URL.Path)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Content-Type = %v, want application/json", r.Header.Get("Content-Type"))
		}
		if r.Header.Get("x-lettermint-token") != "test-token" {
			t.Errorf("x-lettermint-token = %v, want test-token", r.Header.Get("x-lettermint-token"))
		}

		// Verify body
		body, _ := io.ReadAll(r.Body)
		var payload emailPayload
		json.Unmarshal(body, &payload)

		if payload.From != "sender@example.com" {
			t.Errorf("payload.From = %v, want sender@example.com", payload.From)
		}

		// Send response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(SendResponse{
			MessageID: "msg_123",
			Status:    "queued",
		})
	}))
	defer server.Close()

	client, _ := New("test-token", WithBaseURL(server.URL))
	ctx := context.Background()

	resp, err := client.Email(ctx).
		From("sender@example.com").
		To("recipient@example.com").
		Subject("Test").
		HTML("<p>Body</p>").
		Send()

	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}

	if resp.MessageID != "msg_123" {
		t.Errorf("MessageID = %v, want msg_123", resp.MessageID)
	}
	if resp.Status != "queued" {
		t.Errorf("Status = %v, want queued", resp.Status)
	}
}

func TestEmailBuilder_Send_WithIdempotencyKey(t *testing.T) {
	var receivedKey string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedKey = r.Header.Get("Idempotency-Key")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(SendResponse{MessageID: "msg_123", Status: "queued"})
	}))
	defer server.Close()

	client, _ := New("test-token", WithBaseURL(server.URL))
	ctx := context.Background()

	_, err := client.Email(ctx).
		From("sender@example.com").
		To("recipient@example.com").
		Subject("Test").
		HTML("<p>Body</p>").
		IdempotencyKey("unique-key-123").
		Send()

	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}

	if receivedKey != "unique-key-123" {
		t.Errorf("Idempotency-Key = %v, want unique-key-123", receivedKey)
	}
}

func TestEmailBuilder_Send_ValidationError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnprocessableEntity)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message":    "Validation failed",
			"error_type": "validation_error",
			"errors": map[string][]string{
				"from": {"invalid email format"},
			},
		})
	}))
	defer server.Close()

	client, _ := New("test-token", WithBaseURL(server.URL))
	ctx := context.Background()

	_, err := client.Email(ctx).
		From("invalid-email").
		To("recipient@example.com").
		Subject("Test").
		HTML("<p>Body</p>").
		Send()

	if err == nil {
		t.Fatal("Send() expected error, got nil")
	}

	if !errors.Is(err, ErrValidation) {
		t.Errorf("Send() error should wrap ErrValidation, got %v", err)
	}

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatal("Send() error should be *APIError")
	}

	if apiErr.StatusCode != 422 {
		t.Errorf("APIError.StatusCode = %v, want 422", apiErr.StatusCode)
	}
}

func TestEmailBuilder_Send_UnauthorizedError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message": "Invalid API token",
		})
	}))
	defer server.Close()

	client, _ := New("invalid-token", WithBaseURL(server.URL))
	ctx := context.Background()

	_, err := client.Email(ctx).
		From("sender@example.com").
		To("recipient@example.com").
		Subject("Test").
		HTML("<p>Body</p>").
		Send()

	if err == nil {
		t.Fatal("Send() expected error, got nil")
	}

	if !errors.Is(err, ErrUnauthorized) {
		t.Errorf("Send() error should wrap ErrUnauthorized, got %v", err)
	}
}

func TestEmailBuilder_Send_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
	}))
	defer server.Close()

	client, _ := New("test-token", WithBaseURL(server.URL))
	ctx := context.Background()

	_, err := client.Email(ctx).
		From("sender@example.com").
		To("recipient@example.com").
		Subject("Test").
		HTML("<p>Body</p>").
		Send()

	if err == nil {
		t.Fatal("Send() expected error, got nil")
	}

	if !errors.Is(err, ErrServerError) {
		t.Errorf("Send() error should wrap ErrServerError, got %v", err)
	}
}

func TestEmailBuilder_Send_InvalidRequest(t *testing.T) {
	client, _ := New("test-token")
	ctx := context.Background()

	// Try to send without required fields
	_, err := client.Email(ctx).Send()

	if err == nil {
		t.Fatal("Send() expected error, got nil")
	}

	if !errors.Is(err, ErrInvalidRequest) {
		t.Errorf("Send() error should wrap ErrInvalidRequest, got %v", err)
	}
}
