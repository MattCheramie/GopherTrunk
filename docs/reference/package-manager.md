---
slug: package-manager
title: Package manager & dependency management
entry_type: concept
category: testing-delivery
description: A package manager automates declaring, resolving, fetching, and pinning the external libraries a project depends on, using lockfiles to make dependency resolution reproducible.
keywords: package manager, dependency management, dependencies, lockfile, go modules, Cargo, npm, pip, transitive dependencies, reproducible, version resolution
aka: []
autolink: true
infobox:
  - { label: Category, value: "Developer tooling" }
  - { label: Job, value: "Declare, resolve, fetch, pin dependencies" }
  - { label: Reproducibility, value: "Lockfiles pin exact versions" }
  - { label: Examples, value: "go modules, Cargo, npm, pip" }
  - { label: Versioning, value: "Uses semantic versioning ranges" }
see_also: [build-systems, semantic-versioning, ci-cd, api, version-control, static-binary]
related_lessons:
  - { title: "Build systems, CI/CD & automation", url: /learn/intro-software-dev/build-and-ci-cd/ }
external:
  - { title: "Package manager — Wikipedia", url: https://en.wikipedia.org/wiki/Package_manager }
---

**A package manager** automates declaring, resolving, fetching, and pinning the external
libraries a project depends on, using lockfiles to make dependency resolution
reproducible.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 120" role="img" aria-label="A project manifest lists dependencies; the package manager resolves and fetches them, and a lockfile pins exact versions." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <rect x="20" y="42" width="80" height="36" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="60" y="60">manifest</text><text x="60" y="72" font-size="8">"needs v1.x"</text>
    <line x1="100" y1="60" x2="150" y2="60" stroke="currentColor" stroke-width="1.1"/>
    <rect x="150" y="42" width="100" height="36" rx="5" fill="currentColor" fill-opacity="0.14" stroke="currentColor" stroke-width="1.4"/><text x="200" y="60">resolve +</text><text x="200" y="72" font-size="8">fetch</text>
    <line x1="250" y1="60" x2="300" y2="60" stroke="currentColor" stroke-width="1.1"/>
    <circle cx="330" cy="42" r="12" fill="none" stroke="currentColor" stroke-width="1.2"/><circle cx="362" cy="60" r="12" fill="none" stroke="currentColor" stroke-width="1.2"/><circle cx="330" cy="78" r="12" fill="none" stroke="currentColor" stroke-width="1.2"/>
    <text x="345" y="103" font-size="8">deps (+ transitive)</text>
    <rect x="392" y="48" width="56" height="24" rx="4" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.2"/><text x="420" y="63" font-size="8">lockfile</text>
  </g>
</svg>
<figcaption>A manifest declares needs; the package manager resolves, fetches, and pins them in a lockfile.</figcaption>
</figure>

## What it does

Almost no project is self-contained — you stand on libraries written by others. A package
manager (Go modules, Cargo, npm, pip, and so on) records what your project needs in a
manifest and fetches it, including the *transitive* dependencies your direct dependencies
pull in. The libraries you depend on expose their capabilities through an
[API](/reference/api/), and the package manager's job is to get a compatible set of them
onto your machine and, with the [build system](/reference/build-systems/), into your
program.

## Lockfiles and reproducibility

A subtlety lurks here: a requirement like "version 1.x" can resolve to different exact
versions over time, the classic "works on my machine" problem. The fix is a **lockfile**
(`go.sum`, `Cargo.lock`, `package-lock.json`, `poetry.lock`) that pins the *exact* version
— often with a checksum — of every dependency, direct and transitive. Commit the lockfile,
and everyone, plus every [CI/CD](/reference/ci-cd/) run, installs a byte-for-byte
identical dependency tree. That is the foundation of a reproducible build: without pinned
inputs you can't have a reproducible output, which is why the lockfile lives in
[version control](/reference/version-control/) alongside the code.

## Versioning and trust

Dependency ranges and lockfiles both rest on [semantic versioning](/reference/semantic-versioning/):
the version number tells the resolver (and you) whether an upgrade is a safe patch, a
backward-compatible feature, or a breaking major change. Package managers also distribute
*your* library outward — publishing a versioned package others can depend on — which is
why a stable, well-versioned API matters as much for what you ship as for what you consume.
