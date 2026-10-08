---
title: "The Conventional Scanner, Part 9: The AM Carrier Tracker — Why a 3 kHz Tuning Error Cost a Sideband"
description: "The reporter's air-band captures carried the carrier 3315 Hz off the tuned frequency, and the ±4.5 kHz channel filter centred on that frequency cut a sideband without a log line — this part reads amCarrierAFC in the composer's am_afc.go (FFT peak within ±4.5 kHz, ≥15 dB over the median, parabolic interpolation, hold on noise, mix to DC ahead of the filter), the tests that measure the loss in the 1–3 kHz band rather than by correlation, and the 29 Sep on-air verification."
category: tutorials
keywords: am carrier tracking sdr, air band tuning error sideband, amCarrierAFC, am afc fft peak, 123.453 vs 123.450, rtl-sdr ppm air band, TestAMCaptureReplay GT_AM_IQ, am channel filter centred on carrier, kric 126.400 capture, gophertrunk conventional scanner
tags: [conventional-scanner, am, air-band, afc, composer, tutorial]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "The Conventional Scanner"
series_part: 9
---

*Part 9 of **The Conventional Scanner**, a 14-part operator's tutorial on
GopherTrunk's conventional (non-trunked) scanner, `scanner.conventional`, from
the dwell loop to a verified kitchen-sink config.
[Part 8]({{ '/blog/tutorials/conventional-scanner-08-am-for-the-air-band/' | relative_url }})
built the AM chain around a ±4.5 kHz channel filter centred on the frequency
the operator typed. The first on-air captures from
[#1219](https://github.com/MattCheramie/GopherTrunk/issues/1219) showed why
that is not enough: the carrier sat 3315 Hz from centre, the far sideband
fell outside the filter, and the audio lost 4 dB above 1 kHz with nothing in
the log to say so. This part reads the tracker that fixed it, the report
that prompted it, and — the part that cost the most — how to measure the
loss at all.*

> **TL;DR:** The reporter wrote that 123.450 MHz "tunes best at 123.453".
> Their capture — the composer's own `voice_iq_debug` cs16, i.e. the exact IQ
> the `am-conv` chain decoded — put the carrier at **−3315 Hz of 123.453**,
> so the transmitter as their SDR saw it was ≈123.4497 MHz: 123.450 was right,
> and the offset was the rig's ppm plus the radio's tolerance, not a GT
> shift. What it exposed: `amChannelCutoffHz` = 4500 centred on the TUNED
> frequency cut the lower sideband above ~1.2 kHz of audio, leaving the
> envelope detector carrier + one sideband. `amCarrierAFC`
> (`internal/voice/composer/am_afc.go`) now runs ahead of the filter: a
> 2048-point Hann FFT at 48 kHz (~23 Hz bins, every ~43 ms), the strongest
> bin within ±`amAFCSearchHz` = 4500 must stand `amAFCMinLineDb` = 15 dB over
> the window's median (a carrier does 30–55 dB, noise ≤ ~10), parabolic
> interpolation refines it, the first lock snaps and later ones smooth at
> `amAFCSmooth` = 0.3, and an NCO mixes the carrier to DC; no line ⇒ hold.
> At call end an offset ≥ `amAFCReportHz` = 1000 logs
> `composer: AM carrier found off the tuned frequency`. Pinned by
> `TestAMChainRealAirTuningErrorKeepsAudio` on the KRIC 126.400 slice
> (1–3 kHz power within ±1 dB of on-centre) and `TestAMCaptureReplay`
> (`GT_AM_IQ`). On-air verified 29 Sep.

**Key takeaways**

- **The filter must follow the carrier, not the dial.** An AM carrier is a
  strong, clean spectral line (at least half the signal power), so it is
  easy to find; centring the channel filter on it makes the audio
  independent of tuning error.
- **Measure in the band where the loss is.** The KRIC capture is ~84 % hum
  below 300 Hz; whole-band correlation stayed at 0.995 while 4.1 dB vanished
  from 1–3 kHz. `voiceBandPower` measures 1–3 kHz; correlation is not a
  verdict.
- **A tracker that locks on noise is worse than none.** The 15 dB line gate
  holds the last estimate when no carrier is present
  (`TestAMCarrierAFCHoldsOnNoise`), so a faded channel does not steer the
  filter onto a noise peak.
- **The log tells you to fix the dial anyway.** Tracking recovers the audio;
  the ≥1 kHz report at call end says `frequency_hz` or `sdr.ppm` still needs
  attention, because the squelch meter's window and the filter are finite.

## Cheat sheet

| Line / key | Meaning | Healthy / worry when |
|---|---|---|
| `composer: AM carrier found off the tuned frequency; tracked and recentred, but set the channel's frequency_hz (or sdr.ppm) closer to it` | call-end INFO, `carrier_offset_hz` ≥ 1000 | occasional on a new rig; every call ⇒ fix `sdr.ppm` or the carrier frequency |
| `amAFCSearchHz` = 4500 | where a carrier may be found; same window as `amCNMeter` | matches the ±4.5 kHz filter, clear of an 8.33 kHz neighbour |
| `amAFCMinLineDb` = 15 | peak-over-median gate before steering | carrier 30–55 dB on the #1219 captures; noise ≤ ~10 |
| `amAFCBlock` = 2048, `amAFCSmooth` = 0.3 | ~23 Hz bins, new estimate every ~43 ms; one-pole after first lock | `newAMCarrierAFC`, `estimate` |
| `TestAMChainRealAirTuningErrorKeepsAudio` | KRIC slice shifted 3315 Hz: 1–3 kHz power within ±1 dB of on-centre | `internal/voice/composer/am_realair_test.go` |
| `TestAMCaptureReplay` | `GT_AM_IQ=<cs16> [GT_AM_SHIFT_HZ=…]`: carrier, band fractions, correlation | `am_capture_replay_test.go` |
| `TestComposerAMChainTracksMistunedCarrier` | tones at 0 / ±3315 Hz all within 20 % | `am_chain_test.go` |

## In this post

- **"Tunes best at 123.453"** — reading the report against the capture.
- **What a 3 kHz offset costs** — the geometry of a DSB signal in a ±4.5 kHz window.
- **`amCarrierAFC`** — the estimator, the gate, the hold, the mix.
- **Measuring it right** — why correlation lied and the 1–3 kHz band does not.
- **Verified, and what is still soft** — the 29 Sep run, the LPF skirt, the high-pass.

## "Tunes best at 123.453"

The report was simple: the reporter's own aircraft radio, known to be on
123.450 MHz, sounded best in GopherTrunk entered as 123.453. A frequency
shift in the chain would have been the obvious reading; the capture said
otherwise. The composer's `voice_iq_debug` tap writes the exact IQ the voice
chain decoded — the channel at DC, the LO offset of
[Part 3]({{ '/blog/tutorials/conventional-scanner-03-offset-tuning/' | relative_url }})
already mixed out — and the carrier in it sits at **−3315 Hz of 123.453**. So
the transmitter, as that SDR saw it, was at ≈123.4497 MHz: 123.450 was
correct, the dongle's ppm and the radio's tolerance put the carrier ~3 kHz
below the dial, and 123.453 happened to centre the filter on it.

A few kHz of tuning error costs an FM channel nothing audible — the
discriminator sees a DC bias the high-pass removes. It costs an AM channel a
sideband.

## What a 3 kHz offset costs

Double-sideband AM with ~3 kHz audio occupies the carrier ±3 kHz. Centre a
±4.5 kHz filter on the carrier and everything passes. Centre it 3.3 kHz
away and one sideband is cut from ~1.2 kHz of audio outward: the envelope
detector then sees carrier + one full sideband + the inner part of the
other, which is roughly half the audio level above the cut plus the
distortion that comes with asymmetric sidebands.

<figure class="lab-figure">
<svg viewBox="0 0 680 190" width="680" height="190" role="img" aria-label="A frequency axis with the tuned frequency at zero and a fixed plus or minus 4.5 kHz channel filter drawn around it. A carrier line is drawn at minus 3315 Hz with sidebands extending 3 kHz either side; the lower sideband crosses the filter edge at minus 4500 Hz, and the part beyond, from about 1.2 kHz of audio outward, is shaded as lost. Below, the same scene after the tracker mixes the carrier to zero: the filter now contains both sidebands completely.">
  <text x="340" y="14" text-anchor="middle" fill="currentColor" font-size="10" font-weight="bold">±4.5 kHz filter centred on the TUNED frequency vs on the CARRIER</text>
  <line x1="40" y1="70" x2="640" y2="70" stroke="var(--fg-muted)"/>
  <text x="340" y="86" text-anchor="middle" fill="var(--fg-muted)" font-size="8">0 Hz = tuned frequency</text>
  <rect x="160" y="34" width="360" height="36" fill="none" stroke="currentColor" stroke-dasharray="4 2"/>
  <text x="160" y="30" fill="currentColor" font-size="8">−4.5 kHz</text>
  <text x="520" y="30" text-anchor="end" fill="currentColor" font-size="8">+4.5 kHz</text>
  <line x1="207" y1="38" x2="207" y2="70" stroke="var(--accent)" stroke-width="2"/>
  <text x="207" y="100" text-anchor="middle" fill="var(--accent)" font-size="8">carrier −3315 Hz</text>
  <path d="M207 66 L327 70" fill="none" stroke="var(--accent)"/>
  <path d="M207 66 L87 70" fill="none" stroke="var(--accent)"/>
  <rect x="87" y="52" width="73" height="18" fill="none" stroke="var(--accent)" stroke-dasharray="2 2"/>
  <text x="123" y="48" text-anchor="middle" fill="var(--accent)" font-size="8">lost: audio above ~1.2 kHz, lower sideband</text>
  <line x1="40" y1="150" x2="640" y2="150" stroke="var(--fg-muted)"/>
  <rect x="160" y="114" width="360" height="36" fill="none" stroke="currentColor" stroke-dasharray="4 2"/>
  <line x1="340" y1="118" x2="340" y2="150" stroke="var(--accent)" stroke-width="2"/>
  <path d="M340 146 L460 150 M340 146 L220 150" fill="none" stroke="var(--accent)"/>
  <text x="340" y="166" text-anchor="middle" fill="var(--accent)" font-size="8">after amCarrierAFC: carrier mixed to DC, both sidebands inside the filter</text>
  <text x="340" y="184" text-anchor="middle" fill="var(--fg-muted)" font-size="8">measured on the KRIC slice: 4.1 dB lost in 1–3 kHz with the old chain; within ±1 dB of on-centre with the tracker</text>
</svg>
<figcaption>The filter is the right width; it was centred on the wrong thing. Mixing the carrier to DC first makes the audio independent of the tuning error.</figcaption>
</figure>

The number is measured. `TestAMChainRealAirTuningErrorKeepsAudio` takes the
reporter's committed KRIC 126.400 MHz slice (carrier ≈ −80 Hz as captured),
moves it 3315 Hz off centre with an NCO, and records both through
`recordAnalog(t, "am-conv", …)`. On the old chain the shifted copy lost
**4.1 dB in 1–3 kHz**. The synthetic twin,
`TestComposerAMChainTracksMistunedCarrier`, puts a 1 kHz and a 2 kHz tone at
30 % depth each on carriers at 0, −3315 and +3315 Hz: before the tracker,
the 2 kHz tone recorded at roughly half its level on the offset carriers.

## `amCarrierAFC`

The fix is small and sits in one place: between the 48 kHz decimator and
the AM channel filter. `runFMChain` builds it only on the AM branch —
`amAFC = newAMCarrierAFC(intermediateHzf)` — and runs
`decimated = amAFC.Process(decimated, decimated)` before `chanFilter`. Its
premise is in the file header: an AM carrier is a strong, clean spectral
line holding at least half the signal power, so finding it is cheap, and the
search window is the same ±4.5 kHz the scanner's C/N squelch opens on, so
the squelch and the chain agree on what "the channel" is.

```go
// internal/voice/composer/am_afc.go (shape) — amCarrierAFC.estimate
peak, peakIdx := -1.0, 0
for i, k := range a.search {           // bins within ±amAFCSearchHz of DC
    m := |out[k]|²
    a.mags[i] = m
    if m > peak { peak, peakIdx = m, i }
}
med := median(a.mags)
if peak <= 0 || med <= 0 || 10*math.Log10(peak/med) < amAFCMinLineDb {
    return                             // no carrier line: hold the last estimate
}
// parabolic interpolation on log power of the peak and its neighbours
l, c, r := pow(k-1), pow(k), pow(k+1)
delta := 0.5 * (l - r) / (l - 2*c + r)
f := (float64(k) + delta) * a.rate / amAFCBlock   // wrapped to ±rate/2
if !a.locked { a.offHz, a.locked = f, true } else { a.offHz += amAFCSmooth * (f - a.offHz) }
a.nco.SetOffset(a.offHz, a.rate)
```

Reading it stage by stage:

- **Estimator.** `Process` mixes the chunk through the NCO at the *current*
  estimate first (so a chunk is never delayed), then folds the raw samples
  into a pending buffer and runs a 2048-point Hann FFT per full block: ~23 Hz
  bins at 48 kHz, a fresh estimate every ~43 ms.
- **Gate.** The strongest bin in the ±4500 Hz search set must stand
  `amAFCMinLineDb` = 15 dB over the median of that set. The margins are
  measured: a carrier stands 30–55 dB over the median on the #1219 captures,
  receiver noise at most ~10 dB (the largest of ~390 exponentially
  distributed bins over their median). Below the gate the estimator returns
  without touching anything — **hold**, not reset — which
  `TestAMCarrierAFCHoldsOnNoise` pins with a second of white noise that must
  not lock.
- **Refinement.** Parabolic interpolation on the log power of the peak and
  its two neighbours resolves below one bin; the search set is contiguous
  except at the DC wrap, where `k` and its neighbours are still adjacent
  modulo N.
- **Smoothing.** The first lock snaps to the estimate so the first chunk of
  a call is already centred; later estimates fold in at `amAFCSmooth` = 0.3
  so a momentary mis-pick cannot swing the filter.
- **Report.** `OffsetHz()` returns the tracked offset and whether a carrier
  was ever found. At chain end, if the magnitude is ≥ `amAFCReportHz`
  (1000 Hz — well past a TCXO's error, well inside the search window), the
  chain logs:

```text
INF composer: AM carrier found off the tuned frequency; tracked and recentred, but set the channel's frequency_hz (or sdr.ppm) closer to it device=CONV-R1 carrier_offset_hz=-3315
```

The line is deliberate: tracking removes the audio loss, but a carrier 3 kHz
off sits 3 kHz closer to the squelch meter's and the filter's window edge,
and 4.6 kHz would be outside both. It exists so an operator fixes `sdr.ppm`
once rather than living on the tracker.

## Measuring it right

The reason this part exists as its own post is the measurement, because the
obvious one gave the wrong answer. `TestAMCaptureReplay`
(`am_capture_replay_test.go`, skip-guarded on `GT_AM_IQ`) replays a
`voice_iq_debug` cs16 through the production AM chain twice — as captured,
and with the measured carrier mixed to DC — and prints the carrier offset,
RMS, Welch band fractions (0–300 / 300–1k / 1–2k / 2–3k / 3–4k Hz) and the
correlation between the two decodes; `GT_AM_SHIFT_HZ` moves the carrier
first so one capture A/Bs any tuning error:

```text
GT_AM_IQ=<capture.cs16> [GT_AM_RATE=2400000] [GT_AM_SHIFT_HZ=0] \
  go test ./internal/voice/composer -run TestAMCaptureReplay -v
```

On the reporter's two 28 September captures the as-captured-vs-recentred
correlation went from **0.954 / 0.886 to 1.000 / 0.999** once the tracker
landed — a clean before/after. But the same instrument then measured that
**60–83 % of the decoded energy was below 300 Hz**: the reporter's hangar
air-conditioning, hum that production's default 300 Hz `fm_audio_highpass_hz`
removes but the harness, which runs without it, does not. Hum dominates a
whole-band correlation. On the KRIC slice the **pre-fix chain still
correlates 0.995** with a 3.3 kHz error while losing **4.1 dB above 1 kHz** —
a correlation verdict would have called the broken chain fine.

So the committed regression measures the band where the lost sideband
shows. `TestAMChainRealAirTuningErrorKeepsAudio` compares `voiceBandPower`
— mean power in 1–3 kHz — of the 3315 Hz-shifted decode to the on-centre
decode and requires the ratio within 0.8..1.25 (±1 dB), logging the
correlation beside it for the record. It is the lesson the TETRA equalizer
work learned with EVM and the DMR encryption work with loudness: pick the
metric the defect actually moves.

One more detail from the synthetic twin: `mistunedAMIQ` checks both tones at
−3315 **and** +3315 Hz, because a one-sided test would pass a tracker with
the sign wrong.

## Verified, and what is still soft

**On air.** On 29 September, on a build carrying this tracker (PR #1227),
the reporter heard "very good and clear" audio, on frequency, with no extra
noise or artifacts. Their 126.400 MHz KRIC Potomac Departure capture became
the two committed 48 kHz slices that
[Part 8]({{ '/blog/tutorials/conventional-scanner-08-am-for-the-air-band/' | relative_url }})
used for the squelch and this part uses for the tracker — the same file
serving two tests, from two sides of the same chain.

**Still soft, and recorded as such.** The 127-tap 3 kHz AM audio low-pass has
a ~2 kHz skirt (−2.4 dB at 2.5 kHz); it is noted in the repo and not changed.
The harness runs without the 300 Hz high-pass, so its band fractions
over-report hum relative to what a recording contains. And the tracker's
window is finite: a carrier more than 4.5 kHz off is neither squelched nor
tracked, by design, because that is where an 8.33 kHz neighbour's carrier
would be — the fix for a rig that far off is `sdr.ppm`, which the call-end
line tells you.

## Where this goes next

The AM channel now hears the air band cleanly. The air band also carries
data: ACARS, 2400 bit/s MSK on an AM carrier, which an envelope detector
hands over without any carrier recovery at all.
[Part 10]({{ '/blog/tutorials/conventional-scanner-10-acars/' | relative_url }})
puts `decoders: [acars]` on an AM scan-list channel and reads the coherent
MSK detector, the line code that was settled empirically against acarsdec,
and a parity repair that refuses to validate garbage.

## FAQ

**Why did my air-band channel sound best tuned 3 kHz off its real frequency?**
Because the ±4.5 kHz AM channel filter was centred on the tuned frequency,
and the SDR's ppm error plus the radio's tolerance put the carrier ~3 kHz
away, cutting a sideband. The reporter's capture showed the carrier at
−3315 Hz of 123.453, i.e. ≈123.4497 MHz — 123.450 was right. `amCarrierAFC`
now centres the filter on the carrier it finds.

**What does "AM carrier found off the tuned frequency" mean?**
At call end the AM chain reports the carrier offset it tracked when it is
≥ 1000 Hz (`amAFCReportHz`). The audio was already recentred, but the
offset eats into the squelch meter's and filter's ±4.5 kHz windows; set the
channel's `frequency_hz` to the actual carrier or correct `sdr.ppm` so
future calls start centred.

**How does the AM carrier tracker avoid locking onto noise?**
`estimate` steers the NCO only when the strongest FFT bin within ±4.5 kHz
stands at least 15 dB (`amAFCMinLineDb`) over the median of that window.
A carrier measures 30–55 dB; noise at most ~10 dB. Below the gate the
tracker holds its last estimate (`TestAMCarrierAFCHoldsOnNoise`), so a faded
channel is not steered onto a noise peak.

**Why doesn't audio correlation show the lost sideband?**
The KRIC capture is ~84 % hum below 300 Hz, and the hum dominates a
whole-band correlation: the pre-fix chain correlates 0.995 with a 3.3 kHz
error while losing 4.1 dB in 1–3 kHz. `TestAMChainRealAirTuningErrorKeepsAudio`
compares `voiceBandPower` (1–3 kHz) instead and requires the shifted decode
within ±1 dB of on-centre.

**How do I check my own AM capture through the tracker?**
Enable `baseband.voice_iq_debug` to capture the chain's input, then run
`GT_AM_IQ=<capture.cs16> go test ./internal/voice/composer -run TestAMCaptureReplay -v`.
It prints the carrier offset, band fractions and the as-captured vs
recentred correlation; `GT_AM_SHIFT_HZ` simulates a tuning error on the
same file.

## Series navigation

**Part 9 of 14** · ←
[Part 8: AM for the Air Band — The Carrier as the AGC Reference]({{ '/blog/tutorials/conventional-scanner-08-am-for-the-air-band/' | relative_url }})
· Next →
[Part 10: ACARS on a Scanner Channel — Coherent MSK and Parity Repair]({{ '/blog/tutorials/conventional-scanner-10-acars/' | relative_url }})
