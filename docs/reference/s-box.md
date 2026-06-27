---
slug: s-box
title: S-box (substitution box)
entry_type: algorithm
category: cryptography
description: An S-box is a fixed lookup table that maps an input bit pattern to an output bit pattern, providing the nonlinear substitution step that supplies confusion in a block cipher.
keywords: S-box, substitution box, lookup table, nonlinear, confusion, AES, DES, block cipher, substitution
aka: [substitution box, S-box]
autolink: true
infobox:
  - { label: Type, value: "Substitution lookup table" }
  - { label: Role, value: "Nonlinear element" }
  - { label: Provides, value: "Confusion" }
see_also: [substitution-permutation-network, block-cipher, feistel-network, advanced-encryption-standard, cipher]
cite_urls:
  - https://en.wikipedia.org/wiki/S-box
---

**An S-box (substitution box)** is a fixed lookup table that replaces an input bit pattern
with an output bit pattern.[^wiki] It is the nonlinear heart of most modern ciphers: where
the surrounding XOR and permutation steps are linear, the S-box deliberately is not, supplying
the *confusion* that makes the key-to-ciphertext relationship hard to unravel.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 100" role="img" aria-label="An input value indexes into a lookup table and is replaced by the stored output value." xmlns="http://www.w3.org/2000/svg">
  <rect x="20" y="40" width="60" height="28" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="50" y="58" text-anchor="middle" font-size="9" fill="currentColor">input</text>
  <line x1="80" y1="54" x2="150" y2="54" stroke="currentColor" marker-end="url(#sbar)"/>
  <rect x="152" y="30" width="120" height="48" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.2"/><text x="212" y="50" text-anchor="middle" font-size="8" fill="currentColor">lookup table</text><text x="212" y="64" text-anchor="middle" font-size="8" fill="currentColor">in → out</text>
  <line x1="272" y1="54" x2="342" y2="54" stroke="currentColor" marker-end="url(#sbar)"/>
  <rect x="344" y="40" width="60" height="28" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="374" y="58" text-anchor="middle" font-size="9" fill="currentColor">output</text>
  <defs><marker id="sbar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>An S-box is a fixed table: each input value indexes to a predetermined output, a nonlinear substitution.</figcaption>
</figure>

## How it works

An S-box takes `m` input bits and produces `n` output bits by table lookup — for example,
AES uses a single 8-bit-to-8-bit S-box, a 256-entry table applied to every byte. The table
is a fixed part of the cipher, the same for every key; secrecy lives in the key, not in the
table (Kerckhoffs's principle).

A good S-box is carefully designed, not arbitrary. It should be strongly nonlinear so that no
simple linear or affine relation approximates it, and well-balanced so that flipping one
input bit changes many output bits unpredictably. These properties are what blunt
[differential and linear cryptanalysis](/reference/cryptanalysis/). S-boxes appear in both
[Feistel networks](/reference/feistel-network/) (inside the round function) and
[substitution-permutation networks](/reference/substitution-permutation-network/) (as the
substitution layer).

## Relevance to SDR

S-boxes sit inside the ciphers that protect digital radio traffic — AES (P25 AES-256, DMR)
and DES (P25) both rely on S-box substitution — so the construction is part of what makes
encrypted voice GopherTrunk monitors infeasible to recover without the key.

The term is also relevant to weaker, non-encryption schemes. The clean-room analysis of the
Motorola P25 talker-alias [obfuscation](/reference/obfuscation/) in issue #773 recovered a
256-entry substitution table purely from public on-air data, with no third-party source: a
fixed lookup like this is exactly an S-box in form, even though, used alone for reversible
hiding rather than keyed encryption, it provides [obfuscation](/reference/obfuscation/) and
not secrecy.

## Sources

[^wiki]: [S-box](https://en.wikipedia.org/wiki/S-box) — Wikipedia, for the substitution-table definition and its role as the nonlinear, confusion-providing element of a cipher.
