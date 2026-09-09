---
title: "The Operator's Cookbook, Part 7: Many Systems, One Box — The SDR Pool"
description: "A complete GopherTrunk recipe for monitoring several trunked systems from one daemon: pinning dongles by serial, control vs voice vs wideband roles, sizing the shared voice pool against concurrent calls, and the priority, scan-list and preemption rules that decide who gets a radio."
category: tutorials
keywords: multiple sdr dongles scanner, monitor multiple trunked systems, sdr voice pool sizing, talkgroup priority preemption, scan list config sdr, rtl-sdr serial assignment, usb bandwidth multiple sdr, trunking scanner multi system, gophertrunk cookbook
tags: [operator-cookbook, sdr-pool, priority, scanning, config, hardware]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "The Operator's Cookbook"
series_part: 7
---

*Part 7 of **The Operator's Cookbook**, a 14-part series of complete,
copy-paste GopherTrunk builds — one working rig per part, antenna to browser.
[Part 6]({{ '/blog/tutorials/operator-cookbook-06-analog-fm-tone-out/' | relative_url }})
added analog FM and pager tones to the fleet of things one box can hear.
Every recipe so far ran one system on one or two dongles; real monitoring
rarely stays that small. This part is the scale-out recipe: several dongles,
several systems — P25 county, DMR Tier III utility, and whatever else your
area runs — behind **one daemon and one shared voice pool**, with the
priority and preemption rules that decide, when calls outnumber radios,
which call wins.*

> **TL;DR:** Pin every dongle by **serial** and give it a role: a
> `role: wideband` device per system carries that system's control channel
> as a `channels:` tap plus `voice_taps: N` in-window voice slots, and one
> `role: voice` dongle backstops out-of-window grants for *all* systems.
> Contention is policy, not luck: talkgroup CSV `Priority` (1 highest, 10
> lowest) drives preemption — higher preempts lower, equal never preempts,
> emergency preempts everything, `Lockout` drops a grant entirely —
> and `scanner.scan_mode: list` follows only talkgroups marked `Scan`.
> Watch startup for one `device opened … serial=… role=…` line per dongle,
> and treat `no voice SDR available; voice grants will be dropped` as your
> sizing alarm.

**Key takeaways**

- **Serials are the only stable identity.** With several identical dongles,
  "first-found" ordering is a lottery — burn unique serials into your sticks
  (`rtl_eeprom`) and pin every config entry, or a reboot reshuffles roles.
- **Wideband taps run in parallel; the cc-hunt runs in series.** Each
  wideband `channels:` entry gets its own decoder simultaneously; systems
  left to a single `role: control` dongle are hunted one at a time by the
  supervisor. Parallel monitoring means a wideband tap per system.
- **The voice pool is shared, and preemption is the safety valve.** All
  voice capacity — physical voice dongles and every `voice_taps` slot —
  serves every system, allocated per grant by talkgroup priority.
- **Sizing shows up in the log, not in a crash.** An undersized pool drops
  grants with a specific, greppable WARN; count those lines before buying
  another dongle.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| Device identity | match config to hardware by serial | `sdr.devices[].serial`, `gophertrunk sdr list` |
| Roles | control / voice / wideband / auto | `sdr.devices[].role` |
| Per-system CC taps | parallel control decode on one capture | `role: wideband` + `channels:` ([voice taps]({{ '/reference/wideband-voice-taps/' | relative_url }})) |
| Voice capacity | in-window slots + physical backstop | `voice_taps`, a `role: voice` device |
| Priority & lockout | who wins a scarce radio | talkgroup CSV `Priority` / `Lockout` columns |
| Scan list | follow only flagged talkgroups | `scanner.scan_mode: list` + CSV `Scan` column ([priority scan]({{ '/reference/priority-scan/' | relative_url }})) |
| Hotplug survival | re-acquire vanished dongles by serial | `sdr.watchdog_interval_ms` |

## In this post

- **What you're building** — three systems, four dongles, one cockpit.
- **The shopping list** — dongles multiply; USB rules multiply faster.
- **The config** — serials, roles, taps and a prioritized talkgroup file.
- **First run — what healthy looks like** — the pool assembling itself.
- **When it doesn't work** — starvation, stealing, shuffling, stalling.
- **Variations** — one-dongle rotation, multi-site P25, scan-list mode.

## What you're building

The finished rig monitors a P25 system and a DMR Tier III system
*simultaneously* — both control channels decoding at once, calls from either
landing in one History panel, one recordings tree, one live-audio cockpit —
with a third analog-capable dongle in reserve for voice grants that fall
outside the wideband windows. The machinery underneath is the **SDR pool**:
a registry of opened devices, keyed by serial, that decoders and voice
chains borrow from and return to — the
[pool & concurrency deep dive]({{ '/blog/deep-dives/sdr-internals-03-sdr-pool-streaming-concurrency/' | relative_url }})
is its full story.

The design question this recipe answers is *contention*. Two systems can
easily produce five simultaneous calls against three voice slots. GopherTrunk
resolves that with the trunking engine's
[voice pool]({{ '/blog/deep-dives/trunking-engine-04-voice-pool/' | relative_url }})
and its
[priority & preemption]({{ '/blog/deep-dives/trunking-engine-05-priority-preemption/' | relative_url }})
rules — policy you write in the talkgroup CSV, enforced per grant.

<figure class="lab-figure">
<svg viewBox="0 0 680 250" width="680" height="250" role="img" aria-label="The multi-system SDR pool: three dongles pinned by serial feed the pool — a wideband dongle decoding the P25 control channel with three voice tap slots, a second wideband dongle decoding the DMR Tier III control channel with two voice taps, and a physical voice dongle; both control decoders publish grants to the trunking engine, whose priority gate allocates voice capacity from the shared pool and preempts lower-priority calls when slots run out">
  <rect x="8" y="24" width="118" height="40" rx="4" fill="none" stroke="currentColor"/>
  <text x="67" y="40" text-anchor="middle" fill="currentColor" font-size="10">dongle A (wideband)</text>
  <text x="67" y="53" text-anchor="middle" fill="var(--fg-muted)" font-size="9">P25 CC tap + 3 voice taps</text>
  <rect x="8" y="104" width="118" height="40" rx="4" fill="none" stroke="currentColor"/>
  <text x="67" y="120" text-anchor="middle" fill="currentColor" font-size="10">dongle B (wideband)</text>
  <text x="67" y="133" text-anchor="middle" fill="var(--fg-muted)" font-size="9">DMR T3 CC tap + 2 voice taps</text>
  <rect x="8" y="184" width="118" height="40" rx="4" fill="none" stroke="currentColor"/>
  <text x="67" y="200" text-anchor="middle" fill="currentColor" font-size="10">dongle C (voice)</text>
  <text x="67" y="213" text-anchor="middle" fill="var(--fg-muted)" font-size="9">out-of-window backstop</text>
  <line x1="126" y1="44" x2="168" y2="90" stroke="currentColor"/>
  <line x1="126" y1="124" x2="168" y2="124" stroke="currentColor"/>
  <line x1="126" y1="204" x2="168" y2="158" stroke="currentColor"/>
  <rect x="168" y="84" width="120" height="80" rx="6" fill="none" stroke="var(--fg-muted)" stroke-dasharray="4 3"/>
  <text x="228" y="112" text-anchor="middle" fill="var(--fg-muted)" font-size="10">SDR pool</text>
  <text x="228" y="126" text-anchor="middle" fill="var(--fg-muted)" font-size="9">keyed by serial,</text>
  <text x="228" y="138" text-anchor="middle" fill="var(--fg-muted)" font-size="9">watchdog re-acquires</text>
  <line x1="288" y1="124" x2="330" y2="124" stroke="currentColor"/>
  <rect x="330" y="24" width="150" height="56" rx="4" fill="none" stroke="var(--accent)"/>
  <text x="405" y="46" text-anchor="middle" fill="var(--accent)" font-size="10">P25 + DMR CC decoders</text>
  <text x="405" y="60" text-anchor="middle" fill="var(--fg-muted)" font-size="9">grants, in parallel</text>
  <rect x="330" y="104" width="150" height="90" rx="4" fill="none" stroke="var(--accent)"/>
  <text x="405" y="126" text-anchor="middle" fill="var(--accent)" font-size="10">trunking engine</text>
  <text x="405" y="142" text-anchor="middle" fill="var(--fg-muted)" font-size="9">priority gate:</text>
  <text x="405" y="155" text-anchor="middle" fill="var(--fg-muted)" font-size="9">higher preempts lower,</text>
  <text x="405" y="168" text-anchor="middle" fill="var(--fg-muted)" font-size="9">emergency beats all,</text>
  <text x="405" y="181" text-anchor="middle" fill="var(--fg-muted)" font-size="9">lockout drops the grant</text>
  <line x1="405" y1="80" x2="405" y2="104" stroke="var(--accent)"/>
  <line x1="480" y1="149" x2="522" y2="149" stroke="currentColor"/>
  <rect x="522" y="104" width="148" height="90" rx="4" fill="none" stroke="currentColor"/>
  <text x="596" y="126" text-anchor="middle" fill="currentColor" font-size="10">shared voice capacity</text>
  <text x="596" y="142" text-anchor="middle" fill="var(--fg-muted)" font-size="9">A: 3 taps · B: 2 taps</text>
  <text x="596" y="155" text-anchor="middle" fill="var(--fg-muted)" font-size="9">C: physical voice SDR</text>
  <text x="596" y="176" text-anchor="middle" fill="var(--fg-muted)" font-size="9">→ recordings + web console</text>
</svg>
<figcaption>Both control channels decode in parallel; the engine allocates one shared pot of voice capacity per grant, with talkgroup priority deciding who gets — and who loses — a radio.</figcaption>
</figure>

## The shopping list

| Item | Price (rough) | Notes |
|---|---|---|
| 3–4 RTL-SDR dongles | ~$35 each | identical hardware is fine — serials make them distinct |
| Antennas / splitter | varies | one antenna per band beats one splitter per rig to start |
| Powered USB hub or rear-panel ports | ~$20 | each dongle streams ~2.4 MS/s continuously; bus power on a passive hub is where multi-dongle rigs die |
| Computer | $0 | CPU scales with taps — a modern quad-core carries this build easily |

Total: **well under $200** on top of what you own. Before configuring
anything, give every stick a unique serial (`rtl_eeprom -s 00000101` style)
and verify with `gophertrunk sdr list` — the
[USB]({{ '/reference/usb/' | relative_url }}) layer identifies dongles by
serial and nothing else ([hardware
checklist]({{ '/what-do-i-need-for-gophertrunk/' | relative_url }})).

## The config

```yaml
log:
  level: info

storage:
  path: "../data/calls.db"

recordings:
  dir: "../recordings"

sdr:
  sample_rate: 2_400_000
  watchdog_interval_ms: 30000
  devices:
    - serial: "00000101"            # P25 county
      role: wideband
      gain: "auto"
      center_freq_hz: 858_000_000
      voice_taps: 3
      channels:
        - frequency_hz: 857_262_500
          system: "County-P25"
    - serial: "00000102"            # DMR Tier III utility
      role: wideband
      gain: "auto"
      center_freq_hz: 465_000_000
      voice_taps: 2
      channels:
        - frequency_hz: 465_337_500
          system: "Utility-DMR"
    - serial: "00000103"            # shared out-of-window voice backstop
      role: voice
      gain: "auto"

scanner:
  scan_mode: list                   # follow only Scan=true talkgroups

trunking:
  systems:
    - name: "County-P25"
      protocol: p25
      control_channels: [857_262_500, 858_487_500]
      talkgroup_file: "../config/talkgroups-p25.csv"
    - name: "Utility-DMR"
      protocol: dmr
      control_channels: [465_337_500]
      talkgroup_file: "../config/talkgroups-dmr.csv"
      # no dmr_band_plan: the wideband dongle learns LCN→frequency
      # off the air — see Part 2
```

And the policy file — a few rows of `talkgroups-p25.csv`:

```
Decimal,Alpha Tag,Priority,Scan,Lockout
9001,Fire Dispatch,1,Y,
9014,PD Tac 2,3,Y,
9101,Schools Ops,,N,
9230,Water Dept,,,Y
```

How the pieces interact. **`Priority` is 1 (highest) to 10 (lowest), 0/empty
= unset** — and it's what preemption keys on. **`Scan`** matters only
because `scan_mode: list` is set: unlisted or `Scan=N` talkgroups are
observed on the control channel but never tie up a voice slot (Emergency
grants bypass the list). **`Lockout`** (or the classic `L` in the Priority
column) drops the grant entirely. All three are editable live from the web
UI too — [Trunking Engine Part
6]({{ '/blog/deep-dives/trunking-engine-06-talkgroups-scan-modes/' | relative_url }})
covers the semantics in depth.

**Voice sizing math:** this config has 3 + 2 in-window taps plus one
physical voice dongle = six concurrent calls before anything is dropped —
but taps only serve grants inside their own dongle's 2.4 MHz window, and the
DMR system needs its learned band plan before grants resolve to frequencies
at all ([Part 2]({{ '/blog/tutorials/operator-cookbook-02-dmr-tier3/' | relative_url }})).
The physical voice dongle is the flexible remainder: it can tune anywhere,
for either system.

## First run — what healthy looks like

Startup narrates the pool assembling — one line per stick, and a line for
anything it *didn't* take:

```
INF device opened driver=rtlsdr serial=00000101 role=wideband rate_hz=2400000 ppm=0 bias_tee=false
INF device opened driver=rtlsdr serial=00000102 role=wideband rate_hz=2400000 ppm=0 bias_tee=false
INF device opened driver=rtlsdr serial=00000103 role=voice rate_hz=2400000 ppm=0 bias_tee=false
INF skipping non-configured SDR; add its serial to sdr.devices to use it
```

Then both systems lock — in parallel, no taking turns — and grants start
flowing. The policy lines are the ones specific to this recipe:

```
INF grant locked out grant=... tg=Water Dept
INF call ended device=... reason=preempted
INF no voice device available for grant grant=...
```

The first is your `Lockout` column working. The second is priority doing its
job: a Priority-1 Fire Dispatch grant arrived with every slot busy, and the
engine tore down the lowest-priority active call to serve it. The rules,
exactly as the engine's tests pin them: **higher preempts lower; equal never
preempts** (a stable call holds its radio against peers); **emergency
preempts everything; a locked-out grant preempts nothing**. The third line
is your sizing meter — occasional during a genuine pile-up, but if it's
routine, add a voice dongle or raise `voice_taps` before touching anything
else. Two sharper WARNs distinguish *why* nothing was available:
`no voice SDR available; voice grants will be dropped` means the pool has no
voice capacity at all, and
`voice grant frequency outside every voice device's tuning window` means it
has capacity in the wrong place — widen a window or re-center it.

Pull a dongle mid-run to watch the watchdog earn its keep:

```
WRN sdr: watchdog: device missing from USB enumerate
INF sdr: watchdog: device reappeared; reacquiring
```

— re-acquired **by serial**, which is the deep reason this recipe insists on
unique ones. The [USB hotplug deep
dive]({{ '/blog/deep-dives/rf-front-end-12-sdr-pool-usb-watchdog/' | relative_url }})
explains the machinery.

## When it doesn't work

| Symptom | Likely cause | Fix |
|---|---|---|
| Roles land on the wrong dongles after reboot | duplicate/blank serials, first-found matching | Unique serials via `rtl_eeprom`, pin every entry. `matched configured SDR by partial serial` in the log means pin the *full* serial |
| `configured SDR not present on the bus; check the cable / dmesg / lsusb` | dead port, starved hub, wrong serial | Rear-panel ports or a powered hub; re-check `gophertrunk sdr list` |
| Frequent `no voice device available for grant` | pool undersized for real concurrency | Count concurrent calls in History at busy hour; raise `voice_taps` (CPU ~linear per tap, warns above 16) or add a voice dongle |
| High-priority calls missed during pile-ups | priorities unset, so nothing may preempt | Set `Priority` on the talkgroups you care about — an unset incoming grant never preempts a set one |
| Analog scan list (Part 6) stopped working after adding trunking | the conventional scanner takes the *last* voice device — trunking and the scan list now compete | Add a dedicated voice dongle for the scanner; the daemon auto-detects a spare |
| One system decodes, its co-tenant tap sits at the noise floor | shared front end: one gain for all taps on a dongle | `gain: "auto"` on multi-tap dongles (issue #749); a genuinely weak system deserves its own dongle |
| Dongles vanish under load, watchdog churns | USB bandwidth/power exhaustion | Spread across root hubs; powered hub; lower `sample_rate` if the hardware allows |
| `ccdecoder: decode can't keep up with real time` | CPU oversubscribed by tap count | Shed taps or systems, or move to a bigger box — this WARN means decode, not RF |

### How this recipe shapes operator practice

- **Write policy before you need it.** Priority and lockout only help during
  the pile-up you didn't predict; a CSV without a `Priority` column is a
  pool allocated by luck.
- **Grep, then buy.** The dropped-grant WARN is a precise demand signal —
  count it for a week before spending $35 on capacity you may not need.
- **One knob per dongle.** Every device block has exactly serial, role,
  center and taps; when the pool misbehaves, that's the whole search space.

## Variations

- **One dongle, many systems.** Skip wideband: list every system, give one
  dongle `role: control`, and the
  [cc-hunt supervisor]({{ '/blog/deep-dives/the-hunt-06-control-channel-hunting/' | relative_url }})
  rotates through them (`scanner.cc_hunt` dwell/backoff), sticky to the last
  lock. Sequential, not parallel — reconnaissance mode, not monitoring mode.
- **Multi-site P25 on one capture.** Several sites of *one* system fit as
  multiple `channels:` entries on one wideband dongle — all decoded in
  parallel ([multisite trunking]({{ '/reference/multisite-trunking/' | relative_url }}),
  and [The Hunt Part 9]({{ '/blog/deep-dives/the-hunt-09-wideband-multisite-p25/' | relative_url }})
  for the field story).
- **`scan_mode: all`** — drop the scan list and follow everything
  non-locked-out; right for archival builds (Part 10), wrong for a small
  pool on busy systems.
- **`role: auto`** lets the daemon assign a spare stick where needed —
  convenient on a bench, too nondeterministic for a production box.

## Where this goes next

Every dongle so far plugs into the box that decodes it. [Part
8]({{ '/blog/tutorials/operator-cookbook-08-remote-radios/' | relative_url }})
cuts that cable: SoapyRemote, rtl_tcp and ka9q-radio put the radio at the
antenna and the decoder in the rack — with network budgets, antenna-port
validation, and the drop counters that tell you which side of the wire is
actually struggling.

## FAQ

**How many trunked systems can GopherTrunk monitor at once?**
As many as you give it capture for: each wideband `channels:` tap runs its
own control-channel decoder in parallel, so the practical limits are dongles,
USB bandwidth, and CPU (~linear per tap). Systems sharing a single
`role: control` dongle are instead hunted one at a time.

**How many voice dongles do I need for a trunked system?**
Count concurrent calls, not talkgroups. Each wideband dongle's `voice_taps`
serve in-window grants free of extra hardware; one physical voice dongle
backstops the rest. When `no voice device available for grant` shows up
routinely, that's the signal to add capacity.

**How does talkgroup priority work in GopherTrunk?**
`Priority` in the talkgroup CSV runs 1 (highest) to 10 (lowest). When a
grant arrives and every voice slot is busy, a higher-priority grant preempts
the lowest-priority active call; equal priorities never preempt; emergency
grants preempt everything; locked-out talkgroups never follow at all.

**Can I mix protocols on one SDR dongle?**
Yes — a wideband dongle's `channels:` list can mix P25, DMR, NXDN and more,
each entry pointing at its own system, provided every frequency fits the
dongle's `center ± sample_rate/2` window (with a 5% edge guard).

**Why do my dongles swap roles when I reboot?**
Identical dongles ship with identical serials, and a blank `serial:` matches
first-found — enumeration order isn't stable across boots. Write unique
serials with `rtl_eeprom`, pin them in `sdr.devices`, and the pool (and its
hotplug watchdog) becomes deterministic.

## Series navigation

**Part 7 of 14** · ←
[Part 6: Analog FM & Tone-Out Paging]({{ '/blog/tutorials/operator-cookbook-06-analog-fm-tone-out/' | relative_url }})
· Next →
[Part 8: Radios Far Away — SoapyRemote, rtl_tcp & ka9q]({{ '/blog/tutorials/operator-cookbook-08-remote-radios/' | relative_url }})
