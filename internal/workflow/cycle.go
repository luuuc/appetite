package workflow

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/luuuc/appetite/internal/model"
	"github.com/luuuc/appetite/internal/store"
)

// CycleNewRequest opens a new cycle in `building` status.
type CycleNewRequest struct {
	ID       string // required — operator's choice (2026-w15, auth-refactor-cycle, …)
	Duration string // required — "5d", "2w" (calendar budget, not bet appetite)
	Now      time.Time
}

// CycleNewResult reports what was written.
type CycleNewResult struct {
	Cycle model.Cycle
	Path  string
}

// CycleNew creates a new cycle in `building` status. It refuses to
// open a second active cycle: only one cycle may be open at a time
// (active = status building, shipping, or cooldown — i.e. not
// closed). This invariant is what lets `bet`/`pass`/`cut` default to
// "the active cycle" without ambiguity.
func CycleNew(s store.Store, req CycleNewRequest) (CycleNewResult, error) {
	if req.ID == "" {
		return CycleNewResult{}, fmt.Errorf("cycle id is required")
	}
	dur, err := ParseDuration(req.Duration)
	if err != nil {
		return CycleNewResult{}, fmt.Errorf("cycle appetite: %w", err)
	}

	if existing, err := store.ReadAs[model.Cycle](context.Background(), s, model.Cycle{ID: req.ID}.Path()); err == nil {
		return CycleNewResult{}, fmt.Errorf("cycle %q already exists at status %q", existing.ID, existing.Status)
	}

	if active, _ := findActiveCycle(s); active != nil {
		return CycleNewResult{}, fmt.Errorf("%w: cycle %q is still %s; close it before opening a new cycle",
			ErrInvalidTransition, active.ID, active.Status)
	}

	now := defaultNow(req.Now)
	c := model.Cycle{
		ID:       req.ID,
		Duration: req.Duration,
		Started:  now,
		Ends:     now.Add(dur),
		Status:   model.CycleStatusBuilding,
	}
	path, err := s.Write(context.Background(), c)
	if err != nil {
		return CycleNewResult{}, err
	}
	return CycleNewResult{Cycle: c, Path: path}, nil
}

// ParseDuration parses a Go-ish duration string limited to the units
// Appetite cares about: `<n>d` (days), `<n>w` (weeks), `<n>h` (hours,
// for tests). The grammar is intentionally narrow — time.ParseDuration
// rejects `5d`, and we don't want to invite ambiguity by also
// accepting `5d12h` mixed forms.
func ParseDuration(s string) (time.Duration, error) {
	if len(s) < 2 {
		return 0, fmt.Errorf("expected <n>{d|w|h}, got %q", s)
	}
	unit := s[len(s)-1]
	n, err := strconv.Atoi(s[:len(s)-1])
	if err != nil || n <= 0 {
		return 0, fmt.Errorf("expected <n>{d|w|h}, got %q", s)
	}
	switch unit {
	case 'd':
		return time.Duration(n) * 24 * time.Hour, nil
	case 'w':
		return time.Duration(n) * 7 * 24 * time.Hour, nil
	case 'h':
		return time.Duration(n) * time.Hour, nil
	}
	return 0, fmt.Errorf("expected <n>{d|w|h}, got %q", s)
}

// findActiveCycle returns the single open (non-closed) cycle, or nil
// if none exists. Returns an error only if more than one open cycle
// is found — that's a broken invariant the user must resolve before
// the workflow can proceed.
func findActiveCycle(s store.Store) (*model.Cycle, error) {
	kind := model.KindCycle
	entities, err := s.List(context.Background(), store.Filter{Kind: &kind})
	if err != nil {
		return nil, err
	}
	var active []model.Cycle
	for _, e := range entities {
		c, ok := e.(model.Cycle)
		if !ok {
			continue
		}
		if c.Status != model.CycleStatusClosed {
			active = append(active, c)
		}
	}
	switch len(active) {
	case 0:
		return nil, nil
	case 1:
		return &active[0], nil
	default:
		return nil, fmt.Errorf("inconsistent state: %d open cycles (close all but one)", len(active))
	}
}
