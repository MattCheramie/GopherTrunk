---
slug: preamble-correlation
title: Preamble correlation
entry_type: term
category: sdr-dsp
description: "Preamble correlation slides a matched filter for a known preamble along the received stream and declares detection at the correlation peak, marking packet start."
keywords: preamble correlation, preamble detection, correlator, matched filter, cross-correlation, sync detection, packet detection, correlation peak, acquisition
aka: [preamble detection, sync correlation, correlator detection]
autolink: true
infobox:
  - { label: Type, value: Detection / acquisition stage }
  - { label: Method, value: Matched-filter cross-correlation }
  - { label: Output, value: Peak position = packet start }
see_also: [matched-filter, frame-synchronization, barker-code, clock-recovery, gold-code, symbol-rate]
cite_urls:
  - https://en.wikipedia.org/wiki/Matched_filter
  - https://en.wikipedia.org/wiki/Cross-correlation
---

**Preamble correlation** is the technique that detects the start of a packet by sliding a
[matched filter](/reference/matched-filter/) for a known **preamble** along the received stream
and declaring detection wherever the cross-correlation forms a sharp peak.[^wiki] The preamble is
a fixed sequence the transmitter sends before the payload; correlating against a stored copy of
it is the optimal way to find that sequence buried in noise, and the position of the peak fixes
where the packet — and its timing — begins.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 165" role="img" aria-label="A noisy received stream passing through a correlator matched to the preamble, producing a flat low output that jumps to a single sharp peak when the preamble aligns." xmlns="http://www.w3.org/2000/svg">
  <path d="M20 45 L28 40 L36 52 L44 38 L52 50 L60 42 L68 48 L76 40 L84 51 L92 44 L100 47" fill="none" stroke="currentColor" stroke-width="1.1"/>
  <text x="60" y="30" font-size="8.5" fill="currentColor" text-anchor="middle">received + noise</text>
  <rect x="150" y="30" width="86" height="34" rx="5" fill="none" stroke="currentColor" stroke-width="1.4"/>
  <text x="193" y="45" font-size="8.5" fill="currentColor" text-anchor="middle">correlator</text>
  <text x="193" y="57" font-size="7" fill="currentColor" text-anchor="middle">h[n]=preamble*</text>
  <line x1="105" y1="47" x2="149" y2="47" stroke="currentColor" stroke-width="1.1" marker-end="url(#pcar)"/>
  <line x1="236" y1="47" x2="280" y2="47" stroke="currentColor" stroke-width="1.1" marker-end="url(#pcar)"/>
  <line x1="285" y1="140" x2="450" y2="140" stroke="currentColor" stroke-width="1.1"/>
  <path d="M285 132 L330 130 L360 133 L378 131 L388 45 L398 131 L420 130 L450 132" fill="none" stroke="currentColor" stroke-width="1.5"/>
  <line x1="388" y1="45" x2="388" y2="150" stroke="currentColor" stroke-width="0.8" stroke-dasharray="3 3"/>
  <text x="388" y="162" font-size="8" fill="currentColor" text-anchor="middle">peak = packet start</text>
  <text x="320" y="120" font-size="8" fill="currentColor">correlation output</text>
  <defs><marker id="pcar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>The correlator's output stays near noise level until the preamble aligns with its stored template, where it spikes; the peak's location times the packet.</figcaption>
</figure>

## How it works

Correlation computes, at each offset, the sliding dot product of the incoming samples with a
time-reversed, conjugated copy of the known preamble — which is exactly what a
[matched filter](/reference/matched-filter/) does. When the preamble is not present the terms add
with random signs and the output hovers near zero; when it aligns, every term reinforces and the
output jumps to a large peak. This maximizes the signal-to-noise ratio *at the peak*, so it is
the optimal detector for a known sequence in additive white noise, regardless of how the noise
scatters the individual samples.

The quality of detection depends on the preamble's autocorrelation. An ideal preamble has one tall
main lobe and small sidelobes, so the peak is unmistakable and a slight misalignment does not
raise a competing bump. Sequences engineered for this — [Barker codes](/reference/barker-code/),
[Gold codes](/reference/gold-code/), Zadoff–Chu sequences — are chosen precisely because their
sidelobes are low. Practical detectors add:

- **Normalization** — dividing the correlation by the running input energy, so the threshold
  works regardless of absolute signal level (a constant-false-alarm-rate detector).
- **Non-coherent operation** — taking the magnitude of a complex correlation so detection
  survives an unknown carrier phase, at a small SNR cost versus a coherent (known-phase)
  correlator.
- **A threshold** balancing missed detections against false alarms, often set from the measured
  noise floor.

Beyond finding the packet, the peak's exact sample index gives a coarse **timing estimate**, and
the complex value at the peak carries the carrier phase and frequency offset — useful seeds for
[clock recovery](/reference/clock-recovery/) and carrier correction that follow.

## In practice

Correlation is computed one of two ways. A time-domain sliding correlator is cheap for a short
preamble and can run continuously, one multiply-accumulate window per sample. For a long preamble
it is more efficient to correlate in the **frequency domain** — an FFT-based fast convolution
computes the whole correlation for a block at once, trading latency for far fewer operations, the
same overlap-save trick used for long FIR filters. Either way, a carrier frequency offset is the
practical enemy: a large offset rotates the samples during the correlation window and shrinks the
peak, so wideband systems either search a bank of frequency hypotheses in parallel or use a
**differential** correlation (correlating sample-to-sample phase changes rather than absolute
phase) that is inherently offset-tolerant at some SNR cost. The preamble length is chosen to
balance these forces: longer gives a taller, sharper peak and better estimates, but costs airtime
and widens the frequency-search burden.

## Relevance to SDR

Preamble correlation is how nearly every packet radio acquires a burst. Wi-Fi correlates its
short/long training fields, Bluetooth and Zigbee correlate an access-address/preamble, LoRa
correlates chirp symbols, and LTE/5G correlate Zadoff–Chu synchronization signals. In land-mobile
digital voice the same mechanism finds the [frame-sync](/reference/frame-synchronization/)
pattern that marks each frame. GopherTrunk relies on correlation-style sync detection inside its
protocol decoders: matching the known P25/DMR/NXDN sync sequence against the demodulated symbols
is what triggers frame alignment and lets the parsers lock onto a control or voice channel. Because
the sync words in these land-mobile formats are short and the symbols are already timing-recovered,
the correlation runs in the time domain over the symbol stream rather than through an FFT, and
tolerating a few symbol errors lets the detector hold sync through brief fades instead of dropping
the channel on every glitch.

## Sources

[^wiki]: [Matched filter](https://en.wikipedia.org/wiki/Matched_filter) — Wikipedia, on the correlation receiver as the optimal detector of a known signal in white noise.
