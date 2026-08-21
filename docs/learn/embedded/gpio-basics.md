---
slug: gpio-basics
title: GPIO basics
description: General-purpose input/output pins you control from code — digital inputs and outputs, blinking an LED with a resistor, reading a button with a pull-up, and the 3.3 V rules that keep the magic smoke in.
keywords: GPIO, raspberry pi pins, blink an LED, read a button, pull-up resistor, 3.3V logic, gpio header, gpioset, digital input output
level: intermediate
status: full
prereq:
  - first-boot-and-ssh
faq:
  - q: What does GPIO stand for and what is it for?
    a: "General-Purpose Input/Output. GPIO pins are wires on the board's header that your code can drive high or low (output) or read as high or low (input). General-purpose means the board attaches no meaning to them — an LED, a relay, a button, a sensor: whatever you wire up, your software defines what the pin means."
  - q: Can GPIO pins damage my board?
    a: "Yes, in two classic ways. Feeding a pin more voltage than the board's logic level (3.3 V on a Pi — 5 V signals are too much) can kill the pin or the SoC, and drawing too much current from an output (pins supply a few milliamps, enough for an LED with a resistor but not a motor) can burn it out. The rules are simple: respect 3.3 V, always use resistors with LEDs, and switch big loads through a transistor or relay board."
  - q: Do I need GPIO for the GopherTrunk appliance?
    a: "No — the scanner build talks to its radio over USB and needs nothing on the header. GPIO is in this module because it is the SBC's defining feature and the gateway to the wider embedded world: status LEDs, buttons, fans, and sensors you may well add to an appliance once you have the skill."
---

# GPIO basics

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**GPIO** (general-purpose input/output) pins are the SBC's direct line to
electronics: your code can set an **output** pin high (3.3 V) or low (0 V), or read
an **input** pin's state. Two safety rules carry the day: pins are **3.3 V logic**
(5 V kills them) and can source only a few **milliamps** (LEDs need a **resistor**;
anything bigger needs a transistor or relay). Inputs must never float — a
**pull-up/pull-down** gives them a defined rest state. Blinking an LED and reading
a button are the hello-world of physical computing, and everything on the header
scales up from them.
</div>

Unit 4 turns to the SBC's defining feature: the double row of pins that no laptop
has. The appliance build won't strictly need them — but the header is why SBCs
exist, and thirty minutes with an LED and a button will change how you see every
device around you.

## What is the header, physically?

Along the board's edge sit 40 pins (on a Pi and many others). A few are fixed
supplies — **3.3 V**, **5 V**, and **ground** — and the rest are GPIOs, numbered by
the SoC (BCM numbering on a Pi: "GPIO17" is a chip signal name, *not* physical pin
17 — the perennial beginner trap; always check a pinout diagram).

<figure class="figure" markdown="0">
<svg viewBox="0 0 520 150" role="img" aria-label="A simplified GPIO header: two rows of pins with 3.3 volt, 5 volt, ground, and numbered GPIO pins labelled, and a circuit from GPIO17 through a resistor and LED to ground." xmlns="http://www.w3.org/2000/svg">
  <rect x="20" y="30" width="300" height="54" fill="none" stroke="currentColor" stroke-width="1.5" rx="6"/>
  <g fill="currentColor">
    <circle cx="45" cy="47" r="4"/><circle cx="45" cy="67" r="4"/>
    <circle cx="85" cy="47" r="4"/><circle cx="85" cy="67" r="4"/>
    <circle cx="125" cy="47" r="4"/><circle cx="125" cy="67" r="4"/>
    <circle cx="165" cy="47" r="4"/><circle cx="165" cy="67" r="4"/>
    <circle cx="205" cy="47" r="4"/><circle cx="205" cy="67" r="4"/>
    <circle cx="245" cy="47" r="4"/><circle cx="245" cy="67" r="4"/>
    <circle cx="285" cy="47" r="4"/><circle cx="285" cy="67" r="4"/>
  </g>
  <text x="45" y="22" text-anchor="middle" font-size="11" fill="currentColor">3.3V</text>
  <text x="85" y="22" text-anchor="middle" font-size="11" fill="currentColor">5V</text>
  <text x="125" y="22" text-anchor="middle" font-size="11" fill="currentColor">GND</text>
  <text x="205" y="22" text-anchor="middle" font-size="11" fill="currentColor">GPIO17</text>
  <text x="165" y="100" text-anchor="middle" font-size="11" fill="currentColor" fill-opacity="0.7">…40 pins: supplies, grounds, GPIOs</text>
  <line x1="205" y1="47" x2="380" y2="47" stroke="currentColor" stroke-width="1.5"/>
  <path d="M380 47 l8 -6 8 12 8 -12 8 12 8 -6 h8" fill="none" stroke="currentColor" stroke-width="1.5"/>
  <text x="404" y="34" text-anchor="middle" font-size="11" fill="currentColor">330 Ω</text>
  <polygon points="436,41 436,53 448,47" fill="none" stroke="currentColor" stroke-width="1.5"/>
  <line x1="448" y1="41" x2="448" y2="53" stroke="currentColor" stroke-width="1.5"/>
  <text x="442" y="70" text-anchor="middle" font-size="11" fill="currentColor">LED</text>
  <line x1="448" y1="47" x2="480" y2="47" stroke="currentColor" stroke-width="1.5"/>
  <line x1="480" y1="47" x2="480" y2="80" stroke="currentColor" stroke-width="1.5"/>
  <line x1="470" y1="80" x2="490" y2="80" stroke="currentColor" stroke-width="1.5"/>
  <line x1="474" y1="85" x2="486" y2="85" stroke="currentColor" stroke-width="1.5"/>
  <line x1="478" y1="90" x2="482" y2="90" stroke="currentColor" stroke-width="1.5"/>
</svg>
<figcaption>The hello-world circuit: <strong>GPIO17 → resistor → LED → ground</strong>. The resistor limits current; the pin's high/low state is the switch.</figcaption>
</figure>

## What are the two rules that protect the board?

1. **3.3 V logic, absolutely.** A Pi's GPIOs speak 3.3 V: output high *is* 3.3 V,
   and an input must never be fed more. Plenty of hobby electronics (many Arduino
   modules) are 5 V — connecting their outputs to a Pi input without a level
   shifter risks the pin or the whole SoC. Check every module's logic level before
   wiring.
2. **Milliamps only.** An output pin can safely source roughly 8–16 mA — an LED
   through a **current-limiting resistor** (330 Ω is the classic value), no more. A
   motor, a lamp, a fan: those are switched *indirectly* — the pin drives a
   transistor or a relay module, which switches the real current from a proper
   supply. The pin is a **signal**, never a power source.

> Rule of thumb: before connecting anything to the header, answer two questions —
> *what voltage does it present to the pin?* and *how much current will flow?* If
> either answer is "not sure," stop and look it up. Magic smoke is not covered by
> warranty.

## How do you blink an LED from the shell?

Wire the figure's circuit (long LED leg toward the pin, short leg to ground via
resistor — a breadboard and jumper wires make it solderless). Modern Linux exposes
GPIO through the `gpiod` tools:

```bash
$ sudo apt install gpiod
$ gpiodetect                      # list GPIO chips
$ gpioset gpiochip0 17=1          # GPIO17 high — LED on
$ gpioset gpiochip0 17=0          # low — off

# blink forever
$ while true; do gpioset gpiochip0 17=1; sleep 0.5; gpioset gpiochip0 17=0; sleep 0.5; done
```

That shell loop is real physical computing: code changing the world, twice a
second. Every language has libraries wrapping the same interface (Python's
`gpiozero` is the gentlest; Go has `periph.io`) — see
[Programming an SBC](/learn/intro-hardware/programming-an-sbc/) for the software
side.

## How do you read a button — and what is a floating input?

An input pin reads high or low — but a pin connected to *nothing* reads **neither
reliably**: it "floats," picking up stray charge and reading back noise. Every
input needs a defined rest state via a **pull-up** (resistor to 3.3 V — rests
high) or **pull-down** (to ground — rests low). SoCs have internal ones you can
enable in software. The standard button circuit: internal pull-up on, button
wired from pin to ground — released reads 1, pressed reads 0:

```bash
$ gpioget --bias=pull-up gpiochip0 27
1        # not pressed
```

Mechanical buttons also **bounce** — one press lands as a burst of transitions
over a few milliseconds. Libraries debounce for you; just know the word for when a
single press "counts" five times.

## Where does GPIO go from here?

Single pins get you LEDs, buttons, relays, and fans. Richer devices — sensors,
displays, converters — speak *protocols* over dedicated header pins, which is the
[next-but-one lesson](/learn/embedded/serial-i2c-spi/); whole pre-wired boards
stack onto the header as [HATs](/learn/embedded/hats-and-add-ons/). And on an
appliance, GPIO is how you'd add a physical status LED ("decoding now") or a safe
shutdown button — small touches that make a headless box friendlier to the humans
living with it.

<div class="knowledge-check" data-quiz data-correct-msg="Right — an unconnected input floats; a pull-up or pull-down gives it a defined rest state." markdown="0">
  <p class="knowledge-check__q">Quick check: why does a button input need a pull-up or pull-down resistor?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">To make the button light up when pressed</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">To protect the button from the pin's high current</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Because a disconnected input floats and reads random noise without a defined rest state</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **GPIO** pins are code-controlled wires: **outputs** you set high/low, **inputs**
  you read — meaning defined entirely by what you wire and write.
- The header mixes **supplies, grounds, and GPIOs**, and chip numbering ≠ physical
  numbering — always consult a pinout.
- Safety rules: **3.3 V logic** (never 5 V into a pin) and **milliamps only**
  (LED + resistor direct; bigger loads via transistor/relay).
- Inputs need a **pull-up/pull-down** or they float; buttons also **bounce**.
- `gpioset`/`gpioget` blink and read from the shell; libraries and
  [buses](/learn/embedded/serial-i2c-spi/) scale the same idea to real devices.

Next up: [USB &amp; powered hubs](/learn/embedded/usb-and-powered-hubs/).
