package cronspec

import (
	"fmt"
	"strings"
	"time"

	"github.com/robfig/cron/v3"
)

type WallClock struct {
	Year                      int
	Month                     time.Month
	Day, Hour, Minute, Second int
}

type Spec struct {
	schedule cron.Schedule
}

func (s *Spec) NextWallClocks(after WallClock, n int) ([]WallClock, error) {
	cursor := time.Date(after.Year, after.Month, after.Day, after.Hour, after.Minute, after.Second, 0, time.UTC)
	out := make([]WallClock, 0, n)
	for len(out) < n {
		next := s.schedule.Next(cursor)
		if next.IsZero() {
			return nil, fmt.Errorf("no occurrence within the 5-year search horizon after %s", cursor.Format("2006-01-02 15:04:05"))
		}
		out = append(out, WallClock{
			Year:   next.Year(),
			Month:  next.Month(),
			Day:    next.Day(),
			Hour:   next.Hour(),
			Minute: next.Minute(),
			Second: next.Second(),
		})
		cursor = next
	}
	return out, nil
}

func Parse(expr string) (*Spec, error) {
	trimmed := strings.TrimSpace(expr)
	if strings.HasPrefix(trimmed, "CRON_TZ=") || strings.HasPrefix(trimmed, "TZ=") {
		return nil, fmt.Errorf("timezone prefix in expression is not supported; use --tz instead")
	}
	schedule, err := cron.ParseStandard(trimmed)
	if err != nil {
		return nil, err
	}
	return &Spec{schedule: schedule}, nil
}
