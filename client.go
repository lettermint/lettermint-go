package lettermint

import (
	"context"
	"fmt"
	"iter"
	"log/slog"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

const (
	// DefaultBaseURL is the Lettermint API base URL.
	DefaultBaseURL = "https://api.lettermint.co/v1"

	// DefaultTimeout is the default request timeout. It covers the response
	// headers and the body.
	DefaultTimeout = 30 * time.Second
)

var (
	// Team API tokens: ApiToken::TEAM_PREFIX in the Lettermint backend. Checked
	// first, because every team token also starts with lm_.
	teamTokenPattern = regexp.MustCompile(`^lm_team_[0-9A-Za-z]+$`)
	// Project sending tokens: ApiToken::PROJECT_PREFIX.
	sendingTokenPattern = regexp.MustCompile(`^lm_[0-9A-Za-z]+$`)
	// Characters allowed in a token, so that it is a valid HTTP header value.
	headerSafe = regexp.MustCompile(`^[\x21-\x7e]+$`)
)

// Client is the Lettermint client. Create it once with New or NewFromToken
// and share it: it holds no per-request or per-message state and is safe for
// concurrent use.
//
// Emails uses the sending token (x-lettermint-token). Every other part uses
// the team token (Authorization: Bearer). Ping, Messages.Reschedule and
// Messages.Cancel use the team token when it is set, otherwise the sending
// token. A part never falls back to the other token: a missing token is a
// *ConfigError that names the option, returned before any request.
//
// Printing, logging or encoding a Client never shows its tokens.
type Client struct {
	t *transport

	// Emails sends email. Needs a sending token.
	Emails *EmailsService
	// Domains manages sending domains. Needs a team token.
	Domains *DomainsService
	// Messages reads sent and received messages. Needs a team token.
	Messages *MessagesService
	// Projects manages projects and their report forwarding. Needs a team token.
	Projects *ProjectsService
	// Routes manages the routes of a project. Needs a team token.
	Routes *RoutesService
	// Stats reads sending statistics. Needs a team token.
	Stats *StatsService
	// Suppressions manages the suppression list. Needs a team token.
	Suppressions *SuppressionsService
	// Team manages the team and its members. Needs a team token.
	Team *TeamService
	// Webhooks manages webhook endpoints and their deliveries. Needs a team
	// token. To verify incoming deliveries, use NewWebhook.
	Webhooks *WebhooksService
}

// Option configures a Client.
type Option func(*clientConfig)

type clientConfig struct {
	sendingToken *string
	teamToken    *string
	baseURL      string
	timeout      time.Duration
	httpClient   *http.Client
}

// WithSendingToken sets the project sending token (lm_...). Emails uses it.
func WithSendingToken(token string) Option {
	return func(c *clientConfig) { c.sendingToken = &token }
}

// WithTeamToken sets the team API token (lm_team_...). Every part except
// Emails uses it.
func WithTeamToken(token string) Option {
	return func(c *clientConfig) { c.teamToken = &token }
}

// WithBaseURL sets the API base URL. Default DefaultBaseURL.
func WithBaseURL(baseURL string) Option {
	return func(c *clientConfig) { c.baseURL = baseURL }
}

// WithTimeout sets the timeout of every request, covering the response
// headers and the body. Default DefaultTimeout. WithRequestTimeout overrides
// it for one call, and a deadline on the call's context also applies.
func WithTimeout(timeout time.Duration) Option {
	return func(c *clientConfig) { c.timeout = timeout }
}

// WithHTTPClient sets the HTTP client, for example to use a proxy or a custom
// transport. The SDK copies the client and never changes yours. The copy
// never follows redirects; the client's Timeout still applies in addition to
// WithTimeout.
func WithHTTPClient(client *http.Client) Option {
	return func(c *clientConfig) { c.httpClient = client }
}

// New creates a client. Pass WithSendingToken, WithTeamToken or both:
//
//	client, err := lettermint.New(lettermint.WithSendingToken(os.Getenv("LETTERMINT_PROJECT_TOKEN")))
//
// It returns a *ConfigError when no token is set or an option is invalid.
func New(options ...Option) (*Client, error) {
	config := clientConfig{baseURL: DefaultBaseURL, timeout: DefaultTimeout}
	for _, option := range options {
		if option != nil {
			option(&config)
		}
	}
	return newClient(config)
}

// NewFromToken creates a client from one token and picks its type by the
// format: lm_team_ followed by letters and digits is a team token, and lm_
// followed by letters and digits is a sending token. Any other value, such as
// an SSO verification token (lm_sso_...), is a *ConfigError; use New with
// WithSendingToken or WithTeamToken for it. Errors never contain the token.
//
//	client, err := lettermint.NewFromToken(os.Getenv("LETTERMINT_TOKEN"), lettermint.WithTimeout(10*time.Second))
func NewFromToken(token string, options ...Option) (*Client, error) {
	config := clientConfig{baseURL: DefaultBaseURL, timeout: DefaultTimeout}
	for _, option := range options {
		if option != nil {
			option(&config)
		}
	}
	if config.sendingToken != nil || config.teamToken != nil {
		return nil, &ConfigError{Message: "NewFromToken takes the token as its argument; do not combine it with WithSendingToken or WithTeamToken"}
	}
	switch {
	case teamTokenPattern.MatchString(token):
		config.teamToken = &token
	case sendingTokenPattern.MatchString(token):
		config.sendingToken = &token
	default:
		return nil, &ConfigError{Message: "unrecognised token format; pass WithSendingToken or WithTeamToken to New instead"}
	}
	return newClient(config)
}

func newClient(config clientConfig) (*Client, error) {
	sending, err := checkToken("WithSendingToken", config.sendingToken)
	if err != nil {
		return nil, err
	}
	team, err := checkToken("WithTeamToken", config.teamToken)
	if err != nil {
		return nil, err
	}
	if sending == "" && team == "" {
		return nil, &ConfigError{Message: "pass WithSendingToken, WithTeamToken or both"}
	}
	baseURL, err := checkBaseURL(config.baseURL)
	if err != nil {
		return nil, err
	}
	if config.timeout <= 0 {
		return nil, &ConfigError{Message: "WithTimeout needs a positive duration"}
	}
	t := &transport{
		sendingToken: sending,
		teamToken:    team,
		baseURL:      baseURL,
		timeout:      config.timeout,
		httpClient:   noRedirectClient(config.httpClient),
		userAgent:    userAgent(),
	}
	return &Client{
		t:            t,
		Emails:       &EmailsService{service{t, "EmailsService"}},
		Domains:      &DomainsService{service{t, "DomainsService"}},
		Messages:     &MessagesService{service{t, "MessagesService"}},
		Projects:     &ProjectsService{service: service{t, "ProjectsService"}, ReportForwarding: &ReportForwardingService{service{t, "ReportForwardingService"}}},
		Routes:       &RoutesService{service{t, "RoutesService"}},
		Stats:        &StatsService{service{t, "StatsService"}},
		Suppressions: &SuppressionsService{service{t, "SuppressionsService"}},
		Team:         &TeamService{service: service{t, "TeamService"}, Members: &TeamMembersService{service{t, "TeamMembersService"}}},
		Webhooks:     &WebhooksService{service: service{t, "WebhooksService"}, Deliveries: &WebhookDeliveriesService{service{t, "WebhookDeliveriesService"}}},
	}, nil
}

// checkToken validates an explicitly set token. The error names the option,
// never the token.
func checkToken(option string, token *string) (secret, error) {
	if token == nil {
		return "", nil
	}
	if *token == "" {
		return "", &ConfigError{Message: option + " needs a non-empty token"}
	}
	if !headerSafe.MatchString(*token) {
		return "", &ConfigError{Message: option + ": the token contains whitespace or characters that are not allowed in an HTTP header"}
	}
	return secret(*token), nil
}

func checkBaseURL(value string) (string, error) {
	parsed, err := url.Parse(value)
	if err != nil || (parsed.Scheme != "https" && parsed.Scheme != "http") || parsed.Host == "" {
		return "", &ConfigError{Message: "WithBaseURL needs an absolute http(s) URL"}
	}
	if parsed.User != nil || parsed.RawQuery != "" || parsed.ForceQuery || parsed.Fragment != "" {
		return "", &ConfigError{Message: "WithBaseURL must not contain credentials, a query string or a fragment"}
	}
	return strings.TrimRight(value, "/"), nil
}

// noRedirectClient copies the caller's client (or a new one) and makes it
// return redirects instead of following them. The caller's client is not
// changed.
func noRedirectClient(base *http.Client) *http.Client {
	var client http.Client
	if base != nil {
		client = *base
	}
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	return &client
}

// Ping checks the configured token: GET /ping returns "pong". It uses the
// team token when it is set, otherwise the sending token.
func (c *Client) Ping(ctx context.Context, options ...RequestOption) (string, error) {
	text, err := callText(ctx, c.t, opPing, callArgs{label: "Ping", options: requestOptions(options)})
	return strings.TrimSpace(text), err
}

// Analytics queries email analytics. Needs a team token.
func (c *Client) Analytics(ctx context.Context, query AnalyticsQuery, options ...RequestOption) (*AnalyticsResponse, error) {
	return callJSON[AnalyticsResponse](ctx, c.t, opQueryAnalytics, callArgs{label: "Analytics", body: query, options: requestOptions(options)})
}

// AnalyticsPages queries email analytics and follows Pagination.NextCursor,
// yielding one whole response per request. Each response carries the next
// page of Data.Breakdown with its own Meta and Pagination. Needs a team token.
//
// A cursor expires 60 seconds after its response, so request the next page
// promptly; an expired cursor yields a *ValidationError. It requests the next
// page only when the loop gets to it, and yields an error at most once. The
// query passed in is not changed.
//
//	for page, err := range client.AnalyticsPages(ctx, query) {
//		if err != nil {
//			return err
//		}
//		rows = append(rows, page.Data.Breakdown...)
//	}
func (c *Client) AnalyticsPages(ctx context.Context, query AnalyticsQuery, options ...RequestOption) iter.Seq2[*AnalyticsResponse, error] {
	callOptions := requestOptions(options)
	return func(yield func(*AnalyticsResponse, error) bool) {
		args := callArgs{label: "AnalyticsPages", body: query, options: callOptions}
		seen := map[string]bool{}
		if query.Cursor != nil {
			seen[*query.Cursor] = true
		}
		for {
			page, err := callJSON[AnalyticsResponse](ctx, c.t, opQueryAnalytics, args)
			if err != nil {
				yield(nil, err)
				return
			}
			if !yield(page, nil) {
				return
			}
			next := page.Pagination.NextCursor
			if next == nil || *next == "" || seen[*next] {
				return
			}
			seen[*next] = true
			body := query
			body.Cursor = Ptr(*next)
			args.body = body
		}
	}
}

// BlockedFileTypes lists the file extensions and MIME types that cannot be
// attached. Needs a team token.
func (c *Client) BlockedFileTypes(ctx context.Context, options ...RequestOption) (*BlockedFileTypes, error) {
	return callJSON[BlockedFileTypes](ctx, c.t, opListBlockedFileTypes, callArgs{label: "BlockedFileTypes", options: requestOptions(options)})
}

func (c Client) view() view {
	if c.t == nil {
		return view{name: "Client"}
	}
	return view{name: "Client", fields: []viewField{
		{"BaseURL", c.t.baseURL},
		{"Timeout", c.t.timeout.String()},
		{"SendingToken", c.t.sendingToken.shown()},
		{"TeamToken", c.t.teamToken.shown()},
	}}
}

// String describes the client without credentials: tokens show as [redacted].
func (c Client) String() string { return c.view().String() }

// GoString describes the client without credentials.
func (c Client) GoString() string { return c.view().String() }

// Format prints the client without credentials for every verb, including %+v
// and %#v.
func (c Client) Format(f fmt.State, verb rune) { c.view().format(f, verb) }

// LogValue implements slog.LogValuer without credentials.
func (c Client) LogValue() slog.Value { return c.view().logValue() }

// MarshalJSON encodes the configuration without credentials.
func (c Client) MarshalJSON() ([]byte, error) { return c.view().marshalJSON() }
