package workflow

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/luuuc/appetite/internal/model"
)

func TestSignalAdd_HappyPath(t *testing.T) {
	s := newStore(t)
	res, err := SignalAdd(s, SignalAddRequest{
		Body:   "Hello world",
		Source: model.SignalSourceBeacon,
		Tags:   []string{"a", "b"},
		Now:    fixedNow,
	})
	if err != nil {
		t.Fatalf("SignalAdd: %v", err)
	}
	if res.Signal.Source != model.SignalSourceBeacon {
		t.Errorf("source: got %q", res.Signal.Source)
	}
	if !res.Signal.Captured.Equal(fixedNow) {
		t.Errorf("captured: got %v want %v", res.Signal.Captured, fixedNow)
	}
	if res.Path == "" {
		t.Error("empty path")
	}
}

func TestSignalAdd_DefaultsAndSlug(t *testing.T) {
	s := newStore(t)
	res, err := SignalAdd(s, SignalAddRequest{
		Body: "PaymentGateway TimeoutError spotted",
		Now:  fixedNow,
	})
	if err != nil {
		t.Fatalf("SignalAdd: %v", err)
	}
	if res.Signal.Source != model.SignalSourceOperator {
		t.Errorf("default source: got %q want operator", res.Signal.Source)
	}
	want := "2026-05-22-paymentgateway-timeouterror-spotted"
	if !strings.Contains(res.Path, want) {
		t.Errorf("auto-slug: got %q want substring %q", res.Path, want)
	}
}

func TestSignalAdd_CustomSlug(t *testing.T) {
	s := newStore(t)
	res, err := SignalAdd(s, SignalAddRequest{
		Body: "x",
		Slug: "explicit-slug",
		Now:  fixedNow,
	})
	if err != nil {
		t.Fatalf("SignalAdd: %v", err)
	}
	if res.Path != "signals/raw/explicit-slug.md" {
		t.Errorf("custom slug path: got %q", res.Path)
	}
}

func TestSignalAdd_Errors(t *testing.T) {
	s := newStore(t)
	tests := []struct {
		name string
		req  SignalAddRequest
	}{
		{"empty body", SignalAddRequest{Body: ""}},
		{"whitespace body", SignalAddRequest{Body: "   "}},
		{"invalid source", SignalAddRequest{Body: "x", Source: model.SignalSource("bogus")}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := SignalAdd(s, tc.req); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestDefaultNow(t *testing.T) {
	t1 := defaultNow(time.Time{})
	if t1.IsZero() {
		t.Error("zero in → expected non-zero out")
	}
	want := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	if got := defaultNow(want); !got.Equal(want) {
		t.Errorf("explicit time: got %v want %v", got, want)
	}
}

func TestSlugifyWords(t *testing.T) {
	tests := []struct {
		in   string
		n    int
		want string
	}{
		{"", 5, ""},
		{"hello world", 5, "hello-world"},
		{"   spaced   words   ", 5, "spaced-words"},
		{"too many words here please cut", 3, "too-many-words"},
		{"emoji 🎉 only 🚀", 5, "emoji-only"},
		{"!!!!", 5, ""},
		{"MIXED Case 123", 5, "mixed-case-123"},
	}
	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			if got := slugifyWords(tc.in, tc.n); got != tc.want {
				t.Errorf("slugifyWords(%q, %d): got %q want %q", tc.in, tc.n, got, tc.want)
			}
		})
	}
}

func TestEnsureTrailingNewline(t *testing.T) {
	cases := map[string]string{
		"":     "",
		"x":    "x\n",
		"x\n":  "x\n",
		"x\nz": "x\nz\n",
	}
	for in, want := range cases {
		if got := ensureTrailingNewline(in); got != want {
			t.Errorf("ensureTrailingNewline(%q): got %q want %q", in, got, want)
		}
	}
}

func TestAutoSignalSlug_EmojiOnlyFallsBackToDate(t *testing.T) {
	slug := autoSignalSlug(fixedNow, "🎉🚀")
	if slug != "2026-05-22" {
		t.Errorf("emoji-only body: got %q want date-only", slug)
	}
}

func TestSignalAdd_WrittenEntityRoundTrips(t *testing.T) {
	s := newStore(t)
	res, err := SignalAdd(s, SignalAddRequest{Body: "round trip", Now: fixedNow})
	if err != nil {
		t.Fatalf("SignalAdd: %v", err)
	}
	read, err := s.Read(context.Background(), res.Path)
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	sig, ok := read.(model.Signal)
	if !ok {
		t.Fatalf("read as Signal: got %T", read)
	}
	if !strings.Contains(sig.Body, "round trip") {
		t.Errorf("body lost in round trip: %q", sig.Body)
	}
}
