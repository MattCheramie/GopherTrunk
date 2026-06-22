---
slug: esp32
title: ESP32 & wireless microcontrollers
description: The ESP32 is a microcontroller with Wi-Fi and Bluetooth built in for a couple of dollars. Learn why it powers most DIY smart-home gadgets and the IoT.
keywords: esp32, wireless microcontroller, esp8266, esp32-s3, esp32-c3, wifi microcontroller, IoT, ESPHome, Tasmota, MicroPython, ESP-IDF
level: intermediate
status: full
faq:
  - q: "What is the ESP32?"
    a: "The ESP32 is a low-cost microcontroller from Espressif with **Wi-Fi and Bluetooth built right into the chip**. It's typically dual-core, runs faster and with far more memory than a classic 8-bit Arduino, and costs only a few dollars on a small dev board. That combination of cheap, capable, and wireless is why it became the default chip for hobbyist Internet-of-Things projects."
  - q: "How is the ESP32 different from an Arduino Uno?"
    a: "The Uno is an 8-bit chip at 16 MHz with 2 KB of RAM and **no networking**. The ESP32 is a 32-bit, usually dual-core chip running at up to 240 MHz with hundreds of kilobytes of RAM and **built-in Wi-Fi and Bluetooth**. Crucially, you can program the ESP32 with the same friendly Arduino tooling, so it feels familiar while being far more powerful and wireless."
  - q: "What can you build with an ESP32?"
    a: "Mostly connected gadgets: smart-home sensors and switches, environmental monitors that post data to the cloud, Bluetooth peripherals, and DIY IoT devices of every kind. Ready-made firmware like **ESPHome** and **Tasmota** runs on ESP chips, letting you build smart-home devices without writing much code at all. It's still a microcontroller, though — for video or a full Linux app you'd want a single-board computer instead."
---
# ESP32 & wireless microcontrollers

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Wireless built in** — the ESP32 packs Wi-Fi and Bluetooth onto a cheap microcontroller, which is the game-changer for IoT. **Powerful for an MCU** — dual-core, up to 240 MHz, far more memory than a classic Arduino, for a couple of dollars. **Same friendly tooling** — program it with the Arduino core, ESP-IDF in C, or MicroPython, backed by a huge community.
</div>

The classic Arduino gave you a cheap brain but no way to talk to the network. The ESP32 fixes exactly that. It's a microcontroller with Wi-Fi and Bluetooth radios baked onto the same chip, for the price of a coffee — and that single feature is why it sits inside most DIY smart-home gadgets and hobbyist IoT projects today. This lesson covers what the ESP32 is, why built-in wireless matters so much, how you program it, and where its limits are.

## What the ESP32 is

The **ESP32** is a family of microcontrollers from **Espressif**. A typical ESP32 is a 32-bit, **dual-core** chip running at up to 240 MHz with several hundred kilobytes of RAM and, most importantly, **Wi-Fi and Bluetooth radios integrated onto the chip itself**. On a small development board with USB and a voltage regulator, it costs just a few dollars.

It has a lineage worth knowing. Its predecessor, the **ESP8266**, was the chip that first made cheap Wi-Fi microcontrollers mainstream — it's still around and still cheap, but single-core and more limited. The ESP32 superseded it, and Espressif has since shipped a range of **variants**: the **S3** with extra power and AI-friendly features, and the smaller, cheaper single-core **C3** built on a RISC-V core. They share tooling, so what you learn on one transfers to the others.

Compared with a classic 8-bit Arduino, the ESP32 is in a different league for raw capability — more cores, more speed, much more memory — while staying just as cheap. The headline difference, though, isn't the compute. It's the radio.

## Why built-in wireless changes everything

For the **Internet of Things (IoT)** — the idea of small devices that sense or control the physical world and report over a network — connectivity is the whole point. A temperature sensor that can't tell anyone the temperature isn't very useful. A light switch you can't reach from your phone is just a light switch.

Before the ESP chips, adding network access to a microcontroller meant bolting on a separate Wi-Fi or Ethernet shield — more cost, more wiring, more power, more complexity. The ESP32 collapses all of that into the main chip. For a couple of dollars you get a computer that can read a sensor *and* publish the reading to a server, host a little configuration web page, or pair with your phone over Bluetooth — all from one tiny part.

That economics is why the ESP32 became the backbone of the maker IoT world. When the radio is free and the chip is cheap, building a connected gadget stops being a project and starts being a weekend.

## How you program an ESP32

You have three well-trodden options, from easiest to most low-level:

| Option | Language | Best for |
|--------|----------|----------|
| **Arduino core for ESP32** | C++ (Arduino style) | Beginners and anyone already comfortable with Arduino sketches |
| **ESP-IDF** | C | Espressif's official SDK — full control, all features, production work |
| **MicroPython** | Python | Quick experiments and scripting on-device with a familiar language |

The **Arduino core for ESP32** is the common starting point: the same `setup()`/`loop()` model and libraries you saw in the last lesson, now with Wi-Fi and Bluetooth functions available. When you outgrow it, **ESP-IDF** gives you Espressif's full C SDK with access to every feature and finer control over power and timing. And **MicroPython** lets you write Python that runs right on the chip, which is great for tinkering. Most people start with the Arduino core and only drop down when they need to.

There's also a no-code-needed path. Off-the-shelf firmware like **ESPHome** and **Tasmota** runs on ESP chips and turns them into smart-home devices configured through simple text files or a web UI — a huge shortcut for home automation.

## Strengths and drawbacks

The ESP32's appeal is easy to state, and so are the trade-offs that come with cramming a radio onto a microcontroller.

| | ESP32 |
|---|---|
| **Strengths** | Built-in Wi-Fi and Bluetooth; very cheap; powerful for an MCU (dual-core, lots of RAM); reuses Arduino tooling; large, active community |
| **Drawbacks** | Wireless draws real current, so battery life needs careful **deep-sleep** design; RF and antenna layout add complexity; still far more constrained than an SBC; the chip runs warm under heavy radio use |

The biggest practical gotcha is **power**. A tiny sensor MCU can run for years on a coin cell, but an ESP32 actively using Wi-Fi can draw a few hundred milliamps in bursts — enough to flatten a small battery fast. The standard fix is **deep sleep**: the chip wakes briefly, takes a reading, sends it, then powers almost everything down for minutes or hours. Designed that way, an ESP32 sensor can still last a long time; designed naively, it won't. And remember the bigger picture — capable as it is, the ESP32 is still a microcontroller, not a Linux computer. For displays, heavy compute, or running full applications like GopherTrunk, you'd reach back up the spectrum to a single-board computer.

<div class="knowledge-check" data-quiz data-correct-msg="Right — the ESP32's built-in Wi-Fi and Bluetooth are what make it the default chip for connected IoT projects." markdown="0">
  <p class="knowledge-check__q">Quick check: What is the ESP32's defining advantage over a classic Arduino?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">It can run desktop Linux and full applications</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">It has Wi-Fi and Bluetooth built into the chip, making it ideal for IoT</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">It uses an entirely different language that Arduino tooling can't touch</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Wireless on-chip** — the ESP32 integrates Wi-Fi and Bluetooth onto a cheap microcontroller, the feature that defines it.
- **Powerful and cheap** — dual-core, up to 240 MHz, hundreds of KB of RAM, for a couple of dollars; the ESP8266 came before it and the S3/C3 are variants.
- **Built for IoT** — free connectivity turns building a connected sensor or switch into a weekend project.
- **Several ways in** — program it with the Arduino core, ESP-IDF in C, or MicroPython, or flash ready-made ESPHome/Tasmota firmware.
- **Mind the power** — active Wi-Fi draws real current, so battery projects lean hard on deep sleep; it's still an MCU, not an SBC.

Next up: the languages, workflow, and hard limits that apply across every microcontroller. See [Programming microcontrollers: languages, use cases & limits](/learn/intro-hardware/mcu-programming-and-tradeoffs/).
