---
slug: continuous-wave
title: Continuous wave (CW)
entry_type: technology
category: modulation
description: Continuous wave (CW) is an unmodulated carrier keyed on and off to send Morse code; it is the oldest radio mode and the narrowest, most power-efficient way to communicate.
keywords: CW, continuous wave, Morse code, keying, telegraphy, A1A, narrowband, amateur radio, on-off keying
aka: [continuous wave, CW]
autolink: true
infobox:
  - { label: Type, value: Keyed carrier (Morse) }
  - { label: Idea, value: Carrier on/off = dots and dashes }
  - { label: Used by, value: Amateur radio, beacons, telegraphy }
see_also: [morse-code, on-off-keying, carrier-wave, amplitude-shift-keying, single-sideband]
cite_urls:
  - https://en.wikipedia.org/wiki/Continuous_wave
  - https://en.wikipedia.org/wiki/Morse_code
---

**Continuous wave** (**CW**) is a steady, single-frequency
[carrier](/reference/carrier-wave/) that is switched on and off by hand or machine to
send [Morse code](/reference/morse-code/).[^wiki] It is the oldest practical radio mode
and, despite the name, the transmitter is *not* continuous in use — the carrier is keyed
into the dots and dashes of Morse. CW is essentially
[on-off keying](/reference/on-off-keying/) of a pure tone, which makes it the narrowest
and most power-efficient way to move information over the air.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 130" role="img" aria-label="A carrier keyed into short and long bursts spelling a Morse sequence of a dot, a dash, and a dot separated by gaps." xmlns="http://www.w3.org/2000/svg">
  <line x1="20" y1="70" x2="440" y2="70" stroke="currentColor" stroke-opacity="0.3"/>
  <path d="M30 70 q4 -22 8 0 t8 0 t8 0" fill="none" stroke="currentColor" stroke-width="1.5"/>
  <path d="M110 70 q4 -22 8 0 t8 0 t8 0 t8 0 t8 0 t8 0 t8 0 t8 0 t8 0 t8 0 t8 0 t8 0" fill="none" stroke="currentColor" stroke-width="1.5"/>
  <path d="M290 70 q4 -22 8 0 t8 0 t8 0" fill="none" stroke="currentColor" stroke-width="1.5"/>
  <g font-size="11" fill="currentColor" font-family="monospace"><text x="34" y="105">.</text><text x="150" y="100">_</text><text x="294" y="105">.</text></g>
  <text x="20" y="122" font-size="9" fill="currentColor">short key-down = dot, long key-down = dash, gaps separate elements</text>
</svg>
<figcaption>CW keys a single carrier into the dots and dashes of Morse code — the narrowest and most efficient radio mode.</figcaption>
</figure>

## How it works

A CW transmitter generates a stable carrier and gates it with a key. To avoid a harsh
click that splatters energy across the band, the on/off transitions are shaped with a
gentle rise and fall time, keeping the occupied bandwidth just a few tens of hertz wide.
Because there is no sideband information beyond the keying, all of the transmitter's
power sits in one tone, so CW reaches far on very little power — the reason it remains
popular for weak-signal work and beacons.

A CW signal is, strictly, unmodulated when the key is down, so a plain amplitude
receiver would only hear silence and clicks. Receivers therefore use a **beat-frequency
oscillator (BFO)**: they mix the received carrier against a local oscillator offset by a
few hundred hertz, turning the on/off carrier into an audible tone that pulses in time
with the Morse. In [single-sideband](/reference/single-sideband/) receivers this is the
same product-detector path used for voice, simply tuned so the carrier lands at an
audio pitch.

## Relevance to SDR

CW is trivially handled in software radio: tune near the carrier, take a narrow slice of
spectrum, and either listen to the beat tone or detect the on/off envelope to decode
Morse automatically. Because the mode is so narrow, an SDR can pack many CW signals into
a small span, and a waterfall of the amateur CW sub-bands shows dozens of thin dashed
vertical lines, each a separate operator. Amateur radio, navigational and propagation
beacons, and some legacy maritime identifiers still use CW.

For GopherTrunk this is background rather than a decode target: GopherTrunk is a trunked
land-mobile scanner (P25, DMR, NXDN, TETRA), and CW telegraphy belongs to the HF/amateur
world served by general-purpose SDR tools. CW is included here because it is the
historical root of digital keying — the direct ancestor of
[ASK](/reference/amplitude-shift-keying/) and OOK — and remains the canonical example of
trading data rate for range and bandwidth.

## Sources

[^wiki]: [Continuous wave](https://en.wikipedia.org/wiki/Continuous_wave) — Wikipedia, for the definition of CW as a keyed carrier for Morse telegraphy and its bandwidth/efficiency properties.
