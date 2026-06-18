---
title: "RF Front End, Part 3: The Driver Registry & Enumeration"
description: How GopherTrunk's self-registering driver registry keeps the core hardware-agnostic — drivers register at init(), the binary's import set chooses the hardware, and EnumerateAll walks every backend to list attached dongles while staying resilient when one driver fails.
category: deep-dives
tags: [sdr, go, registry, dependency-inversion, software-design]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "RF Front End"
series_part: 3
---

*Part 3 of **RF Front End**. The engine needs a driver but must not depend on
one. We cover the self-registering registry that inverts that dependency, the
blank imports that choose the hardware, and the enumeration walk that lists every
attached dongle without one flaky backend blanking out the rest.*

## In this post

- The **self-registering driver registry** — a `sync.RWMutex`-guarded map and the
  `init()`-time `Register` call.
- **`DriverByName`** for picking a backend and **`EnumerateAll`** for walking
  every backend to discover hardware.
- **The problem we hit:** enumeration must survive one driver erroring without
  hiding everyone else's devices.
- **The design principle:** registry + dependency inversion — and where it recurs
  in the codebase.

## What the registry does

The `Device` contract from
[Part 2]({{ '/blog/deep-dives/rf-front-end-02-the-device-contract/' | relative_url }})
tells you what a radio can do. The registry answers the prior question: *which
drivers exist, and how does the engine reach one without importing it?* It is a
process-global map from driver name to `Driver`, the factory each backend exposes:

```go
// internal/sdr/device.go
type Driver interface {
    Name() string
    Enumerate() ([]Info, error)
    Open(idx int) (Device, error)
}
```

`Name()` is the registry key. `Enumerate()` lists the devices that driver can see
right now. `Open(idx)` turns one of those into a live `Device`. The engine talks
to the registry in these abstract terms and never names a concrete driver type.

Notice the two-phase split between `Enumerate` and `Open`. Enumeration is cheap
and read-only: it probes the bus and returns `[]Info` describing what's attached,
without claiming a USB handle or starting a stream. Opening is the expensive,
exclusive step — it takes a USB device for the daemon's lifetime. Keeping them
separate is what lets the engine *list* every dongle across every backend for the
TUI or the web console, let the operator assign roles, and only then `Open` the
specific devices it actually needs. A `Driver` is a factory you can poll freely; a
`Device` is a resource you commit to.

## How GopherTrunk implements it in Go

### The map and the lock

The whole registry is a guarded map plus a handful of accessors in
`internal/sdr/registry.go`:

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
```

`sync.RWMutex` rather than a plain `Mutex` because registration happens once, at
startup, but reads (`Drivers`, `DriverByName`, every `EnumerateAll`) happen
throughout the daemon's life — the read lock lets those run concurrently.

### Self-registration at `init()`

Each driver registers itself from its package `init()` function, so importing the
package is all it takes to make the driver available:

```go
// internal/sdr/rtlsdr/purego/register.go
func init() { sdr.Register(&Driver{}) }
```

Nothing calls `init()` explicitly — Go runs it when the package is first imported.
This is the linchpin of the whole pattern: registration is a *side effect of
importing*, which means the act of choosing to include a driver in the build is
also the act of making it available at runtime. There is no separate "plugin
loading" step, no configuration file listing drivers, no reflection scanning for
implementations. The Go linker does the work — a package that nobody imports is
not in the binary, and a package that is imported has already registered itself by
the time `main` runs.

The only place those driver packages are imported is `cmd/gophertrunk`, with blank
imports purely for their side effect:

```go
// cmd/gophertrunk/main.go
import (
    _ "github.com/MattCheramie/GopherTrunk/internal/sdr/airspy"
    _ "github.com/MattCheramie/GopherTrunk/internal/sdr/airspyhf"
    _ "github.com/MattCheramie/GopherTrunk/internal/sdr/hackrf"
    _ "github.com/MattCheramie/GopherTrunk/internal/sdr/rtlsdr/purego"
)
```

That import block *is* the hardware-support list. Want a build that only talks to
RTL-SDR? Drop three lines. The engine doesn't change; the binary's import set does.

This is more than a tidy trick — it is the mechanism by which the engine stays
genuinely ignorant of its drivers. The `internal/sdr` package, which defines
`Device`, `Driver`, and the registry, does *not* import the driver sub-packages;
the dependency runs the other way, with each driver importing `internal/sdr` to
call `Register`. If the core imported the drivers, it would know their concrete
types, and the whole abstraction would be a fiction. The blank import in
`cmd/gophertrunk` is the *only* edge that connects a concrete driver to the
running program, and it lives at the very top of the dependency graph — in `main`,
where it belongs — rather than buried in the engine.

### Looking up and enumerating

Two reads drive everything downstream. `DriverByName` picks a backend by its
registry key:

```go
// internal/sdr/registry.go
func DriverByName(name string) (Driver, error) {
    registryMu.RLock()
    defer registryMu.RUnlock()
    d, ok := registry[name]
    if !ok {
        return nil, fmt.Errorf("sdr: unknown driver %q", name)
    }
    return d, nil
}
```

And `EnumerateAll` walks *every* registered driver to build the full list of
attached hardware — covered next, because that walk is where a real design
decision lives.

## The problem we hit: one flaky driver hid every dongle

**Symptom.** A user with a working RTL-SDR and a half-supported second backend
ran device discovery and got back an *empty* list — no devices at all, even though
a perfectly good dongle was plugged in.

**Root cause.** Enumeration walked the drivers and one of them returned an error
from `Enumerate()` — a backend whose platform USB enumeration hiccuped. The walk
treated that single error as fatal: it bailed on the first failure and returned no
devices. One flaky backend blanked out every other driver's perfectly enumerable
hardware. Worse, because the failing call returned an empty-but-no-error result up
the chain, the operator saw "no SDRs found" with no hint that a driver had
actually failed.

**The Go fix.** `EnumerateAll` aggregates *per driver*: it collects devices from
every driver that succeeds and collects one error per driver that fails, returning
both. A failure in one backend is reported, not propagated as global emptiness:

```go
// internal/sdr/registry.go
func EnumerateAll() ([]Info, []error) {
    var out []Info
    var errs []error
    for _, d := range Drivers() {
        infos, err := d.Enumerate()
        if err != nil {
            errs = append(errs, err)
            continue
        }
        out = append(out, infos...)
    }
    return out, errs
}
```

The `continue` is the whole fix: a driver that errors contributes an entry to
`errs` and is skipped, while every other driver still appends its devices to
`out`. The doc comment states the contract directly — it returns "the combined
device list plus one error per driver that failed to enumerate, so callers can
surface the failure instead of silently reporting an empty list." The operator now
sees their RTL-SDR *and* a precise note about which backend failed and why.

`Drivers()` itself returns the backends sorted by name, so enumeration order — and
the device list the operator sees — is deterministic across runs rather than
dependent on Go's randomized map iteration:

```go
// internal/sdr/registry.go
func Drivers() []Driver {
    registryMu.RLock()
    defer registryMu.RUnlock()
    out := make([]Driver, 0, len(registry))
    for _, d := range registry {
        out = append(out, d)
    }
    sort.Slice(out, func(i, j int) bool { return out[i].Name() < out[j].Name() })
    return out
}
```

Without that `sort.Slice`, two runs of the same binary could list the same dongles
in different orders, and an operator scripting against device indices would get
non-reproducible results. The sort costs nothing at startup scale and buys a
stable, predictable enumeration.

## The design principle: registry + dependency inversion

This is **dependency inversion**. The high-level engine depends on the abstract
`Driver`/`Device` contracts; the low-level drivers depend *up* on those same
contracts by registering themselves into the registry. The dependency arrow points
the "wrong" way on purpose — toward the abstraction, from both sides.

### How that principle shaped the Go code

- **Blank imports choose the hardware.** `cmd/gophertrunk` blank-imports the four
  real drivers; their `init()` functions populate the registry. The supported-
  hardware list is the import list.
- **The engine enumerates, it doesn't instantiate.** It calls `Drivers()` to see
  what's available and `DriverByName()` to pick one. There is no
  `switch deviceType { case "rtlsdr": ... }` anywhere in the core.
- **New hardware is purely additive.** A new backend is a package that satisfies
  `Device`/`Driver` and calls `Register` in its `init()`. No existing file changes
  — the same hook the baseband-replay "virtual tuner" uses to mount recorded
  captures as if they were dongles.
- **Failures are localized, not fatal.** Because the registry walks drivers
  independently, `EnumerateAll` degrades gracefully: one bad backend costs you that
  backend's devices and an error entry, never the whole list.
- **The lock matches the access pattern.** `sync.RWMutex` lets the many readers —
  every `Drivers`, `DriverByName`, and `EnumerateAll` over the daemon's life — run
  concurrently, while the rare `Register` writes at startup take the exclusive
  lock. The synchronization is right-sized to "write once, read forever."

This registry-plus-inversion shape is one of the most reused ideas in the
codebase. It is the same pattern that drives the vocoders in
[SDR Internals]({{ '/blog/series/sdr-internals/' | relative_url }})
[Part 12]({{ '/blog/deep-dives/sdr-internals-12-voice-coding-vocoders/' | relative_url }})
— self-registering plugins behind a shared core, chosen by the binary's import
set. Once you've seen it for SDR drivers and vocoders you start to recognize it as
GopherTrunk's default answer to "how does the core get an implementation it must
not depend on": a small interface, a guarded map, an `init()`-time registration,
and a blank import in `main` that decides what's compiled in. The RF front end is
where the pattern is easiest to see, because the things being registered —
physical radios — are the most concrete, but the mechanism is identical wherever
it recurs.

## Where this goes next

We now have a contract (Part 2) and a way to discover and open devices (this
post). Everything below that is the hard part: how a `Device` actually moves bytes
without `libusb`.
[Part 4]({{ '/blog/deep-dives/rf-front-end-04-usb-without-libusb/' | relative_url }})
starts the USB transport story — issuing control transfers and bulk reads against
the kernel directly, in pure Go.

## FAQ

**Why a process-global registry instead of passing drivers in explicitly?**
So the engine never imports a concrete driver. Drivers self-register at `init()`,
and the binary's blank-import set decides what's available. The core depends only
on the abstract `Driver`/`Device` contracts — classic dependency inversion.

**What happens if two drivers register the same name?**
The map keys on `Name()`, so the later `Register` wins. In practice names are
unique per backend; the deterministic ordering operators see comes from `Drivers()`
sorting by name before returning.

**Why does `EnumerateAll` return `[]error` instead of a single error?**
Because each driver fails independently and the caller wants to know *which* one
did. Aggregating per-driver lets a flaky backend report its failure without hiding
the devices every other backend enumerated successfully.

## Series navigation

**Part 3 of 14** · ←
[Part 2]({{ '/blog/deep-dives/rf-front-end-02-the-device-contract/' | relative_url }})
· Next →
[Part 4: Talking to USB without libusb]({{ '/blog/deep-dives/rf-front-end-04-usb-without-libusb/' | relative_url }})
