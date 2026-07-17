---
slug: hardening-systems
title: Hardening systems
description: What system hardening is and how to do it — shrinking the attack surface by removing unused software and services, patching known flaws, applying least privilege, setting secure defaults, and following an established baseline like the CIS Benchmarks.
keywords: system hardening, attack surface, patching, least privilege, secure configuration, disable services, baseline, CIS benchmark, secure defaults, close ports
level: intermediate
status: full
prereq:
  - defense-in-depth
faq:
  - q: What is system hardening?
    a: Hardening is the process of reducing a system's exposure to attack by removing what it doesn't need and locking down what it does. In practice that means uninstalling unused software, disabling services and ports you don't use, patching known flaws, running with least privilege, and replacing insecure defaults. Every removed or restricted thing is one less thing an attacker can target.
  - q: What is the single most valuable hardening step?
    a: For most systems it is removing or disabling what you don't need and keeping everything patched. Both are cheap, need no special tools, and directly shrink what an attacker can reach — unused software and unpatched, known flaws are behind a large share of real breaches. Least privilege and secure configuration build on that foundation.
  - q: What is a hardening baseline?
    a: A baseline is a published checklist of secure settings for a specific operating system or product, such as the CIS Benchmarks. Instead of inventing your own list of what to disable and how to configure it, you follow an established, peer-reviewed guide, apply it once, and then keep the system in that state over time.
  - q: Why is patching so important for hardening?
    a: Most breaches abuse flaws that were already known and already fixed — the patch existed, but the system was never updated. Keeping the operating system and installed software current closes those known holes before an attacker can use them, which is why regular patching is one of the highest-impact hardening steps.
---

# Hardening systems

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
Hardening is **removing what you don't need and locking down what you do**. Four
moves carry most of the value: **shrink the attack surface**, **patch**, apply
**least privilege**, and set **secure defaults**. A hardened system is one of
several layers — hardening sits inside
[defense in depth](/learn/cybersecurity/defense-in-depth/), not on its own.
</div>

Out of the box, most systems ship to be *easy*, not *safe*: extra software
installed, services running, ports open, default passwords set. Hardening is the
work of turning that convenient-but-exposed default into something an attacker
struggles to get a foothold on. It is methodical, not clever — a checklist you run
and then maintain.

## Reduce the attack surface

The **attack surface** is everything an attacker could poke at: installed
programs, running services, open network ports, user accounts, exposed features.
The bigger it is, the more chances something has a flaw. So the first move is to
make it smaller.

- **Uninstall unused software.** If it isn't needed, remove it — code that isn't
  there can't be exploited.
- **Disable services you don't use.** A database, web server, or remote-access
  daemon left running is an open door even if you never use it.
- **Close ports nothing needs.** Pair this with a
  [firewall](/learn/networking/firewalls/) so only the services you actually
  offer are reachable.

Every removed thing is one less
[vulnerability](/learn/cybersecurity/threats-vulnerabilities-risk/) to track,
patch, and worry about. The cheapest way to secure something is for it not to be
there at all.

## Patch and update

Most breaches don't use exotic, unknown flaws. They abuse **known** flaws that
were **already fixed** — the patch existed, but the system was never updated. The
window between "fix released" and "you applied it" is exactly when attackers
strike, because the fix also tells them what to attack.

So keep the operating system and all installed software **current**. On Linux this
is largely handled for you by
[package management](/learn/linux-cli/package-management/) — one command updates
the whole system. Automate it where you can, and check regularly where you can't.
Patching is unglamorous and it is one of the highest-impact things you will ever do.

## Least privilege

**Least privilege** means every user and every service runs with only the rights
it genuinely needs — and nothing more. A web server that only needs to read some
files should not run as **root** or **administrator**, because if it's ever
compromised, the attacker inherits exactly the powers it had.

- Don't run services as root/admin when a limited account will do.
- Give user accounts the minimum access for their job.
- Grant extra rights only when needed, and take them back afterward.

This is the same principle behind
[authorization and access control](/learn/cybersecurity/authorization-and-access/),
applied to the machine itself. On Linux, file
[permissions](/learn/linux-cli/permissions/) are the everyday tool for enforcing
it. When a limited account is breached, the blast radius stays small.

## Secure configuration

Defaults are chosen to get you running, not to keep you safe. Hardening replaces
the risky ones:

- **Change every default password.** Default credentials are public knowledge and
  the first thing attackers try.
- **Require strong authentication** and prefer **keys over passwords**. For remote
  access, key-based [SSH](/learn/linux-cli/ssh-and-remote/) beats a password every
  time, and [MFA](/learn/cybersecurity/passwords-and-mfa/) raises the bar further.
- **Turn on a firewall** so the network side matches the services you actually run.
- **Enable logging.** You can't notice an attack you never recorded —
  [monitoring and logs](/learn/linux-cli/monitoring-and-logs/) turn a silent
  compromise into something you can see and respond to.

## Use a baseline

You don't have to invent all of this yourself. Established **hardening guides** —
most famously the **CIS Benchmarks** — give you a peer-reviewed checklist of secure
settings for a specific operating system or product: what to disable, which options
to change, what to enable. Someone has already done the thinking; you follow it.

The workflow is simple: pick the baseline for your system, apply it once, and then
**keep it that way**. Systems drift over time as software is added and settings
change, so hardening is not a one-time event — it's a state you return the system
to and maintain.

<div class="knowledge-check" data-quiz data-correct-msg="Right — removing/disabling the unnecessary and staying patched is cheap and shrinks the attack surface the most." markdown="0">
  <p class="knowledge-check__q">Quick check: the cheapest, highest-impact hardening step is usually what?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Buying a dedicated security appliance</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Removing/disabling what you don't need and keeping everything patched</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Writing your own configuration guide from scratch</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- Hardening is **removing what you don't need and locking down what you do**.
- **Shrink the attack surface**: uninstall unused software, disable services, close
  ports.
- **Patch**: most breaches abuse known, already-fixed flaws — stay current.
- **Least privilege**: run users and services with only the rights they need, not as
  root/admin.
- **Secure defaults**: change default passwords, prefer keys and MFA, firewall on,
  logging on.
- **Use a baseline** like the CIS Benchmarks, apply it once, then keep the system there.

Next up: [secure coding](/learn/cybersecurity/secure-coding/)
