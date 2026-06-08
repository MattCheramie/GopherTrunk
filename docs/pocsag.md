---
layout: page
title: POCSAG paging decoder
description: Protocol layer for decoding POCSAG (CCIR 584) wireline FSK pager traffic — fire / EMS dispatch, commercial paging, amateur DAPNET
nav_group: Reference
---

# POCSAG paging decoder

GopherTrunk now decodes the **POCSAG** (Post Office Code
Standardisation Advisory Group, CCIR Recommendation 584) wireline
FSK pager protocol — the dominant pager protocol globally and the
one most fire / EMS departments use for tone-out dispatch
forwarding. The protocol layer (codeword parsing, BCH(31,21), batch
carve-up, numeric + alphanumeric message reassembly) and the DSP
wiring (FM demod → bit slicer → sync detector → batch decoder → bus
event) are both shipped. The higher-rate Motorola **FLEX** protocol
shares this page's bus event, storage, and panel — see the FLEX
section below.

## What's here today

- **`internal/radio/framing/bch_pocsag.go`** — BCH(31,21) encoder
  + brute-force minimum-Hamming-distance decoder, plus the
  trailing-parity helper POCSAG uses to stretch the code to 32
  wire bits. Generator polynomial: g(x) = x^10 + x^9 + x^8 + x^6
  + x^5 + x^3 + 1 (CCIR 584 §3.2.1).
- **`internal/radio/pager/pocsag/codeword.go`** — 32-bit codeword
  struct, sync (`0x7CD215D8`) + idle (`0x7A89C197`) pattern
  recognition, address/message/function decoding, parity check
  on the trailing bit.
- **`internal/radio/pager/pocsag/batch.go`** — batch carve-up
  (sync + 16 codewords × 8 frame slots), frame-slot index
  resolution, full-RIC reconstruction (18-bit address codeword
  field + 3-bit slot index → 21-bit pager address).
- **`internal/radio/pager/pocsag/message.go`** — numeric (5 BCD
  digits per codeword, with the CCIR 584 extended-character
  table: 0-9, *, U, space, -, ), ( ) and alphanumeric (7-bit
  LSB-first ASCII packed across 20-bit message fields)
  decoders. Trailing space-padding is trimmed.

## What's pending

- **End-to-end IQ validation.** The receiver code is wired and
  builds against the full pipeline, but the synthetic FM-modulated
  IQ test is skipped pending real captured fixtures (a `.cfile`
  under `samples/pocsag/`). The receiver's API surface is unit-
  tested (Options validation, ctx cancel, nil input); the protocol
  + storage + REST + UI stack is covered by their respective
  package tests. The remaining piece is tuning the
  integrator-and-slicer timing against real on-air bits — likely
  swapping the running-mean slicer for a proper Mueller-Müller +
  matched-filter combination once we have signal to calibrate
  against.
  Multi-channel POCSAG / FLEX from one wideband SDR is now shipped —
  see **Multi-channel (wideband) paging** below.

## Multi-channel (wideband) paging

A `paging.wideband` group puts several paging channels — any mix of
POCSAG and FLEX — on a **single** SDR. The daemon tunes the dongle to
a center frequency and runs an `internal/dsp/tuner.DDCBank`: one NCO
mixer + decimating resampler per channel pulls each narrow paging
channel down to a 48 kHz baseband stream, which feeds the same
per-protocol receiver the single-frequency path uses. This is the
same DDC primitive `role: wideband` uses for DMR Tier II.

So two pagers a few hundred kHz apart (e.g. FLEX on 153.0250 MHz and
POCSAG on 153.3500 MHz, 325 kHz apart) share one stick instead of
needing two. Each channel frequency must sit inside the dongle's IQ
window (`center ± sample_rate/2`, minus a 5% guard); a channel outside
the window is logged and skipped without disturbing its siblings. When
`center_freq_hz` is omitted (or 0) the daemon centers on the midpoint
of the channel frequencies.

## FLEX

FLEX is the higher-rate Motorola pager protocol that shares POCSAG's
operator workflow, bus event, `pager_log` table, and `/pager` panel —
rows are tagged with `protocol = "flex"`. The decoder
(`internal/radio/pager/flex`) handles the **1600 bps / 2-level** mode:

```
IQ → FM demod → resample → slicer → flex.Decoder.Push
   → 32-bit sync marker (0xA6C6AAAA) + 16-bit mode code (0x870C)
   → frame-info word → block de-interleave (88 codewords)
   → BCH(31,21)+parity → BIW → address / vector / message words
   → KindPagerMessage (Protocol="flex")
```

- **FEC:** FLEX BCH(31,21) reuses the tested POCSAG primitive via a
  codeword bit-reversal (`framing.FLEXBCHEncode21` /
  `FLEXBCHDecode32`), correcting up to 2 bit errors.
- **De-interleave:** the block transpose `idx = ((c>>5)&0xFFF8)|(c&7)`,
  bit `(c>>3)&31` per the reference decoders.
- **Polarity** auto-resolves (the sync hunt accepts the inverted
  marker), like POCSAG.
- **Scope / pending:** 3200 / 6400 bps and 4-level multi-phase modes,
  long (2-word) addresses, and multi-frame message reassembly are
  follow-ups. The sync framing, BCH bit order, and capcode mapping are
  validated end-to-end against a synthetic encoder; confirming them
  against a captured FLEX signal is the remaining real-world
  calibration step (shared caveat with the DSC frontend).

## Configuration

```yaml
paging:
  pocsag:
    - serial: "antenna-pi"       # SDR serial (must be in sdr.devices
                                 # or sdr.rtl_tcp)
      frequency_hz: 152_007_500  # local commercial paging / fire
                                 # dispatch / DAPNET / etc.
      baud_hz: 1200              # 512 / 1200 / 2400; default 1200
  flex:
    - serial: "antenna-pi2"      # SDR serial
      frequency_hz: 929_612_500  # local FLEX paging channel
  wideband:                      # two pagers on one dongle via DDC
    - serial: "pager-sdr"
      center_freq_hz: 153_187_500 # optional; auto = midpoint when 0
      channels:
        - protocol: flex
          frequency_hz: 153_025_000
        - protocol: pocsag
          frequency_hz: 153_350_000
          baud_hz: 1200
```

For `pocsag` / `flex` entries the daemon retunes the named SDR to
`frequency_hz` on startup and runs the receiver against its full IQ
stream via the iqtap broker. For a `wideband` group the SDR is tuned
to the group center and a DDC tap feeds each channel's receiver (see
**Multi-channel (wideband) paging** above). Either way, pages flow
onto `events.KindPagerMessage`, land in the SQLite `pager_log` table,
and render on the web `/pagers` panel tagged by protocol.

## What's shipped now

- **Syncer + page assembler** — `pocsag.Syncer` consumes a bit
  stream (one bit per byte), locks on the sync codeword (with
  polarity-inverse fallback so a flipped FM demod still works),
  carves out batches, decodes each codeword through BCH(31,21),
  and reassembles pages by correlating address codewords with
  the message codewords that follow them. Idle codewords +
  uncorrectable codewords + the next address terminate an
  in-progress page.
- **Bus event** — `events.KindPagerMessage` payload is a
  `storage.PagerMessage` carrying protocol ("pocsag" | "flex"), RIC /
  capcode, function code, encoding ("numeric" | "alpha" | "tone"),
  decoded text, and total BCH bit-error count.
- **SQLite persistence** — `storage.PagerLog` subscribes to the
  bus event and writes to a new `pager_log` table (mirrors
  `call_log` / `location_log`). Retention sweeper can be extended
  later when growth becomes a concern.
- **REST endpoint** — `GET /api/v1/pager/messages?limit=N`
  returns the most recent N pages (default 200, max 5000),
  newest first.
- **Web panel** — `/pagers` renders the live page list:
  Received / Type / RIC / Function / Encoding / Body / BER columns,
  polled every 5 s. The Type column shows a POCSAG / FLEX badge so
  mixed traffic (including a wideband group) is distinguishable at a
  glance. Non-zero BER is highlighted yellow.

## Testing

The protocol layer has 13 unit tests covering:

- BCH(31,21) round-trip (encode → decode = no errors), single-
  bit and double-bit error correction, triple-bit rejection
- Sync + idle codeword recognition
- Address + message codeword round-trips (encode → wire → decode
  reproduces the original fields)
- Single-bit error correction at every codeword position
- Parity-bit flip detection
- Frame-slot mapping (word index → slot index → RIC reconstruction)
- Batch carve-up with a synthetic address + message at a known slot
- Numeric BCD decode (including the CCIR 584 extended symbols,
  trailing-space trimming, LSB-first nibble order)
- Alphanumeric ASCII reassembly (7-bit LSB-first, character
  straddling 20-bit boundaries)
- Mixed-codeword slices (address + message words in one buffer
  — DecodeNumeric / DecodeAlpha ignore non-message codewords)

## Why now

Most operator workflows that already use GopherTrunk for trunked
voice want the local fire / EMS pager traffic alongside —
dispatch goes out on the trunked system AND on a paging
frequency, and seeing the pager text helps confirm "this
specific tone-out matched these specific crews."

Building on the iqtap broker (PR #365), the eventual DSP
pipeline will tap the same IQ stream the trunking decoder reads
on a separate broker subscriber, so adding POCSAG decode doesn't
double the USB / CPU cost of the SDR.

## References

- CCIR Recommendation 584-1, "Standard Codes and Format for
  International Radio Paging"
- [sdrtrunk POCSAG decoder](https://github.com/DSheirer/sdrtrunk)
  (Java) — sanity reference for the BCH polynomial choice and
  the LSB-first bit-order quirks
- [multimon-ng](https://github.com/EliasOenal/multimon-ng) (C) —
  the BCH lookup table + numeric BCD table used by most
  open-source POCSAG decoders
