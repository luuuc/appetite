package workflow

import (
	"regexp"
	"strings"
)

// scopeCardRE matches a single scope-card line in a pitch's
// `## Scope` section. The intentional strictness is documented in the
// pitch's "Rabbit holes" — variant markdown (different dashes, bold
// styles, missing checkbox) is silently ignored.
//
//	- [ ] **Export button** — Button on list views wired to the export endpoint
//
// Both the em-dash (—, U+2014) and the literal "- [ ]" prefix are
// required; tabs and trailing whitespace are not tolerated.
var scopeCardRE = regexp.MustCompile(`^- \[ \] \*\*([^*]+)\*\* — (.+)$`)

// ScopeCard is one parsed scope-card line.
type ScopeCard struct {
	Title       string // the **bold** part
	Description string // the text after the em-dash
}

// ParseScope extracts scope cards from a pitch body. It locates the
// "## Scope" heading and parses subsequent lines until the next
// heading (any `## ` line) or EOF.
//
// invalidLines reports lines inside the Scope section that look like
// they were intended as cards but fail the strict pattern — useful
// for finalize/cut error messages. A line "looks like" a card if it
// starts with "- [" (the operator clearly meant to write a task).
// Other lines (prose, blanks) are quietly ignored.
func ParseScope(body string) (cards []ScopeCard, invalidLines []string) {
	inScope := false
	for _, raw := range strings.Split(body, "\n") {
		line := strings.TrimRight(raw, "\r")
		if strings.HasPrefix(line, "## ") {
			inScope = strings.TrimSpace(strings.TrimPrefix(line, "## ")) == "Scope"
			continue
		}
		if !inScope {
			continue
		}
		if m := scopeCardRE.FindStringSubmatch(line); m != nil {
			cards = append(cards, ScopeCard{
				Title:       strings.TrimSpace(m[1]),
				Description: strings.TrimSpace(m[2]),
			})
			continue
		}
		if strings.HasPrefix(line, "- [") {
			invalidLines = append(invalidLines, line)
		}
	}
	return cards, invalidLines
}
