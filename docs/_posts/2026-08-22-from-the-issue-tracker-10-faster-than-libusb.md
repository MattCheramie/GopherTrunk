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

## Cheat sheet

| | |
|---|---|
| Issue | [#395](https://github.com/MattCheramie/GopherTrunk/issues/395) — Windows 10, `ERROR_GEN_FAILURE` at baseband init step 0 |
| Symptom | the warmup write succeeds; the byte-identical step-0 write fails — every attempt, cold boots, reboots, every port |
| Wrong theory | wedged clone firmware needing a wider reset-and-retry envelope |
| Why it failed | the fault sits *between two transfers inside one attempt*; between-attempt hygiene can't reach it by construction |
| Real cause | direct `WinUsb_ControlTransfer` is hundreds of microseconds faster than libusb's stack; the second write outruns the clone's firmware |
| Fix | 10 ms settle (`warmupSettleDuration`) between warmup and baseband init + eagerly seed `PIPE_TRANSFER_TIMEOUT=0` the way libusb does |
| Rule that survives | match the reference stack's *pace* and endpoint policy, not just its bytes |

## In this post

- **The symptom: the same bytes, accepted then refused** — a healthy dongle rejects a write it just accepted.
- **The theory that couldn't work** — the widened retry envelope, and the three-sentence reply that killed it.
- **The real cause: we removed the stack that was the delay** — counting the layers, and what each one costs.
- **The fix** — one settle at one boundary, plus libusb's endpoint policy.
- **What we keep** — failure position, undocumented timing, and the pacing tell.

## The symptom: the same bytes, accepted then refused

The Windows report looked at first like a cousin of Part 9's EPIPE: device
enumerates, interface claims, then bring-up dies — this time with Windows'
maximally unhelpful `ERROR_GEN_FAILURE` (0x1F), the WinUSB way of saying "the
device did not like that."

```text
rtlsdr: init baseband: init baseband step 0 ... ERROR_GEN_FAILURE
```

Everything the user could check had checked out. `sdr list` showed the dongle.
`sdr doctor` reported the WinUSB binding OK. The dongle worked in other SDR
software on the same machine. The report (against v0.2.4) even called the
jurisdiction correctly: this fails at the first USB write to the chip, so
"there's nothing on the user side left to try." They were right.

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
Part 9 had made chip-state theories respectable, and GopherTrunk already carried a
recovery path for exactly that on Windows. So the first round widened it,
concretely and conservatively:

- Open-time retries: **3 attempts → 5**, with exponential backoff
  (200 / 400 / 800 / 1200 ms in place of the old 100 / 200 ms).
- The WinUSB `Reset()` settle: **50 ms → 150 ms**, on the reasoning that libusb's
  50 ms is calibrated for clean closes, not wedged-firmware recovery.
- A sharper terminal error, telling the operator to unplug the dongle for ten
  seconds — the one action that physically clears firmware state.

Healthy hardware still opened on attempt zero with no delay; only a dongle that
actually needed recovery paid the new cost, about 2.6 seconds worst case. A
defensible change, built on a real failure class from other reports — and
completely beside the point.

The reporter's follow-up was three sentences and worth more than the whole round:
cold starts, reboots, different ports, no hub — and the dongle worked fine in
SDRTrunk and OP25 on the same machine. That killed the wedged-firmware theory on
its own terms, because a genuinely wedged dongle fails the *warmup* write too.
This one wasn't arriving wedged: it *accepted the first write of every attempt*.
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
latency between consecutive transfers. That overhead isn't one thing; it's a tax
collected at every layer. libusb allocates and tracks a transfer object per
request, serializes access to the device handle behind its own locks, and crosses
from `libusb-1.0.dll`'s abstractions into `winusb.dll`'s before the kernel
transition into `WinUsb.sys` ever happens. GopherTrunk's pure-Go driver invokes
the WinUSB entry point directly — one `procWinUsbControlTransfer.Call` — and
issues back-to-back writes with almost no gap.

On well-behaved silicon that's free performance. On this clone, the firmware needs
a moment after acknowledging one vendor request before it can accept the next; land
the second write inside that window and it NAKs, which WinUSB surfaces as
`ERROR_GEN_FAILURE`. Every libusb-based program on the same machine kept working —
not because it did anything differently, but because its overhead *was* the settle
time.

We also checked the boring alternative the reporter had asked about directly: is
this a Windows 10 problem? No. The WinUSB surface is the same on Windows 10 and
11 — same `WinUsb_ControlTransfer` semantics, same control-endpoint behavior, and
the same pipe-policy defaults:

| WinUSB pipe policy | Default (Win10 and Win11) |
|---|---|
| `PIPE_TRANSFER_TIMEOUT` | 5000 ms |
| `AUTO_CLEAR_STALL` | false |
| `RAW_IO` | false |

The dongle would almost certainly reproduce this on Windows 11 too; the OS version
in the report's title was incidental. The variable was the stack we had
deliberately made thinner.

This is Part 9's lesson with the sign flipped. There, byte-identical writes weren't
enough because the Linux driver skipped a settling gap librtlsdr paid in function
calls. Here, the same discovery on Windows: matching the reference implementation's
*bytes* is necessary; matching its *pace* is the part nobody writes down.

## The fix

Two changes, both small:

- **A 10 ms settle between the USB warmup write and baseband init** —
  `warmupSettleDuration` in `internal/sdr/rtlsdr/purego/driver.go`. Placed once at
  the boundary the evidence identified, not sprinkled defensively. It bounds the
  cost at open time and gives the slowest firmware observed comfortable margin.
- **Eagerly seed `PIPE_TRANSFER_TIMEOUT = 0` on the control endpoint** — in
  `internal/sdr/rtlsdr/usb/usb_windows.go`, matching what libusb's
  `winusbx_configure_endpoints` does on claim; GopherTrunk had left the defaults
  in place. This is parity hardening rather than the root cause, but it removes
  one more silent difference between our WinUSB usage and the one every dongle
  firmware has been de facto validated against.

The widened retry envelope from round one stayed in the tree — it was aimed at a
real failure class from other reports, just not this one — and the settle came
with a falsifiable follow-up plan. The debug trace is the ground truth for
whether 10 ms is the right number:

```powershell
$env:RTLSDR_DEBUG_USB="1"
gophertrunk sdr list --probe
```

If step 0 still failed with the settle in place, the value needed widening. If
step 0 succeeded and the failure *moved* to step 1 or later, that would mean the
gap wasn't warmup-specific and `InitBaseband` needed per-step settles instead.
Neither escalation was needed: the reporter's dongle opened cleanly with both
changes in place.

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

## FAQ

**Why did the dongle work in SDRTrunk and OP25 on the same machine?**
Both reach the chip through libusb, whose per-transfer overhead — transfer
objects, locking, an extra DLL boundary — adds hundreds of microseconds between
consecutive writes. That incidental latency *is* the settle time the clone's
firmware needs. Clone firmware is de facto validated against libusb's pace, never
against a specification.

**Was this a Windows 10 problem?**
No. The WinUSB API surface, control-endpoint semantics, and pipe-policy defaults
(`PIPE_TRANSFER_TIMEOUT=5000`, `AUTO_CLEAR_STALL=false`, `RAW_IO=false`) are the
same on Windows 10 and 11. The same dongle would almost certainly reproduce the
race on either; a fast machine and a slow clone were the actual ingredients.

**Why didn't the wider retry envelope help at all?**
Because the failure lived *inside* one attempt — between the warmup write and
step 0 — and every retry re-ran that same gap at the same pace. Between-attempt
remedies (more attempts, longer backoff, device rebinds) can only fix state that
persists *across* attempts, and the dongle demonstrably arrived healthy each
time, accepting the first write.

**How was 10 ms chosen, and what if it isn't enough on some other dongle?**
It was chosen with comfortable margin over the observed pattern, at a cost paid
once per open. The `RTLSDR_DEBUG_USB=1` trace is the arbiter: a still-failing
step 0 means widen the settle; a failure that moves to a later init step means
the gap isn't warmup-specific and `InitBaseband` needs per-step settles. The
ladder lives in [RTL-SDR USB recovery]({{ '/reference/rtlsdr-usb-recovery/' | relative_url }}).

**If direct WinUSB causes this, why not just use libusb?**
The pure-Go, CGO-free driver is what makes GopherTrunk a single static binary
with no driver-library dependencies — that's worth keeping. The price is that
every incidental behavior libusb ships — its pacing, its endpoint policy, its
stall clearing ([Part 11]({{ '/blog/solution-postmortem/from-the-issue-tracker-11-detected-but-not-present/' | relative_url }}))
— must be identified and reimplemented deliberately.

## Series navigation

**Part 10 of 22** · ←
[Part 9: Broken Pipe — Six Rounds of Traces for One USB Write]({{ '/blog/solution-postmortem/from-the-issue-tracker-09-broken-pipe/' | relative_url }})
· Next →
[Part 11: Detected but Not Present — One Hex Code from a Fix That Already Existed]({{ '/blog/solution-postmortem/from-the-issue-tracker-11-detected-but-not-present/' | relative_url }})
