package store

import (
	"path/filepath"
	"reflect"
	"testing"
)

func testStore(t *testing.T) *Store {
	t.Helper()
	s, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestListRulesPagedFiltersEveryPage(t *testing.T) {
	s := testStore(t)
	for i := 1; i <= 7; i++ {
		r := Rule{Name: "ordinary", Type: "forward", Proto: "tcp", ListenPort: 3000 + i, TargetHost: "localhost", TargetPort: 80}
		switch i {
		case 2:
			r.Name = "needle name"
		case 4:
			r.Remark = "needle remark"
		case 6:
			r.TargetHost = "needle.example"
		}
		if _, err := s.CreateRule(r); err != nil {
			t.Fatal(err)
		}
	}
	for _, tc := range []struct {
		name, keyword string
		page          int
		wantIDs       []int
		wantTotal     int64
	}{
		{"first", "needle", 1, []int{6, 4}, 3},
		{"second", "needle", 2, []int{2}, 3},
		{"past end", "needle", 3, []int{}, 3},
		{"port", "3005", 1, []int{5}, 1},
		{"quoted", "' OR 1=1 --", 1, []int{}, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			items, total, err := s.ListRulesPaged(RuleQuery{Page: tc.page, PageSize: 2, Keyword: tc.keyword})
			if err != nil {
				t.Fatal(err)
			}
			ids := make([]int, 0, len(items))
			for _, item := range items {
				ids = append(ids, item.ID)
			}
			if total != tc.wantTotal || !reflect.DeepEqual(ids, tc.wantIDs) {
				t.Fatalf("got ids=%v total=%d; want ids=%v total=%d", ids, total, tc.wantIDs, tc.wantTotal)
			}
		})
	}
}

func TestClearAuditPreservesRules(t *testing.T) {
	s := testStore(t)
	if _, err := s.CreateRule(Rule{Name: "keep me"}); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 5; i++ {
		s.AddAudit("rule.update", "test")
	}
	items, total, err := s.ListAuditPaged(2, 2, "")
	if err != nil || total != 5 || len(items) != 2 || items[0].ID != 3 || items[1].ID != 2 {
		t.Fatalf("audit page: items=%v total=%d err=%v", items, total, err)
	}
	n, err := s.ClearAudit()
	if err != nil || n != 5 {
		t.Fatalf("clear: deleted=%d err=%v", n, err)
	}
	items, total, err = s.ListAuditPaged(1, 20, "")
	if err != nil || total != 0 || len(items) != 0 {
		t.Fatalf("audit after clear: total=%d err=%v", total, err)
	}
	if n, err = s.ClearAudit(); err != nil || n != 0 {
		t.Fatalf("clear empty: deleted=%d err=%v", n, err)
	}
	rules, err := s.ListRules()
	if err != nil || len(rules) != 1 || rules[0].Name != "keep me" {
		t.Fatalf("rules changed by audit clear: %v %v", rules, err)
	}
}
