---
slug: p25-algorithm-id
title: P25 Algorithm ID (ALGID)
entry_type: term
category: cryptography
description: The P25 Algorithm ID (ALGID) is the one-byte code in a call's Encryption Sync that names the cipher — 0x80 clear, DES-OFB, AES-128/256, TDES, or ADP/RC4 — letting a monitor label traffic as clear or encrypted.
keywords: P25 ALGID, algorithm ID, encryption algorithm, 0x80 clear, DES-OFB, AES-256, AES-128, TDES, ADP, DES-XL, TIA-102.AACE, P25 encryption registry
aka: [ALGID, "algorithm ID"]
autolink: true
infobox:
  - { label: Field, value: 1 byte in the Encryption Sync }
  - { label: Clear value, value: "0x80 (unencrypted)" }
  - { label: Registry, value: TIA-102.AACE-A }
  - { label: Paired with, value: Key ID (KID) }
see_also: [p25-encryption, p25-des-xl, data-encryption-standard, advanced-encryption-standard, rc4-cipher, p25-encryption-sync, key-id-algid, otar, scrambling]
cite_urls:
  - https://en.wikipedia.org/wiki/Project_25
---

The **P25 Algorithm ID** (**ALGID**) is the one-byte code that names the cipher protecting a
P25 call.[^wiki] It rides in the [Encryption Sync](/reference/p25-encryption-sync/) word
alongside the [key ID](/reference/key-id-algid/) and the message indicator, and it is the
value a monitor reads first to decide whether traffic is clear or encrypted — and, if
encrypted, which [keystream generator](/reference/p25-encryption/) would be needed to decrypt
it. The value `0x80` is the reserved **CLEAR** code that unencrypted voice advertises;
anything else marks the call as protected.

<figure class="figure" markdown="0">
<svg viewBox="0 0 470 120" role="img" aria-label="A single algorithm-ID byte selecting one entry from the TIA algorithm registry, with 0x80 marking clear voice and other values naming DES, AES, TDES, or ADP ciphers." xmlns="http://www.w3.org/2000/svg">
  <rect x="30" y="30" width="90" height="30" rx="4" fill="currentColor" fill-opacity="0.22" stroke="currentColor" stroke-width="1.1"/>
  <text x="75" y="45" text-anchor="middle" font-size="9" fill="currentColor">ALGID byte</text>
  <text x="75" y="55" text-anchor="middle" font-size="7" fill="currentColor">e.g. 0x84</text>
  <path d="M120 45 L165 45" fill="none" stroke="currentColor" stroke-width="1.2" marker-end="url(#a)"/>
  <defs><marker id="a" markerWidth="7" markerHeight="7" refX="5" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 Z" fill="currentColor"/></marker></defs>
  <rect x="170" y="20" width="270" height="80" rx="4" fill="none" stroke="currentColor" stroke-width="1.1"/>
  <text x="305" y="34" text-anchor="middle" font-size="8" fill="currentColor">TIA-102.AACE-A registry</text>
  <text x="182" y="50" font-size="7.5" fill="currentColor">0x80 CLEAR · 0x81 DES-OFB · 0x84 AES-256</text>
  <text x="182" y="64" font-size="7.5" fill="currentColor">0x85 AES-128 · 0x86 TDES · 0x9F DES-XL</text>
  <text x="182" y="78" font-size="7.5" fill="currentColor">0xAA ADP/RC4 · … (unknown ⇒ mis-decode)</text>
</svg>
<figcaption>The ALGID indexes a registry of ciphers; GopherTrunk only surfaces a value that is in the registry, because an off-registry ID is provably a bit-error smear rather than a real algorithm.</figcaption>
</figure>

## The registry

GopherTrunk's `AlgorithmName` (`internal/radio/p25/algorithm.go`) covers the algorithms seen
on monitored systems; the table follows the TIA-102.AACE-A registry.

| ALGID | Algorithm | Notes |
|---|---|---|
| `0x80` | CLEAR | Unencrypted voice — the absence of encryption |
| `0x81` | DES-OFB | Single [DES](/reference/data-encryption-standard/) in OFB mode; legacy |
| `0x83` | TDES-2 | Two-key Triple-DES (K1K2K1) |
| `0x84` | AES-256 | [AES](/reference/advanced-encryption-standard/) with a 256-bit key |
| `0x85` | AES-128 | AES with a 128-bit key |
| `0x86` | TDES | Three-key Triple-DES |
| `0x89` | AES-256-OFB | AES-256 in OFB framing |
| `0x9F` | DES-XL | Motorola [DES variant](/reference/p25-des-xl/) |
| `0xAA` | ADP / [RC4](/reference/rc4-cipher/) | "Advanced Digital Privacy" — 40-bit RC4; the weak, common option |

`0x84` (AES-256) is the FIPS-grade choice for public-safety encryption; `0xAA` (ADP) is a
lightweight 40-bit RC4 variant that is cheap to enable and correspondingly weak; the DES
family is legacy. GopherTrunk also exposes `FormatAlgorithm`, which renders an ID as
`0x84 (AES-256)`, and `AlgorithmName` returns `"unknown"` for anything outside the table.

## Why unknown IDs are dropped

An ALGID is only one byte inside a heavily error-protected field, but a bit-error that
survives the [Encryption Sync](/reference/p25-encryption-sync/) FEC smears the value roughly
uniformly across `0x00`–`0xFF`, with a near-random key ID beside it. Surfaced, such a garbage
ALGID is downstream indistinguishable from a real key. GopherTrunk therefore gates on
`AlgorithmKnown`: the call path omits the algorithm and key fields entirely when the ID is not
in the registry, rather than emit a plausible-looking but fabricated algorithm. The set tracks
`AlgorithmName`, so a genuinely new algorithm is admitted the moment it is added there.

## Relevance to SDR

The ALGID drives everything GopherTrunk reports about an encrypted call: the `CLEAR`-vs-
encrypted indicator in logs and the UI, the algorithm label an operator sees, and the choice
of [keystream generator](/reference/p25-encryption/) if a key is on hand to attempt
decryption. Pairing the ALGID with the [key ID](/reference/key-id-algid/) also lets an
operator group calls by key and watch [OTAR](/reference/otar/) rekeying shift them over time.
Getting the registry right — and refusing to guess outside it — is what keeps GopherTrunk's
encryption reporting honest.

## Sources

[^wiki]: [Project 25](https://en.wikipedia.org/wiki/Project_25) — Wikipedia, on the P25 standard and its encryption. Algorithm IDs follow the TIA-102.AACE-A registry.
