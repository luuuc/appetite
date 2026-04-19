package model

import "time"

// Bet is a decision to build a pitch in a specific cycle with a stated
// appetite. Bets live inside Cycle.Bets — they are not stored as
// standalone files.
type Bet struct {
	Pitch    string    `yaml:"pitch"`
	Appetite Appetite  `yaml:"appetite"`
	BetAt    time.Time `yaml:"bet_at"`
}

// Passed records a pitch that was explicitly dropped during the cycle's
// betting table. Passed entries live inside Cycle.Passed.
type Passed struct {
	Pitch    string    `yaml:"pitch"`
	Reason   string    `yaml:"reason"`
	PassedAt time.Time `yaml:"passed_at"`
}
