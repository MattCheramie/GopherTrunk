---
slug: dmr-encryption
title: DMR encryption
entry_type: algorithm
category: cryptography
description: DMR link-layer privacy XORs a keystream onto the voice frames — Motorola Enhanced Privacy uses RC4, and DMRA algorithms map to DES, 3DES, or AES in OFB mode; the cipher cores are standard but the key and IV derivation is vendor-specific and reverse-engineered.
keywords: DMR encryption, Enhanced Privacy, DMR RC4, DMR AES, DES OFB, keystream XOR, DMRA algid, Motorola Hytera privacy
aka: ["DMR privacy", "Enhanced Privacy", "DMR link-layer encryption"]
autolink: true
infobox:
  - { label: Construction, value: keystream XORed onto AMBE frames }
  - { label: RC4 core, value: Enhanced Privacy (algid 0x21) }
  - { label: Block ciphers, value: DES / 3DES / AES in OFB }
  - { label: Key/IV derivation, value: vendor-specific, reverse-engineered }
see_also: [rc4-cipher, keystream, advanced-encryption-standard, data-encryption-standard, ambe-plus-2]
cite_urls:
  - https://en.wikipedia.org/wiki/Digital_mobile_radio
  - https://en.wikipedia.org/wiki/RC4
---

**DMR encryption** is the link-layer privacy that scrambles voice on a DMR channel by XORing a
[keystream](/reference/keystream/) onto the coded voice frames.[^wiki] Several schemes exist. The
cipher *cores* are standard — Motorola "Enhanced Privacy" is [RC4](/reference/rc4-cipher/), and the
DMRA algorithm identifiers map to DES, triple-DES, and [AES](/reference/advanced-encryption-standard/)
— but the *key and IV derivation* from the air interface is proprietary, reverse-engineered, and
differs between Motorola and Hytera.[^rc4] For a monitoring tool this splits the problem cleanly: the
standard cores are straightforward to implement, and the hard, vendor-specific part is recovering the
key, which no amount of decoding the signalling accomplishes on its own.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 120" role="img" aria-label="A key and an initialization vector feed a cipher core — RC4 or a block cipher in output-feedback mode — which produces a keystream that is XORed with the encrypted voice frames to recover the clear AMBE payload." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" stroke-width="1.1" font-size="8" fill="currentColor" text-anchor="middle">
    <rect x="18" y="24" width="70" height="22" fill="currentColor" fill-opacity="0.22"/><text x="53" y="39">key</text>
    <rect x="18" y="58" width="70" height="22" fill="currentColor" fill-opacity="0.14"/><text x="53" y="73">IV</text>
    <rect x="120" y="38" width="120" height="30" fill="none"/><text x="180" y="52">cipher core</text><text x="180" y="63" font-size="7">RC4 / OFB block</text>
    <rect x="288" y="38" width="70" height="30" fill="currentColor" fill-opacity="0.14"/><text x="323" y="57">keystream</text>
  </g>
  <path d="M88 35 L120 46" fill="none" stroke="currentColor" stroke-width="1.1"/>
  <path d="M88 69 L120 60" fill="none" stroke="currentColor" stroke-width="1.1"/>
  <path d="M240 53 L288 53" fill="none" stroke="currentColor" stroke-width="1.1"/>
  <circle cx="390" cy="53" r="11" fill="none" stroke="currentColor" stroke-width="1.1"/><text x="390" y="57" text-anchor="middle" font-size="10" fill="currentColor">⊕</text>
  <path d="M358 53 L379 53" fill="none" stroke="currentColor" stroke-width="1.1"/>
  <path d="M390 30 L390 42" fill="none" stroke="currentColor" stroke-width="1.1"/><text x="390" y="24" text-anchor="middle" font-size="7.5" fill="currentColor">cipher voice</text>
  <path d="M390 64 L390 84" fill="none" stroke="currentColor" stroke-width="1.1"/><text x="390" y="98" text-anchor="middle" font-size="7.5" fill="currentColor">clear AMBE</text>
</svg>
<figcaption>Whatever the algorithm, DMR privacy reduces to a keystream XORed onto the voice frames; recovering the clear payload needs the correct key and IV, which the standard cipher cores cannot supply on their own.</figcaption>
</figure>

## The algorithms

GopherTrunk's assessment harness realises the standard cores keyed with material an analyst
supplies. The DMRA algorithm identifiers it recognises:

| algid | Algorithm | Key size | Core |
|-------|-----------|----------|------|
| 0x21 | RC4 (Enhanced Privacy) | ~40 bits (variable) | RC4 stream cipher |
| 0x22 | DES-OFB | 8 bytes | DES in OFB |
| 0x23 | Triple-DES | 24 bytes | 3DES in OFB |
| 0x24 | AES-128 | 16 bytes | AES in OFB |
| 0x25 | AES-256 | 32 bytes | AES in OFB |

Motorola's product line also includes a lighter "Basic Privacy" — a short-key scrambler rather than
a real cipher — but the Enhanced Privacy tier and the DMRA block-cipher options are the ones with
standard cores worth implementing.

## How the keystream is built

For **RC4**, the supplied key and the frame's IV bytes are concatenated and used directly as the
RC4 key — the common Enhanced-Privacy model — and, unlike P25's ADP, no warm-up keystream bytes are
discarded before use. The RC4 core accepts a variable key length; the 40-bit size is the common
Enhanced-Privacy default used for the weak-key dictionary.

For the **block ciphers**, the scheme runs the cipher in **output-feedback (OFB)** mode: the IV
seeds OFB directly and the cipher generates a continuous keystream independent of the ciphertext,
which is exactly the additive-keystream shape the voice XOR needs. The IV is left-justified into a
full cipher block, and each algorithm requires its exact key length. In every case the resulting
keystream is XORed onto the coded voice frames — the [AMBE+2](/reference/ambe-plus-2/) payloads —
so decryption is the same XOR applied with the same keystream.

Crucially, the package implements only the cores; it does **not** fabricate a vendor key schedule.
The key and IV derivation from the air interface is the proprietary part, and it differs between
Motorola and Hytera. An analyst who has already recovered the key can decrypt, but nothing here
claims to derive that key from the signalling — the honest boundary between a standard cipher and a
reverse-engineered protocol.

## Relevance to SDR

`internal/cryptolab/engine/dmrcrypto/keystream.go` exposes `Keystream(algid, key, iv, n)`, which
dispatches to RC4 or an OFB block cipher and returns `n` keystream bytes for the caller to XOR onto
the frames, plus `KeySize`, `AlgName`, `Supported`, and a small `DefaultKeys` dictionary (all-zero,
all-ones, and an incrementing pattern) for trying weak or test keys. Keeping this in the crypto
lab — as a keystream generator keyed with analyst-supplied material, deliberately without any key
recovery — mirrors how the P25 crypto path is structured and makes the tool's capabilities and its
limits explicit.

## Sources

[^wiki]: [Digital mobile radio](https://en.wikipedia.org/wiki/Digital_mobile_radio) — Wikipedia, on DMR and its optional link-layer privacy features.
[^rc4]: [RC4](https://en.wikipedia.org/wiki/RC4) — Wikipedia, on the stream cipher underlying Motorola Enhanced Privacy.
