package cli

import (
	"flag"
	"fmt"
	"strings"

	"github.com/luuuc/appetite/internal/model"
	"github.com/luuuc/appetite/internal/workflow"
)

func init() {
	register(Command{
		Name:     "signal",
		Synopsis: "record a new signal (raw input)",
		Run:      runSignal,
	})
}

// runSignal dispatches the `signal <subcommand>` two-level command.
// Today only `add` exists; `list` and `sweep` are deferred to a
// later cycle.
func runSignal(env Env, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("signal: subcommand required (add)")
	}
	switch args[0] {
	case "add":
		return runSignalAdd(env, args[1:])
	default:
		return fmt.Errorf("signal: unknown subcommand %q", args[0])
	}
}

func runSignalAdd(env Env, args []string) error {
	fs := flag.NewFlagSet("signal add", flag.ContinueOnError)
	fs.SetOutput(env.Stderr)
	source := fs.String("source", string(model.SignalSourceOperator), "signal source (operator|beacon|customer|council|brain|manual)")
	tags := fs.String("tags", "", "comma-separated tags")
	slug := fs.String("slug", "", "override the auto-generated slug")
	if err := fs.Parse(reorderFlags(fs, args)); err != nil {
		return err
	}
	if fs.NArg() == 0 {
		return fmt.Errorf("signal add: body is required (positional argument)")
	}
	body := strings.Join(fs.Args(), " ")

	res, err := workflow.SignalAdd(openStore(env), workflow.SignalAddRequest{
		Body:   body,
		Source: model.SignalSource(*source),
		Tags:   splitTags(*tags),
		Slug:   *slug,
	})
	if err != nil {
		return err
	}
	fmt.Fprintf(env.Stdout, "signal: wrote %s\n", res.Path)
	return nil
}

// splitTags parses a comma-separated tag list. Empty entries (from
// trailing commas or double commas) are dropped.
func splitTags(s string) []string {
	if s == "" {
		return nil
	}
	raw := strings.Split(s, ",")
	out := raw[:0]
	for _, t := range raw {
		t = strings.TrimSpace(t)
		if t != "" {
			out = append(out, t)
		}
	}
	return out
}
