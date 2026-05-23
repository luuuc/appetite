package main

import (
	"bufio"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strings"
)

// defaultMCPSourceFile is the Go file that holds the canonical MCP
// dispatch table. The checker scans this file rather than spawning
// `appetite mcp` because the server intentionally does not implement
// MCP's `tools/list` capability — its tools are bare JSON-RPC
// methods (see the `pitch decision: minimal handshake` note in the
// initialize handler).
const defaultMCPSourceFile = "internal/mcp/tools.go"

// mcpTablePattern matches the heading that anchors the MCP Tools
// table in the canonical doc. Anchoring on the heading keeps the
// parser deterministic when the doc grows other tables later.
const mcpTableHeading = "## MCP Tools"

// mcpRowPattern picks the first backticked cell out of a markdown
// table row. Real rows look like `| `appetite_shape` | `appetite
// shape` | description |` — the leading column always has the MCP
// tool name in backticks, which is what the diff needs.
var mcpRowPattern = regexp.MustCompile(`^\|\s*` + "`" + `(\w+)` + "`")

// mcpMapEntryPattern matches a single entry in the defaultMethods
// map literal: `"<name>": handle<Something>,`. The whitespace and
// trailing comma are forgiving so the parser survives small
// formatting tweaks (Go's gofmt normalizes columns, but extra blank
// lines around the entries don't break the match).
var mcpMapEntryPattern = regexp.MustCompile(`"(appetite_\w+)"\s*:\s*handle\w+`)

// parseDocMCPTools extracts the set of MCP tool names declared in
// the doc's MCP Tools table. The parser locates the table by its
// heading, then walks rows until the next `## ` heading or `---`
// horizontal rule.
func parseDocMCPTools(r io.Reader) (map[string]int, error) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 1<<20), 1<<20)
	var (
		inSection bool
		lineNo    int
	)
	tools := map[string]int{}
	for scanner.Scan() {
		lineNo++
		line := scanner.Text()
		switch {
		case strings.HasPrefix(line, mcpTableHeading):
			inSection = true
			continue
		case inSection && strings.HasPrefix(line, "## "):
			// next section: stop scanning rows
			inSection = false
		case inSection && strings.TrimSpace(line) == "---":
			inSection = false
		}
		if !inSection {
			continue
		}
		m := mcpRowPattern.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		name := m[1]
		// The table header row (`| Tool | CLI equivalent | ... |`)
		// has no backticks around its cell content, so the regex
		// only matches real data rows. Recording first-seen line.
		if _, ok := tools[name]; !ok {
			tools[name] = lineNo
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return tools, nil
}

// parseBinaryMCPTools extracts the set of MCP tool names from the
// dispatch-table source file. The map literal is small (~12
// entries) and gofmt-stable, so a regex match per line is enough.
// The `initialize` method is excluded because it is the MCP
// handshake, not a tool — the doc table treats it the same way.
func parseBinaryMCPTools(r io.Reader) (map[string]struct{}, error) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 1<<20), 1<<20)
	tools := map[string]struct{}{}
	for scanner.Scan() {
		line := scanner.Text()
		m := mcpMapEntryPattern.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		tools[m[1]] = struct{}{}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return tools, nil
}

// checkMCPToolList diffs the doc's declared tools against the
// binary's dispatch table. Returns a stable list of drift lines:
// each surplus and each missing name appears on its own line, with
// the doc file:line pointer when available.
func checkMCPToolList(docFile string, doc io.Reader, srcFile string, src io.Reader) ([]string, error) {
	docTools, err := parseDocMCPTools(doc)
	if err != nil {
		return nil, fmt.Errorf("parse doc tools: %w", err)
	}
	binTools, err := parseBinaryMCPTools(src)
	if err != nil {
		return nil, fmt.Errorf("parse binary tools: %w", err)
	}

	var drifts []string
	// Missing in binary (doc-only).
	for _, name := range sortMapKeys(docTools) {
		if _, ok := binTools[name]; !ok {
			drifts = append(drifts, fmt.Sprintf(
				"%s:%d: doc lists MCP tool `%s` but %s does not register it",
				docFile, docTools[name], name, srcFile,
			))
		}
	}
	// Extra in binary (binary-only).
	bin := make([]string, 0, len(binTools))
	for name := range binTools {
		bin = append(bin, name)
	}
	sort.Strings(bin)
	for _, name := range bin {
		if _, ok := docTools[name]; !ok {
			drifts = append(drifts, fmt.Sprintf(
				"%s: %s registers MCP tool `%s` but doc table has no row for it",
				docFile, srcFile, name,
			))
		}
	}
	return drifts, nil
}

// sortMapKeys returns the keys of a string-keyed map in sorted
// order. Local to this file so the diff output stays deterministic
// without sharing helpers across packages.
func sortMapKeys(m map[string]int) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
