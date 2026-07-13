---
slug: direct-conversion-receiver
title: Direct-conversion receiver (homodyne / zero-IF)
entry_type: term
category: sdr-dsp
description: "A direct-conversion receiver mixes the wanted RF signal straight down to baseband in one stage, producing the IQ streams most SDRs digitise."
keywords: direct-conversion receiver, homodyne receiver, zero-IF, direct conversion, synchrodyne, quadrature downconversion, IQ receiver, SDR front end
aka: [homodyne receiver, zero-IF receiver, direct-conversion receiver, synchrodyne]
autolink: true
infobox:
  - { label: Type, value: Receiver architecture }
  - { label: IF, value: 0 Hz (baseband) }
  - { label: Output, value: Complex I/Q pair }
see_also: [zero-if, iq-imbalance, dc-offset, superheterodyne-receiver, low-if, iq-modulation]
cite_urls:
  - https://en.wikipedia.org/wiki/Direct-conversion_receiver
  - https://en.wikipedia.org/wiki/Homodyne_detection
---

A **direct-conversion receiver** — also called a **homodyne** or **zero-IF** receiver —
translates the wanted radio signal directly from its carrier frequency down to
[baseband](/reference/baseband/) in a single mixing stage, with no intervening
[intermediate frequency](/reference/intermediate-frequency/).[^wiki] Because the
[local oscillator](/reference/local-oscillator/) is tuned to the carrier itself, the
mixer output lands centred on 0 Hz, and a pair of mixers driven 90° apart delivers the
in-phase and quadrature ([IQ](/reference/iq-data/)) streams an SDR digitises. It is the
dominant front-end architecture in low-cost SDRs and modern integrated radios precisely
because it collapses the whole down-conversion chain into one step.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 170" role="img" aria-label="Antenna feeds a splitter into two mixers driven by a local oscillator with a 90-degree phase shift, each mixer followed by a low-pass filter producing the in-phase I and quadrature Q baseband outputs." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="dcrar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <text x="14" y="55" font-size="9" fill="currentColor">RF in</text>
  <line x1="14" y1="65" x2="70" y2="65" stroke="currentColor" marker-end="url(#dcrar)"/>
  <line x1="70" y1="35" x2="70" y2="105" stroke="currentColor"/>
  <line x1="70" y1="35" x2="120" y2="35" stroke="currentColor" marker-end="url(#dcrar)"/>
  <line x1="70" y1="105" x2="120" y2="105" stroke="currentColor" marker-end="url(#dcrar)"/>
  <circle cx="135" cy="35" r="15" fill="none" stroke="currentColor"/><text x="129" y="39" font-size="12" fill="currentColor">×</text>
  <circle cx="135" cy="105" r="15" fill="none" stroke="currentColor"/><text x="129" y="109" font-size="12" fill="currentColor">×</text>
  <rect x="215" y="70" width="34" height="30" fill="none" stroke="currentColor"/><text x="221" y="88" font-size="8" fill="currentColor">LO</text>
  <line x1="215" y1="78" x2="150" y2="45" stroke="currentColor" stroke-opacity="0.7"/>
  <line x1="215" y1="92" x2="150" y2="105" stroke="currentColor" stroke-opacity="0.7"/>
  <text x="205" y="120" font-size="8" fill="currentColor">90° split</text>
  <line x1="150" y1="35" x2="300" y2="35" stroke="currentColor" marker-end="url(#dcrar)"/>
  <line x1="150" y1="105" x2="300" y2="105" stroke="currentColor" marker-end="url(#dcrar)"/>
  <rect x="300" y="22" width="40" height="26" fill="none" stroke="currentColor"/><text x="308" y="39" font-size="8" fill="currentColor">LPF</text>
  <rect x="300" y="92" width="40" height="26" fill="none" stroke="currentColor"/><text x="308" y="109" font-size="8" fill="currentColor">LPF</text>
  <line x1="340" y1="35" x2="400" y2="35" stroke="currentColor" marker-end="url(#dcrar)"/>
  <line x1="340" y1="105" x2="400" y2="105" stroke="currentColor" marker-end="url(#dcrar)"/>
  <text x="405" y="39" font-size="11" fill="currentColor">I</text>
  <text x="405" y="109" font-size="11" fill="currentColor">Q</text>
</svg>
<figcaption>Two mixers fed by a 90°-split local oscillator translate the RF carrier straight to 0 Hz, yielding the I and Q baseband channels.</figcaption>
</figure>

## How it works

The receiver sets its [local oscillator](/reference/local-oscillator/) to the exact
carrier frequency of the channel of interest. A [mixer](/reference/mixer-rf/) multiplies
the incoming RF by this oscillator, producing sum and difference terms; a
[low-pass filter](/reference/digital-filter/) keeps the difference, which — because the
oscillator equals the carrier — sits at 0 Hz. A single real mixer cannot tell a signal
just above the carrier from one just below it (they fold onto the same baseband
frequency), so a homodyne receiver uses **two** mixers with the oscillator phase-shifted
90° between them. The two outputs, I (in-phase) and Q (quadrature), form a complex
signal in which positive and negative offsets are distinct, preserving the full spectrum
either side of the tuning point.

This quadrature pair is exactly what an [ADC](/reference/analog-to-digital-converter/)
digitises in an SDR, so the architecture maps naturally onto
[IQ demodulation](/reference/iq-modulation/) in software.

## In practice

Direct conversion trades the [superheterodyne](/reference/superheterodyne-receiver/)
receiver's IF filtering for a set of well-known baseband impairments:

- **[DC offset](/reference/dc-offset/).** Local-oscillator energy leaks back to the
  mixer input and self-mixes to a large, drifting spike at 0 Hz — right in the middle of
  the wanted signal. Receivers remove it with DC-blocking, high-pass filtering, or
  digital estimation.
- **[IQ imbalance](/reference/iq-imbalance/).** Any gain or phase mismatch between the I
  and Q paths breaks the quadrature symmetry, letting an unwanted **image** of each
  signal appear at its mirror frequency. Calibration or adaptive correction restores
  balance.
- **Flicker (1/f) noise.** Because the signal lives near DC, low-frequency device noise
  falls directly in band, hurting sensitivity for narrowband modes.
- **Even-order distortion and LO radiation.** Strong nearby signals can rectify to
  baseband, and the on-frequency oscillator can leak out of the antenna.

The [low-IF](/reference/low-if/) architecture is a common compromise: it places the
signal at a small nonzero offset to dodge the DC and flicker problems while keeping most
of direct conversion's simplicity.

## Relevance to SDR

Almost every affordable SDR uses direct conversion. The tuners behind
[RTL-SDR](/reference/rtl-sdr/) dongles, the [Airspy](/reference/airspy/) family,
[HackRF](/reference/hackrf/), [LimeSDR](/reference/limesdr/), and
[PlutoSDR](/reference/plutosdr/) all present a zero-IF complex-baseband stream to the
host. That is why the impairments above are staples of SDR life: the DC spike at the
centre of the [waterfall](/reference/waterfall-display/) and the faint mirror-image
signals from residual [IQ imbalance](/reference/iq-imbalance/) are both fingerprints of a
homodyne front end.

GopherTrunk consumes the complex IQ these receivers produce and does its channel
selection, filtering, and demodulation in software. It does not build the analog front
end, but its DSP routinely copes with the DC offset and image artefacts that direct
conversion leaves behind, and understanding the architecture explains why those
artefacts appear where they do.

## Sources

[^wiki]: [Direct-conversion receiver](https://en.wikipedia.org/wiki/Direct-conversion_receiver) — Wikipedia, on homodyne/zero-IF architecture and its DC-offset and IQ-imbalance impairments.
