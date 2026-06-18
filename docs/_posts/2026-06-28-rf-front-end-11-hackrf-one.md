---
title: "RF Front End, Part 11: HackRF One — Signed 8-Bit IQ End to End"
description: The HackRF One driver end to end — the simplest IQ format of the three, signed 8-bit interleaved I,Q scaled to complex64; enumeration across One/Jawbreaker/Rad1o PIDs; and the open-time board-ID readback that fixed empty probe gains by establishing identity and the gain ladder before streaming.
category: deep-dives
tags: [sdr, go, hackrf, usb, software-design]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "RF Front End"
series_part: 11
---

*Part 11 of **RF Front End**. After the Airspy's host-side DSP, the
[HackRF One]({{ '/reference/hackrf/' | relative_url }}) is a palate cleanser: the
simplest wire format of the three. The interesting part isn't the samples — it's
identity. The driver guesses the model from a USB PID, then asks the firmware who
it really is, and that open-time handshake is where a probe-gains bug lived.*

## In this post

- Decoding the HackRF's **signed 8-bit interleaved IQ** into `complex64` in `[-1, 1]`.
- Enumeration across the **One / Jawbreaker / Rad1o** PIDs on VID `0x1d50`, and
  why the **board-ID read at open** takes precedence over the PID guess.
- The **SET_TRANSCEIVER_MODE** state machine (0 off / 1 receive) and the default
  gain ladder (LNA 16 dB, VGA 20 dB).
- The bug: `sdr list --probe` showed **empty gains** because we read them before
  the device was fully open — and the open-time fix the tests pin down.

## What the HackRF driver does

The [HackRF]({{ '/reference/hackrf/' | relative_url }}) One is a half-duplex
transceiver from Great Scott Gadgets covering 1 MHz–6 GHz. GopherTrunk only
receives, so the driver's surface is small: enumerate, claim, tune, set rate and
gain, flip into receive mode, and reap bulk-IN URBs into `complex64`. Like every
other backend it speaks the vendor protocol — here libhackrf's — directly over
the shared pure-Go USB transport, so no CGO and no libhackrf land in the build.

Its sample format is the friendliest of the bunch. The firmware delivers
**signed 8-bit interleaved IQ** — `I, Q, I, Q, …` — on bulk endpoint `0x81` once
the transceiver is in receive mode. There's no real-to-complex synthesis (that
was the Airspy R2/Mini in
[Part 10]({{ '/blog/deep-dives/rf-front-end-10-airspy-real-to-complex/' | relative_url }}))
and no 16-bit unpacking; just two signed bytes per complex sample.

## How GopherTrunk implements it in Go

### Decoding the samples

The decode is a tight loop over `internal/sdr/hackrf/hackrf.go`: read a signed
byte for I, one for Q, normalize each to roughly `[-1, 1]` by dividing by 128:

```go
// internal/sdr/hackrf/hackrf.go
// decodeInt8IQ converts a HackRF bulk-IN payload (signed 8-bit
// interleaved IQ) into normalised complex64 samples in [-1, 1].
func decodeInt8IQ(buf []byte) []complex64 {
    n := len(buf) / 2
    out := make([]complex64, n)
    for i := 0; i < n; i++ {
        iv := float32(int8(buf[2*i])) / 128
        qv := float32(int8(buf[2*i+1])) / 128
        out[i] = complex(iv, qv)
    }
    return out
}
```

The `int8` conversion is load-bearing: the bytes arrive as `uint8` in the slice,
and reinterpreting them as `int8` is what makes `0x80` read as `-128` rather than
`+128`. The in-package test pins that down with a `(-128, +64)` sample expected
near `(-1, +0.5)`.

### The transceiver state machine

The HackRF won't stream until you tell it to receive, and you must put it back to
off when you stop. That's a two-value state machine over `SET_TRANSCEIVER_MODE`
(`reqSetTransceiverMode`):

```go
// internal/sdr/hackrf/hackrf.go
const (
    transceiverModeOff     uint16 = 0
    transceiverModeReceive uint16 = 1
)

func (d *Device) setMode(mode uint16) error {
    return d.t.ControlOut(reqSetTransceiverMode, mode, 0, nil, controlTimeoutMs)
}
```

`StreamIQ` flips to receive, starts the bulk-IN reaper, and a detached goroutine
flips back to off on either context cancellation or an unrecoverable URB death —
the same teardown shape every driver in this series shares.

### Tuning and the tracking baseband filter

`SetCenterFreq` is the one place the HackRF's wire encoding is slightly unusual:
libhackrf splits the frequency into an MHz part and an Hz remainder, two
little-endian `uint32`s in the data stage:

```go
// internal/sdr/hackrf/hackrf.go
payload := make([]byte, 8)
binary.LittleEndian.PutUint32(payload[0:4], hz/1_000_000)
binary.LittleEndian.PutUint32(payload[4:8], hz%1_000_000)
return d.t.ControlOut(reqSetFreq, 0, 0, payload, controlTimeoutMs)
```

`SetSampleRate` does a little extra work: the rate goes out as a
numerator/divider pair (the driver always uses divider 1, so the host sees
exactly the requested rate), and then the **baseband filter cutoff tracks the
rate** — set to ~75 % of it to keep the passband flat while rejecting alias
energy near the band edge. The filter program is fire-and-forget; the rate
program is the one that must land:

```go
// internal/sdr/hackrf/hackrf.go  (shape)
// rate = (hz, divider 1); then baseband filter ≈ 0.75 × rate
bw := uint32(float64(hz) * 0.75)
_ = d.t.ControlOut(reqBasebandFilterBwSet,
    uint16(bw&0xFFFF), uint16(bw>>16), nil, controlTimeoutMs)
```

The `TestSetSampleRateProgramsFilter` test scripts both transfers, asserting the
filter program follows the rate program with the 0.75 cutoff.

### The gain ladder

The HackRF has no true AGC. Its gain is three independent stages: an RF amp
(on/off), an LNA (0–40 dB in 8 dB steps), and a VGA (0–62 dB in 2 dB steps). The
driver maps a single tenth-dB target onto that triple, and maps "auto" (a
negative value) onto a sane fixed preset rather than a non-existent AGC:

```go
// internal/sdr/hackrf/hackrf.go
// splitGain maps an SDR-interface tenth-dB target to the HackRF's
// (LNA, VGA, AMP-on) triple. LNA must be a multiple of 8; VGA a multiple of 2.
func splitGain(tenthDB int) (lna, vga int, amp bool) {
    if tenthDB < 0 {
        return defaultLNAGainDB, defaultVGAGainDB, false // 16 dB LNA, 20 dB VGA
    }
    target := tenthDB / 10
    lna = (target / 8) * 8
    if lna > 40 {
        lna = 40
    }
    rem := target - lna
    vga = (rem / 2) * 2
    if vga > 62 {
        vga = 62
    }
    return lna, vga, false
}
```

The default preset — **LNA 16 dB, VGA 20 dB, amp off** — is the
hardware-friendly mid-band split the test ladder asserts on (`splitGain(-1) ==
(16, 20, false)`, and rungs like `300 → (24, 6)`).

### Identity: PID guess, then ask the board

Three HackRF variants share VID `0x1d50` under different PIDs — **One** (`0x6089`),
the **Jawbreaker** prototype (`0x604b`), and the **Rad1o** badge (`0xcc15`). Unlike
the Airspy (one PID) or RTL-SDR, enumeration here has to **scan every known PID**
and concatenate the results into one descriptor list:

```go
// internal/sdr/hackrf/hackrf.go  (shape)
descs := make([]usb.Descriptor, 0)
for _, pid := range []uint16{pidHackRFOne, pidHackRFJawbrk, pidHackRFRad1o} {
    found, err := d.enum.List(vidHackRF, pid)
    // ...append found to descs...
}
```

Enumeration then maps each PID to a canonical product name, since USB descriptor
strings vary by vendor and firmware while the PID is stable:

```go
// internal/sdr/hackrf/hackrf.go
var pidProductNames = map[uint16]string{
    pidHackRFOne:    "HackRF One",
    pidHackRFJawbrk: "HackRF Jawbreaker",
    pidHackRFRad1o:  "Rad1o",
}
```

But the PID is only a *guess* about what the box is — a unit flashed with the
wrong firmware will enumerate under one PID while actually running another
board's image. So at **open** time the driver asks the firmware directly, via
`BOARD_ID_READ`, and that answer **takes precedence** over the PID:

```go
// internal/sdr/hackrf/hackrf.go
product := productForPID(desc.PID, desc.Product)
if bid, err := readBoardID(t); err == nil {
    if name, ok := boardIDNames[bid]; ok && name != "" {
        product = name // firmware self-report wins over the PID guess
    }
}
```

`boardIDNames` mirrors libhackrf's `hackrf_board_id` enum (`2` → "HackRF One",
`3` → "Rad1o", …). The same open also reads `VERSION_STRING_READ` to suffix the
tuner name with firmware (and to tag PortaPack/Mayhem builds). Both readbacks are
best-effort: older firmware that doesn't implement them just falls through to the
PID-derived name and a plain tuner string — pinned by
`TestReadVersionStringIgnoredOnError` in `hackrf_test.go`.

## The problem we hit: `--probe` reported empty gains

**Symptom.** `sdr list --probe` — which actually opens each device and reports
`dev.Info()` — listed HackRFs (and Airspys) with an **empty gain list**. The
non-probing `sdr list`, which only enumerates, showed the ladder fine. Same
hardware, two different answers, depending on whether the device had been opened.

**Root cause.** The gain ladder was attached to the `sdr.Info` produced by
`Enumerate`, but the `Info` carried by an **opened** `Device` was built
separately at open time and never got the `Gains` field. So the moment a probe
opened the device and read `dev.Info().Gains`, it got back `nil`. The information
existed; it just wasn't established on the device-side `Info` that `--probe`
reads. (The same gap bit the Airspy driver, and is called out in #454's
probe-gains fix.)

**The Go fix.** Establish the full identity — board-ID, version string, **and**
the gain ladder — at open time, on the same `Info` the opened device hands back.
The ladder is shared between `Enumerate` and `Open` so a probed device reports
exactly what an enumerated one does:

```go
// internal/sdr/hackrf/hackrf.go
// gainPresetsTenthDB ... Shared by Enumerate and Open so a probed
// (opened) device reports the same ladder an enumerated one does —
// otherwise `--probe` showed an empty list.
var gainPresetsTenthDB = []int{0, 80, 160, 240, 320, 400, 480, 560}

return &Device{
    t: t,
    info: sdr.Info{
        // ...board-ID-resolved Product, fw-suffixed TunerName...
        // Carry the gain ladder onto the opened device so dev.Info()
        // (used by `sdr list --probe`) reports it, not an empty list.
        Gains: gainPresetsTenthDB,
    },
}, nil
```

The fix is pinned by a test in `hackrf_test.go` that opens a device through a
scripted mock and asserts the opened `Info` carries the ladder:

```go
// internal/sdr/hackrf/hackrf_test.go
// The opened device must carry the gain ladder so `sdr list --probe`
// (which reads dev.Info() post-open) doesn't report an empty list.
if got := dev.Info().Gains; len(got) == 0 {
    t.Errorf("opened device Info().Gains is empty; want the gain ladder")
}
```

That same test (`TestDriverEnumerateAndOpen`) scripts the `BOARD_ID_READ` and
`VERSION_STRING_READ` exchanges the open now issues — proof that the board-id and
version readbacks, the firmware suffix, and the gain ladder all get established as
one atomic open-time step rather than lazily on first use.

It's worth dwelling on why this class of bug is easy to introduce. `Enumerate`
and `Open` each build an `sdr.Info` independently — they have to, because an
opened device knows things an enumerated one can't (the board-ID, the firmware
string). The trap is that the two `Info` constructions drift: a field added to
one is forgotten on the other, and nothing in the type system flags it. The fix
isn't just "stamp the ladder on at open" — it's making the *shared* facts (the
`gainPresetsTenthDB` ladder, the canonical product name via `productForPID`) come
from one source that both paths read, so they can't disagree. The open-only facts
(board-ID override, firmware suffix) layer on top.

## The design principle: identity by capability, established at open

The thread tying the decode loop to the probe-gains fix is one principle:
**a device's identity and capabilities are established when it's opened, by
asking the hardware — not guessed lazily and not deferred until first use.**

### How that principle shaped the Go code

- **The board reports itself; the PID is a fallback.** `BOARD_ID_READ` at open
  overrides the PID-derived name, so a mis-flashed unit reports what's actually
  running. The PID guess only survives when the firmware can't answer.
- **Open is the single point of truth for `Info`.** Product, tuner/firmware
  string, and the gain ladder are all stamped onto the device's `Info` in `Open`.
  Nothing downstream has to re-derive or re-probe them — `dev.Info()` is complete
  the instant `Open` returns, which is exactly what `--probe` depends on.
- **Best-effort readbacks degrade, they don't fail.** A board-ID or
  version-string NAK on old firmware is swallowed; the device still opens with a
  sensible PID-derived identity. Capability discovery never blocks bring-up.
- **The mock scripts the open handshake.** Because identity is established through
  ordinary control transfers, the tests drive the whole open — board-ID, version,
  gain ladder — against a `MockTransport` with no hardware, so the
  identity-by-capability contract is regression-tested.

## Where this goes next

That's all four wire formats and all four open-time identity strategies covered:
RTL-SDR, Airspy R2/Mini, Airspy HF+, and HackRF. The remaining question is what
sits *above* the drivers — the pool that owns opened devices, dispatches retunes,
and survives a USB unplug-and-replug mid-stream. That's
[Part 12]({{ '/blog/deep-dives/rf-front-end-12-sdr-pool-usb-watchdog/' | relative_url }}):
the SDR pool and the USB hotplug watchdog.

## FAQ

**Why divide the 8-bit samples by 128 and not 127?**
Dividing by 128 maps the full signed range cleanly: `-128 → -1.0` exactly, and
`+127 → ~0.992`. It keeps the scale a power of two and guarantees the result
never exceeds `[-1, 1)`, which downstream DSP assumes.

**Why does board-ID override the PID if both are available?**
The PID is what the device *enumerated as*; the board-ID is what the firmware
*reports it is*. A unit flashed with another board's image enumerates under one
PID but runs another board — the firmware's self-report is the ground truth, so it
wins.

**Why did probe show empty gains but plain list didn't?**
`sdr list` only enumerates, reading the `Info` from `Enumerate` (which had the
ladder). `sdr list --probe` opens each device and reads `dev.Info()`, which was
built separately at open and lacked the `Gains` field until we stamped the ladder
on at open time.

**Does the HackRF have AGC?**
No. It has three manual stages (amp, LNA, VGA). The driver maps a negative
("auto") target to a fixed hardware-friendly preset — amp off, LNA 16 dB, VGA
20 dB — rather than pretending an AGC exists.

**Why scan multiple PIDs at enumerate instead of one?**
Because the HackRF family ships under three PIDs on the same VID. A single
`List(vid, pid)` would miss Jawbreakers and Rad1os, so enumeration loops over all
three known PIDs and concatenates the descriptors into one ordered list that
`Open` later indexes into.

**What's the bias-tee for, and is it on by default?**
`SetBiasTee` toggles the +3.3 V antenna-port bias (`ANTENNA_ENABLE`) for powering
an external LNA up the coax. It's off unless explicitly enabled — the test
round-trips it on then off — so you never accidentally feed DC into a passive
antenna.

## Series navigation

**Part 11 of 14** · ←
[Part 10]({{ '/blog/deep-dives/rf-front-end-10-airspy-real-to-complex/' | relative_url }})
· Next →
[Part 12: The SDR pool & USB hotplug watchdog]({{ '/blog/deep-dives/rf-front-end-12-sdr-pool-usb-watchdog/' | relative_url }})
