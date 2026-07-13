---
slug: bpsk
title: BPSK
entry_type: technology
category: modulation
description: BPSK (binary phase-shift keying) carries one bit per symbol as a 0 or 180 degree carrier phase; it is the most robust PSK and anchors GPS, telemetry, and PSK31.
keywords: BPSK, binary phase shift keying, phase shift keying, 180 degrees, antipodal, one bit per symbol, GPS, telemetry, PSK31, Costas loop
aka: [BPSK, binary phase-shift keying]
autolink: true
infobox:
  - { label: Type, value: Digital modulation (PSK) }
  - { label: Carries, value: 1 bit per symbol (two phases) }
  - { label: Used by, value: GPS, telemetry, PSK31 }
see_also: [phase-shift-keying, qpsk, constellation-diagram, costas-loop, phase, gps-gnss]
cite_urls:
  - https://en.wikipedia.org/wiki/Phase-shift_keying
  - https://en.wikipedia.org/wiki/Binary_phase-shift_keying
---

**BPSK** (binary phase-shift keying) is the simplest form of
[phase-shift keying](/reference/phase-shift-keying/): it encodes one bit per
[symbol](/reference/symbol-rate/) by transmitting the [carrier](/reference/carrier-wave/)
at one of **two phases 180° apart**, conventionally 0° for a one and 180° for a
zero.[^wiki] The two states are *antipodal* — exact opposites on the
[IQ](/reference/iq-data/) plane — which makes BPSK the most noise-robust of all PSK
schemes, at the cost of the lowest data rate.

<figure class="figure" markdown="0">
<svg viewBox="0 0 300 200" role="img" aria-label="An IQ plane with two BPSK constellation points on the horizontal axis, one at the left labelled bit 0 and one at the right labelled bit 1, separated by 180 degrees." xmlns="http://www.w3.org/2000/svg">
  <line x1="30" y1="100" x2="270" y2="100" stroke="currentColor" stroke-opacity="0.4"/>
  <line x1="150" y1="20" x2="150" y2="180" stroke="currentColor" stroke-opacity="0.4"/>
  <text x="262" y="114" font-size="10" fill="currentColor">I</text><text x="136" y="30" font-size="10" fill="currentColor">Q</text>
  <g fill="currentColor"><circle cx="60" cy="100" r="6"/><circle cx="240" cy="100" r="6"/></g>
  <g font-size="10" fill="currentColor"><text x="46" y="88">0 (180&#176;)</text><text x="210" y="88">1 (0&#176;)</text></g>
  <path d="M60 100 A 90 90 0 0 1 240 100" fill="none" stroke="currentColor" stroke-opacity="0.5" stroke-dasharray="4 3"/>
</svg>
<figcaption>BPSK places two constellation points 180° apart on the I axis; the maximum possible separation gives it the best noise immunity of any PSK.</figcaption>
</figure>

## How it works

A BPSK modulator multiplies the carrier by +1 or −1 according to the data, which flips
the phase by 180°. On the constellation the two points sit at the extremes of the I
axis, as far apart as any two unit-power symbols can be. That maximal distance is why
BPSK needs the least energy per bit of any PSK to hit a given error rate — it is tied
with [QPSK](/reference/qpsk/) for the best bit-error performance, while QPSK doubles the
throughput in the same bandwidth by using both I and Q.

Demodulation requires a **coherent** reference: the receiver must know which way is 0°.
A [Costas loop](/reference/costas-loop/) or squaring loop recovers the carrier phase, but
it resolves the axis only up to a 180° ambiguity — the loop cannot tell 0° from 180° on
its own. Systems solve this with **differential encoding** (BPSK's differential cousin,
DBPSK, encodes bits in phase *changes* so an inverted reference still decodes) or with a
known preamble/unique word that pins down the polarity.

## Relevance to SDR

BPSK is ubiquitous where robustness matters more than raw rate. The
[GPS](/reference/gps-gnss/) L1 C/A signal is BPSK spread by a code; spacecraft and
CubeSat telemetry, beacons, and many low-SNR links use it; and the amateur keyboard mode
[PSK31](/reference/psk31/) is differential BPSK at 31 baud, prized for punching through
noise on crowded HF bands. On a constellation display BPSK shows two clusters on a line;
on a waterfall it is a narrow carrier that, unlike CW, never goes silent.

For GopherTrunk, BPSK is background rather than a decode target — the trunked land-mobile
systems it handles use 4FSK/C4FM and, in P25 Phase 2, a QPSK-family scheme. BPSK is
documented here as the foundation of the PSK family and the reference point for
understanding why higher-order constellations trade noise margin for throughput.

## Sources

[^wiki]: [Phase-shift keying](https://en.wikipedia.org/wiki/Phase-shift_keying) — Wikipedia, for the BPSK definition, its antipodal constellation, and the carrier-recovery ambiguity resolved by differential encoding.
