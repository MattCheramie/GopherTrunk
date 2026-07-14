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

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 224" role="img" aria-label="A flow from left to right. A Dockerfile is built into an image made of stacked read-only layers — a base image, dependencies, and the app. Running the image produces a container: the same read-only layers with a thin writable layer added on top." xmlns="http://www.w3.org/2000/svg">
  <rect x="18" y="80" width="80" height="56" rx="5" fill="currentColor" fill-opacity="0.1" stroke="currentColor" stroke-width="1.2"/>
  <text x="58" y="104" text-anchor="middle" font-size="9" fill="currentColor" font-weight="600">Dockerfile</text>
  <text x="58" y="118" text-anchor="middle" font-size="7.5" fill="currentColor" fill-opacity="0.85">the recipe</text>
  <g font-size="8" fill="currentColor" text-anchor="middle">
    <rect x="152" y="120" width="104" height="24" rx="3" fill="currentColor" fill-opacity="0.08" stroke="currentColor" stroke-width="1"/><text x="204" y="136">base image</text>
    <rect x="152" y="96" width="104" height="24" rx="3" fill="currentColor" fill-opacity="0.14" stroke="currentColor" stroke-width="1"/><text x="204" y="112">dependencies</text>
    <rect x="152" y="72" width="104" height="24" rx="3" fill="currentColor" fill-opacity="0.2" stroke="currentColor" stroke-width="1"/><text x="204" y="88">app</text>
    <text x="204" y="62" font-size="9" font-weight="600">Image · read-only</text>
  </g>
  <g font-size="8" fill="currentColor" text-anchor="middle">
    <rect x="330" y="120" width="104" height="24" rx="3" fill="currentColor" fill-opacity="0.06" stroke="currentColor" stroke-width="1"/><text x="382" y="136">base image</text>
    <rect x="330" y="96" width="104" height="24" rx="3" fill="currentColor" fill-opacity="0.1" stroke="currentColor" stroke-width="1"/><text x="382" y="112">dependencies</text>
    <rect x="330" y="72" width="104" height="24" rx="3" fill="currentColor" fill-opacity="0.16" stroke="currentColor" stroke-width="1"/><text x="382" y="88">app</text>
    <rect x="330" y="48" width="104" height="24" rx="3" fill="currentColor" fill-opacity="0.04" stroke="currentColor" stroke-width="1" stroke-dasharray="4 3"/><text x="382" y="64">writable layer</text>
    <text x="382" y="38" font-size="9" font-weight="600">Container · running</text>
  </g>
  <g stroke="currentColor" stroke-width="1.3" fill="none">
    <line x1="98" y1="108" x2="150" y2="108" stroke-opacity="0.8" marker-end="url(#dk_ar)"/>
    <line x1="258" y1="108" x2="328" y2="108" stroke-opacity="0.8" marker-end="url(#dk_ar)"/>
  </g>
  <g font-size="7.5" fill="currentColor" fill-opacity="0.85" text-anchor="middle">
    <text x="124" y="100">docker build</text>
    <text x="293" y="100">docker run</text>
  </g>
  <text x="230" y="176" text-anchor="middle" font-size="8" fill="currentColor" fill-opacity="0.9">the read-only layers are shared and cached; only the top writable layer differs per container</text>
  <defs><marker id="dk_ar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>A Dockerfile is built into an image: a stack of read-only layers — base, dependencies, then your app — each cached and reusable. Running the image adds a thin writable layer on top, and that is the container. Because the heavy layers are shared, many containers from one image cost almost nothing extra.</figcaption>
</figure>

## Overview

A developer describes an image in a *Dockerfile* — a recipe listing a base image, dependencies, and the application — and Docker builds it into a layered, portable image. That image can be pushed to a registry such as Docker Hub and pulled to run identically anywhere Docker is installed. Docker did not invent the underlying kernel features (namespaces and cgroups), but its tooling and image format made them easy enough that containers became the default way to ship server software.[^home]

## Where it fits

Docker handles single containers on one host; when you need to run many containers across many machines, [Kubernetes](/reference/kubernetes/) orchestrates them. Containers are lighter than the full [virtualization](/reference/virtualization/) a [hypervisor](/reference/hypervisor/) provides. A Docker image of GopherTrunk gives a clean, reproducible decoder build; on a capture node you pass the SDR device through from the host into the container.

## Sources

[^wiki]: [Docker (software)](https://en.wikipedia.org/wiki/Docker_(software)) — Wikipedia, on Docker's history and design.
[^home]: [Docker](https://www.docker.com/) — the project's official site.
