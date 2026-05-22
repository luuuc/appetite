package workflow

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/luuuc/appetite/internal/model"
	"github.com/luuuc/appetite/internal/store"
)

// CutRequest breaks a bet pitch's scope into cards inside a cycle.
type CutRequest struct {
	Pitch string // pitch slug (required)
	Cycle string // optional; resolved from the pitch's bet entry when empty
	Now   time.Time
}

// CutResult lists the cards that were written and the new pitch
// state.
type CutResult struct {
	Pitch model.Pitch
	Cycle string       // resolved cycle id
	Cards []model.Card // in scope-order
}

// Cut reads the pitch's `## Scope`, writes one card per parsed line
// to cycles/<cycle>/cards/<slug>.md, and flips the pitch from `bet`
// to `building`. On any write failure (card or pitch) the cards
// already written are deleted best-effort. Multi-write failure
// mode: see the package doc.
func Cut(s store.Store, req CutRequest) (CutResult, error) {
	if req.Pitch == "" {
		return CutResult{}, fmt.Errorf("cut: pitch slug is required")
	}

	ctx := context.Background()

	p, err := store.ReadAs[model.Pitch](ctx, s, model.Pitch{Slug: req.Pitch}.Path())
	if err != nil {
		return CutResult{}, err
	}
	if err := model.ValidatePitchTransition(p.Status, model.PitchStatusBuilding); err != nil {
		return CutResult{}, err
	}

	cycleID, err := resolveCutCycle(s, p.Slug, req.Cycle)
	if err != nil {
		return CutResult{}, err
	}

	cards, invalid := ParseScope(p.Body)
	if len(invalid) > 0 {
		return CutResult{}, fmt.Errorf("%w: pitch %q has malformed scope line: %s",
			ErrInvalidTransition, p.Slug, invalid[0])
	}
	if len(cards) == 0 {
		return CutResult{}, fmt.Errorf("%w: pitch %q has no scope cards to cut",
			ErrInvalidTransition, p.Slug)
	}

	now := defaultNow(req.Now)
	models, err := buildCards(p.Slug, cycleID, cards, now)
	if err != nil {
		return CutResult{}, err
	}

	// Pre-check that none of the target paths already exist. Two
	// reasons: avoid clobbering operator-edited cards from a prior
	// partial cut, and surface a precise error before any write.
	for _, c := range models {
		if _, err := s.Read(ctx, c.Path()); err == nil {
			return CutResult{}, fmt.Errorf("card %s already exists; resolve before re-cutting", c.Path())
		} else if !errors.Is(err, store.ErrNotFound) {
			return CutResult{}, err
		}
	}

	written := make([]string, 0, len(models))
	for _, c := range models {
		path, err := s.Write(ctx, c)
		if err != nil {
			rollbackCards(s, written)
			return CutResult{}, fmt.Errorf("write card %s: %w", c.Slug, err)
		}
		written = append(written, path)
	}

	p.Status = model.PitchStatusBuilding
	if _, err := s.Write(ctx, p); err != nil {
		rollbackCards(s, written)
		return CutResult{}, fmt.Errorf("flip pitch to building: %w", err)
	}

	return CutResult{Pitch: p, Cycle: cycleID, Cards: models}, nil
}

// resolveCutCycle picks the cycle this cut belongs to. Priority: an
// explicit req.Cycle wins; otherwise look for the (single) cycle
// whose Bets list contains the pitch slug.
func resolveCutCycle(s store.Store, pitch, explicit string) (string, error) {
	if explicit != "" {
		// Surface a clear error if the explicit cycle doesn't exist.
		if _, err := store.ReadAs[model.Cycle](context.Background(), s, model.Cycle{ID: explicit}.Path()); err != nil {
			return "", err
		}
		return explicit, nil
	}

	kind := model.KindCycle
	entities, err := s.List(context.Background(), store.Filter{Kind: &kind})
	if err != nil {
		return "", err
	}
	var matches []string
	for _, e := range entities {
		c, ok := e.(model.Cycle)
		if !ok {
			continue
		}
		for _, b := range c.Bets {
			if b.Pitch == pitch {
				matches = append(matches, c.ID)
				break
			}
		}
	}
	switch len(matches) {
	case 0:
		return "", fmt.Errorf("cut: pitch %q has no bet in any cycle; pass --cycle to disambiguate", pitch)
	case 1:
		return matches[0], nil
	default:
		return "", fmt.Errorf("cut: pitch %q appears in multiple cycles %v; pass --cycle to disambiguate", pitch, matches)
	}
}

// buildCards turns parsed scope lines into Card entities, slugifying
// titles. Slug collisions are rejected so the operator must rename a
// scope card rather than silently overwrite. now seeds HillUpdatedAt
// so stuck detection has a baseline from card creation.
func buildCards(pitch, cycle string, parsed []ScopeCard, now time.Time) ([]model.Card, error) {
	out := make([]model.Card, 0, len(parsed))
	seen := make(map[string]string, len(parsed))
	createdAt := now
	for _, sc := range parsed {
		slug := slugifyWords(sc.Title, 8)
		if slug == "" {
			return nil, fmt.Errorf("cut: scope card title %q has no slug-able characters", sc.Title)
		}
		if existing, dup := seen[slug]; dup {
			return nil, fmt.Errorf("cut: scope card titles %q and %q both slug to %q — rename one",
				existing, sc.Title, slug)
		}
		seen[slug] = sc.Title

		out = append(out, model.Card{
			Slug:          slug,
			Pitch:         pitch,
			Cycle:         cycle,
			Hill:          model.HillUphill,
			Progress:      0,
			HillUpdatedAt: &createdAt,
			Body:          ensureTrailingNewline(strings.TrimSpace(sc.Description)),
		})
	}
	return out, nil
}

// rollbackCards best-effort deletes any cards written before a
// failure. Delete errors are intentionally swallowed: the operator
// already has a write failure to deal with, and a stuck card file is
// surfaced by the next cut's pre-existence check.
func rollbackCards(s store.Store, paths []string) {
	for _, p := range paths {
		_ = s.Delete(context.Background(), p)
	}
}
