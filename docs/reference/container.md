---
slug: container
title: Container
entry_type: concept
category: hw-servers
description: A container is a lightweight, isolated package that bundles an application with its dependencies and runs as an ordinary process on a shared operating-system kernel, making software portable and reproducible.
keywords: container, containerization, OS-level virtualization, image, namespaces, cgroups, portable app
infobox:
  - { label: Type, value: OS-level isolation }
  - { label: Shares, value: Host kernel }
  - { label: Packages, value: App + dependencies }
  - { label: Contrast, value: Virtual machine }
see_also: [docker, kubernetes, virtualization, hypervisor, operating-system, serverless-computing]
cite_urls:
  - https://en.wikipedia.org/wiki/Containerization_(computing)
---

A **container** is a lightweight, isolated package that bundles an application with its dependencies and runs as an ordinary process on a shared [operating-system](/reference/operating-system/) kernel — making software portable and reproducible across machines.[^wiki]

## Overview

Containers are a form of OS-level [virtualization](/reference/virtualization/). Unlike a virtual machine, a container does not carry its own kernel or boot an operating system; it shares the host's kernel and is isolated using kernel features such as namespaces (separate views of processes, files, and networking) and cgroups (resource limits). A container starts from an *image* — a layered, read-only template — so the same image runs identically on a laptop, a server, or the cloud. The result is fast startup, low overhead, and "it works the same everywhere" packaging.

## Where it fits

Containers are usually built and run with [Docker](/reference/docker/) and orchestrated at scale by [Kubernetes](/reference/kubernetes/). They are lighter than full virtualization under a [hypervisor](/reference/hypervisor/) but offer weaker isolation, since all containers share one kernel. Packaging GopherTrunk in a container makes the decoder easy to deploy reproducibly, though USB radio access (the SDR dongle) must be passed through from the host.

## Sources

[^wiki]: [Containerization (computing)](https://en.wikipedia.org/wiki/Containerization_(computing)) — Wikipedia, on OS-level containers.
