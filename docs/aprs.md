---
layout: page
title: APRS / AX.25 protocol layer
description: AX.25 frame parsing + APRS info-field decoding — positions, messages, status, bulletins
nav_group: Reference
---

# APRS / AX.25 protocol layer

GopherTrunk now decodes the **APRS** (Automatic Packet Reporting
System) packet format that rides on top of **AX.25**, the amateur-
radio link-layer frame used by packet radio. APRS is the dominant
amateur-radio metadata bus — position beacons, weather reports,
messages, bulletins, SAR coordination — and complements the
trunking-voice pipeline well for emergency-comms-adjacent operators
who already run GopherTrunk for public-safety scanning.

This PR lands the **protocol layer** in pure Go: AX.25 frame
parsing with HDLC CRC-16-CCITT validation, plus the APRS info-
field decoder for the operator-visible majority of packet types
(position reports, messages, status, bulletins). The DSP wiring
(1200 Bd Bell-202 AFSK → HDLC bit-stuff/de-stuff → frame
delimiter) follows in a focused follow-up PR, mirroring the
POCSAG protocol-first split from #372.

## What's wired

### `internal/radio/aprs/ax25`
AX.25 UI-frame parser.

- 7-byte AX.25 address packing (callsign + SSID + H/C-bit) with
  the spec's bit-shifted ASCII convention
- Up to 8 digipeater path entries past the source address
- Control + PID byte parsing (UI is the only frame type APRS
  uses, but the parser passes other types through)
- **HDLC CRC-16-CCITT** validation via the standard 0x8408
  reflected-bit polynomial; `frame.FCSOK` distinguishes clean
  vs. CRC-failed frames
- Display helpers (`Address.String()` → `"W1AW-9"` or `"WIDE2-1*"`,
  `Frame.PathString()` → `"WIDE1-1,WIDE2-1*"`)

### `internal/radio/aprs`
APRS info-field decoder.

Recognised packet types and what we extract:

| DTI byte | Type | Decoded fields |
| --- | --- | --- |
| `!` | Position (no timestamp, no msg) | lat/lon, symbol, comment |
| `=` | Position (no timestamp, messaging) | lat/lon, symbol, comment, `WithMessaging=true` |
| `/` | Position (timestamp, no msg) | lat/lon, symbol, comment, raw timestamp |
| `@` | Position (timestamp, messaging) | lat/lon, symbol, comment, raw timestamp, `WithMessaging=true` |
| `:` | Message / Bulletin | addressee, body, seqno, ack/rej flag |
| `>` | Status | text |
| `;` | Object | recognised; payload TBD |
| `_` | Weather | recognised; payload TBD |
| `T#` | Telemetry | recognised; payload TBD |
| `` ` ``, `'`, `0x1C`, `0x1D` | Mic-E (compressed) | recognised; decoder pending |
| anything else | Unknown | raw bytes preserved |

Position parsing covers:
- Standard `DDMM.hhH/DDDMM.hhH` lat/lon (hundredths of a minute)
- Both hemispheres for latitude (`N`/`S`) and longitude (`E`/`W`)
- Symbol table + symbol code byte pair
- Free-form comment after the symbol code
- Ambiguity-space tolerance (the spec allows spaces in low-
  precision digits — we treat them as 0)

Message parsing covers:
- 9-character addressee, trimmed of trailing spaces
- Optional `{NNN}` sequence number suffix
- `ack` / `rej` short-form replies (body becomes the seqno)
- Bulletin board (addressee starts with `BLN`) → `Bulletin` struct

## What's pending

- **DSP integration.** 1200 Bd Bell-202 AFSK demodulation, HDLC
  bit-stuffing reversal, and 0x7E flag-delimited frame extraction.
  Mirrors the POCSAG receiver pattern (FM demod → bit slicer →
  syncer); the AX.25 parser is ready to consume frame-delimited
  bytes the moment the DSP layer exists. Bell-202 itself is
  audio-band so the demod can run off a narrow-channelized FM
  audio stream — same path POCSAG uses minus the bit-stuffing.
- **Mic-E.** The compressed lat/lon format common on mobile
  trackers (Kenwood TH-D74, Yaesu FT-3D). Adds ~200 LOC for the
  base-91 unpacking + speed / course / altitude decode.
- **Weather / telemetry / object payloads.** The decoder
  recognises the type bytes but leaves the body in `Raw` for
  now. Operator-visible information from these would be helpful
  but the spec is involved.
- **Bus event + SQLite log + REST + web panel.** Once the DSP
  wiring exists, a new `events.KindAPRSPacket` plus an `aprs_log`
  table (mirroring `pager_log` from PR #373) ship alongside.
  The web panel renders position reports on a map and messages /
  status / bulletins in a chat-style log.

## References

- APRS Protocol Reference 1.0.1 (1998-08-07) — the canonical text.
  `http://www.aprs.org/doc/APRS101.PDF`
- AX.25 Link Access Protocol for Amateur Packet Radio, v2.2 — TAPR
  / ARRL, 1998. `http://www.ax25.net/AX25.2.2-Jul%2098-2.pdf`
- aprs.fi parser source — cross-reference for the messy real-world
  variants the spec doesn't fully pin down
