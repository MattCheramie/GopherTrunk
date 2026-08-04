---
slug: tetra-burst-formats
title: TETRA burst formats
entry_type: term
category: synchronization
description: TETRA burst formats are the fixed 510-bit (255-symbol) timeslot layouts — the Normal Continuous Downlink Burst, Normal Downlink Burst, and Synchronisation Burst — that place two data blocks (BKN1, BKN2) around a training sequence so a receiver can find and slice each slot.
keywords: TETRA burst, NCDB, NDB, synchronisation burst, BKN1 BKN2, TETRA timeslot, 510 bits 255 symbols, TETRA slot layout, EN 300 392-2 9.4.4
aka: [NCDB, NDB, "TETRA SB", "TETRA downlink burst"]
autolink: true
infobox:
  - { label: Slot, value: "510 bits (255 symbols)" }
  - { label: Frame, value: 4 slots / 56.67 ms }
  - { label: Blocks, value: "BKN1 + BKN2 (216 bits each)" }
  - { label: Spec, value: EN 300 392-2 §9.4.4 }
see_also: [tetra-training-sequences, tetra-receiver-chain, tetra-aach, tetra-logical-channels, frame-synchronization, tdma, pi-4-dqpsk, tetra, direct-mode-operation, tetra-dmo]
cite_urls:
  - https://en.wikipedia.org/wiki/Terrestrial_Trunked_Radio
  - https://en.wikipedia.org/wiki/Burst_transmission
---

**TETRA burst formats** are the fixed timeslot layouts that a [TETRA](/reference/tetra/) transmitter
uses to pack data, signalling, and a synchronisation pattern into each [TDMA](/reference/tdma/)
slot.[^tetra] A slot is **510 bits — 255 π/4-DQPSK symbols** — and four slots make one 56.67 ms TDMA
frame. The burst format defines exactly where a receiver will find the training sequence to lock onto
and where the two payload blocks (BKN1 and BKN2) sit relative to it, so the framer can slice each slot
without ambiguity.[^burst] The main downlink formats are the Normal Continuous Downlink Burst (NCDB),
the Normal Downlink Burst (NDB), and the Synchronisation Burst (SB).

<figure class="figure" markdown="0">
<svg viewBox="0 0 500 128" role="img" aria-label="A single TETRA downlink slot drawn as a horizontal bar: BKN1 block on the left, then the first AACH half, then the central normal training sequence, then the second AACH half, then the BKN2 block on the right, spanning 255 symbols." xmlns="http://www.w3.org/2000/svg">
  <rect x="20" y="36" width="150" height="30" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.1"/>
  <text x="95" y="55" text-anchor="middle" font-size="9" fill="currentColor">BKN1 · 216 bits</text>
  <rect x="170" y="36" width="26" height="30" fill="currentColor" fill-opacity="0.28" stroke="currentColor" stroke-width="1.1"/>
  <text x="183" y="30" text-anchor="middle" font-size="6.5" fill="currentColor">AACH</text>
  <rect x="196" y="36" width="40" height="30" fill="currentColor" fill-opacity="0.22" stroke="currentColor" stroke-width="1.1"/>
  <text x="216" y="55" text-anchor="middle" font-size="7.5" fill="currentColor">train</text>
  <rect x="236" y="36" width="30" height="30" fill="currentColor" fill-opacity="0.28" stroke="currentColor" stroke-width="1.1"/>
  <text x="251" y="30" text-anchor="middle" font-size="6.5" fill="currentColor">AACH</text>
  <rect x="266" y="36" width="150" height="30" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.1"/>
  <text x="341" y="55" text-anchor="middle" font-size="9" fill="currentColor">BKN2 · 216 bits</text>
  <path d="M20 78 L416 78" stroke="currentColor" stroke-width="1" fill="none"/>
  <path d="M20 74 L20 82 M416 74 L416 82" stroke="currentColor" stroke-width="1"/>
  <text x="218" y="94" text-anchor="middle" font-size="8" fill="currentColor">one slot = 510 bits = 255 symbols (14.167 ms)</text>
  <text x="218" y="110" text-anchor="middle" font-size="7.5" fill="currentColor">the training sequence anchors every offset; BKN1/BKN2 are sliced around it</text>
</svg>
<figcaption>A downlink slot carries two 216-bit blocks (BKN1, BKN2) split by a central training sequence, with the access-assignment channel (AACH) in the two half-blocks flanking it; the framer locates the training sequence, then slices the blocks by fixed offsets.</figcaption>
</figure>

## The downlink bursts

The **Normal Continuous Downlink Burst** carries traffic and signalling on a continuously-transmitting
downlink carrier. Around its normal [training sequence](/reference/tetra-training-sequences/) sit two
108-symbol (216-bit) blocks — **BKN1** ahead of the training sequence and **BKN2** after it — plus the
two halves of the [access-assignment channel (AACH)](/reference/tetra-aach/) that flank the training
sequence. GopherTrunk's traffic extractor measures the geometry in dibits relative to the training-sequence
lead dibit `L`: BKN1 spans `[L−115, L−7)`, the first AACH half `[L−7, L)`, the 11-dibit training sequence
`[L, L+11)`, the second AACH half `[L+11, L+19)`, and BKN2 `[L+19, L+127)`. Concatenating BKN1 and BKN2
yields one 432-bit full-slot traffic frame. The **Normal Downlink Burst** shares that block geometry; the
difference between the *continuous* and *discontinuous* downlink lies in how the carrier is keyed, not in
where the blocks fall.

## The synchronisation burst

The **Synchronisation Burst** is the one a cold receiver hunts first. Instead of two equal blocks it
carries a frequency-correction field and a broadcast synchronisation channel (BSCH) block ahead of a
longer 38-bit *synchronisation* training sequence, with a normal-length block after. The SB is transmitted
in slot 1 (TN1) of frame 18 of every multiframe, so once detected it anchors the whole slot grid: any burst
leading at dibit `L` then falls in slot `(round((L − sbAnchor)/255) mod 4) + 1`. A subtlety GopherTrunk pins
is that the SB's synchronisation training sequence sits *late* in the burst, one NDB-slot after the frame's
TN1 traffic position, so the decoded anchor must be shifted by one slot to line up with the control channel's
granted timeslots. TETRA's infrastructure-free [direct mode](/reference/direct-mode-operation/) reuses the
same physical layer but with its own block boundaries — see [TETRA DMO burst framing](/reference/tetra-dmo/).

## Relevance to SDR

`internal/radio/tetra/traffic.go` encodes the NCDB geometry as the `ndbBKN1Start`/`ndbBKN2Start` offsets and
slices each detected burst into a raw 54-byte type-5 frame; `dmo.go` carries the parallel direct-mode
geometry. Getting these offsets exactly right — and anchoring the slot grid on the SB — is what lets the
framer demultiplex four concurrent slots on one carrier and hand each block to the descrambler and channel
decoder. Every TETRA logical channel rides inside one of these bursts, so the burst format is the boundary
between raw symbols and decodable content.

## Sources

[^tetra]: [Terrestrial Trunked Radio](https://en.wikipedia.org/wiki/Terrestrial_Trunked_Radio) — Wikipedia, on the TETRA air interface and its TDMA frame structure.
[^burst]: [Burst transmission](https://en.wikipedia.org/wiki/Burst_transmission) — Wikipedia, on packing payload and synchronisation into fixed time-limited bursts.
