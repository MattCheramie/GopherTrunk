---
title: "Trunking Engine, Part 6: Talkgroups, Aliases & Scan Modes"
description: How GopherTrunk's talkgroup database gives numeric IDs names and policy, how hold and lockout gate a call, and how the engine flips between scan-all and scan-list at runtime from the cockpit without a restart.
category: deep-dives
keywords: talkgroup database, talkgroup alias lookup, scan mode list all, talkgroup lockout, runtime scan mode, trunk recorder csv, gophertrunk talkgroups, scan list gate, talkgroupdb
tags: [trunking, go, talkgroups, scanning, configuration, software-design]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "Trunking Engine"
series_part: 6
---

*Part 6 of **Trunking Engine**. [Part 5]({{ '/blog/deep-dives/trunking-engine-05-priority-preemption/' | relative_url }})
showed priority deciding which call keeps a radio — a policy that reads the
talkgroup's `Priority` and `Lockout` fields. This post opens the database those
fields live in, and the scan modes that decide which grants are even eligible for
a radio in the first place.*

> **TL;DR:** `TalkgroupDB` is a thread-safe map from a 32-bit talkgroup ID to a
> `TalkGroup` record — alias, priority, lockout, and per-talkgroup gates for scan,
> record, stream, and mute. It loads from Trunk-Recorder-style CSV or JSON, with
> every boolean gate defaulting to the backwards-compatible value. Two scan modes
> govern dispatch: `ScanModeAll` follows every non-locked-out grant (the legacy
> default), `ScanModeList` follows only talkgroups flagged `Scan`. The mode is
> read under `modeMu` so the cockpit can flip it live — `SetScanMode` — without
> restarting the daemon.

**Key takeaways**

- A `TalkGroup` is the numeric ID plus operator-meaningful metadata and four
  independent boolean gates: `Scan`, `Record`, `Stream`, `Mute`.
- Every gate **defaults to the legacy behavior** so a plain community CSV keeps
  working — `Scan`/`Record`/`Stream` default true, `Mute`/`Lockout` false.
- `Lookup` is an `RWMutex`-guarded map read; the loaders sanitize imported text
  and skip malformed rows rather than failing the whole file.
- Scan mode is a runtime knob: `ScanModeList` enforces the scan list,
  `ScanModeAll` ignores it, and `SetScanMode` flips between them under a lock
  without a restart.

## Cheat sheet

| Field / mode | Default | Effect |
|---|---|---|
| `TalkGroup.AlphaTag` | `""` | operator-facing name for the numeric ID |
| `TalkGroup.Priority` | `0` (unset) | preemption rank (Part 5) |
| `TalkGroup.Lockout` | `false` | grant dropped before dispatch (Emergency bypasses) |
| `TalkGroup.Scan` | `true` | participates in `ScanModeList` |
| `TalkGroup.Record` / `Stream` / `Mute` | `true`/`true`/`false` | per-TG recorder, upload, live-audio gates |
| `ScanModeAll` | **default** | follow every non-locked-out grant |
| `ScanModeList` | — | follow only `Scan==true` talkgroups (+ Emergency) |
| `SetScanMode(m)` | — | flip mode at runtime under `modeMu` |

## In this post

- **The `TalkGroup` record** — metadata plus four independent gates.
- **The database** — `Lookup`, the `RWMutex`, and forgiving loaders.
- **Lockout and the scan-list gate** — where a grant can be dropped.
- **Scan modes at runtime** — `modeMu`, `SetScanMode`, and the live cockpit.

## The `TalkGroup` record

A talkgroup is a numeric ID on the wire and nothing else — `32181` means nothing
to an operator. The `TalkGroup` record is what attaches meaning and policy to it:

```go
// internal/trunking/talkgroup.go (shape)
type TalkGroup struct {
    ID          uint32 // the on-air talkgroup number
    AlphaTag    string // "PD Dispatch 1" — the operator-facing name
    Description string
    Tag         string // department / category
    Group       string // top-level group
    Priority    int    // 1 = highest, 10 = lowest, 0 = unset
    Lockout     bool   // drop grants for this TG (Emergency bypasses)
    Scan        bool   // participate in ScanModeList
    Stream      bool   // upload completed calls to broadcast aggregators
    Record      bool   // write calls to disk
    Mute        bool   // suppress from the live audio player
    Icon        string // per-TG glyph for the UIs
}
```

The metadata half (`AlphaTag`, `Description`, `Tag`, `Group`, `Icon`) is what the
web UI and TUI render so a call shows "PD Dispatch 1" instead of `32181`. The
policy half is four *independent* boolean gates, and the independence is the
point:

- **`Scan`** — does this talkgroup participate when the engine runs in
  `ScanModeList`? (Covered below.)
- **`Record`** — write the call's WAV to disk? A talkgroup can be followed and
  played live but never recorded.
- **`Stream`** — upload completed calls to the outbound aggregators
  (Broadcastify Calls, RdioScanner, OpenMHz)? A sensitive talkgroup can record
  locally but stay off every public feed.
- **`Mute`** — suppress from the host's speakers while still following,
  recording, and streaming.

Because they're independent, an operator composes exactly the behavior they want:
record-but-don't-stream for a sensitive channel, follow-but-mute for a chatty
one, and so on. `Priority` and `Lockout` are the two the rest of the engine reads
— priority in [Part 5]({{ '/blog/deep-dives/trunking-engine-05-priority-preemption/' | relative_url }}),
lockout below.

## The database: `Lookup` and forgiving loaders

`TalkgroupDB` wraps a map behind an `RWMutex`:

```go
// internal/trunking/talkgroup.go
type TalkgroupDB struct {
    mu  sync.RWMutex
    tgs map[uint32]*TalkGroup
}

func (d *TalkgroupDB) Lookup(id uint32) *TalkGroup {
    d.mu.RLock()
    defer d.mu.RUnlock()
    return d.tgs[id] // nil if unknown
}
```

`Lookup` returns `nil` for an unknown ID — a first-class state the engine handles
by *discovering* the talkgroup from the air (cataloguing it so the UI fills in
from the wire, the Trunk-Recorder behavior operators expect). The `RWMutex` lets
many concurrent lookups run while the API mutates a record: `UpdateFields(id, fn)`
applies a closure under the write lock so the cockpit can change `Priority` or
`Lockout` without exposing the raw pointer, and `Add`/`Delete` manage records.

The loaders are deliberately forgiving. `LoadCSV` reads a Trunk-Recorder /
RadioReference CSV — the only required column is a numeric `Decimal`/`DEC` — and
matches optional columns case-insensitively by header. An empty file is a
legitimate "no talkgroups" state, not an error; a malformed row is skipped, not
fatal; and every text column is run through a sanitizer that strips imported
mojibake. Critically, each gate **defaults to the legacy value** so a bare CSV
with no `Scan`/`Stream`/`Record` column keeps the old "follow, record, and stream
everything" behavior:

```go
// internal/trunking/talkgroup.go — per-row defaults
tg := &TalkGroup{ID: uint32(idVal), Scan: true, Stream: true, Record: true}
// …then only an explicit "no"/"false"/"0"/"n" flips a gate off.
```

`LoadJSON` mirrors this with `*bool` pointers for the gates, so a missing key
resolves to the same defaults while an explicit `"scan": false` opts out. The
same conventions are shared with the RID database in
[Part 7]({{ '/blog/deep-dives/trunking-engine-07-source-rid-recovery/' | relative_url }}) —
the per-radio analogue of this file.

<figure class="lab-figure">
<svg viewBox="0 0 680 150" width="680" height="150" role="img" aria-label="A grant's GroupID looks up the TalkgroupDB, returning a record with alias and four gate flags, or nil which triggers talkgroup discovery">
  <rect x="12" y="54" width="120" height="42" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="72" y="72" text-anchor="middle" fill="var(--accent)" font-size="11">grant.GroupID</text>
  <text x="72" y="87" text-anchor="middle" fill="var(--fg-muted)" font-size="10">32181</text>
  <line x1="132" y1="75" x2="182" y2="75" stroke="currentColor"/>
  <polygon points="182,71 192,75 182,79" fill="currentColor"/>
  <rect x="194" y="50" width="140" height="50" rx="6" fill="none" stroke="currentColor"/>
  <text x="264" y="70" text-anchor="middle" fill="currentColor" font-size="11">TalkgroupDB.Lookup</text>
  <text x="264" y="86" text-anchor="middle" fill="var(--fg-muted)" font-size="9">RWMutex map read</text>
  <line x1="334" y1="66" x2="392" y2="40" stroke="currentColor"/>
  <polygon points="392,44 401,38 391,36" fill="currentColor"/>
  <line x1="334" y1="86" x2="392" y2="112" stroke="var(--fg-muted)"/>
  <polygon points="392,108 401,114 391,116" fill="var(--fg-muted)"/>
  <rect x="404" y="18" width="264" height="46" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="536" y="36" text-anchor="middle" fill="var(--accent)" font-size="10">record: "PD Dispatch 1"</text>
  <text x="536" y="52" text-anchor="middle" fill="var(--fg-muted)" font-size="9">prio · Lockout · Scan · Record · Stream · Mute</text>
  <rect x="404" y="94" width="264" height="40" rx="6" fill="none" stroke="var(--fg-muted)"/>
  <text x="536" y="111" text-anchor="middle" fill="var(--fg-muted)" font-size="10">nil → discover talkgroup</text>
  <text x="536" y="125" text-anchor="middle" fill="var(--fg-muted)" font-size="9">catalogue from the air</text>
</svg>
<figcaption>A lookup either resolves the numeric ID to a named, policy-bearing record or returns nil, which the engine turns into a freshly-discovered talkgroup.</figcaption>
</figure>

## Lockout and the scan-list gate

A grant can be dropped in two places before it ever reaches allocation, both
early in `HandleGrant`.

**Lockout** is unconditional (except for emergencies):

```go
// internal/trunking/engine.go (shape)
if tg != nil && tg.Lockout && !g.Emergency {
    e.log.Info("grant locked out", "grant", g.String(), "tg", tg.AlphaTag)
    return
}
```

A locked-out talkgroup is one the operator never wants — its grants are dropped
before priority, before allocation, before anything. `Emergency` is the one
override, the same exception lockout and the scan gate both honor.

**The scan-list gate** is conditional on the scan mode:

```go
// internal/trunking/engine.go (shape)
if e.ScanMode() == ScanModeList && !g.Emergency {
    scanned := tg != nil && tg.Scan
    for _, m := range g.PatchedGroups { // a patch passes if any member is scanned
        if mt := e.talkgroups.Lookup(m); mt != nil && mt.Scan {
            scanned = true
            break
        }
    }
    if !scanned {
        e.log.Debug("grant not in scan list", "grant", g.String())
        return
    }
}
```

In `ScanModeAll` this block is skipped entirely. In `ScanModeList` a grant passes
only if its talkgroup is flagged `Scan` — with two carve-outs that mirror the
policy elsewhere: `Emergency` always passes, and a patched super-group
([Part 9]({{ '/blog/deep-dives/trunking-engine-09-patches-supergroups/' | relative_url }}))
passes if the super-group *or any member* is scanned. An unknown talkgroup ID in
`ScanModeList` is dropped — there's no way to know an uncatalogued TG is
scannable.

## Scan modes at runtime

The two modes are a tiny enum with a safe parser:

```go
// internal/trunking/scanmode.go
type ScanMode uint8
const (
    ScanModeAll  ScanMode = iota // default: follow every non-locked-out grant
    ScanModeList                 // follow only Scan==true talkgroups (+ Emergency)
)
// ParseScanMode maps "" and any typo to ScanModeAll so a config mistake
// never silently silences the daemon.
```

`ScanModeAll` is the default for backwards compatibility — pre-scanner configs
see no change. `ScanModeList` turns the `Scan` flags into an active allow-list.
The safe-default parser matters: an empty or misspelled config value resolves to
`all`, so a typo can't accidentally make the daemon follow *nothing*.

The interesting part is that the mode is a *live* knob, not a boot-time one. It's
stored on the engine behind a dedicated `RWMutex` so the API cockpit can flip it
while the engine's `select` loop is running:

```go
// internal/trunking/engine.go
func (e *Engine) ScanMode() ScanMode {          // read path (HandleGrant)
    e.modeMu.RLock(); defer e.modeMu.RUnlock()
    return e.scanMode
}
func (e *Engine) SetScanMode(m ScanMode) ScanMode { // write path (cockpit)
    e.modeMu.Lock(); defer e.modeMu.Unlock()
    prev := e.scanMode
    e.scanMode = m
    return prev
}
```

`HandleGrant` takes a snapshot with `ScanMode()` under the read lock — cheap, and
it never blocks the bus loop. The cockpit calls `SetScanMode` under the write
lock and gets the previous mode back so it can log/audit the change. This is a
narrow, deliberate exception to the single-writer rule from
[Part 1]({{ '/blog/deep-dives/trunking-engine-01-grant-to-call/' | relative_url }}):
the engine loop is still the only mutator of *call state*, but the *scan mode* is
a shared configuration value that an operator changes out-of-band, so it gets its
own lock rather than being funneled through the event bus. An operator toggling
`scan_mode` from `all` to `list` in the TUI sees the very next grant filtered,
with no restart and no dropped calls.

<figure class="lab-figure">
<svg viewBox="0 0 680 160" width="680" height="160" role="img" aria-label="The API cockpit calls SetScanMode under the write side of modeMu while the engine select loop reads the mode under the read side in HandleGrant, so a live mode flip takes effect on the next grant">
  <rect x="14" y="24" width="150" height="42" rx="6" fill="none" stroke="var(--fg-muted)"/>
  <text x="89" y="42" text-anchor="middle" fill="var(--fg-muted)" font-size="11">API cockpit / TUI</text>
  <text x="89" y="57" text-anchor="middle" fill="var(--fg-muted)" font-size="9">operator flips scan_mode</text>
  <line x1="164" y1="45" x2="250" y2="60" stroke="var(--fg-muted)"/>
  <polygon points="250,56 260,61 249,64" fill="var(--fg-muted)"/>
  <rect x="262" y="52" width="156" height="52" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="340" y="72" text-anchor="middle" fill="var(--accent)" font-size="11">modeMu</text>
  <text x="340" y="88" text-anchor="middle" fill="var(--fg-muted)" font-size="9">Write: SetScanMode</text>
  <text x="340" y="99" text-anchor="middle" fill="var(--fg-muted)" font-size="9">Read: ScanMode() snapshot</text>
  <rect x="14" y="94" width="150" height="42" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="89" y="112" text-anchor="middle" fill="var(--accent)" font-size="11">engine select loop</text>
  <text x="89" y="127" text-anchor="middle" fill="var(--fg-muted)" font-size="9">HandleGrant reads mode</text>
  <line x1="164" y1="115" x2="255" y2="96" stroke="var(--accent)"/>
  <polygon points="255,100 265,95 254,92" fill="var(--accent)"/>
  <line x1="418" y1="78" x2="520" y2="78" stroke="currentColor"/>
  <polygon points="520,74 530,78 520,82" fill="currentColor"/>
  <rect x="532" y="54" width="136" height="48" rx="6" fill="none" stroke="currentColor"/>
  <text x="600" y="74" text-anchor="middle" fill="currentColor" font-size="10">next grant filtered</text>
  <text x="600" y="90" text-anchor="middle" fill="var(--fg-muted)" font-size="9">no restart</text>
</svg>
<figcaption>Scan mode is shared config, not call state, so it gets its own RWMutex: the cockpit writes it, HandleGrant snapshots it, and the flip lands on the next grant.</figcaption>
</figure>

## Where this goes next

The talkgroup database names the *destination* of a call; the next post names its
*source*. [Part 7]({{ '/blog/deep-dives/trunking-engine-07-source-rid-recovery/' | relative_url }})
opens the RID database — the per-radio analogue of this file, sharing the same
loaders and conventions — and the marquee current-work fix behind it: recovering
the source radio ID for the ~85% of calls whose grant arrived without one.
[Part 9]({{ '/blog/deep-dives/trunking-engine-09-patches-supergroups/' | relative_url }})
returns to the `PatchedGroups` carve-out in the scan gate.

## FAQ

**What's the difference between lockout and scan-off?**
Lockout drops a talkgroup's grants unconditionally (except emergencies),
regardless of scan mode — it's "never follow this." Scan-off (`Scan == false`)
only matters in `ScanModeList`, where it excludes the talkgroup from the active
allow-list; in `ScanModeAll` it's ignored. Lockout is a blocklist; the scan flag
drives an allow-list.

**What happens when a grant arrives for an unknown talkgroup?**
In `ScanModeAll` the engine discovers it — catalogues a new record so the UI
fills in from the air — and follows the call. In `ScanModeList` an unknown ID is
dropped, because there's no way to know an uncatalogued talkgroup is scannable.

**Can I change scan mode without restarting the daemon?**
Yes. `SetScanMode` swaps the mode under `modeMu` (an `RWMutex`), and `HandleGrant`
snapshots it under the read lock on every grant. The cockpit calls it live, and
the next grant is filtered under the new mode — no restart, no dropped calls.

**Why do the boolean gates default to true?**
For backwards compatibility. A plain community CSV with no `Scan`/`Stream`/
`Record` columns should behave exactly as it did before those gates existed —
follow, record, and stream every call. Only an explicit `no`/`false` in the file
turns a gate off.

**Do `Record`, `Stream`, and `Mute` affect whether a call is followed?**
No — those gate *what happens to the audio* after the call is followed. A call
can be followed (a radio bound, the call decoded) and still not recorded, not
streamed, or not played live. Only `Scan` (in list mode), `Lockout`, and priority
affect whether a radio is allocated at all.

## Series navigation

**Part 6 of 12** · ←
[Part 5: Priority & Preemption]({{ '/blog/deep-dives/trunking-engine-05-priority-preemption/' | relative_url }})
· Next →
[Part 7: Recovering the Source Radio ID]({{ '/blog/deep-dives/trunking-engine-07-source-rid-recovery/' | relative_url }})
