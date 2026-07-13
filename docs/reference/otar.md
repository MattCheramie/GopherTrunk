---
slug: otar
title: Over-the-Air Rekeying (OTAR)
entry_type: term
category: cryptography
description: "OTAR is the P25 key-management protocol that distributes and updates encryption keys to radios over the RF link itself, using Key Management Messages instead of physical key loaders."
keywords: OTAR, over-the-air rekeying, over-the-air-rekeying, KMM, Key Management Message, KMF, key management facility, P25 encryption, TIA-102, rekey, keyset, crypto period, traffic encryption key, key encryption key
aka: [OTAR, over-the-air rekeying, over-the-air rekey]
autolink: true
infobox:
  - { label: Type, value: Key-management protocol }
  - { label: Carries, value: Encryption keys (KMM) }
  - { label: P25 standard, value: "TIA-102.AACA" }
see_also: [cryptographic-key, key-loader-kfd, key-id-algid, project-25, advanced-encryption-standard, data-encryption-standard]
cite_urls:
  - https://en.wikipedia.org/wiki/Over-the-air_rekeying
  - https://tia.org/
---

**Over-the-Air Rekeying (OTAR)** is the method by which a
[P25](/reference/project-25/) radio system delivers and updates its encryption
[keys](/reference/cryptographic-key/) to fielded radios over the RF channel
itself, rather than by touching each radio with a physical
[key loader](/reference/key-loader-kfd/).[^wiki] A central Key Management Facility
(KMF) sends structured Key Management Messages (KMMs) to individual radios or to
whole groups, letting an agency rotate keys, add a compromised radio to a stun
list, or push a new keyset to a fleet of thousands without recalling a single unit.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 120" role="img" aria-label="A key management facility encrypts a new traffic key under each radio's key-encryption key and sends it over the air as a KMM, which the radio unwraps and stores." xmlns="http://www.w3.org/2000/svg">
  <g font-size="8" fill="currentColor" text-anchor="middle">
    <rect x="18" y="44" width="70" height="34" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="53" y="58">KMF</text><text x="53" y="70">(key facility)</text>
    <rect x="150" y="30" width="70" height="24" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.2"/><text x="185" y="45">wrap TEK</text>
    <text x="185" y="22">under KEK</text>
    <rect x="150" y="66" width="70" height="24" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="185" y="81">KMM</text>
    <line x1="88" y1="55" x2="148" y2="45" stroke="currentColor" marker-end="url(#otarar)"/>
    <line x1="220" y1="45" x2="248" y2="72" stroke="currentColor"/><line x1="220" y1="78" x2="248" y2="78" stroke="currentColor" marker-end="url(#otarar)"/>
    <path d="M270 78 q30 -20 60 0" fill="none" stroke="currentColor" stroke-dasharray="3 3"/><text x="300" y="60">over the air</text>
    <rect x="332" y="60" width="70" height="34" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="367" y="74">radio</text><text x="367" y="86">unwraps TEK</text>
  </g>
  <defs><marker id="otarar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>OTAR wraps a new traffic key under each radio's key-encryption key and ships it as a KMM the radio unwraps and stores.</figcaption>
</figure>

## How it works

OTAR relies on a two-level key hierarchy. Each radio holds a long-lived
**Key Encryption Key (KEK)** — loaded once at provisioning time from a
[key loader](/reference/key-loader-kfd/) — and the day-to-day **Traffic
Encryption Keys (TEKs)** that actually protect voice are the ones rotated over
the air. When the KMF wants to rekey a radio, it encrypts the new TEK *under that
radio's KEK* and sends the result inside a KMM. Because only that radio (and the
KMF) knows the KEK, an eavesdropper who captures the KMM sees only ciphertext and
cannot recover the TEK.

KMMs are carried as data on the P25 system and are individually addressed by a
radio's unique ID, so the KMF can rekey one unit, a subset, or an entire group.
The protocol defines a family of message types beyond the basic rekey: warm-start
and inventory commands to learn which keys a radio holds, *changeover* commands to
switch the active keyset at a scheduled time, and *zeroize* commands that erase a
lost or stolen radio's keys remotely. Delivery is acknowledged, so the KMF knows
which radios accepted the new key and which must be retried.

A **crypto period** is the operational lifetime of a TEK; when it expires the KMF
pushes a fresh key and commands a coordinated changeover so the whole talkgroup
switches keys at once and stays in sync.

## Relevance to SDR

OTAR is the logistics layer that makes large encrypted [P25](/reference/project-25/)
systems practical: without it, rotating [AES](/reference/advanced-encryption-standard/)
or legacy [DES](/reference/data-encryption-standard/) keys across a metropolitan
fleet would mean physically touching every radio. For a monitoring receiver, OTAR
traffic is visible but opaque. GopherTrunk can see KMM data messages flow on a P25
control or data path and can log that rekeying activity is occurring, but the KMMs
are themselves encrypted under KEKs the scanner does not hold, so no key material
can be recovered from them — which is exactly the security property OTAR is
designed to provide. The practical takeaway for a listener is diagnostic, not
offensive: a burst of KMM activity often precedes a keyset changeover, after which
previously followable *clear* talkgroups may go encrypted, or the message
indicators on already-encrypted talkgroups change.

## In practice

The security of OTAR rests entirely on the secrecy of the KEK, which is why KEKs
are loaded only from a trusted [key loader](/reference/key-loader-kfd/) over a
direct wired connection and are never sent over the air in the clear. Some systems
add a still-higher **Key Encryption Key for KEKs** so that even the key-encryption
keys can be updated remotely. The identifiers that select which stored key a
message or call uses — the Key ID and algorithm ID — travel in the clear (see
[Key ID & ALGID](/reference/key-id-algid/)); only the key *values* are protected.
Mismatched keysets, expired crypto periods, or radios that missed a rekey are a
common cause of "encrypted but unintelligible even to authorized users" faults,
which is why inventory and acknowledgement messaging is a core part of the protocol.

## Sources

[^wiki]: [Over-the-air rekeying](https://en.wikipedia.org/wiki/Over-the-air_rekeying) — Wikipedia, for the KEK/TEK hierarchy, KMM/KMF roles, and remote zeroize/rekey capabilities.
