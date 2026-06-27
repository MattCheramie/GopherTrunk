---
slug: cipher
title: Cipher
entry_type: term
category: cryptography
description: A cipher is an algorithm for encryption and decryption — transforming plaintext into ciphertext under a key and back again — classified as stream or block, symmetric or asymmetric, and classical or modern.
keywords: cipher, encryption algorithm, stream cipher, block cipher, symmetric, asymmetric, classical cipher, plaintext, ciphertext, key
aka: [encryption algorithm]
autolink: true
infobox:
  - { label: Type, value: "Encryption/decryption algorithm" }
  - { label: By unit, value: "Stream or block" }
  - { label: By key, value: "Symmetric or asymmetric" }
see_also: [cryptography, encryption, cryptanalysis, obfuscation]
cite_urls:
  - https://en.wikipedia.org/wiki/Cipher
---

**A cipher** is an algorithm for [encryption](/reference/encryption/) and decryption —
transforming plaintext into ciphertext under a key and back again.[^wiki] The cipher is the
public, fixed procedure; the key is the secret that makes one party's output unreadable to
everyone else.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 100" role="img" aria-label="Plaintext and key enter a cipher to produce ciphertext." xmlns="http://www.w3.org/2000/svg">
  <rect x="20" y="30" width="70" height="22" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="55" y="45" text-anchor="middle" font-size="8" fill="currentColor">plaintext</text>
  <rect x="20" y="62" width="70" height="22" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="55" y="77" text-anchor="middle" font-size="8" fill="currentColor">key</text>
  <line x1="90" y1="41" x2="150" y2="50" stroke="currentColor" marker-end="url(#cpar)"/>
  <line x1="90" y1="73" x2="150" y2="60" stroke="currentColor" marker-end="url(#cpar)"/>
  <rect x="152" y="40" width="64" height="26" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.2"/><text x="184" y="57" text-anchor="middle" font-size="9" fill="currentColor">cipher</text>
  <line x1="216" y1="53" x2="276" y2="53" stroke="currentColor" marker-end="url(#cpar)"/>
  <text x="284" y="57" font-size="8" fill="currentColor">ciphertext</text>
  <defs><marker id="cpar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>A cipher combines plaintext and a key to produce ciphertext; the same key reverses it.</figcaption>
</figure>

## How it works

Ciphers are classified along a few independent axes:

- **By unit of operation** — a *stream cipher* processes data bit- or byte-at-a-time,
  usually by XOR with a keystream (RC4); a *block cipher* processes fixed-size blocks under
  a key (AES with 128-bit blocks, DES with 64-bit), chained together by a mode of operation.
- **By key relationship** — a *symmetric* cipher uses the same secret key to encrypt and
  decrypt; an *asymmetric* (public-key) cipher uses a public key to encrypt and a separate
  private key to decrypt.
- **By era** — *classical* ciphers (substitution, transposition) operate on letters and
  fall to [cryptanalysis](/reference/cryptanalysis/) such as frequency analysis; *modern*
  ciphers operate on bits and are designed against far stronger attacks.

A genuine cipher keeps its security in the key, per Kerckhoffs's principle. A reversible
transformation with no secret key is not a cipher but [obfuscation](/reference/obfuscation/).

## Relevance to SDR

Trunked-radio systems specify particular ciphers for protected voice. DMR Enhanced Privacy
uses the [RC4](/reference/rc4-cipher/) stream cipher; [P25](/reference/project-25/) voice
encryption uses DES-OFB or AES-256 block ciphers. Recognizing which class a system uses
tells GopherTrunk what to expect: a stream cipher leaves frame sizes intact while a block
cipher operates on fixed blocks, and in every case the audio stays unrecoverable without
the key. Transformations that carry no key — scrambling whitening, or the Motorola
talker-alias [obfuscation](/reference/obfuscation/) studied in issue #773 — are not ciphers
and can be reversed by anyone who works out the method.

## Sources

[^wiki]: [Cipher](https://en.wikipedia.org/wiki/Cipher) — Wikipedia, for the definition and the stream/block and symmetric/asymmetric classifications.
