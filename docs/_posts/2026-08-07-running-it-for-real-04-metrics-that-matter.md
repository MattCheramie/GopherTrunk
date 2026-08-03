---
title: "Running It For Real, Part 4: Metrics That Matter — Prometheus & SDR Tiles"
description: How GopherTrunk's Prometheus registry is built — bus-driven counters versus scrape-time snapshot collectors, the SDR tiles worth alerting on (IQ power, clip ratio, gain, lock state), why calls_started beats calls_total, and the opt-in FEC histograms.
category: deep-dives
keywords: prometheus metrics, sdr monitoring, iq power dbfs, clip ratio overload, control channel locked gauge, scrape-time collector, calls_started_total, detailed fec histogram, gophertrunk running it for real
tags: [running-it-for-real, metrics, observability, ops, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "Running It For Real"
series_part: 4
---

*Part 4 of **Running It For Real**, the series taking one GopherTrunk daemon from
a laptop demo to a hardened 24/7 service. Parts 2 and 3 hardened the edge; this
post is about knowing whether the thing behind that edge is actually *healthy*.
A demo is healthy when you can see a call. A service is healthy when a dashboard
says so at 3am — and, more importantly, pages you when a dongle goes deaf hours
before you'd otherwise notice the silence.*

> **TL;DR:** GopherTrunk owns a private `prometheus.Registry` and exposes it on
> `GET /metrics`. Its metrics come from two sources with different truth
> guarantees: **bus-driven counters/gauges** that a goroutine increments as it
> observes engine events (calls, grants, CC lock transitions), and a
> **scrape-time snapshot collector** that reads `sdr.Pool.Snapshot()` on every
> request so per-device tuning values (gain, AGC, PPM, bias-tee) can never go
> stale. The SDR tiles are the ones worth alerting on: `sdr_iq_power_dbfs` and
> `sdr_iq_clip_ratio` catch a front-end going deaf or clipping, and
> `control_channel_locked` plus `control_channel_transitions_total` catch a system
> that's churning. Two small design calls carry a lot of weight —
> `calls_started_total` over `calls_total`, and pull-mode for anything that must
> reflect live state.

**Key takeaways**

- **Two metric sources, two truth models.** Bus-driven counters are cheap and
  monotonic; the scrape-time collector re-reads the pool on every `/metrics` hit
  so gauges can't lie about the current tuner state.
- **`calls_started_total` is the reliable rate signal.** `CallEnd` can be missed
  if the daemon dies mid-call, so the *started* counter — incremented at grant
  time — is the honest input to a calls-per-minute alert.
- **The SDR tiles are the ones that page you.** `iq_power_dbfs` (idle ≈ −45,
  healthy ≈ −25, clipping > −3) and `iq_clip_ratio` catch the deaf/overloaded
  front-end that a lock gauge alone can't explain; `control_channel_locked` and
  `_transitions_total` catch churn.
- **Cardinality and cost are opt-in.** The detailed per-protocol FEC histograms
  (`tetra_viterbi_corrections`) are off unless `metrics.detailed_fec: true`, so a
  stock deployment doesn't carry a histogram family it can't interpret.

## Cheat sheet

| Metric | Type | What it tells you |
|---|---|---|
| `gophertrunk_calls_started_total{system,protocol,encrypted}` | counter | honest call-rate signal (survives mid-call death) |
| `gophertrunk_calls_active{system,protocol}` | gauge | concurrent calls; `sum()` for daemon-wide |
| `gophertrunk_control_channel_locked{system}` | gauge | `1` while CC is locked, `0` otherwise |
| `gophertrunk_control_channel_transitions_total{system,event}` | counter | lock/lost churn under poor SNR |
| `gophertrunk_sdr_iq_power_dbfs{system}` | gauge | front-end level: idle ≈ −45, healthy ≈ −25, clip > −3 |
| `gophertrunk_sdr_iq_clip_ratio{system}` | gauge | authoritative overload signal (RMS power averages it away) |
| `gophertrunk_sdr_gain_db{driver,serial,role}` | gauge | configured gain; `NaN` under AGC |
| `gophertrunk_decode_errors_total{protocol,stage}` | counter | where decode fails, by stage |

Sources: `internal/metrics/prom.go` (registry + bus loop), `internal/metrics/sdr_snapshot.go` (scrape-time collector).

## In this post

- **One registry, two sources** — bus-driven versus scrape-time, and why both.
- **The started-vs-total lesson** — the counter that survives a crash.
- **The SDR tiles worth an alert** — power, clipping, lock, churn.
- **Opt-in cost** — the detailed FEC histograms and why they're off by default.

## One registry, two sources

The `Metrics` type owns a private `prometheus.Registry` (private, so it doesn't
collide with anything else in-process) and populates it two very different ways.
The distinction matters because they have different failure modes, and getting a
metric from the wrong source is how dashboards start lying.

<figure class="lab-figure">
<svg viewBox="0 0 660 176" width="660" height="176" role="img" aria-label="The event bus feeds a metrics goroutine that increments push-mode counters and gauges; separately a scrape-time snapshot collector reads the SDR pool on each request; both register into one private Prometheus registry served at slash metrics">
  <rect x="8" y="26" width="120" height="42" rx="6" fill="none" stroke="currentColor"/>
  <text x="68" y="44" text-anchor="middle" fill="currentColor" font-size="11">event bus</text>
  <text x="68" y="59" text-anchor="middle" fill="var(--fg-muted)" font-size="9">engine events</text>
  <line x1="128" y1="47" x2="160" y2="47" stroke="currentColor"/><polygon points="160,43 170,47 160,51" fill="currentColor"/>
  <rect x="170" y="26" width="150" height="42" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="245" y="44" text-anchor="middle" fill="var(--accent)" font-size="11">observeEvent</text>
  <text x="245" y="59" text-anchor="middle" fill="var(--fg-muted)" font-size="9">push counters/gauges</text>
  <rect x="8" y="108" width="120" height="42" rx="6" fill="none" stroke="currentColor"/>
  <text x="68" y="126" text-anchor="middle" fill="currentColor" font-size="11">sdr.Pool</text>
  <text x="68" y="141" text-anchor="middle" fill="var(--fg-muted)" font-size="9">Snapshot()</text>
  <line x1="128" y1="129" x2="160" y2="129" stroke="currentColor"/><polygon points="160,125 170,129 160,133" fill="currentColor"/>
  <rect x="170" y="108" width="150" height="42" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="245" y="126" text-anchor="middle" fill="var(--accent)" font-size="11">snapshot collector</text>
  <text x="245" y="141" text-anchor="middle" fill="var(--fg-muted)" font-size="9">reads at scrape time</text>
  <line x1="320" y1="47" x2="392" y2="80" stroke="currentColor"/><polygon points="390,76 400,84 388,84" fill="currentColor"/>
  <line x1="320" y1="129" x2="392" y2="96" stroke="currentColor"/><polygon points="388,92 400,92 390,100" fill="currentColor"/>
  <rect x="400" y="66" width="130" height="44" rx="6" fill="none" stroke="currentColor"/>
  <text x="465" y="84" text-anchor="middle" fill="currentColor" font-size="11">Registry</text>
  <text x="465" y="99" text-anchor="middle" fill="var(--fg-muted)" font-size="9">private, one process</text>
  <line x1="530" y1="88" x2="562" y2="88" stroke="currentColor"/><polygon points="562,84 572,88 562,92" fill="currentColor"/>
  <rect x="572" y="66" width="80" height="44" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="612" y="84" text-anchor="middle" fill="var(--accent)" font-size="11">/metrics</text>
  <text x="612" y="99" text-anchor="middle" fill="var(--fg-muted)" font-size="9">scrape</text>
  <text x="330" y="170" text-anchor="middle" fill="var(--fg-muted)" font-size="10">push mode for event counts (monotonic); pull mode for live tuner state (can't go stale)</text>
</svg>
<figcaption>Two sources into one registry. Push-mode counters accumulate as events flow; the pull-mode collector re-reads the pool on every scrape so gauges reflect live hardware, not a cached snapshot.</figcaption>
</figure>

**Push mode** is a goroutine subscribed to the event bus. It runs the same
`Run(ctx)` loop every other subsystem uses, and for each event it calls
`observeEvent`, which increments the right counter — a `KindCallStart` bumps
`calls_active` and `calls_started_total`, a `KindGrant` bumps `grants_total`, a
`KindCallEnd` decrements active and bumps `calls_total{reason}`. These are cheap,
monotonic, and exactly right for *rates* and *totals*.

**Pull mode** is the SDR snapshot collector, and it exists precisely because a
tuner's gain or AGC state is *current state*, not an event stream. If it were
pushed on a change event, a missed event would leave the gauge wrong until the
next change — which might be never. Instead it implements `prometheus.Collector`
and reads the pool on demand:

```go
// internal/metrics/sdr_snapshot.go (shape) — Collect runs on every /metrics scrape
func (c *sdrSnapshotCollector) Collect(ch chan<- prometheus.Metric) {
    for _, s := range c.pool.Snapshot() {
        if !s.Attached {
            continue
        }
        gain := math.NaN()
        if !s.GainAuto {
            gain = float64(s.GainTenthDB) / 10.0 // NaN under AGC; pair with gain_auto to filter
        }
        ch <- prometheus.MustNewConstMetric(c.gainDB,   prometheus.GaugeValue, gain, s.Driver, s.Serial, s.Role)
        ch <- prometheus.MustNewConstMetric(c.gainAuto, prometheus.GaugeValue, boolToFloat(s.GainAuto), s.Driver, s.Serial, s.Role)
        ch <- prometheus.MustNewConstMetric(c.ppm,      prometheus.GaugeValue, float64(s.PPM), s.Driver, s.Serial, s.Role)
        ch <- prometheus.MustNewConstMetric(c.biasTee,  prometheus.GaugeValue, boolToFloat(s.BiasTee), s.Driver, s.Serial, s.Role)
    }
}
```

The `gain_db` = `NaN`-under-AGC detail is a nice touch: AGC has no fixed gain to
report, so `NaN` is honest, and pairing it with the `gain_auto` gauge lets a query
filter AGC devices out rather than plotting a meaningless number. The collector
takes a `Snapshotter` interface (just `Snapshot() []sdr.SDRStatus`), so tests pass
a fake and a daemon with no pool passes nil — no dependency on the full `sdr.Pool`.

## The started-vs-total lesson

Here's a bug that only bites a *service*, never a demo: the daemon dies mid-call.
On a laptop you'd never notice — you re-run. As a 24/7 feed, the daemon that
crashes during an active call never emits that call's `CallEnd`, so
`calls_total` under-counts by exactly the calls that were in flight when it died.
Build your "calls per minute" alert on `calls_total` and it *dips* precisely when
things are going wrong, which is backwards.

That's why `calls_started_total` exists as a separate counter, incremented at
grant time — before anything can go wrong with the call:

```go
// internal/metrics/prom.go (shape) — observeEvent
case events.KindCallStart:
    if cs, ok := ev.Payload.(trunking.CallStart); ok {
        sys, proto, enc := callLabels(cs.Grant)
        m.activeCalls.WithLabelValues(sys, proto).Inc()
        m.callsStarted.WithLabelValues(sys, proto, enc).Inc() // survives mid-call death
    }
```

The help text says it plainly: *more reliable as a rate signal than calls_total
because CallEnd can be missed when the daemon dies mid-call.* Use `calls_total`
(with its `reason` label) for outcome analysis — how many calls ended normally
versus were pre-empted — and use `calls_started_total` for the rate you alert on.
It's a two-counter design that costs almost nothing and makes the difference
between an alert that fires on trouble and one that goes quiet on it.

### How that principle shaped the Go code

- **The registry is private.** `prometheus.NewRegistry()` rather than the global
  default means no accidental collision with a library that also registers
  metrics, and the handler serves exactly this daemon's series.
- **Bus counters are label-parsimonious.** Labels are `system`/`protocol`/
  `encrypted`/`reason` — bounded, low-cardinality dimensions — never a radio ID or
  a frequency, which would explode the series count.
- **DMR gets a per-slot counter, gated.** `dmr_voice_calls_total{timeslot}` only
  increments for grants that actually carry a DMR timeslot, so a TETRA call's 1–4
  slot isn't miscounted under a DMR-named metric.
- **CC frequency series is deleted on loss.** `control_channel_frequency_hz` is
  removed when the lock drops rather than left showing a stale frequency — a gauge
  that lingers is a gauge that lies.

## The SDR tiles worth an alert

Most of the counters are for dashboards you glance at. A handful of gauges are for
*alerts* — they catch a class of failure that is otherwise silent, because the
daemon keeps running and the logs keep flowing while no calls come in. The `iq`
tiles are the front-of-house here.

`sdr_iq_power_dbfs` is the mean IQ power on the control SDR, and its help text
carries the thresholds: **idle noise ≈ −45 dBFS, healthy signal ≈ −25 dBFS, and
above −3 dBFS means clipping.** A tile pinned near −45 on a system that should be
locked is a deaf front-end — no antenna, gain at zero, a browned-out dongle. But
there's a trap the code is explicit about: RMS power *averages away* peak
clipping, so a front-end that's slamming the ADC rail on peaks can still read a
comfortable-looking RMS. That's why `sdr_iq_clip_ratio` (the fraction of samples
pinned to the rail) is the *authoritative* overload signal, and the deep-dive
notes throughout the codebase warn against ever concluding "gain is fine" from
power alone. The right alert pairs them: low power → deaf; non-zero clip ratio →
overloaded; and the daemon's own gain warnings (from Part 1's construction phase)
point the operator at whichever direction to move.

Wideband dongles get their own pair — `wideband_input_iq_power_dbfs{serial}` and
`wideband_input_iq_clip_ratio{serial}` — because a single overloaded wideband
front-end buries *every* tap at once, and a per-tap gauge can't show a failure
that's common to all of them. On the lock side, `control_channel_locked{system}`
is the binary "are we decoding this system" gauge, and
`control_channel_transitions_total{system,event}` counts lock/lost flips — a
system that's *technically* locked but churning through dozens of transitions an
hour is a marginal-SNR site you want to know about before it drops for good.

## Opt-in cost

Not every useful metric belongs in every deployment. The per-protocol FEC
correction-depth histograms — today `tetra_viterbi_corrections`, the count of
channel bits the TETRA §8.3.1 FEC chain fixed per recovered burst — are genuinely
useful for someone profiling on-air recovery margin, and genuinely noise for
everyone else. Their buckets only make sense if you know what a healthy p95 looks
like (≤ 8). So they're gated behind `metrics.detailed_fec: true`, and when it's
off the histogram is nil and its `Record*` method no-ops. A stock daemon doesn't
pay the cardinality or the cognitive cost of a metric family it can't read; the
operator who wants it opts in. That's the same graceful-scaling instinct the
feature matrix (Part 8) applies to whole subsystems — the default carries what
everyone needs, and depth is a flag away.

## Where this goes next

Metrics tell you *that* something is wrong and roughly where. Logs tell you *what*
happened, in order. [Part 5]({{ '/blog/deep-dives/running-it-for-real-05-structured-logs/' | relative_url }})
opens the three structured logs GopherTrunk writes — the JSONL event log, the
human-readable decoded-message log, and the decode-gated power log — plus the
panic-recovery guard that keeps one bad goroutine from taking the timeline down
with it. The [Hardening]({{ '/hardening.html' | relative_url }}) doc has the full
metric table and Prometheus wiring; the [status]({{ '/status.html' | relative_url }})
reference covers the health endpoint alongside it.

## FAQ

**Why two ways of producing metrics instead of one?**
Because event counts and live device state have different truth models. Counts are
naturally push (accumulate as events happen); tuner state is naturally pull (read
it fresh so it can't be stale). Forcing either into the other's model produces a
metric that eventually lies.

**Which metric should my main "is it working" alert use?**
`control_channel_locked` for "are we decoding this system," backed by
`sdr_iq_power_dbfs` and `sdr_iq_clip_ratio` to explain a `0` — deaf versus
overloaded. For call throughput, alert on `calls_started_total`, not
`calls_total`.

**Why is `sdr_gain_db` sometimes NaN?**
Because the tuner is running AGC and has no fixed gain to report. `NaN` is the
honest value; filter it out by pairing the query with `sdr_gain_auto == 1`.

**Do metrics cost anything when nothing's happening?**
Almost nothing. Push counters idle with the event bus, and the snapshot collector
only runs work when Prometheus actually scrapes `/metrics`. The optional FEC
histograms are off unless you enable `metrics.detailed_fec`.

**What clips versus what's deaf — how do I tell from the tiles?**
Deaf reads low `iq_power_dbfs` (near −45). Overloaded reads a non-zero
`iq_clip_ratio` even if power looks fine, because RMS averages peaks away. Never
raise gain on a non-zero clip ratio — that's the front-end already too hot.

## Series navigation

**Part 4 of 14** · ←
[Part 3: TLS & Sitting Behind a Reverse Proxy]({{ '/blog/deep-dives/running-it-for-real-03-tls-reverse-proxy/' | relative_url }})
· Next →
[Part 5: Structured Logs — Event, Message & Power]({{ '/blog/deep-dives/running-it-for-real-05-structured-logs/' | relative_url }})
