---
title: "Beyond Voice, Part 6: POCSAG & FLEX — Paging Networks"
description: "How GopherTrunk decodes the two paging protocols that still carry fire, EMS and hospital dispatch: POCSAG's polarity-agnostic sync, BCH(31,21) codewords and frame-slot RIC arithmetic, the FLEX decoder that reuses the same BCH primitive bit-reversed for its one supported mode, and the wideband DDC bank that puts both on one dongle."
category: deep-dives
keywords: pocsag decoder go, flex paging decoder sdr, bch 31 21 codeword, pocsag sync codeword 0x7cd215d8, pocsag ric frame slot, flex 1600 bps decoder, pager_log sqlite, paging wideband ddc, sdr pager decoding, gophertrunk paging
tags: [beyond-voice, pocsag, flex, paging, bch, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "Beyond Voice"
series_part: 6
---

*Part 6 of **Beyond Voice**, a 14-part deep dive into everything GopherTrunk
decodes that is not a trunked voice call — the AFSK signalling formats, paging,
APRS, ADS-B, AIS, DSC, LoRa, M17 and the amateur digital-voice family — and the
one eleven-place wiring pattern that carries each of them from a burst on the
air to a row in the web console.
[Part 5]({{ '/blog/deep-dives/beyond-voice-05-two-tone-tone-out/' | relative_url }})
listened for pager *tones* in decoded audio. This part decodes the pager
*data*: the POCSAG batches that tell a crew what the call is, and the Motorola
FLEX frames that succeeded them — with a precise account of which FLEX mode
GopherTrunk decodes and which it only recognises.*

> **TL;DR:** `internal/radio/pager/pocsag` is a bit-in, page-out protocol
> layer. `Syncer.Push(bit)` slides a 32-bit window until it equals
> `SyncCodeword` `0x7CD215D8` **or its bit-inverse**, remembers the polarity,
> then XORs the next 512 bits into 16 codewords. `Decode` strips the trailing
> parity bit, runs `framing.BCHDecode31_21` (generator `0x769`, ≤ 2
> corrections) and splits the 21 info bits into address (18 + 2-bit function)
> or message (20 bits); `ReconstructRIC` shifts the address left 3 and ORs in
> the frame slot. Function B decodes numeric, everything else alpha, both
> LSB-first per symbol. `internal/radio/pager/flex` hunts `SyncMarker`
> `0xA6C6AAAA`, accepts **only mode code `0x870C` — 1600 bps, 2-level, one
> phase** — de-interleaves 88 words and BCH-decodes each through
> `FLEXBCHDecode32`, a 31-bit reversal around the POCSAG primitive. Both
> publish `events.KindPagerMessage` (`pager.message`) into `pager_log`,
> `GET /api/v1/pager/messages` and `/pagers`; `paging.wideband` puts several
> channels on one SDR through a 48 kHz `tuner.DDCBank`.

**Key takeaways**

- **Polarity is the classic FSK-over-FM trap, and the syncer eats it.** Both
  decoders match the sync word and its complement and correct every following
  bit, so an inverted discriminator costs nothing.
- **One BCH primitive, two bit orders.** FLEX's BCH(31,21) codeword is the
  bit-reverse of POCSAG's; reversal preserves Hamming distance, so
  `FLEXBCHDecode32` wraps `BCHDecode31_21` instead of a second code.
- **The address is not all in the codeword.** An address codeword carries 18
  bits; its batch position supplies the other three of the 21-bit RIC.
- **FLEX support is one mode, deliberately.** 1600 bps / 2-level / short
  addresses; the other speeds, 4-level, long addresses and multi-frame
  reassembly are recognised-and-skipped follow-ups, synthetic-verified only.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| Codeword FEC | BCH(31,21) `0x769` + trailing even parity | `framing/bch_pocsag.go`; `pocsag.Decode` |
| Sync + polarity | window == `0x7CD215D8` or its inverse; XOR the batch | `pocsag/syncer.go` (`Syncer.Push`) |
| Full RIC | `(address18 << 3) \| slot`, slot = `wordIdx / 2` | `pocsag/batch.go` (`ReconstructRIC`, `FrameSlot`) |
| Message text | numeric (function B) or alpha, both LSB-first per symbol | `pocsag/message.go` (`DecodeNumeric`, `DecodeAlpha`) |
| FLEX frame | marker `0xA6C6AAAA` → mode `0x870C` → FIW → 88-word phase | `flex/decoder.go` (`Decoder.Push`, `placeDataBit`) |
| FLEX FEC | reverse 31 bits into the POCSAG decoder and back | `framing/bch_flex.go` (`FLEXBCHDecode32`) |
| Bus → table → route → panel | `pager.message` → `pager_log` → `/api/v1/pager/messages` → `/pagers` | `events.KindPagerMessage`, `storage.PagerLog`, `handlePagerMessages` |
| One SDR, many channels | `paging.wideband` → `tuner.DDCBank`, 48 kHz per tap | `buildWidebandPagingGroup` (`daemon.go`) |

## In this post

- **The codeword and its two checks** — BCH(31,21) plus a parity bit.
- **Batches, slots and the polarity-agnostic syncer** — 0x7CD215D8 either way.
- **Numeric, alpha and the LSB-first quirks** — five digits per word.
- **FLEX: the same code, reversed** — and exactly one mode.
- **From receiver to panel** — the shared pager pipeline and the wideband bank.

## The codeword and its two checks

Everything in [POCSAG]({{ '/reference/pocsag/' | relative_url }}) is a 32-bit
[codeword]({{ '/reference/pocsag-codeword/' | relative_url }}): a flag bit,
20 data bits, 10 [BCH]({{ '/reference/bch-code/' | relative_url }}) parity
bits and one even-parity bit. `framing/bch_pocsag.go` models the 31-bit core —
21 info bits above 10 parity bits, generator
`g(x) = x¹⁰ + x⁹ + x⁸ + x⁶ + x⁵ + x³ + 1`, the constant `0x769` from CCIR
Recommendation 584 — and `BCHDecode31_21` finds the nearest valid codeword,
reporting the corrections or `-1` beyond two bits. `pocsag.Decode` wraps it:

```go
// internal/radio/pager/pocsag/codeword.go (shape)
body31 := raw >> 1                       // strip the trailing parity bit
corrected, errs := framing.BCHDecode31_21(body31)
cw.CorrectedErrors = errs
clean := framing.BCHEncode31_21(corrected)   // parity over what the air SHOULD have carried
cw.ParityOK = framing.BCH3121ParityBit(clean) == byte(raw&1)
flag := (corrected >> 20) & 1
if flag == 0 {                           // address: 18-bit RIC bits + 2-bit function
    cw.Address = (corrected >> 2) & 0x3FFFF
    cw.Func = Function(corrected & 0x3)
} else {                                 // message: 20-bit field
    cw.MessageBits = corrected & 0xFFFFF
}
```

Two fixed patterns short-circuit the decoder before BCH runs: the sync
codeword `0x7CD215D8` and the idle codeword `0x7A89C197`, which fills unused
slots and must be treated as skip. The parity re-check runs over the
*re-encoded* clean word, so `ParityOK` answers "did the parity position itself
flip" rather than double-counting an error BCH already fixed.

## Batches, slots and the polarity-agnostic syncer

A batch is the sync codeword plus sixteen codewords as eight frames of two
(`BatchBits = 544`). The frame a pager listens in is part of its identity — a
RIC's low three bits pick the slot — so the address codeword carries only the
high 18: `ReconstructRIC` returns `(address18 & 0x3FFFF) << 3 | slot`, with
`FrameSlot(wordIdx) = wordIdx / 2`. The syncer is where that arithmetic and
the polarity problem meet:

```go
// internal/radio/pager/pocsag/syncer.go (shape)
s.window = (s.window << 1) | uint32(bit)
if !s.locked {
    if s.window == SyncCodeword  { s.locked, s.polarity = true, 0; return nil }
    if ^s.window == SyncCodeword { s.locked, s.polarity = true, 1; return nil }
    return nil
}
s.batchBits[s.bitsInBatch] = bit ^ s.polarity   // correct every bit of the batch
s.bitsInBatch++
if s.bitsInBatch < BatchBits-CodewordBits { return nil }
pages := s.processBatch()                       // 16 codewords → page assembler
s.locked = false                                // hunt for the next sync
```

FSK over FM hands the slicer an inverted stream whenever the discriminator's
sense disagrees with the transmitter's — the "usual gotcha" the package doc
names — so the window matches both senses and records which won;
`TestSyncerHandlesPolarityInverted` feeds a complemented batch and expects the
same page. After the sixteenth codeword the syncer drops back to hunting, so a
lost sync costs one batch.

`processBatch` is the page assembler. An address codeword flushes any page in
progress and opens a new one with its RIC and function; message codewords
append; an idle codeword flushes; and an **uncorrectable codeword** flushes
too, because "the body is now suspect". `Flush()` releases the last page at
end of stream.

<figure class="lab-figure">
<svg viewBox="0 0 680 250" width="680" height="250" role="img" aria-label="The POCSAG syncer and codeword. A 32-bit sliding window is compared against the sync codeword and its bit inverse; the match sets the polarity and the next 512 bits are XORed with it into sixteen codewords. One codeword is drawn as flag, twenty data bits, ten BCH parity bits and an even-parity bit; an address's eighteen bits shifted left three plus the frame slot form the 21-bit RIC. A footer notes FLEX reverses the same code.">
  <rect x="12" y="30" width="150" height="34" fill="none" stroke="var(--accent)"/>
  <text x="87" y="44" text-anchor="middle" fill="var(--accent)" font-size="9">32-bit sliding window</text>
  <text x="180" y="40" fill="currentColor" font-size="9">== 0x7CD215D8 → polarity 0</text>
  <text x="180" y="56" fill="currentColor" font-size="9">^window == 0x7CD215D8 → polarity 1</text>
  <line x1="340" y1="47" x2="372" y2="47" stroke="currentColor"/>
  <rect x="372" y="30" width="296" height="34" fill="none" stroke="currentColor"/>
  <text x="520" y="44" text-anchor="middle" fill="currentColor" font-size="9">next 512 bits, each XOR polarity</text>
  <text x="520" y="57" text-anchor="middle" fill="var(--fg-muted)" font-size="8">16 codewords · slot = wordIdx / 2</text>
  <text x="12" y="100" fill="currentColor" font-size="10" font-weight="bold">one codeword (32 bits, MSB first)</text>
  <rect x="12" y="108" width="24" height="30" fill="none" stroke="currentColor"/>
  <text x="24" y="127" text-anchor="middle" fill="currentColor" font-size="8">flag</text>
  <rect x="36" y="108" width="300" height="30" fill="none" stroke="var(--accent)"/>
  <text x="186" y="127" text-anchor="middle" fill="var(--accent)" font-size="9">20 data bits · address: 18-bit RIC bits + 2-bit function</text>
  <rect x="336" y="108" width="150" height="30" fill="none" stroke="currentColor"/>
  <text x="411" y="127" text-anchor="middle" fill="currentColor" font-size="9">10 BCH parity (0x769)</text>
  <rect x="486" y="108" width="40" height="30" fill="none" stroke="currentColor"/>
  <text x="506" y="127" text-anchor="middle" fill="currentColor" font-size="8">parity</text>
  <text x="540" y="127" fill="var(--fg-muted)" font-size="8">BCH ≤ 2 fixes, else −1</text>
  <text x="12" y="176" fill="currentColor" font-size="10" font-weight="bold">RIC = (address18 &lt;&lt; 3) | slot</text>
  <text x="12" y="192" fill="var(--fg-muted)" font-size="9">18 bits from the codeword, 3 from where in the batch it landed</text>
  <text x="12" y="230" fill="var(--fg-muted)" font-size="9">FLEX twin: same BCH(31,21) with info in the LOW bits — FLEXBCHDecode32 reverses 31 bits into BCHDecode31_21</text>
</svg>
<figcaption>The syncer locks on the sync codeword in either polarity and corrects the whole batch; an address's RIC is completed by the frame slot it arrived in.</figcaption>
</figure>

## Numeric, alpha and the LSB-first quirks

POCSAG's payload formats are where on-air bit order bites. `DecodeNumeric`
takes each message codeword's 20-bit field as five 4-bit BCD symbols — but
POCSAG "sends the LSB first within each nibble", so `reverseNibble` flips each
before the CCIR 584 table `0123456789*U -)(` maps it (`0xB` is urgent `U`,
`0xC` space); trailing spaces are trimmed. `DecodeAlpha` concatenates every
field into one bit string and reads 7-bit characters that may straddle
codeword boundaries, again LSB-first, dropping the NUL pad and mapping
controls to `.`.

Which decoder runs is a heuristic in `flushPage`: function code `1` (B)
"traditionally carries numeric", `2` (C) alpha, and A/D "vary by network" so
they fall back to alpha, the printable-safe choice. `TestSyncerNumericEncodingForFunctionB`
pins the round trip: a function-B page whose nibbles encode `12345` decodes to
that string. The `Page` also carries `Corrected`, the total BCH fixes across
its codewords — the marginal-signal meter that rides to the panel.

## FLEX: the same code, reversed — and exactly one mode

[FLEX]({{ '/reference/flex/' | relative_url }}) is Motorola's scheduled
successor: 128 frames per 4-minute cycle, each announcing its own speed and
level. GopherTrunk's
[`Decoder`]({{ '/reference/flex-protocol-coding/' | relative_url }}) is a
four-state machine fed one wire bit at a time — `stHuntMarker`, `stSpeed`,
`stFIW`, `stData`. It hunts the 32-bit `SyncMarker` `0xA6C6AAAA` (and its
complement, setting `inverted`), reads the 16-bit mode code, and here is the
scope boundary in one line:

```go
// internal/radio/pager/flex/decoder.go (shape)
if code == speed1600x2 {   // 0x870C — 1600 bps / 2-level, the one supported mode
    d.st = stFIW
} else {
    d.reset()              // 1600/4, 3200/2, 3200/4: recognised, skipped cleanly
}
```

The frame-information word yields `cycle = (fiw>>4)&0x0F` and
`frame = (fiw>>8)&0x7F`; then 2816 data bits are de-interleaved on the fly —
bit `c` lands in word `((c>>5)&0xFFF8)|(c&7)` at position `(c>>3)&31`,
undoing the block interleave that spreads a fade into single errors. Each of
the 88 assembled words then goes through `FLEXBCHDecode32`:

```go
// internal/radio/framing/bch_flex.go
func FLEXBCHDecode32(cw uint32) (uint32, int) {
    data, errs := BCHDecode31_21(reverse31(cw & 0x7FFFFFFF))
    return data & 0x1FFFFF, errs
}
```

FLEX uses the *same* BCH(31,21) as POCSAG with the information in the **low**
bits; bit-reversal preserves Hamming distance, so the tested POCSAG decoder
does the work. `DecodePhase` then walks the structure multimon-ng's
`demod_flex.c` documents: word 0 is the Block Information Word, address words
occupy `[aoffset, voffset)`, and each address's Vector Information Word names
its type and message span. Alphanumeric words hold three 7-bit characters; the
first character of the first word is a skipped header and `0x03` ends the
message.

Say precisely what this does *not* do. Only short addresses map to a capcode
(`word − 0x8000` for words in `[0x8001, 0x1E0000]`). There is no 3200 or
6400 bps, no 4-level, no multi-phase, no long address, no multi-frame
reassembly — each a listed follow-up. And the code's own caveat: sync framing,
BCH bit order and capcode mapping are "validated end-to-end against a
synthetic encoder" (`TestDecodeAlphanumericFrame`, `TestDecodeInvertedPolarity`,
`TestBCHCorrectsBitErrors`), with real-capture calibration pending. By the
repo's
[definition of verified]({{ '/blog/deep-dives/from-spec-to-shipping-14-definition-of-verified/' | relative_url }}),
FLEX is synthetic-verified only. POCSAG's synthetic IQ test,
`TestReceiverDecodesSyntheticPage`, is `t.Skip`-ped pending a real fixture
under `samples/pocsag/`: protocol layer unit-tested, DSP front end not yet
capture-pinned.

## From receiver to panel

The DSP front ends are the series' simplest. `pocsag/receiver` runs
`demod.NewFM()` → `dsp.NewRealResampler(L, M, 16, 7.0)` down to `baud × 8` →
an **open-loop integrator** averaging eight samples per bit → a slicer against
a `1/64` EMA of the running mean → `Syncer.Push`. `BaudHz` defaults to 1200
and accepts 512 or 2400. There is no Mueller-Müller loop — `docs/pocsag.md`
names swapping one in as the thing to tune "once we have signal to calibrate
against". The FLEX receiver is the same chain at a fixed 1600 baud. Both
publish one shape:

```go
// internal/radio/pager/pocsag/receiver/receiver.go (shape)
msg := storage.PagerMessage{Protocol: "pocsag", RIC: p.RIC, Func: uint8(p.Func),
    Encoding: enc /* "numeric" | "alpha" */, Body: p.Text, Corrected: p.Corrected}
r.bus.Publish(events.Event{Kind: events.KindPagerMessage, Payload: msg})
```

`events.KindPagerMessage` is `"pager.message"`; `storage.PagerLog` inserts one
row per page into `pager_log`;
`handlePagerMessages` serves `GET /api/v1/pager/messages?limit=` (default 200)
or **503** with "set storage.path in config" when no log is wired;
`Pagers.tsx` polls every 5 s and tags rows `pocsag` or `flex`. The
configbuilder's `paging` section, `KnownUITabs["pagers"]`, the `/pagers`
route in `App.panels.test.tsx`, the `paging:` block in `config.example.yaml`
and `pager_log` in the retention sweeper's `decoderLogTables` finish the walk. Two storage guards exist because an
empty panel was a real report (#565): `warnPagingNeedsStorage` at daemon load
and the doctor preflight's "these decoders need it" list — though the
preflight counts only `pocsag` and `flex` entries, not `wideband`.

The wideband path is the one architectural addition. A `paging.wideband` group
tunes one SDR to `center_freq_hz` (or the channels' midpoint when 0) and runs
`tuner.NewDDCBank(sampleRate, pagingWidebandRateHz, 0.05)` — one NCO +
decimator per channel down to 48 kHz, the DDC bank
[wideband DMR]({{ '/blog/deep-dives/protocol-decoders-09-conventional-wideband/' | relative_url }})
uses — so FLEX on 153.0250 MHz and POCSAG on 153.3500 MHz share a stick. A
channel outside the 5%-guarded IQ window logs
`paging.wideband: channel offset out of band, skipping`; its siblings carry on.

### How paging shaped the Go code

- **Match both senses, then correct.** Polarity is a receiver property, not a
  config key: both `Push` methods accept the complemented sync and XOR every
  following bit.
- **Reuse a proven code through a bijection.** `reverse31` around
  `BCHDecode31_21` gives FLEX a tested decoder and keeps one BCH to trust.
- **Let uncorrectable mean "flush".** A `-1` from BCH ends the page in
  progress rather than appending garbage; `Corrected` rides to the panel.
- **Recognise more than you decode.** Unsupported FLEX modes reset the
  decoder cleanly, so the boundary of support is a branch, not a crash.

## Where this goes next

Pagers carry text; the next decoder carries positions to a map.
[Part 7]({{ '/blog/deep-dives/beyond-voice-07-aprs-ax25-location/' | relative_url }})
follows APRS from Bell 202 tones through NRZI, HDLC bit-stuffing and AX.25
addressing to Mic-E, which hides half a position in the destination callsign —
and to `internal/radio/location`, the NMEA core any protocol's GPS report can
feed.

## FAQ

**What POCSAG baud rates does GopherTrunk decode?**
512, 1200 and 2400 bps — set `baud_hz` per `paging.pocsag` entry (default
1200). The receiver resamples to eight samples per bit, integrates, slices
against a running mean and feeds `pocsag.Syncer`; the protocol layer is the
same at all three rates. Rate is configured, not auto-detected.

**Does GopherTrunk decode FLEX paging?**
One mode: 1600 bps, 2-level FSK, single phase, short addresses — mode code
`0x870C`. Other speed/level codes are recognised and skipped cleanly. Long
addresses, multi-frame reassembly, 3200/6400 bps and 4-level are documented
follow-ups, and the decoder is validated against a synthetic encoder only, not
yet a captured FLEX signal.

**Why does a POCSAG address codeword carry only 18 bits of the RIC?**
Because the other three bits are the frame slot. A pager sleeps except during
its assigned frame (the RIC's low bits), so the batch position implies them.
`ReconstructRIC(address18, slot)` returns `address18 << 3 | slot`, the slot
being the codeword's batch index divided by two.

**Why are my pager messages missing from the web panel?**
Pages persist and surface only through SQLite. Without `storage.path`,
`GET /api/v1/pager/messages` returns 503 and `/pagers` stays empty; the daemon
warns at startup (issue #565) and `doctor` lists paging among the decoders
that need `storage.path`.

**Can two paging channels share one SDR?**
Yes — a `paging.wideband` group tunes one dongle to a centre frequency and runs
a `tuner.DDCBank` with a 48 kHz tap per channel, any mix of POCSAG and FLEX.
Each channel must sit inside `center ± sample_rate/2` minus a 5% guard; one
outside is logged and skipped.

## Series navigation

**Part 6 of 14** · ←
[Part 5: Two-Tone Paging — Goertzel, Profiles & Tone-Out]({{ '/blog/deep-dives/beyond-voice-05-two-tone-tone-out/' | relative_url }})
· Next →
[Part 7: APRS, AX.25 & the Location Layer]({{ '/blog/deep-dives/beyond-voice-07-aprs-ax25-location/' | relative_url }})
