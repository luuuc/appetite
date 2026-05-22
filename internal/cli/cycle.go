package cli

import (
	"flag"
	"fmt"

	"github.com/luuuc/appetite/internal/workflow"
)

func init() {
	register(Command{
		Name:     "cycle",
		Synopsis: "manage cycles (new)",
		Run:      runCycle,
	})
}

func runCycle(env Env, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("cycle: subcommand required (new)")
	}
	switch args[0] {
	case "new":
		return runCycleNew(env, args[1:])
	default:
		return fmt.Errorf("cycle: unknown subcommand %q", args[0])
	}
}

func runCycleNew(env Env, args []string) error {
	fs := flag.NewFlagSet("cycle new", flag.ContinueOnError)
	fs.SetOutput(env.Stderr)
	appetite := fs.String("appetite", "", "calendar budget for the cycle (e.g. 5d, 2w)")
	if err := fs.Parse(reorderFlags(fs, args)); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("cycle new: exactly one positional <id> is required")
	}
	if *appetite == "" {
		return fmt.Errorf("cycle new: --appetite is required (e.g. 5d, 2w)")
	}

	res, err := workflow.CycleNew(openStore(env), workflow.CycleNewRequest{
		ID:       fs.Arg(0),
		Duration: *appetite,
	})
	if err != nil {
		return err
	}
	fmt.Fprintf(env.Stdout, "cycle: opened %s (status: building, %s → %s)\n",
		res.Path, res.Cycle.Started.Format("2006-01-02"), res.Cycle.Ends.Format("2006-01-02"))
	return nil
}
