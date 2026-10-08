package dst

import (
	"sort"
	"time"

	"github.com/umarsabirin/cronx/internal/cronspec"
)

type Kind int

const (
	KindNormal Kind = iota
	KindSkipped
	KindDuplicated
)

type Transition struct {
	At                   time.Time
	FromOffset, ToOffset int
	FromAbbrev, ToAbbrev string
}

type Resolution struct {
	Wall       cronspec.WallClock
	Kind       Kind
	Instants   []time.Time
	Transition Transition
}

const window = 36 * time.Hour

func Resolve(w cronspec.WallClock, loc *time.Location) Resolution {
	probe := time.Date(w.Year, w.Month, w.Day, w.Hour, w.Minute, w.Second, 0, time.UTC)

	seen := map[int]bool{}
	var instants []time.Time
	for _, delta := range []time.Duration{-window, 0, window} {
		_, offset := probe.Add(delta).In(loc).Zone()
		if seen[offset] {
			continue
		}
		seen[offset] = true
		c := probe.Add(-time.Duration(offset) * time.Second)
		if renders(c, loc, probe) {
			instants = append(instants, c)
		}
	}
	sort.Slice(instants, func(i, j int) bool { return instants[i].Before(instants[j]) })

	res := Resolution{Wall: w, Instants: instants}
	switch len(instants) {
	case 0:
		res.Kind = KindSkipped
	case 1:
		res.Kind = KindNormal
	default:
		res.Kind = KindDuplicated
	}
	if res.Kind != KindNormal {
		res.Transition = findTransition(probe, loc)
	}
	return res
}

func findTransition(probe time.Time, loc *time.Location) Transition {
	lo, hi := probe.Add(-window), probe.Add(window)
	fromAbbrev, fromOffset := lo.In(loc).Zone()
	toAbbrev, toOffset := hi.In(loc).Zone()
	if fromOffset == toOffset {
		return Transition{}
	}
	loSec, hiSec := lo.Unix(), hi.Unix()
	for hiSec-loSec > 1 {
		mid := loSec + (hiSec-loSec)/2
		if _, offset := time.Unix(mid, 0).In(loc).Zone(); offset == fromOffset {
			loSec = mid
		} else {
			hiSec = mid
		}
	}
	return Transition{
		At:         time.Unix(hiSec, 0).UTC(),
		FromOffset: fromOffset,
		ToOffset:   toOffset,
		FromAbbrev: fromAbbrev,
		ToAbbrev:   toAbbrev,
	}
}

func renders(c time.Time, loc *time.Location, probe time.Time) bool {
	l := c.In(loc)
	return l.Year() == probe.Year() && l.Month() == probe.Month() && l.Day() == probe.Day() &&
		l.Hour() == probe.Hour() && l.Minute() == probe.Minute() && l.Second() == probe.Second()
}
