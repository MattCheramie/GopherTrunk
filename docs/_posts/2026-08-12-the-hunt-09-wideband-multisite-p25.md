---
title: "The Hunt, Part 9: Wideband Multi-Site P25 — Watching Many Channels at Once"
description: How GopherTrunk pins one wideband SDR to a band centre, splits its IQ into a narrowband stream per control channel with a tuner bank, and runs a per-channel P25 (or DMR) state machine on each — so a whole multi-site system's control channels are watched simultaneously off a single dongle.
category: deep-dives
keywords: wideband p25 monitoring, multi-site trunking, tuner bank channelizer, per-channel state machine, ddc polyphase, shared front end gain, control channel parallel decode, single sdr multi channel, gophertrunk the hunt
tags: [the-hunt, p25, dmr, dsp, wideband, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "The Hunt"
series_part: 9
---

*Part 9 of **The Hunt**. By now our stray 851–869 MHz carrier is no longer a
mystery: it locked (Part 7), and its control channel named a **band plan and a
list of neighbour sites** — a multi-site system, not a lone repeater. The map
in the `DiscoveredSystem` says there are three or four control channels out
there. This post is about confirming that: how GopherTrunk watches every one of
those control channels **at the same time** on a single wideband dongle, so the
neighbours the CC advertised actually show up decoding.*

> **TL;DR:** A wideband monitor pins **one SDR** to a band centre and streams
> its whole IQ band continuously. A `tuner.Bank` splits that band into one
> narrowband (48 kHz) stream **per channel**, and each stream feeds a
> protocol-specific receiver plus a per-channel control-channel state machine
> that publishes `cc.locked` / `grant` on the shared bus. So a four-site P25
> system's four control channels decode **in parallel** off one front end —
> `internal/scanner/widebandt2`. The catch is a shared front end: one gain, one
> ADC, one noise floor for every site — and the engine spends most of its
> diagnostics budget making the failure modes of *that* visible.

**Key takeaways**

- **One dongle, N channels.** The engine pins the SDR to a fixed centre and a
  `tuner.Bank` (a per-tap `DDCBank` or a shared polyphase `ChannelizerBank`)
  extracts a 48 kHz stream per control channel — the same narrowband rate the
  single-channel decoder targets, so a wideband tap decodes **identically** to a
  dedicated dongle.
- **Strategy scales with channel count.** At ≤6 channels a per-tap DDC wins on
  simplicity; above that the shared channelizer's amortised wide-band filter is
  the only thing that stays real-time — a dense 71-repeater plan benches ~6×
  cheaper on the channelizer.
- **A shared front end is the whole risk.** One fixed gain can only be right for
  one site, so the engine warns loudly about pinned gain, ADC-clip overload, and
  tuner-offset — the failure modes a per-tap power gauge alone can't see.
- **The engine yields to a hunt.** `Suspend`/`Resume` let a live hunt borrow the
  wideband SDR mid-run and retune it, then hand it back on-channel — so
  discovery and steady-state monitoring share one radio.

## Cheat sheet

| Piece | What it does | Where it lives |
|---|---|---|
| `Engine` | owns one wideband SDR + the per-channel fan-out | `internal/scanner/widebandt2/engine.go` |
| `tuner.Bank` | band IQ → one 48 kHz stream per channel | `internal/dsp/tuner` (`DDCBank`, `ChannelizerBank`) |
| `buildChannel` | wire the right receiver + state machine per protocol | `engine.go` (`buildChannel`) |
| `pickStrategy` | DDC vs polyphase by channel count | `engine.go` (`pickStrategy`) |
| `Suspend`/`Resume` | lend the SDR to a live hunt and take it back | `engine.go` |
| Diagnostics | low-power / overload / no-sync WARNs | `engine.go` (`maybeLogDiagnostics`) |

## In this post

- **Why watch in parallel** — and why the CC hunter alone isn't enough.
- **The tuner bank** — one wide band split into a narrowband stream per channel.
- **DDC vs polyphase** — the strategy that scales with the plan.
- **The shared front end** — one gain, one ADC, and the WARNs that make it legible.
- **Yielding to a hunt** — how the engine lends its SDR mid-run.

## Why watch in parallel

The [control-channel hunter]({{ '/blog/deep-dives/the-hunt-06-control-channel-hunting/' | relative_url }})
does one thing beautifully: it retunes a single radio across candidate
frequencies until one locks. That is exactly right for *finding* a system. It is
the wrong shape for *watching* one. A multi-site P25 system has a control channel
per site, and each site's CC advertises the others as neighbours — but you only
ever see a neighbour advertised, never decoding, if you can only listen to one
frequency at a time. To confirm the topology our carrier claimed — to see RFSS
2 / Site 4 actually broadcasting its own identity — we have to be on all of those
frequencies simultaneously.

You could buy four dongles. Or you could notice that four control channels
spread across a couple of megahertz all fit inside **one** SDR's IQ band. An
Airspy at 10 MS/s covers 10 MHz; a whole 800 MHz P25 system's control channels
live in a fraction of that. `internal/scanner/widebandt2` is the engine that
turns one wide capture into many decoders.

## The tuner bank: one band, many streams

The engine's core idea is a **fan-out**. One goroutine pumps the SDR's IQ chunks;
a `tuner.Bank` decimates that wide stream into one narrowband stream per
configured channel; each stream drives its own receiver and state machine. The
package doc says it plainly:

```go
// internal/scanner/widebandt2/engine.go (shape)
// The dongle is pinned to a configured centre frequency; an
// internal/dsp/tuner.Bank extracts one narrow-band IQ stream per channel
// inside the dongle's IQ band; each stream feeds a protocol-specific
// receiver and per-channel state machine that publishes cc.locked / grant
// events on the bus.
const narrowbandRateHz = 48_000.0 // per-tap output rate
```

That 48 kHz is not arbitrary — it is the **same** rate the single-channel
`ccdecoder` down-converter targets (≈10 samples/symbol at 4800 baud). A P25
receiver sizes its matched filter and clock loop to *that* rate, not the capture
rate, so a channel decoded via a wideband tap behaves identically to one on its
own dongle. This is the rate-invariance property the whole codebase leans on,
applied one level up: the bank absorbs the capture rate, the receiver never sees
it.

Each channel is added as a **tap** — an offset from the dongle centre plus a sink
that folds power for diagnostics and feeds the receiver:

```go
// internal/scanner/widebandt2/engine.go (shape) — New, per channel
offset := float64(ch.FrequencyHz) - float64(opts.CenterFreqHz)
ec, _ := buildChannel(sys, ch, bank.OutputRateHz(), opts.Bus, log, opts.Now)
sink := func(out []complex64) {
    ec.pwr.Add(out)          // window-averaged dBFS gauge
    ec.receiver.Process(out) // → DibitSink → per-channel state machine
}
bank.AddTap(offset, sink)
```

`buildChannel` is where a channel's protocol identity enters. It dispatches on
the referenced system's protocol and wires the matching receiver to the matching
state machine — DMR Tier II conventional, DMR Tier III trunked, P25 Phase 1, or
P25 Phase 2 — behind two tiny interfaces (`narrowbandReceiver.Process([]complex64)`
and `channelProcessor.Process(dibits, baseIdx)`) so the engine treats every
channel uniformly. For the trunked protocols it also enforces a rule: the tap's
frequency **must** be one of the system's declared `control_channels`
(`requireControlChannel`) — a control-channel state machine only makes sense on a
control channel.

<figure class="lab-figure">
<svg viewBox="0 0 680 220" width="680" height="220" role="img" aria-label="One wideband SDR pinned to a band centre streams IQ into a tuner bank, which splits it into one 48 kHz narrowband stream per control channel; each stream feeds a per-channel P25 or DMR receiver and control-channel state machine, all of which publish lock and grant events onto the shared event bus.">
  <rect x="8" y="92" width="104" height="40" rx="6" fill="none" stroke="currentColor"/>
  <text x="60" y="110" text-anchor="middle" fill="currentColor" font-size="11">wideband SDR</text>
  <text x="60" y="125" text-anchor="middle" fill="var(--fg-muted)" font-size="9">one IQ band</text>
  <line x1="112" y1="112" x2="140" y2="112" stroke="currentColor"/><polygon points="140,108 150,112 140,116" fill="currentColor"/>
  <rect x="150" y="80" width="110" height="64" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="205" y="104" text-anchor="middle" fill="var(--accent)" font-size="11">tuner.Bank</text>
  <text x="205" y="119" text-anchor="middle" fill="var(--fg-muted)" font-size="9">split → 48 kHz</text>
  <text x="205" y="132" text-anchor="middle" fill="var(--fg-muted)" font-size="9">per channel</text>
  <line x1="260" y1="92" x2="292" y2="58" stroke="currentColor"/><polygon points="288,55 298,54 293,63" fill="currentColor"/>
  <line x1="260" y1="112" x2="292" y2="112" stroke="currentColor"/><polygon points="292,108 302,112 292,116" fill="currentColor"/>
  <line x1="260" y1="132" x2="292" y2="166" stroke="currentColor"/><polygon points="293,161 298,170 288,169" fill="currentColor"/>
  <rect x="302" y="40" width="150" height="36" rx="6" fill="none" stroke="currentColor"/>
  <text x="377" y="56" text-anchor="middle" fill="currentColor" font-size="10">rx + CC state (site 1)</text>
  <text x="377" y="69" text-anchor="middle" fill="var(--fg-muted)" font-size="9">p25 phase 1</text>
  <rect x="302" y="94" width="150" height="36" rx="6" fill="none" stroke="currentColor"/>
  <text x="377" y="110" text-anchor="middle" fill="currentColor" font-size="10">rx + CC state (site 2)</text>
  <text x="377" y="123" text-anchor="middle" fill="var(--fg-muted)" font-size="9">p25 phase 1</text>
  <rect x="302" y="148" width="150" height="36" rx="6" fill="none" stroke="currentColor"/>
  <text x="377" y="164" text-anchor="middle" fill="currentColor" font-size="10">rx + CC state (site 3)</text>
  <text x="377" y="177" text-anchor="middle" fill="var(--fg-muted)" font-size="9">dmr tier III</text>
  <line x1="452" y1="58" x2="512" y2="104" stroke="currentColor"/><polygon points="508,101 518,104 509,110" fill="currentColor"/>
  <line x1="452" y1="112" x2="512" y2="112" stroke="currentColor"/><polygon points="512,108 522,112 512,116" fill="currentColor"/>
  <line x1="452" y1="166" x2="512" y2="120" stroke="currentColor"/><polygon points="509,114 518,120 508,123" fill="currentColor"/>
  <rect x="522" y="92" width="150" height="40" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="597" y="110" text-anchor="middle" fill="var(--accent)" font-size="11">event bus</text>
  <text x="597" y="125" text-anchor="middle" fill="var(--fg-muted)" font-size="9">cc.locked · grant</text>
  <text x="340" y="208" text-anchor="middle" fill="var(--fg-muted)" font-size="10">one shared front end, one ADC, one noise floor for every site</text>
</svg>
<figcaption>The wideband engine fans one IQ band out into a narrowband stream per control channel, each with its own receiver and state machine, all publishing to the same bus.</figcaption>
</figure>

## DDC vs polyphase: the strategy that scales

There are two ways to split the band, and the right one depends on how many
channels you are watching. `pickStrategy` chooses:

```go
// internal/scanner/widebandt2/engine.go (shape)
const strategyAutoThreshold = 6 // DDC below, channelizer above
func pickStrategy(requested string, channelCount int) (kind, tag string) {
    switch requested {
    case "ddc":       return "ddc", "ddc"
    case "polyphase": return "polyphase", "polyphase"
    case "", "auto":
        if channelCount <= strategyAutoThreshold {
            return "ddc", "auto(ddc)" // a handful of taps: linear, no bin constraint
        }
        return "polyphase", "auto(polyphase)" // many taps: one shared filter
    }
    // …explicit values honoured verbatim
}
```

A `DDCBank` runs an independent NCO + decimator per tap — perfect for a few
control channels, no bin-alignment constraints. But it scales linearly: 70 taps
means 70 reduced-rate resamplers, and the host stops keeping up. A
`ChannelizerBank` instead runs **one** shared polyphase filter/FFT across the
whole band and reads each tap out of a bin — a dense 71-repeater DMR plan benches
**~6× cheaper** on the channelizer than on a per-tap DDC. The cost is that a bin
has a fixed width, so a carrier that lands near a bin edge decodes at reduced
SNR. The engine keeps the bin width roughly constant (`channelizerBinsFor` aims
for ~150 kHz per bin, so a 10 MS/s band gets ~64 bins instead of 16), and when a
dense plan crowds taps onto bin edges it **warns** rather than silently
degrading — the channelizer is still the only bank that stays real-time at that
tap count, so the trade-off is made visible, not avoided.

### How that principle shaped the Go code

- **Two interfaces hide every protocol.** `narrowbandReceiver` and
  `channelProcessor` are each one method. `buildChannel` is the only place that
  knows P25 from DMR; the pump loop drives whatever it built.
- **The bank sizes itself from the plan.** `New` pre-computes every tap's offset
  and the widest one, so a `DDCBank` can size its shared decimator to keep the
  outermost tap in band and the channelizer can pick a bin count — the plan
  drives the DSP, not a fixed constant.
- **Rate-invariance is reused, not reinvented.** Because taps emit at 48 kHz, the
  per-channel receivers are byte-for-byte the ones the dedicated-dongle path
  builds. A wideband channel is not a second decoder — it is the same decoder fed
  differently.

## The shared front end is the whole risk

Everything above is upside. The downside is a single sentence: **one dongle means
one gain, one ADC, and one noise floor for every site.** A control-channel
hunter tunes to one site and the AGC settles on it; a wideband engine serves four
sites at four signal strengths through one front end, and a single fixed gain can
only be right for one of them. The strongest co-tenant sets the level and a
weaker site sits below the shared ADC floor and never decodes. So the engine
warns — once, at startup, before any traffic:

```go
// internal/scanner/widebandt2/engine.go (shape) — New
if len(opts.Channels) > 1 {
    if tenthDB, fixed := fixedGainTenthDB(opts.Gain); fixed {
        log.Warn("widebandt2: multi-channel wideband device is pinned to a fixed " +
            "gain — one gain can't serve sites at different signal strengths off a " +
            "shared front end … try 'gain: auto' (AGC) on this dongle (issue #749)")
    }
}
```

The runtime diagnostics (`maybeLogDiagnostics`, once per power window) round it
out, and each one localises a different fault:

- **Channel power very low** but the *wideband input* is healthy → the carrier is
  outside the captured passband or mistuned, **or** a weak co-tenant is under a
  fixed gain — the WARN suggests `gain: auto`.
- **Wideband input clipping** (ADC rail-pinned above a small ratio) → a strong
  site is overloading the shared ADC and burying the weak taps. The fix is
  *less* RF, not more — the WARN says so explicitly, because the instinct is
  always to raise gain.
- **Strong in-channel signal but zero sync for three windows** → the classic
  fingerprint of an uncorrected tuner PPM error. A narrowband 4-level demod
  tolerates only ~±75 Hz of carrier offset while an FM audio app still sounds
  fine, so a mistuned dongle "works" everywhere except here (issue #836).

None of these are decode logic — they are **operator feedback**, and on a shared
front end they are the difference between "one site decodes and I don't know why
the others don't" and a WARN that names the cause.

## Yielding to a hunt

There is one more piece that ties this post back to the series. A live hunt needs
an SDR, and a wideband dongle *is* an SDR sitting right on the band you want to
survey. So the engine can lend it. `Suspend` makes the pump keep draining the IQ
stream (so the SDR never blocks) but process nothing; `Resume` reprograms the
dongle back to the engine's centre frequency — because the borrower retuned
it — and clears the flag:

```go
// internal/scanner/widebandt2/engine.go (shape)
func (e *Engine) Resume() {
    if !e.suspended.Load() {
        return
    }
    // The borrower retuned the shared SDR; put it back on-channel first.
    if err := e.device.SetCenterFreq(e.centerHz); err != nil {
        e.log.Warn("widebandt2: resume retune failed; resuming anyway", "err", err)
    }
    e.suspended.Store(false)
}
```

Without `Suspend`, an engine whose SDR had been retuned by a hunt would decode
off-frequency IQ — flooding the log with low-power WARNs and emitting spurious
grants whose voice taps capture silence. With it, the same dongle that watches
our system's four control channels all day can be borrowed for a five-minute
survey and handed back on-channel, no drift. Discovery and steady-state
monitoring share one radio.

## Where this goes next

We have now watched our system's control channels in parallel and confirmed its
sites. But every result so far came off the air, live. [Part 10]({{ '/blog/deep-dives/the-hunt-10-offline-vs-live-surveys/' | relative_url }})
asks the question that makes any of this trustworthy: does a find **reproduce**?
It walks the offline survey path — replaying a recorded capture through the same
sweep → identify → map pipeline — and shows why what you decode from a `.cfile`
is byte-for-byte what you'd decode on the air.

## FAQ

**Is this the same decoder as a single-channel control receiver?**
The middle is identical. Every tap emits at 48 kHz, the same rate the
single-channel `ccdecoder` down-converter targets, and the per-channel receivers
are the same P25/DMR receivers built with the same constants. Only the front —
one shared bank instead of one dedicated down-converter — differs.

**When should I force `polyphase` instead of `auto`?**
Rarely — `auto` picks the channelizer above six channels, which is the right call
for a dense plan. Force `ddc` only for a handful of channels where you want no
bin-alignment constraint at all; force `polyphase` if you have a mid-count plan
whose host is falling behind on per-tap DDC cost.

**Why does one site decode and the others don't?**
Almost always the shared front end. A fixed gain suits the strongest site and
buries the weak ones below the ADC floor, or a strong site is clipping the ADC
and raising the noise floor for everyone. Try `gain: auto`, and check the
overload WARN — the fix there is to *reduce* gain, not raise it.

**Can one wideband dongle both watch a system and run a hunt?**
Yes — that is what `Suspend`/`Resume` are for. A live hunt borrows the SDR,
retunes it to sweep, and hands it back; `Resume` reprograms it to the engine's
centre so the next chunk is on-channel again.

**Does the wideband engine decode voice too?**
It can — published grants flow through the trunking engine's voice-pool
allocator, which can bind a grant to a per-grant DDC tap on this same wideband IQ
stream (a "virtual voice tap"), so a wideband dongle can record its own voice
grants without a separate voice SDR when the grant lands inside its IQ window.

## Series navigation

**Part 9 of 14** · ←
[Part 8: DMR LCN Correlation — Rebuilding a Channel Map]({{ '/blog/deep-dives/the-hunt-08-dmr-lcn-correlation/' | relative_url }})
· Next →
[Part 10: Offline vs Live Surveys — Hunting a Recording]({{ '/blog/deep-dives/the-hunt-10-offline-vs-live-surveys/' | relative_url }})
