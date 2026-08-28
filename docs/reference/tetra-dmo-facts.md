---
slug: tetra-dmo-facts
title: TETRA DMO facts
entry_type: term
category: fn-protocol
description: "TETRA DMO facts are the direct-mode truths GopherTrunk verified on air: the DNB payload geometry, why the DM colour code must be recovered empirically with the network MNI folded in, and why a raw burst detection is not evidence of traffic."
keywords: tetra dmo, direct mode, DSB, DNB, DM colour code, colour recovery, MNI, slot grid, dnb_qualified, burst correlator false positives, EN 300 396, osmo-tetra-dmo
aka: [TETRA direct mode facts, DMO bring-up facts]
infobox:
  - { label: Type, value: Protocol + DSP facts }
  - { label: Key rule, value: "DNB payload geometry is −108/+11, not TMO's −115/+19" }
  - { label: Trap, value: "The DM colour code is not on the air — recover it empirically" }
  - { label: Noise meter, value: "dnb_total ≈ 18/s on an idle channel; dnb_qualified means traffic" }
see_also: [tetra-dmo, direct-mode-operation, tetra-lock-facts, tetra-mobile-network-identity, tetra-sync-pdu, tetra-scrambler, tetra-extended-colour-code]
related_reading:
  - { title: "From the Issue Tracker, Part 22: Two Pipelines", url: /blog/solution-postmortem/from-the-issue-tracker-22-two-pipelines/ }
cite_urls:
  - https://github.com/MattCheramie/GopherTrunk/issues/1003
  - https://github.com/MattCheramie/GopherTrunk/issues/764
---

**TETRA DMO facts** are the direct-mode truths that separate a
[TETRA DMO](/reference/tetra-dmo/) receiver that decodes real radios from one that only
passes its own tests. GopherTrunk's DMO bring-up (issue
[#1003](https://github.com/MattCheramie/GopherTrunk/issues/1003)) ran through a false
"encrypted" verdict, a noise-driven grant storm, and a colour-recovery blind spot before
clear voice decoded — and each wrong turn left behind a fact worth keeping.

## The DNB payload geometry is −108/+11, and it was measured, not assumed

A Direct Mode Normal Burst's two payload blocks sit at
`BKN1 [L−108, L)` and `BKN2 [L+11, L+119)` dibits relative to the training-sequence lead
`L`. The TMO-copied geometry (−115/+19, which osmo-tetra-dmo uses) is measurably worse: on
a real capture the TCH/S CRC yield has a *sharp* optimum at −108/+11. The general lesson —
when two layouts are plausible, sweep and let CRC yield vote — recurs throughout DMO work.

## "Encrypted" was a descramble bug: colour 0 still scrambles

The first on-air verdict was "traffic is encrypted": every speech CRC sat at the ~1/256
chance floor while signalling decoded fine. The radios were in fact clear (TEA0). The bug:
the DMO voice path inherited a TMO `if colour != 0` shortcut and skipped descrambling at
colour 0 — but TETRA [scrambling](/reference/tetra-scrambler/) is non-identity at colour 0
(the LFSR seeds to `0xC0000000`). Synthetic round-trips scrambled *and* descrambled
consistently, so they passed either way — the
[self-consistent test trap](/reference/self-consistent-test-trap/). The regression now
scrambles unconditionally on encode, the way the air does.

## The DM colour code is not recoverable from the air — sweep for it

The SCH/S is always colour-0-scrambled, so it always *reads* colour 0; the colour field's
exact bit offset in the DSB SCH/H is ambiguous on available captures, and hardcoding a
guessed offset is the same trap again. GopherTrunk instead recovers the colour empirically:
decode candidate colours and keep the one that maximises CRC-valid TCH/S, behind a
**dominance gate** (best ≥ 3× runner-up) that refuses a chance-floor winner. On the
verifying capture the true colour won 35 CRC-valid frames against ≤3 for every other — an
unambiguous verdict with no manual override.

## The colour sweep needs the network MNI folded in

On one operator's Motorola network (MCC 250 / MNC 1) the sweep found *no* dominant colour:
several candidates rose modestly at once — a pattern one radio scrambling with one label
cannot produce. The cause was upstream of the sweep: TETRA seeds the scrambler from the
full 30-bit [extended colour code](/reference/tetra-extended-colour-code/), and a sweep
over colours 0–63 with an implicit MNI of 0 can never reach an MNI≠0 network's seed. The
DMO SCH/S carries MNI 0 *on air* (correct parsing, not a bug), so the real
[mobile network identity](/reference/tetra-mobile-network-identity/) must come from
configuration (`tetra_mcc`/`tetra_mnc`), folded into every candidate seed.

## A raw DNB detection is not evidence of traffic

The DNB correlator is an 11-dibit training match at tolerance 2 under 8 filters, so about
1 in a thousand random dibit positions matches by chance — **~18 false DNBs per second** on
an idle channel at TETRA's dibit rate. The first live run granted a recording on a silent
channel 230 ms after startup. The qualifier that separates traffic from noise is *timing
structure*: one transmitter on one clock puts every burst on a single residue modulo the
255-dibit timeslot, while noise is uniform over all 255. GopherTrunk votes DNB leads onto
that slot grid, latches the agreeing residue (learned, never spec-derived — a wrong
hardcoded offset would silently kill granting), tracks symbol slips, and drops the latch
when the train ends so residual noise cannot keep a finished transmission "alive". In the
decode-status line, **`dnb_total` is a noise meter and `dnb_qualified` is the number that
means traffic** — a large gap between them is normal, not a fault.

## Operating quirks a DMO listener should expect

DMO grants carry **no talkgroup** (call-control is not decoded), so recordings file under
group 0. The channel lock is *sticky* across inter-PTT silence — frame-number liveness from
the [DM-SYNC PDU](/reference/tetra-sync-pdu/), not decode rate, keeps a camped channel from
being re-hunted. Colour recovery needs ~20 bursts, so a very short first transmission may
grant before the colour is known (the voice chain re-recovers it retroactively so leading
speech is kept). And with no release PDU on the air, even a healthy DMO call ends on
hangtime rather than at the moment of PTT release.

## Symptom table

| Symptom | Looks like | Actually | Fix / check |
|---|---|---|---|
| Signalling decodes, speech at CRC chance floor | Encryption | Colour-0 descramble skip, or wrong colour/MNI seed | Descramble unconditionally; sweep colours with the configured MNI ([#1003](https://github.com/MattCheramie/GopherTrunk/issues/1003)) |
| Colour sweep: several modest winners, none dominant | Weak signal / "guessing broken" | Wrong MNI in every candidate seed | Set `tetra_mcc`/`tetra_mnc` and re-sweep |
| Grants and recordings on a silent channel | Real traffic | Chance DNB correlations (~18/s) driving the grant | Slot-grid qualification; watch `dnb_qualified`, not `dnb_total` |
| `dnb_total` climbs steadily with no calls | Decoder bug | Normal correlator false-positive rate | Nothing — that counter is a noise meter |
| Recording ends "at a timeout", not at PTT release | Teardown bug | DMO has no release PDU; BFI-only calls die at first hangtime | Expect hangtime endings; zero speech frames means a decode problem upstream |

## Provenance

- [#1003](https://github.com/MattCheramie/GopherTrunk/issues/1003) — the DMO thread end to end: the geometry sweep, the colour-0 descramble fix, empirical colour recovery, the MNI fold, the slot-grid grant qualifier, and the on-air captures behind each.
- [#764](https://github.com/MattCheramie/GopherTrunk/issues/764) — the verification discipline (a green synthetic is not an on-air verdict) that shaped every step above.
