---
slug: alan-turing
title: Alan Turing
entry_type: person
category: hw-people
description: Alan Turing (1912–1954) was a British mathematician and logician whose 1936 model of computation defined what a general-purpose computer is and laid the theoretical foundation for the modern stored-program machine.
keywords: Alan Turing, Turing machine, computability, Bletchley Park, Enigma, Turing test, theoretical computer science, stored-program computer, universal machine
aka: [Alan Turing, Turing]
autolink: true
infobox:
  - { label: Lived, value: "1912–1954" }
  - { label: Field, value: Mathematics / logic / computing }
  - { label: Known for, value: Turing machine, computability }
see_also: [john-von-neumann, von-neumann-architecture, central-processing-unit, computer-hardware]
cite_urls:
  - https://en.wikipedia.org/wiki/Alan_Turing
---

**Alan Turing** (1912–1954) was a British mathematician and logician whose abstract
**Turing machine** defined what it means for a problem to be computable and framed the idea
of a single machine that can run any program.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 130" role="img" aria-label="A life-and-work timeline of Alan Turing: born in 1912, he published On Computable Numbers describing the universal machine in 1936, led Enigma code-breaking at Bletchley Park from 1939 to 1945, proposed the Turing test in 1950, and died in 1954." xmlns="http://www.w3.org/2000/svg">
  <line x1="30" y1="66" x2="440" y2="66" stroke="currentColor" stroke-width="1.4"/>
  <g stroke="currentColor" fill="currentColor" stroke-width="1.2">
    <circle cx="50" cy="66" r="4" fill-opacity="0.15"/>
    <circle cx="160" cy="66" r="6" fill="currentColor"/>
    <circle cx="260" cy="66" r="5" fill-opacity="0.15"/>
    <circle cx="360" cy="66" r="5" fill-opacity="0.15"/>
    <circle cx="430" cy="66" r="4" fill-opacity="0.15"/>
  </g>
  <g fill="currentColor" stroke="none" text-anchor="middle">
    <text x="50" y="50" font-size="9" font-weight="600">1912</text>
    <text x="50" y="86" font-size="8">born</text>
    <text x="160" y="50" font-size="9" font-weight="600">1936</text>
    <text x="160" y="86" font-size="8">universal machine</text>
    <text x="260" y="50" font-size="9" font-weight="600">1939&#8211;45</text>
    <text x="260" y="86" font-size="8">Bletchley Park</text>
    <text x="360" y="50" font-size="9" font-weight="600">1950</text>
    <text x="360" y="86" font-size="8">Turing test</text>
    <text x="430" y="50" font-size="9" font-weight="600">1954</text>
    <text x="430" y="86" font-size="8">died</text>
    <text x="235" y="112" font-size="8" fill-opacity="0.9">one machine, any program &#8212; the idea underneath every general-purpose computer</text>
  </g>
</svg>
<figcaption>Turing's arc from the 1936 universal machine through wartime code-breaking to the Turing test: a single thread that turned "computation" from a human activity into something a machine could, in principle, do for any task.</figcaption>
</figure>

## Life and work

In his 1936 paper "On Computable Numbers," Turing described a simple abstract device — a tape,
a read/write head, and a table of rules — and proved that one universal version of it could
simulate any other. That universal machine is the theoretical ancestor of every
programmable computer.[^wiki] During the Second World War he led code-breaking work at
Bletchley Park, helping build electromechanical machines that broke the German Enigma
cipher. After the war he contributed to early stored-program computer designs and to the
foundations of artificial intelligence, proposing the test that bears his name.[^wiki]

| Year | Milestone |
|------|-----------|
| 1936 | "On Computable Numbers" defines the universal machine |
| 1939–45 | Leads Enigma code-breaking at Bletchley Park |
| 1948–49 | Contributes to early stored-program computer design |
| 1950 | Proposes the Turing test for machine intelligence |

## Why they matter

Turing's universal machine is the reason a [CPU](/reference/central-processing-unit/) can run
word processors, web servers, and an SDR decoder without being rewired for each — software,
not hardware, defines the task. His computability results sit alongside the practical
[von Neumann architecture](/reference/von-neumann-architecture/) that
[John von Neumann](/reference/john-von-neumann/) formalised a decade later, and together they
underpin all general-purpose [computer hardware](/reference/computer-hardware/).[^wiki]

## Legacy

The Turing Award, computing's highest honour, is named after him, and his model remains the
standard reference for what computers can and cannot, in principle, do. Every time GopherTrunk
loads a new protocol decoder as software on the same hardware, it is exercising exactly the
universality Turing proved possible.

## Sources

[^wiki]: [Alan Turing](https://en.wikipedia.org/wiki/Alan_Turing) — Wikipedia, for biography, the Turing machine, and Bletchley Park.
