package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"

	"github.com/luuuc/appetite/internal/store"
)

// Handler is one tool implementation. It receives the raw JSON params
// (possibly empty) and returns a value to marshal into the response's
// `result` field. Returning a *Error sends that error verbatim;
// returning any other error runs it through errorFromWorkflow.
type Handler func(ctx context.Context, s store.Store, params json.RawMessage) (any, error)

// Server is a stateless JSON-RPC 2.0 dispatcher. Construct one per
// process via New; Serve reads frames from in, writes responses to
// out, and logs to logger (stderr by default — never the protocol
// channel). The store is injected so tests can swap in an in-memory
// adapter without touching the filesystem.
type Server struct {
	Store   store.Store
	Logger  *log.Logger
	methods map[string]Handler
}

// New builds a Server with the default tool dispatch table. The
// Logger defaults to a stderr logger; callers can override before
// calling Serve.
func New(s store.Store, logger *log.Logger) *Server {
	if logger == nil {
		logger = log.New(io.Discard, "", 0)
	}
	srv := &Server{Store: s, Logger: logger}
	srv.methods = defaultMethods(srv)
	return srv
}

// Methods exposes the dispatch table for tests. Callers must not
// mutate the returned map.
func (s *Server) Methods() map[string]Handler { return s.methods }

// Register adds or overrides a handler. Primarily for tests; the
// default table is populated by New so the map is never nil here.
func (s *Server) Register(method string, h Handler) {
	s.methods[method] = h
}

// Serve reads framed JSON-RPC requests from in and writes framed
// responses to out until in returns EOF or ctx is cancelled. Each
// request is handled synchronously — the JSON-RPC spec allows
// batching but MCP stdio does not, so single-request frames keep the
// implementation honest. Returns nil on a clean EOF.
func (s *Server) Serve(ctx context.Context, in io.Reader, out io.Writer) error {
	br := bufio.NewReader(in)
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		payload, err := readFrame(br)
		if err != nil {
			if errors.Is(err, errFrameClosed) {
				return nil
			}
			// Frame-level parse error — emit a parse-error response
			// with null id (we couldn't extract a real one) and keep
			// going if the stream is still alive.
			s.Logger.Printf("frame error: %v", err)
			resp := Response{
				JSONRPC: "2.0",
				ID:      json.RawMessage("null"),
				Error:   &Error{Code: CodeParseError, Message: err.Error()},
			}
			if werr := writeResponse(out, resp); werr != nil {
				return werr
			}
			// Header errors leave the body unconsumed; bail out to
			// avoid an infinite error loop on the same bytes.
			return err
		}
		if err := s.handleOne(ctx, out, payload); err != nil {
			return err
		}
	}
}

// handleOne parses a single frame payload and dispatches it. Returns
// an error only when the output stream itself fails — handler errors
// turn into JSON-RPC error responses, not loop exits.
func (s *Server) handleOne(ctx context.Context, out io.Writer, payload []byte) error {
	var req Request
	if err := json.Unmarshal(payload, &req); err != nil {
		return writeResponse(out, Response{
			JSONRPC: "2.0",
			ID:      json.RawMessage("null"),
			Error:   &Error{Code: CodeParseError, Message: err.Error()},
		})
	}
	if req.JSONRPC != "2.0" {
		if req.IsNotification() {
			return nil
		}
		return writeResponse(out, Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   &Error{Code: CodeInvalidRequest, Message: fmt.Sprintf("unsupported jsonrpc version %q", req.JSONRPC)},
		})
	}

	handler, ok := s.methods[req.Method]
	if !ok {
		if req.IsNotification() {
			return nil
		}
		return writeResponse(out, Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   &Error{Code: CodeMethodNotFound, Message: fmt.Sprintf("unknown method %q", req.Method)},
		})
	}

	result, herr := handler(ctx, s.Store, req.Params)
	if req.IsNotification() {
		if herr != nil {
			s.Logger.Printf("notification %s failed: %v", req.Method, herr)
		}
		return nil
	}
	if herr != nil {
		return writeResponse(out, Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   toError(herr),
		})
	}

	encoded, err := json.Marshal(result)
	if err != nil {
		return writeResponse(out, Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   &Error{Code: CodeInternalError, Message: fmt.Sprintf("marshal result: %v", err)},
		})
	}
	return writeResponse(out, Response{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  encoded,
	})
}

// toError unwraps *Error if the handler returned one (so the
// pre-built code/message survives) and otherwise routes through
// errorFromWorkflow.
func toError(err error) *Error {
	var apiErr *Error
	if errors.As(err, &apiErr) {
		return apiErr
	}
	return errorFromWorkflow(err)
}

