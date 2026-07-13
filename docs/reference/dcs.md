---
slug: dcs
title: DCS
entry_type: technology
category: modulation
description: "DCS (Digital-Coded Squelch) sends a continuous low-speed digital codeword beneath FM voice so a receiver's squelch opens only for the matching code."
keywords: DCS, DPL, Digital Coded Squelch, Digital Private Line, digital squelch, 23-bit Golay, 134.4 bps, sub-audible code, turn-off code
aka: [DCS, DPL, Digital-Coded Squelch, Digital Private Line]
autolink: true
infobox:
  - { label: Type, value: "Analog sub-audible digital signaling" }
  - { label: Idea, value: "Repeating 23-bit code gates squelch" }
  - { label: Examples, value: "DPL codes on FM land-mobile radios" }
see_also: [ctcss, squelch, frequency-modulation, golay-code]
cite_urls:
  - https://en.wikipedia.org/wiki/Continuous_Tone-Coded_Squelch_System#Digital-Coded_Squelch
---

**DCS** (Digital-Coded Squelch, and marketed as Motorola's **DPL** / Digital Private Line) is the
digital counterpart to [CTCSS](/reference/ctcss/): instead of a single sub-audible tone, it sends a
continuous **low-speed digital codeword** beneath [FM](/reference/frequency-modulation/) voice, and a
receiver's [squelch](/reference/squelch/) opens only when it decodes the matching code.[^wiki] Like
CTCSS it manages channel sharing and repeater access, not privacy.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 140" role="img" aria-label="A sub-audible digital waveform of a repeating 23-bit codeword shown below the voice band, framed as a continuously looping code." xmlns="http://www.w3.org/2000/svg">
  <line x1="30" y1="70" x2="440" y2="70" stroke="currentColor" stroke-opacity="0.35" stroke-dasharray="4 3"/>
  <path d="M40 90 V50 H70 V90 H90 V50 H120 V90 H150 V50 H165 V90 H195 V50 H215 V90 H240 V50 H260 V90 H290 V50 H305 V90 H335 V50 H360 V90 H380 V50 H410 V90" fill="none" stroke="currentColor" stroke-width="1.5"/>
  <text x="230" y="118" text-anchor="middle" font-size="9" fill="currentColor">23-bit codeword, repeated at ~134.4 bps</text>
  <line x1="40" y1="30" x2="410" y2="30" stroke="currentColor" stroke-opacity="0.4" marker-start="url(#dcsar)" marker-end="url(#dcsar)"/>
  <text x="225" y="25" text-anchor="middle" font-size="8" fill="currentColor">continuous loop while keyed</text>
  <defs><marker id="dcsar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>DCS continuously repeats a 23-bit codeword in the sub-audible band; the receiver opens squelch only on a match.</figcaption>
</figure>

## How it works

DCS transmits a continuously repeating 23-bit word at a low bit rate (134.4 bit/s) using slow
frequency-shift keying at low deviation, placed in the same sub-audible region below 300 Hz that
[CTCSS](/reference/ctcss/) uses, so it is separated from voice with a simple filter. Of the 23 bits,
9 carry the user-chosen code (quoted as a three-digit octal number, e.g. 023, 754) and the remaining
bits are a fixed-polynomial [Golay](/reference/golay-code/) error-correcting/framing structure that
lets the receiver find the word boundary and reject noise. The receiver continuously demodulates and
correlates the sub-audible bitstream; when it matches the programmed code, squelch opens. Because
there are far more valid DCS codes (on the order of 80-100 in common use) than CTCSS tones, DCS gives
system planners more distinct channel-sharing groups.

A characteristic detail is the **turn-off code**: when the transmitter unkeys, it briefly sends a
distinctive ~134 Hz phase reversal / turn-off sequence so receivers close their squelch immediately
rather than waiting to time out, avoiding a burst of noise ("squelch tail") at the end of each
transmission.

## Relevance to SDR

Like CTCSS, DCS is something a software scanner can decode to sort and label conventional FM traffic
by group. After FM-demodulating a channel, a receiver low-pass filters the sub-audible band, recovers
the 134.4 bit/s stream, finds the 23-bit frame using the known Golay structure, and reads out the octal
code — useful metadata when monitoring shared conventional channels. The mechanism is analog land-mobile
signaling and sits outside GopherTrunk's digital-trunking focus, where group membership is carried
explicitly as talkgroup identifiers in the control-channel messaging rather than as a sub-audible code;
GopherTrunk therefore does not decode DCS in its trunking path. Its relevance here is as the digital
sibling of CTCSS and a compact real-world example of a Golay-protected sub-audible codeword.

## Sources

[^wiki]: [Digital-Coded Squelch](https://en.wikipedia.org/wiki/Continuous_Tone-Coded_Squelch_System#Digital-Coded_Squelch) — Wikipedia, for the 23-bit codeword, 134.4 bit/s rate, and turn-off behavior.
