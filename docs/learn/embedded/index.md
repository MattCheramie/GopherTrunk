---
layout: learn-hub
learn_module: embedded
permalink: /learn/embedded/
title: Learn Embedded & Single-Board Computers — from newbie to expert
description: A free, structured module on embedded systems and single-board computers — what SBCs and microcontrollers are, choosing and setting up a board, running it headless, connecting hardware, keeping it alive 24/7, and finishing with a Raspberry Pi GopherTrunk scanner appliance.
keywords: embedded systems, single-board computer, SBC, Raspberry Pi, microcontroller, headless Linux, systemd service, GPIO, RTL-SDR on Raspberry Pi, scanner appliance, SD card, undervoltage, thermal throttling
---

Most of the computers around you don't look like computers. They live inside routers,
thermostats, cars, and radios — small boards doing one job, all day, with nobody
watching. This module is about those machines: what **embedded systems** and
**single-board computers** are, how to choose one, and how to run one like a tiny,
reliable server. The destination is concrete: by the end you'll know how to build the
classic [GopherTrunk]({{ '/' | relative_url }}) project — a Raspberry Pi that decodes
trunked radio around the clock.

**Who this is for.** Anyone who has heard of the Raspberry Pi and wondered what people
actually *do* with one — or who has a project (like a radio scanner) that wants a
dedicated, always-on home. It assumes no hardware background; a little command-line
comfort helps, and the [Linux &amp; the Command Line]({{ '/learn/linux-cli/' | relative_url }})
module fills any gaps. It pairs naturally with
[Computer Hardware]({{ '/learn/intro-hardware/' | relative_url }}) (the wider hardware
landscape) and [RF &amp; SDR]({{ '/learn/rf-sdr/' | relative_url }}) (the radio side of
the final build).

**How the path works.** Six units take you from "what even is an embedded system?" to a
finished appliance. Unit 1 maps the **small-computer landscape** — SBCs, microcontrollers,
and the ARM chips inside them. Unit 2 covers the **buying decisions**: boards, storage,
power, and cooling. Unit 3 teaches **headless Linux** — flash a card, SSH in, run
services. Unit 4 connects the board to the **physical world** through GPIO, USB, and the
little serial buses. Unit 5 is **reliability engineering**: heat, SD-card wear,
watchdogs, backups, and monitoring. Unit 6 puts it all together as the **GopherTrunk
appliance** — an ARM build of the daemon, an RTL-SDR on USB, and a web console on your
LAN. Mark lessons complete as you go — your progress is saved in your browser. New here?
**[Start with lesson 1: What is an embedded system?](/learn/embedded/what-is-embedded/)**
