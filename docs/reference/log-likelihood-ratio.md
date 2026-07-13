---
slug: log-likelihood-ratio
title: Log-likelihood ratio (LLR)
entry_type: term
category: sdr-dsp
description: "A log-likelihood ratio is the log of the probability a received bit is a 0 versus a 1; it is the standard soft-metric that feeds modern FEC decoders."
keywords: log-likelihood ratio, LLR, soft metric, soft bit, bit reliability, log ratio, soft-decision metric, channel LLR, a posteriori LLR, decoder input
aka: [LLR, log-likelihood ratio, soft-bit metric]
autolink: true
infobox:
  - { label: Symbol, value: "L(b) = ln[P(b=0)/P(b=1)]" }
  - { label: Sign, value: Hard bit decision }
  - { label: Magnitude, value: Reliability of that bit }
see_also: [soft-decision, bcjr-algorithm, ldpc-code, turbo-code, forward-error-correction, signal-to-noise-ratio]
cite_urls:
  - https://en.wikipedia.org/wiki/Log_probability
  - https://en.wikipedia.org/wiki/Soft-decision_decoder
---

**A log-likelihood ratio (LLR)** is the natural logarithm of the probability that a received bit
is a 0 divided by the probability it is a 1 — the calibrated
[soft-decision](/reference/soft-decision/) metric that modern
[forward-error-correction](/reference/forward-error-correction/) decoders consume.[^wiki] Its sign
is the hard decision (positive favors 0, negative favors 1) and its magnitude is the reliability:
an LLR near zero is a coin toss, while a large-magnitude LLR is a near-certain bit. Packing both
the decision and its confidence into one signed number is what makes LLRs the lingua franca of
iterative decoding.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="A number line of log-likelihood ratio values: large positive means a confident zero, near zero means uncertain, large negative means a confident one." xmlns="http://www.w3.org/2000/svg">
  <line x1="30" y1="75" x2="440" y2="75" stroke="currentColor" stroke-width="1.2" marker-start="url(#llrar)" marker-end="url(#llrar)"/>
  <line x1="235" y1="60" x2="235" y2="90" stroke="currentColor" stroke-width="1"/>
  <text x="235" y="106" font-size="8.5" fill="currentColor" text-anchor="middle">0 (uncertain)</text>
  <text x="70" y="60" font-size="8.5" fill="currentColor" text-anchor="middle">+ large</text>
  <text x="70" y="106" font-size="8.5" fill="currentColor" text-anchor="middle">confident "0"</text>
  <text x="400" y="60" font-size="8.5" fill="currentColor" text-anchor="middle">− large</text>
  <text x="400" y="106" font-size="8.5" fill="currentColor" text-anchor="middle">confident "1"</text>
  <text x="235" y="35" font-size="9" fill="currentColor" text-anchor="middle">L(b) = ln[ P(b=0) / P(b=1) ]</text>
  <defs><marker id="llrar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>The LLR axis: sign carries the bit decision, magnitude carries confidence, and zero is maximal uncertainty.</figcaption>
</figure>

## How it works

For a bit `b` and an observation `y`, the LLR is `L(b) = ln[ P(b=0 | y) / P(b=1 | y) ]`. The log
turns a probability *ratio* into a signed additive quantity, which is the property that makes it
so convenient: independent pieces of evidence about the same bit simply **add** in the LLR domain,
so a decoder combines the channel observation with parity-check evidence by summation rather than
by multiplying probabilities.

For the common case of BPSK-like symbols in additive white Gaussian noise, the math collapses to
a strikingly simple result: the channel LLR is `L = 2·y / σ²`, i.e. the matched-filter output `y`
scaled by twice the inverse noise variance. Two consequences follow. First, the LLR is just the
raw soft value with a known scale factor — which is why a coarse soft metric already works.
Second, the scale depends on the noise power, so **estimating the channel SNR** correctly sets the
LLR magnitudes; get it wrong and a decoder becomes over- or under-confident, which especially
hurts iterative codes.

Decoders come in flavors named for what they compute. A [BCJR](/reference/bcjr-algorithm/) (MAP)
decoder computes the exact *a posteriori* LLR for every bit; the practical **max-log-MAP**
approximation replaces the log-sum-exp with a max, trading a fraction of a dB for a big drop in
complexity. Iterative decoders pass **extrinsic** LLRs — the new information a check node produces
about a bit, excluding what it was already told — back and forth, refining the estimates each
round.

## In practice

- **Clipping** — LLRs are stored in fixed point and clamped to a maximum magnitude; too small a
  clip discards confident bits, too large wastes dynamic range. A few integer bits usually
  suffice.
- **Approximate demappers** — for higher-order QAM the exact per-bit LLR needs all constellation
  points, so implementations use the "min" (nearest-symbol) approximation per bit.
- **Sign-magnitude view** — many hardware decoders store an LLR as a sign bit plus a small
  magnitude, mirroring the intuition of a [soft bit](/reference/soft-decision/).

The additive property is worth dwelling on because it is what makes iterative decoding tractable.
In the probability domain, combining independent evidence about a bit means multiplying
likelihoods and renormalizing — awkward and numerically unstable. Taking the log converts those
products into sums, so a check node in an [LDPC](/reference/ldpc-code/) decoder or a component
decoder in a [turbo](/reference/turbo-code/) decoder simply adds and subtracts LLRs. It also keeps
the arithmetic well conditioned: a probability near 0 or 1, which would underflow in fixed point,
maps to a large but finite LLR magnitude that clips gracefully. This is the same reason
statisticians and machine-learning systems work in log-probabilities, and it is why the LLR, not
the raw probability, is the number that actually flows through a decoder.

## Relevance to SDR

LLRs are the input format of every high-performance FEC in use today:
[LDPC codes](/reference/ldpc-code/) in Wi-Fi, DVB-S2, and 5G data channels;
[turbo codes](/reference/turbo-code/) in LTE and UMTS; and polar codes in 5G control. Any SDR
receiver decoding these standards computes LLRs at the demapper. In land-mobile digital voice the
FEC is simpler (convolutional/trellis and block codes), so full LLR machinery is less central, but
the same principle — carry a signed reliability, not just a bit — underlies the
[soft-decision](/reference/soft-decision/) Viterbi paths that GopherTrunk and similar decoders use
to pull frames out of marginal signals.

## Sources

[^wiki]: [Log probability](https://en.wikipedia.org/wiki/Log_probability) — Wikipedia, on log-domain probability ratios and their additive combination, the basis of the LLR.
