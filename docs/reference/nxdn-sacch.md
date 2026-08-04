---
slug: nxdn-sacch
title: NXDN SACCH
entry_type: term
category: trunked-radio
description: "The NXDN Slow Associated Control Channel (SACCH) carries per-frame signalling as a 60-bit block coded from 32 information bits — 26 payload plus a 6-bit CRC — via K=5 half-rate convolution, 12-position puncturing, and a 60-bit interleave, with fragments reassembled across a superframe."
keywords: NXDN SACCH, slow associated control channel, 32 to 60 bits, CRC-6, K=5 convolution, puncture, interleave, SACCH fragments, superframe reassembly
aka: [SACCH, "slow associated control channel"]
autolink: true
infobox:
  - { label: Info block, value: "32 bits (26 payload + CRC-6)" }
  - { label: On air, value: 60 bits per frame }
  - { label: FEC, value: "K=5 R=1/2 conv + puncture + interleave" }
  - { label: Spec, value: NXDN TS 1-A §6.6 }
see_also: [nxdn, nxdn-frame-structure, nxdn-cac, convolutional-code, puncturing, interleaving, viterbi-algorithm, cyclic-redundancy-check, control-channel]
cite_urls:
  - https://en.wikipedia.org/wiki/NXDN
  - https://en.wikipedia.org/wiki/Convolutional_code
---

The **NXDN Slow Associated Control Channel** (**SACCH**) is the low-rate signalling channel
that rides alongside every [NXDN frame](/reference/nxdn-frame-structure/), carrying the
per-frame housekeeping — the [RAN](/reference/ran-nxdn/), call state, and other control
fields — underneath live traffic without stealing the frame's payload.[^wiki] Each frame
transmits a 60-bit SACCH block coded from just 32 information bits (26 payload bits plus a
6-bit CRC), so one frame alone is too small to hold a full control message; the receiver
accumulates SACCH fragments across a superframe and reassembles the complete message.[^conv]

<figure class="figure" markdown="0">
<svg viewBox="0 0 480 155" role="img" aria-label="The NXDN SACCH coding chain: a 32-bit information block of 26 payload bits plus a 6-bit CRC gains 4 tail bits to make 36 bits, is convolutionally encoded at half rate to 72 bits, punctured to 60 by dropping 12 positions, and interleaved to 60 on-air bits; below, four short per-frame SACCH fragments accumulate into one reassembled message." xmlns="http://www.w3.org/2000/svg">
  <g font-size="7.5" fill="currentColor">
    <rect x="12" y="30" width="70" height="28" fill="currentColor" fill-opacity="0.2" stroke="currentColor" stroke-width="1"/>
    <text x="47" y="43" text-anchor="middle">26 + CRC-6</text>
    <text x="47" y="53" text-anchor="middle">= 32 bits</text>
    <path d="M82 44 L100 44" stroke="currentColor" stroke-width="1" marker-end="url(#s)"/>
    <rect x="100" y="30" width="60" height="28" fill="currentColor" fill-opacity="0.14" stroke="currentColor" stroke-width="1"/>
    <text x="130" y="43" text-anchor="middle">+tail → 36</text>
    <text x="130" y="53" text-anchor="middle">conv → 72</text>
    <path d="M160 44 L178 44" stroke="currentColor" stroke-width="1" marker-end="url(#s)"/>
    <rect x="178" y="30" width="60" height="28" fill="currentColor" fill-opacity="0.14" stroke="currentColor" stroke-width="1"/>
    <text x="208" y="43" text-anchor="middle">puncture</text>
    <text x="208" y="53" text-anchor="middle">drop 12 → 60</text>
    <path d="M238 44 L256 44" stroke="currentColor" stroke-width="1" marker-end="url(#s)"/>
    <rect x="256" y="30" width="60" height="28" fill="currentColor" fill-opacity="0.14" stroke="currentColor" stroke-width="1"/>
    <text x="286" y="43" text-anchor="middle">interleave</text>
    <text x="286" y="53" text-anchor="middle">60 bits</text>
    <path d="M316 44 L334 44" stroke="currentColor" stroke-width="1" marker-end="url(#s)"/>
    <rect x="334" y="30" width="66" height="28" fill="currentColor" fill-opacity="0.22" stroke="currentColor" stroke-width="1"/>
    <text x="367" y="47" text-anchor="middle">on air / frame</text>
  </g>
  <g font-size="7.5" fill="currentColor">
    <rect x="12" y="92" width="40" height="20" fill="currentColor" fill-opacity="0.18" stroke="currentColor" stroke-width="1"/>
    <rect x="56" y="92" width="40" height="20" fill="currentColor" fill-opacity="0.18" stroke="currentColor" stroke-width="1"/>
    <rect x="100" y="92" width="40" height="20" fill="currentColor" fill-opacity="0.18" stroke="currentColor" stroke-width="1"/>
    <rect x="144" y="92" width="40" height="20" fill="currentColor" fill-opacity="0.18" stroke="currentColor" stroke-width="1"/>
    <text x="98" y="106" text-anchor="middle" font-size="6.5">fragment per frame</text>
    <path d="M188 102 L214 102" stroke="currentColor" stroke-width="1" marker-end="url(#s)"/>
    <rect x="214" y="92" width="130" height="20" fill="none" stroke="currentColor" stroke-width="1"/>
    <text x="279" y="106" text-anchor="middle">reassembled message (superframe)</text>
  </g>
  <defs><marker id="s" markerWidth="6" markerHeight="6" refX="5" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>Each frame's SACCH codes a 32-bit block (26 payload + 6-bit CRC) into 60 on-air bits through tail-padding, half-rate convolution, puncturing, and interleaving; because one fragment is small, the receiver accumulates fragments across a superframe to rebuild the full control message.</figcaption>
</figure>

## The coding chain

The 32-bit information block is 26 payload bits followed by a 6-bit CRC computed over those 26
(NXDN's CRC-6 uses the polynomial g(x) = x⁶ + x + 1). Four zero tail bits are appended so the
encoder ends in state 0, giving 36 input bits. Those pass through a constraint-length-5,
rate-½ convolutional code — generators g1 = `1+D³+D⁴` (octal 31) and g2 = `1+D+D²+D⁴`
(octal 27), the same code the [CAC](/reference/nxdn-cac/) uses — producing 72 channel bits.
Twelve fixed positions are then [punctured](/reference/puncturing/) away (the drop list is
evenly spaced, every sixth bit starting at position 5), leaving 60, and a 60-position
[interleaver](/reference/interleaving/) permutes them so a burst on the channel scatters into
isolated errors. The output is the 60 bits transmitted in the frame's SACCH slot.

The receiver inverts this exactly: deinterleave, depuncture (inserting a zero-cost sentinel at
the 12 dropped positions), [Viterbi](/reference/viterbi-algorithm/) decode over 36 stages with
the end-state constrained to 0, strip the 4 tail bits, and verify the CRC-6 over the recovered
26-bit payload. The decoder also returns the Viterbi path metric — zero means a clean decode —
so the caller has a soft-quality read alongside the hard CRC pass/fail.

## Fragment reassembly

A 26-bit payload cannot hold a whole control message, so NXDN treats the SACCH as a
low-throughput pipe: successive frames each carry one fragment of a longer message, and the
receiver stitches the fragments back together across a superframe before acting on the result.
This is the deliberate trade the "slow" in Slow Associated Control Channel names — it accepts
a superframe of latency in exchange for a steady signalling channel that runs continuously
beneath voice or data without ever pre-empting the frame's information field. A message is only
trusted once every fragment has arrived and the reassembled block passes CRC.

## Relevance to SDR

`internal/radio/nxdn/sacch.go` implements the chain end to end: `EncodeSACCH` builds a 60-bit
block from 32 info bits, `DecodeSACCH` runs the deinterleave / depuncture / Viterbi / CRC-6
inverse and returns the decoded bits plus the path metric, and `SACCHCRC6` /
`VerifySACCHCRC6` handle the 6-bit trailer. The interleave permutation and puncture list come
from cross-referenced public NXDN decoders and are pinned in the source. Because the SACCH is
where the RAN and call state live, decoding it reliably is what lets GopherTrunk tell one
co-channel system from another and track call progress frame by frame — the CRC-6 pass is the
gate that keeps a corrupted fragment from poisoning the reassembled message.

## Sources

[^wiki]: [NXDN](https://en.wikipedia.org/wiki/NXDN) — Wikipedia, on the NXDN standard and its associated control channels.
[^conv]: [Convolutional code](https://en.wikipedia.org/wiki/Convolutional_code) — Wikipedia, on the encoder and Viterbi decoding the SACCH uses.
