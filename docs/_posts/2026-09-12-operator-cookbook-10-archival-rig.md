---
title: "The Operator's Cookbook, Part 10: The Archival Rig — FLAC, Retention & the Call Log"
description: The keep-everything GopherTrunk build — lossless FLAC voice recordings at roughly half the size of WAV, FLAC IQ taps, the SQLite call log, retention sweeps that actually run, DSD-FME-playable vocoder sidecars, and disk-budget math with real byte rates.
category: tutorials
keywords: scanner call recording archive, flac scanner recordings, sdr iq recording flac, sqlite call log scanner, recording retention policy sdr, dsd-fme imb amb files, disk space sdr recording, trunking call archive setup, gophertrunk cookbook
tags: [operator-cookbook, flac, recordings, retention, sqlite, archiving]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "The Operator's Cookbook"
series_part: 10
---

*Part 10 of **The Operator's Cookbook**, a 14-part series of complete,
copy-paste GopherTrunk builds — one working rig per part, antenna to browser.
[Part 9]({{ '/blog/tutorials/operator-cookbook-09-sharing-the-feed/' | relative_url }})
sent your calls out into the world; this part is about keeping them. The
archival rig is the build for operators who treat the recordings folder as a
record, not a cache: lossless FLAC everywhere a container exists, a queryable
SQLite call log, IQ taps that archive the radio signal itself, vocoder-frame
sidecars for cross-decoder verification — and a retention policy that keeps
all of it from eating the disk. The running thread: an archive you can't
budget is an outage on a timer.*

> **TL;DR:** `recordings.format: flac` switches per-call voice recordings to
> lossless FLAC at roughly **half the size** of WAV for speech, and the whole
> downstream chain — web playback, loudness normalize (FLAC-in/FLAC-out),
> MP3 uploads, the retention sweeper — reads either container by **content
> sniffing**, never by extension. `baseband.record[].format: flac` does the
> same for IQ taps (~30–50% smaller, mounts back as a virtual tuner
> identically), and `tap: ddc` records the channelized stream at orders of
> magnitude less than wideband. `storage.path` is the SQLite call log behind
> the History panel; the `retention:` block sweeps rows (`call_log_days`),
> decoder logs (`log_days`) and files (`files_days`) on an `interval`, logging
> `retention: deleted recordings count=N` as it goes.

**Key takeaways**

- **FLAC is first-class now, not a transcode step.** Every recorder that had
  a container at all grew a `flac` option — voice recordings, wideband and
  DDC IQ taps, auto-record, voice IQ debug — and every reader sniffs the
  content, so a mixed wav/flac archive Just Works.
- **Archive the samples, not just the audio.** A voice file answers "what was
  said"; a `tap: ddc` IQ file answers "what was received" and replays through
  `gophertrunk replay` when a decode looks wrong. The archival rig keeps both.
- **The call log is the index; files are the payload.** SQLite rows survive
  after `files_days` sweeps the audio, so History can show a 30-day record of
  a system while only 14 days of it is playable — that split is a feature,
  and it's configurable per axis.
- **Retention math is arithmetic, not vibes.** 8 kHz 16-bit voice is
  16 KB/s of talk time; 2.4 MS/s wideband IQ is 9.6 MB/s of wall time. Do
  the multiplication before the disk does it for you — the worked table is
  below.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| FLAC voice recordings | lossless, ~half of WAV for speech | `recordings.format: flac` |
| FLAC IQ taps | ~30–50% smaller, replays identically | `baseband.record[].format: flac` (also `auto_record.format`, `voice_iq_debug.format`) |
| Small IQ archives | channelized stream, not the whole band | `baseband.record[].tap: ddc` |
| Call log database | SQLite behind History + `/api/v1/calls` | `storage.path`, [call-log deep dive]({{ '/blog/deep-dives/recording-streaming-10-call-log-sqlite/' | relative_url }}) |
| Retention sweeps | rows, decoder logs, files — independent clocks | `retention.call_log_days`, `log_days`, `files_days`, `interval` |
| Vocoder-frame sidecars | DSD-FME-playable `.imb`/`.amb` per call | `recordings.mbe_files`, [vocoders guide]({{ '/vocoders.html' | relative_url }}) |
| Naming & layout | date trees, freq-in-filename | `recordings.filename_template`, `path_template` |

## In this post

- **What you're building** — a self-pruning archive of audio, metadata and IQ.
- **The shopping list** — one real purchase: the disk.
- **The config** — FLAC everywhere, retention clocks, sidecars.
- **The disk budget** — real byte rates for every stream you can archive.
- **First run — what healthy looks like** — sweep lines and FLAC files.
- **When it doesn't work** — disks filling, sweeps not sweeping, players balking.

## What you're building

Any working rig from Parts 1–9, with its storage story made deliberate. Per
call you keep up to four artifacts: the **audio** (`.flac`), the **JSON
sidecar** with full metadata, the optional **raw vocoder frames**
(`write_raw`), and the optional **DSD-FME-playable sidecar**
(`mbe_files`: `.imb` for P25 Phase 1, `.amb` for DMR/NXDN/P25 Phase 2).
Alongside them, one SQLite database indexes every call — the same store the
History panel and `GET /api/v1/calls` query — and, if you choose, standing IQ
taps archive the receiver's actual input for replay. The deep-dive series
covers each layer's internals
([WAV on disk]({{ '/blog/deep-dives/recording-streaming-05-wav-on-disk/' | relative_url }}),
[sidecars & naming]({{ '/blog/deep-dives/recording-streaming-06-segmentation-naming-sidecars/' | relative_url }}),
[retention]({{ '/blog/deep-dives/recording-streaming-11-retention-housekeeping/' | relative_url }}));
this recipe assembles them into one build.

The FLAC support is new and worth a paragraph of trust-building: it isn't a
post-processing transcode — the recorder writes FLAC natively through one
shared encode core, and everything downstream identifies the container by
sniffing the `fLaC` marker in the file. Loudness normalization rewrites
FLAC-in/FLAC-out, the broadcast MP3 transcode from Part 9 reads it, the
`/calls/{id}/audio` endpoint serves it as `audio/flac`, and the retention
sweeper counts it. Duration and dead-key accounting use *uncompressed* PCM
byte counts, so call bookkeeping is container-independent. Two deliberate
exceptions stay raw: `diversity_capture` stays cs16 (its branch-alignment
invariant depends on it) and the hunt's survey capture stays f32.

<figure class="lab-figure">
<svg viewBox="0 0 680 252" width="680" height="252" role="img" aria-label="Storage map of the archival rig: one decoded call fans into four on-disk artifacts, a FLAC audio file, a JSON sidecar, raw vocoder frames and a DSD-FME sidecar, plus one row in the SQLite call log; separately the SDR IQ stream feeds a wideband or DDC baseband record tap writing FLAC IQ; a retention sweeper with three independent clocks, files at 14 days, call rows at 30 days and decoder logs at 30 days, prunes each store on its own schedule">
  <rect x="10" y="30" width="96" height="36" rx="4" fill="none" stroke="currentColor"/>
  <text x="58" y="52" text-anchor="middle" fill="currentColor" font-size="10">decoded call</text>
  <rect x="10" y="150" width="96" height="36" rx="4" fill="none" stroke="currentColor"/>
  <text x="58" y="166" text-anchor="middle" fill="currentColor" font-size="10">SDR IQ</text>
  <text x="58" y="179" text-anchor="middle" fill="var(--fg-muted)" font-size="9">2.4 MS/s</text>
  <line x1="106" y1="48" x2="150" y2="48" stroke="currentColor"/>
  <rect x="150" y="16" width="230" height="104" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="265" y="32" text-anchor="middle" fill="var(--accent)" font-size="10">per call — recordings/&lt;system&gt;/&lt;tg&gt;/</text>
  <text x="265" y="50" text-anchor="middle" fill="currentColor" font-size="10">audio .flac (~½ of WAV)</text>
  <text x="265" y="66" text-anchor="middle" fill="var(--fg-muted)" font-size="9">.json sidecar · .raw frames</text>
  <text x="265" y="82" text-anchor="middle" fill="var(--fg-muted)" font-size="9">.imb / .amb (mbe_files)</text>
  <text x="265" y="104" text-anchor="middle" fill="currentColor" font-size="10">+ one row in calls.db (SQLite)</text>
  <line x1="106" y1="168" x2="150" y2="168" stroke="currentColor"/>
  <rect x="150" y="140" width="230" height="60" rx="6" fill="none" stroke="currentColor"/>
  <text x="265" y="158" text-anchor="middle" fill="currentColor" font-size="10">baseband.record tap</text>
  <text x="265" y="174" text-anchor="middle" fill="var(--fg-muted)" font-size="9">wideband: whole band, ~34 GB/h wav</text>
  <text x="265" y="188" text-anchor="middle" fill="var(--fg-muted)" font-size="9">ddc: one channel, ~0.7 GB/h — format: flac</text>
  <rect x="440" y="52" width="226" height="130" rx="6" fill="none" stroke="var(--fg-muted)" stroke-dasharray="4 3"/>
  <text x="553" y="72" text-anchor="middle" fill="var(--fg-muted)" font-size="10">retention sweeper (every interval)</text>
  <text x="553" y="96" text-anchor="middle" fill="currentColor" font-size="10">files_days: 14 → audio + sidecars</text>
  <text x="553" y="118" text-anchor="middle" fill="currentColor" font-size="10">call_log_days: 30 → calls.db rows</text>
  <text x="553" y="140" text-anchor="middle" fill="currentColor" font-size="10">log_days: 30 → decoder log tables</text>
  <text x="553" y="164" text-anchor="middle" fill="var(--fg-muted)" font-size="9">0 on any axis = that sweep disabled</text>
  <line x1="380" y1="68" x2="440" y2="90" stroke="var(--fg-muted)"/>
  <line x1="380" y1="104" x2="440" y2="115" stroke="var(--fg-muted)"/>
  <text x="340" y="234" text-anchor="middle" fill="var(--fg-muted)" font-size="10">the index outlives the payload: History still shows day-30 calls whose audio was swept at day 14</text>
</svg>
<figcaption>Three stores, three clocks: audio artifacts, database rows and decoder logs each age out on their own schedule, so the searchable record outlives the playable one.</figcaption>
</figure>

## The shopping list

| Item | Price (rough) | Notes |
|---|---|---|
| Storage | ~$50+ | an SSD or spinning disk sized by the budget table below; for an [SBC build]({{ '/gophertrunk-sbc-build/' | relative_url }}), external — don't archive to the [SD card]({{ '/reference/sd-card/' | relative_url }}) that boots the box |
| Everything else | $0 | any rig from Parts 1–9 |

## The config

```yaml
storage:
  path: "../data/calls.db"

recordings:
  dir: "../recordings"
  format: flac                 # lossless, ~half the size of wav for speech
  write_raw: true              # .raw vocoder-frame sidecar per call
  mbe_files: true              # DSD-FME-playable .imb/.amb sidecars
  filename_template: "{date}_{time}_{tg}_{freq}"
  path_template: "{system}/{year}/{month}/{day}"   # date-tree layout

retention:
  files_days: 14               # audio + sidecars swept after 14 days
  call_log_days: 30            # SQLite call rows kept twice as long
  log_days: 30                 # pager/aprs/vessel/… decoder-log tables
  interval: "1h"               # sweeper cadence; 0 on any *_days disables it

baseband:
  record:
    - serial: "00000001"       # the control dongle from Part 1
      dir: "../iq/"
      tap: ddc                 # the channelized CC stream, not the whole band
      format: flac
```

Choices worth explaining. **`format: flac`** costs CPU only at call
finalization and buys ~2× more days per gigabyte, with no fidelity trade —
it decodes bit-identical. **`path_template`** is taste, but a date tree keeps
directory sizes sane at archive scale. **`tap: ddc`** is the archival sweet
spot for IQ: it records the down-converted channel the decoder actually
consumed (48 kHz for the C4FM family, 144 kHz for TETRA), directly
replayable with `gophertrunk replay` — where a wideband tap at the full SDR
rate is for short diagnostic grabs, not standing archival. For event-driven
IQ instead of a standing tap, `baseband.auto_record` fires captures on
encrypted/emergency/concurrent-call triggers and takes `format: flac` too.

**`mbe_files`** is the cross-verification lever: each call gets a `.imb`
(P25 Phase 1) or `.amb` (DMR/NXDN/P25 Phase 2) file in DSD-FME's native
container, so `dsd-fme -f1 -w out.wav -r call.imb` re-decodes your call with
an entirely independent vocoder — the workflow the
[vocoders guide]({{ '/vocoders.html' | relative_url }}) documents, and the
one that let us measure GopherTrunk's AMBE+2 high-band deficit against
mbelib frame-for-frame. TETRA and ProVoice have no DSD-FME playback mode and
produce none.

## The disk budget

The rates are exact; the durations are what they buy on a **1 TB** disk:

| Stream | Rate | 1 TB holds |
|---|---|---|
| Voice WAV (8 kHz, 16-bit mono) | 16 KB/s of *talk time* | ~17,000 h of talk |
| Voice FLAC | ~8 KB/s of talk time | ~35,000 h of talk |
| DDC IQ tap, C4FM family (48 kHz cs16-in-container) | 192 KB/s wall time | ~60 days continuous (WAV); ~3–4 months FLAC |
| DDC IQ tap, TETRA (144 kHz) | 576 KB/s wall time | ~20 days (WAV); ~1 month FLAC |
| Wideband IQ tap (2.4 MS/s) | 9.6 MB/s wall time | ~29 h (WAV); ~40–60 h FLAC |

Two readings. First, **voice archives are effectively free**: a busy system
doing two hours of actual talk per day writes ~58 MB/day as FLAC —
`files_days: 0` (never sweep audio) is legitimate on any modern disk, and
the config above sweeps files mainly to bound the IQ tap. Second,
**wideband IQ is a different regime entirely** — three orders of magnitude —
which is why the standing tap is `ddc` and wideband belongs to the
event-driven `auto_record` with its `seconds:` cap. The IQ FLAC savings are
real but signal-dependent; budget on the WAV number and treat compression as
margin — the [capture-discipline]({{ '/blog/tutorials/analog-edge-10-capture-discipline/' | relative_url }})
habit.

## First run — what healthy looks like

Restart and let a few calls land. The recorder lines look exactly like
Part 1's, with the extension telling you the container took:

```
INF recorder: call started device=... wav=../recordings/Metro-P25/2026/09/12/20260912_141212_9001_857262500.flac tg=9001 vocoder=imbe
INF recorder: call ended device=... duration=6.42s reason=released
```

Play the call in the web History panel (served as `audio/flac`), and confirm
the `.json`, `.raw` and `.imb`/`.amb` siblings landed next to it. Then, within
the first `interval` (and every hour after), the sweeper reports — only when
it deletes something:

```
INF retention: deleted recordings count=214
INF retention: deleted call rows count=3120
INF retention: deleted log rows table=pager_log count=87
```

Silence from the sweeper on a young archive is normal — nothing has aged
past a threshold yet; `POST /api/v1/retention/sweep` forces a pass. Finally
check the IQ tap: a growing `.flac` in `../iq/` that `gophertrunk replay`
accepts — the replay driver sniffs the `fLaC` marker, never the extension,
so even a misnamed file mounts correctly.

## When it doesn't work

| Symptom | Likely cause | Fix |
|---|---|---|
| Disk filling despite `retention:` configured | the standing tap is `tap: wideband`, or `files_days: 0` | Wideband is ~34 GB/h — switch the standing tap to `ddc` and leave wideband to `auto_record` with a `seconds:` cap |
| No `retention:` lines ever | all thresholds `0` (each `0` disables that sweep), or nothing is old enough yet | Set non-zero days; force a pass via `POST /api/v1/retention/sweep` and watch for the `deleted` lines |
| `retention: file sweep failed` / `retention: rm failed` | permissions or a vanished mount under `recordings.dir` | The sweeper logs the path — fix ownership/mount; on the Part 11 systemd build check `ReadWritePaths` |
| A player won't open the recordings | an old external tool that predates FLAC | Everything *inside* GopherTrunk (web player, normalize, MP3 uploads) sniffs content and reads either; for stubborn external tooling, set `format:` back to `wav` — the archive can mix containers freely |
| History shows calls with no playable audio | working as designed: `call_log_days` > `files_days` | Rows outlive files on purpose; align the two values if you want them to expire together |
| No `.imb`/`.amb` for some calls | TETRA / ProVoice / analog have no DSD-FME playback mode | Expected — those protocols produce no sidecar; the `.raw` frames still archive if `write_raw` is on |
| DSD-FME plays a sidecar slow and low-pitched | DSD-FME's `-w` writer stamps 8 kHz on 12 kHz synthesis | Upstream quirk, [documented]({{ '/vocoders.html' | relative_url }}) — use its per-call mode or relabel the WAV to 12 kHz |
| Recording rate isn't what you set | digital voice forces vocoder-native output | `recorder: forcing WAV rate to vocoder-native 8000` — `recordings.sample_rate` applies to analog/NBFM only |

### How this recipe shapes operator practice

- **Archive the evidence, not just the product.** Every hard bug this project
  has chased ended with "get the raw capture" — the `ddc` tap means the
  capture already exists when the bad decode happens.
- **Let the index outlive the payload.** Cheap rows, expensive files,
  separate clocks: query 30 days, store 14, and nobody has to choose one
  number for both.
- **Verify with a second decoder.** `mbe_files` exists so your archive can be
  cross-examined; a claim about audio quality backed by a paired `.amb` is
  measurement, not opinion.

## Variations

- **Maximum-compatibility archive.** `format: wav` everywhere and accept 2×
  the bytes — every audio tool since the 1990s opens it. Mixing is fine:
  readers sniff per file, so switching formats mid-archive breaks nothing.
- **Frames-only cold storage.** `write_raw` + `mbe_files` with aggressive
  `files_days`: vocoder frames are a few hundred bytes per second, and the
  audio can be re-synthesized from them offline via `gophertrunk decode`.
- **Forensic maximalist.** Add `auto_record` (event-triggered wideband
  slices) and `voice_iq_debug` (per-call channelized voice IQ, `format: flac`)
  — every questionable call arrives with its own replayable IQ attached.
- **Network storage.** The `dir` keys take absolute paths, so a NAS mount
  works — but keep `storage.path` (SQLite) on local disk; databases and
  network filesystems are old enemies.

## Where this goes next

An archive this good deserves better than a laptop that gets rebooted for OS
updates. [Part 11]({{ '/blog/tutorials/operator-cookbook-11-closet-appliance/' | relative_url }})
turns the rig into a closet appliance: a Pi or mini-PC under systemd with
hardening that actually fits an SDR daemon, Docker USB pass-through for the
container crowd, watchdogs for dongles that fall off the bus, and the
graceful-shutdown work that made `systemctl restart` take milliseconds
instead of thirty silent seconds.

## FAQ

**Should I record scanner calls as FLAC or WAV?**
FLAC, unless a specific external tool you depend on can't read it. It's
lossless — bit-identical audio at roughly half the size for speech — and
GopherTrunk's entire downstream chain (web playback, loudness normalize, MP3
upload, retention) reads both containers transparently by content sniffing.

**How much disk space do SDR call recordings use?**
At the vocoder-native 8 kHz mono, WAV costs 16 KB per second of talk time
and FLAC about half that — so ~29 MB per hour of actual talk as FLAC. Even
busy systems accumulate slowly; it's IQ recording (192 KB/s for a DDC
channel tap, 9.6 MB/s for a 2.4 MS/s wideband tap) that needs a real budget.

**Can I replay a FLAC IQ recording like a cs16 or WAV capture?**
Yes — `baseband.replay` and `gophertrunk replay` mount a `.flac` IQ file as
a virtual tuner exactly like a WAV, detecting the container from the `fLaC`
marker in the file body rather than the extension. A DDC-tap recording
replays through the same channel-rate decode path the live daemon used.

**Why are old calls still in History after retention deleted the files?**
Because the sweeps are independent: `files_days` ages out audio and sidecars
on disk while `call_log_days` ages out SQLite rows, and the config above
deliberately keeps rows longer. Set both to the same value if you want the
searchable record and the playable record to expire together.

**Do the .imb/.amb files replace the audio recording?**
No — they're sidecars carrying the raw vocoder frames in DSD-FME's native
container, written alongside the normal recording when
`recordings.mbe_files: true`. Their job is independent verification and very
compact archival, not playback convenience. Note DSD-FME's own `-w` writer
stamps an 8 kHz header on 12 kHz synthesis (plays ~1.5× slow) — an upstream
quirk documented in the [vocoders guide]({{ '/vocoders.html' | relative_url }}),
not a sidecar defect.

## Series navigation

**Part 10 of 14** · ←
[Part 9: Sharing the Feed — Broadcastify, OpenMHz & Friends]({{ '/blog/tutorials/operator-cookbook-09-sharing-the-feed/' | relative_url }})
· Next →
[Part 11: The Closet Appliance — Pi, systemd & Docker]({{ '/blog/tutorials/operator-cookbook-11-closet-appliance/' | relative_url }})
