---
title: "From the Issue Tracker, Season 2, Part 14: The 0x03 That Read as 0x00 — An Explicit Channel Update Parsed With the Grant Layout"
description: "Why every P25 voice grant on a UHF site resolved to exactly 450.000 MHz and timed out at 7 s: TSBK opcode 0x03 (Group Voice Channel Update – Explicit) was decoded with opcode 0x00's layout, reading the channel from a reserved byte. The real layout, the literal-bit-position pin, the band-plan signature, and what fourteen Season 2 bugs had in common."
category: solution-postmortem
keywords: p25 opcode 0x03, grp_v_ch_grant_updt_exp, p25 grant frequency base_hz, p25 explicit channel update layout, tsbk payload layout sdrtrunk, p25 calls timeout 7s, p25 band plan channel number, literal bit position test, p25 phase 1 control channel, gophertrunk from the issue tracker s2
tags: [from-the-issue-tracker-s2, p25, tsbk, band-plan, parsing, postmortem]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "From the Issue Tracker, Season 2"
series_part: 14
---

*Part 14 of **From the Issue Tracker, Season 2**, a 14-part run of
postmortems on GopherTrunk bugs that fought back.
[Part 13]({{ '/blog/solution-postmortem/issue-tracker-s2-13-whole-span-squelch/' | relative_url }})
replaced a squelch that integrated the whole 2.4 MHz span with one that
measures the channel. This closing part is a P25 bug whose reporter
supplied the clue that solved it
([#1242](https://github.com/MattCheramie/GopherTrunk/issues/1242)): every
grant on their site landed on the band plan's base frequency. It ends
with what the season's fourteen bugs had in common.*

> **TL;DR:** A P25 Phase 1 site (WACN C0907 / SYS 6F5 / NAC 1780, control
> channel 450.2875 MHz, band plan base 450 000 000 Hz, spacing 6250 Hz)
> announced calls with TSBK opcode 0x03, Group Voice Channel Update –
> Explicit. GopherTrunk parsed it with `ParseGroupVoiceChannelGrant`, the
> opcode 0x00 layout `svc | channel | group | source`, but 0x03 is
> `svc | reserved | downlink channel | uplink channel | group` with no
> source unit (SDRTrunk bits 16–79, OP25 `tk_p25.py`). The channel field
> therefore came from the reserved byte plus the downlink channel's high
> byte: channel ID 0, number ≈ 0, which `BandPlan.Frequency` resolves to
> exactly `base_hz` — 450.000 MHz, an empty carrier at −63 dBFS — with a
> garbage talkgroup and source, so every `call.end` read
> `reason=timeout duration=7001ms`. The parser had **no test at all**.
> Fix: `ParseGroupVoiceChannelUpdateExplicit` (`opcodes.go`), the grant
> following the downlink channel, pinned by
> `TestParseGroupVoiceChannelUpdateExplicitLayout` (literal positions via
> `tsbkBits`) and `TestExplicitUpdateGrantResolvesDownlinkChannel`
> (reproduces `frequency_hz=450000000` on the old code; 450 500 000 on the
> new). Signature to recognise: `frequency_hz` equal to an IDEN_UP
> `base_hz` exactly. Reporter confirmation still pending.

**Key takeaways**

- **"Reuse the grant parser" is a shared layout, not a shortcut.** Two
  opcodes with the same first byte and the same width are not the same
  message; 0x03's channel sits one byte later and it carries no source.
- **A frequency equal to `base_hz` is a parsing signature.** Channel 0 of
  any band is the base; a stream of grants that all resolve there is a
  field read from zeros, not a system that uses channel 0.
- **Pin parsers with literal bit positions from an independent decoder.**
  `tsbkBits` builds the payload from SDRTrunk's indexes, so a mistake
  shared by parser and assembler cannot hide.
- **The absence of a test is the finding.** Opcode 0x03 was dispatched for
  as long as the P25 decoder existed, through a parser nothing exercised.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| The two layouts | 0x00 `svc│chan│group│src`; 0x03 `svc│rsvd│dl chan│ul chan│group` | `internal/radio/p25/phase1/opcodes.go` (`ParseGroupVoiceChannelGrant`, `ParseGroupVoiceChannelUpdateExplicit`) |
| Dispatch | follows the **downlink** channel; uplink not followed; no source | `control.go` (`dispatchTSBK`, case `OpGroupVoiceChannelUpdateExpl`) |
| Channel → hertz | `BaseHz + channelNumber × SpacingHz` | `identifier.go` (`BandPlan.Frequency`) |
| Literal pin | SDRTrunk indexes 16–23 / 24–31 / 32–47 / 48–63 / 64–79 | `opcodes_explicit_test.go` (`explicitUpdatePayload`, `tsbkBits`) |
| End-to-end pin | VUHF IDEN_UP base 450 MHz, step 6.25 kHz, channel 0-80 → 450.500 MHz | `TestExplicitUpdateGrantResolvesDownlinkChannel` |
| Grant before band plan | queued 5 s, surfaced as `decode.error` | `publishVoiceGrant`, `pendingGrants` |
| Opcode table | 0x03 = `GRP_V_CH_GRANT_UPDT_EXP` | `docs/specs/p25-tsbk-opcodes.md` |

## In this post

- **What the reporter saw** — every call at 450.000 MHz.
- **Two layouts, one first byte** — a 0x03 payload through the 0x00 read.
- **Pinning it with bits, not round-trips** — `tsbkBits` and the UHF replay.
- **The signature, and what is still unverified** — `frequency_hz == base_hz`.
- **What Season 2 taught** — fourteen bugs, five habits.

## What the reporter saw

The 4 Oct report is a model of a useful one. System: P25 Phase 1, WACN
C0907 / SYS 6F5 / NAC 1780, RFSS 1, Site 4. Control channel 450.2875 MHz,
"channel 0-46, confirmed in band plan". Band plan: base 450 000 000,
spacing 6250, bandwidth 12 500. Observed: the control channel decodes fine
— MBT, identifier updates, TSBKs all OK — but every `call.start` carries
`"frequency_hz": 450000000`, channel 0; every `call.end` is
`reason=timeout, duration=7001ms`; a tap on 450.000 MHz reads −63 dBFS,
the noise floor. Expected: `base + N × 6250`.

Two of those numbers deserve a second look. 7001 ms is not a hangtime an
operator set; it is a timeout *shape*, the same duration on every call,
which says the calls never carried voice at all. And −63 dBFS at
450.000 MHz is the reporter measuring the frequency GopherTrunk chose and
finding nothing there — ruling out the voice tuner, the DDC and the
vocoder before anyone else looked.

Nothing in that list was wrong, and the reply said so: the clue was that
every grant equalled the band plan's *base* frequency. A system does not
put every call on channel 0; a decoder that reads a channel field from
bytes that are always zero does. The voice tuner was sent to an empty
carrier, no voice frame ever arrived, and the composer's no-voice window
— `noVoiceTimeout`, the 3.5 s voice hangtime × `noVoiceStartupFactor` = 2,
hence 7001 ms — tore each call down. The control-channel decoder itself
was healthy, which is why the reporter could see the band plan at all.

## Two layouts, one first byte

The site announces its calls with **Group Voice Channel Update –
Explicit**, TSBK opcode 0x03 (`GRP_V_CH_GRANT_UPDT_EXP` in TIA-102.AABC
terms). GopherTrunk's dispatcher handled it by calling the opcode 0x00
parser:

```go
// internal/radio/p25/phase1/control.go — before b1b4a48
case OpGroupVoiceChannelUpdateExpl:
    c.publishGroupGrant(ParseGroupVoiceChannelGrant(t.Payload), nac)
```

The two eight-byte payloads start identically — service options — and
then diverge:

```text
opcode 0x00  GRP_V_CH_GRANT        : svc(1) │ channel(2)  │ group(2)   │ source(3)
opcode 0x03  GRP_V_CH_GRANT_UPDT_EXP: svc(1) │ reserved(1) │ downlink(2)│ uplink(2) │ group(2)
```

Read a 0x03 payload through the 0x00 parser and every field lands one
byte off or worse. The 16-bit channel field is taken from `p[1:3]`: the
reserved byte (always 0) as its high byte, the downlink channel's high
byte as its low byte. The channel ID is the top nibble — 0 — and the
number is the downlink's high byte, which for channel 80 (`0x0050`) is 0.
`BandPlan.Frequency(0, 0)` is `BaseHz + 0 × SpacingHz` = 450 000 000 Hz.
The group comes from `p[3:5]`, the downlink's low byte joined to the
uplink's high byte, and the source from `p[5:8]`, the uplink's low byte
joined to the real group — garbage both. The message was decoded
*successfully*, into a grant on the wrong frequency for the wrong
talkgroup from a nonexistent radio.

<figure class="lab-figure">
<svg viewBox="0 0 680 200" width="680" height="200" role="img" aria-label="Two eight-byte TSBK payload strips: opcode 0x00 with service options, channel, group and source; opcode 0x03 with service options, a reserved byte, downlink channel, uplink channel and group. A bracket shows the 0x00 channel read covering the reserved byte and the downlink's high byte, resolving to the band plan's base frequency.">
  <text x="340" y="14" text-anchor="middle" fill="currentColor" font-size="10" font-weight="bold">TSBK payload bytes 0..7 (bits 16–79): what the 0x00 read made of a 0x03 message</text>
  <text x="20" y="52" fill="var(--fg-muted)" font-size="9">0x00</text>
  <g font-size="8" text-anchor="middle">
    <rect x="60" y="38" width="70" height="22" fill="none" stroke="var(--fg-muted)"/><text x="95" y="52" fill="var(--fg-muted)">svc</text>
    <rect x="130" y="38" width="140" height="22" fill="none" stroke="var(--fg-muted)"/><text x="200" y="52" fill="var(--fg-muted)">channel (id 4 + num 12)</text>
    <rect x="270" y="38" width="140" height="22" fill="none" stroke="var(--fg-muted)"/><text x="340" y="52" fill="var(--fg-muted)">group</text>
    <rect x="410" y="38" width="210" height="22" fill="none" stroke="var(--fg-muted)"/><text x="515" y="52" fill="var(--fg-muted)">source unit (24)</text>
  </g>
  <g font-size="8" text-anchor="middle" fill="var(--fg-muted)">
    <text x="95" y="80">byte 0</text><text x="165" y="80">1</text><text x="235" y="80">2</text><text x="305" y="80">3</text><text x="375" y="80">4</text><text x="445" y="80">5</text><text x="515" y="80">6</text><text x="585" y="80">7</text>
  </g>
  <text x="20" y="108" fill="var(--accent)" font-size="9">0x03</text>
  <g font-size="8" text-anchor="middle">
    <rect x="60" y="94" width="70" height="22" fill="none" stroke="var(--accent)" stroke-width="1.5"/><text x="95" y="108" fill="var(--accent)">svc</text>
    <rect x="130" y="94" width="70" height="22" fill="none" stroke="var(--accent)" stroke-width="1.5"/><text x="165" y="108" fill="var(--accent)">reserved</text>
    <rect x="200" y="94" width="140" height="22" fill="none" stroke="var(--accent)" stroke-width="1.5"/><text x="270" y="108" fill="var(--accent)">downlink channel</text>
    <rect x="340" y="94" width="140" height="22" fill="none" stroke="var(--accent)" stroke-width="1.5"/><text x="410" y="108" fill="var(--accent)">uplink channel</text>
    <rect x="480" y="94" width="140" height="22" fill="none" stroke="var(--accent)" stroke-width="1.5"/><text x="550" y="108" fill="var(--accent)">group</text>
  </g>
  <path d="M130 124 L130 130 L270 130 L270 124" fill="none" stroke="currentColor"/>
  <text x="200" y="144" text-anchor="middle" fill="currentColor" font-size="8">0x00's channel read = 0x00 ‖ dl high byte → id 0, num 0 → BaseHz</text>
  <text x="340" y="166" text-anchor="middle" fill="var(--accent)" font-size="8">fix: dl = p[2:4], ul = p[4:6], group = p[6:8]; the grant follows dl</text>
  <text x="340" y="186" text-anchor="middle" fill="var(--fg-muted)" font-size="8">reporter's site: dl 0-80 → 450 000 000 + 80 × 6250 = 450 500 000 Hz; old read → 450 000 000 Hz (−63 dBFS, empty)</text>
</svg>
<figcaption>Same first byte, different everything else: the 0x00 channel read lands on the reserved byte, so every grant resolved to the base.</figcaption>
</figure>

The fix (`b1b4a48`, PR #1243) gives 0x03 its own type and parser and
follows the **downlink** channel; the uplink is where the subscribers
transmit, the band plan's `tx_offset_hz` away, and is not a channel to tune:

```go
// internal/radio/p25/phase1/opcodes.go
func ParseGroupVoiceChannelUpdateExplicit(p [8]byte) GroupVoiceChannelUpdateExplicit {
    dl := binary.BigEndian.Uint16(p[2:4])
    ul := binary.BigEndian.Uint16(p[4:6])
    return GroupVoiceChannelUpdateExplicit{
        ServiceOptions:        p[0],
        DownlinkChannelID:     uint8(dl >> 12), DownlinkChannelNumber: dl & 0x0FFF,
        UplinkChannelID:       uint8(ul >> 12), UplinkChannelNumber:   ul & 0x0FFF,
        GroupAddress:          binary.BigEndian.Uint16(p[6:8]),
    }
}
```

`dispatchTSBK` now publishes a `voiceGrant` with the downlink channel,
the group and the service options — and no source ID, because the message
has none. The layout is cross-checked against SDRTrunk's
`GroupVoiceChannelGrantUpdateExplicit` bit indexes and OP25's `tk_p25.py`,
the discipline of
[From Spec to Shipping Part 3]({{ '/blog/deep-dives/from-spec-to-shipping-03-literal-vectors/' | relative_url }}):
a parser is pinned to what independent decoders read off the air, never
to its own inverse.

## Pinning it with bits, not round-trips

Two tests landed with the fix; neither is a round-trip. The parser test
uses `tsbkBits`, the helper the unit-signalling parsers already relied
on: `(start, width, value)` triples in TSBK bit coordinates, laid into a
96-bit frame MSB first, bits 16–79 packed into the eight payload bytes —
an encoder with nothing in common with any `Assemble*` function:

```go
// internal/radio/p25/phase1/opcodes_explicit_test.go
func explicitUpdatePayload(svc, dlID, dlNum, ulID, ulNum, group uint64) [8]byte {
    return tsbkBits(
        [3]uint64{16, 8, svc},
        [3]uint64{32, 4, dlID}, [3]uint64{36, 12, dlNum},
        [3]uint64{48, 4, ulID}, [3]uint64{52, 12, ulNum},
        [3]uint64{64, 16, group},
    )
}
```

`TestParseGroupVoiceChannelUpdateExplicitLayout` feeds
`explicitUpdatePayload(0x40, 1, 0x2AB, 2, 0x3CD, 0xBEEF)` and expects
every field back at its named position. The values are chosen so that a
read one byte off cannot accidentally agree.

The end-to-end test is the reporter's site. `TestExplicitUpdateGrantResolvesDownlinkChannel`
builds a VUHF identifier update — `ChannelID 0, BandwidthHz 12_500,
SpacingHz 6_250, TxOffsetHz 5_000_000, BaseHz 450_000_000` — and an
explicit update naming downlink channel 0-80 and uplink 0-880 for group
`0x0123` with service options `0x40`, runs both through a locked control
channel at NAC `0x780` and 450.2875 MHz via `buildLockedStreamWithTSBK`,
and drains the bus. It expects exactly one grant at
`450_000_000 + 80 × 6_250 = 450_500_000` Hz, group `0x123`, source 0,
`Encrypted` set; the uplink's 0-880 "must not win". On the old code the
same test produces the reporter's exact `frequency_hz=450000000`. Band-plan resolution itself —
`BaseHz + channelNumber × SpacingHz`, and the 5 s `pendingGrants` queue
for a grant that beats its IDEN_UP — is unchanged and already covered in
[P25 End to End Part 5]({{ '/blog/deep-dives/p25-end-to-end-05-channels-band-plans/' | relative_url }}).

## The signature, and what is still unverified

The general lesson is a one-line field check. **A grant whose
`frequency_hz` equals an IDEN_UP's `base_hz` exactly is a field read
from zeros.** Channel 0 exists in every band plan, but a *stream* of
grants resolving there, with talkgroups that match nothing and calls that
only end by timeout, is a parser landing on a zero byte. The
`p25: identifier update` log line prints `base_hz`, so the comparison is
one grep of `debug.log`.

Status: the fix is merged, under `[Unreleased]` in the changelog at the
time of writing. The two tests fail on the old code and
pass on the new, and the layout is pinned against two independent
decoders. The reporter has been asked to confirm three things —
`call.start` shows real voice frequencies, the talkgroups match what a
radio hears, and calls end normally instead of `reason=timeout` at 7 s —
and to expect **no radio ID** on these calls, since an explicit update
carries none. Until that lands, #1242 stays open under the
[closing rule]({{ '/blog/solution-postmortem/from-the-issue-tracker-22-two-pipelines/' | relative_url }})
the project adopted after #764: a green test is a claim of consistency,
and only the reporter's air closes it.

### How the layouts shaped the Go code

- **One type per opcode.** `GroupVoiceChannelUpdateExplicit` has its own
  `Downlink*` and `Uplink*` fields; nothing is reused from
  `GroupVoiceChannelGrant`.
- **`voiceGrant` carries what the message carries.** The dispatch sets
  `groupID`, `channelID`, `channelNumber`, `serviceOptions` and leaves
  `sourceID` zero.
- **Literal-position tests are the pattern.** New TSBK parsers follow the
  `tsbkBits` convention of the unit-signalling opcodes.
- **The parser comment names the bug.** `ParseGroupVoiceChannelUpdateExplicit`'s
  doc comment records the old reuse and why it resolved to the base.

## What Season 2 taught

Fourteen bugs from September and October's reports, and the same few
habits behind nearly all of them.

**The self-consistent trap did not retire with
[Season 1 Part 20]({{ '/blog/solution-postmortem/from-the-issue-tracker-20-self-consistent-trap/' | relative_url }}).**
MDC1200's line code
([Part 1]({{ '/blog/solution-postmortem/issue-tracker-s2-01-tones-were-the-data/' | relative_url }}))
and the DCS codeword
([Part 4]({{ '/blog/solution-postmortem/issue-tracker-s2-04-invented-dcs-codeword/' | relative_url }}))
were invented and round-tripped; the TETRA MAC seams
([Part 11]({{ '/blog/solution-postmortem/issue-tracker-s2-11-phantom-neighbours/' | relative_url }}))
had an encoder sharing the wrong layout; this part's parser had no test
at all. The escape was the same each time: a literal vector from an
independent decoder, or real air.

**Units and rates are a trap of their own.** A threshold in radians per
sample calibrated at 48 kHz and fed 2.4 MS/s
([Part 3]({{ '/blog/solution-postmortem/issue-tracker-s2-03-radians-per-sample/' | relative_url }})),
a FLAC frame header whose sample-rate table ends at 655 350 Hz
([Part 8]({{ '/blog/solution-postmortem/issue-tracker-s2-08-sample-rate-880029/' | relative_url }})),
and a squelch integrating 2.4 MHz instead of 16 kHz
([Part 13]({{ '/blog/solution-postmortem/issue-tracker-s2-13-whole-span-squelch/' | relative_url }}))
all passed every test written at the author's rate.

**Measure what the code actually sees.** Zero-IF clipping products at
4δ ([Part 2]({{ '/blog/solution-postmortem/issue-tracker-s2-02-tone-at-four-delta/' | relative_url }})),
the AACH classification trusted only below two errors (Part 11), the
sixteen-thousand-codeword search behind "overruns at 200 kS/s"
([Part 9]({{ '/blog/solution-postmortem/issue-tracker-s2-09-sixteen-thousand-codewords/' | relative_url }})).

**Build an instrument before a theory.** The GF(2) seed solve
([Part 5]({{ '/blog/solution-postmortem/issue-tracker-s2-05-seed-is-the-source-address/' | relative_url }}))
replaced four wrong colour codes; `solve_rejects`
([Part 6]({{ '/blog/solution-postmortem/issue-tracker-s2-06-false-exact-solves/' | relative_url }}))
caught false exact solves; `tl_sdu_hex` (Part 11) root-caused a splice
from a log alone; `requested_center_hz`
([Part 7]({{ '/blog/solution-postmortem/issue-tracker-s2-07-capture-at-the-wrong-centre/' | relative_url }}))
names a centre that once fell back silently — the
[census-everything]({{ '/blog/solution-postmortem/from-the-issue-tracker-21-census-everything/' | relative_url }})
rule on new ground.

**The pipeline includes the build and the seams between components.**
A re-key fenced by call ID
([Part 10]({{ '/blog/solution-postmortem/issue-tracker-s2-10-rekey-dropped-both-overs/' | relative_url }})),
a binary that was not static
([Part 12]({{ '/blog/solution-postmortem/issue-tracker-s2-12-cgo-disabled-is-not-static/' | relative_url }}))
— the "two pipelines" lesson of
[Season 1 Part 22]({{ '/blog/solution-postmortem/from-the-issue-tracker-22-two-pipelines/' | relative_url }})
extended past DSP into recorders, linkers and web forms.

And the standing rule under every post: a green synthetic is a claim, a
reporter's confirmation is a verification, and the two are logged
separately. Several of this season's fixes still wait on the second —
including this one.

## Where this goes next

The season ends here. The subsystem most of its bugs lived in has its own
tutorial, [The Conventional Scanner]({{ '/blog/series/conventional-scanner/' | relative_url }});
its log lines are read in
[The Field Notebook]({{ '/blog/series/field-notebook/' | relative_url }});
and the meta-lessons it kept re-learning are
[Season 1]({{ '/blog/series/from-the-issue-tracker/' | relative_url }})'s
closing parts. The next issue is already open.

## FAQ

**Why did all my P25 grants show frequency_hz equal to the band plan base?**
Your site announces calls with opcode 0x03 (Group Voice Channel Update –
Explicit), which GopherTrunk parsed with opcode 0x00's layout before
`b1b4a48`. The channel field was read from the reserved byte, giving
channel ID 0 and number 0, which `BandPlan.Frequency` resolves to
`base_hz`. The fix reads the downlink channel at bytes 2–3.

**What is the layout of P25 TSBK opcode 0x03?**
Service options (byte 0), a reserved byte, the downlink channel (4-bit ID
+ 12-bit number, bytes 2–3), the uplink channel (bytes 4–5) and the 16-bit
group address (bytes 6–7) — SDRTrunk bit indexes 16–79. It carries no
source unit, so grants from it have no radio ID.
`ParseGroupVoiceChannelUpdateExplicit` is the parser.

**Why follow the downlink channel and not the uplink?**
The downlink is what the repeater transmits and what a scanner hears; the
uplink is the subscribers' transmit side, `tx_offset_hz` away (+5 MHz in
the regression's band plan). `TestExplicitUpdateGrantResolvesDownlinkChannel`
pins that uplink channel 0-880 must not win over downlink 0-80.

**How do I recognise this class of bug on another opcode?**
Compare grant `frequency_hz` against the `base_hz` values in the `p25:
identifier update` log lines. A run of grants that all equal a base
exactly, with talkgroups that match nothing and calls ending only by
timeout, means a channel field is read from zero bytes — a layout
mismatch, not a site using channel 0.

**Is the #1242 fix verified on air?**
Not yet. The parser is pinned with literal bit positions from SDRTrunk and
the end-to-end test reproduces the reporter's `frequency_hz=450000000` on
the old code, but the reporter's confirmation — real voice frequencies,
matching talkgroups, calls ending normally — is the gate, and the issue
stays open until it lands.

## Series navigation

**Part 14 of 14** · ←
[Part 13: The Whole-Span Squelch — A Scanner Gated by Every Carrier in 2.4 MHz]({{ '/blog/solution-postmortem/issue-tracker-s2-13-whole-span-squelch/' | relative_url }})
· Season 1: [From the Issue Tracker]({{ '/blog/series/from-the-issue-tracker/' | relative_url }})
· This season: [From the Issue Tracker, Season 2]({{ '/blog/series/from-the-issue-tracker-s2/' | relative_url }})
