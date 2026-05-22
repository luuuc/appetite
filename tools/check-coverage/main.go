// Command check-coverage parses a Go coverage profile and fails if
// any function or file in the watched paths reports coverage below
// the floor. Per-file coverage is statement-weighted (parsed from
// coverage.out directly); per-function coverage uses `go tool cover
// -func` since the profile alone does not carry function names.
package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
)

// floor is the per-function and per-file coverage floor. The pitch
// pins it at 90.0%; this constant is the canonical source.
const floor = 90.0

// watched are the package paths whose functions/files the floor
// applies to. Other packages (model, store, markdown, version) are
// outside the gate by the pitch's wording.
var watched = []string{
	"github.com/luuuc/appetite/internal/workflow/",
	"github.com/luuuc/appetite/internal/cli/",
	"github.com/luuuc/appetite/cmd/appetite/",
}

type funcCov struct {
	file string
	name string
	pct  float64
}

type fileAgg struct {
	stmts   int
	covered int
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "check-coverage:", err)
		os.Exit(1)
	}
}

func run() error {
	profile := "coverage.out"
	if len(os.Args) > 1 {
		profile = os.Args[1]
	}
	if _, err := os.Stat(profile); err != nil {
		return fmt.Errorf("profile %s: %w", profile, err)
	}

	perFile, err := perFileCoverage(profile)
	if err != nil {
		return err
	}

	perFunc, err := perFunctionCoverage(profile)
	if err != nil {
		return err
	}

	var funcFailures, fileFailures []string
	for _, f := range perFunc {
		if f.pct < floor {
			funcFailures = append(funcFailures, fmt.Sprintf("  %s\t%s\t%.1f%% < %.1f%%", f.file, f.name, f.pct, floor))
		}
	}
	for _, file := range sortedKeys(perFile) {
		agg := perFile[file]
		if agg.stmts == 0 {
			continue
		}
		pct := 100.0 * float64(agg.covered) / float64(agg.stmts)
		if pct < floor {
			fileFailures = append(fileFailures, fmt.Sprintf("  %s\t%.1f%% (%d/%d stmts) < %.1f%%",
				file, pct, agg.covered, agg.stmts, floor))
		}
	}

	if len(funcFailures) == 0 && len(fileFailures) == 0 {
		fmt.Printf("check-coverage: all watched functions/files ≥ %.1f%%\n", floor)
		return nil
	}
	if len(funcFailures) > 0 {
		fmt.Fprintln(os.Stderr, "Functions below floor:")
		for _, l := range funcFailures {
			fmt.Fprintln(os.Stderr, l)
		}
	}
	if len(fileFailures) > 0 {
		fmt.Fprintln(os.Stderr, "Files below floor:")
		for _, l := range fileFailures {
			fmt.Fprintln(os.Stderr, l)
		}
	}
	return errors.New("coverage floor violated")
}

// perFileCoverage reads the raw Go coverage profile and returns a
// statement-weighted aggregate per file (only for watched paths).
// The profile format per non-header line is:
//
//	<file>:<startLine>.<startCol>,<endLine>.<endCol> <numStmt> <count>
//
// When running tests with -coverpkg over multiple packages, each
// block can appear multiple times — once per test binary that was
// instrumented for that package. We dedupe by location and treat
// the block as hit if ANY entry has count > 0.
func perFileCoverage(profile string) (map[string]*fileAgg, error) {
	f, err := os.Open(profile)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	// blocks: file → (loc → hit). loc is the "<start>,<end>" suffix.
	blocks := make(map[string]map[string]bool)
	stmtCounts := make(map[string]map[string]int)

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" || strings.HasPrefix(line, "mode:") {
			continue
		}
		spaceIdx := strings.LastIndex(line, " ")
		if spaceIdx <= 0 {
			continue
		}
		count, err := strconv.Atoi(line[spaceIdx+1:])
		if err != nil {
			continue
		}
		rest := line[:spaceIdx]
		spaceIdx2 := strings.LastIndex(rest, " ")
		if spaceIdx2 <= 0 {
			continue
		}
		stmts, err := strconv.Atoi(rest[spaceIdx2+1:])
		if err != nil {
			continue
		}
		fileLoc := rest[:spaceIdx2]
		colonIdx := strings.Index(fileLoc, ":")
		if colonIdx < 0 {
			continue
		}
		file := fileLoc[:colonIdx]
		loc := fileLoc[colonIdx+1:]
		if !watchedFile(file) {
			continue
		}
		if blocks[file] == nil {
			blocks[file] = make(map[string]bool)
			stmtCounts[file] = make(map[string]int)
		}
		stmtCounts[file][loc] = stmts
		if count > 0 {
			blocks[file][loc] = true
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	agg := make(map[string]*fileAgg, len(blocks))
	for file, locs := range stmtCounts {
		a := &fileAgg{}
		hits := blocks[file]
		for loc, stmts := range locs {
			a.stmts += stmts
			if hits[loc] {
				a.covered += stmts
			}
		}
		agg[file] = a
	}
	return agg, nil
}

// perFunctionCoverage runs `go tool cover -func` and parses the
// per-function table. Statement counts are not available here, so
// the per-function percentage is taken as the tool's own output —
// good enough since the gate is "no function below 90%", not an
// aggregate.
func perFunctionCoverage(profile string) ([]funcCov, error) {
	cmd := exec.Command("go", "tool", "cover", "-func="+profile)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("go tool cover: %w", err)
	}
	return parseFuncCov(strings.NewReader(string(out)))
}

// parseFuncCov reads `go tool cover -func` output. Each non-total
// line is `<file>:<line>:\t<name>\t<pct>%`. Lines we can't parse are
// quietly skipped — they're either the header or the total.
func parseFuncCov(r io.Reader) ([]funcCov, error) {
	scanner := bufio.NewScanner(r)
	var out []funcCov
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "total:") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		filePart := fields[0]
		name := fields[1]
		pctRaw := strings.TrimSuffix(fields[2], "%")
		pct, err := strconv.ParseFloat(pctRaw, 64)
		if err != nil {
			continue
		}
		if !watchedFile(filePart) {
			continue
		}
		out = append(out, funcCov{file: filePart, name: name, pct: pct})
	}
	return out, scanner.Err()
}

// watchedFile reports whether file is under one of the watched
// package paths.
func watchedFile(file string) bool {
	for _, w := range watched {
		if strings.HasPrefix(file, w) {
			return true
		}
	}
	return false
}

func sortedKeys(m map[string]*fileAgg) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
