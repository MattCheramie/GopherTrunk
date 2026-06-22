---
slug: programming-an-sbc
title: Programming & running software on an SBC
description: An SBC is just a Linux box, so you use the same tools as a PC — Python, C, Go, git, Docker, SSH — plus GPIO libraries to talk to sensors and hardware.
keywords: programming a Raspberry Pi, SBC programming, Python on Raspberry Pi, GPIO programming, gpiozero, libgpiod, flash Raspberry Pi OS, headless Raspberry Pi, SSH, Docker on Pi
level: intermediate
status: full
faq:
  - q: "What programming languages can I use on a single-board computer?"
    a: "Because an SBC runs full Linux, you can use almost any language: **Python** (the most popular for beginners and GPIO), **C/C++** for performance, **Go** for self-contained services, plus **Node.js**, **Rust**, **Java**, and more. The same compilers, interpreters, and package managers you'd use on a Linux PC work here."
  - q: "How do I set up a Raspberry Pi for the first time?"
    a: "You **flash an operating system** (such as Raspberry Pi OS) onto a microSD card using a tool like the Raspberry Pi Imager, insert the card, and power on. The Imager can pre-configure Wi-Fi and SSH so the board comes up **headless** — no monitor needed — and you connect to it over the network."
  - q: "How does a program control GPIO pins?"
    a: "Through a **library**. On Raspberry Pi, **gpiozero** offers a friendly high-level API, while **RPi.GPIO** and the modern **libgpiod** give lower-level control. For sensors you'll often use **I2C, SPI, or UART** libraries. Your code reads pin states as inputs or drives them as outputs, just like any other function call."
---
# Programming & running software on an SBC

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**It's just Linux** — so you use ordinary tools: Python, C, Go, git, Docker, and SSH, exactly as on a PC. **You boot from a flashed SD card** and usually run it *headless* over SSH, with no monitor. **GPIO is what's extra** — libraries like gpiozero and libgpiod, plus I2C/SPI/UART, let your code read sensors and drive hardware. **It runs services 24/7**, which is why it's perfect for an always-on job like a GopherTrunk scanner.
</div>

The intimidating part of an SBC turns out not to be intimidating at all. People expect special embedded tooling, but the truth is simpler: a single-board computer is a normal Linux computer, and you program it with the same skills and tools you'd use on any Linux machine. This lesson covers the one big idea, how to get an OS onto the board, how you typically work with it day to day, and the one genuinely new skill — talking to the physical world through GPIO.

## The big idea: it's a normal Linux box

Everything else in this lesson follows from one fact: an SBC runs a full operating system, almost always **Linux**. That means there's very little new to learn about the *software* side.

You write code in the same languages you'd use on a PC. **Python** is the most popular on SBCs — easy to learn and the best supported for GPIO work — but **C and C++** are there when you need speed, **Go** is excellent for compiling a single self-contained binary you can drop onto the board, and **Node.js**, **Rust**, and **Java** all run fine too. Whatever you'd reach for on a Linux desktop, you can reach for here.

The tooling is identical as well. You install software with the system **package manager** (`apt` on Raspberry Pi OS), pull your code with **git**, isolate and ship apps with **Docker**, and connect remotely with **SSH**. There's no special "SBC SDK" standing between you and the machine. If you can work on a Linux server, you can work on a Pi.

This is the deepest practical difference from the next module's microcontrollers: there, you cross-compile a single program and flash it onto bare metal. Here, you just... use a computer.

## Getting an OS onto the board

An SBC has no built-in disk in the PC sense — it boots from a **microSD card** (or, increasingly, an NVMe SSD on a Pi 5). So the first step is putting an operating system onto that card, a process called **flashing**.

The standard route on a Raspberry Pi is the official **Raspberry Pi Imager**: you pick an OS image (Raspberry Pi OS is the default; many people run plain Debian or Ubuntu), point it at the SD card, and write. Crucially, the Imager lets you pre-configure settings *before* first boot — your Wi-Fi network, a username and password, and **SSH enabled**. Set those, and the board will join your network and accept remote logins the moment it powers up.

After flashing, you insert the card, connect power, and wait a few seconds for it to boot. From there it behaves like any Linux machine on your network.

## Headless vs desktop use

There are two ways to actually use an SBC, and which you choose shapes your whole workflow:

- **Desktop use** — plug in an HDMI monitor, keyboard, and mouse, and use the graphical desktop directly, like a small PC. Good for learning and for projects where someone interacts with the screen.
- **Headless use** — no monitor at all. You connect from your main computer over **SSH** and work entirely on the command line (and copy files with `scp` or `rsync`). This is how most real SBC projects run, because the board is usually tucked away doing a job, not sitting on a desk.

For an embedded job, headless is the norm. The board lives wherever it needs to be — in a robot, in a closet, at the base of an antenna — and you reach it over the network when you need to.

## Talking to the physical world with GPIO

The one genuinely new skill compared to PC programming is driving the **GPIO** header — the pins that let your code touch real hardware. On Linux this is done through libraries rather than raw register pokes:

| Tool / interface | What it's for | Notes |
|------------------|---------------|-------|
| **gpiozero** | Simple on/off GPIO in Python | Friendly, beginner-first; great for LEDs, buttons, motors |
| **RPi.GPIO** | Lower-level Python GPIO | Older, more manual; widely used in tutorials |
| **libgpiod** | Modern Linux GPIO interface | The current standard; usable from C and Python |
| **I2C** | Talk to chips over 2 wires | Many sensors, displays, real-time clocks |
| **SPI** | Faster 4-wire bus | Screens, ADCs, some radio front ends |
| **UART** | Serial (TX/RX) link | GPS modules, debug consoles, other boards |

A first program is usually blinking an LED with gpiozero — set a pin as an output, turn it on, wait, turn it off. From there you read a button (an input), then a sensor over I2C or SPI, then build up to a real device. The mental model is comforting: a pin is just something your code reads from or writes to, like a variable that happens to be wired to the outside world.

## Running services 24/7

The reason an SBC is so well suited to embedded work is that, being a real Linux box, it can run a program **continuously and unattended** for months. You set your software up as a system service (typically with `systemd`) so it starts automatically on boot and restarts if it crashes — no one has to log in to kick it off.

This is exactly the **GopherTrunk** workflow. Picture a Raspberry Pi sitting at the antenna with a USB SDR dongle plugged in. You flash an SD card, bring the Pi up headless, and SSH in from your laptop. You install GopherTrunk and its dependencies with the package manager and git, attach the dongle, and configure the scanner to run as a service. From then on the Pi quietly scans around the clock, while you check on it remotely whenever you like. The board is small, quiet, and low-power — and it's a *real computer*, so it does the job a PC would, without tying up a PC.

<div class="knowledge-check" data-quiz data-correct-msg="Right — an SBC runs full Linux, so you use the same languages and tools as on a PC, plus GPIO libraries for hardware." markdown="0">
  <p class="knowledge-check__q">Quick check: How do you typically program a single-board computer?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">With a special embedded SDK that replaces normal Linux tools</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">With ordinary Linux tools and languages (Python, C, Go, git, Docker), plus GPIO libraries for hardware</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">By flashing a single bare-metal program with no operating system</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **It's just Linux** — write in Python, C/C++, Go, Node, or Rust, and use apt, git, Docker, and SSH exactly as on a PC.
- **You boot from a flashed card** — write an OS image to a microSD (or NVMe), pre-configure Wi-Fi and SSH, then power on.
- **Headless is the norm** — most projects run with no monitor, accessed over SSH, with the board tucked away doing its job.
- **GPIO is the new skill** — gpiozero, RPi.GPIO, and libgpiod drive pins; I2C/SPI/UART talk to sensors and chips.
- **It runs services 24/7** — set software up with systemd to run unattended, which is why an SBC suits an always-on GopherTrunk scanner.

Next up: where SBCs shine, and the limits that tell you when to choose something else. See [SBC use cases, strengths & drawbacks](/learn/intro-hardware/sbc-use-cases-and-limits/).
