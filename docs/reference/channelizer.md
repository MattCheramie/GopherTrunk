---
slug: channelizer
title: Channelizer
entry_type: algorithm
category: filtering-multirate
description: A channelizer splits one wideband IQ capture into many narrow channels at once — how a single SDR follows a control channel and several voice channels in parallel.
keywords: channelizer, channelization, polyphase filter bank, FFT channelizer, wideband, multi-channel, DDC, parallel decode, filter bank
aka: [channelizer, channelization, polyphase channelizer]
autolink: true
infobox:
  - { label: Type, value: Multi-channel filter bank }
  - { label: Splits, value: One wide band → many narrow channels }
  - { label: Efficient form, value: Polyphase + FFT }
see_also: [polyphase-filter-bank, digital-down-converter, decimation, fast-fourier-transform, control-channel]
related_lessons:
  - { title: "Filtering & decimation", url: /learn/rf-sdr/filtering-decimation/ }
related_reading:
  - { title: "SDR Internals, Part 5: Tuning & channelization", url: /blog/deep-dives/sdr-internals-05-tuning-channelization/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Filter_bank
  - https://en.wikipedia.org/wiki/Polyphase_matrix
---

A **channelizer** splits a single wideband [IQ](/reference/iq-data/) capture into **many
narrow channels at once**.[^wiki] Each output is one channel, shifted to
[baseband](/reference/baseband/), [filtered](/reference/digital-filter/), and
[decimated](/reference/decimation/) to its own low rate — so one SDR front end can feed a
whole rack of decoders in parallel. In a trunking scanner this is the difference between
watching one frequency and watching an entire system's control and voice traffic
simultaneously.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 140" role="img" aria-label="A single wide IQ band enters a channelizer block and fans out into several separate narrow channel outputs — a control channel and two voice channels — each ready for its own decoder." xmlns="http://www.w3.org/2000/svg">
  <rect x="18" y="52" width="90" height="30" rx="5" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.2"/><text x="63" y="71" text-anchor="middle" font-size="9" fill="currentColor">wide IQ</text>
  <rect x="172" y="44" width="96" height="46" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="220" y="64" text-anchor="middle" font-size="9" fill="currentColor">channelizer</text><text x="220" y="78" text-anchor="middle" font-size="7.5" fill="currentColor">polyphase + FFT</text>
  <line x1="108" y1="67" x2="171" y2="67" stroke="currentColor" marker-end="url(#char)"/>
  <g font-size="8.5" fill="currentColor" text-anchor="middle">
    <rect x="338" y="22" width="104" height="22" rx="4" fill="none" stroke="currentColor" stroke-width="1.1"/><text x="390" y="37">control ch</text>
    <rect x="338" y="56" width="104" height="22" rx="4" fill="none" stroke="currentColor" stroke-width="1.1"/><text x="390" y="71">voice ch 1</text>
    <rect x="338" y="90" width="104" height="22" rx="4" fill="none" stroke="currentColor" stroke-width="1.1"/><text x="390" y="105">voice ch 2</text>
  </g>
  <g stroke="currentColor" stroke-width="1"><line x1="268" y1="60" x2="337" y2="33" marker-end="url(#char)"/><line x1="268" y1="67" x2="337" y2="67" marker-end="url(#char)"/><line x1="268" y1="74" x2="337" y2="101" marker-end="url(#char)"/></g>
  <defs><marker id="char" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>A channelizer fans one wide capture into many narrow channels at once, so a single SDR can decode a control channel and its assigned voice channels in parallel.</figcaption>
</figure>

## How it works

The brute-force way to extract *N* channels is to run *N* independent
[digital down-converters](/reference/digital-down-converter/): mix each channel to
[baseband](/reference/baseband/) with its own oscillator, low-pass filter, and decimate. That
works, but the cost scales with *N* — every channel pays for its own full-rate mixer and
filter, and most of that arithmetic is spent computing samples that are immediately thrown
away by the decimation.

An **efficient channelizer** collapses all of that shared work. When the channels are
uniformly spaced, three observations combine:

- The per-channel low-pass filters are the same prototype, just frequency-shifted — so their
  taps can be shared rather than replicated.
- Because each output is decimated by the number of channels, most of the filtering
  arithmetic is redundant. Reorganising the prototype filter into a
  [polyphase filter bank](/reference/polyphase-filter-bank/) does only the multiplies whose
  results survive decimation.
- The bank of frequency shifts — one complex exponential per channel — is exactly a
  Discrete Fourier Transform of the polyphase outputs, so a single
  [FFT](/reference/fast-fourier-transform/) produces **all** channels in one shot.

The result is the **polyphase FFT channelizer**: one shared prototype filter plus one FFT per
input block yields every channel simultaneously, at roughly the cost of a single filter plus a
`log N` FFT instead of *N* separate down-converters. The saving grows with the channel count,
which is what makes wideband, many-channel monitoring practical on an ordinary CPU.

## Variants

- **Maximally decimated ("critically sampled").** Output rate equals channel spacing; most
  efficient, but adjacent-channel energy can alias unless the prototype is designed carefully.
- **Oversampled / M:N channelizer.** Decimates by less than the channel count, leaving guard
  room so channels don't overlap at the edges — needed when the wanted signals don't sit
  exactly on the channel grid.
- **Non-uniform / per-tap DDC.** When only a few channels are wanted, or they sit at arbitrary
  offsets, individual [DDCs](/reference/digital-down-converter/) can be cheaper than a full
  bank — a channelizer wins when many uniformly-spaced channels are needed at once.

## In practice

The channelizer is the front end that turns a wideband SDR into a multi-channel receiver:
spectrum monitoring, multi-carrier base stations, and trunking scanners all rely on it. The
design choice is always the same trade — a full polyphase FFT bank when you want *everything*
in a band, versus a few targeted DDCs when you want only a handful of channels.

## Relevance to SDR

Channelization is central to trunking with [SDR](/reference/software-defined-radio/): a single
capture must yield the [control channel](/reference/control-channel/) plus every
[voice channel](/reference/voice-channel/) it assigns. GopherTrunk down-converts and decimates
the wideband capture into the individual channels it needs to decode, so it can keep the
control channel locked while simultaneously following the voice calls that control channel
grants — the practical payoff of channelization.

## Sources

[^wiki]: [Filter bank](https://en.wikipedia.org/wiki/Filter_bank) — Wikipedia, on filter banks and the polyphase channelizer that splits a band into channels.
[^poly]: [Polyphase matrix](https://en.wikipedia.org/wiki/Polyphase_matrix) — Wikipedia, on the polyphase decomposition underlying the efficient FFT channelizer.
