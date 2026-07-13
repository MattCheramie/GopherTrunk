---
slug: p25-des-xl
title: P25 DES-XL / ADP encryption
entry_type: algorithm
category: cryptography
description: "DES-XL and ADP are the legacy P25 encryption options — DES run as a keystream cipher in OFB mode and Motorola's RC4-based 40-bit ADP — both weak by modern standards and superseded by AES."
keywords: P25 DES, DES-XL, DES-OFB, ADP, Advanced Digital Privacy, RC4, 40-bit encryption, P25 encryption, ALGID, legacy encryption, OFB mode, keystream, brute force, Motorola, AES migration
aka: [DES-XL, DES-OFB, ADP, Advanced Digital Privacy]
autolink: true
infobox:
  - { label: Type, value: Legacy keystream ciphers }
  - { label: DES-XL, value: 56-bit DES in OFB }
  - { label: ADP, value: RC4, 40-bit key }
see_also: [data-encryption-standard, rc4-cipher, stream-cipher, project-25, advanced-encryption-standard, key-id-algid]
cite_urls:
  - https://en.wikipedia.org/wiki/Project_25
  - https://en.wikipedia.org/wiki/Data_Encryption_Standard
---

**DES-XL and ADP** are the legacy encryption options that predate — and still
coexist with — [AES](/reference/advanced-encryption-standard/) on
[P25](/reference/project-25/) systems. **DES-XL** is the
[Data Encryption Standard](/reference/data-encryption-standard/) run as a keystream
cipher (Output-Feedback mode) over the vocoder frames, while **ADP** (Advanced
Digital Privacy) is Motorola's proprietary option built on the
[RC4](/reference/rc4-cipher/) [stream cipher](/reference/stream-cipher/) with a
40-bit key.[^p25] Both are weak by modern standards — DES for its 56-bit key and ADP
for its 40-bit key — and both survive mainly on older or budget-tier deployments
where they provide obscurity against casual scanning rather than real security.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 110" role="img" aria-label="Both DES-XL and ADP feed a short key and message indicator into a keystream generator whose output is XORed with the P25 vocoder frames; DES-XL uses 56-bit DES in OFB mode and ADP uses 40-bit RC4." xmlns="http://www.w3.org/2000/svg">
  <g font-size="8" fill="currentColor" text-anchor="middle">
    <rect x="18" y="20" width="150" height="30" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="93" y="32">DES-XL: 56-bit DES</text><text x="93" y="44" font-size="7">OFB feedback → keystream</text>
    <rect x="18" y="62" width="150" height="30" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="93" y="74">ADP: 40-bit RC4</text><text x="93" y="86" font-size="7">state → keystream</text>
    <line x1="168" y1="35" x2="212" y2="52" stroke="currentColor" marker-end="url(#dxar)"/>
    <line x1="168" y1="77" x2="212" y2="60" stroke="currentColor" marker-end="url(#dxar)"/>
    <rect x="214" y="42" width="70" height="28" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.2"/><text x="249" y="59">keystream</text>
    <line x1="284" y1="56" x2="312" y2="56" stroke="currentColor" marker-end="url(#dxar)"/>
    <circle cx="334" cy="56" r="13" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="334" y="60" font-size="11">⊕</text>
    <text x="334" y="92">vocoder frames</text><line x1="334" y1="88" x2="334" y2="69" stroke="currentColor"/>
    <line x1="347" y1="56" x2="404" y2="56" stroke="currentColor" marker-end="url(#dxar)"/><text x="426" y="60">on air</text>
  </g>
  <defs><marker id="dxar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>DES-XL (56-bit DES in OFB) and ADP (40-bit RC4) both manufacture a keystream that is XORed with the P25 vocoder frames.</figcaption>
</figure>

## How it works

Both options fit P25's keystream model. The clear header advertises the algorithm
and key (see [Key ID & ALGID](/reference/key-id-algid/)) and a Message Indicator
that seeds the cipher, and the resulting keystream is XORed with the IMBE/AMBE
vocoder bitstream frame by frame:

- **DES-OFB / DES-XL** — the [DES](/reference/data-encryption-standard/)
  [block cipher](/reference/block-cipher/) is never applied to the voice directly.
  In **Output-Feedback (OFB)** mode it is run repeatedly over a feedback register
  seeded by the MI, turning the 64-bit block cipher into a keystream generator so a
  continuous voice stream can be encrypted without block padding. Standard P25
  DES-OFB carries ALGID `0x81`; Motorola's DES-XL is a closely related proprietary
  variant. DES uses a **56-bit** effective key.
- **ADP** — Motorola's Advanced Digital Privacy runs [RC4](/reference/rc4-cipher/),
  a byte-oriented [stream cipher](/reference/stream-cipher/), keyed by a **40-bit**
  user key combined with the message indicator to form RC4's initial state. It
  carries ALGID `0xAA` and is popular because it is cheap to enable on many radios.

## In practice — why they are weak

Neither cipher offers meaningful security against a determined attacker, and this is
a matter of key length, not just age. Single **DES**'s 56-bit key was broken by
public brute-force hardware in the late 1990s and is trivially recoverable today with
modern compute given known plaintext; the P25 vocoder's structured frames make such
plaintext readily available. **ADP is weaker still**: a 40-bit key is a
[brute-force](/reference/brute-force-attack/) target well within reach of a single
GPU, and RC4 additionally has well-documented keystream biases that undermine it
independent of key size. Reusing a message indicator — a two-time-pad mistake for any
keystream cipher — leaks the XOR of two plaintexts and is a further practical
pitfall. These weaknesses, plus DES's obsolescence, are exactly why P25 standardized
on **AES-256** (ALGID `0x84`) for secure voice; DES-XL and ADP persist as legacy
interoperability and low-cost "privacy" rather than as genuine confidentiality.

## Relevance to SDR

DES-XL and ADP are the encryption a scanner most often sees on older P25 fleets, and
they mark a real distinction in how far a *passive* receiver can go. GopherTrunk
reads the clear [ALGID and Key ID](/reference/key-id-algid/) and can therefore
report precisely that a talkgroup is running DES-XL or ADP rather than AES, which is
useful intelligence about a system's age and posture. It does **not**, however,
attempt key recovery: cracking a 40-bit ADP key or a 56-bit DES key is an active
cryptanalytic effort requiring captured ciphertext, known plaintext, and dedicated
compute, and it is entirely outside GopherTrunk's decode chain. The honest framing is
consistent across the project — GopherTrunk detects, identifies, and follows
encrypted P25 calls and decodes clear ones, but recovering keyed audio, even under
these weak legacy ciphers, is not something it does.

## Sources

[^p25]: [Project 25](https://en.wikipedia.org/wiki/Project_25) — Wikipedia, for the P25 encryption options including DES-OFB and AES and their algorithm identifiers. See also [Data Encryption Standard](https://en.wikipedia.org/wiki/Data_Encryption_Standard) — Wikipedia, for DES's 56-bit key and its brute-force break.
