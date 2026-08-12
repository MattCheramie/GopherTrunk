---
layout: page
title: AIS / Marine
description: Decoded AIS vessel-tracking pipeline — events bus, SQLite log, REST endpoint, web panel
nav_group: Reference
---

# AIS / Marine

GopherTrunk decodes the **Automatic Identification System** messages
commercial vessels broadcast on marine VHF channels 87B / 88B
(161.975 / 162.025 MHz). AIS is the "transponder for ships" — every
SOLAS-covered vessel (passenger ships, tankers, cargo > 300 GT)
transmits Class A position reports continuously, and recreational
vessels increasingly carry Class B units. The data is useful for:
marine-coast monitoring, traffic deconfliction inside shipping
lanes, search-and-rescue coordination, port arrival / departure
tracking, and as a free positional ground-truth for receivers that
want a known-good wide-area data feed.

This page documents the **pipeline scaffolding**: what's wired,
what's persisted, what's queryable, and what the web panel
renders. The DSP layer (9600 Bd GMSK demod → HDLC framer →
message parser) is tracked separately under "What's pending"
below.

## What's wired

### Events bus
- `events.KindAISMessage` — published once per decoded AIS message
  off the air. Payload is a `storage.AISMessage` carrying MMSI +
  message type + (for position-bearing types) lat/lon/COG/SOG/
  heading + (for static types) vessel name / callsign /
  destination / ship type / IMO.

### Storage
- New SQLite table `vessel_log` (append-only migration alongside
  `aprs_log`, `pager_log`, etc.):
  - `received_at` (nanoseconds), `mmsi`, `type`, `body`,
    `latitude`, `longitude`, `sog`, `cog`, `heading`,
    `has_position`, `vessel_name`, `callsign`, `destination`,
    `ship_type`, `imo`, `raw_hex`, `fcs_ok`
  - Indexes on `(received_at)` and `(mmsi, received_at)`
- `storage.VesselLog` bus subscriber writes one row per
  `KindAISMessage` event. Mirrors the `APRSLog` lifecycle —
  subscribes at construction so events published before `Run()`
  begins aren't lost.

### REST
- `GET /api/v1/ais/vessels?limit=N` — most recent messages,
  newest first. Default 200, max 5000. Returns 503 when the
  storage layer isn't wired (daemon started without
  `storage.path`).

### Web panel
- `/ais` — live list with columns:
  - Received (HH:MM:SS, daemon clock)
  - MMSI (with vessel name on a second line for static types)
  - Type (position-a / position-b / static-voyage / static-b /
    base-station / aid-to-nav / ...)
  - Body (one-line summary)
  - Lat / Lon (em-dash for static-only messages)
  - SOG / COG (knots / degrees, em-dash when not applicable)
- Polls every 5 s. Messages with `fcs_ok = false` highlight in
  yellow as a marginal-signal indicator.

## Protocol layer (`internal/radio/ais`)

Pure-Go AIS message parser turning the bit-stream payload into
the `events.KindAISMessage` payload the bus / log / REST / UI
scaffolding above expects.

- **6-bit ASCII** (M.1371-5 Table 47) — packed-text fields
  (vessel name, call-sign, destination) decode via the standard
  64-entry character table.
- **Bit-field readers** — `readBitsUint` and `readBitsInt`
  pull MSB-first signed / unsigned integers from the unpacked
  bit stream. Signed-field sign-extension handles the spec's
  signed lat/lon fields (28-bit longitude, 27-bit latitude,
  resolution 1/600000 minute = ~0.18 m).
- **Type dispatch** (M.1371-5 §3.3) — bytes [0..5] are the
  6-bit message-type tag, [6..7] the repeat indicator,
  [8..37] the 30-bit MMSI. Type-specific layouts:
  - `1`, `2`, `3` — Class A position report (nav status, SOG,
    lat/lon, COG, heading, timestamp). 168 bits.
  - `4` — base-station report (UTC + lat/lon). 168 bits.
  - `5` — static + voyage data (IMO, call-sign, vessel name,
    ship type, dimensions, ETA, draught, destination). 424
    bits.
  - `18` — Class B position report. 168 bits.
  - `19` — Class B extended position report (Class B with
    vessel name + ship type appended). 312 bits.
  - `24` — Class B static data report — Part A (vessel name)
    or Part B (call-sign + ship type + dimensions).
- **"Not available" sentinel handling** — lat 91° / lon 181°
  collapses `HasPosition` to false; SOG / COG `not available`
  values pass through as their raw spec sentinels.

Spec references:
- ITU-R M.1371-5 (Recommendation, 2014) — bit-by-bit layout
  for every message type.
- `https://gpsd.gitlab.io/gpsd/AIVDM.html` — the de-facto
  reference parser docs (gpsd's AIVDM decoder), cross-checked
  against real on-the-air payloads.

## Bit-stream pipeline (`internal/radio/ais/receiver`)

The orchestrator. Threads bits through the HDLC framer (reused
from `aprs/hdlc`), validates the trailing CRC-CCITT (same
polynomial 0x8408 / init 0xFFFF / final XOR 0xFFFF AX.25 uses —
AIS inherits the HDLC link-layer conventions verbatim per ITU-R
M.1371-5 §4.2), unpacks the payload bytes into the MSB-first bit
slice the AIS message parser expects, and publishes one
`events.KindAISMessage` per successfully-parsed message.

- **`internal/radio/ais/receiver`** — `Push(bit byte)` consumes
  one LSB-first wire bit. Bus payload is `storage.AISMessage`
  carrying MMSI + type + (for position-bearing types) lat/lon +
  COG / SOG + heading + (for static types) vessel name + callsign
  + destination + ship-type + IMO. Options expose `DropBadFCS`
  (silent-drop CRC-failed messages; default false) and
  `DropNonPosition` (silent-drop static / base-station chatter;
  default false). The receiver counts frames in / parsed /
  CRC-failed / emitted / too-short for future `/metrics` surfacing.

## DSP frontend (`internal/radio/ais/gmsk`)

The IQ-to-bits layer. One `gmsk.Receiver` per configured AIS
channel; the daemon subscribes each to its assigned SDR's iqtap
broker. Pipeline:

```
IQ chunks (Fs Hz, complex64)
  → FM demod (internal/dsp/demod/fm.FM)
  → real resampler down to 76,800 sps audio
  → GFSK matched filter (BT = 0.4, span 4 symbols)
  → Mueller-Müller symbol-timing recovery (8 sps → 1 sample/symbol)
  → zero-threshold slicer (raw NRZI bit)
  → NRZI decode (transition = 0, no transition = 1)
  → ais/receiver.Push(bit)
  → events.KindAISMessage on the bus
```

`Stats()` surfaces IQ-samples-seen + bits-emitted counters for
`/metrics`. The bit-stream layer's own `Stats()` (frames in /
parsed / CRC-failed / emitted / too-short) is reachable via
`Inner()`.

## Configuration

```yaml
ais:
  channels:
    - serial: "marine-antenna"
      frequency_hz: 161_975_000      # 87B (or 162_025_000 = 88B)
      drop_bad_fcs: false            # default false; publishes CRC-failed
                                     #  messages with FCSOK=false
      drop_non_position: false       # default false; static-data and
                                     #  base-station messages still surface
```

A misconfigured entry surfaces as a startup warning and is
skipped — same non-essential treatment as `paging.pocsag` and
`aprs.channels`.

## What's pending

- **Real-fixture validation.** The synthetic IQ end-to-end test
  for the GMSK frontend is deferred to the same follow-up that
  drops captured `samples/ais/` recordings; the receiver code
  is exercised end-to-end by the bit-stream synthetic in
  `internal/radio/ais/receiver/receiver_test.go` (AIVDM type-1
  sample → buildAISFrame → HDLC wrap → Receiver.Push → bus event
  with expected MMSI + lat/lon).
- **Multi-slot frame reassembly.** Several message types span
  two AIS slots when transmitted (type 5 + type 19 + type 26).
  The current parser handles the single-slot variants; the
  multi-slot path needs a per-MMSI buffer plus the channel-A /
  channel-B re-orderer.

## Live map

Position-bearing rows (Class A / B position reports, base
station, SAR aircraft) light up the shared Leaflet map at the
top of `/ais` — cyan markers plotted on the OpenStreetMap tile
layer with vessel name (or MMSI) on hover, camera auto-fits
to the current rowset. The same `<PositionMap>` component
renders on `/aprs`, `/dsc`, and `/adsb`.

## References

- ITU-R M.1371-5 (2014) "Technical characteristics for an
  automatic identification system" — `https://www.itu.int/rec/R-REC-M.1371`
- AIVDM/AIVDO protocol decoding — `https://gpsd.gitlab.io/gpsd/AIVDM.html`
