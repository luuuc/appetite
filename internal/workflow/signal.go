package workflow

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/luuuc/appetite/internal/model"
	"github.com/luuuc/appetite/internal/store"
)

// SignalAddRequest carries the fields needed to record a new raw
// signal. Body is required; everything else has a sensible default.
type SignalAddRequest struct {
	// Body is the freeform Markdown body of the signal. Required.
	Body string

	// Source identifies where the signal came from. Defaults to
	// "operator" (a human typed it).
	Source model.SignalSource

	// Tags are optional free-form labels.
	Tags []string

	// Slug overrides the auto-generated slug. Optional. When empty,
	// the slug is derived from the captured timestamp plus the first
	// few words of the body — collision-resistant enough for solo
	// use without imposing UUIDs.
	Slug string

	// Now overrides the captured timestamp. Tests inject a fixed
	// time; production callers leave this zero.
	Now time.Time
}

// SignalAddResult reports what was written.
type SignalAddResult struct {
	Signal model.Signal
	Path   string
}

// SignalAdd writes a new signal to signals/raw/.
func SignalAdd(s store.Store, req SignalAddRequest) (SignalAddResult, error) {
	if strings.TrimSpace(req.Body) == "" {
		return SignalAddResult{}, fmt.Errorf("signal body is required")
	}
	now := defaultNow(req.Now)
	source := req.Source
	if source == "" {
		source = model.SignalSourceOperator
	}
	if !source.Valid() {
		return SignalAddResult{}, fmt.Errorf("unknown signal source %q", source)
	}

	slug := req.Slug
	if slug == "" {
		slug = autoSignalSlug(now, req.Body)
	}

	sig := model.Signal{
		Slug:     slug,
		Source:   source,
		Captured: now,
		Tags:     req.Tags,
		Body:     ensureTrailingNewline(req.Body),
	}

	path, err := s.Write(context.Background(), sig)
	if err != nil {
		return SignalAddResult{}, err
	}
	return SignalAddResult{Signal: sig, Path: path}, nil
}

// autoSignalSlug builds a stable, human-readable slug from the
// capture time and the body's first few words. Format:
// "YYYY-MM-DD-<words>".
func autoSignalSlug(now time.Time, body string) string {
	date := now.UTC().Format("2006-01-02")
	tail := slugifyWords(body, 5)
	if tail == "" {
		return date
	}
	return date + "-" + tail
}

// slugifyWords lowercases, strips non-alphanumerics, and joins the
// first n word-tokens of s with dashes. Empty input → empty output.
func slugifyWords(s string, n int) string {
	var words []string
	var cur strings.Builder
	flush := func() {
		if cur.Len() > 0 {
			words = append(words, cur.String())
			cur.Reset()
		}
	}
	for _, r := range strings.ToLower(s) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			cur.WriteRune(r)
		default:
			flush()
			if len(words) >= n {
				return strings.Join(words, "-")
			}
		}
	}
	flush()
	if len(words) > n {
		words = words[:n]
	}
	return strings.Join(words, "-")
}

// ensureTrailingNewline guarantees body ends with "\n" so the
// rendered Markdown file has a final newline (POSIX habit + cleaner
// diffs).
func ensureTrailingNewline(body string) string {
	if body == "" || strings.HasSuffix(body, "\n") {
		return body
	}
	return body + "\n"
}

// defaultNow returns t if non-zero, else time.Now() truncated to the
// second in UTC. Truncation keeps frontmatter dates diff-clean.
func defaultNow(t time.Time) time.Time {
	if !t.IsZero() {
		return t.UTC().Truncate(time.Second)
	}
	return time.Now().UTC().Truncate(time.Second)
}
