package mcp

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/luuuc/appetite/internal/markdown"
	"github.com/luuuc/appetite/internal/model"
	"github.com/luuuc/appetite/internal/store"
	"github.com/luuuc/appetite/internal/workflow"
)

// seedFullCycle stages one cycle with one bet whose card is uphill,
// plus a passed pitch, a cooldown, and a stale stuck card. It's
// enough fixture to exercise every branch of buildStatusResponse.
func seedFullCycle(t *testing.T) store.Store {
	t.Helper()
	s := markdown.New(t.TempDir())
	ctx := context.Background()
	now := time.Date(2026, 5, 22, 0, 0, 0, 0, time.UTC)

	cycle := model.Cycle{
		ID:       "2026-w15",
		Duration: "5d",
		Started:  now.Add(-4 * 24 * time.Hour),
		Ends:     now.Add(1 * 24 * time.Hour),
		Status:   model.CycleStatusBuilding,
		Bets: []model.Bet{
			{Pitch: "csv-export", Appetite: model.AppetiteMedium, BetAt: now.Add(-3 * 24 * time.Hour)},
		},
		Passed: []model.Passed{
			{Pitch: "dark-mode", Reason: "not worth it", PassedAt: now.Add(-2 * 24 * time.Hour)},
		},
	}
	if _, err := s.Write(ctx, cycle); err != nil {
		t.Fatalf("seed cycle: %v", err)
	}
	pitch := model.Pitch{
		Slug:   "csv-export",
		Title:  "CSV Export",
		Status: model.PitchStatusBuilding,
	}
	if _, err := s.Write(ctx, pitch); err != nil {
		t.Fatalf("seed pitch: %v", err)
	}
	stuckAt := now.Add(-4 * 24 * time.Hour)
	card := model.Card{
		Slug:          "export-button",
		Pitch:         "csv-export",
		Cycle:         "2026-w15",
		Hill:          model.HillUphill,
		Progress:      30,
		HillUpdatedAt: &stuckAt, // stuck threshold for 5d = 60h; -96h is stuck
	}
	if _, err := s.Write(ctx, card); err != nil {
		t.Fatalf("seed card: %v", err)
	}
	cooldown := model.Cooldown{
		Cycle:   "2026-w15",
		Started: now.Add(-12 * time.Hour),
		Ends:    now.Add(12 * time.Hour),
		Status:  model.CooldownActive,
	}
	if _, err := s.Write(ctx, cooldown); err != nil {
		t.Fatalf("seed cooldown: %v", err)
	}
	// Independent pitches and a closed cycle so list filters have
	// something to discriminate on.
	if _, err := s.Write(ctx, model.Pitch{Slug: "dark-mode", Title: "Dark Mode", Status: model.PitchStatusPassed}); err != nil {
		t.Fatalf("seed pitch dark-mode: %v", err)
	}
	if _, err := s.Write(ctx, model.Pitch{Slug: "auth", Title: "Auth", Status: model.PitchStatusShaped}); err != nil {
		t.Fatalf("seed pitch auth: %v", err)
	}
	if _, err := s.Write(ctx, model.Cycle{
		ID:       "2025-w50",
		Duration: "5d",
		Started:  now.Add(-365 * 24 * time.Hour),
		Ends:     now.Add(-360 * 24 * time.Hour),
		Status:   model.CycleStatusClosed,
	}); err != nil {
		t.Fatalf("seed closed cycle: %v", err)
	}
	return s
}

func TestHandleStatusReturnsFullSnapshot(t *testing.T) {
	s := seedFullCycle(t)
	raw, err := callHandler(t, handleStatus, s, nil)
	if err != nil {
		t.Fatalf("handleStatus: %v", err)
	}
	var resp statusResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Cycle.ID != "2026-w15" {
		t.Errorf("cycle.id = %q", resp.Cycle.ID)
	}
	if resp.Cycle.Status != "building" {
		t.Errorf("cycle.status = %q", resp.Cycle.Status)
	}
	if len(resp.Bets) != 1 {
		t.Fatalf("bets = %d, want 1", len(resp.Bets))
	}
	bet := resp.Bets[0]
	if bet.Pitch != "csv-export" || bet.Appetite != "medium" || bet.Status != "building" {
		t.Errorf("bet = %+v", bet)
	}
	if len(bet.Cards) != 1 || bet.Cards[0].Slug != "export-button" {
		t.Errorf("bet.cards = %+v", bet.Cards)
	}
	if len(bet.Stuck) != 1 || bet.Stuck[0].Slug != "export-button" {
		t.Errorf("bet.stuck = %+v", bet.Stuck)
	}
	if len(resp.Stuck) != 1 {
		t.Errorf("top-level stuck = %d, want 1", len(resp.Stuck))
	}
	if len(resp.Passed) != 1 || resp.Passed[0].Pitch != "dark-mode" {
		t.Errorf("passed = %+v", resp.Passed)
	}
	if resp.Cooldown == nil || resp.Cooldown.Status != "active" {
		t.Errorf("cooldown = %+v", resp.Cooldown)
	}
	if resp.SyncProposals == nil {
		t.Errorf("sync_proposals must be present (empty slice, not null)")
	}
	if len(resp.SyncProposals) != 0 {
		t.Errorf("sync_proposals = %v, want empty for v0.1", resp.SyncProposals)
	}
}

func TestHandleStatusAcceptsExplicitCycle(t *testing.T) {
	s := seedFullCycle(t)
	params := mustMarshal(t, statusParams{Cycle: "2025-w50"})
	raw, err := callHandler(t, handleStatus, s, params)
	if err != nil {
		t.Fatalf("handleStatus: %v", err)
	}
	var resp statusResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Cycle.ID != "2025-w50" {
		t.Errorf("cycle = %q, want 2025-w50", resp.Cycle.ID)
	}
}

func TestHandleStatusReturnsNotFoundForMissingCycle(t *testing.T) {
	s := seedFullCycle(t)
	params := mustMarshal(t, statusParams{Cycle: "nope"})
	_, err := callHandler(t, handleStatus, s, params)
	apiErr := mustAPIError(t, err)
	if apiErr.Code != CodeNotFound {
		t.Errorf("code = %d, want %d", apiErr.Code, CodeNotFound)
	}
}

func TestHandleStatusReturnsNotFoundWhenNoActiveCycle(t *testing.T) {
	s := markdown.New(t.TempDir())
	_, err := callHandler(t, handleStatus, s, nil)
	apiErr := mustAPIError(t, err)
	if apiErr.Code != CodeNotFound {
		t.Errorf("code = %d, want %d", apiErr.Code, CodeNotFound)
	}
}

func TestHandleStatusRejectsBadParams(t *testing.T) {
	s := seedFullCycle(t)
	_, err := handleStatus(context.Background(), s, json.RawMessage(`{"cycle":1}`))
	apiErr := mustAPIError(t, err)
	if apiErr.Code != CodeInvalidParams {
		t.Errorf("code = %d, want %d", apiErr.Code, CodeInvalidParams)
	}
}

func TestHandlePitchListReturnsAll(t *testing.T) {
	s := seedFullCycle(t)
	raw, err := callHandler(t, handlePitchList, s, nil)
	if err != nil {
		t.Fatalf("handlePitchList: %v", err)
	}
	var resp pitchListResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Pitches) != 3 {
		t.Fatalf("pitches = %d, want 3", len(resp.Pitches))
	}
	// Sorted by slug ascending.
	if resp.Pitches[0].Slug != "auth" || resp.Pitches[1].Slug != "csv-export" || resp.Pitches[2].Slug != "dark-mode" {
		t.Errorf("order = %+v", resp.Pitches)
	}
}

func TestHandlePitchListFiltersByStatus(t *testing.T) {
	s := seedFullCycle(t)
	params := mustMarshal(t, pitchListParams{Status: "shaped"})
	raw, err := callHandler(t, handlePitchList, s, params)
	if err != nil {
		t.Fatalf("handlePitchList: %v", err)
	}
	var resp pitchListResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Pitches) != 1 || resp.Pitches[0].Slug != "auth" {
		t.Errorf("got %+v, want auth only", resp.Pitches)
	}
}

func TestHandlePitchListRejectsBadStatus(t *testing.T) {
	s := seedFullCycle(t)
	params := mustMarshal(t, pitchListParams{Status: "garbage"})
	_, err := callHandler(t, handlePitchList, s, params)
	apiErr := mustAPIError(t, err)
	if apiErr.Code != CodeInvalidParams {
		t.Errorf("code = %d, want %d", apiErr.Code, CodeInvalidParams)
	}
}

func TestHandlePitchListEmptyStore(t *testing.T) {
	s := markdown.New(t.TempDir())
	raw, err := callHandler(t, handlePitchList, s, nil)
	if err != nil {
		t.Fatalf("handlePitchList: %v", err)
	}
	if string(raw) != `{"pitches":[]}` {
		t.Errorf("got %s, want empty pitches list", raw)
	}
}

func TestHandleCycleListReturnsAll(t *testing.T) {
	s := seedFullCycle(t)
	raw, err := callHandler(t, handleCycleList, s, nil)
	if err != nil {
		t.Fatalf("handleCycleList: %v", err)
	}
	var resp cycleListResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Cycles) != 2 {
		t.Fatalf("cycles = %d, want 2", len(resp.Cycles))
	}
	if resp.Cycles[0].ID != "2025-w50" || resp.Cycles[1].ID != "2026-w15" {
		t.Errorf("order = %+v", resp.Cycles)
	}
	if resp.Cycles[1].BetCount != 1 {
		t.Errorf("bet_count = %d, want 1", resp.Cycles[1].BetCount)
	}
}

func TestHandleCycleListFiltersByStatus(t *testing.T) {
	s := seedFullCycle(t)
	params := mustMarshal(t, cycleListParams{Status: "closed"})
	raw, err := callHandler(t, handleCycleList, s, params)
	if err != nil {
		t.Fatalf("handleCycleList: %v", err)
	}
	var resp cycleListResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Cycles) != 1 || resp.Cycles[0].ID != "2025-w50" {
		t.Errorf("got %+v", resp.Cycles)
	}
}

func TestHandleCycleListRejectsBadStatus(t *testing.T) {
	s := seedFullCycle(t)
	params := mustMarshal(t, cycleListParams{Status: "garbage"})
	_, err := callHandler(t, handleCycleList, s, params)
	apiErr := mustAPIError(t, err)
	if apiErr.Code != CodeInvalidParams {
		t.Errorf("code = %d, want %d", apiErr.Code, CodeInvalidParams)
	}
}

// callHandler invokes h and marshals the result so tests can assert
// on JSON shape verbatim.
func callHandler(t *testing.T, h Handler, s store.Store, params json.RawMessage) ([]byte, error) {
	t.Helper()
	res, err := h(context.Background(), s, params)
	if err != nil {
		return nil, err
	}
	data, mErr := json.Marshal(res)
	if mErr != nil {
		t.Fatalf("marshal: %v", mErr)
	}
	return data, nil
}

// mustMarshal wraps json.Marshal with t.Fatalf on failure.
func mustMarshal(t *testing.T, v any) json.RawMessage {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return data
}

// mustAPIError asserts err is a *Error and returns it.
func mustAPIError(t *testing.T, err error) *Error {
	t.Helper()
	if err == nil {
		t.Fatalf("want error, got nil")
	}
	apiErr, ok := err.(*Error)
	if !ok {
		// Errors from workflow flow through errorFromWorkflow at the
		// server boundary, not the handler. Tests that need the
		// translated code can call errorFromWorkflow themselves.
		apiErr = errorFromWorkflow(err)
	}
	return apiErr
}

// Compile-time guarantee that handleStatus uses workflow.Status with
// only fields we accept — silences staticcheck warnings about unused
// imports if we ever stub the handler.
var _ = workflow.Status
