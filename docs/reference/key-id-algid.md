---
slug: key-id-algid
title: Key ID & Algorithm ID (KID / ALGID)
entry_type: term
category: cryptography
description: "Key ID and ALGID are the in-the-clear P25 fields that name which stored encryption key and which cipher a secure call uses, letting a receiver identify the algorithm without ever decrypting."
keywords: Key ID, KID, ALGID, algorithm ID, P25 encryption, message indicator, MI, keyID, algorithm identifier, AES-256, DES-OFB, ADP, TIA-102, encryption sync, keyset, crypto period
aka: [KID, ALGID, Key ID, Algorithm ID]
autolink: true
infobox:
  - { label: Type, value: In-the-clear metadata }
  - { label: Key ID, value: Selects stored key (16-bit) }
  - { label: ALGID, value: Selects cipher (8-bit) }
see_also: [cryptographic-key, otar, key-loader-kfd, project-25, advanced-encryption-standard, data-encryption-standard]
cite_urls:
  - https://en.wikipedia.org/wiki/Project_25
  - https://tia.org/
---

**The Key ID (KID) and Algorithm ID (ALGID)** are the small
[P25](/reference/project-25/) header fields that tell a radio *which* stored
encryption [key](/reference/cryptographic-key/) and *which* cipher a secure
transmission uses — and, crucially, they travel in the clear even when the voice
is fully encrypted.[^wiki] The ALGID is an 8-bit number naming the algorithm (for
example `0x84` for [AES-256](/reference/advanced-encryption-standard/), `0x81` for
[DES-OFB](/reference/data-encryption-standard/), `0xAA` for the ADP stream cipher,
and `0x80` for unencrypted/clear), and the Key ID is a 16-bit label that selects
one specific key out of the many a radio may hold.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 96" role="img" aria-label="A P25 secure voice frame carries a clear header of algorithm ID, key ID, and message indicator, followed by an encrypted voice payload; a receiver reads the header but not the payload." xmlns="http://www.w3.org/2000/svg">
  <g font-size="8" fill="currentColor" text-anchor="middle">
    <rect x="20" y="34" width="52" height="30" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="46" y="52">ALGID</text>
    <rect x="72" y="34" width="52" height="30" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="98" y="52">Key ID</text>
    <rect x="124" y="34" width="72" height="30" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="160" y="49">MI</text><text x="160" y="60">(IV)</text>
    <rect x="196" y="34" width="244" height="30" fill="currentColor" fill-opacity="0.14" stroke="currentColor" stroke-width="1.2"/><text x="318" y="52">encrypted voice payload</text>
    <text x="108" y="24">clear header</text>
    <line x1="20" y1="28" x2="196" y2="28" stroke="currentColor"/>
    <text x="318" y="82">opaque without the key</text>
  </g>
</svg>
<figcaption>P25 sends the algorithm ID, key ID, and message indicator in the clear ahead of the encrypted payload, so a receiver can name the cipher without decoding the voice.</figcaption>
</figure>

## How it works

Encryption on P25 needs three things agreed between transmitter and receiver: the
algorithm, the key, and the per-transmission starting state. The ALGID and Key ID
handle the first two by *reference* rather than by value. A radio provisioned via a
[key loader](/reference/key-loader-kfd/) or [OTAR](/reference/otar/) stores keys in
numbered slots; when it transmits, it advertises the ALGID and Key ID in the frame
header so every receiving radio knows to look up the matching key in its own store
and select the matching decryption algorithm. The actual key *bits* are never sent —
only the label that names them.

The third element, the starting state, is the **Message Indicator (MI)** — the
initialization vector / keystream seed for that transmission. It too is sent in the
clear (it must be, so a late-joining receiver can synchronize the keystream) and is
carried and re-sent periodically through the call so a radio that keys up mid-
transmission can lock on. Together `{ALGID, KID, MI}` are the complete public
"envelope" of a secure P25 call: enough to reproduce the cipher's setup, useless
without the secret key the KID points to.

## Relevance to SDR

For a monitoring receiver these clear fields are the single most useful artifact of
an encrypted call. GopherTrunk cannot decrypt keyed P25 voice, but it *can* read the
ALGID and Key ID and therefore report exactly which cipher and which key slot a
talkgroup is using — distinguishing, say, AES-256 from legacy DES or ADP, and
telling apart two talkgroups that use different keys. That is genuinely actionable
metadata: it identifies encrypted-versus-clear talkgroups, exposes when a system
migrates algorithms, and (via changing Key IDs) reveals keyset rotations driven by
[OTAR](/reference/otar/). The honest boundary is firm — reading the label that
names a key is not the same as possessing the key, and no amount of header parsing
recovers the audio.

## In practice

Because the ALGID space is standardized by the TIA, published algorithm-ID tables
let tools decode the number into a human-readable cipher name; a scanner display
that shows "ENC AES-256, KID 0x2A11" is simply rendering these two clear fields. A
common field fault is an **ALGID/KID mismatch**: a radio holding the right key in
the wrong slot, or the wrong keyset after a missed rekey, hears the header, fails to
find a matching key, and mutes — audible to the user as encryption failure even
though the signal decodes cleanly. Note that clear traffic is itself just
`ALGID = 0x80`, so the same field that flags encryption also positively confirms an
unencrypted call.

## Sources

[^wiki]: [Project 25](https://en.wikipedia.org/wiki/Project_25) — Wikipedia, for the P25 encryption model in which the algorithm and key identifiers and message indicator are transmitted alongside the encrypted payload.
