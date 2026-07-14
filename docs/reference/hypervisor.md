---
slug: hypervisor
title: Hypervisor
entry_type: concept
category: hw-servers
description: A hypervisor is software or firmware that creates and runs virtual machines, dividing one physical computer's resources among several isolated guest operating systems.
keywords: hypervisor, virtual machine monitor, VMM, type 1, type 2, bare metal, guest OS
aka: [Virtual machine monitor, VMM]
infobox:
  - { label: Type, value: Virtualization layer }
  - { label: Creates, value: Virtual machines }
  - { label: "Type 1", value: Runs on bare metal }
  - { label: "Type 2", value: Runs on a host OS }
see_also: [virtualization, virtual-private-server, container, bare-metal-server, infrastructure-as-a-service, server]
cite_urls:
  - https://en.wikipedia.org/wiki/Hypervisor
---

A **hypervisor** is software or firmware that creates and runs virtual machines, dividing one physical computer's CPU, memory, and I/O among several isolated guest operating systems.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 420 232" role="img" aria-label="One physical computer at the bottom, an emphasized hypervisor layer above it drawn with a heavier border, and three isolated virtual machines stacked on top, each with its own guest operating system and application. A note below explains that a Type 1 hypervisor runs on bare metal while a Type 2 runs on a host operating system." xmlns="http://www.w3.org/2000/svg">
  <g fill="currentColor" font-size="9" text-anchor="middle">
    <g stroke="currentColor">
      <rect x="48" y="44" width="104" height="92" rx="5" fill="none" stroke-width="1.2" stroke-dasharray="4 3"/>
      <text x="100" y="59" font-size="8.5" font-weight="600">VM 1</text>
      <rect x="55" y="66" width="90" height="28" rx="3" fill="currentColor" fill-opacity="0.2" stroke-width="1"/><text x="100" y="84">App</text>
      <rect x="55" y="98" width="90" height="32" rx="3" fill="currentColor" fill-opacity="0.1" stroke-width="1"/><text x="100" y="118" font-size="8.5">Guest OS</text>
      <rect x="162" y="44" width="104" height="92" rx="5" fill="none" stroke-width="1.2" stroke-dasharray="4 3"/>
      <text x="214" y="59" font-size="8.5" font-weight="600">VM 2</text>
      <rect x="169" y="66" width="90" height="28" rx="3" fill="currentColor" fill-opacity="0.2" stroke-width="1"/><text x="214" y="84">App</text>
      <rect x="169" y="98" width="90" height="32" rx="3" fill="currentColor" fill-opacity="0.1" stroke-width="1"/><text x="214" y="118" font-size="8.5">Guest OS</text>
      <rect x="276" y="44" width="96" height="92" rx="5" fill="none" stroke-width="1.2" stroke-dasharray="4 3"/>
      <text x="324" y="59" font-size="8.5" font-weight="600">VM 3</text>
      <rect x="283" y="66" width="82" height="28" rx="3" fill="currentColor" fill-opacity="0.2" stroke-width="1"/><text x="324" y="84">App</text>
      <rect x="283" y="98" width="82" height="32" rx="3" fill="currentColor" fill-opacity="0.1" stroke-width="1"/><text x="324" y="118" font-size="8.5">Guest OS</text>
    </g>
    <rect x="40" y="142" width="340" height="30" rx="4" fill="currentColor" fill-opacity="0.3" stroke="currentColor" stroke-width="2"/><text x="210" y="161" font-weight="600">Hypervisor — creates &amp; isolates the VMs</text>
    <rect x="40" y="176" width="340" height="26" rx="4" fill="currentColor" fill-opacity="0.08" stroke="currentColor" stroke-width="1.2"/><text x="210" y="193">One physical computer · CPU · memory · storage</text>
    <text x="210" y="223" font-size="8" fill-opacity="0.85">Type 1 sits on the bare hardware; Type 2 runs on a host OS</text>
  </g>
  <g stroke="currentColor" stroke-width="1" stroke-opacity="0.5">
    <line x1="100" y1="136" x2="100" y2="142"/>
    <line x1="214" y1="136" x2="214" y2="142"/>
    <line x1="324" y1="136" x2="324" y2="142"/>
  </g>
</svg>
<figcaption>The hypervisor is the emphasized band in the middle — the engine that turns one machine into many. It carves the host's CPU, memory, and storage into virtual machines and walls each off, so one guest's crash or load can't reach another. A Type 1 hypervisor runs straight on the hardware; a Type 2 runs as an app on a host OS.</figcaption>
</figure>

## Overview

Each virtual machine believes it has its own hardware; the hypervisor mediates access to the real resources and keeps the guests separated. There are two broad kinds. A *Type 1* (bare-metal) hypervisor runs directly on the hardware and is what data centers and cloud platforms use for performance and density. A *Type 2* (hosted) hypervisor runs as an application on top of an ordinary [operating system](/reference/operating-system/), which is convenient for desktops and testing. Either way, the hypervisor is the engine that makes [virtualization](/reference/virtualization/) possible.

## Where it fits

The hypervisor is what lets one [server](/reference/server/) be sliced into many [virtual private servers](/reference/virtual-private-server/) and underpins [infrastructure as a service](/reference/infrastructure-as-a-service/). It provides stronger isolation than a [container](/reference/container/), at the cost of running a full guest OS per machine; a [bare-metal server](/reference/bare-metal-server/) deliberately omits it. Running GopherTrunk inside a VM works, though passing a USB SDR through the hypervisor to the guest takes extra configuration.

## Sources

[^wiki]: [Hypervisor](https://en.wikipedia.org/wiki/Hypervisor) — Wikipedia, on Type 1 and Type 2 hypervisors.
