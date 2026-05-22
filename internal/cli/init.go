package cli

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

func init() {
	register(Command{
		Name:     "init",
		Synopsis: "create the .appetite/ skeleton (idempotent)",
		Run:      runInit,
	})
}

// initSubdirs is the directory skeleton `appetite init` lays down.
// It mirrors the layout in .doc/definition/05-storage.md — minus
// pitches/archived/, signals/archived/, and cycles/archived/, which
// are created lazily by the sweep commands (deferred to a later
// cycle).
var initSubdirs = []string{
	"signals/raw",
	"pitches",
	"cycles",
}

func runInit(env Env, args []string) error {
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	fs.SetOutput(env.Stderr)
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("init takes no positional arguments")
	}

	// If MkdirAll fails partway through this loop, some subdirs may
	// already exist on disk. That's benign — init is idempotent, so
	// re-running it after the operator fixes the underlying cause
	// (permissions, full disk) completes the skeleton without harm.
	created := 0
	for _, sub := range initSubdirs {
		path := filepath.Join(env.Dir, sub)
		if _, err := os.Stat(path); err == nil {
			continue
		}
		if err := os.MkdirAll(path, 0o755); err != nil {
			return fmt.Errorf("create %s: %w", path, err)
		}
		created++
	}

	if created == 0 {
		fmt.Fprintf(env.Stdout, "appetite: %s already initialized\n", env.Dir)
		return nil
	}
	fmt.Fprintf(env.Stdout, "appetite: initialized %s\n", env.Dir)
	return nil
}
