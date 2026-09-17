---
title: "The Field Notebook, Part 5: The DMR Activity Line"
description: "How to read GopherTrunk's widebandt2 channel decode activity line for conventional DMR — dibits, sync_hits, fec_pass, beacons, late_entries, rekeys, csbk_crc_fail and deaf_heals — what a healthy IPSC tap prints, what the deaf-tap heal WARN's receiver internals mean, and which DMR deep dive to open when a counter goes wrong."
category: tutorials
keywords: gophertrunk debug log dmr, dmr decode activity line, sync_hits zero dmr, deaf tap heal warn, dmr idle beacon log, late entry counter dmr, ipsc repeater log reading, widebandt2 log lines, gophertrunk dmr activity line
tags: [field-notebook, logs, dmr, ipsc, widebandt2, diagnostics, tutorial]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "The Field Notebook"
series_part: 5
---

*Part 5 of **The Field Notebook**, a 14-part operator's tutorial that reads
GopherTrunk's debug.log one line family at a time — what each field measures,
what a healthy rig prints, which number is a noise meter and which one means
traffic, and which deep dive to open when a line goes wrong.
[Part 4]({{ '/blog/tutorials/field-notebook-04-carrier-offset-warn/' | relative_url }})
read the TETRA carrier-offset WARN. This part switches protocols: the
**conventional DMR activity line** a wideband tap prints for a repeater, the
`site alive` lines around it, and the deaf-tap heal WARN whose receiver
internals are the instrument for the one IPSC defect still open.*

> **TL;DR:** With `log.level: debug`, every conventional DMR channel on a
> `role: wideband` device prints `widebandt2: channel decode activity` once
> per 30 s parked interval, or at once when its activity class changes. The
> fields are per-window **deltas** of `tier2.Counters`
> (`internal/radio/dmr/tier2/conventional.go`): `dibits` proves the receiver
> is emitting (4800/s), `sync_hits` that bursts are found, `fec_pass` that
> Voice LC Headers decode, `beacons` that the ETSI Idle pattern keeps the
> site alive, `late_entries`/`rekeys` that the recovery paths work,
> `csbk_crc_fail` that the unpinned proprietary train is on the air.
> `deaf_heals` is lifetime: `healDeafTier2`
> (`internal/scanner/widebandt2/engine.go`) resets a receiver that stopped
> syncing at the power it last synced at — the channel's **own** level, never
> an absolute dBFS — and its WARN carries `mm_mu`, `mm_sps`, `agc_level` and
> `coarse_offset_hz`; on the 15 Sep log the last read ≈−20100 and named a
> false coarse-carrier engage the heal now rejects.

**Key takeaways**

- **`dibits` and `sync_hits` are two different silences.** 144 000 dibits in
  30 s with zero sync words is a deaf receiver; zero dibits is one that is
  not running. The 12 Sep log could not tell them apart — hence `dibits`.
- **`beacons` is decode evidence.** A keyed idle repeater sends the ETSI Idle
  burst ~33 times a second; each BPTC-clean copy of the fixed pattern counts,
  locks the site and mutes the no-sync hint.
- **`late_entries` and `rekeys` moving is normal.** They count transmissions
  recovered without a header and re-keys inside hangtime. Climbing
  `late_entries` beside `fec_pass≈0` says the *headers* are lost.
- **The heal WARN is a throttled instrument.** One WARN per channel per
  10 min, DEBUG between, `deaf_heals` counting every reset. A
  `coarse_offset_hz` the channel never synced under is a neighbour.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| The activity line | per-window counter deltas, DEBUG, parked at 30 s | `internal/scanner/widebandt2/engine.go` (`maybeLogDiagnostics`) |
| The counters | lock-free monotonic snapshot per channel | `internal/radio/dmr/tier2/conventional.go` (`Counters`) |
| Site-alive lines | rate-limited INFO on Idle / CSBK beacons | `conventional.go` (`handleIdle`, `handleCSBK`) |
| Deaf-tap heal | 3 windows no sync within 6 dB of last decode level ⇒ reset | `engine.go` (`healDeafTier2`) |
| False-engage reject | an offset never synced under is ruled out for good | `dmrrx.Receiver.RejectCoarseCarrierOffset` |
| CSBK train instrument | parked DEBUG with `csbko`/`fid`/`info_hex`, 10 s | `conventional.go` (`logCSBKFailure`) |
| The background | two calls, the beacon, the heal | [DMR End to End 6]({{ '/blog/deep-dives/dmr-end-to-end-06-conventional-ipsc-two-calls/' | relative_url }}), [7]({{ '/blog/deep-dives/dmr-end-to-end-07-idle-beacon-csbk/' | relative_url }}), [10]({{ '/blog/deep-dives/dmr-end-to-end-10-wideband-bin-edge-deaf-heal/' | relative_url }}) |

## In this post

- **What this line is telling you** — a four-way story in eleven counters.
- **What healthy looks like** — an idle beacon train, then a keyup.
- **Field by field** — measures, healthy, worry-when.
- **When it doesn't look like that** — the deaf tap and the −20 kHz engage,
  then symptom → cause → read.

## What this line is telling you

Between calls a conventional DMR repeater drops carrier or fills both
timeslots with Idle bursts. The `Counters` doc comment frames the reading as
a four-way story: `SyncHits == 0`
means no DMR sync at all (no signal, mistune, wrong offset, or an inversion
the polarity pass cannot recover); sync with `FECPass == 0` and
`FECFail > 0` means bursts found but every Voice LC Header failing FEC;
`FECPass > 0` means genuine headers. `dibits` sits in front of all three
because a field log could not distinguish "the receiver emits nothing" from
"the receiver emits junk that never syncs".

```go
// internal/scanner/widebandt2/engine.go (shape) — the parked activity line
e.log.Debug("widebandt2: channel decode activity",
    "freq_hz", ec.freqHz, "system", ec.sysName,
    "dibits", c.Dibits-ec.lastLogCnt.Dibits,        // receiver output: 4800/s live
    "sync_hits", c.SyncHits-ec.lastLogCnt.SyncHits, // burst-sync matches
    "bursts", c.Bursts-ec.lastLogCnt.Bursts,        // slot-type-parsed data bursts
    "fec_pass", c.FECPass-ec.lastLogCnt.FECPass,    // Voice LC Header BPTC+RS valid
    "fec_fail", c.FECFail-ec.lastLogCnt.FECFail,
    "beacons", c.Beacons-ec.lastLogCnt.Beacons,     // Idle pattern / CRC-valid CSBK
    "late_entries", c.LateEntries-ec.lastLogCnt.LateEntries,
    "csbk_crc_fail", c.CSBKCRCFail-ec.lastLogCnt.CSBKCRCFail,
    "rekeys", c.Rekeys-ec.lastLogCnt.Rekeys,
    "locks_total", c.Locks,      // lifetime
    "deaf_heals", ec.deafHeals)  // lifetime
```

Every field but the last two is a **delta since the previous emitted line**,
so nothing is lost while parked. `activityClass` folds each window into idle
(no sync), syncing (sync, no FEC pass or beacon) or decoding (an FEC pass or
a beacon); a class change logs immediately, so a keyup on a quiet channel
produces a short line the moment its first header decodes.

## What healthy looks like

A camped IPSC repeater, idle, on a wideband X310 tap — beacons by the
hundred, nothing else moving:

```
INF dmr/tier2 cc locked freq=442387500 cc=12 system=Fire2
INF dmr/tier2 site alive (idle beacon) freq=442387500 cc=12 system=Fire2
DBG widebandt2: channel decode activity freq_hz=442387500 system=Fire2 dibits=144000 sync_hits=612 bursts=608 fec_pass=0 fec_fail=0 beacons=598 late_entries=0 csbk_crc_fail=0 rekeys=0 locks_total=1 deaf_heals=0
```

`dibits=144000` is exactly 30 s at 4800/s. `sync_hits ≈ bursts ≈ beacons`:
nearly every burst carried the Idle block, and `fec_pass=0` is *correct* —
an idle repeater sends no Voice LC Headers. The `site
alive` INFO repeats at most every 30 s (`beaconLogInterval`) while the
counter records every beacon. Then someone keys up:

```
DBG widebandt2: channel decode activity freq_hz=442387500 system=Fire2 dibits=24000 sync_hits=166 bursts=40 fec_pass=10 fec_fail=0 beacons=0 late_entries=0 csbk_crc_fail=0 rekeys=0 locks_total=1 deaf_heals=0
```

`fec_pass=10` is one keyup: this radio repeats its Voice LC Header ten times
over 0.6 s, which is why `headerRekeyDibits` (1200 dibits, 0.25 s) is
measured from the *last* copy. `bursts=40` against `sync_hits=166` is right
too: only data-sync bursts are slot-type parsed, so voice superframes count
as syncs but not as bursts.

<figure class="lab-figure">
<svg viewBox="0 0 680 250" width="680" height="250" role="img" aria-label="Timeline of one wideband DMR channel. The dibits row stays at one hundred forty-four thousand per window throughout; the sync hits row runs high during an idle beacon train, drops to zero for a three-minute stretch, then recovers; the power row stays flat near minus fifty-one dBFS. Below, three consecutive windows at the channel's own decoding level trigger the deaf heal, which resets the receiver and logs the coarse offset, AGC level and timing-loop state.">
  <line x1="40" y1="60" x2="660" y2="60" stroke="var(--fg-muted)"/>
  <text x="8" y="64" fill="var(--fg-muted)" font-size="9">dibits</text>
  <text x="640" y="52" fill="currentColor" font-size="8" text-anchor="end">144000 every window — receiver running</text>
  <line x1="40" y1="110" x2="660" y2="110" stroke="var(--fg-muted)" stroke-dasharray="2 3"/>
  <text x="8" y="114" fill="var(--fg-muted)" font-size="9">sync_hits</text>
  <polyline points="40,80 120,78 200,82 260,84 262,110 460,110 462,80 540,78 660,82" fill="none" stroke="currentColor"/>
  <text x="120" y="74" fill="currentColor" font-size="8">~600/window: idle beacon train</text>
  <text x="300" y="104" fill="var(--accent)" font-size="8">sync_hits=0 for 180–210 s</text>
  <text x="640" y="142" fill="var(--fg-muted)" font-size="8" text-anchor="end">dbfs −51 throughout: within 6 dB of decode_dbfs</text>
  <rect x="262" y="170" width="128" height="18" rx="3" fill="none" stroke="var(--accent)"/>
  <text x="326" y="183" text-anchor="middle" fill="var(--accent)" font-size="8">3 windows, no sync, own level</text>
  <line x1="390" y1="179" x2="420" y2="179" stroke="var(--accent)"/>
  <rect x="420" y="164" width="236" height="30" rx="4" fill="none" stroke="var(--accent)"/>
  <text x="538" y="176" text-anchor="middle" fill="var(--accent)" font-size="9">healDeafTier2: Reset + ResyncReset</text>
  <text x="538" y="188" text-anchor="middle" fill="var(--fg-muted)" font-size="8">WRN … coarse_offset_hz mm_mu mm_sps agc_level</text>
  <text x="340" y="230" text-anchor="middle" fill="var(--fg-muted)" font-size="9">an unkeyed repeater falls well below decode_dbfs − 6 dB, so idle silence never trips it</text>
</svg>
<figcaption>The deaf tap the 12 Sep log exposed: dibits and power steady, sync_hits at zero for minutes. The heal fires on the channel's own last-decoding level and logs the internals that pin the latch next time.</figcaption>
</figure>

## Field by field

| Field | Measures | Healthy | Worry when |
|---|---|---|---|
| `dibits` | dibits handed to the slicer this window | 4800 × window seconds | 0 — receiver not running or tap starved |
| `sync_hits` | burst-sync word matches | ≈ `bursts` when idle; ~33/s in traffic | 0 with `dibits` nominal and power at its decoding level — deaf tap |
| `fec_pass` / `fec_fail` | Voice LC Headers with BPTC + RS valid / not | ~10 per keyup, `fec_fail` 0–1 | `fec_fail` ≫ `fec_pass` — weak, clipped or bin-edge signal |
| `beacons` | BPTC-clean Idle carrying `IdleInfoPattern`, or CRC-valid CSBK | hundreds per window on a keyed idle repeater | 0 while `sync_hits` and `bursts` run high — vendor idle variant (`info_hex` in DEBUG) |
| `late_entries` | grants from two agreeing embedded LCs, no header | 0–1 per transmission on a weak tap | climbing with `fec_pass≈0`: every header lost at keyup |
| `rekeys` | re-keys inside hangtime released and re-granted | 0–1 per reply | never moves while History merges replies |
| `csbk_crc_fail` | BPTC-clean CSBKs whose CRC fails | 0 | a train for seconds — the unpinned cc=7 train |
| `deaf_heals` | lifetime receiver resets by the guard | 0 | any — read the WARN's internals |

`beacons` counts only a BPTC-clean Idle carrying the ETSI pattern
(`ff83df1732094ed1e7cd8a91`) or a CSBK that passed BPTC *and* its CRC; noise
forges neither, which is why a beacon may declare the lock.
`csbk_crc_fail` is deliberately **not** a bus decode error: the 9 Sep
443.2375 MHz log showed a real proprietary train of BPTC-clean, CRC-failing
CSBKs at cc=7 on both slots for ~4 s on a cc=12 system. The parked DEBUG
line (`dmr/tier2: CSBK CRC mismatch …`) carries `csbko`, `fid`, `lb`, `pf`
and `info_hex` for that case; this counter is its rate meter.

## When it doesn't look like that

The 12 Sep "missed a lot of calls" report is the canonical unhealthy
excerpt:

```
DBG widebandt2: channel decode activity freq_hz=442387500 system=Fire2 dibits=144000 sync_hits=0 bursts=0 fec_pass=0 fec_fail=0 beacons=0 late_entries=0 csbk_crc_fail=0 rekeys=0 locks_total=1 deaf_heals=0
```

Six stretches of 180–210 s like that in 30 minutes, alive 60–110 s between,
while the tap's power sat at its normal −51 dBFS and the Fire tap on the
*same* channelizer decoded throughout. The 20-minute capture aligned to that log decodes all 26 of its
transmissions offline, so the latch was state inside the live receiver, not
RF. Nothing WARNed: −51 dBFS is below the −45 floor of the no-sync hint
(`noSyncHintDbFS`) — the absolute-dBFS trap from the other side, and why
`healDeafTier2` gates on the channel's **own** last-decoding level.

Now the heal, as the 15 Sep log printed it 21 times in 6 minutes on a
GPSDO-locked X310 that decodes at 0 Hz:

```
WRN widebandt2: conventional DMR tap deaf at its decoding level — resetting the receiver (field report: 3-minute deaf stretches at a steady -51 dBFS; the receiver internals logged here are the instrument for pinning the latch) freq_hz=442387500 system=Fire2 dbfs=-49.3 decode_dbfs=-49.1 heals=1 coarse_offset_hz=-20100 agc_level=0.21 mm_mu=0.37 mm_sps=10 coarse_offset_rejected=true
```

`mm_sps=10` is 48 kHz over 4800 baud; `mm_mu` is the timing loop's phase, a
per-symbol countdown, **not a drift meter** (those two values and
`agc_level` are illustrative here; the offsets and levels are the log's). The
field that told the story
is `coarse_offset_hz=-20100`. The coarse carrier acquirer is a discriminator
mean with no bound and no decode check; in every 5–9 s gap between this
repeater's ~10 s idle trains the tap's power stayed at the train level
(Fire's dropped 5 dB, so this was *not* the repeater) while something at
≈442.3675 MHz dominated it, and the acquirer froze on it — every gap. By
frequency alone the stage cannot tell a neighbour from a 45 ppm tuner error,
so decode evidence settles it: an offset held for a whole window in which the
channel *synced* is confirmed; one it never synced under is rejected
(`coarse_offset_rejected=true`) and ignored within 500 Hz afterwards.

What the −20 kHz emitter *is* remains unknown — the 15 Sep capture does not
cover it. Do not "fix" it with a frequency bound: a 12.5 kHz adjacent
channel sits inside any bound that still serves the direct-mode offsets of
[#836](https://github.com/MattCheramie/GopherTrunk/issues/836). A capture
centred on 442.3875 MHz, ±100 kHz, over one idle cycle is the instrument.

| Symptom | Likely cause | Fix / read |
|---|---|---|
| `sync_hits=0`, `dibits` nominal, `dbfs` at its usual level | deaf tap — a receiver latch or a false coarse engage | read `coarse_offset_hz`; [DMR End to End 10]({{ '/blog/deep-dives/dmr-end-to-end-10-wideband-bin-edge-deaf-heal/' | relative_url }}) |
| `sync_hits=0`, `dibits=0` | the tap is starved — CPU, or the pump stopped | [Part 7]({{ '/blog/tutorials/field-notebook-07-overruns-host-drops/' | relative_url }}) |
| `sync_hits` high, `fec_pass=0`, `fec_fail` climbing | weak or clipped headers | [gain staging]({{ '/blog/tutorials/analog-edge-03-gain-staging/' | relative_url }}), [dBFS]({{ '/reference/dbfs/' | relative_url }}) |
| `late_entries` climbing, `fec_pass≈0`; replies merged in History | headers lost at keyup; a re-key deduped | [late entry]({{ '/reference/late-entry/' | relative_url }}), [DMR End to End 6]({{ '/blog/deep-dives/dmr-end-to-end-06-conventional-ipsc-two-calls/' | relative_url }}), [Cookbook 3]({{ '/blog/tutorials/operator-cookbook-03-conventional-dmr-two-slots/' | relative_url }}) |
| `beacons=0` on a keyed idle repeater | vendor idle variant, or a CRC-failing CSBK train | the parked `info_hex` DEBUG; [DMR End to End 7]({{ '/blog/deep-dives/dmr-end-to-end-07-idle-beacon-csbk/' | relative_url }}), [CSBK]({{ '/reference/csbk/' | relative_url }}) |
| WARN "strong in-channel signal but no sync" | tuner offset — or, in the OVERLOADED variant, clipping | fix `clip_ratio` first; [Spec to Shipping 13]({{ '/blog/deep-dives/from-spec-to-shipping-13-instruments-not-logs/' | relative_url }}) |

### How this line shapes operator practice

- **Read `dibits` first.** It alone separates a dead receiver from a deaf
  one — and it read `0` on every tap until the 15 Sep fix, so `dibits=0`
  beside `sync_hits>0` means an old build.
- **A quiet channel is judged against itself.** `decode_dbfs` in the WARN is
  the level the channel last synced at; no constant is involved. Never raise
  gain to make a heal stop.
- **One heal per idle gap is a neighbour, not a fault.** With
  `coarse_offset_rejected=true` the tap decodes anyway. A heal that does
  *not* restore sync is the open question — save that log.
- **Copy the parked DEBUG lines too.** `csbk_crc_fail` and `beacons=0` both
  point at an `info_hex` that pins a vendor train without a capture.

## Where this goes next

The DMR line is a decode census on a channel that is either silent or
keyed. [Part 6]({{ '/blog/tutorials/field-notebook-06-dmo-status-line/' | relative_url }})
reads a line where one counter is *designed* to race on an idle channel: the
TETRA DMO decode-status line, `dnb_total` against `dnb_qualified`, and the
per-transmission scramble-seed fields.

## FAQ

**What does `sync_hits=0` mean in a GopherTrunk DMR log?**
No burst-sync word matched in that window. Alone it is ambiguous — an
unkeyed repeater prints it too. Paired with `dibits` at its nominal 4800/s
and power at the channel's usual decoding level, it means a deaf tap, and
the heal resets the receiver after three such windows.

**Why does the DMR activity line only appear at debug level?**
It is a periodic census for investigations, so GopherTrunk parks it at DEBUG
on a 30 s cadence and logs it at once only when the channel's activity class
changes. Transitions (`cc locked`, `site alive`) stay at INFO and the hints
at WARN whatever `log.level` says.

**Is a climbing `late_entries` counter a problem?**
Not by itself. It counts transmissions granted from two agreeing embedded
Link Control blocks because the Voice LC Header bursts were lost — what a
subscriber radio does. It becomes a symptom beside `fec_pass≈0`: every
header is being lost, usually to level.

**What is `coarse_offset_hz` in the deaf-tap WARN?**
The frozen correction of the receiver's coarse carrier acquirer. A value the
channel actually synced under is the wanted carrier's offset; one it never
synced under (the 15 Sep log's −20100 Hz) is a neighbour that dominated the
tap while the repeater was silent, and the heal rejects it for good.

**Why does `beacons` stop while the repeater is still keyed?**
Either it dropped to a vendor idle variant the Idle-pattern match does not
count — the parked DEBUG line prints its `info_hex` — or it is sending a
CSBK train whose CRC fails, so `csbk_crc_fail` climbs instead. Both want a
capture of the frequency idle.

## Series navigation

**Part 5 of 14** · ←
[Part 4: The Carrier-Offset WARN — Alias Buckets & Persistence]({{ '/blog/tutorials/field-notebook-04-carrier-offset-warn/' | relative_url }})
· Next →
[Part 6: The DMO Status Line — Noise Meters & Traffic Counters]({{ '/blog/tutorials/field-notebook-06-dmo-status-line/' | relative_url }})
