---
title: "Squelch, Part 14: The Squelch Playbook"
description: "The finale — every gate on GopherTrunk's scan list folded into one map from channel mode to statistic to config key, the failure signatures operators report matched to the gate and log line that explains each, the ledger of twin paths that drifted apart, and the honest list of what is still unverified on air."
category: deep-dives
keywords: scanner squelch troubleshooting, sdr scan list opens on noise, noise squelch vs level squelch, ctcss dcs gate not opening, am airband squelch carrier to noise, dmr nxdn d-star call gate, scan list call cut into pieces, conventional scanner playbook, gophertrunk squelch
tags: [squelch, playbook, troubleshooting, scanner, methodology, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "Squelch"
series_part: 14
---

*Part 14 — the last — of **Squelch**, a 14-part deep dive into GopherTrunk's
conventional scanner — the scan list with no control channel to tell it a
call exists — and every gate that decides when a channel opens. Thirteen
parts ago we started with a dwell loop that asked one question, "is the
level above a number?". Since then that question has been split into
channel power, a noise squelch, CTCSS and DCS, AM carrier-to-noise, call
gates for five digital protocols, a bridge for bursty carriers, data taps
and a shared front end. This closing part compresses it into what you
need at the bench: one gate map, the failure signatures, the twins that
drifted, and an honest list of what is still open.*

> **TL;DR:** A scan-list channel opens when its **level** clears
> `squelch_dbfs` *and* its mode's **statistic** agrees — and every symptom
> lives at one of those gates. Opens on empty air → the level measured the
> wrong thing (Part 2) or sat under the noise floor (Part 3); read
> `quieting_db`. Records noise for half an hour → no noise squelch. Never
> opens → tone polarity, mode or width (Parts 5, 6, 9). Opens late → the
> gate's own window: 25 ms for the noise squelch, a 250 ms CTCSS block, up
> to 770 ms for NXDN48. One over cut into pieces → DMR direct mode without
> the 100 ms bridge (Part 10). A call that never ends on a silent channel →
> a stalled stream, ended after 3 s. The running rule: **never gate on
> absolute dBFS when a scale-invariant statistic exists**.

**Key takeaways**

- **Find the gate before touching a threshold.** Each gate logs what it
  measured as it opened; the easiest mistake to make is moving
  `squelch_dbfs` to fix a problem a level cannot see.
- **Every level threshold ended up with a statistic beside it.** A noise
  ratio for FM, carrier-to-noise for AM, frame evidence for digital — and
  the two are ANDed, because each covers the other's blind spot.
- **Every gate costs milliseconds, and the cost is stated.** A statistic
  needs a window; the scan window is extended once to give it one, and a
  silent channel pays nothing.
- **The open items wait on evidence, not effort.** CTCSS, DCS and AM are
  on-air verified; the rest are named below with what each needs.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| "Opens on empty air" triage | span power, then `quieting_db`, then a missing call gate | [Part 2]({{ '/blog/deep-dives/squelch-02-channel-power/' | relative_url }}), [Part 3]({{ '/blog/deep-dives/squelch-03-noise-squelch/' | relative_url }}), [Part 8]({{ '/blog/deep-dives/squelch-08-call-gates-p25-dmr/' | relative_url }}) |
| "Never opens" triage | tone and polarity, then mode and width, then what the noise squelch calls a carrier | [Part 5]({{ '/blog/deep-dives/squelch-05-ctcss/' | relative_url }}), [Part 6]({{ '/blog/deep-dives/squelch-06-dcs/' | relative_url }}), [Part 9]({{ '/blog/deep-dives/squelch-09-call-gates-ysf-nxdn-dstar/' | relative_url }}) |
| Whistle or thin audio | clipping products at 4× the carrier offset; an AM sideband cut by a mistuned filter | [Part 4]({{ '/blog/deep-dives/squelch-04-offset-tuning/' | relative_url }}), [Part 7]({{ '/blog/deep-dives/squelch-07-am/' | relative_url }}) |
| Calls cut up, or never ending | `dmrBurstBridge`, `StreamStallTimeout` | [Part 10]({{ '/blog/deep-dives/squelch-10-bursty-carriers/' | relative_url }}) |
| Data bursts | scan-list taps and dedicated-SDR decoders behind one filter design | [Part 11]({{ '/blog/deep-dives/squelch-11-data-on-the-scan-list/' | relative_url }}), [Part 12]({{ '/blog/deep-dives/squelch-12-narrowband-front-end/' | relative_url }}) |
| What the operator turns | priority scan, lockouts by frequency, `audio_gain_db` | [Part 13]({{ '/blog/deep-dives/squelch-13-operator-controls/' | relative_url }}) |

## In this post

- **The gate map** — channel mode to gate to statistic to config key, in one table.
- **Failure signatures** — the reports operators file, each matched to a gate and a log line.
- **The twin ledger** — the pairs that drifted apart, with receipts.
- **What's still open** — every unverified gate, named, with what closes it.

## The gate map

The scanner of
[Part 1]({{ '/blog/deep-dives/squelch-01-no-control-channel/' | relative_url }})
tunes a channel, watches it for a scan window (100 ms unless a gate needs
longer), and opens a synthetic call when the gates agree. Which gates
depends on the channel's `mode`: find the row, then read its statistic.

| Channel mode | Gates, in order | Statistic | Default | Config | Part |
|---|---|---|---|---|---|
| Analog FM | level AND noise squelch | RMS power behind a ±8 kHz filter; high-passed discriminator noise over its white-noise reference | −50 dBFS; ratio opens at 0.5, holds to 0.6 | `squelch_dbfs`, `noise_squelch` | [2]({{ '/blog/deep-dives/squelch-02-channel-power/' | relative_url }}), [3]({{ '/blog/deep-dives/squelch-03-noise-squelch/' | relative_url }}) |
| FM with CTCSS | the FM gates AND tone | exact-frequency Goertzel over 250 ms that must beat the adjacent EIA tones | off | `tone.mode: ctcss`, `tone.ctcss_hz` | [5]({{ '/blog/deep-dives/squelch-05-ctcss/' | relative_url }}) |
| FM with DCS | the FM gates AND code | 23-bit window against the code's rotations, held for 16 bits | off; polarity normal | `tone.dcs_code`, `tone.dcs_polarity` | [6]({{ '/blog/deep-dives/squelch-06-dcs/' | relative_url }}) |
| AM | carrier-to-noise alone | strongest bin within ±4.5 kHz over the 20th-percentile floor within ±12 kHz | 12 dB | `mode: am`, `squelch_cn_db` | [7]({{ '/blog/deep-dives/squelch-07-am/' | relative_url }}) |
| P25 | level AND NAC, when set | a NID carrying the wanted NAC within 750 ms | no NAC filter | `mode: p25`, `nac` | [8]({{ '/blog/deep-dives/squelch-08-call-gates-p25-dmr/' | relative_url }}) |
| DMR | level AND call gate | voice sync, Voice LC Header or PI header within 1 s | always on | `mode: dmr` | [8]({{ '/blog/deep-dives/squelch-08-call-gates-p25-dmr/' | relative_url }}) |
| System Fusion | level AND call gate | a frame sync within 500 ms | always on | `mode: ysf` | [9]({{ '/blog/deep-dives/squelch-09-call-gates-ysf-nxdn-dstar/' | relative_url }}) |
| NXDN | level AND call gate | two frame syncs 192 dibits apart and a traffic LICH | always on | `mode: nxdn` or `nxdn48` | [9]({{ '/blog/deep-dives/squelch-09-call-gates-ysf-nxdn-dstar/' | relative_url }}) |
| D-STAR | level AND call gate | a CRC-valid header, or slow-data syncs one or two cycles apart | always on | `mode: dstar` | [9]({{ '/blog/deep-dives/squelch-09-call-gates-ysf-nxdn-dstar/' | relative_url }}) |
| FM or AM with data taps | the channel's first test only; the tone gate is ignored | the burst's own sync word and block check | none | `decoders` | [11]({{ '/blog/deep-dives/squelch-11-data-on-the-scan-list/' | relative_url }}) |

Three readings of the table. The level is alone only where the operator
asked (`noise_squelch: false`) or configured nothing better: a `p25`
channel without `nac`. AM has no level test at all — `squelch_dbfs` is
ignored there. And once a call is open, the same statistics hold it: the
dwell ends after `hangtime_ms` (1500) of the gates reading false, with a
3 dB hysteresis on the level and a 50 ms debounce on blips.

<figure class="lab-figure">
<svg viewBox="0 0 680 250" width="680" height="250" role="img" aria-label="The decision to open a scan-list channel as a chain: a scan window, the in-channel level test, a second gate chosen by channel mode, then dwell and hangtime. Five rows list each mode's statistic and its part: FM noise squelch, CTCSS and DCS, the P25 NAC and DMR call evidence, System Fusion, NXDN and D-STAR frame evidence, and AM carrier-to-noise.">
  <g fill="none" stroke="currentColor">
    <rect x="10" y="22" width="120" height="40" rx="6"/>
    <rect x="160" y="22" width="150" height="40" rx="6"/>
    <rect x="530" y="22" width="140" height="40" rx="6"/>
    <line x1="130" y1="42" x2="160" y2="42"/>
    <line x1="310" y1="42" x2="340" y2="42"/>
    <line x1="500" y1="42" x2="530" y2="42"/>
  </g>
  <rect x="340" y="22" width="160" height="40" rx="6" fill="none" stroke="var(--accent)" stroke-width="1.5"/>
  <g text-anchor="middle" font-size="9" fill="currentColor">
    <text x="70" y="39">scan window</text>
    <text x="70" y="52">100 ms by default</text>
    <text x="235" y="39">in-channel level</text>
    <text x="235" y="52">squelch_dbfs · P2</text>
    <text x="600" y="39">dwell, then hangtime</text>
    <text x="600" y="52">1500 ms by default</text>
  </g>
  <g text-anchor="middle" font-size="9" fill="var(--accent)">
    <text x="420" y="39">the mode's second gate</text>
    <text x="420" y="52">a statistic, not a level</text>
  </g>
  <line x1="420" y1="62" x2="420" y2="80" stroke="var(--accent)"/>
  <g fill="none" stroke="var(--fg-muted)">
    <rect x="10" y="80" width="660" height="24" rx="4"/>
    <rect x="10" y="108" width="660" height="24" rx="4"/>
    <rect x="10" y="136" width="660" height="24" rx="4"/>
    <rect x="10" y="164" width="660" height="24" rx="4"/>
    <rect x="10" y="192" width="660" height="24" rx="4"/>
  </g>
  <g font-size="8.5" fill="currentColor">
    <text x="18" y="96">fm / nfm — noise squelch: discriminator noise over its white-noise reference, opens at 0.5 and holds to 0.6</text>
    <text x="18" y="124">fm / nfm with tone — a CTCSS Goertzel over 250 ms, or the 23-bit DCS word held for 16 bits</text>
    <text x="18" y="152">p25 — a NID carrying the configured NAC · dmr — voice sync, Voice LC Header or PI header</text>
    <text x="18" y="180">ysf — frame sync · nxdn — two syncs 192 dibits apart and a traffic LICH · dstar — header or sync pair</text>
    <text x="18" y="208">am — carrier-to-noise from the channel's own spectrum replaces the level: opens at 12 dB</text>
  </g>
  <g text-anchor="end" font-size="8.5" fill="var(--accent)">
    <text x="662" y="96">P3</text>
    <text x="662" y="124">P5 · P6</text>
    <text x="662" y="152">P8</text>
    <text x="662" y="180">P9</text>
    <text x="662" y="208">P7</text>
  </g>
  <text x="10" y="236" fill="var(--fg-muted)" font-size="8.5">data decoders (mdc1200, fleetsync, acars) ride the channel's first test and ignore the tone gate — P11</text>
</svg>
<figcaption>One decision, two questions: is there power in the channel, and is it the kind of signal this channel is for.</figcaption>
</figure>

## Failure signatures

The map is for methodical triage; these are the shortcuts — the reports
filed during this series, each matched to its gate.

**"It opens on empty air."** Read the line the scanner writes as it opens,
`conv: squelch opened`: it carries `level_dbfs`, `squelch_dbfs` and, on
analog FM, `quieting_db`. Before
[#1239](https://github.com/MattCheramie/GopherTrunk/issues/1239) the level
was `PowerDbFS` of the SDR's whole span, so any carrier anywhere held every
channel open (Part 2). After it, the level is the channel's — but still
absolute, so a threshold under the noise floor or a tuner on `gain: auto`
reads "carrier" for good (Part 3). `quieting_db` in `GET /api/v1/scanner`
settles which: about 0 on noise, 3 where the squelch opens, 15 and up on a
usable signal. On a digital channel the question is whether a call gate
exists: a `p25` channel without `nac` still opens on level alone.

**"It recorded noise for half an hour."** The PlutoSky scan list of
[#1329](https://github.com/MattCheramie/GopherTrunk/issues/1329) opened a
call 100 ms after start-up and recorded noise for 33 and 41 minutes. The
noise squelch reads 0.98–1.04 on noise at any level and rate, so it never
opens there, and a call in progress must stay under 0.6 — it ends when the
transmitter drops even if what is left still clears `squelch_dbfs`. The
informational line `conv: channel is above squelch_dbfs with no carrier on
it` names a channel the level alone would have opened;
`noise_squelch: false` is the A/B.

**"It never opens."** On a toned channel, look for `conv: DCS code heard
with the opposite polarity; gate kept shut` — an N/I mismatch, fixed with
`tone.dcs_polarity`. On a digital channel check the mode: NXDN96 audio on
an `nxdn48` channel never confirms a frame, and an analog carrier on a
`ysf` channel correctly never opens. On FM, remember what the noise squelch
calls a carrier: a broadcast station's ±75 kHz reads 1.0 in a ±8 kHz
channel, like noise.

**"It opens late."** Each statistic needs a window. The noise squelch
reports a carrier about 25 ms after it comes up; CTCSS needs a 250 ms
block; a DCS match takes a full word plus 16 confirming bits; NXDN96
confirmed its first frame in 60–100 ms on the real-air slices and NXDN48 in
440–770 ms, hence the 1 s window on `nxdn48`. `squelch_open_ms` adds a
deliberate delay on top, and on the scan-list scanner nothing from before
the open is recorded.

**"One over is cut into a string of short calls."** A simplex DMR handheld
is off for 32.5 ms of every 60 ms frame; judged chunk by chunk, every gap
started hangtime and no 27.5 ms burst could outlast the 50 ms debounce to
stop it, so one over became 1.5–2.5 s WAVs. A DMR channel now bridges gaps
shorter than `dmrBurstBridge` (100 ms); analog keeps the plain debounce
(Part 10).

**"The tone gate opens on the wrong tone."** Before the #1184 fixes the
Goertzel rounded to the nearest bin, so a 162.2 Hz gate stayed shut and
159.8 opened (Part 5). Two cases remain by design: 150.0 and 151.4 Hz are
1.4 Hz apart and open together, and a DCS code opens on its aliases — a
normal 023 gate still opens on 047I, the same bits (Part 6). `conv: DCS
gate opened` logs `nrz_inverted`.

**"AM tunes best 3 kHz off."** It was not a frequency shift. In the
capture tuned to 123.453 MHz the carrier sat −3315 Hz away, so 123.450 was
right; what the offset exposed was an AM channel filter centred on the
*tuned* frequency, cutting the far sideband. `amCarrierAFC` now mixes the
carrier to DC first (Part 7). Its cousin on FM is a whistle at 4× the
carrier offset on a strong signal — a clipped ADC, and `conv: front end
overloaded` says so (Part 4).

**"A call on a silent channel that never ends."** A dwell whose IQ stream
*stops* without closing cannot end on hangtime, which counts chunks that
never arrive. `StreamStallTimeout` (3 s) ends it with `conv: IQ stream
stalled during dwell — ending call and re-opening the stream` (Part 10).

## The twin ledger

Many confusing reports in this series were one of a pair behaving
differently from its sibling. Ask which side you are on:

| Twin pair | The drift, when it happened |
|---|---|
| level ↔ statistic | a level opens on noise under an AGC; a statistic reads digital silence as perfectly quiet — so FM ANDs them (Part 3) |
| span ↔ channel | squelch, tone detector and eight decoders each measured the whole SDR span before a ±8 kHz filter was put in front (Parts 2, 5, 12) |
| scanner gate ↔ voice chain | the scanner opened a `ysf` call the chain decoded nothing from and logged nothing about; the chain's ended line now carries `sync_hits` and `adc_rail_pct` (Part 9) |
| scan-list tap ↔ dedicated SDR | the scanner's data taps were filtered while the dedicated-SDR decoders still ran raw — and raw scored 0 of 2 bursts where filtered scored 2 of 2 on the FleetSync slices (Parts 11, 12) |
| fixture ↔ air | the CTCSS threshold was calibrated at 48 kHz and fed 2.4 MS/s; the DCS test synthesiser shared the detector's invented codeword (Parts 5, 6) |
| conventional lockout ↔ trunked hold | one persists by frequency, the other is session-only, and neither path's controls reach the other's calls (Part 13) |

Two of these rows have their own postmortems in Rebuilt From the Air:
[Part 3]({{ '/blog/solution-postmortem/rebuilt-from-the-air-03-ctcss-dcs-tone-gates/' | relative_url }})
(the tone gates) and
[Part 13]({{ '/blog/solution-postmortem/rebuilt-from-the-air-13-read-the-capture-first/' | relative_url }})
(the "YSF" capture).

### How the gate rule shaped the Go code

- **One interface for every second gate.** CTCSS, DCS and the NAC gate are
  `toneDetector`s; the call gates add `gateName` and `minDwell` as a
  `callGate`, which the scan window type-asserts.
- **Thresholds are ratios, calibrated at construction.** The noise
  squelch's reference is measured by pushing fixed-seed noise through the
  real filter, per rate.
- **Front ends normalise the rate.** Every detector decimates to ~48 kHz
  first, so a constant means the same thing at any SDR rate.
- **Gates log what they decided on.** `conv: squelch opened` carries the
  level, threshold and quieting; `conv: call gate opened` names the gate.

## What's still open

An honest playbook ends with its open items, each at a named gate. First
what is closed: **CTCSS and
DCS** are on-air verified (24 Sep), **AM** is on-air verified (29 Sep),
**MDC1200** is on-air verified through the scanner-channel path, and the
**NXDN** call gate is pinned on the reporter's real-air slices. The rest:

- **Noise squelch on a PlutoSky** — measured on real receiver noise and
  real FM handhelds, but not run on the board that prompted it; whether
  `gain: auto` was still set on the reporter's 18 Nov run is not known.
- **A `p25` channel without `nac`** — still opens on level alone;
  `nac: "F7E"` (any NAC, P25 only) is the remedy.
- **Wide FM has no mode** — a broadcast station reads as noise and never
  opens; `noise_squelch: false` leaves the level as that channel's only
  test.
- **DMR call gate** — synthetic-verified only. Part 10's burst bridge and
  stream-reopen fix are on-air-gated, and why both dongles stalled at the
  same instant in the FleetSync coexistence run is not root-caused.
- **System Fusion and D-STAR** — no real capture of either yet. The YSF
  gate is pinned against a real-air slice it must *not* open on; D-STAR is
  rebuilt to the MMDVM firmware's air format and unverified on real air.
- **ACARS** — verified against acarsdec in both directions; the live
  scanner run is pending, and a burst also opens an ordinary AM call.
- **Offset tuning** — on-air-gated. `dc_avoid`'s default offset of
  `sample_rate/4` on control and voice SDRs has the same clipping exposure
  and was not changed.
- **The shared narrowband front end** — synthetic-verified; nothing from
  #1316 has run on the reporter's dongles.

Every item is blocked on evidence, not effort — the sentence the
[P25]({{ '/blog/deep-dives/p25-end-to-end-14-playbook/' | relative_url }})
and
[DMR]({{ '/blog/deep-dives/dmr-end-to-end-14-playbook/' | relative_url }})
playbooks closed on, because it is the project's operating principle.

## Where to go from here

The [series index]({{ '/blog/series/squelch/' | relative_url }}) lists all
fourteen parts. Two series ran alongside this one:
[The Operator's Cookbook, Volume 2]({{ '/blog/series/operator-cookbook-volume-2/' | relative_url }})
turns these gates into recipes — its
[airband rig]({{ '/blog/tutorials/cookbook-volume-2-01-airband-rig/' | relative_url }})
and
[fleet signalling]({{ '/blog/tutorials/cookbook-volume-2-04-fleet-signalling/' | relative_url }})
parts run the AM, tone and data rows of the map — and
[Rebuilt From the Air]({{ '/blog/series/rebuilt-from-the-air/' | relative_url }})
tells the investigations behind the decoders the call gates listen with.
The rule this series kept returning to comes from
[The Analog Edge]({{ '/blog/series/analog-edge/' | relative_url }}):
[Part 13]({{ '/blog/tutorials/analog-edge-13-coherence-not-dbfs/' | relative_url }})
states it, and its
[field checklist]({{ '/blog/tutorials/analog-edge-14-field-checklist/' | relative_url }})
is the RF half of every "squelch or antenna?" question. The first
cookbook's
[analog FM recipe]({{ '/blog/tutorials/operator-cookbook-06-analog-fm-tone-out/' | relative_url }})
is where a scan list starts; the
[squelch]({{ '/reference/squelch/' | relative_url }}),
[CTCSS]({{ '/reference/ctcss/' | relative_url }}) and
[DCS]({{ '/reference/dcs/' | relative_url }}) references are the glossary.

## FAQ

**Why does my scan-list channel open when nobody is transmitting?**
Either the level is not screening it or a gate is missing. On analog FM,
read `quieting_db` in `GET /api/v1/scanner`: about 0 means noise, and the
noise squelch should be keeping the channel shut unless it was turned off.
On a `p25` channel, set `nac`; without it any carrier above `squelch_dbfs`
opens the channel.

**How do I tell which gate opened or refused a channel?**
An analog or P25 channel logs `conv: squelch opened` with the level, its
threshold and, on FM, the quieting figure; the NAC and DCS gates also log
the NAC or code they matched. A channel with a call gate logs `conv: call
gate opened` with the gate's name. A channel held shut by the noise
squelch alone is named once in `conv: channel is above squelch_dbfs with
no carrier on it`.

**Which GopherTrunk squelch gates are verified on air?**
CTCSS and DCS (24 Sep), AM (29 Sep) and MDC1200 through a scan-list
channel are on-air verified; the NXDN call gate is pinned on real-air
slices. The DMR call gate is synthetic-verified only, System Fusion and
D-STAR have no real capture yet, and the noise squelch has not been run on
the PlutoSky that prompted it.

**Should I raise squelch_dbfs to stop false opens?**
Rarely. `squelch_dbfs` is an absolute level that moves with gain, and
under `gain: auto` it screens nothing. Leave the noise squelch on, give
digital channels their gate, and treat the level as the cheap first test
it is.

## Series navigation

**Part 14 of 14** · ←
[Part 13: Operator Controls — Priority, Lockouts and Loudness]({{ '/blog/deep-dives/squelch-13-operator-controls/' | relative_url }})
· [Back to the series index]({{ '/blog/series/squelch/' | relative_url }})
