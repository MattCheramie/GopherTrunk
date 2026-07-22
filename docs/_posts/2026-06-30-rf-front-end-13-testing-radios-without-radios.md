---
title: "RF Front End, Part 13: Testing Radios Without Radios"
description: How GopherTrunk's USB driver code is tested in CI with no hardware attached — replaying captured control-transfer sequences through a scripted mock transport, pinning the pure-Go sample conversion bit-for-bit against librtlsdr with a golden-master test, and an opt-in real-hardware tier gated behind environment flags with bias-tee safety resets.
category: deep-dives
tags: [sdr, go, testing, usb, software-design]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "RF Front End"
series_part: 13
---

*Part 13 of **RF Front End**. We've described eleven drivers and a pool that
herds them, and at every turn we've cited a test that proves it works. This post
answers the question those citations dodge: how do you trust a USB driver you
can't run on real hardware in CI?*

> **TL;DR** — GopherTrunk tests its USB drivers with no hardware attached by replaying captured control-transfer sequences through a scripted mock transport and pinning the pure-Go sample conversion bit-for-bit against librtlsdr with a golden master. An opt-in real-hardware tier, gated behind env flags and `-short` with bias-tee safety resets, confirms against actual silicon when you have it.

## In this post

- The **`MockTransport`** — a scripted control-transfer replayer that lets the
  RTL-SDR driver run its full bring-up with no dongle present.
- The **bit-identical conversion test** (`TestConvertU8IQ_BitIdenticalWithCGO`)
  that pins the pure-Go U8→`complex64` lookup table against librtlsdr's math.
- The **pool tests** for role assignment, strict mode, and the watchdog.
- The **opt-in real-hardware tier**, guarded by `GOPHERTRUNK_AIRSPY_REAL=1`,
  `-short`, hard timeouts, and a bias-tee safety reset.

## What "testing a radio" actually means

A driver is a contract negotiation. The RTL-SDR bring-up isn't "open the device"
— it's dozens of vendor control transfers in a precise order: reset the demod,
poke the RTL2832U registers, walk the R820T's PLL through its lock sequence, set
the sample-rate divider. Each transfer has a `bRequest`, a `wValue`, a `wIndex`,
and either a payload (OUT) or an expected reply (IN). Get one wrong and the chip
either rejects it or — worse — accepts it and streams garbage.

That's actually good news for testing, because a contract is *recordable*. The
real conversation between librtlsdr (or the chip's firmware) and the hardware is
a deterministic sequence of bytes on the wire. Capture it once, and you can replay
it forever in CI. The driver doesn't know it's talking to a recording — it issues
the same control transfers it would to silicon, and something on the other end
either matches the script or fails the test.

So GopherTrunk's testing rests on three legs: **replay captured wire contracts**
so the protocol logic is exercised, **a golden-master conversion test** so the
DSP-facing output is byte-for-byte correct, and **an opt-in real-hardware tier**
that smoke-tests against actual silicon when you have it plugged in.

<figure class="lab-figure">
<svg viewBox="0 0 650 170" width="650" height="170" role="img" aria-label="Every driver depends on the Transport interface, not on the OS USB stack. At that seam two peers are interchangeable: the real platform transport backed by USBDEVFS, WinUSB, or IOKit talking to silicon, and MockTransport, which replays a captured control-transfer Script with no hardware present. The driver cannot tell them apart.">
  <rect x="8" y="66" width="130" height="46" rx="6" fill="none" stroke="currentColor"/>
  <text x="73" y="86" text-anchor="middle" fill="currentColor" font-size="11">driver</text>
  <text x="73" y="100" text-anchor="middle" fill="var(--fg-muted)" font-size="9">rtl2832u · tuners</text>
  <line x1="138" y1="89" x2="184" y2="89" stroke="currentColor"/><polygon points="184,85 194,89 184,93" fill="currentColor"/>
  <rect x="194" y="64" width="140" height="50" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="264" y="86" text-anchor="middle" fill="var(--accent)" font-size="11">Transport</text>
  <text x="264" y="101" text-anchor="middle" fill="var(--fg-muted)" font-size="9">interface (the seam)</text>
  <line x1="334" y1="78" x2="388" y2="46" stroke="currentColor"/><polygon points="384,42 394,44 389,53" fill="currentColor"/>
  <line x1="334" y1="100" x2="388" y2="132" stroke="currentColor"/><polygon points="389,125 394,135 384,137" fill="currentColor"/>
  <rect x="394" y="24" width="248" height="42" rx="6" fill="none" stroke="currentColor"/>
  <text x="518" y="43" text-anchor="middle" fill="currentColor" font-size="10">real transport → silicon</text>
  <text x="518" y="57" text-anchor="middle" fill="var(--fg-muted)" font-size="9">USBDEVFS · WinUSB · IOKit</text>
  <rect x="394" y="112" width="248" height="42" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="518" y="131" text-anchor="middle" fill="var(--accent)" font-size="10">MockTransport → Script</text>
  <text x="518" y="145" text-anchor="middle" fill="var(--fg-muted)" font-size="9">replays captured transfers · no hardware</text>
</svg>
<figcaption>The <code>Transport</code> interface is the seam: the real platform transport and <code>MockTransport</code> are interchangeable peers, so the driver runs its full bring-up against a replayed <code>Script</code> with no hardware present.</figcaption>
</figure>

## How GopherTrunk implements it in Go

### The scripted mock transport

The `Transport` interface every driver speaks to has a test double:
`MockTransport`. It holds a `Script` — an ordered list of expected exchanges —
and matches each call against the next entry:

```go
// internal/sdr/rtlsdr/usb/usb_mock.go
type CtrlExchange struct {
    In        bool
    BRequest  uint8
    WValue    uint16
    WIndex    uint16
    Data      []byte // expected for OUT, ignored for IN
    Reply     []byte // returned for IN, ignored for OUT
    Err       error
    N         int
    TimeoutOK bool
}
```

When the driver issues a `ControlOut`, the mock pops the next script entry and
checks everything — direction, request, value, index, and the exact payload
bytes — recording a descriptive error on any mismatch:

```go
// internal/sdr/rtlsdr/usb/usb_mock.go (shape)
if want.BRequest != bRequest || want.WValue != wValue || want.WIndex != wIndex {
    err := fmt.Errorf("mock step %d: ControlOut mismatch: got (req=0x%02x,val=0x%04x,idx=0x%04x), ...")
    m.recordErr(err)
    return err
}
if !bytesEqual(want.Data, data) {
    err := fmt.Errorf("mock step %d: ControlOut data mismatch: got %#x, want %#x", m.Step-1, data, want.Data)
    m.recordErr(err)
    return err
}
```

**A test populates the script with the captured sequence, runs the driver's
bring-up, and asserts `MockTransport.Remaining() == 0` — every expected transfer
happened, in order, with the right bytes.** The mock even models the messy parts:
`ResetCalls` counts `USBDEVFS_RESET` invocations so a test can prove the recovery
path ran, `ClaimErr` injects an `EBUSY` so the claim-failure branch is exercised,
and `BulkSimulateDeath` fires `onStreamDead` after delivering its packets so the
issue #345 reaper-death path is covered without unplugging anything.

<figure class="lab-figure">
<svg viewBox="0 0 660 176" width="660" height="176" role="img" aria-label="How MockTransport verifies a bring-up: the driver issues a ControlOut, the mock pops the next CtrlExchange from its ordered Script and compares direction, bRequest, wValue, wIndex, and the exact payload bytes; a match advances the step, a mismatch records a descriptive error, and after bring-up the test asserts Remaining equals zero, proving every expected transfer happened in order.">
  <rect x="8" y="62" width="140" height="46" rx="6" fill="none" stroke="currentColor"/>
  <text x="78" y="82" text-anchor="middle" fill="currentColor" font-size="10">driver ControlOut</text>
  <text x="78" y="96" text-anchor="middle" fill="var(--fg-muted)" font-size="9">req · val · idx · data</text>
  <line x1="148" y1="85" x2="184" y2="85" stroke="currentColor"/><polygon points="184,81 194,85 184,89" fill="currentColor"/>
  <rect x="194" y="62" width="140" height="46" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="264" y="82" text-anchor="middle" fill="var(--accent)" font-size="10">pop CtrlExchange</text>
  <text x="264" y="96" text-anchor="middle" fill="var(--fg-muted)" font-size="9">next Script step</text>
  <line x1="334" y1="85" x2="370" y2="85" stroke="currentColor"/><polygon points="370,81 380,85 370,89" fill="currentColor"/>
  <rect x="380" y="62" width="140" height="46" rx="6" fill="none" stroke="currentColor"/>
  <text x="450" y="82" text-anchor="middle" fill="currentColor" font-size="10">compare fields</text>
  <text x="450" y="96" text-anchor="middle" fill="var(--fg-muted)" font-size="9">+ exact bytes</text>
  <line x1="520" y1="74" x2="566" y2="42" stroke="currentColor"/><polygon points="562,38 572,40 567,49" fill="currentColor"/>
  <text x="548" y="30" text-anchor="middle" fill="var(--fg-muted)" font-size="9">match</text>
  <rect x="566" y="24" width="86" height="30" rx="5" fill="none" stroke="var(--accent)"/>
  <text x="609" y="43" text-anchor="middle" fill="var(--accent)" font-size="10">advance</text>
  <line x1="520" y1="96" x2="566" y2="128" stroke="currentColor"/><polygon points="567,121 572,131 562,133" fill="currentColor"/>
  <text x="546" y="150" text-anchor="middle" fill="var(--fg-muted)" font-size="9">mismatch</text>
  <rect x="566" y="116" width="86" height="30" rx="5" fill="none" stroke="var(--fg-muted)"/>
  <text x="609" y="135" text-anchor="middle" fill="var(--fg-muted)" font-size="10">recordErr</text>
  <text x="264" y="130" text-anchor="middle" fill="var(--fg-muted)" font-size="9">after bring-up: assert Remaining() == 0</text>
</svg>
<figcaption><code>MockTransport</code> replays a captured <code>Script</code>: each driver transfer is matched against the next <code>CtrlExchange</code> — direction, request, value, index, and exact bytes — and <code>Remaining() == 0</code> proves every expected transfer happened in order.</figcaption>
</figure>

The same file ships a `MockEnumerator` so even discovery is fakeable — it returns
a fixed set of descriptors and opens each as a `MockTransport`, which is how the
higher layers (rtl2832u, the tuners) run end-to-end with no bus.

### The bit-identical conversion test

Replaying control transfers proves the *protocol* is right. It says nothing about
the sample math — the conversion of raw 8-bit IQ bytes into the `complex64` the
whole DSP chain consumes. That math is a port of librtlsdr's C, and a port can
drift. So GopherTrunk pins it with a golden master.

The production conversion uses a 256-entry lookup table, one float32 per possible
byte value, built from the exact expression librtlsdr uses:

```go
// internal/sdr/rtlsdr/purego/stream.go (shape)
// Each entry uses ((b-127.5)/127.5), the exact float32 expression as the
// original per-sample math, so the result is bit-identical — pinned by
// TestConvertU8IQ_BitIdenticalWithCGO.
var u8ToF32 [256]float32
```

`TestConvertU8IQ_BitIdenticalWithCGO` writes that C-side math out longhand as an
*oracle* — a reference implementation deliberately written the naive,
unoptimized way — and asserts the production LUT path produces identical bytes:

```go
// internal/sdr/rtlsdr/purego/stream_test.go
func referenceConvertU8IQ(buf []byte) []complex64 {
    n := len(buf) / 2
    out := make([]complex64, n)
    for i := 0; i < n; i++ {
        out[i] = complex(
            (float32(buf[2*i])-127.5)/127.5,
            (float32(buf[2*i+1])-127.5)/127.5,
        )
    }
    return out
}
```

The point of the oracle isn't to re-test the formula — it's to catch *drift*. The
moment someone "optimizes" the LUT and changes a result in the last bit of the
mantissa, this test fails and names the offending byte. Companion tests pin the
physically meaningful cases: 127/128 brackets the chip's 127.5 DC bias and must
land near zero; 0 and 255 must scale to −1 and +1; and
`TestConvertU8IQInto_ZeroAllocMatchesAllocating` proves the zero-allocation
hot-path variant (issue #489) matches the allocating one sample for sample.

### The pool tests

The pool from [Part 12]({{ '/blog/deep-dives/rf-front-end-12-sdr-pool-usb-watchdog/' | relative_url }})
gets the same treatment with an even lighter fake: a `fakeDriver` returning a
fixed `[]Info` and a `fakeDevice` that records the calls made to it. With that,
the fleet logic is fully testable. `TestPoolAssignsRoles` checks that a
control-hinted serial wins `RoleControl` and the rest fall to voice;
`TestPoolStrictOpensOnlyHintedSerial` proves strict mode opens only the allowlisted
dongle and skips the others with a log line; and the watchdog tests swap the
fake's `Enumerate` return between calls to simulate an unplug/replug:

```go
// internal/sdr/watchdog_test.go (shape)
// Pull the device off the bus, then tick.
drv.infos = nil
missing := map[string]bool{}
p.watchdogTick(missing, 0)
// assert KindSDRDetached published exactly once, idempotent on the next tick.
```

None of these touch USB. The `fakeDevice` just flips booleans —
`biasTeeOn`, `sampleRate`, `ppm` — and the tests assert against them, so the
entire supervisor and recovery logic is verified deterministically in
milliseconds.

## The problem we hit: trusting a driver you can't run in CI

**Symptom.** Here's the uncomfortable truth the three legs are built to address. CI runs on a
cloud box with no SDR plugged in. We can prove the control-transfer *sequence*
matches a capture, and we can prove the conversion math is bit-identical to
librtlsdr — but a capture is a snapshot of one dongle's firmware on one day. What
if a chip revision reorders its init? What if a real R820T rejects a transfer the
mock happily accepts because the mock only checks the bytes, not the silicon's
state machine?

**Root cause.** Replay and golden masters give *necessary* confidence, not *sufficient*. So
**the fix** is a fourth thing: an **opt-in real-hardware tier** that runs only when you
ask it to, against the dongle on your desk. The Airspy suite is the template. Every
real-hardware test starts with a guard:

```go
// internal/sdr/airspy/airspy_real_test.go
func requireRealAirspy(t *testing.T) {
    t.Helper()
    v := strings.TrimSpace(os.Getenv(airspyRealEnv))
    if v == "" || v == "0" || strings.EqualFold(v, "false") {
        t.Skipf("set %s=1 to run real Airspy hardware validation", airspyRealEnv)
    }
    if testing.Short() {
        t.Skip("skipping real Airspy hardware validation in -short mode")
    }
}
```

So `go test ./...` in CI skips them silently; `GOPHERTRUNK_AIRSPY_REAL=1 go test`
on a workstation with hardware runs them. They're bounded by a context timeout so
a dead radio fails fast instead of hanging the suite:

```go
// internal/sdr/airspy/airspy_real_test.go (shape)
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()
iq, err := dev.StreamIQ(ctx)
// assert a non-empty packet arrives before ctx.Done().
```

And they clean up after themselves — the bias-tee test puts a safety reset in
`t.Cleanup` so the suite never leaves DC voltage on the antenna port, which can
damage a passive antenna or a downstream LNA:

```go
// internal/sdr/airspy/airspy_real_test.go
t.Cleanup(func() {
    // Best-effort safety reset: leave bias-tee off when the test exits.
    _ = dev.SetBiasTee(false)
    _ = dev.Close()
})
```

There are even raw USB probe tests behind their own sub-flags
(`GOPHERTRUNK_AIRSPY_REAL_DIAG=1`) that sweep candidate `wIndex` values to
discover what the firmware actually accepts — the tool you reach for when adding
support for a chip whose control map you don't yet have captured.

## The design principle: captured contracts + golden master + an opt-in tier

The unifying idea is to **test against the contract, not the hardware** — and to
keep the hardware available as a separate, opt-in confirmation. **Each leg answers
a different question:**

- The **mock replay** asks: does the driver speak the protocol correctly?
- The **golden-master conversion** asks: is the math byte-for-byte right?
- The **fake-driver pool tests** ask: does the fleet logic behave?
- The **opt-in real tier** asks: does it actually work on silicon?

### How that principle shaped the Go code

- **The transport is an interface, so the seam is free.** Drivers depend on
  `Transport`, not on USBDEVFS. `MockTransport` and the real platform transports
  are peers; the driver can't tell them apart. No mocking framework, no
  monkey-patching — just a struct with the right methods.
- **Golden masters pin behavior, not implementation.** The oracle in
  `stream_test.go` is intentionally slow and obvious. Its only job is to fail
  loudly if the fast production path ever diverges, so the LUT and the
  zero-alloc variant can be optimized fearlessly.
- **Hardware tests are off by default and self-protecting.** Env guards plus
  `testing.Short()` keep CI green and fast; context timeouts bound them;
  `t.Cleanup` safety resets mean a real-hardware run can't leave the radio in a
  dangerous state.
- **It's all pure-Go `go test`.** No CGO, no `LD_LIBRARY_PATH`, no librtlsdr to
  install on the CI runner. The same `go test ./...` that builds the binary tests
  the drivers — which is exactly the payoff we close the series on next.

## Where this goes next

We can prove the front end is correct. The finale,
[Part 14]({{ '/blog/deep-dives/rf-front-end-14-diagnostics-metrics-payoff/' | relative_url }}),
turns from *correctness* to *observability* — the `sdr list --probe` tooling, the
`RTLSDR_DEBUG_USB` trace, the `iq_underruns_total` metric that finally made
silent IQ loss visible — and then collects the whole pure-Go thesis the series
has been building toward.

## FAQ

**Where do the captured control-transfer scripts come from?** From a known-good
reference: librtlsdr's `rtl_test` traced under `LIBUSB_DEBUG=4`, or the
`RTLSDR_DEBUG_USB` trace (next post) from GopherTrunk's own driver against real
hardware. The captured sequence becomes the `Script` the mock replays.

**Isn't a golden-master test brittle?** Deliberately. The conversion math is a
contract with librtlsdr — every existing SDR tool's output assumes it. "Brittle"
here means "any change to a load-bearing formula is caught immediately," which is
the property you want for something this far upstream of every decoder.

**How do you add a driver for a chip you've never captured?** Start with the raw
USB probe tests — sweep `wIndex` candidates against the real device behind a diag
flag to learn what it accepts, capture the working sequence, then encode it as a
mock script so CI can replay it forever.

## Series navigation

**Part 13 of 14** · ← [Part 12]({{ '/blog/deep-dives/rf-front-end-12-sdr-pool-usb-watchdog/' | relative_url }}) · Next → [Part 14: Diagnostics, metrics & the pure-Go payoff]({{ '/blog/deep-dives/rf-front-end-14-diagnostics-metrics-payoff/' | relative_url }})
