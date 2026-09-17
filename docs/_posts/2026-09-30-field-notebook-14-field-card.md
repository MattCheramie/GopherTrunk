---
title: "The Field Notebook, Part 14: The One-Page Field Card"
description: "The series finale — one card for reading any GopherTrunk debug.log: the five lines to read first on any report, one table per protocol family mapping each line to healthy, worry-when and the part that explains it, the noise-meter-versus-traffic-counter list, and an honest note on what the log still cannot tell you."
category: tutorials
keywords: gophertrunk log cheat sheet, sdr scanner log field card, read debug log quickly, decode status healthy values, noise meter vs traffic counter, sync_hits dibits host_drops coherence, tetra dmr log troubleshooting card, scanner log reference card, gophertrunk field notebook
tags: [field-notebook, logs, reference, finale, troubleshooting, tutorial]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "The Field Notebook"
series_part: 14
---

*Part 14 — the last — of **The Field Notebook**, a 14-part operator's
tutorial that reads GopherTrunk's `debug.log` one line family at a time —
what each field measures, what a healthy rig prints, which number is a noise
meter and which one means traffic, and which deep dive to open when a line
goes wrong. Thirteen parts read the log from the startup banner
([Part 1]({{ '/blog/tutorials/field-notebook-01-startup-lines/' | relative_url }}))
to the bug report
([Part 13]({{ '/blog/tutorials/field-notebook-13-from-log-to-issue/' | relative_url }})).
This closing part folds them onto one page: the card to keep beside the
terminal.*

> **TL;DR:** On any report, read five lines first: the `daemon:` WARNs in
> the startup block, the protocol's `cc locked` line, its periodic status
> line (`tetra: decode status`, `widebandt2: channel decode activity`,
> `tetra dmo: decode status`, the MRC health line), the `soapyremote: SDR
> overruns` / `ccdecoder: decode can't keep up` WARNs, and
> `recorder: call ended … reason=`. Then the family tables give each line's
> healthy shape, its worry-when, and the part that explains it. Keep noise
> meters apart from traffic counters — `dnb_total` vs `dnb_qualified`,
> `bsch_ok` vs `sch_pdus`, `dibits` vs `sync_hits`, `coherence` vs
> `lock_gate`, `updates` vs `holds`. And know what the log cannot yet say:
> the −20 kHz emitter, the cc=7 CSBK train, and the on-air gates
> ([#836](https://github.com/MattCheramie/GopherTrunk/issues/836),
> [#1003](https://github.com/MattCheramie/GopherTrunk/issues/1003),
> [#1187](https://github.com/MattCheramie/GopherTrunk/issues/1187)).

**Key takeaways**

- **Five lines settle most reports.** Startup WARNs, lock, status line,
  host WARNs, teardown reason — in that order, before any DSP theory.
- **Every family has a noise meter and a traffic counter.** The card names
  the pair; reason from the one that means decode.
- **"Worry when" is relative, never absolute.** Deaf is judged at the
  channel's *own* last-decoding level, coherence against its *own*
  `lock_gate`.
- **The card ends where the log does.** Some symptoms are open questions
  with an instrument waiting for a capture; the card says which.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| Startup & lock | banner, `daemon:` WARNs, `cchunt:` lines, `cc locked` | Parts [1]({{ '/blog/tutorials/field-notebook-01-startup-lines/' | relative_url }}), [2]({{ '/blog/tutorials/field-notebook-02-lock-lines/' | relative_url }}) |
| TETRA TMO | `tetra: decode status`, the #815 offset WARN | Parts [3]({{ '/blog/tutorials/field-notebook-03-tetra-decode-status/' | relative_url }}), [4]({{ '/blog/tutorials/field-notebook-04-carrier-offset-warn/' | relative_url }}) |
| DMR | `widebandt2: channel decode activity`, deaf heal, idle beacon | Part [5]({{ '/blog/tutorials/field-notebook-05-dmr-activity-line/' | relative_url }}) |
| TETRA DMO | `tetra dmo: decode status`, seed lines, grant | Part [6]({{ '/blog/tutorials/field-notebook-06-dmo-status-line/' | relative_url }}) |
| Host & front end | overruns, `decode can't keep up`, MRC health line | Parts [7]({{ '/blog/tutorials/field-notebook-07-overruns-host-drops/' | relative_url }}), [8]({{ '/blog/tutorials/field-notebook-08-mrc-health-line/' | relative_url }}) |
| Captures & recordings | `siglab: capture …`, `recorder: call …`, composer lines | Parts [9]({{ '/blog/tutorials/field-notebook-09-capture-lines/' | relative_url }}), [10]({{ '/blog/tutorials/field-notebook-10-recorder-lines/' | relative_url }}), [11]({{ '/blog/tutorials/field-notebook-11-voice-chain-lines/' | relative_url }}) |
| Web & reporting | panel symptoms with log evidence; what to paste | Parts [12]({{ '/blog/tutorials/field-notebook-12-web-symptoms-log-stories/' | relative_url }}), [13]({{ '/blog/tutorials/field-notebook-13-from-log-to-issue/' | relative_url }}) |

## In this post

- **The five lines to read first** — the order that settles most reports.
- **The card** — one table per family: line → healthy → worry → open this.
- **Noise meters vs traffic counters** — the pairs that mislead.
- **What the log still cannot tell you** — the honest list.

## The five lines to read first

Every report in this series' source material was settled faster by this
reading order than by any theory.

**1. The startup WARNs.** `daemon: gain looks like dB, not tenths-of-dB —
radio may be effectively deaf` explained a −70 dBFS control channel before
anyone measured anything. Two systems on one `role: control` tuner, dead
`channels:`, a missing band plan — all announced here.

**2. The lock line.** `tetra cc locked freq= mcc= mnc= la=`,
`control channel locked nac= freq=` (P25), `dmr cc locked freq= cc= sysid=`,
`tetra dmo cc locked`. Without one, nothing downstream can be judged;
`cchunt: hunt failed — no control-channel lock` with `iq_observed=` says
whether IQ even arrived.

**3. The status line.** The protocol's periodic DEBUG line is the
instrument. Read its trend across intervals, never one line.

**4. The host WARNs.** `soapyremote: SDR overruns … host_drops=` and
`ccdecoder: decode can't keep up with real time … dropped_since_last=` are
downstream signals — the consumer stopped draining
([Part 7]({{ '/blog/tutorials/field-notebook-07-overruns-host-drops/' | relative_url }})).

**5. The teardown reason.** `recorder: call ended … reason=timeout` on
every call means the voice chain decoded nothing that refreshed liveness —
a decode problem in a recorder's clothes
([Part 11]({{ '/blog/tutorials/field-notebook-11-voice-chain-lines/' | relative_url }})).

Healthy, on a TETRA rig:

```
INF tetra cc locked freq=467912500 mcc=250 mnc=1 la=1021 system=Metro-TETRA
DBG tetra: decode status system=Metro-TETRA locked=true carrier_off_hz=-412.5 baud=18000 baud_dev_pct=0.01 sb_bursts=72 bsch_ok=72 bsch_fail=0 sysinfo=18 sch_pdus=340 sch_pdus_fail=6 frag_abandons=1 grants=3 colour_code=1
INF recorder: call ended device=cc:same-carrier:1 wav=../recordings/Metro-TETRA/2001/... duration=5.6s reason=normal
```

Unhealthy — three lines, three different parts:

```
WRN daemon: gain looks like dB, not tenths-of-dB — radio may be effectively deaf serial=32B0A4C configured=50 parsed_db=5 did_you_mean=500
DBG tetra: decode status system=Metro-TETRA locked=true carrier_off_hz=4950.0 baud=17998 baud_dev_pct=0.02 sb_bursts=70 bsch_ok=69 bsch_fail=1 sysinfo=0 sch_pdus=0 sch_pdus_fail=58 frag_abandons=0 grants=0 colour_code=1
INF recorder: call ended device=cc:same-carrier:1 wav=../recordings/Site-DMO/0/... duration=7.0s reason=timeout
```

The middle line is the 19 Aug "locked but deaf" signature — `bsch_ok` ~100 %,
`sch_pdus=0`, `carrier_off_hz` one alias bucket (4500 Hz) off
([Part 4]({{ '/blog/tutorials/field-notebook-04-carrier-offset-warn/' | relative_url }})).

## The card

Every field below is a real `slog` key; every "open" is the part or deep
dive that explains the mechanism.

**Startup & lock — all protocols**

| Line / field | Healthy | Worry when | Open |
|---|---|---|---|
| `daemon:` WARNs at startup | none | any — `gain` in dB, dead `channels:`, two systems on one control tuner | [Part 1]({{ '/blog/tutorials/field-notebook-01-startup-lines/' | relative_url }}) |
| `cchunt: hunt failed — no control-channel lock` `iq_observed=` | absent | repeating — `false` means no IQ, `true` means IQ but no decode | [Part 2]({{ '/blog/tutorials/field-notebook-02-lock-lines/' | relative_url }}) |
| `ccdecoder: control carrier offset far from configured frequency … (issue #815)` `offset_hz=` | absent | persistent > 10 s — wrong site or mistuned; a one-off 5004 Hz was an alias bucket | [Part 4]({{ '/blog/tutorials/field-notebook-04-carrier-offset-warn/' | relative_url }}) |

**TETRA TMO**

| Line / field | Healthy | Worry when | Open |
|---|---|---|---|
| `bsch_ok` / `bsch_fail` | ~100 % ok | `bsch_fail` climbing, `sb_bursts` collapsing → resync storm, `MarkLost` | [TETRA 10]({{ '/blog/deep-dives/tetra-end-to-end-10-control-channel-sync-loss/' | relative_url }}) |
| `sch_pdus` vs `bsch_ok` | both advancing | `bsch_ok` high, `sch_pdus=0` — locked but deaf (AFC alias) | [Part 3]({{ '/blog/tutorials/field-notebook-03-tetra-decode-status/' | relative_url }}) |
| `tetra: payload drought persists across resyncs … declaring lock lost` | absent | present — the escape hatch fired; re-hunt follows | [Part 3]({{ '/blog/tutorials/field-notebook-03-tetra-decode-status/' | relative_url }}) |

**TETRA DMO**

| Line / field | Healthy | Worry when | Open |
|---|---|---|---|
| `dnb_total` | thousands, ~18/s on silence | never — a noise meter | [Part 6]({{ '/blog/tutorials/field-notebook-06-dmo-status-line/' | relative_url }}) |
| `dnb_qualified`, `tch_crc` | rise together in a PTT | `dsb_schs_crc` climbs, `tch_crc` at the floor — seed not recovered | [Cookbook 5]({{ '/blog/tutorials/operator-cookbook-05-tetra-dmo/' | relative_url }}) |
| `seed_known` / `seed_verified` / `solve_rejects` | known + verified per PTT | `composer: tetra DMO scramble seed changed` flip-flopping — rejected solves are now counted, not adopted | [TETRA 13]({{ '/blog/deep-dives/tetra-end-to-end-13-dmo-pipeline-grants/' | relative_url }}) |

**DMR — wideband and conventional**

| Line / field | Healthy | Worry when | Open |
|---|---|---|---|
| `dibits` vs `sync_hits` | both advancing | `dibits` up, `sync_hits=0` for minutes at the channel's normal level — the deaf tap | [DMR 10]({{ '/blog/deep-dives/dmr-end-to-end-10-wideband-bin-edge-deaf-heal/' | relative_url }}) |
| `csbk_crc_fail` | ~0 | a 30 ms-cadence train at another colour code — real, vendor, unpinned | [DMR 7]({{ '/blog/deep-dives/dmr-end-to-end-07-idle-beacon-csbk/' | relative_url }}) |
| `deaf_heals`, `widebandt2: conventional DMR tap deaf at its decoding level — resetting the receiver` | 0 | every ~15 s with `coarse_offset_hz` far off — a false coarse engage, now rejected | [Part 5]({{ '/blog/tutorials/field-notebook-05-dmr-activity-line/' | relative_url }}) |

**Host & front end**

| Line / field | Healthy | Worry when | Open |
|---|---|---|---|
| `soapyremote: SDR overruns …` `host_drops=` | absent | present — find what got slower downstream | [Part 7]({{ '/blog/tutorials/field-notebook-07-overruns-host-drops/' | relative_url }}) |
| MRC `coherence` vs `lock_gate` | above the gate, `calibrated=true` | below — judge against the logged gate, never 0.5 | [Weak-Signal 10]({{ '/blog/deep-dives/weak-signal-engineering-10-mrc-calibration/' | relative_url }}) |
| `updates` vs `holds` | `updates` climbing, `holds` ~0 | `updates` frozen while `holds` climbs — stale gain | [Part 8]({{ '/blog/tutorials/field-notebook-08-mrc-health-line/' | relative_url }}) |

**Captures, recorder, voice chain**

| Line / field | Healthy | Worry when | Open |
|---|---|---|---|
| `siglab: capture started` `center_hz` vs `requested_center_hz` | equal | differ — the slice was carved at the tuner centre | [Part 9]({{ '/blog/tutorials/field-notebook-09-capture-lines/' | relative_url }}) |
| `composer: … voice follow ended` `speech_frames`, `bfi_count`, `vocoder_drops` | `speech_frames` ≫ `bfi_count` | `speech_frames=0` — the seed or the tap | [Part 11]({{ '/blog/tutorials/field-notebook-11-voice-chain-lines/' | relative_url }}) |

**Web**

| Line / field | Healthy | Worry when | Open |
|---|---|---|---|
| History vs `recorder: call started` timestamps | agree in local time | offset by exactly your UTC offset — a render bug | [Part 12]({{ '/blog/tutorials/field-notebook-12-web-symptoms-log-stories/' | relative_url }}) |

<figure class="lab-figure">
<svg viewBox="0 0 680 250" width="680" height="250" role="img" aria-label="The field card as a procedure: a report passes through five numbered checks in order — startup WARNs, the lock line, the status-line trend, host WARNs, the teardown reason — and the first failing check routes to a family table, which points to the Field Notebook part and deep dive that explain the mechanism.">
  <rect x="8" y="100" width="70" height="40" rx="5" fill="none" stroke="currentColor"/>
  <text x="43" y="124" text-anchor="middle" fill="currentColor" font-size="10">report</text>
  <line x1="78" y1="120" x2="100" y2="120" stroke="currentColor"/><polygon points="98,116 106,120 98,124" fill="currentColor"/>
  <rect x="106" y="24" width="120" height="30" rx="4" fill="none" stroke="var(--accent)"/>
  <text x="166" y="43" text-anchor="middle" fill="var(--accent)" font-size="9">1 startup WARNs</text>
  <rect x="106" y="62" width="120" height="30" rx="4" fill="none" stroke="var(--accent)"/>
  <text x="166" y="81" text-anchor="middle" fill="var(--accent)" font-size="9">2 cc locked</text>
  <rect x="106" y="100" width="120" height="30" rx="4" fill="none" stroke="var(--accent)"/>
  <text x="166" y="119" text-anchor="middle" fill="var(--accent)" font-size="9">3 status-line trend</text>
  <rect x="106" y="138" width="120" height="30" rx="4" fill="none" stroke="var(--accent)"/>
  <text x="166" y="157" text-anchor="middle" fill="var(--accent)" font-size="9">4 host WARNs</text>
  <rect x="106" y="176" width="120" height="30" rx="4" fill="none" stroke="var(--accent)"/>
  <text x="166" y="195" text-anchor="middle" fill="var(--accent)" font-size="9">5 reason=</text>
  <line x1="226" y1="39" x2="270" y2="39" stroke="currentColor"/><polygon points="268,35 276,39 268,43" fill="currentColor"/>
  <line x1="226" y1="77" x2="270" y2="77" stroke="currentColor"/><polygon points="268,73 276,77 268,81" fill="currentColor"/>
  <line x1="226" y1="115" x2="270" y2="115" stroke="currentColor"/><polygon points="268,111 276,115 268,119" fill="currentColor"/>
  <line x1="226" y1="153" x2="270" y2="153" stroke="currentColor"/><polygon points="268,149 276,153 268,157" fill="currentColor"/>
  <line x1="226" y1="191" x2="270" y2="191" stroke="currentColor"/><polygon points="268,187 276,191 268,195" fill="currentColor"/>
  <rect x="276" y="18" width="200" height="214" rx="6" fill="none" stroke="var(--fg-muted)" stroke-dasharray="4 3"/>
  <text x="376" y="34" text-anchor="middle" fill="var(--fg-muted)" font-size="9">family tables</text>
  <text x="376" y="82" text-anchor="middle" fill="currentColor" font-size="9">startup · TETRA · DMO · DMR</text>
  <text x="376" y="120" text-anchor="middle" fill="currentColor" font-size="9">host · captures · recorder · web</text>
  <line x1="476" y1="125" x2="520" y2="125" stroke="currentColor"/><polygon points="518,121 526,125 518,129" fill="currentColor"/>
  <rect x="526" y="90" width="146" height="70" rx="6" fill="none" stroke="var(--accent)" stroke-width="2"/>
  <text x="599" y="120" text-anchor="middle" fill="var(--accent)" font-size="9">Field Notebook part</text>
</svg>
<figcaption>Five checks in order; the first failure routes to a family table, and every row ends in a part that explains the mechanism.</figcaption>
</figure>

## Noise meters vs traffic counters

Five pairs recur through the series; in each, the first moves on noise and
the second means decode. Reason from the second.

- **`dnb_total` vs `dnb_qualified`** (DMO). The DNB correlator false-fires
  ~18/s on a silent channel by design arithmetic; only slot-grid-qualified
  bursts mean traffic.
- **`bsch_ok` vs `sch_pdus`** (TETRA). BSCH survives a 4-rotation search and
  heavy FEC, so it decodes through a wrong AFC alias bucket; `sch_pdus` does
  not.
- **`dibits` vs `sync_hits`** (DMR). Dibits prove the stream flows, not that
  any of it is DMR; the 12 Sep tap produced dibits for minutes with
  `sync_hits=0`.
- **Wideband `coherence` vs `lock_gate`** (MRC). |ρ| is diluted by every
  hertz of noise-only bandwidth — 0.16 on a rig decoding 1425 CRC-clean
  BSCH — so it only means something against the logged gate.
- **`updates` vs `holds`** (MRC). `updates` frozen with `holds` climbing is
  a combine running on a gain measured minutes ago.

Two rules of the same shape: **never judge a channel by absolute dBFS**
([Analog Edge 13]({{ '/blog/tutorials/analog-edge-13-coherence-not-dbfs/' | relative_url }})),
and **CRC yield is the only trustworthy metric; EVM is a trap**
([Weak-Signal 2]({{ '/blog/deep-dives/weak-signal-engineering-02-metrics-that-lie/' | relative_url }})).

## What the log still cannot tell you

An honest card ends with the rows that have no "healthy" column yet.

- **The −20 kHz emitter near 442.3675 MHz** (15 Sep). The deaf heal rejects
  the false coarse engage it caused, but what transmits there is unknown; a
  ±100 kHz capture centred on 442.3875 MHz over one idle cycle is the
  instrument.
- **The cc=7 CSBK-CRC-fail train** on a cc=12 IPSC system (9 Sep). Parked
  with `csbko`/`fid`/`info_hex` in the DEBUG line.
- **Three on-air gates.** DMR direct mode is capture-verified but not yet
  run live on the fixed build (#836); the DMO pre-roll and grant-on-adoption
  pair is capture-validated only (#1003); DMR Enhanced Privacy decrypts the
  known-key captures and the live call is still owed (#1187). Those rows
  read "capture-verified, live run pending".
- **P25 Phase 1 weak-signal voice.** The one C4FM path with neither
  equalizer nor soft FEC; no fix without a marginal voice capture
  ([P25 12]({{ '/blog/deep-dives/p25-end-to-end-12-weak-signal-gap/' | relative_url }})).

### How this card shapes operator practice

- **Read in order, stop at the first failure.** The checks are ranked by
  how often they ended a thread.
- **Quote the pair, not the number.** `bsch_ok=72 sch_pdus=0` is a
  diagnosis; `bsch_ok=72` alone is a reassurance.
- **Judge every "worry" relatively.** Against the channel's own level, its
  own gate, its own previous interval.
- **Treat the open rows as invitations.** Each names the capture that would
  close it; [Part 13]({{ '/blog/tutorials/field-notebook-13-from-log-to-issue/' | relative_url }})
  says how to make and paste it.

## Where to go from here

The notebook read the log; the rest of the blog explains the machinery. The
[series index]({{ '/blog/series/field-notebook/' | relative_url }}) is the
card's table of contents. For the recipes that produce these logs,
[The Operator's Cookbook]({{ '/blog/series/operator-cookbook/' | relative_url }});
for when a fix is *verified*,
[From Spec to Shipping]({{ '/blog/series/from-spec-to-shipping/' | relative_url }}).
Two series launch alongside this finale:
[DMR End to End]({{ '/blog/series/dmr-end-to-end/' | relative_url }}) follows
one carrier from 4FSK to decrypted Enhanced Privacy, and
[Beyond Voice]({{ '/blog/series/beyond-voice/' | relative_url }}) walks the
eleven-place pattern behind every non-voice decoder. When a row sends you to
a capture, the [Analog Edge]({{ '/blog/series/analog-edge/' | relative_url }})
tells you how to record it well.

## FAQ

**What are the first lines to check in a GopherTrunk log?**
In order: the `daemon:` WARNs in the startup block, the protocol's
`cc locked` line, the trend of its periodic status line, the
`soapyremote: SDR overruns` / `ccdecoder: decode can't keep up` WARNs, and
`recorder: call ended … reason=`. The first one that fails names the family
table — and the part — to open.

**Which log counters are noise meters rather than traffic?**
`dnb_total` (DMO), `dibits` (DMR), `bsch_ok` on its own (TETRA), wideband
`coherence` read without its `lock_gate`, and `calibrated=true` without
`updates` moving. Their partners — `dnb_qualified`, `sync_hits`,
`sch_pdus`, `lock_gate`, `updates` — are the numbers that mean decode.

**What does "locked but deaf" look like in the TETRA status line?**
`locked=true`, `bsch_ok` near 100 % of `sb_bursts`, `sch_pdus=0`, and
`carrier_off_hz` about ±4500 Hz from where it sat — the AFC latched an alias
bucket. The payload heartbeat forces a resync after 12 s and escalates to
`MarkLost` after three fruitless ones.

**Why does the card give no dBFS thresholds?**
Because every absolute-power gate in the project's history became a
gain-staging trap: a −56 dBFS control channel decoded every C_ALOHA while a
WARN called it too weak, and MRC once demanded −40 dBFS to calibrate. The
card judges everything relative to the channel's own level or gate.

**Which parts of GopherTrunk are still awaiting on-air verification?**
DMR direct mode's live run on the fixed build (#836), the DMO pre-roll and
grant-on-adoption pair in the live daemon (#1003), a live DMR Enhanced
Privacy call with `encryption_keys` (#1187), and P25 Phase 1 weak-signal
voice, which needs a marginal capture first.

## Series navigation

**Part 14 of 14** · ←
[Part 13: From Log to Issue — What to Paste, What to Capture]({{ '/blog/tutorials/field-notebook-13-from-log-to-issue/' | relative_url }})
· [Back to the series index]({{ '/blog/series/field-notebook/' | relative_url }})
