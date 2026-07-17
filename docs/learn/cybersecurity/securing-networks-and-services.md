---
slug: securing-networks-and-services
title: Securing networks & services
description: Turning the defensive ideas into practice on real systems — expose only what you need, encrypt everything in transit, segment the network so one breach can't reach everything, put services behind a reverse proxy or VPN, and monitor so you notice trouble early.
keywords: network security, firewall, TLS, segmentation, VPN, reverse proxy, least exposure, secure service
level: intermediate
status: full
prereq:
  - defense-in-depth
faq:
  - q: What is the safest way to put a service on the internet?
    a: "Don't expose the application directly. Put a reverse proxy with TLS and authentication in front of it, open only the ports that service needs, and default-deny everything else at the firewall. If the service is only for you, skip public exposure entirely and reach it over a VPN or tunnel instead."
  - q: What does least exposure mean?
    a: "Least exposure means running and exposing only the services you actually need, and closing everything else. Every open port and running service is a way in, so the fewer you leave reachable, the smaller the attack surface. A default-deny firewall enforces this: nothing is reachable unless you explicitly allow it."
  - q: Why segment a network?
    a: "Segmentation separates systems into zones — guests, internal machines, sensitive servers — so a breach in one zone can't automatically reach the others. It's defense in depth applied to the network: an attacker who lands a foothold in one segment still has to get past another boundary to reach anything that matters."
  - q: Should I use a VPN or expose the service publicly?
    a: "If the service is only for you or a small trusted group, prefer a VPN or tunnel — it keeps the service off the public internet entirely, so attackers can't even reach it to probe. Public exposure with a reverse proxy, TLS, and authentication is for services that genuinely need to be reachable by anyone."
---

# Securing networks & services

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
Four habits turn the defensive ideas into working systems: **least exposure**
(run and open only what you need), **encrypt in transit** (TLS, SSH, VPNs so
nothing travels in the clear), **segment** (zones so one breach can't reach
everything), and **expose safely** (a reverse proxy or VPN in front, never the
raw app). This is [defense in depth](/learn/cybersecurity/defense-in-depth/)
applied to real networks and services. The networking path covers the
mechanics in [network security basics](/learn/networking/network-security-basics/).
</div>

The earlier lessons were about principles — layering, assume breach, matching
depth to risk. This one is where they meet a real machine with a real IP address.
The good news is that a handful of concrete habits cover most of it, and each one
maps directly to something you already know.

## Least exposure

The first and cheapest defense is simply **not being there to attack**. Every
service you run and every port you leave open is a way in; the fewer of them exist,
the less an attacker can even try.

So run only the services you actually need, and expose only the ones that must be
reachable from outside. Turn off the rest. Then enforce it with a **default-deny**
[firewall](/learn/networking/firewalls/): block everything by default and open only
the specific ports a service requires. Nothing is reachable unless you decided it
should be — the opposite of leaving doors open and hoping nobody tries them.

This pairs with host-level cleanup: removing unused packages, closing unneeded
ports, and disabling default accounts is the same idea one layer in, covered in
[hardening systems](/learn/cybersecurity/hardening-systems/). Less surface, fewer
holes.

## Encrypt everything in transit

Anything that travels the network in the clear can be read or tampered with by
whoever sits along the path. The fix is to encrypt it, and there's a standard tool
for each kind of traffic:

- **TLS** for web traffic and APIs — this is the "S" in HTTPS, and it's what keeps
  logins, tokens, and data private between browser and server. (See
  [TLS & HTTPS](/learn/networking/tls-and-https/).)
- **SSH** for remote administration — never log into a machine or push commands
  over an unencrypted channel. (See [SSH & remote access](/learn/linux-cli/ssh-and-remote/).)
- **VPNs** for private access — a VPN wraps a whole connection in encryption so you
  can reach internal systems safely across an untrusted network. (See
  [VPNs](/learn/networking/vpns/).)

The rule of thumb is simple: if it crosses a network you don't fully control,
encrypt it. On a modern setup that's nearly everything.

## Segment the network

A flat network — where every machine can talk to every other machine — means one
compromised device can reach them all. **Segmentation** breaks the network into
zones and controls what may cross between them, so a breach in one place doesn't
become a breach everywhere.

The everyday version is a guest network kept separate from your internal machines,
which in turn are kept separate from anything sensitive. An attacker who lands on a
guest device still has to get past another boundary to reach the systems that
matter — and that boundary is a chance to stop or notice them. This is exactly
[defense in depth](/learn/cybersecurity/defense-in-depth/) applied to the network:
you assume one zone will fall, and you make sure that failure stays contained.

## Expose services the right way

When a service genuinely has to be reachable from the internet, don't put the raw
application out there. Front it with a **reverse proxy** that terminates TLS and
handles authentication, and let only the proxy face the outside world. The app
itself listens on a private address behind it, so attackers reach a hardened,
purpose-built front door instead of the application's own code. (See
[exposing a service safely](/learn/networking/exposing-a-service-safely/).)

Even better, ask whether it needs to be public at all. If a service is just for you
or a small trusted group, prefer a **VPN or tunnel**: the service stays off the
public internet entirely, so an attacker can't even reach it to start probing. The
safest exposed port is the one that was never opened.

## Watch it

None of this is set-and-forget. Prevention eventually fails, so you need to
**notice** when something is wrong. Log and monitor your traffic and services —
watch for failed logins, unexpected connections, and services behaving oddly — so a
problem shows up as an alert instead of a surprise weeks later. (See
[monitoring & incident response](/learn/cybersecurity/monitoring-and-incident-response/)
and, for the hands-on side, [monitoring & logs](/learn/linux-cli/monitoring-and-logs/).)

Monitoring is the detection layer from defense in depth made concrete: the point of
segmenting and hardening is partly to give you fewer, clearer signals when something
does slip through.

<div class="knowledge-check" data-quiz data-correct-msg="Right — a reverse proxy with TLS and a default-deny firewall is the safe front door." markdown="0">
  <p class="knowledge-check__q">Quick check: before putting a web service on the internet, the safest first move is to...?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Expose the application directly and open every port so nothing breaks</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Front it with TLS and a reverse proxy and open only the ports it needs</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Turn off the firewall so the service is easy to reach</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Least exposure** — run and expose only what you need; close unused ports and
  default-deny at the firewall.
- **Encrypt in transit** — TLS for web and APIs, SSH for admin, VPNs for private
  access; encrypt anything crossing a network you don't control.
- **Segment** — split the network into zones so a breach in one can't reach
  everything; it's defense in depth for the network.
- **Expose safely** — a reverse proxy with TLS and authentication in front, or a
  VPN when the service is just for you — never the raw app on the open internet.
- **Watch it** — log and monitor traffic and services so trouble shows up as an
  alert, not a surprise.

Next up: [monitoring & incident response](/learn/cybersecurity/monitoring-and-incident-response/)
