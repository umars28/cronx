package dst

import (
	"testing"
	"time"

	"github.com/umarsabirin/cronx/internal/cronspec"
)

func TestResolveNormal(t *testing.T) {
	loc := mustLoad(t, "America/New_York")
	w := cronspec.WallClock{Year: 2026, Month: time.June, Day: 15, Hour: 9, Minute: 30}

	got := Resolve(w, loc)

	if got.Kind != KindNormal {
		t.Fatalf("Kind = %v, want KindNormal", got.Kind)
	}
	if len(got.Instants) != 1 {
		t.Fatalf("got %d instants, want 1", len(got.Instants))
	}
	want := time.Date(2026, time.June, 15, 13, 30, 0, 0, time.UTC)
	if !got.Instants[0].Equal(want) {
		t.Errorf("instant = %s, want %s", got.Instants[0].UTC(), want)
	}
}

func TestResolveSkipped(t *testing.T) {
	loc := mustLoad(t, "America/New_York")
	w := cronspec.WallClock{Year: 2026, Month: time.March, Day: 8, Hour: 2, Minute: 30}

	got := Resolve(w, loc)

	if got.Kind != KindSkipped {
		t.Fatalf("Kind = %v, want KindSkipped", got.Kind)
	}
	if len(got.Instants) != 0 {
		t.Fatalf("got %d instants, want 0", len(got.Instants))
	}
}

func TestResolveDuplicated(t *testing.T) {
	loc := mustLoad(t, "America/New_York")
	w := cronspec.WallClock{Year: 2026, Month: time.November, Day: 1, Hour: 1, Minute: 30}

	got := Resolve(w, loc)

	if got.Kind != KindDuplicated {
		t.Fatalf("Kind = %v, want KindDuplicated", got.Kind)
	}
	want := []time.Time{
		time.Date(2026, time.November, 1, 5, 30, 0, 0, time.UTC),
		time.Date(2026, time.November, 1, 6, 30, 0, 0, time.UTC),
	}
	if len(got.Instants) != len(want) {
		t.Fatalf("got %d instants, want %d", len(got.Instants), len(want))
	}
	for i := range want {
		if !got.Instants[i].Equal(want[i]) {
			t.Errorf("instant %d = %s, want %s", i, got.Instants[i].UTC(), want[i])
		}
	}

	abbrevs := []string{"EDT", "EST"}
	offsets := []int{-4 * 3600, -5 * 3600}
	for i := range abbrevs {
		abbrev, offset := got.Instants[i].In(loc).Zone()
		if abbrev != abbrevs[i] || offset != offsets[i] {
			t.Errorf("instant %d zone = %s %d, want %s %d", i, abbrev, offset, abbrevs[i], offsets[i])
		}
	}
}

func TestResolveAcrossZones(t *testing.T) {
	tests := []struct {
		name     string
		zone     string
		wall     cronspec.WallClock
		kind     Kind
		instants []time.Time
	}{
		{
			name: "Lord Howe 30-minute spring gap",
			zone: "Australia/Lord_Howe",
			wall: cronspec.WallClock{Year: 2026, Month: time.October, Day: 4, Hour: 2, Minute: 15},
			kind: KindSkipped,
		},
		{
			name: "Lord Howe 30-minute fall overlap",
			zone: "Australia/Lord_Howe",
			wall: cronspec.WallClock{Year: 2026, Month: time.April, Day: 5, Hour: 1, Minute: 45},
			kind: KindDuplicated,
			instants: []time.Time{
				time.Date(2026, time.April, 4, 14, 45, 0, 0, time.UTC),
				time.Date(2026, time.April, 4, 15, 15, 0, 0, time.UTC),
			},
		},
		{
			name: "Dublin negative DST spring gap",
			zone: "Europe/Dublin",
			wall: cronspec.WallClock{Year: 2026, Month: time.March, Day: 29, Hour: 1, Minute: 30},
			kind: KindSkipped,
		},
		{
			name: "Dublin negative DST fall overlap",
			zone: "Europe/Dublin",
			wall: cronspec.WallClock{Year: 2026, Month: time.October, Day: 25, Hour: 1, Minute: 30},
			kind: KindDuplicated,
			instants: []time.Time{
				time.Date(2026, time.October, 25, 0, 30, 0, 0, time.UTC),
				time.Date(2026, time.October, 25, 1, 30, 0, 0, time.UTC),
			},
		},
		{
			name: "Jakarta never shifts",
			zone: "Asia/Jakarta",
			wall: cronspec.WallClock{Year: 2026, Month: time.March, Day: 29, Hour: 1, Minute: 30},
			kind: KindNormal,
			instants: []time.Time{
				time.Date(2026, time.March, 28, 18, 30, 0, 0, time.UTC),
			},
		},
		{
			name: "Apia skipped the whole of 2011-12-30",
			zone: "Pacific/Apia",
			wall: cronspec.WallClock{Year: 2011, Month: time.December, Day: 30, Hour: 12, Minute: 0},
			kind: KindSkipped,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := Resolve(tc.wall, mustLoad(t, tc.zone))
			if got.Kind != tc.kind {
				t.Fatalf("Kind = %v, want %v", got.Kind, tc.kind)
			}
			if len(got.Instants) != len(tc.instants) {
				t.Fatalf("got %d instants %v, want %d", len(got.Instants), got.Instants, len(tc.instants))
			}
			for i := range tc.instants {
				if !got.Instants[i].Equal(tc.instants[i]) {
					t.Errorf("instant %d = %s, want %s", i, got.Instants[i].UTC(), tc.instants[i])
				}
			}
		})
	}
}

func TestResolveJakartaNeverSkipsOrDuplicates(t *testing.T) {
	loc := mustLoad(t, "Asia/Jakarta")
	day := cronspec.WallClock{Year: 2026, Month: time.March, Day: 29}

	for hour := 0; hour < 24; hour++ {
		w := day
		w.Hour = hour
		if got := Resolve(w, loc); got.Kind != KindNormal {
			t.Errorf("Resolve(%02d:00) Kind = %v, want KindNormal", hour, got.Kind)
		}
	}
}

func TestResolveTransition(t *testing.T) {
	tests := []struct {
		name string
		zone string
		wall cronspec.WallClock
		want Transition
	}{
		{
			name: "New York spring forward",
			zone: "America/New_York",
			wall: cronspec.WallClock{Year: 2026, Month: time.March, Day: 8, Hour: 2, Minute: 30},
			want: Transition{
				At:         time.Date(2026, time.March, 8, 7, 0, 0, 0, time.UTC),
				FromOffset: -5 * 3600, ToOffset: -4 * 3600,
				FromAbbrev: "EST", ToAbbrev: "EDT",
			},
		},
		{
			name: "New York fall back",
			zone: "America/New_York",
			wall: cronspec.WallClock{Year: 2026, Month: time.November, Day: 1, Hour: 1, Minute: 30},
			want: Transition{
				At:         time.Date(2026, time.November, 1, 6, 0, 0, 0, time.UTC),
				FromOffset: -4 * 3600, ToOffset: -5 * 3600,
				FromAbbrev: "EDT", ToAbbrev: "EST",
			},
		},
		{
			name: "Lord Howe 30-minute shift",
			zone: "Australia/Lord_Howe",
			wall: cronspec.WallClock{Year: 2026, Month: time.October, Day: 4, Hour: 2, Minute: 15},
			want: Transition{
				At:         time.Date(2026, time.October, 3, 15, 30, 0, 0, time.UTC),
				FromOffset: 10*3600 + 30*60, ToOffset: 11 * 3600,
				FromAbbrev: "+1030", ToAbbrev: "+11",
			},
		},
		{
			name: "Dublin negative DST",
			zone: "Europe/Dublin",
			wall: cronspec.WallClock{Year: 2026, Month: time.October, Day: 25, Hour: 1, Minute: 30},
			want: Transition{
				At:         time.Date(2026, time.October, 25, 1, 0, 0, 0, time.UTC),
				FromOffset: 3600, ToOffset: 0,
				FromAbbrev: "IST", ToAbbrev: "GMT",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := Resolve(tc.wall, mustLoad(t, tc.zone)).Transition
			if !got.At.Equal(tc.want.At) {
				t.Errorf("At = %s, want %s", got.At.UTC(), tc.want.At)
			}
			if got.FromOffset != tc.want.FromOffset || got.ToOffset != tc.want.ToOffset {
				t.Errorf("offsets = %d -> %d, want %d -> %d",
					got.FromOffset, got.ToOffset, tc.want.FromOffset, tc.want.ToOffset)
			}
			if got.FromAbbrev != tc.want.FromAbbrev || got.ToAbbrev != tc.want.ToAbbrev {
				t.Errorf("abbrevs = %s -> %s, want %s -> %s",
					got.FromAbbrev, got.ToAbbrev, tc.want.FromAbbrev, tc.want.ToAbbrev)
			}
		})
	}
}

func TestResolveNormalHasZeroTransition(t *testing.T) {
	loc := mustLoad(t, "America/New_York")
	w := cronspec.WallClock{Year: 2026, Month: time.June, Day: 15, Hour: 9, Minute: 30}

	if got := Resolve(w, loc).Transition; got != (Transition{}) {
		t.Errorf("Transition = %+v, want zero value", got)
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
