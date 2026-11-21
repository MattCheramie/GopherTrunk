---
title: "The Operator's Cookbook Vol. 2, Part 1: The Airband Rig — AM Voice and ACARS on One Dongle"
description: "A complete GopherTrunk airband build on one RTL-SDR: a scanner.conventional list of mode: am channels squelched on carrier-to-noise instead of dBFS, plus decoders: [acars] on a data channel, with the copy-paste config, the log lines a healthy run prints, and the honest status of each half."
category: tutorials
keywords: airband sdr scanner, am aviation scanner software, acars decoder rtl-sdr, vhf airband rtl-sdr config, carrier to noise squelch, atc scanner sdr setup, acars 131.550 decode, aircraft radio sdr recording, gophertrunk airband
tags: [operator-cookbook, cookbook-volume-2, airband, acars, am, scanning]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "The Operator's Cookbook, Volume 2"
series_part: 1
---

*Part 1 of **The Operator's Cookbook, Volume 2**, a 14-part second run of
complete, copy-paste GopherTrunk builds — one working rig per part.
[Volume 1]({{ '/blog/series/operator-cookbook/' | relative_url }}) took one
box from a $40 P25 starter to a
[kitchen-sink config]({{ '/blog/tutorials/operator-cookbook-14-kitchen-sink-config/' | relative_url }}).
Volume 2 is built on what landed in the project afterwards: an airband rig,
paging, NXDN, fleet signalling, alerts, transcription, decryption with your
own keys, a phone in your pocket, a band survey, a meter walk and a Raspberry
Pi sized honestly. Same skeleton as before — hardware, config, the log lines
that prove it works, the table for when it doesn't — with one addition: every
recipe says in a plain sentence what has been verified on air and what has
not. We open with the build Volume 1 could not do at all: VHF airband, which
is AM, and the ACARS data link that shares the band.*

> **TL;DR:** Airband is a **scan list of `mode: am` channels** under
> `scanner.conventional`, driven by one `role: voice` SDR. An AM channel
> ignores `squelch_dbfs`: it opens on the carrier's **carrier-to-noise ratio**,
> `squelch_cn_db` (default 12 dB; noise alone reads about 3–7 dB), so a gain
> change does not move it. The voice chain finds the carrier within ±4.5 kHz
> and recentres on it, so a few ppm of tuning error costs nothing. Add
> `decoders: [acars]` to an AM channel on 131.550 MHz and decoded blocks land
> in the `acars_log` table, `GET /api/v1/acars/messages` and the **ACARS**
> panel — which needs `storage.path`. Status: AM voice is on-air verified
> (#1219); ACARS is verified against acarsdec in both directions, with the
> reporter's live run still pending (#1231).

**Key takeaways**

- **Airband squelch is a ratio, not a level.** `squelch_cn_db` compares the
  strongest bin near the channel centre with the channel's own noise floor;
  both move together with gain, so the threshold you set today still means
  the same thing after you touch the gain.
- **`frequency_hz` is the carrier, not the channel name.** An 8.33 kHz
  channel "118.005" is the 118.000 MHz carrier; the ±4.5 kHz channel filter
  covers 25 kHz and 8.33 kHz spacing alike.
- **ACARS is a tap on a channel you already scan.** `decoders: [acars]`
  runs on the AM channel's own IQ, so no dongle is dedicated to it — at the
  price that a block which starts while the scanner is elsewhere is missed.
- **The two halves have different evidence.** The AM chain was confirmed by a
  reporter on real aircraft radio; the ACARS decoder is pinned to an
  independent decoder and real-air blocks, not yet to a live scanner run.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| The scan list | fixed channels, scanned in order on one SDR | `scanner.conventional[]`, `role: voice` |
| AM channel | envelope detection, ±4.5 kHz channel filter | `mode: am` |
| AM squelch | carrier-to-noise in dB, gain-independent | `squelch_cn_db` (default 12) |
| Off-frequency carrier | found within ±4.5 kHz and mixed to centre | log `composer: AM carrier found off the tuned frequency` |
| Hum under the voice | 300 Hz audio high-pass, on by default | `recordings.fm_audio_highpass_hz` |
| ACARS decode | 2400 bit/s MSK off an AM channel | `decoders: [acars]` |
| Where ACARS lands | SQLite log, REST, web panel | `acars_log`, `GET /api/v1/acars/messages`, `/acars` |
| Did it open, and on what | level against threshold at the moment of opening | log `conv: squelch opened` |

## In this post

- **How Volume 2 relates to Volume 1** — what changed, and how to read the status lines.
- **What you're building** — a scan list of AM channels with a data tap.
- **The shopping list** — one dongle, an airband antenna.
- **The config** — the complete file, every key verified.
- **First run — what healthy looks like** — a carrier opening, a block landing.
- **When it doesn't work** — symptom → cause → fix, then the variations.

## How Volume 2 relates to Volume 1

Volume 1 is still the place to start: its
[Part 1]({{ '/blog/tutorials/operator-cookbook-01-forty-dollar-p25-rig/' | relative_url }})
gets a dongle recognised and a first system locked, and nothing here replaces
it. Volume 2 assumes you can already run `gophertrunk sdr list` and
`gophertrunk run -config config.yaml`, and spends its words on rigs that did
not exist in September.

Two conventions hold for all fourteen parts. Log lines are printed the way
Volume 1 printed them — the level abbreviated to `INF` / `WRN`, the timestamp
dropped — but the message text and the `key=value` fields are exactly what
the daemon writes, so they grep. And each recipe uses the project's four
status words literally: **synthetic-verified** (decodes its own test
signals), **reference-pinned** (matches an independent decoder),
**capture-verified** (decodes a recording off the air) and **on-air
verified** (an operator ran it live and confirmed the result). A recipe never
rounds one of those up to the next.

## What you're building

A laptop or a small box with one dongle, stepping through a list of airband
channels — a tower, a ground frequency, a departure sector — the way a
handheld scanner does. When a pilot or controller keys up, the scanner
dwells, demodulates AM, and files the transmission as a call in **History**
with a WAV on disk, exactly like the analog FM dwells of
[Volume 1, Part 6]({{ '/blog/tutorials/operator-cookbook-06-analog-fm-tone-out/' | relative_url }}),
which this recipe extends. One more entry in the same list sits on an
[ACARS]({{ '/reference/acars/' | relative_url }}) frequency and decodes the
aircraft data link into text: registration, flight number, label, message.

Two things make airband different from the FM channels of Volume 1. First,
civil aviation voice is
[amplitude-modulated]({{ '/reference/amplitude-modulation/' | relative_url }});
an FM chain pointed at a tower frequency records silence. `mode: am` swaps
the discriminator for an envelope detector that divides by the tracked
carrier, so the carrier itself is the gain reference and audio level does not
depend on signal strength. Second, the squelch. A dBFS threshold is a number
about your gain knob; under an automatic gain control it screens nothing. An
AM channel instead measures
[carrier-to-noise]({{ '/reference/carrier-to-noise-ratio/' | relative_url }})
from its own spectrum, and the mechanism — and the captures that shaped it —
is the subject of
[Squelch, Part 7]({{ '/blog/deep-dives/squelch-07-am/' | relative_url }}).

<figure class="lab-figure">
<svg viewBox="0 0 680 250" width="680" height="250" role="img" aria-label="Block diagram of the airband rig: an antenna feeds one voice-role SDR owned by the conventional scanner, whose loop steps through Tower, Ground and an ACARS channel; an AM channel opens on carrier-to-noise in dB rather than dBFS; an open channel feeds the AM voice chain, which recentres on the carrier and records a call, and on the ACARS channel also feeds the MSK decoder that writes blocks to the acars_log table, the REST route and the ACARS web panel">
  <rect x="8" y="104" width="62" height="34" rx="4" fill="none" stroke="currentColor"/>
  <text x="39" y="125" text-anchor="middle" fill="currentColor" font-size="10">antenna</text>
  <line x1="70" y1="121" x2="92" y2="121" stroke="currentColor"/>
  <rect x="92" y="104" width="84" height="34" rx="4" fill="none" stroke="currentColor"/>
  <text x="134" y="119" text-anchor="middle" fill="currentColor" font-size="10">RTL-SDR</text>
  <text x="134" y="132" text-anchor="middle" fill="var(--fg-muted)" font-size="9">role: voice</text>
  <line x1="176" y1="121" x2="200" y2="121" stroke="currentColor"/>
  <rect x="200" y="14" width="214" height="222" rx="6" fill="none" stroke="var(--fg-muted)" stroke-dasharray="4 3"/>
  <text x="307" y="30" text-anchor="middle" fill="var(--fg-muted)" font-size="10">scanner.conventional</text>
  <rect x="214" y="42" width="186" height="42" rx="4" fill="none" stroke="currentColor"/>
  <text x="307" y="58" text-anchor="middle" fill="currentColor" font-size="10">scan loop: tune next channel</text>
  <text x="307" y="72" text-anchor="middle" fill="var(--fg-muted)" font-size="9">Tower → Ground → ACARS → …</text>
  <line x1="307" y1="84" x2="307" y2="102" stroke="currentColor"/>
  <rect x="214" y="102" width="186" height="42" rx="4" fill="none" stroke="var(--accent)"/>
  <text x="307" y="118" text-anchor="middle" fill="var(--accent)" font-size="10">mode: am squelch</text>
  <text x="307" y="132" text-anchor="middle" fill="var(--fg-muted)" font-size="9">squelch_cn_db: carrier ÷ noise, dB</text>
  <line x1="307" y1="144" x2="307" y2="162" stroke="var(--accent)"/>
  <rect x="214" y="162" width="186" height="42" rx="4" fill="none" stroke="currentColor"/>
  <text x="307" y="178" text-anchor="middle" fill="currentColor" font-size="10">dwell until carrier drops</text>
  <text x="307" y="192" text-anchor="middle" fill="var(--fg-muted)" font-size="9">+ hangtime_ms</text>
  <line x1="400" y1="183" x2="440" y2="76" stroke="currentColor"/>
  <line x1="400" y1="183" x2="440" y2="186" stroke="currentColor"/>
  <rect x="440" y="48" width="230" height="56" rx="4" fill="none" stroke="currentColor"/>
  <text x="555" y="65" text-anchor="middle" fill="currentColor" font-size="10">AM voice chain</text>
  <text x="555" y="79" text-anchor="middle" fill="var(--fg-muted)" font-size="9">find carrier ±4.5 kHz → envelope → 3 kHz</text>
  <text x="555" y="93" text-anchor="middle" fill="var(--fg-muted)" font-size="9">WAV + History row (am-conv)</text>
  <rect x="440" y="158" width="230" height="56" rx="4" fill="none" stroke="var(--accent)"/>
  <text x="555" y="175" text-anchor="middle" fill="var(--accent)" font-size="10">decoders: [acars]</text>
  <text x="555" y="189" text-anchor="middle" fill="var(--fg-muted)" font-size="9">2400 bit/s MSK → block check</text>
  <text x="555" y="203" text-anchor="middle" fill="var(--fg-muted)" font-size="9">acars_log → REST → ACARS panel</text>
  <text x="555" y="232" text-anchor="middle" fill="var(--fg-muted)" font-size="9">the data tap runs only on the channel that lists it</text>
</svg>
<figcaption>One dongle, one scan list: every AM channel opens on carrier-to-noise and records voice, and the channel that lists the ACARS decoder also turns its bursts into log rows.</figcaption>
</figure>

## The shopping list

| Item | Price (rough) | Notes |
|---|---|---|
| RTL-SDR Blog V3/V4 | ~$35 | airband AM is an easy decode; the squelch does not care about gain |
| Airband antenna | $0–$40 | 118–137 MHz, vertically polarised: the kit whip extended to about 60 cm works, a dedicated airband antenna works better |
| Computer | $0 | anything from the last decade |

You look frequencies up by airport: tower, ground, approach and departure
each have their own. The
[aviation scanner guide]({{ '/best-aviation-scanner/' | relative_url }})
covers where to find them, and the
[scanner frequencies guide]({{ '/scanner-frequencies/' | relative_url }})
lists the band by service. The ACARS frequencies in common use are in the
[ACARS operator page]({{ '/acars.html' | relative_url }}): 131.550, 131.525,
130.025, 129.125 and 131.125 MHz among them. Receiving airband is lawful in
many countries and restricted in some; check yours before you record.

## The config

```yaml
log:
  level: info

storage:
  path: "../data/calls.db"      # the ACARS log is a table in this database

recordings:
  dir: "../recordings"

sdr:
  sample_rate: 2_400_000
  devices:
    - serial: "00000001"        # from `gophertrunk sdr list`
      role: voice               # the scan list drives a voice SDR
      gain: "auto"

scanner:
  conventional:
    - label: "Tower"
      frequency_hz: 118_700_000 # the actual carrier, not an 8.33 kHz channel name
      mode: am
      squelch_cn_db: 12         # carrier-to-noise in dB; 12 is the default
      hangtime_ms: 1500
      priority: 4
      talkgroup_id: 2147500101  # pin it (Volume 1, Part 6)
    - label: "Ground"
      frequency_hz: 121_900_000
      mode: am
      talkgroup_id: 2147500102
    - label: "ACARS primary"
      frequency_hz: 131_550_000
      mode: am                  # required: acars is refused on an fm channel
      decoders: [acars]
      hangtime_ms: 250          # no voice here: keep the burst "calls" short
      talkgroup_id: 2147500103

audio:
  enabled: true                 # hear dwells on the host speakers
```

The frequencies are placeholders; replace them with your airport's. The
decisions behind the rest:

**`role: voice`.** The conventional scanner takes the last voice device in
the pool. On an airband-only rig that is your only dongle. With no voice SDR
the daemon logs a WARN at start-up and skips the scanner — the row in the
table below.

**`mode: am` and `squelch_cn_db`.** On an AM channel `squelch_dbfs` is
ignored. The scanner decimates the channel to about 48 kHz, averages a short
power spectrum, and divides the strongest bin within ±4.5 kHz of the channel
centre by the 20th-percentile bin across ±12 kHz. On receiver noise that
ratio reads roughly 3–7 dB at any gain; 12 dB opens on a weak but
intelligible carrier. Raise it on a channel that opens on interference;
you should rarely need to lower it. Because nothing here is absolute,
`gain: "auto"` is safe — the property an FM channel only got later, with the
noise squelch of
[Squelch, Part 3]({{ '/blog/deep-dives/squelch-03-noise-squelch/' | relative_url }}).

**`frequency_hz`.** Enter the carrier. The 8.33 kHz channel names printed on
charts are labels, not frequencies: the name "118.005" denotes the 118.000 MHz
carrier. The ±4.5 kHz channel filter passes both spacings.

**`decoders: [acars]`.** Config validation accepts `acars` only on a
`mode: am` channel; on `fm` or `nfm` the file is refused with an error that
says so. A block lasts up to about 0.9 s; once the decoder has locked a
block's sync, the scanner holds the channel for up to one second so a hop
cannot cut it off. The short `hangtime_ms` on that entry is deliberate: a
data burst raises a carrier, so it opens an AM call like any transmission,
and with no voice to wait for there is no reason to hold the channel 1.5 s
after each one.

**`storage.path`.** Decoded blocks reach you only through the SQLite log.
Without `storage.path` the decoder still runs, but `GET
/api/v1/acars/messages` answers 503 and the panel stays empty.

**`talkgroup_id`.** Unpinned channels get an ID built from their list
position, which shifts when you edit the list. Pin them, as in Volume 1.

## First run — what healthy looks like

```sh
gophertrunk run -config config.yaml
```

A first run on a new machine stops at the Terms of Service: read them with
`gophertrunk terms` and acknowledge with `gophertrunk terms accept`. After
that the scanner announces how it will tune:

```text
INF conv: scanner LO offset tuning serial=00000001 lo_offset_hz=518000 mode=auto sample_rate_hz=2400000
INF api: listening addr=127.0.0.1:8080 tls=false
```

The first line is not specific to airband, but it matters here: the scanner
parks the tuner 518 kHz below each channel at 2.4 MS/s and mixes the channel
back, so the tuner's DC spur never sits on your carrier
([Squelch, Part 4]({{ '/blog/deep-dives/squelch-04-offset-tuning/' | relative_url }})
has the reason).

An idle list is silent in the log. When a carrier appears, four lines tell
the story. These are the lines the daemon printed when the project's
committed real-air AM slice — a 126.400 MHz departure channel, interpolated
to 2.4 MS/s and mounted as a `baseband.replay` tuner — was run through an AM
scan-list channel:

```text
INF conv: squelch opened freq_hz=126400000 label=Departure mode=am level_cn_db=60.6 squelch_cn_db=12
INF synthetic call started device=replay-am grant="scanner/am-conv tg=2147500101 src=0 freq=126400000 "
INF recorder: call started device=replay-am wav=../recordings/scanner/2147500101/20261121_193112_2147500101.wav tg=2147500101 provoice=false vocoder=""
```

`level_cn_db` against `squelch_cn_db` is the whole squelch decision in one
line: 60.6 dB of carrier over a 12 dB threshold. The protocol tag is
`am-conv`, the system is `scanner`, and recordings file under
`recordings/scanner/<talkgroup_id>/`. On your rig `device=` is your dongle's
serial. When the carrier drops and `hangtime_ms` expires:

```text
INF synthetic call ended device=... reason=normal
INF recorder: call ended device=... wav=... duration=... reason=normal
```

For ACARS there is no log line per block; the proof is the log table. A
synthetic AM burst carrying one of the real-air blocks pinned in
`internal/radio/acars/acars_test.go`, replayed through the `ACARS primary`
entry above, produced this from
`curl 'http://127.0.0.1:8080/api/v1/acars/messages?limit=1'`:

```json
[
  {
    "id": 5,
    "received_at": "2026-11-21T19:35:54.141660126Z",
    "mode": "E",
    "address": "PH-BXR",
    "ack": "!",
    "label": "5V",
    "block_id": "4",
    "downlink": true,
    "msg_no": "S53A",
    "flight_id": "KL1681",
    "crc_ok": true,
    "raw_hex": "45aed0c8adc2585215b5d63402d3b5b3c1cb4c31b638318314fc",
    "serial": "replay-acars",
    "frequency_hz": 131550000
  }
]
```

The same row shows in the web console's **ACARS** panel (`/acars`) with
columns Received, Aircraft, Flight, Dir, Label, Text, Channel and Check;
`web.tabs.acars: false` hides the tab. A block that needed bits repaired
before its check validated carries a `corrected` count. That run also showed
the known limit plainly: the 0.66 s burst opened an ordinary AM call. With
the default 1.5 s hangtime it left a 2.17 s recording; with `hangtime_ms:
250` as configured above, 0.92 s — and the block decoded either way.

**Status, in one sentence each.** AM voice on the scan list is on-air
verified: on 29 Sep a reporter on a build with the carrier-tracking fix heard
"very good and clear" audio, on frequency, from real aircraft radio (#1219).
ACARS is reference-pinned and capture-verified offline — GopherTrunk decodes
all seven real-air messages in the acarsdec project's own test recording to
the fields acarsdec prints, and acarsdec decodes GopherTrunk's synthesised
audio — but the reporter's live scanner run is still pending (#1231).

## When it doesn't work

| Symptom | Likely cause | Fix |
|---|---|---|
| `daemon: scanner.conventional / manual_tune_enabled configured but no Voice SDRs in the pool; skipping` | no `role: voice` device | Give the scanner a voice SDR; on a combined trunked + airband rig, add a second one — the scanner takes the last |
| Config refused: `acars needs mode: am` | the ACARS channel is `fm` or `nfm` | Set `mode: am` on that entry |
| Channel never opens, though you hear traffic elsewhere | carrier under 12 dB C/N, or the wrong frequency | Read `level_cn_db` in `conv: squelch opened` on a channel that does open; fix the antenna first, then lower `squelch_cn_db` a little — noise alone reaches about 7 dB, so stay well above that |
| Channel opens on noise or a neighbour | local interference in the channel | Raise `squelch_cn_db` (the advanced guide's ATIS example uses 15) |
| `composer: AM carrier found off the tuned frequency; tracked and recentred` | `frequency_hz` or the dongle's `ppm` is off by 1 kHz or more | Nothing is lost — the chain recentres within ±4.5 kHz — but set `frequency_hz` to the carrier, or the device `ppm` |
| Low hum under every transmission | mains or airframe hum below 300 Hz | Leave `recordings.fm_audio_highpass_hz` at its default 300; it applies to AM too. If you disabled it (`-1`), re-enable it |
| `GET /api/v1/acars/messages` returns 503, panel empty | no `storage.path` | Set `storage.path` |
| Short AM recordings beside every ACARS row | by design today: a burst raises the channel's carrier, so it opens an AM call | Expected; keep them short with a small `hangtime_ms` on the ACARS entry, as in the config above |
| Scanner parks on one channel for hours | a continuous carrier (ATIS, VOLMET) is on the scan list | Take it off the list, or use the wideband variation below, where nothing scans |

Two of those rows deserve their history. The **off-frequency** row comes
from the first on-air captures: a reporter found a channel "tuned best at
123.453, not 123.450". The capture showed the carrier 3.3 kHz away from where
the filter was centred — the transmitter was on frequency; the channel filter
was cutting a sideband. The chain now finds the carrier (the strongest
spectral peak within ±4.5 kHz, at least 15 dB over the window's median, held
on noise) and mixes it to the centre before filtering. The **hum** row is
not hypothetical either: in that reporter's recordings most of the decoded
energy sat below 300 Hz, from interference in a hangar. The audio high-pass
that strips sub-audible CTCSS tones from FM channels is what removes it.

One caveat on reading levels. `GET /api/v1/scanner` reports each channel's
`level_cn_db`, and during a dwell it is trustworthy. Between dwells the
figure is seeded while the meter is still filling, so an idle AM channel can
show a large negative number. Use the value in the `conv: squelch opened`
line.

### How this recipe shapes operator practice

- **Stop tuning squelch against gain.** On AM the threshold is a ratio. If a
  channel misbehaves, the cause is the antenna, the frequency or
  interference — not the gain setting, and not a number to nudge.
- **Enter carriers, not channel names.** One wrong 8.33 kHz conversion gives
  a channel that opens and records thin audio; the recentring line in the
  log is how you catch it.
- **Treat a scan-list data tap as sampling, not logging.** It reports what
  was sent while the scanner was listening. For a complete ACARS log, give
  the data channels a receiver that never leaves them.

## Variations

- **Continuous coverage of one airport.** A `role: wideband` dongle accepts a
  `conventional:` list of the same channel entries, each on its own
  always-on tap — nothing scans, so a long transmission on tower never hides
  ground, and a continuous ATIS is harmless. Every channel must fit the
  dongle's window around `center_freq_hz`:

  ```yaml
  sdr:
    sample_rate: 2_400_000
    devices:
      - serial: "00000001"
        role: wideband
        gain: "auto"
        center_freq_hz: 119_000_000
        conventional:
          - { label: "Tower", frequency_hz: 118_300_000, mode: am }
          - { label: "Ground", frequency_hz: 119_900_000, mode: am }
          - { label: "ATIS", frequency_hz: 119_450_000, mode: am, squelch_cn_db: 15 }
  ```

  The same shape, centred near 131.3 MHz with `decoders: [acars]` on each
  entry, watches several ACARS channels at once. The
  [advanced guide]({{ '/guide-advanced.html' | relative_url }}) documents
  the wideband conventional list.
- **A dongle per band.** Add `device: "<serial>"` to a channel and it scans on
  that voice SDR instead; channels naming the same device form that SDR's
  list, and the lists run in parallel. Airband on one antenna, VHF FM on
  another.
- **Per-channel gain and level.** `gain` sets the tuner gain for one channel
  on a mixed list; `audio_gain_db` (±30 dB) balances its recorded loudness
  against FM and digital channels.
- **Aircraft positions too.** ADS-B at 1090 MHz is a separate receiver on its
  own dongle;
  [Beyond Voice, Part 8]({{ '/blog/deep-dives/beyond-voice-08-adsb-mode-s/' | relative_url }})
  covers it.

## Where this goes next

Airband needed a new demodulator and a new squelch. [Part
2]({{ '/blog/tutorials/cookbook-volume-2-02-paging-rig/' | relative_url }})
needs neither voice nor a scan list: it pins one dongle to a group of paging
channels and decodes POCSAG and FLEX off all of them at once — including the
fix that made a paging-only box work without a browser tab open.

## FAQ

**Can an RTL-SDR receive aircraft radio?**
Yes. Civil airband voice on 118–137 MHz is AM and well within an RTL-SDR's
range. In GopherTrunk it is a `scanner.conventional` channel with `mode: am`
on a `role: voice` dongle; each transmission is recorded and logged as a call.

**Why does GopherTrunk use squelch_cn_db instead of squelch_dbfs for AM?**
A dBFS threshold moves with gain and means nothing under automatic gain
control. `squelch_cn_db` is the carrier's level over the channel's own noise
floor, in dB, so it is the same number at any gain. Noise alone reads about
3–7 dB and the default threshold is 12.

**How do I decode ACARS with GopherTrunk?**
Add an AM scan-list channel on an ACARS frequency such as 131.550 MHz with
`decoders: [acars]`, and set `storage.path`. Decoded blocks appear in
`GET /api/v1/acars/messages` and the ACARS web panel, tagged with the channel
they came from.

**Is GopherTrunk's ACARS decoder verified on air?**
Not yet by a live scanner run. It decodes all seven real-air messages in the
acarsdec project's test recording to the same fields acarsdec prints, and
acarsdec decodes GopherTrunk's own synthesised audio. The live confirmation
on issue #1231 is still outstanding.

**Why do I get short recordings on my ACARS channel?**
An ACARS burst is a keyed AM carrier, so it opens the channel's squelch like
a voice transmission and is recorded as a short call. That is a documented
limit of the scan-list tap. A small `hangtime_ms` on the ACARS entry keeps
each one to the length of the burst plus that hangtime.

## Series navigation

**Part 1 of 14** · Next →
[Part 2: The Paging Rig — POCSAG and FLEX on One Dongle]({{ '/blog/tutorials/cookbook-volume-2-02-paging-rig/' | relative_url }})
