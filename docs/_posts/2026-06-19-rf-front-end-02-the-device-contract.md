---
title: "RF Front End, Part 2: The Device Contract"
description: A method-by-method tour of GopherTrunk's Device interface — the one abstraction four very different radios hide behind — and the optional, type-asserted extension interfaces that expose RTL-SDR-only quirks without polluting the universal contract.
category: deep-dives
tags: [sdr, go, interfaces, software-design, rtl-sdr]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "RF Front End"
series_part: 2
---

*Part 2 of **RF Front End**. One interface stands between the whole engine and
four very different radios. We take the `Device` contract apart method by method,
and look at the trick that lets one radio's quirks ride along without leaking into
the others: optional, type-asserted extension interfaces.*

> **TL;DR** — The `Device` interface is the single narrow contract every radio hides behind. The trick: RTL-SDR-only quirks (like the Blog V4 deafness fix) would pollute that universal contract if added to it, so they live in optional, type-asserted extension interfaces instead.

## In this post

- The **`Device` interface** in full — `Info`, the four setters, the AGC and
  bias-tee conventions, and `StreamIQ`.
- The **"one interface, four radios"** tension and why the contract stays narrow.
- The **optional extension interfaces** (`TunerDiagnoser`, `BlogV4Forcer`) that
  callers type-assert for — interface segregation in practice.
- **The problem we hit:** exposing RTL-SDR-only quirks without polluting the
  universal contract.

## What the Device contract is

`Device` is the per-dongle handle. Every backend — RTL-SDR, Airspy R2/Mini,
Airspy HF+, HackRF, plus the network and mock drivers — implements it, and it is
the *only* SDR type the rest of the engine ever sees. Here it is in full, from
`internal/sdr/device.go`:

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

Eight methods. That's the entire surface the engine is allowed to touch. The
discipline is deliberate: the narrower this contract, the more radios can satisfy
it honestly, and the less chip-specific knowledge leaks upward.

The comment above the type spells out the concurrency rule that makes streaming
work: implementations must be safe for the goroutines that call `StreamIQ`, and
"concurrent `SetCenterFreq` during streaming is allowed (the underlying USB
transport handles it)." That one line is why the trunking engine can retune a
control channel without tearing down the IQ stream.

It is worth dwelling on *why* the contract is this small, because narrowness is a
design choice, not an accident. Four radios that share almost nothing at the
silicon level — an RTL2832U demodulator fronted by an R820T, an Airspy with its
own register map, a HackRF transceiver, an Airspy HF+ tuned for the low bands —
have to be interchangeable from the engine's point of view. The only way that
works is to find the *intersection* of what they can all do and make that the
contract: tune, set rate, set gain, set PPM, maybe toggle a bias-tee, stream,
close. Anything one radio can do that another cannot is, by definition, *not* in
`Device`. The interface is small because the common ground is small, and the
discipline to keep it that way is what lets a recorded WAV file or a network
socket satisfy the same eight methods as a $25 dongle.

## How GopherTrunk implements it in Go

### Identity: `Info`

`Info()` returns a value, not a pointer, describing what the dongle *is*:

```go
// internal/sdr/device.go
type Info struct {
    Driver       string
    Index        int
    Serial       string
    Manufacturer string
    Product      string
    TunerName    string
    Gains        []int
}
```

The same `Info` struct is what a driver's enumeration returns *before* a device is
opened (Part 3), so the engine can list and pick hardware without committing to a
USB handle. `Gains` carries the discrete gain steps a tuner actually supports —
the engine snaps a requested gain onto the nearest legal step rather than guessing.
Returning a value rather than a pointer is deliberate: `Info` is an immutable
description, and handing back a copy means a caller can stash it, compare it, or
log it with no chance of it mutating under them while the device streams. The
`Serial` and `Index` fields together let the pool address a specific dongle even
when several identical RTL-SDRs are plugged into the same hub — serial is the
stable identity, index is the bus position, and the pool prefers serial so a
reordered USB tree doesn't reshuffle which dongle plays which role.

### The four setters, and their conventions

`SetCenterFreq`, `SetSampleRate`, `SetGain`, and `SetPPM` all take a plain integer
and return an `error`. Two of them carry conventions worth calling out, because
they are how the contract absorbs differences between radios instead of exposing
them:

- **Gain: `-1` means AGC.** `SetGain(tenthDB int)` takes gain in tenths of a dB,
  and a sentinel `-1` selects automatic gain control. One integer parameter
  expresses both "manual, this many tenths of a dB" and "let the tuner decide,"
  so the engine never needs a separate `SetAGC(bool)` method that only some
  radios would honor.
- **Bias-tee: silent no-op when absent.** Many dongles have a 5V bias-tee that
  powers an external LNA through the antenna SMA; many don't. Rather than splitting
  `Device` into "has bias-tee" and "doesn't," the contract requires every
  implementation to accept `SetBiasTee`. The doc comment is explicit: "Devices
  without the circuit silently no-op. Implementations should return `nil` if the
  underlying driver doesn't model bias-tee at all." A capability that only some
  hardware has is modeled as a method everyone implements, where "I can't" is
  spelled `return nil`.

**These conventions are the load-bearing idea of a narrow contract: the *differences*
between radios are pushed into agreed-upon values and no-ops, so the *interface*
stays the same for all of them.**

### The stream: `StreamIQ`

```go
StreamIQ(ctx context.Context) (<-chan []complex64, error)
```

Pass a `context.Context`, get back a receive-only channel of `[]complex64`
chunks. Each chunk is a few milliseconds of baseband; the engine ranges over the
channel until the context is cancelled, at which point the driver closes the
channel and tears down its USB transfers. Returning `<-chan` (receive-only) is a
small but real piece of the contract — the consumer can read but cannot close or
send, so ownership of the channel's lifecycle stays with the driver that knows how
to drain the USB side cleanly.

The `context` argument is the cancellation half of that ownership story. The
engine never reaches into the driver to say "stop"; it cancels the context it
handed to `StreamIQ`, the driver observes the cancellation, stops submitting USB
transfers, drains the ones in flight, and closes the channel. The consumer's
`for chunk := range ch` loop ends naturally when the channel closes. No shared
"stop" flag, no second method, no risk of the consumer closing a channel the
driver is still writing to — the context flows down, the channel and its chunks
flow up, and the lifecycle has exactly one owner at each end. This is the same
pipes-and-filters-over-CSP shape the DSP layer uses, applied at the very bottom of
the stack where the bytes first become samples.

## The problem we hit: RTL-SDR quirks vs. the universal contract

The narrow contract works beautifully right up until a radio has a quirk no other
radio shares — and the RTL-SDR Blog V4 has two.

**Symptom.** Some Blog V4 dongles come up *deaf*: tuned to a known-good control
channel, they hear nothing, or hear a signal mistuned by a factor of ~1.8×. The
boot diagnostics also wanted to surface, per-dongle, which tuner was detected and
what reference crystal it latched onto — information that simply does not exist for
an Airspy or a HackRF.

**Root cause.** The R828D tuner in the Blog V4 uses a 28.8 MHz reference crystal
and a switched HF/VHF/UHF input bank. Auto-detection keys off the USB EEPROM
strings; when a V4's strings are blank or non-standard, detection misses it, the
R828D stays on the wrong 16 MHz crystal, and the local oscillator mistunes
(issue #264). Fixing it needs two RTL-SDR-only capabilities: **read** the tuner's
detected state for diagnostics, and **force** Blog V4 mode regardless of what the
USB strings say.

The tempting-but-wrong fix is to add `TunerDiag()` and `SetBlogV4()` to `Device`.
That would force every Airspy and HackRF driver to implement two methods about a
crystal they don't have — polluting the universal contract with one vendor's
hardware detail.

**The Go fix.** Put the quirks in *optional* interfaces that callers type-assert
for, and leave `Device` untouched:

```go
// internal/sdr/device.go
type TunerDiagnoser interface {
    TunerDiag() (tunerName string, blogV4, blogV4Lite bool, xtalHz uint32)
}

type BlogV4Forcer interface {
    SetBlogV4(lite bool) error
}
```

Only the RTL-SDR driver implements these. A caller that wants them asks at runtime:

```go
if d, ok := dev.(sdr.BlogV4Forcer); ok {
    _ = d.SetBlogV4(lite) // RTL-SDR only; everything else doesn't satisfy this
}
```

The type assertion succeeds for the RTL-SDR driver and fails — cleanly, with `ok
== false` — for every other backend. The Blog V4 fix reaches exactly the hardware
that needs it, and the Airspy and HackRF drivers never hear about a 28.8 MHz
crystal. The doc comments on both interfaces even encode the failure signature:
"`xtalHz` of 16 MHz on an R828D is the signature that RTL-SDR Blog V4
auto-detection missed and the LO is mistuned by ~1.8×."

`TunerDiagnoser` works the same way for the read side. Boot-time diagnostics
type-assert for it to surface, per dongle, which tuner was detected, whether it
came up in Blog V4 (or V4 Lite) mode, and what reference crystal it latched onto —
the four return values of `TunerDiag()`. A backend that has no tuner crystal to
report simply doesn't implement the interface, the assertion fails, and the
diagnostics skip it. Nothing forces an Airspy to answer a question about a crystal
it doesn't have. The whole Blog V4 story — detect the mistune via
`TunerDiagnoser`, correct it via `BlogV4Forcer` — is wired through two tiny
optional interfaces and zero changes to `Device`. Part 8 of this series follows
that thread all the way down into the R82xx tuner to show what the forced mode
actually reprograms.

## The design principle: interface segregation

The lesson is the **interface segregation principle**: no client should be forced
to depend on methods it does not use. `Device` is the small contract *every* radio
needs. `TunerDiagnoser` and `BlogV4Forcer` are tiny contracts only the radios with
those capabilities satisfy — and only the callers who care go looking for them.

### How that principle shaped the Go code

- **The universal contract stays universal.** `Device` holds the eight methods
  every SDR honors. Nothing RTL-SDR-specific lives there.
- **Capabilities are discovered, not assumed.** Optional behavior is a runtime
  `dev.(Interface)` type assertion, so a backend opts in simply by implementing
  the extra interface — no flags, no capability enums.
- **Extensions are additive.** A new optional interface (say, a future
  calibration hook) is a new tiny `interface` plus the one driver that implements
  it. No existing driver changes, because none of them are forced to.
- **The narrowest possible coupling.** A caller depends on `BlogV4Forcer` — one
  method — not on the concrete RTL-SDR package. The dependency is on exactly the
  capability used, nothing more.
- **The compiler enforces it for free.** Because Go interfaces are satisfied
  structurally, a driver "joins" an optional interface simply by having the right
  method — and the type assertion that finds it is a compile-checked, allocation-
  free runtime test. There is no capability registry to keep in sync and no enum
  to extend; the method set *is* the capability declaration.

## Where this goes next

The contract tells you *what* a device does, but not *which* devices exist or
*how* the engine gets a handle to one. That's the registry's job.
[Part 3]({{ '/blog/deep-dives/rf-front-end-03-driver-registry-enumeration/' | relative_url }})
covers the self-registering driver registry and the enumeration walk that lists
every attached dongle across every backend — and how it stays resilient when one
driver's enumeration fails.

## FAQ

**Why is gain a single `int` with a `-1` sentinel instead of a richer type?**
Because it keeps the contract narrow. One integer parameter expresses "manual, this
many tenths of a dB" and "AGC" without a second method that only some radios would
implement. The `Gains` slice in `Info` carries the legal manual steps for snapping.

**Why must `SetBiasTee` exist on radios that have no bias-tee?**
So the engine can call it unconditionally. A capability some hardware lacks is
modeled as a method everyone implements, where "not applicable" is `return nil`.
The alternative — a separate `BiasTeeCapable` interface — would make the common
case (just toggle it) require a type assertion.

**How does a caller know whether a device supports Blog V4 forcing?**
It type-asserts: `dev.(sdr.BlogV4Forcer)`. The assertion succeeds only for the
RTL-SDR driver and returns `ok == false` for everything else, so the capability is
discovered at runtime with no central registry of who-can-do-what.

## Series navigation

**Part 2 of 14** · ←
[Part 1]({{ '/blog/deep-dives/rf-front-end-01-why-pure-go-drivers/' | relative_url }})
· Next →
[Part 3: The driver registry & enumeration]({{ '/blog/deep-dives/rf-front-end-03-driver-registry-enumeration/' | relative_url }})
