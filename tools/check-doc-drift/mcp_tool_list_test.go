package main

import (
	"strings"
	"testing"
)

const sampleDocWithMCPTable = `# Appetite — MCP and CLI

## MCP Tools

| Tool | CLI equivalent | Description |
|---|---|---|
| ` + "`appetite_shape`" + ` | ` + "`appetite shape`" + ` | shape pitches |
| ` + "`appetite_signal`" + ` | ` + "`appetite signal add`" + ` | record signals |
| ` + "`appetite_sweep`" + ` | ` + "`appetite … sweep`" + ` | archive sweeps |

## CLI Shape

` + "```bash" + `
appetite shape --new csv
` + "```" + `
`

const sampleBinarySource = `package mcp

func defaultMethods(s *Server) map[string]Handler {
	return map[string]Handler{
		"initialize":      handleInitialize,
		"appetite_shape":  handleShape,
		"appetite_signal": handleSignal,
		"appetite_cycle":  handleCycleNew,
	}
}
`

func TestParseDocMCPTools(t *testing.T) {
	got, err := parseDocMCPTools(strings.NewReader(sampleDocWithMCPTable))
	if err != nil {
		t.Fatalf("parseDocMCPTools: %v", err)
	}
	want := []string{"appetite_shape", "appetite_signal", "appetite_sweep"}
	for _, name := range want {
		if _, ok := got[name]; !ok {
			t.Errorf("missing tool %q in parsed set: %v", name, got)
		}
	}
	if len(got) != len(want) {
		t.Errorf("len = %d, want %d; got=%v", len(got), len(want), got)
	}
}

func TestParseDocMCPToolsStopsAtNextSection(t *testing.T) {
	// A second section's table must not leak rows into the MCP set.
	doc := `## MCP Tools

| ` + "`appetite_one`" + ` | x |

## Other

| ` + "`should_not_appear`" + ` | y |
`
	got, err := parseDocMCPTools(strings.NewReader(doc))
	if err != nil {
		t.Fatalf("parseDocMCPTools: %v", err)
	}
	if _, ok := got["appetite_one"]; !ok {
		t.Error("missing appetite_one")
	}
	if _, ok := got["should_not_appear"]; ok {
		t.Error("rows after next ## leaked through")
	}
}

func TestParseDocMCPToolsStopsAtHorizontalRule(t *testing.T) {
	doc := `## MCP Tools

| ` + "`appetite_one`" + ` | x |

---

| ` + "`should_not_appear`" + ` | y |
`
	got, _ := parseDocMCPTools(strings.NewReader(doc))
	if _, ok := got["should_not_appear"]; ok {
		t.Error("rows after --- leaked through")
	}
}

func TestParseBinaryMCPTools(t *testing.T) {
	got, err := parseBinaryMCPTools(strings.NewReader(sampleBinarySource))
	if err != nil {
		t.Fatalf("parseBinaryMCPTools: %v", err)
	}
	for _, name := range []string{"appetite_shape", "appetite_signal", "appetite_cycle"} {
		if _, ok := got[name]; !ok {
			t.Errorf("missing %q in parsed set: %v", name, got)
		}
	}
	if _, ok := got["initialize"]; ok {
		t.Error("initialize handshake should not appear in tool set")
	}
}

func TestCheckMCPToolListReportsDocOnlyAndBinaryOnly(t *testing.T) {
	drifts, err := checkMCPToolList(
		"07-mcp-and-cli.md", strings.NewReader(sampleDocWithMCPTable),
		"tools.go", strings.NewReader(sampleBinarySource),
	)
	if err != nil {
		t.Fatalf("checkMCPToolList: %v", err)
	}
	joined := strings.Join(drifts, "\n")

	// Doc has `appetite_sweep`; binary does not register it.
	if !strings.Contains(joined, "appetite_sweep") || !strings.Contains(joined, "does not register") {
		t.Errorf("expected doc-only drift for appetite_sweep; got:\n%s", joined)
	}
	// Binary registers `appetite_cycle`; doc has no row for it.
	if !strings.Contains(joined, "appetite_cycle") || !strings.Contains(joined, "no row") {
		t.Errorf("expected binary-only drift for appetite_cycle; got:\n%s", joined)
	}
}

func TestCheckMCPToolListReportsCleanWhenSetsAgree(t *testing.T) {
	doc := `## MCP Tools

| ` + "`appetite_shape`" + ` | x |
| ` + "`appetite_signal`" + ` | y |
`
	src := `func defaultMethods(s *Server) map[string]Handler {
	return map[string]Handler{
		"initialize":      handleInitialize,
		"appetite_shape":  handleShape,
		"appetite_signal": handleSignal,
	}
}`
	drifts, err := checkMCPToolList("07-mcp-and-cli.md", strings.NewReader(doc), "tools.go", strings.NewReader(src))
	if err != nil {
		t.Fatalf("checkMCPToolList: %v", err)
	}
	if len(drifts) != 0 {
		t.Errorf("want no drift, got:\n%s", strings.Join(drifts, "\n"))
	}
}

// erroringReader is an io.Reader that fails on Read so we can drive
// the scanner-error branches in the parsers (otherwise unreachable
// with byte-slice inputs).
type erroringReader struct{}

func (erroringReader) Read([]byte) (int, error) { return 0, errSimulated }

var errSimulated = errSentinel("simulated read error")

type errSentinel string

func (e errSentinel) Error() string { return string(e) }

func TestCheckMCPToolListSurfacesDocParseError(t *testing.T) {
	_, err := checkMCPToolList("doc.md", erroringReader{}, "src.go", strings.NewReader(""))
	if err == nil {
		t.Error("want error from doc parser, got nil")
	}
	if !strings.Contains(err.Error(), "parse doc tools") {
		t.Errorf("error = %v, want parse-doc-tools wrap", err)
	}
}

func TestCheckMCPToolListSurfacesBinaryParseError(t *testing.T) {
	_, err := checkMCPToolList("doc.md", strings.NewReader(""), "src.go", erroringReader{})
	if err == nil {
		t.Error("want error from binary parser, got nil")
	}
	if !strings.Contains(err.Error(), "parse binary tools") {
		t.Errorf("error = %v, want parse-binary-tools wrap", err)
	}
}

func TestSortMapKeys(t *testing.T) {
	m := map[string]int{"c": 1, "a": 2, "b": 3}
	got := sortMapKeys(m)
	want := []string{"a", "b", "c"}
	if len(got) != 3 || got[0] != want[0] || got[1] != want[1] || got[2] != want[2] {
		t.Errorf("sortMapKeys = %v, want %v", got, want)
	}
}
