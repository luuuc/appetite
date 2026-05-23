package main

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// writeFakeBinary drops an executable shell script at <dir>/appetite
// that responds to `-h` like the real binary for the subset of
// commands the tests need. The script is only valid on POSIX shells;
// Windows is out of scope for this dogfood-era checker.
func writeFakeBinary(t *testing.T, dir, script string) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("POSIX-only fake binary; skipping on windows")
	}
	path := filepath.Join(dir, "appetite")
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+script), 0o755); err != nil {
		t.Fatalf("write fake binary: %v", err)
	}
	return path
}

func writeDocFile(t *testing.T, dir, body string) string {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", dir, err)
	}
	path := filepath.Join(dir, canonicalDocFile)
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write doc: %v", err)
	}
	return path
}

func TestRunPassesWhenDocDirAbsent(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "definitely-not-here")
	var stdout, stderr bytes.Buffer

	code := run([]string{"-doc", missing}, &stdout, &stderr)

	if code != exitOK {
		t.Errorf("code = %d, want %d", code, exitOK)
	}
	if !strings.Contains(stdout.String(), "not present, skipping") {
		t.Errorf("stdout = %q, want skip note", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Errorf("stderr = %q, want empty", stderr.String())
	}
}

func TestRunReportsCleanWhenDocAndBinaryAgree(t *testing.T) {
	docDir := t.TempDir()
	writeDocFile(t, docDir, "```bash\nappetite shape --new csv-export\n```\n")
	bin := writeFakeBinary(t, t.TempDir(), `case "$1" in
  shape) echo "Usage of shape:"; echo "  -new string"; echo "    new pitch slug"; exit 1 ;;
  *) echo "appetite: unknown command \"$1\""; exit 1 ;;
esac
`)

	var stdout, stderr bytes.Buffer
	code := run([]string{"-doc", docDir, "-binary", bin}, &stdout, &stderr)

	if code != exitOK {
		t.Errorf("code = %d (stderr=%q), want %d", code, stderr.String(), exitOK)
	}
	if !strings.Contains(stdout.String(), "contracts match") {
		t.Errorf("stdout = %q, want contracts-match note", stdout.String())
	}
}

func TestRunReportsDriftWhenDocFlagMissingInBinary(t *testing.T) {
	docDir := t.TempDir()
	writeDocFile(t, docDir, "```bash\nappetite mcp --dir .appetite\n```\n")
	bin := writeFakeBinary(t, t.TempDir(), `case "$1" in
  mcp) echo "Usage of mcp:"; exit 1 ;;
  *) echo "appetite: unknown command \"$1\""; exit 1 ;;
esac
`)

	var stdout, stderr bytes.Buffer
	code := run([]string{"-doc", docDir, "-binary", bin}, &stdout, &stderr)

	if code != exitDrift {
		t.Errorf("code = %d, want %d", code, exitDrift)
	}
	if !strings.Contains(stderr.String(), "does not advertise `--dir`") {
		t.Errorf("stderr = %q, want flag-missing note", stderr.String())
	}
}

func TestRunReportsDriftWhenDocCommandMissingInBinary(t *testing.T) {
	docDir := t.TempDir()
	writeDocFile(t, docDir, "```bash\nappetite ghost --whisper boo\n```\n")
	bin := writeFakeBinary(t, t.TempDir(), `case "$1" in
  *) echo "appetite: unknown command \"$1\""; exit 1 ;;
esac
`)

	var stdout, stderr bytes.Buffer
	code := run([]string{"-doc", docDir, "-binary", bin}, &stdout, &stderr)

	if code != exitDrift {
		t.Errorf("code = %d, want %d", code, exitDrift)
	}
	if !strings.Contains(stderr.String(), "does not recognize this command") {
		t.Errorf("stderr = %q, want unrecognized-command note", stderr.String())
	}
}

func TestRunReportsRequiredFlagNotExercisedByDoc(t *testing.T) {
	// The doc mentions `appetite bet <slug>` but no example exercises
	// the `--appetite` flag, which the binary marks `(required)`.
	docDir := t.TempDir()
	writeDocFile(t, docDir, "```bash\nappetite bet csv-export\n```\n")
	bin := writeFakeBinary(t, t.TempDir(), `case "$1" in
  bet) echo "Usage of bet:"; echo "  -appetite string"; echo "    appetite (required)"; exit 1 ;;
  *) echo "appetite: unknown command \"$1\""; exit 1 ;;
esac
`)

	var stdout, stderr bytes.Buffer
	code := run([]string{"-doc", docDir, "-binary", bin}, &stdout, &stderr)

	if code != exitDrift {
		t.Errorf("code = %d, want %d", code, exitDrift)
	}
	if !strings.Contains(stderr.String(), "is not exercised") {
		t.Errorf("stderr = %q, want required-flag note", stderr.String())
	}
}

func TestRunHonoursEqualsInFlagSyntax(t *testing.T) {
	// `--source=beacon` should be treated identically to `--source beacon`.
	docDir := t.TempDir()
	writeDocFile(t, docDir, "```bash\nappetite signal --source=beacon\n```\n")
	bin := writeFakeBinary(t, t.TempDir(), `case "$1" in
  signal) echo "Usage of signal:"; echo "  -source string"; echo "    where the signal came from"; exit 1 ;;
  *) echo "appetite: unknown command \"$1\""; exit 1 ;;
esac
`)

	var stdout, stderr bytes.Buffer
	code := run([]string{"-doc", docDir, "-binary", bin}, &stdout, &stderr)

	if code != exitOK {
		t.Errorf("code = %d (stderr=%q), want %d", code, stderr.String(), exitOK)
	}
}

func TestRunReturnsInternalErrorOnUnknownFlag(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"-no-such-flag"}, &stdout, &stderr)
	if code != exitInternal {
		t.Errorf("code = %d, want %d", code, exitInternal)
	}
}

func TestRunReturnsInternalErrorOnPositionalArg(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"unexpected"}, &stdout, &stderr)
	if code != exitInternal {
		t.Errorf("code = %d, want %d", code, exitInternal)
	}
}

func TestRunReturnsInternalErrorWhenDocPathIsAFile(t *testing.T) {
	file := filepath.Join(t.TempDir(), "not-a-dir")
	if err := os.WriteFile(file, []byte("blocker"), 0o644); err != nil {
		t.Fatalf("seed file: %v", err)
	}
	var stdout, stderr bytes.Buffer
	code := run([]string{"-doc", file}, &stdout, &stderr)
	if code != exitInternal {
		t.Errorf("code = %d, want %d", code, exitInternal)
	}
	if !strings.Contains(stderr.String(), "not a directory") {
		t.Errorf("stderr = %q", stderr.String())
	}
}

func TestRunReturnsInternalErrorOnStatFailure(t *testing.T) {
	parent := filepath.Join(t.TempDir(), "file-not-dir")
	if err := os.WriteFile(parent, []byte("blocker"), 0o644); err != nil {
		t.Fatalf("seed parent: %v", err)
	}
	var stdout, stderr bytes.Buffer
	code := run([]string{"-doc", filepath.Join(parent, "child")}, &stdout, &stderr)
	if code != exitInternal {
		t.Errorf("code = %d, want %d", code, exitInternal)
	}
}

func TestRunReturnsInternalErrorWhenCanonicalDocMissing(t *testing.T) {
	// `.doc/definition/` exists but `07-mcp-and-cli.md` is absent —
	// that is operator error, not the gitignored-skip case.
	docDir := t.TempDir()
	var stdout, stderr bytes.Buffer
	code := run([]string{"-doc", docDir}, &stdout, &stderr)
	if code != exitInternal {
		t.Errorf("code = %d, want %d", code, exitInternal)
	}
	if !strings.Contains(stderr.String(), "open") {
		t.Errorf("stderr = %q, want open-error message", stderr.String())
	}
}

func TestRunSurfacesParseError(t *testing.T) {
	// An unterminated quoted string in a fenced bash block makes the
	// tokenizer return an error; run() should turn that into exit 2.
	docDir := t.TempDir()
	writeDocFile(t, docDir, "```bash\nappetite signal add \"unterminated\n```\n")
	bin := writeFakeBinary(t, t.TempDir(), `exit 0`)

	var stdout, stderr bytes.Buffer
	code := run([]string{"-doc", docDir, "-binary", bin}, &stdout, &stderr)
	if code != exitInternal {
		t.Errorf("code = %d, want %d", code, exitInternal)
	}
}
