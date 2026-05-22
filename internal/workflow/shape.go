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

// requiredIngredients is the ordered list of section headings a pitch
// must carry before it can be finalized. The order is preserved in
// the generated --new template so first-time pitches start with the
// right scaffolding.
var requiredIngredients = []string{
	"Problem",
	"Appetite",
	"Solution",
	"Rabbit holes",
	"No-gos",
}

// --- Request / result types ---------------------------------------------

// ShapeNewRequest creates an empty pitch in `shaping` from scratch.
type ShapeNewRequest struct {
	Slug  string // required
	Title string // required (display name)
	Now   time.Time
}

// ShapeFromRequest creates a pitch seeded from a signal. The signal
// body lands inside the new pitch's Problem section.
type ShapeFromRequest struct {
	// SignalPath is the relative path to the source signal, e.g.
	// "signals/raw/export-request.md".
	SignalPath string
	// Slug overrides the derived pitch slug (default: signal slug).
	Slug string
	// Title overrides the derived title (default: titleized slug).
	Title string
	Now   time.Time
}

// ShapeFinalizeRequest flips a `shaping` pitch to `shaped` after
// validating the five ingredients and at least one scope card.
type ShapeFinalizeRequest struct {
	Slug string
	Now  time.Time
}

// ShapeResult reports what was written.
type ShapeResult struct {
	Pitch model.Pitch
	Path  string
}

// --- ShapeNew ----------------------------------------------------------

// ShapeNew writes a fresh pitch in `shaping` status with a template
// body that includes every required ingredient heading and an empty
// Scope section. The operator (or AI) fills the body iteratively
// before calling Finalize.
func ShapeNew(s store.Store, req ShapeNewRequest) (ShapeResult, error) {
	if req.Slug == "" {
		return ShapeResult{}, fmt.Errorf("pitch slug is required")
	}
	if req.Title == "" {
		return ShapeResult{}, fmt.Errorf("pitch title is required")
	}

	if existing, err := store.ReadAs[model.Pitch](context.Background(), s, model.Pitch{Slug: req.Slug}.Path()); err == nil {
		return ShapeResult{}, fmt.Errorf("pitch %q already exists at status %q", existing.Slug, existing.Status)
	} else if !errors.Is(err, store.ErrNotFound) {
		return ShapeResult{}, err
	}

	p := model.Pitch{
		Slug:   req.Slug,
		Title:  req.Title,
		Status: model.PitchStatusShaping,
		Body:   pitchTemplate(""),
	}
	return writePitch(s, p, defaultNow(req.Now))
}

// --- ShapeFrom ---------------------------------------------------------

// ShapeFrom reads a signal and writes a pitch seeded from its body.
// The signal stays in place; archiving is the operator's call (sweep
// commands live in a later cycle).
func ShapeFrom(s store.Store, req ShapeFromRequest) (ShapeResult, error) {
	if req.SignalPath == "" {
		return ShapeResult{}, fmt.Errorf("signal path is required")
	}

	sig, err := store.ReadAs[model.Signal](context.Background(), s, req.SignalPath)
	if err != nil {
		return ShapeResult{}, err
	}

	slug := req.Slug
	if slug == "" {
		slug = sig.Slug
	}
	title := req.Title
	if title == "" {
		title = titleize(slug)
	}

	if _, err := store.ReadAs[model.Pitch](context.Background(), s, model.Pitch{Slug: slug}.Path()); err == nil {
		return ShapeResult{}, fmt.Errorf("pitch %q already exists", slug)
	} else if !errors.Is(err, store.ErrNotFound) {
		return ShapeResult{}, err
	}

	p := model.Pitch{
		Slug:       slug,
		Title:      title,
		Status:     model.PitchStatusShaping,
		ShapedFrom: []string{sig.Slug},
		Body:       pitchTemplate(strings.TrimSpace(sig.Body)),
	}
	return writePitch(s, p, defaultNow(req.Now))
}

// --- ShapeFinalize -----------------------------------------------------

// ShapeFinalize validates the five ingredients and at least one
// scope card, then flips status from `shaping` to `shaped` and sets
// ShapedAt. The path on disk does not change — only frontmatter.
func ShapeFinalize(s store.Store, req ShapeFinalizeRequest) (ShapeResult, error) {
	if req.Slug == "" {
		return ShapeResult{}, fmt.Errorf("pitch slug is required")
	}

	p, err := store.ReadAs[model.Pitch](context.Background(), s, model.Pitch{Slug: req.Slug}.Path())
	if err != nil {
		return ShapeResult{}, err
	}

	if err := model.ValidatePitchTransition(p.Status, model.PitchStatusShaped); err != nil {
		return ShapeResult{}, err
	}

	if missing := missingIngredients(p.Body); len(missing) > 0 {
		return ShapeResult{}, fmt.Errorf("%w: pitch %q is missing required section(s): %s",
			ErrInvalidTransition, p.Slug, strings.Join(missing, ", "))
	}
	cards, invalid := ParseScope(p.Body)
	if len(invalid) > 0 {
		return ShapeResult{}, fmt.Errorf("%w: pitch %q has malformed scope line: %s",
			ErrInvalidTransition, p.Slug, invalid[0])
	}
	if len(cards) == 0 {
		return ShapeResult{}, fmt.Errorf("%w: pitch %q has no scope cards (need at least one `- [ ] **Title** — Description` line under ## Scope)",
			ErrInvalidTransition, p.Slug)
	}

	now := defaultNow(req.Now)
	p.Status = model.PitchStatusShaped
	p.ShapedAt = &now

	return writePitch(s, p, now)
}

// --- helpers -----------------------------------------------------------

// writePitch is the shared write path for ShapeNew/From/Finalize.
// It persists the pitch and returns the canonical result.
func writePitch(s store.Store, p model.Pitch, _ time.Time) (ShapeResult, error) {
	path, err := s.Write(context.Background(), p)
	if err != nil {
		return ShapeResult{}, err
	}
	return ShapeResult{Pitch: p, Path: path}, nil
}

// pitchTemplate renders the body skeleton for a freshly-shaped pitch.
// problem is pre-filled into the ## Problem section (used by
// ShapeFrom to carry the signal body forward); empty for ShapeNew.
func pitchTemplate(problem string) string {
	var b strings.Builder
	for i, h := range requiredIngredients {
		if i > 0 {
			b.WriteString("\n")
		}
		b.WriteString("## ")
		b.WriteString(h)
		b.WriteString("\n\n")
		if h == "Problem" && problem != "" {
			b.WriteString(problem)
			b.WriteString("\n")
		}
	}
	b.WriteString("\n## Scope\n\n")
	return b.String()
}

// missingIngredients returns the ingredient names absent from body.
// Headings are matched exact-case at line start; the template emits
// them in the canonical form so a round-trip of --new → finalize
// always passes.
func missingIngredients(body string) []string {
	have := make(map[string]bool, len(requiredIngredients))
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimRight(line, "\r")
		if !strings.HasPrefix(line, "## ") {
			continue
		}
		heading := strings.TrimSpace(strings.TrimPrefix(line, "## "))
		for _, h := range requiredIngredients {
			if heading == h {
				have[h] = true
			}
		}
	}
	var missing []string
	for _, h := range requiredIngredients {
		if !have[h] {
			missing = append(missing, h)
		}
	}
	return missing
}

// titleize converts a kebab-case slug to a human Title Case string.
// "csv-export" → "Csv Export". Crude but enough for an auto-derived
// title — the operator can rewrite it.
func titleize(slug string) string {
	parts := strings.Split(slug, "-")
	for i, p := range parts {
		if p == "" {
			continue
		}
		parts[i] = strings.ToUpper(p[:1]) + p[1:]
	}
	return strings.Join(parts, " ")
}
