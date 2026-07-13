---
slug: forward-error-correction
title: Forward error correction (FEC)
entry_type: term
category: error-correction
description: Forward error correction adds structured redundancy so the receiver can correct errors without retransmission; it is essential to one-way radio links and comes in block and convolutional forms.
keywords: forward error correction, FEC, redundancy, error correcting code, code rate, coding gain, hard decision, soft decision, block code, convolutional code
aka: [forward error correction, FEC, channel coding]
autolink: true
infobox:
  - { label: Type, value: Error-control strategy }
  - { label: Adds, value: Redundancy for correction (no retransmit) }
  - { label: Families, value: Block (RS, BCH, Golay) & convolutional / turbo / LDPC }
see_also: [convolutional-code, reed-solomon-code, viterbi-algorithm, ldpc-code, turbo-code, interleaving, hamming-code]
related_lessons:
  - { title: "The demodulation pipeline", url: /learn/rf-sdr/demodulation-pipeline/ }
related_reading:
  - { title: "SDR Internals, Part 9: Framing & forward error correction", url: /blog/deep-dives/sdr-internals-09-framing-fec/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Error_correction_code
  - https://en.wikipedia.org/wiki/Forward_error_correction
---

**Forward error correction** (**FEC**) adds structured redundancy to transmitted data so the
receiver can **correct** errors on its own — no acknowledgement, no retransmission.[^wiki] That
one-way property is what makes it essential to broadcast and radio: a scanner listening to a
[trunked-radio](/reference/trunked-radio/) control channel cannot ask the tower to resend a garbled
grant, so the redundancy has to be baked into the transmission. FEC is the counterpart to the
detect-and-retransmit approach (ARQ); a
[cyclic redundancy check](/reference/cyclic-redundancy-check/) only *detects* corruption, whereas FEC
*repairs* it up to a bounded amount.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 120" role="img" aria-label="Data plus added parity is transmitted, a bit flips in transit, and the decoder uses the redundancy to recover the original data without any retransmission." xmlns="http://www.w3.org/2000/svg">
  <g font-family="monospace" font-size="11" fill="currentColor"><text x="30" y="45">1011</text><text x="88" y="45" fill-opacity="0.6">+parity</text></g>
  <line x1="150" y1="40" x2="212" y2="40" stroke="currentColor" marker-end="url(#fecar)"/><text x="181" y="32" text-anchor="middle" font-size="8" fill="currentColor">bit flip</text>
  <g font-family="monospace" font-size="11" fill="currentColor"><text x="226" y="45">1<tspan font-weight="bold">1</tspan>11</text></g>
  <line x1="286" y1="40" x2="348" y2="40" stroke="currentColor" marker-end="url(#fecar)"/><text x="317" y="32" text-anchor="middle" font-size="8" fill="currentColor">decode</text>
  <g font-family="monospace" font-size="11" fill="currentColor"><text x="362" y="45">1011</text></g>
  <text x="230" y="92" text-anchor="middle" font-size="9" fill="currentColor">redundancy lets the receiver repair the error with no retransmission</text>
  <defs><marker id="fecar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>Forward error correction adds parity so the receiver can identify and repair bit errors itself, which is mandatory on a one-way link where retransmission is impossible.</figcaption>
</figure>

## How it works

An FEC encoder maps *k* information bits to *n > k* coded bits; the extra `n − k` bits are a
structured function of the data, so only a small subset of all `2ⁿ` bit patterns are legal codewords.
When errors nudge a received word off a codeword, the decoder snaps it back to the nearest legal one.
Two numbers characterise any scheme:

- **Code rate `R = k/n`** — the fraction of transmitted bits that carry payload. Lower rate (more
  redundancy) tolerates more errors but costs bandwidth or throughput.
- **Coding gain** — how many dB of [signal-to-noise ratio](/reference/signal-to-noise-ratio/) the code
  saves for a target error rate versus sending uncoded. A few dB of coding gain can be the difference
  between a locked and an unlocked channel.

## Variants

FEC splits into two great families:

- **Block codes** treat data as fixed-size blocks and add parity per block. Examples include
  [Hamming](/reference/hamming-code/) (1 error), [Golay](/reference/golay-code/) (3 errors),
  [BCH](/reference/bch-code/), and [Reed–Solomon](/reference/reed-solomon-code/) (symbol-oriented,
  excellent against bursts). Modern capacity-approaching block codes include
  [LDPC](/reference/ldpc-code/), used in Wi-Fi, DVB-S2, and 5G data channels.
- **Convolutional codes** encode a continuous stream, each output depending on a sliding window of
  input bits; they are decoded with the [Viterbi algorithm](/reference/viterbi-algorithm/) and are
  often [punctured](/reference/puncturing/) to raise the rate. [Turbo codes](/reference/turbo-code/)
  combine two convolutional codes with interleaving and iterative decoding to approach the Shannon
  limit, and power 3G/4G traffic channels.

A second axis is **decision type**. A **hard-decision** decoder is handed already-sliced bits (0/1);
a **soft-decision** decoder is handed the demodulator's confidence in each bit (e.g. a log-likelihood)
and typically wins ~2 dB of coding gain by using that reliability information. Viterbi, turbo, and
LDPC decoders are all naturally soft. Codes are also frequently paired with
[interleaving](/reference/interleaving/) so that channel **bursts** are scattered into the isolated
errors the code is designed to fix.

## In practice

Where FEC sits in a frame matters: control and signalling fields usually get the strongest, lowest-rate
protection because a single lost grant can break a call, while bulk voice or data may run a higher-rate
code to conserve throughput. Systems commonly **concatenate** codes — an outer Reed–Solomon block code
over an inner convolutional code, separated by an interleaver — so the inner code cleans up random noise
and the outer code sweeps up the residual bursts the inner decoder emits.

## Relevance to SDR

Every digital protocol GopherTrunk decodes leans on FEC: [P25](/reference/project-25/) uses
trellis/convolutional coding, Golay and Reed–Solomon; [DMR](/reference/dmr/) uses the Hamming-based
[BPTC](/reference/bptc/) product code; [TETRA](/reference/tetra/) and [NXDN](/reference/nxdn/) layer
convolutional coding with interleaving; and [ADS-B](/reference/ads-b/) uses a CRC that doubles as a
short error-correcting code. GopherTrunk implements the matching decoders in each chain, which is why a
digital signal stays perfect until it abruptly fails — the famous **cliff effect**. The decoder repairs
errors invisibly until the error rate exceeds the code's correcting power, at which point frames stop
validating and audio drops out all at once, rather than degrading gracefully like analog.

## Sources

[^wiki]: [Error correction code](https://en.wikipedia.org/wiki/Error_correction_code) — Wikipedia, for the block/convolutional families, code rate, and hard vs soft decoding.
[^fec]: [Forward error correction](https://en.wikipedia.org/wiki/Forward_error_correction) — Wikipedia, for coding gain, concatenation, and the one-way-link rationale.
