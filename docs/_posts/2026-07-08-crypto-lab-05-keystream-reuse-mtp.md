---
title: "Crypto Lab, Part 5: Keystream Reuse — Many-Time-Pad & Crib-Dragging"
description: Crypto Lab's ks tool exploits keystream reuse — the core real-world break against additive stream ciphers like P25 DES-OFB, ADP/RC4, and TETRA AIE — detecting IV/MI collisions with ks reuse and recovering plaintext with ks mtp via known-plaintext or crib-dragging, no key recovery required.
category: tutorials
keywords: keystream reuse, many-time-pad, crib dragging, iv reuse, p25 message indicator, ofb stream cipher, adp rc4, tetra aie, known plaintext attack, gophertrunk cryptolab
tags: [cryptolab, keystream-reuse, p25, many-time-pad, advanced, security-testing]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "Crypto Lab"
series_part: 5
---

*Part 5 of **Crypto Lab**, a 10-part series on GopherTrunk's optional cryptographic-research toolkit. This is the break that actually happens in the field — and where the Mercury frames from [RF Scope]({{ '/blog/tutorials/rf-scope-07-entropy-encryption-triage/' | relative_url }}) land on the bench.*

> **TL;DR:** Additive OFB/stream ciphers (P25 DES-OFB and ADP/RC4, TETRA AIE) reuse their keystream whenever the same **key + IV** recur, so two frames sharing an IV satisfy `c1 ⊕ c2 = p1 ⊕ p2` — a running-key system you break with **no key recovery**. For P25 the IV is the 72-bit **Message Indicator (MI)**. `ks reuse` reports IV/MI collisions; `ks mtp` recovers a reuse group from a known plaintext (`-known-label`/`-known-pt`) or by crib-drag (`-crib`); `ks extract` computes a keystream from a known pt/ct pair.

> **Authorized testing only.** These frames must come from a system you own or are licensed to assess.

**Key takeaways**

- **Reused IVs are the field failure.** Same key + same IV = same keystream = `c1⊕c2 = p1⊕p2`.
- **No key recovery needed.** One known plaintext, or a good crib, unravels the whole reuse group.
- **The P25 IV is the Message Indicator** — 72 bits the decoders already extract.
- **The frames file is the bridge format** — `{label, iv, ct}` JSONL, exactly what the live decoder emits.

## Cheat sheet

| Command / flag | What it does |
|---|---|
| `cryptolab ks reuse -in frames.jsonl` | Report IV/MI collision groups |
| `cryptolab ks mtp -in frames.jsonl -known-label call-12 -known-pt known.bin` | Recover a group from one known plaintext |
| `cryptolab ks mtp -in frames.jsonl -crib " the "` | Crib-drag a group with no known plaintext |
| `cryptolab ks extract -pt p.bin -ct c.bin -out ks.bin` | Compute keystream = pt ⊕ ct |
| `-top N` | Crib-drag hits to report per group (default 10) |

## In this post

- **Why keystream reuse breaks stream ciphers** — the two-time-pad identity.
- **The P25 Message Indicator as IV** — where the collision comes from.
- **`ks reuse`** — finding the collisions.
- **`ks mtp`** — recovering content by known plaintext or crib-drag.
- **The Mercury beat** — running `ks` on the frames RF Scope handed over.

## Why reuse breaks a stream cipher

A stream cipher encrypts by XORing plaintext with a pseudorandom keystream: `c = p ⊕ ks`. The keystream is a function of the key and an initialization vector (IV). This is fine — *as long as the IV never repeats under the same key*. The moment it does, two messages share the identical keystream, and the algebra is unforgiving:

```text
c1 = p1 ⊕ ks
c2 = p2 ⊕ ks
c1 ⊕ c2 = (p1 ⊕ ks) ⊕ (p2 ⊕ ks) = p1 ⊕ p2
```

The keystream cancels. You are left with the XOR of two plaintexts, and no key anywhere in sight. This is the classic **two-time-pad** (or, across many colliding frames, **many-time-pad**) failure, and it is not theoretical — it is the single most common way real stream-cipher deployments leak, because IV management is exactly the boring operational detail that gets done wrong.

<figure class="lab-figure">
<svg viewBox="0 0 640 210" width="640" height="210" role="img" aria-label="Many-time-pad: frames sharing an IV cancel the keystream to reveal p1 XOR p2">
  <g font-size="12">
  <text x="10" y="24" fill="var(--fg-muted)">same key + same IV → same keystream</text>
  <rect x="10" y="36" width="300" height="26" rx="4" fill="none" stroke="currentColor"/>
  <text x="20" y="53" fill="currentColor">c1 = p1 ⊕ ks</text>
  <rect x="10" y="70" width="300" height="26" rx="4" fill="none" stroke="currentColor"/>
  <text x="20" y="87" fill="currentColor">c2 = p2 ⊕ ks</text>
  <rect x="10" y="104" width="300" height="26" rx="4" fill="none" stroke="currentColor"/>
  <text x="20" y="121" fill="currentColor">c3 = p3 ⊕ ks</text>
  <line x1="330" y1="83" x2="380" y2="83" stroke="currentColor" stroke-width="1.5"/>
  <polygon points="380,83 372,79 372,87" fill="currentColor"/>
  <rect x="384" y="60" width="246" height="46" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="507" y="80" text-anchor="middle" fill="var(--accent)" font-size="13">c1 ⊕ c2 = p1 ⊕ p2</text>
  <text x="507" y="98" text-anchor="middle" fill="var(--fg-muted)" font-size="11">keystream cancels — no key</text>
  <text x="10" y="160" fill="var(--fg-muted)">crib-drag a known word across the offset:</text>
  <rect x="10" y="170" width="620" height="26" rx="4" fill="none" stroke="currentColor"/>
  <text x="20" y="187" fill="currentColor" font-size="12">(p1 ⊕ p2) ⊕ crib  →  reveals partner plaintext where the crib actually sits</text>
  </g>
</svg>
<figcaption>Frames sharing an IV cancel the keystream. XORing a guessed crib against p1⊕p2 reveals the partner's plaintext wherever the crib truly appears.</figcaption>
</figure>

## The P25 Message Indicator as IV

In P25 the IV has a name: the **Message Indicator (MI)**, a 72-bit value transmitted with each encrypted superframe. The decoder already extracts it — it's how the receiver keys the OFB cipher for each call. That makes the MI the perfect collision detector. If two encrypted frames carry the same MI under the same key, they share a keystream, and `ks` finds them. The same logic applies to ADP (P25's RC4 mode), DMR's stream modes, and TETRA's AIE — any additive stream cipher whose keystream is a deterministic function of key and IV.

This is why the frames file carries an explicit `iv` field. Each line is either a JSON object or a CSV triple:

```text
{"label":"call-12","iv":"a1b2c3d4e5f60708090a","ct":"deadbeef..."}
call-12,a1b2c3d4e5f60708090a,deadbeef...
```

`label` identifies the source (a call id or timestamp), `iv` is the hex IV / MI, and `ct` is the hex ciphertext. Extra JSON fields — `system`, `protocol`, `tg`, `algid`, `keyid`, `at` — are preserved for provenance and ignored by the parser. Frames with an empty IV are skipped, and `#` lines are comments. This is **exactly the format the live decoder→cryptolab bridge emits** ([Part 9]({{ '/blog/tutorials/crypto-lab-09-web-console-bridge-external/' | relative_url }})) and what [RF Scope Part 7]({{ '/blog/tutorials/rf-scope-07-entropy-encryption-triage/' | relative_url }}) produces with `-frames-out`. The whole trilogy converges on this one JSONL shape.

## `ks reuse`: finding the collisions

```bash
gophertrunk cryptolab ks reuse -in frames.jsonl
```

```text
ks/reuse — 214 frames: 3 IV/MI reuse group(s) found
  frames  214
  reuse_groups  3
  iv=a1b2c3d4e5f60708090a   4   frames 4  labels [call-12 call-31 call-58 call-90]
  iv=00000000000000000000   7   frames 7  labels [call-03 call-19 ...]
note: reused IVs leak plaintext^plaintext — run `cryptolab ks mtp` on this
  same file to recover content from each group.
```

Each finding is a collision group: an IV value and the frames that share it. That second group — IV all zeros, seven frames — is the operational nightmare Reese warns about: an all-zero MI usually means a device that never rotates its IV at all, turning every call it makes into one giant many-time-pad. When `ks reuse` finds *no* collisions, that's the secure case, and the tool says so: every frame uses a distinct IV, so there's no keystream collision to exploit.

## `ks mtp`: recovering content

Two paths recover plaintext from a reuse group. The strongest is **known-plaintext**: if you know what one frame in the group said (a standard call-setup string, a recorded test transmission you sent yourself), you derive the keystream from it and decrypt the *entire group*:

```bash
gophertrunk cryptolab ks mtp -in frames.jsonl -known-label call-12 -known-pt known.bin
```

```text
ks/mtp — 214 frames: 3 reuse group(s)
  keystream_hex  7f3a91c2e4...
  call-31   1.0   plaintext_ascii "DISPATCH TO UNIT 214, PROCEED..."   method known-plaintext
  call-58   1.0   plaintext_ascii "UNIT 214 COPY, EN ROUTE..."         method known-plaintext
note: recovered the keystream from the known frame and decrypted its reuse
  group; the same keystream decrypts any future frame that reuses this IV.
```

The second path needs no known plaintext at all — **crib-dragging**. You slide a guessed word (`-crib`, default `" the "`) across the `p1 ⊕ p2` of each colliding pair; wherever the result is fully printable, the crib probably sits there, and it reveals the partner frame's plaintext at that offset:

```bash
gophertrunk cryptolab ks mtp -in frames.jsonl -crib "UNIT "
```

```text
  iv=a1b2c3d4... off=0   0.91   crib_in call-12  reveals_in call-31  revealed "DISPATCH TO"
note: each hit reveals plaintext in the partner frame at that offset; extend
  the highest-scoring cribs to grow the recovered text.
```

Crib-dragging is iterative: each revealed fragment becomes the next crib, and you peel the messages open a few characters at a time. A good crib is a word you're confident appears — a call sign, a unit id, a common function word. `-top` controls how many hits per group it reports. If a crib produces no fully-printable placements, the tool tells you to try a different one or supply a known plaintext.

The third mode, `ks extract`, is the plumbing between attacks: given a matched plaintext/ciphertext pair, it computes the raw keystream (`pt ⊕ ct`) and can write it to a file, so you can feed it straight into `randomness battery` or `lfsr bm` to test whether the *keystream itself* is strong or an exploitable generator:

```bash
gophertrunk cryptolab ks extract -pt known.bin -ct call-12.bin -out ks.bin
```

## The Mercury beat

Here is where the trilogy's mystery moves forward. Ada's Mercury capture — the intermittent burst near 453 MHz — went through [Signal Lab Part 8]({{ '/blog/tutorials/signal-lab-08-naming-the-unknown/' | relative_url }}) (named a candidate, no lock) and [RF Scope Part 7]({{ '/blog/tutorials/rf-scope-07-entropy-encryption-triage/' | relative_url }}), where the entropy analyzer triaged its payload as *not obviously strong* and exported the frames with `-frames-out`. Ada dropped that `mercury-frames.jsonl` straight into `ks reuse`:

```bash
gophertrunk cryptolab ks reuse -in mercury-frames.jsonl
```

```text
ks/reuse — 61 frames: 0 IV/MI reuse group(s) found
note: no IV reuse: every frame uses a distinct IV/MI... This is the secure case.
```

No reuse. Reese raised an eyebrow — *"either it's rotating IVs properly, or there's no IV at all because there's no real cipher."* That ambiguity is exactly what the `assess` battery ([Part 6]({{ '/blog/tutorials/crypto-lab-06-assess-battery/' | relative_url }})) and, ultimately, the subject framework ([Part 10]({{ '/blog/tutorials/crypto-lab-10-subject-framework-alias/' | relative_url }})) are built to resolve. Mercury doesn't reuse a keystream because Mercury, it turns out, has no keystream. Hold that thought.

## Where this goes next

`ks` attacks one specific weakness by hand. [Part 6]({{ '/blog/tutorials/crypto-lab-06-assess-battery/' | relative_url }}) is the harness that runs *every* applicable attack — IV reuse included — against a captured system and grades each on the RESISTANT/PARTIAL/BROKEN scale. That's `assess crypto`, the headline of the whole toolkit. The [cryptolab docs]({{ '/cryptolab.html' | relative_url }}) cover the frames format and the live-capture bridge in full.

## FAQ

**Why can you break this without recovering the key?**
Because when two frames share a keystream, XORing them cancels the keystream entirely, leaving `p1 ⊕ p2`. From there a known plaintext or a crib recovers the messages. The key never enters the attack.

**What is the IV in a P25 system?**
The 72-bit Message Indicator (MI), transmitted with each encrypted superframe and already extracted by the decoder. Two frames with the same MI under the same key share a keystream.

**Where do the frames come from?**
From [RF Scope Part 7]({{ '/blog/tutorials/rf-scope-07-entropy-encryption-triage/' | relative_url }})'s `-frames-out`, or the live decoder→cryptolab bridge, both of which emit the same `{label, iv, ct}` JSONL that `ks` reads.

**`ks reuse` found nothing — is the system secure?**
Against this specific attack, yes: distinct IVs mean no keystream collision. But run the full `assess` battery ([Part 6]({{ '/blog/tutorials/crypto-lab-06-assess-battery/' | relative_url }})) — a system with no reuse can still fail on default keys, a backdoored keyspace, or (like Mercury) turn out not to be real encryption at all.

## Series navigation

**Part 5 of 10** · ←[Part 4: Classical Ciphers]({{ '/blog/tutorials/crypto-lab-04-classical-ciphers-brute/' | relative_url }}) · Next →[Part 6: The `assess` Battery — Attacking P25/DMR/TETRA]({{ '/blog/tutorials/crypto-lab-06-assess-battery/' | relative_url }})
