---
slug: symmetric-key-cryptography
title: Symmetric-key cryptography
entry_type: term
category: cryptography
description: Symmetric-key cryptography uses one shared secret key for both encryption and decryption; it is fast and compact but requires the key to be distributed securely to every party in advance.
keywords: symmetric-key cryptography, secret key, shared key, AES, DES, RC4, key distribution, stream cipher, block cipher, encryption
aka: [symmetric encryption, secret-key cryptography]
autolink: true
infobox:
  - { label: Key model, value: One shared secret key }
  - { label: Examples, value: AES, DES, RC4 }
  - { label: Trade-off, value: Fast, but key distribution is hard }
see_also: [cryptographic-key, public-key-cryptography, keystream, kerckhoffs-principle]
cite_urls:
  - https://en.wikipedia.org/wiki/Symmetric-key_algorithm
---

**Symmetric-key cryptography** uses the *same* secret key to encrypt and to decrypt, so the
sender and receiver must both hold that key and keep it secret.[^wiki] It underlies the fast
bulk ciphers such as AES, DES, and [RC4](/reference/rc4-cipher/).

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 100" role="img" aria-label="One shared key feeds both encryption of plaintext into ciphertext and decryption back to plaintext." xmlns="http://www.w3.org/2000/svg">
  <rect x="200" y="10" width="60" height="22" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.2"/><text x="230" y="25" text-anchor="middle" font-size="9" fill="currentColor">shared key</text>
  <line x1="220" y1="32" x2="120" y2="56" stroke="currentColor" marker-end="url(#symkar)"/>
  <line x1="240" y1="32" x2="340" y2="56" stroke="currentColor" marker-end="url(#symkar)"/>
  <text x="20" y="64" font-size="9" fill="currentColor">plaintext</text>
  <rect x="90" y="56" width="60" height="22" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="120" y="71" text-anchor="middle" font-size="8" fill="currentColor">encrypt</text>
  <line x1="150" y1="67" x2="200" y2="67" stroke="currentColor" marker-end="url(#symkar)"/>
  <text x="225" y="71" text-anchor="middle" font-size="8" fill="currentColor">cipher</text>
  <line x1="252" y1="67" x2="310" y2="67" stroke="currentColor" marker-end="url(#symkar)"/>
  <rect x="312" y="56" width="60" height="22" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="342" y="71" text-anchor="middle" font-size="8" fill="currentColor">decrypt</text>
  <line x1="372" y1="67" x2="410" y2="67" stroke="currentColor" marker-end="url(#symkar)"/><text x="412" y="71" font-size="9" fill="currentColor">plaintext</text>
  <defs><marker id="symkar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>One secret key encrypts and decrypts — both ends must already share it.</figcaption>
</figure>

## How it works

A single [cryptographic key](/reference/cryptographic-key/) parameterises both directions of
the cipher. Encryption transforms plaintext to ciphertext under the key; decryption is the
inverse transform under the *same* key. Because there is only one key and no public-key
mathematics, symmetric ciphers are extremely fast and are used for nearly all bulk data
encryption.

Symmetric ciphers come in two main shapes:

- **Stream ciphers** generate a [keystream](/reference/keystream/) and combine it with the
  data, usually by XOR — for example [RC4](/reference/rc4-cipher/).
- **Block ciphers** transform fixed-size blocks under the key — for example AES (128-bit
  blocks) and DES (64-bit blocks).

The central weakness is *key distribution*: every pair of parties needs a shared secret,
delivered over some channel that an eavesdropper cannot read. [Public-key
cryptography](/reference/public-key-cryptography/) was developed largely to solve this
problem. Per [Kerckhoffs's principle](/reference/kerckhoffs-principle/), the algorithm may be
public; only the key must stay secret.

## Relevance to SDR

Most trunked-radio voice encryption is symmetric. [DMR](/reference/dmr/) "Enhanced Privacy"
uses [RC4](/reference/rc4-cipher/); [P25](/reference/project-25/) voice protection uses
DES-OFB or AES-256. In every case GopherTrunk can detect and follow the encrypted traffic but
cannot recover the audio without the shared key — that is the whole point of a symmetric
cipher. This is distinct from reversible [scrambling](/reference/scrambling/) or whitening,
which use a publicly known sequence and so can be undone without any secret.

## Sources

[^wiki]: [Symmetric-key algorithm](https://en.wikipedia.org/wiki/Symmetric-key_algorithm) — Wikipedia, for the shared-secret-key model and its key-distribution trade-off.
