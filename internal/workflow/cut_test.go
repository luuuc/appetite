package workflow

import (
	"context"
	"errors"
	"testing"

	"github.com/luuuc/appetite/internal/model"
	"github.com/luuuc/appetite/internal/store"
)

func setupForCut(t *testing.T, scopeBody string) store.Store {
	t.Helper()
	body := shapedBody
	if scopeBody != "" {
		body = scopeBody
	}
	s := newStore(t)
	writePitchDirect(t, s, model.Pitch{
		Slug: "csv", Title: "CSV", Status: model.PitchStatusBet, Body: body,
	})
	writeCycleDirect(t, s, model.Cycle{
		ID: "w22", Duration: "5d", Status: model.CycleStatusBuilding,
		Bets: []model.Bet{{Pitch: "csv", Appetite: model.AppetiteMedium, BetAt: fixedNow}},
	})
	return s
}

func TestCut_HappyPath(t *testing.T) {
	s := setupForCut(t, "")
	res, err := Cut(s, CutRequest{Pitch: "csv", Now: fixedNow})
	if err != nil {
		t.Fatalf("Cut: %v", err)
	}
	if len(res.Cards) != 2 {
		t.Errorf("cards: got %d want 2", len(res.Cards))
	}
	if res.Cycle != "w22" {
		t.Errorf("cycle: got %q", res.Cycle)
	}
	if res.Pitch.Status != model.PitchStatusBuilding {
		t.Errorf("pitch status: got %q", res.Pitch.Status)
	}
	for _, c := range res.Cards {
		if c.Hill != model.HillUphill || c.Progress != 0 {
			t.Errorf("card defaults: %+v", c)
		}
		if c.HillUpdatedAt == nil {
			t.Errorf("card %s missing HillUpdatedAt", c.Slug)
		}
	}
}

func TestCut_ExplicitCycle(t *testing.T) {
	s := setupForCut(t, "")
	// Add a second cycle that doesn't have the bet — explicit --cycle wins
	writeCycleDirect(t, s, model.Cycle{ID: "w23", Duration: "5d", Status: model.CycleStatusClosed})
	res, err := Cut(s, CutRequest{Pitch: "csv", Cycle: "w22"})
	if err != nil {
		t.Fatalf("Cut: %v", err)
	}
	if res.Cycle != "w22" {
		t.Errorf("explicit cycle: got %q", res.Cycle)
	}
}

func TestCut_Errors(t *testing.T) {
	tests := []struct {
		name    string
		seed    func(t *testing.T) store.Store
		req     CutRequest
		wantErr error
	}{
		{
			name: "missing pitch arg",
			seed: func(t *testing.T) store.Store { return newStore(t) },
			req:  CutRequest{},
		},
		{
			name:    "missing pitch entity",
			seed:    func(t *testing.T) store.Store { return newStore(t) },
			req:     CutRequest{Pitch: "nope"},
			wantErr: ErrNotFound,
		},
		{
			name: "wrong pitch status",
			seed: func(t *testing.T) store.Store {
				s := newStore(t)
				writePitchDirect(t, s, model.Pitch{Slug: "p", Status: model.PitchStatusShaping, Body: shapedBody})
				return s
			},
			req:     CutRequest{Pitch: "p"},
			wantErr: ErrInvalidTransition,
		},
		{
			name: "no bet in any cycle",
			seed: func(t *testing.T) store.Store {
				s := newStore(t)
				writePitchDirect(t, s, model.Pitch{Slug: "p", Status: model.PitchStatusBet, Body: shapedBody})
				return s
			},
			req: CutRequest{Pitch: "p"},
		},
		{
			name: "bet in multiple cycles",
			seed: func(t *testing.T) store.Store {
				s := newStore(t)
				writePitchDirect(t, s, model.Pitch{Slug: "p", Status: model.PitchStatusBet, Body: shapedBody})
				writeCycleDirect(t, s, model.Cycle{ID: "a", Duration: "5d", Status: model.CycleStatusBuilding,
					Bets: []model.Bet{{Pitch: "p", Appetite: model.AppetiteMedium}}})
				writeCycleDirect(t, s, model.Cycle{ID: "b", Duration: "5d", Status: model.CycleStatusClosed,
					Bets: []model.Bet{{Pitch: "p", Appetite: model.AppetiteSmall}}})
				return s
			},
			req: CutRequest{Pitch: "p"},
		},
		{
			name: "explicit missing cycle",
			seed: func(t *testing.T) store.Store {
				return setupForCut(t, "")
			},
			req:     CutRequest{Pitch: "csv", Cycle: "nope"},
			wantErr: ErrNotFound,
		},
		{
			name: "malformed scope line",
			seed: func(t *testing.T) store.Store {
				return setupForCut(t, `## Problem
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
- [ ] **A** - hyphen
`)
			},
			req:     CutRequest{Pitch: "csv"},
			wantErr: ErrInvalidTransition,
		},
		{
			name: "no scope cards",
			seed: func(t *testing.T) store.Store {
				return setupForCut(t, `## Problem
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
`)
			},
			req:     CutRequest{Pitch: "csv"},
			wantErr: ErrInvalidTransition,
		},
		{
			name: "slug collision",
			seed: func(t *testing.T) store.Store {
				return setupForCut(t, `## Problem
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
- [ ] **Foo bar** — one
- [ ] **Foo Bar** — two
`)
			},
			req: CutRequest{Pitch: "csv"},
		},
		{
			name: "all-punctuation title",
			seed: func(t *testing.T) store.Store {
				return setupForCut(t, `## Problem
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
- [ ] **!!!** — punctuation only title
`)
			},
			req: CutRequest{Pitch: "csv"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := tc.seed(t)
			_, err := Cut(s, tc.req)
			if err == nil {
				t.Fatal("expected error")
			}
			if tc.wantErr != nil && !errors.Is(err, tc.wantErr) {
				t.Errorf("error: got %v want errors.Is %v", err, tc.wantErr)
			}
		})
	}
}

func TestCut_PreExistingCard(t *testing.T) {
	s := setupForCut(t, "")
	// Pre-write a card with one of the names we'd generate
	writeCardDirect(t, s, model.Card{Slug: "card-one", Pitch: "csv", Cycle: "w22", Hill: model.HillUphill})
	if _, err := Cut(s, CutRequest{Pitch: "csv"}); err == nil {
		t.Fatal("expected error on pre-existing card")
	}
}

func TestRollbackCards_SwallowsErrors(t *testing.T) {
	s := newStore(t)
	// Best-effort delete of non-existent paths shouldn't panic.
	rollbackCards(s, []string{"cycles/x/cards/nope.md"})
}

func TestBuildCards_OK(t *testing.T) {
	cards, err := buildCards("p", "c", []ScopeCard{
		{Title: "Foo", Description: "Bar"},
	}, fixedNow)
	if err != nil {
		t.Fatalf("buildCards: %v", err)
	}
	if cards[0].Slug != "foo" {
		t.Errorf("slug: got %q", cards[0].Slug)
	}
	_ = context.Background()
}
