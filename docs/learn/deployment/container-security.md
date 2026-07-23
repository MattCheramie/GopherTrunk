---
slug: container-security
title: Container security
description: "Running as non-root, dropping capabilities, read-only filesystems, and scanning images — giving a container exactly the access it needs and no more."
keywords: container security, docker non-root, cap_drop, linux capabilities, read-only rootfs, image scanning, least privilege container, docker security
level: advanced
status: full
prereq:
  - container-networking-and-volumes
faq:
  - q: Why shouldn't containers run as root?
    a: "By default a container process runs as root inside the container, and a container escape or a compromised app then has root-level power over mounted volumes and, in some misconfigurations, the host. Running as an unprivileged user means a break-in is confined to what that user can touch. It's the single highest-value container hardening step and costs almost nothing."
  - q: What does cap_drop ALL do?
    a: "Linux splits root's power into fine-grained capabilities — binding low ports, changing file ownership, loading kernel modules, and so on. cap_drop ALL removes every one of them, then cap_add grants back only the specific few the app genuinely needs. GopherTrunk drops ALL and adds back only DAC_OVERRIDE, the one capability its non-root user needs to open the USB device node."
---

# Container security

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
A container is only as safe as the access you give it. Run as a **non-root user**, not
root. **Drop all Linux capabilities** and add back only the few you truly need. Make the
root filesystem **read-only** where you can. Keep **secrets out of the image**. And
**scan** images for known vulnerabilities. GopherTrunk's compose file is a working
example of every one of these.
</div>

[Networking & volumes](/learn/deployment/container-networking-and-volumes/) gave a
container reach; this lesson takes reach *away* until only what's needed remains. The
principle is **least privilege**: give a container exactly the access it needs and
nothing more — the same instinct as the [systemd hardening](/learn/deployment/production-hardening/)
on the bare-metal side, and it connects straight to
[hardening systems](/learn/cybersecurity/hardening-systems/) in the security module.

## Don't run as root

By default the process inside a container runs as **root**. If the app is compromised or
escapes its isolation, that root carries over to mounted volumes and, in bad
configurations, toward the host. The fix is one line in the Dockerfile — create an
unprivileged user and switch to it, which GopherTrunk does:

```dockerfile
RUN useradd --system --create-home --shell /usr/sbin/nologin gopher
USER gopher
```

Everything after `USER gopher` runs as an ordinary user with no special powers. A
break-in is now confined to what `gopher` can touch. This is the highest-value step for
the least effort — always do it.

## Drop capabilities to the bare minimum

Linux breaks root's power into individual **capabilities**: `NET_BIND_SERVICE` to bind
low ports, `CHOWN` to change ownership, `SYS_MODULE` to load kernel modules, and dozens
more. A container gets a default set it almost never fully uses. Drop them all, then add
back only what's required. GopherTrunk's compose file:

```yaml
cap_drop:
  - ALL
cap_add:
  - DAC_OVERRIDE     # only to let the non-root user open the USB device node
```

That's least privilege made concrete: **every** capability removed, exactly **one**
added back, with a comment saying why. If you can't name why a capability is on the
list, it shouldn't be.

## Make the filesystem read-only

An app that never needs to write to its own program files shouldn't be able to. A
**read-only root filesystem** stops an attacker from dropping a payload into the
container's filesystem:

```yaml
services:
  app:
    read_only: true
    tmpfs:
      - /tmp                     # writable scratch, if the app needs it
    volumes:
      - ./recordings:/var/lib/gophertrunk/recordings   # the one path that must persist
```

The app can still write to its explicit [volumes](/learn/deployment/container-networking-and-volumes/)
and any `tmpfs` scratch space — everything else is frozen. It's the container mirror of
systemd's `ProtectSystem=strict`.

## Keep secrets out of the image

A [secret](/learn/deployment/secrets-and-configuration/) baked into an image — an API
token, a password — is baked into *every layer forever*, readable by anyone who pulls it,
even if a later layer deletes it. Never `COPY` a secrets file or hard-code a token in a
Dockerfile. Inject secrets at **runtime** instead — an environment variable, a mounted
file, or a secret store — so they live only in the running container, not the shipped
artifact.

## Scan images and know what you ship

Even a minimal image contains packages that grow vulnerabilities over time. An **image
scanner** flags known CVEs so you patch before an attacker exploits them:

```bash
trivy image gophertrunk:latest       # or: docker scout cves gophertrunk:latest
```

Run it in [CI](/learn/deployment/ci-cd-pipelines/) and on a schedule — a base image
that's fine today picks up a CVE tomorrow. The smaller your
[optimized image](/learn/deployment/container-image-optimization/), the shorter this
report, which is why small and secure go together.

## The layers of container security

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 130" role="img" aria-label="Concentric boxes showing defense in depth: scanned image, non-root user, dropped capabilities, read-only filesystem, no secrets." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
  <rect x="10" y="10" width="440" height="110" rx="6" fill="none" stroke="currentColor" stroke-opacity="0.4"/><text x="230" y="26">scanned image (known-good packages)</text>
  <rect x="40" y="34" width="380" height="80" rx="5" fill="none" stroke="currentColor" stroke-opacity="0.6"/><text x="230" y="49">non-root USER</text>
  <rect x="80" y="56" width="300" height="56" rx="5" fill="none" stroke="currentColor" stroke-opacity="0.8"/><text x="230" y="70">cap_drop ALL + minimal cap_add</text>
  <rect x="120" y="78" width="220" height="32" rx="5" fill="none" stroke="currentColor" stroke-width="1.7"/><text x="230" y="98">read-only fs, no baked secrets</text>
  </g>
</svg>
<figcaption>Defense in depth: each layer removes power, so a break-in at one level still hits a wall at the next.</figcaption>
</figure>

<div class="knowledge-check" data-quiz data-correct-msg="Right — cap_drop ALL removes every capability, then cap_add grants back only what's needed." markdown="0">
  <p class="knowledge-check__q">Quick check: what does GopherTrunk's <code>cap_drop: ALL</code> plus <code>cap_add: DAC_OVERRIDE</code> achieve?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Grants the container every Linux capability</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Removes all capabilities, then adds back only the one needed to open the device</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Disables container networking</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- Run as a **non-root user** (`USER gopher`) — the highest-value, lowest-cost step.
- **`cap_drop: ALL`** then add back only what's needed (`DAC_OVERRIDE` for GopherTrunk).
- Use a **read-only root filesystem** with explicit writable volumes and tmpfs.
- Never bake **secrets** into an image; inject them at runtime.
- **Scan** images for CVEs in CI and on a schedule; smaller images have less to scan.

Next up: turning a plain binary into a managed service with systemd.
