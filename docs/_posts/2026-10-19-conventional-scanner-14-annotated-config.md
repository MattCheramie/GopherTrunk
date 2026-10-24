---
title: "The Conventional Scanner, Part 14: The Complete Annotated Conventional Config — Verified Key by Key"
description: "A complete scanner.conventional configuration — SDR, scanner, channels, tone gates, data decoders, recordings, storage and web tabs — with every key checked against config.example.yaml and the Go struct tags, the validation rule and default behind each one, the part of this series that explains it, and a startup checklist built from the log."
category: tutorials
keywords: gophertrunk conventional config, scanner conventional yaml, squelch_dbfs hangtime_ms, tone ctcss_hz dcs_code, decoders mdc1200 fleetsync acars, fm_deemphasis fm_channel_bandwidth_hz, lo_offset_hz priority_interleave, talkgroup_id conventional, sdr scanner config example, gophertrunk conventional scanner
tags: [conventional-scanner, config, scanner, yaml, reference, tutorial]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "The Conventional Scanner"
series_part: 14
---

*Part 14 of **The Conventional Scanner**, a 14-part operator's tutorial on
GopherTrunk's non-trunked scan list — what you configure, what the code does
with it, what the log prints, and what is verified on air.
[Part 13]({{ '/blog/tutorials/conventional-scanner-13-reading-the-scanner-log/' | relative_url }})
read the scanner's log line by line. This closing part is the page to copy:
one complete configuration, every key verified against `config.example.yaml`
and the `yaml:` tags in `internal/config/config.go`, annotated with the
default, the validation rule, and the part that explains it.*

> **TL;DR:** A conventional rig needs five blocks. `sdr` with a `role: voice`
> dongle (the scanner takes the **last** voice SDR; `sdr.sample_rate` is the
> rate every per-channel meter is built from). `scanner` with `lo_offset_hz`
> (0 = auto, `convScannerLOOffsetHz`), `priority_interleave` (≥ 0) and the
> `conventional` list — per channel `label`, `frequency_hz` (required),
> `mode` (`fm|nfm|am`), `squelch_dbfs` (−50) or `squelch_cn_db` (12),
> `hangtime_ms` (1500), `activity_debounce_ms` (50),
> `squelch_hysteresis_db` (3), `priority`, `talkgroup_id` (collision-checked
> against `0x80000000|index`), `gain` (tenths of a dB or `auto`), `tone`
> (`mode`, `ctcss_hz` 50..300, `dcs_code` 3 octal digits, `dcs_polarity`
> `normal|inverted|both`) and `decoders` (`mdc1200|fleetsync|acars`, `acars`
> needs `mode: am`). `recordings` with the four `fm_*` keys. `storage.path`,
> without which lockouts and the data-decoder logs are runtime-only. And
> `web.tabs` to show or hide the `mdc1200` / `fleetsync` / `acars` panels.
> Every rule here is `validateConvChannel` or `New`'s defaults; the startup
> checklist at the end is the log of Part 13 in order.

**Key takeaways**

- **`frequency_hz` is the only required key.** Everything else has a default
  applied in `conventional.New`, and the config-builder's help text, the
  example file and the struct tags are policed by tests.
- **Two squelches, one per mode.** FM/NFM channels read `squelch_dbfs`
  through the ±8 kHz in-channel meter; AM channels ignore it and read
  `squelch_cn_db`.
- **Decoders and lockouts need `storage.path`.** Without it the decoders run
  but their routes return 503, and lockouts are forgotten at restart —
  `gophertrunk doctor` says so.
- **Check it with the log, not by ear.** Five startup lines and one dwell
  pair prove every block of this file is live.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| Scanner dongle | the last `role: voice` SDR in the pool | `daemon.go` (`voiceEntries[len(voiceEntries)-1]`) |
| Channel struct | every `yaml:` tag below | `config.ConvChannelConfig`, `config.ConvToneConfig` |
| Validation | required / range / collision rules | `validateConvChannel`, `config_validate.go` |
| Defaults | −50 dBFS, 1500 ms, 50 ms, 3 dB, `fm`, 12 dB C/N | `conventional.New` (`scanner.go`), `DefaultAMSquelchCNDb` |
| Analog audio | de-emphasis, LPF, HPF, IF bandwidth | `RecordingsConfig` `fm_*` keys (`config.go`) |
| Persistence | lockouts, `mdc1200_log` / `fleetsync_log` / `acars_log` | `storage.path`, `preflight.go` |
| Example file | the authoritative commented copy | `config.example.yaml` (`scanner:` block) |

## In this post

- **The SDR and the scanner's dongle** — which radio the scanner takes.
- **The scanner block** — mode, offset, interleave, and the channel list in full.
- **Recordings, storage, retention, web** — the keys outside `scanner:` the rig depends on.
- **The key table** — default, rule, and the part that explains each key.
- **Startup checklist and wrap-up** — the lines that prove it, and where the series stands.

## The SDR and the scanner's dongle

The conventional scanner is constructed when `scanner.conventional` lists
channels, when `manual_tune_enabled: true`, or when two or more `role:
voice` SDRs exist and `manual_tune_disabled` is false. It takes the **last**
voice SDR in the pool so a trunking engine keeps the first N−1 for grants.
A scanner-only rig therefore needs one voice-role dongle; a rig that also
trunks needs a second.

```yaml
sdr:
  sample_rate: 2_400_000     # every per-channel meter is sized from this rate
  devices:
    - serial: "00000002"     # match `gophertrunk sdr list`
      role: voice            # the scanner takes the LAST voice device
      ppm: 0
      gain: "280"            # TENTHS of a dB: "280" = 28.0 dB; "auto" for AGC
```

`sdr.sample_rate` matters more here than on a trunked rig: `daemon.go`
passes `float64(cfg.SDR.SampleRate)` to the scanner as `SampleRateHz`, and
the in-channel power meter, the AM carrier-to-noise meter, both tone
detectors and the data front end are all built from it
([Part 4]({{ '/blog/tutorials/conventional-scanner-04-squelch-on-the-channel/' | relative_url }}),
[Part 5]({{ '/blog/tutorials/conventional-scanner-05-ctcss-done-right/' | relative_url }})).
The `gain` unit trap is the same as everywhere else in the file — tenths of
a dB — and the per-channel `gain` below follows the same rule
(`convChannelGain` reuses `parseGain` and `gainLooksLikeDBMistake`).

## The scanner block

Every key below exists in `config.ScannerConfig` or `config.ConvChannelConfig`
with the spelling shown; the comments name the default, the rule, and the part.

```yaml
scanner:
  scan_mode: all             # all | list — the ENGINE's talkgroup gate; a conventional
                             # call never passes through it (Part 12)
  manual_tune_enabled: false # force the scanner even with no channels (TUI `f`)
  manual_tune_disabled: false # veto the "spare voice SDR" auto-detect
  lo_offset_hz: 0            # 0 = clip-safe offset chosen per rate (150 kHz..35 % of
                             # sample_rate); > 0 pins it; < 0 tunes on-channel (Part 3)
  priority_interleave: 3     # ≥ 0; after 3 ordinary picks visit the next channel
                             # with a `priority` (Part 12)
  conventional:
    - label: "Sheriff Repeater"
      frequency_hz: 155895000      # REQUIRED; the actual carrier
      mode: nfm                    # fm | nfm | am (default fm; nfm narrows the audio LPF)
      squelch_dbfs: -48            # in-channel power, ±8 kHz (default -50) — Part 4
      squelch_hysteresis_db: 3     # close-side margin below squelch_dbfs (default 3)
      activity_debounce_ms: 50     # sustained activity before the countdown resets (default 50)
      hangtime_ms: 1500            # silence before the call ends (default 1500) — Part 1
      priority: 1                  # > 0 marks a priority channel for the interleave
      talkgroup_id: 2147500000     # pins the synthetic TG; unset = 0x80000000|index — Part 2
      tone:
        mode: ctcss                # ctcss | dcs | none (default none)
        ctcss_hz: 100.0            # 50..300 Hz; EIA tones — Part 5
    - label: "Fire Dispatch"
      frequency_hz: 154190000
      mode: nfm
      gain: "280"                  # per-channel tuner gain, same unit as sdr.devices[].gain
      tone:
        mode: dcs
        dcs_code: "023"            # 3 octal digits — Part 6
        dcs_polarity: normal       # normal (D023N, default) | inverted (D023I) | both
      decoders: [mdc1200]          # Motorola PTT ID on this channel's IQ — Part 11
    - label: "Kenwood fleet"
      frequency_hz: 462562500
      mode: nfm
      decoders: [fleetsync]        # ANI bursts; may be listed with mdc1200
    - label: "Tower"
      frequency_hz: 118700000      # the ACTUAL carrier: "118.005" on 8.33 kHz is 118.000 MHz
      mode: am                     # envelope detection + carrier-to-noise squelch — Part 8
      squelch_cn_db: 12            # default 12; noise alone reads ~3–7 dB; squelch_dbfs ignored
    - label: "ACARS primary"
      frequency_hz: 131550000
      mode: am                     # acars REQUIRES mode: am
      decoders: [acars]            # Part 10
```

Three rules bite at validation time
(`validateConvChannel`, `internal/config/config_validate.go`). `frequency_hz`
is required and `mode` must be `fm`, `nfm` or `am`. The tone block is
checked per mode: `ctcss_hz` outside 50..300 Hz, a `dcs_code` that is not
three octal digits, or a `dcs_polarity` other than `normal|n|inverted|i|both`
is refused. And the effective talkgroup ID — `talkgroup_id`, or
`0x80000000 | index` when unset — must be unique across the list, because
the engine keys a synthetic call on it; two channels folding into one ID is
rejected with "effective talkgroup id … collides". `priority` has no range
check despite the struct comment's "1..10"; the scanner treats any value
above zero as a priority channel. `hangtime_ms`, `activity_debounce_ms` and
`squelch_hysteresis_db` fall back to their defaults at zero, and the two
latter are refused when negative.

<figure class="lab-figure">
<svg viewBox="0 0 680 220" width="680" height="220" role="img" aria-label="A map from configuration keys to the stage of the conventional scanner each one feeds. Along the top, five stages in order: the tuner and LO offset, the in-channel squelch, the tone gate, the dwell and engine, and the composer and recorder. Beneath each stage the keys that configure it: sdr sample_rate and lo_offset_hz and gain under the tuner; squelch_dbfs, squelch_cn_db and squelch_hysteresis_db under the squelch; tone mode, ctcss_hz, dcs_code and dcs_polarity under the gate; hangtime_ms, activity_debounce_ms, priority, priority_interleave and talkgroup_id under the dwell; fm_deemphasis, fm_audio_lowpass_hz, fm_audio_highpass_hz and fm_channel_bandwidth_hz under the composer. A separate row shows decoders feeding the data front end, which bypasses the tone gate and lands in storage.path.">
  <text x="340" y="14" text-anchor="middle" fill="currentColor" font-size="10" font-weight="bold">which key feeds which stage</text>
  <rect x="10" y="30" width="120" height="30" fill="none" stroke="currentColor"/><text x="70" y="49" text-anchor="middle" fill="currentColor" font-size="9">tuner + LO offset</text>
  <rect x="146" y="30" width="120" height="30" fill="none" stroke="currentColor"/><text x="206" y="49" text-anchor="middle" fill="currentColor" font-size="9">in-channel squelch</text>
  <rect x="282" y="30" width="120" height="30" fill="none" stroke="currentColor"/><text x="342" y="49" text-anchor="middle" fill="currentColor" font-size="9">tone gate</text>
  <rect x="418" y="30" width="120" height="30" fill="none" stroke="currentColor"/><text x="478" y="49" text-anchor="middle" fill="currentColor" font-size="9">dwell → engine</text>
  <rect x="554" y="30" width="120" height="30" fill="none" stroke="currentColor"/><text x="614" y="49" text-anchor="middle" fill="currentColor" font-size="9">composer → recorder</text>
  <path d="M130 45 L146 45 M266 45 L282 45 M402 45 L418 45 M538 45 L554 45" stroke="var(--fg-muted)"/>
  <text x="70" y="82" text-anchor="middle" fill="var(--fg-muted)" font-size="8">sdr.sample_rate</text>
  <text x="70" y="94" text-anchor="middle" fill="var(--fg-muted)" font-size="8">lo_offset_hz</text>
  <text x="70" y="106" text-anchor="middle" fill="var(--fg-muted)" font-size="8">gain (device / channel)</text>
  <text x="206" y="82" text-anchor="middle" fill="var(--fg-muted)" font-size="8">squelch_dbfs (fm/nfm)</text>
  <text x="206" y="94" text-anchor="middle" fill="var(--fg-muted)" font-size="8">squelch_cn_db (am)</text>
  <text x="206" y="106" text-anchor="middle" fill="var(--fg-muted)" font-size="8">squelch_hysteresis_db</text>
  <text x="342" y="82" text-anchor="middle" fill="var(--fg-muted)" font-size="8">tone.mode</text>
  <text x="342" y="94" text-anchor="middle" fill="var(--fg-muted)" font-size="8">ctcss_hz · dcs_code</text>
  <text x="342" y="106" text-anchor="middle" fill="var(--fg-muted)" font-size="8">dcs_polarity</text>
  <text x="478" y="82" text-anchor="middle" fill="var(--fg-muted)" font-size="8">hangtime_ms · activity_debounce_ms</text>
  <text x="478" y="94" text-anchor="middle" fill="var(--fg-muted)" font-size="8">priority · priority_interleave</text>
  <text x="478" y="106" text-anchor="middle" fill="var(--fg-muted)" font-size="8">talkgroup_id · label</text>
  <text x="614" y="82" text-anchor="middle" fill="var(--fg-muted)" font-size="8">fm_deemphasis</text>
  <text x="614" y="94" text-anchor="middle" fill="var(--fg-muted)" font-size="8">fm_audio_lowpass/highpass_hz</text>
  <text x="614" y="106" text-anchor="middle" fill="var(--fg-muted)" font-size="8">fm_channel_bandwidth_hz</text>
  <line x1="10" y1="130" x2="670" y2="130" stroke="var(--fg-muted)" stroke-dasharray="2 2"/>
  <rect x="146" y="146" width="120" height="30" fill="none" stroke="var(--accent)" stroke-width="1.5"/><text x="206" y="165" text-anchor="middle" fill="var(--accent)" font-size="9">decoders: [...]</text>
  <path d="M266 161 L418 161" stroke="var(--accent)"/>
  <text x="342" y="156" text-anchor="middle" fill="var(--accent)" font-size="8">data front end — bypasses the tone gate</text>
  <rect x="418" y="146" width="120" height="30" fill="none" stroke="var(--accent)" stroke-width="1.5"/><text x="478" y="165" text-anchor="middle" fill="var(--accent)" font-size="9">bus → storage.path</text>
  <text x="340" y="205" text-anchor="middle" fill="var(--fg-muted)" font-size="9">lockouts and the mdc1200 / fleetsync / acars logs persist only with storage.path set</text>
</svg>
<figcaption>Each key configures exactly one stage; the data decoders run beside the tone gate, not behind it.</figcaption>
</figure>

## Recordings, storage, retention, web

Four blocks outside `scanner:` decide what a dwell becomes.

```yaml
recordings:
  dir: "../recordings"
  sample_rate: 8000
  # format: flac                   # wav (default) | flac
  fm_deemphasis: us                # us/75us (default) | eu/50us | off — Part 7
  fm_audio_lowpass_hz: 3400        # post-demod LPF corner; 0 = default 3400; < 0 off
  fm_audio_highpass_hz: 300        # strips DC bias + sub-audible tones; 0 = default 300
  fm_channel_bandwidth_hz: 12500   # analog-FM IF channel TOTAL width, 2500..50000;
                                   # 0 = the legacy 25 kHz front end
storage:
  path: "../data/calls.db"         # lockout memory + the decoder logs live here
retention:
  log_days: 30                     # sweeps mdc1200 / fleetsync / acars logs by received_at
web:
  tabs:
    scanner: true
    mdc1200: true
    fleetsync: true
    acars: true                    # false hides a tab from the nav strip only
```

The `fm_*` keys are `RecordingsConfig` fields (`FMDeEmphasis`,
`FMAudioLowpassHz`, `FMAudioHighpassHz`, `FMChannelBandwidthHz` in
`internal/config/config.go`) and apply to the conventional scanner's FM voice
chain and to analog SmartNet/Type II voice; digital protocols size their own
filters and are unaffected
([Part 7]({{ '/blog/tutorials/conventional-scanner-07-fm-voice-chain/' | relative_url }})).
AM channels skip de-emphasis entirely
([Part 8]({{ '/blog/tutorials/conventional-scanner-08-am-for-the-air-band/' | relative_url }})).
`storage.path` is the one key two different subsystems silently depend on:
`convLockoutPersistence` returns nil/nil without it (lockouts become
runtime-only), and the data decoders' logs are never wired, so
`GET /api/v1/mdc1200/messages` and its siblings return 503 — the
`preflight.go` warning ("storage.path is empty but these decoders need it
…") names `mdc1200`, `fleetsync` or `acars` when a scan-list channel lists
them. The `web.tabs` keys are listed in `config.example.yaml`; the feature
tabs are web-only, so hiding them is a no-op in the TUI.

## The key table

| Key | Default · rule | Explained in |
|---|---|---|
| `lo_offset_hz` | 0 = auto; > 0 pinned (≤ 35 % of rate); < 0 off | [Part 3]({{ '/blog/tutorials/conventional-scanner-03-offset-tuning/' | relative_url }}) |
| `priority_interleave` | 0 = off; must be ≥ 0 | [Part 12]({{ '/blog/tutorials/conventional-scanner-12-priority-interleave-and-lockouts/' | relative_url }}) |
| `frequency_hz` | required | [Part 2]({{ '/blog/tutorials/conventional-scanner-02-scan-list-as-config/' | relative_url }}) |
| `mode` | `fm`; `fm|nfm|am` | [Part 2]({{ '/blog/tutorials/conventional-scanner-02-scan-list-as-config/' | relative_url }}), [Part 8]({{ '/blog/tutorials/conventional-scanner-08-am-for-the-air-band/' | relative_url }}) |
| `squelch_dbfs` | −50; in-channel ±8 kHz; ignored on `am` | [Part 4]({{ '/blog/tutorials/conventional-scanner-04-squelch-on-the-channel/' | relative_url }}) |
| `squelch_cn_db` | 12 (`DefaultAMSquelchCNDb`); ≥ 0; `am` only | [Part 8]({{ '/blog/tutorials/conventional-scanner-08-am-for-the-air-band/' | relative_url }}) |
| `hangtime_ms` / `activity_debounce_ms` / `squelch_hysteresis_db` | 1500 / 50 / 3; zero = default | [Part 1]({{ '/blog/tutorials/conventional-scanner-01-what-conventional-means/' | relative_url }}) |
| `priority` / `talkgroup_id` | 0 = unset; effective ID must be unique | [Part 12]({{ '/blog/tutorials/conventional-scanner-12-priority-interleave-and-lockouts/' | relative_url }}), [Part 2]({{ '/blog/tutorials/conventional-scanner-02-scan-list-as-config/' | relative_url }}) |
| `gain` | empty = device gain; `auto` or tenths of a dB | [Part 4]({{ '/blog/tutorials/conventional-scanner-04-squelch-on-the-channel/' | relative_url }}) |
| `tone.ctcss_hz` | 50..300 Hz | [Part 5]({{ '/blog/tutorials/conventional-scanner-05-ctcss-done-right/' | relative_url }}) |
| `tone.dcs_code` / `tone.dcs_polarity` | 3 octal digits; `normal` | [Part 6]({{ '/blog/tutorials/conventional-scanner-06-dcs/' | relative_url }}) |
| `decoders` | `mdc1200|fleetsync|acars`; no repeats; `acars` ⇒ `mode: am` | [Part 10]({{ '/blog/tutorials/conventional-scanner-10-acars/' | relative_url }}), [Part 11]({{ '/blog/tutorials/conventional-scanner-11-mdc1200-fleetsync-on-scan-channels/' | relative_url }}) |

One default is applied above the channel level: `MinDwellPerChannel`
(100 ms) is raised at construction to the slowest configured detector —
`ctcssMinDwell` (350 ms) for a CTCSS gate, `dcsMinDwell` (600 ms) for DCS,
`amMinDwell` (100 ms) for an AM channel — so a gated channel is never
advanced before its detector could report. That is not a key; it follows
from the ones you set.

## Startup checklist and wrap-up

Run the daemon with this file and read `debug.log` in this order. Each line
proves one block above is live
([Part 13]({{ '/blog/tutorials/conventional-scanner-13-reading-the-scanner-log/' | relative_url }})
has every field):

1. `config: source path=… origin=…` — the file you edited is the one running;
   `no config file found` or `multiple config files found` means it is not.
2. `device opened` for serial `00000002` with `role=voice`, and no `gain
   looks like dB` WARN
   ([Field Notebook Part 1]({{ '/blog/tutorials/field-notebook-01-startup-lines/' | relative_url }})).
3. `conv: scanner LO offset tuning … mode=auto` — the `scanner:` block
   reached the scanner; `mode=no room …` means the rate is too low.
4. `conv: restored persisted lockouts count=N` (only with rows in
   `conv_lockouts`) and **no** `conv:` WARN ending in "every signal passes the
   gate" or "decoders disabled".
5. On the first keyup: `synthetic call started device=00000002 grant=…` —
   for a DCS channel preceded by `conv: DCS gate opened … nrz_inverted=`.
6. `synthetic call ended … reason=normal` then `recorder: call ended` with
   a `wav=` path under `recordings.dir` — the composer and recorder are
   attached.
7. For a decoder channel: a row in `/mdc1200`, `/fleetsync` or `/acars` with
   the channel's frequency in the Channel column — `storage.path` is wired.

That is the series. Fourteen parts ago the question was why a trunking
engine carries a conventional scanner at all; the answer turned out to be
that most of what makes a scanner trustworthy — a squelch that measures the
channel, a tone gate that opens on the right radio and not its neighbour, an
LO that sits off the channel, a lockout that survives a restart, a decoder
that rides the dwell — is mechanism, and mechanism can be verified. Where
this series stands on verification: CTCSS, DCS, AM and MDC1200-on-a-scan-channel
are on-air verified on the reporters' rigs; the data front end, offset
tuning, the in-channel squelch, lockout persistence and priority interleave
are pinned by real-air slices or failing-first tests; ACARS live, FleetSync
on a scan-list channel live, the #1184 two-dongle stall and a
noise-quieting squelch are open. The series it stands on are
[The Analog Edge]({{ '/blog/series/analog-edge/' | relative_url }}) for the
RF in front of the dongle,
[Beyond Voice]({{ '/blog/series/beyond-voice/' | relative_url }}) for the
decoders behind the dwell, the
[Cookbook's analog FM rig]({{ '/blog/tutorials/operator-cookbook-06-analog-fm-tone-out/' | relative_url }})
for a first config, and the
[Field Notebook]({{ '/blog/series/field-notebook/' | relative_url }}) for the
rest of the log.

## FAQ

**What is the smallest working scanner.conventional config?**
One `role: voice` device under `sdr.devices` and one channel with
`frequency_hz` — every other key defaults in `conventional.New`: `mode: fm`,
`squelch_dbfs: -50`, `hangtime_ms: 1500`, `activity_debounce_ms: 50`,
`squelch_hysteresis_db: 3`, no tone gate, no decoders. Add `storage.path`
before you rely on lockouts or data decoders.

**Why does validation reject my talkgroup_id?**
Because the effective ID — `talkgroup_id`, or `0x80000000 | index` when
unset — must be unique across the list; the engine keys a synthetic call on
it, so two channels sharing one would fold into one call. The error names
both indices ("effective talkgroup id … collides"). Pick distinct explicit
IDs well above `0x80000000 + len(list)`.

**Do I set squelch_dbfs on an AM channel?**
No — it is ignored there. An AM channel squelches on `squelch_cn_db` (default
12, `DefaultAMSquelchCNDb`), the carrier's C/N measured from the channel's
own spectrum, so it does not move with gain. `squelch_dbfs` applies to `fm`
and `nfm` channels through the ±8 kHz in-channel meter.

**Which keys need storage.path?**
Lockout persistence (`conv_lockouts`) and every data-decoder log —
`mdc1200_log`, `fleetsync_log`, `acars_log`. Without it lockouts are
runtime-only and the decoders run but `GET /api/v1/mdc1200/messages` and
its siblings return 503; `gophertrunk doctor` and the preflight warning name
the decoders that need it.

**Does nfm change the squelch or only the audio?**
Only the audio: `fm` and `nfm` share the same in-channel power squelch, and
`nfm` narrows the post-demod audio low-pass. The IF channel width is a
separate, recordings-level key, `fm_channel_bandwidth_hz` (12500 for a
narrow channel, 0 for the legacy 25 kHz front end).

## Series navigation

**Part 14 of 14** · ←
[Part 13: Reading the Scanner's Log — Lines, Symptoms, and What Is Still Open]({{ '/blog/tutorials/conventional-scanner-13-reading-the-scanner-log/' | relative_url }})
