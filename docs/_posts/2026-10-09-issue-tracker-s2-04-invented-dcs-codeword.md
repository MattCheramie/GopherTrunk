---
title: "From the Issue Tracker, Season 2, Part 4: The Invented DCS Codeword — Pinned by Parity Equations, Not the Encoder"
description: "How GopherTrunk's DCS squelch never matched a real radio — its 23-bit codeword layout and bit order were invented, and the test synthesiser transmitted the same invention — and how the rebuilt word (C0..C8, 0 0 1, P0..P10; 023 = 0x763813) was pinned against the published parity equations for all 512 codes, then had its N/I polarity read off the reporter's Kenwoods on air."
category: solution-postmortem
keywords: dcs squelch not detecting, dcs codeword 23 bit golay, dcs 023 0x763813, dcs parity equations, dcs normal inverted polarity, d025n d025i, dcs aliases 023 340 766, dcs 023i equals 047n, tone dcs_polarity, gophertrunk from the issue tracker season 2
tags: [from-the-issue-tracker-s2, dcs, golay, self-consistent-trap, scanner, postmortem]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "From the Issue Tracker, Season 2"
series_part: 4
---

*Part 4 of **From the Issue Tracker, Season 2**, postmortems of GopherTrunk
bugs told with receipts.
[Part 3]({{ '/blog/solution-postmortem/issue-tracker-s2-03-radians-per-sample/' | relative_url }})
fixed a CTCSS gate that was off by a factor of fifty. This part is its
digital sibling, DCS, which was not off by a factor — it was off by
construction. The codeword GopherTrunk hunted for had never been on the
air, and the test that "proved" the detector transmitted the same
fiction. The way out was a test written from the published parity
equations rather than from the encoder, and the last piece — which NRZ
sense means "normal" — came from the reporter's radios.*

> **TL;DR:** DCS cycles a 23-bit Golay(23,12) word at 134.4 baud as
> sub-audible NRZ. GopherTrunk's `dcsCodewordFromOctal` built that word with
> the octal digits in the high bits, a "100" sync in the low bits, and slid
> received bits in MSB-first — a layout that exists nowhere but in the
> package and its test synthesiser, so every unit test passed and no radio
> ever matched ([#1184](https://github.com/MattCheramie/GopherTrunk/issues/1184)).
> On air, bit 0 goes first: **C0..C8** (the code, LSB first), the fixed
> **0 0 1**, then **P0..P10**; code 023 is `0x763813`. The new builder is a
> systematic cyclic encode with generator `0xAE3`, and `dcs_reference_test.go`
> pins all 512 codes against `refDCSWord`, written from the published parity
> equations independently of the encoder, plus SDRangel's alias tables
> (023 ≡ 340 ≡ 766; 023N ≡ 047I). The detector now shares `toneFrontEnd`,
> slices at the recent high/low midpoint (a carrier offset is DC in the
> discriminator), integrates at four staggered clock phases and needs
> `dcsConfirmBits` = 16 consecutive matches. The reporter's 24 September run
> then pinned the convention the gate-open log was waiting for — **D025N →
> 1 bits as positive deviation, D025I → negative** — and `tone.dcs_polarity`
> (normal | inverted | both) now opens only on the configured sense.

**Key takeaways**

- **An invented layout with a matching synthesiser is a perfect test and a
  dead detector.** Nothing in the old DCS suite came from outside the
  package.
- **Pin against equations, not code.** `refDCSWord` is eleven parity lines
  copied from the published description and evaluated per code; it shares
  no function with `dcsCodewordFromOctal`.
- **Alias structure is a second, independent witness.** Only the correct
  construction makes 023, 340 and 766 rotations of one word and 023's
  complement equal 047's word.
- **Some facts only the air can supply.** Which NRZ sense is "N" is a
  convention no equation fixes; the detector logged both senses until a
  radio set to D025N answered.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| On-air word | C0..C8 LSB-first, 0 0 1, P0..P10; bit i is the i-th bit sent | `internal/scanner/conventional/dcs.go` (`dcsCodewordFromOctal`, `dcsGolayPoly` = 0xAE3) |
| Independent pin | published parity equations, all 512 codes, 023 → `0x763813` | `dcs_reference_test.go` (`refDCSWord`, `TestDCSCodewordMatchesPublishedParity`) |
| Alias pin | 023 ≡ 340 ≡ 766, 054 ≡ 405 ≡ 675; 023 ↔ 047, 754 ↔ 116 flipped | `TestDCSCodewordAliasesMatchReferenceTables` |
| Slicing | midpoint of the last two half-word blocks' high/low | `trackLevel`, `blkLen` |
| Bit clock | 4 staggered integrate-and-dump phases, no clock recovery | `dcsPhases`, `dcsPhase` |
| Match | 23-bit window vs 46 rotations at Hamming ≤ 2, 16 consecutive bits | `checkTargets`, `distanceThreshold`, `dcsConfirmBits` |
| Polarity | normal (default) / inverted / both; opposite sense WARNs once | `DCSPolarity`, `tone.dcs_polarity`, `warnDCSOppositePolarity` |

## In this post

- **"Does not detect any code, normal or inverted"** — the report and the green suite.
- **What was invented** — digits in the high bits, a sync of "100", MSB first.
- **What is actually on the air** — the transmit-order word and its cyclic encode.
- **Pinned twice, from outside** — parity equations and alias tables.
- **A detector for a 134.4 baud stream at 2.4 MS/s** — midpoint slicing, four phases, 16 bits.
- **N and I, read off the radios** — the polarity convention and the one alias nobody can fix.

## "Does not detect any code, normal or inverted"

The same Kenwood lab that produced Parts 1–3 reported that "DCS squelch
does not detect any code, normal or inverted". Alongside the CTCSS rate bug
that was easy to believe as one more symptom of the same cause — the DCS
detector also ran the discriminator on the raw 2.4 MHz band — and the
`toneFrontEnd` fix of
[Part 3]({{ '/blog/solution-postmortem/issue-tracker-s2-03-radians-per-sample/' | relative_url }})
did apply to it. But DCS slices by sign, which is rate-invariant. The rate
fix alone would not have made it decode, and it did not.

The suite said otherwise. `dcs_test.go` synthesised a DCS stream, fed it
through the detector, and asserted a match; the detector matched. Every
test in the file was green. The question Season 1's
[Part 20]({{ '/blog/solution-postmortem/from-the-issue-tracker-20-self-consistent-trap/' | relative_url }})
taught the tracker to ask — *what does this test share with the code under
test?* — had an uncomfortable answer: the codeword. The synthesiser built
its stream by calling the same `dcsCodewordFromOctal` the detector hunted
with. Whatever that function returned, the test would transmit it and the
detector would find it.

## What was invented

Here is what it returned:

```go
// internal/scanner/conventional/dcs.go — as it shipped before #1184
// Layout of the 12 info bits passed to the Golay encoder:
// [octal-digit-1(3) | octal-digit-2(3) | octal-digit-3(3) | sync(3 = "100")]
// with octal-digit-1 in bits 11..9 and sync "100" in bits 2..0.
for i, r := range code {
    …
    info |= uint16(d) << (9 - (i+1)*3)
}
info = (info << 3) | 0b100
return framing.GolayEncode23_12(info), nil
```

Three decisions, each plausible in isolation, none from the air: the octal
digits most-significant first in the high bits of the information field; a
three-bit "sync" of `100` in the low bits; and the framing package's
generic Golay encoder emitting `[data | parity]`, with the detector then
sliding received bits in MSB-first. The real word has the code's bits
least-significant first at the *start*, the fixed pattern is `0 0 1`, the
parity follows, and the first bit on the air is bit 0. Every one of the
three choices was the mirror of the truth, and because the synthesiser
mirrored them identically, nothing in the repository could tell.

Two more defects rode along, both the kind a synthetic stream never
exercises. The detector sliced the discriminator on its raw **sign**, and a
carrier offset is a DC term in a discriminator — so a few hundred hertz of
tuning error alone pinned every bit to one value. And, as in Part 3, the
discriminator saw the whole SDR band.

## What is actually on the air

DCS transmits a 23-bit word, bit by bit, continuously: the 9-bit code
(three octal digits, least significant bit first), the fixed `0 0 1`, then
eleven parity bits, after which it starts again. A receiver can lock onto
any of the 23 cyclic rotations, which is why a DCS detector needs no frame
sync — and also why a code's rotations are other codes. The parity is the
remainder of a systematic Golay(23,12) encode with generator
x¹¹+x⁹+x⁷+x⁶+x⁵+x+1 (`dcsGolayPoly` = `0xAE3`), taken over the on-air bit
order:

```go
// internal/scanner/conventional/dcs.go
data := uint32(c) | 1<<11 // code + the fixed 0 0 1
// Systematic cyclic encoding over the transmit order: with bit i the
// coefficient of x^(22-i), the data occupies x^22..x^11 and the
// parity x^10..x^0 is (data · x^11) mod g(x).
var reg uint32
for i := 0; i < 12; i++ {
    fb := (data>>uint(i))&1 ^ (reg>>10)&1
    reg = (reg << 1) & 0x7FF
    if fb != 0 {
        reg ^= dcsGolayPoly & 0x7FF
    }
}
w := data
for i := 0; i < 11; i++ {
    w |= ((reg >> uint(10-i)) & 1) << uint(12+i)
}
```

Bit i of the result is the i-th bit sent. For code 023 the word is
`0x763813`.

<figure class="lab-figure">
<svg viewBox="0 0 680 200" width="680" height="200" role="img" aria-label="Two 23-bit layouts drawn as rows of cells with bit 0 on the left, the first bit sent. The top row, labelled invented, holds octal digit one in the highest bits, then digits two and three, a sync of one zero zero, and eleven parity bits in the low end, with the arrow of transmission pointing from the high bit. The bottom row, labelled on air, holds C0 to C8 in bits 0 to 8, the fixed zero zero one in bits 9 to 11 and P0 to P10 in bits 12 to 22, transmitted from bit 0, with code 023 annotated as 0x763813.">
  <text x="340" y="14" text-anchor="middle" fill="currentColor" font-size="10" font-weight="bold">the 23-bit DCS word, bit 0 at the left</text>
  <text x="60" y="58" text-anchor="end" fill="var(--fg-muted)" font-size="9">invented</text>
  <rect x="70" y="42" width="286" height="22" fill="none" stroke="var(--fg-muted)" stroke-dasharray="3 3"/>
  <text x="213" y="57" text-anchor="middle" fill="var(--fg-muted)" font-size="8">11 parity bits (low end)</text>
  <rect x="356" y="42" width="78" height="22" fill="none" stroke="var(--fg-muted)" stroke-dasharray="3 3"/>
  <text x="395" y="57" text-anchor="middle" fill="var(--fg-muted)" font-size="8">sync "100"</text>
  <rect x="434" y="42" width="78" height="22" fill="none" stroke="var(--fg-muted)" stroke-dasharray="3 3"/>
  <text x="473" y="57" text-anchor="middle" fill="var(--fg-muted)" font-size="8">digit 3</text>
  <rect x="512" y="42" width="78" height="22" fill="none" stroke="var(--fg-muted)" stroke-dasharray="3 3"/>
  <text x="551" y="57" text-anchor="middle" fill="var(--fg-muted)" font-size="8">digit 2</text>
  <rect x="590" y="42" width="78" height="22" fill="none" stroke="var(--fg-muted)" stroke-dasharray="3 3"/>
  <text x="629" y="57" text-anchor="middle" fill="var(--fg-muted)" font-size="8">digit 1</text>
  <text x="668" y="80" text-anchor="end" fill="var(--fg-muted)" font-size="8">← slid in MSB first; only the test synthesiser ever sent it</text>
  <text x="60" y="128" text-anchor="end" fill="currentColor" font-size="9">on air</text>
  <rect x="70" y="112" width="234" height="22" fill="none" stroke="currentColor" stroke-width="1.5"/>
  <text x="187" y="127" text-anchor="middle" fill="currentColor" font-size="8">C0 … C8 (code, LSB first)</text>
  <rect x="304" y="112" width="78" height="22" fill="none" stroke="currentColor" stroke-width="1.5"/>
  <text x="343" y="127" text-anchor="middle" fill="currentColor" font-size="8">0 0 1</text>
  <rect x="382" y="112" width="286" height="22" fill="none" stroke="var(--accent)" stroke-width="1.5"/>
  <text x="525" y="127" text-anchor="middle" fill="var(--accent)" font-size="8">P0 … P10 = (data · x¹¹) mod g(x), g = 0xAE3</text>
  <text x="70" y="150" fill="currentColor" font-size="8">bit 0 sent first →</text>
  <text x="668" y="150" text-anchor="end" fill="currentColor" font-size="8">023 = 0x763813 · cycles with no gap · any of 23 rotations locks</text>
  <text x="340" y="184" text-anchor="middle" fill="var(--fg-muted)" font-size="8">pinned for all 512 codes by refDCSWord (published parity equations) — a function that shares nothing with the encoder</text>
</svg>
<figcaption>Every choice in the invented layout was the mirror of the real one: digit order, position of the fixed bits, which end the parity lives at, and which bit goes first. A synthesiser that shared the function could not see any of it.</figcaption>
</figure>

## Pinned twice, from outside

The fix's regression is `dcs_reference_test.go`, and the point of it is the
provenance of `refDCSWord`. It is eleven parity equations from the
published DCS description (the same equations SDRangel's `NFMModDCS::setDCS`
transmits), evaluated per code as parity-of-masked-bits — P0 = C0+C1+C2+C3+C4+C7,
P1 = NOT(C1+C2+C3+C4+C5+C8), and so on through P10 — assembled into a word
with C0..C8 in bits 0..8 and `1<<11` for the fixed pattern. It calls nothing
in `dcs.go`. `TestDCSCodewordMatchesPublishedParity` then compares
`dcsCodewordFromOctal` against it for **every one of the 512 three-digit
octal codes**, and separately against the literal `0x763813` for 023 —
literal vectors of the kind
[From Spec to Shipping Part 3]({{ '/blog/deep-dives/from-spec-to-shipping-03-literal-vectors/' | relative_url }})
made the rule.

The second witness is structural. Only the correct construction has the
alias properties the published tables record: a code's word read from a
different start bit is another code, and the complement of a code's word
is another code. `TestDCSCodewordAliasesMatchReferenceTables` checks, from
SDRangel's `dcscodes.cpp`, that 023 ≡ 340 ≡ 766 and 054 ≡ 405 ≡ 675 are
rotations of one word, that 025 is its own rotation, and that 023 flipped
reads as 047, 754 as 116, 627 as 031. An invented layout can be
self-consistent; it cannot reproduce another implementation's alias table
by accident.

## A detector for a 134.4 baud stream at 2.4 MS/s

With the word right, the detector was rebuilt around how the stream
actually arrives. `toneFrontEnd` from Part 3 decimates to ~48 kHz,
channel-filters ±8 kHz and discriminates; a single-pole low-pass at 250 Hz
— above the 134.4 baud fundamental, below the voice band — rolls off the
audio. Then:

- **Slicing level.** `trackLevel` keeps the high and low of the last two
  half-word blocks (`blkLen` = 23/2 bits of samples) and slices at their
  midpoint. A DCS word always holds both bit values, so the extremes
  bracket the NRZ levels whatever DC the carrier offset adds — the fix for
  the sign-slicer that a tuning error alone could pin.
- **Bit clock.** There is no clock recovery. `dcsPhases` = 4 integrate-and-
  dump accumulators run at staggered phases; one of them is always within
  1/8 bit of the transmitter's, which is plenty at 134.4 baud.
- **Match.** Each phase keeps a 23-bit window in on-air order — the oldest
  bit ends up in bit 0, so a window aligned to a word boundary reads
  exactly the codeword — and compares it against the 46 precomputed
  targets (23 rotations × 2 polarities) at Hamming distance ≤
  `distanceThreshold` (2): 46 XOR-and-popcount operations per bit.
- **Confirmation.** A real stream matches at every bit shift; noise that
  matches one window keeps matching with probability ~1/2 per bit. The
  gate opens after `dcsConfirmBits` = 16 consecutive matches (~120 ms),
  cutting the false-open rate by ~2¹⁶, and closes once no phase has
  matched for a word (`sinceMatch > wordLen`). `dcsMinDwell` (600 ms) keeps
  the scanner on a DCS channel long enough for a first report.

`TestDCSDetectsOnAirWordAtSDRRate` drives it with `synthDCSOnAir`, which
modulates `refDCSWord` — the reference word, not the encoder's — bit 0
first at 134.4 baud with 350 Hz of deviation, a 600 Hz carrier offset, 25 dB
SNR and a random start bit and sub-bit phase, at 2.4 MS/s, for codes 023,
754 and 411 in both polarities; detection must be continuous after
`dcsMinDwell`. `TestDCSRejectsOtherCodeAndBareCarrierAtSDRRate` sends 754 to
an 023 gate and a bare carrier with noise, and requires neither ever opens.

## N and I, read off the radios

One fact no equation supplies: which NRZ sense a radio means by "normal".
The first fixed detector accepted both polarities and logged which one
matched, and `TestDCSReportsMatchedPolarity` pinned that `Inverted()`
reports the sense correctly. The log line was an instrument waiting for a
reading, and on 24 September the reporter's Kenwood NX-300/NX-5000 with a
service monitor supplied it: a radio set to **D025N** reads
`nrz_inverted=false` — the code's 1 bits arrive as positive frequency
deviation — and **D025I** reads `true`.

The gate therefore accepts only `tone.dcs_polarity` (`normal`, the default,
matching a plain "D023" or "D023N"; `inverted`; or `both`). `checkTargets`
checks the accepted rotations first, so a word near both senses never reads
as a mismatch; the other sense is still tracked, but only to flag a likely
misconfiguration — `oppStreak` reaching `dcsConfirmBits` sets
`oppositeHeard`, and `warnDCSOppositePolarity` logs once per channel:

```text
WRN conv: DCS code heard with the opposite polarity; gate kept shut — set tone.dcs_polarity to match the radio (N = normal, I = inverted) or both dcs_code=025 dcs_polarity=normal heard=inverted
```

`TestDCSGateAcceptsOnlyConfiguredPolarity` fails against the old
both-senses detector: normal must open on N and stay shut on I (raising the
hint), inverted the reverse, both on either. One alias is inherent and is
left alone: 023's inverted word is a rotation of 047's normal word, so a
normal 023 gate still opens on 047I, and `both` on 023 opens for 047N too.
No receiver can tell those apart on air, and `TestDCSDefaultGateIgnoresAliasedCode`
only requires that the *default* gate ignore 047N — the case the N/I
convention does settle — and that a different code never raises the
mismatch hint. All three DCS changes are on-air verified from that run.

## Where this goes next

Four parts on one reporter's lab; the next two go back to TETRA direct
mode, where a different kind of fiction had been accumulating: four
"colour codes" recovered from four captures — 3, 39, 36, 31 — none of
which was a colour code at all.
[Part 5]({{ '/blog/solution-postmortem/issue-tracker-s2-05-seed-is-the-source-address/' | relative_url }})
replaces the 64-way brute force with an exact GF(2) solve of the 30-bit
scramble seed from a single burst, and finds that the seed changes on every
PTT because it is the transmitting radio's source address.

## FAQ

**Why did GopherTrunk's DCS squelch never match a real radio?**
`dcsCodewordFromOctal` built the 23-bit word with the octal digits in the
high bits, a "100" sync in the low bits and parity after, and the detector
slid bits in MSB-first — a layout that exists only in the package. The
test synthesiser used the same function, so every test passed. The real
word is C0..C8 (LSB first), 0 0 1, P0..P10, bit 0 sent first.

**What is the DCS codeword for code 023?**
`0x763813`, with bit i the i-th bit on the air: bits 0..8 hold the code
least-significant bit first, bits 9..11 the fixed 0 0 1, bits 12..22 the
Golay parity from generator `0xAE3`. `TestDCSCodewordMatchesPublishedParity`
pins it, and all 511 other codes, against the published parity equations.

**What does tone.dcs_polarity mean?**
Which NRZ sense opens the gate. On the reporter's Kenwoods a radio set to
D025N sends the code's 1 bits as positive deviation (`normal`, the default)
and D025I as negative (`inverted`); `both` accepts either. A code heard in
the other sense keeps the gate shut and logs one `conv: DCS code heard with
the opposite polarity` WARN per channel.

**Why does my 023 gate open on a radio set to 047?**
Because 023 inverted and 047 normal are the same bit pattern, and so are
023N and 047I — DCS aliases are inherent, which the published tables
record (023 ≡ 340 ≡ 766 by rotation, 023 ↔ 047 by complement). With the
default `normal` polarity a 023 gate ignores 047N; it cannot ignore 047I,
and no receiver can.

**Is the DCS detector verified on air?**
Yes. The reporter's 24 September run on Kenwood NX-300/NX-5000 radios with a
service monitor decoded DCS and pinned the N/I convention
(`D025N → nrz_inverted=false`, `D025I → true`). The committed regressions
run the reference word through the production detector at 2.4 MS/s
(`TestDCSDetectsOnAirWordAtSDRRate`, `TestDCSGateAcceptsOnlyConfiguredPolarity`).

## Series navigation

**Part 4 of 14** · ←
[Part 3: Radians per Sample — A CTCSS Gate Calibrated at 48 kHz and Fed 2.4 MS/s]({{ '/blog/solution-postmortem/issue-tracker-s2-03-radians-per-sample/' | relative_url }})
· Next →
[Part 5: The Seed Is the Source Address — Solving TETRA DMO's Scramble Seed in GF(2)]({{ '/blog/solution-postmortem/issue-tracker-s2-05-seed-is-the-source-address/' | relative_url }})
