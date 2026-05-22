package model

import (
	"errors"
	"fmt"
)

// ErrInvalidTransition is returned when a status change is not allowed
// by the workflow's state machine. Callers distinguish it with
// errors.Is; the wrapped error carries the from/to/kind context.
var ErrInvalidTransition = errors.New("invalid transition")

// The state machine is encoded from .doc/definition/03-workflow.md.
// Keep it strict — adding a skipped state is not a bug fix, it is a
// design change.
var (
	pitchTransitions = map[PitchStatus]map[PitchStatus]bool{
		PitchStatusShaping:  {PitchStatusShaped: true},
		PitchStatusShaped:   {PitchStatusBet: true, PitchStatusPassed: true},
		PitchStatusBet:      {PitchStatusBuilding: true},
		PitchStatusBuilding: {PitchStatusShipped: true},
		// Shipped and Passed are terminal.
	}

	cycleTransitions = map[CycleStatus]map[CycleStatus]bool{
		CycleStatusBuilding: {CycleStatusShipping: true},
		CycleStatusShipping: {CycleStatusCooldown: true},
		CycleStatusCooldown: {CycleStatusClosed: true},
		// Closed is terminal.
	}

	cooldownTransitions = map[CooldownStatus]map[CooldownStatus]bool{
		CooldownActive: {CooldownClosed: true},
		// Closed is terminal.
	}
)

// statusEnum constrains the generic validator to string-backed status
// types that carry a Valid() method. The three Validate*Transition
// wrappers below all flow through here. A type that isn't a string
// alias or lacks Valid() fails to compile — this is the constraint's
// whole point, so don't loosen it.
type statusEnum interface {
	~string
	Valid() bool
}

// validateTransition is the single implementation behind the per-kind
// validators. Same-state transitions (from == to) are rejected via the
// table never listing an entry for (s, s) — a status flip that does
// not change anything is a caller bug.
func validateTransition[S statusEnum](from, to S, table map[S]map[S]bool, kind string) error {
	if !from.Valid() {
		return fmt.Errorf("%w: unknown %s status %q", ErrInvalidTransition, kind, string(from))
	}
	if !to.Valid() {
		return fmt.Errorf("%w: unknown %s status %q", ErrInvalidTransition, kind, string(to))
	}
	if table[from][to] {
		return nil
	}
	return fmt.Errorf("%w: %s %s → %s", ErrInvalidTransition, kind, string(from), string(to))
}

// ValidatePitchTransition reports whether a pitch may move from `from`
// to `to`.
func ValidatePitchTransition(from, to PitchStatus) error {
	return validateTransition(from, to, pitchTransitions, "pitch")
}

// ValidateCycleTransition reports whether a cycle may move from `from`
// to `to`.
func ValidateCycleTransition(from, to CycleStatus) error {
	return validateTransition(from, to, cycleTransitions, "cycle")
}

// ValidateCooldownTransition reports whether a cooldown may move from
// `from` to `to`.
func ValidateCooldownTransition(from, to CooldownStatus) error {
	return validateTransition(from, to, cooldownTransitions, "cooldown")
}
