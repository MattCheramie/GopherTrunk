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

## Cheat sheet

| Layer | Symptom | First read | Real cause | Fix |
|---|---|---|---|---|
| Hardware | decoding stops minutes or hours after a clean lock | decoder stall / deadlock | NESDR repeatedly dropping off the USB bus and re-enumerating | daemon now survives it loudly; rule out cable / hub power / EMI / thermal |
| Error handling | one retry, then the daemon ran on with no decoder | recovery machinery working | retry matched only `ErrIQStreamClosed`; the rebuild's `usb: device disconnected` didn't | wrap open errors with `%w`; exhaust 1/2/5/10 s; exit non-zero |
| Protocol | id 10 grants dropped `stage=no-bandplan` | site not broadcasting id 10 | `IDEN_UP_TDMA` (`0x33`) never dispatched — proven by `p25_survey` | dispatch entry + deferred-grant ring + `p25_band_plan` seed |

## In this post

- **The symptom: a lock, then silence** — the report, and the stall theory it invited.
- **The diagnostic: read the other log** — the kernel-log correlation that ended the concurrency hunt.
- **Bug one: the retry loop that matched one sentinel** — how the recovery mechanism became the outage.
- **Bug two: the grants going nowhere** — `stage=no-bandplan`, and the oracle that proved the site innocent.
- **One issue, three fixes** — the layered shape of the whole thing.
- **What we keep** — durable lessons, cross-linked to the Field Guide.

## The symptom: a lock, then silence

The report described the pattern every scanner developer dreads: the daemon starts,
hunts, locks the control channel cleanly, decodes traffic — and then, minutes or
hours in, the event stream just stops. No error, no crash, no exit. The obvious
reading is a stall: a deadlock, a stuck goroutine, a channel nobody drains. That
reading sent the first round of investigation into the pipeline's concurrency,
where there was nothing to find.

## The diagnostic: read the other log

The proof came from outside the process — though it took one build's worth of added
visibility to get there. An earlier round on the issue had widened the retry path's
logging, and on the next live session the "stall" stopped being silent. The
daemon's last normal lines were routine decode traffic — a vendor TSBK, a patch
update, two VUHF identifier updates — and then:

```
WARN msg="daemon: ccdecoder: IQ stream died; retrying" attempt=1 max_attempts=4 backoff=1s
     err="ccdecoder: IQ stream closed unexpectedly"
WARN msg="daemon: component exited with error" component=ccdecoder
     err="rtlsdr: reset buffer: rtl2832u: write block=1 addr=0x2148 val=0x1002: usb: device disconnected"
```

The kernel had been narrating the whole time:

```
usb 1-5: USB disconnect, device number 31
usb 1-5: new high-speed USB device number 32 using xhci_hcd
usb 1-5: Product: NESDR SMArt v5
usb 1-5: SerialNumber: 76361606
```

Timestamp-match the two logs and the story writes itself: the daemon's
`19:19:52 IQ stream died` and the kernel's `19:19:53 USB disconnect` are the same
event seen from two sides, every time. The NESDR was physically leaving the bus and
re-enumerating with a new device number — the classic signature of a marginal
cable, an underpowered hub, EMI, or a dongle running hot — and the same serial had
disconnect/reconnect events scattered across the whole day. The decoder never
stalled; its sample source ceased to exist mid-read, and before the added
visibility GopherTrunk had nothing to say about it.

Correlating your application log against `dmesg` timestamps costs one terminal and
two minutes, and it cleanly splits the world: if the hardware event precedes the
software silence, stop debugging the software's logic and start debugging its
*error handling*. Which is exactly where the next bug was.

## Bug one: the retry loop that matched one sentinel

GopherTrunk did have a recovery mechanism: `runCCDecoderWithRetry`, which catches a
dying control-channel decoder and rebuilds it with backoff. It watched for one
error: `ccdecoder.ErrIQStreamClosed`.

Here's the sequence on a real disconnect, exactly as the two WARN lines above
record it:

1. The USB reaper dies; the IQ channel closes; the decoder returns
   `ErrIQStreamClosed`. The sentinel matches — the loop backs off one second and
   rebuilds the decoder. So far so good; that is the first WARN.
2. The rebuilt decoder calls `StreamIQ` on the **same dead Tuner** — the physical
   device has detached, so the handle is dead. `ResetBuffer` fails; that's the
   `write block=1 addr=0x2148 val=0x1002: usb: device disconnected` in the second
   WARN.
3. That error is not `ErrIQStreamClosed`. No match, no retry — the loop treated it
   as unknown and bailed.
4. Because the ccdecoder task is spawned `essential=false`, its death didn't take
   the daemon down: the web UI stayed up, the API answered, the process looked
   alive. **Half-dead** — the worst failure mode, because nothing supervising the
   process (systemd, Docker, a human glancing at the dashboard) had any cue to
   restart it.

The fix has three parts, each generalizable:

- **Widen the sentinel at the seam, not the matcher.** Non-context errors from the
  `StreamIQ` open path are now wrapped as `ErrIQStreamClosed` via `%w`, so every
  flavor of "the device went away" funnels into the one error the retry loop
  understands — mid-stream EOF *and* open-time disconnect on rebuild feed the same
  path.
- **Exhaust a real backoff ladder** — 1, 2, 5, 10 seconds, about 18 seconds in
  total — giving a re-enumerating dongle time to come back as a new device.
- **Then die loudly.** If the ladder is exhausted, exit non-zero so the supervisor
  restarts the whole process against fresh device handles; on startup the SDR pool
  re-discovers the re-enumerated dongle by serial. A daemon that can't do its job
  must not keep pretending it can.

The expected sequence on flaky hardware is now unambiguous:

```
WARN  daemon: ccdecoder: IQ stream died; retrying attempt=1 ...
WARN  daemon: ccdecoder: IQ stream died; retrying attempt=2 ...
WARN  daemon: ccdecoder: IQ stream died; retrying attempt=3 ...
WARN  daemon: ccdecoder: IQ stream died; retrying attempt=4 ...
ERROR daemon: ccdecoder: IQ stream died and retries exhausted; escalating to fatal
```

A regression test (`TestRunCCDecoderWithRetry_USBDisconnectEscalatesToFatal`)
reproduces the exact mid-stream-death-then-open-disconnect pattern from the field
logs. One tempting extra was deliberately deferred: in-process re-acquire — polling
for the same serial to reappear and swapping it into the running decoder without a
restart. More resilient on paper, substantially more code (pool re-enumeration,
driver re-open, tuner-state restoration), and unjustified until there's evidence
the supervisor-restart path isn't enough.

## Bug two: the grants going nowhere

With the daemon now surviving disconnects, the reporter's logs got clean enough to
show a second, entirely unrelated problem. Voice grants referenced channel
`id=10` — and every one of them was dropped with `stage=no-bandplan`:

```
p25: grant before identifier update nac=356 id=10 num=176
     err="p25/phase1: no IdentifierUpdate for channel ID: id=10"
```

The daemon received identifier updates for ids 2, 3, 4, 7, 8 and 15, over and
over — never for id 10. With no `IDEN_UP` (Identifier Update) for that id, it had
no way to turn "channel 10-176" into a frequency, and grants on that identifier
were silently black-holed. This is exactly why grants showed up in the logs while
no call or talkgroup ever reached the UI.

Was the system not sending it, or was GopherTrunk not hearing it? The first
response hedged toward the former: a deferred-grant queue and a `p25_band_plan`
config seed (both below) were built so an operator could bridge any genuinely
missing entry. Then the reporter used a trick worth stealing: **run a second,
independent decoder as an oracle.** GopherTrunk ships `p25_survey`, a standalone
control-channel survey tool with its own, more complete TSBK parsing. Pointed at
the same site, it printed the full band plan almost immediately:

```
420.08750 MHz  WACN BEE00  SYS 164  NAC 164
RFSS / Site:  2 / 7
Band plan:    16 entries
  iden 8:  base 467.51250 MHz, step 6.25 kHz, offset -10 MHz [FDMA]
  iden 10: base 467.51250 MHz, step 6.25 kHz, offset -10 MHz [TDMA x2]
  ...
```

(The survey prints the NAC in hex — its `164` is the daemon's decimal `nac=356`.)
Sixteen entries, and the pattern was the
tell: eight FDMA idens — precisely the ones the daemon had been logging — and
eight TDMA×2 twins (ids 0, 1, 5, 9, 10, 11, 12, 13) it had never once mentioned.
Id 10 is the TDMA×2 twin of FDMA id 8: same 467.5125 MHz base, same −10 MHz
offset. The site broadcast it constantly. The daemon was deaf to it — deaf,
specifically, to every TDMA-flavored identifier update.

The deafness was one missing case. `dispatchTSBK` handled the FDMA Identifier
Update variants — `0x34` (VHF/UHF) and `0x3D` (standard) — but
`OpIdentifierUpdateTDMA = 0x33` **was never dispatched**. The message had a parser;
nothing routed messages to it. Per TIA-102.AABF Table 14, the TDMA variant's
frequency fields pack identically to the VHF/UHF variant — only the lower nibble of
byte 0 differs — so the fix was a dispatch entry and a thin decode, not new
parsing machinery.

With `0x33` dispatched, Mt Anakie's `id=10 num=176` resolves to
467.5125 MHz + 176 × 6.25 kHz = **468.6125 MHz** — a number now pinned in a
regression test so the TDMA packing can't silently rot. On the next build the
reporter's log gained the lines it had been missing all along:

```
DEBUG p25: identifier update (TDMA) nac=356 id=10 base_hz=467512500 spacing_hz=6250 tx_offset_hz=-10000000
DEBUG p25: grant ... id=10 num=176 freq_hz=468612500
```

The two hedges stayed in, for the next time a band-plan entry goes missing for any
reason:

| Guard | What it does |
|---|---|
| Deferred-grant ring | grants for an unknown channel id are held (cap 4 per id, 5 s TTL) and replayed when the `IDEN_UP` arrives, instead of dropped |
| `p25_band_plan` YAML seed | an operator can pre-load band-plan entries from config as an escape hatch; over-the-air `IDEN_UP`s override the seed |

The ring turns the race case — a grant landing moments before its identifier
update — into a self-healing hiccup with its own log signature:

```
DEBUG p25: grant before identifier update nac=356 id=10 num=176
DEBUG p25: identifier update (TDMA) nac=356 id=10 ...
DEBUG p25: draining deferred grants nac=356 id=10 count=1
DEBUG p25: grant ... freq_hz=468612500
```

## One issue, three fixes

It's worth pausing on the shape of this issue, because "the decoder stalls" was
never one bug. It was a hardware condition (dongle off the bus), an error-handling
bug (one-sentinel retry → half-dead daemon), and a protocol-coverage bug (opcode
`0x33` undispatched) — three different layers, each hiding behind the one above it.
The kernel log peeled the first; surviving the disconnect peeled the second; a
clean log plus an oracle tool peeled the third.

The ending was as clean as the layering: retesting on the fixed build, the
reporter closed this issue and the flagship lock issue
([#275](https://github.com/MattCheramie/GopherTrunk/issues/275)) together — stable
control-channel lock, FDMA and TDMA identifier updates decoding, grants resolving
to valid frequencies, calls appearing in both UIs. Their sign-off named the
method: "the p25_survey output ended up being the missing clue."

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

## FAQ

**Why didn't the daemon just crash when the dongle disconnected?**
By design, the control-channel decoder runs as a non-essential task so a decoder
fault can't take down the API and web UI. That's the right call for a decode bug —
and exactly wrong for "the radio is gone," which removes the daemon's purpose. The
fix keeps the task non-essential but escalates to a non-zero process exit once the
retry ladder is exhausted, so the supervisor sees a death it can act on.

**What was making the dongle leave the bus in the first place?**
Nothing GopherTrunk can fix. Repeated `USB disconnect` plus instant re-enumeration
on the same serial is a hardware or host-side condition — a marginal cable, a hub
exceeding its power budget, EMI, or thermal — and the reporter's kernel log showed
the same NESDR dropping multiple times across a day. The daemon's job is to
survive it loudly; ruling out the physical causes is the operator's.

**Why restart the whole process instead of re-acquiring the dongle in-process?**
Re-acquire-by-serial was considered and deliberately deferred: doing it right
needs pool re-enumeration, driver re-open, and tuner-state restoration, and a
supervisor restart reaches the same end state because the SDR pool re-discovers
the re-enumerated dongle by serial at startup. If field evidence ever shows
restart-thrashing, in-process re-acquire is the next lever.

**How would I notice grants being black-holed on my own system?**
Two log signatures: `p25: grant before identifier update` naming a channel id that
never gets an identifier update, and drops tagged `stage=no-bandplan`. Cross-check
with `p25_survey`: if it prints a band-plan entry your daemon never logs, the gap
is in dispatch, not on the air. The `p25_band_plan` seed bridges you until a
decoder fix lands.

**Why did the site have two identifier entries for the same frequencies?**
P25 sites advertise FDMA and TDMA channel plans as separate identifiers, and this
site carried both — eight FDMA entries and their eight TDMA×2 twins over the same
bases and offsets. TDMA voice grants reference the TDMA identifiers, so a decoder
that only dispatches the FDMA variants resolves none of them.

## Series navigation

**Part 18 of 22** · ←
[Part 17: Placeholder Constants — The TETRA Sync That Never Existed]({{ '/blog/solution-postmortem/from-the-issue-tracker-17-placeholder-constants/' | relative_url }})
· Next →
[Part 19: One Render Loop — A Blank UI, a Host-less URL, and React Error #185]({{ '/blog/solution-postmortem/from-the-issue-tracker-19-one-render-loop/' | relative_url }})
