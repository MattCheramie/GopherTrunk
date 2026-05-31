---
layout: page
title: ADS-B / Aviation
description: Decoded ADS-B aircraft pipeline — events bus, SQLite log, REST endpoint, web panel
nav_group: Reference
---

# ADS-B / Aviation

GopherTrunk decodes **Automatic Dependent Surveillance — Broadcast**
(ADS-B) messages aircraft transponders broadcast on 1090 MHz.
ADS-B is the ICAO-mandated cooperative aviation surveillance
protocol — every commercial passenger flight, most general-
aviation, and all military aircraft over US / EU airspace
continuously broadcasts: ICAO 24-bit address, callsign,
barometric or GNSS altitude, lat/lon, ground speed, vertical
rate, true heading. This is the same data feeding every public
flight-tracking service (FlightRadar24, FlightAware, adsb.lol,
OpenSky); GopherTrunk now has the protocol layer to decode it
end-to-end on the operator's own SDR.

This page documents the **end-to-end pipeline**: two ways frames
reach the bus — a native PPM DSP frontend on the operator's own
1090 MHz SDR, and a BEAST upstream that consumes a separately-
running dump1090 / readsb — plus what's persisted, queryable, and
rendered on the web panel.

## What's wired

### Events bus
- `events.KindAircraftReport` — published once per decoded
  Mode-S frame off the 1090 MHz channel. Payload is a
  `storage.AircraftReport` carrying the 24-bit ICAO address +
  message-kind-specific fields.

### Storage
- New SQLite table `aircraft_log` (append-only migration
  alongside `vessel_log`, `dsc_log`, `aprs_log`, `pager_log`):
  - `received_at`, `icao`, `icao_hex`, `kind`, `body`,
    `crc_valid`, `callsign`, `category`, `latitude`,
    `longitude`, `altitude_ft`, `has_position`, `has_altitude`,
    `ground_speed_kn`, `track_deg`, `vertical_rate_fpm`,
    `raw_hex`
  - Indexes on `(received_at)` and `(icao, received_at)`
- `storage.AircraftLog` bus subscriber writes one row per
  `KindAircraftReport` event.

### REST
- `GET /api/v1/adsb/aircraft?limit=N` — most recent reports,
  newest first. Default 200, max 5000. ADS-B is the highest-
  rate decoder (2-3 msg/s per aircraft on a busy channel) so
  tighter limits make sense for live rendering.

### Web panel
- `/adsb` — live list with columns:
  - Received (HH:MM:SS, daemon clock)
  - ICAO (6-char hex, the standard "tail identifier")
  - Kind (ident / airborne-pos / surface-pos / velocity / ...)
  - Callsign (for identification messages)
  - Lat / Lon (for position messages with a successful CPR
    global decode)
  - Alt (ft) (for airborne / surface position)
  - GS / Track (for velocity messages, subtype 1 / 2)
  - VR (fpm) (vertical rate, signed)
- CRC-failed frames highlight yellow.

## Protocol layer (`internal/radio/adsb`)

Pure-Go Mode-S parser. The bit-stream layer above (the native
PPM receiver or a BEAST upstream) hands it complete Mode-S
frames with the trailing 24-bit CRC included. Both sources go
through one shared path — `adsb.ProcessFrame(frame, tracker,
now)` decodes, gates on CRC, runs the CPR tracker, and returns
the `storage.AircraftReport` — so a frame off the air and the
same frame from dump1090 produce identical rows.

- **CRC-24 codec** (`parse.go`) — Mode-S CRC with polynomial
  0xFFF409. Verifies DF 11 / 17 / 18 frames directly (zero
  residue over `message || crc`); for DF 0 / 4 / 5 / 20 / 21
  the trailing 24 bits = `CRC XOR ICAO`, so the parser
  recovers the ICAO address by XORing the computed CRC.
- **DF dispatch** (`adsb.go`) — recognises every documented
  downlink format; fully decodes DF 17 / 18 extended squitter
  (the operator-visible majority) and tags the others with
  the raw payload preserved.
- **TC dispatch** for extended squitter ME payloads:
  - **TC 1-4**: Identification — 8-char callsign decoded from
    the 6-bit ICAO alphabet (A-Z, space, 0-9, with trailing
    spaces / underscores stripped).
  - **TC 5-8**: Surface position — CPR-encoded.
  - **TC 9-18, 20-22**: Airborne position — CPR-encoded
    lat/lon + 12-bit Q-bit altitude (Q=1 = 25 ft resolution).
  - **TC 19**: Airborne velocity — subtypes 1/2 = ground
    speed + track, subtypes 3/4 = air speed + heading; common
    vertical-rate field across all subtypes.
- **CPR decode** (`cpr.go`) — globally-unambiguous lat/lon
  recovery from an even + odd CPR pair (DO-260B
  §2.2.3.2.3.7). The NL (number of longitude zones) lookup
  table mirrors the dump1090 reference implementation. The
  locally-referenced decode (single message against a known
  receiver location) is the obvious follow-up once the daemon
  caches an operator-configured reference position.

Validated against the canonical dump1090 / mode-s.org reference
samples:
- Identification `8D4840D6202CC371C32CE0576098` → ICAO 4840D6,
  callsign "KLM1023".
- Airborne-position CPR pair `8D40621D58C382D690C8AC2863A7`
  + `8D40621D58C386435CC412692AD6` → ICAO 40621D, lat
  52.2572 N, lon 3.91937 E, alt 38,000 ft.
- Airborne velocity `8D485020994409940838175B284F` → ICAO
  485020, GS 159 kn, track ≈ 183°, VR -832 fpm.

Spec references:
- ICAO Annex 10 Volume IV (Aeronautical Telecommunications —
  Surveillance and Collision Avoidance Systems), Chapter 3
  (Mode-S).
- RTCA DO-260B / EUROCAE ED-102A — ADS-B 1090 ES Minimum
  Operational Performance Standards.
- `https://mode-s.org/decode/` — the de-facto reference parser
  documentation, cross-checked against real on-the-air
  payloads.

## BEAST upstream (`internal/radio/adsb/beast`)

GopherTrunk consumes Mode-S frames from any BEAST-protocol
upstream — the de-facto wire format dump1090, readsb,
BeastSplitter, and every commercial ADS-B hub speak. Operators
keep their existing 1090 MHz receive chain (RTL-SDR + filter +
LNA + dump1090) and point GopherTrunk at it over TCP. No
native 1 Msps PPM demod required.

```yaml
adsb:
  beast_upstreams:
    - addr: "127.0.0.1:30005"   # dump1090 default BEAST port
      name: "local-dump1090"
    - addr: "rooftop-pi:30005"  # pi-at-the-antenna setup
      name: "rooftop"
```

Frame layout (`0x1A <type> <timestamp 6B> <signal 1B>
<payload>`) — type codes:
- `0x31` = Mode-AC (skipped)
- `0x32` = Mode-S short (56 bits)
- `0x33` = Mode-S long (112 bits)

`0x1A` bytes inside the body are escaped as `0x1A 0x1A` for
sync framing; the client un-escapes transparently. Reconnects
with backoff on TCP drops (default 2 s); each disconnect
resets the embedded CPR tracker so stale even/odd halves
don't pair across a gap.

## Native PPM receiver (`internal/radio/adsb/ppm`)

The alternative to a BEAST upstream: pin one of GopherTrunk's own
SDRs to 1090 MHz and demodulate Mode-S straight off the air, no
external decoder. A 1090 MHz SAW filter + LNA ahead of the SDR is
strongly recommended — Mode-S is a weak, bursty signal.

```yaml
adsb:
  channels:
    - serial: "1090-antenna"      # SDR sampling >= 2 Msps
      frequency_hz: 1_090_000_000 # 1090 MHz (default when omitted)
```

Pipeline (one goroutine per channel, subscribing to that SDR's
iqtap broker):

```
IQ → resample to 2 Msps → magnitude² envelope → 8 µs preamble
   correlation (pulses at 0, 1, 3.5, 4.5 µs) → PPM bit slice
   (1 µs/bit: "1" = high-then-low, "0" = low-then-high) → DF
   frame-length (56 / 112 bit) → adsb.ProcessFrame
```

The detector and slicer follow the dump1090 magnitude-domain
baseline at a fixed 2 Msps; the receiver resamples to that rate
internally if the SDR runs faster. A magnitude carry buffer
spans chunk boundaries so a preamble split across two IQ chunks
still decodes. Phase-corrected re-detection and 2.4 Msps
operation are refinements left for later — the baseline locks
cleanly on the strong signals a filtered + amplified chain
delivers. Frames feed the same `adsb.ProcessFrame` path the
BEAST client uses, so storage, tracker, panel, and map are
shared.

## CPR pair tracker (`internal/radio/adsb.Tracker`)

ADS-B aircraft alternate between even-encoded (`CPRFormat=0`)
and odd-encoded (`CPRFormat=1`) position reports roughly every
0.5 s; recovering the global lat/lon needs both halves.
`Tracker.Update(msg, now)` buffers the most-recent half per
ICAO and calls `CPRDecodeGlobal` when both arrive within the
spec's 10 s window (DO-260B §2.2.3.2.3.7). Position rows show
globally-decoded lat/lon on the `/adsb` panel + map as soon as
the pair completes. `Prune()` evicts ICAOs that haven't
transmitted in > 10 s so the state map doesn't grow with every
aircraft ever seen.

## What's pending

- **Aircraft tracker.** An `aircraft_current` SQL view (or a
  separate live-state table indexed by ICAO) showing the
  latest known position / altitude / callsign per aircraft,
  joining identification + position + velocity rows over the
  last few minutes. Powers a "currently visible aircraft"
  panel distinct from the raw message log.
## Live map

Aircraft positions (once the per-ICAO CPR pairing lands) render
as purple markers on the shared Leaflet map at the top of
`/adsb` — callsign + altitude on hover, camera auto-fits to the
current rowset. The same `<PositionMap>` component renders on
`/aprs`, `/ais`, and `/dsc`.

## References

- ICAO Annex 10 Volume IV — Mode-S protocol.
- RTCA DO-260B / EUROCAE ED-102A — 1090 ES MOPS.
- `https://mode-s.org/decode/` — comprehensive worked
  examples + CPR walk-through.
- dump1090 / readsb — open-source reference implementations.
