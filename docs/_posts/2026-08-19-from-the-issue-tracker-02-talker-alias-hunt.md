---
title: "From the Issue Tracker, Part 2: The Talker-Alias Hunt — Three Wrong Transports and an Architectural Gate"
description: A month of chasing blank P25 talker aliases through three wrong transport theories, until a per-opcode census and SDRTrunk ground truth revealed the aliases were riding traffic-channel signalling — and GopherTrunk's decoder was architecturally gated behind winning a voice tuner.
category: solution-postmortem
keywords: p25 talker alias, radio id, motorola alias, facch-s, tdulc link control, mac pdu census, unhandled tsbk, sdrtrunk ground truth, signalling follower, sigfollow, gophertrunk postmortem
tags: [from-the-issue-tracker, p25, phase2, metadata, debugging, postmortem]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "From the Issue Tracker"
series_part: 2
---

*Part 2 of **From the Issue Tracker**, postmortems of GopherTrunk bugs that fought
back. [Part 1]({{ '/blog/solution-postmortem/from-the-issue-tracker-01-first-p25-lock/' | relative_url }})
ended with the first control-channel lock and a warning about self-consistent tests.
This part picks up two days later, on the same system, with a subtler kind of bug:
everything decoding, nothing wrong in any log — and a field the reporter could see
in another program, blank in ours, for a month
([#376](https://github.com/MattCheramie/GopherTrunk/issues/376)).*

> **TL;DR:** Radio IDs populated fine, but the talker alias stayed blank for every
> radio — while SDRTrunk showed aliases on the same system, even on encrypted calls.
> Three successive transport theories (standard voice LCOs, a control-channel vendor
> TSBK, a speculative Phase 2 opcode) each shipped a decoder for a message format
> that doesn't exist on air. A per-(opcode, MFID) census with raw payload dumps
> proved the alias was *not* on the control channel; SDRTrunk's decode logs then
> pinned the two real transports. The final root cause was architectural: alias
> decode lived inside the voice composer, so it only ran when a call won a voice
> tuner — which, on a busy system, almost never happens. The fix was a
> signalling-only follower that decodes traffic-channel signalling without
> following the call as voice.

## The report

The issue opened modestly: source RIDs were showing up in grant payloads
(`TG 20202 ← 207545`), could radios become first-class entities with alias support?
The entity work shipped quickly — an alias table, `/api/v1/rids` endpoints, a
Radio IDs panel. Then the field note that turned a feature request into a
month-long hunt:

> SDRTrunk currently displays Talker Alias on MMR, including encrypted Ambulance
> talkgroups.

The aliases were on the air. RID tracking confirmed the plumbing worked — one radio
showed 18 observed calls with its last talkgroup — but `talker_alias` stayed blank
for every radio, run after run, build after build. Something was transmitting
aliases that GopherTrunk structurally could not see.

## Three wrong transports

The hunt shipped three alias decoders before discovering none of them matched the
air.

**Theory 1: the standard voice-channel LCOs.** TIA-102.AABF defines talker-alias
link-control opcodes 0x15/0x16/0x17 on the voice channel. A reassembler for the
standard form landed and produced nothing — the field test showed RIDs populating
and aliases blank. Motorola systems don't use the standard form.

**Theory 2: the Motorola voice-LC form — plus a control-channel working model.** The
next round implemented Motorola's vendor variant (LCO 0x15 header + 0x17 data
blocks, MFID 0x90) on the LDU1 voice path, alongside a working model that assumed a
plain-ASCII vendor TSBK on the control channel. Zero fragments were ever observed —
including on the clean site where decode quality was excellent, which mattered: a
missing message on a clean channel is evidence about the *transport*, not the RF.

**Theory 3: a speculative Phase 2 opcode.** The Phase 2 MAC path carried an alias
placeholder keyed to opcode 0x82 with plain-ASCII payloads — a guess, and another
decoder for a message nobody transmits.

Each theory was plausible, each shipped with passing tests, and each was falsified
only by field data. The common flaw: all three described what the alias *might* look
like, with no captured bytes saying what it *did* look like.

## Detours that paid rent

Two side discoveries along the way were real bugs in their own right:

- **`OpMACPTT` was a fiction.** The Phase 2 MAC dispatcher named opcode 0x01 "MAC
  PTT" — a constant that exists in no spec. Opcode 0x01 is
  `GROUP_VOICE_CHANNEL_USER_ABBREVIATED`, the in-call broadcast carrying the source
  RID and service options on the traffic channel. Real PDUs were being parsed as
  "MAC PTT" and discarded, which is why encrypted-talkgroup grants surfaced as
  `src=0 enc=false` while SDRTrunk showed both correctly. Fixing it required a
  `KindCallSourceUpdate` backfill path so metadata arriving mid-call patches the
  active call.
- **The Phase 2 alias path was dead code on hybrid systems.** MMR runs a Phase 1
  control channel with Phase 2 TDMA traffic channels — the most common P25 hybrid
  layout. The Phase 1 CC tagged every grant `Protocol="p25"`, so the composer never
  entered the Phase 2 voice chain at all: none of the newly added MAC dispatch ever
  executed. The fix propagates an `AccessTDMA` flag from the TDMA identifier update
  through the band plan, so TDMA-channel grants publish as `p25-phase2` with their
  scrambler seed derived from (WACN, SystemID, NAC).

Both were found because the reporter grepped for the promised log lines and reported
their *absence* — a null result stated precisely enough to act on.

## The diagnostic that cracked it: a census, then the bytes

After three wrong transports, the next step was deliberately not a fourth theory. It
was instrumentation: one Info line per distinct `(opcode, MFID)` pair the control
channel emits that GopherTrunk doesn't dispatch —

```text
p25: unhandled tsbk mfid=144 opcode=Opcode(16) nac=353
p25: unhandled tsbk mfid=0 opcode=Opcode(30) nac=353
```

— followed, one round later, by raw payload dumps capped at 8 samples per pair:

```text
p25: unhandled tsbk payload mfid=144 opcode=0x16 lb=false payload=c23f3040ffff0000 nac=353
```

Two implementation details mattered more than they look. First, the census prints
the **numeric** opcode, because Go's stringer had been mislabeling vendor opcodes
with standard-namespace names — Motorola opcode 0x15 printed as
`UnitToUnitAnswerRequest`, sending everyone reading the logs down the wrong row of
the wrong table. Second, the payloads could be cross-checked against ground truth:
SDRTrunk supplied known RID↔alias pairs (`202438 MBR02 2438`, `206806 AGP 6806`,
`203530 ZR5 3530`), and the Motorola alias format embeds the 24-bit RID — so
`printf '%06x'` turned each known radio into a grep pattern (`202438` → `031ad6`).

The census answered the question definitively, in both directions:

- **No alias anywhere on the control channel.** No payload contained a known RID or
  alias ASCII. A control-channel `mfid=0 opcode=0x15` hit turned out to be
  `SNDCP_DAT_PAGE_REQ` — same number as the alias LCO, unrelated namespace, pure red
  herring.
- **Two of the unknown opcodes were decodable wins.** Motorola MFID 0x90 opcodes
  0x02 and 0x03 decoded as **patch-group voice grants** — e.g. payload
  `401001006904e2cc` carried the encryption bit, a channel, super-group 105, and
  source RID 320204. GopherTrunk had been silently dropping those calls; wiring them
  through the grant path (PR #692) fixed `src=0 enc=false` on patch calls and made
  the super-group traffic followable.

A diagnostic built to find one thing found two others and ruled out a theory — the
hallmark of instrumenting instead of guessing.

## Ground truth from another decoder

With the control channel exonerated, the reporter went to SDRTrunk's decode logs on
the same system — roughly 600 alias-bearing traffic-channel messages per hour — and
came back with the actual bytes of both real transports:

```text
NAC:361/x169 TDULC MOTOROLA TALKER ALIAS HEADER TG:30583 SEQUENCE:9 BLOCKS TO FOLLOW:7
    FORMAT:1-UNICODE MSG:159077770701009902
TS1 FACCH-S HANGTIME MOTOROLA TALKER ALIAS HEADER TG:20208 RADIO:ISSI 781824.356.200062
    SEQUENCE:0 BLOCKS TO FOLLOW:2 FORMAT:1-UNICODE MSG:9190114EF002010006BEE00164030D7E24CC8
```

| | Phase 1 | Phase 2 |
|---|---|---|
| Carrier | **TDULC** (terminator link control) — not LDU1 | **FACCH-S** during call hangtime — not a TSBK |
| Framing | MFID 0x90, LCO 0x15 header + N× 0x17 blocks | header bytes `91 90 11 …`, data blocks `95 90 11 …` |
| RID binding | header carries the talkgroup only | header carries the source RID inline — self-contained |
| Payload | Motorola substitution cipher, UTF-16 BE, CRC-16 | same cipher, framed `WACN(20) · System(12) · RadioID(24) · alias · CRC-16` |

Two details explain months of confusion. The Phase 1 alias rides the *terminator* —
a decoder that only scans LDU1 link control returns before ever seeing it. And the
consistent `BEE00164…` prefix in every fragment is not noise: it's the system's own
identity (WACN BEE00, SysID 164) leading the encoded frame. The aliases decode on
encrypted calls because the alias link control sits entirely outside the AES voice
payload.

## The real root cause was architectural

Even with the correct transports known, one more test run produced zero aliases —
voice chains started, calls followed, nothing else. The reporter then read the
source and found the actual root cause, and it wasn't a parser:

**Alias decode lived inside the voice composer.** The FACCH-S dispatch ran only when
a grant was followed to a voice tuner and the Phase 2 voice chain spun up. On a busy
multi-site system that gate almost never opens:

- most grants get `no voice device available` — the voice pool can't cover the
  system's full frequency spread, so the composer never runs for them;
- encrypted calls are torn down at `reason=encrypted` the moment encryption is
  detected — *before hangtime, which is exactly when the alias is transmitted*;
- and even clear, fully followed calls surfaced no FACCH signalling decode.

SDRTrunk harvests aliases from the traffic channel's *signalling* stream, without
following the call as voice — which is why it collects them across a busy system
with two dongles, encrypted or not. GopherTrunk had welded metadata extraction to
media capture, so the metadata inherited every constraint of recording: tuner
availability, encryption policy, call priority. The blank column was never a decode
bug. It was a dependency graph bug.

## The fix: a signalling-only follower

PR #762 landed `internal/sigfollow`: a follower that subscribes to the event bus
and, for each in-window Phase 2 grant, opens a **signalling-only DDC tap** on the
wideband IQ broker — decoding FACCH-S MAC PDUs and publishing aliases with no voice
tuner, no audio path, and no interaction with the encrypted-call teardown.

The design points that made it stick:

- **One dispatcher, two callers.** The FACCH-S MAC dispatch (alias reassembly plus
  the per-PDU census) became a shared `MACDispatcher` used by both the voice
  composer and the follower, so the two paths cannot drift apart on what they
  decode.
- **Opt-in and bounded.** `signalling_taps: N` per `role: wideband` device (0
  disables; 2–4 suits a busy multi-site system); each tap is an independent DDC, so
  cost scales linearly and visibly.
- **No double work.** Per-frequency dedupe plus a voice-call skip mean a channel
  already being recorded isn't followed twice.
- **Fail-safe by default.** The FACCH-S bit packing was still unverified against
  captures when the follower shipped, so the decoder publishes nothing unless the
  frame yields a coherent radio identity — a wrong offset produces silence, not a
  garbled alias bound to a real radio.

Decoded aliases bind onto the RID through the existing affiliation tracker, so the
column the issue opened about fills in with no further wiring.

## What we keep

- **Never gate metadata extraction behind media capture.** Identity, encryption
  state, and aliases ride signalling; if decoding them requires winning a voice
  tuner, a busy system guarantees you lose them. The follower pattern and the
  encrypted-hangtime interaction are written up in
  [encrypted call handling]({{ '/reference/encrypted-call-handling/' | relative_url }}).
- **When theories keep missing, stop theorizing and take a census.** One log line
  per unhandled `(opcode, MFID)` plus capped payload dumps falsified a theory,
  found two droppable-call bugs, and named the next step — the pattern is in the
  [diagnostic playbook]({{ '/reference/diagnostic-playbook/' | relative_url }}).
- **Log numeric opcodes.** A stringer that maps vendor opcodes onto
  standard-namespace names is actively misleading; `0x15` meant three different
  things in this one thread.
- **Another decoder's output is legitimate ground truth.** SDRTrunk's decode logs
  pinned both real transports — TDULC and FACCH-S — after three GopherTrunk
  decoders had been written for formats that don't exist on air. The
  system-identity fields that anchored the fragments (`WACN | SysID | RID`) are
  covered in
  [P25 site identity semantics]({{ '/reference/p25-site-identity-semantics/' | relative_url }}).
- **A precise null result is a contribution.** "The promised log line did not fire"
  found dead code twice. Grep patterns belong in the fix announcement, so the field
  tester can report absence as sharply as presence.

## Series navigation

**Part 2** · ←
[Part 1: The First P25 Lock]({{ '/blog/solution-postmortem/from-the-issue-tracker-01-first-p25-lock/' | relative_url }})
