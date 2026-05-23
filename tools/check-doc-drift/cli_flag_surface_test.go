package main

import (
	"reflect"
	"strings"
	"testing"
)

func TestTokenizeShell(t *testing.T) {
	tests := []struct {
		in   string
		want []string
	}{
		{`appetite bet`, []string{"appetite", "bet"}},
		{`appetite shape --new csv-export`, []string{"appetite", "shape", "--new", "csv-export"}},
		{`appetite signal add "two words" --source op`,
			[]string{"appetite", "signal", "add", "two words", "--source", "op"}},
		{`appetite signal add 'single quoted' --tags a,b`,
			[]string{"appetite", "signal", "add", "single quoted", "--tags", "a,b"}},
		{`appetite hill --done`, []string{"appetite", "hill", "--done"}},
	}
	for _, tc := range tests {
		got, err := tokenizeShell(tc.in)
		if err != nil {
			t.Errorf("tokenizeShell(%q): unexpected error %v", tc.in, err)
			continue
		}
		if !reflect.DeepEqual(got, tc.want) {
			t.Errorf("tokenizeShell(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

func TestTokenizeShellRejectsUnterminatedQuote(t *testing.T) {
	if _, err := tokenizeShell(`appetite signal add "unterminated`); err == nil {
		t.Error("want error for unterminated double quote")
	}
	if _, err := tokenizeShell(`appetite signal add 'unterminated`); err == nil {
		t.Error("want error for unterminated single quote")
	}
}

func TestNormalizeFlag(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"source", "source"},
		{"source=beacon", "source"},
		{"appetite", "appetite"},
		{"appetite=small", "appetite"},
	}
	for _, tc := range tests {
		if got := normalizeFlag(tc.in); got != tc.want {
			t.Errorf("normalizeFlag(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestCommandWordsOnly(t *testing.T) {
	tests := []struct {
		in   []string
		want []string
	}{
		{[]string{"shape", "--new", "csv-export"}, []string{"shape"}},
		{[]string{"signal", "add", "--source", "beacon"}, []string{"signal", "add"}},
		{[]string{"--dir", ".appetite"}, nil},
		{[]string{"version"}, []string{"version"}},
		{nil, nil},
	}
	for _, tc := range tests {
		got := commandWordsOnly(tc.in)
		if !reflect.DeepEqual(got, tc.want) {
			t.Errorf("commandWordsOnly(%v) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

func TestParseDocExamples(t *testing.T) {
	doc := strings.Join([]string{
		"# Heading",
		"",
		"Prose mentioning `appetite shape` outside a fenced block is ignored.",
		"",
		"```bash",
		"# Signals",
		"appetite signal add \"body\" --source beacon",
		"appetite shape --new csv --title \"CSV Export\" # inline-comment stripped",
		"",
		"appetite hill --done",
		"```",
		"",
		"```jsonc",
		"// not a bash block; appetite mcp --dir should be ignored here",
		"```",
		"",
		"```",
		"# unfenced language; ignored too",
		"appetite bet --appetite small",
		"```",
		"",
	}, "\n")

	got, err := parseDocExamples(strings.NewReader(doc))
	if err != nil {
		t.Fatalf("parseDocExamples: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("got %d examples, want 3; got=%+v", len(got), got)
	}

	// `signal add "body" --source beacon`
	if got[0].words[0] != "signal" || got[0].words[1] != "add" {
		t.Errorf("example 0 words = %v", got[0].words)
	}
	if !contains(got[0].flags, "source") {
		t.Errorf("example 0 flags = %v, want `source`", got[0].flags)
	}

	// `shape --new csv --title "CSV Export"` with `# inline-comment` stripped
	if !contains(got[1].flags, "new") || !contains(got[1].flags, "title") {
		t.Errorf("example 1 flags = %v, want new+title", got[1].flags)
	}

	// `hill --done` (boolean flag)
	if !contains(got[2].flags, "done") {
		t.Errorf("example 2 flags = %v, want `done`", got[2].flags)
	}
}

func TestParseDocExamplesSurfacesTokenizerError(t *testing.T) {
	doc := "```bash\nappetite signal add \"unterminated\n```\n"
	if _, err := parseDocExamples(strings.NewReader(doc)); err == nil {
		t.Error("want error for unterminated quote")
	}
}

func TestSortedKeys(t *testing.T) {
	m := map[string]int{"b": 1, "a": 2, "c": 3}
	got := sortedKeys(m)
	want := []string{"a", "b", "c"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("sortedKeys = %v, want %v", got, want)
	}

	if got := sortedKeys(map[string]struct{}{}); len(got) != 0 {
		t.Errorf("sortedKeys of empty = %v, want empty", got)
	}
}

func contains(xs []string, s string) bool {
	for _, x := range xs {
		if x == s {
			return true
		}
	}
	return false
}
