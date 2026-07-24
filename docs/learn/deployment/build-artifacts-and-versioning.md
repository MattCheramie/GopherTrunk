---
slug: build-artifacts-and-versioning
title: Build artifacts & versioning
description: What a build produces, why one reproducible artifact beats rebuilding everywhere, and how version tags let you ship and roll back with confidence.
keywords: build artifact, versioning, semantic versioning, semver, reproducible build, immutable artifact, rollback, release tag, deployment artifact
level: beginner
status: full
prereq:
  - what-is-deployment
faq:
  - q: What is a build artifact?
    a: A build artifact is the concrete output of compiling or packaging your software — a binary, a container image, a zip file — ready to run or deploy. The idea is to build it once and deploy that exact same artifact everywhere, rather than rebuilding on each machine and risking subtle differences.
  - q: What is semantic versioning?
    a: Semantic versioning (SemVer) labels releases as MAJOR.MINOR.PATCH — for example 1.4.2. You bump PATCH for backward-compatible fixes, MINOR for backward-compatible new features, and MAJOR for breaking changes. The scheme lets anyone tell at a glance how risky an upgrade is.
---

# Build artifacts & versioning

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
A **build artifact** is the concrete thing you deploy — a binary, a container image, a
tarball. Build it **once** and ship that *same* immutable artifact everywhere, rather
than rebuilding per machine. Label each with a **version** (often **SemVer**:
MAJOR.MINOR.PATCH) so you know exactly what's running and can **roll back** to a known-
good one instantly.
</div>

Before you can deploy something, you have to know precisely *what* that something is.
This lesson is about the artifact you ship and how versioning keeps releases sane.

## What a build produces

Compiling or packaging your software yields an **artifact** — the deployable output.
What it looks like depends on the stack:

| Stack | Typical artifact |
|-------|------------------|
| Go (like GopherTrunk) | a single static binary |
| Any language, containerized | a container image |
| Python / Node | a package or a zip of files |

Because Go compiles to [one self-contained binary](/learn/programming-go/hello-go/),
GopherTrunk's artifacts are simple: a `gophertrunk` executable per platform. Its
release process builds Linux, macOS, and Windows artifacts and publishes them with
checksums.

## Build once, deploy that exact thing

The key discipline: **an artifact is immutable, and you deploy the same one
everywhere.** Rebuilding on each server invites the "works on my machine" gremlins from
[lesson 1](/learn/deployment/what-is-deployment/) — a slightly different compiler,
library, or timestamp. Build the artifact a single time, test *that* artifact, and
promote the very same bytes from staging to production. What you tested is exactly what
you ship.

## Versioning: naming what you ship

Every artifact needs a **version** so you can talk about it precisely. The common
scheme is **semantic versioning**:

```text
   1   .   4   .   2
 MAJOR   MINOR   PATCH
   |       |       |
   |       |       +-- backward-compatible bug fixes
   |       +---------- backward-compatible new features
   +------------------ breaking changes
```

GopherTrunk stamps the version *into the binary at build time* — its build injects the
tag via Go's linker (`-ldflags "-X …version.Version=<tag>"`), so a running instance can
report exactly which release it is. Versions usually come from
[Git tags](/learn/git/releases-and-tags/): tag `v1.4.2`, and the release build produces
the `1.4.2` artifacts.

## Why versioning enables rollback

Immutable, versioned artifacts give you a safety net. If `v1.4.2` misbehaves in
production, you don't debug live — you **roll back** by redeploying the previous
known-good `v1.4.1` artifact, which still exists untouched. That ability to instantly
return to a working version is one of the biggest reasons to build discrete, versioned
artifacts, and it underpins the safe-update practices in
[monitoring & updates](/learn/deployment/monitoring-and-updates/).

<div class="knowledge-check" data-quiz data-correct-msg="Right — build one immutable artifact and deploy that same one everywhere." markdown="0">
  <p class="knowledge-check__q">Quick check: why build an artifact once and deploy that same artifact everywhere?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">To save disk space on the build server</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">So what you tested is exactly what you ship, with no rebuild differences</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Because rebuilding is impossible</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- A **build artifact** is the deployable output — a binary, image, or package.
- Build it **once** and deploy the same **immutable** artifact everywhere.
- **Version** each artifact (often **SemVer** MAJOR.MINOR.PATCH); GopherTrunk stamps
  the version into the binary.
- Versioned artifacts make instant **rollback** to a known-good release possible.

Next up: Unit 2 — the technology that bundles an artifact with everything it needs: containers.
