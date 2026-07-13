---
slug: direct-sequence-spread-spectrum
title: Direct-sequence spread spectrum (DSSS)
entry_type: algorithm
category: spread-spectrum
description: DSSS multiplies each data bit by a fast pseudo-random chip sequence to spread it over a wide bandwidth, trading bandwidth for processing gain, interference rejection, and low probability of intercept.
keywords: direct-sequence spread spectrum, DSSS, chip sequence, PN code, processing gain, spreading code, GPS, 802.11b, CDMA, despreading, correlation
aka: [DSSS, direct sequence]
autolink: true
infobox:
  - { label: Type, value: Spread-spectrum modulation }
  - { label: Spreads via, value: High-rate PN chip multiply }
  - { label: Used by, value: GPS, 802.11b, CDMA }
see_also: [maximal-length-sequence, gold-code, barker-code, cdma, matched-filter, scrambling]
cite_urls:
  - https://en.wikipedia.org/wiki/Direct-sequence_spread_spectrum
  - https://www.gps.gov/technical/icwg/
---

**Direct-sequence spread spectrum (DSSS)** multiplies each data symbol by a much faster
pseudo-random **chip sequence**, spreading the signal's energy across a bandwidth many times
wider than the data rate would require.[^wiki] Because the receiver knows the exact code, it
can *despread* the wanted signal back to a narrow band while smearing any interferer out
into the noise — buying interference rejection, a **low probability of intercept (LPI)**,
and multi-user separation from what looks, on a spectrum analyzer, like a raised noise floor.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="A slow data bit multiplied by a fast chip sequence produces a wideband chip stream; at the receiver the same code despreads it back to the narrow data bit." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="dsssar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <g stroke="currentColor" stroke-width="1.2" fill="none">
    <path d="M20 30 H120 V30"/>
    <path d="M20 30 L20 30 M20 30 H120"/>
    <path d="M150 20 h6 v20 h6 v-20 h6 v20 h6 v-20 h6 v20 h6 v-20 h6 v20 h6 v-20 h6 v20 h6 v-20"/>
    <path d="M290 25 h9 v10 h9 v-10 h9 v10 h9 v-10 h9 v10 h9 v-10"/>
  </g>
  <text x="70" y="55" text-anchor="middle" font-size="9" fill="currentColor">1 data bit (slow)</text>
  <text x="205" y="55" text-anchor="middle" font-size="9" fill="currentColor">× chip code (fast) = wideband</text>
  <text x="335" y="52" text-anchor="middle" font-size="9" fill="currentColor">despread → data</text>
  <g stroke="currentColor" stroke-width="1" marker-end="url(#dsssar)"><path d="M124 30 H144"/><path d="M262 30 H286"/></g>
  <text x="230" y="120" text-anchor="middle" font-size="9" fill="currentColor">Processing gain Gp = chip rate / bit rate</text>
</svg>
<figcaption>DSSS trades bandwidth for processing gain: a slow bit becomes a fast chip stream, and only the matching code despreads it.</figcaption>
</figure>

## How it works

Each information bit is XORed (or BPSK-multiplied, mapping 0/1 to +1/−1) with a
**spreading code** running at the *chip rate*, which is an integer multiple of the bit rate.
A single bit therefore becomes a burst of many chips. The transmitted bandwidth grows in
proportion to the chip rate, so the same energy is spread thinner in frequency — often
below the ambient noise floor.

At the receiver, the incoming samples are multiplied by a locally generated *replica* of the
same code, aligned in time. For the wanted signal the code multiplies by itself
(±1 × ±1 = +1 everywhere), so its chips coherently add back up into full-amplitude data
symbols — this is a [matched-filter](/reference/matched-filter/)/correlation operation. Any
signal *not* correlated with the code — narrowband jammers, thermal noise, other users'
codes — gets multiplied by the pseudo-random pattern and spread out, so only a small
fraction of its power lands in the despread bandwidth.

The key figure of merit is **processing gain**, Gp = chip rate ÷ data rate (often quoted in
dB as 10·log₁₀ Gp). It quantifies how much the despreading lifts the wanted signal above
interference: a 1023-chip GPS C/A code over a 50 bit/s data stream is roughly 43 dB of gain,
which is why GPS works from signals ~20 dB *below* the noise floor.

## In practice

- **Code choice matters.** The spreading sequence needs a sharp autocorrelation peak (so
  timing alignment is unambiguous) and, for multi-user systems, low cross-correlation with
  other codes. [Maximal-length sequences](/reference/maximal-length-sequence/),
  [Gold codes](/reference/gold-code/), and short
  [Barker codes](/reference/barker-code/) are the common families.
- **Acquisition and tracking.** The receiver must first *find* the code phase (a search over
  chip offsets) and then keep the replica aligned as Doppler and clock drift move it — the
  classic acquisition-then-tracking loop of a GPS or CDMA receiver.
- **Near-far problem.** A strong nearby transmitter can overwhelm a weak far one even after
  despreading, which is why [CDMA](/reference/cdma/) cellular systems add tight power control.

## Relevance to SDR

DSSS underpins several everyday RF systems: **GPS** and other GNSS civil signals, the
original **802.11b** Wi-Fi (Barker-coded 1–2 Mbit/s and CCK), and **IS-95 / cdmaOne / UMTS**
cellular via [CDMA](/reference/cdma/). Military links use it for LPI/anti-jam. It is closely
related to [scrambling](/reference/scrambling/), which whitens data with a PN sequence but
without the bandwidth expansion.

None of GopherTrunk's target land-mobile trunking protocols (P25, DMR, NXDN, TETRA) use DSSS
— they are narrowband FDMA/TDMA voice systems — so GopherTrunk does not implement a
despreading correlator. DSSS is documented here as the foundational spread-spectrum
technique that GNSS and cellular receivers depend on, and as context for the code families
(m-sequences, Gold, Barker) that the scanner *does* touch elsewhere.

## Sources

[^wiki]: [Direct-sequence spread spectrum](https://en.wikipedia.org/wiki/Direct-sequence_spread_spectrum) — Wikipedia, for the chip-multiply mechanism, processing gain, and despreading.
