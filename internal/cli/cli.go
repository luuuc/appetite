// Package cli is the human-facing wrapper around the workflow engine.
// It parses argv, dispatches to subcommands, formats human-readable
// output, and translates workflow errors into the exit codes spelled
// out in .doc/definition/07-mcp-and-cli.md.
//
// Nothing in this package mutates state directly — every subcommand
// translates argv into an internal/workflow call and renders the
// result. The split keeps the workflow package callable from the
// future MCP server (01-04) without rework.
//
// Commands self-register at package init via register(). The registry
// is a package-level map populated by side-effecting init() functions
// in each subcommand file (init.go, version.go, …). This pattern
// keeps the router file ignorant of which commands exist; adding a
// command is a single new file with one init() block.
package cli

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Env carries the I/O and filesystem context that every subcommand
// needs. Tests construct an Env with bytes.Buffer for Stdout/Stderr
// and a temp dir for Dir; production code calls DefaultEnv.
type Env struct {
	// Dir is the path to the .appetite/ root. It does not need to
	// exist — `init` creates it; other commands fail loudly if it's
	// missing. Main normalizes it via filepath.Clean once on entry
	// so subcommands joining against it inherit a stable shape.
	Dir string

	Stdout io.Writer
	Stderr io.Writer
}

// DefaultEnv returns an Env wired to the process's stdio with Dir
// defaulting to ".appetite" (relative). The Dir is the resolved
// CWD-relative path operators expect when running inside a project.
func DefaultEnv() Env {
	return Env{
		Dir:    ".appetite",
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}
}

// Command is one subcommand. Run receives the remaining argv (after
// the subcommand name has been consumed) and the Env, returns an
// error whose exit-code mapping happens in errors.go. A nil error is
// exit 0.
type Command struct {
	Name     string
	Synopsis string
	Run      func(env Env, args []string) error
}

// registry is the static list of commands the binary knows about,
// populated by per-command init() functions. It is mutable package
// state by design — see the package doc comment for the rationale.
var registry = map[string]Command{}

// register adds c to the registry. It panics on duplicate names so
// programmer errors surface during package init, not at runtime.
func register(c Command) {
	if _, dup := registry[c.Name]; dup {
		panic("cli: duplicate command " + c.Name)
	}
	registry[c.Name] = c
}

// Main is the entry point called from cmd/appetite/main.go. It
// returns the process exit code; the caller wires it to os.Exit. This
// shape makes Main testable from in-process tests too — they can
// invoke it without spawning a subprocess.
func Main(argv []string, env Env) int {
	fs := flag.NewFlagSet("appetite", flag.ContinueOnError)
	fs.SetOutput(env.Stderr)
	fs.Usage = func() { printUsage(env.Stderr) }

	dir := fs.String("dir", env.Dir, "path to the .appetite/ directory")
	help := fs.Bool("h", false, "show help")

	if err := fs.Parse(argv); err != nil {
		// flag.ContinueOnError already printed the error.
		return 1
	}
	env.Dir = filepath.Clean(*dir)

	if *help {
		printUsage(env.Stdout)
		return 0
	}

	rest := fs.Args()
	if len(rest) == 0 {
		printUsage(env.Stderr)
		return 1
	}

	name, sub := rest[0], rest[1:]
	cmd, ok := registry[name]
	if !ok {
		fmt.Fprintf(env.Stderr, "appetite: unknown command %q\n\n", name)
		printUsage(env.Stderr)
		return 1
	}

	if err := cmd.Run(env, sub); err != nil {
		fmt.Fprintf(env.Stderr, "appetite %s: %v\n", name, err)
		return exitCodeFor(err)
	}
	return 0
}

// reorderFlags rewrites args so every flag (and its value, if any)
// precedes every positional argument. Go's stdlib flag.Parse stops
// at the first non-flag token; preprocessing here lets operators
// type the natural mix order from the docs.
//
// Handled cases:
//
//	--flag value pos        → [--flag value pos]
//	--flag=value pos        → [--flag=value pos]
//	--bool pos              → [--bool pos]
//	pos --flag value        → [--flag value pos]
//	pos --bool              → [--bool pos]
//	pos -- --literal        → [pos --literal] (everything after `--` is positional)
//
// Unknown flags (not registered with fs) keep their adjacent
// non-flag token bundled so fs.Parse's error message names both
// pieces — losing the value to the positional bucket would print a
// misleading "flag provided but not defined" without showing what
// the operator was trying to set.
func reorderFlags(fs *flag.FlagSet, args []string) []string {
	var flags, positional []string
	i := 0
	for i < len(args) {
		a := args[i]
		if a == "--" {
			positional = append(positional, args[i+1:]...)
			break
		}
		if !strings.HasPrefix(a, "-") || a == "-" {
			positional = append(positional, a)
			i++
			continue
		}
		flags = append(flags, a)
		name := strings.TrimLeft(a, "-")
		if strings.Contains(name, "=") {
			i++
			continue
		}
		f := fs.Lookup(name)
		isBool := false
		if f != nil {
			if bf, ok := f.Value.(interface{ IsBoolFlag() bool }); ok && bf.IsBoolFlag() {
				isBool = true
			}
		}
		// f == nil: unknown flag. Keep the next non-flag token with
		// it so fs.Parse errors point at the real argv shape.
		consumesValue := !isBool && (f != nil || (i+1 < len(args) && !strings.HasPrefix(args[i+1], "-")))
		if consumesValue && i+1 < len(args) {
			flags = append(flags, args[i+1])
			i += 2
			continue
		}
		i++
	}
	return append(flags, positional...)
}

// printUsage writes the top-level help to w. Command synopses are
// pulled from the registry so adding a command never requires editing
// the help text.
func printUsage(w io.Writer) {
	fmt.Fprintln(w, "appetite — Shape Up workflow engine")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Usage: appetite [--dir <path>] <command> [args]")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Commands:")

	names := make([]string, 0, len(registry))
	for n := range registry {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		fmt.Fprintf(w, "  %-12s %s\n", n, registry[n].Synopsis)
	}
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Run `appetite <command> -h` for command-specific help.")
}
