---
slug: ysf-fich
title: YSF FICH
entry_type: algorithm
category: amateur-digital
description: The YSF FICH (Frame Information Channel) is the 32-bit Yaesu System Fusion header that names each frame's type, call mode, and data mode, protected by a CRC-16 CCITT and a K=5 half-rate trellis with puncturing and a 10×10 interleave.
keywords: YSF FICH, frame information channel, Yaesu System Fusion, C4FM, CRC-16 CCITT, K=5 trellis, Viterbi, puncturing, 10x10 interleave, FSW D471C9634D
aka: [FICH, "frame information channel", "YSF FICH"]
autolink: true
infobox:
  - { label: Info size, value: 32 bits + 16-bit CRC }
  - { label: On-air, value: 100 channel bits (50 dibits) }
  - { label: FEC, value: "K=5 ½-rate trellis + puncture + interleave" }
  - { label: FSW, value: "0xD471C9634D (40 bits)" }
see_also: [system-fusion-ysf, crc-16-ccitt, convolutional-code, viterbi-algorithm, interleaving, forward-error-correction, four-fsk, frame-synchronization]
cite_urls:
  - https://en.wikipedia.org/wiki/System_Fusion
  - https://en.wikipedia.org/wiki/Convolutional_code
---

The **YSF FICH** (**Frame Information Channel**) is the header block that opens the payload of
every [Yaesu System Fusion](/reference/system-fusion-ysf/) frame.[^ysf] Where the frame sync
word identifies *that* a YSF transmission is on-air, the FICH identifies *what kind*: the call
type (group or radio ID), the data mode (voice, data, or the mixed V/D modes), and the frame's
position inside its block and transmission sequence. Because everything downstream depends on
reading it correctly, its 32 information bits are wrapped in a
[CRC-16](/reference/crc-16-ccitt/) and a heavy [convolutional](/reference/convolutional-code/)
FEC chain so it survives a fading, mobile [4-FSK](/reference/four-fsk/) channel.[^conv]

<figure class="figure" markdown="0">
<svg viewBox="0 0 470 160" role="img" aria-label="The YSF FICH encoding chain: 32 information bits plus a 16-bit CRC form 48 bits, four zero tail bits are appended, a K=5 half-rate trellis produces 104 channel bits, four puncture positions are dropped to leave 100 on-air bits, and a 10 by 10 column interleave permutes them into the FICH region of the frame." xmlns="http://www.w3.org/2000/svg">
  <g font-size="8" fill="currentColor" text-anchor="middle">
  <rect x="16" y="20" width="150" height="24" fill="currentColor" fill-opacity="0.18" stroke="currentColor" stroke-width="1.1"/><text x="91" y="36">32 info + 16 CRC = 48</text>
  <path d="M166 32 L196 32" stroke="currentColor" stroke-width="1.1" marker-end="url(#a)"/>
  <rect x="196" y="20" width="120" height="24" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.1"/><text x="256" y="36">+ 4 tail bits</text>
  <path d="M256 44 L256 60" stroke="currentColor" stroke-width="1.1" marker-end="url(#a)"/>
  <rect x="150" y="60" width="212" height="24" fill="currentColor" fill-opacity="0.18" stroke="currentColor" stroke-width="1.1"/><text x="256" y="76">K=5 ½-rate trellis → 104 channel bits</text>
  <path d="M256 84 L256 100" stroke="currentColor" stroke-width="1.1" marker-end="url(#a)"/>
  <rect x="150" y="100" width="212" height="24" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.1"/><text x="256" y="116">puncture {0,1,102,103} → 100 bits</text>
  <path d="M256 124 L256 140" stroke="currentColor" stroke-width="1.1" marker-end="url(#a)"/>
  <rect x="150" y="140" width="212" height="18" fill="none" stroke="currentColor" stroke-width="1.1"/><text x="256" y="153" font-size="7.5">10×10 column interleave → FICH region</text>
  </g>
  <defs><marker id="a" markerWidth="6" markerHeight="6" refX="5" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>The FICH's 32 info bits plus a 16-bit CRC are trellis-encoded, punctured, and interleaved into 100 on-air channel bits — the protection a decoder reverses before it can trust the header.</figcaption>
</figure>

## How it works

A YSF frame is 480 symbols of 4800-baud 4-level C4FM. The first 20 dibits are the 40-bit
frame sync word **0xD471C9634D**, the next 100 dibits carry the FICH, and the rest is the DCH
voice/data payload. A sliding [detector](/reference/frame-synchronization/) locks onto the
FSW, then the decoder reverses the FICH's FEC chain:

1. **De-interleave.** The 100 on-air bits are permuted by a column-major 10×10 interleaver
   (output bit *k* pulls from input bit `(k%10)*10 + (k/10)`), which spreads a burst of channel
   errors across the codeword so no single trellis section takes them all.
2. **Depuncture.** Four channel bits punctured on the wire — positions `{0, 1, 102, 103}`,
   flanking the trellis's tail-bit boundary — are re-inserted as erasures, restoring the full
   104-bit trellis output.
3. **Viterbi decode.** A K=5 ½-rate [Viterbi](/reference/viterbi-algorithm/) decoder over the
   104 channel bits recovers the 48 information bits (32 FICH + 16 CRC); its path metric
   reports how many bit-flips it had to repair.
4. **Check the CRC.** The 16-bit trailer is a [CRC-16 CCITT](/reference/crc-16-ccitt/) (poly
   0x1021, initial value 0x0000) over the four leading info octets. A mismatch drops the frame.

## FICH fields

The 32 information bits pack MSB-first across four octets:

| Field | Bits | Meaning |
|---|---|---|
| FT | 2 | Frame Type — Header, Communications, Terminator, Test |
| CT | 2 | Call Type / call-sign mode — Group, Radio ID, reserved |
| BN | 2 | Block Number within the transmission |
| BT | 2 | Block Total |
| FN | 3 | Frame Number within the current block |
| FT (total) | 3 | Frame Total inside the block |
| DT | 2 | Data Type — V/D mode 1, Data FR, V/D mode 2, Voice FR |
| VoIP | 1 | 1 = transmission carries WIRES-X / VoIP |
| DT2 | 2 | Data Type 2 / mode-2 sub-field |
| SQM | 1 | Squelch mode — 0 open, 1 code-squelch active |
| SQ | 7 | Squelch code (split across octets 2 and 3) |
| DEV | 2 | Device / reserved |

The DT field is what tells a decoder how to treat the DCH region — the V/D modes multiplex
half-rate voice and data, while the full-rate modes carry only one of the two.

## In practice

The field packing matches what the open-source YSF decoders (DSDcc, MMDVMHost, Pi-Star) use,
and the four implementations agree byte-for-byte on the puncture schedule. That schedule's
exact origin is the JARL CAI reference; GopherTrunk flags it as best-effort pending real-air
capture validation — if a captured YSF transmission fails FICH CRC after Viterbi decode, the
puncture positions are the two lines most likely to need adjusting. The whole point of the
chain is robustness: interleaving turns a fading burst into isolated section errors, the
½-rate trellis mops those up, and the CRC is the final gate that keeps a mis-decoded header
from steering the rest of the frame wrong.

## Relevance to SDR

`internal/radio/ysf/fich.go` holds the field parse and CRC check (`ParseFICH`,
`AssembleFICH`); `fich_trellis.go` implements the K=5 ½-rate encode/decode, the puncture
schedule, and the 10×10 interleave (`EncodeFICHOnAir` / `DecodeFICHOnAir`); and `sync.go`
holds the FSW and its detector. Together they let GopherTrunk recognise a YSF transmission and
read its metadata — enough to surface a ham repeater's activity in the active-systems view.

## Sources

[^ysf]: [System Fusion](https://en.wikipedia.org/wiki/System_Fusion) — Wikipedia, on Yaesu System Fusion (C4FM) and its frame structure.
[^conv]: [Convolutional code](https://en.wikipedia.org/wiki/Convolutional_code) — Wikipedia, on the trellis coding the FICH uses for forward error correction.
