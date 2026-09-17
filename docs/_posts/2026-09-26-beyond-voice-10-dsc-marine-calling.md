---
title: "Beyond Voice, Part 10: DSC — Marine Digital Selective Calling"
description: How GopherTrunk decodes DSC on marine channel 70 — 1200-baud FFSK at 1300/2100 Hz, ten-bit characters with a detect-only check code, the phasing hunt that locks a 20-bit DX grid, the format and category tables, MMSI and position codecs, and what stays unverified without a capture.
category: deep-dives
keywords: dsc decoder sdr, digital selective calling, channel 70 156.525, itu-r m.493, ffsk 1300 2100, bch 10 7 dsc, distress alert decode, mmsi decoder, dx rx redundancy, gophertrunk dsc
tags: [beyond-voice, dsc, ffsk, marine, gmdss, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "Beyond Voice"
series_part: 10
---

*Part 10 of **Beyond Voice**, a 14-part deep dive into everything GopherTrunk
decodes that is not a trunked voice call — the AFSK signalling formats,
paging, APRS, ADS-B, AIS, DSC, LoRa, M17 and the amateur digital-voice
family — and the one eleven-place wiring pattern that carries each of them
from a burst on the air to a row in the web console.
[Part 9]({{ '/blog/deep-dives/beyond-voice-09-ais-gmsk-nrzi/' | relative_url }})
decoded the ships' transponders. This part decodes the channel they use to
call each other and to call for help: DSC, whose tones, characters and
framing share nothing with AIS except the antenna — and whose one design
choice, a check code that can only detect, shapes the whole receiver.*

> **TL;DR:** DSC is the GMDSS calling and distress layer on marine VHF
> channel 70 (156.525 MHz): 1200-baud FFSK, mark 1300 Hz / space 2100 Hz,
> 10-bit characters (7 data + 3 check) each sent twice as DX and RX copies.
> `internal/radio/dsc/ffsk` is the front end — `demod.FM` → `demod.FFSK`
> at 9600 sps → `sync.MuellerMuller` → a **DC-tracking slicer**, no NRZI. `internal/radio/dsc/receiver` hunts two phasing DX
> characters (`phasingDX` 125) exactly `dxStride` = 20 bits apart at the
> same polarity, locks the DX grid, reads DX only, and closes on an EOS
> (117 / 122 / 127). `dsc.BCHCheck` is a CRC-3 syndrome that detects but
> cannot correct; `dsc.Decode` fills format, category, MMSIs and — for
> distress — nature, position and time. Output: `KindDSCMessage` →
> `dsc_log` → `GET /api/v1/dsc/messages` → `/dsc`. Synthetic-verified only
> (`TestEndToEndDistressDecode`); the wire format is unconfirmed on air.

**Key takeaways**

- **No NRZI means polarity matters again.** DSC is direct FSK, so the
  receiver slices against a slow-EMA mean and lets the phasing hunt accept
  either tone sense.
- **The "BCH(10,7)" has minimum distance 2.** `BCHCheck` returns pass/fail
  only; real correction is DX/RX redundancy, not merged yet.
- **Sync is a repeating character, not a flag.** Two phasing DX characters
  20 bits apart establish grid and polarity; the first non-phasing DX
  symbol is the format specifier.
- **`Decode` parses the operational core and keeps the rest.** Distress
  alerts get nature, position and time; routine calls get target and source
  MMSI; type-of-call and frequency fields stay on `RawSymbols`.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| Front end | FM → resample to 9600 → FFSK 1300/2100 → MM → EMA slicer | `internal/radio/dsc/ffsk/receiver.go` (`processChunk`, `feedSymbol`) |
| Character code | 7 data + 3 check bits, CRC-3 g(x) = x³+x+1, detect only | `internal/radio/dsc/bch.go` (`BCHEncode`, `BCHCheck`) |
| Framing | phasing hunt → 20-bit DX grid → EOS 117/122/127 | `internal/radio/dsc/receiver/receiver.go` (`hunt`, `collect`, `finish`) |
| Sequence parser | format, category, MMSIs, nature, position, time | `internal/radio/dsc/dsc.go`, `codec.go` (`Decode`, `decodePosition`) |
| Bus → storage → REST → panel | `KindDSCMessage` → `dsc_log` → `/api/v1/dsc/messages` → `/dsc` | `internal/storage/dsclog.go`, `internal/api/handlers_dsc.go`, `web/src/panels/DSC.tsx` |
| End-to-end test | modulated distress → bus event | `dsc/ffsk/receiver_test.go` (`TestEndToEndDistressDecode`) |

## In this post

- **Channel 70** — what DSC is for.
- **FFSK at 1300/2100 Hz** — the front end, and why this slicer tracks DC.
- **Ten-bit characters** — a check code that only detects.
- **Phasing and the DX grid** — hunting, locking, polarity and EOS.
- **Format, category, MMSI and position** — what `Decode` fills in.
- **From bus event to the `/dsc` panel** — the eleven places.

## Channel 70: the calling layer under marine voice

[DSC]({{ '/reference/dsc/' | relative_url }}) is the digital layer of the
GMDSS: a short FFSK burst on channel 70 that says *who* is calling *whom*,
at what priority, and — for a distress alert — where and what is wrong. A routine call names the working voice channel on
[marine VHF]({{ '/reference/marine-vhf/' | relative_url }}), so DSC is the
*alerting* layer that says when to listen — the role
[RF Scope Part 3]({{ '/blog/tutorials/rf-scope-03-protocol-hierarchy/' | relative_url }})
gives control channels.

The decoder landed in two slices — #433 the protocol layer and scaffolding,
#448 the FFSK front end — and `docs/dsc.md` is its
[operator page]({{ '/dsc.html' | relative_url }}). One scope note: the front
end is **VHF only**. `BaudHz` = 1200, and the source says "HF DSC uses 100
baud, not handled by this frontend".

## FFSK at 1300/2100 Hz — and why this slicer tracks DC

The front end is the MDC1200 receiver's shape with DSC's constants — the
family resemblance running through the series since
[Part 2]({{ '/blog/deep-dives/beyond-voice-02-afsk-ffsk-fundamentals/' | relative_url }}):

```go
// internal/radio/dsc/ffsk/receiver.go (shape) — MarkHz 1300, SpaceHz 2100
func (r *Receiver) processChunk(chunk []complex64) {
    r.demodBuf = r.fm.Process(r.demodBuf, chunk)              // demod.FM
    r.rsmpBuf  = r.rsmp.Process(r.rsmpBuf, r.demodBuf)        // L/M → 9600
    r.ffskBuf  = r.ffsk.Discriminate(r.ffskBuf, r.rsmpBuf)    // demod.FFSK
    r.symBuf   = r.mm.Process(r.symBuf, r.ffskBuf)            // 1 sample/bit
    for _, s := range r.symBuf { r.feedSymbol(s) }
}
```

`demod.FFSK` is the shared [FFSK]({{ '/reference/ffsk/' | relative_url }})
discriminator: mix the audio down by the tones' midpoint, low-pass filter,
FM-discriminate, sign arranged so mark reads 1. Here the pair is
1300/2100 Hz rather than CCIR's 1200/1800, at 9600 sps into the same
[Mueller-Müller]({{ '/reference/mueller-muller-timing-recovery/' | relative_url }})
loop at `mmGain` 0.05.

The slicer is where DSC parts company with Part 9. AIS sliced at a fixed
zero because NRZI plus bit stuffing made tone sense irrelevant and bounded
every run. **DSC has no NRZI**: the sliced bit *is* the data bit, so tone
sense matters and nothing bounds a run of one level.
`feedSymbol` therefore tracks the discriminator mean with a slow EMA
(`r.meanEMA += (s - r.meanEMA) / 64`) and slices 1 when the sample sits
above it.

The source is explicit about the division of labour: "Tone-sense ambiguity
is resolved by the orchestrator's dual-polarity phasing hunt, so a fixed
mark=1 slicer is sufficient here." The slicer removes bias; the framer
decides which way is up — the
[MDC1200]({{ '/blog/deep-dives/beyond-voice-03-mdc1200-template-decoder/' | relative_url }})
receiver's approach, and the reason the front end stays polarity-agnostic.

## Ten-bit characters and a check code that only detects

Every DSC character is ten bits: seven data bits then three check bits. The
spec calls the code BCH(10,7); GopherTrunk implements the check bits as a
**CRC-3** with generator g(x) = x³ + x + 1 (`0x0B`):

```go
// internal/radio/dsc/bch.go (shape): codeword = data<<3 | (data<<3) mod g(x)
for i := 9; i >= 3; i-- {
    if dividend&(1<<uint(i)) != 0 { dividend ^= 0xB << uint(i-3) } // g = 1011
}
func BCHCheck(codeword uint16) (data uint16, ok bool) // ok: syndrome == 0
```

The load-bearing fact is in the file's own comment: the code's **minimum
Hamming distance is 2**. A single flipped bit is reliably *detected* —
`TestBCHFlagsSingleBitErrorsAsSyndromeMismatch` walks every bit of every
codeword — but cannot be *corrected*, because legal codewords sit one bit
apart (`encode(69)` and `encode(85)` differ in two bits). `BCHCheck`
therefore returns only `(data, ok)`.

That redundancy is how DSC really corrects errors: each character goes out
twice, DX then RX. This slice **reads DX only and drops the RX copies** —
`docs/dsc.md` calls the merge "a yield-improving follow-up" — so a sequence
with a failed DX character still publishes (unless `drop_bad_fcs`), counted
in `SequencesBad` and flagged on the panel.

One more thing to say plainly. The reference page describes the M.493 check
bits as a count of the zeros in the seven data bits; the code computes a
CRC-3. Those are different constructions, and only a channel-70 capture can
say which the air agrees with — the `receiver` package's own doc comment
lists wire bit order, tone sense and DX/RX offset as things to confirm
"before relying on field decodes." A modulator and demodulator that agree
with each other prove consistency, not correctness.

## Phasing, the DX grid and the end of sequence

There is no flag byte. A DSC sequence opens with a dot pattern (alternating
bits for timing) and a **phasing** run in which the DX slots repeat the
character 125. Because DX and RX interleave at the 10-bit cadence, DX
characters land exactly 20 bits apart — `dxStride` — and that repetition is
what the receiver locks onto:

```go
// internal/radio/dsc/receiver/receiver.go (shape) — dxStride 20, phasingDX 125
func (r *Receiver) hunt() {
    pol, ok := r.windowIsPhasingDX()   // BCHCheck(reg) or BCHCheck(^reg) == 125
    if !ok { return }
    if r.dxSeen && r.lastDXPol == pol && r.nbits-r.lastDXBit == dxStride {
        r.st, r.inverted, r.lockBit = stateLocked, pol, r.nbits // locked
        return
    }
    r.dxSeen, r.lastDXBit, r.lastDXPol = true, r.nbits, pol
}
```

`Push` slides every bit into a 10-bit window and, while hunting, asks
whether it decodes to the phasing character under **either** polarity. Two sightings at the same polarity exactly
one stride apart lock the grid; `inverted` remembers which sense won and
`collect` complements every later window accordingly
(`TestDecodeInvertedPolarity`).

Locked, `collect` samples only at DX boundaries, skipping phasing characters
until the first non-phasing DX symbol — the **format specifier** — and
abandons the lock if that symbol fails its check. From there it appends
symbols, counting failures in `badInSeq`, until an end-of-sequence character
arrives — 117 (acknowledge required), 122 (acknowledge) or 127 — or
`maxSeqSyms` (40) is exceeded and the runaway is dropped
(`TestNoEOSDoesNotPublish`: without an EOS, nothing reaches the bus).

<figure class="lab-figure">
<svg viewBox="0 0 680 250" width="680" height="250" role="img" aria-label="Ten-bit DSC character cells alternating DX and RX: phasing DX cells carry 125 twenty bits apart under a dxStride bracket, RX cells are greyed as dropped, later DX cells carry the format specifier and the EOS; one cell expands into seven data bits and three check bits.">
  <g font-size="9" text-anchor="middle">
    <rect x="20" y="40" width="90" height="28" fill="none" stroke="currentColor"/><text x="65" y="58" fill="currentColor">DX 125</text>
    <rect x="110" y="40" width="90" height="28" fill="none" stroke="var(--fg-muted)"/><text x="155" y="58" fill="var(--fg-muted)">RX (dropped)</text>
    <rect x="200" y="40" width="90" height="28" fill="none" stroke="currentColor"/><text x="245" y="58" fill="currentColor">DX 125</text>
    <rect x="290" y="40" width="90" height="28" fill="none" stroke="var(--fg-muted)"/><text x="335" y="58" fill="var(--fg-muted)">RX (dropped)</text>
    <rect x="380" y="40" width="90" height="28" fill="none" stroke="var(--accent)"/><text x="425" y="58" fill="var(--accent)">DX 112</text>
    <rect x="560" y="40" width="90" height="28" fill="none" stroke="var(--accent)"/><text x="605" y="58" fill="var(--accent)">DX 127</text>
    <text x="515" y="58" fill="var(--fg-muted)">…</text>
  </g>
  <line x1="65" y1="30" x2="245" y2="30" stroke="var(--accent)"/>
  <line x1="65" y1="26" x2="65" y2="34" stroke="var(--accent)"/><line x1="245" y1="26" x2="245" y2="34" stroke="var(--accent)"/>
  <text x="155" y="22" text-anchor="middle" fill="var(--accent)" font-size="9">dxStride = 20 bits, same polarity → lock</text>
  <text x="605" y="84" text-anchor="middle" fill="var(--accent)" font-size="8">EOS → finish()</text>
  <g font-size="9" text-anchor="middle">
    <rect x="200" y="140" width="200" height="28" fill="none" stroke="currentColor"/><text x="300" y="158" fill="currentColor">7 data bits (symbol 0..127)</text>
    <rect x="400" y="140" width="90" height="28" fill="none" stroke="var(--accent)"/><text x="445" y="158" fill="var(--accent)">3 check bits</text>
  </g>
  <text x="340" y="192" text-anchor="middle" fill="currentColor" font-size="9">check = (data « 3) mod (x³ + x + 1) — distance 2: BCHCheck detects, never corrects</text>
  <text x="340" y="216" text-anchor="middle" fill="var(--fg-muted)" font-size="9">hunt → collect every 20th bit → finish: Decode → KindDSCMessage</text>
</svg>
<figcaption>DX and RX characters interleave, so DX symbols land every 20 bits; the hunt locks on two agreeing phasing characters, <code>collect</code> samples the DX grid to the EOS, and each character's three check bits can only say pass or fail.</figcaption>
</figure>

## Format, category, MMSI and position: what `Decode` fills in

By the time symbols reach `dsc.Decode` each is a 7-bit value, and the first
is the **format specifier**. `Decode` never returns an error — malformed runs
come back as `FormatUnknown` with `RawSymbols` preserved — and its dispatch
is deliberately narrow:

| Format | Code | Category | Code |
|---|---|---|---|
| Distress | 112 | Distress | 112 |
| All ships (and distress relay) | 116 | Urgency | 110 |
| Group | 114 | Safety | 108 |
| Individual | 120 | Routine | 100 |
| Geographic area · Automatic individual | 102 · 123 | | |

The distress **nature** symbol runs 100–112 — fire, flooding, collision,
grounding, listing, sinking, disabled, undesignated, abandoning ship,
piracy, man overboard (110), EPIRB emission (112) — named by `NatureString`.
A **distress** sequence carries the sender's own MMSI immediately, then one
nature symbol, five position symbols and two time symbols read as HH:MM;
`Decode` forces `Category = CategoryDistress`. Every other
format reads a five-symbol target address, a category symbol, then the
five-symbol self-MMSI — and stops. Type of call and working frequency are
**not parsed**; they stay on `RawSymbols`, and `WorkingChannel` is never
filled. `FormatDistressRelay` shares code point 116 with all-ships and is
told apart only by category.

The codecs are pairs of decimal digits. `decodeMMSI` takes five symbols,
each 0..99, and keeps nine digits (the tenth is a format extension). `decodePosition` reads ten digits as
`Q DD MM DDD MM`: a quadrant (0 NE, 1 NW, 2 SE, 3 SW) applied as sign to
degrees + minutes/60 (`TestDecodePositionApplyQuadrantSign`), with the all-9s
sentinel collapsing to `HasPosition = false`.

`TestEndToEndDistressDecode` ties it together: it modulates the run
`112, 36 60 53 20 90, 100, 3 74 81 22 24, 14 25, 127` — distress, MMSI
366053209, fire, a position, 14:25 UTC, EOS — with `demod.ModulateFFSK`
behind 240 dotting bits and 12 phasing pairs, and expects a `distress`
event for that MMSI.

## From bus event to the `/dsc` panel

The [Part 1]({{ '/blog/deep-dives/beyond-voice-01-eleven-place-pattern/' | relative_url }})
pattern, walked for DSC:

1. **Front end + framer**: `dsc/ffsk` and `dsc/receiver`, above.
2. **Bus event**: `events.KindDSCMessage` (`"dsc.message"`), payload
   `storage.DSCMessage`.
3. **Storage**: `storage.DSCLog` → `dsc_log`, indexed on `(received_at)`
   and `(self_mmsi, received_at)`.
4. **REST**: `GET /api/v1/dsc/messages?limit=N` (default 200, max 5000).
5. **Panel**: `DSC.tsx` polls every 5 s; `categoryTone` paints distress
   `err` and urgency `warn`; a positioned distress alert becomes a red,
   oversized marker on the shared `PositionMap`.
6. **Config**: `dsc.channels[]` — `serial`, `frequency_hz`, `drop_bad_fcs`.
7. **Field help**: `DSCConfig.*` / `DSCChannelConfig.*` in `fieldmeta.go`.
8. **Route tests**: `"/dsc"` in `web/src/nav/registry.test.ts`.
9. **Preflight**: `"dsc"` in the storage.path warning.
10. **Retention**: `dsc_log` in `storage.decoderLogTables`.
11. **config.example.yaml**: the commented `dsc:` block.

The daemon spawns one `ffsk.Receiver` per entry with the same closure and
warnings as AIS (`dsc: SDR not found, skipping receiver`,
`dsc: SetCenterFreq failed`, `dsc: open IQ failed`); `Stats()` gives
`SequencesIn`, `SequencesBad`, `SequencesEmit`.

Verification status, without softening: every DSC test is synthetic, and
the fixture's `buildWireBits` is the decoder's own idea of the wire. No channel-70 capture is committed.
That is synthetic-verified only in
[From Spec to Shipping Part 10]({{ '/blog/deep-dives/from-spec-to-shipping-10-the-on-air-gate/' | relative_url }})'s
sense, and the check-code construction is one of the things a capture has to
settle.

### How DSC shaped the Go code

- **Detect-only codes return `(data, ok)`, never a corrected value.**
  `BCHCheck` refuses to guess; `badInSeq` carries the doubt to the operator.
- **Polarity is the framer's job.** The slicer only removes bias; the hunt
  tests both senses and records `inverted` once.
- **Bounded state machines.** `maxSeqSyms` and the noise-before-format abort
  always return the receiver to hunting.
- **Parse the operational core, preserve the rest.** `Decode` fills the
  fields that drive alerts and the map; unparsed tails stay on `RawSymbols`.

## Where this goes next

Both marine protocols were narrowband FSK through a discriminator. LoRa is
nothing of the kind: each symbol is a chirp across the whole channel,
recovered with a dechirp and an FFT.
[Part 11]({{ '/blog/deep-dives/beyond-voice-11-lora-chirp-receiver/' | relative_url }})
follows the pure-Go chirp receiver from the wideband IQ bank to a decrypted
LoRaWAN payload.

## FAQ

**What frequency does GopherTrunk decode DSC on?**
Marine VHF channel 70, 156.525 MHz, at 1200 baud; `dsc.channels` pins one
SDR to it. The HF DSC channels (2187.5, 8414.5 kHz and so on) use 100-baud
FSK, which the front end does not implement.

**Why can't GopherTrunk correct errors in a DSC character?**
Because the ten-bit character's check code has minimum Hamming distance 2:
a single bit flip is detected but cannot be located, since another valid
codeword sits one bit away. DSC corrects by sending each character twice;
GopherTrunk reads only the DX copy and flags failures.

**Which DSC formats does GopherTrunk decode?**
Distress (112), all ships (116), group (114), individual (120), geographic
area (102) and automatic individual (123). Distress alerts yield nature,
position and UTC time; the others yield target and source MMSI plus
category. Type-of-call and frequency fields stay raw.

**How does the receiver handle an inverted discriminator?**
The phasing hunt tests each 10-bit window against the phasing character 125
in both plain and complemented form. Whichever sense produces two matches
exactly 20 bits apart wins, and later characters are complemented to match —
no configuration needed.

**Has the DSC decoder been verified on real channel-70 traffic?**
No. All tests are synthetic, including the modulated end-to-end distress
decode, and no capture is committed. Wire bit order, tone sense, DX/RX
offset and the check-code construction all await a recorded ITU-R M.493
signal.

## Series navigation

**Part 10 of 14** · ←
[Part 9: AIS — GMSK, NRZI & Bit Stuffing]({{ '/blog/deep-dives/beyond-voice-09-ais-gmsk-nrzi/' | relative_url }})
· Next →
[Part 11: LoRa — A Pure-Go Chirp Receiver]({{ '/blog/deep-dives/beyond-voice-11-lora-chirp-receiver/' | relative_url }})
