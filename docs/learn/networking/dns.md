---
slug: dns
title: DNS — names to addresses
description: How the Domain Name System turns a name people remember into the IP address machines use — what a resolver does, how a lookup walks the hierarchy from root to authoritative nameserver, the common record types (A, AAAA, CNAME, MX, TXT, NS), and why results are cached for a TTL.
keywords: DNS, domain name, resolver, A record, AAAA, CNAME, MX, TXT, nameserver, TTL, caching, DNS lookup
level: beginner
status: full
prereq:
  - ip-addresses
faq:
  - q: What does DNS do?
    a: "DNS, the Domain Name System, translates a human-friendly name like a website's into the numeric IP address a machine needs to connect. It's the internet's phone book: you remember the name, DNS looks up the number behind it so your device knows where to send its traffic."
  - q: What is a DNS resolver?
    a: "A resolver is the service your device asks whenever it needs the IP address behind a name. It checks its own cache first, and if the answer isn't there it queries the DNS hierarchy on your behalf — root, then the top-level domain, then the domain's authoritative nameserver — and hands back the result."
  - q: What is the difference between an A record and a CNAME?
    a: "An A record maps a name directly to an IPv4 address (AAAA does the same for IPv6). A CNAME doesn't hold an address at all — it's an alias that points one name at another name, and the resolver then looks up that second name to find the actual address."
  - q: Why does a DNS change take time to appear?
    a: "Answers are cached for a period called the TTL (time to live). Until a cached record expires, resolvers keep serving the old answer instead of asking again. Lower the TTL before a planned change and the new answer spreads faster once you make it."
---

# DNS — names to addresses

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**DNS** (the Domain Name System) is the internet's phone book: it turns names
people remember into the [IP addresses](/learn/networking/ip-addresses/) machines
actually use. Your **resolver** does the lookup — checking its cache, then asking
up the hierarchy until it finds the answer. The answers come as **records**: an
**A** record maps a name to an IPv4 address, **AAAA** to IPv6, **CNAME** aliases
one name to another, **MX** points to mail servers, and **TXT** holds text used
for verification. To keep it fast, results are **cached** for a **TTL** before
they're looked up again.
</div>

Every time you type a name and a page loads, something quietly turned that name
into a number first. That something is DNS, and it runs so smoothly you rarely
notice — until it breaks. Understanding it makes a whole class of "the site is
down" mysteries obvious.

## The problem DNS solves

People remember names. `example.com` is easy; the machine behind it lives at a
numeric [IP address](/learn/networking/ip-addresses/) like `93.184.216.34` (or an
even longer IPv6 address), which nobody wants to memorise. But the network itself
only knows how to deliver data to **addresses**, not names.

DNS bridges that gap. It's a giant, distributed lookup system whose one job is to
answer the question "what address is this name?" You work with the friendly name;
DNS quietly supplies the number underneath so your traffic reaches the right
place.

## How a lookup works

When your device needs the address for a name, it doesn't search the whole
internet itself — it asks a **resolver** (usually run by your ISP or a public
service) to do the legwork.

The resolver works in steps:

1. **Check the cache.** If it looked this name up recently, it already has the
   answer and returns it immediately.
2. **Ask the root.** If not, it starts at the top — a **root** server — which
   doesn't know the address but knows who handles the **top-level domain** (the
   `.com`, `.org`, or `.net` part).
3. **Ask the top-level domain.** That server points to the **authoritative
   nameserver** for the specific domain — the one that actually holds its records.
4. **Ask the authoritative nameserver.** It returns the real answer, and the
   resolver passes it back to your device.

It sounds like a lot of hops, but it happens in milliseconds, and thanks to
caching most lookups never travel the full path.

## Why caching and TTL matter

Every answer comes with a **TTL** — a *time to live*, in seconds — that says how
long it may be reused before it must be looked up again. While a record is cached,
resolvers hand out the stored answer instead of walking the hierarchy, which is
what keeps DNS fast.

The trade-off shows up when a record changes: until the old TTL expires, resolvers
around the world keep serving the stale answer. That's why a DNS change can take a
while to "take effect" everywhere. Lowering the TTL ahead of a planned change lets
the new answer spread more quickly.

## Common record types

A domain's records come in several types, each answering a different question:

| Record | Maps a name to | Used for |
| --- | --- | --- |
| **A** | an IPv4 address | the everyday "where does this name live?" |
| **AAAA** | an IPv6 address | the same, for IPv6 |
| **CNAME** | another name (an alias) | pointing one name at another |
| **MX** | a domain's mail servers | where email for the domain is delivered |
| **TXT** | arbitrary text | verification, SPF, and other proofs |
| **NS** | a domain's nameservers | which servers are authoritative for it |

You'll meet **A** and **AAAA** constantly — they're the address records. **CNAME**
is handy when several names should follow one target: point them all at one name
and you only update that name's address once. **MX** and **TXT** show up whenever
email or domain ownership is involved.

## Why it matters

DNS is what lets a name stay stable while the machine behind it changes. Move a
service to a **new IP address**, update its A record, and everyone keeps using the
same name — no one has to learn a new number.

It also underpins more than web browsing. **MX** records are how email finds the
right servers, and **TXT** records are how a provider verifies you actually
control a domain. So DNS quietly sits behind email delivery and domain
verification too, not just websites.

And because so much depends on it, a slow or wrong DNS answer often *looks* like
something else entirely: the connection never starts, and it feels like "the site
is down" when really the name just didn't resolve. We'll turn that guess into a
test in [testing connectivity](/learn/networking/testing-connectivity/).

## Seeing it yourself

You don't have to take DNS on faith. Command-line tools like `dig` and `nslookup`
ask a resolver directly and print the records a name returns — the address, its
type, and the TTL counting down. We give them a fuller treatment in
[testing connectivity](/learn/networking/testing-connectivity/); for now, just
know the lookup is something you can watch happen.

<div class="knowledge-check" data-quiz data-correct-msg="Right — DNS turns names into the IP addresses machines use." markdown="0">
  <p class="knowledge-check__q">Quick check: what does DNS translate?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">IP addresses into MAC addresses</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Domain names into IP addresses</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Web pages into packets</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **DNS** is the internet's phone book — it turns names into
  [IP addresses](/learn/networking/ip-addresses/).
- Your **resolver** does the lookup: cache first, then up the hierarchy (root →
  top-level domain → authoritative nameserver).
- Records carry the answers: **A** (IPv4), **AAAA** (IPv6), **CNAME** (alias),
  **MX** (mail), **TXT** (verification), **NS** (nameservers).
- Answers are **cached for a TTL**, which is why changes take time to spread.
- A wrong or slow lookup often looks like "the site is down" — DNS is worth
  checking first.

Next up: [ports & sockets](/learn/networking/ports-and-sockets/)
