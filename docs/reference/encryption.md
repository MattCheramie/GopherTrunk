---
slug: encryption
title: Encryption
entry_type: term
category: cryptography
description: Encryption is the process of transforming plaintext into ciphertext with a key so that only holders of the matching key can recover the original — distinct from keyless obfuscation or scrambling.
keywords: encryption, decryption, plaintext, ciphertext, key, cipher, confidentiality, symmetric, asymmetric, AES
aka: [encipherment]
autolink: true
infobox:
  - { label: Goal, value: "Confidentiality" }
  - { label: Needs, value: "A secret key" }
  - { label: Inverse, value: "Decryption" }
see_also: [cryptography, cipher, obfuscation, cryptanalysis]
cite_urls:
  - https://en.wikipedia.org/wiki/Encryption
---

**Encryption** transforms plaintext into ciphertext using a key, so that only holders of
the matching key can recover the original message.[^wiki] It is the practical means by
which [cryptography](/reference/cryptography/) delivers confidentiality, performed by a
[cipher](/reference/cipher/) and reversed by decryption.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 100" role="img" aria-label="Plaintext encrypted with a key into ciphertext, then decrypted back." xmlns="http://www.w3.org/2000/svg">
  <rect x="14" y="40" width="66" height="24" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="47" y="55" text-anchor="middle" font-size="8" fill="currentColor">plaintext</text>
  <line x1="80" y1="52" x2="120" y2="52" stroke="currentColor" marker-end="url(#enar)"/>
  <rect x="122" y="40" width="74" height="24" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.2"/><text x="159" y="55" text-anchor="middle" font-size="8" fill="currentColor">encrypt</text>
  <line x1="196" y1="52" x2="236" y2="52" stroke="currentColor" marker-end="url(#enar)"/>
  <rect x="238" y="40" width="66" height="24" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="271" y="55" text-anchor="middle" font-size="8" fill="currentColor">ciphertext</text>
  <line x1="304" y1="52" x2="344" y2="52" stroke="currentColor" marker-end="url(#enar)"/>
  <rect x="346" y="40" width="74" height="24" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.2"/><text x="383" y="55" text-anchor="middle" font-size="8" fill="currentColor">decrypt</text>
  <text x="159" y="80" text-anchor="middle" font-size="8" fill="currentColor">key</text><line x1="159" y1="76" x2="159" y2="64" stroke="currentColor"/>
  <text x="383" y="80" text-anchor="middle" font-size="8" fill="currentColor">key</text><line x1="383" y1="76" x2="383" y2="64" stroke="currentColor"/>
  <defs><marker id="enar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>Encryption needs a key both ways: without it, ciphertext stays opaque.</figcaption>
</figure>

## How it works

Encryption applies a [cipher](/reference/cipher/) parameterized by a secret key. In
symmetric encryption the same key encrypts and decrypts; in asymmetric (public-key)
encryption a public key encrypts and a private key decrypts. Either way, the defining
property is that confidentiality depends on the key, not on hiding the algorithm — the
essence of Kerckhoffs's principle.

This is what separates encryption from look-alikes:

- **[Obfuscation](/reference/obfuscation/)** rearranges or hides data with *no secret key*,
  so anyone who learns the method can reverse it.
- **Scrambling/whitening** applies a fixed, public sequence to balance a signal, not to
  keep it secret.

Only encryption gives security that survives full public knowledge of the method.

## Relevance to SDR

For a scanner, encryption is the hard wall. When [DMR](/reference/dmr/) Enhanced Privacy
([RC4](/reference/rc4-cipher/)) or [P25](/reference/project-25/) DES-OFB / AES-256 voice
encryption is in use, GopherTrunk can detect the encrypted call, identify the talkgroup,
and follow it across the trunked system, but it cannot recover audio without the key — and
it does not attempt to. The contrast matters because much of what crosses the air only
*looks* protected: a [scrambling](/reference/scrambling/) sequence is reversible, a CRC is
for integrity, and the Motorola talker-alias scheme analyzed in issue #773 is keyless
[obfuscation](/reference/obfuscation/) rather than encryption. Distinguishing real
encryption from the rest tells you immediately what is decodable.

## Sources

[^wiki]: [Encryption](https://en.wikipedia.org/wiki/Encryption) — Wikipedia, for the definition of transforming plaintext to ciphertext under a key and its inverse, decryption.
