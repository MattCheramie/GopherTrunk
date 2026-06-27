---
slug: public-key-cryptography
title: Public-key cryptography
entry_type: term
category: cryptography
description: Public-key (asymmetric) cryptography uses a mathematically linked key pair — a public key that anyone may use to encrypt or verify, and a private key kept secret to decrypt or sign — solving the key-distribution problem.
keywords: public-key cryptography, asymmetric cryptography, key pair, public key, private key, RSA, ECC, digital signature, key distribution, encryption
aka: [asymmetric cryptography, asymmetric encryption]
autolink: true
infobox:
  - { label: Key model, value: Public/private key pair }
  - { label: Examples, value: RSA, ECC, Diffie-Hellman }
  - { label: Trade-off, value: Solves key distribution, but slower }
see_also: [cryptographic-key, symmetric-key-cryptography, kerckhoffs-principle]
cite_urls:
  - https://en.wikipedia.org/wiki/Public-key_cryptography
---

**Public-key cryptography** (also *asymmetric* cryptography) uses a linked pair of keys: a
**public key** that may be shared freely and a **private key** kept secret.[^wiki] Data
encrypted with the public key can only be decrypted with the matching private key, so no
shared secret has to be distributed in advance.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 100" role="img" aria-label="A public key encrypts plaintext into ciphertext and the matching private key decrypts it back." xmlns="http://www.w3.org/2000/svg">
  <rect x="84" y="10" width="72" height="22" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.2"/><text x="120" y="25" text-anchor="middle" font-size="9" fill="currentColor">public key</text>
  <rect x="304" y="10" width="72" height="22" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="340" y="25" text-anchor="middle" font-size="9" fill="currentColor">private key</text>
  <line x1="120" y1="32" x2="120" y2="56" stroke="currentColor" marker-end="url(#pubkar)"/>
  <line x1="340" y1="32" x2="340" y2="56" stroke="currentColor" marker-end="url(#pubkar)"/>
  <text x="20" y="71" font-size="9" fill="currentColor">plaintext</text>
  <rect x="90" y="56" width="60" height="22" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="120" y="71" text-anchor="middle" font-size="8" fill="currentColor">encrypt</text>
  <line x1="150" y1="67" x2="200" y2="67" stroke="currentColor" marker-end="url(#pubkar)"/>
  <text x="225" y="71" text-anchor="middle" font-size="8" fill="currentColor">cipher</text>
  <line x1="252" y1="67" x2="310" y2="67" stroke="currentColor" marker-end="url(#pubkar)"/>
  <rect x="312" y="56" width="60" height="22" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="342" y="71" text-anchor="middle" font-size="8" fill="currentColor">decrypt</text>
  <line x1="372" y1="67" x2="410" y2="67" stroke="currentColor" marker-end="url(#pubkar)"/><text x="412" y="71" font-size="9" fill="currentColor">plaintext</text>
  <defs><marker id="pubkar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>Two linked keys: the public key encrypts; only the private key decrypts.</figcaption>
</figure>

## How it works

The two keys are generated together and bound by a mathematical relationship — typically one
based on a problem believed hard to reverse, such as integer factorisation (RSA) or the
elliptic-curve discrete logarithm (ECC). The public key can be published openly; deriving the
private key from it is computationally infeasible.

This gives two complementary uses:

- **Encryption** — anyone encrypts with the recipient's public key, and only the holder of
  the matching private key can decrypt.
- **Digital signatures** — the holder signs with the private key, and anyone can verify with
  the public key, proving authenticity and integrity.

Because the heavy mathematics make asymmetric operations far slower than symmetric ones,
real systems usually use public-key cryptography only to agree on or transport a short
[symmetric key](/reference/symmetric-key-cryptography/), then switch to a fast symmetric
[cipher](/reference/cryptographic-key/) for the bulk data. As with all modern cryptography,
its safety rests on [Kerckhoffs's principle](/reference/kerckhoffs-principle/): the algorithm
is public and only the private key is secret.

## Relevance to SDR

Public-key cryptography rarely appears in the over-the-air voice path of conventional
trunked-radio protocols, which protect speech with fast [symmetric
ciphers](/reference/symmetric-key-cryptography/) instead. It is more likely to surface in key
management and provisioning — distributing or rekeying the symmetric traffic keys, and in
device authentication — rather than in the per-call audio GopherTrunk demodulates. For the
purposes of decoding received signals it is mostly background context, but it explains *how*
the symmetric keys that do protect voice get delivered.

## Sources

[^wiki]: [Public-key cryptography](https://en.wikipedia.org/wiki/Public-key_cryptography) — Wikipedia, for the public/private key-pair model and the key-distribution problem it solves.
