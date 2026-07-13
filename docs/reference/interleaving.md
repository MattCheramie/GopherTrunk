---
slug: interleaving
title: Interleaving
entry_type: algorithm
category: error-correction
description: Interleaving reorders bits or symbols before transmission so a burst of channel errors is scattered across many codewords, converting bursts into the isolated errors an FEC code can fix.
keywords: interleaving, de-interleaving, block interleaver, convolutional interleaver, burst error, interleaving depth, latency, FEC, fading
aka: [interleaving, interleaver, de-interleaving]
autolink: true
infobox:
  - { label: Type, value: Error-resilience technique }
  - { label: Converts, value: Burst errors → random errors }
  - { label: Costs, value: Latency proportional to depth }
see_also: [forward-error-correction, reed-solomon-code, bch-code, bptc, multipath-propagation]
related_lessons:
  - { title: "The demodulation pipeline", url: /learn/rf-sdr/demodulation-pipeline/ }
related_reading:
  - { title: "SDR Internals, Part 9: Framing & forward error correction", url: /blog/deep-dives/sdr-internals-09-framing-fec/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Interleaving_(data)
  - https://en.wikipedia.org/wiki/Burst_error-correcting_code
---

**Interleaving** reorders bits or symbols before transmission and restores their order on
receive, so that a **burst** of channel errors — from a fade, a click of interference, or a
dropped symbol — is spread thinly across many codewords instead of overwhelming one.[^wiki] It
adds no redundancy of its own; it is a **rearrangement** that makes the redundancy already
present in a [forward-error-correction](/reference/forward-error-correction/) code far more
effective, because FEC codes tolerate many *scattered* errors but only a *few* clustered ones.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 140" role="img" aria-label="Bits are written into a matrix by rows and read out by columns, so a contiguous burst of channel errors lands in different codewords after de-interleaving." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" stroke-width="1" fill="none">
    <rect x="50" y="30" width="120" height="80"/>
    <line x1="50" y1="50" x2="170" y2="50"/><line x1="50" y1="70" x2="170" y2="70"/><line x1="50" y1="90" x2="170" y2="90"/>
    <line x1="80" y1="30" x2="80" y2="110"/><line x1="110" y1="30" x2="110" y2="110"/><line x1="140" y1="30" x2="140" y2="110"/>
  </g>
  <text x="110" y="127" text-anchor="middle" font-size="8" fill="currentColor">write rows → read columns</text>
  <line x1="188" y1="70" x2="236" y2="70" stroke="currentColor" marker-end="url(#ilar)"/>
  <text x="360" y="56" text-anchor="middle" font-size="9" fill="currentColor">a contiguous burst</text>
  <text x="360" y="72" text-anchor="middle" font-size="9" fill="currentColor">becomes one error</text>
  <text x="360" y="88" text-anchor="middle" font-size="9" fill="currentColor">per codeword</text>
  <defs><marker id="ilar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>A block interleaver writes codewords into a matrix by rows and transmits by columns; a burst on the channel is then split so each codeword loses only a few bits it can correct.</figcaption>
</figure>

## How it works

The reasoning is a simple counting argument. Suppose an FEC code corrects up to *t* errors per
*N*-bit codeword, and the channel produces bursts up to *B* bits long. Sent one codeword at a
time, a single *B*-bit burst with *B > t* is uncorrectable and the codeword is lost. Interleave
*D* codewords together, though, and the same burst is diluted: consecutive channel bits now come
from *different* codewords, so a burst of *B* bits deposits at most `⌈B/D⌉` errors into any one
codeword. Choose the interleaving **depth** *D* large enough that `⌈B/D⌉ ≤ t` and every codeword
survives. The transmitter interleaves before sending; the receiver de-interleaves *before*
decoding, undoing the permutation so the FEC sees near-random errors, which is the regime its
correcting power is designed for.

## Variants

- **Block interleaver.** Write *R* codewords row-by-row into an *R×C* matrix, transmit
  column-by-column. Simple and exact, but it must fill and drain the whole block, so it adds
  latency proportional to the matrix size and protects only against bursts shorter than a column
  span. Best when data is naturally framed.
- **Convolutional interleaver.** Feed symbols through a bank of shift-register delay lines of
  increasing length, cycling across the lines. It achieves the same burst-spreading with roughly
  **half** the memory and half the end-to-end latency of a block interleaver of equal power, and
  it works on a continuous stream — which is why it appears in DVB and DSL.
- **Matrix / product interleaving.** Two-dimensional schemes (rows *and* columns) underpin
  product codes such as [BPTC](/reference/bptc/), where the interleave pattern is chosen so that a
  physical burst never lands entirely within one row or one column codeword.

## In practice

Interleaving's cost is **latency and memory**: deeper interleaving spreads longer bursts but
delays every bit by the time it takes to fill and drain the interleaver, so voice systems must
balance robustness against a delay budget of tens of milliseconds. The depth is therefore tuned
to the channel's expected fade duration at the symbol rate, not made arbitrarily large. A common
design rule is to size the depth so the longest expected [multipath](/reference/multipath-propagation/)
fade, measured in symbols, is broken across more codewords than the code has spare correcting
capacity.

## Relevance to SDR

Nearly every digital land-mobile protocol interleaves its bursts. [DMR](/reference/dmr/) uses a
block interleave inside [BPTC(196,96)](/reference/bptc/) so that the row/column Hamming passes see
scattered errors; [P25](/reference/project-25/), [TETRA](/reference/tetra/), and
[NXDN](/reference/nxdn/) all interleave their coded payloads; and broadcast systems from DAB to DVB
lean on convolutional interleaving against fading. GopherTrunk implements the matching
**de-interleavers** as an explicit step in each protocol's decode chain — it reverses the standard's
bit permutation before handing the data to the FEC decoder. Getting that permutation exactly right
is essential: an off-by-one de-interleave scrambles otherwise-perfect symbols into garbage, so it is
a frequent culprit when a new protocol decoder "sees" a signal but never validates a frame.

## Sources

[^wiki]: [Interleaving (data)](https://en.wikipedia.org/wiki/Interleaving_(data)) — Wikipedia, for reordering data to spread burst errors across codewords, and block vs convolutional interleavers.
[^burst]: [Burst error-correcting code](https://en.wikipedia.org/wiki/Burst_error-correcting_code) — Wikipedia, for the depth/latency trade-off and how interleaving converts bursts into correctable random errors.
