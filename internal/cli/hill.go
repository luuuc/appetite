package cli

import (
	"flag"
	"fmt"

	"github.com/luuuc/appetite/internal/model"
	"github.com/luuuc/appetite/internal/workflow"
)

func init() {
	register(Command{
		Name:     "hill",
		Synopsis: "update a card's hill position and/or progress (--done to ship)",
		Run:      runHill,
	})
}

func runHill(env Env, args []string) error {
	fs := flag.NewFlagSet("hill", flag.ContinueOnError)
	fs.SetOutput(env.Stderr)
	position := fs.String("position", "", "hill position: uphill|downhill")
	progress := fs.Int("progress", -1, "0-100; pass with --done to ship (100)")
	done := fs.Bool("done", false, "assert done_looks_like satisfied (required for progress=100)")
	cycle := fs.String("cycle", "", "cycle id (optional; resolved by card slug when omitted)")
	if err := fs.Parse(reorderFlags(fs, args)); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("hill: exactly one positional <card-slug> is required")
	}

	req := workflow.HillRequest{
		Card:     fs.Arg(0),
		Cycle:    *cycle,
		Position: model.HillPosition(*position),
		Done:     *done,
	}
	// -1 sentinel distinguishes "not set" from a real 0 update.
	if *progress >= 0 {
		p := *progress
		req.Progress = &p
	}

	res, err := workflow.Hill(openStore(env), req)
	if err != nil {
		return err
	}

	fmt.Fprintf(env.Stdout, "hill: %s → %s / %d%%\n", res.Card.Slug, res.Card.Hill, res.Card.Progress)
	if res.PitchShipped != nil {
		fmt.Fprintf(env.Stdout, "  pitch %s shipped\n", res.PitchShipped.Slug)
	}
	if res.CycleShipped != nil {
		fmt.Fprintf(env.Stdout, "  cycle %s → shipping\n", res.CycleShipped.ID)
	}
	return nil
}
