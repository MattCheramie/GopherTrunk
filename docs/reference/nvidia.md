---
slug: nvidia
title: NVIDIA
entry_type: organization
category: hw-organizations
description: NVIDIA is an American company that designs GPUs and the CUDA platform, and is the leading supplier of accelerators for graphics, AI, and high-performance computing.
keywords: NVIDIA, GPU, CUDA, Jetson, GeForce, AI accelerator, Jensen Huang, Tegra
aka: [NVIDIA Corporation]
autolink: true
infobox:
  - { label: Type, value: Public semiconductor company }
  - { label: Founded, value: "1993" }
  - { label: HQ, value: Santa Clara, California, USA }
  - { label: Makes, value: GPUs, AI accelerators, SoCs }
see_also: [graphics-processing-unit, cuda, nvidia-jetson, jensen-huang, tsmc, ai-accelerator]
cite_urls:
  - https://www.nvidia.com/
  - https://en.wikipedia.org/wiki/Nvidia
---

**NVIDIA** is an American company founded in 1993 that designs
[GPUs](/reference/graphics-processing-unit/) and the
[CUDA](/reference/cuda/) software platform for general-purpose GPU computing.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 130" role="img" aria-label="A timeline of NVIDIA. Founded in 1993, it popularizes the term GPU with the 1999 GeForce, turns the GPU into a general parallel processor by releasing CUDA in 2007, and in the modern era becomes the leading supplier of AI and high-performance computing accelerators." xmlns="http://www.w3.org/2000/svg">
  <line x1="30" y1="66" x2="440" y2="66" stroke="currentColor" stroke-width="1.4"/>
  <g stroke="currentColor" fill="currentColor" stroke-width="1.2">
    <circle cx="58" cy="66" r="5" fill-opacity="0.15"/>
    <circle cx="178" cy="66" r="5" fill-opacity="0.15"/>
    <circle cx="300" cy="66" r="6" fill="currentColor"/>
    <circle cx="412" cy="66" r="5" fill-opacity="0.4"/>
  </g>
  <g fill="currentColor" stroke="none" text-anchor="middle">
    <text x="58" y="50" font-size="9" font-weight="600">1993</text>
    <text x="58" y="86" font-size="8">founded</text>
    <text x="178" y="50" font-size="9" font-weight="600">1999</text>
    <text x="178" y="86" font-size="8">GeForce</text>
    <text x="178" y="97" font-size="8">coins "GPU"</text>
    <text x="300" y="50" font-size="9" font-weight="600">2007</text>
    <text x="300" y="86" font-size="8">CUDA</text>
    <text x="300" y="97" font-size="8">GPU compute</text>
    <text x="412" y="50" font-size="9" font-weight="600">now</text>
    <text x="412" y="86" font-size="8">AI / HPC</text>
    <text x="412" y="97" font-size="8">accelerators</text>
  </g>
</svg>
<figcaption>NVIDIA moved from graphics to general compute: the 1999 GeForce popularized the term GPU, CUDA in 2007 turned that GPU into a programmable parallel processor, and the same hardware now leads the AI and HPC accelerator market.</figcaption>
</figure>

## Overview

NVIDIA was co-founded and is led by [Jensen Huang](/reference/jensen-huang/). It popularised
the term "GPU" with its GeForce line and later turned the GPU into a general parallel
processor by releasing CUDA in 2007. That move made NVIDIA hardware the standard platform
for deep learning, and the company is now a leading supplier of AI and high-performance
computing accelerators.[^home]

Beyond add-in cards, NVIDIA designs Tegra systems-on-chip and the
[Jetson](/reference/nvidia-jetson/) line of embedded AI modules, bringing GPU-accelerated
computing to robotics, cameras, and edge devices.

## Product lines

NVIDIA's silicon now spans from gaming cards to data-center accelerators and edge modules:

| Product line | What it is |
|--------------|------------|
| GeForce | Consumer GPUs for gaming and graphics |
| Data-center GPUs | Accelerators for AI training and HPC |
| Jetson | Embedded GPU modules for edge AI |
| CUDA | The software platform that unlocks GPU compute |

## Where it fits

GPUs excel at the massively parallel arithmetic that DSP and machine learning both demand.
An NVIDIA card can accelerate heavy SDR signal processing or run on-device classification,
and a Jetson [AI accelerator](/reference/ai-accelerator/) can do GPU-accelerated work at the
edge — for example, near an antenna where sending raw IQ back to a server would be
impractical. Like most leading-edge designs, NVIDIA's chips are fabricated by a foundry such
as [TSMC](/reference/tsmc/).

## Sources

[^home]: [NVIDIA](https://www.nvidia.com/) — the company's official site, for products and CUDA.
[^wiki]: [Nvidia](https://en.wikipedia.org/wiki/Nvidia) — Wikipedia, for the company's history and significance.
