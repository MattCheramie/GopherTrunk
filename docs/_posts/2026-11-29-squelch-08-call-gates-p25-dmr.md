---
title: "Squelch, Part 8: Call Gates I — An Idle Repeater Is Not a Call"
description: "On a digital scan-list channel, in-channel power proves a carrier, not a call. How GopherTrunk's P25 NAC gate and DMR call gate make decoded frames the squelch decision — a NID with the right NAC, a Voice LC Header or a voice sync — so another system's traffic and a repeater's idle beacon stop holding the scanner."
category: deep-dives
keywords: p25 nac squelch, p25 conventional nac filter, dmr idle beacon scanner, dmr call detection voice sync, digital scan list squelch, dmr sync polarity inverted, symbol timing acquisition retune, nac f7e receive any, sdr scanner call gate, gophertrunk call gate
tags: [squelch, p25, dmr, conventional, scanner, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "Squelch"
series_part: 8
---

*Part 8 of **Squelch**, a 14-part deep dive into GopherTrunk's conventional
scanner — the scan list with no control channel to tell it a call exists —
and every gate that decides when a channel opens.
[Part 7]({{ '/blog/deep-dives/squelch-07-am/' | relative_url }})
gave AM channels a squelch measured in carrier-to-noise instead of dBFS. This
part crosses into the digital modes, where even a perfect carrier measurement
answers the wrong question: a P25 frequency two systems share, and a DMR
repeater that stays keyed while nobody talks. In both cases the fix is to let
a decoded frame, not a power reading, say "call".*

> **TL;DR:** A `p25` or `dmr` scan-list channel squelches on in-channel
> power, which proves a carrier and nothing else. Two gates add the missing
> evidence. `NACDetector` (config `nac: "293"`) runs the production P25
> Phase 1 receiver and is present only within 750 ms of a NID carrying the
> wanted NAC; another NAC shuts it at once, and `F7E` accepts any NAC but
> only P25. `DMRCallDetector` is present only within 1 s of a Voice LC
> Header, a PI header or a voice-burst sync, so a repeater's Idle bursts and
> CSBK beacons never open a call. It locks polarity from two clean slot
> types, arms feed-forward timing on every retune (0/60 syncs at some start
> phases without it, 59–60/60 with) and may extend a power-open scan window
> once, to 500 ms. A `p25` channel without `nac` still opens on level alone.
> Both gates are synthetic-verified.

**Key takeaways**

- **Power answers "is a carrier up?", never "is someone talking?".** A
  keyed-but-idle DMR repeater and a foreign P25 system both clear
  `squelch_dbfs` honestly; no threshold separates them from a call.
- **The evidence is a decoded frame, so the gate is a receiver.** Each gate
  runs the production demodulator on the channel's own narrowband IQ and
  reports the age of its last good frame.
- **A DMR sync word alone is ambiguous.** Data and voice syncs are each
  other's polarity flip, so the gate fixes the polarity from clean slot types
  before it believes a voice sync.
- **A gate has one scan window to acquire.** Tuning onto a carrier that is
  already up lands on a random symbol phase; feed-forward timing is what
  makes 500 ms enough.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| Gate interface | a detector that reports a call; sees every chunk and may hold a power-open window to its minimum dwell | `scanner.go` (`callGate`, `scanWindow`) |
| P25 NAC filter | frame sync within 4 dibits, then the NID's NAC; present for 750 ms; a foreign NAC shuts it at once | `nac.go` (`NACDetector`, `nacHold`, `ParseNAC`) |
| NAC config | three hex digits; F7E or F7F = any NAC, P25 only; empty = no gate | `config.example.yaml` (`nac`, `p25_phase1_demod_mode`) |
| DMR call evidence | Voice LC Header, PI header or voice sync heard within 1 s | `dmr_gate.go` (`classify`, `dmrCallHold`) |
| DMR polarity | two zero-correction slot types at one polarity lock it | `dmr_gate.go` (`dmrPolarityLockVotes`), `dmr.SyncIsDataAtPolarity` |
| Timing at retune | feed-forward acquisition armed at construction and after every reset | `internal/radio/dmr/receiver/timing_acq.go` (`ArmTimingAcquisition`) |
| Scan cost | NAC: the list's minimum dwell rises to 500 ms; DMR: one extension to 500 ms, only after power breaks | `scanner.go` (`minToneDwell`), `nacMinDwell`, `dmrGateMinDwell` |
| Pins | an idle beacon opens no call at either polarity; a foreign NAC opens no call | `dmr_gate_test.go`, `nac_test.go` |

## In this post

- **A carrier is not a call** — what a power squelch can and cannot say on a digital channel.
- **The NAC gate** — CTCSS for P25: a NID with the right NAC.
- **The DMR call gate** — three kinds of evidence a beacon never produces.
- **Polarity** — why an inverted beacon reads as voice until the lock.
- **One scan window to acquire** — the sub-symbol phase problem.
- **How the scanner drives a gate** — `callGate`, the extension, the dwell.

## A carrier is not a call

The digital modes joined the scan list in two steps — `p25`
([#1239](https://github.com/MattCheramie/GopherTrunk/issues/1239)), then
`dmr`, `nxdn`, `nxdn48`, `dstar` and `ysf`
([#1261](https://github.com/MattCheramie/GopherTrunk/issues/1261)) — on one
assumption, written beside the mode constants: *a conventional digital
transmitter is only on the air while keyed*. So a digital channel squelches
like an FM one, on the in-channel power of
[Part 2]({{ '/blog/deep-dives/squelch-02-channel-power/' | relative_url }}),
and hands the call to that protocol's voice chain. The assumption fails in
two directions.

**A frequency is not a system.** The regression test for
[#1262](https://github.com/MattCheramie/GopherTrunk/issues/1262) records the
field case: the reporter's P25 frequency carried their own units, the
repeater and a third party's encrypted units, all recorded under one
channel. Every one of those is a real P25 transmission well above squelch.

**A repeater's carrier is not a transmission.**
[DMR End to End Part 7]({{ '/blog/deep-dives/dmr-end-to-end-07-idle-beacon-csbk/' | relative_url }})
showed what a keyed-but-idle repeater sends: Idle bursts on both timeslots,
or CSBK beacons — a strong, clean carrier with nobody on it. The #1261 field
report, quoted in `dmr_gate_test.go`, is the result: "the scanner stays on
the channel while the beacon is up", on a call that recorded nothing.

Neither is a threshold problem. No value of `squelch_dbfs` separates two
carriers of the same strength, and the rule this series keeps returning to
— never gate on absolute dBFS when a scale-invariant statistic exists
([The Analog Edge Part 13]({{ '/blog/tutorials/analog-edge-13-coherence-not-dbfs/' | relative_url }}))
— has a blunt form here: the most scale-invariant statistic a digital
channel offers is a frame that decoded. The level squelch stays as the cheap
first test; the gate is ANDed onto it.

## The NAC gate: CTCSS for P25

Every P25 data unit opens with a 24-dibit frame sync and a
[NID]({{ '/reference/p25-nid-duid/' | relative_url }}): the 12-bit
[Network Access Code]({{ '/reference/network-access-code/' | relative_url }})
and a 4-bit data-unit ID under a BCH(63,16) code
([P25 End to End Part 2]({{ '/blog/deep-dives/p25-end-to-end-02-sync-nid-lock/' | relative_url }})).
A radio uses the NAC the way an analog radio uses a CTCSS tone, and
`NACDetector` gives the scan list the same thing:

```yaml
scanner:
  conventional:
    - label: "VCALL10"
      frequency_hz: 155752500
      mode: p25
      squelch_dbfs: -50
      nac: "293"                  # three hex digits; "F7E" = any NAC, P25 only
      p25_phase1_demod_mode: c4fm # or cqpsk for a linear (LSM) transmitter
```

The detector takes the tone detectors' front end before its discriminator
— IQ decimated to about 48 kHz behind the ±8 kHz channel filter — and feeds
the production P25 Phase 1 receiver. `scan` walks the dibits:

```go
// internal/scanner/conventional/nac.go (shape)
if syncMismatch(d.dibits[pos:pos+len(phase1.FrameSyncWord)]) > nacSyncTolerance {
    continue
}
nid, err := phase1.ParseFrameNID(d.dibits[pos:])
if err != nil {
    continue
}
if d.any || nid.NAC == d.want {
    d.lastMatch = d.samples
} else {
    d.lastMatch = -1 // another system keyed up: shut now, not after the hold
    d.sawForeign = true
}
```

Three rules fall out of those lines. The sync may miss by four dibits
(`nacSyncTolerance`, the LDU assembler's default), because a false sync
almost never survives the BCH decode behind it. A matching NID keeps the
gate present for `nacHold` = 750 ms — a voice transmission carries a NID on
every LDU, one each 180 ms, so the hold rides out three or four frames lost
to a fade. And a NID with any *other* NAC closes the gate immediately: when
a different system keys up mid-dwell, the hangtime countdown starts on its
first frame.

`ParseNAC` reads hex with or without a `0x` prefix — a bare `"293"` is
0x293, never decimal. `F7E` and `F7F`, the values a radio programs to
receive any NAC, open the gate on every NAC but still only on a decoded P25
NID: a "P25 only" filter. The scanner logs `conv: NAC gate opened` with the
decoded `nac`, and at DEBUG `conv: P25 carrier with another NAC; channel
skipped`.

That matters for the gap this gate leaves. A `p25` channel **without** `nac`
has no gate at all: any carrier above `squelch_dbfs` opens it. It has no
noise squelch either — that statistic
([Part 3]({{ '/blog/deep-dives/squelch-03-noise-squelch/' | relative_url }}))
is built only for `fm` and `nfm` — so under automatic gain an unfiltered
`p25` channel is the one place on the list where an absolute level is still
the whole decision. `nac: F7E` is the remedy.

## The DMR call gate: three kinds of evidence

`DMRCallDetector` asks a different question: not *whose* traffic this is,
but whether the carrier holds traffic at all. It runs the production DMR
receiver on the same front end — channel filter on, the Tier II pipeline's
clock gain — hunts the 24-dibit sync words with the Tier II slicer's
tolerance of 2, and judges each burst it can cut around a sync. Three things
count as a call:

- a data burst whose [slot type]({{ '/reference/dmr-slot-type/' | relative_url }})
  decodes **without correction** as a Voice LC Header;
- the same for a PI header, the privacy header an encrypted call opens with;
- a voice sync — burst A of a voice superframe, once every 360 ms per slot.

Each one stamps `lastCall`, and the gate is present while that stamp is
younger than `dmrCallHold` = 1 s: long enough to ride out two lost
superframe syncs, short enough that a finished call does not hold the
channel much past its terminator. An idle repeater produces none of the
three: its Idle bursts and CSBKs carry other slot types, so the gate reads
them, learns the polarity from them, and stays shut.

Why "without correction"?
[DMR End to End Part 4]({{ '/blog/deep-dives/dmr-end-to-end-04-fec-stack-forged-terminator/' | relative_url }})
measured it: the slot type's Golay code decodes about a third of arbitrary
20-bit words to *some* codeword, so a slot type that needed fixing is weak
evidence of anything.

## Polarity: when a beacon looks like voice

There is a trap in "a voice sync is evidence".
[DMR's sync words]({{ '/reference/dmr-sync-patterns/' | relative_url }})
come in data/voice pairs that are each other's image under a discriminator
polarity flip
([DMR End to End Part 2]({{ '/blog/deep-dives/dmr-end-to-end-02-bursts-sync-polarity/' | relative_url }})).
On a spectrum-inverted front end, every Idle burst's *data* sync arrives
looking exactly like a *voice* sync — and an idle beacon would open the gate
it exists to keep shut.

So the gate fixes the stream's polarity the way the Tier II slicer does.
For each of the two candidate polarities it asks whether the sync is a data
sync there (`dmr.SyncIsDataAtPolarity`), rotates the burst and parses the
slot type. A clean parse is a vote; `dmrPolarityLockVotes` = 2 votes at one
polarity lock it — a beacon supplies those within a few tens of
milliseconds — and from then on every sync is classified at that polarity.

The interesting cases come before the lock, while the receiver is still
acquiring and every burst carries bit errors:

- A slot type that parses only **with** correction decides nothing — but it
  ends the classification, so that sync is not counted as voice.
- A voice-looking sync with no data reading is remembered, not trusted. A
  second within the hold makes a call (a superframe repeats its sync every
  360 ms); a lone one could be a garbled inverted data burst.

`TestDMRCallDetectorIgnoresIdleBeacon` runs 2.4 s of beacon at both
polarities and requires the gate never to open *and* the polarity to end up
locked. `TestDMRCallDetectorOpensOnCallAndClosesAfter` puts a call between
two beacons: shut before, open for at least 80 % of the call, shut again
within 1.2 s of its end.

## One scan window to acquire

A gate that is right but slow is still wrong, because the scanner leaves.
[DMR End to End Part 9]({{ '/blog/deep-dives/dmr-end-to-end-09-direct-mode-carrier-gate/' | relative_url }})
described the receiver's feed-forward timing acquisition: after a carrier
*absence* it estimates the symbol phase from the eye instead of waiting for
the Mueller-Müller loop to crawl there. That trigger never fires for a scan
list. The scanner retunes onto a repeater carrier that is usually already
up, the stream is present from its first sample, and the loop starts at an
arbitrary sub-symbol phase.

Measured at the scanner's 2.4 MS/s, some of those phases decoded **0 of 60**
sync words; from near a half-symbol offset the loop needs seconds, and the
window is half of one. `ArmTimingAcquisition()` is the opt-in for exactly
this caller — the same acquisition, armed without a prior absence — and the
gate arms it after `New` and after every `Reset`. With it the count is
**59–60 of 60 at every phase**; `TestDMRCallDetectorAcquiresAtAnySymbolPhase`
requires a call to open the gate from ten start phases.

## How the scanner drives a gate

The NAC gate arrived first and plugs in as a tone detector: the scanner
feeds it the chunks that clear the level squelch. It buys its acquisition
time the way CTCSS does — `minToneDwell` raises the scanner's
`MinDwellPerChannel` to at least `nacMinDwell` (500 ms). That is one option
for the whole list, so a NAC filter anywhere on it lengthens every visit
that does not open.

The DMR gate needed two things a tone detector does not have, and they
became an interface:

```go
// internal/scanner/conventional/scanner.go
type callGate interface {
    toneDetector
    gateName() string
    minDwell() time.Duration
}
```

`scanWindow` type-asserts its detector to `callGate` and changes two
behaviours. A call gate is fed **every** chunk, not only the loud ones,
because its receiver needs the quiet stretches too — a simplex handheld's
TDMA gaps, its own loops settling. And instead of slowing the whole list,
it spends time only where a carrier is: once a chunk clears the power
squelch, the window may be extended **once**, to `minDwell()` measured from
its start. A silent channel costs no extra time; a beacon costs
`dmrGateMinDwell` = 500 ms and the scan moves on. The window opens only when
the power test and the gate agree, logging `conv: call gate opened` with
the gate's name.

<figure class="lab-figure">
<svg viewBox="0 0 680 230" width="680" height="230" role="img" aria-label="A timeline of one DMR repeater channel: an idle beacon of Idle bursts, a call made of Voice LC Headers and voice superframes, then the idle beacon again. In-channel power sits above the squelch line the whole time, while the call gate is present only from the first header until one second after the last voice sync.">
  <text x="340" y="14" text-anchor="middle" fill="currentColor" font-size="10" font-weight="bold">one keyed DMR repeater: beacon · call · beacon</text>
  <text x="56" y="43" text-anchor="end" fill="var(--fg-muted)" font-size="8">on air</text>
  <rect x="60" y="28" width="170" height="24" fill="none" stroke="var(--fg-muted)" stroke-dasharray="3 2"/>
  <text x="145" y="43" text-anchor="middle" fill="var(--fg-muted)" font-size="8">Idle bursts, both slots</text>
  <rect x="230" y="28" width="20" height="24" fill="none" stroke="var(--accent)" stroke-width="1.5"/>
  <rect x="250" y="28" width="240" height="24" fill="none" stroke="currentColor" stroke-width="1.5"/>
  <text x="370" y="43" text-anchor="middle" fill="currentColor" font-size="8">voice superframes (one slot)</text>
  <rect x="490" y="28" width="160" height="24" fill="none" stroke="var(--fg-muted)" stroke-dasharray="3 2"/>
  <text x="570" y="43" text-anchor="middle" fill="var(--fg-muted)" font-size="8">Idle bursts, both slots</text>
  <text x="240" y="64" text-anchor="middle" fill="var(--accent)" font-size="8">Voice LC Headers</text>
  <text x="56" y="89" text-anchor="end" fill="var(--fg-muted)" font-size="8">power</text>
  <line x1="60" y1="82" x2="650" y2="82" stroke="currentColor" stroke-width="1.5"/>
  <line x1="60" y1="98" x2="650" y2="98" stroke="var(--fg-muted)" stroke-dasharray="3 3"/>
  <text x="646" y="110" text-anchor="end" fill="var(--fg-muted)" font-size="8">squelch_dbfs — cleared the whole time</text>
  <text x="56" y="139" text-anchor="end" fill="var(--fg-muted)" font-size="8">evidence</text>
  <line x1="60" y1="140" x2="650" y2="140" stroke="var(--fg-muted)"/>
  <line x1="234" y1="128" x2="234" y2="140" stroke="var(--accent)" stroke-width="1.5"/>
  <line x1="242" y1="128" x2="242" y2="140" stroke="var(--accent)" stroke-width="1.5"/>
  <line x1="250" y1="128" x2="250" y2="140" stroke="currentColor" stroke-width="1.5"/>
  <line x1="279" y1="128" x2="279" y2="140" stroke="currentColor" stroke-width="1.5"/>
  <line x1="308" y1="128" x2="308" y2="140" stroke="currentColor" stroke-width="1.5"/>
  <line x1="337" y1="128" x2="337" y2="140" stroke="currentColor" stroke-width="1.5"/>
  <line x1="366" y1="128" x2="366" y2="140" stroke="currentColor" stroke-width="1.5"/>
  <line x1="395" y1="128" x2="395" y2="140" stroke="currentColor" stroke-width="1.5"/>
  <line x1="424" y1="128" x2="424" y2="140" stroke="currentColor" stroke-width="1.5"/>
  <line x1="453" y1="128" x2="453" y2="140" stroke="currentColor" stroke-width="1.5"/>
  <text x="366" y="154" text-anchor="middle" fill="currentColor" font-size="8">voice sync, burst A — every 360 ms</text>
  <text x="145" y="154" text-anchor="middle" fill="var(--fg-muted)" font-size="8">none: Idle is not evidence</text>
  <text x="56" y="187" text-anchor="end" fill="var(--fg-muted)" font-size="8">gate</text>
  <rect x="234" y="176" width="299" height="16" fill="none" stroke="var(--accent)" stroke-width="1.5"/>
  <text x="383" y="188" text-anchor="middle" fill="var(--accent)" font-size="8">present</text>
  <line x1="453" y1="200" x2="533" y2="200" stroke="var(--fg-muted)"/>
  <text x="493" y="212" text-anchor="middle" fill="var(--fg-muted)" font-size="8">dmrCallHold = 1 s</text>
  <text x="145" y="188" text-anchor="middle" fill="var(--fg-muted)" font-size="8">shut</text>
  <text x="600" y="188" text-anchor="middle" fill="var(--fg-muted)" font-size="8">shut</text>
</svg>
<figcaption>The power squelch cannot tell the three segments apart; the gate is present only where a header or a voice sync was heard within the last second.</figcaption>
</figure>

Inside the dwell the same rule holds: a call gate sees every chunk, and a
chunk is active only when the power test *and* the gate say so. When the
call ends and the beacon resumes, the gate goes absent within its hold and
the ordinary hangtime finishes the job — the scanner releases a repeater
that never unkeys.

### How call gates shaped the Go code

- **A gate is a detector plus two methods.** `callGate` embeds
  `toneDetector`, so the per-channel `detectors` slice and the reset on
  every retune needed no second mechanism.
- **Evidence is stamped in input samples, not wall time.** `lastMatch` and
  `lastCall` are sample indices, so a gate behaves the same at any chunk
  size — and in a test running slower than real time.
- **Gates borrow constants, they do not invent them.** The sync tolerances
  are the LDU assembler's and the Tier II slicer's, the clock gain the
  Tier II pipeline's; a gate that framed differently from the decoder
  behind it would open on calls that decoder cannot hear.
- **Reset means a cold start.** `DMRCallDetector.Reset` resets the receiver,
  re-arms timing acquisition, rebuilds the sync detector and forgets the
  polarity.

What this does not have is air. `TestScannerDMRIdleBeaconOpensNoCall` and
`TestScannerDMRCallStillOpens` run the whole scanner at 2.4 MS/s,
`TestDMRCallDetectorOpensOnDirectModeCall` covers a simplex handheld, and
`TestP25ChannelNACFilter` is the #1262 regression — but every one of those
signals was built by the test. Both gates are **synthetic-verified**;
neither has a capture or a reporter's live run behind it yet.

## Where this goes next

P25 and DMR have long sync words and a field that names the frame. The
other three digital modes on the list each needed their own definition of
evidence.
[Part 9]({{ '/blog/deep-dives/squelch-09-call-gates-ysf-nxdn-dstar/' | relative_url }})
covers them: System Fusion, where one sync word is enough; NXDN, where one
is nowhere near enough; and D-STAR, where the gate runs two receivers.

## FAQ

**Why does my scanner stop on a DMR repeater when nobody is talking?**
A keyed repeater transmits Idle bursts or CSBK beacons between calls — a
strong carrier that clears any power squelch. GopherTrunk's `dmr` channels
also require a Voice LC Header, a PI header or a voice-burst sync within
the last second. A beacon produces none, so the scan moves on after the
gate's half-second window.

**How do I make a P25 scan-list channel ignore another agency on the same frequency?**
Set `nac` on the channel to your system's three-hex-digit NAC, for example
`nac: "293"`. The channel then opens, and stays open, only while it decodes
NIDs carrying that NAC; a different NAC is skipped, and one that keys up
mid-call shuts the gate on its first frame.

**What does nac F7E do on a scan-list channel?**
`F7E` (and `F7F`) accept any NAC but still require a decoded P25 NID, so the
channel opens on any P25 transmission and on nothing else. Use it under
automatic gain: a `p25` channel without `nac` opens on signal level alone.

**Does the DMR call gate work on a spectrum-inverted receiver?**
Yes. DMR's data and voice sync words are each other's polarity flip, so the
gate locks the stream's polarity from two cleanly decoded slot types and
classifies every sync at that polarity.

**Are the P25 NAC gate and the DMR call gate verified on air?**
Not yet. Both are synthetic-verified: their tests modulate P25 LDUs and DMR
bursts — an idle beacon at both polarities, a direct-mode call — and run the
full scanner at 2.4 MS/s. No capture or live run is recorded for either.

## Series navigation

**Part 8 of 14** · ←
[Part 7: AM — A Carrier-to-Noise Squelch and a Carrier That Drifts]({{ '/blog/deep-dives/squelch-07-am/' | relative_url }})
· Next →
[Part 9: Call Gates II — System Fusion, NXDN and D-STAR]({{ '/blog/deep-dives/squelch-09-call-gates-ysf-nxdn-dstar/' | relative_url }})
