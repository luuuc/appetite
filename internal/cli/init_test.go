package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// withTempCommandsSource swaps commandsSourceDir to a fresh temp dir
// holding the two listed slash command files. Returns the temp dir
// path. Tests rely on this to avoid depending on the repo-root
// commands/ tree.
func withTempCommandsSource(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatalf("seed %s: %v", name, err)
		}
	}
	prev := commandsSourceDir
	commandsSourceDir = dir
	t.Cleanup(func() { commandsSourceDir = prev })
	return dir
}

func TestInitInstallsCommandsForClaude(t *testing.T) {
	withTempCommandsSource(t, map[string]string{
		"signal.md": "# signal\n",
		"shape.md":  "# shape\n",
	})
	cwd := chdir(t, t.TempDir())
	_ = cwd

	dir := filepath.Join(".", ".appetite")
	code, out, _ := runMain(t, dir, "init", "--commands", "claude")
	if code != 0 {
		t.Fatalf("code %d", code)
	}
	wantTarget := filepath.Join(".claude", "commands", "appetite")
	if !strings.Contains(out, "installed 2 commands into "+wantTarget) {
		t.Errorf("stdout = %q", out)
	}
	if _, err := os.Stat(filepath.Join(wantTarget, "signal.md")); err != nil {
		t.Errorf("signal.md missing under appetite/: %v", err)
	}
	// `.claude/commands/` itself stays free of appetite files so it
	// doesn't collide with bootstrap commands the operator already
	// has installed there.
	if _, err := os.Stat(filepath.Join(".claude", "commands", "signal.md")); !os.IsNotExist(err) {
		t.Errorf("top-level .claude/commands/signal.md should not exist (err=%v)", err)
	}
	if _, err := os.Stat(filepath.Join(".claude", "commands", "shape.md")); !os.IsNotExist(err) {
		t.Errorf("top-level .claude/commands/shape.md should not exist (err=%v)", err)
	}
}

func TestInitCommandsIdempotentOnIdenticalContent(t *testing.T) {
	withTempCommandsSource(t, map[string]string{"signal.md": "# signal\n"})
	chdir(t, t.TempDir())

	dir := filepath.Join(".", ".appetite")
	if code, _, _ := runMain(t, dir, "init", "--commands", "claude"); code != 0 {
		t.Fatalf("first install: code %d", code)
	}
	code, out, _ := runMain(t, dir, "init", "--commands", "claude")
	if code != 0 {
		t.Fatalf("second install: code %d", code)
	}
	if !strings.Contains(out, "already up-to-date") {
		t.Errorf("expected skip message, got %q", out)
	}
}

func TestInitCommandsRejectsDivergentContentWithoutForce(t *testing.T) {
	withTempCommandsSource(t, map[string]string{"signal.md": "# signal v2\n"})
	chdir(t, t.TempDir())

	// Pre-place a divergent file in the target.
	target := filepath.Join(".claude", "commands", "appetite")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatalf("mkdir target: %v", err)
	}
	if err := os.WriteFile(filepath.Join(target, "signal.md"), []byte("# signal v1\n"), 0o644); err != nil {
		t.Fatalf("seed target: %v", err)
	}

	dir := filepath.Join(".", ".appetite")
	code, _, errOut := runMain(t, dir, "init", "--commands", "claude")
	if code != 2 {
		t.Errorf("code = %d, want 2", code)
	}
	if !strings.Contains(errOut, "diverged") {
		t.Errorf("stderr = %q", errOut)
	}
	if !strings.Contains(errOut, "signal.md") {
		t.Errorf("stderr missing file name: %q", errOut)
	}
}

func TestInitCommandsForceOverwritesDivergent(t *testing.T) {
	withTempCommandsSource(t, map[string]string{"signal.md": "# signal v2\n"})
	chdir(t, t.TempDir())

	target := filepath.Join(".claude", "commands", "appetite")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatalf("mkdir target: %v", err)
	}
	if err := os.WriteFile(filepath.Join(target, "signal.md"), []byte("# signal v1\n"), 0o644); err != nil {
		t.Fatalf("seed target: %v", err)
	}

	dir := filepath.Join(".", ".appetite")
	code, _, _ := runMain(t, dir, "init", "--commands", "claude", "--force")
	if code != 0 {
		t.Fatalf("code = %d, want 0", code)
	}
	data, err := os.ReadFile(filepath.Join(target, "signal.md"))
	if err != nil {
		t.Fatalf("read after force: %v", err)
	}
	if string(data) != "# signal v2\n" {
		t.Errorf("contents = %q, want v2", data)
	}
}

func TestInitCommandsRejectsUnknownTool(t *testing.T) {
	withTempCommandsSource(t, map[string]string{"signal.md": "# x\n"})
	chdir(t, t.TempDir())

	dir := filepath.Join(".", ".appetite")
	code, _, errOut := runMain(t, dir, "init", "--commands", "rabbithole")
	if code != 2 {
		t.Errorf("code = %d, want 2", code)
	}
	if !strings.Contains(errOut, "unknown commands tool") {
		t.Errorf("stderr = %q", errOut)
	}
}

func TestInitWithoutCommandsFlagStillInitializes(t *testing.T) {
	chdir(t, t.TempDir())
	dir := filepath.Join(".", ".appetite")
	code, out, _ := runMain(t, dir, "init")
	if code != 0 {
		t.Fatalf("code %d", code)
	}
	if !strings.Contains(out, "initialized") {
		t.Errorf("out = %q", out)
	}
}

func TestInitIsIdempotentOnSkeleton(t *testing.T) {
	chdir(t, t.TempDir())
	dir := filepath.Join(".", ".appetite")
	runMain(t, dir, "init")
	_, out, _ := runMain(t, dir, "init")
	if !strings.Contains(out, "already initialized") {
		t.Errorf("second init out = %q", out)
	}
}

func TestInitCommandsErrorsWhenSourceMissing(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "no-such-dir")
	prev := commandsSourceDir
	commandsSourceDir = missing
	t.Cleanup(func() { commandsSourceDir = prev })

	chdir(t, t.TempDir())
	dir := filepath.Join(".", ".appetite")
	code, _, errOut := runMain(t, dir, "init", "--commands", "claude")
	if code == 0 {
		t.Errorf("expected non-zero exit, stderr=%q", errOut)
	}
}

func TestCopyCommandFileErrorsOnMissingSource(t *testing.T) {
	dir := t.TempDir()
	dst := filepath.Join(dir, "out.md")
	_, err := copyCommandFile(filepath.Join(dir, "missing.md"), dst, false)
	if err == nil {
		t.Fatal("want error for missing source")
	}
}

func TestCopyCommandFileErrorsWhenDstUnwritable(t *testing.T) {
	srcDir := t.TempDir()
	src := filepath.Join(srcDir, "x.md")
	if err := os.WriteFile(src, []byte("x"), 0o644); err != nil {
		t.Fatalf("seed src: %v", err)
	}
	// Read-only parent dir → rename fails.
	roDir := filepath.Join(t.TempDir(), "ro")
	if err := os.MkdirAll(roDir, 0o500); err != nil {
		t.Fatalf("mkdir ro: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(roDir, 0o755) })

	if _, err := copyCommandFile(src, filepath.Join(roDir, "x.md"), false); err == nil {
		t.Errorf("want error writing under read-only dir")
	}
}

func TestCopyCommandFileSurfacesUnreadableDst(t *testing.T) {
	srcDir := t.TempDir()
	src := filepath.Join(srcDir, "x.md")
	if err := os.WriteFile(src, []byte("x"), 0o644); err != nil {
		t.Fatalf("seed src: %v", err)
	}
	// Create a dst path whose parent exists as a regular file (not
	// a directory), so reading dst returns a non-ErrNotExist error.
	parent := filepath.Join(t.TempDir(), "file-not-dir")
	if err := os.WriteFile(parent, []byte("blocker"), 0o644); err != nil {
		t.Fatalf("seed parent: %v", err)
	}
	if _, err := copyCommandFile(src, filepath.Join(parent, "x.md"), false); err == nil {
		t.Errorf("want error reading inside a regular file parent")
	}
}

func TestInitRejectsPositionalArgs(t *testing.T) {
	dir := t.TempDir()
	code, _, _ := runMain(t, dir, "init", "extra")
	if code == 0 {
		t.Errorf("expected non-zero exit")
	}
}

// chdir temporarily changes the working directory to dir for the test
// and restores it on cleanup. Used by tests that need
// `commandsSourceDir` (a relative path) to resolve against a known
// root.
func chdir(t *testing.T, dir string) string {
	t.Helper()
	prev, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir %s: %v", dir, err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(prev); err != nil {
			t.Logf("restore cwd: %v", err)
		}
	})
	return dir
}
