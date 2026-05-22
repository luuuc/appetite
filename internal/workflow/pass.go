package workflow

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/luuuc/appetite/internal/model"
	"github.com/luuuc/appetite/internal/store"
)

// PassRequest drops a shaped pitch with a recorded reason.
type PassRequest struct {
	Pitch  string // pitch slug (required)
	Reason string // free-form reason (required)
	Cycle  string // optional; defaults to the active cycle
	Now    time.Time
}

// PassResult reports the paths of the two writes.
type PassResult struct {
	PitchPath string
	CyclePath string
}

// Pass flips a shaped pitch's status to `passed` and records the
// reason on the cycle. When req.Cycle is empty, the active cycle
// (the one not yet closed) is used. Passing without any active cycle
// is allowed: the pitch flips to `passed`, but no cycle entry is
// recorded (CyclePath is empty in the result). Multi-write failure
// mode: see the package doc.
func Pass(s store.Store, req PassRequest) (PassResult, error) {
	if req.Pitch == "" {
		return PassResult{}, fmt.Errorf("pass: pitch slug is required")
	}
	if strings.TrimSpace(req.Reason) == "" {
		return PassResult{}, fmt.Errorf("pass: --reason is required")
	}

	ctx := context.Background()

	p, err := store.ReadAs[model.Pitch](ctx, s, model.Pitch{Slug: req.Pitch}.Path())
	if err != nil {
		return PassResult{}, err
	}
	if err := model.ValidatePitchTransition(p.Status, model.PitchStatusPassed); err != nil {
		return PassResult{}, err
	}

	// Cycle resolution: explicit --cycle is held to building-only
	// (the operator named it, surface the mismatch); the implicit
	// default just silently skips recording when no building cycle
	// is open — passing a shaped pitch is always a valid pitch
	// transition (03-workflow.md) regardless of cycle state.
	var cyclePtr *model.Cycle
	if req.Cycle != "" {
		c, err := store.ReadAs[model.Cycle](ctx, s, model.Cycle{ID: req.Cycle}.Path())
		if err != nil {
			return PassResult{}, err
		}
		if c.Status != model.CycleStatusBuilding {
			return PassResult{}, fmt.Errorf("%w: cannot record pass on cycle %q in status %s (must be building)",
				ErrInvalidTransition, c.ID, c.Status)
		}
		cyclePtr = &c
	} else {
		active, err := findActiveCycle(s)
		if err != nil {
			return PassResult{}, err
		}
		if active != nil && active.Status == model.CycleStatusBuilding {
			cyclePtr = active
		}
	}

	now := defaultNow(req.Now)
	p.Status = model.PitchStatusPassed

	pitchPath, err := s.Write(ctx, p)
	if err != nil {
		return PassResult{}, err
	}
	if cyclePtr == nil {
		return PassResult{PitchPath: pitchPath}, nil
	}

	cyclePtr.Passed = append(cyclePtr.Passed, model.Passed{
		Pitch:    req.Pitch,
		Reason:   strings.TrimSpace(req.Reason),
		PassedAt: now,
	})
	cyclePath, err := s.Write(ctx, *cyclePtr)
	if err != nil {
		return PassResult{PitchPath: pitchPath}, fmt.Errorf("pitch flipped to passed but cycle write failed: %w", err)
	}
	return PassResult{PitchPath: pitchPath, CyclePath: cyclePath}, nil
}
