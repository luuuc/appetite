package model

import (
	"fmt"
	"time"
)

// CooldownStatus is the state of a cycle's cooldown.
type CooldownStatus string

const (
	CooldownActive CooldownStatus = "active"
	CooldownClosed CooldownStatus = "closed"
)

// Valid reports whether s is a known cooldown status.
func (s CooldownStatus) Valid() bool {
	switch s {
	case CooldownActive, CooldownClosed:
		return true
	}
	return false
}

// Cooldown is the post-cycle state. During cooldown, Appetite refuses
// new bets; operators polish shipped work, chase debt, and write
// lessons. Cooldown is bounded — when ends passes (or the operator
// closes it explicitly), the cycle moves to closed.
type Cooldown struct {
	// Cycle is the owning cycle ID — required to locate the file on
	// disk. It is not serialized into the YAML body (the parent
	// directory already names the cycle).
	Cycle string `yaml:"-"`

	Started time.Time      `yaml:"started"`
	Ends    time.Time      `yaml:"ends"`
	Status  CooldownStatus `yaml:"status"`
	Notes   string         `yaml:"notes,omitempty"`
}

// Kind implements Entity.
func (Cooldown) Kind() Kind { return KindCooldown }

// Path implements Entity.
func (c Cooldown) Path() string {
	return fmt.Sprintf("cycles/%s/cooldown.yml", c.Cycle)
}
