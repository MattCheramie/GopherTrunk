---
slug: docker
title: Docker
entry_type: concept
category: hw-servers
description: Docker is a popular platform and toolset for building, sharing, and running containers, which packaged container images and a simple command line and made containerization mainstream.
keywords: Docker, container, Dockerfile, container image, Docker Hub, containerization
autolink: true
infobox:
  - { label: Type, value: Container platform }
  - { label: Introduced, value: "2013" }
  - { label: Builds from, value: Dockerfile }
  - { label: Shares via, value: Image registry }
see_also: [container, kubernetes, virtualization, operating-system, hypervisor, serverless-computing]
cite_urls:
  - https://en.wikipedia.org/wiki/Docker_(software)
  - https://www.docker.com/
---

**Docker** is a popular platform and toolset for building, sharing, and running [containers](/reference/container/). Introduced in 2013, it standardized container images and a simple command line and brought containerization into the mainstream.[^wiki]

## Overview

A developer describes an image in a *Dockerfile* — a recipe listing a base image, dependencies, and the application — and Docker builds it into a layered, portable image. That image can be pushed to a registry such as Docker Hub and pulled to run identically anywhere Docker is installed. Docker did not invent the underlying kernel features (namespaces and cgroups), but its tooling and image format made them easy enough that containers became the default way to ship server software.[^home]

## Where it fits

Docker handles single containers on one host; when you need to run many containers across many machines, [Kubernetes](/reference/kubernetes/) orchestrates them. Containers are lighter than the full [virtualization](/reference/virtualization/) a [hypervisor](/reference/hypervisor/) provides. A Docker image of GopherTrunk gives a clean, reproducible decoder build; on a capture node you pass the SDR device through from the host into the container.

## Sources

[^wiki]: [Docker (software)](https://en.wikipedia.org/wiki/Docker_(software)) — Wikipedia, on Docker's history and design.
[^home]: [Docker](https://www.docker.com/) — the project's official site.
