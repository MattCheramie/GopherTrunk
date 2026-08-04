---
slug: nvidia-jetson
title: NVIDIA Jetson
entry_type: hardware
category: hw-sbc
description: NVIDIA Jetson is a family of single-board computers with a powerful onboard GPU, aimed at on-device AI and computer vision such as edge inference and robotics.
keywords: NVIDIA Jetson, Jetson Nano, Jetson Orin, edge AI, GPU SBC, computer vision, edge inference, robotics, CUDA, JetPack
autolink: true
affiliate: true
product:
  name: "NVIDIA Jetson Orin Nano"
  brand: NVIDIA
  category: Single-board computer
  lowPrice: "220"
  highPrice: "280"
  url: https://www.amazon.com/dp/B0BZJTQ5YP?tag=gophertrunk-20
infobox:
  - { label: Type, value: Single-board computer (GPU) }
  - { label: CPU, value: ARM + NVIDIA GPU }
  - { label: RAM, value: ~4 GB – 64 GB }
  - { label: Runs, value: Linux (JetPack) }
  - { label: Typical price, value: ~$100 – $2000+ }
  - { label: Buy, value: "<a class=\"btn btn--buy\" href=\"https://www.amazon.com/dp/B0BZJTQ5YP?tag=gophertrunk-20\" rel=\"nofollow sponsored noopener\">View on Amazon &rarr;</a>" }
see_also: [single-board-computer, raspberry-pi, beaglebone, gpio, central-processing-unit, edge-ai]
related_lessons:
  - { title: "Raspberry Pi and family", url: /learn/intro-hardware/raspberry-pi-and-family/ }
  - { title: "SBC use cases and limits", url: /learn/intro-hardware/sbc-use-cases-and-limits/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Nvidia_Jetson
faq:
  - q: "Should I buy a Jetson to run GopherTrunk?"
    a: "No — for plain trunk-following it is overkill. GopherTrunk is pure Go and uses the CPU only; it does not touch CUDA or the Jetson's GPU at all. A ~$55–80 Raspberry Pi decodes exactly the same channels. Buy a Jetson only if you also want its GPU for other work — signal-classification, machine-learning analysis of decoded traffic, or computer vision on the same node."
  - q: "Does GopherTrunk use the Jetson's GPU or CUDA?"
    a: "No. GopherTrunk's DSP and decoders run on the ARM CPU. On a Jetson the expensive GPU sits idle during decoding, which is why it is poor value as a dedicated decode host."
  - q: "Can the Jetson decode encrypted channels a Pi can't?"
    a: "No. No single-board computer changes the encryption wall — GopherTrunk cannot decode AES-protected traffic on any host, Jetson included. The GPU does not help here."
  - q: "Will GopherTrunk even install on a Jetson?"
    a: "Yes — it cross-compiles to ARM64 and runs on the JetPack Linux image with just a USB port for your SDR, no vendor toolchain. It simply won't use most of what you paid for."
---

**NVIDIA Jetson** is a family of [single-board computers](/reference/single-board-computer/) with a powerful onboard GPU, aimed at on-device AI and computer vision.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 156" role="img" aria-label="An edge inference pipeline on a Jetson. A camera feeds frames into the module, where an ARM CPU handles the operating system and a large array of GPU cores runs the neural-network math in parallel; the result — a detection or classification — comes straight back out on-device, with no cloud round trip." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" fill="none" stroke-width="1.3">
    <rect x="26" y="56" width="46" height="34" rx="4" fill-opacity="0.06" fill="currentColor"/>
    <path d="M72 73 H104"/>
    <rect x="104" y="30" width="220" height="90" rx="6" fill-opacity="0.05" fill="currentColor"/>
    <rect x="120" y="48" width="60" height="54" rx="4" fill-opacity="0.12" fill="currentColor"/>
    <g stroke-width="0.9">
      <rect x="196" y="48" width="112" height="54" rx="4" fill-opacity="0.05" fill="currentColor"/>
      <rect x="204" y="56" width="10" height="10" rx="1.5" fill-opacity="0.35" fill="currentColor"/>
      <rect x="220" y="56" width="10" height="10" rx="1.5" fill-opacity="0.35" fill="currentColor"/>
      <rect x="236" y="56" width="10" height="10" rx="1.5" fill-opacity="0.35" fill="currentColor"/>
      <rect x="252" y="56" width="10" height="10" rx="1.5" fill-opacity="0.35" fill="currentColor"/>
      <rect x="268" y="56" width="10" height="10" rx="1.5" fill-opacity="0.35" fill="currentColor"/>
      <rect x="284" y="56" width="10" height="10" rx="1.5" fill-opacity="0.35" fill="currentColor"/>
      <rect x="204" y="72" width="10" height="10" rx="1.5" fill-opacity="0.35" fill="currentColor"/>
      <rect x="220" y="72" width="10" height="10" rx="1.5" fill-opacity="0.35" fill="currentColor"/>
      <rect x="236" y="72" width="10" height="10" rx="1.5" fill-opacity="0.35" fill="currentColor"/>
      <rect x="252" y="72" width="10" height="10" rx="1.5" fill-opacity="0.35" fill="currentColor"/>
      <rect x="268" y="72" width="10" height="10" rx="1.5" fill-opacity="0.35" fill="currentColor"/>
      <rect x="284" y="72" width="10" height="10" rx="1.5" fill-opacity="0.35" fill="currentColor"/>
      <rect x="204" y="88" width="10" height="10" rx="1.5" fill-opacity="0.35" fill="currentColor"/>
      <rect x="220" y="88" width="10" height="10" rx="1.5" fill-opacity="0.35" fill="currentColor"/>
      <rect x="236" y="88" width="10" height="10" rx="1.5" fill-opacity="0.35" fill="currentColor"/>
      <rect x="252" y="88" width="10" height="10" rx="1.5" fill-opacity="0.35" fill="currentColor"/>
      <rect x="268" y="88" width="10" height="10" rx="1.5" fill-opacity="0.35" fill="currentColor"/>
      <rect x="284" y="88" width="10" height="10" rx="1.5" fill-opacity="0.35" fill="currentColor"/>
    </g>
    <path d="M324 73 H356"/>
    <rect x="356" y="56" width="80" height="34" rx="4" fill-opacity="0.06" fill="currentColor"/>
  </g>
  <g fill="currentColor" stroke="none" font-size="8" text-anchor="middle">
    <text x="49" y="77" font-size="7.5">camera</text>
    <text x="150" y="79" font-size="8" font-weight="600">ARM</text>
    <text x="150" y="90" font-size="6.5">OS</text>
    <text x="252" y="40" font-size="8" font-weight="600">GPU cores</text>
    <text x="396" y="72" font-size="7.5">detection</text>
    <text x="396" y="84" font-size="7">on-device</text>
    <text x="230" y="140" font-size="7.5" fill-opacity="0.9">CPU + many parallel GPU cores run inference locally — no cloud</text>
  </g>
</svg>
<figcaption>A Jetson pairs a conventional ARM CPU with a large array of GPU cores; frames from a camera are classified by the neural network running across those cores and the result comes straight back on-device, without a cloud round trip.</figcaption>
</figure>

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Overkill for GopherTrunk.** The Jetson Orin Nano (~$250) is a GPU board built for
[edge AI](/reference/edge-ai/) — but **GopherTrunk uses the CPU only, never CUDA or the
GPU**, so the pricey silicon sits idle during decoding. A ~$55–80
[Raspberry Pi](/reference/raspberry-pi/) follows the exact same channels for a fraction of
the cost. **Buy a Jetson only if you also need the GPU** for signal-classification, ML
analysis of decoded traffic, or vision on the same node. It runs GopherTrunk fine
(pure-Go ARM64 binary), but no board — Jetson included — decodes
[AES encryption](/police-scanner-encryption/). For a decode host, get a
[Pi](/best-single-board-computer-for-gophertrunk/) instead.
</div>

## Overview

Where a general-purpose board leans on its [CPU](/reference/central-processing-unit/), a Jetson pairs an ARM CPU with an NVIDIA GPU so that machine-learning inference can run locally. Neural-network math is overwhelmingly parallel — the same operation across thousands of values — which is precisely what a GPU's many cores are built for, so a model that would crawl on a CPU runs in real time on the Jetson's silicon. This makes it a fit for robotics, cameras, and other edge devices that cannot rely on the cloud.

Jetsons run Linux through NVIDIA's JetPack distribution, which bundles the CUDA and TensorRT libraries that let frameworks reach the GPU. The family spans a wide range, from the modest Jetson Nano to the far more capable Orin modules, so "a Jetson" can mean anything from a hobby board to a several-thousand-dollar module driving an autonomous machine.

## How it compares

| | [Raspberry Pi](/reference/raspberry-pi/) | NVIDIA Jetson | [Google Coral](/reference/google-coral/) |
|---|-------------|---------------|--------|
| Accelerator | None (CPU only) | CUDA GPU | Edge TPU |
| ML flexibility | Light CPU inference | Broad, general GPU | Quantised TF Lite only |
| Power draw | ~2–8 W | ~5–60 W | ~2 W |
| Price | ~$15–80 | ~$100–2000+ | ~$60–150 |
| Best role | General-purpose node | Heavy edge AI / vision | Cheap fixed inference |

## Where it fits

A Jetson is more expensive and more power-hungry than a [Raspberry Pi](/reference/raspberry-pi/), so it is the SBC you reach for specifically when you need GPU compute at the edge — [edge AI](/reference/edge-ai/), computer vision, robotics — rather than a general-purpose board. For lighter, always-on roles a Pi is usually the better fit; when real-time I/O matters more than compute, look at the [BeagleBone](/reference/beaglebone/). In a GopherTrunk context a Jetson is overkill for plain decoding, but its GPU could accelerate signal-classification or machine-learning analysis of decoded traffic on the same node.

## Running GopherTrunk on the Jetson Orin Nano

A Jetson runs GopherTrunk perfectly well — it simply does so on its ARM CPU, leaving the
expensive GPU idle, which is why it's overkill as a dedicated decode host. Concretely:

- **Architecture** — the Orin Nano is ARM64 (aarch64), so it takes the same static `linux/arm64` Go binary as a [Raspberry Pi](/reference/raspberry-pi/) from the [downloads page](/downloads.html). No CUDA build, no vendor toolchain, and GopherTrunk never links against the GPU stack.
- **CPU** — its 6-core Arm Cortex-A78AE (~1.5 GHz) is more than enough for real-time DSP on a multi-SDR pool — but so is a far cheaper Pi. The GPU that justifies the Jetson's price does no decoding work.
- **RAM** — the 8 GB shared between CPU and GPU leaves plenty for recording, the web console, and several systems at once; only a fraction is needed for the decode itself.
- **USB** — USB 3.2 ports handle one or more SDR dongles with bandwidth to spare for wideband Airspy capture; use a powered hub for [several dongles](/multi-dongle-sdr-setup/).
- **Storage** — microSD plus an M.2 [NVMe](/reference/nvme/) slot, so continuous IQ recording and a large call database can live on fast solid-state storage.
- **Power / thermals** — considerably more draw than a Pi (roughly 7–25 W depending on power mode) and it ships with an active heatsink-fan; fine for 24/7 but far from the most efficient always-on option.
- **OS / networking** — runs NVIDIA's JetPack (an Ubuntu-based 64-bit Linux); gigabit Ethernet is on board (Wi-Fi is an M.2 add-in) for reaching the [web console](/what-do-i-need-for-gophertrunk/) headless.

**Bottom line:** the Orin Nano handles the same workload as a Raspberry Pi — a couple of SDRs
with recording, or a small pool — with the GPU sitting unused; buy it only if that GPU will
earn its keep on other work.

## Where to buy

Be honest with yourself about why you want one. As a GopherTrunk decode host the Jetson
Orin Nano is overkill — it runs the decoders on its CPU and leaves the GPU idle, so a
[Raspberry Pi](/reference/raspberry-pi/) at a quarter of the price does the same job. The
Jetson earns its keep only if the GPU will do real work alongside decoding. If that's you,
the Orin Nano developer kit is the sensible entry point; otherwise see
[best single-board computer for GopherTrunk](/best-single-board-computer-for-gophertrunk/).

<a class="btn btn--buy" href="https://www.amazon.com/dp/B0BZJTQ5YP?tag=gophertrunk-20" rel="nofollow sponsored noopener">Check price on Amazon &rarr;</a>

*As an Amazon Associate, GopherTrunk earns from qualifying purchases — at no extra cost
to you. It never changes what we recommend.*

## Sources

[^wiki]: [Nvidia Jetson](https://en.wikipedia.org/wiki/Nvidia_Jetson) — Wikipedia, on the Jetson family and its edge-AI focus.
