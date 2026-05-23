// Command check-doc-drift asserts that contract-bearing documents in
// .doc/definition/ match the shipped binary. The skeleton wires up
// the CLI, exit codes, and the graceful no-op for when .doc/ is
// absent (it is git-ignored as private workspace, so fresh clones
// and CI environments must pass through without firing the gate).
// Concrete checks (CLI flag surface, MCP tool list) land in
// follow-up cards of pitch 03-01.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
)

const (
	exitOK       = 0
	exitDrift    = 1
	exitInternal = 2
)

// defaultDocDir is the conventional location of the definition docs
// relative to the repo root. Overridable via the `-doc` flag so tests
// (and the operator running `make ci` from a sub-directory) can point
// at a different tree.
const defaultDocDir = ".doc/definition"

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// run is the testable entry point. It returns the exit code to pass
// to os.Exit and writes user-visible output to stdout/stderr. The
// three exit codes mirror what the pitch pinned: 0 = no drift, 1 =
// drift found, 2 = internal error (file/flag/spawn failure).
func run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("check-doc-drift", flag.ContinueOnError)
	fs.SetOutput(stderr)
	docDir := fs.String("doc", defaultDocDir, "directory containing definition docs")
	if err := fs.Parse(args); err != nil {
		return exitInternal
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(stderr, "check-doc-drift: takes no positional arguments")
		return exitInternal
	}

	switch present, err := dirPresent(*docDir); {
	case err != nil:
		fmt.Fprintf(stderr, "check-doc-drift: stat %s: %v\n", *docDir, err)
		return exitInternal
	case !present:
		// .doc/ is git-ignored; absent in fresh clones and CI. Pass
		// silently rather than failing — the gate exists for the
		// operator's local `make ci`, not for downstream consumers.
		fmt.Fprintf(stdout, "check-doc-drift: %s not present, skipping\n", *docDir)
		return exitOK
	}

	// Cards 03-01#2 (CLI flag surface) and 03-01#3 (MCP tool list)
	// add their contracts here. With no contracts wired the skeleton
	// reports a clean pass — drift can only be found once a check
	// knows what to compare against.
	fmt.Fprintln(stdout, "check-doc-drift: contracts match")
	return exitOK
}

// dirPresent reports whether path exists and is a directory. Returns
// (false, nil) for "not present" so the caller can distinguish a
// graceful skip from a stat error worth surfacing.
func dirPresent(path string) (bool, error) {
	info, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if !info.IsDir() {
		return false, fmt.Errorf("%s is not a directory", path)
	}
	return true, nil
}
