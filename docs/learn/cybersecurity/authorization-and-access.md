---
slug: authorization-and-access
title: Authorization & access control
description: What a system lets you do once it knows who you are — the difference between authentication and authorization, least privilege, how access is modeled with ACLs and role-based access control (RBAC), default-deny, and why over-broad permissions make privilege escalation dangerous.
keywords: authorization, access control, least privilege, RBAC, ACL, permissions, privilege escalation, separation of duties, default-deny, role-based access control
level: intermediate
status: full
prereq:
  - authentication-basics
faq:
  - q: What is the difference between authentication and authorization?
    a: "Authentication proves who you are — it checks your identity. Authorization decides what you are allowed to do once that identity is known. Logging in is authentication; being permitted to read a file or press a button is authorization. A system needs both: knowing who you are is useless if it never limits what you can touch."
  - q: What is least privilege?
    a: "Least privilege means giving each user, service, or process only the access it needs to do its job — and nothing more. If that account or program is later compromised, the limited access limits the damage. It is the single most effective access-control habit, and it costs nothing but discipline."
  - q: What is role-based access control (RBAC)?
    a: "RBAC attaches permissions to roles rather than to individual people. You define roles like operator or auditor, grant each role the permissions its job needs, then assign users to roles. It scales far better than editing permissions per person and makes it easy to see who can do what."
  - q: What is default-deny?
    a: "Default-deny means the safe baseline is no access — everything is forbidden unless a rule explicitly allows it. New users, new files, and new services start with nothing and are granted access deliberately. It is safer than default-allow, where a forgotten rule quietly leaves something open."
---

# Authorization & access control

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
Once a system knows who you are, **access control** decides what you're allowed to do.
That's **authorization (what you may do)** — separate from proving your identity. The
guiding habit is **least privilege**: grant only the access a job needs. Real systems
organise that with **roles / RBAC** and start from a **default-deny** baseline, so
anything not explicitly permitted is refused. This builds directly on
[authentication basics](/learn/cybersecurity/authentication-basics/).
</div>

Authentication answers *who are you?* Authorization answers the very next question:
*and what are you allowed to do?* You can have a perfect login system and still be wide
open if, once inside, everyone can touch everything. Access control is the layer that
keeps a valid identity from becoming unlimited power.

## Authentication vs. authorization

These two words sound alike and are constantly confused, but they're different jobs:

- **Authentication** proves *who you are* — a password, a key, a token. It's the front
  door checking your identity. (That's the subject of
  [authentication basics](/learn/cybersecurity/authentication-basics/).)
- **Authorization** decides *what you may do* — which files you can read, which actions
  you can take, which settings you can change.

Both are needed. Authentication without authorization means anyone who logs in can do
anything. Authorization without authentication means the rules exist but the system has
no idea whom they apply to. In practice you always authenticate first, then every
sensitive action is checked against what that identity is authorized to do.

## Least privilege

The most important idea in this whole topic is **least privilege**: grant each user,
service, or process only the access it needs to do its job, and nothing more.

That's it. A backup script needs to *read* files, not delete them — so give it read
access only. A junior operator needs to acknowledge alarms, not reconfigure the system
— so their account can do exactly that and no more. A web service that only ever reads
from a database shouldn't hold write credentials.

Least privilege is the single most effective access-control habit because it shrinks
the damage when something goes wrong. Accounts get phished, software has bugs, mistakes
happen. When they do, the harm is bounded by whatever that account or process was
allowed to reach — and if you granted only the minimum, that's a small blast radius
instead of a catastrophe. It's a recurring theme of the
[security mindset](/learn/cybersecurity/security-mindset/) for exactly this reason.

## How access is modeled

Two common ways to describe *who can do what*:

- **Access control lists (ACLs).** Each resource — a file, a folder, a device — carries
  a list of who may use it and how. "This file: Alice can read and write, Bob can read,
  everyone else nothing." ACLs are simple and precise but get unwieldy when you have
  many users and many resources, because the permissions are scattered per-resource.
- **Role-based access control (RBAC).** Instead of naming individuals everywhere, you
  attach **permissions to roles**, then **assign roles to users**. Define an `operator`
  role, an `auditor` role, an `admin` role; grant each the permissions its job needs;
  then a new hire simply gets the right role. RBAC scales far better and makes it easy
  to answer "who can do this?" by looking at the role, not hunting through every
  resource.

Underneath either model sits one crucial default. The safe baseline is **default-deny**:
nothing is allowed unless a rule explicitly permits it. New users, new files, and new
services start with **no** access and are granted what they need deliberately. The
opposite — default-allow, where everything is open until someone remembers to lock it
down — is how forgotten permissions quietly become breaches. When in doubt, the answer
should be *no*.

## Privilege escalation (defensively)

**Privilege escalation** is when someone (or some malware, or a bug) manages to act
with *more* access than they were meant to have — a low-level account reaching
administrator powers, a limited service touching data it should never see. Understanding
it as a defender is about *preventing* it, not performing it.

This is exactly why over-broad permissions are dangerous. If every account can reach
everything, then compromising *any* account — the weakest password, the least careful
user — hands an attacker the whole system. But if accounts are scoped tightly, a
compromised foothold stays contained: the attacker has to find *another* weakness to go
further, and every extra step is a chance to detect and stop them.

**Separation of duties** reinforces this. No single person or account should be able to
carry a sensitive action end to end on their own — the person who requests a change
isn't the one who approves it; the account that writes logs can't erase them. Splitting
powers means one compromised credential, or one insider, can't quietly do damage alone.
Together, least privilege and separation of duties keep any single failure small.

## On your own machines

You've already met this exact idea if you've used Linux. It's the same principle as
[users and permissions](/learn/linux-cli/permissions/): everyday work happens as a
**regular user**, not as **root**, and you borrow elevated powers only for the moment
you actually need them — that's what
[sudo](/learn/linux-cli/sudo-and-root/) is for.

Running as root all the time is default-allow applied to your own computer: any mistake
or any malicious program you run inherits total control. Running as a normal user with
`sudo` for the rare privileged task is least privilege in daily practice — a typo or a
bad download can only damage what your user account could already touch, not the whole
machine. The same habit that protects a server protects your laptop.

<div class="knowledge-check" data-quiz data-correct-msg="Right — least privilege means only what the task needs, and nothing extra." markdown="0">
  <p class="knowledge-check__q">Quick check: applying least privilege to access control means...</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Giving trusted users full access so they're never blocked</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Granting only the permissions needed for the task — nothing extra</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Allowing everything by default and removing access after problems</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Authentication** proves who you are; **authorization** decides what you may do — you
  need both.
- **Least privilege** — only the access a job needs, nothing more — is the most
  effective access-control habit and bounds the damage when things go wrong.
- Access is modeled with **ACLs** (permissions per resource) and **RBAC** (permissions
  attached to roles, roles assigned to users).
- **Default-deny** is the safe baseline: nothing is allowed unless explicitly permitted.
- Over-broad permissions make **privilege escalation** dangerous; **separation of
  duties** keeps any single compromise contained.
- On your own machines it's the same idea: a regular user with **sudo** only when
  needed, never root all the time.

Next up: [managing secrets & keys](/learn/cybersecurity/secrets-management/)
