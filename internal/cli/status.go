package cli

import (
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/luuuc/appetite/internal/workflow"
)

func init() {
	register(Command{
		Name:     "status",
		Synopsis: "render the current cycle (bets, hill, passed, stuck warnings)",
		Run:      runStatus,
	})
}

func runStatus(env Env, args []string) error {
	fs := flag.NewFlagSet("status", flag.ContinueOnError)
	fs.SetOutput(env.Stderr)
	cycle := fs.String("cycle", "", "cycle id (optional; defaults to the active cycle)")
	if err := fs.Parse(reorderFlags(fs, args)); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("status: takes no positional arguments")
	}

	res, err := workflow.Status(openStore(env), workflow.StatusRequest{Cycle: *cycle})
	if err != nil {
		return err
	}
	renderStatus(env.Stdout, res)
	return nil
}

// renderStatus draws the 80-column text view described in
// .doc/definition/07-mcp-and-cli.md. The format is fixed: 10-char
// progress bar, 18-char pitch column, 9-char appetite, then the
// cards-down summary and any stuck warning.
func renderStatus(w io.Writer, r workflow.StatusResult) {
	fmt.Fprintf(w, "Cycle %s (%s, day %d/%d, status: %s)\n",
		r.Cycle.ID, r.Cycle.Duration, r.DayN, r.DayTotal, r.Cycle.Status)
	fmt.Fprintln(w)

	if len(r.Bets) > 0 {
		fmt.Fprintln(w, "Bets:")
		for _, b := range r.Bets {
			ratio := 0
			if total := len(b.Cards); total > 0 {
				ratio = (100 * b.CardsDone) / total
			}
			line := fmt.Sprintf("  %s  %-18s %-9s %d/%d cards down",
				progressBar(ratio, 12),
				truncate(b.Pitch.Slug, 18),
				truncate(string(b.Bet.Appetite), 9),
				b.CardsDone, len(b.Cards),
			)
			if len(b.StuckCards) > 0 {
				stuck := b.StuckCards[0]
				line += fmt.Sprintf("  ⚠ stuck on '%s' (uphill %d%%)", stuck.Slug, stuck.Progress)
			}
			fmt.Fprintln(w, line)
		}
		fmt.Fprintln(w)
	}

	if len(r.Passed) > 0 {
		parts := make([]string, 0, len(r.Passed))
		for _, p := range r.Passed {
			parts = append(parts, fmt.Sprintf("%s (%s)", p.Pitch, p.Reason))
		}
		fmt.Fprintf(w, "Passed: %s\n", strings.Join(parts, ", "))
	}

	if r.Cooldown != nil {
		fmt.Fprintf(w, "\nCooldown: %s (ends %s)\n",
			r.Cooldown.Status, r.Cooldown.Ends.Format("2006-01-02"))
	}
}

// progressBar renders an N-character ASCII bar reading `pct` percent
// full as `[==>      ]`. pct is clamped to [0,100]. The bar uses an
// `=` body, a `>` tip when partial, and spaces for the remainder —
// matching the example in 07-mcp-and-cli.md.
func progressBar(pct, width int) string {
	if pct < 0 {
		pct = 0
	}
	if pct > 100 {
		pct = 100
	}
	inner := width - 2
	filled := (pct * inner) / 100
	var b strings.Builder
	b.WriteByte('[')
	for i := 0; i < inner; i++ {
		switch {
		case i < filled-1:
			b.WriteByte('=')
		case i == filled-1 && filled > 0 && filled < inner:
			b.WriteByte('=')
		case i == filled && filled < inner && pct > 0 && filled > 0:
			b.WriteByte('>')
		case i < filled:
			b.WriteByte('=')
		default:
			b.WriteByte(' ')
		}
	}
	b.WriteByte(']')
	return b.String()
}

// truncate trims s to n runes, adding no ellipsis (the column is
// fixed-width and the reader sees the rest in `pitch list`).
func truncate(s string, n int) string {
	if len([]rune(s)) <= n {
		return s
	}
	return string([]rune(s)[:n])
}
