---
slug: exposing-a-service-safely
title: Exposing a service safely
description: How to put a self-hosted service on the open internet without getting burned — decide whether it needs to be public at all, then front it with a reverse proxy, real HTTPS, a tight firewall, and strong authentication, layered so one weak spot isn't fatal.
keywords: expose service, reverse proxy, TLS, HTTPS, firewall, authentication, Let's Encrypt, hardening, least privilege, self-hosting security
level: advanced
status: full
prereq:
  - firewalls
faq:
  - q: Is it safe to expose a service to the internet?
    a: "It can be, if you put the right layers in front of it. On its own an app facing the open internet is a target for constant automated probing, but a reverse proxy terminating TLS, a firewall that opens only 443, and strong authentication turn it into a hardened front door. The safest option is often not to expose it at all — reach it over a VPN or SSH tunnel instead."
  - q: Why put a reverse proxy in front of a service?
    a: "A reverse proxy is the single front door: it terminates TLS, presents one clean certificate, and forwards requests to the app running privately behind it. The app itself never faces the internet, so its own port can stay closed and bound to localhost. The proxy is also where you add HTTPS, rate limiting, and access rules once, rather than baking them into every app."
  - q: Do I need HTTPS for a service only I use?
    a: "Yes. Without TLS, anyone between you and the server can read passwords and data in the clear or tamper with the page. A free automated certificate from Let's Encrypt makes HTTPS effectively zero-cost, so there is no reason to serve a login — or anything else — over plain HTTP, even for a personal service."
  - q: Which ports should I open to expose a web service?
    a: "Only 80 and 443 — or just 443 if you redirect all traffic to HTTPS. Everything else, including the app's own port, stays closed at the firewall and bound to localhost behind the reverse proxy. The narrower the opening, the less there is to attack."
---

# Exposing a service safely

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
Putting a service on the internet is fine — *if* you put the right layers in
front of it. Front it with a **reverse proxy + TLS**, open only the ports you
need at the **[firewall](/learn/networking/firewalls/)**, require
**authentication**, and — when you can — **prefer not to expose it at all**.
Each layer is independent, so one weak spot doesn't hand over the whole service.
</div>

A service reachable from the open internet is probed within minutes — automated
bots scan the whole address space constantly, looking for anything that answers.
That's not a reason to never expose anything; it's a reason to expose it
*deliberately*, with a few well-understood layers in front. This lesson walks
through them in the order you should think about them.

## First ask: does it need to be public?

The safest service is the one that was never exposed. Before opening a single
port, ask who actually needs to reach it. If it's just for you — or a handful of
people you can set up — you probably don't need the public internet at all.

A **[VPN](/learn/networking/vpns/)** puts your devices on the same private
network as the service, so it stays invisible to everyone else. An
**[SSH tunnel](/learn/networking/ssh-tunnels-and-transfers/)** forwards a single
port over an encrypted connection you already trust. Either one lets you reach
the service securely while it remains completely unexposed. Only go public when
you *truly* must — because other people need to reach it and you can't hand them
all a VPN profile.

## Put a reverse proxy in front

Don't expose the application directly. Put a
**[reverse proxy](/learn/networking/proxies-and-load-balancers/)** in front of
it and let that be the only thing facing the internet.

The proxy is your single front door: it accepts the connection, terminates TLS,
and forwards the request to the app running privately behind it. The app doesn't
need to know anything about certificates or the outside world — it just answers
on localhost. That separation is what lets you add HTTPS, rate limiting, and
access rules in **one** place instead of rebuilding them inside every service.

It also means you can run *several* services behind one address. The proxy looks
at the requested hostname or path and routes each request to the right app, so
adding a second service later doesn't mean opening a second hole in the firewall.
One door, many rooms.

## Use HTTPS with a real certificate

Everything the proxy serves should be over **TLS**. Not "the login page" —
*everything*. Without it, anyone on the path between the visitor and your server
can read passwords and page content in the clear, or quietly alter what's sent
back.

You no longer have to pay for or hand-manage certificates. A free, automated
certificate from **Let's Encrypt** renews itself, and most reverse proxies can
obtain and rotate one for you with a line or two of config. See
**[TLS & HTTPS](/learn/networking/tls-and-https/)** for how the handshake and
certificates actually work. The rule is simple: **never serve a login — or any
real data — over plain HTTP.**

## Lock down the firewall

The internet should be able to reach exactly two ports: **80 and 443** — or just
**443** if you redirect all HTTP straight to HTTPS. Everything else stays closed
at the **[firewall](/learn/networking/firewalls/)**.

That includes the application's own port. The app should bind to localhost and
sit privately behind the proxy, never listening on the public interface. If you
followed **[running a server](/learn/networking/running-a-server/)**, this is the
difference between a service that's *running* and one the whole internet can
*reach* — you want the former, plus a proxy that deliberately bridges the gap.
The narrower the opening, the less there is to attack.

## Authenticate and harden

The proxy and firewall control *who can knock*. Authentication controls *who
gets in*. Even a hardened front door is only as good as the lock on it, so treat
this as the layer that actually keeps people out — the rest just narrows who
reaches it. Require it, and don't stop there:

- **Require authentication** on anything that isn't meant to be public — and put
  it in front of admin panels, not just the data.
- **Use strong credentials and keys.** Long, unique passwords; SSH keys over
  passwords; no vendor defaults left in place.
- **Keep software patched.** The proxy, the app, and the OS — an unpatched
  service is the most common way in.
- **Rate-limit** login and expensive endpoints so a bot can't hammer them.
- **Run with least privilege.** A dedicated, non-root user with only the access
  the service needs, so a compromise doesn't own the whole box.
- **Log and monitor.** You can't react to what you can't see. Watch the proxy's
  access logs and the system journal.

These build on the
**[network security basics](/learn/networking/network-security-basics/)** —
assume the network is hostile and layer independent defenses — and on the Linux
path's **[monitoring & logs](/learn/linux-cli/monitoring-and-logs/)** for
actually reading what your service is doing.

## A launch checklist

Before you point DNS at the box and walk away:

- [ ] Confirmed it *needs* to be public — a VPN or tunnel wouldn't do.
- [ ] Reverse proxy is the only thing facing the internet.
- [ ] HTTPS everywhere, with an auto-renewing certificate.
- [ ] Firewall opens only 80/443; the app binds to localhost.
- [ ] Authentication required, strong credentials, defaults removed.
- [ ] Software patched, least-privilege user, rate limits in place.
- [ ] Logs are being written *and* watched.

<div class="knowledge-check" data-quiz data-correct-msg="Right — the reverse proxy is the single front door that handles TLS and keeps the real app private." markdown="0">
  <p class="knowledge-check__q">Quick check: the standard front door for a self-hosted web service — handling TLS and hiding the real app — is...</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Opening the app's port directly to the internet</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">A reverse proxy</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Turning off the firewall so nothing blocks it</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Prefer not to expose it.** A VPN or SSH tunnel reaches a private service
  without opening anything to the world.
- When you must go public, front it with a **reverse proxy** — the only thing
  the internet touches.
- **HTTPS everywhere** with a free auto-renewing certificate; never a login over
  plain HTTP.
- **Open only 80/443** at the firewall; the app stays private on localhost.
- **Authenticate and harden**: strong credentials, patches, least privilege,
  rate limits, and logs you actually watch.
- The layers are independent by design, so one weak spot doesn't lose the whole
  service.

Next up: [self-hosting on a home network](/learn/networking/self-hosting-at-home/)