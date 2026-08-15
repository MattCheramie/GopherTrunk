---
slug: rtlsdr-usb-recovery
title: RTL-SDR USB stall recovery
entry_type: term
category: fn-hardware
description: "RTL-SDR USB stall recovery is the collected fix-lore for tuner-init USB stalls — Linux broken pipe, Windows access denied and general failure, macOS IOKit pipe stall — with the per-OS recovery facts and the traps that misdirect the diagnosis."
keywords: rtl-sdr, usb, epipe, broken pipe, error_gen_failure, access denied, iokit, 0xe000404f, pipe stall, winusb, libusb, dmesg, tuner init, nesdr
aka: [RTL-SDR broken pipe, EPIPE at tuner init, USB pipe stall recovery]
infobox:
  - { label: Type, value: Failure family + fixes }
  - { label: Applies to, value: RTL2832U / R820T dongles }
  - { label: Symptom, value: "OS-specific USB error on the first tuner-init I²C burst" }
  - { label: Key rule, value: A dongle that works in libusb tools proves nothing about the thin pure-Go path }
see_also: [rtl-sdr, rtl2832u, r820t-tuner, nesdr, usb, zadig, sdr-gain-overload, airspy-rate-selection, diagnostic-playbook, signal-signatures]
related_reading:
  - { title: "From the Issue Tracker, Part 9: Broken Pipe — Six Rounds of Traces for One USB Write", url: /blog/solution-postmortem/from-the-issue-tracker-09-broken-pipe/ }
  - { title: "From the Issue Tracker, Part 10: Faster Than libusb — When the Second Write Outruns the First", url: /blog/solution-postmortem/from-the-issue-tracker-10-faster-than-libusb/ }
  - { title: "From the Issue Tracker, Part 11: Detected but Not Present — One Hex Code from a Fix That Already Existed", url: /blog/solution-postmortem/from-the-issue-tracker-11-detected-but-not-present/ }
cite_urls:
  - https://github.com/MattCheramie/GopherTrunk/issues/248
  - https://github.com/MattCheramie/GopherTrunk/issues/395
  - https://github.com/MattCheramie/GopherTrunk/issues/333
  - https://github.com/MattCheramie/GopherTrunk/issues/1038
  - https://github.com/MattCheramie/GopherTrunk/issues/345
---

**RTL-SDR USB stall recovery** is the collected fix-lore for one failure family: the
dongle enumerates fine, then the very first burst of tuner-init I²C traffic (or an
early baseband write) fails with an OS-specific USB error. The
[R820T](/reference/r820t-tuner/)-family tuner behind the
[RTL2832U](/reference/rtl2832u/) bridge is sensitive to *timing* on its control-pipe
writes, and every operating system reports the resulting stall with a different — and
differently misleading — error string.

## Symptom table by OS

| Symptom | Looks like | Actually | Fix/Check |
|---|---|---|---|
| Linux: `r82xx init: burst write: rtl2832u: I2CWrite addr=0x34: broken pipe` (EPIPE) | Dead dongle / bad cable | The tuner's I²C bridge was not ready for a multi-byte write; the chip needs settle time after the I²C repeater is enabled | Fixed in GopherTrunk with a 5 ms chip-settle after `SetI2CRepeater(true)` ([#248](https://github.com/MattCheramie/GopherTrunk/issues/248)); update, then retest with `gophertrunk sdr list --probe` |
| Windows 10: `ERROR_GEN_FAILURE` on baseband init step 0 | Wedged clone firmware needing more resets | A race *inside one attempt*: GopherTrunk calls WinUSB directly and its second byte-identical write lands while clone firmware is still digesting the first — libusb's per-transfer overhead was incidentally hiding the race | Fixed with a 10 ms settle between warmup and baseband init ([#395](https://github.com/MattCheramie/GopherTrunk/issues/395)); no retry loop can help, because the failure is between two transfers within a single attempt |
| Windows 11: access denied (`ERROR_ACCESS_DENIED`) opening the SDR | Driver/permissions problem ([Zadig](/reference/zadig/)) | The config listed the same serial twice (e.g. once as `role: control`, once as `role: voice`); Windows refused the second `CreateFile` on the already-open device | Remove the duplicate — now rejected at config load ([#333](https://github.com/MattCheramie/GopherTrunk/issues/333)) |
| macOS: "configured SDR not present on the bus" though the device is detected | Cable / lsusb problem | IOKit `kern_return 0xe000404f` (`kIOUSBPipeStalled`) — a transient control-pipe stall on the R820T's first I²C burst, common on cold boot; the failed probe made detection conclude no tuner exists | Fixed by mapping the IOKit code to the existing stall-retry path ([#1038](https://github.com/MattCheramie/GopherTrunk/issues/1038)); a control-pipe stall clears on the next SETUP, so retrying the identical request is the correct recovery |
| Decode goes silent mid-run, ~0% CPU, no errors | Pipeline stall / deadlock | The dongle dropped off the bus and re-enumerated (marginal cable, hub power, EMI, thermal) | Check `dmesg` for `USB disconnect, device number N` timestamp-matching the last decoded line ([#345](https://github.com/MattCheramie/GopherTrunk/issues/345)) |

## "Works in SDR++" does not exonerate the dongle path

The single most repeated misdirection in these threads: the same dongle works in
SDR++, SDRTrunk, or `rtl_test`, so the reporter (reasonably) concludes the hardware is
fine and GopherTrunk is broken — or the maintainer concludes the dongle is an
incompatible clone. Both readings miss that those tools sit on **libusb, which clears
endpoint stalls automatically** and adds per-transfer latency that hides timing races.
GopherTrunk's pure-Go [USB](/reference/usb/) stack is several layers thinner, so it
both outruns slow clone firmware ([#395](https://github.com/MattCheramie/GopherTrunk/issues/395))
and sees raw stalls libusb would have absorbed
([#1038](https://github.com/MattCheramie/GopherTrunk/issues/1038)). A dongle that works
elsewhere tells you the *hardware* is capable; it says nothing about whether the thin
path needs a settle delay or a stall retry.

The same class of subtlety cut the other way in
[#248](https://github.com/MattCheramie/GopherTrunk/issues/248): a prior "optimization"
removed a redundant-looking I²C repeater off-toggle, and a register cache then
swallowed the subsequent on-write — but on NESDR v5 silicon the chip needed the *fresh
wire write*, not just the register value, to arm the I²C bridge.

## Recovery semantics worth knowing

- **A control-pipe (EP0) stall is self-clearing.** The host controller clears it on the
  next SETUP packet, so the correct recovery is to retry the identical request —
  GopherTrunk retries per-chunk with chunk-size halving. `libusb_clear_halt` is the
  wrong hammer for EP0 (verified in #248: the endpoint was not actually halted).
- **WinUSB "reset" is not a device reset.** `WinUsb_ResetPipe(0)` only clears a stalled
  endpoint. Recovering firmware wedged by a previously crashed process needs a full
  device-handle rebind: clear-halt, drop the WinUSB handles, re-`CreateFile` +
  `WinUsb_Initialize`. Some clone dongles need **two** resets with backoff
  ([#333](https://github.com/MattCheramie/GopherTrunk/issues/333)).
- **A retry loop is only as good as its error matching.** In
  [#345](https://github.com/MattCheramie/GopherTrunk/issues/345) the decoder-restart
  loop matched one sentinel error; the post-disconnect error was a different string, so
  the loop bailed and the daemon ran on half-dead. If a component silently stops
  retrying, check what error text it actually saw.

## Diagnostic tools

- `gophertrunk sdr list --probe` — one line of probe output cracked
  [#1038](https://github.com/MattCheramie/GopherTrunk/issues/1038) by printing the raw
  IOKit kern_return the normal path swallowed.
- Paired USB traces: run GopherTrunk with `RTLSDR_DEBUG_USB=1` and a known-good tool
  (`rtl_test`) with `LIBUSB_DEBUG=4` in the same session, then diff the wire traffic.
  Six rounds of exactly this cracked
  [#248](https://github.com/MattCheramie/GopherTrunk/issues/248) — and reading the
  error *wrapper* prefix (`USB warmup:` vs `r82xx init: burst write:`) shows which
  layer fired and whether a fix moved the failure.
- `dmesg` (Linux) for disconnect/re-enumeration lines when the symptom is silence
  rather than an error.

See the [diagnostic playbook](/reference/diagnostic-playbook/) for where these steps
sit in the wider escalation ladder.

## Provenance

- [#248](https://github.com/MattCheramie/GopherTrunk/issues/248) — NESDR SMArt v5 EPIPE at tuner init; six rounds of paired traces; the 5 ms chip-settle fix.
- [#395](https://github.com/MattCheramie/GopherTrunk/issues/395) — Windows 10 `ERROR_GEN_FAILURE`: GopherTrunk outran clone firmware between two byte-identical writes; libusb's overhead was hiding the race.
- [#333](https://github.com/MattCheramie/GopherTrunk/issues/333) — Windows access denied from a duplicated serial in config; WinUSB reset-vs-rebind semantics; double reset for clones.
- [#1038](https://github.com/MattCheramie/GopherTrunk/issues/1038) — macOS IOKit `0xe000404f` pipe stall masquerading as "device detected but not present."
- [#345](https://github.com/MattCheramie/GopherTrunk/issues/345) — the "pipeline stall" that was a USB disconnect, proven by `dmesg` correlation; the too-narrow retry sentinel.
