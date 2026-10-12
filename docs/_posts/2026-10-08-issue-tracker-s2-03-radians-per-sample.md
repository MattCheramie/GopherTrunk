---
title: "From the Issue Tracker, Season 2, Part 3: Radians per Sample — A CTCSS Gate Calibrated at 48 kHz and Fed 2.4 MS/s"
description: "Why GopherTrunk's conventional-scanner CTCSS gate never opened on any real radio — a Goertzel magnitude threshold calibrated in radians per sample at 48 kHz, fed the SDR's 2.4 MS/s stream where the same deviation reads 50× smaller — plus the two defects the real-air captures exposed next: a Goertzel bin that rounded 162.2 Hz to 160 Hz, and a magnitude floor no narrowband radio could reach."
category: solution-postmortem
keywords: ctcss not detected sdr, ctcss goertzel sample rate, radians per sample threshold, ctcss 162.2 opens on 159.8, goertzel bin rounding, newgoertzelexact, narrowband fm ctcss deviation, eia tone table reverse bins, tone squelch rtl-sdr 2.4 msps, gophertrunk from the issue tracker season 2
tags: [from-the-issue-tracker-s2, ctcss, goertzel, sample-rate, scanner, postmortem]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "From the Issue Tracker, Season 2"
series_part: 3
---

*Part 3 of **From the Issue Tracker, Season 2**, postmortems of GopherTrunk
bugs told with receipts.
[Part 2]({{ '/blog/solution-postmortem/issue-tracker-s2-02-tone-at-four-delta/' | relative_url }})
moved the conventional scanner's LO off the channel so a clipped front end
stopped whistling into the audio. This part is about the detector that
stream feeds next — the CTCSS tone gate — which had never opened on a real
radio, and whose three defects came out one capture at a time: a number
with the wrong units, a bin that rounded to the wrong tone, and a floor set
for wideband radios in a narrowband world.*

> **TL;DR:** `CTCSSDetector` compared the Goertzel power of its FM
> discriminator output against a magnitude threshold calibrated at
> **48 kHz**, but the scanner fed it the SDR's **2.4 MS/s** IQ. A
> discriminator reads radians *per sample*, so the same tone deviation is
> 50× smaller per sample at 2.4 MS/s — ~3000× in power — and a real CTCSS
> tone sat ~3000× below the threshold: 0 % detection on every capture
> ([#1184](https://github.com/MattCheramie/GopherTrunk/issues/1184)).
> `toneFrontEnd` now decimates to `toneRefRateHz` (48 kHz), channel-filters
> ±8 kHz and normalises the discriminator by `discScale`, so a threshold
> means the same deviation at any SDR rate. The real-air pins then exposed
> two more: `toneout.NewGoertzel` **rounds to the nearest bin**, so at the
> 5 Hz resolution of a 200 ms block the "162.2 Hz" gate was really 160 Hz —
> shut on 162.2, open on 159.8, the reporter's exact report — fixed by
> `NewGoertzelExact`, 250 ms blocks and the adjacent EIA tones as reverse
> bins; and the fixed `5e-4` magnitude needed ~540 Hz of deviation when
> NFM radios send ~350, now derived from `ctcssMinDeviationHz` = 100.
> All three on-air verified 24 September on the reporter's Kenwoods.

**Key takeaways**

- **A threshold in radians per sample is a sample-rate trap.** The unit
  carries the rate inside it; a detector calibrated at one rate and run at
  another is wrong by the ratio, silently.
- **Unit tests at the calibration rate cannot see it.** Every CTCSS test
  ran at 48 kHz with a clean synthetic tone — the self-consistent trap in
  a new dress. The regression interpolates real air *back up* to 2.4 MS/s.
- **Rounding a bin is fine for paging tones and fatal for CTCSS.** EIA tones
  sit 2.3–3.0 Hz apart; a 5 Hz grid lands "162.2" on 160, where 159.8 hits
  dead-on.
- **Derive thresholds from physics, not from a number that worked once.**
  `ctcssToneMagnitude` converts a minimum deviation through the exact
  discriminator, scaling and low-pass chain the detector runs.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| Rate normalisation | decimate to ~48 kHz, ±8 kHz channel filter, `discScale` = rate/48000 | `internal/scanner/conventional/tone_frontend.go` (`toneFrontEnd`, `toneRefRateHz`) |
| Exact bin | DTFT at exactly `targetHz`, no rounding | `internal/voice/toneout/goertzel.go` (`NewGoertzelExact`) |
| Block length | 250 ms → 4 Hz resolution (was 200 ms / 5 Hz) | `ctcss.go` (`ctcssBlockSeconds`) |
| Reverse bins | ±5 Hz plus the adjacent EIA tones within 10 Hz; must beat them ×1.5 | `eiaNeighbourTones`, `ctcssReverseOffsetHz`, `rejectRatio` |
| Threshold | 100 Hz minimum deviation through discriminator + int16 scale + LPF | `ctcssMinDeviationHz`, `ctcssToneMagnitude` |
| Real-air pins | reporter's 447.100 MHz slices, interpolated ×50 back to 2.4 MS/s | `ctcss_realair_test.go` (`TestCTCSSDetectsRealAirToneAtSDRRate`) |
| Off-by-one pin | 162.2/192.8 configured, 159.8/189.9 sent, at 2.4 MS/s | `ctcss_eia_test.go` (`TestCTCSSReportedToneOffsetAtSDRRate`) |

## In this post

- **"Tones were not being detected"** — the report, and the unit nobody checked.
- **Fifty times smaller per sample** — the front end that normalises the rate.
- **The bin that rounded 162.2 to 160** — `NewGoertzel` versus `NewGoertzelExact`.
- **A floor set for wideband radios** — 5e-4 against 350 Hz of deviation.
- **Pins that cannot share the mistake** — real air, upsampled.
- **On air, 24 September** — the Kenwood lab run.

## "Tones were not being detected"

The conventional scanner's tone gate is what turns carrier squelch into
"right-system" squelch: with `tone: {mode: ctcss, ctcss_hz: 100}` on a scan
list entry, the dwell holds only while the 100 Hz sub-audible tone is
present
([Operator Cookbook Part 6]({{ '/blog/tutorials/operator-cookbook-06-analog-fm-tone-out/' | relative_url }})
has the config). The reporter's Radtel on 447.100 MHz sent 100 Hz CTCSS and
the gate never opened. Not on that radio, not on the Kenwoods, not on any
tone. The scanner fell through to carrier-only behaviour on every channel.

The suspects were the usual ones: deviation too low, the audio low-pass
eating the tone, the reverse-bin ratio rejecting it. None of those produce
*zero* on every radio at every level. The detector had a full unit suite —
`TestCTCSSDetector_MatchesConfiguredTone`, `_RejectsOffFrequencyTone`,
`_RejectsAdjacentTone` — all green, all at 48 kHz.

That rate was the whole bug. The scanner feeds the detector the SDR's
full-rate IQ: 2.4 MS/s on an RTL-SDR. `CTCSSDetector.Process` ran a
quadrature discriminator directly on it and compared the Goertzel power of
the result against `magThreshold`, a constant that had been tuned by hand
against 48 kHz synthetic tones.

## Fifty times smaller per sample

A quadrature discriminator outputs `arg(z[n]·conj(z[n−1]))` — the phase
advance *per sample*, in radians per sample. A tone with peak deviation
`dev` reads `2π·dev/rate` rad/sample. At 48 kHz a 350 Hz deviation is
0.046 rad/sample; at 2.4 MS/s the same tone, the same deviation, reads
0.00092 — **fifty times smaller**, because fifty times as many samples
share each radian of phase. The Goertzel reports squared magnitude, so the
power is down by the square of that — the measurement on the reporter's
captures put a real tone ~3000× below the threshold. It could never open.

`toneFrontEnd` (`tone_frontend.go`) is the fix, shared by the CTCSS and DCS
detectors:

```go
// internal/scanner/conventional/tone_frontend.go (shape)
const toneRefRateHz = 48_000       // the rate thresholds are calibrated at
const toneChannelCutoffHz = 8_000  // one-sided channel filter

func newToneFrontEnd(sampleHz float64) *toneFrontEnd {
    m := int(sampleHz / toneRefRateHz)            // 2.4 MS/s → 50
    if m < 2 { m = 1 } else { pre = dsp.NewResampler(1, m, m*8+1, 8.6) }
    rate := sampleHz / float64(m)
    if fc := toneChannelCutoffHz / rate; fc < 0.45 {
        chanFilter = filter.NewFIR(filter.LowpassKaiser(63, fc, 8.6))
    }
    return &toneFrontEnd{…, discScale: rate / toneRefRateHz, rate: rate, m: m}
}
```

Three stages. An integer decimator behind an anti-alias polyphase filter
takes the SDR rate down to ~48 kHz (only when the input is ≥ 96 kHz, so the
48 kHz behaviour and its unit tests are unchanged). A 63-tap ±8 kHz Kaiser
low-pass keeps the FM channel and rejects the 12.5/25 kHz neighbours and
the rest of the band — whose FM noise, run through the discriminator
unfiltered, had been swamping a sub-audible tone. And the discriminator
output is multiplied by `discScale` = rate/48000, so the result is in
radians per sample **at 48 kHz** whatever the input rate. A threshold now
means one deviation everywhere.

<figure class="lab-figure">
<svg viewBox="0 0 680 220" width="680" height="220" role="img" aria-label="Left panel: a vertical axis of discriminator output in radians per sample with a threshold line; a 350 hertz tone at 48 kilohertz sits above the line while the same tone at 2.4 megasamples per second sits fifty times lower, far below it. Right panel: a frequency axis from 155 to 165 hertz with the rounded Goertzel bins at 155, 160 and 165 hertz marked; the EIA tones 159.8 and 162.2 are plotted, 159.8 landing almost on the 160 bin and 162.2 missing it by 2.2 hertz, while the exact bin sits on 162.2 itself.">
  <text x="160" y="16" text-anchor="middle" fill="currentColor" font-size="10" font-weight="bold">same tone, two sample rates</text>
  <line x1="60" y1="40" x2="60" y2="190" stroke="var(--fg-muted)"/>
  <text x="56" y="44" text-anchor="end" fill="var(--fg-muted)" font-size="8">rad/sample</text>
  <line x1="60" y1="92" x2="300" y2="92" stroke="var(--accent)" stroke-dasharray="3 3"/>
  <text x="304" y="95" fill="var(--accent)" font-size="8">threshold (48 kHz)</text>
  <rect x="100" y="60" width="50" height="130" fill="none" stroke="currentColor" stroke-width="1.5"/>
  <text x="125" y="54" text-anchor="middle" fill="currentColor" font-size="8">48 kHz</text>
  <text x="125" y="205" text-anchor="middle" fill="currentColor" font-size="8">0.046</text>
  <rect x="210" y="187" width="50" height="3" fill="none" stroke="currentColor" stroke-width="1.5"/>
  <text x="235" y="180" text-anchor="middle" fill="currentColor" font-size="8">2.4 MS/s</text>
  <text x="235" y="205" text-anchor="middle" fill="currentColor" font-size="8">0.00092 (÷50)</text>
  <text x="420" y="16" text-anchor="middle" fill="currentColor" font-size="10" font-weight="bold">NewGoertzel's 5 Hz grid</text>
  <line x1="340" y1="140" x2="660" y2="140" stroke="var(--fg-muted)"/>
  <g font-size="8" fill="var(--fg-muted)" text-anchor="middle">
    <line x1="360" y1="134" x2="360" y2="146" stroke="var(--fg-muted)"/><text x="360" y="160">155</text>
    <line x1="500" y1="134" x2="500" y2="146" stroke="var(--fg-muted)"/><text x="500" y="160">160 ← "162.2" bin</text>
    <line x1="640" y1="134" x2="640" y2="146" stroke="var(--fg-muted)"/><text x="640" y="160">165</text>
  </g>
  <circle cx="494" cy="110" r="4" fill="currentColor"/>
  <text x="494" y="100" text-anchor="middle" fill="currentColor" font-size="8">159.8 sent → opens</text>
  <circle cx="562" cy="110" r="4" fill="var(--accent)"/>
  <text x="562" y="100" text-anchor="middle" fill="var(--accent)" font-size="8">162.2 sent → shut</text>
  <line x1="562" y1="134" x2="562" y2="146" stroke="var(--accent)"/>
  <text x="562" y="178" text-anchor="middle" fill="var(--accent)" font-size="8">NewGoertzelExact: bin at 162.2</text>
  <text x="500" y="205" text-anchor="middle" fill="var(--fg-muted)" font-size="8">EIA neighbours 2.3–3.0 Hz apart; block 200 → 250 ms (5 → 4 Hz)</text>
</svg>
<figcaption>Left: a discriminator measures phase per sample, so a threshold set at 48 kHz is fifty times too high for the same tone at 2.4 MS/s. Right: rounding the Goertzel bin to a 5 Hz grid put the "162.2 Hz" detector on 160 Hz, where the next tone down lands dead-on.</figcaption>
</figure>

## The bin that rounded 162.2 to 160

With the rate fixed, the reporter's next run found the gate opening — on
the wrong tone. A channel configured for 162.2 Hz stayed shut when the
radio sent 162.2 and opened when it sent 159.8; 192.8 behaved the same way
against 189.9. One tone *down* the EIA table, in both cases.

The detector used `toneout.NewGoertzel`, the same single-bin power detector
the two-tone paging decoder uses
([Beyond Voice Part 5]({{ '/blog/deep-dives/beyond-voice-05-two-tone-tone-out/' | relative_url }})).
For numerical tidiness it rounds the target to the nearest exact DFT bin:

```go
// internal/voice/toneout/goertzel.go
k := math.Round(float64(blockSize) * targetHz / sampleHz)
omega := 2 * math.Pi * k / float64(blockSize)
```

With a 200 ms block at 48 kHz the bin spacing is 5 Hz, so k for 162.2 Hz is
round(32.44) = 32 — a bin centred on **160.0 Hz**. 159.8 Hz hits that bin
within 0.2 Hz; 162.2 Hz misses it by 2.2 Hz and leaks most of its power
into the reverse bins, which the ×1.5 rejection ratio then uses to refuse
it. Paging tones are hundreds of hertz apart and never notice. EIA CTCSS
tones sit 2.3–3.0 Hz apart, and the rounding was larger than the spacing.

`NewGoertzelExact` is `NewGoertzel` without the rounding: `omega =
2π·targetHz/sampleHz`. The magnitude formula `Process` evaluates is the
DTFT at that frequency and is exact for any ω, so nothing is lost.
`NewCTCSSDetector` now builds every bin with it. Two more changes came with
it. The block grew from 200 ms to `ctcssBlockSeconds` = 250 ms (4 Hz
resolution): at 5 Hz an on-tone signal leaked ~57 % of its power into a
neighbour 2.3 Hz away, leaving the rejection ratio almost no margin; at
4 Hz the leak is ~29 %, and a detection still lands inside the ~300 ms
decode time radios quote. And the reverse bins now include the **adjacent
EIA tones** themselves — `eiaNeighbourTones` returns the table entries
immediately below and above the target, within 10 Hz — alongside the two
fixed ±`ctcssReverseOffsetHz` (5 Hz) bins, so a transmission on the next
tone up or down lands dead-on its own reverse bin and can never open the
gate. One pair is deliberately excluded: 150.0 and 151.4 Hz are 1.4 Hz
apart (`ctcssMinNeighbourSpacingHz` = 2), a 250 ms block cannot separate
them (each leaks ~66 % into the other), so a gate on either opens on both,
as on most radios.

## A floor set for wideband radios

The third defect was the magnitude threshold itself: a fixed `5e-4`. Run
through the detector's own scaling that corresponds to ~540 Hz of tone
deviation. Radios put roughly 15 % of peak deviation into CTCSS — about
750 Hz on a 5 kHz wideband channel, about 350 Hz on a 2.5 kHz narrowband
one — so a wideband radio cleared the floor and a narrowband one never
could. "With narrow FM TX the squelch does not open for any configured
tone" was the report.

The threshold is now derived, not tuned:

```go
// internal/scanner/conventional/ctcss.go
const ctcssMinDeviationHz = 100

func ctcssToneMagnitude(devHz, toneHz, cutoffHz float64) float64 {
    a := 2 * devHz / toneRefRateHz               // 2π·dev/rate, then ÷π for the int16 scale
    g := 1 / (1 + (toneHz/cutoffHz)*(toneHz/cutoffHz)) // single-pole LPF, power
    return a * a * g                              // Goertzel reports squared amplitude
}
```

It follows the chain exactly: the discriminator reads `2π·dev/rate`
rad/sample at 48 kHz; `Process` scales by `32768/π` into the int16 range the
Goertzel takes, which divides out the π; the single-pole audio low-pass at
500 Hz attenuates a tone at `toneHz` by `1/(1+(f/fc)²)` in power; and the
Goertzel reports squared amplitude. A 100 Hz minimum keeps more than 10 dB
of margin under a narrowband tone, and a carrier *without* a tone still
stays shut, because it puts nothing coherent in the target bin and the
reverse-bin ratio holds.

## Pins that cannot share the mistake

The rate bug is a textbook case of the
[self-consistent trap]({{ '/blog/solution-postmortem/from-the-issue-tracker-20-self-consistent-trap/' | relative_url }}):
every test generated a clean tone at 48 kHz and fed it to a detector
calibrated at 48 kHz. The regression had to run at the production rate on
real modulation. `ctcss_realair_test.go` reads two 48 kHz cs16 slices of the
reporter's 447.100 MHz Radtel captures — `ctcss100_radtel_447100k_48k.cs16`
(voice plus 100 Hz CTCSS plus receiver noise) and
`ctcss_none_radtel_447100k_48k.cs16` (the same radio keyed with no tone) —
and **interpolates them back up by 50** (`dsp.NewResampler(50, 1, 16, 8.6)`)
to 2.4 MS/s, the rate the scanner actually feeds the detector:

- `TestCTCSSDetectsRealAirToneAtSDRRate`: the 100 Hz gate present in ≥ 95 %
  of RTL-sized chunks on the tone transmission; 94.8, 103.5 and 107.2 Hz
  never open on it. Old detector: 0 %.
- `TestCTCSSRejectsRealAirToneFreeCarrierAtSDRRate`: the tone-free
  transmission never opens a 100 Hz gate.
- `TestCTCSSRealAirOnlyTheSentToneOpens`: the whole 51-tone EIA table over
  both captures — only 100 Hz may open on the tone file, nothing on the
  tone-free one. This is the no-harm pin for the lower threshold.

`ctcss_eia_test.go` covers the other two defects synthetically but at the
production rate where it matters. `TestCTCSSReportedToneOffsetAtSDRRate`
replays the report at 2.4 MS/s: 162.2 and 192.8 configured must open on
their own tone at 350 and 750 Hz of deviation and must stay shut on 159.8
and 189.9. `TestCTCSSOpensOnEveryEIAToneAtNarrowbandDeviation` sweeps the
table at 350 Hz deviation and 20 dB SNR, requiring ≥ 90 % presence;
`TestCTCSSRejectsAdjacentEIATones` sends each tone's neighbours at full
750 Hz wideband deviation and requires 0 %. Both skip the 150.0/151.4
pair.

## On air, 24 September

The rate fix landed on 24 September; the bin and threshold fixes the same
day, after the first real-air pass showed the off-by-one. The reporter ran
the build on their Kenwood NX-300 and NX-5000 against a service monitor,
and all three CTCSS fixes are **on-air verified** — the same run that
pinned the DCS polarity convention the next part is about. The scanner's
minimum dwell on a tone-gated channel is now 350 ms, covering the 250 ms
Goertzel block plus chunk granularity, so a tone-gated channel cannot be
left before the detector has reported once.

The DCS detector shares `toneFrontEnd` and so inherited the rate fix, but
it slices by sign, which is rate-invariant — a different detector with a
different failure, and the subject of Part 4.

## Where this goes next

CTCSS failed by a factor of fifty; DCS failed by construction. Its 23-bit
codeword layout and bit order were invented, the test synthesiser transmitted
the same invention, and no real radio could ever match.
[Part 4]({{ '/blog/solution-postmortem/issue-tracker-s2-04-invented-dcs-codeword/' | relative_url }})
rebuilds the word from the published parity equations, pins all 512 codes
against them, and reads the N/I polarity convention off the reporter's
Kenwoods.

## FAQ

**Why did the CTCSS squelch never open on any tone?**
The detector's magnitude threshold was calibrated in radians per sample at
48 kHz, but the scanner fed it 2.4 MS/s IQ, where the same deviation reads
50× smaller per sample and ~3000× smaller in power. `toneFrontEnd` now
decimates to `toneRefRateHz` (48 kHz), channel-filters ±8 kHz and scales the
discriminator by `discScale`, so the threshold means one deviation at any
rate.

**Why did a 162.2 Hz gate open on 159.8 Hz?**
`toneout.NewGoertzel` rounds to the nearest DFT bin. At the old 200 ms block
the bins are 5 Hz apart, so "162.2" became bin 32 — 160.0 Hz — which
159.8 Hz hits within 0.2 Hz and 162.2 Hz misses by 2.2 Hz. CTCSS now uses
`NewGoertzelExact` (no rounding), 250 ms blocks and the adjacent EIA tones
as reverse bins.

**Why did narrowband radios fail where wideband ones worked?**
The fixed `5e-4` threshold corresponded to ~540 Hz of tone deviation. NFM
radios put ~350 Hz into CTCSS (15 % of 2.5 kHz), wideband ones ~750 Hz.
`ctcssToneMagnitude` now derives the threshold from `ctcssMinDeviationHz`
(100 Hz) through the discriminator, int16 scaling and low-pass, leaving
> 10 dB of margin under an NFM tone.

**Can GopherTrunk tell 150.0 Hz from 151.4 Hz CTCSS?**
No. They are 1.4 Hz apart, under `ctcssMinNeighbourSpacingHz` (2 Hz); a
250 ms block leaks ~66 % of one into the other, so a gate on either opens
on both. Most radios behave the same way. Every other adjacent EIA pair
(2.3 Hz apart or more) is separated.

**Is the CTCSS fix verified on air?**
Yes. The reporter's 24 September run on Kenwood NX-300/NX-5000 radios with a
service monitor confirmed all three fixes. The committed regressions are the
reporter's own 447.100 MHz slices, interpolated to 2.4 MS/s
(`TestCTCSSDetectsRealAirToneAtSDRRate`, `TestCTCSSRealAirOnlyTheSentToneOpens`).

## Series navigation

**Part 3 of 14** · ←
[Part 2: A Tone at Four Times the Offset — Zero-IF Clipping on the Analog Scanner]({{ '/blog/solution-postmortem/issue-tracker-s2-02-tone-at-four-delta/' | relative_url }})
· Next →
[Part 4: The Invented DCS Codeword — Pinned by Parity Equations, Not the Encoder]({{ '/blog/solution-postmortem/issue-tracker-s2-04-invented-dcs-codeword/' | relative_url }})
