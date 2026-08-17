---
title: "The Analog Edge, Part 6: Sample Rate — The Decode Path Doesn't Care; the Front End Does"
description: Why GopherTrunk's decoders are deliberately rate-invariant — every capture rate is normalized to one per-protocol channel rate before the receiver ever runs — so a symptom that tracks the sample rate indicts the capture, plus what higher rates genuinely buy and the two warnings that mean the consumer fell behind.
category: tutorials
keywords: sdr sample rate choice, rate invariant dsp, downconverter channel rate, 48 khz channel rate, tetra 144 khz, decode cant keep up, host drops soapyremote, wideband sample rate tradeoff, gophertrunk analog edge
tags: [analog-edge, sample-rate, dsp, sdr, performance, tutorial]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "The Analog Edge"
series_part: 6
---

*Part 6 of **The Analog Edge**, a 14-part field guide to the analog half of a
GopherTrunk installation. [Part 5]({{ '/blog/tutorials/analog-edge-05-phase-noise-reciprocal-mixing/' | relative_url }})
kept leaning on one claim: that GopherTrunk's decode path treats a 2.4 MS/s
capture and a 10 MS/s capture identically, so a symptom that follows the
rate is a statement about the front end. Our marginal reader deserves to see
why that claim is true rather than take it on faith — because it's the claim
that decides which side of Part 1's line their next debugging hour goes to.
This part opens the down-converters, then prices the rate knob honestly:
what more megasamples buy, what they cost, and which scary log lines they
tend to produce.*

> **TL;DR:** Every capture, whatever its rate, is resampled to **one
> per-protocol channel rate** before the receiver runs — **48 kHz** for the
> 4800-baud C4FM family (P25/DMR/NXDN/…), **144 kHz** for TETRA's 18000-baud
> π/4-DQPSK (`ddcTargetForProtocol`, `internal/scanner/ccdecoder/ddc.go`) —
> and the receiver and AGC are sized from that *output* rate. So the decode
> path is **rate-invariant** (pinned by `TestDownconverterSNRInvariantAcrossRate`),
> and "locks at 2.4 MS/s, fails at 10 MS/s" points at the captured data
> (Part 5), not the DSP. Mind the map though: the replay path's
> single-channel `ccdecoder.Downconverter` and the wideband `DDCBank`
> (`internal/dsp/tuner/ddc.go`) are **separate code paths** — the gap that
> let [#764](https://github.com/MattCheramie/GopherTrunk/issues/764)'s first
> "fix" miss [#771](https://github.com/MattCheramie/GopherTrunk/issues/771)'s
> replay symptom. Higher rates buy coverage, and cost front-end cleanliness,
> CPU, and USB; when the CPU loses, the WARNs (`decode can't keep up with
> real time`, soapyremote `host_drops`) name a **downstream** problem, not a
> driver bug.

**Key takeaways**

- **Rate-invariance is a designed property, not luck.** Normalizing every
  input to the protocol's channel rate means one receiver tuning, one AGC
  sizing, one set of constants — verified at 2.5 and 10 MS/s by test.
- **That design is a diagnostic instrument.** If the DSP genuinely cannot
  tell capture rates apart, then a rate-dependent failure that reproduces in
  offline replay *must* live in the samples — the deduction that closed #764.
- **The sample rate knob prices in three currencies:** front-end cleanliness
  (Part 5's lesson), CPU per tap, and USB/network throughput. Spectrum
  coverage is what you're buying; know what you're paying.
- **Overrun warnings point downstream.** `sendOrDrop` sheds IQ only when the
  consumer stops draining it — so when `host_drops` climbs, ask what got
  slower on the decode side, not what's wrong with the driver.

## Cheat sheet

| Concern | The fact | Where it lives |
|---|---|---|
| Per-protocol channel rate | 48 kHz C4FM family, 144 kHz TETRA (incl. DMO) | `ddcTargetForProtocol` in `internal/scanner/ccdecoder/ddc.go` |
| Rate invariance, pinned | same in-channel SNR at 10 MS/s native vs decimated 2.5 MS/s | `TestDownconverterSNRInvariantAcrossRate` (`internal/scanner/ccdecoder/ddc_highrate_test.go`) |
| The two DDC paths | replay `-tune-hz` uses `ccdecoder.Downconverter`; multi-tap wideband uses `DDCBank` | `internal/scanner/ccdecoder/ddc.go` vs `internal/dsp/tuner/ddc.go` |
| Rate limits per hardware | RTL caps at 3.2 MHz; 2.4 MS/s is its sweet spot; 10/20 MS/s need a wideband source | `sdr.sample_rate` comments in `config.example.yaml` |
| Decode falling behind | IQ shed at the decode queue, WARN + metric | `internal/scanner/ccdecoder/decoder.go` (issue #402) |
| Network source falling behind | oldest chunk shed, `host_drops` in the WARN | `sendOrDrop` in `internal/sdr/soapyremote/driver.go` |

## In this post

- **What rate-invariance means** — one funnel, one receiver.
- **One channel rate per protocol** — the code that pins it.
- **Two down-converters, two paths** — the map that saved #771.
- **What higher rates buy, and what they cost** — the honest ledger.
- **The warnings that mean "downstream"** — reading overruns correctly.

## What rate-invariance means

Think of the decode pipeline as a funnel. At the wide end, captures arrive
at whatever rate your hardware produced: 2.4 MS/s from an RTL dongle,
6 MS/s from a B210, 10 MS/s from an Airspy. At the funnel's neck, a digital
down-converter (DDC) mixes the wanted channel to baseband, filters it, and
resamples it to a **fixed channel rate** chosen by protocol. Everything
after the neck — matched filter, timing recovery, AGC, slicer, FEC — sees
only that fixed rate and is *sized from it*. The receiver literally has no
input that tells it what the capture rate was.

That's the whole claim. A well-implemented funnel neck (proper anti-alias
filtering, exact rational resampling — the `Downconverter` uses a >60 dB
stopband polyphase design) delivers the channel's contents *as captured*:
whatever SNR the samples carried in-channel arrives at the receiver intact,
whether the journey started at 2.5 or 10 MS/s. The project didn't leave
that as an assertion: `TestDownconverterSNRInvariantAcrossRate` synthesizes
a noisy channel, decodes it natively at 10 MS/s and decimated at 2.5 MS/s,
and pins that the receiver sees the same in-channel SNR both ways.

And that test is why the claim doubles as a diagnostic tool. When a decode
works at one rate and fails at another *and the failure reproduces from a
capture in offline replay*, the invariant funnel can't be the difference —
the difference was already in the samples when they crossed Part 1's line.
That deduction, plus Part 5's physics, is #764's entire resolution.

<figure class="lab-figure">
<svg viewBox="0 0 680 210" width="680" height="210" role="img" aria-label="A funnel diagram. Three inputs at different sample rates — 2.4, 6, and 10 megasamples per second — converge into a digital down-converter box, which outputs one fixed per-protocol channel rate: 48 kilohertz for the C4FM family or 144 kilohertz for TETRA. A single receiver box follows, annotated that it is sized from the output rate and cannot tell capture rates apart. A note says a rate-dependent symptom that survives replay therefore lives in the samples.">
  <text x="60" y="40" fill="currentColor" font-size="10">2.4 MS/s (RTL-SDR)</text>
  <text x="60" y="86" fill="currentColor" font-size="10">6 MS/s (USRP)</text>
  <text x="60" y="132" fill="currentColor" font-size="10">10 MS/s (Airspy)</text>
  <line x1="190" y1="36" x2="268" y2="76" stroke="var(--fg-muted)"/><polygon points="264,72 274,79 266,82" fill="var(--fg-muted)"/>
  <line x1="190" y1="82" x2="268" y2="88" stroke="var(--fg-muted)"/><polygon points="264,84 274,89 264,92" fill="var(--fg-muted)"/>
  <line x1="190" y1="128" x2="268" y2="100" stroke="var(--fg-muted)"/><polygon points="264,96 274,97 266,104" fill="var(--fg-muted)"/>
  <rect x="276" y="62" width="120" height="52" rx="6" fill="none" stroke="currentColor"/>
  <text x="336" y="84" text-anchor="middle" fill="currentColor" font-size="10">DDC: mix, filter,</text>
  <text x="336" y="98" text-anchor="middle" fill="currentColor" font-size="10">rational resample</text>
  <line x1="396" y1="88" x2="424" y2="88" stroke="currentColor"/><polygon points="424,84 432,88 424,92" fill="currentColor"/>
  <rect x="432" y="62" width="112" height="52" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="488" y="82" text-anchor="middle" fill="var(--accent)" font-size="10">one channel rate</text>
  <text x="488" y="97" text-anchor="middle" fill="var(--fg-muted)" font-size="9">48 kHz C4FM · 144 kHz TETRA</text>
  <line x1="544" y1="88" x2="572" y2="88" stroke="currentColor"/><polygon points="572,84 580,88 572,92" fill="currentColor"/>
  <rect x="580" y="62" width="92" height="52" rx="6" fill="none" stroke="currentColor"/>
  <text x="626" y="82" text-anchor="middle" fill="currentColor" font-size="10">receiver</text>
  <text x="626" y="97" text-anchor="middle" fill="var(--fg-muted)" font-size="9">sized from output</text>
  <text x="340" y="156" text-anchor="middle" fill="var(--fg-muted)" font-size="10">the receiver has no input that says what the capture rate was — it cannot treat rates differently</text>
  <text x="340" y="182" text-anchor="middle" fill="var(--fg-muted)" font-size="10">so a symptom that tracks the rate AND survives offline replay was already in the samples (#764)</text>
</svg>
<figcaption>The rate funnel: any capture rate in, one per-protocol channel rate out, one receiver sized once — which is what turns "fails only at 10 MS/s" into evidence about the front end.</figcaption>
</figure>

## One channel rate per protocol

The neck of the funnel is small enough to quote in full:

```go
// internal/scanner/ccdecoder/ddc.go (shape) — ddcTargetForProtocol
// The 4800-baud C4FM family (P25 / DMR / NXDN / dPMR / YSF / D-STAR) and the
// other ≤9600-baud protocols all channelize to ddcTargetRateHz; TETRA's
// 18000-baud π/4-DQPSK needs a wider channel, so it gets
// tetraDDCTargetRateHz — including TETRA DMO (Direct Mode).
func ddcTargetForProtocol(p trunking.Protocol) float64 {
    if p == trunking.ProtocolTETRA || p == trunking.ProtocolTETRADMO {
        return tetraDDCTargetRateHz
    }
    return ddcTargetRateHz
}
```

Two rates cover the roster: 48 kHz gives the 4800-baud family ten samples
per symbol; TETRA's 18000-baud π/4-DQPSK gets 144 kHz — eight per symbol —
because feeding it the C4FM target would leave under three samples per
symbol and the timing loop would never lock (the exported
`DDCTargetForProtocol` wrapper exists precisely so offline replay sizes its
DDC to the *same* target the daemon uses). One subtlety worth knowing: the
funnel normalizes in both directions. A narrowband capture recorded *below*
the target is interpolated up to it, so the receiver always runs at its
designed samples-per-symbol — interpolation can't create information the
capture lacked, but it keeps the receiver's constants honest either way.

## Two down-converters, two paths

Rate-invariance holds *within* each path — but there are two. The
single-channel `ccdecoder.Downconverter` serves replay's `-tune-hz` and the
per-channel decoders; the multi-tap wideband `DDCBank` in
`internal/dsp/tuner/ddc.go` serves `role: wideband` dongles, channelizing
many taps out of one capture. They are separate implementations, and a fix
to one does not touch the other — the exact gap that let #764's first fix
(in the wideband path) leave #771's replay symptom (in the single-channel
path) standing. The operator-level takeaway isn't the code layout, it's the
debugging move it implies: when a wideband symptom needs isolating,
re-testing the same capture through `gophertrunk replay -tune-hz` swaps in
a different, simpler DDC — if the symptom follows the capture across *both*
paths, the DDCs are exonerated together and the samples are indicted. The
[Ten Megasamples postmortem]({{ '/blog/solution-postmortem/from-the-issue-tracker-05-ten-megasamples/' | relative_url }})
runs that play in full.

## What higher rates buy, and what they cost

With invariance established, rate choice is purely a front-end and
resources decision. The honest ledger:

| | Higher rate (6–10+ MS/s) | Lower rate (≤ 2.4–3 MS/s) |
|---|---|---|
| Spectrum coverage | many taps / whole sub-band per dongle; wideband hunting | one system cluster per dongle |
| Front-end cleanliness | configuration-dependent — measure it (Part 5; #764's 10 MS/s clock) | RTL's 2.4 MS/s sweet spot is well-trodden |
| CPU | every tap's DDC runs against the full input rate | proportionally cheaper |
| USB / network | 10 MS/s CS16 ≈ 40 MB/s sustained | comfortable on USB 2 |
| Headroom coupling | one gain/ADC shared by everything in the capture (issue #749) | fewer co-tenants to stage around |

The coverage upside is real — the entire
[wideband multi-site architecture]({{ '/blog/deep-dives/the-hunt-09-wideband-multisite-p25/' | relative_url }})
rides on it, and a [multi-dongle setup]({{ '/multi-dongle-sdr-setup/' | relative_url }})
is the alternative spend. But each cost column has produced tracker
traffic: Part 5 covered cleanliness; the shared-ADC column was Part 2 and
Part 4; and CPU is next. For the underlying theory (Nyquist, aliasing, and
why "more rate" never means "more fidelity per channel"), the
[sample-rate lesson]({{ '/learn/rf-sdr/sample-rate-nyquist/' | relative_url }})
and the [Field Guide entry]({{ '/reference/sample-rate/' | relative_url }})
have you covered.

## The warnings that mean "downstream"

Raise the rate past what your host digests and GopherTrunk starts shedding
— deliberately, at two designed points, each with a WARN that operators
routinely read backwards.

`ccdecoder: decode can't keep up with real time; dropping IQ at the decode
queue (raise CPU / lower sample rate / shed load)` means exactly what it
says: live IQ is arriving faster than the decode side consumes it, so
chunks are dropped at the decode queue (counted in a dedicated metric,
issue #402) rather than letting latency grow without bound.

`soapyremote: SDR overruns — the host can't keep up with the configured
sample rate…` with its `device_overflows` / `host_drops` counters is the
network-source sibling. The driver's read loop *never* blocks on the DSP
consumer — a blocked read stalls flow control and makes the radio itself
drop samples in a way that shreds every channel at once — so when the
bounded queue fills, `sendOrDrop` sheds the **oldest** queued chunk and
counts it. Each shed is an IQ discontinuity that will glitch decodes.

The reading discipline: `internal/sdr/soapyremote/` sheds only when the
consumer stops draining, so a climbing `host_drops` is a symptom of
something *downstream* getting slower — too many taps for the CPU, a new
per-call workload, an over-ambitious rate — and the fix lives there (the
companion ccdecoder WARN appearing too confirms CPU rather than network).
The project has already had one hunt where an apparent "driver overrun bug"
was a voice-chain CPU regression discovered exactly this way. Lower the
rate, shed taps, or add CPU; the driver is the messenger.

## Where this goes next

Every part so far has staged the signal the antenna delivered. Starting
with [Part 7]({{ '/blog/tutorials/analog-edge-07-antennas/' | relative_url }}),
we improve what gets delivered in the first place — the antenna itself:
which bands trunking actually lives in, why antenna "gain" is a shape
rather than a magnitude (and can point that shape away from a close
tower), polarization, and how to choose between a discone, a tuned
vertical, and a yagi for a fixed system.

## FAQ

**If the decode path is rate-invariant, why does my system decode better at
2.4 MS/s than at 8 MS/s?**
Because invariance is precisely what localizes the difference to your front
end and host: oscillator behavior at the higher clock (Part 5), a shared
ADC now staging more spectrum (Parts 2–4), or shed IQ from an overloaded
consumer (this part's WARNs). Capture at both rates and replay — if the gap
reproduces from the files, it's the front end; if the live system lags but
replay is clean, it's the host.

**Does capturing at a higher rate improve per-channel quality?**
No. The channel's information is set by its bandwidth and SNR at the
antenna; the funnel delivers exactly that to the receiver regardless of
capture rate. More rate widens the funnel's mouth (coverage), never the
quality at its neck — and per Part 5 it can degrade the neck's contents if
the front end is less clean at the higher clock.

**What rate should I run an RTL-SDR at?**
2.4 MS/s is the config's stated sweet spot, and RTL hardware caps at
3.2 MHz (the top rates trade reliability for little). The interesting rate
decisions arrive with hardware that offers real choices — Airspy, USRP,
Lime — where the Part 5 A/B (capture at each candidate, replay, compare
demod SNR) should settle it for your unit.

**Are dropped chunks ever acceptable?**
As a designed failure mode under transient load, yes — one bursty event
during a scan storm is survivable, and shedding beats unbounded latency or
device-side overflow storms. As a steady state, no: every shed chunk is an
IQ discontinuity mid-decode. A `host_drops` counter that climbs whenever a
voice call starts is telling you which workload to trim.

**Why does TETRA get 144 kHz instead of 48 kHz?**
Symbol rate. TETRA runs 18000 symbols/s against the C4FM family's 4800; at
48 kHz it would get under 3 samples per symbol, below what the Gardner
timing loop needs to lock. 144 kHz restores the designed 8 samples per
symbol. Same funnel, protocol-appropriate neck.

## Series navigation

**Part 6 of 14** · ←
[Part 5: Phase Noise & Reciprocal Mixing — The Ten-Megasamples Lesson]({{ '/blog/tutorials/analog-edge-05-phase-noise-reciprocal-mixing/' | relative_url }})
· Next →
[Part 7: Antennas for Trunking Bands]({{ '/blog/tutorials/analog-edge-07-antennas/' | relative_url }})
