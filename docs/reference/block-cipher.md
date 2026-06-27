---
slug: block-cipher
title: Block cipher
entry_type: algorithm
category: cryptography
description: A block cipher is a symmetric cipher that encrypts fixed-size blocks of data under a key; a mode of operation chains blocks together so it can encrypt messages of any length.
keywords: block cipher, AES, DES, block size, mode of operation, ECB, CBC, OFB, CTR, symmetric encryption
aka: [block cipher]
autolink: true
infobox:
  - { label: Type, value: "Symmetric cipher" }
  - { label: Unit, value: "Fixed-size block" }
  - { label: Examples, value: "AES (128-bit), DES (64-bit)" }
see_also: [cipher, stream-cipher, advanced-encryption-standard, feistel-network, substitution-permutation-network]
cite_urls:
  - https://en.wikipedia.org/wiki/Block_cipher
---

**A block cipher** encrypts data in fixed-size blocks — for example 128 bits in
[AES](/reference/advanced-encryption-standard/) or 64 bits in DES — transforming each block
under a key with a keyed, invertible permutation.[^wiki] A *mode of operation* then chains
the blocks so the cipher can handle messages of any length.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 100" role="img" aria-label="A fixed-size plaintext block and a key enter a block cipher, producing a ciphertext block of the same size." xmlns="http://www.w3.org/2000/svg">
  <rect x="20" y="40" width="70" height="30" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="55" y="59" text-anchor="middle" font-size="8" fill="currentColor">block (n bits)</text>
  <line x1="90" y1="55" x2="160" y2="55" stroke="currentColor" marker-end="url(#bcar)"/>
  <rect x="162" y="36" width="96" height="38" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.2"/><text x="210" y="59" text-anchor="middle" font-size="8" fill="currentColor">block cipher</text>
  <text x="210" y="24" text-anchor="middle" font-size="8" fill="currentColor">key</text><line x1="210" y1="26" x2="210" y2="36" stroke="currentColor" marker-end="url(#bcar)"/>
  <line x1="258" y1="55" x2="328" y2="55" stroke="currentColor" marker-end="url(#bcar)"/><text x="333" y="59" font-size="8" fill="currentColor">cipher block (n bits)</text>
  <defs><marker id="bcar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>A block cipher maps an n-bit plaintext block to an n-bit ciphertext block under a key; a mode chains blocks.</figcaption>
</figure>

## How it works

The core block cipher is a single keyed permutation: for a fixed key it maps every possible
input block to a distinct output block, and the mapping is invertible so the holder of the
key can reverse it. Internally this permutation is built from many simple rounds, most often
as a [Feistel network](/reference/feistel-network/) (DES) or a
[substitution-permutation network](/reference/substitution-permutation-network/) (AES), each
combining confusion and diffusion so the output depends intricately on every input and key
bit.

Because real messages are longer than one block, a **mode of operation** specifies how
successive blocks combine:

- **ECB** encrypts each block independently — simple but leaks patterns, since identical
  plaintext blocks yield identical ciphertext.
- **CBC** XORs each plaintext block with the previous ciphertext block before encrypting.
- **OFB / CTR** turn the block cipher into a [keystream](/reference/keystream/) generator,
  making it behave like a [stream cipher](/reference/stream-cipher/).

## Relevance to SDR

Block ciphers underpin the strong encryption on the digital systems GopherTrunk monitors.
P25 voice may use DES or AES-256, and DMR encryption options likewise build on AES — but
these protocols run the block cipher in a feedback mode (OFB/CTR), so on air the protected
voice is delivered as a [stream cipher](/reference/stream-cipher/). The practical upshot is
the same: GopherTrunk can recognize and follow an encrypted call but cannot recover audio
without the key, since the underlying permutation is infeasible to invert by brute force.

## Sources

[^wiki]: [Block cipher](https://en.wikipedia.org/wiki/Block_cipher) — Wikipedia, for fixed-size block encryption and modes of operation.
