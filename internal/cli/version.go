package cli

import (
	"fmt"

	"github.com/luuuc/appetite/internal/version"
)

func init() {
	register(Command{
		Name:     "version",
		Synopsis: "print the appetite version",
		Run:      runVersion,
	})
}

func runVersion(env Env, _ []string) error {
	_, err := fmt.Fprintln(env.Stdout, version.Version)
	return err
}
