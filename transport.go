package lettermint

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"iter"
	"math"
	"mime"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// IdempotentOption configures a call that accepts an Idempotency-Key:
// Emails.Send, Emails.SendBatch, EmailBuilder.Send and Messages.Process.
// Every RequestOption is also an IdempotentOption.
type IdempotentOption interface {
	applyIdempotent(*callOptions)
}

// RequestOption configures one call.
type RequestOption interface {
	IdempotentOption
	applyRequest(*callOptions)
}

type callOptions struct {
	timeout        time.Duration
	timeoutSet     bool
	idempotencyKey string
	keySet         bool
}

type timeoutOption time.Duration

func (o timeoutOption) applyRequest(c *callOptions)    { c.timeout, c.timeoutSet = time.Duration(o), true }
func (o timeoutOption) applyIdempotent(c *callOptions) { o.applyRequest(c) }

// WithRequestTimeout overrides the client's timeout for one call. Like the
// client's timeout, it covers the response headers and the body.
func WithRequestTimeout(timeout time.Duration) RequestOption {
	return timeoutOption(timeout)
}

type idempotencyKeyOption string

func (o idempotencyKeyOption) applyIdempotent(c *callOptions) {
	c.idempotencyKey, c.keySet = string(o), true
}

// WithIdempotencyKey sends the key as the Idempotency-Key header of this call
// only. The API processes a key once, so a retry with the same key does not
// send the email again. The SDK never stores the key and never retries.
func WithIdempotencyKey(key string) IdempotentOption {
	return idempotencyKeyOption(key)
}

func requestOptions(options []RequestOption) callOptions {
	var result callOptions
	for _, option := range options {
		if option != nil {
			option.applyRequest(&result)
		}
	}
	return result
}

func idempotentOptions(options []IdempotentOption) callOptions {
	var result callOptions
	for _, option := range options {
		if option != nil {
			option.applyIdempotent(&result)
		}
	}
	return result
}

// transport sends the requests of one client. It holds the tokens in
// unexported fields that render as [redacted].
type transport struct {
	sendingToken secret
	teamToken    secret
	baseURL      string
	timeout      time.Duration
	httpClient   *http.Client
	userAgent    string
}

// callArgs describes one call of an operation from the generated table.
type callArgs struct {
	// label is the public method name used in error messages: Domains.List.
	label string
	// path holds the path parameter values in the order of the table's PathParams.
	path []string
	// query is a pointer to a generated query struct, or nil.
	query any
	// cursor, when set, replaces the operation's cursor query parameter.
	cursor string
	body   any
	// auth overrides the table's auth surface.
	auth    authSurface
	options callOptions
}

type response struct {
	status int
	body   []byte
}

var errSDKTimeout = errors.New("lettermint: client timeout")

// authHeader returns the header that carries the token for the surface, or a
// *ConfigError that names the missing option.
func (t *transport) authHeader(label string, auth authSurface) (string, string, error) {
	useTeam := auth == authTeam || (auth == authEither && t.teamToken != "")
	if useTeam {
		if t.teamToken == "" {
			return "", "", &ConfigError{Message: label + " needs a team token; create the client with lettermint.WithTeamToken"}
		}
		return "Authorization", "Bearer " + t.teamToken.reveal(), nil
	}
	if t.sendingToken == "" {
		return "", "", &ConfigError{Message: label + " needs a sending token; create the client with lettermint.WithSendingToken"}
	}
	return "x-lettermint-token", t.sendingToken.reveal(), nil
}

func (t *transport) send(ctx context.Context, key operationKey, args callArgs) (*response, error) {
	if ctx == nil {
		return nil, &ConfigError{Message: args.label + " needs a non-nil context"}
	}
	op, ok := operations[key]
	if !ok {
		return nil, &ConfigError{Message: args.label + ": unknown operation " + string(key)}
	}
	auth := op.Auth
	if args.auth != "" {
		auth = args.auth
	}
	authName, authValue, err := t.authHeader(args.label, auth)
	if err != nil {
		return nil, err
	}
	path, err := buildPath(args.label, op, args.path)
	if err != nil {
		return nil, err
	}
	timeout := t.timeout
	if args.options.timeoutSet {
		if args.options.timeout <= 0 {
			return nil, &ConfigError{Message: args.label + ": WithRequestTimeout needs a positive duration"}
		}
		timeout = args.options.timeout
	}
	pairs, err := encodeQuery(args.query)
	if err != nil {
		return nil, &ConfigError{Message: args.label + ": " + err.Error()}
	}
	if args.cursor != "" && op.Pagination != nil {
		pairs = setQueryParam(pairs, op.Pagination.CursorParam, args.cursor)
	}
	target := t.baseURL + path
	if query := formEncode(pairs); query != "" {
		target += "?" + query
	}

	var body io.Reader
	if args.body != nil {
		var buffer bytes.Buffer
		encoder := json.NewEncoder(&buffer)
		encoder.SetEscapeHTML(false)
		if err := encoder.Encode(args.body); err != nil {
			return nil, &ClientValidationError{Field: "body", Message: "the request body cannot be encoded as JSON: " + err.Error()}
		}
		body = bytes.NewReader(bytes.TrimSuffix(buffer.Bytes(), []byte("\n")))
	}

	if err := ctx.Err(); err != nil {
		return nil, context.Cause(ctx)
	}
	callCtx, cancel := context.WithTimeoutCause(ctx, timeout, errSDKTimeout)
	defer cancel()
	request, err := http.NewRequestWithContext(callCtx, op.Method, target, body)
	if err != nil {
		return nil, &ConfigError{Message: args.label + ": the request URL is invalid"}
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("User-Agent", t.userAgent)
	if args.body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	if args.options.keySet {
		key := args.options.idempotencyKey
		if key == "" || strings.ContainsAny(key, "\r\n\x00") {
			return nil, &ClientValidationError{Field: "IdempotencyKey", Message: "the idempotency key must be a non-empty string without line breaks"}
		}
		request.Header.Set("Idempotency-Key", key)
	}
	request.Header.Set(authName, authValue)

	failed := func(err error) error {
		if ctx.Err() != nil {
			return context.Cause(ctx)
		}
		if errors.Is(context.Cause(callCtx), errSDKTimeout) {
			return &TimeoutError{Timeout: timeout}
		}
		return &ConnectionError{Err: err}
	}
	resp, err := t.httpClient.Do(request)
	if err != nil {
		return nil, failed(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 && resp.StatusCode < 400 {
		return nil, &RedirectError{Status: resp.StatusCode}
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, failed(err)
	}
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return &response{status: resp.StatusCode, body: data}, nil
	}
	if resp.StatusCode < 400 {
		return nil, &UnexpectedResponseError{Status: resp.StatusCode, BodyExcerpt: excerpt(data), Message: fmt.Sprintf("the Lettermint API answered with an unexpected HTTP status %d", resp.StatusCode)}
	}
	return nil, apiError(resp, data)
}

// callJSON calls an operation with a JSON success body and decodes it.
func callJSON[T any](ctx context.Context, t *transport, key operationKey, args callArgs) (*T, error) {
	resp, err := t.send(ctx, key, args)
	if err != nil {
		return nil, err
	}
	if len(bytes.TrimSpace(resp.body)) == 0 {
		return nil, &UnexpectedResponseError{Status: resp.status, Message: fmt.Sprintf("the Lettermint API answered with HTTP %d and an empty body where JSON was expected", resp.status)}
	}
	if !json.Valid(resp.body) {
		return nil, &UnexpectedResponseError{Status: resp.status, BodyExcerpt: excerpt(resp.body), Message: fmt.Sprintf("the Lettermint API answered with HTTP %d and a body that is not valid JSON", resp.status)}
	}
	var out T
	if err := json.Unmarshal(resp.body, &out); err != nil {
		return nil, &UnexpectedResponseError{Status: resp.status, BodyExcerpt: excerpt(resp.body), Message: fmt.Sprintf("the Lettermint API answered with HTTP %d and JSON that does not match %s: %v", resp.status, args.label, err)}
	}
	return &out, nil
}

// callText calls an operation with a text success body.
func callText(ctx context.Context, t *transport, key operationKey, args callArgs) (string, error) {
	resp, err := t.send(ctx, key, args)
	if err != nil {
		return "", err
	}
	return string(resp.body), nil
}

// callEmpty calls an operation without a success body (HTTP 204).
func callEmpty(ctx context.Context, t *transport, key operationKey, args callArgs) error {
	_, err := t.send(ctx, key, args)
	return err
}

// iterate follows next_cursor through every page of a cursor-paginated list.
// It stops at the last page, when the API repeats a cursor, at the first
// error (yielded once), or when the caller stops ranging.
func iterate[T any](ctx context.Context, t *transport, key operationKey, args callArgs) iter.Seq2[T, error] {
	return func(yield func(T, error) bool) {
		var zero T
		seen := map[string]bool{}
		for {
			page, err := callJSON[CursorPage[T]](ctx, t, key, args)
			if err == nil && page.Data == nil {
				err = &UnexpectedResponseError{Status: http.StatusOK, Message: args.label + ": the API returned a page without a data array"}
			}
			if err != nil {
				yield(zero, err)
				return
			}
			for _, item := range page.Data {
				if !yield(item, nil) {
					return
				}
			}
			if page.NextCursor == nil || *page.NextCursor == "" || seen[*page.NextCursor] {
				return
			}
			seen[*page.NextCursor] = true
			args.cursor = *page.NextCursor
		}
	}
}

// buildPath fills the operation's path parameters. Empty values, "." and ".."
// are rejected, so an ID can never change the path.
func buildPath(label string, op operation, values []string) (string, error) {
	if len(values) != len(op.PathParams) {
		return "", &ConfigError{Message: fmt.Sprintf("%s: expected %d path parameters", label, len(op.PathParams))}
	}
	path := op.Path
	for i, name := range op.PathParams {
		value := values[i]
		if value == "" || value == "." || value == ".." {
			return "", &ConfigError{Message: fmt.Sprintf("%s: %s must be a non-empty string other than \".\" and \"..\"", label, name)}
		}
		path = strings.Replace(path, "{"+name+"}", encodeURIComponent(value), 1)
	}
	return path, nil
}

// encodeURIComponent escapes a path parameter like JavaScript's
// encodeURIComponent, so that every SDK sends the same path.
func encodeURIComponent(value string) string {
	var builder strings.Builder
	for i := 0; i < len(value); i++ {
		c := value[i]
		if isAlphanumeric(c) || strings.IndexByte("-_.!~*'()", c) >= 0 {
			builder.WriteByte(c)
		} else {
			fmt.Fprintf(&builder, "%%%02X", c)
		}
	}
	return builder.String()
}

func isAlphanumeric(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')
}

// apiError maps an error response with a JSON or empty body to the error
// type for its status. A body that is not JSON is an UnexpectedResponseError.
func apiError(resp *http.Response, data []byte) error {
	status := resp.StatusCode
	base := APIError{Status: status}
	var errs map[string][]string
	if len(bytes.TrimSpace(data)) > 0 {
		if !json.Valid(data) {
			kind := ""
			if mediaType, _, err := mime.ParseMediaType(resp.Header.Get("Content-Type")); err == nil {
				kind = " (" + mediaType + ")"
			}
			return &UnexpectedResponseError{Status: status, BodyExcerpt: excerpt(data), Message: fmt.Sprintf("the Lettermint API answered with HTTP %d and a body that is not JSON%s", status, kind)}
		}
		base.Body = json.RawMessage(bytes.Clone(data))
		var object map[string]json.RawMessage
		if json.Unmarshal(data, &object) == nil {
			var detail struct {
				Code    *string         `json:"code"`
				Message *string         `json:"message"`
				Details json.RawMessage `json:"details"`
			}
			var code string
			if raw, ok := object["error"]; ok {
				if json.Unmarshal(raw, &detail) == nil && bytes.HasPrefix(bytes.TrimSpace(raw), []byte("{")) {
					if detail.Code != nil {
						base.Code = *detail.Code
					}
					if detail.Message != nil {
						base.Message = *detail.Message
					}
					if len(detail.Details) > 0 {
						_ = json.Unmarshal(detail.Details, &base.Details)
					}
				} else if json.Unmarshal(raw, &code) == nil {
					base.Code = code
				}
			}
			if base.Message == "" {
				var message string
				if raw, ok := object["message"]; ok && json.Unmarshal(raw, &message) == nil {
					base.Message = message
				}
			}
			if raw, ok := object["errors"]; ok {
				var fieldErrors map[string][]string
				if json.Unmarshal(raw, &fieldErrors) == nil {
					errs = fieldErrors
				}
			}
		}
	}
	if base.Message == "" {
		base.Message = statusText(resp)
	}
	switch {
	case status == http.StatusUnauthorized:
		return &AuthenticationError{base}
	case status == http.StatusForbidden:
		return &PermissionError{base}
	case status == http.StatusNotFound:
		return &NotFoundError{base}
	case status == http.StatusConflict:
		return &ConflictError{base}
	case status == http.StatusUnprocessableEntity:
		return &ValidationError{APIError: base, Errors: errs}
	case status == http.StatusTooManyRequests:
		return &RateLimitError{APIError: base, RetryAfter: parseRetryAfter(resp.Header.Get("Retry-After"), time.Now())}
	case status >= 500:
		return &ServerError{APIError: base, RetryAfter: parseRetryAfter(resp.Header.Get("Retry-After"), time.Now())}
	default:
		return &base
	}
}

func statusText(resp *http.Response) string {
	if text := strings.TrimSpace(strings.TrimPrefix(resp.Status, strconv.Itoa(resp.StatusCode))); text != "" {
		return text
	}
	if text := http.StatusText(resp.StatusCode); text != "" {
		return text
	}
	return "HTTP " + strconv.Itoa(resp.StatusCode)
}

// parseRetryAfter reads a Retry-After header in seconds or as an HTTP date.
func parseRetryAfter(value string, now time.Time) *time.Duration {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	if seconds, err := strconv.ParseUint(value, 10, 32); err == nil {
		d := time.Duration(seconds) * time.Second
		return &d
	}
	date, err := http.ParseTime(value)
	if err != nil {
		return nil
	}
	d := time.Duration(math.Max(0, math.Ceil(date.Sub(now).Seconds()))) * time.Second
	return &d
}
