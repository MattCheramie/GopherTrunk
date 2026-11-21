---
title: "The Operator's Cookbook Vol. 2, Part 3: An NXDN System — Trunked and Conventional"
description: "Two complete GopherTrunk NXDN recipes: a trunked system with protocol: nxdn and the nxdn_band_plan that turns a grant's channel number into a frequency, and conventional NXDN96 and NXDN48 channels on the scan list behind a call gate, with real lock and call lines and the verification status of each path."
category: tutorials
keywords: nxdn scanner sdr, nxdn trunking decoder, nexedge idas rtl-sdr, nxdn band plan config, nxdn48 nxdn96 scanner, nxdn control channel decode, conventional nxdn scan list, nxdn rdio scanner upload, gophertrunk nxdn
tags: [operator-cookbook, cookbook-volume-2, nxdn, trunking, scanning, config]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "The Operator's Cookbook, Volume 2"
series_part: 3
---

*Part 3 of **The Operator's Cookbook, Volume 2**, a 14-part second run of
complete, copy-paste GopherTrunk builds — one working rig per part, each with
its verification status stated plainly.
[Part 2]({{ '/blog/tutorials/cookbook-volume-2-02-paging-rig/' | relative_url }})
parked one dongle on a group of paging channels. This part is two recipes for
one protocol: an NXDN system with a control channel to follow, and plain
NXDN channels on a scan list with nothing to follow at all. Both rest on a
frame layer that was rebuilt against a reporter's captures this autumn — so
this is also the first part where "does it work on air?" has a dated answer.*

> **TL;DR:** Trunked [NXDN]({{ '/reference/nxdn/' | relative_url }}) is a
> `trunking.systems` entry with `protocol: nxdn`, its control channel, and an
> `nxdn_band_plan` — `linear` or `table`, exactly one. A grant names a
> channel **number**, so without a plan the control channel locks and every
> voice grant is dropped. Healthy looks like `nxdn cc locked` then
> `nxdn: site info`. Conventional NXDN is a `scanner.conventional` channel
> with `mode: nxdn` (12.5 kHz NXDN96) or `mode: nxdn48` (6.25 kHz); a call
> gate opens it only on confirmed traffic frames, so a wrong-width channel
> never opens instead of recording silence. Carrier tracking holds about
> ±3 kHz; set `ppm` anyway. Status: trunked voice was on-air verified on
> 21 Oct on a reporter's 10-channel 453 MHz system (#1261); the grant message
> itself has no capture pin.

**Key takeaways**

- **A grant is a channel number, not a frequency.** `nxdn_band_plan` is what
  turns it into one. Leave it out and the system still locks, but nothing
  records — and each dropped grant names the channel number you are missing.
- **Channel width is part of the mode.** `nxdn` and `nxdn48` are the same
  frame at 4800 and 2400 symbols per second. Pick the wrong one and the
  channel stays shut, which is the intended failure.
- **Frequency error matters less than it did.** A carrier tracker ahead of
  the discriminator follows about ±3 kHz of tuner error and drift. `ppm` is
  still worth setting: it is what the first frames depend on.
- **Each path has its own evidence.** Trunked follow was confirmed live;
  conventional voice and the control channel are pinned on real-air
  captures; the scan-list gate has no live report yet.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| Trunked system | locks the control channel, follows grants | `trunking.systems[]`, `protocol: nxdn`, `control_channels` |
| Channel number → frequency | required before any voice is followed | `nxdn_band_plan.linear` or `nxdn_band_plan.table` |
| Lock evidence | site identity decoded off the air | log `nxdn cc locked`, then `nxdn: site info` |
| A grant the plan cannot place | dropped, with the channel number named | log `nxdn: voice grant dropped` |
| Conventional channel | 12.5 kHz or 6.25 kHz voice on the scan list | `mode: nxdn`, `mode: nxdn48` |
| Call gate | opens on confirmed traffic frames only | log `conv: call gate opened` |
| Call summary | frames, tracked carrier offset, cipher | log `composer: nxdn voice chain ended` |
| Upload | talkgroup and radio ID to Rdio Scanner | `broadcast.rdioscanner[]` |

## In this post

- **What you're building** — two rigs, one frame layer.
- **The shopping list** — two dongles for trunked, one for conventional.
- **The config** — the trunked file, then the scan-list file.
- **First run — what healthy looks like** — a real lock, a real call.
- **When it doesn't work** — symptom → cause → fix.
- **Variations** — a linear plan, a scrambler key, continuous channels.

## What you're building

**Recipe A** is a trunked NXDN site — Kenwood sells it as NEXEDGE, Icom as
IDAS; the [NXDN scanner guide]({{ '/nxdn-scanner/' | relative_url }}) sorts
out the names. One dongle camps the
[control channel]({{ '/reference/control-channel/' | relative_url }}) and
decodes its messages: site information, and a voice-call assignment each time
a [talkgroup]({{ '/reference/talkgroup/' | relative_url }}) keys up. The
assignment says "talkgroup 21, radio 8814, channel 4". A band plan turns
"channel 4" into hertz, a second dongle tunes there, and the call is decoded,
recorded and, if you want, uploaded.

**Recipe B** is NXDN without trunking: a business or plant channel where
radios simply key up. It goes on the scan list of
[Volume 1, Part 6]({{ '/blog/tutorials/operator-cookbook-06-analog-fm-tone-out/' | relative_url }})
as a digital mode. Power alone does not open it; a call gate does, and
[Squelch, Part 9]({{ '/blog/deep-dives/squelch-09-call-gates-ysf-nxdn-dstar/' | relative_url }})
explains what it listens for.

If you built the DMR Tier III rig in
[Volume 1, Part 2]({{ '/blog/tutorials/operator-cookbook-02-dmr-tier3/' | relative_url }}),
Recipe A will feel familiar: a control channel that hands out logical
channel numbers, and a plan to resolve them. What is different is the
history. The NXDN frame layer described in
[Protocol Decoders, Part 6]({{ '/blog/deep-dives/protocol-decoders-06-nxdn-dpmr/' | relative_url }})
never matched a real radio; it was rebuilt from captures, and
Rebuilt From the Air tells it in three parts — the
[frame layer]({{ '/blog/solution-postmortem/rebuilt-from-the-air-07-nxdn-frame-layer/' | relative_url }}),
the
[control channel]({{ '/blog/solution-postmortem/rebuilt-from-the-air-08-nxdn-control-channel/' | relative_url }})
and the
[scrambler]({{ '/blog/solution-postmortem/rebuilt-from-the-air-09-nxdn-scrambler/' | relative_url }}).

<figure class="lab-figure">
<svg viewBox="0 0 680 250" width="680" height="250" role="img" aria-label="Two block chains. Recipe A, trunked: a control SDR feeds the NXDN control-channel decoder, whose voice-call assignments pass through the band plan that converts a channel number to a frequency, then to a voice SDR and the AMBE+2 voice chain that records and uploads the call; a grant with no plan entry is dropped. Recipe B, conventional: a voice SDR runs the scan loop with a power squelch, then the NXDN call gate that needs two frame syncs 192 dibits apart with a valid LICH, then the voice chain at 4800 or 2400 symbols per second and the recorder; a wrong-width channel never opens">
  <text x="8" y="16" fill="var(--fg-muted)" font-size="10">A · trunked — protocol: nxdn</text>
  <rect x="8" y="26" width="84" height="38" rx="4" fill="none" stroke="currentColor"/>
  <text x="50" y="42" text-anchor="middle" fill="currentColor" font-size="10">SDR</text>
  <text x="50" y="55" text-anchor="middle" fill="var(--fg-muted)" font-size="9">role: control</text>
  <line x1="92" y1="45" x2="108" y2="45" stroke="currentColor"/>
  <rect x="108" y="26" width="154" height="38" rx="4" fill="none" stroke="currentColor"/>
  <text x="185" y="42" text-anchor="middle" fill="currentColor" font-size="10">NXDN control channel</text>
  <text x="185" y="55" text-anchor="middle" fill="var(--fg-muted)" font-size="9">site info · call assignment</text>
  <line x1="262" y1="45" x2="278" y2="45" stroke="currentColor"/>
  <rect x="278" y="26" width="146" height="38" rx="4" fill="none" stroke="var(--accent)"/>
  <text x="351" y="42" text-anchor="middle" fill="var(--accent)" font-size="10">nxdn_band_plan</text>
  <text x="351" y="55" text-anchor="middle" fill="var(--fg-muted)" font-size="9">channel number → Hz</text>
  <line x1="424" y1="45" x2="440" y2="45" stroke="currentColor"/>
  <rect x="440" y="26" width="84" height="38" rx="4" fill="none" stroke="currentColor"/>
  <text x="482" y="42" text-anchor="middle" fill="currentColor" font-size="10">SDR</text>
  <text x="482" y="55" text-anchor="middle" fill="var(--fg-muted)" font-size="9">role: voice</text>
  <line x1="524" y1="45" x2="540" y2="45" stroke="currentColor"/>
  <rect x="540" y="26" width="132" height="38" rx="4" fill="none" stroke="currentColor"/>
  <text x="606" y="42" text-anchor="middle" fill="currentColor" font-size="10">AMBE+2 → WAV</text>
  <text x="606" y="55" text-anchor="middle" fill="var(--fg-muted)" font-size="9">History · Rdio upload</text>
  <text x="351" y="82" text-anchor="middle" fill="var(--fg-muted)" font-size="9">no entry for the channel number → the grant is dropped</text>
  <line x1="8" y1="112" x2="672" y2="112" stroke="var(--fg-muted)" stroke-dasharray="4 3"/>
  <text x="8" y="136" fill="var(--fg-muted)" font-size="10">B · conventional — mode: nxdn or nxdn48</text>
  <rect x="8" y="146" width="84" height="38" rx="4" fill="none" stroke="currentColor"/>
  <text x="50" y="162" text-anchor="middle" fill="currentColor" font-size="10">SDR</text>
  <text x="50" y="175" text-anchor="middle" fill="var(--fg-muted)" font-size="9">role: voice</text>
  <line x1="92" y1="165" x2="108" y2="165" stroke="currentColor"/>
  <rect x="108" y="146" width="116" height="38" rx="4" fill="none" stroke="currentColor"/>
  <text x="166" y="162" text-anchor="middle" fill="currentColor" font-size="10">scan loop</text>
  <text x="166" y="175" text-anchor="middle" fill="var(--fg-muted)" font-size="9">squelch_dbfs</text>
  <line x1="224" y1="165" x2="240" y2="165" stroke="currentColor"/>
  <rect x="240" y="146" width="184" height="38" rx="4" fill="none" stroke="var(--accent)"/>
  <text x="332" y="162" text-anchor="middle" fill="var(--accent)" font-size="10">NXDN call gate</text>
  <text x="332" y="175" text-anchor="middle" fill="var(--fg-muted)" font-size="9">two syncs 192 dibits apart + LICH</text>
  <line x1="424" y1="165" x2="440" y2="165" stroke="currentColor"/>
  <rect x="440" y="146" width="126" height="38" rx="4" fill="none" stroke="currentColor"/>
  <text x="503" y="162" text-anchor="middle" fill="currentColor" font-size="10">NXDN voice chain</text>
  <text x="503" y="175" text-anchor="middle" fill="var(--fg-muted)" font-size="9">4800 or 2400 sym/s</text>
  <line x1="566" y1="165" x2="582" y2="165" stroke="currentColor"/>
  <rect x="582" y="146" width="90" height="38" rx="4" fill="none" stroke="currentColor"/>
  <text x="627" y="169" text-anchor="middle" fill="currentColor" font-size="10">AMBE+2 → WAV</text>
  <text x="332" y="202" text-anchor="middle" fill="var(--fg-muted)" font-size="9">wrong width, a control channel, an analog carrier → never opens</text>
  <text x="340" y="238" text-anchor="middle" fill="var(--fg-muted)" font-size="10">same frame layer, same vocoder, two ways of learning that a call exists</text>
</svg>
<figcaption>Trunked NXDN learns about a call from the control channel and needs the band plan to find it; conventional NXDN has to recognise the call on the channel itself, which is the call gate's job.</figcaption>
</figure>

## The shopping list

| Item | Price (rough) | Notes |
|---|---|---|
| RTL-SDR Blog V3/V4, two for Recipe A | ~$70 | one camps the control channel, one follows voice |
| RTL-SDR Blog V3/V4, one for Recipe B | ~$35 | the scan list drives a single voice dongle |
| UHF or VHF antenna | $0–$30 | for the band the system is licensed in |
| The system's frequency list | $0 | control channel, plus every voice channel with its channel number |

A dongle with a TCXO earns its price here. NXDN48's inner symbols are only
±350 Hz from centre, so half a kilohertz of tuner error is a different
symbol.

## The config

Recipe A, trunked:

```yaml
log:
  level: info

storage:
  path: "../data/calls.db"
  cc_cache_file: "../data/cc-cache.json"

recordings:
  dir: "../recordings"

sdr:
  sample_rate: 2_400_000
  devices:
    - serial: "00000001"            # from `gophertrunk sdr list`
      role: control
      ppm: 0                        # set it from a calibration; see below
      gain: "auto"
    - serial: "00000002"
      role: voice
      ppm: 0
      gain: "auto"

trunking:
  systems:
    - name: "Plant-NXDN"
      protocol: nxdn
      control_channels:
        - 453_775_000
      # talkgroup_file: "../config/talkgroups-nxdn.csv"   # optional names
      nxdn_band_plan:               # REQUIRED to follow voice
        table:                      # one entry per channel number
          - { channel: 1, freq_hz: 453_775_000 }
          - { channel: 2, freq_hz: 453_675_000 }

broadcast:
  rdioscanner:
    - enabled: true
      name: "local-rdio"
      url: "https://scanner.example.org"
      api_key: "YOUR_RDIOSCANNER_API_KEY"
      system_id: 1
      systems: ["Plant-NXDN"]
```

**`nxdn_band_plan`.** Give exactly one of two shapes; both together are
refused at load. `table` lists each channel number with its downlink
frequency, which suits a site whose channels are scattered. `linear` suits a
regular grid: `freq = base_hz + (channel − offset) × spacing_hz`, with
`offset: 1` when channels are numbered from 1 (the Variations section has
the snippet). Where do the numbers come from? The licence or a frequency
listing gives the frequencies. The site tells you one number itself — the
`nxdn: site info` line carries `control_channel_1`, the channel number of
the control channel you are tuned to. Every other gap announces itself the
first time it is used, in a WARN that names the channel number.

**Two dongles.** The example config documents the one-dongle `voice_taps`
arrangement for P25 and DMR Tier III; it does not list NXDN, so this recipe
keeps the classic split. Voice grants can then land anywhere the voice
dongle tunes.

**`ppm`.** Set it. The receivers track the carrier now, but tracking starts
from wherever the dongle is, and a cheap dongle moves: in one reporter
capture it drifted from −645 Hz to −679 Hz in under nine seconds, just after
a calibration.

**`broadcast.rdioscanner`.** Optional. Each completed call is uploaded with
its talkgroup and source radio ID;
[Volume 1, Part 9]({{ '/blog/tutorials/operator-cookbook-09-sharing-the-feed/' | relative_url }})
covers the feed.

The control-channel pipeline is built for a 12.5 kHz NXDN96 control channel;
nothing in a system entry selects NXDN48.

Recipe B, conventional — a separate file, one dongle:

```yaml
log:
  level: info

storage:
  path: "../data/calls.db"

recordings:
  dir: "../recordings"

sdr:
  sample_rate: 2_400_000
  devices:
    - serial: "00000001"
      role: voice                   # the scan list drives a voice SDR
      ppm: 0
      gain: "auto"

scanner:
  conventional:
    - label: "Plant ops"
      frequency_hz: 447_100_000
      mode: nxdn                    # 12.5 kHz NXDN96
      squelch_dbfs: -50
      hangtime_ms: 1500
      talkgroup_id: 2147500201
    - label: "Yard 6.25"
      frequency_hz: 447_112_500
      mode: nxdn48                  # 6.25 kHz NXDN48
      squelch_dbfs: -50
      talkgroup_id: 2147500202
```

`squelch_dbfs` is still there, and still an absolute in-channel level, but
it only nominates the channel. The gate decides: a frame sync, then another
exactly 192 dibits later, with a valid link-information field marking a
traffic frame. Once the level breaks, the scan window stretches to half a
second for `nxdn` and a full second for `nxdn48` so the gate can hear that
pair. Tone gates and data decoders are analog-only and rejected on these
entries, and there is no
[RAN]({{ '/reference/ran-nxdn/' | relative_url }}) filter key: any NXDN
traffic of the right width on the frequency opens the channel. To run both
recipes on one box, add a second voice dongle — the scanner takes the last
one.

## First run — what healthy looks like

```sh
gophertrunk run -config config.yaml
```

Recipe A first. These lines are from the daemon running the project's
committed real-air control-channel slice (453.775 MHz, one second, looped
through a `baseband.replay` tuner) against the system entry above:

```text
INF daemon: system config name=Plant-NXDN protocol=nxdn control_channels=1 enhancements="soft_decision=off afc=off"
INF cc-hunt: trying system=Plant-NXDN freq_hz=453775000
INF nxdn cc locked freq=453775000 rate=9600 site=1 sys=85
INF cc-hunt: locked system=Plant-NXDN freq_hz=453775000 nac=1
INF nxdn: site info system=Plant-NXDN ran=1 location_id=4197025 category=local system_code=85 site_code=1 control_channel_1=1 control_channel_2=0 adjacent_sites=2
```

The lock arrived 90 ms after the hunt started. `site info` is the line to
copy into your notes: system 85, site 1, RAN 1, two adjacent sites, and
`control_channel_1=1` — the control channel is channel 1, which is the first
row of the table. That capture is idle, so it cannot show a grant. On a live
system each call adds `call started` with a grant reading
`Plant-NXDN/nxdn tg=… src=… freq=…`, then `composer: nxdn voice chain
started`, `recorder: call started` and, with the feed enabled, `broadcast:
call streamed`.

Recipe B, from the committed real-air NXDN96 voice slice replayed through
the `Plant ops` entry:

```text
INF conv: call gate opened gate=NXDN freq_hz=447100000 label="Plant ops"
INF synthetic call started device=replay-01 grant="scanner/nxdn tg=2147500201 src=0 freq=447100000 "
INF recorder: call started device=replay-01 wav=../recordings/scanner/2147500201/20261121_192928_2147500201.wav tg=2147500201 provoice=false vocoder=ambe2-dmr
INF composer: nxdn voice chain started serial=replay-01 system=scanner group=2147500201 symbol_rate=4800
INF composer: nxdn voice chain ended serial=replay-01 voice_frames=1236 frames=617 lich_ok=617 carrier_offset_hz=395 carrier_offset_at_voice_hz=426 iq_samples=59429888 sacch_ok=617 sacch_fail=0 sacch_superframes=154 src=8814 dst=0 cipher=clear
```

The last line is the one worth learning. `lich_ok` equal to `frames` means
every frame was recognised; `carrier_offset_hz=395` is the tuner error the
receiver tracked out, which across calls shows your dongle drifting; `src` is
the radio ID read from the slow signalling; `cipher=clear` says the call was
not scrambled. The NXDN48 slice on the `Yard 6.25` entry printed the same
sequence with `scanner/nxdn48` and `symbol_rate=2400`. And the test that
matters most: the NXDN96 slice fed to an `nxdn48` entry sat at −5.4 dBFS, 45
dB over its squelch, and never opened.

**Status, in one sentence each.** Trunked voice is on-air verified: on
21 Oct a reporter locked their 10-channel 453 MHz system, followed grants and
uploaded calls with talkgroup and radio ID
([#1261](https://github.com/MattCheramie/GopherTrunk/issues/1261)). The
control-channel decode is capture-verified — all 214 frames of that
reporter's capture pass their CRC. The voice-call assignment has no capture
pin, because the committed capture is idle: its layout is reference-pinned
and otherwise rests on that live run. Conventional NXDN96 and NXDN48 voice
is capture-verified on two real-air slices, and the reporter confirmed both
through tuner drift on the same date. The scan-list call gate came later, is
pinned on those slices, and has no live report yet.

## When it doesn't work

| Symptom | Likely cause | Fix |
|---|---|---|
| Locks, but no call ever starts | no band plan | Start-up prints `daemon: nxdn system has no nxdn_band_plan; voice grants will be dropped (no-bandplan)`, and each grant prints `nxdn: voice grant dropped; no band plan configured` with `channel=`. Add the plan |
| Some calls follow, others vanish | a channel number the plan does not cover | `nxdn: voice grant dropped; channel outside band plan` names the number; add it to `table`, or fix the `linear` base and offset |
| Daemon refuses the config at the band plan | both shapes given, or neither | Keep `linear` or `table`, not both |
| Locks and grants, nothing recorded, a WARN about no voice source | no voice dongle | Add a `role: voice` SDR |
| No lock at all | wrong control frequency, weak signal, or not an NXDN96 control channel | Read the diagnosis on `cchunt: hunt failed — no control-channel lock`; check the frequency first |
| Worked for minutes, then failed until the correction was nudged | a build from before carrier tracking | Update, and set `ppm`; the tracker follows drift the old receiver could not |
| A call ends early with the tracked offset near 3000 | the tracker reached its bound | Compare `carrier_offset_at_voice_hz` with `carrier_offset_hz` on the ended line; the first is where it sat while speech decoded. Correct `ppm` |
| A scan-list channel shows a strong level and never opens | wrong width, or the carrier is not NXDN traffic | Swap `mode` between `nxdn` and `nxdn48`. A control channel or an analog carrier never opens, by design |
| The audio is noise though every frame decodes | the radios use the scrambler | The log says `composer: nxdn VCALL — call is encrypted`; see the key variation |
| A filter switch seemed to fix decoding on an RTL-SDR | coincidence | `narrowband_filter` exists only on a HackRF Pro; elsewhere it logs a WARN and does nothing |

That last row has a story. A reporter saw decoding recover after setting
`narrowband_filter: true` on an RTL-SDR, where the key is a no-op. The real
causes were the ones in the rows above it. When a config change "fixes"
something it cannot touch, keep looking.

### How this recipe shapes operator practice

- **Lock first, plan second.** Run Recipe A with a one-row table. The lock
  and `site info` confirm the system; the dropped-grant warnings then hand
  you the channel numbers still to fill in.
- **Read the ended line after every odd call.** Frames, offset and cipher
  separate "weak signal", "drifting dongle" and "scrambled" in one glance.
- **Treat a shut channel as information.** A gate that will not open on a
  strong carrier is telling you the mode is wrong.

## Variations

- **A linear plan.** For a site on a regular grid, replace `table` with:

  ```yaml
      nxdn_band_plan:
        linear:
          base_hz: 453_150_000
          spacing_hz: 12_500
          offset: 1                 # channels numbered from 1
  ```

  The Config Builder edits either shape, in the browser
  (`gophertrunk config serve`) or the terminal (`gophertrunk config`).
- **A scrambler key you hold.** For radios you run or are authorised to
  monitor, the built-in 15-bit scrambler descrambles in-process when you
  list the key — GopherTrunk recovers no keys:

  ```yaml
      encryption_keys:
        - key_id: 3
          algorithm: scrambler
          key: "32767"              # the radio's decimal key, 1..32767
  ```

  It goes on a scan-list channel or a trunked system alike, is
  capture-verified with known keys, and is the subject of
  [Part 8]({{ '/blog/tutorials/cookbook-volume-2-08-your-own-keys/' | relative_url }}).
- **Continuous channels.** A `role: wideband` dongle's `conventional:` list
  accepts `nxdn` and `nxdn48` entries, each on an always-on tap, so nothing
  is missed while the scanner is elsewhere.
- **More systems.** Several NXDN systems share dongles under the pool rules
  of
  [Volume 1, Part 7]({{ '/blog/tutorials/operator-cookbook-07-multi-system-pool/' | relative_url }}).

## Where this goes next

NXDN identifies every transmission digitally. [Part
4]({{ '/blog/tutorials/cookbook-volume-2-04-fleet-signalling/' | relative_url }})
goes back to analog channels and asks the same question — who keyed up? —
with MDC1200 and FleetSync bursts, and gets CTCSS and DCS tone gates right
along the way.

## FAQ

**Why does my NXDN control channel lock but no calls record?**
An NXDN grant names a channel number, not a frequency. Without an
`nxdn_band_plan` on the system the daemon cannot tune the call and drops the
grant, logging the channel number each time. Add a `linear` or `table` plan
and the same grants are followed.

**How do I find the channel numbers for an NXDN band plan?**
Start from the system's frequency list. The `nxdn: site info` log line gives
the control channel's own number, and every grant the plan cannot place logs
its channel number in a warning, so a short session fills in the rest.

**What is the difference between mode nxdn and nxdn48?**
`nxdn` is the 12.5 kHz NXDN96 mode at 4800 symbols per second; `nxdn48` is
the 6.25 kHz NXDN48 mode at 2400. The frame and vocoder are identical. A
channel set to the wrong one never opens, because its call gate never sees a
valid frame pair.

**Is NXDN trunking in GopherTrunk verified on air?**
Yes: on 21 Oct a reporter locked a 10-channel 453 MHz system, followed voice
grants and uploaded calls with talkgroup and radio ID. The control-channel
decode is additionally pinned on a real capture; the grant message has no
capture pin because that capture is idle.

**Can GopherTrunk decode scrambled NXDN?**
Only with the key. For the built-in 15-bit scrambler it descrambles calls
when a matching `encryption_keys` entry is configured, and that path is
verified on captures with known keys. NXDN DES and AES are identified, not
decrypted.

## Series navigation

**Part 3 of 14** · ←
[Part 2: The Paging Rig — POCSAG and FLEX on One Dongle]({{ '/blog/tutorials/cookbook-volume-2-02-paging-rig/' | relative_url }})
· Next →
[Part 4: Fleet Signalling — MDC1200, FleetSync and Tone Gates]({{ '/blog/tutorials/cookbook-volume-2-04-fleet-signalling/' | relative_url }})
