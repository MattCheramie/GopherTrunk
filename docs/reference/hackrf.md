---
slug: hackrf
title: HackRF One
entry_type: hardware
category: sdr-devices
description: HackRF One is an open-source wideband half-duplex software-defined radio transceiver covering 1 MHz to 6 GHz with up to 20 MHz of bandwidth and transmit capability.
keywords: HackRF One, Great Scott Gadgets, wideband SDR, transceiver, 1 MHz 6 GHz, transmit
aka: [HackRF, HackRF One]
autolink: true
affiliate: true
product:
  name: "HackRF One"
  brand: Great Scott Gadgets
  category: Software-defined radio
  lowPrice: "140"
  highPrice: "170"
  url: https://www.amazon.com/dp/B0BKH7Z2NJ?tag=gophertrunk-20
infobox:
  - { label: Type, value: Wideband SDR transceiver }
  - { label: Vendor, value: Great Scott Gadgets }
  - { label: Range, value: 1 MHz – 6 GHz }
  - { label: Bandwidth, value: up to ~20 MHz }
  - { label: TX, value: Yes (half-duplex) }
  - { label: With GopherTrunk, value: Receive-only decode }
  - { label: Price, value: around $150 }
  - { label: Buy, value: "<a class=\"btn btn--buy\" href=\"https://www.amazon.com/dp/B0BKH7Z2NJ?tag=gophertrunk-20\" rel=\"nofollow sponsored noopener\">View on Amazon &rarr;</a>" }
see_also: [rtl-sdr, airspy, airspy-hf-plus, bladerf, limesdr, plutosdr, sdrplay-rsp1a, zadig, software-defined-radio]
related_lessons:
  - { title: "SDR hardware — RTL-SDR, HackRF, Airspy", url: /learn/rf-sdr/sdr-hardware/ }
related_reading:
  - { title: "RF Front End, Part 11: HackRF One", url: /blog/deep-dives/rf-front-end-11-hackrf-one/ }
cite_urls:
  - https://en.wikipedia.org/wiki/HackRF_One
faq:
  - q: "Is the HackRF One good for police scanning with GopherTrunk?"
    a: "It works, but it is overkill. HackRF's 8-bit ADC gives less dynamic range than an Airspy, and its transmit and 6 GHz reach are wasted on receive-only VHF/UHF scanning. For following P25/DMR/NXDN trunked systems, a $30 RTL-SDR or an Airspy is usually the better buy. Reach for a HackRF when you also want its huge tuning range or transmit for other projects."
  - q: "Can GopherTrunk transmit with a HackRF?"
    a: "No. GopherTrunk is a receive-only decoder — it uses the HackRF purely as a receiver. Transmitting also requires appropriate licensing and is outside GopherTrunk's scope."
  - q: "Does GopherTrunk need libhackrf or hackrf-tools?"
    a: "No. GopherTrunk speaks the HackRF's USB protocol directly with a pure-Go driver. On Linux you add one udev rule; on Windows the in-box WinUSB driver usually binds automatically. No SoapySDR or vendor C libraries required."
  - q: "HackRF One or Airspy for GopherTrunk?"
    a: "For scanning, the Airspy — its 12-bit ADC and cleaner front end decode weak and busy channels better than the HackRF's 8-bit sampling. Choose the HackRF only if you need 1 MHz–6 GHz coverage or transmit for other work."
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

<a class="btn btn--buy" href="https://www.amazon.com/dp/B0BKH7Z2NJ?tag=gophertrunk-20" rel="nofollow sponsored noopener">Check price on Amazon &rarr;</a>

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Huge range, transmit-capable, 8-bit.** HackRF One covers **1 MHz–6 GHz** and can
transmit — but its 8-bit ADC gives less dynamic range than an
[Airspy](/reference/airspy/), and **GopherTrunk uses it receive-only**. **For scanning
it's overkill:** a ~$30 [RTL-SDR](/reference/rtl-sdr/) or an Airspy decodes
[P25](/reference/project-25/)/[DMR](/reference/dmr/)/[NXDN](/reference/nxdn/) better for
less. **Buy it** if you also want 6 GHz reach or transmit for other projects. **~$150.**
Like every receiver, it can't decode [AES encryption](/police-scanner-encryption/).
</div>

## Overview

Its huge range and TX capability make it popular for experimentation, but it uses 8-bit
sampling (less dynamic range than [Airspy](/reference/airspy/)) and transmit is
irrelevant to receive-only scanning.

## Relevance to SDR

For decoding trunked voice, HackRF is overkill; an [RTL-SDR](/reference/rtl-sdr/) or
[Airspy](/reference/airspy/) is usually the better fit, but GopherTrunk can use it as a
receiver. Among transmit-capable peers the [bladeRF](/reference/bladerf/),
[LimeSDR](/reference/limesdr/) and [PlutoSDR](/reference/plutosdr/) trade the HackRF's
6 GHz reach for higher-bit ADCs and, in some cases, full-duplex operation.

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
      # rf_amp: true              # front-end RF amp on the "auto" preset — see below
      # narrowband_filter: true   # HackRF Pro only — see below
      # fpga_dc_block: true       # HackRF Pro only — see below
```

The HackRF has no true AGC, so `gain: auto` maps to a fixed LNA 16 / VGA 20 dB
split with the front-end **RF amplifier off**. Set `rf_amp: true` to turn the
amp on for that preset: it lowers the noise figure by ~14 dB and can recover a
weak-signal site, but because it adds gain ahead of everything it can overload
the front end near a strong transmitter — so it is opt-in and off by default.
(SDRTrunk defaults the amp on but likewise lets you disable it.) A manual
tenths-of-dB `gain:` value is unaffected by `rf_amp`; `rf_amp` is ignored, with
a one-line startup warning, on a device that has no switchable amplifier.

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

## HackRF Pro

GopherTrunk identifies the board from the firmware's `board_id` readback at
open, so a **HackRF Pro** (board ID 5, codename *Praline*) reports `HackRF Pro`
as its product name in `gophertrunk sdr list`, the **Devices** panel, and the
startup log — distinct from the original `HackRF One` (board ID 2) and the
`HackRF One R9` (board ID 4). No configuration is needed for detection.

The Pro adds a **switchable narrowband anti-alias filter** in its RF front end.
Engaging it tightens adjacent-channel rejection, which can lift a marginal
decode on a crowded band where a strong neighbour is spilling into a narrowband
voice channel (e.g. 12.5 kHz P25). The trade-off is reduced usable bandwidth, so
it is off by default; enable it per device:

```yaml
sdr:
  devices:
    - serial: "…"
      role: voice
      narrowband_filter: true   # HackRF Pro only
```

The option is ignored — with a one-line startup warning — on any board that
lacks the filter, including the original HackRF One, so it is safe to leave in a
shared config.

The Pro can also strip the zero-IF **DC-offset spike in its FPGA**, before the
samples ever leave the device. This is the same spur GopherTrunk's P25 voice
path removes in software (a first-order DC-block), but doing it in the gateware
is cheaper and — unlike the software block, which only sits on the voice decode
path — it also cleans the control channel. Measured on hardware, engaging it
drops the raw stream's DC magnitude from several counts to zero. Enable it per
device with `fpga_dc_block: true`:

```yaml
sdr:
  devices:
    - serial: "…"
      role: control
      fpga_dc_block: true       # HackRF Pro only
```

Both options are ignored — with a startup warning — on any board without the
hardware, so they are safe to leave in a shared config.

> **Extended precision is blocked in firmware, not here.** The Pro advertises a
> 16-bit *extended-precision* RX mode (FPGA down-conversion + decimation, ~9–11
> ENOB). The host command to select the bitstream exists
> (`hackrf_set_fpga_bitstream`), and the gateware itself is complete, but the
> coordinated mode is **not implemented in the released Pro firmware**: as of
> `master` (Aug 2026) `fpga_init()` only programs the standard bitstream's
> registers (`// TODO support the other bitstreams`), the `RADIO_CONFIG_EXT_PRECISION_RX`
> path is dead code, and the MCU's SGPIO capture stays hardwired to the standard
> 2-byte sample format with no host request to change it. So loading bitstream 2
> from the host produces an incoherent stream — this is a Great Scott Gadgets
> firmware to-do, not something a host driver can work around. When GSG ships the
> mode switch, the driver side is well-understood (interleaved int16 LE I/Q,
> 4 bytes/pair carrying a sign-extended 12-bit value; decimation register = log2
> of 16×–128×). Detection, the narrowband filter, and the FPGA DC-block above are
> the Pro pieces that work today.

## Where to buy

The HackRF One is sold by Great Scott Gadgets and resellers; the Amazon listing below
is a Nooelec bundle that adds an ANT500 telescopic antenna and SMA adapters — a
convenient way to get on the air. If you only want to *scan* trunked systems, a
[RTL-SDR](/reference/rtl-sdr/) or [Airspy](/reference/airspy/) is the cheaper, better-
suited pick — see [best SDR for GopherTrunk](/best-sdr-for-gophertrunk/) and
[Airspy vs RTL-SDR vs HackRF](/airspy-vs-rtl-sdr-vs-hackrf/).

<a class="btn btn--buy" href="https://www.amazon.com/dp/B0BKH7Z2NJ?tag=gophertrunk-20" rel="nofollow sponsored noopener">HackRF One bundle on Amazon &rarr;</a>

*As an Amazon Associate, GopherTrunk earns from qualifying purchases — at no extra cost
to you. It never changes what we recommend.*

## Sources

[^wiki]: [HackRF One](https://en.wikipedia.org/wiki/HackRF_One) — Wikipedia, on the HackRF One wideband half-duplex transceiver and its specifications.
