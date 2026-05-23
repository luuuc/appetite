package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/luuuc/appetite/internal/store"
)

// defaultMethods returns the tool dispatch table the Server wires up
// in New. Individual tool handlers live in their own files
// (read.go, signal.go, …) and call back through the Server for the
// shared logger/store dependencies.
//
// One method is universal: `initialize`. The MCP spec mandates it as
// the first call from a client; the implementation here is the
// minimum that satisfies the handshake without committing to the
// full capabilities object (deferred per the pitch).
func defaultMethods(s *Server) map[string]Handler {
	return map[string]Handler{
		"initialize":          handleInitialize,
		"appetite_status":     handleStatus,
		"appetite_pitch_list": handlePitchList,
		"appetite_cycle_list": handleCycleList,
		"appetite_signal":     handleSignal,
		"appetite_shape":      handleShape,
		"appetite_cycle_new":  handleCycleNew,
		"appetite_bet":        handleBet,
		"appetite_pass":       handlePass,
		"appetite_cut":        handleCut,
		"appetite_hill":       handleHill,
		"appetite_cooldown":   handleCooldown,
	}
}

// initializeResult is the smallest viable handshake response. The
// protocol version mirrors the MCP draft Appetite targets; the
// server info is informational only.
type initializeResult struct {
	ProtocolVersion string            `json:"protocolVersion"`
	ServerInfo      serverInfo        `json:"serverInfo"`
	Capabilities    map[string]any    `json:"capabilities"`
	Instructions    string            `json:"instructions,omitempty"`
	Meta            map[string]string `json:"_meta,omitempty"`
}

type serverInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// ProtocolVersion is the MCP protocol version this server speaks.
// Exposed so the CLI subcommand can echo it in --help if needed.
const ProtocolVersion = "2025-03-26"

// ServerName is the static name reported in initialize responses.
const ServerName = "appetite"

func handleInitialize(_ context.Context, _ store.Store, _ json.RawMessage) (any, error) {
	return initializeResult{
		ProtocolVersion: ProtocolVersion,
		ServerInfo: serverInfo{
			Name:    ServerName,
			Version: "0.1",
		},
		Capabilities: map[string]any{
			// Appetite's tools are bare JSON-RPC methods, not the
			// MCP-style tools/list + tools/call pair (pitch decision:
			// minimal handshake, no tool registry). Capabilities
			// stays empty so clients understand that.
		},
	}, nil
}

// decodeParams unmarshals raw into v, surfacing a CodeInvalidParams
// JSON-RPC error rather than a generic internal error. Handlers
// call this on every typed-params tool.
func decodeParams(raw json.RawMessage, v any) error {
	if len(raw) == 0 || string(raw) == "null" {
		// Caller passes a zero-valued v; nothing to copy in.
		return nil
	}
	if err := json.Unmarshal(raw, v); err != nil {
		return &Error{Code: CodeInvalidParams, Message: fmt.Sprintf("invalid params: %v", err)}
	}
	return nil
}
