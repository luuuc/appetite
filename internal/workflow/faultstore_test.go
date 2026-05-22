package workflow

import (
	"context"
	"errors"
	"strings"

	"github.com/luuuc/appetite/internal/model"
	"github.com/luuuc/appetite/internal/store"
)

// errInjected is the sentinel error every faultStore method returns
// when its predicate matches. Tests use errors.Is(err, errInjected)
// to confirm the fault path actually fired.
var errInjected = errors.New("injected fault")

// faultStore wraps another store.Store and returns errInjected from
// any of Read/Write/List/Delete whose path matches the configured
// predicate. The wrapper is for testing only — it does not implement
// the full contract under failure (atomicity, etc.).
type faultStore struct {
	inner store.Store

	// readMatch / writeMatch / listMatch are substring filters: if
	// non-empty and the operation's path (or, for List, the cycle
	// filter or kind name) contains it, the operation fails with
	// errInjected. listMatch=="any" fails List unconditionally.
	readMatch  string
	writeMatch string
	listMatch  string
}

func (f *faultStore) Read(ctx context.Context, path string) (model.Entity, error) {
	if f.readMatch != "" && strings.Contains(path, f.readMatch) {
		return nil, errInjected
	}
	return f.inner.Read(ctx, path)
}

func (f *faultStore) Write(ctx context.Context, e model.Entity) (string, error) {
	if f.writeMatch != "" && strings.Contains(e.Path(), f.writeMatch) {
		return "", errInjected
	}
	return f.inner.Write(ctx, e)
}

func (f *faultStore) List(ctx context.Context, fl store.Filter) ([]model.Entity, error) {
	if f.listMatch == "any" {
		return nil, errInjected
	}
	if f.listMatch != "" && fl.Kind != nil && strings.Contains(string(*fl.Kind), f.listMatch) {
		return nil, errInjected
	}
	return f.inner.List(ctx, fl)
}

func (f *faultStore) Delete(ctx context.Context, path string) error {
	return f.inner.Delete(ctx, path)
}
