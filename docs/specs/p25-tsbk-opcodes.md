# P25 Phase 1 TSBK opcodes (OSP) — TIA designations

Reference table for the P25 Phase 1 (FDMA) Trunking Signalling Block (TSBK)
**Outbound Signalling Packet (OSP)** opcodes the control-channel decoder in
[`internal/radio/p25/phase1`](../../internal/radio/p25/phase1/) recognises.

The Go constant identifiers in [`opcodes.go`](../../internal/radio/p25/phase1/opcodes.go)
stay descriptive/CamelCase (idiomatic Go), while `Opcode.String()` and the
trailing comment on each constant carry the canonical **TIA designation** — so
debug logs read in spec terms (`opcode=NET_STS_BCST`, `opcode=UU_ANS_REQ`)
rather than the OP25-derived names they used to. This mirrors the convention
already used in [`internal/radio/nxdn`](../../internal/radio/nxdn/) for NXDN
message names.

**Source:** TIA-102.AABC-D, *Project 25 — FDMA Common Air Interface — Trunking
Control Channel Messages*, Table 7-1 (TSBK opcodes). The standard is published
by TIA and is **not** redistributable, so — per this directory's policy — it is
cited, not committed. See also the Aeroflex/Cobham application note
*Understanding Advanced P25 Control Channel Functions* for worked message
walk-throughs (e.g. UU_V_REQ → UU_ANS_REQ → UU_ANS_RSP call setup).

## OSP opcode table

| Hex | TIA designation | Meaning | Dispatched |
| --- | --- | --- | --- |
| 0x00 | `GRP_V_CH_GRANT` | Group Voice Channel Grant | ✅ grant |
| 0x02 | `GRP_V_CH_GRANT_UPDT` | Group Voice Channel Grant Update | ✅ grant |
| 0x03 | `GRP_V_CH_GRANT_UPDT_EXP` | Group Voice Channel Grant Update — Explicit | ✅ grant |
| 0x04 | `UU_V_CH_GRANT` | Unit-to-Unit Voice Channel Grant | ✅ grant |
| 0x05 | `UU_ANS_REQ` | Unit-to-Unit Answer Request | ✅ (standard MFID only) |
| 0x06 | `UU_V_CH_GRANT_UPDT` | Unit-to-Unit Voice Channel Grant Update | — |
| 0x08 | `TELE_INT_CH_GRANT` | Telephone Interconnect Voice Channel Grant | ✅ grant |
| 0x09 | `TELE_INT_CH_GRANT_UPDT` | Telephone Interconnect Voice Channel Grant Update | — |
| 0x0A | `TELE_INT_ANS_REQ` | Telephone Interconnect Answer Request | — |
| 0x14 | `SNDCP_DAT_CH_GRANT` | SNDCP Data Channel Grant | ✅ grant |
| 0x15 | `SNDCP_DAT_PAGE_REQ` | SNDCP Data Channel Page Request | — |
| 0x16 | `SNDCP_DAT_CH_ANN_EXP` | SNDCP Data Channel Announcement — Explicit | — |
| 0x18 | `STS_UPDT` | Status Update | — |
| 0x1A | `STS_Q` | Status Query | — |
| 0x1C | `MSG_UPDT` | Message Update | — |
| 0x1D | `RAD_MON_CMD` | Radio Unit Monitor Command | — |
| 0x1E | `RAD_MON_ENH_CMD` | Radio Unit Monitor Enhanced Command | — |
| 0x1F | `CALL_ALRT` | Call Alert | — |
| 0x20 | `ACK_RSP_FNE` | Acknowledge Response — FNE | — |
| 0x21 | `QUE_RSP` | Queued Response | — |
| 0x24 | `EXT_FNCT_CMD` | Extended Function Command | — |
| 0x27 | `DENY_RSP` | Deny Response | — |
| 0x28 | `GRP_AFF_RSP` | Group Affiliation Response | ✅ affiliation |
| 0x29 | `SCCB_EXP` | Secondary Control Channel Broadcast — Explicit | ✅ topology |
| 0x2A | `GRP_AFF_Q` | Group Affiliation Query | — |
| 0x2B | `LOC_REG_RSP` | Location Registration Response | ✅ topology (RFSS/Site) |
| 0x2C | `U_REG_RSP` | Unit Registration Response | ✅ registration |
| 0x2D | `U_REG_CMD` | Unit Registration Command | — |
| 0x2E | `AUTH_CMD` | Authentication Command | — |
| 0x2F | `U_DE_REG_ACK` | Unit De-Registration Acknowledge | — |
| 0x30 | `SYNC_BCST` | TDMA Synchronisation Broadcast | — |
| 0x33 | `IDEN_UP_TDMA` | Identifier Update for TDMA | ✅ band plan |
| 0x34 | `IDEN_UP_VU` | Identifier Update — VHF/UHF | ✅ band plan |
| 0x35 | `PROT_PARM_UPDT` | Protection Parameter Update | — |
| 0x36 | `ROAM_ADDR_CMD` | Roaming Address Command | — |
| 0x37 | `ROAM_ADDR_UPDT` | Roaming Address Update | — |
| 0x38 | `SYS_SRV_BCST` | System Service Broadcast | — |
| 0x39 | `SCCB` | Secondary Control Channel Broadcast | ✅ topology |
| 0x3A | `RFSS_STS_BCST` | RFSS Status Broadcast | ✅ topology |
| 0x3B | `NET_STS_BCST` | Network Status Broadcast (WACN + System ID) | ✅ topology |
| 0x3C | `ADJ_STS_BCST` | Adjacent Site Status Broadcast (neighbours) | ✅ topology |
| 0x3D | `IDEN_UP` | Identifier Update (700/800/900 MHz) | ✅ band plan |
| 0x3F | `PROT_PARM_BCST` | Protection Parameter Broadcast | — |

"Dispatched" marks opcodes the control channel acts on (`dispatchTSBK` in
[`control.go`](../../internal/radio/p25/phase1/control.go)); the rest are
decoded enough to log at debug and otherwise ignored. The
standard-broadcast/band-plan opcodes (`0x29/0x33/0x34/0x39/0x3A/0x3B/0x3C/0x3D`)
are dispatched **regardless of MFID**, because they live in the standard TIA
namespace on every vendor's control channel.

Note `0x29` is the *explicit* Secondary Control Channel Broadcast (separate
transmit/receive channels); `0x39` is the non-explicit variant. Earlier
revisions of this decoder mislabelled `0x39` as `SCCB_EXP` — the explicit
message is `0x29`, decoded by `ParseSecondaryControlChannelBroadcastExplicit`
and folded into the topology via `NetworkModel.ApplySecondaryControlChannelExplicit`.

## UU_ANS_REQ (0x05) note

`UU_ANS_REQ` is an OSP the FNE sends to a called unit during unit-to-unit call
setup; it carries Service Options, a 24-bit **Target Address** (called unit) and
a 24-bit **Source Address** (calling unit) — no channel is granted yet. The
decoder parses it (`ParseUnitToUnitAnswerRequest`) and publishes a
`unit.request` bus event **only under the standard MFID**. Under a vendor MFID
(e.g. Motorola 0x90) opcode 0x05 is in the manufacturer's namespace, not the
standard one: a field decode of a Motorola 0x05 with the standard layout
produced garbage IDs (`src=0`, `target=0x3C0000`), so vendor 0x05 is left
unhandled until its real layout is reversed.
