---
slug: mimo
title: MIMO (multiple-input multiple-output)
entry_type: term
category: antennas
description: MIMO uses multiple transmit and receive antennas to multiply link capacity through spatial multiplexing or to harden it through spatial diversity, exploiting a rich multipath channel.
keywords: MIMO, multiple-input multiple-output, spatial multiplexing, spatial diversity, SU-MIMO, MU-MIMO, massive MIMO, spatial streams, Wi-Fi, LTE, 5G NR, antenna array
aka: [MIMO, multiple-input multiple-output]
autolink: true
infobox:
  - { label: Type, value: Multi-antenna technique }
  - { label: Gains, value: Capacity (multiplexing) or robustness (diversity) }
  - { label: Used by, value: Wi-Fi, LTE, 5G NR }
see_also: [beamforming, antenna-diversity, ofdm, multipath-propagation, antenna-gain]
cite_urls:
  - https://en.wikipedia.org/wiki/MIMO
  - https://en.wikipedia.org/wiki/Spatial_multiplexing
---

**MIMO (multiple-input multiple-output)** is a radio technique that places several antennas
at both the transmitter and the receiver so that a single channel carries more data, or
carries it more reliably, than any one antenna pair could.[^wiki] Where a conventional link
has one path, an *M*×*N* MIMO link has *M*×*N* paths between the two arrays, and a rich
[multipath](/reference/multipath-propagation/) environment turns those paths into
independent data pipes rather than a source of fading.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 170" role="img" aria-label="Two transmit antennas on the left connect through a mesh of four propagation paths to two receive antennas on the right, illustrating a two-by-two MIMO channel carrying two spatial streams." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="mimoar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <line x1="60" y1="45" x2="60" y2="70" stroke="currentColor" stroke-width="1.6"/>
  <line x1="60" y1="110" x2="60" y2="135" stroke="currentColor" stroke-width="1.6"/>
  <circle cx="60" cy="45" r="4" fill="currentColor"/><circle cx="60" cy="110" r="4" fill="currentColor"/>
  <text x="60" y="152" text-anchor="middle" font-size="9" fill="currentColor">TX (2)</text>
  <line x1="400" y1="45" x2="400" y2="70" stroke="currentColor" stroke-width="1.6"/>
  <line x1="400" y1="110" x2="400" y2="135" stroke="currentColor" stroke-width="1.6"/>
  <circle cx="400" cy="45" r="4" fill="currentColor"/><circle cx="400" cy="110" r="4" fill="currentColor"/>
  <text x="400" y="152" text-anchor="middle" font-size="9" fill="currentColor">RX (2)</text>
  <line x1="70" y1="48" x2="388" y2="45" stroke="currentColor" stroke-width="0.9" stroke-opacity="0.7" marker-end="url(#mimoar)"/>
  <line x1="70" y1="52" x2="388" y2="107" stroke="currentColor" stroke-width="0.9" stroke-opacity="0.7" marker-end="url(#mimoar)"/>
  <line x1="70" y1="112" x2="388" y2="48" stroke="currentColor" stroke-width="0.9" stroke-opacity="0.7" marker-end="url(#mimoar)"/>
  <line x1="70" y1="114" x2="388" y2="107" stroke="currentColor" stroke-width="0.9" stroke-opacity="0.7" marker-end="url(#mimoar)"/>
  <text x="230" y="15" text-anchor="middle" font-size="9" fill="currentColor">4 paths → 2 independent streams</text>
</svg>
<figcaption>A 2×2 MIMO channel: four propagation paths let the receiver separate two simultaneous data streams sent on the same frequency.</figcaption>
</figure>

## How it works

MIMO exploits the fact that, in a scattering environment, the signal from each transmit
antenna arrives at each receive antenna with a distinct amplitude and phase. The set of these
coefficients forms a *channel matrix* **H**. If **H** is well-conditioned — meaning its paths
are sufficiently independent — the receiver can invert it and recover several data streams that
were transmitted **simultaneously on the same frequency**. This is **spatial multiplexing**, and
the number of separable streams (the *rank* of **H**, bounded by min(*M*,*N*)) sets how many
times capacity is multiplied. Two ideas make it work in practice:

- **Spatial multiplexing** sends independent bit streams from each antenna and relies on **H**
  being invertible. Capacity scales roughly linearly with the number of antenna pairs, at
  constant bandwidth and power — the headline result of MIMO information theory.
- **Spatial diversity** sends the *same* information over multiple antennas (often with a
  space-time code such as Alamouti). Because the paths fade independently, the odds that *all*
  of them are in a deep fade at once are small, so the link stays up. This trades the capacity
  gain for robustness — closely related to receive [antenna diversity](/reference/antenna-diversity/).

A third mode, **beamforming**, uses knowledge of **H** to weight the antennas so their signals
add coherently toward the intended receiver; MIMO and [beamforming](/reference/beamforming/)
are often combined in modern systems.

## Variants

- **SU-MIMO** serves one user with multiple streams. **MU-MIMO** splits the streams among
  several users at once, using the spatial dimension as a shared resource.
- **Massive MIMO** puts dozens to hundreds of elements at the base station, sharpening beams
  and serving many users simultaneously; it is a cornerstone of [5G NR](/reference/5g-nr/).
- MIMO is almost always paired with [OFDM](/reference/ofdm/), so the channel matrix is
  estimated and inverted independently on each flat-fading subcarrier — a "one-tap-per-stream"
  simplification that keeps the equalizer tractable.

## Relevance to SDR

MIMO is standard in Wi-Fi (802.11n and later), [LTE](/reference/lte/), and 5G NR, and it is
what lets a crowded band deliver hundreds of megabits per second without extra spectrum. For a
software-defined radio hobbyist, MIMO shows up mainly on the *analysis* side: multi-channel,
phase-coherent SDRs (for example a KrakenSDR) use several synchronized receivers to do
direction finding and passive radar, which are receive-side cousins of MIMO processing.

**GopherTrunk** is a single-stream trunking decoder for land-mobile protocols
([P25](/reference/p25-phase-1/), [DMR](/reference/dmr/), [NXDN](/reference/nxdn/),
[TETRA](/reference/tetra/)), none of which use MIMO on their traffic channels, so GopherTrunk
does not implement MIMO. Understanding it still matters for context: it explains why the
cellular and Wi-Fi bands a scanner sees are so spectrally dense, and it is the reason a single
whip antenna cannot demodulate an LTE data stream that was designed for a multi-antenna handset.

## Sources

[^wiki]: [MIMO](https://en.wikipedia.org/wiki/MIMO) — Wikipedia, for the definition of MIMO and the distinction between spatial multiplexing and spatial diversity.
