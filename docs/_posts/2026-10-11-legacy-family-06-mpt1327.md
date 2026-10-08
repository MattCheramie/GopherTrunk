---
title: "The Legacy Family End to End, Part 6: MPT 1327 — FFSK Codewords, CWSC Tolerance and BCH(64,48)"
description: "How GopherTrunk decodes an MPT 1327 control channel: 1200-baud FFSK through an FM discriminator, the 16-bit Codeword Synchronisation Code matched at a 2-bit Hamming tolerance, the 64-bit codeword's BCH(64,48) single-error correction, the Aloha / AHYC / GTC opcodes — and exactly which of those layers a real recording has confirmed."
category: deep-dives
keywords: mpt 1327 decoder, mpt1327 ffsk 1200 baud, codeword synchronisation code cwsc, mpt1327_cwsc_tolerance, bch 64 48 codeword, mpt 1327 go to channel gtc, aloha codeword, mpt1327_bch_mode, mpt 1327 sdr, gophertrunk legacy family end to end
tags: [legacy-family-end-to-end, mpt1327, ffsk, bch, trunking, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "The Legacy Family End to End"
series_part: 6
---

*Part 6 of **The Legacy Family End to End**, a 14-part deep dive through
the protocols the P25, DMR and TETRA series left out — the FM-era trunking
generation and the AMBE-era narrowband and amateur modes — with each
protocol placed honestly on one verification ladder.
[Part 5]({{ '/blog/deep-dives/legacy-family-05-ltr/' | relative_url }})
followed LTR, the system with no control channel at all. This part takes
MPT 1327, whose control channel is a stream of 64-bit codewords riding
1200-baud FFSK tones inside a narrowband FM channel — and the one legacy
decoder in the tree that has decoded a recording of a real transmitter.*

> **TL;DR:** `internal/radio/mpt1327` decodes the control channel in three
> layers. The receiver (`mpt1327/receiver`) runs `demod.FM` → `demod.FFSK`
> (mark 1200 Hz = 1, space 1800 Hz = 0) → `sync.MuellerMuller` at the
> 48 kHz `ddcTargetRateHz` and emits raw bits through a `BitSink`.
> `ControlChannel.Process` aligns on the 16-bit Codeword Synchronisation
> Code `1100010011010111` (`cwscPattern`) within `mpt1327_cwsc_tolerance`
> bit errors (default `cwscDefaultMaxErrors` = 2), then slices 64-bit
> codewords through `framing.BCHDecodeMPT1327`: 48 info bits, a 15-bit
> check from generator 0x6815 seeded 0x0001, one parity bit, single-error
> correction. `Ingest` locks on an Aloha or AHYC only after
> `mpt1327ProdMinConfirm` = 2 codewords sharing one Prefix, and republishes
> a GTC as a `trunking.Grant` with `Protocol "mpt1327"`. Rung: the
> codeword, CWSC and BCH layers are **capture-pinned on real audio** — the
> two committed `samples/mpt1327/*.mp3` recordings decode through the manual
> `samples/cmd/audio_smoketest` harness (7 cc.locked + 4 grants, SystemID
> 0x1fd7 on the second); the IQ receiver is synthetic-verified; and no
> daemon has followed a live call, because the pipeline installs no
> band-plan resolver.

**Key takeaways**

- **FFSK is audio-band, so an audio recording is real evidence here.**
  An MP3 of the discriminator output is the signal the second stage
  expects, not a lost constellation.
- **The sync match is loose because the codeword check is tight.** A
  2-of-16 Hamming tolerance accepts ~0.21 % of random windows; the
  BCH(64,48) check that must follow rejects ~2⁻¹⁵ of them.
- **Two codewords with one Prefix make a lock, never one.** A single
  recognised codeword is what a cross-protocol false parse looks like.
- **A grant without a frequency is dropped before any tuner moves.**
  `newMPT1327Pipeline` passes no `Resolver`, so every live GTC carries
  `FrequencyHz` 0 — the gap
  [Part 7]({{ '/blog/deep-dives/legacy-family-07-analog-voice-on-trunked-fm/' | relative_url }})
  measures.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| IQ → bits | FM discriminator → FFSK tone discriminator (1200 / 1800 Hz) → Mueller-Müller at 40 sps → 2-level slice | `internal/radio/mpt1327/receiver/receiver.go` (`SymbolRate`, `MarkHz`, `SpaceHz`) |
| Sync | 16-bit CWSC `1100010011010111` at Hamming ≤ `cwscDefaultMaxErrors` (2) | `process.go` (`cwscPattern`, `findCWSC`), key `mpt1327_cwsc_tolerance` |
| Codeword check | 48 info + 15-bit BCH (gen 0x6815, init 0x0001) + even parity; single-error correct | `internal/radio/framing/bch_mpt1327.go` |
| Fields | Type 1 · Prefix 7 · Ident 13 · Op 10 · Function 17; Kind = Function ≫ 13 | `codeword.go` (`CodewordFromBits48`), `opcodes.go` (`CodewordKind`) |
| Lock | Aloha / AHYC after 2 recognised codewords on one Prefix | `control.go` (`noteConfirmation`, `mpt1327ProdMinConfirm`) |
| Grant | GTC → `GroupID` = Prefix ≪ 16 ‖ Ident, `ChannelNum`, `FrequencyHz` via `Resolver` (none installed) | `control.go` (`publishGrant`), `bandplan.go` |
| Pins | `process_cwsc_test.go`, `process_bch_test.go`, `minconfirm_test.go`, `TestDaemonCCDecodesMPT1327` | `internal/radio/mpt1327/`, `cmd/gophertrunk/` |
| Real audio | two sigidwiki MP3s → ffmpeg → FFSK chain → `ControlChannel` (manual) | `samples/cmd/audio_smoketest/main.go`, `samples/README.md` |

## In this post

- **Tones inside an FM channel** — the receiver, and why 48 kHz is more than enough.
- **The 64-bit codeword** — fields, the BCH(64,48) primitive and its one documented ambiguity.
- **Finding the codeword boundary** — CWSC, tolerance, the false-lock arithmetic, the fallback.
- **From codeword to event** — Kind dispatch, the two-codeword lock, the grant's missing frequency.
- **Where MPT 1327 stands on the ladder** — what real audio proved, and what no capture has.

## Tones inside an FM channel

MPT 1327 ([reference]({{ '/reference/mpt-1327/' | relative_url }})) is the
1988 UK Code of Practice still carrying taxi, transport and utility fleets.
Its control channel is continuous 1200-baud CCIR FFSK — 1200 Hz for a
binary 1, 1800 Hz for a 0 — modulated as audio on a 12.5 kHz NBFM carrier
([FFSK]({{ '/reference/ffsk/' | relative_url }})). The information lives
in the audio band, so the receiver is the shortest in the family and an
audio recording of the discriminator is what the second stage expects.

```go
// internal/radio/mpt1327/receiver/receiver.go (shape)
const (
    SymbolRate = 1200.0
    MarkHz     = 1200.0 // binary 1
    SpaceHz    = 1800.0 // binary 0
)
r.disc    = r.fm.Process(r.disc, iq)            // demod.FM
r.tone    = r.ffsk.Discriminate(r.tone, r.disc) // demod.FFSK(rate, 1200, 1800)
r.symbols = r.clock.Process(r.symbols, r.tone)  // sync.MuellerMuller(sps, 0.05)
for i, s := range r.symbols { r.bits[i] = byte(r.ffsk.Slice(s)) }
r.bitSink(r.bits, r.bitBase)
```

`New` panics below 3600 Hz (twice the space tone) and below two samples
per symbol; the production pipeline hands it the C4FM family's 48 kHz
`ddcTargetRateHz` — 40 samples per symbol, far more than the loop needs,
but the DDC's anti-alias filter is the channel filter, and the shared
target keeps MPT on the down-converter
[Part 2]({{ '/blog/deep-dives/legacy-family-02-the-shared-skeleton/' | relative_url }})
described.

The load-bearing test is `TestReceiverRecoversTransmittedBits`: a 1010…
preamble and a 36-bit payload, FM- and FFSK-modulated by the test's own
`makeFMFFSKIQ`, must return bit-exact at the best alignment. Its comment
says why emitting *some* symbols was not enough: a plausible but wrong
2-level stream "would emit bits but never align to the payload" — the
failure it cites from
[#927](https://github.com/MattCheramie/GopherTrunk/issues/927). It is still
a self-generated fixture; the independent evidence comes at the end.

## The 64-bit codeword

An address codeword is 64 bits on the wire
([codeword reference]({{ '/reference/mpt-1327-codeword/' | relative_url }})):
a 48-bit information field, a 15-bit check and one parity bit.
`codeword.go` models the information field two ways — a legacy 38-bit
layout for older fixtures (`AssembleCodeword` / `ParseCodeword`, no `Op`)
and the spec-complete 48-bit layout the BCH path populates:

```go
// internal/radio/mpt1327/codeword.go — 48-bit information field
//  bit 47      Type     0 = address codeword, 1 = data codeword
//  bits 46..40 Prefix   7-bit area / system prefix
//  bits 39..27 Ident    13-bit radio or fleet identity
//  bits 26..17 Op       10-bit operation field
//  bits 16..0  Function 17-bit opcode-specific information
func (c Codeword) Kind() CodewordKind { return CodewordKind((c.Function >> 13) & 0xF) }
func (c Codeword) FunctionPayload() uint16 { return uint16(c.Function & 0x1FFF) }
```

The top four Function bits are the spec's Address Categorisation subfield;
`opcodes.go` names `KindAloha` 0x1 (ALH, the idle beacon), `KindAhoy` 0x2,
`KindAhoyChan` 0x3 (AHYC, whose payload the package treats as a system
identifier), `KindGoToChan` 0x4 (GTC, payload = channel number), `KindAck`
0x5, `KindDisconnect` 0x6, `KindData` 0x7 and `KindEmergency` 0xE.

The check is `framing.BCHEncodeMPT1327` / `BCHDecodeMPT1327`
([BCH]({{ '/reference/bch-code/' | relative_url }})), whose header credits
the layout to SDRTrunk's `CRCFleetsync` — FleetSync and MPT 1327 share it —
so the primitive is **reference-pinned**, not a spec transcription:

```go
// internal/radio/framing/bch_mpt1327.go (shape)
// bits 0..47 info · bits 48..62 15-bit check · bit 63 even parity over the 63-bit body
// g(x) = x^15 + x^14 + x^13 + x^11 + x^4 + x^2 + 1
const (
    bchMPT1327PolyHigh uint16 = 0x6815 // generator without the implicit x^15
    bchMPT1327Init     uint16 = 0x0001 // so the all-zero codeword is not trivially valid
)
var bchMPT1327Syndromes [48]uint16 // x^i mod g(x), built at init
```

Decode recomputes check and parity, matches a lone syndrome against the 48
info columns and 15 check positions, and corrects only if parity *also*
disagrees — a mismatched syndrome with matching parity means two or more
errors, `errs` −1, and the window is dropped. One ambiguity is documented
rather than hidden: a check bit at offset k shares syndrome `1 << k` with
info bit k for k = 0..14, so the decoder prefers the info-bit correction,
and `TestBCHMPT1327CorrectsAnySingleBitError` demands exact recovery for
positions 0..47 and 63 but only *detection* for 48..62. The comments drift
on the code's name — `BCH(63,38)` in the package header, `BCH(64,48,2)` at
the primitive; what runs is 48 + 15 + 1.

<figure class="lab-figure">
<svg viewBox="0 0 680 170" width="680" height="170" role="img" aria-label="Bit layout of one 64-bit MPT 1327 codeword as the BCH primitive sees it: a 48-bit information field split into Type 1 bit, Prefix 7, Ident 13, Op 10 and Function 17, where the top four Function bits are the Kind; then a 15-bit BCH check from generator 0x6815 seeded 0x0001, then one overall even parity bit. Below, the 16-bit CWSC 1100010011010111 precedes the first codeword of a message, followed by back-to-back 64-bit codewords.">
  <text x="340" y="14" text-anchor="middle" fill="currentColor" font-size="10" font-weight="bold">64-bit codeword · 48 info + 15 check + 1 parity</text>
  <rect x="20" y="28" width="12" height="34" fill="none" stroke="currentColor"/>
  <text x="26" y="76" text-anchor="middle" fill="var(--fg-muted)" font-size="8">T</text>
  <rect x="32" y="28" width="56" height="34" fill="none" stroke="currentColor"/>
  <text x="60" y="48" text-anchor="middle" fill="currentColor" font-size="9">Prefix</text>
  <text x="60" y="76" text-anchor="middle" fill="var(--fg-muted)" font-size="8">7</text>
  <rect x="88" y="28" width="104" height="34" fill="none" stroke="currentColor"/>
  <text x="140" y="48" text-anchor="middle" fill="currentColor" font-size="9">Ident</text>
  <text x="140" y="76" text-anchor="middle" fill="var(--fg-muted)" font-size="8">13</text>
  <rect x="192" y="28" width="80" height="34" fill="none" stroke="currentColor"/>
  <text x="232" y="48" text-anchor="middle" fill="currentColor" font-size="9">Op</text>
  <text x="232" y="76" text-anchor="middle" fill="var(--fg-muted)" font-size="8">10</text>
  <rect x="272" y="28" width="136" height="34" fill="none" stroke="currentColor"/>
  <rect x="272" y="28" width="32" height="34" fill="none" stroke="var(--accent)" stroke-dasharray="3 2"/>
  <text x="288" y="48" text-anchor="middle" fill="var(--accent)" font-size="8">Kind</text>
  <text x="356" y="48" text-anchor="middle" fill="currentColor" font-size="9">Function</text>
  <text x="340" y="76" text-anchor="middle" fill="var(--fg-muted)" font-size="8">17 · top 4 = Kind · low 13 = payload</text>
  <rect x="408" y="28" width="120" height="34" fill="none" stroke="var(--accent)"/>
  <text x="468" y="48" text-anchor="middle" fill="var(--accent)" font-size="9">BCH check</text>
  <text x="468" y="76" text-anchor="middle" fill="var(--fg-muted)" font-size="8">15 · g 0x6815 · init 0x0001</text>
  <rect x="528" y="28" width="12" height="34" fill="none" stroke="var(--accent)"/>
  <text x="534" y="76" text-anchor="middle" fill="var(--fg-muted)" font-size="8">P</text>
  <line x1="20" y1="100" x2="660" y2="100" stroke="var(--fg-muted)" stroke-dasharray="2 3"/>
  <rect x="20" y="112" width="128" height="30" fill="none" stroke="var(--accent)"/>
  <text x="84" y="131" text-anchor="middle" fill="var(--accent)" font-size="9">CWSC 1100010011010111</text>
  <rect x="148" y="112" width="200" height="30" fill="none" stroke="currentColor"/>
  <text x="248" y="131" text-anchor="middle" fill="currentColor" font-size="9">codeword 1 (always Address)</text>
  <rect x="348" y="112" width="200" height="30" fill="none" stroke="currentColor"/>
  <text x="448" y="131" text-anchor="middle" fill="currentColor" font-size="9">codeword 2 …</text>
  <text x="604" y="131" text-anchor="middle" fill="var(--fg-muted)" font-size="8">64 bits each</text>
</svg>
<figcaption>The 48-bit information field the BCH path recovers, with the Kind nibble at the top of Function; the CWSC precedes the first codeword of every message and is the stream's only fixed pattern.</figcaption>
</figure>

## Finding the codeword boundary

MPT 1327 has no frame sync in the P25 sense. It has the **Codeword
Synchronisation Code** — `1100010011010111`, 0xC4D7 MSB-first — before the
first codeword of every message, kept in `process.go` as a 16-entry byte
array. `findCWSC(buf, from, maxErrors)` returns the first window within
Hamming distance `maxErrors`:

```go
// internal/radio/mpt1327/process.go (shape)
const cwscDefaultMaxErrors = 2 // of 16 — matches commercial MPT 1327 receivers
for i := from; i <= end; i++ {
    errs := 0
    for j := 0; j < cwscBits; j++ {
        if buf[i+j]&1 != cwscPattern[j] { if errs++; errs > maxErrors { break } }
    }
    if errs <= maxErrors { return i, true }
}
```

The tolerance is a trade stated in numbers. With two errors allowed, a
random 16-bit window matches with probability C(16,0) + C(16,1) + C(16,2)
= 137 / 65536 ≈ 0.21 % (`TestFindCWSCFalsePositiveControl`, 65 536 random
windows, ceiling 0.5 %). But a sync match alone never produces an event:
the 64-bit window after it must pass `BCHDecodeMPT1327`, whose random-pass
rate the comments put near 2⁻¹⁵, keeping the per-bit-position false-lock
rate under 1e-7. `TestFindCWSCWithinTolerance` walks the table — zero, one
and two flips match at tolerance 2, three do not.

Alignment in `Process` is two-stage: the CWSC first, locking at the bit
after the sync and consuming at a fixed 64-bit stride (38 under `BCHOff`);
with no CWSC in the buffer, a fallback slide one bit at a time until a
window passes the check and parses as a recognised Address codeword.
Aligned, each failing frame counts against `maxConsecBad` = 8; at eight it
drops alignment and restarts one bit *after* the failed frame so it cannot
re-lock to the same wrong offset. The buffer tail survives chunk boundaries
(`TestProcessHandlesCodewordSpanningCalls`).

Both knobs are per-system config: `mpt1327_bch_mode` (`ParseBCHMode`:
empty or `on` → `BCHOn`, `off` → the 38-bit pre-stripped path) and
`mpt1327_cwsc_tolerance` (`ParseCWSCTolerance`: empty → 2, `exact` / `off`
/ `0` → exact, or an integer in [0, 15]). Both return `ok = false` on
nonsense so `newMPT1327Pipeline` warns instead of guessing.

## From codeword to event

`Ingest` is small: data codewords are dropped, unknown Kinds too under
`SetStrictValidation(true)` (`TestStrictValidationDropsUnknownKind`); Aloha
and AHYC are lock evidence, GTC is a grant.

The lock has a discipline the other legacy packages lack. A real control
channel streams codewords continuously, so demanding two costs nothing —
while one recognised codeword is what an off-channel P25 or DMR carrier
produces by chance. `noteConfirmation`
counts recognised Address codewords sharing one 7-bit Prefix; a different
Prefix restarts the count. `newMPT1327Pipeline` sets
`SetMinConfirm(mpt1327ProdMinConfirm)` = 2, while `New` zero-values to
lock-on-first for in-package fixtures. `minconfirm_test.go` pins the
shapes, including a 1,2,1,2,… alternation that never locks
(`TestMinConfirmAlternatingIdentityNeverLocks`).

`LockState` carries `FrequencyHz`, the AHYC payload as `SystemID`, and the
`Prefix`; `LockedNAC` returns the SystemID so the cchunt supervisor's
`trunking.LockedPayload` assertion accepts it. A GTC becomes a grant:

```go
// internal/radio/mpt1327/control.go (shape)
groupID := uint32(g.Prefix)<<16 | uint32(g.Ident) // (Prefix, Ident) is the called party
c.bus.Publish(events.Event{Kind: events.KindGrant, Payload: trunking.Grant{
    System: c.systemName, Protocol: "mpt1327",
    GroupID: groupID, FrequencyHz: freq, ChannelNum: g.Channel, At: c.now(),
}})
```

`freq` comes from `Options.Resolver` — `LinearBandPlan` or `TableBandPlan`
in `bandplan.go`. And here is the gap: **`newMPT1327Pipeline` passes no
`Resolver`**, and no `mpt1327_*` band-plan key exists, so a live GTC is
published with `FrequencyHz` 0 (`TestControlChannelGrantWithoutResolverHasZeroFreq`)
and `Engine.HandleGrant` drops it with `dropping grant with zero frequency`.
The codeword layer is complete; the follow-the-call layer is not wired.

## Where MPT 1327 stands on the ladder

**Synthetic, in CI.** `process_bch_test.go` feeds framing-encoded 64-bit
codewords to `Process(BCHOn)`: `TestProcessBCHOnDecodesEncodedCodeword`
(Aloha then GTC → lock and a channel-7 grant),
`TestProcessBCHOnCorrectsSingleBitError` (bit 35 flipped, still locks),
`TestProcessBCHOnDropsUncorrectableCodeword` (bits 20 and 33, no lock).
`TestDaemonCCDecodesMPT1327` boots the daemon on `demod.ModulateFFSK`
output — 100 Aloha codewords at 48 kHz on 169.2125 MHz. Every one shares
its encoder with the decoder: the
[self-consistent trap]({{ '/blog/solution-postmortem/from-the-issue-tracker-20-self-consistent-trap/' | relative_url }}).

**Real audio, by hand.** `samples/mpt1327/` holds two sigidwiki MP3s of a
423.6 MHz control channel. `samples/cmd/audio_smoketest` shells out to
ffmpeg for 8 kHz PCM, runs `demod.FFSK` → `sync.MuellerMuller` → slice —
step 2 onward of the production receiver — into a `ControlChannel` with
`SetBCHMode(BCHOn)`. `samples/README.md` records the result:
`MPT1327_423.6_1.mp3` gives 7 cc.locked and 1 grant, BCH-verified;
`MPT1327_423.6_2.mp3` gives 7 cc.locked and 4 grants with SystemID
`0x1fd7`. That is a real transmitter agreeing with the decoder — evidence
no round-trip can forge. Its limits: the harness bypasses `demod.FM`, sets
no `SetMinConfirm`, and is not a test — nothing in CI replays those files.

**On air, nothing.** No daemon has followed an MPT 1327 call, and none can
until a resolver is wired. `docs/decoder-capture-needs.md` files MPT 1327
under Tier 3 — "pipeline closed, captures only add robustness" — and
`samples/mpt1327/README.md` sets the bar: ≥ 60 s of IQ or audio, ≥ 95 % of
messages locking or granting at tolerance 2, a monotone sweep over
{0, 1, 2, 3}.

The honest rung: **codeword / CWSC / BCH capture-pinned on real audio
(manual harness); IQ receiver and daemon path synthetic-verified; call
follow unwired.**

### How MPT 1327 shaped the Go code

- **A bit-per-byte sync table.** `cwscPattern` is `[16]byte`; the matcher
  never packs the stream.
- **Two information layouts, one struct.** `Op` exists only on the 48-bit
  path (`TestLegacy38BitHelpersIgnoreOp`).
- **Config parsers return `ok`.** Empty string → production default;
  garbage → a warning.
- **Lock confirmation lives in the channel.** `minConfirm` defaults to
  lock-on-first; the connector raises it.

## Where this goes next

Every FM-era decoder in Parts 3–6 ends in a `trunking.Grant` with a channel
number, and a frequency only if someone resolved it.
[Part 7]({{ '/blog/deep-dives/legacy-family-07-analog-voice-on-trunked-fm/' | relative_url }})
follows that grant into the engine and the composer's `runFMChain`, and
measures which of the four protocols can reach it today.

## FAQ

**What does `mpt1327_cwsc_tolerance` do and why is the default 2?**
The Hamming distance `findCWSC` accepts against the Codeword
Synchronisation Code `1100010011010111`. Two of sixteen matches commercial
receivers on noisy air, and the codeword that follows must still pass
`BCHDecodeMPT1327`, so a loose sync does not loosen frame acceptance. Set
`0` or `exact` for pre-stripped fixtures.

**Does GopherTrunk correct errors in MPT 1327 codewords?**
One bit per 64-bit codeword. `framing.BCHDecodeMPT1327` recomputes the
15-bit check (generator 0x6815, seed 0x0001) and the parity; a single flip
changes both and the syndrome names its position. Two or more errors are
detected and the codeword dropped.

**Can an MP3 of an MPT 1327 control channel be decoded?**
Yes — FFSK rides audio-band tones, so FM-demodulated audio is what
`demod.FFSK` expects. The two `samples/mpt1327/*.mp3` recordings decode
through `samples/cmd/audio_smoketest` to 7 cc.locked and 1 or 4 grants.
NXDN, TETRA and DMR lose their constellations in FM demodulation.

**Why does an MPT 1327 lock need two codewords?**
Because one recognised codeword is what a cross-protocol false parse looks
like. `noteConfirmation` requires `mpt1327ProdMinConfirm` = 2 recognised
Address codewords sharing one Prefix before `cc.locked`; a real control
channel streams codewords continuously, so the latency cost is negligible.

**Is MPT 1327 verified on air?**
Not end to end. Codeword, CWSC and BCH layers are capture-pinned on real
demodulated audio; the IQ receiver and daemon pipeline are
synthetic-verified; no live call has been followed, because
`newMPT1327Pipeline` installs no resolver and every GTC grant carries
frequency 0.

## Series navigation

**Part 6 of 14** · ←
[Part 5: LTR — Subaudible Data on Every Repeater, No Control Channel]({{ '/blog/deep-dives/legacy-family-05-ltr/' | relative_url }})
· Next →
[Part 7: Analog Voice on Trunked FM — From a Grant to the Composer's FM Chain]({{ '/blog/deep-dives/legacy-family-07-analog-voice-on-trunked-fm/' | relative_url }})
