package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"gost-panel/internal/auth"
	"gost-panel/internal/config"
	"gost-panel/internal/store"
)

func paginationTestServer(t *testing.T) (*Server, string) {
	t.Helper()
	baseDir := t.TempDir()
	st, err := store.Open(filepath.Join(baseDir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	manager := &auth.Manager{Secret: "pagination-tests-only", ExpireHours: 1}
	token, err := manager.CreateToken("test-user")
	if err != nil {
		t.Fatal(err)
	}
	return NewServer(&config.Config{BaseDir: baseDir}, st, nil, manager), token
}

func requestTestAPI(t *testing.T, s *Server, token, method, path string) json.RawMessage {
	t.Helper()
	req := httptest.NewRequest(method, path, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)
	var body struct {
		Code int             `json:"code"`
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if w.Code != http.StatusOK || body.Code != 0 {
		t.Fatalf("%s %s: status=%d body=%s", method, path, w.Code, w.Body)
	}
	return body.Data
}

func TestPaginatedAndLegacyLists(t *testing.T) {
	s, token := paginationTestServer(t)
	for i := 1; i <= 23; i++ {
		_, err := s.st.CreateRule(store.Rule{
			Name: fmt.Sprintf("rule-%02d", i), Type: "forward", Proto: "tcp", Enabled: 1,
			ListenPort: 4000 + i, TargetHost: "localhost", TargetPort: 80,
		})
		if err != nil {
			t.Fatal(err)
		}
		s.st.AddAudit("rule.update", fmt.Sprintf("rule-%02d", i))
	}
	for _, resource := range []string{"rules", "audit"} {
		t.Run(resource, func(t *testing.T) {
			base := "/api/" + resource
			legacyPath, wantCount := base, 23
			if resource == "audit" {
				legacyPath, wantCount = base+"?limit=5", 5
			}
			var legacy []struct{ ID int }
			if err := json.Unmarshal(requestTestAPI(t, s, token, "GET", legacyPath), &legacy); err != nil {
				t.Fatal(err)
			}
			if len(legacy) != wantCount || legacy[0].ID != 23 {
				t.Fatalf("legacy array count=%d; want=%d", len(legacy), wantCount)
			}
			for _, tc := range []struct {
				query                  string
				page, size, total, len int
				firstID                int
			}{
				{"?page=2&page_size=8", 2, 8, 23, 8, 15},
				{"?page_size=4", 1, 4, 23, 4, 23},
				{"?keyword=rule-02", 1, 20, 1, 1, 2},
				{"?page=0&page_size=0", 1, 20, 23, 20, 23},
				{"?page=invalid&page_size=99999", 1, 200, 23, 23, 23},
				{"?page=", 1, 20, 23, 20, 23},
				{"?page=99&page_size=8", 99, 8, 23, 0, 0},
				{"?keyword=missing", 1, 20, 0, 0, 0},
			} {
				t.Run(tc.query, func(t *testing.T) {
					var result struct {
						Items    []struct{ ID int } `json:"items"`
						Total    int                `json:"total"`
						Page     int                `json:"page"`
						PageSize int                `json:"page_size"`
					}
					raw := requestTestAPI(t, s, token, "GET", base+tc.query)
					if err := json.Unmarshal(raw, &result); err != nil {
						t.Fatal(err)
					}
					if result.Items == nil || result.Page != tc.page || result.PageSize != tc.size || result.Total != tc.total || len(result.Items) != tc.len {
						t.Fatalf("unexpected pagination response: %s", raw)
					}
					if tc.len > 0 && result.Items[0].ID != tc.firstID {
						t.Fatalf("first id=%d; want=%d", result.Items[0].ID, tc.firstID)
					}
				})
			}
		})
	}
}

func TestClearAuditRequiresAuthAndPreservesRules(t *testing.T) {
	s, token := paginationTestServer(t)
	if _, err := s.st.CreateRule(store.Rule{Name: "keep me"}); err != nil {
		t.Fatal(err)
	}
	s.st.AddAudit("login", "test-user")
	s.st.AddAudit("rule.update", "test rule")
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, httptest.NewRequest("DELETE", "/api/audit", nil))
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated clear status=%d", w.Code)
	}
	entries, err := s.st.ListAudit(50)
	if err != nil || len(entries) != 2 {
		t.Fatalf("unauthenticated clear changed audit data: %v %v", entries, err)
	}
	for _, wantDeleted := range []int{2, 1} {
		var result struct{ Deleted int }
		if err := json.Unmarshal(requestTestAPI(t, s, token, "DELETE", "/api/audit"), &result); err != nil {
			t.Fatal(err)
		}
		if result.Deleted != wantDeleted {
			t.Fatalf("deleted=%d; want=%d", result.Deleted, wantDeleted)
		}
		entries, err := s.st.ListAudit(50)
		if err != nil || len(entries) != 1 || entries[0].Action != "audit.clear" {
			t.Fatalf("audit clear marker missing: %v %v", entries, err)
		}
	}
	rules, err := s.st.ListRules()
	if err != nil || len(rules) != 1 || rules[0].Name != "keep me" {
		t.Fatalf("clear changed rules: %v %v", rules, err)
	}
}

func TestRuleStatsAcrossPages(t *testing.T) {
	s, token := paginationTestServer(t)
	for _, r := range []store.Rule{
		{Name: "tcp", Type: "forward", Proto: "tcp", Enabled: 1},
		{Name: "udp", Type: "forward", Proto: "udp", Enabled: 0},
		{Name: "tunnel", Type: "tunnel", Proto: "tcp", Enabled: 1},
	} {
		if _, err := s.st.CreateRule(r); err != nil {
			t.Fatal(err)
		}
	}
	var stats store.RuleStats
	if err := json.Unmarshal(requestTestAPI(t, s, token, "GET", "/api/rules/stats"), &stats); err != nil {
		t.Fatal(err)
	}
	if stats.Total != 3 || stats.Enabled != 2 || stats.TCP != 1 || stats.UDP != 1 || stats.Tunnel != 1 {
		t.Fatalf("unexpected stats: %+v", stats)
	}
}
