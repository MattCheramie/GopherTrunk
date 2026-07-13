---
slug: keystream
title: Keystream
entry_type: term
category: cryptography
description: A keystream is the pseudo-random sequence a stream cipher generates from a key and combines, usually by XOR, with the plaintext; reusing a keystream catastrophically weakens the cipher.
keywords: keystream, stream cipher, XOR, pseudo-random, key reuse, two-time pad, RC4, OFB, initialization vector, nonce, encryption
aka: [key stream]
autolink: true
infobox:
  - { label: Produced by, value: Stream cipher from a key }
  - { label: Combined via, value: XOR with plaintext }
  - { label: Rule, value: Never reuse a keystream }
see_also: [cryptographic-key, symmetric-key-cryptography, kerckhoffs-principle, p25-des-xl, tetra-tea]
cite_urls:
  - https://en.wikipedia.org/wiki/Keystream
  - https://en.wikipedia.org/wiki/Stream_cipher
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
is its own inverse. Crucially the keystream depends only on the key and nonce, *not* on the
plaintext — so the sender and receiver, holding the same secrets, produce the same stream
independently and never have to transmit it.

## Variants

Keystreams are generated two broadly different ways. A **dedicated stream cipher** such as
[RC4](/reference/rc4-cipher/) has an internal state (RC4's is a 256-byte permutation) that it
stirs to emit each byte. Alternatively, a **block cipher run in a stream mode** — output
feedback (OFB) or counter (CTR) — repeatedly encrypts a counter or feedback value to
manufacture keystream from a block primitive like DES or AES. Either way the result is the
same over the air: a pseudo-random stream XORed onto the data. The design tension is the
nonce/IV. A *synchronous* cipher generates keystream independent of the ciphertext, so a lost
bit desynchronises everything after it; a *self-synchronising* cipher derives each keystream
symbol partly from recent ciphertext, so it recovers after an error at the cost of propagating
that error briefly.

## In practice

The cardinal rule is that **a keystream must never be reused** under the same key/nonce. If
two messages are XORed with the same keystream, an attacker who XORs the two ciphertexts
cancels the keystream entirely and is left with the XOR of the two plaintexts — a "two-time
pad" that often unravels with simple analysis. Systems avoid this by mixing a fresh IV, frame
counter, or message number into the seed for every transmission; a cipher that reuses
keystream (as some early deployments of RC4 did) is broken regardless of key length. This is
also why the [one-time pad](/reference/one-time-pad/) is unbreakable only when its keystream
is truly random and used exactly once. As always, the secrecy lives in the key, not in the
generator, in line with [Kerckhoffs's principle](/reference/kerckhoffs-principle/).

## Relevance to SDR

[DMR](/reference/dmr/) "Enhanced Privacy" protects voice with the
[RC4](/reference/rc4-cipher/) keystream; [P25](/reference/project-25/) uses DES or
[DES-XL](/reference/p25-des-xl/) and AES in OFB, which is a block cipher used as a keystream
generator; and [TETRA](/reference/tetra/) [TEA](/reference/tetra-tea/) is likewise a stream
construction. For GopherTrunk this draws a sharp line: a keystream produced from a secret
[key](/reference/cryptographic-key/) cannot be reproduced without that key, so the voice is
unrecoverable — whereas a publicly defined [scrambling](/reference/scrambling/) or whitening
sequence carries no secret and can simply be undone. Knowing which one a system uses tells you
immediately whether decoding is even possible.

## Sources

[^wiki]: [Keystream](https://en.wikipedia.org/wiki/Keystream) — Wikipedia, for the key-derived pseudo-random stream XORed with plaintext and the danger of reuse.
[^stream]: [Stream cipher](https://en.wikipedia.org/wiki/Stream_cipher) — Wikipedia, for synchronous vs self-synchronising generation and the role of the IV/nonce.
