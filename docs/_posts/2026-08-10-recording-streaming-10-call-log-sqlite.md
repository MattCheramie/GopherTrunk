---
title: "Recording, Composition & Streaming, Part 10: The Call Log — SQLite Persistence & Never-Downgrade Updates"
description: How GopherTrunk indexes every call into a CGO-free SQLite call_log by subscribing to the bus — an INSERT on start, an UPDATE on end keyed by device and timestamp, and a COALESCE/NULLIF discipline that backfills late RID and encryption without ever clobbering a value it already knew.
category: deep-dives
keywords: sqlite call log persistence, modernc pure go sqlite, coalesce nullif never downgrade, call history rest api, insert on start update on end, p25 rid backfill, event bus subscriber storage, gophertrunk call_log schema
tags: [storage, sqlite, persistence, events, go, database]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "Recording, Composition & Streaming"
series_part: 10
---

*Part 10 of **Recording, Composition & Streaming**, following one call — the
3 p.m. dispatch on talkgroup 101 — from PCM to a Broadcastify upload. The
[recorder]({{ '/blog/deep-dives/recording-streaming-04-recording-session/' | relative_url }})
is busy turning that call into a WAV; this post is about the other subscriber
that woke up when the call started. While the audio path writes bytes to disk,
the **call log** writes one row to SQLite so the dispatch is searchable months
later — and it has to write that row before anyone knows who was talking. This
is the story of how a row that starts life half-blank gets filled in without
ever being made worse.*

> **TL;DR:** The call log is a bus subscriber that keeps `call_log` in sync with
> live calls. `KindCallStart` does an INSERT with a null `ended_at`;
> `KindCallEnd` does an UPDATE keyed by `(device_serial, started_at)` that fills
> duration, end reason, and quality. The catch is that a P25 caller's RID and
> encryption often arrive *after* the call started, so the UPDATE uses
> `COALESCE(NULLIF(?, 0), col)` to backfill late values while **never** letting a
> blank end-grant overwrite an identity the start already captured. Reads go
> through `DB.History`, which is what the REST `/api/v1/calls/history` endpoint
> serves.

**Key takeaways**

- The call log is **two SQL statements**: INSERT on `KindCallStart`, UPDATE on
  `KindCallEnd`, both keyed by `(device_serial, started_at)`. A unique index makes
  duplicate starts idempotent.
- **Never downgrade.** `COALESCE(NULLIF(?, 0), col)` backfills a late RID / ALGID
  / KID but keeps a known value when the end grant is zero; encryption only ever
  goes clear→encrypted, never back.
- **Unset is not zero.** `signal_dbfs`, `evm_pct`, and `snr_db` are nullable and
  bound as `sql.NullFloat64`, so a non-composer end (a watchdog timeout) writes
  NULL and leaves an earlier measurement intact.
- The store is **pure-Go SQLite** (`modernc.org/sqlite`), so `CGO_ENABLED=0`
  holds across the daemon and it cross-compiles to `linux/arm64` with no toolchain.

## Cheat sheet

| Thing | What it does | Where |
|---|---|---|
| `CallLog` | Bus subscriber that keeps `call_log` current | `internal/storage/calllog.go` |
| `NewCallLog` | Subscribes to the bus immediately; `Run` drains later | `internal/storage/calllog.go` |
| `recordStart` | `INSERT OR REPLACE` on `KindCallStart`, `ended_at` NULL | `internal/storage/calllog.go` |
| `recordEnd` | Never-downgrade `UPDATE` on `KindCallEnd` | `internal/storage/calllog.go` |
| `call_log` table | Schema + idempotency index, migrated on Open | `internal/storage/sqlite.go` |
| `HistoryFilter` / `CallRow` | Read query shape + row DTO | `internal/storage/calllog.go` |
| `DB.History` | Newest-first read behind the REST history endpoint | `internal/storage/calllog.go` |

## In this post

- **Two writes, one key** — the INSERT/UPDATE split and why the row starts blank.
- **The never-downgrade SQL** — `COALESCE`, `NULLIF`, and the encryption `CASE`.
- **Unset ≠ zero** — nullable quality columns and why a watchdog end writes NULL.
- **The schema** — pure-Go SQLite, the idempotency index, and forward migrations.
- **The read path** — `HistoryFilter` → `DB.History` → the REST endpoint.

## A row that starts blank

When the engine grants the 3 p.m. dispatch and retunes a voice SDR, it publishes
`KindCallStart` on the same bus the recorder listens to (see
[Part 1]({{ '/blog/deep-dives/recording-streaming-01-the-output-half/' | relative_url }})).
The `CallLog` subscriber is one of the objects that wakes up. Its whole job is to
keep the `call_log` table tracking the live call, and it does that with exactly
two SQL statements.

`NewCallLog` subscribes to the bus **immediately**, in the constructor, so a
caller can publish events before `Run` is even scheduled — the subscription
buffers them:

```go
// internal/storage/calllog.go (shape)
type CallLog struct {
    db  *DB
    bus *events.Bus
    sub *events.Subscription
    // …
}

func NewCallLog(db *DB, bus *events.Bus, logger *slog.Logger) (*CallLog, error) {
    // …validate db + bus…
    cl := &CallLog{db: db, bus: bus, /* … */}
    cl.sub = bus.Subscribe() // subscribe now; drain in Run
    return cl, nil
}
```

`Run` is a plain select loop over the subscription channel. It cares about
exactly two of the four call events — start and end — and ignores segment and
complete entirely (those belong to the recorder and the uploader):

```go
// internal/storage/calllog.go (shape)
func (c *CallLog) Run(ctx context.Context) error {
    for {
        select {
        case <-ctx.Done():
            return ctx.Err()
        case ev := <-c.sub.C:
            switch ev.Kind {
            case events.KindCallStart:
                cs := ev.Payload.(trunking.CallStart)
                c.recordStart(cs) // INSERT, ended_at NULL
            case events.KindCallEnd:
                ce := ev.Payload.(trunking.CallEnd)
                c.recordEnd(ce)   // UPDATE, fill the rest
            }
        }
    }
}
```

`recordStart` writes what the grant knows at grant time — system, protocol,
group, frequency, the device serial, and the start timestamp in Unix
nanoseconds. It does *not* know how long the call ran, why it ended, or how loud
it was, so those columns stay null:

```go
// internal/storage/calllog.go (shape)
func (c *CallLog) recordStart(cs trunking.CallStart) error {
    const q = `
INSERT OR REPLACE INTO call_log (
    system, protocol, group_id, source_id, frequency_hz,
    encrypted, algorithm_id, key_id, emergency, data_call, timeslot,
    device_serial, started_at, talkgroup_alpha
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
    // …bind grant fields; started_at = cs.StartedAt.UnixNano()…
}
```

<figure class="lab-figure">
<svg viewBox="0 0 660 210" width="660" height="210" role="img" aria-label="A sequence: the engine publishes call.start, which the call log turns into an INSERT with ended_at NULL; the call runs; then the engine publishes call.end, which the call log turns into an UPDATE keyed by device serial and started_at that fills duration, end reason, and quality figures into the same row.">
  <rect x="10" y="16" width="130" height="30" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="75" y="35" text-anchor="middle" fill="var(--accent)" font-size="11">trunking engine</text>
  <rect x="270" y="16" width="120" height="30" rx="6" fill="none" stroke="currentColor"/>
  <text x="330" y="35" text-anchor="middle" fill="currentColor" font-size="11">CallLog.Run</text>
  <rect x="520" y="16" width="130" height="30" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="585" y="35" text-anchor="middle" fill="var(--accent)" font-size="11">call_log row</text>
  <line x1="75" y1="46" x2="75" y2="200" stroke="var(--fg-muted)" stroke-dasharray="3 3"/>
  <line x1="330" y1="46" x2="330" y2="200" stroke="var(--fg-muted)" stroke-dasharray="3 3"/>
  <line x1="585" y1="46" x2="585" y2="200" stroke="var(--fg-muted)" stroke-dasharray="3 3"/>
  <line x1="75" y1="66" x2="326" y2="66" stroke="currentColor"/><polygon points="326,62 336,66 326,70" fill="currentColor"/>
  <text x="200" y="61" text-anchor="middle" fill="currentColor" font-size="9">KindCallStart</text>
  <line x1="330" y1="86" x2="581" y2="86" stroke="var(--accent)"/><polygon points="581,82 591,86 581,90" fill="var(--accent)"/>
  <text x="458" y="81" text-anchor="middle" fill="var(--accent)" font-size="9">INSERT · ended_at = NULL</text>
  <text x="585" y="108" text-anchor="middle" fill="var(--fg-muted)" font-size="8">(active call — searchable now)</text>
  <line x1="75" y1="140" x2="326" y2="140" stroke="currentColor"/><polygon points="326,136 336,140 326,144" fill="currentColor"/>
  <text x="200" y="135" text-anchor="middle" fill="currentColor" font-size="9">KindCallEnd</text>
  <line x1="330" y1="164" x2="581" y2="164" stroke="var(--accent)"/><polygon points="581,160 591,164 581,168" fill="var(--accent)"/>
  <text x="458" y="159" text-anchor="middle" fill="var(--accent)" font-size="9">UPDATE WHERE device_serial, started_at</text>
  <text x="458" y="184" text-anchor="middle" fill="var(--fg-muted)" font-size="8">fill duration · reason · signal · evm · snr</text>
</svg>
<figcaption>Two writes, one row. Start inserts an active call the instant the SDR retunes; end updates the same row by its natural key once the call is over.</figcaption>
</figure>

The pair `(device_serial, started_at)` is the row's natural key: one physical
voice SDR can only carry one call at a time, so the serial plus the start
timestamp uniquely names a call. That is exactly the unique index the schema
declares, which is why `recordStart` can use `INSERT OR REPLACE` and stay
idempotent — three duplicate `KindCallStart` events (a real possibility on a
noisy control channel) collapse to a single row, as
`TestCallLogIdempotentStart` pins.

## The never-downgrade UPDATE

Here is the subtlety that makes this post worth writing. On a P25 Phase 2
compressed grant, the caller's **radio ID (RID)** is not on the control channel
at grant time — `source_id` starts at 0. It surfaces mid-call on the traffic
channel. Encryption is the same: the algorithm ID and key ID often arrive
mid-call (P25 Phase 1 LDU2, Phase 2 EncryptionSync). The engine backfills these
onto the bound grant, so by `KindCallEnd` the grant carries the *real* values —
and if `recordEnd` did a naive `SET source_id = ?`, it would finally record the
RID that `recordStart` couldn't. That much is issue #696, and
`TestCallLogBackfillsSourceOnEnd` covers it: start with `source_id = 0`, end with
`source_id = 778899`, and the row must end up at `778899`.

But the reverse must **not** happen. A later end grant can arrive *blanker* than
the start — a compressed-form update that dropped the RID, a preemption whose
grant snapshot lost the encryption fields. A naive `SET` would clobber a good
value with zero. The fix is a single guarded UPDATE:

```go
// internal/storage/calllog.go (shape)
const q = `
UPDATE call_log
   SET ended_at     = ?,
       duration_ms  = ?,
       end_reason   = ?,
       source_id    = COALESCE(NULLIF(?, 0), source_id),
       encrypted    = CASE WHEN ? != 0 THEN 1 ELSE encrypted END,
       algorithm_id = COALESCE(NULLIF(?, 0), algorithm_id),
       key_id       = COALESCE(NULLIF(?, 0), key_id),
       signal_dbfs  = COALESCE(?, signal_dbfs),
       evm_pct      = COALESCE(?, evm_pct),
       snr_db       = COALESCE(?, snr_db)
 WHERE device_serial = ? AND started_at = ?`
```

Read `COALESCE(NULLIF(?, 0), source_id)` inside-out. `NULLIF(?, 0)` turns a bound
value of `0` into SQL NULL; a real RID passes through unchanged. `COALESCE(…,
source_id)` then falls back to the column's current value when the first argument
is NULL. So a genuine mid-call RID overwrites the placeholder, and a zero end
grant leaves whatever the row already had. `algorithm_id` and `key_id` follow the
identical pattern.

Encryption uses a `CASE` instead of `COALESCE` because it is a one-way latch:
`CASE WHEN ? != 0 THEN 1 ELSE encrypted END` only ever flips clear→encrypted. The
protocol defines no mid-call *decryption*, so once a call is known encrypted it
stays that way even if a later grant reports clear.
`TestCallLogEndDoesNotDowngrade` is the guard: a call that starts with RID 5150,
ALG `0x84`, KID 3 survives a completely blank end grant with all three intact.

<figure class="lab-figure">
<svg viewBox="0 0 660 220" width="660" height="220" role="img" aria-label="A which-write-wins table for the never-downgrade rules. For source_id, algorithm_id and key_id, a nonzero end value wins and a zero end value keeps the existing column. For encrypted, an encrypted end latches to encrypted and a clear end keeps whatever was there. For signal_dbfs, evm_pct and snr_db, a measured value wins and a NULL bind keeps the existing column.">
  <text x="70" y="24" text-anchor="middle" fill="var(--fg-muted)" font-size="10">column</text>
  <text x="300" y="24" text-anchor="middle" fill="var(--fg-muted)" font-size="10">end value present</text>
  <text x="520" y="24" text-anchor="middle" fill="var(--fg-muted)" font-size="10">end value blank / NULL</text>
  <line x1="20" y1="32" x2="640" y2="32" stroke="var(--fg-muted)"/>
  <rect x="20" y="42" width="180" height="30" rx="5" fill="none" stroke="currentColor"/>
  <text x="110" y="61" text-anchor="middle" fill="currentColor" font-size="10">source_id · alg · key</text>
  <rect x="212" y="42" width="196" height="30" rx="5" fill="none" stroke="var(--accent)"/>
  <text x="310" y="61" text-anchor="middle" fill="var(--accent)" font-size="9">NULLIF(?,0) → new wins</text>
  <rect x="420" y="42" width="220" height="30" rx="5" fill="none" stroke="currentColor"/>
  <text x="530" y="61" text-anchor="middle" fill="currentColor" font-size="9">COALESCE → keep column</text>
  <rect x="20" y="80" width="180" height="30" rx="5" fill="none" stroke="currentColor"/>
  <text x="110" y="99" text-anchor="middle" fill="currentColor" font-size="10">encrypted</text>
  <rect x="212" y="80" width="196" height="30" rx="5" fill="none" stroke="var(--accent)"/>
  <text x="310" y="99" text-anchor="middle" fill="var(--accent)" font-size="9">CASE → latch to 1</text>
  <rect x="420" y="80" width="220" height="30" rx="5" fill="none" stroke="currentColor"/>
  <text x="530" y="99" text-anchor="middle" fill="currentColor" font-size="9">keep column (no downgrade)</text>
  <rect x="20" y="118" width="180" height="30" rx="5" fill="none" stroke="currentColor"/>
  <text x="110" y="137" text-anchor="middle" fill="currentColor" font-size="10">signal · evm · snr</text>
  <rect x="212" y="118" width="196" height="30" rx="5" fill="none" stroke="var(--accent)"/>
  <text x="310" y="137" text-anchor="middle" fill="var(--accent)" font-size="9">COALESCE(?,…) → new wins</text>
  <rect x="420" y="118" width="220" height="30" rx="5" fill="none" stroke="currentColor"/>
  <text x="530" y="137" text-anchor="middle" fill="currentColor" font-size="9">NULL bind → keep column</text>
  <text x="330" y="180" text-anchor="middle" fill="var(--fg-muted)" font-size="9">Rule: a known value is never replaced by zero or NULL.</text>
  <text x="330" y="198" text-anchor="middle" fill="var(--fg-muted)" font-size="9">The end write can only add information, never remove it.</text>
</svg>
<figcaption>Which write wins. Every column's UPDATE is a one-way ratchet: the end event can enrich the row but can never downgrade a value the start already established.</figcaption>
</figure>

## Unset is not zero

The last three columns — `signal_dbfs`, `evm_pct`, `snr_db` — are the call's
measured quality, and they carry a distinction the integer columns don't: **the
difference between "measured as 0" and "never measured."** Only some ends carry a
measurement. A clean composer teardown stamps the received channel power and, on
a P25 Phase 1 chain, the demod EVM and SNR. A *non-composer* end — a watchdog
timeout, a preemption, a shutdown — carries nothing.

The code models "nothing" as a NULL bind rather than a zero. `recordEnd` reads
the pointer fields on `CallEnd` and only marks the `sql.NullFloat64` valid when
the pointer is non-nil:

```go
// internal/storage/calllog.go (shape)
var sig sql.NullFloat64
if ce.SignalDbFS != nil {
    sig = sql.NullFloat64{Float64: *ce.SignalDbFS, Valid: true}
}
// evm_pct / snr_db bound the same way from ce.EVMPct / ce.SNRDb
```

An invalid `sql.NullFloat64` binds as SQL NULL, so `COALESCE(?, signal_dbfs)`
keeps whatever the column already held — the same never-clobber discipline as the
RID guard, applied to floats. That is why the columns have no `DEFAULT` in the
schema: a fresh call with no reading must read back as `nil`, distinct from a
legitimate `0.0 dBFS`. `TestCallLogPersistsSignalDbFS` and
`TestCallLogPersistsDemodMetrics` assert both halves — a measured call round-trips
its value, an unmeasured call reads back `nil`.

> ⚠ These figures are for *your* records, not cross-decoder comparison across the
> board. `signal_dbfs` is channel power in dBFS, not calibrated RSSI or SNR; only
> the `evm_pct` / `snr_db` pair (populated by P25 Phase 1 chains) is a true demod-
> quality measure. The `CallRow` doc comments spell this out so a history consumer
> doesn't read too much into a raw power figure.

The DSP behind these numbers — how the composer measures EVM and SNR — belongs to
[Voice Coding Part 10]({{ '/blog/deep-dives/voice-coding-10-enhancement-loudness/' | relative_url }}).
Here they are just three more columns the call log persists faithfully.

## The schema and why it's pure Go

The table lives in `internal/storage/sqlite.go`, applied once at `Open` time. The
driver is `modernc.org/sqlite` — a pure-Go SQLite with no CGO. As the package doc
in `internal/storage/storage.go` puts it, that keeps `CGO_ENABLED=0` true across
the whole daemon so it cross-compiles to `linux/arm64` without toolchain
gymnastics — a real concern for a scanner that runs on a Raspberry Pi.

```sql
-- internal/storage/sqlite.go (shape)
CREATE TABLE IF NOT EXISTS call_log (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    system        TEXT    NOT NULL,
    group_id      INTEGER NOT NULL,
    source_id     INTEGER NOT NULL DEFAULT 0,
    -- …encrypted, algorithm_id, key_id, emergency, timeslot…
    device_serial TEXT    NOT NULL,
    started_at    INTEGER NOT NULL,  -- unix nanoseconds
    ended_at      INTEGER,           -- NULL while the call is active
    duration_ms   INTEGER,
    signal_dbfs   REAL,  -- NULL = unmeasured (no DEFAULT)
    evm_pct       REAL,  -- NULL = unmeasured
    snr_db        REAL   -- NULL = unmeasured
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_call_log_active
    ON call_log(device_serial, started_at);
```

That last unique index is the one enforcing idempotent starts, and the plain
indexes on `started_at`, `(system, started_at)`, and `(group_id, started_at)`
serve the history filters below. Because `CREATE TABLE IF NOT EXISTS` never alters
an existing table, a database from an older GopherTrunk keeps its old column set;
`ensureCallLogColumns` inspects `PRAGMA table_info` and `ALTER TABLE`s in any
missing columns (`algorithm_id`, `key_id`, `timeslot`, and the three nullable
quality columns). It is idempotent, so it needs no schema-version gate —
`TestMigrationAddsEncryptionColumns` builds a pre-migration table and confirms an
old row still reads back with sensible defaults after `Open`.

## The read path

Writes are only half the story; the row exists to be read back. `HistoryFilter`
describes a query and `CallRow` is the row DTO:

```go
// internal/storage/calllog.go (shape)
type HistoryFilter struct {
    System    string
    GroupID   uint32    // 0 = no filter
    SourceID  uint32    // 0 = no filter (filters on RID)
    Since     time.Time
    Until     time.Time
    Limit     int
    OnlyEnded bool
}
```

`DB.History` builds a parameterised `WHERE` incrementally — each non-zero filter
appends one `AND` clause and one bound arg — and always orders `started_at DESC`
so results are newest-first. `OnlyEnded` adds `ended_at IS NOT NULL` to hide calls
still in progress; `Limit` caps the result. The scan is where the nullable columns
pay off: `ended_at`, `duration_ms`, and the three quality floats scan into
`sql.Null*` locals and only populate the `CallRow` pointer fields when valid, so a
JSON consumer sees `null`, not a fabricated zero.

```go
// internal/storage/calllog.go (shape)
func (d *DB) History(ctx context.Context, f HistoryFilter) ([]CallRow, error) {
    q := `SELECT … FROM call_log WHERE 1=1`
    // append " AND system = ?", " AND group_id = ?", " AND source_id = ?",
    //        " AND started_at >= ?", " AND started_at < ?", " AND ended_at IS NOT NULL"
    q += " ORDER BY started_at DESC"
    // …scan rows; nullable columns → pointer fields…
}
```

This is the single read helper behind the REST `/api/v1/calls/history` endpoint —
the storage package doc notes there is no gRPC call-log service; history is REST
only. `TestHistoryFilters` exercises every clause, including the `SourceID` filter
that the `/rids` history view uses to pull every call from one radio across
systems — the exact query that only works because the never-downgrade UPDATE got
the RID into the row in the first place.

## Where this goes next

The call log grows one row per call forever, and the recorder grows one WAV per
call forever. Something has to bound both.
[Part 11]({{ '/blog/deep-dives/recording-streaming-11-retention-housekeeping/' | relative_url }})
covers the retention sweeper — the background job that ages out old `call_log`
rows, the decoder-log tables, and (opt-in) the `.wav` / `.raw` files on disk,
each gated by its own configurable cutoff.

## FAQ

**Why key the row on `(device_serial, started_at)` instead of an ID?**
Because the writer needs a *natural* key it can compute at both start and end
without carrying state between events. One physical voice SDR carries one call at
a time, so its serial plus the start timestamp names the call uniquely. That pair
is a unique index, which also makes duplicate `KindCallStart` events idempotent.

**What does `COALESCE(NULLIF(?, 0), col)` actually do?**
`NULLIF(?, 0)` converts a bound `0` to SQL NULL; `COALESCE` then falls back to the
existing column value when the argument is NULL. Net effect: a real mid-call RID
(or ALGID/KID) overwrites the grant-time placeholder, but a zero end grant leaves
the known value untouched. It is a one-way ratchet that only ever adds information.

**Why can a call's RID be missing at start but present at end?**
On P25 Phase 2 compressed grants the radio ID isn't on the control channel; it
surfaces mid-call on the traffic channel. The engine backfills it onto the bound
grant, so `KindCallEnd` carries the real RID. Without the never-downgrade UPDATE
(issue #696) the history row would keep `source_id = 0` and the caller would never
appear in call history.

**Does the call log need CGO or a native SQLite library?**
No. It uses `modernc.org/sqlite`, a pure-Go SQLite driver, so `CGO_ENABLED=0`
holds across the daemon and it cross-compiles to targets like `linux/arm64` with
no C toolchain — which matters when the scanner runs on a Raspberry Pi.

## Series navigation

**Part 10 of 14** · ←
[Part 9: The Call-Complete Seam]({{ '/blog/deep-dives/recording-streaming-09-call-complete-seam/' | relative_url }})
· Next →
[Part 11: Retention & Housekeeping]({{ '/blog/deep-dives/recording-streaming-11-retention-housekeeping/' | relative_url }})
