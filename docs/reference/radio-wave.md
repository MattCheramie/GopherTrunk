---
slug: radio-wave
title: Radio wave
entry_type: term
category: rf-fundamentals
description: A radio wave is electromagnetic radiation in the radio frequency range, used to carry information wirelessly by varying its amplitude, frequency, or phase.
keywords: radio wave, electromagnetic radiation, RF, carrier, propagation
aka: [radio wave, radio waves]
autolink: true
infobox:
  - { label: Type, value: Electromagnetic radiation }
  - { label: Frequency range, value: ~3 kHz – 300 GHz }
  - { label: Speed, value: ~299,792,458 m/s (vacuum) }
see_also: [electromagnetic-spectrum, frequency, wavelength, carrier-wave, modulation]
related_lessons:
  - { title: "What is a radio wave?", url: /learn/rf-sdr/radio-waves/ }
external:
  - { title: "Radio wave (Wikipedia)", url: https://en.wikipedia.org/wiki/Radio_wave }
---

A **radio wave** is electromagnetic radiation whose [frequency](/reference/frequency/)
lies in the radio range of the [electromagnetic spectrum](/reference/electromagnetic-spectrum/).
Radio waves travel at the speed of light and carry information wirelessly when their
[amplitude](/reference/amplitude/), frequency, or [phase](/reference/phase/) is varied
— a process called [modulation](/reference/modulation/).

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="A sine wave with one wavelength marked between crests and amplitude marked as height from the centre line." xmlns="http://www.w3.org/2000/svg">
  <line x1="20" y1="80" x2="440" y2="80" stroke="currentColor" stroke-opacity="0.3"/>
  <path d="M20 80 C 60 10, 120 10, 160 80 S 260 150, 300 80 S 400 10, 440 80" fill="none" stroke="currentColor" stroke-width="2.2"/>
  <line x1="160" y1="35" x2="300" y2="35" stroke="currentColor" stroke-dasharray="4 3"/>
  <text x="230" y="28" text-anchor="middle" font-size="12" fill="currentColor">wavelength (λ)</text>
  <line x1="90" y1="80" x2="90" y2="25" stroke="currentColor" stroke-dasharray="4 3"/>
  <text x="98" y="52" font-size="12" fill="currentColor">amplitude</text>
</svg>
<figcaption>A radio wave is described by its wavelength, amplitude, and frequency (cycles per second).</figcaption>
</figure>

## How it works

A transmitter drives alternating current into an [antenna](/reference/antenna/),
radiating a self-propagating electric and magnetic field. A distant antenna converts
the passing field back into a tiny current that a receiver amplifies and decodes.

## Relevance to SDR

Radio waves are the raw input to any receiver. An SDR digitises a slice of them into
[IQ data](/reference/iq-data/) for software to process.
