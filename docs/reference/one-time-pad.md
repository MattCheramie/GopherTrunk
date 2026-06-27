---
slug: one-time-pad
title: One-time pad (OTP)
entry_type: algorithm
category: cryptography
description: The one-time pad encrypts a message by XORing it with a truly random key as long as the message and used only once; it is the only cipher proven information-theoretically unbreakable.
keywords: one-time pad, OTP, perfect secrecy, information-theoretic security, XOR, keystream, key reuse, Vernam cipher, stream cipher, random key
aka: [OTP, Vernam cipher]
autolink: true
infobox:
  - { label: Type, value: Symmetric cipher }
  - { label: Key, value: Random, message-length, single-use }
  - { label: Security, value: Information-theoretic (proven) }
see_also: [keystream, stream-cipher, cryptographic-key, linear-feedback-shift-register, encryption]
cite_urls:
  - https://en.wikipedia.org/wiki/One-time_pad
---

**The one-time pad (OTP)** encrypts a message by combining it (usually by XOR) with a key
that is truly random, at least as long as the message, and used only once — making it the
only cipher proven to be information-theoretically unbreakable.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 100" role="img" aria-label="A random pad XORed with plaintext yields ciphertext; the same pad XORed back recovers plaintext." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <text x="55" y="40">plaintext</text>
    <text x="55" y="80">random pad</text>
    <line x1="95" y1="36" x2="135" y2="50" stroke="currentColor"/>
    <line x1="100" y1="76" x2="135" y2="62" stroke="currentColor"/>
    <circle cx="150" cy="56" r="13" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="150" y="60" font-size="12">⊕</text>
    <line x1="163" y1="56" x2="210" y2="56" stroke="currentColor" marker-end="url(#otpar)"/>
    <rect x="212" y="42" width="70" height="28" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.2"/><text x="247" y="60">ciphertext</text>
    <line x1="282" y1="56" x2="320" y2="56" stroke="currentColor" marker-end="url(#otpar)"/>
    <circle cx="338" cy="56" r="13" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="338" y="60" font-size="12">⊕</text>
    <text x="338" y="86">same pad</text><line x1="338" y1="82" x2="338" y2="69" stroke="currentColor"/>
    <line x1="351" y1="56" x2="400" y2="56" stroke="currentColor" marker-end="url(#otpar)"/><text x="425" y="60">plaintext</text>
  </g>
  <defs><marker id="otpar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>XOR with a single-use random pad gives perfect secrecy; XOR with the same pad recovers the message.</figcaption>
</figure>

## How it works

Each bit (or byte) of plaintext is XORed with the matching bit of the pad. Because every
pad bit is uniformly random and independent, the ciphertext is also uniformly random: for
any given ciphertext, *every* plaintext of that length is equally likely, so the ciphertext
reveals nothing about the message. Decryption is the same XOR with the same pad. This is the
ideal that a practical [stream cipher](/reference/stream-cipher/) approximates by replacing
the random pad with a [keystream](/reference/keystream/).

Three conditions are mandatory, and all three are what make the proof hold:

- the key is **truly random**, not pseudo-random;
- the key is **at least as long** as the message;
- the key is **used only once** — hence the name.

Break any one and the guarantee collapses. Reusing a pad is the classic failure: XOR two
ciphertexts encrypted under the same pad and the pad cancels, leaving the XOR of the two
plaintexts open to analysis. A pseudo-random "pad" from a
[linear-feedback shift register](/reference/linear-feedback-shift-register/) is not random
and can be solved for, so it is not a one-time pad.

## Relevance to SDR

The one-time pad rarely appears on the air — distributing message-length random
[keys](/reference/cryptographic-key/) is impractical for routine radio traffic — but it is
the conceptual yardstick for everything GopherTrunk sees. Practical voice
[encryption](/reference/encryption/) such as P25's AES or DES key-stream modes is a
*finite-key* approximation of the pad: a short key drives a long pseudo-random keystream.
The pad's "never reuse" rule is exactly why those modes use per-transmission message
indicators, and why keystream reuse (a real-world misconfiguration) is a recoverable
weakness rather than a theoretical one.

## Sources

[^wiki]: [One-time pad](https://en.wikipedia.org/wiki/One-time_pad) — Wikipedia, for perfect secrecy, the three key conditions, and the key-reuse failure.
