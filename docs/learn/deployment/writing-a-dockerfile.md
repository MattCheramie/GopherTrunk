---
slug: writing-a-dockerfile
title: Writing a Dockerfile
description: Build a container image step by step — base images, layers, and multi-stage builds — using GopherTrunk's real Dockerfile that turns Go source into a small runtime image.
keywords: dockerfile, docker build, base image, image layers, multi-stage build, docker from, dockerfile example, go docker
level: intermediate
status: full
prereq:
  - what-is-a-container
faq:
  - q: What is a multi-stage Docker build?
    a: A multi-stage build uses one stage to compile your software (with all the heavy build tools) and a second, clean stage that copies out only the finished binary. The final image contains just what's needed to run — not the compiler or source — so it's much smaller and has less to attack or maintain.
  - q: What are image layers?
    a: Each instruction in a Dockerfile creates a layer — a saved filesystem change stacked on the previous one. Docker caches layers, so if nothing above a layer changed, it's reused instead of rebuilt. Ordering your Dockerfile so rarely-changing steps come first (like downloading dependencies) makes rebuilds much faster.
---

# Writing a Dockerfile

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
A **Dockerfile** is the recipe for a container image: a **base image** to start from,
then instructions that each add a cached **layer**. A **multi-stage build** compiles in
one stage and copies only the finished binary into a small final image. GopherTrunk's
real Dockerfile does exactly this — Go source in, a slim runtime image out.
</div>

An image doesn't appear by magic — you write a **Dockerfile** describing how to build
it. This lesson walks through one, using GopherTrunk's actual file.

## The idea: a recipe of layers

A Dockerfile is a list of instructions, top to bottom. Each one — `FROM`, `COPY`, `RUN`
— produces a **layer**, a saved filesystem change stacked on the one before. Docker
caches layers, so an unchanged step is reused on the next build. That's why you put the
slowest, least-changing steps (like downloading dependencies) *early*: change your code
and only the layers below the copy rebuild.

## GopherTrunk's Dockerfile, stage one

GopherTrunk uses a **multi-stage** build. The first stage compiles the Go binary:

```dockerfile
FROM golang:1.25-bookworm AS builder
WORKDIR /src

# Cache deps before copying the rest of the source.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG VERSION=docker
ENV CGO_ENABLED=0
RUN go build -trimpath \
        -ldflags "-s -w -X github.com/MattCheramie/GopherTrunk/internal/version.Version=${VERSION}" \
        -o /out/gophertrunk ./cmd/gophertrunk
```

Notice the deliberate ordering: `go.mod`/`go.sum` are copied and dependencies
downloaded *before* the full source, so editing code doesn't invalidate the cached
dependency layer. `CGO_ENABLED=0` produces a [pure-Go static binary](/learn/programming-go/hello-go/),
and the `-ldflags -X` stamps in the [version](/learn/deployment/build-artifacts-and-versioning/).

## Stage two: a small runtime image

The second stage starts fresh from a *slim* base and copies in only the finished
binary — none of the Go compiler or source comes along:

```dockerfile
FROM debian:bookworm-slim AS runtime
RUN apt-get update \
 && apt-get install -y --no-install-recommends ca-certificates \
 && rm -rf /var/lib/apt/lists/*

# Non-root user for safety.
RUN useradd --system --create-home --shell /usr/sbin/nologin gopher
USER gopher
WORKDIR /home/gopher

COPY --from=builder /out/gophertrunk /usr/local/bin/gophertrunk

EXPOSE 8080 50051
ENTRYPOINT ["/usr/local/bin/gophertrunk"]
CMD ["run", "-config", "/etc/gophertrunk/config.yaml"]
```

Three good practices are visible here:

- **`COPY --from=builder`** pulls just the binary out of the build stage — the final
  image is tiny and contains no toolchain to attack.
- **`USER gopher`** runs the app as a non-root user, a security default worth copying
  (see [secure deployment](/learn/deployment/secrets-and-configuration/)).
- **`EXPOSE`**, **`ENTRYPOINT`**, and **`CMD`** document the ports and set what runs
  when the container starts.

## Building it

With the Dockerfile in place, one command builds the image:

```bash
docker build -t gophertrunk:dev .
```

Docker runs each instruction, caching layers as it goes, and tags the result
`gophertrunk:dev`. That image is now a portable artifact you can run anywhere Docker
runs — and share through a [registry](/learn/deployment/images-and-registries/), the
next lesson.

<div class="knowledge-check" data-quiz data-correct-msg="Right — a multi-stage build keeps only the finished binary, leaving the toolchain behind." markdown="0">
  <p class="knowledge-check__q">Quick check: why does GopherTrunk's Dockerfile use two stages?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">To build the image twice for reliability</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">To compile in one stage but ship only the small binary in the final image</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Because Go requires two Dockerfiles</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- A **Dockerfile** is a recipe of instructions, each producing a cached **layer**.
- Order steps so **slow, stable ones come first** (dependencies before source) for fast
  rebuilds.
- A **multi-stage build** compiles in a builder stage and copies only the **binary**
  into a slim final image.
- Run apps as a **non-root user**; `docker build -t name .` produces the image.

Next up: naming, storing, and sharing images through registries.
