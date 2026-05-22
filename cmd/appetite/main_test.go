package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestMain re-enters the test binary as the real `appetite` CLI
// when the APPETITE_SUBPROCESS environment variable is set. This
// lets subprocess tests below exec os.Args[0] (the test binary) and
// have it call main() — which gives main.go coverage attribution
// without needing to import the main package elsewhere.
func TestMain(m *testing.M) {
	if os.Getenv("APPETITE_SUBPROCESS") == "1" {
		// APPETITE_SUBPROCESS_ARGS is a NUL-separated argv list set
		// by the subprocess helper below.
		raw := os.Getenv("APPETITE_SUBPROCESS_ARGS")
		os.Args = append([]string{"appetite"}, splitArgs(raw)...)
		main()
		// main calls os.Exit; this line is unreachable but appeases
		// the compiler if main ever falls through.
		return
	}
	os.Exit(m.Run())
}

func splitArgs(s string) []string {
	if s == "" {
		return nil
	}
	return strings.Split(s, "\x1f")
}

// runBinary execs the test binary as a subprocess running main(),
// returning stdout, stderr, and the exit code. Tests share this
// helper instead of inlining the exec.Command dance.
func runBinary(t *testing.T, dir string, args ...string) (stdout, stderr string, code int) {
	t.Helper()
	cmd := exec.Command(os.Args[0])
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	cmd.Env = append(os.Environ(),
		"APPETITE_SUBPROCESS=1",
		"APPETITE_SUBPROCESS_ARGS="+strings.Join(append([]string{"--dir", dir}, args...), "\x1f"),
	)
	err := cmd.Run()
	stdout = outBuf.String()
	stderr = errBuf.String()
	if err == nil {
		code = 0
	} else if exitErr, ok := err.(*exec.ExitError); ok {
		code = exitErr.ExitCode()
	} else {
		t.Fatalf("subprocess: %v", err)
	}
	return stdout, stderr, code
}

func TestSubprocess_Version(t *testing.T) {
	dir := t.TempDir()
	stdout, _, code := runBinary(t, dir, "version")
	if code != 0 {
		t.Errorf("code %d", code)
	}
	if strings.TrimSpace(stdout) == "" {
		t.Errorf("expected version output, got empty")
	}
}

func TestSubprocess_Init(t *testing.T) {
	dir := filepath.Join(t.TempDir(), ".appetite")
	_, _, code := runBinary(t, dir, "init")
	if code != 0 {
		t.Errorf("code %d", code)
	}
	if _, err := os.Stat(filepath.Join(dir, "signals/raw")); err != nil {
		t.Errorf("skeleton missing: %v", err)
	}
}

func TestSubprocess_UnknownCommand(t *testing.T) {
	dir := t.TempDir()
	_, stderr, code := runBinary(t, dir, "blarg")
	if code != 1 {
		t.Errorf("expected 1, got %d", code)
	}
	if !strings.Contains(stderr, "unknown command") {
		t.Errorf("stderr: %q", stderr)
	}
}

func TestSubprocess_FullLoop(t *testing.T) {
	dir := filepath.Join(t.TempDir(), ".appetite")

	must := func(args ...string) {
		t.Helper()
		_, errBuf, code := runBinary(t, dir, args...)
		if code != 0 {
			t.Fatalf("%v: code %d stderr=%q", args, code, errBuf)
		}
	}

	must("init")
	must("signal", "add", "Need exports")
	must("shape", "--new", "csv", "--title", "CSV")

	// Inject a complete pitch body for finalize.
	completePitch := `---
slug: csv
title: CSV
appetite: medium
status: shaping
---
## Problem
x
## Appetite
x
## Solution
x
## Rabbit holes
x
## No-gos
x
## Scope
- [ ] **Card one** — one
`
	if err := os.WriteFile(filepath.Join(dir, "pitches/csv.md"), []byte(completePitch), 0o644); err != nil {
		t.Fatal(err)
	}

	must("shape", "--finalize", "csv")
	must("cycle", "new", "w22", "--appetite", "5d")
	must("bet", "csv", "--cycle", "w22", "--appetite", "medium")
	must("cut", "csv")
	must("hill", "card-one", "--progress", "100", "--done")
	must("status")
	must("cooldown")
	must("cooldown", "close")
}
