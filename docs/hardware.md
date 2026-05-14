---
layout: page
title: Hardware setup
description: RTL-SDR dongles, udev rules, DVB blacklist, and supported USB chipsets
nav_group: Install
---

# Hardware Setup

GopherTrunk ships with a pure-Go RTL-SDR driver — no `librtlsdr`,
`libusb`, or C toolchain on the build host. The daemon talks to
RTL2832U dongles directly through the platform's USB stack:
USBDEVFS ioctls on Linux, IOKit on macOS, WinUSB on Windows.

## Supported devices

The same code path covers everything `librtlsdr` did — RTL2832U
paired with R820T / R820T2 / R828D / E4000 / FC0012 / FC0013 /
FC2580. Tuner detection is automatic on `Open`.

Tested combinations:

| Device | Tuner | Notes |
| --- | --- | --- |
| **NooElec NESDR Smart v5** | R820T2 | 0.5 ppm TCXO, software-controllable bias-tee. Use `bias_tee: true` in config to power an external LNA via the SMA. |
| NooElec NESDR Smart (v4 and earlier) | R820T2 | TCXO; no bias-tee on early units. |
| Generic RTL-SDR Blog v3 / v4 | R820T2 / R828D | Bias-tee on most units. |
| Plain RTL2832U DVB-T sticks | R820T | No TCXO; expect a few ppm offset — set `ppm:` in config after measuring. |

If you have a v5 (or any modern dongle with a bias-tee) and want to
power an LNA, the config snippet looks like:

```yaml
sdr:
  devices:
    - serial: "00000001"      # whatever `gophertrunk sdr list` shows
      role: control            # or voice / auto
      ppm: 0                   # 0 is fine for TCXO-equipped units
      gain: "auto"             # or a numeric tenths-of-dB string like "496"
      bias_tee: true           # 5V on the SMA — only enable if you want it
```

## Linux

No package install is needed for the build itself; the driver only
needs USB-device permissions at runtime.

Add a udev rule so non-root processes can claim the dongle:

```
# /etc/udev/rules.d/20-rtlsdr.rules
SUBSYSTEM=="usb", ATTRS{idVendor}=="0bda", ATTRS{idProduct}=="2838", MODE="0666"
```

Reload udev (`sudo udevadm control --reload && sudo udevadm trigger`) and
unplug/replug the dongle.

Blacklist the kernel's DVB driver, which otherwise grabs the device first:

```
# /etc/modprobe.d/blacklist-dvb_usb_rtl28xxu.conf
blacklist dvb_usb_rtl28xxu
```

See [`install-linux.md`]({{ '/install-linux.html' | relative_url }})
for the full step-by-step including systemd service setup.

## macOS

IOKit lets user-space claim USB devices without rebinding the kernel
driver, so there's no kext to install and no driver swap to perform.
Plug in the dongle and run `gophertrunk sdr list`. See
[`install-macos.md`]({{ '/install-macos.html' | relative_url }}) for
the full step-by-step including the Gatekeeper bypass and launchd
service setup.

## Windows

Bind the dongle to **WinUSB** with [Zadig](https://zadig.akeo.ie)
once per device — see [`install-windows.md`]({{ '/install-windows.html' | relative_url }})
for the click-by-click walkthrough.

## Verifying the build

```sh
make build
./bin/gophertrunk sdr list
```

You should see one row per attached dongle with index, serial, tuner type
(usually `R820T2` or `R828D`), and the supported gain values. The
`Driver` column reads `rtlsdr` for every entry.

## Capturing IQ for replay

The mock driver replays raw u8-IQ files (`.cfile` format). You can
generate one with any tool that produces interleaved unsigned-8-bit
samples — e.g. `gqrx`'s baseband recorder or a dedicated capture
utility. Drop `.cfile` files under `testdata/iq/` to use them
through the mock driver.
