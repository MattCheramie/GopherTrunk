---
slug: advanced-encryption-standard
title: Advanced Encryption Standard (AES)
entry_type: algorithm
category: cryptography
description: AES is a symmetric block cipher standardized as FIPS-197, encrypting 128-bit blocks with 128-, 192-, or 256-bit keys via a substitution-permutation network; P25 and DMR use AES for secure voice.
keywords: AES, Advanced Encryption Standard, Rijndael, FIPS-197, block cipher, symmetric, S-box, substitution-permutation network, key schedule, AES-256, CTR mode, OFB mode, side-channel, P25 encryption, DMR encryption
aka: [AES, Rijndael]
autolink: true
infobox:
  - { label: Type, value: Symmetric block cipher }
  - { label: Block, value: 128-bit }
  - { label: Keys, value: 128 / 192 / 256-bit }
see_also: [substitution-permutation-network, s-box, block-cipher, data-encryption-standard, stream-cipher, symmetric-key-cryptography, project-25, dmr]
cite_urls:
  - https://en.wikipedia.org/wiki/Advanced_Encryption_Standard
  - https://nvlpubs.nist.gov/nistpubs/FIPS/NIST.FIPS.197-upd1.pdf
---

**The Advanced Encryption Standard (AES)** is a symmetric
[block cipher](/reference/block-cipher/), standardized by NIST as FIPS-197, that encrypts
128-bit blocks under a 128-, 192-, or 256-bit key.[^wiki] It was selected in 2000 from the
Rijndael design of Joan Daemen and Vincent Rijmen, replacing the aging
[DES](/reference/data-encryption-standard/), and is now the dominant cipher for secure
digital voice and data worldwide.

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
The 128-bit state is arranged as a 4×4 matrix of bytes, and each round applies four steps:

- **SubBytes** — a fixed [S-box](/reference/s-box/) substitutes each byte (nonlinear
  *confusion*), built from inversion in the finite field GF(2⁸);
- **ShiftRows** — cyclically shifts the matrix rows to spread bytes across columns;
- **MixColumns** — mixes the four bytes of each column by a fixed matrix multiply over
  GF(2⁸) (together with ShiftRows this provides *diffusion*);
- **AddRoundKey** — XORs in a round key derived from the cipher key.

The final round omits MixColumns.[^fips] Decryption runs the inverse steps with the round
keys in reverse order. As a [symmetric](/reference/symmetric-key-cryptography/) cipher, the same key
encrypts and decrypts, so a listener without the key cannot recover the plaintext.

### Rounds and the key schedule

The round count scales with key length: **10 rounds for AES-128, 12 for AES-192, and 14 for
AES-256**. Each round needs its own 128-bit round key, and the *key schedule* expands the
original key into that sequence. It processes the key in 4-byte words, and at word boundaries
applies a rotation, the same S-box, and a round constant (Rcon) before XOR-chaining words
together. This nonlinear, round-dependent expansion is what stops an attacker who recovers
one round key from trivially rolling back to the master key.

## Variants — modes of operation

A bare block cipher only transforms one 128-bit block, so real traffic uses AES inside a
*mode of operation* that chains blocks and injects a nonce or initialization vector:

- **CTR (counter)** and **OFB (output feedback)** turn AES into a keystream generator: AES
  encrypts a running counter or feedback register, and the output is XORed with the data.
  This makes AES behave exactly like a [stream cipher](/reference/stream-cipher/), which is
  ideal for a continuous voice bitstream because it needs no block-boundary padding and
  errors do not propagate.
- **CBC** chains each block into the next for bulk data at rest.
- Authenticated modes such as **GCM** add integrity on top of confidentiality.

P25 and DMR secure voice use the keystream-style modes (CTR/OFB) so a fixed-rate vocoder
stream can be encrypted symbol-for-symbol.

## In practice — side channels

AES is not broken mathematically; the best key-recovery attacks are only marginally faster
than brute force and remain utterly infeasible. Real attacks target *implementations*
instead. Naive lookup-table S-boxes leak key-dependent cache-timing information, so
production code uses constant-time implementations or the hardware **AES-NI** instructions.
Power-analysis and fault attacks threaten smart cards and radios physically, which is why
secure handhelds store keys in tamper-resistant modules rather than in software.

## Relevance to SDR

AES is the encryption a scanner most often runs into and cannot defeat. [P25](/reference/project-25/)
systems carry **AES-256** (often alongside legacy DES) for secure voice, and
[DMR](/reference/dmr/) equipment offers AES options; in both, AES runs in a keystream mode so
it behaves like a stream cipher over the audio, re-seeded per transmission by a message
indicator. GopherTrunk can detect, follow, and log these encrypted calls — it sees the
talkgroup, source, and that the traffic is encrypted — but it cannot decode the voice without
the key, which is the entire point of the standard. This is the honest boundary of the
project: clear and scrambled traffic decode, keyed AES traffic does not.

## Sources

[^wiki]: [Advanced Encryption Standard](https://en.wikipedia.org/wiki/Advanced_Encryption_Standard) — Wikipedia, for the Rijndael origin, round structure, key sizes, key schedule, and modes.
[^fips]: [FIPS-197 (updated)](https://nvlpubs.nist.gov/nistpubs/FIPS/NIST.FIPS.197-upd1.pdf) — NIST, the primary standard defining SubBytes, ShiftRows, MixColumns, AddRoundKey, and the key expansion.
