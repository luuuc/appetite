package cli

import (
	"flag"
	"fmt"

	"github.com/luuuc/appetite/internal/model"
	"github.com/luuuc/appetite/internal/workflow"
)

func init() {
	register(Command{
		Name:     "bet",
		Synopsis: "place a shaped pitch into a building cycle",
		Run:      runBet,
	})
}

func runBet(env Env, args []string) error {
	fs := flag.NewFlagSet("bet", flag.ContinueOnError)
	fs.SetOutput(env.Stderr)
	cycle := fs.String("cycle", "", "cycle id (required)")
	appetite := fs.String("appetite", "", "bet appetite: micro|small|medium|large (required)")
	if err := fs.Parse(reorderFlags(fs, args)); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("bet: exactly one positional <pitch-slug> is required")
	}
	if *cycle == "" {
		return fmt.Errorf("bet: --cycle is required")
	}
	if *appetite == "" {
		return fmt.Errorf("bet: --appetite is required")
	}

	if _, err := workflow.Bet(openStore(env), workflow.BetRequest{
		Pitch:    fs.Arg(0),
		Cycle:    *cycle,
		Appetite: model.Appetite(*appetite),
	}); err != nil {
		return err
	}
	fmt.Fprintf(env.Stdout, "bet: %s placed on %s (%s)\n", fs.Arg(0), *cycle, *appetite)
	return nil
}
