---
slug: ofdma
title: OFDMA
entry_type: algorithm
category: spread-spectrum
description: OFDMA turns OFDM into a multiple-access scheme by assigning different groups of subcarriers (resource blocks) to different users in each time slot; used in LTE, 5G NR, and Wi-Fi 6.
keywords: OFDMA, orthogonal frequency-division multiple access, resource block, subcarrier allocation, LTE, 5G NR, Wi-Fi 6, 802.11ax, scheduling, resource unit
aka: [OFDMA, orthogonal frequency-division multiple access]
autolink: true
infobox:
  - { label: Type, value: Multiple-access scheme }
  - { label: Assigns, value: Subcarrier groups per user }
  - { label: Used by, value: LTE, 5G NR, Wi-Fi 6 }
see_also: [ofdm, fast-fourier-transform, tdma, fdma, cdma, quadrature-amplitude-modulation]
cite_urls:
  - https://en.wikipedia.org/wiki/Orthogonal_frequency-division_multiple_access
  - https://www.etsi.org/standards
---

**OFDMA (orthogonal frequency-division multiple access)** is [OFDM](/reference/ofdm/) used as
a *multiple-access* method: instead of one transmitter filling every subcarrier, the pool of
subcarriers is partitioned and different **groups of subcarriers** are assigned to different
users in each time slot.[^wiki] It combines the multipath robustness of OFDM with the
flexible, fine-grained sharing of a scheduler — a user can be given many subcarriers for a
short time, or a few for longer, matched to its data need and channel quality.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 160" role="img" aria-label="A time-frequency grid of resource blocks where each block is shaded to show it is allocated to user A, B, or C, so users share both dimensions simultaneously." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" stroke-width="0.7">
    <rect x="45" y="20" width="45" height="24" fill="currentColor" fill-opacity="0.5"/><rect x="90" y="20" width="45" height="24" fill="currentColor" fill-opacity="0.15"/><rect x="135" y="20" width="45" height="24" fill="currentColor" fill-opacity="0.5"/><rect x="180" y="20" width="45" height="24" fill="currentColor" fill-opacity="0.3"/>
    <rect x="45" y="44" width="45" height="24" fill="currentColor" fill-opacity="0.15"/><rect x="90" y="44" width="45" height="24" fill="currentColor" fill-opacity="0.15"/><rect x="135" y="44" width="45" height="24" fill="currentColor" fill-opacity="0.3"/><rect x="180" y="44" width="45" height="24" fill="currentColor" fill-opacity="0.5"/>
    <rect x="45" y="68" width="45" height="24" fill="currentColor" fill-opacity="0.3"/><rect x="90" y="68" width="45" height="24" fill="currentColor" fill-opacity="0.5"/><rect x="135" y="68" width="45" height="24" fill="currentColor" fill-opacity="0.15"/><rect x="180" y="68" width="45" height="24" fill="currentColor" fill-opacity="0.15"/>
  </g>
  <g font-size="9" fill="currentColor">
    <rect x="270" y="24" width="14" height="12" fill="currentColor" fill-opacity="0.5"/><text x="292" y="34">user A</text>
    <rect x="270" y="44" width="14" height="12" fill="currentColor" fill-opacity="0.15"/><text x="292" y="54">user B</text>
    <rect x="270" y="64" width="14" height="12" fill="currentColor" fill-opacity="0.3"/><text x="292" y="74">user C</text>
  </g>
  <text x="130" y="112" text-anchor="middle" font-size="9" fill="currentColor">time (slots) →</text>
  <text x="30" y="56" text-anchor="middle" font-size="9" fill="currentColor" transform="rotate(-90 30 56)">subcarriers ↑</text>
</svg>
<figcaption>OFDMA schedules the time-frequency grid: each resource block is handed to a user, so many users share the band in both time and frequency at once.</figcaption>
</figure>

## How it works

The base station (or Wi-Fi access point) runs the same [IFFT/FFT](/reference/fast-fourier-transform/)
OFDM engine, but a **scheduler** decides, slot by slot, which subcarriers belong to which
user. The smallest schedulable unit is a **resource block** in LTE/5G (12 subcarriers over
one slot) or a **resource unit** in Wi-Fi 6. Because subcarriers stay orthogonal, users'
allocations do not interfere even though they transmit simultaneously in the shared band.

OFDMA is best understood as a hybrid of the classic access schemes: it divides users in
*both* frequency (like [FDMA](/reference/fdma/)) and time (like [TDMA](/reference/tdma/)) on
one grid, giving the scheduler two degrees of freedom. This lets it exploit **frequency-
selective scheduling** — assigning each user the subcarriers where *its* channel happens to
be strong, since a deep fade rarely hits the whole band at once — for a multi-user diversity
gain that single-carrier schemes cannot match.

On the uplink, pure OFDMA's high peak-to-average power ratio is hard on battery-powered
handset amplifiers. LTE therefore uses a precoded variant, **SC-FDMA** (single-carrier FDMA),
which lowers PAPR while keeping the same resource-block structure; 5G NR uses it as an option.

## Relevance to SDR

OFDMA is the air-interface backbone of **4G LTE** and **5G NR** and was added to Wi-Fi in
**802.11ax (Wi-Fi 6)** to let one access point serve many devices efficiently in the same
transmission. It is the natural evolution of OFDM from a point-to-point waveform into a
cellular multiple-access layer, and a peer of [CDMA](/reference/cdma/) as a way for many
users to share spectrum.

None of GopherTrunk's target land-mobile trunking protocols use OFDMA — they rely on FDMA and
TDMA over narrowband single carriers — so the scanner does not decode OFDMA. It is documented
here to complete the multiple-access picture (FDMA, TDMA, CDMA, OFDMA) and to explain how
modern broadband cellular schedules its shared spectrum.

## Sources

[^wiki]: [Orthogonal frequency-division multiple access](https://en.wikipedia.org/wiki/Orthogonal_frequency-division_multiple_access) — Wikipedia, for resource-block assignment, the FDMA/TDMA hybrid view, and SC-FDMA.
