---
slug: matched-filter
title: Matched filter
entry_type: algorithm
category: filtering-multirate
description: A matched filter is the optimal linear filter for maximising signal-to-noise ratio against a known pulse shape, used in receivers to sharpen symbol detection and correlation.
keywords: matched filter, optimal filter, SNR, correlation, pulse shape, detection, time-reversed, white noise, template, correlator
aka: [matched filter, correlation filter, correlator]
autolink: true
infobox:
  - { label: Type, value: Optimal detection filter }
  - { label: Maximises, value: SNR at the sampling instant }
  - { label: Form, value: Time-reversed copy of the pulse }
see_also: [root-raised-cosine-filter, barker-code, maximal-length-sequence, signal-to-noise-ratio, demodulation]
related_lessons:
  - { title: "Filtering & decimation", url: /learn/rf-sdr/filtering-decimation/ }
related_reading:
  - { title: "SDR Internals, Part 4: DSP foundations — filters, NCO & AGC", url: /blog/deep-dives/sdr-internals-04-dsp-foundations-filters-nco-agc/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Matched_filter
  - https://en.wikipedia.org/wiki/Correlation
---

A **matched filter** is the linear filter that **maximises the
[signal-to-noise ratio](/reference/signal-to-noise-ratio/)** at a chosen sampling instant
when the shape of the wanted signal is known and the noise is white.[^wiki] Its impulse
response is simply a **time-reversed, conjugated copy** of the pulse it is matched to, and it
is the single most important filter in a digital receiver: every symbol decision is only as
good as the SNR the matched filter delivers to the slicer.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 140" role="img" aria-label="A noisy, hard-to-read input on the left; after matched filtering, a single sharp correlation peak rises well above the noise on the right, marking where the known pulse occurred." xmlns="http://www.w3.org/2000/svg">
  <path d="M20 70 L40 63 L55 80 L72 58 L90 82 L108 64 L126 76 L144 60 L162 80 L180 66 L198 74" fill="none" stroke="currentColor" stroke-width="1.3" stroke-opacity="0.6"/>
  <text x="108" y="108" text-anchor="middle" font-size="9" fill="currentColor">noisy input (known pulse buried in noise)</text>
  <line x1="215" y1="66" x2="258" y2="66" stroke="currentColor" marker-end="url(#mfar)"/><text x="236" y="58" text-anchor="middle" font-size="8" fill="currentColor">matched filter</text>
  <line x1="278" y1="92" x2="440" y2="92" stroke="currentColor" stroke-opacity="0.3"/>
  <path d="M278 90 L348 88 L360 30 L372 88 L440 90" fill="none" stroke="currentColor" stroke-width="1.8"/>
  <circle cx="360" cy="30" r="2.6" fill="currentColor"/>
  <text x="360" y="112" text-anchor="middle" font-size="9" fill="currentColor">sharp correlation peak → sample here</text>
  <defs><marker id="mfar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>A matched filter concentrates a known pulse's energy into one sharp peak, lifting it above the noise so the detector knows exactly when — and whether — the pulse arrived.</figcaption>
</figure>

## How it works

Filtering with a time-reversed copy of the target pulse is mathematically identical to
**correlating** the input against that pulse: at each instant the filter output measures how
well the incoming samples line up with the template. When the wanted pulse is present and
aligned, all of its energy adds coherently into a single tall peak, while white noise — which
does not correlate with the template — averages toward zero. The result is the largest
possible ratio of peak signal power to noise power at the sampling instant, and no linear
filter can do better against white noise. This is why the matched filter is called *optimal*:
it is a provable, not heuristic, result.

Two ways of seeing the same operation:

- **Filter view** — convolve the received signal with `h(t) = s(T − t)`, the flipped pulse.
  Convolving with a flipped template is exactly correlation.
- **Correlator view** — multiply the input by the stored template and integrate over the
  symbol. For a rectangular pulse this is a plain "integrate-and-dump"; for a shaped pulse it
  is the shaped correlation.

The peak's *height* gives the detection statistic (is the symbol a one or a zero, is the sync
word present) and its *location in time* gives timing — which is why matched filtering
underpins both symbol slicing and frame synchronisation.

## Variants: symbol shaping vs. sequence detection

For linearly-modulated data the matched filter is the receive pulse-shaping filter. When the
transmitter uses [root-raised-cosine](/reference/root-raised-cosine-filter/) shaping, the
receiver's RRC *is* the matched filter — the same square-root split that gives zero ISI also
gives optimal SNR, which is the elegant reason RRC is split symmetrically across the link.

For **detection of a known sequence** — a preamble, sync word, or spreading code — the
"pulse" is an entire code word and the matched filter is a **correlator** against it. This is
how a receiver finds a [Barker code](/reference/barker-code/) preamble or despreads a
[maximal-length sequence](/reference/maximal-length-sequence/): correlate against the known
code and watch for the peak. Radar reuses the identical idea (pulse compression), correlating
the echo against the transmitted chirp to trade a long, low-power pulse for fine range
resolution.

## In practice

The matched filter is where receiver sensitivity is won or lost. Its benefit is greatest for
weak signals and long, structured pulses; its one prerequisite is that you actually know the
pulse or code shape, so it applies to the deterministic parts of a waveform (shaping,
preambles, sync, spreading codes) rather than to the random payload itself.

## Relevance to SDR

Matched filtering is a standard receive step in essentially every digital
[SDR](/reference/software-defined-radio/) decode chain, improving
[demodulation](/reference/demodulation/) of weak signals and enabling reliable frame sync.
GopherTrunk applies receive matched/pulse-shaping filters for the linear modes it decodes and
uses correlation against known sync patterns to find frame boundaries in the trunking control
and voice streams it follows.

## Sources

[^wiki]: [Matched filter](https://en.wikipedia.org/wiki/Matched_filter) — Wikipedia, on the SNR-optimal time-reversed correlation filter and its derivation for white noise.
[^corr]: [Correlation](https://en.wikipedia.org/wiki/Correlation) — Wikipedia, on the correlation operation that a matched filter implements against a known template.
