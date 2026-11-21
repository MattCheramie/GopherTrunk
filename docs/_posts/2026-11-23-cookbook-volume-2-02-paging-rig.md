---
title: "The Operator's Cookbook Vol. 2, Part 2: The Paging Rig — POCSAG and FLEX on One Dongle"
description: "A complete paging-only GopherTrunk build: one role: dedicated RTL-SDR and a paging.wideband group that splits several POCSAG and FLEX channels out of a single capture, with the copy-paste config, the pager log and REST route, real start-up lines, and a plain account of which FLEX rates are pinned on real air."
category: tutorials
keywords: pocsag decoder rtl-sdr, flex pager decoder sdr, pager decoding software, pocsag flex one dongle, fire paging text decoder, flex 3200 6400 bps, pager log rest api, rtl-sdr paging receiver config, gophertrunk paging
tags: [operator-cookbook, cookbook-volume-2, paging, pocsag, flex, config]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "The Operator's Cookbook, Volume 2"
series_part: 2
---

*Part 2 of **The Operator's Cookbook, Volume 2**, a 14-part second run of
complete, copy-paste GopherTrunk builds — one working rig per part, each with
its verification status stated plainly.
[Part 1]({{ '/blog/tutorials/cookbook-volume-2-01-airband-rig/' | relative_url }})
put AM airband and ACARS on a scan list. This part has no voice and no scan
list at all: one dongle parked on a slice of spectrum, decoding every POCSAG
and FLEX paging channel inside it at once. It is also the rig that, until
recently, quietly decoded nothing unless a browser tab was open — and the
recipe starts from the fix for that.*

> **TL;DR:** The `paging:` section has three lists. `pocsag` and `flex` pin
> one SDR to one frequency each; `wideband` tunes one SDR to
> `center_freq_hz` and runs a digital down-converter bank, one 48 kHz tap per
> channel, any mix of the two protocols. Mark the dongle `role: dedicated`.
> Pages land in the `pager_log` table, `GET /api/v1/pager/messages` and the
> **Pagers** panel, which needs `storage.path`. FLEX decodes 1600, 3200 and
> 6400 bps (2- and 4-level), reading the mode from each frame; only the
> 1600 bps path is pinned on real-air frames, and 3200 / 6400 are
> synthetic-verified. Channels more than 45 % of the sample rate from the
> centre are dropped at start-up with a warning. The whole live path has not
> been run on a real dongle.

**Key takeaways**

- **A paging group streams its own SDR now.** It used to wait on a
  subscription nothing fed, so it decoded only while the Spectrum view was
  open; it now drives the dongle from start-up and retries if the stream
  dies.
- **One capture, several channels.** `paging.wideband` gives each channel
  its own down-converter tap, so two pagers a few hundred kHz apart share a
  dongle — as long as both sit inside the dongle's window.
- **Pick the centre yourself.** Left at 0 the centre is the midpoint of the
  outermost channels, which lands exactly on the middle channel of an odd,
  evenly spaced group — on the tuner's DC spike.
- **FLEX needs no rate setting, and its evidence is uneven.** The mode comes
  from each frame's sync. 1600 bps decodes real-air frames; the faster modes
  have only synthetic signals behind them.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| Several channels, one dongle | tunes to a centre, one tap per channel | `paging.wideband[]`, `center_freq_hz`, `channels[]` |
| One channel, one dongle | tunes off-channel and mixes back | `paging.pocsag[]`, `paging.flex[]` |
| Reserve the dongle | nothing else may retune it | `role: dedicated` |
| POCSAG rate | 512, 1200 or 2400 baud | `baud_hz` (default 1200) |
| FLEX rate | read from each frame, nothing to set | log `flex: frame sync` |
| Where pages land | SQLite log, REST, web panel | `pager_log`, `GET /api/v1/pager/messages`, `/pagers` |
| Persistence | without it the route answers 503 | `storage.path` |
| Check a recording | replay discriminator audio offline | `GT_FLEX_AUDIO`, `TestFLEXAudioReplay` |

## In this post

- **What you're building** — one parked dongle, a tap per paging channel.
- **The shopping list** — and where the frequencies come from.
- **The config** — the complete file, with the centre chosen on purpose.
- **First run — what healthy looks like** — group start, frame sync, rows.
- **When it doesn't work** — symptom → cause → fix.
- **Variations** — single-frequency entries, a slower dongle, remote radios.

## What you're building

A box with one dongle that never retunes. It sits on a centre frequency, and
inside its 2.4 MHz of captured spectrum GopherTrunk runs one narrow
down-converter per paging channel — a
[POCSAG]({{ '/reference/pocsag/' | relative_url }}) channel here, a
[FLEX]({{ '/reference/flex/' | relative_url }}) channel 325 kHz away — and
feeds each 48 kHz stream to the receiver for its protocol. Every decoded page
becomes a row: when it arrived, POCSAG or FLEX, the pager address, the
encoding, the text, and how many bits the error correction repaired.

This extends the paging half of
[Beyond Voice, Part 6]({{ '/blog/deep-dives/beyond-voice-06-pocsag-flex-paging/' | relative_url }}),
which describes the protocols, and the one-line "actual pagers" pointer at
the end of
[Volume 1, Part 6]({{ '/blog/tutorials/operator-cookbook-06-analog-fm-tone-out/' | relative_url }}).
Two things have changed since those were written. The FLEX decoder was
rebuilt for all four modes —
[Rebuilt From the Air, Part 4]({{ '/blog/solution-postmortem/rebuilt-from-the-air-04-flex-four-modes/' | relative_url }})
tells that story. And a paging-only rig did not work on its own. The group
only *subscribed* to its dongle's IQ, and a subscription delivers nothing
unless some decoder is streaming the device; on a box with no trunking
system, the only thing that ever streamed it was a live view. The reported
symptom was exact: "paging decodes only while the Spectrum view is open".
The group now opens the stream itself, like every other decoder that pins a
dongle;
[Squelch, Part 12]({{ '/blog/deep-dives/squelch-12-narrowband-front-end/' | relative_url }})
covers that rule for always-on consumers.

One thing to settle before building. Pager traffic is sent in clear and
routinely carries names, addresses and medical detail. Whether you may
receive it, and what you may do with what you read, depends on where you
live; in some places disclosing it is an offence. Read the
[legal guide]({{ '/police-scanner-legal/' | relative_url }}), decode what you
are entitled to, and keep message bodies and pager addresses to yourself.

<figure class="lab-figure">
<svg viewBox="0 0 680 250" width="680" height="250" role="img" aria-label="Diagram of a wideband paging group: the dongle's IQ band is the centre frequency plus or minus half the sample rate, of which the centre plus or minus forty-five percent is usable; a FLEX channel 175 kHz below the centre and a POCSAG channel 150 kHz above it each get a down-converter tap to 48 kHz and their own receiver, the tuner's DC spike sits exactly on the centre, a channel 1.3 MHz away is outside the band and skipped, and decoded pages flow to the pager log, the REST route and the Pagers panel">
  <text x="340" y="18" text-anchor="middle" fill="var(--fg-muted)" font-size="10">the dongle's IQ band: center_freq_hz ± sample_rate / 2</text>
  <rect x="70" y="30" width="540" height="88" fill="none" stroke="var(--fg-muted)" stroke-dasharray="4 3"/>
  <text x="78" y="46" fill="var(--fg-muted)" font-size="9">usable window: centre ± 45 % of the sample rate (±1.08 MHz at 2.4 MS/s)</text>
  <line x1="40" y1="118" x2="640" y2="118" stroke="currentColor"/>
  <text x="55" y="110" text-anchor="middle" fill="var(--fg-muted)" font-size="8">guard</text>
  <text x="625" y="110" text-anchor="middle" fill="var(--fg-muted)" font-size="8">guard</text>
  <line x1="340" y1="72" x2="340" y2="118" stroke="var(--accent)"/>
  <text x="340" y="66" text-anchor="middle" fill="var(--accent)" font-size="9">centre 153.200 MHz = DC spike</text>
  <rect x="292" y="94" width="8" height="24" fill="none" stroke="currentColor"/>
  <text x="288" y="90" text-anchor="end" fill="currentColor" font-size="9">FLEX 153.025</text>
  <rect x="374" y="94" width="8" height="24" fill="none" stroke="currentColor"/>
  <text x="386" y="90" fill="currentColor" font-size="9">POCSAG 153.350</text>
  <rect x="656" y="94" width="8" height="24" fill="none" stroke="var(--fg-muted)" stroke-dasharray="2 2"/>
  <text x="672" y="136" text-anchor="end" fill="var(--fg-muted)" font-size="9">a channel 1.3 MHz away: outside the band, skipped at start-up</text>
  <line x1="296" y1="118" x2="296" y2="158" stroke="currentColor"/>
  <line x1="378" y1="118" x2="378" y2="158" stroke="currentColor"/>
  <rect x="150" y="158" width="168" height="36" rx="4" fill="none" stroke="currentColor"/>
  <text x="234" y="173" text-anchor="middle" fill="currentColor" font-size="10">FLEX receiver</text>
  <text x="234" y="186" text-anchor="middle" fill="var(--fg-muted)" font-size="9">tap at −175 kHz → 48 kHz</text>
  <rect x="356" y="158" width="168" height="36" rx="4" fill="none" stroke="currentColor"/>
  <text x="440" y="173" text-anchor="middle" fill="currentColor" font-size="10">POCSAG receiver</text>
  <text x="440" y="186" text-anchor="middle" fill="var(--fg-muted)" font-size="9">tap at +150 kHz → 48 kHz</text>
  <line x1="234" y1="194" x2="234" y2="212" stroke="currentColor"/>
  <line x1="440" y1="194" x2="440" y2="212" stroke="currentColor"/>
  <rect x="150" y="212" width="374" height="30" rx="4" fill="none" stroke="var(--accent)"/>
  <text x="337" y="231" text-anchor="middle" fill="var(--accent)" font-size="10">pager.message → pager_log → REST → Pagers panel</text>
  <text x="40" y="180" fill="var(--fg-muted)" font-size="9">one SDR,</text>
  <text x="40" y="192" fill="var(--fg-muted)" font-size="9">role: dedicated</text>
</svg>
<figcaption>A wideband paging group is a fixed window with one tap per channel: channels must sit inside the usable 90 % of the band, and none should sit on the centre, where the tuner's DC spike lives.</figcaption>
</figure>

## The shopping list

| Item | Price (rough) | Notes |
|---|---|---|
| RTL-SDR Blog V3/V4 | ~$35 | one dongle covers every channel within about ±1 MHz of the centre |
| Antenna for the paging band | $0–$30 | in North America paging sits at VHF (the examples here are near 153 MHz) and at 929–932 MHz; use an antenna for the band you chose |
| Computer | $0 | a tap per channel is light work |

Paging frequencies come from your regulator's licence database or a local
frequency listing. You need, per channel, the frequency and the protocol;
for POCSAG also the baud rate, which is 1200 more often than not.

## The config

```yaml
log:
  level: info

storage:
  path: "../data/calls.db"          # pages are rows in the pager_log table

sdr:
  sample_rate: 2_400_000
  devices:
    - serial: "00000001"            # from `gophertrunk sdr list`
      role: dedicated               # reserved for the paging group below
      gain: "auto"

paging:
  wideband:
    - serial: "00000001"
      center_freq_hz: 153_200_000   # chosen: inside both channels' reach, on neither
      channels:
        - protocol: flex
          frequency_hz: 153_025_000
        - protocol: pocsag
          frequency_hz: 153_350_000
          baud_hz: 1200             # 512 | 1200 | 2400
```

The frequencies are the pair from the project's example config; substitute
yours. What each choice does:

**`role: dedicated`.** A dongle that a paging entry names by serial is
reserved for it automatically — trunking, the scan list, `hunt` and rigctld
never retune it — so the role is as much documentation as instruction. It
is also checked: `role: dedicated` on a dongle that no decoder section names
is refused at load, because that dongle would sit idle for ever, which is
almost always a typo in a serial.

**`center_freq_hz`.** Every channel must lie within the centre ± 45 % of the
sample rate: half the rate, less a 5 % guard at each edge. At 2.4 MS/s that
is ±1.08 MHz. Leave the key out and the daemon centres on the midpoint
between the lowest and highest channel. That is fine for two channels and
wrong for three evenly spaced ones: the middle channel then sits at exactly
0 Hz, on the tuner's DC spike. Choose a centre that is at least a channel
width away from every entry.

**`protocol` and `baud_hz`.** `protocol` is `pocsag` or `flex`; anything
else is skipped with a warning. `baud_hz` applies to POCSAG only. FLEX has no
rate key because the rate is not a property of the channel — every 1.875 s
frame opens at 1600 symbols per second with a sync word that names the mode
for the rest of that frame.

**`storage.path`.** Pages surface only through the SQLite log. No
`recordings:` section is needed: this rig records no audio, and the daemon
says so at start-up rather than complaining.

## First run — what healthy looks like

```sh
gophertrunk run -config config.yaml
```

These lines are from a run of that exact `paging:` block against a
`baseband.replay` tuner carrying a FLEX signal, so `serial=` and `driver=`
name the replay source where yours will name the dongle:

```text
INF daemon: SDR reserved for its single-channel decoder (role: dedicated) serial=replay-pg configured_role=dedicated
INF device opened driver=baseband-replay serial=replay-pg role=dedicated rate_hz=2400000 ppm=0 bias_tee=false
INF daemon: recordings.dir not set — decode-only (live audio only, no .wav files written)
INF paging.wideband: group starting serial=replay-pg center_hz=153200000 sample_rate_hz=2400000 channels_hz="[153025000 153350000]"
INF flex: frame sync source=replay-pg bps=1600 symbol_rate=1600 levels=2 inverted=false
```

`group starting` is the line that proves the fix: the group has tuned the
dongle and opened its stream, with no viewer involved. `channels_hz` lists
the channels that survived the window check. `flex: frame sync` appears at
INFO the first time a FLEX channel syncs and tells you what the network
runs — `bps=6400 symbol_rate=3200 levels=4` on a fast one; later mode changes
log at DEBUG. POCSAG prints nothing on sync; its proof is a row.

The signal in that run was three of the real-air FLEX frames committed in
`internal/radio/pager/flex/testdata/p2000_1600x2_frames.hex`, re-modulated
175 kHz below the group centre. `curl
'http://127.0.0.1:8080/api/v1/pager/messages?limit=2'` returned the network's
own test pages:

```json
[
  {
    "id": 2,
    "received_at": "2026-11-21T19:45:45.708338178Z",
    "protocol": "flex",
    "ric": 1180000,
    "func": 0,
    "encoding": "alpha",
    "body": "TESTOPROEP MOB",
    "corrected": 0
  },
  {
    "id": 1,
    "received_at": "2026-11-21T19:45:43.834294479Z",
    "protocol": "flex",
    "ric": 1120103,
    "func": 0,
    "encoding": "alpha",
    "body": "test",
    "corrected": 1
  }
]
```

Before the first page the route returns `[]`. The **Pagers** panel
(`/pagers`) shows the same rows — Received, Type, RIC, Fn, Enc, Body, BER —
with a POCSAG or FLEX badge per row, and the stream carries each page as a
`pager.message` event.

**Status, in one sentence each.** FLEX at 1600 bps is capture-verified from
the symbol stream up: five frames from a recording of a live network decode
to the pages an independent decoder's test suite expects, fragment
reassembly included. FLEX at 3200 and 6400 bps is reference-derived and
synthetic-verified only — the phase demultiplexer has never seen a capture.
POCSAG's protocol layer is unit-tested, but its receiver's end-to-end IQ test
is still skipped for want of a real capture. And the fixed live path — a real
dongle, a real paging channel, this config — has not been run on hardware;
the run above used a replay tuner.

## When it doesn't work

| Symptom | Likely cause | Fix |
|---|---|---|
| Pagers panel empty and the REST route answers 503 | no database | Set `storage.path`; the daemon also prints a start-up warning that paging is configured without it |
| A channel is missing from the group's start line | it is outside the window | The warning reads `paging.wideband[<serial>]: channel … Hz is … kHz from the group center … — skipped`. Move `center_freq_hz`, or put that channel on another SDR |
| Pages arrive only while a Spectrum tab is open | a build from before the fix | Update. The signature is the pair `sdr: no decoder is streaming this SDR — streaming it for the live view` and `sdr: last live view closed — stopping its IQ stream` on a dongle that has a paging group |
| One channel of an odd-count group decodes badly or not at all | it sits on the centre, under the DC spike | Set `center_freq_hz` explicitly, off every channel |
| The group never starts | the serial is not in the pool | Look for `paging.wideband: SDR not found, skipping group`, then compare the serial with `gophertrunk sdr list` |
| Daemon refuses the config over a dedicated role | no decoder section names that dongle | Fix the serial in the `paging:` entry, or change the role |
| POCSAG channel is strong but silent | wrong rate | Try `baud_hz: 512` or `2400` |
| FLEX syncs at a fast rate but pages are missing or garbled | the 3200 / 6400 bps paths are not capture-verified | Record discriminator audio and replay it with `GT_FLEX_AUDIO=<file> go test ./internal/radio/pager/flex/receiver -run TestFLEXAudioReplay -v`; a capture of a multi-rate channel is what closes this gate |
| Pages stop after a USB hiccup | the stream ended | The group retries with a back-off and logs `paging.wideband: decoder stopped — restarting`; if that repeats without recovering, restart the daemon — the paging decoders do not yet re-acquire an unplugged dongle the way the control-channel decoder does |

Polarity is not on that list on purpose. Both receivers resolve an inverted
discriminator from the sync word, so there is no "invert" key to get wrong.

### How this recipe shapes operator practice

- **Read the start line, not the config.** `channels_hz` in `group starting`
  is the list that is actually being decoded. A channel you configured and
  do not see there was dropped, and a warning above it says why.
- **Plan the window before the frequencies.** Centre, sample rate and guard
  decide what one dongle can hold. Do that arithmetic once, with the centre
  off every channel, and adding a channel later is a one-line change.
- **Believe rows, not sync.** A sync line says a frame was recognised. A row
  with a sensible body and a low `corrected` count says it was decoded.

## Variations

- **One frequency per dongle.** `paging.pocsag` and `paging.flex` take
  `serial` and `frequency_hz` (plus `baud_hz` for POCSAG) and need no centre:
  the daemon tunes the dongle below the channel and mixes it back, so the DC
  spike misses it by construction. A start-up line beginning
  `pocsag: tuning the SDR off-channel` shows the arithmetic: at 2.4 MS/s a
  152.0075 MHz entry logged `lo_hz=151489500 offset_hz=518000`.
- **A slower, quieter dongle.** A dongle pinned by a paging entry may carry
  its own `sample_rate` under `sdr.devices`. A wideband group's window
  follows that rate, so a lower rate narrows it.
- **A dongle somewhere else.** The `serial` may name an `sdr.rtl_tcp` entry
  instead of a local device —
  [Volume 1, Part 8]({{ '/blog/tutorials/operator-cookbook-08-remote-radios/' | relative_url }})
  covers remote radios.
- **Pages beside voice.** Add this `paging:` block and its dongle to any
  Volume 1 rig. The two-tone detector of
  [Beyond Voice, Part 5]({{ '/blog/deep-dives/beyond-voice-05-two-tone-tone-out/' | relative_url }})
  tells you a station was paged; the page text tells you why.

The operator page for all of this is the
[POCSAG and FLEX reference]({{ '/pocsag.html' | relative_url }}).

## Where this goes next

Two parts in, nothing has had a control channel. [Part
3]({{ '/blog/tutorials/cookbook-volume-2-03-nxdn/' | relative_url }}) brings
one back: an NXDN system, trunked and conventional, whose frame layer was
rebuilt against real captures and whose trunked voice a reporter has now
followed on air.

## FAQ

**Can one RTL-SDR decode POCSAG and FLEX at the same time?**
Yes, when the channels fit one capture. A `paging.wideband` group tunes the
dongle to a centre frequency and gives each channel its own down-converter
tap and receiver; at 2.4 MS/s every channel must be within about 1.08 MHz of
the centre.

**Why did GopherTrunk only decode pages while the Spectrum view was open?**
The paging group subscribed to its dongle's IQ without starting the stream,
and on a paging-only rig nothing else started it except a live view. The
group now streams the dongle itself from start-up, so a current build decodes
with no browser open.

**Does GopherTrunk decode 3200 and 6400 bps FLEX?**
It decodes every FLEX mode — 1600, 3200 (2- and 4-level) and 6400 bps — and
reads the mode from each frame. Only 1600 bps is pinned on real-air frames;
the faster modes decode synthetic signals and still await a capture from a
live multi-rate network.

**How do I see decoded pager messages?**
Set `storage.path`, then open the Pagers panel in the web console or call
`GET /api/v1/pager/messages?limit=N`. Rows carry the protocol, the pager
address, the encoding, the text and the number of corrected bits.

**Is it legal to decode pagers?**
It depends on the country, and often on what you do with the content. Pager
messages are unencrypted but frequently private. Check your local law before
decoding, and do not publish what you read.

## Series navigation

**Part 2 of 14** · ←
[Part 1: The Airband Rig — AM Voice and ACARS on One Dongle]({{ '/blog/tutorials/cookbook-volume-2-01-airband-rig/' | relative_url }})
· Next →
[Part 3: An NXDN System — Trunked and Conventional]({{ '/blog/tutorials/cookbook-volume-2-03-nxdn/' | relative_url }})
