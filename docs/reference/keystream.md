---
slug: keystream
title: Keystream
entry_type: term
category: cryptography
description: A keystream is the pseudo-random sequence a stream cipher generates from a key and combines, usually by XOR, with the plaintext; reusing a keystream catastrophically weakens the cipher.
keywords: keystream, stream cipher, XOR, pseudo-random, key reuse, two-time pad, RC4, initialization vector, nonce, encryption
aka: [key stream]
autolink: true
infobox:
  - { label: Produced by, value: Stream cipher from a key }
  - { label: Combined via, value: XOR with plaintext }
  - { label: Rule, value: Never reuse a keystream }
see_also: [cryptographic-key, symmetric-key-cryptography, kerckhoffs-principle]
cite_urls:
  - https://en.wikipedia.org/wiki/Keystream
---

**A keystream** is the sequence of pseudo-random bits or bytes a stream cipher derives from
its key, which is then combined with the plaintext — almost always by XOR — to produce
ciphertext.[^wiki] Recovering the data requires regenerating the identical keystream, which
needs the [key](/reference/cryptographic-key/).

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 100" role="img" aria-label="A key generates a keystream that is XORed bit by bit with plaintext to give ciphertext." xmlns="http://www.w3.org/2000/svg">
  <rect x="24" y="40" width="46" height="24" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="47" y="56" text-anchor="middle" font-size="9" fill="currentColor">key</text>
  <line x1="70" y1="52" x2="104" y2="52" stroke="currentColor" marker-end="url(#kstar)"/>
  <rect x="106" y="40" width="86" height="24" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.2"/><text x="149" y="56" text-anchor="middle" font-size="8" fill="currentColor">keystream</text>
  <line x1="192" y1="52" x2="234" y2="52" stroke="currentColor" marker-end="url(#kstar)"/>
  <circle cx="252" cy="52" r="12" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="252" y="56" text-anchor="middle" font-size="11" fill="currentColor">⊕</text>
  <text x="252" y="90" text-anchor="middle" font-size="8" fill="currentColor">plaintext</text><line x1="252" y1="86" x2="252" y2="64" stroke="currentColor"/>
  <line x1="264" y1="52" x2="332" y2="52" stroke="currentColor" marker-end="url(#kstar)"/><text x="384" y="56" text-anchor="middle" font-size="9" fill="currentColor">ciphertext</text>
  <defs><marker id="kstar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>The keystream is XORed with the data; decryption regenerates the same stream and XORs again.</figcaption>
</figure>

## How it works

A stream cipher uses the [key](/reference/cryptographic-key/) (often together with a per-message
nonce or initialisation vector) to seed a generator that emits a long, statistically random
keystream. Encryption combines this stream with the plaintext bit by bit or byte by byte,
typically with XOR; decryption regenerates the *identical* stream and XORs again, because XOR
is its own inverse. [RC4](/reference/rc4-cipher/) is a well-known example.

The cardinal rule is that **a keystream must never be reused** under the same key/nonce. If
two messages are XORed with the same keystream, an attacker who XORs the two ciphertexts
cancels the keystream entirely and is left with the XOR of the two plaintexts — a "two-time
pad" that often unravels with simple analysis. This is also why the
[one-time pad](/reference/symmetric-key-cryptography/) is unbreakable only when its
keystream is truly random and used exactly once. As always, the secrecy lives in the key,
not in the generator, in line with [Kerckhoffs's
principle](/reference/kerckhoffs-principle/).

## Relevance to SDR

[DMR](/reference/dmr/) "Enhanced Privacy" protects voice with the
[RC4](/reference/rc4-cipher/) keystream, and other trunked systems use keystream-based or
keystream-like modes. For GopherTrunk this draws a sharp line: a keystream produced from a
secret [key](/reference/cryptographic-key/) cannot be reproduced without that key, so the
voice is unrecoverable — whereas a publicly defined [scrambling](/reference/scrambling/) or
whitening sequence carries no secret and can simply be undone. Knowing which one a system uses
tells you immediately whether decoding is even possible.

## Sources

[^wiki]: [Keystream](https://en.wikipedia.org/wiki/Keystream) — Wikipedia, for the key-derived pseudo-random stream XORed with plaintext and the danger of reuse.
