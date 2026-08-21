---
slug: serial-i2c-spi
title: Serial, I2C & SPI
description: The three little buses that connect chips to computers — UART serial for consoles and GPS, I2C for fleets of slow sensors, SPI for fast displays — and how to recognize and enable each on a pinout.
keywords: UART, I2C, SPI, serial console, i2cdetect, bus protocols, sensor bus, pinout, chip-to-chip communication, raspi-config interfaces
level: intermediate
status: full
prereq:
  - gpio-basics
---

# Serial, I2C & SPI

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
Beyond single pins, chips talk over three little buses that appear on every SBC
header. **UART serial** — two wires, point-to-point — carries consoles, GPS
modules, and radios' config ports. **I2C** — two *shared* wires, many devices,
each with an **address** — is the sensor bus: slow, simple, everywhere. **SPI** —
four-ish wires with a **chip-select** per device — is the fast one, for displays
and converters. Recognize them on a pinout (TX/RX; SDA/SCL; MOSI/MISO/SCLK/CE),
enable them in the board's config, and half the module datasheets on earth start
making sense.
</div>

GPIO gave you single wires; real chips — temperature sensors, displays, GPS
receivers — communicate in bytes over small standard buses. You don't need to
master electronics here: the goal is to *recognize* the three families, know what
each is for, and be able to enable and probe them. That literacy unlocks the whole
add-on ecosystem, including next lesson's HATs.

## Why buses instead of more pins?

A chip reporting a 16-bit temperature can't have 16 wires — pins are the most
precious resource in electronics. A **bus** sends bits *in sequence* over a couple
of shared wires instead, trading speed for pin count. The three survivors of
decades of chip design each picked a different point on that trade — which is why
all three still exist, and why your header carries all of them.

## What is UART serial, and why does it refuse to die?

**UART** (plain "serial") is the oldest and simplest: one wire for transmit
(**TX**), one for receive (**RX**), no shared clock — both ends just agree on a
speed (the **baud rate**, e.g. 115200). Point-to-point, one device per port.

Its killer feature on SBCs is the **serial console**: the kernel can present its
boot messages and a login prompt over the UART pins, working before networking,
before SSH — even while the board is failing to boot. With a $5 USB-to-serial
adapter between the board's pins and a laptop, you can watch a headless board's
boot scroll by — the debugging tool of last resort when
[first boot](/learn/embedded/first-boot-and-ssh/) never reaches the network.
Beyond consoles, UART is how GPS modules (a stream of text sentences) and many
radio modules present themselves. The one hardware rule from
[GPIO basics](/learn/embedded/gpio-basics/) applies doubly: check logic levels —
classic "RS-232" serial uses ±12 V and must never touch 3.3 V pins directly.

## What is I2C, and why do sensors love it?

**I2C** ("I-squared-C") is a *shared* two-wire bus: **SDA** (data) and **SCL**
(clock), with every device hanging on the same two wires. Each device has a 7-bit
**address**, and the board (the controller) calls devices by address. Dozens of
devices, two pins, done — which is why nearly every hobby sensor (temperature,
pressure, light, motion) and small OLED display speaks I2C. It's slow (100–400
kbit/s typically), and nobody cares; a thermometer has little to say.

Enable it (`sudo raspi-config` → Interface Options on a Pi), then probe the bus:

```bash
$ sudo apt install i2c-tools
$ i2cdetect -y 1
     0  1  2  3  4  5  6  7  8  9  a  b  c  d  e  f
40:          -- -- -- -- -- -- -- -- -- -- -- -- --
70:          -- -- -- -- 76 --
```

That `76` is a device answering at address 0x76 — a common pressure/temperature
sensor. `i2cdetect` drawing a map of everything wired to two pins is the moment
I2C clicks for most people: the bus is *discoverable*.

## What is SPI, and when do you need the speed?

**SPI** spends more pins to go fast: **MOSI** (controller out), **MISO**
(controller in), **SCLK** (clock), plus one **chip-select (CE)** line *per
device* — the controller raises a device's CE to address it, so device count is
limited by spare CE pins, not addresses. In exchange: tens of megabits, full
duplex. That speed is why pixel-pushing displays, fast analog-to-digital
converters, and flash chips choose SPI. Fun fact with a familiar ring: an SD card
is essentially a flash chip that can speak an SPI-family protocol — buses like
these are underneath everything.

## How do you tell them apart on a datasheet?

| | UART serial | I2C | SPI |
|---|---|---|---|
| **Wires** | TX, RX | SDA, SCL (shared) | MOSI, MISO, SCLK + CE each |
| **Devices per bus** | 1 (point-to-point) | Many, by address | One per CE line |
| **Speed** | ~0.1–1 Mbit/s | 0.1–1 Mbit/s | 10s of Mbit/s |
| **Addressing** | None | 7-bit address | Chip-select wire |
| **Typical devices** | Console, GPS, radio modules | Sensors, small OLEDs, RTCs | Displays, ADCs, flash |
| **Spot it by** | "TX/RX", baud rate | "SDA/SCL", 0x-address | "MOSI/MISO/SCLK" |

> Rule of thumb: read the module's pin labels and you know the bus — TX/RX means
> UART, SDA/SCL means I2C, MOSI/MISO means SPI. Then enable that interface in the
> board's config, and reach for the matching library.

## Where will you actually meet these?

Practically: every sensor tutorial assumes you can identify and enable the right
bus — now you can. On the appliance path, two cameos are worth knowing. A GPS
module (UART or USB) can give a scanner box a precise clock and location. And
next lesson's **HATs** are add-on boards that use these same buses — including an
I2C identity chip that tells the board what's stacked on top of it.

<div class="knowledge-check" data-quiz data-correct-msg="Right — I2C shares two wires among many devices, each reached by its address." markdown="0">
  <p class="knowledge-check__q">Quick check: how do many I2C devices share just two wires?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Each device gets its own dedicated chip-select wire</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Every device sits on the same two wires and answers to its own address</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">They take turns using the bus on a fixed one-second schedule</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- Buses trade **pins for sequence**: bits sent one after another over a few
  shared wires — the reason three little protocols run the chip world.
- **UART serial** (TX/RX, baud rate): point-to-point; consoles, GPS — and the
  headless board's debugging lifeline.
- **I2C** (SDA/SCL, addresses): many slow devices on two shared wires — the
  sensor bus, discoverable with `i2cdetect`.
- **SPI** (MOSI/MISO/SCLK/CE): pin-hungry but fast — displays, converters,
  flash.
- **Pin labels identify the bus**; enable it in board config and the module
  ecosystem opens up — starting with HATs, next.

Next up: [HATs &amp; add-ons](/learn/embedded/hats-and-add-ons/).
