---
slug: tinysa
title: TinySA
entry_type: hardware
category: test-equipment
description: "The tinySA is a low-cost, open-source handheld spectrum analyzer covering roughly 100 kHz to 960 MHz, with a signal-generator mode and USB PC control."
keywords: tinySA, tinySA Ultra, handheld spectrum analyzer, low-cost spectrum analyzer, RF spectrum, signal generator mode, open source test equipment, 100 kHz 960 MHz
aka: [tinySA, tinySA Ultra, tiny SA]
autolink: true
infobox:
  - { label: Type, value: Handheld spectrum analyzer }
  - { label: Vendor/Chip, value: "Open source; Si4432 / ADF4351 (Ultra)" }
  - { label: Range, value: "~100 kHz – 960 MHz (Ultra: to ~6 GHz)" }
  - { label: Inputs, value: "Low band + high band" }
  - { label: TX, value: "Yes (built-in signal generator)" }
  - { label: Typical price, value: "$50 – $130" }
see_also: [spectrum-analyzer, nanovna, signal-generator, fast-fourier-transform, frequency-bands, harmonics]
cite_urls:
  - https://www.tinysa.org/wiki/
  - https://en.wikipedia.org/wiki/Spectrum_analyzer
---

**The tinySA** is a low-cost, open-source handheld
[spectrum analyzer](/reference/spectrum-analyzer/) that puts power-versus-frequency
measurement — carrier hunting, [harmonic](/reference/harmonics/) and spurious checks,
occupied-[bandwidth](/reference/bandwidth/) estimates — into a pocket instrument for well
under $100.[^wiki][^home] Designed by Erik Kaashoek as a companion to the
[NanoVNA](/reference/nanovna/), it covers roughly **100 kHz to 960 MHz** on the original
and up to about 6 GHz on the tinySA Ultra, and it doubles as a rudimentary
[signal generator](/reference/signal-generator/).

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="A tinySA handheld schematic: low-band and high-band RF inputs feed a swept receiver whose spectrum trace, a carrier peak above a noise floor, is drawn on a small touchscreen." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="tsar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <text x="55" y="52" font-size="8" fill="currentColor" text-anchor="middle">Low in</text>
  <circle cx="55" cy="62" r="6" fill="none" stroke="currentColor"/>
  <text x="55" y="102" font-size="8" fill="currentColor" text-anchor="middle">High in</text>
  <circle cx="55" cy="92" r="6" fill="none" stroke="currentColor"/>
  <rect x="110" y="58" width="80" height="38" rx="4" fill="none" stroke="currentColor"/>
  <text x="150" y="74" font-size="8" fill="currentColor" text-anchor="middle">Swept</text>
  <text x="150" y="85" font-size="8" fill="currentColor" text-anchor="middle">receiver</text>
  <line x1="61" y1="62" x2="110" y2="70" stroke="currentColor" marker-end="url(#tsar)"/>
  <line x1="61" y1="92" x2="110" y2="84" stroke="currentColor" marker-end="url(#tsar)"/>
  <rect x="250" y="35" width="170" height="80" rx="6" fill="currentColor" fill-opacity="0.08" stroke="currentColor"/>
  <line x1="262" y1="105" x2="410" y2="105" stroke="currentColor" stroke-opacity="0.4"/>
  <polyline points="262,102 320,100 335,55 350,100 380,101 395,80 410,102" fill="none" stroke="currentColor" stroke-width="1.4"/>
  <text x="335" y="130" font-size="8" fill="currentColor" text-anchor="middle">spectrum on display</text>
  <line x1="190" y1="77" x2="250" y2="80" stroke="currentColor" marker-end="url(#tsar)"/>
</svg>
<figcaption>A tinySA: separate low-band and high-band inputs feed a swept receiver, and the resulting power-versus-frequency trace is shown live on the built-in touchscreen.</figcaption>
</figure>

## What it is

The original tinySA is a clever two-path instrument. Its **low band** (≈100 kHz–350 MHz)
uses an Si4432 ISM-radio chip as a genuine narrowband swept
[superheterodyne](/reference/superheterodyne-receiver/) receiver with selectable
resolution bandwidth, giving good dynamic range and a low noise floor. Its **high band**
(≈240–960 MHz) runs the Si4432 as a harmonic mixer, trading some sensitivity for reach.
The **tinySA Ultra** replaces this with an ADF4351-based synthesizer front end that
extends coverage to roughly 6 GHz with better performance. A small touchscreen shows the
trace; USB connects it to PC software for larger displays and logging.

## In practice

The tinySA's honest role is **survey and troubleshooting, not metrology**. It will not
match a bench analyzer's amplitude accuracy, dynamic range, or noise floor, and its input
can be overloaded by strong nearby transmitters — always mind the stated maximum input
level and use an [attenuator](/reference/attenuator/) when probing a live antenna near
high-power sources. Within those limits it is excellent for:

- Confirming a transmitter is on frequency and checking its
  [harmonics](/reference/harmonics/) and spurious output.
- Surveying which frequencies in a [band](/reference/frequency-bands/) are active before
  pointing an SDR at them.
- Estimating relative signal strength and occupied bandwidth.
- Acting as a simple [signal generator](/reference/signal-generator/) source, or — with
  its output — a scalar tracking-generator sweep of a filter.

## Relevance to SDR

For SDR scanning the tinySA is the natural low-cost bench partner to the
[NanoVNA](/reference/nanovna/): where the NanoVNA tunes the antenna and feedline, the
tinySA surveys the spectrum — showing whether a control channel or voice frequency is
actually radiating, how strong it is, and whether interference or a nearby harmonic
threatens the receive band. Any SDR already computes an
[FFT](/reference/fast-fourier-transform/) spectrum, and GopherTrunk's own waterfall serves
the same purpose in software; the tinySA adds a standalone, calibrated-ish view you can
carry to the antenna. GopherTrunk does not interface with a tinySA — it is a separate
handheld aid, not part of the decode chain.

## Sources

[^wiki]: [Spectrum analyzer](https://en.wikipedia.org/wiki/Spectrum_analyzer) — Wikipedia, on swept and FFT spectrum analyzer architectures that the tinySA implements in miniature.
[^home]: [tinySA wiki](https://www.tinysa.org/wiki/) — official documentation for the tinySA and tinySA Ultra hardware, bands, and limits.
