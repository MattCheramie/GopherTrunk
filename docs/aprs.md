---
layout: page
title: APRS / AX.25
description: Decoded APRS packet pipeline — events bus, SQLite log, REST endpoint, web panel
nav_group: Reference
---

# APRS / AX.25

GopherTrunk surfaces decoded **APRS** (Automatic Packet Reporting
System) packets through the same bus / SQLite / REST / web-panel
stack the POCSAG pager pipeline uses. APRS is the amateur-radio
metadata bus carrying position beacons, SAR coordination, weather,
status messages, and direct text messaging — useful alongside
trunked-voice scanning for emergency-comms-adjacent operators.

This page documents the **pipeline scaffolding**: what's wired,
what's persisted, what's queryable, and what the web panel
renders. The protocol parsers (AX.25 frame layout, APRS info-field
sub-types) and the DSP layer (1200 Bd Bell-202 AFSK → HDLC
de-stuff → frame delivery) are tracked separately under "What's
pending" below.

## What's wired

### Events bus
- `events.KindAPRSPacket` — published once per decoded frame.
  Payload is a `storage.APRSPacket` carrying the AX.25 envelope
  (src + dst + path) plus the APRS sub-type + decoded summary +
  optional lat/lon.

### Storage
- New SQLite table `aprs_log` (append-only migration alongside
  `pager_log`, `call_log`, etc.):
  - `received_at` (nanoseconds), `src`, `dst`, `path`, `type`,
    `body`, `latitude`, `longitude`, `raw_info`, `fcs_ok`
  - Indexes on `(received_at)` and `(src, received_at)`
- `storage.APRSLog` bus subscriber writes one row per
  `KindAPRSPacket` event. Mirrors the `PagerLog` lifecycle —
  subscribes at construction so events published before `Run()`
  begins aren't lost.

### REST
- `GET /api/v1/aprs/packets?limit=N` — most recent packets,
  newest first. Default 200, max 5000. Returns 503 when the
  storage layer isn't wired (daemon started without
  `storage.path`).

### Web panel
- `/aprs` — live list with columns:
  - Received (HH:MM:SS, daemon clock)
  - Src → Dst (with optional digipeater path on a second line)
  - Type (position / message / status / bulletin / weather /
    telemetry / object / mic-e / unknown)
  - Body (one-line summary)
  - Lat / Lon (for position-bearing types; em-dash otherwise)
- Polls every 5 s. Frames with `fcs_ok = false` highlight in
  yellow as a marginal-signal indicator.

## Protocol layer (`internal/radio/aprs`)

Pure-Go AX.25 frame parser + APRS info-field decoder that turn
bit-de-stuffed AX.25 frames into the `events.KindAPRSPacket`
payload the bus / log / REST / UI scaffolding above expects.

- **`internal/radio/aprs/ax25`** — AX.25 UI-frame parser:
  7-byte address packing with the spec's bit-shifted ASCII
  callsigns, up to 8 digipeater path entries, HDLC CRC-16-CCITT
  validation via the standard 0x8408 reflected-bit polynomial,
  conventional `W1AW-9` / `WIDE2-1*` display helpers. HBit
  handling is only meaningful for path entries; dst + src use
  that bit position for the C-bit (Command/Response indicator),
  which the APRS console convention ignores.
- **`internal/radio/aprs`** — info-field decoder. Recognised
  packet types (data-type indicator → decoded fields):
  - `!` / `=` — position without timestamp; messaging flag set
    on `=`
  - `/` / `@` — position with raw 7-char timestamp; messaging
    flag set on `@`
  - `:` — message (9-char trimmed addressee, body, optional
    `{seqno}`, `ack` / `rej` short-form) or bulletin
    (`BLN`-prefixed addressee → `Bulletin` struct)
  - `>` — status (free-text)
  - `;` / `_` / `T#` — type-tagged with payloads stashed for
    follow-up decoders
  - **Mic-E** (0x1C / 0x1D / "`" / "'") — full decode of the
    compressed mobile-tracker format that packs lat/lon, speed,
    course, symbol code and table, message code (M0 Off Duty
    through M7 Priority, plus Emergency and custom codes) into
    the 7-byte AX.25 destination address + 9 bytes of info
    field. Optional altitude trailer (`XXX}` base-91, meters
    above sea level) decoded into the same `MicE` payload.
    Surfaces lat/lon through the standard `Position` field so
    the storage / REST / web-panel layers pick it up without
    special-casing. Spec: APRS 1.0.1 §10.
  - anything else — `TypeUnknown`, raw bytes preserved
- Position parsing covers both hemispheres (N/S, E/W), the
  spec's "ambiguity space" convention (low-precision digits as
  space — treated as 0), and the standard `DDMM.hhH` /
  `DDDMM.hhH` hundredths-of-a-minute encoding.

## Bit-stream pipeline (`internal/radio/aprs/hdlc` + `receiver`)

- **`internal/radio/aprs/hdlc`** — bit-stream framer. Tracks the
  sliding-flag detector (HDLC's 0x7E delimiter), reverses the
  bit-stuffing (after 5 consecutive 1s, drop the next 0),
  resyncs on shared-flag packing, aborts on 7-or-more-1s runs
  (the HDLC abort sequence). Emits one fully-formed AX.25 frame
  body per (flag, ..., flag) sequence. Note: HDLC framing is
  the layer below the AX.25 frame parser — the bit-stuffing
  reversal happens here, not in `ax25`.
- **`internal/radio/aprs/receiver`** — bit-stream orchestrator.
  Threads bits through the framer, hands frame bodies to
  `ax25.Parse`, the info field to `aprs.Decode`, and publishes
  one `events.KindAPRSPacket` per successfully-decoded UI frame.
  Bus payload is `storage.APRSPacket` carrying the AX.25
  envelope, the APRS sub-type label, the decoded summary
  string, and (for position-bearing types) lat/lon. The
  receiver counts frames in / parsed / CRC-failed / emitted
  for future `/metrics` surfacing. Options expose
  `DropBadFCS` and `DropNonUI` toggles for operators who'd
  rather lose marginal frames than see them on the panel.

## DSP frontend (`internal/radio/aprs/afsk`)

The IQ-to-bits layer. One `afsk.Receiver` per configured APRS
channel; the daemon subscribes each to its assigned SDR's iqtap
broker. Pipeline:

```
IQ chunks (Fs Hz, complex64)
  → FM demod (internal/dsp/demod/fm.FM)
  → real resampler down to 9600 sps audio
  → FFSK tone discriminator (mark=1200 Hz, space=2200 Hz)
  → Mueller-Müller symbol-timing recovery (8 sps → 1 sample/symbol)
  → DC-tracking slicer (raw NRZI bit)
  → NRZI decode (transition = 0, no transition = 1)
  → aprs/receiver.Push(bit)
  → events.KindAPRSPacket on the bus
```

`Stats()` surfaces IQ-samples-seen + bits-emitted counters for
`/metrics`. The bit-stream layer's own `Stats()` (frames in /
parsed / CRC-failed / emitted) is reachable via `Inner()`.

## Configuration

```yaml
aprs:
  channels:
    - serial: "antenna-pi"
      frequency_hz: 144_390_000   # NA primary; EU R1 144.575, ISS 145.825
      drop_bad_fcs: false         # default false; publishes CRC-failed
                                  #  frames with FCSOK=false
      drop_non_ui: false          # default false; APRS only emits UI
```

A misconfigured entry surfaces as a startup warning and is
skipped — same non-essential treatment as `paging.pocsag`.

## What's pending

- **Real-fixture validation.** The synthetic IQ end-to-end test
  is currently `t.Skip`-ped (same as POCSAG's synth test). A
  captured `.wav` / `.cfile` from a real APRS channel under
  `samples/aprs/` would let the test suite replay through
  `internal/sdr/baseband` and assert the full chain.
- **Mic-E rich fields on the panel.** Speed, course, altitude,
  and the human-readable message code (`M3 Returning`, etc.)
  decode into the `MicE` struct but the storage row only carries
  the standard `latitude` / `longitude` columns. Surfacing the
  rich fields needs an `aprs_log` schema bump + REST DTO + a
  new column on `/aprs`.

## Live map

Position-bearing rows (uncompressed positions, Mic-E) light up
the shared Leaflet map at the top of `/aprs` — blue markers
plotted on the OpenStreetMap tile layer with the station
callsign on hover, camera auto-fits to the current rowset. The
same `<PositionMap>` component renders on `/ais`, `/dsc` (for
distress alerts that included a position), and `/adsb`.

## References

- APRS Protocol Reference 1.0.1 (1998-08-07) — `http://www.aprs.org/doc/APRS101.PDF`
- AX.25 Link Access Protocol v2.2 (TAPR / ARRL, 1998) — `http://www.ax25.net/AX25.2.2-Jul%2098-2.pdf`
- aprs.fi parser source — real-world variant cross-check
