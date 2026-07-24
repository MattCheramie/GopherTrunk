---
slug: constellations-and-symbol-mapping
title: Constellations & symbol mapping
description: Reading digital modulation as points on the I/Q plane — how PSK and QAM map bits to symbols, why Gray coding matters, and what a constellation diagram tells you.
keywords: constellation diagram, symbol mapping, psk, qpsk, qam, gray coding, iq plane symbols, bits to symbols, digital modulation constellation
level: intermediate
status: full
prereq:
  - complex-signals-and-iq
  - demodulation
faq:
  - q: What is a constellation diagram?
    a: "A constellation diagram is a scatter plot of received symbols on the I/Q plane. Each modulation scheme defines a set of ideal points — the constellation — and every received symbol lands near one of them. Tight clusters right on the ideal points mean a clean signal; a fuzzy, spread-out cloud means noise, and a rotated or warped pattern points to a specific impairment."
  - q: Why is Gray coding used to map bits to symbols?
    a: "Gray coding arranges the bit patterns so that adjacent constellation points differ by only a single bit. When noise nudges a symbol into a neighbouring point — the most common kind of error — only one bit flips instead of several. That keeps the bit error rate low for a given symbol error rate, which is why almost every real system uses it."
---

# Constellations & symbol mapping

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
A **constellation** is the set of ideal points a modulation uses on the
[I/Q plane](/learn/dsp/complex-signals-and-iq/). **PSK** places points around a circle
(varying phase); **QAM** fills a grid (varying phase *and* amplitude). Each point encodes
a group of bits; **Gray coding** ensures neighbours differ by one bit so a slip costs one
error. Plotting received symbols against the ideal points — the **constellation diagram**
— is the fastest read on link health.
</div>

[Demodulation](/learn/dsp/demodulation/) turned I/Q into symbols; this lesson is the map
from those symbols to *bits*, and the picture that shows how well it's working. It
connects to the RF path's [digital modulation](/learn/rf-sdr/digital-modulation/) lesson.

## Symbols as points on a plane

Each symbol period, the transmitter sends one point from a fixed set. Because a baseband
sample is a complex I/Q value, that point has a position on the plane — a phase (angle)
and amplitude (radius). The receiver, once [carrier-locked](/learn/dsp/carrier-and-frequency-recovery/),
reads the incoming point and decides which ideal point it is nearest. That decision is the
symbol; a lookup turns the symbol into its bits.

<figure class="figure" markdown="0">
<svg viewBox="0 0 520 170" role="img" aria-label="Three constellations: BPSK with two points on the I axis, QPSK with four points, and 16-QAM as a four-by-four grid of points." xmlns="http://www.w3.org/2000/svg">
  <g>
    <line x1="15" y1="85" x2="145" y2="85" stroke="currentColor" stroke-opacity="0.3"/>
    <line x1="80" y1="30" x2="80" y2="140" stroke="currentColor" stroke-opacity="0.3"/>
    <circle cx="45" cy="85" r="4" fill="currentColor"/><circle cx="115" cy="85" r="4" fill="currentColor"/>
    <text x="80" y="158" text-anchor="middle" font-size="9" fill="currentColor">BPSK (1 bit)</text>
  </g>
  <g>
    <line x1="185" y1="85" x2="315" y2="85" stroke="currentColor" stroke-opacity="0.3"/>
    <line x1="250" y1="30" x2="250" y2="140" stroke="currentColor" stroke-opacity="0.3"/>
    <circle cx="285" cy="55" r="4" fill="currentColor"/><circle cx="215" cy="55" r="4" fill="currentColor"/>
    <circle cx="215" cy="115" r="4" fill="currentColor"/><circle cx="285" cy="115" r="4" fill="currentColor"/>
    <text x="250" y="158" text-anchor="middle" font-size="9" fill="currentColor">QPSK (2 bits)</text>
  </g>
  <g>
    <line x1="355" y1="85" x2="505" y2="85" stroke="currentColor" stroke-opacity="0.3"/>
    <line x1="430" y1="30" x2="430" y2="140" stroke="currentColor" stroke-opacity="0.3"/>
    <g fill="currentColor">
      <circle cx="385" cy="45" r="3"/><circle cx="415" cy="45" r="3"/><circle cx="445" cy="45" r="3"/><circle cx="475" cy="45" r="3"/>
      <circle cx="385" cy="72" r="3"/><circle cx="415" cy="72" r="3"/><circle cx="445" cy="72" r="3"/><circle cx="475" cy="72" r="3"/>
      <circle cx="385" cy="99" r="3"/><circle cx="415" cy="99" r="3"/><circle cx="445" cy="99" r="3"/><circle cx="475" cy="99" r="3"/>
      <circle cx="385" cy="126" r="3"/><circle cx="415" cy="126" r="3"/><circle cx="445" cy="126" r="3"/><circle cx="475" cy="126" r="3"/>
    </g>
    <text x="430" y="158" text-anchor="middle" font-size="9" fill="currentColor">16-QAM (4 bits)</text>
  </g>
</svg>
<figcaption>More points per constellation carry more bits per symbol — but pack the points closer, so noise defeats them sooner.</figcaption>
</figure>

## PSK and QAM: two families

| Scheme | Varies | Points | Bits/symbol |
|--------|--------|--------|-------------|
| **BPSK** | phase | 2 | 1 |
| **QPSK** | phase | 4 | 2 |
| **16-QAM** | phase **and** amplitude | 16 | 4 |

**PSK** keeps every point at the same radius and spreads them by angle — robust, because
amplitude carries no data. **QAM** uses a grid, squeezing more bits per symbol by using
amplitude too, at the cost of points sitting closer together and thus being easier for
noise to confuse. The four-level [C4FM](/learn/dsp/demodulation/) of P25 Phase 1 is a
frequency-domain cousin of the same bits-per-symbol idea.

## Gray coding: one slip, one bad bit

How you *label* the points matters as much as where they sit. Noise most often bumps a
symbol into an **adjacent** point, so the bit labels are assigned by **Gray coding** —
neighbours differ in exactly one bit. Then the common single-step slip flips only one
bit, keeping the [bit error rate](/learn/dsp/snr-evm-and-ber/) far below what a careless
labelling would give.

```text
QPSK, Gray-coded:   00 | 01        a slip to any *neighbour*
                    ----+----      changes exactly ONE bit,
                    10 | 11        never two.
```

## The diagram as a diagnostic

Plot the received symbols and the pattern *is* the diagnosis. Tight dots on the ideal
points: healthy. A fuzzy cloud: noise — low [SNR](/learn/dsp/snr-evm-and-ber/). A pattern
rotating or held at an angle: an uncorrected [carrier offset](/learn/dsp/carrier-and-frequency-recovery/).
Points smeared radially: [multipath](/learn/dsp/equalization/). The spread of each cluster
from its ideal point is measured as [EVM](/learn/dsp/snr-evm-and-ber/), the next unit's
first metric.

<div class="knowledge-check" data-quiz data-correct-msg="Right — Gray coding makes adjacent points differ by one bit, so a single-step slip flips only one bit." markdown="0">
  <p class="knowledge-check__q">Quick check: why are constellation points Gray-coded?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">To pack more points into the same space</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">So a slip to an adjacent point flips only one bit</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">To remove the carrier frequency offset</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- A **constellation** is the set of ideal I/Q points a modulation uses; symbols map to bits.
- **PSK** varies phase (points on a circle); **QAM** varies phase and amplitude (a grid).
- **Gray coding** makes neighbours differ by one bit, minimizing the bit error rate.
- The **constellation diagram** is a fast diagnostic — its shape names the impairment.

Next up: why reflections smear symbols together, and the equalizer that undoes a channel.
