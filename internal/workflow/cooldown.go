package workflow

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/luuuc/appetite/internal/model"
	"github.com/luuuc/appetite/internal/store"
)

// CooldownOpenRequest opens cooldown for the active (or explicit)
// cycle. Duration defaults to "1d" when empty.
type CooldownOpenRequest struct {
	Cycle    string // optional
	Duration string // optional (default "1d")
	Now      time.Time
}

// CooldownCloseRequest closes the active cooldown.
type CooldownCloseRequest struct {
	Cycle string // optional
	Now   time.Time
}

// CooldownResult reports the cooldown and (possibly transitioned)
// cycle.
type CooldownResult struct {
	Cooldown model.Cooldown
	Cycle    model.Cycle
}

// CooldownOpen flips a cycle from `shipping` to `cooldown` and
// writes the cooldown.yml file. Refuses to open while any pitch in
// the cycle's bets is still `building` (or `bet`) — cooldown is
// not a way to abandon unfinished work.
func CooldownOpen(s store.Store, req CooldownOpenRequest) (CooldownResult, error) {
	ctx := context.Background()
	dur, err := ParseDuration(coalesceDuration(req.Duration))
	if err != nil {
		return CooldownResult{}, fmt.Errorf("cooldown: %w", err)
	}

	cycle, err := resolveCycle(s, req.Cycle)
	if err != nil {
		return CooldownResult{}, err
	}

	if err := refuseIfBuildingPitches(s, cycle); err != nil {
		return CooldownResult{}, err
	}

	if err := model.ValidateCycleTransition(cycle.Status, model.CycleStatusCooldown); err != nil {
		return CooldownResult{}, err
	}

	now := defaultNow(req.Now)
	cooldown := model.Cooldown{
		Cycle:   cycle.ID,
		Started: now,
		Ends:    now.Add(dur),
		Status:  model.CooldownActive,
	}
	if _, err := s.Write(ctx, cooldown); err != nil {
		return CooldownResult{}, err
	}

	cycle.Status = model.CycleStatusCooldown
	if _, err := s.Write(ctx, cycle); err != nil {
		return CooldownResult{Cooldown: cooldown}, fmt.Errorf("cooldown opened but cycle status write failed: %w", err)
	}
	return CooldownResult{Cooldown: cooldown, Cycle: cycle}, nil
}

// CooldownClose flips the cooldown to `closed` and the cycle to
// `closed`. Both writes happen; cooldown first.
func CooldownClose(s store.Store, req CooldownCloseRequest) (CooldownResult, error) {
	ctx := context.Background()

	cycle, err := resolveCycle(s, req.Cycle)
	if err != nil {
		return CooldownResult{}, err
	}
	cd, err := store.ReadAs[model.Cooldown](ctx, s, model.Cooldown{Cycle: cycle.ID}.Path())
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return CooldownResult{}, fmt.Errorf("%w: no cooldown open for cycle %q", ErrInvalidTransition, cycle.ID)
		}
		return CooldownResult{}, err
	}
	if err := model.ValidateCooldownTransition(cd.Status, model.CooldownClosed); err != nil {
		return CooldownResult{}, err
	}
	if err := model.ValidateCycleTransition(cycle.Status, model.CycleStatusClosed); err != nil {
		return CooldownResult{}, err
	}

	cd.Status = model.CooldownClosed
	if _, err := s.Write(ctx, cd); err != nil {
		return CooldownResult{}, err
	}
	cycle.Status = model.CycleStatusClosed
	if _, err := s.Write(ctx, cycle); err != nil {
		return CooldownResult{Cooldown: cd}, fmt.Errorf("cooldown closed but cycle status write failed: %w", err)
	}
	return CooldownResult{Cooldown: cd, Cycle: cycle}, nil
}

// resolveCycle picks the cycle for cooldown commands. Explicit
// req.Cycle wins; otherwise the (single) non-closed cycle is used.
func resolveCycle(s store.Store, explicit string) (model.Cycle, error) {
	ctx := context.Background()
	if explicit != "" {
		return store.ReadAs[model.Cycle](ctx, s, model.Cycle{ID: explicit}.Path())
	}
	active, err := findActiveCycle(s)
	if err != nil {
		return model.Cycle{}, err
	}
	if active == nil {
		return model.Cycle{}, fmt.Errorf("%w: no active cycle", ErrNotFound)
	}
	return *active, nil
}

// refuseIfBuildingPitches rejects cooldown-open while any bet pitch
// is still being worked. Shipped + passed are fine; bet + building
// block.
func refuseIfBuildingPitches(s store.Store, cycle model.Cycle) error {
	ctx := context.Background()
	for _, b := range cycle.Bets {
		p, err := store.ReadAs[model.Pitch](ctx, s, model.Pitch{Slug: b.Pitch}.Path())
		if err != nil {
			return err
		}
		switch p.Status {
		case model.PitchStatusShipped, model.PitchStatusPassed:
			// fine
		default:
			return fmt.Errorf("%w: pitch %q is still %s — ship or pass it before opening cooldown",
				ErrInvalidTransition, p.Slug, p.Status)
		}
	}
	return nil
}

// coalesceDuration returns d if non-empty, else "1d" — the v0.1
// hardcoded default per the pitch's "no config loader" no-go.
func coalesceDuration(d string) string {
	if d == "" {
		return "1d"
	}
	return d
}
