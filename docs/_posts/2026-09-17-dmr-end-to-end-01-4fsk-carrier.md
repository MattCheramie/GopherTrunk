---
title: "DMR End to End, Part 1: The 4FSK Carrier & the Tier Family"
description: What a DMR carrier actually is — 4800 symbols per second of four-level FSK in a 12.5 kHz channel, dibits riding on ±648 and ±1944 Hz deviations, two 30 ms timeslots sharing one frequency, three tiers on one wire format, and the 48 kHz FM-discriminator chain GopherTrunk sizes around all of it.
category: deep-dives
keywords: dmr 4fsk modulation, dmr 4800 baud, dmr deviation 1944 hz, dmr two slot tdma, dmr tier 1 2 3, dmr fm discriminator decoder, dmr 12.5 khz channel, dmr receiver chain, gophertrunk dmr
tags: [dmr-end-to-end, dmr, 4fsk, dsp, demodulation, tdma, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "DMR End to End"
series_part: 1
---

*Part 1 of **DMR End to End**, a 14-part deep dive that follows the world's
most widely deployed digital PMR protocol through GopherTrunk — from a 4FSK
carrier to two simultaneous recorded calls, direct-mode handhelds, and
decrypted Enhanced Privacy voice.
[TETRA End to End]({{ '/blog/series/tetra-end-to-end/' | relative_url }})
followed a carrier whose information rode in phase transitions;
[P25 End to End]({{ '/blog/series/p25-end-to-end/' | relative_url }})
followed the amplitude-carried twin family. DMR borrows P25's physics and
TETRA's time-slicing, and adds the thread that runs this whole series:
**one carrier carries two of everything** — two slots, two sync polarities,
two tiers of trunking, two decode cadences — and the receiver only works
when it knows which one it is looking at. This opener is the physical layer
everything else stands on.*

> **TL;DR:** DMR is **4FSK at 4800 symbols/s** in a 12.5 kHz channel — the
> same four-level family as P25 Phase 1, with outer symbols at **±1944 Hz**
> and inner ones at ±648 Hz (`DeviationHz: 1944.0` in
> `internal/scanner/ccdecoder/pipelines.go`). Unlike P25 the transmit pulse
> *is* a root-raised-cosine (α = 0.20, `RolloffAlpha`); unlike TETRA the
> carrier is **time-sliced** — two 30 ms slots per 60 ms frame, each holding
> one 27.5 ms burst of 132 dibits. GopherTrunk channelizes it to **48 kHz —
> 10 samples/symbol** (`ddcTargetForProtocol`,
> `internal/scanner/ccdecoder/ddc.go`), and the receiver
> (`internal/radio/dmr/receiver`, imported as `dmrrx`) is a discriminator →
> carrier gate → RRC matched filter → Mueller-Müller → **post-clock**
> `CoarseAFC` → symbol AGC → slicer → `SymbolToDibit` chain. Three tiers ride
> the same dibits — `dmr-tier1`, `dmr-tier2` and `dmr` (Tier III) — differing
> only in which sync words they hunt and which state machine reads them.

**Key takeaways**

- **DMR asks P25's absolute question on TETRA's clock.** After the FM
  discriminator a symbol is a level, so the chain needs level calibration
  and offset control — but the carrier is TDMA, so every tracker must also
  survive gaps a repeater never shows and a handheld always does.
- **DMR is the RRC-matched member of the 4800-baud family.** P25 shapes
  with raised-cosine plus inverse-sinc; DMR shapes with a plain RRC at
  α = 0.20, so the textbook matched filter is *correct* here.
- **The tier is a reader, not a waveform.** Tier I, II and III pipelines
  build the same `dmrrx.Receiver` at the same 1944 Hz deviation; they differ
  in the sync alphabet and in what drives the grant.
- **48 kHz is a design rate, shared five ways.** Ten samples per symbol
  sizes the matched filter, the timing-loop gain and the AGC window for
  P25, DMR, NXDN, dPMR and YSF alike.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| On-air constants | 4800 sym/s, RRC α = 0.20 | `internal/radio/dmr/receiver/receiver.go` (`SymbolRate`, `RolloffAlpha`) |
| Deviation calibration | slicer scale = 2π·1944/48000 | `receiver.go` (`Options.DeviationHz`), set in `ccdecoder/pipelines.go` |
| Channel rate | 48 kHz for the 4800-baud family | `internal/scanner/ccdecoder/ddc.go` (`ddcTargetForProtocol`) |
| Symbol AGC | mean\|x\| → slicerScale·2/3 | `internal/dsp/demod/symbol_agc.go` (`C4FMAGCTarget`) |
| Carrier-offset control | post-clock `CoarseAFC` + coarse acquirer | `receiver.go`, `receiver/coarse_carrier.go` |
| Dibit mapping | {+1,+3,−1,−3} → 00, 01, 10, 11 | `receiver.go` (`SymbolToDibit`) |
| Tier pipelines | one receiver, three readers | `ccdecoder/pipelines.go` (`newDMRTier1Pipeline`, `newDMRTier2Pipeline`) |

## In this post

- **Where DMR sits** — amplitude-carried like P25, time-sliced like TETRA.
- **The deviation ladder, DMR edition** — ±1944/±648 Hz, the RRC that is right this time.
- **One carrier, two slots** — the 60 ms frame, the 27.5 ms burst, the gaps.
- **The tier family tree** — Tier I, II and III as three readers of one dibit stream.
- **48 kHz and the receiver in one pass** — IQ to dibits, and why the AFC sits after the clock.

## Where DMR sits

The two earlier series each opened with a table; here it is with all three
columns, because DMR borrows from both sides:

| Axis | P25 Phase 1 | DMR | TETRA TMO |
|---|---|---|---|
| Modulation | 4-level FSK | 4-level FSK | π/4-DQPSK |
| Outer deviation | ±1800 Hz | ±1944 Hz | — (phase) |
| Transmit pulse | RC × inverse-sinc | RRC α = 0.20 | RRC α = 0.35 |
| Access | FDMA | 2-slot TDMA, 60 ms frame | 4-slot TDMA |
| Downlink | continuous CC | repeater continuous, handheld bursty | continuous |

Row one decides the demodulator: like P25, DMR's dibit lives in an
*amplitude* after the FM discriminator, so the receiver asks the absolute
question and inherits everything the
[P25 opener]({{ '/blog/deep-dives/p25-end-to-end-01-c4fm-carrier/' | relative_url }})
catalogued — symbol AGC, an AFC because a tuner offset is a slicer bias,
thresholds derived from a physical deviation (the
[demodulation primer]({{ '/blog/deep-dives/sdr-internals-06-demodulation/' | relative_url }})
has the theory).

Row five is what P25 Phase 1 never faced. A DMR repeater keys both timeslots
continuously, so its carrier looks as steady as a P25 control channel; a DMR
*handheld* in direct mode transmits one burst per frame — 27.5 ms on,
32.5 ms off. The level trackers were built for the first shape and are
poisoned by the second, so the receiver carries a stage (`carrierGate`,
Part 9) whose only job is to know which shape it is hearing. **Two shapes of
the same carrier, and a receiver that has to tell them apart.**

## The deviation ladder, DMR edition

DMR's four levels sit at ±1944 Hz (outer, symbols ±3) and ±648 Hz (inner,
symbols ±1) — the standard 3:1 ratio, so only the slicer scale differs from
P25's, and GopherTrunk calibrates it once:

```go
// internal/radio/dmr/receiver/receiver.go (shape)
slicerScale := 1.0
if opts.DeviationHz > 0 {
    slicerScale = 2.0 * math.Pi * opts.DeviationHz / opts.SampleRateHz
}
```

At 1944 Hz and 48 kHz that is ≈ 0.254 rad/sample for an outer symbol — the
"±0.25 rad/sample" the carrier gate's documentation sets against the ±π of
receiver noise, a comparison Part 9 turns into a squelch. The
`DeviationHz <= 0` fallback survives only for legacy pre-scaled fixtures.

The dibit map is P25's, pinned by test:

```go
// internal/radio/dmr/receiver/receiver.go (shape)
// +3 → 01, +1 → 00, -1 → 10, -3 → 11 (TIA-102.BAAA; ETSI TS 102 361-1 agrees)
func SymbolToDibit(sym int8) uint8 {
    switch sym {
    case 1:  return 0
    case 3:  return 1
    case -1: return 2
    case -3: return 3
    }
    return 0
}
```

The high bit is the sign, the low bit selects inner/outer — so negating every
symbol, which is what a spectrum-inverted front end does, is exactly "add 2
mod 4" in dibit space. Part 2 leans on that.

**The matched filter is where DMR parts from P25.** The P25 series spent a
section on the RRC model being *wrong* for C4FM — modelling the real
raised-cosine-plus-inverse-sinc pulse as RRC left ISI on every capture
(issue #275). DMR's pulse is a plain root-raised-cosine at α = 0.20, so here
the textbook filter is correct: `demod.NewC4FM` over
`filter.RootRaisedCosine(sps, span, alpha)` taps, `RolloffAlpha = 0.20`,
`PulseSpanSymbols = 8` a side, where P25 goes through
`NewC4FMWithTaps(P25C4FMRxTaps(...))`.

## One carrier, two slots

DMR is two-slot TDMA. A 60 ms frame holds two 30 ms timeslots; each slot
carries one burst of 264 bits — 132 dibits, `dmr.BurstDibits` — lasting
27.5 ms, with the remaining 2.5 ms as guard or, on a base-station outbound,
the 24-bit [CACH]({{ '/reference/dmr-cach/' | relative_url }}). Every burst
has the skeleton
[Protocol Decoders Part 5]({{ '/blog/deep-dives/protocol-decoders-05-dmr-tier-2-3/' | relative_url }})
drew and Part 2 dissects: two payload halves around a central 48-bit field
that is either a sync word or embedded signalling.

The arithmetic the whole series rests on follows. At 4800 dibits/s a 60 ms
frame is **288 dibits**. A repeater fills both slots, so consecutive bursts
of *one* call — same slot, one frame apart — sit 288 dibits apart on a live
outbound (132 + 12 CACH, twice) or 264 on a CACH-free stream. A direct-mode
handheld transmits in one slot and leaves the other empty, so its bursts are
*also* 288 apart — with noise where the repeater put the other slot. Same
cadence, different filler; Part 3 is what happened when a decoder sliced
those gaps as voice.

## The tier family tree

ETSI TS 102 361 defines three tiers on one air interface; `ParseProtocol`
(`internal/trunking/site.go`) maps them to three names:

| Tier | What it is | `protocol:` | Sync words hunted | Grant source |
|---|---|---|---|---|
| I | licence-free direct mode | `dmr-tier1` | DM-Voice/Data TS1/TS2 | Voice LC Header |
| II | conventional repeater / IPSC | `dmr-tier2` | all nine | Voice LC Header (+ late entry) |
| III | trunked control channel | `dmr` | all nine | CSBK grants |

The [Tier I]({{ '/reference/dmr-tier-1/' | relative_url }}),
[Tier II]({{ '/reference/dmr-tier-2/' | relative_url }}) and
[Tier III]({{ '/reference/dmr-tier-3/' | relative_url }}) reference pages
carry the deployment context; here the point is how little changes below the
state machine. `newDMRTier1Pipeline` builds a Tier II `ConventionalChannel`
with the detector restricted to the four direct-mode words and the tag
`dmr-tier1`; `newDMRTier2Pipeline` builds the same channel with all nine plus
the operator's colour-code filter; Tier III swaps in `tier3.ControlChannel`
and an LCN resolver from `dmr_band_plan`
([band-plan reference]({{ '/reference/dmr-bandplan/' | relative_url }}),
[Cookbook Part 2]({{ '/blog/tutorials/operator-cookbook-02-dmr-tier3/' | relative_url }})).
All three construct the receiver identically:

```go
// internal/scanner/ccdecoder/pipelines.go (shape)
rx := dmrrx.New(dmrrx.Options{
    SampleRateHz: opts.SampleRateHz,
    DeviationHz:  1944.0, // ETSI TS 102 361-1 §6.3 peak deviation
    ClockGain:    0.015,  // Tier I/II; Tier III uses 0.025
    DibitSink: func(dibits []uint8, baseIdx int) {
        opts.tapDibits(dibits, baseIdx)
        cc.Process(dibits, baseIdx)
    },
})
```

The one tier-dependent DSP constant is `ClockGain`, and it is measured:
Tier II Voice LC Header bursts have a higher per-symbol transition magnitude
than Tier III's CSBK Aloha bursts (1.27 vs 0.90,
`TestDMRTier2VsTier3SymbolDensity`) and the loop slips at 0.025 on them.
Everything else is shared, and Parts 2, 4 and 5 walk it once for all three.

## 48 kHz and the receiver in one pass

The channel rate is chosen in one function:

```go
// internal/scanner/ccdecoder/ddc.go (shape)
func ddcTargetForProtocol(p trunking.Protocol) float64 {
    switch p {
    case trunking.ProtocolTETRA, trunking.ProtocolTETRADMO:
        return tetraDDCTargetRateHz // 144_000
    case trunking.ProtocolMotorola:
        return motorolaDDCTargetRateHz // 18_000
    }
    return ddcTargetRateHz // 48_000 — DMR and the rest of the C4FM family
}
```

Ten samples per symbol: the AGC's `Rate: 1/256` is a ~53 ms window and the
[Mueller-Müller loop]({{ '/blog/deep-dives/sdr-internals-07-symbol-timing-sync-recovery/' | relative_url }})
is sized for 10 sps. `Receiver.Process` composes the chain:

<figure class="lab-figure">
<svg viewBox="0 0 680 250" width="680" height="250" role="img" aria-label="The DMR receiver chain as seven stages from IQ to dibits: FM discriminator, carrier gate, RRC matched filter, Mueller-Muller clock, post-clock coarse AFC, symbol AGC and slicer, with a bracket noting the gate holds every tracker on absent samples and a note that the AFC runs after the clock.">
  <g font-size="9" fill="currentColor">
    <rect x="10" y="60" width="86" height="40" rx="4" fill="none" stroke="currentColor"/>
    <text x="53" y="84" text-anchor="middle">FM disc</text>
    <rect x="106" y="60" width="86" height="40" rx="4" fill="none" stroke="var(--accent)"/>
    <text x="149" y="84" text-anchor="middle" fill="var(--accent)">carrier gate</text>
    <rect x="202" y="60" width="86" height="40" rx="4" fill="none" stroke="currentColor"/>
    <text x="245" y="84" text-anchor="middle">RRC MF α=0.20</text>
    <rect x="298" y="60" width="86" height="40" rx="4" fill="none" stroke="currentColor"/>
    <text x="341" y="84" text-anchor="middle">MM clock</text>
    <rect x="394" y="60" width="86" height="40" rx="4" fill="none" stroke="var(--accent)"/>
    <text x="437" y="84" text-anchor="middle" fill="var(--accent)">CoarseAFC</text>
    <rect x="490" y="60" width="86" height="40" rx="4" fill="none" stroke="currentColor"/>
    <text x="533" y="84" text-anchor="middle">symbol AGC</text>
    <rect x="586" y="60" width="84" height="40" rx="4" fill="none" stroke="currentColor"/>
    <text x="628" y="84" text-anchor="middle">slicer ±1 ±3</text>
  </g>
  <g stroke="var(--fg-muted)">
    <line x1="96" y1="80" x2="106" y2="80"/><line x1="192" y1="80" x2="202" y2="80"/>
    <line x1="288" y1="80" x2="298" y2="80"/><line x1="384" y1="80" x2="394" y2="80"/>
    <line x1="480" y1="80" x2="490" y2="80"/><line x1="576" y1="80" x2="586" y2="80"/>
  </g>
  <text x="10" y="40" fill="currentColor" font-size="10" font-weight="bold">IQ at 48 kHz</text>
  <text x="670" y="40" text-anchor="end" fill="currentColor" font-size="10" font-weight="bold">4800 dibits/s → DibitSink</text>
  <path d="M 149 104 L 149 130 L 437 130 L 437 104" fill="none" stroke="var(--accent)" stroke-dasharray="3 3"/>
  <text x="293" y="146" text-anchor="middle" fill="var(--accent)" font-size="9">presence flags hold AGC, AFC and MM error on absent samples</text>
  <text x="437" y="196" text-anchor="middle" fill="currentColor" font-size="10" font-weight="bold">AFC after the clock, not before</text>
  <text x="437" y="211" text-anchor="middle" fill="var(--fg-muted)" font-size="9">its few-Hz wander in the timing loop drove sync to zero</text>
</svg>
<figcaption>The DMR receiver: a P25-shaped C4FM chain with two DMR-specific placements — a carrier-presence gate ahead of the matched filter, and the coarse AFC moved behind the symbol clock.</figcaption>
</figure>

Two stages the figure omits sit at the ends: before the discriminator, a
one-shot **coarse carrier acquirer** (`coarse_carrier.go`) de-rotates grossly
mistuned dongles; after the slicer, `SymbolToDibit` hands dibits to a
`dmr.DibitSink` with the absolute `baseIdx` every framer downstream uses.

The **carrier gate** yields one presence flag per sample; on a continuous
carrier every flag is true and the gated stages equal their ungated forms
(`TestReceiverCarrierGateIsNoOpOnContinuousCarrier`). Then the DMR-specific
placement: **`CoarseAFC` runs on the recovered symbol stream**, post-clock
and pre-slicer, where P25 corrects before timing. The DC bias is identical in
both domains, but the open-loop estimate wanders a few hertz chasing the data
mean, and fed into the timing loop it destabilised symbol timing on a clean
signal — verified to drive synthetic sync to zero. The **symbol AGC**
normalises mean|x| to `slicerScale·2/3` (`C4FMAGCTarget`): the unit-energy
RRC has a DC gain near 3.1, and without it every inner symbol slices as outer
while sync and slot type still pass — the "Tier III BPTC uncorrectable /
Tier II instalock then nothing" field failure.

### How the twin thread shaped the Go code

- **One receiver, parameterised by evidence.** `dmrrx.Options` has no tier
  field; tiers differ in the sync alphabet handed to `dmr.NewSyncDetector`
  and in a measured `ClockGain`.
- **Gated variants are provably neutral.** `ProcessGated` on the clock, AFC
  and AGC equal their ungated forms when every sample is present — the
  [Two Pipelines]({{ '/blog/solution-postmortem/from-the-issue-tracker-22-two-pipelines/' | relative_url }})
  lesson inside one receiver.
- **Calibration is derived, never assumed.** `DeviationHz` drives the slicer
  scale and `C4FMAGCTarget` derives the AGC target from it.
- **The stream has an absolute clock.** `DibitSink(dibits, baseIdx)` carries
  a monotonic index that `Reset` restarts at zero — the coordinate every
  cadence, late-entry and re-key rule in Parts 3–6 is expressed in.

## Where this goes next

A dibit stream at 4800/s is a firehose with no punctuation.
[Part 2]({{ '/blog/deep-dives/dmr-end-to-end-02-bursts-sync-polarity/' | relative_url }})
adds the structure: the 132-dibit burst, the nine sync words and the
uncomfortable fact that they are closed under a polarity flip, the
Golay-protected slot type that routes each burst, and the moment a FEC-valid
burst locks the stream's polarity for good.

## FAQ

**Is DMR the same modulation as P25 Phase 1?**
Same family, different pulse and deviation. Both are 4800-baud four-level FSK
in 12.5 kHz demodulated by an FM discriminator, and GopherTrunk uses the same
`SymbolToDibit` map for both. DMR's outer deviation is ±1944 Hz against
P25's ±1800, and DMR shapes with a root-raised-cosine where P25 does not.

**Why does GopherTrunk decode DMR at 48 kHz?**
Because 48 kHz is exactly 10 samples per symbol at 4800 baud, the operating
point every loop constant in the C4FM receivers was tuned at.
`ddcTargetForProtocol` returns it for the whole 4800-baud family; TETRA gets
144 kHz and Motorola SmartNet 18 kHz from the same function.

**What is the difference between DMR Tier I, II and III?**
Tier I is licence-free direct mode — handheld to handheld on one frequency.
Tier II is licensed conventional: a repeater carrying two timeslots. Tier III
adds a dedicated control channel and CSBK grants for trunking. In GopherTrunk
they are `dmr-tier1`, `dmr-tier2` and `dmr`, all built on one receiver.

**Does a DMR repeater transmit continuously?**
Yes — a keyed base station fills both timeslots every 60 ms frame, with a
CACH between bursts, so its carrier is as steady as a P25 control channel.
A direct-mode handheld sends one 27.5 ms burst per frame and is silent for
32.5 ms, which is why the receiver carries a carrier-presence gate.

**Why does DMR's AFC run after the symbol clock when P25's runs before?**
Because the open-loop coarse estimate wanders by a few hertz chasing the data
mean, and on DMR feeding that bias into the Mueller-Müller loop destabilised
timing on a clean signal — it drove synthetic decode sync to zero.
Correcting the recovered symbols recentres the eye without touching the
timing loop; the pre-clock coarse acquirer handles large offsets.

## Series navigation

**Part 1 of 14** · Next →
[Part 2: Bursts, Sync Words & Polarity]({{ '/blog/deep-dives/dmr-end-to-end-02-bursts-sync-polarity/' | relative_url }})
