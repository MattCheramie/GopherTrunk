---
slug: dmr-reverse-channel
title: DMR Reverse Channel
entry_type: term
category: trunked-radio
description: The DMR Reverse Channel is in-call signalling carried in the 32-bit embedded field of a voice burst — 11 information bits protected by 21 FEC parity bits — used for fast link feedback distinct from the trunking control channel's CSBKs.
keywords: DMR Reverse Channel, RC signalling, embedded fragment, 11 info 21 parity, MS-RC, in-call signalling, LCSS single, ETSI TS 102 361-1 9.1.4
aka: ["RC", "Reverse Channel", "DMR reverse channel"]
autolink: true
infobox:
  - { label: Carrier, value: 32-bit embedded fragment }
  - { label: Split, value: 11 info + 21 parity bits }
  - { label: Gate, value: "EMB LCSS == Single + non-null" }
  - { label: Spec, value: ETSI TS 102 361-1 §6.2 / §9.1.4 }
see_also: [dmr-emb, dmr-embedded-lc, dmr-voice-superframe, dmr-sync-patterns, csbk, control-channel, forward-error-correction, dmr]
cite_urls:
  - https://en.wikipedia.org/wiki/Digital_mobile_radio
  - https://en.wikipedia.org/wiki/Forward_error_correction
---

The **DMR Reverse Channel** (**RC**) is in-call signalling carried in the same 32-bit embedded
fragment a voice [burst](/reference/dmr-burst/) uses for link control — but as a single,
self-contained word rather than a fragment of a longer [Full LC](/reference/dmr-full-link-control/).[^wiki]
It is fast link feedback that rides *inside* an ongoing call, distinct from the
[CSBKs](/reference/csbk/) a trunking [control channel](/reference/control-channel/) carries. Of the
32 fragment bits, 11 are information and the remaining 21 are FEC parity protecting them.[^fec]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 120" role="img" aria-label="A 32-bit embedded fragment split into an 11-bit reverse-channel information field followed by a 21-bit FEC parity field; it is only treated as a reverse channel when the framing EMB marks a single non-link-control fragment and the field is not all zero." xmlns="http://www.w3.org/2000/svg">
  <rect x="20" y="34" width="130" height="28" fill="currentColor" fill-opacity="0.26" stroke="currentColor" stroke-width="1.1"/>
  <text x="85" y="52" text-anchor="middle" font-size="8" fill="currentColor">RC info · 11 bits</text>
  <rect x="150" y="34" width="270" height="28" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.1"/>
  <text x="285" y="52" text-anchor="middle" font-size="8" fill="currentColor">FEC parity · 21 bits</text>
  <text x="20" y="84" font-size="8" fill="currentColor">only when EMB LCSS == Single (single, non-LC fragment)</text>
  <text x="20" y="100" font-size="8" fill="currentColor">all-zero field = ETSI null / idle → rejected</text>
</svg>
<figcaption>The 32-bit embedded fragment splits into 11 information bits and 21 parity bits; a fragment is treated as RC only when its EMB marks a single non-LC fragment and the field is non-null.</figcaption>
</figure>

## Carriers and framing

The RC rides one of two carriers, per ETSI TS 102 361-1 §6.2 / §9.1.4:

- the **32-bit embedded fragment** of an ordinary voice burst, when the framing
  [EMB](/reference/dmr-emb/) marks a single, non-LC fragment (`LCSS == Single`); or
- a short MS-sourced **RC burst** framed by the dedicated MS-RC
  [sync word](/reference/dmr-sync-patterns/) (`dmr.MSRC`).

GopherTrunk decodes only the first — the embedded carrier a downlink receiver actually observes,
since it is already pulling the embedded field out of every voice burst. The short MS-RC burst is a
mobile *uplink* transmission a downlink scanner does not normally receive: its sync is detected by
`dmr.MSRC`, but the burst itself is not present on the downlink, so no payload decode is attempted.
`DecodeReverseChannel` (`internal/radio/dmr/rc.go`) reads the 32-bit fragment, rejects the all-zero
"null embedded message" idle, and returns the 11-bit `Info` and 21-bit `Parity`. The caller must
confirm the EMB's `LCSS == LCSSSingle` before treating a fragment as RC; a
[voice superframe](/reference/dmr-voice-superframe/) decodes the first RC it sees across bursts B–E.

## Impl-gap: parity preserved, not applied

GopherTrunk **preserves the 21 parity bits but does not yet use them** to correct or validate the 11
information bits, and it leaves the per-opcode sub-field semantics of `Info` unparsed. The reason is
honestly documented in `rc.go`: there is no real off-air RC capture and no open reference decoder
(dsd-fme, pd0mz/go-dmr) that implements the (32,11) RC FEC to validate against. The integrity gate is
therefore the caller's EMB framing check (`LCSS == Single`) plus the null/idle rejection, mirroring
the capture-pending posture already documented for the EMB's own QR(16,7) and the MBC last-block CRC.
A future off-air capture can promote the parity to a real FEC gate without changing any callers.

## Relevance to SDR

For a scanner, the RC matters mostly as something to recognise and *not* mis-handle. A single-fragment
embedded field is not part of a four-fragment Full LC, so a decoder that treated every fragment as LC
would corrupt its reassembly; separating the RC path (gated on `LCSS == Single`) from the LC path keeps
the Full LC reassembly clean and lets the null/idle case be dropped rather than parsed. `HasRC` / `RC`
on the decoded superframe surface the 11-bit word when present, in-call signalling independent of the
LC, ready to become a fully-validated channel once a capture exists to pin the FEC against.

## Sources

[^wiki]: [Digital mobile radio](https://en.wikipedia.org/wiki/Digital_mobile_radio) — Wikipedia, on the DMR standard and its in-call embedded signalling.
[^fec]: [Forward error correction](https://en.wikipedia.org/wiki/Forward_error_correction) — Wikipedia, on the parity protection the RC information bits carry.
