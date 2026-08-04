---
slug: p25-link-control-word
title: P25 Link Control Word
entry_type: term
category: trunked-radio
description: The P25 Link Control Word is the 72-bit message inside an LDU1 or TDULC whose Link Control Opcode names its format — most often Group Voice Channel User, carrying talkgroup and source — protected by an outer RS(24,12,13) code.
keywords: P25 link control word, LCW, LCO, LCF, link control opcode, group voice channel user, talkgroup source, talker alias, RS 24 12, TIA-102 AABF
aka: [LCW, "link control word", LCO, "link control opcode"]
autolink: true
infobox:
  - { label: Size, value: 72 bits (9 octets) }
  - { label: Opcode, value: LCF / LCO in octet 0 }
  - { label: FEC, value: "Hamming(10,6,3) inner + RS(24,12,13)" }
  - { label: Spec, value: TIA-102.AABF }
see_also: [p25-logical-data-unit, p25-terminator-data-unit, p25-reed-solomon, p25-hamming-10-6, talkgroup, radio-id, channel-grant]
related_reading:
  - { title: "SDR Internals, Part 9: Framing & forward error correction", url: /blog/deep-dives/sdr-internals-09-framing-fec/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Project_25
  - https://en.wikipedia.org/wiki/Reed%E2%80%93Solomon_error_correction
---

The **P25 Link Control Word** (**LCW**) is the 72-bit message carried in an
[LDU1](/reference/p25-logical-data-unit/) voice frame — and, in terminator form, a
[TDULC](/reference/p25-terminator-data-unit/) — that says who is talking on the voice channel and
in what mode.[^wiki] Its first octet is the **Link Control Opcode** (LCO, also written LCF for
Link Control Format), which names the word's layout; the most common,
**Group Voice Channel User** (`0x00`), carries the destination [talkgroup](/reference/talkgroup/)
and the source [radio ID](/reference/radio-id/). The 72-bit word is protected by an inner
[Hamming(10,6,3)](/reference/p25-hamming-10-6/) layer under an outer
[RS(24,12,13)](/reference/p25-reed-solomon/) code.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 130" role="img" aria-label="The 72-bit Group Voice Channel User link control word as nine octets: link control format, manufacturer ID, service options, a reserved octet, a 16-bit talkgroup across octets four and five, and a 24-bit source ID across octets six through eight." xmlns="http://www.w3.org/2000/svg">
  <g font-size="7.5" fill="currentColor" stroke="currentColor" stroke-width="1.1">
    <rect x="14" y="34" width="46" height="26" fill="currentColor" fill-opacity="0.30"/><text x="37" y="50" text-anchor="middle" fill="currentColor" stroke="none">LCF</text>
    <rect x="60" y="34" width="42" height="26" fill="currentColor" fill-opacity="0.14"/><text x="81" y="50" text-anchor="middle" fill="currentColor" stroke="none">MFID</text>
    <rect x="102" y="34" width="44" height="26" fill="currentColor" fill-opacity="0.14"/><text x="124" y="47" text-anchor="middle" fill="currentColor" stroke="none">svc</text>
    <rect x="146" y="34" width="40" height="26" fill="currentColor" fill-opacity="0.08"/><text x="166" y="47" text-anchor="middle" fill="currentColor" stroke="none">rsvd</text>
    <rect x="186" y="34" width="84" height="26" fill="currentColor" fill-opacity="0.22"/><text x="228" y="50" text-anchor="middle" fill="currentColor" stroke="none">Talkgroup 16</text>
    <rect x="270" y="34" width="118" height="26" fill="currentColor" fill-opacity="0.22"/><text x="329" y="50" text-anchor="middle" fill="currentColor" stroke="none">Source ID 24</text>
  </g>
  <text x="14" y="82" font-size="8" fill="currentColor">octet 0 opcode selects the layout · LCO 0x00 = Group Voice Channel User (TG + source)</text>
</svg>
<figcaption>For the Group Voice Channel User opcode, the nine octets are LCF, manufacturer ID, service options, a reserved octet, a 16-bit talkgroup, and a 24-bit source ID; other opcodes reinterpret the same nine octets.</figcaption>
</figure>

## How it works

The LCO in octet 0 is the discriminator: it tells the parser how to read the remaining eight
octets. GopherTrunk models the Group Voice Channel User layout directly and recognises the
standard talker-alias opcodes that carry a radio's display name across an active call as a
header-plus-fragments sequence:

| LCO | Name | Content |
|-----|------|---------|
| `0x00` | Group Voice Channel User | MFID, service options, talkgroup (octets 4–5), source ID (octets 6–8) |
| `0x15` | Talker Alias — Header | Character set + total alias length |
| `0x16` | Talker Alias — Block 1 | First alias-data fragment |
| `0x17` | Talker Alias — Block 2 | Second alias-data fragment |

Many more LCOs are defined in TIA-102.AABF (unit-to-unit users, various broadcast and status
words); GopherTrunk decodes the ones above and exposes the raw nine octets for any other opcode a
caller wants to interpret. Whatever the opcode, the FEC is identical: the 72 content bits are the
first 12 six-bit symbols of an RS(24,12,13) codeword, each symbol wrapped in a Hamming(10,6,3)
inner codeword. On receive, the inner Hamming pass fixes single-bit hits, then the outer RS pass
corrects up to six residual **symbol** errors before the octets are read.

## In practice

The outer RS layer is not optional decoration. Residual bit errors that survive the single-error
Hamming layer will corrupt the talkgroup or source under marginal SNR, and because the voice
composer gates a call on the on-air talkgroup matching the granted one, a corrupted LCW ends the
call early and fragments one transmission into many tiny recordings. Running RS(24,12,13) after
the Hamming layer is what holds a marginal call together. The octet layout is equally
unforgiving: an earlier GopherTrunk model placed the talkgroup at octets 2–3 and the source at
4–6, which made the talkgroup decode to the constant service-options byte and the real talkgroup
land inside the misread source — the same early-termination symptom, from a one-octet shift.

## Relevance to SDR

`internal/radio/p25/phase1/link_control.go` implements `ParseLinkControl` (the LCO-0 struct with
talkgroup and source), `ParseLinkControlContent` (raw octets for other opcodes), and
`AssembleLinkControl` (the symmetric encoder that computes RS parity so synthetic frames carry
valid FEC). The same 72-bit word is reached two ways — from LDU1 via
[ExtractLCESBlocks](/reference/p25-logical-data-unit/) and from a
[TDULC](/reference/p25-terminator-data-unit/) terminator — so the LCW is the single most important
metadata unit on a P25 voice channel: it is what turns decoded audio into a labelled call with a
talkgroup and a source radio.

## Sources

[^wiki]: [Project 25](https://en.wikipedia.org/wiki/Project_25) — Wikipedia, on the P25 standard and its Phase 1 link-control signalling.
[^rs]: [Reed-Solomon error correction](https://en.wikipedia.org/wiki/Reed%E2%80%93Solomon_error_correction) — Wikipedia, on the RS(24,12,13) outer code protecting the Link Control Word.
