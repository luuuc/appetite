// Package model defines the core Appetite entities — signals, pitches,
// cycles, cards, cooldowns, bets — plus the state machine that gates
// their transitions. Entities carry YAML struct tags so the storage
// adapter can marshal them directly; they carry no persistence logic.
package model

// Kind identifies which entity type a value represents. The storage
// layer uses Kind to route reads and lists; file paths also encode it
// via the top-level directory (signals/, pitches/, cycles/).
type Kind string

const (
	KindSignal   Kind = "signal"
	KindPitch    Kind = "pitch"
	KindCycle    Kind = "cycle"
	KindCard     Kind = "card"
	KindCooldown Kind = "cooldown"
)

// Entity is implemented by every stored type. Kind and Path let the
// storage adapter handle entities uniformly without type switches in
// the hot path.
type Entity interface {
	// Kind returns the entity kind.
	Kind() Kind
	// Path returns the entity's relative path within .appetite/. It is
	// computed from the entity's own fields (slug, cycle id) — no
	// lookup required.
	Path() string
}

// Appetite is the budget size attached to a pitch or bet. It is never
// a duration estimate — it is a cap.
type Appetite string

const (
	AppetiteMicro  Appetite = "micro"
	AppetiteSmall  Appetite = "small"
	AppetiteMedium Appetite = "medium"
	AppetiteLarge  Appetite = "large"
)

// Valid reports whether a is one of the four defined appetites.
func (a Appetite) Valid() bool {
	switch a {
	case AppetiteMicro, AppetiteSmall, AppetiteMedium, AppetiteLarge:
		return true
	}
	return false
}
