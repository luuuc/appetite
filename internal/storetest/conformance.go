// Package storetest holds the conformance suite every store.Store
// implementation must pass. It lives in its own package rather than
// under internal/store so production binaries that depend on the Store
// contract don't also pull in the testing package.
package storetest

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/luuuc/appetite/internal/model"
	"github.com/luuuc/appetite/internal/store"
)

// TestStore runs the conformance suite against any store.Store
// implementation. newStore must return a fresh, empty store for each
// call — subtests rely on isolation.
func TestStore(t *testing.T, newStore func(t *testing.T) store.Store) {
	t.Helper()

	ctx := context.Background()

	t.Run("signal_roundtrip", func(t *testing.T) {
		s := newStore(t)
		sig := newSignal("checkout-timeout")

		path, err := s.Write(ctx, sig)
		if err != nil {
			t.Fatalf("Write: %v", err)
		}
		if path != sig.Path() {
			t.Errorf("Write returned path %q, want %q", path, sig.Path())
		}

		got, err := store.ReadAs[model.Signal](ctx, s, path)
		if err != nil {
			t.Fatalf("ReadAs: %v", err)
		}
		if got.Source != sig.Source {
			t.Errorf("Source = %q, want %q", got.Source, sig.Source)
		}
		if !got.Captured.Equal(sig.Captured) {
			t.Errorf("Captured = %v, want %v", got.Captured, sig.Captured)
		}
		if got.Body != sig.Body {
			t.Errorf("Body = %q, want %q", got.Body, sig.Body)
		}
	})

	t.Run("pitch_roundtrip", func(t *testing.T) {
		s := newStore(t)
		p := newPitch("csv-export")

		path, err := s.Write(ctx, p)
		if err != nil {
			t.Fatalf("Write: %v", err)
		}

		got, err := store.ReadAs[model.Pitch](ctx, s, path)
		if err != nil {
			t.Fatalf("ReadAs: %v", err)
		}
		if got.Slug != p.Slug {
			t.Errorf("Slug = %q, want %q", got.Slug, p.Slug)
		}
		if got.Title != p.Title {
			t.Errorf("Title = %q, want %q", got.Title, p.Title)
		}
		if got.Appetite != p.Appetite {
			t.Errorf("Appetite = %q, want %q", got.Appetite, p.Appetite)
		}
		if got.Status != p.Status {
			t.Errorf("Status = %q, want %q", got.Status, p.Status)
		}
		if got.Body != p.Body {
			t.Errorf("Body = %q, want %q", got.Body, p.Body)
		}
	})

	t.Run("cycle_roundtrip", func(t *testing.T) {
		s := newStore(t)
		c := newCycle("2026-w15")

		path, err := s.Write(ctx, c)
		if err != nil {
			t.Fatalf("Write: %v", err)
		}

		got, err := store.ReadAs[model.Cycle](ctx, s, path)
		if err != nil {
			t.Fatalf("ReadAs: %v", err)
		}
		if got.ID != c.ID {
			t.Errorf("ID = %q, want %q", got.ID, c.ID)
		}
		if got.Status != c.Status {
			t.Errorf("Status = %q, want %q", got.Status, c.Status)
		}
		if len(got.Bets) != len(c.Bets) {
			t.Fatalf("Bets length = %d, want %d", len(got.Bets), len(c.Bets))
		}
		if got.Bets[0].Pitch != c.Bets[0].Pitch {
			t.Errorf("Bets[0].Pitch = %q, want %q", got.Bets[0].Pitch, c.Bets[0].Pitch)
		}
	})

	t.Run("card_roundtrip", func(t *testing.T) {
		s := newStore(t)
		c := newCard("2026-w15", "export-button")

		path, err := s.Write(ctx, c)
		if err != nil {
			t.Fatalf("Write: %v", err)
		}

		got, err := store.ReadAs[model.Card](ctx, s, path)
		if err != nil {
			t.Fatalf("ReadAs: %v", err)
		}
		if got.Hill != c.Hill {
			t.Errorf("Hill = %q, want %q", got.Hill, c.Hill)
		}
		if got.Progress != c.Progress {
			t.Errorf("Progress = %d, want %d", got.Progress, c.Progress)
		}
		if len(got.DoneLooksLike) != len(c.DoneLooksLike) {
			t.Errorf("DoneLooksLike length = %d, want %d", len(got.DoneLooksLike), len(c.DoneLooksLike))
		}
	})

	t.Run("cooldown_roundtrip", func(t *testing.T) {
		s := newStore(t)
		c := newCooldown("2026-w15")

		path, err := s.Write(ctx, c)
		if err != nil {
			t.Fatalf("Write: %v", err)
		}

		got, err := store.ReadAs[model.Cooldown](ctx, s, path)
		if err != nil {
			t.Fatalf("ReadAs: %v", err)
		}
		if got.Status != c.Status {
			t.Errorf("Status = %q, want %q", got.Status, c.Status)
		}
		if got.Notes != c.Notes {
			t.Errorf("Notes = %q, want %q", got.Notes, c.Notes)
		}
	})

	t.Run("update_existing", func(t *testing.T) {
		s := newStore(t)
		p := newPitch("csv-export")

		path, err := s.Write(ctx, p)
		if err != nil {
			t.Fatalf("Write (create): %v", err)
		}

		// Mutate and write again at the same path.
		p.Title = "CSV Export v2"
		path2, err := s.Write(ctx, p)
		if err != nil {
			t.Fatalf("Write (update): %v", err)
		}
		if path2 != path {
			t.Errorf("update returned different path: %q vs %q", path2, path)
		}

		got, err := store.ReadAs[model.Pitch](ctx, s, path)
		if err != nil {
			t.Fatalf("Read after update: %v", err)
		}
		if got.Title != "CSV Export v2" {
			t.Errorf("Title after update = %q, want %q", got.Title, "CSV Export v2")
		}
	})

	t.Run("list_empty_store", func(t *testing.T) {
		s := newStore(t)
		got, err := s.List(ctx, store.Filter{})
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if got == nil {
			t.Fatal("List returned nil, want empty slice")
		}
		if len(got) != 0 {
			t.Errorf("List returned %d entities, want 0", len(got))
		}
	})

	t.Run("list_all", func(t *testing.T) {
		s := newStore(t)
		seed(t, ctx, s)

		got, err := s.List(ctx, store.Filter{})
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		// 2 signals + 2 pitches + 1 cycle + 2 cards + 1 cooldown = 8
		if len(got) != 8 {
			t.Errorf("List(no filter) returned %d entities, want 8", len(got))
		}
	})

	t.Run("list_filter_by_kind", func(t *testing.T) {
		s := newStore(t)
		seed(t, ctx, s)

		kind := model.KindPitch
		got, err := s.List(ctx, store.Filter{Kind: &kind})
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if len(got) != 2 {
			t.Fatalf("List(kind=pitch) = %d, want 2", len(got))
		}
		for _, e := range got {
			if e.Kind() != model.KindPitch {
				t.Errorf("got kind %q, want %q", e.Kind(), model.KindPitch)
			}
		}
	})

	t.Run("list_filter_by_cycle", func(t *testing.T) {
		s := newStore(t)
		seed(t, ctx, s)

		id := "2026-w15"
		got, err := s.List(ctx, store.Filter{Cycle: &id})
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		// Cycle 2026-w15: 1 cycle.yml + 2 cards + 1 cooldown = 4
		if len(got) != 4 {
			t.Errorf("List(cycle=%q) returned %d, want 4", id, len(got))
		}
	})

	t.Run("list_filter_by_tags", func(t *testing.T) {
		s := newStore(t)
		seed(t, ctx, s)

		got, err := s.List(ctx, store.Filter{Tags: []string{"checkout"}})
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if len(got) != 1 {
			t.Fatalf("List(tags=checkout) = %d, want 1", len(got))
		}
		sig, ok := got[0].(model.Signal)
		if !ok {
			t.Fatalf("got %T, want model.Signal", got[0])
		}
		if sig.Slug != "checkout-timeout" {
			t.Errorf("Slug = %q, want checkout-timeout", sig.Slug)
		}
	})

	t.Run("list_filter_kind_and_cycle", func(t *testing.T) {
		s := newStore(t)
		seed(t, ctx, s)

		kind := model.KindCard
		id := "2026-w15"
		got, err := s.List(ctx, store.Filter{Kind: &kind, Cycle: &id})
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if len(got) != 2 {
			t.Errorf("List(kind=card, cycle=%q) = %d, want 2", id, len(got))
		}
	})

	t.Run("list_filter_kind_and_tags", func(t *testing.T) {
		s := newStore(t)
		seed(t, ctx, s)

		kind := model.KindSignal
		got, err := s.List(ctx, store.Filter{Kind: &kind, Tags: []string{"performance"}})
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if len(got) != 1 {
			t.Fatalf("List(kind=signal, tags=performance) = %d, want 1", len(got))
		}
		sig, ok := got[0].(model.Signal)
		if !ok {
			t.Fatalf("got %T, want model.Signal", got[0])
		}
		if sig.Slug != "dashboard-slow" {
			t.Errorf("Slug = %q, want dashboard-slow", sig.Slug)
		}
	})

	t.Run("list_filter_nonmatching_cycle", func(t *testing.T) {
		s := newStore(t)
		seed(t, ctx, s)

		id := "2026-w99"
		got, err := s.List(ctx, store.Filter{Cycle: &id})
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if len(got) != 0 {
			t.Errorf("List(cycle=%q) = %d, want 0 — adapter must not ignore Cycle filter", id, len(got))
		}
	})

	t.Run("archived_signal_roundtrip", func(t *testing.T) {
		s := newStore(t)
		sig := newSignal("moved-signal")
		sig.Archived = true

		path, err := s.Write(ctx, sig)
		if err != nil {
			t.Fatalf("Write: %v", err)
		}
		if path != "signals/archived/moved-signal.md" {
			t.Errorf("archived signal path = %q, want signals/archived/moved-signal.md", path)
		}

		// Archived is a path-derived field (not in YAML); the adapter must
		// set it on read based on which subdirectory the file lives under.
		got, err := store.ReadAs[model.Signal](ctx, s, path)
		if err != nil {
			t.Fatalf("ReadAs: %v", err)
		}
		if !got.Archived {
			t.Error("Archived = false after roundtrip, want true — adapter must reconstruct Archived from path")
		}
	})

	t.Run("delete", func(t *testing.T) {
		s := newStore(t)
		p := newPitch("csv-export")

		path, err := s.Write(ctx, p)
		if err != nil {
			t.Fatalf("Write: %v", err)
		}

		if err := s.Delete(ctx, path); err != nil {
			t.Fatalf("Delete: %v", err)
		}

		if _, err := s.Read(ctx, path); !errors.Is(err, store.ErrNotFound) {
			t.Errorf("Read after Delete: got %v, want ErrNotFound", err)
		}
	})

	t.Run("read_nonexistent", func(t *testing.T) {
		s := newStore(t)

		_, err := s.Read(ctx, "pitches/does-not-exist.md")
		if !errors.Is(err, store.ErrNotFound) {
			t.Errorf("Read nonexistent: got %v, want ErrNotFound", err)
		}
	})

	t.Run("delete_nonexistent", func(t *testing.T) {
		s := newStore(t)

		err := s.Delete(ctx, "pitches/does-not-exist.md")
		if !errors.Is(err, store.ErrNotFound) {
			t.Errorf("Delete nonexistent: got %v, want ErrNotFound", err)
		}
	})

	t.Run("readas_wrong_type", func(t *testing.T) {
		s := newStore(t)
		p := newPitch("csv-export")

		path, err := s.Write(ctx, p)
		if err != nil {
			t.Fatalf("Write: %v", err)
		}

		_, err = store.ReadAs[model.Signal](ctx, s, path)
		if err == nil {
			t.Error("ReadAs[Signal] on pitch path: expected error, got nil")
		}
	})
}

// --- Test entity factories ------------------------------------------------

func newSignal(slug string) model.Signal {
	return model.Signal{
		Slug:     slug,
		Source:   model.SignalSourceBeacon,
		Captured: time.Date(2026, 4, 17, 0, 0, 0, 0, time.UTC),
		Tags:     []string{"checkout", "error"},
		Body:     "Three new PaymentGateway::TimeoutError fingerprints since the 04-16 deploy.\n",
	}
}

func newPitch(slug string) model.Pitch {
	shaped := time.Date(2026, 4, 17, 0, 0, 0, 0, time.UTC)
	return model.Pitch{
		Slug:       slug,
		Title:      "CSV Export",
		Appetite:   model.AppetiteMedium,
		Status:     model.PitchStatusShaped,
		ShapedAt:   &shaped,
		ShapedFrom: []string{"signals/raw/export-request.md"},
		Body:       "## Problem\n\nCustomers want CSV exports.\n",
	}
}

func newCycle(id string) model.Cycle {
	return model.Cycle{
		ID:       id,
		Duration: "5d",
		Started:  time.Date(2026, 4, 13, 0, 0, 0, 0, time.UTC),
		Ends:     time.Date(2026, 4, 17, 0, 0, 0, 0, time.UTC),
		Status:   model.CycleStatusBuilding,
		Bets: []model.Bet{
			{Pitch: "csv-export", Appetite: model.AppetiteMedium, BetAt: time.Date(2026, 4, 13, 0, 0, 0, 0, time.UTC)},
		},
		Passed: []model.Passed{
			{Pitch: "dark-mode", Reason: "not worth it this cycle", PassedAt: time.Date(2026, 4, 13, 0, 0, 0, 0, time.UTC)},
		},
	}
}

func newCard(cycle, slug string) model.Card {
	return model.Card{
		Slug:     slug,
		Pitch:    "csv-export",
		Cycle:    cycle,
		Hill:     model.HillDownhill,
		Progress: 70,
		Assigned: "claude",
		DoneLooksLike: []string{
			"Export button visible on all list views",
			"CSV downloads in under 2 seconds",
		},
		Body: "Button component, wired to the export endpoint.\n",
	}
}

func newCooldown(cycle string) model.Cooldown {
	return model.Cooldown{
		Cycle:   cycle,
		Started: time.Date(2026, 4, 17, 0, 0, 0, 0, time.UTC),
		Ends:    time.Date(2026, 4, 19, 0, 0, 0, 0, time.UTC),
		Status:  model.CooldownActive,
		Notes:   "Polish pass on CSV export empty states.\n",
	}
}

// seed populates a store with a known mix of entities spanning every
// kind so List filters have something to slice. Totals: 2 signals,
// 2 pitches, 1 cycle (2026-w15), 2 cards in that cycle, 1 cooldown for
// that cycle = 8 entities.
func seed(t *testing.T, ctx context.Context, s store.Store) {
	t.Helper()

	entities := []model.Entity{
		newSignal("checkout-timeout"),
		model.Signal{
			Slug:     "dashboard-slow",
			Source:   model.SignalSourceOperator,
			Captured: time.Date(2026, 4, 15, 0, 0, 0, 0, time.UTC),
			Tags:     []string{"dashboard", "performance"},
			Body:     "Dashboard page takes 4s to render.\n",
		},
		newPitch("csv-export"),
		model.Pitch{
			Slug:     "dark-mode",
			Title:    "Dark Mode",
			Appetite: model.AppetiteSmall,
			Status:   model.PitchStatusShaped,
			Body:     "## Problem\n\nUsers want dark mode.\n",
		},
		newCycle("2026-w15"),
		newCard("2026-w15", "export-button"),
		model.Card{
			Slug:     "csv-format",
			Pitch:    "csv-export",
			Cycle:    "2026-w15",
			Hill:     model.HillUphill,
			Progress: 20,
			Body:     "Flat-table CSV format with ISO 8601 dates.\n",
		},
		newCooldown("2026-w15"),
	}

	for _, e := range entities {
		if _, err := s.Write(ctx, e); err != nil {
			t.Fatalf("seed: write %T %q: %v", e, e.Path(), err)
		}
	}
}
