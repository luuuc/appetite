// Command check-doc-drift asserts that contract-bearing documents in
// .doc/definition/ match the shipped binary. The skeleton wires up
// the CLI, exit codes, and the graceful no-op for when .doc/ is
// absent (it is git-ignored as private workspace, so fresh clones
// and CI environments must pass through without firing the gate).
// Concrete checks (CLI flag surface, MCP tool list) land in
// follow-up cards of pitch 03-01.
package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
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

// defaultBinaryPath is where `make build` lands the appetite binary
// that the checker introspects via `-h`. Overridable via `-binary`
// for the rare case the operator runs the gate against a non-default
// build output.
const defaultBinaryPath = "./bin/appetite"

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
	binary := fs.String("binary", defaultBinaryPath, "path to the appetite binary under test")
	mcpSrc := fs.String("mcp-source", defaultMCPSourceFile, "path to the Go source file holding the MCP dispatch table")
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

	docPath := filepath.Join(*docDir, canonicalDocFile)
	docBytes, err := os.ReadFile(docPath)
	if err != nil {
		fmt.Fprintf(stderr, "check-doc-drift: open %s: %v\n", docPath, err)
		return exitInternal
	}

	cliDrifts, err := checkCLIFlagSurface(docPath, canonicalDocFile, *binary, bytes.NewReader(docBytes))
	if err != nil {
		fmt.Fprintf(stderr, "check-doc-drift: %v\n", err)
		return exitInternal
	}

	src, err := os.Open(*mcpSrc)
	if err != nil {
		fmt.Fprintf(stderr, "check-doc-drift: open %s: %v\n", *mcpSrc, err)
		return exitInternal
	}
	defer src.Close()
	mcpDrifts, err := checkMCPToolList(canonicalDocFile, bytes.NewReader(docBytes), *mcpSrc, src)
	if err != nil {
		fmt.Fprintf(stderr, "check-doc-drift: %v\n", err)
		return exitInternal
	}

	all := append(cliDrifts, mcpDrifts...)
	if len(all) == 0 {
		fmt.Fprintln(stdout, "check-doc-drift: contracts match")
		return exitOK
	}
	if len(cliDrifts) > 0 {
		fmt.Fprintln(stderr, "doc-drift: CLI flag surface diverged:")
		for _, d := range cliDrifts {
			fmt.Fprintln(stderr, "  "+d)
		}
	}
	if len(mcpDrifts) > 0 {
		fmt.Fprintln(stderr, "doc-drift: MCP tool list diverged:")
		for _, d := range mcpDrifts {
			fmt.Fprintln(stderr, "  "+d)
		}
	}
	return exitDrift
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
