package mcp

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// Request is one inbound JSON-RPC 2.0 frame.
//
// Notifications (no id) are accepted — handlers run, no response is
// written. Requests with a non-null id get a response (result or
// error) keyed by the same id.
type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// IsNotification reports whether the request omitted (or nulled)
// its id field — JSON-RPC 2.0 says no response should be sent.
func (r Request) IsNotification() bool {
	if len(r.ID) == 0 {
		return true
	}
	return string(r.ID) == "null"
}

// Response is one outbound JSON-RPC 2.0 frame. Exactly one of Result
// or Error is set when the response is sent.
type Response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *Error          `json:"error,omitempty"`
}

// Error is the JSON-RPC 2.0 error object.
type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// Error implements the error interface so handlers can return *Error
// directly. The text is "code: message" so it round-trips through
// wrapping without losing the code.
func (e *Error) Error() string {
	if e == nil {
		return "<nil>"
	}
	return fmt.Sprintf("%d: %s", e.Code, e.Message)
}

// Standard JSON-RPC 2.0 error codes plus the Appetite-specific codes
// that mirror the CLI exit codes (see package doc).
const (
	CodeParseError     = -32700
	CodeInvalidRequest = -32600
	CodeMethodNotFound = -32601
	CodeInvalidParams  = -32602
	CodeInternalError  = -32603

	// CodeWorkflowViolation mirrors CLI exit 2.
	CodeWorkflowViolation = -32002
	// CodeNotFound mirrors CLI exit 3.
	CodeNotFound = -32003
)

// errFrameClosed signals a clean end-of-stream on the input side. The
// server loop treats it as "stop reading" rather than as a protocol
// error.
var errFrameClosed = errors.New("mcp: input stream closed")

// readFrame reads one JSON-RPC frame from r. The frame is preceded by
// HTTP-style headers (currently only Content-Length is required),
// terminated by a blank line, followed by exactly Content-Length
// bytes of JSON.
//
// Returns errFrameClosed when r returns io.EOF before any header
// bytes have been read (clean shutdown). Header parse errors and
// truncated bodies are surfaced as ordinary errors — the server
// turns them into JSON-RPC parse errors when possible.
func readFrame(r *bufio.Reader) ([]byte, error) {
	contentLength := -1
	headerSeen := false
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			if errors.Is(err, io.EOF) && !headerSeen && line == "" {
				return nil, errFrameClosed
			}
			return nil, fmt.Errorf("mcp: read header: %w", err)
		}
		headerSeen = true
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			break
		}
		colon := strings.IndexByte(line, ':')
		if colon < 0 {
			return nil, fmt.Errorf("mcp: malformed header %q", line)
		}
		name := strings.TrimSpace(line[:colon])
		value := strings.TrimSpace(line[colon+1:])
		if strings.EqualFold(name, "Content-Length") {
			n, err := strconv.Atoi(value)
			if err != nil || n < 0 {
				return nil, fmt.Errorf("mcp: bad Content-Length %q", value)
			}
			contentLength = n
		}
		// Other headers (Content-Type) are accepted and ignored.
	}
	if contentLength < 0 {
		return nil, errors.New("mcp: missing Content-Length header")
	}
	buf := make([]byte, contentLength)
	if _, err := io.ReadFull(r, buf); err != nil {
		return nil, fmt.Errorf("mcp: read body (%d bytes): %w", contentLength, err)
	}
	return buf, nil
}

// writeFrame writes one JSON payload prefixed by the required
// Content-Length header. The payload is written exactly once in a
// single Write call so partial writes do not interleave with another
// frame.
func writeFrame(w io.Writer, payload []byte) error {
	header := fmt.Sprintf("Content-Length: %d\r\n\r\n", len(payload))
	buf := make([]byte, 0, len(header)+len(payload))
	buf = append(buf, header...)
	buf = append(buf, payload...)
	if _, err := w.Write(buf); err != nil {
		return fmt.Errorf("mcp: write frame: %w", err)
	}
	return nil
}

// writeResponse marshals resp and frames it onto w.
func writeResponse(w io.Writer, resp Response) error {
	data, err := json.Marshal(resp)
	if err != nil {
		return fmt.Errorf("mcp: marshal response: %w", err)
	}
	return writeFrame(w, data)
}
