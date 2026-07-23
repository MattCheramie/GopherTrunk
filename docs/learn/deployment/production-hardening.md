---
slug: production-hardening
title: Production hardening
description: "Least privilege, firewalls, updates, and the systemd sandboxing GopherTrunk ships — locking a deployment down against the threats a real host faces."
keywords: production hardening, least privilege, firewall, ufw, automatic updates, ssh keys, systemd sandbox, protectsystem, nonewprivileges, server hardening
level: advanced
status: full
prereq:
  - container-security
faq:
  - q: What is the principle of least privilege?
    a: "Give every user, process, and service exactly the access it needs to do its job and nothing more. A web app doesn't need root; a scanner doesn't need to write outside its data directory; an admin doesn't need password SSH. When something is compromised, least privilege limits how far the damage spreads — the attacker inherits only the narrow access that component had."
  - q: Should I turn on automatic security updates?
    a: "For most servers, yes. Unattended security updates close known holes without waiting for you to log in, and known-but-unpatched vulnerabilities are how most servers get compromised. On Debian and Ubuntu, unattended-upgrades applies security patches automatically. Pair it with a reboot policy for kernel updates and you close the largest, easiest attack window."
---

# Production hardening

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
Hardening is **least privilege applied to a whole host**. Close every port you don't
serve with a **firewall**. Log in with **SSH keys**, not passwords. Turn on **automatic
security updates**. And confine each service to a **sandbox** — GopherTrunk's systemd
unit does this with `DynamicUser`, `ProtectSystem=strict`, `NoNewPrivileges`, and
tightly scoped device access. Every switch removes power an attacker could otherwise
inherit.
</div>

[Container security](/learn/deployment/container-security/) locked down one container;
this lesson locks down the **host it runs on** and any bare-metal service beside it. The
idea is the same — least privilege — scaled up to the machine. It leans on
[firewalls](/learn/networking/firewalls/), [SSH](/learn/linux-cli/ssh-and-remote/), and
[hardening systems](/learn/cybersecurity/hardening-systems/) from the other modules.

## Close everything you don't serve

Every open port is a door. A **firewall** shuts every door except the few you actually
use. On Ubuntu/Debian, `ufw` makes this a few lines:

```bash
sudo ufw default deny incoming      # deny everything inbound by default
sudo ufw default allow outgoing
sudo ufw allow 22/tcp               # SSH
sudo ufw allow 443/tcp              # HTTPS via the reverse proxy
sudo ufw enable
```

Notice GopherTrunk's API port `8080` is *not* opened: it's bound to `127.0.0.1` and
reached only through the [reverse proxy](/learn/deployment/reverse-proxies-and-tls/) on
443. If a service doesn't need to be reachable from outside, don't open its port — the
proxy and localhost handle the rest.

## Lock down SSH

SSH is the front door to the host, so harden it. Use **key authentication** and turn
password login off entirely — a key can't be guessed the way a password can:

```text
# /etc/ssh/sshd_config
PasswordAuthentication no
PermitRootLogin no
PubkeyAuthentication yes
```

No password auth means brute-force attempts against `sshd` are pointless, and
`PermitRootLogin no` forces admins through an unprivileged account that then uses `sudo`
— an audit trail and one less privileged door. See
[SSH & remote access](/learn/linux-cli/ssh-and-remote/) for the key setup.

## Patch automatically

Most compromised servers were running software with a *known* fix that nobody applied.
**Automatic security updates** close that gap:

```bash
sudo apt install unattended-upgrades
sudo dpkg-reconfigure -plow unattended-upgrades   # enable security updates
```

Security patches now land without you logging in. Pair it with a reboot window for kernel
updates and you shut the largest, cheapest attack vector — see
[package management](/learn/linux-cli/package-management/).

## Sandbox each service

The [systemd lesson](/learn/deployment/services-and-systemd/) introduced GopherTrunk's
unit; hardening is where its sandbox directives earn their place. The real unit confines
the scanner tightly:

```ini
DynamicUser=true            # run as an ephemeral, unprivileged user
ProtectSystem=strict        # the whole filesystem is read-only, except...
ReadWritePaths=/var/lib/gophertrunk
ProtectHome=true            # /home is invisible to the service
NoNewPrivileges=true        # can never gain new privileges via setuid
ProtectKernelModules=true   # can't load kernel modules
ProtectKernelTunables=true  # can't rewrite /proc/sys
RestrictRealtime=true
RestrictNamespaces=true
DeviceAllow=char-usb_device rwm   # only the USB device it actually needs
```

Read that as a list of powers *removed*: the scanner can't write outside its data
directory, can't see home directories, can't escalate privileges, can't touch the kernel,
and can reach only USB device nodes. Even if the process is fully compromised, the
attacker inherits this tiny box — the exact mirror of the container's dropped
capabilities, applied to a bare-metal service.

## Hardening is layers, not a switch

<figure class="figure" markdown="0">
<svg viewBox="0 0 470 108" role="img" aria-label="Layered defenses from outside in: firewall, SSH keys, automatic updates, service sandbox." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
  <rect x="8" y="8" width="454" height="92" rx="6" fill="none" stroke="currentColor" stroke-opacity="0.4"/><text x="235" y="24">firewall — only 22 and 443 open</text>
  <rect x="46" y="32" width="378" height="62" rx="5" fill="none" stroke="currentColor" stroke-opacity="0.6"/><text x="235" y="47">SSH keys only, no root login</text>
  <rect x="92" y="52" width="286" height="40" rx="5" fill="none" stroke="currentColor" stroke-opacity="0.8"/><text x="235" y="66">automatic security updates</text>
  <rect x="132" y="70" width="206" height="22" rx="4" fill="none" stroke="currentColor" stroke-width="1.7"/><text x="235" y="85" font-size="8">systemd-sandboxed service</text>
  </g>
</svg>
<figcaption>Each layer removes attacker options; a breach at the edge still meets a locked-down service in the middle.</figcaption>
</figure>

<div class="knowledge-check" data-quiz data-correct-msg="Right — ProtectSystem=strict makes the filesystem read-only except explicit ReadWritePaths." markdown="0">
  <p class="knowledge-check__q">Quick check: what does <code>ProtectSystem=strict</code> in GopherTrunk's unit do?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Encrypts the service's network traffic</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Makes the filesystem read-only except the explicit ReadWritePaths</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Restarts the service if it crashes</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- Hardening is **least privilege applied to the whole host**, in layers.
- A **firewall** (`ufw`) closes every port but the few you serve — GopherTrunk's 8080
  stays on localhost.
- Use **SSH keys**, disable password and root login.
- Turn on **automatic security updates** to close known holes unattended.
- **Sandbox** each service — GopherTrunk's systemd unit uses `DynamicUser`,
  `ProtectSystem=strict`, `NoNewPrivileges`, and scoped `DeviceAllow`.

Next up: automating build, test, and release with CI/CD pipelines.
