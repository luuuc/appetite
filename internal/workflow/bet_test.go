package workflow

import (
	"context"
	"errors"
	"testing"

	"github.com/luuuc/appetite/internal/model"
	"github.com/luuuc/appetite/internal/store"
)

func setupForBet(t *testing.T) store.Store {
	t.Helper()
	s := newStore(t)
	writePitchDirect(t, s, model.Pitch{
		Slug: "csv", Title: "CSV", Status: model.PitchStatusShaped, Body: shapedBody,
	})
	writeCycleDirect(t, s, model.Cycle{
		ID: "w22", Duration: "5d", Status: model.CycleStatusBuilding,
		Started: fixedNow, Ends: fixedNow,
	})
	return s
}

func TestBet_HappyPath(t *testing.T) {
	s := setupForBet(t)
	res, err := Bet(s, BetRequest{
		Pitch: "csv", Cycle: "w22", Appetite: model.AppetiteMedium, Now: fixedNow,
	})
	if err != nil {
		t.Fatalf("Bet: %v", err)
	}
	if res.PitchPath == "" || res.CyclePath == "" {
		t.Errorf("paths: %+v", res)
	}
	// Verify pitch flipped and cycle bets recorded.
	p, _ := store.ReadAs[model.Pitch](context.Background(), s, "pitches/csv.md")
	if p.Status != model.PitchStatusBet {
		t.Errorf("pitch status: got %q", p.Status)
	}
	c, _ := store.ReadAs[model.Cycle](context.Background(), s, "cycles/w22/cycle.yml")
	if len(c.Bets) != 1 || c.Bets[0].Pitch != "csv" {
		t.Errorf("bets: %v", c.Bets)
	}
}

func TestBet_Errors(t *testing.T) {
	tests := []struct {
		name    string
		seed    func(t *testing.T) store.Store
		req     BetRequest
		wantErr error
	}{
		{
			name: "missing pitch arg", seed: setupForBet,
			req: BetRequest{Cycle: "w22", Appetite: model.AppetiteMedium},
		},
		{
			name: "missing cycle arg", seed: setupForBet,
			req: BetRequest{Pitch: "csv", Appetite: model.AppetiteMedium},
		},
		{
			name: "bad appetite", seed: setupForBet,
			req: BetRequest{Pitch: "csv", Cycle: "w22", Appetite: model.Appetite("huge")},
		},
		{
			name:    "missing pitch entity",
			seed:    setupForBet,
			req:     BetRequest{Pitch: "nope", Cycle: "w22", Appetite: model.AppetiteMedium},
			wantErr: ErrNotFound,
		},
		{
			name: "wrong pitch status",
			seed: func(t *testing.T) store.Store {
				s := newStore(t)
				writePitchDirect(t, s, model.Pitch{Slug: "csv", Status: model.PitchStatusShaping})
				writeCycleDirect(t, s, model.Cycle{ID: "w22", Duration: "5d", Status: model.CycleStatusBuilding})
				return s
			},
			req:     BetRequest{Pitch: "csv", Cycle: "w22", Appetite: model.AppetiteMedium},
			wantErr: ErrInvalidTransition,
		},
		{
			name:    "missing cycle entity",
			seed:    setupForBet,
			req:     BetRequest{Pitch: "csv", Cycle: "nope", Appetite: model.AppetiteMedium},
			wantErr: ErrNotFound,
		},
		{
			name: "cycle not building",
			seed: func(t *testing.T) store.Store {
				s := newStore(t)
				writePitchDirect(t, s, model.Pitch{Slug: "csv", Status: model.PitchStatusShaped, Body: shapedBody})
				writeCycleDirect(t, s, model.Cycle{ID: "w22", Duration: "5d", Status: model.CycleStatusShipping})
				return s
			},
			req:     BetRequest{Pitch: "csv", Cycle: "w22", Appetite: model.AppetiteMedium},
			wantErr: ErrInvalidTransition,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := tc.seed(t)
			_, err := Bet(s, tc.req)
			if err == nil {
				t.Fatal("expected error")
			}
			if tc.wantErr != nil && !errors.Is(err, tc.wantErr) {
				t.Errorf("error: got %v want errors.Is %v", err, tc.wantErr)
			}
		})
	}
}
