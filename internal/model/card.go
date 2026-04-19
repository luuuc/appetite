package model

import "fmt"

// HillPosition is the coarse position of a card on the hill chart.
// There are only two values: uphill (figuring it out) and downhill
// (executing known work).
type HillPosition string

const (
	HillUphill   HillPosition = "uphill"
	HillDownhill HillPosition = "downhill"
)

// Valid reports whether h is a known hill position.
func (h HillPosition) Valid() bool {
	switch h {
	case HillUphill, HillDownhill:
		return true
	}
	return false
}

// Card is a unit of work cut from a pitch. Each card lives inside a
// cycle and carries its own hill position and progress. Progress is a
// vibe (0-100), not a burndown.
type Card struct {
	Slug  string `yaml:"slug"`
	Pitch string `yaml:"pitch"`
	Cycle string `yaml:"cycle"`

	Hill     HillPosition `yaml:"hill"`
	Progress int          `yaml:"progress"` // 0-100
	Assigned string       `yaml:"assigned,omitempty"`

	DoneLooksLike []string `yaml:"done_looks_like,omitempty"`

	Body string `yaml:"-"`
}

// Kind implements Entity.
func (Card) Kind() Kind { return KindCard }

// Path implements Entity.
func (c Card) Path() string {
	return fmt.Sprintf("cycles/%s/cards/%s.md", c.Cycle, c.Slug)
}
