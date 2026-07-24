---
slug: glossary
title: Glossary of deployment terms
description: Plain-language definitions of deployment terms — container, image, registry, Dockerfile, Compose, systemd, CI/CD, artifact, secret, and more — each cross-linked to the lesson where it's explained.
keywords: deployment glossary, docker terms, devops glossary, container image registry definition, ci cd systemd glossary, deployment dictionary
level: beginner
status: full
lesson_standalone: true
---

# Glossary of deployment terms

Every term used across the [deployment module](/learn/deployment/), defined in plain
language and linked to the lesson where it's explained in full. Skim it as a refresher,
or use your browser's find (Ctrl/Cmd-F) to jump to a word. Terms are grouped by theme.

## Foundations

**Deployment** — Getting software running reliably somewhere other than your own
machine. See [What is deployment?](/learn/deployment/what-is-deployment/)

**Environment** — A place software runs — development, staging, or production — each the
same build with different configuration. See [Environments & configuration](/learn/deployment/environments-and-config/)

**Configuration** — Settings that change how software behaves (ports, paths, log
levels), supplied at startup rather than baked in. See [Environments & configuration](/learn/deployment/environments-and-config/)

**Environment variable** — A key/value setting the operating system passes to a program,
a common way to inject configuration. See [Environments & configuration](/learn/deployment/environments-and-config/)

**Build artifact** — The concrete deployable output of a build — a binary, image, or
package. See [Build artifacts & versioning](/learn/deployment/build-artifacts-and-versioning/)

**Immutable artifact** — An artifact built once and never changed, deployed identically
everywhere. See [Build artifacts & versioning](/learn/deployment/build-artifacts-and-versioning/)

**Semantic versioning (SemVer)** — Labelling releases MAJOR.MINOR.PATCH to signal how
risky an upgrade is. See [Build artifacts & versioning](/learn/deployment/build-artifacts-and-versioning/)

**Rollback** — Returning to a previous known-good version by redeploying its artifact.
See [Monitoring & updates](/learn/deployment/monitoring-and-updates/)

## Containers

**Container** — An application bundled with its dependencies, running isolated on a
shared host kernel. See [What is a container?](/learn/deployment/what-is-a-container/)

**Virtual machine (VM)** — An emulated computer running a full guest OS — stronger
isolation than a container but much heavier. See [What is a container?](/learn/deployment/what-is-a-container/)

**Image** — The packaged, read-only template a container runs from. See [What is a container?](/learn/deployment/what-is-a-container/)

**Namespaces / cgroups** — Kernel features that isolate a container's view of the system
and cap its resources. See [What is a container?](/learn/deployment/what-is-a-container/)

**Dockerfile** — The recipe of instructions that builds a container image. See [Writing a Dockerfile](/learn/deployment/writing-a-dockerfile/)

**Base image** — The starting image a Dockerfile builds on top of. See [Writing a Dockerfile](/learn/deployment/writing-a-dockerfile/)

**Layer** — A cached filesystem change produced by one Dockerfile instruction. See [Writing a Dockerfile](/learn/deployment/writing-a-dockerfile/)

**Multi-stage build** — A Dockerfile that compiles in one stage and copies only the
finished binary into a small final image. See [Writing a Dockerfile](/learn/deployment/writing-a-dockerfile/)

**Registry** — A server that stores and distributes images (Docker Hub, ghcr.io). See
[Images & registries](/learn/deployment/images-and-registries/)

**Tag** — A movable label on an image, like `:latest` or `:1.4.2`. See [Images & registries](/learn/deployment/images-and-registries/)

**Digest** — An immutable hash identifying an image's exact contents. See [Images & registries](/learn/deployment/images-and-registries/)

**Docker Compose** — A tool that describes services, networks, and volumes in one file
and runs them together. See [Multi-container apps with Compose](/learn/deployment/docker-compose/)

**Service (Compose)** — One container definition within a Compose file. See [Multi-container apps with Compose](/learn/deployment/docker-compose/)

**Volume** — A host directory mapped into a container so data persists across restarts.
See [Multi-container apps with Compose](/learn/deployment/docker-compose/)

## Running in production

**Service (systemd)** — A background program managed by systemd — started on boot,
restarted on failure. See [Services & systemd](/learn/deployment/services-and-systemd/)

**systemd** — The Linux init and service manager that runs and supervises services. See
[Services & systemd](/learn/deployment/services-and-systemd/)

**Unit file** — The file describing a systemd service (`ExecStart`, `Restart`, etc.).
See [Services & systemd](/learn/deployment/services-and-systemd/)

**Hardening** — Restricting a service to the least privilege it needs, via systemd
sandboxing or dropped container capabilities. See [Services & systemd](/learn/deployment/services-and-systemd/)

**Log** — A recorded narrative of a service's events, read via `journalctl` or `docker
logs`. See [Logging & health checks](/learn/deployment/logging-and-health-checks/)

**Structured logging** — Recording log events as machine-readable fields, making them
searchable. See [Logging & health checks](/learn/deployment/logging-and-health-checks/)

**Health check** — An endpoint a monitor polls to learn whether a service is working.
See [Logging & health checks](/learn/deployment/logging-and-health-checks/)

**Liveness / readiness** — Whether a service is alive (restart if not) versus ready to
serve (hold traffic if not). See [Logging & health checks](/learn/deployment/logging-and-health-checks/)

**Secret** — Confidential configuration (API key, password, token) that must never be
committed or baked into an image. See [Secrets & configuration management](/learn/deployment/secrets-and-configuration/)

**Secret store** — A dedicated service that holds secrets and hands them to authorized
services at runtime. See [Secrets & configuration management](/learn/deployment/secrets-and-configuration/)

## Automate & scale

**Continuous integration (CI)** — Automatically building and testing every change as
it's pushed. See [CI/CD pipelines](/learn/deployment/ci-cd-pipelines/)

**Continuous delivery (CD)** — Automatically packaging and releasing changes that pass
CI. See [CI/CD pipelines](/learn/deployment/ci-cd-pipelines/)

**Pipeline** — An automated sequence of build/test/release steps triggered by an event.
See [CI/CD pipelines](/learn/deployment/ci-cd-pipelines/)

**VPS (virtual private server)** — A rented Linux virtual machine you fully control. See
[Deploying to the cloud & VPS](/learn/deployment/deploying-to-cloud-and-vps/)

**PaaS (managed platform)** — A host that runs your app and handles the servers for you.
See [Deploying to the cloud & VPS](/learn/deployment/deploying-to-cloud-and-vps/)

**Metrics** — Numbers measured over time (request rate, errors, memory) that reveal
trends. See [Monitoring & updates](/learn/deployment/monitoring-and-updates/)

**Alert** — An automatic notification when a metric crosses a threshold. See [Monitoring & updates](/learn/deployment/monitoring-and-updates/)

**Rolling update** — Deploying a new version while watching its health, ready to roll
back. See [Monitoring & updates](/learn/deployment/monitoring-and-updates/)

## Lifecycle & releases

**Deployment lifecycle** — The repeating loop every change moves through: build, test,
release, deploy, operate. See [The deployment lifecycle](/learn/deployment/the-deployment-lifecycle/)

**Rolling / blue-green / canary** — Release strategies that push a new version gradually,
by switching between two environments, or to a small slice of users first. See [Release strategies](/learn/deployment/release-strategies/)

## More on containers

**Distroless / scratch base** — Minimal container base images with little or no OS
userland, shrinking size and attack surface. See [Optimizing container images](/learn/deployment/container-image-optimization/)

**.dockerignore** — A file listing paths to exclude from the build context, keeping images
small and builds fast. See [Optimizing container images](/learn/deployment/container-image-optimization/)

**Bridge network / port publishing** — How Docker gives containers an internal network and
maps their ports to the host. See [Container networking & volumes](/learn/deployment/container-networking-and-volumes/)

**Capability (Linux)** — A fine-grained privilege; containers drop all and add back only
what they need. See [Container security](/learn/deployment/container-security/)

**Image scanning** — Automatically checking an image for known vulnerabilities before
deploying it. See [Container security](/learn/deployment/container-security/)

## Networking, data & hardening

**Reverse proxy** — A server (nginx, Caddy) in front of an app that terminates TLS and
routes requests to it. See [Reverse proxies & TLS](/learn/deployment/reverse-proxies-and-tls/)

**TLS termination** — Handling HTTPS encryption at the proxy so the app behind it can
speak plain HTTP. See [Reverse proxies & TLS](/learn/deployment/reverse-proxies-and-tls/)

**Stateful vs stateless** — Whether a service keeps important data locally; stateful data
needs backups and a persistence plan. See [Backups & persistent data](/learn/deployment/backups-and-data/)

**Least privilege** — Granting a service only the access it needs — the core idea behind
production hardening. See [Production hardening](/learn/deployment/production-hardening/)

## Automate & scale

**Infrastructure as code (IaC)** — Describing servers and services in versioned,
declarative files (Terraform, Ansible). See [Infrastructure as code](/learn/deployment/infrastructure-as-code/)

**Orchestrator** — A system (like Kubernetes) that schedules, heals, and scales many
containers across machines. See [Container orchestration](/learn/deployment/container-orchestration/)

**Horizontal vs vertical scaling** — Adding more instances vs making one instance bigger;
a load balancer spreads traffic across instances. See [Scaling & load balancing](/learn/deployment/scaling-and-load-balancing/)

**Load balancer** — A component that distributes incoming requests across several service
instances. See [Scaling & load balancing](/learn/deployment/scaling-and-load-balancing/)

**Connection draining** — Letting in-flight requests finish before removing an old
instance during a deploy. See [Zero-downtime deploys](/learn/deployment/zero-downtime-deploys/)

## Operate

**Observability** — Understanding a system from its outputs, via the three pillars of
metrics, logs, and traces. See [Observability — metrics, logs & traces](/learn/deployment/observability/)

**Trace** — A record following one request across services, showing where time is spent.
See [Observability — metrics, logs & traces](/learn/deployment/observability/)

**Alert / on-call** — An automatic notification when a metric crosses a threshold, and the
rotation of who responds. See [Alerting & on-call](/learn/deployment/alerting-and-oncall/)

**Runbook / postmortem** — A documented response procedure, and the blameless review that
follows an incident. See [Incident response & runbooks](/learn/deployment/incident-response/)

**Right-sizing** — Matching allocated CPU and memory to what a service actually needs, to
control cost. See [Cost & resource management](/learn/deployment/cost-and-resources/)
