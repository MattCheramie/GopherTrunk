---
slug: john-von-neumann
title: John von Neumann
entry_type: person
category: hw-people
description: John von Neumann (1903–1957) was a Hungarian-American mathematician who described the stored-program computer architecture that bears his name and still organizes nearly every general-purpose computer.
keywords: John von Neumann, von Neumann architecture, stored-program computer, EDVAC, computer science, mathematics
aka: [John von Neumann, von Neumann]
autolink: true
infobox:
  - { label: Lived, value: "1903–1957" }
  - { label: Field, value: Mathematics / computing }
  - { label: Known for, value: Von Neumann architecture }
see_also: [von-neumann-architecture, central-processing-unit, alan-turing, computer-hardware]
cite_urls:
  - https://en.wikipedia.org/wiki/John_von_Neumann
---

**John von Neumann** (1903–1957) was a Hungarian-American mathematician and polymath who
described the stored-program computer design — the
[von Neumann architecture](/reference/von-neumann-architecture/) — that still organises
nearly every general-purpose machine.[^wiki]

## Life and work

Von Neumann made foundational contributions across mathematics, physics, game theory, and
the design of the atomic bomb before turning to computing. His 1945 "First Draft of a Report
on the EDVAC" set out a machine in which program instructions and data share the same memory,
fetched and executed by a single processing unit. That single insight — that a computer
should store its own program — separated software from wiring and made the
programmable computer practical.[^wiki]

## Why they matter

In the [von Neumann architecture](/reference/von-neumann-architecture/), a
[CPU](/reference/central-processing-unit/) repeatedly fetches an instruction from memory,
decodes it, and acts — the fetch-decode-execute cycle at the heart of essentially all of
today's [computer hardware](/reference/computer-hardware/). It built on the universal-machine
idea of [Alan Turing](/reference/alan-turing/) and turned it into a buildable blueprint. When
GopherTrunk runs a decode pipeline, it runs as instructions in shared memory on exactly this
kind of machine.[^wiki]

## Legacy

The shared instruction/data memory is sometimes called the "von Neumann bottleneck," and
modern caches and Harvard-style splits are responses to it — but the basic model he named is
still the default mental picture of how a computer works.

## Sources

[^wiki]: [John von Neumann](https://en.wikipedia.org/wiki/John_von_Neumann) — Wikipedia, for biography and the EDVAC report.
