---
title: "The Conventional Scanner, Part 12: Priority Interleave and Persistent Lockouts — Keyed by Frequency, Never List Index"
description: "How GopherTrunk's conventional scanner samples a priority channel several times per lap (scanner.priority_interleave, pickNextChannel, the cursor visit that resets the clock) and remembers a lockout across a restart in the conv_lockouts table keyed by frequency — plus why talkgroup hold and avoid deliberately do not persist."
category: tutorials
keywords: scanner priority channel, priority_interleave, conventional scanner lockout, lockout survives restart, conv_lockouts table, lockout by frequency, scanner hold avoid, pickNextChannel rotation, uniden style priority scan, gophertrunk conventional scanner
tags: [conventional-scanner, scanner, lockout, priority, storage, tutorial]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "The Conventional Scanner"
series_part: 12
---

*Part 12 of **The Conventional Scanner**, a 14-part operator's tutorial on
GopherTrunk's non-trunked scan list — what you configure, what the code does
with it, what the log prints, and what is verified on air.
[Part 11]({{ '/blog/tutorials/conventional-scanner-11-mdc1200-fleetsync-on-scan-channels/' | relative_url }})
put the MDC1200 and FleetSync decoders behind the dwell. This part is about
the rotation itself — the two front-panel habits every hardware scanner has
that a plain round robin lacks: a priority channel checked several times per
lap, and a lockout memory that survives a power cycle.*

> **TL;DR:** `pickNextChannel` (`internal/scanner/conventional/scanner.go`)
> walks the list skipping `lockedOut` indices; a forced dwell (`DwellOn`)
> beats a lockout for one cycle, and ok=false idles the loop when every
> channel is locked. `scanner.priority_interleave: N` makes the rotation
> visit the next channel with a `priority` after every N ordinary picks
> (`nextPriorityLocked` walks `priCursor`), without moving the main cursor —
> and a cursor visit to a priority channel is itself a sample, so it resets
> `sincePriority` (pinned: `A P C D P A P C D P` in
> `TestConvScannerPriorityInterleave`). Runtime lockouts
> (`POST /api/v1/scanner/conventional/{index}/lockout`, TUI `L`, the web
> Lock button) end a dwelling call with `reason=lockout` and, when
> `storage.path` is set, persist through `storage.ConvLockoutStore` into the
> `conv_lockouts` table **by `frequency_hz`**, never by list index — a
> reordered list re-applies to the same frequency and a vanished one is
> ignored (`TestConvScannerRestoresPersistedLockouts`). Talkgroup hold and
> avoid (`trunking/holdavoid.go`) stay session-only on purpose, and they never
> touch a conventional call at all: `HandleSyntheticCall` bypasses
> `HandleGrant`'s gates.

**Key takeaways**

- **A priority channel is sampled, not favoured.** The interleave inserts
  extra visits between ordinary picks; it does not preempt a dwell, and a
  list with only priority channels degenerates to the plain rotation.
- **The cursor's own visit counts.** If the rotation reaches a priority
  channel by itself, the interleave clock restarts — otherwise the channel
  would be visited twice in a row.
- **Lockouts are a property of a frequency.** List indices shift the moment
  the operator inserts or reorders a channel; `conv_lockouts` keys on
  `frequency_hz` and the scanner re-maps at construction.
- **Hold and avoid are a different axis.** They gate trunked grants in the
  engine and clear on restart like a scanner power cycle; a conventional
  channel's only runtime control is its lockout.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| Rotation | walk ≤ n steps for an unlocked index; forced dwell first | `pickNextChannel`, `forcedDwellIndex` (`scanner.go`) |
| Priority scan | every N ordinary picks, visit the next `Priority > 0` channel | `Options.PriorityInterleave`, `nextPriorityLocked`, `sincePriority`, `priCursor` |
| Config | `scanner.priority_interleave` (≥ 0, 0 = off), `priority` per channel | `config.ScannerConfig`, `validateConvChannel` |
| Runtime lockout | skip the index; end a dwelling call with `EndReasonLockout` | `LockoutChannel`, `UnlockoutChannel`, `POST …/{index}/lockout` |
| Persistence | `conv_lockouts(frequency_hz PRIMARY KEY, label, locked_at)` | `storage.ConvLockoutStore` (`convlockouts.go`), `sqlite.go` |
| Wiring | restore at construction, persist every toggle | `convLockoutPersistence` (`cmd/gophertrunk/conv_lockouts.go`), `LockedOutHz`, `OnLockoutChange` |
| Pins | restore by frequency, interleave order, store round trip | `lockout_persist_test.go`, `TestConvLockoutStoreRoundTrip` |

## In this post

- **The rotation before priority** — the walk, the forced dwell, the all-locked idle.
- **Priority interleave** — `nextPriorityLocked` and the clock the cursor resets.
- **Lockout at runtime** — what one keypress does to a live dwell.
- **Persistence by frequency** — the table, the restore, the three log lines.
- **Why hold and avoid stay session-only** — the engine's gates and the conventional path that skips them.

## The rotation before priority

Every scan pass starts in `pickNextChannel`, called under the scanner's
mutex from `Run`. It returns an index, a copy of the channel, and an `ok`
that is false only when there is nothing to tune — an empty list, or every
channel locked out — in which case `Run` idles in 100 ms ticks until
`AddTemporaryChannel` or an unlock changes that
(`TestConvScannerLockoutAllIdles`). Two things come before the walk. A
forced dwell from `DwellOn` (the TUI's `Enter`, `POST
/api/v1/scanner/conventional/{index}/dwell`) is consumed first and
**beats a lockout for one cycle**: the operator's explicit intent wins, the
flag stays set, and the scanner skips the channel again on the next pass. Then
the cursor is clamped to the list length, because `RemoveTemporaryChannel`
can shrink it.

The walk itself is bounded:

```go
// internal/scanner/conventional/scanner.go (shape) — pickNextChannel
for attempts := 0; attempts < n; attempts++ {
    idx := s.cursor
    s.cursor = (s.cursor + 1) % n
    if s.lockedOut[idx] {
        continue
    }
    if s.opts.PriorityInterleave > 0 {
        if s.channels[idx].Priority > 0 {
            s.sincePriority = 0 // the rotation reached a priority channel on its own
        } else {
            s.sincePriority++
        }
    }
    return idx, s.channels[idx], true
}
return 0, Channel{}, false
```

At most `n` steps, so one full lap with no candidate returns ok=false rather
than spinning. Without `PriorityInterleave` the counter never moves and the
function is the plain round robin
[Part 1]({{ '/blog/tutorials/conventional-scanner-01-what-conventional-means/' | relative_url }})
described.

## Priority interleave

A hardware scanner's priority check samples one channel every few hundred
milliseconds regardless of where the rotation is. `scanner.priority_interleave:
N` is that behaviour: after `N` ordinary picks, `pickNextChannel` asks
`nextPriorityLocked` for the next unlocked channel whose `Priority` is set and
returns it **without moving the main cursor**, then zeroes `sincePriority`.
`nextPriorityLocked` keeps its own `priCursor` so several priority channels
take turns, and it refuses when every unlocked channel is a priority one —
interleaving would then only reorder the plain rotation
(`TestConvScannerPriorityInterleaveOnlyPriority`: two priority channels at
interleave 1 still rotate `P1 P2 P1 P2`).

The subtle rule is in the code above: when the cursor reaches a priority
channel on its own, that visit **is** a sample, so the clock restarts.
`TestConvScannerPriorityInterleave` pins the consequence with four channels
`A P C D`, `P` priority, interleave 2:

```text
A P C D P A P C D P
    ^   ^   ^   ^
    cursor reaches P → clock restarts → C D → interleave visits P →
    A, then the cursor reaches P again → C D → interleave visits P …
```

`P` is never more than two ordinary channels away, and never visited twice
in a row. Lock `P` out and the rotation degenerates to `A C D` — a locked-out
priority channel is skipped by `nextPriorityLocked` too.

<figure class="lab-figure">
<svg viewBox="0 0 680 190" width="680" height="190" role="img" aria-label="A timeline of ten scan picks with priority interleave set to two over four channels A, P, C and D, where P carries a priority. The sequence reads A, P, C, D, P, A, P, C, D, P. Picks made by the main cursor are drawn as plain boxes; the two extra visits to P inserted by the interleave are drawn in the accent colour above the cursor row and labelled interleave. Below the row a counter line shows sincePriority rising to one at A, resetting to zero when the cursor reaches P on its own, rising through C and D to two, and resetting at each interleave visit.">
  <text x="340" y="14" text-anchor="middle" fill="currentColor" font-size="10" font-weight="bold">priority_interleave: 2 · channels A P C D (P has a priority)</text>
  <text x="24" y="70" text-anchor="end" fill="var(--fg-muted)" font-size="8">cursor</text>
  <rect x="40" y="56" width="44" height="22" fill="none" stroke="currentColor"/><text x="62" y="71" text-anchor="middle" fill="currentColor" font-size="9">A</text>
  <rect x="100" y="56" width="44" height="22" fill="none" stroke="currentColor"/><text x="122" y="71" text-anchor="middle" fill="currentColor" font-size="9">P</text>
  <rect x="160" y="56" width="44" height="22" fill="none" stroke="currentColor"/><text x="182" y="71" text-anchor="middle" fill="currentColor" font-size="9">C</text>
  <rect x="220" y="56" width="44" height="22" fill="none" stroke="currentColor"/><text x="242" y="71" text-anchor="middle" fill="currentColor" font-size="9">D</text>
  <rect x="280" y="30" width="44" height="22" fill="none" stroke="var(--accent)" stroke-width="1.5"/><text x="302" y="45" text-anchor="middle" fill="var(--accent)" font-size="9">P</text>
  <text x="302" y="26" text-anchor="middle" fill="var(--accent)" font-size="8">interleave</text>
  <rect x="340" y="56" width="44" height="22" fill="none" stroke="currentColor"/><text x="362" y="71" text-anchor="middle" fill="currentColor" font-size="9">A</text>
  <rect x="400" y="56" width="44" height="22" fill="none" stroke="currentColor"/><text x="422" y="71" text-anchor="middle" fill="currentColor" font-size="9">P</text>
  <rect x="460" y="56" width="44" height="22" fill="none" stroke="currentColor"/><text x="482" y="71" text-anchor="middle" fill="currentColor" font-size="9">C</text>
  <rect x="520" y="56" width="44" height="22" fill="none" stroke="currentColor"/><text x="542" y="71" text-anchor="middle" fill="currentColor" font-size="9">D</text>
  <rect x="580" y="30" width="44" height="22" fill="none" stroke="var(--accent)" stroke-width="1.5"/><text x="602" y="45" text-anchor="middle" fill="var(--accent)" font-size="9">P</text>
  <text x="602" y="26" text-anchor="middle" fill="var(--accent)" font-size="8">interleave</text>
  <line x1="40" y1="110" x2="640" y2="110" stroke="var(--fg-muted)"/>
  <text x="24" y="113" text-anchor="end" fill="var(--fg-muted)" font-size="8">sincePriority</text>
  <text x="62" y="128" text-anchor="middle" fill="currentColor" font-size="9">1</text>
  <text x="122" y="128" text-anchor="middle" fill="var(--accent)" font-size="9">0</text>
  <text x="182" y="128" text-anchor="middle" fill="currentColor" font-size="9">1</text>
  <text x="242" y="128" text-anchor="middle" fill="currentColor" font-size="9">2</text>
  <text x="302" y="128" text-anchor="middle" fill="var(--accent)" font-size="9">0</text>
  <text x="362" y="128" text-anchor="middle" fill="currentColor" font-size="9">1</text>
  <text x="422" y="128" text-anchor="middle" fill="var(--accent)" font-size="9">0</text>
  <text x="482" y="128" text-anchor="middle" fill="currentColor" font-size="9">1</text>
  <text x="542" y="128" text-anchor="middle" fill="currentColor" font-size="9">2</text>
  <text x="602" y="128" text-anchor="middle" fill="var(--accent)" font-size="9">0</text>
  <text x="122" y="150" text-anchor="middle" fill="var(--fg-muted)" font-size="8">cursor reached P: a sample, clock restarts</text>
  <text x="340" y="176" text-anchor="middle" fill="var(--fg-muted)" font-size="9">P is never more than two ordinary channels away and never visited twice in a row (TestConvScannerPriorityInterleave)</text>
</svg>
<figcaption>The interleave inserts visits between ordinary picks and leaves the main cursor where it was; the cursor's own arrival at a priority channel resets the clock.</figcaption>
</figure>

Two configuration facts worth knowing. The config comment calls `priority`
"1..10, 0 = unset", but `validateConvChannel` does not range-check it — the
scanner treats any `Priority > 0` as a priority channel, and the number's
*rank* does not affect the interleave order (`priCursor` walks list order).
And `priority` on a conventional channel does **not** ride on the synthetic
grant: `beginDwell` builds its `trunking.Grant` without a priority, and the
engine's preemption rule (`EffectivePriority`) reads the talkgroup
catalogue's priority for the grant's `GroupID` — so a `talkgroup_id` with a
roster row is how a conventional call gets a priority at the engine, while
the scan-list `priority` only shapes the rotation.

## Lockout at runtime

`LockoutChannel(idx)` sets `lockedOut[idx]` and does one more thing: if the
channel is dwelling right now, it ends the synthetic call immediately with
`trunking.EndReasonLockout`, so "don't listen to this" takes effect within one
IQ chunk instead of waiting for hangtime. The engine logs it as any other
synthetic teardown:

```text
INF synthetic call ended device=00000002 reason=lockout
```

`UnlockoutChannel` deletes the flag and the scanner picks the channel back up
on the next pass. Both are idempotent and return false only for an index out
of range (`TestConvScannerLockoutOutOfRangeReturnsFalse`). The REST
surface is one route per side — `POST
/api/v1/scanner/conventional/{index}/lockout` and `…/unlockout`, answering
`{"ok":true,"index":N,"locked_out":true|false}` (`convLockoutOp` in
`internal/api/handlers_scanner.go`; 404 for a bad index, 503 when the
scanner is not wired) — and two front ends sit on it: the TUI Scanner panel's
`L` key, and the web Scanner panel's **Lock** / **Unlock** buttons, where
Lock asks for confirmation ("Stop scanning … until the lockout is cleared")
because it silences a channel. The snapshot carries the state back as
`locked_out` on each row of `GET /api/v1/scanner`'s `conventional.channels`,
and `TestConvScannerLockoutSkipsChannel` pins that a locked channel's
frequency never reaches the tuner.

## Persistence by frequency

Until the lockout store landed, that map lived only in memory, and every
restart forgot it — a hardware scanner's lockout memory is what the
operator expects. The store is `storage.ConvLockoutStore`
(`internal/storage/convlockouts.go`) over a three-column table:

```sql
-- internal/storage/sqlite.go
CREATE TABLE IF NOT EXISTS conv_lockouts (
    frequency_hz INTEGER PRIMARY KEY,
    label        TEXT    NOT NULL DEFAULT '',
    locked_at    INTEGER NOT NULL              -- unix nanoseconds
);
```

**The key is the frequency, never the list index.** An index shifts the
moment the operator reorders, inserts or removes a channel in the config
between restarts; a frequency re-applies to whichever channel carries it,
and a frequency no longer in the list is simply ignored — kept, so it
re-applies if the channel returns. `Set` is an upsert that refreshes the
label, `Clear` deletes one frequency (clearing an unknown one is not an
error), `List` returns oldest first, and `Set(0, …)` is refused;
`TestConvLockoutStoreRoundTrip` pins all four.

The wiring is `convLockoutPersistence` in `cmd/gophertrunk/conv_lockouts.go`,
called from `daemon.go` when the scanner is built. It `List`s the store once
and hands the frequencies to `Options.LockedOutHz`; `New` maps them onto
indices before the first pick, so a locked channel is never dwelt on even
once after a restart. It also returns the `Options.OnLockoutChange` callback,
which `LockoutChannel` / `UnlockoutChannel` call off the scanner's lock with
the affected `Channel` — `Set(ch.FrequencyHz, ch.Label)` or
`Clear(ch.FrequencyHz)`. Three log lines cover the store:

```text
INF conv: restored persisted lockouts count=2
WRN conv: persisted lockouts not restored err=…
WRN conv: lockout not persisted freq_hz=462562500 locked=true err=…
```

The two WARNs share a rule: **a store error never reaches the scanner.** The
lockout still applies in memory and the failure is logged, because refusing
an operator's lockout over a database hiccup is the worse outcome. A nil
store — no `storage.path` — yields nil/nil and the scanner keeps its
runtime-only behaviour, which is why `config.example.yaml` notes that
lockouts persist "when storage.path is set".
`TestConvScannerRestoresPersistedLockouts` pins the restart contract end to
end: channel B at 200 MHz restores as locked and is never picked, a stale
999 MHz entry is ignored, restoring does not re-persist anything, and a
subsequent unlock of B and lock of C reach the callback as exactly
`{200 MHz, false}, {300 MHz, true}`.

## Why hold and avoid stay session-only

The engine has its own pair of front-panel controls,
`trunking/holdavoid.go`: **Hold** pins the engine to one talkgroup (every
other grant is dropped until `DELETE /api/v1/scanner/hold`; emergency grants
still pass, and a held talkgroup bypasses the scan-list gate), and **Avoid**
is a lockout with an expiry (`POST /api/v1/talkgroups/{id}/avoid`, default
30 minutes). In `HandleGrant` they sit after the catalogue `Lockout` check
and before the `scan_mode: list` gate, and `grant dropped by hold` / `by
avoid` is logged at DEBUG. Nothing is written to the catalogue, so a restart
clears both — deliberately, like a scanner power cycle. The
[competitive feature assessment]({{ '/competitive-feature-assessment.html' | relative_url }})
lists "hold / avoid surviving a restart" as a known, intentional gap next to
the conventional lockouts that do persist.

There is a structural reason the two mechanisms do not overlap. A
conventional dwell reaches the engine through `HandleSyntheticCall`, which
publishes `KindCallStart` directly and **never passes through `HandleGrant`**
— no catalogue lockout, no hold, no avoid, no scan-list gate applies to it.
The conventional channel's runtime control is its own lockout, keyed by
frequency, enforced before the tuner is touched; the engine's hold and avoid
shape which *trunked* grants get a voice SDR. Two axes, two stores, one of
them persistent on purpose.

### How the rotation shaped the Go code

- **Lockouts are an index set inside the scanner, a frequency set outside
  it.** `lockedOut map[int]bool` is what `pickNextChannel` reads;
  `LockedOutHz []uint32` / `OnLockoutChange` are the only persistence
  surface, so the scanner never imports storage.
- **The callback runs off the lock.** `LockoutChannel` releases `mu` before
  calling `OnLockoutChange`, so a slow database write cannot stall the scan
  loop.
- **Index shifts are handled where they happen.** `RemoveTemporaryChannel`
  rebuilds `lockedOut` through `shiftIndexSet` so a VFO removal does not
  lock out the wrong row.
- **The interleave is a counter, not a timer.** `sincePriority` counts
  picks, so a long dwell on an ordinary channel does not pile up overdue
  priority visits.

## Where this goes next

Everything in this series has printed something — a startup line, a dwell,
a WARN. [Part 13]({{ '/blog/tutorials/conventional-scanner-13-reading-the-scanner-log/' | relative_url }})
collects the conventional scanner's log lines in one place in the Field
Notebook's format — line, meaning, healthy, worry — alongside the
`/api/v1/scanner` snapshot fields, and closes with the honest list of what
is still open.

## FAQ

**How do I make the conventional scanner check a priority channel more often?**
Set `scanner.priority_interleave: N` and give the channel a `priority`. After
every N ordinary picks `pickNextChannel` visits the next priority channel
(`nextPriorityLocked`) without moving the main cursor; with four channels and
interleave 2 the rotation is `A P C D P A P C D P`. A list of only priority
channels stays a plain round robin.

**Does a conventional lockout survive a daemon restart?**
Yes, when `storage.path` is set. `storage.ConvLockoutStore` writes every
toggle to the `conv_lockouts` table keyed by `frequency_hz`, and the scanner
restores the set at construction (`Options.LockedOutHz`) before its first
pick. Reordering the scan list cannot lock out the wrong channel; a frequency
no longer in the list is ignored but kept.

**What happens if I lock out the channel that is currently playing?**
`LockoutChannel` ends the synthetic call at once with `reason=lockout`
(`trunking.EndReasonLockout`) instead of waiting for hangtime, flags the
index, and the rotation skips it until `UnlockoutChannel`. A forced dwell on
a locked-out row (`DwellOn`) still works for one cycle; the flag stays set.

**Why don't talkgroup hold and avoid persist like lockouts do?**
By design: `trunking/holdavoid.go` keeps them in memory and a restart clears
them like a scanner power cycle. They gate trunked grants inside
`HandleGrant`; a conventional call enters through `HandleSyntheticCall`,
which bypasses those gates entirely, so the conventional lockout is the only
control that applies to a scan-list channel.

**Does `priority` on a conventional channel preempt other calls?**
Not by itself. The scan-list `priority` only drives the interleave;
`beginDwell`'s synthetic grant carries no priority, and the engine's
`EffectivePriority` reads the talkgroup catalogue for the grant's `GroupID`.
Give the channel a `talkgroup_id` with a roster row if you want it ranked at
the engine.

## Series navigation

**Part 12 of 14** · ←
[Part 11: MDC1200 and FleetSync on Scan-List Channels — The Data Front End Behind the Dwell]({{ '/blog/tutorials/conventional-scanner-11-mdc1200-fleetsync-on-scan-channels/' | relative_url }})
· Next →
[Part 13: Reading the Scanner's Log — Lines, Symptoms, and What Is Still Open]({{ '/blog/tutorials/conventional-scanner-13-reading-the-scanner-log/' | relative_url }})
