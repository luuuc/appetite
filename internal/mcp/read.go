package mcp

import (
	"context"
	"encoding/json"
	"sort"

	"github.com/luuuc/appetite/internal/model"
	"github.com/luuuc/appetite/internal/store"
	"github.com/luuuc/appetite/internal/workflow"
)

// --- appetite_status -----------------------------------------------------

// statusParams selects the cycle to inspect. Empty cycle id falls
// back to the single active cycle, matching the CLI default.
type statusParams struct {
	Cycle string `json:"cycle,omitempty"`
}

// statusResponse is the structured snapshot every AI tool sees. It
// flattens StatusResult into JSON-friendly shapes (no Go-only types,
// no nested pointers that look ambiguous in JSON).
type statusResponse struct {
	Cycle          cycleSummary    `json:"cycle"`
	Day            int             `json:"day"`
	DayTotal       int             `json:"day_total"`
	Bets           []betSummary    `json:"bets"`
	Passed         []passedSummary `json:"passed,omitempty"`
	Cooldown       *cooldownView   `json:"cooldown,omitempty"`
	Stuck          []cardSummary   `json:"stuck"`
	SyncProposals  []syncProposal  `json:"sync_proposals"`
}

type cycleSummary struct {
	ID       string `json:"id"`
	Status   string `json:"status"`
	Duration string `json:"duration"`
	Started  string `json:"started"`
	Ends     string `json:"ends"`
}

type betSummary struct {
	Pitch     string        `json:"pitch"`
	Status    string        `json:"pitch_status"`
	Appetite  string        `json:"appetite"`
	BetAt     string        `json:"bet_at"`
	Cards     []cardSummary `json:"cards"`
	CardsDone int           `json:"cards_done"`
	Stuck     []cardSummary `json:"stuck,omitempty"`
}

type cardSummary struct {
	Slug          string `json:"slug"`
	Pitch         string `json:"pitch"`
	Cycle         string `json:"cycle"`
	Hill          string `json:"hill"`
	Progress      int    `json:"progress"`
	HillUpdatedAt string `json:"hill_updated_at,omitempty"`
}

type passedSummary struct {
	Pitch    string `json:"pitch"`
	Reason   string `json:"reason"`
	PassedAt string `json:"passed_at"`
}

type cooldownView struct {
	Status  string `json:"status"`
	Started string `json:"started"`
	Ends    string `json:"ends"`
}

// syncProposal is the placeholder shape `.doc/definition/07-mcp-and-cli.md`
// reserves for PM-tool integration. Cycle 3+ populates it; for now
// the field exists and is always empty so the response shape is
// stable.
type syncProposal struct {
	Kind             string `json:"kind"`
	Pitch            string `json:"pitch,omitempty"`
	Adapter          string `json:"adapter"`
	SuggestedCommand string `json:"suggested_command,omitempty"`
}

// handleStatus wraps workflow.Status. Empty params is equivalent to
// {} — the active cycle is resolved by the workflow layer.
func handleStatus(_ context.Context, s store.Store, raw json.RawMessage) (any, error) {
	var p statusParams
	if err := decodeParams(raw, &p); err != nil {
		return nil, err
	}
	res, err := workflow.Status(s, workflow.StatusRequest{Cycle: p.Cycle})
	if err != nil {
		return nil, err
	}
	return buildStatusResponse(res), nil
}

func buildStatusResponse(res workflow.StatusResult) statusResponse {
	out := statusResponse{
		Cycle: cycleSummary{
			ID:       res.Cycle.ID,
			Status:   string(res.Cycle.Status),
			Duration: res.Cycle.Duration,
			Started:  res.Cycle.Started.Format("2006-01-02T15:04:05Z07:00"),
			Ends:     res.Cycle.Ends.Format("2006-01-02T15:04:05Z07:00"),
		},
		Day:           res.DayN,
		DayTotal:      res.DayTotal,
		Bets:          make([]betSummary, 0, len(res.Bets)),
		Stuck:         []cardSummary{},
		SyncProposals: []syncProposal{},
	}
	for _, b := range res.Bets {
		bs := betSummary{
			Pitch:     b.Bet.Pitch,
			Status:    string(b.Pitch.Status),
			Appetite:  string(b.Bet.Appetite),
			BetAt:     b.Bet.BetAt.Format("2006-01-02T15:04:05Z07:00"),
			Cards:     make([]cardSummary, 0, len(b.Cards)),
			CardsDone: b.CardsDone,
		}
		for _, c := range b.Cards {
			bs.Cards = append(bs.Cards, cardSummaryOf(c))
		}
		for _, c := range b.StuckCards {
			cs := cardSummaryOf(c)
			bs.Stuck = append(bs.Stuck, cs)
			out.Stuck = append(out.Stuck, cs)
		}
		out.Bets = append(out.Bets, bs)
	}
	for _, p := range res.Passed {
		out.Passed = append(out.Passed, passedSummary{
			Pitch:    p.Pitch,
			Reason:   p.Reason,
			PassedAt: p.PassedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}
	if res.Cooldown != nil {
		out.Cooldown = &cooldownView{
			Status:  string(res.Cooldown.Status),
			Started: res.Cooldown.Started.Format("2006-01-02T15:04:05Z07:00"),
			Ends:    res.Cooldown.Ends.Format("2006-01-02T15:04:05Z07:00"),
		}
	}
	return out
}

func cardSummaryOf(c model.Card) cardSummary {
	cs := cardSummary{
		Slug:     c.Slug,
		Pitch:    c.Pitch,
		Cycle:    c.Cycle,
		Hill:     string(c.Hill),
		Progress: c.Progress,
	}
	if c.HillUpdatedAt != nil {
		cs.HillUpdatedAt = c.HillUpdatedAt.Format("2006-01-02T15:04:05Z07:00")
	}
	return cs
}

// --- appetite_pitch_list -------------------------------------------------

// pitchListParams optionally narrows by status. Empty means "all
// statuses".
type pitchListParams struct {
	Status string `json:"status,omitempty"`
}

type pitchListResponse struct {
	Pitches []pitchSummary `json:"pitches"`
}

type pitchSummary struct {
	Slug     string `json:"slug"`
	Title    string `json:"title"`
	Status   string `json:"status"`
	ShapedAt string `json:"shaped_at,omitempty"`
}

func handlePitchList(_ context.Context, s store.Store, raw json.RawMessage) (any, error) {
	var p pitchListParams
	if err := decodeParams(raw, &p); err != nil {
		return nil, err
	}
	if p.Status != "" && !model.PitchStatus(p.Status).Valid() {
		return nil, &Error{Code: CodeInvalidParams, Message: "unknown pitch status " + p.Status}
	}

	kind := model.KindPitch
	entities, err := s.List(context.Background(), store.Filter{Kind: &kind})
	if err != nil {
		return nil, err
	}
	out := pitchListResponse{Pitches: make([]pitchSummary, 0, len(entities))}
	for _, e := range entities {
		if pitch, ok := e.(model.Pitch); ok {
			if p.Status != "" && string(pitch.Status) != p.Status {
				continue
			}
			ps := pitchSummary{
				Slug:   pitch.Slug,
				Title:  pitch.Title,
				Status: string(pitch.Status),
			}
			if pitch.ShapedAt != nil {
				ps.ShapedAt = pitch.ShapedAt.Format("2006-01-02T15:04:05Z07:00")
			}
			out.Pitches = append(out.Pitches, ps)
		}
	}
	sort.Slice(out.Pitches, func(i, j int) bool { return out.Pitches[i].Slug < out.Pitches[j].Slug })
	return out, nil
}

// --- appetite_cycle_list -------------------------------------------------

type cycleListParams struct {
	Status string `json:"status,omitempty"`
}

type cycleListResponse struct {
	Cycles []cycleListEntry `json:"cycles"`
}

type cycleListEntry struct {
	ID       string `json:"id"`
	Status   string `json:"status"`
	Duration string `json:"duration"`
	Started  string `json:"started"`
	Ends     string `json:"ends"`
	BetCount int    `json:"bet_count"`
}

func handleCycleList(_ context.Context, s store.Store, raw json.RawMessage) (any, error) {
	var p cycleListParams
	if err := decodeParams(raw, &p); err != nil {
		return nil, err
	}
	if p.Status != "" && !model.CycleStatus(p.Status).Valid() {
		return nil, &Error{Code: CodeInvalidParams, Message: "unknown cycle status " + p.Status}
	}

	kind := model.KindCycle
	entities, err := s.List(context.Background(), store.Filter{Kind: &kind})
	if err != nil {
		return nil, err
	}
	out := cycleListResponse{Cycles: make([]cycleListEntry, 0, len(entities))}
	for _, e := range entities {
		if cyc, ok := e.(model.Cycle); ok {
			if p.Status != "" && string(cyc.Status) != p.Status {
				continue
			}
			out.Cycles = append(out.Cycles, cycleListEntry{
				ID:       cyc.ID,
				Status:   string(cyc.Status),
				Duration: cyc.Duration,
				Started:  cyc.Started.Format("2006-01-02T15:04:05Z07:00"),
				Ends:     cyc.Ends.Format("2006-01-02T15:04:05Z07:00"),
				BetCount: len(cyc.Bets),
			})
		}
	}
	sort.Slice(out.Cycles, func(i, j int) bool { return out.Cycles[i].ID < out.Cycles[j].ID })
	return out, nil
}
