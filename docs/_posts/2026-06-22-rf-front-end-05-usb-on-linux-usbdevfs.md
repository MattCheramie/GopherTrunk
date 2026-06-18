---
title: "RF Front End, Part 5: USB on Linux — USBDEVFS & Async URBs"
description: How GopherTrunk's Linux USB backend hand-rolls USBDEVFS ioctl encoding, submits a ring of async URBs, and reaps them in a single goroutine for sustained IQ throughput — all in pure Go behind a safe Transport API.
category: deep-dives
tags: [sdr, go, usb, usbdevfs, software-design]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "RF Front End"
series_part: 5
---

*Part 5 of **RF Front End**. We implement the Linux adapter behind the
`Transport` port from Part 4: USBDEVFS ioctls encoded by hand, a ring of async
URBs in flight at once, and a single reaper goroutine draining them — plus the
bit-exact struct layout that decides whether the kernel accepts a URB or silently
drops it.*

> **TL;DR** — The Linux USB backend hand-rolls USBDEVFS ioctls and runs a ring of async URBs reaped by a single goroutine for sustained IQ throughput, all in pure Go. The headline bug: a struct field-alignment mismatch made the size-encoded ioctl number wrong, so streaming failed silently until the Go struct was made provably byte-identical to the kernel's.

## In this post

- What **USBDEVFS** is and why GopherTrunk drives `/dev/bus/usb` directly instead
  of linking `libusb`.
- Hand-rolling the **`_IOWR`-style ioctl numbers** from Go, and the structs that
  must match the kernel's ABI byte-for-byte.
- Submitting a **ring of async URBs** and reaping them in **one goroutine** — why
  multiple in-flight transfers are required for sustained throughput.
- The problem of getting the **ioctl encoding bit-exact** when a wrong number
  fails silently.

## What the Linux USB backend does

On Linux, every USB device shows up as a character device under
`/dev/bus/usb/BBB/DDD`, and the kernel's **usbfs** (a.k.a. USBDEVFS) interface
lets a userspace process do real USB I/O on it through `ioctl(2)`. That's how
`libusb` works under the hood on Linux, and it's how GopherTrunk works without
`libusb`: open the device node, issue the right ioctls, and you have control
transfers and bulk streaming with nothing between you and the kernel.

The backend lives in `usb_linux.go` and provides `platformEnumerator()` — the
per-OS hook `DefaultEnumerator` calls. Enumeration walks **sysfs**
(`/sys/bus/usb/devices`) to read VID/PID/serial without opening anything, so
listing dongles needs no privilege beyond sysfs read permission:

```go
// internal/sdr/rtlsdr/usb/usb_linux.go
func platformEnumerator() Enumerator { return &linuxEnumerator{} }

func (l *linuxEnumerator) Name() string { return "usbdevfs" }
```

`Open` then opens the matching node under `/dev/bus/usb` with `O_RDWR | O_CLOEXEC`
and wraps the file descriptor in a `linuxTransport`. Everything after that is
ioctls on that one fd.

## How GopherTrunk implements it in Go

**The first thing the backend needs is the ioctl request numbers, and Go gives us no
header to crib them from — `libusb` and the kernel get them from C macros.** So we
reproduce the **asm-generic ioctl encoding** by hand. Every ioctl number on Linux
packs a direction, a "type" letter, a sequence number, and the argument size into
a single integer:

```go
// internal/sdr/rtlsdr/usb/usb_linux.go
const (
    iocNRShift   = 0
    iocTypeShift = 8
    iocSizeShift = 16
    iocDirShift  = 30

    iocNone  = 0
    iocWrite = 1
    iocRead  = 2
)

func ioc(dir, typ, nr, size uintptr) uintptr {
    return (dir << iocDirShift) | (typ << iocTypeShift) | (nr << iocNRShift) | (size << iocSizeShift)
}
```

That `ioc` is the Go equivalent of the kernel's `_IOWR(type, nr, size)` family of
macros. With it we can name every USBDEVFS command the driver uses — and the
*size* field is computed from the actual Go struct via `unsafe.Sizeof`, so the
number and the payload can never drift apart:

```go
// internal/sdr/rtlsdr/usb/usb_linux.go
var (
    usbdevfsControl          = ioc(iocRead|iocWrite, 'U', 0, unsafe.Sizeof(usbdevfsCtrltransfer{}))
    usbdevfsSubmitURB        = ioc(iocRead, 'U', 10, unsafe.Sizeof(usbdevfsURB{}))
    usbdevfsDiscardURB       = ioc(iocNone, 'U', 11, 0)
    usbdevfsReapURB          = ioc(iocWrite, 'U', 12, unsafe.Sizeof(uintptr(0)))
    usbdevfsClaimInterface   = ioc(iocRead, 'U', 15, 4)
    usbdevfsReleaseInterface = ioc(iocRead, 'U', 16, 4)
    usbdevfsIoctlCmd         = ioc(iocRead|iocWrite, 'U', 18, unsafe.Sizeof(usbdevfsIoctlArg{}))
    usbdevfsReset            = ioc(iocNone, 'U', 20, 0)
    usbdevfsDisconnect       = ioc(iocNone, 'U', 22, 0)
)
```

Each of those structs is a Go mirror of a kernel `struct`. A control transfer, for
example, mirrors `struct usbdevfs_ctrltransfer`:

```go
// internal/sdr/rtlsdr/usb/usb_linux.go
type usbdevfsCtrltransfer struct {
    BmRequestType uint8
    BRequest      uint8
    WValue        uint16
    WIndex        uint16
    WLength       uint16
    Timeout       uint32
    Data          *byte
}
```

`ControlIn` and `ControlOut` fill one of these in and fire a single
`unix.Syscall(SYS_IOCTL, fd, usbdevfsControl, &ctrl)`. There's no copy, no
marshalling layer — the Go struct *is* the wire format, which is why the field
layout has to be exact (more on that in the problem section).

### A ring of async URBs

Control transfers are easy because they're synchronous. The IQ stream is not. At
2.4 MS/s of 8-bit IQ, the dongle produces ~2.4 MB/s and will overrun any scheme
that does one blocking read, processes it, then reads again — the gap between
reads is dead air on the bus. The fix is the same one `libusb` and `librtlsdr` use:
keep **many transfers in flight at once** so the host controller always has a
buffer to fill while userspace drains the previous one.

USBDEVFS does this with **URBs** (USB Request Blocks): you `SUBMITURB` a buffer,
the kernel fills it asynchronously, and you `REAPURB` to collect completions.
`StartBulkIn` submits a whole ring up front:

```go
// internal/sdr/rtlsdr/usb/usb_linux.go
for i := 0; i < ringBufs; i++ {
    buf := make([]byte, bufLen)
    urb := &usbdevfsURB{
        Type:         usbdevfsURBTypeBULK,
        Endpoint:     epAddr,
        Buffer:       &buf[0],
        BufferLength: int32(bufLen),
    }
    _, _, errno := unix.Syscall(unix.SYS_IOCTL, uintptr(t.fd), usbdevfsSubmitURB, uintptr(unsafe.Pointer(urb)))
    if errno != 0 {
        // ...discard the URBs already submitted, drain, and bail
        return fmt.Errorf("usbdevfs: SUBMITURB[%d]: %w", i, translateErrno(errno))
    }
    slots = append(slots, &bulkSlot{urb: urb, buf: buf})
}
```

The driver's default geometry is **32 buffers of 16 KiB** — `DefaultRingBuffers`
and `DefaultBufferLen` in `usb.go`. With 32 URBs queued, the kernel can keep the
bus saturated even if the consumer stalls for several milliseconds.

Once the ring is submitted, a **single reaper goroutine** owns the completion
loop. It pins itself to its OS thread, reaps one URB at a time, hands the bytes to
`onPacket`, and immediately resubmits that same URB to keep the ring full:

```go
// internal/sdr/rtlsdr/usb/usb_linux.go
func (t *linuxTransport) reapLoop(onPacket func([]byte), onStreamDead func(error), slots []*bulkSlot, submitted int, done chan struct{}) {
    runtime.LockOSThread()
    defer runtime.UnlockOSThread()
    defer close(done)
    // ...
    for remaining > 0 {
        var ptr uintptr
        _, _, errno := unix.Syscall(unix.SYS_IOCTL, uintptr(t.fd), usbdevfsReapURB, uintptr(unsafe.Pointer(&ptr)))
        // ...
        slot := findSlotByAddr(slots, ptr)
        // ...
        if urb.Status == 0 && urb.ActualLength > 0 {
            onPacket(slot.buf[:urb.ActualLength])
        }
        urb.ActualLength = 0
        urb.Status = 0
        unix.Syscall(unix.SYS_IOCTL, uintptr(t.fd), usbdevfsSubmitURB, uintptr(unsafe.Pointer(urb)))
    }
}
```

One detail worth flagging: `REAPURB` hands back the *address* of a URB we
previously submitted. We never dereference that raw `uintptr` — instead
`findSlotByAddr` looks it up against our slot ring, whose entries Go's GC keeps
alive, and we use *that* pointer. Trusting the kernel's bare address would race the
garbage collector; the lookup keeps us on pointers the runtime knows about.

Cancellation is the mirror image. `StopBulkIn` sets an atomic stop flag, issues
`DISCARDURB` against every slot to unblock the kernel, and waits on the `done`
channel for the reaper to exit:

```go
// internal/sdr/rtlsdr/usb/usb_linux.go
t.bulkStopFlag.Store(1)
for _, s := range slots {
    unix.Syscall(unix.SYS_IOCTL, uintptr(t.fd), usbdevfsDiscardURB, uintptr(unsafe.Pointer(s.urb)))
}
<-done
```

If instead the reaper exits because every URB *died* — a host-controller hang, a
yanked dongle (`ENODEV`) — with the stop flag still unset, it fires `onStreamDead`
from a fresh goroutine so the driver can tear down its consumer rather than block
forever. (That callback is the fix for issue #345, which we'll revisit when we
build the driver in Part 7.)

### Claiming the interface away from the kernel

There's one more Linux-specific wrinkle. When you plug in an RTL-SDR, the kernel
helpfully binds the `dvb_usb_rtl28xxu` DVB-T TV-tuner driver to it. To stream IQ we
have to take the interface back. `ClaimInterface` tries `USBDEVFS_CLAIMINTERFACE`,
and on `EBUSY` it detaches the kernel driver via the "ioctl within an ioctl"
(`USBDEVFS_IOCTL` carrying `USBDEVFS_DISCONNECT`) and retries once — mirroring
libusb's `AUTO_DETACH_KERNEL_DRIVER`. The policy is factored into a pure function
so it's unit-testable without a real device node:

```go
// internal/sdr/rtlsdr/usb/usb_linux.go
func claimWithAutoDetach(claim, detach func() error) error {
    err := claim()
    if !errors.Is(err, unix.EBUSY) {
        return err
    }
    if detachErr := detach(); detachErr != nil {
        return fmt.Errorf("usbdevfs: interface busy, kernel-driver auto-detach failed: %w (original: %w)", detachErr, err)
    }
    return claim()
}
```

## The problem we hit: a wrong ioctl number fails silently

**The symptom.** Control transfers worked perfectly on the very first try — read
the EEPROM, write a register, all of it. But the moment we called `SUBMITURB` to
start streaming, nothing happened. No bytes, no error we could act on, sometimes a
bare `EINVAL` and sometimes a clean return with an empty ring that never produced
a completion. The dongle's LED suggested it was alive; the reaper just sat there.

**The root cause.** The ioctl number was wrong by a few bits. Because USBDEVFS
encodes the *argument size* into the request number itself, the kernel compares the
size you encoded against the size it expects for that command. Our first
`usbdevfsURB` struct had a field-alignment mismatch — Go's natural padding put the
`Buffer` pointer at a different offset than the kernel's `struct usbdevfs_urb`, so
`unsafe.Sizeof(usbdevfsURB{})` produced a size the kernel didn't recognize for
command `'U', 10`. The encoded number was self-consistent and *looked* right, but
it named a command the kernel had no handler matching, so the submit was rejected
or misread the buffer pointer. Control transfers had survived because their struct
happened to be alignment-clean; the URB struct, with its mix of `int32`s and a
trailing pointer, was not.

**The Go fix.** Two things, captured and verified rather than guessed:

- We pinned the struct layout to the kernel's by **constraining the build tag** to
  architectures where Go's natural alignment matches the kernel's asm-generic
  layout, and documented it on the struct itself:

```go
// internal/sdr/rtlsdr/usb/usb_linux.go
// usbdevfsCtrltransfer mirrors struct usbdevfs_ctrltransfer in
// <linux/usbdevice_fs.h>. The build tag above restricts this file to
// architectures where Go's natural field alignment matches the kernel's
// ... so the Go compiler's auto-padding produces a wire-identical struct.
```

  The `//go:build linux && (amd64 || arm64 || 386 || arm || riscv64 || loong64)`
  header isn't cosmetic — it's the guarantee that `unsafe.Sizeof` yields the
  kernel's size on every target we compile for.

- We **verified the encoded numbers against the kernel's own macros** by computing
  what `_IOWR('U', 10, struct usbdevfs_urb)` expands to and diffing it against our
  `ioc()` output for each command, and by capturing a working `libusb` session
  with `strace -e ioctl` to read the exact request numbers the kernel accepts off
  the wire. Once our `usbdevfsSubmitURB` matched the strace value byte-for-byte,
  URBs started completing immediately.

The broader lesson: when the request number *is* a function of the struct size, a
struct bug and an ioctl bug are the same bug, and it surfaces as silence. The
fix had to make the Go struct and the kernel struct provably identical — and then
prove it against a real trace, not just by re-reading the header.

## The design principle: a thin syscall layer over an unsafe ABI

The whole backend is one idea: **encapsulate an unsafe kernel ABI behind a safe Go
API**. Everything dangerous — raw `unsafe.Pointer`, hand-encoded ioctl numbers,
structs that must match a C header bit-for-bit, a `uintptr` from the kernel we
must not dereference — is confined to `usb_linux.go`. What escapes the file is the
ordinary, garbage-collected, error-returning `Transport` from Part 4.

### How that principle shaped the Go code

- **`unsafe` is contained, not spread.** Every `unsafe.Pointer` and every
  `SYS_IOCTL` syscall lives in this one file. The RTL2832U driver above never
  touches `unsafe`; it calls `ControlOut` and `StartBulkIn` and gets Go errors
  back.
- **The size and the struct are derived from one source.** Encoding ioctl numbers
  with `unsafe.Sizeof(theStruct{})` means you cannot change a struct without the
  request number tracking it — the ABI mismatch that bit us can't silently
  reappear.
- **Kernel pointers stay at arm's length.** The reaper looks reaped URB addresses
  up in a GC-visible slot table (`findSlotByAddr`) instead of dereferencing the
  bare `uintptr`, so the unsafe boundary never hands a raw kernel pointer to Go's
  memory model.
- **Errno is translated at the boundary.** `translateErrno` maps `ENODEV` →
  `ErrDeviceGone`, `ETIMEDOUT` → `ErrTimeout`, so callers compare against the
  package's portable sentinels and never learn they're on Linux.

## Where this goes next

The Linux adapter is the reference implementation; **Part 6** builds the other two.
macOS reaches the same `Transport` contract through IOKit and CoreFoundation —
loaded via purego, with one OS-thread-pinned goroutine per ring slot doing
synchronous `ReadPipe`. Windows reaches it through WinUSB and overlapped I/O,
draining completions with `WaitForMultipleObjects`. Both are deliberately
different machinery underneath the *same* eight methods — the clearest possible
demonstration of why we drew the port the way we did in Part 4.

## FAQ

**Why submit 32 URBs instead of just a couple?**
Sustained 2.4 MS/s leaves no slack for round-trips. With a deep ring the host
controller always has a buffer to fill while userspace drains the last one, so a
multi-millisecond consumer stall doesn't drop samples. Two or three URBs overrun
the moment the scheduler hiccups.

**Why pin the reaper goroutine with `runtime.LockOSThread`?**
The reap loop blocks in a syscall for long stretches; pinning it keeps the Go
scheduler from migrating it mid-`ioctl` and keeps the blocking I/O off the threads
serving the rest of the program.

**What happens on an architecture not in the build tag?**
It falls through to `usb_other.go`'s unsupported enumerator from Part 4. We only
claim the arches where Go's struct alignment provably matches the kernel's, rather
than ship a backend whose `unsafe.Sizeof` might lie.

## Series navigation

**Part 5 of 14** · ←
[Part 4]({{ '/blog/deep-dives/rf-front-end-04-usb-without-libusb/' | relative_url }})
· Next →
[Part 6: USB on macOS & Windows]({{ '/blog/deep-dives/rf-front-end-06-usb-on-macos-windows/' | relative_url }})
