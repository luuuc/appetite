package cli

import (
	"github.com/luuuc/appetite/internal/markdown"
	"github.com/luuuc/appetite/internal/store"
)

// openStore returns the Markdown-backed store rooted at env.Dir.
// Subcommands call this once; failure to open is handled by the
// store's first read/write (the adapter creates directories lazily
// and surfaces filesystem errors at the call site).
func openStore(env Env) store.Store {
	return markdown.New(env.Dir)
}
