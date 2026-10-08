# 03 — Slices: `cronx`

Source: `.aiflow/02-design.md`. Test command for every slice is `go test ./...`, copied from
`.aiflow/project.yaml`.

| # | Slice | Primary files | Test that proves it |
|---|---|---|---|
| 1 | Module scaffold + `cronspec`: `go mod init github.com/umarsabirin/cronx`, require `robfig/cron/v3`, define `WallClock`, `Spec`, `Parse`, `NextWallClocks`. Rejects `CRON_TZ=` prefixes; converts robfig's zero-time sentinel into the no-occurrence error. | `go.mod`, `go.sum`, `internal/cronspec/spec.go`, `internal/cronspec/spec_test.go` | `go test ./...` — table-driven cases over field matching and `@daily`/`@every` descriptors; `CRON_TZ=America/New_York 0 9 * * *` returns an error; `0 0 30 2 *` returns the horizon error; an assertion that `Next` on a UTC-located time is pure field matching (same wall clocks regardless of the date's DST neighbourhood). |
| 2 | `dst.Resolve`: the ±36h candidate-offset probe, the render-back equality check, the 0/1/2 classification, and the binary search that fills `Transition` for non-normal cases. Pure stdlib, no I/O, no `time.Now()`. | `internal/dst/resolve.go`, `internal/dst/resolve_test.go` | `go test ./...` — fixed wall clocks against fixed zones: `America/New_York` 02:30 on the spring date → `KindSkipped` with 0 instants, 01:30 on the fall date → `KindDuplicated` with 2 instants and `-04:00 EDT` → `-05:00 EST`; `Australia/Lord_Howe` 30-minute gap; `Europe/Dublin` negative DST; `Asia/Jakarta` always `KindNormal`; `Pacific/Apia` 2011-12-30 whole-day skip. `Transition.At` asserted to the exact UTC instant. |
| 3 | `plan.Build`: drive `NextWallClocks`, resolve each wall clock, and account rows — a duplicate emits two `Occurrence`s and consumes two slots, a skip emits none and only a `Warning`. Pairs local/UTC and fills `Abbrev`, `DupIndex`, `DupTotal`. | `internal/plan/build.go`, `internal/plan/build_test.go` | `go test ./...` — with an injected `from` so nothing depends on the clock: `30 1 * * *` in `America/New_York` across the fall date with `n=3` yields rows `dup 1/2`, `dup 2/2`, next day; `30 2 * * *` across the spring date yields `n` rows and one skip warning, with no row for the skipped day. |
| 4 | `render`: `Table` and `Warnings` via `text/tabwriter` — numeric offset plus abbreviation on every row, `dup i/N` in the NOTE column, the three-line warning block. | `internal/render/text.go`, `internal/render/text_test.go` | `go test ./...` — golden strings for a `Plan` built in-test covering a normal row, a duplicated pair, and both warning shapes (skip and overlap); writers are `bytes.Buffer`, so stdout/stderr separation is asserted by which function was called. |
| 5 | `zoneinfo.Source`: `os.Stat` probe of `$ZONEINFO`, then the standard system zoneinfo paths, falling back to `debug.ReadBuildInfo().GoVersion` as `embedded (go1.x.y)`. | `internal/zoneinfo/source.go`, `internal/zoneinfo/source_test.go` | `go test ./...` — `t.Setenv("ZONEINFO", <tempfile>)` returns `$ZONEINFO=<path>`; an unset env with a non-existent probe root falls back to the `embedded (...)` form; the returned string is never empty. |
| 6 | CLI: flag parsing (`--tz`, `-n`, `--from`, `--version`), `time.LoadLocation`, `_ "time/tzdata"`, wiring slices 1–5, rows to stdout and warnings to stderr, and exit codes 0/1/2/3/4. The only file calling `os.Exit`. | `cmd/cronx/main.go`, `cmd/cronx/main_test.go` | `go test ./...` — `main` body factored into `run(args []string, stdout, stderr io.Writer) int` so the test calls it directly: missing `--tz` → 1, bad expression → 2, `--tz Mars/Phobos` → 3 with a message naming `ZONEINFO`, `0 0 30 2 *` → 4, and a happy path with a fixed `--from` asserting rows landed on stdout and warnings on stderr. |

Order follows the dependency arrows in the design: `cronspec` owns `WallClock`, so nothing can be
written before it; `dst` consumes `WallClock`; `plan` consumes both; `render` consumes `plan`'s
types. `zoneinfo` has no dependencies at all and could sit anywhere, so it sits at 5, adjacent to
`main`, its only consumer. Slice 1 carries a one-time setup risk worth naming early — the design
flags Go 1.22 in `go.mod` as its only unverified assumption, and `go mod init` is where that gets
confirmed or corrected. But the real risk is **slice 2**. It is the entire reason the tool exists,
it is the one piece with no prior art to copy, and three of its five test zones (Lord Howe's
30-minute shift, Dublin's negative DST, Apia's whole-day skip) are exactly the cases that break an
offset-arithmetic implementation — if the render-back equality check in step 3 is subtly wrong,
every slice downstream faithfully formats a wrong answer. Its tests also read the host's tzdata, so
a stale system database shows up as a test failure here first, which is the cheapest place to find
it.
