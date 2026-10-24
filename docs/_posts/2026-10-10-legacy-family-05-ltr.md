---
title: "The Legacy Family End to End, Part 5: LTR — Subaudible Data on Every Repeater, No Control Channel"
description: "GopherTrunk's LTR decoder read at the bit level — the 41-bit status word every repeater sends at 300 bps under its voice, the sub-audible receiver and its Manchester modes, the CRC-7 that covers 24 of those bits per sdrtrunk's table, the per-repeater lock the hunter treats as a control channel, the grant the engine drops for want of a frequency, and the rung it honestly stands on."
category: deep-dives
keywords: ltr trunking decoder, logic trunked radio sdr, ltr status word 41 bits, ltr subaudible 300 baud, ltr manchester decode, ltr crc-7 fcs sdrtrunk, ltr home repeater area code, ltr_fcs_mode, ltr_manchester_mode, ef johnson ltr, gophertrunk legacy family
tags: [legacy-family-end-to-end, ltr, trunking, subaudible, fec, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "The Legacy Family End to End"
series_part: 5
---

*Part 5 of **The Legacy Family End to End**, a 14-part deep dive into the
FM-era trunking generation and the AMBE-era narrowband and amateur modes
through GopherTrunk.
[Part 4]({{ '/blog/deep-dives/legacy-family-04-edacs/' | relative_url }})
read EDACS, a fast control channel of block-coded words. LTR — Logic
Trunked Radio, E.F. Johnson's 1970s design — has no control channel at
all: every repeater transmits its own 41-bit status word at 300 bits per
second underneath the voice, and a radio or a scanner assembles the
system from all of them. This part reads `internal/radio/ltr` from the
sub-audible receiver to the grant, and is candid about where the
package's layout comes from and which path its only end-to-end test
exercises.*

> **TL;DR:** LTR is **distributed trunking**: a 41-bit `Status` word
> at **300 bps** under each repeater's voice — `Sync` (always 1), 5-bit
> `Area`, the `Group` F-bit, 4-bit `Channel`, 5-bit `Home`, 8-bit
> `GroupID`, 5-bit `Free`, 12-bit `FCS` (`status.go`). The receiver is
> FM → a 101-tap Kaiser low-pass at 300 Hz → Mueller-Müller at 300 baud
> (160 samples/symbol at 48 kHz) → zero slicer. `ControlChannel.Process`
> Manchester-decodes (`ltr_manchester_mode`, default `ManchesterSoft`),
> then aligns on a `1` with room for 41 bits. `Ingest` verifies a
> **CRC-7** over a 24-bit message (`ltr_fcs_mode`, default `FCSOn`,
> table copied from sdrtrunk's `CRCLTR.java`), locks on the first valid
> word — `LockedNAC` = `Area<<8 | Repeater` — and publishes a grant when
> `IsActive()` (`Group && GroupID != 0`), de-duplicated by `activeGroup`.
> `newLTRPipeline` wires no `Resolver` and no `Area`, so grants carry
> `FrequencyHz: 0` and the engine drops them. `TestDaemonCCDecodesLTR`
> runs with Manchester and FCS **off**. Rung: **reference-pinned** FCS;
> the 41-bit layout is "the most-cited public reference" with a
> cross-check caveat in the code.

**Key takeaways**

- **"Locked" means "hearing a repeater."** There is no control channel;
  the first well-formed status word on a frequency publishes
  `cc.locked`, and the hunter reads it like any other.
- **A grant is a state, not a message.** `IsActive()` — F-bit set and a
  non-zero group — republishes as `KindGrant`; `activeGroup` suppresses
  the repeats and an idle word clears it.
- **The FCS covers 24 bits, not 41.** `fcs.go` builds sdrtrunk's
  message — the F-bit as a 1-bit "Area", then Channel, Home, Group,
  Free — and compares the CRC-7 to the low 7 bits of the 12-bit field.
- **The production defaults are untested end to end.** The integration
  test sets `ltr_manchester_mode: off` and `ltr_fcs_mode: off`; the
  on-air path (`ManchesterSoft` + `FCSOn`) has unit tests only.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| Status word | 41 bits: Sync 40 · Area 39..35 · Group 34 · Chan 33..30 · Home 29..25 · GroupID 24..17 · Free 16..12 · FCS 11..0 | `internal/radio/ltr/status.go` |
| Activity | `IsActive()` = `Group && GroupID != 0`; `IsWellFormed()` = Sync, Channel 1..20, Home 1..20 | `status.go` |
| Receiver | FM → Kaiser LPF (`LPFLen` 101, `LPFCutoffHz` 300, `LPFBeta` 8.6) → M&M 300 baud → slicer | `receiver/receiver.go` |
| Manchester | `ManchesterOff` / `ManchesterStrict` / `ManchesterSoft`; `ParseManchesterMode("")` → Soft | `process.go`, `framing/manchester.go` |
| Alignment | first `1` with 41 bits of room; unlock when `window[0] != 1` | `process.go` (`statusBits` 41) |
| FCS | CRC-7 over 24 message bits, `framing.CRC7LTR`, table from sdrtrunk | `fcs.go`, `framing/crc_ltr.go` |
| Lock | `LockState{FrequencyHz, Area, Repeater}`, `LockedNAC` packs both | `control.go` |
| Band plan | `LinearBandPlan` / `TableBandPlan` — unwired in `newLTRPipeline` | `bandplan.go`, `ccdecoder/pipelines.go` |

## In this post

- **A system with no control channel** — what "locked" means here.
- **The 41-bit status word** — layout, semantics, caveat.
- **The sub-audible receiver and the Manchester modes** — 300 Hz, 300 baud, three decoders.
- **The FCS** — 24 bits under a CRC-7, and a documented layout tension.
- **Following a call** — `Ingest`'s order, the lock, the grant, the drop.
- **The rung** — what the tests pin, and the path they skip.

## A system with no control channel

The package doc sets the frame: "every repeater transmits its own 41-bit
status word at 300 bps, on top of the in-band voice. There is no central
control channel; an LTR scanner follows calls by watching every
repeater's status word and tuning to whichever one currently announces
the talkgroup of interest."

GopherTrunk models that with the same `ControlChannel` shape as the
rest of the family, bound to one frequency — one repeater. Its
`LockState` is `{FrequencyHz, Area, Repeater}`, and its doc is precise
about the word: "LTR has no central control channel, so 'locked' here
means 'we're receiving valid status words from the repeater on this
frequency'." The hunter does not know the difference. `LockedNAC` packs
`(Area << 8) | Repeater` into the slot P25 uses for a NAC, and `cchunt`
type-asserts the payload like any other. One consequence follows from
`site.go`: `CampsWhenIdle` returns true only for `ProtocolDMRTier2`,
`ProtocolDMRTier1` and `ProtocolTETRADMO`, so an LTR repeater that is
simply quiet during a hunt round is a failed acquisition, not a camped
channel — the hunter backs off rather than waiting. The package's "Still
NOT wired" note is LTR-Net repeater-pair coordination: "Each repeater is
tracked on its own."

## The 41-bit status word

`status.go` lays the word out MSB-first:

```go
// internal/radio/ltr/status.go
type Status struct {
    Sync    bool   // bit 40 — frame-start marker (always 1)
    Area    uint8  // bits 39..35 — area code (0..31)
    Group   bool   // bit 34 — the "F-bit": 1 = call active for GroupID
    Channel uint8  // bits 33..30 — physical channel (1..20)
    Home    uint8  // bits 29..25 — home repeater (1..20)
    GroupID uint16 // bits 24..17 — group / talkgroup ID (1..250)
    Free    uint8  // bits 16..12 — free-repeater hint (for handoff)
    FCS     uint16 // bits 11..0 — frame check
}
```

`AssembleStatus` packs it into six bytes left-aligned (41 bits shifted
up by 7), `ParseStatus` reverses it, and `StatusBits` / `StatusFromBits`
convert to and from a 41-entry bit slice; `TestStatusAssembleParseRoundTrip`
and `TestStatusFromBitsRoundTrip` cover the pair. The semantics the
comments give are the ones the state machine uses: `Channel` is the
physical channel this word references, `Home` the home repeater of the
active group, `Free` a hint at which repeater is currently unallocated,
and `Area` disambiguates LTR systems sharing a frequency. Two predicates
carry the logic. `IsActive()` is `Group && GroupID != 0` — the F-bit set
with a non-zero group, the convention the package treats as "a call is
up." `IsWellFormed()` requires `Sync` plus `Channel` and `Home` both in
1..20, because "either field being zero means the frame was almost
certainly bit-garbage."

And the caveat, in the same file: "As with the other protocol packages,
the field positions follow the most-cited public reference; some LTR-Net
variants pack fields slightly differently. Cross-check before trusting
live captures." No upstream literal pins these positions; the round-trip
tests pin them to themselves.

<figure class="lab-figure">
<svg viewBox="0 0 680 150" width="680" height="150" role="img" aria-label="The 41-bit LTR status word as a row of proportional field boxes: Sync 1, Area 5, Group 1, Channel 4, Home 5, Group ID 8, Free 5, FCS 12. A bracket beneath marks the 24 bits the CRC-7 covers — the Group F-bit, Channel, Home, Group ID and Free — and notes that Area and the top five FCS bits are outside it.">
  <rect x="16" y="30" width="14" height="30" fill="none" stroke="var(--accent)" stroke-width="1.5"/>
  <text x="23" y="24" text-anchor="middle" fill="var(--accent)" font-size="8">Sync</text>
  <rect x="30" y="30" width="70" height="30" fill="none" stroke="var(--fg-muted)"/>
  <text x="65" y="49" text-anchor="middle" fill="var(--fg-muted)" font-size="9">Area 5</text>
  <rect x="100" y="30" width="14" height="30" fill="none" stroke="currentColor"/>
  <text x="107" y="24" text-anchor="middle" fill="currentColor" font-size="8">F</text>
  <rect x="114" y="30" width="56" height="30" fill="none" stroke="currentColor"/>
  <text x="142" y="49" text-anchor="middle" fill="currentColor" font-size="9">Chan 4</text>
  <rect x="170" y="30" width="70" height="30" fill="none" stroke="currentColor"/>
  <text x="205" y="49" text-anchor="middle" fill="currentColor" font-size="9">Home 5</text>
  <rect x="240" y="30" width="112" height="30" fill="none" stroke="currentColor"/>
  <text x="296" y="49" text-anchor="middle" fill="currentColor" font-size="9">Group ID 8</text>
  <rect x="352" y="30" width="70" height="30" fill="none" stroke="currentColor"/>
  <text x="387" y="49" text-anchor="middle" fill="currentColor" font-size="9">Free 5</text>
  <rect x="422" y="30" width="168" height="30" fill="none" stroke="var(--accent)" stroke-dasharray="4 3"/>
  <text x="506" y="49" text-anchor="middle" fill="var(--accent)" font-size="9">FCS 12 (low 7 compared)</text>
  <text x="604" y="49" fill="var(--fg-muted)" font-size="8">bit 0</text>
  <line x1="100" y1="76" x2="422" y2="76" stroke="currentColor"/>
  <line x1="100" y1="70" x2="100" y2="82" stroke="currentColor"/>
  <line x1="422" y1="70" x2="422" y2="82" stroke="currentColor"/>
  <text x="261" y="94" text-anchor="middle" fill="currentColor" font-size="9">24 message bits under the CRC-7 (sdrtrunk CRCLTR.java order): F-bit as "Area 1" · Channel 5 · Home 5 · Group 8 · Free 5</text>
  <text x="65" y="94" text-anchor="middle" fill="var(--fg-muted)" font-size="8">not covered</text>
  <text x="340" y="122" text-anchor="middle" fill="var(--fg-muted)" font-size="9">crc_ltr.go: sdrtrunk's 1-bit Area vs GopherTrunk's 5-bit Area and 4-bit Channel — "reconciling those layouts … is the documented follow-up"</text>
  <text x="340" y="140" text-anchor="middle" fill="currentColor" font-size="9" font-weight="bold">status.go: "the field positions follow the most-cited public reference … cross-check before trusting live captures"</text>
</svg>
<figcaption>The 41-bit word and the 24 bits its CRC-7 protects; the 5-bit Area sits outside the check, and the package's two layouts disagree on where an area bit lives.</figcaption>
</figure>

## The sub-audible receiver and the Manchester modes

The receiver header explains the one unusual stage: the status word is
"sub-audible (the bulk of the signal energy sits below 300 Hz), so the
receiver extracts it via FM demod + a narrow low-pass filter, then runs
symbol-rate clock recovery + 2-level slicing." The low-pass is a Kaiser
FIR of `LPFLen` = 101 taps at `LPFCutoffHz` = 300 with `LPFBeta` = 8.6 —
"a transition width of ~430 Hz … ~85 dB stopband" at 48 kHz, rejecting
the voice above the cutoff. `SymbolRate` is 300, so at 48 kHz the
Mueller-Müller loop runs at 160 samples per symbol; `New` panics below
600 Hz. The receiver test synthesises IQ whose FM output is ±1 NRZ at
300 baud with 2 kHz deviation (`makeLTRFMIQ`); the daemon test uses
`demod.ModulateSubAudibleNRZ` at amplitude 0.05.

Framing then has a pre-step the other packages lack. "Some LTR variants
transmit the sub-audible status word in bi-phase / Manchester encoding
(each bit doubled, requiring a mid-bit transition); others ship raw
NRZ", so `Process` switches on `manchesterMode`:

```go
// internal/radio/ltr/process.go (shape)
switch c.manchesterMode {
case ManchesterStrict:
    decoded, err := framing.ManchesterDecode(bits) // 01→0, 10→1; 00/11 ⇒ ErrManchesterInvalid
    if err != nil { p.aligned = false }           // drop alignment, re-anchor
    bits = decoded
case ManchesterSoft:
    decoded, _ := framing.ManchesterDecodeMajority(bits) // tie → first sample, count invalid
    bits = decoded
}
```

`ManchesterOff` is the package's zero value "matches the in-package
synthesized fixtures"; `ParseManchesterMode("")` returns
`ManchesterSoft`, which the comment calls "the dominant on-air encoding
for sub-audible LTR signaling"; `"strict"` selects the pair-validating
decoder and `"off"` or `"nrz"` the raw path.
`TestProcessManchesterStrictDecodesEncodedStream` encodes a word with a
local Manchester helper and expects a lock through the strict path.
After the pre-step, alignment is the sync-bit search from
[Part 2]({{ '/blog/deep-dives/legacy-family-02-the-shared-skeleton/' | relative_url }}):
find a `1` with 41 bits of room, commit, and unlock when a later
window's first bit is `0`. The adapter's own doc is frank about the
noise floor that leaves: "false alignment leads to spurious Ingest
calls that the state machine silently drops, and a small fraction of
correctly-aligned frames drive cc.locked + grant publication."

## The FCS

`Status.FCS` is a 12-bit field, but the check GopherTrunk runs is a
7-bit one. `fcs.go` follows "sdrtrunk's CRCLTR.java layout": a 24-bit
message vector whose index 0 is sdrtrunk's 1-bit "Area" — mapped from
GopherTrunk's `Group` F-bit — then `Channel` as 5 bits MSB-first, `Home`
5, `GroupID` 8 and `Free` 5. `framing.CRC7LTR` XORs a 24-entry syndrome
table, `crc7LTRChecksums`, "copied from sdrtrunk's CRCLTR.java" with
polynomial 0xFD and initial fill 0, and `verifyStatusFCS` compares the
result to `FCS & 0x7F`. Under `FCSOn` — the production default from
`ParseFCSMode("")` — `Ingest` drops a word whose low 7 FCS bits
disagree; `FCSOff` treats the field as opaque.

The framing package's header documents a tension the series should not
smooth over: "sdrtrunk's 'Area (1 bit)' reading is one of several LTR
Standard interpretations in circulation. GopherTrunk's existing 41-bit
Status struct models Area as 5 bits and Channel as 4 bits (different
convention); reconciling those layouts before wiring this primitive into
the LTR adapter is the documented follow-up." The wiring happened — the
F-bit stands in for sdrtrunk's area bit, and a 4-bit `Channel` is
written into a 5-bit slot with its top bit always zero — and the
reconciliation did not. The 5-bit `Area` the multi-system filter uses
is outside the check entirely.

The tests pin what they can. `TestCRC7LTRSingleBitMatchesTable` checks
the function against its own table; `TestFCSOnAcceptsValidChecksum`,
`TestFCSOnDropsCorruptedChecksum` and `TestFCSOnDropsCorruptedMessage`
check a computed FCS passes and a flipped FCS or `Channel` bit fails;
`TestFCSOffIgnoresChecksum` checks the opt-out. The table is
reference-pinned by copy; the message layout it is applied to is not.

## Following a call

`Ingest` runs its checks in a fixed order: strict validation if enabled,
the FCS under `FCSOn`, `Sync`, then the area filter — a non-zero
`Options.Area` restricts the channel to one area
(`TestControlChannelFiltersByArea`: area 7 dropped, area 5 accepted). A
word that passes reaches `maybeLock`, which publishes `LockState{FrequencyHz,
Area: s.Area, Repeater: s.Home}` on the first or a changed identity and
logs `ltr cc locked`. Then:

```go
// internal/radio/ltr/control.go — Ingest (shape)
if !s.IsActive() { c.activeGroup = 0; return } // idle word ends the call's dedup
if c.activeGroup == s.GroupID { return }       // same call, repeated word
c.activeGroup = s.GroupID
c.publishGrant(s)
```

`TestControlChannelDoesNotRepublishSameGroup` sends six identical active
words and expects one grant, then a different group and expects another.
`publishGrant` emits `trunking.Grant{Protocol: "ltr", GroupID,
ChannelNum: s.Channel, FrequencyHz}` with the frequency from the
`Resolver` — `TestControlChannelPublishesGrant` wires a
`LinearBandPlan{461_000_000, 12_500, Offset: -1}` and expects channel 3
at 461.025 MHz. There is no source radio: an LTR status word carries
none, so `SourceID` stays zero.

In production the resolver is absent. `newLTRPipeline` constructs
`ltr.New(ltr.Options{Bus, Log, SystemName, FrequencyHz})` with neither
`Resolver` nor `Area` — `config.go` has no LTR band-plan or area key —
so the grant publishes with `FrequencyHz: 0`, as
`TestControlChannelGrantWithoutResolverHasZeroFreq` pins, and
`Engine.HandleGrant` drops it with `dropping grant with zero frequency`.
A live LTR repeater therefore locks, announces calls, and is not
followed. `MarkLost` clears both the lock and `activeGroup`.

## The rung

What the tests cover, and which path: `ltr_test.go` and `process_test.go`
exercise the state machine and the NRZ alignment path; `fcs_test.go` the
CRC under `FCSOn`; `process_test.go` the strict Manchester decoder on a
locally encoded stream; `strict_test.go` the `IsWellFormed` filter. The
daemon integration test `TestDaemonCCDecodesLTR` boots the full chain on
synthesised sub-audible NRZ — and sets `LTRManchesterMode: "off"` and
`LTRFCSMode: "off"`, with `Status.FCS` left at zero in
`buildLTRStatusStream`; the siglab fixture carries the same two knobs.
So the one end-to-end test runs the opt-out configuration; the
production defaults, `ManchesterSoft` and `FCSOn`, are exercised only at
the unit level, and `ManchesterSoft` has parse tests but no decode test
of its own.

The rung, then. The CRC-7 table is **reference-pinned** by copy from
sdrtrunk. The 41-bit layout is what `status.go` calls "the most-cited
public reference", pinned by round-trips, with a cross-check caveat and
an unreconciled disagreement with the FCS layout — closer to the
**placeholder** rung than the SmartNet constants that cite OP25 line by
line. `docs/decoder-capture-needs.md` lists LTR among the control chains
that "ship; FEC is on by default with no outstanding capture";
`samples/README.md` notes audio captures "still work for … sub-audible
LTR Manchester". A sub-audible capture with a known area, home repeater
and group, replayed with the defaults on, would settle the layout, the
Manchester default and the FCS mapping in one pass. None exists.

## Where this goes next

LTR has no control channel; MPT 1327 has one, but puts it in the audio
band as 1200-baud FFSK tones, so a demodulated recording is as good as
IQ.
[Part 6]({{ '/blog/deep-dives/legacy-family-06-mpt1327/' | relative_url }})
reads `internal/radio/mpt1327` — the 64-bit codeword and its BCH, the
16-bit Codeword Synchronisation Code matched at a Hamming tolerance of
2 and the false-lock arithmetic behind that default, the `minConfirm`
gate, and the committed MP3 captures that make it the one FM-era
protocol pinned to real air.

## FAQ

**How does GopherTrunk decode LTR without a control channel?**
It binds one `ltr.ControlChannel` to one repeater frequency. The
receiver low-passes the FM output at 300 Hz and recovers the 300-baud
sub-audible bits; `Process` Manchester-decodes (soft by default) and
aligns 41-bit `Status` words on their sync bit; `Ingest` checks the
CRC-7, publishes `cc.locked` on the first valid word and a grant on
`IsActive()`.

**What is in an LTR status word?**
41 bits: a sync bit, 5-bit area, the group F-bit, 4-bit channel, 5-bit
home repeater, 8-bit group ID, 5-bit free-repeater hint and a 12-bit
FCS (`ltr.Status` in `status.go`). A call is active when the F-bit is
set and the group is non-zero. The positions follow "the most-cited
public reference" and the code asks for a cross-check on live captures.

**Which bits does the LTR FCS check cover?**
A 24-bit message per sdrtrunk's `CRCLTR.java`: the F-bit (as sdrtrunk's
1-bit area), channel (5), home (5), group (8) and free (5), under a CRC-7
with polynomial 0xFD compared against the low 7 bits of the 12-bit
field. The 5-bit `Area` is outside it, and `crc_ltr.go` records the two
layouts as not yet reconciled.

**Why are LTR calls not followed in GopherTrunk?**
`newLTRPipeline` wires no `Resolver`, and no LTR band-plan config key
exists, so every grant publishes with `FrequencyHz: 0` and
`Engine.HandleGrant` drops it with `dropping grant with zero frequency`.
The lock and grant decode work; the channel-to-frequency map is the
missing piece, the same gap EDACS has.

## Series navigation

**Part 5 of 14** · ←
[Part 4: EDACS — 9600-Baud Control Channel Words, BCH, and the LCN Map]({{ '/blog/deep-dives/legacy-family-04-edacs/' | relative_url }})
· Next →
[Part 6: MPT 1327 — FFSK Codewords, CWSC Tolerance and BCH(64,48)]({{ '/blog/deep-dives/legacy-family-06-mpt1327/' | relative_url }})
