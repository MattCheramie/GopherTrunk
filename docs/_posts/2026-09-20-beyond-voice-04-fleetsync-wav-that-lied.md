---
title: "Beyond Voice, Part 4: FleetSync — Cloning the Template & the WAV That Lied"
description: "How Kenwood FleetSync was cloned from the MDC1200 template — a preamble-plus-sync framer, a bit-serial block check, the FleetSync II single-error-correcting code — and how the reporter's two SDR# captures were decoded only after three traps fell: a WAV reader that read 32-bit float as int16, a baseband recording whose carrier sat 424.5 kHz off centre, and a carrier estimator that looked before the radio keyed up."
category: deep-dives
keywords: fleetsync decoder, kenwood fleetsync ani, fleetsync ii ecc, fleetsync sync word 0xa23e, sdr# wav 32-bit float iq, iq wav encoding bug, sdr baseband recording carrier offset, capture replay harness go, fleetsync fleet unit id, gophertrunk fleetsync
tags: [beyond-voice, fleetsync, kenwood, capture, wav, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "Beyond Voice"
series_part: 4
---

*Part 4 of **Beyond Voice**, a 14-part deep dive into everything GopherTrunk
decodes that is not a trunked voice call, and the one eleven-place wiring
pattern that carries each from a burst on the air to a row in the web console.
[Part 3]({{ '/blog/deep-dives/beyond-voice-03-mdc1200-template-decoder/' | relative_url }})
walked the MDC1200 template and ended on its gap: no on-air fixture. This part
is the clone that closed that gap — Kenwood FleetSync, requested in
[#437](https://github.com/MattCheramie/GopherTrunk/issues/437) and verified on
the captures posted to
[#1184](https://github.com/MattCheramie/GopherTrunk/issues/1184) — and the
three traps between "the decoder is written" and "the decoder decodes,"
none of them in the decoder.*

> **TL;DR:** `internal/radio/fleetsync` clones the MDC1200 shape with the
> FleetSync facts: a 24-bit alternating preamble (≤ 3 errors) plus the 16-bit
> sync `0xA23E` (≤ 1 error, or its complement), a 4-bit lead-in, then two
> 32-bit words whose low 16 bits are a **bit-serial block check** (polynomial
> `0x6815`); **FleetSync II** spreads the same words as nibbles across four
> 64-bit blocks of 15-bit single-error-correcting code words.
> `FE80083053059B6C` is Fleet 107 / Unit 1772. The reporter's SDR# captures
> decoded only after three fixes outside the decoder: every IQ WAV reader
> read the 32-bit float body as int16 pairs (`IQWavInfo.Encoding` now drives
> them all); a baseband recording is the whole tuner span, so the harness
> searches an averaged spectrum (`fleetSyncCarrierCandidates`) and found the
> carriers at +424.5 and +323 kHz; and the shared
> `dsp.EstimateCarrierCandidatesHz` looks only at the first ~256 windows,
> before the radio keyed. Result: FS-I **5/6**, FS-II **8/8** CRC-valid.
> Daemon wiring landed; the live Kenwood lab test is open.

**Key takeaways**

- **A clone changes constants and one code, not the shape.** Framer, decoder,
  front end and emit are the MDC1200 layout; the FleetSync work is the block
  check, the FS-II ECC and the ANI extraction, all pinned to multimon-ng.
- **The first thing to check on a capture is its level.** The harness's
  `signal:` line — rms, peak, DC — exposed the WAV bug: a real recording does
  not sit at −2.5 dBFS with a 0.16 DC offset.
- **A baseband recording is not a channel.** SDR# writes the whole tuner span
  with the signal wherever the VFO sat; a decoder run at centre decodes
  nothing, so the harness finds the carrier itself.
- **Capture-verified is the honest ceiling.** Two channelized real-air slices
  are committed fixtures and the bus path is pinned on one, but a live run on
  the Kenwood radios has not happened.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| Sync hunt | 24-bit preamble + `0xA23E` (or complement) | `internal/radio/fleetsync/framer.go` (`syncMatched`) |
| FS-I block check | bit-serial LFSR `0x6815` + parity, `^0x0002` | `internal/radio/fleetsync/fleetsync.go` (`fsyncCRC`) |
| FS-II ECC | 8 parity masks, 15 repair positions | `fleetsync.go` (`fs2ECCRepair`, `decodeFS2`) |
| ANI fields | fleet = byte2+99, unit = (byte3<<4 \| nibble)+999 | `fleetsync.go` (`newMessage`) |
| WAV encoding | pcm16 / pcm8 / float32 from the fmt chunk | `internal/sdr/baseband/wav.go` (`IQWavInfo.Encoding`) |
| Carrier search | whole-capture 4096-pt spectrum, ≥ 6 dB peaks | `cmd/gophertrunk/fleetsync_replay_test.go` (`fleetSyncCarrierCandidates`) |
| Replay gate | production chain → per-burst timeline → VERDICT | `fleetsync_replay_test.go` (`TestFleetSyncReplay`) |
| Real-air pins | 1.3 s channelized slices, Fleet 107 / Unit 1772 | `internal/radio/fleetsync/afsk/realair_test.go` |

## In this post

- **Cloning the template** — what FleetSync kept from MDC1200, and changed.
- **Words, checks and nibbles** — the block check, the ECC, the ANI fields.
- **The WAV that lied** — 32-bit float read as int16, and the fmt-chunk fix.
- **The carrier that wasn't at centre** — a whole-capture spectrum search.
- **Verdict, fixtures and the open gate** — 5/6, 8/8, and what's still open.

## Cloning the template

FleetSync is Kenwood's answer to the question MDC1200 answers for Motorola:
which radio just keyed up on an analog channel. Same 1200-baud CCIR FFSK,
same head-of-PTT burst — so the package was built as a clone of the template
[Part 3]({{ '/blog/deep-dives/beyond-voice-03-mdc1200-template-decoder/' | relative_url }})
laid out. The framer keeps the register hunt, the complement lock and the
reset-after-burst; the front end keeps the FM → resample → `demod.FFSK` →
Mueller-Müller chain from
[Part 2]({{ '/blog/deep-dives/beyond-voice-02-afsk-ffsk-fundamentals/' | relative_url }}).
What changed is the framing constants.

The sync hunt is stricter and two-stage. Where MDC1200 tolerates five errors
in forty bits, `syncMatched` demands a 24-bit alternating preamble
(≤ `preambleMaxErrors` = 3) in the register's high bits *and* the 16-bit sync
`0xA23E` within `syncMaxErrors` = 1 in its low bits — the block check
downstream is the real gate. The complement path is MDC1200's:
`hamming16(syn, ^SyncWord) ≤ 1` sets `inverted` and every bit is XORed back.
`TestFramerIgnoresIdleAndPreambleOnly` feeds pure preamble and demands zero
locks; the sync, not the dotting, arms capture.

Capture is `FrameBits` = 260 bits: a 4-bit lead-in (`frameOffset`) then
either a FleetSync I frame in the first 68 or a FleetSync II frame in all
260. The whole layout — sync, offset, check polynomial, ECC tables, fields —
is the multimon-ng `fsync` decoder's, ported clean-room and cross-checked
against a Python reference contributed on #437 — the provenance
[From Spec to Shipping Part 3]({{ '/blog/deep-dives/from-spec-to-shipping-03-literal-vectors/' | relative_url }})
asks for. `TestFS2ECCRepairCorrectsEverySingleBit` flips each of the 15 code
bits and demands the tables repair it.

## Words, checks and nibbles

`DecodeFrame` reads `word1` from bits 4–35 and `word2` from 36–67 and tries
FleetSync I first:

```go
// internal/radio/fleetsync/fleetsync.go (shape)
func DecodeFrame(raw []byte) (Message, bool) {
    word1 := bitsToUint32(raw[frameOffset : frameOffset+32])
    word2 := bitsToUint32(raw[frameOffset+32 : frameOffset+64])
    if fsyncCRC(word1, word2) == uint16(word2&0xFFFF) {
        return newMessage(word1, word2, false, true), true      // FleetSync I
    }
    if len(raw) >= FrameBits {
        if w1, w2, ok := decodeFS2(raw); ok {
            return newMessage(w1, w2, true, true), true          // FleetSync II
        }
    }
    return newMessage(word1, word2, false, false), false        // best effort
}
```

`fsyncCRC` is not a table CRC. The first 48 bits clock a bit-serial LFSR
with polynomial `0x6815`; `word2`'s low 15 bits feed a running parity bit;
the result is `(lfsr ^ 0x0002) + parity`, compared against `word2`'s low 16.
It is ported bit-for-bit from multimon-ng, and the synthesiser does not
invert it in closed form — `SolveBlockCheck` searches all 65 536 check
values for the fixed point rather than risk a wrong constant.

**FleetSync II** protects the same two words with forward error correction:
four 64-bit blocks of four 16-bit half-words, each a 15-bit code word whose
high nibble is one data nibble. `fs2ECCRepair` computes an 8-bit syndrome
from `fs2ParityCheck`; a syndrome matching one of the 15 `fs2RepairPos`
entries flips that bit, anything else fails. `decodeFS2` repairs all sixteen,
reassembles the two words and runs the same check.

Then the ANI. `newMessage` reads fleet from `word1`'s byte 2 plus 99, unit
from byte 3 shifted left four plus `word2`'s high nibble plus 999. The
reporter's `FE80083053059B6C` gives fleet 107, unit **1772** — the pair
every test and the harness assert.

## The WAV that lied

With the decoder written, the reporter posted two SDR# baseband recordings —
2.048 MS/s, VFO 462.5625 MHz — and `TestFleetSyncReplay` decoded nothing.
The blocker was never FleetSync, it was the WAV reader. The harness prints
its capture's level first, and that line was the tell:

```
capture: FleetSync-II.wav format=container samples=… rate=2048000 Hz
signal: rms=-2.5 dBFS peak=… dBFS dc=(0.1600,…)
```

A real capture does not sit at −2.5 dBFS with a 0.16 DC offset. SDR# writes
IQ WAVs as two-channel **32-bit IEEE float**; `siglab.DecodeContainerFile`
stripped 44 bytes and read the body as int16 pairs — twice the sample count,
all noise — while `baseband.parseIQWavHeader` refused anything but 16-bit.
The [#764](https://github.com/MattCheramie/GopherTrunk/issues/764)
gain-staging lesson applied to a file: an absurd level answers nothing.

The fix makes the fmt chunk decide. `IQWavInfo` gained an `Encoding` —
`IQWavPCM16`, `IQWavPCM8`, `IQWavFloat32` — from the format tag and bits per
sample, `WAVE_FORMAT_EXTENSIBLE` unwrapped and pre-`data` chunks skipped:

```go
// internal/sdr/baseband/wav.go (shape)
switch {
case tag == wavFormatPCM && bits == 16:       info.Encoding = IQWavPCM16
case tag == wavFormatPCM && bits == 8:        info.Encoding = IQWavPCM8
case tag == wavFormatIEEEFloat && bits == 32: info.Encoding = IQWavFloat32
default:
    return fmt.Errorf("baseband: WAV tag 0x%04X at %d bits is not a supported IQ encoding", tag, bits)
}
```

That one field now drives the replay `FileDriver`, `siglab.UnwrapContainer`,
`DecodeContainerFile` and siglab's input path, pinned failing-first by
`TestIQWavReadersHonourFmtChunk` and `TestDecodeContainerFileReadsFloat32WAV`.

## The carrier that wasn't at centre

The second trap was geometry. An SDR# baseband recording is the whole tuner
span, not the VFO — it holds ±1 MHz with the channel wherever the VFO sat.
In these two files the carrier sat at **+424.5 kHz** and **+323 kHz**; a
decoder run at centre sees empty spectrum.

The harness therefore finds the carrier itself. `fleetSyncCarrierCandidates`
averages a 4096-point periodogram over the *whole* capture, keeps maxima at
least 6 dB over the median floor, and `searchFleetSyncCarrier` runs the
production chain on up to eight of them — each mixed to DC through
`ccdecoder.NewDownconverterWithOffset` into the same 48 kHz slice every NBFM
path uses — until one yields a CRC-valid burst:

```
carrier search: candidate 1 at +424500 Hz (64.0 dB over the floor): sync_locks=… crc_ok=…
carrier search: tuning to +424500 Hz (set GT_FLEETSYNC_TUNE_HZ to override)
```

Why not the shared `dsp.EstimateCarrierCandidatesHz` the hunt uses? It is
right for a continuous signal and wrong for a bursty one: its periodogram
sums up to 256 windows of `coarseWin` samples — tens of milliseconds at
2.048 MS/s — which here is *before the radio keys up*. It ranked a −423 kHz
spur first and never saw the 64 dB carrier. The harness averages over
seconds and leaves the shared estimator alone.

<figure class="lab-figure">
<svg viewBox="0 0 680 250" width="680" height="250" role="img" aria-label="Two panels. Left: a baseband spectrum spanning plus and minus one megahertz around the tuner centre, with a spur at minus 423 kilohertz and a strong FleetSync carrier at plus 424.5 kilohertz; the shared estimator's short early window sees only the spur while the harness's whole-capture average finds the carrier. Right: the same file read as int16 gives noise at minus 2.5 dBFS, while the fmt-chunk reader yields the true float samples.">
  <line x1="20" y1="150" x2="400" y2="150" stroke="var(--fg-muted)"/>
  <text x="20" y="166" fill="var(--fg-muted)" font-size="8">−1 MHz</text>
  <text x="210" y="166" text-anchor="middle" fill="var(--fg-muted)" font-size="8">tuner centre</text>
  <text x="400" y="166" text-anchor="end" fill="var(--fg-muted)" font-size="8">+1 MHz</text>
  <path d="M 20 140 L 120 138 L 130 118 L 140 138 L 290 140 L 296 60 L 300 58 L 304 60 L 310 140 L 400 138" fill="none" stroke="currentColor"/>
  <text x="130" y="110" text-anchor="middle" fill="var(--fg-muted)" font-size="8">spur −423 kHz</text>
  <text x="300" y="50" text-anchor="middle" fill="var(--accent)" font-size="9">FleetSync +424.5 kHz, 64 dB</text>
  <rect x="290" y="56" width="20" height="94" fill="none" stroke="var(--accent)" stroke-dasharray="3 2"/>
  <text x="300" y="182" text-anchor="middle" fill="var(--accent)" font-size="8">48 kHz slice</text>
  <text x="130" y="205" text-anchor="middle" fill="var(--fg-muted)" font-size="8">shared estimator: first ~256 windows, before keyup → spur</text>
  <text x="300" y="205" text-anchor="middle" fill="currentColor" font-size="8">harness: whole-capture average → the carrier</text>
  <line x1="430" y1="30" x2="430" y2="230" stroke="var(--fg-muted)" stroke-dasharray="3 4"/>
  <rect x="450" y="40" width="210" height="34" rx="5" fill="none" stroke="var(--fg-muted)"/>
  <text x="555" y="54" text-anchor="middle" fill="currentColor" font-size="9">SDR# WAV: fmt tag 3, 32-bit float</text>
  <text x="555" y="67" text-anchor="middle" fill="var(--fg-muted)" font-size="8">two channels, 2.048 MS/s</text>
  <line x1="500" y1="74" x2="480" y2="110" stroke="currentColor"/>
  <line x1="610" y1="74" x2="630" y2="110" stroke="var(--accent)"/>
  <rect x="440" y="110" width="100" height="60" rx="5" fill="none" stroke="currentColor"/>
  <text x="490" y="128" text-anchor="middle" fill="currentColor" font-size="9">old reader</text>
  <text x="490" y="142" text-anchor="middle" fill="var(--fg-muted)" font-size="8">skip 44 B, read int16</text>
  <text x="490" y="156" text-anchor="middle" fill="var(--fg-muted)" font-size="8">2× samples, −2.5 dBFS, dc 0.16</text>
  <rect x="570" y="110" width="100" height="60" rx="5" fill="none" stroke="var(--accent)"/>
  <text x="620" y="128" text-anchor="middle" fill="var(--accent)" font-size="9">IQWavInfo.Encoding</text>
  <text x="620" y="142" text-anchor="middle" fill="var(--fg-muted)" font-size="8">fmt chunk decides</text>
  <text x="620" y="156" text-anchor="middle" fill="var(--fg-muted)" font-size="8">float32 body, true samples</text>
</svg>
<figcaption>Two traps outside the decoder: the carrier sits 424.5 kHz off the tuner centre where a short early window never sees it, and the float body was read as int16 until the fmt chunk decided.</figcaption>
</figure>

## Verdict, fixtures and the open gate

With the reader honest and the carrier found, one command per file prints a
per-burst timeline and a verdict:

```
=== baud 1200: sync_locks=… framed=… crc_ok=5 crc_fail=…
  t=   1.204s OK FleetSync-I   fleet=107  unit=1772  raw=FE80083053059B6C
  …
VERDICT: PASS — the capture's ANI (fleet 107 unit 1772) decodes CRC-valid through the production front end
```

FleetSync-I decoded **5 of 6** ANI bursts CRC-valid (one RF miss at
t = 3.0 s), FleetSync-II **8 of 8**, the raw words `FE80083053059B6C` and
`FC80083053057E59` repeating — through the production front end. It also
ranks common rates by CRC-valid bursts for a headerless file, since a wrong
rate decodes nothing (`TestReceiverWrongBaudDoesNotFalseDecode`), and a
synthetic self-check means a FAIL on a real file is the capture or framing,
never a broken harness — the discipline
[From Spec to Shipping Part 11]({{ '/blog/deep-dives/from-spec-to-shipping-11-capture-driven-development/' | relative_url }})
calls capture-driven development.

Then the fixtures. Two 1.3 s slices of the reporter's bursts, channelized to
48 kHz and committed under `afsk/testdata/`, are decoded by
`TestFleetSyncRealAirSlices` — the on-air pin the #764/#771 rule demands, and
the one the MDC1200 template lacks. `TestReceiverPublishesRealAirBurstsOnBus`
checks the `storage.FleetSyncMessage` payloads through the bus path.

The daemon wiring landed with it — `fleetsync.channels`,
`KindFleetSyncMessage`, `fleetsync_log`, `GET /api/v1/fleetsync/messages`,
the `/fleetsync` panel, the Builder section and the `doctor` list — the
eleven places of
[Part 1]({{ '/blog/deep-dives/beyond-voice-01-eleven-place-pattern/' | relative_url }}).
What has **not** happened is a live run: CLAUDE.md keeps the Kenwood lab test
open (#764/#771). The captures prove the decoder; only the radios prove the
daemon.

### How the clone shaped the Go code

- **Reference literals over closed forms.** `fs2ParityCheck` and
  `fs2RepairPos` are transcribed verbatim; `SolveBlockCheck` searches rather
  than inverting `fsyncCRC`, so the test builders cannot encode a wrong
  constant.
- **The synthesiser confesses its limits.** `synth.go`'s doc says a green
  round-trip proves the wiring, not the framing — the self-consistent trap
  named in the code.
- **The harness measures before it decodes.** `signalStats()` prints rms,
  peak and DC first; the VERDICT line names the layer to suspect.
- **One field, every reader.** `IQWavInfo.Encoding` is resolved once in
  `applyWavFmt` and consumed by every reader, so the fix could not land in
  one and miss another.

## Where this goes next

FleetSync and MDC1200 both ride the tone pair of Part 2. The next decoder
rides *tones* alone — no bits at all.
[Part 5]({{ '/blog/deep-dives/beyond-voice-05-two-tone-tone-out/' | relative_url }})
turns to two-tone sequential paging: the Goertzel detector that measures one
frequency's energy without an FFT, the A/B tone timing of fire-department
tone-outs, and the alert path that beeps before a dispatcher speaks.

## FAQ

**What is Kenwood FleetSync and what does GopherTrunk decode from it?**
FleetSync is the 1200-baud FFSK burst Kenwood radios key at the start of a
transmission on analog channels, carrying a fleet number and unit ID (ANI).
GopherTrunk decodes both the original and the error-corrected FleetSync II
frame, publishing fleet, unit, variant and raw words to the bus, SQLite,
REST and the `/fleetsync` panel.

**How is FleetSync II different from FleetSync I?**
Same two 32-bit data words, but FleetSync II spreads their nibbles across
four 64-bit blocks of 15-bit single-error-correcting code words, so one
flipped bit per half-word is repaired before the block check. GopherTrunk
tries the FleetSync I check first and falls back to the ECC path.

**Why did the SDR# capture decode nothing at first?**
Two reasons outside the decoder: the IQ WAV readers read SDR#'s 32-bit float
body as int16 pairs, producing noise at −2.5 dBFS; and the recording is the
whole tuner span, with the carrier 424.5 kHz off centre. The fmt chunk now
selects the encoding and the harness searches an averaged spectrum.

**How do I replay my own FleetSync capture?**
`GT_FLEETSYNC_IQ=<file> go test ./cmd/gophertrunk -run 'TestFleetSyncReplay$'
-v`. WAV and FLAC carry their own rate; headerless files take
`GT_FLEETSYNC_RATE` and `GT_FLEETSYNC_FORMAT`. Set
`GT_FLEETSYNC_FLEET`/`GT_FLEETSYNC_UNIT` to your radios' ANI, or `FLEET=0` to
report without asserting.

**Is FleetSync verified on air?**
Offline, yes: the two captures decode 5 of 6 and 8 of 8 bursts CRC-valid as
Fleet 107 / Unit 1772, and slices are committed fixtures. The daemon wiring
is tested with those slices and fakes, but a live run on the Kenwood radios
is still open under the #764/#771 rule — synthetic plus offline is not on air.

## Series navigation

**Part 4 of 14** · ←
[Part 3: MDC1200 — The Template Decoder]({{ '/blog/deep-dives/beyond-voice-03-mdc1200-template-decoder/' | relative_url }})
· Next →
[Part 5: Two-Tone Paging — Goertzel, Profiles & Tone-Out]({{ '/blog/deep-dives/beyond-voice-05-two-tone-tone-out/' | relative_url }})
