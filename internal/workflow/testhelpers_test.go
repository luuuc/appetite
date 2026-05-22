package workflow

import (
	"context"
	"testing"
	"time"

	"github.com/luuuc/appetite/internal/markdown"
	"github.com/luuuc/appetite/internal/model"
	"github.com/luuuc/appetite/internal/store"
)

// fixedNow is the deterministic clock used by every workflow test —
// midnight UTC on 2026-05-22. Tests pass this via the Now field on
// request structs so written entities have stable timestamps in
// golden comparisons.
var fixedNow = time.Date(2026, 5, 22, 0, 0, 0, 0, time.UTC)

// newStore returns a Markdown-backed store rooted at a fresh temp
// directory. The directory is automatically cleaned up at test end.
func newStore(t *testing.T) store.Store {
	t.Helper()
	return markdown.New(t.TempDir())
}

// writePitchDirect bypasses ShapeNew to seed a pitch in a specific
// status — useful for testing transitions that ShapeNew can't
// produce (e.g. a pitch already in `bet`).
func writePitchDirect(t *testing.T, s store.Store, p model.Pitch) {
	t.Helper()
	if _, err := s.Write(context.Background(), p); err != nil {
		t.Fatalf("seed pitch: %v", err)
	}
}

// writeCycleDirect seeds a cycle with arbitrary fields.
func writeCycleDirect(t *testing.T, s store.Store, c model.Cycle) {
	t.Helper()
	if _, err := s.Write(context.Background(), c); err != nil {
		t.Fatalf("seed cycle: %v", err)
	}
}

// writeCardDirect seeds a card.
func writeCardDirect(t *testing.T, s store.Store, c model.Card) {
	t.Helper()
	if _, err := s.Write(context.Background(), c); err != nil {
		t.Fatalf("seed card: %v", err)
	}
}

// writeSignalDirect seeds a signal.
func writeSignalDirect(t *testing.T, s store.Store, sig model.Signal) {
	t.Helper()
	if _, err := s.Write(context.Background(), sig); err != nil {
		t.Fatalf("seed signal: %v", err)
	}
}

// shapedBody is a pitch body that finalizes successfully — has all
// five ingredient headings and one scope card.
const shapedBody = `## Problem
x
## Appetite
x
## Solution
x
## Rabbit holes
x
## No-gos
x
## Scope
- [ ] **Card one** — does the first thing
- [ ] **Card two** — does the second thing
`
