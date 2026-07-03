package anthropic

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestClient_HeadersAndBaseURLTrim(t *testing.T) {
	var captured *http.Request
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured = r
		_ = json.NewEncoder(w).Encode(Workspace{ID: "wrkspc_test", Name: "ok", Type: "workspace"})
	}))
	defer srv.Close()

	// Trailing slash must be trimmed so the request path is clean.
	c := NewClient(srv.URL+"/", "test-key", "v0.1.0")

	ws, err := c.GetWorkspace(context.Background(), "wrkspc_test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ws.ID != "wrkspc_test" {
		t.Errorf("id: got %q, want wrkspc_test", ws.ID)
	}
	if got := captured.Header.Get("x-api-key"); got != "test-key" {
		t.Errorf("x-api-key: got %q", got)
	}
	if got := captured.Header.Get("anthropic-version"); got != "2023-06-01" {
		t.Errorf("anthropic-version: got %q", got)
	}
	if got := captured.Header.Get("user-agent"); !strings.HasPrefix(got, "terraform-provider-claude-admin/") {
		t.Errorf("user-agent: got %q", got)
	}
	if got := captured.URL.Path; got != "/v1/organizations/workspaces/wrkspc_test" {
		t.Errorf("path: got %q", got)
	}
}

func TestClient_APIError404IsNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":{"type":"not_found_error","message":"workspace not found"}}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "k", "v")
	_, err := c.GetWorkspace(context.Background(), "wrkspc_missing")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !IsNotFound(err) {
		t.Errorf("expected IsNotFound==true, got false (err=%v)", err)
	}
	if !strings.Contains(err.Error(), "workspace not found") {
		t.Errorf("expected API message to surface, got: %v", err)
	}
}

func TestClient_APIError500NonJSONFallback(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("internal whoops"))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "k", "v")
	_, err := c.GetWorkspace(context.Background(), "wrkspc_test")
	if err == nil {
		t.Fatal("expected error")
	}
	if IsNotFound(err) {
		t.Errorf("500 should not be a not-found")
	}
	if !strings.Contains(err.Error(), "internal whoops") {
		t.Errorf("expected raw body in error, got: %v", err)
	}
}

func TestClient_RetriesOn429UntilSuccess(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++
		if calls < 3 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"error":{"type":"rate_limit_error","message":"slow down"}}`))
			return
		}
		_ = json.NewEncoder(w).Encode(Workspace{ID: "wrkspc_after_retry", Name: "ok", Type: "workspace"})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "k", "v")
	// Replace sleep with a no-op so the test doesn't actually wait.
	slept := 0
	c.sleeper = func(_ time.Duration) { slept++ }

	ws, err := c.GetWorkspace(context.Background(), "wrkspc_x")
	if err != nil {
		t.Fatalf("expected eventual success after retries, got: %v", err)
	}
	if ws.ID != "wrkspc_after_retry" {
		t.Errorf("got id %q", ws.ID)
	}
	if calls != 3 {
		t.Errorf("expected 3 attempts (2 retries + success), got %d", calls)
	}
	if slept != 2 {
		t.Errorf("expected 2 sleep calls, got %d", slept)
	}
}

func TestClient_GivesUpAfterMaxRetries(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++
		w.Header().Set("Retry-After", "0")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "k", "v")
	c.sleeper = func(_ time.Duration) {}

	_, err := c.GetWorkspace(context.Background(), "wrkspc_x")
	if err == nil {
		t.Fatal("expected error after max retries")
	}
	if !strings.Contains(err.Error(), "max retries exceeded") {
		t.Errorf("expected 'max retries exceeded' message, got: %v", err)
	}
	// maxRetries=5, plus the initial attempt = 6 total calls expected.
	if calls != 6 {
		t.Errorf("expected 6 total attempts, got %d", calls)
	}
}

func TestClient_EmptyBodyResponseOK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "k", "v")
	if err := c.DeleteInvite(context.Background(), "invite_anything"); err != nil {
		t.Errorf("expected nil error for 204, got %v", err)
	}
}
