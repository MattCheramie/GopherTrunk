---
slug: cloud-computing
title: Cloud computing
entry_type: concept
category: hw-servers
description: Cloud computing is computing power and storage provided over the internet on demand, scaling up and down with near-zero upfront cost and ongoing fees.
keywords: cloud computing, on demand, data center, IaaS, scalable, pay as you go
aka: [Cloud computing, The cloud]
infobox:
  - { label: Type, value: On-demand computing }
  - { label: Cost model, value: Pay as you go }
  - { label: Scales, value: Up and down }
  - { label: Built on, value: Virtualization }
see_also: [virtualization, virtual-private-server, server, dedicated-server, home-server]
related_lessons:
  - { title: "Combining tiers", url: /learn/intro-hardware/combining-tiers/ }
  - { title: "Virtual private server (VPS)", url: /learn/intro-hardware/vps/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Cloud_computing
---

**Cloud computing** is computing power and storage provided over the internet on demand, instead of from machines you own and run.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 464 268" role="img" aria-label="A grid of who manages each layer across four tiers — on-premises, IaaS, PaaS, and SaaS — with an arrow across the top showing that your responsibility shrinks as you move right. Rows are application, data, runtime, operating system, virtualization, and hardware. Filled cells are managed by you and light cells by the provider; moving right, the filled region shrinks from the bottom up until nothing is left." xmlns="http://www.w3.org/2000/svg">
  <text x="270" y="14" text-anchor="middle" font-size="8.5" fill="currentColor" fill-opacity="0.85">your responsibility shrinks &#8594;</text>
  <line x1="116" y1="24" x2="452" y2="24" stroke="currentColor" stroke-width="1.2" stroke-opacity="0.7" marker-end="url(#cc_ar)"/>
  <g fill="currentColor" font-size="9" text-anchor="middle">
    <text x="144" y="42" font-weight="600">On-prem</text>
    <text x="234" y="42" font-weight="600">IaaS</text>
    <text x="324" y="42" font-weight="600">PaaS</text>
    <text x="414" y="42" font-weight="600">SaaS</text>
  </g>
  <g fill="currentColor" font-size="8.5" text-anchor="end">
    <text x="94" y="63">Application</text>
    <text x="94" y="93">Data</text>
    <text x="94" y="123">Runtime</text>
    <text x="94" y="153">OS</text>
    <text x="94" y="183">Virtualization</text>
    <text x="94" y="213">Hardware</text>
  </g>
  <g stroke="currentColor">
    <rect x="103" y="46" width="82" height="26" rx="2" fill="currentColor" fill-opacity="0.22" stroke-width="1"/>
    <rect x="103" y="76" width="82" height="26" rx="2" fill="currentColor" fill-opacity="0.22" stroke-width="1"/>
    <rect x="103" y="106" width="82" height="26" rx="2" fill="currentColor" fill-opacity="0.22" stroke-width="1"/>
    <rect x="103" y="136" width="82" height="26" rx="2" fill="currentColor" fill-opacity="0.22" stroke-width="1"/>
    <rect x="103" y="166" width="82" height="26" rx="2" fill="currentColor" fill-opacity="0.22" stroke-width="1"/>
    <rect x="103" y="196" width="82" height="26" rx="2" fill="currentColor" fill-opacity="0.22" stroke-width="1"/>
    <rect x="193" y="46" width="82" height="26" rx="2" fill="currentColor" fill-opacity="0.22" stroke-width="1"/>
    <rect x="193" y="76" width="82" height="26" rx="2" fill="currentColor" fill-opacity="0.22" stroke-width="1"/>
    <rect x="193" y="106" width="82" height="26" rx="2" fill="currentColor" fill-opacity="0.22" stroke-width="1"/>
    <rect x="193" y="136" width="82" height="26" rx="2" fill="currentColor" fill-opacity="0.22" stroke-width="1"/>
    <rect x="193" y="166" width="82" height="26" rx="2" fill="currentColor" fill-opacity="0.04" stroke-width="0.8" stroke-opacity="0.5"/>
    <rect x="193" y="196" width="82" height="26" rx="2" fill="currentColor" fill-opacity="0.04" stroke-width="0.8" stroke-opacity="0.5"/>
    <rect x="283" y="46" width="82" height="26" rx="2" fill="currentColor" fill-opacity="0.22" stroke-width="1"/>
    <rect x="283" y="76" width="82" height="26" rx="2" fill="currentColor" fill-opacity="0.22" stroke-width="1"/>
    <rect x="283" y="106" width="82" height="26" rx="2" fill="currentColor" fill-opacity="0.04" stroke-width="0.8" stroke-opacity="0.5"/>
    <rect x="283" y="136" width="82" height="26" rx="2" fill="currentColor" fill-opacity="0.04" stroke-width="0.8" stroke-opacity="0.5"/>
    <rect x="283" y="166" width="82" height="26" rx="2" fill="currentColor" fill-opacity="0.04" stroke-width="0.8" stroke-opacity="0.5"/>
    <rect x="283" y="196" width="82" height="26" rx="2" fill="currentColor" fill-opacity="0.04" stroke-width="0.8" stroke-opacity="0.5"/>
    <rect x="373" y="46" width="82" height="26" rx="2" fill="currentColor" fill-opacity="0.04" stroke-width="0.8" stroke-opacity="0.5"/>
    <rect x="373" y="76" width="82" height="26" rx="2" fill="currentColor" fill-opacity="0.04" stroke-width="0.8" stroke-opacity="0.5"/>
    <rect x="373" y="106" width="82" height="26" rx="2" fill="currentColor" fill-opacity="0.04" stroke-width="0.8" stroke-opacity="0.5"/>
    <rect x="373" y="136" width="82" height="26" rx="2" fill="currentColor" fill-opacity="0.04" stroke-width="0.8" stroke-opacity="0.5"/>
    <rect x="373" y="166" width="82" height="26" rx="2" fill="currentColor" fill-opacity="0.04" stroke-width="0.8" stroke-opacity="0.5"/>
    <rect x="373" y="196" width="82" height="26" rx="2" fill="currentColor" fill-opacity="0.04" stroke-width="0.8" stroke-opacity="0.5"/>
  </g>
  <g font-size="8" fill="currentColor">
    <rect x="150" y="242" width="14" height="12" rx="2" fill="currentColor" fill-opacity="0.22" stroke="currentColor" stroke-width="1"/>
    <text x="170" y="252" text-anchor="start">you manage</text>
    <rect x="252" y="242" width="14" height="12" rx="2" fill="currentColor" fill-opacity="0.04" stroke="currentColor" stroke-width="0.8" stroke-opacity="0.5"/>
    <text x="272" y="252" text-anchor="start">provider manages</text>
  </g>
  <defs><marker id="cc_ar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>Cloud computing is an umbrella over several service tiers. Each column is a line drawn through one stack: on-premises you run everything, and each step right hands the provider more layers from the hardware up — until, at SaaS, you manage nothing but your data. Picking a tier is choosing how much of the stack you still want to own.</figcaption>
</figure>

## Overview

The defining trait is elasticity: you can scale resources up when traffic spikes and back down when it fades, with near-zero upfront cost and ongoing fees for what you use. Under the hood it is [virtualization](/reference/virtualization/) at scale, run across large data centers — the same technology behind a single [virtual private server](/reference/virtual-private-server/), multiplied.

## Trade-offs

The cloud trades capital cost for recurring fees and adds network latency compared with a local device, so it is not always the right home for low-latency or always-on tasks. It often pairs with on-site hardware in combined-tier systems: a [home server](/reference/home-server/) captures and pre-processes locally while the cloud stores and serves the results — the model GopherTrunk fits, since the cloud cannot touch RF directly.

## Sources

[^wiki]: [Cloud computing](https://en.wikipedia.org/wiki/Cloud_computing) — Wikipedia, on on-demand computing over the internet.
