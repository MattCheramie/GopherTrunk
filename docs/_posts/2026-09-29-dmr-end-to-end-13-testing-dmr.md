---
title: "DMR End to End, Part 13: Testing DMR Without a Repeater"
description: How GopherTrunk tests a DMR decoder with no repeater in reach — literal vectors from independent decoders, synthetic streams that finally model a real transmitter's gaps, committed real-air slices in CI, replay harnesses with header and terminator scrubs, and the Golay histogram that says whether frames are scrambled or mis-sliced.
category: deep-dives
keywords: testing dmr decoder, dmr replay harness, dmr real-air fixtures, synthetic dmr fixture gaps, self-consistent test trap, golay histogram dmr, GT_DMR_IQ replay, dmr direct mode test, dmr ipsc replay, gophertrunk dmr testing
tags: [dmr-end-to-end, dmr, testing, replay, fixtures, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "DMR End to End"
series_part: 13
---

*Part 13 of **DMR End to End**, a 14-part deep dive that follows the world's
most widely deployed digital PMR protocol through GopherTrunk — from a 4FSK
carrier to two simultaneous recorded calls, direct-mode handhelds, and
decrypted Enhanced Privacy voice.
[Part 12]({{ '/blog/deep-dives/dmr-end-to-end-12-enhanced-privacy-rc4/' | relative_url }})
closed with a working cipher hidden by two receiving-side defects that only
instruments exposed. This part is about those instruments: what a DMR test
can prove at each layer, the fixture that lied to every earlier test by
transmitting like a repeater, and the harnesses that let an operator's
capture argue with the code.*

> **TL;DR:** GopherTrunk tests DMR in four layers. **Literal vectors** pin
> constants against sources that are not this code — real off-air CSBKs
> prove the `0x5A5A` CRC mask, MMDVMHost's `DMR_IDLE_DATA` BPTC-decodes to
> `IdleInfoPattern`, an independent CRC pins the PI header, and
> `ep_issue1187_ptt1.json` holds on-air Enhanced Privacy frames. **Synthetic
> streams** now transmit like a handheld — one 132-dibit burst per 288-dibit
> frame with receiver noise in the gaps — because every earlier fixture laid
> bursts back-to-back like a base station and hid
> [#836](https://github.com/MattCheramie/GopherTrunk/issues/836) for its
> whole life. **Committed slices** (`dmr-*.cfile`,
> `dmr-directmode-446500-*.cs16`) replay real air in CI. **Replay harnesses**
> (`TestDMRIPSCReplay` with `GT_DMR_DROP_HEADERS` / `GT_DMR_DROP_TERMINATORS`,
> `TestDMRIPSCWidebandReplay`) turn an operator's capture into an A/B. A
> C0/C1 Golay histogram is the first instrument when frames look scrambled:
> 73 % at exactly three corrections means mis-slicing, not a cipher.

**Key takeaways**

- **A fixture that models the wrong transmitter validates the wrong
  receiver.** Back-to-back bursts hid the direct-mode blindness; a header
  framed with the voice sync passed only because the slicer parsed slot
  types on every sync match.
- **Constants need a witness that is not you.** Off-air CSBKs, an MMDVMHost
  template, a Python CRC and a reporter's frames each caught what a
  round-trip could not.
- **Scrubs turn a capture into a counterfactual.** Deleting every header or
  terminator from a real dibit stream models the fade or the missed release
  — and the harness must still grant.
- **Read the Golay histogram before any cipher theory.** Random words need
  exactly three corrections ~73 % of the time; real frames mostly need none.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| Off-air CSBK vectors | pins CRC mask `0x5A5A` + opcodes against real Tier III bursts | `tier3/csbk_realvectors_test.go` (`TestParseCSBKRealOffAirVectors`) |
| Idle-beacon constant | MMDVMHost `DMR_IDLE_DATA` → `ff83df1732094ed1e7cd8a91` | `tier2/conventional_rekey_test.go` (`TestIdleInfoPatternMatchesMMDVMHostConstant`) |
| Direct-mode fixture | 288-dibit frame grid, noise gaps, delayed RRC gating | `receiver/receiver_burst_test.go` (`directModeIQ`) |
| Forged terminator pin | a voice burst A cannot end a call | `tier2/conventional_lateentry_test.go` (`TestConventionalVoiceBurstCannotForgeTerminator`) |
| Committed real air | Tier III CC lock, FLC decode, direct-mode keyup | `cmd/gophertrunk/testdata/dmr-*.cfile`, `dmr-directmode-446500-*.cs16` |
| IPSC replay + scrubs | grants / late entries / re-keys from a capture | `cmd/gophertrunk/dmr_ipsc_replay_test.go` (`TestDMRIPSCReplay`) |
| Scrambled-or-sliced? | per-frame Golay corrections in the EP dump | `dmr_ep_replay_test.go` (`GT_DMR_EP_DUMP`, `golay_errs`) |

## In this post

- **The fixture that transmitted like a repeater** — how self-consistency hid #836.
- **Literal vectors** — constants pinned against witnesses outside the repo.
- **Honest synthetic streams** — gaps, header trains, phase sweeps, deaf taps.
- **Committed real air** — the slices CI replays every run.
- **Replay harnesses and scrubs** — the operator's capture as an argument.
- **What only air can prove** — the open items and the histogram habit.

## The fixture that transmitted like a repeater

The founding bug of this part is the one
[Part 9]({{ '/blog/deep-dives/dmr-end-to-end-09-direct-mode-carrier-gate/' | relative_url }})
fixed. A simplex handheld transmits one 27.5 ms burst per 60 ms frame — 132
dibits on, 156 off — and in the gap the discriminator of receiver noise is
uniform over ±π, several times the signal's swing. That noise inflated the
symbol AGC, decayed the AFC, halved the coarse acquirer's window mean and
random-walked the timing loop, so no sync word ever matched. Every synthetic
DMR fixture had laid its bursts **back to back** like a base station, so
the receiver was never asked the question it failed.

That is the
[self-consistent trap]({{ '/blog/solution-postmortem/from-the-issue-tracker-20-self-consistent-trap/' | relative_url }})
in its transmitter-model dress, and DMR has two more specimens. The
siglab Tier I fixture framed a Voice LC Header — a *data* burst — with the
DM **voice** sync, and passed because the old slicer parsed a slot type on
every sync match; once slot types were read only from data-sync bursts
([Part 4]({{ '/blog/deep-dives/dmr-end-to-end-04-fec-stack-forged-terminator/' | relative_url }})),
the fixture had to be corrected to `DMData1`.
And the idle beacon
([Part 7]({{ '/blog/deep-dives/dmr-end-to-end-07-idle-beacon-csbk/' | relative_url }}))
read `beacons=0` because no fixture ever contained an Idle burst —
`IngestBurst` dropped slot type 9 and nothing objected. Each time, encoder
and decoder written from one reading of the air agreed with each other and
disagreed with the radio. The layers below are the countermeasures, in
increasing power of proof.

## Literal vectors: constants with an outside witness

The cheapest defence is a test whose expected bytes came from outside this
code.

**Off-air CSBKs.** `TestParseCSBKRealOffAirVectors` holds 12-byte info
blocks from 2 MS/s captures of two live Tier III control channels (440.5625
and 440.2625 MHz) that BPTC-decode with zero corrections. Under the
previous CRC convention every one failed and the
control channel never locked; under init `0x0000` with the `0x5A5A` mask
they validate — and their opcode bytes pinned the corrected table.

**A template from another project.** `IdleInfoPattern`
(`ff83df1732094ed1e7cd8a91`) was measured on a beacon-only capture (2402 of
2416 Idle bursts). The independent witness is
`TestIdleInfoPatternMatchesMMDVMHostConstant`, which BPTC-decodes MMDVMHost's
`DMR_IDLE_DATA` template from `DMRDefines.h` and requires the same 12 bytes
with zero corrections — two codebases, one constant.

**An independent CRC.** `TestParsePIHeaderReferenceLiteral` pins the PI
header's `0x9696` mask with a Python-computed vector, because
`AssemblePIHeader`'s round-trip agrees with any mask
([Part 12]({{ '/blog/deep-dives/dmr-end-to-end-12-enhanced-privacy-rc4/' | relative_url }})).

**A reporter's frames.** `TestEPCaptureIssue1187` loads the PI header and
three on-air superframes from `voice/testdata/ep_issue1187_ptt1.json` and
asserts what no synthetic could: each embedded IV equals `AdvanceMI` of the
MI before it, and the chain yields speech while the old semantics stay at
ciphertext — the
[literal-vector rule]({{ '/blog/deep-dives/from-spec-to-shipping-03-literal-vectors/' | relative_url }})
at its strongest.

## Honest synthetic streams

Vectors test parsers; demodulators need signal that models the transmitter
the reporter owns. The direct-mode fixture in
`receiver/receiver_burst_test.go` is the template. `directModeIQ` lays bursts on a `dmFrameDibits` = 288 grid with
`dmOffDibits` = 156 of transmitter-OFF per frame, renders the OFF time as
AWGN when `gapped`, applies a carrier offset and per-axis noise, and gates on
the grid **delayed by the RRC filter's `modSpan·sps` samples**;
`dmHeaderBursts` cycles the source ID from burst to burst.

Three lessons are encoded there. **Gaps are
noise, not filler** — the point of #836. **Repetition is a bias**: one
identical burst repeated for seconds carries a symbol-mean bias the open-loop
AFC reads as drift (the #402 failure mode), so the source ID varies, and the
pipeline fixture `directModeTransmissionIQ` models a PTT as **3 headers plus
random voice bursts**, two PTTs apart, at the reporter's −1.2 kHz offset.
**Group delay is real**: gate on the undelayed grid and every burst loses its
tail.

`TestReceiverDecodesDirectModeBurstCadence` is the failing-first pin — with
`NoCarrierGate` the production receiver yields zero sync words from a 27 dB
stream — and `TestReceiverCarrierGateIsNoOpOnContinuousCarrier` keeps the
repeater path byte-identical. Around it: `timing_acq_test.go` cold-starts a
keyup at every sub-symbol phase; `conventional_headertrain_test.go` lays ten
header copies 60 ms apart and requires **one** grant;
`conventional_lateentry_test.go` forges a terminator from voice bits
(`TestConventionalVoiceBurstCannotForgeTerminator`) and parks the CSBK
failure log (100 identical failures → one line, `suppressed_repeats=99`);
`conventional_rekey_test.go` and `conventional_twoslot_test.go` cover
re-keys and two-slot calls. Above the protocol, `engine_deafheal_test.go`
stands a `resetCountingReceiver` in for the Tier II receiver and feeds
`scaledIQ` at −51 dBFS: a tap that synced and then sees three windows with
no sync at its own level is reset once; one whose carrier drops to −70 dBFS
is idle and never reset. `coarse_reject_test.go` pins that a rejected engage
reverts and never re-engages nearby.

## Committed real air

The third layer embeds the reporter's air in CI. `dmr-t3-cc.cfile` and
`dmr-voice-term.cfile` are channelized slices of a live 441 MHz Tier III
system; `TestReplayDMRTier3ControlDecodesRealAir` requires the production
receiver plus `ControlChannel` to lock on a CRC-valid Aloha, and
`TestReplayDMRVoiceLCFECDecodesRealAir` decodes Terminator-with-LC bursts
through the exact BPTC(196,96) → RS(12,9) → FLC stack the Voice LC Header
uses (TG 24 / source 4209000), the confirmation issue #527 called blocking.

The direct-mode pair is newer. `dmr-directmode-446500-keyup-48k.cs16` (a
ten-copy header train, then voice) and
`dmr-directmode-446500-ptt2-48k.cs16` (the second PTT's onset) are 48 kHz
slices of the #836 reporter's 446.500 MHz captures through the production
`ccdecoder.Downconverter`. They pin what only that air could: `TestDMRDirectModeRealAirKeyup` requires the
**ungated** receiver to find zero sync words (the gate is load-bearing) and
the gated one at least 8 MS-Data and 2 MS-Voice syncs;
`TestDMRDirectModeRealAirOnsetAtEveryPhase` cold-starts the second PTT at
ten sub-sample offsets and requires each to decode — the pre-fix receiver
took 1.3–2.8 s at three. With 36 % of raw samples at the ADC rail it decodes
anyway, which retired "clipped ⇒ undecodable" as too strong.

<figure class="lab-figure">
<svg viewBox="0 0 680 190" width="680" height="190" role="img" aria-label="The DMR replay harness: an IQ capture enters the downconverter and production receiver, header and terminator scrubs act on the dibit stream, which feeds both the Tier II state machine reporting grants and re-keys and the voice superframe decoder reporting a Golay corrections histogram.">
  <g fill="none" stroke="currentColor">
    <rect x="8" y="80" width="90" height="40" rx="6"/>
    <rect x="118" y="80" width="120" height="40" rx="6"/>
    <rect x="258" y="66" width="130" height="68" rx="6" stroke-dasharray="4 3"/>
    <rect x="410" y="30" width="130" height="50" rx="6"/>
    <rect x="410" y="120" width="130" height="50" rx="6"/>
    <line x1="98" y1="100" x2="112" y2="100"/><line x1="238" y1="100" x2="252" y2="100"/>
    <path d="M 388 100 L 400 100 L 400 55 L 404 55"/><path d="M 400 100 L 400 145 L 404 145"/>
    <line x1="540" y1="55" x2="556" y2="55"/><line x1="540" y1="145" x2="556" y2="145"/>
  </g>
  <g fill="currentColor"><polygon points="110,96 118,100 110,104"/><polygon points="250,96 258,100 250,104"/><polygon points="402,51 410,55 402,59"/><polygon points="402,141 410,145 402,149"/></g>
  <g text-anchor="middle" font-size="10" fill="currentColor" font-weight="bold">
    <text x="53" y="97">capture</text>
    <text x="178" y="97">DDC → receiver</text>
    <text x="323" y="86">scrubs</text>
    <text x="475" y="50">Tier II state machine</text>
    <text x="475" y="140">voice superframes</text>
  </g>
  <g text-anchor="middle" font-size="9" fill="var(--fg-muted)">
    <text x="53" y="111">GT_DMR_IQ</text>
    <text x="323" y="100">GT_DMR_DROP_HEADERS</text>
    <text x="323" y="114">GT_DMR_DROP_TERMINATORS</text>
    <text x="475" y="66">grants · late_entries · rekeys</text>
    <text x="475" y="156">superframes · lc · ambe_ok</text>
  </g>
  <g text-anchor="start" font-size="9" fill="var(--accent)">
    <text x="560" y="58">drop terminators: 1 → 5 grants</text>
    <text x="560" y="148">Golay histogram: 73 % at 3</text>
  </g>
</svg>
<figcaption>The replay harness: one real dibit stream, optional scrubs, two consumers — the state machine's grant/re-key ledger and the voice decoder's Golay histogram.</figcaption>
</figure>

## Replay harnesses and scrubs

Committed slices are seconds long; the reporter's problem is minutes long.
The fourth layer is the skip-gated harness that takes an operator's whole
capture: `TestDMRIPSCReplay` (`GT_DMR_IQ`; raw cs16/f32 take a rate and
format, wav and flac are content-sniffed).
It mirrors the daemon's Tier II decode — DDC, receiver, then the dibits to
**both** the conventional state machine and the voice superframe decoder —
and prints one ledger:

```
grants=5 late_entries=0 rekeys=4 fec_pass=… csbk_crc_fail=… beacons=… scrubbed_headers=0
superframes=… lc_superframes=… ambe_ok=… ambe_uncorrectable=…
```

The two consumers are the triage: zero superframes is a receiver problem;
superframes with AMBE decoding but no grants is a control-path problem. The
**scrubs** make the harness a counterfactual engine. `GT_DMR_DROP_HEADERS=1`
deletes every Voice LC Header burst from the real dibit stream — the on-air
model of a keyup lost to a fade — and late entry
([Part 5]({{ '/blog/deep-dives/dmr-end-to-end-05-link-control-late-entry/' | relative_url }}))
must still grant every transmission from its embedded LC: 5/5 with
`late_entries=5` on the 9 Sep captures. `GT_DMR_DROP_TERMINATORS=1` deletes every
Terminator-with-LC instead — the 10 Sep condition where the control path
never saw the release — and each reply must be re-granted: old code 1 grant
for 5 transmissions, new 5 with `rekeys=4`
([Part 6]({{ '/blog/deep-dives/dmr-end-to-end-06-conventional-ipsc-two-calls/' | relative_url }})).
`GT_DMR_INTERLEAVED=1` selects the cadence-detecting voice decoder — the
harness asks for it when it sees MS-sourced sync words, because the
single-slot decoder's `ambe_ok` on a simplex capture is bogus and
`lc_superframes` reads 0
([Part 3]({{ '/blog/deep-dives/dmr-end-to-end-03-two-slots-one-carrier/' | relative_url }})).

`TestDMRIPSCWidebandReplay` (`GT_DMR_WB=1`) is the
[two-pipelines]({{ '/blog/solution-postmortem/from-the-issue-tracker-22-two-pipelines/' | relative_url }})
instrument: it interpolates a 25 kS/s slice to the field bin geometry (8
bins of 6.25 MS/s ÷ 32, the tap at +687.5 kHz, 0.48 bins off centre) and
feeds a `ChannelizerBank` and a `DDCBank` from the **same** stream
(`GT_DMR_WB_CHUNK` for live-sized chunks). It reproduced the 12 Sep deafness
exactly — polyphase 1991 beacons / 2 grants versus DDC 6103 / 7 over 300 s
— and after `channelizer.Oversampled` the arms match window for window
([Part 10]({{ '/blog/deep-dives/dmr-end-to-end-10-wideband-bin-edge-deaf-heal/' | relative_url }})).
A clean synthetic carrier at residual 0.48 still *granted* through the old
bank, so the regression is a tone-flatness pin
(`TestChannelizerBankBinEdgeChannelIsFlat`), never a grant count. `TestDMREnhancedPrivacyReplay` completes the set, all following the
[capture-driven method]({{ '/blog/deep-dives/from-spec-to-shipping-11-capture-driven-development/' | relative_url }}).

## What only air can prove

The histogram habit first, because it settled #1187 in one pass. When frames
look scrambled, dump them (`GT_DMR_EP_DUMP`, `golay_errs` per frame) and
count Golay corrections per C0/C1 word before any cipher theory. Random
input needs exactly 3 corrections most of the time — the #1187 bursts B–F
showed **73 % at exactly 3, 12 % at exactly 2** while burst A was clean:
a slicer reading inter-burst gaps, not encryption. Real frames cluster at
zero.

Then the honest list — no layer here proves on-air behaviour, the #764/#771
rule this repo carries as policy. Direct mode is verified on the reporter's
15 Sep captures and the **live run on a fixed build is still open**;
Enhanced Privacy is **capture-verified, live daemon call open**; the cc=7
CSBK-CRC-fail train and the −20 kHz emitter beside the IPSC repeater are
**unresolved**, waiting for the capture that names them.

### How the fixtures shaped the Go code

- **Fixtures model the transmitter the reporter owns.** `directModeIQ` and
  `directModeTransmissionIQ` carry the frame grid, gaps, header train and
  RRC delay as named constants.
- **Every constant has a second source.** `IdleInfoPattern`, the CSBK mask
  and the PI mask each ship with a test whose expected value came from
  elsewhere.
- **Harnesses skip, never rot.** Each `GT_DMR_*` harness `t.Skip`s without
  its input, so a contributed capture slots into an existing socket.
- **Scrubs are first-class.** Header and terminator deletion live in the
  harness — a field condition is one env var away.

## Where this goes next

Thirteen parts down, one to go.
[Part 14]({{ '/blog/deep-dives/dmr-end-to-end-14-playbook/' | relative_url }})
folds the series into what you want at the bench: the layer map from antenna
to WAV, the failure signatures operators report, the twin ledger and the
open list with its gates.

## FAQ

**How do I test a DMR decoder without a repeater?**
In layers: pin wire formats with vectors from independent decoders (off-air
CSBKs, MMDVMHost templates), synthesise carriers that model a real
transmitter's gaps and header trains, replay the committed real-air slices,
and run your own capture through `TestDMRIPSCReplay` with `GT_DMR_IQ`.

**Why did synthetic DMR tests pass while direct mode never decoded?**
Because every fixture laid bursts back-to-back like a base station. A
handheld transmits one burst per 60 ms frame with receiver noise between, and
that noise wrecked the AGC, AFC and timing loops. The fixture had to model
the gaps before the failure could be observed.

**What do the header and terminator scrubs prove?**
Counterfactuals on real air. `GT_DMR_DROP_HEADERS=1` deletes every Voice LC
Header from a captured dibit stream and late entry must still grant every
transmission; `GT_DMR_DROP_TERMINATORS=1` deletes every release and each
reply must be re-granted. Both reproduce field reports from one capture.

**How can I tell scrambled frames from mis-sliced frames?**
Count Golay corrections per frame (`GT_DMR_EP_DUMP` writes `golay_errs`).
Random words need exactly three corrections about 73 % of the time and two
about 12 %; real frames need almost none. A histogram peaked at three means
the slicer is reading inter-burst gaps, not that the payload is encrypted.

**What still needs on-air confirmation for DMR?**
The reporter's live direct-mode run on a build with the #836 fixes, a live
Enhanced Privacy call through the daemon with `encryption_keys` configured,
the cc=7 CSBK-CRC-fail train, and the −20 kHz emitter near the IPSC
repeater. Each waits for a capture (#764/#771).

## Series navigation

**Part 13 of 14** · ←
[Part 12: Enhanced Privacy — RC4 & the IV That Names the Next Superframe]({{ '/blog/deep-dives/dmr-end-to-end-12-enhanced-privacy-rc4/' | relative_url }})
· Next →
[Part 14: The DMR Playbook]({{ '/blog/deep-dives/dmr-end-to-end-14-playbook/' | relative_url }})
