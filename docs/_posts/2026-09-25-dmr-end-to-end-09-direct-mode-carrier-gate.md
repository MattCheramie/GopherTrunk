---
title: "DMR End to End, Part 9: Direct Mode — The Gaps That Blinded the Receiver"
description: "Why a DMR handheld on a simplex frequency never decoded for the whole life of issue 836 — the 32.5 ms gaps between bursts fed receiver noise to every slow tracker — and the three fixes that make it decode from the first header: a variance carrier gate, feed-forward timing acquisition and an acquirer that keeps its lock."
category: deep-dives
keywords: dmr direct mode decode, dmr simplex sdr, dmr tier 1 receiver, carrier gate fm discriminator variance, symbol timing acquisition kurtosis, mueller muller pull in, coarse carrier acquisition dmr, pmr446 dmr decode, gophertrunk dmr
tags: [dmr-end-to-end, dmr, direct-mode, dsp, receiver, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "DMR End to End"
series_part: 9
---

*Part 9 of **DMR End to End**, a 14-part deep dive that follows the world's
most widely deployed digital PMR protocol through GopherTrunk — from a 4FSK
carrier to two simultaneous recorded calls, direct-mode handhelds, and
decrypted Enhanced Privacy voice.
[Part 8]({{ '/blog/deep-dives/dmr-end-to-end-08-tier3-trunking/' | relative_url }})
finished the trunked story on a base station that never stops transmitting.
This part turns to the carrier that does stop — a simplex handheld, on for
27.5 ms and off for 32.5 — and the bug that hid in those gaps for the
entire life of
[#836](https://github.com/MattCheramie/GopherTrunk/issues/836). DMR's one
carrier has two cadences, and a receiver that only knows one is blind to the
other.*

> **TL;DR:** A direct-mode (Tier I / simplex) DMR handheld transmits one
> 27.5 ms burst per 60 ms frame. In the 32.5 ms gap the FM discriminator of
> receiver noise is uniform over ±π rad/sample — several times the signal's
> ±0.25 — and every slow tracker in `internal/radio/dmr/receiver` ate it:
> the symbol AGC's 53 ms EMA inflated so no ±3 sync word ever sliced as ±3,
> the AFC decayed, the coarse acquirer's mean halved and the Mueller-Müller
> loop random-walked. Fixes: `carrierGate` — the discriminator's mean-removed variance
> (0.01–0.08 rad² in bursts, 2.3–3.9 in gaps), open below 1.0, close at 2.5,
> trackers held via `ProcessGated`, discriminator muted; feed-forward timing
> at every onset after a 100 ms absence
> (`sync.EstimateSymbolPhase`); and an acquirer that re-mixes its engage
> chunk and re-locks the clock only at ≥ 1400 Hz. On-air verified on the
> reporter's 15 Sep captures; the live run on a fixed build is still open.

**Key takeaways**

- **A burst-mode carrier is a different signal, not a weaker one.** The gaps
  fed noise to four trackers whose time constants were longer than a burst.
- **Gate on a statistic, never on dBFS.** The discriminator variance is a
  phase increment — blind to IQ amplitude and carrier offset, a 30×
  separation between the populations at any gain.
- **Feedback loops need a seed.** A Mueller-Müller loop started near a
  half-symbol offset takes seconds to pull in; a 96-symbol kurtosis search
  gives it the phase inside the first burst.
- **Green synthetic ≠ on-air.** The back-to-back fixture passed for the
  issue's whole life; the reporter's clipped captures proved the fix — and
  the live daemon run is still the open gate.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| Carrier presence | running discriminator variance, hysteresis 1.0 / 2.5 rad², 4-symbol delay line | `internal/radio/dmr/receiver/carrier_gate.go` (`carrierGate`) |
| Gated trackers | AGC / AFC / timing updates hold on absent samples | `ProcessGated` on `demod.C4FMSymbolAGC`, `demod.CoarseAFC`, `sync.MuellerMuller` |
| Onset timing | kurtosis eye search, seeds `SetPhase` after a 100 ms absence | `receiver/timing_acq.go`, `internal/dsp/sync/timing_estimate.go` |
| Coarse offset | two agreeing 512-symbol windows above a 500 Hz deadband, frozen NCO; clock relock only ≥ 1400 Hz | `receiver/coarse_carrier.go` (`coarseAcqClockRelockHz`) |
| Failing-first pins | zero syncs ungated on a synthetic direct-mode stream | `receiver_burst_test.go`, `ccdecoder/pipelines_dmr_directmode_test.go` |
| Real-air pins | reporter's 446.500 MHz slices, gated vs `NoCarrierGate` | `cmd/gophertrunk/dmr_directmode_realair_test.go` |

## In this post

- **The transmission every fixture got wrong** — gap noise, four poisoned trackers.
- **A squelch expressed as a statistic** — the variance gate and what it holds.
- **Every keyup lands at a random phase** — feed-forward timing acquisition.
- **The acquirer's two traps** — the engage chunk and the thrown-away lock.
- **What the reporter's air proved, and what it hasn't yet** — the 15 Sep captures.

## The transmission every fixture got wrong

A base-station DMR carrier is continuous — both timeslots are always
transmitted — and every synthetic DMR fixture modelled exactly that,
132-dibit bursts laid back to back (see
[Part 3]({{ '/blog/deep-dives/dmr-end-to-end-03-two-slots-one-carrier/' | relative_url }})).
A direct-mode MS — PMR446's 446.500 MHz
"DMR simplex", the case of #836 — transmits **one 27.5 ms burst per 60 ms
TDMA frame** and is off for the other 32.5 ms (ETSI TS 102 361-1 §4.2). In
that gap the FM discriminator of receiver noise is uniform over ±π
rad/sample — several times *larger* than the ±0.25 rad/sample the signal's
±1944 Hz deviation produces at 48 kHz — in exactly the quantity every
tracker averages. The symbol AGC's 256-symbol EMA (53 ms) inflated every
gap, so outer symbols sliced as inner and **no sync word — all ±3 — ever
matched**. The post-clock `CoarseAFC` decayed to zero. The coarse acquirer's
512-symbol window was ~54 % noise, so a 1.2 kHz offset read ~550 Hz, at its
own 500 Hz deadband. And the Mueller-Müller error term random-walked between
bursts.

Reproduced failing-first: the production Tier I **and** Tier II pipelines
decoded **zero** sync words from a synthetic direct-mode stream at 27 dB SNR
(`TestDMRPipelinesDecodeDirectModeTransmission`,
`TestReceiverDecodesDirectModeBurstCadence`) while decoding the same bursts
perfectly back to back — the
[self-consistent trap]({{ '/blog/solution-postmortem/from-the-issue-tracker-20-self-consistent-trap/' | relative_url }})
as a fixture and a receiver sharing a wrong *world*.

## A squelch expressed as a statistic

The fix is an FM noise-quieting squelch written as a statistic. `carrierGate`
keeps the running mean and mean-square of the discriminator over a
four-symbol EMA window and reads the variance:

```go
// internal/radio/dmr/receiver/carrier_gate.go (shape)
const (
    carrierGateOpenBelow = 1.0   // rad²: absent → present
    carrierGateCloseAt   = 2.5   // rad²: present → absent
)
g.mean += g.rate * (v - g.mean)
g.sq   += g.rate * (v*v - g.sq)
variance := g.sq - g.mean*g.mean
if g.open { g.open = variance < g.closeAt } else { g.open = variance < g.openBelow }
```

The two populations were **measured on the reporter's capture through the
production DDC**: 0.01–0.08 rad² inside every burst — even ADC-clipped — and
2.3–3.9 rad² in every gap. **Mean removal makes it offset-independent**: a
ppm error is a constant bias, not variance. **Scale invariance is
inherent**: a phase increment is blind to IQ amplitude, the
coherence-over-dBFS rule
[MRC calibration]({{ '/blog/deep-dives/weak-signal-engineering-10-mrc-calibration/' | relative_url }})
learned the hard way. And an exactly-zero IQ sample closes the gate, since a
dead input's constant-0 discriminator would read as a quiet carrier.

<figure class="lab-figure">
<svg viewBox="0 0 680 250" width="680" height="250" role="img" aria-label="A timeline of a direct-mode DMR transmission: two 27.5 millisecond bursts of tight modulation between 32.5 millisecond gaps of noise swinging over plus and minus pi. Below, a variance curve reads under 0.1 in bursts and about 3 in gaps, crossing an open threshold at 1.0 and a close threshold at 2.5; a presence row opens shortly after each burst start with a four-symbol lag, and a bottom row shows the trackers holding in gaps and updating in bursts.">
  <text x="340" y="14" text-anchor="middle" fill="currentColor" font-size="10" font-weight="bold">direct mode: 27.5 ms burst · 32.5 ms gap · 60 ms frame</text>
  <line x1="30" y1="60" x2="650" y2="60" stroke="var(--fg-muted)"/>
  <rect x="30" y="30" width="138" height="60" fill="none" stroke="var(--fg-muted)" stroke-dasharray="2 2"/>
  <rect x="168" y="53" width="128" height="14" fill="none" stroke="currentColor" stroke-width="1.5"/>
  <rect x="296" y="30" width="150" height="60" fill="none" stroke="var(--fg-muted)" stroke-dasharray="2 2"/>
  <rect x="446" y="53" width="128" height="14" fill="none" stroke="currentColor" stroke-width="1.5"/>
  <rect x="574" y="30" width="76" height="60" fill="none" stroke="var(--fg-muted)" stroke-dasharray="2 2"/>
  <text x="232" y="104" text-anchor="middle" fill="currentColor" font-size="8">burst: ±0.25 rad/sample</text>
  <text x="371" y="104" text-anchor="middle" fill="var(--fg-muted)" font-size="8">gap: noise over ±π</text>
  <text x="26" y="132" text-anchor="end" fill="var(--fg-muted)" font-size="8">var</text>
  <line x1="30" y1="122" x2="650" y2="122" stroke="var(--fg-muted)" stroke-dasharray="3 3"/>
  <text x="654" y="125" fill="var(--fg-muted)" font-size="8">2.5 close</text>
  <line x1="30" y1="136" x2="650" y2="136" stroke="var(--accent)" stroke-dasharray="3 3"/>
  <text x="654" y="139" fill="var(--accent)" font-size="8">1.0 open</text>
  <path d="M30 116 L168 116 L190 146 L296 146 L318 116 L446 116 L468 146 L574 146 L596 116 L650 116" fill="none" stroke="currentColor" stroke-width="1.5"/>
  <text x="26" y="176" text-anchor="end" fill="var(--fg-muted)" font-size="8">present</text>
  <rect x="180" y="166" width="126" height="12" fill="none" stroke="var(--accent)"/>
  <rect x="458" y="166" width="126" height="12" fill="none" stroke="var(--accent)"/>
  <text x="243" y="175" text-anchor="middle" fill="var(--accent)" font-size="8">open, 4-symbol lag</text>
  <text x="521" y="175" text-anchor="middle" fill="var(--accent)" font-size="8">closes ~3 symbols late</text>
  <text x="26" y="210" text-anchor="end" fill="var(--fg-muted)" font-size="8">trackers</text>
  <text x="105" y="208" text-anchor="middle" fill="var(--fg-muted)" font-size="8">HOLD · muted</text>
  <text x="243" y="208" text-anchor="middle" fill="currentColor" font-size="8" font-weight="bold">UPDATE</text>
  <text x="382" y="208" text-anchor="middle" fill="var(--fg-muted)" font-size="8">HOLD · muted</text>
  <text x="521" y="208" text-anchor="middle" fill="currentColor" font-size="8" font-weight="bold">UPDATE</text>
</svg>
<figcaption>The gate reads the discriminator's variance, not the IQ level: bursts and gaps sit thirty times apart on that axis at any gain.</figcaption>
</figure>

The decision does two things. The trackers hold — AGC, AFC, acquirer and
Mueller-Müller skip absent samples through gated variants, byte-identical to
plain `Process` when every flag is true. And the discriminator is **muted**:
the RRC matched filter otherwise carries ±π gap noise into every burst's
first span — unmuted, half the header bursts failed BPTC. The statistic is
causal, so the discriminator runs through a `carrierGateSymbols · sps` delay
line while the decision does not, and on a continuous carrier the gated
output is the ungated one shifted by exactly that lag —
`TestReceiverCarrierGateIsNoOpOnContinuousCarrier` pins it dibit for dibit.

## Every keyup lands at a random phase

The reporter's captures then showed the next layer. The gate holds the
timing loop across gaps *within* a transmission, but every keyup starts at a
symbol phase of its own, and a Mueller-Müller loop pulls a phase error in at
gain·error per symbol — from near a half-symbol offset, its unstable
equilibrium, that takes seconds. Cold-started on the second PTT at ten
sub-sample offsets, the receiver decoded the first burst at seven and took
1.3–2.8 s at the other three; live, the phase *held* from PTT 1 cost PTT 2
its entire ten-copy header train.

The fix is feed-forward acquisition in `receiver/timing_acq.go`. After an
absence longer than `timingAcqGapSymbols` (480 symbols = 100 ms — a new
transmission, not the next burst's 156-symbol gap), the receiver skips
`timingAcqSkipSymbols` = 10 symbols of filter transient, collects
`timingAcqSymbols` = 96 symbol periods from inside the first burst, and
hands them to `sync.EstimateSymbolPhase`: for each of the sps candidate
phases it decimates and measures the kurtosis E[x⁴]/E[x²]² about the mean.
A 4-level signal at its ISI-free instant takes four discrete values; between
instants the ISI mixtures smear toward Gaussian. Lowest wins, a parabolic
fit refines below one sample, and `maxEyeKurtosis` = 2.2 /
`minEyeContrast` = 1.12 refuse a window with no eye. The moments are
**central**, since an uncorrected carrier offset is a DC bias on the
discriminator. The square-law Oerder-Meyr estimator was tried first and is
**useless here**: at the C4FM family's 20 % roll-off the symbol-rate line is
often below the noise floor while the eye stays open. The estimate seeds
`MuellerMuller.SetPhase` mid-chunk; the first burst's own sync word is still
lost, but DMR radios repeat their header. Acquisition arms only *after* an
absence, so the continuous-carrier pin stays byte-identical
([SDR Internals Part 7]({{ '/blog/deep-dives/sdr-internals-07-symbol-timing-sync-recovery/' | relative_url }})
has the loop this seeds).

## The acquirer's two traps

The coarse carrier acquirer (`coarse_carrier.go`) is the one frozen stage in
the chain: it averages the discriminator over
`coarseAcqSymbols` = 512 symbols, needs **two consecutive windows** clearing
a 500 Hz deadband and agreeing within 250 Hz — a tuner offset repeats, noise
doesn't — then de-rotates once through an NCO and freezes. Fed only present
samples, its halved-estimate failure is gone. Two more traps surfaced:

- **The correction applied "on the NEXT chunk".** A 4096-sample chunk is
  ~85 ms, a whole burst, so the rest of the engage chunk re-locked the reset
  loop against the still-shifted signal — 34/40 burst syncs with 10 ms
  chunks, 20/40 with 4096-sample ones. `Process` now re-mixes the engage
  chunk at the frozen offset.
- **`clock.Reset()` on engage threw away a good lock.** A de-rotation moves
  nothing in the timing domain; at −1.2 kHz the reset dropped bursts from
  129/132 to ~80/132 dibits while the loop crawled back. The clock now
  re-locks only at `coarseAcqClockRelockHz` = 1400 Hz — the #1165 notch
  where the sgn() error term genuinely breaks.

## What the reporter's air proved, and what it hasn't yet

Everything above could still have been another self-consistent fixture. What
settled #836 was Alvin's three 20 s cs16 captures of 15 Sep — 446.500 MHz,
2.4 MS/s, gain auto / 200 / 250, −1.6..−1.9 kHz off the carrier, and still
**36–40 % ADC-clipped** by a handheld in the same room. Two 48 kHz
slices are committed as
`cmd/gophertrunk/testdata/dmr-directmode-446500-*.cs16`, and
`TestDMRDirectModeRealAirKeyup` runs them through the production receiver
and Tier II state machine: exactly one grant — tg 99 / src 3024109 / cc 1 —
34–39 voice superframes, zero uncorrectable AMBE frames, and **zero** sync
words with `NoCarrierGate` — the real-air pin of the gate itself.
`TestDMRDirectModeRealAirOnsetAtEveryPhase` demands first sync within 0.15 s
at all ten start phases of the second PTT.

Two more defects fell out of the same captures.
[Part 5]({{ '/blog/deep-dives/dmr-end-to-end-05-link-control-late-entry/' | relative_url }})'s
0.25 s re-key rule (`headerRekeyDibits` = 1200) was measured from the
*first* header copy — but this radio repeats its Voice LC Header **ten times
over 0.6 s**, so every keyup produced a phantom release and re-grant; the
anchor now follows the last copy (`TestConventionalHeaderTrainIsOneKeyup`).
And direct-mode voice must slice at the 288-dibit cadence: the 132-cadence
decoder sliced the gaps, so `trunking.DMRVoiceCadenceDetected` now defaults
every DMR protocol to the cadence-detecting decoder.

On clipping: the 14 Sep capture at ~40 % with `gain: auto` was
**undecodable by construction** — `capture`, `replay` and the wideband
no-sync WARN now say "front-end overload" instead of "set sdr.ppm" — yet the
15 Sep files at 36–40 % decode, so "clipped ⇒ undecodable" was too strong
([The Analog Edge Part 4]({{ '/blog/tutorials/analog-edge-04-clipping-overload-intermod/' | relative_url }}),
[Nineteen Dibits]({{ '/blog/solution-postmortem/from-the-issue-tracker-08-nineteen-dibits/' | relative_url }})).
What is **not** verified: the reporter's **live daemon run on a build with
these fixes** — chunking, pool, composer and recorder — and #836 stays open
until that recording lands.

### How the gaps shaped the Go code

- **Gated variants, never gated callers.** Each tracker grew a
  `ProcessGated(…, present []bool)` whose nil-present path is its old
  `Process`.
- **The gate lives on the calibrated path only.** `newCarrierGate` is built
  when `DeviationHz > 0 && !NoCarrierGate`.
- **Measured constants carry their measurements.** `carrierGateOpenBelow`
  and `coarseAcqClockRelockHz` cite the capture numbers that set them.
- **Fixtures model a PTT, not a loop.** `dmHeaderBursts` cycles source IDs —
  one burst repeated for seconds carries a symbol-mean bias the AFC reads as
  drift.

## Where this goes next

Direct mode is one handheld on one dongle. The other end of the scale is a
wideband X310 carrying two IPSC repeaters through a polyphase channelizer —
and one tap went deaf for minutes while its neighbour decoded everything.
[Part 10]({{ '/blog/deep-dives/dmr-end-to-end-10-wideband-bin-edge-deaf-heal/' | relative_url }})
follows that tap to the bin edge, the self-healing guard, and the still
unnamed −20 kHz emitter.

## FAQ

**Why did GopherTrunk decode DMR repeaters but never a simplex handheld?**
A repeater transmits continuously; a direct-mode handheld sends one 27.5 ms
burst per 60 ms frame. In the 32.5 ms gaps the FM discriminator of receiver
noise swings over ±π rad/sample and inflated the symbol AGC, decayed the AFC
and random-walked the timing loop.

**What is the DMR carrier gate and does it use a power threshold?**
It is `carrierGate` in `internal/radio/dmr/receiver`: the running
mean-removed variance of the discriminator output, open below 1.0 rad² and
closed at 2.5 rad², with a four-symbol delay line. No dBFS anywhere — a
phase increment is blind to IQ level and carrier offset, so bursts and gaps
separate at any gain.

**Why does the first PTT after start-up decode late while later ones decode from the first burst?**
With a tuner offset above ~1 kHz the coarse acquirer needs two agreeing
512-symbol windows of present samples (~0.5 s of bursts) before it freezes
its correction. Set `sdr.ppm` from `gophertrunk capture`'s measured line and
the first PTT decodes too.

**Is the direct-mode fix verified on air?**
Offline, yes: the reporter's 15 Sep 446.500 MHz captures decode through the
production receiver and Tier II state machine (tg 99, radio 3024109, 34–39
superframes, zero uncorrectable AMBE) and yield zero sync words with the gate
off. The reporter's live daemon run on a fixed build is still the open gate
on #836.

**Can a 36–40 % ADC-clipped capture really decode?**
On these captures it did — the discriminator variance stays inside the
gate's signal population even when clipped — which refuted "clipped ⇒
undecodable" as a rule. Clipping still costs margin; the 14 Sep capture at
~40 % with `gain: auto` did not decode.

## Series navigation

**Part 9 of 14** · ←
[Part 8: Tier III Trunking — C_ALOHA, Grants & LCNs]({{ '/blog/deep-dives/dmr-end-to-end-08-tier3-trunking/' | relative_url }})
· Next →
[Part 10: Wideband DMR — Bin Edges & the Deaf Heal]({{ '/blog/deep-dives/dmr-end-to-end-10-wideband-bin-edge-deaf-heal/' | relative_url }})
