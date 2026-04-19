package model

import (
	"fmt"
	"time"
)

// SignalSource names the origin of a signal.
type SignalSource string

const (
	SignalSourceBeacon   SignalSource = "beacon"
	SignalSourceOperator SignalSource = "operator"
	SignalSourceCustomer SignalSource = "customer"
	SignalSourceCouncil  SignalSource = "council"
	SignalSourceBrain    SignalSource = "brain"
	SignalSourceManual   SignalSource = "manual"
)

// Valid reports whether s is a known signal source.
func (s SignalSource) Valid() bool {
	switch s {
	case SignalSourceBeacon, SignalSourceOperator, SignalSourceCustomer,
		SignalSourceCouncil, SignalSourceBrain, SignalSourceManual:
		return true
	}
	return false
}

// Signal is a raw input — something worth paying attention to. Signals
// accumulate in signals/raw/ until they are shaped into a pitch or
// dropped. They are deliberately minimal: shaping is where richness
// arrives.
type Signal struct {
	Slug     string       `yaml:"-"`
	Archived bool         `yaml:"-"` // true once the signal has been moved to signals/archived/

	Source   SignalSource `yaml:"source"`
	Captured time.Time    `yaml:"captured"`
	Tags     []string     `yaml:"tags,omitempty,flow"`

	Body string `yaml:"-"`
}

// Kind implements Entity.
func (Signal) Kind() Kind { return KindSignal }

// Path implements Entity. Archived signals move under signals/archived/;
// raw signals live in signals/raw/.
func (s Signal) Path() string {
	dir := "signals/raw"
	if s.Archived {
		dir = "signals/archived"
	}
	return fmt.Sprintf("%s/%s.md", dir, s.Slug)
}
