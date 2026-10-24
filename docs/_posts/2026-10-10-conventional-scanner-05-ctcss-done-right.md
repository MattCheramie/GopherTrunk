---
title: "The Conventional Scanner, Part 5: CTCSS Done Right — Exact Bins, Reverse Bins, Deviation Thresholds"
description: "The three defects issue 1184 found in GopherTrunk's CTCSS gate and how ctcss.go fixes them: a Goertzel bin rounded to a 5 Hz grid so 162.2 Hz opened on 159.8, a threshold calibrated at 48 kHz but fed 2.4 MS/s, and a floor needing 540 Hz of deviation no narrowband radio sends; now exact bins, reverse bins, a 100 Hz floor."
category: tutorials
keywords: ctcss decoder sdr, ctcss squelch goertzel, ctcss tone not detected sdr scanner, 162.2 hz opens on 159.8, eia ctcss tone table, narrowband fm ctcss deviation, goertzel exact frequency bin, reverse bin tone rejection, ctcss 150.0 151.4 inseparable, gophertrunk conventional scanner
tags: [conventional-scanner, ctcss, goertzel, tone-squelch, analog-fm, tutorial]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "The Conventional Scanner"
series_part: 5
---

*Part 5 of **The Conventional Scanner**, a 14-part operator's tutorial on
GopherTrunk's `scanner.conventional` scan list.
[Part 4]({{ '/blog/tutorials/conventional-scanner-04-squelch-on-the-channel/' | relative_url }})
made the carrier squelch measure the channel, which restored the tone gate
to its proper role as a second condition. This part is the tone gate
itself — `CTCSSDetector` in `internal/scanner/conventional/ctcss.go` — and
the three separate defects
[#1184](https://github.com/MattCheramie/GopherTrunk/issues/1184)'s reporter
found in it with a lab of Kenwood and Radtel radios: a tone that never
opened at all, a tone that opened on the wrong neighbour, and a narrowband
radio that never opened anything. Each had its own cause, each has its own
pin, and all three are on-air verified.*

> **TL;DR:** `CTCSSDetector` runs IQ → `toneFrontEnd` (decimate to ~48 kHz
> behind a `m·8+1`-tap anti-alias FIR, ±8 kHz channel filter, quadrature
> discriminator scaled to radians per sample *at 48 kHz*) → a one-pole
> ~500 Hz low-pass → Goertzel at **exactly** the target (`NewGoertzelExact`,
> no bin rounding) over `ctcssBlockSeconds` = 0.25 s (4 Hz resolution),
> compared against reverse bins at ±5 Hz and at the adjacent EIA tones
> (`eiaNeighbourTones`), with `rejectRatio` 1.5 and a magnitude floor
> derived from `ctcssMinDeviationHz` = 100 Hz by `ctcssToneMagnitude`.
> Three #1184 defects: the threshold was calibrated in rad/sample at 48 kHz
> but the scanner fed 2.4 MS/s, where the same deviation reads 50× smaller
> (~3000× in power) — the gate never opened; `NewGoertzel` rounded the bin
> to a 5 Hz grid, so "162.2" was 160 Hz and 159.8 opened it; the fixed 5e-4
> floor needed ~540 Hz of deviation, so NFM radios at ~350 Hz never opened.
> 150.0/151.4 Hz (1.4 Hz apart) remain inseparable and open together.
> Pinned by `ctcss_eia_test.go` and the reporter's real-air slices in
> `ctcss_realair_test.go`; on-air verified 24 Sep on an NX-300/NX-5000.

**Key takeaways**

- **A detector threshold in rad/sample is a sample-rate trap.** Calibrate
  at a reference rate and normalise the input to it, or the gate is wrong
  at every other rate.
- **Round nothing when tones sit 2.3 Hz apart.** `NewGoertzelExact`
  measures the DTFT at the configured frequency; the magnitude formula is
  exact at any frequency, so rounding bought nothing.
- **Reverse bins are the real selectivity.** The adjacent EIA tones are
  measured too, and the target must beat the loudest by 1.5×.
- **Set the floor from physics.** Radios put ~15 % of peak deviation into
  CTCSS; 100 Hz keeps >10 dB margin under an NFM tone.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| Front end | ↓m to ~48 kHz, ±8 kHz LPF, discriminator × `discScale` | `toneFrontEnd` (`tone_frontend.go`), `toneRefRateHz`, `toneChannelCutoffHz` |
| Audio roll-off | one-pole IIR, `AudioCutoffHz` default 500 | `onePoleAlpha`, `lpfAlpha` |
| Target bin | exact DTFT at `TargetHz`, block = 0.25 s × rate | `toneout.NewGoertzelExact`, `ctcssBlockSeconds` |
| Reverse bins | ±`ctcssReverseOffsetHz` (5) + EIA neighbours within 10 Hz, none closer than 2 Hz | `eiaNeighbourTones`, `ctcssMinNeighbourSpacingHz`, `EIACTCSSTones` |
| Decision | `targetMag ≥ magThreshold` and `≥ rejectRatio × max(reverse)` | `Process`, `ctcssToneMagnitude(ctcssMinDeviationHz, …)` |
| Pins | every EIA tone at 350 Hz dev opens; neighbours at 750 Hz never; real air at 2.4 MS/s | `ctcss_eia_test.go`, `ctcss_realair_test.go`, `testdata/ctcss*_radtel_447100k_48k.cs16` |

## In this post

- **Three reports, three defects** — what each symptom pointed at.
- **The chain** — front end, low-pass, Goertzel, decision.
- **Exact bins and the 250 ms block** — the rounding bug and the resolution trade.
- **Reverse bins** — why the adjacent EIA tones are measured too, and the pair that cannot be.
- **A floor set by deviation** — `ctcssMinDeviationHz` and the formula behind it.
- **The rate trap and the real-air pin** — 2.4 MS/s, the Radtel slices, and the 24 Sep run.

## Three reports, three defects

The #1184 thread accumulated three CTCSS complaints that looked like one
("tones are not being detected") and were not. First, on the reporter's
Radtel keyed on 447.100 MHz with a 100.0 Hz tone, the gate never opened at
all. Second, with 162.2 Hz and 192.8 Hz configured, the gate stayed shut on
those tones but *opened* when the radio sent the next tone down the table,
159.8 and 189.9 Hz. Third, "with narrow FM TX the squelch does not open for
any configured tone". The causes, in the order found: a threshold
calibrated at one sample rate and applied at another; a Goertzel bin
rounded to a 5 Hz grid; and a fixed floor that needed more deviation than a
narrowband radio puts into its tone. The
[Cookbook]({{ '/blog/tutorials/operator-cookbook-06-analog-fm-tone-out/' | relative_url }})
shows the config that gates on a tone; the glossary's
[CTCSS entry]({{ '/reference/ctcss/' | relative_url }}) has the background.

## The chain

The file header gives the pipeline and the implementation follows it.
`toneFrontEnd` (shared with DCS and, since Part 4, the power meter) decimates the SDR-rate IQ to ~`toneRefRateHz` = 48 000 behind
an anti-alias FIR, channel-filters at ±8 kHz, and FM-discriminates with
`arg(z[n]·conj(z[n−1]))`, multiplying by `discScale = rate / toneRefRateHz`
so the output is in radians per sample *at 48 kHz* whatever the input rate.
A one-pole IIR low-pass (`onePoleAlpha(500, rate)`) rolls the audio band off
above ~500 Hz — above the highest CTCSS tone, below the lowest voice
formant. Then every sample is scaled into int16 range by 32768/π and fed to
every Goertzel at once — one at the target, several reverse — because they
share a block size and report on the same sample:

```go
// internal/scanner/conventional/ctcss.go (shape) — Process
targetMag, ready := d.goertzel.Process(sample)
var maxReverseMag float64
for _, rb := range d.reverseBins {
    if rmag, _ := rb.Process(sample); rmag > maxReverseMag { maxReverseMag = rmag }
}
if !ready { continue }
if targetMag < d.magThreshold                       { d.present = false; continue }
if d.rejectRatio > 0 && targetMag < d.rejectRatio*maxReverseMag { d.present = false; continue }
d.present = true
```

`present` is sticky between blocks, and `Reset` clears everything on each
retune. `NewCTCSSDetector` returns
nil for a zero sample rate or target, which is why `buildDetector` WARNs
`conv: tone gating configured but scanner sample rate is zero; tone gate disabled`
instead of quietly passing everything. The minimum dwell rises to
`ctcssMinDwell` = 350 ms whenever a CTCSS channel exists, so the first
250 ms block completes inside the scan window.

## Exact bins and the 250 ms block

`toneout.NewGoertzel` — the detector the
[two-tone paging decoder]({{ '/blog/deep-dives/beyond-voice-05-two-tone-tone-out/' | relative_url }})
uses — computes its coefficient from a bin index `k = round(N·f/fs)`. For
paging tones hundreds of hertz apart that is fine. For CTCSS it is fatal:
with a 200 ms block the bin spacing is 5 Hz, so the "162.2 Hz" detector was
really a 160.0 Hz detector, which a 159.8 Hz tone hits dead-on and a real
162.2 Hz tone misses by 2.2 Hz — the second report exactly.
`NewGoertzelExact` drops the rounding (`omega = 2π·f/fs`) and loses
nothing, because the magnitude formula `Process` evaluates is exact at any
frequency. `TestCTCSSReportedToneOffsetAtSDRRate` reproduces the report
at the production rate: 162.2 and 192.8 Hz configured must open on
themselves at 350 and 750 Hz deviation and must stay shut on 159.8 and
189.9 Hz.

Block length is the resolution trade. `ctcssBlockSeconds` = 0.25 gives
4 Hz resolution. Adjacent EIA tones sit 2.3–3.0 Hz apart; at the former
200 ms block (5 Hz) an on-tone signal leaked ~57 % of its power into a
neighbour 2.3 Hz away, leaving the rejection ratio almost no margin; at
4 Hz the leak is ~29 %, and a detection still lands inside the ~300 ms
decode time radios quote.

<figure class="lab-figure">
<svg viewBox="0 0 680 210" width="680" height="210" role="img" aria-label="A frequency axis from 155 to 170 hertz with EIA tones at 156.7, 159.8, 162.2, 165.5 and 167.9. Above it the old rounded bin is a lobe centred on 160.0 hertz, hitting 159.8 and missing 162.2. Below, the exact detector is a lobe centred on 162.2 flanked by reverse bins at plus and minus 5 hertz and at the EIA neighbours; the target must exceed 1.5 times the loudest reverse bin.">
  <text x="340" y="14" text-anchor="middle" fill="currentColor" font-size="10" font-weight="bold">a 162.2 Hz gate: rounded bin (old) vs exact bin + reverse bins (new)</text>
  <line x1="40" y1="100" x2="640" y2="100" stroke="var(--fg-muted)"/>
  <text x="40" y="114" text-anchor="middle" fill="var(--fg-muted)" font-size="8">155 Hz</text>
  <text x="640" y="114" text-anchor="middle" fill="var(--fg-muted)" font-size="8">170 Hz</text>
  <line x1="108" y1="96" x2="108" y2="104" stroke="currentColor"/>
  <text x="108" y="126" text-anchor="middle" fill="currentColor" font-size="8">156.7</text>
  <line x1="232" y1="96" x2="232" y2="104" stroke="currentColor"/>
  <text x="232" y="126" text-anchor="middle" fill="currentColor" font-size="8">159.8</text>
  <line x1="328" y1="96" x2="328" y2="104" stroke="var(--accent)" stroke-width="2"/>
  <text x="328" y="126" text-anchor="middle" fill="var(--accent)" font-size="8" font-weight="bold">162.2</text>
  <line x1="460" y1="96" x2="460" y2="104" stroke="currentColor"/>
  <text x="460" y="126" text-anchor="middle" fill="currentColor" font-size="8">165.5</text>
  <line x1="556" y1="96" x2="556" y2="104" stroke="currentColor"/>
  <text x="556" y="126" text-anchor="middle" fill="currentColor" font-size="8">167.9</text>
  <text x="46" y="44" fill="var(--fg-muted)" font-size="8">old: NewGoertzel, 200 ms</text>
  <path d="M140 96 Q240 20 340 96" fill="none" stroke="var(--fg-muted)" stroke-dasharray="3 2"/>
  <line x1="240" y1="36" x2="240" y2="96" stroke="var(--fg-muted)" stroke-dasharray="2 2"/>
  <text x="240" y="30" text-anchor="middle" fill="var(--fg-muted)" font-size="8">bin = 160.0 Hz (5 Hz grid)</text>
  <text x="400" y="56" fill="var(--fg-muted)" font-size="8">159.8 hits it dead-on;</text>
  <text x="400" y="68" fill="var(--fg-muted)" font-size="8">162.2 misses by 2.2 Hz</text>
  <text x="46" y="150" fill="var(--accent)" font-size="8">new: NewGoertzelExact, 250 ms (4 Hz)</text>
  <path d="M248 104 Q328 190 408 104" fill="none" stroke="var(--accent)" stroke-width="1.5"/>
  <path d="M192 104 Q232 150 272 104" fill="none" stroke="currentColor"/>
  <path d="M420 104 Q460 150 500 104" fill="none" stroke="currentColor"/>
  <line x1="128" y1="104" x2="128" y2="136" stroke="currentColor" stroke-dasharray="2 2"/>
  <line x1="528" y1="104" x2="528" y2="136" stroke="currentColor" stroke-dasharray="2 2"/>
  <text x="128" y="146" text-anchor="middle" fill="currentColor" font-size="8">157.2 (−5)</text>
  <text x="528" y="146" text-anchor="middle" fill="currentColor" font-size="8">167.2 (+5)</text>
  <text x="232" y="164" text-anchor="middle" fill="currentColor" font-size="8">reverse: EIA 159.8</text>
  <text x="460" y="164" text-anchor="middle" fill="currentColor" font-size="8">reverse: EIA 165.5</text>
  <text x="340" y="200" text-anchor="middle" fill="var(--accent)" font-size="9">open only if target ≥ magThreshold AND target ≥ 1.5 × max(reverse bins)</text>
</svg>
<figcaption>The rounded bin sat on the wrong tone; the exact bin sits on the right one and the neighbours are measured as evidence against it.</figcaption>
</figure>

## Reverse bins

Selectivity comes from measuring the competition. `NewCTCSSDetector`
builds reverse Goertzels at the target ±`ctcssReverseOffsetHz` (5 Hz) —
catching non-EIA tones and broadband sub-audible energy — and at the EIA
tones immediately below and above the target (`eiaNeighbourTones`), from
the 51-entry `EIACTCSSTones` table when they lie within 10 Hz. A match
requires `targetMag ≥ rejectRatio × max(reverse)` with `rejectRatio` 1.5,
so a transmission on the next tone up or down the table — dead-on its own
reverse bin — can never open the gate however strong it is.
`TestCTCSSRejectsAdjacentEIATones` sweeps the table: every EIA neighbour at
full 750 Hz deviation must open the gate in exactly 0 % of chunks.
`SetRejectRatio(0)` is the escape hatch to the old magnitude-only behaviour.

One pair is excluded on purpose. 150.0 and 151.4 Hz are 1.4 Hz apart, under
`ctcssMinNeighbourSpacingHz` = 2.0, and a 250 ms block cannot separate them
— each leaks ~66 % into the other — so neither is the other's reverse bin
and a gate on either opens on both, as on most radios. The EIA tests skip
the pair; every other adjacent pair is separable at 4 Hz.

## A floor set by deviation

The third report was the magnitude floor. The old fixed threshold, 5e-4 in
the detector's normalised units, corresponded to ~540 Hz of peak FM
deviation in the tone. Radios put roughly 15 % of their peak deviation into
CTCSS: ~750 Hz on a 5 kHz wideband channel, which cleared the floor, and
~350 Hz on a 2.5 kHz narrowband channel, which never did — so no narrowband
radio ever opened the gate. The floor is now derived from a deviation:

```go
// internal/scanner/conventional/ctcss.go
const ctcssMinDeviationHz = 100

func ctcssToneMagnitude(devHz, toneHz, cutoffHz float64) float64 {
    a := 2 * devHz / toneRefRateHz                      // 2π·dev/rate, ÷π from the int16 scaling
    g := 1 / (1 + (toneHz/cutoffHz)*(toneHz/cutoffHz))  // one-pole low-pass power gain
    return a * a * g                                    // Goertzel reports squared amplitude
}
```

A steady tone at `toneHz` with `devHz` of deviation reads `2π·dev/rate`
radians per sample at 48 kHz, the int16 scaling divides by π, the low-pass
attenuates by `1/sqrt(1+(f/fc)²)`, and the Goertzel reports squared
amplitude. 100 Hz keeps >10 dB margin under an NFM tone, and a carrier
*without* a tone stays shut: nothing coherent in the target bin, and the
reverse-bin ratio holds.
`TestCTCSSOpensOnEveryEIAToneAtNarrowbandDeviation` is the NFM regression:
every EIA tone at 350 Hz deviation with 20 dB of noise must be present in
≥ 90 % of chunks after the first half second.

## The rate trap and the real-air pin

The first report — nothing ever opened — was the deepest, because the
detector's unit tests were green. They ran at 48 kHz with clean synthetic
tones, where the threshold was calibrated. The scanner feeds the detector
the SDR's full-rate IQ, 2.4 MS/s on an RTL-SDR, and the discriminator
output is radians per *sample*: the same deviation reads 50× smaller at
2.4 MS/s, ~3000× smaller in power, so a real tone sat three orders of
magnitude below its threshold. No test at the calibration rate could see
it — the
[self-consistent trap]({{ '/blog/solution-postmortem/from-the-issue-tracker-20-self-consistent-trap/' | relative_url }})
along the sample-rate axis. `toneFrontEnd` closes it structurally:
decimation to ~48 kHz, the ±8 kHz filter and `discScale` mean a threshold
expresses the same deviation at any input rate.

The pin is real air at the production rate. The reporter's captures — a
Radtel keyed on 447.100 MHz NFM with and without a 100.0 Hz tone, recorded
by the per-call voice IQ debug tap at 2.4 MS/s — are committed as two 1.2 s
slices channelised to 48 kHz
(`testdata/ctcss100_radtel_447100k_48k.cs16`,
`testdata/ctcss_none_radtel_447100k_48k.cs16`), and
`ctcss_realair_test.go` interpolates them back to 2.4 MS/s with
`dsp.NewResampler(50, 1, 16, 8.6)` so the detector runs at the rate the
scanner feeds it. `TestCTCSSDetectsRealAirToneAtSDRRate` demands the
100 Hz gate present in ≥ 95 % of chunks on the tone-carrying slice and
0 % for 94.8, 103.5 and 107.2 Hz;
`TestCTCSSRejectsRealAirToneFreeCarrierAtSDRRate` demands 0 % on the
tone-free slice; and `TestCTCSSRealAirOnlyTheSentToneOpens` sweeps all 51
tones over both captures (skipped under `-short`). Against the old code the first reads 0 %.

Verification is the strongest in this series so far. Offline, the real-air
slices pin the production rate. On air, the project's record of the 24 Sep
round-two fixes says all three — the exact bin, the deviation floor, and
Part 6's DCS bit order — were verified on the reporter's Kenwood NX-300 and
NX-5000 with a service monitor. What remains is by design: the 150.0/151.4
pair is inseparable, and the 350 ms min-dwell bump applies to every channel
in a list with one CTCSS entry.

### How the tone gate shaped the Go code

- **Calibrate at a reference rate, normalise to it.** `ctcssRefRateHz`
  aliases `toneRefRateHz`; every threshold is stated at 48 kHz and
  `discScale` makes the input match.
- **Exactness as a separate constructor.** `NewGoertzelExact` sits beside
  `NewGoertzel` so paging keeps its grid and CTCSS gets its precision.
- **Neighbours from the table, not from an offset.** `eiaNeighbourTones`
  reads `EIACTCSSTones`, so the reverse bins track the standard's real
  spacing.
- **Real-air fixtures at the production rate.** The slices are stored at
  48 kHz and upsampled in the test, so the detector runs at 2.4 MS/s.

## Where this goes next

CTCSS is a tone; its digital cousin is a 23-bit Golay codeword at 134.4 baud,
and it had been invented twice — layout and bit order — by the encoder and
the decoder together.
[Part 6]({{ '/blog/tutorials/conventional-scanner-06-dcs/' | relative_url }})
reads `dcs.go`: the on-air order `C0..C8, 0 0 1, P0..P10` with bit 0 first
(023 = 0x763813), the parity equations pinned independently of the encoder,
SDRangel's alias tables (023 ≡ 340 ≡ 766, 023N ≡ 047I), the `dcs_polarity`
key, and the N/I mismatch WARN that the 24 Sep run pinned to
`D025N → false`.

## FAQ

**Why does my CTCSS gate open on the tone next to the one I configured?**
Before #1184 the Goertzel bin was rounded to a 5 Hz grid, so a "162.2 Hz"
detector was really 160.0 Hz, which a 159.8 Hz tone hits exactly.
`NewGoertzelExact` now measures at the configured frequency over a 250 ms
block (4 Hz resolution), and the adjacent EIA tones are measured as reverse
bins the target must beat by 1.5×.

**Why did CTCSS never open at all on my RTL-SDR?**
The magnitude threshold was calibrated in radians per sample at 48 kHz, but
the scanner fed the detector 2.4 MS/s IQ, where the same deviation reads 50×
smaller. `toneFrontEnd` now decimates to ~48 kHz, channel-filters at ±8 kHz
and normalises the discriminator to the calibration rate; the reporter's
real-air slices pin it at 2.4 MS/s.

**Does the CTCSS gate work with narrowband (12.5 kHz) radios?**
Yes. NFM radios put ~350 Hz of deviation into the tone; the old fixed floor
needed ~540 Hz. `ctcssMinDeviationHz` = 100 sets the floor from a deviation
with >10 dB margin, and `TestCTCSSOpensOnEveryEIAToneAtNarrowbandDeviation`
pins every EIA tone at 350 Hz. A tone-free carrier still stays shut.

**Can GopherTrunk tell 150.0 Hz from 151.4 Hz?**
No. They are 1.4 Hz apart and a 250 ms block cannot separate them (each
leaks ~66 % into the other), so neither is used as the other's reverse bin
and a gate on either opens on both — as on most radios. Every other
adjacent EIA pair (≥ 2.3 Hz) is rejected.

## Series navigation

**Part 5 of 14** · ←
[Part 4: Squelch on the Channel, Not the Span — The In-Channel Power Meter]({{ '/blog/tutorials/conventional-scanner-04-squelch-on-the-channel/' | relative_url }})
· Next →
[Part 6: DCS — The On-Air Bit Order, Polarity, and the Aliases You Cannot Fix]({{ '/blog/tutorials/conventional-scanner-06-dcs/' | relative_url }})
