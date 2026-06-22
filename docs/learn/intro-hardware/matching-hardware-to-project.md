---
slug: matching-hardware-to-project
title: Matching hardware to the project type
description: Websites, mobile apps, data crunching, home automation, sensors, and robots each tend toward a different platform. A practical map from project type to the hardware that fits.
keywords: hardware for web app, hardware for mobile app, hardware for machine learning, home automation hardware, sensor hardware, robotics hardware, project to platform mapping, choosing a platform
level: intermediate
status: full
faq:
  - q: "What hardware should I use to host a website?"
    a: "Start at the smallest rung that meets the load. A static or low-traffic site is happy on [shared web hosting](/learn/intro-hardware/web-hosting/); a real app with a database wants a [VPS](/learn/intro-hardware/vps/); only heavy, sustained traffic justifies a [dedicated server](/learn/intro-hardware/dedicated-servers/). Most projects never need to leave the first two rungs."
  - q: "Where do I run a machine-learning or data-crunching workload?"
    a: "On something with real muscle, often a GPU. A powerful [desktop](/learn/intro-hardware/desktop-computers/) is the cheapest fast option you own outright; a [dedicated server](/learn/intro-hardware/dedicated-servers/) or rented cloud GPU is right when you need more than one machine's worth, or only burst usage. The job is compute-bound, so this is the one place raw performance genuinely dominates the choice."
  - q: "What's the right platform for a battery-powered sensor?"
    a: "A [microcontroller](/learn/intro-hardware/what-is-a-microcontroller/), almost always — typically an [ESP32](/learn/intro-hardware/esp32/) if it needs Wi-Fi. It sips power, costs a few dollars, wakes to take a reading, and sleeps. A full computer would drain the battery and waste capability the job never uses."
  - q: "Why not just use the most powerful platform for everything?"
    a: "Because power you don't need is cost, complexity, and wasted electricity you pay for forever. The skill is matching the platform to the *type* of work — a microcontroller for a sensor, a VPS for a web app, a GPU box for ML — so each project runs on the smallest thing that does the job well."
---
# Matching hardware to the project type

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Project type strongly predicts platform** — websites reach for hosting and servers, sensors reach for microcontrollers, ML reaches for GPUs. **The mapping isn't rigid but it's a fast first cut** — start where your kind of project usually lands, then adjust for your specifics. **Smallest thing that does the job** — match the platform to the work, don't over-buy capability you'll never use.
</div>

In the last lesson you turned a project into a list of requirements. This lesson shortcuts the next step: most projects fall into a handful of familiar *types*, and each type tends to reach for the same region of the [hardware spectrum](/learn/intro-hardware/hardware-spectrum/). Knowing those patterns gets you to a strong default fast, which you can then refine. Let's walk the common types and where they land.

## Websites and web apps

A website's hardware is a question of traffic and dynamism. A static site or a small blog needs almost nothing — [shared web hosting](/learn/intro-hardware/web-hosting/) serves it for a few dollars a month, with someone else handling the machine entirely. Add a database, user accounts, and server-side logic and you've outgrown shared hosting; a [VPS](/learn/intro-hardware/vps/) gives you a real, isolated Linux box you control, which is where the large majority of web apps live comfortably for years. Only sustained heavy traffic, or a need for raw single-tenant performance, justifies stepping up to a [dedicated server](/learn/intro-hardware/dedicated-servers/). The discipline here is resisting the jump: people routinely rent far more than their traffic warrants.

## Mobile apps

A mobile app splits cleanly across two machines. You *develop* it on a capable [laptop](/learn/intro-hardware/laptops/) or [desktop](/learn/intro-hardware/desktop-computers/) — the build tools, simulators, and IDEs are demanding, and an iOS app specifically requires a Mac (see [choosing a dev machine](/learn/intro-hardware/choosing-a-dev-machine/) and [developing for mobile](/learn/intro-hardware/developing-for-mobile/)). The app then *runs* on the user's [phone](/learn/intro-hardware/smartphones/) or [tablet](/learn/intro-hardware/tablets/), which you don't choose — your users bring their own. If the app has a backend, that's a separate web-app decision: hosting, VPS, or server as above.

## Data crunching and machine learning

This is the one project type where raw performance honestly dominates. Training models, processing large datasets, or rendering all want maximum throughput, and increasingly a GPU. A well-specced [desktop](/learn/intro-hardware/desktop-computers/) with a strong GPU is the cheapest fast option you own outright, ideal for steady local work. When you need more than one machine, only occasional bursts, or hardware you'd rather rent than buy, a [dedicated server](/learn/intro-hardware/dedicated-servers/) or a cloud GPU instance fits. Here, more compute really does buy more — the opposite of most projects, where it's just waste.

## Home automation and always-on services

Automation hubs, media servers, network storage, and personal dashboards share a profile: modest compute, but they must run quietly, cheaply, and *constantly*. That's the sweet spot for a [home server](/learn/intro-hardware/home-servers/) — a [single-board computer](/learn/intro-hardware/what-is-an-sbc/) like a [Raspberry Pi](/learn/intro-hardware/raspberry-pi-and-family/), or a repurposed mini-PC. Low power draw matters because the thing never turns off, so a 5-watt board beats a 200-watt tower over a year of electricity bills. A cloud VPS can host the *software* parts, but anything touching local devices needs a box in the house.

## Physical sensors and gadgets

A device that senses or controls the physical world — temperature, soil moisture, a button, a relay — and especially one on a battery, is a [microcontroller](/learn/intro-hardware/what-is-a-microcontroller/) job. An [Arduino](/learn/intro-hardware/arduino/) suits simple offline control; an [ESP32](/learn/intro-hardware/esp32/) is the default when you want built-in Wi-Fi to report readings. These parts cost a few dollars, run for months on a battery, and do exactly one job reliably. A full computer would be overkill that drains the battery and adds failure points.

## Robots and edge-AI vision

Robotics and on-device vision sit one rung up from sensors: they need real compute *and* physical-world contact at once. A [single-board computer](/learn/intro-hardware/what-is-an-sbc/) is the natural fit — a Raspberry Pi for general robotics, or a board like an NVIDIA Jetson when the job involves running vision or ML models locally at the edge. Often these pair a capable SBC for the "brain" with microcontrollers for low-level motor and sensor timing, a combination the next lesson explores.

## The mapping at a glance

| Project type | Typical platform | Why it fits |
|--------------|------------------|-------------|
| **Static site / blog** | [Shared web hosting](/learn/intro-hardware/web-hosting/) | Cheap, zero machine management, plenty for low traffic |
| **Web app with a database** | [VPS](/learn/intro-hardware/vps/) | Full control of a real Linux box; scales to most apps |
| **High-traffic / heavy web service** | [Dedicated server](/learn/intro-hardware/dedicated-servers/) | Single-tenant performance, no noisy neighbors |
| **Mobile app** | [Dev machine](/learn/intro-hardware/choosing-a-dev-machine/) → user's [phone](/learn/intro-hardware/smartphones/) | Build on a capable PC/Mac; runs on user devices |
| **Data crunching / ML** | Powerful [desktop](/learn/intro-hardware/desktop-computers/) or cloud GPU | Compute-bound — raw throughput and a GPU pay off |
| **Home automation / always-on** | [Home server](/learn/intro-hardware/home-servers/) / [SBC](/learn/intro-hardware/what-is-an-sbc/) | Low power, runs 24/7, sits near local devices |
| **Battery sensor / gadget** | [ESP32](/learn/intro-hardware/esp32/) / [microcontroller](/learn/intro-hardware/what-is-a-microcontroller/) | Tiny, cheap, ultra-low power, touches the physical world |
| **Robot / edge-AI vision** | [SBC](/learn/intro-hardware/what-is-an-sbc/) (Pi, Jetson) | Real compute plus on-site sensing and control |

GopherTrunk is a useful edge case to keep in mind: it's web-app-*like* (it serves a dashboard you view from a browser) but it also has a hard physical-world requirement — the SDR dongle plugged in at the antenna. That mix is exactly why it doesn't sit cleanly in one row, and why the next lesson on combining tiers matters.

<div class="knowledge-check" data-quiz data-correct-msg="Right — a battery-powered sensor is a classic microcontroller job, typically an ESP32 when it needs Wi-Fi." markdown="0">
  <p class="knowledge-check__q">Quick check: What platform best fits a battery-powered soil-moisture sensor that reports over Wi-Fi?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">A cloud VPS</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">A microcontroller such as an ESP32</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">A high-end desktop with a GPU</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Project type predicts platform** — each common kind of project tends toward the same region of the spectrum, giving you a fast, strong default.
- **Websites climb a ladder** — shared hosting, then VPS, then dedicated, only as traffic and complexity demand.
- **ML is the exception that loves power** — data crunching is compute-bound, so a strong desktop or cloud GPU genuinely pays off.
- **Always-on and physical jobs stay home** — home servers, SBCs, and microcontrollers sit near the devices they serve.
- **Match the work, don't over-buy** — pick the smallest platform that does the job type well; the mapping table is your shortcut.

Next up: real systems rarely live on one platform — see how a device, a server, and the cloud work together in [Combining tiers: device, server & cloud](/learn/intro-hardware/combining-tiers/).
