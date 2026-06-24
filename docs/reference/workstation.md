---
slug: workstation
title: Workstation
entry_type: hardware
category: hw-personal-computers
description: A workstation is a high-end personal computer built for demanding professional work — CAD, video, simulation, software builds — with powerful CPUs, lots of memory, and often error-correcting RAM.
keywords: workstation, professional PC, ECC memory, Xeon, Threadripper, CAD, render workstation, ISV certified
aka: [Pro workstation]
infobox:
  - { label: Type, value: High-end personal computer }
  - { label: Built for, value: Professional / compute-heavy work }
  - { label: CPU, value: Many-core (Xeon, Threadripper) }
  - { label: Memory, value: Large, often ECC }
  - { label: GPU, value: Pro / compute card common }
see_also: [personal-computer, desktop-computer, gaming-pc, central-processing-unit, graphics-processing-unit, server]
related_lessons:
  - { title: "Choosing a dev machine", url: /learn/intro-hardware/choosing-a-dev-machine/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Workstation
---

A **workstation** is a high-end [personal computer](/reference/personal-computer/) built for demanding professional work — engineering, content creation, scientific computing, and large software builds — rather than ordinary office or home use.[^wiki]

## Overview

A workstation looks like a [desktop computer](/reference/desktop-computer/) but is specified for sustained heavy load: a many-core [CPU](/reference/central-processing-unit/) (Intel Xeon or AMD Threadripper class), large amounts of [RAM](/reference/random-access-memory/) — frequently error-correcting (ECC) memory to catch bit errors — and a professional [GPU](/reference/graphics-processing-unit/) tuned for compute or certified rendering. Vendors often "certify" workstations against specific professional applications (ISV certification) so the hardware and drivers are validated for that software.

## Where it fits

Pick a workstation when the work is reliably compute-bound: CAD, finite-element simulation, 3D rendering, video editing, or compiling large codebases all day. For gaming the lighter, GPU-centric [gaming PC](/reference/gaming-pc/) is usually a better value, and a true [server](/reference/server/) wins when the job is serving many clients rather than driving one user's desktop. In GopherTrunk terms, a workstation is overkill for a single capture node but handy as the bench machine where you replay captures and run wideband DSP across many channels at once.

## Sources

[^wiki]: [Workstation](https://en.wikipedia.org/wiki/Workstation) — Wikipedia, on workstations as high-end computers for technical and professional work.
