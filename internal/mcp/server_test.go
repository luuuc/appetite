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
	"strings"
	"testing"

	"github.com/luuuc/appetite/internal/markdown"
	"github.com/luuuc/appetite/internal/store"
	"github.com/luuuc/appetite/internal/workflow"
)

// frameRequest wraps payload in Content-Length framing the server
// will accept on input.
func frameRequest(t *testing.T, method string, id any, params any) []byte {
	t.Helper()
	body := map[string]any{
		"jsonrpc": "2.0",
		"method":  method,
	}
	if id != nil {
		body["id"] = id
	}
	if params != nil {
		body["params"] = params
	}
	data, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return []byte(fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(data), data))
}

// readFrames decodes every frame out of buf and returns the parsed
// responses. It exists so tests can assert the stream contains only
// protocol frames — anything left over after parsing means stdout
// was polluted.
func readFrames(t *testing.T, raw []byte) []Response {
	t.Helper()
	br := bufio.NewReader(bytes.NewReader(raw))
	var out []Response
	for {
		payload, err := readFrame(br)
		if errors.Is(err, errFrameClosed) {
			break
		}
		if err != nil {
			t.Fatalf("readFrame: %v\nremaining:\n%s", err, string(raw))
		}
		var resp Response
		if err := json.Unmarshal(payload, &resp); err != nil {
			t.Fatalf("unmarshal response %s: %v", payload, err)
		}
		out = append(out, resp)
	}
	// readFrame loop must exit because there is no more data — any
	// trailing bytes (newlines, log spillage) would have been swallowed
	// here. Sanity-check that the buffer has actually been consumed.
	if rest, _ := br.Peek(1); len(rest) > 0 {
		t.Fatalf("trailing bytes on output stream: %q", rest)
	}
	return out
}

// newServer returns a server backed by a fresh tempdir-rooted store.
// Tests that don't need disk state can ignore the store; it's there
// so handler tests in later cards can write entities through the
// same fixture.
func newServer(t *testing.T) *Server {
	t.Helper()
	store := markdown.New(t.TempDir())
	logger := log.New(io.Discard, "", 0)
	return New(store, logger)
}

func TestServeHandlesInitialize(t *testing.T) {
	srv := newServer(t)
	in := bytes.NewReader(frameRequest(t, "initialize", 1, nil))
	var out bytes.Buffer

	if err := srv.Serve(context.Background(), in, &out); err != nil {
		t.Fatalf("Serve: %v", err)
	}

	responses := readFrames(t, out.Bytes())
	if len(responses) != 1 {
		t.Fatalf("got %d responses, want 1", len(responses))
	}
	resp := responses[0]
	if resp.JSONRPC != "2.0" {
		t.Errorf("jsonrpc = %q, want 2.0", resp.JSONRPC)
	}
	if string(resp.ID) != "1" {
		t.Errorf("id = %s, want 1", resp.ID)
	}
	if resp.Error != nil {
		t.Errorf("unexpected error: %+v", resp.Error)
	}

	var result initializeResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		t.Fatalf("decode result: %v", err)
	}
	if result.ProtocolVersion != ProtocolVersion {
		t.Errorf("protocolVersion = %q, want %q", result.ProtocolVersion, ProtocolVersion)
	}
	if result.ServerInfo.Name != ServerName {
		t.Errorf("serverInfo.name = %q, want %q", result.ServerInfo.Name, ServerName)
	}
}

func TestServeHandlesMultipleFrames(t *testing.T) {
	srv := newServer(t)
	var in bytes.Buffer
	in.Write(frameRequest(t, "initialize", 1, nil))
	in.Write(frameRequest(t, "initialize", 2, nil))

	var out bytes.Buffer
	if err := srv.Serve(context.Background(), &in, &out); err != nil {
		t.Fatalf("Serve: %v", err)
	}

	responses := readFrames(t, out.Bytes())
	if len(responses) != 2 {
		t.Fatalf("got %d responses, want 2", len(responses))
	}
	if string(responses[0].ID) != "1" || string(responses[1].ID) != "2" {
		t.Errorf("ids = %s, %s; want 1, 2", responses[0].ID, responses[1].ID)
	}
}

func TestServeStdoutContainsOnlyProtocolFrames(t *testing.T) {
	// Build a logger that writes to a buffer we can inspect, then
	// pipe the protocol output through a separate buffer. Any
	// handler that mistakenly writes to stdout would leave bytes
	// that don't round-trip through readFrames.
	var logBuf bytes.Buffer
	srv := New(markdown.New(t.TempDir()), log.New(&logBuf, "", 0))
	// A handler that does its own (buggy) write — useful as a
	// regression: if someone ever passes os.Stdout into a tool we
	// catch it here. The handler still returns normally so the
	// loop produces a real response, and the test asserts the
	// stray bytes were caught.
	stray := []byte("OOPS stdout pollution\n")
	srv.Register("echo", func(_ context.Context, _ store.Store, _ json.RawMessage) (any, error) {
		_, _ = logBuf.Write(stray) // simulate a real "log" to stderr-mock
		return map[string]string{"ok": "yes"}, nil
	})
	// type guard: methods table holds Handler funcs
	_ = srv.Methods()

	in := bytes.NewReader(frameRequest(t, "echo", 7, nil))
	var out bytes.Buffer
	if err := srv.Serve(context.Background(), in, &out); err != nil {
		t.Fatalf("Serve: %v", err)
	}

	// out should round-trip cleanly through readFrames — no stray
	// log bytes, no missing Content-Length.
	responses := readFrames(t, out.Bytes())
	if len(responses) != 1 {
		t.Fatalf("got %d responses, want 1", len(responses))
	}
	if string(responses[0].ID) != "7" {
		t.Errorf("id = %s, want 7", responses[0].ID)
	}
	// And logBuf carries the stray write, proving the handler ran
	// and the discipline kept it off stdout.
	if !bytes.Contains(logBuf.Bytes(), stray) {
		t.Errorf("logBuf does not contain expected stray write")
	}
}

// Register accepts the local handler signature so the test compiles
// even with the same Handler type the real tools use.
func (s *Server) registerSimple(t *testing.T, name string, fn func(json.RawMessage) (any, error)) {
	t.Helper()
	s.Register(name, func(_ context.Context, _ store.Store, params json.RawMessage) (any, error) {
		return fn(params)
	})
}

func TestServeReportsUnknownMethod(t *testing.T) {
	srv := newServer(t)
	in := bytes.NewReader(frameRequest(t, "nope", 1, nil))
	var out bytes.Buffer
	if err := srv.Serve(context.Background(), in, &out); err != nil {
		t.Fatalf("Serve: %v", err)
	}
	responses := readFrames(t, out.Bytes())
	if len(responses) != 1 || responses[0].Error == nil {
		t.Fatalf("want 1 error response, got %+v", responses)
	}
	if responses[0].Error.Code != CodeMethodNotFound {
		t.Errorf("code = %d, want %d", responses[0].Error.Code, CodeMethodNotFound)
	}
}

func TestServeReportsInvalidJSONRPCVersion(t *testing.T) {
	srv := newServer(t)
	body := `{"jsonrpc":"1.0","id":1,"method":"initialize"}`
	frame := fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(body), body)
	var out bytes.Buffer
	if err := srv.Serve(context.Background(), strings.NewReader(frame), &out); err != nil {
		t.Fatalf("Serve: %v", err)
	}
	responses := readFrames(t, out.Bytes())
	if len(responses) != 1 || responses[0].Error == nil {
		t.Fatalf("want 1 error response, got %+v", responses)
	}
	if responses[0].Error.Code != CodeInvalidRequest {
		t.Errorf("code = %d, want %d", responses[0].Error.Code, CodeInvalidRequest)
	}
}

func TestServeReportsParseError(t *testing.T) {
	srv := newServer(t)
	body := "{not json"
	frame := fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(body), body)
	var out bytes.Buffer
	if err := srv.Serve(context.Background(), strings.NewReader(frame), &out); err != nil {
		t.Fatalf("Serve: %v", err)
	}
	responses := readFrames(t, out.Bytes())
	if len(responses) != 1 || responses[0].Error == nil {
		t.Fatalf("want 1 error response, got %+v", responses)
	}
	if responses[0].Error.Code != CodeParseError {
		t.Errorf("code = %d, want %d", responses[0].Error.Code, CodeParseError)
	}
	if string(responses[0].ID) != "null" {
		t.Errorf("id = %s, want null", responses[0].ID)
	}
}

func TestServeNotificationsDoNotEmitResponses(t *testing.T) {
	srv := newServer(t)
	// id omitted entirely → notification.
	body := `{"jsonrpc":"2.0","method":"initialize"}`
	frame := fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(body), body)
	var out bytes.Buffer
	if err := srv.Serve(context.Background(), strings.NewReader(frame), &out); err != nil {
		t.Fatalf("Serve: %v", err)
	}
	if out.Len() != 0 {
		t.Fatalf("notification produced output: %q", out.String())
	}
}

func TestServeHandlerErrorMapsThroughWorkflow(t *testing.T) {
	srv := newServer(t)
	srv.registerSimple(t, "boom", func(_ json.RawMessage) (any, error) {
		return nil, fmt.Errorf("wrap: %w", workflow.ErrNotFound)
	})
	in := bytes.NewReader(frameRequest(t, "boom", 5, nil))
	var out bytes.Buffer
	if err := srv.Serve(context.Background(), in, &out); err != nil {
		t.Fatalf("Serve: %v", err)
	}
	responses := readFrames(t, out.Bytes())
	if len(responses) != 1 || responses[0].Error == nil {
		t.Fatalf("want 1 error response, got %+v", responses)
	}
	if responses[0].Error.Code != CodeNotFound {
		t.Errorf("code = %d, want %d", responses[0].Error.Code, CodeNotFound)
	}
}

func TestServeHandlerErrorMapsInvalidTransition(t *testing.T) {
	srv := newServer(t)
	srv.registerSimple(t, "boom", func(_ json.RawMessage) (any, error) {
		return nil, workflow.ErrInvalidTransition
	})
	in := bytes.NewReader(frameRequest(t, "boom", 8, nil))
	var out bytes.Buffer
	if err := srv.Serve(context.Background(), in, &out); err != nil {
		t.Fatalf("Serve: %v", err)
	}
	resp := readFrames(t, out.Bytes())[0]
	if resp.Error == nil || resp.Error.Code != CodeWorkflowViolation {
		t.Errorf("err = %+v, want CodeWorkflowViolation", resp.Error)
	}
}

func TestServeHandlerErrorMapsDoneUnmet(t *testing.T) {
	srv := newServer(t)
	srv.registerSimple(t, "boom", func(_ json.RawMessage) (any, error) {
		return nil, workflow.ErrDoneCriteriaUnmet
	})
	in := bytes.NewReader(frameRequest(t, "boom", 9, nil))
	var out bytes.Buffer
	if err := srv.Serve(context.Background(), in, &out); err != nil {
		t.Fatalf("Serve: %v", err)
	}
	resp := readFrames(t, out.Bytes())[0]
	if resp.Error == nil || resp.Error.Code != CodeWorkflowViolation {
		t.Errorf("err = %+v, want CodeWorkflowViolation", resp.Error)
	}
}

func TestServeHandlerErrorDefaultsToInternal(t *testing.T) {
	srv := newServer(t)
	srv.registerSimple(t, "boom", func(_ json.RawMessage) (any, error) {
		return nil, errors.New("something else")
	})
	in := bytes.NewReader(frameRequest(t, "boom", 10, nil))
	var out bytes.Buffer
	if err := srv.Serve(context.Background(), in, &out); err != nil {
		t.Fatalf("Serve: %v", err)
	}
	resp := readFrames(t, out.Bytes())[0]
	if resp.Error == nil || resp.Error.Code != CodeInternalError {
		t.Errorf("err = %+v, want CodeInternalError", resp.Error)
	}
}

func TestServeHandlerCanReturnPrebuiltError(t *testing.T) {
	srv := newServer(t)
	srv.registerSimple(t, "boom", func(_ json.RawMessage) (any, error) {
		return nil, &Error{Code: CodeInvalidParams, Message: "bad params"}
	})
	in := bytes.NewReader(frameRequest(t, "boom", 11, nil))
	var out bytes.Buffer
	if err := srv.Serve(context.Background(), in, &out); err != nil {
		t.Fatalf("Serve: %v", err)
	}
	resp := readFrames(t, out.Bytes())[0]
	if resp.Error == nil || resp.Error.Code != CodeInvalidParams {
		t.Errorf("err = %+v, want CodeInvalidParams", resp.Error)
	}
}

func TestServeMarshalFailureReportsInternalError(t *testing.T) {
	srv := newServer(t)
	srv.registerSimple(t, "bad", func(_ json.RawMessage) (any, error) {
		// json.Marshal cannot encode channels — forces the marshal
		// branch in handleOne.
		return make(chan int), nil
	})
	in := bytes.NewReader(frameRequest(t, "bad", 12, nil))
	var out bytes.Buffer
	if err := srv.Serve(context.Background(), in, &out); err != nil {
		t.Fatalf("Serve: %v", err)
	}
	resp := readFrames(t, out.Bytes())[0]
	if resp.Error == nil || resp.Error.Code != CodeInternalError {
		t.Errorf("err = %+v, want CodeInternalError", resp.Error)
	}
}

func TestServeRespectsContextCancellation(t *testing.T) {
	srv := newServer(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // pre-cancel
	var out bytes.Buffer
	err := srv.Serve(ctx, strings.NewReader(""), &out)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("err = %v, want context.Canceled", err)
	}
}

func TestServeIgnoresEmptyStream(t *testing.T) {
	srv := newServer(t)
	var out bytes.Buffer
	if err := srv.Serve(context.Background(), strings.NewReader(""), &out); err != nil {
		t.Errorf("empty stream returned %v, want nil", err)
	}
	if out.Len() != 0 {
		t.Errorf("empty stream produced output: %q", out.String())
	}
}

func TestReadFrameRejectsMalformedHeader(t *testing.T) {
	br := bufio.NewReader(strings.NewReader("bad header\r\n\r\n{}"))
	_, err := readFrame(br)
	if err == nil {
		t.Fatalf("want error, got nil")
	}
	if !strings.Contains(err.Error(), "malformed header") {
		t.Errorf("err = %v, want malformed header", err)
	}
}

func TestReadFrameRejectsMissingContentLength(t *testing.T) {
	br := bufio.NewReader(strings.NewReader("Content-Type: application/json\r\n\r\n{}"))
	_, err := readFrame(br)
	if err == nil {
		t.Fatalf("want error, got nil")
	}
	if !strings.Contains(err.Error(), "missing Content-Length") {
		t.Errorf("err = %v, want missing Content-Length", err)
	}
}

func TestReadFrameRejectsBadContentLength(t *testing.T) {
	br := bufio.NewReader(strings.NewReader("Content-Length: nope\r\n\r\n{}"))
	_, err := readFrame(br)
	if err == nil {
		t.Fatalf("want error, got nil")
	}
	if !strings.Contains(err.Error(), "bad Content-Length") {
		t.Errorf("err = %v, want bad Content-Length", err)
	}
}

func TestReadFrameRejectsShortBody(t *testing.T) {
	br := bufio.NewReader(strings.NewReader("Content-Length: 50\r\n\r\n{}"))
	_, err := readFrame(br)
	if err == nil {
		t.Fatalf("want error, got nil")
	}
	if !strings.Contains(err.Error(), "read body") {
		t.Errorf("err = %v, want short-body error", err)
	}
}

func TestWriteFrameRoundTrip(t *testing.T) {
	var buf bytes.Buffer
	if err := writeFrame(&buf, []byte(`{"k":"v"}`)); err != nil {
		t.Fatalf("writeFrame: %v", err)
	}
	br := bufio.NewReader(&buf)
	got, err := readFrame(br)
	if err != nil {
		t.Fatalf("readFrame: %v", err)
	}
	if string(got) != `{"k":"v"}` {
		t.Errorf("payload = %q, want %q", got, `{"k":"v"}`)
	}
}

func TestErrorErrorFormatsCodeMessage(t *testing.T) {
	e := &Error{Code: 42, Message: "boom"}
	if got := e.Error(); got != "42: boom" {
		t.Errorf("Error() = %q, want %q", got, "42: boom")
	}
	var nilErr *Error
	if got := nilErr.Error(); got != "<nil>" {
		t.Errorf("nil Error() = %q, want <nil>", got)
	}
}

func TestNewDefaultsLoggerToDiscard(t *testing.T) {
	srv := New(markdown.New(t.TempDir()), nil)
	if srv.Logger == nil {
		t.Fatal("Logger nil after New(_, nil)")
	}
	// Writing should be safe and silent.
	srv.Logger.Print("ignored")
}

func TestDecodeParamsHandlesNullAndEmpty(t *testing.T) {
	var v struct{}
	if err := decodeParams(nil, &v); err != nil {
		t.Errorf("nil params: %v", err)
	}
	if err := decodeParams(json.RawMessage("null"), &v); err != nil {
		t.Errorf("null params: %v", err)
	}
}

func TestDecodeParamsRejectsBadJSON(t *testing.T) {
	var v struct {
		N int `json:"n"`
	}
	err := decodeParams(json.RawMessage(`{"n":"oops"}`), &v)
	apiErr, ok := err.(*Error)
	if !ok {
		t.Fatalf("err = %T, want *Error", err)
	}
	if apiErr.Code != CodeInvalidParams {
		t.Errorf("code = %d, want %d", apiErr.Code, CodeInvalidParams)
	}
}
