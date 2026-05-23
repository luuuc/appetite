package cli

import (
	"errors"

	"github.com/luuuc/appetite/internal/workflow"
)

// exitCodeFor maps an error to the process exit code documented in
// .doc/definition/07-mcp-and-cli.md. The map is intentionally narrow:
// only workflow-level sentinels translate to a non-1 code. Everything
// else (filesystem errors, parse failures, YAML errors, argv shape)
// is a generic failure.
func exitCodeFor(err error) int {
	switch {
	case err == nil:
		return 0
	case errors.Is(err, workflow.ErrInvalidTransition),
		errors.Is(err, workflow.ErrDoneCriteriaUnmet),
		errors.Is(err, errInitUnknownTool),
		errors.Is(err, errInitCommandsDiverged):
		return 2
	case errors.Is(err, workflow.ErrNotFound):
		return 3
	default:
		return 1
	}
}
