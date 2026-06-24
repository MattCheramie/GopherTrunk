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

## Overview

Each virtual machine believes it has its own hardware; the hypervisor mediates access to the real resources and keeps the guests separated. There are two broad kinds. A *Type 1* (bare-metal) hypervisor runs directly on the hardware and is what data centers and cloud platforms use for performance and density. A *Type 2* (hosted) hypervisor runs as an application on top of an ordinary [operating system](/reference/operating-system/), which is convenient for desktops and testing. Either way, the hypervisor is the engine that makes [virtualization](/reference/virtualization/) possible.

## Where it fits

The hypervisor is what lets one [server](/reference/server/) be sliced into many [virtual private servers](/reference/virtual-private-server/) and underpins [infrastructure as a service](/reference/infrastructure-as-a-service/). It provides stronger isolation than a [container](/reference/container/), at the cost of running a full guest OS per machine; a [bare-metal server](/reference/bare-metal-server/) deliberately omits it. Running GopherTrunk inside a VM works, though passing a USB SDR through the hypervisor to the guest takes extra configuration.

## Sources

[^wiki]: [Hypervisor](https://en.wikipedia.org/wiki/Hypervisor) — Wikipedia, on Type 1 and Type 2 hypervisors.
