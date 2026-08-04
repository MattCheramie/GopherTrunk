---
slug: tetra-llc
title: TETRA LLC
entry_type: term
category: trunked-radio
description: "The TETRA LLC (Logical Link Control) sublayer sits between the MAC and the MLE; its basic-link PDUs — BL-ADATA, BL-DATA, BL-UDATA and BL-ACK, each with or without a 32-bit FCS — wrap the TL-SDU that carries the MLE and CMCE call-control messages."
keywords: TETRA LLC, logical link control, BL-ADATA, BL-DATA, BL-UDATA, BL-ACK, basic link, TL-SDU, FCS, EN 300 392-2 21.2
aka: [LLC, "basic link", "logical link control"]
autolink: true
infobox:
  - { label: Sublayer, value: "Between MAC and MLE (Layer 2)" }
  - { label: Basic-link PDUs, value: "BL-ADATA, BL-DATA, BL-UDATA, BL-ACK" }
  - { label: Header width, value: 4–6 bits (+ optional 32-bit FCS) }
  - { label: Spec, value: "ETSI EN 300 392-2 §21.2" }
see_also: [tetra, tetra-mac-pdu, tetra-cmce-mle-pdu, tetra-logical-channels, cyclic-redundancy-check, control-channel]
cite_urls:
  - https://en.wikipedia.org/wiki/Terrestrial_Trunked_Radio
  - https://en.wikipedia.org/wiki/Logical_link_control
---

The **TETRA LLC** (**Logical Link Control**) sublayer is the thin Layer-2 layer between the
[MAC](/reference/tetra-mac-pdu/) and the MLE.[^tetra][^llc] When the MAC delivers a TM-SDU it
is not the MLE PDU directly — it is an *LLC PDU*: a short basic-link header (the LLC PDU type
plus sequence numbers, and an optional 32-bit trailer) wraps the **TL-SDU**, and the TL-SDU
is what carries the 3-bit MLE protocol discriminator and the
[CMCE](/reference/tetra-cmce-mle-pdu/) call-control message beneath it. Skipping this layer —
feeding the raw MAC TM-SDU straight into the CMCE parser — reads every Layer-3 field at the
wrong bit offset, because the basic-link header is 4 to 6 bits wide, and that corrupts the
decoded ISSI/GSSI.

<figure class="figure" markdown="0">
<svg viewBox="0 0 470 140" role="img" aria-label="A MAC TM-SDU carries an LLC basic-link PDU: a 4-bit type field, up to two sequence-number bits, then the TL-SDU, and for the with-FCS variants a 32-bit frame check sequence trailer; stripping the header and FCS yields the TL-SDU that carries the MLE and CMCE message." xmlns="http://www.w3.org/2000/svg">
  <rect x="14" y="34" width="150" height="26" rx="3" fill="none" stroke="currentColor" stroke-width="1.1" stroke-dasharray="3 2"/>
  <text x="89" y="50" text-anchor="middle" font-size="8" fill="currentColor">MAC TM-SDU</text>
  <rect x="14" y="78" width="44" height="26" fill="currentColor" fill-opacity="0.28" stroke="currentColor" stroke-width="1.1"/>
  <text x="36" y="91" text-anchor="middle" font-size="7" fill="currentColor">type 4</text>
  <text x="36" y="100" text-anchor="middle" font-size="6.5" fill="currentColor">N(S)/N(R)</text>
  <rect x="58" y="78" width="270" height="26" fill="currentColor" fill-opacity="0.10" stroke="currentColor" stroke-width="1.1"/>
  <text x="193" y="94" text-anchor="middle" font-size="8" fill="currentColor">TL-SDU → MLE discriminator → CMCE PDU</text>
  <rect x="328" y="78" width="128" height="26" fill="none" stroke="currentColor" stroke-width="1.1" stroke-dasharray="3 2"/>
  <text x="392" y="91" text-anchor="middle" font-size="7.5" fill="currentColor">FCS · 32 bits</text>
  <text x="392" y="100" text-anchor="middle" font-size="6.5" fill="currentColor">"with FCS" only</text>
  <path d="M89 60 L89 78" fill="none" stroke="currentColor" stroke-width="1"/>
  <text x="235" y="128" text-anchor="middle" font-size="8" fill="currentColor">strip header (+ FCS) → hand the TL-SDU up</text>
</svg>
<figcaption>The MAC TM-SDU is an LLC basic-link PDU: a 4-bit type field, up to two sequence-number bits, the TL-SDU, and for the with-FCS variants a 32-bit trailer; stripping the header and any FCS leaves the TL-SDU the MLE and CMCE layers parse.</figcaption>
</figure>

## The basic-link PDUs

The 4-bit LLC PDU type (§21.2.1, Table 21.1) selects the basic-link PDU and whether a 32-bit
Frame Check Sequence trails the PDU. Values 0–7 are the basic-link PDUs GopherTrunk handles;
8–15 are advanced-link, supplementary or Layer-2-signalling PDUs that do not carry CMCE call
control.

| Type | PDU | Header after the 4-bit type | FCS |
| --- | --- | --- | --- |
| 0x0 | BL-ADATA | N(R) + N(S) — 2 bits | no |
| 0x1 | BL-DATA | N(S) — 1 bit | no |
| 0x2 | BL-UDATA | (none — 4-bit type only) | no |
| 0x3 | BL-ACK | N(R) — 1 bit | no |
| 0x4 | BL-ADATA | N(R) + N(S) — 2 bits | yes |
| 0x5 | BL-DATA | N(S) — 1 bit | yes |
| 0x6 | BL-UDATA | (none) | yes |
| 0x7 | BL-ACK | N(R) — 1 bit | yes |

**BL-ADATA** carries acknowledged data with both a send and a receive sequence number;
**BL-DATA** carries acknowledged data with only a send sequence number; **BL-UDATA** is
unacknowledged with no sequence numbers (the smallest header, just the 4-bit type); and
**BL-ACK** is the acknowledgement, carrying a receive sequence number and, legitimately, no
TL-SDU of its own. The **with-FCS** variants add a 32-bit
[CRC](/reference/cyclic-redundancy-check/) computed over the PDU, stripped from the tail
before the TL-SDU is handed up.

## Stripping the header

Decoding is a matter of reading the 4-bit type, consuming the right number of
sequence-number bits, and — for a with-FCS PDU — trimming 32 bits off the end. What remains
between the header and the FCS is the TL-SDU, ready for the MLE discriminator and
[CMCE](/reference/tetra-cmce-mle-pdu/) parse. This is exactly the basic link used for CMCE
call-control signalling on the main control channel; the advanced link (AL-*, used for packet
data and SNDCP) is not a CMCE carrier and is reported as not-basic-link rather than
mis-parsed. osmo-tetra strips this layer in `tetra_llc_pdu_parse()` before
`tetra_mle_decode()`, and tetra-kit does the same in `llc.cc`.

## Relevance to SDR

`internal/radio/tetra/llc.go` implements the basic-link strip as a single function,
`ParseLLC`, which returns the TL-SDU and an `ok` flag. It reports `ok=false` for a
non-basic-link PDU, for a BL-ACK that carries no TL-SDU, and for a slice too short to hold
the header. Getting the header width right per PDU type is what keeps the CMCE parser reading
from the correct bit offset — a four-bit slip here silently corrupts every source and
destination identity in the decoded grant, a failure invisible to any test that encodes and
decodes with the same wrong assumption on both ends.

## Sources

[^tetra]: [Terrestrial Trunked Radio](https://en.wikipedia.org/wiki/Terrestrial_Trunked_Radio) — Wikipedia, on the TETRA protocol stack.
[^llc]: [Logical link control](https://en.wikipedia.org/wiki/Logical_link_control) — Wikipedia, on the LLC sublayer's place in the data-link layer.
