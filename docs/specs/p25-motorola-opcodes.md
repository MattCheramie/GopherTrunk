# Motorola proprietary P25 TSBK opcodes (MFID 0x90) — reverse-engineering status

Tracking doc for Motorola-specific (MFID `0x90`) Phase 1 TSBK opcodes observed
on real systems but **not yet decoded**. Most P25 systems run Motorola gear, so
these are worth reversing — but only from a *confirmed* layout. This file
records the leads so the work isn't lost and isn't re-guessed.

> ⚠️ **Nothing here is wired into the system map.** Opcodes `0x05` and `0x09`
> (MFID 0x90) are captured raw by `logVendorProbe`
> ([`control.go`](../../internal/radio/p25/phase1/control.go)) for offline
> analysis and decode/publish **nothing**. A guessed bit layout would inject
> phantom neighbours / frequencies — the exact bad-data class the corroboration
> and identity work removed — so real decoding waits on verification.

## Already decoded (MFID 0x90)

See [`tsbk_vendor.go`](../../internal/radio/p25/phase1/tsbk_vendor.go): patch /
group-regroup add (0x00) / delete (0x01), patch-group channel grant (0x02) /
update (0x03), and talker-alias fragments (0x15). Note also that the **standard**
band-plan and network/site/secondary broadcasts (`0x33/0x34/0x39/0x3A/0x3B/0x3C/0x3D`)
are decoded even under a vendor MFID (PR #728) — so on a Motorola site,
neighbours should come from the standard `ADJ_STS_BCST` (0x3C) first. If they're
empty, capture the `0x3C` debug lines before assuming the data only lives in a
proprietary opcode.

## Candidate opcodes (observed, undecoded)

### Opcode 0x09 (MFID 0x90)
- **Observed payload:** `0c80000000000000` (`lb=true`), repeated continuously.
- **Hypothesis (community, unverified):** a "scan marker" radios hunt for when
  picking up a control channel.
- **Assessment:** the payload is `0x0C80` followed by all zeros — a near-constant
  beacon with **no embedded site/frequency/neighbour fields**. Even the
  community parser only pulls two bytes (`0x0C`, `0x80`) and yields no map data.
  So this opcode, as observed, does **not** carry neighbour or secondary-CC
  information. Likely a periodic status/keepalive flag, not a topology message.

### Opcode 0x05 (MFID 0x90)
- **Observed:** logged on the same Motorola site; raw payload bytes not yet
  captured in a shared sample.
- **Hypotheses (unverified):** community snippet proposes a "roam beacon"
  (sub-type / sys-id fragment / neighbour site / channel IDEN + number /
  preference class / RSSI threshold); separately it's the *standard* opcode
  number for `UU_ANS_REQ`.
- **Assessment:** decoding this 0x05 with the **standard** UU_ANS_REQ layout
  produced garbage in the field (`src=0`, `target=0x3C0000`) — which is why
  PR #730 stopped decoding vendor-MFID 0x05. The "roam beacon" layout is a
  different, equally-unverified guess. No real `0x05` payload bytes have been
  correlated against known neighbours yet.

## Why these are not implemented

The community-provided Go parser (Discord, 2026-06) is **not** sourced from a
spec or a reference decoder and shows hallmarks of AI fabrication: invented
terminology ("RoamBeacon", "ScanMarker", "MarkerMagic", "injection marker
0x0C80"), no checkable citation, and a 0x09 parser that extracts no usable data.
Implementing it would re-introduce phantom map entries. P25 standard opcode
designations come from TIA-102.AABC-D; **vendor** opcodes are not in any
standard, so the only trustworthy sources are a reference decoder or
ground-truth correlation.

## Verification plan

To decode either opcode for real, gather **one** of:

1. **A reference decode** — what OP25 (boatbod, `trunking.py`) and/or SDRTrunk
   (`MotorolaTalkgroupOpcode` / MFID-90 handlers) actually do with MFID 0x90
   opcodes `0x05` / `0x09`. Mirror that layout (with the working-model
   disclaimer + symmetric `Assemble`/`Parse` + round-trip test the existing
   vendor opcodes use). If neither decodes them, they are genuinely
   undocumented and a guess is unjustified.
2. **Ground-truth correlation** — ~10–20 real raw payloads (now captured by
   `logVendorProbe`, up to 64 per opcode at INFO: grep
   `motorola candidate opcode`) **paired** with the site's true neighbour list /
   secondary-CC frequencies from a known-good decoder. Correlate the varying
   payload fields against the known values to reverse the bit layout, then prove
   it with an `Assemble`/`Parse` round-trip test before wiring it into
   `NetworkModel`.

Once a layout is confirmed, the dispatch in `dispatchVendorTSBK` swaps
`logVendorProbe` for a real parser + `ApplyAdjacentSite` /
`ApplySecondaryControlChannel` call.
