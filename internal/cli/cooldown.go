package cli

import (
	"flag"
	"fmt"

	"github.com/luuuc/appetite/internal/workflow"
)

func init() {
	register(Command{
		Name:     "cooldown",
		Synopsis: "open or close cooldown for the active cycle",
		Run:      runCooldown,
	})
}

// runCooldown dispatches the two-shape command: `cooldown [--days <n>]`
// opens, `cooldown close` closes.
func runCooldown(env Env, args []string) error {
	if len(args) > 0 && args[0] == "close" {
		return runCooldownClose(env, args[1:])
	}
	return runCooldownOpen(env, args)
}

func runCooldownOpen(env Env, args []string) error {
	fs := flag.NewFlagSet("cooldown", flag.ContinueOnError)
	fs.SetOutput(env.Stderr)
	days := fs.Int("days", 1, "cooldown duration in days")
	cycle := fs.String("cycle", "", "cycle id (optional; defaults to the active cycle)")
	if err := fs.Parse(reorderFlags(fs, args)); err != nil {
		return err
	}
	if *days <= 0 {
		return fmt.Errorf("cooldown: --days must be positive")
	}

	res, err := workflow.CooldownOpen(openStore(env), workflow.CooldownOpenRequest{
		Cycle:    *cycle,
		Duration: fmt.Sprintf("%dd", *days),
	})
	if err != nil {
		return err
	}
	fmt.Fprintf(env.Stdout, "cooldown: opened on cycle %s (ends %s)\n",
		res.Cycle.ID, res.Cooldown.Ends.Format("2006-01-02"))
	return nil
}

func runCooldownClose(env Env, args []string) error {
	fs := flag.NewFlagSet("cooldown close", flag.ContinueOnError)
	fs.SetOutput(env.Stderr)
	cycle := fs.String("cycle", "", "cycle id (optional; defaults to the active cycle)")
	if err := fs.Parse(reorderFlags(fs, args)); err != nil {
		return err
	}
	res, err := workflow.CooldownClose(openStore(env), workflow.CooldownCloseRequest{Cycle: *cycle})
	if err != nil {
		return err
	}
	fmt.Fprintf(env.Stdout, "cooldown: closed on cycle %s (status: closed)\n", res.Cycle.ID)
	return nil
}
