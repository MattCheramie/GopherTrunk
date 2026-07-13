---
slug: differential-decoding
title: Differential decoding
entry_type: algorithm
category: synchronization
description: Differential decoding recovers data carried in the transitions between symbols rather than their absolute values, removing the need for an absolute phase reference.
keywords: differential decoding, differential encoding, DBPSK, DQPSK, pi/4-DQPSK, NRZI, phase ambiguity, differential coding, transition coding, non-coherent detection
aka: [differential decoding, differential coding]
autolink: true
infobox:
  - { label: Type, value: Transition-based line/phase coding }
  - { label: Resolves, value: Absolute phase ambiguity }
  - { label: Used by, value: DBPSK/DQPSK, π/4-DQPSK, NRZI }
see_also: [pi-4-dqpsk, phase-shift-keying, nrzi, differential-cryptanalysis, costas-loop, constellation-diagram]
cite_urls:
  - https://en.wikipedia.org/wiki/Differential_coding
  - https://en.wikipedia.org/wiki/Phase-shift_keying
---

**Differential decoding** recovers information that was encoded in the *change*
between successive symbols rather than in their absolute values.[^wiki] By making
each symbol's meaning depend only on how it differs from the one before it, a
differentially-coded link needs no absolute [phase](/reference/phase/) reference —
it does not matter which of several equivalent constellation rotations the receiver
happens to lock onto, because the *transition* is the same in every rotation. This
neatly resolves the **phase ambiguity** left by carrier-recovery loops.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 130" role="img" aria-label="A symbol stream where the data is decoded from the difference between each symbol and its predecessor: a delay-by-one element feeds a comparison that outputs the transition." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="ddar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <line x1="14" y1="45" x2="120" y2="45" stroke="currentColor" stroke-width="1.1" marker-end="url(#ddar)"/><text x="55" y="37">rₙ (symbol)</text>
    <rect x="120" y="30" width="70" height="30" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="155" y="49">compare</text>
    <rect x="120" y="90" width="70" height="26" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="155" y="107">delay z⁻¹</text>
    <path d="M100 45 V 103 H 119" fill="none" stroke="currentColor" stroke-width="1" marker-end="url(#ddar)"/>
    <path d="M155 90 V 61" fill="none" stroke="currentColor" stroke-width="1" marker-end="url(#ddar)"/><text x="188" y="78">rₙ₋₁</text>
    <line x1="190" y1="45" x2="300" y2="45" stroke="currentColor" stroke-width="1.1" marker-end="url(#ddar)"/><text x="260" y="37">Δ = data</text>
    <text x="360" y="41">transition,</text><text x="360" y="53">not level,</text><text x="360" y="65">carries bits</text>
  </g>
</svg>
<figcaption>Differential decoding compares each received symbol with the previous one; the difference carries the data, so a constant unknown phase rotation cancels out.</figcaption>
</figure>

## How it works

At the transmitter, **differential encoding** maps each data symbol to a *change*
applied to the running state: the sent symbol is the previous sent symbol combined
with the current data (for phase modulation, the phase *increment*; for a binary line
code, whether to toggle). At the receiver, decoding inverts that step — it compares the
current received symbol against the previous one and outputs their difference.

The crucial property: any fixed offset common to both symbols cancels in the
comparison. If a carrier-recovery loop settles on a phase rotated by, say, 90° from the
transmitter's, every received symbol is rotated the same way — but the *difference*
between consecutive symbols is unchanged, so the data survives. This is why
differential coding is the standard cure for the **N-fold phase ambiguity** that a
PSK carrier loop or [Costas loop](/reference/costas-loop/) inherently leaves (a QPSK
loop can lock in any of four orientations).

The trade-off is a small noise penalty: because a decision leans on a possibly-noisy
*previous* symbol, a single symbol error tends to corrupt two output symbols, costing
roughly 2–3 dB versus ideal coherent detection with perfect absolute phase.

## Variants

- **DBPSK / DQPSK** — differential [PSK](/reference/phase-shift-keying/): data is the
  phase *step* (0/π for DBPSK; 0, ±π/2, π for DQPSK). Can be detected non-coherently by
  comparing consecutive symbol phases, avoiding a carrier loop entirely.
- **[π/4-DQPSK](/reference/pi-4-dqpsk/)** — the phase reference alternates between two
  QPSK constellations offset by π/4, so every symbol forces a phase change (no 0°
  transitions). That guarantees frequent transitions for timing recovery and avoids
  zero-crossings through the origin, which is why P25 Phase 1 and TETRA use it at
  4800 symbols/s.
- **[NRZI](/reference/nrzi/)** — a differential *line* code: a data 0 (or 1, by
  convention) is encoded as a level *transition*, a data 1 as no change. Because
  meaning rides on transitions, it is immune to an inverted line and guarantees edges
  for clock recovery; it appears in USB, magnetic recording, and various data radios.

## Relevance to SDR

Differential coding is pervasive in real systems precisely because carrier and phase
references are imperfect. Trunked-radio waveforms lean on it heavily: P25 Phase 1 and
TETRA transmit π/4-DQPSK, so a decoder must **differentially decode** the recovered
phases to get dibits, and many FSK/line-coded control formats use NRZI-style
transition coding. GopherTrunk performs differential decoding in its π/4-DQPSK demod
path — recovering symbols from phase transitions after timing and carrier recovery — so
that a locked-but-rotated constellation still yields the correct bitstream.

## Sources

[^wiki]: [Differential coding](https://en.wikipedia.org/wiki/Differential_coding) — Wikipedia, on encoding data in symbol transitions to remove absolute-reference dependence; see also [Phase-shift keying](https://en.wikipedia.org/wiki/Phase-shift_keying) for DPSK.
