package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// testEnv constructs an Env with buffered stdio rooted at dir.
func testEnv(dir string) (Env, *bytes.Buffer, *bytes.Buffer) {
	var out, errBuf bytes.Buffer
	return Env{Dir: dir, Stdout: &out, Stderr: &errBuf}, &out, &errBuf
}

func TestMain_NoArgs_PrintsUsageAndExits1(t *testing.T) {
	env, _, errBuf := testEnv(t.TempDir())
	if code := Main(nil, env); code != 1 {
		t.Fatalf("exit code: got %d want 1", code)
	}
	if !strings.Contains(errBuf.String(), "Usage: appetite") {
		t.Fatalf("expected usage on stderr, got %q", errBuf.String())
	}
}

func TestMain_DashH_PrintsUsageAndExits0(t *testing.T) {
	env, out, _ := testEnv(t.TempDir())
	if code := Main([]string{"-h"}, env); code != 0 {
		t.Fatalf("exit code: got %d want 0", code)
	}
	if !strings.Contains(out.String(), "Usage: appetite") {
		t.Fatalf("expected usage on stdout, got %q", out.String())
	}
}

func TestMain_UnknownCommand_Exits1(t *testing.T) {
	env, _, errBuf := testEnv(t.TempDir())
	if code := Main([]string{"blarg"}, env); code != 1 {
		t.Fatalf("exit code: got %d want 1", code)
	}
	if !strings.Contains(errBuf.String(), `unknown command "blarg"`) {
		t.Fatalf("expected unknown-command message, got %q", errBuf.String())
	}
}

func TestMain_Version(t *testing.T) {
	env, out, _ := testEnv(t.TempDir())
	if code := Main([]string{"version"}, env); code != 0 {
		t.Fatalf("exit code: got %d want 0", code)
	}
	if strings.TrimSpace(out.String()) == "" {
		t.Fatalf("expected version on stdout, got empty")
	}
}

func TestMain_Init_CreatesSkeleton(t *testing.T) {
	dir := filepath.Join(t.TempDir(), ".appetite")
	env, out, _ := testEnv(dir)

	if code := Main([]string{"init"}, env); code != 0 {
		t.Fatalf("first init exit code: got %d want 0", code)
	}
	if !strings.Contains(out.String(), "initialized") {
		t.Fatalf("expected 'initialized' on stdout, got %q", out.String())
	}
	for _, sub := range initSubdirs {
		path := filepath.Join(dir, sub)
		if !isDir(t, path) {
			t.Errorf("missing subdir %s", path)
		}
	}
}

func TestMain_Init_Idempotent(t *testing.T) {
	dir := filepath.Join(t.TempDir(), ".appetite")
	env, _, _ := testEnv(dir)

	if code := Main([]string{"init"}, env); code != 0 {
		t.Fatalf("first init exit code: got %d want 0", code)
	}

	env2, out2, _ := testEnv(dir)
	if code := Main([]string{"init"}, env2); code != 0 {
		t.Fatalf("second init exit code: got %d want 0", code)
	}
	if !strings.Contains(out2.String(), "already initialized") {
		t.Fatalf("expected 'already initialized' on stdout, got %q", out2.String())
	}
}

func TestMain_Init_TakesNoArgs(t *testing.T) {
	dir := filepath.Join(t.TempDir(), ".appetite")
	env, _, _ := testEnv(dir)
	if code := Main([]string{"init", "stray"}, env); code != 1 {
		t.Fatalf("init with stray arg: got %d want 1", code)
	}
}

func TestMain_DashDir_OverridesEnvDir(t *testing.T) {
	custom := filepath.Join(t.TempDir(), "custom")
	env, _, _ := testEnv(".appetite") // wrong default; --dir should win
	if code := Main([]string{"--dir", custom, "init"}, env); code != 0 {
		t.Fatalf("init with --dir: got %d want 0", code)
	}
	if !isDir(t, filepath.Join(custom, "signals/raw")) {
		t.Errorf("--dir custom skeleton missing")
	}
}

func TestRegister_PanicsOnDuplicate(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic on duplicate command")
		}
	}()
	register(Command{Name: "init"}) // already registered
}

func isDir(t *testing.T, path string) bool {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}
