---
slug: p25-logical-data-unit
title: P25 Logical Data Unit
entry_type: term
category: trunked-radio
description: The P25 Logical Data Units LDU1 and LDU2 are the 1728-bit voice frames that alternate through a transmission, each carrying nine IMBE voice codewords plus Link Control (LDU1) or Encryption Sync (LDU2) and Low-Speed Data.
keywords: P25 LDU, LDU1, LDU2, logical data unit, IMBE voice frame, 1728 bits, link control, encryption sync, low-speed data, voice codeword, TIA-102 BAAA
aka: [LDU, LDU1, LDU2, "logical link data unit"]
autolink: true
infobox:
  - { label: Size, value: 1728 bits (180 ms audio) }
  - { label: Voice, value: 9 IMBE codewords × 144 bits }
  - { label: Metadata, value: LC (LDU1) or ES (LDU2) + LSD }
  - { label: Spec, value: TIA-102.BAAA §8 }
see_also: [imbe, p25-link-control-word, p25-encryption-sync, p25-status-symbols, p25-hamming-10-6, p25-nid-duid, vocoder]
related_reading:
  - { title: "SDR Internals, Part 9: Framing & forward error correction", url: /blog/deep-dives/sdr-internals-09-framing-fec/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Project_25
  - https://en.wikipedia.org/wiki/Multi-Band_Excitation
---

The **P25 Logical Data Units** — **LDU1** and **LDU2** — are the 1728-bit voice frames that carry
a P25 Phase 1 transmission.[^wiki] Each spans 180 ms of audio as nine
[IMBE](/reference/imbe/) voice codewords, and the two types alternate through a call, differing
only in the metadata woven between the voice: LDU1 carries a
[Link Control word](/reference/p25-link-control-word/) (who is talking to whom), LDU2 carries
[Encryption Sync](/reference/p25-encryption-sync/) (how to decrypt), and both carry two blocks of
Low-Speed Data. The recurring LDU1→LDU2→LDU1… sequence keeps that metadata fresh throughout the
transmission.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 140" role="img" aria-label="An LDU laid out as frame sync, NID, then nine 144-bit IMBE voice subframes u0 through u8 with six 40-bit Link Control or Encryption Sync blocks and two 16-bit Low-Speed Data blocks interleaved between them, plus 24 status symbols woven across the whole 1728-bit stream." xmlns="http://www.w3.org/2000/svg">
  <g font-size="7.5" fill="currentColor" stroke="currentColor" stroke-width="1">
    <rect x="12" y="34" width="26" height="24" fill="currentColor" fill-opacity="0.30"/><text x="25" y="49" text-anchor="middle" fill="currentColor" stroke="none">FS</text>
    <rect x="38" y="34" width="30" height="24" fill="currentColor" fill-opacity="0.30"/><text x="53" y="49" text-anchor="middle" fill="currentColor" stroke="none">NID</text>
    <rect x="68" y="34" width="34" height="24" fill="currentColor" fill-opacity="0.20"/><text x="85" y="49" text-anchor="middle" fill="currentColor" stroke="none">u0</text>
    <rect x="102" y="34" width="34" height="24" fill="currentColor" fill-opacity="0.20"/><text x="119" y="49" text-anchor="middle" fill="currentColor" stroke="none">u1</text>
    <rect x="136" y="34" width="20" height="24" fill="currentColor" fill-opacity="0.08"/><text x="146" y="49" text-anchor="middle" fill="currentColor" stroke="none">LC</text>
    <rect x="156" y="34" width="34" height="24" fill="currentColor" fill-opacity="0.20"/><text x="173" y="49" text-anchor="middle" fill="currentColor" stroke="none">u2</text>
    <rect x="190" y="34" width="20" height="24" fill="currentColor" fill-opacity="0.08"/><text x="200" y="49" text-anchor="middle" fill="currentColor" stroke="none">LC</text>
    <rect x="210" y="34" width="34" height="24" fill="currentColor" fill-opacity="0.20"/><text x="227" y="49" text-anchor="middle" fill="currentColor" stroke="none">u3</text>
    <rect x="244" y="34" width="70" height="24" fill="none" stroke-dasharray="3 2"/><text x="279" y="49" text-anchor="middle" fill="currentColor" stroke="none">u4…u6 · LC</text>
    <rect x="314" y="34" width="34" height="24" fill="currentColor" fill-opacity="0.20"/><text x="331" y="49" text-anchor="middle" fill="currentColor" stroke="none">u7</text>
    <rect x="348" y="34" width="30" height="24" fill="currentColor" fill-opacity="0.08"/><text x="363" y="49" text-anchor="middle" fill="currentColor" stroke="none">LSD</text>
    <rect x="378" y="34" width="34" height="24" fill="currentColor" fill-opacity="0.20"/><text x="395" y="49" text-anchor="middle" fill="currentColor" stroke="none">u8</text>
  </g>
  <text x="12" y="86" font-size="8" fill="currentColor">9 voice (144b) + 6 LC/ES (40b) + 2 LSD (16b) + FS + NID + 24 status = 1728 bits</text>
</svg>
<figcaption>An LDU interleaves nine 144-bit IMBE voice subframes (u0…u8) with six 40-bit LC/ES blocks and two 16-bit LSD blocks after the frame sync and NID; 24 status symbols are woven across the whole 1728-bit stream.</figcaption>
</figure>

## How it works

Stripped of its 24 interleaved [status symbols](/reference/p25-status-symbols/), the 1728-bit
LDU leaves a 1680-bit payload whose fields sit in a fixed interleaved order:

| Field | Length | Notes |
|-------|--------|-------|
| Frame Sync (FS) | 48 | Fixed sync pattern opening the LDU |
| Network ID (NID) | 64 | NAC + DUID (`0x5` LDU1, `0xA` LDU2) + BCH |
| Voice frames u0–u8 | 9 × 144 | IMBE codewords, one per 20 ms of audio |
| LC / ES blocks | 6 × 40 | Link Control (LDU1) or Encryption Sync (LDU2) |
| Low-Speed Data | 2 × 16 | Cyclic-coded out-of-band signalling |

The nine voice subframes are not contiguous — LDU1 reads u0 and u1 back-to-back, then a
40-bit LC/ES block after each of u1 through u6, with the two Low-Speed Data blocks sitting
together between u7 and u8. Each 144-bit voice subframe is an on-air IMBE codeword that the
[vocoder](/reference/vocoder/) front end deinterleaves, descrambles, and FEC-decodes into an
11-byte recorder-ready frame. The six 40-bit metadata blocks concatenate into the 240-bit LC or
ES field, which is 24 inner [Hamming(10,6,3)](/reference/p25-hamming-10-6/) codewords protecting
a 72-bit word under an outer [RS(24,12,13)](/reference/p25-reed-solomon/) (Link Control) or
RS(24,16,9) (Encryption Sync) code.

## In practice

The exact interleave order is load-bearing and easy to get wrong. An earlier GopherTrunk model
placed an LC/ES block between u0 and u1, shifting u1…u7 by one 40-bit block; it round-tripped
against the synthetic injector but did not match real on-air P25, so only u0 and u8 decoded and
every other subframe was read from the wrong bits. Getting the offsets right — sourced against a
reference decoder — is what makes a whole call decode instead of stuttering. Because the metadata
repeats every LDU pair, a decoder also stays robust to loss: miss one LDU1's Link Control and the
next one restates the talkgroup and source.

## Relevance to SDR

`internal/radio/p25/phase1/ldu.go` holds the structural constants and the extractors:
`ExtractVoiceFrames` slices u0…u8 at the sourced offsets and IMBE-decodes each,
`ExtractLCESBlocks` returns the six metadata blocks for the [Link Control](/reference/p25-link-control-word/)
or [Encryption Sync](/reference/p25-encryption-sync/) parser, and `ExtractLSDBlocks` pulls the
Low-Speed Data. A compile-time assertion pins the field arithmetic to 1728 bits. The LDU is the
heart of P25 voice — everything a scanner records and every talkgroup it reports flows through
these frames.

## Sources

[^wiki]: [Project 25](https://en.wikipedia.org/wiki/Project_25) — Wikipedia, on the P25 standard and its Phase 1 voice frames.
[^imbe]: [Multi-Band Excitation](https://en.wikipedia.org/wiki/Multi-Band_Excitation) — Wikipedia, on the IMBE vocoder whose codewords fill the LDU voice slots.
