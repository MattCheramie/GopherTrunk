---
slug: carrier-to-noise-ratio
title: Carrier-to-noise ratio (C/N)
entry_type: term
category: rf-metrics
description: Carrier-to-noise ratio is the ratio of modulated carrier power to noise power in the receiver bandwidth, the pre-detection SNR that governs a link before demodulation.
keywords: carrier to noise ratio, C/N, CNR, pre-detection SNR, carrier power, receiver bandwidth, satellite link, C/N0, Eb/N0 relationship
aka: [C/N, CNR, carrier-to-noise ratio]
autolink: true
infobox:
  - { label: Symbol, value: "C/N" }
  - { label: Unit, value: Decibels (dB) }
  - { label: Stage, value: Pre-detection (RF/IF) }
see_also: [signal-to-noise-ratio, eb-n0, bit-error-rate, noise-floor, demodulation]
cite_urls:
  - https://en.wikipedia.org/wiki/Carrier-to-noise_ratio
---

**Carrier-to-noise ratio** (**C/N** or **CNR**) is the ratio, in
[decibels](/reference/decibel/), of the received modulated carrier power to the noise
power within the receiver's bandwidth — the [signal-to-noise ratio](/reference/signal-to-noise-ratio/)
measured *before* [demodulation](/reference/demodulation/).[^wiki] Where plain SNR is
often quoted at baseband after detection, C/N is a **pre-detection** figure taken at RF
or IF, on the carrier as it arrives. It is the natural link-quality metric for systems
where the carrier is measured directly, such as satellite and digital-broadcast
receivers.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 175" role="img" aria-label="A receiver chain: antenna into a bandpass filter and amplifier where carrier-to-noise ratio is measured pre-detection, then a demodulator where post-detection SNR is measured." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="cnar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <path d="M40 40 L40 80 M30 40 L50 40" stroke="currentColor" stroke-width="1.3" fill="none"/>
  <line x1="40" y1="80" x2="80" y2="80" stroke="currentColor" stroke-width="1.3" marker-end="url(#cnar)"/>
  <rect x="85" y="65" width="70" height="30" fill="none" stroke="currentColor" stroke-width="1.3"/>
  <text x="120" y="84" text-anchor="middle" font-size="9" fill="currentColor">RF / IF</text>
  <line x1="155" y1="80" x2="210" y2="80" stroke="currentColor" stroke-width="1.3" marker-end="url(#cnar)"/>
  <text x="182" y="55" text-anchor="middle" font-size="9" fill="currentColor">C/N</text>
  <text x="182" y="68" text-anchor="middle" font-size="8" fill="currentColor">(pre-detect)</text>
  <rect x="215" y="65" width="80" height="30" fill="none" stroke="currentColor" stroke-width="1.3"/>
  <text x="255" y="84" text-anchor="middle" font-size="9" fill="currentColor">demod</text>
  <line x1="295" y1="80" x2="350" y2="80" stroke="currentColor" stroke-width="1.3" marker-end="url(#cnar)"/>
  <text x="380" y="76" text-anchor="middle" font-size="9" fill="currentColor">SNR</text>
  <text x="380" y="90" text-anchor="middle" font-size="8" fill="currentColor">(post)</text>
</svg>
<figcaption>C/N is measured on the modulated carrier before the demodulator; post-detection SNR is measured on the recovered baseband afterward.</figcaption>
</figure>

## How it works

C/N = carrier power − noise power (both in the same units) over the noise-equivalent
bandwidth of the receiver. Because it is defined over the *carrier's* occupied
bandwidth, C/N depends explicitly on that bandwidth: widen the receiver and you admit
more noise, lowering C/N for the same carrier. This bandwidth dependence is exactly why
C/N is not directly comparable between systems — and why engineers convert to
[Eb/N0](/reference/eb-n0/), which normalizes out bandwidth and bit rate, when comparing
modems.

The two are related by C/N = (Eb/N0) × (Rb / B), where Rb is the bit rate and B the
noise bandwidth — the ratio Rb/B being the system's
[spectral efficiency](/reference/spectral-efficiency/). A closely related figure,
**C/N0** (carrier to noise *density*, in dB-Hz), divides carrier power by noise power
*per hertz* rather than over the full bandwidth; it is common in GNSS and satellite
work because it separates the carrier strength from the choice of receiver bandwidth.

For analog FM, C/N above the detector's threshold produces a much higher post-detection
SNR thanks to FM's processing gain; below threshold the familiar "picket-fencing"
breakup sets in. For digital systems, C/N (via Eb/N0) maps onto a
[bit error rate](/reference/bit-error-rate/) through the modulation's waterfall curve.

## In practice

- Satellite TV and [DVB-S](/reference/dvb-s/) receivers report C/N (or the related
  MER/link margin) as the headline link-health indicator, since the carrier is measured
  directly off the transponder.
- [GNSS](/reference/gnss/) receivers report per-satellite C/N0 in dB-Hz — typically
  35–50 dB-Hz for a healthy fix — because bandwidth-independent carrier density is the
  meaningful quantity there.
- A "threshold C/N" is specified for each system: the minimum below which the
  [demodulator](/reference/demodulation/) can no longer maintain lock.

## Relevance to SDR

C/N is the pre-detection language of satellite, digital-TV, and GNSS links, and it is
the quantity a [spectrum analyzer](/reference/spectrum-analyzer/) or
[waterfall display](/reference/waterfall-display/) most directly shows — a carrier
rising above the surrounding [noise floor](/reference/noise-floor/). For a trunking
decoder like [GopherTrunk](/reference/software-defined-radio/), the carrier's C/N as it
arrives at the front end sets the ceiling on everything downstream: the demodulator can
only recover a post-detection SNR and [BER](/reference/bit-error-rate/) as good as the
incoming C/N and the mode's processing gain allow. When GT reports a demod SNR for a
[P25](/reference/p25-phase-1/) or [DMR](/reference/dmr/) channel, it is effectively
translating the arriving carrier-to-noise ratio into a post-detection quality figure —
and no processing can conjure margin the carrier never had.

## Sources

[^wiki]: [Carrier-to-noise ratio](https://en.wikipedia.org/wiki/Carrier-to-noise_ratio) — Wikipedia, definition, bandwidth dependence, and relationship to SNR and Eb/N0.
