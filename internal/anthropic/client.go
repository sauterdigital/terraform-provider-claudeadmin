package anthropic

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	apiVersionHeader = "2023-06-01"
	defaultTimeout   = 30 * time.Second

	// Retry tuning for 429s. Conservative defaults — adjust if telemetry warrants.
	maxRetries    = 5
	baseBackoff   = 500 * time.Millisecond
	maxBackoff    = 30 * time.Second
	backoffJitter = 250 * time.Millisecond
)

type Client struct {
	baseURL       string
	apiKey        string // x-api-key auth (Admin API key sk-ant-admin01-...)
	oauthToken    string // Bearer auth (preferred for newer endpoints; some require it)
	complianceKey string // x-api-key auth for Compliance API (sk-ant-api01-...)
	userAgent     string
	httpClient    *http.Client

	// sleeper exists so tests can swap in a fake clock without waiting real
	// seconds during retry exercises. nil means use time.Sleep.
	sleeper func(time.Duration)
}

// NewClient builds a client authenticated with an Admin API key (x-api-key).
// Use SetOAuthToken to also enable Bearer auth — when both are set, Bearer
// takes precedence (the doc's modern preferred pattern), and a handful of
// newer endpoints (Service Accounts, Federation, MCP Tunnels) reject the
// x-api-key path outright.
func NewClient(baseURL, apiKey, version string) *Client {
	return &Client{
		baseURL:    strings.TrimRight(baseURL, "/"),
		apiKey:     apiKey,
		userAgent:  fmt.Sprintf("terraform-provider-anthropic/%s", version),
		httpClient: &http.Client{Timeout: defaultTimeout},
	}
}

// SetOAuthToken enables OAuth Bearer auth on the client. When set, every
// outgoing request uses `Authorization: Bearer <token>` instead of x-api-key.
// Pass an empty string to clear.
func (c *Client) SetOAuthToken(token string) {
	c.oauthToken = token
}

// HasOAuth reports whether the client has an OAuth token configured. Used by
// endpoints that REQUIRE Bearer auth to fail-fast at the provider layer with
// a clear message instead of letting the API return a 401.
func (c *Client) HasOAuth() bool { return c.oauthToken != "" }

// SetComplianceKey enables Compliance API auth on the client. Compliance
// endpoints (/v1/compliance/*) accept only their dedicated key
// (sk-ant-api01-...), passed in the same x-api-key header as Admin API keys
// but with a distinct format. Pass an empty string to clear.
func (c *Client) SetComplianceKey(key string) { c.complianceKey = key }

// HasCompliance reports whether the client has a Compliance API key
// configured. Compliance data sources fail-fast when this is false rather
// than relying on the API's 401.
func (c *Client) HasCompliance() bool { return c.complianceKey != "" }

// complianceAuthKey is the context marker requesting compliance-key auth
// on the outgoing request.
type complianceAuthKey struct{}

// WithComplianceAuth returns a child context that forces the outgoing
// request to use the Compliance API key instead of the Admin key / OAuth
// token. Used by /v1/compliance/* endpoints.
func WithComplianceAuth(ctx context.Context) context.Context {
	return context.WithValue(ctx, complianceAuthKey{}, true)
}

type APIError struct {
	StatusCode int
	Type       string `json:"type"`
	Message    string `json:"message"`
}

func (e *APIError) Error() string {
	return fmt.Sprintf("anthropic API error %d (%s): %s", e.StatusCode, e.Type, e.Message)
}

func IsNotFound(err error) bool {
	var apiErr *APIError
	return errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound
}

func (c *Client) sleep(d time.Duration) {
	if c.sleeper != nil {
		c.sleeper(d)
		return
	}
	time.Sleep(d)
}

// betaHeadersKey is the context key under which callers attach
// `anthropic-beta` header values to be added to outgoing requests.
// Used by beta endpoints (MCP Tunnels) to declare the beta version they
// need without forcing every other call site to thread an extra param.
type betaHeadersKey struct{}

// WithBetaHeaders returns a child context that adds the given beta versions
// to any request made through Client.do(). Multiple values may be passed;
// each gets its own `anthropic-beta` header on the outgoing request.
func WithBetaHeaders(ctx context.Context, versions ...string) context.Context {
	if len(versions) == 0 {
		return ctx
	}
	return context.WithValue(ctx, betaHeadersKey{}, versions)
}

func (c *Client) do(ctx context.Context, method, path string, body, out any) error {
	var bodyBytes []byte
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal request body: %w", err)
		}
		bodyBytes = b
	}

	for attempt := 0; attempt <= maxRetries; attempt++ {
		// New reader per attempt — bytes.Reader is single-use after first read.
		var reqBody io.Reader
		if bodyBytes != nil {
			reqBody = bytes.NewReader(bodyBytes)
		}

		req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reqBody)
		if err != nil {
			return fmt.Errorf("build request: %w", err)
		}
		c.setAuthHeaders(req, ctx)
		req.Header.Set("anthropic-version", apiVersionHeader)
		req.Header.Set("content-type", "application/json")
		req.Header.Set("user-agent", c.userAgent)
		if v := ctx.Value(betaHeadersKey{}); v != nil {
			if versions, ok := v.([]string); ok {
				for _, b := range versions {
					req.Header.Add("anthropic-beta", b)
				}
			}
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return fmt.Errorf("request: %w", err)
		}

		respBody, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			return fmt.Errorf("read response: %w", readErr)
		}

		if resp.StatusCode == http.StatusTooManyRequests {
			if attempt >= maxRetries {
				return &APIError{
					StatusCode: resp.StatusCode,
					Type:       "rate_limit",
					Message:    fmt.Sprintf("max retries exceeded after %d attempts; still rate-limited", maxRetries+1),
				}
			}
			wait := retryAfterFromResponse(resp, attempt)
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}
			c.sleep(wait)
			continue
		}

		if resp.StatusCode >= 400 {
			apiErr := &APIError{StatusCode: resp.StatusCode}
			var wrap struct {
				Error APIError `json:"error"`
			}
			if json.Unmarshal(respBody, &wrap) == nil && wrap.Error.Message != "" {
				apiErr.Type = wrap.Error.Type
				apiErr.Message = wrap.Error.Message
			} else {
				apiErr.Message = strings.TrimSpace(string(respBody))
			}
			return apiErr
		}

		if out != nil && len(respBody) > 0 {
			if err := json.Unmarshal(respBody, out); err != nil {
				return fmt.Errorf("decode response: %w", err)
			}
		}
		return nil
	}
	return fmt.Errorf("unreachable: retry loop exited without a result")
}

// setAuthHeaders prefers Bearer when both are present — that's the doc's
// modern preferred pattern, and several newer endpoints (Service Accounts,
// Federation, MCP Tunnels) reject x-api-key outright. Compliance API
// endpoints override via WithComplianceAuth(ctx) — they REQUIRE the
// compliance key in x-api-key and reject Admin / OAuth auth.
func (c *Client) setAuthHeaders(req *http.Request, ctx context.Context) {
	if v, ok := ctx.Value(complianceAuthKey{}).(bool); ok && v && c.complianceKey != "" {
		req.Header.Set("x-api-key", c.complianceKey)
		return
	}
	if c.oauthToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.oauthToken)
		return
	}
	if c.apiKey != "" {
		req.Header.Set("x-api-key", c.apiKey)
	}
}

// retryAfterFromResponse picks a wait duration: prefer the API's Retry-After
// header (we support the seconds-integer form, which is what the docs show),
// otherwise fall back to exponential backoff with a small jitter.
func retryAfterFromResponse(resp *http.Response, attempt int) time.Duration {
	if h := resp.Header.Get("Retry-After"); h != "" {
		if secs, err := strconv.Atoi(strings.TrimSpace(h)); err == nil && secs > 0 {
			d := time.Duration(secs) * time.Second
			if d > maxBackoff {
				return maxBackoff
			}
			return d
		}
	}
	expo := baseBackoff << attempt
	if expo > maxBackoff {
		expo = maxBackoff
	}
	jitter := time.Duration(rand.Int63n(int64(backoffJitter)))
	return expo + jitter
}

// ErrOAuthRequired signals that an endpoint requires Bearer auth but the
// client only has an Admin API key. Wrapped at resource layer with a clear
// remediation message.
var ErrOAuthRequired = errors.New("this endpoint requires OAuth bearer auth (set provider attribute oauth_token or env ANTHROPIC_OAUTH_TOKEN)")

// ErrComplianceRequired signals that an endpoint requires a Compliance API
// key. Wrapped at data source layer with a clear remediation message.
var ErrComplianceRequired = errors.New("this endpoint requires a Compliance API key (set provider attribute compliance_api_key or env ANTHROPIC_COMPLIANCE_API_KEY)")
