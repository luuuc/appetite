package mcp

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/luuuc/appetite/internal/markdown"
	"github.com/luuuc/appetite/internal/model"
	"github.com/luuuc/appetite/internal/store"
)

// seedShapedPitch writes a pitch already in `shaped` status so bet/pass
// tests have a transition-ready entity without going through the full
// shape→finalize path.
func seedShapedPitch(t *testing.T, s store.Store, slug string) {
	t.Helper()
	if _, err := s.Write(context.Background(), model.Pitch{
		Slug:   slug,
		Title:  slug,
		Status: model.PitchStatusShaped,
	}); err != nil {
		t.Fatalf("seed pitch %s: %v", slug, err)
	}
}

// --- cycle_new -----------------------------------------------------------

func TestHandleCycleNewWritesCycle(t *testing.T) {
	s := markdown.New(t.TempDir())
	params := mustMarshal(t, cycleNewParams{ID: "2026-w20", Duration: "5d"})
	raw, err := callHandler(t, handleCycleNew, s, params)
	if err != nil {
		t.Fatalf("handleCycleNew: %v", err)
	}
	var resp cycleNewResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.ID != "2026-w20" || resp.Status != "building" || resp.Duration != "5d" {
		t.Errorf("response = %+v", resp)
	}
	if _, err := store.ReadAs[model.Cycle](context.Background(), s, model.Cycle{ID: "2026-w20"}.Path()); err != nil {
		t.Fatalf("cycle not on disk: %v", err)
	}
}

func TestHandleCycleNewRequiresID(t *testing.T) {
	s := markdown.New(t.TempDir())
	params := mustMarshal(t, cycleNewParams{Duration: "5d"})
	_, err := callHandler(t, handleCycleNew, s, params)
	apiErr := mustAPIError(t, err)
	if apiErr.Code != CodeInvalidParams {
		t.Errorf("code = %d, want %d", apiErr.Code, CodeInvalidParams)
	}
}

func TestHandleCycleNewRequiresDuration(t *testing.T) {
	s := markdown.New(t.TempDir())
	params := mustMarshal(t, cycleNewParams{ID: "2026-w20"})
	_, err := callHandler(t, handleCycleNew, s, params)
	apiErr := mustAPIError(t, err)
	if apiErr.Code != CodeInvalidParams {
		t.Errorf("code = %d, want %d", apiErr.Code, CodeInvalidParams)
	}
}

func TestHandleCycleNewRejectsBadDuration(t *testing.T) {
	s := markdown.New(t.TempDir())
	params := mustMarshal(t, cycleNewParams{ID: "2026-w20", Duration: "5x"})
	_, err := callHandler(t, handleCycleNew, s, params)
	apiErr := mustAPIError(t, err)
	if apiErr.Code != CodeInternalError {
		t.Errorf("code = %d, want %d", apiErr.Code, CodeInternalError)
	}
}

func TestHandleCycleNewRefusesSecondOpenCycle(t *testing.T) {
	s := markdown.New(t.TempDir())
	if _, err := s.Write(context.Background(), model.Cycle{
		ID: "2026-w19", Duration: "5d", Status: model.CycleStatusBuilding,
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	params := mustMarshal(t, cycleNewParams{ID: "2026-w20", Duration: "5d"})
	_, err := callHandler(t, handleCycleNew, s, params)
	apiErr := mustAPIError(t, err)
	if apiErr.Code != CodeWorkflowViolation {
		t.Errorf("code = %d, want %d", apiErr.Code, CodeWorkflowViolation)
	}
}

// --- bet -----------------------------------------------------------------

func TestHandleBetFlipsPitchAndAppendsBet(t *testing.T) {
	s := markdown.New(t.TempDir())
	seedShapedPitch(t, s, "csv-export")
	if _, err := s.Write(context.Background(), model.Cycle{
		ID: "2026-w20", Duration: "5d", Status: model.CycleStatusBuilding,
	}); err != nil {
		t.Fatalf("seed cycle: %v", err)
	}
	params := mustMarshal(t, betParams{Pitch: "csv-export", Cycle: "2026-w20", Appetite: "medium"})
	raw, err := callHandler(t, handleBet, s, params)
	if err != nil {
		t.Fatalf("handleBet: %v", err)
	}
	var resp betResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Pitch != "csv-export" || resp.Cycle != "2026-w20" || resp.Appetite != "medium" {
		t.Errorf("response = %+v", resp)
	}

	p, err := store.ReadAs[model.Pitch](context.Background(), s, model.Pitch{Slug: "csv-export"}.Path())
	if err != nil {
		t.Fatalf("read pitch: %v", err)
	}
	if p.Status != model.PitchStatusBet {
		t.Errorf("pitch status = %s, want bet", p.Status)
	}
}

func TestHandleBetRequiresFields(t *testing.T) {
	s := markdown.New(t.TempDir())
	cases := []struct {
		name   string
		params betParams
	}{
		{"missing pitch", betParams{Cycle: "c", Appetite: "small"}},
		{"missing cycle", betParams{Pitch: "p", Appetite: "small"}},
		{"missing appetite", betParams{Pitch: "p", Cycle: "c"}},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			_, err := callHandler(t, handleBet, s, mustMarshal(t, tc.params))
			apiErr := mustAPIError(t, err)
			if apiErr.Code != CodeInvalidParams {
				t.Errorf("code = %d, want %d", apiErr.Code, CodeInvalidParams)
			}
		})
	}
}

func TestHandleBetRejectsBadAppetite(t *testing.T) {
	s := markdown.New(t.TempDir())
	seedShapedPitch(t, s, "p")
	if _, err := s.Write(context.Background(), model.Cycle{
		ID: "c", Duration: "5d", Status: model.CycleStatusBuilding,
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	params := mustMarshal(t, betParams{Pitch: "p", Cycle: "c", Appetite: "huge"})
	_, err := callHandler(t, handleBet, s, params)
	apiErr := mustAPIError(t, err)
	if apiErr.Code != CodeInternalError {
		t.Errorf("code = %d, want %d", apiErr.Code, CodeInternalError)
	}
}

func TestHandleBetReportsMissingPitch(t *testing.T) {
	s := markdown.New(t.TempDir())
	if _, err := s.Write(context.Background(), model.Cycle{
		ID: "c", Duration: "5d", Status: model.CycleStatusBuilding,
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	params := mustMarshal(t, betParams{Pitch: "missing", Cycle: "c", Appetite: "small"})
	_, err := callHandler(t, handleBet, s, params)
	apiErr := mustAPIError(t, err)
	if apiErr.Code != CodeNotFound {
		t.Errorf("code = %d, want %d", apiErr.Code, CodeNotFound)
	}
}

func TestHandleBetRejectsWrongPitchStatus(t *testing.T) {
	s := markdown.New(t.TempDir())
	if _, err := s.Write(context.Background(), model.Pitch{Slug: "p", Title: "p", Status: model.PitchStatusShipped}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if _, err := s.Write(context.Background(), model.Cycle{ID: "c", Duration: "5d", Status: model.CycleStatusBuilding}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	params := mustMarshal(t, betParams{Pitch: "p", Cycle: "c", Appetite: "small"})
	_, err := callHandler(t, handleBet, s, params)
	apiErr := mustAPIError(t, err)
	if apiErr.Code != CodeWorkflowViolation {
		t.Errorf("code = %d, want %d", apiErr.Code, CodeWorkflowViolation)
	}
}

// --- pass ----------------------------------------------------------------

func TestHandlePassFlipsAndRecords(t *testing.T) {
	s := markdown.New(t.TempDir())
	seedShapedPitch(t, s, "dark-mode")
	if _, err := s.Write(context.Background(), model.Cycle{
		ID: "c", Duration: "5d", Status: model.CycleStatusBuilding,
	}); err != nil {
		t.Fatalf("seed cycle: %v", err)
	}
	params := mustMarshal(t, passParams{Pitch: "dark-mode", Reason: "not worth it"})
	raw, err := callHandler(t, handlePass, s, params)
	if err != nil {
		t.Fatalf("handlePass: %v", err)
	}
	var resp passResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Pitch != "dark-mode" || resp.CyclePath == "" {
		t.Errorf("response = %+v", resp)
	}
}

func TestHandlePassRequiresFields(t *testing.T) {
	s := markdown.New(t.TempDir())
	cases := []struct {
		name   string
		params passParams
	}{
		{"missing pitch", passParams{Reason: "x"}},
		{"missing reason", passParams{Pitch: "p"}},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			_, err := callHandler(t, handlePass, s, mustMarshal(t, tc.params))
			apiErr := mustAPIError(t, err)
			if apiErr.Code != CodeInvalidParams {
				t.Errorf("code = %d, want %d", apiErr.Code, CodeInvalidParams)
			}
		})
	}
}

func TestHandlePassNotFoundWhenPitchMissing(t *testing.T) {
	s := markdown.New(t.TempDir())
	params := mustMarshal(t, passParams{Pitch: "ghost", Reason: "n/a"})
	_, err := callHandler(t, handlePass, s, params)
	apiErr := mustAPIError(t, err)
	if apiErr.Code != CodeNotFound {
		t.Errorf("code = %d, want %d", apiErr.Code, CodeNotFound)
	}
}

func TestHandlePassRejectsWrongStatus(t *testing.T) {
	s := markdown.New(t.TempDir())
	if _, err := s.Write(context.Background(), model.Pitch{Slug: "p", Title: "p", Status: model.PitchStatusShipped}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	params := mustMarshal(t, passParams{Pitch: "p", Reason: "no"})
	_, err := callHandler(t, handlePass, s, params)
	apiErr := mustAPIError(t, err)
	if apiErr.Code != CodeWorkflowViolation {
		t.Errorf("code = %d, want %d", apiErr.Code, CodeWorkflowViolation)
	}
}
