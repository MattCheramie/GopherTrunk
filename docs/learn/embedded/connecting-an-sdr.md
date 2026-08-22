---
slug: connecting-an-sdr
title: Connecting an SDR
description: An RTL-SDR on a Raspberry Pi is the classic pairing — install the driver tools, blacklist the TV kernel module, verify the dongle with rtl_test, and prove clean streaming from the command line.
keywords: rtl-sdr raspberry pi, rtl_test, blacklist dvb kernel module, rtl-sdr driver linux, librtlsdr, sdr dongle setup, test rtl-sdr command line
level: intermediate
status: full
prereq:
  - usb-and-powered-hubs
  - first-boot-and-ssh
gophertrunk_links:
  - title: Hardware guide
    url: /hardware.html
    note: which SDR dongles work well, and what to look for when buying.
---

# Connecting an SDR

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
An **RTL-SDR** dongle on an SBC is the classic pairing this whole module builds
toward. Setup is three moves: install the **rtl-sdr tools** (`apt install
rtl-sdr`), **blacklist the DVB-T kernel module** that would otherwise claim the
dongle as a TV tuner, and prove the stream with **`rtl_test`** — a clean minute
with **zero lost samples** is the green light. Test *before* installing any
decoder: it splits every future problem into "hardware/USB" vs "software," which
is half of SDR debugging.
</div>

Unit 4 closes by connecting the peripheral the appliance exists for. The RF theory
lives in the [RF &amp; SDR module](/learn/rf-sdr/what-is-sdr/) — this lesson is the
embedded engineer's version: get the dongle recognized, streaming, and *proven*
from a headless shell, so Unit 6 starts on solid ground.

## What is an RTL-SDR, one paragraph's worth?

A **software-defined radio** receiver digitises a slice of radio spectrum and
hands the raw samples to software, which does all tuning and decoding. The
**RTL-SDR** is the famous $30 way in: a USB dongle built around a chip designed
for European TV reception, discovered to expose its raw samples — turning it into
a general receiver covering roughly 24 MHz–1.7 GHz. It streams a few MHz of
spectrum continuously over USB — the sustained load that
[USB &amp; powered hubs](/learn/embedded/usb-and-powered-hubs/) prepared you for.
(Choosing a dongle — TCXO stability, enclosures, filtered variants — is the
[hardware guide](/hardware.html)'s territory; any current name-brand RTL-SDR
works for this lesson.)

## Does the system see it?

Plug it into the board (directly, not a hub, for this first test) and ask:

```bash
$ lsusb
Bus 001 Device 004: ID 0bda:2838 Realtek Semiconductor Corp. RTL2838 DVB-T
```

The `0bda:2838` Realtek line is the dongle. Present in `lsusb` means the hardware
and USB layers are fine — everything from here is software.

## Why must you blacklist a kernel module?

Linux helpfully recognises the chip as what it was designed to be: a **TV
tuner**. The `dvb_usb_rtl28xxu` kernel module claims the device at plug-in, and
while it holds the dongle, no SDR software can open it — the classic first-run
error (`usb_claim_interface error -6`). Tell the kernel to leave it alone:

```bash
$ sudo tee /etc/modprobe.d/blacklist-rtlsdr.conf <<'EOF'
blacklist dvb_usb_rtl28xxu
EOF
$ sudo reboot
```

Distribution packages often install this blacklist for you — but knowing the
mechanism beats trusting it, because the symptom ("device busy") says nothing
about the cause.

## How do you prove the dongle streams cleanly?

Install the userspace tools and run the acceptance test:

```bash
$ sudo apt install rtl-sdr
$ rtl_test
Found 1 device(s):
  0:  Realtek, RTL2838UHIDIR, SN: 00000001
Using device 0: Generic RTL2832U OEM
Found Rafael Micro R820T tuner
Supported gain values (29): 0.0 0.9 1.4 2.7 ...
Sampling at 2048000 S/s.
Reading samples in async mode...
```

Let it run for a couple of minutes. What you're watching for is what it *doesn't*
print: lines like `lost at least 148 bytes` report **dropped samples** — the
board failed to drain the USB stream in time. Occasional small losses in the
first seconds are normal as buffers settle; steady losses are a real problem
with exactly three usual causes: **undervoltage** (check
`vcgencmd get_throttled` — [Power supplies](/learn/embedded/power-supplies/)),
**USB contention** (what else is on the bus? — `lsusb -t`), or **CPU starvation**
(is something pinning the cores?). Unit 6's
[USB SDR gotchas](/learn/embedded/usb-sdr-gotchas/) drills each.

Two more tools complete the smoke test. `rtl_eeprom` reads the dongle's info —
and can write a **serial number**, which matters the day you run two dongles and
need software to tell them apart. And for proof the *radio* side works, tune an
FM broadcast station and pipe audio to a file:

```bash
$ rtl_fm -f 101.1M -M wbfm -s 200k -r 48k out.raw   # Ctrl-C after a few seconds
```

Copy `out.raw` to your PC (`scp scanner.local:out.raw .`) and play it (it's
headerless 48 kHz 16-bit mono). Hearing a radio station recorded by a headless
board you configured over SSH is the moment the whole build becomes real.

> Rule of thumb: never install the decoder until `rtl_test` runs clean. A
> verified-clean dongle turns every later mystery into a software question — and
> an unverified one poisons every conclusion above it.

## What about antenna placement — already?

Yes, one habit now, because it's physical: the little whip antenna that ships
with dongles is for experiments, and wherever the story ends, **keep the antenna
away from the board** — the SBC and its supply radiate exactly the kind of RF
hash that buries weak signals, and a metre of separation is free signal quality.
The full antenna story is in [Antennas](/learn/rf-sdr/antennas/); the
interference story concludes in [USB SDR gotchas](/learn/embedded/usb-sdr-gotchas/).

<div class="knowledge-check" data-quiz data-correct-msg="Right — the kernel's TV-tuner driver claims the dongle, so it must be blacklisted before SDR software can open it." markdown="0">
  <p class="knowledge-check__q">Quick check: why does a fresh Linux system stop SDR software opening an RTL-SDR?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">The dongle needs a firmware download that only Windows can perform</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">The kernel's DVB-T TV-tuner module claims the device first and must be blacklisted</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Linux blocks all USB radio devices until a license file is installed</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- An **RTL-SDR** streams a slice of spectrum continuously over USB — the
  peripheral the appliance is built around.
- Setup: `lsusb` to confirm the hardware, **blacklist `dvb_usb_rtl28xxu`**, then
  `apt install rtl-sdr`.
- **`rtl_test` is the acceptance test**: minutes of streaming with no `lost
  bytes` lines. Steady losses mean power, USB contention, or CPU — not decoder
  settings.
- `rtl_eeprom` sets **serial numbers** (essential with two dongles);
  `rtl_fm` proves end-to-end radio with your own ears.
- **Verify the dongle before installing any decoder**, and keep the antenna
  **away from the board**.

Next up: [Thermal throttling](/learn/embedded/thermal-throttling/).
