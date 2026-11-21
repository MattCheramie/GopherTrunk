---
title: "Rebuilt From the Air, Part 7: NXDN I — The Frame Sync Was Twenty Bits, Not Sixteen"
description: "GopherTrunk opened NXDN calls, counted voice frames and produced no audio: its frame layer searched for a 16-bit sync word, a 32-dibit SACCH and no scrambler, none of which exist on air. A reporter's two captures and two reference decoders rebuilt it around a 20-bit sync, PN9 scrambling and DMR's AMBE+2 layout."
category: solution-postmortem
keywords: nxdn decoder frame sync, nxdn fsw 0xcdf59, nxdn pn9 scrambler, nxdn lich parity, nxdn48 vs nxdn96, nxdn ambe+2 dmr layout, nxdn voice no audio sdr, nxdn carrier offset afc, nxdn adjacent channel filter, gophertrunk nxdn
tags: [rebuilt-from-the-air, nxdn, self-consistent-trap, real-air-captures, reference-implementations]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "Rebuilt From the Air"
series_part: 7
---

*Part 7 of **Rebuilt From the Air**, a 14-part postmortem series on the
GopherTrunk decoders that passed every test and never matched a real radio —
each diagnosed from a reference implementation or a reporter's capture,
rebuilt against independent decoders, and pinned to real air where a capture
exists.
[Part 6]({{ '/blog/solution-postmortem/rebuilt-from-the-air-06-ysf-fich/' | relative_url }})
rebuilt the System Fusion header from a reference alone. This part opens
three on NXDN, and it has what LoRa and Fusion lacked: a reporter with the
radios, two recordings of them, and a log line that counted sixteen voice
frames the decoder had never actually found.*

> **TL;DR:** A `mode: nxdn` scan-list channel opened calls and logged
> `voice_frames=16`, then `voice_frames=12`, with no audio
> ([#1261](https://github.com/MattCheramie/GopherTrunk/issues/1261)). The
> counts were noise matching a sync word no radio sends. The frame layer in
> `internal/radio/nxdn` modelled a 16-bit FSW `0xC55A`, a 32-dibit SACCH and
> no scrambler; the air carries a 20-bit FSW `0xCDF59`, an 8-dibit LICH, a
> 30-dibit SACCH and 144 information dibits, with every dibit after the sync
> inverted by a PN9 sequence. `airframe.go` now frames that, and
> `DecodeAirVoice` hands each 72-bit voice frame to
> `dmrvoice.DecodeAMBEFrame`, because the layout is DMR's verbatim. Both
> captures sat about 400 Hz off, so the voice receiver gained `EnableAFC` and
> a per-mode channel filter. Two real-air slices pin it in CI: 36 and 56
> voice frames where the old code found none.

**Key takeaways**

- **A counter that counts sync matches is not evidence of frames.** A
  16-bit word at tolerance 1 matches random dibits about twice a second.
- **The invented frame had the right length.** Both layouts are 192 dibits
  with the information field at dibit 48, so nothing about its size looked
  wrong.
- **NXDN48 is NXDN96 at half the symbol rate.** Same four-level frame, 2400
  symbols a second — not a two-level mode.
- **A plausible operator fix can be a coincidence that points somewhere
  real.** `narrowband_filter` does nothing on an RTL-SDR; the missing filter
  was in the NXDN receiver.

## Cheat sheet

| | |
|---|---|
| Issue | [#1261](https://github.com/MattCheramie/GopherTrunk/issues/1261) — NXDN on a scan-list channel: detected, no audio |
| Symptom | calls opened on the reporter's 447.100 MHz channel, ended `voice_frames=16` and `voice_frames=12`, and recorded nothing intelligible in either channel width |
| Wrong theories | the frame layer is fine and a non-zero frame count points at the vocoder; narrow NXDN is a 2-level mode needing its own receiver; the SDR's `narrowband_filter` switch fixed NXDN48 |
| Real cause | fabricated framing — a 16-bit sync word, a doubled-bit LICH, a 32-dibit SACCH, no scrambler, a placeholder voice bit order — plus a voice receiver with no carrier correction and no channel filter |
| Fix | `nxdn/airframe.go` and `TrafficChannel`; `EnableAFC` and `EnableChannelFilter` on the voice receiver; `mode: nxdn48` |
| Diagnostic | the reporter's two `voice_iq_debug` recordings read against MMDVMHost and DSD-FME; speech judged by AMBE pitch continuity, never by level |
| Status | capture-verified on two real-air slices in CI; on-air verified on the reporter's Kenwood radios; control channel, drift and scrambler follow in Parts 8 and 9 |

## In this post

- **Sixteen voice frames and no voice** — what the reporter's log said.
- **Three wrong theories** — the vocoder, the two-level mode, the hardware filter.
- **The diagnostic: two captures against two references** — and a metric that cannot be fooled by loudness.
- **The frame on the air** — FSW, scrambler, LICH, SACCH, voice.
- **The fix, and two receiver gaps the captures exposed** — AFC and the channel filter.
- **What this overturns** — Protocol Decoders 6's NXDN half.

## Sixteen voice frames and no voice

The scan list had just learned digital modes when the reporter ran five
SDRs at once and listed what worked. P25, air band and analog FM did.
[NXDN]({{ '/reference/nxdn/' | relative_url }}) did not: "detected but no
audio is recovered", tried in both channel widths. The next day's log, from
two five-second keyups on 447.100 MHz, showed how far the decoder thought it
got:

```text
composer: nxdn voice chain started (VCH decode is unverified on air — experimental) serial=R1
composer: nxdn voice chain ended serial=R1 voice_frames=16
composer: nxdn voice chain ended serial=R1 voice_frames=12
```

For scale, a slice of the reporter's 15 October recording, now in the test
suite, carries 40 voice frames in 0.8 seconds. Sixteen in five seconds is
not a weak decode of that; it is a different quantity.

## Three wrong theories

**The frame layer works; the vocoder stage is the unverified part.** The
start-up line said as much — "VCH decode is unverified on air" — and the
first reply on the issue gave the reporter a rule built on it: a count of
zero means the voice frames are not being found, a non-zero count with bad
audio points at the vocoder. By that rule sixteen frames meant framing was
fine. The rule assumed the counter counted frames.

**Narrow NXDN is a different modulation.** The package comment described a
"4800 baud (BFSK)" rate with one bit per symbol beside the four-level 9600
one, and the receiver called the two-level variant "a separate receiver
path". So the 6.25 kHz failure was filed as expected — not decoded at all —
and only the 12.5 kHz one as a bug.

**A hardware filter fixes NXDN48.** This one came after the rebuild and is
taken up below, because it was wrong in an instructive way.

## The diagnostic: two captures against two references

The reporter sent `voice_iq_debug` recordings of both widths — the
composer's own per-call IQ, so exactly what the voice chain had been
handed. Two decoders that are proven on air were read beside them, for
protocol facts only: MMDVMHost's NXDN definitions, LICH and scrambler
table, and DSD-FME's NXDN frame, deinterleave and voice sources. Nothing was
ported. The two agree on every constant that follows, and they share
almost nothing with the layout GopherTrunk had.

Read that way the log explained itself. The old sync search took an
8-dibit word at tolerance 1. Twenty-five of the 65,536 possible 8-dibit
patterns pass that test, so random dibits at 4800 a second match about
twice a second. The "voice frames" were those false syncs, each followed by
a block of dibits decoded as if it were voice.

Judging the rebuild needed a metric that could not be fooled the same way.
Random bits through an AMBE+2 vocoder are loud, so level proves nothing.
The test uses pitch continuity instead — the verdict
[DMR Enhanced Privacy]({{ '/blog/deep-dives/dmr-end-to-end-12-enhanced-privacy-rc4/' | relative_url }})
had already adopted: the share of consecutive voice frames whose 7-bit
pitch index `b0` moves by ten or less. Speech scores above about 0.5; random
data scores about 0.16.

## The frame on the air

| Field | Invented layout | On the air |
|---|---|---|
| Frame sync | 16 bits, 0xC55A outbound and its complement inbound | 20 bits, `0xCDF59`, ten dibits (`AirFSW`) |
| LICH | 8 bits, each sent twice; a 1-bit channel type; even parity over seven bits | 8 dibits, one bit in each dibit's MSB; a 2-bit channel type; parity over the top four bits (`ParseAirLICH`) |
| SACCH | 32 dibits | 30 dibits (`AirSACCHDibits`) |
| Information | 144 dibits; voice in a placeholder bit order | 144 dibits; on a voice frame, four 72-bit AMBE+2 frames in DMR's layout (`DecodeAirVoice`) |
| Scrambler | none | PN9 over every dibit after the sync word (`DescrambleAir`) |
| NXDN48 | a two-level mode | the same frame at 2400 symbols a second (`SymbolRateNXDN48`) |

Add the columns up and both come to 192 dibits, with the information field
starting at dibit 48 either way: 8 + 8 + 32 on one side, 10 + 8 + 30 on the
other. That is how a
[fabricated framing]({{ '/reference/self-consistent-test-trap/' | relative_url }})
survives review. Its totals check out against any summary of the standard.

<figure class="lab-figure">
<svg viewBox="0 0 680 230" width="680" height="230" role="img" aria-label="Two NXDN frame layouts of 192 dibits. The invented one has an 8-dibit frame sync word 0xC55A, an 8-dibit LICH with doubled bits, a 32-dibit SACCH and a 144-dibit information field, with no scrambler. The on-air one has a 10-dibit frame sync word 0xCDF59, an 8-dibit LICH, a 30-dibit SACCH and a 144-dibit information field holding four 36-dibit AMBE+2 voice frames, and a PN9 scrambler covers every dibit after the sync word. The information field starts at dibit 48 in both.">
  <text x="10" y="16" fill="var(--fg-muted)" font-size="9" font-weight="bold">the invented frame — no radio sends it</text>
  <rect x="10" y="24" width="70" height="34" fill="none" stroke="var(--fg-muted)" stroke-dasharray="4 3"/>
  <text x="45" y="39" text-anchor="middle" fill="var(--fg-muted)" font-size="9">FSW 0xC55A</text>
  <text x="45" y="52" text-anchor="middle" fill="var(--fg-muted)" font-size="8">8 dibits</text>
  <rect x="80" y="24" width="80" height="34" fill="none" stroke="var(--fg-muted)" stroke-dasharray="4 3"/>
  <text x="120" y="39" text-anchor="middle" fill="var(--fg-muted)" font-size="9">LICH</text>
  <text x="120" y="52" text-anchor="middle" fill="var(--fg-muted)" font-size="8">bits sent twice</text>
  <rect x="160" y="24" width="120" height="34" fill="none" stroke="var(--fg-muted)" stroke-dasharray="4 3"/>
  <text x="220" y="39" text-anchor="middle" fill="var(--fg-muted)" font-size="9">SACCH</text>
  <text x="220" y="52" text-anchor="middle" fill="var(--fg-muted)" font-size="8">32 dibits</text>
  <rect x="280" y="24" width="390" height="34" fill="none" stroke="var(--fg-muted)" stroke-dasharray="4 3"/>
  <text x="475" y="39" text-anchor="middle" fill="var(--fg-muted)" font-size="9">information field · 144 dibits</text>
  <text x="475" y="52" text-anchor="middle" fill="var(--fg-muted)" font-size="8">no scrambler · placeholder voice bit order</text>
  <text x="10" y="92" fill="currentColor" font-size="9" font-weight="bold">the frame on the air — NXDN96 and NXDN48 alike</text>
  <rect x="10" y="100" width="84" height="34" fill="none" stroke="var(--accent)" stroke-width="1.5"/>
  <text x="52" y="115" text-anchor="middle" fill="var(--accent)" font-size="9">FSW 0xCDF59</text>
  <text x="52" y="128" text-anchor="middle" fill="var(--accent)" font-size="8">10 dibits</text>
  <rect x="94" y="100" width="80" height="34" fill="none" stroke="currentColor"/>
  <text x="134" y="115" text-anchor="middle" fill="currentColor" font-size="9">LICH</text>
  <text x="134" y="128" text-anchor="middle" fill="currentColor" font-size="8">8 dibits · MSB</text>
  <rect x="174" y="100" width="106" height="34" fill="none" stroke="currentColor"/>
  <text x="227" y="115" text-anchor="middle" fill="currentColor" font-size="9">SACCH</text>
  <text x="227" y="128" text-anchor="middle" fill="currentColor" font-size="8">30 dibits</text>
  <rect x="280" y="100" width="98" height="34" fill="none" stroke="currentColor"/>
  <text x="329" y="115" text-anchor="middle" fill="currentColor" font-size="9">AMBE+2</text>
  <text x="329" y="128" text-anchor="middle" fill="currentColor" font-size="8">36 dibits</text>
  <rect x="378" y="100" width="97" height="34" fill="none" stroke="currentColor"/>
  <text x="426" y="115" text-anchor="middle" fill="currentColor" font-size="9">AMBE+2</text>
  <text x="426" y="128" text-anchor="middle" fill="currentColor" font-size="8">36 dibits</text>
  <rect x="475" y="100" width="98" height="34" fill="none" stroke="currentColor"/>
  <text x="524" y="115" text-anchor="middle" fill="currentColor" font-size="9">AMBE+2</text>
  <text x="524" y="128" text-anchor="middle" fill="currentColor" font-size="8">36 dibits</text>
  <rect x="573" y="100" width="97" height="34" fill="none" stroke="currentColor"/>
  <text x="621" y="115" text-anchor="middle" fill="currentColor" font-size="9">AMBE+2</text>
  <text x="621" y="128" text-anchor="middle" fill="currentColor" font-size="8">36 dibits</text>
  <line x1="94" y1="146" x2="670" y2="146" stroke="var(--accent)"/>
  <line x1="94" y1="141" x2="94" y2="151" stroke="var(--accent)"/>
  <line x1="670" y1="141" x2="670" y2="151" stroke="var(--accent)"/>
  <text x="382" y="162" text-anchor="middle" fill="var(--accent)" font-size="8">PN9 scrambler, one bit per dibit: where the sequence is 1 the symbol is inverted — every dibit after the FSW</text>
  <line x1="280" y1="62" x2="280" y2="96" stroke="var(--fg-muted)" stroke-dasharray="2 2"/>
  <text x="340" y="190" text-anchor="middle" fill="var(--fg-muted)" font-size="8">both layouts total 192 dibits and put the information field at dibit 48 — the frame's size never looked wrong</text>
  <text x="340" y="208" text-anchor="middle" fill="var(--fg-muted)" font-size="8">192 symbols: 40 ms at NXDN96 (4800 symbols a second) · 80 ms at NXDN48 (2400 symbols a second)</text>
</svg>
<figcaption>Same length, same payload offset, and not one field in common before the information field: a 16-bit sync search cannot find a 20-bit sync word, and without the descrambler nothing after it reads true.</figcaption>
</figure>

**The sync word and the scrambler.** The sync word is the only part of the
frame sent as-is. Everything after it is
[scrambled]({{ '/reference/scrambling/' | relative_url }}) symbol by symbol:

```go
// internal/radio/nxdn/airframe.go (shape)
var AirFSW = [AirFSWDibits]uint8{3, 0, 3, 1, 3, 3, 1, 1, 2, 1} // 0xCDF59

lfsr := uint16(0xE4)
for i := range t { // one PN bit per post-FSW dibit
	t[i] = uint8(lfsr & 1)
	fb := (lfsr ^ lfsr>>4) & 1
	lfsr = lfsr>>1 | fb<<8
}
// DescrambleAir: post[i] ^= airScramble[i] << 1 — flip the dibit's MSB
```

A set bit inverts the symbol, and in this package's dibit mapping a symbol
inversion is a flip of the dibit's most significant bit.
`TestAirScrambleMatchesMMDVMHostTable` checks the generated sequence against
MMDVMHost's scrambler table typed in as 48 literal bytes — an independent
encoder's constant, not GopherTrunk's own output.

**The LICH.** Eight dibits carry eight bits, one in each dibit's MSB:
channel type (2), function (2), option (2), direction (1) and a parity bit
that covers only the first four. `TestParseAirLICHOnAirValues` pins values
read off the reporter's captures: `0xAC`, conventional voice in both halves
of the frame, and `0xA0`, a signalling frame with no voice. That second
value matters. NXDN96 alternates voice frames with signalling frames, and
the LICH's option bits say which halves of the information field hold voice.
The reporter's radios were talking direct, so their channel type is the
conventional one, RDCH; the old gate tested for trunked traffic only.

**The voice.** A voice frame's 144 information dibits are four 72-bit
[AMBE+2]({{ '/reference/ambe-plus-2-fec/' | relative_url }}) frames, and
this is the pleasant surprise of the rebuild: both references use, for
NXDN, exactly the deinterleave tables and FEC they use for DMR.

```go
// internal/radio/nxdn/airframe.go (shape)
lich := ParseAirLICH(post)
if !lich.ParityOK {
	return lich, nil
}
first, second := lich.VoiceHalves()
for v := 0; v < 4; v++ { // four 36-dibit voice frames
	if (v < 2 && !first) || (v >= 2 && !second) {
		continue
	}
	payload, errs, err := dmrvoice.DecodeAMBEFrame(bits) // DMR's layout, verbatim
	// …append VCHFrame{Payload: payload, Errors: errs, Slot: v}
}
```

So NXDN voice runs through a decoder that
[DMR had already proven on air]({{ '/blog/deep-dives/dmr-end-to-end-11-ambe2-silence-frames/' | relative_url }}),
and the placeholder bit order was deleted instead of calibrated.

## The fix, and two receiver gaps the captures exposed

`TrafficChannel` replaces the old voice path: an `airFramer` hunts the
20-bit sync at one symbol error cold and three at the expected position
once locked, collects 182 dibits, descrambles, checks the LICH and decodes
what it announces. NXDN48 became `mode: nxdn48`, the same decoder with
`SymbolRateHz` set to 2400.

`TestTrafficChannelDecodesRealAirNXDN` is the pin: two 48 kS/s slices of
the reporter's recordings, 0.8 s of NXDN96 and 1.2 s of NXDN48, committed
under `internal/radio/nxdn/testdata/`. When the rebuild landed they decoded
36 voice frames at a pitch continuity of 0.86 and 56 at 0.82, where the old
path produced nothing that scored as speech. (Run today the NXDN96 slice
logs 40 voice frames at 0.85: one more air frame is framed, thanks to the
carrier tracker
[Part 8]({{ '/blog/solution-postmortem/rebuilt-from-the-air-08-nxdn-control-channel/' | relative_url }})
adds.) The test also bounds Golay corrections at one per frame.

The captures were not centred, and that exposed the first receiver gap.
Both sat about 400 Hz off frequency — ordinary for two radios and a dongle.
NXDN48's inner symbols are only about ±350 Hz from centre, so with no
correction the NXDN48 slice frames nothing at all. The voice receiver now
runs `EnableAFC`; the control-channel receiver had kept it off because the
invented control frame has long constant-symbol runs that walk a tracker
off, a worry the scrambled real frame does not share.

**The red herring.** On a build with the fix the reporter confirmed NXDN96
live — good audio, fast detection — and found NXDN48 "failing to
capture/decode every other transmission" until they set `narrowband_filter`
on the SDR, after which it decoded every time. That switch engages a
hardware filter on a HackRF Pro and nothing else. On the reporter's RTL-SDR
the pool logs `narrowband_filter requested but this driver has no
switchable filter (HackRF Pro only)` and carries on. Whatever changed
between the two runs, it was not that.

But the request behind it — a narrow filter for a narrow mode — described a
real hole. The voice chain's front end barely band-limits at a 2.4 MS/s SDR
rate, so the FM discriminator was seeing roughly ±24 kHz around a signal a
few kilohertz wide. `EnableChannelFilter` now puts a low-pass ahead of the
discriminator: ±6.25 kHz for NXDN96, ±3.5 kHz for NXDN48, chosen from the
channel's mode. `TestNXDNVoiceReceiverRejectsAdjacentChannel` adds a
same-power copy of each capture one channel away: without the filter the
receiver decodes 0 voice frames of NXDN96 and 12 of NXDN48's 56; with it,
the whole call on both.

## What this overturns

The NXDN half of
[Protocol Decoders Part 6]({{ '/blog/deep-dives/protocol-decoders-06-nxdn-dpmr/' | relative_url }})
described the invented layout:

- Its frame figure and `frame.go` excerpt (FSW 8 dibits, SACCH 32) and the
  sync words `0xC55A` / `0x3AA5` are not the air format.
- The LICH is not a doubled-bit repetition code with a one-bit channel
  type.
- "BFSK 4800 / 4-FSK 9600 … only the symbol mapping differs" is wrong: both
  widths are four-level, and the symbol rate is what differs.
- Its CAC section is overturned separately in Part 8.

The glossary entries for the
[NXDN frame sync word]({{ '/reference/nxdn-fsw/' | relative_url }}) and
[frame structure]({{ '/reference/nxdn-frame-structure/' | relative_url }})
were written from the same model. The legacy files are still in the
package for their own tests; nothing on the voice path reads them.

Status, in the project's vocabulary: the voice frame layer is
**capture-verified** — two real-air slices in CI — and **on-air verified**
on the reporter's Kenwood radios, NXDN96 first and both widths once the
drift fix in Part 8 landed. The scan-list
[call gate]({{ '/blog/deep-dives/squelch-09-call-gates-ysf-nxdn-dstar/' | relative_url }})
and the operator recipe in
[Cookbook Volume 2 Part 3]({{ '/blog/tutorials/cookbook-volume-2-03-nxdn/' | relative_url }})
build on it.

## What we keep

- **Make counters count verified things.** `voice_frames` now sits beside
  `frames` and `lich_ok` in the end-of-call line; a healthy call has
  `lich_ok` close to `frames`.
- **A total that adds up proves nothing about the parts.** The
  [self-consistent trap]({{ '/blog/solution-postmortem/from-the-issue-tracker-20-self-consistent-trap/' | relative_url }})
  kept a 192-dibit frame with every interior boundary wrong.
- **Two references that agree on every constant, then a capture.** The
  [SmartNet rebuild]({{ '/blog/deep-dives/from-spec-to-shipping-08-smartnet-rebuild/' | relative_url }})
  had the first; this one got the second within a day.
- **Judge a voice decode by structure.** Pitch continuity separates speech
  from random bits; loudness does not.
- **Check what a config key touches before crediting it.** An operator's
  observation is data; the cause attached to it is a hypothesis.

## Where this goes next

Conventional NXDN voice decoded. The trunked control channel was still on
the invented layout, and the reporter's next report was that NXDN worked for
five to ten minutes and then failed until the ppm setting was nudged by
one.
[Part 8]({{ '/blog/solution-postmortem/rebuilt-from-the-air-08-nxdn-control-channel/' | relative_url }})
follows both: a carrier drifting through a timing loop, and 214 control
frames out of 214.

## FAQ

**Why did GopherTrunk detect NXDN calls but record no audio?**
Its NXDN frame layer searched for a 16-bit sync word and a frame layout
that radios do not send, did not remove the scrambler, and read voice bits
in a placeholder order. The frames it counted were chance matches of that
sync word. The voice path was rebuilt on the real format in v1.3.0.

**What is the real NXDN frame layout?**
192 dibits: a 20-bit frame sync word `0xCDF59`, an 8-dibit LICH, a 30-dibit
SACCH and a 144-dibit information field. Every dibit after the sync word
is scrambled by a 9-bit PN sequence that inverts symbols.

**Is NXDN48 a different modulation from NXDN96?**
No. Both are four-level FSK with the same 192-dibit frame. NXDN96 runs at
4800 symbols a second in a 12.5 kHz channel, NXDN48 at 2400 in 6.25 kHz. Use
`mode: nxdn` or `mode: nxdn48` to match the radio.

**Does `narrowband_filter` help NXDN48 on an RTL-SDR?**
No. It switches a hardware filter on a HackRF Pro only; on other drivers
the daemon logs a warning and ignores it. The NXDN receiver now applies its
own channel filter automatically, ±3.5 kHz for `nxdn48`.

**Is NXDN voice decoding verified on air?**
Conventional voice is: capture-verified on two committed real-air slices,
and confirmed live by the reporter on Kenwood radios in both channel
widths. The control channel and the scrambler have their own status in
Parts 8 and 9.

## Series navigation

**Part 7 of 14** · ←
[Part 6: System Fusion — A FICH of 100 Dibits, Not 100 Bits]({{ '/blog/solution-postmortem/rebuilt-from-the-air-06-ysf-fich/' | relative_url }})
· Next →
[Part 8: NXDN II — The Control Channel on Real Air, 214 of 214]({{ '/blog/solution-postmortem/rebuilt-from-the-air-08-nxdn-control-channel/' | relative_url }})
