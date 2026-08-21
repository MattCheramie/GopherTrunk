---
slug: power-supplies
title: Power supplies
description: Undervoltage is the classic mystery failure on single-board computers. Learn volts vs amps, why cable quality matters as much as the charger, and how to budget power for a board plus its USB peripherals.
keywords: Raspberry Pi power supply, undervoltage warning, lightning bolt icon, 5V 3A, USB cable voltage drop, powered hub, power budget, SBC power problems
level: beginner
status: full
prereq:
  - picking-a-board
---

# Power supplies

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
More "broken" SBCs are victims of **undervoltage** than of anything else. A board
needs its supply to hold a steady voltage (typically **5 V**) while delivering
enough current (**amps**) for the board *plus every USB device hanging off it* —
and a thin or long **cable** can drop the voltage below spec even when the charger
is rated correctly. Symptoms are maddeningly random: crashes, USB resets, SD
corruption, throttling. The fixes are boring and absolute: the **official or
name-brand supply**, a **short thick cable**, and a **powered hub** when peripherals
add up.
</div>

Power is the least glamorous purchase in the box and the most common cause of
"my board is flaky." This lesson gives you just enough electricity to buy correctly
and to recognise undervoltage instantly when you meet it — because in Unit 6 an
undervoltage event means dropped radio samples.

## Volts and amps, in two paragraphs

**Voltage** is the electrical pressure a supply maintains; your board is designed for
a specific one (5 V on most SBCs) and malfunctions when it sags below a threshold
(roughly 4.63 V on a Pi). **Current**, in amps, is how much flow the load draws; a
supply's amp rating is the *maximum it can deliver while holding its voltage*. The
board draws what it draws — a bigger-amp supply is never harmful, it's headroom.

The failure mode: the total draw approaches the supply's limit, or resistance in a
poor cable eats the margin, and the voltage **sags**. Digital logic misbehaves in
random ways at low voltage — which is why undervoltage symptoms look like *anything
except* a power problem.

## Why is the cable half the story?

Every centimetre of wire has resistance, and at the currents an SBC draws (2–5 A at
just 5 V), a thin charging cable drops a meaningful fraction of a volt between charger
and board. A charger rated 5 V/3 A through a bad cable delivers 4.5 V at the
connector — under spec before the board even boots. Long cables and worn connectors
make it worse.

> Rule of thumb: buy the **official supply** for your board (its cable is captive and
> correctly sized), or a name-brand supply explicitly rated for that board. Never
> power an appliance from a phone charger, a laptop USB port, or "some cable from
> the drawer."

## How do you recognise undervoltage?

Learn the signature now and save hours later:

- **Random crashes and reboots**, especially under load — the CPU's draw spikes,
  the voltage dips, the system falls over.
- **USB devices disconnecting and reconnecting** — an SDR that "vanishes" mid-decode
  is classic ([USB SDR gotchas](/learn/embedded/usb-sdr-gotchas/) returns to this).
- **SD-card corruption** — writes interrupted by brownouts.
- **A lightning-bolt icon** on a Pi's display, and warnings in the logs.

The board records it. On a Pi, ask the firmware directly:

```bash
$ vcgencmd get_throttled
throttled=0x50005
```

Any non-zero value decodes to a set of flags; bit 0 (`0x1`) means undervoltage *right
now*, and bit 16 (`0x10000`) means it has occurred since boot. The kernel log tells
the same story:

```bash
$ dmesg | grep -i voltage
[ 1234.567890] hwmon hwmon1: Undervoltage detected!
```

If you see these, stop debugging software. It's the power.

## How do you budget for peripherals?

The supply powers the whole system: board, storage, and **every unpowered USB
device**. Sketch a budget:

| Load | Rough draw at 5 V |
|------|-------------------|
| Board, idle | 0.5–1 A |
| Board, CPU fully loaded | 1.5–2.5 A (more on faster boards) |
| RTL-SDR dongle | ~0.3 A |
| USB SSD | 0.5–1 A, with spin-up spikes |
| Fan | ~0.1 A |

A loaded board plus SDR plus SSD can brush against a 3 A supply's ceiling. Two ways
out: a supply with more headroom (a 5 A-class supply on boards that accept it), or
moving hungry peripherals onto a **powered USB hub** — a hub with its own supply, so
peripherals stop drawing through the board at all.
[USB &amp; powered hubs](/learn/embedded/usb-and-powered-hubs/) covers when and how.

## What about the appliance angle — outages and placement?

Two habits for an always-on build. First, **expect power cuts**: the appliance must
boot back into its job unattended, which is exactly what Unit 3's
[systemd services](/learn/embedded/services-with-systemd/) guarantee, and your
filesystem should survive the cut (journalled Linux filesystems do; quality storage
from [last lesson](/learn/embedded/storage-and-sd-cards/) helps). A small UPS is a
nice-to-have, not a requirement. Second, **placement**: cheap switching supplies can
radiate RF noise — for a radio appliance, a quality supply and a bit of distance
between it and your antenna feedline are part of the RF hygiene story that
[USB SDR gotchas](/learn/embedded/usb-sdr-gotchas/) completes.

<div class="knowledge-check" data-quiz data-correct-msg="Right — voltage sag from supply or cable causes random crashes, USB resets, and corruption." markdown="0">
  <p class="knowledge-check__q">Quick check: a board crashes randomly under load and its USB SDR keeps disconnecting. What's the first suspect?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">A bug in the decoder software</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Undervoltage — an inadequate supply or a thin, lossy cable</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">A corrupted OS image that needs re-flashing immediately</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Undervoltage** is the classic SBC mystery failure: the supply (or cable) lets the
  5 V rail sag, and everything downstream misbehaves randomly.
- The **cable is half the story** — thin or long cables drop voltage even behind a
  correctly rated charger. Buy the official or name-brand supply.
- Recognise the signature: random crashes, **USB resets**, SD corruption — and
  confirm with `vcgencmd get_throttled` / `dmesg` rather than guessing.
- **Budget amps for the whole system**, board plus peripherals; move hungry devices
  to a **powered hub** when the sum gets close.
- An appliance must **recover from power cuts unattended** — services at boot plus
  sound storage make an outage a non-event.

Next up: [Cases &amp; cooling](/learn/embedded/cases-and-cooling/).
