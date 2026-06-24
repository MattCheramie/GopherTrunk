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

## Overview

With IaaS the provider owns the [data center](/reference/data-center/), the physical servers, and the [virtualization](/reference/virtualization/) layer; you receive a virtual machine and are responsible for the [operating system](/reference/operating-system/), patching, services, and your application. It is the most flexible cloud tier and the closest to running your own server, billed on demand by the hour or second. A [virtual private server](/reference/virtual-private-server/) is essentially a small, fixed slice of IaaS.

## Where it fits

IaaS is the bottom rung of the cloud stack, beneath [platform as a service](/reference/platform-as-a-service/) (which also runs the OS and runtime) and [software as a service](/reference/software-as-a-service/) (a finished application). Choose IaaS when you need full control of the environment but not physical hardware; choose a [bare-metal server](/reference/bare-metal-server/) when virtualization overhead or multi-tenancy is unacceptable. A GopherTrunk back end fits IaaS well for storage and serving, though capture still lives at the antenna.

## Sources

[^wiki]: [Infrastructure as a service](https://en.wikipedia.org/wiki/Infrastructure_as_a_service) — Wikipedia, on the IaaS cloud model.
