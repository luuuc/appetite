// Package workflow is the rules layer that sits between storage and the
// CLI/MCP surfaces. Every loop operation — shape, finalize, open-cycle,
// bet, pass, cut, hill, cooldown, status — is a function in this
// package that takes a store.Store and a typed request, returns a typed
// result, and reports rule violations as typed errors.
//
// The package owns three things and only three things:
//
//   - Reading current state from the store.
//   - Validating the operation against the state machine in
//     internal/model/transitions.go and the pitch/cycle invariants
//     spelled out in .doc/definition/03-workflow.md.
//   - Writing the next state back atomically.
//
// It does not know about argv, exit codes, stdout, JSON, or MCP. Those
// belong to internal/cli and the future MCP server.
//
// # Failure modes for multi-entity writes
//
// Several operations write more than one entity (Bet, Pass, Cut,
// Hill). The store gives us per-file atomicity (write-temp + rename
// per call), not multi-file transactions, so each operation picks
// an ordering whose worst-case state is recoverable by retry:
//
//   - Bet: pitch first, then cycle. If the pitch flip succeeds but
//     cycle.Bets append fails, retrying hits ErrInvalidTransition
//     (pitch is already `bet`); the operator either fixes the cycle
//     write or accepts that pitch=bet with no cycle record is the
//     true state. The reverse order would silently double-bet on
//     retry — strictly worse.
//
//   - Pass: pitch first, then cycle. Same reasoning as Bet.
//
//   - Cut: cards first, then pitch. If any card write fails, the
//     cards already written are rolled back via store.Delete (best
//     effort) and the pitch stays `bet`. If the final pitch flip
//     fails, the cards stay on disk and the pitch stays `bet` —
//     retry hits the cards-already-exist pre-check, which surfaces
//     the inconsistency precisely.
//
//   - Hill: card first, then pitch cascade, then cycle cascade.
//     Each cascade is gated by an idempotence check (skip if the
//     pitch/cycle is already in the target state), so partial
//     failures are recoverable by replay.
//
// Single-operator use is assumed (see .doc/definition/05-storage.md):
// no file locking, no cross-file transactions. If concurrent writers
// race, last writer wins.
package workflow

import (
	"errors"

	"github.com/luuuc/appetite/internal/model"
	"github.com/luuuc/appetite/internal/store"
)

// ErrInvalidTransition is returned when a status change violates the
// state machine. It re-exports model.ErrInvalidTransition so CLI/MCP
// callers can errors.Is against a single workflow-level sentinel
// without reaching into the model package.
var ErrInvalidTransition = model.ErrInvalidTransition

// ErrNotFound is returned when a referenced entity (pitch, cycle,
// card, signal) is missing. It re-exports store.ErrNotFound for the
// same reason.
var ErrNotFound = store.ErrNotFound

// ErrDoneCriteriaUnmet is returned when a hill update would set
// progress to 100 without the --done assertion. The state machine
// itself does not encode this — it is a pitch-level invariant
// enforced in workflow.Hill.
var ErrDoneCriteriaUnmet = errors.New("done criteria unmet")
