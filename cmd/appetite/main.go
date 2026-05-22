// Command appetite is the CLI entry point. It is intentionally tiny:
// argv goes to internal/cli.Main, the returned exit code goes to
// os.Exit. Everything else — parsing, dispatch, workflow calls,
// rendering — lives in internal/cli and internal/workflow. This
// keeps main coverage-able via subprocess tests in main_test.go
// without importing main.
package main

import (
	"os"

	"github.com/luuuc/appetite/internal/cli"
)

func main() {
	os.Exit(cli.Main(os.Args[1:], cli.DefaultEnv()))
}
