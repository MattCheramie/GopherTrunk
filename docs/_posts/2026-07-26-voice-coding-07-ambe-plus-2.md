---
title: "Voice Coding, Part 7: AMBE+2 — One Decoder, Two Rates"
description: How GopherTrunk's pure-Go AMBE+2 decoder unpacks 49 information bits into shared MBE parameters, and why P25 Phase 2, DMR, and NXDN all run through one Go decoder with two rate tables.
category: deep-dives
keywords: ambe+2 decoder, ambe 3600x2400, ambe 3600x2450, p25 phase 2 vocoder, dmr vocoder, nxdn vocoder, pure go ambe, mbe parameters, gophertrunk voice coding
tags: [ambe2, vocoder, mbe, dmr, p25, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "Voice Coding"
series_part: 7
---

*Part 7 of **Voice Coding**, a 12-part tour of how GopherTrunk turns
compressed digital-radio voice frames into PCM. Parts 2–6 built the MBE
model, IMBE decode, and acquisition. This post crosses the aisle to the
other half of the MBE family — AMBE+2 — and shows how one Go decoder
serves three protocols by swapping a table, not a codepath.*

> **TL;DR:** `internal/voice/ambe2` is the pure-Go AMBE+2 2400 bps decoder
> P25 Phase 2, DMR Tier II/III, and NXDN all use. It removes the CGO
> dependency on libmbe. Its job is the *front half* of the vocoder — unpack
> 49 information bits into nine quantization indices, run the codebook +
> inverse-DCT reconstruction, and project the result into the **shared**
> `mbe.Params` shape — then hand off to the exact same synthesis primitives
> IMBE uses. Two rate configurations (3600×2400 and 3600×2450) differ only
> in bit positions and tables; the synthesis core never knows which one ran.

**Key takeaways**

- AMBE+2 is the same **Multi-Band Excitation** algorithm family as
  [IMBE]({{ '/blog/deep-dives/voice-coding-04-imbe-decode/' | relative_url }}):
  8 kHz / 20 ms / 160-sample PCM built from voiced harmonics plus an
  FFT-shaped unvoiced excitation.
- One `Decoder` type serves **two rates**: the default `"ambe2"` (3600×2400,
  P25 P2 / NXDN) and `"ambe2-dmr"` (3600×2450, DMR). The only difference is
  which `unpack` function and codebook tables are wired in.
- The clever bit is `foldGammaIntoTl`: it rewrites the spectral residuals so
  the *IMBE-shaped* `mbe.PredictLog2Ml` produces spec-correct AMBE+2 output
  **without an AMBE+2-aware synthesis variant**.
- Every table value is machine-translated from szechyjs/mbelib's ISC-licensed
  constants; the algorithm's separate patent status is unchanged by a pure-Go
  re-implementation (see `docs/vocoders.md`).

## Cheat sheet

| Index | Bits | Role in the frame |
|---|---|---|
| `b0` | 7 | fundamental-frequency parameter → ω₀ + L; also the tone-frame flag (`b0 & 0x7E == 0x7E`) |
| `b1` | 4 | V/UV voicing-pattern row → `Vl[1..L]` |
| `b2` | 6 | gain delta → `DeltaGamma`; absolute γ resolved cross-frame |
| `b3` | 9 | PRBA index for `Gm[2..4]` |
| `b4` | 7 | PRBA index for `Gm[5..8]` |
| `b5..b7` | 4 each | HOC indices → higher `Cik` coefficients (bands 1–3) |
| `b8` | 3 | HOC index for band 4 (LSB forced 0) |

## In this post

- **What AMBE+2 is** and where it sits relative to IMBE.
- **The 49-bit unpack** — nine indices, a codebook lookup, and two DCTs.
- **One decoder, two rates** — how the 2400 and 2450 variants share a body.
- **Fold-into-Tl** — the design trick that reuses the IMBE synthesis core.

## Where AMBE+2 fits

The [Protocol Decoders]({{ '/blog/series/protocol-decoders/' | relative_url }})
series hands us a stream of *information bits* — the bare vocoder payload
after the upstream protocol layer (P25 Phase 2, DMR, or NXDN) has stripped its
own error-control coding. For AMBE+2 that payload is exactly **49 bits per
20 ms frame**. GopherTrunk packs those into 7 bytes (49 bits + 7 padding),
which is the same contract the old libmbe C wrapper accepted, so callers
didn't have to repack when the pure-Go decoder replaced it:

```go
// internal/voice/ambe2/decoder.go (shape)
const (
    InfoBits    = 49       // information bits per frame
    FrameBytes  = 7        // 49 bits + 7 padding
    VocoderName = "ambe2"  // 3600x2400 (P25 P2 / NXDN)
)
func (d *Decoder) FrameSize() int { return FrameBytes }
```

AMBE+2 and IMBE are cousins: both are Multi-Band Excitation vocoders, both
emit 160 int16 samples per frame by summing voiced cosine harmonics and an
overlap-added unvoiced excitation. What differs is the *bit format* — how the
handful of model parameters (pitch, voicing, spectral envelope, gain) are
quantized into the frame. So GopherTrunk splits the work: the AMBE+2 package
owns the front half (bits → parameters); the
[shared MBE package]({{ '/blog/deep-dives/voice-coding-03-mbe-synthesis/' | relative_url }})
owns the back half (parameters → PCM).

<figure class="lab-figure">
<svg viewBox="0 0 680 156" width="680" height="156" role="img" aria-label="A 49-bit AMBE+2 frame is unpacked into nine indices, run through codebook lookups and inverse DCTs to produce MBE parameters, then handed to the shared MBE synthesis core to produce PCM">
  <rect x="6" y="54" width="120" height="46" rx="6" fill="none" stroke="currentColor"/>
  <text x="66" y="74" text-anchor="middle" fill="currentColor" font-size="12">49 info bits</text>
  <text x="66" y="90" text-anchor="middle" fill="var(--fg-muted)" font-size="10">7-byte frame</text>
  <line x1="126" y1="77" x2="172" y2="77" stroke="currentColor"/>
  <polygon points="172,73 182,77 172,81" fill="currentColor"/>
  <rect x="182" y="42" width="150" height="70" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="257" y="64" text-anchor="middle" fill="var(--accent)" font-size="12">UnpackParams</text>
  <text x="257" y="80" text-anchor="middle" fill="var(--fg-muted)" font-size="10">b0..b8 → tables</text>
  <text x="257" y="96" text-anchor="middle" fill="var(--fg-muted)" font-size="10">inverse DCTs → Tl</text>
  <line x1="332" y1="77" x2="378" y2="77" stroke="currentColor"/>
  <polygon points="378,73 388,77 378,81" fill="currentColor"/>
  <rect x="388" y="54" width="128" height="46" rx="6" fill="none" stroke="currentColor"/>
  <text x="452" y="74" text-anchor="middle" fill="currentColor" font-size="12">mbe.Params</text>
  <text x="452" y="90" text-anchor="middle" fill="var(--fg-muted)" font-size="10">W0, L, Vl, Tl</text>
  <line x1="516" y1="77" x2="562" y2="77" stroke="currentColor"/>
  <polygon points="562,73 572,77 562,81" fill="currentColor"/>
  <rect x="572" y="54" width="102" height="46" rx="6" fill="none" stroke="currentColor"/>
  <text x="623" y="74" text-anchor="middle" fill="currentColor" font-size="12">shared MBE</text>
  <text x="623" y="90" text-anchor="middle" fill="var(--fg-muted)" font-size="10">→ 160 PCM</text>
  <text x="340" y="140" text-anchor="middle" fill="var(--fg-muted)" font-size="10">AMBE+2 owns the front half; the mbe package owns synthesis — the same core IMBE uses</text>
</svg>
<figcaption>The decoder's entire job is the leftmost box: turn 49 bits into the shared parameter shape. Everything right of it is code IMBE already exercises.</figcaption>
</figure>

## The 49-bit unpack

`UnpackParams` reads the 49 bits into nine indices, `b0` through `b8`, each
drawn from a *scattered* bit layout (the AMBE+2 bit-interleave is not
contiguous — bit 48 is the low bit of `b0`, for instance). From there it is
table lookups and DCTs:

```go
// internal/voice/ambe2/params.go (shape)
b0 := int(info[0])<<6 | ... | int(info[48]) // scattered layout
f0 := math.Pow(2, -4.311767578125-2.1336e-2*(float64(b0)+0.5))
w0 := f0 * 2 * math.Pi
unvc := 0.2046 / math.Sqrt(w0)              // unvoiced amplitude scale
L := int(AmbePlusLtable[b0])                // harmonic count, 9..56
```

- **`b0`** sets the fundamental frequency `ω₀` and, via `AmbePlusLtable`, the
  harmonic count `L`. It also doubles as the tone-frame flag.
- **`b1`** selects a row of `AmbePlusVuv`, the voicing-pattern table, that
  paints each harmonic voiced or unvoiced (`Vl[l]`).
- **`b2`** indexes `AmbePlusDg` for the per-frame gain delta.
- **`b3`, `b4`** are PRBA (Predictive Residual Block Average) indices that
  seed an inverse 8-point DCT-II, producing the first two DCT coefficients of
  each of four spectral bands.
- **`b5`–`b8`** are HOC (Higher-Order Coefficient) indices that fill in the
  remaining per-band coefficients.

A per-band inverse DCT then expands those coefficients into the `L` spectral
residuals `Tl[1..L]` — the log-amplitude envelope the synthesizer reconstructs
harmonics from. The whole flow mirrors szechyjs/mbelib's
`mbe_decodeAmbe2400Parms` 1:1, deliberately, so any codebook drift between the
two implementations stays detectable in the test suite.

## One decoder, two rates

Here is the design payoff. AMBE+2 ships in two flavours GopherTrunk cares
about: **3600×2400** (P25 Phase 2, NXDN) and **3600×2450** (DMR). The "3600"
is the on-air channel rate including the protocol's FEC; the "2400"/"2450" is
the vocoder payload rate. Both still arrive here as 49 information bits — the
difference is purely *which bits mean what* and *which codebook tables* decode
them.

GopherTrunk models that as a single `Decoder` struct whose only per-variant
state is a function pointer and a name:

```go
// internal/voice/ambe2/decoder.go (shape)
type Decoder struct {
    // ... shared synthesis state (SynthState, AGC, DCBlock, ...) ...
    unpack func([]byte) (Params, error) // UnpackParams or unpackParams2450
    name   string
}

func New() *Decoder    { d := ...; d.unpack = UnpackParams;      d.name = "ambe2";     return d }
func NewDMR() *Decoder { d := New(); d.unpack = unpackParams2450; d.name = "ambe2-dmr"; return d }
```

`unpackParams2450` is a near-copy of `UnpackParams` — same PRBA→DCT→HOC→Tl
reconstruction, different `b0..b8` bit positions and different tables
(`dmrLtable`, `dmrVuv`, `dmrDg`, `dmrPRBA24/58`, `dmrHOCb5..b8`). Everything
after the unpack — the gamma fold, `mbe.PredictLog2Ml`, enhancement, voiced +
unvoiced synthesis, the AGC — is byte-for-byte identical between the two.

<figure class="lab-figure">
<svg viewBox="0 0 640 178" width="640" height="178" role="img" aria-label="Two unpack functions, one for 3600x2400 and one for 3600x2450, both feed a single shared synthesis body, with the decoder selecting the unpack function at construction time">
  <rect x="24" y="24" width="170" height="44" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="109" y="44" text-anchor="middle" fill="var(--accent)" font-size="12">UnpackParams</text>
  <text x="109" y="59" text-anchor="middle" fill="var(--fg-muted)" font-size="10">3600x2400 · P25 P2 / NXDN</text>
  <rect x="24" y="86" width="170" height="44" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="109" y="106" text-anchor="middle" fill="var(--accent)" font-size="12">unpackParams2450</text>
  <text x="109" y="121" text-anchor="middle" fill="var(--fg-muted)" font-size="10">3600x2450 · DMR</text>
  <line x1="194" y1="46" x2="286" y2="70" stroke="var(--fg-muted)"/>
  <polygon points="284,65 294,71 283,75" fill="var(--fg-muted)"/>
  <line x1="194" y1="108" x2="286" y2="84" stroke="var(--fg-muted)"/>
  <polygon points="283,79 294,83 284,89" fill="var(--fg-muted)"/>
  <rect x="296" y="52" width="150" height="50" rx="6" fill="none" stroke="currentColor"/>
  <text x="371" y="72" text-anchor="middle" fill="currentColor" font-size="12">shared body</text>
  <text x="371" y="88" text-anchor="middle" fill="var(--fg-muted)" font-size="10">fold · synth · AGC</text>
  <line x1="446" y1="77" x2="492" y2="77" stroke="currentColor"/>
  <polygon points="492,73 502,77 492,81" fill="currentColor"/>
  <rect x="502" y="52" width="112" height="50" rx="6" fill="none" stroke="currentColor"/>
  <text x="558" y="80" text-anchor="middle" fill="currentColor" font-size="12">160 PCM</text>
  <text x="320" y="150" text-anchor="middle" fill="var(--fg-muted)" font-size="10">the decoder picks its unpack func at construction; the body never branches on rate</text>
</svg>
<figcaption>The rate variants fork only at the unpack function. Selecting one is a constructor decision, so the hot synthesis path has zero rate branching.</figcaption>
</figure>

### How that principle shaped the Go code

The temptation with two rates is two decoders, or a rate flag threaded through
every synthesis function. GopherTrunk avoids both. But there's a subtler
problem: the shared `mbe.PredictLog2Ml` was written for **IMBE's** parameter
form, which folds gain into the amplitudes differently than the AMBE+2 spec.
The AMBE+2 spec form is:

```
log2Ml[l] = γ·(prev_interp[l] − mean_prev_interp)
            + (Tl[l] − mean(Tl))
            + (γ_abs − 0.5·log2(L))
```

Rather than write an AMBE+2-aware predictor, the decoder **pre-folds** the
gamma and mean-removal into `Tl` up front, so the IMBE-shaped predictor
produces the right answer unchanged:

```go
// internal/voice/ambe2/decoder.go (shape)
func foldGammaIntoTl(p Params, gamma float64) mbe.Params {
    meanTl  := mean(p.Tl[1:p.L+1])
    bigGamma := gamma - 0.5*math.Log2(float64(p.L))
    folded  := p.Params
    for l := 1; l <= p.L; l++ {
        folded.Tl[l] = p.Tl[l] - meanTl + bigGamma
    }
    return folded
}
```

The absolute gain itself is the one genuinely cross-frame quantity AMBE+2 has
that IMBE lacks: `γ_curr = DeltaGamma + 0.5·γ_prev`. The decoder caches
`prevGamma` and resolves it before folding. This is the whole reason the
package is "the front half plus a fold" — that ~10-line function is what lets
two vocoder families share one synthesizer.

## The unvoiced scale, and tones

Two AMBE+2-specific touches ride along. First, unvoiced harmonics are
attenuated by `Unvc = 0.2046/√ω₀` before enhancement — GopherTrunk applies it
in the same place mbelib does, between the log2→linear amplitude conversion and
the §6.2 enhancement. Second, `b0 ∈ {0x7E, 0x7F}` marks a **tone frame** rather
than voice: single tones (`b1·31.25 Hz`), DTMF dual-tones from the ITU-T Q.23
keypad matrix, and vendor-specific "knox" pairs. Those get their own oscillator
path — the subject of
[Part 8]({{ '/blog/deep-dives/voice-coding-08-ambe-plus-2-fec-knox/' | relative_url }}).

## Where this goes next

[Part 8]({{ '/blog/deep-dives/voice-coding-08-ambe-plus-2-fec-knox/' | relative_url }})
takes on the error-control side of AMBE+2: how a bad frame is replayed with
progressive attenuation, how the FEC corrected-bit count drives adaptive
smoothing, and what the vendor-specific "knox" tone path actually is (and
honestly, what it isn't). After that,
[Part 9]({{ '/blog/deep-dives/voice-coding-09-the-composer/' | relative_url }})
zooms out to the composer that wires demodulated frames into whichever vocoder
a protocol needs.

## FAQ

**Is AMBE+2 the same as AMBE or IMBE?**
They are the same *family* — all Multi-Band Excitation vocoders that model
speech as voiced harmonics plus unvoiced noise. IMBE is P25 Phase 1's 4400 bps
codec; AMBE+2 is the newer 2400 bps codec used by P25 Phase 2, DMR, and NXDN.
GopherTrunk decodes both in pure Go over a shared synthesis core.

**What is the difference between 3600×2400 and 3600×2450?**
Both carry 49 information bits per 20 ms frame, so they are the same *rate* to
the vocoder; the "3600" is the on-air channel rate with FEC. The variants
differ only in bit layout and codebook tables — P25 Phase 2 and NXDN use
3600×2400 (`"ambe2"`), DMR uses 3600×2450 (`"ambe2-dmr"`).

**Does the pure-Go decoder avoid AMBE patents?**
No — re-implementing the algorithm in Go doesn't change its patent status. The
ISC license covers the *table values* (facts about the published algorithm),
not the algorithm itself. Operators in licence-restrictive jurisdictions should
evaluate before deploying; see `docs/vocoders.md`.

**Why fold gamma into Tl instead of writing an AMBE+2 predictor?**
Because it lets AMBE+2 and IMBE share one synthesis core. `foldGammaIntoTl`
rewrites the spectral residuals so the IMBE-shaped `mbe.PredictLog2Ml` yields
spec-correct AMBE+2 output — one small function instead of a parallel
synthesis pipeline.

## Series navigation

**Part 7 of 12** · ←
[Part 6: Acquisition & Squelch]({{ '/blog/deep-dives/voice-coding-06-acquisition-squelch/' | relative_url }})
· Next →
[Part 8: AMBE+2 FEC & the Knox Path]({{ '/blog/deep-dives/voice-coding-08-ambe-plus-2-fec-knox/' | relative_url }})
