package cli

import (
	"bytes"
	"errors"
	"flag"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/luuuc/appetite/internal/model"
	"github.com/luuuc/appetite/internal/workflow"
)

// fixtureLoop seeds a fully-progressed loop so the happy-path branches
// of runCut/runHill/runPass/runCooldown all exercise.
func fixtureLoop(t *testing.T) string {
	t.Helper()
	dir := initialized(t)
	must := func(args ...string) {
		t.Helper()
		if code, _, errBuf := runMain(t, dir, args...); code != 0 {
			t.Fatalf("%v: code %d, stderr=%q", args, code, errBuf)
		}
	}
	must("shape", "--new", "csv", "--title", "CSV")
	os.WriteFile(filepath.Join(dir, "pitches/csv.md"), []byte(`---
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
- [ ] **One** — one
`), 0o644)
	must("shape", "--finalize", "csv")
	must("cycle", "new", "w22", "--appetite", "5d")
	must("bet", "csv", "--cycle", "w22", "--appetite", "medium")
	return dir
}

// --- defaults + helpers --------------------------------------------------

func TestDefaultEnv(t *testing.T) {
	env := DefaultEnv()
	if env.Dir != ".appetite" {
		t.Errorf("Dir: %q", env.Dir)
	}
	if env.Stdout != os.Stdout {
		t.Error("Stdout should be os.Stdout")
	}
	if env.Stderr != os.Stderr {
		t.Error("Stderr should be os.Stderr")
	}
}

func TestExitCodeFor(t *testing.T) {
	cases := []struct {
		err  error
		want int
	}{
		{nil, 0},
		{workflow.ErrInvalidTransition, 2},
		{workflow.ErrDoneCriteriaUnmet, 2},
		{workflow.ErrNotFound, 3},
		{errors.New("generic"), 1},
	}
	for _, c := range cases {
		if got := exitCodeFor(c.err); got != c.want {
			t.Errorf("exitCodeFor(%v): got %d want %d", c.err, got, c.want)
		}
	}
}

func TestReorderFlags_AllShapes(t *testing.T) {
	fs := flag.NewFlagSet("t", flag.ContinueOnError)
	fs.String("name", "", "")
	fs.Bool("yes", false, "")

	cases := []struct {
		in   []string
		want []string
	}{
		{[]string{"--name", "x", "pos"}, []string{"--name", "x", "pos"}},
		{[]string{"pos", "--name", "x"}, []string{"--name", "x", "pos"}},
		{[]string{"--name=x", "pos"}, []string{"--name=x", "pos"}},
		{[]string{"--yes", "pos"}, []string{"--yes", "pos"}},
		{[]string{"pos", "--yes"}, []string{"--yes", "pos"}},
		{[]string{"pos", "--"}, []string{"pos"}},
		{[]string{"pos", "--", "--literal"}, []string{"pos", "--literal"}},
		{[]string{"-"}, []string{"-"}},
		{[]string{"--unknown", "value"}, []string{"--unknown", "value"}},
	}
	for _, c := range cases {
		got := reorderFlags(fs, c.in)
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("reorderFlags(%v): got %v want %v", c.in, got, c.want)
		}
	}
}

// --- happy-path executions for sub-commands -------------------------------

func TestCut_HappyPath(t *testing.T) {
	dir := fixtureLoop(t)
	if code, out, _ := runMain(t, dir, "cut", "csv"); code != 0 {
		t.Errorf("code %d", code)
	} else if !strings.Contains(out, "cut: csv") {
		t.Errorf("stdout: %q", out)
	}
}

func TestHill_HappyPath(t *testing.T) {
	dir := fixtureLoop(t)
	if code, _, _ := runMain(t, dir, "cut", "csv"); code != 0 {
		t.Fatal("cut failed")
	}
	if code, _, _ := runMain(t, dir, "hill", "one", "--position", "downhill", "--progress", "70"); code != 0 {
		t.Errorf("position+progress: code %d", code)
	}
	if code, out, _ := runMain(t, dir, "hill", "one", "--progress", "100", "--done"); code != 0 {
		t.Errorf("ship: code %d", code)
	} else if !strings.Contains(out, "shipped") {
		t.Errorf("expected cascade message, got %q", out)
	}
}

func TestPass_HappyPath_NoActiveCycle(t *testing.T) {
	dir := initialized(t)
	runMain(t, dir, "shape", "--new", "dark", "--title", "Dark")
	os.WriteFile(filepath.Join(dir, "pitches/dark.md"), []byte(`---
slug: dark
title: Dark
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
- [ ] **T** — t
`), 0o644)
	runMain(t, dir, "shape", "--finalize", "dark")
	code, out, _ := runMain(t, dir, "pass", "dark", "--reason", "no")
	if code != 0 {
		t.Errorf("code %d", code)
	}
	if !strings.Contains(out, "no active cycle") {
		t.Errorf("expected no-active-cycle message, got %q", out)
	}
}

func TestPass_HappyPath_WithCycle(t *testing.T) {
	dir := fixtureLoop(t)
	// Add a shaped pitch to pass on (the fixture's csv is already bet).
	os.WriteFile(filepath.Join(dir, "pitches/dark.md"), []byte(`---
slug: dark
title: Dark
status: shaped
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
- [ ] **T** — t
`), 0o644)
	code, out, _ := runMain(t, dir, "pass", "dark", "--reason", "no")
	if code != 0 {
		t.Errorf("code %d", code)
	}
	if !strings.Contains(out, "reason recorded") {
		t.Errorf("expected reason-recorded message, got %q", out)
	}
}

func TestStatus_HappyPath(t *testing.T) {
	dir := fixtureLoop(t)
	if code, out, _ := runMain(t, dir, "status"); code != 0 {
		t.Errorf("code %d", code)
	} else if !strings.Contains(out, "Cycle w22") {
		t.Errorf("stdout: %q", out)
	}
}

func TestCooldown_OpenAndClose(t *testing.T) {
	dir := fixtureLoop(t)
	runMain(t, dir, "cut", "csv")
	runMain(t, dir, "hill", "one", "--progress", "100", "--done")
	if code, _, _ := runMain(t, dir, "cooldown", "--days", "2"); code != 0 {
		t.Errorf("open: code %d", code)
	}
	if code, _, _ := runMain(t, dir, "cooldown", "close"); code != 0 {
		t.Errorf("close: code %d", code)
	}
}

func TestCooldown_BadDuration(t *testing.T) {
	dir := initialized(t)
	if code, _, _ := runMain(t, dir, "cooldown", "--days", "-1"); code != 1 {
		t.Errorf("expected 1, got %d", code)
	}
}

// --- renderer pure tests --------------------------------------------------

func TestProgressBar(t *testing.T) {
	cases := []struct {
		pct, w int
		want   string
	}{
		{0, 12, "[          ]"},
		{50, 12, "[=====>    ]"},
		{100, 12, "[==========]"},
		{-10, 12, "[          ]"},
		{200, 12, "[==========]"},
		{1, 12, "[          ]"},
	}
	for _, c := range cases {
		if got := progressBar(c.pct, c.w); got != c.want {
			t.Errorf("progressBar(%d, %d): got %q want %q", c.pct, c.w, got, c.want)
		}
	}
}

func TestTruncate(t *testing.T) {
	cases := map[string]string{
		"short":             "short",
		"this is too long":  "this is to",
		"":                  "",
		"exactlyten":        "exactlyten",
		"unicode—em—dashes": "unicode—em",
	}
	for in, want := range cases {
		if got := truncate(in, 10); got != want {
			t.Errorf("truncate(%q): got %q want %q", in, got, want)
		}
	}
}

func TestRenderStatus_StuckCard(t *testing.T) {
	now := time.Now()
	stuckTime := now.Add(-100 * time.Hour)
	res := workflow.StatusResult{
		Cycle:    model.Cycle{ID: "w22", Duration: "5d", Status: model.CycleStatusBuilding, Started: now},
		DayN:     2,
		DayTotal: 5,
		Bets: []workflow.BetView{
			{
				Bet:       model.Bet{Pitch: "csv", Appetite: model.AppetiteMedium},
				Pitch:     model.Pitch{Slug: "csv"},
				Cards:     []model.Card{{Slug: "a"}},
				CardsDone: 0,
				StuckCards: []model.Card{
					{Slug: "a", Hill: model.HillUphill, Progress: 30, HillUpdatedAt: &stuckTime},
				},
			},
		},
		Passed: []model.Passed{{Pitch: "dark", Reason: "no"}},
		Cooldown: &model.Cooldown{
			Cycle: "w22", Status: model.CooldownActive, Ends: now.Add(24 * time.Hour),
		},
	}
	var buf bytes.Buffer
	renderStatus(&buf, res)
	out := buf.String()
	for _, want := range []string{"Cycle w22", "csv", "stuck on 'a'", "Passed: dark", "Cooldown:"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q. got:\n%s", want, out)
		}
	}
}

// TestSubcommands_BadFlag exercises the fs.Parse-error branch on
// every subcommand by feeding an unregistered flag.
func TestSubcommands_BadFlag(t *testing.T) {
	dir := initialized(t)
	subs := [][]string{
		{"signal", "add", "x", "--bogusflag"},
		{"shape", "--bogusflag", "x"},
		{"cycle", "new", "x", "--bogusflag"},
		{"bet", "x", "--bogusflag"},
		{"pass", "x", "--bogusflag"},
		{"cut", "x", "--bogusflag"},
		{"hill", "x", "--bogusflag"},
		{"status", "--bogusflag"},
		{"cooldown", "--bogusflag"},
		{"cooldown", "close", "--bogusflag"},
		{"init", "--bogusflag"},
	}
	for _, args := range subs {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			if code, _, _ := runMain(t, dir, args...); code == 0 {
				t.Errorf("expected non-zero exit for %v", args)
			}
		})
	}
}

func TestRenderStatus_Empty(t *testing.T) {
	var buf bytes.Buffer
	renderStatus(&buf, workflow.StatusResult{
		Cycle: model.Cycle{ID: "x", Duration: "5d", Status: model.CycleStatusBuilding}, DayN: 1, DayTotal: 5,
	})
	if !strings.Contains(buf.String(), "Cycle x") {
		t.Errorf("expected cycle header, got %q", buf.String())
	}
}
