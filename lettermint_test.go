package lettermint

import (
	"context"
	"net/http"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name      string
		apiToken  string
		opts      []Option
		wantErr   error
		wantURL   string
	}{
		{
			name:     "valid token",
			apiToken: "test-token",
			wantErr:  nil,
			wantURL:  DefaultBaseURL,
		},
		{
			name:     "empty token",
			apiToken: "",
			wantErr:  ErrInvalidAPIToken,
		},
		{
			name:     "custom base URL",
			apiToken: "test-token",
			opts:     []Option{WithBaseURL("https://custom.api.com")},
			wantErr:  nil,
			wantURL:  "https://custom.api.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := New(tt.apiToken, tt.opts...)

			if tt.wantErr != nil {
				if err != tt.wantErr {
					t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Errorf("New() unexpected error = %v", err)
				return
			}

			if client == nil {
				t.Error("New() returned nil client")
				return
			}

			if tt.wantURL != "" && client.baseURL != tt.wantURL {
				t.Errorf("New() baseURL = %v, want %v", client.baseURL, tt.wantURL)
			}
		})
	}
}

func TestWithTimeout(t *testing.T) {
	client, err := New("test-token", WithTimeout(60*time.Second))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if client.httpClient.Timeout != 60*time.Second {
		t.Errorf("WithTimeout() timeout = %v, want %v", client.httpClient.Timeout, 60*time.Second)
	}
}

func TestWithHTTPClient(t *testing.T) {
	customClient := &http.Client{
		Timeout: 120 * time.Second,
	}

	client, err := New("test-token", WithHTTPClient(customClient))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if client.httpClient != customClient {
		t.Error("WithHTTPClient() did not set custom client")
	}
}

func TestClientEmail(t *testing.T) {
	client, err := New("test-token")
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	ctx := context.Background()
	builder := client.Email(ctx)

	if builder == nil {
		t.Error("Email() returned nil builder")
	}

	if builder.client != client {
		t.Error("Email() builder has wrong client reference")
	}

	if builder.payload == nil {
		t.Error("Email() builder has nil payload")
	}
}
