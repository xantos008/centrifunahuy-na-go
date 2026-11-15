package harness

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// Mode is chosen by COMPAT_MODE:
//
//	diff   (default) run on ref and sut, compare normalized transcripts
//	golden run on sut only, compare with testdata/golden/<name>.json
//	record run on ref only, (re)write testdata/golden/<name>.json
func Mode() string { return envOr("COMPAT_MODE", "diff") }

// GoldenDir is relative to the scenarios package directory.
var GoldenDir = filepath.Join("..", "testdata", "golden")

// Scenario drives one target through a fixed sequence and records it in s.
type Scenario func(s *Session)

// Run executes scenario per the current mode and fails t on any difference.
func Run(t *testing.T, name string, sc Scenario) {
	t.Helper()
	switch Mode() {
	case "diff":
		ref, ok := runOn(t, Ref(), sc)
		if !ok {
			return
		}
		sut, ok := runOn(t, SUT(), sc)
		if !ok {
			return
		}
		if d := Diff("ref", ref, "sut", sut); d != "" {
			t.Errorf("%s: transcripts differ (ref vs sut):\n%s", name, d)
		}
	case "golden":
		want, err := loadGolden(name)
		if err != nil {
			t.Fatalf("%s: %v (record it with COMPAT_MODE=record)", name, err)
		}
		got, ok := runOn(t, SUT(), sc)
		if !ok {
			return
		}
		if d := Diff("golden", want, "sut", got); d != "" {
			t.Errorf("%s: sut differs from golden:\n%s", name, d)
		}
	case "record":
		got, ok := runOn(t, Ref(), sc)
		if !ok {
			return
		}
		if err := saveGolden(name, got); err != nil {
			t.Fatal(err)
		}
	default:
		t.Fatalf("unknown COMPAT_MODE %q", Mode())
	}
}

// runOn runs sc in a subtest named after the target, so a Fatal in one target
// does not hide the other's result.
func runOn(t *testing.T, tg Target, sc Scenario) (Transcript, bool) {
	t.Helper()
	var tr Transcript
	ok := t.Run(tg.Name, func(t *testing.T) {
		s := &Session{T: t, Target: tg}
		sc(s)
		n, err := Normalize(s.Transcript())
		if err != nil {
			t.Fatal(err)
		}
		tr = n
	})
	return tr, ok
}

func goldenPath(name string) string { return filepath.Join(GoldenDir, name+".json") }

func loadGolden(name string) (Transcript, error) {
	b, err := os.ReadFile(goldenPath(name))
	if err != nil {
		return Transcript{}, err
	}
	var tr Transcript
	if err := json.Unmarshal(b, &tr); err != nil {
		return Transcript{}, fmt.Errorf("%s: %w", goldenPath(name), err)
	}
	return tr, nil
}

func saveGolden(name string, tr Transcript) error {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false) // must match Lines(), which compares unescaped text
	enc.SetIndent("", "  ")
	if err := enc.Encode(tr); err != nil {
		return err
	}
	return os.WriteFile(goldenPath(name), buf.Bytes(), 0o644) //nolint:gosec // a repository file
}
