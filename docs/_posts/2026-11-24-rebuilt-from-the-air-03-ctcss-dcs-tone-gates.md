---
title: "Rebuilt From the Air, Part 3: CTCSS & DCS — A Rate Trap, a Rounded Bin and an Invented Codeword"
description: "GopherTrunk's CTCSS and DCS tone gates failed on air four separate ways across two rounds of issue 1184: a threshold in radians per sample calibrated at the wrong rate, a Goertzel bin rounded onto the neighbouring tone, a level set above narrowband radios, and a DCS codeword whose layout and bit order were invented."
category: solution-postmortem
keywords: ctcss decoder sdr, dcs digital coded squelch decoder, goertzel bin rounding, ctcss tone squelch sample rate, dcs codeword 0x763813, dcs polarity normal inverted, sub-audible tone detector, tone squelch scanner, self-consistent test trap, gophertrunk tone squelch
tags: [rebuilt-from-the-air, ctcss, dcs, tone-squelch, scanner, postmortem]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "Rebuilt From the Air"
series_part: 3
---

*Part 3 of **Rebuilt From the Air**, a 14-part postmortem series on the
GopherTrunk decoders that passed every test and never matched a real radio —
each diagnosed from a reference or a reporter's capture, rebuilt against
something independent, and pinned to real air where a capture exists.
[Part 2]({{ '/blog/solution-postmortem/rebuilt-from-the-air-02-mdc1200-tones-as-data/' | relative_url }})
found one wrong step between two right layers in MDC1200. This part stays on
the same scan list, where the sub-audible tone gates were wrong four
separate ways and took two rounds of one issue to find.*

> **TL;DR:** The conventional scanner's tone gates failed on air in four
> ways ([#1184](https://github.com/MattCheramie/GopherTrunk/issues/1184)).
> **Round 1:** the CTCSS threshold was a magnitude in radians per sample,
> calibrated by 48 kHz tests; the scanner feeds 2.4 MS/s, where the same
> deviation reads 50× smaller — a real tone sat about 3,000× under the
> threshold — so no gate ever opened. `toneFrontEnd` now decimates to
> ~48 kHz behind a ±8 kHz filter. **Round 2:** `toneout.NewGoertzel` rounds
> to the nearest bin, so a 162.2 Hz gate listened at 160 Hz and opened on
> 159.8; the level needed ~540 Hz of tone deviation where narrowband radios
> send ~350; and the DCS codeword's layout and bit order were invented, with
> `synthesizeDCSIQ` transmitting the same invention. All are fixed, pinned
> on real-air slices and published parity equations, and **on-air verified**
> 24 Sep 2026 — polarity included: D025N reads `nrz_inverted=false`.

**Key takeaways**

- **A threshold has units, and radians per sample is not a portable one.**
  The same tone is fifty times smaller at 2.4 MS/s than at 48 kHz; a number
  tuned at one rate means nothing at another.
- **Bin rounding is right for paging tones and wrong for CTCSS.** Tones
  hundreds of hertz apart survive a 5 Hz grid; tones 2.3 Hz apart do not.
- **DCS was the self-consistent trap in full.** The detector and its test
  transmitter shared an invented word, and only equations neither of them
  wrote could say so.
- **An instrument can settle what no reference states.** The polarity
  convention was pinned by logging which sense matched and keying a radio
  set to N and then to I.

## Cheat sheet

| | |
|---|---|
| Issue | [#1184](https://github.com/MattCheramie/GopherTrunk/issues/1184) — two rounds on the conventional scanner's tone gates |
| Symptom | round 1: no CTCSS tone ever opened a gate. Round 2: 162.2 Hz stayed shut on its own tone and opened on 159.8; on narrow FM no configured tone opened; no DCS code was detected, normal or inverted |
| Wrong theories | that 48 kHz unit tests covered a detector fed 2.4 MS/s; that DCS, slicing by sign, was rate-invariant and so probably fine |
| Real cause | a rad/sample threshold at the wrong rate over an unfiltered band; a Goertzel bin on a 5 Hz grid; a level above a narrowband tone; a DCS word and bit order shared only with its own test synthesiser |
| Fix | `toneFrontEnd`; `NewGoertzelExact` with 250 ms blocks and EIA neighbours as reverse bins; `ctcssMinDeviationHz`; `tonecode.DCSCodeword` in on-air order; `tone.dcs_polarity` |
| Diagnostic | real-air slices replayed at the production rate; bin arithmetic; the published DCS parity equations |
| Status | on-air verified 24 Sep 2026 on the reporter's Kenwood NX-300 and NX-5000 against a service monitor |

## In this post

- **Round 1** — the gate that never opened, and a unit that changes with rate.
- **Round 2** — the wrong tone opens, and no narrowband radio does.
- **DCS** — a codeword only GopherTrunk transmitted.
- **On air** — how a log field pinned the polarity convention.
- **What this overturns** — one sentence from July, and the rules that outlast it.

## Round 1: the gate that never opened

A scan-list channel can carry a `tone:` block. With it the scanner opens
only when the carrier is present *and* the configured
[CTCSS]({{ '/reference/ctcss/' | relative_url }}) tone or
[DCS]({{ '/reference/dcs/' | relative_url }}) code is detected. The
detectors' design is the subject of
[Squelch, Part 5]({{ '/blog/deep-dives/squelch-05-ctcss/' | relative_url }})
and
[Part 6]({{ '/blog/deep-dives/squelch-06-dcs/' | relative_url }}); this post
is how they came to be redesigned.

The reporter of #1184 — the bench that had already supplied the
[FleetSync captures]({{ '/blog/deep-dives/beyond-voice-04-fleetsync-wav-that-lied/' | relative_url }})
— put tones on channels at 447.100 MHz and keyed radios programmed with
them. The gate never opened on any configured tone.

The detector chain was short: FM-discriminate the IQ, low-pass at about
500 Hz, run a
[Goertzel]({{ '/reference/goertzel-algorithm/' | relative_url }}) resonator
at the tone, compare its magnitude with a threshold. The trap is in the
first step. A discriminator's output is the phase step
between consecutive samples: a
[deviation]({{ '/reference/fm-deviation/' | relative_url }}) of Δf at sample
rate fs is 2π·Δf/fs radians per sample. The threshold had been set against
unit tests that built their signal at 48 kHz with 1 kHz of tone deviation.
The scanner hands its detectors the SDR's own stream — 2.4 MS/s on an
RTL-SDR — where the same tone's phase step is **fifty times smaller**. The
Goertzel reports squared amplitude, so the rate alone costs a factor of
2,500, and a real tone of a few hundred hertz sat about 3,000 times under
the threshold. Nothing could open it.

A second fault sat under the first. At 2.4 MS/s the discriminator saw the
whole tuner span, and the FM noise of the rest of the band swamps a
sub-audible tone in one channel. The
[channel-power squelch]({{ '/blog/deep-dives/squelch-02-channel-power/' | relative_url }})
had the same blind spot.

No round trip could catch either. The tests and the detector agreed on a
sample rate the scanner never uses — the
[self-consistent trap]({{ '/blog/solution-postmortem/from-the-issue-tracker-20-self-consistent-trap/' | relative_url }})
with a shared *world* instead of a shared constant. The fix puts one front
end ahead of both tone detectors:

```go
// internal/scanner/conventional/tone_frontend.go (shape)
const toneRefRateHz = 48_000      // the rate thresholds are calibrated at
const toneChannelCutoffHz = 8_000 // one-sided channel filter

m := int(sampleHz / toneRefRateHz)        // 50 at 2.4 MS/s
pre := dsp.NewResampler(1, m, m*8+1, 8.6) // anti-aliased decimation
rate := sampleHz / float64(m)
chanFilter := filter.NewFIR(filter.LowpassKaiser(63, toneChannelCutoffHz/rate, 8.6))
discScale := rate / toneRefRateHz         // output is rad/sample at 48 kHz
```

Its output is in radians per sample at 48 kHz whatever the input rate, "so
a threshold means the same deviation at any SDR sample rate."

The regression had to run at the rate that failed. `ctcss_realair_test.go`
loads two 1.2 s slices of the reporter's captures — a handheld on
447.100 MHz narrowband FM, keyed with and without a 100.0 Hz tone — stored
at 48 kHz and interpolated back to 2.4 MS/s by the test itself, then fed in
16,384-sample chunks as the dwell loop delivers them.
`TestCTCSSDetectsRealAirToneAtSDRRate` reads 0 % present on the old code.
With the front end the tone was detected in 81 % of chunks on the
reporter's captures — the misses being the first Goertzel block and the
gaps around each keyup — while the neighbouring 94.8, 103.5 and 107.2 Hz
gates and the tone-free transmission stayed shut. That shipped in v1.2.0.
(One limit: an interpolated slice has real voice, tone and receiver noise
at the production rate but no out-of-channel energy, so it pins the unit,
not the band-noise half.)

The notes from that round carry one more sentence: the DCS detector "slices
by sign (rate-invariant); it has not been checked on air."

## Round 2: the gate that opened on the wrong tone

With gates opening at last, the reporter could test whether they opened on
the *right* thing. Three reports came back. A channel configured for
162.2 Hz stayed shut on 162.2 and opened when the radio sent 159.8, the
next tone down the table. "With narrow FM TX the squelch does not open for
any configured tone." And DCS detected no code, normal or inverted.

<figure class="lab-figure">
<svg viewBox="0 0 680 260" width="680" height="260" role="img" aria-label="Top: a frequency axis from 155 to 165 hertz with the old five-hertz bin grid. A gate set to 162.2 hertz was evaluated at 160 hertz, 0.2 hertz from a 159.8 hertz tone and 2.2 from its own. Bottom: the 23 bits of DCS code 023 in on-air order: nine code bits, a fixed 001, eleven parity bits.">
  <text x="340" y="14" text-anchor="middle" fill="currentColor" font-size="10" font-weight="bold">CTCSS: a 162.2 Hz gate on a 5 Hz bin grid (200 ms block)</text>
  <line x1="60" y1="74" x2="620" y2="74" stroke="var(--fg-muted)"/>
  <line x1="60" y1="69" x2="60" y2="79" stroke="var(--fg-muted)"/>
  <line x1="340" y1="40" x2="340" y2="79" stroke="var(--fg-muted)" stroke-width="2"/>
  <line x1="620" y1="69" x2="620" y2="79" stroke="var(--fg-muted)"/>
  <text x="60" y="92" text-anchor="middle" fill="var(--fg-muted)" font-size="8">155 Hz</text>
  <text x="352" y="92" text-anchor="start" fill="var(--fg-muted)" font-size="8">160 Hz</text>
  <text x="620" y="92" text-anchor="middle" fill="var(--fg-muted)" font-size="8">165 Hz</text>
  <text x="340" y="34" text-anchor="middle" fill="var(--fg-muted)" font-size="8">where the old "162.2" gate listened: bin 160.0</text>
  <line x1="329" y1="50" x2="329" y2="74" stroke="var(--accent)" stroke-width="2"/>
  <text x="322" y="108" text-anchor="end" fill="var(--accent)" font-size="8">159.8 Hz sent: 0.2 Hz from the bin — gate opens</text>
  <line x1="463" y1="46" x2="463" y2="74" stroke="currentColor" stroke-width="2"/>
  <text x="470" y="52" text-anchor="start" fill="currentColor" font-size="8">162.2 Hz sent: 2.2 Hz off — stays shut</text>
  <text x="470" y="64" text-anchor="start" fill="var(--accent)" font-size="8">now evaluated exactly here</text>
  <text x="340" y="140" text-anchor="middle" fill="currentColor" font-size="10" font-weight="bold">DCS 023 on air: 0x763813, bit 0 sent first, cycled at 134.4 bit/s</text>
  <text x="58" y="172" text-anchor="end" fill="var(--fg-muted)" font-size="8">first</text>
  <rect x="64" y="154" width="216" height="28" fill="none" stroke="currentColor"/>
  <rect x="280" y="154" width="72" height="28" fill="none" stroke="var(--fg-muted)"/>
  <rect x="352" y="154" width="264" height="28" fill="none" stroke="var(--accent)"/>
  <text x="76" y="172" text-anchor="middle" fill="currentColor" font-size="10">1</text>
  <text x="100" y="172" text-anchor="middle" fill="currentColor" font-size="10">1</text>
  <text x="124" y="172" text-anchor="middle" fill="currentColor" font-size="10">0</text>
  <text x="148" y="172" text-anchor="middle" fill="currentColor" font-size="10">0</text>
  <text x="172" y="172" text-anchor="middle" fill="currentColor" font-size="10">1</text>
  <text x="196" y="172" text-anchor="middle" fill="currentColor" font-size="10">0</text>
  <text x="220" y="172" text-anchor="middle" fill="currentColor" font-size="10">0</text>
  <text x="244" y="172" text-anchor="middle" fill="currentColor" font-size="10">0</text>
  <text x="268" y="172" text-anchor="middle" fill="currentColor" font-size="10">0</text>
  <text x="292" y="172" text-anchor="middle" fill="var(--fg-muted)" font-size="10">0</text>
  <text x="316" y="172" text-anchor="middle" fill="var(--fg-muted)" font-size="10">0</text>
  <text x="340" y="172" text-anchor="middle" fill="var(--fg-muted)" font-size="10">1</text>
  <text x="364" y="172" text-anchor="middle" fill="var(--accent)" font-size="10">1</text>
  <text x="388" y="172" text-anchor="middle" fill="var(--accent)" font-size="10">1</text>
  <text x="412" y="172" text-anchor="middle" fill="var(--accent)" font-size="10">0</text>
  <text x="436" y="172" text-anchor="middle" fill="var(--accent)" font-size="10">0</text>
  <text x="460" y="172" text-anchor="middle" fill="var(--accent)" font-size="10">0</text>
  <text x="484" y="172" text-anchor="middle" fill="var(--accent)" font-size="10">1</text>
  <text x="508" y="172" text-anchor="middle" fill="var(--accent)" font-size="10">1</text>
  <text x="532" y="172" text-anchor="middle" fill="var(--accent)" font-size="10">0</text>
  <text x="556" y="172" text-anchor="middle" fill="var(--accent)" font-size="10">1</text>
  <text x="580" y="172" text-anchor="middle" fill="var(--accent)" font-size="10">1</text>
  <text x="604" y="172" text-anchor="middle" fill="var(--accent)" font-size="10">1</text>
  <text x="172" y="198" text-anchor="middle" fill="currentColor" font-size="8">C0 … C8: the code, low bit first</text>
  <text x="316" y="198" text-anchor="middle" fill="var(--fg-muted)" font-size="8">fixed 0 0 1</text>
  <text x="484" y="198" text-anchor="middle" fill="var(--accent)" font-size="8">P0 … P10: Golay(23,12) parity</text>
  <text x="340" y="224" text-anchor="middle" fill="var(--fg-muted)" font-size="8">old model: code in the high bits, a "100" in the low bits, bits read MSB-first — and a test synthesiser that sent the same</text>
  <text x="340" y="242" text-anchor="middle" fill="var(--fg-muted)" font-size="8">pinned by the published parity equations for all 512 codes, and by the aliases only a real word has</text>
</svg>
<figcaption>Two unrelated ways to listen for the wrong thing: a resonator rounded onto the neighbouring tone, and a codeword no radio sends.</figcaption>
</figure>

**The rounded bin.** `toneout.NewGoertzel` picks its frequency as
`k := math.Round(float64(blockSize) * targetHz / sampleHz)` — the nearest
exact DFT bin. It was written for two-tone paging
([Beyond Voice, Part 5]({{ '/blog/deep-dives/beyond-voice-05-two-tone-tone-out/' | relative_url }})),
where tones are hundreds of hertz apart and an exact bin is a virtue. The
CTCSS detector used it with a 200 ms block, which puts a bin every 5 Hz.
The EIA tone table has neighbours 2.3 Hz apart. Configured for 162.2,
the gate listened at 160.0: a real 162.2 Hz tone was 2.2 Hz off its own
bin, and the 159.8 Hz tone one row down was 0.2 Hz from it. Nineteen pairs
of adjacent tones cross-opened this way.

The detector now builds its resonators with `NewGoertzelExact`, which
evaluates at exactly the target frequency whatever the block size. The
block grew to 250 ms (`ctcssBlockSeconds`) — 4 Hz of resolution, at which a
tone leaks about 29 % of its power into a neighbour 2.3 Hz away — and the
adjacent EIA tones became **reverse bins**: the target must beat the
largest of them by `rejectRatio` (1.5), so a transmission on the next tone
up or down lands dead-on a bin that vetoes it. One pair is beyond a 250 ms
block: 150.0 and 151.4 Hz sit 1.4 Hz apart and each leaks about 66 % into
the other, so a gate on either opens on both — as on most radios — and
config validation refuses two logical channels on one frequency that only
that pair would separate.

**The level.** The threshold was a fixed magnitude of 5e-4. Run backwards
through the detector's gain, that is about 540 Hz of tone deviation. A
radio puts roughly 15 % of its peak deviation into CTCSS — about 350 Hz on a
2.5 kHz narrowband channel, 750 Hz on a 5 kHz one. A typical narrowband
tone sat under the bar. The threshold is now computed from a stated
deviation:

```go
// internal/scanner/conventional/ctcss.go (shape)
const ctcssMinDeviationHz = 100

func ctcssToneMagnitude(devHz, toneHz, cutoffHz float64) float64 {
	a := 2 * devHz / toneRefRateHz
	g := 1 / (1 + (toneHz/cutoffHz)*(toneHz/cutoffHz))
	return a * a * g
}
```

A bare carrier still stays shut: it puts nothing coherent in the target bin
and the reverse-bin ratio holds. Three tests hold the round:
`TestCTCSSReportedToneOffsetAtSDRRate` reproduces the report at 2.4 MS/s;
`TestCTCSSOpensOnEveryEIAToneAtNarrowbandDeviation` keys the table at
350 Hz; and `TestCTCSSRealAirOnlyTheSentToneOpens` sweeps all 51 entries
across both real-air slices — only 100 Hz may open on the toned one,
nothing on the other. On the reporter's captures the tone is now present in
100 % of chunks.

## DCS: a codeword only GopherTrunk transmitted

DCS cycles a 23-bit Golay(23,12) word at 134.4 bit/s under the voice. The
old detector built that word with the code in the high bits and a "100" in
the low bits, and shifted received bits in MSB-first. Its test synthesiser,
`synthesizeDCSIQ`, transmitted the same layout. Every unit test passed; no
radio could match — fabricated framing of the
[SmartNet]({{ '/blog/deep-dives/from-spec-to-shipping-08-smartnet-rebuild/' | relative_url }})
kind, in 23 bits. Two front-end faults sat on top: the discriminator ran
on the whole 2.4 MHz span, and the bits were sliced on the raw sign, so a
carrier offset — a DC term on a discriminator — pinned every bit to one
value. "Rate-invariant" had been true and beside the point.

On the air, bit 0 goes first: the nine code bits C0..C8, least significant
first; a fixed `0 0 1`; then eleven parity bits P0..P10. Code 023 is the
word `0x763813`.

What makes the rebuilt word trustworthy is that two derivations meet.
Production (`tonecode.DCSCodeword`) computes the parity as the remainder of
a systematic Golay(23,12) encoding with generator `0xAE3`. The test
(`dcs_reference_test.go`) builds the word from the eleven published parity
equations instead — P0 = C0+C1+C2+C3+C4+C7 and so on — in a function written
"independently of dcsCodewordFromOctal on purpose."
`TestDCSCodewordMatchesPublishedParity` compares the two for all 512 codes
and checks the literal `0x763813`. A second test holds the word to
properties only a real DCS word has, taken from a reference modulator's
code tables: read from a different start bit, 023 is also 340 and 766; and
023 inverted is 047. And a new test transmitter behaves like a radio:
`synthDCSOnAir` sends bit 0 first at a narrowband 350 Hz deviation, 600 Hz
off frequency, from a random start bit and sub-bit phase, at 2.4 MS/s.

The detector changed to match the air: the shared front end, a
slicing level halfway between the recent high and low, four staggered
bit-clock phases, a 23-bit window compared with all 46 rotations at Hamming
distance ≤ 2, and sixteen consecutive matching bits before the gate opens.

## On air: the log field that pinned the polarity

One fact was still open: which electrical sense a radio's "N" setting is.
So the detector reported it. `Inverted()` records the polarity of the
match, and the scanner logs it:

```text
conv: DCS gate opened freq_hz=… label=… dcs_code=025 nrz_inverted=false
```

On 24 Sep the reporter keyed Kenwood NX-300 and NX-5000 radios against a
service monitor and read the field back. **D025N gives
`nrz_inverted=false`** — the code's 1 bits arrive as positive deviation —
and **D025I gives `true`**. With the convention known, the gate could stop
accepting both senses:

```yaml
scanner:
  conventional:
    - label: "UHF repeater"
      frequency_hz: 447100000
      mode: nfm
      tone:
        mode: dcs
        dcs_code: "023"
        dcs_polarity: normal   # the radio's D023N; inverted for D023I; or both
```

The other sense is still tracked, only to warn once per channel:
`conv: DCS code heard with the opposite polarity; gate kept shut`. One
ambiguity is left on purpose. A code's inverted word is a rotation of a
different code's normal word, so 023N and 047I are the same bits on air and
a normal 023 gate opens on 047I. No receiver can tell them apart, and the
detector's own comment calls it inherent in DCS. The same run confirmed the
CTCSS fixes, and
[Cookbook Volume 2, Part 4]({{ '/blog/tutorials/cookbook-volume-2-04-fleet-signalling/' | relative_url }})
has the working recipe.

## What this overturns

[Protocol Decoders, Part 9]({{ '/blog/deep-dives/protocol-decoders-09-conventional-wideband/' | relative_url }})
said in July that "CTCSS is fully wired; DCS parses and validates but its
detector is a tracked follow-up." The wiring was real. The CTCSS gate behind
it had never opened on a radio, and the DCS detector that followed matched
a word nothing transmits. Both gates are on-air verified as of 24 Sep 2026,
with `tone.dcs_polarity` defaulting to normal.
The recipe in
[The Operator's Cookbook, Part 6]({{ '/blog/tutorials/operator-cookbook-06-analog-fm-tone-out/' | relative_url }})
stands.

## What we keep

- **Give every threshold a unit that survives a rate change.** A deviation
  in hertz does; a magnitude in radians per sample does not. Derive the
  number from the physical quantity, as `ctcssToneMagnitude` does.
- **Pin the production rate, not the convenient one.** A real-air slice
  interpolated back to 2.4 MS/s tests what the scanner runs.
- **Know what a shared primitive rounds.** `NewGoertzel` was correct for the
  job it was written for; reuse moved it somewhere its rounding mattered.
- **Derive a codeword two ways.** Polynomial division in production,
  parity equations in the test, a literal for one code, aliases as a
  cross-check: four chances to disagree.
- **Log the ambiguous bit and ask.** `nrz_inverted` cost one field and
  turned a convention into a measurement.

## Where this goes next

The tone gates were small detectors. The next decoder had a whole protocol
stack and an encoder to match.
[Part 4]({{ '/blog/solution-postmortem/rebuilt-from-the-air-04-flex-four-modes/' | relative_url }})
is FLEX paging: a decoder that hunted its mode code on the wrong side of
the sync marker, and a rebuild that pins 1600 bps on five frames off the
air and says plainly what 3200 and 6400 bps still lack.

## FAQ

**Why did the CTCSS tone squelch never open on a real signal?**
Its threshold was a magnitude in radians per sample tuned against 48 kHz
test signals, but the scanner feeds the detector the SDR's 2.4 MS/s stream,
where the same tone deviation reads fifty times smaller. A real tone sat
about 3,000 times under the threshold, across an unfiltered band. The
detector now decimates to about 48 kHz behind a ±8 kHz channel filter.

**Why did a 162.2 Hz CTCSS gate open on 159.8 Hz?**
The Goertzel resonator was rounded to the nearest bin of a 200 ms block,
which is a 5 Hz grid, so the "162.2 Hz" bin was really 160.0 Hz — 0.2 Hz
from a 159.8 Hz tone and 2.2 Hz from its own. Resonators are now evaluated
at the exact frequency, with the adjacent EIA tones as reverse bins.

**What does a DCS codeword look like on the air?**
Twenty-three bits cycled at 134.4 bit/s, bit 0 first: the nine bits of the
octal code least significant first, a fixed 0 0 1, then eleven Golay
parity bits. Code 023 is `0x763813`. Because the word repeats without a
sync, any rotation of it is the same signal, which is why some codes alias.

**How do I set DCS polarity in GopherTrunk?**
With `dcs_polarity` in the channel's `tone:` block: `normal` (the default,
a radio's D023N), `inverted` (D023I) or `both`. If the configured code is
heard in the other sense the gate stays shut and the scanner logs one
warning for that channel naming the setting.

**Are the CTCSS and DCS gates verified on air?**
Yes — on 24 Sep 2026, on the reporter's Kenwood NX-300 and NX-5000 against
a service monitor. That run also pinned the polarity convention: D025N reads
`nrz_inverted=false` and D025I reads `true`.

## Series navigation

**Part 3 of 14** · ←
[Part 2: MDC1200 — The Tones Were Sliced as the Data]({{ '/blog/solution-postmortem/rebuilt-from-the-air-02-mdc1200-tones-as-data/' | relative_url }})
· Next →
[Part 4: FLEX — The Mode Code Comes Before the Marker]({{ '/blog/solution-postmortem/rebuilt-from-the-air-04-flex-four-modes/' | relative_url }})
