---
slug: pulse-amplitude-modulation
title: Pulse-amplitude modulation (PAM)
entry_type: technology
category: modulation
description: Pulse-amplitude modulation (PAM) encodes data in the amplitude of a train of pulses; multi-level PAM underlies baseband links like Ethernet and is one axis of QAM.
keywords: PAM, pulse amplitude modulation, PAM-4, multi-level, baseband, line code, Ethernet, QAM, symbol levels, eye diagram
aka: [pulse-amplitude modulation, PAM, PAM-4]
autolink: true
infobox:
  - { label: Type, value: Baseband digital modulation }
  - { label: Varies, value: Pulse amplitude (discrete levels) }
  - { label: Used by, value: Ethernet, DSL; one axis of QAM }
see_also: [pulse-code-modulation, quadrature-amplitude-modulation, amplitude-shift-keying, pulse-shaping, eye-diagram, symbol-rate]
cite_urls:
  - https://en.wikipedia.org/wiki/Pulse-amplitude_modulation
  - https://en.wikipedia.org/wiki/Quadrature_amplitude_modulation
---

**Pulse-amplitude modulation** (**PAM**) encodes data in the **amplitude of a train of
pulses**: each [symbol](/reference/symbol-rate/) sets the height of one pulse to one of a
fixed set of levels.[^wiki] It is a *baseband* scheme — the information rides directly on
pulse amplitudes rather than on a modulated [carrier](/reference/carrier-wave/) — and it
is the amplitude backbone of many wired links and one of the two axes of
[quadrature amplitude modulation](/reference/quadrature-amplitude-modulation/).

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="A sequence of rectangular pulses of four different heights on a baseline, illustrating four-level pulse-amplitude modulation." xmlns="http://www.w3.org/2000/svg">
  <line x1="20" y1="125" x2="440" y2="125" stroke="currentColor" stroke-opacity="0.4"/>
  <g stroke="currentColor" stroke-opacity="0.2"><line x1="20" y1="95" x2="440" y2="95"/><line x1="20" y1="65" x2="440" y2="65"/><line x1="20" y1="35" x2="440" y2="35"/></g>
  <g fill="currentColor"><rect x="40" y="65" width="40" height="60"/><rect x="100" y="35" width="40" height="90"/><rect x="160" y="95" width="40" height="30"/><rect x="220" y="35" width="40" height="90"/><rect x="280" y="95" width="40" height="30"/><rect x="340" y="65" width="40" height="60"/></g>
  <text x="20" y="145" font-size="9" fill="currentColor">each pulse height is one of four levels -> 2 bits per symbol (PAM-4)</text>
</svg>
<figcaption>PAM sets each pulse to one of several amplitude levels; PAM-4 (four levels) carries two bits per symbol and is common on high-speed wired links.</figcaption>
</figure>

## How it works

The transmitter produces a pulse each symbol period and scales its height by the data. A
two-level scheme (PAM-2) is ordinary binary NRZ signalling; PAM-4 uses four levels for 2
bits per symbol; PAM-8 and PAM-16 push further. To limit bandwidth and control
intersymbol interference, the rectangular pulses are replaced by shaped ones — a
[root-raised-cosine](/reference/root-raised-cosine-filter/) or similar
[pulse shape](/reference/pulse-shaping/) — so that at each sampling instant only the
current symbol contributes. The receiver samples at the symbol rate and slices the level.

PAM's clarity makes the [eye diagram](/reference/eye-diagram/) especially useful: an
*L*-level PAM signal opens *L−1* stacked eyes, and their height and width directly show
the noise and timing margin. As with all amplitude schemes, PAM is vulnerable to gain
drift and noise, and every extra level halves the spacing between adjacent levels, so
higher-order PAM demands a cleaner channel and careful equalisation.

## Variants and relation to QAM

Passband PAM — modulating a carrier's amplitude with PAM levels — is exactly
[amplitude-shift keying](/reference/amplitude-shift-keying/), and its binary case is
[on-off keying](/reference/on-off-keying/). The most important relationship, though, is
to **QAM**: a QAM signal is two independent PAM streams placed on the orthogonal I and Q
carriers. Square QAM constellations (16-QAM, 64-QAM, 256-QAM) are literally the Cartesian
product of two PAM axes, which is why understanding PAM levels is the key to reading a
QAM constellation.

## Relevance to SDR

Pure baseband PAM is most familiar from wired links: **PAM-4** carries 100 Mbit and
multi-gigabit Ethernet over copper, DSL and many SerDes lanes use multi-level PAM, and it
is a staple of digital-communications teaching because its eye diagram is so legible. In
the wireless world PAM appears as the I/Q building blocks of QAM rather than on its own.

GopherTrunk decodes frequency- and phase-keyed land-mobile signals rather than baseband
PAM or high-order QAM, so it does not demodulate PAM directly. It is documented here
because PAM is the amplitude primitive underneath both ASK and QAM, and its eye-diagram
intuition transfers straight to the multi-level [4FSK](/reference/four-fsk/) symbols
GopherTrunk does recover.

## Sources

[^wiki]: [Pulse-amplitude modulation](https://en.wikipedia.org/wiki/Pulse-amplitude_modulation) — Wikipedia, for the amplitude-of-pulses definition, PAM-4 usage, and the relationship to QAM.
