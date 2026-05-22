package cli

import (
	"flag"
	"fmt"

	"github.com/luuuc/appetite/internal/workflow"
)

func init() {
	register(Command{
		Name:     "cut",
		Synopsis: "break a bet pitch into scope cards inside its cycle",
		Run:      runCut,
	})
}

func runCut(env Env, args []string) error {
	fs := flag.NewFlagSet("cut", flag.ContinueOnError)
	fs.SetOutput(env.Stderr)
	cycle := fs.String("cycle", "", "cycle id (optional; resolved from the bet entry when omitted)")
	if err := fs.Parse(reorderFlags(fs, args)); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("cut: exactly one positional <pitch-slug> is required")
	}

	res, err := workflow.Cut(openStore(env), workflow.CutRequest{
		Pitch: fs.Arg(0),
		Cycle: *cycle,
	})
	if err != nil {
		return err
	}
	fmt.Fprintf(env.Stdout, "cut: %s → %d card(s) under cycles/%s/cards/ (status: building)\n",
		res.Pitch.Slug, len(res.Cards), res.Cycle)
	for _, c := range res.Cards {
		fmt.Fprintf(env.Stdout, "  %s\n", c.Slug)
	}
	return nil
}
