---
title: "The Operator's Cookbook, Part 4: A TETRA TMO Rig"
description: A complete GopherTrunk build for a TETRA trunked-mode network — one dongle camped on a 25 kHz control carrier, the CC equalizer that now ships enabled, MCC/MNC identity, four-slot same-carrier voice through the clean-room ACELP vocoder, and the sync-loss troubleshooting that came from real field logs.
category: tutorials
keywords: tetra scanner setup, tetra sdr decoder, tetra tmo trunked mode, tetra control channel lock, tetra acelp voice decode, tetra mcc mnc, pi/4 dqpsk sdr, tetra encrypted network metadata, gophertrunk cookbook
tags: [operator-cookbook, tetra, tmo, dqpsk, acelp, config]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "The Operator's Cookbook"
series_part: 4
---

*Part 4 of **The Operator's Cookbook**, a 14-part series of complete,
copy-paste GopherTrunk builds — one working rig per part, antenna to browser.
[Part 3]({{ '/blog/tutorials/operator-cookbook-03-conventional-dmr-two-slots/' | relative_url }})
squeezed two conversations out of one conventional DMR repeater. This part
leaves the 4-level-FSK family entirely for TETRA — π/4-DQPSK, a downlink that
never stops transmitting, and the protocol GopherTrunk has spent more
hard-won engineering on than any other. The payoff for you: the recipe is
*short*, because the difficult parts — the control-channel equalizer, the
clean-room ACELP vocoder, the sync-loss watchdogs — now ship as defaults.*

> **TL;DR:** `protocol: tetra` plus the cell's main-carrier frequency in
> `control_channels` is a working TETRA TMO rig on one `role: control`
> dongle: GopherTrunk channelizes the 25 kHz carrier to 144 kHz, runs the
> **blind CMA equalizer that's now default on the CC path** (it lifted a
> marginal real-world control channel from ~12% to ~100% CRC-clean BSCH),
> decodes the cell identity, and registers **four same-carrier voice taps** —
> one per TDMA slot — feeding the clean-room ACELP vocoder
> (`vocoder=tetra-acelp`). Healthy looks like
> `tetra cc locked freq=… mcc=… mnc=… la=…` and calls in the browser;
> the classic failure signature (locked but silent, `bsch_ok` high with
> `sch_pdus=0`) now self-heals via the payload watchdog.

**Key takeaways**

- **TETRA is a different physical layer, same recipe.** π/4-DQPSK at 18000
  symbols/s in a 25 kHz channel — nothing like P25/DMR's FSK — but the config
  shape you learned in Parts 1–3 carries over unchanged.
- **The equalizer is on because real networks needed it.** Field captures
  showed marginal control channels (~10 dB in-channel SNR) locking but
  decoding ~22% of their sync bursts; the default-on CMA equalizer recovers
  them. You configure nothing.
- **One carrier can be the whole cell.** A single-carrier TETRA site keeps
  voice on the control carrier's other TDMA slots, so the same-carrier taps
  give you up to four concurrent calls with no voice radio.
- **Encrypted networks still yield the map.** TEA-encrypted voice is not
  decryptable, but the control channel is clear — talkgroups, identities and
  activity all decode. Know what your local law permits before you camp one.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| System definition | trunked TETRA cell, camped CC | `protocol: tetra`, `control_channels` |
| Cell identity | MCC / MNC / location area, decoded + displayed | `tetra cc locked` log line; Systems panel — [MNI]({{ '/reference/tetra-mobile-network-identity/' | relative_url }}) |
| CC equalizer | blind CMA, default-on for the TETRA CC path | nothing to configure; [the equalizer story]({{ '/blog/deep-dives/tetra-end-to-end-10-control-channel-sync-loss/' | relative_url }}) |
| Voice | four same-carrier slot taps → ACELP | automatic; [ACELP]({{ '/reference/acelp/' | relative_url }}), [clean-room vocoder]({{ '/blog/deep-dives/tetra-end-to-end-06-clean-room-acelp/' | relative_url }}) |
| Decode health | per-interval counters at debug | `tetra: decode status` (`bsch_ok`, `sch_pdus`, `grants`); cadence via `tetra_status_interval_secs` |
| Talkgroup names | GSSIs in the same CSV format | `talkgroup_file` |
| Is TETRA for you? | region + hardware reality check | [TETRA scanner guide]({{ '/tetra-scanner/' | relative_url }}) |

## In this post

- **What you're building, and where it works** — region and legality, briefly.
- **The shopping list** — Part 1's again, with one frequency-band caveat.
- **The config** — two meaningful lines.
- **First run — what healthy looks like** — lock, identity, slots, voice.
- **When it doesn't work** — sync loss, carrier offset, encryption.
- **Variations** — multi-carrier cells, status cadence, capture taps.

## What you're building, and where it works

TETRA is the trunked-radio standard of most of the world outside North
America: public safety across Europe, plus transit, utilities, ports and
airports on several continents. If you're in the US, your local
public-safety system is not TETRA — but TETRA gear does show up in
commercial and industrial use, and [Part 5]({{ '/blog/tutorials/operator-cookbook-05-tetra-dmo/' | relative_url }})'s
direct-mode radios travel everywhere. One sentence on law, kept factual:
countries differ sharply on whether receiving public-safety traffic is legal
at all, and some prohibit even possession of a capable receiver — check your
jurisdiction before you camp a government network. The
[TETRA scanner guide]({{ '/tetra-scanner/' | relative_url }}) covers the
landscape, including why almost no consumer hardware scanner decodes TETRA
voice — which is precisely why this recipe exists.

Technically, a TETRA cell's **main carrier** is this build's target: a
25 kHz channel whose downlink transmits continuously, four TDMA slots per
frame, control signalling on one and — on single-carrier sites — voice calls
on the others. GopherTrunk channelizes it to 144 kHz (8 samples per symbol),
demodulates the π/4-DQPSK, and decodes everything from the sync bursts up.
The physics of that carrier fills its own series opener,
[TETRA End to End Part 1]({{ '/blog/deep-dives/tetra-end-to-end-01-pi4-dqpsk-carrier/' | relative_url }});
the cookbook just points a dongle at it.

<figure class="lab-figure">
<svg viewBox="0 0 680 250" width="680" height="250" role="img" aria-label="Signal chain of the TETRA TMO rig: antenna and RTL-SDR feed a downconverter to 144 kilohertz; a highlighted default-on CMA equalizer feeds the pi over four DQPSK receiver; the decoded dibit stream splits into the control-channel decoder reading BSCH and SCH signalling on slot one, and three same-carrier voice slot taps feeding the ACELP vocoder; outputs are the cell identity, recordings and the web console">
  <rect x="8" y="104" width="60" height="32" rx="4" fill="none" stroke="currentColor"/>
  <text x="38" y="124" text-anchor="middle" fill="currentColor" font-size="10">antenna</text>
  <line x1="68" y1="120" x2="92" y2="120" stroke="currentColor"/>
  <rect x="92" y="104" width="70" height="32" rx="4" fill="none" stroke="currentColor"/>
  <text x="127" y="124" text-anchor="middle" fill="currentColor" font-size="10">RTL-SDR</text>
  <line x1="162" y1="120" x2="186" y2="120" stroke="currentColor"/>
  <rect x="186" y="104" width="78" height="32" rx="4" fill="none" stroke="currentColor"/>
  <text x="225" y="118" text-anchor="middle" fill="currentColor" font-size="10">DDC</text>
  <text x="225" y="130" text-anchor="middle" fill="var(--fg-muted)" font-size="9">144 kHz</text>
  <line x1="264" y1="120" x2="288" y2="120" stroke="currentColor"/>
  <rect x="288" y="100" width="96" height="40" rx="4" fill="none" stroke="var(--accent)" stroke-width="2"/>
  <text x="336" y="116" text-anchor="middle" fill="var(--accent)" font-size="10">CMA equalizer</text>
  <text x="336" y="130" text-anchor="middle" fill="var(--fg-muted)" font-size="9">default ON</text>
  <line x1="384" y1="120" x2="408" y2="120" stroke="currentColor"/>
  <rect x="408" y="100" width="100" height="40" rx="4" fill="none" stroke="currentColor"/>
  <text x="458" y="116" text-anchor="middle" fill="currentColor" font-size="10">π/4-DQPSK RX</text>
  <text x="458" y="130" text-anchor="middle" fill="var(--fg-muted)" font-size="9">18000 sym/s</text>
  <line x1="508" y1="112" x2="546" y2="52" stroke="var(--accent)"/>
  <line x1="508" y1="128" x2="546" y2="180" stroke="currentColor"/>
  <rect x="546" y="30" width="124" height="44" rx="4" fill="none" stroke="var(--accent)"/>
  <text x="608" y="47" text-anchor="middle" fill="var(--accent)" font-size="10">CC decode (slot 1)</text>
  <text x="608" y="61" text-anchor="middle" fill="var(--fg-muted)" font-size="9">BSCH → MCC/MNC · grants</text>
  <rect x="546" y="158" width="124" height="58" rx="4" fill="none" stroke="currentColor"/>
  <text x="608" y="175" text-anchor="middle" fill="currentColor" font-size="10">voice slot taps ×4</text>
  <text x="608" y="189" text-anchor="middle" fill="var(--fg-muted)" font-size="9">TCH/S → ACELP</text>
  <text x="608" y="203" text-anchor="middle" fill="var(--fg-muted)" font-size="9">→ WAV + web console</text>
  <text x="230" y="60" fill="var(--fg-muted)" font-size="10">the downlink never stops: lock quality is</text>
  <text x="230" y="74" fill="var(--fg-muted)" font-size="10">measurable every frame, so the decode-status</text>
  <text x="230" y="88" fill="var(--fg-muted)" font-size="10">counters below are a continuous health meter</text>
  <text x="340" y="240" text-anchor="middle" fill="var(--fg-muted)" font-size="10">a single-carrier cell keeps voice on the control carrier's own slots — one dongle carries all four</text>
</svg>
<figcaption>The TETRA chain: the equalizer sits ahead of the receiver by default — on a real marginal control channel it was the difference between ~12% and ~100% CRC-clean sync bursts.</figcaption>
</figure>

## The shopping list

The [Part 1]({{ '/blog/tutorials/operator-cookbook-01-forty-dollar-p25-rig/' | relative_url }})
hardware transfers whole: one ~$35 RTL-SDR, an antenna, a computer. Two
caveats. TETRA lives mostly at 380–430 MHz (public safety and commercial),
so extend the kit whip accordingly, and expect ACELP + equalizer + four
potential slot decodes to want more CPU than a P25 rig — any recent laptop
is fine; the smallest Pis will feel it. If your target network sits at the
weak end, the [Analog Edge]({{ '/blog/tutorials/analog-edge-07-antennas/' | relative_url }})
antenna and feedline parts are the upgrade path; the numbers below tell you
whether you need them.

## The config

Every key verified against `config.example.yaml`:

```yaml
storage:
  path: "../data/calls.db"

recordings:
  dir: "../recordings"

sdr:
  sample_rate: 2_400_000
  devices:
    - serial: "00000001"
      role: control
      gain: "auto"

trunking:
  systems:
    - name: "Harbour-TETRA"
      protocol: tetra
      control_channels:
        - 419_512_500        # the cell's main carrier
      talkgroup_file: "../config/talkgroups-tetra.csv"   # optional (GSSIs)
```

That's the whole rig. Things you might expect to configure and don't: the
144 kHz channel rate (per-protocol, automatic), the equalizer (default-on for
the TETRA CC path), the colour code for descrambling (recovered from the
BSCH), and voice devices (four same-carrier slot taps are registered for
every TETRA system automatically). The optional
`tetra_status_interval_secs` sets how often the decode-status counters
accumulate before logging — leave it defaulted until the troubleshooting
table sends you there. Talkgroups in TETRA are GSSIs; the CSV format is the
same `Decimal`/`Alpha Tag` shape as every other protocol's.

## First run — what healthy looks like

Start the daemon with `log.level: debug` for the first session — TETRA's
continuous downlink makes the debug stream genuinely informative rather than
spammy. The lock line carries the decoded network identity:

```
INF tetra cc locked freq=419512500 mcc=262 mnc=1023 la=2101 system=Harbour-TETRA
```

`mcc`/`mnc` are the [Mobile Network Identity]({{ '/reference/tetra-mobile-network-identity/' | relative_url }})
— country code and network — and they appear in the web Systems panel too;
check them against what you expect for the network, because a strong adjacent
cell is a real possibility (see the table). Then the five-second heartbeat:

```
DBG tetra: decode status system=Harbour-TETRA locked=true carrier_off_hz=-412.3 baud=18000 sb_bursts=38 bsch_ok=37 bsch_fail=1 sysinfo=37 sch_pdus=214 sch_pdus_fail=6 grants=3 colour_code=27
```

Read it like a meter: `bsch_ok` near `sb_bursts` means sync is healthy;
`sch_pdus` climbing means the signalling payload is decoding (this is the
number that matters — more below); `grants` means traffic. When a call goes
out, the composer announces the slot-tap chain and the recorder does its
usual thing with a TETRA-specific vocoder name:

```
INF composer: tetra voice follow started — TCH/S decode + ACELP vocoder serial=cc:same-carrier:2 group=61432 timeslot=2 ...
INF recorder: call started ... tg=61432 ... vocoder=tetra-acelp
```

The voice you hear came through a clean-room ACELP implementation validated
bit-identical against the ETSI EN 300 395-2 reference codec — the
[conformance story]({{ '/blog/deep-dives/tetra-end-to-end-07-etsi-conformance/' | relative_url }})
is worth reading once, because it's why "TETRA voice sounds wrong" almost
never means the vocoder.

## When it doesn't work

TETRA's troubleshooting rows are unusually well-grounded: most came from
multi-hour operator field logs, diagnosed in the
[sync-loss deep dive]({{ '/blog/deep-dives/tetra-end-to-end-10-control-channel-sync-loss/' | relative_url }}).

| Symptom | Likely cause | Fix |
|---|---|---|
| No lock at all | wrong frequency, or the carrier isn't TETRA | `gophertrunk hunt -candidates` identifies it; TETRA carriers are 25 kHz and continuously keyed — easy to spot on the Spectrum panel |
| Locks, drops, `tetra: dsp resync (signal-time decode drought…)`, re-hunts | weak signal — the marginal regime. Field captures split cleanly: ~18 dB in-channel SNR decodes ~100% of BSCH, ~10 dB locks but decodes ~22% and storms resyncs | The equalizer already mitigates this; the rest is RF — antenna, feedline, siting. Raising software knobs will not help; raising the signal will |
| `bsch_ok` healthy, `sch_pdus=0`, no traffic ever | the "locked but deaf" signature — historically the AFC latching a wrong ±4.5 kHz alias bucket | Self-healing now: the payload watchdog forces a resync after 12 s of lock without payload and escalates to a re-hunt. If you still see it persist, that's a capture worth reporting |
| `ccdecoder: control carrier offset far from configured frequency` (issue #815) | you've locked an adjacent cell's stronger carrier, or the tuner is badly mistuned | Check the reported MCC/MNC/LA against your target; set the device `ppm`. The WARN requires the offset to persist 10 s, so it isn't estimator jitter |
| MCC/MNC in the lock line looks absurd | one corrupted sync burst can't rewrite identity — changes need two consecutive agreeing decodes — so a stable absurd value means you're genuinely on a different network | Believe the decoder; re-check which carrier you camped |
| Calls appear, recordings are silent or missing, talkgroups decode fine | the network encrypts voice (TEA air-interface encryption) | Nothing to fix: GopherTrunk decodes all clear signalling — talkgroups, identities, activity — but does not decrypt TETRA. Per-system `encrypted_calls: mode: metadata` keeps voice taps free |
| Garbled voice on an otherwise-strong signal | multipath/ISI on the traffic slots | The voice path runs its own equalizer; if it persists, capture IQ — the [soft-decision + equalizer work]({{ '/blog/deep-dives/tetra-end-to-end-09-equalizer-voice-path/' | relative_url }}) roughly doubled yield on exactly such captures |

The second row carries this series' standing RF sermon. The equalizer, the
soft-decision decoding, the watchdogs — all of them widen the marginal band,
none of them repeal it. **The decoder can only be as good as the samples**,
and the 10-vs-18 dB split above is what that abstraction costs in concrete
numbers on a real network.

### How this recipe shapes operator practice

- **`sch_pdus` is the truth; `bsch_ok` is just the handshake.** Sync bursts
  survive conditions that kill payload. A rig is healthy when payload
  counters climb, not when lock flags are green.
- **Identity is a checksum for your config.** MCC/MNC/LA in the lock line
  either match your intended network or you're camped somewhere else — check
  it on day one, before you name a single talkgroup.
- **Defaults encode field experience.** Every default this recipe leans on —
  equalizer, watchdogs, confirmation thresholds — exists because an operator's
  log demanded it. Overriding them is for A/B tests, not setup.

## Variations

- **Multi-carrier cells.** Bigger sites put voice on secondary carriers the
  same-carrier taps can't reach. Add a second dongle as `role: voice` and
  off-carrier grants bind it automatically — the same spill-over pattern as
  Parts 1–2.
- **Status cadence.** `tetra_status_interval_secs: 30` calms the debug
  heartbeat for long soak sessions; the counters accumulate over the window
  either way.
- **Capture-on-failure.** `baseband.auto_record` with `on_cc_sync_loss: true`
  and `tap: ddc` writes a small 144 kHz IQ capture at the exact moment a
  locked CC loses sync — the feature the sync-loss investigation was solved
  with, and the single best thing you can attach to a TETRA bug report.
- **Pre-stripped fixtures.** The `tetra_channel` / `tetra_channel_coding` /
  `tetra_colour_code` opt-out keys exist for feeding pre-decoded capture
  files, not live air — leave them alone on a real rig.

## Where this goes next

TMO assumed infrastructure: a base station, a cell, a downlink that never
stops. [Part 5]({{ '/blog/tutorials/operator-cookbook-05-tetra-dmo/' | relative_url }})
scans TETRA radios talking **directly to each other** — Direct Mode, where
there is no control channel, no talkgroup in the grant, and a colour-code
recovery problem interesting enough that the rig solves it statistically.
It's the newest and most honestly-caveated recipe in the series.

## FAQ

**Can an RTL-SDR really decode TETRA voice?**
Yes — a 25 kHz π/4-DQPSK carrier is comfortably within an RTL-SDR's
bandwidth and dynamic range, and GopherTrunk's TETRA path (144 kHz channel
rate, CMA equalizer, soft-decision channel decoding, clean-room ACELP) was
developed and validated on exactly such captures. Voice quality tracks
signal quality; the equalizer buys real margin but not miracles.

**What do MCC and MNC mean in the TETRA lock line?**
Mobile Country Code and Mobile Network Code — together the Mobile Network
Identity that names the network, like a cellular PLMN. GopherTrunk decodes
them from the broadcast sync channel and shows them in the log and the
Systems panel; they're your first check that you locked the network you
intended.

**Can GopherTrunk decode encrypted TETRA networks?**
It decodes everything that's clear — and on TEA-encrypted networks that's
the entire control channel: talkgroups, subscriber activity, cell topology.
Voice on such networks is not recoverable, and GopherTrunk performs no
key recovery. The `encrypted_calls` per-system policy stops encrypted calls
from tying up voice taps.

**Why does GopherTrunk process TETRA at 144 kHz when the channel is 25 kHz?**
144 kHz is a processing rate — 8 samples per symbol at 18000 symbols/s —
not a bandwidth. Every protocol gets channelized to its own designed rate
(the C4FM family gets 48 kHz); the receiver's filters and timing loops are
sized from that constant, which is what makes the decode rate-invariant to
whatever your dongle captured at.

**Do I need a separate voice SDR for TETRA?**
Not for a single-carrier cell: voice occupies the control carrier's other
TDMA slots, and GopherTrunk registers four same-carrier slot taps per TETRA
system automatically. Only multi-carrier sites — where grants point at
secondary carriers — need a `role: voice` dongle for the overflow.

## Series navigation

**Part 4 of 14** · ←
[Part 3: One Repeater, Two Conversations — Conventional DMR]({{ '/blog/tutorials/operator-cookbook-03-conventional-dmr-two-slots/' | relative_url }})
· Next →
[Part 5: TETRA Direct Mode — Scanning DMO]({{ '/blog/tutorials/operator-cookbook-05-tetra-dmo/' | relative_url }})
