package workflow

import "testing"

func TestParseScope(t *testing.T) {
	tests := []struct {
		name           string
		body           string
		wantCards      int
		wantInvalid    int
		wantFirstTitle string
	}{
		{
			name:      "empty body",
			body:      "",
			wantCards: 0,
		},
		{
			name:      "no scope section",
			body:      "## Problem\nfoo\n",
			wantCards: 0,
		},
		{
			name: "single card",
			body: "## Scope\n- [ ] **Foo** — bar\n",
			wantCards: 1, wantFirstTitle: "Foo",
		},
		{
			name: "multiple cards",
			body: "## Scope\n- [ ] **A** — one\n- [ ] **B** — two\n",
			wantCards: 2,
		},
		{
			name:        "hyphen instead of em-dash is invalid",
			body:        "## Scope\n- [ ] **Foo** - bar\n",
			wantInvalid: 1,
		},
		{
			name:      "scope section ends at next heading",
			body:      "## Scope\n- [ ] **A** — x\n## Other\n- [ ] **Z** — ignored\n",
			wantCards: 1,
		},
		{
			name:      "lines that don't look like cards are ignored",
			body:      "## Scope\nSome prose.\n- [ ] **A** — x\nMore prose.\n",
			wantCards: 1,
		},
		{
			name:        "checkbox-shaped but malformed line counts as invalid",
			body:        "## Scope\n- [x] **A** — done\n- [ ] **B** — ok\n",
			wantInvalid: 1, wantCards: 1,
		},
		{
			name:      "CRLF line endings",
			body:      "## Scope\r\n- [ ] **A** — x\r\n",
			wantCards: 1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cards, invalid := ParseScope(tc.body)
			if len(cards) != tc.wantCards {
				t.Errorf("cards: got %d want %d (%v)", len(cards), tc.wantCards, cards)
			}
			if len(invalid) != tc.wantInvalid {
				t.Errorf("invalid: got %d want %d (%v)", len(invalid), tc.wantInvalid, invalid)
			}
			if tc.wantFirstTitle != "" && len(cards) > 0 && cards[0].Title != tc.wantFirstTitle {
				t.Errorf("first title: got %q want %q", cards[0].Title, tc.wantFirstTitle)
			}
		})
	}
}
