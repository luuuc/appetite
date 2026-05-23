package mcp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"testing"

	"github.com/luuuc/appetite/internal/markdown"
)

// TestEndToEndWalk drives the JSON-RPC server through the full
// workflow loop the pitch documents:
//
//	init → signal → shape (new + finalize) → cycle_new → bet → cut → hill → cooldown
//
// The test is single-process: requests are framed into a bytes.Buffer
// piped into Server.Serve, and responses are decoded out the other
// side. Every step asserts on JSON-RPC id round-trip and (where
// relevant) a result-shape field — enough to catch protocol drift
// without coupling the test to the exact JSON of every tool.
func TestEndToEndWalk(t *testing.T) {
	store := markdown.New(t.TempDir())
	srv := New(store, log.New(io.Discard, "", 0))

	var in bytes.Buffer
	type step struct {
		id     int
		method string
		params any
		check  func(t *testing.T, resp Response)
	}
	mustResult := func(t *testing.T, resp Response) {
		t.Helper()
		if resp.Error != nil {
			t.Fatalf("%s err: %+v", "step", resp.Error)
		}
	}

	steps := []step{
		{1, "initialize", nil, mustResult},
		{2, "appetite_signal", signalParams{Text: "Export slow", Source: "operator", Slug: "slow-export"}, mustResult},
		{3, "appetite_shape", shapeParams{Mode: shapeModeFrom, SignalPath: "signals/raw/slow-export.md"}, mustResult},
		// shape_from leaves the pitch in `shaping` with a template
		// body that has all 5 ingredients and no scope cards. Append
		// a scope card before finalizing.
		{4, "appetite_shape", shapeParams{Mode: shapeModeFinalize, Slug: "slow-export"}, func(t *testing.T, resp Response) {
			// This finalize is expected to fail (no scope card yet);
			// the next step adds one via a direct store write.
			if resp.Error == nil {
				t.Fatalf("expected finalize without scope to fail, got: %s", resp.Result)
			}
			if resp.Error.Code != CodeWorkflowViolation {
				t.Errorf("code = %d, want %d", resp.Error.Code, CodeWorkflowViolation)
			}
		}},
	}

	for _, st := range steps {
		writeStep(t, &in, st.id, st.method, st.params)
	}

	var out bytes.Buffer
	if err := srv.Serve(context.Background(), &in, &out); err != nil {
		t.Fatalf("Serve: %v", err)
	}

	responses := decodeAllFrames(t, out.Bytes())
	if len(responses) != len(steps) {
		t.Fatalf("got %d responses, want %d", len(responses), len(steps))
	}
	for i, st := range steps {
		if string(responses[i].ID) != fmt.Sprintf("%d", st.id) {
			t.Errorf("step %d: id = %s, want %d", i+1, responses[i].ID, st.id)
		}
		st.check(t, responses[i])
	}

	// Stage 2: scope card append + the rest of the loop. The
	// pitch's e2e checklist requires we exercise cycle_new → bet →
	// cut → hill → cooldown; we resume here in a second Serve call
	// after patching the pitch body so finalize succeeds.
	patchPitchScope(t, store, "slow-export")

	var in2, out2 bytes.Buffer
	finalizeStep := step{10, "appetite_shape", shapeParams{Mode: shapeModeFinalize, Slug: "slow-export"}, mustResult}
	steps2 := []step{
		finalizeStep,
		{11, "appetite_cycle_new", cycleNewParams{ID: "c1", Duration: "5d"}, mustResult},
		{12, "appetite_bet", betParams{Pitch: "slow-export", Cycle: "c1", Appetite: "small"}, mustResult},
		{13, "appetite_cut", cutParams{Pitch: "slow-export"}, mustResult},
		// One scope card was added in patchPitchScope; cut produces
		// one card slugged "the-only-card".
		{14, "appetite_hill", hillParams{Card: "the-only-card", Progress: ptrInt(60), Position: "downhill"}, mustResult},
		{15, "appetite_hill", hillParams{Card: "the-only-card", Progress: ptrInt(100), Done: true}, func(t *testing.T, resp Response) {
			mustResult(t, resp)
			var hr hillResponse
			if err := json.Unmarshal(resp.Result, &hr); err != nil {
				t.Fatalf("decode hill: %v", err)
			}
			if hr.PitchShipped == nil || hr.CycleShipped == nil {
				t.Errorf("cascade missing: %+v", hr)
			}
		}},
		{16, "appetite_cooldown", cooldownParams{Op: cooldownOpOpen}, mustResult},
		{17, "appetite_cooldown", cooldownParams{Op: cooldownOpClose}, func(t *testing.T, resp Response) {
			mustResult(t, resp)
			var cr cooldownResponse
			if err := json.Unmarshal(resp.Result, &cr); err != nil {
				t.Fatalf("decode cooldown close: %v", err)
			}
			if cr.Cycle.Status != "closed" {
				t.Errorf("cycle = %q, want closed", cr.Cycle.Status)
			}
		}},
	}
	for _, st := range steps2 {
		writeStep(t, &in2, st.id, st.method, st.params)
	}

	if err := srv.Serve(context.Background(), &in2, &out2); err != nil {
		t.Fatalf("Serve stage 2: %v", err)
	}
	responses2 := decodeAllFrames(t, out2.Bytes())
	if len(responses2) != len(steps2) {
		t.Fatalf("stage 2 got %d responses, want %d", len(responses2), len(steps2))
	}
	for i, st := range steps2 {
		if string(responses2[i].ID) != fmt.Sprintf("%d", st.id) {
			t.Errorf("stage 2 step %d: id = %s, want %d", i+1, responses2[i].ID, st.id)
		}
		st.check(t, responses2[i])
	}
}

func writeStep(t *testing.T, w *bytes.Buffer, id int, method string, params any) {
	t.Helper()
	body := map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"method":  method,
	}
	if params != nil {
		body["params"] = params
	}
	data, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	fmt.Fprintf(w, "Content-Length: %d\r\n\r\n", len(data))
	w.Write(data)
}

func decodeAllFrames(t *testing.T, raw []byte) []Response {
	t.Helper()
	br := bufio.NewReader(bytes.NewReader(raw))
	var out []Response
	for {
		payload, err := readFrame(br)
		if errors.Is(err, errFrameClosed) {
			break
		}
		if err != nil {
			t.Fatalf("readFrame: %v", err)
		}
		var resp Response
		if err := json.Unmarshal(payload, &resp); err != nil {
			t.Fatalf("decode: %v", err)
		}
		out = append(out, resp)
	}
	return out
}

func ptrInt(n int) *int { return &n }
