---
title: "RF Front End, Part 7: RTL-SDR I — Bringing Up the RTL2832U"
description: How GopherTrunk brings up the RTL2832U demodulator in pure Go — the order-sensitive init sequence captured verbatim from librtlsdr, the I2C bridge to the tuner, GPIO and bias-tee, and the USB register transport underneath it all.
category: deep-dives
tags: [sdr, go, rtl-sdr, rtl2832u, software-design]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "RF Front End"
series_part: 7
---

*Part 7 of **RF Front End**. We've spent the last three posts building a pure-Go
USB transport. Now we use it: the first chip a sample passes through on its way
out of an [RTL-SDR]({{ '/reference/rtl-sdr/' | relative_url }}) is the RTL2832U
demodulator, and bringing it up means reproducing a register sequence where the
*order* is load-bearing — get it wrong and the chip comes up half-initialized
and deaf.*

> **TL;DR** — The RTL2832U is a DVB-T demodulator pressed into service as a raw
> ADC, brought up in pure Go via USB control transfers. The headline trap: its
> init is an order-sensitive state machine, so any reordered, missing, or
> uncommitted write leaves the chip present but deaf. The fix is to reproduce
> librtlsdr's sequence byte-for-byte and pin it with golden tests.

## In this post

- **What the RTL2832U is** — a DVB-T demodulator repurposed as a raw ADC, and the
  four jobs the bring-up layer has to do to it.
- **The init sequence** — `InitBaseband()` and the exact, order-sensitive
  register flood captured from librtlsdr, where the FIR upload sits *between* the
  demod reset and SDR-mode enable.
- **The I2C bridge, GPIO, and the USB register transport** that the tuner driver
  (Part 8) and the streaming layer (Part 9) build on.
- **The problem we hit:** reproducing that sequence faithfully — any reordering
  or missing write leaves the chip silent — and how we captured and verified it.

## What the RTL2832U does

The RTL2832U was designed to demodulate DVB-T television. The SDR community
discovered it has a debug mode that dumps the raw 8-bit IQ stream straight off
its ADC over USB — bypassing the television demodulator entirely. That's the
mode GopherTrunk uses: the RTL2832U becomes a 28.8 MHz-clocked ADC with a USB
FIFO bolted on, and *all* the actual tuning happens in a separate tuner chip
(the R820T/R828D, covered in
[Part 8]({{ '/blog/deep-dives/rf-front-end-08-rtlsdr-r82xx-blog-v4/' | relative_url }}))
that the RTL2832U only talks to as an I2C master.

So the bring-up layer has four distinct jobs, and they map cleanly onto four
files in `internal/sdr/rtlsdr/rtl2832u`:

- **Register transport** (`transport.go`, `regs.go`) — turn "write register X in
  block Y" into the right USB control transfer, byte-for-byte what librtlsdr
  sends.
- **Baseband init** (`demod.go`) — the one-time flood of register writes that
  switches the chip out of DVB-T mode and into raw-IQ-over-USB mode.
- **The I2C bridge** (`i2c.go`) — so the tuner driver can reach its chip through
  the same USB control endpoint.
- **GPIO** (`gpio.go`) — bias-tee power, tuner reset pulses, and the V4's input
  switching.

This post is the bring-up; the tuner that rides on top of the I2C bridge is
Part 8, and sustained streaming through the USB FIFO is Part 9.

<figure class="lab-figure">
<svg viewBox="0 0 660 150" width="660" height="150" role="img" aria-label="The RTL-SDR signal path: the antenna feeds the R82xx tuner, which mixes RF down to a 3.57 MHz IF and hands it to the RTL2832U acting as a raw 8-bit ADC clocked at 28.8 MHz; the RTL2832U streams unsigned IQ out the USB bulk-IN endpoint to the host, and separately reaches the tuner as an I2C master through its bridge repeater.">
  <rect x="150" y="14" width="210" height="28" rx="6" fill="none" stroke="var(--fg-muted)"/>
  <text x="255" y="31" text-anchor="middle" fill="var(--fg-muted)" font-size="9">I2C bridge · repeater bit (control)</text>
  <line x1="321" y1="68" x2="321" y2="42" stroke="var(--fg-muted)"/>
  <line x1="200" y1="42" x2="162" y2="66" stroke="var(--fg-muted)"/><polygon points="171,60 158,68 172,71" fill="var(--fg-muted)"/>
  <rect x="8" y="68" width="70" height="46" rx="6" fill="none" stroke="currentColor"/>
  <text x="43" y="94" text-anchor="middle" fill="currentColor" font-size="9">antenna</text>
  <line x1="78" y1="91" x2="92" y2="91" stroke="currentColor"/><polygon points="92,87 102,91 92,95" fill="currentColor"/>
  <rect x="102" y="68" width="130" height="46" rx="6" fill="none" stroke="currentColor"/>
  <text x="167" y="88" text-anchor="middle" fill="currentColor" font-size="10">R82xx tuner</text>
  <text x="167" y="102" text-anchor="middle" fill="var(--fg-muted)" font-size="8">RF → 3.57 MHz IF</text>
  <line x1="232" y1="91" x2="246" y2="91" stroke="currentColor"/><polygon points="246,87 256,91 246,95" fill="currentColor"/>
  <rect x="256" y="68" width="158" height="46" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="335" y="88" text-anchor="middle" fill="var(--accent)" font-size="10">RTL2832U</text>
  <text x="335" y="102" text-anchor="middle" fill="var(--fg-muted)" font-size="8">raw 8-bit ADC · 28.8 MHz</text>
  <line x1="414" y1="91" x2="428" y2="91" stroke="currentColor"/><polygon points="428,87 438,91 428,95" fill="currentColor"/>
  <rect x="438" y="68" width="104" height="46" rx="6" fill="none" stroke="currentColor"/>
  <text x="490" y="88" text-anchor="middle" fill="currentColor" font-size="9">USB bulk-IN</text>
  <text x="490" y="102" text-anchor="middle" fill="var(--fg-muted)" font-size="8">FIFO endpoint</text>
  <line x1="542" y1="91" x2="556" y2="91" stroke="currentColor"/><polygon points="556,87 566,91 556,95" fill="currentColor"/>
  <rect x="566" y="68" width="86" height="46" rx="6" fill="none" stroke="var(--fg-muted)"/>
  <text x="609" y="88" text-anchor="middle" fill="var(--fg-muted)" font-size="9">host</text>
  <text x="609" y="102" text-anchor="middle" fill="var(--fg-muted)" font-size="8">unsigned IQ</text>
</svg>
<figcaption>All the tuning happens in the R82xx; the RTL2832U is only a raw ADC with a USB FIFO, and the same chip reaches its tuner as an I2C master through the bridge the bring-up layer opens.</figcaption>
</figure>

## How GopherTrunk implements it in Go

### The register transport

Everything starts at the control transfer. The RTL2832U exposes seven
memory-mapped "blocks" (USB controller, system registers, the I2C bridge, …),
and a separate page-addressed *demod* register space. `regs.go` names the blocks
and addresses; `transport.go` turns a logical access into the exact vendor
control transfer librtlsdr would send:

```go
// internal/sdr/rtlsdr/rtl2832u/transport.go
func (d *Demod) writeBlockRegLocked(block uint8, addr, val uint16, n int) error {
    index := uint16(block)<<8 | 0x10
    data := encodeWriteVal(val, n)
    if err := d.t.ControlOut(0, addr, index, data, CtrlTimeoutMs); err != nil {
        return fmt.Errorf("rtl2832u: write block=%d addr=0x%04x val=0x%04x: %w", block, addr, val, err)
    }
    return nil
}
```

The encoding is fussy and we copy it exactly: 1-byte writes send `val & 0xff`;
2-byte writes send **big-endian** (high byte first); reads come back
little-endian. `encodeWriteVal` panics on anything but `n ∈ {1, 2}` because the
package never has cause to issue any other width — a programmer error, not a
runtime one.

The demod register space has its own wrinkle that turns out to matter a great
deal. Every demod write must be followed by a "commit" read:

```go
// internal/sdr/rtlsdr/rtl2832u/transport.go
func (d *Demod) writeDemodRegLocked(page uint8, addr, val uint16, n int) error {
    wValue := (addr << 8) | 0x20
    wIndex := uint16(0x10) | uint16(page)
    data := encodeWriteVal(val, n)
    if err := d.t.ControlOut(0, wValue, wIndex, data, CtrlTimeoutMs); err != nil {
        return fmt.Errorf("rtl2832u: write demod page=%d addr=0x%02x val=0x%04x: %w", page, addr, val, err)
    }
    // Commit. Required by the RTL2832U register interface — without it
    // the demod write doesn't take effect. Errors here are non-fatal.
    _, _ = d.readDemodRegLocked(0x0A, 0x01, 1)
    return nil
}
```

**That dummy read of page `0x0A`, register `0x01` is not optional. Without it the
chip does not latch the previous write — the value you think you set never lands.**
This is the kind of detail you cannot derive from a datasheet; it lives in the C
library because someone discovered it on real silicon, and we keep it bit for
bit. Every `Demod` access also runs under a single mutex, because the tuner
driver, the bias-tee toggler, and the streaming setup can all issue control
transfers concurrently and they must interleave cleanly on the one USB wire.

### The init sequence

The actual bring-up is `InitBaseband()`. It walks a fixed table of register
writes — `initBasebandSteps` — captured verbatim from librtlsdr's
`rtlsdr_init_baseband`. A flavor of it:

```go
// internal/sdr/rtlsdr/rtl2832u/demod.go
var initBasebandSteps = []initBasebandStep{
    // USB init
    {block: BlockUSB, addr: USBSysctl, val: 0x09, n: 1},
    {block: BlockUSB, addr: USBEpaMaxpkt, val: 0x0002, n: 2},
    {block: BlockUSB, addr: USBEpaCtl, val: 0x1002, n: 2},
    // Power on demod
    {block: BlockSys, addr: SysDemodCtl1, val: 0x22, n: 1},
    {block: BlockSys, addr: SysDemodCtl, val: 0xE8, n: 1},
    // Demod soft-reset (bit 3) + release
    {demod: true, page: 1, addr: 0x01, val: 0x14, n: 1},
    {demod: true, page: 1, addr: 0x01, val: 0x10, n: 1},
    // ...disable spectrum inversion, clear DDC/IF registers...
    // (FIR upload happens here, between reset and SDR-mode enable)
    // SDR mode, DAGC off (bit 5)
    {demod: true, page: 0, addr: 0x19, val: 0x05, n: 1},
    // ...FSM hold registers, disable AGC loops, PID filter, etc...
    // Zero-IF mode + DC cancellation + IQ estimation/compensation
    {demod: true, page: 1, addr: 0xB1, val: 0x1B, n: 1},
    // Disable 4.096 MHz clock output on TP_CK0
    {demod: true, page: 0, addr: 0x0D, val: 0x83, n: 1},
}
```

The structure is the point. `InitBaseband` runs the first fifteen steps, then
uploads the FIR coefficients, then runs the rest:

```go
// internal/sdr/rtlsdr/rtl2832u/demod.go
func (d *Demod) InitBaseband() error {
    d.mu.Lock()
    defer d.mu.Unlock()
    // Steps before the FIR upload.
    for i := 0; i < 15; i++ {
        if err := d.runInitStep(initBasebandSteps[i]); err != nil {
            return fmt.Errorf("init baseband step %d: %w", i, err)
        }
    }
    // Default FIR (20 single-byte writes at page 1, addr 0x1C..0x2F).
    if err := d.setFIRLocked(firDefault); err != nil {
        return fmt.Errorf("init baseband: FIR upload: %w", err)
    }
    for i := 15; i < len(initBasebandSteps); i++ {
        if err := d.runInitStep(initBasebandSteps[i]); err != nil {
            return fmt.Errorf("init baseband step %d: %w", i, err)
        }
    }
    return nil
}
```

The FIR upload sits *inside* the sequence — after the demod soft-reset (bit 3)
and the DDC/IF-register clear, but before SDR mode is enabled and zero-IF gets
turned on. That placement is what the table's comment means by "order is
load-bearing."

<figure class="lab-figure">
<svg viewBox="0 0 660 150" width="660" height="150" role="img" aria-label="The InitBaseband sequence as a timeline: steps 0 to 14 configure the USB endpoints, power the demod, soft-reset it and clear the DDC/IF registers; then the 20-tap FIR is uploaded to page 1, addresses 0x1C to 0x2F; then steps 15 to the end enable SDR mode, zero-IF, and DC and IQ compensation. The FIR upload sits inside the sequence, and every demod write must be followed by a commit read of page 0x0A register 0x01 or it never latches.">
  <text x="330" y="20" text-anchor="middle" fill="var(--fg-muted)" font-size="10">InitBaseband() — order is load-bearing</text>
  <rect x="16" y="36" width="190" height="52" rx="6" fill="none" stroke="currentColor"/>
  <text x="111" y="58" text-anchor="middle" fill="currentColor" font-size="10">steps 0–14</text>
  <text x="111" y="72" text-anchor="middle" fill="var(--fg-muted)" font-size="8">USB EP cfg · demod power</text>
  <text x="111" y="83" text-anchor="middle" fill="var(--fg-muted)" font-size="8">soft-reset · clear DDC/IF</text>
  <line x1="206" y1="62" x2="234" y2="62" stroke="currentColor"/><polygon points="234,58 244,62 234,66" fill="currentColor"/>
  <rect x="244" y="36" width="172" height="52" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="330" y="58" text-anchor="middle" fill="var(--accent)" font-size="10">FIR upload</text>
  <text x="330" y="72" text-anchor="middle" fill="var(--fg-muted)" font-size="8">20 taps · page 1</text>
  <text x="330" y="83" text-anchor="middle" fill="var(--fg-muted)" font-size="8">addr 0x1C–0x2F</text>
  <line x1="416" y1="62" x2="444" y2="62" stroke="currentColor"/><polygon points="444,58 454,62 444,66" fill="currentColor"/>
  <rect x="454" y="36" width="190" height="52" rx="6" fill="none" stroke="currentColor"/>
  <text x="549" y="58" text-anchor="middle" fill="currentColor" font-size="10">steps 15–end</text>
  <text x="549" y="72" text-anchor="middle" fill="var(--fg-muted)" font-size="8">SDR mode · zero-IF</text>
  <text x="549" y="83" text-anchor="middle" fill="var(--fg-muted)" font-size="8">DC + IQ compensation</text>
  <rect x="120" y="106" width="420" height="30" rx="6" fill="none" stroke="var(--fg-muted)" stroke-dasharray="4 3"/>
  <text x="330" y="125" text-anchor="middle" fill="var(--fg-muted)" font-size="9">every demod write → commit read (page 0x0A, reg 0x01) or it never latches</text>
</svg>
<figcaption>The FIR upload is split into the middle of the flood — after the soft-reset and DDC/IF clear, before SDR mode — which is the "order is load-bearing" boundary the <code>initBasebandSteps</code> table pins as code.</figcaption>
</figure>

The 20-tap FIR filter itself is the DAB/FM-tuned coefficient set
librtlsdr ships, packed with a fiddly layout — the first 8 taps are 8-bit signed
(one byte each), and taps 8–15 are 12-bit signed and pack three nibbles per pair
into 12 bytes:

```go
// internal/sdr/rtlsdr/rtl2832u/demod.go (shape)
for i := 0; i < 4; i++ {
    v0 := coeffs[8+i*2]
    v1 := coeffs[8+i*2+1]
    fir[8+i*3]   = byte(v0 >> 4)
    fir[8+i*3+1] = byte((v0 << 4) | ((v1 >> 8) & 0x0F))
    fir[8+i*3+2] = byte(v1)
}
```

### The I2C bridge

Once the baseband is up, the only way to reach the tuner is through the
RTL2832U's I2C bridge. `i2c.go` exposes it as a handful of methods. The bridge
has to be explicitly opened and closed around each tuner burst — librtlsdr
toggles a "repeater" bit so the demod doesn't compete with the tuner for the
bus:

```go
// internal/sdr/rtlsdr/rtl2832u/i2c.go
func (d *Demod) SetI2CRepeater(on bool) error {
    d.mu.Lock()
    defer d.mu.Unlock()
    if d.repON == on {
        return nil
    }
    val := uint16(0x10)
    if on {
        val = 0x18
    }
    if err := d.writeDemodRegLocked(1, 0x01, val, 1); err != nil {
        return fmt.Errorf("rtl2832u: SetI2CRepeater(%v): %w", on, err)
    }
    d.repON = on
    return nil
}
```

That cached `repON` looks like a harmless optimization — and it's free when the
value is unchanged — but it bites us hard on one specific dongle in Part 8, where
a *fresh* wire write of the repeater bit turns out to be load-bearing even when
the register already holds the on-value. The tuner driver issues bulk
`I2CWrite`/`I2CRead` calls through this bridge; we'll lean on them heavily next
post.

### GPIO and bias-tee

The last piece is GPIO. The RTL2832U has eight general-purpose pins in its system
block, and dongles wire the 5 V bias-tee LNA power to one of them (pin 0 on the
RTL-SDR.com v3+, pin 4 on some NESDR revisions). Driving a pin is a
read-modify-write on the direction and output-enable registers:

```go
// internal/sdr/rtlsdr/rtl2832u/gpio.go
func (d *Demod) SetBiasTee(gpio uint8, on bool) error {
    if err := d.SetGPIOOutput(gpio); err != nil {
        return fmt.Errorf("rtl2832u: SetBiasTee: configure output: %w", err)
    }
    return d.SetGPIOBit(gpio, on)
}
```

The same GPIO machinery does double duty: tuner detection (Part 8) pulses GPIO5
and GPIO4 to reset and power-enable the non-R820T tuners, and the Blog V4 drives
its upconverter relay off GPIO5. GPIO writes target the system block, not the
demod's bridge register, so they never disturb the I2C-repeater state — a
property the detection code relies on.

## The problem we hit: a half-initialized, deaf chip

**The symptom.** Early in the pure-Go rewrite, the dongle would open, claim its
interface, accept every register write without error — and then stream pure
noise. No tuner lock, no signal, no error returned anywhere. The chip was
*present* but *deaf*.

**The root cause.** The init sequence is not a set of independent register
writes. It is a state machine being walked through a specific path, and the
RTL2832U latches intermediate state at several points. Three classes of mistake
all produce the same silent-noise symptom:

- **A missing commit read.** Drop the dummy `readDemodRegLocked(0x0A, 0x01, 1)`
  after a demod write and that write silently never takes effect — including the
  zero-IF and IQ-compensation writes that the whole SDR mode depends on.
- **A reordered FIR upload.** Move the FIR write before the soft-reset, or after
  SDR mode is already enabled, and the filter loads into the wrong chip state.
- **A skipped USB-block write.** Omit one of the leading `BlockUSB` EP-config
  writes and the bulk-IN FIFO never gets sized correctly; the stream comes up but
  the framing is wrong.

None of these throw. That's what made it brutal: the failure surfaces three
layers downstream as "decoder sees noise," with nothing in between to point at
the cause.

**The fix.** We stopped treating the sequence as something to be *understood* and
started treating it as something to be *reproduced exactly*. The constants were
captured verbatim from librtlsdr's `rtlsdr_init_baseband` into the
`initBasebandSteps` table, with the order — including the FIR split — preserved
as code structure rather than prose. Then we pinned it: the `initBasebandStep`
struct exists specifically so a unit test can replay the entire sequence against
a mock USB transport and assert that the bytes leaving a `Demod` are
**byte-identical** to a real-hardware capture from the C library. The package
doc says it outright — "every control transfer that leaves a `Demod` matches what
the C library would have sent under the same call," confirmed by golden tests.

There's even a deliberate redundancy that came out of this. `WarmupUSBSysctl`
issues a single sacrificial USB write whose wire bytes are identical to
`initBasebandSteps[0]`, purely to absorb the first-control-transfer NAK some
clone dongles emit right after the interface is claimed. The caller swallows any
error it returns and proceeds straight to `InitBaseband`, exactly as librtlsdr
does — and because step 0 is byte-identical, a genuine stall re-surfaces on the
real init where the open-path reset+retry envelope can act on it.

## The design principle: faithful protocol reproduction

The RTL2832U bring-up is the cleanest example in the codebase of a principle we
reach for whenever we're talking to an opaque chip ABI: **when you don't fully
understand a protocol but you have a known-good implementation, reproduce it
faithfully and verify against it — don't improvise.**

librtlsdr is the de facto reference for the RTL2832U. It encodes years of
accumulated knowledge about this chip's undocumented behavior: the commit read,
the FIR placement, the dummy warmup write. We don't have a datasheet that
explains *why* the order matters. What we have is a C implementation that works
on millions of dongles. The correct engineering move is not to reverse-engineer
the silicon — it's to port the known-good sequence exactly and pin it with tests.

### How that principle shaped the Go code

- **The init sequence is data, not prose.** `initBasebandSteps` is a literal
  table, and `InitBaseband` is a small interpreter over it. The order lives in
  the data structure where it can't drift, and the FIR split is an explicit index
  boundary, not a comment someone might "clean up."
- **The wire format is encapsulated and panic-guarded.** `encodeWriteVal` and the
  `writeBlockReg`/`writeDemodReg` pair are the *only* place control-transfer
  encoding lives. The big-endian-write / little-endian-read asymmetry and the
  commit read are written once and reused everywhere.
- **Golden tests are the contract.** The package's job is "be byte-identical to
  librtlsdr," so the tests assert exactly that against the mock transport. A
  regression shows up as a diff in the captured byte stream, not as a mysterious
  deaf dongle in the field.
- **Quirks are documented at the point of reproduction.** Every load-bearing
  oddity — the commit read, the warmup write, the repeater cache — carries a
  comment explaining *that* it matters and *why* librtlsdr does it, so the next
  person doesn't "simplify" it away.

## Where this goes next

The RTL2832U is now in raw-IQ mode with its I2C bridge open for business — but it
can't tune anything. That job belongs to the tuner chip hanging off the bridge.
[Part 8]({{ '/blog/deep-dives/rf-front-end-08-rtlsdr-r82xx-blog-v4/' | relative_url }})
brings up the R820T/R828D ([R820T]({{ '/reference/r820t-tuner/' | relative_url }})),
walks its PLL math and shadow-register cache, and tells the headline RTL-SDR
story: why the RTL-SDR Blog V4 came up *deaf* and what the manual override and
per-band input switching had to do to fix it.

## FAQ

**Why is the RTL2832U called a "demodulator" if we use it as an ADC?**
Because that's what it was built for — DVB-T television demodulation. The SDR use
case exploits a debug mode that streams the raw ADC output over USB before the
television demodulator touches it. We disable the DVB-T path during
`InitBaseband` (that's most of what the sequence does) and take the raw samples.

**Why keep the FIR filter at all if it's "unused for SDR"?**
The DVB-T variant of the filter is unused, but the DAB/FM-tuned coefficient set
that librtlsdr ships is part of the known-good init state. We load it exactly as
the C library does rather than risk leaving the demod's decimation path in an
unexpected configuration.

**Could the commit read just be a `librtlsdr` superstition?**
We treat it as load-bearing because it is observably load-bearing on real
hardware — the comment in `writeDemodRegLocked` is blunt about it: without the
read, the demod write doesn't take effect. When a known-good implementation does
something and removing it breaks real silicon, that's not superstition, that's
the ABI.

## Series navigation

**Part 7 of 14** · ←
[Part 6]({{ '/blog/deep-dives/rf-front-end-06-usb-on-macos-windows/' | relative_url }})
· Next →
[Part 8: RTL-SDR II — the R82xx tuner & the Blog V4 deafness]({{ '/blog/deep-dives/rf-front-end-08-rtlsdr-r82xx-blog-v4/' | relative_url }})
