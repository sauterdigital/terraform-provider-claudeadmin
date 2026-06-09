package anthropic

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
)

// ---------- Usage Report (messages) ----------

type UsageReportParams struct {
	StartingAt        string
	EndingAt          string
	BucketWidth       string
	Limit             int
	Page              string
	GroupBy           []string
	WorkspaceIDs      []string
	APIKeyIDs         []string
	AccountIDs        []string
	ServiceAccountIDs []string
	Models            []string
	ServiceTiers      []string
	ContextWindow     []string
	InferenceGeos     []string
	Speeds            []string
}

type UsageReport struct {
	Data     []UsageBucket `json:"data"`
	HasMore  bool          `json:"has_more"`
	NextPage string        `json:"next_page"`
}

type UsageBucket struct {
	StartingAt string        `json:"starting_at"`
	EndingAt   string        `json:"ending_at"`
	Results    []UsageResult `json:"results"`
}

type UsageResult struct {
	AccountID            *string       `json:"account_id"`
	APIKeyID             *string       `json:"api_key_id"`
	ServiceAccountID     *string       `json:"service_account_id"`
	WorkspaceID          *string       `json:"workspace_id"`
	Model                *string       `json:"model"`
	ContextWindow        *string       `json:"context_window"`
	InferenceGeo         *string       `json:"inference_geo"`
	ServiceTier          *string       `json:"service_tier"`
	UncachedInputTokens  int64         `json:"uncached_input_tokens"`
	CacheReadInputTokens int64         `json:"cache_read_input_tokens"`
	OutputTokens         int64         `json:"output_tokens"`
	CacheCreation        CacheCreation `json:"cache_creation"`
	ServerToolUse        ServerToolUse `json:"server_tool_use"`
}

type CacheCreation struct {
	Ephemeral1HInputTokens int64 `json:"ephemeral_1h_input_tokens"`
	Ephemeral5MInputTokens int64 `json:"ephemeral_5m_input_tokens"`
}

type ServerToolUse struct {
	WebSearchRequests int64 `json:"web_search_requests"`
}

func (c *Client) GetUsageReport(ctx context.Context, p UsageReportParams) (*UsageReport, error) {
	q := buildReportQuery(p.StartingAt, p.EndingAt, p.BucketWidth, p.Limit, p.Page, p.GroupBy)
	addList(q, "workspace_ids[]", p.WorkspaceIDs)
	addList(q, "api_key_ids[]", p.APIKeyIDs)
	addList(q, "account_ids[]", p.AccountIDs)
	addList(q, "service_account_ids[]", p.ServiceAccountIDs)
	addList(q, "models[]", p.Models)
	addList(q, "service_tiers[]", p.ServiceTiers)
	addList(q, "context_window[]", p.ContextWindow)
	addList(q, "inference_geos[]", p.InferenceGeos)
	addList(q, "speeds[]", p.Speeds)

	var out UsageReport
	if err := c.do(ctx, http.MethodGet, "/v1/organizations/usage_report/messages?"+q.Encode(), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ---------- Claude Code Usage Report ----------

type ClaudeCodeUsageReportParams struct {
	StartingAt string // required, format YYYY-MM-DD
	Limit      int
	Page       string
}

type ClaudeCodeUsageReport struct {
	Data     []ClaudeCodeUsageEntry `json:"data"`
	HasMore  bool                   `json:"has_more"`
	NextPage *string                `json:"next_page"`
}

type ClaudeCodeUsageEntry struct {
	Date             string                       `json:"date"`
	OrganizationID   string                       `json:"organization_id"`
	CustomerType     string                       `json:"customer_type"`
	SubscriptionType *string                      `json:"subscription_type"`
	TerminalType     string                       `json:"terminal_type"`
	Actor            ClaudeCodeActor              `json:"actor"`
	CoreMetrics      ClaudeCodeCoreMetrics        `json:"core_metrics"`
	ModelBreakdown   []ClaudeCodeModelBreakdown   `json:"model_breakdown"`
	ToolActions      map[string]ClaudeCodeToolAct `json:"tool_actions"`
}

// ClaudeCodeActor flattens the API's UserActor | APIActor polymorphism.
// Exactly one of EmailAddress (when Type=="user_actor") or APIKeyName
// (when Type=="api_actor") is populated.
type ClaudeCodeActor struct {
	Type         string `json:"type"`
	EmailAddress string `json:"email_address,omitempty"`
	APIKeyName   string `json:"api_key_name,omitempty"`
}

type ClaudeCodeCoreMetrics struct {
	CommitsByClaudeCode      int64                `json:"commits_by_claude_code"`
	PullRequestsByClaudeCode int64                `json:"pull_requests_by_claude_code"`
	NumSessions              int64                `json:"num_sessions"`
	LinesOfCode              ClaudeCodeLineCounts `json:"lines_of_code"`
}

type ClaudeCodeLineCounts struct {
	Added   int64 `json:"added"`
	Removed int64 `json:"removed"`
}

type ClaudeCodeModelBreakdown struct {
	Model         string              `json:"model"`
	EstimatedCost ClaudeCodeCost      `json:"estimated_cost"`
	Tokens        ClaudeCodeTokenInfo `json:"tokens"`
}

type ClaudeCodeCost struct {
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`
}

type ClaudeCodeTokenInfo struct {
	Input         int64 `json:"input"`
	Output        int64 `json:"output"`
	CacheCreation int64 `json:"cache_creation"`
	CacheRead     int64 `json:"cache_read"`
}

type ClaudeCodeToolAct struct {
	Accepted int64 `json:"accepted"`
	Rejected int64 `json:"rejected"`
}

func (c *Client) GetClaudeCodeUsageReport(ctx context.Context, p ClaudeCodeUsageReportParams) (*ClaudeCodeUsageReport, error) {
	q := url.Values{}
	if p.StartingAt != "" {
		q.Set("starting_at", p.StartingAt)
	}
	if p.Limit > 0 {
		q.Set("limit", strconv.Itoa(p.Limit))
	}
	if p.Page != "" {
		q.Set("page", p.Page)
	}
	var out ClaudeCodeUsageReport
	if err := c.do(ctx, http.MethodGet, "/v1/organizations/usage_report/claude_code?"+q.Encode(), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ---------- Cost Report ----------

type CostReportParams struct {
	StartingAt   string
	EndingAt     string
	BucketWidth  string
	Limit        int
	Page         string
	GroupBy      []string
	WorkspaceIDs []string
}

type CostReport struct {
	Data     []CostBucket `json:"data"`
	HasMore  bool         `json:"has_more"`
	NextPage string       `json:"next_page"`
}

type CostBucket struct {
	StartingAt string       `json:"starting_at"`
	EndingAt   string       `json:"ending_at"`
	Results    []CostResult `json:"results"`
}

type CostResult struct {
	Amount        string  `json:"amount"`
	Currency      string  `json:"currency"`
	CostType      *string `json:"cost_type"`
	Description   *string `json:"description"`
	WorkspaceID   *string `json:"workspace_id"`
	Model         *string `json:"model"`
	TokenType     *string `json:"token_type"`
	ContextWindow *string `json:"context_window"`
	InferenceGeo  *string `json:"inference_geo"`
	ServiceTier   *string `json:"service_tier"`
}

func (c *Client) GetCostReport(ctx context.Context, p CostReportParams) (*CostReport, error) {
	q := buildReportQuery(p.StartingAt, p.EndingAt, p.BucketWidth, p.Limit, p.Page, p.GroupBy)
	addList(q, "workspace_ids[]", p.WorkspaceIDs)

	var out CostReport
	if err := c.do(ctx, http.MethodGet, "/v1/organizations/cost_report?"+q.Encode(), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ---------- shared query builders ----------

func buildReportQuery(startingAt, endingAt, bucketWidth string, limit int, page string, groupBy []string) url.Values {
	q := url.Values{}
	if startingAt != "" {
		q.Set("starting_at", startingAt)
	}
	if endingAt != "" {
		q.Set("ending_at", endingAt)
	}
	if bucketWidth != "" {
		q.Set("bucket_width", bucketWidth)
	}
	if limit > 0 {
		q.Set("limit", strconv.Itoa(limit))
	}
	if page != "" {
		q.Set("page", page)
	}
	addList(q, "group_by[]", groupBy)
	return q
}

func addList(q url.Values, key string, values []string) {
	for _, v := range values {
		q.Add(key, v)
	}
}
