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

## Overview

Each gate takes inputs that are either 0 or 1 and produces an output defined by a *truth table*: an AND gate outputs 1 only when all inputs are 1, an OR when any is 1, a NOT inverts its input. Physically, gates are built from a handful of [transistors](/reference/transistor/) acting as switches. From these primitives — especially the universal NAND and NOR — you can construct adders, memory cells, and ultimately every part of a processor.

## Where it fits

Logic gates are the bridge from raw [bits](/reference/bits-and-bytes/) to computation: arrange enough of them and you get the arithmetic and control circuits inside a [CPU](/reference/central-processing-unit/), all fabricated together on an [integrated circuit](/reference/integrated-circuit/). The whole [von Neumann](/reference/von-neumann-architecture/) machine — fetch, decode, execute — reduces, at the bottom, to networks of gates switching in step with the clock.

## Sources

[^wiki]: [Logic gate](https://en.wikipedia.org/wiki/Logic_gate) — Wikipedia, on logic gates, Boolean functions, and truth tables.
