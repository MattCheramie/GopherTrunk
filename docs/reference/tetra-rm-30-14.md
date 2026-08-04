---
slug: tetra-rm-30-14
title: TETRA RM(30,14)
entry_type: algorithm
category: error-correction
description: RM(30,14) is the shortened Reed–Muller block code TETRA uses on its access-assignment channel — a systematic (30,14) code that expands 14 information bits into 30, decoded by nearest-codeword search in hard or soft form, with no interleaving or convolutional coding.
keywords: TETRA RM 30 14, Reed-Muller code, AACH, access assignment channel, systematic block code, soft-decision decode, parity matrix, EN 300 392-2 8.2.3.2
aka: ["RM(30,14)", "TETRA (30,14) code", "AACH block code"]
autolink: true
infobox:
  - { label: Code, value: "shortened Reed–Muller (30,14)" }
  - { label: Generator, value: "systematic G = [I14 | P]" }
  - { label: Decode, value: nearest-codeword, hard + soft }
  - { label: Spec, value: EN 300 392-2 §8.2.3.2 }
see_also: [reed-muller-code, hadamard-code, soft-decision, tetra-aach, forward-error-correction, log-likelihood-ratio, tetra, tetra-logical-channels]
cite_urls:
  - https://en.wikipedia.org/wiki/Reed%E2%80%93Muller_code
  - https://en.wikipedia.org/wiki/Block_code
---

**RM(30,14)** is the shortened [Reed–Muller](/reference/reed-muller-code/) block code TETRA uses to protect
its access-assignment channel (AACH).[^rm] It expands **14 type-1 information bits into 30 type-2 channel
bits** and — unusually for TETRA — needs no further convolutional coding or [interleaving](/reference/interleaving/):
the AACH's type-4 block simply *is* its type-2 block. That compactness is deliberate. The AACH is broadcast in
the two half-blocks flanking every downlink burst's training sequence, telling a receiver which slot carries
what, so it must decode from a very short, self-contained codeword on every single burst.[^block]

<figure class="figure" markdown="0">
<svg viewBox="0 0 440 118" role="img" aria-label="Fourteen information bits pass unchanged into the first fourteen output positions of a systematic codeword, while a fourteen-by-sixteen parity matrix generates sixteen parity bits appended to make a thirty-bit codeword." xmlns="http://www.w3.org/2000/svg">
  <rect x="20" y="34" width="150" height="28" fill="currentColor" fill-opacity="0.22" stroke="currentColor" stroke-width="1.1"/>
  <text x="95" y="52" text-anchor="middle" font-size="9" fill="currentColor">14 info bits (I₁₄)</text>
  <rect x="170" y="34" width="180" height="28" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.1"/>
  <text x="260" y="52" text-anchor="middle" font-size="9" fill="currentColor">16 parity bits (P)</text>
  <text x="20" y="26" font-size="7.5" fill="currentColor">systematic: first 14 output bits = the input</text>
  <path d="M95 70 L95 82" stroke="currentColor" stroke-width="1" fill="none"/>
  <path d="M260 70 L260 82" stroke="currentColor" stroke-width="1" fill="none"/>
  <text x="185" y="98" text-anchor="middle" font-size="8" fill="currentColor">30-bit codeword = b₁ · G, G = [ I₁₄ | P ]</text>
</svg>
<figcaption>The code is systematic: the 14 information bits appear unchanged in the first 14 output positions, and 16 parity bits computed from the 14×16 matrix P are appended to form the 30-bit codeword.</figcaption>
</figure>

## The generator

The generator matrix is `G = [I14 | P]`: the 14×14 identity makes the code **systematic** — the first 14
codeword bits equal the information bits — and `P` is the fixed 14×16 parity matrix from EN 300 392-2 eq.
(8.13). Encoding is `b2 = b1 · G` over GF(2): copy the 14 input bits, then for each of the 16 parity columns
XOR together the input bits whose row carries a 1 in that column. GopherTrunk stores `P` as a literal
`[14][16]byte` table so there is nothing to derive at runtime, and `EncodeRM3014Tetra` implements exactly that
copy-then-parity rule.

## Decoding, hard and soft

The codeword is short enough — only `2^14 = 16 384` valid codewords — that decoding by exhaustive nearest-codeword
search is cheap. The **hard** decoder, `DecodeRM3014Tetra`, encodes every candidate and returns the information
vector at minimum Hamming distance from the 30 received bits; distance 0 means a clean codeword. The code's
minimum distance is at least 4, giving guaranteed single-bit correction across the 30-bit block and detection of
up to three errors. The **soft** decoder, `DecodeRM3014TetraSoft`, runs the same search but maximises a soft
correlation over the received [log-likelihood ratios](/reference/log-likelihood-ratio/) instead of minimising
Hamming distance — adding a codeword's LLR where its bit is 0 and subtracting it where the bit is 1, then picking
the maximum. Soft decoding buys roughly 2 dB of coding gain, which recovers the AACH usage marker on marginal
concurrent-call bursts a hard decoder mis-corrects. The soft decoder also returns a confidence margin (the
normalised gap to the second-best codeword), which callers use to gate a low-confidence pick so a rescued marker
never misroutes another call's speech.

Reed–Muller codes are closely related to [Hadamard codes](/reference/hadamard-code/) — first-order Reed–Muller is
a Hadamard code plus its complement — and both admit the fast correlation decoding that makes an exhaustive search
over a short block practical.

## Relevance to SDR

`internal/radio/framing/rm_30_14_tetra.go` holds the parity matrix, the systematic encoder, and both decoders.
The [AACH](/reference/tetra-aach/) path calls the hard decoder first and falls back to the soft decoder — gated
on a confident, valid decode within a bounded Hamming distance — precisely because the AACH is the per-slot call
identifier the voice chain routes by. On a marginal burst, whether that block decodes decides whether a
concurrent call is attributed to the right slot or dropped, so the small (30,14) code sits directly on the
voice-follow critical path.

## Sources

[^rm]: [Reed–Muller code](https://en.wikipedia.org/wiki/Reed%E2%80%93Muller_code) — Wikipedia, on the Reed–Muller family and its distance and decoding properties.
[^block]: [Block code](https://en.wikipedia.org/wiki/Block_code) — Wikipedia, on systematic generator matrices and nearest-codeword decoding.
