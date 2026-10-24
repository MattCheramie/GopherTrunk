---
title: "From the Issue Tracker, Season 2, Part 11: Phantom Neighbours — Spliced Broadcasts and the Seams in TETRA MAC Reassembly"
description: "How a TETRA rig's Neighbor sites table filled with cells that did not exist — a 1.5 GHz entry, a 402.0125 MHz phantom confirmed twice — through four rounds of MAC-layer seams: a stray bit at every fragment boundary, a TM-SDU that ran to the block end, and a reassembly that spliced two transmissions of the rotating D-NWRK-BROADCAST."
category: solution-postmortem
keywords: tetra neighbour cells, d-nwrk-broadcast decode, tetra mac fragment reassembly, mac-frag mac-end layout, tetra fill bits length indication, aach reed muller classification, phantom neighbour sites, tl_sdu_hex, tetra neighbor sites report, gophertrunk from the issue tracker s2
tags: [from-the-issue-tracker-s2, tetra, mac, neighbours, debugging, postmortem]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "From the Issue Tracker, Season 2"
series_part: 11
---

*Part 11 of **From the Issue Tracker, Season 2**, a 14-part run of
postmortems on GopherTrunk bugs that fought back.
[Part 10]({{ '/blog/solution-postmortem/issue-tracker-s2-10-rekey-dropped-both-overs/' | relative_url }})
followed a DMR re-key through the composer's drain coordination to the
call-ID fence that keeps both overs. This part moves to TETRA's control
plane and a "Neighbor sites" table that filled with sites nobody could
find on the dial — and the four rounds of MAC-layer seams, each
deterministic, each repeating bit-identically past a confirm-twice gate,
that put them there.*

> **TL;DR:** TETRA's D-NWRK-BROADCAST (`ParseDNwrkBroadcast`,
> `internal/radio/tetra/mle_parse.go`) advertises the serving cell's
> neighbours and fills the systems report's "Neighbor sites". On an
> operator's 467.9125 MHz rig it decoded two cells then garbage, then a
> 1.5 GHz neighbour, then a 402.0125 MHz phantom confirmed twice. Four
> seams: `macFragmentPayload` leaked the fill-bit indication on
> MAC-FRAG and two flags on MAC-END (one stray bit per seam); `tmSDU` ran
> to the block end instead of the MAC length indication, so fill bits became
> phantom optional fields; the reassembly had no continuity check, so a
> MAC-END spliced onto any stale start fragment; and three holes let a chain
> survive a lost block: a half-decoded control slot counted as recovered,
> `DecodeAACH` classified garbage as "traffic", and adjacency was four
> frames with no slot-grid check. Now: `fragMaxGapDibits` (two frames),
> `onChainSlotGrid`, `aachClassifyMaxErrs` = 2, ≥ 8 trailing bits rejects
> the broadcast, an implausible cell distrusts the whole list
> (`plausibleNeighbourCell`), and `neighbourExpiry` = 30 min. Pinned by a
> literal field TL-SDU (`TestParseDNwrkBroadcastRejectsSplicedFieldCapture`)
> and `frag_continuity_test.go`; both 4 Sep captures replay 8/8 real cells.

**Key takeaways**

- **Confirm-twice cannot catch deterministic corruption.** A splice or a
  leaked tail repeats bit-identically on every rebroadcast, so a gate that
  waits for the same content twice promotes garbage with truth.
- **A parser that reads a fixed prefix never notices a bad boundary.**
  Every other L3 parser tolerated bits leaking past the PDU; the one with
  trailing presence bits turned them into fields.
- **An ML decoder always returns *something*.** `DecodeAACH` hands back the
  nearest RM(30,14) codeword for any input; its distance is the confidence,
  and a classification at distance 4–6 is a coin flip.
- **Integrity evidence must sit on the chain's own grid.** Real slots are
  255 dibits apart; the correlator's off-grid hits prove nothing, and a
  check that counted them aborted every chain on a timeshare carrier.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| Neighbour broadcast parse | decodes D-NWRK-BROADCAST; rejects ≥ 8 trailing bits | `internal/radio/tetra/mle_parse.go` (`ParseDNwrkBroadcast`) |
| Fragment payload seams | strips header fields per osmo-tetra `rx_macfrag`/`rx_macend` | `internal/radio/tetra/mac.go` (`macFragmentPayload`, `tmSDU`, `stripFillBits`) |
| Chain continuity | two frames + slot grid between pieces | `control.go` (`fragMaxGapDibits`, `onChainSlotGrid`, `fragChainAdjacentLocked`) |
| Lost-block detection | per block, trusted AACH only | `downlink.go` (`decodeDownlinkSlot`, `aachClassifyMaxErrs` = 2) |
| Whole-list distrust | one implausible cell drops the broadcast, logs `tl_sdu_hex` | `control.go` (`learnNeighbourCells`, `plausibleNeighbourCell`) |
| Expiry | 30 min without re-advertisement, aged only across accepted broadcasts | `neighbourExpiry`, `TestLearnNeighbourCellsExpiresUnadvertisedCells` |
| Field pin | literal TL-SDU from the 4 Sep log, 132 bits of residue | `mle_parse_test.go` (`TestParseDNwrkBroadcastRejectsSplicedFieldCapture`) |

## In this post

- **Seven cells, two decoded** — a stray bit at every fragment seam.
- **A neighbour at 1.5 GHz** — a TM-SDU that ran to the block end.
- **Confirmed twice, still wrong** — the 402.0125 MHz phantom and the instrument.
- **Reading the hex** — two transmissions in one PDU, three holes.
- **The regressions, failing first** — literal field bits and synthetic chains.

## Seven cells, two decoded

A TETRA base station advertises its neighbours in the MLE
D-NWRK-BROADCAST (EN 300 392-2 §18.4.1.4.1): re-selection parameters,
then up to seven neighbour-cell elements, each a 12-bit main carrier and
a P-bit-gated tail of optionals — band extension, MCC, MNC, LA, status
fields. It is the TETRA analogue of the P25 adjacent status broadcast
([P25 End to End Part 10]({{ '/blog/deep-dives/p25-end-to-end-10-sites-roaming/' | relative_url }})).

The broadcast is too long for one MAC block, so it arrives fragmented: a
start-fragment MAC-RESOURCE, zero or more MAC-FRAG, then a MAC-END. On
the operator's 120 s 467.9125 MHz MRC capture it decoded the first ~two
cells and garbage after them. **Deleting exactly one bit at the seam made all seven
advertised cells decode.** `macFragmentPayload` had skipped only the type
and subtype on a MAC-FRAG — the fill-bit indication leaked into the
payload — and only fill and length on a MAC-END, where the slot-granting
and channel-allocation flags leaked too:

```go
// internal/radio/tetra/mac.go (shape) — per osmo-tetra rx_macfrag / rx_macend
r.u(3)               // MAC PDU type + subtype
fill := r.bit() == 1 // fill-bit indication: MAC-FRAG and MAC-END alike
if sub == macEnd {
    if n := macPDULengthBits(uint8(r.u(6))); n > 0 && n < end { end = n }
    if r.bit() == 1 { r.u(8) }           // basic slot-granting element
    if r.bit() == 1 { parseChanAlloc(r) } // channel allocation
}
```

The round-trip test had been green the whole time because its encoder
shared the wrong layout — the
[self-consistent trap]({{ '/blog/solution-postmortem/from-the-issue-tracker-20-self-consistent-trap/' | relative_url }})
again. Single-block PDUs were unaffected, so everything else looked fine; only a fragmented L3 PDU corrupts, from the seam
onward. With the layouts pinned against osmo-tetra, the parser went live
behind a confirm-twice gate (`neighboursPending`): the 6-bit PD+type
check is thin enough that a corrupted-but-CRC-passing TL-SDU parses
plausibly, and a one-shot version surfaced 18 "neighbours". The
repeating 8 were real — carriers 467.99–470.00 MHz, LAs
1021–1089 — and `TestTETRANeighbourReportReplay` became the on-air harness.

## A neighbour at 1.5 GHz

The follow-up report, in the operator's words: "totally bogus entries in
the neighbours list, like a 1.5 GHz one". Those entries had confirmed
twice — the tell that the corruption was **deterministic**, so the
confirm-twice gate could not help. Two more MAC boundary holes were
behind it.

First, `tmSDU` and `macFragmentPayload` handed everything up to the
**block** end to Layer 3. The MAC length indication — the PDU's own length
in octets, which osmo-tetra's `rx_resrc` and tetra-kit's `decodeLength`
both honour — was parsed and never applied, and the fill-bit indication
never stripped the fill run ('1' then '0's, §23.4.3.2). Whatever followed
the PDU inside the block leaked into the TL-SDU. Prefix-reading parsers
never noticed; `ParseDNwrkBroadcast`'s last neighbour cell reads a tail of
P-bits, so it turned the leaked bits into phantom optionals — band bits
`1111` render as a neighbour above 1 GHz, which no TETRA allocation
occupies. `tmSDU` now ends at the length
indication and strips fill when the flag says so (`stripFillBits`); the
test builders had been writing a placeholder length the decoder ignored,
and honouring the field forced them to stamp real lengths
(`stampMACResourceLength`) — encoder/decoder drift in yet another dress.

Second, the reassembly had **no continuity check at all**. A MAC-END was
spliced onto whatever start fragment was in flight, however old, across
however many lost blocks. Pieces must now be stream-adjacent: the NCDB
detector's dibit position is plumbed through `decodeDownlinkSlot` into
`curSlotPos`, and `fragChainAdjacentLocked` bounds the gap by
`fragMaxGapDibits`. `plausibleNeighbourCell` — main carrier ≠ 0, band 1..9
when the extension is present — went in as defence in depth behind both.

## Confirmed twice, still wrong

The 4 Sep field report was "one bogus neighbor site": a "cell 0, carrier
80, synced" phantom — 65 MHz off-network, the web UI's 402.0125 MHz
neighbour — confirming twice at 02:11, 02:31 and 02:33, *alongside* a
band-13 implausible sibling in the same broadcasts. The list is parsed
sequentially, so an impossible cell is proof the bit alignment was lost,
and its plausible-looking siblings came from the same misaligned bits.
`learnNeighbourCells` now drops the **whole** list when any cell fails
plausibility, and logs the raw TL-SDU:

```text
DBG tetra: dropping d-nwrk-broadcast with implausible neighbour cell — whole list distrusted system=… cell_id=2 main_carrier=698 has_ext=true band=13 cells=2 tl_sdu_hex=…
```

`TestLearnNeighbourCellsRejectsImplausibleCells` reproduces the operator's
exact row beside a band-13 sibling, learns it twice, and asserts zero
surfaced neighbours; its second half is the no-harm control, a fully
plausible broadcast learned twice afterwards that still surfaces its cell
(cell 4, carrier 2720), so one rejected list does not poison the next.
Determinism is the whole point. A random bit error fails confirm-twice on
its own; a splice or a leaked tail is the same bits every cycle, which is
exactly what the gate reads as confirmation, and no amount of repetition
can tell the two apart. The `tl_sdu_hex` field is therefore the point of
the fix as much as the drop: it is the instrument for the *next* report,
the
[census-everything]({{ '/blog/solution-postmortem/from-the-issue-tracker-21-census-everything/' | relative_url }})
move of logging evidence on the failure path. The corruption source was
still unknown. The hex dump found it one day later, with no new capture.

## Reading the hex

The operator's 10.4 h log held 11 rejected dumps. Decoded by hand, every
one is the **real cell list** — cells 9, 10, 11 … as 57-bit elements,
perfect — that mid-cell jumps back to another copy of the same list.
Every dump left 15–388 trailing bits after a "complete" list, where a
genuine broadcast ends within a fill run of under 8 bits. The mechanism: a mis-continued
reassembly **splicing two transmissions** of the rotating broadcast.
Bogus sites that had looked plausible — `mnc=1021`, `mnc=1032`, the real
cells' LAs shifted into the MNC field — were the same splices confirming
twice, because the loss pattern repeats with the broadcast schedule.

Three holes let a chain survive the losses that set a splice up:

1. **A control slot with one decoded SCH/HD half counted as recovered.**
   On an AACH-confirmed control slot both halves carry signalling, so the
   failed half *is* a lost block.
   "Lost" is now per block: `slotFullyRecovered := fullOK || (half1OK && half2OK)`.
2. **`DecodeAACH` is a maximum-likelihood search that always returns the
   nearest RM(30,14) codeword.** Its `errs` is the Hamming distance; garbage
   lands at 4–6, so a faded AACH was a coin-flip "control" or "traffic",
   and whenever it read "traffic" the chain kept going. The classification
   is trusted only at `errs ≤ aachClassifyMaxErrs` (2); an unconfident slot
   abandons the chain. (Searching the other rotations on unconfident slots
   was tried and reverted: it took the 120 s replay from 61 s to 139 s.)
3. **Adjacency was four frames, with no grid.** It is now two frames plus
   jitter — `fragMaxGapDibits = 2*4*255 + 2*fragGridJitterDibits` — and the
   continuation must land on the chain's 255-dibit slot grid
   (`onChainSlotGrid`). The grid is load-bearing: the NCDB detector also
   emits spurious off-grid correlator hits (+92 and +163 dibits on the 4 Sep
   captures), and a naive gap-between-emits check aborted every chain on this timeshare carrier — neighbours 0/8.

<figure class="lab-figure">
<svg viewBox="0 0 680 220" width="680" height="220" role="img" aria-label="A timeline on a 255-dibit slot grid: transmission A's start fragment and MAC-FRAG, then its lost MAC-END slot, then transmission B's MAC-END three frames later. The old window spliced B onto A; the new rule abandons A at the lost slot and refuses B as stale.">
  <text x="340" y="14" text-anchor="middle" fill="currentColor" font-size="10" font-weight="bold">fragment chain on the 255-dibit slot grid (one MCCH frame = 1020 dibits)</text>
  <line x1="30" y1="60" x2="650" y2="60" stroke="var(--fg-muted)"/>
  <g fill="var(--fg-muted)" font-size="8" text-anchor="middle">
    <text x="60" y="76">0</text><text x="200" y="76">1020</text><text x="340" y="76">2040</text><text x="480" y="76">3060</text><text x="620" y="76">4080</text>
  </g>
  <rect x="44" y="40" width="32" height="16" fill="none" stroke="currentColor" stroke-width="1.5"/>
  <text x="60" y="35" text-anchor="middle" fill="currentColor" font-size="8">A start</text>
  <rect x="184" y="40" width="32" height="16" fill="none" stroke="currentColor" stroke-width="1.5"/>
  <text x="200" y="35" text-anchor="middle" fill="currentColor" font-size="8">A MAC-FRAG</text>
  <rect x="324" y="40" width="32" height="16" fill="none" stroke="var(--fg-muted)" stroke-dasharray="2 2"/>
  <text x="340" y="35" text-anchor="middle" fill="var(--fg-muted)" font-size="8">A MAC-END lost</text>
  <rect x="604" y="40" width="32" height="16" fill="none" stroke="var(--accent)" stroke-width="1.5"/>
  <text x="620" y="35" text-anchor="middle" fill="var(--accent)" font-size="8">B MAC-END</text>
  <circle cx="96" cy="60" r="3" fill="none" stroke="var(--fg-muted)"/>
  <text x="96" y="90" text-anchor="middle" fill="var(--fg-muted)" font-size="8">+92 off-grid</text>
  <text x="26" y="124" text-anchor="end" fill="var(--fg-muted)" font-size="8">old</text>
  <path d="M60 120 L620 120" fill="none" stroke="var(--fg-muted)" stroke-dasharray="4 3"/>
  <text x="340" y="112" text-anchor="middle" fill="var(--fg-muted)" font-size="8">4-frame window, lost slot read "traffic" → B spliced onto A → 132 trailing bits → phantom sites</text>
  <text x="26" y="178" text-anchor="end" fill="var(--accent)" font-size="8">new</text>
  <path d="M60 174 L340 174" fill="none" stroke="var(--accent)" stroke-width="1.5"/>
  <line x1="340" y1="166" x2="340" y2="182" stroke="var(--accent)" stroke-width="1.5"/>
  <text x="340" y="196" text-anchor="middle" fill="var(--accent)" font-size="8">lost control slot on the grid → abandonFragment · 3 frames late → stale</text>
  <text x="340" y="214" text-anchor="middle" fill="currentColor" font-size="8">fragMaxGapDibits = 2046 · onChainSlotGrid · aachClassifyMaxErrs = 2</text>
</svg>
<figcaption>A loses its MAC-END; B's arrives three frames later inside the old window and the halves parse as one broadcast. The new rule abandons A at the lost slot and refuses B as stale.</figcaption>
</figure>

Two layers sit behind those. `ParseDNwrkBroadcast` rejects **≥ 8
trailing bits** after the neighbour list — the whole splice class, even
through a new hole — and logs the same `tl_sdu_hex`
(`tetra: rejecting misframed d-nwrk-broadcast`). And neighbours **expire**
after `neighbourExpiry` (30 min) without re-advertisement, aged only
across *accepted* broadcasts so a CC outage expires nothing.

## The regressions, failing first

The field pin is literal. `TestParseDNwrkBroadcastRejectsSplicedFieldCapture`
feeds the 17:02 `tl_sdu_hex` from the 4 Sep log — 552 bits — to the parser
and asserts rejection; the old parser returned `ok=true` with seven
"cells" and 132 bits of residue. `TestParseDNwrkBroadcastTrailingResidue`
pins the boundary: 7 fill bits parse, 8 residual bits reject.

The chain holes are pinned in `frag_continuity_test.go`, each failing
against the old code by publishing an enriched grant from a corrupt
reassembly (`enrichedGrantSeen`: a D-SETUP with source and emergency bit
that only a completed chain carries). `TestFragmentReassemblyAbandonedOnUndecodedControlSlot`
decodes a slot whose AACH says "control" over garbage halves;
`…AbandonedOnHalfSlotLoss` gives it one clean SCH/HD half and one lost;
`…AbandonedOnUnclassifiableSlot` picks a garbage AACH beyond the trust
gate at every rotation that *also* coin-flips to "traffic" under the old
code; `…RequiresOnGridContinuation` refuses a MAC-END at +92 dibits and
one three frames late, and still accepts one two frames late (the
frame-18 skip). `TestFragmentReassemblySurvivesDecodedSlots` is the
no-harm control: clean control slots mid-chain must not abandon it.

On air, both 4 Sep captures replay **8/8 real cells** through
`TestTETRANeighbourReportReplay`. The periodic DEBUG line

```text
DBG tetra: abandoning TM-SDU fragment reassembly — awaiting rebroadcast (continuity guard, not a parse error) reason="undecoded control slot mid-reassembly"
```

is the guard working, each abandon costing one broadcast cycle; the
reporter read the earlier wording as a parse bug, so the line now says
what it is, and `frag_abandons` in the decode-status line tracks the rate
([Field Notebook Part 3]({{ '/blog/tutorials/field-notebook-03-tetra-decode-status/' | relative_url }})
reads that line). Not closed: the §18.5.17 status optionals are surfaced
raw, their bit maps un-named until a capture confirms them.

### How the seams shaped the Go code

- **Boundary metadata is honoured, never inferred.** `tmSDU` and
  `macFragmentPayload` end at the length indication and strip fill only
  when the PDU's own flag says to.
- **Confidence travels with the decode.** `DecodeAACH` returns `errs`;
  `aachTrusted` is a separate boolean, and a classification is data only
  below the gate.
- **Positions are stamped before any block decodes.** `curSlotPos` is set
  at the top of `decodeDownlinkSlot` so every check sees the same grid.
- **Rejections carry their evidence.** Every reject path logs
  `packBitsHex(tl)` — how the root cause was found from a log alone.

## Where this goes next

The next seam is not in a protocol at all. An Android user asked for an
rtl_power-style sweep over rtl_tcp and could not start the binary: a
Linux build with `CGO_ENABLED=0` still linked `libdl.so.2` through the
ALSA player's `purego` loader.
[Part 12]({{ '/blog/solution-postmortem/issue-tracker-s2-12-cgo-disabled-is-not-static/' | relative_url }})
measures the dynamic section, adds the `nolibasound` tag and `check-static.sh`, and reports what three phones did.

## FAQ

**Why did GopherTrunk show TETRA neighbour sites that do not exist?**
Three MAC-layer boundary bugs: `macFragmentPayload` leaked a bit at each
fragment seam, `tmSDU` passed block-tail bits that `ParseDNwrkBroadcast`
read as phantom optional fields, and the reassembly spliced a MAC-END from
one transmission onto another's start fragment. All three repeated bit-identically past the confirm-twice gate.

**What does "abandoning TM-SDU fragment reassembly" in the log mean?**
It is the continuity guard working, not a parse error. A control slot on
the chain's grid decoded nothing or only half, or its AACH could not be
classified confidently, so the chain is dropped rather than spliced
across the loss. The broadcast repeats within seconds; `frag_abandons`
counts the rate.

**Why is an AACH decode only trusted at two or fewer errors?**
`DecodeAACH` is a maximum-likelihood search over RM(30,14) that always
returns the nearest codeword, and its error count is that codeword's
Hamming distance. Garbage lands at distance 4–6, so an unconfident decode
is a coin-flip classification. `aachClassifyMaxErrs` = 2 keeps the chance
of garbage passing near 0.7 %.

**How is the fix verified on air?**
A literal TL-SDU from the operator's 4 Sep log is committed as
`TestParseDNwrkBroadcastRejectsSplicedFieldCapture` and fails against the
old parser, and both 4 Sep diversity captures replay all 8 real neighbour
cells through `TestTETRANeighbourReportReplay`. The chain holes are pinned
synthetically in `frag_continuity_test.go` with a no-harm control.

**Why do neighbours expire after 30 minutes?**
The broadcast rotates through the live neighbour list within a minute or
two, so a cell absent for `neighbourExpiry` has left it — a
reconfigured network or a phantom that slipped an earlier gate. Expiry is
measured only across accepted broadcasts, so a CC outage never expires anything.

## Series navigation

**Part 11 of 14** · ←
[Part 10: The Re-Key That Dropped Both Overs — Drain Coordination Fenced by Call ID]({{ '/blog/solution-postmortem/issue-tracker-s2-10-rekey-dropped-both-overs/' | relative_url }})
· Next →
[Part 12: CGO_ENABLED=0 Is Not Static — purego, libasound and the Termux Build]({{ '/blog/solution-postmortem/issue-tracker-s2-12-cgo-disabled-is-not-static/' | relative_url }})
