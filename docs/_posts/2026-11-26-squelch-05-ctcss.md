---
title: "Squelch, Part 5: CTCSS — Separating Tones 2.3 Hz Apart"
description: "How GopherTrunk's CTCSS gate tells a sub-audible tone from its neighbour 2.3 Hz away: a shared front end that keeps the discriminator in 48 kHz units at any SDR rate, an exact-frequency Goertzel over 250 ms blocks, the adjacent EIA tones as reverse bins and a threshold in hertz of deviation. On-air verified 24 Sep."
category: deep-dives
keywords: ctcss decoder sdr, ctcss tone detection goertzel, sub-audible tone squelch, pl tone scanner software, goertzel frequency resolution, eia ctcss tone table, tone squelch fm discriminator, ctcss deviation threshold, conventional scanner tone gate, gophertrunk ctcss
tags: [squelch, ctcss, goertzel, analog-fm, scanning, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "Squelch"
series_part: 5
---

*Part 5 of **Squelch**, a 14-part deep dive into GopherTrunk's conventional
scanner — the scan list with no control channel to announce a call — and
every gate that decides when a channel opens.
[Part 4]({{ '/blog/deep-dives/squelch-04-offset-tuning/' | relative_url }})
moved the tuner off the channel so a clipped ADC's products land outside it.
Every gate so far has asked whether *something* is there. This part is the
first that asks whether it is *yours*: the CTCSS tone gate as it stands in
the code today. How it got here — two rounds of
[#1184](https://github.com/MattCheramie/GopherTrunk/issues/1184) — is
[Rebuilt From the Air Part 3]({{ '/blog/solution-postmortem/rebuilt-from-the-air-03-ctcss-dcs-tone-gates/' | relative_url }})'s
story; this post is the design.*

> **TL;DR:** A CTCSS gate is `CTCSSDetector` in
> `internal/scanner/conventional/ctcss.go`. Its front end (`toneFrontEnd`)
> decimates the SDR's IQ to ~48 kHz, channel-filters ±8 kHz, discriminates
> and scales the output to radians per sample at `toneRefRateHz` = 48 000,
> so a threshold means the same deviation at any sample rate. A 500 Hz
> one-pole low-pass feeds `toneout.NewGoertzelExact`, a resonator at exactly
> the tone frequency, over `ctcssBlockSeconds` = 250 ms (4 Hz resolution).
> The tone must reach the magnitude of a 100 Hz-deviation tone
> (`ctcssMinDeviationHz`) **and** beat 1.5× the largest reverse bin — the
> target ±5 Hz plus the adjacent EIA tones. 150.0 / 151.4 Hz, 1.4 Hz apart,
> open together by design. The scan window grows to 350 ms. On-air verified
> 24 Sep.

**Key takeaways**

- **A tone gate answers "whose?", not "whether".** The carrier tests of
  Parts 2 and 3 still have to pass; the tone decides which user of a shared
  frequency opens the channel.
- **Resolution is bought with time, and 250 ms is the price.** A block of T
  seconds resolves 1/T hertz, and the EIA table's tight pairs sit 2.3–3.0 Hz
  apart.
- **A tone has to beat its neighbours, not only a floor.** A transmission on
  the next tone in the table lands dead-on its own reverse bin and can never
  open the gate.
- **The threshold is hertz of deviation, not level.** A discriminator is
  blind to IQ amplitude; the 100 Hz floor sits more than 10 dB under a
  narrowband radio's ~350 Hz tone at any gain.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| Front end | decimate to ~48 kHz, ±8 kHz channel filter, discriminate, scale to rad/sample at 48 kHz | `tone_frontend.go` (`toneFrontEnd`, `toneRefRateHz`) |
| Target bin | Goertzel at exactly the tone frequency, 250 ms blocks, after a 500 Hz low-pass | `toneout.NewGoertzelExact`, `ctcssBlockSeconds` |
| Reverse bins | target ±5 Hz and the EIA tone either side; target must exceed 1.5× the largest | `ctcssReverseOffsetHz`, `eiaNeighbourTones`, `rejectRatio` |
| Threshold | reading of a tone with 100 Hz of peak deviation | `ctcssMinDeviationHz`, `ctcssToneMagnitude` |
| Inseparable pair | 150.0 / 151.4 Hz open together; config refuses them on one shared frequency | `tonecode.MinSeparableCTCSSHz`, `tonecode.Overlap` |
| Scan cost | every scan window on that scanner grows to 350 ms | `ctcssMinDwell`, `minToneDwell` (`scanner.go`) |
| Pins | the EIA table, the reported offsets at 2.4 MS/s, real-air slices | `ctcss_eia_test.go`, `ctcss_realair_test.go` |

## In this post

- **A tone under the voice** — what the gate must decide.
- **The shared front end** — one discriminator scale at any SDR rate.
- **One bin, exactly on frequency** — `NewGoertzelExact` and what 250 ms buys.
- **Neighbours as reverse bins** — the ratio test, the inseparable pair, the threshold.
- **Carrier AND tone** — how the scanner uses the answer, and its cost.
- **What pins it** — the table, the production rate, real air.

## A tone under the voice

[CTCSS]({{ '/reference/ctcss/' | relative_url }}) — the Continuous
Tone-Coded Squelch System — is a sine wave between 67.0 and 254.1 Hz that a
transmitter mixes into its audio for as long as it is keyed. The receiving
radio filters it out of the speaker and uses it for one decision: stay muted
unless my tone is present. That is how two agencies share a frequency, and a
scan list needs the same decision — without it, says the comment at the top
of `ctcss.go`, the scanner "stops on every nearby transmission on the same
frequency".
[Operator's Cookbook Part 6]({{ '/blog/tutorials/operator-cookbook-06-analog-fm-tone-out/' | relative_url }})
showed the operator's side, a `tone:` block with `mode: ctcss` and
`ctcss_hz`.

On an FM carrier the tone is a slow, small wobble of the carrier frequency.
The code's figure is that radios put about 15 % of their peak
[deviation]({{ '/reference/fm-deviation/' | relative_url }}) into it:
roughly 350 Hz on a 2.5 kHz narrowband channel, 750 Hz on a 5 kHz one.

The gate has three things to settle: is there a steady tone at my frequency,
is it *my* tone and not the one beside it in the table, and is it strong
enough to be a radio's. The second is the hard one. `EIACTCSSTones` lists 51
tones, unevenly spaced: gaps run from 8.5 Hz at the top of the table down to
2.3 Hz (67.0 / 69.3 Hz), with one pair — 150.0 and 151.4 Hz — only 1.4 Hz
apart. A gate set to 162.2 Hz that also opens on 159.8 Hz is not a tone
gate.
[Protocol Decoders Part 9]({{ '/blog/deep-dives/protocol-decoders-09-conventional-wideband/' | relative_url }})
covered this stage in one phrase, "a Goertzel bin". The bin has since
changed in every respect that matters.

## The shared front end: one discriminator scale at any rate

The scanner hands a tone detector the SDR's full-rate IQ — 2.4 MS/s on an
RTL-SDR — and `toneFrontEnd`, shared by the CTCSS and DCS detectors,
discards nearly all of it:

```go
// internal/scanner/conventional/tone_frontend.go (shape)
const toneRefRateHz = 48_000
const toneChannelCutoffHz = 8_000

m := int(sampleHz / toneRefRateHz)        // 50 at 2.4 MS/s
pre := dsp.NewResampler(1, m, m*8+1, 8.6) // anti-aliased decimation
rate := sampleHz / float64(m)
chanFilter := filter.NewFIR(filter.LowpassKaiser(63, toneChannelCutoffHz/rate, 8.6))
discScale := rate / toneRefRateHz
// per sample: arg(z[n] · conj(z[n-1])) · discScale
```

**Decimate, then channel-filter.** The ±8 kHz filter passes one FM channel
and rejects its neighbours along with, in the constant's comment, "the rest
of the SDR's band, whose FM noise would otherwise swamp a sub-audible
signalling tone". It is the recipe
[Part 2]({{ '/blog/deep-dives/squelch-02-channel-power/' | relative_url }})'s
channel power meter uses, for the same reason: a statistic about a channel
has to be computed on the channel.

**Discriminate, then normalise.** The
[discriminator]({{ '/reference/quadrature-demodulation/' | relative_url }})
output is the phase step between consecutive samples — radians *per
sample*. That unit carries the sample rate inside it: 350 Hz of deviation is
a 0.046 rad step at 48 kHz and one fifty times smaller at 2.4 MS/s.
`discScale` multiplies it back, so the output is always "radians per sample
at 48 kHz". The type's comment states the contract: "a threshold means the
same deviation at any SDR sample rate". The detector before this one had a
threshold calibrated at 48 kHz and was fed 2.4 MS/s, and a real tone sat
about 3000× below it. That story belongs to the postmortem; the design
consequence is that nothing after `toneFrontEnd` knows the SDR's rate.

A one-pole low-pass at 500 Hz (`AudioCutoffHz`) then rolls the voice off.

## One bin, exactly on frequency

The measurement is the
[Goertzel algorithm]({{ '/reference/goertzel-algorithm/' | relative_url }}):
a two-term recurrence that evaluates one frequency of a block's spectrum
without computing the rest.
[Beyond Voice Part 5]({{ '/blog/deep-dives/beyond-voice-05-two-tone-tone-out/' | relative_url }})
took the resonator apart for two-tone paging, where `toneout.NewGoertzel`
rounds the target onto the nearest exact bin of the block. That suits paging
tones tens of hertz apart. For CTCSS it is fatal, and the second
constructor's comment says why: at a 5 Hz bin spacing the "162.2 Hz" bin is
really 160 Hz, "which a 159.8 Hz tone hits dead-on and a real 162.2 Hz tone
misses by 2.2 Hz".

`NewGoertzelExact` differs in one line: its `omega` is
`2π·targetHz/sampleHz`, the tone itself, where `NewGoertzel` uses a rounded
bin index. The magnitude formula in `Process` is exact at any frequency, so
nothing is lost by leaving the grid. `TestGoertzelExactHitsOffGridTone` pins
the pair on a 162.2 Hz sine.

Putting the bin on the tone fixes *where* it looks. How *sharply* is set by
the block length: a block of T seconds has a main lobe 1/T hertz wide to its
first null, and `ctcssBlockSeconds` is 0.25 — 12 000 samples at 48 kHz, 4 Hz
to the null. The constant's comment records the trade: a tone 2.3 Hz away
still leaks about 29 % of its power into the bin, "and a detection still
lands well inside the ~300 ms decode time radios quote". Blocks are aligned,
not sliding, so the gate decides once every quarter second.

<figure class="lab-figure">
<svg viewBox="0 0 680 236" width="680" height="236" role="img" aria-label="One 250 millisecond Goertzel block's response against a tone's offset from the bin: 100 percent on the bin, about 66 percent at 1.4 hertz, about 29 percent at 2.3 hertz, nulls at plus and minus 4 hertz, reverse bins at plus and minus 5 hertz, and a dashed limit at one over 1.5 of the target.">
  <text x="340" y="16" text-anchor="middle" fill="currentColor" font-size="10" font-weight="bold">one 250 ms Goertzel block: power a tone puts in a bin it is offset from</text>
  <line x1="52" y1="196" x2="628" y2="196" stroke="var(--fg-muted)"/>
  <path d="M52 196.0 L61 195.9 L70 195.5 L79 194.8 L88 193.8 L97 192.8 L106 191.7 L115 190.8 L124 190.1 L133 189.9 L142 190.1 L151 190.7 L160 191.8 L169 193.1 L178 194.5 L187 195.6 L196 196.0 L205 195.4 L214 193.5 L223 189.8 L232 184.3 L241 176.7 L250 167.2 L259 156.0 L268 143.3 L277 129.8 L286 116.1 L295 102.8 L304 90.6 L313 80.4 L322 72.5 L331 67.7 L340 66.0 L349 67.7 L358 72.5 L367 80.4 L376 90.6 L385 102.8 L394 116.1 L403 129.8 L412 143.3 L421 156.0 L430 167.2 L439 176.7 L448 184.3 L457 189.8 L466 193.5 L475 195.4 L484 196.0 L493 195.6 L502 194.5 L511 193.1 L520 191.8 L529 190.7 L538 190.1 L547 189.9 L556 190.1 L565 190.8 L574 191.7 L583 192.8 L592 193.8 L601 194.8 L610 195.5 L619 195.9 L628 196.0" fill="none" stroke="currentColor" stroke-width="1.5"/>
  <line x1="52" y1="109" x2="628" y2="109" stroke="var(--accent)" stroke-dasharray="3 3"/>
  <text x="56" y="104" fill="var(--accent)" font-size="8">1 / 1.5 of the target: a reverse bin must read below this</text>
  <text x="340" y="58" text-anchor="middle" fill="currentColor" font-size="9">on the bin: 100 %</text>
  <text x="56" y="40" fill="var(--fg-muted)" font-size="8">reverse bins: the target ±5 Hz</text>
  <text x="56" y="52" fill="var(--fg-muted)" font-size="8">and the EIA tone on each side</text>
  <text x="624" y="40" text-anchor="end" fill="var(--fg-muted)" font-size="8">first nulls at ±4 Hz</text>
  <text x="624" y="52" text-anchor="end" fill="var(--fg-muted)" font-size="8">(1 / 0.25 s)</text>
  <line x1="257" y1="158" x2="257" y2="196" stroke="currentColor" stroke-dasharray="2 2"/>
  <text x="250" y="136" text-anchor="end" fill="currentColor" font-size="8">tightest separable EIA pair</text>
  <text x="250" y="147" text-anchor="end" fill="currentColor" font-size="8">2.3 Hz off: ~29 %</text>
  <line x1="390" y1="111" x2="390" y2="196" stroke="var(--accent)" stroke-dasharray="2 2"/>
  <text x="400" y="86" fill="var(--accent)" font-size="8">150.0 / 151.4 Hz</text>
  <text x="400" y="97" fill="var(--accent)" font-size="8">1.4 Hz off: ~66 %</text>
  <line x1="160" y1="182" x2="160" y2="200" stroke="var(--accent)"/>
  <line x1="520" y1="182" x2="520" y2="200" stroke="var(--accent)"/>
  <text x="52" y="210" text-anchor="middle" fill="var(--fg-muted)" font-size="8">−8</text>
  <text x="160" y="210" text-anchor="middle" fill="var(--accent)" font-size="8">−5</text>
  <text x="196" y="210" text-anchor="middle" fill="var(--fg-muted)" font-size="8">−4</text>
  <text x="257" y="210" text-anchor="middle" fill="var(--fg-muted)" font-size="8">−2.3</text>
  <text x="340" y="210" text-anchor="middle" fill="var(--fg-muted)" font-size="8">0</text>
  <text x="390" y="210" text-anchor="middle" fill="var(--fg-muted)" font-size="8">+1.4</text>
  <text x="484" y="210" text-anchor="middle" fill="var(--fg-muted)" font-size="8">+4</text>
  <text x="520" y="210" text-anchor="middle" fill="var(--accent)" font-size="8">+5</text>
  <text x="628" y="210" text-anchor="middle" fill="var(--fg-muted)" font-size="8">+8</text>
  <text x="340" y="228" text-anchor="middle" fill="var(--fg-muted)" font-size="8">offset of the tone from the bin (Hz)</text>
</svg>
<figcaption>A 250 ms block resolves 4 Hz: a tone 2.3 Hz off leaves about 29 % in the bin next door and loses the 1.5× comparison, while one 1.4 Hz off leaves about 66 % and cannot be told apart.</figcaption>
</figure>

## Neighbours as reverse bins

A 29 % leak means a wideband radio on the *wrong* tone still puts plenty of
power in the target bin, so a magnitude threshold alone cannot reject it.
The gate measures the competition as well. Every detector carries **reverse
bins**, each its own exact Goertzel over the same block: two at the target
±5 Hz (`ctcssReverseOffsetHz`) and the EIA tones immediately below and above
(`eiaNeighbourTones`). A 162.2 Hz gate also listens at 157.2, 167.2, 159.8
and 165.5 Hz, and decides once per block:

```go
// internal/scanner/conventional/ctcss.go (shape)
if targetMag < d.magThreshold {
	d.present = false // weaker than a 100 Hz-deviation tone
} else if d.rejectRatio > 0 && targetMag < d.rejectRatio*maxReverseMag {
	d.present = false // something beside the target is nearly as strong
} else {
	d.present = true
}
```

`rejectRatio` is 1.5, and the figure shows why that works with a 250 ms
block. A transmission on the neighbouring tone lands dead-on its own reverse
bin and leaks about 29 % into the target: the target reads under a third of
the reverse bin, nowhere near 1.5× it. The right tone does the opposite —
full strength in the target, about 29 % next door, a ratio around 3.4 — and
passes with room to spare.

The same arithmetic explains the one pair the gate gives up on. 150.0 and
151.4 Hz leak about 66 % into each other, which puts the *correct* tone's
ratio at about 1.5: on the limit, with no margin for noise. So
`eiaNeighbourTones` skips any neighbour closer than
`tonecode.MinSeparableCTCSSHz` (2.0 Hz), only that pair qualifies, and a
gate on either tone opens on both — "as on most radios", the comment adds.
The limit is stated once and used twice: a wideband dongle's `conventional:`
list lets several logical channels share one frequency when their tone gates
tell the transmissions apart, and config validation (`tonecode.Overlap`)
refuses a pair "under 2.0 Hz apart".

The other test, `magThreshold`, is not a typed-in number.
`ctcssToneMagnitude` computes what a steady tone of a given deviation reads
at the detector's output, and the floor is that value for
`ctcssMinDeviationHz` = 100 Hz. The constant's comment argues it in radio
terms: the fixed threshold it replaced needed about 540 Hz, "so no
narrowband radio ever opened the gate"; 100 Hz "keeps a >10 dB margin under
an NFM tone"; and a carrier with no tone stays shut "because it puts nothing
coherent in the target bin and the reverse-bin ratio holds".

Here the gate meets the rule this series leans on, from
[The Analog Edge Part 13]({{ '/blog/tutorials/analog-edge-13-coherence-not-dbfs/' | relative_url }}):
never gate a decision on absolute dBFS when a scale-invariant statistic
exists. A phase step does not grow when the tuner's gain does. One test is a
number of hertz, the other a ratio; neither is a level.

## Carrier AND tone: how the scanner uses the answer

The scanner knows a tone gate only as a `toneDetector` — `Process`,
`Present`, `Reset`, the interface the DCS, NAC and digital call gates also
satisfy — and uses it in two places in `scanner.go`.

In the **scan window**, the detector is fed only the chunks whose in-channel
power clears `squelch_dbfs`, and the channel opens when the detector reports
the tone *and* the carrier squelch is open on that chunk — which on an FM
channel includes the noise squelch of
[Part 3]({{ '/blog/deep-dives/squelch-03-noise-squelch/' | relative_url }}).
This is what an operator means by "CSQ AND tone". When the level test
measured the whole SDR span (Part 2) it was always true and the tone
detector was the only gate left — "Tone only, not CSQ AND Tone", as that
issue put it.

In the **dwell**, the tone has to stay: a chunk counts as active only while
the carrier holds *and* `det.Process` still reports the tone, so a
transmitter that drops its tone starts the hangtime countdown exactly as a
dropped carrier does. The detector is `Reset` on every retune.

The gate has costs:

- **Every scan window gets longer.** A decision needs one full block, so
  `New` raises the scanner's `MinDwellPerChannel` from its 100 ms default to
  `ctcssMinDwell` (350 ms) as soon as *any* channel is CTCSS-gated. A scan
  window is a timer, so ten channels go from one second of windows per lap
  to three and a half.
- **The call starts late.** A call is declared only once the gate confirms,
  "up to about half a second after the carrier came up". A retuning
  scan-list SDR starts recording there; the continuous wideband tap
  implements `OnsetIQSource` and hands over the IQ from the transmission's
  first samples.
- **Data decoders ignore it**, by design
  ([Part 11]({{ '/blog/deep-dives/squelch-11-data-on-the-scan-list/' | relative_url }})).
- **It fails open, loudly.** If a detector cannot be built the scanner logs
  `conv: CTCSS detector failed to initialise; tone gate disabled — every
  signal passes the gate` and carries on with the carrier squelch alone.

## What pins it

**The table.** `ctcss_eia_test.go` sweeps `EIACTCSSTones` on FM with added
noise and a random tone phase.
`TestCTCSSOpensOnEveryEIAToneAtNarrowbandDeviation` demands every tone open
at 350 Hz of deviation in at least 90 % of chunks;
`TestCTCSSRejectsAdjacentEIATones` sends the next tone up or down at 750 Hz
and demands the gate never open. Both skip the 150.0 / 151.4 pair, whose
inseparability is pinned where it has consequences — `TestOverlap` in
`tonecode`.

**The production rate.** `TestCTCSSReportedToneOffsetAtSDRRate` runs at
2 400 000 samples per second: 162.2 and 192.8 Hz gates must open on their
own tone and stay shut on 159.8 and 189.9 Hz.

**Real air.** `ctcss_realair_test.go` loads two 48 kHz slices of the
reporter's captures — a handheld keyed on 447.100 MHz with and without a
100.0 Hz tone — and interpolates them fifty-fold back to 2.4 MS/s, so the
detector runs at the scanner's rate "with real transmitter modulation
(voice + CTCSS + receiver noise), not a clean synthetic tone".
`TestCTCSSDetectsRealAirToneAtSDRRate` wants the 100 Hz gate present in at
least 95 % of chunks and the 94.8, 103.5 and 107.2 Hz gates never;
`TestCTCSSRealAirOnlyTheSentToneOpens` sweeps all 51 tones over both slices.

Status: **on-air verified** on 24 Sep, when the reporter ran the fixed build
against Kenwood NX-300 and NX-5000 radios and a service monitor. The
committed real-air fixtures carry one tone from one radio; the rest of the
table is pinned synthetically.

### How CTCSS shaped the Go code

- **One front end, several tenants.** `toneFrontEnd` serves CTCSS and DCS,
  and its complex stage (`channel`) feeds the P25 NAC gate.
- **Thresholds derive from a unit a radio person can check.** The source
  constant is `ctcssMinDeviationHz = 100`; the magnitude is computed.
- **Facts DSP and config both need live in a package with no imports.**
  `tonecode` owns the separability limit, so validator and detector cannot
  disagree.

## Where this goes next

CTCSS has a sibling that sends bits instead of a sine: a 23-bit word looped
at 134.4 bit/s under the voice.
[Part 6]({{ '/blog/deep-dives/squelch-06-dcs/' | relative_url }}) takes the
DCS gate apart. The operator's recipe for both gates is
[Cookbook Vol. 2 Part 4]({{ '/blog/tutorials/cookbook-volume-2-04-fleet-signalling/' | relative_url }}).

## FAQ

**How does GopherTrunk detect a CTCSS tone?**
It decimates the channel to about 48 kHz, filters it to ±8 kHz, FM
discriminates, low-passes at 500 Hz and runs a Goertzel resonator at exactly
the configured frequency over 250 ms blocks. The tone must exceed the
reading of a 100 Hz-deviation tone and be 1.5× stronger than every reverse
bin — the target ±5 Hz and the adjacent EIA tones. Neither test depends on
SDR gain or sample rate.

**Why does a CTCSS gate need 250 ms to decide?**
Frequency resolution is the reciprocal of the block length. Adjacent EIA
tones sit as little as 2.3 Hz apart; a 250 ms block resolves 4 Hz and leaves
about 29 % leakage between such a pair, enough margin for the 1.5×
comparison.

**Why do 150.0 Hz and 151.4 Hz both open the same gate?**
They are 1.4 Hz apart and each leaks about 66 % of its power into the
other's bin over a 250 ms block, so the comparison has no margin. A gate on
one opens on both, and config validation refuses the pair on one shared
frequency.

**Is the CTCSS gate verified on air?**
Yes — on-air verified on 24 Sep with the reporter's Kenwood NX-300 and
NX-5000 radios and a service monitor. Two real-air slices, the same handheld
with and without a 100.0 Hz tone, are committed as regression fixtures; the
rest of the table is covered synthetically.

## Series navigation

**Part 5 of 14** · ←
[Part 4: Offset Tuning — The Tone a Clipped ADC Adds]({{ '/blog/deep-dives/squelch-04-offset-tuning/' | relative_url }})
· Next →
[Part 6: DCS — Twenty-Three Bits, Two Polarities, Many Aliases]({{ '/blog/deep-dives/squelch-06-dcs/' | relative_url }})
