package lettermint

import (
	"net/http"
	"time"
)

// Option is a functional option for configuring the Client.
type Option func(*Client)

// WithBaseURL sets a custom base URL for the Lettermint API.
//
// By default, the client uses https://api.lettermint.co/v1.
// Use this option for testing or if you have a custom API endpoint.
func WithBaseURL(baseURL string) Option {
	return func(c *Client) {
		c.baseURL = baseURL
	}
}

// WithTimeout sets the HTTP client timeout for all requests.
//
// By default, the timeout is 30 seconds.
func WithTimeout(timeout time.Duration) Option {
	return func(c *Client) {
		c.httpClient.Timeout = timeout
	}
}

// WithHTTPClient sets a custom HTTP client.
//
// Use this option to configure custom transport settings, proxies,
// or to inject a mock client for testing.
//
// Note: If you set a custom HTTP client, the WithTimeout option
// will modify the timeout of the provided client.
func WithHTTPClient(client *http.Client) Option {
	return func(c *Client) {
		c.httpClient = client
	}
}
