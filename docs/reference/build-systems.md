---
slug: build-systems
title: Build systems
entry_type: concept
category: testing-delivery
description: A build system is the tooling that transforms source code plus dependencies into a runnable artifact — compiling, linking, and packaging — ideally as a deterministic, reproducible function of its inputs.
keywords: build systems, build tool, Make, Makefile, compile, link, go build, Cargo, npm, Maven, Gradle, artifact, reproducible build
aka: []
autolink: true
infobox:
  - { label: Category, value: "Developer tooling" }
  - { label: Job, value: "Source + deps → runnable artifact" }
  - { label: Classic tool, value: "Make / Makefile" }
  - { label: Language tools, value: "go build, Cargo, npm, Maven, Gradle" }
  - { label: Ideal, value: "Deterministic, reproducible output" }
see_also: [compiler, package-manager, ci-cd, semantic-versioning, static-binary, cross-compilation, version-control]
related_lessons:
  - { title: "Build systems, CI/CD & automation", url: /learn/intro-software-dev/build-and-ci-cd/ }
external:
  - { title: "Build automation — Wikipedia", url: https://en.wikipedia.org/wiki/Build_automation }
---

**A build system** is the tooling that transforms source code plus its dependencies into
a runnable artifact — compiling, linking, and packaging — ideally as a deterministic
function of its inputs.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 120" role="img" aria-label="Source files and dependencies feed into a build step that produces a single runnable artifact." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <rect x="20" y="22" width="80" height="30" rx="4" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="60" y="41">source</text>
    <rect x="20" y="68" width="80" height="30" rx="4" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="60" y="87">dependencies</text>
    <line x1="100" y1="37" x2="160" y2="55" stroke="currentColor" stroke-width="1.1"/><line x1="100" y1="83" x2="160" y2="65" stroke="currentColor" stroke-width="1.1"/>
    <rect x="160" y="42" width="90" height="36" rx="5" fill="currentColor" fill-opacity="0.14" stroke="currentColor" stroke-width="1.4"/><text x="205" y="64">build</text>
    <line x1="250" y1="60" x2="320" y2="60" stroke="currentColor" stroke-width="1.1"/>
    <rect x="320" y="42" width="120" height="36" rx="5" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.3"/><text x="380" y="64">artifact</text>
  </g>
</svg>
<figcaption>A build is a function from source and dependencies to a runnable artifact.</figcaption>
</figure>

## What a build does

A build transforms source into something runnable. The exact steps vary by language but
the idea is universal: a [compiler](/reference/compiler/) translates source into machine
code or bytecode, and a *linker* stitches the pieces into an executable. **Make** and its
descendants are the classic orchestrators — a `Makefile` declares targets and their
dependencies, and `make` rebuilds only what changed. That principle (describe outputs in
terms of inputs, rebuild only the stale parts) underlies nearly every build tool since.
Modern ecosystems wrap this per language: `go build`, `cargo build`, `npm`/`yarn`, Maven,
and Gradle each know their language's conventions so you rarely hand-write build steps.

## Reproducibility

The common thread is that a build is a *function* from source (plus dependencies) to an
artifact, and the more deterministic that function, the more you can trust and automate
it. **Reproducible builds** mean the same source and the same pinned dependencies always
produce a bit-for-bit identical artifact, regardless of who builds it or when. That
depends on pinned inputs, which is why a build system works hand in hand with a
[package manager](/reference/package-manager/) and its lockfiles — without pinned inputs,
you can't have a reproducible output. Many builds also support
[cross-compilation](/reference/cross-compilation/), emitting a
[static binary](/reference/static-binary/) for other platforms from one machine.

## Where it fits

The build sits at the heart of a [CI/CD](/reference/ci-cd/) pipeline: on every push from
[version control](/reference/version-control/), the build runs, the
[tests](/reference/unit-testing/) follow, and a successful run yields a versioned artifact
ready to release under [semantic versioning](/reference/semantic-versioning/). A
deterministic build is what makes that whole chain trustworthy.
