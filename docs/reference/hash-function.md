---
slug: hash-function
title: Cryptographic hash function
entry_type: algorithm
category: cryptography
description: A cryptographic hash function maps arbitrary-length input to a fixed-size digest in a one-way fashion, with pre-image and collision resistance; it provides integrity and authentication, not secrecy.
keywords: hash function, cryptographic hash, digest, SHA-256, pre-image resistance, collision resistance, one-way function, integrity, fingerprint, message digest, CRC
aka: [hash, message digest, SHA]
autolink: true
infobox:
  - { label: Type, value: One-way function }
  - { label: Output, value: Fixed-size digest }
  - { label: Goal, value: Integrity, not secrecy }
see_also: [cyclic-redundancy-check, advanced-encryption-standard, cryptographic-key, frequency-analysis, encryption]
cite_urls:
  - https://en.wikipedia.org/wiki/Cryptographic_hash_function
---

**A cryptographic hash function** maps input of any length to a fixed-size *digest* in a
way that is easy to compute but practically impossible to invert, so the digest acts as a
compact, tamper-evident fingerprint of the data.[^wiki]

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
function is designed to resist three things:

- **Pre-image resistance** — given a digest, you cannot feasibly find an input that
  produces it (the function is one-way).
- **Second-pre-image resistance** — given one input, you cannot find a different input with
  the same digest.
- **Collision resistance** — you cannot feasibly find *any* two distinct inputs that hash
  to the same digest.

Because the output is fixed-size and the input is unbounded, collisions must exist in
principle; the security claim is only that they are computationally infeasible to find.
Common examples are the SHA-2 and SHA-3 families. A hash uses no
[key](/reference/cryptographic-key/) and does not hide the data, so it is *not*
[encryption](/reference/encryption/) — anyone can recompute it.

## Relevance to SDR

Hashing is about *integrity*, which on the radio is usually handled by something simpler. A
[cyclic redundancy check](/reference/cyclic-redundancy-check/) protects P25/DMR frames
against accidental bit errors, but a CRC is *not* a cryptographic hash: it is short, linear,
and trivial to forge on purpose, so it detects noise, not tampering. GopherTrunk uses CRCs
to validate decoded frames, and the distinction matters — a valid CRC means "probably not
corrupted by the channel," never "authenticated." Cryptographic hashes proper show up in the
key-management and authentication layers of secure systems (deriving or checking
[AES](/reference/advanced-encryption-standard/) keys), which a scanner observes only as
opaque encrypted traffic.

## Sources

[^wiki]: [Cryptographic hash function](https://en.wikipedia.org/wiki/Cryptographic_hash_function) — Wikipedia, for pre-image and collision resistance, the avalanche effect, and the integrity-not-secrecy distinction.
