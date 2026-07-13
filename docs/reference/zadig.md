---
slug: zadig
title: Zadig
entry_type: hardware
category: sdr-software
description: Zadig is a Windows utility that installs the generic WinUSB driver onto an SDR dongle, replacing the default TV-tuner driver so SDR software can access the device.
keywords: Zadig, WinUSB, RTL-SDR driver, Windows, libusb, DVB-T driver, USB driver, libusbK, driver installation
aka: [Zadig, WinUSB]
autolink: true
see_also: [rtl-sdr, rtl2832u, rtl-tcp, soapysdr, gqrx, usb]
related_lessons:
  - { title: "SDR hardware — RTL-SDR, HackRF, Airspy", url: /learn/rf-sdr/sdr-hardware/ }
cite_urls:
  - https://zadig.akeo.ie/
  - https://github.com/libusb/libusb/wiki/Windows
---

**Zadig** is a small Windows utility that installs the generic **WinUSB** driver onto an
SDR dongle. Out of the box, Windows binds an [RTL-SDR](/reference/rtl-sdr/) to its TV-tuner
(DVB-T) driver, because the [RTL2832U](/reference/rtl2832u/) chip was designed as a digital
television receiver; Zadig replaces that with WinUSB so SDR software can bypass the TV stack
and talk to the [USB](/reference/usb/) device directly.[^home]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 110" role="img" aria-label="A dongle's default TV-tuner driver being replaced by the WinUSB driver via Zadig, after which SDR software can access it." xmlns="http://www.w3.org/2000/svg">
  <rect x="30" y="44" width="90" height="30" rx="5" fill="none" stroke="currentColor" stroke-width="1.2" stroke-dasharray="4 3"/><text x="75" y="63" text-anchor="middle" font-size="8.5" fill="currentColor">DVB-T driver</text>
  <line x1="125" y1="59" x2="185" y2="59" stroke="currentColor" marker-end="url(#zar)"/><text x="155" y="51" text-anchor="middle" font-size="8" fill="currentColor">Zadig</text>
  <rect x="195" y="44" width="90" height="30" rx="5" fill="currentColor" fill-opacity="0.18" stroke="currentColor" stroke-width="1.2"/><text x="240" y="63" text-anchor="middle" font-size="8.5" fill="currentColor">WinUSB</text>
  <line x1="290" y1="59" x2="350" y2="59" stroke="currentColor" marker-end="url(#zar)"/>
  <rect x="360" y="44" width="80" height="30" rx="5" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="400" y="63" text-anchor="middle" font-size="8.5" fill="currentColor">SDR app</text>
  <defs><marker id="zar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>Zadig swaps the dongle's default Windows driver for WinUSB so SDR software can access it.</figcaption>
</figure>

## How it works

On Windows, a USB device is claimed by whatever driver matches its vendor/product ID — for
an RTL dongle that is the bundled DVB-T television driver, which only knows how to hand back
demodulated TV, not the raw [IQ](/reference/iq-data/) an SDR needs. Zadig is a friendly
front end to **libwdi**, a library that generates and installs a signed driver package on
the fly.[^lu] You pick the device from a list, choose a target driver — **WinUSB** is the
usual choice, with **libusb-win32** and **libusbK** as alternatives — and Zadig builds a
matching `.inf`, signs it, and swaps the binding. Afterwards the device is a generic USB
endpoint that libusb-based SDR software (via the librtlsdr driver) can open directly, read
bulk-transfer samples from, and control.

The change is per-device and per-USB-port association, and it is reversible: Windows' Device
Manager can roll the driver back to the original TV driver if you ever want the DVB-T
function again. A frequent gotcha is installing WinUSB onto the *wrong* interface of a
composite device, or onto only one physical USB port — plugging the dongle into a different
port can present it as "new" hardware that needs the assignment repeated.

## In practice

Zadig is a Windows-only concern, and only for the first-time setup of a device. The typical
flow: plug in the dongle, run Zadig, enable *Options → List All Devices*, select
"Bulk-In, Interface (Interface 0)" (or the RTL entry), confirm the target is WinUSB, and
click *Replace Driver*. Do this before launching any SDR application. On **Linux and
macOS** there is no equivalent step — libusb (and IOKit on macOS) can claim the device
without swapping a kernel driver, so tools like [rtl_tcp](/reference/rtl-tcp/) and
[GQRX](/reference/gqrx/) work straight away, sometimes after blacklisting the `dvb_usb_rtl28xxu`
kernel module.

## Relevance to SDR

Nearly every Windows SDR newcomer meets Zadig on day one, because without the WinUSB swap the
dongle simply will not appear to SDR software. GopherTrunk's Windows installer bundles Zadig
to automate this step, so the driver is in place before the decoder first runs. GopherTrunk
itself talks to the radio through the standard libusb/librtlsdr path — the same interface
Zadig unlocks — so once the driver is swapped, the platform difference disappears and the
decode chain behaves identically to Linux and macOS.

## Sources

[^home]: [Zadig](https://zadig.akeo.ie/) — the official Zadig site, the USB driver installer that swaps a dongle's driver for WinUSB.
[^lu]: [libusb on Windows](https://github.com/libusb/libusb/wiki/Windows) — the libusb project wiki, on the WinUSB/libusbK back ends that Zadig (via libwdi) installs to give user-space access to USB devices.
