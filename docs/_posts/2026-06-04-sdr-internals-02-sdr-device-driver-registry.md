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
