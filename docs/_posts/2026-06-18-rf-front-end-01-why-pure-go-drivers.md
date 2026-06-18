---
title: "RF Front End, Part 1: Why Drive Radios in Pure Go?"
description: The opening post of RF Front End, a 14-part deep dive into GopherTrunk's RF source layer — the pure-Go USB drivers that turn RTL-SDR, Airspy, and HackRF hardware into IQ samples without libusb, librtlsdr, or any C dependency.
category: deep-dives
tags: [sdr, go, usb, drivers, software-design]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "RF Front End"
series_part: 1
---

*Part 1 of **RF Front End**, a 14-part series that walks the RF source layer
behind GopherTrunk — the pure-Go USB drivers that turn a dongle into a stream of
IQ samples — one component per post, and explains the software-design principle
behind each piece and how that principle shaped the Go code.*

## In this post

- **Why the RF source layer gets its own series** — [SDR Internals]({{ '/blog/series/sdr-internals/' | relative_url }})
  Part 2 only skimmed it; the drivers are deep enough to fill fourteen posts.
- **The CGO/libusb "tax"** GopherTrunk refuses to pay, and the single static
  binary you get for refusing it.
- **What an RF source actually is**, and the three device families GopherTrunk
  drives in pure Go.
- **A map of the series** so you can jump to the chip or layer you care about.

## What an RF source is

Every signal GopherTrunk decodes starts the same way: some piece of hardware
digitizes a slice of the radio spectrum and hands the computer a torrent of
**IQ samples** — complex numbers that capture the amplitude and phase of the
signal. Everything after that — tuning, filtering, demodulation, FEC, the
trunking engine — is software. The "RF source" is the thing that produces those
samples: open the USB hardware, tune it, set gain and sample rate, and stream
`[]complex64` chunks back without dropping any.

That job sounds small. It is not. Each dongle does it differently — different
chips (the RTL2832U demodulator, the Airspy register map, HackRF's transceiver
command set), different tuners
([R820T]({{ '/reference/r820t-tuner/' | relative_url }}), R828D, the Airspy
front ends), different USB control transfers, different sample formats (unsigned
8-bit, signed 8-bit, real 12-bit packed). Hiding all of that behind one clean
`<-chan []complex64` is the entire point of the RF front end, and it is where a
surprising amount of GopherTrunk's hardest bugs live.

> New here? The reference entries on
> [software-defined radio]({{ '/reference/software-defined-radio/' | relative_url }})
> and [IQ data]({{ '/reference/iq-data/' | relative_url }}) cover the
> fundamentals this series builds on.

To make the scope concrete: an RF source has to keep up with the firehose. At
2.4 MS/s, an RTL-SDR delivers 2.4 million complex samples *every second*, packed
as interleaved bytes the USB endpoint streams in continuously. A chunk arrives
roughly every few milliseconds and must be unpacked from the hardware's wire
format — unsigned 8-bit for the RTL2832U, signed 8-bit for HackRF, real 12-bit
for Airspy — into `complex64`, and handed off before the next chunk lands. There
is no room to fall behind: a dropped URB is a gap in the IQ, and a gap in the IQ
is a corrupted symbol in whatever the decoder was tracking. The RF front end is a
soft-real-time problem wearing the clothes of a USB driver, and that tension —
keep up, allocate nothing you can avoid, never block the bus — is the through-line
of Parts 4 through 11.

### Why a separate series

[SDR Internals]({{ '/blog/series/sdr-internals/' | relative_url }}) Part 2 gave
the RF source layer a single post: here's the `Device` interface, here's the
registry, drivers register themselves, the engine stays hardware-agnostic. True,
and enough to understand the *shape*. But it left the inside of each driver as a
black box — "the RTL2832U register dance, the R820T PLL math, HackRF's
transceiver state machine," in that post's own words, deferred to "a future
series." This is that series.

The reason it deserves fourteen posts is that the RF front end is where pure-Go
ambition meets the metal. There is no `libusb` underneath catching our mistakes.
We are issuing raw USB control transfers, packing async URBs, decoding 8-bit IQ
in tight loops, and recovering dongles that fall off the bus mid-stream. There is
also no shared C library quietly normalizing four vendors' hardware into a common
shape: where a C-linked project inherits `librtlsdr`, `libairspy`, and
`libhackrf` and lets each vendor's headers define the abstraction, GopherTrunk has
to *choose* the abstraction and then earn it, chip by chip. That is more work, but
it is also the only way to keep the whole receive chain — antenna to audio —
inside one language you can read, test with the race detector, and cross-compile
with `go build`. The bugs are real, the fixes are specific, and the design
choices — one narrow interface,
self-registering drivers, a supervised pool — are what keep a pile of
chip-specific hacks from leaking into the rest of the engine.

## The CGO tax we refuse to pay

The conventional way to talk to an RTL-SDR from any language is to link
`librtlsdr`, which links `libusb`. For Airspy you link `libairspy`; for HackRF,
`libhackrf`; for the HF+, `libairspyhf`. Each is a C library, and in Go each one
drags in **CGO**.

CGO is a tax with several line items:

- **You lose the single static binary.** A CGO build links against shared
  objects that must be present on the target machine. "Install GopherTrunk" turns
  into "install GopherTrunk *and* `librtlsdr0` *and* the right `libusb-1.0` *and*
  hope the versions match."
- **You lose trivial cross-compilation.** `GOOS=windows GOARCH=amd64 go build`
  stops being a one-liner the moment CGO is on; you need a C cross-toolchain and
  the target's C libraries for every platform you ship.
- **You lose a uniform debugging story.** A crash in `libusb` is a crash in C —
  no Go stack trace, no race detector coverage, a different mental model at the
  exact layer where the timing-sensitive USB bugs live.
- **You lose build hermeticity.** The version of `libusb` on the developer's
  machine, the CI runner, and the operator's box can all differ, and the
  RF-front-end behavior can quietly differ with them. A pure-Go transport pins
  *all* of that behavior in the module graph, so the bytes-on-the-bus logic is the
  same everywhere the binary runs.

GopherTrunk pays none of it. As the README puts it, the project "has no C
dependencies at build or runtime (no `librtlsdr` / `libhackrf` / `libairspy` /
`libairspyhf` / `libusb` …) and ships as a single ~10 MB static binary for
Linux, macOS, and Windows." Every driver speaks to the **kernel's** USB interface
directly — USBDEVFS on Linux, WinUSB on Windows, IOKit on macOS — instead of
linking a C USB stack. The build stays `CGO_ENABLED=0`, and `go build` keeps
cross-compiling to every target with no toolchain gymnastics.

The cost of refusing the tax is that we have to *write* the USB transport and the
chip protocols ourselves. That cost is exactly what this series documents.

It is worth being precise about what "pure Go" buys an operator, because the
benefit is not abstract. A GopherTrunk release is one file. You download it,
`chmod +x`, and run it — no package manager, no `apt install librtlsdr0`, no
matching a runtime `libusb` against the one the binary was built against, no
LD_LIBRARY_PATH surprises on a stripped-down host. The Windows build talks WinUSB;
the macOS build talks IOKit; the Linux build talks USBDEVFS — and all three come
out of the *same* source tree with `GOOS`/`GOARCH` set, because none of them shell
out to a C toolchain. For a daemon that is meant to run unattended at the antenna —
often on a Raspberry Pi or a small ARM box — "one static binary with no shared
libraries to drift out from under it" is a reliability feature, not just a
packaging convenience.

The other half of the payoff is on *our* side of the build. Because every byte
from the USB endpoint to the `complex64` is Go, the whole RF front end is in scope
for `go test -race`, for table-driven unit tests against a mock USB transport, and
for the same profiler that watches the DSP layer. When a streaming bug shows up as
GC churn (Part 9) or a tuner comes up deaf (Part 8), we debug it with a Go stack
trace and a Go profile — not by attaching `gdb` to `libusb`. The pure-Go decision
is what makes the rest of this series *debuggable*, and that is most of why it was
worth making.

## The three device families

GopherTrunk drives three families of USB SDR in pure Go, plus a couple of network
backends and a mock. The families differ enough that each gets its own pair of
posts later:

- **RTL-SDR** — the $25 RTL2832U dongles (every osmocom tuner). The RTL2832U is a
  DVB-T demodulator repurposed as a raw ADC; the actual tuning happens in a
  separate tuner chip, usually the R820T/R828D. Cheap, ubiquitous, and full of
  quirks — including the RTL-SDR Blog V4 "deafness" bug that gets its own post.
- **Airspy R2 / Mini and Airspy HF+** — higher-dynamic-range receivers. The R2 and
  Mini stream **real** samples that we convert to complex baseband in Go; the HF+
  is a separate family tuned for HF/VHF. Different register map, different sample
  math.
- **HackRF One** (and Jawbreaker / Rad1o) — a half-duplex transceiver that streams
  **signed 8-bit** IQ. We only ever drive it in receive, but the command set is a
  full transceiver state machine.

All of them satisfy the same `Device` interface, so the engine can't tell an
R820T from an Airspy HF+ from a recorded capture. Reference entries:
[RTL-SDR]({{ '/reference/rtl-sdr/' | relative_url }}),
[HackRF]({{ '/reference/hackrf/' | relative_url }}),
[Airspy]({{ '/reference/airspy/' | relative_url }}).

Beyond the three USB families there are two more `Device` implementations worth
knowing exist, because they prove the abstraction holds. The `rtltcp` and
`soapyremote` backends mount *network* SDRs — a dongle on a Raspberry Pi at the
antenna, or professional gear like a USRP, LimeSDR, or bladeRF reached over the
SoapyRemote protocol — as if they were local hardware, all in pure Go with no
CGO. And the `mock` driver replays recorded IQ captures back into the pool as a
virtual tuner. None of these are USB at all, yet every one is a `Device`, which is
the clearest evidence that the contract in Part 2 is drawn at the right level: the
engine asks for IQ, and it does not care whether the IQ came from an R820T, a
network socket, or a WAV file on disk.

## How GopherTrunk implements it in Go

The whole series hangs off one interface in `internal/sdr/device.go`. Every
backend, regardless of silicon, hides behind it:

```go
// internal/sdr/device.go
type Device interface {
    Info() Info
    SetCenterFreq(hz uint32) error
    SetSampleRate(hz uint32) error
    SetGain(tenthDB int) error // -1 selects automatic gain control
    SetPPM(ppm int) error
    SetBiasTee(enable bool) error
    StreamIQ(ctx context.Context) (<-chan []complex64, error)
    Close() error
}
```

`StreamIQ` is the heart of it: give it a `context.Context`, get back a channel of
IQ chunks; cancel the context and the stream stops. The rest of the engine only
ever sees this interface — it never imports `rtlsdr`, `hackrf`, or `airspy`. The
concrete drivers live in sub-packages (`internal/sdr/rtlsdr/purego`,
`internal/sdr/airspy`, `internal/sdr/airspyhf`, `internal/sdr/hackrf`), and each
one registers itself with the package-global registry at `init()`. Part 2 takes
the contract apart method by method; Part 3 takes the registry apart.

The setters carry conventions that let one interface stand in for very different
silicon. `SetGain(tenthDB int)` takes gain in tenths of a dB, with `-1` as a
sentinel that selects automatic gain control — one integer expresses both "manual"
and "let the tuner decide," so the engine never needs a separate AGC method that
only some radios would honor. `SetBiasTee(enable bool)` toggles the 5V bias-tee
that powers an external LNA through the antenna SMA; on dongles without the
circuit it is required to silently no-op and `return nil`, so the engine can call
it unconditionally. These conventions are the small print of a narrow contract,
and Part 2 is largely about why they are drawn the way they are.

## The design principle: one narrow contract, many implementations

The architectural bet of the RF front end is the same bet the whole engine makes,
applied to hardware: **depend on a small abstract contract, never on a concrete
device.** The engine asks for "a thing that streams IQ and can be tuned." It does
not ask for "an RTL-SDR." Everything chip-specific lives behind the interface,
and the chip-specific *quirks* live behind *optional* interfaces the caller
type-asserts for (the subject of Part 2).

### How that principle shaped the Go code

- **One interface, four radios.** `Device` is the only type the engine knows.
  Adding the Airspy HF+ didn't change a line of engine code — it added a package
  that satisfies `Device`.
- **Quirks stay optional.** RTL-SDR-only behavior (Blog V4 override, tuner
  diagnosis) lives in `TunerDiagnoser` and `BlogV4Forcer`, optional extensions
  callers type-assert for — so the universal contract stays universal.
- **The binary's import set picks the hardware.** `cmd/gophertrunk`
  blank-imports the real drivers; their `init()` functions populate the registry.
  Change the imports, change the supported hardware — no engine edit.
- **Pure Go, no CGO, end to end.** USB transport, register protocols, IQ
  decode — all Go, all `CGO_ENABLED=0`, all cross-compilable.

## The series map

| Part | Topic | Problem solved |
|---|---|---|
| 1 | Why drive radios in pure Go? (this post) | The CGO/libusb tax |
| [2]({{ '/blog/deep-dives/rf-front-end-02-the-device-contract/' | relative_url }}) | The Device contract | One interface, four radios |
| [3]({{ '/blog/deep-dives/rf-front-end-03-driver-registry-enumeration/' | relative_url }}) | The driver registry & enumeration | Hardware-agnostic core |
| [4]({{ '/blog/deep-dives/rf-front-end-04-usb-without-libusb/' | relative_url }}) | Talking to USB without libusb | No C USB stack |
| [5]({{ '/blog/deep-dives/rf-front-end-05-usb-on-linux-usbdevfs/' | relative_url }}) | USB on Linux: USBDEVFS & async URBs | High-throughput transfers |
| [6]({{ '/blog/deep-dives/rf-front-end-06-usb-on-macos-windows/' | relative_url }}) | USB on macOS & Windows | Cross-platform transport |
| [7]({{ '/blog/deep-dives/rf-front-end-07-rtlsdr-rtl2832u-bringup/' | relative_url }}) | RTL-SDR I: bringing up the RTL2832U | The register dance |
| [8]({{ '/blog/deep-dives/rf-front-end-08-rtlsdr-r82xx-blog-v4/' | relative_url }}) | RTL-SDR II: the R82xx tuner & Blog V4 deafness | A mistuned LO |
| [9]({{ '/blog/deep-dives/rf-front-end-09-rtlsdr-streaming-gc-churn/' | relative_url }}) | RTL-SDR III: IQ streaming & the GC-churn bug | Allocation pressure |
| [10]({{ '/blog/deep-dives/rf-front-end-10-airspy-real-to-complex/' | relative_url }}) | Airspy R2/Mini & HF+: real samples to complex baseband | Real-to-IQ conversion |
| [11]({{ '/blog/deep-dives/rf-front-end-11-hackrf-one/' | relative_url }}) | HackRF One: signed 8-bit IQ end to end | Transceiver state machine |
| [12]({{ '/blog/deep-dives/rf-front-end-12-sdr-pool-usb-watchdog/' | relative_url }}) | The SDR pool & USB hotplug watchdog | Self-healing on disconnect |
| [13]({{ '/blog/deep-dives/rf-front-end-13-testing-radios-without-radios/' | relative_url }}) | Testing radios without radios | CI with no hardware |
| [14]({{ '/blog/deep-dives/rf-front-end-14-diagnostics-metrics-payoff/' | relative_url }}) | Diagnostics, metrics & the pure-Go payoff | Observability + the payoff |

Each post is an overview — enough to understand the component, the Go that
implements it, and the principle behind it. The pipeline below the RF front end
is the subject of the sister series,
[SDR Internals]({{ '/blog/series/sdr-internals/' | relative_url }}); this series
is the layer that feeds it.

## Where this goes next

[Part 2]({{ '/blog/deep-dives/rf-front-end-02-the-device-contract/' | relative_url }})
opens up the `Device` interface itself — every method, the `-1`-means-AGC gain
convention, the bias-tee no-op rule, and the tension of expressing four very
different radios through one contract without letting any one radio's quirks bleed
into the others. From there the series descends through the stack: the registry
and enumeration (Part 3), the USB transport that has to replace `libusb` on three
operating systems (Parts 4-6), then the chip drivers one family at a time — the
RTL2832U bring-up and its R82xx tuner, including the Blog V4 deafness bug and the
GC-churn bug in its streaming path (Parts 7-9), Airspy's real-to-complex
conversion (Part 10), and HackRF's signed-8-bit transceiver (Part 11). The last
three posts step back up: the pool and hotplug watchdog that supervise a fleet of
dongles (Part 12), how all of this is tested with no radios attached (Part 13),
and the diagnostics, metrics, and the pure-Go payoff that ties the series off
(Part 14).

## FAQ

**Why not just link librtlsdr and move on?**
Linking `librtlsdr` reintroduces CGO and a runtime shared-library dependency,
which breaks the single-static-binary promise and turns cross-compilation into a
C-toolchain problem. Writing the USB transport and chip protocols in Go keeps the
build pure Go and the binary self-contained.

**Is the RF front end just USB plumbing?**
The USB transport is the bottom of it (Parts 4-6), but each driver is also a small
chip-protocol implementation — the RTL2832U bring-up, the R82xx tuner math,
HackRF's transceiver commands, Airspy's real-to-complex conversion. The plumbing
is necessary; the protocols are where the radios differ.

**Do I have to read this in order?**
No. Part 1 is the map and each later part stands alone. But the layers stack —
USB transport under the chip drivers under the pool — so top-to-bottom mirrors how
a sample actually climbs out of the hardware.

## Series navigation

**Part 1 of 14** · Next →
[Part 2: The Device contract]({{ '/blog/deep-dives/rf-front-end-02-the-device-contract/' | relative_url }})
