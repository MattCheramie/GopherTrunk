---
slug: advanced-encryption-standard
title: Advanced Encryption Standard (AES)
entry_type: algorithm
category: cryptography
description: AES is a symmetric block cipher standardized as FIPS-197, encrypting 128-bit blocks with 128-, 192-, or 256-bit keys via a substitution-permutation network; P25 and DMR use AES for secure voice.
keywords: AES, Advanced Encryption Standard, Rijndael, FIPS-197, block cipher, symmetric, S-box, substitution-permutation network, AES-256, P25 encryption, DMR encryption
aka: [AES, Rijndael]
autolink: true
infobox:
  - { label: Type, value: Symmetric block cipher }
  - { label: Block, value: 128-bit }
  - { label: Keys, value: 128 / 192 / 256-bit }
see_also: [substitution-permutation-network, s-box, block-cipher, symmetric-key-cryptography, hash-function]
cite_urls:
  - https://en.wikipedia.org/wiki/Advanced_Encryption_Standard
---

**The Advanced Encryption Standard (AES)** is a symmetric
[block cipher](/reference/block-cipher/), standardized by NIST as FIPS-197, that encrypts
128-bit blocks under a 128-, 192-, or 256-bit key.[^wiki] It was selected from the Rijndael
design of Daemen and Rijmen and is the dominant cipher for secure digital voice.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 100" role="img" aria-label="A 128-bit block and key enter several AES rounds of S-box substitution and permutation, producing ciphertext." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <rect x="20" y="38" width="56" height="28" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="48" y="55" font-size="8">128-bit block</text>
    <line x1="76" y1="52" x2="108" y2="52" stroke="currentColor" marker-end="url(#aesar)"/>
    <rect x="110" y="34" width="56" height="36" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.2"/><text x="138" y="50" font-size="8">S-box</text><text x="138" y="62" font-size="8">sub</text>
    <line x1="166" y1="52" x2="186" y2="52" stroke="currentColor" marker-end="url(#aesar)"/>
    <rect x="188" y="34" width="56" height="36" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="216" y="50" font-size="8">permute</text><text x="216" y="62" font-size="8">+ mix</text>
    <text x="276" y="55" font-size="14">×N</text>
    <line x1="300" y1="52" x2="332" y2="52" stroke="currentColor" marker-end="url(#aesar)"/>
    <rect x="334" y="38" width="56" height="28" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="362" y="55" font-size="8">ciphertext</text>
    <text x="138" y="22" font-size="8">round key ↓</text><line x1="138" y1="24" x2="138" y2="34" stroke="currentColor"/>
  </g>
  <defs><marker id="aesar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>AES iterates substitution and permutation rounds, each mixing in a round key, to turn a block into ciphertext.</figcaption>
</figure>

## How it works

AES is a [substitution-permutation network](/reference/substitution-permutation-network/).
Each round applies four steps to the 128-bit state arranged as a 4×4 byte matrix:

- **SubBytes** — a fixed [S-box](/reference/s-box/) substitutes each byte (nonlinear
  *confusion*);
- **ShiftRows** and **MixColumns** — permute and mix bytes across the block (*diffusion*);
- **AddRoundKey** — XOR in a round key derived from the cipher key.

The number of rounds depends on key length: 10 for AES-128, 12 for AES-192, and 14 for
AES-256. Decryption runs the inverse steps with the round keys in reverse order. As a
[symmetric](/reference/symmetric-key-cryptography/) cipher, the same key encrypts and
decrypts, so a listener without the key cannot recover the plaintext. A bare block cipher
only encrypts one block, so it is used with a *mode of operation* (CTR, OFB, CBC) to handle
streams of data.

## Relevance to SDR

AES is the encryption a scanner most often runs into and cannot defeat. P25 systems carry
**AES-256** (often alongside legacy DES) for secure voice, and DMR equipment offers AES
options; in both, AES typically runs in a keystream mode so it behaves like a
[stream cipher](/reference/stream-cipher/) over the audio. GopherTrunk can detect, follow,
and log these encrypted calls — it sees the talkgroup, source, and that the traffic is
encrypted — but it cannot decode the voice without the key, which is the entire point of the
standard. This is the honest boundary of the project: clear and scrambled traffic decode,
keyed AES traffic does not.

## Sources

[^wiki]: [Advanced Encryption Standard](https://en.wikipedia.org/wiki/Advanced_Encryption_Standard) — Wikipedia, for the FIPS-197 standard, Rijndael origin, round structure, and key sizes.
