---
slug: glossary
title: Glossary of hardware terms
description: Plain-language definitions for every key hardware term in the Intro to Hardware learning path, cross-linked to the lessons that explain them.
keywords: hardware glossary, computer hardware terms, server hosting terms, single board computer glossary, microcontroller terms
level: beginner
status: full
lesson_standalone: true
---

# Glossary of hardware terms

This is a plain-language reference for the key terms used across the Intro to Hardware path. Definitions are short on purpose; each links to the lesson that explains the idea in full. Terms are grouped by theme and roughly ordered from foundational to advanced within each group, following the shape of the path itself.

## Foundations

**Hardware** — The physical parts of a computing device you can touch: chips, boards, memory, drives, and the case around them. See [What is computer hardware?](/learn/intro-hardware/what-is-hardware/)

**Software** — The instructions that run on hardware. Unlike hardware, it can be changed without replacing the machine. See [What is computer hardware?](/learn/intro-hardware/what-is-hardware/)

**Computer** — Any machine that takes input, processes it by following instructions, and produces output, from a data-centre server down to a tiny chip in a sensor. See [What is computer hardware?](/learn/intro-hardware/what-is-hardware/)

**CPU (central processing unit)** — The chip that carries out a program's instructions; the part most people call the "brain" of a computer. See [CPU, memory, storage & I/O](/learn/intro-hardware/building-blocks/)

**Core** — A self-contained processing unit inside a CPU; a multi-core chip can work on several tasks at once. See [CPU, memory, storage & I/O](/learn/intro-hardware/building-blocks/)

**Clock speed** — How many cycles per second a CPU runs, measured in gigahertz (GHz); a rough, not absolute, guide to how fast it works. See [CPU, memory, storage & I/O](/learn/intro-hardware/building-blocks/)

**GPU (graphics processing unit)** — A processor with thousands of small cores built for massively parallel math; used for graphics, scientific computing, and modern AI. See [Beyond the CPU — GPUs, FPGAs & accelerators](/learn/intro-hardware/specialized-compute/)

**FPGA (field-programmable gate array)** — A chip you configure into a custom digital circuit, giving deterministic, low-latency hardware — often used in the front end of high-end SDRs. See [Beyond the CPU — GPUs, FPGAs & accelerators](/learn/intro-hardware/specialized-compute/)

**DSP (digital signal processor)** — A processor specialized for signal math like filtering and transforms, common in radios, modems, and audio gear. See [Beyond the CPU — GPUs, FPGAs & accelerators](/learn/intro-hardware/specialized-compute/)

**ASIC / accelerator** — Fixed-function silicon built for one job at maximum efficiency, including AI accelerators (NPUs and TPUs). See [Beyond the CPU — GPUs, FPGAs & accelerators](/learn/intro-hardware/specialized-compute/)

**RAM (memory)** — Fast, temporary working storage that holds the data and programs a computer is actively using; it is cleared when power is lost. See [CPU, memory, storage & I/O](/learn/intro-hardware/building-blocks/)

**Storage** — Where data is kept long-term so it survives a power cycle, such as a hard drive or solid-state drive. See [CPU, memory, storage & I/O](/learn/intro-hardware/building-blocks/)

**SSD (solid-state drive)** — Storage that keeps data on flash memory chips with no moving parts, far faster and more rugged than a spinning hard drive. See [CPU, memory, storage & I/O](/learn/intro-hardware/building-blocks/)

**Flash memory** — Non-volatile storage that holds data without power, used in SSDs, memory cards, and the on-board storage of small devices. See [CPU, memory, storage & I/O](/learn/intro-hardware/building-blocks/)

**I/O (input/output)** — The channels through which a computer takes in and sends out data, from keyboards and screens to network ports and sensors. See [CPU, memory, storage & I/O](/learn/intro-hardware/building-blocks/)

**Operating system (OS)** — The software that manages a device's hardware and resources and gives other programs a consistent way to run. See [CPU, memory, storage & I/O](/learn/intro-hardware/building-blocks/)

**Hardware spectrum** — The full range of computing hardware, from distant cloud servers through personal devices down to tiny microcontrollers, each suited to different jobs. See [The hardware spectrum — cloud to microcontroller](/learn/intro-hardware/hardware-spectrum/)

**General-purpose computer** — A machine designed to run many different programs and adapt to many tasks, like a laptop or server. See [The hardware spectrum — cloud to microcontroller](/learn/intro-hardware/hardware-spectrum/)

**Embedded computer** — A computer built into a larger device to do one specific job, rather than serving as a general-purpose machine. See [The hardware spectrum — cloud to microcontroller](/learn/intro-hardware/hardware-spectrum/)

**Compiled language** — A language whose programs are translated to machine code ahead of time, common where performance and small footprints matter. See [Programming languages & where they run](/learn/intro-hardware/languages-and-hardware/)

**Interpreted language** — A language whose programs are read and run on the fly, convenient on capable hardware but heavier on resources. See [Programming languages & where they run](/learn/intro-hardware/languages-and-hardware/)

**Cost, power & performance** — The three competing pressures behind almost every hardware choice: what it costs, how much energy it draws, and how much work it can do. See [Cost, power & performance trade-offs](/learn/intro-hardware/cost-power-performance/)

**Trade-off** — A choice where gaining one quality means giving up another, such as more performance for higher cost or power draw. See [Cost, power & performance trade-offs](/learn/intro-hardware/cost-power-performance/)

## Servers and hosting

**Server** — A computer that provides services or data to other machines over a network, such as serving web pages or running an application. See [Web & shared hosting](/learn/intro-hardware/web-hosting/)

**Web hosting** — A service that runs your website on someone else's servers so it is reachable on the internet. See [Web & shared hosting](/learn/intro-hardware/web-hosting/)

**Shared hosting** — The cheapest kind of web hosting, where many customers' sites run together on one server and share its resources. See [Web & shared hosting](/learn/intro-hardware/web-hosting/)

**VPS (virtual private server)** — A slice of a physical server that acts like your own machine, giving more control than shared hosting at lower cost than a whole server. See [Virtual private servers (VPS)](/learn/intro-hardware/vps/)

**Virtualization** — Splitting one physical computer into several isolated virtual machines, each behaving like a separate computer. See [Virtual private servers (VPS)](/learn/intro-hardware/vps/)

**Hypervisor** — The software layer that creates and runs virtual machines on a physical host, dividing its resources between them. See [Virtual private servers (VPS)](/learn/intro-hardware/vps/)

**Root access** — Full administrative control over a server, letting you install software and change any setting. See [Virtual private servers (VPS)](/learn/intro-hardware/vps/)

**Dedicated server** — A whole physical server rented for your exclusive use, with no other customers sharing its hardware. See [Dedicated servers](/learn/intro-hardware/dedicated-servers/)

**Home server** — A computer you run at home to host services for yourself, from file storage to a personal website. See [Home servers & self-hosting](/learn/intro-hardware/home-servers/)

**Self-hosting** — Running services on hardware you own and control instead of paying a provider to run them for you. See [Home servers & self-hosting](/learn/intro-hardware/home-servers/)

**NAS (network-attached storage)** — A dedicated device that stores files and shares them with other machines over a home or office network. See [Home servers & self-hosting](/learn/intro-hardware/home-servers/)

**Port forwarding** — A router setting that lets traffic from the internet reach a specific device on your home network, needed to expose a self-hosted service. See [Home servers & self-hosting](/learn/intro-hardware/home-servers/)

## Personal and mobile devices

**Desktop** — A stationary personal computer, typically more powerful, expandable, and cheaper per unit of performance than a laptop. See [Desktop computers](/learn/intro-hardware/desktop-computers/)

**Laptop** — A portable personal computer with screen, keyboard, and battery built in, trading some performance and expandability for mobility. See [Laptops](/learn/intro-hardware/laptops/)

**Thermal throttling** — When a chip deliberately slows itself to avoid overheating, a common limit on sustained performance in thin laptops and small devices. See [Laptops](/learn/intro-hardware/laptops/)

**Development machine** — The computer you write and test code on; choosing one means balancing power, portability, and budget against how you work. See [Choosing your development machine](/learn/intro-hardware/choosing-a-dev-machine/)

**Smartphone** — A pocket-sized, touchscreen computer with a cellular connection, sensors, and a battery, running apps as well as making calls. See [Smartphones](/learn/intro-hardware/smartphones/)

**Tablet** — A larger touchscreen device that sits between a phone and a laptop, good for media and light work and sometimes paired with a keyboard. See [Tablets](/learn/intro-hardware/tablets/)

**Native app** — An application built specifically for one platform's hardware and operating system, giving full access to its features and best performance. See [Developing for mobile devices](/learn/intro-hardware/developing-for-mobile/)

**Cross-platform** — Building software that runs on more than one kind of device or operating system from largely shared code. See [Developing for mobile devices](/learn/intro-hardware/developing-for-mobile/)

**PWA (progressive web app)** — A website built to behave like an installed app, working offline and launching from the home screen without an app store. See [Developing for mobile devices](/learn/intro-hardware/developing-for-mobile/)

## Single-board computers and microcontrollers

**Single-board computer (SBC)** — A complete computer built on one small circuit board, with CPU, memory, storage, and ports, capable of running a full operating system. See [What is a single-board computer?](/learn/intro-hardware/what-is-an-sbc/)

**GPIO (general-purpose input/output)** — Pins on an SBC or microcontroller that your code can read or switch on and off to talk to sensors, lights, motors, and other electronics. See [What is a single-board computer?](/learn/intro-hardware/what-is-an-sbc/)

**Raspberry Pi** — A popular, low-cost single-board computer widely used for learning, hobby projects, and small servers. See [The Raspberry Pi & its alternatives](/learn/intro-hardware/raspberry-pi-and-family/)

**HAT (hardware attached on top)** — An add-on board that stacks onto a Raspberry Pi's pins to give it extra features like displays, sensors, or power options. See [The Raspberry Pi & its alternatives](/learn/intro-hardware/raspberry-pi-and-family/)

**Programming an SBC** — Writing software for a single-board computer much as you would for a desktop, using its operating system and ordinary languages and tools. See [Programming & running software on an SBC](/learn/intro-hardware/programming-an-sbc/)

**Edge computing** — Running computation close to where data is produced, on devices or local hardware like an SBC, rather than sending everything to a distant data centre. See [SBC use cases, strengths & drawbacks](/learn/intro-hardware/sbc-use-cases-and-limits/)

**Microcontroller (MCU)** — A tiny, low-power computer on a single chip that combines processor, memory, and I/O, dedicated to controlling one device or task. See [What is a microcontroller?](/learn/intro-hardware/what-is-a-microcontroller/)

**Bare metal** — Running code directly on a chip with no operating system in between, common on microcontrollers for tight control and predictability. See [What is a microcontroller?](/learn/intro-hardware/what-is-a-microcontroller/)

**Firmware** — Low-level software loaded onto a device's chip to control its hardware directly, sitting between pure hardware and ordinary programs. See [What is a microcontroller?](/learn/intro-hardware/what-is-a-microcontroller/)

**Arduino** — A widely used family of beginner-friendly microcontroller boards and the open-source ecosystem of tools and code around them. See [Arduino & the maker ecosystem](/learn/intro-hardware/arduino/)

**Sketch** — The name Arduino gives to a program written for its boards. See [Arduino & the maker ecosystem](/learn/intro-hardware/arduino/)

**ESP32** — A low-cost microcontroller with built-in Wi-Fi and Bluetooth, popular for connected and wireless projects. See [ESP32 & wireless microcontrollers](/learn/intro-hardware/esp32/)

**MicroPython** — A compact version of the Python language made to run directly on microcontrollers. See [Programming microcontrollers: languages, use cases & limits](/learn/intro-hardware/mcu-programming-and-tradeoffs/)

**Real-time** — A requirement that a device respond within strict, guaranteed time limits, where being late is itself a failure; common in microcontroller work. See [Programming microcontrollers: languages, use cases & limits](/learn/intro-hardware/mcu-programming-and-tradeoffs/)

**Deep sleep** — A very low-power mode a microcontroller drops into between tasks to stretch battery life, waking only when needed. See [Programming microcontrollers: languages, use cases & limits](/learn/intro-hardware/mcu-programming-and-tradeoffs/)

## Choosing hardware

**Requirements** — What your project must actually do and the conditions it must meet, worked out before deciding what hardware to buy. See [Start with your requirements](/learn/intro-hardware/start-with-requirements/)

**Constraint** — A fixed limit your choice must respect, such as budget, size, power supply, or response time. See [Start with your requirements](/learn/intro-hardware/start-with-requirements/)

**Project type** — The broad category of what you are building (a website, a mobile app, a sensor, a home server), which steers you toward fitting hardware. See [Matching hardware to the project type](/learn/intro-hardware/matching-hardware-to-project/)

**Combining tiers** — Using device, server, and cloud hardware together in one system, with each tier handling the part of the job it suits best. See [Combining tiers: device, server & cloud](/learn/intro-hardware/combining-tiers/)

**Cloud** — Computing power and storage provided over the internet on demand, instead of from machines you own and run yourself. See [Combining tiers: device, server & cloud](/learn/intro-hardware/combining-tiers/)

**Latency** — The delay between asking for something and getting a response; keeping it low is a common reason to move work closer to the user. See [Combining tiers: device, server & cloud](/learn/intro-hardware/combining-tiers/)

**Scaling** — Handling more work by adding capacity, whether by upgrading a machine or by adding more machines. See [Combining tiers: device, server & cloud](/learn/intro-hardware/combining-tiers/)

**Decision framework** — A structured way to weigh hardware options against your needs so the choice is reasoned rather than arbitrary. See [A practical decision framework](/learn/intro-hardware/decision-framework/)

**Total cost of ownership** — The full cost of a hardware choice over time, including power, maintenance, and replacement, not just the sticker price. See [A practical decision framework](/learn/intro-hardware/decision-framework/)

**Worked example** — A realistic scenario walked through end to end to show how the decision framework picks hardware for a real project. See [Worked examples: picking hardware for real projects](/learn/intro-hardware/worked-examples/)
