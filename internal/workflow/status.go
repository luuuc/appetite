package workflow

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/luuuc/appetite/internal/model"
	"github.com/luuuc/appetite/internal/store"
)

// StatusRequest selects the cycle to render. Empty Cycle resolves
// to the single open cycle (building, shipping, or cooldown).
type StatusRequest struct {
	Cycle string // optional
	Now   time.Time
}

// BetView aggregates one bet's display state: the bet itself, its
// pitch (for status), the cards (for progress + stuck), and a
// computed cards-down count.
type BetView struct {
	Bet         model.Bet
	Pitch       model.Pitch
	Cards       []model.Card
	CardsDone   int
	StuckCards  []model.Card // subset of Cards
}

// StatusResult is the structured snapshot the CLI renderer turns
// into the 80-column text view.
type StatusResult struct {
	Cycle    model.Cycle
	DayN     int // 1-based day-of-cycle, capped at DayTotal
	DayTotal int // cycle.Duration in days (rounded up; minimum 1)
	Bets     []BetView
	Passed   []model.Passed
	Cooldown *model.Cooldown // non-nil if a cooldown file exists
}

// Status gathers the active (or explicit) cycle, its cards, the
// pitches referenced by its bets, and any cooldown — everything the
// renderer in internal/cli needs to draw the status view. It does
// not produce text; rendering is the CLI's responsibility.
func Status(s store.Store, req StatusRequest) (StatusResult, error) {
	ctx := context.Background()
	now := defaultNow(req.Now)

	var cycle model.Cycle
	if req.Cycle != "" {
		c, err := store.ReadAs[model.Cycle](ctx, s, model.Cycle{ID: req.Cycle}.Path())
		if err != nil {
			return StatusResult{}, err
		}
		cycle = c
	} else {
		active, err := findActiveCycle(s)
		if err != nil {
			return StatusResult{}, err
		}
		if active == nil {
			return StatusResult{}, fmt.Errorf("%w: no active cycle (run `appetite cycle new`)", ErrNotFound)
		}
		cycle = *active
	}

	// Day math: total = ceil(Duration / 24h); current = days since
	// Started + 1, clamped to [1, total]. A malformed Duration is
	// propagated rather than silently zero'd — a zero threshold
	// would mark every uphill card stuck.
	dur, err := ParseDuration(cycle.Duration)
	if err != nil {
		return StatusResult{}, fmt.Errorf("cycle %q has malformed duration: %w", cycle.ID, err)
	}
	total := int((dur + 24*time.Hour - 1) / (24 * time.Hour))
	if total < 1 {
		total = 1
	}
	elapsed := now.Sub(cycle.Started)
	dayN := int(elapsed/(24*time.Hour)) + 1
	if dayN < 1 {
		dayN = 1
	}
	if dayN > total {
		dayN = total
	}

	cards, err := listCardsForCycle(s, cycle.ID)
	if err != nil {
		return StatusResult{}, err
	}

	bets := make([]BetView, 0, len(cycle.Bets))
	for _, b := range cycle.Bets {
		p, err := store.ReadAs[model.Pitch](ctx, s, model.Pitch{Slug: b.Pitch}.Path())
		if err != nil {
			return StatusResult{}, err
		}
		bv := BetView{Bet: b, Pitch: p}
		for _, c := range cards {
			if c.Pitch == b.Pitch {
				bv.Cards = append(bv.Cards, c)
				if c.Progress >= 100 {
					bv.CardsDone++
				}
				if isStuck(c, dur, now) {
					bv.StuckCards = append(bv.StuckCards, c)
				}
			}
		}
		// Stable card order — by slug — so output is deterministic.
		sort.Slice(bv.Cards, func(i, j int) bool { return bv.Cards[i].Slug < bv.Cards[j].Slug })
		sort.Slice(bv.StuckCards, func(i, j int) bool { return bv.StuckCards[i].Slug < bv.StuckCards[j].Slug })
		bets = append(bets, bv)
	}

	var cooldown *model.Cooldown
	if cd, err := store.ReadAs[model.Cooldown](ctx, s, model.Cooldown{Cycle: cycle.ID}.Path()); err == nil {
		cooldown = &cd
	}

	return StatusResult{
		Cycle:    cycle,
		DayN:     dayN,
		DayTotal: total,
		Bets:     bets,
		Passed:   cycle.Passed,
		Cooldown: cooldown,
	}, nil
}

// isStuck reports whether c is "stuck": uphill, and HillUpdatedAt is
// older than half the cycle's duration. A card with no HillUpdatedAt
// (legacy data) is not stuck — the absence is honest about not
// knowing.
func isStuck(c model.Card, cycleDur time.Duration, now time.Time) bool {
	if c.Hill != model.HillUphill {
		return false
	}
	if c.HillUpdatedAt == nil {
		return false
	}
	return now.Sub(*c.HillUpdatedAt) > cycleDur/2
}

// listCardsForCycle returns every card under a single cycle.
func listCardsForCycle(s store.Store, cycle string) ([]model.Card, error) {
	kind := model.KindCard
	entities, err := s.List(context.Background(), store.Filter{Kind: &kind, Cycle: &cycle})
	if err != nil {
		return nil, err
	}
	out := make([]model.Card, 0, len(entities))
	for _, e := range entities {
		if c, ok := e.(model.Card); ok {
			out = append(out, c)
		}
	}
	return out, nil
}
