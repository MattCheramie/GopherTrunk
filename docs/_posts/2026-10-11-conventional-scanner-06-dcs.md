---
title: "The Conventional Scanner, Part 6: DCS — The On-Air Bit Order, Polarity, and the Aliases You Cannot Fix"
description: "How GopherTrunk's DCS squelch gate reads a 134.4 baud Golay(23,12) word off a conventional FM channel — the on-air bit order C0..C8, 0 0 1, P0..P10 that the first detector got backwards, the N/I polarity pinned on a Kenwood service monitor, the tone.dcs_polarity key, the once-per-channel mismatch WARN, and why 023N and 047I can never be told apart."
category: tutorials
keywords: dcs squelch sdr, digital coded squelch decoder, dpl code detection, dcs 023 codeword 0x763813, dcs polarity normal inverted, dcs_polarity config, golay 23 12 dcs, dcs alias 023 047, rtl-sdr dcs gate, gophertrunk conventional scanner
tags: [conventional-scanner, dcs, squelch, analog-fm, tone-gating, tutorial]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "The Conventional Scanner"
series_part: 6
---

*Part 6 of **The Conventional Scanner**, a 14-part operator's tutorial on
GopherTrunk's conventional (non-trunked) scanner, `scanner.conventional`, from
the dwell loop to a verified kitchen-sink config.
[Part 5]({{ '/blog/tutorials/conventional-scanner-05-ctcss-done-right/' | relative_url }})
rebuilt the CTCSS gate around exact Goertzel bins. This part reads the other
sub-audible gate: DCS, the 134.4 baud digital word a radio loops under its
voice — invented on both sides of the test suite for months, pinned to the
published parity equations, verified on a service monitor, and still unable
to tell two of its own codes apart, by construction.*

> **TL;DR:** DCS (Digital-Coded Squelch, Motorola's DPL) is a continuously
> cycled 23-bit Golay(23,12) word at 134.4 baud, NRZ-coded as sub-audible
> FM deviation. On air the bits go out **bit 0 first: C0..C8 (the octal code,
> LSB first), the fixed 0 0 1, then P0..P10** — code 023 is `0x763813`.
> `internal/scanner/conventional/dcs.go` runs the shared `toneFrontEnd`, a
> 250 Hz one-pole low-pass, a slicing level tracked as the midpoint of the
> last half-word blocks (the carrier offset is DC), four staggered
> integrate-and-dump bit clocks, and a 23-bit window checked against 46
> targets (23 rotations × 2 polarities) at Hamming distance ≤ 2;
> `dcsConfirmBits` = 16 consecutive matches (~120 ms) open the gate. The
> word is pinned against the published parity equations for all 512 codes
> (`TestDCSCodewordMatchesPublishedParity`) and SDRangel's alias tables
> (023 ≡ 340 ≡ 766, 023N ≡ 047I). `tone.dcs_polarity` (normal | inverted |
> both) picks the NRZ sense, pinned on air: **D025N → `nrz_inverted=false`,
> D025I → true**. The other sense WARNs once per channel and keeps the gate
> shut.

**Key takeaways**

- **The on-air bit order is the whole protocol.** The old codeword put the
  code in the high bits and the sync as `100` in the low bits, slid MSB-first,
  and its own synthesizer transmitted the same invention — green tests, zero
  radios ever matched.
- **Pin a codeword against something you did not write.** `refDCSWord` in
  `dcs_reference_test.go` is the published parity equations typed in
  independently of `dcsCodewordFromOctal`; all 512 codes must agree, and 023
  must equal the literal `0x763813`.
- **Polarity is a convention, so it was measured, not assumed.** The
  reporter's Kenwood NX-300/NX-5000 and a service monitor settled it: a
  radio set to N sends the code's 1 bits as positive deviation. The gate
  accepts one sense and flags the other.
- **Some aliases are physics.** A code's inverted word is a rotation of a
  different code's normal word, so a 023 gate opens on 047I whatever the
  software does. Do not file that as a bug.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| Codeword | 23-bit word, bit i = i-th bit on air; 023 → `0x763813` | `dcsCodewordFromOctal`, `dcsGolayPoly` = `0xAE3` (`dcs.go`) |
| Reference pin | 512 codes vs published parity equations; alias tables | `dcs_reference_test.go` (`refDCSWord`, `TestDCSCodewordAliasesMatchReferenceTables`) |
| Bit recovery | 250 Hz IIR, midpoint slicing level, `dcsPhases` = 4 clocks | `DCSDetector.Process`, `trackLevel` |
| Match | 46 targets at Hamming ≤ 2, `dcsConfirmBits` = 16 to open; `dcsMinDwell` = 600 ms | `checkTargets`, `dcsRotations`, `minToneDwell` |
| Polarity | `tone.dcs_polarity`: normal (default) / inverted / both | `DCSPolarity`, `TestDCSGateAcceptsOnlyConfiguredPolarity` |
| Mismatch WARN | once per channel, gate stays shut | `warnDCSOppositePolarity`, `TestWarnDCSOppositePolarityOncePerChannel` |

## In this post

- **What the radio actually sends** — the 23-bit word, bit by bit.
- **The detector that never matched a real radio** — the self-consistent trap, round three.
- **From IQ to a matched word** — front end, slicer, four clocks, 46 targets.
- **Polarity, pinned on a service monitor** — `dcs_polarity`, the gate, the WARN.
- **The aliases you cannot fix** — rotations and complements.
- **Configuring it, and reading the log** — the YAML and the two lines it prints.

## What the radio actually sends

A DCS transmitter encodes a three-digit octal code — 023, 754, 411 — as nine
bits C0..C8, least significant bit first, appends the fixed bits 0 0 1 and
eleven Golay parity bits P0..P10, and loops the 23-bit word forever at 134.4
bits per second as NRZ deviation under the voice. There is no sync word: the
`001` is the only fixed pattern, and because the word is a cyclic
Golay(23,12) codeword a receiver may lock onto any of its 23 rotations.

`dcsCodewordFromOctal` builds it: `data = code | 1<<11`, then a systematic
cyclic encode over the transmit order with generator `x^11+x^9+x^7+x^6+x^5+x+1`
(`dcsGolayPoly` = `0xAE3`), parity in bits 12..22. **Bit i of the result is
the i-th bit sent.** For 023 that is `0x763813`:

<figure class="lab-figure">
<svg viewBox="0 0 680 150" width="680" height="150" role="img" aria-label="A row of 23 boxes, left to right, labelled as the on-air order of a DCS word: bits 0 to 8 hold the code C0 through C8 with the least significant bit first, bits 9 to 11 hold the fixed values 0 0 1, and bits 12 to 22 hold the Golay parity P0 through P10. Below, the word for code 023 is written as 0x763813 and an arrow marks that bit 0 goes out first and the word repeats without a gap.">
  <text x="340" y="14" text-anchor="middle" fill="currentColor" font-size="10" font-weight="bold">one DCS word on air · 23 bits · 134.4 baud · repeated without a gap</text>
  <g font-size="8" fill="currentColor">
    <rect x="20" y="30" width="243" height="28" fill="none" stroke="currentColor" stroke-width="1.5"/>
    <text x="141" y="48" text-anchor="middle">bits 0..8 · C0 … C8 (code, LSB first)</text>
    <rect x="263" y="30" width="81" height="28" fill="none" stroke="var(--accent)" stroke-width="1.5"/>
    <text x="303" y="48" text-anchor="middle" fill="var(--accent)">bits 9..11 · 0 0 1</text>
    <rect x="344" y="30" width="316" height="28" fill="none" stroke="currentColor" stroke-width="1.5"/>
    <text x="502" y="48" text-anchor="middle">bits 12..22 · P0 … P10 (Golay parity, g = 0xAE3)</text>
  </g>
  <line x1="20" y1="72" x2="660" y2="72" stroke="var(--fg-muted)"/>
  <path d="M20 72 L34 66 L34 78 Z" fill="var(--fg-muted)"/>
  <text x="24" y="90" fill="var(--fg-muted)" font-size="8">bit 0 goes out first</text>
  <text x="656" y="90" text-anchor="end" fill="var(--fg-muted)" font-size="8">… then bit 0 again: any of 23 rotations may be the first one heard</text>
  <text x="340" y="118" text-anchor="middle" fill="currentColor" font-size="9">code 023 → word 0x763813 · pinned for all 512 codes by refDCSWord (published parity equations)</text>
  <text x="340" y="136" text-anchor="middle" fill="var(--fg-muted)" font-size="8">the old detector held the code in the HIGH bits with the sync as "100" in the LOW bits, slid MSB-first — no radio ever matched</text>
</svg>
<figcaption>The layout the reference test pins. The parity equations are independent of the encoder in dcs.go, so a wrong construction on either side now fails.</figcaption>
</figure>

The parity bits are where the independence lives. `refDCSWord` in
`dcs_reference_test.go` never calls the encoder; it types in the eleven
published parity equations (P0 = C0+C1+C2+C3+C4+C7, P1 = NOT(C1+…+C8), … —
the equations SDRangel's `NFMModDCS::setDCS` transmits), and
`TestDCSCodewordMatchesPublishedParity` demands agreement on all 512 codes
and the literal `0x763813` for 023.

## The detector that never matched a real radio

This is the
[self-consistent trap]({{ '/blog/solution-postmortem/from-the-issue-tracker-20-self-consistent-trap/' | relative_url }})
again. The header of `dcs.go` records what was wrong before
[#1184](https://github.com/MattCheramie/GopherTrunk/issues/1184): the
codeword "put the code in the high bits and the sync as `100` in the low
bits, and it slid bits in MSB-first; its test synthesizer transmitted that
same invented layout, so every unit test passed." A round-trip test cannot
see a mistake it makes twice.

Two more defects rode along, the same family as the CTCSS fixes in
[Part 5]({{ '/blog/tutorials/conventional-scanner-05-ctcss-done-right/' | relative_url }}):
the old detector ran its discriminator on the **whole 2.4 MHz SDR band**, so
the span's FM noise swamped a signalling tone a few hundred hertz wide, and
it **sliced on the raw sign** — but the discriminator carries the carrier
offset as DC, so a radio 600 Hz off frequency pinned every bit to one value.
The reporter's summary, "DCS squelch does not detect any code, normal or
inverted", is exactly what both produce.

The regression that fails first is `TestDCSDetectsOnAirWordAtSDRRate`: a
narrowband radio's ~350 Hz of DCS deviation, 600 Hz off frequency, 25 dB
SNR, synthesized at the **2.4 MS/s the scanner actually feeds the detector**
— and synthesized from `refDCSWord`, not from the encoder under test. Codes
023, 754 and 411, both senses, must be continuously present after
`dcsMinDwell`; a 754 transmission and a bare carrier with CTCSS must never
open a 023 gate (`TestDCSRejectsOtherCodeAndBareCarrierAtSDRRate`).

## From IQ to a matched word

`NewDCSDetector` takes a `DCSConfig{SampleHz, Code, AudioCutoffHz, Polarity}`
and returns nil on a bad code or a zero rate — in which case the scanner
falls back to power-only squelch and says so with
`conv: DCS detector failed to initialise; tone gate disabled — every signal passes the gate`.
The chain it builds:

```go
// internal/scanner/conventional/dcs.go (shape) — NewDCSDetector
fe := newToneFrontEnd(cfg.SampleHz)         // decimate → ±8 kHz → discriminator, rad/sample @ 48 kHz
spb := fe.rate / dcsBitRate                 // samples per bit at 134.4 baud
d := &DCSDetector{
    lpfAlpha:          onePoleAlpha(250, fe.rate),
    blkLen:            int(spb * 23 / 2),   // half a word per slicing-level block
    samplesPerBit:     spb,
    targets:           dcsRotations(target), // 23 rotations × 2 polarities = 46
    distanceThreshold: 2,
    wordLen:           int(spb * 23),
}
```

The `toneFrontEnd` is Part 5's: decimation to ~48 kHz, a ±8 kHz channel
filter (`toneChannelCutoffHz`) that admits one NFM channel and rejects the
rest of the band, and a discriminator scaled so its output means the same
deviation at any SDR rate. A 250 Hz single-pole low-pass then rolls the
voice band off before the bit integrator.

The slicing level is the fix for the carrier-offset trap. `trackLevel` keeps
the maximum and minimum of the filtered discriminator over half-word blocks
(`blkLen` = 11.5 bits) and sets `mid` to the midpoint of the last two blocks'
extremes. A DCS word always holds both bit values, so the extremes bracket
the NRZ levels whatever DC the offset adds; nothing is sliced until the
first block completes (`levelValid`).

There is no clock recovery. `dcsPhases` = 4 integrate-and-dump bit clocks run
in parallel, staggered a quarter bit apart, so one is always within an
eighth of a bit of the transmitter's clock — plenty at 134.4 baud. Each
phase accumulates `x − mid` across a bit period, slices the sign, and shifts
the bit into a 23-bit window **with the oldest bit at bit 0**, so a window
aligned to a word boundary reads exactly the codeword. `checkTargets`
compares it against the 46 targets at Hamming distance ≤ `distanceThreshold`
(2): 46 XOR-and-popcount operations.

A single match opens nothing. A real stream matches at **every** bit shift
(each shift is the next rotation); random data that matches one window keeps
matching with probability about one half per bit. So the gate opens only
after `dcsConfirmBits` = 16 consecutive matches (~120 ms), cutting the
false-open rate by ~2^16, and closes once `sinceMatch` exceeds a word with no
phase matching. The first report needs half a word for the level, a full
window, sixteen confirm bits and the filter delay — `dcsMinDwell` = 600 ms,
which `minToneDwell` raises `MinDwellPerChannel` to.

## Polarity, pinned on a service monitor

NRZ has a sign: does a 1 bit go out as positive or negative deviation?
Radios expose it as a suffix — D023N or D023I — and the `dcsRotations` table
has always carried both senses (odd entries are the complements). The first
fixed detector simply accepted either, which is wrong for the reason the
next section gives, so the convention was measured instead of assumed.

The instrument was the `nrz_inverted` field on the gate-open log line, added
for exactly this purpose. On 24 September the reporter ran the fixed build
against their Kenwood NX-300/NX-5000 and a service monitor, and the line
settled it: **D025N → `nrz_inverted=false`, D025I → true.** A radio set to N
sends the code's 1 bits as positive deviation. `TestDCSReportsMatchedPolarity`
keeps the instrument honest in both senses.

From that run (commit `182ae6e`, 25 September) the gate accepts only one
sense. `tone.dcs_polarity` parses through `ParseDCSPolarity` — `""`, `normal`
or `n`; `inverted` or `i`; `both` — and `DCSPolarity.accepts` decides which
rotations may open the gate. The other sense is still tracked, but only to
raise a hint:

```go
// internal/scanner/conventional/dcs.go (shape) — in Process, per phase
match, opposite, inverted := d.checkTargets(p.window)
if !match {
    p.streak = 0
    if opposite {               // the code, in the sense the gate rejects
        p.oppStreak++
        if p.oppStreak >= dcsConfirmBits { d.oppositeHeard = true }
    }
    continue
}
p.streak++
if p.streak >= dcsConfirmBits && !d.present {
    d.present, d.inverted = true, inverted
}
```

`TakeOppositeHeard` hands that flag to the scanner once, and
`warnDCSOppositePolarity` logs a single WARN per channel for the life of the
process (`dcsPolarityWarned` is keyed on the detector), with the gate kept
shut:

```text
WRN conv: DCS code heard with the opposite polarity; gate kept shut — set tone.dcs_polarity to match the radio (N = normal, I = inverted) or both freq_hz=447100000 label="UHF Simplex" index=0 dcs_code=025 dcs_polarity=normal heard=inverted
```

`TestDCSGateAcceptsOnlyConfiguredPolarity` walks all six combinations —
normal/inverted/both against a transmitter sending N or I — and asserts the
gate opens only where it should and that `TakeOppositeHeard` is set exactly
when it should not. `TestWarnDCSOppositePolarityOncePerChannel` pins the
once-per-channel rate and the `heard=inverted` / `dcs_polarity=normal`
fields.

## The aliases you cannot fix

Two properties of the real code fall out of its construction, and both are
pinned by `TestDCSCodewordAliasesMatchReferenceTables` against SDRangel's
`dcscodes.cpp` tables:

- **A rotation of one code's word is another code's word.** Read 023's word
  from a different start bit and you get 340 or 766; 054 reads as 405 or 675.
  The detector matches any rotation, so it cannot — and must not — try to
  distinguish them. A radio set to 340 opens a 023 gate. That is DCS.
- **The complement of one code's word is a rotation of another code's.**
  023 inverted is 047 normal; 754I is 116N; 627I is 031N. This is why
  accepting both senses was wrong: a `023` gate that took either polarity
  opened on every radio set to 047N, which is a different system.

With the default `normal` polarity, a 023 gate stays shut on 047N
(`TestDCSDefaultGateIgnoresAliasedCode`) — but it still opens on **047I**,
because 047I and 023N are the same bit pattern on air. No receiver can tell
them apart; `NewDCSDetector`'s docstring says so. `both` on 023 opens for
023N, 047I, 023I and 047N alike, which is why it is not the default.

## Configuring it, and reading the log

The scan-list entry, verified against `config.ConvToneConfig`'s YAML tags and
`config.example.yaml`:

```yaml
scanner:
  conventional:
    - label: "UHF Simplex"
      frequency_hz: 447100000
      mode: fm
      squelch_dbfs: -48
      hangtime_ms: 1500
      tone:
        mode: dcs            # ctcss | dcs | none
        dcs_code: "025"      # 3-digit octal; the radio's D025
        dcs_polarity: normal # normal (D025N, default) | inverted (D025I) | both
```

`config_validate.go` rejects a `dcs_code` that is not three octal digits and
a `dcs_polarity` outside the three spellings; the scanner's `validateTone`
repeats the check for channels added at runtime. Two log lines belong to
this gate. The INFO on every open, with the field the polarity was pinned
from:

```text
INF conv: DCS gate opened freq_hz=447100000 label="UHF Simplex" dcs_code=025 nrz_inverted=false
```

And the once-per-channel WARN above. If it appears, the fix is in the config:
set `dcs_polarity` to what the radio sends, or `both` if the alias cost is
acceptable. A gate that never opens and never warns is a different code, or a
channel too weakly deviated for the slicer.

## Where this goes next

Opening the gate is half the job; what the recorder hears once it is open is
the other half.
[Part 7]({{ '/blog/tutorials/conventional-scanner-07-fm-voice-chain/' | relative_url }})
follows the dwell into the composer's analog FM chain — the decimating front
end, the selectable `fm_channel_bandwidth_hz` filter, the discriminator, the
300 Hz high-pass that strips exactly the sub-audible tones this part and the
last one detected, de-emphasis, the 3.4 kHz low-pass — and explains which of
`recordings.fm_*` keys to touch, and when.

## FAQ

**Why did GopherTrunk's DCS gate never open before issue #1184?**
Three defects, all masked by a self-consistent test suite: the codeword's bit
order was invented (code in the high bits, sync as `100` in the low bits,
MSB-first), the discriminator ran on the whole 2.4 MHz band, and the slicer
used the raw sign, so a carrier offset pinned every bit. `dcs.go` now builds
the on-air word (023 = `0x763813`), pinned by `dcs_reference_test.go`.

**What does tone.dcs_polarity do and which value should I use?**
It selects the NRZ sense that opens the gate: `normal` (default) matches a
radio's D023N, `inverted` D023I, `both` either. On air a radio set to N
sends the code's 1 bits as positive deviation (`nrz_inverted=false`), pinned
on the reporter's Kenwoods. Use the radio's own suffix.

**Why does my 023 DCS gate open on a radio set to 047?**
A code's inverted word is a rotation of a different code's normal word: 023I
and 047N are one bit pattern on air, as are 023N and 047I. A `normal` 023
gate stays shut on 047N but still opens on 047I; no receiver can tell those
apart. `TestDCSCodewordAliasesMatchReferenceTables` pins the pairs.

**What does "DCS code heard with the opposite polarity" mean?**
The configured code arrived in the NRZ sense the gate rejects, held for the
same 16 bits a gate-open needs — a radio set to I on a channel configured
`normal`, or the reverse. The gate stays shut; the WARN fires once per
channel. Set `tone.dcs_polarity` to match the radio.

## Series navigation

**Part 6 of 14** · ←
[Part 5: CTCSS Done Right — Exact Bins, Reverse Bins, Deviation Thresholds]({{ '/blog/tutorials/conventional-scanner-05-ctcss-done-right/' | relative_url }})
· Next →
[Part 7: The FM Voice Chain — De-emphasis, Band Limit, the 300 Hz High-Pass, Bandwidth]({{ '/blog/tutorials/conventional-scanner-07-fm-voice-chain/' | relative_url }})
