---
slug: qpsk
title: QPSK
entry_type: technology
category: modulation
description: QPSK (quadrature phase-shift keying) carries two bits per symbol as one of four carrier phases; its variants pi/4-DQPSK and CQPSK appear in P25, TETRA, and satellite links.
keywords: QPSK, quadrature phase shift keying, four phases, two bits per symbol, OQPSK, DQPSK, pi/4-DQPSK, CQPSK, constellation, satellite, P25 Phase 2
aka: [QPSK, quadrature phase-shift keying]
autolink: true
infobox:
  - { label: Type, value: Digital modulation (PSK) }
  - { label: Carries, value: 2 bits per symbol (four phases) }
  - { label: Variants, value: OQPSK, pi/4-DQPSK, CQPSK }
see_also: [phase-shift-keying, bpsk, pi-4-dqpsk, cqpsk, constellation-diagram, costas-loop]
cite_urls:
  - https://en.wikipedia.org/wiki/Phase-shift_keying
  - https://en.wikipedia.org/wiki/Quadrature_phase-shift_keying
---

**QPSK** (quadrature phase-shift keying) is [phase-shift
keying](/reference/phase-shift-keying/) with **four** carrier phases, so each
[symbol](/reference/symbol-rate/) carries **two bits**.[^wiki] By using both the in-phase
(I) and quadrature (Q) components of the [carrier](/reference/carrier-wave/), QPSK
doubles the throughput of [BPSK](/reference/bpsk/) in the same bandwidth while keeping
the same bit-error performance — which makes it one of the most widely deployed digital
modulations in the world.

<figure class="figure" markdown="0">
<svg viewBox="0 0 300 220" role="img" aria-label="A QPSK constellation with four points at the diagonals of the IQ plane, each labelled with a two-bit dibit." xmlns="http://www.w3.org/2000/svg">
  <line x1="30" y1="110" x2="270" y2="110" stroke="currentColor" stroke-opacity="0.4"/>
  <line x1="150" y1="20" x2="150" y2="200" stroke="currentColor" stroke-opacity="0.4"/>
  <text x="262" y="124" font-size="10" fill="currentColor">I</text><text x="136" y="30" font-size="10" fill="currentColor">Q</text>
  <g fill="currentColor"><circle cx="210" cy="55" r="5"/><circle cx="90" cy="55" r="5"/><circle cx="90" cy="165" r="5"/><circle cx="210" cy="165" r="5"/></g>
  <g font-size="10" fill="currentColor"><text x="218" y="50">01</text><text x="60" y="50">11</text><text x="60" y="182">10</text><text x="218" y="182">00</text></g>
</svg>
<figcaption>QPSK places four phase points at the diagonals of the IQ plane; each represents a Gray-coded dibit, so two bits ride on every symbol.</figcaption>
</figure>

## How it works

QPSK can be seen as two independent [BPSK](/reference/bpsk/) streams: the I channel
carries one bit and the Q channel the other, added in quadrature. The four resulting
phases — typically at 45°, 135°, 225°, 315° — are usually labelled with a
[Gray code](/reference/gray-code/), so neighbouring points differ by a single bit and the
most likely symbol error costs only one bit. Because a QPSK symbol packs two bits into
the same energy that BPSK spends on one, QPSK carries twice the data in the same spectrum
without losing noise margin, which is its defining advantage.

Demodulation is coherent: a [Costas loop](/reference/costas-loop/) recovers the carrier
so the received cloud lines up with the reference constellation, then the I and Q signs
give the dibit. The carrier-recovery loop has a four-fold phase ambiguity, so systems use
differential encoding or a known sync word to fix the absolute rotation. A drawback of
plain QPSK is that a symbol transition can swing the phase by 180°, driving the envelope
briefly through zero and stressing non-linear amplifiers.

## Variants

Two refinements dominate in radio. **Offset QPSK (OQPSK)** staggers the I and Q bit
transitions by half a symbol so the phase never jumps a full 180°, easing amplifier
requirements. **[π/4-DQPSK](/reference/pi-4-dqpsk/)** rotates the constellation by 45°
each symbol and encodes bits differentially in phase *changes*, again avoiding
through-origin transitions; it is the modulation of P25 Phase 1's CQPSK path, TETRA, and
several cordless standards. **[CQPSK](/reference/cqpsk/)** (compatible QPSK) is the
linear-modulation twin of P25 C4FM, sharing the same over-the-air symbols so the two
interoperate.

## Relevance to SDR

QPSK and its variants are everywhere a link must be spectrally efficient yet robust:
satellite downlinks, DVB, cable and cellular systems, and land-mobile radio. In P25,
π/4-DQPSK/CQPSK is the linear cousin of C4FM at 4800 symbols/s; TETRA uses π/4-DQPSK at
18 000 symbols/s; P25 Phase 2 uses an H-DQPSK-family scheme. On a constellation display,
QPSK shows four clusters at the diagonals; timing and carrier errors rotate or smear them.

GopherTrunk decodes P25 (whose symbols can be viewed through the QPSK/CQPSK lens) and
TETRA, so QPSK-family demodulation is directly relevant to its decode chain, alongside
the equivalent 4FSK view of the same land-mobile signals.

## Sources

[^wiki]: [Phase-shift keying](https://en.wikipedia.org/wiki/Phase-shift_keying) — Wikipedia, for the QPSK definition, the two-bits-per-symbol constellation, and the OQPSK/π-4-DQPSK variants.
