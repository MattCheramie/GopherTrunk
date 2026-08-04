---
slug: p25-isch
title: P25 ISCH
entry_type: term
category: trunked-radio
description: The Inter-Slot Signalling Channel (ISCH) is the short field that prefixes every P25 Phase 2 sub-frame — a 12-bit slot-type-plus-counter payload protected by extended Golay(24,12,8) — that tells a receiver what the sub-frame carries and keeps the two TDMA slots in step.
keywords: P25 ISCH, inter-slot signalling channel, Phase 2 slot type, Golay 24 12, sub-frame counter, TDMA slot coordination, TIA-102.BBAB 6.3
aka: [ISCH, "inter-slot signalling channel", "inter-slot signaling channel"]
autolink: true
infobox:
  - { label: Payload, value: 12 bits (SlotType + Counter) }
  - { label: FEC, value: extended Golay(24,12,8) }
  - { label: On wire, value: 24 bits = 12 dibits }
  - { label: Spec, value: TIA-102.BBAB §6.3 }
see_also: [golay-code, tdma, p25-phase-2, p25-phase-2-sync-word, p25-phase-2-superframe, p25-mac-pdu, p25-phase-2-hdqpsk]
related_reading:
  - { title: "SDR Internals, Part 9: Framing & forward error correction", url: /blog/deep-dives/sdr-internals-09-framing-fec/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Project_25
  - https://en.wikipedia.org/wiki/Binary_Golay_code
---

The **P25 ISCH** (**Inter-Slot Signalling Channel**) is the short field that prefixes every
[Phase 2](/reference/p25-phase-2/) sub-frame and names what that sub-frame carries.[^wiki] It is the
field a receiver reads to tell a voice sub-frame from a MAC-signalling sub-frame *without* inspecting
the payload — 12 information bits protected by an extended [Golay(24,12,8)](/reference/golay-code/)
code, so it survives a noisy channel and can be trusted before any of the heavier FEC downstream has
run.[^golay] Because it recurs on both [TDMA](/reference/tdma/) slots, the ISCH is also how the two
slots stay coordinated: each one carries a counter naming its position in the 12-sub-frame superframe.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="A sub-frame begins with the sync region, then the ISCH: 4 bits of slot type, 4 bits of counter, and 4 reserved bits, Golay(24,12,8) encoded to 24 bits (12 dibits) that correct up to three bit errors before the payload is read." xmlns="http://www.w3.org/2000/svg">
  <rect x="20" y="30" width="70" height="26" rx="3" fill="currentColor" fill-opacity="0.10" stroke="currentColor" stroke-width="1.1"/>
  <text x="55" y="47" text-anchor="middle" font-size="8" fill="currentColor">sync</text>
  <rect x="90" y="30" width="70" height="26" rx="3" fill="currentColor" fill-opacity="0.28" stroke="currentColor" stroke-width="1.1"/>
  <text x="125" y="47" text-anchor="middle" font-size="8" fill="currentColor">ISCH · 24b</text>
  <rect x="160" y="30" width="280" height="26" rx="3" fill="none" stroke="currentColor" stroke-width="1.1"/>
  <text x="300" y="47" text-anchor="middle" font-size="8" fill="currentColor">sub-frame payload (voice or MAC)</text>
  <rect x="60" y="78" width="70" height="24" fill="currentColor" fill-opacity="0.22" stroke="currentColor" stroke-width="1"/>
  <text x="95" y="94" text-anchor="middle" font-size="8" fill="currentColor">SlotType 4b</text>
  <rect x="130" y="78" width="70" height="24" fill="currentColor" fill-opacity="0.14" stroke="currentColor" stroke-width="1"/>
  <text x="165" y="94" text-anchor="middle" font-size="8" fill="currentColor">Counter 4b</text>
  <rect x="200" y="78" width="70" height="24" fill="none" stroke="currentColor" stroke-width="1"/>
  <text x="235" y="94" text-anchor="middle" font-size="8" fill="currentColor">rsvd 4b</text>
  <path d="M125 56 L145 74" fill="none" stroke="currentColor" stroke-width="0.8" stroke-dasharray="2 2"/>
  <text x="20" y="128" font-size="8" fill="currentColor">Golay(24,12,8) corrects up to 3 bit errors → SlotType read before any payload FEC</text>
</svg>
<figcaption>Each Phase 2 sub-frame opens with the ISCH: 12 data bits (slot type, sub-frame counter, reserved) Golay-encoded to 24 bits, letting the decoder classify the sub-frame under error before touching the payload.</figcaption>
</figure>

## Field layout

The 12 data bits, in GopherTrunk's working model, are: **bits 0–3** the 4-bit `SlotType` (voice-4V,
voice-2V, MAC-PTT, MAC-idle, MAC-active, and the other MAC variants); **bits 4–7** a **counter** naming
the sub-frame's 0..11 position within the 360 ms superframe; and **bits 8–11** reserved, transmitted as
zero. Those 12 bits are wrapped in the extended **Golay(24,12,8)** code already in
`internal/radio/framing` — a distance-8 block code that corrects up to three bit errors within its 24
coded bits (12 dibits on wire). A receiver runs the Golay decoder over the codeword, reports how many
bits it corrected, and rejects the ISCH outright if the codeword falls outside the correction radius.

TIA-102.BBAB §6.3 defines the ISCH and its FEC, but the repository has no figure pinning the exact bit
packing or code choice, so this layout is deliberately the project's *working model*. All ISCH
knowledge is confined to one file: the rest of the pipeline only ever sees a decoded `SlotType`, so if
a real capture shows a different packing, the correction stays local.

## Keeping the slots coordinated

Phase 2 interleaves two logical timeslots on one carrier, and both need to agree on where they are in
the shared 12-sub-frame [superframe](/reference/p25-phase-2-superframe/). The ISCH counter is what
supplies that agreement: every sub-frame announces its own position, so a decoder that locks the
[sync word](/reference/p25-phase-2-sync-word/) can immediately place the sub-frame on the 0..11 grid and
know which slot's stream it belongs to — even across a burst that briefly corrupts the payload. The
`SlotType` half then routes the sub-frame: a voice type sends the payload to the vocoder chain, while a
MAC type ([MAC PDU](/reference/p25-mac-pdu/)) sends it to the signalling FEC and opcode dispatch. In
effect the ISCH is a compact, heavily protected table-of-contents entry emitted once per sub-frame.

## Relevance to SDR

`internal/radio/p25/phase2/isch.go` implements `DecodeISCH` (Golay decode → `SlotType` + `Counter` +
corrected-error count) and its `EncodeISCH` inverse for building fixtures. The superframe decoder calls
it once per sub-frame and stamps the result onto the sub-frame it just sliced, which is what lets the
trunking layer follow a call across the two slots. Because the ISCH is short and well protected, it is
one of the most reliable fields on a marginal Phase 2 channel — often decodable when the payload behind
it is not. The spec is TIA-102.BBAB §6.3.

## Sources

[^wiki]: [Project 25](https://en.wikipedia.org/wiki/Project_25) — Wikipedia, on P25 Phase 2 and its TDMA sub-frame structure.
[^golay]: [Binary Golay code](https://en.wikipedia.org/wiki/Binary_Golay_code) — Wikipedia, on the extended Golay(24,12,8) code that protects the ISCH.
