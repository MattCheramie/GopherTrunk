---
slug: p25-site-identity-semantics
title: P25 site-identity semantics
entry_type: term
category: fn-protocol
description: "What P25 site identity actually means on air: why a grant's site is not a radio's location, why the NAC cannot key a site, and how network identity, neighbor lists, and hybrid Phase 2 systems are decoded."
keywords: p25, site identity, rfss, site id, nac, wacn, system id, grant, registration, affiliation, neighbor sites, adjacent status broadcast, rfss status, network status, hybrid phase 2, tdma identifier update
see_also: [rfss, system-id, wacn, network-access-code, neighbor-site, trunking-site, multisite-trunking, roaming, registration, affiliation, p25-tsbk-opcodes, p25-identifier-update, p25-nid-duid, p25-onair-constants, p25-demod-mode-selection]
---

**P25 site identity** is the pair of identifiers — [RFSS](/reference/rfss/) ID plus site ID —
that a [trunking site](/reference/trunking-site/) broadcasts about itself on its
[control channel](/reference/control-channel/). The semantics look obvious and are widely
mis-assumed. Every fact below was established the hard way while building GopherTrunk's site
tracking and site API, mostly against a large multi-site network, and each one contradicts a
plausible first guess.

## A grant's site is not the radio's location

On a [multisite](/reference/multisite-trunking/) system, a wide-area talkgroup is granted on
*every* participating site simultaneously — one press of PTT surfaces the same call under
multiple site IDs at the same instant. Grant activity is therefore fine for
"talkgroup activity by site" metrics and useless for answering "where is this radio."

What *does* fix a radio to a site: [unit registration](/reference/registration/) and
[group affiliation](/reference/affiliation/) are handled by the radio's actual serving site,
so those events are genuine location observations. That is the basis for RID-to-site mobility
tracking ([#698](https://github.com/MattCheramie/GopherTrunk/issues/698)).

## What keys a site — and what doesn't

- **Key on `(rfss_id, site_id)`** within a system. Nothing weaker works.
- **The [NAC](/reference/network-access-code/) is not a site identifier.** On the network
  where this was measured, roughly 12 NAC values were reused across 70+ sites
  ([#698](https://github.com/MattCheramie/GopherTrunk/issues/698)).
- **`channel_id` is a band-plan slot** (an [Identifier Update](/reference/p25-identifier-update/)
  table index), not a per-site key — GopherTrunk deliberately omits it from `/api/v1/sites`
  rows.
- **Voice-channel frequencies do not cluster near their site's control channel.**
  Nearest-frequency site matching produces wrong answers; resolve channels through the
  decoded band plan instead.

## Network identity is decoded, never configured

A system's [WACN](/reference/wacn/), [system ID](/reference/system-id/), RFSS, and site ID
only exist over the air: they arrive in the Network Status Broadcast
([TSBK](/reference/p25-tsbk-opcodes/) opcode `0x3B`) and RFSS Status Broadcast (`0x3A`).
GopherTrunk once served `/api/v1/systems` from the static config copy, so the System
Information panel sat on "Awaiting status broadcasts" forever while the identical decoded
values were already live on `/api/v1/sites` — the endpoint just never consulted the site
tracker ([#673](https://github.com/MattCheramie/GopherTrunk/issues/673)).

Phase 2 TDMA control channels need their own camped-site cache fed by the same two
broadcasts, with the Color Code standing in for the NAC as the access code, per spec
([#698](https://github.com/MattCheramie/GopherTrunk/issues/698)).

## Neighbor lists attach to the camped site

The Adjacent Status Broadcast (opcode `0x3C`) carries a [neighbor site](/reference/neighbor-site/)
list, and the list belongs to the site that broadcast it — the one you are camped on, not the
neighbor. Each entry's `downlink_hz` is the *neighbor's* control-channel frequency, resolved
through the Identifier Update band plan. This enables passive backfill: sites you have never
directly decoded get their CC frequencies filled in from a neighbor's advertisements — useful
for [roaming](/reference/roaming/) ([#864](https://github.com/MattCheramie/GopherTrunk/issues/864)).

## Hybrid systems: Phase 1 control channel, Phase 2 traffic

Many networks run a Phase 1 FDMA control channel granting Phase 2 TDMA voice channels. Three
traps, all hit in the field:

1. **`OpIdentifierUpdateTDMA` (`0x33`) must be dispatched.** GopherTrunk once handled only the
   FDMA identifier variants (`0x34`, `0x3D`), so grants referencing a TDMA channel ID were
   black-holed with no band plan. A TDMA channel ID is typically the ×2 twin of an FDMA ID; the
   frequency-field bit packing matches the VHF/UHF variant
   ([#345](https://github.com/MattCheramie/GopherTrunk/issues/345)).
2. **TDMA-ness must flow into the grant.** The Identifier Update TDMA parse sets a per-channel
   flag; the band plan's `IsTDMA(channel_id)` decides whether a grant is Phase 2. Until that
   plumbing existed, every grant was tagged plain `p25` and the Phase 2 voice chain was dead
   code ([#376](https://github.com/MattCheramie/GopherTrunk/issues/376)).
3. **Real on-air Phase 2 is always PN44-scrambled**, with the seed derived from
   (WACN, system ID, NAC). GopherTrunk's `p25_phase2_scrambler_mode` defaults to `off` for
   fixture compatibility — leaving it there on a live system decodes nothing
   ([#376](https://github.com/MattCheramie/GopherTrunk/issues/376)).

## Symptom table

| Symptom | Looks like | Actually | Fix / check |
|---|---|---|---|
| One PTT appears under several site IDs at once | Duplicate or corrupted grants | Wide-area talkgroup granted on every participating site | Expected; use registration/affiliation for location |
| Two sites treated as one | Dedup bug | Keyed on NAC, which repeats across sites | Key on `(rfss_id, site_id)` |
| System panel stuck "Awaiting status broadcasts" | No 0x3A/0x3B on air | Endpoint served static config, never the decoded snapshot | Read identity from the live site tracker ([#673](https://github.com/MattCheramie/GopherTrunk/issues/673)) |
| Grants black-holed with no matching band plan | Site missing Identifier Updates | TDMA identifier variant (`0x33`) not dispatched | Handle all Identifier Update opcodes ([#345](https://github.com/MattCheramie/GopherTrunk/issues/345)) |
| Phase 2 voice silent on a Phase-1-CC system | RF or vocoder problem | Grants never tagged Phase 2, or scrambler left `off` (on-air is always PN44) | Wire `IsTDMA` into grants; enable PN44 ([#376](https://github.com/MattCheramie/GopherTrunk/issues/376)) |

One more consequence worth internalizing: because site identity is only known *after* a
control channel decodes, anything that must be chosen *before* lock — such as the
[demodulator mode](/reference/p25-demod-mode-selection/) — cannot be keyed by RFSS/site and
has to be keyed by frequency instead.

## Provenance

- [#698](https://github.com/MattCheramie/GopherTrunk/issues/698) — exposing P25 site identity in grants and the API; grant-vs-location, NAC reuse, keying rules.
- [#673](https://github.com/MattCheramie/GopherTrunk/issues/673) — System Information panel stuck because network identity was read from config, not the air.
- [#864](https://github.com/MattCheramie/GopherTrunk/issues/864) — neighbor-site lists (opcode 0x3C) attach to the camped site; passive CC-frequency backfill.
- [#376](https://github.com/MattCheramie/GopherTrunk/issues/376) — hybrid Phase 1 CC / Phase 2 traffic plumbing and the always-PN44 scrambler gotcha.
- [#345](https://github.com/MattCheramie/GopherTrunk/issues/345) — the undispatched TDMA Identifier Update that black-holed grants.
