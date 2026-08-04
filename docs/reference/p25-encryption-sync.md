---
slug: p25-encryption-sync
title: P25 Encryption Sync (ES)
entry_type: term
category: cryptography
description: The P25 Encryption Sync is the LDU2 field that publishes an encrypted call's crypto parameters — a 72-bit Message Indicator (IV), an algorithm ID, and a key ID — protected by RS(24,16) so a monitor can read the metadata even when it cannot decrypt.
keywords: P25 Encryption Sync, ES, message indicator, MI, IV, ALGID, KID, key ID, LDU2, RS 24 16, P25 encryption metadata, initialization vector
aka: [ES, "encryption sync", "message indicator"]
autolink: true
infobox:
  - { label: Carried in, value: LDU2 (voice) }
  - { label: Fields, value: MI 72-bit + ALGID + KID }
  - { label: FEC, value: "Hamming(10,6) + RS(24,16,9)" }
  - { label: Spec, value: TIA-102.AABF / AACE }
see_also: [p25-algorithm-id, key-id-algid, otar, p25-encryption, p25-logical-data-unit, p25-reed-solomon, advanced-encryption-standard, data-encryption-standard, scrambling]
cite_urls:
  - https://en.wikipedia.org/wiki/Project_25
---

The **P25 Encryption Sync** (**ES**) is the field an encrypted P25 voice call publishes so a
receiver holding the key can decrypt it — and so any monitor can *identify* the
encryption.[^wiki] It carries three things: the **Message Indicator** (MI), a 72-bit
per-call cryptographic sync vector that seeds the keystream generator (the
[initialization vector](/reference/p25-encryption/)); the **ALGID**
([algorithm ID](/reference/p25-algorithm-id/)) naming the cipher; and the **KID**
([key ID](/reference/key-id-algid/)) selecting which key in the radio's keyset the call uses.
Like SDRTrunk, GopherTrunk reads these to *label* a call as encrypted — algorithm and key —
without decrypting it.

<figure class="figure" markdown="0">
<svg viewBox="0 0 470 130" role="img" aria-label="The Encryption Sync content laid out as a 72-bit Message Indicator, a one-byte algorithm ID, and a two-byte key ID, wrapped by an inner Hamming code and an outer Reed-Solomon 24,16 code." xmlns="http://www.w3.org/2000/svg">
  <rect x="20" y="34" width="210" height="28" fill="currentColor" fill-opacity="0.22" stroke="currentColor" stroke-width="1.1"/>
  <text x="125" y="52" text-anchor="middle" font-size="8.5" fill="currentColor">Message Indicator · 72 bits (IV)</text>
  <rect x="230" y="34" width="80" height="28" fill="currentColor" fill-opacity="0.14" stroke="currentColor" stroke-width="1.1"/>
  <text x="270" y="52" text-anchor="middle" font-size="8.5" fill="currentColor">ALGID</text>
  <rect x="310" y="34" width="120" height="28" fill="currentColor" fill-opacity="0.10" stroke="currentColor" stroke-width="1.1"/>
  <text x="370" y="52" text-anchor="middle" font-size="8.5" fill="currentColor">KID · 16 bits</text>
  <text x="20" y="86" font-size="8" fill="currentColor">inner Hamming(10,6,3) → outer RS(24,16,9), t=4 → corrected ALGID/KID</text>
  <text x="20" y="100" font-size="7.5" fill="currentColor">uncorrectable ES ⇒ low-confidence, not surfaced</text>
</svg>
<figcaption>The ES packs the MI, ALGID, and KID, then wraps them in two FEC layers; the outer RS(24,16) is what keeps a marginal channel from producing a plausible-looking but wrong algorithm ID.</figcaption>
</figure>

## Where it rides and how it is protected

The ES travels in an **LDU2** — the second of the two voice frames a P25 call alternates —
using the same six 40-bit slots an LDU1 uses for its
[Link Control](/reference/p25-link-control-word/) word. Those slots carry an inner shortened
**Hamming(10,6,3)** code across the 24 codewords, exactly as Link Control does. Above that
sits an outer **[Reed-Solomon](/reference/p25-reed-solomon/) RS(24,16,9)** code (t = 4
symbols), which GopherTrunk's `ParseEncryptionSync` runs after the Hamming layer to correct
the residual symbol errors that would otherwise smear the algorithm ID under marginal SNR.
If more than four symbol errors survive the inner layer, the RS code fails and the parser
returns `ErrEncryptionSyncUncorrectable` — the decoded algorithm and key are then
low-confidence and are not surfaced as a real encryption change.

## The fields

`internal/radio/p25/phase1/encryption_sync.go` decodes the 12-octet ES content:

| Octets | Field | Meaning |
|---|---|---|
| 0–8 | Message Indicator | 72-bit per-call IV that seeds the keystream |
| 9 | Algorithm ID | cipher in use (`0x80` = clear/unencrypted) |
| 10–11 | Key ID | which key in the keyset the call uses |

A monitor uses the MI even when it cannot decrypt: the MI advances predictably across the
call, so it also serves as a sanity check that the crypto framing is being tracked correctly.
Because a bit-error in a traffic-channel ES smears the algorithm ID roughly uniformly across
`0x00`–`0xFF`, GopherTrunk gates surfacing on the ALGID being a *registered* value — an
out-of-set ALGID is provably a mis-decode and is dropped rather than shown as a fabricated
algorithm.

## Relevance to SDR

Reading the ES is how GopherTrunk answers the operator's first question about an encrypted
call — *what algorithm and key?* — which drives call-log fields, the encryption indicator in
the UI, and any decrypt attempt in the crypto lab. The ALGID tells whether the traffic is
[clear or protected](/reference/p25-algorithm-id/); the MI feeds the
[keystream generator](/reference/p25-encryption/) if a key is available; and the KID lets an
operator correlate calls that share a key, which is exactly the metadata
[OTAR](/reference/otar/) rekeying changes over time. The two-layer FEC is what makes this
metadata trustworthy off-air, so GopherTrunk decodes both layers before believing an ALGID.

## Sources

[^wiki]: [Project 25](https://en.wikipedia.org/wiki/Project_25) — Wikipedia, on the P25 standard and its encryption services. ES layout and FEC follow TIA-102.AABF/AACE as the project's working model.
