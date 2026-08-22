---
slug: usb-and-powered-hubs
title: USB & powered hubs
description: USB is how most real peripherals reach an SBC — bus power budgets, lsusb and dmesg for diagnosis, why flaky USB is usually a power story, and when a powered hub fixes everything.
keywords: usb power budget, powered usb hub, lsusb, dmesg usb, usb device reset, bus powered, sbc usb problems, usb bandwidth
level: beginner
status: full
prereq:
  - power-supplies
---

# USB & powered hubs

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
GPIO is charming, but **USB is how real peripherals arrive** — SDRs, storage,
sound cards. Every unpowered device draws its current *through the board*, from a
shared **bus power budget**, and exceeding it produces the classic flaky-USB
symptoms: devices resetting, vanishing, or "working until they don't."
**`lsusb`** shows what's attached, **`dmesg`** narrates connects, resets, and
over-current events. The cure for a crowded bus is a **powered hub** — its own
supply feeds the peripherals, the board only carries data. On SBCs, **flaky USB is
a power story until proven otherwise**.
</div>

The appliance's most important peripheral — the SDR — arrives over USB, so this
lesson builds the mental model and the two-command diagnostic habit that Unit 6's
[USB SDR gotchas](/learn/embedded/usb-sdr-gotchas/) will lean on hard.

## Where does a USB device's power come from?

A USB port supplies both **data and power**. On a bus-powered device (no supply of
its own — an RTL-SDR, a flash stick, most sound cards), every milliamp flows from
the board's 5 V rail, through the board's circuitry, out the port. Consequences:

- The board's **power supply carries the total** — board plus all bus-powered
  devices, as [Power supplies](/learn/embedded/power-supplies/) budgeted.
- The **ports share a budget** — boards limit total USB current (Pis around
  1.2 A across all ports), regardless of how many ports exist.
- **Marginal is worse than broken.** A device just inside the budget works — until
  the CPU spikes, the SDR's draw peaks, and the voltage dips for a millisecond.
  The device resets. Rarely. Intermittently. Maddeningly.

That last mode is why experienced SBC hands treat any intermittent USB fault as a
power problem first: it's the diagnosis that's cheap to test and most often right.

## How do you see what's on the bus?

Two commands, worth making reflexes. **`lsusb`** lists devices:

```bash
$ lsusb
Bus 002 Device 003: ID 0bda:2838 Realtek Semiconductor Corp. RTL2838 DVB-T
Bus 001 Device 002: ID 2109:3431 VIA Labs, Inc. Hub
```

That Realtek `0bda:2838` line is an RTL-SDR as the system sees it — if it's absent,
no software will find the radio either. Add `-t` for the topology tree (which
devices hang off which hub and at which speed).

**`dmesg`** narrates events — plug something in and watch (`dmesg -w`):

```bash
$ dmesg | tail
[ 8123.4] usb 2-1: new high-speed USB device number 5 using xhci_hcd
[ 8123.6] usb 2-1: Product: RTL2838UHIDIR
[ 9451.2] usb 2-1: reset high-speed USB device number 5
[ 9460.9] usb usb2-port1: over-current condition
```

Those last two lines are the smoking guns of this lesson: a **reset** mid-operation
(the device dropped and re-enumerated — a streaming SDR loses samples every time)
and an explicit **over-current** report. Either one says: power, not software.

## What does a powered hub change?

A **powered hub** has its own wall supply. Downstream devices draw from the hub's
supply; the board's port carries only data. One purchase converts "the SDR resets
when the SSD spins up" into two devices that never compete. Buying notes:

- **Match the generation to the need** — a USB 3 hub for USB 3 throughput; a good
  USB 2 hub is fine for a single RTL-SDR.
- **A real supply,** comfortably rated for the sum of downstream devices — a hub
  with a feeble adapter just relocates the shortage.
- **Name brand.** Cheap hubs add their own resets and noise; the hub is
  infrastructure, not the place to save four dollars.
- **One caution:** some boards can *back-power* through a hub (current flowing the
  wrong way into the board) — name-brand hubs behave; bargain ones sometimes don't.

**Bandwidth** is the budget that a hub does *not* add. All devices on one hub —
and sometimes several of the board's own ports — share one upstream link's
throughput ([Picking a board](/learn/embedded/picking-a-board/)'s controller
topology). Power problems reset devices; bandwidth problems drop data with the
device still attached. Keep a streaming SDR on the least-shared port the board has,
and put bursty devices (storage) elsewhere.

> Rule of thumb: one bus-powered SDR straight into the board is fine on a good
> supply. The day you add a second hungry device — SSD, second dongle, sound card
> — is the day you add a powered hub.

## What's the appliance-shaped takeaway?

For the scanner build: the RTL-SDR plugs **directly into the board** (fewest parts,
least noise) on a quality main supply; a powered hub enters the design when the
peripheral count grows; and the first response to any "SDR disappeared" or
"samples dropping" report is `lsusb` + `dmesg`, not a config file. You now have
the eyes to tell a power fault from a bandwidth fault from a software fault —
which is most of USB debugging.

<div class="knowledge-check" data-quiz data-correct-msg="Right — resets and over-current lines in dmesg point at power, and a powered hub or better supply is the fix." markdown="0">
  <p class="knowledge-check__q">Quick check: a USB SDR keeps "vanishing" under load and dmesg shows reset and over-current lines. The likely fix is…</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">reinstalling the SDR software, since drivers cause resets</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">a powered hub or stronger supply — the bus power budget is being exceeded</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">a longer USB extension cable to reduce interference</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- Bus-powered USB devices draw through the board from a **shared power budget**;
  exceeding it causes intermittent **resets**, the classic flaky-USB signature.
- **`lsusb`** shows what the system sees; **`dmesg`** narrates connects, **resets,
  and over-current** events — the two-command diagnostic habit.
- A **powered hub** feeds peripherals from its own supply — the cure for crowded
  buses; buy name-brand with a real adapter.
- **Bandwidth** is a separate budget a hub doesn't add — keep the streaming SDR on
  the least-shared port.
- On SBCs, **flaky USB is a power story until proven otherwise**.

Next up: [Serial, I2C &amp; SPI](/learn/embedded/serial-i2c-spi/).
