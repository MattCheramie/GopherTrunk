---
slug: p25-talker-alias
title: P25 talker alias (Motorola)
entry_type: term
category: trunked-radio
description: The Motorola talker alias is a human-readable name a radio broadcasts for itself — framed as an SUID (WACN, System, Radio ID) plus an obfuscated alias string and a CRC-16 — reassembled across several link-control or TSBK fragments.
keywords: P25 talker alias, Motorola talker alias, radio display name, SUID, WACN, system ID, radio ID, alias reassembly, CRC-16 GSM, talker alias fragment
aka: [talker alias, "radio alias", "unit alias"]
autolink: true
infobox:
  - { label: Purpose, value: Human-readable radio name }
  - { label: Framing, value: "SUID + alias + CRC-16" }
  - { label: SUID, value: WACN(20) + System(12) + RID(24) }
  - { label: Carriers, value: LDU1 / TDULC LC, Phase 2 FACCH-S }
see_also: [motorola-talker-alias-cipher, radio-id, wacn, system-id, p25-tsbk-vendor-opcodes, motorola-type-ii, p25-link-control-word, cyclic-redundancy-check]
cite_urls:
  - https://en.wikipedia.org/wiki/Project_25
---

A **talker alias** is a short human-readable name a radio transmits for itself — "ENGINE 12",
"DISPATCH", a unit's callsign — so a listener sees a name instead of a bare numeric
[radio ID](/reference/radio-id/).[^wiki] On Motorola P25 systems the alias is a proprietary
feature: the radio periodically broadcasts its display name, framed together with its full
source identity and a checksum, spread across several small fragments that a receiver
reassembles. GopherTrunk decodes the framing and identity; the alias text itself is protected
by a [proprietary cipher](/reference/motorola-talker-alias-cipher/) that remains unverified.

<figure class="figure" markdown="0">
<svg viewBox="0 0 470 140" role="img" aria-label="Several alias fragments concatenating into one message consisting of a source SUID of WACN, System ID, and Radio ID, followed by the obfuscated alias bytes and a trailing sixteen-bit CRC." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" stroke-width="1.1" fill="none"><rect x="20" y="18" width="60" height="22"/><rect x="84" y="18" width="60" height="22"/><rect x="148" y="18" width="60" height="22"/></g>
  <g font-size="7.5" fill="currentColor" text-anchor="middle"><text x="50" y="33">frag 0</text><text x="114" y="33">frag 1</text><text x="178" y="33">frag 2</text></g>
  <path d="M114 40 L114 56 L235 56" fill="none" stroke="currentColor" stroke-width="1" stroke-dasharray="2 2"/>
  <rect x="20" y="72" width="80" height="26" fill="currentColor" fill-opacity="0.22" stroke="currentColor" stroke-width="1.1"/>
  <text x="60" y="85" text-anchor="middle" font-size="7.5" fill="currentColor">WACN 20</text>
  <text x="60" y="94" text-anchor="middle" font-size="7.5" fill="currentColor">Sys 12</text>
  <rect x="100" y="72" width="70" height="26" fill="currentColor" fill-opacity="0.14" stroke="currentColor" stroke-width="1.1"/>
  <text x="135" y="88" text-anchor="middle" font-size="7.5" fill="currentColor">RID 24</text>
  <rect x="170" y="72" width="200" height="26" fill="none" stroke="currentColor" stroke-width="1.1"/>
  <text x="270" y="88" text-anchor="middle" font-size="7.5" fill="currentColor">obfuscated alias bytes</text>
  <rect x="370" y="72" width="70" height="26" fill="currentColor" fill-opacity="0.10" stroke="currentColor" stroke-width="1.1"/>
  <text x="405" y="88" text-anchor="middle" font-size="7.5" fill="currentColor">CRC-16</text>
  <text x="20" y="120" font-size="7.5" fill="currentColor">SUID (56 bits) identifies the radio · CRC-16/GSM validates the reassembly</text>
</svg>
<figcaption>Fragments concatenate into one message: a 56-bit SUID that names the radio, the cipher-obfuscated alias, and a trailing CRC-16 that rejects a garbled fragment sequence.</figcaption>
</figure>

## Message framing

Once its fragments are concatenated, an alias message has the same layout regardless of which
carrier delivered it (`internal/radio/p25/motorola/alias.go`):

| Bits | Field | Meaning |
|---|---|---|
| 0–19 | WACN | 20-bit [wide-area network](/reference/wacn/) ID |
| 20–31 | System ID | 12-bit [system](/reference/system-id/) ID |
| 32–55 | Radio ID | 24-bit source [unit](/reference/radio-id/) ID |
| 56 … end−16 | Alias | cipher-obfuscated display name (UTF-16 BE once decoded) |
| last 16 | CRC-16/GSM | checksum over everything before it |

The first 56 bits are the **SUID** (system unit ID) — WACN, System, and Radio ID together —
which uniquely names the radio across the network. The alias bytes that follow are run through
the [per-byte cipher](/reference/motorola-talker-alias-cipher/) to recover the name. A trailing
**CRC-16/GSM** (polynomial `0x1021`, init `0x0000`, xor-out `0xFFFF`) validates that the
fragments reassembled correctly and rejects a garbled sequence.

## Carriers and reassembly

Three on-air carriers deliver alias fragments, and all reassemble to the same framing above:
LDU1 [link control](/reference/p25-link-control-word/), TDULC link control, and the
[Phase 2](/reference/p25-tsbk-vendor-opcodes/) FACCH-S MAC PDU. On the trunking side, the
alias also rides in a [vendor TSBK](/reference/p25-tsbk-vendor-opcodes/) (opcode `0x15`, under
either the Motorola or Harris MFID), each block carrying a source ID, a block index, a block
count, and three bytes of alias data. GopherTrunk keeps the cipher, framing, and CRC in one
shared package so each carrier owns only its fragment transport and shares the decode.

## Verification and trust

The **SUID framing is verified**: the reassembly fix (issue #778) reproduces SDRTrunk's
fragment byte stream exactly, which is why the WACN, System, and Radio ID fall out correctly
on real traffic. The **alias text is not** — the obfuscation cipher is unverified and gated
off (`CipherVerified = false`, issue #773), so GopherTrunk never surfaces the decoded name as
a confirmed alias. Even the CRC is advisory: its parameters are inferred from open decoders
and not yet confirmed against a committed real-frame fixture, so GopherTrunk logs the CRC
result rather than gating on it — a wrong polynomial must not suppress a valid alias. The net
effect is that GopherTrunk reliably tells you *which radio* is announcing an alias, while being
scrupulous about not fabricating *what* the alias says.

## Sources

[^wiki]: [Project 25](https://en.wikipedia.org/wiki/Project_25) — Wikipedia, on the P25 standard. The talker alias is a proprietary Motorola feature; its framing is verified against SDRTrunk while the alias cipher remains unverified (issues #778, #773).
