---
title: "DMR End to End, Part 3: Two Slots, One Carrier — Repeater vs Simplex Cadence"
description: How GopherTrunk follows a DMR voice superframe across six bursts when only the first carries a sync — the 132, 264 and 288-dibit cadences, embedded-LC and AMBE-FEC cadence detection, the relative Phase label separating two interleaved calls, and the simplex handheld whose empty slot a single-slot decoder sliced as voice.
category: deep-dives
keywords: dmr voice superframe, dmr tdma cadence, dmr two slot interleaved decoder, dmr 288 dibit cadence, dmr cach spacing, dmr direct mode voice decode, dmr embedded lc cadence lock, dmr slot phase routing, gophertrunk dmr
tags: [dmr-end-to-end, dmr, tdma, voice, cadence, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "DMR End to End"
series_part: 3
---

*Part 3 of **DMR End to End**, a 14-part deep dive that follows the world's
most widely deployed digital PMR protocol through GopherTrunk — from a 4FSK
carrier to two simultaneous recorded calls, direct-mode handhelds, and
decrypted Enhanced Privacy voice.
[Part 2]({{ '/blog/deep-dives/dmr-end-to-end-02-bursts-sync-polarity/' | relative_url }})
sliced bursts from sync matches and left one asymmetry hanging: only burst A
of a voice superframe carries a sync at all. This part is about the rhythm
that locates the other five — and the thread's second twin. A repeater
interleaves two timeslots so one call's bursts sit 264 or 288 dibits apart;
a simplex handheld transmits alone at the same cadence with silence where
the other slot would be. A decoder that assumed back-to-back bursts sliced
that silence as speech, and its counters looked healthy.*

> **TL;DR:** DMR voice is a **superframe of six bursts A–F**, 360 ms, 18
> AMBE+2 frames (`internal/radio/dmr/voice/superframe.go`). Only A carries a
> voice sync; B–F are found by **cadence** — the same-slot stride between a
> call's bursts. `NewDecoder` assumes 132 (back-to-back);
> `NewInterleavedDecoder` auto-detects **264** (no CACH) or **288** (12-dibit
> CACH before each burst). Detection is authoritative from a CRC-valid
> embedded LC (`lockedByLC`) and *provisional* from AMBE Golay(23,12)
> corrected-bit scores (`ambeErrorScore`, ceiling 24, margin 6) — a wrong
> guess can be overridden later (reopened #644). Each superframe carries a
> relative `Phase` the composer's `slotRouter` binds to a talkgroup via the
> embedded LC. A direct-mode handheld is one burst per 60 ms frame — **also
> 288 dibits** — so `trunking.DMRVoiceCadenceDetected` now defaults every
> DMR protocol to the cadence-detecting decoder; the single-slot decoder
> sliced the gaps, reporting bogus `ambe_ok` and `lc_superframes=0` on the
> #836 captures.

**Key takeaways**

- **Five of six voice bursts have no landmark.** Burst A's sync anchors the
  superframe; B–F are cut at a fixed stride, so a wrong stride splices the
  other timeslot — or the gap — into every AMBE frame.
- **Cadence is detected, not assumed.** A CRC-valid embedded LC locks it;
  absent one, the AMBE FEC score picks a provisional winner a later LC can
  still overturn — the fix for a call that "sounded encrypted" throughout.
- **Phase is relative.** `(start / (step/2)) mod 2` tells two interleaved
  calls apart; it is not TS1/TS2, because both slots share the BS sync.
- **A repeater's other slot and a handheld's silence are the same stride.**
  288 dibits either way — so one decoder now serves all three tiers, and the
  132-dibit decoder's counters on a simplex capture were fiction.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| Superframe | bursts A–F, 18 AMBE frames, LC from B–E | `internal/radio/dmr/voice/superframe.go` (`VoiceSuperframe`) |
| Cadences | 132 single-slot; 264 / 288 interleaved | `superframe.go` (`NewDecoder`, `NewInterleavedDecoder`, `cachDibits`) |
| Cadence lock | LC authoritative, AMBE score provisional | `superframe.go` (`resolveAndSlice`, `lockedByLC`, `ambeErrorScore`) |
| Score gates | winner ≤ 24 corrected bits, runner-up ≥ 2× + 6 | `superframe.go` (`ambeCadenceLockCeiling`, `ambeCadenceLockMargin`) |
| Phase | relative slot label per superframe | `superframe.go` (`sliceAt`); `composer/dmr_voice.go` (`slotRouter`) |
| Per-protocol default | cadence decoder on for every DMR protocol | `internal/trunking/site.go` (`DMRVoiceCadenceDetected`) |
| Replay instrument | `GT_DMR_INTERLEAVED=1`, per-phase counts | `cmd/gophertrunk/dmr_ipsc_replay_test.go` (`TestDMRIPSCReplay`) |

## In this post

- **Six bursts, one sync** — the superframe and what each burst carries.
- **Three cadences for one call** — 132, 264, 288, and where the CACH comes from.
- **Detecting the stride** — LC first, Golay score second, provisional until proven.
- **Phase is a relative label** — separating two calls without TS1/TS2.
- **The gap that read as voice** — #836's simplex captures and the 132-dibit decoder.

## Six bursts, one sync

A DMR voice call is organised into superframes of six 132-dibit bursts,
A through F, spanning 360 ms and carrying three 72-bit AMBE+2 frames each —
18 per superframe
([superframe reference]({{ '/reference/dmr-voice-superframe/' | relative_url }})).
Burst A is framed by a voice sync word. Bursts B–F replace the sync with
embedded signalling — a 16-bit EMB around a 32-bit fragment — and B–E's four
fragments reassemble into the embedded Link Control that names the call
(Part 5). The decoder's output carries both:

```go
// internal/radio/dmr/voice/superframe.go (shape)
type VoiceSuperframe struct {
    Frames     [18][]byte // bursts A..F, three 72-bit frames each
    StartDibit int        // absolute dibit index of burst A
    Phase      uint8      // relative slot label on a 2-slot carrier
    HasLC      bool       // embedded LC from B–E passed BPTC + checksum
    LC         dmr.FLC
    /* … EMB colour code, RC, talker alias, GPS … */
}
```

The consequence Part 2 planted now bites: **only burst A produces a sync
match.** The `Decoder` locks onto it with a detector restricted to the four
voice syncs, then cuts B–F at a fixed stride and pulls `AMBEFrames` from
each. Get the stride wrong and every frame after A is cut from the wrong
dibits, and nothing in B–F will complain — nothing in them is a landmark.

## Three cadences for one call

Part 1 did the arithmetic: a 60 ms frame is 288 dibits. What sits between
two bursts of *one* call depends on who is transmitting:

| Stream | Between a call's bursts | Stride |
|---|---|---|
| synthetic back-to-back (old fixtures) | nothing | 132 |
| 2-slot carrier, no CACH | the other slot's burst | 264 |
| base-station outbound (live repeater) | CACH + other slot + CACH | 288 |
| direct-mode handheld | 156 dibits of receiver noise | 288 |

The CACH is the 24-bit Common Announcement Channel a base station inserts
before each outbound burst
([CACH reference]({{ '/reference/dmr-cach/' | relative_url }})); GopherTrunk
uses it purely as spacing, `cachDibits = 12`. The constructors encode the
table:

```go
// internal/radio/dmr/voice/superframe.go (shape)
func NewDecoder() *Decoder { return newDecoder([]int{dmr.BurstDibits}) } // 132

func NewInterleavedDecoder() *Decoder {
    return newDecoder([]int{2 * dmr.BurstDibits, 2 * (dmr.BurstDibits + cachDibits)}) // 264, 288
}
```

A single-cadence decoder locks its one stride at construction; the
interleaved decoder buffers enough for every candidate to be sliceable, then
chooses.

<figure class="lab-figure">
<svg viewBox="0 0 680 250" width="680" height="250" role="img" aria-label="Three timelines of DMR bursts: a live repeater alternating CACH, TS1 and TS2 bursts with a call's TS1 bursts 288 dibits apart; a direct-mode handheld with one burst then noise, also 288 apart; and the single-slot decoder's 132-dibit slices laid over the handheld row so bursts B onward land in the gaps.">
  <text x="10" y="22" fill="currentColor" font-size="10" font-weight="bold">repeater outbound — two slots, CACH between</text>
  <g font-size="8">
    <rect x="10" y="30" width="14" height="24" fill="none" stroke="var(--fg-muted)"/>
    <rect x="24" y="30" width="96" height="24" fill="none" stroke="var(--accent)"/><text x="72" y="46" text-anchor="middle" fill="var(--accent)">TS1 · A</text>
    <rect x="120" y="30" width="14" height="24" fill="none" stroke="var(--fg-muted)"/>
    <rect x="134" y="30" width="96" height="24" fill="none" stroke="currentColor"/><text x="182" y="46" text-anchor="middle" fill="currentColor">TS2</text>
    <rect x="230" y="30" width="14" height="24" fill="none" stroke="var(--fg-muted)"/>
    <rect x="244" y="30" width="96" height="24" fill="none" stroke="var(--accent)"/><text x="292" y="46" text-anchor="middle" fill="var(--accent)">TS1 · B</text>
    <rect x="340" y="30" width="14" height="24" fill="none" stroke="var(--fg-muted)"/>
    <rect x="354" y="30" width="96" height="24" fill="none" stroke="currentColor"/><text x="402" y="46" text-anchor="middle" fill="currentColor">TS2</text>
  </g>
  <path d="M 24 60 L 24 68 L 244 68 L 244 60" fill="none" stroke="var(--accent)"/>
  <text x="134" y="80" text-anchor="middle" fill="var(--accent)" font-size="9">same-slot stride 288 = 2 × (132 + 12)</text>
  <text x="10" y="108" fill="currentColor" font-size="10" font-weight="bold">direct-mode handheld — one slot, noise between</text>
  <g font-size="8">
    <rect x="24" y="116" width="96" height="24" fill="none" stroke="var(--accent)"/><text x="72" y="132" text-anchor="middle" fill="var(--accent)">A</text>
    <text x="182" y="132" text-anchor="middle" fill="var(--fg-muted)">156 dibits noise</text>
    <rect x="244" y="116" width="96" height="24" fill="none" stroke="var(--accent)"/><text x="292" y="132" text-anchor="middle" fill="var(--accent)">B</text>
    <text x="402" y="132" text-anchor="middle" fill="var(--fg-muted)">noise</text>
  </g>
  <path d="M 24 146 L 24 154 L 244 154 L 244 146" fill="none" stroke="var(--accent)"/>
  <text x="134" y="166" text-anchor="middle" fill="var(--accent)" font-size="9">stride 288 again — the same cadence, different filler</text>
  <text x="10" y="194" fill="currentColor" font-size="10" font-weight="bold">single-slot decoder (stride 132) over the handheld row</text>
  <g font-size="8" fill="var(--fg-muted)">
    <rect x="24" y="202" width="96" height="20" fill="none" stroke="currentColor"/><text x="72" y="216" text-anchor="middle" fill="currentColor">A ✓</text>
    <rect x="120" y="202" width="96" height="20" fill="none" stroke="var(--fg-muted)" stroke-dasharray="3 2"/><text x="168" y="216" text-anchor="middle">"B" = gap</text>
    <rect x="216" y="202" width="96" height="20" fill="none" stroke="var(--fg-muted)" stroke-dasharray="3 2"/><text x="264" y="216" text-anchor="middle">"C" = gap + real B</text>
  </g>
  <text x="340" y="242" text-anchor="middle" fill="var(--fg-muted)" font-size="9">B–F cut from silence: no embedded LC reassembles, and muted gaps Golay-decode as valid all-zero words</text>
</svg>
<figcaption>Two shapes of one cadence: a repeater's other slot and a handheld's silence both put a call's bursts 288 dibits apart — and a 132-dibit decoder slices the silence as if it were speech.</figcaption>
</figure>

## Detecting the stride

`resolveAndSlice` decides the cadence per call with two grades of evidence.
The strong one is signalling: a stride whose bursts B–E reassemble a
CRC-valid embedded LC is *right* — a wrong slice cannot reassemble one — so
it locks authoritatively (`lockedByLC = true`). The weak one is speech
quality: `ambeErrorScore` sums the Golay(23,12) corrected-bit count across
a slice's 18 AMBE frames (`DecodeAMBEFrame`, Part 11); a correct slice
needs very few corrections, a wrong one pulls bits from the other slot or
the CACH and averages ~4–5 per frame.

```go
// internal/radio/dmr/voice/superframe.go (shape)
const (
    ambeCadenceLockCeiling = 24 // winner's total corrected bits (~1.3/frame)
    ambeCadenceLockMargin  = 6  // runner-up must exceed 2×winner by this
)

func (d *Decoder) resolveAndSlice(start int, syncName string) VoiceSuperframe {
    if d.lockedStep != 0 && d.lockedByLC {
        return d.sliceAt(start, d.lockedStep, syncName) // authoritative
    }
    for i, step := range d.cadenceCandidates {
        sf := d.sliceAt(start, step, syncName)
        if sf.HasLC { d.lockedStep, d.lockedByLC = step, true; return sf }
        /* … score with ambeErrorScore, track best and runner-up … */
    }
    clearWinner := bestScore <= ambeCadenceLockCeiling &&
        bestScore*2+ambeCadenceLockMargin <= secondScore
    /* … clearWinner ⇒ provisional lock (lockedByLC stays false) … */
}
```

The provisional grade is the hard-won part. The first #644 fix chose
cadence by LC alone; on a carrier whose LC never validated it fell back to
264, which on a 288 carrier sliced every burst 24 dibits off — structured
noise that "sounded encrypted". The AMBE score fixed that. Then #644
reopened: a call that *opened* with no decodable LC could have the score
pick the wrong stride, and the old code froze that lock — and a wrong slice
never reassembles the LC that would correct it, so the rest of the call
garbled with no way back. Now an FEC lock stays **re-checkable every
superframe**: a later CRC-valid LC, or a clear FEC winner at another stride,
moves it. `TestInterleavedDecoderLCOverridesWrongProvisionalCadence` drives
that sequence — a CACH-free preamble locks 264 provisionally, then 288
traffic with a valid LC arrives and the lock must move.

## Phase is a relative label

A 2-slot carrier runs two calls at once, and the interleaved decoder emits
superframes for both. The BS-Voice sync in burst A is **identical on both
slots**, and the wire format does not label a burst's physical slot. What the decoder can compute is
parity — two slots' burst-A anchors sit one physical burst (step/2) apart,
so `sliceAt` stamps `sf.Phase = (start / (step/2)) % 2` on a multi-candidate
decoder.

`Phase` is a *relative* discriminator — stable per call, distinct between
the two concurrent calls, not an absolute TS1/TS2. Binding a phase to a
talkgroup is the embedded LC's job, done by the composer's `slotRouter`
(`internal/voice/composer/dmr_voice.go`): a superframe whose LC names this
call's destination binds its phase; one naming a *different* destination
marks that phase foreign (`foreignPhaseMask`); and if no LC decodes, after
`unboundPhaseFallbackGrace = 2` LC-less superframes the router binds the
active slot's phase so the call records rather than dropping. The same
parity becomes the synthetic `Timeslot` on a Tier II grant, the
engine-identity token Part 6 builds two concurrent calls on.

## The gap that read as voice

Now the twin in the field.
[Part 1]({{ '/blog/deep-dives/dmr-end-to-end-01-4fsk-carrier/' | relative_url }})
described a direct-mode handheld: one 132-dibit burst, then 156 dibits of
receiver noise — a *same-slot stride of 288*, indistinguishable in cadence
from a repeater's TS1 bursts with TS2 and two CACHs between them.

For the whole life of
[#836](https://github.com/MattCheramie/GopherTrunk/issues/836) the Tier I
pipeline, siglab and `replay -record-voice` ran the **single-slot** decoder;
only Tier II/III got the interleaved one. On the reporter's 15 Sep
446.500 MHz captures it anchored burst A correctly and Its counters were not just low, they were *wrong*:
`lc_superframes=0` because no LC reassembles from noise, and `ambe_ok`
**inflated**, because the gate mutes the discriminator on absent samples and
muted gaps Golay-decode as valid all-zero words. The same defect resurfaced
in the Enhanced Privacy harness a day later (Part 12) as frames that looked
scrambled *through* their FEC — hence the standing instruction: **a C0/C1
Golay histogram is the first thing to check when frames look scrambled.**

The fix is a default, not a new decoder:

```go
// internal/trunking/site.go (shape)
// On air no DMR carrier lays a call's bursts back to back — the #836
// captures decode their embedded LC in 26 of 35 superframes at the 288
// cadence and in none at 132.
func DMRVoiceCadenceDetected(p Protocol) bool {
    switch p {
    case ProtocolDMR, ProtocolDMRTier2, ProtocolDMRTier1:
        return true
    }
    return false
}
```

`resolveDMRInterleavedVoice` in `daemon.go` applies it unless the operator
forces `dmr_interleaved_voice`; `replay` and siglab build their
`trunking.System` from the same function. `TestDMRDirectModeRealAirKeyup`
asserts superframes with an LC naming talkgroup 99 at 288, and
`TestDMRIPSCReplay` warns when it sees MS-sourced syncs without
`GT_DMR_INTERLEAVED=1`. The direct-mode chain is **on-air verified on the
15 Sep captures**; the reporter's live run on a build carrying these fixes
is still open (#836).

### How the cadence twin shaped the Go code

- **One decoder, a list of candidates.** `newDecoder([]int{…})` takes the
  strides it may choose between; single-slot is the one-element case, not a
  fork.
- **Evidence has grades.** `lockedByLC` separates "proven by signalling"
  from "guessed by FEC quality", and only the second grade is re-checked —
  the reopened #644 lesson as a boolean.
- **Labels are honest about what they are.** `Phase` is documented as
  relative, `Timeslot` as synthetic; nothing pretends to know TS1 from TS2.
- **Defaults follow the air, not the fixture.** `DMRVoiceCadenceDetected`
  exists because every synthetic DMR fixture laid bursts back to back — the
  [self-consistent trap]({{ '/blog/solution-postmortem/from-the-issue-tracker-20-self-consistent-trap/' | relative_url }})
  in TDMA dress.

## Where this goes next

Cadence tells the decoder *where* a burst is; whether to *believe* it is the
FEC stack's job.
[Part 4]({{ '/blog/deep-dives/dmr-end-to-end-04-fec-stack-forged-terminator/' | relative_url }})
walks BPTC(196,96), RS(12,9) and the two CRC masks — and the voice burst
whose AMBE bits Golay-decoded to a phantom "Terminator with LC" and ended a
live call mid-sentence.

## FAQ

**What is a DMR voice superframe?**
Six consecutive bursts of one call, labelled A–F, spanning 360 ms and
carrying 18 AMBE+2 voice frames of 72 bits. Burst A carries a voice sync
word; bursts B–E carry the four fragments of the embedded Link Control that
names the talkgroup and radio. GopherTrunk decodes it in
`internal/radio/dmr/voice`.

**Why are a DMR call's bursts 264 or 288 dibits apart?**
Because a 2-slot carrier interleaves the other timeslot's burst between a
call's own: 2 × 132 = 264 dibits with no CACH, or 2 × (132 + 12) = 288 when a
base station inserts its 12-dibit CACH before each outbound burst. The
interleaved decoder auto-detects which is in use.

**How does GopherTrunk know which cadence a call uses?**
Two ways. A CRC-valid embedded Link Control reassembled from bursts B–E is
authoritative — a wrong stride cannot produce one. Without it, the decoder
scores each candidate by the Golay(23,12) corrections its AMBE frames needed
and locks the clear winner provisionally; a later valid LC can override it.

**Can GopherTrunk tell TS1 from TS2?**
Not from the air interface alone: both slots use the same BS-Voice sync and
the burst carries no slot number. Each superframe gets a relative `Phase`
from its anchor's parity, and the composer's `slotRouter` binds a phase to
the talkgroup named by the embedded LC. The grant's `Timeslot` is a
synthetic identity token.

**Why did direct-mode DMR decode no embedded LC before the #836 fixes?**
Because Tier I, siglab and replay used the single-slot decoder, which
assumes a call's bursts are back to back. A simplex handheld transmits one
burst per 60 ms frame, so bursts B–F were cut from the inter-burst gaps —
no LC could reassemble, and muted gaps Golay-decoded as valid all-zero
words, inflating `ambe_ok`.

## Series navigation

**Part 3 of 14** · ←
[Part 2: Bursts, Sync Words & Polarity]({{ '/blog/deep-dives/dmr-end-to-end-02-bursts-sync-polarity/' | relative_url }})
· Next →
[Part 4: The FEC Stack & the Forged Terminator]({{ '/blog/deep-dives/dmr-end-to-end-04-fec-stack-forged-terminator/' | relative_url }})
