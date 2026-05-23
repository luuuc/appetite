package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os/exec"
	"regexp"
	"sort"
	"strings"
)

// canonicalDocFile is the relative path of the definition doc whose
// CLI block the checker treats as the contract source. Other files
// in `.doc/definition/` are out of scope for this card.
const canonicalDocFile = "07-mcp-and-cli.md"

// docExample is a single `appetite ...` invocation parsed out of a
// fenced bash block. words is the token sequence after `appetite`,
// quoted-string segments collapsed into one token each. flags is the
// subset of words that begin with `-` or `--`, dash prefix stripped.
type docExample struct {
	line  int
	words []string
	flags []string
}

// parseDocExamples scans markdown for fenced bash/sh blocks and
// extracts every `appetite …` invocation, recording the source line
// number so drift reports can point a human at the line to fix.
// Comments (`# …`) inside fenced blocks and blank lines are ignored.
func parseDocExamples(r io.Reader) ([]docExample, error) {
	scanner := bufio.NewScanner(r)
	// CLI examples can include long lines (JSON args, multi-flag
	// invocations); a 1 MiB token cap is generous and matches what
	// `go tool cover -func`'s own parser uses.
	scanner.Buffer(make([]byte, 1<<20), 1<<20)

	var (
		examples []docExample
		inBlock  bool
		lineNo   int
	)
	for scanner.Scan() {
		lineNo++
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") {
			if !inBlock {
				lang := strings.TrimPrefix(trimmed, "```")
				inBlock = lang == "bash" || lang == "sh"
			} else {
				inBlock = false
			}
			continue
		}
		if !inBlock {
			continue
		}
		// Strip inline shell comments so a line like
		// `appetite hill … # ship the card` doesn't pollute the flag
		// list. The doc never uses `#` inside a quoted string in CLI
		// examples; if it does later, this is the natural place to
		// upgrade to a quote-aware splitter.
		if i := strings.Index(line, " #"); i >= 0 {
			line = line[:i]
		}
		stripped := strings.TrimSpace(line)
		if !strings.HasPrefix(stripped, "appetite ") && stripped != "appetite" {
			continue
		}
		tokens, err := tokenizeShell(stripped)
		if err != nil {
			return nil, fmt.Errorf("%s:%d: %w", canonicalDocFile, lineNo, err)
		}
		// tokens[0] is "appetite". Words after that are the example.
		words := tokens[1:]
		var flags []string
		for _, w := range words {
			if strings.HasPrefix(w, "--") {
				flags = append(flags, normalizeFlag(strings.TrimPrefix(w, "--")))
			} else if strings.HasPrefix(w, "-") && len(w) > 1 {
				flags = append(flags, normalizeFlag(strings.TrimPrefix(w, "-")))
			}
		}
		examples = append(examples, docExample{line: lineNo, words: words, flags: flags})
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return examples, nil
}

// normalizeFlag strips an `=<value>` tail and lowercases nothing —
// flag names are already lower-cased in this codebase. Returning the
// name in isolation lets the diff treat `--source beacon` and
// `--source=beacon` as the same flag.
func normalizeFlag(raw string) string {
	if i := strings.Index(raw, "="); i >= 0 {
		return raw[:i]
	}
	return raw
}

// tokenizeShell splits a single shell-command line into tokens,
// collapsing single- and double-quoted segments to one token each
// (with quotes removed). It does not honor backslash escapes,
// command substitutions, or environment variables — the doc's CLI
// block has none of those, and adding them would invite surprises.
func tokenizeShell(line string) ([]string, error) {
	var (
		tokens   []string
		current  strings.Builder
		inSingle bool
		inDouble bool
	)
	flush := func() {
		if current.Len() == 0 {
			return
		}
		tokens = append(tokens, current.String())
		current.Reset()
	}
	for i := 0; i < len(line); i++ {
		ch := line[i]
		switch {
		case ch == '\'' && !inDouble:
			inSingle = !inSingle
		case ch == '"' && !inSingle:
			inDouble = !inDouble
		case ch == ' ' && !inSingle && !inDouble:
			flush()
		default:
			current.WriteByte(ch)
		}
	}
	if inSingle || inDouble {
		return nil, fmt.Errorf("unterminated quote in %q", line)
	}
	flush()
	return tokens, nil
}

// flagLine matches a leading `-flagname` in `go tool flag`'s default
// usage output: two-space indent, dash, letters/numbers/dashes/under,
// then a value-type or end of "word". It deliberately captures only
// the first `-` so `--double-dash` arguments to subcommands (rare in
// this binary) still classify cleanly — Go's flag package prints
// every flag with a single leading dash regardless of how the user
// types it.
var flagLine = regexp.MustCompile(`^\s+-([A-Za-z0-9_-]+)\b`)

// requiredHint matches the "(required)" annotation appetite's
// commands embed in their flag descriptions to flag must-pass
// parameters (e.g. `appetite bet` requires `-appetite` and `-cycle`).
var requiredHint = regexp.MustCompile(`\(required\)`)

// binaryHelp captures the help output for a single command path and
// the set of flags / required flags it advertises.
type binaryHelp struct {
	flags        map[string]bool
	requiredOnly map[string]bool
	recognized   bool
}

// introspect runs `binary <words...> -h` and parses the resulting
// usage output into a binaryHelp. Both stdout and stderr are merged
// because Go's flag package writes its default usage to stderr.
// recognized is true iff the output looks like a real flag.PrintDefaults
// dump rather than an "unknown subcommand" error.
func introspect(binary string, words []string) (binaryHelp, error) {
	args := append([]string{}, words...)
	args = append(args, "-h")
	cmd := exec.Command(binary, args...)
	var combined bytes.Buffer
	cmd.Stdout = &combined
	cmd.Stderr = &combined
	// We ignore the run error: Go's flag package signals
	// "help requested" with a non-zero exit, but the help text is
	// still in `combined`. A genuine failure (binary missing,
	// permission denied) surfaces via the bytes — `combined` will
	// not contain "Usage of".
	_ = cmd.Run()
	text := combined.String()

	help := binaryHelp{
		flags:        map[string]bool{},
		requiredOnly: map[string]bool{},
	}
	// "unknown command" / "unknown subcommand" are the appetite
	// binary's own markers for an unrecognized path. Commands that
	// declare no flags (e.g. `appetite version`) print no "Usage of"
	// banner, so absence of the banner alone is not unknown-ness —
	// only the explicit unknown markers are.
	switch {
	case strings.Contains(text, "unknown command"),
		strings.Contains(text, "unknown subcommand"):
		return help, nil
	}
	help.recognized = true

	sc := bufio.NewScanner(strings.NewReader(text))
	for sc.Scan() {
		line := sc.Text()
		m := flagLine.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		name := m[1]
		help.flags[name] = true
		// flag.PrintDefaults emits the flag header on one line and
		// the indented description on the next; the `(required)`
		// hint is on that description line, so we capture it on the
		// next iteration. Cheaper than a multiline regex: peek by
		// scanning ahead one line. We let the next iteration pick
		// the next flag header on its own.
		if sc.Scan() {
			desc := sc.Text()
			if requiredHint.MatchString(desc) {
				help.requiredOnly[name] = true
			}
			// Reinject: most flags have a description on a single
			// trailing line — the next iteration will move on to
			// the next flag header (or `flag: help requested`
			// trailer). Nothing more to do here.
		}
	}
	return help, sc.Err()
}

// resolveCommandPath finds the longest prefix of words that the
// binary recognizes as a subcommand path. It walks from longest to
// shortest, caching subprocess results in seen so repeated examples
// (e.g. several `appetite shape …` lines) don't pay the spawn cost
// per example.
func resolveCommandPath(binary string, words []string, seen map[string]binaryHelp) ([]string, binaryHelp) {
	for n := len(words); n > 0; n-- {
		path := words[:n]
		key := strings.Join(path, " ")
		help, ok := seen[key]
		if !ok {
			h, err := introspect(binary, path)
			if err != nil {
				// A scanner error is unusual; record the path as
				// unrecognized so the doc line gets surfaced rather
				// than swallowed.
				h = binaryHelp{flags: map[string]bool{}, requiredOnly: map[string]bool{}}
			}
			seen[key] = h
			help = h
		}
		if help.recognized {
			return path, help
		}
	}
	return nil, binaryHelp{}
}

// checkCLIFlagSurface compares the CLI examples in the doc against
// the binary's advertised flag surface and returns one drift line
// per mismatch. The function performs no fewer subprocess calls than
// the number of distinct command paths in the doc — quoting on the
// upper bound: ~15 today.
func checkCLIFlagSurface(docPath, docFile, binary string, r io.Reader) ([]string, error) {
	examples, err := parseDocExamples(r)
	if err != nil {
		return nil, err
	}
	seen := map[string]binaryHelp{}
	usedFlags := map[string]map[string]bool{} // command path → flag → seen-in-doc?
	var drifts []string

	for _, ex := range examples {
		nonFlagWords := commandWordsOnly(ex.words)
		path, help := resolveCommandPath(binary, nonFlagWords, seen)
		if path == nil {
			drifts = append(drifts, fmt.Sprintf(
				"%s:%d: `appetite %s` — binary does not recognize this command",
				docFile, ex.line, strings.Join(nonFlagWords, " "),
			))
			continue
		}
		key := strings.Join(path, " ")
		if usedFlags[key] == nil {
			usedFlags[key] = map[string]bool{}
		}
		for _, f := range ex.flags {
			usedFlags[key][f] = true
			if !help.flags[f] {
				drifts = append(drifts, fmt.Sprintf(
					"%s:%d: `appetite %s --%s` — binary does not advertise `--%s`",
					docFile, ex.line, key, f, f,
				))
			}
		}
	}

	// Second pass: required-looking flags the binary advertises but
	// no doc example exercises. Sorted so the report is stable —
	// drift output that re-orders per run undermines diffs in code
	// review.
	for _, key := range sortedKeys(seen) {
		help := seen[key]
		if !help.recognized {
			continue
		}
		for _, name := range sortedKeys(help.requiredOnly) {
			if !usedFlags[key][name] {
				drifts = append(drifts, fmt.Sprintf(
					"%s: `appetite %s` — required flag `--%s` is not exercised by any doc example",
					docFile, key, name,
				))
			}
		}
	}

	_ = docPath // reserved for future cards that resolve absolute paths
	return drifts, nil
}

// commandWordsOnly returns the leading prefix of words that do not
// begin with `-` and are not flag values that follow a flag. The
// returned slice is what resolveCommandPath walks over.
func commandWordsOnly(words []string) []string {
	var out []string
	for _, w := range words {
		if strings.HasPrefix(w, "-") {
			break
		}
		out = append(out, w)
	}
	return out
}

// sortedKeys returns map keys in deterministic order. Generic enough
// that both the binaryHelp seen-map and the bool-set use it.
func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
