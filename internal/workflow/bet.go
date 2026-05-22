package workflow

import (
	"context"
	"fmt"
	"time"

	"github.com/luuuc/appetite/internal/model"
	"github.com/luuuc/appetite/internal/store"
)

// BetRequest places a bet — flipping a shaped pitch to `bet` status
// and appending an entry to the cycle's Bets list.
type BetRequest struct {
	Pitch    string         // pitch slug (required)
	Cycle    string         // cycle id (required)
	Appetite model.Appetite // micro|small|medium|large (required)
	Now      time.Time
}

// BetResult reports the paths of the two writes that happen during a
// bet — the pitch (status flip) and the cycle (Bets append).
type BetResult struct {
	PitchPath string
	CyclePath string
}

// Bet validates the bet, flips the pitch's status from `shaped` to
// `bet`, and appends the bet to the cycle's Bets list. Multi-write
// failure mode: see the package doc.
func Bet(s store.Store, req BetRequest) (BetResult, error) {
	if req.Pitch == "" {
		return BetResult{}, fmt.Errorf("bet: pitch slug is required")
	}
	if req.Cycle == "" {
		return BetResult{}, fmt.Errorf("bet: cycle id is required")
	}
	if !req.Appetite.Valid() {
		return BetResult{}, fmt.Errorf("bet: appetite %q is not one of micro|small|medium|large", req.Appetite)
	}

	ctx := context.Background()

	p, err := store.ReadAs[model.Pitch](ctx, s, model.Pitch{Slug: req.Pitch}.Path())
	if err != nil {
		return BetResult{}, err
	}
	if err := model.ValidatePitchTransition(p.Status, model.PitchStatusBet); err != nil {
		return BetResult{}, err
	}

	c, err := store.ReadAs[model.Cycle](ctx, s, model.Cycle{ID: req.Cycle}.Path())
	if err != nil {
		return BetResult{}, err
	}
	if c.Status != model.CycleStatusBuilding {
		return BetResult{}, fmt.Errorf("%w: cannot bet into cycle %q in status %s (must be building)",
			ErrInvalidTransition, c.ID, c.Status)
	}

	now := defaultNow(req.Now)
	p.Status = model.PitchStatusBet
	c.Bets = append(c.Bets, model.Bet{
		Pitch:    req.Pitch,
		Appetite: req.Appetite,
		BetAt:    now,
	})

	pitchPath, err := s.Write(ctx, p)
	if err != nil {
		return BetResult{}, err
	}
	cyclePath, err := s.Write(ctx, c)
	if err != nil {
		return BetResult{PitchPath: pitchPath}, fmt.Errorf("pitch flipped to bet but cycle write failed: %w", err)
	}
	return BetResult{PitchPath: pitchPath, CyclePath: cyclePath}, nil
}
