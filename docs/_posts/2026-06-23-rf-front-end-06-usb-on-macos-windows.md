---
title: "RF Front End, Part 6: USB on macOS & Windows"
description: How GopherTrunk reaches the same Transport contract through macOS IOKit (via purego, with OS-thread-pinned reader goroutines) and Windows WinUSB (with overlapped I/O), and why macOS forced us to own the calling thread.
category: deep-dives
tags: [sdr, go, usb, cross-platform, software-design]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "RF Front End"
series_part: 6
---

*Part 6 of **RF Front End**. We finish the USB adapters: macOS through IOKit
loaded by purego with no CGO, Windows through WinUSB with overlapped I/O. Three
operating systems, three completely different threading and I/O models — all
folded back onto the one eight-method `Transport` contract from Part 4.*

> **TL;DR** — These are the macOS (IOKit via purego) and Windows (WinUSB,
> overlapped I/O) USB backends, both satisfying the same eight-method `Transport`
> port as Linux. The headache: IOKit ties USB I/O state to the issuing OS thread,
> so early macOS streaming builds crashed — fixed by pinning each reader to its
> own thread with `runtime.LockOSThread`.

## In this post

- The **macOS** backend: IOKit + CoreFoundation via **purego** (no CGO), a lazy
  `sync.Once` framework load, and **one OS-thread-pinned goroutine per slot**.
- The **Windows** backend: lazily-loaded **WinUSB** function pointers and
  **overlapped (async) I/O** drained with `WaitForMultipleObjects`.
- How both reach the **same `Transport`** that Linux does — and where they
  diverge under the hood.
- The problem of **macOS IOKit owning the calling thread**, and how
  `runtime.LockOSThread` fixed the aborts it caused.

## What these two backends do

Both are *adapters* for the `Transport` port from
[Part 4]({{ '/blog/deep-dives/rf-front-end-04-usb-without-libusb/' | relative_url }}):
same `ControlIn`/`ControlOut`, same `ClaimInterface`, same
`StartBulkIn`/`StopBulkIn` callbacks. The RTL2832U driver above them can't tell
which one it's talking to. Underneath, they could hardly be more different from
Linux's USBDEVFS, or from each other.

macOS has no usbfs. You reach USB through **IOKit** (the device-driver registry
and the `IOUSBDeviceInterface`/`IOUSBInterfaceInterface` C++ vtables) and
**CoreFoundation** (for the dictionaries and strings IOKit speaks in). Windows has
no usbfs either; you bind your device to the in-box **WinUSB** function driver
(via Zadig, typically) and call into `winusb.dll`, with device enumeration coming
from `setupapi.dll`. Both are normally consumed from C. GopherTrunk consumes them
from pure Go — purego on macOS, lazy DLL procs on Windows — to keep the
CGO-free, single-static-binary promise intact on every OS.

<figure class="lab-figure">
<svg viewBox="0 0 660 210" width="660" height="210" role="img" aria-label="The RTL2832U driver talks to one eight-method Transport port, which is satisfied by three interchangeable operating-system adapters underneath: Linux USBDEVFS ioctls with an async URB ring, macOS IOKit vtables via purego with N pinned reader threads, and Windows WinUSB procs with overlapped I/O.">
  <rect x="230" y="12" width="200" height="30" rx="6" fill="none" stroke="currentColor"/>
  <text x="330" y="31" text-anchor="middle" fill="currentColor" font-size="11">RTL2832U driver</text>
  <line x1="330" y1="42" x2="330" y2="64" stroke="currentColor"/><polygon points="326,56 330,66 334,56" fill="currentColor"/>
  <rect x="168" y="66" width="324" height="38" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="330" y="84" text-anchor="middle" fill="var(--accent)" font-size="11">Transport port · eight methods</text>
  <text x="330" y="98" text-anchor="middle" fill="var(--fg-muted)" font-size="8">ControlIn · ControlOut · ClaimInterface · StartBulkIn / StopBulkIn</text>
  <line x1="250" y1="104" x2="132" y2="146" stroke="currentColor"/><polygon points="136,137 126,148 140,148" fill="currentColor"/>
  <line x1="330" y1="104" x2="330" y2="146" stroke="currentColor"/><polygon points="326,138 330,148 334,138" fill="currentColor"/>
  <line x1="410" y1="104" x2="528" y2="146" stroke="currentColor"/><polygon points="520,148 534,148 524,137" fill="currentColor"/>
  <rect x="20" y="150" width="200" height="48" rx="6" fill="none" stroke="currentColor"/>
  <text x="120" y="170" text-anchor="middle" fill="currentColor" font-size="10">Linux — USBDEVFS</text>
  <text x="120" y="184" text-anchor="middle" fill="var(--fg-muted)" font-size="8">ioctls · URB ring · 1 reaper</text>
  <rect x="230" y="150" width="200" height="48" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="330" y="170" text-anchor="middle" fill="var(--accent)" font-size="10">macOS — IOKit (purego)</text>
  <text x="330" y="184" text-anchor="middle" fill="var(--fg-muted)" font-size="8">vtables · sync ReadPipe · N pinned</text>
  <rect x="440" y="150" width="200" height="48" rx="6" fill="none" stroke="currentColor"/>
  <text x="540" y="170" text-anchor="middle" fill="currentColor" font-size="10">Windows — WinUSB</text>
  <text x="540" y="184" text-anchor="middle" fill="var(--fg-muted)" font-size="8">procs · overlapped · 1 + N events</text>
</svg>
<figcaption>All three USB backends satisfy the identical eight-method <code>Transport</code> port; the RTL2832U driver above can't tell which OS adapter — or which threading model — is underneath it.</figcaption>
</figure>

## How GopherTrunk implements it in Go

### macOS: IOKit through purego

There is no C compiler in the loop. The IOKit and CoreFoundation symbols are
resolved at runtime through **purego**, and the load is deliberately **lazy**: a
`sync.Once` runs `loadIOKit()` the first time anyone asks for the enumerator, so a
framework-resolution glitch surfaces as an error from `List`/`Open` instead of
crashing the test binary at startup.

```go
// internal/sdr/rtlsdr/usb/usb_darwin.go
var (
    darwinLoadOnce sync.Once
    darwinLoadErr  error
)

func platformEnumerator() Enumerator {
    darwinLoadOnce.Do(func() {
        darwinLoadErr = loadIOKit()
    })
    if darwinLoadErr != nil {
        return loadFailedEnumerator{err: darwinLoadErr}
    }
    return &darwinEnumerator{}
}
```

Enumeration queries IOKit's USB-device service registry and reads VID/PID/serial as
IORegistry properties — no device is opened during `List`. `Open` runs the standard
IOCFPlugIn dance to get an `IOUSBDeviceInterface`, opens the device, walks its
interface iterator, and claims interface 0 (the only one the RTL2832U exposes).
Control transfers go through `IOUSBDeviceInterface::DeviceRequest` with a struct
that mirrors the USB 2.0 setup packet:

```go
// internal/sdr/rtlsdr/usb/usb_darwin.go
req := iousbDevRequest{
    BmRequestType: VendorIn,
    BRequest:      bRequest,
    WValue:        wValue,
    WIndex:        wIndex,
    WLength:       uint16(n),
}
if n > 0 {
    req.PData = unsafe.Pointer(&buf[0])
}
rc := vtableCall(t.devIface, deviceDeviceRequest, uintptr(unsafe.Pointer(&req)))
```

**The streaming model is the most distinctive part. Where Linux uses one reaper for
the whole URB ring, macOS spawns one goroutine per ring slot, each pinned to its
own OS thread, doing a *synchronous* `ReadPipe` in a loop.** Cancellation is
`AbortPipe`: every blocked `ReadPipe` returns `kIOReturnAborted`, the goroutines
see the stop flag, and exit. This sidesteps CFRunLoop callbacks entirely — no
C-to-Go callback marshalling, no run-loop thread to babysit — at the cost of
`ringBufs` OS threads (32 by default).

```go
// internal/sdr/rtlsdr/usb/usb_darwin.go
func (t *darwinTransport) bulkLoop(pipeRef uint8, slot *darwinBulkSlot, onPacket func([]byte)) {
    runtime.LockOSThread()
    defer runtime.UnlockOSThread()
    // ...
    for {
        if t.bulkStopFlag.Load() != 0 {
            return
        }
        size := uint32(len(slot.buf))
        rc := vtableCall(t.ifaceIface, ifaceReadPipe,
            uintptr(pipeRef),
            uintptr(unsafe.Pointer(&slot.buf[0])),
            uintptr(unsafe.Pointer(&size)),
        )
        if t.bulkStopFlag.Load() != 0 {
            return
        }
        if rc != kIOReturnSuccess {
            t.recordBulkErr(fmt.Errorf("usb: ReadPipe: 0x%08x", uint32(rc)))
            return
        }
        if size > 0 {
            onPacket(slot.buf[:size])
        }
    }
}
```

### Windows: WinUSB and overlapped I/O

The Windows backend lazily loads its entry points so the package still imports
cleanly on Wine or older installs missing the DLLs — the failure becomes a runtime
error from the first proc call, not a load-time panic:

```go
// internal/sdr/rtlsdr/usb/usb_windows.go
var (
    modWinUSB = windows.NewLazySystemDLL("winusb.dll")

    procWinUsbControlTransfer     = modWinUSB.NewProc("WinUsb_ControlTransfer")
    procWinUsbReadPipe            = modWinUSB.NewProc("WinUsb_ReadPipe")
    procWinUsbAbortPipe           = modWinUSB.NewProc("WinUsb_AbortPipe")
    procWinUsbGetOverlappedResult = modWinUSB.NewProc("WinUsb_GetOverlappedResult")
    // ...
)
```

`Open` opens the device-interface path with `FILE_FLAG_OVERLAPPED` so every pipe
operation is asynchronous, then calls `WinUsb_Initialize` — which *also* claims
interface 0, so `ClaimInterface` on Windows is a no-op. Streaming arms a ring of
reads, each with its own auto-reset event in an `OVERLAPPED`, and the reaper waits
on all of them at once:

```go
// internal/sdr/rtlsdr/usb/usb_windows.go
ret, err := windows.WaitForMultipleObjects(wait, false, windows.INFINITE)
// ...
slot := t.bulkSlots[slotIdx]
var transferred uint32
result, _, _ := procWinUsbGetOverlappedResult.Call(
    t.ifaceHandle,
    uintptr(unsafe.Pointer(&slot.overlapped)),
    uintptr(unsafe.Pointer(&transferred)),
    0, // bWait = FALSE
)
// ...
if result != 0 && transferred > 0 {
    onPacket(slot.buf[:transferred])
}
if err := t.issueReadPipe(t.bulkEpAddr, slot); err != nil {
    // slot is dead; mark consumed
    consumed[slotIdx] = true
}
```

`StopBulkIn` calls `WinUsb_AbortPipe`, which completes every pending read with
`ERROR_OPERATION_ABORTED`; each event signals once, the reaper drains them and
exits on `<-done`. (`ringBufs` is capped at 64 because `WaitForMultipleObjects`
can't wait on more than `MAXIMUM_WAIT_OBJECTS`.)

### Three OSes, one contract

It's worth lining them up against the single `Transport` from Part 4:

| | Linux | macOS | Windows |
|---|---|---|---|
| Access path | USBDEVFS ioctls | IOKit vtables (purego) | WinUSB procs |
| Enumerate | walk sysfs | IOKit registry | SetupAPI |
| Claim iface | ioctl + auto-detach DVB driver | IOCFPlugIn dance | no-op (init claimed) |
| Bulk-IN | async URB ring | sync `ReadPipe` per slot | overlapped `ReadPipe` ring |
| Reaper | **1** goroutine | **N** pinned goroutines | 1 goroutine + N events |
| Cancel | `DISCARDURB` | `AbortPipe` | `AbortPipe` |

Three radically different I/O models, and the driver above sees `(shape)` the same
eight methods regardless. That table *is* the payoff of drawing the port by what
the device needs rather than what any one OS offers.

## The problem we hit: IOKit demands you own the thread

**The symptom.** The very first macOS streaming build didn't return errors — it
*crashed*. Under load the process would abort with low-level IOKit / Mach
complaints, sometimes a `kIOReturnAborted` storm, sometimes a hard abort deep
inside the IOUSB user client. It was intermittent, worse the more buffers we ran,
and it never reproduced on Linux or Windows with the identical driver on top.

**The root cause.** IOKit's user-client interfaces are **not goroutine-portable**.
The `IOUSBInterfaceInterface` ties its I/O state to the OS thread that issues the
calls, and the Go scheduler, by default, freely migrates a goroutine across OS
threads — and parks it on one thread while running other goroutines on it in
between. So a `ReadPipe` could be issued from thread A, then its continuation
resumed on thread B, while thread A was simultaneously driving an *unrelated*
goroutine into the same user client. IOKit saw concurrent, thread-crossing access
to state it assumed was single-threaded and owned, and it did what a C API does
when its invariants are violated: it aborted the process.

**The Go fix.** Pin each reader to its own OS thread for that thread's entire life.
Every per-slot reader calls `runtime.LockOSThread` on entry and `UnlockOSThread`
only on exit, so the goroutine and its OS thread are welded together for as long as
it's doing IOKit I/O:

```go
// internal/sdr/rtlsdr/usb/usb_darwin.go
func (t *darwinTransport) bulkLoop(pipeRef uint8, slot *darwinBulkSlot, onPacket func([]byte)) {
    runtime.LockOSThread()
    defer runtime.UnlockOSThread()
    // ...synchronous ReadPipe loop, AbortPipe cancellation...
}
```

With the lock in place, a slot's `ReadPipe` calls always issue from *one* thread
that does *nothing else*, which is exactly the ownership model IOKit expects. The
aborts vanished. This is also why the macOS design uses **one OS thread per slot**
in the first place: pinning makes the synchronous-`ReadPipe`-per-thread model the
*natural* one, and lets us skip CFRunLoop callbacks entirely. The threading rule
isn't an implementation detail we tolerated — it dictated the whole bulk-IN shape.

<figure class="lab-figure">
<svg viewBox="0 0 660 190" width="660" height="190" role="img" aria-label="Before and after the macOS threading fix. Before: one reader goroutine issues ReadPipe on OS thread A but the Go scheduler resumes it on thread B, so two threads reach the same IOKit user client concurrently and the process aborts. After: runtime.LockOSThread welds the bulkLoop goroutine to a single OS thread that owns the slot, so IOKit sees a single owner and the aborts vanish.">
  <text x="166" y="16" text-anchor="middle" fill="var(--fg-muted)" font-size="10">before — one goroutine, two threads</text>
  <rect x="22" y="34" width="128" height="34" rx="6" fill="none" stroke="currentColor"/>
  <text x="86" y="50" text-anchor="middle" fill="currentColor" font-size="9">thread A</text>
  <text x="86" y="62" text-anchor="middle" fill="var(--fg-muted)" font-size="8">issues ReadPipe</text>
  <rect x="182" y="34" width="128" height="34" rx="6" fill="none" stroke="currentColor"/>
  <text x="246" y="50" text-anchor="middle" fill="currentColor" font-size="9">thread B</text>
  <text x="246" y="62" text-anchor="middle" fill="var(--fg-muted)" font-size="8">resumes goroutine</text>
  <line x1="150" y1="51" x2="178" y2="51" stroke="currentColor" stroke-dasharray="4 3"/><polygon points="178,47 188,51 178,55" fill="currentColor"/>
  <text x="164" y="30" text-anchor="middle" fill="var(--fg-muted)" font-size="7">migrate</text>
  <line x1="86" y1="68" x2="140" y2="118" stroke="currentColor"/><polygon points="132,110 143,120 129,118" fill="currentColor"/>
  <line x1="246" y1="68" x2="192" y2="118" stroke="currentColor"/><polygon points="203,118 189,120 200,110" fill="currentColor"/>
  <rect x="80" y="120" width="172" height="42" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="166" y="138" text-anchor="middle" fill="var(--accent)" font-size="9">IOKit user client</text>
  <text x="166" y="152" text-anchor="middle" fill="var(--fg-muted)" font-size="8">cross-thread access → process abort</text>
  <line x1="330" y1="14" x2="330" y2="172" stroke="var(--fg-muted)" stroke-dasharray="3 4"/>
  <text x="496" y="16" text-anchor="middle" fill="var(--fg-muted)" font-size="10">after — runtime.LockOSThread</text>
  <rect x="416" y="34" width="160" height="34" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="496" y="50" text-anchor="middle" fill="var(--accent)" font-size="9">bulkLoop goroutine</text>
  <text x="496" y="62" text-anchor="middle" fill="var(--fg-muted)" font-size="8">LockOSThread on entry</text>
  <line x1="496" y1="68" x2="496" y2="90" stroke="currentColor"/><polygon points="492,82 496,92 500,82" fill="currentColor"/>
  <rect x="416" y="92" width="160" height="28" rx="6" fill="none" stroke="currentColor"/>
  <text x="496" y="110" text-anchor="middle" fill="currentColor" font-size="9">one OS thread · owns slot</text>
  <line x1="496" y1="120" x2="496" y2="142" stroke="currentColor"/><polygon points="492,134 496,144 500,134" fill="currentColor"/>
  <rect x="408" y="144" width="176" height="32" rx="6" fill="none" stroke="currentColor"/>
  <text x="496" y="164" text-anchor="middle" fill="currentColor" font-size="9">IOKit user client · single owner</text>
</svg>
<figcaption>Pinning each per-slot reader to its own OS thread gives IOKit the single-owner model it demands — which is exactly why the macOS backend runs one pinned thread per ring slot rather than one shared reaper.</figcaption>
</figure>

(The same `runtime.LockOSThread` shows up in the Linux and Windows reapers too, but
for a milder reason: keeping a long-blocking syscall loop off the threads serving
the rest of the program. On macOS it's load-bearing for *correctness*.)

## The design principle: adapters that hide platform threading rules

This is the **adapter pattern** doing exactly what it's for: isolating
platform-specific rules behind a shared contract. **The most important thing each
adapter hides isn't the API surface — it's the threading and I/O model.** Linux
hides "async URBs reaped in one goroutine." Windows hides "overlapped I/O drained
by `WaitForMultipleObjects`." macOS hides "IOKit owns the calling thread, so we
pin one thread per slot." None of that crosses the port.

### How that principle shaped the Go code

- **Each adapter owns its concurrency model.** The driver requests a stream with
  geometry and two callbacks; whether that becomes 1 goroutine, N pinned
  goroutines, or 1 goroutine plus N events is entirely the adapter's business and
  never leaks upward.
- **Platform threading rules stay platform-local.** `runtime.LockOSThread` for
  IOKit ownership lives inside `bulkLoop` in `usb_darwin.go`. The RTL2832U driver
  has no idea macOS has a thread-affinity rule, and shouldn't.
- **Loading is lazy and failure is an error, not a panic.** macOS defers
  `loadIOKit` behind `sync.Once`; Windows defers DLL resolution to first use. A
  missing framework or DLL surfaces as a returned error from the same `List`/`Open`
  methods on every OS, keeping the port total.
- **Errors converge on the shared sentinels.** macOS `translateIOReturn` and
  Windows `winErr` both fold platform codes into `ErrDeviceGone`, `ErrTimeout`,
  `ErrPipeStalled`. Callers `errors.Is` against portable values and never branch on
  GOOS.

## Where this goes next

All three USB adapters are now standing, each satisfying the identical `Transport`
contract. The plumbing is done — we can do vendor control transfers and stream
bulk IQ on Linux, macOS, and Windows without a line of CGO. **Part 7** climbs one
layer up and starts using it for real: bringing up the **RTL2832U** demodulator
itself — the register dance, the EEPROM read, the bring-up retry envelope — all
written against the port, so it runs unchanged on every backend we just built.

## FAQ

**Why one OS thread per slot on macOS instead of one reaper like Linux?**
Because IOKit ties USB I/O state to the issuing OS thread. Pinning one thread per
slot makes synchronous `ReadPipe` the natural model and avoids both CFRunLoop
callback marshalling and the cross-thread access that crashed early builds. The
cost is ~32 OS threads, acceptable for a foreground SDR daemon.

**Why is `ClaimInterface` a no-op on Windows?**
`WinUsb_Initialize` already grants exclusive access to interface 0 when the device
is opened, so there's nothing left to claim. The method still rejects `num != 0`
so a caller asking for a second interface gets an explicit error rather than a
silent success.

**How is any of this tested without a Mac or a Windows box in CI?**
The same way the rest of the driver is: `MockTransport` from Part 4 satisfies the
port, so the RTL2832U and tuner logic above run identically on a headless Linux
runner. The platform-specific files compile under cross-compilation
(`GOOS=darwin`/`windows CGO_ENABLED=0`), and the macOS backend's hardware
validation is tracked as a follow-up against real dongles.

## Series navigation

**Part 6 of 14** · ←
[Part 5]({{ '/blog/deep-dives/rf-front-end-05-usb-on-linux-usbdevfs/' | relative_url }})
· Next →
[Part 7: RTL-SDR I — bringing up the RTL2832U]({{ '/blog/deep-dives/rf-front-end-07-rtlsdr-rtl2832u-bringup/' | relative_url }})
