---
slug: threats-vulnerabilities-risk
title: Threats, vulnerabilities & risk
description: The core vocabulary of security explained plainly — what a threat, vulnerability, exploit, and risk actually mean, how they combine into real danger, what an attack surface is, and how defenders measure and manage risk instead of trying to fix everything at once.
keywords: threat, vulnerability, exploit, risk, attack surface, risk management, likelihood, impact, threat vs vulnerability, security risk, mitigate risk, prioritize security
level: beginner
status: full
prereq:
  - what-is-cybersecurity
faq:
  - q: What is the difference between a threat and a vulnerability?
    a: A vulnerability is a weakness that already exists in a system — a missing patch, a weak password rule, an unlocked door. A threat is the potential harmful event that could take advantage of it, or the actor behind that event. The weakness is a property of your system; the threat comes from the outside world. Risk is what you get when a real threat can reach a weakness that matters.
  - q: What is an exploit?
    a: An exploit is the means of abusing a vulnerability — the specific technique or tool that turns a weakness into actual harm. For a defender, the useful point is not how any particular exploit works but that a vulnerability with a known, reachable path to abuse is far more urgent to fix than one with none.
  - q: How do defenders measure risk?
    a: Informally, risk is likelihood times impact — how probable a harmful event is, multiplied by how bad it would be if it happened. This lets you rank many possible problems and spend your limited time on the ones that combine a real chance of occurring with serious consequences, instead of treating every weakness as equally urgent.
  - q: Can you ever eliminate risk completely?
    a: No. Every system that does anything useful carries some risk, and driving it to zero would usually mean shutting the system off. The goal of security is not zero risk but managed risk — reducing what you can affordably reduce, and making a conscious, informed decision about what remains.
---

# Threats, vulnerabilities & risk

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
A **vulnerability** is a weakness. A **threat** is a potential harmful event, or
the actor behind it. An **exploit** is the means of abusing a vulnerability.
**Risk** is the chance that harm actually happens and how bad it would be — it
exists only where a threat can reach a vulnerability that matters. Your **attack
surface** is everything an attacker could interact with. Getting these few words
right lets you talk about security *precisely* instead of vaguely, which is the
first thing every later lesson depends on. New to the path? Start with
[what cybersecurity is](/learn/cybersecurity/what-is-cybersecurity/).
</div>

People use "threat", "risk", and "vulnerability" as if they mean the same thing.
They don't, and blurring them leads to muddled decisions — chasing scary-sounding
threats that can't actually reach you, or ignoring a boring weakness that's wide
open. A few precise words fix that. This lesson is just vocabulary, but it's the
vocabulary the whole rest of the path is built on.

## The vocabulary

Four words do most of the work. Learn them once and the rest of security gets
much easier to reason about.

- **Vulnerability** — a *weakness*. Something about a system that could allow harm:
  an unpatched bug, a default password left in place, a service exposed that
  didn't need to be, a person who can be talked into clicking a link. A
  vulnerability is a property of *your* system, and it exists whether or not
  anyone ever takes advantage of it.
- **Threat** — a potential *harmful event*, or the *actor* behind it. The
  possibility that something or someone causes damage: a criminal after money, a
  piece of malware, even a flood that takes out a server room. A threat comes from
  the *outside world*, not from inside your system.
- **Exploit** — the *means* of abusing a vulnerability. The specific technique or
  tool that turns a weakness into actual harm. As a defender you rarely need the
  details; what matters is *whether a workable path exists*, because a weakness
  with a known, reachable way to abuse it is far more pressing than one without.
- **Risk** — the *chance* that harm actually happens, together with *how bad* it
  would be. Risk is the word that ties the others together, and it's the one you
  ultimately make decisions about.

Keep the shapes straight: the vulnerability is *yours*, the threat is *out there*,
the exploit is the *bridge* between them, and the risk is the *result*.

## How they combine

Here's the key idea: **risk exists only where a threat can reach a vulnerability
that matters.** All three legs have to be present at once. Take any one of them
away and the risk drops — often to nothing.

- No **vulnerability**? A threat with nothing to abuse can't hurt you.
- No **threat**? A weakness nobody and nothing will ever act against causes no
  harm (though threats can appear later, so this is the weakest leg to rely on).
- No **path** the exploit can travel, or nothing of value at the end? Even a real
  weakness facing a real threat may carry little risk.

<figure class="figure" markdown="0">
<svg viewBox="0 0 440 130" role="img" aria-label="Three circles labelled threat, vulnerability, and value overlapping in the middle. The overlap is labelled risk. Text notes that removing any circle shrinks the overlap." xmlns="http://www.w3.org/2000/svg">
  <g fill="none" stroke="currentColor" stroke-width="1.3">
    <circle cx="150" cy="60" r="45" stroke-opacity="0.8"/>
    <circle cx="220" cy="60" r="45" stroke-opacity="0.8"/>
    <circle cx="185" cy="95" r="45" stroke-opacity="0.8"/>
  </g>
  <g font-size="10" fill="currentColor" text-anchor="middle">
    <text x="120" y="40">threat</text>
    <text x="252" y="40">weakness</text>
    <text x="185" y="120">value</text>
    <text x="185" y="70" font-weight="bold">risk</text>
  </g>
  <g font-size="9" fill="currentColor" text-anchor="start">
    <text x="315" y="58">Risk lives only</text>
    <text x="315" y="72">where all three</text>
    <text x="315" y="86">overlap.</text>
  </g>
</svg>
<figcaption>Risk is the overlap: a threat, a weakness it can reach, and something of value at stake. Remove any one and the overlap shrinks.</figcaption>
</figure>

**A worked example.** Suppose a spare monitoring machine on your desk runs an old
service with a known weakness (a **vulnerability**). Criminals routinely scan the
internet for exactly that weakness (a **threat**). If that machine sits directly
on the public internet, the two overlap and the **risk** is real. Now change one
leg: put the machine behind your home router where it isn't reachable from
outside. The vulnerability still exists and the threat still exists, but the
threat can no longer *reach* the weakness — so the risk falls sharply. You didn't
patch anything; you removed the overlap. (Patching the service as well would be
better still — defenders like to remove *more* than one leg.)

## Attack surface

Your **attack surface** is everything an attacker could interact with — every
point where the outside world touches your system. Open network ports, every
input field or file your software accepts, the third-party libraries and
**dependencies** you pull in, the accounts and the *people* who hold them. Each of
those is a place a threat might find a vulnerability.

The bigger the attack surface, the more chances something is weak somewhere. So
one of the cheapest, most reliable wins in all of security is simply **making the
attack surface smaller**: turn off services you don't use, close ports you don't
need, drop dependencies you don't rely on, remove accounts nobody uses. You can't
have a vulnerability in a thing that isn't there. We'll return to this as a core
practice in [hardening systems](/learn/cybersecurity/hardening-systems/).

## Measuring risk

You will always find more weaknesses than you have time to fix. That's normal —
so defenders *rank* risks instead of treating them all alike. The everyday rule of
thumb is:

> **risk ≈ likelihood × impact**

**Likelihood** is how probable the harmful event is; **impact** is how bad it
would be if it happened. A weakness that is easy to reach *and* would be
catastrophic is a top priority. One that is nearly impossible to reach and would
barely matter can wait. Multiplying the two lets you sort a long, messy list into
a short one worth acting on.

This is why "just fix everything" is not a real strategy. Time, money, and
attention are limited; spending them on a remote, low-impact problem while a
likely, high-impact one waits is a bad trade. Measuring risk — even roughly — is
what lets you spend your effort where it actually reduces harm.

## Managing risk

Once you've sized a risk, you have four honest options for what to do about it:

- **Reduce (mitigate)** — make the likelihood or the impact smaller: patch the
  weakness, add a control, shrink the attack surface. The most common response.
- **Accept** — decide the risk is small enough to live with, and consciously do
  nothing. This is a legitimate choice *when it's deliberate and informed* — not
  when it's the result of never looking.
- **Transfer** — hand some of the impact to someone else, most familiarly by
  buying **insurance**, or by using a service that carries the risk for you. The
  risk doesn't vanish; the cost of it lands elsewhere.
- **Avoid** — don't do the risky thing at all: drop the feature, retire the
  system, don't collect the data you'd have to protect.

Real security work is just applying these four choices, over and over, to the
risks that matter most. That's the whole game: **security is applied risk
management**, and the habit of thinking this way is what the next lesson calls the
[security mindset](/learn/cybersecurity/security-mindset/).

<div class="knowledge-check" data-quiz data-correct-msg="Right — a weakness in software that could be abused is a vulnerability. The threat is what might act against it; the risk is the chance of harm." markdown="0">
  <p class="knowledge-check__q">Quick check: A flaw in software that could be abused is a...</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Threat</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Vulnerability</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Risk</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- A **vulnerability** is a weakness in your system; a **threat** is a potential
  harmful event or the actor behind it; an **exploit** is the means of abusing a
  weakness; **risk** is the chance and severity of actual harm.
- **Risk exists only where a threat can reach a vulnerability that matters** —
  remove any leg and the risk drops.
- Your **attack surface** is everything an attacker could interact with; shrinking
  it is one of the cheapest wins in security.
- You can't fix everything, so rank risks by **likelihood × impact** and act on
  the top of the list.
- Manage each risk one of four ways: **reduce, accept, transfer, or avoid** —
  security is applied risk management.

Next up: [who attacks, and why](/learn/cybersecurity/threat-actors/)
