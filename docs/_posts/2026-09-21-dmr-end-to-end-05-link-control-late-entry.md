---
title: "DMR End to End, Part 5: Link Control, Embedded LC & Late Entry"
description: How a DMR call names itself three times over — the 72-bit Full Link Control in the Voice LC Header, the Terminator with LC, and the fragments embedded in voice bursts B–E — and how GopherTrunk turns the embedded copy into late entry, a re-key anchor, and a talker alias.
category: deep-dives
keywords: dmr full link control, dmr embedded lc, dmr voice lc header, dmr late entry, dmr emb lcss, dmr talker alias, flco group voice, dmr terminator with lc, embedded signalling bptc 128 72, gophertrunk dmr link control
tags: [dmr-end-to-end, dmr, link-control, late-entry, embedded-signalling, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "DMR End to End"
series_part: 5
---

*Part 5 of **DMR End to End**, a 14-part deep dive that follows the world's
most widely deployed digital PMR protocol through GopherTrunk — from a 4FSK
carrier to two simultaneous recorded calls, direct-mode handhelds, and
decrypted Enhanced Privacy voice.
[Part 4]({{ '/blog/deep-dives/dmr-end-to-end-04-fec-stack-forged-terminator/' | relative_url }})
built the FEC stack. This part is about what the protected bits *say*: the
Full Link Control word that names a call, the three places DMR puts it, and
late entry — the mechanism every subscriber radio uses and the scanner once
ignored.*

> **TL;DR:** A DMR call is named by a 72-bit **Full Link Control** PDU
> (`internal/radio/dmr/flc.go`: PF, 6-bit FLCO, FID, service options, 24-bit
> destination, 24-bit source). It is sent whole in the **Voice LC Header**
> (slot type `0x1`, RS(12,9) seed `0x96`) and the **Terminator with LC**
> (`0x2`, seed `0x99`), and again as four 32-bit fragments **embedded** in
> voice bursts B–E, framed by a 16-bit EMB (`emb.go`) and reassembled under
> BPTC(128,72) + a 5-bit checksum. The conventional path used to grant only
> on headers — one chance per PTT. `ingestVoiceSuperframe`
> (`tier2/conventional.go`) now grants a header-less transmission from **two
> agreeing CRC-valid embedded LCs** (`lateEntryConfirm = 2`, ~720 ms in),
> gated by `endedAtDibit` so a dead call's closing superframes cannot
> resurrect it. `GT_DMR_DROP_HEADERS=1` scrubs every header from a real
> capture and 5/5 transmissions still grant. The same embedded LC carries the
> **talker alias** (`talker_alias.go`).

**Key takeaways**

- **DMR names a call three times so a receiver can join it late.** Header,
  terminator and the embedded copy in every superframe carry the same FLC.
- **Late entry is confirm-twice, never confirm-once.** One CRC-valid embedded
  LC can be a miscorrection; two agreeing copies within 3 s grant *and*
  declare the lock.
- **Stream order is not arrival order.** The superframe assembler runs a span
  behind the burst slicer, so a call's closing LCs surface after its
  terminator — `endedAtDibit` stops them re-granting it.
- **The header is a train, not a burst.** The re-key anchor follows the
  *last* copy, or every keyup opens with a phantom release
  (`TestConventionalHeaderTrainIsOneKeyup`).

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| FLC parse | 9 octets → PF / FLCO / FID / options / dst / src | `internal/radio/dmr/flc.go` (`ParseFLC`, `AsGroupVoiceUser`) |
| Header / terminator | BPTC(196,96) + RS(12,9), seed `0x96` / `0x99` | `tier2/conventional.go` (`handleVoiceHeader`, `terminatorDest`) |
| Embedded LC | EMB split, 4 × 32 → BPTC(128,72) + checksum → FLC | `internal/radio/dmr/emb.go` (`SplitEmbeddedField`, `ReassembleEmbeddedLCInfo`) |
| Late entry | two agreeing LCs grant a header-less over | `conventional.go` (`ingestVoiceSuperframe`, `lateEntryConfirm`) |
| Dead-call gate | drop LCs whose superframe predates the terminator | `endedAtDibit`, set in `releaseCall` |
| Talker alias | header 0x04 + blocks 0x05–0x07 → display name | `internal/radio/dmr/talker_alias.go` (`TalkerAliasAssembler`) |

## In this post

- **The Full Link Control word** — nine octets, two call shapes.
- **Three carriages** — header, terminator, embedded copy.
- **Embedded signalling** — the EMB frame and what is not yet corrected.
- **Late entry** — granting a transmission whose header was never heard.
- **The header train** — why the anchor follows the last copy.
- **Talker alias** — the other passenger.

## The Full Link Control word

Everything downstream of the burst — the grant, the recording's folder, the
Radio IDs roster — hangs off nine octets.
[Protocol Decoders Part 5]({{ '/blog/deep-dives/protocol-decoders-05-dmr-tier-2-3/' | relative_url }})
introduced the struct; this is the reading GopherTrunk performs:

```go
// internal/radio/dmr/flc.go (shape)
//	octet 0 PF|Res|FLCO(6) · 1 FID · 2 ServiceOptions
//	octets 3-5 destination (24-bit) · 6-8 source (24-bit)
const (
    FLCOGroupVoiceUser  FLCO = 0x00
    FLCOUnitToUnitVoice FLCO = 0x03
    FLCOTalkerAlias     FLCO = 0x04 // header; blocks 0x05, 0x06, 0x07
    FLCOGPS             FLCO = 0x08
)
```

Two opcodes define a voice call. `FLCOGroupVoiceUser` makes the destination a
talkgroup; `FLCOUnitToUnitVoice` makes it a called subscriber, and
`AsUnitToUnitVoice` marks the grant `Individual` so a private call's
destination RID never lands in the talkgroup list. The service-options octet
is shared — **bit 7 emergency, bit 6 privacy, bits 2..0 priority** — and is
the only place DMR signals encryption; a Tier III grant CSBK has no options
octet ([Part 8]({{ '/blog/deep-dives/dmr-end-to-end-08-tier3-trunking/' | relative_url }})).
The [reference page]({{ '/reference/dmr-full-link-control/' | relative_url }})
has the ETSI view; what matters here is that **one struct feeds every
reader** — header, terminator, embedded LC and the composer's slot router all
call `ParseFLC`, so the group-versus-individual decision cannot drift.

## Three carriages

**The Voice LC Header** (`DTVoiceLCHeader`, 0x1) opens a transmission: 9
octets of FLC plus a 24-bit RS(12,9) trailer XORed with the seed
`0x96 0x96 0x96`. `handleVoiceHeader` requires BPTC *and* RS — BPTC cannot
catch a systematic miss, and the RS check is what earns the lock (`FECPass`,
`dmr/tier2 cc locked`). A failing header is dumped at DEBUG with its exact
132 dibits (`burst_dibits`) — an instrument in the sense of
[From Spec to Shipping Part 13]({{ '/blog/deep-dives/from-spec-to-shipping-13-instruments-not-logs/' | relative_url }}).

**The Terminator with LC** (`DTTerminatorWithLC`, 0x2) closes it under the
seed `0x99 0x99 0x99` ([RS(12,9) reference]({{ '/reference/dmr-rs-12-9/' | relative_url }}));
`terminatorDest` returns the destination it names, so a TS1 terminator
releases only the TS1 call.

**The embedded copy** is the interesting one. Voice bursts carry no slot type
and no FLC; their sync field holds the voice sync on burst A and *embedded
signalling* on B–F. Bursts B–E each carry 32 bits of a 128-bit block that
reassembles into the same FLC, repeated every 360 ms superframe.

## Embedded signalling: the EMB frame

```go
// internal/radio/dmr/emb.go (shape)
//	bits  0..7   : EMB MSB half
//	bits  8..39  : embedded signalling fragment (32 bits)
//	bits 40..47  : EMB LSB half
const (
    LCSSSingle LCSS = 0 // single-fragment LC (CSBK/RC) or null
    LCSSFirst  LCSS = 1 // burst B
    LCSSLast   LCSS = 2 // burst E
    LCSSCont   LCSS = 3 // bursts C, D
)
```

The 16-bit **EMB** — colour code in bits 15..12, privacy indicator in
bit 11, the 2-bit **LCSS** in bits 10..9 — frames the fragment and says where
it sits. `ReassembleEmbeddedLCInfo` concatenates the four ordered fragments
and hands 128 on-air bits to `framing.DecodeEmbeddedLC`, the variable
BPTC(128,72) the
[embedded-LC reference]({{ '/reference/dmr-embedded-lc/' | relative_url }})
describes. A negative correction count rejects the whole word — a corrupted
fragment yields *no* LC, never a garbled one.

Two honesty notes. The EMB's own QR(16,7) protection is **not applied** — its
bits are read systematically and the embedded LC's BPTC + checksum is the
integrity gate ([EMB reference]({{ '/reference/dmr-emb/' | relative_url }})).
And because one bit error can flip an EMB colour-code nibble, the superframe
decoder exposes `EMBColorCode` as the **majority over the four LC-bearing
bursts** (`VoiceSuperframe.HasEMB`) — all an IPSC `color_code` filter has on
a header-less transmission.

## Late entry

The field report that forced this layer: an operator with a handheld beside
the scanner watched GopherTrunk say a call had ended while the conversation
continued on the radio. The tap was a weak −60 dBFS bin edge; the next
headers were lost to a fade at keyup, and the conventional path had **no
other way to grant**. Every subscriber radio does late entry from the
embedded LC. Now so does the scanner:

```go
// internal/radio/dmr/tier2/conventional.go (shape)
const lateEntryConfirm = 2          // agreeing superframes to grant
const lateEntryWindow  = 3 * time.Second

func (c *ConventionalChannel) ingestVoiceSuperframe(sf dmrvoice.VoiceSuperframe) {
    if end, ok := c.endedAtDibit[dest]; ok && sf.StartDibit < end {
        return // closing superframe of a call its terminator already released
    }
    cand := c.lateEntry[dest]
    if cand == nil || cand.src != src || now.Sub(cand.firstAt) > lateEntryWindow {
        c.lateEntry[dest] = &lateEntryCandidate{src: src, firstAt: now, seen: 1}
        return
    }
    if cand.seen++; cand.seen < lateEntryConfirm { return }
    c.cnt.lateEntries.Add(1)
    c.maybeLock(LockState{FrequencyHz: c.freqHz, ColorCode: sf.EMBColorCode})
    c.publishGrant(dest, src, individual, enc, emer, prio, ts, sf.EMBColorCode, true)
}
```

`Process` (`tier2/process.go`) runs a `dmrvoice` superframe assembler beside
the burst slicer and hands every CRC-valid embedded LC here. Three rules make
it safe.

**Confirm twice.** A lone CRC-valid LC is not enough
(`TestConventionalLateEntryNeedsTwoAgreeingLCs`): the copy repeats every
superframe, so two agreeing `(destination, source)` cost ~720 ms and rule out
one miscorrected word forging a phantom call. Two copies are as strong as a
BPTC+RS-clean header — they grant, count in `late_entries`, and **declare the
lock**.

**Respect the terminator.** The assembler cannot emit a superframe until
burst F arrives, so it runs *behind* the slicer and a transmission's closing
superframes surface after its terminator. Ungated, they re-granted every dead
call — measured on air, **7 phantom grant/release pairs in a 120 s capture**.
`releaseCall` records the terminator's dibit index in `endedAtDibit[dest]`;
any LC whose superframe *started* before it is dropped.

**Let the header win.** A header grant deletes any pending candidate; later
LCs only refresh the call (`TestConventionalHeaderAndEmbeddedLCGrantOnce`).

The real-air check is `TestDMRIPSCReplay` with `GT_DMR_DROP_HEADERS=1`, which
scrubs every Voice LC Header out of the operator's 9 Sep 442.3875 MHz
captures: all 5/5 transmissions still grant (`late_entries=5`), and
un-scrubbed runs grant exactly once per over.
`TestConventionalLateEntryGrantsHeaderlessTransmission` is the synthetic
failing-first regression — before this layer it produced no grant at all, the
rule
[From Spec to Shipping Part 12]({{ '/blog/deep-dives/from-spec-to-shipping-12-failing-first/' | relative_url }})
insists on.

<figure class="lab-figure">
<svg viewBox="0 0 680 250" width="680" height="250" role="img" aria-label="A DMR transmission timeline: two header copies lost to a fade, then superframes whose embedded Link Control makes the first a late-entry candidate and the second an agreeing copy that grants the call; a Terminator with LC records endedAtDibit, and a closing superframe surfacing after it is dropped by that gate.">
  <line x1="20" y1="120" x2="660" y2="120" stroke="var(--fg-muted)"/>
  <text x="20" y="22" fill="currentColor" font-size="11" font-weight="bold">one transmission, stream order (dibits →)</text>
  <rect x="30" y="98" width="22" height="22" fill="none" stroke="var(--fg-muted)" stroke-dasharray="3 2"/>
  <rect x="56" y="98" width="22" height="22" fill="none" stroke="var(--fg-muted)" stroke-dasharray="3 2"/>
  <text x="54" y="90" text-anchor="middle" fill="var(--fg-muted)" font-size="9">headers lost to fade</text>
  <rect x="96" y="98" width="120" height="22" fill="none" stroke="currentColor"/>
  <text x="156" y="113" text-anchor="middle" fill="currentColor" font-size="9">superframe 1 · A B C D E F</text>
  <rect x="222" y="98" width="120" height="22" fill="none" stroke="var(--accent)" stroke-width="2"/>
  <text x="282" y="113" text-anchor="middle" fill="var(--accent)" font-size="9">superframe 2 · A B C D E F</text>
  <rect x="348" y="98" width="120" height="22" fill="none" stroke="currentColor"/>
  <rect x="482" y="98" width="30" height="22" fill="none" stroke="currentColor"/>
  <text x="497" y="113" text-anchor="middle" fill="currentColor" font-size="8">TERM</text>
  <rect x="530" y="98" width="120" height="22" fill="none" stroke="var(--fg-muted)" stroke-dasharray="3 2"/>
  <text x="590" y="113" text-anchor="middle" fill="var(--fg-muted)" font-size="9">closing SF (surfaces late)</text>
  <text x="282" y="140" text-anchor="middle" fill="var(--accent)" font-size="9">embedded LC #2 agrees</text>
  <text x="282" y="152" text-anchor="middle" fill="var(--accent)" font-size="9" font-weight="bold">late-entry GRANT · ~720 ms</text>
  <text x="497" y="140" text-anchor="middle" fill="currentColor" font-size="9">releaseCall</text>
  <text x="497" y="152" text-anchor="middle" fill="currentColor" font-size="9">endedAtDibit[dest]</text>
  <line x1="497" y1="70" x2="497" y2="96" stroke="currentColor"/>
  <text x="590" y="140" text-anchor="middle" fill="var(--fg-muted)" font-size="9">StartDibit &lt; endedAtDibit</text>
  <text x="590" y="152" text-anchor="middle" fill="var(--fg-muted)" font-size="9">→ dropped, not re-granted</text>
</svg>
<figcaption>Late entry on a header-less transmission: the first embedded LC is a candidate, the second agreeing copy grants and locks, and the closing superframe that surfaces after the terminator is stopped by the dibit-position gate.</figcaption>
</figure>

## The header train

The embedded LC cannot say where a transmission *started*. That stays with
the header — and the header is not one burst. A repeater sends two or three
copies at keyup, 60 ms (288 dibits) apart; the direct-mode handheld of
[#836](https://github.com/MattCheramie/GopherTrunk/issues/836) repeats its
Voice LC Header **ten times over 0.6 s**. Copies are deduped by the
`headerRekeyDibits` rule — 1200 dibits, 0.25 s — from an anchor. Measured
from the *first* copy, the handheld's sixth was already a "new transmission",
and every PTT opened with a phantom release and re-grant. The fix: the anchor
**moves up to this copy**.

```go
// internal/radio/dmr/tier2/conventional.go (shape)
if existing.src == src {
    if c.ingestDibit-existing.anchorDibit <= headerRekeyDibits {
        if c.ingestDibit > existing.anchorDibit {
            existing.anchorDibit = c.ingestDibit // anchor follows the LAST copy
        }
        existing.touch(c.ingestDibit)
        return
    }
    /* re-key: release + re-grant — Part 6 */
}
```

`TestConventionalHeaderTrainIsOneKeyup` lays a ten-copy train on the
288-dibit direct-mode grid, two seconds of silence with the terminator lost,
then a three-copy train, and asserts exactly two grants and `Rekeys == 1`.
The train is one keyup; the genuine re-key still re-grants — the subject of
[Part 6]({{ '/blog/deep-dives/dmr-end-to-end-06-conventional-ipsc-two-calls/' | relative_url }}),
where the same anchor arithmetic recovers replies an IPSC tap was dropping.

## Talker alias: the other passenger

The embedded LC is a carriage for *any* 72-bit FLC, and the second most
useful thing it carries is a name. FLCO `0x04` is a **talker alias header**
declaring a 2-bit format and a 5-bit character count plus six octets of text;
`0x05`–`0x07` are blocks of seven octets. `TalkerAliasAssembler` buffers
fragments per source radio — they carry no address, so the voice chain keys
them on the call's current source — and emits the name once the text covers
the declared length:

```go
// internal/radio/dmr/talker_alias.go (shape)
//	Header (FLCO 0x04):  bits 16..17 format · 18..22 length · 24..71 text (6 octets)
//	Blocks (0x05–0x07):  bits 16..71 text (7 octets)
// formats: 0 = 7-bit packed, 1 = ISO 8859-1, 2 = UTF-8, 3 = UTF-16BE
const dmrAliasStaleAfter = 10 * time.Second
```

The composer's DMR chain runs the assembler on every superframe of *both*
slots before the slot-router gate — alias and GPS LCs are call metadata, not
one slot's audio — and publishes `KindTalkerAlias` with the INFO line
`composer: dmr talker alias`. The layout is cross-checked against ok-dmrlib,
and the code flags its working model: header bit 23 is treated as reserved,
and "a capture that decodes to garbage should re-check bit 23's
inclusion first." That is the posture the P25 alias took in
[Protocol Decoders Part 10]({{ '/blog/deep-dives/protocol-decoders-10-alias-hunt-framing/' | relative_url }})
and [Part 11]({{ '/blog/deep-dives/protocol-decoders-11-alias-hunt-cryptanalysis/' | relative_url }})
— except DMR's alias is plaintext where Motorola's is an unsolved cipher. Both
feed the one `trunking.TalkerAlias` shape
[Trunking Engine Part 7]({{ '/blog/deep-dives/trunking-engine-07-source-rid-recovery/' | relative_url }})
plumbs into the roster.

### How link control shaped the Go code

- **One parser, many readers.** `ParseFLC` plus the two `As*` accessors are
  the only group/individual decision in the DMR tree.
- **Evidence is stream-positioned, not wall-clocked.** `anchorDibit`,
  `lastDibit` and `endedAtDibit` are dibit indices, so the rules hold in an
  offline replay running faster than real time.
- **Raw octets survive the parse.** `ReassembleEmbeddedLCInfo` returns the
  9-octet block beside the FLC for the alias and GPS readers.
- **Counters name the mechanism.** `late_entries` climbing on a live repeater
  means a weak tap is catching conversations mid-transmission.

## Where this goes next

Late entry and the header anchor are half the conventional state machine.
[Part 6]({{ '/blog/deep-dives/dmr-end-to-end-06-conventional-ipsc-two-calls/' | relative_url }})
supplies the other half: how one IPSC repeater's two timeslots become two
concurrent calls, and the 10 Sep field report in which replies inside
hangtime were deduped against calls the engine had already ended.

## FAQ

**What is DMR Full Link Control?**
A 72-bit PDU that identifies a voice call: a 6-bit FLCO opcode, feature-set
ID, service options (emergency, privacy, priority) and 24-bit destination and
source addresses. GopherTrunk parses it from the Voice LC Header, the
Terminator with LC, and the embedded signalling of bursts B–E.

**How does DMR late entry work in a scanner?**
Every voice superframe embeds the call's Link Control across bursts B–E, so a
receiver that missed the header still learns the talkgroup and source.
GopherTrunk grants a header-less transmission once two agreeing CRC-valid
embedded LCs arrive, about 720 ms in.

**Why does GopherTrunk need two embedded LCs before granting?**
Because one CRC-valid word can be a miscorrection. The embedded LC repeats
every 360 ms, so a second agreeing copy costs little and rules out a phantom
call; two copies grant, lock the channel and count in `late_entries`.

**What is the DMR EMB field?**
The 16-bit embedded-signalling header in the sync region of voice bursts B–F:
colour code, privacy indicator and the 2-bit LCSS marking a fragment first,
continuation or last. GopherTrunk reads it systematically and relies on the
embedded LC's BPTC(128,72) plus checksum as the integrity gate.

**Does GopherTrunk decode the DMR talker alias?**
Yes. The alias header (FLCO 0x04) declares format and length, blocks 0x05–0x07
carry the rest, and `TalkerAliasAssembler` reassembles the radio's display
name per source in 7-bit, ISO 8859-1, UTF-8 or UTF-16 form and publishes it
on the event bus.

## Series navigation

**Part 5 of 14** · ←
[Part 4: The FEC Stack & the Forged Terminator]({{ '/blog/deep-dives/dmr-end-to-end-04-fec-stack-forged-terminator/' | relative_url }})
· Next →
[Part 6: Conventional IPSC — Two Slots as Two Calls]({{ '/blog/deep-dives/dmr-end-to-end-06-conventional-ipsc-two-calls/' | relative_url }})
