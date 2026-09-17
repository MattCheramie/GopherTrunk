---
title: "DMR End to End, Part 14: The DMR Playbook"
description: "The finale — the whole DMR stack folded into one layer map from antenna to WAV: which symptom points at which layer, the instrument that confirms it, the failure signatures operators actually report, the twin ledger of everything DMR carries two of, and the honest list of what is still open."
category: deep-dives
keywords: dmr troubleshooting guide, dmr decoder playbook, dmr never syncs, dmr false call ended, dmr missed re-key, dmr deaf wideband tap, dmr layer map, dmr twin paths, trunking scanner debugging, gophertrunk dmr
tags: [dmr-end-to-end, dmr, playbook, troubleshooting, methodology, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "DMR End to End"
series_part: 14
---

*Part 14 — the last — of **DMR End to End**, a 14-part deep dive that
follows the world's most widely deployed digital PMR protocol through
GopherTrunk — from a 4FSK carrier to two simultaneous recorded calls,
direct-mode handhelds, and decrypted Enhanced Privacy voice. Thirteen parts
ago we started with four deviations on a 4FSK ladder; since then we have
climbed through sync and polarity, two slots and two cadences, the forged
terminator, link control, IPSC, idle beacons, Tier III, direct mode,
wideband, AMBE+2, RC4 and the testing discipline that holds it together.
This closing part compresses the climb into what you need at the bench: one
layer map from symptom to part, the failure signatures operators report, the
ledger of everything DMR carries two of, and an honest list of what remains
open.*

> **TL;DR:** Debugging DMR in GopherTrunk is deciding **which layer** a
> symptom lives in, then reaching for that layer's instrument. Never syncs →
> the 4FSK front end and, on a handheld, the carrier gate (Parts 1, 9).
> Syncs but no grants → cadence and link control (Parts 3, 5). One call for
> two talkgroups → the two-slot machinery (Part 6). "Call ended"
> mid-sentence → the forged terminator (Part 4). Missed replies → re-key
> rules (Part 6). A tap deaf for minutes at steady power → the bin edge and
> the deaf heal (Part 10). Frames that look scrambled → the Golay histogram
> (Parts 12, 13). Silent recordings on an encrypted call → the PI header
> line (Part 12). The running thread is the
> **twin pair**: one carrier carries two of everything — TS1/TS2, data/voice
> polarity, 132/288 cadence, Tier II/III, DDC/channelizer, live/replay — and
> half the confusing reports were one twin behaving differently from its
> sibling.

**Key takeaways**

- **Locate the layer before touching anything.** Every layer has a distinct
  counter in the activity line; the costliest mistake in this series was
  chasing ppm and gain on a receiver blinded by inter-burst gaps.
- **Ask "which twin am I on?" early.** Tier II never stamped the flag Tier
  III did; the harness sliced at 132 while production sliced at 288; the
  channelizer was deaf where the DDC heard.
- **Counters beat absolute levels.** `dibits`, `sync_hits`, `beacons`,
  `late_entries`, `rekeys` and `deaf_heals` diagnose; a dBFS figure never
  does — the deaf tap sat at −51 dBFS throughout.
- **The open items are gated on evidence, not effort.** Two live runs, one
  proprietary CSBK train and one unnamed emitter each wait for a capture
  this playbook names.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| "Never syncs" triage | 4FSK demod, then the direct-mode gate, then RF | [Part 1]({{ '/blog/deep-dives/dmr-end-to-end-01-4fsk-carrier/' | relative_url }}), [Part 9]({{ '/blog/deep-dives/dmr-end-to-end-09-direct-mode-carrier-gate/' | relative_url }}) |
| "Syncs, no grants" triage | cadence → late entry → Tier III band plan | [Part 3]({{ '/blog/deep-dives/dmr-end-to-end-03-two-slots-one-carrier/' | relative_url }}), [Part 5]({{ '/blog/deep-dives/dmr-end-to-end-05-link-control-late-entry/' | relative_url }}), [Part 8]({{ '/blog/deep-dives/dmr-end-to-end-08-tier3-trunking/' | relative_url }}) |
| Two slots, one carrier | synthetic Timeslot 1/2, `slotRouter`, re-key rules | [Part 6]({{ '/blog/deep-dives/dmr-end-to-end-06-conventional-ipsc-two-calls/' | relative_url }}) |
| Deaf wideband tap | bin-edge flatness, `healDeafTier2`, coarse-offset reject | [Part 10]({{ '/blog/deep-dives/dmr-end-to-end-10-wideband-bin-edge-deaf-heal/' | relative_url }}) |
| Scrambled-looking frames | Golay C0/C1 histogram before cipher theory | [Part 12]({{ '/blog/deep-dives/dmr-end-to-end-12-enhanced-privacy-rc4/' | relative_url }}), [Part 13]({{ '/blog/deep-dives/dmr-end-to-end-13-testing-dmr/' | relative_url }}) |
| Reading the activity line | field by field | [Field Notebook Part 5]({{ '/blog/tutorials/field-notebook-05-dmr-activity-line/' | relative_url }}) |

## In this post

- **The layer map** — the whole stack, one table, symptom → instrument → part.
- **Failure signatures** — the reports operators actually file, decoded.
- **The twin ledger** — every pair where a fix touched one sibling, with receipts.
- **What's still open** — the honest punch list, each item at a named gate.

## The layer map

The series in one table: find the row whose symptom matches, read its
instrument, and only then open the code.

| Layer | Symptom when broken | Instrument | Part |
|---|---|---|---|
| 4FSK demod | never syncs; `dibits` climbing, `sync_hits=0` | eye diagram, `capture`'s verdicts | [1]({{ '/blog/deep-dives/dmr-end-to-end-01-4fsk-carrier/' | relative_url }}) |
| Sync + polarity | data bursts read as voice | `SyncIsDataAtPolarity` | [2]({{ '/blog/deep-dives/dmr-end-to-end-02-bursts-sync-polarity/' | relative_url }}) |
| Slot cadence | bogus `ambe_ok`, `lc_superframes=0` on a handheld | the harness's MS-sourced note | [3]({{ '/blog/deep-dives/dmr-end-to-end-03-two-slots-one-carrier/' | relative_url }}) |
| FEC + slot type | "call ended" mid-sentence | `terminator slot type without a BPTC-valid payload ignored` | [4]({{ '/blog/deep-dives/dmr-end-to-end-04-fec-stack-forged-terminator/' | relative_url }}) |
| Link control | keyups missed on a weak tap | `late_entries` | [5]({{ '/blog/deep-dives/dmr-end-to-end-05-link-control-late-entry/' | relative_url }}) |
| IPSC two-slot | one call for two TGs, missed replies | `rekeys`, `dmr_interleaved_voice` | [6]({{ '/blog/deep-dives/dmr-end-to-end-06-conventional-ipsc-two-calls/' | relative_url }}) |
| Idle + CSBK | `beacons=0` on a keyed repeater; CRC-fail storms | `site alive (idle beacon)`, `info_hex` | [7]({{ '/blog/deep-dives/dmr-end-to-end-07-idle-beacon-csbk/' | relative_url }}) |
| Tier III | locks on C_ALOHA, grants tune to static | LCN → Hz by hand from `dmr_band_plan` | [8]({{ '/blog/deep-dives/dmr-end-to-end-08-tier3-trunking/' | relative_url }}) |
| Direct mode | a handheld never decodes | `NoCarrierGate` A/B, timing acquisition | [9]({{ '/blog/deep-dives/dmr-end-to-end-09-direct-mode-carrier-gate/' | relative_url }}) |
| Wideband | one tap deaf for minutes at steady power | `deaf_heals`, the heal WARN's `coarse_offset_hz` | [10]({{ '/blog/deep-dives/dmr-end-to-end-10-wideband-bin-edge-deaf-heal/' | relative_url }}) |
| AMBE+2 voice | "computer voice" after every pause | band fractions on voice frames only | [11]({{ '/blog/deep-dives/dmr-end-to-end-11-ambe2-silence-frames/' | relative_url }}) |
| Enhanced Privacy | encrypted call records silence | the PI header INFO line, `key_configured` | [12]({{ '/blog/deep-dives/dmr-end-to-end-12-enhanced-privacy-rc4/' | relative_url }}) |
| Testing | green suite, red air | replay harnesses + scrubs, Golay histogram | [13]({{ '/blog/deep-dives/dmr-end-to-end-13-testing-dmr/' | relative_url }}) |

<figure class="lab-figure">
<svg viewBox="0 0 680 160" width="680" height="160" role="img" aria-label="The DMR decode stack from antenna to WAV with the series part covering each block — DDC or channelizer, 4FSK demodulator with carrier gate, sync and polarity, the 132 or 288 slicer, the FEC stack, link control and Tier II or III, AMBE+2 and Enhanced Privacy — and a brace below marking the testing pyramid across everything.">
  <g fill="none" stroke="currentColor">
    <rect x="4" y="50" width="62" height="40" rx="6"/><rect x="76" y="50" width="70" height="40" rx="6"/>
    <rect x="156" y="50" width="86" height="40" rx="6"/><rect x="252" y="50" width="70" height="40" rx="6"/>
    <rect x="332" y="50" width="76" height="40" rx="6"/><rect x="418" y="50" width="60" height="40" rx="6"/>
    <rect x="488" y="50" width="96" height="40" rx="6"/><rect x="594" y="50" width="82" height="40" rx="6"/>
    <line x1="66" y1="70" x2="76" y2="70"/><line x1="146" y1="70" x2="156" y2="70"/><line x1="242" y1="70" x2="252" y2="70"/>
    <line x1="322" y1="70" x2="332" y2="70"/><line x1="408" y1="70" x2="418" y2="70"/><line x1="478" y1="70" x2="488" y2="70"/><line x1="584" y1="70" x2="594" y2="70"/>
  </g>
  <g text-anchor="middle" font-size="9" fill="currentColor">
    <text x="35" y="74">antenna</text>
    <text x="111" y="67">DDC / channelizer</text>
    <text x="199" y="67">4FSK + carrier gate</text>
    <text x="287" y="67">sync + polarity</text>
    <text x="370" y="67">slicer 132 / 288</text>
    <text x="448" y="67">FEC stack</text>
    <text x="536" y="67">LC · Tier II / III</text>
    <text x="635" y="67">AMBE+2 → EP → WAV</text>
  </g>
  <g text-anchor="middle" font-size="8" fill="var(--accent)">
    <text x="111" y="81">P10</text><text x="199" y="81">P1 · P9</text><text x="287" y="81">P2</text>
    <text x="370" y="81">P3</text><text x="448" y="81">P4</text><text x="536" y="81">P5 · P6 · P8</text><text x="635" y="81">P11 · P12</text>
  </g>
  <path d="M 4 118 L 4 130 L 676 130 L 676 118" fill="none" stroke="var(--accent)"/>
  <text x="340" y="148" text-anchor="middle" fill="var(--accent)" font-size="9">testing: literal vectors → honest synthetics → committed slices → replay harnesses → on-air — P13</text>
</svg>
<figcaption>Antenna to WAV with part numbers attached, and the testing pyramid spanning every block.</figcaption>
</figure>

## Failure signatures

The layer map is for methodical triage; these are the shortcuts — the
reports operators filed during this series, matched to their cause.

**"It never syncs."** On a *handheld*, inter-burst noise blinded the
receiver throughout
[#836](https://github.com/MattCheramie/GopherTrunk/issues/836) — with
`NoCarrierGate` the production receiver yields **zero** sync words from the
reporter's 446.500 MHz slices
([Part 9]({{ '/blog/deep-dives/dmr-end-to-end-09-direct-mode-carrier-gate/' | relative_url }})).
On a *repeater*, the wideband hint `strong in-channel signal but no sync`
names an uncorrected tuner offset, and its overload variant says gain comes
first — though the #836 captures decode at 36–40 % of samples on the rail.

**"It syncs but never grants."** Ask which cadence the voice decoder slices
at: a handheld's bursts are 288 dibits apart, and the single-slot 132 decoder
slices the gaps — bogus `ambe_ok`, `lc_superframes=0`, no late entry
([Part 3]({{ '/blog/deep-dives/dmr-end-to-end-03-two-slots-one-carrier/' | relative_url }})).
If headers are lost on a weak tap, late entry grants from two agreeing
CRC-valid embedded LCs ~720 ms in — check `late_entries`. On Tier III, work
one LCN through `dmr_band_plan`
([Part 8]({{ '/blog/deep-dives/dmr-end-to-end-08-tier3-trunking/' | relative_url }})).

**"One call for two talkgroups, and the audio is DJ-scratchy."**
`dmr_interleaved_voice` was computed by the daemon but Tier II's `Options`
had no field for it, so both slots' AMBE frames were sliced into one
superframe. Fixed with a per-destination map assigning each call a
**synthetic** Timeslot 1/2 and `dmrSameCarrierTaps = 2`
([Part 6]({{ '/blog/deep-dives/dmr-end-to-end-06-conventional-ipsc-two-calls/' | relative_url }})).
With no embedded LC, both `slotRouter`s fall back to phase parity.

**"GT said the call ended, but the conversation continued."** A forged
terminator: Golay(20,8) decodes roughly a third of arbitrary words to *some*
codeword, so about one voice burst A in twelve read as Terminator-with-LC.
Slot types now come only from data-sync bursts, and an undecodable
terminator needs a BPTC-valid payload
([Part 4]({{ '/blog/deep-dives/dmr-end-to-end-04-fec-stack-forged-terminator/' | relative_url }})).

**"Calls completely missed while the radios decode."** A re-key within
hangtime: the slicer decoded the terminator 0.8–6.5 s after the composer, so
the reply's header was deduped as a repeat. A header more than
`headerRekeyDibits` (0.25 s) past the call's anchor re-grants; a fresh
embedded LC after `superframeRekeyGapDibits` (2 s) re-grants by late entry —
watch `rekeys`. Its mirror: ten header copies over 0.6 s are **one** keyup,
anchored on the last copy
([Part 5]({{ '/blog/deep-dives/dmr-end-to-end-05-link-control-late-entry/' | relative_url }})).

**"A wideband tap goes deaf for minutes at normal power."** Two mechanisms
([Part 10]({{ '/blog/deep-dives/dmr-end-to-end-10-wideband-bin-edge-deaf-heal/' | relative_url }})):
the channelizer bin was −6 dB *at* its edge until `channelizer.Oversampled`,
and the coarse acquirer froze on a −20.1 kHz neighbour in every idle gap.
`healDeafTier2` resets a tap that sees no sync for three windows within 6 dB
of the level it **last synced at** — never an absolute dBFS — and rejects an
engage the channel never synced under; `deaf_heals` counts.

**"The frames look scrambled."** Count Golay corrections before any cipher
theory: random words need exactly three ~73 % of the time, the shape the
#1187 bursts showed when a simplex radio was sliced at 132
([Part 13]({{ '/blog/deep-dives/dmr-end-to-end-13-testing-dmr/' | relative_url }})).

**"An encrypted call records silence."** Read `composer: dmr PI header —
call is encrypted`: `algorithm`, `key_id`, `key_configured`. A key under a
different `key_id` is never applied; Hytera, DES and AES are identified, not
decrypted
([Part 12]({{ '/blog/deep-dives/dmr-end-to-end-12-enhanced-privacy-rc4/' | relative_url }})).

## The twin ledger

The series' running thread, settled as a checklist. Every entry is a real
divergence with a receipt — ask which side you are on before concluding
anything:

| Twin pair | The drift, when it happened |
|---|---|
| TS1 ↔ TS2 | scalar call state folded both slots into one call; now a per-destination map with synthetic Timeslots (Part 6) |
| data ↔ voice polarity | the sync words are each other's `PolarityFlip` image, so "is this a data burst?" is per polarity — `SyncIsDataAtPolarity` (Parts 2, 4) |
| 132 ↔ 288 cadence | the EP harness sliced a handheld at 132 the day after production moved to cadence detection (#1192) (Parts 3, 13) |
| Tier II ↔ Tier III | Tier III stamped `Grant.DMRInterleavedVoice`, Tier II never did, so `dmr_interleaved_voice` was dropped on the floor (Part 6) |
| DDC ↔ channelizer | polyphase 1991 beacons / 2 grants vs DDC 6103 / 7 over the same 300 s until the bank was oversampled (Part 10) |
| slicer ↔ composer | the composer saw the release 0.8–6.5 s before the tier2 slicer, hence the re-key rule (Part 6) |
| live ↔ replay | a harness that lags production produces a symptom production no longer has (Part 13) |

The lesson has
[its own postmortem]({{ '/blog/solution-postmortem/from-the-issue-tracker-22-two-pipelines/' | relative_url }}):
every fix needs an explicit answer to "and the twin?" — in the PR, not in
the next bug report.

### How the twin thread shaped the Go code

- **One flag, threaded through both tiers.** `InterleavedVoice` rides
  `tier2.Options` and both constructors (`ccdecoder/pipelines.go`,
  `widebandt2/engine.go`).
- **Per-polarity predicates, not a global sync table.**
  `SyncIsDataAtPolarity` makes the data/voice question exact.
- **Cadence is a default, not a config key.** `DMRVoiceCadenceDetected` is
  true for every DMR protocol.
- **Harnesses call the production constructors.** `TestDMRIPSCReplay` builds
  `tier2.New` as the daemon does.

## What's still open

An honest playbook ends with its open items, each at a named gate, in the
repo's own verified-language:

- **Direct mode on a live build** — verified on the reporter's 15 Sep
  captures; the **live daemon run** on a fixed build is the gate on #836.
- **Enhanced Privacy through the daemon** — capture-verified (b0 continuity
  0.77 / 0.74); a **live call** with `encryption_keys` recording intelligible
  audio closes
  [#1187](https://github.com/MattCheramie/GopherTrunk/issues/1187).
- **The cc=7 CSBK-CRC-fail train** — a 30 ms-cadence train of BPTC-clean,
  CRC-failing CSBKs on the 443.2375 MHz repeater of a cc=12 system; real,
  not noise. Parked with `info_hex`; a capture of that frequency idle pins it.
- **The −20 kHz emitter** — ≈442.3675 MHz, dominating the Fire2 tap only in
  the repeater's idle gaps. A ±100 kHz capture centred on 442.3875 MHz over
  one idle cycle names it; do **not** add a frequency bound to the acquirer.
- **Two-slot audio on air** — synthetic-verified; the A/B of two
  simultaneously recorded talkgroups still needs a decodable capture.
- **`srcList: []`** — the per-call JSON shipped an empty talker list where
  trunk-recorder listed two radios; not investigated.

Every item is blocked on evidence, not effort — the sentence the
[P25 playbook]({{ '/blog/deep-dives/p25-end-to-end-14-playbook/' | relative_url }})
closed on, because it is the project's operating principle.

## Where to go from here

For the same protocol-length treatment of the other trunking families, cross
to [TETRA End to End]({{ '/blog/series/tetra-end-to-end/' | relative_url }})
and [P25 End to End]({{ '/blog/series/p25-end-to-end/' | relative_url }}).
Two series launch alongside this one:
[Beyond Voice]({{ '/blog/series/beyond-voice/' | relative_url }}) follows
everything GopherTrunk decodes that is not a trunked call through one
eleven-place wiring pattern, and
[The Field Notebook]({{ '/blog/series/field-notebook/' | relative_url }})
reads `debug.log` one line family at a time — its
[Part 5]({{ '/blog/tutorials/field-notebook-05-dmr-activity-line/' | relative_url }})
is the DMR activity line. The Cookbook's
[Tier III]({{ '/blog/tutorials/operator-cookbook-02-dmr-tier3/' | relative_url }})
and [two-slot conventional]({{ '/blog/tutorials/operator-cookbook-03-conventional-dmr-two-slots/' | relative_url }})
recipes run every layer mapped here; the
[DMR reference]({{ '/reference/dmr/' | relative_url }}) is the glossary.

## FAQ

**My DMR system won't decode — where do I start?**
At the bottom of the layer map, with counters: is `dibits` climbing while
`sync_hits` stays at zero? Is the radio a handheld or a repeater? Does the
wideband hint name a tuner offset or an overloaded front end? A "no" at any
row names your layer.

**Why does GopherTrunk grant one call but I hear two conversations?**
The conventional path once folded a repeater's two timeslots into one call
because Tier II never stamped the interleaved-voice flag. Since the two-slot
fix each concurrent talkgroup gets its own synthetic Timeslot and
same-carrier tap; the two-slot A/B on a decodable capture is still pending.

**What does a burst of "CSBK CRC mismatch" lines mean?**
Usually between-beacon noise, parked to one line per 10 s with a
`suppressed_repeats` count. A sustained 30 ms-cadence train at a colour code
your system does not use (cc=7 on a cc=12 repeater) is a real proprietary
train that is still unpinned — its `info_hex` is the evidence to file.

**Is DMR direct mode supported?**
Yes, and verified on the reporter's 15 Sep captures: the carrier gate,
feed-forward timing acquisition and header-train anchoring let a simplex
handheld decode from its first burst. Still open is the reporter's live
daemon run on a build with those fixes (#764/#771).

**Does this playbook transfer to TETRA or P25?**
The method does — locate the layer, read its counter, ask which twin you are
on. The rows differ: P25's twins are C4FM/CQPSK and Phase 1/2, TETRA's are
slot-grid and scramble-seed questions. Their playbooks close the
[TETRA]({{ '/blog/series/tetra-end-to-end/' | relative_url }}) and
[P25]({{ '/blog/series/p25-end-to-end/' | relative_url }}) series.

## Series navigation

**Part 14 of 14** · ←
[Part 13: Testing DMR Without a Repeater]({{ '/blog/deep-dives/dmr-end-to-end-13-testing-dmr/' | relative_url }})
· [Back to the series index]({{ '/blog/series/dmr-end-to-end/' | relative_url }})
