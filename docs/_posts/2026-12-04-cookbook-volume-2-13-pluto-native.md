---
title: "The Operator's Cookbook Vol. 2, Part 13: Pluto Without Soapy — The Native IIO Driver"
description: "A complete GopherTrunk recipe for an ADALM-Pluto or PlutoSky with no SoapySDR or libiio on the host: the sdr.iio config that speaks the board's own iiod protocol on TCP 30431, the read-only iio probe, the -iio flag on power and capture, the AD9361's rate limits, AGC versus fixed gain, and what is confirmed on air so far."
category: tutorials
keywords: plutosdr without soapysdr, adalm-pluto iiod driver, plutosky sdr scanner, pluto sdr libiio alternative, ad9361 sample rate limits, pluto sdr agc squelch, soapyplutosdr full scale 2048, iiod tcp port 30431, gophertrunk pluto
tags: [operator-cookbook, cookbook-volume-2, plutosdr, iio, ad9361, network-sdr]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "The Operator's Cookbook, Volume 2"
series_part: 13
---

*Part 13 of **The Operator's Cookbook, Volume 2**, a 14-part tutorial series
of complete, copy-paste GopherTrunk builds — one working rig per part — built
on what landed after Volume 1.
[Part 12]({{ '/blog/tutorials/cookbook-volume-2-12-meter-walk/' | relative_url }})
walked an RTL-SDR past a block of flats. This part changes the radio: an
**ADALM-Pluto or PlutoSky**, reached the way Volume 1's
[remote radios]({{ '/blog/tutorials/operator-cookbook-08-remote-radios/' | relative_url }})
were — over the network — but without the server that recipe needed. The
board already runs one. GopherTrunk now speaks that daemon's protocol
itself, so the host needs no SoapySDR, no SoapyPlutoSDR module and no
libiio: an address in `config.yaml` is the whole install.*

> **TL;DR:** List the board under `sdr.iio` by address and GopherTrunk
> dials **iiod** — the libiio daemon running on the Pluto itself — on TCP
> 30431, over the USB-network link (`192.168.2.1`) or the LAN, and drives
> the `ad9361-phy` and `cf-ad9361-lpc` devices directly. Run
> `gophertrunk iio probe -addr 192.168.2.1` first: it changes nothing on the
> board and says whether `sdr.iio` can use it. The one-shot tools read no
> config file, so they take `-iio <addr>`. Samples are 12-bit (full scale
> 2048); rates run from about 260.4 kS/s to 61.44 MS/s, with ×8 decimation
> on the board below 25 MHz/12; `gain: "auto"` is the AD9361's AGC, under
> which an absolute `squelch_dbfs` screens nothing. Status: the probe and a
> sample read are confirmed on a reporter's PlutoSky; the daemon's decode
> path is not yet confirmed on air.

**Key takeaways**

- **Nothing to build on the host.** The board serves `iiod` on TCP 30431;
  the driver is a pure-Go client for its text protocol, pinned to libiio
  v0.25's source and SoapyPlutoSDR's attribute calls.
- **Probe before you configure.** `gophertrunk iio probe` reads the board's
  device list, its current settings and one buffer of samples, writes
  nothing, and exits non-zero when `sdr.iio` would refuse the board.
- **An AGC turns a level squelch into a no-op.** Under `gain: "auto"` the
  AD9361 raises noise to its own target level, so a scan list survives only
  because FM channels also need a carrier — the noise squelch.
- **"Confirmed" is a narrow word here.** Probe, settings read-back and a
  65,536-sample read at the right level: yes, on a real PlutoSky. Tune,
  stream and lock a control channel in the daemon: still an on-air gate.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| Native driver | iiod over TCP 30431, nothing installed on the host | `sdr.iio[]` |
| Health check | read-only board report plus one buffer of samples | `gophertrunk iio probe -addr 192.168.2.1` |
| Sweep or capture | the one-shot tools name the board by flag, not by config | `power -iio`, `capture -iio` |
| Gain | fixed, in tenths of a dB, or the AD9361's AGC | `gain: "400"` or `gain: "auto"` |
| Analog bandwidth | follows the sample rate unless pinned | `rf_bandwidth_hz` |
| RX input | selects the board's receive port | `rf_port` |
| The Soapy route | the same board behind a SoapySDRServer ([SoapyRemote]({{ '/reference/soapyremote/' | relative_url }})) | `sdr.soapy_remote[]` |

## In this post

- **What you're building** — a Pluto in the pool, and the two ways to reach it.
- **The shopping list** — the board, a cable, and no packages.
- **The config** — `sdr.iio`, a scan list, and the three one-shot commands.
- **First run — what healthy looks like** — the probe report, then the daemon.
- **When it doesn't work** — dial errors, the rate floor, a squelch under AGC.
- **Variations** — a trunked build, the SoapyRemote route, a band sweep.

## What you're building

A [PlutoSDR]({{ '/reference/plutosdr/' | relative_url }})-class radio — the
ADALM-Pluto, or a Zynq + AD9361/AD9363 board running Pluto-compatible
firmware such as the PlutoSky — mounted in GopherTrunk's SDR pool as a
virtual tuner with a serial and a role, like every remote radio in Volume 1.
The difference is what sits between the two. A Pluto runs **iiod**, the
network daemon of [libiio]({{ '/reference/libiio/' | relative_url }}), on
the board, on TCP port 30431, over Ethernet or its USB-network link. The
`iio` driver connects to it, reads the board's device list, and from then
on tunes, sets the rate and gain, and streams samples with a handful of
text commands. Nothing has to be installed or built on the host.

The route this replaces still works and is the second half of the figure:
a `SoapySDRServer` with the SoapyPlutoSDR module, which has to be built
against the host's SoapySDR, libiio and libad9361. The reporter whose
PlutoSky started this
([#1329](https://github.com/MattCheramie/GopherTrunk/issues/1329)) could not
build it, which is why the native driver exists.

<figure class="lab-figure">
<svg viewBox="0 0 680 246" width="680" height="246" role="img" aria-label="Two ways to reach a Pluto-class board. In the native route GopherTrunk's pure-Go iio driver opens two TCP connections to port 30431 on the board, one for control commands and one for the sample stream, and iiod on the board drives the ad9361-phy and cf-ad9361-lpc devices. In the SoapySDR route GopherTrunk's soapyremote driver connects on TCP 55132 to a SoapySDRServer with the SoapyPlutoSDR module and libiio built on some host, which in turn reaches the same board's iiod.">
  <text x="8" y="16" fill="var(--accent)" font-size="10">native — sdr.iio</text>
  <rect x="8" y="24" width="184" height="78" rx="4" fill="none" stroke="var(--accent)"/>
  <text x="100" y="46" text-anchor="middle" fill="var(--accent)" font-size="10">GopherTrunk</text>
  <text x="100" y="62" text-anchor="middle" fill="currentColor" font-size="9">iio driver (pure Go)</text>
  <text x="100" y="78" text-anchor="middle" fill="var(--fg-muted)" font-size="9">no SoapySDR, no libiio</text>
  <line x1="192" y1="46" x2="430" y2="46" stroke="var(--accent)"/>
  <text x="311" y="40" text-anchor="middle" fill="var(--fg-muted)" font-size="9">control: VERSION, PRINT, READ, WRITE</text>
  <text x="311" y="67" text-anchor="middle" fill="var(--accent)" font-size="10">TCP 30431</text>
  <line x1="192" y1="80" x2="430" y2="80" stroke="var(--accent)"/>
  <text x="311" y="94" text-anchor="middle" fill="var(--fg-muted)" font-size="9">stream: OPEN, READBUF — a second socket</text>
  <rect x="430" y="24" width="242" height="78" rx="4" fill="none" stroke="currentColor"/>
  <text x="551" y="46" text-anchor="middle" fill="currentColor" font-size="10">Pluto / PlutoSky — iiod</text>
  <text x="551" y="62" text-anchor="middle" fill="var(--fg-muted)" font-size="9">ad9361-phy: LO, rate, gain, bandwidth</text>
  <text x="551" y="78" text-anchor="middle" fill="var(--fg-muted)" font-size="9">cf-ad9361-lpc: 12-bit I/Q, full scale 2048</text>
  <text x="8" y="140" fill="var(--fg-muted)" font-size="10">through SoapySDR — sdr.soapy_remote</text>
  <rect x="8" y="148" width="150" height="78" rx="4" fill="none" stroke="currentColor"/>
  <text x="83" y="178" text-anchor="middle" fill="currentColor" font-size="10">GopherTrunk</text>
  <text x="83" y="194" text-anchor="middle" fill="var(--fg-muted)" font-size="9">soapyremote driver</text>
  <line x1="158" y1="187" x2="250" y2="187" stroke="currentColor"/>
  <text x="204" y="180" text-anchor="middle" fill="var(--fg-muted)" font-size="9">TCP 55132</text>
  <rect x="250" y="148" width="190" height="78" rx="4" fill="none" stroke="var(--fg-muted)" stroke-dasharray="4 3"/>
  <text x="345" y="172" text-anchor="middle" fill="currentColor" font-size="10">SoapySDRServer</text>
  <text x="345" y="188" text-anchor="middle" fill="var(--fg-muted)" font-size="9">+ SoapyPlutoSDR + libiio</text>
  <text x="345" y="204" text-anchor="middle" fill="var(--fg-muted)" font-size="9">built on some host</text>
  <line x1="440" y1="187" x2="530" y2="187" stroke="currentColor"/>
  <text x="485" y="180" text-anchor="middle" fill="var(--fg-muted)" font-size="9">uri=ip:… or usb:…</text>
  <rect x="530" y="148" width="142" height="78" rx="4" fill="none" stroke="currentColor"/>
  <text x="601" y="178" text-anchor="middle" fill="currentColor" font-size="10">the same board</text>
  <text x="601" y="194" text-anchor="middle" fill="var(--fg-muted)" font-size="9">iiod</text>
</svg>
<figcaption>Both routes end at the same iiod on the board; the native one removes the middle box, and with it everything that had to be compiled.</figcaption>
</figure>

**Status, plainly.** On a reporter's PlutoSky (Pluto v0.38 firmware and
tezuka-v0.3.24) the probe connects, reads the device list and the current
settings, and reads one buffer of samples at the right level — over the USB
link and over the LAN, from Linux with and without libiio and from Termux
over WLAN. The board's own device listing is the driver's test fixture, and
every attribute the driver writes is checked against it. The daemon's
decode path — tune, stream, lock a control channel — is reference-pinned
and synthetic-tested, and **not yet confirmed on air**. The operator
reference is the
[hardware guide's Pluto section]({{ '/hardware.html#adalm-pluto-plutosky-and-other-ad936x-boards' | relative_url }}).

## The shopping list

| Item | Price (rough) | Notes |
|---|---|---|
| ADALM-Pluto, PlutoSky or another Pluto-firmware AD936x board | varies | must expose the standard devices; the probe tells you |
| The link | $0 | the board's USB cable is a network link (the board sits at 192.168.2.1), or Ethernet / WLAN where the board has it |
| Antenna for your band | varies | the AD9361 tunes far wider than an RTL-SDR; the antenna does not |
| Host | $0 | anything GopherTrunk runs on — nothing else to install |

## The config

A complete scan-list rig with the Pluto as the scanner's radio — the shape
of the reporter's build, a PMR446 scan list:

```yaml
log:
  level: info

storage:
  path: "../data/calls.db"

recordings:
  dir: "../recordings"

sdr:
  sample_rate: 2_400_000        # above 25 MHz/12: no x8 decimation on the board
  iio:
    - addr: "192.168.2.1"       # USB-network default; or a LAN host, host:port, "ip:..."
      serial: "plutosky"
      role: voice               # the conventional scanner drives a voice SDR
      gain: "400"               # 40 dB fixed; "auto" = the AD9361 AGC (slow_attack)
      rf_bandwidth_hz: 0        # 0 = follow sdr.sample_rate
      rf_port: ""               # e.g. "A_BALANCED"; empty = the board's setting
      connect_timeout_ms: 5000

scanner:
  conventional:
    - label: "PMR446 ch 1"
      frequency_hz: 446_006_250
      mode: nfm
      squelch_dbfs: -60
      # noise_squelch: true     # on by default for fm / nfm
      hangtime_ms: 1500
      talkgroup_id: 2147500001
    - label: "PMR446 ch 2"
      frequency_hz: 446_018_750
      mode: nfm
      squelch_dbfs: -60
      hangtime_ms: 1500
      talkgroup_id: 2147500002
```

And the three commands that never read that file:

```bash
gophertrunk iio probe -addr 192.168.2.1 -out board.xml
gophertrunk power -iio 192.168.2.1 -f 88M:108M:10k -gain 400 -out fm.csv
gophertrunk capture -iio 192.168.2.1 -freq 446100000 -seconds 10 -gain 400 -out pmr.cfile
```

Decisions worth understanding. **`role`** is `control`, `voice`, `auto` or
`dedicated`; the scan list takes a voice device, exactly as in
[Volume 1's analog recipe]({{ '/blog/tutorials/operator-cookbook-06-analog-fm-tone-out/' | relative_url }}).
**The rate has a floor and a knee.** Without a FIR filter loaded the AD9361
runs from 25 MHz/96 (about 260.4 kS/s) to 61.44 MS/s, and config validation
rejects a lower `sdr.sample_rate`. Below 25 MHz/12 (about 2.083 MS/s) the
driver sets the PHY to eight times the rate and lets the board's RX core
decimate by 8, as SoapyPlutoSDR does. The stream is 16 bits each for I and
Q, so the Volume 1 network arithmetic applies unchanged: 2.4 MS/s is about
77 Mbit/s, and a board on USB 2.0 cannot sustain the top of that range.

**`gain` is the decision that matters.** A number is tenths of a dB over
the AD9361's roughly 0–73 dB range, written as manual gain in whole dB.
`"auto"` selects the chip's `slow_attack` AGC, which raises whatever it
receives — noise included — to its own target level. **There is no `ppm`
key**: the AD9361 has no trim through the attributes this driver drives, so
read the decoder's measured carrier offset instead (on a P25 channel the
[Tuning panel]({{ '/tuning.html' | relative_url }}) plots it). And like
every network radio in Volume 1, the link is plaintext — trusted networks
only.

## First run — what healthy looks like

Probe first. The report is made to be pasted into an issue; this is its
shape, with the identification block from the reporter's real board:

```text
iiod 0.25.… at 192.168.2.1:30431
  hw_model:          Analog Devices PlutoSDR Rev.C (Z7020-AD9361)
  fw_version:        v0.38
  ad9361-phy,model:  ad9361
  hw_serial:         …
  devices:
    iio:device0   ad9361-phy               11 channels
    …
    iio:device3   cf-ad9361-lpc            4 channels  scan: voltage0=le:S12/16>>0 voltage1=le:S12/16>>0 …
usable by sdr.iio: RX iio:device3, format le:S12/16>>0, full scale 2048
  RX LO frequency:   …
  PHY sample rate:   …
  RF bandwidth:      …
  gain mode:         …
  hardware gain:     …
stream: 65536 samples, RMS … dBFS, peak … dBFS, 0.00% at the ADC rail
context XML (… bytes) written to board.xml
```

Three lines carry the verdict: `usable by sdr.iio` (or
`NOT USABLE by sdr.iio:` and the reason), the `gain mode` the board is
currently in, and the `stream:` line — RMS and peak in dBFS and the share
of values at the ADC rail. The reporter's reads came back at −33 to
−35 dBFS RMS with no sample at the rail.

Then the daemon. The board announces itself once at open, and the scanner
logs what its squelch measured each time a channel opens:

```text
INF iio (Pluto) boards mounted count=1
INF iio: connected addr=192.168.2.1:30431 iiod_version=… hw_model="Analog Devices PlutoSDR Rev.C (Z7020-AD9361)" fw_version=v0.38 rx_device=iio:device3 sample_format=le:S12/16>>0 full_scale=2048 freq_range_hz=…
INF conv: scanner LO offset tuning serial=plutosky lo_offset_hz=… mode=… sample_rate_hz=2400000
INF conv: squelch opened freq_hz=446006250 label="PMR446 ch 1" mode=nfm level_dbfs=… squelch_dbfs=-60 quieting_db=…
INF synthetic call started device=plutosky grant=...
```

`full_scale=2048` is the line to check: levels in dBFS, `squelch_dbfs` and
the overload checks are all normalised by it. `quieting_db` is the noise
squelch's reading — about 0 on noise, 3 where the squelch opens, 15 and up
on a usable signal — and is also in `GET /api/v1/scanner`.

**Why that last number exists.** The reporter's PlutoSky scan list opened
a call 100 ms after start-up and recorded noise for 33 and 41 minutes. One
log showed the AGC holding −12 dBFS across 4.8 MHz — about −37 dBFS in a
±8 kHz channel, far above a `squelch_dbfs` of −60. FM channels now also
require a carrier that quiets the discriminator, which receiver noise never
does at any level;
[Squelch Part 3]({{ '/blog/deep-dives/squelch-03-noise-squelch/' | relative_url }})
is the full account. Under AGC the scanner says so once per channel:

```text
INF conv: channel is above squelch_dbfs with no carrier on it (noise, or a signal that is not FM); the noise squelch is keeping it shut. squelch_dbfs is below the noise floor at this gain, or the gain is automatic, so the level is not screening this channel freq_hz=… label=… level_dbfs=… squelch_dbfs=-60 quieting_db=…
```

That fix has not been run on a PlutoSky either.

## When it doesn't work

| Symptom | Likely cause | Fix |
|---|---|---|
| “iio: dial … nothing answers at … at the network level” | no route to the board: cable, USB link, or the host has no address on the board's subnet | Check that `ping` reaches it; USB networking puts the board at 192.168.2.1. A board that never answers ping has a link or firmware problem, before iiod and GopherTrunk |
| “… answered but nothing is listening on the iiod port” | the board is up, iiod is not, or the port is not 30431 | Check iiod on the board, or give `host:port` in `addr` |
| “no answer from … within the timeout” | a firewall dropping TCP 30431, or an unreachable board | If ping works, look at the firewall; raise `connect_timeout_ms` on a slow link |
| “NOT USABLE by sdr.iio: not a Pluto-style AD936x board” | the board lacks the two standard devices | The line lists the devices it does have; attach the XML from `-out` to a report |
| Config rejected: “sdr.sample_rate … is below the AD9361's … Hz floor without a FIR filter” | a rate under 25 MHz/96 | Raise `sdr.sample_rate` to about 260.4 kS/s or more |
| A one-shot tool says “no SDRs found” though the Pluto is in config.yaml | the power and capture tools read no config file | Pass `-iio <addr>`; the error now names that flag for each network radio the config lists |
| Scan list opens calls on empty channels | the AGC lifts noise over the level threshold, and the noise squelch is off or the build predates it | Leave `noise_squelch` on (the default) and use a fixed `gain` |
| “iio: IQ stream ended” | the stream socket closed: link, power, or the board | Check the link; the line carries how long it ran and the error |
| A power sweep keeps a spike at each hop's centre | the tuner's offset moved during the hop, as an AGC stepping its gain does | Sweep with a fixed `-gain`: [Part 11]({{ '/blog/tutorials/cookbook-volume-2-11-band-survey/' | relative_url }}) |

Two smaller things real boards taught the driver. The AD9361's
kernel driver reads a sample rate back with its fraction cut off — 4,800,000
comes back as 4,799,999 — so a read-back one hertz off the written rate is
now taken as the written rate; before that, every hop of a power sweep
started a hertz further off. And the first test fixture, reconstructed from
a published layout, numbered the devices differently from the real board;
the driver binds by name and was unaffected, but the fake server was not —
the [self-consistent trap]({{ '/blog/solution-postmortem/from-the-issue-tracker-20-self-consistent-trap/' | relative_url }})
in a new place, closed by committing the board's real listing.

### How this recipe shapes operator practice

- **Ask the board, read-only, before changing anything.** A probe that
  writes nothing cannot leave the board in a state you did not choose.
- **Fix the gain when you want to read a level.** An AGC makes every dBFS
  figure a statement about the AGC — the
  [gain-staging]({{ '/blog/tutorials/analog-edge-03-gain-staging/' | relative_url }})
  rule, with a chip enforcing it.
- **Config is for the daemon; flags are for tools.** `power`, `capture`
  and `wmbus` never read `config.yaml`.

## Variations

**A trunked build.** Give the Pluto `role: control` and add a second radio
for voice — the Pluto hunts the control channel, the dongle follows the
grants:

```yaml
sdr:
  sample_rate: 2_400_000
  devices:
    - serial: "00000002"
      role: voice
      gain: "auto"
  iio:
    - addr: "192.168.2.1"
      serial: "plutosky"
      role: control
      gain: "400"

trunking:
  systems:
    - name: "Metro-P25"
      protocol: p25
      control_channels: [851_012_500]
```

- **The SoapyRemote route.** If you already run SoapySDR, point
  `sdr.soapy_remote` at a server with `driver: "plutosdr"` and
  `args: "uri=ip:192.168.2.1"`. A Pluto streams 12-bit values in 16-bit
  words, full scale 2048; builds before #1329 divided by 32768 and read
  every Pluto signal 24 dB low. The `soapyremote: connected` line now
  prints `cs16_full_scale`, which should read 2048. Not confirmed on air
  with a Pluto either.
- **A band sweep.** A reporter's PlutoSky sweep is where Part 11's DC-line
  work came from: the tuner's DC line survived in its busiest hops — an AGC
  stepping its gain is the inferred cause — so sweep with a fixed `-gain`
  ([power sweep reference]({{ '/power-sweep.html' | relative_url }})).

## Where this goes next

One part left.
[Part 14]({{ '/blog/tutorials/cookbook-volume-2-14-pi4-budget-kitchen-sink/' | relative_url }})
sizes a Raspberry Pi 4 against a reporter's real drop counts — an Airspy
Mini and several TETRA cells — and then folds every Volume 2 recipe,
this one included, into one annotated kitchen-sink config.

## FAQ

**Do I need SoapySDR or libiio to use a PlutoSDR with GopherTrunk?**
No. List the board under `sdr.iio` and GopherTrunk talks to iiod, the
daemon already running on the Pluto, over TCP 30431. Nothing is installed
or built on the host. The SoapyRemote route still exists if you already
run a SoapySDRServer with the SoapyPlutoSDR module.

**How do I check that GopherTrunk can talk to my Pluto?**
Run `gophertrunk iio probe -addr 192.168.2.1 -out board.xml`. It prints the
iiod version, the board model and firmware, whether `sdr.iio` can use the
board, the current tuning, rate and gain, and the level of one buffer of
samples. It writes nothing to the board and exits non-zero on failure.

**Why does gophertrunk power say no SDRs found when my Pluto is in config.yaml?**
Because `power` and `capture` read no config file; an `sdr.iio` entry is
for the daemon only. Pass the board on the command line with `-iio`. When
the daemon's config lists network radios, the error names the flag for
each of them.

**What sample rates can a Pluto run with the native driver?**
From 25 MHz/96, about 260.4 kS/s, to the AD9361's 61.44 MS/s, without a FIR
filter loaded. Below about 2.083 MS/s the board's RX core decimates by 8.
What the link can carry is separate: four bytes per complex sample.

**Is the native Pluto driver verified on air?**
Partly. The read-only probe, the settings read-back and a sample read at
the right level are confirmed on a reporter's PlutoSky over USB and LAN.
The daemon path — tune, stream, lock a control channel — is reference-pinned
and synthetic-tested but not yet confirmed on air.

## Series navigation

**Part 13 of 14** · ←
[Part 12: The Meter Walk — Wireless M-Bus at 868 MHz]({{ '/blog/tutorials/cookbook-volume-2-12-meter-walk/' | relative_url }})
· Next →
[Part 14: The Pi 4 Budget — and the Kitchen Sink, Revisited]({{ '/blog/tutorials/cookbook-volume-2-14-pi4-budget-kitchen-sink/' | relative_url }})
