---
title: "From the Issue Tracker, Part 9: Broken Pipe — Six Rounds of Traces for One USB Write"
description: Two NESDR SMArt v5 dongles that worked in every other SDR app failed GopherTrunk's tuner init with one EPIPE. Five parity fixes didn't take, one of them shipped a regression of our own making, and paired USB traces ruled out everything until a 5 ms sleep closed it.
category: solution-postmortem
keywords: rtl-sdr, nesdr smart v5, epipe, broken pipe, i2c burst write, r820t2, usb trace, librtlsdr parity, tuner init, i2c repeater, usb debugging
tags: [from-the-issue-tracker, rtl-sdr, usb, drivers, debugging, postmortem]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "From the Issue Tracker"
series_part: 9
---

*Part 9 of **From the Issue Tracker**, postmortems of GopherTrunk bugs that fought
back. [Part 8]({{ '/blog/solution-postmortem/from-the-issue-tracker-08-nineteen-dibits/' | relative_url }})
closed out the DSP cluster. This part opens a new one: five stories from the USB and
driver layer, where the bug is a NAK on a wire, the debugger is a hex dump of control
transfers, and "works in every other SDR app" is the least helpful clue you'll get.*

> **TL;DR:** Two NESDR SMArt v5 dongles failed GopherTrunk's tuner init with
> `I2CWrite addr=0x34: broken pipe` — while `rtl_test`, SDRTrunk, and p25survey all
> worked on the same host ([#248](https://github.com/MattCheramie/GopherTrunk/issues/248)).
> Six rounds of fixes followed. The diagnostic that made them converge was a pair of
> USB traces from the same session — `RTLSDR_DEBUG_USB=1` on our side,
> `LIBUSB_DEBUG=4` on librtlsdr's — diffed transfer by transfer. Along the way we
> shipped a regression of our own: an "optimization" whose cache guard ate a wire
> write the silicon actually needed. The fix that finally carried it was a 5 ms
> settle between arming the I²C repeater and the first burst write. The wire bytes
> were byte-identical to librtlsdr's all along; the *timing* wasn't.

## The symptom: works everywhere but here

The report was as clean as they come: Ubuntu 24.04, a NESDR SMArt v5, and one line
of log.

```text
level=ERROR msg="open device failed" driver=rtlsdr index=0
  err="rtlsdr: tuner init: r82xx init: burst write: rtl2832u: I2CWrite addr=0x34: broken pipe"
```

`0x34` is the R820T2 tuner's I²C address, so enumeration and interface claim had
succeeded — the daemon just couldn't complete the first multi-byte write to the
tuner. `EPIPE` on a control transfer is USB for "the device refused that request."
Meanwhile `rtl_test` identified the tuner, printed the full 29-entry gain table, and
streamed samples from the same dongle seconds later. The reporter then reproduced it
on a *second* NESDR v5. Two units failing identically while three other programs
succeed is the signature of a systematic software difference, not broken hardware.

## Rounds one through three: parity fixes that didn't take

The obvious theory was drift from librtlsdr, so the first three rounds were
line-for-line audits of our open path against `rtlsdr_open`:

| Round | Change | Result |
|---|---|---|
| 1 | librtlsdr's defensive dummy write (`USB_SYSCTL = 0x09`, reset on failure) before baseband init | Still fails |
| 2 | Chunk the 27-byte R820T init flood at librtlsdr's `NMAX_WRITES = 16` | Still fails |
| 3 | The missing R820T demod-prep sequence (disable Zero-IF, in-phase ADC only, 3.57 MHz IF, spectrum inversion) | Still fails |

Each was real librtlsdr parity work, independently correct, and none of it moved the
symptom. But the error *wrapper strings* were quietly doing diagnostic work each
round: had the warmup write been the failure, the error would have read
`rtlsdr: USB warmup: ...`. It kept coming back as `r82xx init: burst write: ...` —
so each new stage was passing and the failure stayed pinned to the same transfer.
Instrumented error wrapping meant every retest located the failure without a
debugger attached.

## The diagnostic: two traces of the same conversation

Round three also shipped the tool that ended the guessing: `RTLSDR_DEBUG_USB=1`
wraps the USB transport in a logging decorator — one line per control transfer with
request type, value/index, payload hex, and outcome. The reporter captured it
alongside librtlsdr's own debug output, same dongle, same session:

```bash
RTLSDR_DEBUG_USB=1 ./gophertrunk sdr list --probe 2> usb-trace-gophertrunk.log
LIBUSB_DEBUG=4 rtl_test -t 2> usb-trace-rtl-test.log
```

The first diff found something embarrassing. Our last transfers before the EPIPE:

```text
ControlOut wValue=0x1520 wIndex=0x0011 data=01     ← demod prep step 4
ControlIn  wValue=0x0120 wIndex=0x000a             ← commit read
ControlOut wValue=0x0034 wIndex=0x0610 wLength=17  ← FAILING I²C burst
  data=05 83 32 75 c0 40 d6 6c f5 63 75 68 6c 83 80 00 0f
  -> err=broken pipe
```

The 17-byte payload was byte-identical to librtlsdr's first `r82xx_write` chunk.
What was *missing* was the transfer before it: a fresh `SetI2CRepeater(true)` wire
write arming the RTL2832U's I²C bridge.

## The regression we shipped ourselves

That missing write was self-inflicted. Round three had "optimized away" a toggle
that looked like wasteful wire traffic: tuner detection used to switch the I²C
repeater off when it returned, and the round-three change left it on, reasoning the
software cache made re-arming redundant. But `SetI2CRepeater`'s cache guard —
`if d.repON == on { return nil }` — now saw the repeater already on and silently ate
the arm call the burst write depended on. On NESDR v5 silicon, the register already
*holding* the right value isn't enough: the chip needs the fresh wire write to arm
the bridge for a multi-byte transfer. The revert went in with `issue #248` comments
on the tests that had been inverted, so the toggle can't be "optimized" away again
without tripping an explanation.

An optimization that changes what appears on a wire is not a refactor. The cache was
correct about the register's value and wrong about what the transaction itself meant
to the hardware.

## What the next trace ruled out

The revert was necessary and still not sufficient. The next paired trace showed the
`SetI2CRepeater(true)` write present on the wire — and the burst still EPIPE'd. But
that trace bought three rule-outs that reshaped the problem:

- **The endpoint was not halted.** Immediately after the EPIPE, the deferred
  repeater-off write and a commit read both succeeded. A stalled control pipe would
  have failed everything until a clear-halt — so `libusb_clear_halt` was the wrong
  hammer, ruled out by evidence instead of applied on faith.
- **`rtl_test` performs no secret recovery.** Grepping its `LIBUSB_DEBUG=4` trace
  for `reset_device` and `clear_halt`: zero hits. It simply succeeds on equivalent
  wire bytes.
- **A full `USBDEVFS_RESET` didn't clear it.** A retry envelope that reset the
  device and re-ran the whole bring-up produced the identical EPIPE on the second
  pass.

Identical bytes, healthy endpoint, no recovery ritual to copy — the only variable
left was *time*.

## Five milliseconds

librtlsdr reaches the same burst write through libusb's queueing, syscall layers,
and its own function-call overhead — incidental latency between every transfer. Our
Go driver issues the arm write and the 28-byte burst back-to-back with almost none.
The working hypothesis: the chip needs a moment after the repeater arms before it
can absorb a multi-byte OUT, and every other program had been paying that moment by
accident.

The final round shipped two layered defenses: a 5 ms settle between
`SetI2CRepeater(true)` and the first burst write, and a chunk-size halving fallback
(16 → 8 → 4) in case the effective FIFO depth was the real limit. Both dongles
opened immediately — full gain tables, clean probes. And the fallback's marker
string never appeared in the trace:

```bash
grep "tried chunk sizes" trace.log   # no output — the settle alone carried it
```

The 5 ms sleep was the entire fix. Six rounds, five correct-but-insufficient
changes, one regression, and the load-bearing line is a `time.Sleep`.

## What we keep

- **Paired traces beat serial guessing.** One trace shows what you did; two traces
  of the same conversation show what you did *differently*. The
  `RTLSDR_DEBUG_USB=1` / `LIBUSB_DEBUG=4` diff is now a standard play in the
  [diagnostic playbook]({{ '/reference/diagnostic-playbook/' | relative_url }}).
- **Error wrapper strings are instrumentation.** `USB warmup:` vs
  `r82xx init: burst write:` located every failure to a pipeline stage without a
  debugger. Wrap errors with stage names; future-you will read them under pressure.
- **Byte-identical is not behavior-identical.** A thin USB stack removes latency
  that vendor libraries provide by accident, and some silicon has come to depend on
  it. The same lesson returns in Part 10 on a different OS.
- **Caches over wires are contracts with hardware.** A guard that elides a "redundant"
  write assumes the transaction is idempotent from the chip's point of view. Verify
  that on real silicon before trusting it.
- **Rule-outs are progress.** "EP0 is not halted, rtl_test does no reset, the bytes
  match" is what shrank the search space to timing. The recovery ladder that came
  out of all six rounds — retry, settle, chunk-halving, reset — is documented in
  [RTL-SDR USB recovery]({{ '/reference/rtlsdr-usb-recovery/' | relative_url }}).

*Next: [Part 10]({{ '/blog/solution-postmortem/from-the-issue-tracker-10-faster-than-libusb/' | relative_url }})
takes the same lesson to Windows, where being thinner than libusb's stack outran a
clone dongle's firmware.*
