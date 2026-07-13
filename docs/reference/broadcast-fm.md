---
slug: broadcast-fm
title: Broadcast FM
entry_type: technology
category: broadcast
description: "Broadcast FM is the 88–108 MHz frequency-modulated radio band carrying a stereo multiplex with a 19 kHz pilot and a 57 kHz RDS subcarrier."
keywords: broadcast FM, FM radio, 88-108 MHz, FM band, stereo pilot, 19 kHz pilot tone, MPX multiplex, 57 kHz RDS subcarrier, wideband FM, WFM
aka: [FM radio, FM broadcast, WFM, Band II]
autolink: true
infobox:
  - { label: Type, value: Analog broadcast modulation }
  - { label: Band, value: "88–108 MHz (VHF Band II)" }
  - { label: Idea, value: Wideband FM carrying a stereo multiplex }
  - { label: Subcarriers, value: "19 kHz pilot, 38 kHz stereo, 57 kHz RDS" }
see_also: [frequency-modulation, subcarrier, rds, pre-emphasis-de-emphasis, modulation, software-defined-radio]
cite_urls:
  - https://en.wikipedia.org/wiki/FM_broadcasting
  - https://www.itu.int/rec/R-REC-BS.450/en
---

**Broadcast FM** is the analog radio service that occupies the 88–108 MHz VHF band
in most of the world, carrying audio by
[frequency modulation](/reference/frequency-modulation/) of a high-power
carrier.[^wiki] Each station transmits a *composite* baseband signal — the stereo
multiplex, or MPX — in which a mono sum, a stereo difference on a
[subcarrier](/reference/subcarrier/), and low-rate data are stacked below 100 kHz
and then frequency-modulated onto the RF carrier. It is the most familiar wideband
FM signal on the spectrum and a common first target for
[software-defined-radio](/reference/software-defined-radio/) receivers.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 170" role="img" aria-label="The FM stereo multiplex baseband spectrum showing the mono L+R signal from 0 to 15 kHz, a 19 kHz pilot tone, the L-R stereo difference around 38 kHz, and the RDS subcarrier at 57 kHz." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="fma" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <line x1="30" y1="130" x2="440" y2="130" stroke="currentColor" stroke-opacity="0.5" marker-end="url(#fma)"/>
  <text x="435" y="150" text-anchor="end" font-size="9" fill="currentColor">baseband freq (kHz)</text>
  <rect x="34" y="70" width="70" height="60" fill="currentColor" fill-opacity="0.22" stroke="currentColor"/>
  <text x="69" y="105" text-anchor="middle" font-size="8" fill="currentColor">L+R</text>
  <text x="69" y="118" text-anchor="middle" font-size="7" fill="currentColor">0–15</text>
  <line x1="130" y1="55" x2="130" y2="130" stroke="currentColor" stroke-width="1.4"/>
  <text x="130" y="48" text-anchor="middle" font-size="8" fill="currentColor">19 pilot</text>
  <rect x="160" y="85" width="90" height="45" fill="none" stroke="currentColor"/>
  <text x="205" y="108" text-anchor="middle" font-size="8" fill="currentColor">L−R (38)</text>
  <rect x="300" y="98" width="40" height="32" fill="currentColor" fill-opacity="0.35" stroke="currentColor"/>
  <text x="320" y="90" text-anchor="middle" font-size="8" fill="currentColor">RDS 57</text>
</svg>
<figcaption>The FM stereo multiplex: mono sum to 15 kHz, a 19 kHz pilot, the stereo difference at 38 kHz, and RDS data at 57 kHz.</figcaption>
</figure>

## How it works

At the transmitter, the left and right audio channels are matrixed into a *sum*
(L+R) and a *difference* (L−R). The sum occupies 0–15 kHz so that mono receivers
recover full audio simply by low-pass filtering. The difference is amplitude-modulated
(double-sideband suppressed carrier) onto a 38 kHz subcarrier, and a low-level 19 kHz
**pilot tone** — exactly half the subcarrier frequency — is added so the receiver can
regenerate a phase-locked 38 kHz reference to demodulate it. A stereo decoder adds and
subtracts the recovered sum and difference to reconstruct left and right. Above the
stereo pair, a 57 kHz subcarrier — the third harmonic of the pilot — carries
[RDS](/reference/rds/) digital data, and some stations add further subcarriers (SCA)
at 67 or 92 kHz for private audio services.

Before modulation the audio is boosted at high frequencies by
[pre-emphasis](/reference/pre-emphasis-de-emphasis/) (50 µs in ITU regions, 75 µs in
North America); the receiver applies the complementary de-emphasis, trading the FM
noise spectrum's rising high-frequency hiss for an improved signal-to-noise ratio.
The composite signal frequency-modulates the carrier with a peak deviation of ±75 kHz,
and with Carson's-rule bandwidth this gives an occupied width of roughly 180–200 kHz
per station on a 200 kHz raster.

## Relevance to SDR

Broadcast FM is a staple SDR target precisely because its wide deviation makes it
loud and easy to demodulate: an [RTL-SDR](/reference/rtl-sdr/) tuned to a strong local
station and passed through a quadrature FM discriminator produces clear audio with
almost no tuning finesse, which is why "listen to FM" is the canonical first SDR
experiment. Recovering stereo requires locking a 19 kHz pilot loop and demodulating the
38 kHz DSB signal; recovering [RDS](/reference/rds/) requires a further
[subcarrier](/reference/subcarrier/) demodulator and differential decoder.

**GopherTrunk** is a trunked land-mobile scanner (P25, DMR, NXDN, TETRA and similar)
and does not decode broadcast FM audio or RDS. The band is nonetheless relevant as
context: it is a strong, ever-present signal an SDR front end must handle without
overloading, and its multiplex structure is a clean illustration of the subcarrier and
pre-emphasis ideas that recur throughout narrowband land-mobile FM.

## Sources

[^wiki]: [FM broadcasting](https://en.wikipedia.org/wiki/FM_broadcasting) — Wikipedia, for the 88–108 MHz band, the stereo multiplex with 19 kHz pilot and 38 kHz subcarrier, the 57 kHz RDS subcarrier, and ±75 kHz deviation.
