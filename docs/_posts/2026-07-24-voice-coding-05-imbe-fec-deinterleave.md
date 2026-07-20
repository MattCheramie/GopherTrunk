---
title: "Voice Coding, Part 5: IMBE FEC & De-Interleave — Recovering the Bit Soup"
description: How GopherTrunk turns 144 noisy on-air IMBE channel bits into 88 clean information bits — the Golay and Hamming FEC on the eight u-vectors, the channel layout, and why de-interleave must run before descramble.
category: deep-dives
keywords: imbe fec, golay 23 12, hamming 15 11, imbe deinterleave, p25 channel coding, imbe u vectors, deinterleave before descramble, gophertrunk imbe fec
tags: [voice, imbe, fec, p25, error-correction, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "Voice Coding"
series_part: 5
---

*Part 5 of **Voice Coding**. Part 4 decoded 88 clean information bits into MBE
parameters — and quietly assumed those 88 bits were correct. This post is the
layer that makes that assumption safe: the forward error correction and
de-interleaving that turn 144 noisy on-air channel bits into 88 trustworthy ones.*

> **TL;DR:** Each on-air IMBE frame is **144 channel bits** protecting 88
> information bits, split across eight vectors `u_0..u_7`. Four vectors use
> **Golay(23,12,7)** (corrects up to 3 errors), three use **Hamming(15,11,3)**
> (corrects 1), and `u_7` carries 7 raw bits with no FEC. But before any of that
> runs, a **§7.5 de-interleaver** must un-scatter the bits, because the burst-error
> spreading permutation and the descrambler both need the bits in *vector* order.
> The receive order is **de-interleave → descramble → per-vector FEC**, and getting
> it wrong reports ~100% uncorrectable frames.

**Key takeaways**

- 144 → 88: four Golay(23,12) vectors (12 info bits each) + three Hamming(15,11)
  vectors (11 each) + a 7-bit no-FEC `u_7` = 88 information bits.
- Both FEC decoders use an **exhaustive nearest-codeword search** over a
  precomputed codeword table — optimal within the correction radius and dead
  simple to verify.
- The **de-interleaver runs first**. Its permutation scatters codeword bits across
  the burst so a localized fade damages several codewords a little instead of one
  fatally — but the descrambler's `u_0` seed is only valid once bits are back in
  vector order.
- IMBE's Golay/Hamming are a **separate** implementation from the framing
  package's — same generator family, opposite bit association, so they're
  different codes in practice (issue #489).

## Cheat sheet

| Vector | Channel bits | Info bits | FEC |
|---|---|---|---|
| `u_0` | 23 | 12 | Golay(23,12,7) — also seeds the scrambler |
| `u_1..u_3` | 23 each | 12 each | Golay(23,12,7) |
| `u_4..u_6` | 15 each | 11 each | Hamming(15,11,3) |
| `u_7` | 7 | 7 | none (least-sensitive bits) |
| **Total** | **144** | **88** | |
| Receive order | — | — | Deinterleave → Descramble → DecodeChannel |

## In this post

- **The channel layout** — how 144 bits split into eight vectors.
- **The FEC** — Golay and Hamming by nearest-codeword search.
- **The de-interleaver** — why the burst-spreading permutation exists and runs
  first.
- **The ordering trap** — the bug that made every real frame uncorrectable.

## The channel layout: eight vectors

Part 4's 88 information bits are the *output* of this stage. The *input* is 144
channel bits off the air, laid out as eight vectors with fixed bit offsets:

```go
// internal/voice/imbe/channel.go
const (
    ChannelBits = 144
    u0Offset, u0Bits, u0InfoBits = 0,   23, 12
    u1Offset, u1Bits, u1InfoBits = 23,  23, 12
    u2Offset, u2Bits, u2InfoBits = 46,  23, 12
    u3Offset, u3Bits, u3InfoBits = 69,  23, 12
    u4Offset, u4Bits, u4InfoBits = 92,  15, 11
    u5Offset, u5Bits, u5InfoBits = 107, 15, 11
    u6Offset, u6Bits, u6InfoBits = 122, 15, 11
    u7Offset, u7Bits, u7InfoBits = 137, 7,  7
)
```

Not every bit is equally important. IMBE spends its FEC budget where it matters:
the four Golay-protected vectors carry the pitch codeword (`u_0`) and the
most-significant spectral bits, so they get the strong code (corrects 3 errors in
23 bits). The Hamming vectors carry mid-significance bits (corrects 1 in 15). And
`u_7` — the seven least-sensitive bits — gets no protection at all, because
spending parity on bits a listener won't miss is a waste of the 4.4 kbps budget.

<figure class="lab-figure">
<svg viewBox="0 0 680 130" width="680" height="130" role="img" aria-label="The 144-bit IMBE channel frame laid out as eight vectors: four 23-bit Golay vectors u0 to u3, three 15-bit Hamming vectors u4 to u6, and a 7-bit unprotected u7, totalling 144 channel bits protecting 88 information bits">
  <text x="20" y="30" fill="var(--fg-muted)" font-size="11">144 channel bits →</text>
  <g font-size="9">
    <rect x="20" y="45" width="70" height="40" rx="3" fill="var(--accent)" opacity="0.18" stroke="var(--accent)"/>
    <text x="55" y="62" text-anchor="middle" fill="var(--accent)" font-size="10">u_0</text>
    <text x="55" y="76" text-anchor="middle" fill="var(--fg-muted)">23→12 Golay</text>
    <rect x="92" y="45" width="70" height="40" rx="3" fill="var(--accent)" opacity="0.18" stroke="var(--accent)"/>
    <text x="127" y="62" text-anchor="middle" fill="var(--accent)" font-size="10">u_1</text>
    <text x="127" y="76" text-anchor="middle" fill="var(--fg-muted)">23→12</text>
    <rect x="164" y="45" width="70" height="40" rx="3" fill="var(--accent)" opacity="0.18" stroke="var(--accent)"/>
    <text x="199" y="62" text-anchor="middle" fill="var(--accent)" font-size="10">u_2</text>
    <text x="199" y="76" text-anchor="middle" fill="var(--fg-muted)">23→12</text>
    <rect x="236" y="45" width="70" height="40" rx="3" fill="var(--accent)" opacity="0.18" stroke="var(--accent)"/>
    <text x="271" y="62" text-anchor="middle" fill="var(--accent)" font-size="10">u_3</text>
    <text x="271" y="76" text-anchor="middle" fill="var(--fg-muted)">23→12</text>
    <rect x="308" y="45" width="60" height="40" rx="3" fill="currentColor" opacity="0.12" stroke="currentColor"/>
    <text x="338" y="62" text-anchor="middle" fill="currentColor" font-size="10">u_4</text>
    <text x="338" y="76" text-anchor="middle" fill="var(--fg-muted)">15→11 Ham</text>
    <rect x="370" y="45" width="60" height="40" rx="3" fill="currentColor" opacity="0.12" stroke="currentColor"/>
    <text x="400" y="62" text-anchor="middle" fill="currentColor" font-size="10">u_5</text>
    <text x="400" y="76" text-anchor="middle" fill="var(--fg-muted)">15→11</text>
    <rect x="432" y="45" width="60" height="40" rx="3" fill="currentColor" opacity="0.12" stroke="currentColor"/>
    <text x="462" y="62" text-anchor="middle" fill="currentColor" font-size="10">u_6</text>
    <text x="462" y="76" text-anchor="middle" fill="var(--fg-muted)">15→11</text>
    <rect x="494" y="45" width="46" height="40" rx="3" fill="none" stroke="var(--fg-muted)" stroke-dasharray="3 2"/>
    <text x="517" y="62" text-anchor="middle" fill="var(--fg-muted)" font-size="10">u_7</text>
    <text x="517" y="76" text-anchor="middle" fill="var(--fg-muted)">7, raw</text>
  </g>
  <text x="560" y="68" fill="var(--accent)" font-size="12">= 88 info</text>
</svg>
<figcaption>The FEC budget is spent by sensitivity: strong Golay on the pitch and top spectral bits, weaker Hamming on mid bits, nothing on the seven least-sensitive bits. 144 channel bits protect 88 information bits.</figcaption>
</figure>

## The FEC: nearest-codeword search

Both decoders take the same approach — build the full set of valid codewords once
at `init()`, then decode by finding the closest one by Hamming distance:

```go
// internal/voice/imbe/p25fec.go — Golay(23,12) decode
var golay23Codewords [4096]uint32 // built at init from golay23Encode

func golay23Decode(cw uint32) (uint16, int) {
    cw &= 0x7FFFFF
    bestData := uint16(0); bestDist := 24
    for d := 0; d < 4096; d++ {
        dist := popcount32(golay23Codewords[d] ^ cw)
        if dist < bestDist {
            bestDist = dist; bestData = uint16(d)
            if dist == 0 { return bestData, 0 }
        }
    }
    if bestDist > 3 { return bestData, -1 } // beyond correction radius
    return bestData, bestDist               // corrected `bestDist` errors
}
```

For a code with minimum distance 7, the nearest codeword within radius 3 is the
unique correct one, so this brute-force search is provably optimal — identical to a
syndrome-table decode, and vastly easier to audit. Hamming(15,11) works the same
way over its 2048 codewords, correcting single-bit errors (including data-bit
errors, which mbelib's compact syndrome table actually misses). Both return the
**corrected-error count**, which propagates all the way up: it feeds the adaptive
smoothing that tames weak-channel frames, and it's the signal the frame-repeat and
mute logic key off.

`DecodeChannel` runs all eight vectors and sums the corrections; a vector that
exceeds its radius sets an `ErrUncorrectable` flag but the partially-recovered bits
are still returned, so upstream can log and frame-repeat rather than drop.

### Why a separate FEC implementation

The comment atop `p25fec.go` flags something that looks like duplication and
isn't: GopherTrunk already has Golay and Hamming in `internal/radio/framing`, but
those are *deliberately not reused* here. The framing package's codes are built for
DMR/P25 link-control framing, and although the generator polynomials are related,
the two associate generator rows with data bits in **opposite order** — which makes
them *different codes* in practice. Feeding a real IMBE codeword through framing's
Golay decodes it to the wrong data. This was verified against a real P25 voice
capture and the mbelib reference (issue #489). So the IMBE FEC is its own
transcription of mbelib's `ecc.c`, and the redundancy is the point: matching the
transmitter exactly beats a shared abstraction that's subtly wrong.

## The de-interleaver: spread the burst, then run first

On air, the 144 channel bits are **interleaved** — scattered by a fixed
permutation so that adjacent codeword bits land far apart in the transmitted
burst. The reason is the physics of a fading channel: a signal drop-out damages a
*contiguous run* of on-air bits. If codeword bits were contiguous, one fade would
blow through a single vector's whole correction budget and destroy it. Interleaving
spreads each vector's bits across the burst, so the same fade instead sprinkles a
*few* errors into *several* vectors — errors each vector's FEC can absorb.

<figure class="lab-figure">
<svg viewBox="0 0 680 180" width="680" height="180" role="img" aria-label="A burst error on interleaved on-air bits, when de-interleaved back to vector order, is spread into isolated single-bit errors across multiple codewords, each within its FEC correction radius">
  <text x="20" y="24" fill="var(--fg-muted)" font-size="11">on-air burst order (a fade hits contiguous bits)</text>
  <g>
    <rect x="20" y="34" width="620" height="24" rx="3" fill="none" stroke="currentColor"/>
    <rect x="250" y="34" width="90" height="24" fill="currentColor" opacity="0.35"/>
    <text x="295" y="51" text-anchor="middle" fill="currentColor" font-size="10">fade / burst</text>
  </g>
  <text x="330" y="88" text-anchor="middle" fill="var(--accent)" font-size="12">↓ Deinterleave (permute to vector order)</text>
  <text x="20" y="112" fill="var(--fg-muted)" font-size="11">vector order (errors now spread across codewords)</text>
  <g font-size="9">
    <rect x="20" y="122" width="150" height="26" rx="3" fill="none" stroke="var(--accent)"/>
    <rect x="60" y="122" width="8" height="26" fill="currentColor" opacity="0.5"/>
    <text x="95" y="164" text-anchor="middle" fill="var(--fg-muted)">u_1: 1 err ✓</text>
    <rect x="180" y="122" width="150" height="26" rx="3" fill="none" stroke="var(--accent)"/>
    <rect x="250" y="122" width="8" height="26" fill="currentColor" opacity="0.5"/>
    <text x="255" y="164" text-anchor="middle" fill="var(--fg-muted)">u_3: 1 err ✓</text>
    <rect x="340" y="122" width="130" height="26" rx="3" fill="none" stroke="currentColor"/>
    <rect x="400" y="122" width="8" height="26" fill="currentColor" opacity="0.5"/>
    <text x="405" y="164" text-anchor="middle" fill="var(--fg-muted)">u_5: 1 err ✓</text>
    <rect x="480" y="122" width="160" height="26" rx="3" fill="none" stroke="var(--accent)"/>
    <rect x="560" y="122" width="8" height="26" fill="currentColor" opacity="0.5"/>
    <text x="560" y="164" text-anchor="middle" fill="var(--fg-muted)">u_2: 1 err ✓</text>
  </g>
</svg>
<figcaption>Interleaving trades a fatal concentrated hit for survivable scattered ones. A burst that would exceed one codeword's correction radius becomes a single correctable error in each of several codewords once de-interleaved.</figcaption>
</figure>

The permutation is a verified bijection of `0..143`, and the package guards it at
load time — `init()` panics if the table isn't a permutation, because a
non-bijective table would silently corrupt every frame:

```go
// internal/voice/imbe/interleave.go (shape)
func Deinterleave(onAir []byte) ([]byte, error) {
    if len(onAir) != ChannelBits { return nil, ErrChannelLength }
    vec := make([]byte, ChannelBits)
    for v, t := range imbeDeinterleave {
        vec[v] = onAir[t] & 1   // vector-order bit v comes from on-air bit t
    }
    return vec, nil
}
```

## The ordering trap

Here's the detail that ties Parts 4 and 5 together and cost a real debugging
session. The three inverse steps — de-interleave, descramble, per-vector FEC — must
run in that exact order:

```go
// internal/voice/imbe/channel.go — DecodeChannelToFrame (shape)
deinterleaved, _ := Deinterleave(channel)                    // 1. §7.5 first
u0Data, _ := golay23Decode(vectorToBlock(deinterleaved[u0Offset:u0Offset+u0Bits]))
descrambleWithSeed(deinterleaved, u0Data<<4)                 // 2. §7.4 descramble
info, errs, err := DecodeChannel(deinterleaved)              // 3. per-vector FEC
```

Why de-interleave *first*? Because the descrambler's seed comes from `u_0`, and
`u_0` only *is* `u_0` once the bits are back in vector order — on air its bits are
scattered across the burst. Run the descrambler on interleaved bits and you seed it
from the wrong bits and whiten the wrong positions. And why correct `u_0`'s Golay
*before* deriving the seed (the extra `golay23Decode` above)? So a single channel
error in `u_0` doesn't corrupt the seed and desync the entire `u_1..u_6`
descramble — mbelib runs its `u_0` correction before demodulation for exactly this
reason.

**The problem we hit:** omit the de-interleave step, or run it in the wrong place,
and the per-vector FEC decodes interleaved garbage — reporting **~100%
uncorrectable LDUs on real signals** (issue #489). The fix, and the guarantee it
worked, is that the full channel decode is now pinned against mbelib/DSD-faithful
reference vectors: a real P25 voice subframe decodes to the expected information
frame **bit-for-bit**. That's the difference between "it compiles and produces
audio" and "it produces the *right* audio."

## Where this goes next

FEC and de-interleave deliver clean bits when the signal is good. But at the very
*start* of a call, the receiver hasn't locked symbol timing yet, and the FEC — good
as it is — resolves marginal dibits to bit-patterns that are *valid* IMBE frames
but *wrong* speech.
[Part 6]({{ '/blog/deep-dives/voice-coding-06-acquisition-squelch/' | relative_url }})
is the current-work post on that exact failure: the "startup scratch" a fresh P25
recording opened with, and the acquisition squelch that mutes it. For the wider P25
Phase 1 picture, the [P25 Phase 1]({{ '/reference/p25-phase-1/' | relative_url }})
Field Guide entry covers the LDU structure these frames arrive in.

## FAQ

**Why does an IMBE frame need 144 bits to carry 88?**
The extra 56 bits are forward error correction. Golay(23,12) and Hamming(15,11)
parity let the receiver correct bit errors from a noisy channel without
retransmission — essential for a one-way broadcast codec.

**Why two different FEC codes instead of one?**
IMBE spends parity by sensitivity. The pitch and top spectral bits get strong
Golay (corrects 3 errors); mid bits get lighter Hamming (corrects 1); the seven
least-sensitive bits get none, saving budget for where errors actually hurt.

**Why must de-interleave run before descramble?**
The descrambler is keyed off the `u_0` vector, which only exists in vector order —
on air, `u_0`'s bits are scattered by the interleaver. Descrambling interleaved
bits seeds from the wrong bits and corrupts the frame. Correct order:
de-interleave → descramble → FEC.

**Why not reuse GopherTrunk's existing Golay/Hamming code?**
The framing package's codes associate generator rows with data bits in the opposite
order, making them different codes in practice — a real IMBE codeword decodes to the
wrong data under them. The IMBE FEC is a separate transcription of mbelib's
reference (issue #489).

**What does the corrected-error count get used for?**
It's the channel-quality signal. It drives adaptive smoothing on weak frames, and
it's how the decoder decides whether to trust, repeat, or mute a frame — the
telemetry that makes bad-channel behaviour graceful.

## Series navigation

**Part 5 of 12** · ←
[Part 4: IMBE Decode]({{ '/blog/deep-dives/voice-coding-04-imbe-decode/' | relative_url }})
· Next →
[Part 6: The Call-Startup Acquisition Squelch]({{ '/blog/deep-dives/voice-coding-06-acquisition-squelch/' | relative_url }})
