---
title: "P25 End to End, Part 7: Phase 2 TDMA — Two Voices per Carrier"
description: How P25 Phase 2 fits two voice calls in one 12.5 kHz channel — H-DQPSK at 6000 symbols per second, the 360 ms superframe and ISCH slot typing, MAC PDUs in place of TSBKs, and the FEC hand-off that silently defaulted to zero on one pipeline.
category: deep-dives
keywords: p25 phase 2 tdma, h-dqpsk demodulation, p25 phase 2 mac pdu, p25 superframe isch, two slot tdma 12.5 khz, p25 phase 2 decoder, pn44 scrambler p25, hybrid phase 1 phase 2 system, gophertrunk p25
tags: [p25-end-to-end, p25, phase2, tdma, mac, dsp]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "P25 End to End"
series_part: 7
---

*Part 7 of **P25 End to End**, a 14-part deep dive that follows North
America's dominant trunking protocol through GopherTrunk — from a raw C4FM
carrier to recorded, named, multi-site voice.
[Part 6]({{ '/blog/deep-dives/p25-end-to-end-06-cqpsk-lsm/' | relative_url }})
walked Phase 1's linear twin; this part is the generational twin. Phase 2
keeps the Phase 1 control channel and swaps the traffic channel for a
2-slot TDMA carrier — two simultaneous calls where Phase 1 fits one. New
symbol rate, new differential modulation, a MAC layer where TSBKs used to
be, and a fresh chapter of the running lesson: a knob that exists on one
pipeline can silently default to zero on its twin.*

> **TL;DR:** P25 Phase 2 traffic channels run **H-DQPSK at 6000 symbols/s**
> — decoded by the *same* `demod.PiOver4DQPSK` primitive TETRA uses, one
> rotation argument (π/8) apart. The stream is organised as a **360 ms
> superframe of twelve 30 ms sub-frames alternating between two
> timeslots**; a Golay(24,12)-protected **ISCH** field names each
> sub-frame's SlotType (4V/2V voice or MAC). Signalling arrives as 18-byte
> **MAC PDUs** through a trellis → RS(24,16,9) → PN44-descramble FEC chain
> (`internal/radio/p25/phase2/process.go`), the PN44 seeded from
> WACN/System/NAC. The control channel stays Phase 1 — and issue #882
> proved the FEC knobs can validate, display, and still never arrive on the
> wideband pipeline.

**Key takeaways**

- **Phase 2 is a traffic-channel upgrade, not a new system.** The control
  channel stays Phase 1 C4FM; grants steer subscribers to 6000-baud TDMA
  carriers, and Part 5's `AccessTDMA` flag routes them into the right
  voice chain.
- **The demodulator is a rotation argument.** `PiOver4DQPSK` serves TETRA
  at π/4 and Phase 2 H-DQPSK at π/8 — one differential core, so a fix in
  it lands in both protocols at once.
- **The superframe is the addressing scheme.** Twelve 30 ms sub-frames,
  even index → timeslot 0, odd → timeslot 1; the ISCH tells you what each
  carries without inspecting the payload.
- **Descramble in the channel-bit domain, before the FEC.** The PN44
  applies to *channel* bits per TIA-102.BBAC-1 §7.2.5; descrambling the
  information bits after the trellis fails the outer RS on every real
  scrambled burst (the issue #915 ordering lesson).

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| IQ → dibits | RRC → Gardner → carrier recovery → π/8 differential | `internal/radio/p25/phase2/receiver/receiver.go` |
| Shared demod core | rotation π/4 = TETRA, π/8 = Phase 2 | `internal/dsp/demod/piover4_dqpsk.go` (`PiOver4DQPSK`) |
| Superframe slicing | dibits → 12 SlotType-tagged sub-frames | `superframe_decoder.go` (`SuperframeDecoder`) |
| Slot typing | Golay(24,12)-protected SlotType + counter | `isch.go` (`DecodeISCH`) |
| MAC FEC chain | descramble → deinterleave → trellis → RS(24,16,9) | `process.go` (`decodeMACPDUDibits`) |
| MAC opcodes | grants, idle, hangtime, encryption sync, IDEN_UP | `mac.go` / `mac_standard.go` / `mac_vendor.go` |
| Grant routing | Phase 1 CC stamps Phase 2 FEC config on TDMA grants | `phase1/control.go` (`publishVoiceGrant`) |

## In this post

- **Two voices per carrier** — TDMA arithmetic and the hybrid reality.
- **π/8 on a shared core** — the H-DQPSK receiver's borrowed lessons.
- **The superframe and the ISCH** — how 30 ms sub-frames get names.
- **MAC PDUs, where TSBKs used to be** — the FEC chain's ordering lesson.
- **The knob that defaulted to zero** — issue #882, twin-pipeline audit.

## Two voices per carrier

Phase 1 spends a whole 12.5 kHz channel on one call. Phase 2 raises the
symbol rate from 4800 to **6000** — one dibit per symbol, 12 kbps of
channel capacity — then time-division-multiplexes it into two slots, each
sufficient for one AMBE+2 voice stream plus signalling. Two concurrent
calls per carrier: half the frequencies, or twice the capacity:

| Axis | Phase 1 FDMA | Phase 2 TDMA |
|---|---|---|
| Symbol rate | 4800 sym/s | 6000 sym/s |
| Modulation (downlink) | C4FM (or CQPSK/LSM) | H-DQPSK |
| Calls per 12.5 kHz | 1 | 2 |
| Voice codec | IMBE 4400 | AMBE+2 |
| Signalling unit | TSBK | MAC PDU |
| Control channel | Phase 1 | still Phase 1 |

The last row is the practical one. On real deployments the **control
channel stays Phase 1** — FDMA, 4800 baud, C4FM, everything Parts 2–5
built — and voice grants direct subscribers to Phase 2 traffic carriers.
GopherTrunk models this exactly: the Phase 1 CC resolves a grant through
the band plan, asks `BandPlan.IsTDMA(channelID)` (the 0x33 IDEN_UP flag
from [Part 5]({{ '/blog/deep-dives/p25-end-to-end-05-channels-band-plans/' | relative_url }})),
and publishes it with `Protocol: "p25-phase2"` plus a `P25Phase2Decode`
config block — trellis/RS/interleave/scrambler modes and the PN44 seed —
so the composer's Phase 2 voice chain decodes the traffic carrier the way
this site expects. Hybrid systems that lost this routing were issue #376;
the config block that rode along badly was #882, below.

## π/8 on a shared core

The Phase 2 receiver (`internal/radio/p25/phase2/receiver`) recovers the
dibit stream with a familiar chain: RRC matched filter (α = 0.20, span 8
symbols), Gardner timing, carrier recovery, differential decode. The
differential core is not new code — it is the same `PiOver4DQPSK`
primitive that decodes TETRA, selected by one constructor argument:

```go
// internal/dsp/demod/piover4_dqpsk.go (shape)
// The same primitive serves multiple control-channel modulations; the
// rotation argument selects between them:
//   - math.Pi/4 — true π/4-DQPSK as used by TETRA TMO (18000 sym/s, α=0.35)
//   - math.Pi/8 — the π/8-shifted variant P25 Phase 2 H-DQPSK
//     (6000 sym/s, α=0.20)
func NewPiOver4DQPSK(sps, span int, alpha, rotation float64) *PiOver4DQPSK
```

One primitive, one rotation apart — so the differential-decode lessons the
[TETRA series]({{ '/blog/deep-dives/tetra-end-to-end-01-pi4-dqpsk-carrier/' | relative_url }})
paid for apply here for free, and a fix in the core lands in both
protocols. The borrowing runs deeper. At 6000 baud a residual carrier
offset of ~750 Hz already rotates every symbol a full π/4 into the wrong
quadrant, so the receiver's `ClockGardner` path carries the same
coarse-seed → NCO → Costas architecture
[Part 6]({{ '/blog/deep-dives/p25-end-to-end-06-cqpsk-lsm/' | relative_url }})'s
CQPSK path built for issue #492 — ported for issue #813, multipath
modulus-CV seed gate included, retuned for the rate (`costasLoopBWHz =
150`, the same ~2.5%-of-baud normalised bandwidth).

And Phase 2 got the levers Phase 1 C4FM still lacks: an opt-in **blind CMA
equalizer** on the recovered symbol stream and a per-bit **soft-decision**
track feeding a true soft Viterbi in the MAC path (both issue #915), plus
an opt-in DC blocker for zero-IF spurs. The receiver options read like a
checklist of everything the weak-signal work concluded — which makes the
default C4FM voice path's bare chain, Part 12's subject, all the more
conspicuous.

## The superframe and the ISCH

A Phase 2 carrier is a continuous dibit stream with a strict rhythm: a
**360 ms superframe** of **twelve 30 ms sub-frames** (180 dibits each),
alternating between the two timeslots — even sub-frame index → timeslot 0,
odd → timeslot 1 (`Subframe.Timeslot = i & 1` in
`superframe_decoder.go`). `SuperframeDecoder` anchors that grid on the
20-dibit outbound sync word and slices the stream into tagged sub-frames.

The tag is the **ISCH** — the Inter-Slot Signalling Channel field
prefixing every sub-frame: a 12-bit word protected by extended
Golay(24,12,8), carrying a 4-bit SlotType and a 0..11 sub-frame counter
(`DecodeISCH`, `isch.go`). The SlotType says what a sub-frame carries
without touching the payload:

| SlotType | Carries |
|---|---|
| `Voice4V` | 4 AMBE+2 voice frames |
| `Voice2V` | 2 voice frames + MAC |
| `MAC_PTT` / `MAC_END` | transmission start / end signalling |
| `MAC_IDLE` / `MAC_ACTIVE` / `MAC_HANGTIME` | channel state |
| `MAC_SIGNALING` / `MAC_END_CONT` | signalling / end + continuation |

<figure class="lab-figure">
<svg viewBox="0 0 680 240" width="680" height="240" role="img" aria-label="One 12.5 kilohertz carrier at 6000 symbols per second splits into a 360 millisecond superframe of twelve 30 millisecond sub-frames; even sub-frames belong to timeslot 0 and odd sub-frames to timeslot 1, so two independent voice calls interleave on one carrier, each sub-frame prefixed by an ISCH naming its slot type">
  <text x="30" y="30" fill="currentColor" font-size="11" font-weight="bold">one 12.5 kHz carrier · 6000 sym/s</text>
  <line x1="30" y1="44" x2="650" y2="44" stroke="currentColor"/>
  <!-- superframe row -->
  <text x="30" y="72" fill="var(--fg-muted)" font-size="10">superframe (360 ms):</text>
  <!-- 12 subframes, alternate accent -->
  <rect x="170" y="58" width="40" height="26" fill="none" stroke="var(--accent)" stroke-width="2"/>
  <rect x="210" y="58" width="40" height="26" fill="none" stroke="currentColor"/>
  <rect x="250" y="58" width="40" height="26" fill="none" stroke="var(--accent)" stroke-width="2"/>
  <rect x="290" y="58" width="40" height="26" fill="none" stroke="currentColor"/>
  <rect x="330" y="58" width="40" height="26" fill="none" stroke="var(--accent)" stroke-width="2"/>
  <rect x="370" y="58" width="40" height="26" fill="none" stroke="currentColor"/>
  <text x="440" y="76" fill="var(--fg-muted)" font-size="11">…</text>
  <rect x="570" y="58" width="40" height="26" fill="none" stroke="var(--accent)" stroke-width="2"/>
  <rect x="610" y="58" width="40" height="26" fill="none" stroke="currentColor"/>
  <text x="190" y="75" text-anchor="middle" fill="var(--accent)" font-size="9">0</text>
  <text x="230" y="75" text-anchor="middle" fill="currentColor" font-size="9">1</text>
  <text x="270" y="75" text-anchor="middle" fill="var(--accent)" font-size="9">2</text>
  <text x="630" y="75" text-anchor="middle" fill="currentColor" font-size="9">11</text>
  <text x="170" y="100" fill="var(--fg-muted)" font-size="9">each sub-frame: 30 ms · 180 dibits · ISCH prefix names its SlotType</text>
  <!-- slot lanes -->
  <text x="30" y="140" fill="var(--accent)" font-size="10" font-weight="bold">timeslot 0 (even) — call A</text>
  <rect x="240" y="126" width="52" height="22" fill="none" stroke="var(--accent)" stroke-width="2"/>
  <rect x="304" y="126" width="52" height="22" fill="none" stroke="var(--accent)" stroke-width="2"/>
  <rect x="368" y="126" width="52" height="22" fill="none" stroke="var(--accent)" stroke-width="2"/>
  <text x="266" y="141" text-anchor="middle" fill="var(--accent)" font-size="9">4V</text>
  <text x="330" y="141" text-anchor="middle" fill="var(--accent)" font-size="9">4V</text>
  <text x="394" y="141" text-anchor="middle" fill="var(--accent)" font-size="9">2V</text>
  <text x="30" y="180" fill="currentColor" font-size="10" font-weight="bold">timeslot 1 (odd) — call B</text>
  <rect x="240" y="166" width="52" height="22" fill="none" stroke="currentColor"/>
  <rect x="304" y="166" width="52" height="22" fill="none" stroke="currentColor"/>
  <rect x="368" y="166" width="52" height="22" fill="none" stroke="currentColor"/>
  <text x="266" y="181" text-anchor="middle" fill="currentColor" font-size="9">4V</text>
  <text x="330" y="181" text-anchor="middle" fill="currentColor" font-size="9">MAC</text>
  <text x="394" y="181" text-anchor="middle" fill="currentColor" font-size="9">4V</text>
  <text x="440" y="141" fill="var(--fg-muted)" font-size="9">← two independent calls,</text>
  <text x="440" y="155" fill="var(--fg-muted)" font-size="9">interleaved 30 ms at a time</text>
  <text x="30" y="222" fill="var(--fg-muted)" font-size="10">Subframe.Timeslot = index &amp; 1 — the grid, not the payload, assigns each sub-frame to its call</text>
</svg>
<figcaption>The 360 ms Phase 2 superframe: twelve ISCH-tagged sub-frames alternating between two timeslots, so one carrier interleaves two independent voice calls.</figcaption>
</figure>

One honesty note the package documentation (`phase2.go`) makes explicitly:
several TIA-102.BBAB/BBAC wire details — the exact sync cadence, the ISCH
bit packing, the per-burst interleaver — are not in the spec figures the
project has access to, so each is a **documented working model confined to
one file with a symmetric encoder**. If a real capture shows the sync
recurring at a different sub-frame, the fix is one constant
(`SyncSubframeIndex`). Containment is what separates a working model from
folklore.

## MAC PDUs, where TSBKs used to be

Phase 2 signalling arrives as **MAC PDUs** (surveyed in
[Protocol Decoders Part 4]({{ '/blog/deep-dives/protocol-decoders-04-p25-phase-2-tdma-mac/' | relative_url }})):
after FEC removal, an 18-byte unit — opcode byte, optional MFID for
vendor messages, then payload. The
opcode families rhyme with Part 3's TSBKs (`OpGroupVoiceChannelGrant`,
`OpIdentifierUpdate`, `OpNetworkStatusBroadcastUpdate`…) plus
TDMA-specific channel-state messages (`OpMACIdle`, `OpMACHangtime`,
`OpEncryptionSync`). The FEC chain recovering one is the densest in the
P25 tree, and its *ordering* carries a paid-for lesson:

```go
// internal/radio/p25/phase2/process.go (shape) — the MAC FEC chain
// Per TIA-102.BBAC-1 §7.2.5 the PN44 scrambler applies to CHANNEL bits:
// descramble the 146 on-wire dibits first, then deinterleave, then the
// 4-state ½-rate trellis, then re-group to 24 hex symbols for the outer
// RS(24, 16, 9). Descrambling the recovered information bits AFTER the
// trellis fails the outer RS on a genuinely scrambled burst.
func decodeMACPDUDibits(macDibits []uint8, mode TrellisMode, /* … */) (MACPDU, bool)
```

GopherTrunk originally descrambled the 144 information bits after the
trellis. Synthetic round-trips that made the same choice on both sides
stayed green — the same self-consistency trap as Part 1's RRC filter and
Part 3's SCCB offsets — while the outer RS(24,16,9) would have failed on
any genuinely scrambled on-air burst. Issue #915 fixed the order and put
the rule in the comment. The PN44 runs continuously over the 4320-bit
superframe with per-slot offsets, seeded from the site's identity:
`framing.PN44SeedFromIdentity(WACN, SystemID, NAC)` — which is why the
Phase 1 control channel snapshots its network model at grant time and
ships the seed *with* the grant. A wrong or missing seed doesn't look like
"wrong seed"; it looks like a dead traffic channel.

## The knob that defaulted to zero

That config block riding on the grant is where issue #882 lived — this
series' cleanest specimen of the twin-pipeline failure. The single-channel
CC pipeline (`internal/scanner/ccdecoder/pipelines.go`) parsed the Phase 2
FEC options — trellis, RS, interleave, scrambler — and passed them into
the decoder it built. The wideband engine
(`internal/scanner/widebandt2/engine.go`) built the same decoder and
passed **none of them**: they defaulted to zero. The options validated,
the API read them back correctly, and the decoder never saw them — and
the reporter's topology, a Phase 1 CC granting Phase 2 voice on a hybrid
system, fell exactly in the gap.

The observable tell was one field in a startup line: `composer: p25p2
voice chain started trellis=0` where a correctly-plumbed system says
`trellis=1`. The composer now warn-logs the configuration itself — live
Phase 2 MAC PDUs are always trellis-encoded, so `trellis=0` on a real
system is announced as a misconfiguration rather than discovered as
silence. The fix (`parseP25Phase2FECModes` in the wideband engine,
mirroring the ccdecoder path) verified on air the same day, and the
deferred remainder — the wideband path also dropped the Phase 1 demod
mode — became Part 6's issue #935. The full postmortem is the
[Two Pipelines finale]({{ '/blog/solution-postmortem/from-the-issue-tracker-22-two-pipelines/' | relative_url }});
the durable rule: **a config knob is only real once the pipeline announces
its effective value and the announcement matches the file.**

### How the twin-generation design shaped the Go code

- **Shared primitives, parameterised.** `PiOver4DQPSK(rotation)` and the
  ported carrier-recovery stack mean Phase 2 inherits every
  differential-decode fix TETRA and CQPSK earn, instead of forking one.
- **Working models are quarantined.** Each unverified wire detail lives in
  one file with one constant and a symmetric encoder, so a real-capture
  correction is a local change, never archaeology.
- **The grant carries the decode contract.** `P25Phase2Decode` ships FEC
  modes and the PN44 seed from the CC that knows them to the voice chain
  that needs them — resolved once, announced at chain start.
- **Slot identity is grid arithmetic.** `Timeslot = index & 1` comes from
  the sync-anchored superframe, not payload inspection, so voice routing
  survives sub-frames whose payload FEC fails.

## Where this goes next

Phase 2 splits a carrier between two AMBE+2 streams; Phase 1 gives one
IMBE stream the whole channel — the voice most P25 systems still carry.
[Part 8]({{ '/blog/deep-dives/p25-end-to-end-08-imbe-voice/' | relative_url }})
follows it end to end: the 1728-bit LDU with nine IMBE frames woven
between link-control words, the Golay/Hamming FEC, and the composer chain
that turns a grant into a WAV — including the layout bug that let only two
of nine voice frames decode while every test stayed green.

## FAQ

**Does GopherTrunk need a separate control channel config for Phase 2?**
Usually not. Most Phase 2 deployments keep a Phase 1 control channel, so
you configure the system like any P25 system; grants to TDMA carriers are
detected via the 0x33 IDEN_UP flag and routed automatically. The
per-system Phase 2 FEC knobs exist for sites whose MAC framing needs
non-default handling.

**How is P25 Phase 2's modulation related to TETRA's?**
Both are differential QPSK variants with a per-symbol rotation — TETRA at
π/4 and 18000 baud, Phase 2 H-DQPSK at π/8 and 6000 baud. GopherTrunk
decodes both with one rotation-parameterised primitive, which is why
equalizer and soft-decision work migrates between them so readily.

**Why did Phase 2 change vocoder from IMBE to AMBE+2?**
Each TDMA slot has roughly half a Phase 1 channel's bit budget, and
[AMBE+2]({{ '/blog/deep-dives/voice-coding-07-ambe-plus-2/' | relative_url }})
delivers comparable voice at a lower rate — the same codec family DMR and
NXDN use, which is why GopherTrunk's Phase 2 voice extraction delegates
its frame FEC to the shared DMR path.

**What does `trellis=0` in my p25p2 startup log mean?**
That the voice chain believes MAC PDUs arrive un-trellis-coded — true only
for pre-stripped test fixtures, never for live air. On a real system it
means the Phase 2 FEC options aren't reaching this pipeline (the issue
#882 failure) or are misconfigured; set `p25_phase2_trellis_mode=on` and
check the startup line agrees.

**Can GopherTrunk decode both timeslots at once?**
Yes — the superframe grid assigns every sub-frame to its slot by index,
and each granted call's voice chain consumes its own slot. Two concurrent
calls on one Phase 2 carrier are two calls, recorded separately.

## Series navigation

**Part 7 of 14** · ←
[Part 6: CQPSK & LSM — The Linear Twin Path]({{ '/blog/deep-dives/p25-end-to-end-06-cqpsk-lsm/' | relative_url }})
· Next →
[Part 8: IMBE Voice — From LDU to WAV]({{ '/blog/deep-dives/p25-end-to-end-08-imbe-voice/' | relative_url }})
