package cronspec

import (
	"testing"
	"time"
)

func TestParseRejectsCRONTZPrefix(t *testing.T) {
	if _, err := Parse("CRON_TZ=America/New_York 0 9 * * *"); err == nil {
		t.Fatal("Parse accepted a CRON_TZ= prefix, want error")
	}
}

func TestNextWallClocks(t *testing.T) {
	after := WallClock{Year: 2026, Month: time.March, Day: 7, Hour: 12, Minute: 0}

	got, err := mustParse(t, "30 1 * * *").NextWallClocks(after, 3)
	if err != nil {
		t.Fatalf("NextWallClocks: %v", err)
	}
	want := []WallClock{
		{Year: 2026, Month: time.March, Day: 8, Hour: 1, Minute: 30},
		{Year: 2026, Month: time.March, Day: 9, Hour: 1, Minute: 30},
		{Year: 2026, Month: time.March, Day: 10, Hour: 1, Minute: 30},
	}
	if len(got) != len(want) {
		t.Fatalf("got %d wall clocks, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("wall clock %d = %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestParseRejectsInvalidExpressions(t *testing.T) {
	for _, expr := range []string{"", "not a cron expression", "0 9 * *", "60 * * * *", "* * * * 8", "@every"} {
		if _, err := Parse(expr); err == nil {
			t.Errorf("Parse(%q) = nil error, want error", expr)
		}
	}
}

func TestNextWallClocksFieldMatching(t *testing.T) {
	after := WallClock{Year: 2026, Month: time.March, Day: 7, Hour: 12, Minute: 0}

	tests := []struct {
		expr string
		want []WallClock
	}{
		{"*/15 13 * * *", []WallClock{
			{Year: 2026, Month: time.March, Day: 7, Hour: 13, Minute: 0},
			{Year: 2026, Month: time.March, Day: 7, Hour: 13, Minute: 15},
			{Year: 2026, Month: time.March, Day: 7, Hour: 13, Minute: 30},
		}},
		{"0 9-11 * * *", []WallClock{
			{Year: 2026, Month: time.March, Day: 8, Hour: 9},
			{Year: 2026, Month: time.March, Day: 8, Hour: 10},
			{Year: 2026, Month: time.March, Day: 8, Hour: 11},
		}},
		{"0 0 * * MON", []WallClock{
			{Year: 2026, Month: time.March, Day: 9},
			{Year: 2026, Month: time.March, Day: 16},
		}},
		{"0 0 1 JAN,JUL *", []WallClock{
			{Year: 2026, Month: time.July, Day: 1},
			{Year: 2027, Month: time.January, Day: 1},
		}},
		{"0 0 13 * FRI", []WallClock{
			{Year: 2026, Month: time.March, Day: 13},
			{Year: 2026, Month: time.March, Day: 20},
		}},
	}

	for _, tc := range tests {
		t.Run(tc.expr, func(t *testing.T) {
			got, err := mustParse(t, tc.expr).NextWallClocks(after, len(tc.want))
			if err != nil {
				t.Fatalf("NextWallClocks: %v", err)
			}
			for i := range tc.want {
				if got[i] != tc.want[i] {
					t.Errorf("wall clock %d = %+v, want %+v", i, got[i], tc.want[i])
				}
			}
		})
	}
}

func TestNextWallClocksDescriptors(t *testing.T) {
	after := WallClock{Year: 2026, Month: time.March, Day: 7, Hour: 12, Minute: 0}

	tests := []struct {
		expr string
		want []WallClock
	}{
		{"@daily", []WallClock{
			{Year: 2026, Month: time.March, Day: 8},
			{Year: 2026, Month: time.March, Day: 9},
		}},
		{"@hourly", []WallClock{
			{Year: 2026, Month: time.March, Day: 7, Hour: 13},
			{Year: 2026, Month: time.March, Day: 7, Hour: 14},
		}},
		{"@weekly", []WallClock{
			{Year: 2026, Month: time.March, Day: 8},
			{Year: 2026, Month: time.March, Day: 15},
		}},
		{"@monthly", []WallClock{
			{Year: 2026, Month: time.April, Day: 1},
			{Year: 2026, Month: time.May, Day: 1},
		}},
		{"@yearly", []WallClock{
			{Year: 2027, Month: time.January, Day: 1},
			{Year: 2028, Month: time.January, Day: 1},
		}},
		{"@every 90m", []WallClock{
			{Year: 2026, Month: time.March, Day: 7, Hour: 13, Minute: 30},
			{Year: 2026, Month: time.March, Day: 7, Hour: 15, Minute: 0},
		}},
	}

	for _, tc := range tests {
		t.Run(tc.expr, func(t *testing.T) {
			got, err := mustParse(t, tc.expr).NextWallClocks(after, len(tc.want))
			if err != nil {
				t.Fatalf("NextWallClocks: %v", err)
			}
			for i := range tc.want {
				if got[i] != tc.want[i] {
					t.Errorf("wall clock %d = %+v, want %+v", i, got[i], tc.want[i])
				}
			}
		})
	}
}

func TestNextWallClocksIgnoresDST(t *testing.T) {
	tests := []struct {
		name  string
		expr  string
		after WallClock
		want  []WallClock
	}{
		{
			name:  "wall clock inside a spring-forward gap is still generated",
			expr:  "30 2 * * *",
			after: WallClock{Year: 2026, Month: time.March, Day: 7, Hour: 12},
			want: []WallClock{
				{Year: 2026, Month: time.March, Day: 8, Hour: 2, Minute: 30},
				{Year: 2026, Month: time.March, Day: 9, Hour: 2, Minute: 30},
			},
		},
		{
			name:  "wall clock inside a fall-back overlap is generated once",
			expr:  "30 1 * * *",
			after: WallClock{Year: 2026, Month: time.October, Day: 31, Hour: 12},
			want: []WallClock{
				{Year: 2026, Month: time.November, Day: 1, Hour: 1, Minute: 30},
				{Year: 2026, Month: time.November, Day: 2, Hour: 1, Minute: 30},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := mustParse(t, tc.expr).NextWallClocks(tc.after, len(tc.want))
			if err != nil {
				t.Fatalf("NextWallClocks: %v", err)
			}
			for i := range tc.want {
				if got[i] != tc.want[i] {
					t.Errorf("wall clock %d = %+v, want %+v", i, got[i], tc.want[i])
				}
			}
		})
	}
}

func TestNextWallClocksBeyondHorizon(t *testing.T) {
	after := WallClock{Year: 2026, Month: time.March, Day: 7, Hour: 12, Minute: 0}

	_, err := mustParse(t, "0 0 30 2 *").NextWallClocks(after, 1)
	if err == nil {
		t.Fatal("NextWallClocks accepted an expression with no occurrence, want error")
	}
}

func mustParse(t *testing.T, expr string) *Spec {
	t.Helper()
	s, err := Parse(expr)
	if err != nil {
		t.Fatalf("Parse(%q): %v", expr, err)
	}
	return s
}
