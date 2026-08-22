---
slug: glossary
title: Glossary of embedded & SBC terms
description: Plain-language definitions of the terms used in embedded systems and single-board computing — embedded system, SBC, microcontroller, SoC, ARM, GPIO, I2C, SPI, HAT, headless, systemd, watchdog, thermal throttling, undervoltage, and more — each cross-linked to the lesson that explains it.
keywords: embedded glossary, SBC terms, microcontroller, SoC, ARM, GPIO, I2C, SPI, HAT, headless, systemd, watchdog, thermal throttling, undervoltage, SD card wear
level: beginner
status: full
lesson_standalone: true
---

# Glossary of embedded & SBC terms

Every term used across the [Embedded &amp; Single-Board Computers](/learn/embedded/)
module, defined in plain language and linked to the lesson where it's explained in
full. Skim it as a refresher, or use your browser's find (Ctrl/Cmd-F) to jump to a
word. Terms are grouped by theme, roughly in the order the module introduces them.

> Looking for deeper reference entries on hardware families, chips, and organizations?
> The [Field Guide](/reference/) covers them alongside the RF and software worlds.

## The landscape

**Embedded system** — A computer built into a device to do one dedicated job,
unattended — the computers hidden in routers, cars, and appliances. See
[What is an embedded system?](/learn/embedded/what-is-embedded/)

**Single-board computer (SBC)** — A complete computer — processor, RAM, storage slot,
USB, networking — on one small board running a real operating system; the Raspberry
Pi is the canonical example. See
[SBC vs microcontroller vs PC](/learn/embedded/sbc-vs-microcontroller-vs-pc/)

**Microcontroller (MCU)** — A single-chip computer with no operating system that runs
one program with instant startup, precise timing, and tiny power draw — Arduino and
ESP32 territory. See
[SBC vs microcontroller vs PC](/learn/embedded/sbc-vs-microcontroller-vs-pc/)

**Appliance** — A computer dedicated to one job, expected to run unattended and
recover on its own — the design goal of this module's final build. See
[What is an embedded system?](/learn/embedded/what-is-embedded/)

**Headroom** — Spare capacity (CPU, RAM, power, thermal) left above a workload's
needs, so busy moments don't push the system over a cliff. See
[Picking a board](/learn/embedded/picking-a-board/)

**System-on-chip (SoC)** — One chip integrating CPU cores, GPU, memory controller,
and I/O controllers — the reason an SBC fits on a credit card and can't be upgraded.
See [ARM and the system-on-chip](/learn/embedded/arm-and-socs/)

**ARM** — The company (and architecture) whose licensed CPU core designs power nearly
every phone and small board; optimised for performance per watt. See
[ARM and the system-on-chip](/learn/embedded/arm-and-socs/)

**Instruction set** — The vocabulary of operations a CPU understands; compiled
software is built for one instruction set and won't run on another. See
[ARM and the system-on-chip](/learn/embedded/arm-and-socs/)

**arm64 / aarch64** — The 64-bit ARM architecture; the flavour to match when
downloading binaries for a modern Pi (`uname -m` tells you which you have). See
[ARM and the system-on-chip](/learn/embedded/arm-and-socs/)

**Ecosystem** — The software, documentation, accessories, and community around a
board — the Raspberry Pi's real moat, and worth more than spec-sheet numbers for a
first build. See [Why the Raspberry Pi?](/learn/embedded/why-the-raspberry-pi/)

## OS & setup

**OS image** — A byte-for-byte snapshot of a complete bootable disk, written directly
onto an SD card; how operating systems are installed on SBCs. See
[Operating systems for small boards](/learn/embedded/operating-systems-for-sbcs/)

**Raspberry Pi OS** — The first-party Debian-based Linux distribution for Raspberry
Pi boards; the default foundation for Pi projects. See
[Operating systems for small boards](/learn/embedded/operating-systems-for-sbcs/)

**Lite image** — An OS image with no graphical desktop — boots to a text console;
the right choice for headless appliances. See
[Operating systems for small boards](/learn/embedded/operating-systems-for-sbcs/)

**Boot partition** — The small FAT partition of an SBC's card holding firmware,
kernel, and plain-text config files — editable from any PC, which enables headless
preconfiguration. See
[Operating systems for small boards](/learn/embedded/operating-systems-for-sbcs/)

**Flashing** — Writing an OS image byte-for-byte onto a card or drive, replacing its
entire contents with a bootable system. See
[Flashing an OS image](/learn/embedded/flashing-an-os-image/)

**Headless** — Running a computer with no monitor or keyboard, operated entirely over
the network — the normal state of an SBC appliance. See
[First boot &amp; SSH](/learn/embedded/first-boot-and-ssh/)

**SSH (Secure Shell)** — The encrypted remote-login protocol that puts a board's
shell in your terminal; the way everything on a headless board gets done. See
[First boot &amp; SSH](/learn/embedded/first-boot-and-ssh/)

**mDNS** — Multicast DNS, which lets a board advertise itself as `hostname.local` so
you can find it without knowing its IP address. See
[First boot &amp; SSH](/learn/embedded/first-boot-and-ssh/)

## Buying decisions

**A1 / A2 class** — SD-card application-performance ratings measuring random
read/write speed — the best predictor of how fast Linux feels on a card. See
[Storage &amp; SD cards](/learn/embedded/storage-and-sd-cards/)

**High-endurance card** — An SD card built with wear headroom for continuous
recording — the right card class for an always-on appliance. See
[Storage &amp; SD cards](/learn/embedded/storage-and-sd-cards/)

**NVMe** — A fast, direct SSD interface some boards offer; with USB SSDs, the
durable, quick upgrade over SD storage. See
[Storage &amp; SD cards](/learn/embedded/storage-and-sd-cards/)

**Undervoltage** — The board's supply voltage sagging below spec — the classic cause
of random crashes, USB resets, and SD corruption on SBCs. See
[Power supplies](/learn/embedded/power-supplies/)

**Power budget** — The sum of current the supply must deliver — board plus every
bus-powered peripheral — with margin; exceeded budgets fail intermittently. See
[Power supplies](/learn/embedded/power-supplies/)

**vcgencmd** — The Raspberry Pi firmware query tool: `measure_temp` for temperature,
`get_throttled` for undervoltage and throttling flags. See
[Power supplies](/learn/embedded/power-supplies/)

**Passive cooling** — Removing heat with heatsinks or a thermal metal case — silent,
with no moving parts to fail; the appliance favourite. See
[Cases &amp; cooling](/learn/embedded/cases-and-cooling/)

**Active cooling** — Removing heat with a fan — more cooling per dollar, at the cost
of noise, dust, and a wear item. See
[Cases &amp; cooling](/learn/embedded/cases-and-cooling/)

**Load average** — Linux's measure of runnable work; sustained values at or above
the core count mean saturation — the evidence-based upgrade signal. See
[When you need more than a Pi](/learn/embedded/when-you-need-more/)

**Mini PC** — A small, quiet x86 computer — the next rung when a project outgrows an
SBC; same Linux, same skills, more of everything except GPIO. See
[When you need more than a Pi](/learn/embedded/when-you-need-more/)

## Running like a server

**sudo** — Per-command administrative privilege for a normal user, logged each use —
how root power is exercised without logging in as root. See
[Users, permissions &amp; updates](/learn/embedded/users-and-updates/)

**unattended-upgrades** — The Debian mechanism that installs security patches
automatically — how an appliance stays patched during the months you forget it. See
[Users, permissions &amp; updates](/learn/embedded/users-and-updates/)

**Service (daemon)** — A long-running background program with no terminal, started
and supervised by the init system. See
[Services with systemd](/learn/embedded/services-with-systemd/)

**systemd** — The init system on modern Linux: starts services at boot, restarts them
on failure, and collects their logs; driven with `systemctl`. See
[Services with systemd](/learn/embedded/services-with-systemd/)

**Unit file** — The short INI-style file describing how systemd runs a program —
`ExecStart`, the service user, and restart policy. See
[Services with systemd](/learn/embedded/services-with-systemd/)

**journal / journalctl** — systemd's log store and its reader (`journalctl -u name
-f` follows a service live) — the window into a headless daemon. See
[Services with systemd](/learn/embedded/services-with-systemd/)

**DHCP reservation** — A router setting that always hands a given board the same IP
address — the low-maintenance way to make an appliance's address permanent. See
[Networking your board](/learn/embedded/networking-your-board/)

## Talking to hardware

**GPIO** — General-purpose input/output: header pins your code drives high/low or
reads — the SBC's direct line to LEDs, buttons, and electronics. See
[GPIO basics](/learn/embedded/gpio-basics/)

**Logic level** — The voltage a board's pins speak (3.3 V on a Pi); feeding a pin
more than its logic level damages it. See
[GPIO basics](/learn/embedded/gpio-basics/)

**Pull-up / pull-down** — A resistor (often internal) giving an input pin a defined
rest state, without which it floats and reads noise. See
[GPIO basics](/learn/embedded/gpio-basics/)

**Bus-powered** — A USB device drawing all its power through the port — every
milliamp comes out of the board's own supply budget. See
[USB &amp; powered hubs](/learn/embedded/usb-and-powered-hubs/)

**Powered hub** — A USB hub with its own supply, so downstream devices stop drawing
through the board — the cure for crowded-bus resets. See
[USB &amp; powered hubs](/learn/embedded/usb-and-powered-hubs/)

**UART (serial)** — The two-wire point-to-point bus (TX/RX at an agreed baud rate) —
consoles, GPS modules, and the headless board's debugging lifeline. See
[Serial, I2C &amp; SPI](/learn/embedded/serial-i2c-spi/)

**I2C** — The shared two-wire bus (SDA/SCL) where many slow devices coexist, each at
its own address — the sensor bus, probed with `i2cdetect`. See
[Serial, I2C &amp; SPI](/learn/embedded/serial-i2c-spi/)

**SPI** — The fast four-wire bus (MOSI/MISO/SCLK plus a chip-select per device) for
displays, converters, and flash. See
[Serial, I2C &amp; SPI](/learn/embedded/serial-i2c-spi/)

**HAT** — Hardware Attached on Top: a standardised add-on board that presses onto the
full header and identifies itself via an onboard EEPROM. See
[HATs &amp; add-ons](/learn/embedded/hats-and-add-ons/)

**PoE (Power over Ethernet)** — Powering a board through its network cable via a PoE
HAT and switch — one-cable installs for closet and attic appliances. See
[HATs &amp; add-ons](/learn/embedded/hats-and-add-ons/)

**RTL-SDR** — The ~$30 USB software-defined-radio dongle that streams a slice of raw
spectrum to software — the appliance's radio front end. See
[Connecting an SDR](/learn/embedded/connecting-an-sdr/)

**Module blacklist** — Telling the kernel not to load a driver — required so the
DVB-T TV-tuner module doesn't claim the RTL-SDR before SDR software can. See
[Connecting an SDR](/learn/embedded/connecting-an-sdr/)

**rtl_test** — The acceptance test for a dongle: stream samples and report losses; a
clean run separates hardware problems from software ones forever after. See
[Connecting an SDR](/learn/embedded/connecting-an-sdr/)

## Running 24/7

**Thermal throttling** — The SoC reducing its own clock speed as it nears its
temperature limit — silent, protective, and a real-time decoder's enemy. See
[Thermal throttling](/learn/embedded/thermal-throttling/)

**Write cycles / wear** — Flash cells endure a finite number of writes; constant
small writes (logs above all) wear cheap cards to death. See
[SD-card wear](/learn/embedded/sd-card-wear/)

**Wear-levelling** — A flash controller spreading writes across all cells so no
block dies early — done well in SSDs, barely in cheap cards. See
[SD-card wear](/learn/embedded/sd-card-wear/)

**Write amplification** — Small writes costing full flash-block erase cycles —
why a trickle of log lines wears more than its byte count suggests. See
[SD-card wear](/learn/embedded/sd-card-wear/)

**tmpfs** — A RAM-backed filesystem: contents vanish at reboot, wear cost zero —
where scratch data and (via log2ram) buffered logs belong. See
[SD-card wear](/learn/embedded/sd-card-wear/)

**Retention policy** — The rule bounding stored data (recordings pruned after N
days) so growth is flat instead of a countdown to a full disk. See
[SD-card wear](/learn/embedded/sd-card-wear/)

**Hardware watchdog** — A silicon countdown timer that reboots the board unless the
OS keeps resetting it — the escape from a hung kernel that software can't provide.
See [Watchdogs &amp; recovery](/learn/embedded/watchdogs-and-recovery/)

**Health check** — A scheduled probe of observable work (console answers, recordings
land) that restarts or alerts when the answer is no. See
[Watchdogs &amp; recovery](/learn/embedded/watchdogs-and-recovery/)

**Power-cycle test** — The design test for appliances: pull power, restore it, and
everything must resume unattended. See
[Watchdogs &amp; recovery](/learn/embedded/watchdogs-and-recovery/)

**SSH keys** — A cryptographic key pair replacing password login — stronger,
unphishable, and the standard for administering remote boards. See
[Remote administration](/learn/embedded/remote-administration/)

**tmux** — A terminal multiplexer keeping remote work alive when the connection
drops — where every long or risky remote command belongs. See
[Remote administration](/learn/embedded/remote-administration/)

**Golden image** — A full byte-for-byte backup of the working card, taken at
milestones — the ten-minute path back from any disaster. See
[Backups &amp; images](/learn/embedded/backups-and-images/)

**rsync** — The sync tool that copies changed files efficiently over SSH — how
configs and data flow off the board on a schedule. See
[Backups &amp; images](/learn/embedded/backups-and-images/)

**Vitals** — The four numbers worth watching on any board: temperature, load, disk,
and service state — plus whether the work itself is happening. See
[Monitoring your board](/learn/embedded/monitoring-your-board/)

**Alert fatigue** — The failure mode where alerts fire so often you ignore them; the
reason to alert only on actionable states. See
[Monitoring your board](/learn/embedded/monitoring-your-board/)

## The GopherTrunk appliance

**ARM build** — A binary compiled for the ARM architecture; GopherTrunk ships ARM
Linux builds, matched to the OS via `uname -m`. See
[Install GopherTrunk on a Pi](/learn/embedded/installing-gophertrunk-on-a-pi/)

**Web console** — GopherTrunk's browser interface, served over the network — the
reason the appliance itself never needs a display. See
[Install GopherTrunk on a Pi](/learn/embedded/installing-gophertrunk-on-a-pi/)

**Real-time load** — Work with a hard deadline: samples arrive continuously, and
falling behind means dropped samples, not lateness. See
[Tuning for small CPUs](/learn/embedded/tuning-for-small-cpus/)

**Dropped samples** — Radio samples lost because power, USB, or CPU failed to keep
the stream flowing — the common currency of SBC-SDR failure modes. See
[USB SDR gotchas](/learn/embedded/usb-sdr-gotchas/)

**Self-interference** — The board's own RF emissions (SoC clocks, supplies, cheap
cables) raising the noise floor of the SDR beside it — fixed with antenna distance
and shielded cables. See [USB SDR gotchas](/learn/embedded/usb-sdr-gotchas/)

**Port-forward** — A router rule exposing an internal service to the internet — the
convenient click an appliance's console must never receive. See
[Appliance networking &amp; access](/learn/embedded/appliance-networking/)

**VPN** — An encrypted, authenticated tunnel that puts your remote device virtually
on your home LAN — the right way to reach the console from anywhere. See
[Appliance networking &amp; access](/learn/embedded/appliance-networking/)

**SSH tunnel** — Carrying a service's port through an SSH connection — on-demand
remote access with zero new infrastructure. See
[Appliance networking &amp; access](/learn/embedded/appliance-networking/)

**Acceptance test** — The pass/fail check ending each build phase, proving its layer
before the next is stacked — the method that localizes every failure. See
[A complete appliance build](/learn/embedded/a-complete-appliance-build/)
