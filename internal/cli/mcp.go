package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/luuuc/appetite/internal/mcp"
)

func init() {
	register(Command{
		Name:     "mcp",
		Synopsis: "run the MCP stdio JSON-RPC server",
		Run:      runMCP,
	})
}

// mcpStdin is overridden in tests to feed a scripted byte stream.
// Production callers leave it nil, which falls back to os.Stdin.
var mcpStdin io.Reader

// runMCP boots the MCP server pointed at env.Dir's store. The
// transport channel (stdout) is reserved for protocol frames — every
// log line lands on env.Stderr (or APPETITE_MCP_LOG if set), per the
// pitch's logging discipline.
func runMCP(env Env, args []string) error {
	fs := flag.NewFlagSet("mcp", flag.ContinueOnError)
	fs.SetOutput(env.Stderr)
	dir := fs.String("dir", "", "path to the .appetite/ directory (overrides the top-level --dir)")
	if err := fs.Parse(reorderFlags(fs, args)); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("mcp: takes no positional arguments")
	}
	if *dir != "" {
		env.Dir = *dir
	}

	logger, closer, err := buildMCPLogger(env)
	if err != nil {
		return err
	}
	if closer != nil {
		defer func() { _ = closer.Close() }()
	}

	in := mcpStdin
	if in == nil {
		in = os.Stdin
	}

	srv := mcp.New(openStore(env), logger)
	return srv.Serve(context.Background(), in, env.Stdout)
}

// buildMCPLogger returns the logger Serve writes to. APPETITE_MCP_LOG
// promotes a file path; anything falsy keeps the default (env.Stderr).
// The returned io.Closer is non-nil only when a file was opened, so
// the caller can close it without nil-checking the path twice.
func buildMCPLogger(env Env) (*log.Logger, io.Closer, error) {
	if path := os.Getenv("APPETITE_MCP_LOG"); path != "" {
		f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
		if err != nil {
			return nil, nil, fmt.Errorf("mcp: open log %s: %w", path, err)
		}
		return log.New(f, "appetite-mcp ", log.LstdFlags), f, nil
	}
	return log.New(env.Stderr, "appetite-mcp ", log.LstdFlags), nil, nil
}
