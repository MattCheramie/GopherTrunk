---
title: "The Operator's Cookbook, Part 1: The $40 P25 Starter Rig"
description: A complete first GopherTrunk build — one RTL-SDR dongle, the stock whip, and a laptop decoding a local P25 Phase 1 system, with a copy-paste config.yaml, the exact log lines a healthy first run prints, and a troubleshooting table for when it won't lock.
category: tutorials
keywords: rtl-sdr p25 scanner setup, gophertrunk config example, p25 phase 1 decoder software, cheap police scanner sdr, rtl-sdr trunking scanner, p25 control channel lock, sdr scanner under 50 dollars, first sdr scanner build, gophertrunk cookbook
tags: [operator-cookbook, p25, rtl-sdr, config, getting-started, hardware]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "The Operator's Cookbook"
series_part: 1
---

*Part 1 of **The Operator's Cookbook**, a 14-part series of complete,
copy-paste GopherTrunk builds — one working rig per part, antenna to browser.
Every other tutorial series on this blog is organized by subsystem — RF, DSP,
recording, operations. This one is organized by **what you're trying to
build**: each part is one full recipe, hardware list to working config to the
log lines that prove it's alive. The running thread is a rig that grows — the
$40 starter you build today is the same box that streams to Broadcastify in
Part 9 and runs headless in a closet by Part 11. We start with the most common
first build there is: one cheap dongle on a local P25 Phase 1 system.*

> **TL;DR:** One **RTL-SDR Blog V3/V4 (~$35)**, the whip it ships with, and a
> laptop decode a P25 Phase 1 system end to end. The trick that makes a single
> dongle work is `role: wideband` + `voice_taps: 2` — the control channel and
> the voice grants all decode out of one 2.4 MS/s capture, no second radio.
> You find your system on RadioReference (or with
> [`gophertrunk hunt`]({{ '/blog/deep-dives/the-hunt-07-locking-a-p25-system/' | relative_url }})),
> paste its control channels into `config.yaml`, run
> `gophertrunk run -config config.yaml`, and watch for
> `control channel locked` then `recorder: call started` in the log. The web
> console at `http://127.0.0.1:8080` does the rest.

**Key takeaways**

- **$40 is a real number, not a teaser.** A ~$35 RTL-SDR with a TCXO plus the
  antenna in the box decodes P25 Phase 1 cleanly on any system you have
  reasonable signal from. The upgrades in later parts buy margin, not
  possibility.
- **One dongle carries control and voice.** `role: wideband` with
  `voice_taps: 2` taps voice calls out of the same IQ capture the control
  channel decodes from — as long as the system's voice channels fit inside the
  2.4 MHz window.
- **The config is four blocks.** SDR device, one system, its control channels,
  and where recordings go. Everything else in
  `config.example.yaml` is optional and defaulted.
- **Healthy has a signature.** `control channel locked` → `call started` →
  `recorder: call ended` is the heartbeat of a working rig; this post shows
  the real lines so you know them on sight.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| Hardware list | dongle + whip + adapter, well under $100 total | [what do I need?]({{ '/what-do-i-need-for-gophertrunk/' | relative_url }}) |
| Which dongle | V3 vs V4 — either decodes identically | [V3 vs V4 guide]({{ '/rtl-sdr-blog-v3-vs-v4/' | relative_url }}), [RTL-SDR]({{ '/reference/rtl-sdr/' | relative_url }}) |
| One-dongle trunking | wideband capture + per-call voice taps | `sdr.devices[].role: wideband`, `voice_taps` |
| System definition | protocol, control channels, talkgroup names | `trunking.systems[]`, `control_channels`, `talkgroup_file` |
| Finding your system | RadioReference, or map it yourself | [`gophertrunk hunt`]({{ '/blog/deep-dives/the-hunt-06-control-channel-hunting/' | relative_url }}) |
| Frequency error | per-device `ppm`, or let `autotune` suggest one | [ppm correction]({{ '/reference/ppm-frequency-correction/' | relative_url }}) |
| First-run cockpit | web console on the API port | `api.http_addr`, [web guide]({{ '/web.html' | relative_url }}) |

## In this post

- **What you're building** — one dongle, one P25 system, calls on disk and in the browser.
- **The shopping list** — three items, round numbers.
- **Finding your system** — RadioReference first, `hunt` when it isn't listed.
- **The config** — a complete, minimal `config.yaml` with every key verified.
- **First run — what healthy looks like** — the exact log lines and web panels.
- **When it doesn't work** — symptom → cause → fix.

## What you're building

The finished rig is a laptop with a USB dongle hanging off it, doing what a
$500 digital scanner does: camp a P25 Phase 1
[control channel]({{ '/reference/control-channel/' | relative_url }}), decode
every voice grant, follow calls to their voice channels, decode the IMBE audio,
and write one WAV per call into a folder tree sorted by
[talkgroup]({{ '/reference/talkgroup/' | relative_url }}) — with a live web
console showing the system light up in real time.

The architectural trick that makes one dongle enough: GopherTrunk's wideband
engine treats the dongle's whole 2.4 MHz capture as a band, not a channel. One
tap inside that capture decodes the control channel continuously; when a voice
grant arrives, a `voice_taps` slot spins up a second down-converter *on the
same IQ stream* at the granted frequency. No retuning, no second radio —
provided the voice channels land inside the window. (When your system spreads
wider than 2.4 MHz, that's the two-dongle variation at the end.)

<figure class="lab-figure">
<svg viewBox="0 0 680 240" width="680" height="240" role="img" aria-label="Signal chain of the one-dongle P25 starter rig: a whip antenna feeds an RTL-SDR dongle producing a 2.4 megasample wideband IQ stream; inside GopherTrunk one DDC tap feeds the P25 control-channel decoder which issues grants, and two voice taps on the same stream feed the IMBE vocoder, producing WAV recordings on disk and live panels in the web console">
  <rect x="10" y="92" width="70" height="36" rx="4" fill="none" stroke="currentColor"/>
  <text x="45" y="114" text-anchor="middle" fill="currentColor" font-size="10">whip</text>
  <line x1="80" y1="110" x2="112" y2="110" stroke="currentColor"/>
  <rect x="112" y="92" width="86" height="36" rx="4" fill="none" stroke="currentColor"/>
  <text x="155" y="110" text-anchor="middle" fill="currentColor" font-size="10">RTL-SDR</text>
  <text x="155" y="122" text-anchor="middle" fill="var(--fg-muted)" font-size="9">2.4 MS/s IQ</text>
  <line x1="198" y1="110" x2="240" y2="110" stroke="currentColor"/>
  <rect x="240" y="20" width="200" height="200" rx="6" fill="none" stroke="var(--fg-muted)" stroke-dasharray="4 3"/>
  <text x="340" y="36" text-anchor="middle" fill="var(--fg-muted)" font-size="10">GopherTrunk (role: wideband)</text>
  <rect x="256" y="52" width="168" height="34" rx="4" fill="none" stroke="var(--accent)"/>
  <text x="340" y="66" text-anchor="middle" fill="var(--accent)" font-size="10">CC tap → P25 decoder</text>
  <text x="340" y="79" text-anchor="middle" fill="var(--fg-muted)" font-size="9">TSBK grants</text>
  <rect x="256" y="104" width="168" height="34" rx="4" fill="none" stroke="currentColor"/>
  <text x="340" y="118" text-anchor="middle" fill="currentColor" font-size="10">voice tap 1 → IMBE</text>
  <rect x="256" y="148" width="168" height="34" rx="4" fill="none" stroke="currentColor"/>
  <text x="340" y="162" text-anchor="middle" fill="currentColor" font-size="10">voice tap 2 → IMBE</text>
  <line x1="340" y1="86" x2="340" y2="104" stroke="var(--accent)"/>
  <polygon points="336,98 340,106 344,98" fill="var(--accent)"/>
  <line x1="440" y1="121" x2="490" y2="94" stroke="currentColor"/>
  <line x1="440" y1="165" x2="490" y2="150" stroke="currentColor"/>
  <rect x="490" y="76" width="176" height="34" rx="4" fill="none" stroke="currentColor"/>
  <text x="578" y="90" text-anchor="middle" fill="currentColor" font-size="10">recordings/&lt;system&gt;/&lt;tg&gt;/*.wav</text>
  <rect x="490" y="132" width="176" height="34" rx="4" fill="none" stroke="var(--accent)"/>
  <text x="578" y="146" text-anchor="middle" fill="var(--accent)" font-size="10">web console :8080</text>
  <text x="578" y="159" text-anchor="middle" fill="var(--fg-muted)" font-size="9">Active · History · CC panel</text>
  <text x="340" y="234" text-anchor="middle" fill="var(--fg-muted)" font-size="10">one IQ capture, decoded twice — control continuously, voice per grant</text>
</svg>
<figcaption>The whole $40 rig: one wideband capture carries the control channel and up to two concurrent voice calls, so a single dongle does what used to take two.</figcaption>
</figure>

## The shopping list

| Item | Price (rough) | Notes |
|---|---|---|
| RTL-SDR Blog V3 or V4 | ~$35 | TCXO matters — `ppm: 0` actually holds. [V3 vs V4]({{ '/rtl-sdr-blog-v3-vs-v4/' | relative_url }}) |
| Antenna | $0 | the kit's telescopic [whip]({{ '/reference/whip-antenna/' | relative_url }}) — extend ~one-quarter wavelength for your band |
| Computer | $0 | any laptop/desktop from the last decade; a Pi works too (Part 11) |

That's it — about $40 shipped. A no-name dongle without a
[TCXO]({{ '/reference/tcxo/' | relative_url }}) still works; you'll just meet
the `ppm` troubleshooting row sooner. The
[full hardware checklist]({{ '/what-do-i-need-for-gophertrunk/' | relative_url }})
covers adapters and upgrade paths; resist all of them until the rig works.

## Finding your system

You need two facts: your local system's **protocol** (this recipe wants P25
Phase 1) and its **control channel frequencies**. RadioReference's database
has both for nearly every public-safety system in North America — copy every
frequency marked as a control channel (they rotate; list them all).

No listing, or abroad? GopherTrunk maps unknown systems itself:

```sh
gophertrunk hunt -serial 00000001 -band 851:869
```

sweeps the band, identifies control channels, decodes their identity, and
exports a ready-to-merge config (`-commit` writes it into `config.yaml` for
you). The whole discovery pipeline has its own series —
[The Hunt]({{ '/blog/deep-dives/the-hunt-06-control-channel-hunting/' | relative_url }})
— and [Part 7]({{ '/blog/deep-dives/the-hunt-07-locking-a-p25-system/' | relative_url }})
walks a P25 lock specifically. New to trunking as a concept? The
[scanning module]({{ '/learn/scanning/' | relative_url }}) is the ten-minute
primer on why one control channel governs a whole system.

## The config

Plug in the dongle and confirm GopherTrunk sees it:

```sh
gophertrunk sdr list
```

Note the serial. Then this is the entire `config.yaml` — every key verified
against `config.example.yaml`:

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
    - serial: "00000001"          # from `gophertrunk sdr list`
      role: wideband
      gain: "auto"
      center_freq_hz: 858_000_000 # middle of YOUR system's channels
      voice_taps: 2
      channels:
        - frequency_hz: 857_262_500   # your control channel
          system: "Metro-P25"

trunking:
  systems:
    - name: "Metro-P25"
      protocol: p25
      control_channels:
        - 857_262_500
        - 858_487_500               # list the alternates too
      talkgroup_file: "../config/talkgroups-p25.csv"   # optional
```

Three decisions worth explaining. **`center_freq_hz`** should sit near the
middle of your system's full frequency list (control *and* voice channels), so
as many as possible fall inside the ±1.2 MHz window — a channel must be within
`center ± sample_rate/2` with a 5% edge guard, and voice grants outside it
can't get a tap. **`gain: "auto"`** is deliberate: gain is in *tenths* of a dB
in this file (`"496"` = 49.6 dB), the single most common config typo, and AGC
sidesteps it entirely until [Part 6 of The Analog
Edge]({{ '/blog/tutorials/analog-edge-03-gain-staging/' | relative_url }})
teaches you to stage it by hand. **`talkgroup_file`** is optional — a
RadioReference-style CSV with a `Decimal` column plus optional `Alpha Tag`,
`Description`, `Tag`, `Priority` columns. Skip it for now; calls record by
number, and Part 13 is entirely about naming things.

## First run — what healthy looks like

```sh
gophertrunk run -config config.yaml
```

Within a few seconds, three log lines tell the whole story. First the API
comes up:

```
INF api: listening addr=127.0.0.1:8080 tls=false
```

Then — this is the moment — the control channel decoder finds P25 frame sync
and reads the network ID:

```
INF control channel locked nac=659 freq=857262500 rot=0 delta=0.02
```

`nac` is the system's Network Access Code; check it against RadioReference to
confirm you're on the system you think you are. From here the rig is a
spectator to every grant. When someone keys up:

```
INF call started device=cc:wideband:... grant=... priority=5
INF recorder: call started device=... wav=../recordings/Metro-P25/9001/... tg=9001 provoice=false vocoder=imbe
INF recorder: call ended device=... wav=... duration=4.86s reason=released
```

Open `http://127.0.0.1:8080` and you get the same story visually: the
**Dashboard** shows the locked system, **Active** lights up per call with live
audio if you want it, and **History** is your searchable call log. The [web
guide]({{ '/web.html' | relative_url }}) tours every panel; the
[TUI]({{ '/tui.html' | relative_url }}) gives you the same cockpit in a
terminal for the Part 11 headless build.

Let it run ten minutes. A quiet system is normal; no
`control channel locked` line is not — which brings us to:

## When it doesn't work

| Symptom | Likely cause | Fix |
|---|---|---|
| `cchunt: hunt failed — no control-channel lock` with a `diagnosis` field | wrong/stale CC frequencies, or no signal | Read the `diagnosis` — it distinguishes dead IQ from undecodable IQ. Re-check RadioReference, list *all* CC alternates, or run `gophertrunk hunt -candidates` on them |
| Locks briefly, drops, hunts again | marginal signal or gain mis-staged | Move the whip to a window; see [gain staging]({{ '/blog/tutorials/analog-edge-03-gain-staging/' | relative_url }}) before buying anything |
| `ccdecoder: control carrier offset far from configured frequency` (issue #815) | crystal ppm error, or you've locked an adjacent site's stronger carrier | Set the device `ppm`, or turn on `sdr.autotune: true` and paste the value it suggests; verify the reported site identity |
| Locked, but `TSBK blocks are failing at a high rate while tuned at zero-IF` (issue #402) | the front end's DC spur sits on your control channel | Set `dc_avoid: true` on the device — GT tunes the LO off-channel and mixes back, like SDRTrunk/OP25 |
| CC locked, grants logged, but `no voice device available for grant` | granted voice channel is outside the 2.4 MHz window | Re-pick `center_freq_hz`, or add a second dongle as `role: voice` (variation below) |
| Dongle vanishes mid-run, then reappears | USB power/hub flakiness | The watchdog (`sdr.watchdog_interval_ms`) re-acquires by serial automatically; use a rear-panel port, no hub — see the [USB watchdog deep dive]({{ '/blog/deep-dives/rf-front-end-12-sdr-pool-usb-watchdog/' | relative_url }}) |
| Everything decodes but audio sounds thin/quiet | nothing is wrong — faithful decode is conservative | `recordings.enhance.enabled: true` for the louder OP25-style chain |

One principle from the deep-dive side of the house applies on day one: when
decode is bad, the samples are usually bad first. **The decoder can only be as
good as the samples** — a rig that won't lock wants a better window sill before
it wants a config change.

### How this recipe shapes operator practice

- **Trust log lines over vibes.** Each stage of this rig announces itself with
  a specific string. Learn the healthy trio now; every later part's
  troubleshooting table is written against lines like them.
- **Change one knob at a time.** The config above has exactly three numbers
  you chose (serial, center, control channels). When it misbehaves, that's
  your entire search space — keep it that way as the rig grows.
- **Keep `config.example.yaml` open.** Every key in this series exists there
  with a comment; it's the authoritative reference between cookbook parts.

## Variations

- **Two dongles, classic split.** Delete the wideband block; give one device
  `role: control` and a second `role: voice` (that's the `devices:` shape at
  the top of `config.example.yaml`). No window limit — voice grants can land
  anywhere the voice dongle can tune. Cost: one more ~$35 dongle.
- **Monitor-only.** `role: control` alone, no voice device: you get the full
  control-channel picture (grants, talkgroups, the CC panel) with zero voice.
  Great for reconnaissance on a new system.
- **More concurrent calls.** Raise `voice_taps` — CPU scales roughly linearly
  per tap, and the daemon warns above 16. Two is right for a starter system.
- **Better antenna.** The single highest-value upgrade, and Part 7 of
  [The Analog Edge]({{ '/blog/tutorials/analog-edge-07-antennas/' | relative_url }})
  plus the [antenna guide]({{ '/best-scanner-antenna/' | relative_url }}) cover
  it — but make the whip work first so you have a baseline.

## Where this goes next

This rig assumed the friendliest protocol in the fleet. [Part
2]({{ '/blog/tutorials/operator-cookbook-02-dmr-tier3/' | relative_url }})
points the same hardware at a DMR Tier III network — where the control channel
hands out *logical channel numbers* instead of frequencies, and the config
needs a band plan (or GopherTrunk's ability to learn one off the air) before a
single call records.

## FAQ

**Can I really decode P25 with a $35 RTL-SDR?**
Yes — P25 Phase 1 at 4800 baud is well within an RTL-SDR's dynamic range and
stability, and GopherTrunk's whole P25 path was built and tested against
exactly this hardware. What the cheap dongle costs you is margin on *weak*
systems, which is what the antenna and RF parts of this series buy back.

**Do I need one dongle or two for a trunked system?**
One, if your system's control and voice channels fit inside the dongle's
2.4 MHz window — the `role: wideband` + `voice_taps` config above decodes
both from a single capture. Two, if the system spans more spectrum than that:
keep the wideband CC tap and add a `role: voice` dongle for out-of-window
grants.

**How do I find my local P25 control channel frequency?**
RadioReference lists control channels for most North American systems — use
every frequency flagged as a primary or alternate CC. Without a listing, run
`gophertrunk hunt` with a `-band` sweep; it finds control channels by decoding
them, not by guessing from the license database.

**Why is GopherTrunk's gain setting in tenths of a dB?**
It matches the raw units the tuner driver speaks, so `"496"` means 49.6 dB.
If you're translating a gain figure from SDRTrunk, OP25 or gqrx, multiply by
ten — or start with `gain: "auto"` like this recipe and skip the issue.

**Does this rig decode encrypted P25 calls?**
No — encrypted calls are flagged and logged with their metadata (talkgroup,
source, algorithm ID), but GopherTrunk does not decrypt P25. The per-system
`encrypted_calls` policy controls whether encrypted grants tie up your voice
taps at all; on a starter rig the default is fine.

## Series navigation

**Part 1 of 14** · Next →
[Part 2: A DMR Tier III Network, End to End]({{ '/blog/tutorials/operator-cookbook-02-dmr-tier3/' | relative_url }})
