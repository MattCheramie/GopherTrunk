---
slug: infrastructure-as-code
title: Infrastructure as code
description: "Describing servers and services declaratively so environments are reproducible — the idea behind Terraform and Ansible, versioned like code."
keywords: infrastructure as code, iac, terraform, ansible, declarative infrastructure, idempotent, provisioning, configuration management, reproducible environments
level: intermediate
status: full
prereq:
  - ci-cd-pipelines
faq:
  - q: What is infrastructure as code?
    a: "Infrastructure as code (IaC) means describing your servers, networks, and services in text files that a tool reads to create or update the real thing — instead of clicking around a console or running commands by hand. The files live in version control, get reviewed in pull requests, and can rebuild an environment identically from scratch. Your infrastructure becomes reproducible and auditable, just like your application code."
  - q: What's the difference between Terraform and Ansible?
    a: "Terraform provisions infrastructure — it creates the servers, networks, and cloud resources themselves, tracking their state so it knows what to add, change, or destroy. Ansible configures machines that already exist — installing packages, writing config files, starting services. In a common workflow Terraform builds the server and Ansible sets it up; they're complementary, not competitors."
---

# Infrastructure as code

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Infrastructure as code (IaC)** describes servers and services in **text files** a tool
applies, instead of clicking a console or typing commands by hand. Good IaC is
**declarative** (you state the desired end state) and **idempotent** (applying it twice
changes nothing the second time). It lives in **version control** like any code.
**Terraform** provisions the infrastructure; **Ansible** configures it.
</div>

[CI/CD](/learn/deployment/ci-cd-pipelines/) automated building and shipping your
*artifact*. But the *server* it ships to also has to exist and be set up — and doing that
by hand is exactly the "works on my machine" problem one level up. **IaC** solves it: the
server's setup becomes a reviewed, versioned file you can apply anywhere.

## The problem with clicking around

Set a server up by hand — install packages, edit configs, open firewall ports — and you
own a machine nobody can reproduce. The steps live only in your memory and shell history.
Six months later you can't rebuild it, a teammate can't verify it, and the staging box
has quietly drifted from production. IaC replaces the memory with a file.

## Declarative and idempotent

Two ideas make IaC trustworthy:

- **Declarative** — you describe the *desired end state* ("a server with nginx installed
  and running"), not the steps to get there. The tool figures out the difference between
  now and desired and makes only the needed changes.
- **Idempotent** — applying the same file twice is safe. If the server already matches,
  the second run changes nothing. This is what lets you run IaC repeatedly and on a
  schedule without fear.

Contrast that with a shell script, which blindly re-runs every command whether or not it
was already done — installing twice, appending config lines again. Declarative +
idempotent means the file *is* the source of truth for what the server should be.

## Terraform: provision the infrastructure

**Terraform** creates the infrastructure itself — cloud servers, networks, DNS records —
and tracks their **state** so it knows what already exists. You declare resources:

```text
resource "digitalocean_droplet" "gophertrunk" {
  name   = "gophertrunk-prod"
  image  = "debian-12-x64"
  size   = "s-1vcpu-1gb"
  region = "nyc3"
}

resource "digitalocean_firewall" "gt" {
  droplet_ids = [digitalocean_droplet.gophertrunk.id]
  inbound_rule {
    protocol   = "tcp"
    port_range = "443"        # HTTPS via the reverse proxy only
  }
}
```

`terraform plan` shows exactly what it *would* change; `terraform apply` makes it so.
Change the size to `s-2vcpu-4gb`, run apply, and Terraform resizes the one droplet — it
doesn't rebuild the world.

## Ansible: configure what exists

**Ansible** takes a server that already exists and brings it to a desired configuration —
packages, files, services — over SSH, with no agent to install. You write **playbooks**:

```yaml
- hosts: gophertrunk-prod
  become: true
  tasks:
    - name: Install the GopherTrunk systemd unit
      copy:
        src: gophertrunk.service
        dest: /etc/systemd/system/gophertrunk.service

    - name: Enable and start GopherTrunk
      systemd:
        name: gophertrunk
        enabled: true
        state: started
```

Each task is idempotent: if the unit file is already in place and the service running,
the playbook reports "ok" and changes nothing. Run it against ten servers and they all
end up identical — the [systemd service](/learn/deployment/services-and-systemd/) you'd
otherwise install by hand, now reproducible.

## They divide the work

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 96" role="img" aria-label="Terraform provisions a server from nothing, then Ansible configures the running server." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
  <rect x="10" y="34" width="120" height="34" rx="4" fill="none" stroke="currentColor"/><text x="70" y="48">Terraform</text><text x="70" y="60" font-size="7.5">provisions the server</text>
  <rect x="180" y="34" width="120" height="34" rx="4" fill="none" stroke="currentColor"/><text x="240" y="48">Ansible</text><text x="240" y="60" font-size="7.5">configures the server</text>
  <rect x="350" y="34" width="100" height="34" rx="4" fill="none" stroke="currentColor" stroke-width="1.7"/><text x="400" y="48">running</text><text x="400" y="60" font-size="7.5">GopherTrunk</text>
  </g>
  <g stroke="currentColor" fill="none"><line x1="130" y1="51" x2="180" y2="51"/><line x1="300" y1="51" x2="350" y2="51"/></g>
</svg>
<figcaption>A common flow: Terraform builds the machine, Ansible sets it up, and the service runs — all from versioned files.</figcaption>
</figure>

The whole point: your infrastructure lives in [git](/learn/git/github-actions/), gets
reviewed in pull requests, and can be rebuilt identically from scratch. It's the same
reproducibility CI gave your builds, extended to the servers.

<div class="knowledge-check" data-quiz data-correct-msg="Right — idempotent means applying the same definition twice is safe and changes nothing the second time." markdown="0">
  <p class="knowledge-check__q">Quick check: what does it mean that IaC is idempotent?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">It can only be applied once ever</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Applying the same definition again is safe and makes no changes if nothing differs</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">It runs faster each time you apply it</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **IaC** describes infrastructure in **versioned text files** a tool applies — no
  hand-clicking.
- Good IaC is **declarative** (state the end state) and **idempotent** (safe to reapply).
- **Terraform** provisions infrastructure and tracks its state; **Ansible** configures
  existing machines over SSH.
- The payoff is **reproducible, reviewable, auditable** environments — CI's
  reproducibility, applied to servers.

Next up: where to actually run your deployment — cloud and VPS options.
