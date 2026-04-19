// Package store defines the storage interface every Appetite backend
// implements. The Markdown/YAML adapter under internal/markdown is the
// only implementation today; a future PG or sync adapter would live in
// its own subpackage and satisfy the same contract.
//
// Callers address entities by their relative path inside .appetite/
// (for example "pitches/csv-export.md" or "cycles/2026-w15/cycle.yml").
// The path is derived from the entity's own fields via
// model.Entity.Path — adapters do not impose their own layout.
package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/luuuc/appetite/internal/model"
)

// ErrNotFound is returned when an entity cannot be located at the
// requested path. Callers distinguish it with errors.Is.
var ErrNotFound = errors.New("entity not found")

// Store is the contract every storage backend satisfies. All methods
// take a context so future adapters (network-backed sync, for instance)
// can honor cancellation; today's Markdown adapter ignores it.
type Store interface {
	// Write persists an entity. The adapter derives the target path
	// from e.Path(). On success the written path is returned —
	// callers should use it for subsequent reads to avoid reconstructing
	// it from the entity's own fields.
	Write(ctx context.Context, e model.Entity) (path string, err error)

	// Read loads a single entity by its relative path. Returns
	// ErrNotFound if no entity exists at that path.
	Read(ctx context.Context, path string) (model.Entity, error)

	// List returns every entity matching the filter. An empty filter
	// returns every entity under .appetite/. Callers receive a
	// non-nil empty slice when nothing matches.
	List(ctx context.Context, f Filter) ([]model.Entity, error)

	// Delete removes an entity by path. Returns ErrNotFound if the
	// path does not exist.
	Delete(ctx context.Context, path string) error
}

// Filter narrows List results. Nil/zero fields mean "don't filter on
// that dimension." The adapter pushes these down where cheap — the
// Markdown adapter skips top-level directories that don't match Kind,
// and skips cycle subdirectories that don't match Cycle.
type Filter struct {
	// Kind restricts results to one entity kind (signals, pitches, cycles,
	// cards, cooldowns). Nil returns every kind.
	Kind *model.Kind

	// Cycle restricts cycle-scoped entities (cards, cooldowns, the
	// cycle itself) to a specific cycle id. Nil returns every cycle.
	// When Cycle is set, non-cycle-scoped kinds (signals, pitches) are
	// excluded from results — "all signals in cycle X" is a nonsensical
	// intersection by design, so it resolves to empty.
	Cycle *string

	// Tags requires that the entity carry every listed tag. Only
	// signals have tags today; for other kinds this field is ignored.
	Tags []string
}

// ReadAs is a type-safe wrapper around Store.Read. It reads the entity
// at path and asserts it to the requested concrete type. Returns a
// typed zero value and an error if the path is missing or the stored
// kind does not match T.
//
// Example:
//
//	p, err := store.ReadAs[model.Pitch](ctx, s, "pitches/csv-export.md")
func ReadAs[T model.Entity](ctx context.Context, s Store, path string) (T, error) {
	var zero T
	e, err := s.Read(ctx, path)
	if err != nil {
		return zero, err
	}
	t, ok := e.(T)
	if !ok {
		return zero, fmt.Errorf("store: path %q holds %T, not %T", path, e, zero)
	}
	return t, nil
}
