---
slug: nxdn-lich
title: NXDN LICH
entry_type: term
category: trunked-radio
description: "The NXDN Link Information Channel (LICH) is an 8-bit steering field carried with 1-to-2 bit-repetition as 16 wire bits and recovered by majority vote, telling a receiver the RF channel type, function, option, and link direction of the frame."
keywords: NXDN LICH, link information channel, RF channel type, RCCH RDCH, bit repetition, majority vote, NXDN direction, half-rate repetition
aka: [LICH, "link information channel"]
autolink: true
infobox:
  - { label: Info field, value: 8 bits }
  - { label: On air, value: 16 bits (1-to-2 repetition) }
  - { label: Recovery, value: per-bit majority vote + parity }
  - { label: Spec, value: NXDN TS 1-A §6.2.2 }
see_also: [nxdn, nxdn-frame-structure, nxdn-fsw, ran-nxdn, forward-error-correction, control-channel]
cite_urls:
  - https://en.wikipedia.org/wiki/NXDN
  - https://en.wikipedia.org/wiki/Repetition_code
---

The **NXDN Link Information Channel** (**LICH**) is an 8-bit field near the head of every
[NXDN frame](/reference/nxdn-frame-structure/) that tells the receiver how to interpret the
rest of the frame: whether it is a control or traffic channel, what function the frame serves,
and which direction it is travelling.[^wiki] Those 8 information bits are so important — a
decoder that misreads them mis-routes the whole frame — that NXDN protects them with the
simplest robust code there is: each bit is transmitted **twice** (1-to-2 repetition), making
16 wire bits, and the receiver recovers the original 8 by a per-bit majority vote.[^rep]

<figure class="figure" markdown="0">
<svg viewBox="0 0 470 160" role="img" aria-label="Eight LICH information bits each duplicated into a pair of wire bits, forming sixteen transmitted bits; on receive each pair is majority-voted back to one bit and an even-parity check over the first seven bits validates the result." xmlns="http://www.w3.org/2000/svg">
  <text x="20" y="26" font-size="9" fill="currentColor">8 info bits</text>
  <g font-size="8" fill="currentColor">
    <rect x="20" y="34" width="22" height="20" fill="currentColor" fill-opacity="0.2" stroke="currentColor" stroke-width="1"/>
    <rect x="42" y="34" width="22" height="20" fill="currentColor" fill-opacity="0.2" stroke="currentColor" stroke-width="1"/>
    <rect x="64" y="34" width="22" height="20" fill="currentColor" fill-opacity="0.2" stroke="currentColor" stroke-width="1"/>
    <rect x="86" y="34" width="130" height="20" fill="none" stroke="currentColor" stroke-width="1" stroke-dasharray="2 2"/>
    <text x="151" y="48" text-anchor="middle" font-size="7.5">… 8 bits …</text>
  </g>
  <text x="240" y="47" font-size="9" fill="currentColor">each bit → 2 wire bits</text>
  <g font-size="8" fill="currentColor">
    <rect x="20" y="82" width="14" height="20" fill="currentColor" fill-opacity="0.28" stroke="currentColor" stroke-width="1"/>
    <rect x="34" y="82" width="14" height="20" fill="currentColor" fill-opacity="0.28" stroke="currentColor" stroke-width="1"/>
    <rect x="48" y="82" width="14" height="20" fill="currentColor" fill-opacity="0.14" stroke="currentColor" stroke-width="1"/>
    <rect x="62" y="82" width="14" height="20" fill="currentColor" fill-opacity="0.14" stroke="currentColor" stroke-width="1"/>
    <rect x="76" y="82" width="160" height="20" fill="none" stroke="currentColor" stroke-width="1" stroke-dasharray="2 2"/>
    <text x="156" y="96" text-anchor="middle" font-size="7.5">16 wire bits transmitted</text>
  </g>
  <path d="M27 108 L27 122 L41 122 L41 108" fill="none" stroke="currentColor" stroke-width="1"/>
  <text x="34" y="136" text-anchor="middle" font-size="7.5" fill="currentColor">majority vote → 1 bit</text>
  <text x="300" y="96" font-size="8" fill="currentColor">then even-parity check over bits 0–6</text>
</svg>
<figcaption>The LICH doubles each of its 8 information bits into a 16-bit on-air field; the receiver majority-votes each pair back to one bit, then validates an even-parity bit computed over the first seven, flagging any pair whose two copies disagreed as soft.</figcaption>
</figure>

## What the bits mean

The 8-bit information field packs four steering values plus a parity bit, laid out
most-significant-first:

| Info bit | Field | Meaning |
| --- | --- | --- |
| 0 | RFCT — RF channel type | 0 = RCCH (control), 1 = RDCH (traffic) |
| 1–2 | FCT — function channel type | 00 = NSACCH, 01 = NUDCH, 10 = Frame Step, 11 = Reserved |
| 3–4 | Option | 2-bit service/variant selector |
| 5 | Reserved | fixed 0 |
| 6 | Direction | 0 = outbound (BS→MS), 1 = inbound (MS→BS) |
| 7 | Parity | even parity over bits 0–6 |

RFCT is the first thing a scanner reads once a frame locks: it says whether the frame is
control-channel signalling (RCCH) worth ingesting into the trunking state machine, or traffic
(RDCH) to be followed as a call. FCT then narrows the function, Option selects a service
variant, and Direction distinguishes downlink from uplink — consistent with, but independent
of, the directional [Frame Sync Word](/reference/nxdn-fsw/) that located the frame. The parity
bit is a final cheap integrity check on top of the repetition code.

## Recovery and reliability

Decoding is deliberately trivial. For each of the 8 bit positions the receiver takes the two
wire copies and, if they agree, keeps that value; if they disagree, it flags the pair as
unreliable (a disagreement means at least one of the two copies took a bit error) and falls
back to the first copy. The count of disagreeing pairs is a direct, per-frame quality metric:
zero means a clean LICH, and a rising count is an early warning that the channel is
degrading before the heavier-coded fields start failing. After the vote, the even-parity bit
over bits 0–6 gives one more line of defence — it catches a fraction of the cases where both
copies of a bit flipped the same way, which majority vote alone cannot.

Because the LICH is short and cheaply coded, it is one of the most reliably recovered fields
on an NXDN channel, which is exactly why NXDN puts the frame's routing decision there rather
than deeper in the more fragile [CAC](/reference/nxdn-cac/) or SACCH.

## Relevance to SDR

`internal/radio/nxdn/lich.go` implements the whole path: `EncodeLICHWire` doubles the 8 info
bits to 16, `DecodeLICHWire` majority-votes them back and returns the count of disagreeing
pairs, and `ParseLICH` unpacks the 8-bit field into typed `RFChannelType`,
`FunctionChannelType`, `Option`, and `Direction` values with a validated even-parity flag. The
disagreement count and the parity flag together give GopherTrunk a soft-quality read on every
frame at almost no cost — the LICH is the first checkpoint that tells the decoder whether a
freshly-synced frame is a control block to route or traffic to follow, and how much to trust
that decision.

## Sources

[^wiki]: [NXDN](https://en.wikipedia.org/wiki/NXDN) — Wikipedia, on the NXDN standard and its logical channels.
[^rep]: [Repetition code](https://en.wikipedia.org/wiki/Repetition_code) — Wikipedia, on the bit-repetition-plus-majority-vote coding the LICH uses.
