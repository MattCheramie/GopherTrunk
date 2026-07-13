---
slug: bcjr-algorithm
title: BCJR algorithm
entry_type: algorithm
category: error-correction
description: BCJR is a MAP forward–backward decoder on a trellis that outputs per-bit soft reliability (LLRs); it is the engine inside turbo decoding, contrasting Viterbi's ML sequence decoding.
keywords: BCJR algorithm, MAP decoding, maximum a posteriori, forward-backward, soft-output, log-likelihood ratio, LLR, trellis, turbo code, BCJR vs Viterbi
aka: [BCJR algorithm, MAP decoder, forward-backward algorithm, Bahl-Cocke-Jelinek-Raviv]
autolink: true
infobox:
  - { label: Type, value: Soft-output MAP decoder }
  - { label: Produces, value: Per-bit log-likelihood ratios (LLRs) }
  - { label: Used by, value: Turbo codes (iterative decoding) }
see_also: [turbo-code, convolutional-code, viterbi-algorithm, forward-error-correction, trellis-coded-modulation, ldpc-code]
cite_urls:
  - https://en.wikipedia.org/wiki/BCJR_algorithm
  - https://ieeexplore.ieee.org/document/1055186
---

The **BCJR algorithm** is a soft-output trellis decoder that computes, for every information
bit, the **maximum a posteriori (MAP)** probability that the bit was 0 or 1 given the entire
received sequence.[^wiki] It does this with a **forward–backward** sweep over the code's
trellis and reports each decision as a **log-likelihood ratio (LLR)** — a signed confidence
value rather than a hard 0/1.[^bcjr] That soft output is exactly what makes iterative decoders
like [turbo codes](/reference/turbo-code/) work, and it is the main way BCJR differs from the
[Viterbi algorithm](/reference/viterbi-algorithm/), which finds one most-likely *sequence*.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 160" role="img" aria-label="A trellis swept forward with alpha probabilities and backward with beta probabilities; combined with branch metrics gamma they produce a per-bit log-likelihood ratio for each stage." xmlns="http://www.w3.org/2000/svg">
  <defs>
    <marker id="bjf" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker>
  </defs>
  <g fill="currentColor"><circle cx="50" cy="45" r="3"/><circle cx="50" cy="100" r="3"/><circle cx="170" cy="45" r="3"/><circle cx="170" cy="100" r="3"/><circle cx="290" cy="45" r="3"/><circle cx="290" cy="100" r="3"/><circle cx="410" cy="45" r="3"/><circle cx="410" cy="100" r="3"/></g>
  <g stroke="currentColor" stroke-opacity="0.3" stroke-width="1"><line x1="50" y1="45" x2="170" y2="45"/><line x1="50" y1="45" x2="170" y2="100"/><line x1="50" y1="100" x2="170" y2="45"/><line x1="50" y1="100" x2="170" y2="100"/><line x1="170" y1="45" x2="290" y2="45"/><line x1="170" y1="45" x2="290" y2="100"/><line x1="170" y1="100" x2="290" y2="45"/><line x1="170" y1="100" x2="290" y2="100"/><line x1="290" y1="45" x2="410" y2="45"/><line x1="290" y1="45" x2="410" y2="100"/><line x1="290" y1="100" x2="410" y2="45"/><line x1="290" y1="100" x2="410" y2="100"/></g>
  <line x1="30" y1="25" x2="140" y2="25" stroke="currentColor" stroke-width="1.4" marker-end="url(#bjf)"/>
  <text x="80" y="20" font-size="9" fill="currentColor">forward α</text>
  <line x1="430" y1="128" x2="320" y2="128" stroke="currentColor" stroke-width="1.4" marker-end="url(#bjf)"/>
  <text x="360" y="145" font-size="9" fill="currentColor">backward β</text>
  <text x="230" y="150" text-anchor="middle" font-size="9" fill="currentColor">LLR(bit) = log P(bit=1) − log P(bit=0), from α · γ · β at each stage</text>
</svg>
<figcaption>BCJR combines a forward pass (α), a backward pass (β), and branch metrics (γ) to give a soft reliability for every bit, not just one winning path.</figcaption>
</figure>

## How it works

BCJR treats the encoder as a Markov process moving through trellis states and asks, for each
information bit, how probable each value is after weighing *all* paths consistent with the
received signal — not just the single best one. It computes three quantities:

- **γ (gamma)** — a *branch metric* for each trellis transition, from the channel likelihood
  of the received symbol plus any prior information about the bit.
- **α (alpha)** — a *forward* recursion accumulating the probability of reaching each state
  from the start of the block.
- **β (beta)** — a *backward* recursion accumulating the probability of reaching the end from
  each state.

Multiplying α, γ, and β for the transitions that carry a given bit value, and summing over
states, gives the total probability of that value. The **LLR** is the log of the ratio between
the two totals. In practice the recursions are done in the log domain (log-MAP, or the
lower-complexity max-log-MAP approximation) to replace multiplications with additions and
avoid underflow.

The crucial output is not the hard bit but the *magnitude and sign* of the LLR: sign is the
decision, magnitude is the confidence.

## BCJR vs Viterbi

Both run on the same trellis, but they optimise different things. Viterbi is a
**maximum-likelihood sequence** decoder — it returns the single most probable path and, in
its classic form, a hard bit stream. BCJR is a **bit-wise MAP** decoder — it minimises the
per-bit error probability and emits soft reliabilities. Viterbi is cheaper (one forward pass,
survivor selection); BCJR costs a forward *and* a backward pass plus more arithmetic. You pay
that cost when you need soft output to feed *another* decoder, which is precisely the turbo
setting. (A soft-output Viterbi variant, SOVA, is a cheaper approximation that also yields
LLRs.)

## Relevance to SDR

BCJR is the component decoder inside [turbo codes](/reference/turbo-code/), which carry the
data channels of 3G/4G cellular and many deep-space links; iterating two BCJR decoders that
exchange extrinsic LLRs is what gets those systems close to the Shannon limit. It is a
[forward error correction](/reference/forward-error-correction/) algorithm for the
convolutional-code family and a cousin of the belief-propagation decoding used for
[LDPC codes](/reference/ldpc-code/). The land-mobile and aviation formats GopherTrunk targets
use block and plain convolutional codes rather than turbo codes, so GT does not run a BCJR
decoder; it belongs to the broader cellular/space FEC world and is included here for context
alongside Viterbi.

## Sources

[^wiki]: [BCJR algorithm](https://en.wikipedia.org/wiki/BCJR_algorithm) — Wikipedia, for the MAP forward–backward soft-output decoder.
[^bcjr]: [Optimal decoding of linear codes for minimizing symbol error rate](https://ieeexplore.ieee.org/document/1055186) — Bahl, Cocke, Jelinek, Raviv, IEEE Trans. Information Theory (1974), the original MAP algorithm.
