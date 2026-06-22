---
slug: smartphones
title: Smartphones
description: A smartphone is a powerful ARM computer packed with sensors. Learn what it can do, the platforms and languages behind apps, and why it's a deployment target rather than a dev machine.
keywords: what is a smartphone, smartphone hardware, ARM SoC, iOS development, Android development, Swift, Kotlin, React Native, Flutter, mobile sensors, smartphone as a computer
level: beginner
status: full
faq:
  - q: "Is a smartphone a real computer?"
    a: "Yes, and a remarkably capable one. A modern phone has a multi-core **ARM system-on-chip**, several gigabytes of RAM, fast storage, and more raw performance than a desktop from a decade ago. What sets it apart is the package around the chip: a touch screen, a battery, and a dense cluster of sensors and radios. It is a full general-purpose computer that happens to fit in a pocket."
  - q: "What languages are used to build smartphone apps?"
    a: "It depends on the platform. **iOS** apps are written mainly in **Swift** (with SwiftUI) and older code in Objective-C. **Android** apps are written in **Kotlin** or Java. If you want one codebase for both, cross-platform frameworks like **Flutter** (Dart) and **React Native** (JavaScript) let you target iOS and Android together, trading a little native polish for shared effort."
  - q: "Can I use a smartphone as my development machine?"
    a: "Not really, and not for long. Phones are locked down by design — you can't freely install compilers, system tools, or arbitrary software, and the small screen and on-screen keyboard make sustained coding painful. Phones are something you **build for**, not something you build **on**. You write code on a laptop or desktop and deploy it to the phone."
---
# Smartphones

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**A smartphone is a powerful ARM computer with rich sensors** — GPS, cameras, an accelerometer, and several radios, all behind a touch screen and a battery. **It's the dominant computing platform on earth**, which makes it a deployment target you reach millions of people through. **You build for a phone, not on it** — phones are locked down and awkward to develop on, so the work happens on a desktop and ships to the device.
</div>

This lesson opens Module 4, on the computers most people actually carry. The smartphone is the most widespread computer ever made, and for a developer it plays a very particular role: it is somewhere your software *runs*, not somewhere you *write* software. By the end you'll understand what the hardware can do, which platforms and languages produce apps for it, and why "target, not workstation" is the right mental model.

## What a smartphone actually is

Underneath the glass, a smartphone is a small, dense computer built around an **ARM system-on-chip (SoC)** — a single piece of silicon that combines the CPU, graphics, memory controllers, and specialized accelerators. That efficiency-first ARM design is what lets a phone deliver serious performance while running on a battery.

What makes a phone distinct from other computers is everything bolted around that chip:

- A **touch screen** as the primary input and output, with no keyboard or mouse.
- A **battery** that forces every part of the system to care about power.
- A cluster of **sensors**: GPS for location, one or more **cameras**, an **accelerometer** and gyroscope for motion and orientation, plus a microphone, ambient light, and often more.
- A stack of **radios**: cellular, Wi-Fi, Bluetooth, NFC, and GPS receivers.

That last two points are the magic. A laptop can compute, but a phone *senses the world and stays connected to it* wherever you go. Apps that know where you are, what you're pointing the camera at, and how the device is moving are built directly on this sensor hardware.

## What smartphones are for

The obvious answer is consumer apps — messaging, maps, banking, media, games — and that alone makes the phone the dominant computing platform for most of humanity. For many people it is their *only* computer. But the same hardware serves quieter, more technical roles too:

- **IoT controllers.** A phone app is the standard way to set up and control a smart bulb, a thermostat, or a drone, talking over Bluetooth or Wi-Fi.
- **Field data capture.** GPS plus camera plus a form makes the phone an excellent tool for inspections, surveys, and logging readings where a laptop would be clumsy.
- **Thin clients and dashboards.** Because phones are always connected, they make great windows onto software running elsewhere. With GopherTrunk, for instance, a scanner can run on a Raspberry Pi at the antenna while you watch and steer it from a phone on the same network — the heavy work stays on the server, the phone is just the remote screen.

The thread through all of these: the phone's value comes from its sensors, its screen, and its constant connection, not from raw compute alone.

## Platforms, languages, and toolchains

Building for phones means picking a platform and its toolchain. There are two native ecosystems plus a cross-platform option that spans both.

| Platform | Primary languages | Toolchain | Notes |
|----------|-------------------|-----------|-------|
| **iOS** (Apple) | Swift (with SwiftUI), Objective-C | Xcode on a Mac | Native development requires a Mac; distribution goes through the App Store |
| **Android** (Google) | Kotlin (preferred), Java | Android Studio | Runs on Windows, macOS, or Linux; more open distribution |
| **Cross-platform** | Dart (Flutter), JavaScript/TypeScript (React Native) | Flutter / RN tooling | One codebase targeting both iOS and Android |

Native code gives you the fullest access to the device and the most polished platform feel. Cross-platform frameworks let a single team ship to both stores from one codebase, trading a little native fidelity for a lot of shared effort. We compare these approaches in depth in [Developing for mobile devices](/learn/intro-hardware/developing-for-mobile/). The key point for now: in every case you write the code on a real computer and the toolchain *builds and signs* an app that gets installed on the phone.

## Strengths

The smartphone's advantages are hard to overstate:

- **Ubiquity.** Billions of people carry one. No other platform offers comparable reach.
- **Sensors.** GPS, cameras, motion sensors, and radios come built in, ready for apps that perceive and respond to the physical world.
- **Always connected.** Cellular and Wi-Fi mean an app can assume a network most of the time, which makes the phone an ideal client for cloud and server software.
- **Reach with good UX.** App stores, push notifications, and a mature design language let a small team put a polished, personal experience directly in someone's pocket.

For getting software in front of ordinary people, nothing else competes.

## Drawbacks

The same traits that make phones great products make them frustrating as machines you control:

- **Locked down.** You don't fully own the software environment. Installing apps means going through a vetted store, and the operating system tightly limits what apps may touch.
- **App-store gatekeeping.** Apple and Google review submissions, take a cut, and can reject or remove apps. Your access to users runs through their rules.
- **Poor as a dev machine.** No real compilers, a tiny screen, and an on-screen keyboard make sustained development impractical. Phones are built to *consume and run*, not to *build*.
- **Battery and thermal limits.** Sustained heavy compute drains the battery and triggers throttling. The phone is powerful in bursts, not as a tireless workhorse.

None of these are flaws exactly — they're the price of a sealed, secure, pocket-sized appliance. They just explain why the phone sits on the *deployment* side of the line.

<div class="knowledge-check" data-quiz data-correct-msg="Right — a phone is something you build for; the actual development happens on a desktop or laptop." markdown="0">
  <p class="knowledge-check__q">Quick check: In a typical workflow, what role does a smartphone play for a developer?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">It's the main machine you write and compile code on</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">It's a deployment target — you build for it on a desktop and install the app onto it</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">It can't run real software and is only useful for phone calls</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **A phone is a powerful ARM computer** — a capable SoC, lots of RAM and storage, wrapped in a touch screen and battery.
- **Its edge is sensors and connectivity** — GPS, cameras, motion, and radios make it perceive the world and stay online.
- **It's the dominant platform** — consumer apps, IoT control, field capture, and thin-client dashboards all run on it.
- **Two native stacks plus cross-platform** — iOS (Swift), Android (Kotlin), and Flutter/React Native for both at once.
- **Target, not workstation** — locked down, store-gated, and awkward to code on, so you build for the phone elsewhere.

Next up: the phone's bigger-screened cousin, and where that extra size actually pays off. See [Tablets](/learn/intro-hardware/tablets/).
