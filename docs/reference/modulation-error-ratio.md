---
slug: modulation-error-ratio
title: Modulation error ratio (MER)
entry_type: term
category: rf-metrics
description: Modulation error ratio is the ratio of ideal signal power to constellation error power in decibels, an aggregate SNR-like measure of digital modulation quality.
keywords: modulation error ratio, MER, constellation SNR, digital modulation quality, MER dB, cable DOCSIS, DVB quality, EVM relationship
aka: [MER]
autolink: true
infobox:
  - { label: Symbol, value: MER }
  - { label: Unit, value: Decibels (dB) }
  - { label: Relation, value: "MER ≈ −EVM(dB)" }
see_also: [error-vector-magnitude, constellation-diagram, signal-to-noise-ratio, quadrature-amplitude-modulation]
cite_urls:
  - https://en.wikipedia.org/wiki/Modulation_error_ratio
---

**Modulation error ratio** (**MER**) is the ratio, in [decibels](/reference/decibel/),
of the average power of the ideal transmitted symbols to the average power of the
constellation error — the scatter of received symbols away from their ideal
positions.[^wiki] It is effectively a [constellation](/reference/constellation-diagram/)
signal-to-noise ratio, and higher MER means cleaner, more tightly clustered symbols and
a more decodable signal.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 200" role="img" aria-label="A four-point QPSK constellation where each ideal point is surrounded by a cloud of received samples, with an inset showing MER as the ratio of average signal power to average error power in decibels." xmlns="http://www.w3.org/2000/svg">
  <line x1="40" y1="100" x2="240" y2="100" stroke="currentColor" stroke-opacity="0.5"/>
  <line x1="140" y1="20" x2="140" y2="180" stroke="currentColor" stroke-opacity="0.5"/>
  <g fill="currentColor">
    <circle cx="190" cy="60" r="1.4"/><circle cx="196" cy="66" r="1.4"/><circle cx="184" cy="64" r="1.4"/><circle cx="192" cy="55" r="1.4"/><circle cx="188" cy="70" r="1.4"/>
    <circle cx="90" cy="60" r="1.4"/><circle cx="96" cy="66" r="1.4"/><circle cx="84" cy="55" r="1.4"/><circle cx="92" cy="70" r="1.4"/><circle cx="86" cy="64" r="1.4"/>
    <circle cx="190" cy="140" r="1.4"/><circle cx="184" cy="146" r="1.4"/><circle cx="196" cy="134" r="1.4"/><circle cx="188" cy="150" r="1.4"/><circle cx="192" cy="138" r="1.4"/>
    <circle cx="90" cy="140" r="1.4"/><circle cx="96" cy="146" r="1.4"/><circle cx="84" cy="134" r="1.4"/><circle cx="92" cy="150" r="1.4"/><circle cx="86" cy="138" r="1.4"/>
  </g>
  <circle cx="190" cy="60" r="12" fill="none" stroke="currentColor" stroke-dasharray="2 2" stroke-opacity="0.7"/>
  <text x="300" y="90" font-size="11" fill="currentColor">MER (dB) =</text>
  <text x="300" y="108" font-size="10" fill="currentColor">10·log₁₀( P_signal /</text>
  <text x="330" y="122" font-size="10" fill="currentColor">P_error )</text>
</svg>
<figcaption>MER compares the average power of the ideal symbols to the average power of the error clouds; tighter clusters mean higher MER.</figcaption>
</figure>

## How it works

MER is computed over a block of symbols. Sum the squared magnitudes of the ideal
reference points to get the signal power, sum the squared magnitudes of the error
vectors — the differences between measured
[IQ](/reference/iq-data/) samples and their ideal points — to get the error power,
then take:

MER(dB) = 10 · log₁₀ ( Σ|ideal|² / Σ|error|² )

Because it is a ratio of powers averaged over the whole constellation, MER behaves like
an [SNR](/reference/signal-to-noise-ratio/) and can be read the same way: a marginal
digital channel might sit at 15–20 dB MER, a solid one at 30 dB or more. It captures the
combined effect of noise, phase noise, [IQ imbalance](/reference/iq-imbalance/),
[intersymbol interference](/reference/intersymbol-interference/), and nonlinearity in one
number.

MER is the reciprocal, in the log domain, of
[error vector magnitude](/reference/error-vector-magnitude/): for the same reference,
MER(dB) ≈ −EVM(dB), or MER(dB) = −20·log₁₀(EVM_rms). The two metrics carry the same
information; which one a standard quotes is largely convention. Cable, DOCSIS, and
digital-broadcast worlds favor MER in dB; cellular and Wi-Fi favor EVM in percent.

## In practice

- MER degrades before symbol errors appear, so it is an early-warning quality indicator
  — a channel can hold zero errors while MER slowly falls toward the failure threshold.
- Each modulation order needs a minimum MER: denser
  [QAM](/reference/quadrature-amplitude-modulation/) constellations, with points packed
  closer together, demand higher MER than sparse ones like
  [QPSK](/reference/qpsk/) for the same [bit error rate](/reference/bit-error-rate/).
- Unlike a raw SNR from a spectrum measurement, MER is computed on the *demodulated*
  symbols, so it reflects everything the receiver actually sees after equalization and
  synchronization.

## Relevance to SDR

MER is the primary quality metric in cable/DOCSIS and in [DVB](/reference/dvb-t/) and
[ATSC](/reference/atsc-1/) digital broadcast, and it is used to grade land-mobile
digital transmitters alongside EVM. For a software receiver like
[GopherTrunk](/reference/software-defined-radio/), the per-symbol error it computes
while demodulating [P25](/reference/p25-phase-1/), [DMR](/reference/dmr/), or
[TETRA](/reference/tetra/) constellations is exactly the quantity MER summarizes. Read
as a constellation SNR, it tells you at a glance whether a channel is comfortably locked
or teetering on the decode cliff — the same insight EVM gives, expressed as a
higher-is-better dB figure.

## Sources

[^wiki]: [Modulation error ratio](https://en.wikipedia.org/wiki/Modulation_error_ratio) — Wikipedia, definition, formula, and relationship to EVM and SNR.
