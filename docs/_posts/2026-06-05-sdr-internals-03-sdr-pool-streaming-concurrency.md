---
title: "SDR in Pure Go, Part 3: The SDR Pool & Streaming Concurrency"
description: How GopherTrunk streams IQ samples through Go channels and goroutines, manages a fleet of SDR dongles with a device pool, and fans one stream to many consumers without blocking the decode path.
category: deep-dives
tags: [sdr, go, concurrency, goroutines, channels, software-design]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "SDR Internals"
series_part: 3
---

*Part 3 of **SDR Internals**. IQ samples never stop arriving, so the radio is
really a concurrency problem. This post is about the goroutines, channels, and
back-pressure that move samples through the engine — and the pool that herds a
fleet of dongles.*

## In this post

- The **pipes-and-filters + CSP** model: every stage a goroutine, every seam a
  channel.
- The **SDR pool** — enumerating, role-assigning, and supervising multiple
  dongles.
- The **IQ tap broker** that copies one stream to many consumers without ever
  stalling the decoder.

## What "streaming concurrency" means for a radio

At 2.4 MS/s an RTL-SDR produces about 2.4 million complex samples every second,
forever. Nothing in the pipeline can pause to think. The natural Go answer is a
**pipeline of goroutines connected by channels**: each device's USB reader runs
in its own goroutine and pushes `[]complex64` chunks onto a buffered channel;
each downstream DSP stage reads from one channel and writes to the next. This is
the textbook *pipes-and-filters* architecture, and it maps directly onto Go's
CSP-style concurrency.

```go
out, err := device.StreamIQ(ctx)   // <-chan []complex64
for chunk := range out {
    // hand the chunk to the next stage…
}
```

Because the channels are **buffered**, back-pressure flows naturally: if a
downstream stage falls behind, the buffer fills, the producer blocks, and the
system self-regulates instead of growing an unbounded queue. Cancel the
`context` and every goroutine in the chain unwinds cleanly.

## How GopherTrunk implements it in Go

### The pool: a fleet of dongles

A serious scanner uses more than one SDR — one locked to a control channel,
others chasing voice grants. `internal/sdr/pool.go` defines a `Pool` that:

- **enumerates and opens** every attached device via the registry from
  [Part 2]({{ '/blog/deep-dives/sdr-internals-02-sdr-device-driver-registry/' | relative_url }}),
- **assigns roles** — `RoleControl`, `RoleVoice`, `RoleWideband` — so the engine
  can ask the pool for "a voice device" rather than juggling serial numbers,
- **supervises** them with a watchdog (`watchdog.go`) that detects USB
  disconnects and re-enumerates, so a bumped cable heals itself.

```go
type Role int

const (
    RoleAuto Role = iota
    RoleControl   // locked to a system control channel
    RoleVoice     // retunes to granted voice frequencies
    RoleWideband  // one wide capture, many virtual tuners
)
```

<figure class="lab-figure">
<svg viewBox="0 0 560 190" width="560" height="190" role="img" aria-label="The SDR pool enumerates and opens several USB dongles, then hands the engine role-tagged devices. Three dongles feed one internal/sdr Pool, which assigns roles and runs a watchdog that re-enumerates on disconnect, and exposes RoleControl, RoleVoice, and RoleWideband handles.">
  <text x="58" y="14" text-anchor="middle" fill="var(--fg-muted)" font-size="10">USB dongles</text>
  <text x="485" y="14" text-anchor="middle" fill="var(--fg-muted)" font-size="10">roles</text>
  <rect x="10" y="20" width="96" height="28" rx="6" fill="none" stroke="currentColor"/><text x="58" y="38" text-anchor="middle" fill="currentColor" font-size="10">dongle A</text>
  <rect x="10" y="64" width="96" height="28" rx="6" fill="none" stroke="currentColor"/><text x="58" y="82" text-anchor="middle" fill="currentColor" font-size="10">dongle B</text>
  <rect x="10" y="108" width="96" height="28" rx="6" fill="none" stroke="currentColor"/><text x="58" y="126" text-anchor="middle" fill="currentColor" font-size="10">dongle C</text>
  <line x1="106" y1="34" x2="140" y2="60" stroke="currentColor"/><polygon points="140,56 150,60 140,64" fill="currentColor"/>
  <line x1="106" y1="78" x2="140" y2="90" stroke="currentColor"/><polygon points="140,86 150,90 140,94" fill="currentColor"/>
  <line x1="106" y1="122" x2="140" y2="120" stroke="currentColor"/><polygon points="140,116 150,120 140,124" fill="currentColor"/>
  <rect x="150" y="40" width="150" height="100" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="225" y="70" text-anchor="middle" fill="var(--fg-muted)" font-size="8">internal/sdr/pool.go</text>
  <text x="225" y="90" text-anchor="middle" fill="var(--accent)" font-size="13">Pool</text>
  <text x="225" y="108" text-anchor="middle" fill="var(--fg-muted)" font-size="9">assign roles</text>
  <text x="225" y="122" text-anchor="middle" fill="var(--fg-muted)" font-size="9">watchdog re-enumerates</text>
  <line x1="300" y1="60" x2="410" y2="34" stroke="currentColor"/><polygon points="410,30 420,34 410,38" fill="currentColor"/>
  <line x1="300" y1="90" x2="410" y2="78" stroke="currentColor"/><polygon points="410,74 420,78 410,82" fill="currentColor"/>
  <line x1="300" y1="120" x2="410" y2="122" stroke="currentColor"/><polygon points="410,118 420,122 410,126" fill="currentColor"/>
  <rect x="420" y="20" width="130" height="28" rx="6" fill="none" stroke="currentColor"/><text x="485" y="38" text-anchor="middle" fill="currentColor" font-size="10">RoleControl</text>
  <rect x="420" y="64" width="130" height="28" rx="6" fill="none" stroke="currentColor"/><text x="485" y="82" text-anchor="middle" fill="currentColor" font-size="10">RoleVoice</text>
  <rect x="420" y="108" width="130" height="28" rx="6" fill="none" stroke="currentColor"/><text x="485" y="126" text-anchor="middle" fill="currentColor" font-size="10">RoleWideband</text>
</svg>
<figcaption>The <code>Pool</code> herds a fleet of dongles: it enumerates and opens each device through the registry, tags them by role, and a watchdog re-enumerates a bumped cable so the engine asks for "a voice device" instead of juggling serial numbers.</figcaption>
</figure>

### The IQ tap: one stream, many readers

Several subsystems want the *same* IQ at once — the decoder, the live spectrum
display, a paging decoder, signal diagnostics. But a slow spectrum panel must
never stall the decoder. `internal/sdr/iqtap` solves this with a **broker**: the
primary consumer reads the stream untouched, while secondary subscribers receive
*copies* over their own channels. If a subscriber is too slow, its copies are
**dropped and counted** — never back-pressured onto the decode path.

This is fan-out with a deliberate asymmetry: the decoder gets guaranteed
delivery; observers get best-effort.

<figure class="lab-figure">
<svg viewBox="0 0 640 200" width="640" height="200" role="img" aria-label="The internal/sdr/iqtap broker fans one IQ stream to many readers. The USB reader goroutine produces chunks into the broker; the broker delivers guaranteed copies to the decoder while sending best-effort copies to the spectrum display and diagnostics, whose frames are dropped and counted if they fall behind.">
  <rect x="8" y="80" width="120" height="44" rx="6" fill="none" stroke="currentColor"/>
  <text x="68" y="100" text-anchor="middle" fill="currentColor" font-size="10">USB reader</text>
  <text x="68" y="114" text-anchor="middle" fill="var(--fg-muted)" font-size="8">StreamIQ chunk</text>
  <line x1="128" y1="102" x2="158" y2="102" stroke="currentColor"/><polygon points="158,98 168,102 158,106" fill="currentColor"/>
  <rect x="170" y="76" width="110" height="52" rx="6" fill="none" stroke="currentColor"/>
  <text x="225" y="98" text-anchor="middle" fill="currentColor" font-size="10">iqtap broker</text>
  <text x="225" y="112" text-anchor="middle" fill="var(--fg-muted)" font-size="8">fan-out / copy</text>
  <line x1="280" y1="98" x2="322" y2="34" stroke="currentColor"/><polygon points="318,30 328,32 322,41" fill="currentColor"/>
  <line x1="280" y1="104" x2="322" y2="106" stroke="currentColor"/><polygon points="322,101 330,105 322,109" fill="currentColor"/>
  <line x1="280" y1="112" x2="322" y2="168" stroke="currentColor"/><polygon points="318,161 328,170 316,170" fill="currentColor"/>
  <rect x="330" y="16" width="200" height="34" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="430" y="33" text-anchor="middle" fill="var(--accent)" font-size="10">decoder</text>
  <text x="430" y="45" text-anchor="middle" fill="var(--fg-muted)" font-size="8">guaranteed delivery</text>
  <rect x="330" y="88" width="200" height="34" rx="6" fill="none" stroke="currentColor"/>
  <text x="430" y="105" text-anchor="middle" fill="currentColor" font-size="10">spectrum display</text>
  <text x="430" y="117" text-anchor="middle" fill="var(--fg-muted)" font-size="8">best-effort</text>
  <rect x="330" y="150" width="200" height="34" rx="6" fill="none" stroke="currentColor"/>
  <text x="430" y="167" text-anchor="middle" fill="currentColor" font-size="10">diagnostics</text>
  <text x="430" y="179" text-anchor="middle" fill="var(--fg-muted)" font-size="8">best-effort</text>
  <text x="590" y="140" text-anchor="middle" fill="var(--fg-muted)" font-size="9">slow →</text>
  <text x="590" y="152" text-anchor="middle" fill="var(--fg-muted)" font-size="9">dropped</text>
  <text x="590" y="164" text-anchor="middle" fill="var(--fg-muted)" font-size="9">&amp; counted</text>
</svg>
<figcaption>The tap broker breaks the back-pressure rule on purpose: the decoder is guaranteed every chunk, while a slow spectrum or diagnostics subscriber has its copies dropped and counted so it can never stall the decode path.</figcaption>
</figure>

## The design principle: pipes-and-filters with bounded channels

The guiding rule is that **stages communicate only through channels, and those
channels are bounded**. No stage reaches into another's state; no queue is
allowed to grow without limit.

### How that principle shaped the Go code

- **One goroutine owns one piece of state.** A device's USB reader is the sole
  writer of its IQ channel, so there are no locks on the hot path — ownership,
  not mutexes, provides the safety.
- **`context.Context` is the off-switch.** Every streaming goroutine takes a
  context; shutdown is a single cancel that propagates down the pipeline.
- **Back-pressure is a feature, not a bug.** Buffered channels bound memory and
  signal overload upstream. The only place the rule is *intentionally* broken is
  the tap broker, where dropping observer frames is the correct trade-off — and
  the drops are measured, not hidden.
- **Mutexes guard structure, not flow.** The pool uses a `sync.RWMutex` for its
  device map (read-heavy: lots of "give me the voice device", rare mutations),
  while the sample flow stays lock-free through channels.

## Where this goes next

The concurrency model here is the backbone for everything downstream — DSP
stages ([Part 4]({{ '/blog/deep-dives/sdr-internals-04-dsp-foundations-filters-nco-agc/' | relative_url }})),
the event bus ([Part 11]({{ '/blog/deep-dives/sdr-internals-11-trunking-engine-event-bus/' | relative_url }})),
and the broadcast uploaders ([Part 13]({{ '/blog/deep-dives/sdr-internals-13-recording-composition-streaming/' | relative_url }}))
are all goroutines hanging off channels. A future deep dive will benchmark
buffer sizing and drop rates under load.

## FAQ

**Why channels of `[]complex64` instead of one sample at a time?**
Chunking amortizes per-message overhead. Sending 2.4 million individual samples
per second over a channel would be dominated by scheduling cost; sending ~400
chunks of several thousand samples is efficient and still low-latency.

**What happens if a downstream stage is too slow?**
On the main decode path, the buffered channel fills and the producer blocks —
back-pressure. On the observer path (spectrum, diagnostics), the tap broker
drops frames and increments a counter, protecting the decoder.

**How are multiple dongles kept in sync?**
They aren't tightly synchronized; each runs its own stream and goroutine. The
pool coordinates *roles* and *tuning*, and the trunking engine correlates their
output through events rather than shared clocks.

## Series navigation

**Part 3 of 14** · ←
[Part 2]({{ '/blog/deep-dives/sdr-internals-02-sdr-device-driver-registry/' | relative_url }})
· Next →
[Part 4: DSP foundations — filters, NCO & AGC]({{ '/blog/deep-dives/sdr-internals-04-dsp-foundations-filters-nco-agc/' | relative_url }})
