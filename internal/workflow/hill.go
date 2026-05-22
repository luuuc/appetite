package workflow

import (
	"context"
	"fmt"
	"time"

	"github.com/luuuc/appetite/internal/model"
	"github.com/luuuc/appetite/internal/store"
)

// HillRequest updates a card's hill position and/or progress, with
// an explicit --done assertion required to ship (progress=100).
type HillRequest struct {
	Card     string             // card slug (required)
	Cycle    string             // optional; resolved by scanning cards when empty
	Position model.HillPosition // optional ("" = no change)
	Progress *int               // optional pointer so 0 is distinguishable from "not set"
	Done     bool
	Now      time.Time
}

// HillResult reports what changed. Cascades into pitch.shipped and
// cycle.shipping are surfaced via non-nil pointers so the CLI can
// render them.
type HillResult struct {
	Card         model.Card
	PitchShipped *model.Pitch
	CycleShipped *model.Cycle
}

// Hill updates the card and cascades pitch/cycle transitions when
// the card's update completes the work for its pitch or cycle.
//
// Validation rules:
//   - Position, if set, must be uphill or downhill.
//   - Progress, if set, must be 0..100.
//   - Progress == 100 requires Done. This is the workflow rule the
//     pitch documents and exits 2 on (ErrDoneCriteriaUnmet).
//   - Done requires Progress == 100. This is an argv-shape error
//     (exit 1), not a workflow rule — a "done" without "progress 100"
//     is incoherent typing, not a state-machine violation.
//   - At least one of Position / Progress / Done must change
//     something — empty requests are rejected as argv errors.
//
// Cascade order: card write → pitch-shipped check → cycle-shipping
// check. Each cascade is gated on the target not already being in
// the destination state, so re-running --done on an already-shipped
// card is a clean no-op. Multi-write failure mode: see the package
// doc.
func Hill(s store.Store, req HillRequest) (HillResult, error) {
	if req.Card == "" {
		return HillResult{}, fmt.Errorf("hill: card slug is required")
	}
	if req.Position != "" && !req.Position.Valid() {
		return HillResult{}, fmt.Errorf("hill: unknown position %q (uphill|downhill)", req.Position)
	}
	if req.Progress != nil {
		if *req.Progress < 0 || *req.Progress > 100 {
			return HillResult{}, fmt.Errorf("hill: progress %d out of range 0..100", *req.Progress)
		}
		if *req.Progress == 100 && !req.Done {
			return HillResult{}, fmt.Errorf("%w: setting progress=100 requires --done", ErrDoneCriteriaUnmet)
		}
	}
	if req.Done && (req.Progress == nil || *req.Progress != 100) {
		return HillResult{}, fmt.Errorf("hill: --done requires --progress 100")
	}
	if req.Position == "" && req.Progress == nil && !req.Done {
		return HillResult{}, fmt.Errorf("hill: at least one of --position / --progress is required")
	}

	ctx := context.Background()

	card, err := findCard(s, req.Card, req.Cycle)
	if err != nil {
		return HillResult{}, err
	}

	// Card mutations are only meaningful while the cycle is still
	// building — once the cycle moves to shipping/cooldown/closed,
	// the work has been handed off and the hill is no longer the
	// state of record. Re-runs against a shipped card surface here.
	cyc, err := store.ReadAs[model.Cycle](ctx, s, model.Cycle{ID: card.Cycle}.Path())
	if err != nil {
		return HillResult{}, err
	}
	if cyc.Status != model.CycleStatusBuilding {
		return HillResult{}, fmt.Errorf("%w: cannot update card %q in cycle %q (status %s)",
			ErrInvalidTransition, card.Slug, card.Cycle, cyc.Status)
	}

	now := defaultNow(req.Now)
	if req.Position != "" {
		card.Hill = req.Position
	}
	if req.Progress != nil {
		card.Progress = *req.Progress
	}
	if req.Done {
		// Ship implies downhill — uphill at 100% is nonsense.
		card.Hill = model.HillDownhill
	}
	card.HillUpdatedAt = &now

	if _, err := s.Write(ctx, card); err != nil {
		return HillResult{}, err
	}

	res := HillResult{Card: card}
	if !req.Done {
		return res, nil
	}

	// Cascade: pitch ships when all its cards in this cycle hit 100.
	allDone, err := pitchCardsAllDone(s, card.Pitch, card.Cycle)
	if err != nil {
		return res, err
	}
	if !allDone {
		return res, nil
	}

	pitch, err := store.ReadAs[model.Pitch](ctx, s, model.Pitch{Slug: card.Pitch}.Path())
	if err != nil {
		return res, err
	}
	// Re-running --done on a card whose pitch already shipped is a
	// no-op: the cascade has nothing new to assert. Skip past the
	// transition validator (which would reject shipped → shipped)
	// and the subsequent cycle cascade is checked below independently.
	if pitch.Status != model.PitchStatusShipped {
		if err := model.ValidatePitchTransition(pitch.Status, model.PitchStatusShipped); err != nil {
			return res, err
		}
		pitch.Status = model.PitchStatusShipped
		if _, err := s.Write(ctx, pitch); err != nil {
			return res, err
		}
		res.PitchShipped = &pitch
	}

	// Cascade: cycle flips to shipping when every bet is shipped.
	// The early guard at the top of Hill already confirmed cycle.Status
	// == building when this op started, and we have not mutated the
	// cycle since — so we reuse the already-read `cyc`.
	cycle := cyc
	allShipped, err := cycleBetsAllShipped(s, cycle)
	if err != nil {
		return res, err
	}
	if !allShipped {
		return res, nil
	}
	if err := model.ValidateCycleTransition(cycle.Status, model.CycleStatusShipping); err != nil {
		return res, err
	}
	cycle.Status = model.CycleStatusShipping
	if _, err := s.Write(ctx, cycle); err != nil {
		return res, err
	}
	res.CycleShipped = &cycle
	return res, nil
}

// findCard locates a card by slug. An explicit cycle short-circuits
// the scan. Otherwise List+Filter narrows to cards and the slug is
// matched in-memory — collisions across cycles return an error so
// the operator picks one via --cycle.
func findCard(s store.Store, slug, explicitCycle string) (model.Card, error) {
	ctx := context.Background()
	if explicitCycle != "" {
		return store.ReadAs[model.Card](ctx, s, model.Card{Cycle: explicitCycle, Slug: slug}.Path())
	}
	kind := model.KindCard
	entities, err := s.List(ctx, store.Filter{Kind: &kind})
	if err != nil {
		return model.Card{}, err
	}
	var matches []model.Card
	for _, e := range entities {
		c, ok := e.(model.Card)
		if !ok {
			continue
		}
		if c.Slug == slug {
			matches = append(matches, c)
		}
	}
	switch len(matches) {
	case 0:
		return model.Card{}, fmt.Errorf("%w: card %q", ErrNotFound, slug)
	case 1:
		return matches[0], nil
	default:
		return model.Card{}, fmt.Errorf("card %q exists in multiple cycles; pass --cycle to disambiguate", slug)
	}
}

// pitchCardsAllDone reports whether every card for `pitch` in
// `cycle` has progress=100. Used to gate the pitch-shipped cascade.
func pitchCardsAllDone(s store.Store, pitch, cycle string) (bool, error) {
	kind := model.KindCard
	entities, err := s.List(context.Background(), store.Filter{Kind: &kind, Cycle: &cycle})
	if err != nil {
		return false, err
	}
	any := false
	for _, e := range entities {
		c, ok := e.(model.Card)
		if !ok || c.Pitch != pitch {
			continue
		}
		any = true
		if c.Progress < 100 {
			return false, nil
		}
	}
	return any, nil
}

// cycleBetsAllShipped reports whether every pitch in cycle.Bets is
// in `shipped` status. Used to gate the cycle-shipping cascade.
func cycleBetsAllShipped(s store.Store, cycle model.Cycle) (bool, error) {
	if len(cycle.Bets) == 0 {
		return false, nil
	}
	ctx := context.Background()
	for _, b := range cycle.Bets {
		p, err := store.ReadAs[model.Pitch](ctx, s, model.Pitch{Slug: b.Pitch}.Path())
		if err != nil {
			return false, err
		}
		if p.Status != model.PitchStatusShipped {
			return false, nil
		}
	}
	return true, nil
}
