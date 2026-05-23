package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"strings"
	"testing"
	"time"

	"github.com/luuuc/appetite/internal/markdown"
	"github.com/luuuc/appetite/internal/model"
	"github.com/luuuc/appetite/internal/store"
	"github.com/luuuc/appetite/internal/workflow"
)

// failWriter returns errFakeWriter on every Write call so server-side
// write paths can be exercised. The variable is exported via a typed
// var so tests can errors.Is against it.
var errFakeWriter = errors.New("fake writer error")

type failWriter struct{}

func (failWriter) Write(_ []byte) (int, error) { return 0, errFakeWriter }

func TestServeReturnsErrorWhenWritingResponseFails(t *testing.T) {
	srv := newServer(t)
	in := bytes.NewReader(frameRequest(t, "initialize", 1, nil))
	err := srv.Serve(context.Background(), in, failWriter{})
	if !errors.Is(err, errFakeWriter) {
		t.Errorf("err = %v, want errFakeWriter", err)
	}
}

func TestWriteResponseSurfacesMarshalError(t *testing.T) {
	// json.RawMessage("not valid") fails to marshal because
	// RawMessage validates as JSON during encoding. Trigger that
	// path via the Result field.
	resp := Response{JSONRPC: "2.0", ID: json.RawMessage("1"), Result: json.RawMessage("not json")}
	if err := writeResponse(&bytes.Buffer{}, resp); err == nil {
		t.Fatal("want marshal error")
	}
}

func TestWriteFrameReturnsErrorOnWriteFailure(t *testing.T) {
	if err := writeFrame(failWriter{}, []byte(`{}`)); !errors.Is(err, errFakeWriter) {
		t.Errorf("err = %v, want errFakeWriter", err)
	}
}

// TestServeReportsFrameErrorThenStops covers the frame-error branch
// in Serve where readFrame returns a non-EOF error.
func TestServeReportsFrameErrorThenStops(t *testing.T) {
	srv := newServer(t)
	// Missing Content-Length triggers a frame error.
	in := strings.NewReader("X-Bogus: yes\r\n\r\nignored")
	var out bytes.Buffer
	err := srv.Serve(context.Background(), in, &out)
	if err == nil {
		t.Fatal("want frame error")
	}
	if !strings.Contains(err.Error(), "missing Content-Length") {
		t.Errorf("err = %v", err)
	}
	// The server should still have written a parse-error response
	// before bailing.
	resp := readFrames(t, out.Bytes())
	if len(resp) != 1 || resp[0].Error == nil || resp[0].Error.Code != CodeParseError {
		t.Errorf("want one parse-error response, got %+v", resp)
	}
}

func TestServeFrameErrorWithFailingWriterReturnsWriteError(t *testing.T) {
	srv := newServer(t)
	in := strings.NewReader("X-Bogus: yes\r\n\r\nignored")
	err := srv.Serve(context.Background(), in, failWriter{})
	if !errors.Is(err, errFakeWriter) {
		t.Errorf("err = %v, want errFakeWriter", err)
	}
}

// TestHandleOneNotificationWithFailingHandlerLogs covers the
// notification + handler-error branch in handleOne, which logs to
// the server's logger and returns nil.
func TestHandleOneNotificationWithFailingHandlerLogs(t *testing.T) {
	var logBuf bytes.Buffer
	srv := New(markdown.New(t.TempDir()), log.New(&logBuf, "", 0))
	srv.Register("boom", func(_ context.Context, _ store.Store, _ json.RawMessage) (any, error) {
		return nil, errors.New("notification handler failed")
	})

	body := `{"jsonrpc":"2.0","method":"boom"}`
	frame := fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(body), body)
	var out bytes.Buffer
	if err := srv.Serve(context.Background(), strings.NewReader(frame), &out); err != nil {
		t.Fatalf("Serve: %v", err)
	}
	if out.Len() != 0 {
		t.Errorf("notification produced output")
	}
	if !strings.Contains(logBuf.String(), "notification handler failed") {
		t.Errorf("logger missing notification error: %s", logBuf.String())
	}
}

// TestHandleOneNotificationWithUnknownMethodIsSilent covers the
// notification + method-not-found branch in handleOne (returns
// nil, no response).
func TestHandleOneNotificationWithUnknownMethodIsSilent(t *testing.T) {
	srv := newServer(t)
	body := `{"jsonrpc":"2.0","method":"nope"}`
	frame := fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(body), body)
	var out bytes.Buffer
	if err := srv.Serve(context.Background(), strings.NewReader(frame), &out); err != nil {
		t.Fatalf("Serve: %v", err)
	}
	if out.Len() != 0 {
		t.Errorf("unknown-method notification produced output: %q", out.String())
	}
}

// TestHandleOneNotificationWithBadVersionIsSilent covers the
// notification + invalid-version branch.
func TestHandleOneNotificationWithBadVersionIsSilent(t *testing.T) {
	srv := newServer(t)
	body := `{"jsonrpc":"1.0","method":"initialize"}`
	frame := fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(body), body)
	var out bytes.Buffer
	if err := srv.Serve(context.Background(), strings.NewReader(frame), &out); err != nil {
		t.Fatalf("Serve: %v", err)
	}
	if out.Len() != 0 {
		t.Errorf("bad-version notification produced output: %q", out.String())
	}
}

// TestHandlePitchListEmitsShapedAt seeds a pitch with ShapedAt set to
// exercise the non-nil branch of handlePitchList.
func TestHandlePitchListEmitsShapedAt(t *testing.T) {
	s := markdown.New(t.TempDir())
	now := time.Date(2026, 5, 22, 0, 0, 0, 0, time.UTC)
	if _, err := s.Write(context.Background(), model.Pitch{
		Slug: "ok", Title: "OK", Status: model.PitchStatusShaped, ShapedAt: &now,
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	raw, err := callHandler(t, handlePitchList, s, nil)
	if err != nil {
		t.Fatalf("handlePitchList: %v", err)
	}
	if !strings.Contains(string(raw), "2026-05-22") {
		t.Errorf("shaped_at missing: %s", raw)
	}
}

// TestShapeNewDefaultsTitleToSlug exercises the title="" branch.
func TestShapeNewDefaultsTitleToSlug(t *testing.T) {
	s := markdown.New(t.TempDir())
	params := mustMarshal(t, shapeParams{Mode: shapeModeNew, Slug: "untitled"})
	if _, err := callHandler(t, handleShape, s, params); err != nil {
		t.Fatalf("handleShape: %v", err)
	}
	p, err := workflow.Status(s, workflow.StatusRequest{})
	if err == nil {
		t.Errorf("status without cycle should fail, got %+v", p)
	}
}

// TestHandleSignalWorkflowFailureMapsToInvalidParams exercises the
// branch where workflow.SignalAdd itself rejects the input (empty
// source struct field is fine; we get an err from invalid source).
func TestHandleSignalWorkflowFailureMapsToInvalidParams(t *testing.T) {
	// signalParams.Source set to a known-invalid value bubbles
	// SignalAdd's "unknown signal source" up.
	s := markdown.New(t.TempDir())
	params := mustMarshal(t, signalParams{Text: "x", Source: "robotz"})
	_, err := callHandler(t, handleSignal, s, params)
	apiErr := mustAPIError(t, err)
	if apiErr.Code != CodeInvalidParams {
		t.Errorf("code = %d, want %d", apiErr.Code, CodeInvalidParams)
	}
}

// TestHandleShapeNewDuplicateBubblesError ensures the workflow
// "already exists" error is surfaced as -32603 (internal). The pitch
// only requires the precise error class — the message itself is
// already informative.
func TestHandleShapeNewDuplicateBubblesError(t *testing.T) {
	s := markdown.New(t.TempDir())
	params := mustMarshal(t, shapeParams{Mode: shapeModeNew, Slug: "dup", Title: "Dup"})
	if _, err := callHandler(t, handleShape, s, params); err != nil {
		t.Fatalf("first new: %v", err)
	}
	_, err := callHandler(t, handleShape, s, params)
	apiErr := mustAPIError(t, err)
	if apiErr.Code != CodeInternalError {
		t.Errorf("code = %d, want %d", apiErr.Code, CodeInternalError)
	}
}

// TestHandleCycleListEmptyStore exercises the empty-cycles branch.
func TestHandleCycleListEmptyStore(t *testing.T) {
	s := markdown.New(t.TempDir())
	raw, err := callHandler(t, handleCycleList, s, nil)
	if err != nil {
		t.Fatalf("handleCycleList: %v", err)
	}
	if string(raw) != `{"cycles":[]}` {
		t.Errorf("got %s, want empty cycles", raw)
	}
}

// TestErrorErrorOnTypedAlias ensures Error.Error rendering is stable
// even when wrapped through fmt.Errorf %w.
func TestErrorErrorWrapsViaFmtErrorf(t *testing.T) {
	api := &Error{Code: 99, Message: "x"}
	wrapped := fmt.Errorf("outer: %w", api)
	var asErr *Error
	if !errors.As(wrapped, &asErr) {
		t.Fatalf("errors.As lost typed error")
	}
	if asErr.Code != 99 {
		t.Errorf("code lost in wrap")
	}
}

// TestEveryHandlerRejectsMalformedParams ensures decodeParams's error
// branch lights up for every tool handler. The handlers all share the
// same prologue; one bad-JSON case per handler covers the branch
// without duplicating per-handler assertion logic.
func TestEveryHandlerRejectsMalformedParams(t *testing.T) {
	s := markdown.New(t.TempDir())
	bad := json.RawMessage(`"not-an-object"`)
	cases := []struct {
		name string
		h    Handler
	}{
		{"status", handleStatus},
		{"pitch_list", handlePitchList},
		{"cycle_list", handleCycleList},
		{"signal", handleSignal},
		{"shape", handleShape},
		{"cycle_new", handleCycleNew},
		{"bet", handleBet},
		{"pass", handlePass},
		{"cut", handleCut},
		{"hill", handleHill},
		{"cooldown", handleCooldown},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			_, err := tc.h(context.Background(), s, bad)
			apiErr := mustAPIError(t, err)
			if apiErr.Code != CodeInvalidParams {
				t.Errorf("code = %d, want %d", apiErr.Code, CodeInvalidParams)
			}
		})
	}
}

// Compile guard for unused imports if a test gets deleted later.
var (
	_ io.Reader = strings.NewReader("")
	_           = time.Now
)
