---
layout: page
title: FleetSync / Kenwood Signalling
description: Decoded Kenwood FleetSync FFSK pipeline — DSP frontend, events bus, SQLite log, REST endpoint, web panel
nav_group: Reference
---

# FleetSync / Kenwood Signalling

GopherTrunk decodes **FleetSync** — the analog in-band FFSK burst
Kenwood two-way radios key at the start of a transmission. It carries
the radio's **fleet number** and **unit ID** (ANI — automatic number
identification) on otherwise-analog conventional VHF / UHF voice
channels, so on a Kenwood fleet that is just FM voice it tells you
*which* radio is talking. Both the original **FleetSync** frame and the
error-corrected **FleetSync II** frame are decoded.

Added in response to [#437](https://github.com/MattCheramie/GopherTrunk/issues/437)
and verified against the on-air captures in
[#1184](https://github.com/MattCheramie/GopherTrunk/issues/1184).

## Modulation

FleetSync is a **1200-baud FFSK** burst using the CCIR tones
**mark = 1200 Hz** (binary 1) and **space = 1800 Hz** (binary 0),
carried inside the narrowband-FM voice channel — the same modulation
class as [MDC1200](mdc1200.html), so the DSP frontend reuses the same
building blocks. The line code is plain NRZ, sliced at a fixed zero
threshold (a tracked threshold measurably hurts here: a FleetSync II
frame can open with ~68 symbols of continuous space tone).

## Pipeline

```
IQ chunks (Fs Hz, complex64)
  → FM demod (internal/dsp/demod.FM)
  → real resampler to 9600 Hz (1200 baud × 8 oversample)
  → FFSK tone discriminator (mark 1200 Hz / space 1800 Hz)
  → Mueller-Müller symbol-timing recovery (8 sps → 1 sample/symbol)
  → zero-threshold NRZ slicer
  → 24-bit preamble + 16-bit sync framer (internal/radio/fleetsync.Framer)
  → FleetSync I block check, then FleetSync II ECC (internal/radio/fleetsync)
  → events.KindFleetSyncMessage on the bus
  → storage.FleetSyncLog → fleetsync_log SQLite table
  → GET /api/v1/fleetsync/messages → /fleetsync web panel
```

After the alternating preamble and the 16-bit sync word `0xA23E`, the
frame carries two 32-bit data words. A **FleetSync** frame validates
them with a 16-bit block check (a bit-serial LFSR, polynomial
`0x6815`); a **FleetSync II** frame protects the same words with four
64-bit single-error-correcting blocks and is tried when the FleetSync I
check fails. The fleet is word 1's second byte plus 99; the unit ID is
word 1's third byte and the high nibble of word 2's first byte, plus
999. The sync hunt accepts the bit-complemented sync word so a flipped
FM discriminator (inverted tone sense) still decodes.

## Configuration

Each entry pins one SDR to a conventional analog voice channel:

```yaml
fleetsync:
  channels:
    - serial: "uhf-antenna"
      frequency_hz: 462_562_500   # the analog voice channel to monitor
      baud_hz: 1200               # 0 / omitted = 1200; 2400 also accepted
      drop_bad_crc: false         # true to drop block-check-failed bursts
```

Leave `drop_bad_crc` false to see block-check-failed bursts on the
panel (flagged with `crc_ok=false`); flip it on for noisy channels.
Like the other message decoders, FleetSync needs `storage.path` set —
without it the receiver runs but nothing is persisted, the REST endpoint
answers 503 and the panel stays empty (`gophertrunk doctor` warns).

## What's surfaced

- **Bus event** — `events.KindFleetSyncMessage` (`fleetsync.message`
  over SSE / WS), payload `storage.FleetSyncMessage`: `fleet`, `unit`,
  `fs2`, `crc_ok`, `raw_hex`, `body`, `received_at`.
- **Storage** — the `fleetsync_log` SQLite table (fleet, unit, fs2,
  body, raw hex, crc_ok), indexed by time and by fleet + unit; swept by
  `retention.log_days` with the other decoder logs.
- **REST** — `GET /api/v1/fleetsync/messages?limit=N` (default 200,
  max 5000); 503 when the daemon runs without `storage.path`.
- **Web** — the `/fleetsync` panel polls every 5 s and shows fleet,
  unit, the FleetSync / FleetSync II variant, the raw words and the
  block-check result. Hide it with `web.tabs.fleetsync: false`.

## Verifying a capture offline

`TestFleetSyncReplay` runs a recording through the production chain and
prints a per-burst timeline; it reads SDR# baseband WAVs (32-bit float
I/Q), cs16 / f32 raw I/Q and mono discriminator audio, finds the carrier
in a wideband capture, and sweeps 1200 / 2400 baud:

```
GT_FLEETSYNC_IQ=FleetSync-II.wav go test ./cmd/gophertrunk -run 'TestFleetSyncReplay$' -v
```

Two channelized slices of the #1184 captures are committed under
`internal/radio/fleetsync/afsk/testdata` as real-air regression
fixtures (Fleet 107 / Unit 1772).

## Licensing note

The FleetSync protocol facts implemented here (sync word, frame
layout, block-check polynomial, FleetSync II parity tables, ANI field
layout) are public protocol details. This is a clean-room Go
implementation; no third-party decoder source — including the
GPL-licensed reference decoders — is incorporated, keeping the decoder
under the project's Apache-2.0 license.
