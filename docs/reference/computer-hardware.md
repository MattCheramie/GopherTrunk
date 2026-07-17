---
slug: computer-hardware
title: Computer hardware
entry_type: concept
category: hw-foundations
description: Computer hardware is the physical, touchable part of a computing device — the chips, boards, memory, and drives — as distinct from the software that runs on it.
keywords: computer hardware, physical components, input process output, CPU memory storage I/O, hardware vs software, building blocks
aka: [computer hardware, hardware]
infobox:
  - { label: Type, value: Physical computing parts }
  - { label: Opposite of, value: Software }
  - { label: Model, value: Input → process → output }
  - { label: Building blocks, value: CPU, memory, storage, I/O }
  - { label: Range, value: Sensor chip to data-center server }
see_also: [central-processing-unit, random-access-memory, data-storage, input-output, hardware-spectrum]
related_lessons:
  - { title: "What is hardware?", url: /learn/intro-hardware/what-is-hardware/ }
  - { title: "The building blocks", url: /learn/intro-hardware/building-blocks/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Computer_hardware
---

**Computer hardware** is the set of physical parts you can touch — the chips, circuit boards, memory modules, drives, and case — as opposed to the [software](/reference/operating-system/) that tells them what to do.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 160" role="img" aria-label="A block diagram of the input-process-output model. Input devices feed data to the central processing unit, which works with random-access memory and long-term storage to process it, and the result goes out through output devices." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" fill="none" stroke-width="1.2">
    <rect x="14" y="60" width="80" height="40" rx="4"/>
    <rect x="180" y="60" width="100" height="40" rx="4" fill="currentColor" fill-opacity="0.14"/>
    <rect x="366" y="60" width="80" height="40" rx="4"/>
    <rect x="180" y="10" width="100" height="28" rx="4"/>
    <rect x="180" y="122" width="100" height="28" rx="4"/>
  </g>
  <g stroke="currentColor" stroke-width="1.2" fill="none">
    <path d="M94 80 h74 m-6 -4 l6 4 l-6 4"/>
    <path d="M280 80 h74 m-6 -4 l6 4 l-6 4"/>
    <path d="M230 38 v14 m-4 -6 l4 6 l4 -6"/>
    <path d="M230 100 v14 m-4 -6 l4 6 l4 -6"/>
  </g>
  <g fill="currentColor" stroke="none" text-anchor="middle">
    <text x="54" y="77" font-size="9">Input</text>
    <text x="54" y="90" font-size="7.5" fill-opacity="0.85">keyboard, sensor</text>
    <text x="230" y="77" font-size="9" font-weight="600">Process (CPU)</text>
    <text x="230" y="90" font-size="7.5" fill-opacity="0.85">run instructions</text>
    <text x="406" y="77" font-size="9">Output</text>
    <text x="406" y="90" font-size="7.5" fill-opacity="0.85">screen, radio</text>
    <text x="230" y="28" font-size="8.5">Memory (RAM)</text>
    <text x="230" y="140" font-size="8.5">Storage (disk)</text>
  </g>
</svg>
<figcaption>Every computer, from a sensor chip to a data-center server, follows the same input-process-output model: input arrives, the CPU works with memory and storage to process it, and output goes back out — the four building blocks that hardware provides.</figcaption>
</figure>

## Overview

A computer is anything that takes **input**, follows instructions to **process** it, and produces **output**. That definition spans an enormous range: a warehouse [server](/reference/server/), the [laptop](/reference/laptop/) on your desk, and the tiny chip inside a sensor are all computers by the same rule.

Nearly every one of them is built from the same four building blocks: a [central processing unit](/reference/central-processing-unit/) to run instructions, [random-access memory](/reference/random-access-memory/) for active working data, [storage](/reference/data-storage/) for data that must survive a power cycle, and [input/output](/reference/input-output/) to move data in and out. Everything else — buses, power, cooling, expansion cards — exists to connect and support those four.

## The four building blocks

Each block has a distinct job, and the differences in speed and persistence between them shape how software is written:

| Block | Job | Speed | Keeps data without power? |
|-------|-----|-------|---------------------------|
| CPU | Execute instructions | Fastest | No |
| Memory (RAM) | Hold active working data | Very fast | No |
| Storage | Keep data long-term | Slower | Yes |
| Input/Output | Move data in and out | Varies | — |

This is why a program loads from slow, permanent storage into fast, volatile memory to run, and why losing power mid-task loses anything not yet written back to storage.

## Why it matters

Knowing which physical resources a device has — how fast its CPU is, how much memory and storage it carries, what it can connect to — tells you what software it can realistically run. Software-defined-radio software like GopherTrunk runs across this whole range, from a full desktop down to a [single-board computer](/reference/single-board-computer/) by the antenna; the hardware sets the ceiling on sample rates, channel counts, and how much decoding a node can sustain.

## Sources

[^wiki]: [Computer hardware](https://en.wikipedia.org/wiki/Computer_hardware) — Wikipedia, on the physical components of a computer and the hardware/software distinction.
