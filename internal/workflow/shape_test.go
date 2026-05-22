package workflow

import (
	"errors"
	"strings"
	"testing"

	"github.com/luuuc/appetite/internal/model"
)

func TestShapeNew_HappyPath(t *testing.T) {
	s := newStore(t)
	res, err := ShapeNew(s, ShapeNewRequest{Slug: "csv", Title: "CSV", Now: fixedNow})
	if err != nil {
		t.Fatalf("ShapeNew: %v", err)
	}
	if res.Pitch.Status != model.PitchStatusShaping {
		t.Errorf("status: got %q want shaping", res.Pitch.Status)
	}
	for _, h := range requiredIngredients {
		if !strings.Contains(res.Pitch.Body, "## "+h) {
			t.Errorf("body missing heading %q", h)
		}
	}
	if !strings.Contains(res.Pitch.Body, "## Scope") {
		t.Error("body missing Scope")
	}
}

func TestShapeNew_Errors(t *testing.T) {
	s := newStore(t)
	if _, err := ShapeNew(s, ShapeNewRequest{Title: "x"}); err == nil {
		t.Error("missing slug should error")
	}
	if _, err := ShapeNew(s, ShapeNewRequest{Slug: "x"}); err == nil {
		t.Error("missing title should error")
	}
	if _, err := ShapeNew(s, ShapeNewRequest{Slug: "a", Title: "A"}); err != nil {
		t.Fatalf("first call: %v", err)
	}
	if _, err := ShapeNew(s, ShapeNewRequest{Slug: "a", Title: "A"}); err == nil {
		t.Error("duplicate pitch should error")
	}
}

func TestShapeFrom_HappyPath(t *testing.T) {
	s := newStore(t)
	writeSignalDirect(t, s, model.Signal{
		Slug: "export-request", Source: model.SignalSourceOperator,
		Captured: fixedNow, Body: "Operators want CSV.\n",
	})

	res, err := ShapeFrom(s, ShapeFromRequest{
		SignalPath: "signals/raw/export-request.md",
		Now:        fixedNow,
	})
	if err != nil {
		t.Fatalf("ShapeFrom: %v", err)
	}
	if res.Pitch.Slug != "export-request" {
		t.Errorf("derived slug: got %q", res.Pitch.Slug)
	}
	if res.Pitch.Title != "Export Request" {
		t.Errorf("derived title: got %q", res.Pitch.Title)
	}
	if !strings.Contains(res.Pitch.Body, "Operators want CSV.") {
		t.Error("signal body not preserved in Problem section")
	}
	if len(res.Pitch.ShapedFrom) != 1 || res.Pitch.ShapedFrom[0] != "export-request" {
		t.Errorf("ShapedFrom: got %v", res.Pitch.ShapedFrom)
	}
}

func TestShapeFrom_Overrides(t *testing.T) {
	s := newStore(t)
	writeSignalDirect(t, s, model.Signal{Slug: "src", Captured: fixedNow, Body: "x"})
	res, err := ShapeFrom(s, ShapeFromRequest{
		SignalPath: "signals/raw/src.md", Slug: "renamed", Title: "Renamed",
	})
	if err != nil {
		t.Fatalf("ShapeFrom: %v", err)
	}
	if res.Pitch.Slug != "renamed" || res.Pitch.Title != "Renamed" {
		t.Errorf("overrides: got slug=%q title=%q", res.Pitch.Slug, res.Pitch.Title)
	}
}

func TestShapeFrom_Errors(t *testing.T) {
	s := newStore(t)
	if _, err := ShapeFrom(s, ShapeFromRequest{}); err == nil {
		t.Error("missing signal path should error")
	}
	if _, err := ShapeFrom(s, ShapeFromRequest{SignalPath: "signals/raw/nope.md"}); err == nil {
		t.Error("missing signal should error")
	}
	writeSignalDirect(t, s, model.Signal{Slug: "src", Captured: fixedNow, Body: "x"})
	if _, err := ShapeFrom(s, ShapeFromRequest{SignalPath: "signals/raw/src.md"}); err != nil {
		t.Fatalf("first ShapeFrom: %v", err)
	}
	if _, err := ShapeFrom(s, ShapeFromRequest{SignalPath: "signals/raw/src.md"}); err == nil {
		t.Error("duplicate pitch should error")
	}
}

func TestShapeFinalize_HappyPath(t *testing.T) {
	s := newStore(t)
	writePitchDirect(t, s, model.Pitch{
		Slug: "csv", Title: "CSV", Status: model.PitchStatusShaping, Body: shapedBody,
	})
	res, err := ShapeFinalize(s, ShapeFinalizeRequest{Slug: "csv", Now: fixedNow})
	if err != nil {
		t.Fatalf("ShapeFinalize: %v", err)
	}
	if res.Pitch.Status != model.PitchStatusShaped {
		t.Errorf("status: got %q", res.Pitch.Status)
	}
	if res.Pitch.ShapedAt == nil || !res.Pitch.ShapedAt.Equal(fixedNow) {
		t.Errorf("ShapedAt: got %v", res.Pitch.ShapedAt)
	}
}

func TestShapeFinalize_Errors(t *testing.T) {
	tests := []struct {
		name    string
		seed    *model.Pitch
		slug    string
		wantErr error
	}{
		{"missing slug arg", nil, "", nil},
		{"missing pitch", nil, "nope", ErrNotFound},
		{
			"already shaped",
			&model.Pitch{Slug: "shaped", Status: model.PitchStatusShaped, Body: shapedBody},
			"shaped", ErrInvalidTransition,
		},
		{
			"missing ingredients",
			&model.Pitch{Slug: "incomplete", Status: model.PitchStatusShaping, Body: "## Problem\nx\n## Scope\n- [ ] **A** — x\n"},
			"incomplete", ErrInvalidTransition,
		},
		{
			"no scope cards",
			&model.Pitch{Slug: "noscope", Status: model.PitchStatusShaping, Body: `## Problem
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
`},
			"noscope", ErrInvalidTransition,
		},
		{
			"malformed scope line",
			&model.Pitch{Slug: "malformed", Status: model.PitchStatusShaping, Body: `## Problem
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
- [ ] **A** - hyphen not em-dash
`},
			"malformed", ErrInvalidTransition,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := newStore(t)
			if tc.seed != nil {
				writePitchDirect(t, s, *tc.seed)
			}
			_, err := ShapeFinalize(s, ShapeFinalizeRequest{Slug: tc.slug})
			if err == nil {
				t.Fatal("expected error")
			}
			if tc.wantErr != nil && !errors.Is(err, tc.wantErr) {
				t.Errorf("error type: got %v want errors.Is %v", err, tc.wantErr)
			}
		})
	}
}

func TestTitleize(t *testing.T) {
	cases := map[string]string{
		"":           "",
		"csv-export": "Csv Export",
		"one":        "One",
		"a-b-c":      "A B C",
		"-leading":   " Leading",
	}
	for in, want := range cases {
		if got := titleize(in); got != want {
			t.Errorf("titleize(%q): got %q want %q", in, got, want)
		}
	}
}

func TestMissingIngredients(t *testing.T) {
	if missing := missingIngredients(shapedBody); len(missing) != 0 {
		t.Errorf("shapedBody should have no missing: got %v", missing)
	}
	body := "## Problem\n\n## Solution\n"
	got := missingIngredients(body)
	want := []string{"Appetite", "Rabbit holes", "No-gos"}
	if len(got) != len(want) {
		t.Fatalf("got %v want %v", got, want)
	}
	for i, w := range want {
		if got[i] != w {
			t.Errorf("missing[%d]: got %q want %q", i, got[i], w)
		}
	}
}
