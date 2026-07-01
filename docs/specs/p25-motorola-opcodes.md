# Motorola proprietary P25 TSBK opcodes (MFID 0x90)

Status of Motorola-specific (MFID `0x90`) Phase 1 TSBK opcodes seen on real
systems. Most P25 systems run Motorola gear, so these come up constantly.

> ⚠️ **Nothing here is wired into the system map.** Opcodes `0x05` and `0x09`
> (MFID 0x90) are named and their raw payload is captured by `logVendorProbe`
> ([`control.go`](https://github.com/MattCheramie/GopherTrunk/blob/main/internal/radio/p25/phase1/control.go)) for the record;
> no fields are decoded and nothing is published. A guessed bit layout would
> inject phantom data — the bad-data class the corroboration/identity work
> removed.

## Already decoded (MFID 0x90)

See [`tsbk_vendor.go`](https://github.com/MattCheramie/GopherTrunk/blob/main/internal/radio/p25/phase1/tsbk_vendor.go): patch /
group-regroup add (0x00) / delete (0x01), patch-group channel grant (0x02) /
update (0x03), and talker-alias fragments (0x15). Note also that the **standard**
band-plan and network/site/secondary broadcasts (`0x33/0x34/0x39/0x3A/0x3B/0x3C/0x3D`)
are decoded even under a vendor MFID (PR #728) — so on a Motorola site,
neighbours come from the standard `ADJ_STS_BCST` (0x3C), **not** from a
proprietary opcode. If neighbours are empty, capture the `0x3C` debug lines.

## Opcodes 0x05 and 0x09 — named, not field-decoded

A community snippet (Discord, 2026-06) claimed `0x05` is a "roam beacon" and
`0x09` a "scan marker" carrying neighbour / secondary-CC data, and supplied a Go
parser. That was **wrong** — cross-checked against the two reference decoders:

| Opcode | What it actually is (SDRtrunk) | OP25 (boatbod) | Field-decoded anywhere? |
| --- | --- | --- | --- |
| `0x05` | `MOTOROLA_OSP_TRAFFIC_CHANNEL_ID` ("Motorola Traffic Channel") | not handled | **No** — SDRtrunk's `MotorolaTrafficChannel` is a hex-only stub |
| `0x09` | `MOTOROLA_OSP_SYSTEM_LOADING` ("Motorola System Loading") | not handled | **No** — SDRtrunk's `ChannelLoading` is an explicit "unknown / under reverse-engineering" placeholder |

Findings:

- **Neither is a neighbour or secondary-CC message.** 0x05 concerns a traffic
  channel; 0x09 is system/site loading telemetry. They are not the fix for
  missing neighbours (that's the standard `0x3C`).
- **No authoritative field layout exists.** OP25 ignores both; SDRtrunk
  categorises them by opcode but does **not** extract fields (both classes are
  stubs / RE placeholders). There is nothing trustworthy to "implement from".
- The community `0x09` parser pulled only `0x0C 0x80` from `0c80000000000000`
  (rest zeros) — i.e. no usable data — and the `0x05` "roam beacon" layout was a
  fresh guess (decoding 0x05 with the standard UU_ANS_REQ layout already gave
  garbage: `src=0`, `target=0x3C0000`).

So this repo mirrors the reference decoders: **name** the two opcodes
(`MOTOROLA_OSP_TRAFFIC_CHANNEL_ID` / `MOTOROLA_OSP_SYSTEM_LOADING`) and **capture**
their raw payload (`logVendorProbe`, up to 64 samples/opcode at INFO — grep
`motorola opcode`), but field-decode nothing.

## If they're ever worth field-decoding

They would need genuine reverse-engineering, since no reference decoder has done
it: collect a correlation set of raw payloads (already captured by
`logVendorProbe`) against known site state and reverse the layout, then prove it
with an `Assemble`/`Parse` round-trip test before wiring anything into
`NetworkModel`. Given 0x05/0x09 carry no topology data, this is low priority.

## Sources

- SDRtrunk — `module/decode/p25/phase1/message/tsbk/Opcode.java` (opcode →
  `MOTOROLA_OSP_TRAFFIC_CHANNEL_ID` = 5, `MOTOROLA_OSP_SYSTEM_LOADING` = 9) and
  the `.../motorola/osp/` message classes (`MotorolaTrafficChannel`,
  `ChannelLoading` — both stubs). <https://github.com/DSheirer/sdrtrunk>
- OP25 (boatbod) — `op25/gr-op25_repeater/apps/trunking.py` (Motorola handling
  covers only opcodes 0x00–0x03). <https://github.com/boatbod/op25>
