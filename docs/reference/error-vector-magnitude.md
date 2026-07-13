---
slug: error-vector-magnitude
title: Error vector magnitude (EVM)
entry_type: term
category: rf-metrics
description: Error vector magnitude measures how far received constellation symbols land from their ideal positions, expressed as a percentage or in decibels, quantifying modulation quality.
keywords: error vector magnitude, EVM, modulation quality, constellation error, RMS EVM, transmitter quality, receiver quality, percent EVM, EVM dB
aka: [EVM, error vector, receive constellation error]
autolink: true
infobox:
  - { label: Symbol, value: EVM }
  - { label: Unit, value: "% (RMS) or dB" }
  - { label: Measures, value: Constellation symbol error }
see_also: [constellation-diagram, modulation-error-ratio, signal-to-noise-ratio, quadrature-amplitude-modulation, eye-diagram]
cite_urls:
  - https://en.wikipedia.org/wiki/Error_vector_magnitude
---

**Error vector magnitude** (**EVM**) measures modulation quality by comparing where
each received symbol actually lands on the
[constellation diagram](/reference/constellation-diagram/) against where it ideally
should be.[^wiki] The **error vector** is the line joining the ideal point to the
measured point; EVM is the magnitude of that error, averaged (root-mean-square) over
many symbols and normalized to the constellation's scale, reported as a percentage or
in [decibels](/reference/decibel/). Lower EVM means tighter, cleaner symbols.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 200" role="img" aria-label="A constellation quadrant showing an ideal reference point and a measured point offset from it, with the connecting error vector broken into magnitude and phase components." xmlns="http://www.w3.org/2000/svg">
  <line x1="40" y1="100" x2="300" y2="100" stroke="currentColor" stroke-opacity="0.5"/>
  <line x1="170" y1="20" x2="170" y2="180" stroke="currentColor" stroke-opacity="0.5"/>
  <text x="295" y="114" font-size="9" fill="currentColor">I</text>
  <text x="176" y="28" font-size="9" fill="currentColor">Q</text>
  <line x1="170" y1="100" x2="250" y2="55" stroke="currentColor" stroke-width="1.2" stroke-dasharray="3 2"/>
  <circle cx="250" cy="55" r="4" fill="none" stroke="currentColor" stroke-width="1.5"/>
  <text x="256" y="50" font-size="9" fill="currentColor">ideal</text>
  <circle cx="272" cy="72" r="4" fill="currentColor"/>
  <text x="278" y="70" font-size="9" fill="currentColor">measured</text>
  <line x1="250" y1="55" x2="272" y2="72" stroke="currentColor" stroke-width="2"/>
  <text x="262" y="92" font-size="9" fill="currentColor">error vector</text>
  <text x="330" y="120" font-size="10" fill="currentColor">EVM = |error| / |ref|</text>
</svg>
<figcaption>The error vector links each symbol's ideal position to where it actually landed; EVM is its RMS magnitude relative to the constellation scale.</figcaption>
</figure>

## How it works

For each received symbol the receiver computes the complex difference between the
measured [IQ](/reference/iq-data/) sample and the nearest ideal constellation point.
That difference is the error vector. RMS EVM is the square root of the mean squared
error-vector magnitude, divided by a reference — typically the RMS or peak magnitude of
the ideal constellation:

EVM(%) = 100 × √(mean|error|²) / |reference|

Because it captures the *total* deviation, EVM lumps together every impairment that
scatters symbols: additive noise, phase noise, [IQ imbalance](/reference/iq-imbalance/),
[intersymbol interference](/reference/intersymbol-interference/), amplifier
nonlinearity, carrier-frequency and timing error, and
[DC offset](/reference/dc-offset/). This makes it a single, convenient scalar for
overall signal health — but it also means a high EVM alone does not tell you which
impairment is to blame; the *shape* of the error cloud on the constellation does.

EVM and [modulation error ratio](/reference/modulation-error-ratio/) are two views of
the same measurement — MER is essentially EVM expressed as a power ratio in dB, so
EVM(dB) = −MER(dB) when both use the same reference. For an
[additive white Gaussian noise](/reference/thermal-noise/) channel, EVM relates
directly to [SNR](/reference/signal-to-noise-ratio/): EVM(%) ≈ 100 / √(SNR_linear).

## Variants

- **RMS vs peak EVM** — RMS characterizes average quality; peak EVM captures the worst
  single symbol, relevant for occasional errors.
- **Percentage vs dB** — percentage is common in cellular and Wi-Fi specs; dB is common
  in broadcast and cable. −40 dB, 1%, and a very tight cloud all describe the same
  excellent link.
- **Normalization reference** — average power, peak power, or outermost symbol; specs
  must state which, since the numeric result depends on it.

## Relevance to SDR

EVM is the standard transmitter- and receiver-quality metric across
[LTE](/reference/lte/), [5G NR](/reference/5g-nr/), [Wi-Fi](/reference/wifi-80211/),
and [DVB](/reference/dvb-t/), where standards set hard EVM limits — for example a few
percent for high-order [QAM](/reference/quadrature-amplitude-modulation/). In
land-mobile digital voice, the closely related MER or symbol deviation is used to grade
[P25](/reference/p25-phase-1/) and [DMR](/reference/dmr/) transmitters. A software
receiver such as [GopherTrunk](/reference/software-defined-radio/) computes an
equivalent per-symbol error while it demodulates
[C4FM](/reference/c4fm/)/[π-4-DQPSK](/reference/pi-4-dqpsk/) constellations, and that
error — surfaced as an EVM or SNR estimate — is a direct, real-time gauge of whether a
channel is locked and decodable. Watching EVM rise as a signal fades is a practical way
to see the decode cliff approaching before frames actually start failing.

## Sources

[^wiki]: [Error vector magnitude](https://en.wikipedia.org/wiki/Error_vector_magnitude) — Wikipedia, definition, formula, and relationship to SNR and MER.
