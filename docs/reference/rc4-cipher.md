---
slug: rc4-cipher
title: RC4 (ARC4) cipher
entry_type: algorithm
category: cryptography
description: RC4 (ARC4) is a fast byte-oriented stream cipher that XORs data with a key-derived keystream; once ubiquitous in WEP, SSL, and WPA, it is now broken and appears as DMR "Enhanced Privacy."
keywords: RC4, ARC4, stream cipher, keystream, KSA, PRGA, byte-oriented, Fluhrer-Mantin-Shamir, RC4 bias, WEP, TLS, DMR Enhanced Privacy, encryption, cryptanalysis
aka: [RC4, ARC4, Rivest Cipher 4]
autolink: true
infobox:
  - { label: Type, value: Stream cipher }
  - { label: State, value: 256-byte permutation }
  - { label: Status, value: Broken / deprecated }
see_also: [stream-cipher, keystream, cryptanalysis, scrambling, dmr, forward-error-correction]
related_lessons:
  - { title: "Encryption & what you can decode", url: /learn/rf-sdr/encryption/ }
cite_urls:
  - https://en.wikipedia.org/wiki/RC4
---

**RC4** (also **ARC4**, from "Rivest Cipher 4") is a fast, byte-oriented
[stream cipher](/reference/stream-cipher/) that generates a pseudo-random
[keystream](/reference/keystream/) from a secret key and XORs it with the data.[^wiki]
Designed by Ron Rivest in 1987, it was for two decades the most widely deployed stream
cipher — in WEP, WPA/TKIP, SSL/TLS, and RDP — before accumulated biases made it insecure.
On the radio it survives as the "Enhanced Privacy" encryption option on some
[DMR](/reference/dmr/) systems.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 110" role="img" aria-label="A key generating a keystream that is XORed with plaintext to produce ciphertext." xmlns="http://www.w3.org/2000/svg">
  <rect x="30" y="44" width="50" height="26" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="55" y="61" text-anchor="middle" font-size="9" fill="currentColor">key</text>
  <line x1="80" y1="57" x2="120" y2="57" stroke="currentColor" marker-end="url(#rcar)"/>
  <rect x="122" y="44" width="80" height="26" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.2"/><text x="162" y="58" text-anchor="middle" font-size="7" fill="currentColor">KSA + PRGA</text><text x="162" y="67" text-anchor="middle" font-size="7" fill="currentColor">keystream</text>
  <circle cx="250" cy="57" r="12" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="250" y="61" text-anchor="middle" font-size="11" fill="currentColor">⊕</text>
  <text x="250" y="92" text-anchor="middle" font-size="8" fill="currentColor">plaintext</text><line x1="250" y1="88" x2="250" y2="70" stroke="currentColor"/>
  <line x1="202" y1="57" x2="238" y2="57" stroke="currentColor"/>
  <line x1="262" y1="57" x2="320" y2="57" stroke="currentColor" marker-end="url(#rcar)"/><text x="370" y="61" font-size="9" fill="currentColor">ciphertext</text>
  <defs><marker id="rcar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>RC4 is a stream cipher: a key seeds a 256-byte state that emits a keystream XORed with the data.</figcaption>
</figure>

## How it works

RC4's entire state is a 256-byte array **S** holding a permutation of the values 0–255, plus
two byte indices *i* and *j*. It runs in two phases:

- **KSA (Key-Scheduling Algorithm)** — initialize S to the identity (S[k] = k), then make 256
  passes that swap bytes of S under the control of the key, scrambling the permutation into a
  key-dependent starting state.
- **PRGA (Pseudo-Random Generation Algorithm)** — for each output byte, advance
  `i = i + 1`, then `j = j + S[i]`, swap S[i] and S[j], and output `S[(S[i] + S[j]) mod 256]`.
  All arithmetic is mod 256.

Each output byte is XORed with a plaintext byte to encrypt (and the identical keystream XORs
back to decrypt). RC4 is prized for being tiny and fast in software — no lookup tables beyond
its own state, no wide arithmetic — which is exactly why it spread so far. Only holders of the
correct key can reproduce the keystream, so unlike [scrambling](/reference/scrambling/) (a
public, keyless sequence) the voice cannot be recovered without the secret key.

## Variants — where it appeared

RC4 was a trade secret until source code was anonymously posted in 1994; the unlicensed
clone was named **ARC4** ("alleged RC4") to sidestep the trademark, and the two names are
used interchangeably. It anchored a generation of protocols: **WEP** and **WPA/TKIP** Wi-Fi
encryption, **SSL/TLS**, Microsoft's **RDP** and Office password protection, and various
proprietary radio "privacy" features. DMR "Enhanced Privacy" uses a 40-bit-keyed RC4 variant,
short enough that its main protection is obscurity rather than real key strength.

## In practice — biases and breaks

RC4 is comprehensively broken, and by more than one route:

- **Weak key schedule (Fluhrer–Mantin–Shamir, 2001).** When a per-message IV is prepended to
  a fixed key — exactly WEP's design — the first keystream bytes leak key information. FMS and
  its successors recover a WEP key from enough captured frames in minutes; this is what killed
  WEP.
- **Keystream biases.** The second output byte is biased toward zero, and many further
  position-dependent biases exist. Given the same plaintext (a fixed header, say) encrypted
  under many keys, these biases let an attacker reconstruct it — the basis for practical TLS
  attacks that led browsers to prohibit RC4 in 2015 (RFC 7465).
- **Mitigations** like discarding the first *N* keystream bytes (RC4-drop[n]) patch the KSA
  weakness but not the later biases, so no configuration is considered safe today.

These are textbook [cryptanalysis](/reference/cryptanalysis/): the cipher is not attacked by
brute force but by exploiting statistical structure the keystream should not have.

## Relevance to SDR

RC4 illustrates the line between reversible whitening and true encryption: GopherTrunk can
descramble a public whitening sequence but cannot decode encrypted voice without the key. DMR
Enhanced Privacy's short 40-bit RC4 key is weak by modern standards, but recovering it still
requires an attack GopherTrunk does not attempt — the project decodes clear and scrambled
traffic and honestly stops at keyed encryption, logging the call's metadata while the audio
stays opaque. See [DMR encryption](/dmr-encryption.html).

## Sources

[^wiki]: [RC4](https://en.wikipedia.org/wiki/RC4) — Wikipedia, for the KSA/PRGA algorithm, the ARC4 leak, deployment history, and the FMS and keystream-bias attacks.
