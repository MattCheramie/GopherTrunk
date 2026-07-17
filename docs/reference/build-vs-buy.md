---
slug: build-vs-buy
title: Build vs buy a PC
entry_type: concept
category: hw-personal-computers
description: Build vs buy is the decision between assembling a desktop computer from individual components and purchasing a ready-made pre-built system, trading cost, control, and effort against convenience and warranty.
keywords: build vs buy, custom PC, pre-built PC, DIY computer, PC building, component selection, warranty, upgradeability
infobox:
  - { label: Type, value: Decision / trade-off }
  - { label: Build, value: Cheaper, custom, more effort }
  - { label: Buy, value: Convenient, warrantied, turnkey }
  - { label: Shared need, value: Parts must be compatible }
  - { label: Applies to, value: Desktop PCs }
see_also: [desktop-computer, gaming-pc, motherboard, computer-case, personal-computer, workstation]
related_lessons:
  - { title: "Choosing a dev machine", url: /learn/intro-hardware/choosing-a-dev-machine/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Custom-built_computer
---

**Build vs buy** is the decision, when getting a [desktop computer](/reference/desktop-computer/), between assembling it yourself from separate components and purchasing a ready-made pre-built system.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 200" role="img" aria-label="A build-versus-buy decision split. One branch gathers separate parts — CPU, motherboard, RAM, GPU, PSU, and case — and assembles them into a PC for lower cost and full control. The other branch takes a single sealed pre-built box for convenience and one warranty." xmlns="http://www.w3.org/2000/svg">
  <g fill="currentColor" stroke="none" text-anchor="middle">
    <text x="230" y="16" font-size="9" font-weight="600">getting a desktop</text>
  </g>
  <g stroke="currentColor" fill="none" stroke-width="1.2">
    <path d="M230 22 L120 46 M230 22 L340 46"/>
  </g>
  <g fill="currentColor" stroke="none" text-anchor="middle" font-size="9" font-weight="600">
    <text x="120" y="42">BUILD</text>
    <text x="340" y="42">BUY</text>
  </g>
  <g stroke="currentColor" fill="none" stroke-width="1.1">
    <rect x="30" y="56" width="42" height="20" rx="2"/>
    <rect x="80" y="56" width="42" height="20" rx="2"/>
    <rect x="130" y="56" width="42" height="20" rx="2"/>
    <rect x="30" y="84" width="42" height="20" rx="2"/>
    <rect x="80" y="84" width="42" height="20" rx="2"/>
    <rect x="130" y="84" width="42" height="20" rx="2"/>
    <path d="M100 108 L100 128"/>
    <rect x="78" y="130" width="44" height="52" rx="3" fill="currentColor" fill-opacity="0.1"/>
  </g>
  <g fill="currentColor" stroke="none" text-anchor="middle" font-size="7">
    <text x="51" y="69">CPU</text>
    <text x="101" y="69">board</text>
    <text x="151" y="69">RAM</text>
    <text x="51" y="97">GPU</text>
    <text x="101" y="97">PSU</text>
    <text x="151" y="97">case</text>
    <text x="100" y="160" font-size="8">your</text>
    <text x="100" y="172" font-size="8">PC</text>
  </g>
  <g stroke="currentColor" fill="none" stroke-width="1.1">
    <rect x="318" y="70" width="44" height="34" rx="3" fill="currentColor" fill-opacity="0.16"/>
    <path d="M340 108 L340 128"/>
    <rect x="318" y="130" width="44" height="52" rx="3" fill="currentColor" fill-opacity="0.1"/>
  </g>
  <g fill="currentColor" stroke="none" text-anchor="middle" font-size="7">
    <text x="340" y="90">sealed</text>
    <text x="340" y="99">box</text>
    <text x="340" y="160" font-size="8">ready</text>
    <text x="340" y="172" font-size="8">PC</text>
  </g>
  <g fill="currentColor" stroke="none" text-anchor="middle" font-size="7.5" fill-opacity="0.85">
    <text x="100" y="196">cheaper · custom · your labor</text>
    <text x="340" y="196">turnkey · one warranty · premium</text>
  </g>
</svg>
<figcaption>Building means sourcing each compatible part and assembling it yourself for lower cost and full control; buying means one tested box under a single warranty. Either way the parts must fit — board to socket, board to case, PSU to load.</figcaption>
</figure>

## Overview

Building means choosing each part — [CPU](/reference/central-processing-unit/), [motherboard](/reference/motherboard/), [RAM](/reference/random-access-memory/), [GPU](/reference/graphics-processing-unit/), [storage](/reference/solid-state-drive/), [power supply](/reference/power-supply-unit/), and [case](/reference/computer-case/) — and fitting them together yourself. Buying means a vendor delivers a complete, tested machine under a single warranty.

The parts must still be compatible either way: the [motherboard form factor](/reference/motherboard-form-factor/) has to match the case, the CPU has to match the board's socket, and the power supply has to be sized for the load. A pre-built simply pushes those checks onto the vendor.

## Trade-offs

The two paths weigh cost and control against convenience and support:

| Factor | Build | Buy |
|--------|-------|-----|
| Cost per performance | Usually lower | Higher (assembly premium) |
| Part choice | Exactly what you pick | Vendor's selection |
| Setup effort | Hours + learning curve | Unbox and go |
| Warranty | Per component | One, whole system |
| Upgrades later | Easy | Often limited |
| Troubleshooting | You | The vendor |

Building shines for [gaming PCs](/reference/gaming-pc/) and enthusiast [workstations](/reference/workstation/); buying suits anyone who just wants a working machine.

## Where it fits

The choice is really about how much of the risk and effort you want to own. Build when the savings, the exact spec, or the upgrade path matter to you and you enjoy the assembly; buy when time and a single point of support matter more. For a GopherTrunk bench either works — the radio front end is a USB dongle regardless of how the host PC was put together, so a pre-built and a hand-built machine decode identically.

## Sources

[^wiki]: [Custom-built computer](https://en.wikipedia.org/wiki/Custom-built_computer) — Wikipedia, on building versus buying a PC.
