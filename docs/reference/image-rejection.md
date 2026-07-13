---
slug: image-rejection
title: Image rejection
entry_type: term
category: sdr-dsp
description: "Image rejection suppresses the unwanted mirror frequency that a mixer folds onto the wanted signal, using filtering or quadrature image-reject mixers like Hartley and Weaver."
keywords: image rejection, image-reject mixer, Hartley architecture, Weaver architecture, IMRR, image rejection ratio, quadrature mixer, mirror frequency, phasing
aka: [image suppression, image-reject mixing]
autolink: true
infobox:
  - { label: Type, value: Receiver design technique }
  - { label: Targets, value: The mixer image frequency }
  - { label: Methods, value: Filtering, Hartley, Weaver }
see_also: [image-frequency, superheterodyne-receiver, hilbert-transform, mixer-rf, iq-imbalance, low-if]
cite_urls:
  - https://en.wikipedia.org/wiki/Image_response
  - https://en.wikipedia.org/wiki/Heterodyne
---

**Image rejection** is the suppression of the unwanted **image frequency** — the second
input frequency a [mixer](/reference/mixer-rf/) folds onto the same output as the wanted
signal.[^wiki] Any mixer converts *two* input frequencies, symmetric about the
[local oscillator](/reference/local-oscillator/), down to the same
[intermediate frequency](/reference/intermediate-frequency/); unless one of them is
removed, a signal at the [image frequency](/reference/image-frequency/) interferes with the
one you want. Image rejection is how a receiver keeps the image out. Its effectiveness is
quoted as the **image-rejection ratio** (IMRR), the dB by which the image is attenuated
relative to the wanted signal.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="A frequency axis with the local oscillator in the centre, the wanted signal one IF above it, and the image signal an equal IF below it; the wanted signal passes while the image is cancelled." xmlns="http://www.w3.org/2000/svg">
  <line x1="30" y1="100" x2="440" y2="100" stroke="currentColor"/>
  <line x1="235" y1="55" x2="235" y2="108" stroke="currentColor" stroke-dasharray="3 3" stroke-opacity="0.6"/>
  <text x="215" y="122" font-size="8" fill="currentColor">LO</text>
  <path d="M320 100 Q345 60 370 100 Z" fill="currentColor" fill-opacity="0.28" stroke="currentColor"/>
  <text x="315" y="52" font-size="8" fill="currentColor">wanted</text>
  <path d="M100 100 Q125 70 150 100 Z" fill="currentColor" fill-opacity="0.1" stroke="currentColor" stroke-dasharray="2 2"/>
  <text x="100" y="66" font-size="8" fill="currentColor">image</text>
  <text x="152" y="90" font-size="14" fill="currentColor">✗</text>
  <path d="M175 45 L 300 45" fill="none" stroke="currentColor" stroke-opacity="0.4"/>
  <text x="190" y="40" font-size="7" fill="currentColor">equal IF each side of LO</text>
</svg>
<figcaption>The wanted signal and its image sit symmetrically around the local oscillator; image rejection lets the wanted one through and cancels the mirror.</figcaption>
</figure>

## How it works

There are two broad strategies.

**Filtering (image-reject filter).** In a [superheterodyne](/reference/superheterodyne-receiver/)
receiver the image is separated from the wanted signal by twice the IF. A bandpass filter
ahead of the mixer, tuned to the wanted band, attenuates the image before mixing. The
higher the IF, the further away the image and the easier the filter — the classic reason
superhets use a high first IF.

**Phasing (image-reject mixers).** Instead of filtering, these architectures use
quadrature signals so the image cancels by interference:

- **Hartley.** The signal is mixed with two [local oscillator](/reference/local-oscillator/)
  phases 90° apart, one branch is phase-shifted a further 90° (a
  [Hilbert transform](/reference/hilbert-transform/) / broadband 90° network), and the two
  are summed. The wanted signal adds in phase while the image cancels.
- **Weaver.** Replaces the troublesome broadband 90° phase-shift network with a second
  pair of mixers at a low frequency, achieving the same cancellation with two mixing
  stages and no wideband Hilbert network.

Both phasing methods are really doing complex (I/Q) signal processing: keeping the sign of
the frequency offset, which a single real mixer discards, so the two sides of the
oscillator stay distinguishable.

## In practice

Real image rejection is finite. Filter designs are limited by filter sharpness; phasing
designs are limited by how precisely the 90° phase and the branch gains are matched — the
same [IQ imbalance](/reference/iq-imbalance/) that produces mirror artefacts in a
zero-IF receiver caps the IMRR of a Hartley or Weaver mixer, typically at 30–40 dB
uncorrected and much higher after calibration. Modern receivers often combine a modest
image-reject mixer with digital correction that estimates and cancels the residual
imbalance adaptively.

## Relevance to SDR

Image rejection is central to how SDR front ends and their tuners are built. The quadrature
tuners in [RTL-SDR](/reference/rtl-sdr/) dongles and other zero-/low-IF SDRs are
image-reject mixers, and their residual imbalance is exactly what leaves faint mirror-image
signals on the [waterfall](/reference/waterfall-display/). Many SDR programs apply digital
IQ-correction to push those images down, and [low-IF](/reference/low-if/) tuner modes lean
directly on image rejection to keep a neighbouring channel from folding onto the wanted
one.

GopherTrunk works on the corrected IQ its source provides; it does not build mixers, but
the images that survive imperfect rejection are among the artefacts its channel filtering
must tolerate, and the concept explains why a strong station can appear mirrored across the
tuned frequency.

## Sources

[^wiki]: [Image response](https://en.wikipedia.org/wiki/Image_response) — Wikipedia, on the mixer image and the filtering and quadrature techniques used to reject it.
