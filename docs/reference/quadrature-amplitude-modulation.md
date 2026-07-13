---
slug: quadrature-amplitude-modulation
title: Quadrature amplitude modulation (QAM)
entry_type: technology
category: modulation
description: QAM combines amplitude and phase modulation to pack many bits per symbol; higher-order QAM carries more data but needs a higher SNR to decode.
keywords: QAM, quadrature amplitude modulation, 16-QAM, 64-QAM, 256-QAM, 1024-QAM, constellation, bits per symbol, spectral efficiency, OFDM
aka: [quadrature amplitude modulation, QAM]
autolink: true
infobox:
  - { label: Type, value: Digital modulation }
  - { label: Varies, value: Phase and amplitude }
  - { label: Used by, value: Wi-Fi, cable, LTE, broadcast }
see_also: [phase-shift-keying, frequency-shift-keying, constellation-diagram, signal-to-noise-ratio, iq-data, iq-modulation, spectral-efficiency, qpsk, 8psk, bit-rate-vs-baud, ofdm]
related_lessons:
  - { title: "Digital modulation & constellations", url: /learn/rf-sdr/digital-modulation/ }
related_reading:
  - { title: "SDR Internals, Part 6: Demodulation", url: /blog/deep-dives/sdr-internals-06-demodulation/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Quadrature_amplitude_modulation
  - https://en.wikipedia.org/wiki/Shannon%E2%80%93Hartley_theorem
---

**Quadrature amplitude modulation** (**QAM**) varies **both** the
[phase](/reference/phase/) and [amplitude](/reference/amplitude/) of a carrier, packing
many states into the [IQ](/reference/iq-data/) plane — 16-QAM (4 bits/symbol), 64-QAM
(6 bits/symbol), 256-QAM (8 bits/symbol), and higher.[^wiki] It is the highest-throughput
digital modulation in common use, and the workhorse of every high-rate link — Wi-Fi,
cellular data, cable, and digital TV.

<figure class="figure" markdown="0">
<svg viewBox="0 0 240 240" role="img" aria-label="A 16-QAM constellation: a four-by-four grid of points on the IQ plane." xmlns="http://www.w3.org/2000/svg">
  <line x1="20" y1="120" x2="220" y2="120" stroke="currentColor" stroke-opacity="0.4"/>
  <line x1="120" y1="20" x2="120" y2="220" stroke="currentColor" stroke-opacity="0.4"/>
  <text x="212" y="134" font-size="10" fill="currentColor">I</text><text x="106" y="30" font-size="10" fill="currentColor">Q</text>
  <g fill="currentColor">
    <circle cx="60" cy="60" r="4"/><circle cx="100" cy="60" r="4"/><circle cx="140" cy="60" r="4"/><circle cx="180" cy="60" r="4"/>
    <circle cx="60" cy="100" r="4"/><circle cx="100" cy="100" r="4"/><circle cx="140" cy="100" r="4"/><circle cx="180" cy="100" r="4"/>
    <circle cx="60" cy="140" r="4"/><circle cx="100" cy="140" r="4"/><circle cx="140" cy="140" r="4"/><circle cx="180" cy="140" r="4"/>
    <circle cx="60" cy="180" r="4"/><circle cx="100" cy="180" r="4"/><circle cx="140" cy="180" r="4"/><circle cx="180" cy="180" r="4"/>
  </g>
</svg>
<figcaption>QAM varies both phase and amplitude; 16-QAM packs 4 bits per symbol but needs higher SNR to keep the points apart.</figcaption>
</figure>

## How it works

QAM sends two independent amplitude-modulated streams on carriers 90° apart — the
in-phase (I) and quadrature (Q) components — and sums them. Because sine and cosine are
orthogonal, the receiver can separate the two by
[quadrature (IQ) demodulation](/reference/iq-modulation/) without crosstalk, so a single
channel carries two bitstreams. Plotted on the IQ plane, the allowed states form a
**grid** — 4×4 for 16-QAM, 8×8 for 64-QAM — and each grid point is a symbol worth
log₂(M) bits. [Phase-shift keying](/reference/phase-shift-keying/) is the special case
where every point sits on one amplitude circle; QAM's extra degree of freedom (radius)
lets it pack points more densely for the same peak power.

The fundamental trade-off is set by [Shannon](/reference/shannon-capacity/): more bits
per symbol raises [spectral efficiency](/reference/spectral-efficiency/) but forces the
constellation points closer together, so a smaller amount of noise pushes a received
symbol across a decision boundary. Each step up the QAM ladder (16→64→256) buys 2 more
bits per symbol but demands roughly 6 dB more [SNR](/reference/signal-to-noise-ratio/) to
keep the [bit error rate](/reference/bit-error-rate/) constant.[^shannon] This is why
adaptive systems pick the modulation order to match the link: a strong signal runs
1024-QAM, a weak one falls back to QPSK. QAM also requires a **linear** power amplifier,
since the amplitude information would be destroyed by a saturated amplifier — the opposite
of the constant-envelope [FSK](/reference/frequency-shift-keying/) family used by handheld
radios.

## In practice

QAM almost never rides on a single carrier in modern systems; it is the per-subcarrier
payload of [OFDM](/reference/ofdm/), which spreads thousands of narrow QAM-modulated
subcarriers across the channel. Wi-Fi runs up to 256-QAM (802.11ac) and 1024-QAM
(802.11ax); [LTE](/reference/lte/) and 5G scale 16/64/256-QAM by link quality; cable
(DOCSIS) and [DVB-C](/reference/dvb-c/) use 64- to 4096-QAM; and digital broadcast TV
leans on it heavily. Symbol density means these signals need accurate
[equalisation](/reference/adaptive-filter/) and phase tracking, and their health is read
directly off constellation tightness ([EVM / error vector
magnitude](/reference/error-vector-magnitude/)).

## Relevance to SDR

QAM appears in Wi-Fi, cable, cellular, and broadcast rather than scanner voice traffic,
so GopherTrunk — a narrowband digital-voice trunking decoder — does not decode QAM in its
chain. But the same [constellation](/reference/constellation-diagram/) idea is exactly how
an SDR reads modulation quality for *any* scheme: a QAM grid, a QPSK diamond, and a C4FM
four-cluster all live on the same IQ plane, and a smeared or rotated cloud tells the same
story of low SNR or an unlocked carrier loop.

## Sources

[^wiki]: [Quadrature amplitude modulation](https://en.wikipedia.org/wiki/Quadrature_amplitude_modulation) — Wikipedia, for the definition and the higher-order QAM/SNR trade-off.
[^shannon]: [Shannon–Hartley theorem](https://en.wikipedia.org/wiki/Shannon%E2%80%93Hartley_theorem) — Wikipedia, for the capacity limit that ties bits per symbol to required SNR.
