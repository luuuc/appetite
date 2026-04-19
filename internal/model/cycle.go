package model

import (
	"fmt"
	"time"
)

// CycleStatus is the state of a cycle.
type CycleStatus string

const (
	CycleStatusBuilding CycleStatus = "building"
	CycleStatusShipping CycleStatus = "shipping"
	CycleStatusCooldown CycleStatus = "cooldown"
	CycleStatusClosed   CycleStatus = "closed"
)

// Valid reports whether s is a known cycle status.
func (s CycleStatus) Valid() bool {
	switch s {
	case CycleStatusBuilding, CycleStatusShipping, CycleStatusCooldown, CycleStatusClosed:
		return true
	}
	return false
}

// Cycle is a bounded window with bets, status, dates. The ID is the
// operator's choice: ISO week (2026-w15), quarter (2026-q2-cycle-3),
// or free-form (auth-refactor-cycle). The ID doubles as the cycle's
// subdirectory name under cycles/.
//
// Duration is a time budget expressed as a Go-ish duration string
// ("5d", "2w"). It is intentionally not the Appetite enum: pitches get
// a sized appetite (micro/small/medium/large), cycles get a real clock
// budget. Mixing the two would confuse the workflow engine.
type Cycle struct {
	ID       string      `yaml:"id"`
	Duration string      `yaml:"duration"`
	Started  time.Time   `yaml:"started"`
	Ends     time.Time   `yaml:"ends"`
	Status   CycleStatus `yaml:"status"`

	Bets   []Bet    `yaml:"bets,omitempty"`
	Passed []Passed `yaml:"passed,omitempty"`
}

// Kind implements Entity.
func (Cycle) Kind() Kind { return KindCycle }

// Path implements Entity.
func (c Cycle) Path() string {
	return fmt.Sprintf("cycles/%s/cycle.yml", c.ID)
}
