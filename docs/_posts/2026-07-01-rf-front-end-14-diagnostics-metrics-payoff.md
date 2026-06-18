---
title: "RF Front End, Part 14: Diagnostics, Metrics & the Pure-Go Payoff"
description: "The finale of RF Front End. How GopherTrunk makes the front end observable — sdr list --probe, the RTLSDR_DEBUG_USB control-transfer trace, and the iq_underruns_total metric that turned silent IQ loss into something an operator can see — and then the payoff the whole series argued for: one static binary, CGO_ENABLED=0, no librtlsdr, no libusb, cross-compiled everywhere."
category: deep-dives
tags: [sdr, go, observability, architecture, software-design]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "RF Front End"
series_part: 14
---

*Part 14 of **RF Front End**, the finale. Thirteen posts took a $25 dongle and
turned it into a clean stream of IQ samples in pure Go. Now: how we *see* that
front end when it misbehaves — and the payoff that justified writing every USB
driver from scratch in the first place.*

> **TL;DR** — The finale makes the front end observable: `sdr list --probe`, the `RTLSDR_DEBUG_USB` control-transfer trace, and the `iq_underruns_total` metric that turned silent IQ loss — drops that looked exactly like an RF problem — into a counter operators can watch. The payoff: one static binary, `CGO_ENABLED=0`, no librtlsdr or libusb, cross-compiled everywhere.

## In this post

- The **diagnostic surfaces**: `sdr list --probe`, the `RTLSDR_DEBUG_USB`
  control-transfer trace, and how drop counters reach operators.
- The **`iq_underruns_total` metric** (issue #402) that made silent IQ loss
  visible — and the `NotifyIQDrop` hook behind it.
- **The problem we hit**: silent IQ loss was indistinguishable from RF problems
  until the metric existed.
- **The payoff**: one static binary, `CGO_ENABLED=0`, no librtlsdr / libusb /
  libhackrf / libairspy, cross-compiled to Linux, macOS, and Windows on amd64 and
  arm64 — and a recap of all fourteen parts.

## What observing the front end means

A radio fails quietly. A decoder that won't lock could be a bad antenna, the
wrong gain, a PPM error, a dying dongle, or — the subtle one — a host that can't
keep up, dropping IQ chunks on the floor before they ever reach the demod. From
the operator's chair every one of those looks identical: *no decode.* The job of
front-end observability is to make those causes distinguishable, ideally before
someone spends an afternoon swapping antennas to chase a CPU problem.

GopherTrunk attacks that on three levels. At the **CLI**, `sdr list --probe`
opens each device and reports what it really is. At the **wire**, the
`RTLSDR_DEBUG_USB` trace dumps every control transfer in a format you can diff
against librtlsdr. And at **runtime**, Prometheus metrics turn invisible
sample-loss into a counter that climbs.

## How GopherTrunk implements it in Go

### Probe at the CLI

**`gophertrunk sdr list` enumerates the bus cheaply. Add `--probe` and it opens
each device long enough to run the demod and tuner detection, so the listing
shows the real tuner name and gain ladder instead of a guess:**

```
gophertrunk sdr list [--probe]   list discovered SDR devices
                                 (--probe opens each to fill TUNER + gains)
```

This is also where the issue #264 tuner diagnostics surface. When a driver
implements the optional `TunerDiagnoser` interface, the pool logs a self-diagnosing
line at open time:

```go
// internal/sdr/pool.go (shape)
name, v4, v4lite, xtal := td.TunerDiag()
p.log.Info("sdr tuner detected",
    "serial", info.Serial, "tuner", name,
    "blog_v4", v4, "ref_xtal_hz", xtal)
```

A `ref_xtal_hz=16000000` on an R828D means RTL-SDR Blog V4 auto-detection missed
and the LO will mistune by ~1.8×; `28800000` means the V4 path armed. The
blank/non-standard EEPROM string that *causes* the miss is echoed on the same
line, so a single boot log tells you what went wrong and why.

### Trace at the wire

When a dongle rejects a control transfer, you need to see the conversation.
Setting `RTLSDR_DEBUG_USB` wraps the transport in a logger that prints every
`ControlIn` / `ControlOut` / `Reset`, in a format diffable against osmocom
librtlsdr's `rtl_test` under `LIBUSB_DEBUG=4`:

```go
// internal/sdr/rtlsdr/usb/usb_debug.go
func MaybeWrapDebug(t Transport, desc Descriptor) Transport {
    if os.Getenv(debugUSBEnv) == "" {
        return t
    }
    return newDebugTransport(t, desc)
}
```

Wrapping is gated at Open time, so the off-path stays free — the sample-stream
hot path is never touched, only control transfers are. Each line carries a
monotonic transfer ID, a timestamp, the request fields, and the payload hex:

```
rtlsdr-usb [serial=...]: tx=000042 ControlOut bmReqType=0x40 bReq=0x00 wValue=0x0034 wIndex=0x0610 wLength=17 timeout=300ms
rtlsdr-usb [serial=...]: tx=000042 data=83 32 75 c0 ...
rtlsdr-usb [serial=...]: tx=000042 -> ok (after 1.2ms)
```

There's a CSV mode (`RTLSDR_DEBUG_USB_CSV`) with a fixed column order for diffing
against USBPcap/Wireshark exports — the same traces that become the mock scripts
from [Part 13]({{ '/blog/deep-dives/rf-front-end-13-testing-radios-without-radios/' | relative_url }}).
The debug sink is redirectable (`SetDebugSink`) so tests can assert on it. One
diffable channel, from a developer's terminal to a CI assertion.

### Count at runtime

Probes and traces are things you reach for when you already suspect trouble. The
metric is what tells you trouble exists. `iq_underruns_total` counts every IQ
chunk a driver discards because the primary consumer fell behind:

```go
// internal/metrics/prom.go
m.iqUnderruns = prometheus.NewCounterVec(prometheus.CounterOpts{
    Namespace: namespace,
    Subsystem: "sdr",
    Name:      "iq_underruns_total",
    Help:      "Times the IQ stream pipeline dropped samples because a downstream stage was too slow.",
}, []string{"driver", "serial"})
```

The wiring between a driver's reaper goroutine and that counter is a single
process-wide hook, `NotifyIQDrop`. SDR backends self-register with zero-value
drivers, so there's no constructor seam to thread a per-device sink through —
hence an atomic pointer set once by the daemon during bring-up:

```go
// internal/sdr/iqdrop.go
func NotifyIQDrop(info Info) {
    if obs := iqDropObserver.Load(); obs != nil {
        (*obs)(info)
    }
}
```

A driver calls `NotifyIQDrop(info)` from its overrun branch; the daemon's
installed observer increments `iq_underruns_total{driver,serial}`. Tools and
tests that don't wire metrics leave the observer nil and the call is a no-op.
There's a sibling counter, `ccdecoder_decode_overruns_total`, that distinguishes
*where* the bottleneck is: a climbing `decode_overruns` with `iq_underruns` flat
says "the machine can't sustain this decode," while a climbing `iq_underruns`
says "the USB transport hiccuped." Two counters, one diagnosis.

## The problem we hit: silent IQ loss looked exactly like an RF problem

This is the bug that drove the entire issue #402 investigation, and it's worth
sitting with because it shaped the whole observability design.

**Symptom.** Before any of these counters existed, a busy site would intermittently fail to
decode. The captures the reporter sent us replayed *perfectly* — every TSBK
decoded green. But live, on the same hardware, the control channel kept dropping.
The tell, once we had it, was damning: **live fails, replay is green.** Replay
reads a file at its own pace; live has to keep up with a 2.4 MS/s firehose in
real time. If anything downstream stalled — a momentarily-busy consumer, a GC
pause, a contended lock — the driver's delivery channel overran and chunks were
**silently dropped on the floor.**

**Root cause.** Silent is the operative word. Those drops produced no log, no error, no signal of
any kind. To the operator it was indistinguishable from a weak signal or a bad
antenna — so the natural response was to chase RF: raise gain, swap antennas,
re-aim. None of which touched the actual cause, because the actual cause was the
host falling behind.

**The fix** was to make the loss *observable* before trying to make it *rare*. Issue
#486 surfaced the previously-silent drops via `NotifyIQDrop` → `iq_underruns_total`;
issue #507 then decoupled live IQ ingest from decode with a forwarder goroutine
and a deeper bounded queue so the drops became rare. But the metric came first,
on purpose: you can't fix what you can't see, and you certainly can't tell an
operator "this is a CPU problem, not an antenna problem" without a number to point
at. Once `iq_underruns_total` was climbing in front of them, the diagnosis became
a glance instead of an afternoon.

## The design principle: observability is a feature — and the pure-Go thesis, vindicated

Two principles close the series.

The first is that **observability is a first-class feature, not an afterthought.**
The drop counter isn't instrumentation bolted on; it's the thing that makes the
front end *operable*. A radio you can't see into is a radio you can only guess
about, and guessing is how operators waste hours on the wrong layer.

The second is the thesis this entire series argued: **pure Go, one static binary.**
Every driver in fourteen posts — RTL-SDR, Airspy, HackRF, the USB transports on
three operating systems — is written in Go, talking to the kernel directly. No
librtlsdr, no libusb, no libhackrf, no libairspy. Which means:

```
CGO_ENABLED=0 go build ./cmd/gophertrunk
```

produces **one file**. No shared libraries to install. No driver-install hell, no
`LD_LIBRARY_PATH`, no "works on my machine because I have the right libusb."
`GOOS`/`GOARCH` cross-compile it to Linux, macOS, and Windows on amd64 and arm64
from a single developer laptop. The same source that's readable from antenna to
audio is also the same source CI tests with `go test ./...` — because, as
[Part 13]({{ '/blog/deep-dives/rf-front-end-13-testing-radios-without-radios/' | relative_url }})
showed, there's nothing to link.

### How that principle shaped the Go code

- **The drop hook costs nothing when unused.** `NotifyIQDrop` is one atomic
  pointer load on the hot path. No observer installed, no overhead — the CLI and
  tests pay zero for a daemon-only feature.
- **Debug wrapping is gated at Open.** The `RTLSDR_DEBUG_USB` transport only
  exists when the env var is set; the production path is never wrapped, so the
  sample firehose stays untouched.
- **Diagnostics ride the same interfaces.** `TunerDiagnoser`, `Diagnoser`, and
  the metric hooks are all optional capabilities a driver may implement. A backend
  that doesn't simply contributes no extra diagnostics — no base class, no forced
  boilerplate.
- **No CGO means no exceptions.** Because the whole stack is pure Go, the build,
  the cross-compile, and the test suite have no caveats. There is no "but on
  Windows you also need…" — and that absence is the entire payoff.

## Where this goes next — and the journey across fourteen parts

That's the full RF source layer, end to end:

- **Why pure Go, and the USB transport** that makes it possible — kernel USBDEVFS
  / WinUSB / IOKit, no libusb (Parts 1–3).
- **The RTL2832U and its tuners** — the register dance, the R820T PLL math,
  sample-rate dividers, and bit-identical U8→`complex64` conversion (Parts 4–9).
- **The wider front end** — Airspy, HackRF, and the rest of the fleet
  (Parts 10–11).
- **The fleet and how we trust it** — the pool's roles and USB watchdog, testing
  radios without radios, and the diagnostics and metrics that close the loop
  (Parts 12–14).

Every block is pure Go; every block rests on a software-design principle —
dependency inversion, supervisor/observer, captured-contract testing,
observability-as-a-feature — and every principle shaped the actual code.

This series was the deep dive into one layer that its sister series only skimmed.
If you arrived here from the broader pipeline story, the
[SDR Internals]({{ '/blog/series/sdr-internals/' | relative_url }}) series picks
up where the IQ samples leave the front end and travels all the way to recovered
audio. Or start this journey over from
[Part 1]({{ '/blog/deep-dives/rf-front-end-01-why-pure-go-drivers/' | relative_url }})
— and grab a build to watch a $25 dongle decode a trunked system from a single
static binary you didn't have to install anything to run.

## FAQ

**Why expose drops as a metric instead of just logging them?** A log line per
drop on a busy site is a flood that hides the signal. A counter aggregates: the
*rate* of `iq_underruns_total` is the diagnosis, and Prometheus already knows how
to alert on a rate. The daemon emits a rate-limited warning alongside, so you get
both the number and a nudge.

**How do I tell a CPU problem from an RF problem?** Watch the two counters. A
climbing `iq_underruns_total` (or `ccdecoder_decode_overruns_total`) with a
healthy `iq_power_dbfs` says the host is the bottleneck, not the antenna. Flat
drop counters with weak power says go fix your RF.

**What did pure Go actually buy the end user?** One static binary, no
shared-library install, reproducible builds, and cross-compilation to every
desktop platform from one machine. The same `go build` that ships the daemon also
ships the CLI, the TUI, and the embedded web console — with nothing to link and
nothing to install.

## Series navigation

**Part 14 of 14** · ← [Part 13]({{ '/blog/deep-dives/rf-front-end-13-testing-radios-without-radios/' | relative_url }}) · Back to [Part 1: Why drive radios in pure Go?]({{ '/blog/deep-dives/rf-front-end-01-why-pure-go-drivers/' | relative_url }})
