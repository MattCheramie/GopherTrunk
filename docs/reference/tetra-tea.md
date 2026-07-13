---
slug: tetra-tea
title: TETRA TEA ciphers (TEA1–TEA7)
entry_type: algorithm
category: cryptography
description: "The TEA ciphers are the proprietary stream ciphers that provide air-interface encryption in TETRA radio; TEA1 was deliberately weakened, and TETRA:BURST exposed serious flaws across the family."
keywords: TETRA TEA, TEA1, TEA2, TEA3, TEA4, TEA5, TEA6, TEA7, TETRA encryption, air interface encryption, TETRA:BURST, Midnight Blue, CVE-2022-24401, CVE-2022-24402, stream cipher, keystream, ETSI, TEA1 backdoor
aka: [TEA1, TEA2, TEA3, TEA4, TETRA Encryption Algorithm]
autolink: true
infobox:
  - { label: Type, value: Proprietary stream ciphers }
  - { label: Used by, value: TETRA air interface }
  - { label: Key, value: "80-bit (TEA1 ~32-bit effective)" }
see_also: [tetra, stream-cipher, keystream, block-cipher, linear-feedback-shift-register, encryption]
cite_urls:
  - https://en.wikipedia.org/wiki/Terrestrial_Trunked_Radio
  - https://tetraburst.com/
---

**The TEA ciphers (TETRA Encryption Algorithm 1 through 7)** are the proprietary
[stream ciphers](/reference/stream-cipher/) that provide over-the-air encryption on
[TETRA](/reference/tetra/) networks, generating a [keystream](/reference/keystream/)
that is XORed with the digitized voice and data on the air interface.[^wiki] Kept
secret by ETSI for decades under strict non-disclosure, the family was designed for
different export and user tiers — and in 2023 the **TETRA:BURST** research showed
that at least one of them, TEA1, was intentionally weakened to a level that a laptop
can break.[^burst]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 108" role="img" aria-label="An 80-bit key and per-frame value seed a TEA keystream generator whose output is XORed with the TETRA voice/data bitstream; for TEA1 the effective key entropy is reduced to about 32 bits." xmlns="http://www.w3.org/2000/svg">
  <g font-size="8" fill="currentColor" text-anchor="middle">
    <rect x="20" y="30" width="70" height="24" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="55" y="45">80-bit key</text>
    <rect x="20" y="62" width="70" height="24" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="55" y="77">frame no.</text>
    <rect x="120" y="42" width="96" height="34" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.2"/><text x="168" y="56">TEA keystream</text><text x="168" y="68">generator</text>
    <line x1="90" y1="42" x2="118" y2="52" stroke="currentColor" marker-end="url(#teaar)"/>
    <line x1="90" y1="74" x2="118" y2="66" stroke="currentColor" marker-end="url(#teaar)"/>
    <line x1="216" y1="59" x2="256" y2="59" stroke="currentColor" marker-end="url(#teaar)"/>
    <circle cx="278" cy="59" r="13" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="278" y="63" font-size="11">⊕</text>
    <text x="278" y="94">voice/data</text><line x1="278" y1="90" x2="278" y2="72" stroke="currentColor"/>
    <line x1="291" y1="59" x2="360" y2="59" stroke="currentColor" marker-end="url(#teaar)"/><text x="400" y="63">on air</text>
    <text x="168" y="90" font-size="7">TEA1: ~32-bit effective</text>
  </g>
  <defs><marker id="teaar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>Each TEA cipher seeds a keystream from an 80-bit key and the frame counter and XORs it with the traffic; TEA1's effective key was deliberately shrunk to about 32 bits.</figcaption>
</figure>

## How it works

TETRA air-interface encryption is a synchronous stream cipher: the TEA algorithm
takes the shared **Cipher Key** (80 bits) together with a per-frame value derived
from the network time / frame number, and produces a keystream that both radio and
base station XOR with the traffic. Because the seed advances with the frame counter,
each frame gets fresh keystream, and a receiver that knows the key and is aligned to
network timing can regenerate the same sequence and decrypt. This is TETRA's
*air-interface* (Class 2/3) encryption, distinct from optional end-to-end encryption
that some agencies layer on top.

The four original algorithms map to markets rather than to strength: **TEA2** for
European public-safety use, **TEA1** for commercial and export use, and **TEA3/TEA4**
for other export scenarios. The internal designs were never published — security
rested on secrecy, a violation of [Kerckhoffs's principle](/reference/kerckhoffs-principle/)
that ultimately delayed rather than prevented scrutiny. A newer set, **TEA5–TEA7**,
was introduced by ETSI in 2022 as modern replacements.

## In practice — TETRA:BURST

In 2023 the Dutch firm Midnight Blue reverse-engineered the algorithms from radio
firmware and published **TETRA:BURST**, a set of five vulnerabilities.[^burst] Two
are structural and severe:

- **CVE-2022-24402 (the TEA1 backdoor):** although TEA1 accepts an 80-bit key, a
  reduction step compresses it to an effective keystrength of about **32 bits** —
  brute-forceable in minutes on a laptop. This is a deliberate weakening, consistent
  with TEA1's export role, and it means TEA1-protected traffic offers essentially no
  confidentiality against a capable attacker.
- **CVE-2022-24401 (keystream recovery):** a flaw in the air-interface
  synchronization lets an attacker recover keystream by exploiting the predictable,
  time-derived seed, enabling decryption or injection independent of key strength.

The remaining issues cover deanonymization and message manipulation. The takeaway is
that TEA2/TEA3 are not shown to be mathematically broken, but the ecosystem's
reliance on secrecy hid a purpose-built weak tier and protocol-level flaws for
decades.

## Relevance to SDR

TETRA is squarely in the family of trunked systems a scanner may follow, and TEA is
the reason its voice is usually opaque. A software-defined radio can demodulate the
π/4-DQPSK TETRA carrier, recover the burst structure, and see *that* traffic is
encrypted and which key class is negotiated, but reproducing the keystream requires
the Cipher Key — which a monitoring receiver does not hold. GopherTrunk treats TETRA
as detect-and-follow, not decode: it can identify an encrypted TETRA call and log its
metadata, and it does **not** implement the TETRA:BURST attacks. Those results are
important context for honesty about the medium — a signal being "encrypted" is not a
uniform guarantee, since TEA1 in particular was engineered to be weak — but
recovering keyed audio remains outside a passive scanner's scope and outside
GopherTrunk's feature set.

## Sources

[^wiki]: [Terrestrial Trunked Radio](https://en.wikipedia.org/wiki/Terrestrial_Trunked_Radio) — Wikipedia, for TETRA air-interface encryption, the TEA1–TEA4 market split, and the TEA5–TEA7 successors.
[^burst]: [TETRA:BURST](https://tetraburst.com/) — Midnight Blue, for the reverse-engineering of the TEA ciphers and the TEA1 effective-32-bit reduction (CVE-2022-24402) and keystream-recovery (CVE-2022-24401) findings.
