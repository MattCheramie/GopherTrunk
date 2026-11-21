---
title: "Squelch, Part 10: Bursty Carriers — Direct Mode, Stalled Streams and Decoders That Give Up"
description: "What a scan list does when the carrier is not continuous or the IQ stream is not reliable: a DMR direct-mode over cut into hangtime-sized recordings and the 100 ms burst bridge that fixed it, a stream reopen that raced its own teardown, a phantom 23-second call on a stalled stream, and single-channel decoders that retry instead of giving up."
category: deep-dives
keywords: dmr direct mode scanner segmented recordings, tdma gap hangtime, squelch activity debounce, rtlsdr stream already active, iq stream stall timeout, phantom call on silent channel, sdr decoder retry backoff, setcenterfreq broken pipe retry, conventional scanner dwell loop, gophertrunk bursty carrier
tags: [squelch, dmr, direct-mode, usb, scanner, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "Squelch"
series_part: 10
---

*Part 10 of **Squelch**, a 14-part deep dive into GopherTrunk's conventional
scanner — the scan list that has to infer every call from the air — and the
gates that decide when one of its channels opens.
[Part 9]({{ '/blog/deep-dives/squelch-09-call-gates-ysf-nxdn-dstar/' | relative_url }})
finished the per-protocol call gates. Every gate so far leans on two things
it never checks: that a call's carrier stays up for the length of the call,
and that IQ keeps arriving while the scanner listens. This part is four
field failures in which one of those was false — and one question that is
still open.*

> **TL;DR:** The dwell loop ends a call when the channel stays below squelch
> for the hangtime, and ignores activity shorter than the 50 ms
> `ActivityDebounce`. A DMR direct-mode handheld is off 32.5 ms of every
> 60 ms and its 27.5 ms bursts never outlast the debounce, so one over
> became a string of 1.5–2.5 s recordings; a DMR channel now bridges gaps
> shorter than `dmrBurstBridge` (100 ms). The same run's `rtlsdr: stream
> already active` was a reopen racing the cancelled stream's teardown;
> `StreamIQ` now waits a cancelled stream out. A dwell whose stream *stops*
> without closing held a phantom call for 23 s; `StreamStallTimeout` (3 s)
> ends it with `EndReasonError`. And single-channel decoders that gave up
> for ever on one failed tune now run through `runSingleChannelDecoder`,
> retrying at 500 ms doubling to 30 s. Why two dongles stalled at the same
> instant is not root-caused.

**Key takeaways**

- **A debounce tuned for one carrier breaks another.** The rule that stops
  analog blips from holding a channel open is exactly what stopped a TDMA
  burst from keeping one open.
- **Chunks that never arrive are not silence.** Hangtime measures quiet in
  delivered samples; a stalled stream delivers none, so only a clock can end
  that dwell.
- **Cancelled is not the same as stopped.** A stream's teardown finishes
  some time after its context is cancelled; a reopen has to wait for it, not
  race it.
- **One failed tune must not be for ever.** A pinned decoder's tune, open
  and decode pass are all retried with a bounded backoff — and the stall
  that prompted it is still unexplained.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| Trailing edge | countdown runs 3 dB under the open level; only 50 ms of sustained activity resets it | `scanner.go` (`beginDwell`, `defaultActivityDebounce`) |
| DMR burst bridge | a gap shorter than 100 ms after the last loud chunk still counts as active | `scanner.go` (`dmrBurstBridge`) |
| Onset run | the same 100 ms defines one transmission for the onset mark | `scanner.go` (`onsetGap`, `streamPos`) |
| Stream reopen | waits for a cancelled stream's teardown; still refuses a live one | `internal/sdr/rtlsdr/purego/stream.go` (`StreamIQ`, `streamDone`) |
| Stream stall | no chunk for 3 s ends the call with an error and re-opens the stream | `scanner.go` (`StreamStallTimeout`, `DefaultStreamStallTimeout`) |
| Decoder retry | tune, open and decode retried, 500 ms doubling to a 30 s ceiling | `cmd/gophertrunk/daemon.go` (`runSingleChannelDecoder`, `tuneWithRetry`) |
| Pins | one over is one call; analog still releases; a stalled dwell ends | `dmr_burst_test.go`, `scanner_test.go`, `single_channel_retry_test.go` |

## In this post

- **How a dwell decides a call is over** — hysteresis, debounce, hangtime.
- **Direct mode: every gap looked like a release** — and the 100 ms bridge.
- **A reopen that raced its own teardown** — `stream already active`, again.
- **The call that never ended** — a stream that stops without closing.
- **Decoders that gave up for ever** — retry, with a bounded backoff.
- **What is still open** — the simultaneous stall, and NXDN48 then and now.

## How a dwell decides a call is over

Once a channel opens, `beginDwell` announces the synthetic call and then
reads IQ, classifying each chunk as active or not. Three numbers shape the
trailing edge. The squelch opens at `squelch_dbfs` but a chunk only counts
as inactive once it falls a hysteresis margin — 3 dB by default — below
that. Continuous inactivity for the hangtime, 1500 ms by default, ends the
call with a normal end. And once the countdown is running, activity must be
*sustained* for `ActivityDebounce`, 50 ms by default, before it resets:

```go
// internal/scanner/conventional/scanner.go (shape)
if active {
    if aboveSince.IsZero() {
        aboveSince = now
    }
    if belowSince.IsZero() || now.Sub(aboveSince) >= ch.ActivityDebounce {
        belowSince = time.Time{} // countdown cleared
    }
} else {
    aboveSince = time.Time{}
    if belowSince.IsZero() {
        belowSince = now
    } else if now.Sub(belowSince) >= ch.Hangtime {
        s.endDwell(v, trunking.EndReasonNormal)
        return
    }
}
```

The debounce exists for a good reason
([#1090](https://github.com/MattCheramie/GopherTrunk/issues/1090)): a
single above-threshold chunk — a carrier tail, impulse noise, intermod —
used to zero the countdown, so a channel peppered with blips hung open
indefinitely. Fifty milliseconds is well above one chunk and well below the
shortest real syllable. Every word of that is right for an analog carrier
and for a repeater, and it encodes one assumption: a transmission, while it
lasts, is continuous.

## Direct mode: every gap looked like a release

On the 14 Oct field run of
[#1261](https://github.com/MattCheramie/GopherTrunk/issues/1261) a `dmr`
scan-list channel sat on a simplex frequency with a handheld on it. The
report, as `dmr_burst_test.go` quotes it: "DMR jumps out of decode even
with a good signal, the test audio is segmented". One over had become a
string of 1.5–2.5 s recordings.

A [direct-mode]({{ '/reference/direct-mode-operation/' | relative_url }})
radio does not transmit continuously. It sends one 27.5 ms burst per 60 ms
[TDMA]({{ '/reference/tdma/' | relative_url }}) frame and is off for the
other 32.5 ms — the cadence that blinded the DMR receiver itself in
[DMR End to End Part 9]({{ '/blog/deep-dives/dmr-end-to-end-09-direct-mode-carrier-gate/' | relative_url }}).
Walk it through the snippet above. The first gap is inactive, so
`belowSince` is set and the countdown starts. The next burst is active and
sets `aboveSince` — but the burst is over after 27.5 ms, before the 50 ms
debounce can clear the countdown, and the following gap wipes `aboveSince`
again. The in-channel power drops below squelch about sixteen times a
second and not once does it stay up long enough to count. When the hangtime
runs out the call ends, mid-sentence; the next burst opens a new one; and
so on down the over.

The same log held its own control: the one ten-second call on that channel
had source 9990 — the Parrot, heard through a repeater, which is a
continuous carrier.

The fix is one clause ahead of the debounce:

```go
// internal/scanner/conventional/scanner.go (shape)
active := lv >= keepAlive
if active {
    lastAbove = lastChunk
} else if bridge > 0 && lastChunk.Sub(lastAbove) < bridge {
    active = true // a TDMA gap inside one over, not a release
}
```

`bridge` is `dmrBurstBridge` = 100 ms on a `dmr` channel and zero on every
other mode. A hundred milliseconds covers the 32.5 ms gap with room for
chunk granularity and one lost burst, and it adds at most that much to the
real release at the end of the over. Analog channels keep the #1090 rule
untouched.

<figure class="lab-figure">
<svg viewBox="0 0 680 240" width="680" height="240" role="img" aria-label="A timeline of a DMR direct-mode over: eight 27.5 millisecond bursts, one per 60 millisecond frame. Judged chunk by chunk, the hangtime countdown starts at the first gap and no burst lasts the 50 milliseconds needed to reset it. With the 100 millisecond burst bridge the channel stays active for the whole over.">
  <text x="340" y="14" text-anchor="middle" fill="currentColor" font-size="10" font-weight="bold">DMR direct mode: 27.5 ms on · 32.5 ms off · 60 ms frame</text>
  <text x="64" y="47" text-anchor="end" fill="var(--fg-muted)" font-size="8">carrier</text>
  <line x1="70" y1="56" x2="650" y2="56" stroke="var(--fg-muted)"/>
  <rect x="70" y="32" width="27.5" height="24" fill="none" stroke="currentColor" stroke-width="1.5"/>
  <rect x="130" y="32" width="27.5" height="24" fill="none" stroke="currentColor" stroke-width="1.5"/>
  <rect x="190" y="32" width="27.5" height="24" fill="none" stroke="currentColor" stroke-width="1.5"/>
  <rect x="250" y="32" width="27.5" height="24" fill="none" stroke="currentColor" stroke-width="1.5"/>
  <rect x="310" y="32" width="27.5" height="24" fill="none" stroke="currentColor" stroke-width="1.5"/>
  <rect x="370" y="32" width="27.5" height="24" fill="none" stroke="currentColor" stroke-width="1.5"/>
  <rect x="430" y="32" width="27.5" height="24" fill="none" stroke="currentColor" stroke-width="1.5"/>
  <rect x="490" y="32" width="27.5" height="24" fill="none" stroke="currentColor" stroke-width="1.5"/>
  <text x="584" y="47" text-anchor="middle" fill="var(--fg-muted)" font-size="8">over ends</text>
  <text x="64" y="97" text-anchor="end" fill="var(--fg-muted)" font-size="8">before</text>
  <rect x="70" y="86" width="27.5" height="14" fill="none" stroke="var(--fg-muted)" stroke-dasharray="2 2"/>
  <rect x="130" y="86" width="27.5" height="14" fill="none" stroke="var(--fg-muted)" stroke-dasharray="2 2"/>
  <rect x="190" y="86" width="27.5" height="14" fill="none" stroke="var(--fg-muted)" stroke-dasharray="2 2"/>
  <rect x="250" y="86" width="27.5" height="14" fill="none" stroke="var(--fg-muted)" stroke-dasharray="2 2"/>
  <rect x="310" y="86" width="27.5" height="14" fill="none" stroke="var(--fg-muted)" stroke-dasharray="2 2"/>
  <rect x="370" y="86" width="27.5" height="14" fill="none" stroke="var(--fg-muted)" stroke-dasharray="2 2"/>
  <rect x="430" y="86" width="27.5" height="14" fill="none" stroke="var(--fg-muted)" stroke-dasharray="2 2"/>
  <rect x="490" y="86" width="27.5" height="14" fill="none" stroke="var(--fg-muted)" stroke-dasharray="2 2"/>
  <line x1="97.5" y1="112" x2="650" y2="112" stroke="currentColor" stroke-width="1.5"/>
  <line x1="97.5" y1="106" x2="97.5" y2="118" stroke="currentColor" stroke-width="1.5"/>
  <text x="104" y="128" fill="currentColor" font-size="8">hangtime countdown: starts at the first gap, and no 27.5 ms burst lasts the 50 ms needed to reset it</text>
  <text x="104" y="141" fill="var(--fg-muted)" font-size="8">→ the call ends mid-over, the next burst opens another</text>
  <text x="64" y="177" text-anchor="end" fill="var(--fg-muted)" font-size="8">bridged</text>
  <rect x="70" y="166" width="547.5" height="14" fill="none" stroke="var(--accent)" stroke-width="1.5"/>
  <text x="294" y="177" text-anchor="middle" fill="var(--accent)" font-size="8">active for the whole over</text>
  <line x1="517.5" y1="160" x2="517.5" y2="186" stroke="var(--fg-muted)" stroke-dasharray="2 2"/>
  <text x="567" y="177" text-anchor="middle" fill="var(--accent)" font-size="8">bridge 100 ms</text>
  <line x1="617.5" y1="192" x2="650" y2="192" stroke="currentColor" stroke-width="1.5"/>
  <line x1="617.5" y1="186" x2="617.5" y2="198" stroke="currentColor" stroke-width="1.5"/>
  <text x="646" y="210" text-anchor="end" fill="currentColor" font-size="8">the countdown starts here, once, after the last burst</text>
</svg>
<figcaption>The debounce that protects an analog channel from blips made every TDMA gap count as a release; the bridge treats a gap under 100 ms as part of the over.</figcaption>
</figure>

Two tests pin both sides. `TestConvScannerDMRDirectModeOverIsOneCall` plays
40 frames — 2.4 s, eight times the test's hangtime — and requires exactly
one call with one normal end. `TestConvScannerAnalogDoesNotBridgeGaps`
plays the *same* burst pattern into an FM channel and requires it to
release on hangtime while the pattern runs. The 100 ms also defines
`onsetGap`, so one over's bursts are one run when the scanner marks where a
transmission began; and Part 8's DMR call gate, fed every chunk with its
1 s hold, does not flicker with the gaps either.

## A reopen that raced its own teardown

The same run's log carried a second line: `conv: StreamIQ failed`, with
`rtlsdr: stream already active`. Readers of
[From the Issue Tracker Part 16]({{ '/blog/solution-postmortem/from-the-issue-tracker-16-conventional-fm-broker/' | relative_url }})
have met that error as two consumers contending for one device. This was
one consumer, twice in a row.

The scanner cancels and re-opens its IQ stream on every tune, deliberately,
to drop any in-flight samples from the previous channel. In the pure-Go
RTL-SDR driver a cancelled stream's teardown runs on its own goroutine, and
stopping the bulk transfer can take a while. `StreamIQ` refused to start
while a stream was still registered — so a reopen that arrived before the
teardown finished lost the race, the scanner logged the WARN, slept 200 ms
and moved on.

The driver now records the context a stream was opened with and a channel
that closes when its teardown completes:

```go
// internal/sdr/rtlsdr/purego/stream.go (shape)
if d.streamCtx == nil || d.streamCtx.Err() == nil || d.streamDone == nil {
    return nil, errors.New("rtlsdr: stream already active") // a live consumer
}
done := d.streamDone
d.streamMu.Unlock()
select {
case <-done: // the cancelled stream has finished tearing down
case <-ctx.Done():
    d.streamMu.Lock()
    return nil, ctx.Err()
}
```

A stream whose context is still live is a genuine second consumer and is
still refused. One whose context is already cancelled is waited out, and
the wait is bounded by the caller's own context so shutdown cannot hang in
it. `TestStreamIQ_ReopenAfterCancelWaitsForTeardown` slows the transport's
stop to 100 ms and reopens immediately. The Airspy-family drivers had met
the same race earlier
([RF Front End Part 12]({{ '/blog/deep-dives/rf-front-end-12-sdr-pool-usb-watchdog/' | relative_url }})).

## The call that never ended

The third failure comes from the 16 Sep run of
[#1184](https://github.com/MattCheramie/GopherTrunk/issues/1184) — the
first live test of the FleetSync decoder from
[Beyond Voice Part 4]({{ '/blog/deep-dives/beyond-voice-04-fleetsync-wav-that-lied/' | relative_url }}).
The config pinned one RTL-SDR to
`fleetsync.channels` and gave a second to `scanner.conventional`, with no
trunking system. Each section worked alone. Together, 146 ms after start
the FleetSync receiver logged `SetCenterFreq failed`: the tuner's PLL write
had come back as a broken pipe after every retry stalled, the
[control-pipe stall]({{ '/reference/rtlsdr-usb-recovery/' | relative_url }})
of
[From the Issue Tracker Part 9]({{ '/blog/solution-postmortem/from-the-issue-tracker-09-broken-pipe/' | relative_url }}).
At +627 ms the scanner opened a synthetic call on a silent 146.67 MHz
channel. It ran for 23 s and ended, `reason=error`, only when the operator
pressed Ctrl-C.

The project's notes record that opening but not its cause; this section is
about why the call could not close. The scanner's stream had stopped
delivering chunks *without closing*. Look at the dwell loop again: hangtime
is counted in inactive chunks, and none arrived. The closed-channel branch
never fired, because nothing closed. And a 250 ms ticker kept touching the
engine's watchdog on the scanner's behalf, so the engine's own timeout
could not reap the call either. One wedged pump held a phantom call open
until the process died.

The ticker now carries a clock of its own:

```go
// internal/scanner/conventional/scanner.go (shape)
case <-ticker.C:
    stalled := s.opts.Now().Sub(lastChunk)
    if stalled >= s.opts.StreamStallTimeout {
        // WARN "conv: IQ stream stalled during dwell — ending call and
        // re-opening the stream", with no_samples_for
        s.endDwell(v, trunking.EndReasonError)
        return
    }
    s.opts.Engine.Touch(s.opts.DeviceSerial)
```

`DefaultStreamStallTimeout` is 3 s — long enough that an RTL-SDR's
few-millisecond chunks or a throttled remote stream never trip it, short
enough that a dead pump releases the channel before an operator notices.
The call ends with an *error*, not a normal end, because a stalled stream
is not a hangtime; `Run` then re-tunes and re-opens.
`TestConvScannerStalledStreamEndsDwell` feeds one loud chunk and then
nothing, and requires the end, the reason and the reopen.

## Decoders that gave up for ever

The other half of that run was the FleetSync receiver itself. Its failed
tune was transient — the driver recovers from that stall in place a moment
later — but nothing ever asked again. Every single-channel decoder block
abandoned its receiver for the daemon's whole lifetime on one failed tune
or one failed open.

`runSingleChannelDecoder` replaced that shape with a loop: tune (retried
by `tuneWithRetry` until it succeeds or the daemon stops), open the IQ
stream, decode until the stream ends — then do it again. The wait between
attempts is `singleChannelRetryBackoff`: 500 ms, doubling, capped at 30 s,
and reset once a pass has run for more than a minute. Only a serial with no
device behind it at all is a permanent skip, logged once. MDC1200 and
FleetSync were converted first; every single-channel decoder — POCSAG,
FLEX, M17, APRS, AIS, LoRa, DSC, ADS-B — runs through it now.

The operator sees `fleetsync: SetCenterFreq failed — retrying` with a
`retry_in`, then `fleetsync: SetCenterFreq recovered` with the attempt
count, and `decoder stopped — restarting` with `ran_for` when a stream
dies. `TestTuneWithRetryRecoversFromStartupStall` fails a tuner three times
with the rig's own error text and requires the fourth call;
`TestSingleChannelRetryBackoffIsBoundedAndGrows` pins the schedule.

## What is still open

**Why both dongles stalled at the same instant is not root-caused.** Two
RTL-SDRs failing within half a second of each other are unlikely to be two
independent faults. The standing suspicion is a USB reset on one dongle
disturbing a sibling on the same hub, but the reporter's log excerpt starts
at the banner, so the device-open and tuner diagnostic lines — and whether
the two dongles share a hub — are the next instrument. One trap for whoever
reads that log: the start-up banner showing `rtlsdr[0]` twice is not a
double open. The pool snapshot used to drop the device index; it is carried
now.

**None of the four fixes has a confirming live run recorded.** The stall
timeout and the retry loop are pinned failing-first by their tests; the
burst bridge and the reopen wait are on-air-gated until the reporter's
scan-list run on a build that has them.

**NXDN48, then and now.** The 14 Oct run also established that the NXDN
receiver of the day was NXDN96 only: a 6.25 kHz NXDN48 channel was not
decoded at all, which the documentation had denied. That has since changed
— `nxdn48` is its own scan-list mode, pinned on a real-air slice, with the
call gate of Part 9.

### How bursty carriers shaped the Go code

- **Mode-specific timing is one constant with its arithmetic beside it.**
  The comment on `dmrBurstBridge` carries the 27.5 / 32.5 / 60 ms cadence,
  so the number cannot be "tidied" without its reason.
- **Liveness is measured, not assumed.** `lastChunk` is stamped on every
  delivery; a timer that only *refreshed* a watchdog was the bug.
- **End reasons carry the diagnosis.** A stalled stream ends as an error
  and a hangtime as normal, so the call log distinguishes a quiet channel
  from a dead pump.
- **Retry is the wrapper, not the decoder's job.** One function owns tune,
  open and restart for every pinned decoder, and its backoff is a variable
  so tests can shrink it.

## Where this goes next

The FleetSync receiver in this story needed a whole dongle pinned to one
frequency — while the scanner next to it was already listening to that
kind of channel. The reporter asked the obvious question.
[Part 11]({{ '/blog/deep-dives/squelch-11-data-on-the-scan-list/' | relative_url }})
is the answer: MDC1200, FleetSync and ACARS decoders that hang off a
scan-list channel's own IQ.

## FAQ

**Why does GopherTrunk split one DMR simplex transmission into several short recordings?**
It did before the burst bridge. A direct-mode radio is off for 32.5 ms of
every 60 ms frame, so each gap started the hangtime countdown and no
27.5 ms burst could outlast the 50 ms activity debounce to reset it. A
`dmr` scan-list channel now treats a gap shorter than 100 ms as part of the
same over.

**What does "rtlsdr: stream already active" mean on a conventional scanner?**
The scanner cancels and re-opens its IQ stream on every retune. If the
reopen arrived while the cancelled stream's teardown was still running, the
RTL-SDR driver refused it. `StreamIQ` now waits for a cancelled stream to
finish tearing down and only refuses when another consumer's stream is
still live.

**Why did a call stay open on a silent channel until I stopped the daemon?**
The IQ stream had stalled without closing. Hangtime counts silent chunks,
and a stalled stream delivers none, so the call could not end. A dwell that
receives no samples for `StreamStallTimeout` (3 s) now ends the call with
an error and re-opens the stream.

**What happens if an SDR fails to tune at start-up?**
A single-channel decoder used to give up for the daemon's lifetime. Now the
tune is retried — 500 ms, doubling to a 30 s ceiling — and so are a failed
stream open and a decoder whose stream dies. The log shows
`SetCenterFreq failed — retrying` and later `SetCenterFreq recovered`.

**Is the direct-mode burst bridge verified on air?**
Not yet. It is pinned by a synthetic regression that reproduces the field
report's burst cadence, alongside a test that analog channels still release
normally, but the reporter's confirming run is not recorded.

## Series navigation

**Part 10 of 14** · ←
[Part 9: Call Gates II — System Fusion, NXDN and D-STAR]({{ '/blog/deep-dives/squelch-09-call-gates-ysf-nxdn-dstar/' | relative_url }})
· Next →
[Part 11: Data on the Scan List — MDC1200, FleetSync and ACARS Taps]({{ '/blog/deep-dives/squelch-11-data-on-the-scan-list/' | relative_url }})
