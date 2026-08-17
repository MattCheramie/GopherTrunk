---
title: "TETRA End to End, Part 1: π/4-DQPSK & the Shape of a TETRA Carrier"
description: Why TETRA is not "European P25" — a 25 kHz carrier at 18000 symbols per second, dibits riding in phase transitions instead of amplitudes, a four-slot TDMA downlink that never stops transmitting, and the 144 kHz channel rate GopherTrunk's whole TETRA path is sized around.
category: deep-dives
keywords: tetra pi/4 dqpsk, differential qpsk demodulation, tetra 18000 symbols per second, tetra 25 khz channel, tdma continuous downlink, tetra gray mapping, tetra 144 khz channel rate, differential decode conj, gophertrunk tetra
tags: [tetra-end-to-end, tetra, dqpsk, dsp, demodulation, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "TETRA End to End"
series_part: 1
---

*Part 1 of **TETRA End to End**, a 14-part deep dive that follows one real
25 kHz TETRA carrier — a live TMO cell, MCC 250 / MNC 13, captured by an
operator and replayed through GopherTrunk hundreds of times — all the way from
raw IQ to clear recorded voice. The
[Protocol Decoders survey]({{ '/blog/deep-dives/protocol-decoders-07-tetra/' | relative_url }})
gave TETRA one episode; this series gives it the full treatment, because almost
everything hard we learned about decoding radio in the last year happened on
this carrier. One villain recurs the whole way: the test that passes because
both sides share the same bug — **green synthetic ≠ on-air correct**. This
opener is about the physical layer everything else stands on: what a TETRA
carrier actually looks like, and why information riding in phase *transitions*
shapes every design decision downstream.*

> **TL;DR:** TETRA is **π/4-DQPSK at 18000 symbols/s** in a 25 kHz channel —
> not four-level FSK like the P25/DMR/NXDN family. Each symbol carries one
> dibit in the **phase change** from the previous symbol, recovered as
> `arg(s·conj(last))` (`internal/dsp/demod/dqpsk.go`), so no absolute carrier
> phase is ever needed — a property the equalizer work in Parts 8–10 leans on
> hard. The constellation rotates π/4 every symbol, so there is always a
> transition to clock off. GopherTrunk channelizes TETRA to **144 kHz — 8
> samples/symbol** (`ddcTargetForProtocol`, `internal/scanner/ccdecoder/ddc.go`)
> where the C4FM family gets 48 kHz, and the receiver
> (`internal/radio/tetra/receiver`) is an RRC matched filter → Gardner timing →
> AFC → differential decode chain sized entirely from that rate.

**Key takeaways**

- **The dibit lives in the transition, not the symbol.** The demodulator
  computes `s·conj(last)` and slices the phase delta into quadrants — a
  constant phase rotation of the whole constellation cancels out. That one
  property decides what is and isn't safe to put in front of this decoder.
- **The constellation never sits still.** A π/4 rotation per symbol means
  consecutive symbols always differ, giving timing recovery a transition to
  lock to and leaving eight visible constellation points from two interleaved
  QPSK sets.
- **TETRA gets its own Gray map and its own channel rate.** `TetraBitsToDibits`
  is deliberately separate from the C4FM `framing.DibitsToBits`, and the DDC
  target is 144 kHz where every 4800-baud protocol gets 48 kHz.
- **The downlink never stops.** A TETRA base station transmits continuously —
  four TDMA slots per 56.67 ms frame — so the receiver is a stream processor,
  not a burst hunter, and lock quality is measurable every single frame.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| Differential demod | `arg(s·conj(last)) − rotation` → dibit quadrant | `internal/dsp/demod/dqpsk.go` (`DQPSK.Decode`) |
| π/4 wrapper + RRC | matched filter (α = 0.35) + rotation constant | `internal/dsp/demod/piover4_dqpsk.go` (`PiOver4DQPSK`) |
| Bit↔dibit convention | TETRA Gray map, distinct from C4FM linear | `internal/radio/tetra/sync.go` (`TetraBitsToDibits`) |
| Channel rate | 144 kHz for TETRA, 48 kHz for C4FM family | `internal/scanner/ccdecoder/ddc.go` (`ddcTargetForProtocol`) |
| IQ → dibit chain | filter → timing → AFC → differential decode | `internal/radio/tetra/receiver/receiver.go` (`Receiver.Process`) |
| Timing recovery | Gardner loop, the default clock mode | `receiver.go` (`ParseClockMode`, `ClockGardner`) |

## In this post

- **Not "European P25"** — the assumptions TETRA breaks, in one table.
- **Information in the transition** — the differential decode and what it buys.
- **A constellation that never sits still** — eight points, two QPSK sets, one Gray map.
- **144 kHz or nothing** — why the channel rate is a per-protocol decision.
- **The receiver in one pass** — the chain from IQ to dibits, and the hooks we'll use later.

## Not "European P25"

It's tempting to file TETRA as "the European trunking protocol" and assume the
P25 machinery transfers. Almost none of it does. Every C4FM-family protocol in
GopherTrunk — P25 Phase 1, DMR, NXDN, dPMR — is four-level FSK: an FM
discriminator turns frequency into amplitude, and a slicer maps four amplitude
levels to dibits. TETRA is a different animal on every axis that matters:

| Axis | P25 Phase 1 (C4FM) | TETRA TMO |
|---|---|---|
| Modulation | 4-level FSK (frequency) | π/4-DQPSK (differential phase) |
| Symbol rate | 4800 sym/s | 18000 sym/s |
| Channel width | 12.5 kHz | 25 kHz |
| Channel bit rate | 9.6 kbps | 36 kbps before slot muxing |
| Downlink | bursty | continuous 4-slot TDMA |
| GT channel rate | 48 kHz | 144 kHz |
| Vocoder | IMBE (MBE family) | ACELP (EN 300 395-2) |

The one that shapes this whole series is the first row. A C4FM slicer asks
"what amplitude is this symbol?" — an absolute question. A TETRA demodulator
asks "how far did the phase move since the *last* symbol?" — a relative one.
The [demodulation primer]({{ '/blog/deep-dives/sdr-internals-06-demodulation/' | relative_url }})
covers both families in the abstract; here we care about what the relative
question buys and costs on a real carrier.

## Information in the transition

The core of the demodulator is a dozen lines. At each symbol time it multiplies
the current symbol by the conjugate of the previous one — which subtracts their
phases — removes the π/4 rotation offset, and slices the remaining phase delta
into one of four quadrants:

```go
// internal/dsp/demod/dqpsk.go (shape) — Decode
for i, s := range src {
    // Phase delta: arg(s * conj(last)) − rotation.
    ar := real(s)*real(d.last) + imag(s)*imag(d.last)
    ai := imag(s)*real(d.last) - real(s)*imag(d.last)
    phi := math.Atan2(float64(ai), float64(ar)) - d.rotation
    /* … wrap to [-π, π) … */
    switch {
    case phi >= -math.Pi/4 && phi < math.Pi/4:
        dst[i] = 0b00
    case phi >= math.Pi/4 && phi < 3*math.Pi/4:
        dst[i] = 0b01
    /* … 0b11, 0b10 … */
    }
    d.last = s
}
```

Read what is *not* in that loop: no carrier phase estimate, no phase-locked
loop, no reference symbol. `d.last` is the only memory, initialized to `1+0i`.
If the whole constellation arrives rotated by some unknown constant — because
the LO started at an arbitrary phase, or a filter imposed a phase shift — the
rotation appears in both `s` and `last` and **cancels in the conjugate
product**. That is the deep reason differential modulation exists, and it is
worth planting now because it becomes load-bearing later: any processing
inserted before this decoder is safe *only if* it imposes a constant phase.
A filter whose phase wanders symbol-to-symbol does not cancel in
`s·conj(last)` and corrupts every dibit — the exact trap the equalizer work in
Parts 8–10 had to design around, and a theme the
[Weak-Signal Engineering series]({{ '/blog/series/weak-signal-engineering/' | relative_url }})
develops in full.

The cost of the relative question is error doubling: one bad symbol corrupts
two transitions. TETRA's channel coding (Part 3) is budgeted for that.

## A constellation that never sits still

Plain DQPSK encodes dibits as phase deltas of 0°, ±90°, 180°. The 180° case is
a problem: the trajectory passes through the origin, the envelope collapses,
and timing recovery loses its reference. π/4-DQPSK fixes this by adding 45° to
every transition — the four deltas become ±45° and ±135°, and consecutive
symbols always land on *alternating* QPSK constellations, 45° apart. You see
eight points on a scope, but only four transitions carry information.

<figure class="lab-figure">
<svg viewBox="0 0 680 230" width="680" height="230" role="img" aria-label="A π/4-DQPSK constellation: eight points on a circle formed by two interleaved QPSK sets 45 degrees apart, one set drawn muted on the axes and the other set accented on the diagonals, with an arrow showing a plus-45-degree transition from an axis point to a diagonal point labelled dibit 00, and a second arrow showing a plus-135-degree transition labelled dibit 01">
  <circle cx="170" cy="112" r="82" fill="none" stroke="var(--fg-muted)" stroke-dasharray="3 4"/>
  <!-- set A (axes) -->
  <circle cx="252" cy="112" r="5" fill="var(--fg-muted)"/><circle cx="88" cy="112" r="5" fill="var(--fg-muted)"/>
  <circle cx="170" cy="30" r="5" fill="var(--fg-muted)"/><circle cx="170" cy="194" r="5" fill="var(--fg-muted)"/>
  <!-- set B (diagonals) -->
  <circle cx="228" cy="54" r="5" fill="var(--accent)"/><circle cx="112" cy="54" r="5" fill="var(--accent)"/>
  <circle cx="112" cy="170" r="5" fill="var(--accent)"/><circle cx="228" cy="170" r="5" fill="var(--accent)"/>
  <path d="M 248 104 A 82 82 0 0 0 234 60" fill="none" stroke="var(--accent)"/>
  <polygon points="238,66 231,56 242,58" fill="var(--accent)"/>
  <text x="268" y="76" fill="var(--accent)" font-size="10">+45° → dibit 00</text>
  <path d="M 246 122 A 82 82 0 0 1 122 178" fill="none" stroke="currentColor"/>
  <polygon points="130,172 118,180 128,182" fill="currentColor"/>
  <text x="212" y="212" fill="currentColor" font-size="10">+135° → dibit 01</text>
  <text x="440" y="52" fill="currentColor" font-size="11" font-weight="bold">two QPSK sets, 45° apart</text>
  <text x="440" y="74" fill="var(--fg-muted)" font-size="10">even symbols land on one set,</text>
  <text x="440" y="88" fill="var(--fg-muted)" font-size="10">odd symbols on the other — the</text>
  <text x="440" y="102" fill="var(--fg-muted)" font-size="10">trajectory never crosses the origin</text>
  <text x="440" y="130" fill="var(--fg-muted)" font-size="10">deltas: ±45°, ±135° — never 0°,</text>
  <text x="440" y="144" fill="var(--fg-muted)" font-size="10">so every symbol is a transition</text>
  <text x="440" y="158" fill="var(--fg-muted)" font-size="10">timing recovery can clock off</text>
  <text x="440" y="186" fill="var(--fg-muted)" font-size="10">a constant rotation of ALL points</text>
  <text x="440" y="200" fill="var(--fg-muted)" font-size="10">cancels in s·conj(last)</text>
</svg>
<figcaption>π/4-DQPSK: eight visible points from two interleaved QPSK sets, but only the four phase deltas carry dibits — and a constant rotation of everything cancels in the differential decode.</figcaption>
</figure>

The mapping from bit pairs to those deltas is TETRA's own Gray code, and
GopherTrunk keeps it in exactly one place:

```go
// internal/radio/tetra/sync.go (shape)
func TetraBitsToDibits(bits []uint8) []uint8 {
    out := make([]uint8, len(bits)/2)
    for i := range out {
        b1, b2 := bits[2*i]&1, bits[2*i+1]&1
        out[i] = (b1 << 1) | (b1 ^ b2) // TETRA Gray: 00→0, 01→1, 11→2, 10→3
    }
    return out
}
```

`TetraBitsToDibits` and its inverse are the single source of truth, kept
deliberately apart from the C4FM-linear `framing.DibitsToBits` — mixing the
conventions produces garbage that looks exactly like a demod bug. One residual
wrinkle: a leftover carrier-frequency offset advances the phase a constant
amount per symbol, which shows up as the whole *dibit stream* rotated by an
unknown 0..3. That's why every training-sequence correlator in Part 2 runs
under all four rotations.

## 144 kHz or nothing

GopherTrunk channelizes every protocol to a fixed per-protocol rate, and the
choice is made in one function:

```go
// internal/scanner/ccdecoder/ddc.go (shape)
func ddcTargetForProtocol(p trunking.Protocol) float64 {
    if p == trunking.ProtocolTETRA || p == trunking.ProtocolTETRADMO {
        return tetraDDCTargetRateHz // 144_000
    }
    return ddcTargetRateHz // 48_000 — the 4800-baud C4FM family
}
```

At 18000 sym/s, 144 kHz is exactly **8 samples per symbol**, and the receiver's
Gardner loop, AFC, channel filter and AGC are all sized from that figure. The
exported `DDCTargetForProtocol` exists so offline replay builds an *identical*
down-converter — sizing a TETRA replay to the 48 kHz C4FM default would leave
under 3 samples per symbol and the timing loop would never lock. The war story
of what happens when a capture arrives *below* the channel rate (a 50 kHz
recording that decoded to a plausible-but-wrong 16667 sym/s) is told in
[Protocol Decoders Part 7]({{ '/blog/deep-dives/protocol-decoders-07-tetra/' | relative_url }});
the standing rule is that the down-converter normalises in **both** directions
so the receiver always sees its designed rate.

## The receiver in one pass

`internal/radio/tetra/receiver` composes the chain: optional channel-select
filter (±15 kHz, because the 144 kHz passband is ±72 kHz and adjacent carriers
leak in — measured to cut on-air symbol errors by an order of magnitude), the
α = 0.35 RRC matched filter, symbol-timing recovery, AFC, then the differential
decode. Timing defaults to Gardner (`ParseClockMode("")` returns
`ClockGardner`), which the
[timing-recovery deep dive]({{ '/blog/deep-dives/sdr-internals-07-symbol-timing-sync-recovery/' | relative_url }})
covers in general form; the AFC runs *before* timing because a spinning
constellation corrupts the Gardner metric. The whole thing is stateful and
per-carrier: one `Receiver` per tuned frequency.

Two hooks in `receiver.Options` are worth noticing now even though we won't use
them for several parts: `SoftSink` emits the complex differential per symbol
(the soft information Part 8 builds on), and `SymbolSink` emits the raw
pre-differential symbols (the linear-channel domain the trained equalizer in
the later parts needs). Both are nil by default, cost nothing when unused, and
exist precisely because the differential product `s·conj(last)` destroys
information a linear equalizer requires — a consequence of this part's central
property, arriving early.

### How that principle shaped the Go code

- **One convention, one owner.** The Gray map lives only in
  `TetraBitsToDibits`/`TetraDibitsToBits`; the rotation constant lives only in
  `receiver.Rotation`. Nothing downstream re-derives either.
- **Rates are named, not implied.** `SymbolRate`, `tetraDDCTargetRateHz` and the
  8 sps product are constants with doc comments, so "the receiver's designed
  rate" is a greppable fact rather than folklore.
- **The demod is shared, parameterised.** `PiOver4DQPSK` serves both TETRA
  (rotation π/4) and P25 Phase 2 H-DQPSK (π/8) — the same primitive, one
  rotation argument apart, so a fix in the differential core lands in both.

## Where this goes next

A dibit stream at 18000/s is not yet a protocol — it's a firehose with no
punctuation. [Part 2]({{ '/blog/deep-dives/tetra-end-to-end-02-bursts-slot-grid/' | relative_url }})
adds the structure: the synchronisation and normal downlink bursts, the
training sequences that mark them, the 255-dibit slot grid — and the hard-won
lesson of `ndbSBSlotShift`, where anchoring that grid one slot wrong silently
misfiles every traffic burst on the carrier.

## FAQ

**Why differential modulation at all — why not coherent QPSK?**
Coherent QPSK needs a carrier-phase estimate, and a phase-recovery loop on a
fading mobile channel is a liability: every cycle slip is a burst of errors.
Differential encoding trades ~2–3 dB of sensitivity for never needing absolute
phase — the receiver works the moment two consecutive symbols arrive, which
also makes TDMA slot boundaries and late tune-ins cheap.

**Does GopherTrunk ever see the eight constellation points?**
Yes — the raw symbols exist between timing recovery and the differential
decode, and `SymbolSink` exposes exactly that stream. The hard decode only ever
looks at the transition, but the diagnostics panels and the trained equalizer
(later in this series) both consume the raw-symbol view.

**Why is the TETRA channel rate 144 kHz when the channel is only 25 kHz wide?**
144 kHz is a processing rate, not a bandwidth: 8 samples per symbol at
18000 sym/s. The occupied signal is ≈±12 kHz; the surplus passband is why the
receiver optionally inserts its own ±15 kHz channel-select filter before the
matched filter.

**What happens if a capture was recorded at 50 kHz instead?**
The down-converter interpolates it up to 144 kHz so the receiver still runs at
8 samples/symbol. Feeding it raw gives ~2.78 samples/symbol, which rounds to 3
and drifts the recovered clock ~7% — a plausible-looking symbol rate that never
locks. That story is in
[Protocol Decoders Part 7]({{ '/blog/deep-dives/protocol-decoders-07-tetra/' | relative_url }}).

**Is the 0..3 dibit rotation from carrier offset a bug?**
No — it's inherent. A residual CFO adds a constant phase per symbol, which the
quadrant slicer reads as a constant dibit offset. The AFC removes most of it,
and everything that correlates patterns (sync detectors, Part 2) searches all
four rotations so a residual rotation can't hide a burst.

## Series navigation

**Part 1 of 14** · Next →
[Part 2: The Burst Zoo & the Slot Grid]({{ '/blog/deep-dives/tetra-end-to-end-02-bursts-slot-grid/' | relative_url }})
