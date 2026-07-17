---
slug: network-security-basics
title: Network security basics
description: The handful of principles that keep a networked service from becoming an easy target — assume the network is hostile, encrypt traffic in transit, expose as little as possible, use strong authentication, and layer your defenses so one failure isn't fatal.
keywords: network security, man in the middle, spoofing, DDoS, encryption, least exposure, defense in depth, port scan
level: intermediate
status: full
prereq:
  - tls-and-https
faq:
  - q: What is the most important rule of network security?
    a: "Assume the network is hostile. Traffic crosses machines you don't control, and automated bots probe the whole internet constantly, so design as if someone is always watching and testing. Everything else — encrypting traffic, closing unused ports, using strong credentials — follows from taking that assumption seriously instead of hoping nobody notices you."
  - q: What is a man-in-the-middle attack?
    a: "A man-in-the-middle sits between you and the server you meant to reach, reading or quietly altering the traffic as it passes. If the connection is unencrypted, they can see passwords and page content in the clear; if they can impersonate the server, they can feed you altered data. Encryption with verified identity, like TLS, is what closes this gap."
  - q: What is defense in depth?
    a: "Defense in depth means layering several independent controls so that no single failure is fatal. You encrypt traffic, close unused ports, require strong authentication, patch software, and monitor logs — and if one layer is bypassed, the others still stand. It's the opposite of relying on one wall to keep everything out."
  - q: What is a DDoS attack?
    a: "A distributed denial-of-service flood overwhelms a service with more traffic or requests than it can handle, so legitimate users can't get through. It doesn't break in or steal data — it simply exhausts the target's capacity. Defending against large floods usually needs help from your network or hosting provider, not just your own machine."
---

# Network security basics

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
Four ideas carry most of network security: **assume the network is hostile**,
**encrypt** everything in transit, keep **least exposure** by closing what you don't
need, and build **defense in depth** so one failure isn't the whole game. None of them
is exotic — a handful of principles keep a networked service from becoming an easy
target. This lesson builds on [TLS & HTTPS](/learn/networking/tls-and-https/), which is
the encryption piece in action.
</div>

Security sounds like a specialist's job, and at the deep end it is. But the ideas that
stop the most common trouble are simple and general — the same few principles apply
whether you're running a personal dashboard or a public API. This lesson is the map;
later modules get hands-on.

## Assume the network is hostile

The first principle is a mindset. Your traffic crosses routers, switches, and cables you
don't own and can't see — a café's Wi-Fi, your ISP, the wider internet. Any of those
points could, in principle, watch or interfere with what passes through.

So design as if someone is watching and probing your service — because they are.
Automated bots sweep the entire internet around the clock, knocking on every address and
port, looking for anything unprotected. This isn't personal; it's constant background
noise. Anything you put online gets found and tested within minutes, so "nobody knows
it's here" is never a defense.

## Common threats

You don't need an attacker's toolkit to recognise the handful of shapes trouble usually
takes:

- **Interception / man-in-the-middle** — someone on the path reads or alters your
  traffic as it flows, seeing passwords and content if it isn't encrypted.
- **Spoofing** — someone pretends to be a machine or user they aren't, impersonating a
  trusted server or forging their address to slip past a check.
- **Unauthorized access** — someone gets in through weak, reused, or default credentials
  that were never changed from the factory setting.
- **Floods / DDoS** — someone overwhelms the service with more traffic than it can
  handle, so real users can't get through.

Each has a defense, and the defenses overlap.

## The core defenses

A short, boring checklist stops most of the above:

- **Encrypt everything in transit.** Use [TLS](/learn/networking/tls-and-https/) for web
  traffic and SSH for remote access so anyone intercepting the path sees scrambled bytes,
  not your data. This defeats plain interception outright.
- **Least exposure.** Close ports you aren't using, put a
  [firewall](/learn/networking/firewalls/) in front of the rest, and don't run services
  you don't actually need. Every open door is something to defend; the fewer, the better.
- **Strong authentication.** Use keys and unique, non-guessable passwords — never the
  defaults a device shipped with. Weak or reused credentials are the single most common
  way in.
- **Keep software patched.** Known holes get scanned for automatically. Updating closes
  them before someone walks through.
- **Monitor and log.** Keep records so you actually notice when something is wrong,
  instead of finding out weeks later. You can't respond to what you can't see.

Notice none of these is clever. They're habits, and their power is in doing all of them,
not picking a favourite.

## Defense in depth

That last point is the whole idea behind **defense in depth**: no single control is
enough on its own, so you layer several independent ones. Encryption protects the traffic;
a firewall limits what's reachable; strong authentication guards the door; patching closes
known holes; logging catches what slips through.

Stacked together, one failure isn't fatal. If a password leaks but the port is firewalled
off, the attacker still can't reach it. If a service is exposed but patched and behind
strong auth, a random probe bounces off. The layers cover for each other — which is
exactly why relying on any one of them is a mistake.

## Privacy is part of it

Security isn't only about keeping intruders out; it's also about what data crosses the
wire and where it comes to rest. A service can be perfectly locked down and still leak
sensitive information because it collects more than it needs or stores it somewhere
careless. Thinking about *what* data flows, and *who* can see it along the way, is part of
the same discipline — see
[privacy and data agreements](/learn/software-licensing/privacy-and-data-agreements/) for
the data-handling side of the picture.

## Applied

Before you expose anything to a network, walk the same three questions every time:

1. **Encryption** — is the traffic protected in transit, or is it travelling in the clear?
2. **Exposure** — what's actually reachable, and can you make it less?
3. **Authentication** — who can get in, and are the credentials strong and unique?

That's the routine you'll apply directly when
[exposing a service safely](/learn/networking/exposing-a-service-safely/) later in this
path. Treat this lesson as educational grounding, not a substitute for a real security
review — a service that handles anything sensitive deserves expert eyes.

<div class="knowledge-check" data-quiz data-correct-msg="Right — reading traffic from the middle of the path is a man-in-the-middle attack." markdown="0">
  <p class="knowledge-check__q">Quick check: an attacker on the same Wi-Fi silently reading your unencrypted traffic is an example of...</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">A denial-of-service flood</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Interception / a man-in-the-middle</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Software that's out of date</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Assume the network is hostile** — bots probe constantly, so anything online is found
  and tested fast.
- The common threats are **interception**, **spoofing**, **unauthorized access**, and
  **floods / DDoS**.
- The core defenses: **encrypt** in transit, keep **least exposure**, use **strong
  authentication**, patch software, and monitor logs.
- **Defense in depth** layers those controls so one failure isn't fatal.
- **Privacy** — what data crosses the wire and where it's stored — is part of security too.
- Before exposing anything, run through **encryption, exposure, and auth** — and lean on a
  real review for anything sensitive.

Next up: Module 5 gets hands-on at the command line — [inspecting your network](/learn/networking/inspecting-your-network/)
