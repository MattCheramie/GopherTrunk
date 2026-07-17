---
slug: cia-triad
title: The CIA triad
description: The three goals that define what "secure" means — confidentiality, integrity, and availability — explained in plain language, how each is enforced, why they trade against each other, and how the triad helps you classify what any attack is really targeting.
keywords: CIA triad, confidentiality, integrity, availability, security goals, information security, data protection, denial of service, encryption, access control, security fundamentals
level: beginner
status: full
prereq:
  - what-is-cybersecurity
faq:
  - q: What is the CIA triad in cybersecurity?
    a: "The CIA triad is a model of the three core goals of security: confidentiality (only authorized people can see the information), integrity (the information isn't altered without authorization and you can tell if it was), and availability (the system and data are there when legitimate users need them). Together they define what \"secure\" means, and almost every security control serves one or more of the three."
  - q: What does each letter of the CIA triad stand for?
    a: "C is confidentiality — keeping information secret from those who shouldn't see it, enforced by encryption and access control. I is integrity — making sure data isn't changed without permission, and that tampering is detectable, enforced by hashing and digital signatures. A is availability — keeping systems and data reachable for the people who are supposed to use them, protected by redundancy and backups."
  - q: Do the three goals of the CIA triad ever conflict?
    a: "Yes, and constantly. Locking data down tightly for confidentiality can make it harder for legitimate users to reach it, hurting availability. Adding heavy integrity checks or extra authentication steps can slow a system down. Every real security decision is a trade-off among the three, and the right balance depends on what you're protecting and from whom."
  - q: How does the CIA triad help classify attacks?
    a: "It gives you a quick way to ask what an attack is really after. Eavesdropping on traffic targets confidentiality. Tampering with data or code targets integrity. A flood of traffic or ransomware that locks up your files targets availability. Naming the goal under attack points you straight at the defenses that matter."
---

# The CIA triad

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
Security has three core goals: **confidentiality** (only the right people can
see it), **integrity** (nobody changed it without permission, and you'd know if
they did), and **availability** (it's there when you need it). Together these
three define what "secure" even means. Nearly every control you'll ever meet
exists to protect one or more of them — and, as
[cybersecurity]({{ '/learn/cybersecurity/what-is-cybersecurity/' | relative_url }})
teaches, the defender's job is to keep all three standing at once.
</div>

When someone says a system is "secure," they're making a claim about three
different things at the same time. Pull them apart and security stops being a
vague feeling and becomes a checklist you can actually reason about. That's the
whole value of the **CIA triad**: three plain goals — confidentiality,
integrity, availability — that give you the vocabulary to say exactly what you're
protecting and what an attacker is trying to take away.

## Confidentiality

Confidentiality means **only authorized people can see the information**. Your
medical records, a company's plans, the password you typed this morning — all of
it should be readable by the right people and no one else.

Two families of control enforce it. **Encryption** scrambles data so that
anyone who intercepts it sees only noise without the key — the subject of
[what is cryptography?]({{ '/learn/cybersecurity/what-is-cryptography/' | relative_url }}).
**Access control** decides who is allowed to reach the data in the first place,
so that even people on the system only see what their role permits — covered in
[authorization & access]({{ '/learn/cybersecurity/authorization-and-access/' | relative_url }}).

When confidentiality fails, secrets leak: someone reads what they shouldn't.

## Integrity

Integrity means **the information isn't altered without authorization — and if
it is, you can tell**. It's not enough that data stays secret; you also need
confidence that it's the *real* data. A bank balance quietly changed, a software
update swapped for a tampered one, a log edited to hide a break-in — those are
all integrity failures.

The defenses here are about detection as much as prevention. **Hashing**
produces a short fingerprint of a file that changes completely if even one byte
changes, so tampering is obvious — see
[hashing & integrity]({{ '/learn/cybersecurity/hashing-and-integrity/' | relative_url }}).
**Digital signatures** go further, proving both that the data wasn't changed
*and* who vouched for it, which is how your device trusts a software update or a
website — see
[signatures & certificates]({{ '/learn/cybersecurity/signatures-and-certificates/' | relative_url }}).

When integrity fails, you can no longer trust that what you're looking at is
genuine.

## Availability

Availability means **the system and its data are there when legitimate users
need them**. A perfectly confidential, perfectly intact database is useless if
nobody can reach it. Availability is the goal people forget until it's gone.

It's threatened by ordinary outages — a failed disk, a cut cable, a botched
deploy — and by deliberate **denial-of-service** attacks that flood a system
with traffic until it collapses, one of the patterns in
[network attacks]({{ '/learn/cybersecurity/network-attacks/' | relative_url }}).
The defenses are about resilience: **redundancy** (spare capacity and backup
paths so no single failure takes you down) and **backups** (so you can restore
data that's lost or held hostage). Spotting an availability problem early is a
job for
[monitoring & incident response]({{ '/learn/cybersecurity/monitoring-and-incident-response/' | relative_url }}).

When availability fails, the right people can't get to the thing they need.

## They pull against each other

Here's the part that makes security genuinely hard: **the three goals trade
against one another**. Push one up and you often push another down.

Lock a document behind heavy encryption and a stack of approvals and you've
strengthened confidentiality — but now a doctor in an emergency may struggle to
open it in time, which is an availability cost. Add rigorous integrity checks to
every transaction and the system gets slower. Move all your backups off-site for
availability and you've created more copies to keep confidential.

There's no setting where all three are maxed out for free. Every real security
decision is a **trade-off** among confidentiality, integrity, and availability,
and the right balance depends entirely on what you're protecting and who you're
protecting it from. A public weather page cares most about availability; a
password vault cares most about confidentiality; a financial ledger lives or
dies by integrity.

## Reading attacks through the triad

Once the three goals are in your head, they double as a quick way to classify
**what any attack is actually going after** — which points you straight at the
defenses that matter. Ask: which goal is under threat?

- **Eavesdropping** — quietly reading traffic or data in transit — attacks
  **confidentiality**. The answer is encryption and tighter access.
- **Tampering** — altering data, code, or messages — attacks **integrity**. The
  answer is hashing and signatures that make the change detectable.
- **Ransomware or a flood** — locking up your files, or drowning a server in
  traffic — attacks **availability**. The answer is backups and redundancy.

You'll meet each of these attacks in detail later in the path. For now, the habit
is what counts: name the goal under attack first, and the shape of the defense
follows.

<div class="knowledge-check" data-quiz data-correct-msg="Right — ransomware makes your own data unusable, so its primary target is availability." markdown="0">
  <p class="knowledge-check__q">Quick check: ransomware that encrypts your files and demands payment primarily attacks which goal?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Confidentiality (someone reads your secrets)</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Availability (your own data becomes unusable)</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Integrity (the data is secretly altered)</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- The **CIA triad** — confidentiality, integrity, availability — is the set of
  three goals that together define what "secure" means.
- **Confidentiality**: only authorized people can see it, enforced by encryption
  and access control.
- **Integrity**: it isn't altered without authorization and tampering is
  detectable, enforced by hashing and signatures.
- **Availability**: it's there when legitimate users need it, protected by
  redundancy and backups against outages and denial-of-service.
- The three **pull against each other** — every security decision trades among
  them, and the right balance depends on what you're protecting.
- The triad is also a **lens for attacks**: eavesdropping → confidentiality,
  tampering → integrity, ransomware or a flood → availability.

Next up: [threats, vulnerabilities & risk](/learn/cybersecurity/threats-vulnerabilities-risk/)
