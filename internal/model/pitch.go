package model

import (
	"fmt"
	"time"
)

// PitchStatus is the state of a pitch in the workflow. Transitions
// between statuses are governed by ValidateTransition.
type PitchStatus string

const (
	PitchStatusShaping  PitchStatus = "shaping"
	PitchStatusShaped   PitchStatus = "shaped"
	PitchStatusBet      PitchStatus = "bet"
	PitchStatusBuilding PitchStatus = "building"
	PitchStatusShipped  PitchStatus = "shipped"
	PitchStatusPassed   PitchStatus = "passed"
)

// Valid reports whether s is a known pitch status.
func (s PitchStatus) Valid() bool {
	switch s {
	case PitchStatusShaping, PitchStatusShaped, PitchStatusBet,
		PitchStatusBuilding, PitchStatusShipped, PitchStatusPassed:
		return true
	}
	return false
}

// Pitch is a shaped problem with the five ingredients (problem,
// appetite, solution sketch, rabbit holes, no-gos) plus scope cards.
// The body is the narrative Markdown; frontmatter carries structured
// state.
type Pitch struct {
	Slug string `yaml:"slug"`

	Title       string      `yaml:"title"`
	Appetite    Appetite    `yaml:"appetite"`
	Status      PitchStatus `yaml:"status"`
	ShapedAt    *time.Time  `yaml:"shaped_at,omitempty"`
	ShapedFrom  []string    `yaml:"shaped_from,omitempty,flow"`

	Body string `yaml:"-"`
}

// Kind implements Entity.
func (Pitch) Kind() Kind { return KindPitch }

// Path implements Entity.
func (p Pitch) Path() string {
	return fmt.Sprintf("pitches/%s.md", p.Slug)
}
