---
title: "Voice Coding, Part 4: IMBE — Decoding P25 Phase 1 Voice"
description: How GopherTrunk turns 88 IMBE information bits into MBE parameters — the b_0 pitch codeword, the PRBS scrambler that whitens the spectrum, and the table-driven PRBA and DCT quantization that reconstruct the spectral envelope.
category: deep-dives
keywords: imbe decoder, p25 phase 1 voice, imbe 4400 frame, b_0 fundamental frequency, imbe scrambler prbs, prba gain blocks, imbe quantization tables, gophertrunk imbe
tags: [voice, imbe, p25, dsp, decoding, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "Voice Coding"
series_part: 4
---

*Part 4 of **Voice Coding**. Parts 2 and 3 built the shared MBE core — the
parameter set and the synthesis that grows speech from it. Now we start filling
that struct in for a real codec. This post takes the 88 information bits of a P25
Phase 1 IMBE frame and turns them into the `mbe.Params` the synthesizer wants.*

> **TL;DR:** An IMBE 4400 frame carries **88 information bits** over 20 ms. Decode
> unpacks the **`b_0` pitch codeword** first — from scattered bit positions — and
> derives ω₀, the harmonic count `L`, and the voicing-decision count `K` from it.
> The remaining 79 bits are re-ordered through a table, then decoded into voicing
> flags, six **PRBA gain blocks**, and per-band **DCT spectral coefficients**.
> Two inverse DCTs turn those into the spectral residuals `Tl[1..L]`. A
> `b_0`-keyed **PRBS scrambler** has to be undone first, because P25 whitens the
> on-air bits. The output is `imbe.Params`, projected down to `mbe.Params`.

**Key takeaways**

- The pitch parameter `b_0` lives at scattered bit positions `{0..5, 85, 86}` and
  drives everything: `ω₀ = 4π/(b_0 + 39.5)`, then `L` and `K` derive from ω₀.
- `b_0 > 207` is special: `[216, 219]` marks a **silence** frame; other high
  values are **invalid** (frame-repeat upstream); `b_0 ≤ 7` is a degenerate
  **idle-tone** corner that a *run* of frames flags as a dead carrier.
- The **PRBS scrambler** (a 16-bit LCG keyed off `b_0`) whitens `u_1..u_6`; `u_0`
  stays clear because it carries the seed. Descramble is XOR — self-inverse.
- The spectral envelope is reconstructed table-first: **six PRBA gain blocks** set
  the coarse shape, **HOC DCT coefficients** add per-band detail, and two inverse
  DCT-IIs produce `Tl[1..L]`.

## Cheat sheet

| Frame element | Bits | Becomes |
|---|---|---|
| `b_0` (fundamental) | 8, positions `{0..5, 85, 86}` | ω₀, `L` (9..56), `K` |
| `b_1` (voicing) | `K` = ⌈(L+2)/3⌉ | `Vl[1..L]` (one bit per 3 harmonics) |
| `b_2` (gain) | 6 | `Gm[1]` via `b2Table` lookup |
| `b_3..b_7` (PRBA) | per `ba[L9]` | `Gm[2..6]` → inverse DCT → `Ri[1..6]` |
| HOC coefficients | per `hoba[L9]` | `Cik` → inverse DCT → `Tl[1..L]` |
| Descramble | — | undo `b_0`-keyed PRBS on `u_1..u_6` |

## In this post

- **The 88-bit frame** — what a P25 Phase 1 IMBE frame carries.
- **The pitch codeword** — `b_0`, and everything it derives.
- **The scrambler** — why the bits are whitened and how the LCG undoes it.
- **The spectral unpack** — PRBA gain blocks, DCT coefficients, and two inverse
  DCTs.

## The 88-bit IMBE frame

A P25 Phase 1 voice frame reaches this decoder as **88 information bits** — the
recovered payload *after* the channel FEC and de-interleave that Part 5 covers.
GopherTrunk carries them one bit per byte (`0`/`1`), MSB-first, packed into 11
bytes on the wire (`FrameBytes = 11`). The decoder's job is to interpret those 88
bits as an IMBE model-parameter set and hand a synthesizable `mbe.Params` down to
Part 3's pipeline.

The bits aren't laid out contiguously by parameter — IMBE scatters them for FEC
and whitening reasons — so unpacking is a sequence of table-driven gathers. The
richer `imbe.Params` struct captures the intermediate values before the projection
to the shared shape:

```go
// internal/voice/imbe/params.go (shape)
type Params struct {
    Header                     // W0, L, K, Silent, IdleTone
    Vl  [57]int                // voicing decisions, 1-indexed
    Gm  [7]float64             // Gm[1..6] PRBA gain block values
    Cik [7][11]float64         // per-band DCT coefficients
    Tl  [57]float64            // spectral log-amplitude residuals
}

func (p Params) MBE() mbe.Params {   // drop K, Gm, Cik; keep W0/Vl/Tl
    return mbe.Params{Header: p.Header.MBE(), Vl: p.Vl, Tl: p.Tl}
}
```

That `MBE()` projection is the seam from Part 2: `Gm` and `Cik` are IMBE-only
scaffolding, discarded once they've produced `Tl`.

## The pitch codeword sets everything

Decoding begins with the fundamental frequency, `b_0`. It lives at **scattered bit
positions** — six contiguous high bits plus two trailing bits:

```go
// internal/voice/imbe/params.go
func b0FromInfo(info []byte) uint {
    return uint(info[0])<<7 | uint(info[1])<<6 | uint(info[2])<<5 |
           uint(info[3])<<4 | uint(info[4])<<3 | uint(info[5])<<2 |
           uint(info[85])<<1 | uint(info[86])
}
```

From that one 8-bit value the whole frame geometry follows:

```go
// internal/voice/imbe/params.go — UnpackHeader (shape)
w0 := (4 * math.Pi) / (float64(b0) + 39.5)
// L = floor(0.9254 · floor(π/w0 + 0.25)); the inner truncation matters at L=9/56
L := int(0.9254 * float64(int(math.Pi/w0+0.25)))
K := 12
if L < 37 { K = (L + 2) / 3 }
```

`ω₀` is a direct function of `b_0`; `L` (the harmonic count, 9..56) is a function
of `ω₀`; and `K` (how many bits the voicing field uses) is a function of `L`. The
inner integer truncation in the `L` formula isn't cosmetic — preserving it matters
for the boundary cases at `L = 9` and `L = 56`, and the code mirrors mbelib's
`mbe_decodeImbe4400Parms` exactly.

`b_0` also carries the frame's special states, checked before anything else:

<figure class="lab-figure">
<svg viewBox="0 0 680 150" width="680" height="150" role="img" aria-label="A number line of the b_0 pitch codeword value: 0 to 7 is the idle-tone corner, 8 to 207 is normal speech, 208 to 215 and 220 and up are invalid, and 216 to 219 marks a silence frame">
  <line x1="30" y1="80" x2="650" y2="80" stroke="currentColor"/>
  <rect x="30" y="66" width="40" height="28" fill="var(--fg-muted)" opacity="0.25" stroke="var(--fg-muted)"/>
  <text x="50" y="112" text-anchor="middle" fill="var(--fg-muted)" font-size="9">0–7</text>
  <text x="50" y="126" text-anchor="middle" fill="var(--fg-muted)" font-size="9">idle</text>
  <rect x="70" y="66" width="380" height="28" fill="var(--accent)" opacity="0.18" stroke="var(--accent)"/>
  <text x="260" y="58" text-anchor="middle" fill="var(--accent)" font-size="11">8 – 207 : normal speech (ω₀, L, K derived)</text>
  <rect x="450" y="66" width="60" height="28" fill="none" stroke="var(--fg-muted)" stroke-dasharray="3 2"/>
  <text x="480" y="112" text-anchor="middle" fill="var(--fg-muted)" font-size="9">208–215</text>
  <text x="480" y="126" text-anchor="middle" fill="var(--fg-muted)" font-size="9">invalid</text>
  <rect x="510" y="66" width="40" height="28" fill="currentColor" opacity="0.2" stroke="currentColor"/>
  <text x="530" y="112" text-anchor="middle" fill="currentColor" font-size="9">216–219</text>
  <text x="530" y="126" text-anchor="middle" fill="currentColor" font-size="9">silence</text>
  <rect x="550" y="66" width="100" height="28" fill="none" stroke="var(--fg-muted)" stroke-dasharray="3 2"/>
  <text x="600" y="112" text-anchor="middle" fill="var(--fg-muted)" font-size="9">220+ invalid</text>
</svg>
<figcaption>The b_0 codeword doubles as the frame-type discriminator: a silence window, an invalid range that triggers frame-repeat upstream, and a degenerate low corner that a sustained run flags as an idle carrier.</figcaption>
</figure>

The `[216, 219]` window returns `Header{Silent: true}` and the synthesizer
short-circuits to silence. Anything else above 207 returns
`ErrInvalidFundamental`, and the decoder frame-repeats the last good frame rather
than voicing garbage. And `b_0 ≤ 7` sets an `IdleTone` flag — the `L = 9/10`,
~340–405 Hz corner a flat carrier resolves to; a *run* of these is a dead-key
signature the decoder mutes (Part 6 lives on that idea, and its cousin the
acquisition squelch).

## The scrambler: undo the whitening first

Before any of the *spectral* bits can be read, one thing has to be undone: P25
whitens the IMBE channel bits with a pseudo-random scrambler so the transmitted
spectrum has no DC bias or repetitive structure. `u_0` (which carries `b_0`) is
sent in the clear because it seeds the scrambler; `u_1..u_6` are XORed with a
114-bit PRBS. The generator is a plain 16-bit linear-congruential generator:

```go
// internal/voice/imbe/scrambler.go
func PRBS(seed uint16) [PRBSLength]byte {   // PRBSLength = 114
    const mul, inc, mod uint32 = 173, 13849, 65536
    state := uint32(seed)
    var bits [PRBSLength]byte
    for i := 0; i < PRBSLength; i++ {
        state = (mul*state + inc) % mod
        bits[i] = byte(state >> 15)         // high bit of the state
    }
    return bits
}
```

The seed is the 12 Golay data bits of `u_0` times 16 — so the whitening is keyed
to the frame's own pitch codeword. XOR is self-inverse, which is why `Scramble`
and `Descramble` are literally the same function under two names: applying the
PRBS twice is a no-op, so the decoder just runs it once over `u_1..u_6` to recover
the clean bits. (`u_7`, the least-sensitive bits, is never scrambled.) The LCG
constants — multiplier 173, increment 13849, output bit 15 — are transcribed from
mbelib's reference so a real P25 frame descrambles bit-for-bit.

## The spectral unpack: tables, then two DCTs

With clean bits and a known `L`, the remaining 79 bits are re-ordered through a
per-`L` table `bo[L9]` (where `L9 = L − 9`) into a `bb[vector][position]` layout,
and the spectral envelope is reconstructed in stages. First the voicing decisions:
IMBE fits `L` harmonic voicing bits into a `K = ⌈(L+2)/3⌉`-bit field by using **one
bit per group of three harmonics**, so `Vl[1..L]` is filled by walking the voicing
field and reusing each bit across three consecutive harmonics.

Then the gains. IMBE codes the spectral envelope as a **Predictive Residual Block
Average (PRBA)** — six gain values that capture the coarse shape — plus **Higher
Order Coefficients (HOC)** that add per-band detail:

```go
// internal/voice/imbe/params.go (shape)
p.Gm[1] = b2Table[b2]                         // b_2 (6 bits) → table lookup
for i := 2; i <= 6; i++ {                      // 5 PRBA blocks, Annex E eq. 68
    bits := int(ba[L9][i-2][0]); step := ba[L9][i-2][1]
    // read `bits` MSB-first, then dequantize: step·(bm − 2^(bits−1) + 0.5)
    p.Gm[i] = step * (float64(bm) - float64(int(1)<<uint(bits-1)) + 0.5)
}
// inverse 6-point DCT-II of Gm[1..6] → Ri[1..6] (the per-band DC levels)
// then HOC bits per hoba[L9] fill Cik[i][2..ji], quantized by
//   quantstep[bits-1] · standdev[k-2]
// finally an inverse DCT-II of Cik[i][1..ji] → Tl[l]
```

<figure class="lab-figure">
<svg viewBox="0 0 680 150" width="680" height="150" role="img" aria-label="The IMBE unpack pipeline: 88 information bits become b_0 which derives the header, then the remaining bits are re-ordered and decoded into voicing decisions, PRBA gain blocks, and HOC coefficients, which pass through two inverse DCTs to produce the spectral residuals Tl">
  <rect x="10" y="55" width="90" height="44" rx="6" fill="none" stroke="currentColor"/>
  <text x="55" y="74" text-anchor="middle" fill="currentColor" font-size="11">88 info bits</text>
  <text x="55" y="88" text-anchor="middle" fill="var(--fg-muted)" font-size="9">descrambled</text>
  <line x1="100" y1="77" x2="128" y2="77" stroke="currentColor"/><polygon points="128,73 138,77 128,81" fill="currentColor"/>
  <rect x="138" y="55" width="86" height="44" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="181" y="74" text-anchor="middle" fill="var(--accent)" font-size="11">b_0 → ω₀</text>
  <text x="181" y="88" text-anchor="middle" fill="var(--fg-muted)" font-size="9">L · K</text>
  <line x1="224" y1="77" x2="252" y2="77" stroke="currentColor"/><polygon points="252,73 262,77 252,81" fill="currentColor"/>
  <rect x="262" y="30" width="96" height="30" rx="6" fill="none" stroke="currentColor"/>
  <text x="310" y="49" text-anchor="middle" fill="currentColor" font-size="10">Vl voicing</text>
  <rect x="262" y="66" width="96" height="30" rx="6" fill="none" stroke="currentColor"/>
  <text x="310" y="85" text-anchor="middle" fill="currentColor" font-size="10">Gm PRBA</text>
  <rect x="262" y="102" width="96" height="30" rx="6" fill="none" stroke="currentColor"/>
  <text x="310" y="121" text-anchor="middle" fill="currentColor" font-size="10">Cik HOC</text>
  <line x1="358" y1="81" x2="392" y2="81" stroke="currentColor"/><polygon points="392,77 402,81 392,85" fill="currentColor"/>
  <rect x="402" y="60" width="110" height="40" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="457" y="78" text-anchor="middle" fill="var(--accent)" font-size="10">2 × inverse</text>
  <text x="457" y="92" text-anchor="middle" fill="var(--accent)" font-size="10">DCT-II</text>
  <line x1="512" y1="80" x2="546" y2="80" stroke="var(--accent)"/><polygon points="546,76 556,80 546,84" fill="var(--accent)"/>
  <rect x="556" y="60" width="110" height="40" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="611" y="78" text-anchor="middle" fill="var(--accent)" font-size="11">Tl[1..L]</text>
  <text x="611" y="92" text-anchor="middle" fill="var(--fg-muted)" font-size="9">→ mbe.Params</text>
</svg>
<figcaption>The IMBE unpack is a fan-out then a fan-in: b_0 sets the frame geometry, the remaining bits decode into voicing plus two families of gain/coefficient blocks, and two inverse DCTs collapse those into the spectral residuals the synthesizer consumes.</figcaption>
</figure>

The structure is: **`b2Table`, `ba`, `hoba`, `quantstep`, `standdev`** — a wall of
lookup tables (all of `tables.go`, ~47 KB of them) that encode the IMBE
quantizer, keyed by the harmonic count. The values are transcribed from the
TIA-102.BABA specification with mbelib as a structural reference. The dequantized
gains and coefficients pass through **two inverse DCT-IIs** — one over the six PRBA
gains to recover per-band DC levels `Ri`, one over each band's coefficients — to
produce `Tl[1..L]`, the pre-prediction spectral log-amplitude residuals Part 3's
`PredictLog2Ml` consumes.

### How that principle shaped the Go code

The whole file reads as a faithful transcription rather than a clever
reimplementation, and that's deliberate. Every table is annotated with its
TIA-102.BABA section and its mbelib source function; the integer truncations are
preserved exactly; the 1-based indexing matches the spec. The reason is stated in
the package doc and echoed across the codebase: a codec that's *subtly* wrong
decodes clean-looking frames to the wrong audio, and the only way to catch that is
to match a known-good reference bit-for-bit. GopherTrunk pins the full IMBE decode
against mbelib/DSD-faithful reference vectors precisely so a "cleanup" refactor
that breaks the quantization is caught by a failing test, not a field report.

The payoff for the DRY spine: once `UnpackParams` returns, the IMBE package is
done. `p.MBE()` hands `{W0, Vl, Tl}` to the shared synthesizer, which neither
knows nor cares that these numbers came from PRBA blocks and DCT tables rather than
AMBE+2's vector-quantization codebooks.

## Where this goes next

This post assumed clean 88-bit input. Getting there is a job in itself.
[Part 5]({{ '/blog/deep-dives/voice-coding-05-imbe-fec-deinterleave/' | relative_url }})
covers the layer beneath: the Golay(23,12) and Hamming(15,11) FEC that corrects
bit errors on the eight `u_n` vectors, the channel bit layout, and the
de-interleaver that must run *before* the descrambler because the `u_0` seed is
only valid in vector order. Then
[Part 6]({{ '/blog/deep-dives/voice-coding-06-acquisition-squelch/' | relative_url }})
tackles what happens when the FEC resolves *acquisition* noise to valid-but-wrong
frames at the very start of a call. For the protocol side of P25 Phase 1, the
[P25 Phase 1]({{ '/reference/p25-phase-1/' | relative_url }}) and
[IMBE]({{ '/reference/imbe/' | relative_url }}) Field Guide entries go wider.

## FAQ

**How many bits is an IMBE frame?**
88 information bits per 20 ms frame (4.4 kbps) — but that's *after* channel
coding. On air each frame is 144 channel bits including the Golay/Hamming FEC; Part
5 covers the 144 → 88 recovery.

**What is `b_0` and why does it matter so much?**
`b_0` is the fundamental-frequency codeword. It sets the pitch ω₀, from which the
harmonic count `L` and voicing-bit count `K` both derive, so every other field's
size depends on it. It also encodes the silence, invalid, and idle-tone frame
states.

**Why are the IMBE bits scrambled, and how is it undone?**
P25 XORs a pseudo-random sequence (a 16-bit LCG keyed off `b_0`) onto the channel
bits of `u_1..u_6` to whiten the transmitted spectrum. Since XOR is self-inverse,
the decoder regenerates the same sequence from the recovered seed and XORs once to
descramble; `u_0` and `u_7` are never scrambled.

**What are PRBA gain blocks?**
Predictive Residual Block Averages — six gain values that encode the coarse shape
of the spectral envelope. An inverse DCT turns them into per-band DC levels; HOC
(higher-order) DCT coefficients then add the per-band detail, and a second inverse
DCT produces the residuals `Tl`.

**Why transcribe mbelib's tables instead of computing the quantizer?**
Because the IMBE quantizer is defined *as* those tables in TIA-102.BABA, keyed by
harmonic count. Transcribing them faithfully (and pinning against reference
vectors) is the only way to guarantee a real P25 frame decodes to the correct
audio rather than plausible-sounding garbage.

## Series navigation

**Part 4 of 12** · ←
[Part 3: MBE Synthesis]({{ '/blog/deep-dives/voice-coding-03-mbe-synthesis/' | relative_url }})
· Next →
[Part 5: IMBE FEC & De-Interleave]({{ '/blog/deep-dives/voice-coding-05-imbe-fec-deinterleave/' | relative_url }})
