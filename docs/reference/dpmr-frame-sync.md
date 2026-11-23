---
slug: dpmr-frame-sync
title: dPMR Frame Sync
entry_type: term
category: synchronization
description: "dPMR marks its frames with four sync words (ETSI TS 102 658 clause 6.1): the 48-bit FS1 (57 FF 5F 75 D5 77) opening a header or a Mode 3 beacon message, the 24-bit FS2 (5F F7 7D) opening payload frames 1 and 3, the 24-bit FS3 (7D DF F5) opening the end frame, and the 48-bit FS4, FS1's complement, opening a packet-data header."
keywords: dPMR frame sync, FS1, FS2, FS3, FS4, 57FF5F75D577, 5FF77D, 7DDFF5, dPMR sync word, ETSI TS 102 658, Mode 3 beacon
aka: ["dPMR sync word", "dPMR FS1", "dPMR FS2"]
autolink: true
infobox:
  - { label: Words, value: "FS1, FS4 (48 bits); FS2, FS3 (24 bits)" }
  - { label: FS1, value: "57 FF 5F 75 D5 77" }
  - { label: FS2, value: "5F F7 7D" }
  - { label: FS3, value: "7D DF F5 (end frame)" }
  - { label: Spec, value: ETSI TS 102 658 clause 6.1 }
see_also: [dpmr, dpmr-csbk, dpmr-channel-coding, frame-synchronization, four-fsk, control-channel]
cite_urls:
  - https://www.etsi.org/deliver/etsi_ts/102600_102699/102658/02.06.01_60/ts_102658v020601p.pdf
  - https://en.wikipedia.org/wiki/DPMR
---

A **dPMR frame sync** is one of four fixed patterns that tell a
[dPMR](/reference/dpmr/) receiver where a frame starts and what kind of frame it is.[^etsi]
Every sync symbol is an outer one (±1050 Hz), so a sync word is also a strong level reference
for the 4FSK slicer.

| Word | Hex | Length | Opens |
|------|-----|--------|-------|
| FS1 | `57 FF 5F 75 D5 77` | 48 bits | a header frame; every Mode 3 beacon message |
| FS2 | `5F F7 7D` | 24 bits | payload frames 1 and 3 of each 320 ms superframe |
| FS3 | `7D DF F5` | 24 bits | the end frame of a transmission |
| FS4 | `FD 55 F5 DF 7F DD` | 48 bits | a packet-data header (FS1's symbol-wise complement) |

On a traffic channel a transmission is a header frame (preamble, FS1, two copies of the 72-bit
message information, the channel code), then superframes of four 80 ms payload frames, where FS2
opens frames 1 and 3 and the 24-bit channel code opens frames 2 and 4, then an end frame that FS3
opens.[^etsi] A Mode 3 **beacon** is continuous: a frameset every 110 ms of a message slot and a
SYScast slot, the SYScast slot ending with the FS1 that opens the next message (clause 4.2.3.7).

FS2 is only 12 symbols. A detector that allows one wrong symbol matches noise about once every
few minutes, so a receiver should not trust a lone FS2: GopherTrunk's traffic-channel framer
locks only when a control channel behind it passes its CRC7, or a clean channel code is followed
by the next sync word one frame pair later. FS1 is 24 symbols and far rarer by chance; a beacon
lock still requires the message behind it to pass its CRC8.

## In GopherTrunk

`internal/radio/dpmr/airframe.go` holds the four words; the traffic framer is `traffic.go`, the
beacon framer `beacon.go`. The 48-bit "FS1 / FS2 / FS3" words this page used to list
(`0x57FF5F75F575`, `0x5F7F77FD7DFD`, `0x7DDFFD5F55D5`) came from an earlier model that matched
nothing on the air; they were removed with it (issue #1374).

## Sources

[^etsi]: [ETSI TS 102 658 V2.6.1](https://www.etsi.org/deliver/etsi_ts/102600_102699/102658/02.06.01_60/ts_102658v020601p.pdf) — clauses 4.2.2, 4.2.3.7 and 6.1: frame formats, the beacon channel and the synchronisation sequences.
