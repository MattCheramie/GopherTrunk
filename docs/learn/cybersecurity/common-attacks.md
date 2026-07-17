---
slug: common-attacks
title: Common attacks, in plain terms
description: A defender's field guide to the attacks you'll hear about most — malware, phishing and social engineering, credential attacks, man-in-the-middle, denial of service, and supply-chain attacks — explained at a high level so you can recognise each one and understand how it's broadly prevented.
keywords: cyber attacks, malware, phishing, credential stuffing, brute force, man in the middle, denial of service, supply chain, social engineering, ransomware, security awareness
level: beginner
status: full
prereq:
  - threats-vulnerabilities-risk
faq:
  - q: What is the most common type of cyber attack?
    a: Phishing and other social engineering are the ones most people meet, because they target the person rather than the technology and don't need any deep technical skill. A convincing email or message that tricks someone into typing their password or clicking a link is cheap to send and works often enough to stay popular. Awareness and multi-factor authentication are the broad defences.
  - q: What is the difference between brute force and credential stuffing?
    a: Brute force means guessing a password by trying many possibilities against one account until something works. Credential stuffing skips the guessing — it takes username-and-password pairs already leaked from some other breach and tries them elsewhere, betting that people reuse the same password across sites. Unique passwords and multi-factor authentication defeat both.
  - q: What does a man-in-the-middle attack do?
    a: It places an attacker between two parties who think they are talking directly, so the attacker can read or quietly alter the traffic passing through. The standard defence is encryption in transit — protocols like TLS that let each side confirm who they are talking to and keep the contents unreadable to anyone in the middle.
  - q: How do you defend against a denial-of-service attack?
    a: A denial-of-service attack floods a service until it can't serve real users, so defence is about absorbing or filtering the flood rather than stopping any single message. That means spare capacity, traffic filtering that drops obviously bad requests, and content delivery networks that spread and soak up large volumes before they reach your own servers.
---

# Common attacks, in plain terms

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
This is **a field guide, defensively** — a quick tour of the attacks you'll hear
about, what each one targets, and how they're broadly prevented, so the
specialised lessons that follow make sense. You don't need to know how any of
these are carried out to defend against them; you need to recognise the shape of
each and the handful of controls that blunt it. Every attack here maps back to
the [threats, vulnerabilities and risk](/learn/cybersecurity/threats-vulnerabilities-risk/)
you already met.
</div>

There are a lot of attack names, and the jargon makes them sound more mysterious
than they are. Most fall into a few families, and each family has a small set of
well-understood defences. This lesson stays **high level on purpose**: enough to
recognise and prevent each attack, and nothing that would help carry one out.

## Malware

**Malware** is simply *malicious software* — a broad label covering viruses,
worms, ransomware, spyware, and more. What they share is that they run code on a
machine that the owner didn't want there. Ransomware locks up your files and
demands payment; spyware quietly watches and steals; a worm spreads itself from
machine to machine. Between them they threaten both the **confidentiality** of
your data and the **availability** of your systems.

The broad defences are unglamorous and effective: keep software **updated** so
known holes are closed, run with **least privilege** so a compromise can't reach
everything, and keep **backups** so you can recover without paying anyone. We'll
go deeper in the endpoints lesson later in this path.

## Phishing & social engineering

Not every attack targets a computer — many target the **person** at the keyboard.
**Social engineering** is the art of tricking someone into handing over
credentials, approving a payment, or clicking something they shouldn't.
**Phishing** is the most common form: a message that impersonates someone
trustworthy to lure the reader into acting against their own interest.

Because these attacks bypass the technology and aim at human judgement, the
defences are human and procedural too. **Awareness** — knowing the shape of these
messages and slowing down when one arrives — stops most of them, and
[multi-factor authentication](/learn/cybersecurity/passwords-and-mfa/) means a
stolen password alone isn't enough to get in. There's a whole lesson on
[social engineering](/learn/cybersecurity/social-engineering/) coming, because
it's that important.

## Credential attacks

Some attacks go straight for the front door with someone else's key.
**Brute force** means guessing a password by trying many possibilities against an
account until one works. **Credential stuffing** doesn't bother guessing — it
takes passwords already leaked from *other* breaches and tries them on your site,
betting that people reuse the same password in many places. That bet pays off far
too often.

The defences line up neatly: **unique passwords** so a leak from one site can't
unlock another, [multi-factor authentication](/learn/cybersecurity/passwords-and-mfa/)
so a password by itself is worthless, and **rate limiting** so a system that sees
thousands of login attempts slows or blocks them instead of politely trying each
one.

## Man-in-the-middle

A **man-in-the-middle** attack slips an attacker *between* two parties who believe
they're talking directly to each other. From that position the attacker can read
the traffic, or quietly alter it, without either side noticing. Public Wi-Fi and
other untrusted networks are classic settings for it.

The broad defence is **encryption in transit**: if the traffic is encrypted and
each side can confirm who it's really talking to, an interceptor sees only
unreadable data and can't tamper undetected. This is exactly what
[TLS and HTTPS](/learn/networking/tls-and-https/) provide on the web. We'll cover
the family more fully in the network attacks lesson.

## Denial of service (DoS/DDoS)

A **denial-of-service** attack doesn't try to break in or steal anything — it
tries to make a service **unavailable** by overwhelming it with more traffic or
requests than it can handle. A **distributed** denial of service (DDoS) does the
same from many machines at once, which makes the flood far larger and harder to
trace. This one squarely targets **availability**, the third leg of the
[CIA triad](/learn/cybersecurity/cia-triad/).

You rarely stop such an attack with a single fix; you **absorb and filter** it.
That means having spare **capacity**, **filtering** that drops obviously bad
requests before they cost you real work, and **content delivery networks** that
spread and soak up huge volumes of traffic before it ever reaches your own
servers.

## Supply-chain attacks

Instead of attacking you directly, a **supply-chain attack** compromises
something you already trust — a software dependency, a vendor, an update
mechanism — and reaches you through it. One tampered library or vendor can touch
every organisation that uses it, which is what makes these attacks so efficient
and so damaging.

The defences are about **knowing and controlling what you depend on**: vet the
dependencies and vendors you bring in, keep them **updated** so fixed versions
reach you promptly, and pay attention to where your software actually comes from.
The secure-coding lesson later in this path returns to this from a builder's
point of view.

## Web application attacks

The applications you reach through a browser have their own distinct family of
attacks — ones that abuse how a web app handles input, sessions, and trust.
There's enough there to fill its own lesson, so we give it one: see
[web application attacks](/learn/cybersecurity/web-application-attacks/) next.

<div class="knowledge-check" data-quiz data-correct-msg="Right — reused passwords are exactly what makes stuffing pay off." markdown="0">
  <p class="knowledge-check__q">Quick check: credential stuffing succeeds mainly because…</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Attackers can guess any password given enough time</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">People reuse the same password across sites, so one breach unlocks others</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Encryption in transit has been broken</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- Most attacks fall into a few families, each with a small set of known defences.
- **Malware** is malicious software; counter it with updates, least privilege, and backups.
- **Phishing and social engineering** target people; counter them with awareness and MFA.
- **Credential attacks** (brute force, stuffing) fall to unique passwords, MFA, and rate limiting.
- **Man-in-the-middle** is beaten by encryption in transit; **denial of service** by capacity, filtering, and CDNs.
- **Supply-chain attacks** reach you through what you trust — vet and update your dependencies.

Next up: [web application attacks](/learn/cybersecurity/web-application-attacks/)
