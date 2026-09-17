---
title: "Beyond Voice, Part 9: AIS — GMSK, NRZI & Bit Stuffing"
description: How GopherTrunk decodes marine AIS in pure Go — a 9600-baud GMSK front end behind an FM discriminator, the NRZI line code where the bit is the transition, the HDLC framer borrowed from AX.25 with its bit-stuffing rule, six-bit ASCII, and the message types it really parses.
category: deep-dives
keywords: ais decoder sdr, gmsk demodulation, nrzi decoding, hdlc bit stuffing, ais message types, mmsi decode, six bit ascii, gophertrunk ais
tags: [beyond-voice, ais, gmsk, nrzi, hdlc, marine, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "Beyond Voice"
series_part: 9
---

*Part 9 of **Beyond Voice**, a 14-part deep dive into everything GopherTrunk
decodes that is not a trunked voice call — the AFSK signalling formats,
paging, APRS, ADS-B, AIS, DSC, LoRa, M17 and the amateur digital-voice
family — and the one eleven-place wiring pattern that carries each of them
from a burst on the air to a row in the web console.
[Part 8]({{ '/blog/deep-dives/beyond-voice-08-adsb-mode-s/' | relative_url }})
left the sky. This part goes to sea: AIS, the ship transponder, with a
physical layer nothing like ADS-B's — a constant-envelope GMSK carrier at
9600 baud, a line code that hides the data in transitions, and a framer
GopherTrunk did not have to write twice because APRS had already paid for
it.*

> **TL;DR:** AIS rides marine VHF channels 87B / 88B (161.975 / 162.025 MHz)
> at 9600 Bd GMSK, BT = 0.4. `internal/radio/ais/gmsk` is the IQ-to-bits
> front end: `demod.FM` → resample to `AudioRateHz` 76,800 (8 samples per
> bit) → `demod.GFSK` matched filter → `sync.MuellerMuller` (`mmGain` 0.05)
> → a **zero-threshold slicer** → `NRZIDecoder` (transition = 0, hold = 1). `internal/radio/ais/receiver`
> owns bits-to-message: the AX.25 `hdlc.Framer` (0x7E flags, drop the 0
> after five 1s), `MinPayloadBytes` = 23, the CRC-CCITT FCS (0x8408 /
> 0xFFFF), then `ais.Decode` reads MSB-first fields and six-bit ASCII. Types
> 1/2/3, 4, 5, 18, 19 and 24 are field-parsed; the rest are labelled and
> kept as `RawHex`. Output is `events.KindAISMessage` → `storage.VesselLog`
> → `GET /api/v1/ais/vessels` → the `/ais` panel. Synthetic-verified only.

**Key takeaways**

- **GMSK decodes through the same FM discriminator as everything else.** A
  Gaussian matched filter on the discriminator output is enough at BT 0.4,
  and the slicer needs no bias tracker — the eye is symmetric about zero.
- **NRZI makes polarity irrelevant.** The decoder emits 1 when consecutive
  raw bits match and 0 when they differ; an inverted discriminator changes
  nothing.
- **Bit stuffing is what makes `0x7E` unique.** A 0 after every five 1s means
  six 1s can only be a flag; `hdlc.Framer` slides an 8-bit window at every
  bit and resyncs itself.
- **The parser decodes the operator-visible majority and says so.** Position
  reports and static data are field-parsed; SAR, aids-to-navigation and
  long-range types are labelled, not parsed.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| Front end + NRZI | FM → resample to 76,800 → GFSK matched filter → MM → slicer → transition = 0 / hold = 1 | `internal/radio/ais/gmsk/receiver.go`, `nrzi.go` (`processChunk`, `NRZIDecoder`) |
| Framing | 0x7E flags, five-ones de-stuff, seven-ones abort, LSB-first | `internal/radio/aprs/hdlc` (`Framer.Push`, `MinFrameBytes` 18) |
| CRC + dispatch | CRC-CCITT FCS, `MinPayloadBytes` 23, publish with `FCSOK` | `internal/radio/ais/receiver/receiver.go` (`Push`, `crc16CCITT`) |
| Message parser | MSB-first fields, six-bit ASCII, types 1–5/18/19/24 | `internal/radio/ais/ais.go`, `sixbit.go` (`Decode`, `TypeString`) |
| Bus → storage → REST → panel | `KindAISMessage` → `vessel_log` → `/api/v1/ais/vessels` → `/ais` | `internal/storage/vessellog.go`, `internal/api/handlers_ais.go`, `web/src/panels/AIS.tsx` |

## In this post

- **Two channels, one transponder** — what AIS is on the air.
- **GMSK through a discriminator** — the `gmsk.Receiver` chain and the
  constant-zero slicer.
- **NRZI, the bit in the transition** — the decoder and polarity.
- **HDLC and bit stuffing** — the borrowed framer, the flag and the CRC.
- **Six-bit ASCII and the types really decoded** — layouts and what stays
  raw.
- **From bus event to the `/ais` panel** — the eleven places.

## Two channels, one transponder for ships

AIS is the maritime cousin of Part 8's squitter: every SOLAS vessel
broadcasts identity, position, course and speed. The
[reference page]({{ '/reference/ais/' | relative_url }}) has the system view
— two 25 kHz channels at 161.975 and 162.025 MHz, 2250 slots per minute per
channel, Class A reserving slots by SOTDMA, Class B by carrier sense — and
[marine VHF]({{ '/reference/marine-vhf/' | relative_url }}) places it among
the voice channels.

GopherTrunk does **no slot bookkeeping**: the self-organising
[TDMA]({{ '/reference/tdma/' | relative_url }}) is the transmitters' problem,
and a receiver that frames bursts as they arrive gets every message the slot
map would have given it. It needs a physical layer (burst → bits) and a link
layer (bits → frame body). The chain landed across
issues #427 and #428; `docs/ais.md` is its
[operator page]({{ '/ais.html' | relative_url }}).

## GMSK through a discriminator: the front end

[GMSK]({{ '/reference/gmsk/' | relative_url }}) is continuous-phase FSK with
a Gaussian pre-modulation filter — constant envelope, compact spectrum. A
receiver can track it coherently or run it through a discriminator and a
matched filter; GopherTrunk takes the second road, since it already owns
every stage from
[SDR Internals Part 6]({{ '/blog/deep-dives/sdr-internals-06-demodulation/' | relative_url }})
onward:

```go
// internal/radio/ais/gmsk/receiver.go (shape) — AudioRateHz = 9600 × 8
func (r *Receiver) processChunk(chunk []complex64) {
    r.demodBuf = r.fm.Process(r.demodBuf, chunk)            // demod.FM
    r.rsmpBuf  = r.rsmp.Process(r.rsmpBuf, r.demodBuf)      // L/M to 76,800
    r.gfskBuf  = r.gfsk.MatchedFilter(r.gfskBuf, r.rsmpBuf) // demod.GFSK
    r.symBuf   = r.mm.Process(r.symBuf, r.gfskBuf)          // 1 sample/bit
    for _, s := range r.symBuf { r.feedSymbol(s) }
}
```

`New` reduces the input rate and 76,800 by their GCD for the
`dsp.RealResampler`, so any SDR rate works. The
[GFSK]({{ '/reference/gfsk/' | relative_url }}) matched filter is the shared
`demod.GFSK`, parameterised to AIS's BT 0.4 with a four-symbol span. Timing
recovery is the
[Mueller-Müller]({{ '/reference/mueller-muller-timing-recovery/' | relative_url }})
loop of
[SDR Internals Part 7]({{ '/blog/deep-dives/sdr-internals-07-symbol-timing-sync-recovery/' | relative_url }})
at 8 samples per bit, `mmGain` 0.05 — sized to settle inside the 2.5 ms
training sequence yet not be yanked by a 168-bit payload.

The slicer is the deliberate difference from the
[AFSK family]({{ '/blog/deep-dives/beyond-voice-02-afsk-ffsk-fundamentals/' | relative_url }}).
`feedSymbol`
slices at a **fixed zero** — "GMSK is symmetric around DC so the slicer
threshold is fixed at zero — no DC-tracking needed" — and hands the raw bit
to the NRZI decoder. That constant is the lesson
[FleetSync]({{ '/blog/deep-dives/beyond-voice-04-fleetsync-wav-that-lied/' | relative_url }})
learned the hard way: a bias tracker drifting toward a long run of one tone flips the isolated
bits after it. AIS never gives a tracker the chance, because the line code
below guarantees an edge at least every six bits.

## NRZI: the bit is the transition

Between the bit stuffer and the modulator an AIS transmitter applies
[NRZI]({{ '/reference/nrzi/' | relative_url }}), the
[AX.25]({{ '/reference/ax25/' | relative_url }}) convention: a 0 causes a
tone transition, a 1 leaves the tone alone. The decoder never asks "which
tone?" — only "did it change?":

```go
// internal/radio/ais/gmsk/nrzi.go (shape)
func (d *NRZIDecoder) Decode(raw byte) byte {
    if !d.primed { d.last, d.primed = raw, true; return 1 } // placeholder
    var out byte = 1             // no transition → 1
    if raw != d.last { out = 0 } // transition → 0
    d.last = raw
    return out
}
```

Two consequences do real work. **Polarity is irrelevant**: if the
discriminator's sign is inverted, every raw bit flips but "did it change"
does not, so the slicer needs no dual-polarity hunt. And the **placeholder
bit is harmless**: the first `Decode` has nothing to compare against, emits
a 1 and seeds its state; the framer is hunting for a flag anyway.
`TestNRZIDecoderInitialPlaceholder` and `TestNRZIDecoderTransitionVsHold`
pin both; `Reset` exists for a squelch close or retune. The type is a
declared copy of `aprs/afsk.NRZIDecoder` — fourteen lines of duplication is
cheaper than the coupling.

## HDLC and bit stuffing: the framer AIS borrows

AIS inherits the HDLC link layer verbatim (ITU-R M.1371-5 §4.2), so
`ais/receiver` imports `internal/radio/aprs/hdlc` rather than owning a
framer. [HDLC]({{ '/reference/hdlc/' | relative_url }}) delimits a frame with
the flag `0x7E` — `01111110` — and keeps that pattern out of the data by
**stuffing** a 0 after every run of five 1s. Six consecutive 1s can
therefore only be a flag; seven or more are the abort sequence:

```go
// internal/radio/aprs/hdlc/framer.go (shape)
if f.shiftReg == FlagPattern { /* 0x7E at ANY bit offset → close/open */ }
if bit == 0 && f.onesCount == 5 { f.onesCount = 0; return nil } // stuffed 0
if f.onesCount >= 7 { f.abort = true }                          // HDLC abort
```

The sliding 8-bit window is the whole synchronisation story: a flag is
checked at every bit position, so a misaligned stream or a noise burst costs
nothing but the frame it landed in; `closeFrame` drops bodies under
`MinFrameBytes` (18). `TestFramerBitDestuffsCorrectly`,
`TestFramerResyncsAcrossNoise` and `TestFramerAbortsOnSevenOnes` pin the
rules, and the same code already framed every APRS packet in
[Part 7]({{ '/blog/deep-dives/beyond-voice-07-aprs-ax25-location/' | relative_url }}).

The AIS receiver adds its own gate: 168 bits plus a 16-bit FCS is 23 bytes,
so `MinPayloadBytes` = 23 sits above the framer's 18 and shorter bodies
count as `FramesTooShort`. The last two bytes are the little-endian FCS, and
`crc16CCITT` is the AX.25 routine byte for byte —
[CRC-CCITT]({{ '/reference/crc-16-ccitt/' | relative_url }}), reflected
polynomial 0x8408, init 0xFFFF, final XOR 0xFFFF;
`TestCRC16CCITTSelfConsistency` anchors it with the 0xF0B8 magic residue. A
failed check does **not** drop the frame by default — it publishes with
`FCSOK=false` so the panel can flag it — and `drop_bad_fcs: true` flips that
(`TestReceiverDropBadFCSOption`).

<figure class="lab-figure">
<svg viewBox="0 0 680 250" width="680" height="250" role="img" aria-label="Four rows: the sliced tone level; NRZI output writing 0 under each transition and 1 under each hold; a de-stuff row crossing out the 0 after five 1s; and the flag 01111110 closing the frame.">
  <text x="8" y="30" fill="var(--fg-muted)" font-size="9">sliced tone</text>
  <path d="M90,36 H140 V20 H240 V36 H390 V20 H590 V36 H640" fill="none" stroke="currentColor"/>
  <text x="8" y="72" fill="var(--fg-muted)" font-size="9">NRZI out</text>
  <g fill="currentColor" font-size="11" text-anchor="middle">
    <text x="165" y="76">0</text><text x="215" y="76">1</text><text x="265" y="76">0</text>
    <text x="315" y="76">1</text><text x="365" y="76">1</text><text x="415" y="76">0</text>
    <text x="465" y="76">1</text><text x="515" y="76">1</text><text x="565" y="76">1</text><text x="615" y="76">0</text>
  </g>
  <text x="8" y="126" fill="var(--fg-muted)" font-size="9">de-stuff</text>
  <g fill="currentColor" font-size="11" text-anchor="middle">
    <text x="165" y="130">1</text><text x="215" y="130">1</text><text x="265" y="130">1</text>
    <text x="315" y="130">1</text><text x="365" y="130">1</text><text x="465" y="130">1</text><text x="515" y="130">0</text>
  </g>
  <rect x="140" y="116" width="250" height="20" rx="4" fill="none" stroke="var(--accent)"/>
  <text x="265" y="150" fill="var(--accent)" font-size="8" text-anchor="middle">five 1s → stuffed 0 dropped</text>
  <text x="415" y="130" fill="var(--accent)" font-size="11" text-anchor="middle">0</text>
  <line x1="408" y1="122" x2="422" y2="134" stroke="var(--accent)"/><line x1="422" y1="122" x2="408" y2="134" stroke="var(--accent)"/>
  <text x="8" y="186" fill="var(--fg-muted)" font-size="9">shift reg</text>
  <g fill="currentColor" font-size="11" text-anchor="middle">
    <text x="215" y="190">0</text><text x="265" y="190">1</text><text x="315" y="190">1</text><text x="365" y="190">1</text>
    <text x="415" y="190">1</text><text x="465" y="190">1</text><text x="515" y="190">1</text><text x="565" y="190">0</text>
  </g>
  <rect x="190" y="176" width="400" height="20" rx="4" fill="none" stroke="var(--accent)"/>
  <text x="390" y="212" fill="var(--accent)" font-size="9" text-anchor="middle">0x7E at any bit offset → closeFrame · openFrame</text>
</svg>
<figcaption>NRZI writes 0 under every transition and 1 under every hold; the framer drops the 0 stuffed after five 1s and closes a frame the instant its shift register holds <code>0x7E</code>.</figcaption>
</figure>

## Six-bit ASCII and the types GopherTrunk really decodes

The payload handed to `ais.Decode` is one bit per byte, MSB-first — bit 0
is the spec's "bit 1". AIS fields start at arbitrary offsets (the 30-bit
MMSI at bit 8; a Class A longitude at bit 61), so a bit slice avoids
byte-alignment fences mid-field; `readBitsInt` sign-extends lat/lon, which
arrive in 1/600,000 of a minute. Text is **six-bit ASCII**: a fixed 64-entry
table (M.1371-5 Table 47) of `@`, capitals, punctuation and digits, with
`readAISString` stripping the trailing `@` padding
(`TestSixBitASCIITableMatchesSpec`). Every type opens with a 6-bit message
type, 2-bit repeat indicator and 30-bit MMSI, and `Decode` dispatches on the
first:

| Type | `TypeString` | Parsed fields |
|---|---|---|
| 1, 2, 3 | `position-a` | nav status, SOG, lon, lat, COG, heading, timestamp (168 bits) |
| 4 | `base-station` | station lat/lon |
| 5 | `static-voyage` | IMO, callsign, name, ship type, dimensions, ETA, draught, destination (424 bits) |
| 18, 19 | `position-b`, `position-b-ext` | the Class B position; 19's name/type not parsed |
| 24 | `static-b` | Part A: name · Part B: ship type, callsign, dimensions |
| 9, 21, 27, 6/8/12/14 | `sar-aircraft`, `aid-to-nav`, `long-range`, `safety` | **labelled only** — `RawHex` kept |

Sentinels are honoured, not plotted: longitude 181° or latitude 91° leaves
`HasPosition` false (`TestPositionNotAvailableSentinel`).
The canonical vector is gpsd's type-1 AIVDM sample:
`TestReceiverEmitsPositionMessage` pushes MMSI 366053209 through
`buildAISFrame` → `wrapHDLC` → `Receiver.Push` and expects a `position-a`
event at latitude ≈ 37.802 N.

Two scope limits need saying plainly. Types 5, 19 and 26 span two AIS slots
on the air; the parser decodes their single-slot forms, but **multi-slot
reassembly is not built** — `docs/ais.md` lists it as pending. And the
storage comment that positions cover "types 1/2/3/4/18/19/27" overstates
`Decode`: type 27 is labelled, not positioned. The independent reference is
gpsd's AIVDM documentation — per
[From Spec to Shipping Part 3]({{ '/blog/deep-dives/from-spec-to-shipping-03-literal-vectors/' | relative_url }}),
the external vector that stops a decoder grading its own homework.

## From bus event to the `/ais` panel

Here is the eleven-place pattern of
[Part 1]({{ '/blog/deep-dives/beyond-voice-01-eleven-place-pattern/' | relative_url }})
walked through for AIS:

1. **Front end + framer**: `ais/gmsk` and `ais/receiver`, above.
2. **Bus event**: `events.KindAISMessage` (`"ais.message"`), payload
   `storage.AISMessage`.
3. **Storage**: `storage.VesselLog` → one `vessel_log` row per message,
   indexed on `(received_at)` and `(mmsi, received_at)`.
4. **REST**: `GET /api/v1/ais/vessels?limit=N` (default 200, max 5000; 503
   without `storage.path`).
5. **Panel**: `AIS.tsx` polls every 5 s, badges `fcs`, and feeds the
   `PositionMap` of
   [Operator Cockpit Part 9]({{ '/blog/deep-dives/operator-cockpit-09-the-map/' | relative_url }}).
6. **Config**: `ais.channels[]` — `serial`, `frequency_hz`, `drop_bad_fcs`,
   `drop_non_position`.
7. **Field help**: `AISConfig.*` / `AISChannelConfig.*` in
   `internal/configbuilder/fieldmeta.go`.
8. **Route tests**: `"/ais"` in `web/src/nav/registry.test.ts`.
9. **Preflight**: `"ais"` in the "storage.path is empty but these decoders
   need it" warning.
10. **Retention**: `vessel_log` in `storage.decoderLogTables`.
11. **config.example.yaml**: the commented `ais:` block.

The daemon glue is the standard spawn closure — `SetCenterFreq` on the SDR's
iqtap broker, `openSingleChannelIQ`, `Process` — with `WRN` lines
`ais: SDR not found, skipping receiver` / `ais: SetCenterFreq failed` /
`ais: open IQ failed` for its failure modes, none fatal to trunking.
`BitsEmitted` climbing while `FramesIn` stays at zero means no flag is ever
found.

What is *not* verified deserves equal clarity. Every AIS test is synthetic
and there is **no committed on-air capture** — `docs/ais.md` files
real-fixture validation under pending. By
[From Spec to Shipping Part 10]({{ '/blog/deep-dives/from-spec-to-shipping-10-the-on-air-gate/' | relative_url }})'s
standard AIS is synthetic-verified only — the
[self-consistent trap]({{ '/blog/solution-postmortem/from-the-issue-tracker-20-self-consistent-trap/' | relative_url }})
in waiting.

### How AIS shaped the Go code

- **Borrow the link layer, own the physical layer.** The HDLC framer, CRC
  routine and NRZI rule are AX.25's; `ais/receiver` imports the framer and
  copies the fourteen-line NRZI type rather than growing a shared package
  early.
- **Two receivers, two `Stats()`.** IQ-to-bits and bits-to-message are
  separate types with separate counters, so a dead channel localises to "no
  bits" or "no frames" from `/metrics` alone.
- **Publish marginal frames, flag them.** `FCSOK` travels with the message
  instead of gating it; the operator chooses `drop_bad_fcs`.
- **Never return an error from `Decode`.** Unknown payloads come back as
  `TypeUnknown` with `RawHex` intact.

## Where this goes next

AIS shares its band with a protocol that could not be more different at the
bit level: DSC, the digital calling and distress layer on channel 70 —
1200-baud FFSK tones, ten-bit characters with a detect-only check code, a
phasing sequence instead of a flag.
[Part 10]({{ '/blog/deep-dives/beyond-voice-10-dsc-marine-calling/' | relative_url }})
follows it from the discriminator to a distress alert on the map.

## FAQ

**What frequencies does GopherTrunk decode AIS on?**
The two AIS channels, 161.975 MHz (87B) and 162.025 MHz (88B). Each
`ais.channels` entry pins one SDR to one frequency; Class A vessels alternate
between them, so two SDRs catch both halves.

**Why does the AIS slicer use a fixed zero threshold?**
GMSK's discriminator output is symmetric about DC, and NRZI plus HDLC bit
stuffing guarantees a transition at least every six bits, so no DC bias can
build up. A tracking threshold would only add a way to drift — FleetSync's
long-space-tone failure.

**Which AIS message types does GopherTrunk fully decode?**
Class A position reports (types 1, 2, 3), base-station reports (4), static
and voyage data (5), Class B position reports (18, 19) and Class B static
data (24). Other types are recognised, labelled — `sar-aircraft`,
`long-range` and so on — and kept as raw hex on the row.

**Does GopherTrunk output NMEA AIVDM sentences?**
No. Decoded messages publish as `events.KindAISMessage` with typed fields,
persist to `vessel_log`, and are served as JSON from
`GET /api/v1/ais/vessels`. The `raw_hex` column carries the full payload
bits, so an AIVDM re-encoder could sit on top of it.

**Has the AIS decoder been verified on real signals?**
Not yet. The bit-stream layer is pinned by synthetic tests around gpsd's
canonical type-1 vector; no on-air capture is committed, and `docs/ais.md`
lists real-fixture validation as pending. Treat it as synthetic-verified
only.

## Series navigation

**Part 9 of 14** · ←
[Part 8: ADS-B — Mode S Squitters & CRC-24]({{ '/blog/deep-dives/beyond-voice-08-adsb-mode-s/' | relative_url }})
· Next →
[Part 10: DSC — Marine Digital Selective Calling]({{ '/blog/deep-dives/beyond-voice-10-dsc-marine-calling/' | relative_url }})
