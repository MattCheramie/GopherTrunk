---
slug: decimation
title: Decimation
entry_type: term
category: sdr-dsp
description: Decimation reduces a signal's sample rate by keeping every Nth sample after low-pass filtering, shrinking the data rate once a channel has been isolated.
keywords: decimation, downsampling, sample rate reduction, filter then decimate, CIC, polyphase, CPU
aka: [decimation]
autolink: true
infobox:
  - { label: Type, value: DSP operation }
  - { label: Does, value: Lower sample rate after filtering }
  - { label: Order, value: Filter, then decimate }
see_also: [digital-filter, cic-filter, half-band-filter, polyphase-filter-bank, sample-rate, aliasing, oversampling, digital-down-converter]
related_lessons:
  - { title: "Filtering & decimation", url: /learn/rf-sdr/filtering-decimation/ }
related_reading:
  - { title: "SDR Internals, Part 5: Tuning & channelization", url: /blog/deep-dives/sdr-internals-05-tuning-channelization/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Downsampling_(signal_processing)
  - https://en.wikipedia.org/wiki/Cascaded_integrator%E2%80%93comb_filter
---

**Decimation** reduces a signal's [sample rate](/reference/sample-rate/) by keeping only
every Nth sample, after a low-pass [filter](/reference/digital-filter/) removes the
frequencies that would otherwise [alias](/reference/aliasing/).[^wiki] The integer N is the
decimation factor; the pair "low-pass then downsample" is treated as one operation.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 120" role="img" aria-label="A dense row of samples reduced to a sparse row by keeping every fourth sample after filtering." xmlns="http://www.w3.org/2000/svg">
  <g fill="currentColor"><circle cx="30" cy="40" r="3"/><circle cx="55" cy="40" r="3"/><circle cx="80" cy="40" r="3"/><circle cx="105" cy="40" r="3"/><circle cx="130" cy="40" r="3"/><circle cx="155" cy="40" r="3"/><circle cx="180" cy="40" r="3"/><circle cx="205" cy="40" r="3"/><circle cx="230" cy="40" r="3"/><circle cx="255" cy="40" r="3"/><circle cx="280" cy="40" r="3"/><circle cx="305" cy="40" r="3"/></g>
  <text x="350" y="44" font-size="9" fill="currentColor">high rate</text>
  <line x1="160" y1="55" x2="160" y2="78" stroke="currentColor" marker-end="url(#dcar)"/><text x="200" y="72" font-size="9" fill="currentColor">keep every 4th (÷4)</text>
  <g fill="currentColor"><circle cx="30" cy="95" r="3"/><circle cx="130" cy="95" r="3"/><circle cx="230" cy="95" r="3"/><circle cx="330" cy="95" r="3"/></g>
  <text x="370" y="99" font-size="9" fill="currentColor">low rate</text>
  <defs><marker id="dcar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>Decimation lowers the sample rate after filtering, cutting the data the rest of the pipeline must process.</figcaption>
</figure>

## How it works

Downsampling by N lowers the sample rate to Fs/N, which drops the Nyquist frequency to
Fs/2N. Any energy that was between the new and old Nyquist limits does not disappear — it
**folds** down onto the surviving band as an alias. So the order is non-negotiable: **filter
first, then decimate**. The low-pass stage confines the signal to the new, narrower band
before the samples are thrown away, so nothing folds back. Skipping or under-designing that
filter is one of the most common DSP mistakes, and it manifests as noise or phantom carriers
that no later stage can remove.

Once a narrow channel has been isolated, decimating cuts the workload of everything
downstream in direct proportion — halve the rate and you halve the multiply-adds per second
in the demodulator, the timing recovery, and the decoder. This is why decimation, not raw
CPU speed, is what makes running many channels from one wide capture practical.

## Variants

- **Cascaded (multistage) decimation** — a large factor is split into smaller stages (e.g.
  ÷2 several times). Each early stage runs a short filter at a high rate and a heavier filter
  runs last at a low rate, drastically cutting total computation.
- **[Half-band filters](/reference/half-band-filter/)** — the efficient choice for ÷2
  stages: nearly half their coefficients are zero.
- **[CIC filter](/reference/cic-filter/)** — a multiplierless integrator-comb structure for
  very large decimation factors in FPGAs and radio ASICs; needs a corrective FIR afterward to
  flatten its passband droop.
- **[Polyphase](/reference/polyphase-filter-bank/) decimation** — restructures the filter so
  it only computes the output samples that are kept, never wasting work on discarded ones.

## In practice

Non-integer rate changes need a [resampler](/reference/resampler/) (interpolate, filter,
decimate), since plain decimation only divides by an integer. In GopherTrunk decimation lives
inside the [digital down-converter](/reference/digital-down-converter/): after the NCO shifts
a channel to baseband, the signal is low-pass filtered and decimated down to the per-protocol
channel rate (48 kHz for 4800-baud C4FM, 144 kHz for TETRA), so the demodulator always runs
at a fixed, low rate regardless of the capture rate.

## Relevance to SDR

Decimation is what makes running [many channels](/reference/digital-down-converter/) from one
capture computationally feasible, and it is the reason GopherTrunk's decode path is
rate-invariant to the radio's capture rate.

## Sources

[^wiki]: [Decimation (signal processing)](https://en.wikipedia.org/wiki/Downsampling_(signal_processing)) — Wikipedia, on filter-then-downsample and anti-alias requirements.
[^cic]: [Cascaded integrator–comb filter](https://en.wikipedia.org/wiki/Cascaded_integrator%E2%80%93comb_filter) — Wikipedia, on the multiplierless structure used for large decimation factors.
