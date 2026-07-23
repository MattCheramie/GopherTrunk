---
layout: learn-hub
learn_module: deployment
permalink: /learn/deployment/
title: Learn deployment — from your machine to production
description: A free, structured path through the modern deployment toolkit — build artifacts, containers and Docker, systemd, secrets, CI/CD, and cloud hosting — using GopherTrunk as the worked example.
keywords: learn deployment, docker tutorial, containers, dockerfile, docker compose, systemd service, ci cd, github actions, deploy to vps, devops for beginners
---

Writing software is only half the job. Getting it running reliably somewhere
other than your laptop — where it survives reboots, restarts after a crash, and
behaves the same on a server as it did on your machine — is the other half. That
second half has its own toolkit, and this path teaches it from the ground up.

**Who this is for.** Anyone who can build a program but has never shipped one.
No prior Docker, Linux administration, or DevOps background is assumed. If you
have followed the [Go module](/learn/programming-go/) and know that `go build`
produces a single binary, you already have the one prerequisite that matters.
Comfortable with containers but fuzzy on registries, systemd, or CI/CD? Use the
unit list to jump straight to the part you need.

**How the path works.** Five units take you from "what does deploy even mean?"
to a real application running on a real server. Unit 1 covers the foundations —
environments, configuration, and the build artifacts everything else moves
around. Unit 2 is containers and Docker: packaging an app with everything it
needs to run anywhere. Unit 3 keeps a service alive and observable in
production with systemd, logging, and secrets. Unit 4 automates the whole thing
with CI/CD and takes it to the cloud. Unit 5 is the payoff: deploying
GopherTrunk — a real software-defined-radio scanner — end to end, using its own
Dockerfile, docker-compose stack, and systemd unit. There is also a
[glossary](/learn/deployment/glossary/) you can skim any time.

Every concept is grounded in files that actually ship in the GopherTrunk repo,
so nothing here is hypothetical. Mark lessons complete as you go — your progress
is saved in your browser. New here?
**[Start with lesson 1: What is deployment?](/learn/deployment/what-is-deployment/)**
