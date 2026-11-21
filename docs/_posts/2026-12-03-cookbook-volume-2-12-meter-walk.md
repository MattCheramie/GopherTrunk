---
title: "The Operator's Cookbook Vol. 2, Part 12: The Meter Walk — Wireless M-Bus at 868 MHz"
description: "A complete GopherTrunk recipe for decoding Wireless M-Bus utility-meter telegrams at 868.95 MHz with one RTL-SDR: the wmbus command and every flag that matters on a walk, what an encrypted telegram still shows, gain judged by signal-to-noise rather than dBFS, reading 'cut by dropped IQ', the burst gate, and replaying saved snippets."
category: tutorials
keywords: wireless m-bus rtl-sdr decoder, wmbus 868 mhz sdr, heat cost allocator radio decoder, oms meter telegram sdr, water meter rtl-sdr, wmbus t1 c1 mode decoder, rtl-sdr android termux wmbus, utility meter receive only legal, gophertrunk wmbus
tags: [operator-cookbook, cookbook-volume-2, wmbus, utility-meters, rtl-sdr, termux]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "The Operator's Cookbook, Volume 2"
series_part: 12
---

*Part 12 of **The Operator's Cookbook, Volume 2**, a 14-part tutorial series
of complete, copy-paste GopherTrunk builds — one working rig per part — built
on what landed after Volume 1.
[Part 11]({{ '/blog/tutorials/cookbook-volume-2-11-band-survey/' | relative_url }})
logged a whole band to a CSV. This part points the same dongle at one
frequency and takes it for a walk:
**[Wireless M-Bus]({{ '/reference/wireless-mbus/' | relative_url }})**, the
868 MHz protocol European heat cost allocators, water meters and heat meters
use to report their readings, decoded by `gophertrunk wmbus` live or from a
capture. It is the one recipe in this volume with no `config.yaml` — the
whole build is a command line — and the one where the hardest problems
turned out to be a phone's USB port and its screen timeout.*

> **TL;DR:** `gophertrunk wmbus` decodes Wireless M-Bus / OMS telegrams
> (modes **T1 and C1**, 868.95 MHz, 100 kchip/s) from a local RTL-SDR, an
> `rtl_tcp` server or a capture file — no config file, 1.6 MS/s by default.
> Every telegram prints its sender (manufacturer, meter ID, version, device
> type); unencrypted meters also print their readings, and encrypted ones
> print `encrypted:` with the security mode — no decryption is attempted.
> For a walk: `-gain 200 -all -save-frames meters/ -log wm.log`; judge gain
> by signal-to-noise rather than dBFS, and keep `dropped chunks` at 0. On
> one reporter's five phone walks (2,005 saved telegrams, issue #1256)
> today's receiver decodes 1,418 in the order heard, where the FM
> demodulator it replaced decoded 1,190.

**Key takeaways**

- **The whole recipe is one command.** `wmbus` reads no config file:
  frequency, rate, gain and outputs are flags, and the defaults (868.95 MHz,
  1.6 MS/s) suit nearly every installed meter.
- **The header is always readable; the readings often are not.** Most OMS
  meters encrypt their payload with AES-128 under a key the metering company
  holds. GopherTrunk shows the sender and the security mode, and stops.
- **Signal-to-noise decides, not the dBFS figure.** A −28 dBFS telegram on a
  dongle whose in-channel floor sits near −44 dBFS has 13 dB — the region
  where the current receiver gained the most.
- **A failed CRC is not always the meter's fault.** A live run marks a
  telegram the SDR dropped IQ in the middle of as `cut by dropped IQ`: the
  receiving device's failure, which no replay can recover.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| Live decode | T1 + C1 telegrams at 868.95 MHz, 1.6 MS/s | `gophertrunk wmbus` |
| Phone without root | reach the dongle through a driver app | `-rtltcp 127.0.0.1:1234` ([rtl_tcp]({{ '/reference/rtl-tcp/' | relative_url }})) |
| Gain | fixed tuner gain in tenths of a dB | `-gain 200` (20 dB); `auto` = AGC |
| See the failures | print frames whose CRC failed, with the block | `-all` |
| Keep evidence | one IQ WAV plus a metadata sidecar per telegram | `-save-frames meters/` |
| Survive the walk | a log that outlives Ctrl-C; reopen a lost SDR | `-log wm.log`, `-reconnect` (default on) |
| Burst gate A/B | demodulate all the IQ, as up to v1.4.0 | `-no-gate` |

## In this post

- **What you're building** — a pocket receiver, and what it can read.
- **The shopping list** — one dongle, 8.6 cm of antenna, a laptop or phone.
- **The config** — there is none: four commands, every flag verified.
- **First run — what healthy looks like** — two telegrams and a progress line.
- **When it doesn't work** — CRC failures, dropped IQ, a lost dongle.
- **Variations** — capture first, JSON to a script, replaying snippets.

## What you're building

A meter with a Wireless M-Bus radio (EN 13757-4, and the Open Metering
System built on it) is on the air for a few milliseconds at a time,
anywhere from every few seconds to every few hours. The finished rig is an
RTL-SDR tuned to **868.95 MHz** and a terminal that prints one line per
telegram as you walk past a building: mode, level, manufacturer code
(`TCH` Techem, `QDS` Qundis, `KAM` Kamstrup), the 8-digit meter ID printed
on the device, version and device type — and, for meters that send them in
clear, the data records underneath: volume, energy, heat cost allocation
units, temperatures, dates, and historic values such as the reading at the
last set day.

The scope is narrow and stated in
[the operator doc]({{ '/wmbus.html' | relative_url }}): **modes T1 and C1**,
the two meter-to-reader modes nearly all installed meters use, both at
100 kchip/s. T1 is 3-of-6 coded; C1 is NRZ in frame format A or B. Mode S1
at 868.3 MHz is not decoded.

<figure class="lab-figure">
<svg viewBox="0 0 680 236" width="680" height="236" role="img" aria-label="The wmbus decode chain as five blocks in a row: an RTL-SDR at 868.95 MHz and 1.6 megasamples per second feeds a burst gate that opens 1.5 dB over the noise floor, then the receiver at 400 kilosamples per second with tone correlators and a chip clock, then the framer that finds access code 0x543D and checks each block's CRC, then the telegram parser; below, a timeline shows noise the demodulator skips, a few-millisecond burst, and the 1.3 ms of IQ before the burst that is replayed when the gate opens.">
  <rect x="8" y="22" width="112" height="56" rx="4" fill="none" stroke="currentColor"/>
  <text x="64" y="42" text-anchor="middle" fill="currentColor" font-size="10">RTL-SDR</text>
  <text x="64" y="56" text-anchor="middle" fill="var(--fg-muted)" font-size="9">868.95 MHz</text>
  <text x="64" y="69" text-anchor="middle" fill="var(--fg-muted)" font-size="9">1.6 MS/s IQ</text>
  <line x1="120" y1="50" x2="142" y2="50" stroke="currentColor"/>
  <rect x="142" y="22" width="118" height="56" rx="4" fill="none" stroke="var(--accent)"/>
  <text x="201" y="42" text-anchor="middle" fill="var(--accent)" font-size="10">burst gate</text>
  <text x="201" y="56" text-anchor="middle" fill="var(--fg-muted)" font-size="9">opens 1.5 dB over</text>
  <text x="201" y="69" text-anchor="middle" fill="var(--fg-muted)" font-size="9">the tracked floor</text>
  <line x1="260" y1="50" x2="282" y2="50" stroke="currentColor"/>
  <rect x="282" y="22" width="122" height="56" rx="4" fill="none" stroke="currentColor"/>
  <text x="343" y="42" text-anchor="middle" fill="currentColor" font-size="10">receiver, 400 kS/s</text>
  <text x="343" y="56" text-anchor="middle" fill="var(--fg-muted)" font-size="9">carrier estimate,</text>
  <text x="343" y="69" text-anchor="middle" fill="var(--fg-muted)" font-size="9">tone correlators, clock</text>
  <line x1="404" y1="50" x2="426" y2="50" stroke="currentColor"/>
  <rect x="426" y="22" width="112" height="56" rx="4" fill="none" stroke="currentColor"/>
  <text x="482" y="42" text-anchor="middle" fill="currentColor" font-size="10">framer</text>
  <text x="482" y="56" text-anchor="middle" fill="var(--fg-muted)" font-size="9">access code 0x543D</text>
  <text x="482" y="69" text-anchor="middle" fill="var(--fg-muted)" font-size="9">T1 / C1, CRC per block</text>
  <line x1="538" y1="50" x2="560" y2="50" stroke="currentColor"/>
  <rect x="560" y="22" width="112" height="56" rx="4" fill="none" stroke="currentColor"/>
  <text x="616" y="42" text-anchor="middle" fill="currentColor" font-size="10">telegram</text>
  <text x="616" y="56" text-anchor="middle" fill="var(--fg-muted)" font-size="9">sender: always</text>
  <text x="616" y="69" text-anchor="middle" fill="var(--fg-muted)" font-size="9">readings: if in clear</text>
  <text x="8" y="112" fill="var(--fg-muted)" font-size="10">what the gate lets through</text>
  <line x1="8" y1="160" x2="672" y2="160" stroke="var(--fg-muted)"/>
  <rect x="8" y="146" width="262" height="14" fill="none" stroke="var(--fg-muted)" stroke-dasharray="3 3"/>
  <text x="139" y="140" text-anchor="middle" fill="var(--fg-muted)" font-size="9">noise — skipped, two additions per sample</text>
  <rect x="270" y="130" width="60" height="30" fill="none" stroke="var(--accent)" stroke-dasharray="3 3"/>
  <text x="296" y="112" text-anchor="middle" fill="var(--accent)" font-size="9">1.3 ms lookback</text>
  <rect x="330" y="118" width="150" height="42" fill="none" stroke="var(--accent)"/>
  <text x="405" y="136" text-anchor="middle" fill="var(--accent)" font-size="10">telegram burst</text>
  <text x="405" y="150" text-anchor="middle" fill="var(--fg-muted)" font-size="9">a few ms — demodulated</text>
  <rect x="480" y="146" width="192" height="14" fill="none" stroke="var(--fg-muted)" stroke-dasharray="3 3"/>
  <text x="576" y="140" text-anchor="middle" fill="var(--fg-muted)" font-size="9">noise — skipped</text>
  <text x="8" y="182" fill="var(--fg-muted)" font-size="9">On noise the gated decoder costs 0.004 of a desktop core at 1.6 MS/s, where it cost 0.06 demodulating everything.</text>
  <text x="8" y="198" fill="var(--fg-muted)" font-size="9">The gate needs two consecutive 0.16 ms blocks over the threshold, and stays open while a framer is inside a frame.</text>
  <text x="8" y="214" fill="var(--fg-muted)" font-size="9">Not to scale: on one 57-minute walk the telegrams added up to about 5 s of air.</text>
</svg>
<figcaption>Nearly all of a meter channel is noise, so the demodulator runs only around the bursts — and replays the 1.3 ms before each one so the preamble is not lost.</figcaption>
</figure>

**Encrypted meters.** Most OMS meters encrypt their readings with AES-128
— security mode 5, or mode 7 with an authentication layer — under a
per-meter key held by the metering company. For those, `wmbus` still shows
who sent the telegram, when, the device type and the security mode, and
stops: no decryption is attempted and there is no key option.

**Receive-only, and other people's meters.** This rig transmits nothing and
cannot address a meter. But the telegrams come from other people's homes: a
manufacturer code and a meter ID name one device in one flat, and an
unencrypted reading says something about a household. Whether receiving,
storing or sharing these broadcasts is permitted depends on where you live,
so know your local law before you walk — and do not publish meter IDs: mask
them in anything you post.

**Status, plainly.** The decoder is capture-verified against two independent
decoders (rtl-wmbus's four real-air sample captures, byte for byte, and
wmbusmeters' unencrypted test telegrams) and on-air verified by one
reporter's phone walks
([#1256](https://github.com/MattCheramie/GopherTrunk/issues/1256)). Its two
newest pieces — the tone-comparing detector and the burst gate — are
measured on those walks' saved snippets (the same 602 of the last walk's
770 decode with and without the gate) and not yet run live on a phone.

## The shopping list

| Item | Price (rough) | Notes |
|---|---|---|
| [RTL-SDR]({{ '/reference/rtl-sdr/' | relative_url }}) dongle | ~$35 | the reporter's walks used one on a Samsung S5 |
| Antenna | $0–$15 | a quarter-wave at 868 MHz is about 8.6 cm — shorten a telescopic kit whip to that |
| Laptop, or Android phone + USB-OTG adapter | $0 | the phone build and the driver-app route are [Part 10]({{ '/blog/tutorials/cookbook-volume-2-10-phone-rig/' | relative_url }})'s recipe |

## The config

There is no `config.yaml`: like `power` and `capture`, `wmbus` takes
everything from flags (`gophertrunk help wmbus` lists them all). The recipe
is four steps — a ten-second gain check before leaving, the walk itself on
a local dongle or through a phone's driver app, and the replay at home:

```bash
gophertrunk capture -freq 868950000 -sample-rate 1600000 -seconds 10 \
  -gain 200 -format u8 -out check.cu8          # 1. gain check

gophertrunk wmbus -gain 200 -all -save-frames meters/ \
  -log wm.log -status 1m                        # 2. the walk, local dongle

gophertrunk wmbus -rtltcp 127.0.0.1:1234 -gain 200 -all \
  -save-frames meters/ -log wm.log -status 1m   # 3. the walk, via rtl_tcp

gophertrunk wmbus -in meters/<file>.wav         # 4. replay one telegram
gophertrunk wmbus -in check.cu8                 #    or the whole capture
```

Decisions worth understanding. **`-gain` is tenths of a dB** — `-gain 200`
is 20 dB, `-gain 0` the tuner's minimum, and `auto` hands the choice to the
tuner's AGC, which a strong nearby transmitter can drive into clipping.
Start around 200, step down while `capture` reports clipping, step up only
if telegrams are weak; with a preamplifier in front, start at 0 to 100.
Without one, `-gain 0` is usually too little: on the reporter's walk at
that setting the strongest telegrams reached about 7 of the converter's
127 steps, while a higher-gain test capture clipped — the usual
[gain-staging]({{ '/blog/tutorials/analog-edge-03-gain-staging/' | relative_url }})
squeeze.

**`-all` and `-save-frames` belong together.** Without `-all` only
CRC-valid telegrams are printed and saved, which hides exactly the frames
you need when a meter does not decode. Each saved telegram is an IQ WAV
(typically 100–400 KB) plus a `.metadata.json` naming what it holds.
**`-log`** is written as things happen and records why a run stopped;
prefer it to piping through `tee`, which Ctrl-C kills along with
GopherTrunk.

**Leave `-rate` and `-ppm` alone.** Since v1.3.5 the decoder costs about
the same at any rate from 1.024 to 1.6 MS/s, and 1.6 MS/s decodes slightly
more weak telegrams. A dongle's crystal error does not need `-ppm` either:
the decoder hears a carrier up to about ±120 kHz off, re-centres on the
first CRC-valid telegram and narrows its channel filter to it. On the
reporter's first 156 saved telegrams that tracking took the count from 97
CRC-valid to 128 — every meter sat about 50 kHz (58 ppm) low, which was the
dongle, not the meters.

## First run — what healthy looks like

The run announces its radio, then waits. An unencrypted meter prints its
records under the summary line — this is a Qundis heat cost allocator test
telegram from wmbusmeters' simulation set, the one the operator doc shows:

```text
wmbus: listening on rtlsdr[00000001] at 868.950 MHz, 1600000 S/s (Ctrl-C to stop)
2026-10-10 14:03:12.418  C1-A   -6.9 dBFS  QDS 78563412  heat cost allocator (v35)
    heat cost allocation 127
    heat cost allocation [storage 1] 145
    date [storage 1] 2018-12-31
    …
    date time 2019-02-20 11:32
```

Storage 0 is the current value; higher storage numbers are historic ones.
An encrypted meter is one line, and that line is all there is (the meter ID
is masked here; the tool prints all eight digits):

```text
T1-A  -11.7 dBFS  BMT ########  water (v13)  encrypted: AES-128-CBC with IV (mode 5), access 121
```

Between telegrams the progress line shows the receiver is alive:

```text
wmbus: listening 5m0s (14:02:39): 3 frames (1 CRC-valid, 2 CRC-failed), 5 access codes in 41 bursts, 0 dropped chunks, carrier -50.1 kHz (-58 ppm) — corrected for, but -ppm ±58 would centre the tuner (keep the sign that shrinks it)
```

The first time is **stream time** — how much IQ the decoder has worked
through — and the bracket is the time of day; the two drift apart whenever
IQ is dropped. **Bursts** counts every rise out of the channel's noise, a
meter or anything else: the sign that the receiver is hearing the channel.
**Dropped chunks must stay at 0.** **Carrier** appears once a telegram has
decoded; past about 20 ppm the line adds the `-ppm` hint — a suggestion,
since the offset is already corrected for. On stop the line reads
`wmbus: stopped after …`.

**Reading the level.** The dBFS on each line is a level, not a quality. On
the reporter's dongle at `-gain 100` the noise in the 200 kHz channel sat
near −44 dBFS, so a strong-looking −28 dBFS telegram had 13 dB of
[signal-to-noise]({{ '/reference/signal-to-noise-ratio/' | relative_url }}),
and the failures clustered at 8 to 16 dB. That is why the receiver now
decides each chip by comparing the energy at the two FSK tones instead of
FM-demodulating first: across all five walks it hears 350 different meters
instead of 296
([what a dBFS figure does and does not mean]({{ '/blog/tutorials/analog-edge-02-dbfs/' | relative_url }})).

## When it doesn't work

| Symptom | Likely cause | Fix |
|---|---|---|
| Bursts climb for many minutes, no frames | nothing in range is a T1 / C1 meter, or the meters here send rarely | Walk closer and wait: intervals run from seconds to hours |
| “CRC FAILED (blocks 3,4 of 4; header OK)” | a weak, clipped or interfered telegram: sender right, payload damaged | Compare the level with your noise, adjust `-gain`, get closer. C-mode telegrams already get a repair of up to three bits per block (`CRC repaired: N bits`) |
| “CRC FAILED (block 1 of 4; block 1 is the header, so the sender shown is unreliable)” | the header block itself failed | Ignore the sender: a manufacturer shown as hex, such as `0x097D`, is not a real code |
| “— cut by dropped IQ (the SDR dropped chunks during it), not a weak meter” | the device dropped IQ in the middle of that telegram | Not an RF problem: fix the drops (next row) |
| “WARNING — the SDR has dropped … only 30s of IQ (13% of real time)” | the phone starved the process: screen off, Termux in the background | Keep the screen on, run `termux-wake-lock` first, exempt Termux from battery optimisation. The warning rings the bell; Termux vibrates on it |
| “lost the SDR at …”, later “the dongle is not on the USB bus at all” | the dongle left the bus; a phone's OTG port stays off after a disconnect | Unplug and replug: the run carries on from there. `-reconnect=false` stops with the reason instead |
| Ctrl-C does nothing in Termux under su | the process has no controlling terminal (it says so at start) | Type `q` and Enter, or give `-seconds` |
| A capture reports too many seconds and 0 frames | a headerless file decoded at the wrong rate | Keep the `.metadata.json` beside the capture, or pass `-rate` |

Two of those rows came straight from the field, and a number settled each.
On one walk the dropped chunks multiplied by the chunk length equalled the
run's wall time minus its stream time — USB had delivered everything and
the process had not been scheduled: about 10 of 21 minutes went unheard.
On another, the reconnect worked once (the dongle was back after 8 s), then
the dongle left the bus for good and the run polled for 19 minutes until a
physical replug. Neither was a decoder defect:
[overruns are a downstream signal]({{ '/blog/tutorials/field-notebook-07-overruns-host-drops/' | relative_url }})
here exactly as on a trunking rig. One case has no log line: a strong
transmitter that stays on for seconds elsewhere in the span can become the
burst detector's noise floor and hide a weak telegram; `-no-gate`
demodulates everything, as releases up to v1.4.0 did.

### How this recipe shapes operator practice

- **Measure margin, not level.** Know your dongle's noise in the channel
  before calling a telegram strong.
- **Save the air, not the story.** `-save-frames -all` turned five walks
  into 2,005 replayable telegrams, the yield check every receiver change
  since has been scored against — the
  [capture discipline]({{ '/blog/tutorials/analog-edge-10-capture-discipline/' | relative_url }})
  rule at pocket scale.
- **Do the arithmetic before blaming RF.** Wall time against stream time
  separates a starved process from a weak meter.
- **Treat meter IDs as someone's data.** Mask them in anything you post.

## Variations

- **Capture first, decode later.** Recording has no real-time limit, so a
  slow host loses nothing: run `capture` with `-format u8 -out meters.cu8`,
  then `wmbus -in meters.cu8` (rate, format and centre come from the
  `.metadata.json`). FLAC is smaller but cannot store rates above
  1,048,575 S/s, hence `-sample-rate 1024000 -format flac`.
- **Feed a script.** `-json` prints one JSON object per telegram on
  standard output and nothing else; `-json -log meters.json` keeps a file
  whose every line parses, progress lines included.
- **Score a receiver yourself.** With the source tree,
  `GT_WMBUS_SNIPPETS=<dir> go test ./internal/radio/wmbus -run RealAirSnippets -v`
  replays a snippet directory in the order it was heard — the harness
  behind the yield figures in this post.

## Where this goes next

The dongle goes back in the drawer and a different radio comes out.
[Part 13]({{ '/blog/tutorials/cookbook-volume-2-13-pluto-native/' | relative_url }})
puts an ADALM-Pluto or PlutoSky on the network and has GopherTrunk speak the
board's own `iiod` protocol — no SoapySDR and no libiio on the host — with a
read-only probe to run first.

## FAQ

**Can an RTL-SDR decode Wireless M-Bus water and heat meters?**
Yes. `gophertrunk wmbus` decodes modes T1 and C1 at 868.95 MHz from a local
RTL-SDR, an `rtl_tcp` server or a capture file, and prints each telegram's
manufacturer, meter ID, version and device type. Readings appear only for
meters that send them unencrypted.

**Can GopherTrunk read encrypted wM-Bus meters?**
No. Most OMS meters encrypt their readings with AES-128 (security mode 5
or 7) under a per-meter key held by the metering company. GopherTrunk prints
the sender, device type and security mode for those telegrams, attempts no
decryption, and has no key option.

**What gain should I use for wM-Bus at 868 MHz?**
Start at `-gain 200` (20 dB — the flag is in tenths of a dB), step down
while `gophertrunk capture` reports clipping, and step up only if telegrams
are weak. With a preamplifier start between `-gain 0` and `-gain 100`.
Judge by margin over the channel's noise, not by dBFS alone.

**What does 'cut by dropped IQ' mean in gophertrunk wmbus?**
The SDR dropped IQ chunks while that telegram was on the air, so the
decoder never received the rest of it and its CRC failed. It says nothing
about the meter's signal, and replaying the saved snippet cannot recover
it. The cure is on the host — usually a phone throttling the process.

**Is the wM-Bus decoder verified on air?**
Partly. It matches rtl-wmbus on that project's four
real-air captures and wmbusmeters on its test telegrams, and a reporter's
phone walks decoded real Qundis and Diehl meters. The tone-comparing
detector and the burst gate were measured on those walks' saved snippets
and have not yet been run live on a phone.

## Series navigation

**Part 12 of 14** · ←
[Part 11: The Band Survey — gophertrunk power]({{ '/blog/tutorials/cookbook-volume-2-11-band-survey/' | relative_url }})
· Next →
[Part 13: Pluto Without Soapy — The Native IIO Driver]({{ '/blog/tutorials/cookbook-volume-2-13-pluto-native/' | relative_url }})
