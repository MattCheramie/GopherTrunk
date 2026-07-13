---
slug: morse-code
title: Morse Code
entry_type: technology
category: amateur-digital
description: "Morse code encodes text as short and long on/off events (dots and dashes) whose timing carries the message, most often keyed as a continuous-wave carrier switched on and off."
keywords: Morse code, CW, continuous wave, dot dash, dit dah, telegraphy, Samuel Morse, on-off keying, amateur radio, prosigns
aka: [Morse Code, CW]
autolink: true
infobox:
  - { label: Type, value: Timing-based symbol code }
  - { label: Idea, value: Text as dots, dashes, and timed gaps }
  - { label: Examples, value: Amateur CW, maritime distress (historic), beacons }
see_also: [continuous-wave, on-off-keying, rtty, frequency-shift-keying, amplitude-shift-keying]
cite_urls:
  - https://en.wikipedia.org/wiki/Morse_code
---

**Morse code** is a method of encoding text characters as sequences of short and long
signalling events — **dots (dits) and dashes (dahs)** — where the *timing* of on and off
periods carries the meaning. On radio it is almost always sent as
[continuous wave (CW)](/reference/continuous-wave/): an unmodulated carrier switched on and
off with a key, the simplest form of [on-off keying](/reference/on-off-keying/).[^wiki] A
single tone, turned on and off in a disciplined rhythm, is enough to pass a full text
message across a noisy channel that would defeat wider modes.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 110" role="img" aria-label="Morse code keys a carrier on for one unit as a dot and three units as a dash, with one-unit gaps between elements and three-unit gaps between letters, shown as the word SOS." xmlns="http://www.w3.org/2000/svg">
  <line x1="20" y1="70" x2="440" y2="70" stroke="currentColor" stroke-width="1"/>
  <g fill="currentColor" fill-opacity="0.85">
    <rect x="20" y="45" width="10" height="25"/><rect x="38" y="45" width="10" height="25"/><rect x="56" y="45" width="10" height="25"/>
    <rect x="82" y="45" width="30" height="25"/><rect x="120" y="45" width="30" height="25"/><rect x="158" y="45" width="30" height="25"/>
    <rect x="214" y="45" width="10" height="25"/><rect x="232" y="45" width="10" height="25"/><rect x="250" y="45" width="10" height="25"/>
  </g>
  <text x="43" y="92" font-size="9" fill="currentColor" text-anchor="middle">S (· · ·)</text>
  <text x="135" y="92" font-size="9" fill="currentColor" text-anchor="middle">O (— — —)</text>
  <text x="237" y="92" font-size="9" fill="currentColor" text-anchor="middle">S (· · ·)</text>
  <text x="360" y="60" font-size="8.5" fill="currentColor" text-anchor="middle">dot = 1 unit, dash = 3 units</text>
</svg>
<figcaption>Morse keys the carrier on for one time unit (dot) or three (dash); gaps of one, three, and seven units separate elements, letters, and words — here spelling SOS.</figcaption>
</figure>

## How it works

Morse is built entirely on a single time unit. A **dot** is one unit of carrier-on; a
**dash** is three. Within a character, elements are separated by a one-unit off period;
letters are separated by three units of silence, and words by seven. Each character maps to
a distinct dot/dash pattern — E is a single dot, T a single dash, and common letters get the
shortest codes so ordinary text sends efficiently. Speed is quoted in words per minute (WPM),
benchmarked against the word "PARIS." Because the receiver only has to detect *presence or
absence* of a tone, Morse holds up in weak-signal and interference conditions where schemes
that must recover phase or many amplitude levels would fail; a trained operator's ear (or a
narrow filter and simple detector) does the decoding.

International Morse standardises the letter, digit, and punctuation patterns, and adds
**prosigns** — special element groups like AR (end of message) or SK (end of contact) — for
procedural signalling. There is no forward error correction: robustness comes from the
extreme narrowness of the signal and the redundancy of natural language.

## Relevance to SDR

Morse/CW is still heavily used in amateur radio and lingers in navigation-beacon and station
identifiers, so it is a common sight on any HF or VHF waterfall — a dashed line blinking in a
few-hertz-wide slot. In a software-defined receiver it is trivial to demodulate: mix the CW
carrier down to an audio beat note (BFO), filter narrowly, and detect the on/off envelope; a
[continuous-wave](/reference/continuous-wave/) tone under
[on-off keying](/reference/on-off-keying/) is about the simplest signal a decoder can face.
Many SDR programs (Fldigi, CW Skimmer, and the like) automate the timing-to-text step.

**GopherTrunk** is focused on digital trunked land-mobile systems and does not include a
Morse decoder; CW is out of its scope. It is worth understanding here because CW is the
historical root of the on/off keyed digital modes GopherTrunk *does* touch, and it frequently
shares the bands a wideband SDR sweeps.

## Sources

[^wiki]: [Morse code](https://en.wikipedia.org/wiki/Morse_code) — Wikipedia, for the dot/dash timing rules, WPM/PARIS convention, prosigns, and CW keying on radio.
