package anthropic

import (
	"encoding/json"
	"testing"
)

func TestClaudeCodeActor_Unmarshal_BothShapes(t *testing.T) {
	cases := []struct {
		name      string
		body      string
		wantType  string
		wantEmail string
		wantKey   string
	}{
		{"user_actor", `{"type":"user_actor","email_address":"alice@example.com"}`, "user_actor", "alice@example.com", ""},
		{"api_actor", `{"type":"api_actor","api_key_name":"ci-runner"}`, "api_actor", "", "ci-runner"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var a ClaudeCodeActor
			if err := json.Unmarshal([]byte(tc.body), &a); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			if a.Type != tc.wantType {
				t.Errorf("type: got %q, want %q", a.Type, tc.wantType)
			}
			if a.EmailAddress != tc.wantEmail {
				t.Errorf("email_address: got %q, want %q", a.EmailAddress, tc.wantEmail)
			}
			if a.APIKeyName != tc.wantKey {
				t.Errorf("api_key_name: got %q, want %q", a.APIKeyName, tc.wantKey)
			}
		})
	}
}

func TestClaudeCodeUsageEntry_FullDecode(t *testing.T) {
	body := `{
        "date": "2025-08-08T00:00:00Z",
        "organization_id": "org_x",
        "customer_type": "api",
        "subscription_type": null,
        "terminal_type": "iTerm.app",
        "actor": {"type": "user_actor", "email_address": "a@b.com"},
        "core_metrics": {
            "commits_by_claude_code": 3,
            "pull_requests_by_claude_code": 1,
            "num_sessions": 12,
            "lines_of_code": {"added": 1000, "removed": 50}
        },
        "model_breakdown": [{
            "model": "claude-opus-4-7",
            "estimated_cost": {"amount": 1234.5, "currency": "USD"},
            "tokens": {"input": 100, "output": 200, "cache_creation": 10, "cache_read": 5}
        }],
        "tool_actions": {"edit_tool": {"accepted": 50, "rejected": 2}}
    }`
	var e ClaudeCodeUsageEntry
	if err := json.Unmarshal([]byte(body), &e); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if e.Actor.EmailAddress != "a@b.com" {
		t.Errorf("actor email: %q", e.Actor.EmailAddress)
	}
	if e.CoreMetrics.LinesOfCode.Added != 1000 {
		t.Errorf("lines added: %d", e.CoreMetrics.LinesOfCode.Added)
	}
	if got := e.ToolActions["edit_tool"].Accepted; got != 50 {
		t.Errorf("edit_tool accepted: %d", got)
	}
	if e.SubscriptionType != nil {
		t.Errorf("expected nil subscription_type, got %v", *e.SubscriptionType)
	}
	if len(e.ModelBreakdown) != 1 || e.ModelBreakdown[0].EstimatedCost.Amount != 1234.5 {
		t.Errorf("model_breakdown decode wrong: %+v", e.ModelBreakdown)
	}
}
