---
slug: feistel-network
title: Feistel network
entry_type: algorithm
category: cryptography
description: A Feistel network is a block-cipher construction that splits each block into two halves and mixes them over several rounds using a round function, so the same structure encrypts and decrypts.
keywords: Feistel network, Feistel cipher, DES, round function, block cipher, half block, invertible, rounds
aka: [Feistel cipher]
autolink: true
infobox:
  - { label: Type, value: "Block-cipher construction" }
  - { label: Structure, value: "Split-and-swap halves" }
  - { label: Used by, value: "DES" }
see_also: [block-cipher, substitution-permutation-network, s-box, cipher]
cite_urls:
  - https://en.wikipedia.org/wiki/Feistel_cipher
---

**A Feistel network** is a way of building a [block cipher](/reference/block-cipher/) by
splitting each block into two halves and, round after round, mixing one half into the other
through a *round function*.[^wiki] Its defining property is that the same structure both
encrypts and decrypts — the round function never has to be inverted.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 110" role="img" aria-label="One Feistel round: the right half passes through a round function F and is XORed into the left half, then the halves swap." xmlns="http://www.w3.org/2000/svg">
  <rect x="40" y="20" width="60" height="22" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="70" y="35" text-anchor="middle" font-size="9" fill="currentColor">L</text>
  <rect x="150" y="20" width="60" height="22" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="180" y="35" text-anchor="middle" font-size="9" fill="currentColor">R</text>
  <line x1="180" y1="42" x2="180" y2="60" stroke="currentColor"/>
  <rect x="155" y="60" width="50" height="22" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.2"/><text x="180" y="75" text-anchor="middle" font-size="9" fill="currentColor">F(key)</text>
  <circle cx="70" cy="71" r="11" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="70" y="75" text-anchor="middle" font-size="10" fill="currentColor">⊕</text>
  <line x1="70" y1="42" x2="70" y2="60" stroke="currentColor"/>
  <line x1="155" y1="71" x2="81" y2="71" stroke="currentColor" marker-end="url(#fnar)"/>
  <line x1="70" y1="82" x2="180" y2="100" stroke="currentColor" stroke-dasharray="3 3"/><line x1="180" y1="42" x2="70" y2="100" stroke="currentColor" stroke-dasharray="3 3"/>
  <text x="300" y="60" font-size="9" fill="currentColor">XOR R-via-F into L, then swap</text>
  <defs><marker id="fnar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>One Feistel round: F transforms the right half, XORs it into the left, and the halves swap.</figcaption>
</figure>

## How it works

In each round the right half `R` is fed through a round function `F` (mixing in a round key),
the result is XORed into the left half `L`, and then the two halves swap. Because the
untouched half is carried forward unchanged, decryption simply runs the identical rounds in
reverse order: knowing `R`, you can recompute `F(R)` and undo the XOR. This means the round
function `F` need not be invertible — it can be any nonlinear mapping, which gives designers
great freedom in building `F` from [S-boxes](/reference/s-box/) and bit permutations.

A single round provides little security, so a real Feistel cipher repeats many rounds (DES
uses 16), each with a different round key derived from the master key, until every output bit
depends on every input and key bit.

## Relevance to SDR

The classic Feistel cipher is DES, one of the algorithms used to encrypt P25 voice, so the
construction sits behind some of the encrypted traffic GopherTrunk encounters. As with any
strong cipher, the relevance is honest but bounded: GopherTrunk can identify an encrypted
call but cannot recover its audio without the key.

The structure is also useful as an analysis template. The clean-room study of the Motorola
P25 talker-alias [obfuscation](/reference/obfuscation/) in issue #773 tested whether the
scheme's byte updates followed a Feistel-shaped split-and-mix pattern; that hypothesis was
evaluated and ruled out from public on-air data alone, with no third-party source involved.

## Sources

[^wiki]: [Feistel cipher](https://en.wikipedia.org/wiki/Feistel_cipher) — Wikipedia, for the split-half round structure and the fact that decryption reuses the same rounds.
