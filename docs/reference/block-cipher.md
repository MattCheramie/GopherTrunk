---
slug: block-cipher
title: Block cipher
entry_type: algorithm
category: cryptography
description: A block cipher is a symmetric cipher that encrypts fixed-size blocks under a key using an invertible keyed permutation; a mode of operation chains blocks to cover messages of any length.
keywords: block cipher, AES, DES, block size, mode of operation, ECB, CBC, CTR, OFB, confusion, diffusion, Feistel, SPN, symmetric encryption
aka: [block cipher]
autolink: true
infobox:
  - { label: Type, value: "Symmetric cipher" }
  - { label: Unit, value: "Fixed-size block" }
  - { label: Examples, value: "AES (128-bit), DES (64-bit)" }
see_also: [stream-cipher, advanced-encryption-standard, data-encryption-standard, feistel-network, substitution-permutation-network, s-box]
cite_urls:
  - https://en.wikipedia.org/wiki/Block_cipher
  - https://csrc.nist.gov/pubs/sp/800/38/a/final
---

**A block cipher** encrypts data in fixed-size blocks — for example 128 bits in
[AES](/reference/advanced-encryption-standard/) or 64 bits in
[DES](/reference/data-encryption-standard/) — transforming each block under a key with a
keyed, invertible permutation.[^wiki] A *mode of operation* then chains the blocks so the
cipher can handle messages of any length. Because encryption and decryption use the same
secret key, a block cipher is a [symmetric](/reference/symmetric-key-cryptography/) primitive
and the building block behind most modern data encryption.

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
key can reverse it. A block of *n* bits has 2ⁿ possible values, and the key selects one
particular reordering of that enormous set — far too many to tabulate, so the permutation is
computed on the fly from many simple rounds.

Two design goals, named by Claude Shannon, drive every round:

- **Confusion** makes the relationship between key and ciphertext as complex as possible,
  supplied by nonlinear [S-box](/reference/s-box/) substitution.
- **Diffusion** spreads the influence of each plaintext bit across the whole block, supplied
  by permutation and mixing so that changing one input bit flips about half the output bits
  (the avalanche effect).

Two structural families realize this. A [Feistel network](/reference/feistel-network/) (DES)
splits the block in two and mixes one half into the other through a round function that need
not itself be invertible. A [substitution-permutation network](/reference/substitution-permutation-network/)
(AES) transforms the whole block each round with invertible substitution and permutation
layers. Both repeat many keyed rounds so the output depends intricately on every input and
key bit.

## Variants — modes of operation

Because real messages are longer than one block, a **mode of operation** specifies how
successive blocks combine.[^nist] The mode, not the cipher, determines the security of the
message as a whole:

- **ECB** (Electronic Codebook) encrypts each block independently — simple but insecure,
  since identical plaintext blocks yield identical ciphertext blocks and leak structure (the
  classic "ECB penguin").
- **CBC** (Cipher Block Chaining) XORs each plaintext block with the previous ciphertext
  block before encrypting, so identical plaintext no longer maps to identical ciphertext;
  it needs a random initialization vector and is inherently sequential.
- **CTR** (Counter) and **OFB** (Output Feedback) do not encrypt the plaintext at all —
  they encrypt a counter or a feedback register to generate a [keystream](/reference/keystream/),
  then XOR it with the data, turning the block cipher into a
  [stream cipher](/reference/stream-cipher/). These need only the forward direction of the
  cipher and demand a never-repeating nonce.

## In practice

Block size and key size matter independently. DES's 64-bit block and 56-bit key are both too
small today — the key falls to brute force and the small block invites birthday-bound
collisions on long streams — which is why AES moved to a 128-bit block and 128/192/256-bit
keys. In real protocols the raw block cipher is almost never used alone: it is wrapped in a
mode (often CTR or an authenticated mode like GCM) that supplies the length handling and,
increasingly, integrity as well.

## Relevance to SDR

Block ciphers underpin the strong encryption on the digital systems GopherTrunk monitors.
P25 voice may use DES or AES-256, and DMR encryption options likewise build on AES — but
these protocols run the block cipher in a feedback mode (OFB/CTR), so on air the protected
voice is delivered as a [stream cipher](/reference/stream-cipher/) over the vocoder frames.
The practical upshot is the same: GopherTrunk can recognize and follow an encrypted call but
cannot recover audio without the key, since the underlying permutation is infeasible to
invert by brute force.

## Sources

[^wiki]: [Block cipher](https://en.wikipedia.org/wiki/Block_cipher) — Wikipedia, for fixed-size block encryption, confusion/diffusion, and the Feistel vs SPN structures.
[^nist]: [SP 800-38A, Recommendation for Block Cipher Modes of Operation](https://csrc.nist.gov/pubs/sp/800/38/a/final) — NIST, for the ECB, CBC, CFB, OFB, and CTR modes of operation.
