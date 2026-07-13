---
slug: double-sideband
title: Double sideband (DSB)
entry_type: technology
category: modulation
description: "Double sideband (DSB) is amplitude modulation that sends both mirror-image sidebands, either with the full carrier (DSB-AM) or with it suppressed (DSB-SC)."
keywords: double sideband, DSB, DSB-SC, DSB-AM, suppressed carrier, sidebands, amplitude modulation, product detector
aka: [double sideband, DSB, DSB-SC, DSB-AM]
autolink: true
infobox:
  - { label: Type, value: Analog modulation (AM family) }
  - { label: Sends, value: Both sidebands }
  - { label: Examples, value: AM broadcast, suppressed-carrier subcarriers }
see_also: [amplitude-modulation, single-sideband, vestigial-sideband, carrier-wave, modulation]
cite_urls:
  - https://en.wikipedia.org/wiki/Double-sideband_suppressed-carrier_transmission
  - https://en.wikipedia.org/wiki/Sideband
---

**Double sideband** (**DSB**) is any form of [amplitude modulation](/reference/amplitude-modulation/)
that transmits **both** of the two mirror-image sidebands produced when a message
multiplies a [carrier](/reference/carrier-wave/).[^wiki] Because the upper and lower
sidebands each carry a complete copy of the baseband signal, DSB occupies twice the
bandwidth of the message and — in its plain form — spends power on redundant information.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 130" role="img" aria-label="A DSB spectrum showing a central carrier line flanked by a lower and an upper sideband of equal size." xmlns="http://www.w3.org/2000/svg">
  <line x1="30" y1="100" x2="430" y2="100" stroke="currentColor" stroke-opacity="0.4"/>
  <line x1="230" y1="100" x2="230" y2="35" stroke="currentColor" stroke-width="1.6"/>
  <text x="230" y="28" text-anchor="middle" font-size="9" fill="currentColor">carrier</text>
  <rect x="150" y="60" width="55" height="40" fill="currentColor" fill-opacity="0.22"/>
  <rect x="255" y="60" width="55" height="40" fill="currentColor" fill-opacity="0.22"/>
  <text x="177" y="118" text-anchor="middle" font-size="9" fill="currentColor">lower sideband</text>
  <text x="283" y="118" text-anchor="middle" font-size="9" fill="currentColor">upper sideband</text>
  <line x1="150" y1="46" x2="310" y2="46" stroke="currentColor" stroke-opacity="0.5" marker-start="url(#dsbar)" marker-end="url(#dsbar)"/>
  <text x="230" y="42" text-anchor="middle" font-size="8" fill="currentColor">2 × message bandwidth</text>
  <defs><marker id="dsbar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>DSB keeps both sidebands; the two are mirror images, so each already contains the full message.</figcaption>
</figure>

## How it works

Multiplying a message *m(t)* by a carrier cos(2πf꜀t) shifts the message spectrum up to
±f꜀, producing a lower sideband (the frequency-inverted copy below the carrier) and an
upper sideband (the erect copy above it). Ordinary broadcast AM is **DSB with a large
carrier** (DSB-AM or DSB-LC): the carrier is left in place so a cheap envelope detector
can recover the audio without regenerating a reference. If instead the message is
multiplied directly with no DC offset, the carrier cancels and the result is
**double-sideband suppressed-carrier** (DSB-SC) — the same two sidebands but with the
power-hungry carrier gone.

Suppressing the carrier saves a great deal of transmit power (in broadcast AM the carrier
carries at least two-thirds of the total), but it breaks envelope detection. DSB-SC must
be demodulated with a **product detector**: the receiver multiplies the incoming signal
by a locally generated carrier of the exact same frequency and phase, then low-pass
filters. Getting that local carrier phase-coherent usually needs a
[phase-locked loop](/reference/phase-locked-loop/) or a Costas loop, because a small
phase error attenuates the recovered audio by cos φ (and a 90° error nulls it entirely).

The redundancy is easy to see in the spectrum. Both sidebands are exact mirror images of
one another about the carrier, so either one alone already contains every bit of the
message. DSB therefore spends twice the bandwidth of the baseband and, in the carrier
form, most of its power on things the receiver does not strictly need — which is precisely
the inefficiency that [single sideband](/reference/single-sideband/) and
[vestigial sideband](/reference/vestigial-sideband/) were invented to remove. DSB survives
where that inefficiency buys real simplicity: an envelope detector for DSB-AM is a diode
and a capacitor, cheap enough to put in every pocket radio ever made.

## Variants

- **DSB-AM (DSB-LC)** — full carrier retained; simple envelope detection; the everyday
  AM broadcast signal. Its two sidebands are the reason a shaped
  [single-sideband](/reference/single-sideband/) transmitter can throw one away.
- **DSB-SC** — carrier suppressed; roughly 6 dB more efficient use of transmit power for
  the same sideband strength, at the cost of coherent detection. It is the building block
  of analog color TV chroma, FM stereo difference channels, and many
  [subcarrier](/reference/subcarrier/) schemes.
- **Quadrature multiplexing** — because a DSB-SC signal uses the full ±f꜀ band, two
  independent DSB-SC messages can share it in phase quadrature (sine and cosine carriers),
  the analog ancestor of I/Q modulation.

## Relevance to SDR

DSB is more a conceptual anchor than a mode you tune every day. Standard AM broadcast is
DSB-AM, and a software receiver can demodulate it either by envelope tracking or, better,
by phase-locking to the carrier and doing synchronous DSB detection for lower distortion.
DSB-SC itself appears inside composite signals: the 38 kHz stereo difference channel of
[broadcast FM](/reference/broadcast-fm/) is DSB-SC around a suppressed subcarrier that the
receiver regenerates from the 19 kHz pilot. Understanding DSB is also the natural stepping
stone to [single sideband](/reference/single-sideband/) and
[vestigial sideband](/reference/vestigial-sideband/), which start from the two-sideband
picture and remove or trim what DSB keeps. GopherTrunk targets digital trunking modes, so
it does not demodulate analog DSB voice, but the same product-detector and carrier-recovery
ideas underlie the coherent I/Q processing it uses on digital carriers.

## In practice

The efficiency gap is stark in numbers. A DSB-AM broadcast transmitter at 100% modulation
puts two-thirds of its power in the carrier and only one-sixth in each sideband, so a
1 kW station radiates barely 167 W of useful sideband energy per sideband; the rest heats
the antenna as an unmodulated reference tone. DSB-SC recovers that carrier power for the
sidebands, and SSB recovers the redundant-sideband power on top of it — which is why a
100 W SSB signal reaches as far as a kilowatt-class DSB-AM one. The trade the designer is
buying with DSB is not efficiency but receiver simplicity and tuning tolerance: an envelope
detector needs no frequency reference and does not care about a modest tuning error, whereas
a DSB-SC product detector must reconstruct the carrier's frequency *and* phase or the audio
fades and distorts.

## Sources

[^wiki]: [Double-sideband suppressed-carrier transmission](https://en.wikipedia.org/wiki/Double-sideband_suppressed-carrier_transmission) — Wikipedia, for the DSB/DSB-SC definitions and coherent detection requirement.
