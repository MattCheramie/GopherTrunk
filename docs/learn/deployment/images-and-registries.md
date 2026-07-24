---
slug: images-and-registries
title: Images & registries
description: How container images are named, tagged, stored, and shared through registries, and how a pull-and-run makes deploying somewhere new a one-line operation.
keywords: container registry, docker image tag, docker hub, ghcr, image digest, docker pull, docker push, image naming
level: intermediate
status: full
prereq:
  - writing-a-dockerfile
faq:
  - q: What is a container registry?
    a: A registry is a server that stores and distributes container images, like a package repository for containers. You push an image to it from wherever you built it, and any machine you want to deploy on pulls it back down. Docker Hub and GitHub's ghcr.io are common public registries.
  - q: What is the difference between an image tag and a digest?
    a: A tag is a human-friendly label like :latest or :1.4.2 that you attach to an image and can move to point at a new build. A digest is a cryptographic hash of the exact image contents — it never changes and always refers to the identical bytes. Use tags for convenience, digests when you need to pin an exact, immutable image.
---

# Images & registries

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
A container **image** is named `registry/name:tag`. A **registry** (Docker Hub,
GitHub's `ghcr.io`) stores and distributes images: you **push** from where you built,
and any host **pulls** to deploy. A **tag** (`:latest`, `:1.4.2`) is a movable label; a
**digest** is an immutable hash of the exact bytes. Pull-and-run makes deploying
somewhere new a one-liner.
</div>

You've built an image. Now how do you get it onto a server? Through a **registry** —
this lesson covers naming, storing, and sharing images.

## How images are named

A full image name has three parts:

```text
   ghcr.io/mattcheramie/gophertrunk : latest
   \______/ \___________________/    \____/
   registry        repository          tag
```

- **registry** — where it lives (`docker.io` for Docker Hub, `ghcr.io` for GitHub's
  container registry). Omit it and Docker assumes Docker Hub.
- **repository** — the image's name, usually `owner/project`.
- **tag** — which version, like `latest`, `1.4.2`, or `dev`.

## Tags vs digests

A **tag** is a friendly, *movable* label — `:latest` points at whatever you last
pushed as latest. That's convenient but means "latest" is a moving target. A
**digest** (`sha256:...`) is a hash of the image's exact contents; it never moves, so
pinning to a digest guarantees you run the *identical* bytes you tested — the
immutability idea from [artifacts & versioning](/learn/deployment/build-artifacts-and-versioning/)
applied to images.

## Push and pull

Sharing an image is two commands. From where you built it:

```bash
docker tag gophertrunk:dev ghcr.io/mattcheramie/gophertrunk:1.4.2
docker push ghcr.io/mattcheramie/gophertrunk:1.4.2
```

And on any machine you want to deploy on:

```bash
docker pull ghcr.io/mattcheramie/gophertrunk:1.4.2
docker run ghcr.io/mattcheramie/gophertrunk:1.4.2
```

That `pull` + `run` is the whole deploy step — no build tools, no dependencies to
install on the target, just fetch the image and start it. This is the payoff of
containerizing: **deploying somewhere new is a one-liner.**

<figure class="figure" markdown="0">
<svg viewBox="0 0 520 120" role="img" aria-label="A build machine pushes an image up to a registry in the middle; two servers pull the same image down from the registry." xmlns="http://www.w3.org/2000/svg">
  <rect x="10" y="45" width="90" height="30" rx="5" fill="none" stroke="currentColor"/><text x="55" y="64" text-anchor="middle" font-size="9" fill="currentColor">build machine</text>
  <line x1="100" y1="55" x2="205" y2="45" stroke="currentColor" stroke-width="1.5" marker-end="url(#r1)"/><text x="150" y="40" font-size="8" fill="currentColor">push</text>
  <rect x="205" y="35" width="110" height="42" rx="5" fill="none" stroke="currentColor" stroke-width="1.8"/><text x="260" y="60" text-anchor="middle" font-size="10" fill="currentColor">registry</text>
  <line x1="315" y1="50" x2="420" y2="35" stroke="currentColor" stroke-width="1.5" marker-end="url(#r1)"/>
  <line x1="315" y1="62" x2="420" y2="90" stroke="currentColor" stroke-width="1.5" marker-end="url(#r1)"/><text x="375" y="45" font-size="8" fill="currentColor">pull</text>
  <rect x="420" y="20" width="90" height="28" rx="5" fill="none" stroke="currentColor"/><text x="465" y="38" text-anchor="middle" font-size="9" fill="currentColor">server A</text>
  <rect x="420" y="78" width="90" height="28" rx="5" fill="none" stroke="currentColor"/><text x="465" y="96" text-anchor="middle" font-size="9" fill="currentColor">server B</text>
  <defs><marker id="r1" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>Build once, push to a registry, pull the identical image onto any number of servers.</figcaption>
</figure>

## A note on GopherTrunk

GopherTrunk's [docker-compose](/learn/deployment/docker-compose/) file references the
image name `ghcr.io/mattcheramie/gophertrunk`, but by default you **build it locally**
from the repo Dockerfile (`docker compose build`). GopherTrunk's *official* automated
releases are downloadable binaries and installers rather than a pushed image — a good
reminder that "artifact" can mean a container image *or* a plain binary, and a project
can use whichever fits.

<div class="knowledge-check" data-quiz data-correct-msg="Right — a registry stores images so any host can pull and run them." markdown="0">
  <p class="knowledge-check__q">Quick check: what does a container registry let you do?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Compile source code into an image</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Store an image centrally so any machine can pull and run it</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Run tests on the image automatically</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- An image is named **`registry/repository:tag`**; omit the registry and Docker Hub is
  assumed.
- A **tag** is a movable label; a **digest** is an immutable hash of exact contents.
- **`push`** to a registry, **`pull`** on any host — pull-and-run is the whole deploy.
- GopherTrunk's compose builds the image locally; its official releases are binaries.

Next up: running an app plus its dependencies together with Docker Compose.
