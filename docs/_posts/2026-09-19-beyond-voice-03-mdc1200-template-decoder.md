---
title: "Beyond Voice, Part 3: MDC1200 — The Template Decoder"
description: "The Motorola MDC1200 burst from sync word to web panel — the 40-bit sync hunted with five errors of tolerance and its complement for inverted polarity, the 16×7 column interleave, the reflected CRC-16, the opcode table, double packets, the receiver → bus → mdc1200_log → REST → panel chain later decoders were cloned from, and what its tests do not yet prove."
category: deep-dives
keywords: mdc1200 decoder, mdc1200 sync word 0x07092a446f, mdc1200 frame layout, ptt id ani decode, mdc1200 crc-16 ccitt reflected, 16x7 column interleave, mdc1200 opcodes emergency radio check, motorola signalling sdr, mdc1200 web panel, gophertrunk mdc1200
tags: [beyond-voice, mdc1200, motorola, framing, crc, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "Beyond Voice"
series_part: 3
---

*Part 3 of **Beyond Voice**, a 14-part deep dive into everything GopherTrunk
decodes that is not a trunked voice call, and the one eleven-place wiring
pattern that carries each of them from a burst on the air to a row in the web
console.
[Part 2]({{ '/blog/deep-dives/beyond-voice-02-afsk-ffsk-fundamentals/' | relative_url }})
ended with a 1200-baud stream of sliced NRZ bits and no idea what they mean.
This part gives them meaning for the first protocol in the series — Motorola's
MDC1200 — and, because FleetSync was cloned from it, the clearest look at the template
itself: a framer, a de-interleaver, a CRC, an opcode table, a bus event — and
the honest gap in what its tests prove.*

> **TL;DR:** An MDC1200 burst is a **40-bit sync word `0x07092A446F`**
> followed by **112 payload bits column-interleaved over a 16×7 grid**.
> `mdc1200/receiver.Receiver.Push` declares a burst when the register's
> Hamming distance to the sync is ≤ `gdThresh` (5) — or ≥ 35, the
> complemented sync of an inverted discriminator, in which case the payload
> is inverted back. `mdc1200.DecodeFrame`
> de-interleaves with `bits[j*16+i]`, packs LSB-first into 14 bytes — `op`,
> `arg`, big-endian unit ID, a little-endian **CRC-16/CCITT (reflected
> 0x8408, init 0, xorout 0xFFFF)** over the first four, and eight redundancy
> bytes GopherTrunk does not yet use — and labels `(op, arg)` from a
> best-effort table (PTT ID, Emergency, Radio check, Radio inhibit …). Ops
> `0x35`/`0x55` are two-block double packets. The burst publishes as
> `KindMDC1200Message` → `mdc1200_log` → `GET /api/v1/mdc1200/messages` →
> the `/mdc1200` panel. No on-air fixture is committed, so MDC1200 is
> **synthetic-verified**.

**Key takeaways**

- **The sync word is the protocol's only anchor.** No preamble check, no
  length field: forty bits with five errors of slack decide when a frame
  begins, and the complemented match makes tone sense irrelevant.
- **The interleave is a grid, not a table.** Reading `bits[j*16+i]` for
  `i < 16, j < 7` undoes a column interleave that spreads a fade across the
  codeword; the redundancy bytes it protects are captured but not decoded.
- **The CRC gates nothing by default.** A failed check publishes with
  `CRCOK=false` so the panel can flag marginal bursts; `drop_bad_crc` is the
  operator's choice for noisy channels.
- **The template's tests validate consistency, not correctness.** The
  round-trip helpers encode the layout the decoder reads, and no MDC1200 air
  capture is committed — the gap FleetSync later closed.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| Sync hunt | 40-bit register, ≤ 5 errors or ≥ 35 (inverted) | `internal/radio/mdc1200/receiver/receiver.go` (`Push`, `gdThresh`) |
| De-interleave | 16×7 columns → 14 LSB-first bytes | `internal/radio/mdc1200/mdc1200.go` (`deinterleave`) |
| CRC | reflected CRC-16/CCITT over `data[0:4]` | `mdc1200.go` (`crc16`, `validCRC`) |
| Opcode labels | `(op, arg)` → "PTT ID", "Emergency", … | `mdc1200.go` (`opLabel`, `summary`) |
| Double packets | ops `0x35`/`0x55` capture a second block | `receiver.go` (`stateBlock2`, `decodeExtra`) |
| Bus + storage | `KindMDC1200Message` → `mdc1200_log` | `internal/events/bus.go`, `internal/storage/mdc1200log.go` |
| REST + panel | `GET /api/v1/mdc1200/messages`, 5 s poll | `internal/api/handlers_mdc1200.go`, `web/src/panels/MDC1200.tsx` |

## In this post

- **What a burst carries** — PTT ID at head and tail, and the opcode table.
- **Hunting forty bits** — the sync register, five errors, inverted polarity.
- **Grid, bytes, CRC** — the interleave, the header layout, the reflected check.
- **Double packets and the emit** — from `Message` to bus payload.
- **Bus to panel, and the template's gap** — the wiring, and what tests do not prove.

## What a burst carries

MDC1200 — Motorola Data Communications — is the data burst a Motorola radio
keys at the start and often the end of a transmission on an analog channel.
The package doc lists what it carries: the unit ID (ANI) plus "emergency,
status, call-alert, radio-check and selective-call signaling." On an FM
channel it is the only thing that says *which* radio is talking, and the only
way an emergency button reaches a console. The
[MDC1200 reference]({{ '/reference/mdc1200/' | relative_url }}) covers the
history; the
[frame page]({{ '/reference/mdc1200-frame/' | relative_url }}) the layout
this part walks.

Each decoded burst becomes a `Message` with an `Op`, an `Arg`, a 16-bit
`UnitID` and a label from `opLabel`. The table is deliberately best-effort —
"many vendor-specific and extended opcodes exist" — so unrecognised pairs
return `""` and the panel shows the raw op/arg rather than dropping the burst:

| Op | Arg | Label |
|---|---|---|
| `0x01` | `0x00` / other | PTT ID (end) / PTT ID |
| `0x00` | any | Emergency |
| `0x06` | — | Status request |
| `0x12` | N | Status N |
| `0x0A` | — | Call alert / page |
| `0x2B` / `0x2C` | — | Radio inhibit (stun) / Radio enable (revive) |
| `0x35` | — | Voice selective call (double packet) |
| `0x63` | — | Radio check |

The one-line `Body` — `Unit 1234: PTT ID`, ` (CRC?)` appended on a failed
check — is what the log and panel render.

## Hunting forty bits

Bits arrive one at a time from the slicer
([Part 2]({{ '/blog/deep-dives/beyond-voice-02-afsk-ffsk-fundamentals/' | relative_url }}))
into `receiver.Receiver.Push`, a three-state machine with one anchor: the
40-bit sync word. No preamble test, no length field — the register slides,
and a burst begins when the sync is close enough:

```go
// internal/radio/mdc1200/receiver/receiver.go (shape)
const gdThresh = 5 // bit errors tolerated across the 40-bit sync word

func (r *Receiver) Push(bit byte) {
    bit &= 1
    switch r.st {
    case stateHunt:
        r.reg = (r.reg << 1) | uint64(bit)
        d := bits.OnesCount64((r.reg ^ mdc1200.SyncWord) & syncMask)
        switch {
        case d <= gdThresh:                     r.beginBlock(false)
        case d >= mdc1200.SyncBits-gdThresh:    r.beginBlock(true) // complement
        }
    case stateBlock1, stateBlock2:
        if r.inverted { bit ^= 1 }
        r.buf[r.n] = bit
        r.n++
        if r.n == mdc1200.FrameBits { r.finishBlock() }
    }
}
```

Two decisions in those lines carry the protocol. **Five errors of slack**
match the threshold the reference MDC1200 decoders use — tolerable
false-lock rate on random slicer output, survival of the bit errors a fading
burst delivers. **The complement path** is why the front end never needs to
know the tone sense: an inverted discriminator presents every bit inverted,
sync included; a distance of 35 or more means the complement matched,
`inverted` is set, and every payload bit is XORed back before it is stored.
`TestReceiverDecodesInvertedBurst` pushes a whole stream through `b ^ 1` and
gets `Unit 0x0042: Emergency` out unchanged. After a burst the receiver
returns to the hunt with a cleared register so the frame just decoded cannot
re-trigger — the
[frame-synchronization]({{ '/reference/frame-synchronization/' | relative_url }})
discipline every burst framer in the tree shares.

## Grid, bytes, CRC

`finishBlock` hands 112 captured bits to `mdc1200.DecodeFrame`, which first
undoes the transmitter's interleave: the bits were written column-wise into
a 16-wide, 7-deep grid, and reading them back restores the logical order,
packed LSB-first into 14 bytes:

```go
// internal/radio/mdc1200/mdc1200.go (shape)
func deinterleave(bits []byte) ([]byte, bool) {
    var lbits [FrameBits]byte
    idx := 0
    for i := 0; i < 16; i++ {
        for j := 0; j < 7; j++ {
            lbits[idx] = bits[j*16+i] & 1 // column i, row j
            idx++
        }
    }
    data := make([]byte, 14)
    for i := range data {                 // pack LSB-first
        for j := 0; j < 8; j++ {
            if lbits[i*8+j] != 0 { data[i] |= 1 << uint(j) }
        }
    }
    return data, true
}
```

[Interleaving]({{ '/reference/interleaving/' | relative_url }}) spreads a
short fade across the codeword so the air FEC can recover it. The 14 bytes,
from the package doc:

| Bytes | Field |
|---|---|
| `data[0]` | `op` |
| `data[1]` | `arg` |
| `data[2:4]` | unit ID, **big-endian** |
| `data[4:6]` | CRC-16 of `data[0:4]`, **little-endian** on the wire |
| `data[6:14]` | redundancy — "used by the over-the-air FEC; not yet exploited here" |

The CRC is where a constant drifts silently, so be exact: CRC-16/CCITT
with reflected input and output, polynomial `0x1021` in its reflected form
`0x8408`, initial value `0x0000`, final XOR `0xFFFF`. The same polynomial
appears across GopherTrunk's framers under different settings — the
[CRC-16/CCITT page]({{ '/reference/crc-16-ccitt/' | relative_url }})
catalogues them, and the CSBK mask lesson of the
[DMR series]({{ '/blog/deep-dives/dmr-end-to-end-04-fec-stack-forged-terminator/' | relative_url }})
is that "CCITT" names a family, not a value. `validCRC` recomputes over the
four header bytes and compares against `data[5]<<8 | data[4]`; a mismatch
does **not** discard the burst. `DecodeFrame` returns the `Message` with `CRCOK=false`, and the receiver
publishes it unless `DropBadCRC` is set.

<figure class="lab-figure">
<svg viewBox="0 0 680 250" width="680" height="250" role="img" aria-label="An MDC1200 burst as a timeline: a 40-bit sync word, then 112 payload bits shown as a 16-column by 7-row grid read out column by column into fourteen bytes labelled op, arg, unit ID, CRC-16 and eight redundancy bytes, with a bracket marking the CRC coverage over the first four bytes.">
  <rect x="10" y="20" width="150" height="30" rx="4" fill="none" stroke="var(--accent)"/>
  <text x="85" y="40" text-anchor="middle" fill="var(--accent)" font-size="10">sync 0x07092A446F · 40 bits</text>
  <rect x="160" y="20" width="510" height="30" rx="4" fill="none" stroke="currentColor"/>
  <text x="415" y="40" text-anchor="middle" fill="currentColor" font-size="10">112 payload bits — 16 × 7 column interleave</text>
  <text x="85" y="66" text-anchor="middle" fill="var(--fg-muted)" font-size="8">≤ 5 errors, or ≥ 35 (complement)</text>
  <rect x="200" y="80" width="160" height="70" fill="none" stroke="var(--fg-muted)"/>
  <line x1="210" y1="80" x2="210" y2="150" stroke="var(--accent)"/>
  <text x="280" y="164" text-anchor="middle" fill="var(--fg-muted)" font-size="8">16 columns × 7 rows · read column i, rows j: bits[j*16+i]</text>
  <line x1="360" y1="115" x2="392" y2="115" stroke="currentColor"/>
  <text x="376" y="108" text-anchor="middle" fill="var(--fg-muted)" font-size="8">pack LSB-first</text>
  <rect x="400" y="98" width="30" height="34" fill="none" stroke="var(--accent)"/><text x="415" y="119" text-anchor="middle" fill="var(--accent)" font-size="9">op</text>
  <rect x="430" y="98" width="30" height="34" fill="none" stroke="var(--accent)"/><text x="445" y="119" text-anchor="middle" fill="var(--accent)" font-size="9">arg</text>
  <rect x="460" y="98" width="50" height="34" fill="none" stroke="var(--accent)"/><text x="485" y="119" text-anchor="middle" fill="var(--accent)" font-size="9">unit ID</text>
  <rect x="510" y="98" width="50" height="34" fill="none" stroke="currentColor"/><text x="535" y="119" text-anchor="middle" fill="currentColor" font-size="9">CRC-16</text>
  <rect x="560" y="98" width="110" height="34" fill="none" stroke="var(--fg-muted)"/><text x="615" y="119" text-anchor="middle" fill="var(--fg-muted)" font-size="9">8 redundancy bytes</text>
  <path d="M 400 140 L 400 148 L 510 148 L 510 140" fill="none" stroke="currentColor"/>
  <text x="455" y="160" text-anchor="middle" fill="currentColor" font-size="8">CRC covers these four</text>
  <text x="340" y="200" text-anchor="middle" fill="var(--fg-muted)" font-size="9">unit ID big-endian · CRC little-endian · reflected 0x8408, init 0x0000, xorout 0xFFFF</text>
  <text x="340" y="222" text-anchor="middle" fill="var(--fg-muted)" font-size="8">redundancy captured, not decoded · op 0x35 / 0x55 → a second block follows</text>
</svg>
<figcaption>One burst: forty bits of sync the receiver matches with five errors of slack, then 112 bits read back column-by-column into fourteen bytes, of which the CRC guards only the first four.</figcaption>
</figure>

## Double packets and the emit

Two opcodes select an *extended* two-block message: `0x35` (voice selective
call) and `0x55`. When `finishBlock` decodes a first block whose
`DoublePacket` flag is set, it stashes the `Message` and moves to
`stateBlock2` to capture another 112 bits *without* hunting a second sync
word. The second block's raw header bytes are attached as `Extra`; its
vendor-specific payload is "left to a follow-up." `TestReceiverDoublePacket` asserts `BurstsEmitted == 1`.

The emit is the template's bus edge, and every clone has copied its shape:

```go
// internal/radio/mdc1200/receiver/receiver.go (shape)
func (r *Receiver) emit(msg mdc1200.Message) {
    if !msg.CRCOK {
        r.burstsCRC.Add(1)
        if r.dropBadCRC { return }
    }
    r.bus.Publish(events.Event{
        Kind: events.KindMDC1200Message, Timestamp: time.Now(),
        Payload: storage.MDC1200Message{Op: msg.Op, Arg: msg.Arg, UnitID: msg.UnitID,
            Operation: msg.Operation, Body: msg.Body, RawHex: msg.RawHex, CRCOK: msg.CRCOK},
    })
    r.burstsEmit.Add(1)
}
```

The receiver panics without a bus (`TestNewRequiresBus`) — unlike the
FleetSync framer, this layer *is* the publisher, which is why
[Part 1]({{ '/blog/deep-dives/beyond-voice-01-eleven-place-pattern/' | relative_url }})
called it an orchestrator. Its `Stats()` — `BurstsIn`, `BurstsBadCRC`,
`BurstsEmitted` — tell an operator whether a silent panel means no sync
words, bad CRCs, or a bus nobody drains.

## Bus to panel, and the template's gap

From the bus onward the chain is the one
[Part 1]({{ '/blog/deep-dives/beyond-voice-01-eleven-place-pattern/' | relative_url }})
drew. `storage.MDC1200Log` embeds `eventLog[MDC1200Message]` and inserts into
`mdc1200_log` (`op`, `arg`, `unit_id`, `operation`, `body`, `raw_hex`,
`crc_ok`, indexed by time and unit), swept by `retention.log_days`.
`GET /api/v1/mdc1200/messages?limit=N` (default 200, max 5000) reads it
through the daemon's `mdc1200Provider` adapter and answers `503` without
`storage.path`. The `/mdc1200` panel polls every 5 s, renders the unit ID as
hex, puts an err-toned `!` badge beside `Emergency` and an `ok`/`fail` badge
in the CRC column. Configuration is one list:

```yaml
mdc1200:
  channels:
    - serial: "vhf-antenna"
      frequency_hz: 154_000_000   # the analog voice channel to monitor
      drop_bad_crc: false         # true to drop CRC-failed bursts
```

The daemon constructs one `mdc1200afsk.Receiver` per entry and spawns it
non-essentially; `doctor` names `mdc1200` when storage is missing. Now the
honest part. MDC1200's tests are `TestDecodeFrameRoundTrip`,
`TestDecodeFrameCRCMismatch`, `TestDoublePacketFlag`, `TestCRC16KnownVector`
and the receiver's framer tests — and every frame they feed comes from a
helper, `encodeFrame`, "the transmitter-side inverse of `deinterleave`." That
is the
[self-consistent trap]({{ '/blog/solution-postmortem/from-the-issue-tracker-20-self-consistent-trap/' | relative_url }})
in its purest form: if the grid were transposed in both, every test would
pass and no real burst would decode. `TestCRC16KnownVector`, despite its
name, pins determinism and mutation sensitivity, not a value from an
independent implementation. And the tree carries **no committed MDC1200 air
capture** — the `mdc1200` package has no `testdata`, where
`fleetsync/afsk/testdata` holds two real-air slices. The sync word,
interleave geometry and CRC parameters are public protocol facts, but by the
blog's own rule that makes MDC1200 **synthetic-verified**: the decoder has
not been shown to decode a Motorola radio's burst in this repository. When FleetSync cloned the
template
([Part 4]({{ '/blog/deep-dives/beyond-voice-04-fleetsync-wav-that-lied/' | relative_url }})),
the clone got the reference-literal tests and on-air fixtures the original
still lacks — the point of
[From Spec to Shipping Part 3]({{ '/blog/deep-dives/from-spec-to-shipping-03-literal-vectors/' | relative_url }}).
One MDC1200 capture in a `testdata/` slice would close the gap.

### How the template shaped the Go code

- **Protocol facts are constants with names.** `SyncWord`, `SyncBits`,
  `FrameBits` and `gdThresh` sit at the top of their files; a clone changes
  four numbers and a table.
- **Decode never discards.** `DecodeFrame` returns `(Message, bool)` with the
  `Message` complete even when the bool is false; the drop decision belongs
  to the receiver's `DropBadCRC`.
- **The state machine owns multi-block capture.** Double packets are two
  states in `Push`, not a second framer.
- **Counters are the diagnostic surface.** Three atomics in `Stats()`,
  repeated as `BurstsIn`/`BurstsBadCRC`/`BurstsEmitted` in
  `fleetsync.Framer.Stats`.

## Where this goes next

FleetSync took this template — register hunt, complement lock, best-effort
decode, bus emit — and changed what it had to: a 16-bit sync behind a 24-bit
preamble, two 32-bit words, a bit-serial block check and an error-correcting
FleetSync II variant.
[Part 4]({{ '/blog/deep-dives/beyond-voice-04-fleetsync-wav-that-lied/' | relative_url }})
follows the clone through the reporter's SDR# captures — and the WAV reader
that, not the decoder, turned out to be the blocker.

## FAQ

**What is in an MDC1200 burst?**
A 40-bit sync word (`0x07092A446F`) and 112 payload bits interleaved over a
16×7 grid. De-interleaved they hold an opcode, an argument, a 16-bit unit ID,
a CRC-16 over those four bytes and eight redundancy bytes. The opcode names
the event: PTT ID, emergency, status, radio check, call alert, selective
call, inhibit or enable.

**How does GopherTrunk handle an inverted MDC1200 signal?**
The framer measures the Hamming distance between its 40-bit register and the
sync word every bit. Five or fewer differences is a normal lock; 35 or more
means the complement matched, so the receiver marks the burst inverted and
XORs every captured payload bit back before decoding.

**What CRC does MDC1200 use?**
CRC-16/CCITT with reflected input and output — polynomial `0x1021` in its
reflected form `0x8408`, initial value `0x0000`, final XOR `0xFFFF` — over
the op, arg and two unit-ID bytes, carried little-endian on the wire. A
mismatch publishes with `crc_ok=false` by default so the panel can flag it.

**Why does the panel show bursts that failed the CRC?**
Because a marginal burst with a plausible unit ID is still information on a
weak channel. `drop_bad_crc: false` is the default per channel; setting it
true discards CRC failures at the receiver so a noisy channel does not fill
`mdc1200_log` with `(CRC?)` rows.

**Is the MDC1200 decoder verified against real radios?**
Not in the repository. Its tests round-trip frames built by a helper that
inverts the decoder's own interleave, the CRC test pins determinism rather
than an independent vector, and no on-air capture is committed. By
GopherTrunk's own discipline the decoder is synthetic-verified until a
captured burst decodes.

## Series navigation

**Part 3 of 14** · ←
[Part 2: AFSK & FFSK — Two Tones, One Bit]({{ '/blog/deep-dives/beyond-voice-02-afsk-ffsk-fundamentals/' | relative_url }})
· Next →
[Part 4: FleetSync — Cloning the Template & the WAV That Lied]({{ '/blog/deep-dives/beyond-voice-04-fleetsync-wav-that-lied/' | relative_url }})
