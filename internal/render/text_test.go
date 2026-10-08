package render

import (
	"bytes"
	"testing"
	"time"

	"github.com/umarsabirin/cronx/internal/cronspec"
	"github.com/umarsabirin/cronx/internal/dst"
	"github.com/umarsabirin/cronx/internal/plan"
)

func fallBackPlan(t *testing.T) plan.Plan {
	t.Helper()
	loc := mustLoad(t, "America/New_York")
	return plan.Plan{
		Zone: "America/New_York",
		Occurrences: []plan.Occurrence{
			{
				Local:    time.Date(2026, time.November, 1, 5, 30, 0, 0, time.UTC).In(loc),
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
				Local:  time.Date(2026, time.November, 2, 6, 30, 0, 0, time.UTC).In(loc),
				UTC:    time.Date(2026, time.November, 2, 6, 30, 0, 0, time.UTC),
				Abbrev: "EST",
			},
		},
		Warnings: []plan.Warning{
			{
				Wall: cronspec.WallClock{Year: 2026, Month: time.November, Day: 1, Hour: 1, Minute: 30},
				Kind: dst.KindDuplicated,
				Transition: dst.Transition{
					At:         time.Date(2026, time.November, 1, 6, 0, 0, 0, time.UTC),
					FromOffset: -4 * 3600, ToOffset: -5 * 3600,
					FromAbbrev: "EDT", ToAbbrev: "EST",
				},
			},
		},
	}
}

func TestTable(t *testing.T) {
	var buf bytes.Buffer
	if err := Table(&buf, fallBackPlan(t)); err != nil {
		t.Fatalf("Table: %v", err)
	}

	want := "#  LOCAL                           UTC                   NOTE\n" +
		"1  2026-11-01 01:30:00 -04:00 EDT  2026-11-01 05:30:00Z  dup 1/2\n" +
		"2  2026-11-01 01:30:00 -05:00 EST  2026-11-01 06:30:00Z  dup 2/2\n" +
		"3  2026-11-02 01:30:00 -05:00 EST  2026-11-02 06:30:00Z\n"

	if got := buf.String(); got != want {
		t.Errorf("Table output:\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func TestTableWritesNothingToWarnings(t *testing.T) {
	var table, warnings bytes.Buffer
	p := fallBackPlan(t)

	if err := Table(&table, p); err != nil {
		t.Fatalf("Table: %v", err)
	}
	if warnings.Len() != 0 {
		t.Errorf("Table wrote %q to the warning writer, want nothing", warnings.String())
	}

	if err := Warnings(&warnings, p); err != nil {
		t.Fatalf("Warnings: %v", err)
	}
	if warnings.Len() == 0 {
		t.Error("Warnings wrote nothing, want the warning block")
	}
}

func TestWarnings(t *testing.T) {
	p := plan.Plan{
		Zone: "America/New_York",
		Warnings: []plan.Warning{
			{
				Wall: cronspec.WallClock{Year: 2026, Month: time.March, Day: 8, Hour: 2, Minute: 30},
				Kind: dst.KindSkipped,
				Transition: dst.Transition{
					At:         time.Date(2026, time.March, 8, 7, 0, 0, 0, time.UTC),
					FromOffset: -5 * 3600, ToOffset: -4 * 3600,
					FromAbbrev: "EST", ToAbbrev: "EDT",
				},
			},
			fallBackPlan(t).Warnings[0],
		},
	}

	var buf bytes.Buffer
	if err := Warnings(&buf, p); err != nil {
		t.Fatalf("Warnings: %v", err)
	}

	want := "! 2026-03-08 02:30 does not exist in America/New_York\n" +
		"    clocks jump 02:00 -> 03:00 (EST -05:00 -> EDT -04:00) at 2026-03-08T07:00:00Z\n" +
		"    this occurrence will not fire\n" +
		"! 2026-11-01 01:30 occurs twice in America/New_York\n" +
		"    clocks fall back 02:00 -> 01:00 (EDT -04:00 -> EST -05:00) at 2026-11-01T06:00:00Z\n" +
		"    listed as 2 separate fire times\n"

	if got := buf.String(); got != want {
		t.Errorf("Warnings output:\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func TestWarningsEmptyPlanWritesNothing(t *testing.T) {
	var buf bytes.Buffer
	if err := Warnings(&buf, plan.Plan{Zone: "Asia/Jakarta"}); err != nil {
		t.Fatalf("Warnings: %v", err)
	}
	if buf.Len() != 0 {
		t.Errorf("got %q, want nothing", buf.String())
	}
}

func TestWarningsHalfHourOffset(t *testing.T) {
	p := plan.Plan{
		Zone: "Australia/Lord_Howe",
		Warnings: []plan.Warning{
			{
				Wall: cronspec.WallClock{Year: 2026, Month: time.October, Day: 4, Hour: 2, Minute: 15},
				Kind: dst.KindSkipped,
				Transition: dst.Transition{
					At:         time.Date(2026, time.October, 3, 15, 30, 0, 0, time.UTC),
					FromOffset: 10*3600 + 30*60, ToOffset: 11 * 3600,
					FromAbbrev: "+1030", ToAbbrev: "+11",
				},
			},
		},
	}

	var buf bytes.Buffer
	if err := Warnings(&buf, p); err != nil {
		t.Fatalf("Warnings: %v", err)
	}

	want := "! 2026-10-04 02:15 does not exist in Australia/Lord_Howe\n" +
		"    clocks jump 02:00 -> 02:30 (+1030 +10:30 -> +11 +11:00) at 2026-10-03T15:30:00Z\n" +
		"    this occurrence will not fire\n"

	if got := buf.String(); got != want {
		t.Errorf("Warnings output:\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func mustLoad(t *testing.T, name string) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation(name)
	if err != nil {
		t.Fatalf("LoadLocation(%q): %v", name, err)
	}
	return loc
}
