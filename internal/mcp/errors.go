package mcp

import (
	"errors"

	"github.com/luuuc/appetite/internal/workflow"
)

// errorFromWorkflow maps a workflow error onto a JSON-RPC error. The
// mapping mirrors internal/cli.exitCodeFor so the two surfaces report
// the same class of failure with the same observable code. Callers
// must guard against nil — passing nil here is a programmer bug.
func errorFromWorkflow(err error) *Error {
	switch {
	case errors.Is(err, workflow.ErrInvalidTransition),
		errors.Is(err, workflow.ErrDoneCriteriaUnmet):
		return &Error{Code: CodeWorkflowViolation, Message: err.Error()}
	case errors.Is(err, workflow.ErrNotFound):
		return &Error{Code: CodeNotFound, Message: err.Error()}
	default:
		return &Error{Code: CodeInternalError, Message: err.Error()}
	}
}
