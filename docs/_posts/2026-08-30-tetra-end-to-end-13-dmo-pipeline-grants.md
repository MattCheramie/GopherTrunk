---
title: "TETRA End to End, Part 13: DMO III — A Production Pipeline: Grid Votes, Grants & Noise"
description: "Wiring DMO into the daemon as a first-class protocol — streaming burst extraction, sticky locks, edge-triggered grants — and the first on-air run's humbling lesson: a loose correlator false-alarms eighteen times a second, and only a learned slot-grid vote separates a real transmission from an idle channel."
category: deep-dives
keywords: tetra dmo pipeline, dmslotgrid residue voting, false alarm rate correlator, edge triggered grant, sticky cc lock, dmo voice chain, streaming burst extractor, dnb qualification, gophertrunk tetra
tags: [tetra-end-to-end, tetra, dmo, daemon, statistics, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "TETRA End to End"
series_part: 13
---

*Part 13 of **TETRA End to End**, a 14-part deep dive into how GopherTrunk turns
one real 25 kHz TETRA carrier into clear recorded voice.
[Part 12]({{ '/blog/deep-dives/tetra-end-to-end-12-dmo-descramble-colour/' | relative_url }})
got clear DMO voice decoding offline — colour recovered, speech frames out, PCM
rendered. This part promotes all of it from a replay harness to the production
daemon: a real protocol name in the config, a real pipeline behind the scanner,
a real voice chain into the recorder. And it includes the most instructive bug
of the whole DMO arc — the first live run granted a call on a **silent
channel** 230 milliseconds after startup, then ignored the operator's actual
transmission entirely. The root cause was not a typo. It was a statistics
lesson the offline path never had to learn.*

> **TL;DR:** DMO is now a first-class daemon protocol: **`ProtocolTETRADMO`**
> (`tetra-dmo`/`dmo` in `internal/trunking/site.go`, 144 kHz DDC target, an
> entry in the `factories` map), decoded by **`newTETRADMOPipeline`** driving a
> streaming **`tetra.DMStreamExtractor`**, with a **sticky** `KindCCLocked`
> (never `cc.lost` on inter-PTT silence) and an **edge-triggered** `tetra-dmo`
> grant into **`composer.runTETRADMOVoiceChain`**. The first on-air run exposed
> that a raw DNB detection is *not* evidence of traffic: an 11-dibit training
> match at tolerance 2 under 8 filters fires by chance **~18×/s** on an idle
> channel (529 of 4¹¹ sequences match), so **`tetra.DMSlotGrid`** now votes DNB
> leads onto the 255-dibit slot residue — one radio on one clock concentrates
> on one residue; noise is uniform over all 255 — and only **qualified** DNBs
> count toward the grant, the colour recovery, or the drought. The grant also
> now requires the lock it was always named after, and the re-arm drought is
> evaluated from `Process`, not just from bursts.

**Key takeaways**

- **A loose detector needs a qualifier, not a tighter threshold.** Tolerance 2
  is right for catching marginal real bursts; the fix isn't tolerance 1, it's
  a second, orthogonal test — timing structure — that noise cannot fake.
- **The operator's own log carried the proof.** `dnb_total` climbing
  1076→4541 in 185 s is 18.7/s — the predicted false-alarm rate to the first
  decimal — while `dsb_schs_crc` sat frozen at 46. Numbers first, then code.
- **Learn constants; don't derive them from the spec when a wrong value fails
  silently.** The grid's traffic residue is voted from the stream, never
  computed from a DSB→DNB offset — a wrong hardcoded offset would quietly stop
  DMO granting forever, which is worse than over-granting.
- **A brute force that starves its own input is a self-inflicted outage.** The
  voice chain's uncapped 64-colour recovery was ≈450k Viterbi decodes per
  call — enough to drop 5904 chunks of the very IQ it was decoding.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| Protocol registration | enum, `"tetra-dmo"`, parse, validate | `internal/trunking/site.go` (`ProtocolTETRADMO`) |
| Channel rate | DMO shares TETRA's 144 kHz DDC target | `internal/scanner/ccdecoder/ddc.go` (`ddcTargetForProtocol`) |
| Control pipeline | lock, colour, grant off the dibit stream | `internal/scanner/ccdecoder/pipelines_dmo.go` (`newTETRADMOPipeline`) |
| Streaming extractor | bounded window, dedup, lead-ordered emit | `internal/radio/tetra/dmo_stream.go` (`DMStreamExtractor`) |
| Noise gate | residue voting on the 255-dibit slot grid | `internal/radio/tetra/dmo_grid.go` (`DMSlotGrid`) |
| Grant policy | lock required, 4 qualified DNBs, 3 s re-arm | `pipelines_dmo.go` (`maybeGrant`, `dmoGrantMinDNB`, `dmoGrantRearm`) |
| Voice chain | buffer → colour → retroactive decode → ACELP | `internal/voice/composer/tetra_dmo_voice.go` (`runTETRADMOVoiceChain`) |
| Failing-first pins | idle-channel, lock-gate, re-arm regressions | `pipelines_dmo_test.go`, `dmo_grid_test.go` |

## In this post

- **Registering a protocol** — the three touchpoints that make `tetra-dmo` real.
- **A pipeline for a channel that's usually silent** — sticky locks, streaming extraction.
- **The noise-grant post-mortem** — 18 false DNBs a second, and four compounding bugs.
- **The slot-grid vote** — residue statistics that noise can't fake.
- **The voice chain's own lessons** — cost caps and honest logging.

## Registering a protocol

Making `protocol: tetra-dmo` mean something takes three touchpoints.
`internal/trunking/site.go` adds `ProtocolTETRADMO` to the enum, `String()`,
`ParseProtocol` (accepting `tetra-dmo`, `tetra_dmo`, `tetradmo`, and plain
`dmo`), and `Validate`. `ccdecoder/ddc.go` teaches `ddcTargetForProtocol` that
DMO, like TMO TETRA, needs the 144 kHz channel rate. And the `factories` map in
`pipelines.go` routes the protocol to `newTETRADMOPipeline`. That last hop is
the one that fixes the operator-visible symptom from Part 11: before it, a DMO
capture fell through to the TMO control-channel path and produced the
`sch_pdus=0` / `sync gap` / `dsp resync` churn of a state machine waiting for a
downlink that will never come.

## A pipeline for a channel that's usually silent

`newTETRADMOPipeline` reuses the TMO receiver knobs wholesale — Gardner clock,
AFC, channel filter, `SoftSink`, and the blind CMA equalizer, which for DMO is
*required*, not optional (Part 11's 6→64 SCH/S lift). But it cannot reuse the
TMO decode loop, so the receiver feeds `tetra.DMStreamExtractor` — a bounded
sliding-window adapter over the stateless `ExtractDMBurstsSoft` from Part 11
that re-scans a 384-dibit retained tail each pass and emits every DSB/DNB
exactly once, in lead order, deduped by absolute lead index. Offline and live
paths therefore share the framing code bit-for-bit, which is what
`TestDMStreamExtractorMatchesWholeSlice` (chunk-invariance) pins.

Lifecycle is where DMO diverges hardest. The lock — published on the first
CRC-valid SCH/S with a parseable SYNC PDU — is **sticky**: the pipeline never
publishes `cc.lost`, because inter-PTT silence is the *normal state* of a DMO
channel, and re-hunting a camped frequency every quiet gap is exactly the churn
the TMO watchdog exists to cause. And with no control channel to announce
calls, grants are **inferred from traffic**: a rising edge of DNB activity
publishes one `tetra-dmo` grant (GroupID 0 — DMO carries no talkgroup), re-armed
only after a 3 s traffic drought. Which brings us to the run where all of that
logic met the air.

## The noise-grant post-mortem

First live run: the daemon granted and opened a recording **~230 ms after
startup** on a silent channel — then never granted again. The operator's real
10-second PTT locked cleanly (`dsb_schs_crc=46/54`) and produced nothing. Four
bugs compounded, but the root is worth doing the arithmetic in public. The DNB
correlator matches an 11-dibit training sequence at Hamming tolerance 2, under
8 matched filters (2 sequences × 4 rotations). The number of 11-dibit sequences
within distance 2 of a pattern is:

```go
// internal/radio/tetra/dmo_grid.go (shape) — the arithmetic, in the comment
//   Σ_{k≤2} C(11,k)·3^k = 1 + 33 + 495 = 529
// so p ≈ 529/4^11 ≈ 1.26e-4 per position per filter, ≈ 1.0e-3 across the
// eight filters, and at 18 000 dibit/s that is ≈ 18 spurious DNB
// "detections" per second off an idle channel.
```

**Eighteen false DNBs per second, on noise.** The operator's log confirms it to
the first decimal: `dnb_total` climbed 1076→4541 over 185 s — 18.7/s — while
`dsb_schs_crc` sat frozen. Against that rate: `dmoGrantMinDNB=4` tripped in
~0.22 s of startup noise; `maybeGrant` never actually checked `p.locked`
(despite its counter being *named* `dnbSinceLock`); the 3 s re-arm drought
could never elapse against a 55 ms mean inter-arrival; and the drought check
lived only inside the DNB branch, so on a truly silent channel it could not
run at all — the grant latched for the life of the daemon. A detector loose
enough to catch marginal real bursts is, by the same coin, a noise generator.
The fix had to make noise and traffic *statistically distinguishable*.

## The slot-grid vote

The discriminator is timing structure. A real transmission comes from **one
radio on one clock**, so every burst it emits lands on the 255-dibit timeslot
grid — all its DNB leads share a single residue mod 255. Noise hits are uniform
over all 255 residues. `tetra.DMSlotGrid` votes recent leads onto that ring:

```go
// internal/scanner/ccdecoder/pipelines_dmo.go (shape) — onBurst, DNB branch
p.dnbTotal++
p.maybeRearmGrant(now)
if !p.grid.Observe(b.Lead) {
    return // correlator false alarm: counts nowhere below
}
p.dnbQualified++
p.dnbSinceLock++
p.recoverColour(b)
p.maybeGrant() // requires p.locked AND dmoGrantMinDNB qualified bursts
```

The sizing is Poisson arithmetic, written into the constants' comments: a 1 s
window (`dmGridWindow=18000` dibits) holds ~18 noise leads spread over 255
residues, so the expected count agreeing with any residue within ±1 is ~0.21 —
requiring `dmGridMinTrain=6` in agreement makes a false latch a ~1.4×10⁻⁶
event per detection, about **once per 11 hours**, while a real train at ~17
bursts/s latches in ~340 ms. Three design points matter beyond the math. The
residue is **learned, never hardcoded** — a spec-derived DSB→DNB lead offset
that was wrong would silently stop DMO granting forever, a worse failure than
over-granting. The latch is **centred** on the agreeing leads and tracks slow
symbol slips (`dmGridDriftAdjust`), so jitter doesn't strand half a real train
outside the tolerance band. And the latch is **dropped when the train ends**
(`dmGridTrainGap`, 16 slots): without that, the ~0.2/s of noise landing on the
latched residue keeps a finished transmission "alive" and the grant never
re-arms. All of it is pinned failing-first —
`TestTETRADMOPipelineIgnoresIdleChannel`, `…GrantsOnlyAfterLock`, and
`…RearmsBetweenTransmissions` all fail against the old code — and the test
fixture itself got more honest: `buildDMODibitStream` now lays synthetic bursts
on a true 255-dibit slot grid, because the old arbitrary-filler layout was not
a faithful transmitter and could never have caught this.

<figure class="lab-figure">
<svg viewBox="0 0 680 210" width="680" height="210" role="img" aria-label="Histogram of DNB training-sequence lead residues modulo 255. Noise detections form a low uniform floor across all residues at roughly one per residue per minute, while a real transmission's bursts stack into a single tall spike at one residue — the residue the slot grid latches. A dashed line marks the six-agreeing-leads latch threshold.">
  <line x1="50" y1="20" x2="50" y2="160" stroke="var(--fg-muted)"/>
  <line x1="50" y1="160" x2="650" y2="160" stroke="var(--fg-muted)"/>
  <text x="16" y="30" fill="var(--fg-muted)" font-size="9">leads in</text>
  <text x="16" y="42" fill="var(--fg-muted)" font-size="9">1 s window</text>
  <text x="620" y="178" fill="var(--fg-muted)" font-size="9">residue mod 255</text>
  <g fill="var(--fg-muted)">
    <rect x="60" y="154" width="5" height="6"/><rect x="92" y="157" width="5" height="3"/>
    <rect x="121" y="154" width="5" height="6"/><rect x="160" y="157" width="5" height="3"/>
    <rect x="198" y="154" width="5" height="6"/><rect x="233" y="157" width="5" height="3"/>
    <rect x="269" y="157" width="5" height="3"/><rect x="305" y="154" width="5" height="6"/>
    <rect x="352" y="157" width="5" height="3"/><rect x="390" y="154" width="5" height="6"/>
    <rect x="431" y="157" width="5" height="3"/><rect x="466" y="157" width="5" height="3"/>
    <rect x="503" y="154" width="5" height="6"/><rect x="541" y="157" width="5" height="3"/>
    <rect x="578" y="154" width="5" height="6"/><rect x="616" y="157" width="5" height="3"/>
  </g>
  <rect x="330" y="36" width="8" height="124" fill="var(--accent)"/>
  <text x="334" y="28" text-anchor="middle" fill="var(--accent)" font-size="10">real train: ~17/s on ONE residue</text>
  <line x1="50" y1="112" x2="650" y2="112" stroke="var(--fg-muted)" stroke-dasharray="5 4"/>
  <text x="646" y="106" text-anchor="end" fill="var(--fg-muted)" font-size="9">latch threshold: 6 agreeing leads</text>
  <text x="200" y="146" fill="var(--fg-muted)" font-size="9">noise: ~18/s spread uniformly (~0.21 per residue band)</text>
  <text x="350" y="200" text-anchor="middle" fill="var(--fg-muted)" font-size="10">one radio, one clock, one residue — the vote separates traffic from noise with no decode at all</text>
</svg>
<figcaption>DNB lead residues mod 255: correlator noise is uniform across the ring while a real transmission stacks on a single residue — six agreeing leads latch it, and everything off-residue is discarded.</figcaption>
</figure>

## The voice chain's own lessons

`composer.runTETRADMOVoiceChain` — dispatched from `handleStart` on
`proto == "tetra-dmo"`, riding one same-carrier IQ tap — re-runs the extractor,
decodes DNBs via `DMBurstTCHSpeechSoft` with a hard fallback, and hands
137-bit frames to the recorder under the `tetra-dmo` → `tetra-acelp` vocoder
mapping. Because the grant fires at 4 qualified DNBs and colour recovery needs
~20, the chain buffers early bursts and decodes them **retroactively** once the
colour lands, so no leading speech is lost. The same first run taught it two
more lessons. Re-running the 64-colour brute force on *every* arriving burst
from buffer size 20 to 120 is `64·Σ(20..120)` ≈ **450,000 Viterbi decodes per
call** — the chain starved its own IQ tap (`dropped_chunks=5904`); recovery is
now capped at 6 batch-boundary passes over a bounded, slot-grid-qualified
scoring window, while the full buffer is kept for the retroactive decode. And
`flush()` used to set `colourKnown = true` even when giving up, so a call that
decoded nothing logged `colour=0 colour_known=true` — indistinguishable from a
successful recovery of colour 0, and guaranteed to send the next debugger after
the wrong thing. A separate `colourRecovered` flag now keeps the log honest.
For operators, the status line's split counters are the takeaway:
**`dnb_qualified` is the number that means traffic; `dnb_total` is a noise
meter**, and a large gap between them is normal, not a fault.

## Where this goes next

The pipeline is wired, the statistics are honest, and every regression that
bit us now fails first. What remains is the discipline this series has repeated
since Part 3: none of it is *verified* until the operator's own capture goes
through the full daemon and an intelligible recording lands.
[Part 14]({{ '/blog/deep-dives/tetra-end-to-end-14-testing-open-questions/' | relative_url }})
closes the series with the whole test lattice — synthetic, capture harness,
pipeline, daemon — and an unvarnished list of what is still open, including
the on-air gate that keeps [#1003](https://github.com/MattCheramie/GopherTrunk/issues/1003)
from being declared done.

## FAQ

**Why not just tighten the correlator tolerance instead of adding a grid?**
Because tolerance 2 is doing a job: a marginal real burst with two damaged
midamble dibits still detects. Tightening it trades recall for precision
inside one weak test. The grid adds an *independent* test — timing structure —
that costs real bursts nothing and that noise fails by construction.

**Could the grid reject a real transmission?**
Only a pathological one. A real train latches after ~6 bursts (~340 ms) and
the drift tracker follows symbol slips; the tolerance band absorbs jitter. The
deliberate trade is the ~0.5 s of grant latency (latch + 4 qualified DNBs) —
the buffered voice chain means no speech is lost to it.

**Why does the grant carry GroupID 0?**
DMO's air interface carries no talkgroup in what GT currently decodes — the
call-control PDUs (EN 300 396-3) are not yet parsed. Recordings therefore file
under group 0. It's in Part 14's open-questions list.

**What happens on a very short PTT — shorter than colour recovery needs?**
The grant can fire before the colour is known (hint 0), and the voice chain
recovers independently: it buffers everything, tries the pipeline's
`colourSink` hint (verified before adoption), then its own capped brute force,
then falls back to colour 0 at flush. A short clear call decodes; a short
encrypted one yields BFIs and hangs up normally.

**Is the DMO path now considered verified on air?**
No. Locks, colour recovery, and grants have all run live; the end-to-end gate —
a clear capture through the full daemon producing an intelligible recording —
is still open, and the project's verification discipline (#764/#771) says the
issue stays open until it lands. Part 14 states exactly what's left.

## Series navigation

**Part 13 of 14** · ←
[Part 12: DMO II — The 'Encrypted' Verdict That Was a Descramble Skip]({{ '/blog/deep-dives/tetra-end-to-end-12-dmo-descramble-colour/' | relative_url }})
· Next →
[Part 14: Testing TETRA Without a Network — & What's Still Open]({{ '/blog/deep-dives/tetra-end-to-end-14-testing-open-questions/' | relative_url }})
