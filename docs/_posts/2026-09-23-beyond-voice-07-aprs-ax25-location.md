---
title: "Beyond Voice, Part 7: APRS, AX.25 & the Location Layer"
description: "How GopherTrunk turns 1200-baud Bell 202 tones into APRS positions: Mueller-Müller timing on the FFSK discriminator, NRZI, an HDLC framer that reverses bit-stuffing, an AX.25 parser with a reflected CRC-16, the Mic-E format that hides latitude in the destination address, and the location tables that feed the map."
category: deep-dives
keywords: aprs decoder go, ax25 frame parser, hdlc bit stuffing decoder, bell 202 afsk sdr, mic-e decoding, aprs 144.390 sdr, nmea gga rmc parser, aprs_log sqlite, sdr aprs map, gophertrunk aprs
tags: [beyond-voice, aprs, ax25, hdlc, location, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "Beyond Voice"
series_part: 7
---

*Part 7 of **Beyond Voice**, a 14-part deep dive into everything GopherTrunk
decodes that is not a trunked voice call — the AFSK signalling formats, paging,
APRS, ADS-B, AIS, DSC, LoRa, M17 and the amateur digital-voice family — and the
one eleven-place wiring pattern that carries each of them from a burst on the
air to a row in the web console.
[Part 6]({{ '/blog/deep-dives/beyond-voice-06-pocsag-flex-paging/' | relative_url }})
decoded pages addressed to people. This part decodes packets addressed to
everyone: APRS, the amateur beacon network whose position reports light up the
shared map, and the first decoder here with a real link layer.*

> **TL;DR:** `internal/radio/aprs/afsk` is IQ-to-bits: `demod.NewFM()` →
> resample to 9600 sps → `demod.NewFFSK(9600, 1200, 2200)` →
> `sync.NewMuellerMuller(8, 0.05)` → a `1/64`-EMA slicer → `NRZIDecoder`
> (transition = 0). `aprs/hdlc.Framer` slides an LSB-first byte window for
> `0x7E`, drops the stuffed 0 after five 1s, aborts on seven, and emits bodies
> between 18 and 1024 bytes. `ax25.Parse` unpacks 7-byte addresses (callsigns
> shifted left one, SSID, `HBit` for path entries only) and checks the CRC-16
> (reflected `0x8408`, init `0xFFFF`, inverted) into `FCSOK`.
> `aprs.DecodeWithDst` dispatches on the data-type indicator and fully decodes
> **Mic-E**, whose latitude lives in the destination callsign. Packets publish
> `events.KindAPRSPacket` (`aprs.packet`) into `aprs_log`,
> `GET /api/v1/aprs/packets` and the `/aprs` panel's `<PositionMap>`.
> `location.ParseNMEA` (GGA/RMC) is the protocol-agnostic fix core — staged,
> not yet called; today's live `KindLocation` publisher is the DMR GPS Info LC.

**Key takeaways**

- **APRS needs closed-loop timing where POCSAG did not.** Bell 202 tones off
  an FM discriminator drift and overlap noise, so the front end runs a
  Mueller-Müller loop at 8 samples per bit instead of an open-loop integrator.
- **HDLC and AX.25 are two layers, and the code keeps them apart.** The framer
  owns flags and bit-stuffing; the parser owns addresses and the FCS; NRZI is
  undone in the AFSK stage.
- **Mic-E is a two-half codec.** Six destination characters carry latitude,
  message bits and hemisphere flags; the info field carries longitude, speed,
  course, symbol and an optional base-91 altitude.
- **Positions land on one map through two tables.** APRS rows carry lat/lon in
  `aprs_log`; trunked-radio GPS fixes go through `KindLocation` to
  `location_log`. The NMEA parser that would unify more protocols has no caller
  yet.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| IQ → bits | FM → 9600 sps → FFSK 1200/2200 → MM(8, 0.05) → slicer → NRZI | `internal/radio/aprs/afsk/receiver.go`, `nrzi.go` |
| Flags + de-stuffing | `0x7E` LSB-first window, drop 0 after five 1s, abort on seven | `aprs/hdlc/framer.go` (`Framer.Push`) |
| AX.25 frame | shifted callsigns, SSID, path `HBit`, UI = `0x03`/`0xF0`, FCS | `aprs/ax25/frame.go` (`Parse`, `computeFCS`) |
| Info field | DTI dispatch, `DDMM.hhH` positions, `:ADDRESSEE:` messages | `aprs/aprs.go` (`Decode`, `parseLatLon`) |
| Mic-E | dest chars → lat/msg bits; info → lon/speed/course/altitude | `aprs/mice.go` (`DecodeWithDst`, `parseMicEDest`) |
| Bus → table → route → panel | `aprs.packet` → `aprs_log` → `/api/v1/aprs/packets` → `/aprs` | `events.KindAPRSPacket`, `storage.APRSLog`, `handleAPRSPackets` |
| Trunked GPS fixes | `KindLocation` → `location_log` → `/api/v1/locations` | `trunking.Location`, `storage.LocationLog` |
| NMEA core (staged) | `GGA`/`RMC` → `Position`, XOR checksum | `internal/radio/location/nmea.go` (`ParseNMEA`) |

## In this post

- **Bell 202 to bits** — why this front end closes the timing loop.
- **HDLC: flags, stuffing and the abort** — one framer, LSB-first.
- **AX.25: callsigns shifted left one** — addresses, UI frames, the FCS.
- **The info field and Mic-E's two halves** — where the latitude hides.
- **The location layer and the map** — two tables, one renderer, one staged core.

## Bell 202 to bits

[APRS]({{ '/reference/aprs/' | relative_url }}) rides 1200-baud
[AFSK]({{ '/reference/afsk/' | relative_url }}) with the Bell 202 tones —
`MarkHz` 1200, `SpaceHz` 2200 — inside an FM voice channel, so the front end
in `afsk/receiver.go` opens like Parts 3 and 4: `demod.NewFM()`, a
`dsp.NewRealResampler` down to `AudioRateHz` = 1200 × 8 = 9600, and
`demod.NewFFSK(9600, 1200, 2200)`, whose `Discriminate` mixes the audio to the
tones' midpoint, low-passes and FM-discriminates — the
[Part 2]({{ '/blog/deep-dives/beyond-voice-02-afsk-ffsk-fundamentals/' | relative_url }})
primitive.

The difference from POCSAG is the next stage. Where the pager receiver
integrated eight samples per bit open-loop, this one runs
`dspsync.NewMuellerMuller(8, 0.05)` — closed-loop
[symbol-timing recovery]({{ '/reference/mueller-muller-timing-recovery/' | relative_url }})
— because Bell 202 "signals are messier than POCSAG direct-FSK at the slicer":
tones overlap the discriminator's noise floor, radios have variable AGC, and
the symbol clock "drifts a few hundred ppm". `mmGain` 0.05 settles "inside the
AX.25 preamble (~30 flags = 240 bits)".
Each recovered symbol is sliced against a `1/64` EMA and NRZI-decoded:

```go
// internal/radio/aprs/afsk/nrzi.go (shape)
var out byte = 1
if raw != d.last {   // tone transition on the wire = logical 0
    out = 0
}
d.last = raw
```

[NRZI]({{ '/reference/nrzi/' | relative_url }}) frees the demodulator from
knowing which tone is which: a 0 is a *change*, a 1 is *no change*, so an
inverted discriminator decodes identically — the problem POCSAG's syncer
solved by matching both polarities, solved here at the line-coding layer.

## HDLC: flags, stuffing and the abort

[HDLC]({{ '/reference/hdlc/' | relative_url }}) is the bit-stream-to-bytes
layer, and `aprs/hdlc/framer.go` is explicit about its boundaries: it handles
the `0x7E` delimiter, bit-stuffing and LSB-first byte order, *not* NRZI or the
CRC. `Framer.Push` slides a byte-wide shift register with each new bit landing
in bit 7, and a match on `FlagPattern` is always a frame boundary:

```go
// internal/radio/aprs/hdlc/framer.go (shape)
f.shiftReg = (f.shiftReg >> 1) | (uint8(bit) << 7)
if f.shiftReg == FlagPattern {          // 0x7E: close any frame, open the next
    out := f.closeFrame()
    f.openFrame()
    return out
}
if bit == 0 && f.onesCount == 5 {       // stuffed 0 after five 1s: drop it
    f.onesCount = 0
    return nil
}
/* accumulate LSB-first; seven 1s in a row = HDLC abort, body discarded */
```

Because the flag doubles as closing and opening delimiter, back-to-back frames
sharing one `0x7E` decode cleanly (`TestFramerSharedFlagBetweenAdjacentFrames`).
`closeFrame` discards the partial byte the closing flag left behind, drops
bodies shorter than `MinFrameBytes` (18) or longer than `MaxFrameBytes` (1024,
"generous headroom" over the 332-byte AX.25 maximum), and hands out a *copy*.
Seven consecutive 1s mid-frame is the HDLC abort; the framer latches `abort`
until the next flag (`TestFramerAbortsOnSevenOnes`).

## AX.25: callsigns shifted left one

[AX.25]({{ '/reference/ax25/' | relative_url }}) packs each address into
seven bytes: six callsign characters shifted left one bit (bit 0 is the HDLC
extension flag) and an SSID byte carrying the 4-bit SSID, two flag bits and
the end-of-address sentinel in bit 0. `ax25.Parse` walks destination, source
and up to `MaxPathEntries` 8 digipeaters until that sentinel —
`ErrBadAddress` if it never comes — then reads control, PID, info and the
trailing two bytes:

```go
// internal/radio/aprs/ax25/frame.go (shape)
c := raw[i] >> 1                         // callsign chars are shifted left one on the wire
ssid := (ssidByte >> 1) & 0x0F
hbit := isPathEntry && ssidByte&0x80 != 0 // "has been digipeated": path entries only
/* … */
calc := computeFCS(body[:infoEnd])       // reflected 0x8408, init 0xFFFF, inverted
wireFCS := uint16(fcsBytes[0]) | uint16(fcsBytes[1])<<8
return Frame{Dst: addresses[0], Src: addresses[1], Path: addresses[2:],
    Control: control, PID: pid, Info: info, FCSOK: calc == wireFCS}, nil
```

Two details are load-bearing. `HBit` is read only for path entries because
destination and source use that bit for the Command/Response indicator, "which
the APRS console convention ignores" — so `W1AW-9` never grows a spurious `*`,
while `WIDE2-1*` does once repeated. And `IsUI` is the APRS invariant: control
`0x03`, PID `0xF0` — the only frame type APRS puts on the air, which
`DropNonUI` can enforce.

The FCS is the standard
[CRC-16-CCITT]({{ '/reference/crc-16-ccitt/' | relative_url }}) in its
reflected HDLC form, compared to the little-endian wire value. A mismatch does
not drop the frame: `FCSOK` is a flag, and the default pipeline publishes
CRC-failed frames with `fcs_ok = false` so the panel can flag marginal traffic.
Honest note on evidence: `TestParseRoundTrip` uses the test's own
encoder and `TestParseDetectsCorruptedFCS` flips a byte — consistency and
sensitivity, not an on-air vector — and the chain's IQ test,
`TestReceiverDecodesSyntheticAPRSPacket`, is `t.Skip`-ped "deferred to
real-fixture follow-up". By the repo's
[definition of verified]({{ '/blog/deep-dives/from-spec-to-shipping-14-definition-of-verified/' | relative_url }}),
the APRS front end is synthetic-verified only.

## The info field and Mic-E's two halves

`aprs.Decode` never returns an error — "APRS is messy; we surface what we can
and pass through the rest" — and dispatches on the data-type indicator byte:
`!`/`=` positions without timestamp (messaging-capable on `=`), `/`/`@` with a
7-character timestamp, `:` messages and `BLN` bulletins, `>` status, and `;`,
`_`, `T#` tagged as object, weather and telemetry with raw bytes preserved.
`parseLatLon` reads the spec's `DDMM.hhH` / `DDDMM.hhH` encoding, treats
ambiguity spaces as 0, and signs south and west negative.

[Mic-E]({{ '/reference/aprs-mic-e/' | relative_url }}) — the compressed format
"almost every vehicle tracker and handheld uses" — is why the entry point is
`DecodeWithDst(info, dst)`: half the payload is in the AX.25 envelope.
`parseMicEDest` decodes each of the six destination characters through the
APRS 101 §10.5 table into one latitude digit and one indicator bit whose
meaning depends on position — message bits in characters 1–3, N/S in 4, the
+100° longitude offset in 5, W/E in 6. The info field yields longitude, speed
and course interleaved across three bytes, symbol, and an optional altitude:

```go
// internal/radio/aprs/mice.go (shape)
speed := sp*10 + dc/10           // sp, dc, se = info[4..6] − 28
course := (dc%10)*100 + se
if speed >= 800 { speed -= 800 } // printable-ASCII bias wraps
if course >= 400 { course -= 400 }
/* altitude: 3 base-91 chars before "}" → value − 10000 metres */
```

The independent vector is the spec's own: `TestParseMicEFullRoundTrip` encodes
the APRS101 §10 worked example — destination `S32UVT`, 33° 25.64′ N,
112° 09.18′ W, 20 knots at 251°, `M3 Returning` — and expects every field
back. The decoded Mic-E is also copied into the standard `Position`, so
storage and the panel pick up its lat/lon without knowing the shape existed.

<figure class="lab-figure">
<svg viewBox="0 0 680 250" width="680" height="250" role="img" aria-label="The APRS decode chain and the Mic-E split. Across the top, bits pass through the HDLC framer, the AX.25 parser and the APRS decoder. Below, a Mic-E packet's two halves — the destination callsign carrying latitude and message bits, and the info field carrying longitude, speed, course and symbol — feed one Position.">
  <rect x="12" y="22" width="100" height="30" fill="none" stroke="currentColor"/>
  <text x="62" y="41" text-anchor="middle" fill="currentColor" font-size="9">FM → FFSK → MM → NRZI</text>
  <line x1="112" y1="37" x2="130" y2="37" stroke="currentColor"/>
  <rect x="130" y="22" width="110" height="30" fill="none" stroke="currentColor"/>
  <text x="185" y="36" text-anchor="middle" fill="currentColor" font-size="9">hdlc.Framer</text>
  <text x="185" y="47" text-anchor="middle" fill="var(--fg-muted)" font-size="8">0x7E · de-stuff · abort</text>
  <line x1="240" y1="37" x2="258" y2="37" stroke="currentColor"/>
  <rect x="258" y="22" width="130" height="30" fill="none" stroke="currentColor"/>
  <text x="323" y="36" text-anchor="middle" fill="currentColor" font-size="9">ax25.Parse</text>
  <text x="323" y="47" text-anchor="middle" fill="var(--fg-muted)" font-size="8">dst src path ctl pid info FCS</text>
  <line x1="388" y1="37" x2="406" y2="37" stroke="currentColor"/>
  <rect x="406" y="22" width="150" height="30" fill="none" stroke="var(--accent)"/>
  <text x="481" y="41" text-anchor="middle" fill="var(--accent)" font-size="9">aprs.DecodeWithDst(info, dst)</text>
  <line x1="556" y1="37" x2="574" y2="37" stroke="currentColor"/>
  <rect x="574" y="22" width="94" height="30" fill="none" stroke="currentColor"/>
  <text x="621" y="41" text-anchor="middle" fill="currentColor" font-size="9">aprs.packet</text>
  <text x="12" y="96" fill="currentColor" font-size="10" font-weight="bold">Mic-E: one position, two halves</text>
  <rect x="12" y="108" width="250" height="34" fill="none" stroke="var(--accent)"/>
  <text x="137" y="122" text-anchor="middle" fill="var(--accent)" font-size="9">destination, 6 chars (S32UVT)</text>
  <text x="137" y="135" text-anchor="middle" fill="var(--fg-muted)" font-size="8">lat DDMMhh · 3 msg bits · N/S · +100° · W/E</text>
  <rect x="290" y="108" width="378" height="34" fill="none" stroke="currentColor"/>
  <text x="479" y="122" text-anchor="middle" fill="currentColor" font-size="9">info: DTI · lon 3B · speed/course 3B · symbol 2B · altitude}</text>
  <text x="479" y="135" text-anchor="middle" fill="var(--fg-muted)" font-size="8">bytes offset by 28; speed and course interleaved</text>
  <line x1="137" y1="142" x2="300" y2="184" stroke="var(--accent)"/>
  <line x1="479" y1="142" x2="380" y2="184" stroke="currentColor"/>
  <rect x="280" y="184" width="120" height="30" fill="none" stroke="var(--accent)"/>
  <text x="340" y="203" text-anchor="middle" fill="var(--accent)" font-size="9">Position{lat, lon}</text>
  <text x="340" y="236" text-anchor="middle" fill="var(--fg-muted)" font-size="9">→ aprs_log → /aprs map marker</text>
</svg>
<figcaption>Bits become frames in the HDLC framer, frames become an envelope and an info field in the AX.25 parser, and Mic-E is decoded from both halves at once into the same Position every other report uses.</figcaption>
</figure>

## The location layer and the map

`aprs/receiver.Receiver.Push` is the orchestrator: bits into the framer, a
body into `ax25.Parse`, info field and `frame.Dst.Callsign` into
`aprs.DecodeWithDst`, and one `storage.APRSPacket` per frame onto the bus,
with `Latitude`/`Longitude` whenever `pkt.Position` is set. `events.KindAPRSPacket`
is `"aprs.packet"`; `storage.APRSLog` writes `aprs_log`; `handleAPRSPackets`
serves `GET /api/v1/aprs/packets` (default 200) or **503** "set storage.path
in config"; `web/src/panels/APRS.tsx` polls every 5 s, badges failed frames
`fcs` and turns every row with a non-zero lat/lon into a `MapPoint` of kind
`aprs` for the shared `<PositionMap>`
[Operator Cockpit Part 9]({{ '/blog/deep-dives/operator-cockpit-09-the-map/' | relative_url }})
documents. The configbuilder's `aprs` section, `KnownUITabs["aprs"]`, the
`/aprs` route in `App.panels.test.tsx`, the `aprs:` block in
`config.example.yaml`, `aprs_log` in the retention sweeper and `aprs` in the
doctor preflight's storage list complete the eleven places; the row carries
only Mic-E's `latitude`/`longitude`, not its speed, course or altitude.

The map has a second source. Trunked-radio GPS reports do not pass through
APRS: the DMR voice chain parses the embedded **GPS Info LC**
(`dmr.ParseGPSInfo`), dedupes repeated fixes, logs `composer: dmr gps` and
publishes `events.KindLocation` with a `trunking.Location`, which
`storage.LocationLog` writes to `location_log` and `GET /api/v1/locations`
serves as `LocationFix` rows — the only live `KindLocation` publisher today.

Which brings the series to `internal/radio/location/nmea.go`. `ParseNMEA`
requires the leading `$`, verifies the `*hh` XOR checksum when present,
ignores the talker ID and dispatches on `GGA` or `RMC` (a `V` status returns
`ErrNoFix`); `Position.Valid` rejects `(0,0)` because "Null Island is in open
ocean". Its package doc names the protocols it serves — P25 Motorola Unit GPS,
Tait CCDI, MOTOTRBO profiles carrying "a verbatim NMEA-0183 sentence" — and it
is unit-tested. But grep for callers and there are none outside its tests: it
is the protocol-agnostic core the
[NMEA reference]({{ '/reference/nmea-0183/' | relative_url }}) describes,
staged for decoders not yet wired to it — "parser done, plumbing pending".

### How APRS shaped the Go code

- **Layer boundaries are package boundaries.** `afsk` undoes NRZI, `hdlc`
  undoes stuffing, `ax25` checks the FCS, `aprs` reads the payload — each
  documents what it does *not* do.
- **Never error on messy payloads.** `Decode` returns `TypeUnknown` with
  `Raw` preserved; the panel shows what the air carried.
- **Surface through the common field.** Mic-E copies its fix into `Position`
  so storage, REST and the map needed no Mic-E-specific code.
- **Flag, don't drop.** `FCSOK` rides to the row; dropping is the operator's
  `drop_bad_fcs` choice.

## Where this goes next

APRS positions arrive a few per minute on one VHF channel.
[Part 8]({{ '/blog/deep-dives/beyond-voice-08-adsb-mode-s/' | relative_url }})
goes to 1090 MHz, where aircraft squitter positions several times a second in
112-bit pulse-position-modulated bursts — a 2 Msps preamble detector, a CRC-24
that doubles as an address, the even/odd CPR pair, and the BEAST client that
lets a running dump1090 feed the same aircraft table.

## FAQ

**How does GopherTrunk decode APRS packets from an SDR?**
One `afsk.Receiver` per `aprs.channels` entry FM-demodulates the channel,
resamples to 9600 sps, discriminates the 1200/2200 Hz tones, recovers timing
with a Mueller-Müller loop, slices and NRZI-decodes, then runs the HDLC
framer, AX.25 parser and APRS info-field decoder. Packets publish as
`aprs.packet` and land in `aprs_log`.

**Does GopherTrunk decode Mic-E APRS positions?**
Yes: `DecodeWithDst` reads latitude, message bits and hemisphere flags from
the six-character AX.25 destination and longitude, speed, course, symbol and
optional altitude from the info field, verified against the APRS101 §10 worked
example. Only lat/lon reach the `aprs_log` row today; the rich fields stay in
the `MicE` struct.

**Why does the APRS panel show frames with a yellow fcs badge?**
Because the receiver publishes CRC-failed frames by default with
`fcs_ok = false`, so marginal traffic is visible rather than silently lost.
Set `drop_bad_fcs: true` to discard them, and `drop_non_ui: true` to drop the
rare non-UI AX.25 frames APRS never sends.

**Is the APRS decoder verified against real on-air captures?**
Not yet. The protocol layers are unit-tested, Mic-E against the spec's worked
example, but the synthetic IQ end-to-end test is skipped pending a real
fixture under `samples/aprs/`. By the project's definition it is
synthetic-verified only.

**How do trunked-radio GPS positions reach the same map as APRS?**
Through a separate path: the DMR voice chain parses the embedded GPS Info LC
and publishes `KindLocation`, which `storage.LocationLog` writes to
`location_log` and `GET /api/v1/locations` serves. The `location.ParseNMEA`
core is written and tested but has no production caller yet.

## Series navigation

**Part 7 of 14** · ←
[Part 6: POCSAG & FLEX — Paging Networks]({{ '/blog/deep-dives/beyond-voice-06-pocsag-flex-paging/' | relative_url }})
· Next →
[Part 8: ADS-B — Mode S Squitters & CRC-24]({{ '/blog/deep-dives/beyond-voice-08-adsb-mode-s/' | relative_url }})
