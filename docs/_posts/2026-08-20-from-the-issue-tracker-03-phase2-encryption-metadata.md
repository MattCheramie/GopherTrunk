---
title: "From the Issue Tracker, Part 3: Encrypted, Says Who — Four Layers Between a Flag and Its Metadata"
description: A P25 Phase 2 system flagged every encrypted call correctly but never reported which algorithm or key — because a fictional MAC opcode, a missing carrier loop, a 48-bit sync constant in a 40-bit field, and a swapped dibit map were stacked on top of each other.
category: solution-postmortem
keywords: p25 phase 2, encryption metadata, algorithm id, key id, mac_ptt, superframe sync, dibit remap, hdqpsk, unconditional census, tia-102, gophertrunk postmortem
tags: [from-the-issue-tracker, p25, phase2, encryption, debugging, postmortem]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "From the Issue Tracker"
series_part: 3
---

*Part 3 of **From the Issue Tracker**, postmortems of GopherTrunk bugs that fought
back. [Part 2]({{ '/blog/solution-postmortem/from-the-issue-tracker-02-talker-alias-hunt/' | relative_url }})
chased a talker alias through cipher land. This one is about a field that was
always half right: every encrypted P25 Phase 2 call got flagged `encrypted: true`,
and not one of them ever said *what kind* of encrypted. The answer turned out to be
four independent defects deep — and the tool that finally ordered them was a log
line that fires even when there is nothing to say.*

> **TL;DR:** On a live P25 Phase 2 system, 66 of 66 encrypted calls were flagged
> `encrypted: true` while `algorithm_id`/`key_id` never populated
> ([#813](https://github.com/MattCheramie/GopherTrunk/issues/813)). The flag and
> the metadata come from different places, so one can be right while the other is
> structurally impossible. Underneath sat a four-stage chain: an encryption-sync
> MAC opcode (`0x70`) that doesn't exist on real air, a plausible-but-wrong
> carrier-recovery diagnosis disproven by a TCXO source failing identically, a
> 48-bit sync-word constant silently truncated into a 40-bit field, and a missing
> 2↔3 dibit remap that let superframes lock while every payload decoded to
> garbage. The technique that cracked the ordering was an **unconditional per-call
> census** that logs stage counters even at zero — and the last line of defense is
> a validity gate that refuses to publish an algorithm ID the standard has never
> heard of.

## The symptom as reported

The reporter's setup was deliberately clean: an Airspy on the control channel, a
dedicated RTL-SDR voice follower, `recordings.skip_encrypted: false` so encrypted
calls are actually followed, and a generous 5-second metadata window. Thirty
minutes of live traffic, then straight to the API rather than the logs:

```json
{"id": 9367, "system": "MMR", "protocol": "p25-phase2", "group_id": 3202,
 "frequency_hz": 468612500, "encrypted": true,
 "end_reason": "encrypted", "talkgroup_alpha": "22-02 WD2"}
```

66 calls flagged `encrypted: true`. Zero with an `algorithm_id` or `key_id` key at
all — `omitempty` was hiding zero values. The reporter had already ruled out the
obvious: not the follow window (1500 → 5000 ms changed nothing), not the recorder
short-circuiting (confirmed via log lines that calls were followed), not a logging
gap (the API is the stored value).

Why can the flag be right while the metadata never arrives? Because they are
unrelated. `encrypted: true` comes from the **grant's ServiceOptions "protected"
bit** on the control channel. The algorithm and key IDs must be recovered from the
**voice channel's MAC layer**. One path worked perfectly; the other, it turned
out, had never worked at all.

## Stage zero: the opcode that never existed

The extraction code existed and was unit-tested: a MAC PDU handler keyed on
`OpEncryptionSync = 0x70`, wired all the way through to the engine. Every test
passed — because every test *synthesized* a `0x70` PDU. The constant was an
explicit working model; the source even carried a note that the relevant spec PDF
wasn't available. On real Phase 2 (TIA-102.BBAC), ALGID/KID/MI don't ride a
standalone opcode: they ride the **MAC_PTT** message that begins each
transmission, identified by *slot type*, not by MAC opcode. `AsEncryptionSync`
could never match anything on air.

That explained the shape of the symptom — flagged but never populated — and
produced fix number one: parse MAC_PTT at the documented offsets. It did nothing,
which is how the real diagnostic entered the story.

## The census: a log line that fires at zero

The first instrumentation pass added detail to the existing
`composer: p25p2 mac pdu` line. The reporter ran ~19 minutes of live traffic: 202
voice chains started with valid configuration, 33 encrypted calls — and **zero**
of those log lines, for any slot type.

Here is the trap: that line only fires on a *successful* MAC decode. Its silence
is identical whether superframe sync never locked, the ISCH never classified a MAC
slot, or the MAC FEC failed every time. **The silence of a success-only log line
carries no diagnostic information.**

The replacement was an unconditional per-call census — one line at the end of
every call, even when every counter is zero:

```
composer: p25p2 call census serial=… system=… superframes=N \
    voice_subframes=N mac_subframes=N mac_pdus=N  slot_Voice4V=… slot_Unknown=…
```

Read as a three-way stage disambiguator:

| Census reading | Failing stage |
|---|---|
| `superframes=0` | Upstream of MAC entirely — superframe sync never locks |
| `superframes>0`, `mac_subframes=0` | Sync locks but ISCH never yields a MAC slot |
| `mac_subframes>0`, `mac_pdus=0` | MAC FEC chain fails every slot — *now* byte layouts matter |

The next run returned `superframes=0` on **67 of 67 calls**. Not one superframe
ever locked, encrypted or not — on the same dongle that recorded Phase 1 voice
cleanly the same night. The MAC_PTT byte-offset question was moot; the failure was
upstream of MAC entirely.

## The plausible wrong theory: carrier recovery

The Phase 2 H-DQPSK receiver was `MatchedFilter → Gardner timing → differential
decode` — no NCO, no AGC, no Costas loop. A differential decoder cancels a
*constant* carrier phase but not the per-symbol rotation `2π·Δf/baud` left by a
real tuner's frequency offset. At 6000 baud, ~750 Hz of offset rotates every
symbol a full π/4 into the wrong quadrant. And the hardware split matched
perfectly: the Airspy control channel (TCXO, low offset) decoded; the RTL-SDR
voice follower (no TCXO) never locked. It was even the same bug class already
fixed twice on sibling paths — Phase 1 C4FM got coarse AFC in
[#275](https://github.com/MattCheramie/GopherTrunk/issues/275), Phase 1 CQPSK got
an NCO seed and Costas loop in
[#492](https://github.com/MattCheramie/GopherTrunk/issues/492). Every Phase 2
test had synthesized zero-offset IQ, so CI never noticed.

Carrier recovery was added — coarse seed, NCO, AGC, a rotation-aware Costas loop —
and the synthetic stream went from ~72% symbol errors at 1500 Hz offset to 0%
across ±5 kHz. A genuine defect, genuinely fixed.

The field result: `superframes=0`, 56 of 56 calls. Unchanged.

The clincher came later, unprompted: the reporter re-ran with the **Airspy R2 —
a TCXO source — as the voice follower** and got `superframes=0` on 20 of 20
unencrypted follows too. A low-offset source failing *identically* to a
high-offset one rules out carrier recovery entirely, and rules out hardware with
it. Whatever was broken was hardware-independent.

## Root cause one: a 48-bit constant in a 40-bit field

```go
// internal/radio/p25/phase2/sync.go — before
OutboundSyncHex uint64 = 0x575F7DFF77FF  // 48 bits…
SyncDibits             = 20              // …into a 20-dibit (40-bit) field
```

`hexToDibits` silently used only the low 40 bits, so the correlator hunted for
`0x5F7DFF77FF` — neither the standard sync word nor anything that has ever been
transmitted. No superframe could lock, on any tuner, ever. And every round-trip
test passed, because the test encoder injected sync **from the same wrong
constant** it decoded with. A self-consistent fiction.

The authoritative P25 Phase 2 outbound sync is `0x575D57F7FF` (cross-checked
against OP25's `frame_sync_magics.h` and SDRtrunk, per TIA-102.BBAC). The
regression test that pinned it synthesizes the sync *independently of the
project's own modulator* — the only kind of test that can expose a bad shared
constant: zero locks on the old value, locks on the correct one.

## Root cause two: the 2↔3 remap

Fixing the sync constant surfaced the fourth layer. The shared DQPSK quadrant
slicer assigns the two negative-phase symbols the dibit values 3 and 2 where
TIA-102 says 2 and 3. The Phase 1 CQPSK path — verified on real air in #492 —
already documents and corrects exactly this transposition:

```go
// internal/radio/p25/phase1/receiver/cqpsk.go
var lsmDibitRemap = [4]uint8{0, 1, 3, 2} // swaps 2↔3 → canonical TIA-102
```

Phase 2 was missing the equivalent remap. The swap is its own inverse, which made
for a satisfying sanity check: applying `[0,1,3,2]` to the transposed sync
`0x565956A6AA` yields exactly the authoritative `0x575D57F7FF`, and vice versa.
The failure signature is distinctive and worth remembering: **superframes lock,
but every payload decodes to garbage** — the regression test reproduces the field
symptom precisely, recovering `alg=0x75 key=0x555d` where `0x84/0x1234` was
encoded. The fix canonicalizes the receiver's dibit output right after the slicer
(one point covers sync, ISCH, MAC FEC, and diagnostics together) and restores the
standard sync constant, leaving the shared demodulator — used by TETRA and Phase 1
CQPSK — untouched.

## The numbers, and the gate

A 7-day quantification from the completed-call webhook made the before/after
brutal and the residual honest:

| Path | Encrypted calls | Valid algorithm ID resolved | Rate |
|---|---|---|---|
| P25 Phase 1 | 2,739 | 2,432 (AES-256) | **89%** |
| P25 Phase 2 | 3,107 | 15 | **0.5%** |

Worse than "omitted": the Phase 2 fields were populating with **bit-error
values** — a uniform algorithm-ID smear across 0x00–0xFF, a different key ID
every call. A wrong value published confidently is worse than an absent one.

The mitigation is a validity gate, `p25.AlgorithmKnown(id)`, checked against the
TIA-102 algorithm registry (0x80, 0x81, 0x83, 0x84, 0x85, 0x86, 0x89, 0x9F, 0xAA)
and applied **at the composer** — the single point that both the recorder
(webhooks, call history) and the engine (SSE, TUI) draw from, so every consumer is
covered at once. An out-of-set value is provably a mis-decode and is dropped; the
fields stay absent rather than lying. One deliberate scope call: the gate does
*not* whitelist the classified Type-1 block (0x00–0x41) — admitting 66 low values
would let a large slice of the smear straight through, and a real Type-1 sighting
is a one-line registry addition.

One process footnote: the reporter's raw IQ capture sat as a release asset on a
fork, which the project's offline-replay tooling couldn't fetch — so every fix in
this chain had to be verified against spec-conformant synthetics and code-internal
cross-checks against the on-air-verified Phase 1 CQPSK path instead of the
reporter's own air. Host captures somewhere plain; it matters.

## What we keep

- **A flag and its metadata can have different truth values.** `encrypted: true`
  came from the grant; the algorithm ID needed the voice channel's MAC layer. See
  [encrypted call handling]({{ '/reference/encrypted-call-handling/' | relative_url }})
  for how the two paths relate.
- **Silence of a success-only log line carries no information.** The unconditional
  per-call census — counters logged even at zero — is the single most transferable
  technique in this story, and it's now in the
  [diagnostic playbook]({{ '/reference/diagnostic-playbook/' | relative_url }}).
- **A self-consistent synthetic proves nothing about air.** The truncated sync
  constant survived every round-trip test because encoder and decoder shared the
  fiction. Regression tests for on-air constants must synthesize independently —
  the pinned values live in
  [P25 on-air constants]({{ '/reference/p25-onair-constants/' | relative_url }}).
- **A wrong theory can be a real bug.** The carrier-recovery work was correct and
  necessary — it just wasn't *this* bug. The disproof (a TCXO source failing
  identically) is as valuable as the fix.
- **Never publish a value you can't validate.** The registry gate turns a bit-error
  smear back into honest absence, at the one choke point every consumer shares.
