package mcp

import (
	"context"
	"encoding/json"

	"github.com/luuuc/appetite/internal/model"
	"github.com/luuuc/appetite/internal/store"
	"github.com/luuuc/appetite/internal/workflow"
)

// --- appetite_cut --------------------------------------------------------

type cutParams struct {
	Pitch string `json:"pitch"`
	Cycle string `json:"cycle,omitempty"`
}

type cutResponse struct {
	Pitch string        `json:"pitch"`
	Cycle string        `json:"cycle"`
	Cards []cardSummary `json:"cards"`
}

func handleCut(_ context.Context, s store.Store, raw json.RawMessage) (any, error) {
	var p cutParams
	if err := decodeParams(raw, &p); err != nil {
		return nil, err
	}
	if p.Pitch == "" {
		return nil, &Error{Code: CodeInvalidParams, Message: "cut: pitch is required"}
	}
	res, err := workflow.Cut(s, workflow.CutRequest{Pitch: p.Pitch, Cycle: p.Cycle})
	if err != nil {
		return nil, err
	}
	cards := make([]cardSummary, 0, len(res.Cards))
	for _, c := range res.Cards {
		cards = append(cards, cardSummaryOf(c))
	}
	return cutResponse{Pitch: res.Pitch.Slug, Cycle: res.Cycle, Cards: cards}, nil
}

// --- appetite_hill -------------------------------------------------------

// hillParams uses a pointer for Progress so JSON `0` is
// distinguishable from "field omitted". The workflow layer treats
// progress=100 without Done as a -32002 (ErrDoneCriteriaUnmet),
// which is the precise rule the pitch calls out.
type hillParams struct {
	Card     string `json:"card"`
	Cycle    string `json:"cycle,omitempty"`
	Position string `json:"position,omitempty"`
	Progress *int   `json:"progress,omitempty"`
	Done     bool   `json:"done,omitempty"`
}

type hillResponse struct {
	Card         cardSummary    `json:"card"`
	PitchShipped *pitchSummary  `json:"pitch_shipped,omitempty"`
	CycleShipped *cycleSummary  `json:"cycle_shipped,omitempty"`
}

func handleHill(_ context.Context, s store.Store, raw json.RawMessage) (any, error) {
	var p hillParams
	if err := decodeParams(raw, &p); err != nil {
		return nil, err
	}
	if p.Card == "" {
		return nil, &Error{Code: CodeInvalidParams, Message: "hill: card is required"}
	}
	req := workflow.HillRequest{
		Card:     p.Card,
		Cycle:    p.Cycle,
		Position: model.HillPosition(p.Position),
		Done:     p.Done,
		Progress: p.Progress,
	}
	res, err := workflow.Hill(s, req)
	if err != nil {
		return nil, err
	}
	out := hillResponse{Card: cardSummaryOf(res.Card)}
	if res.PitchShipped != nil {
		out.PitchShipped = &pitchSummary{
			Slug:   res.PitchShipped.Slug,
			Title:  res.PitchShipped.Title,
			Status: string(res.PitchShipped.Status),
		}
	}
	if res.CycleShipped != nil {
		out.CycleShipped = &cycleSummary{
			ID:       res.CycleShipped.ID,
			Status:   string(res.CycleShipped.Status),
			Duration: res.CycleShipped.Duration,
			Started:  res.CycleShipped.Started.Format("2006-01-02T15:04:05Z07:00"),
			Ends:     res.CycleShipped.Ends.Format("2006-01-02T15:04:05Z07:00"),
		}
	}
	return out, nil
}

// --- appetite_cooldown ---------------------------------------------------

type cooldownParams struct {
	Op       string `json:"op"`
	Cycle    string `json:"cycle,omitempty"`
	Duration string `json:"duration,omitempty"` // only honored when op=open
}

type cooldownResponse struct {
	Op       string       `json:"op"`
	Cooldown cooldownView `json:"cooldown"`
	Cycle    cycleSummary `json:"cycle"`
}

const (
	cooldownOpOpen  = "open"
	cooldownOpClose = "close"
)

func handleCooldown(_ context.Context, s store.Store, raw json.RawMessage) (any, error) {
	var p cooldownParams
	if err := decodeParams(raw, &p); err != nil {
		return nil, err
	}
	switch p.Op {
	case cooldownOpOpen:
		res, err := workflow.CooldownOpen(s, workflow.CooldownOpenRequest{Cycle: p.Cycle, Duration: p.Duration})
		if err != nil {
			return nil, err
		}
		return buildCooldownResponse(cooldownOpOpen, res), nil
	case cooldownOpClose:
		res, err := workflow.CooldownClose(s, workflow.CooldownCloseRequest{Cycle: p.Cycle})
		if err != nil {
			return nil, err
		}
		return buildCooldownResponse(cooldownOpClose, res), nil
	case "":
		return nil, &Error{Code: CodeInvalidParams, Message: "cooldown: op is required (open|close)"}
	default:
		return nil, &Error{Code: CodeInvalidParams, Message: "cooldown: unknown op " + p.Op + " (open|close)"}
	}
}

func buildCooldownResponse(op string, res workflow.CooldownResult) cooldownResponse {
	return cooldownResponse{
		Op: op,
		Cooldown: cooldownView{
			Status:  string(res.Cooldown.Status),
			Started: res.Cooldown.Started.Format("2006-01-02T15:04:05Z07:00"),
			Ends:    res.Cooldown.Ends.Format("2006-01-02T15:04:05Z07:00"),
		},
		Cycle: cycleSummary{
			ID:       res.Cycle.ID,
			Status:   string(res.Cycle.Status),
			Duration: res.Cycle.Duration,
			Started:  res.Cycle.Started.Format("2006-01-02T15:04:05Z07:00"),
			Ends:     res.Cycle.Ends.Format("2006-01-02T15:04:05Z07:00"),
		},
	}
}
