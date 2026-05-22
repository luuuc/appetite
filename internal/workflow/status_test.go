package workflow

import (
	"errors"
	"testing"
	"time"

	"github.com/luuuc/appetite/internal/model"
)

func TestStatus_NoActiveCycle(t *testing.T) {
	s := newStore(t)
	_, err := Status(s, StatusRequest{})
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestStatus_HappyPath(t *testing.T) {
	s := newStore(t)
	writePitchDirect(t, s, model.Pitch{
		Slug: "csv", Title: "CSV", Status: model.PitchStatusBuilding, Body: shapedBody,
	})
	writeCycleDirect(t, s, model.Cycle{
		ID: "w22", Duration: "5d", Status: model.CycleStatusBuilding,
		Started: fixedNow, Ends: fixedNow.Add(5 * 24 * time.Hour),
		Bets: []model.Bet{{Pitch: "csv", Appetite: model.AppetiteMedium}},
		Passed: []model.Passed{{Pitch: "dark", Reason: "no"}},
	})
	writeCardDirect(t, s, model.Card{
		Slug: "a", Pitch: "csv", Cycle: "w22", Hill: model.HillUphill, Progress: 100, HillUpdatedAt: &fixedNow,
	})
	writeCardDirect(t, s, model.Card{
		Slug: "b", Pitch: "csv", Cycle: "w22", Hill: model.HillUphill, Progress: 0, HillUpdatedAt: &fixedNow,
	})

	res, err := Status(s, StatusRequest{Now: fixedNow.Add(2 * 24 * time.Hour)})
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if res.Cycle.ID != "w22" {
		t.Errorf("cycle id: got %q", res.Cycle.ID)
	}
	if res.DayN != 3 || res.DayTotal != 5 {
		t.Errorf("day: got %d/%d want 3/5", res.DayN, res.DayTotal)
	}
	if len(res.Bets) != 1 {
		t.Fatalf("bets: got %d want 1", len(res.Bets))
	}
	bv := res.Bets[0]
	if bv.CardsDone != 1 || len(bv.Cards) != 2 {
		t.Errorf("bet view: %+v", bv)
	}
	if len(res.Passed) != 1 {
		t.Errorf("passed: got %d want 1", len(res.Passed))
	}
}

func TestStatus_StuckDetection(t *testing.T) {
	s := newStore(t)
	writePitchDirect(t, s, model.Pitch{Slug: "p", Status: model.PitchStatusBuilding, Body: shapedBody})
	writeCycleDirect(t, s, model.Cycle{
		ID: "c", Duration: "5d", Status: model.CycleStatusBuilding,
		Started: fixedNow, Ends: fixedNow.Add(5 * 24 * time.Hour),
		Bets: []model.Bet{{Pitch: "p", Appetite: model.AppetiteMedium}},
	})
	// Card was last touched 96h ago; half-cycle = 60h → stuck.
	stuckTime := fixedNow.Add(-96 * time.Hour)
	writeCardDirect(t, s, model.Card{
		Slug: "stuck", Pitch: "p", Cycle: "c", Hill: model.HillUphill, Progress: 30, HillUpdatedAt: &stuckTime,
	})
	// Fresh card — not stuck.
	freshTime := fixedNow.Add(-1 * time.Hour)
	writeCardDirect(t, s, model.Card{
		Slug: "fresh", Pitch: "p", Cycle: "c", Hill: model.HillUphill, Progress: 50, HillUpdatedAt: &freshTime,
	})

	res, err := Status(s, StatusRequest{Now: fixedNow})
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if len(res.Bets[0].StuckCards) != 1 || res.Bets[0].StuckCards[0].Slug != "stuck" {
		t.Errorf("stuck cards: %+v", res.Bets[0].StuckCards)
	}
}

func TestStatus_ExplicitCycle(t *testing.T) {
	s := newStore(t)
	writeCycleDirect(t, s, model.Cycle{
		ID: "closed", Duration: "5d", Status: model.CycleStatusClosed,
		Started: fixedNow, Ends: fixedNow.Add(5 * 24 * time.Hour),
	})
	res, err := Status(s, StatusRequest{Cycle: "closed", Now: fixedNow})
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if res.Cycle.ID != "closed" {
		t.Errorf("cycle: got %q", res.Cycle.ID)
	}
}

func TestStatus_ExplicitCycleMissing(t *testing.T) {
	s := newStore(t)
	_, err := Status(s, StatusRequest{Cycle: "nope"})
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("want ErrNotFound, got %v", err)
	}
}

func TestStatus_MalformedDurationErrors(t *testing.T) {
	s := newStore(t)
	writeCycleDirect(t, s, model.Cycle{ID: "c", Duration: "5y", Status: model.CycleStatusBuilding})
	if _, err := Status(s, StatusRequest{}); err == nil {
		t.Error("malformed duration should propagate error")
	}
}

func TestIsStuck(t *testing.T) {
	dur := 5 * 24 * time.Hour
	half := dur / 2
	now := fixedNow

	// uphill + older than half → stuck
	old := now.Add(-half - time.Hour)
	if !isStuck(model.Card{Hill: model.HillUphill, HillUpdatedAt: &old}, dur, now) {
		t.Error("uphill + old → expected stuck")
	}
	// uphill + fresh → not stuck
	fresh := now.Add(-time.Hour)
	if isStuck(model.Card{Hill: model.HillUphill, HillUpdatedAt: &fresh}, dur, now) {
		t.Error("uphill + fresh → expected not stuck")
	}
	// downhill → never stuck
	if isStuck(model.Card{Hill: model.HillDownhill, HillUpdatedAt: &old}, dur, now) {
		t.Error("downhill → expected not stuck")
	}
	// nil HillUpdatedAt → not stuck
	if isStuck(model.Card{Hill: model.HillUphill}, dur, now) {
		t.Error("nil HillUpdatedAt → expected not stuck")
	}
}
