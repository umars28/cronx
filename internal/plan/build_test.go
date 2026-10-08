package plan

import (
	"testing"
	"time"

	"github.com/umarsabirin/cronx/internal/cronspec"
	"github.com/umarsabirin/cronx/internal/dst"
)

func TestBuildNormal(t *testing.T) {
	loc := mustLoad(t, "America/New_York")
	from := time.Date(2026, time.June, 15, 0, 0, 0, 0, time.UTC)

	p, err := Build(mustParse(t, "30 9 * * *"), loc, from, 2)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	if len(p.Warnings) != 0 {
		t.Errorf("got %d warnings, want 0", len(p.Warnings))
	}
	want := []Occurrence{
		{
			Local:  time.Date(2026, time.June, 15, 9, 30, 0, 0, loc),
			UTC:    time.Date(2026, time.June, 15, 13, 30, 0, 0, time.UTC),
			Abbrev: "EDT",
		},
		{
			Local:  time.Date(2026, time.June, 16, 9, 30, 0, 0, loc),
			UTC:    time.Date(2026, time.June, 16, 13, 30, 0, 0, time.UTC),
			Abbrev: "EDT",
		},
	}
	assertOccurrences(t, p.Occurrences, want)
}

func TestBuildDuplicateConsumesTwoSlots(t *testing.T) {
	loc := mustLoad(t, "America/New_York")
	from := time.Date(2026, time.October, 31, 16, 0, 0, 0, time.UTC)

	p, err := Build(mustParse(t, "30 1 * * *"), loc, from, 3)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	want := []Occurrence{
		{
			Local:    time.Date(2026, time.November, 1, 1, 30, 0, 0, loc),
			UTC:      time.Date(2026, time.November, 1, 5, 30, 0, 0, time.UTC),
			Abbrev:   "EDT",
			DupIndex: 1, DupTotal: 2,
		},
		{
			Local:    time.Date(2026, time.November, 1, 6, 30, 0, 0, time.UTC).In(loc),
			UTC:      time.Date(2026, time.November, 1, 6, 30, 0, 0, time.UTC),
			Abbrev:   "EST",
			DupIndex: 2, DupTotal: 2,
		},
		{
			Local:  time.Date(2026, time.November, 2, 1, 30, 0, 0, loc),
			UTC:    time.Date(2026, time.November, 2, 6, 30, 0, 0, time.UTC),
			Abbrev: "EST",
		},
	}
	assertOccurrences(t, p.Occurrences, want)

	if len(p.Warnings) != 1 {
		t.Fatalf("got %d warnings, want 1", len(p.Warnings))
	}
	w := p.Warnings[0]
	if w.Kind != dst.KindDuplicated {
		t.Errorf("Kind = %v, want KindDuplicated", w.Kind)
	}
	if want := (cronspec.WallClock{Year: 2026, Month: time.November, Day: 1, Hour: 1, Minute: 30}); w.Wall != want {
		t.Errorf("Wall = %+v, want %+v", w.Wall, want)
	}
	if at := time.Date(2026, time.November, 1, 6, 0, 0, 0, time.UTC); !w.Transition.At.Equal(at) {
		t.Errorf("Transition.At = %s, want %s", w.Transition.At.UTC(), at)
	}
}

func TestBuildSkipConsumesNoSlot(t *testing.T) {
	loc := mustLoad(t, "America/New_York")
	from := time.Date(2026, time.March, 7, 17, 0, 0, 0, time.UTC)

	p, err := Build(mustParse(t, "30 2 * * *"), loc, from, 3)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	want := []Occurrence{
		{
			Local:  time.Date(2026, time.March, 9, 2, 30, 0, 0, loc),
			UTC:    time.Date(2026, time.March, 9, 6, 30, 0, 0, time.UTC),
			Abbrev: "EDT",
		},
		{
			Local:  time.Date(2026, time.March, 10, 2, 30, 0, 0, loc),
			UTC:    time.Date(2026, time.March, 10, 6, 30, 0, 0, time.UTC),
			Abbrev: "EDT",
		},
		{
			Local:  time.Date(2026, time.March, 11, 2, 30, 0, 0, loc),
			UTC:    time.Date(2026, time.March, 11, 6, 30, 0, 0, time.UTC),
			Abbrev: "EDT",
		},
	}
	assertOccurrences(t, p.Occurrences, want)

	if len(p.Warnings) != 1 {
		t.Fatalf("got %d warnings, want 1", len(p.Warnings))
	}
	w := p.Warnings[0]
	if w.Kind != dst.KindSkipped {
		t.Errorf("Kind = %v, want KindSkipped", w.Kind)
	}
	if want := (cronspec.WallClock{Year: 2026, Month: time.March, Day: 8, Hour: 2, Minute: 30}); w.Wall != want {
		t.Errorf("Wall = %+v, want %+v", w.Wall, want)
	}
	if at := time.Date(2026, time.March, 8, 7, 0, 0, 0, time.UTC); !w.Transition.At.Equal(at) {
		t.Errorf("Transition.At = %s, want %s", w.Transition.At.UTC(), at)
	}
}

func TestBuildStartsFromInstantInTargetZone(t *testing.T) {
	loc := mustLoad(t, "America/New_York")
	from := time.Date(2026, time.June, 15, 16, 0, 0, 0, time.UTC)

	p, err := Build(mustParse(t, "0 14 * * *"), loc, from, 1)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	want := []Occurrence{
		{
			Local:  time.Date(2026, time.June, 15, 14, 0, 0, 0, loc),
			UTC:    time.Date(2026, time.June, 15, 18, 0, 0, 0, time.UTC),
			Abbrev: "EDT",
		},
	}
	assertOccurrences(t, p.Occurrences, want)
}

func TestBuildNeverExceedsN(t *testing.T) {
	loc := mustLoad(t, "America/New_York")
	from := time.Date(2026, time.October, 31, 16, 0, 0, 0, time.UTC)

	p, err := Build(mustParse(t, "30 1 * * *"), loc, from, 1)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	if len(p.Occurrences) != 1 {
		t.Fatalf("got %d occurrences, want 1", len(p.Occurrences))
	}
}

func TestBuildPropagatesHorizonError(t *testing.T) {
	loc := mustLoad(t, "America/New_York")
	from := time.Date(2026, time.June, 15, 0, 0, 0, 0, time.UTC)

	if _, err := Build(mustParse(t, "0 0 30 2 *"), loc, from, 1); err == nil {
		t.Fatal("Build: got nil error, want horizon error")
	}
}

func assertOccurrences(t *testing.T, got, want []Occurrence) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("got %d occurrences %+v, want %d", len(got), got, len(want))
	}
	for i := range want {
		if !got[i].Local.Equal(want[i].Local) {
			t.Errorf("occurrence %d Local = %s, want %s", i, got[i].Local, want[i].Local)
		}
		if !got[i].UTC.Equal(want[i].UTC) {
			t.Errorf("occurrence %d UTC = %s, want %s", i, got[i].UTC, want[i].UTC)
		}
		if got[i].Abbrev != want[i].Abbrev {
			t.Errorf("occurrence %d Abbrev = %s, want %s", i, got[i].Abbrev, want[i].Abbrev)
		}
		if got[i].DupIndex != want[i].DupIndex || got[i].DupTotal != want[i].DupTotal {
			t.Errorf("occurrence %d dup = %d/%d, want %d/%d",
				i, got[i].DupIndex, got[i].DupTotal, want[i].DupIndex, want[i].DupTotal)
		}
	}
}

func mustParse(t *testing.T, expr string) *cronspec.Spec {
	t.Helper()
	s, err := cronspec.Parse(expr)
	if err != nil {
		t.Fatalf("Parse(%q): %v", expr, err)
	}
	return s
}

func mustLoad(t *testing.T, name string) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation(name)
	if err != nil {
		t.Fatalf("LoadLocation(%q): %v", name, err)
	}
	return loc
}
