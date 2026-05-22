package cli

import (
	"flag"
	"fmt"

	"github.com/luuuc/appetite/internal/workflow"
)

func init() {
	register(Command{
		Name:     "shape",
		Synopsis: "create or finalize a pitch (--new | --from | --finalize)",
		Run:      runShape,
	})
}

// runShape parses argv for the three shape modes: --new, --from,
// --finalize. Exactly one mode flag must be set; mixing them is an
// argv error (exit 1). Each mode then takes the slug or signal path
// as a positional or via the mode flag's value.
func runShape(env Env, args []string) error {
	fs := flag.NewFlagSet("shape", flag.ContinueOnError)
	fs.SetOutput(env.Stderr)
	newSlug := fs.String("new", "", "create a new pitch from scratch with this slug")
	fromPath := fs.String("from", "", "create a pitch seeded from this signal path (e.g. signals/raw/foo.md)")
	finalize := fs.String("finalize", "", "finalize the pitch with this slug (flip to 'shaped')")
	title := fs.String("title", "", "pitch title (used with --new or --from)")
	if err := fs.Parse(reorderFlags(fs, args)); err != nil {
		return err
	}

	modes := 0
	for _, v := range []string{*newSlug, *fromPath, *finalize} {
		if v != "" {
			modes++
		}
	}
	if modes != 1 {
		return fmt.Errorf("shape: exactly one of --new, --from, --finalize is required")
	}

	store := openStore(env)
	switch {
	case *newSlug != "":
		t := *title
		if t == "" {
			t = *newSlug
		}
		res, err := workflow.ShapeNew(store, workflow.ShapeNewRequest{Slug: *newSlug, Title: t})
		if err != nil {
			return err
		}
		fmt.Fprintf(env.Stdout, "shape: created %s (status: shaping)\n", res.Path)
	case *fromPath != "":
		res, err := workflow.ShapeFrom(store, workflow.ShapeFromRequest{
			SignalPath: *fromPath,
			Title:      *title,
		})
		if err != nil {
			return err
		}
		fmt.Fprintf(env.Stdout, "shape: created %s from %s (status: shaping)\n", res.Path, *fromPath)
	case *finalize != "":
		res, err := workflow.ShapeFinalize(store, workflow.ShapeFinalizeRequest{Slug: *finalize})
		if err != nil {
			return err
		}
		fmt.Fprintf(env.Stdout, "shape: finalized %s (status: shaped)\n", res.Path)
	}
	return nil
}
