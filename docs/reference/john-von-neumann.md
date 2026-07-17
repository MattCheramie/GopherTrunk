---
slug: john-von-neumann
title: John von Neumann
entry_type: person
category: hw-people
description: John von Neumann (1903–1957) was a Hungarian-American mathematician who described the stored-program computer architecture that bears his name and still organizes nearly every general-purpose computer.
keywords: John von Neumann, von Neumann architecture, stored-program computer, EDVAC, fetch-decode-execute, computer science, mathematics
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

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 130" role="img" aria-label="A life-and-work timeline of John von Neumann: born in 1903, he formalized game theory in 1928, wrote the First Draft of a Report on the EDVAC describing the stored-program computer in 1945, built the IAS machine around 1951, and died in 1957." xmlns="http://www.w3.org/2000/svg">
  <line x1="30" y1="66" x2="440" y2="66" stroke="currentColor" stroke-width="1.4"/>
  <g stroke="currentColor" fill="currentColor" stroke-width="1.2">
    <circle cx="50" cy="66" r="4" fill-opacity="0.15"/>
    <circle cx="150" cy="66" r="5" fill-opacity="0.15"/>
    <circle cx="270" cy="66" r="6" fill="currentColor"/>
    <circle cx="370" cy="66" r="5" fill-opacity="0.15"/>
    <circle cx="430" cy="66" r="4" fill-opacity="0.15"/>
  </g>
  <g fill="currentColor" stroke="none" text-anchor="middle">
    <text x="50" y="50" font-size="9" font-weight="600">1903</text>
    <text x="50" y="86" font-size="8">born</text>
    <text x="150" y="50" font-size="9" font-weight="600">1928</text>
    <text x="150" y="86" font-size="8">game theory</text>
    <text x="270" y="50" font-size="9" font-weight="600">1945</text>
    <text x="270" y="86" font-size="8">EDVAC report</text>
    <text x="370" y="50" font-size="9" font-weight="600">1951</text>
    <text x="370" y="86" font-size="8">IAS machine</text>
    <text x="430" y="50" font-size="9" font-weight="600">1957</text>
    <text x="430" y="86" font-size="8">died</text>
    <text x="235" y="112" font-size="8" fill-opacity="0.9">program and data share one memory &#8212; software becomes soft</text>
  </g>
</svg>
<figcaption>Von Neumann's arc from mathematics to computing peaks at the 1945 EDVAC report, whose single idea — storing the program in the same memory as the data — still names the architecture almost every computer follows.</figcaption>
</figure>

## Life and work

Von Neumann made foundational contributions across mathematics, physics, game theory, and
the design of the atomic bomb before turning to computing. His 1945 "First Draft of a Report
on the EDVAC" set out a machine in which program instructions and data share the same memory,
fetched and executed by a single processing unit. That single insight — that a computer
should store its own program — separated software from wiring and made the
programmable computer practical.[^wiki]

| Year | Milestone |
|------|-----------|
| 1928 | Formalises minimax game theory |
| 1945 | "First Draft of a Report on the EDVAC" |
| 1946 | Joins the IAS computer project |
| 1951 | IAS machine, a widely copied stored-program design |

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
