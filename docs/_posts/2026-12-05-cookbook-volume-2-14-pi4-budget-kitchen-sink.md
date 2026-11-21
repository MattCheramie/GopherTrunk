---
title: "The Operator's Cookbook Vol. 2, Part 14: The Pi 4 Budget — and the Kitchen Sink, Revisited"
description: "The Volume 2 finale: sizing a Raspberry Pi 4 against a reporter's real drop counts — an Airspy Mini and several TETRA cells on one wideband device — then one annotated config.yaml that adds every Volume 2 section to Volume 1's kitchen sink, a goal-to-recipe table covering both volumes, and the hygiene that keeps a growing config honest."
category: tutorials
keywords: raspberry pi 4 sdr scanner cpu, airspy mini tetra raspberry pi, dropping live iq chunks, gophertrunk config.yaml reference, sdr input_sample_rate decimation, wideband sdr multiple control channels, scanner config example annotated, sdr scanner sizing guide, gophertrunk cookbook
tags: [operator-cookbook, cookbook-volume-2, raspberry-pi, config, finale, performance]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "The Operator's Cookbook, Volume 2"
series_part: 14
---

*Part 14 — the last — of **The Operator's Cookbook, Volume 2**, a second
14-part run of complete, copy-paste GopherTrunk builds — one working rig per
part.
[Part 13]({{ '/blog/tutorials/cookbook-volume-2-13-pluto-native/' | relative_url }})
put a Pluto on the network without SoapySDR. This closing part does two
jobs. First it answers the question every small rig eventually asks — how
much can a Raspberry Pi 4 carry? — with a reporter's drop counts instead of
a guess. Then it does what
[Volume 1's finale]({{ '/blog/tutorials/operator-cookbook-14-kitchen-sink-config/' | relative_url }})
did: one kitchen-sink `config.yaml`, every block stamped with the part that
owns it, and a table that turns a goal into a recipe across both volumes.*

> **TL;DR:** A Pi 4 can decode several TETRA cells from one Airspy Mini at
> 3 MS/s — every control channel on one `role: wideband` device, one
> `channels:` entry per cell — but the margin is thin. On a reporter's Pi
> (#1321), two cells went from 513 dropped chunks per ten minutes to 7 once
> the DDC bank pre-decimated its shared stream, and four cells from 46 to 0
> once the USB drivers sized their stream buffers by time (400 ms); that Pi
> runs the pump's stall bench at 0.51× real time. Read
> `sdr: dropping live IQ chunks` by its runtime fields, and check that the
> build you ran contains the fix. The Volume 2 kitchen sink adds `sdr.iio`,
> AM / ACARS / NXDN scan-list channels, `paging`, `alerts`, `transcription`
> and `encryption_keys` to Volume 1's map — every key from
> `config.example.yaml`.

**Key takeaways**

- **The budget is set by the worst millisecond, not the average.** A pump
  that needs half of real time on average still sheds IQ when one chunk
  takes longer than the buffer in front of it holds.
- **Wideband is how one radio carries several systems.** Two systems on one
  `role: control` tuner are decoded one at a time; `channels:` on a
  `role: wideband` device decodes them in parallel.
- **A field result is only as good as the build it ran on.** Check that the
  release contains the fix before reading "still dropping" as "the fix
  failed".
- **Volume 2 adds sections; it changes no rules.** Paths, units, secrets
  and "the example file wins" are Volume 1's, unchanged.

## Cheat sheet

| Config section | What it owns | Volume 2 part |
|---|---|---|
| `sdr.devices`, `sdr.input_sample_rate` | wideband channels; a native rate above the decode rate | 14 (this part) |
| `sdr.iio`, `sdr.rtl_tcp` | a Pluto without Soapy; a phone's dongle | [13]({{ '/blog/tutorials/cookbook-volume-2-13-pluto-native/' | relative_url }}), [10]({{ '/blog/tutorials/cookbook-volume-2-10-phone-rig/' | relative_url }}) |
| `scanner.conventional` | AM and ACARS, NXDN, data decoders, tone gates, priority | [1]({{ '/blog/tutorials/cookbook-volume-2-01-airband-rig/' | relative_url }}), [3]({{ '/blog/tutorials/cookbook-volume-2-03-nxdn/' | relative_url }}), [4]({{ '/blog/tutorials/cookbook-volume-2-04-fleet-signalling/' | relative_url }}), [9]({{ '/blog/tutorials/cookbook-volume-2-09-hold-avoid-priority/' | relative_url }}) |
| `paging`, `mdc1200`, `fleetsync` | message decoders on a dedicated SDR | [2]({{ '/blog/tutorials/cookbook-volume-2-02-paging-rig/' | relative_url }}), [4]({{ '/blog/tutorials/cookbook-volume-2-04-fleet-signalling/' | relative_url }}) |
| `recordings` | voice profile, loudness, FLAC, muting ciphertext | [5]({{ '/blog/tutorials/cookbook-volume-2-05-tuning-the-sound/' | relative_url }}), [8]({{ '/blog/tutorials/cookbook-volume-2-08-your-own-keys/' | relative_url }}) |
| `alerts`, `transcription` | what happens after a call | [6]({{ '/blog/tutorials/cookbook-volume-2-06-alerts/' | relative_url }}), [7]({{ '/blog/tutorials/cookbook-volume-2-07-transcription/' | relative_url }}) |
| No section at all | the one-shot tools: band survey, meter walk, board probe | [11]({{ '/blog/tutorials/cookbook-volume-2-11-band-survey/' | relative_url }}), [12]({{ '/blog/tutorials/cookbook-volume-2-12-meter-walk/' | relative_url }}), [13]({{ '/blog/tutorials/cookbook-volume-2-13-pluto-native/' | relative_url }}) |

## In this post

- **The Pi 4 budget** — an Airspy Mini, TETRA cells, what the drops said.
- **The map** — the Volume 2 kitchen sink, block by block.
- **The decision table** — what do you want? → which part, both volumes.
- **Config hygiene** — the example file, the source line, the doctor.
- **Where to go from here** — the series wrap, and what to read next.

## The Pi 4 budget

The case is issue
[#1321](https://github.com/MattCheramie/GopherTrunk/issues/1321): an
[Airspy Mini]({{ '/reference/airspy-mini/' | relative_url }}), which cannot
stream below 3 MS/s, on a
[Raspberry Pi 4]({{ '/raspberry-pi-sdr-scanner/' | relative_url }}),
decoding several [TETRA]({{ '/reference/tetra/' | relative_url }}) cells
from one capture. The symptom was "one thread at 90 %, dropping live IQ
chunks". The rig itself is short:

```yaml
sdr:
  sample_rate: 3_000_000              # the Airspy Mini's lowest rate
  devices:
    - serial: "AIRSPY SN:0123456789ABCDEF"   # from `gophertrunk sdr list`
      role: wideband
      gain: "auto"
      center_freq_hz: 467_893_750
      channels:                       # one entry per cell, each its own system
        - frequency_hz: 467_912_500
          system: "Cell-A"
        - frequency_hz: 467_875_000
          system: "Cell-B"

trunking:
  systems:
    - name: "Cell-A"
      protocol: tetra
      control_channels: [467_912_500]
    - name: "Cell-B"
      protocol: tetra
      control_channels: [467_875_000]
```

Each `channels:` entry names its own `protocol: tetra` system, and its
frequency must be in that system's `control_channels`. Both control channels
then decode in parallel, each on its own 144 kHz tap with four same-carrier
voice taps. The shape matters: list the same two systems behind one
`role: control` or `auto` tuner and the daemon warns, at startup, that they
`share ONE control SDR` — the hunter decodes them one at a time and camps
on the first that locks, so on a healthy site the second is never hunted.
[Volume 1 Part 7]({{ '/blog/tutorials/operator-cookbook-07-multi-system-pool/' | relative_url }})
is the pool behind that rule.

Healthy looks like one engine line and a lock per cell; every loss has
written its own line, with a reason, since this issue:

```text
INF widebandt2: starting center_freq_hz=467893750 sample_rate_hz=3000000 channels=2 strategy=auto(ddc)
INF tetra cc locked freq=467912500 mcc=… mnc=… la=… system=Cell-A
INF tetra cc lost freq=467912500 mcc=… mnc=… la=… system=Cell-A reason="no sync burst or signalling decoded" silent_for=… timeout=…
```

**What it cost, and what fixed it.** Three rounds, each measured:

| Round | What was wrong | What changed | Measured |
|---|---|---|---|
| 1 | Below 5 MS/s nothing was pre-decimated, so each 144 kHz tap resampled the full 3 MS/s stream | The [DDC]({{ '/reference/digital-down-converter/' | relative_url }}) bank's shared decimator now sizes itself to the channel plan, and taps run on workers | Reporter's Pi 4, 2 cells: 513 → 7 drops per 10 min (28 Oct) |
| 2 | The Airspy driver's output queue held 8 USB buffers, about 11 ms at 3 MS/s | Every USB driver sizes its stream buffer by time: 400 ms | Reporter's Pi 4, 4 cells: 46 → 0 drops per 10 min (3 Nov) |
| 3 | The TETRA receiver's sync search, AFC and filters were the next costs in the pump | Packed sync compare, cheaper AFC, folded filters | Stall bench, five-cell pump: 0.51× → 0.43× of real time (4 Nov) |

<figure class="lab-figure">
<svg viewBox="0 0 680 216" width="680" height="216" role="img" aria-label="Before and after bars for the three rounds of the Raspberry Pi 4 sizing work. Two TETRA cells: dropped chunks per ten minutes fall from 513 to 7. Four cells: from 46 to 0. Stall bench: pump time divided by stream time falls from 0.51 on the reporter's Pi 4 to 0.43 for the five-cell pump, against a dashed outline marking 1.0, real time. Each row is drawn to its own scale.">
  <text x="8" y="14" fill="var(--fg-muted)" font-size="9">Airspy Mini at 3 MS/s, TETRA cells on one wideband device — each row drawn to its own scale</text>
  <text x="8" y="40" fill="currentColor" font-size="10">2 cells</text>
  <text x="8" y="54" fill="var(--fg-muted)" font-size="9">dropped chunks / 10 min</text>
  <rect x="150" y="28" width="400" height="13" fill="none" stroke="var(--fg-muted)"/>
  <text x="556" y="39" fill="var(--fg-muted)" font-size="9">513 before</text>
  <rect x="150" y="47" width="6" height="13" fill="none" stroke="var(--accent)"/>
  <text x="162" y="58" fill="var(--accent)" font-size="9">7 — the DDC bank pre-decimates its shared stream (28 Oct)</text>
  <text x="8" y="98" fill="currentColor" font-size="10">4 cells</text>
  <text x="8" y="112" fill="var(--fg-muted)" font-size="9">dropped chunks / 10 min</text>
  <rect x="150" y="86" width="400" height="13" fill="none" stroke="var(--fg-muted)"/>
  <text x="556" y="97" fill="var(--fg-muted)" font-size="9">46 before</text>
  <line x1="150" y1="105" x2="150" y2="118" stroke="var(--accent)" stroke-width="2"/>
  <text x="162" y="116" fill="var(--accent)" font-size="9">0 — stream buffers sized by time, 400 ms (3 Nov)</text>
  <text x="8" y="158" fill="currentColor" font-size="10">stall bench</text>
  <text x="8" y="172" fill="var(--fg-muted)" font-size="9">pump time ÷ stream time</text>
  <rect x="150" y="142" width="400" height="38" fill="none" stroke="var(--fg-muted)" stroke-dasharray="4 3"/>
  <text x="556" y="165" fill="var(--fg-muted)" font-size="9">1.0 = real time</text>
  <rect x="150" y="146" width="204" height="13" fill="none" stroke="var(--fg-muted)"/>
  <text x="360" y="157" fill="var(--fg-muted)" font-size="9">0.51 — the reporter's Pi 4</text>
  <rect x="150" y="163" width="172" height="13" fill="none" stroke="var(--accent)"/>
  <text x="328" y="174" fill="var(--accent)" font-size="9">0.43 — five-cell pump, 4 Nov round</text>
  <text x="8" y="204" fill="var(--fg-muted)" font-size="9">Drops went to zero; the bench bar is the reason to stay cautious — half of every second is already spent.</text>
</svg>
<figcaption>Two fixes took the drops to zero on the reporter's Pi 4, but the stall bench shows how little is left over: the pump needs about half of real time before any spike.</figcaption>
</figure>

**Reading the margin.** 0.51× means the pump spends half of every second
working, on average — and averages are not what drop IQ. On a desktop CPU
the same bench showed a typical chunk taking a quarter of its own duration
and the slowest one in a hundred taking three times it, which is why an
11 ms queue shed IQ with the cores mostly idle and a 400 ms one does not.
So treat a Pi 4 as full at the cell count you have verified, and spend
nothing you do not need: every symbol view the web console opens (the
[Tuning panel]({{ '/tuning.html' | relative_url }}) and the scopes, and the
signal-quality readout on the Dashboard and Scanner panels) builds its own
down-converter from the full input rate plus a receiver. Several open tabs
are several DSP chains.

**Reading the WARN.** When IQ is shed, the daemon says what the process was
doing at the time:

```text
WRN sdr: dropping live IQ chunks; consumer can't keep up driver=airspy serial=… dropped_since_last=… window_s=… cpu_busy_pct=… gc_cpu_pct=… gc_cycles=… gc_pause_max_ms=… sched_wait_p99_ms=… sched_wait_max_ms=…
```

`cpu_busy_pct` near 100 or a large `sched_wait_p99_ms` means the process
was short of CPU; a large `gc_cpu_pct` or `gc_pause_max_ms` means the
garbage collector stalled the stream; neither means the decode work itself
is too slow. Before acting on any of it, check the build: a "still dropping
on v1.3.6" report turned out to test a release cut two days before the
buffer fix merged.

```bash
git merge-base --is-ancestor <fix-commit> <release-tag> && echo "tag contains the fix"
```

**The other rate knob.** With cells as close together as in the config
above, the engine also warns that the
`capture is oversampled for the channel plan`. On an Airspy Mini that is
expected — the radio cannot go slower, and the DDC strategy already
decimates the shared stream
([DDC versus polyphase]({{ '/blog/deep-dives/sdr-internals-05-tuning-channelization/' | relative_url }})).
`sdr.input_sample_rate` is the systemwide version, for recordings and the
other roles: the hardware runs at the native rate and everything downstream
sees `sample_rate` (logged as `sdr: input pre-decimation enabled`). It is a
load and file-size lever, not an RF fix:

```yaml
sdr:
  input_sample_rate: 3_000_000   # what the hardware runs at
  sample_rate: 1_000_000         # an exact divisor: decimate by 3
```

**Status, plainly.** The drop counts are the reporter's own, on their Pi 4.
The bench figures are synthetic. What the project's notes do not record as
confirmed on air is voice recorded through the wideband TETRA path, so
treat that as this recipe's open gate.

### How this recipe shapes operator practice

- **Verify a cell count, then stop.** Headroom on a Pi 4 is measured, not
  assumed; the [overruns lines]({{ '/blog/tutorials/field-notebook-07-overruns-host-drops/' | relative_url }})
  say when you have crossed it.
- **Name the build in every report.** It settles more than a log excerpt.
- **Put parallel work on the wideband path.** Anything behind one control
  tuner takes turns.

## The map

Here is a rig that has absorbed both volumes, abridged to the keys Volume 2
added. Every key is in `config.example.yaml`, and this exact block loads
and validates; every `← Part` note is a link target in this series:

```yaml
storage:
  path: "../data/calls.db"          # call log + ACARS / pager / MDC logs  ← Parts 1, 2, 4

recordings:
  dir: "../recordings"
  format: flac                      # lossless container            ← Part 5
  voice_profile: op25               # "" / mbelib (default) | op25  ← Part 5
  mute_encrypted: true              # silence, not vocoded ciphertext ← Part 8
  normalize:
    enabled: true                   # EBU R128 per recording        ← Part 5

sdr:
  sample_rate: 2_400_000
  # input_sample_rate: 3_000_000    # native rate above sample_rate ← Part 14
  devices:
    - serial: "00000001"            # P25 CC + in-window voice      ← Vol. 1 Part 1
      role: wideband
      gain: "auto"
      center_freq_hz: 858_000_000
      voice_taps: 2
      channels:                     # one entry per CC, in parallel ← Part 14
        - frequency_hz: 857_262_500
          system: "Metro-P25"
    - serial: "00000002"            # trunking voice spill-over     ← Vol. 1 Part 7
      role: voice
      gain: "auto"
    - serial: "00000003"            # the scan list's radio (last voice device)
      role: voice
      gain: "auto"
    - serial: "00000004"            # pinned by paging below        ← Part 2
      role: dedicated
  rtl_tcp: []                       # a phone's dongle behind its driver app ← Part 10
  iio:                              # Pluto / PlutoSky, no Soapy    ← Part 13
    - addr: "192.168.2.1"
      serial: "plutosky"
      role: control                 # hunts the NXDN control channel
      gain: "400"                   # tenths of a dB; "auto" = AGC

trunking:
  systems:
    - name: "Metro-P25"
      protocol: p25
      control_channels: [857_262_500]
      encryption_keys:              # only keys you hold            ← Part 8
        - key_id: 1
          algorithm: adp            # rc4 | des | tdes | aes by key length
          key: "0000000000"         # placeholder, never a real key in a post
    - name: "County-NXDN"
      protocol: nxdn                #                               ← Part 3
      control_channels: [453_775_000]
      nxdn_band_plan:               # required to follow NXDN voice
        linear:
          base_hz: 453_150_000
          spacing_hz: 12_500
          offset: 1

scanner:
  scan_mode: all                    # all | list; hold / avoid are runtime ← Part 9
  priority_interleave: 4            # priority check every 4 picks  ← Part 9
  conventional:
    - label: "Tower"
      frequency_hz: 118_700_000
      mode: am                      # carrier-to-noise squelch      ← Part 1
      squelch_cn_db: 12
      priority: 5
    - label: "ACARS primary"
      frequency_hz: 131_550_000
      mode: am
      decoders: [acars]             #                               ← Part 1
    - label: "Fire Dispatch"
      frequency_hz: 154_190_000
      mode: nfm
      squelch_dbfs: -50             # ANDed with the noise squelch
      audio_gain_db: 6              # balance against digital       ← Part 5
      decoders: [mdc1200, fleetsync]  # who keyed up                ← Part 4
      tone:
        mode: dcs
        dcs_code: "023"
        dcs_polarity: normal        # normal | inverted | both      ← Part 4
    - label: "Plant ops"
      frequency_hz: 447_100_000
      mode: nxdn                    # nxdn (NXDN96) | nxdn48        ← Part 3
      squelch_dbfs: -50
      encryption_keys:
        - key_id: 3
          algorithm: scrambler      # decimal 15-bit key            ← Part 8
          key: "12345"              # placeholder

paging:                             #                               ← Part 2
  wideband:
    - serial: "00000004"
      channels:
        - protocol: flex
          frequency_hz: 929_612_500
        - protocol: pocsag
          frequency_hz: 929_662_500
          baud_hz: 1200

alerts:                             # rules x channels              ← Part 6
  channels:
    - name: "phone"
      type: ntfy
      url: "https://ntfy.sh/my-scanner-topic"
  rules:
    - name: "Any emergency"
      on: [call.start, grant]
      emergency: true
      channels: [phone]

transcription:                      #                               ← Part 7
  enabled: true
  url: "http://192.168.1.20:8080/inference"   # whisper.cpp server on a bigger box
  upload_format: wav16k

audio:
  enabled: false                    # no sound hardware in Termux   ← Part 10
  sample_rate: 8000                 # live stream rate              ← Part 5
```

Three observations at this altitude. **Volume 2 grew the scan list and the
aftermath.** Most new keys hang off `scanner.conventional` entries or act
on a finished call (`recordings`, `alerts`, `transcription`); the trunking
half barely moved. **Three recipes have no block.** The band survey, the
meter walk and the board probe are commands that read no config file — an
`sdr.iio` entry does nothing for `gophertrunk power`. **Encryption keys are
the one block you should not keep here**: a system's `encryption_key_files`
loads them from an encrypted key container instead, with the password in an
environment variable.

## The decision table

| What do you want? | Read | The load-bearing keys |
|---|---|---|
| Airband voice and ACARS | [Part 1]({{ '/blog/tutorials/cookbook-volume-2-01-airband-rig/' | relative_url }}) | `mode: am`, `squelch_cn_db`, `decoders: [acars]` |
| Pagers: POCSAG and FLEX | [Part 2]({{ '/blog/tutorials/cookbook-volume-2-02-paging-rig/' | relative_url }}) | `paging.wideband`, `role: dedicated` |
| NXDN, trunked or conventional | [Part 3]({{ '/blog/tutorials/cookbook-volume-2-03-nxdn/' | relative_url }}) | `protocol: nxdn`, `nxdn_band_plan`, `mode: nxdn` |
| Who keyed up on an analog channel | [Part 4]({{ '/blog/tutorials/cookbook-volume-2-04-fleet-signalling/' | relative_url }}) | `decoders: [mdc1200, fleetsync]`, `tone.dcs_polarity` |
| Recordings that sound right | [Part 5]({{ '/blog/tutorials/cookbook-volume-2-05-tuning-the-sound/' | relative_url }}) | `recordings.voice_profile`, `audio_gain_db` |
| A push when it matters | [Part 6]({{ '/blog/tutorials/cookbook-volume-2-06-alerts/' | relative_url }}) | `alerts.channels`, `alerts.rules` |
| Text for every call | [Part 7]({{ '/blog/tutorials/cookbook-volume-2-07-transcription/' | relative_url }}) | `transcription.url`, `upload_format` |
| Decrypt with keys you hold | [Part 8]({{ '/blog/tutorials/cookbook-volume-2-08-your-own-keys/' | relative_url }}) | `encryption_keys`, `recordings.mute_encrypted` |
| Drive it like a scanner | [Part 9]({{ '/blog/tutorials/cookbook-volume-2-09-hold-avoid-priority/' | relative_url }}) | `scanner.priority_interleave`; hold and avoid at runtime |
| A rig in your pocket | [Part 10]({{ '/blog/tutorials/cookbook-volume-2-10-phone-rig/' | relative_url }}) | the Termux build, `sdr.rtl_tcp` |
| What is on this band? | [Part 11]({{ '/blog/tutorials/cookbook-volume-2-11-band-survey/' | relative_url }}) | `gophertrunk power -f` |
| Utility meters at 868 MHz | [Part 12]({{ '/blog/tutorials/cookbook-volume-2-12-meter-walk/' | relative_url }}) | `gophertrunk wmbus` |
| A Pluto without SoapySDR | [Part 13]({{ '/blog/tutorials/cookbook-volume-2-13-pluto-native/' | relative_url }}) | `sdr.iio`, `gophertrunk iio probe` |
| Several cells on a Pi 4 | this part | `role: wideband`, `channels` |
| A first trunked system: P25, DMR, TETRA | Volume 1 [Parts 1–5]({{ '/blog/tutorials/operator-cookbook-01-forty-dollar-p25-rig/' | relative_url }}) | `protocol`, `control_channels`, band plans |
| Analog FM and fire tone-out | Volume 1 [Part 6]({{ '/blog/tutorials/operator-cookbook-06-analog-fm-tone-out/' | relative_url }}) | `scanner.conventional`, `tone_out.profiles` |
| Many systems, or radios far away | Volume 1 [Parts 7–8]({{ '/blog/tutorials/operator-cookbook-07-multi-system-pool/' | relative_url }}) | device roles, `rtl_tcp`, `soapy_remote` |
| Share it, archive it, run it 24/7 | Volume 1 [Parts 9–11]({{ '/blog/tutorials/operator-cookbook-09-sharing-the-feed/' | relative_url }}) | `broadcast`, `retention`, `api.auth` |
| A second antenna; names for numbers | Volume 1 [Parts 12–13]({{ '/blog/tutorials/operator-cookbook-12-diversity-mrc/' | relative_url }}) | `diversity: mrc`, `talkgroup_file` |

## Config hygiene

Volume 1's rules stand — paths relative to the config file, gain in tenths
of a dB, secrets out of the file, one knob at a time. Volume 2 adds four:

- **A misspelled key is silent.** The loader does not reject a key it does
  not know; the daemon just runs the default. Copy key names from
  `config.example.yaml`, never from memory. The project's own rule is that
  every key lands there with its code — two keys this volume uses,
  `nxdn_band_plan` and `sdr.input_sample_rate`, were once missing from it,
  and were added for exactly that reason.
- **Read the source line.** Startup logs `config: source` with the path it
  loaded, warns `config: no config file found — running built-in defaults`
  when it found none, and warns when several were found and all but one
  were ignored. A rig that "lost its settings" after an update usually
  loaded a different file.
- **Ask the doctor about the dongle, not the config.**
  `gophertrunk sdr doctor` reports which OS driver is bound to each RTL-SDR
  and HackRF it finds, read-only, alongside a running daemon;
  `gophertrunk sdr list --probe` fills in tuner and gain ladder.
  [Running It For Real Part 7]({{ '/blog/deep-dives/running-it-for-real-07-sdr-doctor-preflight/' | relative_url }})
  covers both.
- **Know which tools never read the file.** `power`, `capture`, `wmbus` and
  `iio probe` take flags only.

## Where to go from here

Two volumes, twenty-eight parts, one rule that outlived all of them: say
what was verified, and on what. The series pages are the index —
[Volume 2]({{ '/blog/series/operator-cookbook-volume-2/' | relative_url }})
and [Volume 1]({{ '/blog/series/operator-cookbook/' | relative_url }}) — and
three sibling series pick up where the recipes stop:

- **Why the scan list opens when it does** —
  [Squelch]({{ '/blog/series/squelch/' | relative_url }}) is the deep dive
  behind every `scanner.conventional` key in the map above.
- **Why several of these decoders exist at all** —
  [Rebuilt From the Air]({{ '/blog/series/rebuilt-from-the-air/' | relative_url }})
  is the postmortem series on the decoders rebuilt against real radios.
- **What the log is telling you** —
  [The Field Notebook]({{ '/blog/series/field-notebook/' | relative_url }})
  reads the lines this volume kept quoting.

## FAQ

**Can a Raspberry Pi 4 decode several TETRA cells at once?**
Yes, with little to spare. A reporter's Pi 4 with an Airspy Mini at 3 MS/s
went to zero dropped chunks with four cells on one wideband device once
the DDC and stream-buffer fixes landed, but the same Pi runs the pump's
stall bench at about half of real time. Verify your own cell count.

**Why does GopherTrunk drop IQ chunks when the CPU is mostly idle?**
Because drops come from the slowest moment, not the average: a driver
queue holding a few milliseconds of IQ overflows on any longer pause.
Every USB driver now sizes its stream buffer by time, 400 ms; if drops
persist, read the runtime fields on the WARN line.

**Why does only one of my two systems decode on one SDR?**
A `role: control` or `auto` tuner hunts systems one at a time and camps on
the first that locks. To decode them together, host every control channel
on a `role: wideband` device with one `channels:` entry per system; the
daemon warns about the first shape at startup.

**How do I know a fix is in the build I am running?**
Compare commits, not dates:
`git merge-base --is-ancestor <fix-commit> <release-tag>` succeeds only
when the release contains the fix.

**Is there one config file that enables every Volume 2 recipe?**
The map above loads and validates, but it is a merge guide, not a rig:
four dongles and a Pluto on one box is a budget nobody has measured. Start
from the part that matches your goal and add one block at a time.

## Series navigation

**Part 14 of 14** · ←
[Part 13: Pluto Without Soapy — The Native IIO Driver]({{ '/blog/tutorials/cookbook-volume-2-13-pluto-native/' | relative_url }})
· [Back to the series index]({{ '/blog/series/operator-cookbook-volume-2/' | relative_url }})
