---
slug: stream-cipher
title: Stream cipher
entry_type: algorithm
category: cryptography
description: A stream cipher is a symmetric cipher that encrypts data one bit or byte at a time by combining it — usually via XOR — with a pseudo-random keystream derived from a secret key.
keywords: stream cipher, keystream, XOR, RC4, symmetric encryption, key reuse, bit cipher, pseudo-random sequence
aka: [stream cipher]
autolink: true
infobox:
  - { label: Type, value: "Symmetric cipher" }
  - { label: Unit, value: "Bit or byte at a time" }
  - { label: Operation, value: "Plaintext XOR keystream" }
see_also: [keystream, cipher, block-cipher, rc4-cipher, one-time-pad]
cite_urls:
  - https://en.wikipedia.org/wiki/Stream_cipher
---

**A stream cipher** encrypts data one bit or byte at a time by combining each unit of
plaintext — almost always with XOR — against a pseudo-random [keystream](/reference/keystream/)
generated from a secret key.[^wiki] It contrasts with a
[block cipher](/reference/block-cipher/), which transforms fixed-size blocks at once.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 100" role="img" aria-label="A keystream generator feeds a keystream into an XOR with the plaintext stream to produce ciphertext." xmlns="http://www.w3.org/2000/svg">
  <rect x="20" y="40" width="86" height="30" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="63" y="59" text-anchor="middle" font-size="8" fill="currentColor">keystream gen</text>
  <line x1="106" y1="55" x2="170" y2="55" stroke="currentColor" marker-end="url(#scar)"/>
  <circle cx="195" cy="55" r="13" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="195" y="59" text-anchor="middle" font-size="11" fill="currentColor">⊕</text>
  <text x="195" y="90" text-anchor="middle" font-size="8" fill="currentColor">plaintext</text><line x1="195" y1="86" x2="195" y2="68" stroke="currentColor"/>
  <line x1="208" y1="55" x2="320" y2="55" stroke="currentColor" marker-end="url(#scar)"/><text x="375" y="59" font-size="9" fill="currentColor">ciphertext</text>
  <defs><marker id="scar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>A stream cipher XORs each plaintext bit with a keystream bit; the same operation decrypts.</figcaption>
</figure>

## How it works

A stream cipher's strength lives entirely in its keystream generator. The key seeds a
state machine that emits a long pseudo-random sequence; XORing that sequence with the
plaintext gives the ciphertext, and XORing the *same* sequence with the ciphertext recovers
the plaintext, because XOR is its own inverse.

Two rules follow directly:

- **Never reuse a keystream.** If two messages are encrypted under the same keystream,
  XORing the two ciphertexts cancels the keystream and leaks the XOR of the two plaintexts.
  This is why stream ciphers pair the key with a per-message nonce or initialization vector.
- **The keystream must look random.** A predictable generator lets an attacker reconstruct
  the sequence without the key. The [one-time pad](/reference/one-time-pad/) is the
  idealized limit — a truly random keystream as long as the message — and is provably
  unbreakable, but practical ciphers use a short key to stretch a long pseudo-random stream.

[RC4](/reference/rc4-cipher/) is the best-known example; modern designs include ChaCha20.

## Relevance to SDR

Stream ciphers are the encryption scheme most often seen on the trunked systems GopherTrunk
monitors. DMR "Enhanced Privacy" uses [RC4](/reference/rc4-cipher/), and P25 voice can be
protected with DES-OFB or AES — both run their underlying [block cipher](/reference/block-cipher/)
in a feedback mode that turns it into a keystream generator, making the on-air result a
stream cipher. Without the key the keystream cannot be reproduced, so GopherTrunk can detect
and follow an encrypted call but cannot recover the audio — distinct from reversible
[scrambling](/reference/scrambling/), where the sequence is public.

## Sources

[^wiki]: [Stream cipher](https://en.wikipedia.org/wiki/Stream_cipher) — Wikipedia, for the per-symbol XOR-with-keystream model and the keystream-reuse pitfall.
