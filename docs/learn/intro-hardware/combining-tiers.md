---
slug: combining-tiers
title: "Combining tiers: device, server & cloud"
description: Real systems span tiers — tiny edge devices, a local server, and the cloud. Learn how they divide the work and how to decide what runs where.
keywords: edge vs cloud, multi-tier architecture, edge computing, IoT gateway, device server cloud, where to run code, latency and bandwidth, edge aggregation
level: intermediate
status: full
faq:
  - q: "What does \"edge vs cloud\" mean?"
    a: "The **edge** is hardware near where the action happens — a sensor, a gateway in the building, a Pi at the antenna. The **cloud** is shared servers in a data center far away. Edge wins on latency, bandwidth, and working offline; cloud wins on storage, heavy compute, and being reachable from anywhere. Most real systems use both, pushing each piece of work to the tier that fits it best."
  - q: "Why not just put everything in the cloud?"
    a: "Because some work can't move. Anything touching physical hardware — a [microcontroller](/learn/intro-hardware/what-is-a-microcontroller/) reading a sensor, a USB radio dongle — must run on a device that's physically there. And pushing raw high-rate data to the cloud can be slow, expensive, or impossible on a flaky link. You keep that work at the edge and send only the distilled results up."
  - q: "What's the job of the middle tier — a local server or gateway?"
    a: "It aggregates. Edge devices are cheap and dumb on purpose; a [home server](/learn/intro-hardware/home-servers/) or [SBC](/learn/intro-hardware/what-is-an-sbc/) gateway collects their data, stores it locally, runs the logic that ties them together, and decides what's worth forwarding to the cloud. It also keeps the system working when the internet is down."
  - q: "How do I decide what runs where?"
    a: "Push each task to the cheapest tier that can do it well. Time-critical or physical work goes to the **edge**; coordination and local storage go to a **local server**; durable storage, heavy compute, and remote access go to the **cloud**. Latency, bandwidth, power, cost, and offline resilience are the levers that decide each one."
---
# Combining tiers: device, server & cloud

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Real systems span tiers** — tiny edge devices, a local server or gateway, and the cloud each do part of the job. **Push work to where it fits** — physical and time-critical tasks stay at the edge; storage, heavy compute, and remote access go to the cloud. **The middle tier aggregates** — a local server collects from cheap devices, keeps working offline, and forwards only what matters.
</div>

The previous lessons treated each project as landing on one platform. Reality is messier and more interesting: most non-trivial systems are *layered*, with different tiers of hardware cooperating. A swarm of cheap microcontrollers, a local server that gathers their output, and a cloud account for storage and remote access is an extremely common shape. This lesson is about that shape — why it exists, and how to decide which tier does what.

## The three tiers

Think of hardware in a real system as three rough tiers, from the physical world inward:

- **Edge devices** sit where the action is. They're cheap, low-power, and often single-purpose — an [ESP32](/learn/intro-hardware/esp32/) reading a sensor, a [microcontroller](/learn/intro-hardware/what-is-a-microcontroller/) driving a motor, a [Raspberry Pi](/learn/intro-hardware/raspberry-pi-and-family/) with a camera. Their defining trait is *physical presence*: they touch the world directly.
- **A local server or gateway** sits in the same building or site. Usually a [home server](/learn/intro-hardware/home-servers/) or [SBC](/learn/intro-hardware/what-is-an-sbc/), it aggregates data from the edge devices, runs the logic that connects them, stores recent history locally, and decides what to send onward. It keeps the system alive when the internet drops.
- **The cloud** is shared servers far away — a [VPS](/learn/intro-hardware/vps/) or managed service. It's the home for durable long-term storage, heavy compute that the edge can't do, and the public-facing parts that let you reach the system from anywhere.

Not every system needs all three. A single sensor reporting to a phone is one tier and a half. But once a project has several devices, or needs both local responsiveness and remote access, the layering pays off.

## Edge vs cloud: why work moves

The central decision in a layered system is *where each piece of work runs*. Five forces pull the answer one way or the other:

- **Latency** — a control loop that must react in milliseconds can't wait for a round trip to a data center. It stays at the edge.
- **Bandwidth** — raw high-rate data (audio, video, radio samples) is expensive or impossible to ship upstream continuously. Process it locally and send only the summary.
- **Power** — battery devices can't afford to keep a radio transmitting constantly; they do the minimum locally and sleep.
- **Cost** — cloud compute and egress are metered. Doing work on hardware you already own at the edge is often far cheaper.
- **Offline resilience** — if the link goes down, edge and local tiers must keep functioning. Anything that *must* keep working can't depend on the cloud.

The rule of thumb: **push each task to the cheapest tier that can do it well.** Physical and time-critical work to the edge, coordination and local storage to the middle, durable storage and remote reach to the cloud.

## What runs where

| Task | Best tier | Why |
|------|-----------|-----|
| Reading a sensor / driving an actuator | **Edge device** | Must be physically attached; needs tight timing |
| Reacting in real time (control loop) | **Edge device** | A cloud round trip is too slow |
| Heavy local data reduction (e.g. decode a stream) | **Edge / local server** | Saves bandwidth by shipping results, not raw data |
| Aggregating several devices, local logic | **Local server / gateway** | One place to coordinate; works offline |
| Recent history, local dashboard | **Local server** | Fast, private, no internet dependency |
| Durable long-term storage, backups | **Cloud** | Reliable, off-site, scales without buying disks |
| Remote access from anywhere | **Cloud** | A public, always-reachable endpoint |
| Occasional heavy compute (ML, reports) | **Cloud** | Rent burst power instead of owning it |

## A worked architecture: home sensor network

Picture a home environment monitor. A handful of **ESP32** nodes around the house each read temperature and humidity and, every few minutes, wake, send a reading over Wi-Fi, and sleep — that's the edge, sipping battery. A **Raspberry Pi** in a closet runs all the time as the gateway: it receives every reading, stores months of history on a local disk, serves a dashboard on the home network, and runs the automation rules ("if the basement gets damp, turn on the fan"). A small **cloud VPS** holds off-site backups and exposes a single secure endpoint so you can check the house from your phone while traveling. Three tiers, each doing what it's best at — and the house keeps regulating itself even if the internet is down, because the logic lives on the Pi, not in the cloud.

## GopherTrunk across tiers

GopherTrunk fits this model almost perfectly. The **edge** is a [Raspberry Pi](/learn/intro-hardware/raspberry-pi-and-family/) (or a PC) sitting right at the antenna with the **SDR dongle plugged into its USB port** — it has to be physically there, and it does the bandwidth-heavy work of capturing and decoding the radio stream locally, because shipping raw radio samples to a data center would be absurd. A **server** — that same Pi, a beefier [home server](/learn/intro-hardware/home-servers/), or a [VPS](/learn/intro-hardware/vps/) — stores the decoded recordings and serves them up. Your **phone** is the dashboard client, a browser pointed at that server from anywhere. This is also exactly *why* a cloud-only setup can't work: the cloud can store and serve recordings happily, but it can never be the capture tier, because you cannot plug a USB dongle into a machine you don't physically touch. The physical-world requirement pins the edge tier in place, and everything else layers around it.

<div class="knowledge-check" data-quiz data-correct-msg="Right — physical, time-critical work belongs at the edge; the cloud handles storage and remote access." markdown="0">
  <p class="knowledge-check__q">Quick check: In a layered system, which tier should handle reading a sensor and reacting to it in real time?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">The cloud, so all the logic lives in one place</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">The edge device, which is physically attached and avoids the cloud round trip</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Whichever tier is cheapest, regardless of latency</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Systems span tiers** — edge devices, a local server or gateway, and the cloud divide the work between them.
- **Push work to where it fits** — physical and time-critical tasks to the edge, coordination and local storage to the middle, durable storage and remote access to the cloud.
- **Five forces decide placement** — latency, bandwidth, power, cost, and offline resilience.
- **The middle tier aggregates** — a local server collects from cheap devices and keeps working when the internet is down.
- **GopherTrunk is layered by necessity** — the Pi at the antenna captures and decodes, a server stores and serves, a phone is the client — and the cloud can never be the capture tier.

Next up: turn all of this into a repeatable method you can run every time. See [A practical decision framework](/learn/intro-hardware/decision-framework/).
