---
slug: cryptographic-key
title: Cryptographic key
entry_type: term
category: cryptography
description: A cryptographic key is the secret parameter that controls a cipher's transformation; the security of a well-designed system rests on the secrecy and size of the key, not on the algorithm, which may be public.
keywords: cryptographic key, secret key, key length, key space, key size, brute force, Kerckhoffs principle, symmetric key, private key, key management, encryption
aka: [key, secret key]
autolink: true
infobox:
  - { label: Role, value: Secret parameter of a cipher }
  - { label: Measured by, value: Key length / key space }
  - { label: Principle, value: Security rests in the key, not the algorithm }
see_also: [keystream, symmetric-key-cryptography, public-key-cryptography, kerckhoffs-principle, key-id-algid, key-loader-kfd, otar]
cite_urls:
  - https://en.wikipedia.org/wiki/Key_(cryptography)
  - https://en.wikipedia.org/wiki/Key_size
---

**A cryptographic key** is the secret value that controls how a cipher transforms data; with
the correct key the transformation can be reversed, and without it the data should be
infeasible to recover.[^wiki] In a sound system the key — not the algorithm — is the only
secret.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 100" role="img" aria-label="A key parameter feeds a cipher that turns plaintext into ciphertext; wrong keys fail." xmlns="http://www.w3.org/2000/svg">
  <rect x="40" y="44" width="60" height="24" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.2"/><text x="70" y="60" text-anchor="middle" font-size="9" fill="currentColor">key</text>
  <line x1="70" y1="44" x2="70" y2="24" stroke="currentColor"/><text x="70" y="18" text-anchor="middle" font-size="8" fill="currentColor">key space = 2^n</text>
  <line x1="100" y1="56" x2="150" y2="56" stroke="currentColor" marker-end="url(#ckeyar)"/>
  <rect x="152" y="42" width="80" height="28" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="192" y="60" text-anchor="middle" font-size="9" fill="currentColor">cipher</text>
  <text x="116" y="92" text-anchor="middle" font-size="8" fill="currentColor">plaintext</text><line x1="116" y1="88" x2="152" y2="70" stroke="currentColor" marker-end="url(#ckeyar)"/>
  <line x1="232" y1="56" x2="300" y2="56" stroke="currentColor" marker-end="url(#ckeyar)"/><text x="352" y="60" text-anchor="middle" font-size="9" fill="currentColor">ciphertext</text>
  <defs><marker id="ckeyar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>The key is the secret input to the cipher; its bit length sets the size of the key space.</figcaption>
</figure>

## How it works

A cipher is a fixed, often publicly documented algorithm; the **key** is the variable secret
that selects one specific transformation out of an astronomically large family. Change the key
and the same algorithm produces a completely different result.

The strength of a key is measured by its **key space** — the number of possible keys, set by
the key length in bits. An *n*-bit key has 2ⁿ possibilities, so a 128-bit key has 2¹²⁸ of
them, far beyond any feasible [brute-force search](/reference/brute-force-attack/).[^size]
This is why key length matters: a longer key exponentially enlarges the search an attacker
must perform. The bound is only an upper limit, though — it holds only if the *effective* key
space really is that large. A cipher with a short key (DES's 56 bits), predictable key
generation, or a mathematical weakness that lets an attacker rule out most keys at once has a
much smaller effective space than its nominal length suggests.

## Variants

Keys take different forms in different systems:

- A single shared secret in [symmetric-key cryptography](/reference/symmetric-key-cryptography/).
- A public/private pair in [public-key cryptography](/reference/public-key-cryptography/).
- A seed that is expanded into a [keystream](/reference/keystream/) in a stream cipher.

Systems also distinguish a *traffic key* (the short-lived key that actually encrypts a call)
from a *key-encryption key* (used only to wrap traffic keys for delivery), and they schedule
periodic **rekeying** so that compromise of one key exposes only a limited window.

## In practice

Because all the secrecy is concentrated in the key, **key management** — generating,
distributing, storing, and retiring keys — is the hard operational core of any deployed
cryptosystem. In land-mobile radio, keys are typically loaded into radios with a physical
[key loader / KFD](/reference/key-loader-kfd/) and can be refreshed remotely by
[over-the-air rekeying](/reference/otar/); each key carries a
[key ID and algorithm ID](/reference/key-id-algid/) so the receiver knows which key and cipher
a given transmission used. By
[Kerckhoffs's principle](/reference/kerckhoffs-principle/), a system should remain secure even
if everything except the key is public — so protecting the key *is* the whole security
argument.

## Relevance to SDR

Whether GopherTrunk can recover encrypted voice comes down to one thing: the key. The decoder
can identify an encrypted [P25](/reference/project-25/) or [DMR](/reference/dmr/) call, read
its [key-identifier and algorithm fields](/reference/key-id-algid/), and follow the traffic,
but the audio stays opaque without the secret key. That is the intended behaviour of a
correctly designed [symmetric](/reference/symmetric-key-cryptography/) cipher. It also
clarifies the difference from reversible [scrambling](/reference/scrambling/), which has no
secret key and so can be undone by anyone who knows the public method.

## Sources

[^wiki]: [Key (cryptography)](https://en.wikipedia.org/wiki/Key_(cryptography)) — Wikipedia, for the key as the secret parameter and the role of key length and key space.
[^size]: [Key size](https://en.wikipedia.org/wiki/Key_size) — Wikipedia, for how key length bounds brute-force effort and the notion of effective vs nominal key strength.
