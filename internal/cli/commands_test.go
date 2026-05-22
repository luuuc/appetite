package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// runMain is a one-liner helper that invokes Main with a fresh
// buffered Env rooted at dir and returns the resulting exit code
// plus captured stdout/stderr. Tests share this helper instead of
// inlining the bytes.Buffer dance.
func runMain(t *testing.T, dir string, argv ...string) (int, string, string) {
	t.Helper()
	env, out, errBuf := testEnv(dir)
	code := Main(argv, env)
	return code, out.String(), errBuf.String()
}

// initialized returns a temp directory pre-initialized with the
// .appetite/ skeleton — every command except `init` needs one.
func initialized(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), ".appetite")
	if code, _, _ := runMain(t, dir, "init"); code != 0 {
		t.Fatalf("init: code %d", code)
	}
	return dir
}

// --- signal ----------------------------------------------------------------

func TestSignal_AddHappyPath(t *testing.T) {
	dir := initialized(t)
	code, out, _ := runMain(t, dir, "signal", "add", "Test signal body", "--source", "beacon", "--tags", "a,b")
	if code != 0 {
		t.Fatalf("code %d", code)
	}
	if !strings.Contains(out, "signal: wrote") {
		t.Errorf("stdout: %q", out)
	}
}

func TestSignal_AddFlagsBeforeBody(t *testing.T) {
	dir := initialized(t)
	code, _, _ := runMain(t, dir, "signal", "add", "--source", "beacon", "Body after flags")
	if code != 0 {
		t.Fatalf("code %d", code)
	}
}

func TestSignal_NoSubcommand(t *testing.T) {
	dir := initialized(t)
	if code, _, _ := runMain(t, dir, "signal"); code != 1 {
		t.Errorf("expected 1, got %d", code)
	}
}

func TestSignal_UnknownSubcommand(t *testing.T) {
	dir := initialized(t)
	if code, _, _ := runMain(t, dir, "signal", "delete", "x"); code != 1 {
		t.Errorf("expected 1, got %d", code)
	}
}

func TestSignal_AddMissingBody(t *testing.T) {
	dir := initialized(t)
	if code, _, _ := runMain(t, dir, "signal", "add"); code != 1 {
		t.Errorf("expected 1, got %d", code)
	}
}

// --- shape -----------------------------------------------------------------

func TestShape_NoMode(t *testing.T) {
	dir := initialized(t)
	if code, _, _ := runMain(t, dir, "shape"); code != 1 {
		t.Errorf("expected 1, got %d", code)
	}
}

func TestShape_NewHappy(t *testing.T) {
	dir := initialized(t)
	if code, _, _ := runMain(t, dir, "shape", "--new", "csv", "--title", "CSV"); code != 0 {
		t.Errorf("code %d", code)
	}
}

func TestShape_FromHappy(t *testing.T) {
	dir := initialized(t)
	runMain(t, dir, "signal", "add", "Body for from", "--slug", "src")
	if code, _, _ := runMain(t, dir, "shape", "--from", "signals/raw/src.md"); code != 0 {
		t.Errorf("code %d", code)
	}
}

func TestShape_FinalizeMissing(t *testing.T) {
	dir := initialized(t)
	if code, _, _ := runMain(t, dir, "shape", "--finalize", "nope"); code != 3 {
		t.Errorf("expected 3, got %d", code)
	}
}

func TestShape_NewDefaultsTitleToSlug(t *testing.T) {
	dir := initialized(t)
	if code, _, _ := runMain(t, dir, "shape", "--new", "csv"); code != 0 {
		t.Errorf("code %d", code)
	}
}

// --- cycle / bet / pass / cut / hill -------------------------------------

func TestCycle_NewHappy(t *testing.T) {
	dir := initialized(t)
	if code, _, _ := runMain(t, dir, "cycle", "new", "w22", "--appetite", "5d"); code != 0 {
		t.Errorf("code %d", code)
	}
}

func TestCycle_NewMissingAppetite(t *testing.T) {
	dir := initialized(t)
	if code, _, _ := runMain(t, dir, "cycle", "new", "w22"); code != 1 {
		t.Errorf("expected 1, got %d", code)
	}
}

func TestCycle_NewMissingId(t *testing.T) {
	dir := initialized(t)
	if code, _, _ := runMain(t, dir, "cycle", "new", "--appetite", "5d"); code != 1 {
		t.Errorf("expected 1, got %d", code)
	}
}

func TestCycle_NoSubcommand(t *testing.T) {
	dir := initialized(t)
	if code, _, _ := runMain(t, dir, "cycle"); code != 1 {
		t.Errorf("expected 1, got %d", code)
	}
}

func TestCycle_UnknownSubcommand(t *testing.T) {
	dir := initialized(t)
	if code, _, _ := runMain(t, dir, "cycle", "delete", "x"); code != 1 {
		t.Errorf("expected 1, got %d", code)
	}
}

func TestBet_MissingFlags(t *testing.T) {
	dir := initialized(t)
	if code, _, _ := runMain(t, dir, "bet", "csv"); code != 1 {
		t.Errorf("expected 1, got %d", code)
	}
	if code, _, _ := runMain(t, dir, "bet", "csv", "--cycle", "w22"); code != 1 {
		t.Errorf("expected 1, got %d", code)
	}
	if code, _, _ := runMain(t, dir, "bet"); code != 1 {
		t.Errorf("expected 1, got %d", code)
	}
}

func TestPass_MissingReason(t *testing.T) {
	dir := initialized(t)
	if code, _, _ := runMain(t, dir, "pass", "csv"); code != 1 {
		t.Errorf("expected 1, got %d", code)
	}
}

func TestPass_MissingPitchArg(t *testing.T) {
	dir := initialized(t)
	if code, _, _ := runMain(t, dir, "pass", "--reason", "x"); code != 1 {
		t.Errorf("expected 1, got %d", code)
	}
}

func TestCut_MissingArg(t *testing.T) {
	dir := initialized(t)
	if code, _, _ := runMain(t, dir, "cut"); code != 1 {
		t.Errorf("expected 1, got %d", code)
	}
}

func TestHill_MissingArg(t *testing.T) {
	dir := initialized(t)
	if code, _, _ := runMain(t, dir, "hill"); code != 1 {
		t.Errorf("expected 1, got %d", code)
	}
}

// --- status / cooldown ---------------------------------------------------

func TestStatus_NoCycle(t *testing.T) {
	dir := initialized(t)
	if code, _, _ := runMain(t, dir, "status"); code != 3 {
		t.Errorf("expected 3, got %d", code)
	}
}

func TestStatus_StrayArg(t *testing.T) {
	dir := initialized(t)
	if code, _, _ := runMain(t, dir, "status", "stray"); code != 1 {
		t.Errorf("expected 1, got %d", code)
	}
}

func TestCooldown_NegativeDays(t *testing.T) {
	dir := initialized(t)
	if code, _, _ := runMain(t, dir, "cooldown", "--days", "0"); code != 1 {
		t.Errorf("expected 1, got %d", code)
	}
}

// --- full-loop integration via Main ---------------------------------------

func TestMain_FullLoop(t *testing.T) {
	dir := initialized(t)

	must := func(label string, args ...string) {
		t.Helper()
		code, _, errBuf := runMain(t, dir, args...)
		if code != 0 {
			t.Fatalf("%s: code %d, stderr=%q", label, code, errBuf)
		}
	}

	must("signal", "signal", "add", "Need CSV", "--slug", "csv-signal")
	must("shape new", "shape", "--new", "csv", "--title", "CSV")

	// Inject a complete pitch body so finalize passes (the template
	// from --new is empty-bodied).
	path := filepath.Join(dir, "pitches/csv.md")
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
- [ ] **Card two** — two
`
	writeFile(t, path, completePitch)

	must("finalize", "shape", "--finalize", "csv")
	must("cycle new", "cycle", "new", "w22", "--appetite", "5d")
	must("bet", "bet", "csv", "--cycle", "w22", "--appetite", "medium")
	must("cut", "cut", "csv")
	must("hill 1", "hill", "card-one", "--progress", "100", "--done")
	must("hill 2", "hill", "card-two", "--progress", "100", "--done")
	must("status", "status")
	must("cooldown open", "cooldown")
	must("cooldown close", "cooldown", "close")
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
