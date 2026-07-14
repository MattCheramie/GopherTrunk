---
slug: logic-gate
title: Logic gate
entry_type: concept
category: hw-foundations
description: A logic gate is a basic building block of digital circuits that computes a Boolean function such as AND, OR, or NOT; combinations of gates implement all digital computation.
keywords: logic gate, Boolean logic, AND, OR, NOT, NAND, XOR, digital circuit, truth table
infobox:
  - { label: Type, value: Digital circuit element }
  - { label: Computes, value: Boolean functions }
  - { label: Basic gates, value: "AND, OR, NOT" }
  - { label: Built from, value: Transistors }
see_also: [transistor, integrated-circuit, semiconductor, central-processing-unit, bits-and-bytes, von-neumann-architecture]
cite_urls:
  - https://en.wikipedia.org/wiki/Logic_gate
---

A **logic gate** is a basic building block of digital circuits that computes a simple Boolean function — such as AND, OR, or NOT — on one or more binary inputs.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 208" role="img" aria-label="Three logic gates with their truth tables. AND outputs 1 only when both inputs are 1. OR outputs 1 when either input is 1. NOT inverts its single input. Each is drawn as its standard symbol above a small truth table." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" fill="currentColor">
    <g transform="translate(64,44)">
      <path d="M0 0 H20 A20 20 0 0 1 20 40 H0 Z" fill-opacity="0.12" stroke-width="1.4"/>
      <line x1="-20" y1="10" x2="0" y2="10" stroke-width="1.2" fill="none"/>
      <line x1="-20" y1="30" x2="0" y2="30" stroke-width="1.2" fill="none"/>
      <line x1="40" y1="20" x2="60" y2="20" stroke-width="1.2" fill="none"/>
      <text x="20" y="24" text-anchor="middle" font-size="9" stroke="none" font-weight="600">AND</text>
    </g>
    <g transform="translate(210,44)">
      <path d="M0 0 C10 12 10 28 0 40 C22 40 40 28 48 20 C40 12 22 0 0 0 Z" fill-opacity="0.12" stroke-width="1.4"/>
      <line x1="-20" y1="10" x2="6" y2="10" stroke-width="1.2" fill="none"/>
      <line x1="-20" y1="30" x2="6" y2="30" stroke-width="1.2" fill="none"/>
      <line x1="48" y1="20" x2="66" y2="20" stroke-width="1.2" fill="none"/>
      <text x="20" y="24" text-anchor="middle" font-size="9" stroke="none" font-weight="600">OR</text>
    </g>
    <g transform="translate(356,44)">
      <path d="M0 0 L0 40 L36 20 Z" fill-opacity="0.12" stroke-width="1.4"/>
      <circle cx="42" cy="20" r="5" fill="none" stroke-width="1.4"/>
      <line x1="-20" y1="20" x2="0" y2="20" stroke-width="1.2" fill="none"/>
      <line x1="47" y1="20" x2="66" y2="20" stroke-width="1.2" fill="none"/>
      <text x="14" y="24" text-anchor="middle" font-size="8.5" stroke="none" font-weight="600">NOT</text>
    </g>
  </g>
  <g font-family="ui-monospace, monospace" font-size="9" fill="currentColor" stroke="none">
    <text x="40" y="128">A B → Y</text>
    <text x="40" y="142">0 0 → 0</text>
    <text x="40" y="156">0 1 → 0</text>
    <text x="40" y="170">1 0 → 0</text>
    <text x="40" y="184">1 1 → 1</text>
    <text x="190" y="128">A B → Y</text>
    <text x="190" y="142">0 0 → 0</text>
    <text x="190" y="156">0 1 → 1</text>
    <text x="190" y="170">1 0 → 1</text>
    <text x="190" y="184">1 1 → 1</text>
    <text x="344" y="128">A → Y</text>
    <text x="344" y="142">0 → 1</text>
    <text x="344" y="156">1 → 0</text>
  </g>
  <text x="230" y="202" text-anchor="middle" font-size="8.5" fill="currentColor" fill-opacity="0.9">built from transistor switches · NAND and NOR alone can build any of them</text>
</svg>
<figcaption>Each gate maps binary inputs to an output by a fixed truth table: AND needs all inputs high, OR needs any, NOT inverts. Physically they're a handful of transistor switches — and from these primitives (especially the universal NAND and NOR) you can build adders, memory, and ultimately a whole processor.</figcaption>
</figure>

## Overview

Each gate takes inputs that are either 0 or 1 and produces an output defined by a *truth table*: an AND gate outputs 1 only when all inputs are 1, an OR when any is 1, a NOT inverts its input. Physically, gates are built from a handful of [transistors](/reference/transistor/) acting as switches. From these primitives — especially the universal NAND and NOR — you can construct adders, memory cells, and ultimately every part of a processor.

## Where it fits

Logic gates are the bridge from raw [bits](/reference/bits-and-bytes/) to computation: arrange enough of them and you get the arithmetic and control circuits inside a [CPU](/reference/central-processing-unit/), all fabricated together on an [integrated circuit](/reference/integrated-circuit/). The whole [von Neumann](/reference/von-neumann-architecture/) machine — fetch, decode, execute — reduces, at the bottom, to networks of gates switching in step with the clock.

## Sources

[^wiki]: [Logic gate](https://en.wikipedia.org/wiki/Logic_gate) — Wikipedia, on logic gates, Boolean functions, and truth tables.
