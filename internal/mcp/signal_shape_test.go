package mcp

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/luuuc/appetite/internal/markdown"
	"github.com/luuuc/appetite/internal/model"
	"github.com/luuuc/appetite/internal/store"
)

func TestHandleSignalWritesEntity(t *testing.T) {
	s := markdown.New(t.TempDir())
	params := mustMarshal(t, signalParams{
		Text:   "TimeoutErrors in checkout",
		Source: "operator",
		Tags:   []string{"checkout"},
	})
	raw, err := callHandler(t, handleSignal, s, params)
	if err != nil {
		t.Fatalf("handleSignal: %v", err)
	}
	var resp signalResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Slug == "" || !strings.HasPrefix(resp.Path, "signals/raw/") {
		t.Errorf("response = %+v", resp)
	}

	// Round-trip: read the file back through the store.
	if _, err := store.ReadAs[model.Signal](context.Background(), s, resp.Path); err != nil {
		t.Fatalf("read back: %v", err)
	}
}

func TestHandleSignalRequiresText(t *testing.T) {
	s := markdown.New(t.TempDir())
	params := mustMarshal(t, signalParams{Source: "operator"})
	_, err := callHandler(t, handleSignal, s, params)
	apiErr := mustAPIError(t, err)
	if apiErr.Code != CodeInvalidParams {
		t.Errorf("code = %d, want %d", apiErr.Code, CodeInvalidParams)
	}
}

func TestHandleSignalRejectsUnknownSource(t *testing.T) {
	s := markdown.New(t.TempDir())
	params := mustMarshal(t, signalParams{Text: "x", Source: "bogus"})
	_, err := callHandler(t, handleSignal, s, params)
	apiErr := mustAPIError(t, err)
	if apiErr.Code != CodeInvalidParams {
		t.Errorf("code = %d, want %d", apiErr.Code, CodeInvalidParams)
	}
}

func TestHandleShapeNewWritesPitch(t *testing.T) {
	s := markdown.New(t.TempDir())
	params := mustMarshal(t, shapeParams{Mode: shapeModeNew, Slug: "csv-export", Title: "CSV Export"})
	raw, err := callHandler(t, handleShape, s, params)
	if err != nil {
		t.Fatalf("handleShape: %v", err)
	}
	var resp shapeResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Slug != "csv-export" || resp.Status != "shaping" {
		t.Errorf("got %+v", resp)
	}
}

func TestHandleShapeNewRequiresSlug(t *testing.T) {
	s := markdown.New(t.TempDir())
	params := mustMarshal(t, shapeParams{Mode: shapeModeNew})
	_, err := callHandler(t, handleShape, s, params)
	apiErr := mustAPIError(t, err)
	if apiErr.Code != CodeInvalidParams {
		t.Errorf("code = %d, want %d", apiErr.Code, CodeInvalidParams)
	}
}

func TestHandleShapeFromSignalSeedsPitch(t *testing.T) {
	s := markdown.New(t.TempDir())
	// Seed a signal so from_signal has something to read.
	if _, err := callHandler(t, handleSignal, s, mustMarshal(t, signalParams{
		Text:   "Export takes too long",
		Source: "operator",
		Slug:   "slow-export",
	})); err != nil {
		t.Fatalf("seed signal: %v", err)
	}

	params := mustMarshal(t, shapeParams{
		Mode:       shapeModeFrom,
		SignalPath: "signals/raw/slow-export.md",
		Title:      "Faster Export",
	})
	raw, err := callHandler(t, handleShape, s, params)
	if err != nil {
		t.Fatalf("handleShape from: %v", err)
	}
	var resp shapeResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Slug != "slow-export" {
		t.Errorf("slug = %q", resp.Slug)
	}
}

func TestHandleShapeFromSignalRequiresPath(t *testing.T) {
	s := markdown.New(t.TempDir())
	params := mustMarshal(t, shapeParams{Mode: shapeModeFrom})
	_, err := callHandler(t, handleShape, s, params)
	apiErr := mustAPIError(t, err)
	if apiErr.Code != CodeInvalidParams {
		t.Errorf("code = %d, want %d", apiErr.Code, CodeInvalidParams)
	}
}

func TestHandleShapeFromSignalReportsMissingSignal(t *testing.T) {
	s := markdown.New(t.TempDir())
	params := mustMarshal(t, shapeParams{Mode: shapeModeFrom, SignalPath: "signals/raw/gone.md"})
	_, err := callHandler(t, handleShape, s, params)
	apiErr := mustAPIError(t, err)
	if apiErr.Code != CodeNotFound {
		t.Errorf("code = %d, want %d", apiErr.Code, CodeNotFound)
	}
}

func TestHandleShapeFinalizeSucceedsOnValidPitch(t *testing.T) {
	s := markdown.New(t.TempDir())
	// Seed a shaping pitch with valid body bypassing the API, then
	// finalize via the handler.
	pitch := model.Pitch{
		Slug:   "csv-export",
		Title:  "CSV Export",
		Status: model.PitchStatusShaping,
		Body: `## Problem
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
- [ ] **Card one** — does the thing
`,
	}
	if _, err := s.Write(context.Background(), pitch); err != nil {
		t.Fatalf("seed: %v", err)
	}

	params := mustMarshal(t, shapeParams{Mode: shapeModeFinalize, Slug: "csv-export"})
	raw, err := callHandler(t, handleShape, s, params)
	if err != nil {
		t.Fatalf("handleShape finalize: %v", err)
	}
	var resp shapeResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Status != "shaped" {
		t.Errorf("status = %q, want shaped", resp.Status)
	}
}

func TestHandleShapeFinalizeMissingIngredientIs32002(t *testing.T) {
	s := markdown.New(t.TempDir())
	// Pitch missing 'Solution' so ShapeFinalize surfaces the
	// precise missing section in its error message.
	pitch := model.Pitch{
		Slug:   "incomplete",
		Title:  "Incomplete",
		Status: model.PitchStatusShaping,
		Body: `## Problem
x
## Appetite
x
## Rabbit holes
x
## No-gos
x
## Scope
- [ ] **Card** — yes
`,
	}
	if _, err := s.Write(context.Background(), pitch); err != nil {
		t.Fatalf("seed: %v", err)
	}

	params := mustMarshal(t, shapeParams{Mode: shapeModeFinalize, Slug: "incomplete"})
	_, err := callHandler(t, handleShape, s, params)
	apiErr := mustAPIError(t, err)
	if apiErr.Code != CodeWorkflowViolation {
		t.Errorf("code = %d, want %d", apiErr.Code, CodeWorkflowViolation)
	}
	// The error must name the missing field — the pitch demands it.
	if !strings.Contains(apiErr.Message, "Solution") {
		t.Errorf("message %q does not name the missing 'Solution' section", apiErr.Message)
	}
}

func TestHandleShapeFinalizeMissingScopeIs32002(t *testing.T) {
	s := markdown.New(t.TempDir())
	pitch := model.Pitch{
		Slug:   "no-scope",
		Title:  "No Scope",
		Status: model.PitchStatusShaping,
		Body: `## Problem
x
## Appetite
x
## Solution
x
## Rabbit holes
x
## No-gos
x
`,
	}
	if _, err := s.Write(context.Background(), pitch); err != nil {
		t.Fatalf("seed: %v", err)
	}
	params := mustMarshal(t, shapeParams{Mode: shapeModeFinalize, Slug: "no-scope"})
	_, err := callHandler(t, handleShape, s, params)
	apiErr := mustAPIError(t, err)
	if apiErr.Code != CodeWorkflowViolation {
		t.Errorf("code = %d, want %d", apiErr.Code, CodeWorkflowViolation)
	}
	if !strings.Contains(apiErr.Message, "scope") {
		t.Errorf("message %q does not mention scope", apiErr.Message)
	}
}

func TestHandleShapeFinalizeRequiresSlug(t *testing.T) {
	s := markdown.New(t.TempDir())
	params := mustMarshal(t, shapeParams{Mode: shapeModeFinalize})
	_, err := callHandler(t, handleShape, s, params)
	apiErr := mustAPIError(t, err)
	if apiErr.Code != CodeInvalidParams {
		t.Errorf("code = %d, want %d", apiErr.Code, CodeInvalidParams)
	}
}

func TestHandleShapeRejectsUnknownMode(t *testing.T) {
	s := markdown.New(t.TempDir())
	params := mustMarshal(t, shapeParams{Mode: "doodle"})
	_, err := callHandler(t, handleShape, s, params)
	apiErr := mustAPIError(t, err)
	if apiErr.Code != CodeInvalidParams {
		t.Errorf("code = %d, want %d", apiErr.Code, CodeInvalidParams)
	}
}

func TestHandleShapeRequiresMode(t *testing.T) {
	s := markdown.New(t.TempDir())
	_, err := callHandler(t, handleShape, s, nil)
	apiErr := mustAPIError(t, err)
	if apiErr.Code != CodeInvalidParams {
		t.Errorf("code = %d, want %d", apiErr.Code, CodeInvalidParams)
	}
}
