---
slug: build-and-ci-cd
title: Build systems, CI/CD & automation
description: How source becomes a running program — build systems, dependency management and lockfiles, continuous integration and delivery, pipelines, artifacts, reproducible builds and cross-compilation.
keywords: build systems, CI/CD, continuous integration, continuous delivery, dependency management, lockfiles, pipelines, build artifacts, reproducible builds, cross-compilation
level: intermediate
status: full
faq:
  - q: "What's the difference between continuous delivery and continuous deployment?"
    a: "Both keep your software always in a releasable state by automating the pipeline up to production. The difference is the final step: continuous delivery produces a deployable build but a human decides when to push it live, while continuous deployment goes all the way automatically — every change that passes the pipeline ships to production with no manual gate. Deployment is delivery with the last button pressed for you."
  - q: "Why do lockfiles matter?"
    a: "A lockfile records the exact versions (and often checksums) of every dependency your project resolved to, including indirect ones. Without it, two people — or your CI server — might install slightly different versions and get different behavior, the classic \"works on my machine\" problem. The lockfile makes dependency resolution reproducible, so everyone builds against identical inputs."
  - q: "What is cross-compilation?"
    a: "Cross-compilation is building a binary for a platform other than the one you're compiling on — for example, producing a Windows and an ARM Linux executable from your x86 Mac or Linux machine. Languages like Go make this remarkably easy, letting one toolchain emit binaries for many operating systems and architectures, which is how a single project ships installers for everyone without needing one machine per target."
---
# Build systems, CI/CD & automation

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Build systems** — turn source into a runnable program and manage dependencies reproducibly. **CI/CD** — build and test on every push, then automate the path to release. **Reproducible & cross-platform** — pin inputs, produce identical artifacts, and emit binaries for many OSes from one toolchain.
</div>

Source code isn't a program — it's instructions for *producing* one. The machinery that compiles, links, tests, packages, and ships your code is some of the highest-leverage automation in software. Done well, a developer pushes a commit and, minutes later, a tested, reproducible artifact exists, ready to deploy. This lesson walks from the build itself through dependency management to **CI/CD** — the pipelines that build and test automatically on every change and carry it toward release — and ends on reproducible builds and cross-compilation.

## Build systems

A **build** transforms your source into something runnable. The exact steps depend on the language, but the idea is universal.

- **Compilers** translate source into machine code or bytecode. A C compiler turns `.c` files into object files, then a **linker** stitches them into an executable.
- **Make** and its descendants are classic build orchestrators: a `Makefile` declares targets and their dependencies, and `make` rebuilds only what changed. The principle — describe outputs in terms of inputs, rebuild only the stale parts — underlies nearly every build tool since.
- **Language build tools** wrap this up per ecosystem: `go build` compiles a Go program, `cargo build` does it for Rust, `npm`/`yarn` build and bundle JavaScript projects, Maven and Gradle handle Java. Each knows its language's conventions so you rarely hand-write build steps.

The common thread: a build is a *function* from source (plus dependencies) to an artifact. The more deterministic that function, the more you can trust and automate it.

## Dependency management and lockfiles

Almost no project is self-contained — you stand on libraries written by others. A **dependency manager** (Go modules, Cargo, npm, pip, and so on) records what your project needs and fetches it. But there's a subtlety: a requirement like "version 1.x" can resolve to different exact versions over time, and that's a recipe for "works on my machine."

The fix is a **lockfile** (`go.sum`, `Cargo.lock`, `package-lock.json`, `poetry.lock`). It pins the *exact* version — often with a checksum — of every dependency, including the indirect ones pulled in transitively. Commit the lockfile, and everyone (and every CI run) installs byte-for-byte the same dependency tree. That's the foundation of reproducibility: without pinned inputs, you can't have a reproducible output.

| Language | Build tool | Lockfile |
| --- | --- | --- |
| Go | `go build` / modules | `go.sum` |
| Rust | Cargo | `Cargo.lock` |
| JavaScript | npm / yarn | `package-lock.json` / `yarn.lock` |
| Python | pip / Poetry | `requirements.txt` / `poetry.lock` |

## Continuous Integration

**Continuous Integration (CI)** is the practice of automatically building and testing your code on *every push* (and every pull request). A CI server watches the repository; when a change lands, it checks out the code, installs the pinned dependencies, builds it, and runs the [test suite](/learn/intro-software-dev/testing/). If anything fails, the team learns within minutes — while the change is fresh and small — rather than discovering a week-old breakage tangled with ten other changes.

The name comes from *integrating* everyone's work frequently into the shared main line, instead of letting branches drift for weeks and merging in one painful "big bang." Frequent integration plus automated checks keeps `main` continuously in a known-good state. CI is the automated enforcement of "don't break the build," and it's what makes the pull-request workflow from the [version control lesson](/learn/intro-software-dev/version-control-collab/) trustworthy: reviewers see a green check confirming the change builds and passes tests before they even read it.

<div class="knowledge-check" data-quiz data-correct-msg="Right — CI automatically builds and runs the tests on every push, catching breakage within minutes." markdown="0">
  <p class="knowledge-check__q">Quick check: what does Continuous Integration do?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Lets each developer keep their branch unmerged for months</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Automatically builds and runs the tests on every push</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Manually deploys code to production once a quarter</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Continuous Delivery and Deployment

CI gets you a tested build. **Continuous Delivery (CD)** extends the automation so that build is always in a *releasable* state — packaged, staged, and ready to ship at the press of a button, with a human choosing *when*. **Continuous Deployment** removes even that button: every change that passes the full pipeline is shipped to production automatically, no manual gate. The distinction is just where the human stands:

- **Continuous Delivery** — always ready to release; a person clicks "go."
- **Continuous Deployment** — passing the pipeline *is* the release.

Which you want depends on risk tolerance and how good your tests are. A high-stakes system might keep a human gate; a web app with strong test coverage and easy rollback might deploy dozens of times a day.

## Pipelines and artifacts

The whole automated sequence is a **pipeline** — a series of stages that a change flows through, each gating the next: lint → build → test → package → deploy. If any stage fails, the pipeline stops and the change doesn't advance. Pipelines are typically defined in code (a YAML file in the repo) so the process itself is versioned and reviewable.

The output of a successful build is an **artifact** — a compiled binary, a container image, a packaged installer. Artifacts are stored and versioned so you can deploy a known build, and roll back to a previous one if something goes wrong. The pipeline produces artifacts once and promotes the *same* artifact through environments, rather than rebuilding at each stage and risking inconsistency.

On GitHub, the common tool for all of this is **[GitHub Actions](/learn/git/github-actions/)**, which runs your pipeline on every push and can build artifacts and publish releases. For a project like GopherTrunk, a single Actions workflow might build the decoder, run the [golden-file tests](/learn/intro-software-dev/testing/) against captured signals, and — only if everything's green — publish release binaries.

## Reproducible builds and cross-compilation

Two qualities elevate a build from "it ran" to "I trust it."

**Reproducible builds** mean that the same source plus the same pinned dependencies always produces a bit-for-bit identical artifact, regardless of who builds it or when. This is powerful: it lets others independently verify that a published binary really came from the claimed source (a security property), and it eliminates a whole class of "the build machine was different" mysteries. Lockfiles, pinned tool versions, and controlled build environments are what make it possible.

**Cross-compilation** is producing binaries for platforms *other than* the one you're building on — a Windows `.exe` and an ARM Linux binary emitted from your x86 development machine. Go is famous for making this trivial: set `GOOS` and `GOARCH` and `go build` produces a binary for that target.

```bash
GOOS=windows GOARCH=amd64 go build -o gophertrunk.exe   # Windows from Linux
GOOS=darwin  GOARCH=arm64 go build -o gophertrunk-mac    # Apple Silicon
```

One toolchain, many targets. A CI pipeline can fan out across a build matrix and emit binaries for every supported OS and architecture in a single run — which is exactly how a project ships installers for Windows, macOS, and Linux without needing a separate machine for each. That hand-off, from "we have binaries for everyone" to "users can actually install it," is the subject of [packaging & distribution](/learn/intro-software-dev/packaging-and-distribution/).

## Recap

- **Build systems** — compilers, Make, and language tools (`go build`, Cargo, npm) turn source plus dependencies into a runnable artifact.
- **Lockfiles** — pin exact dependency versions so every build uses identical inputs; the basis of reproducibility.
- **CI** — automatically builds and tests on every push, catching breakage in minutes and keeping `main` known-good.
- **CD** — Continuous Delivery keeps a build always releasable; Continuous Deployment ships it automatically; the difference is the human gate.
- **Pipelines & artifacts** — staged, versioned-in-code automation that produces stored, deployable, rollback-able artifacts.
- **Reproducible & cross-compiled** — identical outputs from identical inputs, and one toolchain emitting binaries for many platforms.

Next up: the work that outlasts the build — [documentation, tech debt & maintenance](/learn/intro-software-dev/docs-and-maintenance/).
