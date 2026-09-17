---
title: "DMR End to End, Part 7: The Idle Beacon & the CSBK Train"
description: What a keyed-but-idle IPSC repeater puts on the air, how GopherTrunk pinned the ETSI Idle burst against MMDVMHost's constant and turned it into a site-alive lock, how CSBK beacons are gated by CRC — and the cc=7 CSBK train on a cc=12 system that is still, honestly, unresolved.
category: deep-dives
keywords: dmr idle burst, dmr idle beacon, ipsc repeater idle, dmr csbk crc 0x5a5a, mmdvmhost dmr idle data, dmr site alive, dmr csbk crc mismatch, conventional dmr camp, dmr slot type 9, gophertrunk dmr beacon
tags: [dmr-end-to-end, dmr, ipsc, csbk, idle-burst, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "DMR End to End"
series_part: 7
---

*Part 7 of **DMR End to End**, a 14-part deep dive that follows the world's
most widely deployed digital PMR protocol through GopherTrunk — from a 4FSK
carrier to two simultaneous recorded calls, direct-mode handhelds, and
decrypted Enhanced Privacy voice.
[Part 6]({{ '/blog/deep-dives/dmr-end-to-end-06-conventional-ipsc-two-calls/' | relative_url }})
followed two calls through one carrier. This part is about the carrier when
nobody is talking: the Idle bursts a keyed repeater fills both slots with,
the CSBK beacons some repeaters send instead, what the scanner does with
each — and the one train of blocks it can hear but not yet read.*

> **TL;DR:** A keyed-but-idle Motorola/Hytera IPSC repeater fills **both**
> timeslots with the ETSI DMR **Idle** burst (slot type 9, `dmr.DTIdle`) in
> ~10 s trains of ~33 bursts/s. Its BPTC-decoded block is always
> `ff83df1732094ed1e7cd8a91` (2402 of 2416 bursts on the 10 Sep capture) —
> and BPTC-decoding MMDVMHost's `DMR_IDLE_DATA` template yields the same 12
> bytes with zero corrections (`TestIdleInfoPatternMatchesMMDVMHostConstant`).
> `handleIdle` (`tier2/conventional.go`) counts a BPTC-clean Idle carrying
> exactly `IdleInfoPattern` as a **beacon**, declares the lock (site camped
> while idle) and logs `dmr/tier2 site alive (idle beacon)` at most every
> 30 s; any other block is parked with `info_hex` as a vendor variant.
> `handleCSBK` does the same for CRC-valid CSBKs (mask `0x5A5A`). Beacons
> count as decoding for `activityClass`, silencing the "strong signal but no
> sync" hint. What remains **unresolved**: a 30 ms-cadence train of
> BPTC-clean, CRC-failing CSBKs at **cc=7** on a **cc=12** system — real,
> proprietary, parked at DEBUG with `csbko/fid/lb/pf/info_hex`, counted in
> `csbk_crc_fail`, and waiting for a capture.

**Key takeaways**

- **Idle is a burst type, not silence.** A keyed repeater with nothing to
  carry transmits a fixed 96-bit block on both slots — the beacon an
  operator hears as "the repeater is up".
- **A beacon is pinned by content, never by slot type alone.** Golay(20,8)
  decodes ~1/3 of arbitrary words to *some* codeword; the gate is BPTC-clean
  *and* block-equals-pattern (or CSBK CRC-valid).
- **The pattern is pinned against an independent constant.** MMDVMHost's
  `DMR_IDLE_DATA` decodes to the same bytes — a literal vector, so the pattern
  cannot drift with GopherTrunk's own decoder.
- **Unreadable is logged, not guessed.** The cc=7 CSBK train is parked with
  its raw block; no layout is invented for it.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| Idle burst | slot type `0x9`, fixed 96-bit information block | `internal/radio/dmr/slottype.go` (`DTIdle`) |
| The pattern | `ff83df1732094ed1e7cd8a91` | `tier2/conventional.go` (`IdleInfoPattern`) |
| Independent pin | MMDVMHost `DMR_IDLE_DATA` → same 12 bytes | `TestIdleInfoPatternMatchesMMDVMHostConstant` |
| Idle handler | BPTC + pattern → beacon, lock, rate-limited INFO | `handleIdle`, `beaconLogInterval` (30 s) |
| CSBK beacon | BPTC + CRC (`0x5A5A`) → beacon | `handleCSBK`, `tier3.ParseCSBK` |
| Parked failure log | first + summary per 10 s, raw `info_hex` | `logCSBKFailure`, `csbkFailLogInterval` |
| Engine effects | beacons = decoding; low-power WARN at most once | `widebandt2/engine.go` (`activityClass`, `maybeLogDiagnostics`) |

## In this post

- **What a keyed repeater sends** — Idle bursts, both slots, ~33 a second.
- **Pinning the pattern** — the MMDVMHost constant as a literal vector.
- **handleIdle** — content-strict, then camp.
- **CSBK beacons and the engine** — CRC-strict, and what beacons switch off.
- **The cc=7 train** — parked, counted, unresolved.

## What a keyed repeater sends when nobody talks

Between calls a conventional IPSC repeater does not go quiet. On the
operator's 10 Sep beacon-only capture (442.3875 MHz, the same repeater as
[Part 6]({{ '/blog/deep-dives/dmr-end-to-end-06-conventional-ipsc-two-calls/' | relative_url }}))
a keyed Motorola/Hytera repeater with no voice fills **both timeslots** with
BS-Data-synced bursts whose [slot type]({{ '/reference/dmr-slot-type/' | relative_url }})
reads colour code 12, data type 9 — the ETSI **Idle** burst — at ~33 bursts/s
in ~10 s trains, separated on the 15 Sep log by ~5–9 s gaps of true silence.
Their BPTC-decoded information block is the same 12 bytes on 2402 of the
2416 synced bursts.

Before this round, `IngestBurst`'s switch had no case for `dmr.DTIdle`, so
those bursts fell through. Three symptoms followed: `beacons` sat at 0 on a
repeater audibly keyed for ten seconds at a time; the channel never reported
locked until someone spoke; and the wideband engine's "strong in-channel
signal but no sync" hint fired on every train — a mistune diagnosis against a
repeater decoding fine. The operator's request: *camp on the idle beacon*.

## Pinning the pattern against an independent constant

The fix starts with a constant, and the constant needed a second witness:

```go
// internal/radio/dmr/tier2/conventional.go (shape)
// The fixed 96-bit information block of the ETSI DMR Idle burst as
// recovered through BPTC(196,96). Pinned two independent ways: the 10 Sep
// capture (2402/2416 bursts at cc 12) and MMDVMHost's DMR_IDLE_DATA.
var IdleInfoPattern = [12]byte{
    0xff, 0x83, 0xdf, 0x17, 0x32, 0x09, 0x4e, 0xd1, 0xe7, 0xcd, 0x8a, 0x91,
}
```

A capture alone would pin the pattern to *this* repeater and *this* decoder —
the self-consistent trap
[Issue Tracker Part 20]({{ '/blog/solution-postmortem/from-the-issue-tracker-20-self-consistent-trap/' | relative_url }})
dissects. MMDVMHost ships the Idle burst as a 33-byte constant
(`DMR_IDLE_DATA` in `DMRDefines.h`): payload half, slot type, BS-Data sync,
slot type, payload half, at colour code 0. It is a template —
MMDVMHost stamps the colour code and Idle slot type into it per repeater at
transmit time — so only the BPTC-coded information block is comparable, and
`TestIdleInfoPatternMatchesMMDVMHostConstant` does exactly that: unpack the
33 bytes to 132 dibits, run GopherTrunk's own `framing.DecodeBPTC196_96`,
require **zero** corrections, and require byte equality with
`IdleInfoPattern`. That is a literal vector in the sense
[From Spec to Shipping Part 3]({{ '/blog/deep-dives/from-spec-to-shipping-03-literal-vectors/' | relative_url }})
means it: bytes produced by something that is not this codebase.

## handleIdle: content-strict, then camp

```go
// internal/radio/dmr/tier2/conventional.go (shape)
func (c *ConventionalChannel) handleIdle(b *dmr.Burst, slot dmr.SlotType) {
    bits, errs := framing.DecodeBPTC196_96(b.PayloadBits())
    if errs < 0 { return }
    info := infoBitsToBytes(bits)
    if !bytes.Equal(info, IdleInfoPattern[:]) {
        c.logCSBKFailure("dmr/tier2: idle burst with an unknown information block (vendor idle variant?)", slot, nil, info)
        return
    }
    c.cnt.beacons.Add(1)
    c.burstValid = true
    c.maybeLock(LockState{FrequencyHz: c.freqHz, ColorCode: slot.ColorCode})
    /* rate-limited: */ c.log.Info("dmr/tier2 site alive (idle beacon)",
        "freq", c.freqHz, "cc", slot.ColorCode, "system", c.systemName)
}
```

The gate mirrors the header's discipline from
[Part 4]({{ '/blog/deep-dives/dmr-end-to-end-04-fec-stack-forged-terminator/' | relative_url }}):
a noise burst that false-syncs and Golay-decodes to `(cc, Idle)` cannot forge
a beacon, because it would also have to be a BPTC codeword carrying that
exact block. A BPTC-clean Idle with a *different* block is a vendor variant
GopherTrunk has not seen; it is dropped and its `info_hex` parked at DEBUG so
the next field log can pin it — never published as a decode error. A matching
Idle is as strong a proof of a DMR repeater as a voice header, so it sets
`burstValid` (fixing the stream polarity) and **camps**: `maybeLock` reports the channel locked while idle, which is what
lets `cchunt` park (`cchunt: camped on conventional channel — idle, waiting
for traffic`) instead of re-hunting a silent repeater. The INFO line is
rate-limited to `beaconLogInterval` (30 s); the `Beacons` counter keeps the
exact count. `maybeLock` itself dedupes on **frequency**, not on `{freq, cc}`
— a single Golay-miscorrected colour code used to republish `cc.locked` on
every flicker.

`TestConventionalIdleBeaconMarksSiteAliveAndLocks` is the failing-first pin:
a keyed idle carrier at cc 12 must count beacons and lock, and an Idle with a
non-ETSI block must do neither.

## CSBK beacons, and what a beacon switches off

Some repeaters beacon with a **CSBK** instead — a Preamble or a
[C_BCAST]({{ '/reference/dmr-csbk-payloads/' | relative_url }}) carrying valid
sync and colour code but no voice. `handleCSBK` runs BPTC then
`tier3.ParseCSBK`, whose 16-bit trailer must match CRC-CCITT of the leading
80 bits XORed with `0x5A5A`
([DMR CSBK CRC]({{ '/reference/dmr-csbk-crc/' | relative_url }})). A CRC-valid
CSBK bumps `beacons`, sets `burstValid`, and logs — Tier III owns CSBK
*semantics*; here only the keep-alive fact is needed:

```
INF dmr/tier2 site alive (beacon) freq=442387500 cc=12 csbk=Preamble system=Fire2
INF dmr/tier2 site alive (idle beacon) freq=442387500 cc=12 system=Fire2
DBG widebandt2: channel decode activity freq_hz=442387500 system=Fire2 dibits=144000 sync_hits=991 bursts=982 fec_pass=0 fec_fail=0 beacons=978 late_entries=0 csbk_crc_fail=0 rekeys=0 locks_total=1 deaf_heals=0
```

Between-beacon noise on a parked channel is expected, so a failing CSBK is
deliberately **not** published as a `KindDecodeError` — surfacing it was the
"treated as corrupted data" symptom of
[#1036](https://github.com/MattCheramie/GopherTrunk/issues/1036).

Beacons then change how the wideband engine reads the channel.
`activityClass(syncDelta, fecPassDelta, beaconDelta)` buckets a window as
idle / syncing / **decoding**, and a beacon delta counts as decoding — so the
"strong in-channel signal but no sync" hint no longer fires on beacon trains.
And the low-power WARN treats a conventional carrier differently from a
control channel: an idle repeater legitimately drops to no carrier between
transmissions, so `maybeLogDiagnostics` warns **at most once** for a channel
that has never decoded and stays silent once `FECPass > 0 || Beacons > 0`
(`TestLowPowerConventionalDecodedThenIdleIsSilent`), while control-channel
protocols keep the repeating WARN. The parked DEBUG lines summarise every 30 s
instead of once per second — the "endless spam" report.

## The cc=7 CSBK train: parked, counted, unresolved

Now the honest part. The operator's 443.2375 MHz log — the *other* repeater
— shows a train of CSBK-typed bursts on **both** timeslots at a 30 ms
cadence, every one BPTC-clean and every one failing the CSBK CRC, at **colour
code 7** on a **colour code 12** system, for about four seconds. Noise does
not BPTC-decode cleanly at a fixed cadence for four seconds: it is a real,
vendor-proprietary train GopherTrunk does not yet understand.

Two things it is *not*. Not a CRC-convention bug: the `0x5A5A` mask is pinned
by real Tier III vectors (`csbk_realvectors_test.go`, from 2 MS/s captures of
live TSCCs at 440.5625 and 440.2625 MHz, zero BPTC corrections) — and the
*earlier* convention, "init 0xFFFF, store the complement", rejected every
real CSBK while passing synthetic round-trips. And not on the 10 Sep
beacon-only capture, which shows only a few Golay-forged slot types on voice
bursts (Part 4's class), so the train could not be pinned from it.

What landed is an instrument, not a guess:

```go
// internal/radio/dmr/tier2/conventional.go (shape)
const csbkFailLogInterval = 10 * time.Second

func (c *ConventionalChannel) logCSBKFailure(msg string, slot dmr.SlotType, csbk *tier3.CSBK, info []byte) {
    key := msg + "|" + cc            // + "|opcode|fid" when the header parsed
    attrs := []any{"cc", slot.ColorCode,
        "csbko", ..., "fid", ..., "lb", csbk.LB, "pf", csbk.PF,
        "info_hex", hex.EncodeToString(info)}
    // same key within the interval: count it, summarise as suppressed_repeats
}
```

The first failure of a `(message, cc, opcode, fid)` key logs at once with the
header fields and raw block; unchanged repeats within 10 s are counted
(`suppressed_repeats`); a key change logs immediately so a transition is never
hidden. `CSBKCRCFail` counts every one — `csbk_crc_fail` in the activity line.
The 9 Sep storm was one identical
`dmr/tier2: CSBK CRC mismatch (between-beacon noise or proprietary CSBK train)`
line per burst; `TestConventionalCSBKFailureLogIsParked` pins the parking.

What would settle it is a capture of 443.2375 MHz **idle** — the train with
no voice around it — replayed through `TestDMRIPSCBurstDump` (`GT_DMR_DUMP`),
which dumps a capture burst by burst with slot types, colour codes and
embedded LC. Until then, no layout gets invented for it: the
[SmartNet rebuild]({{ '/blog/deep-dives/from-spec-to-shipping-08-smartnet-rebuild/' | relative_url }})
is what a fabricated framing costs. The `info_hex` in the next log is the
whole plan.

<figure class="lab-figure">
<svg viewBox="0 0 680 250" width="680" height="250" role="img" aria-label="Decision tree for a burst on a camped conventional DMR channel: an Idle burst whose BPTC block equals the ETSI pattern becomes a beacon that locks the channel, any other block is parked as a vendor variant with its raw hex; a CSBK that passes the 0x5A5A CRC is a beacon, one that fails is parked and counted in csbk_crc_fail, where the unresolved cc=7 train lands.">
  <rect x="250" y="14" width="180" height="30" rx="5" fill="none" stroke="currentColor"/>
  <text x="340" y="33" text-anchor="middle" fill="currentColor" font-size="10">data-sync burst · slot type OK</text>
  <line x1="300" y1="44" x2="170" y2="76" stroke="currentColor"/>
  <line x1="380" y1="44" x2="510" y2="76" stroke="currentColor"/>
  <rect x="100" y="76" width="140" height="28" rx="5" fill="none" stroke="var(--accent)"/>
  <text x="170" y="94" text-anchor="middle" fill="var(--accent)" font-size="10">Idle (type 9) → BPTC</text>
  <rect x="440" y="76" width="140" height="28" rx="5" fill="none" stroke="var(--accent)"/>
  <text x="510" y="94" text-anchor="middle" fill="var(--accent)" font-size="10">CSBK (type 3) → BPTC</text>
  <line x1="140" y1="104" x2="90" y2="138" stroke="currentColor"/>
  <line x1="200" y1="104" x2="250" y2="138" stroke="var(--fg-muted)"/>
  <line x1="480" y1="104" x2="430" y2="138" stroke="currentColor"/>
  <line x1="540" y1="104" x2="590" y2="138" stroke="var(--fg-muted)"/>
  <text x="90" y="128" text-anchor="middle" fill="var(--fg-muted)" font-size="8">block == IdleInfoPattern</text>
  <text x="250" y="128" text-anchor="middle" fill="var(--fg-muted)" font-size="8">other block</text>
  <text x="430" y="128" text-anchor="middle" fill="var(--fg-muted)" font-size="8">CRC ^ 0x5A5A ok</text>
  <text x="590" y="128" text-anchor="middle" fill="var(--fg-muted)" font-size="8">CRC fails</text>
  <rect x="20" y="140" width="140" height="44" rx="5" fill="none" stroke="var(--accent)" stroke-width="2"/>
  <text x="90" y="157" text-anchor="middle" fill="var(--accent)" font-size="9" font-weight="bold">beacon · lock</text>
  <rect x="180" y="140" width="140" height="44" rx="5" fill="none" stroke="var(--fg-muted)" stroke-dasharray="3 2"/>
  <text x="250" y="157" text-anchor="middle" fill="currentColor" font-size="9">parked DEBUG</text>
  <text x="250" y="171" text-anchor="middle" fill="var(--fg-muted)" font-size="8">vendor idle variant? · info_hex</text>
  <rect x="360" y="140" width="140" height="44" rx="5" fill="none" stroke="var(--accent)" stroke-width="2"/>
  <text x="430" y="157" text-anchor="middle" fill="var(--accent)" font-size="9" font-weight="bold">beacon · lock</text>
  <rect x="520" y="140" width="140" height="44" rx="5" fill="none" stroke="var(--fg-muted)" stroke-dasharray="3 2"/>
  <text x="590" y="157" text-anchor="middle" fill="currentColor" font-size="9">parked · csbk_crc_fail++</text>
  <text x="590" y="205" text-anchor="middle" fill="var(--accent)" font-size="9">← the cc=7 train lands here</text>
  <text x="590" y="218" text-anchor="middle" fill="var(--fg-muted)" font-size="8">30 ms cadence · both slots · unresolved</text>
  <text x="180" y="212" text-anchor="middle" fill="var(--fg-muted)" font-size="9">neither parked branch publishes a decode.error</text>
</svg>
<figcaption>Every idle-channel burst ends in one of four boxes: two beacons that lock the channel, two parked logs that keep the raw block for the next report. The cc=7 train lives bottom-right.</figcaption>
</figure>

### How the idle channel shaped the Go code

- **Beacons are counted, logs are rate-limited.** `Beacons` is exact;
  `beaconLogAt` throttles the INFO line to one per 30 s, so a fast beacon
  interval never floods the log yet the activity line stays precise.
- **Parking is keyed, not blanket.** `logCSBKFailure` keys on
  `(message, cc, opcode, fid)`; a change logs immediately, so the parking
  can never hide the moment a train appears or changes colour code.
- **Unknown blocks keep their bytes.** Both parked branches carry `info_hex`
  — the next field log is a burst dump, not a symptom description.
- **Conventional and control channels warn differently.**
  `maybeLogDiagnostics` warns once for an idle conventional carrier and
  repeatedly for a silent control channel — silence means opposite things.

## Where this goes next

Everything so far ran on a carrier with no control channel. [Part
8]({{ '/blog/deep-dives/dmr-end-to-end-08-tier3-trunking/' | relative_url }})
adds one: DMR Tier III, where the same CSBK that is only a keep-alive here
carries C_ALOHA, C_BCAST and the voice grants — and where an LCN, not a
frequency, is what a grant hands you.

## FAQ

**What is a DMR Idle burst?**
A data burst with slot type 9 whose 96-bit information block is a fixed
ETSI pattern — `ff83df1732094ed1e7cd8a91` after BPTC decoding. A keyed
repeater with nothing to carry transmits it on both timeslots, which is why a
conventional DMR channel is audibly "up" between calls.

**How does GopherTrunk know a conventional DMR repeater is alive?**
From its beacons: a BPTC-clean Idle burst carrying exactly `IdleInfoPattern`,
or a CSBK whose 0x5A5A-masked CRC checks. Either counts in `beacons`, locks
the channel so the hunter camps on it, and logs
`dmr/tier2 site alive (idle beacon)` or `site alive (beacon)` at most every
30 s.

**Why not lock on the slot type alone?**
Because Golay(20,8) decodes roughly a third of random 20-bit words to some
codeword, so a false sync routinely yields a "valid" slot type — the old
"instalock cc=15 then nothing" symptom. GopherTrunk requires content: the idle
pattern in a BPTC-clean block, or a CSBK that passes CRC.

**What does `csbk_crc_fail` in the activity line mean?**
CSBK-typed bursts that BPTC-decoded but failed the CSBK CRC. A few between
beacons is noise; a sustained rate on a keyed repeater is a proprietary CSBK
train GopherTrunk cannot yet read — parked at DEBUG with the raw `info_hex`
so a capture can pin it.

**Is the cc=7 CSBK train a GopherTrunk bug?**
Unknown, and stated so. It is BPTC-clean at a 30 ms cadence on both slots for
seconds, so it is a real signal, not noise; the CRC mask is pinned by real
Tier III vectors, so it is not a checksum bug. It is an unread vendor format
awaiting an idle capture of that frequency.

## Series navigation

**Part 7 of 14** · ←
[Part 6: Conventional IPSC — Two Slots as Two Calls]({{ '/blog/deep-dives/dmr-end-to-end-06-conventional-ipsc-two-calls/' | relative_url }})
· Next →
[Part 8: Tier III Trunking — C_ALOHA, Grants & LCNs]({{ '/blog/deep-dives/dmr-end-to-end-08-tier3-trunking/' | relative_url }})
