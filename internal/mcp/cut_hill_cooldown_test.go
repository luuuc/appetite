package mcp

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/luuuc/appetite/internal/markdown"
	"github.com/luuuc/appetite/internal/model"
	"github.com/luuuc/appetite/internal/store"
)

// seedBetReady puts a pitch into `bet` status and stages a cycle whose
// bets list includes it — enough fixture for cut.
func seedBetReady(t *testing.T, s store.Store, pitch, cycle string, body string) {
	t.Helper()
	if _, err := s.Write(context.Background(), model.Pitch{
		Slug:   pitch,
		Title:  pitch,
		Status: model.PitchStatusBet,
		Body:   body,
	}); err != nil {
		t.Fatalf("seed pitch: %v", err)
	}
	if _, err := s.Write(context.Background(), model.Cycle{
		ID: cycle, Duration: "5d", Status: model.CycleStatusBuilding,
		Bets: []model.Bet{{Pitch: pitch, Appetite: model.AppetiteSmall}},
	}); err != nil {
		t.Fatalf("seed cycle: %v", err)
	}
}

func TestHandleCutWritesCardsAndFlipsPitch(t *testing.T) {
	s := markdown.New(t.TempDir())
	body := `## Problem
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
- [ ] **Card one** — first
- [ ] **Card two** — second
`
	seedBetReady(t, s, "csv-export", "2026-w20", body)
	params := mustMarshal(t, cutParams{Pitch: "csv-export"})
	raw, err := callHandler(t, handleCut, s, params)
	if err != nil {
		t.Fatalf("handleCut: %v", err)
	}
	var resp cutResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Pitch != "csv-export" || resp.Cycle != "2026-w20" {
		t.Errorf("got %+v", resp)
	}
	if len(resp.Cards) != 2 {
		t.Fatalf("cards = %d, want 2", len(resp.Cards))
	}
}

func TestHandleCutRequiresPitch(t *testing.T) {
	s := markdown.New(t.TempDir())
	_, err := callHandler(t, handleCut, s, mustMarshal(t, cutParams{}))
	apiErr := mustAPIError(t, err)
	if apiErr.Code != CodeInvalidParams {
		t.Errorf("code = %d, want %d", apiErr.Code, CodeInvalidParams)
	}
}

func TestHandleCutReportsMissingPitch(t *testing.T) {
	s := markdown.New(t.TempDir())
	_, err := callHandler(t, handleCut, s, mustMarshal(t, cutParams{Pitch: "ghost"}))
	apiErr := mustAPIError(t, err)
	if apiErr.Code != CodeNotFound {
		t.Errorf("code = %d, want %d", apiErr.Code, CodeNotFound)
	}
}

// --- hill ----------------------------------------------------------------

func seedCard(t *testing.T, s store.Store, slug, pitch, cycle string) {
	t.Helper()
	now := time.Date(2026, 5, 22, 0, 0, 0, 0, time.UTC)
	if _, err := s.Write(context.Background(), model.Card{
		Slug:          slug,
		Pitch:         pitch,
		Cycle:         cycle,
		Hill:          model.HillUphill,
		Progress:      0,
		HillUpdatedAt: &now,
	}); err != nil {
		t.Fatalf("seed card: %v", err)
	}
}

func TestHandleHillUpdatesPosition(t *testing.T) {
	s := markdown.New(t.TempDir())
	if _, err := s.Write(context.Background(), model.Cycle{
		ID: "c", Duration: "5d", Status: model.CycleStatusBuilding,
		Bets: []model.Bet{{Pitch: "p", Appetite: model.AppetiteSmall}},
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if _, err := s.Write(context.Background(), model.Pitch{Slug: "p", Title: "p", Status: model.PitchStatusBuilding}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	seedCard(t, s, "card", "p", "c")

	prog := 50
	params := mustMarshal(t, hillParams{Card: "card", Position: "downhill", Progress: &prog})
	raw, err := callHandler(t, handleHill, s, params)
	if err != nil {
		t.Fatalf("handleHill: %v", err)
	}
	var resp hillResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Card.Hill != "downhill" || resp.Card.Progress != 50 {
		t.Errorf("response = %+v", resp)
	}
}

func TestHandleHillProgress100WithoutDoneIs32002(t *testing.T) {
	s := markdown.New(t.TempDir())
	if _, err := s.Write(context.Background(), model.Cycle{
		ID: "c", Duration: "5d", Status: model.CycleStatusBuilding,
		Bets: []model.Bet{{Pitch: "p", Appetite: model.AppetiteSmall}},
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if _, err := s.Write(context.Background(), model.Pitch{Slug: "p", Title: "p", Status: model.PitchStatusBuilding}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	seedCard(t, s, "card", "p", "c")

	prog := 100
	params := mustMarshal(t, hillParams{Card: "card", Progress: &prog, Done: false})
	_, err := callHandler(t, handleHill, s, params)
	apiErr := mustAPIError(t, err)
	if apiErr.Code != CodeWorkflowViolation {
		t.Errorf("code = %d, want %d", apiErr.Code, CodeWorkflowViolation)
	}
}

func TestHandleHillShipsCardAndCascades(t *testing.T) {
	s := markdown.New(t.TempDir())
	if _, err := s.Write(context.Background(), model.Cycle{
		ID: "c", Duration: "5d", Status: model.CycleStatusBuilding,
		Bets: []model.Bet{{Pitch: "p", Appetite: model.AppetiteSmall}},
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if _, err := s.Write(context.Background(), model.Pitch{Slug: "p", Title: "p", Status: model.PitchStatusBuilding}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	seedCard(t, s, "only", "p", "c")

	prog := 100
	params := mustMarshal(t, hillParams{Card: "only", Progress: &prog, Done: true})
	raw, err := callHandler(t, handleHill, s, params)
	if err != nil {
		t.Fatalf("handleHill: %v", err)
	}
	var resp hillResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.PitchShipped == nil || resp.PitchShipped.Status != "shipped" {
		t.Errorf("pitch_shipped = %+v", resp.PitchShipped)
	}
	if resp.CycleShipped == nil || resp.CycleShipped.Status != "shipping" {
		t.Errorf("cycle_shipped = %+v", resp.CycleShipped)
	}
}

func TestHandleHillRequiresCard(t *testing.T) {
	s := markdown.New(t.TempDir())
	_, err := callHandler(t, handleHill, s, mustMarshal(t, hillParams{}))
	apiErr := mustAPIError(t, err)
	if apiErr.Code != CodeInvalidParams {
		t.Errorf("code = %d, want %d", apiErr.Code, CodeInvalidParams)
	}
}

func TestHandleHillReportsMissingCard(t *testing.T) {
	s := markdown.New(t.TempDir())
	prog := 10
	params := mustMarshal(t, hillParams{Card: "ghost", Progress: &prog})
	_, err := callHandler(t, handleHill, s, params)
	apiErr := mustAPIError(t, err)
	if apiErr.Code != CodeNotFound {
		t.Errorf("code = %d, want %d", apiErr.Code, CodeNotFound)
	}
}

// --- cooldown ------------------------------------------------------------

func seedShippingCycle(t *testing.T, s store.Store) {
	t.Helper()
	if _, err := s.Write(context.Background(), model.Pitch{Slug: "p", Title: "p", Status: model.PitchStatusShipped}); err != nil {
		t.Fatalf("seed pitch: %v", err)
	}
	if _, err := s.Write(context.Background(), model.Cycle{
		ID: "c", Duration: "5d", Status: model.CycleStatusShipping,
		Bets: []model.Bet{{Pitch: "p", Appetite: model.AppetiteSmall}},
	}); err != nil {
		t.Fatalf("seed cycle: %v", err)
	}
}

func TestHandleCooldownOpenAndClose(t *testing.T) {
	s := markdown.New(t.TempDir())
	seedShippingCycle(t, s)

	openRaw, err := callHandler(t, handleCooldown, s, mustMarshal(t, cooldownParams{Op: cooldownOpOpen}))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	var openResp cooldownResponse
	if err := json.Unmarshal(openRaw, &openResp); err != nil {
		t.Fatalf("decode open: %v", err)
	}
	if openResp.Cooldown.Status != "active" || openResp.Cycle.Status != "cooldown" {
		t.Errorf("open response = %+v", openResp)
	}

	closeRaw, err := callHandler(t, handleCooldown, s, mustMarshal(t, cooldownParams{Op: cooldownOpClose}))
	if err != nil {
		t.Fatalf("close: %v", err)
	}
	var closeResp cooldownResponse
	if err := json.Unmarshal(closeRaw, &closeResp); err != nil {
		t.Fatalf("decode close: %v", err)
	}
	if closeResp.Cooldown.Status != "closed" || closeResp.Cycle.Status != "closed" {
		t.Errorf("close response = %+v", closeResp)
	}
}

func TestHandleCooldownRejectsUnknownOp(t *testing.T) {
	s := markdown.New(t.TempDir())
	_, err := callHandler(t, handleCooldown, s, mustMarshal(t, cooldownParams{Op: "shrug"}))
	apiErr := mustAPIError(t, err)
	if apiErr.Code != CodeInvalidParams {
		t.Errorf("code = %d, want %d", apiErr.Code, CodeInvalidParams)
	}
}

func TestHandleCooldownRequiresOp(t *testing.T) {
	s := markdown.New(t.TempDir())
	_, err := callHandler(t, handleCooldown, s, nil)
	apiErr := mustAPIError(t, err)
	if apiErr.Code != CodeInvalidParams {
		t.Errorf("code = %d, want %d", apiErr.Code, CodeInvalidParams)
	}
}

func TestHandleCooldownOpenReportsNotFoundWhenNoActiveCycle(t *testing.T) {
	s := markdown.New(t.TempDir())
	_, err := callHandler(t, handleCooldown, s, mustMarshal(t, cooldownParams{Op: cooldownOpOpen}))
	apiErr := mustAPIError(t, err)
	if apiErr.Code != CodeNotFound {
		t.Errorf("code = %d, want %d", apiErr.Code, CodeNotFound)
	}
}

func TestHandleCooldownCloseReportsViolationWhenNoCooldownOpen(t *testing.T) {
	s := markdown.New(t.TempDir())
	// Active cycle but no cooldown file → workflow returns an
	// ErrInvalidTransition wrapping ErrNotFound.
	if _, err := s.Write(context.Background(), model.Cycle{ID: "c", Duration: "5d", Status: model.CycleStatusBuilding}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	_, err := callHandler(t, handleCooldown, s, mustMarshal(t, cooldownParams{Op: cooldownOpClose}))
	apiErr := mustAPIError(t, err)
	if apiErr.Code != CodeWorkflowViolation {
		t.Errorf("code = %d, want %d", apiErr.Code, CodeWorkflowViolation)
	}
}
