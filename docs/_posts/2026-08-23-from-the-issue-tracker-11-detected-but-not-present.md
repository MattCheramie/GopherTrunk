---
title: "From the Issue Tracker, Part 11: Detected but Not Present — One Hex Code from a Fix That Already Existed"
description: macOS enumerated the dongle, then the daemon insisted the configured SDR wasn't on the bus. One probe line surfaced IOKit kern_return 0xe000404f — a pipe stall GopherTrunk already knew how to recover from on Linux and Windows, except the macOS backend never translated the error.
category: solution-postmortem
keywords: rtl-sdr, macos, apple silicon, iokit, kern_return, 0xe000404f, pipe stall, usb recovery, error translation, tuner probe
tags: [from-the-issue-tracker, rtl-sdr, usb, macos, drivers, debugging, postmortem]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "From the Issue Tracker"
series_part: 11
---

*Part 11 of **From the Issue Tracker**, postmortems of GopherTrunk bugs that fought
back. [Part 9]({{ '/blog/solution-postmortem/from-the-issue-tracker-09-broken-pipe/' | relative_url }})
built a USB stall recovery ladder on Linux;
[Part 10]({{ '/blog/solution-postmortem/from-the-issue-tracker-10-faster-than-libusb/' | relative_url }})
extended it to Windows. This part is the third leg of the tripod: macOS had the same
stall, GopherTrunk had the same recovery — and a single untranslated error code stood
between them.*

> **TL;DR:** On an Apple silicon Mac, GopherTrunk enumerated an RTL-SDR and then
> refused to open it, reporting the configured SDR "not present on the bus"
> ([#1038](https://github.com/MattCheramie/GopherTrunk/issues/1038)). One
> `sdr list --probe` line held the real story: the tuner's first I²C burst write
> failed with IOKit `kern_return 0xe000404f` — `kIOUSBPipeStalled`, a transient
> control-pipe stall common on cold boot. The per-chunk retry and chunk-halving
> recovery built for exactly this in Part 9 was already in the tree, keyed on
> `syscall.EPIPE` for Linux and `ERROR_GEN_FAILURE` for Windows. The macOS backend
> let the raw kern_return fall through unrecognized, so the first stalled chunk
> aborted tuner init, detection concluded no tuner existed, and the daemon blamed
> the bus. The fix is one error-translation entry. The lesson is that a recovery
> keyed on platform-specific errors silently doesn't exist on the platform you
> forgot.

## Cheat sheet

| | |
|---|---|
| Issue | [#1038](https://github.com/MattCheramie/GopherTrunk/issues/1038) — macOS/Apple silicon: open fails though the device is detected |
| Symptom | diagnostics list the dongle (`dongles : 1 detected`); the daemon warns `configured SDR not present on the bus` |
| Wrong theories | cables/hubs/`lsusb`; the `blog_v4` config flag; "this dongle brand isn't 100% compatible" |
| Real cause | IOKit `kern_return 0xe000404f` (`kIOUSBPipeStalled`) fell through untranslated, so the existing stall recovery never ran |
| Fix | one `translateIOReturn` entry mapping the stall to `usb.ErrPipeStalled` |
| Verified | v0.9.1 on the reporter's Apple-silicon Mac — R820T2 probe with the full 29-entry gain table, cold boots included |
| Rule that survives | a recovery keyed on platform sentinels doesn't exist on any platform whose backend never translates into them |

## In this post

- **The symptom: the device that was both there and not** — one log screenful contradicting itself.
- **The one probe line** — `sdr list --probe` surfaces the deepest failing operation verbatim.
- **The recovery that already existed** — the cross-platform stall ladder, and the untranslated error that hid from it.
- **Why "works in SDR++" doesn't exonerate your backend** — libusb's silent courtesies.
- **The fix** — one translation entry, tested on the platforms that can test each half.
- **What we keep** — error translation, inherited lies, and retry-versus-reset.

## The symptom: the device that was both there and not

The report's title captured the contradiction: "open device failed although device
is detected." A day-one user on an Apple-silicon Mac (v0.8.8, `darwin/arm64`) got a
startup log that disagreed with itself inside a single screenful — the daemon's own
diagnostics banner *listed* the dongle it had just declared missing:

```text
level=WARN msg="configured SDR not present on the bus; check the cable / dmesg / lsusb"
  serial=77771111153705700
level=WARN msg="daemon: SDR pool open failed" err="no SDR devices opened"
...
level=INFO msg=diagnostics banner="GopherTrunk diagnostics
  ...
  dongles : 1 detected
    - rtlsdr[0] serial=77771111153705700 product=RTL2832U"
```

That WARN points an operator at cables, hubs, and `lsusb`-style checks. The
reporter dutifully chased all of them. Community triage then reached for the
usual config suspects — `blog_v4` was toggled both ways with no change, which
makes sense in hindsight because that flag addresses the RTL-SDR Blog V4's
crystal and input routing, a *tuning* concern, and this failure was in
*bring-up*. A "this brand of dongle isn't 100% compatible with generic RTL-SDRs"
theory got an airing too. Nothing helped, because the message was wrong: the
device was on the bus, enumerated, and answering. What had actually failed was
much deeper and much more specific.

## The one probe line

`gophertrunk sdr list --probe` doesn't just list devices; it attempts a full tuner
bring-up on each and reports the failure verbatim. One line cracked the case:

```text
probe rtlsdr[0]: rtlsdr: tuner init: r82xx init: burst write: rtl2832u:
  I2CWrite addr=0x34: usb: DeviceRequest OUT: usb: IOKit kern_return 0xe000404f
DRIVER  IDX  SERIAL             TUNER  PRODUCT   gains(0.1 dB)
rtlsdr  0    77771111153705700         RTL2832U  []
```

The device row underneath tells the same story in table form: product identified,
serial read — and the `TUNER` column empty with a bare `[]` where 29 gain entries
should be. The dongle answered everything except the one transfer that matters.

Readers of Part 9 will recognize everything up to the last clause: the R820T's
first I²C-bridge burst write during tuner bring-up, the exact transfer that stalled
on Linux (`broken pipe`) and Windows (`ERROR_GEN_FAILURE`). The last clause is the
macOS spelling: `0xe000404f` is IOKit's `kIOUSBPipeStalled` — a transient USB
control-pipe stall, common on a cold boot, and *transient* is the operative word. A
control-pipe (EP0) stall is cleared by the host controller on the next SETUP
packet, so retrying the identical request is the correct and complete recovery.

## The recovery that already existed

Here is the part that stings. GopherTrunk already knew how to handle this stall —
Part 9's issue produced a whole ladder for it: retry the chunk in place, halve the
chunk size, reset and re-run bring-up. That ladder was keyed on the error that
means "pipe stalled" on each platform:

| Platform | Stall surfaces as | Recognized? |
|---|---|---|
| Linux | `syscall.EPIPE` | yes — recovery runs |
| Windows | `ERROR_GEN_FAILURE` | yes — recovery runs |
| macOS | IOKit `kern_return 0xe000404f` | **no — raw code fell through** |

Why does macOS need translating at all? GopherTrunk's macOS backend is pure Go —
no CGO, IOKit and CoreFoundation loaded at runtime — so USB failures don't arrive
as friendly POSIX errno values. They arrive as raw IOKit `kern_return` codes, an
error space of its own, and a translation layer (`translateIOReturn`) is the
bridge that maps the handful of codes the driver must *act on* into the portable
sentinels the cross-platform logic checks. The stall code wasn't in the map. The
kern_return came back wrapped — visible in the error string, as the probe line
shows — but untranslated, so the recovery ladder's `errors.Is` checks matched
nothing, and the first stalled chunk aborted tuner init.

From there the failure cascaded upward, each layer's conclusion locally reasonable
and globally misleading:

| Layer | What it saw | What it concluded |
|---|---|---|
| USB backend | raw `kern_return 0xe000404f` | unrecognized error — fail the write |
| Tuner init | first I²C burst failed | R820T bring-up failed |
| Tuner detection | probe returned no tuner | no supported tuner present |
| Device pool | no usable tuner | filter the device out |
| Daemon | configured serial not in pool | "configured SDR not present on the bus" |

The same stall class had already surfaced on other OS fronts (#753, #922 are its
siblings) — this was simply the macOS spelling of a known failure arriving at a
backend that didn't speak it. And the "isn't 100% compatible" hardware theory is
refuted by the probe line itself: incompatible silicon doesn't fail with a
*transient stall* code on an otherwise clean bring-up.

## Why "works in SDR++" doesn't exonerate your backend

The reporter made the classic (and useful) observation: the same dongle worked in
SDR++ and SDRTrunk on the same Mac. In Parts 9 and 10 that observation pointed at
timing. Here it points at something simpler: those applications reach the dongle
through libusb, and **libusb clears endpoint stalls automatically** as part of its
transfer handling. The stall happens to everyone; libusb users never see it.

So "works in other SDR apps" tells you almost nothing about your own USB backend.
It doesn't mean the hardware is fine *for you* — it means libusb is silently
running a recovery you may not have implemented. GopherTrunk's pure-Go,
no-libusb design (the whole point of which is static binaries and no system
dependencies) means every one of those silent courtesies has to be reimplemented
deliberately, per platform, or it doesn't exist.

## The fix

One entry in the macOS backend's error translation: `translateIOReturn` now maps
`0xe000404f` to the portable `usb.ErrPipeStalled`, mirroring the Windows
`ERROR_GEN_FAILURE` mapping, which the existing recovery ladder already handles.
No new recovery logic, no macOS-specific retry code — just the missing translation
that lets the cross-platform machinery see the stall.

The regression coverage is split the way the bug was: a darwin-gated test asserts
the IOKit stall code maps to `ErrPipeStalled` (the macOS analog of the existing
Windows mapping test), while the Linux-runnable
`TestR82xx_InitBurst_ErrPipeStalled*` tests already pinned the other half of the
chain — that a recognized stall actually triggers the burst-write retry. Each
platform tests the half it can run; together they cover the seam this bug lived
in.

Verified in v0.9.1 on the reporter's hardware, and the closing log is the opening
log's negative image — the same probe, now with a tuner and 29 gain entries where
the empty brackets were:

```text
DRIVER  IDX  SERIAL             TUNER   PRODUCT   gains(0.1 dB)
rtlsdr  0    77771111153705700  R820T2  RTL2832U  [0 9 14 27 37 77 87 125 ... 480 496]

level=INFO msg="device opened" driver=rtlsdr serial=77771111153705700 role=control rate_hz=2400000
level=INFO msg="sdr tuner detected" serial=77771111153705700 product=RTL2832U tuner=R820T2
```

The reporter confirmed cold boots included, and closed the issue themselves.

## What we keep

- **Error translation is load-bearing, not cosmetic.** A recovery path keyed on
  platform-specific sentinel errors exists only on platforms whose backends
  translate into those sentinels. When you add a recovery, grep every backend for
  the raw codes it must map — the fix here was one line, and it was missing for the
  entire life of the macOS backend.
- **Downstream error messages inherit upstream lies.** "Not present on the bus" was
  three layers of reasonable inference sitting on one unrecognized error code. When
  a message contradicts observable reality (the OS *shows* the device), distrust
  the message's layer and go one level down — `sdr list --probe` exists precisely
  to surface the deepest failing operation verbatim, and it's the first step in the
  [diagnostic playbook]({{ '/reference/diagnostic-playbook/' | relative_url }}).
- **"Works in SDR++" means libusb works, not that you do.** Every libusb courtesy —
  automatic stall clearing, transfer pacing, endpoint policy — is an unwritten
  requirement on any backend that replaces it. The full list we've accumulated
  lives in [RTL-SDR USB recovery]({{ '/reference/rtlsdr-usb-recovery/' | relative_url }}).
- **A transient stall wants a retry, not a reset.** EP0 stalls clear on the next
  SETUP; the identical request, re-sent, is the whole cure. Escalating to device
  reset for a transient condition trades a 2 ms recovery for a multi-second one.

*Next: [Part 12]({{ '/blog/solution-postmortem/from-the-issue-tracker-12-seventy-eight-degrees/' | relative_url }})
leaves the RTL family for the Airspy R2 — a device that opened perfectly and decoded
nothing, until one phase measurement named the bug.*

## FAQ

**What is `kern_return 0xe000404f`?**
IOKit's `kIOUSBPipeStalled` — "pipe has stalled, error needs to be cleared." It's
the macOS spelling of the same condition Linux reports as `syscall.EPIPE` and
Windows as `ERROR_GEN_FAILURE`: a USB control-pipe stall, transient and common on
an R820T's first I²C burst during a cold-boot bring-up.

**Why did the daemon claim the SDR wasn't on the bus when the OS clearly listed it?**
Because the conclusion was inherited through three layers. The probe's tuner init
failed on the unrecognized stall, tuner detection therefore concluded no
supported tuner existed, the device was filtered from the pool, and the daemon —
finding the configured serial absent from the pool — reported it "not present on
the bus." Every layer reasoned correctly from its input; only the bottom input
was wrong.

**Why did the same dongle work in SDR++ and SDRTrunk on the same Mac?**
Those applications reach the dongle through libusb, which clears endpoint stalls
automatically as part of its transfer handling. The stall happens to everyone;
libusb users never see it. "Works in other SDR apps" therefore tells you libusb's
recovery works — not that your own backend does.

**Why is retrying the same request the correct recovery rather than resetting the device?**
A control-pipe (EP0) stall is cleared by the host controller on the next SETUP
packet, so re-sending the identical request succeeds — that's the reasoning the
Linux and Windows paths already relied on, now shared by macOS. A full device
reset also works but costs seconds and re-runs the entire bring-up for a
condition that clears in milliseconds.

**Was the dongle brand ever part of the problem?**
No. The affected unit was a 2016-era R820T stick, but the stall is a chip-family
cold-boot behavior GopherTrunk already recovered from on other platforms, and the
same hardware probed perfectly once the translation landed. "Brand X isn't fully
compatible" was the thread's plausible-but-wrong theory; the probe line's
transient stall code is what refuted it.

## Series navigation

**Part 11 of 22** · ←
[Part 10: Faster Than libusb — When the Second Write Outruns the First]({{ '/blog/solution-postmortem/from-the-issue-tracker-10-faster-than-libusb/' | relative_url }})
· Next →
[Part 12: Seventy-Eight Degrees — The Phase Angle That Named the Bug]({{ '/blog/solution-postmortem/from-the-issue-tracker-12-seventy-eight-degrees/' | relative_url }})
