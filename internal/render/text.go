package render

import (
	"fmt"
	"io"
	"text/tabwriter"
	"time"

	"github.com/umarsabirin/cronx/internal/dst"
	"github.com/umarsabirin/cronx/internal/plan"
)

func Table(w io.Writer, p plan.Plan) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	if _, err := fmt.Fprint(tw, "#\tLOCAL\tUTC\tNOTE\n"); err != nil {
		return err
	}
	for i, o := range p.Occurrences {
		local := fmt.Sprintf("%s %s %s",
			o.Local.Format("2006-01-02 15:04:05"), offset(zoneOffset(o.Local)), o.Abbrev)
		utc := o.UTC.Format("2006-01-02 15:04:05Z")
		line := fmt.Sprintf("%d\t%s\t%s", i+1, local, utc)
		if o.DupTotal != 0 {
			line += fmt.Sprintf("\tdup %d/%d", o.DupIndex, o.DupTotal)
		}
		if _, err := fmt.Fprintln(tw, line); err != nil {
			return err
		}
	}
	return tw.Flush()
}

func Warnings(w io.Writer, p plan.Plan) error {
	for _, warn := range p.Warnings {
		headline, verb, trailer := "occurs twice", "fall back", "listed as 2 separate fire times"
		if warn.Kind == dst.KindSkipped {
			headline, verb, trailer = "does not exist", "jump", "this occurrence will not fire"
		}
		t := warn.Transition
		_, err := fmt.Fprintf(w, "! %04d-%02d-%02d %02d:%02d %s in %s\n"+
			"    clocks %s %s -> %s (%s %s -> %s %s) at %s\n"+
			"    %s\n",
			warn.Wall.Year, warn.Wall.Month, warn.Wall.Day, warn.Wall.Hour, warn.Wall.Minute,
			headline, p.Zone,
			verb, shifted(t.At, t.FromOffset), shifted(t.At, t.ToOffset),
			t.FromAbbrev, offset(t.FromOffset), t.ToAbbrev, offset(t.ToOffset),
			t.At.UTC().Format(time.RFC3339),
			trailer)
		if err != nil {
			return err
		}
	}
	return nil
}

func shifted(at time.Time, seconds int) string {
	return at.UTC().Add(time.Duration(seconds) * time.Second).Format("15:04")
}

func zoneOffset(t time.Time) int {
	_, off := t.Zone()
	return off
}

func offset(seconds int) string {
	sign := "+"
	if seconds < 0 {
		sign, seconds = "-", -seconds
	}
	return fmt.Sprintf("%s%02d:%02d", sign, seconds/3600, seconds%3600/60)
}
