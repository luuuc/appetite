package mcp

import (
	"context"
	"testing"

	"github.com/luuuc/appetite/internal/model"
	"github.com/luuuc/appetite/internal/store"
)

// patchPitchScope rewrites the pitch's body so the Scope section
// carries exactly one valid card. It exists so the e2e test can
// progress past finalize without depending on the template format
// staying byte-identical.
func patchPitchScope(t *testing.T, s store.Store, slug string) {
	t.Helper()
	p, err := store.ReadAs[model.Pitch](context.Background(), s, model.Pitch{Slug: slug}.Path())
	if err != nil {
		t.Fatalf("read pitch: %v", err)
	}
	p.Body = `## Problem
exported CSVs take too long for the user
## Appetite
small — 3 days
## Solution
stream the rows as they arrive
## Rabbit holes
do not rewrite the format
## No-gos
no new file format
## Scope
- [ ] **The only card** — write streaming export
`
	if _, err := s.Write(context.Background(), p); err != nil {
		t.Fatalf("patch pitch: %v", err)
	}
}
