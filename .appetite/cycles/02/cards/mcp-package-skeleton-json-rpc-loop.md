---
slug: mcp-package-skeleton-json-rpc-loop
pitch: 02-01-mcp-server
cycle: "02"
hill: downhill
progress: 100
hill_updated_at: 2026-05-22T18:23:46Z
---
`internal/mcp/server.go` with stdio JSON-RPC 2.0 framing (Content-Length headers per the MCP transport spec), method dispatch, typed error mapping in `errors.go`. Logging to stderr only; a test asserts stdout contains only protocol frames.
