---
slug: dpmr-csbk
title: dPMR Beacon Messages
entry_type: term
category: trunked-radio
description: "dPMR has no CSBK. A Mode 3 trunked system signals on a beacon channel: every 110 ms a frameset of one 72-bit message (Message_Type, then fields such as the Aloha or the Goto Channel that sends radios to a traffic channel) and three 17-bit SYScast fields (the System Identity Code and broadcasts such as the calling party)."
keywords: dPMR beacon, Mode 3, Message_Type, Goto Channel, Aloha, SYScast, System Identity Code, dPMR CSBK, ETSI TS 102 658, trunking grant
aka: ["dPMR CSBK", "dPMR beacon message", "dPMR Goto Channel", "SYScast"]
autolink: true
infobox:
  - { label: Message, value: "72 bits: 4-bit Message_Type + 68 bits" }
  - { label: SYScast, value: "3 x 17 bits per frameset" }
  - { label: Frameset, value: "528 bits, 110 ms" }
  - { label: Grant, value: "Goto Channel broadcast" }
  - { label: Spec, value: ETSI TS 102 658 clauses 4.2.3.7, 5.2, 5.5.32 }
see_also: [dpmr, dpmr-frame-sync, dpmr-channel-coding, channel-grant, radio-id, talkgroup, control-channel]
cite_urls:
  - https://www.etsi.org/deliver/etsi_ts/102600_102699/102658/02.06.01_60/ts_102658v020601p.pdf
  - https://en.wikipedia.org/wiki/DPMR
---

A [dPMR](/reference/dpmr/) Mode 3 system has no "CSBK". Its control channel is a **beacon**:
a base station transmitting continuously, in framesets of 528 bits (110 ms), each a **message**
slot and a **SYScast** slot.[^etsi]

```
FS1 | MI0 | CC | MI1 | SYC1 | SYC2 | SYC3 | FS1 | MI0 | ...
 48   120   24   120    72     72     72    48
```

## Messages

The message slot carries one 72-bit message twice (MI0, MI1), either side of the channel code.
Its first four bits are the Message_Type (table 5.3); a beacon's set-up traffic is mostly
**Broadcast** (`1011`), whose MI_TYPE field (bits 61 to 63) says which broadcast it is:

| MI_TYPE | Broadcast | Carries |
|---------|-----------|---------|
| `000` | Aloha | the MS addressed, the 14-bit System Identity Code, random-access parameters |
| `001` | Announcement | call timers, vote now, mass registration, real time |
| `010` | Move | an MS to an alternative base station |
| `011` | **Goto Channel** | the called party or talkgroup, the traffic channel's transmit and receive frequency, the communication mode, the emergency flag |

A **Goto Channel** is the trunking grant. Its frequencies are coded as the band's lower edge plus
the channel separation (3.125 or 5 kHz) times a 15-bit field (clause 5.5.15), so no band plan is
needed except for the band the standard leaves "custom". The calling party is not in the message:
it rides the SYScast2 and SYScast3 of the same frameset.

## SYScast

The three 17-bit SYScast fields are system broadcasts. SYScast1 is always the System Identity
Code and whether radios must register. SYScast2 and SYScast3 open with a 3-bit type: call timers,
real time, the common frame counter, or the calling party address (ten most significant bits in
SYScast2, fourteen least in SYScast3) right after a Goto Channel.

## Addresses

Addresses are 24 bits, mapped from seven dialled characters by the B2 algorithm of annex A:
`1234567` is `1B 91 FD`. A `*` in one of the last four characters makes the address a talkgroup
(`012345*` is `02 C0 0A`); an all-numeric address may still be a talkgroup a radio has in its own
table, which the air interface does not say.

## In GopherTrunk

`internal/radio/dpmr/beacon_msg.go` parses the messages and SYScast fields, `beacon.go` frames
the beacon, and `control.go` locks on the System Identity Code and publishes each Goto Channel
for a voice call as a grant the dPMR voice chain follows. An earlier model on this page, an
80-bit "CSBK" with 5-bit message types and a 16-bit channel number, matched nothing on the air
and was removed (issue #1374).

## Sources

[^etsi]: [ETSI TS 102 658 V2.6.1](https://www.etsi.org/deliver/etsi_ts/102600_102699/102658/02.06.01_60/ts_102658v020601p.pdf) — clauses 4.2.3.7 (beacon channel), 5.2 (message frames, tables 5.3, 5.29, 5.30, 5.38), 5.5.15 (frequency coding), 5.5.32 (SYScast) and annex A.1.2 (B2 address mapping).
