package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunPassesWhenDocDirAbsent(t *testing.T) {
	// .doc/ is git-ignored, so missing-source is the common case for
	// fresh clones and CI runners. The gate must pass silently — not
	// fail — in that scenario.
	missing := filepath.Join(t.TempDir(), "definitely-not-here")
	var stdout, stderr bytes.Buffer

	code := run([]string{"-doc", missing}, &stdout, &stderr)

	if code != exitOK {
		t.Errorf("code = %d, want %d", code, exitOK)
	}
	if !strings.Contains(stdout.String(), "not present, skipping") {
		t.Errorf("stdout = %q, want skip note", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Errorf("stderr = %q, want empty", stderr.String())
	}
}

func TestRunPassesWhenDocDirPresentAndNoContractsWired(t *testing.T) {
	// Until cards 03-01#2 and #3 wire concrete checks, an existing
	// doc dir is a clean pass — drift can only fire once a contract
	// knows what to compare.
	docDir := t.TempDir()
	var stdout, stderr bytes.Buffer

	code := run([]string{"-doc", docDir}, &stdout, &stderr)

	if code != exitOK {
		t.Errorf("code = %d, want %d", code, exitOK)
	}
	if !strings.Contains(stdout.String(), "contracts match") {
		t.Errorf("stdout = %q, want match note", stdout.String())
	}
}

func TestRunReturnsInternalErrorOnUnknownFlag(t *testing.T) {
	var stdout, stderr bytes.Buffer

	code := run([]string{"-no-such-flag"}, &stdout, &stderr)

	if code != exitInternal {
		t.Errorf("code = %d, want %d", code, exitInternal)
	}
	if !strings.Contains(stderr.String(), "no-such-flag") {
		t.Errorf("stderr = %q, want flag.Parse error mentioning the flag", stderr.String())
	}
}

func TestRunReturnsInternalErrorOnPositionalArg(t *testing.T) {
	var stdout, stderr bytes.Buffer

	code := run([]string{"unexpected"}, &stdout, &stderr)

	if code != exitInternal {
		t.Errorf("code = %d, want %d", code, exitInternal)
	}
	if !strings.Contains(stderr.String(), "no positional") {
		t.Errorf("stderr = %q, want positional-arg complaint", stderr.String())
	}
}

func TestRunReturnsInternalErrorWhenDocPathIsAFile(t *testing.T) {
	// A non-directory at the expected location is operator error,
	// not a graceful skip — the doc path was meant to be a tree.
	file := filepath.Join(t.TempDir(), "not-a-dir")
	if err := os.WriteFile(file, []byte("blocker"), 0o644); err != nil {
		t.Fatalf("seed file: %v", err)
	}
	var stdout, stderr bytes.Buffer

	code := run([]string{"-doc", file}, &stdout, &stderr)

	if code != exitInternal {
		t.Errorf("code = %d, want %d", code, exitInternal)
	}
	if !strings.Contains(stderr.String(), "not a directory") {
		t.Errorf("stderr = %q, want not-a-directory complaint", stderr.String())
	}
}

func TestRunReturnsInternalErrorOnStatFailure(t *testing.T) {
	// A path whose parent is a regular file (not a directory) makes
	// os.Stat return an error that is not ErrNotExist — the "stat
	// failed for unexpected reasons" branch.
	parent := filepath.Join(t.TempDir(), "file-not-dir")
	if err := os.WriteFile(parent, []byte("blocker"), 0o644); err != nil {
		t.Fatalf("seed parent: %v", err)
	}
	var stdout, stderr bytes.Buffer

	code := run([]string{"-doc", filepath.Join(parent, "child")}, &stdout, &stderr)

	if code != exitInternal {
		t.Errorf("code = %d, want %d", code, exitInternal)
	}
	if !strings.Contains(stderr.String(), "stat") {
		t.Errorf("stderr = %q, want stat-error message", stderr.String())
	}
}
