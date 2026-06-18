---
title: "RF Front End, Part 8: RTL-SDR II — The R82xx Tuner & the Blog V4 Deafness"
description: "GopherTrunk's pure-Go R820T/R828D tuner driver — PLL tuning, a shadow-register cache that makes read-modify-write free, and tuner auto-detection. Plus the headline bug: why the RTL-SDR Blog V4 came up deaf and mistuned by 1.8×, and how a manual override plus per-band input switching fixed it."
category: deep-dives
tags: [sdr, go, rtl-sdr, r82xx, software-design]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "RF Front End"
series_part: 8
---

*Part 8 of **RF Front End**. The RTL2832U from
[Part 7]({{ '/blog/deep-dives/rf-front-end-07-rtlsdr-rtl2832u-bringup/' | relative_url }})
can't tune anything on its own — the actual frequency synthesis lives in a
separate tuner chip on the I2C bridge. This post brings up the
[R820T]({{ '/reference/r820t-tuner/' | relative_url }})/R828D, and tells the
story of the bug that ate the most field debugging time in the whole RTL-SDR
driver: the RTL-SDR Blog V4 that came up deaf.*

> **TL;DR** — This is the pure-Go R820T/R828D tuner driver, built around a
> shadow-register cache that makes read-modify-write free. The headline bug: the
> RTL-SDR Blog V4 came up deaf and mistuned by ~1.8×, fixed with a crystal
> override plus per-band input switching. A second bug (#248) — a 17-byte I2C
> burst stalling on NESDR v5 — is fixed by halving the chunk size until it fits.

## In this post

- **What the R82xx tuner does** — PLL frequency synthesis, the 3.57 MHz IF, and
  why the RTL2832U only ever sees an intermediate frequency.
- **The shadow-register cache** — how mirroring the chip's writable registers in
  Go makes read-modify-write free and papers over a real R820T quirk.
- **Tuner auto-detection** — probing I2C addresses in librtlsdr's exact order.
- **The problem we hit (issue #264):** the RTL-SDR Blog V4 mistunes by ~1.8× and
  receives only noise, and the manual override + per-band input switching that
  fixed it.
- **A second bug (issue #248):** a multi-byte I2C burst stalls on NESDR v5
  silicon, and the chunk-halving recovery that got it streaming.

## What the R82xx tuner is

The R820T, R820T2, and R828D (collectively the "R82xx" family) share one I2C
register map and one PLL synthesizer. The tuner's job is to take an RF frequency
from the antenna, mix it down to a fixed **3.57 MHz intermediate frequency**, and
hand that to the RTL2832U's ADC. So when you ask GopherTrunk to tune 154 MHz, the
tuner's PLL doesn't target 154 MHz — it targets 154 MHz + 3.57 MHz, and the
RTL2832U is configured (back in `PrepareDemod`) to expect signal at that IF
offset.

The driver is a straight port of osmocom librtlsdr's `tuner_r82xx.c` — register
addresses, the init flood, the PLL math, and the frequency-range mux table are
all kept byte-identical so real-hardware captures replay-validate against the
mock USB transport. The whole driver lives in
`internal/sdr/rtlsdr/tuners/r82xx.go`.

## How GopherTrunk implements it in Go

### The shadow-register cache

**The single most important design decision in the R82xx driver is the
shadow-register cache.** The `R82xx` struct keeps a local mirror of the chip's
registers:

```go
// internal/sdr/rtlsdr/tuners/r82xx.go
type R82xx struct {
    demod    *rtl2832u.Demod
    i2cAddr  uint8
    chipType Type
    xtalHz   uint32

    // regs[0x05..0x1F] is the shadow for writable registers.
    // regs[0x00..0x04] holds the most recently read read-only status bytes.
    regs [r82xxNumRegs]byte
    // ...
}
```

This solves a concrete hardware problem: the R820T's writable registers **can't
be read back** through the I2C bridge. Read-modify-write — "set these three bits,
leave the rest" — is impossible if you can't read the current value. The shadow
makes it trivial, and it has a free side benefit: the chip silently drops a write
whose value matches the previous one, so eliding redundant writes saves a USB
roundtrip without changing observable behavior. Every masked write goes through
`writeRegMask`, which reads the shadow, applies the masked bits, and only touches
the wire if the result actually changed:

```go
// internal/sdr/rtlsdr/tuners/r82xx.go
func (r *R82xx) writeRegMask(addr uint8, val, mask byte) error {
    if addr < r82xxShadowStart {
        return fmt.Errorf("r82xx writeRegMask: addr=0x%02x is read-only", addr)
    }
    cur := r.regs[addr]
    next := (cur &^ mask) | (val & mask)
    if cur == next {
        return nil
    }
    r.regs[addr] = next
    return r.writeBurstRaw(addr, []byte{next})
}
```

### PLL tuning

`setPLL` is a faithful port of `r82xx_set_pll`: it sweeps the mixer divider to
land the VCO inside its valid range, computes an integer + sigma-delta fractional
division, and compensates with a VCO fine-tune read back from the chip. The math
is intricate but the load-bearing detail for this post is small — the divider
sweep and the `nint`/`vcoFra` split off the reference crystal:

```go
// internal/sdr/rtlsdr/tuners/r82xx.go (shape)
vcoFreq := uint64(freqHz) * uint64(mixDiv)
effXtal := r.effectiveXtalHz()
pllRef := uint64(effXtal)
nint := uint32(vcoFreq / (2 * pllRef))
vcoFra := uint32((vcoFreq - 2*pllRef*uint64(nint)) / 1000)
```

Notice `effectiveXtalHz()`. *Every* PLL division is computed off the reference
crystal. If the driver believes the crystal is 16 MHz when it's actually 28.8
MHz, every tune is wrong by the ratio 28.8/16 = **1.8×**. Hold that thought.

### Tuner auto-detection

Before any of this runs, we have to figure out *which* tuner is on the board.
`detect.go` probes candidate I2C addresses in librtlsdr's exact order, reading
the chip-ID byte off each. For the R82xx family that's a 0x69 ID at address 0x34
(R820T) or 0x74 (R828D):

```go
// internal/sdr/rtlsdr/tuners/r82xx.go
func detectR82xx(d *rtl2832u.Demod) Tuner {
    for _, c := range []struct {
        addr uint8
        typ  Type
    }{
        {addr: r82xxI2CAddr, typ: TypeR820T2},
        {addr: r828dI2CAddr, typ: TypeR828D},
    } {
        out, err := d.I2CRead(c.addr, 1)
        if err != nil || len(out) == 0 {
            continue
        }
        id := r82xxBitReverse(out[0])
        if id == 0x69 || id == 0x96 { // includes some bit-reversed clones
            return NewR82xx(d, c.addr, c.typ)
        }
    }
    return nil
}
```

The whole detection runs under a single `SetI2CRepeater(true)/(false)` bracket so
the bridge doesn't flap between candidates. That detail matters more than it
looks — it sets up the second bug below.

## The problem we hit: the RTL-SDR Blog V4 deafness (issue #264)

**The symptom.** Plug in an RTL-SDR Blog V4 — a popular R828D-based dongle — and
it would detect fine, claim its interface, accept every tune… and receive only
noise. An R820T2 dongle on the exact same signal decoded cleanly. Worse, when it
*did* seem to tune, frequencies were off by a factor of roughly 1.8.

**The root cause — three of them, actually.** The V4 is an R828D, and our
`NewR82xx` defaults every R828D to a 16 MHz crystal — which is correct for a
generic R828D. But the Blog V4 runs its R828D from the RTL2832U's **28.8 MHz**
reference crystal. There's the 1.8× mistune, straight out of the PLL math above:

```go
// internal/sdr/rtlsdr/tuners/r82xx.go
func NewR82xx(d *rtl2832u.Demod, i2cAddr uint8, chip Type) *R82xx {
    xtal := r82xxXtalHz
    if chip == TypeR828D {
        xtal = r828dXtalHz // 16 MHz — wrong for the Blog V4
    }
    // ...
}
```

But fixing the crystal alone still left it deaf, because the V4 has a switched
front end: an HF/VHF/UHF input bank with an on-board upconverter for HF. The stock
R828D init leaves every V4 input *off*, so even perfectly tuned, no RF reaches the
mixer. And detection can't reliably tell a V4 from a generic R828D — the
distinguishing signal is the USB iManufacturer/iProduct strings, which are
sometimes blank or non-standard on real units.

**The fix — three parts.** First, an explicit `SetBlogV4` that restores the right
crystal and arms the V4 path:

```go
// internal/sdr/rtlsdr/tuners/r82xx.go
func (r *R82xx) SetBlogV4(lite bool) {
    r.blogV4 = true
    r.blogV4L = lite
    r.xtalHz = r82xxXtalHz // 28.8 MHz, overriding the R828D 16 MHz default
}
```

Detection keys off the USB strings, but because that misses some units, the
driver also exposes a *manual* override all the way up at the `Device` level —
`SetBlogV4` implements `sdr.BlogV4Forcer`, and the pool applies it from a config
hint before the first tune. Autodetection when it can; explicit override when it
can't.

Second, per-band input switching. The V4 routes HF through an upconverter (so the
R828D actually sees `hz + 28.8 MHz`), and switches a physical input bank per band.
`SetFreq` adjusts the PLL target for HF and then drives the input switches:

```go
// internal/sdr/rtlsdr/tuners/r82xx.go
target := hz
if r.blogV4 && hz <= r82xxV4HFCrossHz {
    target = hz + r82xxXtalHz // HF reaches the mixer through the upconverter
}
// ...setMux(target), then:
if r.blogV4 {
    if err := r.applyBlogV4Band(hz); err != nil {
        return fmt.Errorf("r82xx SetFreq: v4 band: %w", err)
    }
}
```

`applyBlogV4Band` drives the notch filter, the HF tracking-filter bypass, and the
HF/VHF/UHF input relays — ported verbatim from the rtlsdr-blog fork's
`r82xx_set_freq` V4 block. It caches the last-selected band in a `v4Band` enum so
it only rewrites the input switches when the band actually changes:

```go
// internal/sdr/rtlsdr/tuners/r82xx.go
band := v4BandFor(hz, r.blogV4L)
// HF: bypass tracking filter (re-applied every tune since setMux rewrites 0x1A/0x1B)
if band == v4BandHF { /* ...writeRegMask(0x1A,...) ... */ }
// Only rewrite the input switches on a band change.
if band == r.v4Input {
    return nil
}
r.v4Input = band
// ...cable2 = HF input, GPIO5 upconverter relay, cable1 = VHF, air-in = UHF...
```

Third, the gain path. The V4 is a marginal-signal dongle, and there was a latent
AGC-mode bug that compounded the deafness: in AGC mode librtlsdr pins the VGA at a
fixed +16.3 dB, but `SetGain` is a no-op in AGC mode, so the VGA was being left at
its init default — running the front end **~17 dB low**. `SetGainMode` now writes
the VGA in the AGC branch:

```go
// internal/sdr/rtlsdr/tuners/r82xx.go
// AGC mode: pin the VGA at librtlsdr's fixed default (+16.3 dB).
if !manual {
    if err := r.writeRegMask(0x0C, 0x0B, 0x9F); err != nil {
        return err
    }
}
```

There was even a fourth, subtler V4 issue tucked into the PLL: the rtlsdr-blog
fork lowers the VCO power reference from 2 to 1 for the R828D, and without that
the V4's LO mistunes and receives noise while an R820T2 on the same signal
decodes cleanly. `vcoPowerRef()` returns 1 for `TypeR828D`. The V4 needed *every
one* of these to receive — which is exactly why it was so painful to chase.

To make the state visible, `TunerDiag`/`IsBlogV4`/`XtalHz` surface the effective
crystal and whether the V4 path armed, so the pool can log a boot-time line: a 16
MHz R828D means the V4 path did *not* arm and the LO is mistuned by 1.8×; 28.8 MHz
means `SetBlogV4` ran.

## The second problem: a 17-byte I2C burst that stalls (issue #248)

**The symptom.** Two NESDR SMArt v5 units would fail tuner init with an `EPIPE`
(Linux) / `ERROR_GEN_FAILURE` (Windows) on the very first multi-byte I2C burst —
the 27-byte R82xx init flood. Detection succeeded; the burst write that follows it
did not.

**The root cause.** librtlsdr assumes the chip's I2C-bridge FIFO can swallow a
16-byte write (`NMAX_WRITES = 16`). On that specific firmware revision the FIFO
depth appears to be smaller, so the 17-byte first chunk (1 address byte + 16 data
bytes) NACKs. An earlier fix added a per-chunk retry; it wasn't enough.

**The fix.** `writeBurstRaw` halves the chunk size on a stall — 16 → 8 → 4 — until
one size succeeds:

```go
// internal/sdr/rtlsdr/tuners/r82xx.go
func (r *R82xx) writeBurstRaw(addr uint8, data []byte) error {
    var lastErr error
    for chunkSize := r82xxBurstMaxData; chunkSize >= r82xxBurstMinData; chunkSize /= 2 {
        err := r.writeBurstAtSize(addr, data, chunkSize)
        if err == nil {
            return nil
        }
        if !isI2CBurstStall(err) {
            return err
        }
        lastErr = err
        if chunkSize > r82xxBurstMinData {
            time.Sleep(r82xxBurstRetryDelayMillis * time.Millisecond)
        }
    }
    return fmt.Errorf("tried chunk sizes 16,8,4; all stalled: %w", lastErr)
}
```

Two details make this robust. The stall predicate has to be
cross-platform — the same logical stall is a raw `syscall.EPIPE` on Linux and a
mapped `usb.ErrPipeStalled` on Windows, so `isI2CBurstStall` checks both;
checking only `EPIPE` meant the whole recovery silently never fired on Windows.
And there's a detection-side dependency: `Detect` deliberately toggles the I2C
repeater *off* before returning so that `Init`'s leading `SetI2CRepeater(true)` is
a *fresh* wire write — that explicit "kick" is load-bearing on NESDR v5 to arm the
bridge, even though the cache from Part 7 would otherwise elide it.

## The design principle: defensive hardware-quirk handling

Both bugs come from the same place: **real hardware deviates from the reference,
and when it does, autodetection is not enough — you need explicit, observable
overrides.**

The Blog V4 looks like a generic R828D until it doesn't. The NESDR v5 looks like
a standard RTL-SDR until its FIFO chokes on a standard-sized write. A driver that
assumes every chip matches the datasheet — or even matches librtlsdr — comes up
deaf on exactly the dongles people actually buy.

### How that principle shaped the Go code

- **Autodetect first, override explicitly.** USB-string detection arms the V4
  path automatically when it can; `SetBlogV4` / `sdr.BlogV4Forcer` is the manual
  escape hatch for units it misses. The default path stays correct for the common
  case and the override is one config hint away.
- **Quirk state is observable.** `IsBlogV4`, `XtalHz`, and `TunerDiag` exist so a
  failed override shows up as a boot-time diagnostic ("16 MHz R828D → mistuned
  1.8×") instead of a silent deaf dongle.
- **Recovery degrades gracefully.** The burst write doesn't give up at the first
  stall — it halves the chunk size and retries, so one firmware's small FIFO
  costs a few extra round-trips instead of a hard failure. The stall predicate is
  written cross-platform so the recovery fires on every OS.
- **Quirk reproduction stays faithful underneath.** Every V4-specific write —
  band switching, VGA pinning, the VCO power reference — is ported verbatim from
  the rtlsdr-blog fork with a comment tying it to the source, so the override path
  is just as byte-faithful as the default path.

## Where this goes next

The RTL2832U is initialized and the R82xx is tuned and locked. The only thing
left is to actually *move samples* — to keep a 2.4 MS/s bulk-IN stream flowing
without dropping IQ.
[Part 9]({{ '/blog/deep-dives/rf-front-end-09-rtlsdr-streaming-gc-churn/' | relative_url }})
takes apart the streaming layer and the GC-churn bug that was shedding a quarter
of the control channel's live IQ.

## FAQ

**Why does the R828D default to 16 MHz if the V4 needs 28.8?**
Because a *generic* R828D really does run from a separate 16 MHz crystal — that
default is correct for the common case. The Blog V4 is the special case, and
`SetBlogV4` is how we mark it. Defaulting to 28.8 would mistune every non-V4
R828D instead.

**Why not just always chunk I2C writes at 4 bytes to dodge issue #248?**
Because that triples the round-trips for every tuner init on every dongle to work
around one firmware revision. The halving fallback only pays the cost when a chunk
actually stalls; healthy chips still write 16 bytes at a time.

**Is the shadow cache ever wrong?**
Only for the read-only status registers (0x00–0x04), which we refresh by reading
the chip. The writable shadow (0x05–0x1F) is authoritative because those
registers can't be read back anyway — the shadow *is* the source of truth, and
`Init` primes it with the init-flood values before the first burst.

## Series navigation

**Part 8 of 14** · ←
[Part 7]({{ '/blog/deep-dives/rf-front-end-07-rtlsdr-rtl2832u-bringup/' | relative_url }})
· Next →
[Part 9: RTL-SDR III — IQ streaming & the GC-churn bug]({{ '/blog/deep-dives/rf-front-end-09-rtlsdr-streaming-gc-churn/' | relative_url }})
