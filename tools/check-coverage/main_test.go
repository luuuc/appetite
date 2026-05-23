package main

import (
	"strings"
	"testing"
)

func TestWatchedFile(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"github.com/luuuc/appetite/internal/workflow/bet.go", true},
		{"github.com/luuuc/appetite/internal/cli/cycle.go", true},
		{"github.com/luuuc/appetite/internal/mcp/server.go", true},
		{"github.com/luuuc/appetite/cmd/appetite/main.go", true},

		{"github.com/luuuc/appetite/internal/markdown/adapter.go", false},
		{"github.com/luuuc/appetite/internal/model/card.go", false},
		{"github.com/luuuc/appetite/internal/store/store.go", false},
		{"github.com/luuuc/appetite/internal/version/version.go", false},
		{"github.com/luuuc/appetite/tools/check-coverage/main.go", false},

		{"", false},
		{"github.com/luuuc/appetite/internal/workflow", false}, // missing trailing slash → not a prefix match
	}
	for _, tc := range tests {
		if got := watchedFile(tc.path); got != tc.want {
			t.Errorf("watchedFile(%q) = %v, want %v", tc.path, got, tc.want)
		}
	}
}

func TestParseFuncCov(t *testing.T) {
	// `go tool cover -func` output: <file>:<line>:\t<name>\t<pct>%, with
	// a leading header line and a trailing total: line. The parser must
	// skip those, filter to watched packages, and round-trip the pct.
	input := strings.Join([]string{
		"github.com/luuuc/appetite/internal/workflow/bet.go:31:\tBet\t96.3%",
		"github.com/luuuc/appetite/internal/cli/cycle.go:10:\tCycle\t89.5%",
		"github.com/luuuc/appetite/internal/markdown/adapter.go:37:\tWrite\t62.5%", // unwatched → dropped
		"github.com/luuuc/appetite/internal/mcp/server.go:5:\tNew\t100.0%",
		"total:\t\t\t\t\t\t\t\t\t(statements)\t\t\t90.9%",
		"",
	}, "\n")

	out, err := parseFuncCov(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parseFuncCov: %v", err)
	}
	if len(out) != 3 {
		t.Fatalf("len = %d, want 3 (markdown filtered, total skipped); got=%+v", len(out), out)
	}

	byName := make(map[string]funcCov, len(out))
	for _, f := range out {
		byName[f.name] = f
	}

	bet, ok := byName["Bet"]
	if !ok || bet.pct != 96.3 {
		t.Errorf("Bet pct = %v, want 96.3 (entry: %+v ok=%v)", bet.pct, bet, ok)
	}
	cycle, ok := byName["Cycle"]
	if !ok || cycle.pct != 89.5 {
		t.Errorf("Cycle pct = %v, want 89.5", cycle.pct)
	}
	if _, ok := byName["Write"]; ok {
		t.Error("Write from unwatched markdown package leaked into results")
	}
}

func TestParseFuncCovSkipsMalformed(t *testing.T) {
	// Lines with too few fields or an unparseable pct must be silently
	// skipped — the real `go tool cover` output has a header and totals
	// line that match this shape.
	input := strings.Join([]string{
		"header line with no useful content",
		"github.com/luuuc/appetite/internal/workflow/bet.go:31:\tBet\tnot-a-number%",
		"github.com/luuuc/appetite/internal/workflow/bet.go:50:\tValid\t100.0%",
		"",
	}, "\n")

	out, err := parseFuncCov(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parseFuncCov: %v", err)
	}
	if len(out) != 1 || out[0].name != "Valid" {
		t.Fatalf("expected one row (Valid), got %+v", out)
	}
}
