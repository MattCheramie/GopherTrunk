---
title: "Protocol Decoders, Part 10: The Alias Hunt I — Framing the Unsolved Cipher"
description: The Motorola P25 talker-alias field carries an encrypted display name. Part 1 of the capstone frames it correctly — the SUID header, the critical strip-the-CRC lesson, the verified per-byte decode model, and the recovered 256-entry LUT.
category: deep-dives
keywords: p25 talker alias, motorola alias cipher, facch-s talker alias, p25 suid framing, crc-16 gsm strip, substitution table recovery, clean room cryptanalysis, gophertrunk alias
tags: [p25, motorola, cryptanalysis, clean-room, decoding, cipher]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "Protocol Decoders"
series_part: 10
---

*Part 10 of **Protocol Decoders** — and the first half of the series capstone.
Back in
[Part 1]({{ '/blog/deep-dives/protocol-decoders-01-anatomy-of-a-cc-decoder/' | relative_url }})
we flagged one field that every other decoder in this series can parse but none
can fully read: the Motorola talker-alias, an obfuscated display name riding P25
link control. It's the same emitter the
[Crypto Lab]({{ '/blog/series/crypto-lab/' | relative_url }}) series calls
**Mercury**. This post frames it; Part 11 attacks it. The honest headline up
front: it is **not cracked**, and this is a clean-room, authorized-use-only
investigation (issue #773).*

> **TL;DR:** A Motorola talker alias reassembles to `WACN | System | RadioID |
> encoded-alias | CRC-16`. The encoded-alias field is exactly **2n bytes** for an
> n-character alias, and the **last 2 bytes are a CRC-16/GSM, not cipher output** —
> feeding them into the cryptanalysis was the mistake that stalled earlier work.
> Strip them, and a clean per-byte decode model emerges:
> `decoded[i] = int8(M·LUT[enc[i]] + c)`, with the **256-entry substitution LUT
> fully recovered**. The cipher stays gated (`CipherVerified = false`) — a
> recovered table is not a verified decode.

**Key takeaways**

- The alias framing is **verified**: a 7-byte SUID header, a `2n`-byte encoded
  alias, and a trailing CRC-16/GSM.
- The single most expensive early error was treating the **CRC bytes as
  ciphertext** — ~11% contamination per message.
- The per-byte decode is `decoded[i] = int8(M·LUT[enc[i]] + c)`; even positions
  decode to `0x00` because aliases are UTF-16BE.
- The 256-entry `LUT` is **fully recovered** (round-trips all 3,607 corpus
  aliases) — yet the cipher is still not verified end-to-end.

## Cheat sheet

| Field | Bits / bytes | Role |
|---|---|---|
| WACN | bits 0..19 | wide-area network ID |
| System ID | bits 20..31 | system ID |
| Radio ID | bits 32..55 | source subscriber (SUID = 56 bits = 7 bytes) |
| Encoded alias | bits 56..end−16 | `2n` bytes, proprietary per-byte cipher |
| CRC-16 | last 16 bits | CRC-16/GSM — **not** cipher output |

## In this post

- **Where the alias comes from** — three carriers, one reassembled message.
- **The framing** — SUID + encoded alias + CRC, and why the CRC bites.
- **The decode model** — the per-byte affine form and the recovered LUT.
- **The gate** — why a recovered table is still `CipherVerified = false`.

## Where the alias comes from

A talker alias is a radio's human-readable display name — too long for a single
message, so Motorola systems fragment it across link-control words. GopherTrunk
sees it on three different carriers, and the key design fact is that **all three
reassemble to the same message** once their fragments are concatenated:

- **LDU1 link control** — voice-channel LC on the traffic channel.
- **TDULC link control** — the terminator LC (where Motorola actually rides most
  aliases, per SDRTrunk ground truth on real systems).
- **Phase 2 FACCH-S MAC** — the TDMA fast-associated control channel.

The transport differs — `LCO 0x15` is the Motorola talker-alias header, `LCO 0x17`
the data blocks, each contributing a 44-bit fragment — but reassembly is
carriage-independent. A `TalkerAliasAssembler` buffers numbered fragments per source
unit, tolerates out-of-order arrival, evicts stale partials, and emits the complete
message when every block is in. Keeping the *cipher*, the *framing*, and the *CRC*
in one shared `internal/radio/p25/motorola` package means each carrier owns only its
fragment transport and shares the decode.

<figure class="lab-figure">
<svg viewBox="0 0 680 168" width="680" height="168" role="img" aria-label="Three P25 carriers — LDU1 link control, TDULC link control, and Phase 2 FACCH-S MAC — each fragment the same talker-alias message, which reassembles to a SUID header, encoded alias, and CRC before the cipher decode">
  <rect x="14" y="16" width="150" height="30" rx="5" fill="none" stroke="currentColor"/>
  <text x="89" y="35" text-anchor="middle" fill="currentColor" font-size="11">LDU1 link control</text>
  <rect x="14" y="60" width="150" height="30" rx="5" fill="none" stroke="currentColor"/>
  <text x="89" y="79" text-anchor="middle" fill="currentColor" font-size="11">TDULC link control</text>
  <rect x="14" y="104" width="150" height="30" rx="5" fill="none" stroke="currentColor"/>
  <text x="89" y="123" text-anchor="middle" fill="currentColor" font-size="11">Phase 2 FACCH-S</text>
  <g stroke="var(--fg-muted)">
    <line x1="164" y1="31" x2="212" y2="70"/><polygon points="210,66 220,71 210,74" fill="var(--fg-muted)"/>
    <line x1="164" y1="75" x2="212" y2="75"/><polygon points="212,71 222,75 212,79" fill="var(--fg-muted)"/>
    <line x1="164" y1="119" x2="212" y2="80"/><polygon points="210,76 220,80 210,84" fill="var(--fg-muted)"/>
  </g>
  <rect x="222" y="56" width="140" height="38" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="292" y="74" text-anchor="middle" fill="var(--accent)" font-size="11">reassemble</text>
  <text x="292" y="88" text-anchor="middle" fill="var(--fg-muted)" font-size="10">TalkerAliasAssembler</text>
  <line x1="362" y1="75" x2="398" y2="75" stroke="currentColor"/>
  <polygon points="398,71 408,75 398,79" fill="currentColor"/>
  <rect x="408" y="56" width="120" height="38" rx="6" fill="none" stroke="currentColor"/>
  <text x="468" y="74" text-anchor="middle" fill="currentColor" font-size="11">framed message</text>
  <text x="468" y="88" text-anchor="middle" fill="var(--fg-muted)" font-size="10">SUID + enc + CRC</text>
  <line x1="528" y1="75" x2="564" y2="75" stroke="currentColor"/>
  <polygon points="564,71 574,75 564,79" fill="currentColor"/>
  <rect x="574" y="56" width="92" height="38" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="620" y="74" text-anchor="middle" fill="var(--accent)" font-size="11">cipher</text>
  <text x="620" y="88" text-anchor="middle" fill="var(--fg-muted)" font-size="10">(gated)</text>
</svg>
<figcaption>Three carriers fragment the same alias; the assembler reassembles it into one framed message, and only the trailing encoded-alias field is cipher input.</figcaption>
</figure>

## The framing, and the CRC that bites

The reassembled message is exactly:

```
// internal/radio/p25/motorola/alias.go (shape)
// bits 0..19      : WACN
// bits 20..31     : System ID
// bits 32..55     : Radio (subscriber) ID   -> SUID = 56 bits = 7 bytes
// bits 56..end-16 : encoded alias bytes (proprietary per-byte cipher)
// last 16 bits    : CRC-16/GSM over the preceding bits
```

The SUID part is *verified* — the #778 reassembly fix reproduces SDRTrunk's fragment
byte stream exactly, which is why WACN / System / Radio ID fall out correctly on
real traffic. `crc16GSM` (poly `0x1021`, init `0x0000`, xorout `0xFFFF`, no
reflection) matches the CRC the reference decoder runs.

Here is the lesson that cost weeks. The encoded-alias field for an `n`-character
alias is **exactly `2n` bytes**, and the `encoded_hex` you capture is `2n + 2` bytes
— because the **last two bytes are the CRC, not cipher output.** The proof is
beautifully direct: the alias `P18` appears on two different radio IDs with
`encoded_hex` `956AB19AE437`**`D7FB`** and `956AB19AE437`**`99BA`** — *identical*
first 6 bytes (the cipher-encoded alias, which depends only on the plaintext),
differing last 2 bytes (the RID-dependent CRC).

An earlier passive analysis fed those trailing CRC bytes straight into the cipher
fit — about **11% contamination per message** — and concluded the cipher had a
"~13.6% non-deterministic high byte." Strip the last two bytes first and that
non-determinism drops to **3.2%**, and the real structure appears. The rule is now
written in the research notes in bold: *strip the trailing 2 CRC bytes before any
cryptanalysis.* It's the single most important framing fact in the whole hunt, and
it's the kind of error that looks like cipher noise when it's actually a boundary
bug.

## The per-byte decode model

With the CRC gone, the decode is a clean per-byte affine form over a fixed
substitution table:

```
decoded[i] = int8( M_i · LUT[ encoded[i] ] + c_i )
```

- `LUT` is a fixed **256-entry int8 substitution table**, and it has been **fully
  recovered** — it round-trips every byte of all 3,607 corpus aliases given the
  per-character keystream, with *zero* inconsistencies.
- `(M_i, c_i)` is a per-character keystream: `M_i` an odd multiplier (a modular
  inverse mod 256, carrying 7 bits of the accumulator's low byte), `c_i` additive
  (carrying the high byte).
- The decoded stream is **UTF-16BE + `0x00` padding**, so **even** byte positions
  decode to `0x00` for ASCII aliases. That means `M·LUT[enc] + c = 0` at even
  positions, which makes the accumulator high byte *readable directly from the
  ciphertext*: `H_k = LUT[encoded[2k]]`. That single observation — the high byte is
  in plain sight — is what turns Part 11 from a black-box search into a structural
  attack.

<figure class="lab-figure">
<svg viewBox="0 0 680 176" width="680" height="176" role="img" aria-label="Byte-field layout of a decoded talker alias: even byte positions are the UTF-16 high byte which decodes to zero and equals LUT of the even ciphertext byte, while odd positions carry the character; the trailing two bytes are the CRC and must be stripped before analysis">
  <text x="20" y="24" fill="var(--fg-muted)" font-size="11">encoded alias (2n bytes) + CRC (2 bytes)</text>
  <g font-size="10" text-anchor="middle">
    <rect x="20" y="34" width="60" height="34" fill="none" stroke="currentColor"/><text x="50" y="50" fill="currentColor">enc[0]</text><text x="50" y="63" fill="var(--fg-muted)">even</text>
    <rect x="80" y="34" width="60" height="34" fill="none" stroke="var(--accent)"/><text x="110" y="50" fill="var(--accent)">enc[1]</text><text x="110" y="63" fill="var(--fg-muted)">odd</text>
    <rect x="140" y="34" width="60" height="34" fill="none" stroke="currentColor"/><text x="170" y="50" fill="currentColor">enc[2]</text><text x="170" y="63" fill="var(--fg-muted)">even</text>
    <rect x="200" y="34" width="60" height="34" fill="none" stroke="var(--accent)"/><text x="230" y="50" fill="var(--accent)">enc[3]</text><text x="230" y="63" fill="var(--fg-muted)">odd</text>
    <rect x="260" y="34" width="80" height="34" fill="none" stroke="var(--fg-muted)" stroke-dasharray="3 2"/><text x="300" y="54" fill="var(--fg-muted)">… 2n …</text>
    <rect x="360" y="34" width="70" height="34" fill="none" stroke="#c0392b"/><text x="395" y="50" fill="#c0392b">CRC hi</text><text x="395" y="63" fill="#c0392b">strip</text>
    <rect x="430" y="34" width="70" height="34" fill="none" stroke="#c0392b"/><text x="465" y="50" fill="#c0392b">CRC lo</text><text x="465" y="63" fill="#c0392b">strip</text>
  </g>
  <text x="20" y="104" fill="var(--fg-muted)" font-size="11">decoded (UTF-16BE)</text>
  <g font-size="10" text-anchor="middle">
    <rect x="20" y="114" width="60" height="34" fill="none" stroke="currentColor"/><text x="50" y="130" fill="currentColor">0x00</text><text x="50" y="143" fill="var(--fg-muted)">hi = 0</text>
    <rect x="80" y="114" width="60" height="34" fill="none" stroke="var(--accent)"/><text x="110" y="130" fill="var(--accent)">'P'</text><text x="110" y="143" fill="var(--fg-muted)">char</text>
    <rect x="140" y="114" width="60" height="34" fill="none" stroke="currentColor"/><text x="170" y="130" fill="currentColor">0x00</text><text x="170" y="143" fill="var(--fg-muted)">hi = 0</text>
    <rect x="200" y="114" width="60" height="34" fill="none" stroke="var(--accent)"/><text x="230" y="130" fill="var(--accent)">'1'</text><text x="230" y="143" fill="var(--fg-muted)">char</text>
  </g>
  <text x="60" y="168" fill="var(--fg-muted)" font-size="10">even ciphertext byte: H_k = LUT[enc[2k]] — accumulator high byte readable directly</text>
</svg>
<figcaption>Because ASCII aliases are UTF-16BE, even decoded bytes are zero — so the even ciphertext byte reveals the accumulator high byte, and the last two bytes are CRC that must be stripped first.</figcaption>
</figure>

## The gate: recovered is not verified

Here is the discipline that separates this from a "we cracked it" blog post.
GopherTrunk *ships* a Motorola alias decoder — `DecodeAliasBytes` — but it is gated
behind a single constant:

```go
// internal/radio/p25/motorola/alias.go
const CipherVerified = false
```

While that is false, `DecodeMessage` never reports an alias as reliable, and every
caller — Phase 1 and Phase 2 — treats the output as suspect. The shipped table and
constants (`accum·293 + 0x72E9`, plus a placeholder LUT) are an **unverified
algebraic placeholder**; they decode nothing on live traffic. The rule for flipping
the gate is explicit in the code: only *together with a committed regression fixture
mapping real encoded bytes to the correct plaintext*, never on inference alone.

Why so strict? Because a wrong table can decode to *coincidentally clean ASCII* — a
plausible-looking name that is pure fiction. Surfacing a fabricated name as a
confirmed talker alias is worse than surfacing nothing: it's misinformation an
operator might act on. So the recovered `LUT` from the corpus (which genuinely
round-trips 3,607 aliases) lives in the *research toolkit*, and the shipped decoder
stays gated until a real frame confirms an end-to-end decode. This is the same
"honesty over polish" instinct that runs through the EDACS FEC docs and the
issue-closing policy — a claim isn't true until it's verified.

## Where this goes next

[Part 11]({{ '/blog/deep-dives/protocol-decoders-11-alias-hunt-cryptanalysis/' | relative_url }})
takes the fight to the cipher itself: the per-character affine keystream, the long
list of structural hypotheses that were *ruled out*, the low-byte accumulator attack
and the "observability floor" that keeps it uncracked, the chosen-plaintext capture
procedure, and the clean-room ethics that gate the whole thing. For the protocol
context, the [P25 Phase 1]({{ '/reference/p25-phase-1/' | relative_url }}) and
[P25 CAI]({{ '/reference/p25-cai/' | relative_url }}) references cover the link
control this rides on; the [Crypto Lab]({{ '/blog/series/crypto-lab/' | relative_url }})
series follows the same emitter, Mercury, from the attacker's side.

## FAQ

**What is a P25 talker alias?**
A radio's human-readable display name, sent over the air so other radios can show
"who's talking." Motorola fragments it across link-control words and obfuscates the
alias bytes with a proprietary per-byte cipher; the SUID (WACN/System/RadioID) around
it is standard and decodes cleanly.

**Why strip the last two bytes before analyzing the cipher?**
Because they're a CRC-16/GSM, not cipher output. The same alias on two different
radios shares identical cipher bytes but different CRCs. Feeding the CRC into the
cipher fit contaminates ~11% of each message and manufactures fake "non-determinism";
stripping it drops that from 13.6% to 3.2% and reveals the real structure.

**Is the talker-alias cipher decoded in GopherTrunk?**
No. The framing (SUID + CRC) is verified and decodes, but the per-byte cipher is
gated behind `CipherVerified = false`. The shipped table is an unverified placeholder
that decodes no live traffic, and the code refuses to present any alias as a
confirmed name until a real capture validates an end-to-end decode.

**If the LUT is fully recovered, why isn't it solved?**
Recovering the output substitution table is necessary but not sufficient. The
per-character keystream still comes from a nonlinear state update whose closed form
isn't known — recovering the table lets you *read* the corpus, not *decode a new
alias*. Part 11 is about that gap.

## Series navigation

**Part 10 of 12** · ←
[Part 9: Conventional, Wideband & the Symbol Scope]({{ '/blog/deep-dives/protocol-decoders-09-conventional-wideband/' | relative_url }})
· Next →
[Part 11: The Alias Hunt II]({{ '/blog/deep-dives/protocol-decoders-11-alias-hunt-cryptanalysis/' | relative_url }})
