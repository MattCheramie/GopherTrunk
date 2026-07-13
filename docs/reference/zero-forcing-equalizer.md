---
slug: zero-forcing-equalizer
title: Zero-forcing equalizer
entry_type: algorithm
category: equalization
description: A zero-forcing (ZF) equalizer inverts the channel response to null intersymbol interference exactly, at the cost of amplifying noise near spectral nulls.
keywords: zero forcing equalizer, ZF, channel inversion, intersymbol interference, ISI, noise enhancement, spectral null, linear equalizer, MMSE comparison
aka: [zero-forcing equalizer, ZF equalizer, channel inverse filter]
autolink: true
infobox:
  - { label: Type, value: Linear equalizer }
  - { label: Criterion, value: Force ISI to zero (invert channel) }
  - { label: Weakness, value: Noise enhancement at spectral nulls }
see_also: [mmse-equalizer, adaptive-filter, decision-feedback-equalizer, matched-filter, signal-to-noise-ratio, constellation-diagram]
cite_urls:
  - https://en.wikipedia.org/wiki/Zero-forcing_equalizer
  - https://en.wikipedia.org/wiki/Intersymbol_interference
---

A **zero-forcing (ZF) equalizer** is a linear filter that removes **intersymbol
interference (ISI)** by applying the *inverse* of the channel's frequency response, so the
cascade of channel and equalizer is flat and each symbol contributes nothing to its
neighbours.[^wiki] It is the most intuitive equalizer — literally undo the channel — but
inverting the channel also inverts and amplifies noise wherever the channel is weak,
which is its defining flaw.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 170" role="img" aria-label="Three frequency plots: a channel response with a deep null, the equalizer's inverse response with a tall peak at the same frequency, and the flat product, with noise shown boosted at the peak." xmlns="http://www.w3.org/2000/svg">
  <g fill="none" stroke="currentColor" font-size="9">
    <g><path d="M20 100 Q 60 40, 80 100 T 140 90" stroke-width="1.4"/><path d="M78 100 Q 85 120, 92 100" stroke-width="1.4"/><text x="80" y="130" text-anchor="middle" fill="currentColor">channel H(f)</text><text x="83" y="118" text-anchor="middle" fill="currentColor" font-size="8">null</text></g>
    <g transform="translate(160,0)"><path d="M20 90 Q 60 100, 78 100" stroke-width="1.4"/><path d="M78 100 Q 85 25, 92 100" stroke-width="1.4"/><path d="M92 100 Q 110 90, 140 92" stroke-width="1.4"/><text x="80" y="130" text-anchor="middle" fill="currentColor">1/H(f)</text><text x="85" y="42" text-anchor="middle" fill="currentColor" font-size="8">peak</text></g>
    <g transform="translate(320,0)"><line x1="20" y1="80" x2="120" y2="80" stroke-width="1.4"/><path d="M60 80 l -3 -6 M60 80 l 3 -6 M66 80 l -3 -6 M66 80 l 3 -6" stroke-width="1"/><text x="70" y="130" text-anchor="middle" fill="currentColor">flat — noise up</text></g>
  </g>
</svg>
<figcaption>ZF inverts the channel: a deep spectral null in H(f) becomes a tall peak in 1/H(f). The signal comes out flat, but noise at that frequency is boosted by the same peak.</figcaption>
</figure>

## How it works

Think of the channel as a filter `H(f)` that colours the transmitted spectrum and smears
symbols together. The ZF equalizer sets its own response to `C(f) = 1/H(f)`:

- The product `H(f)·C(f) = 1` is perfectly flat, so in the time domain the combined
  impulse response is a single spike — one symbol, zero tails, **ISI forced to zero**.
- With a finite-length filter, ZF instead forces the combined impulse response to zero at
  the sampling instants of neighbouring symbols (the Nyquist "no-ISI" condition), which is
  where the name comes from.
- The taps can be solved directly from an estimate of the channel, or learned adaptively
  by an [LMS](/reference/lms-algorithm/)/[RLS](/reference/rls-algorithm/) rule driven
  toward the zero-ISI condition.

## Noise enhancement

The catch is that ZF ignores noise entirely. Wherever `H(f)` is small — a deep fade or
spectral null from [multipath](/reference/multipath-propagation/) — `1/H(f)` is large, so
the equalizer applies enormous gain at exactly the frequencies where the received signal is
weakest and dominated by noise. It removes all ISI but can crater the output
[SNR](/reference/signal-to-noise-ratio/), badly scattering the
[constellation](/reference/constellation-diagram/) on channels with nulls. ZF therefore
performs acceptably only when the channel has no severe nulls and the SNR is high.

## Contrast with MMSE

The [MMSE equalizer](/reference/mmse-equalizer/) fixes this by minimising the *total*
mean-square error — residual ISI **plus** noise — rather than ISI alone. Its response is
roughly `H*/(‖H‖² + σ²)`, which stops short of full inversion near nulls and so avoids the
worst noise blow-up. At high SNR (`σ² → 0`) MMSE reduces to the ZF solution; at low SNR it
is substantially better. In both cases a nonlinear
[decision-feedback equalizer](/reference/decision-feedback-equalizer/) or an
[MLSE](/reference/maximum-likelihood-sequence-estimation/) receiver outperforms a purely
linear equalizer on severe channels.

## Relevance to SDR

Zero forcing is the textbook baseline for linear channel equalization and a useful mental
model for what any equalizer is trying to accomplish — flatten the channel. In real
receivers it is usually passed over in favour of MMSE, DFE, or MLSE precisely because of
its noise-enhancement penalty, but it remains common in high-SNR wireline contexts and as
a starting point for adaptive designs. GopherTrunk decodes narrowband, RRC-shaped
land-mobile signals with a [matched filter](/reference/matched-filter/) and synchronisation
rather than an explicit ZF equalizer, so ZF is described here as a canonical equalization
concept from the wider RF and communications field.

## Sources

[^wiki]: [Zero-forcing equalizer](https://en.wikipedia.org/wiki/Zero-forcing_equalizer) — Wikipedia, on channel inversion, the zero-ISI condition, and noise enhancement versus MMSE.
