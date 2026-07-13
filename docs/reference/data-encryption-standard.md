---
slug: data-encryption-standard
title: Data Encryption Standard (DES)
entry_type: algorithm
category: cryptography
description: DES is an obsolete 56-bit-key Feistel block cipher, now brute-forceable, still seen as DES-OFB in legacy P25 secure voice; its short key is why GopherTrunk still cannot recover keyed audio.
keywords: DES, Data Encryption Standard, Feistel cipher, 56-bit key, block cipher, DES-OFB, DES-XL, 3DES, Triple DES, brute force, P25 encryption, legacy secure voice, FIPS 46
aka: [DES, Data Encryption Standard]
autolink: true
infobox:
  - { label: Type, value: Feistel block cipher }
  - { label: Key / block, value: 56-bit key, 64-bit block }
  - { label: Status, value: Obsolete (brute-forceable) }
see_also: [feistel-network, block-cipher, brute-force-attack, advanced-encryption-standard, stream-cipher]
cite_urls:
  - https://en.wikipedia.org/wiki/Data_Encryption_Standard
  - https://en.wikipedia.org/wiki/EFF_DES_cracker
---

**The Data Encryption Standard (DES)** is a symmetric
[block cipher](/reference/block-cipher/) built as a 16-round
[Feistel network](/reference/feistel-network/), encrypting 64-bit blocks under a **56-bit
key**.[^wiki] Standardised as FIPS 46 in 1977 and dominant for two decades, its short key is
now trivially [brute-forceable](/reference/brute-force-attack/), so DES is obsolete — yet it
survives as **DES-OFB** in legacy P25 secure voice.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="A DES round splits the 64-bit block into two halves; the right half and a round subkey pass through the F function, whose output XORs into the left half before the halves swap, repeated for sixteen rounds." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <rect x="40" y="24" width="70" height="24" rx="3" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="75" y="40" font-size="8">L (32b)</text>
    <rect x="150" y="24" width="70" height="24" rx="3" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="185" y="40" font-size="8">R (32b)</text>
    <rect x="150" y="70" width="70" height="30" rx="4" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.3"/><text x="185" y="89" font-size="9">F</text>
    <line x1="185" y1="48" x2="185" y2="70" stroke="currentColor" marker-end="url(#desar)"/>
    <text x="270" y="88" font-size="8">round key Kᵢ</text><line x1="255" y1="85" x2="221" y2="85" stroke="currentColor" marker-end="url(#desar)"/>
    <circle cx="75" cy="85" r="12" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="75" y="89" font-size="11">⊕</text>
    <line x1="75" y1="48" x2="75" y2="73" stroke="currentColor" marker-end="url(#desar)"/>
    <path d="M150 85 L87 85" fill="none" stroke="currentColor" marker-end="url(#desar)"/>
    <path d="M75 97 L75 120 L220 120 L220 132" fill="none" stroke="currentColor" marker-end="url(#desar)"/>
    <path d="M185 100 L185 120 L75 120" fill="none" stroke="currentColor" stroke-dasharray="3 2"/>
    <text x="150" y="145" font-size="8">halves swap → next round (×16)</text>
  </g>
  <defs><marker id="desar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>Each DES round runs the right half and a subkey through the F function, XORs the result into the left half, and swaps — sixteen Feistel rounds in all.</figcaption>
</figure>

## How it works

DES is the textbook [Feistel cipher](/reference/feistel-network/). A 64-bit block is split
into left and right 32-bit halves and passed through 16 identical rounds:

- **Round function F.** The right half is expanded from 32 to 48 bits, XORed with a 48-bit
  round subkey, then fed through eight **S-boxes** that map 6 bits to 4 (the nonlinear heart
  of the cipher), and finally a fixed permutation.
- **Feistel structure.** F's output XORs into the left half, and the halves swap. Because the
  round is invertible regardless of what F does, decryption is the same machinery with the
  subkeys applied in reverse order.
- **Key schedule.** The 64-bit key is really **56 effective bits** (8 are parity); a schedule
  of rotations and permutations derives the sixteen 48-bit subkeys.
- **Initial/final permutations** bookend the rounds and add no cryptographic strength.

The design (with its later-explained S-box choices) resists differential and linear
cryptanalysis remarkably well. Its fatal flaw is not the algorithm but the **key length**: 56
bits is only ~7.2×10¹⁶ keys.

## Variants

- **Triple DES (3DES).** Encrypt–decrypt–encrypt with two or three keys restores an effective
  ~112-bit strength; used to extend DES's life in finance and legacy systems.
- **DES-OFB / DES-CFB.** Running DES in **output-feedback** mode turns the block cipher into a
  [stream cipher](/reference/stream-cipher/) — a keystream XORed with the digitised voice.
  This is the form P25 uses (Algorithm ID 0x81), so the block cipher never touches the audio
  bits directly.
- **DES-XL / ADP.** Motorola-proprietary DES-derived modes seen in older public-safety radios.

## In practice

DES fell to brute force publicly in 1998–99, when the EFF's "Deep Crack" machine and a
distributed effort recovered a key in **days**, then hours. Modern FPGAs and GPUs do it far
faster. That is why NIST withdrew DES in favour of [AES](/reference/advanced-encryption-standard/).

## Relevance to SDR

On the air a scanner still meets DES as **DES-OFB** on older P25 systems that have not migrated
to AES-256. GopherTrunk can detect and follow such calls — reading the talkgroup, source, and
the encryption algorithm/key ID in the clear — but it does **not** carry out key search and
does not decrypt the voice: recovering the audio would require the actual 56-bit key or an
out-of-band brute-force attack that is outside the project's scope and legality. DES's weakness
is real in theory, but GopherTrunk remains a receiver of clear and scrambled traffic, not a
codebreaker of keyed voice — the same honest boundary that applies to
[AES](/reference/advanced-encryption-standard/).

## Sources

[^wiki]: [Data Encryption Standard](https://en.wikipedia.org/wiki/Data_Encryption_Standard) — Wikipedia, for the Feistel structure, 56-bit key, S-boxes, and DES's brute-force obsolescence.
