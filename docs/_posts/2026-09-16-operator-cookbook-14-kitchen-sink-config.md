---
title: "The Operator's Cookbook, Part 14: The Kitchen-Sink Config, Annotated"
description: "The series finale: one full GopherTrunk config.yaml walked section by section — sdr to systems to recordings to broadcast to api — each block annotated with the cookbook part that owns it, a decision table from goal to recipe, and the config hygiene that keeps a growing rig maintainable."
category: tutorials
keywords: gophertrunk config.yaml reference, sdr scanner configuration guide, trunking scanner config example, config.example.yaml annotated, sdr config best practices, scanner software setup guide, gophertrunk full config, config hygiene sdr daemon, gophertrunk cookbook
tags: [operator-cookbook, config, reference, finale, operations]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "The Operator's Cookbook"
series_part: 14
---

*Part 14 — the last — of **The Operator's Cookbook**, a 14-part series of
complete, copy-paste GopherTrunk builds — one working rig per part, antenna to
browser. Thirteen recipes ago the rig was one $40 dongle and four config
blocks; along the way it learned trunked DMR and TETRA, analog and tone-out,
grew remote radios and a second antenna, started streaming, archiving,
running headless, and calling everything by name. This closing part is the
map: one kitchen-sink `config.yaml` walked top to bottom, every block stamped
with the part that owns it, plus the decision table that turns "what do I
want?" into "which part do I read?".*

> **TL;DR:** A full GopherTrunk config is about ten top-level sections, and
> the cookbook covered each where it mattered: `sdr` (Parts 1–2, 7–8, 12),
> `trunking.systems` (Parts 1–5), `scanner`/`tone_out` (Part 6),
> `recordings`/`retention`/`baseband` (Part 10), `broadcast` (Part 9),
> `api`/`web` (Parts 1, 11), `storage` + alias files (Part 13). Most keys
> should stay at their defaults — the authoritative, commented reference is
> `config.example.yaml`, with `internal/config/config.go` behind it. The
> golden rules: paths resolve **relative to the folder containing
> config.yaml**, `gain` is in **tenths of a dB**, secrets go in `token_file`
> and environment variables, and every change is one knob at a time with the
> log watched.

**Key takeaways**

- **The config is organized by subsystem; your goals aren't.** That's what
  this series was for — the decision table below maps intent to recipe, and
  the annotated config maps each block back to its part.
- **Defaults are load-bearing.** Per-protocol FEC is on without any YAML;
  timeouts, hangtime and the DSP knobs ship at values earned on real
  captures. The `*_mode: "off"` opt-outs exist for pre-stripped test
  fixtures, not for tuning.
- **`config.example.yaml` is the source of truth.** Every key in this series
  exists there with a comment; if a blog post, forum tip or old gist
  disagrees with it, the example file wins.
- **A config is maintainable when every line is explainable.** The starter
  rig had three numbers you chose. Keep that property as it grows: know which
  part of this series justifies each block you've added.

## Cheat sheet

| Config section | What it owns | Cookbook part |
|---|---|---|
| `sdr` | dongles, roles, wideband channels, remote radios, diversity | [1]({{ '/blog/tutorials/operator-cookbook-01-forty-dollar-p25-rig/' | relative_url }}), [7]({{ '/blog/tutorials/operator-cookbook-07-multi-system-pool/' | relative_url }}), [8]({{ '/blog/tutorials/operator-cookbook-08-remote-radios/' | relative_url }}), [12]({{ '/blog/tutorials/operator-cookbook-12-diversity-mrc/' | relative_url }}) |
| `trunking.systems` | protocols, control channels, band plans, per-system policy | [1]({{ '/blog/tutorials/operator-cookbook-01-forty-dollar-p25-rig/' | relative_url }})–[5]({{ '/blog/tutorials/operator-cookbook-05-tetra-dmo/' | relative_url }}) |
| `scanner`, `tone_out` | conventional channels, scan modes, paging tones | [6]({{ '/blog/tutorials/operator-cookbook-06-analog-fm-tone-out/' | relative_url }}), [7]({{ '/blog/tutorials/operator-cookbook-07-multi-system-pool/' | relative_url }}) |
| `recordings`, `retention`, `baseband` | formats, FLAC, sweeper, IQ captures | [10]({{ '/blog/tutorials/operator-cookbook-10-archival-rig/' | relative_url }}) |
| `broadcast` | Broadcastify, Rdio Scanner, OpenMHz, Icecast, webhooks | [9]({{ '/blog/tutorials/operator-cookbook-09-sharing-the-feed/' | relative_url }}) |
| `api`, `web`, `audio` | HTTP/auth posture, tabs, live audio | [1]({{ '/blog/tutorials/operator-cookbook-01-forty-dollar-p25-rig/' | relative_url }}), [11]({{ '/blog/tutorials/operator-cookbook-11-closet-appliance/' | relative_url }}) |
| `storage` + alias files | call log, labels, talkgroup/RID names | [10]({{ '/blog/tutorials/operator-cookbook-10-archival-rig/' | relative_url }}), [13]({{ '/blog/tutorials/operator-cookbook-13-naming-everything/' | relative_url }}) |

## In this post

- **The map** — the kitchen-sink config, block by block, part by part.
- **The decision table** — what do you want? → which part.
- **Knobs that matter, defaults to leave alone** — where tuning helps and where it hurts.
- **Config hygiene** — paths, units, secrets, and the web's write mode.
- **Where to go from here** — the series wrap, and what to read next.

## The map

Here is a rig that has absorbed all thirteen recipes, abridged to the keys
that carry weight. Every key exists in `config.example.yaml`; every `← Part`
note is a link target in this series:

```yaml
log:
  level: info                       # debug adds per-call audio-quality lines

storage:
  path: "../data/calls.db"          # call log + labels  ← Parts 10, 13

recordings:
  dir: "../recordings"
  format: flac                      # lossless, ~half of WAV  ← Part 10
  mbe_files: false                  # DSD-FME sidecars       ← Part 10
  enhance:
    enabled: false                  # faithful by default; OP25-ish when on

retention:
  call_log_days: 30                 # the sweeper            ← Part 10
  files_days: 14
  interval: "1h"

sdr:
  sample_rate: 2_400_000
  autotune: false                   # ppm suggester           ← Part 1
  devices:
    - serial: "00000001"            # the original $40 stick  ← Part 1
      role: wideband
      gain: "auto"                  # TENTHS of a dB if fixed
      center_freq_hz: 858_000_000
      voice_taps: 2
      channels:
        - frequency_hz: 857_262_500
          system: "Metro-P25"
    - serial: "00000002"            # spill-over voice        ← Part 7
      role: voice
      gain: "auto"
  rtl_tcp: []                       # radios far away         ← Part 8
  soapy_remote: []                  # USRP-class + diversity  ← Parts 8, 12

trunking:
  voice_hangtime_ms: 3500           # leave alone (see below)
  systems:
    - name: "Metro-P25"
      protocol: p25                 #                         ← Part 1
      control_channels: [857_262_500, 858_487_500]
      talkgroup_file: "../config/talkgroups-p25.csv"   # ← Part 13
      rid_alias_file: "../config/rids-p25.csv"         # ← Part 13
    - name: "Regional-DMR"
      protocol: dmr                 # Tier III + band plan    ← Part 2
      control_channels: [851_037_500]
      # dmr_band_plan omitted → learned off the air

scanner:
  scan_mode: all                    # all | list              ← Part 7
  conventional: []                  # analog FM channels      ← Part 6

tone_out:
  profiles: []                      # two-tone fire paging    ← Part 6

broadcast:                          # outbound feeds          ← Part 9
  min_duration_ms: 0
  # broadcastify: / rdioscanner: / openmhz: / icecast: / webhook:

baseband:                           # IQ capture & replay     ← Part 10
  # record: / replay: / auto_record:

audio:
  enabled: false                    # live speakers, off headless

api:
  http_addr: "127.0.0.1:8080"       #                         ← Part 1
  auth:
    mode: "auto"                    # LAN posture             ← Part 11
    # token_file: "../config/api-token"

metrics:
  enabled: true                     # /metrics for the watchdogs ← Part 11
```

Two structural observations that only show up at this altitude. First, the
config has a **radio half and an output half** — `sdr` + `trunking` decide
what gets decoded; `recordings`/`broadcast`/`api` decide where it goes — and
they scale independently, which is why Part 7's many-systems build and Part
9's streaming build never stepped on each other. Second, **systems are the
join point**: a `channels:` entry on a dongle, a `system:` filter on a feed,
and an alias file all reference `trunking.systems[].name` — keep those names
stable, because half the config points at them.

<figure class="lab-figure">
<svg viewBox="0 0 680 235" width="680" height="235" role="img" aria-label="The kitchen-sink config drawn as a map: a radio half containing the sdr block labelled parts 1, 7, 8 and 12 and the trunking systems block labelled parts 1 through 5, joined by system names to an output half containing recordings and retention labelled part 10, broadcast labelled part 9, api and web labelled parts 1 and 11, and alias files labelled part 13; scanner and tone_out sit with the radio half labelled part 6.">
  <rect x="14" y="30" width="310" height="190" rx="8" fill="none" stroke="var(--fg-muted)" stroke-dasharray="5 4"/>
  <text x="169" y="48" text-anchor="middle" fill="var(--fg-muted)" font-size="10">radio half — what gets decoded</text>
  <rect x="30" y="60" width="130" height="52" rx="5" fill="none" stroke="currentColor"/>
  <text x="95" y="80" text-anchor="middle" fill="currentColor" font-size="10">sdr</text>
  <text x="95" y="96" text-anchor="middle" fill="var(--accent)" font-size="9">Parts 1 · 7 · 8 · 12</text>
  <rect x="178" y="60" width="130" height="52" rx="5" fill="none" stroke="currentColor"/>
  <text x="243" y="80" text-anchor="middle" fill="currentColor" font-size="10">trunking.systems</text>
  <text x="243" y="96" text-anchor="middle" fill="var(--accent)" font-size="9">Parts 1–5</text>
  <rect x="30" y="140" width="130" height="52" rx="5" fill="none" stroke="currentColor"/>
  <text x="95" y="160" text-anchor="middle" fill="currentColor" font-size="10">scanner · tone_out</text>
  <text x="95" y="176" text-anchor="middle" fill="var(--accent)" font-size="9">Part 6 (· 7)</text>
  <rect x="178" y="140" width="130" height="52" rx="5" fill="none" stroke="currentColor"/>
  <text x="243" y="160" text-anchor="middle" fill="currentColor" font-size="10">alias files</text>
  <text x="243" y="176" text-anchor="middle" fill="var(--accent)" font-size="9">Part 13</text>
  <line x1="324" y1="118" x2="368" y2="118" stroke="var(--accent)" stroke-width="2"/>
  <polygon points="362,113 372,118 362,123" fill="var(--accent)"/>
  <text x="347" y="106" text-anchor="middle" fill="var(--accent)" font-size="9">system names</text>
  <rect x="372" y="30" width="294" height="190" rx="8" fill="none" stroke="var(--fg-muted)" stroke-dasharray="5 4"/>
  <text x="519" y="48" text-anchor="middle" fill="var(--fg-muted)" font-size="10">output half — where it goes</text>
  <rect x="388" y="60" width="128" height="52" rx="5" fill="none" stroke="currentColor"/>
  <text x="452" y="80" text-anchor="middle" fill="currentColor" font-size="10">recordings · retention</text>
  <text x="452" y="96" text-anchor="middle" fill="var(--accent)" font-size="9">Part 10</text>
  <rect x="530" y="60" width="120" height="52" rx="5" fill="none" stroke="currentColor"/>
  <text x="590" y="80" text-anchor="middle" fill="currentColor" font-size="10">broadcast</text>
  <text x="590" y="96" text-anchor="middle" fill="var(--accent)" font-size="9">Part 9</text>
  <rect x="388" y="140" width="128" height="52" rx="5" fill="none" stroke="currentColor"/>
  <text x="452" y="160" text-anchor="middle" fill="currentColor" font-size="10">api · web · audio</text>
  <text x="452" y="176" text-anchor="middle" fill="var(--accent)" font-size="9">Parts 1 · 11</text>
  <rect x="530" y="140" width="120" height="52" rx="5" fill="none" stroke="currentColor"/>
  <text x="590" y="160" text-anchor="middle" fill="currentColor" font-size="10">storage</text>
  <text x="590" y="176" text-anchor="middle" fill="var(--accent)" font-size="9">Parts 10 · 13</text>
</svg>
<figcaption>The config at map scale: a radio half and an output half joined by system names — and every block stamped with the cookbook part that explains it.</figcaption>
</figure>

## The decision table

The series was organized by goal; here is the whole thing as a lookup:

| What do you want? | Read | The load-bearing keys |
|---|---|---|
| Hear a local P25 system for ~$40 | [Part 1]({{ '/blog/tutorials/operator-cookbook-01-forty-dollar-p25-rig/' | relative_url }}) | `role: wideband`, `voice_taps`, `control_channels` |
| Trunked DMR (Tier III) | [Part 2]({{ '/blog/tutorials/operator-cookbook-02-dmr-tier3/' | relative_url }}) | `protocol: dmr`, `dmr_band_plan` (or let it learn) |
| Two conversations off one DMR repeater | [Part 3]({{ '/blog/tutorials/operator-cookbook-03-conventional-dmr-two-slots/' | relative_url }}) | `protocol: dmr-tier2`, `dmr_interleaved_voice` |
| TETRA, infrastructure or direct mode | [Part 4]({{ '/blog/tutorials/operator-cookbook-04-tetra-tmo/' | relative_url }}) / [Part 5]({{ '/blog/tutorials/operator-cookbook-05-tetra-dmo/' | relative_url }}) | `protocol: tetra` / `tetra-dmo`, `tetra_mcc`/`tetra_mnc` |
| Analog FM, marine, fire tone-out | [Part 6]({{ '/blog/tutorials/operator-cookbook-06-analog-fm-tone-out/' | relative_url }}) | `scanner.conventional`, `tone_out.profiles` |
| Several systems on one box | [Part 7]({{ '/blog/tutorials/operator-cookbook-07-multi-system-pool/' | relative_url }}) | device roles, `scan_mode`, priorities |
| Antenna far from the decoder | [Part 8]({{ '/blog/tutorials/operator-cookbook-08-remote-radios/' | relative_url }}) | `rtl_tcp`, `soapy_remote`, `ka9q_radio` |
| Feed Broadcastify / Rdio / OpenMHz | [Part 9]({{ '/blog/tutorials/operator-cookbook-09-sharing-the-feed/' | relative_url }}) | `broadcast.*` backends |
| Keep everything, forever, small | [Part 10]({{ '/blog/tutorials/operator-cookbook-10-archival-rig/' | relative_url }}) | `recordings.format: flac`, `retention` |
| Run 24/7 in a closet | [Part 11]({{ '/blog/tutorials/operator-cookbook-11-closet-appliance/' | relative_url }}) | `api.auth`, `metrics`, systemd/Docker |
| A second antenna against fading | [Part 12]({{ '/blog/tutorials/operator-cookbook-12-diversity-mrc/' | relative_url }}) | `diversity: mrc`, `antenna:`, `diversity_capture` |
| Names instead of numbers | [Part 13]({{ '/blog/tutorials/operator-cookbook-13-naming-everything/' | relative_url }}) | `talkgroup_file`, `rid_alias_file`, labels |

## Knobs that matter, defaults to leave alone

**Worth setting deliberately:** the handful of numbers each recipe made you
choose. Device serials, `center_freq_hz` and `sample_rate` (your spectrum),
`gain` (staged, in tenths, per
[The Analog Edge]({{ '/blog/tutorials/analog-edge-03-gain-staging/' | relative_url }})),
`control_channels` (all the alternates), `voice_taps` (your concurrency),
the storage/recordings paths and retention windows (your disk), and
`api.auth.mode` the moment the daemon binds off loopback.

**Leave alone until a symptom names them:**

- **The FEC opt-outs** (`motorola_bch_mode`, `tetra_channel_coding`,
  `p25_phase2_trellis_mode`, …). Per-protocol FEC is **on by default with no
  YAML at all**; these keys exist for pre-stripped capture fixtures. Turning
  one "off" on a live system just breaks decode.
- **`call_timeout_ms` / `voice_hangtime_ms`** — 30 s and 3.5 s were chosen
  against real systems; shorten them only for a documented reason
  ([call assembly]({{ '/blog/deep-dives/recording-streaming-03-assembling-a-call/' | relative_url }})).
- **`recordings.equalizer` / `enhance` internals** — the corners and targets
  mirror OP25-class defaults; toggle `enhance.enabled`, don't sculpt filters
  before listening.
- **`watchdog_interval_ms`, `tuner_strategy: auto`, `cc_hunt` backoffs** —
  operational machinery with field-tested defaults.
- **Demod mode overrides** (`p25_phase1_demod_mode: cqpsk` and friends) —
  empirical switches, not preferences. The config file itself warns at length:
  don't set CQPSK because a site is "simulcast"; set it only when a strong,
  clean signal won't lock in C4FM.

## Config hygiene

Habits that kept every build in this series debuggable:

- **`config.example.yaml` is the reference, not the internet.** Every
  accepted key lives there with a comment, and `internal/config/config.go`
  is the authoritative list behind it. When in doubt, open the example file —
  it ships beside the binary.
- **Paths are relative to the config file's folder.** `../recordings` lands
  beside the config directory, not your shell's cwd — the number-one moved-rig
  surprise. Absolute paths and environment variables are taken as-is.
- **Units bite twice.** `gain` is tenths of a dB (`"496"` = 49.6 dB);
  frequencies are Hz with `_` separators encouraged. When translating from
  SDRTrunk/OP25 numbers, convert deliberately.
- **Secrets stay out of the file.** `api.auth.token_file` beats an inline
  token (and re-reads on every request, so you can rotate without a restart);
  broadcast API keys make a config unshareable, so strip them before pasting
  a config into an issue.
- **The web can edit the file for you.** Settings changes go through
  `PATCH /api/v1/settings`, which writes back to `config.yaml` *preserving
  comments and formatting* and hot-applies what it can — the
  [write-mode]({{ '/blog/deep-dives/operator-cockpit-10-write-mode/' | relative_url }})
  and [reflect-driven form]({{ '/blog/deep-dives/operator-cockpit-13-reflect-driven-config-form/' | relative_url }})
  deep dives show how. For building a config from scratch, `gophertrunk
  config` launches the Config Builder in the terminal (`config serve` for
  the web version).
- **One knob at a time, log watched.** The Part 1 rule scales all the way up:
  when a change misbehaves, the diff *is* the suspect list.

## Where to go from here

The cookbook told you *what* to do; the rest of this blog is *why it works*.
Three natural next reads:

- **Operations depth** — hardening, TLS, metrics, watchdogs, staying up:
  the [Running It For Real series]({{ '/blog/series/running-it-for-real/' | relative_url }})
  is the production companion to Parts 9–11.
- **One protocol, all the way down** — the new
  [P25 End to End series]({{ '/blog/series/p25-end-to-end/' | relative_url }})
  follows the Part 1 rig's protocol from C4FM symbols to recorded IMBE, the
  deep-dive twin of this series' starter recipe.
- **How the sausage is made** —
  [From Spec to Shipping]({{ '/blog/series/from-spec-to-shipping/' | relative_url }})
  is the methodology series: how a protocol document becomes tested decode
  code, including the verification discipline this cookbook kept citing.

And when a rig misbehaves in ways no troubleshooting table here covered:
[The Hunt]({{ '/blog/series/the-hunt/' | relative_url }}) for discovery,
[The Analog Edge]({{ '/blog/series/analog-edge/' | relative_url }}) for
everything before the ADC. The decoder can only be as good as the samples —
that rule opened this series, and it's still the best first question.

## FAQ

**Is there one config file I can copy for everything?**
Deliberately not — a kitchen-sink config running paging, ADS-B, six systems
and four feeds on hardware that doesn't exist would just fail validation.
Start from Part 1's four blocks, then add sections from the part that matches
your next goal; the map above is the merge guide.

**How do I know a key I found online is still valid?**
Check `config.example.yaml` in your installed version — it is the maintained,
commented reference, and `internal/config/config.go` behind it is
authoritative. This series verified every key it printed against that file,
but configs age; the example file ships with the release you actually run.

**What's the minimal working config?**
Part 1's: `storage.path`, `recordings.dir`, one `sdr.devices` entry, and one
`trunking.systems` entry with `protocol` and `control_channels`. Everything
else in this finale is optional and defaulted — which is the real headline of
the config design.

**Should I edit config.yaml by hand or through the web UI?**
Both are first-class. The web Settings form is generated from the config
schema itself and preserves your file's comments when it writes; hand
editing is faster for bulk changes and works headless over SSH. Whichever
you use, keep the file backed up — it is the one artifact that reproduces
your entire rig.

**Do I need to restart after every config change?**
No — settings changed through the API hot-apply where the daemon knows how
to reload them in-process, and the response tells you per-field what applied
and what needs a restart. Structural changes (new devices, new systems) want
a clean restart; watch the startup lines each part taught you.

## Series navigation

**Part 14 of 14** · ←
[Part 13: Naming Everything — Aliases, Labels & Exports]({{ '/blog/tutorials/operator-cookbook-13-naming-everything/' | relative_url }})
· [Back to the series index]({{ '/blog/series/operator-cookbook/' | relative_url }})
