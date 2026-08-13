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

## The symptom: the device that was both there and not

The report's title captured the contradiction: "open device failed although device
is detected." macOS saw the dongle — enumeration listed it, serial and all — but the
daemon refused to run with it:

```text
configured SDR not present on the bus
```

That message points an operator at cables, hubs, and `lsusb`-style checks. The
reporter dutifully chased all of them. Nothing helped, because the message was
wrong: the device was on the bus, enumerated, and answering. What had actually
failed was much deeper and much more specific.

## The one probe line

`gophertrunk sdr list --probe` doesn't just list devices; it attempts a full tuner
bring-up on each and reports the failure verbatim. One line cracked the case:

```text
tuner init: r82xx init: burst write: rtl2832u: I2CWrite addr=0x34:
  usb: DeviceRequest OUT: usb: IOKit kern_return 0xe000404f
```

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

The macOS USB backend returned the kern_return wrapped but untranslated. The
recovery ladder's `errors.Is` checks matched nothing, the first stalled chunk
aborted tuner init, and the failure cascaded upward: probe failed → tuner detection
concluded no tuner present → device filtered out → "configured SDR not present on
the bus." Each layer's conclusion was locally reasonable and globally misleading.
The thread also burned exchanges on a plausible-but-wrong hardware theory — that
this brand of dongle "isn't 100% compatible" — which the probe line refuted:
incompatible silicon doesn't fail with a *transient stall* code on an otherwise
clean bring-up.

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
`0xe000404f` to the portable `usb.ErrPipeStalled`, which the existing recovery
ladder already handles. No new recovery logic, no macOS-specific retry code — just
the missing translation that lets the cross-platform machinery see the stall.

Verified in v0.9.1 on the reporter's hardware: the probe reports an R820T2 with the
full 29-entry gain table, cold boots included.

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
