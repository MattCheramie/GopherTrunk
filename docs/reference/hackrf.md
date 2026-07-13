---
slug: hackrf
title: HackRF One
entry_type: hardware
category: sdr-devices
description: HackRF One is an open-source wideband half-duplex software-defined radio transceiver covering 1 MHz to 6 GHz with up to 20 MHz of bandwidth and transmit capability.
keywords: HackRF One, Great Scott Gadgets, wideband SDR, transceiver, 1 MHz 6 GHz, transmit
aka: [HackRF, HackRF One]
autolink: true
infobox:
  - { label: Type, value: Wideband SDR transceiver }
  - { label: Vendor, value: Great Scott Gadgets }
  - { label: Range, value: 1 MHz – 6 GHz }
  - { label: Bandwidth, value: up to ~20 MHz }
  - { label: TX, value: Yes (half-duplex) }
see_also: [rtl-sdr, airspy, software-defined-radio]
related_lessons:
  - { title: "SDR hardware — RTL-SDR, HackRF, Airspy", url: /learn/rf-sdr/sdr-hardware/ }
related_reading:
  - { title: "RF Front End, Part 11: HackRF One", url: /blog/deep-dives/rf-front-end-11-hackrf-one/ }
cite_urls:
  - https://en.wikipedia.org/wiki/HackRF_One
---

**HackRF One** is an open-source, wideband, half-duplex
[software-defined radio](/reference/software-defined-radio/) transceiver from Great Scott
Gadgets, covering **1 MHz to 6 GHz** with up to ~20 MHz bandwidth and the ability to
**transmit**.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 110" role="img" aria-label="A frequency coverage bar for HackRF One (~1 MHz–6 GHz) on an axis from about 0 to 6 gigahertz." xmlns="http://www.w3.org/2000/svg">
  <line x1="30" y1="70" x2="430" y2="70" stroke="currentColor" stroke-opacity="0.4"/>
  <g font-size="8" fill="currentColor" text-anchor="middle"><text x="30" y="86">0</text><text x="163" y="86">2 GHz</text><text x="296" y="86">4 GHz</text><text x="430" y="86">6 GHz</text></g>
  <rect x="30" y="40" width="400" height="20" rx="3" fill="currentColor" fill-opacity="0.25" stroke="currentColor" stroke-width="1.3"/>
  <text x="230" y="28" text-anchor="middle" font-size="10" fill="currentColor">HackRF One (~1 MHz–6 GHz) coverage</text>
</svg>
<figcaption>HackRF spans a huge range and can transmit, but with lower dynamic range — overkill for scanning.</figcaption>
</figure>

## Overview

Its huge range and TX capability make it popular for experimentation, but it uses 8-bit
sampling (less dynamic range than [Airspy](/reference/airspy/)) and transmit is
irrelevant to receive-only scanning.

## Relevance to SDR

For decoding trunked voice, HackRF is overkill; an [RTL-SDR](/reference/rtl-sdr/) or
Airspy is usually the better fit, but GopherTrunk can use it as a receiver.

## Setup (Linux)

GopherTrunk talks to the HackRF directly over USB — no `libhackrf` or
`hackrf-tools` install required. The one host-side step is a udev rule so
non-root processes can open the device node. Without it, `gophertrunk sdr list`
still shows the device (it only reads `sysfs`), but opening it — `sdr list
--probe`, `sdr doctor`, and normal capture — fails with `permission denied` on
`/dev/bus/usb/…`.

```sh
sudo tee /etc/udev/rules.d/20-hackrf.rules <<'EOF'
# HackRF One / Jawbreaker / Rad1o (Great Scott Gadgets, VID 0x1d50)
SUBSYSTEM=="usb", ATTRS{idVendor}=="1d50", ATTRS{idProduct}=="6089", MODE="0666"
SUBSYSTEM=="usb", ATTRS{idVendor}=="1d50", ATTRS{idProduct}=="604b", MODE="0666"
SUBSYSTEM=="usb", ATTRS{idVendor}=="1d50", ATTRS{idProduct}=="cc15", MODE="0666"
EOF
sudo udevadm control --reload
sudo udevadm trigger
```

Unplug and re-plug the HackRF once so the rule applies to a freshly-enumerated
device. Swap `MODE="0666"` for `MODE="0660", GROUP="plugdev"` if you'd rather
scope access. Then confirm:

```sh
gophertrunk sdr list --probe
```

## Setup (Windows)

GopherTrunk drives the HackRF through the in-box **WinUSB** driver — no
`libhackrf`/`hackrf-tools` and no [SoapySDR](/reference/soapysdr/) install
required. A HackRF One advertises a Microsoft OS (WCID) descriptor, so Windows
10/11 usually binds WinUSB automatically and it just works: plug it in and run

```powershell
gophertrunk sdr list --probe
gophertrunk sdr doctor
```

`sdr doctor` prints a row per known SDR (RTL-SDR **and** HackRF) showing which
function driver is bound and whether it's the expected WinUSB. If the HackRF row
shows `STATUS BAD` — or `sdr list --probe` reports `WinUsb_Initialize failed` —
a different driver is bound. Fix it with [Zadig](/reference/zadig/):

1. Run Zadig, then **Options → List All Devices**.
2. Select the **HackRF One** entry (Interface 0).
3. Choose **WinUSB** as the target driver and click **Replace Driver**.

If `--probe` fails with an access-denied error, another program (SDR#, GQRX,
SDRangel, the HackRF tools) already holds the device — close it and retry.

## Using the HackRF with the daemon

`gophertrunk sdr list` only enumerates devices; the daemon (and the web UI
**Devices** panel) opens the SDRs named in your `config.yaml`. If the panel says
*"No SDRs known to the daemon"*, add the HackRF under `sdr.devices`:

```yaml
sdr:
  sample_rate: 8_000_000
  devices:
    - serial: "0000000000000000457863c8284a625f"   # from `gophertrunk sdr list`
      role: control          # or voice / wideband
      gain: auto             # or tenths-of-dB, e.g. "320" for 32.0 dB
      bias_tee: false
```

Copy the serial straight from `gophertrunk sdr list`. The match is
case-insensitive and tolerant of a partial serial — a distinctive tail like
`457863c8284a625f`, or a value captured from an older build that displayed only
the `0000000000000000` prefix, still binds as long as it's unambiguous (the
daemon logs when it matches by a partial serial). With multiple HackRFs, use the
full serial so each entry pins exactly one device.

> **Serial number.** A HackRF reports a full 32-hex-digit `part_id + serial_no`
> string — the leading 16 digits are a constant prefix (commonly all zeros) and
> the trailing digits are the unique part. `gophertrunk sdr list` prints the
> whole string, matching `hackrf_info`'s "Serial number".

## Sources

[^wiki]: [HackRF One](https://en.wikipedia.org/wiki/HackRF_One) — Wikipedia, on the HackRF One wideband half-duplex transceiver and its specifications.
