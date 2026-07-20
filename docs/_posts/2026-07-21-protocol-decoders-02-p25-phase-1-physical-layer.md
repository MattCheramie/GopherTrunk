---
title: "Protocol Decoders, Part 2: P25 Phase 1 — NID, Sync & the FEC That Gates Lock"
description: A field-level tour of P25 Phase 1's physical layer in GopherTrunk — the 48-bit frame sync, the BCH-protected NID with its NAC and DUID, C4FM 4-level symbols, and the trellis plus Hamming FEC that decide whether a control channel locks.
category: deep-dives
keywords: p25 phase 1 physical layer, p25 nid nac duid, frame sync word, c4fm dibit, bch 63 16, trellis viterbi decoder, hamming 10 6, p25 fec lock, gophertrunk p25 decoder
tags: [protocol-decoders, p25, fec, dsp, framing, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "Protocol Decoders"
series_part: 2
---

*Part 2 of **Protocol Decoders**. Part 1 mapped the shape every decoder shares;
now we walk P25 Phase 1's physical layer stage by stage — sync, NID, symbols, and
the forward error correction that quietly decides whether you have a lock at all.
The design idea running under it: **FEC is not a polish step, it's the gate.***

> **TL;DR:** P25 Phase 1 is C4FM — four-level FM at 4800 baud, two bits per symbol.
> A frame is a 48-bit **frame sync word** followed by a 64-bit **NID** carrying the
> 12-bit NAC and 4-bit DUID under BCH(63,16,11). Data blocks like the TSBK are
> protected by a 1/2-rate **trellis** code and a 98-dibit block interleaver;
> LDU/ES words use a shortened **Hamming(10,6)**. Each FEC layer is a threshold: a
> codeword either corrects within its radius or it doesn't, and "doesn't" is
> exactly where lock is lost. GopherTrunk treats every FEC stage as a lock gate and
> instruments the ones that matter.

**Key takeaways**

- The **frame sync word** is 24 dibits; GopherTrunk correlates it with a tolerance
  and a rotation search, and the margin of that match is a health metric.
- The **NID** is BCH(63,16,11) with a trailing per-DUID flag bit — the BCH radius
  is the *first* gate a control channel must clear.
- **C4FM** maps `+3/+1/−1/−3` to dibits; the mapping is fixed by spec and drives
  the slicer thresholds.
- The **trellis + interleaver** turn on-air burst errors into scattered
  single-dibit errors the Viterbi decoder can absorb; its path metric is a decode
  confidence you can threshold.
- Restricting the sync rotation search to physical rotations fixed a real
  false-lock bug (issue #275) — a story about FEC lying convincingly.

## Cheat sheet

| Field | Size | FEC / coding | GopherTrunk file |
|---|---|---|---|
| Frame Sync Word | 48 bits (24 dibits) | correlation, tol=4 | `p25/phase1/sync.go` |
| NID (NAC + DUID) | 64 bits | BCH(63,16,11), t=11 | `p25/phase1/nid.go` |
| C4FM symbol | 1 dibit | `+3→01 +1→00 −1→10 −3→11` | `p25/phase1/sync.go` |
| TSBK channel block | 98 dibits | 1/2-rate trellis + interleave | `p25/phase1/trellis.go`, `interleaver.go` |
| LC / ES codeword | 10 bits | shortened Hamming(10,6,3) | `p25/phase1/hamming10_6.go` |

## In this post

- **The frame in one picture** — FSW, NID, payload, and where FEC sits.
- **Frame sync** — correlation, tolerance, and the rotation trap.
- **The NID** — NAC, DUID, and why the BCH radius is the lock gate.
- **C4FM symbols** — four levels, two bits, one fixed mapping.
- **Trellis + interleave** — burst errors made survivable, plus a real false-lock bug.

## The frame, laid out

A P25 Phase 1 data unit begins the same way every time: a fixed 48-bit sync
pattern the receiver can correlate against blind, then a 64-bit Network ID that
says what kind of frame follows and which system it belongs to, then the
protocol-specific payload. Everything after the sync word is under FEC.

<figure class="lab-figure">
<svg viewBox="0 0 680 132" width="680" height="132" role="img" aria-label="A P25 Phase 1 frame layout: a 48-bit frame sync word, a 64-bit NID containing a 12-bit NAC, a 4-bit DUID, and a 48-bit BCH parity field, followed by the FEC-protected payload">
  <rect x="8" y="34" width="150" height="46" rx="5" fill="none" stroke="var(--accent)"/>
  <text x="83" y="54" text-anchor="middle" fill="var(--accent)" font-size="12">Frame Sync</text>
  <text x="83" y="70" text-anchor="middle" fill="var(--fg-muted)" font-size="10">48 bits · 24 dibits</text>
  <rect x="162" y="34" width="270" height="46" rx="5" fill="none" stroke="currentColor"/>
  <text x="297" y="26" text-anchor="middle" fill="var(--fg-muted)" font-size="10">NID — 64 bits, BCH(63,16,11)</text>
  <line x1="242" y1="34" x2="242" y2="80" stroke="var(--fg-muted)" stroke-dasharray="3 3"/>
  <line x1="292" y1="34" x2="292" y2="80" stroke="var(--fg-muted)" stroke-dasharray="3 3"/>
  <text x="202" y="60" text-anchor="middle" fill="currentColor" font-size="11">NAC</text>
  <text x="202" y="73" text-anchor="middle" fill="var(--fg-muted)" font-size="9">12b</text>
  <text x="267" y="60" text-anchor="middle" fill="currentColor" font-size="11">DUID</text>
  <text x="267" y="73" text-anchor="middle" fill="var(--fg-muted)" font-size="9">4b</text>
  <text x="362" y="60" text-anchor="middle" fill="currentColor" font-size="11">BCH parity</text>
  <text x="362" y="73" text-anchor="middle" fill="var(--fg-muted)" font-size="9">47b + flag</text>
  <rect x="436" y="34" width="236" height="46" rx="5" fill="none" stroke="var(--accent)"/>
  <text x="554" y="54" text-anchor="middle" fill="var(--accent)" font-size="12">payload</text>
  <text x="554" y="70" text-anchor="middle" fill="var(--fg-muted)" font-size="10">TSBK / LDU / TDULC — its own FEC</text>
  <text x="340" y="108" text-anchor="middle" fill="var(--fg-muted)" font-size="10">correlate the sync → BCH-decode the NID → hand the payload to the right parser</text>
</svg>
<figcaption>The invariant prefix of every P25 Phase 1 frame. The DUID inside the NID selects which payload parser runs next.</figcaption>
</figure>

## Frame sync: correlation with a rotation trap

The sync word is a compile-time constant — 24 dibits, hex `0x5575F5FF77FF`:

```go
// internal/radio/p25/phase1/sync.go (shape)
var FrameSyncWord = [24]uint8{
    1, 1, 1, 1, 1, 3, 1, 1, 3, 3, 1, 1,
    3, 3, 3, 3, 1, 3, 1, 3, 3, 3, 3, 3,
}
```

`SyncDetector` slides a 24-dibit window over the incoming stream and, at each
position, counts mismatches against `FrameSyncWord`. A hit is any window with at
most `tolerance` mismatches (default 4 of 24). But it doesn't just compare the raw
window — it tries **cyclic rotations** of the dibit alphabet, `(dibit + k) mod 4`,
because a front-end polarity flip or I/Q swap shifts every symbol by a constant.
`ProcessWithMargin` even reports the correlation *margin* per hit (`tolerance+1 −
bestMismatch`), and the distribution of that margin over a capture is the
sync-health metric: a margin pressed against 1 means sync is barely holding.

Here's where it bites. On a C4FM FM-discriminator stream, only two rotations are
*physical*: 0 (identity) and 2 (discriminator polarity flip). Rotations 1 and 3
can't occur on that path — but if you let the correlator try them anyway, it can
find a misaligned window that, once handed to the BCH decoder downstream,
*miscorrects* into a parity-valid pseudo-NID at the wrong rotation. That is issue
#275: post-fix retests kept converging on `rot=3` on a genuine C4FM site. The fix
was to restrict the search:

```go
// internal/radio/p25/phase1/sync.go (shape)
var RotationsAll  = RotationSet{0, 1, 2, 3} // CQPSK: real four-fold ambiguity
var RotationsC4FM = RotationSet{0, 2}       // C4FM: only these are physical
```

The CQPSK path keeps all four (its differential decode has a genuine four-fold
phase ambiguity); the C4FM path is pinned to `{0, 2}`. This is the series' first
lesson in a recurring theme: **FEC can manufacture a confident wrong answer**, so
you constrain its inputs to the physically possible.

## The NID: the first lock gate

Immediately after the sync word come 64 bits of Network ID. The first 63 are a
BCH(63,16,11) codeword; the 64th is a fixed per-DUID flag. Sixteen information bits
come out: 12 for the NAC (the system's Network Access Code) and 4 for the DUID
(which data unit follows).

```go
// internal/radio/p25/phase1/nid.go (shape)
func ParseNID(bits []byte) (NID, int, error) {
    var cw uint64
    for i := 0; i < 63; i++ {
        if bits[i]&1 != 0 { cw |= uint64(1) << uint(62-i) }
    }
    data, errs := framing.BCHDecode63_16(cw)  // errs < 0 ⇒ beyond t=11
    if errs < 0 { return NID{}, -1, ErrNIDUncorrectable }
    duid := DUID(data & 0xF)
    if expectedNIDParity(duid) != bits[63]&1 { // the trailing flag bit
        return NID{}, errs, ErrNIDParity
    }
    return NID{NAC: uint16((data >> 4) & 0xFFF), DUID: duid}, errs, nil
}
```

Two things make this the *gate*. First, BCH(63,16,11) can correct up to 11 bit
errors — a generous radius, which is why a control channel can lock in conditions
where the payload CRCs still fail. Second, the trailing flag bit is a fixed value
per DUID (0 for HDU/TDU/TSDU/PDU/TDULC, 1 for LDU1/LDU2), *not* an overall parity —
the "obvious but wrong" assumption that masked #275 for a long time. Getting that
detail right is what stops the BCH layer from accepting a plausible-looking
miscorrection.

GopherTrunk also keeps `NIDFromDibitsWithErrors`, which returns a per-dibit
error-count pattern against the BCH-corrected codeword. Where the residual errors
*cluster* diagnoses the fault: errors at one end mean post-sync timing slip; errors
in the tail dibits mean a status-symbol phase fault; errors spread across all 32
dibits mean SNR-limited demod. The FEC layer doubles as an oscilloscope.

The DUID is the branch point for the whole rest of the frame:

| DUID | Frame | What follows |
|---|---|---|
| `0x5` | LDU1 | voice + embedded Link Control |
| `0xA` | LDU2 | voice + Encryption Sync |
| `0x7` | TSDU | control-channel TSBKs (Part 3) |
| `0xF` | TDULC | terminator + Link Control (Part 3) |
| `0x3` | TDU | plain terminator |

## C4FM: four levels, two bits

The demodulator upstream produces a slicer output in `{−3, −1, +1, +3}` — the four
C4FM deviation levels. The spec fixes the mapping to dibits, and GopherTrunk states
it as a two-line switch:

```go
// internal/radio/p25/phase1/sync.go (shape) — +3→01 +1→00 −1→10 −3→11
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

There's nothing to correct here — it's a fixed alphabet — but the *thresholds* the
slicer uses to decide which of the four levels a sample is closest to are
calibrated from the spec's peak deviation (1800 Hz for P25). Get the deviation
calibration wrong and every symbol drifts toward its neighbour, which shows up as a
smeared symbol histogram and rising error rates long before the frame outright
fails. That histogram is one of the fields the Signal Lab dashboard reads.

<figure class="lab-figure">
<svg viewBox="0 0 640 168" width="640" height="168" role="img" aria-label="C4FM four-level mapping: deviation levels plus three, plus one, minus one, minus three map to dibits 01, 00, 10, 11 respectively, with slicer decision thresholds between them">
  <line x1="40" y1="130" x2="600" y2="130" stroke="var(--fg-muted)"/>
  <line x1="60" y1="30" x2="60" y2="130" stroke="var(--fg-muted)"/>
  <g stroke="currentColor" stroke-width="2">
    <line x1="120" y1="48" x2="180" y2="48"/>
    <line x1="230" y1="76" x2="290" y2="76"/>
    <line x1="340" y1="104" x2="400" y2="104"/>
    <line x1="450" y1="122" x2="510" y2="122"/>
  </g>
  <g stroke="var(--accent)" stroke-dasharray="4 3">
    <line x1="60" y1="62" x2="600" y2="62"/>
    <line x1="60" y1="90" x2="600" y2="90"/>
    <line x1="60" y1="113" x2="600" y2="113"/>
  </g>
  <text x="150" y="40" text-anchor="middle" fill="currentColor" font-size="11">+3 → 01</text>
  <text x="260" y="68" text-anchor="middle" fill="currentColor" font-size="11">+1 → 00</text>
  <text x="370" y="96" text-anchor="middle" fill="currentColor" font-size="11">−1 → 10</text>
  <text x="480" y="114" text-anchor="middle" fill="currentColor" font-size="11">−3 → 11</text>
  <text x="595" y="58" text-anchor="end" fill="var(--accent)" font-size="9">decision thresholds</text>
  <text x="320" y="152" text-anchor="middle" fill="var(--fg-muted)" font-size="10">the slicer picks the nearest level; deviation calibration sets where the dashed thresholds sit</text>
</svg>
<figcaption>C4FM's four deviation levels each carry a dibit. The dashed lines are the slicer's decision thresholds, positioned from the spec peak deviation.</figcaption>
</figure>

## Trellis + interleave: burst errors, absorbed

The TSBK and other data-block channels carry 48 information dibits (12 bytes),
which a **1/2-rate trellis** convolutional code expands to 98 channel dibits. This
is not the textbook (7,5) NASA code — it's a table-driven code whose state *is* the
most-recent input dibit, with transition outputs pulled from a 16-entry
constellation table (TIA-102.BAAA-A Annex A). Decoding is hard-decision Viterbi:

```go
// internal/radio/p25/phase1/trellis.go (shape)
func DecodeTrellis(channel []uint8) ([]uint8, int) {
    pm := [4]int{0, inf, inf, inf}     // path metric per state
    // …49 stages of add-compare-select over 4 states…
    return out[:48], finalMetric        // metric 0 = clean, higher = corrected
}
```

That `finalMetric` is the sum of dibit-distance penalties along the surviving path.
Zero means the channel was clean; positive means real errors were corrected. It's a
per-block confidence number, and the control channel tracks decoded-vs-failed TSBK
counts from exactly this signal to judge whether a lock is healthy.

The trellis alone isn't enough, because on-air errors arrive in *bursts* — a fade
clobbers several adjacent symbols at once, which is the worst case for a
convolutional code. So the coded stream is passed through a 98-dibit **block
interleaver** before transmission, and de-interleaved before Viterbi on receive:

```go
// internal/radio/p25/phase1/interleaver.go (shape)
func DeinterleaveTSBK(channel []uint8) []uint8 {
    out := make([]uint8, 98)
    for i := 0; i < 98; i++ { out[i] = channel[tsbkDeinterleavePerm[i]] }
    return out
}
```

The permutation scatters a contiguous burst into isolated single-dibit errors
spread across the block, which is exactly what Viterbi handles best. The two
permutation tables are inverses, pinned by an "is invertible" build-time test — the
kind of round-trip check that keeps a hand-transcribed spec table honest.

For the LDU Link Control and Encryption Sync words there's a third, smaller code: a
shortened **Hamming(10,6,3)**, 6 data bits plus 4 parity, single-error-correcting,
applied per codeword across the 240-bit LC/ES field. GopherTrunk implements it
locally as a syndrome-to-position lookup because the framing package's Hamming
helpers are the (15,11) and (13,9) shortenings, not this one.

### FEC as the lock gate — the design principle

Line the FEC layers up and they form a staircase of thresholds, each strictly
harder than the last:

1. **Sync correlation** clears at ≤4 dibit mismatches of 24.
2. **NID BCH(63,16,11)** clears at ≤11 bit errors.
3. **Trellis Viterbi** clears when the path metric stays low.
4. **TSBK CRC** (Part 3) is the final, unforgiving check.

A marginal signal fails these from the bottom up: NID still corrects while TSBK
CRCs collapse. That asymmetry is the whole diagnostic value — "locked but TSBKs
failing" is a *specific* condition (the issue #402 zero-IF DC-spike signature),
distinguishable from "never locked" precisely because the stronger BCH survives
what the weaker CRC does not. GopherTrunk instruments each step, so a capture tells
you *which* gate it fell at, not just that it failed.

## Where this goes next

[Part 3]({{ '/blog/deep-dives/protocol-decoders-03-p25-phase-1-tsbk-link-control/' | relative_url }})
picks up right where the trellis leaves off: the 12 recovered TSBK bytes, their CRC
trailer, the opcode taxonomy, and the grant PDU that becomes the `Grant` the
trunking engine consumes. It also covers Link Control and the TDULC — the exact
carriage where our talker-alias mystery rides. If you want the demod stages *below*
the symbols, the [SDR Internals]({{ '/blog/deep-dives/sdr-internals-06-demodulation/' | relative_url }})
posts cover C4FM demodulation and symbol timing.

## FAQ

**What's the difference between the NAC and the DUID?**
The NAC (Network Access Code, 12 bits) identifies the system, so a receiver can
reject frames from a co-channel neighbour. The DUID (4 bits) identifies the *frame
type* that follows the NID — TSDU, LDU1, TDULC, and so on. Both are recovered
together from the BCH-protected NID.

**Why does a control channel lock but still fail TSBK CRCs?**
Because the NID's BCH(63,16,11) corrects up to 11 errors while the TSBK trailer CRC
corrects none. A degraded front end (e.g. an on-channel DC spike) can leave the NID
recoverable while the TSBK payload fails — GopherTrunk detects that exact signature
and nudges the operator toward `dc_avoid`.

**What is C4FM and how many bits per symbol?**
C4FM is four-level continuous-phase FM at 4800 baud. Each symbol is one of four
deviation levels carrying two bits (a dibit), so the raw symbol rate maps to
9600 bit/s before FEC.

**Why interleave the trellis-coded bits?**
On-air errors cluster in bursts, which convolutional codes handle poorly. The
98-dibit block interleaver scatters a burst into isolated single-dibit errors
across the block, which the Viterbi decoder corrects far more easily.

**What was the issue #275 false-lock bug?**
Letting the sync correlator try non-physical dibit rotations on a C4FM stream let
the downstream BCH decoder miscorrect a misaligned window into a parity-valid
pseudo-NID. Restricting the C4FM rotation search to `{0, 2}` closed it.

## Series navigation

**Part 2 of 12** · ← [Part 1: Anatomy of a Control-Channel Decoder]({{ '/blog/deep-dives/protocol-decoders-01-anatomy-of-a-cc-decoder/' | relative_url }}) · Next →
[Part 3: P25 Phase 1 TSBKs & Link Control]({{ '/blog/deep-dives/protocol-decoders-03-p25-phase-1-tsbk-link-control/' | relative_url }})
