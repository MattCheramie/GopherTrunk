---
slug: low-if
title: Low-IF (low intermediate frequency)
entry_type: term
category: sdr-dsp
description: "Low-IF places the wanted signal at a small nonzero intermediate frequency, dodging zero-IF's DC and flicker-noise problems at the cost of an image to reject."
keywords: low-IF, low intermediate frequency, near-zero IF, receiver architecture, DC offset avoidance, flicker noise, image rejection, complex IF
aka: [low intermediate frequency, near-zero IF]
autolink: true
infobox:
  - { label: Type, value: Receiver front-end scheme }
  - { label: IF, value: Small nonzero (e.g. tens–hundreds of kHz) }
  - { label: Cost, value: Needs image rejection }
see_also: [zero-if, image-rejection, direct-conversion-receiver, dc-offset, intermediate-frequency, superheterodyne-receiver]
cite_urls:
  - https://en.wikipedia.org/wiki/Low_IF_receiver
  - https://en.wikipedia.org/wiki/Intermediate_frequency
---

A **low-IF** (**low intermediate frequency**) receiver mixes the wanted signal down to a
**small, nonzero** [intermediate frequency](/reference/intermediate-frequency/) — often
just one channel width away from zero — instead of all the way to
[baseband](/reference/baseband/).[^wiki] This tiny offset is enough to move the signal off
0 Hz, where [DC offset](/reference/dc-offset/) and flicker noise would otherwise corrupt
it, while keeping almost all the integration and simplicity of the
[zero-IF](/reference/zero-if/) approach. It sits between full direct conversion and the
classic [superheterodyne](/reference/superheterodyne-receiver/) receiver.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="A baseband spectrum in which the wanted channel is offset to a small positive intermediate frequency away from zero hertz, and a mirror-image band sits at the corresponding negative frequency waiting to be rejected." xmlns="http://www.w3.org/2000/svg">
  <line x1="30" y1="110" x2="440" y2="110" stroke="currentColor"/>
  <line x1="150" y1="45" x2="150" y2="118" stroke="currentColor" stroke-dasharray="3 3" stroke-opacity="0.6"/>
  <text x="138" y="132" font-size="9" fill="currentColor">0 Hz</text>
  <path d="M210 110 Q255 55 300 110 Z" fill="currentColor" fill-opacity="0.2" stroke="currentColor"/>
  <text x="215" y="45" font-size="8" fill="currentColor">wanted (+IF)</text>
  <path d="M60 110 Q95 82 130 110 Z" fill="currentColor" fill-opacity="0.08" stroke="currentColor" stroke-dasharray="2 2"/>
  <text x="55" y="78" font-size="8" fill="currentColor">image (−IF)</text>
  <text x="240" y="132" font-size="8" fill="currentColor">+IF</text>
</svg>
<figcaption>Low-IF shifts the signal to a small positive IF, clear of the DC spike; the price is a mirror image at −IF that must be rejected.</figcaption>
</figure>

## How it works

Rather than tuning the [local oscillator](/reference/local-oscillator/) to the carrier
exactly, a low-IF receiver offsets it by a modest amount so the signal lands at a small
IF — anything from a few kilohertz to a few hundred kilohertz, typically comparable to one
channel bandwidth. That offset is chosen large enough to escape the DC region and the
1/f (flicker) noise that plagues low-cost mixers, but small enough that a slow ADC can
still capture it. A final digital [down-converter](/reference/digital-down-converter/)
then shifts the channel to true baseband for demodulation.

The catch is the **image**. Any real or complex mixer with imperfect quadrature lets a
signal at the mirror frequency (an equal offset on the *other* side of the oscillator)
fold onto the wanted IF. In zero-IF the image of a signal is its own mirror around DC; in
low-IF the image is a *separate, potentially strong* neighbouring channel, so it must be
suppressed deliberately.

## In practice

Low-IF designs lean on [image rejection](/reference/image-rejection/): a well-balanced
complex (I/Q) mixer, plus complex bandpass filtering in analog or digital form, cancels
the mirror band. The achievable image suppression depends directly on how well the I and
Q paths are matched — the same [IQ imbalance](/reference/iq-imbalance/) that produces
mirror artefacts in zero-IF sets the image-rejection ceiling here. Because the residual
DC spike now falls *outside* the wanted channel, it can simply be filtered away, which is
the whole point of the scheme.

## Relevance to SDR

Low-IF is common in integrated broadcast and communications receiver chips — many
FM/DAB, Bluetooth, and cellular front ends use it — because it delivers direct
conversion's small size while avoiding the DC headache. Some SDR tuners can be driven with
a deliberate frequency offset for exactly this reason: place the tuner slightly off the
signal so the strong center DC spike does not sit on the carrier, then correct the offset
in software. Operators do this by hand when a station of interest lands on the middle
spike of a [zero-IF](/reference/zero-if/) display.

GopherTrunk itself does its channel selection with a software down-converter, so it can
tune a signal that a user has intentionally placed at a low IF and shift it back to
baseband before decoding — the same trick that keeps a wanted carrier off the receiver's
DC artefact.

## Sources

[^wiki]: [Low IF receiver](https://en.wikipedia.org/wiki/Low_IF_receiver) — Wikipedia, on placing the signal at a small nonzero IF to avoid DC/flicker while requiring image rejection.
