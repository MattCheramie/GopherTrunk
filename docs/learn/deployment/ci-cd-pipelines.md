---
slug: ci-cd-pipelines
title: CI/CD pipelines
description: Automating build, test, and release so every commit is shippable — what a pipeline does, and how GopherTrunk uses GitHub Actions to build, test, and publish.
keywords: ci cd, continuous integration, continuous delivery, pipeline, github actions, automated build, automated tests, release automation
level: intermediate
status: full
prereq:
  - build-artifacts-and-versioning
faq:
  - q: What is the difference between CI and CD?
    a: Continuous integration (CI) automatically builds and tests every change as it's pushed, catching problems early. Continuous delivery/deployment (CD) takes a change that passed CI and automatically packages and releases it — to a registry, a release page, or straight to production. CI keeps the code always-working; CD makes shipping it automatic.
  - q: What is a CI/CD pipeline?
    a: A pipeline is an automated sequence of steps triggered by an event like a push or a tag — typically build, then run tests and checks, then (if all pass) produce and publish artifacts. It runs on a service like GitHub Actions, so the same steps happen identically every time, with no manual effort or forgotten steps.
---

# CI/CD pipelines

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**CI (continuous integration)** automatically **builds and tests** every change so the
code is always working. **CD (continuous delivery)** automatically **packages and
releases** changes that pass. Together they run as a **pipeline** triggered by pushes
and tags. GopherTrunk uses **GitHub Actions**: every PR is built, vetted, and tested,
and a version tag builds and publishes release artifacts.
</div>

Doing build-test-release by hand every time is slow and error-prone. **CI/CD**
automates it, so shipping is reliable and boring — exactly what you want.

## CI: keep the code always working

**Continuous integration** runs your checks automatically on every change. Push a
branch or open a pull request and a fresh machine builds the code and runs the tests —
no "it passed on my machine," because it's an independent, identical run every time.
(The [Git module](/learn/git/github-actions/) introduces GitHub Actions, the service
GopherTrunk uses.)

GopherTrunk's CI runs on every pull request and does the same gate a developer runs
locally, plus more:

```yaml
# from GopherTrunk's CI workflow
- run: go vet ./...
- name: gofmt
  run: |
    unformatted=$(gofmt -l .)
    if [ -n "$unformatted" ]; then exit 1; fi
- run: go build ./...
- run: go test -race -count=1 ./...
- run: make integration
```

Vet, formatting, build, race-tested unit tests, and integration tests — all
automatically, on every change, before it can merge. It even runs the USB backends on
Windows and macOS runners so cross-platform code can't quietly break.

## CD: automate the release

**Continuous delivery** extends the pipeline past testing into *shipping*. When
GopherTrunk gets a version tag, a separate release pipeline runs:

```yaml
on:
  push:
    tags:
      - "v*.*.*"        # e.g. v1.4.2
  workflow_dispatch:     # or trigger manually
```

That release job builds artifacts for every platform (Linux, macOS, Windows — all
`CGO_ENABLED=0` pure-Go builds), stamps in the [version](/learn/deployment/build-artifacts-and-versioning/)
from the tag, generates **`SHA256SUMS`** checksums, and uploads everything to a GitHub
Release. Tag `v1.4.2`, and minutes later the downloadable binaries and installer exist —
no manual build steps to forget.

## The shape of a pipeline

<figure class="figure" markdown="0">
<svg viewBox="0 0 520 90" role="img" aria-label="Pipeline stages: a trigger leads to build, then test and checks, then a gate, then publish artifacts." xmlns="http://www.w3.org/2000/svg">
  <g font-size="8.5" fill="currentColor" text-anchor="middle">
  <rect x="6" y="32" width="74" height="30" rx="4" fill="none" stroke="currentColor"/><text x="43" y="51">push / tag</text>
  <rect x="104" y="32" width="64" height="30" rx="4" fill="none" stroke="currentColor"/><text x="136" y="51">build</text>
  <rect x="192" y="32" width="94" height="30" rx="4" fill="none" stroke="currentColor"/><text x="239" y="47">test + vet +</text><text x="239" y="57">gofmt</text>
  <rect x="310" y="32" width="60" height="30" rx="4" fill="none" stroke="currentColor"/><text x="340" y="51">gate</text>
  <rect x="394" y="32" width="116" height="30" rx="4" fill="none" stroke="currentColor" stroke-width="1.7"/><text x="452" y="51">publish artifacts</text>
  </g>
  <g stroke="currentColor"><line x1="80" y1="47" x2="104" y2="47"/><line x1="168" y1="47" x2="192" y2="47"/><line x1="286" y1="47" x2="310" y2="47"/><line x1="370" y1="47" x2="394" y2="47"/></g>
</svg>
<figcaption>A pipeline: an event triggers build and checks; only if the gate passes are artifacts published.</figcaption>
</figure>

## Why it matters

CI/CD turns shipping from a careful manual ritual into an automatic, repeatable process.
The tests always run, the artifacts are always built the same way, and a bad change is
caught before it reaches users. It's the automation layer that ties together everything
in this module — [artifacts](/learn/deployment/build-artifacts-and-versioning/),
[images](/learn/deployment/images-and-registries/), and
[versioning](/learn/deployment/build-artifacts-and-versioning/) — into a hands-off flow.

<div class="knowledge-check" data-quiz data-correct-msg="Right — CI builds and tests every change; CD automates packaging and releasing." markdown="0">
  <p class="knowledge-check__q">Quick check: what does the CI half of CI/CD do?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="correct">Automatically build and test every change as it's pushed</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Delete old releases</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Write the code for you</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **CI** automatically **builds and tests** every change; **CD** automatically
  **packages and releases** what passes.
- A **pipeline** is triggered by events (push, tag) and runs the same steps every time.
- GopherTrunk's CI gates PRs on **vet, gofmt, build, race tests, and integration
  tests**; a **version tag** builds and publishes release artifacts with checksums.
- CI/CD makes shipping **reliable and repeatable**.

Next up: where to actually run your deployment — cloud and VPS options.
