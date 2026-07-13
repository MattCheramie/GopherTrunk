---
slug: nrzi
title: NRZI
entry_type: term
category: modulation
description: "NRZI (non-return-to-zero inverted) is a line code that represents a bit by the presence or absence of a transition rather than a fixed level — used by AX.25 and AIS."
keywords: NRZI, non-return-to-zero inverted, line code, AX.25, APRS, AIS, bit stuffing, HDLC, differential encoding, clock recovery, USB
aka: [NRZI, "non-return-to-zero inverted"]
autolink: true
see_also: [ax25, aprs, ais, manchester-coding, differential-decoding, clock-recovery]
related_lessons:
  - { title: "The demodulation pipeline", url: /learn/rf-sdr/demodulation-pipeline/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Non-return-to-zero#NRZI
  - https://en.wikipedia.org/wiki/High-Level_Data_Link_Control
---

**NRZI** (**non-return-to-zero inverted**) is a line code in which each bit is conveyed
by **whether the signal level changes**, not by the level itself: conventionally a `0`
causes a transition and a `1` causes none (or vice-versa).[^wiki] It is a *differential*
line code — the meaning is in the change — used by [AX.25](/reference/ax25/) /
[APRS](/reference/aprs/), [AIS](/reference/ais/), USB, and many other framed serial links.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 120" role="img" aria-label="A bit pattern above a NRZI waveform that changes level on each zero and holds level on each one." xmlns="http://www.w3.org/2000/svg">
  <g font-family="monospace" font-size="11" fill="currentColor" text-anchor="middle"><text x="60" y="30">0</text><text x="110" y="30">1</text><text x="160" y="30">1</text><text x="210" y="30">0</text><text x="260" y="30">0</text><text x="310" y="30">1</text></g>
  <path d="M35 80 V50 H85 V80 H135 V80 H185 V50 H235 V80 H285 V50 H335 V50 H385" fill="none" stroke="currentColor" stroke-width="1.8"/>
  <text x="230" y="108" text-anchor="middle" font-size="8.5" fill="currentColor">transition = 0 · no transition = 1</text>
</svg>
<figcaption>NRZI encodes bits as the presence or absence of a level transition, which guarantees frequent edges for timing.</figcaption>
</figure>

## How it works

Ordinary NRZ (non-return-to-zero) maps a `1` to one level and a `0` to the other and holds it for
the bit period — simple, but a long run of identical bits produces a long flat stretch with no
edges, and a receiver with no separate clock line has nothing to synchronise to. NRZI breaks that
coupling between bit *value* and signal *level*. Instead, each bit decides whether to **toggle**:
in the common convention a `0` flips the level and a `1` leaves it unchanged (the "inverted"
refers to the toggle-on-zero rule; the opposite mapping, toggle-on-one, is equally valid and used
by USB).

Two useful properties fall out:

- **Polarity independence.** Because information lives in transitions, not absolute levels, an
  accidental inversion of the whole signal — a swapped pair, an inverting stage in the radio path
  — leaves the decoded bits unchanged. This is the same trick as
  [differential decoding](/reference/differential-decoding/) applied to a baseband line code.
- **It still needs help with long same-bit runs.** NRZI guarantees an edge only for the bit that
  toggles. A long run of the non-toggling bit (all `1`s in the toggle-on-zero convention) still
  produces a flat stretch, so NRZI alone does not solve the timing problem — it must be paired
  with a rule that forces transitions.

That rule is **bit stuffing**. HDLC and its derivatives insert a `0` after every five consecutive
`1`s in the data before NRZI encoding; since a `0` always toggles the line, no run longer than
five bits can pass without an edge, capping the interval the receiver's
[clock recovery](/reference/clock-recovery/) can drift. The receiver removes the stuffed bits
after decoding. The six-consecutive-`1` pattern that bit stuffing can never produce in data is
reserved as the `01111110` flag that marks frame boundaries.

## In practice

NRZI plus bit stuffing is the standard physical-layer coding of the HDLC frame family, which is
why it shows up across so many systems: amateur packet radio and APRS ([AX.25](/reference/ax25/)
is an HDLC variant), marine [AIS](/reference/ais/), the USB serial bus, and a range of industrial
and telemetry links. In the RF cases the NRZI bitstream typically modulates an
[FSK](/reference/frequency-shift-keying/) or [GFSK](/reference/gfsk/) carrier, so the demodulated
baseband is the NRZI waveform the decoder must slice and un-stuff.

## Relevance to SDR

A software decoder for an AX.25/AIS-style signal FM- or FSK-demodulates the channel to recover
the NRZI baseband, slices it to a hard bitstream, then reverses the coding: detect transitions to
undo NRZI, strip stuffed bits, and find the HDLC flags to frame the packet. Getting the NRZI and
bit-stuffing steps right is what makes the framing and CRC check downstream succeed. GopherTrunk's
demodulation pipeline handles the constant-envelope FSK baseband these formats ride on; the
NRZI/HDLC layer is the framing that sits on top of that recovered bitstream.

## Sources

[^wiki]: [Non-return-to-zero — NRZI](https://en.wikipedia.org/wiki/Non-return-to-zero#NRZI) — Wikipedia, for the transition-based line-code definition and conventions.
[^hdlc]: [High-Level Data Link Control](https://en.wikipedia.org/wiki/High-Level_Data_Link_Control) — Wikipedia, for bit stuffing, the flag pattern, and NRZI's role in HDLC framing.
