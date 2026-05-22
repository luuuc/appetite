package workflow

import (
	"errors"
	"testing"
	"time"

	"github.com/luuuc/appetite/internal/model"
)

// These tests exercise error-return branches that are otherwise
// only reachable on a hostile filesystem. The faultStore wrapper
// makes them deterministic.

func TestBet_CycleWriteFails_LeavesPitchFlipped(t *testing.T) {
	s := setupForBet(t)
	fs := &faultStore{inner: s, writeMatch: "cycles/"}
	_, err := Bet(fs, BetRequest{Pitch: "csv", Cycle: "w22", Appetite: model.AppetiteMedium})
	if !errors.Is(err, errInjected) {
		t.Errorf("want injected, got %v", err)
	}
}

func TestPass_CycleWriteFails_LeavesPitchFlipped(t *testing.T) {
	s := newStore(t)
	writePitchDirect(t, s, model.Pitch{Slug: "p", Status: model.PitchStatusShaped, Body: shapedBody})
	writeCycleDirect(t, s, model.Cycle{ID: "c", Duration: "5d", Status: model.CycleStatusBuilding})
	fs := &faultStore{inner: s, writeMatch: "cycles/"}
	_, err := Pass(fs, PassRequest{Pitch: "p", Reason: "x"})
	if !errors.Is(err, errInjected) {
		t.Errorf("want injected, got %v", err)
	}
}

func TestPass_PitchWriteFails(t *testing.T) {
	s := newStore(t)
	writePitchDirect(t, s, model.Pitch{Slug: "p", Status: model.PitchStatusShaped, Body: shapedBody})
	fs := &faultStore{inner: s, writeMatch: "pitches/"}
	_, err := Pass(fs, PassRequest{Pitch: "p", Reason: "x"})
	if !errors.Is(err, errInjected) {
		t.Errorf("want injected, got %v", err)
	}
}

func TestPass_ListErrorPropagates(t *testing.T) {
	s := newStore(t)
	writePitchDirect(t, s, model.Pitch{Slug: "p", Status: model.PitchStatusShaped, Body: shapedBody})
	fs := &faultStore{inner: s, listMatch: "cycle"}
	_, err := Pass(fs, PassRequest{Pitch: "p", Reason: "x"})
	if !errors.Is(err, errInjected) {
		t.Errorf("want injected, got %v", err)
	}
}

func TestCut_CardWriteFails_RollsBack(t *testing.T) {
	s := setupForCut(t, "")
	fs := &faultStore{inner: s, writeMatch: "cards/"}
	if _, err := Cut(fs, CutRequest{Pitch: "csv"}); !errors.Is(err, errInjected) {
		t.Errorf("want injected, got %v", err)
	}
}

func TestCut_PitchWriteFails_RollsBack(t *testing.T) {
	s := setupForCut(t, "")
	fs := &faultStore{inner: s, writeMatch: "pitches/"}
	if _, err := Cut(fs, CutRequest{Pitch: "csv"}); !errors.Is(err, errInjected) {
		t.Errorf("want injected, got %v", err)
	}
}

func TestCut_ListErrorPropagates(t *testing.T) {
	s := setupForCut(t, "")
	fs := &faultStore{inner: s, listMatch: "cycle"}
	if _, err := Cut(fs, CutRequest{Pitch: "csv"}); !errors.Is(err, errInjected) {
		t.Errorf("want injected, got %v", err)
	}
}

func TestHill_CycleReadFails(t *testing.T) {
	s := setupForHill(t)
	fs := &faultStore{inner: s, readMatch: "cycles/w22/cycle.yml"}
	_, err := Hill(fs, HillRequest{Card: "a", Position: model.HillDownhill})
	if !errors.Is(err, errInjected) {
		t.Errorf("want injected, got %v", err)
	}
}

func TestHill_PitchReadFails(t *testing.T) {
	s := newStore(t)
	writePitchDirect(t, s, model.Pitch{Slug: "csv", Status: model.PitchStatusBuilding, Body: shapedBody})
	writeCycleDirect(t, s, model.Cycle{
		ID: "w22", Duration: "5d", Status: model.CycleStatusBuilding,
		Bets: []model.Bet{{Pitch: "csv", Appetite: model.AppetiteMedium}},
	})
	writeCardDirect(t, s, model.Card{Slug: "only", Pitch: "csv", Cycle: "w22", Hill: model.HillUphill, HillUpdatedAt: &fixedNow})
	// One card; shipping it triggers cascade, which reads the pitch → fault.
	fs := &faultStore{inner: s, readMatch: "pitches/csv.md"}
	_, err := Hill(fs, HillRequest{Card: "only", Progress: ptr(100), Done: true})
	if !errors.Is(err, errInjected) {
		t.Errorf("want injected, got %v", err)
	}
}

func TestHill_ListErrorOnFindCard(t *testing.T) {
	s := setupForHill(t)
	fs := &faultStore{inner: s, listMatch: "card"}
	_, err := Hill(fs, HillRequest{Card: "a", Position: model.HillDownhill})
	if !errors.Is(err, errInjected) {
		t.Errorf("want injected, got %v", err)
	}
}

func TestStatus_ListErrorPropagates(t *testing.T) {
	s := newStore(t)
	writeCycleDirect(t, s, model.Cycle{ID: "c", Duration: "5d", Status: model.CycleStatusBuilding})
	fs := &faultStore{inner: s, listMatch: "card"}
	_, err := Status(fs, StatusRequest{})
	if !errors.Is(err, errInjected) {
		t.Errorf("want injected, got %v", err)
	}
}

func TestStatus_BetPitchReadFails(t *testing.T) {
	s := newStore(t)
	writeCycleDirect(t, s, model.Cycle{
		ID: "c", Duration: "5d", Status: model.CycleStatusBuilding,
		Started: fixedNow, Ends: fixedNow.Add(5 * 24 * time.Hour),
		Bets: []model.Bet{{Pitch: "ghost"}},
	})
	fs := &faultStore{inner: s, readMatch: "pitches/"}
	_, err := Status(fs, StatusRequest{Now: fixedNow})
	if err == nil {
		t.Error("expected error")
	}
}

func TestFindActiveCycle_ListErrorPropagates(t *testing.T) {
	s := newStore(t)
	fs := &faultStore{inner: s, listMatch: "any"}
	if _, err := findActiveCycle(fs); !errors.Is(err, errInjected) {
		t.Errorf("want injected, got %v", err)
	}
}

func TestCooldownOpen_CooldownWriteFails(t *testing.T) {
	s := setupForCooldown(t)
	fs := &faultStore{inner: s, writeMatch: "cooldown.yml"}
	_, err := CooldownOpen(fs, CooldownOpenRequest{})
	if !errors.Is(err, errInjected) {
		t.Errorf("want injected, got %v", err)
	}
}

func TestCooldownOpen_CycleWriteFails(t *testing.T) {
	s := setupForCooldown(t)
	fs := &faultStore{inner: s, writeMatch: "cycle.yml"}
	_, err := CooldownOpen(fs, CooldownOpenRequest{})
	if !errors.Is(err, errInjected) {
		t.Errorf("want injected, got %v", err)
	}
}

func TestCooldownClose_CooldownWriteFails(t *testing.T) {
	s := setupForCooldown(t)
	if _, err := CooldownOpen(s, CooldownOpenRequest{}); err != nil {
		t.Fatalf("Open: %v", err)
	}
	fs := &faultStore{inner: s, writeMatch: "cooldown.yml"}
	_, err := CooldownClose(fs, CooldownCloseRequest{})
	if !errors.Is(err, errInjected) {
		t.Errorf("want injected, got %v", err)
	}
}

func TestCooldownClose_CycleWriteFails(t *testing.T) {
	s := setupForCooldown(t)
	if _, err := CooldownOpen(s, CooldownOpenRequest{}); err != nil {
		t.Fatalf("Open: %v", err)
	}
	fs := &faultStore{inner: s, writeMatch: "cycle.yml"}
	_, err := CooldownClose(fs, CooldownCloseRequest{})
	if !errors.Is(err, errInjected) {
		t.Errorf("want injected, got %v", err)
	}
}

func TestPitchCardsAllDone_ListError(t *testing.T) {
	s := newStore(t)
	fs := &faultStore{inner: s, listMatch: "any"}
	if _, err := pitchCardsAllDone(fs, "p", "c"); !errors.Is(err, errInjected) {
		t.Errorf("want injected, got %v", err)
	}
}

func TestCycleBetsAllShipped_PitchReadError(t *testing.T) {
	s := newStore(t)
	c := model.Cycle{ID: "c", Bets: []model.Bet{{Pitch: "ghost"}}}
	if _, err := cycleBetsAllShipped(s, c); err == nil {
		t.Error("expected error reading missing pitch")
	}
}

func TestListCardsForCycle_ListError(t *testing.T) {
	s := newStore(t)
	fs := &faultStore{inner: s, listMatch: "any"}
	if _, err := listCardsForCycle(fs, "c"); !errors.Is(err, errInjected) {
		t.Errorf("want injected, got %v", err)
	}
}

func TestResolveCutCycle_ListError(t *testing.T) {
	s := newStore(t)
	fs := &faultStore{inner: s, listMatch: "any"}
	if _, err := resolveCutCycle(fs, "p", ""); !errors.Is(err, errInjected) {
		t.Errorf("want injected, got %v", err)
	}
}

func TestFindCard_ListError(t *testing.T) {
	s := newStore(t)
	fs := &faultStore{inner: s, listMatch: "any"}
	if _, err := findCard(fs, "x", ""); !errors.Is(err, errInjected) {
		t.Errorf("want injected, got %v", err)
	}
}

func TestRefuseIfBuildingPitches_PitchReadError(t *testing.T) {
	s := newStore(t)
	if err := refuseIfBuildingPitches(s, model.Cycle{ID: "c", Bets: []model.Bet{{Pitch: "ghost"}}}); err == nil {
		t.Error("expected error reading missing pitch")
	}
}

func TestShape_NewExistingPitchReadError(t *testing.T) {
	s := newStore(t)
	fs := &faultStore{inner: s, readMatch: "pitches/"}
	_, err := ShapeNew(fs, ShapeNewRequest{Slug: "x", Title: "X"})
	if !errors.Is(err, errInjected) {
		t.Errorf("want injected, got %v", err)
	}
}

func TestShape_FromExistingPitchReadError(t *testing.T) {
	s := newStore(t)
	writeSignalDirect(t, s, model.Signal{Slug: "src", Captured: fixedNow, Body: "x"})
	fs := &faultStore{inner: s, readMatch: "pitches/"}
	_, err := ShapeFrom(fs, ShapeFromRequest{SignalPath: "signals/raw/src.md"})
	if !errors.Is(err, errInjected) {
		t.Errorf("want injected, got %v", err)
	}
}

func TestSignal_WriteFails(t *testing.T) {
	s := newStore(t)
	fs := &faultStore{inner: s, writeMatch: "signals/"}
	_, err := SignalAdd(fs, SignalAddRequest{Body: "x"})
	if !errors.Is(err, errInjected) {
		t.Errorf("want injected, got %v", err)
	}
}

func TestShapeNew_WriteFails(t *testing.T) {
	s := newStore(t)
	fs := &faultStore{inner: s, writeMatch: "pitches/"}
	_, err := ShapeNew(fs, ShapeNewRequest{Slug: "x", Title: "X"})
	if !errors.Is(err, errInjected) {
		t.Errorf("want injected, got %v", err)
	}
}

func TestResolveCycle_ListErrorPropagates(t *testing.T) {
	s := newStore(t)
	fs := &faultStore{inner: s, listMatch: "any"}
	if _, err := resolveCycle(fs, ""); !errors.Is(err, errInjected) {
		t.Errorf("want injected, got %v", err)
	}
}

func TestCooldownClose_CycleAlreadyClosed(t *testing.T) {
	s := setupForCooldown(t)
	if _, err := CooldownOpen(s, CooldownOpenRequest{}); err != nil {
		t.Fatalf("Open: %v", err)
	}
	if _, err := CooldownClose(s, CooldownCloseRequest{}); err != nil {
		t.Fatalf("Close: %v", err)
	}
	// Cycle is now closed — second close is rejected by state machine.
	_, err := CooldownClose(s, CooldownCloseRequest{Cycle: "w22"})
	if err == nil {
		t.Error("expected error closing already-closed cycle")
	}
}

func TestHill_CardWriteFails(t *testing.T) {
	s := setupForHill(t)
	fs := &faultStore{inner: s, writeMatch: "cards/"}
	_, err := Hill(fs, HillRequest{Card: "a", Position: model.HillDownhill})
	if !errors.Is(err, errInjected) {
		t.Errorf("want injected, got %v", err)
	}
}

func TestHill_PitchWriteFails_DuringCascade(t *testing.T) {
	s := newStore(t)
	writePitchDirect(t, s, model.Pitch{Slug: "csv", Status: model.PitchStatusBuilding, Body: shapedBody})
	writeCycleDirect(t, s, model.Cycle{
		ID: "w22", Duration: "5d", Status: model.CycleStatusBuilding,
		Bets: []model.Bet{{Pitch: "csv", Appetite: model.AppetiteMedium}},
	})
	writeCardDirect(t, s, model.Card{Slug: "only", Pitch: "csv", Cycle: "w22", Hill: model.HillUphill, HillUpdatedAt: &fixedNow})
	fs := &faultStore{inner: s, writeMatch: "pitches/"}
	_, err := Hill(fs, HillRequest{Card: "only", Progress: ptr(100), Done: true})
	if !errors.Is(err, errInjected) {
		t.Errorf("want injected, got %v", err)
	}
}

func TestHill_CycleWriteFails_DuringCascade(t *testing.T) {
	s := newStore(t)
	writePitchDirect(t, s, model.Pitch{Slug: "csv", Status: model.PitchStatusBuilding, Body: shapedBody})
	writeCycleDirect(t, s, model.Cycle{
		ID: "w22", Duration: "5d", Status: model.CycleStatusBuilding,
		Bets: []model.Bet{{Pitch: "csv", Appetite: model.AppetiteMedium}},
	})
	writeCardDirect(t, s, model.Card{Slug: "only", Pitch: "csv", Cycle: "w22", Hill: model.HillUphill, HillUpdatedAt: &fixedNow})
	fs := &faultStore{inner: s, writeMatch: "cycle.yml"}
	_, err := Hill(fs, HillRequest{Card: "only", Progress: ptr(100), Done: true})
	if !errors.Is(err, errInjected) {
		t.Errorf("want injected, got %v", err)
	}
}

func TestHill_BadPitchTransition(t *testing.T) {
	// Set a pitch to passed but still have an at-100 card — the cascade
	// will try to transition passed → shipped which is invalid.
	s := newStore(t)
	writePitchDirect(t, s, model.Pitch{Slug: "csv", Status: model.PitchStatusPassed, Body: shapedBody})
	writeCycleDirect(t, s, model.Cycle{
		ID: "w22", Duration: "5d", Status: model.CycleStatusBuilding,
		Bets: []model.Bet{{Pitch: "csv", Appetite: model.AppetiteMedium}},
	})
	writeCardDirect(t, s, model.Card{Slug: "only", Pitch: "csv", Cycle: "w22", Hill: model.HillUphill, HillUpdatedAt: &fixedNow})
	_, err := Hill(s, HillRequest{Card: "only", Progress: ptr(100), Done: true})
	if !errors.Is(err, ErrInvalidTransition) {
		t.Errorf("want ErrInvalidTransition, got %v", err)
	}
}

func TestHill_ReHillAlreadyShippedPitch(t *testing.T) {
	// Card update succeeds; pitch is already shipped (so skip pitch
	// flip); cycle is in shipping; should error on cycle-status guard.
	s := newStore(t)
	writePitchDirect(t, s, model.Pitch{Slug: "csv", Status: model.PitchStatusShipped, Body: shapedBody})
	writeCycleDirect(t, s, model.Cycle{
		ID: "w22", Duration: "5d", Status: model.CycleStatusShipping,
		Bets: []model.Bet{{Pitch: "csv", Appetite: model.AppetiteMedium}},
	})
	writeCardDirect(t, s, model.Card{Slug: "c", Pitch: "csv", Cycle: "w22", Hill: model.HillDownhill, Progress: 100, HillUpdatedAt: &fixedNow})
	_, err := Hill(s, HillRequest{Card: "c", Progress: ptr(100), Done: true})
	if !errors.Is(err, ErrInvalidTransition) {
		t.Errorf("want ErrInvalidTransition, got %v", err)
	}
}
