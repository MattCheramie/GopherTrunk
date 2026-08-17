---
title: "TETRA End to End, Part 10: The Control Channel Under Stress — Sync Loss & the CC Equalizer"
description: "Reading a one-hour field session's eleven hard control-channel sync losses: why the compute theory was wrong, how a signal-time resync budget survives CPU starvation, and the one TETRA control path that wasn't running the equalizer — lifting a marginal capture from twelve percent to full BSCH yield."
category: deep-dives
keywords: tetra control channel sync loss, bsch decode yield, cc equalizer, signal-time resync, stale watchdog marklost, control channel re-hunt, marginal snr tetra, cma control channel, gophertrunk tetra
tags: [tetra-end-to-end, tetra, control-channel, equalizer, reliability, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "TETRA End to End"
series_part: 10
---

*Part 10 of **TETRA End to End**, a 14-part deep dive into how GopherTrunk turns
one real 25 kHz TETRA carrier into clear recorded voice.
[Part 9]({{ '/blog/deep-dives/tetra-end-to-end-09-equalizer-voice-path/' | relative_url }})
put a snapshot-frozen CMA equalizer on the voice path and roughly doubled TCH/S
yield. This part turns the same forensic discipline on the thing every voice
follow depends on: the control channel itself. A reporter's one-hour session
logged ~210 control-channel transitions and eleven hard sync losses, each one
tearing down the site and re-hunting. The obvious suspect was CPU load. The
data said otherwise — and pointed at the one TETRA control path in the whole
codebase that wasn't running the equalizer we'd already shipped everywhere
else.*

> **TL;DR:** Eleven hard CC sync losses in one hour, each preceded by
> **`bsch_fail`** climbing and **`sb_bursts`** collapsing, then repeated
> `tetra: dsp resync (signal-time decode drought)`, then the 5 s stale watchdog
> (`ControlChannel.CheckStale` → **`MarkLost`**) and a re-hunt. **Not compute:**
> all 11 losses occurred with ~0 concurrent voice follows, and the 704
> `decode_overruns` were one bursty event — zero correlation with load. The
> captures split cleanly by in-channel SNR: healthy ≈**18 dB** replays at BSCH
> **147/0**; a `cc_sync_loss` capture at ≈**10 dB** locks but decodes only
> ~**22%** of its BSCH. The primary single-channel **`newTETRAPipeline`** was
> the one TETRA CC path *not* running `SnapshotCMA`; enabling it lifts the
> marginal fixture from ~**12% to ~100%** CRC-clean BSCH — pinned by
> `pipelines_tetra_equalizer_test.go` against
> `testdata/tetra_cc_sync_loss_2s_144k.cs16`.

**Key takeaways**

- **Correlate before you fix.** The sync losses had zero correlation with call
  or CPU load — every loss happened at ~0 concurrent follows. A compute fix
  would have been engineering against a phantom.
- **Measure resync budgets in signal time, not wall clock.** A descheduled
  goroutine processes no samples, so a sample-count budget never advances
  during starvation — destructive resets fire only on genuine decode droughts.
- **A marginal lock is a yield problem, not a lock problem.** The ~10 dB
  capture *locks*; it just decodes 22% of its sync bursts, and the resulting
  drought-resync storm is what eventually drops the lock.
- **The equalizer mitigates; it does not replace RF.** The residual −44 dBFS
  peak is a weak front end — an antenna/gain condition the equalizer rides
  through but the operator should still fix.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| CC pipeline | receiver + `tetra.ControlChannel`, now with CMA | `internal/scanner/ccdecoder/pipelines.go` (`newTETRAPipeline`) |
| Stale watchdog | 5 s of nothing decoded ⇒ `cc.lost`, re-hunt | `internal/radio/tetra/control.go` (`CheckStale`, `MarkLost`) |
| DSP resync | reset timing/AFC on a *signal-time* decode drought | `pipelines.go` (`tetraResyncTimeout`, 1.5 s of samples) |
| Health counters | `sb_bursts`, `bsch_fail`, `decode_overruns` … | `pipelines.go` (decode-status line), `control.go` stats |
| Loss auto-capture | records IQ around each sync loss | `on_cc_sync_loss` (daemon capture hook) |
| The A/B proof | equalizer off ~12% vs on ~100% BSCH, same fixture | `pipelines_tetra_equalizer_test.go` (`TestTETRACCEqualizerABIsolatesCause`) |
| Ship gate | the LIVE pipeline must reach the healthy regime | `pipelines_tetra_equalizer_test.go` (`TestTETRACCEqualizerLiftsMarginalBSCHYield`) |
| Marginal fixture | real 2 s / 144 kHz cs16 from a re-acquisition | `internal/scanner/ccdecoder/testdata/tetra_cc_sync_loss_2s_144k.cs16` |

## In this post

- **Anatomy of a sync loss** — the counter signature before every drop.
- **Ruling out the compute theory** — what the correlations actually said.
- **Signal-time resync** — the watchdog design that starvation can't trip.
- **The path that missed the equalizer** — and the A/B that pins causation.
- **What the equalizer doesn't fix** — the −44 dBFS residual.

## Anatomy of a sync loss

The reporter's session ran a TETRA site for an hour and logged ~210
`control_channel_transitions` with eleven hard losses, each auto-captured by
the `on_cc_sync_loss` hook — which is why this post gets to be forensic instead
of speculative. Every loss followed the same script. First `bsch_fail` starts
climbing while `sb_bursts` collapses: the receiver still sees the carrier, but
the once-a-second BSCH synchronisation bursts stop surviving their CRC. Then
the log fills with `tetra: dsp resync (signal-time decode drought)` — the
pipeline concluding, repeatedly, that its symbol timing must be off and
resetting to center. Each reset is destructive (it discards a *converged*
Gardner timing loop and AFC), so a receiver that was marginally decoding gets
knocked back to acquisition over and over — the storm. Finally the 5 s stale
watchdog fires:

```go
// internal/radio/tetra/control.go (shape) — the backstop
// CheckStale declares the control channel lost (publishes cc.lost via
// MarkLost) when nothing has decoded within the timeout.
func (c *ControlChannel) CheckStale(now time.Time, timeout time.Duration) {
    /* … lock-free heartbeat read … */
    c.MarkLost() // cc.lost ⇒ the cchunt supervisor leaves StateLocked, re-hunts
}
```

`tetraLockStaleTimeout` is 5 s — generous, about five missed multiframes at the
~1 s BSCH cadence — so a healthy carrier never trips it. After each re-lock the
session also logged a
[#815](https://github.com/MattCheramie/GopherTrunk/issues/815) carrier warning,
the post-relock breadcrumb that these captures were being acquired off-center
(~−3 kHz).

## Ruling out the compute theory

Eleven losses under a scanner that also decodes voice invites the load theory:
the CC goroutine starves under concurrent calls, decode falls behind, watchdog
fires. Two numbers killed it. **All eleven losses occurred with approximately
zero concurrent voice follows** — the site was quiet when it dropped. And the
session's 704 `decode_overruns` turned out to be one bursty event, not a
background rate — no correlation in time with any loss.

## Signal-time resync: a watchdog starvation can't trip

That mattered doubly, because the *resync design itself* had already been
hardened against starvation, and the data confirmed the design rather than
implicating it. The drought detector is measured in **signal time**:

```go
// internal/scanner/ccdecoder/pipelines.go (shape)
// tetraResyncTimeout … is expressed as a duration but measured against
// PROCESSED-SIGNAL time, not wall clock: checkResync counts the post-DDC IQ
// samples actually fed to the receiver since the last decode and compares
// them against a sample budget (tetraResyncTimeout × the channel rate).
const tetraResyncTimeout = 1500 * time.Millisecond
```

A wall-clock window ages out while a descheduled goroutine sits idle, firing a
destructive reset that discards a still-good lock — the field captures from the
earlier multislot investigation showed ~46 such starvation resyncs before the
change. A sample budget is immune: no samples processed, no budget consumed.
So when *this* session showed resync storms, they were real decode droughts on
real signal — the receiver processing 1.5 s of carrier without one CRC-clean
decode. The question became: why does a locked receiver stop decoding?

## The path that missed the equalizer

The auto-captures answered it. They split cleanly by in-channel SNR: a healthy
`concurrent` capture measures ≈**18 dB** and replays at BSCH **147/0** — 100%.
A `cc_sync_loss` capture measures ≈**10 dB**, *locks*, but decodes only
~**22%** of its BSCH, with the constellation visibly ISI-smeared. Ten dB is the
marginal regime — and Part 9 had already shipped the tool for exactly this
signature. The embarrassment was the wiring: the voice composer ran
`SnapshotCMA`; the wideband-T2 TETRA path ran it (its comment even claimed it
mirrored the ccdecoder settings); the primary single-channel
**`newTETRAPipeline`** — the path that hour-long session was running — did
not. One flag closed the gap:

```go
// internal/scanner/ccdecoder/pipelines.go (shape) — newTETRAPipeline
rx := tetrarx.New(tetrarx.Options{
    SampleRateHz: opts.SampleRateHz,
    DibitSink:    func(d []uint8, base int) { opts.tapDibits(d, base); cc.Process(d, base) },
    SoftSink:     func(diffs []complex64, base int) { cc.StashSoft(diffs, base) },
    ClockMode:    tetraClockMode, GardnerGain: 0.005,
    EnableAFC:           true,
    EnableChannelFilter: true,
    // On the reporter's ~10 dB re-acquisition capture it lifts CRC-clean BSCH
    // yield from ~12% to ~100% — the difference between riding through a
    // marginal dip and dropping lock → re-hunt. EnableDCBlock stays OFF —
    // reserved for the voice receivers.
    EnableEqualizer: true,
})
```

The proof is deliberately two tests, because they pin different claims.
`TestTETRACCEqualizerABIsolatesCause` hand-wires two chains that differ *only*
in `EnableEqualizer` and runs both over the same marginal fixture
(`testdata/tetra_cc_sync_loss_2s_144k.cs16`, a real 2 s slice from one of the
session's own auto-captures): off ≈12% BSCH yield, on ≈100%, with guards at
both ends (`okOn >= 4*okOff`, on-yield ≥ 90%, off-yield ≤ 40% so the fixture
stays honestly marginal). Because nothing else varies, the improvement cannot
be chunking, noise, or timing — it is the equalizer. The second,
`TestTETRACCEqualizerLiftsMarginalBSCHYield`, drives the *live*
`newTETRAPipeline` — whatever it is currently configured to do — and requires
the healthy regime. It was red before the flag and green after: the failing-first
ship gate, so the wiring can never silently regress back out.

<figure class="lab-figure">
<svg viewBox="0 0 680 220" width="680" height="220" role="img" aria-label="Timeline of one control-channel sync loss. In the healthy region BSCH bursts decode about once per second. Then in-channel SNR dips to about ten decibels: bsch_fail climbs, sb_bursts collapse, and repeated destructive DSP resyncs fire on decode droughts. After five seconds with no decode the stale watchdog publishes cc.lost and the supervisor re-hunts. A second lane shows the same dip with the equalizer enabled: BSCH yield stays near one hundred percent and no watchdog fires.">
  <line x1="40" y1="80" x2="660" y2="80" stroke="var(--fg-muted)"/>
  <text x="14" y="60" fill="var(--fg-muted)" font-size="9">without</text>
  <text x="14" y="72" fill="var(--fg-muted)" font-size="9">equalizer</text>
  <g stroke="currentColor"><line x1="60" y1="72" x2="60" y2="80"/><line x1="90" y1="72" x2="90" y2="80"/><line x1="120" y1="72" x2="120" y2="80"/><line x1="150" y1="72" x2="150" y2="80"/></g>
  <text x="105" y="98" text-anchor="middle" fill="var(--fg-muted)" font-size="9">BSCH ~1/s, clean</text>
  <rect x="180" y="46" width="300" height="44" rx="4" fill="none" stroke="var(--fg-muted)" stroke-dasharray="4 3"/>
  <text x="330" y="40" text-anchor="middle" fill="var(--fg-muted)" font-size="9">SNR dips to ~10 dB — ISI smears the constellation</text>
  <g stroke="currentColor"><line x1="205" y1="72" x2="205" y2="80"/><line x1="292" y1="72" x2="292" y2="80"/><line x1="401" y1="72" x2="401" y2="80"/></g>
  <text x="330" y="98" text-anchor="middle" fill="var(--fg-muted)" font-size="9">bsch_fail ↑ · sb_bursts ↓ · dsp resync ×N</text>
  <line x1="480" y1="52" x2="480" y2="80" stroke="currentColor"/>
  <text x="482" y="50" fill="currentColor" font-size="9">5 s drought</text>
  <circle cx="540" cy="80" r="5" fill="currentColor"/>
  <text x="540" y="66" text-anchor="middle" fill="currentColor" font-size="9">MarkLost → re-hunt</text>
  <line x1="40" y1="170" x2="660" y2="170" stroke="var(--fg-muted)"/>
  <text x="14" y="150" fill="var(--accent)" font-size="9">with</text>
  <text x="14" y="162" fill="var(--accent)" font-size="9">equalizer</text>
  <g stroke="var(--accent)"><line x1="60" y1="162" x2="60" y2="170"/><line x1="90" y1="162" x2="90" y2="170"/><line x1="120" y1="162" x2="120" y2="170"/><line x1="150" y1="162" x2="150" y2="170"/><line x1="205" y1="162" x2="205" y2="170"/><line x1="245" y1="162" x2="245" y2="170"/><line x1="285" y1="162" x2="285" y2="170"/><line x1="325" y1="162" x2="325" y2="170"/><line x1="365" y1="162" x2="365" y2="170"/><line x1="405" y1="162" x2="405" y2="170"/><line x1="450" y1="162" x2="450" y2="170"/><line x1="495" y1="162" x2="495" y2="170"/><line x1="540" y1="162" x2="540" y2="170"/><line x1="585" y1="162" x2="585" y2="170"/><line x1="630" y1="162" x2="630" y2="170"/></g>
  <text x="350" y="190" text-anchor="middle" fill="var(--accent)" font-size="9">same dip, ~100% BSCH — the lock rides through; no watchdog, no re-hunt</text>
  <text x="350" y="212" text-anchor="middle" fill="var(--fg-muted)" font-size="10">one flag on the one path that lacked it: the difference between a marginal dip and a site teardown</text>
</svg>
<figcaption>The same ~10 dB dip, with and without the CC equalizer: the un-equalized lane decays into a resync storm and a watchdog teardown; the equalized lane decodes through it.</figcaption>
</figure>

### How that principle shaped the Go code

- **The watchdog stack stayed untouched.** Signal-time resync, the 5 s
  backstop, `MarkLost` — all correct, all confirmed by the data. The fix went
  where the evidence pointed: decode yield, not lifecycle.
- **`EnableDCBlock` stays off the CC path.** The DC blocker is a voice-receiver
  tool (per `receiver.go`); the control channel's steady-state decode must not
  be disturbed by a filter it doesn't need. Parity of *settings* across paths
  is checked per-flag, not wholesale.
- **Synthetic fixtures got noisier on purpose.** Blind CMA is well-defined only
  against a noise floor — a literally noise-free constant-modulus input is a
  degenerate case. The full-daemon `TestDaemonCCDecodesTETRA` now adds 40 dB
  AWGN so the equalizer sees a fair fixture (the same reason the clean-channel
  receiver test adds 30 dB).

## What the equalizer doesn't fix

The auto-captures peak at **−44 dBFS** with no clipping. That is a weak front
end — an RF, gain, or antenna condition, the same class of finding as
[#764](https://github.com/MattCheramie/GopherTrunk/issues/764) — and the
equalizer *mitigates* it by spending the remaining SNR more efficiently; it
does not replace the missing decibels. The operator guidance is unchanged and
belongs to another series: stage the gain properly
([The Analog Edge Part 3]({{ '/blog/tutorials/analog-edge-03-gain-staging/' | relative_url }}))
and, when a symptom straddles the boundary, run the is-it-RF-or-software
checklist ([The Analog Edge Part 14]({{ '/blog/tutorials/analog-edge-14-field-checklist/' | relative_url }})).
A control channel that arrives at 18 dB doesn't need rescuing; the equalizer's
job is to make 10 dB survivable, not to make −44 dBFS a lifestyle.

## Where this goes next

That closes the TMO arc: one carrier, from constellation to conformant voice,
hardened at both the traffic and control layers. The next three parts leave the
network behind entirely.
[Part 11]({{ '/blog/deep-dives/tetra-end-to-end-11-dmo-direct-mode/' | relative_url }})
starts the DMO detour — TETRA's infrastructure-less direct mode, where two
radios talk peer-to-peer with no control channel at all, and where GopherTrunk
had to read the burst geometry from the spec and prove it against captures one
constant at a time.

## FAQ

**Why does a resync storm make things worse instead of better?**
Reset-to-center is destructive: it throws away a converged Gardner timing loop
and AFC state. On a genuinely off-lock receiver that's the fast path back. On a
marginally-*decoding* receiver it re-runs acquisition over and over, turning a
22%-yield channel into a 0%-yield one for the acquisition window each time.
That's why the trigger had to be a genuine signal-time drought.

**Why didn't the CC path have the equalizer already?**
Sequencing, honestly. The equalizer shipped on the voice path (Part 9) where
the garble was, and on the wideband-T2 path that copied those settings. The
primary single-channel pipeline predated both and nothing forced the audit
until the sync-loss session did. The ship-gate test now makes the setting
regression-proof.

**Is ~210 transitions/hour normal for a marginal site?**
It was the *symptom* here, not a baseline. Post-fix, the marginal regime that
caused most transitions decodes at ~100% BSCH, so the dips no longer cascade
into teardowns. Residual transitions on a weak install are an RF condition to
fix at the antenna, not a software rate to accept.

**What is `on_cc_sync_loss` and should I enable it?**
A daemon hook that auto-records IQ around each control-channel loss. It's the
reason this investigation had eleven captures instead of eleven anecdotes — on
any system showing unexplained transitions, enable it. The fixture in the
repo's testdata is literally one of its outputs.

**Could this have been the #764 front-end problem again?**
It shares the weak-signal setting (−44 dBFS) but not the mechanism: #764's
deficit was baked into the captured samples (phase noise at the Airspy's 10
MS/s clock), invariant to decode-path changes. Here an A/B on the *same
capture* moves yield 12%→100% — by definition a decode-path gap, with an RF
condition underneath that still deserves fixing.

## Series navigation

**Part 10 of 14** · ←
[Part 9: The Equalizer on the Voice Path]({{ '/blog/deep-dives/tetra-end-to-end-09-equalizer-voice-path/' | relative_url }})
· Next →
[Part 11: DMO I — Direct Mode & the DSB/DNB Geometry]({{ '/blog/deep-dives/tetra-end-to-end-11-dmo-direct-mode/' | relative_url }})
