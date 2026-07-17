---
slug: network-attacks
title: Network attacks
description: "The network-layer attacks from the networking path, seen from the defender's chair — interception (sniffing and man-in-the-middle), spoofing, denial of service, and reconnaissance — and the layered defenses that stop them: encryption in transit, segmentation, and monitoring."
keywords: network attack, man in the middle, spoofing, sniffing, ARP spoofing, DNS spoofing, DDoS, denial of service, network segmentation, reconnaissance, encryption in transit
level: intermediate
status: full
prereq:
  - common-attacks
faq:
  - q: What is a man-in-the-middle attack?
    a: A man-in-the-middle attack is when an attacker quietly positions themselves between two parties and reads or alters the traffic passing between them. Both sides think they're talking directly to each other. The defense is encryption in transit — TLS, SSH, or a VPN — which makes captured traffic useless because the attacker can't read or forge it without the keys.
  - q: How do you defend against network spoofing?
    a: Spoofing forges an identity or address to redirect or impersonate. Defenses are authentication (prove who you are, don't just trust an address), DNS security (validate the answers you get), and a zero-trust posture where being on the network doesn't automatically make you trusted. Encryption also helps, because a forged endpoint can't complete an authenticated, encrypted session.
  - q: What is the difference between DoS and DDoS?
    a: A denial-of-service (DoS) attack floods a service from one source to make it unavailable. A distributed denial-of-service (DDoS) attack does the same thing from many sources at once, which is harder to filter. Both attack availability. Mitigations include spare capacity, rate limiting, upstream filtering, and content delivery networks that absorb the load.
  - q: Why does network segmentation matter?
    a: Segmentation splits a network into separate zones so a breach in one area can't automatically reach everything else. If an attacker compromises one machine, segmentation limits how far they can move. It turns a single foothold into a contained incident instead of a whole-network compromise, and it's a core part of defense in depth.
---

# Network attacks

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
Network attacks come in three broad shapes: **interception** (reading or altering
traffic in transit), **spoofing** (forging an identity or address), and **floods**
(overwhelming a service). These are the same network-layer attacks from the
[networking path](/learn/networking/network-security-basics/), now seen from the
defender's chair. The defenses layer up: **encrypt** everything in transit,
**segment** networks so one breach can't reach everything, and **monitor** for the
anomalies that give an attacker away.
</div>

You've met the [common attacks](/learn/cybersecurity/common-attacks/) that target people
and software. This lesson looks at the attacks that target the **network itself** — the
wires, the routing, and the trust between machines. If you've worked through the
networking path's [network security basics](/learn/networking/network-security-basics/),
the mechanics will be familiar; here we care about what they mean for a **defender** and
what actually stops them.

## Interception (sniffing & man-in-the-middle)

The most basic network attack is simply **reading traffic that isn't yours**. On a shared
or poorly isolated network, an attacker can **sniff** packets as they pass — capturing
whatever flows by. A step further is the **man-in-the-middle (MITM)**: the attacker
positions themselves *between* two parties so that all traffic flows through them. Now they
can not only read it but **alter** it — changing a message, injecting content, or harvesting
credentials — while both sides believe they're talking directly.

**The defense is encryption in transit.** If the traffic is encrypted with
[TLS](/learn/networking/tls-and-https/), SSH, or a [VPN](/learn/networking/vpns/), then
captured packets are just noise: the attacker can't read them and can't forge valid ones
without the keys. A MITM who can't decrypt the session can't meaningfully tamper with it
either. This is why "encrypt everything" has become the default — it turns interception
from a serious breach into a non-event. The remaining job is making sure the encryption is
actually verified (certificates checked, not blindly accepted), so an attacker can't slip in
a fake endpoint.

## Spoofing

**Spoofing** is forging an identity or address to redirect traffic or impersonate someone
trusted. It comes in several flavors:

- **ARP spoofing** — forging the mapping between an IP address and a hardware address on a
  local network, so traffic meant for one machine is quietly sent to the attacker (a common
  way to set up a MITM).
- **DNS spoofing** — feeding a victim a false answer to a name lookup, so a legitimate
  address sends them to an attacker's server instead.
- **IP spoofing** — forging the source address on packets to impersonate a trusted host or
  hide the real origin.
- **Email spoofing** — forging the sender of a message so it appears to come from someone the
  recipient trusts (the backbone of many phishing attempts).

The common thread is **abusing trust in an address or name**. The defenses follow from that:
**authentication** (prove identity cryptographically rather than trusting an address),
**DNS security** (validate the answers a lookup returns), and a habit of **not trusting the
network by default** — the zero-trust idea that being *on* a network shouldn't make you
trusted. Encryption helps here too: a spoofed endpoint can't complete an authenticated,
encrypted session, so the forgery fails at the handshake.

## Denial of service (DoS/DDoS)

Not every attack tries to steal or read data — some just try to **knock a service offline**.
A **denial-of-service (DoS)** attack overwhelms a target with more requests or traffic than
it can handle, so legitimate users can't get through. A **distributed** denial-of-service
(**DDoS**) does the same from **many sources at once** — often a large botnet — which makes
it far harder to simply block the source.

This is an attack on **availability**, the third leg of the
[CIA triad](/learn/cybersecurity/cia-triad/): the data is fine and confidential, but nobody
can reach it. Because it's a volume problem, the defenses are about **absorbing or shedding
load**: spare **capacity** to soak up spikes, **rate limiting** to cap what any one source can
demand, upstream **filtering** to drop obviously malicious traffic before it lands, and
**content delivery networks (CDNs)** that spread requests across a large, distributed
infrastructure built to absorb floods.

## Reconnaissance

Before most attacks comes **reconnaissance** — the attacker mapping what's there. They
**scan** for open ports, running services, software versions, and anything that reveals a way
in. Recon isn't damaging by itself, but it's the step that makes everything after it targeted
instead of blind.

The defensive lesson is about **reducing exposure**. Every open port and reachable service is
something an attacker can find and probe, so **closing ports you don't need** and putting a
[firewall](/learn/networking/firewalls/) in front of what remains shrinks the picture an
attacker can build. You can't stop someone from scanning, but you can make sure the scan comes
back nearly empty — the smaller your exposed surface, the less recon gives them to work with.

## The layered defense

Notice that no single control stops all of these — but a few, layered together, cover most of
the ground:

- **Encrypt everything in transit.** Defeats interception outright and blunts spoofing by
  forcing attackers through an authenticated handshake they can't fake.
- **Segment networks.** Split the network into zones so a breach in one area **can't reach
  everything**. A compromised machine becomes a contained incident instead of a whole-network
  compromise.
- **Monitor for anomalies.** Recon scans, sudden traffic floods, and traffic going somewhere
  it shouldn't all leave traces. Watching for them is how you catch an attack in progress.

This is [defense in depth](/learn/cybersecurity/defense-in-depth/) applied to the network:
overlapping controls so that when one fails, another still stands. Pair it with
[monitoring and incident response](/learn/cybersecurity/monitoring-and-incident-response/) so
that the anomalies you detect actually get acted on.

<div class="knowledge-check" data-quiz data-correct-msg="Right — encryption makes captured traffic useless, so a man-in-the-middle can neither read nor forge it." markdown="0">
  <p class="knowledge-check__q">Quick check: the primary defense against traffic interception (a man-in-the-middle) is...</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">A stronger password</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Encryption in transit (TLS/SSH/VPN)</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">A faster internet connection</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- Network attacks fall into three shapes: **interception**, **spoofing**, and **floods**.
- **Interception** (sniffing and MITM) is defeated by **encryption in transit** — TLS, SSH,
  or a VPN make captured traffic useless.
- **Spoofing** forges an address or identity; counter it with **authentication**, **DNS
  security**, and **not trusting the network by default**.
- **DoS/DDoS** attacks **availability**; mitigate with capacity, rate limiting, filtering,
  and CDNs.
- **Reconnaissance** comes first — **closing ports and using firewalls** shrinks what a scan
  finds.
- The layered defense: **encrypt, segment, and monitor**, so one breach can't become total
  compromise.

Next up: [malware & endpoint security](/learn/cybersecurity/malware-and-endpoints/)
