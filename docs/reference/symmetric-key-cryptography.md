---
slug: symmetric-key-cryptography
title: Symmetric-key cryptography
entry_type: term
category: cryptography
description: Symmetric-key cryptography uses one shared secret key for both encryption and decryption; it is fast and compact but requires the key to be distributed securely to every party in advance.
keywords: symmetric-key cryptography, secret key, shared key, AES, DES, RC4, key distribution, stream cipher, block cipher, mode of operation, encryption
aka: [symmetric encryption, secret-key cryptography]
autolink: true
infobox:
  - { label: Key model, value: One shared secret key }
  - { label: Examples, value: AES, DES, RC4 }
  - { label: Trade-off, value: Fast, but key distribution is hard }
see_also: [cryptographic-key, public-key-cryptography, keystream, kerckhoffs-principle, p25-des-xl, tetra-tea]
cite_urls:
  - https://en.wikipedia.org/wiki/Symmetric-key_algorithm
  - https://csrc.nist.gov/pubs/fips/197/final
---

**Symmetric-key cryptography** uses the *same* secret key to encrypt and to decrypt, so the
sender and receiver must both hold that key and keep it secret.[^wiki] It underlies the fast
bulk ciphers such as AES, DES, and [RC4](/reference/rc4-cipher/), and is the workhorse of
almost every system that protects real traffic in volume.

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
mathematics — no modular exponentiation over thousand-bit numbers, no elliptic-curve point
multiplication — symmetric ciphers run one to three orders of magnitude faster than their
[asymmetric](/reference/public-key-cryptography/) counterparts and are used for essentially
all bulk data encryption. A well-designed symmetric cipher aims for *confusion* (each
ciphertext bit depends on the key in a complicated way) and *diffusion* (flipping one
plaintext bit changes about half the ciphertext bits), so the output is statistically
indistinguishable from random and resists [frequency analysis](/reference/frequency-analysis/).

## Variants

Symmetric ciphers come in two main shapes:

- **Stream ciphers** generate a [keystream](/reference/keystream/) and combine it with the
  data, usually by XOR — for example [RC4](/reference/rc4-cipher/). They encrypt one bit or
  byte at a time and are natural for continuous media such as digital voice.
- **Block ciphers** transform fixed-size blocks under the key — for example AES (128-bit
  blocks) and DES (64-bit blocks). A **mode of operation** turns a block cipher into a way to
  encrypt arbitrary-length data: ECB (blocks independently, weak), CBC (chained), CTR, and
  OFB (which runs the block cipher as a keystream generator, blurring the two categories).

The block/stream distinction is not absolute. P25 voice protection running DES or AES in
output-feedback (OFB) mode uses a block cipher precisely to *manufacture* a keystream that is
then XORed with the vocoder bits, so the radio treats a block cipher exactly like a stream
cipher over the air.

## In practice

The central weakness is *key distribution*: every pair of parties needs a shared secret,
delivered over some channel an eavesdropper cannot read. In a fleet of hundreds of radios this
becomes a logistics problem — keys are loaded physically with a
[key loader](/reference/key-loader-kfd/) or refreshed remotely by
[over-the-air rekeying](/reference/otar/), and each key is tagged with a
[key ID and algorithm ID](/reference/key-id-algid/) so a receiver knows which key and cipher a
given call used. [Public-key cryptography](/reference/public-key-cryptography/) was developed
largely to solve the distribution problem, and modern systems often combine the two: an
asymmetric exchange delivers a fresh symmetric key, then the fast symmetric cipher carries the
traffic. Per [Kerckhoffs's principle](/reference/kerckhoffs-principle/), the algorithm may be
public; only the key must stay secret.

## Relevance to SDR

Most trunked-radio voice encryption is symmetric. [DMR](/reference/dmr/) "Enhanced Privacy"
uses [RC4](/reference/rc4-cipher/); [P25](/reference/project-25/) voice protection uses
DES-OFB, [DES-XL](/reference/p25-des-xl/), or AES-256; [TETRA](/reference/tetra/) uses the
[TEA](/reference/tetra-tea/) family. In every case GopherTrunk can detect the encrypted
traffic, read the [key ID / algorithm ID](/reference/key-id-algid/) that identifies it, and
follow the call, but cannot recover the audio without the shared key — that is the whole point
of a symmetric cipher. This is distinct from reversible
[scrambling](/reference/scrambling/) or whitening, which use a publicly known sequence and so
can be undone without any secret. GopherTrunk decodes clear and scrambled traffic; it does not
attempt to break keyed symmetric encryption.

## Sources

[^wiki]: [Symmetric-key algorithm](https://en.wikipedia.org/wiki/Symmetric-key_algorithm) — Wikipedia, for the shared-secret-key model and its key-distribution trade-off.
[^fips]: [FIPS 197: Advanced Encryption Standard (AES)](https://csrc.nist.gov/pubs/fips/197/final) — NIST, the standard defining the most widely deployed symmetric block cipher.
