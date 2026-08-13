---
title: "From the Issue Tracker, Part 18: The Stall That Wasn't — A Dongle Off the Bus and an Opcode Off the Books"
description: A decoder that "stalled" after a clean control-channel lock turned out to be a dongle leaving the USB bus, a retry loop that matched only one error sentinel and left the daemon half-dead, and — found by running p25_survey as an oracle — a TDMA IDEN_UP opcode the dispatcher never handled.
category: solution-postmortem
keywords: decoder stall, usb disconnect, kernel log, dmesg, retry sentinel, error wrapping, iden_up tdma, 0x33, band plan, no-bandplan, p25 survey, gophertrunk issue tracker
tags: [from-the-issue-tracker, p25, usb, rtl-sdr, reliability, debugging, postmortem]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "From the Issue Tracker"
series_part: 18
---

*Part 18 of **From the Issue Tracker**, postmortems of GopherTrunk bugs that fought
back. [Part 17]({{ '/blog/solution-postmortem/from-the-issue-tracker-17-placeholder-constants/' | relative_url }})
ended with a decoder fixed and a capture too weak to prove it. This one starts with
a decoder accused of stalling — and ends three layers down, with a hardware event
the software never mentioned, a retry loop that gave up for the wrong reason, and a
protocol opcode that had simply never been wired up.*

> **TL;DR:** In [#345](https://github.com/MattCheramie/GopherTrunk/issues/345), the
> decoder pipeline "stalled" after a successful P25 control-channel lock. The
> kernel log told the real story: `usb 1-5: USB disconnect` timestamp-matched to
> the last decoded line — the NESDR was repeatedly dropping off the bus, and
> GopherTrunk had been silent about it. The retry loop only matched one error
> sentinel (`ErrIQStreamClosed`); the *actual* error after re-open
> (`usb: device disconnected`) didn't match, so the loop bailed and, because the
> decoder task was non-essential, the daemon ran on half-dead. Fixing that exposed
> the genuine decoder gap: grants referenced channel id 10 and were black-holed
> `stage=no-bandplan`, because `IDEN_UP_TDMA` (opcode `0x33`) was never dispatched
> — proven by running `p25_survey` against the same site as an oracle.

## The symptom: a lock, then silence

The report described the pattern every scanner developer dreads: the daemon starts,
hunts, locks the control channel cleanly, decodes traffic — and then, minutes or
hours in, the event stream just stops. No error, no crash, no exit. The obvious
reading is a stall: a deadlock, a stuck goroutine, a channel nobody drains. That
reading sent the first round of investigation into the pipeline's concurrency,
where there was nothing to find.

## The diagnostic: read the other log

The proof came from outside the process. The kernel had been narrating the whole
time:

```
usb 1-5: USB disconnect, device number 31
usb 1-5: new high-speed USB device number 32 using xhci_hcd
```

Timestamp-match that against the daemon's log and the story writes itself: the last
successfully decoded line lands moments before the disconnect, every time. The
NESDR was physically leaving the bus and re-enumerating with a new device number —
the classic signature of a marginal cable, an underpowered hub, EMI, or a dongle
running hot. The decoder never stalled; its sample source ceased to exist mid-read,
and GopherTrunk had nothing to say about it.

Correlating your application log against `dmesg` timestamps costs one terminal and
two minutes, and it cleanly splits the world: if the hardware event precedes the
software silence, stop debugging the software's logic and start debugging its
*error handling*. Which is exactly where the next bug was.

## Bug one: the retry loop that matched one sentinel

GopherTrunk did have a recovery mechanism: `runCCDecoderWithRetry`, which catches a
dying control-channel decoder and rebuilds it with backoff. It watched for one
error: `ccdecoder.ErrIQStreamClosed`.

Here's the sequence on a real disconnect. The stream dies; the sentinel matches;
the loop backs off and rebuilds the decoder — so far so good. The rebuilt decoder
then calls `StreamIQ` on the **same dead Tuner**, whose `ResetBuffer` returns:

```
usb: device disconnected
```

That error is not `ErrIQStreamClosed`. No match, no retry — the loop treated it as
unknown and bailed. And because the ccdecoder task is spawned `essential=false`,
its death didn't take the daemon down: the web UI stayed up, the API answered, the
process looked alive. **Half-dead** — the worst failure mode, because nothing
supervising the process (systemd, Docker, a human glancing at the dashboard) had
any cue to restart it.

The fix has three parts, each generalizable:

- **Widen the sentinel at the seam, not the matcher.** Non-context errors from the
  `StreamIQ` open path are now wrapped as `ErrIQStreamClosed` via `%w`, so every
  flavor of "the device went away" funnels into the one error the retry loop
  understands.
- **Exhaust a real backoff ladder** — 1, 2, 5, 10 seconds — giving a re-enumerating
  dongle time to come back as a new device.
- **Then die loudly.** If the ladder is exhausted, exit non-zero so the supervisor
  restarts the whole process against fresh device handles. A daemon that can't do
  its job must not keep pretending it can.

## Bug two: the grants going nowhere

With the daemon now surviving disconnects, the reporter's logs got clean enough to
show a second, entirely unrelated problem. Voice grants referenced channel
`id=10` — and every one of them was dropped with `stage=no-bandplan`: the decoder
had never received an `IDEN_UP` (Identifier Update) for id 10, so it had no way to
turn "channel 10-1234" into a frequency. Grants on that identifier were silently
black-holed.

Was the system not sending it, or was GopherTrunk not hearing it? This is where the
investigation used a trick worth stealing: **run a second, independent decoder as
an oracle.** GopherTrunk ships `p25_survey`, a standalone control-channel survey
tool with its own, more complete TSBK parsing. Pointed at the same site, it printed
the full band plan — all 16 identifier entries — including id 10, which turned out
to be the TDMA×2 twin of FDMA id 8. The site broadcast it constantly. The daemon
was deaf to it.

The deafness was one missing case. `dispatchTSBK` handled the FDMA Identifier
Update variants — `0x34` (VHF/UHF) and `0x3D` (standard) — but
`OpIdentifierUpdateTDMA = 0x33` **was never dispatched**. The message had a parser;
nothing routed messages to it. Per TIA-102.AABF Table 14, the TDMA variant's
frequency fields pack identically to the VHF/UHF variant — only the lower nibble of
byte 0 differs — so the fix was a dispatch entry and a thin decode, not new
parsing machinery.

Two belt-and-braces guards landed alongside it, for the next time a band-plan entry
goes missing for any reason:

| Guard | What it does |
|---|---|
| Deferred-grant ring | grants for an unknown channel id are held (cap 4 per id, 5 s TTL) and replayed when the `IDEN_UP` arrives, instead of dropped |
| `p25_band_plan` YAML seed | an operator can pre-load band-plan entries from config as an escape hatch |

## One issue, three fixes

It's worth pausing on the shape of this issue, because "the decoder stalls" was
never one bug. It was a hardware condition (dongle off the bus), an error-handling
bug (one-sentinel retry → half-dead daemon), and a protocol-coverage bug (opcode
`0x33` undispatched) — three different layers, each hiding behind the one above it.
The kernel log peeled the first; surviving the disconnect peeled the second; a
clean log plus an oracle tool peeled the third.

## What we keep

- **Correlate with the kernel log before debugging a "stall."** A USB disconnect
  timestamp-matched to the last good line ends the concurrency hunt immediately.
  Recovery procedure and failure signatures:
  [RTL-SDR USB recovery]({{ '/reference/rtlsdr-usb-recovery/' | relative_url }}).
- **Retry loops must match error *classes*, not one sentinel.** Wrap at the seam
  with `%w` so every "device gone" variant funnels to the sentinel the loop knows
  — or the retry machinery itself becomes the outage.
- **Half-dead is worse than dead.** A non-essential task whose death removes the
  daemon's purpose should escalate to a non-zero exit; supervisors can only
  restart what dies visibly.
- **Use an independent decoder as an oracle.** `p25_survey` proved the site was
  broadcasting what the daemon wasn't hearing — the fastest possible split between
  "they don't send it" and "we don't parse it." It's a standing entry in the
  [diagnostic playbook]({{ '/reference/diagnostic-playbook/' | relative_url }}).
- **Grant black-holes need a buffer.** Deferring unknown-channel grants briefly
  turns a missing band-plan entry from silent loss into a self-healing hiccup.
  Opcode tables and packing notes:
  [P25 on-air constants]({{ '/reference/p25-onair-constants/' | relative_url }}).

## Series navigation

← [Part 17: placeholder constants]({{ '/blog/solution-postmortem/from-the-issue-tracker-17-placeholder-constants/' | relative_url }})
· Next → [Part 19: one render loop]({{ '/blog/solution-postmortem/from-the-issue-tracker-19-one-render-loop/' | relative_url }})
