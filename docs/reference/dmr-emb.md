---
slug: dmr-emb
title: DMR EMB
entry_type: term
category: trunked-radio
description: The EMB is the 16-bit Embedded signalling header that frames the 32-bit fragment inside the sync field of DMR voice bursts B–F — carrying Color Code, a privacy flag, and the LCSS fragment-position field, nominally protected by a QR(16,7,6) code that GopherTrunk does not yet apply.
keywords: DMR EMB, embedded signalling, LCSS, color code, privacy indicator, QR 16 7, DMR voice burst, ETSI TS 102 361-1 9.1.4
aka: ["EMB", "embedded signalling header"]
autolink: true
infobox:
  - { label: Length, value: 16 bits (two 8-bit halves) }
  - { label: Carries, value: CC (4) + PI (1) + LCSS (2) }
  - { label: FEC, value: "QR(16,7,6) — not applied in GT" }
  - { label: Spec, value: ETSI TS 102 361-1 §9.1.4 }
see_also: [dmr-embedded-lc, dmr-voice-superframe, dmr-reverse-channel, color-code, dmr-full-link-control, dmr, dmr-burst, dmr-encryption]
cite_urls:
  - https://en.wikipedia.org/wiki/Digital_mobile_radio
  - https://en.wikipedia.org/wiki/Quadratic_residue_code
---

The **DMR EMB** (**Embedded signalling** header) is the 16-bit control word that frames the
signalling carried inside the 48-bit sync field of voice [bursts](/reference/dmr-burst/) B–F.[^wiki]
A voice burst has no room for a separate signalling channel, so DMR reuses the central field:
16 EMB bits, split into an 8-bit half at each end, wrap a 32-bit **fragment** in the middle. The
EMB itself carries three things — the 4-bit [Color Code](/reference/color-code/), a 1-bit
**PI** (privacy/[encryption](/reference/dmr-encryption/) indicator), and the 2-bit **LCSS** that
marks where the enclosed fragment sits within a longer message — and is nominally protected by a
QR(16,7,6) [quadratic-residue](/reference/golay-code/) code.[^qr]

<figure class="figure" markdown="0">
<svg viewBox="0 0 470 130" role="img" aria-label="The 48-bit voice-burst sync field split into three parts: an 8-bit EMB most-significant half, a 32-bit embedded fragment, and an 8-bit EMB least-significant half; the two EMB halves reassemble into a 16-bit header holding color code, a privacy bit, and the two-bit LCSS." xmlns="http://www.w3.org/2000/svg">
  <rect x="16" y="34" width="70" height="28" fill="currentColor" fill-opacity="0.26" stroke="currentColor" stroke-width="1.1"/>
  <text x="51" y="52" text-anchor="middle" font-size="7.5" fill="currentColor">EMB hi · 8</text>
  <rect x="86" y="34" width="290" height="28" fill="currentColor" fill-opacity="0.10" stroke="currentColor" stroke-width="1.1"/>
  <text x="231" y="52" text-anchor="middle" font-size="8" fill="currentColor">embedded fragment · 32 bits</text>
  <rect x="376" y="34" width="70" height="28" fill="currentColor" fill-opacity="0.26" stroke="currentColor" stroke-width="1.1"/>
  <text x="411" y="52" text-anchor="middle" font-size="7.5" fill="currentColor">EMB lo · 8</text>
  <text x="16" y="84" font-size="8" fill="currentColor">EMB (16) = CC[15:12] · PI[11] · LCSS[10:9] · spare[8:0]</text>
  <text x="16" y="100" font-size="8" fill="currentColor">LCSS: 00 single/null · 01 first · 11 continuation · 10 last</text>
</svg>
<figcaption>The 48-bit field splits into an 8-bit EMB half at each end wrapping a 32-bit fragment; the reassembled 16-bit EMB gives the color code, a privacy bit, and the LCSS position code.</figcaption>
</figure>

## Fields and LCSS

GopherTrunk reads the EMB systematically in `internal/radio/dmr/emb.go`. Within the 16-bit word
(MSB-first) the Color Code is bits 15–12, PI is bit 11, and LCSS is bits 10–9; the remaining bits
are spare. The **LCSS** (Link Control Start/Stop) is what turns four separate fragments into one
message:

| LCSS | Bits | Meaning |
|---|---|---|
| Single | `00` | single-fragment payload — a [Reverse Channel](/reference/dmr-reverse-channel/) word or the null idle |
| First | `01` | first fragment of a Full Link Control (burst B) |
| Continuation | `11` | continuation fragment (bursts C, D) |
| Last | `10` | last fragment of a Full LC (burst E) |

`SplitEmbeddedField` takes the 48-bit field (one bit per byte, MSB-first) and returns the decoded
EMB plus the 32-bit fragment; `AssembleEmbeddedField` is its inverse for building synthetic
streams. A First → Continuation → Continuation → Last sequence across bursts B–E signals a
four-fragment [embedded Link Control](/reference/dmr-embedded-lc/); a Single fragment is handled
on its own.

## Impl-gap: QR(16,7) not applied

The ETSI spec protects the 7 EMB information bits with a QR(16,7,6) code. **GopherTrunk does not
apply it.** The EMB bits are read systematically, and integrity is enforced *downstream* rather
than at the EMB: the four-fragment [embedded LC](/reference/dmr-embedded-lc/) block has its own
BPTC(128,72) + 5-bit checksum, and a single RC fragment is gated by the caller's LCSS framing plus
a null/idle rejection. This mirrors the capture-pending posture documented for the other embedded
FEC layers (see `emb.go` and `docs/status.md`) — the QR check is a natural follow-up but is not on
the critical path while the BPTC/CRC layers are the integrity gate.

## Relevance to SDR

The EMB is what makes DMR's in-band signalling usable to a downlink scanner. Reading it on every
voice burst is how GopherTrunk knows a Color Code (a second, per-burst confirmation of the system
identity from the slot type), whether the traffic is encrypted (PI), and — via LCSS — how to route
each 32-bit fragment: accumulate it toward a Full LC, or decode it as a Reverse Channel word. The
voice [superframe](/reference/dmr-voice-superframe/) decoder drives all of this from bursts B–E, so
the EMB sits directly on the path from raw voice dibits to a labelled talkgroup call.

## Sources

[^wiki]: [Digital mobile radio](https://en.wikipedia.org/wiki/Digital_mobile_radio) — Wikipedia, on the DMR standard and its embedded voice-burst signalling.
[^qr]: [Quadratic residue code](https://en.wikipedia.org/wiki/Quadratic_residue_code) — Wikipedia, on the QR code family the EMB's nominal FEC belongs to.
