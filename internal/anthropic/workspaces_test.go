package anthropic

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestAllowedInferenceGeos_Unmarshal(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want []string
	}{
		{"string unrestricted", `"unrestricted"`, []string{"unrestricted"}},
		{"array two geos", `["us","eu"]`, []string{"us", "eu"}},
		{"empty array", `[]`, []string{}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var v AllowedInferenceGeos
			if err := json.Unmarshal([]byte(tc.in), &v); err != nil {
				t.Fatalf("unmarshal %s: %v", tc.in, err)
			}
			// Treat nil and []string{} as equivalent for the empty case.
			got := v.Values
			if got == nil {
				got = []string{}
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("got %#v, want %#v", got, tc.want)
			}
		})
	}
}

func TestAllowedInferenceGeos_Marshal(t *testing.T) {
	cases := []struct {
		name string
		in   AllowedInferenceGeos
		want string
	}{
		{"unrestricted collapses to string", AllowedInferenceGeos{Values: []string{"unrestricted"}}, `"unrestricted"`},
		{"two geos serialize as array", AllowedInferenceGeos{Values: []string{"us", "eu"}}, `["us","eu"]`},
		{"nil values serialize as null", AllowedInferenceGeos{}, `null`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			b, err := json.Marshal(tc.in)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			if string(b) != tc.want {
				t.Errorf("got %s, want %s", b, tc.want)
			}
		})
	}
}

func TestListWorkspaces_FollowsCursorPagination(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		switch calls {
		case 1:
			// First page — no cursor passed, has_more=true.
			if got := r.URL.Query().Get("after_id"); got != "" {
				t.Errorf("first call should not pass after_id, got %q", got)
			}
			_, _ = w.Write([]byte(`{"data":[{"id":"wrkspc_1","name":"a","type":"workspace"}],"first_id":"wrkspc_1","last_id":"wrkspc_1","has_more":true}`))
		case 2:
			// Second page — must pass last_id of previous as after_id.
			if got := r.URL.Query().Get("after_id"); got != "wrkspc_1" {
				t.Errorf("expected after_id=wrkspc_1, got %q", got)
			}
			_, _ = w.Write([]byte(`{"data":[{"id":"wrkspc_2","name":"b","type":"workspace"}],"first_id":"wrkspc_2","last_id":"wrkspc_2","has_more":false}`))
		default:
			t.Fatalf("unexpected extra call #%d", calls)
		}
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "k", "v")
	list, err := c.ListWorkspaces(context.Background(), ListWorkspacesParams{})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 workspaces, got %d", len(list))
	}
	if list[0].ID != "wrkspc_1" || list[1].ID != "wrkspc_2" {
		t.Errorf("unexpected ids: %+v", list)
	}
	if calls != 2 {
		t.Errorf("expected exactly 2 calls, got %d", calls)
	}
}
