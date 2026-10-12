---
title: "The Conventional Scanner, Part 2: The Scan List as Config — Channels, Modes, Hangtime, Priority"
description: "Every key of a scanner.conventional entry verified against the Go struct tags and config.example.yaml — mode, squelch_dbfs, hangtime_ms, activity_debounce_ms, squelch_hysteresis_db, priority, talkgroup_id, gain, tone and decoders — what the daemon does with each, where the analog-FM audio keys really live, and the config-discovery log lines that say which file the daemon is actually running."
category: tutorials
keywords: scanner.conventional config, conventional channel yaml keys, squelch_dbfs hangtime_ms, talkgroup_id conventional scanner, priority_interleave config, fm_deemphasis fm_audio_highpass_hz, gophertrunk config discovery, config source origin log, ~/.config/gophertrunk config.yaml, gophertrunk conventional scanner
tags: [conventional-scanner, config, analog-fm, scan-list, yaml, tutorial]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "The Conventional Scanner"
series_part: 2
---

*Part 2 of **The Conventional Scanner**, a 14-part operator's tutorial on
GopherTrunk's `scanner.conventional` scan list.
[Part 1]({{ '/blog/tutorials/conventional-scanner-01-what-conventional-means/' | relative_url }})
followed one channel visit through the dwell loop — scan window, synthetic
grant, hangtime, stall guard. This part turns to the list the loop reads:
every key of a `scanner.conventional[]` entry, checked against the
`ConvChannelConfig` struct tags and `config.example.yaml`, what the daemon
does with each, and — because the most common "my scan list changed nothing"
is a daemon reading a different file — the config-discovery lines that say
which `config.yaml` is actually running.*

> **TL;DR:** A scan-list entry is one `ConvChannelConfig`
> (`internal/config/config.go`): `label`, `frequency_hz` (required),
> `mode` (`fm` | `nfm` | `am`), `squelch_dbfs` (default −50, in-channel),
> `squelch_cn_db` (AM only, default 12), `hangtime_ms` (1500),
> `activity_debounce_ms` (50), `squelch_hysteresis_db` (3), `priority`,
> `talkgroup_id`, `gain` (`"auto"` or tenths of a dB), `tone`
> (`mode`/`ctcss_hz`/`dcs_code`/`dcs_polarity`) and `decoders`
> (`mdc1200`, `fleetsync`, `acars`). `validateConvChannel` rejects a
> missing frequency, an unknown mode, a negative debounce or hysteresis, an
> unparseable gain, a bad tone block, `acars` on a non-AM channel and a duplicated
> decoder; `validateScanner` rejects two channels whose effective
> talkgroup IDs collide. Two keys sit beside the list — `lo_offset_hz`
> (Part 3) and `priority_interleave` (Part 12) — and the analog **audio**
> keys live under `recordings` (`fm_deemphasis`, `fm_audio_lowpass_hz`,
> `fm_audio_highpass_hz`, `fm_channel_bandwidth_hz`, Part 7). At start-up
> the daemon logs `config: source path=… origin=…` or a WARN that it is
> running built-in defaults; read that line before touching a threshold.

**Key takeaways**

- **Every key maps to one field, and the daemon copies them in one
  place.** `daemon.go` builds a `conventional.Channel` per entry; the only
  arithmetic is `msToDuration` on the millisecond keys.
- **`mode: nfm` is a label today.** Only `ValidMode` reads it; the channel
  bandwidth and audio corners are the system-wide `recordings.fm_*` keys.
- **`priority` drives the scan rotation, not the engine.** It feeds
  `priority_interleave`; a conventional call is never pool-allocated, so
  the engine's preemption never sees it.
- **Pin `talkgroup_id` on day one.** The positional `0x80000000 | index`
  default renames the channel's whole history when the list is edited
  ([#1105](https://github.com/MattCheramie/GopherTrunk/issues/1105)).

## Cheat sheet

| Line / key | Meaning | Healthy / worry when |
|---|---|---|
| `frequency_hz` | the carrier the scanner tunes (via the LO offset of Part 3) | required; for AM it is the *carrier*, not an 8.33 kHz channel name |
| `squelch_dbfs` / `squelch_cn_db` | FM in-channel power open threshold / AM carrier-to-noise | −50 / 12 defaults; `squelch_dbfs` ignored on `am`, `squelch_cn_db` on FM |
| `hangtime_ms`, `activity_debounce_ms`, `squelch_hysteresis_db` | release countdown, blip debounce, close-side margin | 1500 / 50 / 3 defaults; 0 = default |
| `priority` | interleave eligibility (`Priority > 0`) | only matters with `scanner.priority_interleave > 0` |
| `talkgroup_id` | the call's `GroupID` | unset ⇒ `0x80000000 \| index`; collisions rejected by `validateScanner` |
| `gain`, `tone`, `decoders` | per-channel tuner gain, CTCSS/DCS gate, in-band data | Parts 4, 5–6, 11 |
| `config: source path=… origin=…` | which file backs this run | WARN `running built-in defaults` or `multiple config files found` |

## In this post

- **The entry, verified** — the YAML, the struct, the daemon's copy.
- **Modes** — what `fm`, `nfm` and `am` select, honestly.
- **The four timing keys** — squelch, hangtime, debounce, hysteresis and their validation.
- **Priority and identity** — what `priority` really does, and the roster behind `talkgroup_id`.
- **Gain, tone, decoders** — the keys later parts own, and their validation today.
- **Is the daemon reading your file?** — discovery precedence and the start-up lines.

## The entry, verified

The authoritative shape is the struct, not the docs. `ConvChannelConfig`
in `internal/config/config.go` carries these yaml tags, and every one
appears in the commented example block under `scanner:` in
`config.example.yaml`:

```yaml
# config.example.yaml (abridged) — scanner.conventional[]
scanner:
  lo_offset_hz: 0           # Part 3: 0 = clip-safe auto, > 0 pins Hz, < 0 disables
  priority_interleave: 0    # Part 12: 0 = plain round robin
  conventional:
    - label: "Sheriff Repeater"
      frequency_hz: 155895000
      mode: fm
      squelch_dbfs: -48       # in-channel power, ±8 kHz filter (#1239)
      # gain: "280"           # "auto" or tenths of a dB (#1239)
      hangtime_ms: 1500
      priority: 4
      # talkgroup_id: 2147500000
      tone:
        mode: ctcss           # ctcss | dcs | none
        ctcss_hz: 100.0
        # dcs_code: "023"
        # dcs_polarity: normal  # normal | inverted | both
    - label: "Tower"
      frequency_hz: 118700000
      mode: am
      squelch_cn_db: 12       # noise alone reads ~3–7 dB
      # decoders: [mdc1200, fleetsync]   # FM channels; acars needs mode: am
```

Two keys from the struct are missing from that excerpt because the example
leaves them at default: `activity_debounce_ms` and `squelch_hysteresis_db`
(both `0 = default`). The daemon's copy in `daemon.go` is a straight field
map into `conventional.Channel`: `HangtimeMs` becomes `Hangtime` through
`msToDuration(ch.HangtimeMs, 1500*time.Millisecond)`,
`ActivityDebounceMs` through `msToDuration(ch.ActivityDebounceMs, 0)` so the
scanner's own `New` applies the 50 ms default, `Gain` through
`convChannelGain` (Part 4), and the `tone` block field for field into a
`conventional.ToneConfig`. `SystemName` is hard-coded to `"scanner"`, which
is the `system` every conventional call carries.

## Modes

`mode` accepts `fm`, `nfm` and `am` (`validateConvChannel`; empty defaults
to `fm` in `New`). `am` is a real switch: `conventionalProtocol` stamps the
grant `am-conv`, `buildAMMeter` replaces the power squelch with the
carrier-to-noise meter, and the composer selects its envelope detector —
Part 8's subject. `fm` and `nfm` are **not** distinguished anywhere in the
current code: the only reader of the string outside validation is
`ValidMode`, and the struct comment's "narrows the post-demod audio LPF" is
a description of an earlier chain. What actually sets the channel width and
audio corners for every analog FM call is the `recordings` section —
`fm_channel_bandwidth_hz` (0 = the legacy 25 kHz front end, else
2500..50000; 12500 for a 12.5 kHz NFM channel), `fm_audio_lowpass_hz`
(default 3400), `fm_audio_highpass_hz` (default 300) and `fm_deemphasis`
(`us`/`75us`, `eu`/`50us`, `off`). They are system-wide, so a scan list
that mixes 25 kHz and 12.5 kHz channels runs one width for both. Part 7
takes them one by one; the point here is where they live. Write `nfm` for documentation — it is reported in the API's `mode` field —
but do not expect it to change the audio.

## The four timing keys

`squelch_dbfs` is the open threshold Part 1's `scanWindow` compares each
chunk against, measured — since
[#1239](https://github.com/MattCheramie/GopherTrunk/issues/1239) — on the
channel through a ±8 kHz filter rather than on the whole SDR span (Part 4).
The default is −50 dBFS, applied in `New` when the key is zero, and the
Cookbook's advice to start near −48 still holds. On an `am` channel it is
ignored in favour of `squelch_cn_db`, whose default is
`DefaultAMSquelchCNDb` = 12.

`hangtime_ms` (1500) is how long the channel must sit below the keep-alive
level before the call ends. `squelch_hysteresis_db` (3) sets that
keep-alive level: the squelch opens at `squelch_dbfs` but the countdown
starts only once power is 3 dB lower, so a transmitter sitting on the
threshold does not chatter the countdown. `activity_debounce_ms` (50) is
the other edge: activity must be continuous for this long before it resets
a running countdown, the
[#1090](https://github.com/MattCheramie/GopherTrunk/issues/1090) fix for a
channel that never released. The struct comment gives the useful knob: set
`activity_debounce_ms` below one chunk's duration to effectively disable
the debounce. `validateConvChannel` rejects a negative
`activity_debounce_ms` or `squelch_hysteresis_db`; `hangtime_ms` is not
range-checked, but `msToDuration` hands anything ≤ 0 to the 1500 ms
default, so a zero hangtime cannot be configured.

The relationship between the three is easiest to see as one dwell:

```text
open at squelch_dbfs (−48)
   │ active while power ≥ −48 − 3 = −51 dBFS (and tone, if gated)
   ▼
power < −51 → belowSince set, countdown runs (hangtime_ms 1500)
   │ a chunk ≥ −51 shorter than 50 ms → ignored, countdown continues
   │ activity ≥ 50 ms continuous      → countdown reset
   ▼
1500 ms below → EndSyntheticCall reason=normal
```

## Priority and identity

`priority` deserves a precise statement because the struct comment on
`conventional.Channel.Priority` says it is "forwarded to the synthetic
talkgroup so the engine's preemption logic respects it", and the code does
something narrower. The only reader of `Channel.Priority` is
`pickNextChannel` / `nextPriorityLocked`: with
`scanner.priority_interleave: N`, after every N ordinary picks the rotation
visits the next channel whose `Priority > 0`, and a priority channel the
plain rotation reaches on its own resets that clock (Part 12,
`TestConvScannerPriorityInterleave`). With `priority_interleave` at its
default 0 the key does nothing. The grant `beginDwell` builds carries no
priority at all, and `HandleSyntheticCall` stores the call in the engine's
`synthetic` map, not the voice pool where `CanPreempt` and
`LowestPriorityActiveForFrequency` operate — so no conventional call is
ever preempted or preempts. `config.example.yaml`'s `priority: 4` is an
interleave flag, nothing more.

`talkgroup_id` is the identity. `beginDwell` uses it as the grant's
`GroupID` when non-zero and `0x80000000 | index` otherwise, and
`validateScanner` computes every channel's *effective* ID and refuses
duplicates with
`scanner.conventional[i]: effective talkgroup id N collides with scanner.conventional[j]`.
The engine then does `e.talkgroups.Lookup(g.GroupID)` — the same roster a
trunked grant uses, loaded from each system's `talkgroup_file` CSV, whose
optional columns (matched by header, case-insensitive) are Alpha Tag,
Description, Mode, Tag, Group, Priority and Lockout. A row whose Decimal
equals the channel's `talkgroup_id` gives the call an alpha tag and the
roster flags; the channel's own `label` rides separately on
`Grant.GroupLabel`, so a channel with no roster row still shows its name.
Names applied from the web (`PATCH /api/v1/talkgroups/{id}`) persist in the
`labels` table and win over the file.

<figure class="lab-figure">
<svg viewBox="0 0 680 190" width="680" height="190" role="img" aria-label="Three columns showing how one scan-list entry becomes a call. Left: the YAML keys of a scanner.conventional entry. Middle: the conventional.Channel fields the daemon copies them into, with hangtime_ms converted by msToDuration and gain parsed by convChannelGain. Right: the trunking.Grant that beginDwell builds — System scanner, Protocol fm-conv or am-conv, GroupID from talkgroup_id or 0x80000000 or index, GroupLabel from label, FrequencyHz — and a note that priority goes only to pickNextChannel.">
  <text x="110" y="16" text-anchor="middle" fill="currentColor" font-size="10" font-weight="bold">config.yaml entry</text>
  <text x="340" y="16" text-anchor="middle" fill="currentColor" font-size="10" font-weight="bold">conventional.Channel</text>
  <text x="570" y="16" text-anchor="middle" fill="currentColor" font-size="10" font-weight="bold">trunking.Grant</text>
  <rect x="20" y="26" width="180" height="150" fill="none" stroke="var(--fg-muted)"/>
  <rect x="250" y="26" width="180" height="150" fill="none" stroke="var(--fg-muted)"/>
  <rect x="480" y="26" width="180" height="150" fill="none" stroke="var(--accent)"/>
  <text x="30" y="44" fill="currentColor" font-size="9">label</text>
  <text x="30" y="60" fill="currentColor" font-size="9">frequency_hz</text>
  <text x="30" y="76" fill="currentColor" font-size="9">mode</text>
  <text x="30" y="92" fill="currentColor" font-size="9">talkgroup_id</text>
  <text x="30" y="108" fill="currentColor" font-size="9">hangtime_ms</text>
  <text x="30" y="124" fill="currentColor" font-size="9">gain</text>
  <text x="30" y="140" fill="currentColor" font-size="9">priority</text>
  <text x="30" y="156" fill="currentColor" font-size="9">tone / decoders</text>
  <text x="260" y="44" fill="currentColor" font-size="9">Label</text>
  <text x="260" y="60" fill="currentColor" font-size="9">FrequencyHz</text>
  <text x="260" y="76" fill="currentColor" font-size="9">Mode</text>
  <text x="260" y="92" fill="currentColor" font-size="9">TalkgroupID</text>
  <text x="260" y="108" fill="currentColor" font-size="9">Hangtime ← msToDuration</text>
  <text x="260" y="124" fill="currentColor" font-size="9">GainTenthDB ← convChannelGain</text>
  <text x="260" y="140" fill="currentColor" font-size="9">Priority → pickNextChannel only</text>
  <text x="260" y="156" fill="currentColor" font-size="9">Tone, Decoders</text>
  <text x="490" y="44" fill="var(--accent)" font-size="9">System: "scanner"</text>
  <text x="490" y="60" fill="var(--accent)" font-size="9">Protocol: fm-conv | am-conv</text>
  <text x="490" y="76" fill="var(--accent)" font-size="9">GroupID: talkgroup_id</text>
  <text x="490" y="90" fill="var(--fg-muted)" font-size="8">   or 0x80000000 | index</text>
  <text x="490" y="108" fill="var(--accent)" font-size="9">GroupLabel: label</text>
  <text x="490" y="124" fill="var(--accent)" font-size="9">FrequencyHz, SourceID 0</text>
  <text x="490" y="148" fill="var(--fg-muted)" font-size="8">no priority field</text>
  <text x="490" y="162" fill="var(--fg-muted)" font-size="8">roster: talkgroups.Lookup(GroupID)</text>
  <line x1="200" y1="100" x2="250" y2="100" stroke="var(--fg-muted)" marker-end="none"/>
  <line x1="430" y1="100" x2="480" y2="100" stroke="var(--fg-muted)"/>
</svg>
<figcaption>Each YAML key lands on one Channel field; only label, frequency, mode and talkgroup_id reach the grant. priority stops at the scan rotation.</figcaption>
</figure>

## Gain, tone, decoders

Three keys belong to later parts but are validated now. `gain` is parsed
like `sdr.devices[].gain`: `"auto"`, or tenths of a dB as a number string
(`"280"` is 28.0 dB; `"28.0"` also reads as 28.0 dB). `validateConvChannel`
rejects anything that is neither, and `convChannelGain` in
`cmd/gophertrunk/conv_gain.go` WARNs when a value looks like whole dB
(`"28"` is 2.8 dB — write `"280"`). An entry with no gain runs at the
device's own gain, and a list where no channel sets one never touches the
tuner at all (Part 4).

`tone.mode` is `ctcss`, `dcs` or `none`/empty. `ctcss_hz` must sit in
50..300 Hz; `dcs_code` must be exactly three octal digits; `dcs_polarity`
accepts `normal`/`n`, `inverted`/`i` or `both`, case-insensitive. The
scanner re-validates the same rules in `validateTone` and, if the SDR
sample rate is missing, WARNs
`conv: tone gating configured but scanner sample rate is zero; tone gate disabled — every signal passes the gate`
rather than failing silently — the Part 5 and Part 6 detectors need the
rate to build.

`decoders` lists `mdc1200`, `fleetsync` and `acars`. The same name twice is
rejected, and `acars` requires `mode: am` with the message
`acars needs mode: am (ACARS is AM; got mode "fm")`. Decoded bursts land
in the same log tables, REST endpoints and web panels as the dedicated
`mdc1200.channels` / `fleetsync.channels` receivers, which is why the
preflight in `cmd/gophertrunk/preflight.go` counts them: with
`storage.path` empty it adds a start-up warning naming every decoder that
needs it — "they will run, but their REST endpoints … return 503 and the
web panels stay empty" — surfaced through `d.addWarning` to the launcher
menu, TUI dashboard and runtime DTO.

## Is the daemon reading your file?

A scan list that changes nothing is often a scan list in a file the daemon
never opened. With no `-config`, `main.go` calls
`config.DiscoverWith`, which walks a precedence list: `$GOPHERTRUNK_CONFIG`
verbatim, then the first candidate directory holding any `*.yaml`/`*.yml`.
`candidateDirs` (`internal/config/discover.go`) scans, in order, each
root and its `config/` subfolder: `os.UserConfigDir()/GopherTrunk`,
`os.UserConfigDir()/gophertrunk`, `~/.config/gophertrunk`,
`~/Documents/GopherTrunk`, then the current directory. The lowercase
`~/.config/gophertrunk` entry is the
[#836](https://github.com/MattCheramie/GopherTrunk/issues/836) fix: the
install docs told Linux and macOS operators to create exactly that path,
but `os.UserConfigDir()` only yields the CamelCase root, so a config placed
where the docs said was discovered only when the daemon happened to run from
that directory.

The structured log then says what won, because the stderr
`config: loaded <path>` note lands before the logger exists and never
reaches `debug.log`:

```text
INF config: source path=/home/op/.config/gophertrunk/config.yaml origin=discovered
WRN config: no config file found — running built-in defaults hint="pass -config, set GOPHERTRUNK_CONFIG, or place config.yaml in a discovered directory (gophertrunk doctor lists them)"
WRN config: multiple config files found; using the one above and ignoring the others using=… ignored=[…]
```

`origin` is `flag` for an explicit `-config`, `env` for
`$GOPHERTRUNK_CONFIG`, `discovered` for the precedence walk, or the WARN
for `defaults`. The second WARN exists because broadening the search order
(as #836 did) can change which of two files wins; `ConfigFilesElsewhere`
names the losers. Read this line before any threshold: a daemon on built-in
defaults has no scan list at all, and one reading a stale copy in
`~/Documents/GopherTrunk` explains missing squelch, gain or recording
settings with no other symptom — why
[#1184](https://github.com/MattCheramie/GopherTrunk/issues/1184) added the
line. `gophertrunk config tui` scans the same `config.CandidateDirs()`
list when it looks for a file to edit.

### How the scan list shaped the Go code

- **Two validators, one rule set.** `validateConvChannel` (config) and
  `validateTone` / `validateDecoders` (scanner) check the same constraints,
  so a hand-built `Channel` from the manual-tune API gets the same answer as
  YAML.
- **Zero means default, never zero.** Every timing key treats 0 as unset,
  so an omitted key and a `0` are the same config.
- **Identity is validated at the list level.** Collisions depend on the
  whole list's positions, so they are checked in `validateScanner`, not per
  entry.
- **Audio keys are deliberately not per channel.** The composer builds one
  FM chain configuration from `recordings`, so a per-channel bandwidth
  would need a per-call chain parameter it does not yet have.

## Where this goes next

The scanner never tunes the SDR *to* `frequency_hz`. Since
[#1184](https://github.com/MattCheramie/GopherTrunk/issues/1184) it tunes
the LO below the channel and mixes the channel back to baseband, because a
zero-IF tune of an overloaded ADC puts a tone at four times the carrier
offset straight into the audio.
[Part 3]({{ '/blog/tutorials/conventional-scanner-03-offset-tuning/' | relative_url }})
reads `cmd/gophertrunk/conv_offset.go`: the `convScannerFrontEnd`, the NCO
mix-back, the offset search that keeps every `4k·offset mod fs` product out
of the channel, and why the obvious `fs/4` is the worst possible choice.

## FAQ

**What are the valid keys for a scanner.conventional channel?**
`label`, `frequency_hz`, `mode` (`fm`/`nfm`/`am`), `squelch_dbfs`,
`squelch_cn_db`, `hangtime_ms`, `activity_debounce_ms`,
`squelch_hysteresis_db`, `priority`, `talkgroup_id`, `gain`, `tone`
(`mode`, `ctcss_hz`, `dcs_code`, `dcs_polarity`) and `decoders` — the yaml
tags on `ConvChannelConfig` in `internal/config/config.go`, all present in
`config.example.yaml`.

**Does mode: nfm narrow the channel?**
Not in the current code. Only `ValidMode` reads the string; the analog FM
channel filter and audio corners are the system-wide `recordings` keys
`fm_channel_bandwidth_hz`, `fm_audio_lowpass_hz`, `fm_audio_highpass_hz`
and `fm_deemphasis`. Set `fm_channel_bandwidth_hz: 12500` for a 12.5 kHz
list; it applies to every analog call.

**What does priority on a conventional channel do?**
It marks the channel for `scanner.priority_interleave`: after N ordinary
channels the rotation visits the next channel with `priority > 0`. With the
interleave at its default 0 it does nothing, and the engine's preemption
never applies — conventional calls live in the engine's `synthetic` map,
not the voice pool.

**Why does the daemon say it is running built-in defaults?**
No config was found on the discovery path: `$GOPHERTRUNK_CONFIG`, then the
first of `UserConfigDir/GopherTrunk`, `UserConfigDir/gophertrunk`,
`~/.config/gophertrunk`, `~/Documents/GopherTrunk` or the current directory
(each with its `config/` subfolder) that holds a `.yaml`. Pass `-config`,
or move the file; the `config: source` line confirms the fix.

**Why does the preflight warn that mdc1200 or acars needs storage.path?**
`decoders` on a scan-list channel write to the same `mdc1200_log` /
`fleetsync_log` / `acars_log` tables as the dedicated receivers.
`preflight` in `cmd/gophertrunk/preflight.go` lists each decoder that
needs `storage.path`; without it the decoders run but
`GET /api/v1/acars/messages` and its siblings return 503 and the panels
stay empty.

## Series navigation

**Part 2 of 14** · ←
[Part 1: What 'Conventional' Means to a Trunking Scanner — The Dwell Loop]({{ '/blog/tutorials/conventional-scanner-01-what-conventional-means/' | relative_url }})
· Next →
[Part 3: Offset Tuning — Why the LO Sits Below the Channel]({{ '/blog/tutorials/conventional-scanner-03-offset-tuning/' | relative_url }})
