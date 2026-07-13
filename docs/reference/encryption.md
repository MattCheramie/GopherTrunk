---
slug: encryption
title: Encryption
entry_type: term
category: cryptography
description: Encryption is the process of transforming plaintext into ciphertext with a key so that only holders of the matching key can recover the original — distinct from keyless obfuscation or scrambling.
keywords: encryption, decryption, plaintext, ciphertext, key, cipher, confidentiality, symmetric, asymmetric, AES, DES, OTAR
aka: [encipherment]
autolink: true
infobox:
  - { label: Goal, value: "Confidentiality" }
  - { label: Needs, value: "A secret key" }
  - { label: Inverse, value: "Decryption" }
see_also: [cryptography, cipher, obfuscation, cryptanalysis, advanced-encryption-standard, data-encryption-standard, rc4-cipher, otar, key-id-algid, tetra-tea]
cite_urls:
  - https://en.wikipedia.org/wiki/Encryption
  - https://en.wikipedia.org/wiki/Advanced_Encryption_Standard
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

Encryption applies a [cipher](/reference/cipher/) parameterized by a secret
[key](/reference/cryptographic-key/). In [symmetric](/reference/symmetric-key-cryptography/)
encryption the same key encrypts and decrypts; in asymmetric
([public-key](/reference/public-key-cryptography/)) encryption a public key encrypts and a
private key decrypts. Either way, the defining property is that confidentiality depends on
the key, not on hiding the algorithm — the essence of
[Kerckhoffs's principle](/reference/kerckhoffs-principle/).

This is what separates encryption from look-alikes:

- **[Obfuscation](/reference/obfuscation/)** rearranges or hides data with *no secret key*,
  so anyone who learns the method can reverse it.
- **[Scrambling](/reference/scrambling/)/whitening** applies a fixed, public sequence to
  balance a signal, not to keep it secret.

Only encryption gives security that survives full public knowledge of the method.

## Variants

Beyond the symmetric/asymmetric split, encryption is characterized by *where* and *how* it
is applied. **Link (over-the-air) encryption** protects a single hop and is what most
land-mobile radio uses; **end-to-end encryption** protects content across every relay so
even the infrastructure cannot read it. Voice traffic is usually encrypted with a keystream
mode (a block cipher run in OFB, or a native stream cipher) so it can be applied continuously
to vocoder frames with no expansion. Each protected transmission signals which key and
algorithm it used through [key-ID / algorithm-ID](/reference/key-id-algid/) fields, and the
keys themselves are distributed by a key loader or by
[over-the-air rekeying](/reference/otar/).

## In practice

Encryption is only as strong as its key management. A perfect cipher is undone by reused
keys, predictable initialization vectors, or keys shared insecurely, which is why real
systems invest heavily in key loaders, rekeying, and key rotation. On public-safety radio
this shows up as OTAR infrastructure, crypto-net planning, and the operational headache of
keeping thousands of field radios in sync — problems entirely separate from the cipher math.

## Relevance to SDR

For a scanner, encryption is the hard wall. When [DMR](/reference/dmr/) Enhanced Privacy
([RC4](/reference/rc4-cipher/)) or [P25](/reference/project-25/)
[DES](/reference/data-encryption-standard/)-OFB /
[AES](/reference/advanced-encryption-standard/)-256 voice encryption is in use, GopherTrunk
can detect the encrypted call, read its [key-ID/algorithm-ID](/reference/key-id-algid/),
identify the talkgroup, and follow it across the trunked system, but it cannot recover audio
without the key — and it does not attempt to. The contrast matters because much of what
crosses the air only *looks* protected: a [scrambling](/reference/scrambling/) sequence is
reversible, a CRC is for integrity, and the Motorola talker-alias scheme analyzed in issue
#773 is keyless [obfuscation](/reference/obfuscation/) rather than encryption.
Distinguishing real encryption from the rest tells you immediately what is decodable.

## Sources

[^wiki]: [Encryption](https://en.wikipedia.org/wiki/Encryption) — Wikipedia, for the definition of transforming plaintext to ciphertext under a key and its inverse, decryption.
[^aes]: [Advanced Encryption Standard](https://en.wikipedia.org/wiki/Advanced_Encryption_Standard) — Wikipedia, for the block cipher used in P25 AES-256 encrypted voice.
