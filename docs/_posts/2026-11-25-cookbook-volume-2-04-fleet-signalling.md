---
title: "The Operator's Cookbook Vol. 2, Part 4: Fleet Signalling — MDC1200, FleetSync and Tone Gates"
description: "A complete GopherTrunk recipe for knowing who keyed up on an analog channel: MDC1200 and FleetSync ID bursts decoded on a dedicated SDR or, usually better, with decoders on a scan-list channel, plus CTCSS and DCS tone gates set correctly, with real log lines, REST rows and each piece's on-air status."
category: tutorials
keywords: mdc1200 decoder sdr, fleetsync decoder rtl-sdr, ptt id unit id analog radio, kenwood fleetsync ani decode, motorola mdc1200 scanner, ctcss dcs squelch sdr, dcs polarity normal inverted, analog radio unit id logging, gophertrunk fleet signalling
tags: [operator-cookbook, cookbook-volume-2, mdc1200, fleetsync, ctcss, dcs]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "The Operator's Cookbook, Volume 2"
series_part: 4
---

*Part 4 of **The Operator's Cookbook, Volume 2**, a 14-part second run of
complete, copy-paste GopherTrunk builds — one working rig per part, each with
its verification status stated plainly.
[Part 3]({{ '/blog/tutorials/cookbook-volume-2-03-nxdn/' | relative_url }})
followed an NXDN system, where every transmission names its radio digitally.
This part goes back to analog FM and answers the same question — who keyed
up? — from the short data burst a Motorola or Kenwood radio sends with its
push-to-talk. Along the way it sets the scan list's CTCSS and DCS gates
the way they now have to be set: exactly.*

> **TL;DR:** On an analog channel the caller ID is an in-band burst:
> Motorola [MDC1200]({{ '/reference/mdc1200/' | relative_url }}) or Kenwood
> FleetSync. Decode it either by pinning a dongle (`mdc1200.channels`,
> `fleetsync.channels`) or — usually better — by adding
> `decoders: [mdc1200, fleetsync]` to a `scanner.conventional` channel you
> already scan. Bursts land in `GET /api/v1/mdc1200/messages`,
> `GET /api/v1/fleetsync/messages` and their panels, and need `storage.path`.
> The decoders ignore the tone gate by design. Tone gates are exact:
> `ctcss_hz` separates tones 2.3 Hz apart, and a DCS gate takes `dcs_code`
> plus `dcs_polarity: normal`, `inverted` or `both`. Status: MDC1200 is on-air
> verified through the scan-list tap (28 Sep), FleetSync on a dedicated
> dongle (16 Sep), the tone gates on 24 Sep.

**Key takeaways**

- **The ID burst is analog radio's caller ID.** A unit number at the start
  or end of a transmission turns "someone on the yard channel" into "unit
  1772", and it costs nothing but a config line to log.
- **Tap the channel you already scan; pin a dongle when a burst must not be
  missed.** The scan-list tap needs no extra hardware but hears only what is
  sent while the scanner is on that channel.
- **Decoders ignore the tone gate, on purpose.** A burst carries its own sync
  word and check, so it is logged even when the squelch stays shut.
- **A tone gate is exact, and it has a price.** One CTCSS channel raises the
  scan window of every channel on that dongle's list to 350 ms; one DCS
  channel raises it to 600 ms.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| Scan-list tap | decodes bursts on a channel you already scan | `decoders: [mdc1200, fleetsync]` |
| Dedicated receiver | one dongle that never leaves the channel | `mdc1200.channels[]`, `fleetsync.channels[]` |
| MDC1200 log | unit ID, operation, CRC, channel | `GET /api/v1/mdc1200/messages`, `/mdc1200` |
| FleetSync log | fleet, unit, FleetSync I or II, channel | `GET /api/v1/fleetsync/messages`, `/fleetsync` |
| CTCSS gate | carrier and one exact tone | `tone.mode: ctcss`, `ctcss_hz` |
| DCS gate | carrier and a code in the chosen sense | `tone.mode: dcs`, `dcs_code`, `dcs_polarity` |
| Declutter the console | hides panels this rig never fills | `web.tabs` |
| Why a channel opened | level, quieting, code and polarity | log `conv: squelch opened`, `conv: DCS gate opened` |

## In this post

- **What you're building** — one channel, two consumers.
- **The config** — the scan-list recipe, then the dedicated-dongle form.
- **First run — what healthy looks like** — a gated call and two real rows.
- **When it doesn't work** — symptom → cause → fix.
- **Variations** — per-dongle lists, shared frequencies, offline checks.

## What you're building

The scan list of
[Volume 1, Part 6]({{ '/blog/tutorials/operator-cookbook-06-analog-fm-tone-out/' | relative_url }})
with two things added to its channels.

The first is a **data tap**. Many analog fleets send a short
[FFSK]({{ '/reference/ffsk/' | relative_url }}) burst — two audio tones at
1200 baud — when a radio keys up or unkeys. Motorola's version is MDC1200,
which carries a unit ID and an operation (PTT ID, emergency, status, radio
check); Kenwood's is FleetSync, which carries a fleet number and a unit ID.
[Beyond Voice, Part 3]({{ '/blog/deep-dives/beyond-voice-03-mdc1200-template-decoder/' | relative_url }})
and
[Part 4]({{ '/blog/deep-dives/beyond-voice-04-fleetsync-wav-that-lied/' | relative_url }})
describe the two decoders, and
[Rebuilt From the Air, Part 2]({{ '/blog/solution-postmortem/rebuilt-from-the-air-02-mdc1200-tones-as-data/' | relative_url }})
explains why MDC1200 only started decoding real radios in late September.
The tap runs those decoders on a scan-list channel's own IQ, and
[Squelch, Part 11]({{ '/blog/deep-dives/squelch-11-data-on-the-scan-list/' | relative_url }})
is the deep dive on how.

The second is a **tone gate** done properly. A
[CTCSS]({{ '/reference/ctcss/' | relative_url }}) tone or
[DCS]({{ '/reference/dcs/' | relative_url }}) code under the voice lets a
shared frequency carry several users; the gate opens the squelch only for
yours. Both detectors were wrong in ways their own
[round-trip tests]({{ '/blog/solution-postmortem/from-the-issue-tracker-20-self-consistent-trap/' | relative_url }})
could not show, and both were corrected against real radios in September —
[Rebuilt From the Air, Part 3]({{ '/blog/solution-postmortem/rebuilt-from-the-air-03-ctcss-dcs-tone-gates/' | relative_url }})
is the investigation;
[Squelch, Part 5]({{ '/blog/deep-dives/squelch-05-ctcss/' | relative_url }})
and
[Part 6]({{ '/blog/deep-dives/squelch-06-dcs/' | relative_url }}) are the
designs.

<figure class="lab-figure">
<svg viewBox="0 0 680 230" width="680" height="230" role="img" aria-label="Block diagram of one scan-list channel with two consumers: a voice-role SDR feeds a channel front end that reduces the channel to about 48 kHz behind a plus or minus 8 kHz filter; the voice branch passes the level squelch and noise squelch, then the CTCSS or DCS tone gate, then FM demodulation to a recording; the data branch feeds the MDC1200 and FleetSync decoders from the level alone, bypassing the tone gate, and writes unit IDs to the two log tables, the REST routes and the web panels">
  <text x="340" y="18" text-anchor="middle" fill="var(--fg-muted)" font-size="10">one scan-list channel, two consumers</text>
  <rect x="8" y="96" width="88" height="38" rx="4" fill="none" stroke="currentColor"/>
  <text x="52" y="112" text-anchor="middle" fill="currentColor" font-size="10">SDR</text>
  <text x="52" y="125" text-anchor="middle" fill="var(--fg-muted)" font-size="9">role: voice</text>
  <line x1="96" y1="115" x2="112" y2="115" stroke="currentColor"/>
  <rect x="112" y="96" width="134" height="38" rx="4" fill="none" stroke="currentColor"/>
  <text x="179" y="112" text-anchor="middle" fill="currentColor" font-size="10">channel front end</text>
  <text x="179" y="125" text-anchor="middle" fill="var(--fg-muted)" font-size="9">~48 kHz, ±8 kHz filter</text>
  <line x1="246" y1="115" x2="270" y2="59" stroke="currentColor"/>
  <line x1="246" y1="115" x2="270" y2="177" stroke="var(--accent)"/>
  <rect x="270" y="40" width="170" height="38" rx="4" fill="none" stroke="currentColor"/>
  <text x="355" y="56" text-anchor="middle" fill="currentColor" font-size="10">level + noise squelch</text>
  <text x="355" y="69" text-anchor="middle" fill="var(--fg-muted)" font-size="9">squelch_dbfs, a quieted carrier</text>
  <line x1="440" y1="59" x2="456" y2="59" stroke="currentColor"/>
  <rect x="456" y="40" width="104" height="38" rx="4" fill="none" stroke="currentColor"/>
  <text x="508" y="56" text-anchor="middle" fill="currentColor" font-size="10">tone gate</text>
  <text x="508" y="69" text-anchor="middle" fill="var(--fg-muted)" font-size="9">CTCSS or DCS</text>
  <line x1="560" y1="59" x2="576" y2="59" stroke="currentColor"/>
  <rect x="576" y="40" width="96" height="38" rx="4" fill="none" stroke="currentColor"/>
  <text x="624" y="56" text-anchor="middle" fill="currentColor" font-size="9">FM voice → WAV</text>
  <text x="624" y="69" text-anchor="middle" fill="var(--fg-muted)" font-size="9">History row</text>
  <text x="470" y="120" text-anchor="middle" fill="var(--accent)" font-size="9">the data branch needs the level only — it never waits for the tone gate</text>
  <rect x="270" y="158" width="200" height="38" rx="4" fill="none" stroke="var(--accent)"/>
  <text x="370" y="174" text-anchor="middle" fill="var(--accent)" font-size="10">decoders: [mdc1200, fleetsync]</text>
  <text x="370" y="187" text-anchor="middle" fill="var(--fg-muted)" font-size="9">own sync word, own check</text>
  <line x1="470" y1="177" x2="486" y2="177" stroke="var(--accent)"/>
  <rect x="486" y="158" width="186" height="38" rx="4" fill="none" stroke="currentColor"/>
  <text x="579" y="174" text-anchor="middle" fill="currentColor" font-size="10">mdc1200_log · fleetsync_log</text>
  <text x="579" y="187" text-anchor="middle" fill="var(--fg-muted)" font-size="9">REST routes · web panels</text>
  <text x="340" y="220" text-anchor="middle" fill="var(--fg-muted)" font-size="9">a burst sent while the scanner is on another channel reaches neither branch</text>
</svg>
<figcaption>The voice branch opens only for carrier and tone; the data branch runs from the carrier alone, so a unit ID is logged even when the tone gate keeps the recording shut.</figcaption>
</figure>

## The shopping list

| Item | Price (rough) | Notes |
|---|---|---|
| RTL-SDR Blog V3/V4 | ~$35 | one dongle for the scan-list recipe; a second if you pin a decoder |
| Antenna for the band | $0–$30 | VHF high band or UHF, wherever the fleet is licensed |
| The channel's tone or code | $0 | from the licence, the radio's programming, or a listening session |

## The config

```yaml
log:
  level: info

storage:
  path: "../data/calls.db"         # mdc1200_log and fleetsync_log live here

recordings:
  dir: "../recordings"

sdr:
  sample_rate: 2_400_000
  devices:
    - serial: "00000001"           # from `gophertrunk sdr list`
      role: voice                  # the scan list drives a voice SDR
      gain: "auto"

scanner:
  conventional:
    - label: "Public Works"
      frequency_hz: 154_000_000
      mode: nfm
      squelch_dbfs: -50
      hangtime_ms: 1500
      talkgroup_id: 2147500301
      decoders: [mdc1200]
      tone:
        mode: ctcss
        ctcss_hz: 162.2            # the exact tone, to the tenth
    - label: "Yard"
      frequency_hz: 462_562_500
      mode: nfm
      squelch_dbfs: -50
      talkgroup_id: 2147500302
      decoders: [mdc1200, fleetsync]
      tone:
        mode: dcs
        dcs_code: "025"            # three octal digits, quoted
        dcs_polarity: normal       # normal (D025N) | inverted (D025I) | both

web:
  tabs:                            # optional: hide panels this rig never fills
    pagers: false
    aprs: false
    ais: false
    dsc: false
    adsb: false
    acars: false
```

Frequencies, tone and code are placeholders. What the keys do:

**`decoders`.** `mdc1200` and `fleetsync` belong on `fm` and `nfm` channels;
a digital mode rejects them. The scanner reduces its IQ to one channel at
about 48 kHz behind a ±8 kHz filter before a decoder sees it, so the result
does not depend on the dongle's rate. The decoders run on every chunk that
clears `squelch_dbfs` and through the whole dwell. One that has locked a
sync word holds the scan window for up to another 500 ms and counts as
activity, so the scanner neither hops away nor ends the call mid-burst.

**`tone.mode: ctcss`.** Enter the tone your radios are programmed with, to
the tenth of a hertz. Standard tones sit 2.3 to 3.0 Hz apart and the gate now
tells neighbours apart — it once opened on 159.8 Hz when asked for 162.2.
The exception is 150.0 and 151.4 Hz, which are 1.4 Hz apart and open
together. The gate accepts a tone down to 100 Hz of deviation, well under
the roughly 350 Hz a narrowband radio sends.

**`tone.mode: dcs`.** `dcs_code` is the three-digit octal code. Quote it.
`dcs_polarity` is the sense that opens the gate: `normal` matches a radio
programmed D025N, `inverted` matches D025I, `both` accepts either. Leave it
out and you get `normal`.

**The cost of a tone.** A tone detector needs time to report, so the scanner
widens its scan window when a list contains one: from 100 ms per channel to
350 ms with a CTCSS channel, 600 ms with a DCS channel — for every channel on
that dongle's list, toned or not.

**`web.tabs`.** Every tab shows by default; `false` removes it from the
navigation. The keys are the panel paths: `mdc1200`, `fleetsync`, `pagers`
and so on.

The other way to run the decoders pins a dongle to one channel for good.
Beside the same `log:` and `storage:` blocks:

```yaml
sdr:
  sample_rate: 2_400_000
  devices:
    - serial: "00000002"
      role: dedicated

mdc1200:
  channels:
    - serial: "00000002"
      frequency_hz: 154_000_000
      drop_bad_crc: false          # true hides CRC-failed bursts
```

`fleetsync.channels` has the same shape plus `baud_hz` (omit it for 1200;
2400 is accepted). At start-up each pinned receiver logs a line beginning
`mdc1200: tuning the SDR off-channel` (or `fleetsync:`): at 2.4 MS/s the
dongle sits 518 kHz below the channel, clear of its own DC spur. A pinned
receiver misses nothing on its channel and records no voice. Prefer the tap
unless the IDs matter more than the dongle.

## First run — what healthy looks like

```sh
gophertrunk run -config config.yaml
```

The lines below come from replays of the project's committed real-air
slices — each interpolated to 2.4 MS/s and mounted as a `baseband.replay`
tuner — so `device=` names the replay source. A transmission carrying a
100.0 Hz tone, on an entry gated with `ctcss_hz: 100.0`:

```text
INF conv: squelch opened freq_hz=447100000 label="Public Works" mode=nfm level_dbfs=-11.8 squelch_dbfs=-50 quieting_db=46
INF synthetic call started device=replay-ctcss grant="scanner/fm-conv tg=2147500301 src=0 freq=447100000 "
INF recorder: call started device=replay-ctcss wav=…/recordings/scanner/2147500301/20261121_193112_2147500301.wav tg=2147500301 provoice=false vocoder=""
```

`quieting_db` is the noise squelch's reading; 46 dB is a full-quieting
carrier. In that run untoned channels opened 20 ms after start-up and this
one 230 ms later: the tone block at work. The untoned twin of that slice, on
the same gate, sat at −11.9 dBFS with the same quieting and never opened. A
DCS channel adds one line as it opens, naming the code and the sense it was
heard in — the on-air run with a radio set to D025N read:

```text
INF conv: DCS gate opened freq_hz=… label=… dcs_code=025 nrz_inverted=false
```

The decoders print nothing per burst; the tables are the evidence.
`curl 'http://127.0.0.1:8080/api/v1/mdc1200/messages?limit=1'`, after the
real-air MDC1200 slice ran through a `decoders: [mdc1200]` entry:

```json
[
  {
    "id": 19,
    "received_at": "2026-11-21T19:31:25.392365572Z",
    "op": 1,
    "arg": 0,
    "unit_id": 6007,
    "operation": "PTT ID (end)",
    "body": "Unit 1777: PTT ID (end)",
    "raw_hex": "01001777e53c",
    "crc_ok": true,
    "serial": "replay-mdc",
    "frequency_hz": 447100000
  }
]
```

`unit_id` is decimal; `body` shows the same ID in hex. The FleetSync route,
after the FleetSync II slice ran through the `Yard` entry:

```json
[
  {
    "id": 21,
    "received_at": "2026-11-21T19:31:25.580741004Z",
    "fleet": 107,
    "unit": 1772,
    "fs2": true,
    "body": "FleetSync II ANI: fleet=107 unit=1772",
    "raw_hex": "FE80083053059B6C",
    "crc_ok": true,
    "serial": "replay-fs",
    "frequency_hz": 462562500
  }
]
```

`serial` and `frequency_hz` are how a burst from a hopping scanner stays
attributable; the panels show them as the Channel column. One more replay
makes the bypass concrete: with the `Yard` entry gated on a tone its signal
did not carry, the channel never opened — and FleetSync rows kept arriving.

**Status, in one sentence each.** MDC1200 is on-air verified
([#1220](https://github.com/MattCheramie/GopherTrunk/issues/1220)): on
28 Sep a reporter's lab decoded eleven PTT ID bursts, start and end, unit
0x1777, on narrow and wide FM — through a scan-list decoder. FleetSync is
on-air verified
([#1184](https://github.com/MattCheramie/GopherTrunk/issues/1184)): on 16 Sep
FleetSync I and II both decoded live, on a dedicated `fleetsync.channels`
dongle; its scan-list tap is pinned on real-air slices and has no live
report. The tone gates are on-air verified: on 24 Sep exact CTCSS, the DCS
codeword and its polarity were confirmed against Kenwood radios and a
service monitor.

## When it doesn't work

| Symptom | Likely cause | Fix |
|---|---|---|
| Both panels empty, both routes answer 503 | no database | Set `storage.path` |
| IDs appear for some transmissions and not others | the scanner was on another channel when the burst was sent | Expected from a tap. Pin a dongle with `mdc1200.channels` or `fleetsync.channels`, or give the channel its own SDR with `device:` |
| IDs are logged for traffic the tone gate kept shut | by design | Nothing to fix; a burst carries its own sync word and check |
| A toned channel never opens on a strong carrier | the tone in the config is not the tone on air | Enter the exact tone in `ctcss_hz`. No `conv: squelch opened` line while the channel's level clears its squelch means the tone, not the signal |
| A CTCSS gate opens on its neighbour | the two tones are 150.0 and 151.4 Hz | Inherent: they cannot be separated |
| A WARN says the DCS code was heard with the opposite polarity | the radio is inverted and the gate is normal, or the reverse | The line is `conv: DCS code heard with the opposite polarity; gate kept shut`, once per channel. Set `dcs_polarity` to match, or to `both` |
| A DCS gate opens on a different code | an alias: the same bits on air | 023 normal and 047 inverted are one signal. Not a bug, and not fixable |
| The scan lap slowed after adding a tone | the scan window grew for the whole list | Move toned channels to their own dongle with `device:` |
| A pinned decoder logs a failed tune at start-up | a USB control stall | It retries with a back-off and logs `SetCenterFreq recovered`; nothing to do |
| A call on a silent channel ends with a stalled-stream WARN | the dongle stopped delivering IQ mid-dwell | The line is `conv: IQ stream stalled during dwell — ending call and re-opening the stream`. Check the USB port and hub |

The last two rows come from one field run: a FleetSync dongle and a scanner
dongle stalled at the same instant at start-up. Both failures now recover on
their own. Why two dongles stalled together was never root-caused; a shared
hub is one suspect.

### How this recipe shapes operator practice

- **Log IDs everywhere, gate recordings where it matters.** The tap is free
  and tone-blind; the gate decides only what is recorded.
- **Copy tones and codes, don't approximate them.** The detector used to
  round a tone to the nearest 5 Hz bin; now 162.2 opens on 162.2 and on
  nothing else.
- **Read polarity from the log.** `nrz_inverted` on the open line, or the
  opposite-polarity WARN, tells you whether the radio is N or I.

## Variations

- **A list per dongle.** `device: "<serial>"` on a channel moves it to that
  voice SDR. Toned channels on one dongle, a fast untoned list on another.
- **Two users, one frequency.** A `role: wideband` dongle's `conventional:`
  list may hold two entries on one frequency when each has a gate that tells
  them apart — CTCSS 100.0 Hz and DCS 023, say. `decoders` may then sit on
  only one of them. The
  [advanced guide]({{ '/guide-advanced.html' | relative_url }}) has the
  rules.
- **Check a recording offline.** Both decoders have replay harnesses that
  read IQ or an SDR# baseband WAV, find the carrier and print each burst:
  `GT_MDC1200_IQ=<file> go test ./cmd/gophertrunk -run 'TestMDC1200Replay$' -v`
  and the same with `GT_FLEETSYNC_IQ` and `TestFleetSyncReplay$`. The
  [MDC1200]({{ '/mdc1200.html' | relative_url }}) and
  [FleetSync]({{ '/fleetsync.html' | relative_url }}) pages document them.
- **Tone-out on the same channels.** The two-tone pager detector of Volume 1,
  Part 6 listens to decoded audio and coexists with everything here.

## Where this goes next

Four parts of new receivers. [Part
5]({{ '/blog/tutorials/cookbook-volume-2-05-tuning-the-sound/' | relative_url }})
turns to what they produce: voice profiles, loudness, per-channel gain and
FLAC — the knobs that decide how a recording sounds, and the measurements
behind their defaults.

## FAQ

**What is MDC1200, and can an SDR decode it?**
MDC1200 is the short data burst Motorola analog radios send at the start or
end of a transmission, carrying the radio's unit ID and an operation such as
PTT ID or emergency. GopherTrunk decodes it from a pinned dongle or from a
scan-list channel with `decoders: [mdc1200]`.

**Do I need a dedicated SDR for MDC1200 or FleetSync?**
No. Adding `decoders: [mdc1200, fleetsync]` to a `scanner.conventional`
channel decodes bursts while the scanner is on that channel. A dedicated
dongle only matters when every burst on one channel must be caught.

**Why did GopherTrunk log a unit ID when the tone squelch stayed closed?**
The data decoders deliberately ignore the CTCSS or DCS gate. A burst carries
its own sync word and check, so it is logged whenever the carrier clears the
level squelch, whichever tone accompanies it.

**Why does my DCS code not open the squelch?**
Usually polarity. A radio set to D025I does not open a gate set to
`dcs_polarity: normal`. The daemon logs a warning once per channel when it
hears the configured code in the opposite sense; set `inverted` or `both`.

**Why are 150.0 and 151.4 Hz CTCSS tones not separated?**
They are 1.4 Hz apart, closer than the gate's 250 ms tone blocks can
resolve, so a gate on either opens on both. Every other pair of standard
tones is told apart.

## Series navigation

**Part 4 of 14** · ←
[Part 3: An NXDN System — Trunked and Conventional]({{ '/blog/tutorials/cookbook-volume-2-03-nxdn/' | relative_url }})
· Next →
[Part 5: Tuning the Sound — Voice Profiles, Loudness and FLAC]({{ '/blog/tutorials/cookbook-volume-2-05-tuning-the-sound/' | relative_url }})
