package lettermint

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"runtime"
	"strings"
)

func (c *Client) doJSON(ctx context.Context, method, path string, query map[string]string, payload interface{}, out interface{}) error {
	body, err := requestBody(payload)
	if err != nil {
		return err
	}

	req, err := c.newRequest(ctx, method, path, query, body)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("%w: %v", ErrTimeout, err)
		}
		if ctx.Err() == context.Canceled {
			return fmt.Errorf("request canceled: %w", err)
		}
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}
	if resp.StatusCode >= 400 {
		return parseAPIError(resp.StatusCode, responseBody)
	}
	if out == nil || len(responseBody) == 0 {
		return nil
	}
	if err := json.Unmarshal(responseBody, out); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}
	return nil
}

func (c *Client) doRaw(ctx context.Context, method, path string, query map[string]string) (string, error) {
	req, err := c.newRequest(ctx, method, path, query, nil)
	if err != nil {
		return "", err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return "", fmt.Errorf("%w: %v", ErrTimeout, err)
		}
		if ctx.Err() == context.Canceled {
			return "", fmt.Errorf("request canceled: %w", err)
		}
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}
	if resp.StatusCode >= 400 {
		return "", parseAPIError(resp.StatusCode, responseBody)
	}
	return strings.TrimSpace(string(responseBody)), nil
}

func (c *Client) newRequest(ctx context.Context, method, path string, query map[string]string, body io.Reader) (*http.Request, error) {
	endpoint, err := c.url(path, query)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, method, endpoint, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", fmt.Sprintf("Lettermint/%s (Go; %s)", Version, runtime.Version()))
	if c.authScheme == authSchemeBearer {
		req.Header.Set("Authorization", "Bearer "+c.apiToken)
	} else {
		req.Header.Set("x-lettermint-token", c.apiToken)
	}

	return req, nil
}

func (c *Client) url(path string, query map[string]string) (string, error) {
	base, err := url.Parse(strings.TrimSuffix(c.baseURL, "/") + "/")
	if err != nil {
		return "", err
	}
	endpoint, err := url.Parse(strings.TrimPrefix(path, "/"))
	if err != nil {
		return "", err
	}
	resolved := base.ResolveReference(endpoint)
	values := resolved.Query()
	for key, value := range query {
		values.Set(key, value)
	}
	resolved.RawQuery = values.Encode()
	return resolved.String(), nil
}

func requestBody(payload interface{}) (io.Reader, error) {
	if payload == nil {
		return nil, nil
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request payload: %w", err)
	}
	return bytes.NewReader(data), nil
}
