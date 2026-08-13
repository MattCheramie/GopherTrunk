---
title: "From the Issue Tracker, Part 10: Faster Than libusb — When the Second Write Outruns the First"
description: A Windows 10 dongle accepted a control write and then rejected its byte-identical twin with ERROR_GEN_FAILURE. The retry-envelope fix couldn't work by construction — the real bug was that a pure-Go WinUSB path is thinner than libusb's stack, and libusb's overhead had been hiding a firmware race.
category: solution-postmortem
keywords: rtl-sdr, windows, error_gen_failure, winusb, libusb, control transfer, clone firmware, usb timing, baseband init, settle delay
tags: [from-the-issue-tracker, rtl-sdr, usb, windows, drivers, debugging, postmortem]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "From the Issue Tracker"
series_part: 10
---

*Part 10 of **From the Issue Tracker**, postmortems of GopherTrunk bugs that fought
back. [Part 9]({{ '/blog/solution-postmortem/from-the-issue-tracker-09-broken-pipe/' | relative_url }})
spent six rounds of paired USB traces on a Linux EPIPE and ended at a 5 ms sleep.
This part is the same species of bug on Windows — shorter, because Part 9 taught us
what to look for, and sharper, because this time the failing write was byte-identical
to the one that had just succeeded.*

> **TL;DR:** On Windows 10, an RTL-SDR clone failed baseband init with
> `ERROR_GEN_FAILURE` ([#395](https://github.com/MattCheramie/GopherTrunk/issues/395)).
> The first fix widened the reset-and-retry envelope on a "wedged clone firmware"
> theory — and could not have worked, because the dongle accepted the warmup write
> fine and only failed the *byte-identical second one*. A bug between two transfers
> inside a single attempt is untouchable by any between-attempts retry. The real
> cause: librtlsdr reaches the chip through `libusb-1.0.dll` → `winusb.dll` →
> `WinUsb.sys`, paying hundreds of microseconds per transfer; GopherTrunk calls
> `WinUsb_ControlTransfer` directly and lands the second write while the clone's
> firmware is still digesting the first. libusb's overhead was incidentally hiding
> the race. Fix: a 10 ms settle between warmup and baseband init, plus eagerly
> seeding the control endpoint's timeout policy the way libusb does.

## The symptom: the same bytes, accepted then refused

The Windows report looked at first like a cousin of Part 9's EPIPE: device
enumerates, interface claims, then bring-up dies — this time with Windows'
maximally unhelpful `ERROR_GEN_FAILURE` (0x1F), the WinUSB way of saying "the
device did not like that."

The detail that mattered was *where* it died: baseband init, step 0. And step 0 is
not an exotic transfer. GopherTrunk's open path had just executed the warmup probe
from Part 9's first round — and both writes are the same bytes:

| Transfer | Block | Address | Value | Length | Outcome |
|---|---|---|---|---|---|
| USB warmup | `USB` | `USB_SYSCTL` (0x2000) | `0x09` | 1 | succeeds |
| Baseband init, step 0 | `USB` | `USB_SYSCTL` (0x2000) | `0x09` | 1 | `ERROR_GEN_FAILURE` |

A vendor-OUT the chip just accepted, re-sent moments later, refused.

## The theory that couldn't work

The first fix read the symptom as "clone firmware wedged by a previous session" —
Part 9 had made chip-state theories respectable — and widened the open path's
reset-and-retry envelope: more attempts, longer backoff, a device rebind between
tries.

It changed nothing, and a second look at the failure showed it never could have.
The dongle wasn't arriving wedged: it *accepted the first write of every attempt*.
Each retry ran warmup (success) then step 0 (failure), identically, every time. A
failure that reproduces at the same point *within* each attempt lives between two
transfers inside the attempt — no amount of between-attempt hygiene can reach it.
The retry envelope was treating the sequence as corrupted state when the evidence
said healthy state, hit too fast.

That reframing — "the bug is in the gap between two writes, not in the state before
the first" — is the whole postmortem. Everything after it was mechanical.

## The real cause: we removed the stack that was the delay

Count the layers each program crosses to move one control transfer on Windows:

```text
librtlsdr:    rtlsdr_write_reg → libusb-1.0.dll → winusb.dll → WinUsb.sys → bus
GopherTrunk:  procWinUsbControlTransfer.Call    → winusb.dll → WinUsb.sys → bus
```

librtlsdr pays libusb's transfer bookkeeping, its internal locking, and a
user-mode round trip on every single write — hundreds of microseconds of incidental
latency between consecutive transfers. GopherTrunk's pure-Go driver invokes the
WinUSB entry point directly and issues back-to-back writes with almost no gap.

On well-behaved silicon that's free performance. On this clone, the firmware needs
a moment after acknowledging one vendor request before it can accept the next; land
the second write inside that window and it NAKs, which WinUSB surfaces as
`ERROR_GEN_FAILURE`. Every libusb-based program on the same machine kept working —
not because it did anything differently, but because its overhead *was* the settle
time. We also checked the boring alternative: there is no relevant WinUSB API
difference between Windows 10 and 11 to blame. The variable was the stack we had
deliberately made thinner.

This is Part 9's lesson with the sign flipped. There, byte-identical writes weren't
enough because the Linux driver skipped a settling gap librtlsdr paid in function
calls. Here, the same discovery on Windows: matching the reference implementation's
*bytes* is necessary; matching its *pace* is the part nobody writes down.

## The fix

Two changes, both small:

- **A 10 ms settle between the USB warmup write and baseband init.** Placed once at
  the boundary the trace identified, not sprinkled defensively. It bounds the cost
  at open time and gives the slowest firmware observed comfortable margin.
- **Eagerly seed `PIPE_TRANSFER_TIMEOUT = 0` on the control endpoint.** libusb's
  `winusbx_configure_endpoints` sets pipe policy on claim; GopherTrunk had left the
  defaults in place. Matching it removes one more silent difference between our
  WinUSB usage and the one every dongle firmware has been de facto validated
  against.

The reporter's dongle opened cleanly with both in place.

## What we keep

- **Read the failure's position before choosing the fix's position.** A
  fails-on-second-write-every-attempt symptom excludes every between-attempts
  remedy by construction. Five minutes with that logic would have saved the retry
  round; it's now an early step in the
  [diagnostic playbook]({{ '/reference/diagnostic-playbook/' | relative_url }}).
- **Vendor libraries ship undocumented timing, and hardware grows to depend on
  it.** Clone firmware is validated against libusb's pace, never against a
  specification. A thinner stack inherits the bytes but not the microseconds —
  budget explicit settles at bring-up boundaries when you remove layers.
- **`ERROR_GEN_FAILURE` on a transfer that just succeeded means pacing, not
  corruption.** The same-bytes-different-outcome pattern is the tell that separates
  a timing race from a state problem, and it's now recorded with the rest of the
  ladder in [RTL-SDR USB recovery]({{ '/reference/rtlsdr-usb-recovery/' | relative_url }}).
- **Match the reference's endpoint policy, not just its transfers.** Pipe timeout
  policy, repeater toggles, settle windows — the reference implementation's side
  effects are part of its protocol whether it knows it or not.

*Next: [Part 11]({{ '/blog/solution-postmortem/from-the-issue-tracker-11-detected-but-not-present/' | relative_url }})
moves to macOS, where the recovery for exactly this family of stalls already
existed — and one untranslated error code kept it from ever running.*
