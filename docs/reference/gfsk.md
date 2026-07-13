---
slug: gfsk
title: GFSK
entry_type: technology
category: modulation
description: "GFSK (Gaussian frequency-shift keying) is FSK whose transitions are smoothed by a Gaussian filter to narrow the spectrum — used by AIS, Bluetooth, and IoT radios."
keywords: GFSK, Gaussian frequency-shift keying, AIS modulation, pulse shaping, narrowband FSK, Bluetooth, BT product, modulation index, spectral efficiency
aka: [GFSK, Gaussian FSK]
autolink: true
see_also: [frequency-shift-keying, gmsk, minimum-shift-keying, pulse-shaping, intersymbol-interference, ais]
related_lessons:
  - { title: "Digital modulation & constellations", url: /learn/rf-sdr/digital-modulation/ }
related_reading:
  - { title: "SDR Internals, Part 6: Demodulation", url: /blog/deep-dives/sdr-internals-06-demodulation/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Frequency-shift_keying#Gaussian_frequency-shift_keying
  - https://en.wikipedia.org/wiki/Minimum-shift_keying#Gaussian_minimum-shift_keying
---

**GFSK** (**Gaussian frequency-shift keying**) is [frequency-shift keying](/reference/frequency-shift-keying/)
in which the data is passed through a **Gaussian filter** before it shifts the carrier,
rounding off the otherwise abrupt frequency steps.[^wiki] This smoothing narrows the
transmitted spectrum, so GFSK fits more signals into less bandwidth than plain FSK carrying the
same data rate.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 120" role="img" aria-label="A sharp two-level FSK transition compared with a smoothly rounded Gaussian-filtered transition." xmlns="http://www.w3.org/2000/svg">
  <path d="M30 80 H100 V40 H170 V80 H240" fill="none" stroke="currentColor" stroke-width="1.3" stroke-opacity="0.5"/>
  <text x="135" y="104" text-anchor="middle" font-size="8.5" fill="currentColor">plain FSK (sharp)</text>
  <path d="M260 80 C 300 80 300 40 330 40 C 360 40 360 80 400 80 C 415 80 420 78 430 78" fill="none" stroke="currentColor" stroke-width="1.8"/>
  <text x="350" y="104" text-anchor="middle" font-size="8.5" fill="currentColor">GFSK (rounded)</text>
</svg>
<figcaption>A Gaussian filter rounds the frequency transitions, narrowing the spectrum at a small cost in detectability.</figcaption>
</figure>

## How it works

In plain FSK the modulating waveform is a rectangular sequence of levels — the frequency jumps
instantly from one tone to another at each symbol boundary. Those instantaneous jumps have sharp
corners, and sharp corners contain high-frequency energy that spreads the transmitted spectrum
into neighbouring channels. GFSK first filters the level sequence with a **Gaussian-shaped
lowpass**, so the frequency glides smoothly from one value to the next instead of stepping. The
result is a continuous-phase signal with a much more compact spectrum.

The filter is characterised by its **BT product** — the Gaussian filter's bandwidth *B*
multiplied by the symbol period *T*. A smaller BT (e.g. 0.3, as in Bluetooth) filters more
aggressively: the spectrum is tighter, but the smoothing now spreads each symbol's influence over
several neighbours, introducing controlled
[intersymbol interference](/reference/intersymbol-interference/) that shrinks the receiver's
noise margin. A larger BT (e.g. 0.5) keeps the symbols cleaner at the cost of a wider spectrum.
Choosing BT is the central design trade in GFSK: spectral compactness versus detectability.
Because GFSK keeps the phase continuous and never blanks the carrier, its envelope is essentially
constant, which — like [π/4-DQPSK](/reference/pi-4-dqpsk/) — allows efficient non-linear
amplification in cheap handheld and IoT transmitters.

## Variants

GFSK is one member of the continuous-phase / [pulse-shaped](/reference/pulse-shaping/) FSK
family. When the modulation index is exactly 0.5 and the filter is Gaussian, GFSK becomes
[GMSK](/reference/gmsk/) (Gaussian minimum-shift keying), the special case used by GSM;
[minimum-shift keying](/reference/minimum-shift-keying/) is the unfiltered index-0.5 form.
Higher-order **4-GFSK** carries two bits per symbol using four Gaussian-smoothed tones and is
used by higher-rate Bluetooth and some DMR-adjacent links. The unshaped ancestor is ordinary
2-level or [4-level FSK](/reference/frequency-shift-keying/).

## In practice

GFSK is ubiquitous wherever spectral efficiency and cheap constant-envelope hardware matter more
than raw sensitivity: **Bluetooth** and Bluetooth Low Energy (BT ≈ 0.5), the **AIS** marine
transponder standard (a GMSK/GFSK variant at 9600 baud), classic ISM-band remote controls,
wireless keyboards and mice, and a wide range of low-power [IoT](/reference/internet-of-things/)
radios. Its combination of a narrow occupied bandwidth, a constant envelope, and a simple
one-bit-per-symbol (or two-bit) alphabet makes it the default choice for low-cost narrowband
digital links.

## Relevance to SDR

For software decoding, GFSK is handled much like other FSK: an FM/frequency discriminator or a
correlator recovers the instantaneous frequency, a filter matched to the Gaussian pulse maximises
the signal-to-noise ratio, and [symbol-timing recovery](/reference/clock-recovery/) then slices
the levels. The Gaussian smoothing means adjacent symbols overlap, so higher-performance
receivers use a sequence estimator (Viterbi) that accounts for the deliberate ISI rather than
slicing each symbol independently. GopherTrunk's FSK demod path covers the constant-envelope
frequency modulations used by the trunking and paging systems it targets; AIS itself is a
maritime data protocol tracked separately in the project's coverage.

## Sources

[^wiki]: [Frequency-shift keying — Gaussian frequency-shift keying](https://en.wikipedia.org/wiki/Frequency-shift_keying#Gaussian_frequency-shift_keying) — Wikipedia, for the Gaussian-filtered FSK definition.
[^gmsk]: [Minimum-shift keying — Gaussian minimum-shift keying](https://en.wikipedia.org/wiki/Minimum-shift_keying#Gaussian_minimum-shift_keying) — Wikipedia, for the BT-product trade-off and the GMSK special case.
