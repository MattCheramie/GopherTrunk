---
title: "P25 End to End, Part 14: The P25 Playbook"
description: "The finale — the whole P25 stack folded into one layer map from antenna to WAV: which symptom points at which layer, the instrument that confirms it, the failure signatures operators actually report, the twin-path ledger, and the honest list of what's still open."
category: deep-dives
keywords: p25 troubleshooting guide, p25 decoder playbook, p25 locks but no voice, p25 grants wrong frequency, p25 failure diagnosis, p25 layer map, p25 twin paths, trunking scanner debugging, gophertrunk p25
tags: [p25-end-to-end, p25, playbook, troubleshooting, methodology, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "P25 End to End"
series_part: 14
---

*Part 14 — the last — of **P25 End to End**, a 14-part deep dive that follows
North America's dominant trunking protocol through GopherTrunk — from a raw
C4FM carrier to recorded, named, multi-site voice. Thirteen parts ago we
started with four frequency levels on a deviation ladder; since then we've
climbed through frame sync, TSBKs, multi-block trunking, band plans, two
physical layers, two phases, voice, encryption, roaming, wideband, and the
testing discipline holding it all together. This closing part compresses the
climb into what you actually need at the bench: one layer map from symptom to
part, the failure signatures operators actually report, the ledger of twin
paths where fixes drift apart, and an honest list of what remains open.*

> **TL;DR:** Debugging P25 in GopherTrunk is deciding **which layer** a
> symptom lives in, then reaching for that layer's instrument. Never locks →
> demod/sync (eye diagram, sync margin — Parts 1–2). Locks but no activity →
> TSBK/MBT (decode counters — Parts 3–4). Grants tune to nothing → band plan
> arithmetic (Part 5). Simulcast garble → the C4FM/CQPSK twin (Part 6,
> `p25_phase1_demod_mode`). Silent recordings → encryption policy (Part 9),
> not the demod. Short 4-frame calls on a marginal site → the weak-signal gap
> (Part 12), where the fix is parked behind a capture — still the
> highest-leverage contribution this series can name. The recurring trap at
> every layer is the **twin pair**: C4FM/CQPSK, Phase 1/Phase 2,
> single-channel/wideband, live/replay — the
> [Two Pipelines lesson]({{ '/blog/solution-postmortem/from-the-issue-tracker-22-two-pipelines/' | relative_url }}),
> which this playbook bakes into its checklist.

**Key takeaways**

- **Locate the layer before touching anything.** Every layer has a distinct
  symptom and a distinct instrument; the costliest debugging mistake in this
  series' history was working the wrong layer with great skill.
- **Ask "which twin am I on?" early.** Half the confusing P25 reports —
  #764/#771, #882, #935 — were one twin path behaving differently from its
  sibling. A fix or a config that touched only one twin is the first
  hypothesis, not the last.
- **Absence of data is data.** A missing WACN, one lonely neighbour site, or
  a grant with no voice each points at a specific decoder gap this series
  covered — and two of them were real shipped bugs.
- **The open items are gated on evidence, not effort.** The C4FM
  equalizer/soft-FEC port waits on a weak voice capture; the levers, the
  harness and the ask are all merged and idle.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| "Never locks" triage | demod + sync layer first, RF second | [Part 1]({{ '/blog/deep-dives/p25-end-to-end-01-c4fm-carrier/' | relative_url }}), [Part 2]({{ '/blog/deep-dives/p25-end-to-end-02-sync-nid-lock/' | relative_url }}) |
| "Locks, no voice" triage | encryption policy → demod twin → band plan | [Part 9]({{ '/blog/deep-dives/p25-end-to-end-09-encryption/' | relative_url }}), [Part 6]({{ '/blog/deep-dives/p25-end-to-end-06-cqpsk-lsm/' | relative_url }}), [Part 5]({{ '/blog/deep-dives/p25-end-to-end-05-channels-band-plans/' | relative_url }}) |
| Missing WACN / neighbours | AMBT-form broadcasts (once dropped as PDU noise) | [Part 4]({{ '/blog/deep-dives/p25-end-to-end-04-mbt-ambt/' | relative_url }}), `phase1/mbt.go` |
| Weak-signal 4-frame calls | the missing levers + the capture ask | [Part 12]({{ '/blog/deep-dives/p25-end-to-end-12-weak-signal-gap/' | relative_url }}), `samples/p25/README.md` |
| Twin-path checklist | which sibling path did the fix/config touch? | [Part 11]({{ '/blog/deep-dives/p25-end-to-end-11-wideband/' | relative_url }}), the ledger below |
| Proof discipline | pyramid: vectors → synthetics → captures → air | [Part 13]({{ '/blog/deep-dives/p25-end-to-end-13-testing-p25/' | relative_url }}) |

## In this post

- **The layer map** — the whole stack, one table, symptom → instrument → part.
- **Failure signatures** — the reports operators actually file, decoded.
- **The twin ledger** — every pair where fixes have drifted, with receipts.
- **What's still open** — the honest punch list, each item at a named gate.

## The layer map

The series in one table. When a P25 system misbehaves, find the row whose
symptom matches, use its instrument, and only then open the code — the
instruments exist so the layer diagnosis is a measurement, not a guess.

| Layer | Symptom when broken | Instrument | Part |
|---|---|---|---|
| C4FM demod | never locks; dibit stream random | eye diagram scope, BER sweep, EVM/SNR | [1]({{ '/blog/deep-dives/p25-end-to-end-01-c4fm-carrier/' | relative_url }}) |
| FSW + NID | no lock, or NAC flapping | FSW sync margin, NID trusted/failed counters | [2]({{ '/blog/deep-dives/p25-end-to-end-02-sync-nid-lock/' | relative_url }}) |
| TSBK | locked but no grants; mislabelled opcodes | TSBK decoded / trellis-failed / CRC-failed | [3]({{ '/blog/deep-dives/p25-end-to-end-03-tsbk-workhorse/' | relative_url }}) |
| MBT / AMBT | missing WACN, one neighbour site | `MBT data CRC failed` line + Viterbi metric | [4]({{ '/blog/deep-dives/p25-end-to-end-04-mbt-ambt/' | relative_url }}) |
| Band plan | grants tune to static | (id, channel) → Hz worked by hand | [5]({{ '/blog/deep-dives/p25-end-to-end-05-channels-band-plans/' | relative_url }}) |
| CQPSK / LSM twin | simulcast garble through the C4FM chain | constellation scope, CMA error | [6]({{ '/blog/deep-dives/p25-end-to-end-06-cqpsk-lsm/' | relative_url }}) |
| Phase 2 TDMA | one slot missing; `trellis=0` in the log | MAC PDU counters, chain-start log line | [7]({{ '/blog/deep-dives/p25-end-to-end-07-phase2-tdma/' | relative_url }}) |
| IMBE voice | short, scratchy or missing audio | LDU yield, uncorrectable-LDU count | [8]({{ '/blog/deep-dives/p25-end-to-end-08-imbe-voice/' | relative_url }}) |
| Encryption | calls recorded silent or skipped | ALGID/KID metadata in the call log | [9]({{ '/blog/deep-dives/p25-end-to-end-09-encryption/' | relative_url }}) |
| Sites / roaming | camped on a fading site | adjacency table, hunt/roam logs | [10]({{ '/blog/deep-dives/p25-end-to-end-10-sites-roaming/' | relative_url }}) |
| Wideband | one channel decodes, its twin doesn't | per-tap counters vs single-channel replay | [11]({{ '/blog/deep-dives/p25-end-to-end-11-wideband/' | relative_url }}) |
| Weak signal | locks, 4-frame calls | `TestReplayP25RealCaptureMetrics` baseline | [12]({{ '/blog/deep-dives/p25-end-to-end-12-weak-signal-gap/' | relative_url }}) |

<figure class="lab-figure">
<svg viewBox="0 0 680 240" width="680" height="240" role="img" aria-label="The P25 decode stack from antenna to WAV, each block annotated with the series part that covers it: antenna and SDR feed the DDC channelizer from Part 11, then the C4FM or CQPSK demodulator from Parts 1 and 6, then frame sync and NID from Part 2, then TSBK and AMBT control decode from Parts 3 and 4, then the band plan lookup from Part 5 that tunes a voice grant, then IMBE or Phase 2 voice decode from Parts 8 and 7, then encryption policy from Part 9, ending in a recorded WAV. Below the chain, three spanning braces mark sites and roaming from Part 10 across the control section, the weak-signal gap from Part 12 across the voice section, and testing from Part 13 across everything.">
  <rect x="6" y="50" width="66" height="40" rx="6" fill="none" stroke="currentColor"/>
  <text x="39" y="67" text-anchor="middle" fill="currentColor" font-size="9">antenna</text>
  <text x="39" y="80" text-anchor="middle" fill="currentColor" font-size="9">+ SDR</text>
  <line x1="72" y1="70" x2="84" y2="70" stroke="currentColor"/>
  <rect x="84" y="50" width="58" height="40" rx="6" fill="none" stroke="currentColor"/>
  <text x="113" y="67" text-anchor="middle" fill="currentColor" font-size="9">DDC</text>
  <text x="113" y="80" text-anchor="middle" fill="var(--accent)" font-size="8">P11</text>
  <line x1="142" y1="70" x2="154" y2="70" stroke="currentColor"/>
  <rect x="154" y="50" width="76" height="40" rx="6" fill="none" stroke="currentColor"/>
  <text x="192" y="67" text-anchor="middle" fill="currentColor" font-size="9">C4FM / CQPSK</text>
  <text x="192" y="80" text-anchor="middle" fill="var(--accent)" font-size="8">P1 · P6</text>
  <line x1="230" y1="70" x2="242" y2="70" stroke="currentColor"/>
  <rect x="242" y="50" width="70" height="40" rx="6" fill="none" stroke="currentColor"/>
  <text x="277" y="67" text-anchor="middle" fill="currentColor" font-size="9">FSW + NID</text>
  <text x="277" y="80" text-anchor="middle" fill="var(--accent)" font-size="8">P2</text>
  <line x1="312" y1="70" x2="324" y2="70" stroke="currentColor"/>
  <rect x="324" y="50" width="82" height="40" rx="6" fill="none" stroke="currentColor"/>
  <text x="365" y="67" text-anchor="middle" fill="currentColor" font-size="9">TSBK / AMBT</text>
  <text x="365" y="80" text-anchor="middle" fill="var(--accent)" font-size="8">P3 · P4</text>
  <line x1="406" y1="70" x2="418" y2="70" stroke="currentColor"/>
  <rect x="418" y="50" width="66" height="40" rx="6" fill="none" stroke="currentColor"/>
  <text x="451" y="67" text-anchor="middle" fill="currentColor" font-size="9">band plan</text>
  <text x="451" y="80" text-anchor="middle" fill="var(--accent)" font-size="8">P5</text>
  <line x1="484" y1="70" x2="496" y2="70" stroke="currentColor"/>
  <rect x="496" y="50" width="86" height="40" rx="6" fill="none" stroke="currentColor"/>
  <text x="539" y="67" text-anchor="middle" fill="currentColor" font-size="9">voice: IMBE / P2</text>
  <text x="539" y="80" text-anchor="middle" fill="var(--accent)" font-size="8">P8 · P7</text>
  <line x1="582" y1="70" x2="594" y2="70" stroke="currentColor"/>
  <rect x="594" y="50" width="80" height="40" rx="6" fill="none" stroke="currentColor"/>
  <text x="634" y="67" text-anchor="middle" fill="currentColor" font-size="9">policy → WAV</text>
  <text x="634" y="80" text-anchor="middle" fill="var(--accent)" font-size="8">P9</text>
  <path d="M 242 108 L 242 120 L 484 120 L 484 108" fill="none" stroke="var(--fg-muted)"/>
  <text x="363" y="136" text-anchor="middle" fill="var(--fg-muted)" font-size="9">sites, WACN &amp; roaming — P10</text>
  <path d="M 154 152 L 154 164 L 674 164 L 674 152" fill="none" stroke="var(--fg-muted)"/>
  <text x="414" y="180" text-anchor="middle" fill="var(--fg-muted)" font-size="9">the weak-signal gap: equalizer + soft FEC missing on default C4FM voice — P12</text>
  <path d="M 6 196 L 6 208 L 674 208 L 674 196" fill="none" stroke="var(--accent)"/>
  <text x="340" y="226" text-anchor="middle" fill="var(--accent)" font-size="9">testing: literal vectors → synthetics → captures → on-air — P13</text>
</svg>
<figcaption>Antenna to WAV with part numbers attached — and the three spans that cut across every block: site topology, the weak-signal gap, and the testing pyramid.</figcaption>
</figure>

## Failure signatures

The layer map is for methodical triage; these are the shortcuts — the
symptoms operators actually report, matched to their usual cause.

**"It locks, but no voice ever records."** Three suspects, in order of how
often they're guilty. First, **encryption policy**: check the call log's
ALGID/KID metadata before blaming the DSP — an encrypted system faithfully
decoded still yields nothing listenable, and the policy knobs decide whether
it's skipped or logged ([Part 9]({{ '/blog/deep-dives/p25-end-to-end-09-encryption/' | relative_url }})).
Second, the **demod twin**: an LSM/simulcast site pushed through the FM
discriminator produces near-random dibits (issue #275's original symptom) —
set the per-channel `p25_phase1_demod_mode` (issue #935). Third, the **band
plan**: the grant may be tuning somewhere the call isn't.

**"Grants tune to nothing."** Band-plan arithmetic, almost always
([Part 5]({{ '/blog/deep-dives/p25-end-to-end-05-channels-band-plans/' | relative_url }})).
Work one grant by hand: base + spacing × channel, minus the identifier
table's tx offset where applicable — and remember the AMBT rule that an
*explicit* uplink channel already encodes the uplink frequency, so it takes
**no** tx offset. A wrong IDEN_UP row moves every voice channel at once,
which is the tell distinguishing it from a per-channel RF problem.

**"Only one neighbour site, and no WACN."** The signature of a system that
broadcasts Network Status and Adjacent Status only in **AMBT** form. Old
GopherTrunk logged each one as `non-control DUID duid=PDU` and dropped it;
since `mbt.go` landed ([Part 4]({{ '/blog/deep-dives/p25-end-to-end-04-mbt-ambt/' | relative_url }}))
those decode. On this symptom today, check the version first. And a `MBT
data CRC failed` line whose identity fields look perfectly decoded is **two
frames, not a contradiction** — the header carries its own CRC and decoded
clean; a data block failed its CRC-32 on RF residuals. The broadcast
repeats; don't chase a parser bug from that log pair.

**"Wideband decodes the control channel, but voice is wrong."** The twin
ledger's territory: the wideband path once silently dropped the Phase 2 FEC
options — `trellis=0` in the chain-start log line is the tell — and stamped
no demod mode onto its grants, so voice always ran the C4FM chain regardless
of config (issues #882, #935). Both fixed; both are the pattern to suspect
when *one pipeline* of a pair misbehaves.

**"It locks and grants, but marginal calls yield a few frames."** The
weak-signal gap ([Part 12]({{ '/blog/deep-dives/p25-end-to-end-12-weak-signal-gap/' | relative_url }})).
No config key fixes this today; the levers are diagnosed and parked behind a
weak C4FM voice capture. If your site produces this symptom, you hold the
unblock.

## The twin ledger

The series' running thread, settled as a checklist. Every entry is a real
divergence with a receipt — ask which side of each pair you're on before
concluding anything:

| Twin pair | The drift, when it happened |
|---|---|
| C4FM ↔ CQPSK | the blind FSE equalizer exists only on the CQPSK path; default C4FM voice has no equalizer at all (Parts 6, 12) |
| single-channel ↔ wideband DDC | the #764 "fix" landed on one down-converter and missed the #771 replay symptom in the other (Part 11) |
| Phase 1 ↔ Phase 2 | the wideband pipeline forwarded Phase 1 options but dropped Phase 2's FEC knobs — `trellis=0` (#882) |
| control ↔ voice chain | soft-decision LLRs feed the TSBK trellis but never the IMBE FEC; the DC blocker is voice-only by design (Part 12) |
| live ↔ replay | prevented drift: replay builds its DDC from the same exported `DDCTargetForProtocol` the live path uses (Part 1) |

The general lesson graduated into
[its own postmortem]({{ '/blog/solution-postmortem/from-the-issue-tracker-22-two-pipelines/' | relative_url }}):
when two code paths implement one concept, every fix needs an explicit
answer to "and the twin?" — in the PR, not in the next bug report.

## What's still open

An honest playbook ends with its open items, each parked at a named gate:

- **The C4FM voice equalizer + soft IMBE FEC** — diagnosed, scoped, proven
  next door; gated on a weak C4FM **voice** capture landing in
  `samples/p25/` (Part 12). The baseline harness runs today.
- **The decision-directed AFC** (`EnableDecisionDirectedAFC`) — built,
  gated, and **off by default**: on the issue #402 capture it stably held a
  wrong offset that CoarseAFC-alone recovered. It waits on the eye-skew root
  cause and a real lock signal to gate its handoff.
- **The adaptive C4FM slicer** (`EnableAdaptiveC4FMSlicer`) — also off by
  default, honestly: on the same #402 capture the fixed thresholds beat
  every adaptive variant, because the eye asymmetry is an RF-domain effect
  the slicer cannot fix. Kept for A/B experimentation.
- **On-air breadth** — every decoder here is pinned to references and
  captures from a handful of systems; each new system an operator runs is a
  fresh verification, and `samples/p25/` is how its lessons persist.

Every item is blocked on evidence, not effort — the same sentence the
[weak-signal playbook]({{ '/blog/deep-dives/weak-signal-engineering-14-playbook/' | relative_url }})
closed on, because it is the project's actual operating principle.

## Where to go from here

The natural next reads depend on which thread of this series hooked you.
For the same protocol-length treatment of the *other* great trunking family
— phase transitions instead of amplitudes, and the carrier most of this
project's hardest lessons came from — cross to
[TETRA End to End]({{ '/blog/series/tetra-end-to-end/' | relative_url }}).
For the method behind Parts 12–13 in full — the four regimes, the levers,
the metrics that lie —
[Weak-Signal Engineering]({{ '/blog/series/weak-signal-engineering/' | relative_url }})
is the theory course. For how a wire format goes from a standards PDF to
shipped, pinned Go —
[From Spec to Shipping]({{ '/blog/series/from-spec-to-shipping/' | relative_url }}).
And to put all of this on your own desk, the
[Operator's Cookbook]({{ '/blog/series/operator-cookbook/' | relative_url }})
starts with a
[$40 P25 rig]({{ '/blog/tutorials/operator-cookbook-01-forty-dollar-p25-rig/' | relative_url }})
that runs every layer this series just mapped.

## FAQ

**My P25 system won't decode — where do I start?**
At the bottom of the layer map, with instruments: does the eye diagram look
like four levels? Does the FSW sync margin show hits? Is NID trusted
climbing? A "no" at any row names your layer, and the linked part covers it.
Working top-down from "no audio" wastes hours in layers that are fine.

**What's the single most common cause of "locks but no voice"?**
Encryption, followed by the demod-mode twin. Check ALGID/KID metadata in the
call log before touching DSP settings — a correctly-decoded encrypted call
and a broken voice chain look identical from the speaker, and only one of
them is a bug.

**Is Phase 2 harder to receive than Phase 1?**
Different, not strictly harder: H-DQPSK demands the differential machinery
TETRA pioneered (and shares its `PiOver4DQPSK` core), but Phase 2 also wires
an equalizer where default Phase 1 C4FM doesn't. The practical Phase 2 traps
in this series were configuration plumbing (#882), not modulation.

**How can I contribute if I can't write DSP code?**
Captures — this series has asked three times and means it. A weak C4FM
voice call per `samples/p25/README.md` unblocks the biggest open item; any
control-channel capture with a metadata sidecar becomes a permanent
regression; and a report that says *which layer's instrument* showed the
anomaly is halfway to a fix on arrival.

**Does this playbook transfer to DMR or NXDN?**
Largely. The 4800-baud C4FM-family physical layer is shared machinery, the
layer-map method is protocol-agnostic, and the twin-pair discipline applies
anywhere two pipelines implement one concept. The protocol-specific rows —
band plans, AMBT, ESS — have DMR/NXDN analogues covered in the
[Protocol Decoders survey]({{ '/blog/deep-dives/protocol-decoders-05-dmr-tier-2-3/' | relative_url }}).

## Series navigation

**Part 14 of 14** · ←
[Part 13: Testing P25 Without a Tower]({{ '/blog/deep-dives/p25-end-to-end-13-testing-p25/' | relative_url }})
· [Back to the series index]({{ '/blog/series/p25-end-to-end/' | relative_url }})
