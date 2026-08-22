---
slug: usb-sdr-gotchas
title: USB SDR gotchas
description: Dropped samples, undervoltage resets, and interference from the Pi itself — the SBC-specific SDR failure modes, how to tell them apart from the shell, and the fixes for each.
keywords: rtl-sdr dropped samples, usb sdr problems raspberry pi, undervoltage sdr, pi rf interference, usb cable quality sdr, dongle reset, emi noise sbc
level: advanced
status: full
prereq:
  - connecting-an-sdr
  - power-supplies
gophertrunk_links:
  - title: Hardware guide
    url: /hardware.html
    note: dongles, cables, and accessories that avoid these failure modes.
---

# USB SDR gotchas

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
A USB SDR on an SBC has three signature failure modes, each with a distinct
fingerprint. **Undervoltage** resets the dongle — `dmesg` resets +
`get_throttled` flags; fix the supply. **USB bandwidth/CPU starvation** drops
samples with the dongle still attached — daemon overrun warnings; fix bus
contention or [tuning](/learn/embedded/tuning-for-small-cpus/). And the sneaky
one: the **board's own RF emissions** — the Pi, its supply, and cheap cables
radiate hash that lands in your passband; fix with a **good shielded USB cable**,
**antenna distance** from the board, and clean power. Diagnose by fingerprint,
not by folklore.
</div>

The scanner decodes and the CPU budget balances. This lesson is the field guide
to the ways SBC + SDR still goes wrong — three failure modes that look similar
from the armchair ("reception is bad, sometimes") but leave different evidence
and take different fixes.

## Gotcha 1: the dongle keeps resetting — power

The RTL-SDR draws its ~300 mA through the board
([USB &amp; powered hubs](/learn/embedded/usb-and-powered-hubs/)), so a marginal
supply shows up first at the dongle: mid-decode **resets**, the device
re-enumerating, the daemon logging the SDR vanished and returned — perhaps only
during CPU-load peaks, which sag the rail hardest.

**Fingerprint:**

```bash
$ dmesg | grep -iE 'reset|over-current'
usb 1-1.2: reset high-speed USB device number 4 using xhci_hcd
$ vcgencmd get_throttled
throttled=0x50005          # undervoltage now AND since boot
```

**Fix:** the [power-supply ladder](/learn/embedded/power-supplies/) — proper
supply, short thick cable, powered hub if peripherals stack up. This gotcha is
the cheapest to rule out and the most common in the wild; check it before
anything clever.

## Gotcha 2: samples drop but nothing resets — bandwidth or CPU

The dongle stays attached, no `dmesg` drama — but decodes glitch and the daemon
warns it **can't keep up**, or the driver reports **dropped/overrun chunks**.
Something between dongle and DSP isn't draining the stream in time. Two
sub-suspects:

- **USB contention** — the SDR shares its controller with a busy device
  ([Picking a board](/learn/embedded/picking-a-board/)'s topology gotcha). A
  storage stick's write bursts on the shared bus are classic. Check `lsusb -t`
  for who shares the SDR's bus; move the SDR to the least-shared port, bursty
  devices elsewhere.
- **CPU starvation** — the busy hour outran the budget; the
  [previous lesson](/learn/embedded/tuning-for-small-cpus/) is the whole
  treatment. (And rule out [throttling](/learn/embedded/thermal-throttling/)
  shrinking the CPU underneath you — one `get_throttled` covers both gotchas'
  flags.)

**Telling 1 from 2 in one line:** resets and re-enumeration = power; clean bus
with overrun warnings = bandwidth/CPU. `rtl_test` on an otherwise idle system is
the referee — losses even at idle point at power/USB; losses only under decode
load point at CPU.

## Gotcha 3: reception is mysteriously poor — the board is jamming itself

The subtlest one, invisible to every shell command: **the SBC is a radio
transmitter**. Not intentionally — but its SoC clocks, power regulators, HDMI,
Ethernet, and above all **cheap switching supplies and unshielded USB cables**
radiate broadband hash and sharp harmonic spurs. Your antenna is connected to
the most sensitive receiver in the house, sitting centimetres from all of it.
The result: a raised [noise floor](/learn/rf-sdr/noise-and-snr/) or spurs in the
passband, weak talkgroups that decode poorly or not at all — *worse than the
same dongle on a distant antenna*, with nothing wrong in any log.

**Fingerprint:** it's an RF diagnosis. In the web console's spectrum view (or
any waterfall), compare the noise floor with the antenna **on the desk beside
the Pi** versus **metres away on coax** — a floor that drops when the antenna
moves away is the board's own hash. Spurs that march in lockstep with board
activity (Ethernet plugged/unplugged, CPU load) are its harmonics.

**Fixes, in order of value:**

- **Antenna distance.** The single biggest lever: metres of coax between board
  and antenna, ideally the antenna high and the electronics low. Free signal
  quality ([antennas](/learn/rf-sdr/antennas/) covers the rest of that story).
- **A good shielded USB cable** (or direct connection) — the classic offender is
  the flimsy unshielded extension acting as an antenna *for the board's noise*,
  coupling it straight into the dongle.
- **Clean power** — quality supply, and distance between any switching wart and
  the coax run.
- **Shielding and ferrites** — a metal case ([Cases &amp; cooling](/learn/embedded/cases-and-cooling/))
  and clip-on ferrite chokes on USB/power leads mop up the residue.

> Rule of thumb: fingerprints first — `dmesg` + `get_throttled` (power), daemon
> overruns + `lsusb -t` (bandwidth/CPU), then the antenna-distance A/B (RF). The
> three fixes don't overlap, so a wrong guess fixes nothing.

## The half-hour triage, all together

When "the scanner got flaky," run the ladder: `get_throttled` → `dmesg` resets →
daemon overrun warnings → `lsusb -t` topology → antenna-distance A/B. Each rung
is minutes, each eliminates one failure mode, and the order runs cheapest-first.
Bake the first three into your [monitoring](/learn/embedded/monitoring-your-board/)
so the flaky *hour* comes with its evidence attached — the difference between
debugging and archaeology.

<div class="knowledge-check" data-quiz data-correct-msg="Right — resets say power; a stable bus with overruns says bandwidth or CPU; a floor that drops with antenna distance says self-interference." markdown="0">
  <p class="knowledge-check__q">Quick check: decodes glitch under load, but dmesg shows no USB resets and get_throttled is 0x0. Which gotcha is most likely?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Undervoltage — the supply must be replaced first</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">USB contention or CPU starvation — the stream isn't being drained in time</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">The dongle has worn out and needs replacing</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- Three SBC-specific SDR failure modes, three fingerprints: **undervoltage**
  (dmesg resets + throttled flags), **bandwidth/CPU** (overrun warnings on a
  stable bus), **self-interference** (noise floor that drops with antenna
  distance).
- Power is the **most common and cheapest to rule out** — always first.
- Bandwidth vs CPU splits with `lsusb -t` and whether `rtl_test` is clean at
  idle.
- The board **jams itself**: fix with **antenna distance**, a **shielded USB
  cable**, clean power, shielding and ferrites.
- Run the **cheapest-first triage ladder**, and let monitoring capture the
  evidence while you sleep.

Next up: [Appliance networking &amp; access](/learn/embedded/appliance-networking/).
