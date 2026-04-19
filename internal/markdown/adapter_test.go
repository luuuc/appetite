package markdown_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/luuuc/appetite/internal/markdown"
	"github.com/luuuc/appetite/internal/model"
	"github.com/luuuc/appetite/internal/store"
	"github.com/luuuc/appetite/internal/storetest"
)

func TestConformance(t *testing.T) {
	storetest.TestStore(t, func(t *testing.T) store.Store {
		t.Helper()
		return markdown.New(t.TempDir())
	})
}

func TestDirectoryAutoCreation(t *testing.T) {
	root := t.TempDir()
	a := markdown.New(root)
	ctx := context.Background()

	// Write a card into a cycle subdirectory that doesn't exist yet —
	// adapter should create cycles/<id>/cards/.
	card := model.Card{
		Slug:     "export-button",
		Pitch:    "csv-export",
		Cycle:    "2026-w15",
		Hill:     model.HillUphill,
		Progress: 0,
		Body:     "body\n",
	}
	path, err := a.Write(ctx, card)
	if err != nil {
		t.Fatalf("Write: %v", err)
	}

	abs := filepath.Join(root, path)
	if _, err := os.Stat(abs); err != nil {
		t.Fatalf("Stat written file: %v", err)
	}
	parent := filepath.Dir(abs)
	info, err := os.Stat(parent)
	if err != nil || !info.IsDir() {
		t.Fatalf("parent %s not a directory: err=%v info=%v", parent, err, info)
	}
}

func TestAtomicWriteLeavesNoTempFile(t *testing.T) {
	root := t.TempDir()
	a := markdown.New(root)
	ctx := context.Background()

	p := model.Pitch{
		Slug:     "csv-export",
		Title:    "CSV Export",
		Appetite: model.AppetiteSmall,
		Status:   model.PitchStatusShaped,
		Body:     "## Problem\n\nCustomers want exports.\n",
	}
	if _, err := a.Write(ctx, p); err != nil {
		t.Fatalf("Write: %v", err)
	}

	entries, err := os.ReadDir(filepath.Join(root, "pitches"))
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".tmp") {
			t.Errorf("temp file left behind: %s", e.Name())
		}
	}
}

func TestOnDiskShapeMarkdown(t *testing.T) {
	root := t.TempDir()
	a := markdown.New(root)
	ctx := context.Background()

	shaped := time.Date(2026, 4, 17, 0, 0, 0, 0, time.UTC)
	p := model.Pitch{
		Slug:     "csv-export",
		Title:    "CSV Export",
		Appetite: model.AppetiteMedium,
		Status:   model.PitchStatusShaped,
		ShapedAt: &shaped,
		Body:     "## Problem\n\nneed exports\n",
	}
	path, err := a.Write(ctx, p)
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(root, path))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	s := string(got)
	if !strings.HasPrefix(s, "---\n") {
		t.Errorf("file does not start with frontmatter open: %q", s[:min(20, len(s))])
	}
	if !strings.Contains(s, "\n---\n## Problem") {
		t.Errorf("file missing frontmatter close + body separator; got:\n%s", s)
	}
	if !strings.Contains(s, "slug: csv-export") {
		t.Errorf("file missing slug in frontmatter; got:\n%s", s)
	}
}

func TestOnDiskShapeYAMLHasNoFrontmatter(t *testing.T) {
	// Cycles and cooldowns are pure YAML — they must not have "---\n"
	// frontmatter delimiters.
	root := t.TempDir()
	a := markdown.New(root)
	ctx := context.Background()

	c := model.Cycle{
		ID:       "2026-w15",
		Duration: "5d",
		Started:  time.Date(2026, 4, 13, 0, 0, 0, 0, time.UTC),
		Ends:     time.Date(2026, 4, 17, 0, 0, 0, 0, time.UTC),
		Status:   model.CycleStatusBuilding,
	}
	path, err := a.Write(ctx, c)
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(root, path))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	s := string(got)
	if strings.HasPrefix(s, "---\n") {
		t.Errorf("cycle.yml should be pure YAML, not frontmatter-wrapped; got:\n%s", s)
	}
	if !strings.Contains(s, "id: 2026-w15") {
		t.Errorf("cycle.yml missing id; got:\n%s", s)
	}
}

func TestFrontmatterEdgeCases(t *testing.T) {
	root := t.TempDir()
	a := markdown.New(root)
	ctx := context.Background()

	t.Run("empty_body", func(t *testing.T) {
		p := model.Pitch{
			Slug:     "no-body",
			Title:    "No Body",
			Appetite: model.AppetiteSmall,
			Status:   model.PitchStatusShaping,
			Body:     "",
		}
		path, err := a.Write(ctx, p)
		if err != nil {
			t.Fatalf("Write: %v", err)
		}
		got, err := store.ReadAs[model.Pitch](ctx, a, path)
		if err != nil {
			t.Fatalf("Read: %v", err)
		}
		if got.Body != "" {
			t.Errorf("Body = %q, want empty", got.Body)
		}
	})

	t.Run("body_with_preserved_content", func(t *testing.T) {
		// Typical Shape Up pitch body with headings and lists —
		// nothing in here triggers the \n---\n edge case.
		body := "## Problem\n\nCustomers want exports.\n\n## Solution\n\n- Button on list views\n- CSV format\n"
		p := model.Pitch{
			Slug:     "preserved",
			Title:    "Preserved",
			Appetite: model.AppetiteMedium,
			Status:   model.PitchStatusShaped,
			Body:     body,
		}
		path, err := a.Write(ctx, p)
		if err != nil {
			t.Fatalf("Write: %v", err)
		}
		got, err := store.ReadAs[model.Pitch](ctx, a, path)
		if err != nil {
			t.Fatalf("Read: %v", err)
		}
		if got.Body != body {
			t.Errorf("Body not preserved.\n got: %q\nwant: %q", got.Body, body)
		}
	})

	t.Run("body_with_yaml_like_content", func(t *testing.T) {
		// A pitch body can legitimately include YAML-like syntax
		// (code fences documenting config, bullet lists, nested
		// keys). As long as no "\n---\n" line appears, content must
		// round-trip verbatim — the split boundary is the frontmatter
		// delimiter, not a YAML-aware scan.
		body := "## Config\n\n```yaml\napi:\n  timeout: 30s\n  retries: 3\n```\n\n- key: value\n- another: item\n"
		p := model.Pitch{
			Slug:     "yaml-body",
			Title:    "YAML Body",
			Appetite: model.AppetiteSmall,
			Status:   model.PitchStatusShaped,
			Body:     body,
		}
		path, err := a.Write(ctx, p)
		if err != nil {
			t.Fatalf("Write: %v", err)
		}
		got, err := store.ReadAs[model.Pitch](ctx, a, path)
		if err != nil {
			t.Fatalf("Read: %v", err)
		}
		if got.Body != body {
			t.Errorf("Body with YAML-like content not preserved.\n got: %q\nwant: %q", got.Body, body)
		}
	})

	t.Run("read_missing_opening_delimiter", func(t *testing.T) {
		// Place a file on disk that doesn't start with "---" and
		// confirm Read surfaces the parse error rather than panicking
		// or returning zero values.
		bad := filepath.Join(root, "pitches", "malformed.md")
		if err := os.MkdirAll(filepath.Dir(bad), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(bad, []byte("# just a heading\nno frontmatter here\n"), 0o644); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}
		_, err := a.Read(ctx, "pitches/malformed.md")
		if err == nil {
			t.Error("expected unmarshal error for missing frontmatter, got nil")
		}
	})

	t.Run("read_missing_closing_delimiter", func(t *testing.T) {
		bad := filepath.Join(root, "pitches", "unclosed.md")
		if err := os.MkdirAll(filepath.Dir(bad), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		content := "---\nslug: unclosed\ntitle: Unclosed\n(no closing delimiter)\n"
		if err := os.WriteFile(bad, []byte(content), 0o644); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}
		_, err := a.Read(ctx, "pitches/unclosed.md")
		if err == nil {
			t.Error("expected unmarshal error for missing closing delimiter, got nil")
		}
	})
}

func TestPathTraversalBlocked(t *testing.T) {
	root := t.TempDir()
	a := markdown.New(root)
	ctx := context.Background()

	// Any attempt to read a path that resolves outside root must error.
	bad := []string{
		"../escape.md",
		"../../etc/passwd",
		"pitches/../../etc/passwd",
	}
	for _, p := range bad {
		if _, err := a.Read(ctx, p); err == nil {
			t.Errorf("Read(%q) must error but returned nil", p)
		}
	}
}

