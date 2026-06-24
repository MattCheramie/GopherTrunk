---
slug: hardware-acceleration
title: Hardware acceleration
entry_type: concept
category: hw-accelerators
description: Hardware acceleration is offloading a task from the general-purpose CPU to specialized hardware built to do it faster or more efficiently, such as a GPU, FPGA, or fixed-function ASIC.
keywords: hardware acceleration, offload, GPU, FPGA, ASIC, fixed-function, accelerator, throughput, efficiency
infobox:
  - { label: Type, value: Computing technique }
  - { label: Idea, value: Offload work from the CPU }
  - { label: Done by, value: "GPU, FPGA, ASIC, DSP, NPU" }
  - { label: Wins, value: Speed and power efficiency }
  - { label: Cost, value: Less flexibility }
see_also: [graphics-processing-unit, field-programmable-gate-array, application-specific-integrated-circuit, vector-processor, ai-accelerator, soc-vs-discrete]
cite_urls:
  - https://en.wikipedia.org/wiki/Hardware_acceleration
---

**Hardware acceleration** is the practice of offloading a task from the general-purpose [CPU](/reference/central-processing-unit/) onto specialized hardware built to do it faster, more efficiently, or both.[^wiki]

## Overview

A CPU is a generalist: it can run any code, but it pays for that flexibility in speed and power. When a task is performed constantly and has a regular structure — graphics, video encoding, neural-network math, signal processing — it is often worth building hardware that does *only* that. Accelerators span a spectrum: the programmable [GPU](/reference/graphics-processing-unit/) and [vector](/reference/vector-processor/) units, the reconfigurable [FPGA](/reference/field-programmable-gate-array/), and the fixed-function [ASIC](/reference/application-specific-integrated-circuit/), with flexibility falling and efficiency rising along the way.

## Where it fits

The engineering question is always *what to offload*: moving work to an accelerator adds complexity and data-transfer overhead, so it pays only when the speedup is large or the CPU genuinely cannot keep up. This is a live trade-off in software-defined radio. GopherTrunk does its DSP and protocol decoding in software on the CPU, deliberately keeping the radio a simple front end; an [FPGA](/reference/field-programmable-gate-array/) doing the channelization in hardware would accelerate wideband, many-channel capture, at the cost of flexibility and a much harder development path.

## Sources

[^wiki]: [Hardware acceleration](https://en.wikipedia.org/wiki/Hardware_acceleration) — Wikipedia, on offloading tasks from the CPU to specialized hardware.
