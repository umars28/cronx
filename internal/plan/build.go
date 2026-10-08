package plan

import (
	"time"

	"github.com/umarsabirin/cronx/internal/cronspec"
	"github.com/umarsabirin/cronx/internal/dst"
)

type Occurrence struct {
	Local, UTC         time.Time
	Abbrev             string
	DupIndex, DupTotal int
}

type Warning struct {
	Wall       cronspec.WallClock
	Kind       dst.Kind
	Transition dst.Transition
}

type Plan struct {
	Zone        string
	Occurrences []Occurrence
	Warnings    []Warning
}

func Build(s *cronspec.Spec, loc *time.Location, from time.Time, n int) (Plan, error) {
	local := from.In(loc)
	cursor := cronspec.WallClock{
		Year:   local.Year(),
		Month:  local.Month(),
		Day:    local.Day(),
		Hour:   local.Hour(),
		Minute: local.Minute(),
		Second: local.Second(),
	}

	p := Plan{Zone: loc.String()}
	for len(p.Occurrences) < n {
		walls, err := s.NextWallClocks(cursor, n-len(p.Occurrences))
		if err != nil {
			return Plan{}, err
		}
		for _, w := range walls {
			r := dst.Resolve(w, loc)
			if r.Kind != dst.KindNormal {
				p.Warnings = append(p.Warnings, Warning{Wall: w, Kind: r.Kind, Transition: r.Transition})
			}
			for i, instant := range r.Instants {
				o := Occurrence{Local: instant.In(loc), UTC: instant.UTC()}
				o.Abbrev, _ = o.Local.Zone()
				if r.Kind == dst.KindDuplicated {
					o.DupIndex, o.DupTotal = i+1, len(r.Instants)
				}
				p.Occurrences = append(p.Occurrences, o)
			}
		}
		cursor = walls[len(walls)-1]
	}
	if len(p.Occurrences) > n {
		p.Occurrences = p.Occurrences[:n]
	}
	return p, nil
}
