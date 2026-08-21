---
slug: hats-and-add-ons
title: HATs & add-ons
description: Stackable add-on boards that give an SBC radios, screens, relays, sensors, and power-over-Ethernet — how HATs identify themselves over I2C, how they share the header, and when to choose a HAT over USB.
keywords: raspberry pi HAT, add-on board, EEPROM identification, PoE HAT, relay board, sensor hat, stacking headers, pin conflicts
level: beginner
status: full
prereq:
  - serial-i2c-spi
---

# HATs & add-ons

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
A **HAT** (Hardware Attached on Top) is a pre-built add-on board that presses onto
the whole 40-pin header — no wiring, no breadboard — adding relays, displays,
sensors, radios, audio, or **PoE** power. A proper HAT carries a tiny **EEPROM**
the board reads over I2C at boot, so it can **identify itself** and auto-configure
drivers. HATs *share* the header's buses, so stacking two works only when their
**pin claims don't collide**. Choose a HAT for header-native hardware and tidy
integration; choose **USB** when the peripheral has a good USB version — like the
appliance's SDR.
</div>

You can now blink pins and name the buses. This lesson is the shortcut culture
built on top of that: an ecosystem of boards that snap onto the header and just
work — and the judgment call of when to use one instead of a USB device.

## What exactly is a HAT?

The Raspberry Pi world standardised the idea in 2014: a **HAT** is an add-on board
of a specified size that mates with all 40 header pins, mounts on standoffs, and —
the clever part — carries an identification **EEPROM** (a tiny memory chip) on a
reserved I2C bus. At boot, the firmware reads it and learns what's attached: the
board's name, and which pins it uses; for many HATs the right device-tree
configuration is then applied automatically. Plug in a well-made audio HAT and a
new sound device simply appears — the closest the header world gets to USB's
plug-and-play.

Plenty of add-ons skip the spec — cheaper "bonnets," bare modules, partial-header
boards. They work fine; you just configure manually what a HAT would have declared
(exactly the interface-enabling you learned [last lesson](/learn/embedded/serial-i2c-spi/)).

## What kinds of HATs exist?

The ecosystem is wide; the families you'll actually meet:

| Family | What it adds | Typical use |
|--------|--------------|-------------|
| **Relay / driver** | Mains-rated relays, motor drivers | Switching real-world loads safely |
| **Sensor** | Temperature, pressure, light, motion, LED matrices | Environment monitoring, learning |
| **Display** | Small LCD/OLED/e-ink screens | Status readouts on a headless box |
| **Audio** | Quality DAC/ADC, amplifiers | Music players, audio capture |
| **Power** | **PoE** (power over the Ethernet cable), UPS batteries | Clean single-cable installs |
| **Radio/comms** | GPS, LoRa, cellular modems | Telemetry, precise time |

Two have obvious appliance appeal. A **PoE HAT** powers the board through the
Ethernet cable from a PoE switch — one cable to an attic-mounted scanner, no mains
socket needed (and one fewer switching supply near the antenna). A small **display
HAT** can show status on the box itself.

## Can you stack them?

Physically, often — stacking headers pass the pins through. Electrically, it
depends on **pin claims**: two HATs both claiming the same SPI chip-select or the
same GPIO collide; two devices *sharing* the I2C bus at different addresses
coexist happily (that's what I2C is for). Before stacking, compare the boards'
pinout claims (vendors publish them; community sites collect them). And remember
the physical budgets: HATs draw from the board's supply — sum the current like
[Power supplies](/learn/embedded/power-supplies/) taught — and a HAT sitting over
the SoC changes airflow, which [Cases &amp; cooling](/learn/embedded/cases-and-cooling/)
warned about; look for HATs with cutouts, or add standoff height.

> Rule of thumb: one HAT is plug-and-play; two HATs is an engineering review.
> Check pin claims, current draw, and airflow before assuming they'll share.

## HAT or USB — how do you choose?

Many capabilities exist in both forms (audio, GPS, even displays). The decision
pattern:

- **Choose the HAT** when the feature is header-native (relays, GPIO-driven
  hardware, PoE), when tidiness matters (everything inside one case, no cable
  spaghetti), or when USB ports/bandwidth are scarce.
- **Choose USB** when a mature USB version exists and portability matters — a USB
  device moves to your laptop or a future mini PC
  ([When you need more](/learn/embedded/when-you-need-more/) — mini PCs have no
  header), and it keeps the header free.

The appliance's radio settles firmly on USB: **RTL-SDR dongles are USB devices**
— that's the standard, well-supported form, movable between machines — which is
exactly where the [next lesson](/learn/embedded/connecting-an-sdr/) picks up. The
HAT slots stay available for the supporting cast: PoE power, a status display, a
real-time clock.

<div class="knowledge-check" data-quiz data-correct-msg="Right — the EEPROM lets the board identify the HAT and configure for it at boot." markdown="0">
  <p class="knowledge-check__q">Quick check: what does the little EEPROM chip on a proper HAT do?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">It stores the user's Wi-Fi password as a backup</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">It boosts the header's power output for the HAT's components</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">It identifies the HAT to the board over I2C at boot so drivers can auto-configure</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- A **HAT** presses onto the full header and adds hardware with no wiring —
  relays, sensors, displays, audio, **PoE**, radios.
- The onboard **EEPROM** identifies the HAT over I2C at boot — header-world
  plug-and-play.
- Stacking works only when **pin claims don't collide**; budget current and
  airflow too.
- **HAT vs USB**: HAT for header-native features and tidy one-box builds; USB for
  mature, portable peripherals.
- The appliance uses the header for the supporting cast (PoE, display); its
  radio is a **USB SDR** — next lesson.

Next up: [Connecting an SDR](/learn/embedded/connecting-an-sdr/).
