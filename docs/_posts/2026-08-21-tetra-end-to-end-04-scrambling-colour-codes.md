---
title: "TETRA End to End, Part 4: Scrambling & Colour Codes — Why Colour 0 Is Not a No-Op"
description: "Inside TETRA's scrambler — the 32-tap LFSR, the 30-bit extended colour code that seeds it, why colour 0 still produces a real scrambling sequence, and the two seed bugs that let a control channel lock perfectly while decoding nothing: a story that sets up the DMO 'encryption' misdiagnosis."
category: deep-dives
keywords: tetra scrambling, tetra colour code, extended colour code, tetra lfsr seed, newscramblertetra, bsch colour zero, tetra descramble bug, scrambler bit reversal, tetra 0xc0000000, gophertrunk tetra
tags: [tetra-end-to-end, tetra, scrambling, lfsr, colour-code, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "TETRA End to End"
series_part: 4
---

*Part 4 of **TETRA End to End**, a 14-part deep dive into how GopherTrunk turns
one real 25 kHz TETRA carrier — the MCC 250 / MNC 13 cell — into clear recorded
voice.
[Part 3]({{ '/blog/deep-dives/tetra-end-to-end-03-channel-coding-crc/' | relative_url }})
walked the channel-coding chain back to information bits and met the
self-consistent-synthetic trap in its CRC disguise. This part is about the
stage that looks least deserving of its own post — the scramble, a plain XOR —
and has hidden more failures per line of code than anything else in the TETRA
path. Two shipped bugs lived here, both invisible to round-trip tests, both
producing the same eerie symptom: a control channel that locks flawlessly and
decodes nothing. And one design fact — colour 0 is not the identity — is the
fuse that detonates three parts from now, in DMO.*

> **TL;DR:** TETRA scrambling XORs type-4 bits with a sequence from a
> **32-tap LFSR** (EN 300 392-2 §8.2.5, connection polynomial eq. 8.40,
> tap mask `0x82608EDB`), seeded by the 30-bit **extended colour code** —
> MCC(10) ‖ MNC(14) ‖ colour(6) (`tetra.ExtendedColourCode`). The seed
> construction is `reverseLow30(colour) | 0xC0000000`
> (`framing.NewScramblerTetra`): the low 30 bits are the colour code
> **bit-reversed**, the top two bits are the constant init `p(−31)=p(−30)=1`.
> So **colour 0 seeds `0xC0000000` — a real, non-trivial sequence**, which is
> why the BSCH (always colour 0) still descrambles explicitly. Two bugs hid
> here: an LFSR shifted the wrong direction (diverging from spec at the second
> output bit) and a seed that skipped the bit reversal — each let the BSCH
> decode (colour 0 masked them) while every colour-scrambled channel failed.
> Both are now pinned by a **real off-air BNCH block** that only descrambles
> CRC-clean under the correct scrambler
> (`internal/radio/tetra/scrambler_realair_test.go`).

**Key takeaways**

- **Scrambling is for the channel, not for secrecy.** It whitens each cell's
  bits with a cell-unique sequence so a receiver on a frequency reuse boundary
  doesn't accidentally decode the co-channel neighbour — colour codes are cell
  identity, not encryption (that's TEA, a different layer entirely).
- **Colour 0 is a sequence, not a skip.** The constant `0xC0000000` init means
  `NewScramblerTetra(0)` generates real output — any code path that treats
  colour 0 as "nothing to do" is wrong by construction.
- **The BSCH's fixed colour is what makes bugs invisible.** Both shipped
  scrambler bugs were no-ops *for the BSCH specifically*, so lock worked, the
  SYNC PDU parsed, and the failure surfaced one layer up as "no messages."
- **The regression guard is a captured burst, not a round-trip.** A real BNCH
  block from a live cell (extended colour 262144845) is committed as the
  on-air pin: it decodes CRC-clean only under the spec-correct scrambler.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| LFSR generator | 32-tap register, taps per eq. 8.40 (`0x82608EDB`) | `internal/radio/framing/scramble_tetra.go` (`ScramblerTetra.Next`) |
| Seed construction | `reverseLow30(colour) \| 0xC0000000` per eq. 8.42 | `scramble_tetra.go` (`NewScramblerTetra`, `reverseLow30`) |
| (De)scramble | XOR-symmetric, one helper for both directions | `scramble_tetra.go` (`ScrambleTetra`, `DescrambleTetra`) |
| Extended colour code | MCC(10) ‖ MNC(14) ‖ colour(6) → 30 bits | `internal/radio/tetra/sync_pdu.go` (`SyncPDU.ExtendedColourCode`) |
| Colour learning | BSCH (colour 0) → SYNC PDU → cell colour | `internal/radio/tetra/process.go` (`processSB` path) |
| On-air regression pin | real BNCH block, CRC-clean only when correct | `internal/radio/tetra/scrambler_realair_test.go` |

## In this post

- **What scrambling is for** — colour codes as cell identity, and the bootstrap ladder.
- **The LFSR and its seed** — eq. 8.40–8.42 as thirty lines of Go.
- **Why colour 0 is not a no-op** — the constant bits that guarantee a sequence.
- **Two bugs, one symptom** — the shift direction and the missing bit reversal.
- **The shortcut that survives in TMO** — `if colour != 0`, and the fuse it lights for DMO.

## What scrambling is for

TETRA cells reuse frequencies, and a receiver near a boundary can hear two
cells at once. Scrambling makes each cell's bits meaningless under any other
cell's descrambler: every channel's type-4 bits are XORed with a pseudo-random
sequence seeded by that cell's identity, so a co-channel neighbour's burst
fails your CRC instead of silently decoding as a wrong message. It is
whitening plus cell addressing — **not** encryption. (TETRA's actual
encryption, TEA, operates on the payload above the MAC; see the
[TEA reference]({{ '/reference/tetra-tea/' | relative_url }}). Confusing the
two layers is exactly the misdiagnosis Part 12 unwinds.)

The identity in question is the **extended colour code**: 30 bits packed as
MCC(10) ‖ MNC(14) ‖ colour(6). Our running cell broadcasts MCC 250, MNC 13 —
and a neighbouring cell of the same network pinned in the test suite uses base
colour 13, packing to extended colour 262144845. The bootstrap ladder from
Part 2 completes here: the BSCH is scrambled with colour 0 by spec (§8.2.5.2),
a cold receiver decodes it with `NewScramblerTetra(0)`, reads MCC/MNC/colour
from the SYNC PDU, calls `ExtendedColourCode()`, and from that moment every
BNCH, SCH and traffic channel on the cell is descramblable. No operator
configuration — the cell tells you its own key.

## The LFSR and its seed

The generator is thirty lines. The connection polynomial (eq. 8.40) becomes a
tap mask; the recurrence (eq. 8.41) becomes popcount-parity of the masked
state; the initialisation (eq. 8.42) becomes the seed layout:

```go
// internal/radio/framing/scramble_tetra.go (shape)
const scrambleTetraTapMask uint32 = 0x82608EDB // c(x) taps, eq. 8.40

func NewScramblerTetra(colourCode uint32) *ScramblerTetra {
    return &ScramblerTetra{
        state: reverseLow30(colourCode) | 0xC0000000, // e(1..30) reversed + p(−31)=p(−30)=1
    }
}

func (s *ScramblerTetra) Next() byte {
    v := s.state & scrambleTetraTapMask
    bit := byte(popcount32(v) & 1)
    s.state = (s.state << 1) | uint32(bit) // shift LEFT, insert at bit 0
    return bit
}
```

Every subtlety lives in the seed line. Eq. 8.42 demands state bit *i* hold
e(*i*+1) — but the packed colour-code value carries e(1) in its **most**
significant bit, so the low 30 bits must be bit-reversed on the way into the
register (`reverseLow30`). And the top two bits are hardwired to 1: the spec's
`p(−31) = p(−30) = 1`.

<figure class="lab-figure">
<svg viewBox="0 0 680 200" width="680" height="200" role="img" aria-label="The TETRA scrambler seed layout: a 32-bit register whose top two bits are the constant ones from the spec initialisation, and whose low thirty bits hold the extended colour code bit-reversed. Below, the colour-zero case shows the register holding 0xC0000000 — still a real seed producing a real scrambling sequence.">
  <rect x="30" y="40" width="80" height="40" fill="var(--accent)" opacity="0.3" stroke="var(--accent)"/>
  <rect x="110" y="40" width="540" height="40" fill="none" stroke="currentColor"/>
  <text x="70" y="64" text-anchor="middle" fill="var(--accent)" font-size="11">1 1</text>
  <text x="70" y="30" text-anchor="middle" fill="var(--accent)" font-size="9">p(−31), p(−30) — constant</text>
  <text x="380" y="64" text-anchor="middle" fill="currentColor" font-size="11">e(30) … e(2) e(1)   ← extended colour code, bit-reversed</text>
  <text x="380" y="30" text-anchor="middle" fill="var(--fg-muted)" font-size="9">30 bits: MCC(10) ‖ MNC(14) ‖ colour(6), e(1) ends up in state bit 0</text>
  <text x="30" y="98" fill="var(--fg-muted)" font-size="9">bit 31</text>
  <text x="620" y="98" fill="var(--fg-muted)" font-size="9">bit 0</text>
  <rect x="30" y="120" width="80" height="34" fill="var(--accent)" opacity="0.3" stroke="var(--accent)"/>
  <rect x="110" y="120" width="540" height="34" fill="none" stroke="var(--fg-muted)"/>
  <text x="70" y="141" text-anchor="middle" fill="var(--accent)" font-size="11">1 1</text>
  <text x="380" y="141" text-anchor="middle" fill="var(--fg-muted)" font-size="11">0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0</text>
  <text x="340" y="180" text-anchor="middle" fill="currentColor" font-size="10">colour 0 ⇒ state = 0xC0000000 — the two constant bits still drive the LFSR: a REAL sequence, not identity</text>
</svg>
<figcaption>The seed register: two hardwired 1s above thirty bit-reversed colour-code bits. At colour 0 the constants remain — the scrambler still scrambles.</figcaption>
</figure>

## Why colour 0 is not a no-op

Look at the colour-0 row of that figure. Zero colour bits, but the register is
`0xC0000000`, not zero — the two constant bits guarantee the LFSR never starts
empty, so `NewScramblerTetra(0)` emits a genuine pseudo-random sequence. This
is why the BSCH decoders *always* run the descramble: "scrambled with colour
0" means XORed with a real 120-bit sequence, not left alone.

The design intuition matters more than the trivia. If colour 0 were the
identity, an all-zeros register would also be a fixed point of the LFSR and a
degenerate cell configuration would transmit raw bits. The spec's constant
init closes that. But it also means any code path written as "colour 0 ⇒
nothing to descramble" is a landmine that *happens* not to explode wherever
colour 0 never legitimately occurs — hold that thought for two sections.

## Two bugs, one symptom

Both shipped scrambler bugs produced the identical clinical picture: training
sequences correlate, the BSCH decodes, the SYNC PDU parses, `cc.locked` goes
true — and not one BNCH SYSINFO or SCH message ever decodes. Lock without
comprehension.

**Bug one: the shift direction** (issue #925). The original `Next()` shifted
the register right and inserted the new bit at bit 31 — reversing the register
relative to the tap convention, so the output sequence diverged from eq. 8.41
at p(2). The second output bit was already wrong.

**Bug two: the missing bit reversal.** The seed loaded the packed colour value
raw, without `reverseLow30`. For the BSCH this is invisible — zero reversed is
zero — and for every round-trip test it is invisible too, because encode and
decode shared the reversed-wrong seed.

Notice what both bugs have in common with Part 3's CRC: **colour 0 is a fixed
point of the mistake**. The BSCH — the channel that produces the visible
"lock" — was immune to both, so the health indicator stayed green while the
payload channels rotted. And both were self-consistent under round-trip. The
fix came with the only kind of test that can hold it: a real SCH/HD block
captured off-air from a live BNCH — 108 dibits exactly as a real base station
scrambled them, extended colour 262144845 — committed in
`scrambler_realair_test.go`. It FEC-decodes CRC-clean under the corrected
scrambler and fails under either bug. One captured burst does what a thousand
synthetic round-trips cannot: it pins the implementation to the world.

## The shortcut that survives in TMO

With the scrambler correct, one optimisation crept into the traffic path and
deserves a hard look, because it is the fuse for Part 12:

```go
// internal/radio/tetra/traffic.go (shape) — emit
bits := TetraDibitsToBits(d)
if te.colourCode != 0 {
    bits = framing.DescrambleTetra(bits, te.colourCode)
}
te.onBurst(framing.PackBitsMSB(bits), te.softFrame(L), te.slotOf(L), te.usageOf(L))
```

"Descramble only when we have a colour" reads as sensible defensive coding —
zero here really means *the extractor hasn't been told the colour yet*, and
descrambling with a wrong guess would be worse than not descrambling. In TMO
this is safe for a structural reason: a real cell's extended colour code packs
MCC and MNC into the high 24 bits, and a live network's MCC/MNC are never both
zero — so extended colour 0 never occurs on a TMO carrier, and the `!= 0`
guard only ever gates the "colour unknown" startup window.

But the guard *encodes the wrong idea* — it conflates "colour unknown" with
"colour 0 needs no descramble," and the second half of that is false, as this
whole post established. TETRA DMO, where radios legitimately operate at colour
0 with no MNI, inherited this exact shortcut — and clear, unencrypted DMO
voice arrived at the Viterbi still scrambled, producing a chance-floor CRC
yield that got misread as encryption.
[Part 12]({{ '/blog/deep-dives/tetra-end-to-end-12-dmo-descramble-colour/' | relative_url }})
tells that story in full; the point to carry forward is that the bug was
armed *here*, in a TMO-safe shortcut, years before DMO pulled the trigger.

### How that principle shaped the Go code

- **One generator, both directions.** `DescrambleTetra` is an alias of
  `ScrambleTetra` because XOR is symmetric — there is no second implementation
  to drift.
- **The seed's invariants are in the doc comment.** `NewScramblerTetra`'s
  comment spells out which bugs are invisible to which tests (BSCH as a
  bit-reversal fixed point; round-trips sharing the seed) — the failure
  analysis lives next to the line that can cause it.
- **The on-air pin is data, not prose.** `realBNCHBlock` is a base64 constant
  in the test file; any future scrambler "cleanup" that breaks spec
  compliance fails CI against a burst a real base station transmitted.

## Where this goes next

The scrambler was the last stage between us and payload. 
[Part 5]({{ '/blog/deep-dives/tetra-end-to-end-05-tchs-traffic-channel/' | relative_url }})
finally decodes what the grants promised: TCH/S, the full-rate speech traffic
channel — two 137-bit speech frames per slot, the class-0/1/2 bit
sensitivity split, the AACH usage marker that routes concurrent same-carrier
calls, and the replay harness that correlated decoded slots against the
control channel's grant timeslots on real air.

## FAQ

**Is a colour code a security feature?**
No. It's cell addressing — six bits of colour plus the network's MCC/MNC,
public by definition (the cell broadcasts them in the clear on its BSCH).
Anyone who can decode the BSCH can descramble the cell. Encryption in TETRA is
TEA, a separate mechanism at a higher layer.

**Why does the BSCH use colour 0 instead of the cell's colour?**
Chicken and egg: the colour code is *learned from* the BSCH, so the BSCH must
be decodable before you know it. Fixing its scrambler seed at colour 0 (still
a real sequence — `0xC0000000`) gives every receiver a universal entry point.

**How would I recognise a scrambler bug in the field?**
The signature is precisely asymmetric: sync correlates and the BSCH/SYNC PDU
decode (they're colour 0), the decoder reports lock, and the message counters
for everything colour-scrambled — BNCH SYSINFO, SCH grants — sit at zero or at
a CRC chance floor. Lock without messages means look at the scrambler seed
path before anything else.

**Why commit a captured burst instead of a longer synthetic test?**
Because every synthetic test shares the implementation's assumptions — Part
3's lesson. The committed BNCH block was scrambled by a real base station, so
it checks GopherTrunk against the spec as deployed, not against itself. It
turned two "works in tests, dead on air" bugs into ordinary red tests.

**Does the `colourCode != 0` guard in the traffic extractor ever misfire in TMO?**
Not on a real network — extended colour 0 requires MCC = MNC = colour = 0,
which no live TMO cell broadcasts. The guard's real cost was conceptual: it
taught the codebase that colour 0 means "skip," and the DMO path (where colour
0 is routine) copied that lesson verbatim. Part 12 is the invoice.

## Series navigation

**Part 4 of 14** · ←
[Part 3: Channel Coding — RCPC, Viterbi & the CRC That Isn't an LFSR]({{ '/blog/deep-dives/tetra-end-to-end-03-channel-coding-crc/' | relative_url }})
· Next →
[Part 5: TCH/S — From Traffic Burst to Speech Frame]({{ '/blog/deep-dives/tetra-end-to-end-05-tchs-traffic-channel/' | relative_url }})
