---
title: "RF Front End, Part 10: Airspy R2/Mini & HF+ — Real Samples to Complex Baseband"
description: How GopherTrunk turns the Airspy R2/Mini's bare real ADC stream into complex baseband entirely on the host — a leaky DC blocker, a multiplier-free Fs/4 mix, and a stateful half-band Hilbert pair — then folds in the Airspy HF+, whose native int16 IQ needs none of it.
category: deep-dives
tags: [sdr, go, airspy, dsp, software-design]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "RF Front End"
series_part: 10
---

*Part 10 of **RF Front End**. The last three posts moved bytes off the wire.
This one does signal processing in the driver: the Airspy R2/Mini hands us bare
real ADC samples, not IQ, so the host has to synthesize complex baseband from
scratch — DC removal, an Fs/4 translation, and a half-band Hilbert pair — and
get the filter state to survive USB packet boundaries.*

> **TL;DR** — The Airspy R2/Mini stream bare real ADC samples, not IQ, so the
> host synthesizes complex baseband: a leaky DC blocker, a multiplier-free Fs/4
> mix, and a half-band Hilbert pair. The headline bug (#454): the converter must
> carry filter state across USB packets, or you get ~78° quadrature imbalance and
> no image rejection — fixed by threading one stateful converter through the
> whole stream. The HF+ needs none of it: it delivers native int16 IQ.

## In this post

- Why the [Airspy]({{ '/reference/airspy/' | relative_url }}) R2/Mini stream
  **bare real samples** (uint16, 12-bit, DC at 2048) at **twice** the IQ rate,
  unlike RTL-SDR's interleaved IQ.
- The four-stage **real-to-complex pipeline** in `iqconverter.go`: leaky-HPF DC
  removal → multiplier-free **Fs/4** mix → polyphase **half-band** split →
  decimate by two.
- Why the converter must be **stateful across packets**, and the bug (#454) we
  hit getting that filter memory right.
- The Airspy **HF+**, which delivers native interleaved **int16 IQ** and needs
  none of this — plus the shared-VID:PID-disambiguated-by-product-string wrinkle.

## What a real-sampling front end is

Most cheap SDRs hand the host quadrature IQ: two streams, in-phase and
quadrature, already mixed to baseband by the tuner. RTL-SDR does this (unsigned
8-bit I,Q); the HackRF does this (signed 8-bit I,Q); the Airspy HF+ does this
(signed 16-bit I,Q). You decode a pair of integers into one complex sample and
you are done.

The Airspy R2 and Airspy Mini do not. They are **real-sampling** receivers: a
single fast ADC digitizes a real intermediate-frequency signal, and the
firmware streams the bare ADC output — unpacked little-endian `uint16`, 12-bit
resolution, DC sitting at 2048 — at **twice** the configured IQ rate. There is
no quadrature channel coming over USB. Producing complex baseband is a host-side
job, and it is the same job an analog superheterodyne would hand to a pair of
mixers and a phasing network: take a real signal, reject its mirror image, and
land it at zero IF as `I + jQ`.

That `2×` is not incidental. A real signal sampled at `Fs` carries usable
bandwidth up to `Fs/2`; a complex signal at `Fs/2` carries the same bandwidth
across `±Fs/4`. So `N` real input samples become `N/2` complex outputs at half
the rate — which is exactly why, in the driver, a requested IQ rate of 3 MHz
selects the **6 MSPS** device mode. We covered that rate-doubling foot-gun in
the [Part 2]({{ '/blog/deep-dives/rf-front-end-02-the-device-contract/' | relative_url }})
device contract; here we build the converter that justifies it.

## How GopherTrunk implements it in Go

**The whole conversion lives in `internal/sdr/airspy/iqconverter.go`, driven from
one method per USB packet. Its job, top to bottom: decode `uint16` reals,
remove DC, mix by `Fs/4`, run the half-band pair, emit `complex64`.**

### Stage 1 — leaky-HPF DC removal

The ADC parks DC at 2048 and the front end adds its own slow bias drift. Left
in, that DC becomes a fat spike at the center of every spectrum. We strip it
with a one-pole leaky high-pass — a running estimate of the mean that we
subtract before anything else touches the sample:

```go
// internal/sdr/airspy/iqconverter.go
x := (float32(binary.LittleEndian.Uint16(buf[2*i:])) - dcBias) / dcFull

// Leaky DC blocker: removes the residual ADC bias before the mix.
x -= c.avg
c.avg += dcLeak * x
```

`dcBias` is 2048, `dcFull` normalizes to roughly `[-1, 1)`, and `dcLeak` is
`0.01` — a slow tracker that follows bias drift without eating low-frequency
signal. The accumulator `c.avg` is **state**: it carries from one packet to the
next so the DC estimate doesn't reset (and re-spike) every 6 ms.

### Stage 2 — the Fs/4 mix, for free

To get to baseband we need to multiply the real stream by a complex sinusoid.
Pick that sinusoid at exactly `Fs/4` and the multiplications collapse into
nothing: `e^{-j(π/2)n}` walks the unit circle in steps of 90°, so its samples
are just `1, -j, -1, +j, …`. No multiplies, no sine table — only a sign flip and
a branch on which polyphase lane the sample falls in. That is the `phase`
counter, cycling `0..3`:

```go
// internal/sdr/airspy/iqconverter.go
switch c.phase {
case 0, 2:
    // In-phase branch. The Fs/4 mix alternates the sign of the
    // even samples; push into the FIR and recompute its output.
    if c.phase == 0 {
        x = -x
    }
    c.pushI(x)
    c.lastI = c.firI()
case 1, 3:
    // Quadrature branch. The mix alternates sign and folds in the
    // 0.5 half-band centre tap; the matched delay aligns it with
    // the in-phase FIR's group delay, then we emit one sample.
    q := 0.5 * x
    if c.phase == 1 {
        q = -q
    }
    // ...delay line, then emit complex(c.lastI, qOut)
}
```

`phase` is also state — it persists across packets so the `(-j)^n` rotation
never glitches at a buffer boundary.

### Stage 3 — the half-band Hilbert pair

Mixing by `Fs/4` puts the wanted signal at baseband but also folds the mirror
image right on top of it. Rejecting that image is the whole game, and it falls
out of a **47-tap half-band** filter, split into its two polyphase lanes:

- **In-phase lane** (even samples) runs a symmetric low-pass FIR. A half-band's
  odd taps are zero except the center, so only **24 non-trivial taps**
  participate — half the multiplies of a general FIR for the same response.
- **Quadrature lane** (odd samples) is a **matched integer delay**. The half-band
  center tap is exactly `0.5`, and rather than spend a tap on it we fold that
  `0.5` straight into the `Fs/4` mix (`q := 0.5 * x` above). The delay line just
  lines Q up with the FIR's group delay.

```go
// internal/sdr/airspy/iqconverter.go
// firI evaluates the symmetric half-band FIR over the current window, with
// hbKernel[0] weighting the newest sample and hbKernel[hbTaps-1] the oldest.
func (c *iqConverter) firI() float32 {
    var acc float32
    idx := c.iwinPos - 1
    if idx < 0 {
        idx += hbTaps
    }
    for j := 0; j < hbTaps; j++ {
        acc += hbKernel[j] * c.iwin[idx]
        if idx--; idx < 0 {
            idx += hbTaps
        }
    }
    return acc
}
```

Both lanes carry state — the in-phase `iwin` FIR window and the quadrature
`qline` delay line are circular buffers on the `iqConverter` struct:

```go
// internal/sdr/airspy/iqconverter.go  (shape)
type iqConverter struct {
    avg     float32         // leaky DC-blocker accumulator
    phase   int             // Fs/4 mix phase, 0..3, persists across packets
    iwin    [hbTaps]float32 // in-phase FIR window
    iwinPos int
    lastI   float32         // in-phase output awaiting its paired quadrature
    qline   [hbHalf]float32 // quadrature delay line
    qpos    int
}
```

### Stage 4 — decimate, and pair up

Because we only ever emit on the odd (quadrature) phases, and only consume the
in-phase FIR's latest output (`c.lastI`) when we do, the decimation by two is
implicit: every fourth real sample produces no output, every pair of lanes
collapses to one `complex64`. The driver allocates `len(buf)/4` outputs and
returns them:

```go
// internal/sdr/airspy/iqconverter.go
func (c *iqConverter) processRaw(buf []byte) []complex64 {
    nReal := len(buf) / 2
    out := make([]complex64, 0, nReal/2+1)
    // ...per-sample pipeline above...
    return out
}
```

One bulk-IN payload of `N` real bytes-as-uint16 yields `N/2` real samples and
`N/4` complex samples, at exactly the configured IQ rate.

## The problem we hit: a half-band Hilbert that joined packets seamlessly (#454)

**Symptom.** When the Airspy R2 first came up, nothing locked. The constellation
showed a massive quadrature imbalance — about **78°** off from the 90° it should
be — and essentially **no image rejection** (~3 dB). Every downstream decoder
mistuned; on a quiet band all that survived was the DC spike. The device
streamed bytes fine, but the signal was garbage.

**Root cause.** Two compounding mistakes. First, the driver originally treated
the receiver's **real** ADC stream as if it were interleaved I/Q — pairing
adjacent real samples into `complex(I, Q)`. That is simply the wrong model for a
real-sampling front end: there is no Q channel on the wire, so the synthetic Q
was just a delayed copy of I, hence the ~78° imbalance and no image rejection.
Second, once we built the proper Fs/4-plus-half-band converter, the filter and
mix had to be **stateful across USB packets**. A naive implementation that reset
the FIR window, the DC accumulator, and the `phase` counter at the start of each
`processRaw` call produced a discontinuity every 6 ms — a click train at the
packet rate that smeared the spectrum.

**The Go fix.** All converter memory lives on the `iqConverter` struct, and a
single converter instance threads through an entire stream. The packet handler
just calls `processRaw` on the *same* converter, packet after packet:

```go
// internal/sdr/airspy/airspy.go
// Fresh real-to-IQ converter per stream so filter memory never carries
// over from a previous session.
d.cnv = newIQConverter()

out := make(chan []complex64, 8)
onPacket := func(buf []byte) {
    samples := d.cnv.processRaw(buf)
    select {
    case out <- samples:
    case <-ctx.Done():
    }
}
```

The converter is created **per stream** (so a retune starts from silent filter
memory, phase 0) but is **shared across every packet within that stream** (so the
FIR window, DC tracker, and mix phase flow continuously across boundaries). The
zero value of `iqConverter` is a valid reset state, which is what makes
`newIQConverter()` a one-liner. After the fix, image rejection came back to
~**70 dB**, matching what libairspy's own IQ modes deliver.

## The design principle: DSP in the driver, from first principles

The Airspy driver is the first place in GopherTrunk where the driver does real
*signal processing*, not just byte shuffling. **The principle: when the hardware
hands you something other than the abstraction the rest of the system wants
(`[]complex64`), the driver pays the cost of bridging the gap — and it does it
from first principles, not by linking someone else's DSP library.**

### How that principle shaped the Go code

- **The abstraction boundary holds.** Upstream code never learns the Airspy is a
  real-sampling device. `StreamIQ` returns the same `<-chan []complex64` every
  other driver returns; the Fs/4 mix and half-band pair are an implementation
  detail behind it.
- **State is explicit and owned.** The DSP carries memory — DC accumulator, mix
  phase, FIR window, delay line — and all of it is fields on one struct with a
  clear lifetime (one per stream). There are no package globals and no hidden
  static buffers, so two Airspys streaming at once can't corrupt each other.
- **Cheap by construction, not by optimization.** The Fs/4 choice turns the mixer
  into a sign flip; the half-band structure zeroes half the taps and folds the
  center tap into the mix. The fast path is fast because the *math* was chosen
  well, not because the loop was hand-tuned.
- **Correctness is testable without hardware.** Because `processRaw` is a pure
  function of `(converter state, bytes)`, the in-package tests feed synthesized
  real tones through it and assert on image rejection — no Airspy required.

## Folding in the Airspy HF+

The Airspy **HF+** family (HF+, HF+ Discovery, HF+ Dual Port) is a sibling, not a
twin, and it is instructive precisely because it makes the opposite choice. It
covers **9 kHz–31 MHz HF plus 60–260 MHz VHF**, and it delivers **native
interleaved int16 IQ** — so there is no real-to-complex stage at all. Decoding is
the boring two-integers-to-one-complex loop:

```go
// internal/sdr/airspyhf/airspyhf.go
func decodeInt16IQ(buf []byte) []complex64 {
    n := len(buf) / 4
    out := make([]complex64, n)
    for i := 0; i < n; i++ {
        iv := int16(binary.LittleEndian.Uint16(buf[4*i:]))
        qv := int16(binary.LittleEndian.Uint16(buf[4*i+2:]))
        out[i] = complex(float32(iv)/32768, float32(qv)/32768)
    }
    return out
}
```

No converter, no filter state, no `2×` rate trick. The HF+ proves the point that
the R2/Mini converter is *device-specific*: the complex-baseband abstraction is
the same, but how much work the driver does to honor it depends entirely on what
the silicon streams.

The HF+ has two wrinkles worth calling out. First, **all three variants share one
VID:PID** (`0x03eb:0x800c`); the USB descriptor's **Product string** is the only
observable model identifier, so the driver disambiguates on it:

```go
// internal/sdr/airspyhf/airspyhf.go
func variantName(product string) string {
    p := strings.ToUpper(product)
    switch {
    case strings.Contains(p, "DISCOVERY"):
        return "Airspy HF+ Discovery"
    case strings.Contains(p, "DUAL"):
        return "Airspy HF+ Dual Port"
    default:
        return "Airspy HF+"
    }
}
```

Second, "gain" on the HF+ is mostly **attenuation reduction**. The front end has
a fixed conversion stage; what the operator controls is HF_AGC (firmware-managed
LNA + mixer), HF_ATT (a 0–48 dB attenuator in 6 dB steps), and HF_LNA (a fixed
`+6` dB preamp). A negative tenth-dB target enables AGC and zeroes the rest;
positive values disable AGC, compute an attenuator step, and switch the LNA in
once attenuation reduction alone has been spent:

```go
// internal/sdr/airspyhf/airspyhf.go  (shape)
att := tenthDB / hfATTStepTenthDB      // 6 dB steps, clamp 0..8
remaining := tenthDB - att*hfATTStepTenthDB
lnaOn := remaining >= hfLNAGainTenthDB // +6 dB preamp
```

Note also that the HF+ vendor opcodes are **not** the R2/Mini's — sibling
devices, sibling protocols (`SET_SAMPLERATE` is 4 here, 12 on the R2). The two
drivers live in separate packages for exactly that reason.

## Where this goes next

We now have three of the four wire formats covered: RTL-SDR's unsigned 8-bit IQ,
the Airspy R2/Mini's synthesized complex baseband, and the HF+'s native int16 IQ.
The simplest of all — HackRF's signed 8-bit interleaved IQ, plus the
identity-by-board-ID dance at open time — is
[Part 11]({{ '/blog/deep-dives/rf-front-end-11-hackrf-one/' | relative_url }}).
After that, Part 12 zooms back out to the pool and USB hotplug watchdog that
keeps all of these streaming through unplug-and-replug.

## FAQ

**Why not just pair adjacent real samples as I and Q?**
Because there is no Q on the wire. The Airspy R2/Mini stream a single real ADC
channel; pairing reals fabricates a Q that's just a delayed I, which is what gave
us the ~78° imbalance and ~3 dB image rejection in #454. You have to *synthesize*
quadrature with a Hilbert-pair filter.

**Why a half-band filter specifically?**
Half-band filters have their odd taps zeroed (except the center) and a center tap
of exactly 0.5. That halves the multiplies in the in-phase lane and lets us fold
the center tap into the free Fs/4 mix — the structure does double duty as the
anti-alias filter for the decimate-by-two.

**Why is the converter created per stream but shared per packet?**
Per stream so a retune starts from clean filter memory and can't inherit a stale
DC estimate. Per packet so the FIR window, DC accumulator, and mix phase flow
continuously and packets join without a click at the boundary.

**Does the HF+ ever need the converter?**
No. The HF+ delivers complex IQ natively (int16 I,Q), so its driver just
normalizes integers to `[-1, 1]`. The real-to-complex converter is unique to the
R2/Mini's real-sampling front end.

## Series navigation

**Part 10 of 14** · ←
[Part 9]({{ '/blog/deep-dives/rf-front-end-09-rtlsdr-streaming-gc-churn/' | relative_url }})
· Next →
[Part 11: HackRF One — signed 8-bit IQ end to end]({{ '/blog/deep-dives/rf-front-end-11-hackrf-one/' | relative_url }})
