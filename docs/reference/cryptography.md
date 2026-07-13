---
slug: cryptography
title: Cryptography
entry_type: term
category: cryptography
description: Cryptography is the science of securing information against adversaries — protecting confidentiality, integrity, authentication, and non-repudiation — using mathematical techniques such as encryption, hashing, and digital signatures.
keywords: cryptography, confidentiality, integrity, authentication, non-repudiation, encryption, cipher, cryptographic key, keystream, cryptanalysis, symmetric, public-key, hashing
aka: [crypto]
autolink: true
infobox:
  - { label: Field, value: "Information security" }
  - { label: Goals, value: "Confidentiality, integrity, authentication, non-repudiation" }
  - { label: Branches, value: "Symmetric, public-key, hashing" }
see_also: [cryptanalysis, cipher, encryption, obfuscation, symmetric-key-cryptography, public-key-cryptography, hash-function, kerckhoffs-principle, otar]
cite_urls:
  - https://en.wikipedia.org/wiki/Cryptography
  - https://en.wikipedia.org/wiki/Kerckhoffs%27s_principle
---

**Cryptography** is the science of securing information against adversaries — protecting the
confidentiality, integrity, authentication, and non-repudiation of data using mathematical
techniques.[^wiki] It is the constructive counterpart to [cryptanalysis](/reference/cryptanalysis/),
the study of breaking such systems, and it underlies everything from HTTPS to the encrypted
voice traffic a scanner encounters on public-safety radio.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 100" role="img" aria-label="Cryptography branches into confidentiality, integrity, and authentication goals." xmlns="http://www.w3.org/2000/svg">
  <rect x="20" y="38" width="110" height="26" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.2"/><text x="75" y="55" text-anchor="middle" font-size="9" fill="currentColor">cryptography</text>
  <line x1="130" y1="51" x2="178" y2="22" stroke="currentColor" marker-end="url(#cgar)"/>
  <line x1="130" y1="51" x2="178" y2="51" stroke="currentColor" marker-end="url(#cgar)"/>
  <line x1="130" y1="51" x2="178" y2="80" stroke="currentColor" marker-end="url(#cgar)"/>
  <text x="184" y="25" font-size="8" fill="currentColor">confidentiality</text>
  <text x="184" y="54" font-size="8" fill="currentColor">integrity</text>
  <text x="184" y="83" font-size="8" fill="currentColor">authentication</text>
  <defs><marker id="cgar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>Cryptography pursues several goals at once: keeping data secret, detecting tampering, and proving who sent it.</figcaption>
</figure>

## How it works

Cryptography combines several goals, only one of which is secrecy:

- **Confidentiality** — keeping data unreadable to anyone without the key, achieved by
  [encryption](/reference/encryption/) with a [cipher](/reference/cipher/).
- **Integrity** — detecting whether data has been altered, typically with
  [hash functions](/reference/hash-function/) or message authentication codes.
- **Authentication** — proving who produced a message.
- **Non-repudiation** — preventing a sender from later denying they sent it, usually via
  digital signatures.

Modern cryptography rests on [Kerckhoffs's principle](/reference/kerckhoffs-principle/): the
algorithm is assumed public and all the security lives in the secret
[key](/reference/cryptographic-key/). That distinguishes it from
[obfuscation](/reference/obfuscation/), which merely hides a method that anyone who learns
it can reverse.

## Variants

The field divides into three broad branches:

- **[Symmetric-key cryptography](/reference/symmetric-key-cryptography/)** uses one shared
  secret for both encryption and decryption. It is fast and is what protects bulk data —
  the [AES](/reference/advanced-encryption-standard/) and
  [DES](/reference/data-encryption-standard/) block ciphers, and stream ciphers like
  [RC4](/reference/rc4-cipher/), all live here. This is the branch land-mobile radio uses.
- **[Public-key cryptography](/reference/public-key-cryptography/)** uses a public/private
  key pair, solving the key-distribution problem and enabling digital signatures; it is the
  basis of TLS handshakes and PKI.
- **Hashing** produces a fixed-size fingerprint of data with no key, underpinning integrity
  checks and message authentication.

Symmetric systems still need a way to distribute keys securely; on radio networks that job
falls to key loaders and [over-the-air rekeying](/reference/otar/).

## In practice

A working cryptosystem is more than a cipher: it needs sound key management, correct modes
of operation, fresh initialization vectors, and integrity protection, because attackers
rarely break the math — they exploit reuse, weak keys, or protocol mistakes. On
public-safety radio the practical stack is a symmetric cipher plus a key-management scheme
([OTAR](/reference/otar/), [key-ID/algorithm-ID](/reference/key-id-algid/) signaling, and a
[key loader](/reference/key-loader-kfd/)).

## Relevance to SDR

A trunked-radio receiver constantly meets the products of cryptography. Voice traffic on
[DMR](/reference/dmr/) and [P25](/reference/project-25/) systems may be encrypted (DMR
Enhanced Privacy using [RC4](/reference/rc4-cipher/); P25 voice using
[DES](/reference/data-encryption-standard/)-OFB,
[DES-XL](/reference/p25-des-xl/), or [AES](/reference/advanced-encryption-standard/)-256;
[TETRA](/reference/tetra/) using the [TEA](/reference/tetra-tea/) algorithms), in which case
GopherTrunk can identify and follow the call — reading its
[key-ID/algorithm-ID](/reference/key-id-algid/) fields — but cannot recover the audio
without the key. Other on-air transformations are *not* cryptography in the security sense:
data-link CRCs provide integrity but no secrecy, and the Motorola P25 talker-alias scheme
is [obfuscation](/reference/obfuscation/) rather than [encryption](/reference/encryption/)
— it was analyzed clean-room in issue #773 using only publicly observable data. Telling
these apart is the first step in deciding what a scanner can decode.

## Sources

[^wiki]: [Cryptography](https://en.wikipedia.org/wiki/Cryptography) — Wikipedia, for the goals of cryptography and its distinction from cryptanalysis.
[^kerck]: [Kerckhoffs's principle](https://en.wikipedia.org/wiki/Kerckhoffs%27s_principle) — Wikipedia, for the foundational rule that security must reside in the key, not the secrecy of the algorithm.
