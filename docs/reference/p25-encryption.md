---
slug: p25-encryption
title: P25 encryption keystream generators
entry_type: algorithm
category: cryptography
description: P25 link-layer encryption XORs a keystream onto the IMBE voice codewords; the keystream comes from the key and the 72-bit Message Indicator via RC4 (ADP), DES-OFB, AES-OFB, or TDES — each realisable for monitoring given the key.
keywords: P25 encryption, keystream, ADP, RC4 discard, DES-OFB, AES-128 OFB, AES-256 OFB, TDES, message indicator, IV, IMBE, P25 decryption, OFB mode
aka: [ADP, "P25 keystream", "P25 decryption"]
autolink: true
infobox:
  - { label: Construction, value: Keystream XOR onto IMBE }
  - { label: Seeded by, value: Key + 72-bit MI (IV) }
  - { label: Modes, value: RC4 / OFB block-cipher }
  - { label: ADP quirk, value: Discard first 256 RC4 bytes }
see_also: [p25-encryption-sync, p25-algorithm-id, rc4-cipher, data-encryption-standard, advanced-encryption-standard, p25-des-xl, key-id-algid, otar, scrambling]
cite_urls:
  - https://en.wikipedia.org/wiki/Project_25
  - https://en.wikipedia.org/wiki/Stream_cipher
---

**P25 link-layer encryption** protects voice by generating a **keystream** from the key and
the call's message indicator, then XORing that keystream onto the IMBE voice codewords
frame by frame.[^wiki] Every P25 cipher is used as a *stream* cipher in this sense:[^stream]
whether the primitive is RC4 or a block cipher, the output is a byte stream XORed onto the
protected bits. This means a monitor that *has the key* can decrypt purely by regenerating
the same keystream — no inversion of the cipher is needed. GopherTrunk's `p25crypto` package
realises these keystreams so its assessment harness can attempt decryption against
candidate, known, or weak keys; it performs no key recovery.

<figure class="figure" markdown="0">
<svg viewBox="0 0 470 140" role="img" aria-label="A key and a message indicator feeding a keystream generator, whose byte output is XORed against the encrypted IMBE voice codewords to recover clear voice bits." xmlns="http://www.w3.org/2000/svg">
  <rect x="20" y="24" width="90" height="24" fill="currentColor" fill-opacity="0.14" stroke="currentColor" stroke-width="1.1"/>
  <text x="65" y="40" text-anchor="middle" font-size="8" fill="currentColor">key</text>
  <rect x="20" y="58" width="90" height="24" fill="currentColor" fill-opacity="0.14" stroke="currentColor" stroke-width="1.1"/>
  <text x="65" y="74" text-anchor="middle" font-size="8" fill="currentColor">MI (72-bit IV)</text>
  <path d="M110 36 L150 50 M110 70 L150 56" fill="none" stroke="currentColor" stroke-width="1"/>
  <rect x="150" y="34" width="120" height="40" rx="4" fill="currentColor" fill-opacity="0.22" stroke="currentColor" stroke-width="1.1"/>
  <text x="210" y="50" text-anchor="middle" font-size="8" fill="currentColor">keystream</text>
  <text x="210" y="62" text-anchor="middle" font-size="8" fill="currentColor">generator</text>
  <path d="M270 54 L310 54" fill="none" stroke="currentColor" stroke-width="1.2"/>
  <circle cx="325" cy="54" r="13" fill="none" stroke="currentColor" stroke-width="1.1"/>
  <text x="325" y="58" text-anchor="middle" font-size="11" fill="currentColor">⊕</text>
  <rect x="300" y="96" width="120" height="24" fill="none" stroke="currentColor" stroke-width="1.1"/>
  <text x="360" y="112" text-anchor="middle" font-size="8" fill="currentColor">encrypted IMBE</text>
  <path d="M360 96 L325 68" fill="none" stroke="currentColor" stroke-width="1"/>
  <path d="M338 54 L400 54" fill="none" stroke="currentColor" stroke-width="1.2" marker-end="url(#x)"/>
  <defs><marker id="x" markerWidth="7" markerHeight="7" refX="5" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 Z" fill="currentColor"/></marker></defs>
  <text x="408" y="50" font-size="7.5" fill="currentColor">clear</text>
  <text x="408" y="60" font-size="7.5" fill="currentColor">voice</text>
</svg>
<figcaption>Decryption mirrors encryption: regenerate the keystream from the key and MI, then XOR it against the ciphered codewords — so possession of the key is the whole game.</figcaption>
</figure>

## The generators

Each [algorithm ID](/reference/p25-algorithm-id/) selects a keystream construction, all
seeded by the [Message Indicator](/reference/p25-encryption-sync/) (MI) as the IV. The
constructions follow the common open implementations (OP25, DSD-FME):

| ALGID | Algorithm | Key | Keystream construction |
|---|---|---|---|
| `0xAA` | ADP / [RC4](/reference/rc4-cipher/) | 5 bytes | RC4 keyed with key ‖ first 8 MI octets, **discard 256 bytes**, then XOR |
| `0x81` | DES-OFB | 8 bytes | Single-[DES](/reference/data-encryption-standard/) in OFB; IV = first 8 MI octets |
| `0x85` | [AES](/reference/advanced-encryption-standard/)-128 | 16 bytes | AES-OFB; IV = MI left-justified into 16 bytes |
| `0x84`/`0x89` | AES-256 (-OFB) | 32 bytes | AES-256-OFB; IV = MI left-justified into 16 bytes |
| `0x83` | TDES-2 | 16 bytes | Two-key 3DES expanded to K1‖K2‖K1, OFB |
| `0x86` | TDES | 24 bytes | Three-key 3DES in OFB |

For the block ciphers, **OFB** (output feedback) mode turns the block cipher into a
byte-stream generator: the IV is repeatedly re-encrypted and the output is the keystream,
independent of the plaintext. The MI supplies that IV — exactly 8 octets for DES, and
left-justified into a 16-byte block for AES (the exact TIA MI-to-IV expansion for AES is a
refinement over this simplest documented form).

## The ADP 256-byte discard

ADP is [RC4](/reference/rc4-cipher/) keyed with the 5-octet key followed by the first eight
octets of the MI, but with one non-obvious step: **the first 256 keystream bytes are thrown
away** before any encryption begins. This warm-up discard is a spec / reverse-engineering
detail — it is the OP25/DSD-FME convention and is what makes GopherTrunk's ADP keystream
line up with real ADP traffic. Getting it wrong (or omitting it) produces a keystream that
XORs to garbage even with the correct key, which is why the constant is called out
explicitly (`adpDiscard = 256`). The discard mitigates the well-known RC4 key-schedule
weakness where early output leaks key structure — though it does not rescue ADP's tiny
40-bit key from brute force.

## Relevance to SDR

`internal/cryptolab/engine/p25crypto/keystream.go` realises these generators so GopherTrunk
can *attempt* decryption when a key is supplied or under test — including a small dictionary
of weak/default keys (all-zero, all-`FF`, incrementing) that a misconfigured radio might
carry. The package deliberately stops at producing the byte-stream keystream; mapping it onto
the exact IMBE voice-bit positions is the caller's job, and no key recovery is performed. This
keeps decryption strictly a function of the [key ID](/reference/key-id-algid/) and key an
operator already holds — the honest boundary between monitoring and attacking a system.

## Sources

[^wiki]: [Project 25](https://en.wikipedia.org/wiki/Project_25) — Wikipedia, on the P25 standard and its encryption. Keystream constructions follow the open OP25 / DSD-FME implementations.
[^stream]: [Stream cipher](https://en.wikipedia.org/wiki/Stream_cipher) — Wikipedia, on XOR-keystream encryption and OFB-mode block-cipher keystreams.
