---
slug: dmr-grant-field-gotchas
title: DMR grant-field gotchas
entry_type: term
category: fn-protocol
description: "DMR grant-field gotchas are field-verified corrections to common DMR Tier III grant-parsing assumptions — the 12-bit LPCN layout, data grants that must not be voice-followed, the non-CSBK reverse channel, and sync words that cannot reveal spectrum inversion."
keywords: dmr, tier iii, csbk, channel grant, lpcn, lcn, logical channel number, pd_grant, td_grant, btv_grant, mbc, reverse channel, sync word, spectrum inversion, hytera xpt, etsi ts 102 361-4
aka: [DMR CSBK grant parsing, LPCN vs LCN, Tier III grant traps]
infobox:
  - { label: Type, value: Protocol gotchas }
  - { label: Applies to, value: DMR Tier III channel-grant CSBKs }
  - { label: Key rule, value: "The grant leads with a 12-bit LPCN — there is no trailing 7-bit LCN" }
  - { label: Blind spot, value: Sync words cannot detect spectrum inversion }
see_also: [dmr-tier-3, dmr-csbk-payloads, dmr-sync-patterns, dmr-full-link-control, dibit, p25-onair-constants, tetra-lock-facts, signal-signatures]
related_reading:
  - { title: "From the Issue Tracker, Part 20: The Self-Consistent Trap — Round-Trip Tests That Validate Their Own Bugs", url: /blog/solution-postmortem/from-the-issue-tracker-20-self-consistent-trap/ }
cite_urls:
  - https://github.com/MattCheramie/GopherTrunk/issues/639
  - https://github.com/MattCheramie/GopherTrunk/issues/626
  - https://github.com/MattCheramie/GopherTrunk/issues/264
---

**DMR grant-field gotchas** are the places where a reasonable reading of a
[DMR Tier III](/reference/dmr-tier-3/) channel-grant [CSBK](/reference/dmr-csbk-payloads/)
is wrong — wrong field boundaries, wrong assumptions about what the grant carries, and wrong
expectations about what the sync layer can tell you. Each item below produced a confusing
field symptom in GopherTrunk before the correct reading was pinned down.

## The grant leads with a 12-bit LPCN

Per ETSI TS 102 361-4, a Tier III channel grant's payload leads with a **12-bit Logical
Physical Channel Number (LPCN)**, followed by a timeslot bit, then the target and source
addresses. It does *not* end with a 7-bit LCN.

GopherTrunk originally read the "LCN" from the last payload octet — which is actually the low
byte of the 24-bit **source radio address**. The result was a memorable symptom on a Hytera
XPT system: the LCN appeared to change on every transmission, because it changed whenever a
different radio keyed up. It looked like channel-plan chaos; it was radio identity leaking
into a channel field, and the LCN autoconfig learner could never converge because every call
introduced a "new" channel. The fix widened the LCN from 7 to 12 bits end to end — parser,
band-plan resolver, learner, config, and events
([#639](https://github.com/MattCheramie/GopherTrunk/issues/639)).

## Encrypted and Emergency are not in the grant

The ETSI DMR channel-grant CSBK does not carry Encrypted or Emergency flags. Those arrive in
the voice [Link Control](/reference/dmr-full-link-control/) once the call is up. A parser
that "finds" them in the grant is reading other fields
([#639](https://github.com/MattCheramie/GopherTrunk/issues/639)).

## Not every grant should be followed

Three grant opcodes look similar and must be treated differently
([#626](https://github.com/MattCheramie/GopherTrunk/issues/626)):

- **`BTV_GRANT` (`0x32`)** is broadcast voice — follow it.
- **`PD_GRANT` (`0x33`) and `TD_GRANT` (`0x34`)** are **data** grants. GopherTrunk
  deliberately does not follow them: retuning a voice receiver onto a packet channel just
  produces garbage audio. They are still *observed*, because data grants are a perfectly good
  source of LPCN-to-frequency learning.
- **MBC (Multi Block Control) can carry grants.** On systems using a Flexible (rather than
  Fixed) channel plan, grants arrive as MBC header + continuation blocks; dropping MBC
  entirely drops those calls.

## The reverse channel is not a CSBK

Inbound/outbound reverse-channel signalling does not ride a CSBK at all. It lives in the
embedded signalling field of a normal DMR burst, or in a shorter burst with its own
[sync pattern](/reference/dmr-sync-patterns/) (MS Sourced RC sync). Searching CSBK opcodes
for it finds nothing ([#626](https://github.com/MattCheramie/GopherTrunk/issues/626)).

## Sync words cannot detect spectrum inversion

DMR's nine sync words are **closed under the polarity flip** `(dibit + 2) mod 4` — the
XOR-`0xAAAA…` pattern a spectrum-inverted capture produces. An inverted *data* sync is
byte-identical to a clean *voice* sync, so the sync correlator alone can never tell you the
spectrum is inverted; it will happily lock and hand every downstream stage complemented
[dibits](/reference/dibit/). Only the FEC-protected payload can detect inversion — persistent
CRC/FEC failure with a solid sync lock is the tell (compare the TETRA equivalent in
[TETRA lock facts](/reference/tetra-lock-facts/), and the general catalog in
[signal signatures](/reference/signal-signatures/))
([#264](https://github.com/MattCheramie/GopherTrunk/issues/264)).

## Symptom table

| Symptom | Looks like | Actually | Fix / check |
|---|---|---|---|
| LCN changes on every transmission; learner never converges | Channel-plan chaos, rekeying | Last octet is the source radio's low byte; the LPCN is the first 12 bits | Parse per ETSI TS 102 361-4 ([#639](https://github.com/MattCheramie/GopherTrunk/issues/639)) |
| Encrypted/Emergency never set at grant time | Parser bug | Not present in the grant CSBK | Read them from voice Link Control ([#639](https://github.com/MattCheramie/GopherTrunk/issues/639)) |
| Scanner "ignores" PD/TD grants | Missing feature | Data grants; following them yields garbage audio | Deliberate — still mined for LPCN learning ([#626](https://github.com/MattCheramie/GopherTrunk/issues/626)) |
| Calls missing on a Flexible-plan system | Weak signal | Grants ride MBC header + continuations | Decode MBC ([#626](https://github.com/MattCheramie/GopherTrunk/issues/626)) |
| Solid sync lock, but every payload fails FEC/CRC | Wrong colour code, encryption | Possible spectrum inversion — sync words are closed under the flip | Check FEC pass rate with `iq_invert` toggled ([#264](https://github.com/MattCheramie/GopherTrunk/issues/264)) |

## Provenance

- [#639](https://github.com/MattCheramie/GopherTrunk/issues/639) — Hytera XPT "LCN changes every call": the 12-bit LPCN layout and the source-address misread.
- [#626](https://github.com/MattCheramie/GopherTrunk/issues/626) — DMR data-type coverage: PD/TD vs BTV grants, MBC grants on Flexible plans, and the non-CSBK reverse channel.
- [#264](https://github.com/MattCheramie/GopherTrunk/issues/264) — RTL-SDR Blog V4 investigation that established the sync-word closure under polarity flip.
