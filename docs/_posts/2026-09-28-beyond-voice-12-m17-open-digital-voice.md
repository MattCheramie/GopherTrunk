---
title: "Beyond Voice, Part 12: M17 — The Open Digital Voice Link Layer"
description: "How GopherTrunk decodes M17's link layer without touching the voice: the 4FSK stream frame, the Golay-coded LICH that spreads a Link Setup Frame across six frames, base-40 callsigns, the CRC-16 with polynomial 0x5935 — and exactly where the decoder stops today, Codec 2 payload and web panel included."
category: deep-dives
keywords: m17 decoder, m17 link setup frame, m17 lich golay, m17 base-40 callsign, m17 4fsk 4800 baud, m17 stream frame sync 0xff5d, m17 crc 0x5935, open source digital voice sdr, codec 2 amateur radio, gophertrunk m17
tags: [beyond-voice, m17, amateur-radio, golay, link-layer, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "Beyond Voice"
series_part: 12
---

*Part 12 of **Beyond Voice**, a 14-part deep dive into everything GopherTrunk
decodes that is not a trunked voice call — the AFSK signalling formats,
paging, APRS, ADS-B, AIS, DSC, LoRa, M17 and the amateur digital-voice
family — and the one eleven-place wiring pattern that carries each of them
from a burst on the air to a row in the web console.
[Part 11]({{ '/blog/deep-dives/beyond-voice-11-lora-chirp-receiver/' | relative_url }})
built a chirp receiver from scratch; this part returns to the 4FSK family
for a protocol whose every layer is published: M17. It follows a Link Setup
Frame from the 4800-baud symbol stream through the Golay-coded LICH to a
row in `m17_log`, and it is exact about the boundary — GopherTrunk decodes
M17's link layer, not its voice.*

> **TL;DR:** `internal/radio/m17` decodes the M17 **link layer**: who is
> transmitting to whom, in what mode. The receiver
> (`internal/radio/m17/receiver`) is the P25 Phase 1 front end re-tuned —
> FM demod → resample to 48 kHz (`BaudHz` 4800 × `Oversample` 10) → C4FM
> matched filter with `rrcAlpha` 0.5 → Mueller-Müller timing → 4FSK slice →
> `symbolToDibit` (+1→00, +3→01, −1→10, −3→11). `Decoder.Push` hunts the
> stream sync `0xFF5D`, collects the 96-bit **LICH** (four Golay(24,12)
> codewords → 40 LSF bits + a 3-bit counter), skips the 272-bit payload, and
> `LICHAssembler` stitches six chunks into the 240-bit **LSF** — DST 48 ·
> SRC 48 · TYPE 16 · META 112 · CRC 16, base-40 callsigns, CRC-16 poly
> `0x5935`. Each LSF becomes `events.KindM17LinkSetup`, an `m17_log` row and
> `GET /api/v1/m17/linksetups`. **Not decoded:** the Codec 2 voice payload,
> the dedicated LSF frame (`0x55F7`), packet frames (`0x75FF`) — and there
> is **no `/m17` web panel** yet. Everything is spec-derived and
> synthetic-verified only ([#479](https://github.com/MattCheramie/GopherTrunk/issues/479)).

**Key takeaways**

- **M17 is the one digital-voice mode with no closed layer.** 4FSK, framing,
  FEC and the Codec 2 vocoder are all published, so the decoder's boundary
  is a milestone line, not a licensing line.
- **The LICH is a late-entry mechanism, and GopherTrunk uses it as the
  primary route.** Every stream frame carries one sixth of the LSF under
  Golay(24,12); six frames — about 240 ms — reassemble it, no Viterbi.
- **The decoder is a three-state machine that reads 96 bits per frame and
  skips 272.** `stHunt` → `stLICH` → `stSkip`.
- **Synthetic-green is the honest status.** `TestDecodeLSFFromLICHStream`
  proves the chain against GopherTrunk's own encoder; the source says the
  Golay matrix "should be confirmed against a capture".

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| 4FSK front end | FM → 48 kHz → RRC α=0.5 → MM timing → slice → dibits | `internal/radio/m17/receiver/receiver.go` (`feedSymbol`) |
| Sync hunt | 16-bit sliding window for `SyncStream` `0xFF5D` | `internal/radio/m17/decoder.go` (`Decoder.Push`) |
| LICH reassembly | 4 × Golay(24,12) → 40-bit fragment + counter → 6 chunks | `internal/radio/m17/lich.go` (`LICHAssembler.Push`) |
| LSF parse | base-40 DST/SRC, TYPE bits, META, CRC-16 `0x5935` | `internal/radio/m17/m17.go` (`ParseLSF`, `DecodeAddress`, `CRC16`) |
| Bus → storage → REST | `KindM17LinkSetup` → `m17_log` → `/api/v1/m17/linksetups` | `internal/events/bus.go`, `internal/storage/m17log.go`, `internal/api/handlers_m17.go` |
| Config, help, doctor | `m17.channels[]`, FieldMeta, storage.path WARN | `internal/config/config_peripherals.go`, `internal/configbuilder/fieldmeta.go`, `cmd/gophertrunk/preflight.go` |

## In this post

- **The friendly protocol** — why openness changes what a decoder can promise.
- **Two roads to the LSF** — the dedicated frame versus the LICH.
- **4FSK to bits** — the receiver constants and the dibit map.
- **The three-state decoder** — sync hunt, LICH collect, payload skip.
- **Inside the Link Setup Frame** — base-40 addresses, TYPE bits, the CRC.
- **Where the decoder stops** — Codec 2, packet mode, panel, capture gate.

## The friendly protocol

Every digital-voice mode this blog has covered ends at a vocoder somebody
else owns — IMBE, AMBE+2, the AMBE variants of
[Part 13]({{ '/blog/deep-dives/beyond-voice-13-dstar-ysf-dpmr/' | relative_url }}).
The [M17 Project]({{ '/reference/m17-project/' | relative_url }}) instead
published a 4FSK air interface at 4800 symbols per second, a framing layer,
a convolutional-plus-Golay FEC stack and the royalty-free
[Codec 2]({{ '/reference/codec2/' | relative_url }}) vocoder
([reference]({{ '/reference/m17/' | relative_url }})). No layer is closed
to a pure-Go receiver.

That changes the honest claim. For D-STAR the line between decoded and
not-decoded is partly a licensing line; for M17 it is purely a milestone
line, and the `internal/radio/m17` package doc draws it in its first
paragraph: the package "owns the *metadata* path" and "the Codec2 voice
payload is a later milestone". M17 runs in two link modes, both
visible in the LSF's TYPE field: **stream** for real-time voice and/or
data, **packet** for connectionless data. Callsigns are **base-40**
integers in 48 bits, so a source and a destination fit in twelve bytes.

## Two roads to the Link Setup Frame

The LSF is 240 bits — a 48-bit destination, a 48-bit source, a 16-bit TYPE,
a 112-bit META block and a 16-bit CRC
([reference]({{ '/reference/m17-link-information/' | relative_url }})) —
and it reaches a receiver two ways. Choosing between them is the package's
central design decision.

The first road is the **dedicated LSF frame** at the head of a
transmission, sync word `0x55F7`. Its payload is convolutionally coded and
punctured, so decoding it needs a K=5 Viterbi decoder and a depuncture
table, which the M17 package deliberately does not use.

The second road is the **LICH** — the Link Information CHannel in *every*
stream frame. Each frame carries one sixth of the LSF, 40 bits, under four
Golay(24,12) codewords, plus a 3-bit counter naming which sixth. Six
consecutive frames reassemble the whole LSF. This is M17's
[late-entry]({{ '/reference/late-entry/' | relative_url }}) mechanism, and
it needs only `framing.GolayDecode24_12`:

```go
// internal/radio/m17/lich.go (shape)
const LICHBits = 96  // four Golay(24,12) codewords per stream frame
const lichChunks = 6 // six 40-bit fragments per 240-bit LSF

func (a *LICHAssembler) Push(lich []byte) (LSF, bool) {
    var chunk uint64
    for i := 0; i < 4; i++ {
        data, errs := framing.GolayDecode24_12(cw) // cw: 24 wire bits
        if errs < 0 {
            a.golayErrors++
            return LSF{}, false // uncorrectable — drop this LICH
        }
        chunk = chunk<<12 | uint64(data&0x0FFF)
    }
    frag := chunk >> 8              // top 40 bits: the LSF fragment
    cnt := int((chunk >> 5) & 0x07) // next 3 bits: which sixth
    /* store frag at lsf[cnt*5:], mark have[cnt]; when all six: ParseLSF */
}
```

A stream frame is 40 ms, so six are ~240 ms. A LICH with an uncorrectable
codeword is simply dropped: the assembler keeps its fragments and waits for
that sixth to repeat, because the LSF repeats for the life of the
transmission.

## 4FSK to bits

The receiver is the C4FM chain from
[P25 End to End Part 1]({{ '/blog/deep-dives/p25-end-to-end-01-c4fm-carrier/' | relative_url }})
with M17's numbers dropped in:

```go
// internal/radio/m17/receiver/receiver.go (shape)
const (
    BaudHz      = 4800   // symbols/s
    Oversample  = 10     // matched-filter samples per symbol
    AudioRateHz = BaudHz * Oversample // 48 000
    rrcAlpha    = 0.5    // M17 RRC roll-off
    deviationHz = 2400.0 // peak (outer-symbol) deviation
    mmGain      = 0.05
)
```

`New` builds `demod.NewC4FM(Oversample, 8, rrcAlpha, dev)` where `dev` is
`2π·deviationHz/AudioRateHz` — the outer symbol's phase step per sample,
the slicer's scale — and a `MuellerMuller` loop at gain 0.05
([SDR Internals Part 7]({{ '/blog/deep-dives/sdr-internals-07-symbol-timing-sync-recovery/' | relative_url }})).
Two numbers differ from the P25 chain: roll-off **0.5**, not 0.2, and
**2400 Hz** deviation at the outer symbol. `feedSymbol` slices each symbol,
maps it with the M17 convention (`+1→00, +3→01, −1→10, −3→11`) and pushes
the two bits into the decoder MSB first. The package doc calls these
constants **nominal**: "like the POCSAG frontend, on-air calibration
against a real capture is a follow-up". `receiver_test.go` pins
construction and the dibit map (`TestSymbolToDibit`) — not decode yield on
real IQ.

## The three-state decoder

`m17.Decoder` consumes one bit at a time. `stHunt` shifts bits into a
16-bit register until it equals `SyncStream`; `stLICH` collects 96 bits for
the assembler; `stSkip` counts out the rest of the 368-bit payload — 272
bits of frame number and Codec 2 — and returns to the hunt:

```go
// internal/radio/m17/decoder.go (shape)
const (
    SyncStream uint16 = 0xFF5D // stream frame
    SyncLSF    uint16 = 0x55F7 // dedicated link-setup frame
    SyncPacket uint16 = 0x75FF // packet frame
)
const streamPayloadBits = 368 // 96 LICH + 272 skipped

case stHunt:
    d.reg = (d.reg << 1) | uint16(b)
    if d.reg == SyncStream {
        d.syncs.Add(1); d.st = stLICH; d.n = 0
    }
    // LSF / packet syncs are recognised but their payloads need the
    // convolutional path; the LICH route carries the same metadata.
```

That comment is the architecture in two lines: the LSF and packet sync
words are *known*, but a match on either does nothing. `Stats` exposes
`Syncs` and `LSFDecoded` — the counters to read first if a configured M17
channel stays silent.

<figure class="lab-figure">
<svg viewBox="0 0 680 250" width="680" height="250" role="img" aria-label="An M17 stream frame as three fields: the sync word 0xFF5D, a 96-bit Golay-coded LICH, and a skipped 272-bit Codec 2 payload; below, the LICH's 40-bit fragment and 3-bit counter, and six fragments assembling into the 240-bit Link Setup Frame.">
  <rect x="10" y="22" width="44" height="34" rx="4" fill="none" stroke="var(--accent)"/>
  <text x="32" y="43" text-anchor="middle" fill="var(--accent)" font-size="9">0xFF5D</text>
  <rect x="54" y="22" width="150" height="34" rx="4" fill="none" stroke="currentColor"/>
  <text x="129" y="37" text-anchor="middle" fill="currentColor" font-size="9">LICH — 96 bits</text>
  <text x="129" y="49" text-anchor="middle" fill="var(--fg-muted)" font-size="8">4 × Golay(24,12)</text>
  <rect x="204" y="22" width="466" height="34" rx="4" fill="none" stroke="var(--fg-muted)" stroke-dasharray="4 3"/>
  <text x="437" y="43" text-anchor="middle" fill="var(--fg-muted)" font-size="9">payload — 272 bits: frame number + Codec 2 — stSkip, not decoded</text>
  <line x1="129" y1="56" x2="129" y2="96" stroke="currentColor"/>
  <polygon points="125,94 129,102 133,94" fill="currentColor"/>
  <rect x="34" y="104" width="140" height="26" rx="4" fill="none" stroke="currentColor"/>
  <text x="104" y="121" text-anchor="middle" fill="currentColor" font-size="9">LSF fragment · 40 bits</text>
  <rect x="174" y="104" width="50" height="26" rx="4" fill="none" stroke="var(--accent)"/>
  <text x="199" y="121" text-anchor="middle" fill="var(--accent)" font-size="9">cnt · 3</text>
  <text x="129" y="146" text-anchor="middle" fill="var(--fg-muted)" font-size="8">Golay-decoded 48 bits: fragment + which sixth (0..5)</text>
  <line x1="224" y1="117" x2="300" y2="117" stroke="currentColor"/>
  <polygon points="298,113 306,117 298,121" fill="currentColor"/>
  <text x="264" y="110" text-anchor="middle" fill="var(--fg-muted)" font-size="8">× 6 frames</text>
  <rect x="310" y="96" width="120" height="30" rx="3" fill="none" stroke="var(--accent)"/>
  <text x="370" y="115" text-anchor="middle" fill="var(--accent)" font-size="9">DST 48 · SRC 48</text>
  <rect x="430" y="96" width="240" height="30" rx="3" fill="none" stroke="currentColor"/>
  <text x="550" y="115" text-anchor="middle" fill="currentColor" font-size="9">TYPE 16 · META 112 · CRC 16</text>
  <text x="490" y="146" text-anchor="middle" fill="var(--fg-muted)" font-size="8">240-bit LSF · base-40 callsigns · CRC-16 poly 0x5935 → ParseLSF → KindM17LinkSetup → m17_log</text>
  <text x="340" y="196" text-anchor="middle" fill="currentColor" font-size="9">six frames ≈ 240 ms: a receiver joining mid-transmission still learns who is calling whom</text>
</svg>
<figcaption>The LICH route: 96 Golay-coded bits read per stream frame, 272 skipped, six 40-bit fragments stacked into the Link Setup Frame — no Viterbi decoder in the path.</figcaption>
</figure>

## Inside the Link Setup Frame

`ParseLSF` takes the 30 assembled bytes and never errors — a bad CRC sets
`CRCOK=false` while the fields stay populated, so a marginal decode is shown
flagged rather than swallowed:

```go
// internal/radio/m17/m17.go (shape)
type LSF struct {
    Dst, Src string // callsign / "BROADCAST" / "#RESERVED"
    Meta     []byte // 14-byte META block
    CRCOK    bool
    Stream   bool   // TYPE bit 0: stream (vs packet) mode
    DataMode uint8  // TYPE bits 1..2: 1 = data, 2 = voice, 3 = voice+data
    CAN      uint8  // TYPE bits 7..10: channel access number
}
```

`DecodeAddress` treats the 48-bit value as a base-40 integer over the
alphabet `" ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-/."`, taking `addr % 40`
as the next character, dividing, and reversing at the end. Zero decodes to
the empty string, all-ones to `BROADCAST`, and anything above 40⁹
(`0xEE6B28000000`) to `#RESERVED`. The CRC is polynomial `0x5935`, initial
value `0xFFFF`, MSB-first, no final XOR, over DST through META.
`TestCRC16DetectsCorruption` flips one bit and asserts the check fails;
`TestDecodeLSFFromLICHStream` pushes six encoded LICH frames through the
decoder and asserts `AB1CDE` → `M17-USA` decodes with `CRCOK` true and
`DataMode` 2.

The receiver's `publish` then builds a `storage.M17LinkSetup` — `Src`,
`Dst`, `Mode` (`ModeString`: voice / data / voice+data / packet), `CAN`, hex
`Meta`, `CRCOK`, a display `Body` — and publishes `events.KindM17LinkSetup`
(`m17.linksetup` over SSE). `storage.M17Log`, one instantiation of the
generic `eventLog[T]` writer, lands rows in `m17_log`, indexed on
`received_at` and `src` and swept by `retention.log_days`
([Recording Part 10]({{ '/blog/deep-dives/recording-streaming-10-call-log-sqlite/' | relative_url }}));
`GET /api/v1/m17/linksetups?limit=N` reads them back newest first.

## Where the decoder stops

Four things are not decoded today, each a different kind of gap.

**The Codec 2 voice payload.** The 272 bits after the LICH are consumed by
`stSkip` and discarded. Decoding them needs the K=5 Viterbi plus depuncture
path and a pure-Go Codec 2 port that does not exist in the tree; the
vocoder registry from
[SDR Internals Part 12]({{ '/blog/deep-dives/sdr-internals-12-voice-coding-vocoders/' | relative_url }})
has no Codec 2 entry.

**The dedicated LSF frame and packet mode.** `SyncLSF` and `SyncPacket` are
recognised; their payloads need the convolutional decoder the package
omits. `docs/m17.md` defers the standalone LSF decode "since the LICH route
already covers it", but packet frames carry data the LICH never sees, so
packet M17 is invisible beyond its sync word.

**The web panel.** This is the eleven-place pattern from
[Part 1]({{ '/blog/deep-dives/beyond-voice-01-eleven-place-pattern/' | relative_url }})
with a hole in it. M17 has the config struct, FieldMeta help, a Config
Builder section (`web/configbuilder/src/sections/M17.tsx`), the
`config.example.yaml` block, the doctor entry, bus kind, storage table,
retention membership and REST route — but `web/src/nav/registry.ts` has no
`/m17` entry, `App.tsx` mounts no M17 panel, and the `web.tabs` key list
has no `m17`. Decoded LSFs are reachable over REST and nowhere in the
console —
[Part 14]({{ '/blog/deep-dives/beyond-voice-14-adding-a-decoder-checklist/' | relative_url }})'s
example of a decoder that stopped two places short.

**The capture.** Everything is transcribed from the specification and
validated against GopherTrunk's own synthetic encoder; the package doc says
the Golay matrix "should be confirmed against a capture", and
`docs/decoder-capture-needs.md` says that capture **must be IQ**. This is the
[self-consistent trap]({{ '/blog/solution-postmortem/from-the-issue-tracker-20-self-consistent-trap/' | relative_url }})
in its mildest form: nothing is known wrong, but the passing tests share an
author with the code. The status is **synthetic-verified only**.

### How the LICH route shaped the Go code

- **One decoder, no Viterbi dependency.** The LICH route kept
  `internal/radio/m17` to three files and one framing import.
- **Never-erroring parse, flagged instead.** `ParseLSF` populates every
  field and sets `CRCOK`, so marginal decodes stay visible.
- **Per-frame drop, per-sixth retry.** An uncorrectable Golay codeword costs
  one LICH, not the transmission.
- **Counters over logs.** `Decoder.Stats` and `Receiver.Stats` are the
  instruments the first field report will need
  ([From Spec to Shipping Part 13]({{ '/blog/deep-dives/from-spec-to-shipping-13-instruments-not-logs/' | relative_url }})).

## Where this goes next

M17 is the open end of the amateur digital-voice spectrum; the other end is
three modes built on AMBE.
[Part 13]({{ '/blog/deep-dives/beyond-voice-13-dstar-ysf-dpmr/' | relative_url }})
takes D-STAR's GMSK header and its 660-bit FEC chain, System Fusion's FICH
trellis, and dPMR Mode 3's CSBKs — three decoders at three distances from
on-air verification.

## FAQ

**Does GopherTrunk decode M17 voice?**
No. It decodes the M17 link layer — source and destination callsigns, stream
versus packet mode, data mode and channel access number — from the LICH in
every stream frame. The Codec 2 payload is skipped; decoding it needs the
convolutional path and a pure-Go Codec 2 port, both pending in `docs/m17.md`.

**Why decode the LICH instead of the Link Setup Frame itself?**
The dedicated LSF frame is convolutionally coded and punctured, so it needs
a Viterbi decoder. The LICH carries the same 240 bits one sixth per stream
frame under Golay(24,12), needing only a Golay decoder, and supports late
entry: a receiver joining mid-transmission has the full LSF after six
frames, about 240 milliseconds.

**How are M17 callsigns encoded on the air?**
As base-40 integers in a 48-bit field over the alphabet space, A–Z, 0–9,
hyphen, slash and dot. `DecodeAddress` repeatedly takes the value modulo 40
and divides. All-ones is `BROADCAST`, zero is the reserved empty address, and
values above 40⁹ decode as `#RESERVED`.

**Where do decoded M17 link setups show up?**
On the events bus as `m17.linksetup`, in the `m17_log` SQLite table, and at
`GET /api/v1/m17/linksetups?limit=N`. Both require `storage.path`;
`gophertrunk doctor` warns when `m17.channels` is configured without it.
There is no `/m17` web panel yet, so REST is the operator-facing surface.

**Has the M17 decoder been verified on air?**
Not yet. The sync words, LSF layout, CRC-16 polynomial 0x5935, base-40
alphabet and LICH structure come from the M17 specification and are proven
against GopherTrunk's own synthetic encoder. The exact Golay matrix, slicer
deviation and timing constants await an IQ capture of real M17 traffic, per
`docs/decoder-capture-needs.md`.

## Series navigation

**Part 12 of 14** · ←
[Part 11: LoRa — A Pure-Go Chirp Receiver]({{ '/blog/deep-dives/beyond-voice-11-lora-chirp-receiver/' | relative_url }})
· Next →
[Part 13: D-STAR, System Fusion & dPMR]({{ '/blog/deep-dives/beyond-voice-13-dstar-ysf-dpmr/' | relative_url }})
