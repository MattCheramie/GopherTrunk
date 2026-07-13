---
slug: low-noise-amplifier
title: Low-noise amplifier (LNA)
entry_type: hardware
category: rf-front-end
description: A low-noise amplifier boosts a weak antenna signal early in the receive chain with minimal added noise, setting much of a receiver's sensitivity.
keywords: LNA, low noise amplifier, noise figure, sensitivity, front end, preamp, preamplifier, Friis, cascade noise, mast-mounted amplifier
aka: [low-noise amplifier, LNA]
autolink: true
infobox:
  - { label: Type, value: RF amplifier }
  - { label: Placed, value: Early in receive chain (near antenna) }
  - { label: Key spec, value: Noise figure }
see_also: [noise-figure, preamplifier, signal-to-noise-ratio, superheterodyne-receiver, bias-tee, antenna]
related_lessons:
  - { title: "How an SDR receiver works", url: /learn/rf-sdr/sdr-receiver/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Low-noise_amplifier
  - https://en.wikipedia.org/wiki/Friis_formulas_for_noise
---

A **low-noise amplifier** (**LNA**) boosts a weak [antenna](/reference/antenna/) signal
**early** in the [receive chain](/reference/superheterodyne-receiver/), adding as little
noise as possible.[^wiki] Because every later stage contributes its own noise, amplifying
the signal first — before cable and mixer losses eat into it — preserves the
[signal-to-noise ratio](/reference/signal-to-noise-ratio/) and largely fixes the whole
receiver's sensitivity. An LNA is a specific kind of [preamplifier](/reference/preamplifier/):
one designed so that its own [noise figure](/reference/noise-figure/) is as low as the gain
it provides is useful.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 110" role="img" aria-label="Antenna into a low-noise amplifier early in the chain, boosting the weak signal before lossy cable and later stages add noise." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <path d="M35 44 v-16 m-7 0 l7 -10 l7 10" fill="none" stroke="currentColor" stroke-width="1.5"/>
    <text x="35" y="60">antenna</text>
    <path d="M120 38 L120 68 L160 53 Z" fill="none" stroke="currentColor" stroke-width="1.4"/><text x="135" y="86" font-size="8">LNA</text>
    <rect x="210" y="38" width="80" height="30" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="250" y="57">receiver</text>
    <g stroke="currentColor" stroke-width="1.1"><line x1="35" y1="44" x2="35" y2="53"/><line x1="35" y1="53" x2="119" y2="53"/><line x1="160" y1="53" x2="209" y2="53"/></g>
    <text x="200" y="86" font-size="7.5">lossy coax</text>
    <text x="360" y="49" font-size="8">boost early =</text><text x="360" y="61" font-size="8">best sensitivity</text>
  </g>
</svg>
<figcaption>A low-noise amplifier boosts the faint antenna signal early — before lossy coax and the mixer — so it sets the receiver's sensitivity.</figcaption>
</figure>

## How it works

The reason placement matters is captured by the **Friis noise formula** for a chain of
stages.[^friis] The total noise factor is

`F_total = F1 + (F2 − 1)/G1 + (F3 − 1)/(G1·G2) + …`

The first stage's noise factor `F1` passes through untouched, but every later stage's noise
contribution is *divided by the gain that precedes it*. Give the first stage enough gain
`G1` and its low `F1` dominates: the noise of the coax, the mixer, and the SDR's own
[ADC](/reference/analog-to-digital-converter/) are suppressed by that gain. This is why the
LNA belongs at the antenna, ahead of the feedline, not bolted to the radio. Its
[noise figure](/reference/noise-figure/) — noise factor expressed in dB — is the single
number that most determines how weak a signal the receiver can still recover.

An LNA is not free gain, though. It also amplifies every strong signal in its passband, and
its output devices have a finite [1 dB compression point](/reference/1-db-compression-point/)
and [third-order intercept](/reference/third-order-intercept/). Drive it too hard and it
generates [intermodulation](/reference/intermodulation/) products that can bury the wanted
signal — so a "better" LNA is often one with *lower* gain but higher linearity.

## Variants

- **Wideband LNA** — flat gain over a broad range (e.g. 20 MHz–2 GHz). Simple and cheap,
  but it amplifies out-of-band offenders too, so it is usually paired with an
  [RF filter](/reference/rf-filter/) or [SAW filter](/reference/saw-filter/) preselector.
- **Band-specific / filtered LNA** — an integrated filter plus amplifier for one service
  (1090 MHz ADS-B, the 137 MHz weather-satellite band). The filter protects the amplifier
  from overload and is the more robust choice near strong transmitters.
- **Mast-mounted amplifier (MMA)** — a weatherproof LNA at the antenna, fed DC up the coax
  by a [bias tee](/reference/bias-tee/) so no separate power cable is needed.
- **GaAs / GaN pHEMT** devices dominate modern low-noise designs; noise figures below
  1 dB are routine at VHF/UHF.

## In practice

Add an LNA only when the front end is genuinely noise-limited — long or lossy coax, a
weak-signal target, or a masthead install. In a strong-signal urban environment an
un-filtered LNA usually makes reception *worse* by pushing the SDR into overload; there,
reach for a filtered LNA or an [attenuator](/reference/attenuator/) instead. Because the
[RTL-SDR](/reference/rtl-sdr/) and similar tuners already have modest noise figures, the
LNA's biggest wins come on the highest bands and longest runs, where cable loss would
otherwise dominate the budget.

## Relevance to SDR

An antenna-mounted LNA can meaningfully improve reception of weak signals on hobby SDRs —
ADS-B at 1090 MHz, L-band satellite, and long UHF runs are classic cases — provided you
guard against overload from strong nearby transmitters. GopherTrunk is downstream of the
front end: it decodes whatever [IQ](/reference/iq-data/) the SDR delivers and does not
control the LNA, but a well-chosen LNA raises the in-channel SNR that GopherTrunk's
demodulators see, which is the difference between a locked control channel and a dropped
one. See the DSP notes on rate-invariance — the decode path is only as good as the samples
handed to it, and the LNA is where that quality is won or lost.

## Sources

[^wiki]: [Low-noise amplifier](https://en.wikipedia.org/wiki/Low-noise_amplifier) — Wikipedia, on LNAs, noise figure, and their role in setting receiver sensitivity.
[^friis]: [Friis formulas for noise](https://en.wikipedia.org/wiki/Friis_formulas_for_noise) — Wikipedia, deriving how each cascaded stage's noise is divided by preceding gain, the reason the LNA goes first.
