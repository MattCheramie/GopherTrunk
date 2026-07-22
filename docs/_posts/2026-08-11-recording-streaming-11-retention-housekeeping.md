---
title: "Recording, Composition & Streaming, Part 11: Retention & Housekeeping"
description: How GopherTrunk bounds disk and database growth with one idempotent background sweeper — call-row age, an allow-listed set of decoder-log tables, and mtime-based deletion of only .wav and .raw artifacts under an opt-in files root, each lane gated independently by config.
category: deep-dives
keywords: sdr recording retention, disk housekeeping sweeper, sqlite row retention, mtime file deletion, opt-in config gating, idempotent background job, decoder log table allow-list, gophertrunk retention
tags: [storage, retention, housekeeping, sqlite, go, filesystem]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "Recording, Composition & Streaming"
series_part: 11
---

*Part 11 of **Recording, Composition & Streaming**, following the 3 p.m.
dispatch on talkgroup 101 from PCM to a Broadcastify upload. By now that call is
two artifacts: a WAV the
[recorder]({{ '/blog/deep-dives/recording-streaming-04-recording-session/' | relative_url }})
wrote and a
[call_log row]({{ '/blog/deep-dives/recording-streaming-10-call-log-sqlite/' | relative_url }})
the persistence layer wrote. Multiply that by every call, every day, and a
scanner left running fills its disk. This post is about the janitor — the one
background job whose entire purpose is to make sure that yesterday's dispatch
doesn't outlive its welcome, without ever touching a file it wasn't told to.*

> **TL;DR:** `storage.Retention` is a single idempotent sweeper that runs at
> startup and then every `Interval`. It has three independent lanes — old
> `call_log` rows, old rows in an allow-listed set of decoder-log tables, and old
> `.wav` / `.raw` files under a `FilesRoot` — and **each lane runs only if its own
> max-age is set**. File deletion is doubly opt-in: it needs both a `FilesRoot` and
> a non-zero `FilesMaxAge`, and it deletes *only* recording artifacts by extension,
> never a config or CSV parked nearby. The daemon builds the sweeper at all only
> when at least one retention window is configured.

**Key takeaways**

- **Three lanes, three gates.** Call rows, decoder-log tables, and files are each
  swept only when their respective max-age is greater than zero. A zero age is a
  disabled lane, not "delete everything."
- **Opt-in by config.** No `retention:` window in the YAML → the daemon never
  constructs a `Retention` at all. File deletion additionally requires an explicit
  `FilesRoot`.
- **Deletes only recording artifacts.** The filesystem walk removes a file only if
  its extension is `.wav` or `.raw` *and* its mtime is past the cutoff — configs
  and talkgroup CSVs in the same tree are left alone.
- **Idempotent and crash-tolerant.** Every sweep recomputes cutoffs from
  `time.Now()`; errors are logged and swallowed so a transient FS or DB hiccup
  never kills the loop.

## Cheat sheet

| Thing | What it does | Where |
|---|---|---|
| `Retention` | The sweeper: three age-gated deletion lanes | `internal/storage/retention.go` |
| `RetentionOptions` | Per-lane max-ages, `FilesRoot`, `Interval` | `internal/storage/retention.go` |
| `SweepOnce` | Runs the configured deletions once | `internal/storage/retention.go` |
| `deleteOldRows` | `DELETE FROM call_log WHERE started_at < cutoff` | `internal/storage/retention.go` |
| `deleteOldLogRows` / `deleteOldByColumn` | Age-out one decoder-log table by its timestamp column | `internal/storage/retention.go` |
| `deleteOldFiles` / `isRecordingArtifact` | `filepath.Walk` + extension + mtime check | `internal/storage/retention.go` |
| `decoderLogTables` | The allow-list of sweepable log tables | `internal/storage/retention.go` |
| daemon wiring | Builds `Retention` only if a window is set | `cmd/gophertrunk/daemon.go` |

## In this post

- **The shape of a sweeper** — `Run`, `SweepOnce`, and the three gated lanes.
- **Aging out rows** — the call-log lane and the decoder-log allow-list.
- **Aging out files** — the walk, the extension guard, and the mtime cutoff.
- **Opt-in by config** — how the daemon decides whether a sweeper exists at all.
- **Idempotence** — why re-running the sweep is always safe.

## A janitor, not a pipeline

Retention is the one subsystem in this series that isn't a bus subscriber. It
reacts to a **timer**, not to call events — nothing about a call's life triggers
a sweep. `Run` does one sweep immediately at startup (so a daemon that was down
over a retention boundary catches up on boot) and then ticks on `Interval`:

```go
// internal/storage/retention.go (shape)
func (r *Retention) Run(ctx context.Context) error {
    r.SweepOnce(ctx)
    tick := time.NewTicker(r.interval)
    defer tick.Stop()
    for {
        select {
        case <-ctx.Done():
            return ctx.Err()
        case <-tick.C:
            r.SweepOnce(ctx)
        }
    }
}
```

`SweepOnce` is the whole policy in one function. It has three lanes, and each is
guarded by a two-part check: the relevant backend must exist (`r.db != nil`, or a
non-empty `r.files`) **and** that lane's max-age must be positive. A zero age
means the lane is off:

```go
// internal/storage/retention.go (shape)
func (r *Retention) SweepOnce(ctx context.Context) {
    if r.db != nil && r.dbAge > 0 {
        r.deleteOldRows(ctx) // call_log rows
    }
    if r.db != nil && r.logAge > 0 {
        for _, table := range decoderLogTables {
            r.deleteOldLogRows(ctx, table) // pager_log, aprs_log, …
        }
    }
    if r.files != "" && r.filesAge > 0 {
        r.deleteOldFiles() // .wav / .raw under FilesRoot
    }
}
```

That two-part gate is the design's spine: **presence of a backend is not consent
to delete.** A daemon with a database attached but no `CallRowMaxAge` sweeps zero
rows. Errors inside each lane are logged and swallowed — a locked file or a
transient DB error must not tear down the ticker — so the sweeper self-heals on
the next tick.

<figure class="lab-figure">
<svg viewBox="0 0 660 210" width="660" height="210" role="img" aria-label="Three independent sweep lanes fan out from SweepOnce. The first lane deletes call_log rows older than CallRowMaxAge and is gated by that age being greater than zero. The second lane iterates the decoder-log table allow-list, deleting rows older than LogRowMaxAge, gated by that age. The third lane walks FilesRoot deleting .wav and .raw files older than FilesMaxAge, gated by both a non-empty FilesRoot and that age. Each gate is independent.">
  <rect x="250" y="14" width="160" height="34" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="330" y="30" text-anchor="middle" fill="var(--accent)" font-size="11">SweepOnce</text>
  <text x="330" y="43" text-anchor="middle" fill="var(--fg-muted)" font-size="8">timer-driven, not events</text>
  <line x1="290" y1="48" x2="130" y2="86" stroke="currentColor"/><polygon points="130,82 122,90 138,90" fill="currentColor"/>
  <line x1="330" y1="48" x2="330" y2="86" stroke="currentColor"/><polygon points="326,86 330,94 334,86" fill="currentColor"/>
  <line x1="370" y1="48" x2="530" y2="86" stroke="currentColor"/><polygon points="530,82 538,90 522,90" fill="currentColor"/>
  <rect x="30" y="94" width="200" height="44" rx="6" fill="none" stroke="currentColor"/>
  <text x="130" y="112" text-anchor="middle" fill="currentColor" font-size="10">call_log rows</text>
  <text x="130" y="128" text-anchor="middle" fill="var(--fg-muted)" font-size="8">gate: dbAge &gt; 0</text>
  <rect x="230" y="94" width="200" height="44" rx="6" fill="none" stroke="currentColor"/>
  <text x="330" y="112" text-anchor="middle" fill="currentColor" font-size="10">decoder-log tables</text>
  <text x="330" y="128" text-anchor="middle" fill="var(--fg-muted)" font-size="8">gate: logAge &gt; 0 · allow-list</text>
  <rect x="430" y="94" width="200" height="44" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="530" y="112" text-anchor="middle" fill="var(--accent)" font-size="10">.wav / .raw files</text>
  <text x="530" y="128" text-anchor="middle" fill="var(--fg-muted)" font-size="8">gate: FilesRoot ≠ "" &amp; filesAge &gt; 0</text>
  <text x="130" y="160" text-anchor="middle" fill="var(--fg-muted)" font-size="9">started_at &lt; cutoff</text>
  <text x="330" y="160" text-anchor="middle" fill="var(--fg-muted)" font-size="9">received_at &lt; cutoff</text>
  <text x="530" y="160" text-anchor="middle" fill="var(--fg-muted)" font-size="9">mtime &lt; cutoff</text>
  <text x="330" y="192" text-anchor="middle" fill="var(--fg-muted)" font-size="9">Each lane is independent — a zero max-age disables just that lane.</text>
</svg>
<figcaption>Three sweep lanes, three independent gates. A missing max-age silences one lane without affecting the others; nothing here is all-or-nothing.</figcaption>
</figure>

## Aging out rows

The database lanes are almost too simple to describe, which is the point. The
call-log lane computes a cutoff from *now* on every sweep and deletes anything
older:

```go
// internal/storage/retention.go (shape)
func (r *Retention) deleteOldRows(ctx context.Context) (int64, error) {
    cutoff := time.Now().Add(-r.dbAge).UnixNano()
    res, err := r.db.sql.ExecContext(ctx,
        `DELETE FROM call_log WHERE started_at < ?`, cutoff)
    // …return RowsAffected…
}
```

`TestRetentionDeletesOldRows` pins the boundary: with a 24-hour age, a 25-hour-old
row is deleted while 10-hour and 1-hour rows survive.

The decoder-log lane is where the allow-list matters. GopherTrunk decodes far
more than trunked voice — POCSAG pagers, APRS, AIS vessels, DSC, ADS-B aircraft,
MDC1200, M17 — and each writes an append-only log table keyed by a `received_at`
nanosecond column. The sweeper knows exactly which tables it may touch:

```go
// internal/storage/retention.go (shape)
var decoderLogTables = []string{
    "pager_log", "aprs_log", "vessel_log", "dsc_log",
    "aircraft_log", "mdc1200_log", "m17_log",
}
```

`deleteOldLogRows` is a thin wrapper over `deleteOldByColumn(ctx, table,
"received_at")`, which interpolates the table and column names directly into the
`DELETE`. That string-building is safe *precisely because* the inputs are
in-code constants — the allow-list slice or a literal — never user input. The
comment in the source says so explicitly, and it's the reason `location_log`
(which keys on `reported_at`, not `received_at`) is swept by a separate call
rather than being bolted into the loop. `TestRetentionLogSweepEveryTable` runs the
delete against every allow-listed table plus the `location_log` special case, so a
typo or a table missing its timestamp column would fail the build's tests.

Note the two log lanes share one `logAge`: every decoder-log table ages out on the
same window. `TestRetentionLogSweepDisabledByDefault` confirms the gate — with
`CallRowMaxAge` set but `LogRowMaxAge` zero, a 100-day-old `m17_log` row survives
untouched.

## Aging out files

The filesystem lane is the one that needs the most care, because a wrong deletion
here is unrecoverable and the recordings directory is a place operators park other
things. `deleteOldFiles` walks `FilesRoot` and applies two filters to every file
it finds — an extension check and an mtime check — deleting only when *both* pass:

```go
// internal/storage/retention.go (shape)
func (r *Retention) deleteOldFiles() (int, error) {
    cutoff := time.Now().Add(-r.filesAge)
    return filepath.Walk(r.files, func(path string, info os.FileInfo, err error) error {
        if err != nil { return nil }        // tolerate transient FS errors
        if info.IsDir() { return nil }
        if !isRecordingArtifact(path) { return nil } // only .wav / .raw
        if info.ModTime().After(cutoff) { return nil } // still fresh
        os.Remove(path)
        return nil
    })
}

func isRecordingArtifact(path string) bool {
    switch strings.ToLower(filepath.Ext(path)) {
    case ".wav", ".raw":
        return true
    }
    return false
}
```

The extension guard is deliberate defensive coding. As the source comment notes,
it's there so the sweep doesn't clobber configs or talkgroup CSVs an operator
parked near the recordings. `TestRetentionDeletesOldFiles` makes the point
concretely: in a tree containing an old `.wav`, an old `.raw`, a fresh `.wav`, and
an old `config.yaml`, the sweep removes the two old artifacts, keeps the fresh
one, and leaves the `config.yaml` alone even though it's well past the cutoff — the
extension check saves it.

Two more details worth noting. The walk swallows per-entry errors by returning
`nil` from the walk function, so one unreadable directory doesn't abort the whole
sweep. And the mtime comparison uses the file's own modification time, not any
database record — the file lane needs no DB at all, which is why `NewRetention`
accepts a `FilesRoot`-only configuration with no `DB`.

<figure class="lab-figure">
<svg viewBox="0 0 640 250" width="640" height="250" role="img" aria-label="A decision tree for one file encountered during the filesystem walk. First: is it a directory? If yes, skip. Otherwise: is the extension .wav or .raw? If no, skip and preserve the file. Otherwise: is the modification time newer than the cutoff? If yes, skip because it is still fresh. Otherwise remove the file.">
  <rect x="240" y="12" width="160" height="30" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="320" y="31" text-anchor="middle" fill="var(--accent)" font-size="10">walk entry</text>
  <line x1="320" y1="42" x2="320" y2="60" stroke="currentColor"/><polygon points="316,60 320,68 324,60" fill="currentColor"/>
  <rect x="230" y="68" width="180" height="30" rx="6" fill="none" stroke="currentColor"/>
  <text x="320" y="87" text-anchor="middle" fill="currentColor" font-size="10">IsDir?</text>
  <line x1="410" y1="83" x2="500" y2="83" stroke="currentColor"/><polygon points="500,79 508,83 500,87" fill="currentColor"/>
  <text x="455" y="78" text-anchor="middle" fill="var(--fg-muted)" font-size="8">yes</text>
  <rect x="508" y="68" width="110" height="30" rx="6" fill="none" stroke="var(--fg-muted)"/>
  <text x="563" y="87" text-anchor="middle" fill="var(--fg-muted)" font-size="9">skip</text>
  <line x1="320" y1="98" x2="320" y2="116" stroke="currentColor"/><polygon points="316,116 320,124 324,116" fill="currentColor"/>
  <text x="336" y="112" text-anchor="middle" fill="var(--fg-muted)" font-size="8">no</text>
  <rect x="220" y="124" width="200" height="30" rx="6" fill="none" stroke="currentColor"/>
  <text x="320" y="143" text-anchor="middle" fill="currentColor" font-size="10">ext .wav / .raw?</text>
  <line x1="420" y1="139" x2="500" y2="139" stroke="currentColor"/><polygon points="500,135 508,139 500,143" fill="currentColor"/>
  <text x="460" y="134" text-anchor="middle" fill="var(--fg-muted)" font-size="8">no</text>
  <rect x="508" y="124" width="120" height="30" rx="6" fill="none" stroke="var(--fg-muted)"/>
  <text x="568" y="143" text-anchor="middle" fill="var(--fg-muted)" font-size="9">preserve</text>
  <line x1="320" y1="154" x2="320" y2="172" stroke="currentColor"/><polygon points="316,172 320,180 324,172" fill="currentColor"/>
  <text x="336" y="168" text-anchor="middle" fill="var(--fg-muted)" font-size="8">yes</text>
  <rect x="220" y="180" width="200" height="30" rx="6" fill="none" stroke="currentColor"/>
  <text x="320" y="199" text-anchor="middle" fill="currentColor" font-size="10">mtime &gt; cutoff?</text>
  <line x1="420" y1="195" x2="500" y2="195" stroke="currentColor"/><polygon points="500,191 508,195 500,199" fill="currentColor"/>
  <text x="458" y="190" text-anchor="middle" fill="var(--fg-muted)" font-size="8">yes (fresh)</text>
  <rect x="508" y="180" width="120" height="30" rx="6" fill="none" stroke="var(--fg-muted)"/>
  <text x="568" y="199" text-anchor="middle" fill="var(--fg-muted)" font-size="9">skip</text>
  <line x1="320" y1="210" x2="320" y2="226" stroke="var(--accent)"/><polygon points="316,226 320,234 324,226" fill="var(--accent)"/>
  <text x="336" y="223" text-anchor="middle" fill="var(--fg-muted)" font-size="8">no (old)</text>
  <rect x="250" y="226" width="140" height="24" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="320" y="242" text-anchor="middle" fill="var(--accent)" font-size="10">os.Remove</text>
</svg>
<figcaption>The per-file decision tree. A file is deleted only after passing both the extension guard and the mtime cutoff; every other path preserves it.</figcaption>
</figure>

## Opt-in by config

None of this exists unless the operator asks for it. In
`cmd/gophertrunk/daemon.go`, the sweeper is constructed only when at least one
retention window is set:

```go
// cmd/gophertrunk/daemon.go (shape)
if cfg.Retention.CallLogDays > 0 ||
    cfg.Retention.LogDays > 0 ||
    cfg.Retention.FilesDays > 0 {
    interval, _ := retentionInterval(cfg.Retention.Interval)
    r, _ := storage.NewRetention(storage.RetentionOptions{
        DB:            db,
        FilesRoot:     cfg.Recordings.Dir,
        CallRowMaxAge: time.Duration(cfg.Retention.CallLogDays) * 24 * time.Hour,
        FilesMaxAge:   time.Duration(cfg.Retention.FilesDays) * 24 * time.Hour,
        Interval:      interval,
    })
    d.retention = r
}
```

This is the same **config-driven lazy init** rule that runs through the whole
output half (see
[Part 1]({{ '/blog/deep-dives/recording-streaming-01-the-output-half/' | relative_url }})):
with no retention days configured, `d.retention` stays nil and the daemon never
spawns the sweep goroutine. The `FilesRoot` is wired from the recordings directory,
so file deletion piggybacks on wherever the recorder is already writing — but only
takes effect when `FilesDays` (hence `FilesMaxAge`) is non-zero, keeping file
deletion doubly opt-in. `retentionInterval` parses the optional interval string,
defaulting to one hour when it's empty, mirroring `NewRetention`'s own default.

`NewRetention` itself refuses a pointless configuration:
`TestRetentionRequiresAtLeastOne` confirms it errors when neither a `DB` nor a
`FilesRoot` is supplied — there'd be nothing to sweep.

## Idempotence

The reason this sweeper can run at startup, on every tick, and even concurrently
with the writers is that it holds no state between runs. Every lane recomputes its
cutoff from `time.Now()` at the moment it executes; a `DELETE ... WHERE ts <
cutoff` that finds nothing to delete is a no-op, and re-deleting an already-deleted
row or file simply affects zero rows or is skipped by the walk. There is no
"mark-then-sweep" bookkeeping to get out of sync.

Concurrency with the [call log]({{ '/blog/deep-dives/recording-streaming-10-call-log-sqlite/' | relative_url }})
writer is safe because SQLite serialises writers — the store caps the pool to a
single connection, so the sweeper's `DELETE` and the call log's `UPDATE` take
turns rather than racing. `TestRetentionRunStopsOnContextCancel` confirms the loop
exits cleanly on context cancellation, returning the cancellation error like every
other long-lived subsystem in the daemon.

## Where this goes next

That's the persistence and housekeeping pair complete — the call is now recorded,
indexed, and destined to be cleaned up on schedule. The next posts turn to the
*live* side of the output half.
[Part 12]({{ '/blog/deep-dives/recording-streaming-12-live-listening/' | relative_url }})
covers live listening: how the same decoded PCM that fills the WAV is fanned out
to a browser or gRPC listener in real time, with no file and no database involved.

## FAQ

**Does GopherTrunk delete my recordings by default?**
No. File deletion is doubly opt-in: it needs both a recordings directory
(`FilesRoot`) and a non-zero `retention.files_days`. With no `retention:` window
configured at all, the daemon never even constructs the sweeper, so nothing is
ever deleted.

**Will the sweeper remove non-recording files in the recordings folder?**
No. `isRecordingArtifact` only matches `.wav` and `.raw` extensions. A
`config.yaml`, a talkgroup CSV, or any other file in the same tree is skipped
regardless of age — the extension guard exists specifically to protect files
operators park nearby.

**How do the three retention lanes relate?**
They're independent. Call-row age (`call_log_days`), decoder-log age
(`log_days`), and file age (`files_days`) each gate their own lane; setting one
does nothing to the others, and a zero value disables just that lane rather than
meaning "delete everything."

**Is it safe to run retention while calls are being logged?**
Yes. The sweep is idempotent — it recomputes cutoffs from the current time each
run and re-running deletes nothing already gone — and SQLite serialises writers
against a single connection, so the sweeper's deletes and the call log's writes
take turns instead of racing.

## Series navigation

**Part 11 of 14** · ←
[Part 10: The Call Log — SQLite Persistence & Never-Downgrade Updates]({{ '/blog/deep-dives/recording-streaming-10-call-log-sqlite/' | relative_url }})
· Next →
[Part 12: Live Listening]({{ '/blog/deep-dives/recording-streaming-12-live-listening/' | relative_url }})
