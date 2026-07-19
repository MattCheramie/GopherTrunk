---
title: "Protocol Decoders, Part 7: TETRA — π/4-DQPSK, Channel Coding & the Sub-Target-Rate Fix"
description: How GopherTrunk decodes TETRA — π/4-DQPSK framing, the type-1 to type-5 channel-coding chain, CMCE grant PDUs, and the recent fix that interpolates a narrowband capture up to the 144 kHz channel rate so the receiver locks.
category: deep-dives
keywords: tetra decoder, pi/4 dqpsk demodulation, tetra channel coding, cmce d-connect grant, tetra 18000 symbols per second, sub target rate resample, tetra sync training sequence, gophertrunk tetra
tags: [tetra, dsp, decoding, go, dqpsk, protocol]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "Protocol Decoders"
series_part: 7
---

*Part 7 of **Protocol Decoders**, a 12-part tour of how GopherTrunk turns
symbols into structured messages. So far the series has stayed inside the C4FM
family — four-level FSK, one dibit per symbol, a linear bit convention shared by
P25, DMR, NXDN and dPMR. TETRA breaks every one of those assumptions, so it gets
its own episode. It's also the first place the **Mercury** thread from
[Part 1]({{ '/blog/deep-dives/protocol-decoders-01-anatomy-of-a-cc-decoder/' | relative_url }})
teaches a lesson we'll need later: a decoder that produces a plausible-but-wrong
symbol rate is worse than one that produces nothing.*

> **TL;DR:** TETRA is π/4-DQPSK at **18000 symbols/s**, one dibit per symbol,
> with its own Gray mapping distinct from the C4FM linear convention. Bursts are
> found by correlating a 19-dibit synchronisation training sequence, sliced, then
> run through a type-1 → type-5 channel-coding chain (CRC-16 + tail + RCPC R=2/3
> + block interleave + colour-code scramble) before the CMCE layer yields grants.
> The receiver is designed for **8 samples/symbol at a 144 kHz channel rate** — and
> the recent fix (#926/#921) makes a narrowband capture recorded *below* that rate
> decode by interpolating it **up** to 144 kHz instead of feeding the receiver a
> broken 2.78 samples/symbol.

**Key takeaways**

- TETRA's bit↔dibit mapping is **not** the linear C4FM one; `TetraBitsToDibits`
  is the single source of truth and is deliberately separate from
  `framing.DibitsToBits`.
- Sync is a **correlation** against real ETSI training sequences (not magic hex
  constants) — the wrong constants meant the CC could never lock on air (#553).
- The channel-coding chain is a strict pipeline of framing primitives; each
  logical channel (AACH, BSCH, SCH/HD, SCH/HU, SCH/F) is the same chain with
  different interleaver sizes.
- The receiver is **rate-invariant by design** at 8 samples/symbol — so a
  capture *below* the channel rate must be normalised **up**, not fed raw.

## Cheat sheet

| Concept | Where it lives | One-line role |
|---|---|---|
| Bit↔dibit mapping | `TetraBitsToDibits` (`sync.go`) | TETRA Gray map `(b1,b2)→(b1<<1)|(b1^b2)` |
| Training sequences | `SyncTrainingSeq` etc. (`sync.go`) | real ETSI EN 300 392-2 §9.4.4.3 bit arrays |
| Sync detector | `SyncDetector` (`sync.go`) | slides a window, reports match indices within tolerance |
| Channel coding | `signalingDecode` (`channel_coding.go`) | descramble → deinterleave → depuncture → Viterbi → CRC |
| CMCE grant | `AsVoiceGrant` (`cmce.go`) | D-CONNECT / D-TX-GRANTED → `VoiceGrant` |
| Control state | `ControlChannel` (`control.go`) | PDUs in, `cc.locked` + `KindGrant` out |

## In this post

- **π/4-DQPSK and the dibit convention** — why TETRA gets its own mapping.
- **Finding a burst** — correlating the synchronisation training sequence.
- **The channel-coding chain** — type-1 to type-5, one pipeline, five channels.
- **CMCE** — turning a decoded PDU into a `VoiceGrant`.
- **The problem we hit** — a 50 kHz capture that produced 16667 sym/s and never
  locked, and the fix that normalises up to 144 kHz.

## π/4-DQPSK and why TETRA gets its own dibit convention

Every C4FM protocol in this series — P25 Phase 1, DMR, NXDN, dPMR — is four-level
FSK: the demodulator reads an amplitude and maps it to one of four dibit values
with a single linear convention (`framing.DibitsToBits`). TETRA is **differential
QPSK with a π/4 rotation** instead. Information rides in the *phase change*
between symbols, and the constellation rotates 45° every symbol so there's always
a transition to clock off. The line rate is **18000 symbols/s**, one dibit per
symbol.

The catch is the bit-to-dibit mapping. TETRA transmits bit pairs `(b1,b2)` that
π/4-DQPSK modulates to a differential phase the demodulator recovers as a dibit
*value* 0..3 using the TETRA Gray map — `00→0, 01→1, 11→2, 10→3`:

```go
// internal/radio/tetra/sync.go (shape)
func TetraBitsToDibits(bits []uint8) []uint8 {
    out := make([]uint8, len(bits)/2)
    for i := range out {
        b1, b2 := bits[2*i]&1, bits[2*i+1]&1
        out[i] = (b1 << 1) | (b1 ^ b2) // TETRA Gray, NOT the C4FM linear map
    }
    return out
}
```

That one line is the reason TETRA can't reuse the C4FM helpers. `TetraBitsToDibits`
and its inverse `TetraDibitsToBits` are the **single source of truth** for the
convention, kept deliberately distinct from `framing.DibitsToBits`. Mix them up
and every training-sequence correlation and every channel decode silently
produces garbage — the kind of bug that looks like a demod problem for a week.

<figure class="lab-figure">
<svg viewBox="0 0 680 150" width="680" height="150" role="img" aria-label="TETRA processing chain from IQ through pi/4-DQPSK demodulation to dibits, sync correlation, channel decode, and CMCE PDU dispatch">
  <rect x="6" y="52" width="104" height="46" rx="6" fill="none" stroke="currentColor"/>
  <text x="58" y="72" text-anchor="middle" fill="currentColor" font-size="12">IQ @ 144 kHz</text>
  <text x="58" y="88" text-anchor="middle" fill="var(--fg-muted)" font-size="10">8 samp/sym</text>
  <line x1="110" y1="75" x2="146" y2="75" stroke="currentColor"/>
  <polygon points="146,71 156,75 146,79" fill="currentColor"/>
  <rect x="156" y="52" width="108" height="46" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="210" y="72" text-anchor="middle" fill="var(--accent)" font-size="12">π/4-DQPSK</text>
  <text x="210" y="88" text-anchor="middle" fill="var(--fg-muted)" font-size="10">→ dibits</text>
  <line x1="264" y1="75" x2="300" y2="75" stroke="currentColor"/>
  <polygon points="300,71 310,75 300,79" fill="currentColor"/>
  <rect x="310" y="52" width="108" height="46" rx="6" fill="none" stroke="currentColor"/>
  <text x="364" y="72" text-anchor="middle" fill="currentColor" font-size="12">sync detect</text>
  <text x="364" y="88" text-anchor="middle" fill="var(--fg-muted)" font-size="10">19-dibit STS</text>
  <line x1="418" y1="75" x2="454" y2="75" stroke="currentColor"/>
  <polygon points="454,71 464,75 454,79" fill="currentColor"/>
  <rect x="464" y="52" width="108" height="46" rx="6" fill="none" stroke="currentColor"/>
  <text x="518" y="72" text-anchor="middle" fill="currentColor" font-size="12">channel decode</text>
  <text x="518" y="88" text-anchor="middle" fill="var(--fg-muted)" font-size="10">type-5 → type-1</text>
  <line x1="572" y1="75" x2="608" y2="75" stroke="currentColor"/>
  <polygon points="608,71 618,75 608,79" fill="currentColor"/>
  <rect x="618" y="52" width="56" height="46" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="646" y="72" text-anchor="middle" fill="var(--accent)" font-size="11">CMCE</text>
  <text x="646" y="88" text-anchor="middle" fill="var(--fg-muted)" font-size="10">grant</text>
</svg>
<figcaption>The TETRA control-channel path: demodulate to dibits, correlate the synchronisation training sequence, slice and channel-decode the burst, then dispatch the CMCE PDU.</figcaption>
</figure>

## Finding a burst: correlating the training sequence

A TETRA downlink burst carries a fixed training sequence the receiver correlates
against to find frame boundaries. GopherTrunk stores the **real ETSI EN 300 392-2
§9.4.4.3 sequences** as MSB-first bit arrays:

```go
// internal/radio/tetra/sync.go (shape)
// Synchronisation training sequence (§9.4.4.3.4) — 38 bits / 19 dibits.
SyncTrainingSeq = []uint8{1,1,0,0,0,0,0,1,1,0,0,1,1,1,0,0,1,1,1,0,
                          1,0,0,1,1,1,0,0,0,0,0,1,1,0,0,1,1,1}
```

This matters more than it looks. The previous constants were placeholder `uint64`
"hex" values that were both truncated (a uint64 holds 64 bits, but the sequences
are up to 76) and matched no spec value — so the control channel could correlate
against them all day and never lock on real air (#553). The lengths are exact:
Normal training is 22 bits / 11 dibits, Extended is 30 / 15, and the
Synchronisation sequence is 38 / 19.

`SyncDetector` slides a window over the dibit stream and reports every index where
the pattern matches within a mismatch `tolerance`, using the same ring-buffer
shape as the Phase 1 / DMR / NXDN detectors so the higher-level state machine
stays consistent across protocols. On the SB (synchronisation) burst the adapter
runs **four** correlators — one per π/4-DQPSK constellation rotation — because a
residual carrier offset rotates the whole dibit stream by an unknown 0..3 and a
single fixed-orientation correlator would miss the burst.

## The channel-coding chain: type-1 to type-5

Once a burst is sliced, the bits are still deep inside TETRA's FEC. ETSI names the
stages type-1 (information) through type-5 (on-air), and GopherTrunk composes them
from framing primitives into one pipeline. Encoding runs:

```go
// internal/radio/tetra/channel_coding.go (shape)
func signalingEncode(type1 []byte, kInter, aInter int, colour uint32) []byte {
    withCRC := appendCRC16(type1)        // K1 + 16  (CRC-CCITT/0x1021)
    type2 := appendTailBits(withCRC, 4)  // K1 + 20  (4 zero tail bits)
    type3 := encodeRCPCRate23(type2)     // × 3/2    (RCPC mother + puncture)
    type4 := framing.BlockInterleaveTetra(type3, kInter, aInter)
    type5 := framing.ScrambleTetra(type4, colour) // colour-code scramble
    return type5
}
```

Decoding is the mirror image: descramble, deinterleave, depuncture + Viterbi,
strip the tail, and verify the CRC-16 (`signalingDecode` returns the info bits
plus a pass/fail flag — a failing CRC means the bits are the best Viterbi guess
but must not be trusted). The CRC is the textbook CRC-CCITT-FALSE: polynomial
`0x1021`, init `0xFFFF`, final XOR `0xFFFF`.

Every logical channel is *this same chain* with different interleaver sizes:

| Channel | type-1 → type-5 | Notes |
|---|---|---|
| AACH | 14 → 30 | RM(30,14) + scramble only — skips RCPC + interleave |
| BSCH | 60 → 120 | colour code forced to 0 per §8.2.5.2 |
| SCH/HD | 124 → 216 | also covers BNCH + STCH (same coding) |
| SCH/HU | 92 → 168 | half-slot uplink |
| SCH/F | 268 → 432 | full-slot signaling |

There's a soft-decision variant of each (`signalingDecodeSoft`, `DecodeSCHHDSoft`)
that runs the same chain on per-bit LLRs from the demodulator — worth a few dB on
a marginal cell — but the hard path is the one to understand first.

### How that principle shaped the Go code

TETRA channel coding is a textbook case for *composition over a monolith*. The
RCPC mother code, the puncture table, the (30,14) Reed-Muller block code, the
block interleaver and the scrambler all live in `internal/radio/framing` and are
unit-tested in isolation. `channel_coding.go` never reimplements any of them — it
wires them into `signalingEncode` / `signalingDecode` and expresses each channel
as a one-line call with the right interleaver constants. The payoff is that the
five channels are *provably* the same pipeline: there's no per-channel copy of the
Viterbi decoder to drift out of sync. It's the same discipline as the CC decoder
registry from
[Part 1]({{ '/blog/deep-dives/protocol-decoders-01-anatomy-of-a-cc-decoder/' | relative_url }}) —
build the hard part once, parameterise the rest.

## CMCE: from PDU to grant

The recovered info bits are a Layer-3 PDU. The trunking-relevant ones live in the
CMCE sub-protocol, and `AsVoiceGrant` pulls the channel assignment out of a
D-CONNECT (or the late-grant D-TX-GRANTED, which shares the layout):

```go
// internal/radio/tetra/cmce.go (shape)
func (p PDU) AsVoiceGrant() (VoiceGrant, bool) {
    if !p.IsCMCE() { return VoiceGrant{}, false }
    switch PDUType(p.Type) {
    case CMCEDConnect, CMCEDTxGranted: // both carry the channel-alloc element
    default: return VoiceGrant{}, false
    }
    // ... Source SSI (24b), Dest SSI (24b), Carrier Number (12b), Timeslot (2b)
}
```

`ControlChannel.Ingest` glues it to the engine: a valid MLE-SYSINFO (carrying the
MCC/MNC/Location-Area that identify the cell) or any non-idle CMCE PDU declares
`cc.locked`, and a voice grant is republished as an `events.KindGrant` with a
`trunking.Grant{Protocol: "tetra"}` payload — the exact shape the
[Trunking Engine]({{ '/blog/series/trunking-engine/' | relative_url }}) consumes,
so from the engine's side TETRA looks like any other trunked system. The carrier
number is resolved to an actual frequency through a band-plan `Resolver`, because
TETRA grants name a *carrier number*, not a frequency.

## The problem we hit: a capture below the channel rate

Here's the war story. TETRA's receiver is built for **8 samples/symbol**: at
18000 sym/s that's a **144 kHz** channel rate, and the Gardner timing loop, AGC
and matched filter are all sized from that. GopherTrunk's replay/analyze DDC has
always *decimated* wideband SDR captures down to 144 kHz cleanly.

But people record TETRA at the *narrowband* rates their SDR software offers —
48 kHz or 50 kHz, the natural SDR++/SDRTrunk channel-record rates. A capture
already at or below 144 kHz was fed to the receiver **raw**. At 50 kHz that's
50000/18000 ≈ **2.78 samples/symbol**, which the receiver rounded to 3 and handed
the Gardner loop as a nominal 3.0 — an ~8% clock error it simply can't pull in. It
produced 50000/3 ≈ **16667 sym/s** (−7.4%) and never locked. A plausible-looking
symbol rate, and completely wrong.

The fix (#926) makes the down-converter *normalise* to the channel rate in
**both** directions. A wideband capture still decimates down exactly as before; a
sub-target capture is now interpolated **up** to 144 kHz (`dsp.Resampler` already
supports L>M), restoring the designed 8 samples/symbol. A 50 kHz capture now
recovers **18004.9 sym/s (+0.0%)** and the training sequences correlate; the
committed 48 kHz `samples/tetra` reference normalises to exactly 144000 Hz.

<figure class="lab-figure">
<svg viewBox="0 0 680 190" width="680" height="190" role="img" aria-label="Before and after the sub-target-rate fix: a 50 kHz capture fed raw gives 2.78 samples per symbol rounded to 3 and 16667 symbols per second with no lock, while normalising up to 144 kHz gives 8 samples per symbol and 18005 symbols per second locked">
  <text x="20" y="24" fill="var(--fg-muted)" font-size="12" font-weight="bold">Before (#926)</text>
  <rect x="20" y="34" width="96" height="40" rx="6" fill="none" stroke="currentColor"/>
  <text x="68" y="52" text-anchor="middle" fill="currentColor" font-size="11">50 kHz cap</text>
  <text x="68" y="67" text-anchor="middle" fill="var(--fg-muted)" font-size="10">fed raw</text>
  <line x1="116" y1="54" x2="150" y2="54" stroke="currentColor"/>
  <polygon points="150,50 160,54 150,58" fill="currentColor"/>
  <rect x="160" y="34" width="150" height="40" rx="6" fill="none" stroke="currentColor"/>
  <text x="235" y="52" text-anchor="middle" fill="currentColor" font-size="11">2.78 → round to 3</text>
  <text x="235" y="67" text-anchor="middle" fill="var(--fg-muted)" font-size="10">samples/symbol</text>
  <line x1="310" y1="54" x2="344" y2="54" stroke="currentColor"/>
  <polygon points="344,50 354,54 344,58" fill="currentColor"/>
  <rect x="354" y="34" width="150" height="40" rx="6" fill="none" stroke="#c0392b"/>
  <text x="429" y="52" text-anchor="middle" fill="#c0392b" font-size="11">16667 sym/s (−7.4%)</text>
  <text x="429" y="67" text-anchor="middle" fill="#c0392b" font-size="10">no lock</text>
  <text x="20" y="112" fill="var(--fg-muted)" font-size="12" font-weight="bold">After (#926)</text>
  <rect x="20" y="122" width="96" height="40" rx="6" fill="none" stroke="currentColor"/>
  <text x="68" y="140" text-anchor="middle" fill="currentColor" font-size="11">50 kHz cap</text>
  <text x="68" y="155" text-anchor="middle" fill="var(--fg-muted)" font-size="10">interpolate ↑</text>
  <line x1="116" y1="142" x2="150" y2="142" stroke="var(--accent)"/>
  <polygon points="150,138 160,142 150,146" fill="var(--accent)"/>
  <rect x="160" y="122" width="150" height="40" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="235" y="140" text-anchor="middle" fill="var(--accent)" font-size="11">144 kHz · 8 samp/sym</text>
  <text x="235" y="155" text-anchor="middle" fill="var(--fg-muted)" font-size="10">normalised up</text>
  <line x1="310" y1="142" x2="344" y2="142" stroke="var(--accent)"/>
  <polygon points="344,138 354,142 344,146" fill="var(--accent)"/>
  <rect x="354" y="122" width="150" height="40" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="429" y="140" text-anchor="middle" fill="var(--accent)" font-size="11">18005 sym/s (+0.0%)</text>
  <text x="429" y="155" text-anchor="middle" fill="var(--accent)" font-size="10">locks</text>
</svg>
<figcaption>The same 50 kHz recording: fed raw it rounds to 3 samples/symbol and drifts to 16667 sym/s; normalised up to 144 kHz it recovers the receiver's designed 8 samples/symbol and locks.</figcaption>
</figure>

A companion fix (#921) closed the *other* end of the same problem. When a wideband
capture reduced to a ratio beyond the resampler's L/M caps, the DDC fell back to a
crude integer decimator that missed the target by a few percent — a 3.019 MS/s
stream landed the 144 kHz channel at 143762 Hz (17970 sym/s, −0.165%), another
decode-breaking clock error. It now falls back to the closest L/M *under* the caps
using the same bounded search the wideband tuner already used, landing the same
stream at 143998 Hz (17999.8 sym/s). The lesson, straight from the
[DSP notes in the repo guidance]({{ '/blog/deep-dives/protocol-decoders-01-anatomy-of-a-cc-decoder/' | relative_url }}):
the decode path is *rate-invariant to the capture rate only because* the
down-converter always delivers the receiver its designed samples/symbol. Break
that contract and you get a symbol clock that's off by exactly the amount you
short-changed the resampler.

## Where this goes next

[Part 8]({{ '/blog/deep-dives/protocol-decoders-08-edacs-ltr-mpt1327/' | relative_url }})
steps *back* in time to the legacy trunking family — EDACS, LTR and MPT-1327 —
where the control channel isn't a clean digital PDU stream at all but sub-audible
tones and embedded signalling. TETRA was the last of the modern digital
protocols; the contrast with the old schemes is the whole point of the next post.
For the standard itself, the [TETRA reference]({{ '/reference/tetra/' | relative_url }})
and [TETRA encryption (TEA)]({{ '/reference/tetra-tea/' | relative_url }}) pages
go deeper on the air interface and its ciphers.

## FAQ

**Why does TETRA need its own dibit mapping instead of reusing the C4FM one?**
Because it's a different modulation. The C4FM protocols are four-level FSK with a
linear amplitude-to-dibit map; TETRA is π/4-DQPSK where the dibit value comes from
a differential phase and a specific Gray code. `TetraBitsToDibits` encodes that
Gray map and is kept separate from `framing.DibitsToBits` so the two conventions
can never be confused.

**What is the difference between the type-1 and type-5 bits?**
They're ETSI's names for stages of the channel-coding chain. Type-1 is the raw
information block; type-5 is what's actually transmitted on air after CRC, tail
bits, RCPC convolutional coding, interleaving and scrambling. Decoding walks type-5
back to type-1 and verifies the CRC.

**Why did my 50 kHz TETRA recording decode to a wrong symbol rate?**
Older builds fed a sub-144-kHz capture to the receiver raw, giving ~2.78
samples/symbol that rounded to 3 and drifted the symbol clock to ~16667 sym/s with
no lock. Current builds (#926) interpolate the capture up to the 144 kHz channel
rate first, restoring 8 samples/symbol so it locks at ~18005 sym/s.

**How does a TETRA grant reach the trunking engine?**
`ControlChannel.Ingest` parses a CMCE D-CONNECT into a `VoiceGrant`, resolves the
carrier number to a frequency via the band-plan resolver, and publishes an
`events.KindGrant` with a `trunking.Grant{Protocol: "tetra"}` payload — identical
in shape to every other protocol's grant, so the engine treats TETRA generically.

## Series navigation

**Part 7 of 12** · ←
[Part 6: NXDN & dPMR]({{ '/blog/deep-dives/protocol-decoders-06-nxdn-dpmr/' | relative_url }})
· Next →
[Part 8: EDACS, LTR & MPT-1327]({{ '/blog/deep-dives/protocol-decoders-08-edacs-ltr-mpt1327/' | relative_url }})
