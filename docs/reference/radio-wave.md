---
slug: radio-wave
title: Radio wave
entry_type: term
category: rf-fundamentals
description: A radio wave is electromagnetic radiation in the radio frequency range, used to carry information wirelessly by varying its amplitude, frequency, or phase.
keywords: radio wave, electromagnetic radiation, RF, carrier, propagation, radio frequency, wireless
aka: [radio wave, radio waves]
autolink: true
infobox:
  - { label: Type, value: Electromagnetic radiation }
  - { label: Frequency range, value: ~3 kHz – 300 GHz }
  - { label: Speed, value: ~299,792,458 m/s (vacuum) }
see_also: [electromagnetic-spectrum, frequency, wavelength, carrier-wave, modulation, antenna]
related_lessons:
  - { title: "What is a radio wave?", url: /learn/rf-sdr/radio-waves/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Radio_wave
  - https://www.itu.int/rec/R-REC-V.431/en
---

A **radio wave** is electromagnetic radiation whose [frequency](/reference/frequency/)
lies in the radio range of the
[electromagnetic spectrum](/reference/electromagnetic-spectrum/), conventionally about
3 kHz to 300 GHz.[^wiki] Radio waves travel at the speed of light and carry information
wirelessly when their [amplitude](/reference/amplitude/), frequency, or
[phase](/reference/phase/) is varied — a process called
[modulation](/reference/modulation/). They are the medium every radio system, from an AM
broadcast to a trunked police network, uses to move a message through empty space.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="A sine wave with one wavelength marked between crests and amplitude marked as height from the centre line, representing a radio wave." xmlns="http://www.w3.org/2000/svg">
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

A transmitter drives an alternating current into an [antenna](/reference/antenna/). The
accelerating charges launch a self-propagating disturbance: an oscillating electric
field regenerates a magnetic field at right angles to it, and that magnetic field in turn
regenerates the electric field, so the pair detaches from the antenna and radiates
outward at the speed of light. The wave carries energy but needs no medium — this is why
radio crosses the vacuum of space. A distant antenna, immersed in the passing field,
develops a tiny induced current (often a few microvolts) that a receiver amplifies,
filters, and decodes.

Three properties fully describe a simple radio wave and are the only things a
transmitter can manipulate: amplitude (strength), frequency (cycles per second), and
phase (position within the cycle). Modulation deliberately varies one or more of these
in step with the information being sent. The wave's [wavelength](/reference/wavelength/)
follows from its frequency by *λ = c / f*, and its
[polarization](/reference/polarization/) — the orientation of the electric field — must
usually match between transmit and receive antennas for good reception.

## In practice

How a radio wave behaves between transmitter and receiver depends strongly on its
frequency:

- **Free-space spreading.** Even in a vacuum, a wave's power density falls with the
  square of distance ([free-space path loss](/reference/free-space-path-loss/)), which
  is why link budgets matter.
- **Interaction with matter.** Lower-frequency waves diffract around hills and
  buildings; higher frequencies travel more like light and are blocked or reflected,
  producing [multipath](/reference/multipath-propagation/).
- **Noise and interference.** The received wave always arrives buried in thermal and
  man-made noise; the ratio of wanted signal to that floor
  ([SNR](/reference/signal-to-noise-ratio/)) bounds how reliably it can be decoded.

## Relevance to SDR

Radio waves are the raw input to any receiver. An SDR does not decode the wave directly;
its front end mixes a slice of spectrum down to [baseband](/reference/baseband/) and its
analog-to-digital converter turns the wave into a stream of
[IQ samples](/reference/iq-data/) — a complex-number representation that captures both
the amplitude and phase of the wave at each instant. From that point on, everything
GopherTrunk does — filtering, demodulation, symbol recovery — is arithmetic on those
samples. The physical radio wave has become numbers, but every property discussed here
survives the conversion and must be tracked to recover the message.

## Sources

[^wiki]: [Radio wave](https://en.wikipedia.org/wiki/Radio_wave) — Wikipedia, on radio-frequency electromagnetic radiation and its use for wireless communication.
[^itu]: [Recommendation ITU-R V.431: Nomenclature of the frequency and wavelength bands](https://www.itu.int/rec/R-REC-V.431/en) — ITU-R, the standard defining the radio-frequency band names and their limits.
