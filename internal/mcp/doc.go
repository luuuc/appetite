// Package mcp wraps internal/workflow as a JSON-RPC 2.0 server over
// stdio. It is the second surface of Appetite, alongside internal/cli.
// MCP tools are 1:1 wrappers around workflow functions — every tool
// is "translate JSON params → workflow request, call workflow → render
// result or error." No workflow rules live here.
//
// Transport: JSON-RPC 2.0 framed with HTTP-style Content-Length
// headers, per .doc/definition/07-mcp-and-cli.md and the MCP
// transport spec. Stdout carries protocol frames only; logging goes
// to a separate writer (stderr by default; APPETITE_MCP_LOG enables
// file logging when set by the CLI subcommand).
//
// Error mapping mirrors the CLI's exit codes:
//
//	workflow.ErrInvalidTransition   → -32002 (matches CLI exit 2)
//	workflow.ErrDoneCriteriaUnmet   → -32002 (matches CLI exit 2)
//	workflow.ErrNotFound            → -32003 (matches CLI exit 3)
//	JSON parse / bad params         → -32700 / -32602 (JSON-RPC stdlib)
//	other                           → -32603 (internal, matches CLI 1)
package mcp
