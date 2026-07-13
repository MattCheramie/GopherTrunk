---
slug: maximum-likelihood-sequence-estimation
title: Maximum-likelihood sequence estimation (MLSE)
entry_type: algorithm
category: equalization
description: Maximum-likelihood sequence estimation (MLSE) uses the Viterbi algorithm over the channel trellis to find the most-likely transmitted symbol sequence, the optimal equalizer.
keywords: MLSE, maximum likelihood sequence estimation, Viterbi equalizer, channel trellis, intersymbol interference, GSM equalizer, optimal detection, Forney, branch metric
aka: [MLSE, maximum-likelihood sequence estimation, Viterbi equalizer]
autolink: true
infobox:
  - { label: Type, value: Optimal (nonlinear) sequence equalizer }
  - { label: Engine, value: Viterbi search over channel trellis }
  - { label: Cost, value: Exponential in channel memory }
see_also: [viterbi-algorithm, decision-feedback-equalizer, mmse-equalizer, matched-filter, gmsk, constellation-diagram]
cite_urls:
  - https://en.wikipedia.org/wiki/Maximum_likelihood_sequence_estimation
  - https://en.wikipedia.org/wiki/Viterbi_algorithm
---

**Maximum-likelihood sequence estimation (MLSE)** is the optimal way to combat
intersymbol interference: rather than filtering each symbol independently, it searches over
*all possible transmitted sequences* and picks the one most likely to have produced the
received waveform, given a known channel.[^wiki] Because the ISI channel has finite memory,
that search is carried out efficiently by the
[Viterbi algorithm](/reference/viterbi-algorithm/) over a **trellis** whose states are the
recent symbol history — which is why MLSE is often called a *Viterbi equalizer*.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 170" role="img" aria-label="A trellis of states across several time steps with many candidate branch paths in light lines and one heavy surviving path threading through, representing the most-likely sequence." xmlns="http://www.w3.org/2000/svg">
  <g fill="currentColor"><g font-size="9"><text x="30" y="15">states</text><text x="215" y="165" text-anchor="middle">time →</text></g></g>
  <g stroke="currentColor" stroke-width="0.7" stroke-opacity="0.35" fill="none">
    <path d="M70 40 L160 40 M70 40 L160 75 M70 75 L160 40 M70 75 L160 110 M70 110 L160 75 M70 110 L160 145 M70 145 L160 110 M70 145 L160 145"/>
    <path d="M160 40 L250 40 M160 75 L250 40 M160 75 L250 110 M160 110 L250 75 M160 110 L250 145 M160 145 L250 110 M160 40 L250 75"/>
    <path d="M250 40 L340 75 M250 75 L340 40 M250 110 L340 110 M250 145 L340 145 M250 40 L340 40 M250 110 L340 75"/>
  </g>
  <g stroke="currentColor" stroke-width="2" fill="none"><path d="M70 75 L160 40 L250 75 L340 110"/></g>
  <g fill="currentColor"><circle cx="70" cy="40" r="3"/><circle cx="70" cy="75" r="3"/><circle cx="70" cy="110" r="3"/><circle cx="70" cy="145" r="3"/><circle cx="160" cy="40" r="3"/><circle cx="160" cy="75" r="3"/><circle cx="160" cy="110" r="3"/><circle cx="160" cy="145" r="3"/><circle cx="250" cy="40" r="3"/><circle cx="250" cy="75" r="3"/><circle cx="250" cy="110" r="3"/><circle cx="250" cy="145" r="3"/><circle cx="340" cy="40" r="3"/><circle cx="340" cy="75" r="3"/><circle cx="340" cy="110" r="3"/><circle cx="340" cy="145" r="3"/></g>
</svg>
<figcaption>MLSE builds a trellis whose states encode recent symbol history; the Viterbi algorithm keeps the best path into each state and the heavy surviving path is the most-likely transmitted sequence.</figcaption>
</figure>

## How it works

The channel with memory `L` (its impulse response spans `L+1` symbols) acts like a
convolutional encoder over the transmitted symbols, so the received signal can be modelled
by a finite-state **trellis**:

- Each **state** is the last `L` symbols the channel "remembers." For an alphabet of `M`
  symbols there are `M^L` states.
- Each **branch** between states corresponds to one candidate new symbol and carries a
  **branch metric** — the squared distance between the received sample and the noiseless
  sample that symbol-plus-history *would* produce through the known channel.
- The [Viterbi algorithm](/reference/viterbi-algorithm/) accumulates these metrics,
  keeping only the single best (smallest-distance) surviving path into each state at each
  step. The path with the lowest total metric at the end is the maximum-likelihood
  sequence.

Minimising total squared distance to the received signal is exactly *maximising the
likelihood* under additive Gaussian noise — hence "maximum-likelihood sequence
estimation." A [matched filter](/reference/matched-filter/) (or whitened matched filter)
front end typically precedes the trellis search, and the channel estimate that defines the
branch metrics is measured from a known training/sync sequence.

## Cost and trade-offs

MLSE is *optimal* — no equalizer achieves a lower sequence error probability — but its
`M^L` state count grows **exponentially** with the channel memory and the modulation order.
For a binary signal over a 4-tap channel that is only 8 states, but for higher-order
constellations or long delay spreads it explodes, which is why practical systems bound `L`
(and often shorten the channel with a prefilter) or fall back on **reduced-state**
variants (DDFSE, RSSE) that prune the trellis. Where the channel is too long or unknown,
the nonlinear [DFE](/reference/decision-feedback-equalizer/) or a linear
[MMSE](/reference/mmse-equalizer/) equalizer trades some performance for far lower cost.

## Relevance to SDR

The classic deployment is **GSM**, whose receivers use a Viterbi equalizer (MLSE) with the
midamble training sequence to defeat the multipath ISI of the mobile channel — the same
Viterbi engine GSM already uses to decode its convolutional
[FEC](/reference/forward-error-correction/). MLSE also appears in magnetic-recording read
channels (PRML) and other severe-ISI links. It sharpens detection well beyond what a linear
equalizer manages when ISI is heavy. GopherTrunk's target systems (P25, DMR, NXDN) use
narrowband, [RRC-shaped](/reference/root-raised-cosine-filter/) modulations decoded via
[matched filtering](/reference/matched-filter/) and synchronisation rather than an MLSE
equalizer — though the [Viterbi algorithm](/reference/viterbi-algorithm/) itself is central
to their error-correction decoding — so MLSE is presented here as the optimal equalization
benchmark from the wider RF world.

## Sources

[^wiki]: [Maximum likelihood sequence estimation](https://en.wikipedia.org/wiki/Maximum_likelihood_sequence_estimation) — Wikipedia, on trellis-based ML detection over ISI channels and the Viterbi equalizer.
