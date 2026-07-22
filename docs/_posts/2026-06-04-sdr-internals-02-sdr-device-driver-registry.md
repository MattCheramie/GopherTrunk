---
title: "SDR in Pure Go, Part 2: SDR Devices & the Driver Registry"
description: How GopherTrunk talks to RTL-SDR, HackRF, and Airspy hardware in pure Go behind one Device interface, and how a self-registering driver registry keeps the core hardware-agnostic.
category: deep-dives
tags: [sdr, go, rtl-sdr, hackrf, airspy, software-design]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "SDR Internals"
series_part: 2
---

*Part 2 of **SDR Internals**. We start at the metal: the USB drivers that turn a
$25 dongle into a stream of IQ samples, and the registry pattern that lets the
engine support new hardware without knowing it exists.*

## In this post

- How GopherTrunk drives **RTL-SDR, HackRF, and Airspy** with **zero CGO**.
- The single **`Device` interface** every backend implements.
- The **driver registry** — a dependency-inversion pattern where drivers
  register themselves at `init()` and the binary's import set decides what
  hardware it can talk to.

## What an SDR device layer does

An SDR device's job is narrow but unforgiving: open the USB hardware, tune it,
set gain and sample rate, and stream IQ samples back without dropping any. Every
dongle does this differently — different chips (RTL2832U, the HackRF SPI command
set, the Airspy register map), different tuners
([R820T]({{ '/reference/r820t-tuner/' | relative_url }}), E4000, FC0012), and
different USB control transfers.

GopherTrunk implements all of this in pure Go, speaking to the kernel's USB
interface directly (USBDEVFS on Linux, WinUSB on Windows, IOKit on macOS)
instead of linking `libusb`. Reference entries:
[RTL-SDR]({{ '/reference/rtl-sdr/' | relative_url }}),
[HackRF]({{ '/reference/hackrf/' | relative_url }}),
[Airspy]({{ '/reference/airspy/' | relative_url }}).

## How GopherTrunk implements it in Go

Every backend, no matter how different the silicon, hides behind one interface
in `internal/sdr/device.go`:

```go
type Device interface {
    StreamIQ(ctx context.Context) (<-chan []complex64, error)
    SetCenterFreq(hz uint32) error
    SetSampleRate(hz uint32) error
    SetGain(tenthDB int) error
    // ...identity and teardown
}
```

`StreamIQ` is the heart of it: give it a context, get back a channel of IQ
chunks (~6 ms each at 2.4 MS/s). Cancel the context and the stream stops. The
rest of the engine only ever sees this interface — it never imports
`rtlsdr`, `hackrf`, or `airspy`.

The concrete drivers live in sub-packages: `internal/sdr/rtlsdr/purego`,
`internal/sdr/hackrf`, `internal/sdr/airspy`, and `internal/sdr/airspyhf`. There
are also `rtltcp` and `soapyremote` backends for networked SDRs, plus a `mock`
driver that replays raw IQ files — and because they all satisfy `Device`, the
engine can't tell a real R820T from a recorded capture.

## The design principle: registry + dependency inversion

How does the engine get a driver without depending on it? Through a
**process-global registry**. Each driver calls `Register` from its `init()`:

```go
// internal/sdr/registry.go
var (
    registryMu sync.RWMutex
    registry   = map[string]Driver{}
)

func Register(d Driver) {
    registryMu.Lock()
    defer registryMu.Unlock()
    registry[d.Name()] = d
}

// internal/sdr/rtlsdr/purego/register.go
func init() { sdr.Register(&Driver{}) }
```

This is **dependency inversion**: the high-level engine depends on the abstract
`Driver`/`Device` contracts, and the low-level drivers depend *up* on those same
contracts by registering themselves. The arrow points the "wrong" way on
purpose.

<figure class="lab-figure">
<svg viewBox="0 0 520 180" width="520" height="180" role="img" aria-label="Four SDR driver packages — rtlsdr/purego, hackrf, airspy, and airspyhf — each call sdr.Register from their init function at import time, populating a single process-global registry that is a map from driver name to Driver.">
  <text x="85" y="14" text-anchor="middle" fill="var(--fg-muted)" font-size="10">init() at import</text>
  <rect x="10" y="22" width="150" height="26" rx="6" fill="none" stroke="currentColor"/>
  <text x="85" y="39" text-anchor="middle" fill="currentColor" font-size="10">rtlsdr/purego</text>
  <rect x="10" y="54" width="150" height="26" rx="6" fill="none" stroke="currentColor"/>
  <text x="85" y="71" text-anchor="middle" fill="currentColor" font-size="10">hackrf</text>
  <rect x="10" y="86" width="150" height="26" rx="6" fill="none" stroke="currentColor"/>
  <text x="85" y="103" text-anchor="middle" fill="currentColor" font-size="10">airspy</text>
  <rect x="10" y="118" width="150" height="26" rx="6" fill="none" stroke="currentColor"/>
  <text x="85" y="135" text-anchor="middle" fill="currentColor" font-size="10">airspyhf</text>
  <line x1="160" y1="35" x2="290" y2="55" stroke="currentColor"/><polygon points="290,51 300,55 290,59" fill="currentColor"/>
  <line x1="160" y1="67" x2="290" y2="75" stroke="currentColor"/><polygon points="290,71 300,75 290,79" fill="currentColor"/>
  <line x1="160" y1="99" x2="290" y2="95" stroke="currentColor"/><polygon points="290,91 300,95 290,99" fill="currentColor"/>
  <line x1="160" y1="131" x2="290" y2="115" stroke="currentColor"/><polygon points="290,111 300,115 290,119" fill="currentColor"/>
  <rect x="300" y="30" width="200" height="110" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="400" y="52" text-anchor="middle" fill="var(--fg-muted)" font-size="8">internal/sdr/registry.go</text>
  <text x="400" y="82" text-anchor="middle" fill="var(--accent)" font-size="12">registry</text>
  <text x="400" y="102" text-anchor="middle" fill="var(--accent)" font-size="11">map[string]Driver</text>
  <text x="400" y="124" text-anchor="middle" fill="var(--fg-muted)" font-size="9">sdr.Register(d)</text>
</svg>
<figcaption>Each driver package calls <code>sdr.Register</code> from its <code>init()</code>, so the binary's blank-import set — not a switch statement — decides which hardware the registry knows about.</figcaption>
</figure>

### How that principle shaped the Go code

- **Blank imports choose the hardware.** `cmd/gophertrunk` blank-imports the
  four real drivers (`import _ ".../rtlsdr/purego"`). Their `init()` functions
  populate the registry. Want a build that only talks to RTL-SDR? Change the
  import set — not the engine.
- **The engine enumerates, it doesn't instantiate.** It calls `sdr.Drivers()`
  to list what's available and `sdr.DriverByName()` to pick one. There is no
  `switch deviceType { case "rtlsdr": ... }` anywhere in the core.
- **New hardware is purely additive.** Adding a backend means writing a package
  that satisfies `Device` and calls `Register` — no existing file changes. The
  same hook is how the baseband-replay "virtual tuner" mounts recorded WAVs as
  if they were real dongles.

<figure class="lab-figure">
<svg viewBox="0 0 680 130" width="680" height="130" role="img" aria-label="The runtime lookup flow: the engine calls sdr.DriverByName to enumerate and pick a driver, calls Open on that Driver to get a Device, and calls StreamIQ on the Device interface to obtain a receive-only channel of complex64 IQ chunks.">
  <rect x="8" y="50" width="68" height="40" rx="6" fill="none" stroke="currentColor"/>
  <text x="42" y="74" text-anchor="middle" fill="currentColor" font-size="10">engine</text>
  <line x1="76" y1="70" x2="90" y2="70" stroke="currentColor"/><polygon points="90,66 98,70 90,74" fill="currentColor"/>
  <rect x="98" y="50" width="138" height="40" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="167" y="68" text-anchor="middle" fill="var(--accent)" font-size="10">sdr.DriverByName()</text>
  <text x="167" y="82" text-anchor="middle" fill="var(--fg-muted)" font-size="8">enumerate → pick</text>
  <line x1="236" y1="70" x2="250" y2="70" stroke="currentColor"/><polygon points="250,66 258,70 250,74" fill="currentColor"/>
  <rect x="258" y="50" width="84" height="40" rx="6" fill="none" stroke="currentColor"/>
  <text x="300" y="68" text-anchor="middle" fill="currentColor" font-size="10">Driver</text>
  <text x="300" y="82" text-anchor="middle" fill="var(--fg-muted)" font-size="8">Open(serial)</text>
  <line x1="342" y1="70" x2="356" y2="70" stroke="currentColor"/><polygon points="356,66 364,70 356,74" fill="currentColor"/>
  <rect x="364" y="50" width="84" height="40" rx="6" fill="none" stroke="currentColor"/>
  <text x="406" y="68" text-anchor="middle" fill="currentColor" font-size="10">Device</text>
  <text x="406" y="82" text-anchor="middle" fill="var(--fg-muted)" font-size="8">one interface</text>
  <line x1="448" y1="70" x2="462" y2="70" stroke="currentColor"/><polygon points="462,66 470,70 462,74" fill="currentColor"/>
  <rect x="470" y="50" width="200" height="40" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="570" y="68" text-anchor="middle" fill="var(--accent)" font-size="10">StreamIQ(ctx)</text>
  <text x="570" y="82" text-anchor="middle" fill="var(--fg-muted)" font-size="8">&lt;-chan []complex64</text>
</svg>
<figcaption>Enumerate, open, stream: the engine names a driver, opens it into a <code>Device</code>, and only ever touches the <code>Device</code> interface — it never imports <code>rtlsdr</code>, <code>hackrf</code>, or <code>airspy</code>.</figcaption>
</figure>

The registry pattern is one of the most reused ideas in the codebase — you'll
see it again for vocoders in
[Part 12]({{ '/blog/deep-dives/sdr-internals-12-voice-coding-vocoders/' | relative_url }}).

## Where this goes next

Each driver is a small protocol implementation in its own right — the RTL2832U
register dance, the R820T PLL math, HackRF's transceiver state machine. A future
**"Pure-Go SDR Drivers"** series will take them one chip at a time. For now, the
takeaway is the shape: one interface, many self-registering implementations.

## FAQ

**Why avoid libusb and write USB transport in Go?**
Linking `libusb` would reintroduce CGO and a runtime dependency, breaking the
single-static-binary promise. Talking to the kernel USB interface directly keeps
the build pure Go and cross-compilable.

**Can GopherTrunk use SDRs it doesn't natively support?**
Yes — via the `rtltcp` and `soapyremote` network backends, which satisfy the
same `Device` interface, anything exposed over those protocols (USRP, LimeSDR,
bladeRF, …) can stream into the pipeline.

**How does testing work without hardware?**
A `mock` driver replays raw `u8`/`f32` IQ files as a `Device`. Integration tests
register it and feed synthesized signals through the full stack — covered in
[Part 14]({{ '/blog/deep-dives/sdr-internals-14-apis-testing-pure-go/' | relative_url }}).

## Series navigation

**Part 2 of 14** · ←
[Part 1]({{ '/blog/deep-dives/sdr-internals-01-what-is-software-defined-radio/' | relative_url }})
· Next →
[Part 3: The SDR pool & streaming concurrency]({{ '/blog/deep-dives/sdr-internals-03-sdr-pool-streaming-concurrency/' | relative_url }})
