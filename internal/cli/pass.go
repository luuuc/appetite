package cli

import (
	"flag"
	"fmt"

	"github.com/luuuc/appetite/internal/workflow"
)

func init() {
	register(Command{
		Name:     "pass",
		Synopsis: "drop a shaped pitch with a recorded reason",
		Run:      runPass,
	})
}

func runPass(env Env, args []string) error {
	fs := flag.NewFlagSet("pass", flag.ContinueOnError)
	fs.SetOutput(env.Stderr)
	reason := fs.String("reason", "", "why the pitch is being passed (required)")
	cycle := fs.String("cycle", "", "cycle id (optional; defaults to the active cycle)")
	if err := fs.Parse(reorderFlags(fs, args)); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("pass: exactly one positional <pitch-slug> is required")
	}
	if *reason == "" {
		return fmt.Errorf("pass: --reason is required")
	}

	res, err := workflow.Pass(openStore(env), workflow.PassRequest{
		Pitch:  fs.Arg(0),
		Reason: *reason,
		Cycle:  *cycle,
	})
	if err != nil {
		return err
	}
	if res.CyclePath != "" {
		fmt.Fprintf(env.Stdout, "pass: %s passed (reason recorded on %s)\n", fs.Arg(0), res.CyclePath)
	} else {
		fmt.Fprintf(env.Stdout, "pass: %s passed (no active cycle, reason not recorded on disk)\n", fs.Arg(0))
	}
	return nil
}
