---
slug: infrastructure-as-a-service
title: Infrastructure as a service (IaaS)
entry_type: concept
category: hw-servers
description: Infrastructure as a service (IaaS) is a cloud model where the provider rents out raw compute, storage, and networking — typically as virtual machines — and the customer manages everything above the hardware.
keywords: IaaS, infrastructure as a service, cloud VM, virtual machine, compute storage networking, self-managed cloud
aka: [IaaS]
infobox:
  - { label: Type, value: Cloud service model }
  - { label: Provider runs, value: Physical hardware, virtualization }
  - { label: You run, value: OS and everything above }
  - { label: Typical unit, value: Virtual machine }
see_also: [platform-as-a-service, software-as-a-service, cloud-computing, virtualization, bare-metal-server, virtual-private-server]
related_lessons:
  - { title: "Virtual private servers", url: /learn/intro-hardware/vps/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Infrastructure_as_a_service
---

**Infrastructure as a service** (**IaaS**) is a [cloud computing](/reference/cloud-computing/) model in which the provider rents out raw compute, storage, and networking — usually as virtual machines — and you manage everything above the bare hardware.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 464 268" role="img" aria-label="A grid of who manages each layer across four tiers. Rows are application, data, runtime, operating system, virtualization, and hardware. Columns are on-premises, IaaS, PaaS, and SaaS. Filled cells are managed by you; light cells are managed by the provider. Moving right, the provider takes over more layers from the bottom up." xmlns="http://www.w3.org/2000/svg">
  <g fill="currentColor" font-size="9" text-anchor="middle">
    <text x="144" y="36" font-weight="600">On-prem</text>
    <text x="234" y="36" font-weight="600">IaaS</text>
    <text x="324" y="36" font-weight="600">PaaS</text>
    <text x="414" y="36" font-weight="600">SaaS</text>
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
</svg>
<figcaption>Each tier is a line drawn through the same stack. On-premises you run everything; IaaS hands you a virtual machine and up (you keep the OS and above); PaaS also runs the runtime; SaaS is a finished application. Moving right, the provider absorbs more layers from the hardware up.</figcaption>
</figure>

## Overview

With IaaS the provider owns the [data center](/reference/data-center/), the physical servers, and the [virtualization](/reference/virtualization/) layer; you receive a virtual machine and are responsible for the [operating system](/reference/operating-system/), patching, services, and your application. It is the most flexible cloud tier and the closest to running your own server, billed on demand by the hour or second. A [virtual private server](/reference/virtual-private-server/) is essentially a small, fixed slice of IaaS.

## Where it fits

IaaS is the bottom rung of the cloud stack, beneath [platform as a service](/reference/platform-as-a-service/) (which also runs the OS and runtime) and [software as a service](/reference/software-as-a-service/) (a finished application). Choose IaaS when you need full control of the environment but not physical hardware; choose a [bare-metal server](/reference/bare-metal-server/) when virtualization overhead or multi-tenancy is unacceptable. A GopherTrunk back end fits IaaS well for storage and serving, though capture still lives at the antenna.

## Sources

[^wiki]: [Infrastructure as a service](https://en.wikipedia.org/wiki/Infrastructure_as_a_service) — Wikipedia, on the IaaS cloud model.
