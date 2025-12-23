package lettermint

import (
	"context"
	"net/http"
	"time"
)

const (
	// DefaultBaseURL is the default Lettermint API base URL.
	DefaultBaseURL = "https://api.lettermint.co/v1"

	// DefaultTimeout is the default HTTP client timeout.
	DefaultTimeout = 30 * time.Second

	// Version is the SDK version.
	Version = "1.0.0"
)

// Client is the main Lettermint SDK client.
//
// The client is safe for concurrent use by multiple goroutines.
// Create a new client using the New function.
type Client struct {
	apiToken   string
	baseURL    string
	httpClient *http.Client
}

// New creates a new Lettermint client with the given API token and options.
//
// The API token is required and can be obtained from the Lettermint dashboard.
// Returns an error if the API token is empty.
//
// Example:
//
//	client, err := lettermint.New("your-api-token")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// With options:
//
//	client, err := lettermint.New("your-api-token",
//	    lettermint.WithBaseURL("https://custom-api.example.com"),
//	    lettermint.WithTimeout(60*time.Second),
//	)
func New(apiToken string, opts ...Option) (*Client, error) {
	if apiToken == "" {
		return nil, ErrInvalidAPIToken
	}

	c := &Client{
		apiToken: apiToken,
		baseURL:  DefaultBaseURL,
		httpClient: &http.Client{
			Timeout: DefaultTimeout,
		},
	}

	for _, opt := range opts {
		opt(c)
	}

	return c, nil
}

// Email creates a new email builder for composing and sending emails.
//
// The builder uses a fluent interface for constructing emails.
// Call Send() on the builder to send the email.
//
// Example:
//
//	resp, err := client.Email(ctx).
//	    From("sender@example.com").
//	    To("recipient@example.com").
//	    Subject("Hello").
//	    HTML("<p>World</p>").
//	    Send()
func (c *Client) Email(ctx context.Context) *EmailBuilder {
	return &EmailBuilder{
		client: c,
		ctx:    ctx,
		payload: &emailPayload{
			To:      []string{},
			CC:      []string{},
			BCC:     []string{},
			ReplyTo: []string{},
		},
	}
}
