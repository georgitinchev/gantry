package lint

import (
	"path/filepath"
	"testing"
)

func run(t *testing.T, root string) []Finding {
	t.Helper()
	apis, err := LoadAPIs(filepath.Join(root, "platform", "apis"))
	if err != nil {
		t.Fatal(err)
	}
	reqs, fs := LoadRequests(root)
	return append(fs, Check(apis, reqs, Config{TeamBudget: 6})...)
}

func TestRepositoryRequestsAreClean(t *testing.T) {
	for _, f := range run(t, "../../../..") {
		t.Errorf("unexpected finding: %+v", f)
	}
}

func TestEveryRuleFires(t *testing.T) {
	got := map[string]int{}
	for _, f := range run(t, "testdata/bad") {
		got[f.Rule]++
	}
	for _, rule := range []string{
		"layout", "single-document", "known-api", "file-name", "scope",
		"schema", "unknown-field", "ownership", "unique", "budget",
	} {
		if got[rule] == 0 {
			t.Errorf("rule %q did not fire", rule)
		}
	}
	if got["ownership"] < 3 {
		t.Errorf("want team-mismatch, foreign-namespace and missing-namespace ownership findings, got %d", got["ownership"])
	}
}

func TestFindingsPointAtTheOffendingLine(t *testing.T) {
	for _, f := range run(t, "testdata/bad") {
		if f.File == "requests/payments/appenvironment.yaml" && f.Rule == "unknown-field" && f.Line != 8 {
			t.Errorf("unknown-field should point at line 8, got %d", f.Line)
		}
	}
}
