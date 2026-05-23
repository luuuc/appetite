package cli

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// frameInitialize builds a Content-Length-framed `initialize` request
// matching what the MCP server expects on stdin.
func frameInitialize(t *testing.T, id int) []byte {
	t.Helper()
	body := fmt.Sprintf(`{"jsonrpc":"2.0","id":%d,"method":"initialize"}`, id)
	return []byte(fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(body), body))
}

// readOneFrame consumes a single Content-Length frame from r and
// returns the JSON payload. Tests use it to peel one response off
// stdout without depending on the mcp package's internal helpers.
func readOneFrame(t *testing.T, r *bufio.Reader) []byte {
	t.Helper()
	contentLength := -1
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			t.Fatalf("read header: %v", err)
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			break
		}
		colon := strings.IndexByte(line, ':')
		if colon < 0 {
			t.Fatalf("bad header %q", line)
		}
		if strings.EqualFold(strings.TrimSpace(line[:colon]), "Content-Length") {
			n, perr := strconv.Atoi(strings.TrimSpace(line[colon+1:]))
			if perr != nil {
				t.Fatalf("bad content-length: %v", perr)
			}
			contentLength = n
		}
	}
	if contentLength < 0 {
		t.Fatalf("no content-length")
	}
	buf := make([]byte, contentLength)
	if _, err := io.ReadFull(r, buf); err != nil {
		t.Fatalf("read body: %v", err)
	}
	return buf
}

func TestMCPCommandHandlesInitialize(t *testing.T) {
	dir := t.TempDir()
	var stdin bytes.Buffer
	stdin.Write(frameInitialize(t, 1))

	prev := mcpStdin
	mcpStdin = &stdin
	t.Cleanup(func() { mcpStdin = prev })

	var stdout, stderr bytes.Buffer
	env := Env{Dir: dir, Stdout: &stdout, Stderr: &stderr}
	if err := runMCP(env, nil); err != nil {
		t.Fatalf("runMCP: %v", err)
	}

	br := bufio.NewReader(bytes.NewReader(stdout.Bytes()))
	payload := readOneFrame(t, br)
	var resp struct {
		ID     json.RawMessage `json:"id"`
		Result struct {
			ProtocolVersion string `json:"protocolVersion"`
		} `json:"result"`
	}
	if err := json.Unmarshal(payload, &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if string(resp.ID) != "1" {
		t.Errorf("id = %s, want 1", resp.ID)
	}
	if resp.Result.ProtocolVersion == "" {
		t.Errorf("protocolVersion empty")
	}
}

func TestMCPCommandRefusesPositionalArgs(t *testing.T) {
	dir := t.TempDir()
	var stdout, stderr bytes.Buffer
	env := Env{Dir: dir, Stdout: &stdout, Stderr: &stderr}
	err := runMCP(env, []string{"extra"})
	if err == nil || !strings.Contains(err.Error(), "no positional") {
		t.Errorf("err = %v, want no positional", err)
	}
}

func TestMCPCommandRejectsUnknownFlag(t *testing.T) {
	dir := t.TempDir()
	var stdout, stderr bytes.Buffer
	env := Env{Dir: dir, Stdout: &stdout, Stderr: &stderr}
	if err := runMCP(env, []string{"--bogus"}); err == nil {
		t.Errorf("want error for unknown flag")
	}
}

func TestMCPCommandLogsToFileViaEnvVar(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "mcp.log")
	t.Setenv("APPETITE_MCP_LOG", logPath)

	dir := t.TempDir()
	var stdin bytes.Buffer
	// Send an unknown method so the server logs the failure.
	body := `{"jsonrpc":"2.0","id":1,"method":"nope"}`
	frame := fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(body), body)
	stdin.WriteString(frame)

	prev := mcpStdin
	mcpStdin = &stdin
	t.Cleanup(func() { mcpStdin = prev })

	var stdout, stderr bytes.Buffer
	env := Env{Dir: dir, Stdout: &stdout, Stderr: &stderr}
	if err := runMCP(env, nil); err != nil {
		t.Fatalf("runMCP: %v", err)
	}
	if _, err := os.Stat(logPath); err != nil {
		t.Errorf("log file not created: %v", err)
	}
	// Stderr must stay clean when a log file is set.
	if stderr.Len() != 0 {
		t.Errorf("stderr leaked when APPETITE_MCP_LOG set: %q", stderr.String())
	}
}

func TestBuildMCPLoggerRejectsUnopenableFile(t *testing.T) {
	// Point APPETITE_MCP_LOG at a path whose parent doesn't exist,
	// forcing OpenFile to fail.
	t.Setenv("APPETITE_MCP_LOG", filepath.Join(t.TempDir(), "missing-dir", "mcp.log"))
	env := Env{Stderr: io.Discard}
	_, _, err := buildMCPLogger(env)
	if err == nil {
		t.Fatal("want error opening log path with missing parent")
	}
}

func TestMCPCommandReturnsErrorOnLogOpenFailure(t *testing.T) {
	t.Setenv("APPETITE_MCP_LOG", filepath.Join(t.TempDir(), "missing-dir", "mcp.log"))
	dir := t.TempDir()
	var stdout, stderr bytes.Buffer
	env := Env{Dir: dir, Stdout: &stdout, Stderr: &stderr}
	err := runMCP(env, nil)
	if err == nil {
		t.Fatal("want error from runMCP when log file cannot open")
	}
	if !strings.Contains(err.Error(), "open log") {
		t.Errorf("err = %v, want 'open log'", err)
	}
}

func TestMCPCommandSucceedsWhenStdinClosesImmediately(t *testing.T) {
	dir := t.TempDir()
	prev := mcpStdin
	mcpStdin = strings.NewReader("")
	t.Cleanup(func() { mcpStdin = prev })

	var stdout, stderr bytes.Buffer
	env := Env{Dir: dir, Stdout: &stdout, Stderr: &stderr}
	if err := runMCP(env, nil); err != nil && !errors.Is(err, io.EOF) {
		t.Fatalf("runMCP on empty stdin: %v", err)
	}
	if stdout.Len() != 0 {
		t.Errorf("empty stdin produced stdout: %q", stdout.String())
	}
}
