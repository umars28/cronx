# 02 — Design: `cronx`

Empty repository — nothing to imitate, so this document defines the template every later slice follows, as decisions rather than suggestions.

**Template decisions**

- Module path `github.com/umarsabirin/cronx` (no git remote configured; taken from the git user). Go 1.22 in `go.mod` — the only unverified assumption here, since `go version` was not runnable; confirm at `go mod init`.
- Layout: one binary under `cmd/cronx`, all logic in `internal/` packages. Nothing exported outside the module; this is a tool, not a library.
- **One external dependency: `github.com/robfig/cron/v3`**, and it is reachable from exactly one package (`internal/cronspec`). robfig/cron v3 has no transitive dependencies of its own. Everything else is stdlib.
- Errors are values returned up to `main`; `main` is the only place that writes to stderr and the only place that calls `os.Exit`. No `log.Fatal` in `internal/`.
- Tests are table-driven `_test.go` files beside the code, stdlib `testing` only, no assertion library.

**Core idea.** A cron expression names *wall-clock* times, not instants. So the pipeline is: generate wall clocks (no timezone involved) → resolve each wall clock against the IANA zone → classify the result. A wall clock resolves to exactly one instant normally, **zero** during a spring-forward gap (skipped), **two** during a fall-back overlap (duplicated). Skip and duplicate are therefore a classification with a definite answer, not an artifact of how some iterator happened to add durations.

## Public surface

### CLI

```
cronx [flags] <cron-expression>

  --tz string     IANA timezone name, e.g. America/New_York (required)
  -n int          number of fire times to print (default 10)
  --from string   RFC3339 instant to start after (default: now)
  --version       print version, Go build version, and resolved zoneinfo source
```

Expression grammar: standard 5-field cron (`minute hour dom month dow`) plus `@daily`/`@hourly`/`@weekly`/`@monthly`/`@yearly`/`@every` — i.e. exactly what `cron.ParseStandard` accepts. No seconds field. A `CRON_TZ=` prefix in the expression is **rejected**; `--tz` is the single source of timezone truth.

Fire-time rows go to **stdout**, warnings to **stderr**, so `cronx ... | column -t` stays clean while warnings still surface. Warnings do not change the exit code.

```
#   LOCAL                           UTC                   NOTE
1   2026-11-01 01:30:00 -04:00 EDT  2026-11-01 05:30:00Z  dup 1/2
2   2026-11-01 01:30:00 -05:00 EST  2026-11-01 06:30:00Z  dup 2/2
3   2026-11-02 01:30:00 -05:00 EST  2026-11-02 06:30:00Z
```

```
! 2026-03-08 02:30 does not exist in America/New_York
    clocks jump 02:00 -> 03:00 (EST -04:00 -> EDT -05:00) at 2026-03-08T07:00:00Z
    this occurrence will not fire
! 2026-11-01 01:30 occurs twice in America/New_York
    clocks fall back 02:00 -> 01:00 (EDT -04:00 -> EST -05:00) at 2026-11-01T06:00:00Z
    listed as 2 separate fire times
```

Exit codes: `0` success (with or without warnings), `1` usage error, `2` invalid cron expression, `3` unknown timezone, `4` expression has no occurrence within robfig's 5-year search horizon.

### Go

```go
// internal/cronspec
type WallClock struct{ Year int; Month time.Month; Day, Hour, Minute, Second int }
type Spec struct{ /* wraps cron.Schedule */ }
func Parse(expr string) (*Spec, error)
func (s *Spec) NextWallClocks(after WallClock, n int) ([]WallClock, error)

// internal/dst
type Kind int
const ( KindNormal Kind = iota; KindSkipped; KindDuplicated )
type Resolution struct {
    Wall       cronspec.WallClock
    Kind       Kind
    Instants   []time.Time // len 1 normal, 0 skipped, 2 duplicated
    Transition Transition  // zero value when KindNormal
}
type Transition struct{ At time.Time; FromOffset, ToOffset int; FromAbbrev, ToAbbrev string }
func Resolve(w cronspec.WallClock, loc *time.Location) Resolution

// internal/plan
type Occurrence struct {
    Local, UTC         time.Time
    Abbrev             string
    DupIndex, DupTotal int // 0,0 when not duplicated
}
type Warning struct{ Wall cronspec.WallClock; Kind dst.Kind; Transition dst.Transition }
type Plan struct{ Occurrences []Occurrence; Warnings []Warning }
func Build(s *cronspec.Spec, loc *time.Location, from time.Time, n int) (Plan, error)

// internal/render
func Table(w io.Writer, p Plan) error
func Warnings(w io.Writer, p Plan) error

// internal/zoneinfo
func Source() string // "$ZONEINFO=...", "/usr/share/zoneinfo", or "embedded (go1.22.x)"
```

`dst` imports `cronspec` for `WallClock` (producer → consumer, no cycle). `plan` imports both.

### The resolution algorithm

`dst.Resolve` is the whole point of the tool, so it is specified here rather than left to the implementer:

1. Build `probe := time.Date(w..., time.UTC)` — the wall clock carried as a naive datetime.
2. Collect candidate offsets: the `Zone()` offsets of `probe.Add(-36h).In(loc)`, `probe.In(loc)`, and `probe.Add(+36h).In(loc)`.
3. For each distinct offset `o`, form `c := probe.Add(-o)` and keep `c` **only if `c.In(loc)` renders wall clock exactly `w`**. This equality check is what makes the method offset-agnostic — it is correct for Lord Howe's 30-minute shift and for Ireland's negative DST without special cases.
4. Sort and dedupe. 0 instants → `KindSkipped`; 1 → `KindNormal`; 2 → `KindDuplicated`.
5. Only for the non-normal cases, binary-search the ±36h window for the offset-change instant to populate `Transition` (~18 probes). This is what makes the warning actionable rather than just alarming.

No tzdata parsing, no transition table, O(1) per occurrence.

`plan.Build` drives `cronspec.NextWallClocks` and keeps pulling until it has emitted `n` **rows**. A duplicated wall clock emits two rows and consumes two slots; a skipped wall clock emits zero rows and only a warning.

## Modules touched

All created — the repo is empty.

| File | Why |
|---|---|
| `go.mod`, `go.sum` | Module `github.com/umarsabirin/cronx`, single require on `robfig/cron/v3`. |
| `cmd/cronx/main.go` | Flag parsing, `time.LoadLocation`, wiring, exit codes. Imports `_ "time/tzdata"` — see failure mode. The only file that touches `os.Exit`. |
| `internal/cronspec/spec.go` | `Parse` + `NextWallClocks`. The sole importer of `robfig/cron`; quarantining it here is what lets the dependency be swapped later. Rejects `CRON_TZ=` prefixes and converts robfig's zero-time "no match" sentinel into exit-code-4's error. |
| `internal/cronspec/spec_test.go` | Field matching, descriptors, the `CRON_TZ` rejection, and the 5-year-horizon case (`0 0 30 2 *`). Must also assert the assumption this design leans on: `Next` on a UTC-located time is pure field matching with no DST behaviour. |
| `internal/dst/resolve.go` | The algorithm above. Pure stdlib, no I/O — the most valuable and most testable code in the project. |
| `internal/dst/resolve_test.go` | Fixed wall clocks against fixed zones: `America/New_York` spring gap and fall overlap, `Australia/Lord_Howe` 30-minute gap, `Europe/Dublin` negative DST, `Asia/Jakarta` (no DST, every case normal), and `Pacific/Apia`'s 2011 whole-day skip. No `time.Now()` anywhere. |
| `internal/plan/build.go` | Row accounting: the `n`-rows-not-`n`-wall-clocks rule, warning collection, local/UTC pairing. |
| `internal/plan/build_test.go` | That a duplicate consumes two slots and a skip consumes none, driven by an injected `from` so results are deterministic. |
| `internal/render/text.go` | Table and warning formatting via `text/tabwriter`. Split from `plan` so output format is assertable with golden strings. |
| `internal/render/text_test.go` | Golden output for the normal, skipped, and duplicated rows. |
| `internal/zoneinfo/source.go` | `os.Stat` probe of `$ZONEINFO` then the standard system paths, falling back to `debug.ReadBuildInfo().GoVersion`. Feeds `--version`. |

## Failure mode

**The binary reports confident, wrong DST warnings because the host's tzdata disagrees with reality.**

Zone rules are political and change with weeks of notice. `time.LoadLocation` resolves against `$ZONEINFO`, then the system zoneinfo directories, then `$GOROOT/lib/time/zoneinfo.zip`. Two things go wrong in production:

1. **No tzdata at all.** A `scratch` or distroless container image has no `/usr/share/zoneinfo` and no Go toolchain. Every invocation fails with `unknown time zone America/New_York`, for every zone. The tool is simply dead in the exact deployment shape a small Go CLI tends to get.
2. **Stale tzdata — the expensive one.** A long-lived base image pinned in 2018 still believes São Paulo observes DST. `cronx '30 2 * * *' --tz America/Sao_Paulo` then warns that an October occurrence will be skipped. It will not be; Brazil abolished DST in 2019. The user reschedules a nightly billing job around a transition that does not exist. A tool whose only job is to warn about DST, warning wrongly about DST, is worse than no tool — it is trusted, and it is silent in the symmetric case where a *new* transition was added and no warning is printed at all.

Design response, in order of what it actually buys:

- `cmd/cronx/main.go` imports `_ "time/tzdata"`. This embeds the IANA database in the binary and registers it as the final fallback, which **eliminates case 1 entirely** — a scratch container now resolves every zone.
- It does **not** fix case 2: the system database still takes precedence when it exists. So the design makes the data source *attributable* instead of pretending to fix it. `cronx --version` prints `internal/zoneinfo.Source()` — the actual path the zones came from, or `embedded (go1.22.x)` — so a wrong-answer report starts with a fact rather than a guess, and `ZONEINFO=/path/to/tzdata.zip` is the documented one-line override.
- Every row prints the **numeric offset and abbreviation** (`-04:00 EDT`), never a bare zone name. A stale ruleset shows up in the output a user is already reading, rather than hiding behind a label that looks identical either way.
- Unknown zone exits `3` with a message naming `ZONEINFO`, distinct from a bad-expression `2`, so a container misconfiguration is not mistaken for user error in a CI log.

## Options rejected

**1. Generate the list with robfig's `Next` directly in the target location.** The obvious path — `cron.ParseStandard` with `CRON_TZ=`, then call `Next` N times — and the one rejected most deliberately. `SpecSchedule.Next` advances by adding absolute durations to a zoned `time.Time`, so DST behaviour is an *emergent artifact of the arithmetic*: spring-forward occurrences silently vanish, fall-back occurrences silently appear twice. To warn, I would have to reverse-engineer which happened by diffing consecutive outputs — and that cannot distinguish "skipped by DST" from an expression that legitimately has a 48-hour gap (`30 2 * * 1`). Generating wall clocks in UTC and resolving them myself turns skip/duplicate into a classification with a definite answer, and confines robfig to field matching, where it is unambiguous.

**2. Scan tzdata for transitions in the window, then intersect with the schedule.** Go exports no transition list, so this means binary-searching `Zone()` across the whole listed window. The window is unbounded for sparse expressions — `0 0 29 2 *` puts the next match years out — and it recomputes, at O(window), information the per-occurrence resolution already produces in O(1). It also inverts the dependency: the DST logic would need to know the schedule's extent before it could run.

**3. Hand-roll the cron parser for a zero-dependency binary.** Tempting, and robfig is small enough to vendor mentally. Rejected because the differentiating work here is DST classification, not field parsing; robfig/cron v3 is itself dependency-free so the supply-chain argument is thin; and re-litigating step ranges, `?`, and day-of-week/day-of-month OR semantics buys nothing a user would notice.

**4. Count a duplicated occurrence as one of the N.** `-n 10` could mean ten *matched wall clocks* or ten *fire times*. The request says "the next N fire times", and a fall-back duplicate genuinely is two executions on a wall-clock-driven daemon, so a duplicate consumes two slots. This is the request's ambiguity and this is the reading taken.

**5. List skipped occurrences in the table, struck through or marked `SKIP`.** It reads well, but a skipped occurrence is not a fire time, and putting it in the same table invites a reader scanning the `#` column to count it as one. Skips appear only as warnings, on stderr, where they cannot be mistaken for output.

**6. `--strict` to exit non-zero when any warning fires.** Genuinely useful for CI gating, and the exit-code space is already reserved for it. Out of scope for this change; the request asks the tool to warn, not to gate.
