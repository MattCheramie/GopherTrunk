---
slug: substitution-permutation-network
title: Substitution-permutation network (SPN)
entry_type: algorithm
category: cryptography
description: A substitution-permutation network is a block-cipher structure that alternates S-box substitution layers with bit-permutation/mixing layers over several rounds to achieve confusion and diffusion.
keywords: substitution-permutation network, SPN, S-box, permutation, confusion, diffusion, avalanche, AES, SubBytes, ShiftRows, MixColumns, block cipher, rounds
aka: [SPN]
autolink: true
infobox:
  - { label: Type, value: "Block-cipher structure" }
  - { label: Layers, value: "Substitution + permutation" }
  - { label: Used by, value: "AES" }
see_also: [block-cipher, s-box, advanced-encryption-standard, feistel-network, data-encryption-standard]
cite_urls:
  - https://en.wikipedia.org/wiki/Substitution%E2%80%93permutation_network
---

**A substitution-permutation network (SPN)** builds a [block cipher](/reference/block-cipher/)
by alternating two kinds of layer over many rounds: a substitution layer of
[S-boxes](/reference/s-box/) and a permutation layer that rearranges or linearly mixes
bits.[^wiki] The two layers deliver Shannon's *confusion* and *diffusion*, and the structure
is the basis of [AES](/reference/advanced-encryption-standard/), today's dominant cipher.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 100" role="img" aria-label="One SPN round: add the round key, pass through a substitution layer of S-boxes, then a permutation layer." xmlns="http://www.w3.org/2000/svg">
  <rect x="20" y="38" width="58" height="30" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="49" y="57" text-anchor="middle" font-size="8" fill="currentColor">⊕ key</text>
  <line x1="78" y1="53" x2="118" y2="53" stroke="currentColor" marker-end="url(#spar)"/>
  <rect x="120" y="38" width="78" height="30" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.2"/><text x="159" y="57" text-anchor="middle" font-size="8" fill="currentColor">S-boxes</text>
  <line x1="198" y1="53" x2="238" y2="53" stroke="currentColor" marker-end="url(#spar)"/>
  <rect x="240" y="38" width="86" height="30" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="283" y="57" text-anchor="middle" font-size="8" fill="currentColor">permutation</text>
  <line x1="326" y1="53" x2="366" y2="53" stroke="currentColor" marker-end="url(#spar)"/><text x="372" y="57" font-size="8" fill="currentColor">next round</text>
  <defs><marker id="spar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>One SPN round: mix in the round key, substitute through S-boxes, then permute — repeated over many rounds.</figcaption>
</figure>

## How it works

Each round of an SPN applies three steps to the whole block:

- **Key mixing** — XOR in a round key derived from the master key by the key schedule.
- **Substitution** — split the block into small chunks and pass each through an
  [S-box](/reference/s-box/), a nonlinear lookup that supplies *confusion* by making the
  relationship between key and ciphertext complicated and non-affine.
- **Permutation / mixing** — shuffle or linearly combine the bits across the block so that
  the local effect of each S-box spreads out, supplying *diffusion*.

Stacking many such rounds means a one-bit change at the input cascades, after only a few
rounds, into a change in roughly half the output bits — the avalanche effect. Unlike a
[Feistel network](/reference/feistel-network/), an SPN transforms the *entire* block each
round, so the substitution and permutation steps must themselves be invertible: decryption
runs the inverse S-boxes and inverse permutation with the round keys reversed.

## Variants — AES as the worked example

AES is the SPN everyone actually uses, and its round names map one-to-one onto the abstract
layers:

- **AddRoundKey** is the key-mixing step (XOR the 128-bit round key).
- **SubBytes** is the substitution layer — every byte of the 16-byte state passes through
  the same 8-to-8-bit AES [S-box](/reference/s-box/).
- **ShiftRows** and **MixColumns** together are the permutation/mixing layer: ShiftRows
  rotates the rows of the state (a byte permutation) and MixColumns mixes each column with a
  fixed matrix over GF(2⁸), so within two rounds every output byte depends on every input
  byte. The final round drops MixColumns to keep encryption and decryption symmetric.

AES runs 10, 12, or 14 such rounds for 128-, 192-, or 256-bit keys. Its MixColumns diffusion
is engineered for a provable resistance bound against
[differential](/reference/differential-cryptanalysis/) and linear attacks — the "wide trail"
strategy — which is a defining advantage of the SPN approach over ad-hoc round functions.

## Relevance to SDR

The most important SPN for radio work is AES, used for P25 AES-256 voice encryption and as a
DMR encryption option. The construction is therefore behind much of the strongest encrypted
traffic GopherTrunk sees, and — as with any well-designed cipher — recovering that audio
without the key is infeasible. The same S-box-plus-permutation vocabulary also frames how
weaker, non-encryption [obfuscation](/reference/obfuscation/) schemes are analyzed: the
clean-room talker-alias work in issue #773 recovered a fixed substitution table from public
data, the kind of nonlinear lookup an SPN would call an S-box.

## Sources

[^wiki]: [Substitution–permutation network](https://en.wikipedia.org/wiki/Substitution%E2%80%93permutation_network) — Wikipedia, for the alternating substitution/permutation rounds, confusion/diffusion, and the AES round mapping.
