---
slug: otar-key-management
title: "OTAR & key management"
description: How encrypted trunked systems distribute and rotate keys — keyloaders and over-the-air rekeying (OTAR) from a key management facility, key IDs and algorithm IDs, crypto periods, and what a scanner can and can't see.
keywords: OTAR, over the air rekeying, encryption key management, key ID, KID, ALGID, algorithm ID, AES-256, DES-OFB, ADP, crypto period, KMF, keyloader
level: advanced
status: full
prereq:
  - encryption-and-authentication
faq:
  - q: What is OTAR?
    a: OTAR stands for over-the-air rekeying. Instead of a technician touching each radio with a keyloader, the system sends encrypted rekey messages over the air from a key management facility (KMF), so keys can be added, changed, or zeroized remotely across a whole fleet. It is standardized in P25 (TIA-102) and exists in different forms on DMR and TETRA.
  - q: Can a scanner decode an OTAR or encrypted system?
    a: No. A scanner can read the clear-text tags a call carries — the algorithm ID (ALGID) and key ID (KID) — so you can see which talkgroups are secure and which algorithm and key are in use. But without the actual key the voice payload stays unintelligible, and OTAR only moves keys between authorized radios; it never exposes them to a monitor.
  - q: What is a key ID?
    a: A key ID (KID) is a short number that names which key a call was encrypted with, without revealing the key itself. It rides in the clear alongside the algorithm ID so the receiving radio knows which stored key to apply. When a system rotates keys, the ALGID usually stays the same while the KID changes.
  - q: How often do keys change?
    a: On a schedule set by the operator called the crypto period — it might be daily, weekly, monthly, or tied to an event. Shorter periods limit how much traffic a single compromised key could expose. OTAR is what makes frequent rotation practical, since no one has to visit each radio to load the new key.
gophertrunk_links:
  - title: DMR encryption
    url: /dmr-encryption.html
    note: how GopherTrunk reports encryption status, algorithm, and key indicators on DMR.
  - title: CC Activity
    url: /cc-activity.html
    note: watch algorithm and key-ID indicators as they appear in the control traffic.
---

# OTAR & key management

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
Encryption is only as practical as its **key** distribution. A **key** is a secret
number; each call tags — in the clear — which **algorithm ID (ALGID)** and which **key
ID (KID)** it used, so the right radio can decrypt. Loading keys by hand with a keyloader
doesn't scale, so big fleets use **OTAR** (over-the-air rekeying) to distribute and rotate
keys remotely from a key management facility. Keys change on a schedule — the **crypto
period** — which limits exposure if one leaks. None of this lets a scanner hear the audio:
you can read the KIDs and ALGIDs, but never the key. See
[encryption & authentication](/learn/digital-trunking/encryption-and-authentication/).
</div>

The [previous lesson](/learn/digital-trunking/encryption-and-authentication/) covered
*how* trunked voice is encrypted and how a decoder can see that a call is protected without
being able to hear it. This one answers the question that raises: with thousands of radios
in a fleet, how do they all get the same secret key — and change it — without a technician
touching every handset? That logistics problem, key *management*, is where OTAR comes in.

## Keys, key IDs and algorithm IDs

A **key** is just a secret number, shared between the radios allowed onto a secure
talkgroup. The cipher mixes that number into the voice bits so only a radio with the
matching key can recover audio. Two labels ride in the clear on every encrypted call:

- the **algorithm ID (ALGID)** — *which* cipher: AES-256, DES-OFB, ADP (an RC4-based
  option), a proprietary mode, or a value meaning **clear** (unencrypted); and
- the **key ID (KID)** — *which* key of the several a radio may hold.

These tags exist so a receiving radio knows how to handle the call — which algorithm to run
and which stored key to apply. Because they travel unencrypted alongside the payload, a
scanner reads them too. The crucial asymmetry: the KID *names* the key, it is **not** the
key. Knowing that a call used key 3 under AES-256 tells you nothing about the secret itself.

## Loading keys: keyloaders vs OTAR

Traditionally, keys were loaded by hand. A **keyloader** — often a KVL (key variable
loader), a small handheld device — is physically touched to each radio's connector, and the
key is transferred over that wire. For a handful of radios this is fine. For a fleet of
hundreds or thousands spread across a region, it is a logistics nightmare: every rekey means
rounding up every radio, and a unit in the field or on a distant shift is easily missed.

**OTAR (over-the-air rekeying)** solves this by sending keys *over the air* instead. A
central **key management facility (KMF)** distributes and updates keys remotely, so a fleet
can be rekeyed without anyone physically handling a radio. Keyloading and OTAR often
coexist: a keyloader may plant an initial key-encryption key, after which OTAR handles the
routine traffic keys.

## How OTAR works (at a high level)

The KMF sends **rekey messages** addressed to individual radios or to groups. These
messages are themselves encrypted — a new traffic key is wrapped under a key the target
radio already holds — so the new key is never exposed on the air. Through this channel the
KMF can:

- **add** a new key to a radio,
- **change** or update an existing key, and
- **zeroize** (erase) keys remotely — useful if a radio is lost or stolen.

Radios can also acknowledge and, in some systems, request rekeys, letting the KMF track who
is current. The mechanism is standardized in **P25 (TIA-102)**, and comparable
over-the-air key management exists in **[DMR](/learn/digital-trunking/dmr-tier-2-3/)** and
**[TETRA](/learn/digital-trunking/tetra/)** in their own forms. The common thread: keys move
between authorized radios only, wrapped so that a monitor sees rekey *activity* but never
the key material.

## Crypto periods & key rotation

Keys don't live forever. Operators rotate them on a schedule called the **crypto period** —
daily, weekly, monthly, or tied to an event or shift. The reason is exposure: if a single
key were somehow compromised, a short crypto period limits how much traffic that key ever
protected. Rotating regularly also retires keys before they've encrypted enough material to
be worth attacking.

OTAR is precisely what makes frequent rotation practical — pushing a new key to a whole
fleet over the air costs the operator almost nothing, whereas hand-loading every radio each
week would be unworkable. To a monitor, a rotation is quietly visible: the **same
talkgroup** keeps running, the **same ALGID** stays in place, but the **KID changes** to the
new key. That's the signature of a fresh crypto period.

<figure class="figure" markdown="0">
<svg viewBox="0 0 540 150" role="img" aria-label="A key management facility at left sends wrapped rekey messages over the air to several radios at right; a monitor below sees only the clear key-ID and algorithm-ID tags, not the key." xmlns="http://www.w3.org/2000/svg">
  <rect x="20" y="45" width="110" height="46" rx="5" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.5"/>
  <text x="75" y="66" text-anchor="middle" font-size="10" fill="currentColor" font-weight="600">KMF</text>
  <text x="75" y="82" text-anchor="middle" font-size="8" fill="currentColor">key mgmt facility</text>
  <g stroke="currentColor" stroke-width="1.3" fill="none" stroke-dasharray="5 3">
    <line x1="130" y1="60" x2="380" y2="40"/>
    <line x1="130" y1="68" x2="380" y2="68"/>
    <line x1="130" y1="76" x2="380" y2="96"/>
  </g>
  <text x="255" y="24" text-anchor="middle" font-size="9" fill="currentColor">wrapped rekey messages (over the air)</text>
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <text x="405" y="43">radio</text>
    <text x="405" y="71">radio</text>
    <text x="405" y="99">radio</text>
  </g>
  <rect x="150" y="118" width="240" height="26" rx="5" fill="none" stroke="currentColor" stroke-width="1.5"/>
  <text x="270" y="135" text-anchor="middle" font-size="9" fill="currentColor">monitor sees: ALGID + KID (clear) — not the key</text>
</svg>
<figcaption>The KMF rekeys radios over the air with wrapped messages. A monitor can log the algorithm and key IDs the traffic advertises, but the key material itself never travels in the clear.</figcaption>
</figure>

## What a scanner can and can't see

Put plainly, a scanner can:

- **log KIDs and ALGIDs** per call and watch them over time,
- **tell encrypted from clear** — the ALGID distinguishes AES-256 or DES-OFB from a clear
  call, and
- **map which talkgroups are secure** and spot a rotation when a KID changes.

What it **cannot** do is recover audio without the key. That isn't a GopherTrunk
limitation; it's the whole point of the design — modern ciphers like AES-256 are built to
resist exactly that, and OTAR is engineered so keys reach authorized radios and no one else.

It's worth being clear about the line this lesson stays on. Understanding the *signaling* —
what an ALGID or KID means, what a rekey looks like from outside — is ordinary radio
literacy and is what a decoder like GopherTrunk reports. Defeating encryption to recover
protected audio is a different thing entirely, and it is neither the goal here nor something
GopherTrunk attempts. See
[encryption & authentication](/learn/digital-trunking/encryption-and-authentication/) for
where that wall sits.

## Per-system notes

The idea is universal; the details differ by standard.

- **P25** — key management is standardized in **TIA-102**, with a **KMF** performing OTAR:
  rekey, key updates, and zeroize, addressed to individual radios or groups. This is the
  most fully specified of the three. See
  [P25 Phase 1](/learn/digital-trunking/p25-phase-1/).
- **DMR** — ranges from **basic/enhanced privacy** (lightweight, often symmetric scrambling)
  up to **AES**, with over-the-air key management offered largely through **vendor**
  implementations rather than one universal scheme. See
  [DMR Tier 2/3](/learn/digital-trunking/dmr-tier-2-3/).
- **TETRA** — layers protection: the **TEA** air-interface encryption family secures the
  radio link, and separate **end-to-end** encryption can protect the payload across the
  whole path, each with its own key handling. See
  [TETRA](/learn/digital-trunking/tetra/).

Across all three, the takeaway for a monitor is the same: you'll see the algorithm and key
identifiers in the clear, and you won't see the keys.

<div class="knowledge-check" data-quiz data-correct-msg="Right — same talkgroup, same algorithm, new KID is the signature of a key rotation, a fresh crypto period." markdown="0">
  <p class="knowledge-check__q">Quick check: a talkgroup you watch suddenly shows a new key ID but the same ALGID. What happened?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">The system switched to a different encryption algorithm</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">The system rotated to a new key (a new crypto period)</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Your decoder recovered the key and can now hear the call</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- A **key** is a secret number; every call tags its **ALGID** (which cipher) and **KID**
  (which key) in the clear so the right radio can decrypt.
- **Keyloaders** (a KVL touched to each radio) don't scale; **OTAR** distributes and updates
  keys over the air from a **key management facility (KMF)**.
- OTAR sends **wrapped rekey messages** to radios or groups, and can add, change, or
  **zeroize** keys remotely — standardized in P25 (TIA-102), with variants in DMR and TETRA.
- Keys rotate on a **crypto period** to limit exposure; to a monitor a rotation looks like
  the **same talkgroup and ALGID with a new KID**.
- A scanner can **log KIDs/ALGIDs and tell secure from clear**, but **cannot recover audio**
  without the key — that's the design working as intended.

Next up: the other things a trunked system carries besides voice — [data services: GPS, text & registration](/learn/digital-trunking/data-services/).
