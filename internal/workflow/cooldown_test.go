package workflow

import (
	"context"
	"errors"
	"testing"

	"github.com/luuuc/appetite/internal/model"
	"github.com/luuuc/appetite/internal/store"
)

func setupForCooldown(t *testing.T) store.Store {
	t.Helper()
	s := newStore(t)
	writePitchDirect(t, s, model.Pitch{
		Slug: "csv", Title: "CSV", Status: model.PitchStatusShipped, Body: shapedBody,
	})
	writeCycleDirect(t, s, model.Cycle{
		ID: "w22", Duration: "5d", Status: model.CycleStatusShipping,
		Started: fixedNow, Ends: fixedNow,
		Bets: []model.Bet{{Pitch: "csv", Appetite: model.AppetiteMedium}},
	})
	return s
}

func TestCooldownOpen_HappyPath(t *testing.T) {
	s := setupForCooldown(t)
	res, err := CooldownOpen(s, CooldownOpenRequest{Now: fixedNow})
	if err != nil {
		t.Fatalf("CooldownOpen: %v", err)
	}
	if res.Cooldown.Status != model.CooldownActive {
		t.Errorf("cooldown status: %q", res.Cooldown.Status)
	}
	if res.Cycle.Status != model.CycleStatusCooldown {
		t.Errorf("cycle status: %q", res.Cycle.Status)
	}
}

func TestCooldownOpen_DefaultsTo1d(t *testing.T) {
	s := setupForCooldown(t)
	res, _ := CooldownOpen(s, CooldownOpenRequest{Now: fixedNow})
	wantDur := int64(24 * 3600)
	gotDur := int64(res.Cooldown.Ends.Sub(res.Cooldown.Started).Seconds())
	if gotDur != wantDur {
		t.Errorf("default duration: got %d want %d seconds", gotDur, wantDur)
	}
}

func TestCooldownOpen_RefusesWhileBuilding(t *testing.T) {
	s := newStore(t)
	writePitchDirect(t, s, model.Pitch{
		Slug: "csv", Status: model.PitchStatusBuilding, Body: shapedBody,
	})
	writeCycleDirect(t, s, model.Cycle{
		ID: "w22", Duration: "5d", Status: model.CycleStatusBuilding,
		Bets: []model.Bet{{Pitch: "csv", Appetite: model.AppetiteMedium}},
	})
	_, err := CooldownOpen(s, CooldownOpenRequest{Now: fixedNow})
	if !errors.Is(err, ErrInvalidTransition) {
		t.Errorf("want ErrInvalidTransition, got %v", err)
	}
}

func TestCooldownOpen_AllPitchesPassed(t *testing.T) {
	s := newStore(t)
	writePitchDirect(t, s, model.Pitch{Slug: "x", Status: model.PitchStatusPassed, Body: shapedBody})
	writeCycleDirect(t, s, model.Cycle{
		ID: "w22", Duration: "5d", Status: model.CycleStatusBuilding,
		Bets: []model.Bet{{Pitch: "x", Appetite: model.AppetiteSmall}},
	})
	// State machine prevents building → cooldown directly; expect ErrInvalidTransition.
	_, err := CooldownOpen(s, CooldownOpenRequest{Now: fixedNow})
	if !errors.Is(err, ErrInvalidTransition) {
		t.Errorf("want ErrInvalidTransition, got %v", err)
	}
}

func TestCooldownOpen_Errors(t *testing.T) {
	tests := []struct {
		name string
		seed func(t *testing.T) store.Store
		req  CooldownOpenRequest
	}{
		{
			name: "no active cycle",
			seed: func(t *testing.T) store.Store { return newStore(t) },
			req:  CooldownOpenRequest{},
		},
		{
			name: "bad duration",
			seed: setupForCooldown,
			req:  CooldownOpenRequest{Duration: "5y"},
		},
		{
			name: "missing referenced pitch",
			seed: func(t *testing.T) store.Store {
				s := newStore(t)
				writeCycleDirect(t, s, model.Cycle{
					ID: "w22", Duration: "5d", Status: model.CycleStatusShipping,
					Bets: []model.Bet{{Pitch: "ghost"}},
				})
				return s
			},
			req: CooldownOpenRequest{},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := CooldownOpen(tc.seed(t), tc.req)
			if err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestCooldownClose_HappyPath(t *testing.T) {
	s := setupForCooldown(t)
	if _, err := CooldownOpen(s, CooldownOpenRequest{Now: fixedNow}); err != nil {
		t.Fatalf("Open: %v", err)
	}
	res, err := CooldownClose(s, CooldownCloseRequest{Now: fixedNow})
	if err != nil {
		t.Fatalf("Close: %v", err)
	}
	if res.Cycle.Status != model.CycleStatusClosed {
		t.Errorf("cycle status: %q", res.Cycle.Status)
	}
	if res.Cooldown.Status != model.CooldownClosed {
		t.Errorf("cooldown status: %q", res.Cooldown.Status)
	}
}

func TestCooldownClose_Errors(t *testing.T) {
	tests := []struct {
		name string
		seed func(t *testing.T) store.Store
		req  CooldownCloseRequest
	}{
		{
			name: "no active cycle",
			seed: func(t *testing.T) store.Store { return newStore(t) },
			req:  CooldownCloseRequest{},
		},
		{
			name: "no cooldown open",
			seed: setupForCooldown,
			req:  CooldownCloseRequest{},
		},
		{
			name: "cooldown already closed",
			seed: func(t *testing.T) store.Store {
				s := setupForCooldown(t)
				if _, err := CooldownOpen(s, CooldownOpenRequest{Now: fixedNow}); err != nil {
					t.Fatalf("Open: %v", err)
				}
				if _, err := CooldownClose(s, CooldownCloseRequest{Now: fixedNow}); err != nil {
					t.Fatalf("Close: %v", err)
				}
				return s
			},
			req: CooldownCloseRequest{},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := CooldownClose(tc.seed(t), tc.req)
			if err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestCooldownClose_ExplicitCycle(t *testing.T) {
	s := setupForCooldown(t)
	if _, err := CooldownOpen(s, CooldownOpenRequest{Cycle: "w22", Now: fixedNow}); err != nil {
		t.Fatalf("Open: %v", err)
	}
	res, err := CooldownClose(s, CooldownCloseRequest{Cycle: "w22", Now: fixedNow})
	if err != nil {
		t.Fatalf("Close: %v", err)
	}
	if res.Cycle.ID != "w22" {
		t.Errorf("cycle: %q", res.Cycle.ID)
	}
}

func TestResolveCycle_Errors(t *testing.T) {
	s := newStore(t)
	if _, err := resolveCycle(s, "missing"); !errors.Is(err, ErrNotFound) {
		t.Errorf("explicit missing: got %v", err)
	}
	if _, err := resolveCycle(s, ""); !errors.Is(err, ErrNotFound) {
		t.Errorf("no active: got %v", err)
	}
	writeCycleDirect(t, s, model.Cycle{ID: "ok", Duration: "5d", Status: model.CycleStatusBuilding})
	c, err := resolveCycle(s, "")
	if err != nil {
		t.Fatalf("implicit: %v", err)
	}
	if c.ID != "ok" {
		t.Errorf("got %q", c.ID)
	}
}

func TestCoalesceDuration(t *testing.T) {
	if got := coalesceDuration(""); got != "1d" {
		t.Errorf("empty: got %q", got)
	}
	if got := coalesceDuration("3d"); got != "3d" {
		t.Errorf("non-empty: got %q", got)
	}
	_ = context.Background()
}
