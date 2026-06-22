---
slug: arduino
title: Arduino & the maker ecosystem
description: Arduino is the open-source board and language that made microcontrollers approachable. Learn the sketch model, the blink program, and the ecosystem around it.
keywords: arduino, arduino uno, arduino ide, arduino sketch, setup loop, maker electronics, microcontroller for beginners, arduino shields, ATmega328
level: beginner
status: full
faq:
  - q: "What exactly is Arduino?"
    a: "Arduino is three things bundled together: open-source **boards** (like the Uno and Nano) built around a microcontroller, the free **Arduino IDE** you write code in, and a friendly **C++ library** that hides the chip's low-level details behind simple commands like `digitalWrite()` and `delay()`. Together they turned bare-metal microcontroller programming into something a beginner can pick up in an afternoon."
  - q: "What is a \"sketch\" in Arduino?"
    a: "A **sketch** is just what Arduino calls a program. Every sketch has two functions: `setup()`, which runs once when the board powers on, and `loop()`, which runs over and over forever afterward. That `setup()` then `loop()` shape mirrors how a microcontroller actually works — initialize once, then repeat the main job endlessly — which is why it's such an effective teaching model."
  - q: "Are Arduino boards powerful enough for real projects?"
    a: "For their niche, yes — but know the limits. The classic Uno uses an 8-bit ATmega328 with just **2 KB of RAM** running at 16 MHz, which is plenty for reading sensors and blinking lights but not for video, heavy math, or networking. For wireless or more compute you step up to a board like the **ESP32**, which uses the same Arduino tooling but adds Wi-Fi and far more power."
---
# Arduino & the maker ecosystem

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Arduino made microcontrollers approachable** — open-source boards, a simple IDE, and a friendly C++ library bundled together. **The sketch model** — every program has a `setup()` that runs once and a `loop()` that repeats forever. **A huge ecosystem** — shields, sensors, libraries, and a vast community made it the standard on-ramp for beginners.
</div>

In the last lesson you met the microcontroller: a whole computer on one bare-metal chip. Programming one *used* to mean wrestling with datasheets, special hardware programmers, and raw registers. Arduino changed that. It took the same chips and wrapped them in tools simple enough for a complete beginner — and in doing so it created the modern maker movement. This lesson covers what Arduino is, the sketch model at its heart, and the ecosystem that grew up around it.

## What Arduino actually is

Arduino isn't a single product — it's a whole approach made of three parts that work together:

- **Open-source boards.** The classic **Uno** and the smaller **Nano** are the famous ones, built around Microchip's 8-bit ATmega microcontrollers. Because the designs are open, dozens of compatible boards exist too.
- **The Arduino IDE.** A free, beginner-friendly editor where you write, compile, and upload code with a single click — no separate toolchain to assemble.
- **A friendly C++ library.** This is the quiet hero. It hides the chip's intimidating low-level registers behind plain-English commands, so turning on a pin is `digitalWrite(13, HIGH)` instead of a cryptic bit operation on a hardware register.

The genius was bundling all three. Before Arduino, getting "hello world" onto a microcontroller could take a beginner days. After Arduino, it took minutes — and that drop in friction is what opened embedded electronics to artists, students, and hobbyists, not just engineers.

## The sketch model: setup() and loop()

An Arduino program is called a **sketch**, and every sketch is built from exactly two functions:

- **`setup()`** runs **once** when the board powers on or resets. You use it to get things ready — configure pins, start a sensor, open a serial connection.
- **`loop()`** runs **over and over, forever**, the instant `setup()` finishes. This is the main job, repeated endlessly until power is cut.

That shape isn't arbitrary — it mirrors how a microcontroller genuinely behaves. With no operating system to return to, the chip must keep doing *something* forever, so "set up once, then repeat" is the natural rhythm of bare-metal code. Here's the canonical first sketch, which blinks the board's built-in LED:

```cpp
void setup() {
  pinMode(LED_BUILTIN, OUTPUT);   // run once: set the LED pin as an output
}

void loop() {
  digitalWrite(LED_BUILTIN, HIGH); // turn the LED on
  delay(1000);                     // wait 1000 ms (one second)
  digitalWrite(LED_BUILTIN, LOW);  // turn the LED off
  delay(1000);                     // wait another second, then loop repeats
}
```

That's a complete, working program. The "Blink" sketch is the embedded world's equivalent of "Hello, world" — once an LED blinks, you know your code compiled, uploaded, and is actually running on the metal.

## The ecosystem that grew around it

Arduino's lasting impact is as much social as technical. A massive **ecosystem** formed around the boards:

- **Shields** — add-on boards that stack onto an Uno's pins to add a feature (a motor driver, a screen, a GPS, an Ethernet port) with no wiring.
- **Sensors and modules** — cheap breakout boards for temperature, motion, light, distance, and more, nearly all with ready-made Arduino libraries.
- **Libraries** — community-written code that lets you drive a complex part in a few lines instead of reading its datasheet.
- **Community** — countless tutorials, forum threads, and project write-ups, so almost any beginner problem has already been solved and posted somewhere.

This is the real reason Arduino became *the* on-ramp. The hardware abstraction got you started, and the ecosystem meant you were never stuck for long. Typical projects: prototyping a product idea, classroom and university teaching, hobby electronics and art installations, and simple home automation.

## Strengths and drawbacks

Arduino is brilliant at what it's for, and it's worth being clear-eyed about where it stops.

| | Arduino (classic AVR boards) |
|---|---|
| **Strengths** | Easy to learn; very cheap; enormous community and library support; hardware details abstracted away; quick from idea to blinking LED |
| **Drawbacks** | Classic boards are slow 8-bit chips with little memory (Uno: 16 MHz, 2 KB RAM); the single `loop()` runs one thing at a time; no built-in networking on basic boards; not for heavy compute, video, or big data |

The drawbacks aren't flaws so much as the cost of simplicity. The 8-bit Uno is wonderful for learning and for small control jobs, but it can't stream video or talk to the internet on its own. When a project needs wireless or more horsepower, you reach for a more capable chip that *keeps the same friendly tooling* — which is exactly the ESP32, the subject of the next lesson. (And to be clear, these chips are far too small to run something like GopherTrunk; an MCU is a teaching contrast to the SBCs and PCs that do.)

<div class="knowledge-check" data-quiz data-correct-msg="Right — setup() runs once at power-on, and loop() repeats forever afterward." markdown="0">
  <p class="knowledge-check__q">Quick check: In an Arduino sketch, what is the role of the two standard functions?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Both `setup()` and `loop()` run once and then the board stops</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">`setup()` runs once at power-on; `loop()` then repeats forever</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">`loop()` runs first to load the operating system, then `setup()` takes over</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Three things in one** — Arduino is open-source boards, a simple IDE, and a friendly C++ library bundled to make microcontrollers approachable.
- **The sketch model** — `setup()` runs once at power-on; `loop()` repeats forever, mirroring how a bare-metal chip really works.
- **Blink is hello-world** — a few lines turn an LED on and off and prove your code is running on the metal.
- **A rich ecosystem** — shields, sensors, libraries, and a huge community made Arduino the standard beginner on-ramp.
- **Know the limits** — classic 8-bit boards are slow with little memory and no networking; step up when you need wireless or compute.

Next up: a microcontroller with Wi-Fi and Bluetooth built in, for the price of a coffee. See [ESP32 & wireless microcontrollers](/learn/intro-hardware/esp32/).
