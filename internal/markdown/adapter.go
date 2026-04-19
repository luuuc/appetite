// Package markdown implements store.Store against a .appetite/
// directory on disk. Pitches, signals, and cards are Markdown files
// with YAML frontmatter; cycles and cooldowns are pure YAML files.
// Writes are atomic (temp file + rename) so a reader never sees a
// partially written entity.
package markdown

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/luuuc/appetite/internal/model"
	"github.com/luuuc/appetite/internal/store"
)

// Adapter implements store.Store against an on-disk .appetite/ tree.
type Adapter struct {
	root string // absolute path to the .appetite/ directory
}

// New constructs a Markdown adapter rooted at root. The directory does
// not need to exist yet — the adapter creates subdirectories on first
// write.
func New(root string) *Adapter {
	return &Adapter{root: root}
}

// Write persists an entity atomically at e.Path(). On success the
// written relative path is returned.
func (a *Adapter) Write(_ context.Context, e model.Entity) (string, error) {
	rel := e.Path()
	abs, err := a.safePath(rel)
	if err != nil {
		return "", err
	}

	data, err := marshal(e)
	if err != nil {
		return "", fmt.Errorf("marshal %s: %w", rel, err)
	}

	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		return "", fmt.Errorf("mkdir %s: %w", filepath.Dir(rel), err)
	}

	tmp := abs + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return "", fmt.Errorf("write tmp %s: %w", rel, err)
	}
	if err := os.Rename(tmp, abs); err != nil {
		_ = os.Remove(tmp)
		return "", fmt.Errorf("rename %s: %w", rel, err)
	}

	return rel, nil
}

// Read loads a single entity by its relative path.
func (a *Adapter) Read(_ context.Context, rel string) (model.Entity, error) {
	abs, err := a.safePath(rel)
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(abs)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, store.ErrNotFound
		}
		return nil, fmt.Errorf("read %s: %w", rel, err)
	}

	e, err := unmarshal(rel, data)
	if err != nil {
		return nil, fmt.Errorf("unmarshal %s: %w", rel, err)
	}
	return e, nil
}

// List walks the .appetite/ tree, returning every entity that matches
// the filter. Push-down: Kind narrows to one top-level directory;
// Cycle narrows to one cycle subdirectory; Tags are checked per-entity
// after unmarshaling.
func (a *Adapter) List(_ context.Context, f store.Filter) ([]model.Entity, error) {
	result := make([]model.Entity, 0)

	kinds := kindsFromFilter(f)
	for _, kind := range kinds {
		paths, err := a.listPathsForKind(kind, f.Cycle)
		if err != nil {
			return result, err
		}
		for _, rel := range paths {
			abs, err := a.safePath(rel)
			if err != nil {
				return result, err
			}
			data, err := os.ReadFile(abs)
			if err != nil {
				return result, fmt.Errorf("read %s: %w", rel, err)
			}
			e, err := unmarshal(rel, data)
			if err != nil {
				return result, fmt.Errorf("unmarshal %s: %w", rel, err)
			}
			if matchesFilter(e, f) {
				result = append(result, e)
			}
		}
	}
	return result, nil
}

// Delete removes an entity at the given relative path.
func (a *Adapter) Delete(_ context.Context, rel string) error {
	abs, err := a.safePath(rel)
	if err != nil {
		return err
	}
	if err := os.Remove(abs); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return store.ErrNotFound
		}
		return fmt.Errorf("delete %s: %w", rel, err)
	}
	return nil
}

// safePath joins rel onto root and verifies it stays within root. This
// rejects traversal payloads like "../../etc/passwd". Symlinks inside
// .appetite/ are not blocked — they resolve during the subsequent
// Read/Write/Delete, after this boundary check.
func (a *Adapter) safePath(rel string) (string, error) {
	abs := filepath.Clean(filepath.Join(a.root, rel))
	rootClean := filepath.Clean(a.root)
	if abs != rootClean && !strings.HasPrefix(abs, rootClean+string(filepath.Separator)) {
		return "", fmt.Errorf("path %q escapes root", rel)
	}
	return abs, nil
}

// --- List traversal -------------------------------------------------------

// kindsFromFilter returns the kinds to walk. Kind narrows to one.
// Cycle implicitly excludes non-cycle-scoped kinds (signals, pitches)
// since their filesystem layout carries no cycle id — asking for "all
// entities in cycle X" is a cycle-scoped question.
func kindsFromFilter(f store.Filter) []model.Kind {
	var kinds []model.Kind
	if f.Kind != nil {
		kinds = []model.Kind{*f.Kind}
	} else {
		kinds = []model.Kind{
			model.KindSignal,
			model.KindPitch,
			model.KindCycle,
			model.KindCard,
			model.KindCooldown,
		}
	}
	if f.Cycle == nil {
		return kinds
	}
	scoped := kinds[:0]
	for _, k := range kinds {
		if isCycleScoped(k) {
			scoped = append(scoped, k)
		}
	}
	return scoped
}

// isCycleScoped reports whether an entity of this kind lives under
// cycles/<id>/ on disk.
func isCycleScoped(k model.Kind) bool {
	return k == model.KindCycle || k == model.KindCard || k == model.KindCooldown
}

// listPathsForKind returns every relative path holding an entity of
// the requested kind, honoring a Cycle filter for cycle-scoped kinds.
func (a *Adapter) listPathsForKind(kind model.Kind, cycle *string) ([]string, error) {
	switch kind {
	case model.KindSignal:
		paths := make([]string, 0)
		for _, sub := range []string{"signals/raw", "signals/archived"} {
			p, err := a.listMDIn(sub)
			if err != nil {
				return nil, err
			}
			paths = append(paths, p...)
		}
		return paths, nil

	case model.KindPitch:
		return a.listMDIn("pitches")

	case model.KindCycle:
		return a.listCycleScoped("cycle.yml", cycle, false)

	case model.KindCard:
		return a.listCycleCards(cycle)

	case model.KindCooldown:
		return a.listCycleScoped("cooldown.yml", cycle, false)
	}
	return nil, fmt.Errorf("unknown kind %q", kind)
}

// listMDIn returns every .md file directly inside dir.
func (a *Adapter) listMDIn(dir string) ([]string, error) {
	entries, err := os.ReadDir(filepath.Join(a.root, dir))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("readdir %s: %w", dir, err)
	}
	paths := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		paths = append(paths, dir+"/"+e.Name())
	}
	return paths, nil
}

// listCycleScoped returns paths for per-cycle YAML files (cycle.yml,
// cooldown.yml). When cycle is nil, every cycle subdirectory is
// scanned.
func (a *Adapter) listCycleScoped(filename string, cycle *string, _ bool) ([]string, error) {
	ids, err := a.cycleIDs(cycle)
	if err != nil {
		return nil, err
	}
	paths := make([]string, 0, len(ids))
	for _, id := range ids {
		rel := "cycles/" + id + "/" + filename
		abs := filepath.Join(a.root, rel)
		if _, err := os.Stat(abs); err == nil {
			paths = append(paths, rel)
		} else if !errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("stat %s: %w", rel, err)
		}
	}
	return paths, nil
}

// listCycleCards returns every cards/<slug>.md across the requested
// cycles (or all cycles when cycle is nil).
func (a *Adapter) listCycleCards(cycle *string) ([]string, error) {
	ids, err := a.cycleIDs(cycle)
	if err != nil {
		return nil, err
	}
	paths := make([]string, 0)
	for _, id := range ids {
		sub := "cycles/" + id + "/cards"
		p, err := a.listMDIn(sub)
		if err != nil {
			return nil, err
		}
		paths = append(paths, p...)
	}
	return paths, nil
}

// cycleIDs returns the cycle ids to scan. When cycle is non-nil, only
// that id is returned (even if it does not exist on disk — callers
// handle absent files via stat). Otherwise every subdirectory of
// cycles/ is returned.
func (a *Adapter) cycleIDs(cycle *string) ([]string, error) {
	if cycle != nil {
		return []string{*cycle}, nil
	}
	entries, err := os.ReadDir(filepath.Join(a.root, "cycles"))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("readdir cycles: %w", err)
	}
	ids := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			ids = append(ids, e.Name())
		}
	}
	return ids, nil
}

// matchesFilter applies post-read filters (currently Tags only; Kind
// and Cycle are pushed down during traversal).
func matchesFilter(e model.Entity, f store.Filter) bool {
	if len(f.Tags) == 0 {
		return true
	}
	sig, ok := e.(model.Signal)
	if !ok {
		// Tags are defined on Signal only; other kinds are excluded
		// when Tags is set so a kind+tags filter doesn't silently
		// match non-signal entities.
		return false
	}
	have := make(map[string]bool, len(sig.Tags))
	for _, t := range sig.Tags {
		have[t] = true
	}
	for _, required := range f.Tags {
		if !have[required] {
			return false
		}
	}
	return true
}

// --- Unmarshal (read path) ------------------------------------------------

// unmarshal turns file bytes into the typed entity matching the path's
// kind. Path-derived fields (slug from filename, cycle id from parent
// directory, archived from subdirectory) are filled in after parsing.
func unmarshal(rel string, data []byte) (model.Entity, error) {
	kind, err := kindFromPath(rel)
	if err != nil {
		return nil, err
	}

	switch kind {
	case model.KindSignal:
		var s model.Signal
		fm, body, err := splitFrontmatter(data)
		if err != nil {
			return nil, err
		}
		if err := yaml.Unmarshal(fm, &s); err != nil {
			return nil, err
		}
		s.Body = body
		s.Slug = slugFromFile(rel)
		s.Archived = strings.HasPrefix(rel, "signals/archived/")
		return s, nil

	case model.KindPitch:
		var p model.Pitch
		fm, body, err := splitFrontmatter(data)
		if err != nil {
			return nil, err
		}
		if err := yaml.Unmarshal(fm, &p); err != nil {
			return nil, err
		}
		p.Body = body
		if p.Slug == "" {
			p.Slug = slugFromFile(rel)
		}
		return p, nil

	case model.KindCard:
		var c model.Card
		fm, body, err := splitFrontmatter(data)
		if err != nil {
			return nil, err
		}
		if err := yaml.Unmarshal(fm, &c); err != nil {
			return nil, err
		}
		c.Body = body
		if c.Slug == "" {
			c.Slug = slugFromFile(rel)
		}
		if c.Cycle == "" {
			c.Cycle = cycleIDFromCardPath(rel)
		}
		return c, nil

	case model.KindCycle:
		var c model.Cycle
		if err := yaml.Unmarshal(data, &c); err != nil {
			return nil, err
		}
		if c.ID == "" {
			c.ID = cycleIDFromYMLPath(rel)
		}
		return c, nil

	case model.KindCooldown:
		var c model.Cooldown
		if err := yaml.Unmarshal(data, &c); err != nil {
			return nil, err
		}
		c.Cycle = cycleIDFromYMLPath(rel)
		return c, nil
	}
	return nil, fmt.Errorf("unknown kind %q for path %q", kind, rel)
}

// --- Marshal (write path) -------------------------------------------------

// marshal renders an entity as bytes suitable for its file extension.
// Markdown kinds (.md) become "---\n<yaml>---\n<body>"; YAML kinds
// (.yml) become raw YAML.
func marshal(e model.Entity) ([]byte, error) {
	switch v := e.(type) {
	case model.Signal:
		return marshalMD(v, v.Body)
	case model.Pitch:
		return marshalMD(v, v.Body)
	case model.Card:
		return marshalMD(v, v.Body)
	case model.Cycle:
		return yaml.Marshal(v)
	case model.Cooldown:
		return yaml.Marshal(v)
	}
	return nil, fmt.Errorf("cannot marshal %T", e)
}

// marshalMD renders a frontmatter-plus-body Markdown file.
func marshalMD(frontmatter any, body string) ([]byte, error) {
	y, err := yaml.Marshal(frontmatter)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	buf.WriteString("---\n")
	buf.Write(y)
	buf.WriteString("---\n")
	if body != "" {
		buf.WriteString(body)
	}
	return buf.Bytes(), nil
}

// splitFrontmatter separates a "---\n<yaml>---\n<body>" markdown file
// into its YAML frontmatter and body. The opening "---" must be at
// byte 0; the closing delimiter is the first "\n---\n" or "\n---"
// at EOF after it.
//
// Known limitation: if the body itself contains a literal "\n---\n"
// line (for example a horizontal rule in Markdown), the splitter
// interprets it as the frontmatter close and truncates the body. Most
// YAML-frontmatter tooling shares this limitation. Workarounds:
// indent the rule or wrap it in a code fence.
func splitFrontmatter(data []byte) (frontmatter []byte, body string, err error) {
	const open = "---\n"
	s := string(data)
	if !strings.HasPrefix(s, open) {
		return nil, "", fmt.Errorf("missing opening frontmatter delimiter")
	}
	rest := s[len(open):]
	// Match "\n---\n" (normal) or "\n---" at EOF (trailing newline-less file).
	idx := strings.Index(rest, "\n---\n")
	if idx < 0 {
		if strings.HasSuffix(rest, "\n---") {
			return []byte(rest[:len(rest)-len("\n---")]), "", nil
		}
		return nil, "", fmt.Errorf("missing closing frontmatter delimiter")
	}
	return []byte(rest[:idx]), rest[idx+len("\n---\n"):], nil
}

// --- Path helpers ---------------------------------------------------------

// kindFromPath maps a relative path to its entity kind.
func kindFromPath(rel string) (model.Kind, error) {
	p := filepath.ToSlash(filepath.Clean(rel))
	switch {
	case strings.HasPrefix(p, "signals/") && strings.HasSuffix(p, ".md"):
		return model.KindSignal, nil
	case strings.HasPrefix(p, "pitches/") && strings.HasSuffix(p, ".md"):
		return model.KindPitch, nil
	case strings.HasPrefix(p, "cycles/") && strings.HasSuffix(p, "/cycle.yml"):
		return model.KindCycle, nil
	case strings.HasPrefix(p, "cycles/") && strings.HasSuffix(p, "/cooldown.yml"):
		return model.KindCooldown, nil
	case strings.HasPrefix(p, "cycles/") && strings.Contains(p, "/cards/") && strings.HasSuffix(p, ".md"):
		return model.KindCard, nil
	}
	return "", fmt.Errorf("unrecognized entity path %q", rel)
}

// slugFromFile strips the directory and .md extension from a path.
func slugFromFile(rel string) string {
	base := filepath.Base(rel)
	return strings.TrimSuffix(base, ".md")
}

// cycleIDFromYMLPath extracts <id> from "cycles/<id>/cycle.yml" or
// "cycles/<id>/cooldown.yml".
func cycleIDFromYMLPath(rel string) string {
	p := filepath.ToSlash(filepath.Clean(rel))
	parts := strings.Split(p, "/")
	// cycles / <id> / cycle.yml  →  len 3
	if len(parts) >= 3 && parts[0] == "cycles" {
		return parts[1]
	}
	return ""
}

// cycleIDFromCardPath extracts <id> from "cycles/<id>/cards/<slug>.md".
func cycleIDFromCardPath(rel string) string {
	p := filepath.ToSlash(filepath.Clean(rel))
	parts := strings.Split(p, "/")
	// cycles / <id> / cards / <slug>.md  →  len 4
	if len(parts) >= 4 && parts[0] == "cycles" && parts[2] == "cards" {
		return parts[1]
	}
	return ""
}
