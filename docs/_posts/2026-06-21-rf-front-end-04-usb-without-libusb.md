---
title: "RF Front End, Part 4: Talking to USB Without libusb"
description: How GopherTrunk speaks to RTL-SDR hardware over raw kernel USB interfaces instead of linking libusb, and the single Transport contract that lets one driver run on three wildly different OS USB stacks.
category: deep-dives
tags: [sdr, go, usb, transport, software-design]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "RF Front End"
series_part: 4
---

*Part 4 of **RF Front End**. We open the box on how GopherTrunk reaches a USB
dongle without `libusb`: one tiny Go interface, three per-OS implementations
chosen at compile time, and a mock that lets the whole RTL2832U stack run in CI
with no hardware attached.*

## In this post

- Why GopherTrunk talks to the **kernel's USB interface directly** instead of
  linking `libusb` — and what that buys us (one static binary, **no CGO**).
- The **`Transport` interface** in `usb.go` — the minimal slice of USB the
  RTL2832U actually needs, and nothing more.
- The **build-tag platform split**: one interface, per-OS files selected by
  `//go:build` tags, plus a `MockTransport` for tests.
- The problem of keeping **one contract** that fits USBDEVFS, WinUSB, and IOKit
  at once.

## What the USB transport layer does

An RTL-SDR dongle is, electrically, a USB 2.0 device with a startlingly narrow
control surface. The RTL2832U demodulator firmware only ever wants three things
from the host: **vendor control transfers out** (write a chip register), **vendor
control transfers in** (read one back), and a **bulk-IN endpoint** that fire-hoses
8-bit IQ samples at 2.4 MS/s. There are no class requests, no isochronous
streams, no multiple-configuration gymnastics. The tuner register dance, the PLL
math, the gain tables — all of it rides on top of those few primitives.

So the job of GopherTrunk's USB layer is to expose exactly that narrow slice and
hide everything else. The package doc says it plainly:

```go
// internal/sdr/rtlsdr/usb/usb.go
// Package usb is the platform-abstraction layer that the pure-Go RTL-SDR
// driver speaks to. It exposes the minimal slice of USB the RTL2832U
// demodulator needs — vendor control transfers in both directions and an
// async bulk-IN endpoint — and nothing else.
```

The conventional way to do this in Go is to link `libusb` via CGO. We
deliberately don't. Linking `libusb` would reintroduce a C toolchain at build
time, a shared-library dependency at run time, and the cross-compilation
headaches that come with both — breaking the single-static-binary promise the
rest of GopherTrunk is built around. Instead, each OS already exposes a way to
drive raw USB from userspace: **USBDEVFS** ioctls on Linux, the **WinUSB** driver
on Windows, **IOKit** on macOS. GopherTrunk speaks to those directly. The cost is
that we write three backends by hand; the payoff is a pure-Go, CGO-free,
`go build`-and-ship binary on every target.

## How GopherTrunk implements it in Go

Everything funnels through two interfaces. The first, `Enumerator`, discovers and
opens devices; the second, `Transport`, is a claimed handle you do I/O on. Here's
the `Transport` contract, trimmed to its shape:

```go
// internal/sdr/rtlsdr/usb/usb.go
type Transport interface {
    ControlIn(bRequest uint8, wValue, wIndex uint16, n int, timeoutMs int) ([]byte, error)
    ControlOut(bRequest uint8, wValue, wIndex uint16, data []byte, timeoutMs int) error

    ClaimInterface(num int) error
    ReleaseInterface(num int) error

    StartBulkIn(epAddr byte, ringBufs, bufLen int, onPacket func([]byte), onStreamDead func(error)) error
    StopBulkIn() error

    Reset() error
    Close() error
}
```

That's the whole vocabulary. Two control methods, two claim methods, a bulk-IN
pair, plus `Reset` and `Close`. Notice what *isn't* here: no endpoint
descriptors, no configuration selection, no transfer-type enums, no
buffer-pool API. The RTL2832U only exposes interface 0 with a single bulk-IN
endpoint (`0x81`), so the interface bakes in those assumptions rather than
modelling the full USB spec. The narrowness is the point — it's what makes three
implementations tractable.

The control transfers carry libusb-style request-type bytes, which the package
exports as the only two values the firmware understands:

```go
// internal/sdr/rtlsdr/usb/usb.go
const (
    VendorOut uint8 = 0x40 // bmRequestType: host → device, vendor, device recipient
    VendorIn  uint8 = 0xC0 // bmRequestType: device → host, vendor, device recipient
)
```

`StartBulkIn` is the only method with real machinery behind it. It submits a ring
of `ringBufs` buffers of `bufLen` bytes each on endpoint `epAddr`, and invokes
`onPacket` from a dedicated reaper goroutine each time a buffer completes. The
slice handed to `onPacket` is **owned by the transport and reused** once the
callback returns, so the driver copies anything it wants to keep. The second
callback, `onStreamDead`, fires exactly once if every buffer dies of an
unrecoverable USB error *without* `StopBulkIn` being called — a hook we added
after a hang we'll get to below.

Discovery rides on the sibling interface:

```go
// internal/sdr/rtlsdr/usb/usb.go
type Enumerator interface {
    Name() string                                // "usbdevfs", "winusb", "iokit", "mock"
    List(vid, pid uint16) ([]Descriptor, error)  // 0 = wildcard
    Open(d Descriptor) (Transport, error)
}

func DefaultEnumerator() Enumerator { return platformEnumerator() }
```

`DefaultEnumerator` is the only public entry point. It calls `platformEnumerator`,
and *that* function is defined once per OS — every backend file provides its own.
On Linux it returns `&linuxEnumerator{}`, on Windows `&winEnumerator{}`, on macOS
the IOKit-backed enumerator. The driver above never names a concrete type; it
asks for `DefaultEnumerator()` and gets whatever the build produced.

The selection is pure build tags. Each backend file opens with a `//go:build`
header so the Go toolchain compiles in exactly one:

```go
// internal/sdr/rtlsdr/usb/usb_linux.go
//go:build linux && (amd64 || arm64 || 386 || arm || riscv64 || loong64)

// internal/sdr/rtlsdr/usb/usb_windows.go
//go:build windows && (amd64 || arm64)

// internal/sdr/rtlsdr/usb/usb_darwin.go
//go:build darwin

// internal/sdr/rtlsdr/usb/usb_other.go
//go:build !(linux && (...)) && !(windows && (amd64 || arm64)) && !darwin
```

The `usb_other.go` catch-all matters more than it looks: it returns an
`unsupportedEnumerator` whose every method yields `ErrUnsupportedPlatform`. That
keeps the package *compilable* on freebsd, plan9, or any target without a real
backend — the higher layers build and link everywhere, and the failure surfaces
at runtime from `List`/`Open` instead of breaking `go build` on an exotic OS.

### The mock that needs no hardware

The same `Transport` shape is what makes the driver testable. `MockTransport`
implements the interface by replaying a scripted sequence of control exchanges:

```go
// internal/sdr/rtlsdr/usb/usb_mock.go
type CtrlExchange struct {
    In       bool
    BRequest uint8
    WValue   uint16
    WIndex   uint16
    Data     []byte // expected for OUT
    Reply    []byte // returned for IN
    Err      error
    // ...
}

type MockTransport struct {
    Script []CtrlExchange
    Step   int
    Err    error
    // ...
    BulkPackets [][]byte
}
```

A test builds a `Script` of the exact register reads and writes the RTL2832U
bring-up should emit, runs the real driver against the mock, and then asserts
`MockTransport.Err` is nil and `Remaining()` is zero. Bulk streaming is mocked by
shoving pre-built byte slices into `BulkPackets`, which the mock's goroutine feeds
through `onPacket` on the same `(shape)` contract the real backends honor. Because
the mock satisfies `Transport`, the rtl2832u register layer and every tuner
driver above it can be unit-tested without a dongle plugged in — which is the only
way CI on a headless Linux runner can exercise Windows-and-macOS-bound code paths
at all.

## The problem we hit: one contract, three alien USB stacks

The first cut of this package was Linux-only. USBDEVFS gave us submit-a-URB,
reap-a-URB, claim-an-interface, and the `Transport` interface was essentially a
thin shadow of those ioctls. That worked beautifully until we started PR-02
(Windows) and PR-10 (macOS) and discovered the three OS USB stacks barely agree
on anything.

**The symptom.** Every method we'd sketched leaked a Linux assumption. Our first
`StartBulkIn` returned URB handles for the caller to manage — fine on USBDEVFS,
meaningless on WinUSB (which thinks in overlapped I/O and event handles) and on
IOKit (which has no URB concept at all, just synchronous `ReadPipe`). Our claim
semantics assumed a kernel driver you detach; WinUSB has *already* taken exclusive
ownership at initialize time and "claim" is a no-op. Cancellation was modelled on
`USBDEVFS_DISCARDURB`; macOS cancels with `AbortPipe`, Windows with `AbortPipe`
plus a manual event drain. Any interface that exposed those mechanics couldn't be
satisfied by all three.

**The root cause.** We had let the *implementation* leak into the *contract*. The
interface was a description of USBDEVFS, not a description of "what the RTL2832U
needs." Every Linux-shaped detail in the signature became a constraint the other
two backends had to either fake or violate.

**The Go fix.** We rewrote the interface around what the *driver* asks for, not
what any OS provides — and pushed every OS-specific mechanism strictly inside the
implementations. Concretely:

- `StartBulkIn` stopped returning handles. The caller supplies geometry
  (`ringBufs`, `bufLen`, `epAddr`) and two callbacks, and **never sees a URB, an
  `OVERLAPPED`, or a `pipeRef`**. Whether the bytes arrive via a single reaper
  goroutine (Linux), one OS thread per slot (macOS), or `WaitForMultipleObjects`
  (Windows) is invisible above the contract.
- `ClaimInterface`/`ReleaseInterface` became *intent*, not mechanism. Linux turns
  a claim into `USBDEVFS_CLAIMINTERFACE` plus auto-detach of the kernel DVB
  driver; Windows makes it a no-op because initialize already claimed; macOS turns
  it into the IOCFPlugIn interface-iterator dance. The driver just says "I own
  interface 0."
- Buffer ownership was nailed down *in the doc comment* so all three could honor
  it identically: the slice passed to `onPacket` is transport-owned and reused.
  That one sentence let each backend reuse its ring buffers without copying, and
  let the driver copy exactly once.

The discipline that made it work was a kind of subtraction: every time a method
needed an OS-specific type to be expressible, that was the signal the method was
wrong. The final `Transport` has **eight methods and zero platform types in its
signature** — and that's precisely why USBDEVFS, WinUSB, and IOKit can each be one
file behind it.

## The design principle: ports & adapters at the OS boundary

This is the **hexagonal / ports-and-adapters** pattern applied at the operating
system boundary. `Transport` and `Enumerator` are the *ports* — abstract contracts
owned by the application's needs. `linuxTransport`, `winTransport`, and
`darwinTransport` are the *adapters* — concrete implementations that translate
those needs into one specific OS's USB ABI. The RTL2832U driver lives entirely on
the port side and is, by construction, ignorant of which adapter it's talking to.

### How that principle shaped the Go code

- **The port is defined by the consumer, not the provider.** `Transport` models
  what the RTL2832U firmware requires — vendor control + one bulk-IN endpoint —
  not what USBDEVFS happens to offer. When an adapter couldn't express a method
  without leaking its own types, we changed the *port*, not the adapter.
- **Adapters are selected at compile time, not runtime.** `//go:build` tags mean
  the Windows binary contains zero IOKit code and the macOS binary contains zero
  WinUSB code. There's no `switch runtime.GOOS` and no dead platform branches
  linked into the wrong binary.
- **The unsupported case is an adapter too.** `usb_other.go` is the null adapter:
  it satisfies the port and fails politely. The port stays total — every platform
  has *an* implementation — so nothing above has to special-case "no USB here."
- **The mock is just another adapter.** `MockTransport` plugs into the same port
  as the real backends, which is what lets hardware-free tests drive the full
  driver. The driver genuinely cannot tell a scripted mock from a real R820T.

You'll recognize this as the same instinct behind the driver registry from the
[SDR Internals series]({{ '/blog/deep-dives/sdr-internals-02-sdr-device-driver-registry/' | relative_url }}):
define the contract by what the core needs, then let interchangeable
implementations satisfy it from below.

## Where this goes next

We've drawn the port; the next three posts build the adapters. **Part 5** dives
into the Linux backend — hand-rolled USBDEVFS ioctl encoding, submitting async
URBs, and reaping them in a single goroutine for sustained throughput. **Part 6**
takes on macOS (IOKit via purego, OS-thread-pinned reader goroutines) and Windows
(WinUSB function pointers, overlapped I/O), and shows how both fold back onto the
exact same eight-method contract. With the transport solid, **Part 7** finally
brings up the RTL2832U itself on top of it.

## FAQ

**Why not just link libusb and be done with it?**
`libusb` would reintroduce CGO and a shared-library run-time dependency, breaking
GopherTrunk's single-static-binary, cross-compilable build. Writing three thin
backends in pure Go is more work up front but keeps `go build` honest on every
target.

**Doesn't writing three USB backends triple the bug surface?**
It spreads it, but the narrow contract contains it. Each adapter is one file
implementing eight methods, and `MockTransport` lets the entire driver above the
port be tested identically regardless of which adapter runs underneath — so most
logic is exercised in CI with no hardware at all.

**Why does the interface assume interface 0 and endpoint 0x81?**
Because the RTL2832U does. The port is shaped by what the device needs, not by the
full generality of USB. If GopherTrunk ever drove a multi-interface device, that
would be a reason to widen the port — deliberately, with all three adapters in
view.

## Series navigation

**Part 4 of 14** · ←
[Part 3]({{ '/blog/deep-dives/rf-front-end-03-driver-registry-enumeration/' | relative_url }})
· Next →
[Part 5: USB on Linux — USBDEVFS & async URBs]({{ '/blog/deep-dives/rf-front-end-05-usb-on-linux-usbdevfs/' | relative_url }})
