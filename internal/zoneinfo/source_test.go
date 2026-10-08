package zoneinfo

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSourcePrefersZONEINFO(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tzdata.zip")
	if err := os.WriteFile(path, []byte("not really a zip"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	t.Setenv("ZONEINFO", path)

	if got, want := Source(), "$ZONEINFO="+path; got != want {
		t.Errorf("Source() = %q, want %q", got, want)
	}
}

func TestSourceIgnoresNonExistentZONEINFO(t *testing.T) {
	t.Setenv("ZONEINFO", filepath.Join(t.TempDir(), "absent.zip"))

	if got := Source(); strings.HasPrefix(got, "$ZONEINFO=") {
		t.Errorf("Source() = %q, want a system path or the embedded form", got)
	}
}

func TestSourceFallsBackToEmbedded(t *testing.T) {
	t.Setenv("ZONEINFO", "")
	setSystemPaths(t, filepath.Join(t.TempDir(), "nowhere"))

	got := Source()
	if !strings.HasPrefix(got, "embedded (go") || !strings.HasSuffix(got, ")") {
		t.Errorf("Source() = %q, want the embedded (go1.x.y) form", got)
	}
}

func TestSourceReportsSystemPath(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ZONEINFO", "")
	setSystemPaths(t, filepath.Join(dir, "nowhere"), dir)

	if got := Source(); got != dir {
		t.Errorf("Source() = %q, want %q", got, dir)
	}
}

func setSystemPaths(t *testing.T, paths ...string) {
	t.Helper()
	saved := systemPaths
	t.Cleanup(func() { systemPaths = saved })
	systemPaths = paths
}

func TestSourceIsNeverEmpty(t *testing.T) {
	if Source() == "" {
		t.Error("Source() = \"\", want a non-empty description")
	}
}
