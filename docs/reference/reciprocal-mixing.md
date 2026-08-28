---
slug: reciprocal-mixing
title: Reciprocal mixing
entry_type: term
category: rf-metrics
description: Reciprocal mixing is receiver self-degradation in which the local oscillator's phase noise convolves onto received signals, raising the in-channel noise floor — a deficit no later DSP stage can remove.
keywords: reciprocal mixing, phase noise, local oscillator, LO phase noise, noise floor, blocking, close-in dynamic range, oscillator quality, mixer, weak signal reception
aka: [reciprocal mixing noise]
autolink: true
infobox:
  - { label: Type, value: Receiver impairment }
  - { label: Cause, value: LO phase noise convolved onto received signals }
  - { label: Signature, value: Carrier-clean but modulation-degraded }
  - { label: Fix, value: A cleaner oscillator — not gain, not DSP }
see_also: [phase-noise, local-oscillator, mixer-rf, intermodulation, desensitization, blocking-dynamic-range, error-vector-magnitude]
cite_urls:
  - https://en.wikipedia.org/wiki/Phase_noise
  - https://en.wikipedia.org/wiki/Superheterodyne_receiver
---

**Reciprocal mixing** is the receiver degrading *itself*: every signal entering the
[mixer](/reference/mixer-rf/) is convolved with the
[local oscillator](/reference/local-oscillator/)'s [phase-noise](/reference/phase-noise/)
skirts, so each received carrier comes out wearing a copy of the LO's noise
pedestal.[^wiki] A strong signal near the tuned channel then splashes noise *into* the
channel — the classic blocking mechanism — but the wanted signal also smears **its own**
modulation with the LO's close-in jitter. Either way the damage is done at the mixer: the
noise is multiplied into the samples, and no amount of filtering, decimation, or clever
demodulation downstream can take it back out.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="A spectrum with a strong nearby signal whose skirts, inherited from the local oscillator's phase noise, spread over the weak wanted channel and raise its floor." xmlns="http://www.w3.org/2000/svg">
  <line x1="30" y1="120" x2="430" y2="120" stroke="currentColor" stroke-opacity="0.4"/>
  <path d="M60 118 C 150 112 200 100 250 60 L 258 30 L 266 60 C 316 100 366 112 430 118" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.2"/>
  <line x1="120" y1="118" x2="120" y2="82" stroke="currentColor" stroke-width="2"/>
  <rect x="100" y="76" width="40" height="44" fill="none" stroke="currentColor" stroke-opacity="0.5" stroke-dasharray="3 3"/>
  <text x="120" y="68" font-size="8.5" fill="currentColor" text-anchor="middle">wanted (weak)</text>
  <text x="258" y="22" font-size="8.5" fill="currentColor" text-anchor="middle">strong neighbour</text>
  <text x="185" y="106" font-size="8" fill="currentColor" text-anchor="middle">LO skirts land here</text>
  <text x="230" y="140" font-size="8.5" fill="currentColor" text-anchor="middle">the neighbour's spread is a mirror of the LO's own phase-noise shape</text>
</svg>
<figcaption>The strong signal is clean on the air; the skirts that bury the weak channel are the receiver's own oscillator noise, transferred onto it in the mixer.</figcaption>
</figure>

## How it works

An ideal LO is a spectral line; mixing with it translates every input frequency crisply. A
real LO carries phase-noise sidebands falling away from the carrier (measured in dBc/Hz at
each offset), and mixing is multiplication in time — convolution in frequency — so every
received signal is smeared by that sideband shape. Two consequences follow:

- **Nearby strong signals raise the floor.** The in-channel noise contributed by a blocker
  at offset Δf is the blocker's power times the LO's phase noise at Δf (integrated over the
  channel). This, not front-end overload, is what limits many receivers'
  close-in [blocking dynamic range](/reference/blocking-dynamic-range/): past a certain
  blocker level, more RF filtering does not help and more gain makes it worse.
- **The wanted signal degrades itself.** Close-in LO jitter rotates the constellation
  symbol-to-symbol. Heavily averaged measurements — a spectrum-analyzer carrier peak, a
  long FFT — can still look clean while
  [EVM](/reference/error-vector-magnitude/) and demodulated SNR are poor. **Carrier-clean
  but modulation-degraded is the reciprocal-mixing signature**, and it is how the effect is
  told apart from [intermodulation](/reference/intermodulation/) or clipping, which show up
  in the amplitude domain.

The oscillator's quality — crystal reference, PLL loop bandwidth, synthesizer architecture,
and in an SDR the sample-clock/PLL configuration at a given rate — sets the effect's size.
Nothing after the mixer can reduce it.

## Relevance to SDR

Reciprocal mixing is the standing explanation for a class of SDR mysteries where a *capture*
is bad in a way no DSP setting changes. GopherTrunk's canonical case (issue #764): the same
TETRA site captured on the same Airspy decoded at ~19.7 dB demod SNR from a 2.5 MS/s
capture but ~9.5 dB from a 10 MS/s capture — neither clipping, and the wideband FFT carrier
SNR actually *higher* at 10 MS/s. Decimating the 10 MS/s file 4:1 with an independent
resampler and replaying it through the proven 2.5 MS/s path reproduced the same ~9.5 dB,
proving the ~10 dB deficit was baked into the captured samples: front-end phase noise at the
device's native 10 MS/s clock, not the decoder
(pinned by `TestDownconverterSNRInvariantAcrossRate` in
`internal/scanner/ccdecoder/ddc_highrate_test.go`). The practical rules that follow: prefer
the sample rate at which your hardware's clocking is clean, verify a "weak signal" problem
against a capture at a different rate before blaming the decoder, and treat
carrier-clean-but-modulation-degraded as an oscillator finding — a
[gain](/reference/sdr-gain-overload/) change will not fix it.

## Sources

[^wiki]: [Phase noise](https://en.wikipedia.org/wiki/Phase_noise) — Wikipedia, on oscillator phase-noise sidebands and their transfer onto received signals in mixing (reciprocal mixing).
