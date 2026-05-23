package cli

import (
	"bytes"
	"errors"
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
	commands := fs.String("commands", "", "also install slash commands into the AI tool's commands dir (claude)")
	force := fs.Bool("force", false, "overwrite divergent slash command files at the target")
	if err := fs.Parse(reorderFlags(fs, args)); err != nil {
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
	} else {
		fmt.Fprintf(env.Stdout, "appetite: initialized %s\n", env.Dir)
	}

	if *commands != "" {
		return installCommands(env, *commands, *force)
	}
	return nil
}

// commandsSourceDir names the slash command source tree relative to
// the operator's CWD. It is a `var` so tests can override it without
// faking the filesystem.
var commandsSourceDir = "commands"

// installCommands copies the reference markdown files from
// commands/ into the target AI tool's commands directory. The only
// supported tool this cycle is `claude` (.claude/commands/). Unknown
// tool names exit with workflow code 2 — they're an operator error,
// not a transient failure.
func installCommands(env Env, tool string, force bool) error {
	target, err := targetCommandsDir(tool)
	if err != nil {
		return err
	}

	entries, err := os.ReadDir(commandsSourceDir)
	if err != nil {
		return fmt.Errorf("read %s/: %w", commandsSourceDir, err)
	}

	if err := os.MkdirAll(target, 0o755); err != nil {
		return fmt.Errorf("create %s: %w", target, err)
	}

	var copied, skipped, conflicted []string
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".md" {
			continue
		}
		src := filepath.Join(commandsSourceDir, e.Name())
		dst := filepath.Join(target, e.Name())
		action, err := copyCommandFile(src, dst, force)
		if err != nil {
			return err
		}
		switch action {
		case actionCopied:
			copied = append(copied, e.Name())
		case actionSkipped:
			skipped = append(skipped, e.Name())
		case actionConflict:
			conflicted = append(conflicted, e.Name())
		}
	}

	if len(conflicted) > 0 {
		fmt.Fprintf(env.Stderr, "init: commands diverged at %s (pass --force to overwrite):\n", target)
		for _, name := range conflicted {
			fmt.Fprintf(env.Stderr, "  %s\n", name)
		}
		return fmt.Errorf("%w: %d slash command file(s) diverged", errInitCommandsDiverged, len(conflicted))
	}

	fmt.Fprintf(env.Stdout, "init: installed %d commands into %s", len(copied), target)
	if len(skipped) > 0 {
		fmt.Fprintf(env.Stdout, " (%d already up-to-date)", len(skipped))
	}
	fmt.Fprintln(env.Stdout)
	return nil
}

// errInitCommandsDiverged is the typed sentinel that maps to exit
// code 2 for divergent slash command files. Both unknown-tool and
// diverged-file are workflow-shape errors the operator must resolve.
var errInitCommandsDiverged = errors.New("init: commands diverged")

// targetCommandsDir resolves the per-tool commands directory. The
// pitch only ships `claude` this cycle; other tools are reserved.
// Claude commands install under an `appetite/` subdirectory so they
// don't collide with whatever else lives in `.claude/commands/`;
// Claude Code's subdir convention turns this into the `/appetite:`
// slash-command namespace at invocation time.
func targetCommandsDir(tool string) (string, error) {
	switch tool {
	case "claude":
		return filepath.Join(".claude", "commands", "appetite"), nil
	default:
		return "", fmt.Errorf("%w: --commands %q (only `claude` is supported in v0.1)",
			errInitUnknownTool, tool)
	}
}

// errInitUnknownTool is the typed sentinel that maps to exit code 2
// for an unknown --commands target.
var errInitUnknownTool = errors.New("init: unknown commands tool")

// commandAction names the three outcomes of copying a single file.
type commandAction int

const (
	actionCopied commandAction = iota + 1
	actionSkipped
	actionConflict
)

// copyCommandFile copies src → dst, returning one of:
//
//   - actionCopied: the file was missing or force-overwritten.
//   - actionSkipped: target exists with identical bytes.
//   - actionConflict: target exists with divergent bytes and force
//     is false; the caller fails the whole install.
func copyCommandFile(src, dst string, force bool) (commandAction, error) {
	srcBytes, err := os.ReadFile(src)
	if err != nil {
		return 0, fmt.Errorf("read %s: %w", src, err)
	}
	existing, readErr := os.ReadFile(dst)
	if readErr == nil {
		if bytes.Equal(srcBytes, existing) {
			return actionSkipped, nil
		}
		if !force {
			return actionConflict, nil
		}
	} else if !errors.Is(readErr, os.ErrNotExist) {
		return 0, fmt.Errorf("read %s: %w", dst, readErr)
	}

	// Slash command files are small and operator-readable; a torn
	// write would be obvious. Skip the temp-file dance the storage
	// layer uses and write directly — one less branch to test, and
	// any failure surfaces immediately with the dst path.
	if err := os.WriteFile(dst, srcBytes, 0o644); err != nil {
		return 0, fmt.Errorf("write %s: %w", dst, err)
	}
	return actionCopied, nil
}
