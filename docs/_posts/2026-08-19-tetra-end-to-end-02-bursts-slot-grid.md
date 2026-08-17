---
title: "TETRA End to End, Part 2: The Burst Zoo & the Slot Grid"
description: How GopherTrunk finds structure in the 18000-dibit-per-second TETRA firehose — the synchronisation and normal downlink bursts, the three training sequences that mark them, the 255-dibit slot grid, and the one-slot anchor shift that silently misfiles every traffic burst when you get it wrong.
category: deep-dives
keywords: tetra burst structure, synchronisation burst sb, normal continuous downlink burst, tetra training sequences, tetra slot grid, tetra sync pdu, tetra multiframe, bsch colour code zero, ndbsbslotshift, gophertrunk tetra
tags: [tetra-end-to-end, tetra, framing, tdma, sync, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "TETRA End to End"
series_part: 2
---

*Part 2 of **TETRA End to End**, a 14-part deep dive into how GopherTrunk turns
one real 25 kHz TETRA carrier — the MCC 250 / MNC 13 cell — into clear recorded
voice. [Part 1]({{ '/blog/deep-dives/tetra-end-to-end-01-pi4-dqpsk-carrier/' | relative_url }})
turned IQ into a dibit stream and planted
the property everything leans on: information rides in phase transitions, so a
residual carrier offset shows up as a constant 0..3 rotation of every dibit.
Now we need punctuation. A TETRA downlink is a continuous stream of bursts on a
rigid TDMA grid, and this part is about how GopherTrunk finds them: which burst
carries what, how the training sequences are correlated, and the one-slot
anchor lesson — `ndbSBSlotShift` — that only a real capture could teach.*

> **TL;DR:** The TMO downlink alternates two burst shapes. The
> **synchronisation burst (SB)** lays out freq-correction → **BSCH**(60
> dibits) → sync training sequence(19) → broadcast(15) → **BNCH**(108); the
> BSCH is always scrambled with colour 0, so a cold receiver can decode its
> **SYNC PDU** (`tetra.ParseSyncPDU`) and learn the cell's colour code, frame
> and multiframe counts. The **normal continuous downlink burst (NCDB)** puts
> its data *around* the training sequence — BKN1 before, AACH halves either
> side, BKN2 after (`internal/radio/tetra/traffic.go`). Bursts are found by
> correlating three real ETSI training sequences under **all four dibit
> rotations** (`SyncDetector`). The SB anchors the **255-dibit slot grid**,
> but its training sequence sits late in the burst, so the anchor lands **one
> NDB slot early** — `ndbSBSlotShift = 3` corrects it, verified against a real
> same-carrier capture where grant timeslots finally lined up with decoded
> slots.

**Key takeaways**

- **Two bursts carry the whole downlink.** SB for acquisition (BSCH + BNCH
  around the long sync training sequence), NCDB for everything else (signalling
  and traffic around the short normal training sequence).
- **The BSCH is the bootstrap.** Scrambled with colour 0 by spec, it decodes
  with zero configuration and hands over the colour code that unlocks every
  other logical channel — Part 4 is entirely about that handoff.
- **Correlation, not magic constants.** The training sequences are the real
  EN 300 392-2 §9.4.4.3 bit arrays; placeholder constants once made lock
  impossible on air, a postmortem told in
  [From the Issue Tracker Part 17]({{ '/blog/solution-postmortem/from-the-issue-tracker-17-placeholder-constants/' | relative_url }}).
- **The slot grid is anchored, and the anchor has an offset.** A burst's slot
  number is `round((L − sbAnchor)/255)` — plus a shift of 3, because the SB's
  training sequence position and the NDB's differ inside the slot. Get that
  wrong and every burst files under the wrong timeslot with no error anywhere.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| Training sequences | real §9.4.4.3 bit arrays: NTS1/NTS2 (22b), extended (30b), STS (38b) | `internal/radio/tetra/sync.go` (`NormalTrainingSeq1`, `SyncTrainingSeq`) |
| Sliding correlator | pattern match within tolerance, all four rotations | `sync.go` (`SyncDetector`, `rotateDibits`) |
| SB geometry | BSCH `[L−60, L)`, BNCH `[L+34, L+142)` around the STS | `internal/radio/tetra/process.go` (`processSB`, `sbBSCHDibits`) |
| NCDB geometry | BKN1 `[L−115, L−7)`, AACH halves, BKN2 `[L+19, L+127)` | `internal/radio/tetra/traffic.go`, `downlink.go` (`downlinkNCDB`) |
| SYNC PDU | colour, TN/FN/MN, MCC/MNC from the BSCH | `internal/radio/tetra/sync_pdu.go` (`ParseSyncPDU`) |
| Slot grid anchor | SB pins slot 1; each slot is 255 dibits | `traffic.go` (`slotOf`, `ndbSBSlotShift`) |

## In this post

- **The burst zoo** — SB and NCDB, and what each block inside them carries.
- **Finding a burst** — three training sequences, four rotations, one detector shape.
- **The SYNC PDU** — the bootstrap message a cold receiver can always read.
- **The slot grid** — 255 dibits per slot, and time as frames and multiframes.
- **The anchor lesson** — why the SB pins the grid one slot early.

## The burst zoo

TETRA numbers its downlink physical layer around two burst shapes, both 255
dibits of slot but organised very differently. The **synchronisation downlink
burst (SB)** is the acquisition burst, transmitted in slot 1 of frame 18 of
every multiframe. Its layout, in dibits relative to the leading dibit `L` of
its 19-dibit synchronisation training sequence (STS):

```go
// internal/radio/tetra/process.go (shape) — SB geometry
// freq-correction → BSCH(60) → STS(19) → broadcast(15) → BNCH(108).
// Relative to the STS leading dibit L: BSCH is [L-60, L), BNCH is [L+34, L+142).
const (
    stsDibits         = 19  // SyncTrainingDibits length
    sbBSCHDibits      = 60  // BSCH block 1 (120 type-5 bits)
    sbBroadcastDibits = 15  // broadcast block between STS and BNCH
    sbBNCHDibits      = 108 // BNCH block 2 (216 type-5 bits, SCH/HD-coded)
)
```

The BSCH half carries the SYNC PDU (below); the BNCH half carries SYSINFO —
the cell's main carrier, band and access parameters, channel-coded like any
SCH/HD block.

The **normal continuous downlink burst (NCDB)** carries everything else:
signalling on the control channel, traffic on a voice carrier. Its defining
quirk is that the data does not follow the training sequence — it *surrounds*
it:

```go
// internal/radio/tetra/traffic.go (shape) — NCDB layout, dibits relative to L
//  BKN1 (block 1) : [L-115, L-7)   108 dibits (216 type-5 bits)
//  AACH half 1    : [L-7,   L)       7 dibits ( 14 bits)
//  training seq   : [L,     L+11)   11 dibits ( 22 bits)
//  AACH half 2    : [L+11,  L+19)    8 dibits ( 16 bits)
//  BKN2 (block 2) : [L+19,  L+127)  108 dibits (216 type-5 bits)
```

Any decoder that slices "N dibits after the training sequence" — the natural
first implementation — misses BKN1 entirely and starts reading mid-AACH. Both
the control-channel slot decoder (`downlinkNCDB` in `downlink.go`) and the
traffic extractor are therefore built as rolling buffers with *look-back*: a
training-sequence hit is queued as pending until the buffer holds the full
`[L−115, L+127)` span, then the blocks are sliced around it. The 30-bit AACH
split across the midamble is the slot's address label — its access-assignment
element says what the slot is carrying — and it becomes the demux key for
concurrent calls in Part 5.

## Finding a burst: three sequences, four rotations

Bursts are located by correlating known training sequences against the dibit
stream. GopherTrunk stores the real ETSI EN 300 392-2 §9.4.4.3 sequences as
bit arrays — normal 1 and 2 at 22 bits / 11 dibits, extended at 30 / 15, and
the synchronisation sequence at 38 / 19 — and `SyncDetector` slides each over
the stream reporting matches within a mismatch tolerance. Two details carry
the weight:

- **All four rotations, always.** Part 1's residual-CFO rotation means the
  pattern may arrive uniformly rotated by 0..3, so every detector bank is
  built as `NewSyncDetector(rotateDibits(pattern, r), tol)` for `r = 0..3` —
  eight detectors for the two normal sequences, four for the STS.
- **NTS1 vs NTS2 is a signal, not a duplicate.** A burst whose midamble matches
  the *second* normal training sequence is a stolen half-slot (STCH) — urgent
  signalling displacing speech — and the traffic extractor tags it
  (`pendingHit.stolen`) rather than treating it as a weaker NTS1 match.

The constants themselves have a scar. The original implementation shipped
placeholder `uint64` "hex" values that were both truncated and matched no spec
value, so the control channel correlated against sequences that never existed
on air — lock was structurally impossible, while every synthetic test (built
from the same wrong constants) passed. That is this series' villain making an
early appearance;
[From the Issue Tracker Part 17]({{ '/blog/solution-postmortem/from-the-issue-tracker-17-placeholder-constants/' | relative_url }})
does the full autopsy, and the
[training-sequence reference]({{ '/reference/tetra-training-sequences/' | relative_url }})
lists the corrected values.

## The SYNC PDU: what a cold receiver can read

The BSCH exists so a receiver with zero configuration can bootstrap. By
§8.2.5.2 it is always scrambled with colour code 0 — every other channel uses
the cell's colour — so its 60 decoded type-1 bits are readable before you know
anything about the cell. GopherTrunk parses them with the osmo-tetra-compatible
field layout:

```go
// internal/radio/tetra/sync_pdu.go (shape)
type SyncPDU struct {
    ColourCode uint8  // 6-bit colour code          (bits  4..9)
    TN         uint8  // timeslot number            (bits 10..11)
    FN         uint8  // frame number               (bits 12..16)
    MN         uint8  // multiframe number          (bits 17..22)
    MCC        uint16 // mobile country code        (bits 31..40)
    MNC        uint16 // mobile network code        (bits 41..54)
}
```

Three of those fields — MCC, MNC, ColourCode — combine into the 30-bit
extended colour code that seeds the scrambler for everything else on the cell
(`SyncPDU.ExtendedColourCode`, Part 4's subject). The other three are *time*:
TETRA counts a 4-slot frame, an 18-frame multiframe, and a 60-multiframe
hyperframe (§4.5.2), and the SYNC PDU tells you where in that hierarchy you
are. The decoder uses MN advancement as its sync-continuity heartbeat: rather
than log thousands of sync bursts, it logs once per multiframe on an MN change
and flags gaps — a monotonically advancing MN is the difference between "lock"
and "the correlator fired on noise," a distinction that pays for itself again
on DMO in Part 11.

## The slot grid: 255 dibits, four slots, 56.67 ms

Under the bursts is arithmetic. One slot is 255 dibits (510 bits); four slots
make a 1020-dibit frame, which at 18000 dibits/s is 56.67 ms. On a voice
carrier all four slots can carry independent calls, so a burst's timeslot
number matters: it decides which call's recording a speech frame belongs to.

The traffic extractor derives the slot from the SB. Its STS detectors watch
the stream, and each hit pins `sbAnchor`; a later burst leading at dibit `L`
is then slot `round((L − sbAnchor)/255) mod 4`, with rounding absorbing small
intra-slot jitter. The anchor refreshes on every SB — once per multiframe — so
slow clock drift can't accumulate.

<figure class="lab-figure">
<svg viewBox="0 0 680 210" width="680" height="210" role="img" aria-label="A TETRA downlink timeline divided into 255-dibit slots. A synchronisation burst in the first slot carries its training sequence late in the burst, and an arrow shows its detected position landing one slot before the frame's TN1 traffic burst; the ndbSBSlotShift constant of 3 realigns the grid so subsequent bursts read as timeslots 1 through 4.">
  <line x1="20" y1="90" x2="660" y2="90" stroke="var(--fg-muted)"/>
  <line x1="20" y1="130" x2="660" y2="130" stroke="var(--fg-muted)"/>
  <g stroke="var(--fg-muted)">
    <line x1="20" y1="90" x2="20" y2="130"/><line x1="148" y1="90" x2="148" y2="130"/>
    <line x1="276" y1="90" x2="276" y2="130"/><line x1="404" y1="90" x2="404" y2="130"/>
    <line x1="532" y1="90" x2="532" y2="130"/><line x1="660" y1="90" x2="660" y2="130"/>
  </g>
  <text x="84" y="115" text-anchor="middle" fill="currentColor" font-size="10">SB (frame 18)</text>
  <text x="212" y="115" text-anchor="middle" fill="currentColor" font-size="10">TN1 traffic</text>
  <text x="340" y="115" text-anchor="middle" fill="currentColor" font-size="10">TN2</text>
  <text x="468" y="115" text-anchor="middle" fill="currentColor" font-size="10">TN3</text>
  <text x="596" y="115" text-anchor="middle" fill="currentColor" font-size="10">TN4</text>
  <text x="84" y="80" text-anchor="middle" fill="var(--fg-muted)" font-size="9">255 dibits per slot</text>
  <!-- STS position late in SB -->
  <rect x="112" y="92" width="20" height="36" fill="var(--accent)" opacity="0.35"/>
  <text x="122" y="56" text-anchor="middle" fill="var(--accent)" font-size="9">STS detected here</text>
  <line x1="122" y1="60" x2="122" y2="88" stroke="var(--accent)"/>
  <polygon points="118,84 122,92 126,84" fill="var(--accent)"/>
  <!-- NTS position of TN1 burst -->
  <rect x="200" y="92" width="20" height="36" fill="currentColor" opacity="0.25"/>
  <text x="210" y="150" text-anchor="middle" fill="var(--fg-muted)" font-size="9">NTS midamble of TN1</text>
  <path d="M 122 168 Q 166 190 206 168" fill="none" stroke="var(--accent)"/>
  <polygon points="200,166 210,166 204,175" fill="var(--accent)"/>
  <text x="166" y="200" text-anchor="middle" fill="var(--accent)" font-size="10">raw anchor lands one NDB slot early — ndbSBSlotShift = 3 (≡ −1 mod 4) realigns TN1</text>
</svg>
<figcaption>The SB anchors the 255-dibit slot grid, but its training sequence sits late in the burst — the detected anchor lands one NDB slot before TN1's midamble, and the shift constant absorbs the offset.</figcaption>
</figure>

## The anchor lesson: `ndbSBSlotShift`

Here is the part no spec diagram hands you. The SB's training sequence is not
where the NDB's is: the STS sits *late* in the SB (after the frequency
correction and the whole BSCH block), while the NDB's midamble sits at the
slot's centre. So the raw subtraction `(L − sbAnchor)/255` produces slot
numbers that are internally consistent but globally off by one — every burst
one slot after the SB reads as slot 0 instead of TN1.

```go
// internal/radio/tetra/traffic.go (shape) — slotOf
const ndbSBSlotShift = 3 // (= −1 mod 4): a burst one slot after the anchor reads as TN1

func (te *TrafficExtractor) slotOf(L int) uint8 {
    if !te.haveAnchor {
        return 0 // unknown until an SB is seen
    }
    si := (int(math.Round(float64(L-te.sbAnchor)/float64(ndbSlotDibits))) + ndbSBSlotShift) % 4
    /* … */
    return uint8(si) + 1
}
```

What makes this a *lesson* rather than a constant is how it fails and how it
was fixed. A wrong shift produces no error at any layer: the bursts decode,
the CRCs pass, and every speech frame files under a mislabeled timeslot —
which on a multi-call carrier means calls swap audio. Nothing synthetic
catches it, because a synthetic fixture built from your own slot math is
self-consistent by construction (the villain again). The value 3 was verified
the only way it could be: against a reporter's real same-carrier capture,
where the control channel's granted timeslots (`ts1`, `ts2`) finally lined up
with the decoded slot tags once the shift was applied. And even then the code
treats the slot number with suspicion — the doc comment in `traffic.go` is
blunt that on real air the *AACH usage marker* is the reliable per-call demux
key while the slot number is telemetry, a distinction Part 5 turns into
routing.

### How that principle shaped the Go code

- **Geometry is a named constant set, not inline arithmetic.** `ndbBKN1Start`,
  `ndbAACH1Start`, `sbBSCHDibits` and friends make each burst's layout a
  reviewable table against the spec — and made the DMO geometry variant
  (Part 11) a diff instead of a rewrite.
- **Look-back is a buffer-management contract.** Both NCDB decoders keep a
  trailing margin (`ndbTrimMargin`) plus any pending hit's look-back, so a
  training hit near a chunk boundary never loses its BKN1.
- **Anchoring is refreshed, never trusted forever.** The SB anchor updates
  every multiframe and reports slot 0 ("unknown") until first seen — a
  traffic-only carrier with no SB degrades to unlabeled bursts rather than
  invented slots.

## Where this goes next

We can now slice a burst's 216 or 432 type-5 bits out of the stream — but
type-5 bits are still scrambled, interleaved, punctured and convolved.
[Part 3]({{ '/blog/deep-dives/tetra-end-to-end-03-channel-coding-crc/' | relative_url }})
walks the channel-coding chain back to information bits: two RCPC mother
codes, the Viterbi decoder, the interleavers — and the star exhibit, a CRC
that is not an LFSR at all, whose wrong implementation silently dropped every
on-air burst while all the synthetic tests stayed green.

## FAQ

**Why does the data surround the training sequence instead of following it?**
A midamble halves the maximum distance between any data bit and the channel
estimate the training sequence provides — both blocks sit within ~half a slot
of it. It also means burst detection is naturally centred: once the midamble
correlates, both blocks are at fixed offsets either side.

**What happens between SBs — is the receiver blind for a whole multiframe?**
No. The SB is only the *anchor and bootstrap*; every slot in between carries
NCDBs the receiver decodes continuously. The once-per-multiframe cadence only
bounds how often the slot-grid anchor and MN counter refresh.

**Can the receiver lock without ever decoding a BSCH?**
It can correlate training sequences and extract bursts, but it can't
*descramble* anything beyond the BSCH itself without the colour code the SYNC
PDU carries — so a "lock" without a BSCH decode is sync without comprehension.
That failure mode (lock succeeds, no messages decode) is exactly what a broken
scrambler looks like, and Part 4 tells that story twice.

**Why tolerance-based correlation instead of exact match?**
Real bursts arrive with bit errors, and an exact match would drop a burst
whose midamble took one hit even though its FEC-protected payload is fully
recoverable. The tolerances (2 for the 11-dibit normal sequences, 3 for the
19-dibit STS) trade a manageable false-hit rate — the downstream CRCs reject
impostors — for not losing marginal bursts. Part 13 shows what that false-hit
rate means on a *silent* channel, and why counting raw hits is never evidence
of traffic.

**How does GopherTrunk know a burst is a stolen half-slot?**
By which normal training sequence matched: NTS2 flags STCH (stolen) bursts,
NTS1 marks ordinary ones. The extractor counts stolen bursts
(`TrafficExtractor.StolenBursts`) so the composer's voice-path summary can
report signalling-in-speech events.

## Series navigation

**Part 2 of 14** · ←
[Part 1: π/4-DQPSK & the Shape of a TETRA Carrier]({{ '/blog/deep-dives/tetra-end-to-end-01-pi4-dqpsk-carrier/' | relative_url }})
· Next →
[Part 3: Channel Coding — RCPC, Viterbi & the CRC That Isn't an LFSR]({{ '/blog/deep-dives/tetra-end-to-end-03-channel-coding-crc/' | relative_url }})
