---
slug: threat-actors
title: Who attacks, and why
description: A plain-language, defender-focused tour of the people and groups behind cyber attacks — opportunists, cybercriminals, hacktivists, insiders, and nation-state APTs — the motives that drive them, and how knowing your own threat model decides where defense is worth the effort.
keywords: threat actor, attacker, cybercriminal, insider threat, nation-state, hacktivist, APT, motive, threat model, opportunistic attack
level: beginner
status: full
prereq:
  - threats-vulnerabilities-risk
faq:
  - q: What is a threat actor?
    a: "A threat actor is any person or group that could cause harm to a system — from a bored opportunist running someone else's tool, to an organised criminal gang, to a well-funded nation-state team. Grouping attackers this way helps defenders reason about who might realistically target them and what those attackers tend to want."
  - q: Who is behind most cyber attacks?
    a: "Most everyday harm is opportunistic and financially motivated — cybercriminals chasing money through fraud, ransomware, and theft, plus automated bots that scan the whole internet looking for anything easy to break into. Targeted attacks by patient, well-resourced groups are real but far rarer for ordinary systems."
  - q: What is an insider threat?
    a: "An insider threat is a risk that comes from someone with legitimate access — an employee, contractor, or partner. It can be malicious (deliberate theft or sabotage) but is very often accidental: a misconfiguration, a lost laptop, or a mistake. Not every 'attacker' means harm, which is why insiders and human error matter as much as outsiders."
  - q: What is an APT?
    a: "APT stands for advanced persistent threat — usually a nation-state or similarly well-resourced group that pursues a target patiently over a long time, typically for espionage or disruption rather than quick money. They are the high end of the threat spectrum and most systems will never face one."
---

# Who attacks, and why

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
Defending well starts with knowing **who** might come after you and **why**. The
people and groups behind attacks are **threat actors**, and what drives them are
their **motives** — money, ideology, espionage, disruption, or plain accident.
Matching those actors and motives against what you actually have to protect is
your **threat model**, and it decides which defenses are worth the effort. Build
on the vocabulary from
[threats, vulnerabilities and risk](/learn/cybersecurity/threats-vulnerabilities-risk/).
</div>

You can't defend sensibly against everyone at once. A small personal blog and a
national bank face very different attackers wanting very different things, and
they should spend their effort differently. This lesson introduces the cast of
**threat actors** and their **motives** so you can start reasoning about who
would realistically target *you* — the first step toward a defense that fits.

## The cast

"Attacker" isn't one kind of person. It helps to picture a few broad groups,
knowing the edges blur and real incidents often mix them:

- **Opportunists** (sometimes called "script kiddies"). People — often
  inexperienced — running tools and exploits that others wrote, for fun,
  curiosity, or bragging rights. They rarely target you specifically; they poke
  at whatever is easy and see what gives way.
- **Cybercriminals.** Organised, professional, and in it for **money**. Fraud,
  ransomware, and outright theft of data or funds are their trade. This is where
  most real-world harm comes from, and it operates like a business — complete
  with tooling, marketplaces, and division of labour.
- **Hacktivists.** Driven by **ideology** or a cause rather than profit. They may
  deface sites, leak documents, or knock services offline to make a political or
  social point.
- **Insiders.** People who already have legitimate access — employees,
  contractors, partners. Some act maliciously, but many "insider" incidents are
  simply **careless**: a mistake, a misconfiguration, a shared password.
- **Nation-state groups / advanced persistent threats (APTs).** The high end:
  government-backed or similarly well-resourced teams pursuing **espionage** and
  **disruption**. They are patient, skilled, and persistent, and most ordinary
  systems will never be worth their attention.

The point of the list isn't to memorise labels — it's to notice that these groups
want *different things* and put in *different amounts of effort*.

## Motives

Strip away the labels and a handful of **motives** explain almost everything:

- **Money.** By far the most common driver — fraud, ransomware, stealing data to
  sell, draining accounts.
- **Ideology.** Advancing a cause, protest, or belief.
- **Espionage.** Quietly stealing secrets — commercial, political, or military.
- **Disruption.** Breaking or degrading a service, whether to send a message or
  cause damage.
- **Accident and negligence.** Not really a "motive" at all, but a huge source of
  harm — someone with access simply makes a mistake.

The practical takeaway: **most everyday harm is opportunistic and financial**, not
a targeted vendetta. Understanding what an attacker is likely to *want* tells you
what they're likely to *go after*, which is exactly what defense planning needs.

## Automated and opportunistic

Here's a fact that reshapes how beginners think about risk: a large share of
attack traffic isn't a person choosing you at all. It's **automated bots**
scanning huge swaths of the internet, constantly, for known weaknesses — an
unpatched service, a default password, an exposed database.

That means **you don't have to be a target to be hit.** If your system is
reachable and has an easy hole, the scanners will find it whether or not anyone
has ever heard of you. This is the whole reason the unglamorous basics matter so
much: keeping software updated, closing default credentials, and removing easy
footholds stops the vast automated majority before a human is ever involved.
We'll cover those basics directly in
[hardening systems](/learn/cybersecurity/hardening-systems/) later in the path.

## Your threat model

Put the actors and motives together against what *you* have, and you get a
**threat model** — a simple, honest answer to two questions:

- **Who would realistically come after this?**
- **What would they want from it?**

That answer decides where your effort belongs. A hobby blog's realistic threat is
opportunistic bots and defacement, so the basics — updates, strong login, backups
— cover most of it. A bank faces organised criminals and possibly nation-state
interest, so it invests far more, across many layers. Neither is "doing security
wrong"; they simply have different threat models.

Threat modelling keeps you from two classic mistakes: **over-spending** to defend
against attackers who'd never bother with you, and **under-spending** because you
assume "nobody would target me" — forgetting that the bots don't care who you are.

## Insiders and mistakes

It's tempting to picture every attacker as a hooded outsider, but a great many
incidents start *inside* and *without malice*. A misconfigured storage bucket left
open to the world. An email sent to the wrong recipient. A laptop left on a train.
A well-meaning employee tricked into handing over a password.

Two lessons follow from this. First, **human error is part of your threat model**,
not a footnote — good defenses assume people will occasionally slip and limit the
damage when they do. Second, one of the most effective "attacks" isn't technical
at all: it's manipulating a person into helping, which is why
[social engineering](/learn/cybersecurity/social-engineering/) gets its own
lesson. Building the everyday habits that resist both mistakes and manipulation is
the heart of the [security mindset](/learn/cybersecurity/security-mindset/), coming
up next.

<div class="knowledge-check" data-quiz data-correct-msg="Right — cybercriminals are the money-motivated group, and they cause most real-world harm." markdown="0">
  <p class="knowledge-check__q">Quick check: which threat actor is primarily motivated by money?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Hacktivists</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Cybercriminals</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Nation-state APTs</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Threat actors** are the people and groups behind attacks: opportunists,
  cybercriminals, hacktivists, insiders, and nation-state APTs.
- Their **motives** boil down to money, ideology, espionage, disruption, and plain
  accident — and **most everyday harm is opportunistic and financial**.
- Much attack traffic is **automated bots** scanning everything, so you don't have
  to be a target to be hit — which is why the hardening basics matter.
- Your **threat model** — who would realistically come after you, and for what —
  decides where defense is worth the effort.
- Not every "attacker" is malicious: **misconfiguration and human error** cause
  many incidents, so people are part of the model too.

Next up: [the security mindset](/learn/cybersecurity/security-mindset/)
