---
slug: hash-function
title: Cryptographic hash function
entry_type: algorithm
category: cryptography
description: A cryptographic hash function maps arbitrary-length input to a fixed-size digest in a one-way fashion, with pre-image and collision resistance; it provides integrity and authentication, not secrecy.
keywords: hash function, cryptographic hash, digest, SHA-256, SHA-3, pre-image resistance, second pre-image, collision resistance, one-way function, Merkle-Damgard, sponge, HMAC, MAC, integrity, fingerprint, message digest, CRC
aka: [hash, message digest, SHA]
autolink: true
infobox:
  - { label: Type, value: One-way function }
  - { label: Output, value: Fixed-size digest }
  - { label: Goal, value: Integrity, not secrecy }
see_also: [cyclic-redundancy-check, advanced-encryption-standard, block-cipher, cryptographic-key, cryptography, frequency-analysis, encryption]
cite_urls:
  - https://en.wikipedia.org/wiki/Cryptographic_hash_function
  - https://en.wikipedia.org/wiki/SHA-3
---

**A cryptographic hash function** maps input of any length to a fixed-size *digest* in a
way that is easy to compute but practically impossible to invert, so the digest acts as a
compact, tamper-evident fingerprint of the data.[^wiki] It is one of the workhorses of
[cryptography](/reference/cryptography/), underpinning integrity checks, digital signatures,
password storage, and message authentication.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 100" role="img" aria-label="Variable-length input passes through a hash function box and out as a fixed-size digest." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <text x="60" y="40">message</text>
    <text x="60" y="56" font-size="8">(any length)</text>
    <line x1="105" y1="52" x2="150" y2="52" stroke="currentColor" marker-end="url(#hashar)"/>
    <rect x="152" y="34" width="110" height="36" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.2"/><text x="207" y="56" font-size="10">hash function</text>
    <line x1="262" y1="52" x2="308" y2="52" stroke="currentColor" marker-end="url(#hashar)"/>
    <rect x="310" y="38" width="100" height="28" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="360" y="50" font-size="8">fixed-size</text><text x="360" y="61" font-size="8">digest</text>
  </g>
  <defs><marker id="hashar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>A hash function compresses any input into a fixed-size digest that is easy to compute but hard to reverse.</figcaption>
</figure>

## How it works

A good cryptographic hash behaves like a deterministic but unpredictable function. Flipping
a single input bit changes roughly half the output bits (the *avalanche* effect), and the
function is designed to resist three distinct attacks:

- **Pre-image resistance** — given a digest *h*, you cannot feasibly find any input *m* with
  hash(*m*) = *h* (the function is one-way). For an *n*-bit digest this should cost about 2ⁿ
  work.
- **Second-pre-image resistance** — given one input *m₁*, you cannot find a different *m₂*
  with the same digest. Also ~2ⁿ work.
- **Collision resistance** — you cannot feasibly find *any* two distinct inputs that hash to
  the same digest. The birthday paradox halves the exponent here: collisions cost only ~2^(n/2)
  work, which is why SHA-256 (128-bit collision strength) is chosen where 128-bit *security*
  is wanted.

Because the output is fixed-size and the input is unbounded, collisions must exist in
principle; the security claim is only that they are computationally infeasible to find. When
one of these properties breaks, the hash is "broken" — MD5 and SHA-1 both fell to practical
collision attacks and are now unsafe for signatures, though pre-image resistance held longer.
A hash uses no [key](/reference/cryptographic-key/) and does not hide the data, so it is *not*
[encryption](/reference/encryption/) — anyone can recompute it.

## Variants — how the pieces fit together

Two dominant construction styles turn a fixed-size mixing primitive into a hash for
arbitrary-length input:

- **Merkle–Damgård** — used by MD5, SHA-1, and the SHA-2 family. The message is padded and
  split into blocks; a *compression function* absorbs one block at a time into a running
  chaining value, starting from a fixed initialization vector. It is simple and provably
  collision-resistant if the compression function is, but its length-extension property
  (an attacker who knows hash(*m*) can compute hash(*m* ‖ suffix)) is a footgun.
- **Sponge** — used by SHA-3 (Keccak). A single wide permutation alternately *absorbs*
  message blocks into part of its state and then *squeezes* out the digest. Sponges avoid
  length extension and can emit output of any length (extendable-output functions, XOFs).[^sha3]

Many SHA-2 compression functions are themselves built like a [block cipher](/reference/block-cipher/)
run in a one-way (Davies–Meyer) mode, so the same confusion-and-diffusion machinery that
powers ciphers also powers hashes.

## In practice — MACs and authentication

A bare hash proves a file was not *accidentally* changed, but anyone can recompute a digest,
so it does not prove *who* produced it. To authenticate a message you combine the hash with a
secret [key](/reference/cryptographic-key/). **HMAC** does this by hashing the key together
with the message in a nested construction (roughly hash(key ⊕ opad ‖ hash(key ⊕ ipad ‖ msg))),
which is secure even on length-extendable Merkle–Damgård hashes. The result — a *message
authentication code* — lets a receiver who shares the key verify both integrity and origin.
Hashes also derive keys (HKDF), store passwords (with deliberately slow variants like bcrypt
or Argon2), and bind data in digital signatures.

## Relevance to SDR

Hashing is about *integrity*, which on the radio is usually handled by something simpler. A
[cyclic redundancy check](/reference/cyclic-redundancy-check/) protects P25/DMR frames
against accidental bit errors, but a CRC is *not* a cryptographic hash: it is short, linear,
and trivial to forge on purpose, so it detects noise, not tampering. GopherTrunk uses CRCs
to validate decoded frames, and the distinction matters — a valid CRC means "probably not
corrupted by the channel," never "authenticated." Cryptographic hashes proper live in the
key-management and authentication layers of secure systems (deriving or checking
[AES](/reference/advanced-encryption-standard/) keys, signing firmware, authenticating
control messages), which a scanner observes only as opaque encrypted or signed traffic.

## Sources

[^wiki]: [Cryptographic hash function](https://en.wikipedia.org/wiki/Cryptographic_hash_function) — Wikipedia, for pre-image and collision resistance, the avalanche effect, Merkle–Damgård, HMAC, and the integrity-not-secrecy distinction.
[^sha3]: [SHA-3](https://en.wikipedia.org/wiki/SHA-3) — Wikipedia, for the Keccak sponge construction and its contrast with Merkle–Damgård.
