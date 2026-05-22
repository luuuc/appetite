package workflow

import (
	"errors"
	"testing"
	"time"

	"github.com/luuuc/appetite/internal/model"
)

func TestCycleNew_HappyPath(t *testing.T) {
	s := newStore(t)
	res, err := CycleNew(s, CycleNewRequest{ID: "w22", Duration: "5d", Now: fixedNow})
	if err != nil {
		t.Fatalf("CycleNew: %v", err)
	}
	if res.Cycle.Status != model.CycleStatusBuilding {
		t.Errorf("status: got %q", res.Cycle.Status)
	}
	wantEnds := fixedNow.Add(5 * 24 * time.Hour)
	if !res.Cycle.Ends.Equal(wantEnds) {
		t.Errorf("ends: got %v want %v", res.Cycle.Ends, wantEnds)
	}
}

func TestCycleNew_Errors(t *testing.T) {
	tests := []struct {
		name string
		req  CycleNewRequest
		seed *model.Cycle
		want error
	}{
		{"missing id", CycleNewRequest{Duration: "5d"}, nil, nil},
		{"bad duration", CycleNewRequest{ID: "w1", Duration: "5x"}, nil, nil},
		{"duplicate id", CycleNewRequest{ID: "w1", Duration: "5d"}, &model.Cycle{ID: "w1", Duration: "5d", Status: model.CycleStatusClosed}, nil},
		{"open cycle exists", CycleNewRequest{ID: "w2", Duration: "5d"}, &model.Cycle{ID: "w1", Duration: "5d", Status: model.CycleStatusBuilding}, ErrInvalidTransition},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := newStore(t)
			if tc.seed != nil {
				writeCycleDirect(t, s, *tc.seed)
			}
			_, err := CycleNew(s, tc.req)
			if err == nil {
				t.Fatal("expected error")
			}
			if tc.want != nil && !errors.Is(err, tc.want) {
				t.Errorf("error: got %v want errors.Is %v", err, tc.want)
			}
		})
	}
}

func TestParseDuration(t *testing.T) {
	good := map[string]time.Duration{
		"1d":  24 * time.Hour,
		"5d":  5 * 24 * time.Hour,
		"2w":  14 * 24 * time.Hour,
		"3h":  3 * time.Hour,
		"12h": 12 * time.Hour,
	}
	for in, want := range good {
		got, err := ParseDuration(in)
		if err != nil {
			t.Errorf("ParseDuration(%q): unexpected error %v", in, err)
			continue
		}
		if got != want {
			t.Errorf("ParseDuration(%q): got %v want %v", in, got, want)
		}
	}
	bad := []string{"", "x", "5", "0d", "-1d", "5y", "abc"}
	for _, in := range bad {
		if _, err := ParseDuration(in); err == nil {
			t.Errorf("ParseDuration(%q): expected error", in)
		}
	}
}

func TestFindActiveCycle(t *testing.T) {
	s := newStore(t)
	got, err := findActiveCycle(s)
	if err != nil {
		t.Fatalf("empty store: %v", err)
	}
	if got != nil {
		t.Error("empty store should return nil")
	}

	writeCycleDirect(t, s, model.Cycle{ID: "closed", Duration: "5d", Status: model.CycleStatusClosed})
	if got, _ := findActiveCycle(s); got != nil {
		t.Errorf("only closed cycles → got %v want nil", got)
	}

	writeCycleDirect(t, s, model.Cycle{ID: "open", Duration: "5d", Status: model.CycleStatusBuilding})
	got, err = findActiveCycle(s)
	if err != nil {
		t.Fatalf("one open: %v", err)
	}
	if got == nil || got.ID != "open" {
		t.Errorf("one open → got %v", got)
	}

	writeCycleDirect(t, s, model.Cycle{ID: "open2", Duration: "5d", Status: model.CycleStatusShipping})
	if _, err := findActiveCycle(s); err == nil {
		t.Error("two open cycles should error")
	}
}
