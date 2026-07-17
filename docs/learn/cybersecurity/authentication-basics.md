---
slug: authentication-basics
title: Authentication basics
description: A plain-language introduction to authentication — how a system proves you are who you claim to be. The three factors (something you know, have, or are), the difference between authentication and authorization, and why single factors are the weak point defenders worry about.
keywords: authentication, factors, something you know, something you have, something you are, biometric, credentials, identity, multi-factor, authentication vs authorization, verifying identity, login
level: beginner
status: full
prereq:
  - what-is-cybersecurity
faq:
  - q: What is authentication in simple terms?
    a: "Authentication is how a system proves you are who you claim to be before it lets you in. You present some evidence — a password, a phone, a fingerprint — and the system checks it against what it expects. It answers one question: are you really this person?"
  - q: What are the three authentication factors?
    a: "The three factors are something you know (a password or PIN), something you have (a phone or hardware key), and something you are (a fingerprint or face). They matter because combining two different factors is far harder to defeat than strengthening just one."
  - q: What is the difference between authentication and authorization?
    a: "Authentication decides who you are; authorization decides what you are allowed to do once you're in. Logging in with your password is authentication; being blocked from a file only admins can open is authorization. They're often confused because they usually happen back to back."
  - q: Is a fingerprint more secure than a password?
    a: A fingerprint is a different kind of factor — something you are rather than something you know — so it resists being guessed or reused across sites. But it isn't automatically stronger on its own; the real gain comes from combining two different factors rather than relying on any single one.
---

# Authentication basics

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Authentication = proving identity.** Before a system grants access it decides
you really are who you claim to be, using one of **the three factors**: something
you **know**, something you **have**, or something you **are**. Don't confuse it
with authorization — **authn is *who* you are, authz is *what* you may do**. Get
authentication wrong and everything built on top of it is standing on sand.
</div>

Every login, unlock, and sign-in is the same question in disguise: *can you prove
you're the person you say you are?* This lesson, building on
[what cybersecurity is](/learn/cybersecurity/what-is-cybersecurity/), unpacks how
systems answer that question — and where the answer tends to go wrong.

## What authentication is

**Authentication** is the process of **verifying an identity** before granting
access. You make a claim ("I'm this user") and present some evidence; the system
checks that evidence against what it already knows and decides whether to believe
you.

It's the **front door** of security. Almost every other control — permissions,
audit logs, encryption tied to an account — assumes the system already knows who
it's dealing with. If an attacker can walk through that front door wearing your
identity, most of the locks inside stop mattering. That's why authentication gets
so much attention: it's the hinge everything else hangs on.

## The three factors

Every piece of authentication evidence falls into one of three **factors** — three
fundamentally different *kinds* of proof:

- **Something you know** — a secret in your head. A password or a PIN is the
  classic example. Its weakness is that a secret can be guessed, watched, or shared.
- **Something you have** — a physical thing in your possession. Your phone
  receiving a code, or a hardware security key you plug in, proves you hold the
  right object. Its weakness is that objects can be lost or stolen.
- **Something you are** — a physical trait that's part of you. A fingerprint or a
  face scan (a **biometric**) measures something about your body. Its weakness is
  that you can't change it if it's ever copied.

The reason these are grouped as *factors* rather than just "passwords, phones and
fingerprints" is that each category fails in a different way. A single attack rarely
defeats two of them at once — which is exactly what the next section builds on.

## Authentication vs. authorization

These two words look alike and are constantly mixed up, but they answer different
questions:

- **Authentication (authn)** asks **who are you?** — proving your identity.
- **Authorization (authz)** asks **what are you allowed to do?** — deciding which
  actions and data your proven identity may reach.

Logging in with the correct password is authentication. Being told you *can't* open
a file that only administrators may see is authorization. You have to authenticate
first — the system can't decide what you're allowed to do until it knows who you
are. We come back to the second half of that story in
[authorization & access control](/learn/cybersecurity/authorization-and-access/).

## Single vs. multi-factor

Using **one** factor is *single-factor* authentication — a password on its own is
the everyday example. The trouble is that if that one secret leaks, the door swings
open.

Combining **two different factors** — say something you know *and* something you
have — is dramatically stronger, because an attacker now has to defeat two unrelated
things at once. Stealing your password no longer helps if they also need the phone
in your pocket. That idea is important enough to get its own lesson:
[passwords, MFA & passkeys](/learn/cybersecurity/passwords-and-mfa/) goes into how
to do it well.

## Where it goes wrong (defensively)

From a defender's point of view, authentication tends to fail in two recurring ways:

- **Weak or reused single factors.** A password that's easy to guess, or one reused
  across many sites, means a single leak unlocks everything. One factor is one point
  of failure.
- **Trusting the wrong signals.** A system that treats a phone number, an email
  address, or an easily-faked detail as proof of identity is trusting something that
  wasn't really secret in the first place.

Neither of these is an attack you carry out — they're the gaps a defender looks for
and closes. This is precisely why the lessons ahead push **multi-factor
authentication** ([passwords, MFA & passkeys](/learn/cybersecurity/passwords-and-mfa/))
and **storing credentials safely** so a leak doesn't hand over usable secrets
([hashing & integrity](/learn/cybersecurity/hashing-and-integrity/)).

<div class="knowledge-check" data-quiz data-correct-msg="Right — a fingerprint is a physical trait, so it's 'something you are'." markdown="0">
  <p class="knowledge-check__q">Quick check: a fingerprint scan is which authentication factor?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Something you know</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Something you have</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Something you are</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Authentication** proves *who you are* before a system grants access — the front
  door of security.
- Evidence comes in **three factors**: something you **know**, **have**, or **are**.
- Each factor fails differently, which is why combining two of them is so much
  stronger than reinforcing one.
- **Authentication** is *who you are*; **authorization** is *what you may do* — don't
  confuse them.
- Defensively, the weak points are **single, weak, or reused factors** and
  **trusting signals that were never secret**.

Next up: [passwords, MFA & passkeys](/learn/cybersecurity/passwords-and-mfa/)
