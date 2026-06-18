---
title: "RF Front End, Part 12: The SDR Pool & USB Hotplug Watchdog"
description: How GopherTrunk manages a fleet of opened dongles behind one pool — assigning control, voice, and wideband roles, running a USB watchdog that re-enumerates every 30 seconds to catch hotplug, and reacquiring a device after the kernel hands it a new bus address. Plus the re-open race that taught us to serialize retune behind stream teardown.
category: deep-dives
tags: [sdr, go, usb, concurrency, software-design]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "RF Front End"
series_part: 12
---

*Part 12 of **RF Front End**. We've spent eleven posts inside a single dongle —
USB transport, the RTL2832U register dance, tuners, sample conversion. Now we
zoom out to the fleet: how the daemon holds several radios at once, gives each a
job, and keeps them alive across the USB drops that real hardware throws at you.*

## In this post

- The **pool** (`internal/sdr/pool.go`) — a fleet of opened devices, each with a
  **role**: control, voice, or wideband.
- **Strict / allowlist mode** and **serial-alias matching** so an operator's
  `config.yaml` selects exactly the dongles they named.
- The **USB watchdog** that re-enumerates every ~30 s, publishes
  `KindSDRAttached` / `KindSDRDetached`, and **reacquires** a device after the
  kernel re-enumerates it with a new address.
- **The re-open race (issue #686)**: a scanner retune raced USB stream teardown,
  and how serializing behind an idempotent stop fixed it.

## What the pool does

A single dongle is one `Device`. A trunked system needs more than one: a radio
camped on the control channel decoding grants, and one or more radios that follow
those grants onto voice frequencies. A wideband Airspy might cover a whole site's
worth of channels at once. The pool is the thing that owns all of them.

Its job is narrow but load-bearing. At boot it enumerates every registered
driver, opens the devices the operator selected, programs a known-good sample
rate on each, and assigns each one a **role**. After boot it answers a single
question for the rest of the engine — "give me the device with role X" — and it
keeps that fleet healthy while USB does what USB does: drop a stick mid-stream,
re-enumerate it under a new device number, and expect the software to cope.

Roles matter because the engine never reaches for a *specific* dongle. The
control-channel decoder asks for `RoleControl`; the voice composer asks the pool
to find a `RoleVoice` device by serial when the engine binds a call. That
indirection is what lets the same code run on a one-stick hobby setup and a
four-stick site without a branch anywhere in the engine.

## How GopherTrunk implements it in Go

A `Pool` is a slice of opened entries behind a mutex, plus an optional event bus:

```go
// internal/sdr/pool.go
type Pool struct {
    mu      sync.RWMutex
    entries []*PoolEntry
    log     *slog.Logger
    bus     *events.Bus
}

type PoolEntry struct {
    Driver Driver
    Device Device
    Info   Info
    Role   Role
    Hint   Hint
}
```

`OpenWith` is the heart of bring-up. It sweeps every registered driver, opens the
selected devices, programs the IQ rate, and assigns roles. Role assignment is one
simple rule: the first opened device that isn't otherwise claimed takes
`RoleControl`; everything after it defaults to `RoleVoice`. A `Hint` can override
that per serial.

```go
// internal/sdr/pool.go (shape)
role := RoleAuto
if hinted {
    role = hint.Role
}
if role == RoleAuto {
    if !controlClaimed {
        role = RoleControl
        controlClaimed = true
    } else {
        role = RoleVoice
    }
}
```

Programming the sample rate at open time isn't optional housekeeping — it's a
fix for issue #275. Without an explicit `SetSampleRate`, the chip streams at
whatever rate its resampler powered up at, while the decoder runs its
symbol-timing math against the *configured* rate. The result is a silent failure
to lock, the worst kind of bug in a radio. So a device whose `SetSampleRate`
fails is closed and skipped: a wrong-rate radio is worse than no radio at all.

### Strict mode and serial aliases

By default the pool opens every dongle it finds. The moment an operator lists
specific devices in config, that's their signal that they want *only* those —
so the daemon engages **strict mode**, where `Hints` becomes an allowlist:

```go
// internal/sdr/pool.go (shape)
if opts.Strict && !hinted {
    p.log.Info("skipping non-configured SDR; add its serial to sdr.devices to use it",
        "driver", d.drv.Name(), "serial", d.info.Serial)
    continue
}
```

Matching a hint to a device means matching serials, and serials aren't always
clean. Airspy reports a legacy form — `AIRSPY SN:35ac63dc2d701c4f` — that an
operator might write a dozen ways. `serialKey` normalizes them so the config and
the wire agree:

```go
// internal/sdr/pool.go
func serialKey(s string) string {
    s = strings.TrimSpace(s)
    s = strings.ToLower(s)
    switch {
    case strings.HasPrefix(s, "airspy sn:"):
        return strings.TrimPrefix(s, "airspy sn:")
    case strings.HasPrefix(s, "airspy_sn:"):
        return strings.TrimPrefix(s, "airspy_sn:")
    default:
        return s
    }
}
```

`TestPoolMatchesAirspySerialAliases` pins this: a hint written
`AIRSPY SN:35ac63dc2d701c4f` opens the device whose raw serial is
`35AC63DC2D701C4F`, and `FindBySerial` resolves all three spellings to the same
entry.

### The USB watchdog

The pool also runs a supervisor loop. `RunWatchdog` ticks every interval — 30
seconds by default — re-enumerates every driver, and acts only on *transitions*:

```go
// internal/sdr/watchdog.go
const DefaultWatchdogInterval = 30 * time.Second

func (p *Pool) RunWatchdog(ctx context.Context, interval time.Duration, sampleRateHz uint32) error {
    if interval <= 0 {
        <-ctx.Done()
        return ctx.Err()
    }
    tick := time.NewTicker(interval)
    defer tick.Stop()

    missing := map[string]bool{}
    for {
        select {
        case <-ctx.Done():
            return ctx.Err()
        case <-tick.C:
            p.watchdogTick(missing, sampleRateHz)
        }
    }
}
```

The `missing` map is the state machine. A pool serial that the enumerate stops
seeing flips to missing and emits one `KindSDRDetached` — the API, TUI, and web
snapshot all show the gap. When that same serial reappears in a later enumerate,
the watchdog deletes it from `missing` and calls `Reacquire`:

```go
// internal/sdr/watchdog.go (shape)
if missing[serial] {
    delete(missing, serial)
    p.log.Info("sdr: watchdog: device reappeared; reacquiring", "serial", serial)
    if _, err := p.Reacquire(serial, sampleRateHz); err != nil {
        p.log.Warn("sdr: watchdog: reacquire failed", "serial", serial, "err", err)
    }
}
```

`Reacquire` is where the hotplug story gets real. When a dongle browns out and
comes back, the kernel assigns it a new device number — but it reports the same
serial. So `Reacquire` closes the (likely dead) handle best-effort, re-enumerates
the driver, finds the serial under its *new* index, opens a fresh handle,
re-programs the rate, and re-applies the original `Hint` (PPM, gain, bias-tee).
Crucially it swaps the `Device` **in place** on the existing `PoolEntry` —
`Role`, serial identity, and any pointer a consumer is holding all survive; only
`Info.Index` updates to the new enumeration. `TestPoolReacquireSwapsDeviceHandleInPlace`
asserts exactly that: same `PoolEntry`, new `*fakeDevice`, stale handle closed,
bias-tee re-applied, index refreshed to 7.

## The problem we hit: the retune-vs-teardown re-open race (issue #686)

The watchdog handles the *idle* case — a device nobody is streaming. The in-use
case is harder, and it bit us in scanner mode.

In scanner mode a fast retune cancels the IQ stream's context and immediately
re-opens it on the new frequency. But USB drivers don't tear a stream down
synchronously — the bulk-IN reaper goroutine runs `cancelStream` asynchronously,
draining URBs and closing the consumer channel on its own schedule. So the
sequence that should have been "stop, then start" became "start while the
previous stop is still in flight." The second `StreamIQ` found the bulk-IN
endpoint still claimed and failed with `stream already active` — surfacing to the
operator as `conv: StreamIQ failed` and a dead capture.

The race was structural, not a missing lock. The teardown path is idempotent via
a `sync.Once`:

```go
// internal/sdr/rtlsdr/purego/device.go
func (d *Device) cancelStream() {
    d.stopOnce.Do(func() {
        _ = d.transport.StopBulkIn()
        // ...close the consumer channel
    })
}
```

`stopOnce` guarantees teardown runs exactly once — but it didn't guarantee the
*next* `StreamIQ` waited for it. The fix was to make re-open serialize behind the
in-flight teardown: a new stream resets `stopOnce` only after the previous stop
has actually completed, so a retune can never out-run the reaper.

```go
// internal/sdr/rtlsdr/purego/stream.go (shape)
out := make(chan []complex64, streamChanDepth)
d.out = out
d.stopOnce = sync.Once{} // only reachable once the prior teardown finished
```

The lesson is a recurring one in this series: with USB, "stop" is a *request*,
not an event. Anything that re-opens has to wait on the teardown completing, not
on having asked for it.

## The design principle: supervisor + observer

Two patterns share the load here. The pool is a **supervisor** (a fleet manager):
it owns the lifecycle of every device, restarts the ones that fail, and presents
the survivors as a roster the engine can query by role. The watchdog is the
supervisor's health check, and `Reacquire` is its restart strategy.

The second pattern is **observer**. The pool never calls into the daemon, the
API, or the TUI. It `Publish`es `KindSDRAttached` / `KindSDRDetached` to an
optional bus and lets whoever cares subscribe:

```go
// internal/sdr/pool.go
func (p *Pool) publish(kind events.Kind, payload any) {
    if p.bus == nil {
        return
    }
    p.bus.Publish(events.Event{Kind: kind, Payload: payload})
}
```

### How that principle shaped the Go code

- **The bus is optional.** `NewPool` takes only a logger; `SetBus` is a separate,
  idempotent step. The `gophertrunk sdr list` CLI and every unit test run the
  pool with `bus == nil`, and `publish` short-circuits — the same fleet code, no
  daemon required.
- **State lives in one goroutine.** The watchdog's `missing` map is owned solely
  by the watchdog goroutine and passed in by value-reference, so attach/detach
  transitions need no extra lock. Only the pool's `entries` slice is shared, and
  that's behind `sync.RWMutex`.
- **Identity is stable across reacquisition.** Because `Reacquire` swaps the
  `Device` inside an existing `PoolEntry` rather than replacing the entry,
  consumers that cached a `*PoolEntry` keep working across a USB cycle. Role and
  serial are the identity; the handle is just an attribute.
- **Recovery is best-effort and idempotent.** Closing a dead handle may error;
  re-enumerate may miss the serial; the in-stream retry loop may beat the
  watchdog to it. Every path logs and moves on, because the next tick — or the
  next consumer — will try again.

## Where this goes next

The pool assigns roles and keeps devices alive, but we've leaned on tests
throughout this post — `TestPoolReacquireSwapsDeviceHandleInPlace`,
`TestPoolMatchesAirspySerialAliases` — without explaining how you test a fleet of
radios in CI where there are no radios at all. That's
[Part 13]({{ '/blog/deep-dives/rf-front-end-13-testing-radios-without-radios/' | relative_url }}):
replaying captured USB control-transfer sequences, bit-identical conversion
golden masters, and an opt-in real-hardware tier.

## FAQ

**Why poll every 30 seconds instead of listening for kernel hotplug events?**
Polling is portable. The same re-enumerate loop works on Linux USBDEVFS, Windows
WinUSB, and macOS IOKit without three platform-specific hotplug listeners. 30 s
is short enough to recover a transient drop inside one failure cycle and long
enough not to load a slow hub.

**What happens to an in-use device that drops?** The watchdog owns the *idle*
case. A device that's actively streaming surfaces its death through the stream
itself — the reaper closes the channel, the consumer (ccdecoder retry loop,
`VoicePool.Bind`) sees EOF and drives its own `Reacquire`. The watchdog is the
backstop for radios nobody is currently touching.

**Why does strict mode skip a device that's physically present?** Because an
allowlist is an allowlist, not a preference. If you named your control stick in
config and an unrelated dongle is on the bus, opening that dongle could let it
win `RoleControl` and bind the decoder to a radio that never got your PPM
correction — the original issue #264 failure. Strict mode refuses to guess.

## Series navigation

**Part 12 of 14** · ← [Part 11]({{ '/blog/deep-dives/rf-front-end-11-hackrf-one/' | relative_url }}) · Next → [Part 13: Testing radios without radios]({{ '/blog/deep-dives/rf-front-end-13-testing-radios-without-radios/' | relative_url }})
