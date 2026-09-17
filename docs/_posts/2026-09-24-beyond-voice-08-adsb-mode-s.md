---
title: "Beyond Voice, Part 8: ADS-B — Mode S Squitters & CRC-24"
description: "How GopherTrunk decodes 1090 MHz aircraft transponders: a 2 Msps magnitude-domain preamble detector and PPM slicer, a CRC-24 that verifies extended squitters and doubles as the address on interrogation replies, the even/odd Compact Position Reporting pair, and a BEAST client that lets a running dump1090 feed the same aircraft table."
category: deep-dives
keywords: ads-b decoder go, mode s crc-24 0xfff409, cpr position decoding even odd, ppm preamble detection 2 msps, beast protocol client dump1090, df17 extended squitter parser, aircraft_log sqlite, 1090 mhz sdr decoder, adsb tracker pairing, gophertrunk ads-b
tags: [beyond-voice, ads-b, mode-s, cpr, aviation, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "Beyond Voice"
series_part: 8
---

*Part 8 of **Beyond Voice**, a 14-part deep dive into everything GopherTrunk
decodes that is not a trunked voice call — the AFSK signalling formats, paging,
APRS, ADS-B, AIS, DSC, LoRa, M17 and the amateur digital-voice family — and the
one eleven-place wiring pattern that carries each of them from a burst on the
air to a row in the web console.
[Part 7]({{ '/blog/deep-dives/beyond-voice-07-aprs-ax25-location/' | relative_url }})
put APRS beacons on the map. This part goes to 1090 MHz, where every airliner
overhead broadcasts identity, position and velocity several times a second —
and where GopherTrunk leaves FM behind for the first time: no discriminator,
no tones, just pulse energy on a 1 µs grid.*

> **TL;DR:** `internal/radio/adsb` is a pure-Go Mode S parser. `Decode` takes
> a 7- or 14-byte frame, reads the 5-bit downlink format, and runs `crc24`
> (polynomial `0xFFF409`): DF 11/17/18 must match the trailing 24 bits and
> carry the ICAO address in bytes 1–3, while DF 4/5/20/21 recover the address
> as `computed ^ stored`. Extended-squitter type codes dispatch to
> `parseIdentification` (6-bit callsign alphabet), `parseAirbornePosition`
> (17-bit CPR halves + 12-bit Q-bit altitude), `parseSurfacePosition` and
> `parseAirborneVelocity`. `Tracker.Update` pairs each ICAO's even and odd CPR
> halves within 10 s and `CPRDecodeGlobal` recovers lat/lon. Two front ends
> feed one `adsb.ProcessFrame`: `adsb/ppm` (2 Msps magnitude envelope,
> `detectPreamble` on pulses at 0, 1, 3.5, 4.5 µs) and `adsb/beast` (TCP
> `0x1A` frames from dump1090/readsb). Reports publish
> `events.KindAircraftReport` (`adsb.aircraft`) into `aircraft_log`,
> `GET /api/v1/adsb/aircraft` and the `/adsb` panel's map.

**Key takeaways**

- **The CRC is also the address.** On extended squitters the trailing 24 bits
  are a plain check; on addressed replies they are `CRC ⊕ ICAO`, so one
  computation either verifies a frame or names the aircraft.
- **A position takes two messages.** CPR sends latitude and longitude as
  17-bit fractions of alternating even and odd zone grids; only a pair within
  10 s resolves the zone, so a per-ICAO `Tracker` sits between parser and
  report.
- **Two sources, one decode path.** The native PPM receiver and the BEAST
  client both hand raw frames to `adsb.ProcessFrame`, so a frame off the air
  and one from dump1090 produce byte-identical rows.
- **Scope is stated in the code.** Q=0 Gillham altitudes, surface movement,
  locally-referenced CPR and status type codes are recognised, not decoded.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| Frame + CRC | DF from `frame[0]>>3`; `crc24` `0xFFF409`; verify or `^` for ICAO | `internal/radio/adsb/adsb.go` (`Decode`), `parse.go` (`crc24`) |
| ME payloads | TC 1–4 ident, 5–8 surface, 9–18/20–22 airborne, 19 velocity | `parse.go` (`parseIdentification`, `parseAirbornePosition`, `parseAirborneVelocity`) |
| CPR pairing | even+odd within 10 s → `CPRDecodeGlobal`; NL zone table | `tracker.go` (`Tracker.Update`), `cpr.go` (`cprNL`) |
| Shared path | decode → CRC gate → track → `storage.AircraftReport` | `report.go` (`ProcessFrame`, `BuildReport`) |
| Native receiver | 2 Msps magnitude², preamble at 0/2/7/9 samples, PPM slice | `adsb/ppm/receiver.go` (`detectPreamble`, `demodFrame`) |
| BEAST upstream | `0x1A <type> <ts 6B> <sig> <payload>`, un-stuff, reconnect | `adsb/beast/beast.go` (`ReadFrame`, `Client.Run`) |
| Bus → table → routes → panel | `adsb.aircraft` → `aircraft_log` → `/adsb/aircraft`, `/current` → `/adsb` | `events.KindAircraftReport`, `storage.AircraftLog`, `handleADSBAircraft` |
| Literal vectors | `8D4840D6…` KLM1023, `8D40621D…` pair, `8D485020…` velocity | `adsb_test.go`, `ppm/receiver_test.go`, `beast/beast_test.go` |

## In this post

- **Pulses, not tones** — the 2 Msps preamble detector and PPM slicer.
- **A CRC that verifies or names** — downlink formats and the address overlay.
- **What the extended squitter carries** — identification, position, velocity.
- **Two halves of a position** — CPR pairing and the tracker.
- **Two front ends, eleven places** — BEAST, the shared path, the panel.

## Pulses, not tones

[Mode S]({{ '/reference/mode-s/' | relative_url }}) replies are
[pulse-position modulated]({{ '/reference/pulse-position-modulation/' | relative_url }})
at 1 Mb/s: each 1 µs bit puts a pulse in its first half for a `1` or its
second half for a `0`, after an 8 µs preamble with pulses at 0, 1, 3.5 and
4.5 µs. `adsb/ppm` therefore never demodulates FM. `SampleRateHz` is a fixed
2 000 000 — "the minimum to resolve a PPM half-bit" — a faster SDR is
resampled down, a slower one refused at `New`. Each IQ sample becomes
`i*i + q*q`, and `scan` walks that magnitude buffer:

```go
// internal/radio/adsb/ppm/receiver.go (shape)
func detectPreamble(m []float32) bool {
    if !(m[0] > m[1] && m[1] < m[2] && m[2] > m[3] && m[3] < m[0] &&
        m[4] < m[0] && m[5] < m[0] && m[6] < m[0] &&
        m[7] > m[8] && m[8] < m[9] && m[9] > m[6]) {
        return false            // relative ordering of the four pulses
    }
    high := (m[0] + m[2] + m[7] + m[9]) / 6
    if m[5] >= high || m[6] >= high { return false }
    for j := 10; j < preambleSamples; j++ {   // quiet zone before the data
        if m[j] >= high { return false }
    }
    return true
}
```

That is the dump1090 magnitude-domain baseline: a relative-ordering test on
the pulse samples plus an absolute quiet check that rejects noise satisfying
the ordering test. After a hit, `demodFrame` reads the five-bit downlink
format, picks 56 or 112 bits from `longFrame(df)`, and packs them MSB-first
with `sliceBit`. A `frameSpan` of 240 samples is retained across chunks so a
split preamble still decodes (`TestChunkBoundarySplit`). The package doc is
candid: phase-corrected re-detection and 2.4 Msps are left for later; this
baseline locks on strong signals, which a filtered, amplified 1090 MHz chain
delivers ([ADS-B antenna]({{ '/reference/adsb-antenna/' | relative_url }})).

## A CRC that verifies or names

Every Mode S frame ends in 24 parity bits, and `Decode` treats them two ways
depending on the 5-bit downlink format in `frame[0] >> 3`:

```go
// internal/radio/adsb/adsb.go (shape)
computed := crc24(frame[:11])                 // polynomial 0xFFF409 over the message
stored := uint32(frame[11])<<16 | uint32(frame[12])<<8 | uint32(frame[13])
switch m.DF {
case DFExtendedSquitter, DFExtendedSquitterAlt, DFAllCallReply:   // 17, 18, 11
    m.CRCValid = computed == stored
    if m.CRCValid { m.ICAO = uint32(frame[1])<<16 | uint32(frame[2])<<8 | uint32(frame[3]) }
case DFAltitudeReply, DFIdentityReply, DFCommBAltitudeReply, DFCommBIdentityReply: // 4, 5, 20, 21
    m.ICAO = computed ^ stored                // address overlay: CRC always "matches"
    m.CRCValid = true
}
```

On the [ADS-B]({{ '/reference/ads-b/' | relative_url }}) extended squitter
(DF 17/18) the address is sent in the clear and the CRC stands alone. On
interrogation replies the transponder XORs its address into the parity, so a
receiver recovers the ICAO by XORing its own computation with the trailing
bits — with no way to verify correctness without an ICAO whitelist.
`ProcessFrame` gates on this: a CRC failure drops the frame unless it is a
DF 11 all-call, still useful for "this aircraft is in range".
`TestCRC24SelfConsistency` and `TestCRC24DetectsCorruption` pin the codec;
the literal vectors below pin it against the world.

## What the extended squitter carries

For DF 17/18 with a valid CRC, the 56-bit ME field's top five bits are the
type code, and `kindFromTC` maps ranges to `PayloadKind`s: **TC 1–4**
identification, **5–8** surface position, **9–18** and **20–22** airborne
position (barometric and GNSS altitude), **19** velocity; 28, 29 and 31
(status, target state, operational status) are labelled and passed through
with `RawHex` preserved.

`parseIdentification` pulls 48 bits into eight 6-bit characters through the
ICAO alphabet. `parseAirbornePosition` reads the 12-bit altitude, the CPR
format flag and the two 17-bit CPR fields; `decodeAltitude` handles Q=1
(`25·N − 1000` ft) and returns 0 for the Q=0 Gillham Gray code, deliberately
left undecoded. `parseAirborneVelocity` splits on subtype: 1 and 2 give
east–west and north–south components that become ground speed and track; 3
and 4 give airspeed and heading; the vertical rate is common.

The tests are the point. `TestDecodeIdentification` feeds
`8D4840D6202CC371C32CE0576098` → ICAO `4840D6`, callsign `KLM1023`;
`TestDecodeAirborneVelocity` → `485020`, ≈ 159 kn, track ≈ 183°, ≈ −832 fpm.
These are canonical dump1090 / mode-s.org samples — bytes not produced by
this code, the
[literal-vector discipline]({{ '/blog/deep-dives/from-spec-to-shipping-03-literal-vectors/' | relative_url }})
the SCCB bug taught.

## Two halves of a position

[Compact Position Reporting]({{ '/reference/compact-position-reporting/' | relative_url }})
is ADS-B's 34-bit answer to a 41-bit problem: the encoder divides the globe
into latitude zones — 60 for even messages, 59 for odd — and transmits a
17-bit fraction within a zone, alternating formats roughly every half second.
One message is ambiguous; a pair is not. `CPRDecodeGlobal` in `cpr.go`
computes the zone index `j = floor(59·lat_even − 60·lat_odd + 0.5)`,
reconstructs both latitudes, wraps values ≥ 270° into the southern
hemisphere, and refuses the pair when the two halves disagree on `cprNL`, the
number of longitude zones at that latitude — a closed form matching the
reference dump1090. Longitude follows against `NL` or `NL − 1`.

`Tracker` supplies the pairing. `Update` buffers the latest even and odd
halves per ICAO, and once both exist within `maxAgeNs` (10 s, DO-260B) it
calls `CPRDecodeGlobal` and returns a *copy* of the message with `Latitude`,
`Longitude` and `HasGlobalPosition` set. `TestDecodeAirbornePositionCPRPair`
pins the reference pair `8D40621D58C382D690C8AC2863A7` /
`8D40621D58C386435CC412692AD6` to ICAO `40621D`, 52.2572° N, 3.91937° E, and
`TestTrackerRejectsPairOlderThan10s` pins the window. Two honest limits: the
locally-referenced decode against a known receiver location is not
implemented; and `Prune`, which evicts idle ICAOs, is tested but has no
production caller — the BEAST client calls `Reset` on disconnect, the PPM
receiver never evicts.

<figure class="lab-figure">
<svg viewBox="0 0 680 250" width="680" height="250" role="img" aria-label="A 112-bit Mode S extended squitter — downlink format, 24-bit ICAO address, 56-bit ME field with a 5-bit type code, and CRC-24 — with the two CRC rules: verified directly on DF 17 and 18, XORed to recover the address on DF 4, 5, 20 and 21. Below, an even and an odd position message from one ICAO enter the tracker, which pairs them within ten seconds and calls CPRDecodeGlobal to produce latitude and longitude.">
  <text x="12" y="20" fill="currentColor" font-size="10" font-weight="bold">112-bit extended squitter (DF 17 / 18)</text>
  <rect x="12" y="30" width="40" height="30" fill="none" stroke="currentColor"/>
  <text x="32" y="49" text-anchor="middle" fill="currentColor" font-size="8">DF 5b</text>
  <rect x="52" y="30" width="110" height="30" fill="none" stroke="var(--accent)"/>
  <text x="107" y="49" text-anchor="middle" fill="var(--accent)" font-size="9">ICAO 24b</text>
  <rect x="162" y="30" width="330" height="30" fill="none" stroke="currentColor"/>
  <text x="327" y="44" text-anchor="middle" fill="currentColor" font-size="9">ME 56b — TC 5b + payload</text>
  <text x="327" y="55" text-anchor="middle" fill="var(--fg-muted)" font-size="8">1–4 ident · 5–8 surface · 9–18/20–22 airborne · 19 velocity</text>
  <rect x="492" y="30" width="176" height="30" fill="none" stroke="var(--accent)"/>
  <text x="580" y="49" text-anchor="middle" fill="var(--accent)" font-size="9">CRC-24 (0xFFF409)</text>
  <text x="12" y="84" fill="currentColor" font-size="9">DF 11/17/18: crc24(message) == stored → CRCValid, ICAO from bytes 1–3</text>
  <text x="12" y="98" fill="currentColor" font-size="9">DF 4/5/20/21: ICAO = crc24(message) ^ stored (address overlay)</text>
  <text x="12" y="134" fill="currentColor" font-size="10" font-weight="bold">CPR: two halves, one fix</text>
  <rect x="12" y="144" width="160" height="30" fill="none" stroke="currentColor"/>
  <text x="92" y="163" text-anchor="middle" fill="currentColor" font-size="9">even (F=0) · 60 lat zones</text>
  <rect x="12" y="186" width="160" height="30" fill="none" stroke="currentColor"/>
  <text x="92" y="205" text-anchor="middle" fill="currentColor" font-size="9">odd (F=1) · 59 lat zones</text>
  <line x1="172" y1="159" x2="230" y2="180" stroke="currentColor"/>
  <line x1="172" y1="201" x2="230" y2="180" stroke="currentColor"/>
  <rect x="230" y="165" width="150" height="30" fill="none" stroke="var(--accent)"/>
  <text x="305" y="179" text-anchor="middle" fill="var(--accent)" font-size="9">Tracker.Update per ICAO</text>
  <text x="305" y="190" text-anchor="middle" fill="var(--fg-muted)" font-size="8">both halves within 10 s</text>
  <line x1="380" y1="180" x2="420" y2="180" stroke="currentColor"/>
  <rect x="420" y="165" width="130" height="30" fill="none" stroke="currentColor"/>
  <text x="485" y="184" text-anchor="middle" fill="currentColor" font-size="9">CPRDecodeGlobal</text>
  <line x1="550" y1="180" x2="580" y2="180" stroke="currentColor"/>
  <text x="584" y="184" fill="var(--accent)" font-size="9">lat, lon → report</text>
  <text x="12" y="240" fill="var(--fg-muted)" font-size="9">reference pair 8D40621D58C382D690C8AC2863A7 + 8D40621D58C386435CC412692AD6 → 52.2572 N, 3.91937 E</text>
</svg>
<figcaption>One CRC computation either verifies an extended squitter or recovers the address from an interrogation reply; one position needs an even and an odd CPR message paired per aircraft inside the ten-second window.</figcaption>
</figure>

## Two front ends, eleven places

The second front end needs no radio of its own. `adsb/beast` is a TCP client
for the [Mode-S Beast]({{ '/reference/mode-s-beast/' | relative_url }})
format dump1090, readsb and BeastSplitter emit on port 30005:
`0x1A <type> <timestamp 6B> <signal 1B> <payload>`, type `0x32` a short frame,
`0x33` a long frame, `0x31` Mode-AC (skipped). `ReadFrame` hunts for a `0x1A`
*not* followed by another `0x1A` — a doubled pair is an escaped data byte —
and `readUnstuffed` collapses each pair (`TestReadFrameUnescapesStuffed1A`).
`Client.Run` reads under a 30 s deadline and on any drop calls
`tracker.Reset()` so stale CPR halves don't pair across a gap.
`TestClientPublishesDecodedFrames` runs a fake TCP server writing the three
reference frames and asserts the bus events. Both sources converge on one
function:

```go
// internal/radio/adsb/report.go
func ProcessFrame(frame []byte, tracker *Tracker, now time.Time) (storage.AircraftReport, bool) {
    m := Decode(frame)
    if !m.CRCValid && m.Kind != KindAllCall {
        return storage.AircraftReport{}, false
    }
    if tracker != nil {
        m, _ = tracker.Update(m, now.UnixNano())
    }
    return BuildReport(m, now), true
}
```

so a frame off the air and the same frame from dump1090 produce identical
rows. `BuildReport` fills `storage.AircraftReport` — ICAO, kind, callsign,
position/altitude guards, velocity, raw hex — and both front ends publish it
as `events.KindAircraftReport` (`"adsb.aircraft"`).

From there the [Part 1]({{ '/blog/deep-dives/beyond-voice-01-eleven-place-pattern/' | relative_url }})
walk: `storage.AircraftLog` writes `aircraft_log`; `handleADSBAircraft` serves
`GET /api/v1/adsb/aircraft` and its sibling `/aircraft/current` folds the
latest fields per ICAO over `?max_age_s=`; both return **503** without
storage. `web/src/panels/ADSB.tsx` polls every 5 s and turns positioned rows
into `MapPoint`s of kind `adsb` for the shared
[map]({{ '/blog/deep-dives/operator-cockpit-09-the-map/' | relative_url }}).
`config.ADSBConfig` carries `beast_upstreams` and `channels` (default
1 090 MHz); the configbuilder section, the `config.example.yaml` block,
`KnownUITabs["adsb"]`, the `App.panels.test.tsx` mock and `aircraft_log` in
the retention sweeper follow. One place is missing: the doctor preflight's
"needs storage.path" list omits adsb, so an ADS-B-only rig without
`storage.path` gets a 503 and no startup hint. On verification: the parser
and Beast path are pinned to independent reference vectors, but the PPM
receiver's `TestEndToEndDF17Decode` modulates ideal pulses, so by the repo's
[definition]({{ '/blog/deep-dives/from-spec-to-shipping-14-definition-of-verified/' | relative_url }})
the native RF front end is synthetic-verified only.

### How Mode S shaped the Go code

- **One decode path, two producers.** `ProcessFrame` owns CRC gating,
  tracking and report building; the receivers own only how bytes arrive.
- **Preserve what you don't decode.** Every `Message` carries `RawHex`, and
  status type codes are labelled, not dropped, so the row can be re-read later.
- **Pair, don't guess.** The tracker returns a copy with `HasGlobalPosition`
  set only when a real even/odd pair exists; unpaired halves stay raw.
- **Reset on discontinuity.** The Beast client clears the tracker on every
  reconnect, because a stale half across a gap is a wrong position, not a late
  one.

## Where this goes next

Aircraft announce themselves in pulses; ships announce themselves in GMSK.
[Part 9]({{ '/blog/deep-dives/beyond-voice-09-ais-gmsk-nrzi/' | relative_url }})
moves to 161.975 and 162.025 MHz, where AIS reuses NRZI and HDLC
bit-stuffing under a Gaussian-filtered FSK front end and packs vessel
identity and position into 6-bit ASCII.

## FAQ

**Does GopherTrunk decode ADS-B directly from an SDR or does it need dump1090?**
Either. `adsb.channels` pins one of GopherTrunk's own SDRs (sampling ≥ 2 Msps)
to 1090 MHz and runs the native PPM receiver; `adsb.beast_upstreams` connects
over TCP to a running dump1090 or readsb on port 30005. Both feed
`adsb.ProcessFrame`, so the rows, panel and map are identical.

**How does the Mode S CRC-24 recover the aircraft address?**
On DF 17/18 extended squitters the 24 parity bits are a plain check and the
ICAO sits in bytes 1–3. On DF 4/5/20/21 replies the transponder XORs its
address into the parity, so `Decode` computes the CRC over the message and
XORs it with the stored bits to recover the ICAO — unverifiable, as the code
notes, without an address whitelist.

**Why does an aircraft appear without a position at first?**
CPR sends positions as alternating even and odd zone fractions; one message is
ambiguous. `Tracker.Update` waits for both halves from the same ICAO within
10 s before `CPRDecodeGlobal` resolves lat/lon. Locally-referenced decoding
against a receiver location is a documented follow-up.

**What ADS-B message types does GopherTrunk decode?**
Identification (TC 1–4), airborne position with Q=1 altitude (TC 9–18,
20–22), surface position CPR fields (TC 5–8), velocity subtypes 1–4 (TC 19),
and DF 11 all-call addresses. Q=0 Gillham altitude, surface movement and
status codes 28/29/31 are kept as raw hex, not decoded.

**Why is my ADS-B panel empty when the log shows frames decoding?**
Reports surface only through SQLite. Without `storage.path`,
`GET /api/v1/adsb/aircraft` returns 503 "adsb subsystem not enabled" and the
panel stays empty — and unlike paging or APRS, ADS-B is not yet on the doctor
preflight's storage warning list, so set `storage.path` explicitly.

## Series navigation

**Part 8 of 14** · ←
[Part 7: APRS, AX.25 & the Location Layer]({{ '/blog/deep-dives/beyond-voice-07-aprs-ax25-location/' | relative_url }})
· Next →
[Part 9: AIS — GMSK, NRZI & Bit Stuffing]({{ '/blog/deep-dives/beyond-voice-09-ais-gmsk-nrzi/' | relative_url }})
