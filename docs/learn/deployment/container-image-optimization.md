---
slug: container-image-optimization
title: Optimizing container images
description: "Small, fast, secure images — multi-stage builds, minimal and distroless bases, layer-cache ordering, and .dockerignore — and why GopherTrunk's image is tiny."
keywords: docker image optimization, multi-stage build, distroless, scratch image, slim base image, dockerignore, layer caching, image scanning, small docker image
level: intermediate
status: full
prereq:
  - writing-a-dockerfile
faq:
  - q: Why should a container image be small?
    a: "A small image builds and pushes faster, pulls faster onto every host you deploy to, and — most importantly — carries less software that could contain vulnerabilities. Every package in the image is attack surface and something to keep patched. Shipping only your binary and its bare runtime dependencies means fewer things can break or be exploited."
  - q: What is a distroless or scratch image?
    a: "scratch is the empty base image — nothing at all — suitable for a fully static binary that needs no operating system files. distroless images (from Google) add just the bare runtime essentials like CA certificates and timezone data but no shell or package manager. Both produce tiny images with almost no attack surface, at the cost of not being able to shell into the container to debug."
---

# Optimizing container images

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
A good image is **small, fast, and secure**. A **multi-stage build** leaves the compiler
behind and ships only the binary. A **minimal base** — slim, distroless, or scratch —
strips the OS down to essentials. **Ordering layers** so stable steps come first keeps
the build cache warm. A **`.dockerignore`** keeps junk out of the build. Smaller means
less to pull, less to patch, and less to attack.
</div>

[Writing a Dockerfile](/learn/deployment/writing-a-dockerfile/) built a working image.
This lesson makes it *good* — because a bloated image is slow to ship and full of
software you didn't mean to run. GopherTrunk's image is deliberately tiny; here's how,
and how to do the same to yours.

## Multi-stage: leave the toolchain behind

The single biggest win is a **multi-stage build**. One stage has the full Go toolchain
and compiles the binary; a second, clean stage copies out *only* the finished binary.
The Go compiler, the source, the module cache — none of it ships. GopherTrunk does
exactly this:

```dockerfile
FROM golang:1.25-bookworm AS builder
# ... compile the binary ...
RUN go build -trimpath -ldflags "-s -w ..." -o /out/gophertrunk ./cmd/gophertrunk

FROM debian:bookworm-slim AS runtime
COPY --from=builder /out/gophertrunk /usr/local/bin/gophertrunk
```

The `golang:1.25-bookworm` builder is close to a gigabyte; the `debian:bookworm-slim`
runtime plus the static binary is a small fraction of that. Note `-ldflags "-s -w"`,
which strips debug symbols from the binary itself — a smaller binary in a smaller image.

## Choose the smallest base you can run on

The runtime base is the floor under your image size and its attack surface. The options,
smallest first:

| Base | Contains | Trade-off |
|------|----------|-----------|
| `scratch` | Nothing at all | Only works for a fully static binary; no shell to debug |
| distroless | CA certs, tzdata, no shell/package manager | Tiny and locked down; harder to poke at |
| `debian:bookworm-slim` | Minimal Debian + apt | Can shell in and add a package; a bit bigger |
| `debian:bookworm` / `ubuntu` | Full OS | Large; lots of software you don't run |

Because GopherTrunk is built `CGO_ENABLED=0` — a **pure-Go static binary** — it *could*
run on `scratch`. It uses `debian:bookworm-slim` and adds only `ca-certificates` so an
operator can still shell in for USB debugging, a deliberate small trade of size for
practicality.

## Order layers so the cache stays warm

Each Dockerfile instruction is a cached [layer](/learn/deployment/writing-a-dockerfile/).
Docker reuses a layer if nothing above it changed — so put the slow, rarely-changing
steps *first*. GopherTrunk copies `go.mod`/`go.sum` and downloads dependencies **before**
copying the source:

```dockerfile
COPY go.mod go.sum ./
RUN go mod download        # cached until dependencies change
COPY . .                   # changes on every code edit
```

Edit a `.go` file and only the layers below `COPY . .` rebuild; the dependency download
is reused. Reverse the order and every code change re-downloads every dependency.

## Keep junk out with .dockerignore

`COPY . .` copies everything in the build context — including `.git`, local build output,
and secrets — unless you exclude it. A `.dockerignore` file, like `.gitignore`, keeps
that out:

```text
.git
*.md
bin/
recordings/
*.cfile
.env
```

This shrinks the build context (faster builds), avoids busting the cache on irrelevant
file changes, and — critically — stops a stray `.env` from being baked into the image.

## Scan what you ship

Small images have less to scan, but you should still scan. An **image scanner** compares
the packages in your image against known-vulnerability databases:

```bash
docker scout cves gophertrunk:latest      # or: trivy image gophertrunk:latest
```

Run it in [CI](/learn/deployment/ci-cd-pipelines/) so a base image that picked up a new
CVE fails the build before it reaches a server. The less you put in the image, the
shorter this report — which is the whole point of optimizing: **small is secure**.

<div class="knowledge-check" data-quiz data-correct-msg="Right — a multi-stage build ships only the binary, leaving the compiler and source behind." markdown="0">
  <p class="knowledge-check__q">Quick check: what is the single biggest reason a Go container image can be tiny?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Docker compresses every image automatically</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">A multi-stage build copies only the compiled binary into a minimal base</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Go images are always small regardless of how they're built</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- A **multi-stage build** ships only the binary, not the compiler or source.
- Pick the **smallest base** you can run on — scratch, distroless, or slim.
- **Order layers** so stable steps (dependency downloads) come before code, keeping the
  cache warm.
- A **`.dockerignore`** keeps junk and secrets out of the build context.
- **Scan** images in CI; **smaller means less to patch and less to attack**.

Next up: naming, storing, and sharing images through registries.
