---
slug: puncturing
title: Puncturing
entry_type: algorithm
category: error-correction
description: Puncturing deletes selected coded bits from a low-rate mother code to raise its effective rate, letting one encoder/decoder pair serve many rates; the decoder fills the gaps with erasures.
keywords: puncturing, punctured convolutional code, rate-compatible, mother code, puncturing pattern, code rate, erasure insertion, RCPC, turbo code, depuncturing
aka: [puncturing, punctured code, rate-compatible puncturing]
autolink: true
infobox:
  - { label: Type, value: Rate-adaptation technique }
  - { label: Effect, value: Raises effective code rate }
  - { label: Decoder step, value: Insert erasures (depuncture) }
see_also: [convolutional-code, viterbi-algorithm, forward-error-correction, turbo-code, trellis-coded-modulation, ldpc-code]
cite_urls:
  - https://en.wikipedia.org/wiki/Punctured_code
  - https://en.wikipedia.org/wiki/Convolutional_code
---

**Puncturing** is the deliberate deletion of selected coded bits from a low-rate
**mother code** so that fewer bits are transmitted, raising the effective
[code rate](/reference/forward-error-correction/) without designing a new
encoder.[^wiki] A single rate-1/2 [convolutional](/reference/convolutional-code/)
encoder, for example, can be punctured to 2/3, 3/4, 5/6 or 7/8 simply by discarding
different coded bits on a fixed schedule — and one
[Viterbi decoder](/reference/viterbi-algorithm/) handles all of them by treating each
deleted bit as an **erasure**. This is the standard way modern links offer a menu of
rates from one piece of hardware.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 140" role="img" aria-label="A rate one-half mother code emits two output streams; a puncturing pattern deletes some of the marked bits so the transmitted stream is shorter and the code rate rises, and the decoder reinserts erasures at the deleted positions." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="pncar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <text x="24" y="34" font-size="9" fill="currentColor">mother code (rate 1/2)</text>
  <g stroke="currentColor" stroke-width="1" fill="none">
    <rect x="24" y="40" width="26" height="24"/><rect x="50" y="40" width="26" height="24"/><rect x="76" y="40" width="26" height="24"/><rect x="102" y="40" width="26" height="24"/><rect x="128" y="40" width="26" height="24"/><rect x="154" y="40" width="26" height="24"/>
  </g>
  <text x="37" y="56" text-anchor="middle" font-size="10" fill="currentColor">A</text><text x="63" y="56" text-anchor="middle" font-size="10" fill="currentColor">B</text><text x="89" y="56" text-anchor="middle" font-size="10" fill="currentColor">A</text><text x="115" y="56" text-anchor="middle" font-size="10" fill="currentColor">B</text><text x="141" y="56" text-anchor="middle" font-size="10" fill="currentColor">A</text><text x="167" y="56" text-anchor="middle" font-size="10" fill="currentColor">B</text>
  <g stroke="currentColor" stroke-width="1.4"><line x1="55" y1="35" x2="71" y2="69"/><line x1="71" y1="35" x2="55" y2="69"/><line x1="133" y1="35" x2="149" y2="69"/><line x1="149" y1="35" x2="133" y2="69"/></g>
  <text x="100" y="86" text-anchor="middle" font-size="9" fill="currentColor">✕ = deleted by puncturing pattern</text>
  <line x1="200" y1="52" x2="238" y2="52" stroke="currentColor" marker-end="url(#pncar)"/>
  <g stroke="currentColor" stroke-width="1" fill="none">
    <rect x="246" y="40" width="26" height="24"/><rect x="272" y="40" width="26" height="24"/><rect x="298" y="40" width="26" height="24"/><rect x="324" y="40" width="26" height="24"/>
  </g>
  <text x="259" y="56" text-anchor="middle" font-size="10" fill="currentColor">A</text><text x="285" y="56" text-anchor="middle" font-size="10" fill="currentColor">A</text><text x="311" y="56" text-anchor="middle" font-size="10" fill="currentColor">B</text><text x="337" y="56" text-anchor="middle" font-size="10" fill="currentColor">A</text>
  <text x="298" y="86" text-anchor="middle" font-size="9" fill="currentColor">shorter stream → higher rate (e.g. 3/4)</text>
</svg>
<figcaption>Puncturing discards coded bits on a fixed pattern to shorten the transmitted stream; the decoder reinserts erasures at those positions before running the mother-code decoder.</figcaption>
</figure>

## How it works

The encoder always runs the full low-rate mother code, producing every coded bit. A
**puncturing pattern** — a small binary matrix, one row per output stream — then marks
which bits are kept (1) and which are thrown away (0) over a short repeating period.
Deleting `p` of every `q` coded bits turns a rate-`R` mother code into a higher rate.
Because only a *fixed, agreed* set of positions is dropped, the receiver knows exactly
where the holes are.

At the decoder the missing bits are **not** guessed. Instead the receiver
**depunctures**: it reinserts a placeholder at each deleted position carrying *zero
soft information* — a neutral log-likelihood, i.e. an erasure that pulls the metric in
neither direction. The [Viterbi](/reference/viterbi-algorithm/) or
[BCJR](/reference/bcjr-algorithm/) decoder then runs on the reconstructed full-length
stream as usual, simply getting no help from the erased branches. The elegance is that
the trellis, the branch metrics and the traceback are all unchanged; only the input
soft values differ. The cost is real, though: fewer transmitted parity bits mean
smaller coding gain and a higher error floor, so a punctured rate-7/8 code is far
weaker than the rate-1/2 mother it came from.

## In practice

**Rate-compatible punctured convolutional (RCPC) codes** carry the idea further: the
puncturing patterns are *nested* so that a higher-rate codeword's kept bits are always
a subset of a lower-rate one's. That lets a transmitter start at a high rate and, if
the receiver reports failure, send just the previously-punctured bits to *lower* the
rate incrementally — the foundation of **hybrid ARQ** and adaptive modulation and
coding in cellular systems. The same mechanism appears in **turbo**
([turbo-code](/reference/turbo-code/)) and [LDPC](/reference/ldpc-code/) coding, where
puncturing the parity stream sets the transmitted rate while the mother code stays
fixed.

## Relevance to SDR

Puncturing is pervasive in the wireless standards a software radio meets: Wi-Fi
(802.11a/g/n) punctures a rate-1/2, K=7 convolutional code to reach 2/3 and 3/4;
DVB, WiMAX and satellite modems publish tables of standard puncturing patterns; and
LTE/5G puncture their turbo and LDPC codes for rate matching. In land-mobile trunking
the effect shows up as the odd-looking code rates in the framing:
[P25](/reference/project-25/) and other C4FM systems mix punctured and shortened codes
to fit protection into fixed-size bursts. **GopherTrunk** must therefore *depuncture*
correctly — reinserting erasures at the standard positions before its convolutional
decoder — to read those channels; that depuncturing step, not a puncturing encoder, is
where the technique lives in a receive-only decoder like GT.

## Sources

[^wiki]: [Punctured code](https://en.wikipedia.org/wiki/Punctured_code) — Wikipedia, for the deletion of coded bits to raise rate, puncturing patterns, erasure-based depuncturing, rate-compatible punctured convolutional codes, and the Wi-Fi/DVB/turbo applications.
