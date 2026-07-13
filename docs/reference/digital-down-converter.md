---
slug: digital-down-converter
title: Digital down-converter (DDC)
entry_type: term
category: sdr-dsp
description: A digital down-converter shifts a channel within a wideband IQ stream to baseband using a numerically controlled oscillator, then filters and decimates it for processing.
keywords: digital down converter, DDC, NCO, channelizer, mixing, decimation, polyphase
aka: [digital down-converter, DDC]
autolink: true
infobox:
  - { label: Type, value: DSP block }
  - { label: Does, value: Shift channel to baseband, filter, decimate }
  - { label: Uses, value: Numerically controlled oscillator }
see_also: [local-oscillator, numerically-controlled-oscillator, decimation, digital-filter, channelizer, cic-filter, iq-data, demodulation]
related_lessons:
  - { title: "Filtering & decimation", url: /learn/rf-sdr/filtering-decimation/ }
related_reading:
  - { title: "SDR Internals, Part 5: Tuning & channelization", url: /blog/deep-dives/sdr-internals-05-tuning-channelization/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Digital_down_converter
  - https://en.wikipedia.org/wiki/Numerically-controlled_oscillator
---

A **digital down-converter** (**DDC**) shifts a chosen channel within a wideband
[IQ](/reference/iq-data/) stream to baseband using a
[numerically controlled oscillator](/reference/numerically-controlled-oscillator/) (a
software [local oscillator](/reference/local-oscillator/)), then
[filters](/reference/digital-filter/) and [decimates](/reference/decimation/) it.[^wiki] It
is the software realisation of a mixer, IF filter, and rate reduction, all in arithmetic.

<figure class="figure" markdown="0">
<svg viewBox="0 0 520 110" role="img" aria-label="Wideband IQ into a numerically controlled oscillator mixer, then a low-pass filter, then decimation, producing one narrow channel." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <text x="40" y="63">wide IQ</text>
    <circle cx="120" cy="58" r="20" fill="none" stroke="currentColor" stroke-width="1.3"/><path d="M106 44 L134 72 M134 44 L106 72" stroke="currentColor" stroke-width="1.1"/><text x="120" y="96" font-size="8">NCO</text><line x1="120" y1="92" x2="120" y2="78" stroke="currentColor"/>
    <rect x="172" y="44" width="74" height="28" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="209" y="62">low-pass</text>
    <rect x="278" y="44" width="84" height="28" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="320" y="62">decimate</text>
    <rect x="394" y="44" width="96" height="28" rx="5" fill="currentColor" fill-opacity="0.15" stroke="currentColor" stroke-width="1.3"/><text x="442" y="62">one channel</text>
    <g stroke="currentColor" stroke-width="1.1"><line x1="70" y1="58" x2="100" y2="58"/><line x1="140" y1="58" x2="171" y2="58"/><line x1="246" y1="58" x2="277" y2="58"/><line x1="362" y1="58" x2="393" y2="58"/></g>
  </g>
</svg>
<figcaption>A digital down-converter shifts a channel to baseband (NCO), filters it, and decimates — the heart of channelising.</figcaption>
</figure>

## How it works

Three stages run in sequence. First, **mix**: an
[NCO](/reference/numerically-controlled-oscillator/) generates a complex tone rotating at
the negative of the channel's offset from the capture centre, and multiplying the IQ stream
by it slides that channel down to exactly 0 Hz. Because both operands are complex, the shift
is one-sided — the wanted channel moves to baseband while everything else moves too, harmlessly.
Second, **filter**: a low-pass [digital filter](/reference/digital-filter/) keeps only the
band now sitting around zero and rejects the neighbouring channels. Third, **decimate**:
with the bandwidth narrowed, [decimation](/reference/decimation/) drops the sample rate to
just what the single channel needs — often a large factor, since one 12.5 kHz channel lives
inside a multi-MHz capture. Filtering must precede decimation so the discarded neighbours
cannot [alias](/reference/aliasing/) back in.

## Variants

- **Per-channel DDC** — one NCO/filter/decimate chain per channel of interest. Simple and
  flexible; cost scales with the number of channels. This is the classic single-channel
  down-converter.
- **[Polyphase channelizer](/reference/channelizer/)** — when many equally spaced channels
  are wanted at once, a [polyphase filter bank](/reference/polyphase-filter-bank/) plus an
  FFT extracts them all far more cheaply than N independent DDCs.
- **[CIC](/reference/cic-filter/)-based DDC** — hardware and FPGA designs favour a
  multiplierless CIC filter for the bulk of the decimation, followed by a short FIR to
  correct its droop; the ubiquitous "DDC" building block in radio ASICs.

## In practice

GopherTrunk runs more than one flavour. Its wideband path feeds a multi-tap DDC *bank* that
pulls the control channel and several voice channels out of a single capture in parallel;
its offline replay path uses a separate single-channel down-converter with its own tuning
offset. The two are genuinely distinct code paths, so a fix to one does not touch the other —
a subtlety that has caused a bug to look "fixed" on the wideband path while still live on
replay. Both normalise their output to the per-protocol channel rate before demodulation.

## Relevance to SDR

The DDC is how GopherTrunk extracts a control channel and multiple voice channels from a
single wideband capture, and how any SDR "tunes" without moving the analog LO. It is the
last frequency-shifting stage before the [demodulator](/reference/demodulation/) sees a clean
baseband channel.

## Sources

[^wiki]: [Digital down converter](https://en.wikipedia.org/wiki/Digital_down_converter) — Wikipedia, for the NCO/filter/decimate channelization architecture.
[^nco]: [Numerically-controlled oscillator](https://en.wikipedia.org/wiki/Numerically-controlled_oscillator) — Wikipedia, on the digital tone generator that performs the DDC's frequency shift.
