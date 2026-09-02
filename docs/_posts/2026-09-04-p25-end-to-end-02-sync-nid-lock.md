---
title: "P25 End to End, Part 2: Frame Sync, the NID & What 'Locked' Means"
description: How GopherTrunk turns a raw P25 dibit stream into frames — the 48-bit frame sync word, the BCH(63,16)-protected NID carrying NAC and DUID, the status symbols hiding every 36th dibit, and the tiered alignment search that decides when a control channel is really locked.
category: deep-dives
keywords: p25 frame sync word, p25 nid nac duid, bch 63 16 error correction, p25 status symbols, p25 control channel lock, p25 duid types, tsdu ldu1 ldu2 hdu, p25 sync detection, gophertrunk p25
tags: [p25-end-to-end, p25, framing, fec, sync, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "P25 End to End"
series_part: 2
---

*Part 2 of **P25 End to End**, a 14-part deep dive that follows North America's
dominant trunking protocol through GopherTrunk — from a raw C4FM carrier to
recorded, named, multi-site voice.
[Part 1]({{ '/blog/deep-dives/p25-end-to-end-01-c4fm-carrier/' | relative_url }})
built the chain that turns IQ into dibits at 4800 a second. This part gives
that firehose punctuation: the frame sync word that marks a frame, the NID
that names it, the status symbols interleaved through everything — and the
deliberately conservative moment at which GopherTrunk is willing to log
`control channel locked`. Getting that moment wrong in either direction cost
real debugging weeks, and the machinery here carries the scars.*

> **TL;DR:** Every P25 Phase 1 frame opens with a **48-bit frame sync word**
> (hex `0x5575F5FF77FF`, 24 dibits — `phase1.FrameSyncWord`) followed by the
> **64-bit NID**: a 12-bit NAC plus 4-bit DUID protected by **BCH(63,16,11)**
> and a per-DUID flag bit (`ParseNID`,
> `internal/radio/p25/phase1/nid.go`). One 2-bit **status symbol** rides after
> every 35 data dibits (`p25StatusStride = 36`), so nothing downstream can
> count on-air dibits naively. `searchNID` probes a bounded grid of alignment
> hypotheses in two tiers — clean BCH decodes (≤ 6 corrections) are trusted,
> marginal ones (7–11) are admitted only when the frame's TSBK CRC
> corroborates the same alignment — and **only a TSDU locks the channel**: a
> PDU is Multi-Block Trunking (Part 4), and voice DUIDs never flip the flag.

**Key takeaways**

- **Sync is a correlation, not an equality.** The FSW detector tolerates up
  to 4 mismatched dibits and tries dibit-alphabet rotations — restricted to
  {0, 2} on the C4FM path, where rotations 1 and 3 can only be BCH
  miscorrections (issue #275).
- **The NID's last bit is a per-DUID flag, not a parity bit.** Assuming
  "obviously it's overall parity" masked issue #275 for a long time; the real
  rule (1 for LDU1/LDU2, 0 for the rest) is pinned against OP25.
- **A frame straddling chunk boundaries must still assemble.** A 16 KiB
  RTL-SDR USB transfer carries ~19 P25 symbols; without cross-call buffering
  every FSW hit was discarded and nothing ever locked — the original issue
  #275 field symptom.
- **Lock is an event with a name attached.** `KindCCLocked` fires on a
  corrected NID with a TSDU DUID, carries frequency and NAC, and the decoder
  honours lock edges only for its own carrier — a foreign system's lock on
  the shared bus must not flip your flag.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| Frame sync word | 24-dibit pattern, tolerance-4 correlator | `internal/radio/p25/phase1/sync.go` (`FrameSyncWord`, `SyncDetector`) |
| Rotation search | {0,2} on C4FM, all four on CQPSK | `sync.go` (`RotationsC4FM`, `RotationsAll`) |
| NID parse | BCH(63,16,11) + per-DUID flag → NAC + DUID | `nid.go` (`ParseNID`, `framing.BCHDecode63_16`) |
| Status symbols | one 2-bit symbol per 35 data dibits | `control.go` (`p25StatusStride`) |
| Alignment search | ±6-dibit grid, two acceptance tiers | `control.go` (`searchNID`, `NIDSearchSpan`, `NIDAcceptErrs`) |
| Cross-call assembly | frame buffering across IQ chunks | `control.go` (`ControlChannel.Process`, `frameLookahead`) |
| Lock event | TSDU-only, per-carrier filtered | `control.go` (`parseFrame`); `decoder.go` (`lockEventMatchesActive`) |

## In this post

- **The flag in the stream** — the FSW correlator, tolerance, rotations and margins.
- **The NID: name tag and type tag** — NAC, the DUID zoo, and BCH(63,16,11).
- **The stowaways** — status symbols and why dibit arithmetic is never simple.
- **When the FSW isn't enough** — the tiered NID alignment search.
- **What "locked" actually means** — the TSDU rule, the event, and the hunt.

## The flag in the stream

The frame sync word is P25's punctuation mark: 48 bits — 24 dibits —
opening every data unit:

```go
// internal/radio/p25/phase1/sync.go (shape)
// FrameSyncWord is the 48-bit P25 Phase 1 frame sync word, expressed as
// 24 dibits (TIA-102.BAAA §6.1.1). Hex: 0x5575F5FF77FF, MSB-first dibits.
var FrameSyncWord = [24]uint8{
    1, 1, 1, 1, 1, 3, 1, 1, 3, 3, 1, 1,
    3, 3, 3, 3, 1, 3, 1, 3, 3, 3, 3, 3,
}
```

Notice the alphabet: only dibits 1 and 3 — the FSW rides entirely on the
*outer* ±1800 Hz symbols, giving the correlator maximum eye separation.
`SyncDetector` slides a 24-dibit window over the stream and fires when the
mismatch count is within tolerance (default 4 of 24). Two refinements matter.
First, the detector tries **cyclic rotations** of the dibit alphabet, because
a front-end quirk (conjugated IQ, discriminator polarity) can present the
whole stream rotated; on the C4FM path the search is restricted to
`RotationsC4FM = {0, 2}` — identity and polarity flip — because rotations 1
and 3 are physically impossible on a discriminator stream, and leaving them
in let the BCH stage "correct" misaligned garbage into a parity-valid
pseudo-NID at rot 3 (issue #275). Second, `ProcessWithMargin` reports how
*comfortably* each hit cleared tolerance — a margin distribution pressed
against 1 says a lock is barely holding before it drops.

## The NID: name tag and type tag

After the FSW comes the Network ID — 64 bits answering two questions: *whose
system is this* and *what kind of frame follows*.

| Field | Bits | Meaning |
|---|---|---|
| NAC | 12 | Network Access Code — the system's colour code; a receiver filters on it |
| DUID | 4 | Data Unit ID — what the rest of the frame is |
| BCH parity | 47 | BCH(63,16,11) over the first 63 bits — corrects up to 11 bit errors |
| Flag bit | 1 | fixed per-DUID value (1 for LDU1/LDU2, 0 otherwise) |

The DUID is the frame-type switch this entire series will keep returning to:

| DUID | Name | What it is | Covered in |
|---|---|---|---|
| 0x0 | HDU | voice call header (MI/ALGID/KID) | Parts 8–9 |
| 0x3 | TDU | terminator, no link control | Part 8 |
| 0x5 | LDU1 | voice superframe half, link control | Part 8 |
| 0x7 | TSDU | trunking signalling (TSBKs) — **the control channel** | Part 3 |
| 0xA | LDU2 | voice superframe half, encryption sync | Parts 8–9 |
| 0xC | PDU | packet data — **or Multi-Block Trunking** | Part 4 |
| 0xF | TDULC | terminator with link control | Part 8 |

Two details in `ParseNID` earn their comments. The BCH(63,16,11) decoder
corrects up to 11 bit errors — generous armour that needs supervision (next
section), because a code that can move 11 bits can also *manufacture* a
valid-looking codeword from a misaligned window. And the 64th bit is **not**
an overall parity bit, however obvious that feels: it is a fixed per-DUID
flag — 1 for LDU1 and LDU2, 0 for everything else — confirmed against OP25.
The obvious-but-wrong parity reading masked issue #275 longer than any other
single mistake. The
[framing & FEC deep dive]({{ '/blog/deep-dives/sdr-internals-09-framing-fec/' | relative_url }})
covers the BCH machinery itself; here it's enough that the NID arrives with
an error count, and that count feeds the acceptance logic below.

## The stowaways

P25 interleaves a 2-bit **status symbol** into the stream after every 35 data
dibits — `p25StatusStride = 36` in `control.go`, counting from the FSW's
first dibit. On the air these carry channel-status signalling; for a decoder
their main effect is arithmetic sabotage: nothing that indexes "N dibits past
the FSW" is correct until the stowaways are counted. The 32-dibit NID plus
first 98-dibit TSBK block — 130 data dibits — occupies 134 on-air dibits
(`frameLookahead = 130 + 4`), and a voice LDU carries 24 of them scattered
through its 1728 bits. Every parser either strips them
(`StripStatusSymbols`) or counts around them, and the alignment search treats
"stripped or not" as an explicit hypothesis axis — a status-phase fault looks
exactly like a corrupted NID until you probe both ways.

<figure class="lab-figure">
<svg viewBox="0 0 680 220" width="680" height="220" role="img" aria-label="P25 frame layout: the 24-dibit frame sync word, the 32-dibit NID holding NAC, DUID and BCH parity, then up to three 98-dibit TSBK blocks, with status-symbol ticks every 36th on-air dibit">
  <!-- stream bar -->
  <rect x="20" y="70" width="110" height="34" rx="4" fill="none" stroke="var(--accent)" stroke-width="2"/>
  <text x="75" y="86" text-anchor="middle" fill="var(--accent)" font-size="10">FSW</text>
  <text x="75" y="99" text-anchor="middle" fill="var(--accent)" font-size="9">24 dibits</text>
  <rect x="130" y="70" width="150" height="34" rx="4" fill="none" stroke="currentColor" stroke-width="2"/>
  <text x="205" y="86" text-anchor="middle" fill="currentColor" font-size="10">NID — 32 dibits</text>
  <text x="205" y="99" text-anchor="middle" fill="var(--fg-muted)" font-size="9">NAC(12) · DUID(4) · BCH(47) · flag(1)</text>
  <rect x="280" y="70" width="120" height="34" rx="4" fill="none" stroke="currentColor"/>
  <text x="340" y="86" text-anchor="middle" fill="currentColor" font-size="10">TSBK block 1</text>
  <text x="340" y="99" text-anchor="middle" fill="var(--fg-muted)" font-size="9">98 dibits</text>
  <rect x="400" y="70" width="120" height="34" rx="4" fill="none" stroke="currentColor" stroke-dasharray="4 3"/>
  <text x="460" y="86" text-anchor="middle" fill="currentColor" font-size="10">TSBK block 2</text>
  <text x="460" y="99" text-anchor="middle" fill="var(--fg-muted)" font-size="9">optional</text>
  <rect x="520" y="70" width="120" height="34" rx="4" fill="none" stroke="currentColor" stroke-dasharray="4 3"/>
  <text x="580" y="86" text-anchor="middle" fill="currentColor" font-size="10">TSBK block 3</text>
  <text x="580" y="99" text-anchor="middle" fill="var(--fg-muted)" font-size="9">LB=1 on last</text>
  <!-- status symbol ticks -->
  <line x1="128" y1="52" x2="128" y2="70" stroke="var(--accent)" stroke-width="3"/>
  <line x1="236" y1="52" x2="236" y2="70" stroke="var(--accent)" stroke-width="3"/>
  <line x1="344" y1="52" x2="344" y2="70" stroke="var(--accent)" stroke-width="3"/>
  <line x1="452" y1="52" x2="452" y2="70" stroke="var(--accent)" stroke-width="3"/>
  <line x1="560" y1="52" x2="560" y2="70" stroke="var(--accent)" stroke-width="3"/>
  <text x="345" y="40" text-anchor="middle" fill="var(--accent)" font-size="10">status symbols — one 2-bit stowaway every 36th on-air dibit</text>
  <!-- annotations -->
  <line x1="130" y1="118" x2="400" y2="118" stroke="var(--fg-muted)"/>
  <line x1="130" y1="112" x2="130" y2="124" stroke="var(--fg-muted)"/>
  <line x1="400" y1="112" x2="400" y2="124" stroke="var(--fg-muted)"/>
  <text x="265" y="136" text-anchor="middle" fill="var(--fg-muted)" font-size="10">NID + first TSBK = 130 data dibits → 134 on-air (frameLookahead)</text>
  <text x="340" y="170" text-anchor="middle" fill="currentColor" font-size="10">DUID = TSDU → decode TSBKs, lock · DUID = PDU → decode MBT, no lock (Part 4)</text>
  <text x="340" y="188" text-anchor="middle" fill="var(--fg-muted)" font-size="10">voice DUIDs (HDU/LDU1/LDU2/TDU) → "non-control DUID", recorded but never locked on</text>
</svg>
<figcaption>One P25 data unit on the air: FSW, NID, up to three TSBK blocks — with a status symbol shifting every count by one per 36 dibits, and only a TSDU allowed to lock the channel.</figcaption>
</figure>

## When the FSW isn't enough

Issue #275's most stubborn field shape: a reliably detected FSW followed by a
NID that *never* BCH-decoded. The NID dibits had to be individually sound —
the FSW wouldn't correlate otherwise — so the fault was alignment, not noise.
`searchNID` is the answer, a small case study in refusing both kinds of
error:

- It probes a bounded grid: NID start ± `NIDSearchSpan` (6 dibits), status
  symbols stripped or not, each allowed rotation.
- A hypothesis whose BCH decode needs **≤ `NIDAcceptErrs` (6) corrections**
  and passes the flag-bit check is *trusted* — accepted on the code's
  strength alone.
- A hypothesis needing **7–11 corrections** lands in the *marginal tier* — as
  likely a miscorrection of a misaligned guess as a real noisy NID — and is
  admitted only if the frame's TSBK **also decodes CRC-clean under the same
  alignment**. A wrong alignment cannot realistically fake both.
- A total reject isn't silent: the diag line carries an `err_pattern` — the
  per-dibit error map of the closest miss — so a reporter can see whether
  errors cluster at one end (timing slip), near dibit 31 (status-phase
  fault) or uniformly (SNR-limited), without re-running anything.

FSW hits found only by a looser fallback tolerance go through with
`requireCorroboration` set — even a clean NID must bring a CRC-valid TSBK —
so extended reach never bypasses the false-lock guard (issue #771's
discipline). And all of it runs over a buffer spanning `Process` calls: a
live RTL-SDR delivers IQ in chunks holding ~19 symbols, far short of a
154-dibit frame, and before the cross-call buffer existed every FSW hit was
discarded before its NID arrived. The
[anatomy of a CC decoder]({{ '/blog/deep-dives/protocol-decoders-01-anatomy-of-a-cc-decoder/' | relative_url }})
covers this buffering pattern across protocols; P25 is where it was learned.

## What "locked" actually means

`parseFrame` makes the final call, and the rule is one line with a long
history behind it:

```go
// internal/radio/p25/phase1/control.go (shape) — parseFrame
if best.nid.DUID == DUIDPacketDataUnit {
    // Multi-Block Trunking — decode it, but it does not lock the
    // channel. Only a TSDU proves a control channel.
    /* … resumeMBTBlocks … */
}
if best.nid.DUID != DUIDTrunkingSignaling {
    c.log.Debug("non-control DUID", "duid", best.nid.DUID, "nac", best.nid.NAC)
    return pendingHit{}, false
}
if !c.locked || c.lastNAC != best.nid.NAC {
    c.locked = true
    c.bus.Publish(events.Event{Kind: events.KindCCLocked, Payload: LockState{ /* … */ }})
    c.log.Info("control channel locked", "nac", best.nid.NAC, "freq", c.freqHz, /* … */)
}
```

**Only a TSDU proves a control channel.** A voice DUID means you're parked on
a traffic channel; a PDU is Multi-Block Trunking, decoded in full (Part 4)
but never lock-worthy on its own. When the flag flips, `KindCCLocked` carries
a `LockState` with frequency and NAC — and the consumer is as careful as the
producer: the ccdecoder honours lock and loss edges **only when the payload's
frequency matches its own active carrier** (`lockEventMatchesActive`),
because the bus is shared by every decoder in the process and a foreign
system's lock must not gate this one's autotune or its decode-quality chip.

Lock is also only the *end* of a longer story. Finding which frequency to
park on is the hunt machinery, told in
[control-channel hunting]({{ '/blog/deep-dives/the-hunt-06-control-channel-hunting/' | relative_url }})
and, for this exact protocol,
[locking a P25 system]({{ '/blog/deep-dives/the-hunt-07-locking-a-p25-system/' | relative_url }});
the first time it all worked end-to-end is
[its own war story]({{ '/blog/solution-postmortem/from-the-issue-tracker-01-first-p25-lock/' | relative_url }}).
This part's contribution is the contract at the boundary: by the time the
hunt hands over a frequency, "locked" means *a validated TSDU decoded here* —
nothing weaker.

### How the framing shaped the Go code

- **Acceptance is tiered, never binary.** `NIDAcceptErrs` splits trust from
  corroboration; the marginal tier borrows the TSBK CRC as a second witness.
- **Every reject is a measurement.** The closest-miss `err_pattern` diag
  turned "it doesn't lock" into "errors cluster at dibit 31, suspect the
  status phase".
- **Rotations are a physics question.** `RotationsC4FM` exists because the
  demod mode determines which alphabet rotations are possible; CQPSK keeps
  all four for its genuine phase ambiguity.
- **The buffer is part of the protocol.** `frameLookahead` and the
  pending-hit continuation make frame assembly independent of IQ chunking —
  the difference between a decoder that works on files and one that works on
  USB.

## Where this goes next

A locked control channel is a promise of content: TSDUs arrive many times a
second, each packing up to three 98-dibit signalling blocks.
[Part 3]({{ '/blog/deep-dives/p25-end-to-end-03-tsbk-workhorse/' | relative_url }})
opens them up — trellis, deinterleaver, CRC, the opcode families that run a
trunking system — and tells the SCCB byte-offset story, where a round-trip
test blessed a parser reading a field one byte early.

## FAQ

**What is a NAC in P25?**
A 12-bit Network Access Code carried in every frame's NID — functionally the
system's colour code, used to reject co-channel traffic from other systems.
GopherTrunk logs it on lock (`control channel locked nac=…`) and re-publishes
the lock event if the NAC changes mid-stream, since that means a different
system has appeared on the frequency.

**Why does GopherTrunk log `nid corrected errs=…` — is that bad?**
Not by itself: a few BCH corrections per NID is normal on a marginal signal.
The number to watch is the tier — `errs` above 6 means the NID was only
admitted because a TSBK CRC corroborated it, and a stream living permanently
in that band is SNR-limited, the demod-quality territory of Part 12.

**Can GopherTrunk lock on a voice channel by mistake?**
No — that is what the TSDU rule prevents. A voice DUID produces a
`non-control DUID` debug line (suppressible via `QuietNonControlDUID`) and
nothing else. Camping on a traffic channel is a hunt-layer mistake; the
framing layer refuses to bless it.

**What are status symbols actually for?**
On the air, channel-status signalling between subscriber units and the
repeater. For a receive-only decoder their content is mostly irrelevant but
their *positions* are everything — they shift every dibit count by one per 36
on-air dibits. GopherTrunk treats them as geometry: stripped before parsing,
probed as a hypothesis when alignment fails.

**How likely is a false FSW hit at tolerance 4?**
Individually unlikely, and it doesn't survive scrutiny anyway: a random hit
must still produce a NID that BCH-decodes with the right per-DUID flag — and
in the marginal tier a CRC-valid TSBK too. A false sync costs a little
compute, never a false lock.

## Series navigation

**Part 2 of 14** · ←
[Part 1: C4FM & the Shape of a P25 Carrier]({{ '/blog/deep-dives/p25-end-to-end-01-c4fm-carrier/' | relative_url }})
· Next →
[Part 3: TSBKs — The Control Channel's Workhorse]({{ '/blog/deep-dives/p25-end-to-end-03-tsbk-workhorse/' | relative_url }})
