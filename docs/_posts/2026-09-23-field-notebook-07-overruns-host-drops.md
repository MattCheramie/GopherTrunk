---
title: "The Field Notebook, Part 7: Overruns & host_drops — Reading a Downstream Signal"
description: "How to read GopherTrunk's overrun family — the soapyremote SDR overruns WARN with device_overflows and host_drops, the ccdecoder decode can't keep up WARN, the same-carrier voice tap dropped_chunks line and the runtime heartbeat — why each one points downstream at the decoder rather than at the radio, and the CPU story behind the 10 Sep dual-TETRA rig."
category: tutorials
keywords: soapyremote sdr overruns, host_drops meaning, decode can't keep up with real time, gophertrunk runtime heartbeat, sdr overrun vs decode overrun, dropped_chunks voice tap, sdr input_sample_rate decimation, gc pressure sdr decoder, gophertrunk overruns host drops
tags: [field-notebook, logs, overruns, soapyremote, performance, diagnostics, tutorial]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "The Field Notebook"
series_part: 7
---

*Part 7 of **The Field Notebook**, a 14-part operator's tutorial that reads
GopherTrunk's debug.log one line family at a time — what each field measures,
what a healthy rig prints, which number is a noise meter and which one means
traffic, and which deep dive to open when a line goes wrong.
[Part 6]({{ '/blog/tutorials/field-notebook-06-dmo-status-line/' | relative_url }})
read the DMO status line, whose counters assume the decoder keeps up with its
tap. This part reads what the log prints when it does not: the **overrun
family** — four lines at four places in the sample path — and the runtime
heartbeat that usually names the culprit first.*

> **TL;DR:** `soapyremote: SDR overruns — the host can't keep up …`
> (`internal/sdr/soapyremote/driver.go`, `overrunThrottle`, one summary per
> 5 s) folds two counts: `device_overflows` (the radio dropped samples) and
> `host_drops` (`sendOrDrop` shed the **oldest** queued chunk because the DSP
> consumer stopped draining a ~400 ms channel). The driver has not changed
> since import, so the cause is always what got slower *downstream*. Two
> lines localise that: `ccdecoder: decode can't keep up with real time` (the
> 0.5 s decode-queue budget in `forwardIQ`, counted as
> `decode_overruns_total`) and the per-call `same-carrier voice tap dropped
> IQ … dropped_chunks=N` WARN. `runtime: heartbeat` (60 s) prints
> `goroutines`, `heap_alloc_mb`, `num_gc`: on the 10 Sep dual-TETRA rig at
> 200 kS/s, ~19 collections/s on a 9 MB heap was the tell — `DecodeAACH`
> re-encoding 16 384 RM(30,14) codewords per slot, fixed by a once-built
> codebook (pump 0.44x → 0.12x real time). `sdr.input_sample_rate` is the
> load lever when the rig, not the code, is the limit.

**Key takeaways**

- **An overrun WARN is a downstream signal.** `sendOrDrop` sheds only when
  the consumer stops draining; the reader never blocks. Look at what got
  slower on the decode side first.
- **`device_overflows` and `host_drops` are two different places.** The
  radio's own flow control giving up, versus GopherTrunk's cushion
  overflowing while the device was fine. Both glitch every channel at once.
- **The heartbeat is the profiler you already have.** `num_gc` climbing
  ~1100 per minute on a small heap is churn in a hot loop; climbing
  `goroutines` or `heap_alloc_mb` is a leak. Read its derivative.
- **Lower the rate before you buy CPU.** The "capture is oversampled" WARN
  and `sdr.input_sample_rate` cut load without losing coverage; the decoder
  is rate-invariant.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| Driver overrun WARN | 5 s summary of `device_overflows` + `host_drops` | `internal/sdr/soapyremote/driver.go` (`overrunThrottle`, `overrunWarnInterval`) |
| The cushion | ~400 ms stream channel, oldest chunk shed when full | `driver.go` (`streamBufferLatency`, `sendOrDrop`) |
| Decode-queue WARN | 0.5 s wall-clock budget; drops at the decode queue | `internal/scanner/ccdecoder/decoder.go` (`forwardIQ`, `decodeQueueSeconds`) |
| Voice-tap starvation | one WARN per call end with `dropped_chunks` | `ccdecoder/voicetap.go`, `widebandt2/channeliq.go` |
| Runtime heartbeat | goroutines, heap, GC count every 60 s | `cmd/gophertrunk/runtime_health.go` (`runHeartbeat`) |
| Two metrics, two places | `sdr_iq_underruns_total` vs `decode_overruns_total` | `internal/metrics/prom.go`; [Running It For Real 4]({{ '/blog/deep-dives/running-it-for-real-04-metrics-that-matter/' | relative_url }}) |
| The load lever | integer pre-decimation at the Device boundary | `sdr.input_sample_rate` → `internal/sdr/decimate` |

## In this post

- **What these lines are telling you** — four drop points, one sample path.
- **What healthy looks like** — a heartbeat and a connected line, nothing else.
- **Field by field** — the WARN fields and the heartbeat fields.
- **When it doesn't look like that** — the 10 Sep dual-TETRA rig and the
  20 Aug DMO voice chain.
- **Symptom → cause → read** — and how the lines shape operator practice.

## What these lines are telling you

Samples flow from the radio through bounded queues, and every queue has a
WARN for the moment it fills. Knowing which queue each line belongs to
localises the slowdown.

```go
// internal/sdr/soapyremote/driver.go (shape) — the one summary WARN
t.log.Warn("soapyremote: SDR overruns — the host can't keep up with the configured sample rate, so samples are being dropped and decoded audio will glitch. Lower sdr.sample_rate or reduce the channel/tap count.",
    "addr", t.addr,
    "device_overflows", t.devOverflows, // SOAPY_SDR_OVERFLOW datagrams: the radio dropped
    "host_drops", t.hostDrops)           // sendOrDrop shed a chunk: the consumer fell behind
```

**Stage one is the driver's stream channel.** The SoapyRemote read loop
never blocks on the DSP consumer — a blocked send stalls socket reads and
flow-control ACKs, and the *radio* then overflows in a way that shreds every
channel's framing. So the channel holds ~400 ms of IQ
(`streamBufferLatency`, clamped to 64–2048 chunks), and when it is full
`sendOrDrop` discards the **oldest** queued chunk and counts a `host_drop`.
A `device_overflow` is a `SOAPY_SDR_OVERFLOW` datagram: the device dropping
because the host stopped draining the socket. Both used to be invisible —
overflow at DEBUG, back-pressure stalling the reader.

**Stage two is the decode queue.** `Decoder.forwardIQ` keeps a wall-clock
sample budget (`decodeQueueSeconds` = 0.5 s) and drops *before copying* when
a chunk would push the backlog past it, logging at most once a second with
`dropped_since_last` and counting `decode_overruns_total` — kept distinct
from the driver's `sdr_iq_underruns_total` so "CPU can't keep up" is
attributable instead of looking like RF. Local drivers surface the same
event as `sdr: dropping live IQ chunks; consumer can't keep up`.

**Stage three is the per-call voice tap.** `voiceFanout` broadcasts a
same-carrier channel's IQ to each voice consumer; a lagging consumer's chunks
are dropped and the total reported once, at unsubscribe, as
`dropped_chunks` — the line that names one slow consumer.

**Stage zero is the heartbeat.** Not an overrun line, but where the cause
shows first: churn reads as `num_gc` racing on a small heap, a leak as
`goroutines` or `heap_alloc_mb` climbing, a hang as a heartbeat that stops.

<figure class="lab-figure">
<svg viewBox="0 0 680 250" width="680" height="250" role="img" aria-label="The sample path from a remote SDR to the decoder and voice chains, drawn as four boxes with a bounded queue between each. Each queue is labelled with the log line that fires when it fills: the driver's four-hundred-millisecond stream channel with device overflows and host drops, the decoder's half-second queue with decode can't keep up, and the voice fan-out with dropped chunks. A runtime heartbeat box sits above the chain reporting goroutines, heap and GC count.">
  <rect x="10" y="100" width="90" height="40" rx="5" fill="none" stroke="currentColor"/>
  <text x="55" y="118" text-anchor="middle" fill="currentColor" font-size="10">SoapySDR</text>
  <text x="55" y="131" text-anchor="middle" fill="var(--fg-muted)" font-size="8">device_overflows</text>
  <line x1="100" y1="120" x2="150" y2="120" stroke="currentColor"/>
  <rect x="150" y="100" width="120" height="40" rx="5" fill="none" stroke="var(--accent)"/>
  <text x="210" y="116" text-anchor="middle" fill="var(--accent)" font-size="10">stream channel</text>
  <text x="210" y="130" text-anchor="middle" fill="var(--fg-muted)" font-size="8">~400 ms · sendOrDrop → host_drops</text>
  <line x1="270" y1="120" x2="320" y2="120" stroke="currentColor"/>
  <rect x="320" y="100" width="120" height="40" rx="5" fill="none" stroke="var(--accent)"/>
  <text x="380" y="116" text-anchor="middle" fill="var(--accent)" font-size="10">decode queue</text>
  <text x="380" y="130" text-anchor="middle" fill="var(--fg-muted)" font-size="8">0.5 s budget · can't keep up</text>
  <line x1="440" y1="120" x2="490" y2="120" stroke="currentColor"/>
  <rect x="490" y="100" width="120" height="40" rx="5" fill="none" stroke="var(--accent)"/>
  <text x="550" y="116" text-anchor="middle" fill="var(--accent)" font-size="10">voice fan-out</text>
  <text x="550" y="130" text-anchor="middle" fill="var(--fg-muted)" font-size="8">per call · dropped_chunks</text>
  <line x1="610" y1="120" x2="660" y2="120" stroke="currentColor"/>
  <text x="640" y="150" text-anchor="end" fill="currentColor" font-size="9">vocoder → recorder</text>
  <rect x="200" y="20" width="280" height="40" rx="5" fill="none" stroke="currentColor"/>
  <text x="340" y="36" text-anchor="middle" fill="currentColor" font-size="10">runtime: heartbeat (60 s)</text>
  <text x="340" y="50" text-anchor="middle" fill="var(--fg-muted)" font-size="8">goroutines · heap_alloc_mb · num_gc</text>
  <line x1="340" y1="60" x2="340" y2="100" stroke="var(--fg-muted)" stroke-dasharray="3 3"/>
  <text x="340" y="206" text-anchor="middle" fill="currentColor" font-size="9">each queue logs on its own line — the first to fire is nearest the slow stage</text>
</svg>
<figcaption>Four bounded queues, four lines. The driver cushion, the decode budget and the voice fan-out each drop at a known place; the heartbeat above them shows the allocation or goroutine curve that explains why.</figcaption>
</figure>

## What healthy looks like

A healthy rig prints two lines from this family, neither a WARN: the
driver's startup announcement and the minute heartbeat:

```
INF soapyremote: connected addr=192.168.1.60:55132 format=CS16 proto=tcp diversity=mrc
INF runtime: heartbeat uptime=1h0m0s goroutines=148 heap_alloc_mb=11 heap_sys_mb=31 sys_mb=92 next_gc_mb=22 num_gc=1410
```

(Values illustrative — the fields are the daemon's.) Read the **difference
between consecutive heartbeats**, not the absolutes: `goroutines` flat,
`heap_alloc_mb` oscillating in a band, `num_gc` advancing a hundred or so
per minute. No `SDR overruns` WARN in an hour; no `decode can't keep up`;
call-end lines without a `dropped_chunks` companion — the 29 Aug X310 log
ran 18.5 minutes with 0 overruns.

## Field by field

| Field | Measures | Healthy | Worry when |
|---|---|---|---|
| `device_overflows` | `SOAPY_SDR_OVERFLOW` datagrams since the last summary | 0 | > 0 — the socket was not drained: stalled reader or saturated NIC |
| `host_drops` | chunks `sendOrDrop` shed locally | 0 | > 0 — the consumer stopped draining a 400 ms cushion |
| `dropped_since_last` | decode-queue drops in the last second | absent | present — this Decoder is sustainedly behind real time |
| `dropped_chunks` | voice-tap chunks a lagging consumer lost, per call | absent | present — that call's chain starved itself |
| `goroutines` | live goroutines | flat | climbing every heartbeat — a leak |
| `heap_alloc_mb` / `next_gc_mb` | live heap / next collection trigger | oscillating in a band | ratcheting up — a leak |
| `num_gc` | lifetime collections | ~100/min | ~1000+/min on a small heap — allocation in a hot loop |

The pairing that matters is `device_overflows` against `host_drops`.
`host_drops>0, device_overflows=0` says the device was fine and
GopherTrunk's own cushion overflowed — pure consumer slowness. The reverse
says the socket was not being read: a stalled reader or a network bottleneck
([Cookbook 8]({{ '/blog/tutorials/operator-cookbook-08-remote-radios/' | relative_url }})
budgets two channels of CS16). Before the cushion was sized in stream time, a
fixed depth of 8 chunks shed ~2% on ordinary scheduling jitter with
`device_overflows=0` throughout and shredded P25 decode.

## When it doesn't look like that

**The 10 Sep dual-TETRA rig.** Two TETRA DDC channels on a wideband X310 at
200 kS/s — a tiny rate — and the operator reported host overruns:

```
WRN soapyremote: SDR overruns — the host can't keep up with the configured sample rate, so samples are being dropped and decoded audio will glitch. Lower sdr.sample_rate or reduce the channel/tap count. addr=192.168.1.60:55132 device_overflows=0 host_drops=41
INF runtime: heartbeat uptime=12m0s goroutines=151 heap_alloc_mb=9 heap_sys_mb=27 sys_mb=84 next_gc_mb=18 num_gc=13660
```

`device_overflows=0`: the radio was fine. No `decode can't keep up` either —
the wideband pump has no decode queue of that kind, so the driver's cushion
was the first to fill. The heartbeat is the instrument: `num_gc` advancing
~1140 per minute (≈19/s) on a 9 MB heap is allocation churn, not load.
`TestEngineDualTETRA200kThroughput` (`GT_WB_BENCH_SECONDS=30
GT_WB_BENCH_PROFILE=…`) measured the pump at **0.44x real time** on one
2.1 GHz core with 65% in `DecodeAACH`: `DecodeRM3014Tetra` *re-encoded* all
16 384 RM(30,14) codewords, allocating, on every call — once per downlink
slot (~70/s per carrier) plus once per traffic burst in the voice demux. A
once-built codebook (`rm3014Table`, popcount + partial-sum tables) made it
17–36 µs and allocation-free, bit-identical to the brute force; a mirrored
FIR history window then ran the filter ~1.8x faster with the same float32
summation order. Pump: **0.44x → 0.12x**. An "overruns at low rate" report
with no `keep up` WARN still means profile the pump — `host_drops` is the
consumer, and here the consumer was a block decoder.

**The 20 Aug DMO voice chain.** A different queue, a different signature:

```
WRN ccdecoder: same-carrier voice tap dropped IQ to a lagging voice consumer — the followed call's decode was starved (expect short/gappy recordings); reduce CPU load or lower sdr.sample_rate (issue #402) dropped_chunks=5904
```

No driver overrun — one consumer, starved by itself. The chain re-ran the
64-colour `RecoverDMColourCode` brute force over its whole growing buffer on
**every** burst, ≈450k Viterbi decodes per call (64·Σ(20..120)); it was
capped at six passes, then replaced by the exact seed solver
([Part 6]({{ '/blog/tutorials/field-notebook-06-dmo-status-line/' | relative_url }})).
`dropped_chunks` on one protocol's calls only means the slow code is inside
that protocol's voice chain.

**The lever when the rig is the limit.** At startup:

```
WRN widebandt2: capture is oversampled for the channel plan — the carriers span less than half the captured band, so a lower sdr.sample_rate would cut DSP + network load (and overrun pressure) for no loss of coverage serial=usrp-attic sample_rate_hz=6250000 channel_span_hz=1375000 min_sample_rate_hz=1718750
```

And when the hardware's native rate cannot be lowered (an Airspy at
10 MS/s), `sdr.input_sample_rate` runs the device native and
integer-decimates to `sample_rate` at the Device boundary (`Pool.WrapDevice`)
before anything downstream sees IQ. A load lever, not an RF fix: it does not
recover front-end degradation baked into a high-rate capture, and the
decoders were already
[rate-invariant]({{ '/reference/sample-rate/' | relative_url }}).

| Symptom | Likely cause | Fix / read |
|---|---|---|
| `host_drops>0`, `device_overflows=0`, `num_gc` racing | allocation churn in a hot decode loop | profile the pump; [garbage collection]({{ '/reference/garbage-collection/' | relative_url }}), [overruns & underruns]({{ '/reference/overruns-underruns/' | relative_url }}) |
| `host_drops>0` with `decode can't keep up` | the decode goroutine is the bottleneck | fewer channels, lower `sdr.sample_rate`, `input_sample_rate`; [SDR Internals 3]({{ '/blog/deep-dives/sdr-internals-03-sdr-pool-streaming-concurrency/' | relative_url }}) |
| `device_overflows>0` | the socket is not being drained — network or a stalled reader | wired network; [Issue Tracker 13]({{ '/blog/solution-postmortem/from-the-issue-tracker-13-soapyremote-handshake/' | relative_url }}) |
| `dropped_chunks` on one protocol's calls | that voice chain is starving itself | the composer line family, [Part 11]({{ '/blog/tutorials/field-notebook-11-voice-chain-lines/' | relative_url }}) |
| `goroutines` or `heap_alloc_mb` ratcheting | a leak | `diagnostics.memory_limit_mb`, [Running It For Real 5]({{ '/blog/deep-dives/running-it-for-real-05-structured-logs/' | relative_url }}) |
| "capture is oversampled" at startup | plan uses < half the band | lower `sdr.sample_rate` to the line's `min_sample_rate_hz`; [Running It For Real 4]({{ '/blog/deep-dives/running-it-for-real-04-metrics-that-matter/' | relative_url }}) for the metric tiles |

### How this line shapes operator practice

- **Read the overrun WARN as "what got slower?", never "is the network
  bad?".** Every field report of overruns so far resolved to a decode-side
  cost.
- **Pair it with the heartbeat's derivative.** Copy two consecutive
  heartbeats into a report — `num_gc` per minute and the `goroutines` delta
  separate churn from leak.
- **Check the companion WARN.** `decode can't keep up` confirms CPU on a
  single-channel path; its absence on a wideband path does not clear the
  pump.
- **Cut the rate first.** The oversampled WARN and `input_sample_rate` are
  free; hardware is not.

## Where this goes next

Overruns are one health line whose absence is the good news. The next one
is the opposite: a line that prints every 30 s and must be read to know
whether a second antenna is helping at all.
[Part 8]({{ '/blog/tutorials/field-notebook-08-mrc-health-line/' | relative_url }})
reads the MRC diversity health line — `coherence`, `branch_phase_deg`,
`updates`/`holds`, `lock_gate` — and the WARNs whose text history is a
lesson in itself.

## FAQ

**What does `host_drops` mean in the GopherTrunk soapyremote WARN?**
Chunks the driver discarded locally because its ~400 ms stream channel was
full — the DSP consumer stopped draining it. The read loop never blocks, so
a host drop is by construction the consumer's fault: something downstream
got slower. `device_overflows` is the radio's own drop count.

**Is an SDR overrun an RF problem?**
No. It is sample loss between the radio and the decoder — a throughput
limit — that looks like RF only because the discontinuity breaks framing on
every channel at once. The WARN names the remedies: lower `sdr.sample_rate`
or reduce the channel/tap count.

**Why did a 200 kS/s rig overrun when a laptop decodes megasamples?**
Because the cost was not sample rate but a block decoder allocating in a hot
loop: `DecodeAACH` re-encoded 16 384 codewords per downlink slot, ~65% of the
pump's CPU and ~19 GCs a second. A once-built codebook took the pump from
0.44x to 0.12x real time.

**What is `decode_overruns_total` versus `sdr_iq_underruns_total`?**
Two Prometheus counters for two queues. `sdr_iq_underruns_total` counts the
driver's delivery channel dropping; `decode_overruns_total` counts the
decode goroutine dropping at its own 0.5 s queue. Decode overruns climbing
with underruns flat says "the machine can't sustain this decode".

**How does `sdr.input_sample_rate` reduce load?**
It runs the hardware at its native rate and integer-decimates to
`sdr.sample_rate` inside the Device wrapper, so every downstream stage —
DDC bank, demods, recording taps, spectrum — sees the lower rate. The ratio
must be an exact integer; it cuts CPU and recording size, not front-end
noise.

## Series navigation

**Part 7 of 14** · ←
[Part 6: The DMO Status Line — Noise Meters & Traffic Counters]({{ '/blog/tutorials/field-notebook-06-dmo-status-line/' | relative_url }})
· Next →
[Part 8: The MRC Health Line]({{ '/blog/tutorials/field-notebook-08-mrc-health-line/' | relative_url }})
