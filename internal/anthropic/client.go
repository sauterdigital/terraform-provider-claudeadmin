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
	baseURL    string
	apiKey     string
	userAgent  string
	httpClient *http.Client

	// sleeper exists so tests can swap in a fake clock without waiting real
	// seconds during retry exercises. nil means use time.Sleep.
	sleeper func(time.Duration)
}

func NewClient(baseURL, apiKey, version string) *Client {
	return &Client{
		baseURL:    strings.TrimRight(baseURL, "/"),
		apiKey:     apiKey,
		userAgent:  fmt.Sprintf("terraform-provider-anthropic/%s", version),
		httpClient: &http.Client{Timeout: defaultTimeout},
	}
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
		req.Header.Set("x-api-key", c.apiKey)
		req.Header.Set("anthropic-version", apiVersionHeader)
		req.Header.Set("content-type", "application/json")
		req.Header.Set("user-agent", c.userAgent)

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
	// Loop body always returns; defensive guard for compiler.
	return fmt.Errorf("unreachable: retry loop exited without a result")
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
	// Exponential backoff: base * 2^attempt, capped, plus 0..jitter.
	expo := baseBackoff << attempt
	if expo > maxBackoff {
		expo = maxBackoff
	}
	jitter := time.Duration(rand.Int63n(int64(backoffJitter)))
	return expo + jitter
}
