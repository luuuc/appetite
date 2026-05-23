package mcp

import (
	"context"
	"encoding/json"

	"github.com/luuuc/appetite/internal/model"
	"github.com/luuuc/appetite/internal/store"
	"github.com/luuuc/appetite/internal/workflow"
)

// --- appetite_cycle_new --------------------------------------------------

type cycleNewParams struct {
	ID       string `json:"id"`
	Duration string `json:"duration"`
}

type cycleNewResponse struct {
	ID       string `json:"id"`
	Status   string `json:"status"`
	Duration string `json:"duration"`
	Started  string `json:"started"`
	Ends     string `json:"ends"`
	Path     string `json:"path"`
}

func handleCycleNew(_ context.Context, s store.Store, raw json.RawMessage) (any, error) {
	var p cycleNewParams
	if err := decodeParams(raw, &p); err != nil {
		return nil, err
	}
	if p.ID == "" {
		return nil, &Error{Code: CodeInvalidParams, Message: "cycle_new: id is required"}
	}
	if p.Duration == "" {
		return nil, &Error{Code: CodeInvalidParams, Message: "cycle_new: duration is required (e.g. 5d, 2w)"}
	}
	res, err := workflow.CycleNew(s, workflow.CycleNewRequest{ID: p.ID, Duration: p.Duration})
	if err != nil {
		return nil, err
	}
	return cycleNewResponse{
		ID:       res.Cycle.ID,
		Status:   string(res.Cycle.Status),
		Duration: res.Cycle.Duration,
		Started:  res.Cycle.Started.Format("2006-01-02T15:04:05Z07:00"),
		Ends:     res.Cycle.Ends.Format("2006-01-02T15:04:05Z07:00"),
		Path:     res.Path,
	}, nil
}

// --- appetite_bet --------------------------------------------------------

type betParams struct {
	Pitch    string `json:"pitch"`
	Cycle    string `json:"cycle"`
	Appetite string `json:"appetite"`
}

type betResponse struct {
	Pitch     string `json:"pitch"`
	Cycle     string `json:"cycle"`
	Appetite  string `json:"appetite"`
	PitchPath string `json:"pitch_path"`
	CyclePath string `json:"cycle_path"`
}

func handleBet(_ context.Context, s store.Store, raw json.RawMessage) (any, error) {
	var p betParams
	if err := decodeParams(raw, &p); err != nil {
		return nil, err
	}
	if p.Pitch == "" {
		return nil, &Error{Code: CodeInvalidParams, Message: "bet: pitch is required"}
	}
	if p.Cycle == "" {
		return nil, &Error{Code: CodeInvalidParams, Message: "bet: cycle is required"}
	}
	if p.Appetite == "" {
		return nil, &Error{Code: CodeInvalidParams, Message: "bet: appetite is required (micro|small|medium|large)"}
	}
	res, err := workflow.Bet(s, workflow.BetRequest{
		Pitch:    p.Pitch,
		Cycle:    p.Cycle,
		Appetite: model.Appetite(p.Appetite),
	})
	if err != nil {
		return nil, err
	}
	return betResponse{
		Pitch:     p.Pitch,
		Cycle:     p.Cycle,
		Appetite:  p.Appetite,
		PitchPath: res.PitchPath,
		CyclePath: res.CyclePath,
	}, nil
}

// --- appetite_pass -------------------------------------------------------

type passParams struct {
	Pitch  string `json:"pitch"`
	Reason string `json:"reason"`
	Cycle  string `json:"cycle,omitempty"`
}

type passResponse struct {
	Pitch     string `json:"pitch"`
	Cycle     string `json:"cycle,omitempty"`
	PitchPath string `json:"pitch_path"`
	CyclePath string `json:"cycle_path,omitempty"`
}

func handlePass(_ context.Context, s store.Store, raw json.RawMessage) (any, error) {
	var p passParams
	if err := decodeParams(raw, &p); err != nil {
		return nil, err
	}
	if p.Pitch == "" {
		return nil, &Error{Code: CodeInvalidParams, Message: "pass: pitch is required"}
	}
	if p.Reason == "" {
		return nil, &Error{Code: CodeInvalidParams, Message: "pass: reason is required"}
	}
	res, err := workflow.Pass(s, workflow.PassRequest{
		Pitch:  p.Pitch,
		Reason: p.Reason,
		Cycle:  p.Cycle,
	})
	if err != nil {
		return nil, err
	}
	return passResponse{
		Pitch:     p.Pitch,
		Cycle:     p.Cycle,
		PitchPath: res.PitchPath,
		CyclePath: res.CyclePath,
	}, nil
}
