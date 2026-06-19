---
slug: ci-cd
title: CI/CD
entry_type: concept
category: testing-delivery
description: CI/CD is the practice of automatically building and testing code on every change (continuous integration) and then automating its path to release (continuous delivery/deployment) through a versioned pipeline.
keywords: CI/CD, continuous integration, continuous delivery, continuous deployment, pipeline, build, test, deploy, GitHub Actions, automation, artifacts
aka: [continuous integration "CI","CD", CI/CD]
autolink: true
infobox:
  - { label: Category, value: "Automation / delivery practice" }
  - { label: CI, value: "Build & test on every push" }
  - { label: CD, value: "Continuous Delivery / Deployment" }
  - { label: Pipeline, value: "lint → build → test → package → deploy" }
  - { label: Tools, value: "GitHub Actions, GitLab CI, Jenkins" }
see_also: [build-systems, version-control, unit-testing, integration-testing, end-to-end-testing, package-manager, semantic-versioning]
related_lessons:
  - { title: "GitHub Actions & CI/CD", url: /learn/git/github-actions/ }
  - { title: "Build systems & CI/CD", url: /learn/intro-software-dev/build-and-ci-cd/ }
external:
  - { title: "CI/CD — Wikipedia", url: https://en.wikipedia.org/wiki/CI/CD }
---

**CI/CD** is the practice of automatically building and testing code on every change
(continuous integration) and then automating its path to release (continuous
delivery/deployment) through a versioned pipeline.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 110" role="img" aria-label="A pipeline flowing from commit to build to test to deploy, each stage gating the next." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <rect x="14" y="42" width="84" height="34" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="56" y="63">commit</text>
    <line x1="98" y1="59" x2="130" y2="59" stroke="currentColor" stroke-width="1.1"/>
    <rect x="130" y="42" width="84" height="34" rx="5" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.3"/><text x="172" y="63">build</text>
    <line x1="214" y1="59" x2="246" y2="59" stroke="currentColor" stroke-width="1.1"/>
    <rect x="246" y="42" width="84" height="34" rx="5" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.3"/><text x="288" y="63">test</text>
    <line x1="330" y1="59" x2="362" y2="59" stroke="currentColor" stroke-width="1.1"/>
    <rect x="362" y="42" width="84" height="34" rx="5" fill="currentColor" fill-opacity="0.18" stroke="currentColor" stroke-width="1.4"/><text x="404" y="63">deploy</text>
  </g>
</svg>
<figcaption>A CI/CD pipeline carries each commit through build, test, and deploy, with every stage gating the next.</figcaption>
</figure>

## Continuous Integration

**Continuous Integration (CI)** automatically builds and runs the
[test suite](/reference/unit-testing/) on *every push* (and every pull request). A CI
server watches the repository in [version control](/reference/version-control/); when a
change lands it checks out the code, installs the pinned dependencies via the
[build system](/reference/build-systems/), compiles it, and runs the tests. If anything
fails, the team learns within minutes — while the change is fresh and small. The name
comes from *integrating* everyone's work frequently into the shared main line instead of
letting branches drift and merging in one painful "big bang." CI is the automated
enforcement of "don't break the build."

## Continuous Delivery and Deployment

CI gets you a tested build; CD carries it toward release.

- **Continuous Delivery** keeps the build always in a *releasable* state — packaged and
  staged — with a human choosing *when* to ship.
- **Continuous Deployment** removes even that gate: every change that passes the full
  pipeline ships to production automatically.

The distinction is just where the human stands. Which you want depends on risk tolerance
and how good your [tests](/reference/integration-testing/) are.

## Pipelines and artifacts

The whole automated sequence is a **pipeline** — staged steps like lint → build → test →
package → deploy, each gating the next, and typically defined in a YAML file committed to
the repo so the process itself is versioned. A successful run produces an **artifact** (a
binary, container image, or installer) that is stored, versioned with
[semantic versioning](/reference/semantic-versioning/), and promoted unchanged through
environments. On GitHub the common tool is **GitHub Actions**, which can run the pipeline,
fan out across a build matrix, run [end-to-end tests](/reference/end-to-end-testing/), and
publish release binaries through a [package manager](/reference/package-manager/) or
release page — only when everything is green.
