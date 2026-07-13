---
slug: soft-decision
title: Soft-decision demodulation
entry_type: term
category: sdr-dsp
description: "Soft-decision demodulation passes a graded reliability value for each received symbol to the decoder instead of a hard 0/1, yielding roughly 2 dB of coding gain."
keywords: soft decision, soft-decision decoding, soft bits, hard decision, reliability metric, coding gain, soft demapping, quantized soft bits, LLR input
aka: [soft-decision decoding, soft bits, soft demapping]
autolink: true
infobox:
  - { label: Type, value: Demodulator output convention }
  - { label: Contrast, value: Hard decision (0/1 only) }
  - { label: Benefit, value: ~2 dB coding gain vs hard }
see_also: [log-likelihood-ratio, viterbi-algorithm, forward-error-correction, bit-error-rate, convolutional-code, matched-filter]
cite_urls:
  - https://en.wikipedia.org/wiki/Soft-decision_decoder
---

**Soft-decision demodulation** passes the decoder a graded *reliability* value for each received
bit — how confidently it is a one or a zero — instead of committing to a hard 0/1, giving the
error-correction stage far more to work with.[^wiki] A hard decision throws away the distance of
a sample from the decision boundary; a soft decision keeps it, so a bit that landed right on the
threshold is flagged as uncertain while one deep in a region is flagged as sure. Handing that
extra information to a [forward-error-correction](/reference/forward-error-correction/) decoder
buys roughly 2 dB of [coding gain](/reference/forward-error-correction/) — a large improvement for
a change confined to the demapper.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="A received sample near a decision boundary; hard decision outputs a single bit, while soft decision outputs a signed reliability value proportional to distance from the boundary." xmlns="http://www.w3.org/2000/svg">
  <line x1="30" y1="70" x2="440" y2="70" stroke="currentColor" stroke-width="1.1"/>
  <line x1="235" y1="35" x2="235" y2="105" stroke="currentColor" stroke-width="1" stroke-dasharray="3 3"/>
  <text x="235" y="122" font-size="8" fill="currentColor" text-anchor="middle">boundary</text>
  <text x="90" y="122" font-size="8.5" fill="currentColor" text-anchor="middle">"0" region</text>
  <text x="380" y="122" font-size="8.5" fill="currentColor" text-anchor="middle">"1" region</text>
  <circle cx="330" cy="70" r="4" fill="currentColor"/>
  <text x="330" y="58" font-size="8" fill="currentColor" text-anchor="middle">sample</text>
  <line x1="235" y1="70" x2="330" y2="70" stroke="currentColor" stroke-width="2"/>
  <text x="283" y="88" font-size="7.5" fill="currentColor" text-anchor="middle">distance</text>
  <text x="35" y="40" font-size="8.5" fill="currentColor">hard → 1</text>
  <text x="345" y="40" font-size="8.5" fill="currentColor">soft → +6 (fairly sure "1")</text>
</svg>
<figcaption>Hard decision reports only which side of the boundary a sample fell on; soft decision reports a signed magnitude — the sign is the bit, the magnitude is the confidence.</figcaption>
</figure>

## How it works

After [matched filtering](/reference/matched-filter/) and slicing, each symbol produces a real
number (or, for multi-bit symbols, one number per bit) whose sign indicates the likely bit and
whose magnitude indicates how far it sat from the decision boundary. A **hard-decision**
demodulator applies `sign()` and forwards only the bit. A **soft-decision** demodulator forwards
the value itself, usually quantized to a few bits — 3-bit (8-level) soft is a common,
near-optimal compromise that captures most of the gain of full-precision soft with modest
hardware.

The decoder then uses these confidences instead of counting bit disagreements. A
[Viterbi decoder](/reference/viterbi-algorithm/), for instance, replaces Hamming-distance branch
metrics with Euclidean (or correlation) metrics computed from the soft values, so a very
confident bit outvotes several marginal ones during the survivor path search. The intuition: two
bits that are "barely 1" and "barely 0" should not carry the same weight as two bits that are
"strongly 1" and "strongly 0," and soft decoding is precisely the machinery that respects that.
The natural, calibrated form of a soft value is the
[log-likelihood ratio](/reference/log-likelihood-ratio/), which scales the raw distance by the
channel noise so the magnitudes mean the same thing across different SNRs.

## In practice

- **Quantization** — going from 1-bit (hard) to 2-bit soft captures much of the gain; 3-bit
  reaches within a few tenths of a dB of unquantized soft. Beyond that returns diminish.
- **Where it matters most** — iterative codes ([turbo](/reference/turbo-code/),
  [LDPC](/reference/ldpc-code/)) essentially *require* soft inputs; their whole advantage comes
  from passing soft reliabilities around, so a hard front end throws away most of their coding
  gain.
- **Cost** — the demapper must expose the metric and the decoder must handle real-valued inputs,
  so datapaths widen; the ~2 dB payoff is almost always worth it.
- **Erasures** — a middle ground between hard and soft marks a bit as "unknown" rather than
  guessing, which some algebraic block decoders can exploit (correcting twice as many erasures as
  errors). It captures part of the benefit of soft decision with hard-decision-like simplicity.

Where does the ~2 dB come from intuitively? Hard slicing discards information at exactly the point
where it is most valuable — for bits near the decision boundary, which are the ones most likely to
be wrong. A hard decoder is then forced to treat a bit that was "51% a one" identically to one
that was "99% a one," and it makes its worst mistakes on precisely the bits the demapper already
knew were shaky. Soft decision keeps that warning attached to each bit, so the decoder spends its
error-correcting power where it is needed and trusts the confident bits, which is why the gain
shows up as a shift of the whole bit-error-rate curve rather than a change in its shape.

Producing good soft values also depends on the stages before the decoder. The demapper needs a
reasonable estimate of the noise level to scale the metrics, and any residual carrier or timing
error skews them, so soft decision rewards a clean front end more than hard decision does — the
extra information is only useful if it is trustworthy.

## Relevance to SDR

Soft decision is standard wherever strong FEC is used: DVB, LTE/5G, Wi-Fi, and deep-space links
all decode soft. In land-mobile radio, P25's rate-1/2 convolutional/trellis code and DMR's FEC
benefit from soft metrics on the demodulated symbols. Whether GopherTrunk exploits this depends on
the decoder path — some protocol decoders in the wider ecosystem (and GT where implemented) feed
soft symbol values into their Viterbi/trellis stages rather than hard-slicing first, recovering
marginal frames that a hard decision would drop. The takeaway is that keeping the analog
confidence a little longer, rather than rounding to bits immediately, is one of the cheapest ways
to improve a marginal digital link.

## Sources

[^wiki]: [Soft-decision decoder](https://en.wikipedia.org/wiki/Soft-decision_decoder) — Wikipedia, on soft reliability inputs and the coding gain over hard decision.
