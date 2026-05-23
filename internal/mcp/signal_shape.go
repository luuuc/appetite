package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/luuuc/appetite/internal/model"
	"github.com/luuuc/appetite/internal/store"
	"github.com/luuuc/appetite/internal/workflow"
)

// --- appetite_signal -----------------------------------------------------

// signalParams mirrors workflow.SignalAddRequest minus the Now field
// (clients can't inject test clocks). Slug is exposed for clients
// that need stable filenames; omit it and the workflow layer derives
// one.
type signalParams struct {
	Text   string   `json:"text"`
	Source string   `json:"source,omitempty"`
	Tags   []string `json:"tags,omitempty"`
	Slug   string   `json:"slug,omitempty"`
}

// signalResponse reports what was written. Path lets the client
// reference the file in a follow-up `appetite_shape` from_signal call
// without reconstructing it.
type signalResponse struct {
	Slug string `json:"slug"`
	Path string `json:"path"`
}

func handleSignal(_ context.Context, s store.Store, raw json.RawMessage) (any, error) {
	var p signalParams
	if err := decodeParams(raw, &p); err != nil {
		return nil, err
	}
	if p.Text == "" {
		return nil, &Error{Code: CodeInvalidParams, Message: "signal: text is required"}
	}

	res, err := workflow.SignalAdd(s, workflow.SignalAddRequest{
		Body:   p.Text,
		Source: model.SignalSource(p.Source),
		Tags:   p.Tags,
		Slug:   p.Slug,
	})
	if err != nil {
		// SignalAdd validates source/body itself; treat its errors
		// as bad params rather than internal failures.
		return nil, &Error{Code: CodeInvalidParams, Message: err.Error()}
	}
	return signalResponse{Slug: res.Signal.Slug, Path: res.Path}, nil
}

// --- appetite_shape ------------------------------------------------------

// shapeParams uses a Mode discriminator. The three branches mirror
// the CLI's --new / --from / --finalize flags; per-mode fields are
// validated below.
type shapeParams struct {
	Mode       string `json:"mode"`
	Slug       string `json:"slug,omitempty"`
	Title      string `json:"title,omitempty"`
	SignalPath string `json:"signal_path,omitempty"`
}

type shapeResponse struct {
	Slug   string `json:"slug"`
	Path   string `json:"path"`
	Status string `json:"status"`
}

const (
	shapeModeNew      = "new"
	shapeModeFrom     = "from_signal"
	shapeModeFinalize = "finalize"
)

func handleShape(_ context.Context, s store.Store, raw json.RawMessage) (any, error) {
	var p shapeParams
	if err := decodeParams(raw, &p); err != nil {
		return nil, err
	}
	switch p.Mode {
	case shapeModeNew:
		return shapeNew(s, p)
	case shapeModeFrom:
		return shapeFromSignal(s, p)
	case shapeModeFinalize:
		return shapeFinalize(s, p)
	case "":
		return nil, &Error{Code: CodeInvalidParams, Message: "shape: mode is required (new|from_signal|finalize)"}
	default:
		return nil, &Error{Code: CodeInvalidParams, Message: fmt.Sprintf("shape: unknown mode %q (new|from_signal|finalize)", p.Mode)}
	}
}

func shapeNew(s store.Store, p shapeParams) (any, error) {
	if p.Slug == "" {
		return nil, &Error{Code: CodeInvalidParams, Message: "shape new: slug is required"}
	}
	title := p.Title
	if title == "" {
		title = p.Slug
	}
	res, err := workflow.ShapeNew(s, workflow.ShapeNewRequest{Slug: p.Slug, Title: title})
	if err != nil {
		return nil, err
	}
	return shapeResponse{Slug: res.Pitch.Slug, Path: res.Path, Status: string(res.Pitch.Status)}, nil
}

func shapeFromSignal(s store.Store, p shapeParams) (any, error) {
	if p.SignalPath == "" {
		return nil, &Error{Code: CodeInvalidParams, Message: "shape from_signal: signal_path is required"}
	}
	res, err := workflow.ShapeFrom(s, workflow.ShapeFromRequest{
		SignalPath: p.SignalPath,
		Slug:       p.Slug,
		Title:      p.Title,
	})
	if err != nil {
		return nil, err
	}
	return shapeResponse{Slug: res.Pitch.Slug, Path: res.Path, Status: string(res.Pitch.Status)}, nil
}

func shapeFinalize(s store.Store, p shapeParams) (any, error) {
	if p.Slug == "" {
		return nil, &Error{Code: CodeInvalidParams, Message: "shape finalize: slug is required"}
	}
	// The workflow validator embeds the missing ingredient or scope
	// detail directly in the error message (`pitch %q is missing
	// required section(s): ...`); errorFromWorkflow then surfaces it
	// as -32002 with that message intact. No special parsing here —
	// the message *is* the precise field info the pitch requires.
	res, err := workflow.ShapeFinalize(s, workflow.ShapeFinalizeRequest{Slug: p.Slug})
	if err != nil {
		return nil, err
	}
	return shapeResponse{Slug: res.Pitch.Slug, Path: res.Path, Status: string(res.Pitch.Status)}, nil
}
