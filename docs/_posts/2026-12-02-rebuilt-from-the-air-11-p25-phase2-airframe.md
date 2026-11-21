---
title: "Rebuilt From the Air, Part 11: P25 Phase 2 I — Twelve Timeslots and the Two That Swap"
description: "A hybrid P25 site never followed a call because a TDMA grant's channel number counts logical channels, not carriers. Fixing that exposed the real problem: GopherTrunk's Phase 2 framing was a working model that matched no reference. The rebuild — S-ISCH and I-ISCH, DUIDs, RS(63,35) MAC PDUs — is pinned to OP25 and SDRTrunk."
category: solution-postmortem
keywords: p25 phase 2 tdma decoder, p25 tdma channel number slots per carrier, iden_up_tdma channel type, p25 phase 2 isch, p25 phase 2 duid, p25 mac pdu reed solomon 63 35, p25 logical channel timeslot, hybrid phase 1 phase 2 site, op25 sdrtrunk reference, gophertrunk p25 phase 2
tags: [rebuilt-from-the-air, p25, phase2, tdma, self-consistent-trap, reference-implementations]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "Rebuilt From the Air"
series_part: 11
---

*Part 11 of **Rebuilt From the Air**, a 14-part postmortem series on the
GopherTrunk decoders that passed every test and never matched a real radio —
each one diagnosed from a reference implementation or a reporter's capture,
rebuilt against independent decoders, and pinned to real air where a capture
exists.
[Part 10]({{ '/blog/solution-postmortem/rebuilt-from-the-air-10-dstar/' | relative_url }})
rebuilt D-STAR from a firmware's own output. This part and the next are the
largest rebuild in the series: P25 Phase 2 TDMA, where a bug in one line of
frequency arithmetic turned out to be standing on a framing layer that
matched nothing on the air.*

> **TL;DR:** A hybrid site — Phase 1 control channel, Phase 2 voice — never
> followed a call
> ([#1273](https://github.com/MattCheramie/GopherTrunk/issues/1273)). Layer
> one: a TDMA grant's channel number counts *logical* channels, so the
> downlink is base + spacing × (n / slots) and the slot is n mod slots;
> GopherTrunk used base + n × spacing and put TDMA grants on the wrong
> carrier (`SlotsPerCarrier`, `p25.ParseTDMAChannelType`). Layer two:
> recording only the granted slot needed real framing, and the model in
> `superframe.go` / `isch.go` / `voice.go` matched no reference. The air
> format (`phase2/airframe.go`) is twelve 180-dibit timeslots, each a
> 20-dibit ISCH — sync word `0x575D57F7FF` or a (40,9,16) codeword — and a
> 160-dibit body with a DUID; logical channels are not slot parity, because
> timeslots 10 and 11 swap. MAC PDUs (`macframe.go`) are RS(63,35) hexbits
> with a CRC-12 or CRC-16. Reference-pinned against OP25 and SDRTrunk.

**Key takeaways**

- **A channel number is not a carrier number on TDMA.** On a 2-slot carrier,
  channel n lives on carrier n / 2, slot n mod 2; ignoring the slot count
  doubles every grant's distance from the band's base.
- **A working model confined to one file is still a guess.** The old ISCH
  was contained, documented and symmetric with its encoder — and failed on
  46–62 % of the sub-frames of a real capture.
- **Two references that agree on every constant are a specification.** The
  I-ISCH table from one and the generator matrix from the other produce the
  same 128 codewords, and the tests pin one against the other.
- **Slot parity is almost right, which is worse than wrong.** Ten of twelve
  timeslots alternate; the last pair is swapped.

## Cheat sheet

| | |
|---|---|
| Issue | [#1273](https://github.com/MattCheramie/GopherTrunk/issues/1273) — hybrid Phase 1 control / Phase 2 voice site never follows a call |
| Symptom | the control channel locks and decodes grants, and no call is ever followed; a Phase 2 control channel publishes its grants at 0 Hz |
| Wrong theories | the Motorola "site broadcast" opcodes are undecoded grants; the site has no traffic; the framing is a working model one constant away from right |
| Real cause | channel numbers resolved as carriers; and a fabricated framing: one sync per superframe, a Golay ISCH, contiguous unscrambled voice, a trellis + RS(24,16) MAC chain |
| Fix | `BandPlan.Frequency` / `Timeslot` with `SlotsPerCarrier`; TDMA identifier forms `0x73` / `0xF3`; `AirFramer`, `DecodeAirMAC`, `ExtractAirVoiceFrames` |
| Diagnostic | payloads built from OP25's bit positions; literal I-ISCH, DUID and mask values from the references; five packets decoded by OP25's own code |
| Status | reference-pinned and synthetic-verified; the reporter's site is still the on-air gate; Part 12 has the first real carrier |

## In this post

- **A hybrid site that never followed a call** — the symptom and two wrong readings.
- **Layer one: logical channels** — the divide, the slot, a self-made pin.
- **Layer two: a model that matched no reference** — why the slot exposed the framing.
- **The real timeslot** — ISCH, DUID, voice positions, and the swap.
- **The MAC path** — RS(63,35), two CRCs and a length table.
- **What is wired, what is verified** — consumers, dead config keys, the open gate.

## A hybrid site that never followed a call

A common P25 Phase 2 deployment is a hybrid: the control channel stays
Phase 1 C4FM and its grants steer radios to 6000-symbol-per-second
[TDMA]({{ '/reference/tdma/' | relative_url }}) carriers. The reporter on
#1273 had one. The control channel locked and decoded grants, and no call
was ever followed.

Two readings of the log were wrong before the right one. The first was the
reporter's, and reasonable: the log carried lines about Motorola opcodes
"named, not field-decoded", which looked like grants the decoder could not
read. They are site broadcasts with constant payloads — opcodes `0x05` and
`0x09` — now logged once per distinct payload as
`p25: motorola opcode (site broadcast, not a call grant …)`.

The second was a silence. A grant naming a channel ID the site had never
announced was dropped at DEBUG, so a site that followed nothing looked like
a site with no traffic. It now warns, rate-limited, that "voice grants name
a channel ID the site has not announced".

Neither was the bug. The grants were read correctly and resolved to the
wrong place.

## Layer one: logical channels

A grant carries a channel ID and a channel number; the site's
[identifier update]({{ '/reference/p25-identifier-update/' | relative_url }})
for that ID supplies a base frequency and a spacing.
[P25 End to End Part 5]({{ '/blog/deep-dives/p25-end-to-end-05-channels-band-plans/' | relative_url }})
gave the arithmetic as one line: base + number × spacing. That is right for
FDMA and wrong for TDMA.

The TDMA identifier (`IDEN_UP_TDMA`, opcode `0x33`) carries a four-bit
channel type that says how many logical channels one carrier holds. Two
independent decoders agree on the table — SDRTrunk's `ChannelType` values
0 to 5 and OP25's `slots_per_carrier` list `[1,1,1,2,4,2]`: types 0–2 are
FDMA, 3 and 5 are 2-slot TDMA, 4 is 4-slot. A grant's channel number
counts *logical* channels, carrier × slots + slot:

```go
// internal/radio/p25/phase1/identifier.go (shape)
carrier := uint64(channelNumber)
if u.SlotsPerCarrier > 1 {
    carrier /= uint64(u.SlotsPerCarrier) // logical channel → carrier
}
hz := u.BaseHz + carrier*uint64(u.SpacingHz)
```

`BandPlan.Timeslot` returns the remainder plus one, the grant's 1-based
`Timeslot`. GopherTrunk had neither: its channel-type table was invented,
agreed only with its own assembler, and never supplied a slot count.

The repository's own fixture shows the size of the error. A band with base
467.5125 MHz and 6.25 kHz spacing, channel 176: the old resolver said
468.6125 MHz; the carrier is 467.5125 + 88 × 0.00625 = 468.0625 MHz, slot 1.
The grant landed 550 kHz away. Worse, a test had "pinned" 468.6125 — a
number GopherTrunk had computed itself. A literal in a test is evidence only
if it came from somewhere else
([From Spec to Shipping Part 3]({{ '/blog/deep-dives/from-spec-to-shipping-03-literal-vectors/' | relative_url }})).

The replacement, `TestBandPlanTDMAGrantResolvesToCarrierNotLogicalChannel`,
builds its `0x33` payloads from OP25's bit positions with a helper that
shares nothing with GopherTrunk's assembler.

The same hole was deeper on a Phase 2 *control* channel. It parsed only the
FDMA identifier form (MAC opcode `0x7D`), never the TDMA forms `0x73` and
`0xF3` a Phase 2 site uses for its voice channels, so those grants resolved
to 0 Hz. Both are parsed now (`phase2.ParseIdentifierUpdateTDMA`) and listed
as known opcodes, or strict validation would drop them. One table serves
both decoders — `p25.ParseTDMAChannelType`, in a package of its own because
`phase2` cannot import `phase1`. An operator can also seed a TDMA band the
site never announces, under the system's entry:

```yaml
      p25_band_plan:
        - channel_id: 3          # a 2-slot Phase 2 voice band
          base_hz: 762_006_250
          spacing_hz: 12_500
          tx_offset_hz: 30_000_000
          bandwidth_hz: 12_500
          slots_per_carrier: 2
```

## Layer two: a model that matched no reference

With grants on the right carrier and carrying a timeslot, the voice chain
had a new obligation. A Phase 2 carrier holds two calls, one per logical
channel, and the chain had been recording every voice burst on the carrier
into one file. Recording only the granted channel means knowing which
timeslot each burst is.

That is what framing answers, and the framing in `superframe.go`,
`isch.go` and `voice.go` could not. It was, by its own comments, a working
model: one sync word per superframe, in sub-frame 0; a 12-bit ISCH under
Golay(24,12) carrying an invented `SlotType` and counter; voice frames laid
contiguously after it; no scrambling on voice. The earlier posts said so
plainly and argued that containment made it safe — a wrong constant would
be a one-file fix.

It was not one constant. Set beside OP25's TDMA decoder and SDRTrunk's
Phase 2 module — two implementations proven on live air, which agree on
every constant below — the model shared almost nothing with either. And the
repository already held evidence nobody had read that way: on the
ground-truth capture from
[#915](https://github.com/MattCheramie/GopherTrunk/issues/915) the model's
ISCH failed its Golay decode on 46–62 % of sub-frames. That had looked like
a weak-signal problem. It was a decoder reading a field that is not there.

The model's encoder built every fixture, so every test passed — the
[self-consistent trap]({{ '/blog/solution-postmortem/from-the-issue-tracker-20-self-consistent-trap/' | relative_url }})
at the scale of the
[SmartNet rebuild]({{ '/blog/deep-dives/from-spec-to-shipping-08-smartnet-rebuild/' | relative_url }}):
a whole frame, not a table.

## The real timeslot

<figure class="lab-figure">
<svg viewBox="0 0 680 262" width="680" height="262" role="img" aria-label="Top: one 360 millisecond superframe drawn as twelve timeslots numbered 0 to 11. An I-ISCH opens timeslots 0, 1, 4, 5, 8 and 9 and an S-ISCH sync word opens timeslots 2, 3, 6, 7, 10 and 11. The logical channel row reads 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, then 1, 0, with the last pair marked as swapped. Bottom: one 180-dibit timeslot, a 20-dibit ISCH followed by a 160-dibit body holding four 36-dibit voice frames at body dibits 1, 38, 86 and 123, with the four DUID dibits at body dibits 0, 37, 122 and 159.">
  <text x="340" y="14" text-anchor="middle" fill="currentColor" font-size="10" font-weight="bold">one 360 ms superframe: 12 timeslots × 180 dibits</text>
  <text x="36" y="34" text-anchor="end" fill="var(--fg-muted)" font-size="8">ISCH</text>
  <text x="36" y="61" text-anchor="end" fill="var(--fg-muted)" font-size="8">slot</text>
  <text x="36" y="88" text-anchor="end" fill="var(--fg-muted)" font-size="8">channel</text>
  <text x="65" y="34" text-anchor="middle" fill="currentColor" font-size="9">I</text>
  <text x="115" y="34" text-anchor="middle" fill="currentColor" font-size="9">I</text>
  <text x="165" y="34" text-anchor="middle" fill="var(--accent)" font-size="9">S</text>
  <text x="215" y="34" text-anchor="middle" fill="var(--accent)" font-size="9">S</text>
  <text x="265" y="34" text-anchor="middle" fill="currentColor" font-size="9">I</text>
  <text x="315" y="34" text-anchor="middle" fill="currentColor" font-size="9">I</text>
  <text x="365" y="34" text-anchor="middle" fill="var(--accent)" font-size="9">S</text>
  <text x="415" y="34" text-anchor="middle" fill="var(--accent)" font-size="9">S</text>
  <text x="465" y="34" text-anchor="middle" fill="currentColor" font-size="9">I</text>
  <text x="515" y="34" text-anchor="middle" fill="currentColor" font-size="9">I</text>
  <text x="565" y="34" text-anchor="middle" fill="var(--accent)" font-size="9">S</text>
  <text x="615" y="34" text-anchor="middle" fill="var(--accent)" font-size="9">S</text>
  <rect x="40" y="42" width="50" height="28" fill="none" stroke="currentColor"/>
  <rect x="90" y="42" width="50" height="28" fill="none" stroke="currentColor"/>
  <rect x="140" y="42" width="50" height="28" fill="none" stroke="currentColor"/>
  <rect x="190" y="42" width="50" height="28" fill="none" stroke="currentColor"/>
  <rect x="240" y="42" width="50" height="28" fill="none" stroke="currentColor"/>
  <rect x="290" y="42" width="50" height="28" fill="none" stroke="currentColor"/>
  <rect x="340" y="42" width="50" height="28" fill="none" stroke="currentColor"/>
  <rect x="390" y="42" width="50" height="28" fill="none" stroke="currentColor"/>
  <rect x="440" y="42" width="50" height="28" fill="none" stroke="currentColor"/>
  <rect x="490" y="42" width="50" height="28" fill="none" stroke="currentColor"/>
  <rect x="540" y="42" width="50" height="28" fill="none" stroke="var(--accent)" stroke-width="1.5"/>
  <rect x="590" y="42" width="50" height="28" fill="none" stroke="var(--accent)" stroke-width="1.5"/>
  <text x="65" y="60" text-anchor="middle" fill="currentColor" font-size="9">0</text>
  <text x="115" y="60" text-anchor="middle" fill="currentColor" font-size="9">1</text>
  <text x="165" y="60" text-anchor="middle" fill="currentColor" font-size="9">2</text>
  <text x="215" y="60" text-anchor="middle" fill="currentColor" font-size="9">3</text>
  <text x="265" y="60" text-anchor="middle" fill="currentColor" font-size="9">4</text>
  <text x="315" y="60" text-anchor="middle" fill="currentColor" font-size="9">5</text>
  <text x="365" y="60" text-anchor="middle" fill="currentColor" font-size="9">6</text>
  <text x="415" y="60" text-anchor="middle" fill="currentColor" font-size="9">7</text>
  <text x="465" y="60" text-anchor="middle" fill="currentColor" font-size="9">8</text>
  <text x="515" y="60" text-anchor="middle" fill="currentColor" font-size="9">9</text>
  <text x="565" y="60" text-anchor="middle" fill="var(--accent)" font-size="9">10</text>
  <text x="615" y="60" text-anchor="middle" fill="var(--accent)" font-size="9">11</text>
  <text x="65" y="88" text-anchor="middle" fill="currentColor" font-size="9">0</text>
  <text x="115" y="88" text-anchor="middle" fill="currentColor" font-size="9">1</text>
  <text x="165" y="88" text-anchor="middle" fill="currentColor" font-size="9">0</text>
  <text x="215" y="88" text-anchor="middle" fill="currentColor" font-size="9">1</text>
  <text x="265" y="88" text-anchor="middle" fill="currentColor" font-size="9">0</text>
  <text x="315" y="88" text-anchor="middle" fill="currentColor" font-size="9">1</text>
  <text x="365" y="88" text-anchor="middle" fill="currentColor" font-size="9">0</text>
  <text x="415" y="88" text-anchor="middle" fill="currentColor" font-size="9">1</text>
  <text x="465" y="88" text-anchor="middle" fill="currentColor" font-size="9">0</text>
  <text x="515" y="88" text-anchor="middle" fill="currentColor" font-size="9">1</text>
  <text x="565" y="88" text-anchor="middle" fill="var(--accent)" font-size="10" font-weight="bold">1</text>
  <text x="615" y="88" text-anchor="middle" fill="var(--accent)" font-size="10" font-weight="bold">0</text>
  <text x="590" y="104" text-anchor="middle" fill="var(--accent)" font-size="8">the last pair swaps</text>
  <text x="240" y="104" text-anchor="middle" fill="var(--fg-muted)" font-size="8">I = (40,9,16) codeword naming the timeslot · S = sync word 0x575D57F7FF</text>
  <text x="340" y="132" text-anchor="middle" fill="currentColor" font-size="10" font-weight="bold">one timeslot: 20-dibit ISCH + 160-dibit body</text>
  <rect x="40" y="142" width="600" height="38" fill="none" stroke="var(--fg-muted)"/>
  <rect x="40" y="142" width="67" height="38" fill="none" stroke="var(--accent)" stroke-width="1.5"/>
  <text x="73" y="165" text-anchor="middle" fill="var(--accent)" font-size="9">ISCH</text>
  <rect x="110" y="148" width="120" height="26" fill="none" stroke="currentColor"/>
  <text x="170" y="165" text-anchor="middle" fill="currentColor" font-size="8">voice 1 · body 1</text>
  <rect x="233" y="148" width="120" height="26" fill="none" stroke="currentColor"/>
  <text x="293" y="165" text-anchor="middle" fill="currentColor" font-size="8">voice 2 · body 38</text>
  <rect x="393" y="148" width="120" height="26" fill="none" stroke="currentColor"/>
  <text x="453" y="165" text-anchor="middle" fill="currentColor" font-size="8">voice 3 · body 86</text>
  <rect x="517" y="148" width="120" height="26" fill="none" stroke="currentColor"/>
  <text x="577" y="165" text-anchor="middle" fill="currentColor" font-size="8">voice 4 · body 123</text>
  <line x1="108" y1="142" x2="108" y2="192" stroke="var(--accent)" stroke-width="1.5"/>
  <line x1="231" y1="142" x2="231" y2="192" stroke="var(--accent)" stroke-width="1.5"/>
  <line x1="515" y1="142" x2="515" y2="192" stroke="var(--accent)" stroke-width="1.5"/>
  <line x1="639" y1="142" x2="639" y2="192" stroke="var(--accent)" stroke-width="1.5"/>
  <text x="340" y="206" text-anchor="middle" fill="var(--accent)" font-size="8">the 8-bit DUID is spread over body dibits 0, 37, 122 and 159</text>
  <text x="340" y="226" text-anchor="middle" fill="currentColor" font-size="9">a voice burst's body is XORed with PN44 superframe bits [20 + 360t, 340 + 360t) for timeslot t</text>
  <text x="340" y="244" text-anchor="middle" fill="var(--fg-muted)" font-size="8">VOICE-4 carries four 72-bit AMBE+2 frames; VOICE-2 carries the first two</text>
</svg>
<figcaption>Every timeslot opens with its own ISCH, and the logical channel follows slot parity for ten timeslots and then does not.</figcaption>
</figure>

The outbound channel is a stream of 30 ms timeslots, twelve to a 360 ms
superframe, and each timeslot is one 180-dibit packet: a 20-dibit ISCH and a
160-dibit body.

**Two kinds of ISCH.** Before timeslots 2, 3, 6, 7, 10 and 11 the ISCH is
the **S-ISCH** — the 40-bit
[sync word]({{ '/reference/p25-phase-2-sync-word/' | relative_url }})
`0x575D57F7FF`, six times a superframe, not once. Before 0, 1, 4, 5, 8 and 9
it is an **I-ISCH**: a (40,9,16) codeword that names the timeslot it opens
as location × 4 + channel, and corrects seven bit errors. `iischCodewords`
is generated from SDRTrunk's generator matrix XORed with `0x184229d461`, and
`TestIISCHCodewordsMatchOP25Table` holds eight of the results to literal
entries of OP25's 128-word table.

**The DUID.** What a burst carries is an 8-bit data-unit ID spread over body
dibits 0, 37, 122 and 159 — a 16-word code of distance 4 — not a field in
the ISCH. `TestDUIDMatchesOP25Lookup` checks it against OP25's 256-entry
lookup.

**Voice.** A VOICE-4 burst holds four 72-bit AMBE+2 frames at body dibits 1,
38, 86 and 123; VOICE-2 holds the first two. The codeword layout is DMR's
exactly, so `ExtractAirVoiceFrames` hands each to `dmrvoice.DecodeAMBEFrame`
after descrambling with the timeslot's slice of the PN44 sequence.

```go
// internal/radio/p25/phase2/airframe.go (shape)
var airVoiceFrameBody = [4]int{1, 38, 86, 123} // VOICE-2 uses the first two
var airDUIDBody = [4]int{0, 37, 122, 159}      // most significant dibit first
var airSlotLCH = [AirSlotsPerSuperframe]int{0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 1, 0}
```

**The swap.** The two logical channels alternate by timeslot — except the
final pair, where timeslot 10 is channel 1 and timeslot 11 is channel 0. OP25 carries it as a slot table and
SDRTrunk as a swap in its final fragment; `TestAirSlotLogicalChannel` pins
it. Slot parity, the old decoder's `i & 1`, is right ten times in twelve;
used as the filter it would hand one burst in six to the wrong call.

`AirFramer` puts this together: it hunts the S-ISCH under all four constant
dibit rotations, cuts 180-dibit packets on that grid, names each timeslot
from its I-ISCH or by counting from the last one, and drops bursts until the
first I-ISCH says where in the superframe it is.

## The MAC path

Signalling was further from the air than voice. The model ran MAC PDUs
through a trellis code and an RS(24,16,9) outer code; on air there is
neither. A signalling burst's body carries six-bit symbols — hexbits — of a
shortened, punctured
[Reed–Solomon]({{ '/reference/reed-solomon-code/' | relative_url }})
RS(63,35) code over GF(64), whose information part is the PDU and a CRC:

| Channel | DUID (scrambled, clear) | Body dibits read | Received hexbits | Check |
|---|---|---|---|---|
| FACCH | 9, 15 | 1–36, 38–68, 90–121, 123–158 | 45 (indexes 9–53) | 144 PDU bits + CRC-12 |
| SACCH | 3, 12 | 1–36, 38–121, 123–158 | 52 (indexes 5–56) | 168 PDU bits + CRC-12 |
| LCCH | 4, 13 | as SACCH | as SACCH | 180 bits under CRC-16 |

`DecodeAirMAC` zero-fills the shortened leading symbols, names the unsent
trailing parity as erasures, and calls `framing.DecodeRS63_35`, an
errors-and-erasures decoder pinned to literal output of the RS library OP25
uses; it repairs 9 symbol errors on a FACCH and 11 on a SACCH, and the CRC
then arbitrates. A PDU's first byte is its
type and an offset; `MAC_PTT` and `MAC_END_PTT` have fixed layouts — the
key-up PDU carries the encryption sync and the talking radio — and every
other type is a run of MAC messages whose lengths come from OP25's
per-opcode length table (`AirMAC.Messages`).

The pin for this path runs in the unusual direction. Five packets — a FACCH
and a SACCH scrambled, a FACCH, a SACCH and an LCCH clear — were built with
`EncodeAirMACPacket` and fed to OP25's own decode path, compiled into a
harness outside the repository. Each came back `crc_ok=1` with the expected
bytes (`TestDecodeAirMACMatchesOP25`). An encoder can share a decoder's
mistakes; an independent *decoder* accepting its output is evidence neither
side wrote alone. OP25 is GPL and was read
for protocol facts; the layouts and lengths above are those facts.

## What is wired, what is verified

Three production paths run on the air format. The voice chain records only
its own logical channel — the grant's `Timeslot`, or the first channel that
carries voice when the grant names none — and takes signalling from that
channel plus any clear LCCH. `sigfollow.DispatchAir` decodes signalling on
every slot. `ControlChannel.ProcessAir` frames a Phase 2 control channel and
bootstraps its descrambling seed from a clear-LCCH Network Status Broadcast
— whose parser had been reading the colour code from the channel bytes and
installing a wrong seed.

The settings `p25_phase2_trellis_mode`, `_rs_mode`, `_interleave_mode` and
`_scrambler_mode` are still accepted and no longer read: the air format
fixes the coding. The working-model decoder survives only for its own
tests, pending deletion.

Everything in this post is **reference-pinned** against two decoders and
**synthetic-verified**. The #1273 reporter's site has not confirmed it, so
the on-air gate is open. Two defects in shared primitives turned up on the
way — a scrambler with mirrored taps and a timing loop with a sign error —
and a real Phase 2 carrier was later found in the repository's own samples.

## What this overturns

[Protocol Decoders Part 4]({{ '/blog/deep-dives/protocol-decoders-04-p25-phase-2-tdma-mac/' | relative_url }})
described an ISCH that "prefixes every sub-frame" as a 12-bit field — a
SlotType and a counter — under Golay(24,12,8), and a superframe grid
anchored on one sync in sub-frame 0.
[P25 End to End Part 7]({{ '/blog/deep-dives/p25-end-to-end-07-phase2-tdma/' | relative_url }})
repeated both, added "even sub-frame index → timeslot 0, odd → timeslot 1",
and walked MAC PDUs through a trellis → RS(24,16,9) chain. None of that is
the air format. What stands: 6000 symbols per second, twelve 30 ms slots of
180 dibits, a PN44 seeded from the system's identity, and the MAC messages
themselves. Part 5 of that series gives a band-plan resolver without the
slot divide; for a TDMA channel ID the arithmetic above replaces it.

## What we keep

- **A literal only pins what it did not come from.** The 468.6125 MHz
  expectation was GopherTrunk checking GopherTrunk.
- **"Contained in one file" limits the cost of a fix, not of a guess.** A
  working model needs a reference before it needs a boundary.
- **Read a failure rate as structure before reading it as signal.** A check
  that fails about half the time may be applied to the wrong bits.
- **When two references agree, pin them against each other.** A matrix from
  one and a table from the other cannot share a transcription error.
- **Let an independent decoder accept your encoder.** Five packets through
  someone else's CRC is the nearest thing to air.

## Where this goes next

The framing was half of what was wrong. The scrambler every Phase 2 burst
depends on tapped the wrong bits, the timing loop under the receiver walked
half a symbol off, and the capture that said so had been in `samples/` all
along —
[Part 12]({{ '/blog/solution-postmortem/rebuilt-from-the-air-12-pn44-gardner-real-carrier/' | relative_url }}).

## FAQ

**Why does a P25 Phase 2 grant resolve to the wrong frequency?**
On a TDMA channel ID the grant's channel number counts logical channels, not
carriers: the downlink is the base plus the spacing times the number divided
by the slots per carrier, and the remainder is the timeslot. A resolver that
skips the divide puts a 2-slot grant twice as far from the base as its
carrier.

**How many timeslots does a P25 Phase 2 superframe have, and which carry which call?**
Twelve 30 ms timeslots in 360 ms. The two logical channels alternate —
timeslots 0, 2, 4, 6, 8 and 11 are channel 0 and 1, 3, 5, 7, 9 and 10 are
channel 1 — with the last pair swapped.

**What protects a P25 Phase 2 MAC PDU on the air?**
A shortened, punctured Reed–Solomon RS(63,35) code over six-bit symbols,
then a CRC-12 on FACCH and SACCH or a CRC-16 on the LCCH. Scrambled bursts
are first XORed with the PN44 sequence for their timeslot. There is no
trellis code.

**Is GopherTrunk's P25 Phase 2 decode verified on air?**
The framing, voice positions and MAC coding are pinned to OP25 and SDRTrunk
constants and to packets OP25's own code accepts. The reporter's hybrid site
has not confirmed it; Part 12 covers the first real Phase 2 carrier decoded
with the rebuilt path.

## Series navigation

**Part 11 of 14** · ←
[Part 10: D-STAR — Wrong Sync, Wrong FEC, Wrong Golay]({{ '/blog/solution-postmortem/rebuilt-from-the-air-10-dstar/' | relative_url }})
· Next →
[Part 12: P25 Phase 2 II — Mirrored Taps, a Sign Error, and the Carrier Already in the Repo]({{ '/blog/solution-postmortem/rebuilt-from-the-air-12-pn44-gardner-real-carrier/' | relative_url }})
