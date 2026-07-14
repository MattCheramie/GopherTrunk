---
slug: receiver-chain
title: Receiver chain
entry_type: concept
category: sdr-app-building
description: A receiver chain is the ordered pipeline of DSP stages — tune, filter, resample, demodulate, synchronise, decode — that carries raw IQ samples through to decoded data.
keywords: receiver chain, RX chain, DSP pipeline, receive pipeline, tune filter demod decode, signal chain, SDR pipeline stages, symbol recovery, deframing
aka: [RX chain, receive chain, receiver pipeline]
autolink: true
infobox:
  - { label: Type, value: DSP pipeline }
  - { label: Stages, value: "Tune → filter → resample → demod → sync → decode" }
  - { label: Property, value: Rate-invariant decode }
see_also: [software-defined-receiver, digital-down-converter, demodulation, clock-recovery, deframing, frame-synchronization]
cite_urls:
  - https://en.wikipedia.org/wiki/Software-defined_radio
  - https://en.wikipedia.org/wiki/Digital_down_converter
---

A **receiver chain** is the ordered pipeline of digital-signal-processing stages a
[software-defined receiver](/reference/software-defined-receiver/) runs to carry raw
[IQ](/reference/iq-data/) samples through to decoded data: **tune → filter → resample →
demodulate → synchronise → decode**.[^wiki] Each stage consumes the output of the one before
it and hands a cleaner, more structured stream to the next, so a defect early in the chain
(a poorly-placed filter, a wrong resample ratio) caps the quality of everything downstream.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 90" role="img" aria-label="Six pipeline blocks in a row — tune, filter, resample, demod, sync, decode — with arrows carrying the signal from raw IQ on the left to bits on the right." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="rxchar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <text x="10" y="48" font-size="7" fill="currentColor">IQ</text>
  <g font-size="7" fill="currentColor" text-anchor="middle">
    <rect x="28" y="34" width="52" height="24" rx="3" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="54" y="49">tune</text>
    <rect x="92" y="34" width="52" height="24" rx="3" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="118" y="49">filter</text>
    <rect x="156" y="34" width="56" height="24" rx="3" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="184" y="49">resample</text>
    <rect x="224" y="34" width="52" height="24" rx="3" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="250" y="49">demod</text>
    <rect x="288" y="34" width="52" height="24" rx="3" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="314" y="49">sync</text>
    <rect x="352" y="34" width="52" height="24" rx="3" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="378" y="49">decode</text>
  </g>
  <g stroke="currentColor" stroke-width="1">
    <line x1="80" y1="46" x2="91" y2="46" marker-end="url(#rxchar)"/>
    <line x1="144" y1="46" x2="155" y2="46" marker-end="url(#rxchar)"/>
    <line x1="212" y1="46" x2="223" y2="46" marker-end="url(#rxchar)"/>
    <line x1="276" y1="46" x2="287" y2="46" marker-end="url(#rxchar)"/>
    <line x1="340" y1="46" x2="351" y2="46" marker-end="url(#rxchar)"/>
    <line x1="404" y1="46" x2="440" y2="46" marker-end="url(#rxchar)"/>
  </g>
  <text x="426" y="40" font-size="7" fill="currentColor">bits</text>
</svg>
<figcaption>The receiver chain: each stage refines the stream, and its output quality bounds every stage after it.</figcaption>
</figure>

## How it works

Reading left to right, each stage has one job:

- **Tune.** A [digital down converter](/reference/digital-down-converter/) mixes the wanted
  channel to 0 Hz using a numerically-controlled oscillator, so the signal of interest sits
  at [baseband](/reference/baseband/).
- **Filter.** A low-pass [FIR filter](/reference/fir-filter/) rejects adjacent channels and
  the noise outside the signal's bandwidth — often the same block as the tune step (a
  [frequency-translating FIR](/reference/frequency-xlating-fir/)).
- **Resample.** [Decimation](/reference/decimation/) drops the rate to a small multiple of
  the symbol rate. This both cuts the arithmetic and sizes the matched filter and timing loop.
- **Demodulate.** The [demodulator](/reference/demodulation/) recovers the modulating
  quantity — instantaneous frequency for C4FM/FSK, phase for PSK — as a symbol-bearing waveform.
- **Synchronise.** [Clock recovery](/reference/clock-recovery/) finds the symbol instants and
  slices the waveform; a correlator then locates the sync word to establish frame timing.
- **Decode.** [Deframing](/reference/deframing/) and forward-error-correction turn aligned
  symbols into validated bytes and messages.

## In practice

A well-built chain fixes its internal rates so the decode stages are **rate-invariant** to the
capture rate: the resample step normalises every input to a per-protocol channel rate, and the
demodulator, matched filter, and clock loop are all sized from *that* rate, not the SDR's. A
symptom that appears only at a higher capture rate but reproduces in offline replay therefore
points at the captured samples (front-end overload, phase noise), not the steady-state chain.

## Relevance to SDR

The receiver chain is the skeleton of every SDR decoding application, and building one well is
mostly about getting the *order* and the *rates* right. **GopherTrunk** is a working example:
`internal/scanner/ccdecoder` implements exactly this pipeline — a `Downconverter` that tunes
and decimates to a fixed 48 kHz channel rate (144 kHz for TETRA), per-protocol demodulators,
`internal/dsp/sync` for clock and frame recovery, and per-radio deframers under
`internal/radio`. The chain is validated end-to-end from recorded IQ files with no hardware
attached, which is only possible because each stage is deterministic software.

## Sources

[^wiki]: [Software-defined radio](https://en.wikipedia.org/wiki/Software-defined_radio) — Wikipedia, on the staged software pipeline that processes sampled RF.
[^ddc]: [Digital down converter](https://en.wikipedia.org/wiki/Digital_down_converter) — Wikipedia, on the tune-filter-decimate front of a receive chain.
