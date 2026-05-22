package workflow

import (
	"context"
	"errors"
	"testing"

	"github.com/luuuc/appetite/internal/model"
	"github.com/luuuc/appetite/internal/store"
)

func setupForHill(t *testing.T) store.Store {
	t.Helper()
	s := newStore(t)
	writePitchDirect(t, s, model.Pitch{
		Slug: "csv", Title: "CSV", Status: model.PitchStatusBuilding, Body: shapedBody,
	})
	writeCycleDirect(t, s, model.Cycle{
		ID: "w22", Duration: "5d", Status: model.CycleStatusBuilding,
		Started: fixedNow, Ends: fixedNow,
		Bets: []model.Bet{{Pitch: "csv", Appetite: model.AppetiteMedium}},
	})
	writeCardDirect(t, s, model.Card{
		Slug: "a", Pitch: "csv", Cycle: "w22", Hill: model.HillUphill, Progress: 0, HillUpdatedAt: &fixedNow,
	})
	writeCardDirect(t, s, model.Card{
		Slug: "b", Pitch: "csv", Cycle: "w22", Hill: model.HillUphill, Progress: 0, HillUpdatedAt: &fixedNow,
	})
	return s
}

func ptr(i int) *int { return &i }

func TestHill_PositionOnly(t *testing.T) {
	s := setupForHill(t)
	res, err := Hill(s, HillRequest{Card: "a", Position: model.HillDownhill, Now: fixedNow})
	if err != nil {
		t.Fatalf("Hill: %v", err)
	}
	if res.Card.Hill != model.HillDownhill {
		t.Errorf("hill: got %q", res.Card.Hill)
	}
	if res.Card.Progress != 0 {
		t.Errorf("progress: got %d want unchanged", res.Card.Progress)
	}
}

func TestHill_ProgressOnly(t *testing.T) {
	s := setupForHill(t)
	res, err := Hill(s, HillRequest{Card: "a", Progress: ptr(70), Now: fixedNow})
	if err != nil {
		t.Fatalf("Hill: %v", err)
	}
	if res.Card.Progress != 70 {
		t.Errorf("progress: got %d", res.Card.Progress)
	}
}

func TestHill_ShipCascades(t *testing.T) {
	s := setupForHill(t)
	// Ship card a — pitch shouldn't ship yet (b is still uphill 0%).
	res, err := Hill(s, HillRequest{Card: "a", Progress: ptr(100), Done: true, Now: fixedNow})
	if err != nil {
		t.Fatalf("Hill a: %v", err)
	}
	if res.PitchShipped != nil || res.CycleShipped != nil {
		t.Errorf("premature cascade: %+v", res)
	}
	// Ship card b — pitch + cycle should now cascade.
	res, err = Hill(s, HillRequest{Card: "b", Progress: ptr(100), Done: true, Now: fixedNow})
	if err != nil {
		t.Fatalf("Hill b: %v", err)
	}
	if res.PitchShipped == nil || res.PitchShipped.Status != model.PitchStatusShipped {
		t.Errorf("pitch cascade: %+v", res.PitchShipped)
	}
	if res.CycleShipped == nil || res.CycleShipped.Status != model.CycleStatusShipping {
		t.Errorf("cycle cascade: %+v", res.CycleShipped)
	}

	// Verify persistence.
	p, _ := store.ReadAs[model.Pitch](context.Background(), s, "pitches/csv.md")
	if p.Status != model.PitchStatusShipped {
		t.Errorf("persisted pitch: %q", p.Status)
	}
}

func TestHill_ValidationErrors(t *testing.T) {
	tests := []struct {
		name    string
		req     HillRequest
		wantErr error
	}{
		{"missing card", HillRequest{}, nil},
		{"bad position", HillRequest{Card: "a", Position: model.HillPosition("sideways")}, nil},
		{"progress below 0", HillRequest{Card: "a", Progress: ptr(-1)}, nil},
		{"progress above 100", HillRequest{Card: "a", Progress: ptr(150)}, nil},
		{"100 without done", HillRequest{Card: "a", Progress: ptr(100)}, ErrDoneCriteriaUnmet},
		{"done with progress != 100", HillRequest{Card: "a", Progress: ptr(70), Done: true}, nil},
		{"done without progress", HillRequest{Card: "a", Done: true}, nil},
		{"nothing to do", HillRequest{Card: "a"}, nil},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := setupForHill(t)
			_, err := Hill(s, tc.req)
			if err == nil {
				t.Fatal("expected error")
			}
			if tc.wantErr != nil && !errors.Is(err, tc.wantErr) {
				t.Errorf("error: got %v want errors.Is %v", err, tc.wantErr)
			}
		})
	}
}

func TestHill_CardNotFound(t *testing.T) {
	s := setupForHill(t)
	_, err := Hill(s, HillRequest{Card: "nope", Position: model.HillDownhill})
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("want ErrNotFound, got %v", err)
	}
}

func TestHill_CardInMultipleCycles(t *testing.T) {
	s := newStore(t)
	writeCardDirect(t, s, model.Card{Slug: "dup", Pitch: "p", Cycle: "a", Hill: model.HillUphill})
	writeCardDirect(t, s, model.Card{Slug: "dup", Pitch: "p", Cycle: "b", Hill: model.HillUphill})
	_, err := Hill(s, HillRequest{Card: "dup", Position: model.HillDownhill})
	if err == nil {
		t.Fatal("expected disambiguation error")
	}
}

func TestHill_NonBuildingCycle(t *testing.T) {
	s := setupForHill(t)
	// Move cycle to shipping.
	c, _ := store.ReadAs[model.Cycle](context.Background(), s, "cycles/w22/cycle.yml")
	c.Status = model.CycleStatusShipping
	writeCycleDirect(t, s, c)

	_, err := Hill(s, HillRequest{Card: "a", Position: model.HillDownhill})
	if !errors.Is(err, ErrInvalidTransition) {
		t.Errorf("want ErrInvalidTransition, got %v", err)
	}
}

func TestHill_ExplicitCycle(t *testing.T) {
	s := setupForHill(t)
	res, err := Hill(s, HillRequest{Card: "a", Cycle: "w22", Position: model.HillDownhill})
	if err != nil {
		t.Fatalf("Hill: %v", err)
	}
	if res.Card.Slug != "a" {
		t.Errorf("card: got %q", res.Card.Slug)
	}
}

func TestPitchCardsAllDone_NoCards(t *testing.T) {
	s := newStore(t)
	done, err := pitchCardsAllDone(s, "nobody", "no-cycle")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if done {
		t.Error("no cards → should not report done")
	}
}

func TestCycleBetsAllShipped_NoBets(t *testing.T) {
	s := newStore(t)
	done, err := cycleBetsAllShipped(s, model.Cycle{ID: "x"})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if done {
		t.Error("no bets → should not report shipped")
	}
}
