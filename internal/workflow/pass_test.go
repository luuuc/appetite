package workflow

import (
	"context"
	"errors"
	"testing"

	"github.com/luuuc/appetite/internal/model"
	"github.com/luuuc/appetite/internal/store"
)

func TestPass_HappyPath_WithCycle(t *testing.T) {
	s := newStore(t)
	writePitchDirect(t, s, model.Pitch{Slug: "dark", Status: model.PitchStatusShaped, Body: shapedBody})
	writeCycleDirect(t, s, model.Cycle{ID: "w22", Duration: "5d", Status: model.CycleStatusBuilding})

	res, err := Pass(s, PassRequest{Pitch: "dark", Reason: "not worth it", Now: fixedNow})
	if err != nil {
		t.Fatalf("Pass: %v", err)
	}
	if res.CyclePath == "" {
		t.Error("expected cycle path")
	}
	p, _ := store.ReadAs[model.Pitch](context.Background(), s, "pitches/dark.md")
	if p.Status != model.PitchStatusPassed {
		t.Errorf("pitch status: got %q", p.Status)
	}
	c, _ := store.ReadAs[model.Cycle](context.Background(), s, "cycles/w22/cycle.yml")
	if len(c.Passed) != 1 || c.Passed[0].Reason != "not worth it" {
		t.Errorf("passed: %v", c.Passed)
	}
}

func TestPass_NoActiveCycle(t *testing.T) {
	s := newStore(t)
	writePitchDirect(t, s, model.Pitch{Slug: "dark", Status: model.PitchStatusShaped, Body: shapedBody})

	res, err := Pass(s, PassRequest{Pitch: "dark", Reason: "x", Now: fixedNow})
	if err != nil {
		t.Fatalf("Pass: %v", err)
	}
	if res.CyclePath != "" {
		t.Errorf("expected empty cycle path, got %q", res.CyclePath)
	}
}

func TestPass_SkipsNonBuildingActiveCycle(t *testing.T) {
	s := newStore(t)
	writePitchDirect(t, s, model.Pitch{Slug: "dark", Status: model.PitchStatusShaped, Body: shapedBody})
	writeCycleDirect(t, s, model.Cycle{ID: "w22", Duration: "5d", Status: model.CycleStatusShipping})

	res, err := Pass(s, PassRequest{Pitch: "dark", Reason: "x", Now: fixedNow})
	if err != nil {
		t.Fatalf("Pass: %v", err)
	}
	if res.CyclePath != "" {
		t.Errorf("shipping cycle: expected no record, got %q", res.CyclePath)
	}
}

func TestPass_ExplicitCycleMustBeBuilding(t *testing.T) {
	s := newStore(t)
	writePitchDirect(t, s, model.Pitch{Slug: "dark", Status: model.PitchStatusShaped, Body: shapedBody})
	writeCycleDirect(t, s, model.Cycle{ID: "w22", Duration: "5d", Status: model.CycleStatusShipping})

	_, err := Pass(s, PassRequest{Pitch: "dark", Reason: "x", Cycle: "w22"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, ErrInvalidTransition) {
		t.Errorf("want ErrInvalidTransition, got %v", err)
	}
}

func TestPass_Errors(t *testing.T) {
	tests := []struct {
		name    string
		seed    func(t *testing.T) store.Store
		req     PassRequest
		wantErr error
	}{
		{
			name: "missing pitch arg",
			seed: func(t *testing.T) store.Store { return newStore(t) },
			req:  PassRequest{Reason: "x"},
		},
		{
			name: "missing reason",
			seed: func(t *testing.T) store.Store { return newStore(t) },
			req:  PassRequest{Pitch: "x"},
		},
		{
			name:    "missing pitch entity",
			seed:    func(t *testing.T) store.Store { return newStore(t) },
			req:     PassRequest{Pitch: "nope", Reason: "x"},
			wantErr: ErrNotFound,
		},
		{
			name: "already bet pitch",
			seed: func(t *testing.T) store.Store {
				s := newStore(t)
				writePitchDirect(t, s, model.Pitch{Slug: "p", Status: model.PitchStatusBet})
				return s
			},
			req:     PassRequest{Pitch: "p", Reason: "x"},
			wantErr: ErrInvalidTransition,
		},
		{
			name: "explicit missing cycle",
			seed: func(t *testing.T) store.Store {
				s := newStore(t)
				writePitchDirect(t, s, model.Pitch{Slug: "p", Status: model.PitchStatusShaped, Body: shapedBody})
				return s
			},
			req:     PassRequest{Pitch: "p", Reason: "x", Cycle: "nope"},
			wantErr: ErrNotFound,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Pass(tc.seed(t), tc.req)
			if err == nil {
				t.Fatal("expected error")
			}
			if tc.wantErr != nil && !errors.Is(err, tc.wantErr) {
				t.Errorf("error: got %v want errors.Is %v", err, tc.wantErr)
			}
		})
	}
}
