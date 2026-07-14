---
slug: interleaved-iq
title: Interleaved IQ
entry_type: term
category: sdr-programming
description: Interleaved IQ is the memory layout that stores IQ samples as alternating I, Q, I, Q values in one contiguous array — the near-universal on-the-wire and on-disk format for SDR sample streams.
keywords: interleaved IQ, IQ interleaving, I Q I Q, complex samples layout, deinterleave, planar IQ, IQ memory layout, IQ file, complex array
aka: [interleaved IQ, "I/Q interleaving"]
autolink: true
infobox:
  - { label: Type, value: Sample memory layout }
  - { label: Order, value: "I, Q, I, Q, … in one array" }
  - { label: Alternative, value: Planar (separate I and Q arrays) }
see_also: [sample-format, iq-data, iq-file-format, sample-rate, sample-buffer]
cite_urls:
  - https://pysdr.org/content/iq_files.html
  - https://en.wikipedia.org/wiki/In-phase_and_quadrature_components
---

**Interleaved IQ** is the memory layout in which the two halves of each [IQ sample](/reference/iq-data/) are stored adjacently and consecutively — *I₀, Q₀, I₁, Q₁, I₂, Q₂, …* — packing a complex stream into a single flat array of real numbers.[^pysdr] It is the near-universal way SDR hardware delivers samples over USB or the network and the way IQ is laid out in files, so nearly every SDR program reads and writes this order at its boundaries.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="A single byte stream labeled I0 Q0 I1 Q1 I2 Q2 splitting downward into two separate arrays, one of I values and one of Q values, illustrating deinterleaving." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="iqar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <g font-size="11" fill="currentColor" text-anchor="middle">
    <g stroke="currentColor">
      <rect x="20" y="14" width="60" height="26" fill="currentColor" fill-opacity="0.22"/><text x="50" y="31">I0</text>
      <rect x="80" y="14" width="60" height="26" fill="none"/><text x="110" y="31">Q0</text>
      <rect x="140" y="14" width="60" height="26" fill="currentColor" fill-opacity="0.22"/><text x="170" y="31">I1</text>
      <rect x="200" y="14" width="60" height="26" fill="none"/><text x="230" y="31">Q1</text>
      <rect x="260" y="14" width="60" height="26" fill="currentColor" fill-opacity="0.22"/><text x="290" y="31">I2</text>
      <rect x="320" y="14" width="60" height="26" fill="none"/><text x="350" y="31">Q2</text>
    </g>
    <line x1="130" y1="42" x2="90" y2="86" stroke="currentColor" marker-end="url(#iqar)"/>
    <line x1="230" y1="42" x2="270" y2="86" stroke="currentColor" marker-end="url(#iqar)"/>
    <g stroke="currentColor">
      <rect x="30" y="90" width="50" height="24" fill="currentColor" fill-opacity="0.22"/><text x="55" y="106">I0</text>
      <rect x="80" y="90" width="50" height="24" fill="currentColor" fill-opacity="0.22"/><text x="105" y="106">I1</text>
      <rect x="130" y="90" width="50" height="24" fill="currentColor" fill-opacity="0.22"/><text x="155" y="106">I2</text>
      <rect x="250" y="90" width="50" height="24" fill="none"/><text x="275" y="106">Q0</text>
      <rect x="300" y="90" width="50" height="24" fill="none"/><text x="325" y="106">Q1</text>
      <rect x="350" y="90" width="50" height="24" fill="none"/><text x="375" y="106">Q2</text>
    </g>
    <text x="105" y="134">I array</text><text x="325" y="134">Q array</text>
  </g>
</svg>
<figcaption>Interleaved on the wire (I, Q, I, Q); some DSP deinterleaves into separate I and Q (planar) arrays for vectorized processing.</figcaption>
</figure>

## How it works

Interleaving is just a serialization order. Each complex sample is two real numbers, and interleaved layout writes them in the sequence they were captured, alternating I then Q, using whatever [sample format](/reference/sample-format/) the stream carries — so a CS16 interleaved buffer is `int16 I, int16 Q, int16 I, …`, and a CF32 `.cfile` is `float32 I, float32 Q, …`. To recover complex numbers a program strides through the array two elements at a time: element *2k* is the I of sample *k* and element *2k+1* is its Q.

The chief alternative is **planar** (or "split") layout, where all the I values sit in one array and all the Q values in another. Planar is friendlier to [SIMD](/reference/vectorization-simd/) kernels that want to load a run of same-component values, and some math libraries expect it. **Deinterleaving** converts interleaved input into planar buffers, and re-interleaving does the reverse on the way out. Because both orders describe the identical samples, the choice is purely about which downstream code runs fastest.

## Relevance to SDR

Interleaved IQ is what crosses almost every SDR boundary: the USB bulk transfer from an RTL-SDR, the UDP payload from a networked radio, and the bytes of a saved capture are all interleaved. That makes the interleaved buffer the natural unit for a [sample buffer](/reference/sample-buffer/) or ring, and the interleaving convention part of any [IQ file format](/reference/iq-file-format/) definition — the header (or external metadata) has to state the sample format *and* that the data is interleaved for a reader to make sense of it. A subtle but real bug class is a half-sample misalignment that swaps every I and Q, rotating the whole constellation by 90°; it comes from reading an interleaved stream at the wrong byte offset.

**GopherTrunk**, as a pure-Go SDR application, ingests interleaved IQ throughout. Its device drivers hand up interleaved sample buffers, and its `.cfile` replay reads interleaved complex-float32 data straight from disk. GT treats an interleaved buffer as its working unit, converting the element pairs into the complex values the demodulator consumes at the front of the chain. Keeping the layout explicit at these boundaries is what lets the same decode path serve both live hardware and recorded files.

## In practice

When wiring up a new source or file, pin down three things before trusting a single sample: the [sample format](/reference/sample-format/) (byte width and signedness), the interleave order (essentially always I-first), and any per-sample padding. Get all three right and the [sample rate](/reference/sample-rate/) alone tells you how to slice time; get the interleave stride wrong and every downstream measurement is quietly corrupt.

## Sources

[^pysdr]: [IQ files and SigMF](https://pysdr.org/content/iq_files.html) — PySDR, on interleaved I/Q storage and reading complex samples from a flat array.
