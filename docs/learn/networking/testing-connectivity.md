---
slug: testing-connectivity
title: Testing connectivity
description: A repeatable way to turn "it won't connect" into a specific answer — using ping to test reachability, traceroute to see the path, dig and nslookup to check DNS, and a port check to confirm a service is listening, worked from the bottom of the stack upward.
keywords: ping, traceroute, dig, nslookup, connectivity, diagnose network, latency, DNS lookup, port open, ICMP, layered diagnosis
level: intermediate
status: full
prereq:
  - dns
faq:
  - q: How do I test if a host is reachable?
    a: "Use ping — it sends small ICMP echo requests and reports whether replies come back and how long they take. Replies mean the host is up and the path to it works; steady low times mean low latency. But some hosts and firewalls block ping on purpose, so no reply doesn't always mean the host is down — check the actual service port before concluding it's offline."
  - q: What is the difference between ping and traceroute?
    a: "Ping tells you whether the destination answers and how long a round trip takes. Traceroute shows the route your packets take to get there — each router hop along the way, with a time for each. Ping answers 'can I reach it?'; traceroute answers 'where does the path break or slow down?' when the answer to the first question is no or sluggish."
  - q: A site works by IP but not by name — what's wrong?
    a: "That's the classic signature of a DNS problem. The machine is reachable, since the raw IP address connects, but the name isn't resolving to that address. Test it with dig or nslookup: if the lookup fails or returns the wrong address, the fault is in name resolution — your resolver, the record, or its cache — not in the network path or the service itself."
  - q: How do I check whether a port is open?
    a: "Point a tool like nc (netcat) or telnet at the host and the port number, for example nc -vz host 443. If the connection is accepted, a service is listening there; if it's refused or times out, either nothing is listening or a firewall is blocking it. This is a different question from whether the host is up — a machine can answer ping while the specific port you need stays closed."
---

# Testing connectivity

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
When something "won't connect," don't guess — test. **ping** checks whether a host
is reachable and how fast; **traceroute** shows the path your packets take and where
it breaks; **dig** (or `nslookup`) confirms [DNS](/learn/networking/dns/) is
answering with the right address. Put them together as a **layered diagnosis**:
work up from the bottom of the stack — does the name resolve, can I reach the IP,
is the port open, does the app respond — and each layer's result points at the
next thing to check. That turns a vague failure into a specific answer.
</div>

"It won't connect" is not a diagnosis — it's a symptom with a dozen possible
causes. The skill isn't knowing every cause; it's having a repeatable order of
checks that eliminates them one at a time until the real one is left standing.

## ping — is it reachable?

`ping` sends small **ICMP echo request** messages to a host and listens for the
replies that come back. If replies arrive, the host is up and the path between you
and it works; the round-trip time printed with each reply is your **latency** to
that host.

```
$ ping example.com
64 bytes from 93.184.216.34: icmp_seq=1 ttl=56 time=11.4 ms
64 bytes from 93.184.216.34: icmp_seq=2 ttl=56 time=11.2 ms
```

Steady, low times mean a healthy connection. Rising times or dropped replies point
at congestion or a flaky link. One caveat: **some hosts and firewalls block ping**
deliberately, so **no reply is not proof the host is down** — it may simply be
ignoring ICMP while its real services answer fine. Treat a silent ping as "can't
confirm," not "confirmed dead."

## traceroute — the path

When a host is slow or unreachable, `traceroute` (`tracert` on Windows) shows you
*where*. It lists the **hops** — each router your packets pass through — between
you and the destination, with a time for each.

```
$ traceroute example.com
 1  router.home  1.2 ms
 2  isp-gateway  9.8 ms
 3  * * *
 4  core1.isp    12.1 ms
```

Read it top to bottom: the hops climb steadily until they either reach the
destination or stop advancing. A line of `* * *` that never recovers is where the
path **breaks**; a hop where times suddenly jump is where it **slows**. That tells
you whether the problem is near you, out in the middle, or at the far end.

## dig / nslookup — is DNS working?

If a name won't connect, the very first thing worth ruling out is name resolution.
`dig` and `nslookup` ask a resolver directly and print what a name resolves to —
so you can confirm [DNS](/learn/networking/dns/) is answering and handing back the
**right address**.

```
$ dig +short example.com
93.184.216.34
```

If that returns nothing, or returns an address that doesn't match where the service
actually lives, DNS is your culprit — no amount of pinging the *name* will help
until the lookup is fixed. A successful lookup, by contrast, hands you the raw IP
address to test the layers below with.

## Is the port open?

A host being up doesn't mean the **service** you want is running. Reaching the
machine and reaching a specific [port](/learn/networking/ports-and-sockets/) on it
are two different questions. A quick way to check the second is to try opening a
connection to that port:

```
$ nc -vz example.com 443
Connection to example.com 443 port [tcp/https] succeeded!
```

`nc` (netcat) or `telnet host port` will either connect — a service is **listening**
there — or fail with "refused" or a timeout, meaning nothing is listening or a
firewall is in the way. A machine can happily answer ping while the exact port you
need stays closed, so this check catches problems the earlier ones miss.

## A layered method

Each tool above answers one layer. Run them in the order the layers stack, and the
first failure tells you where the problem lives. This mirrors
[how a web request works](/learn/networking/how-a-web-request-works/) — the same
steps, now used as a checklist:

1. **Does the name resolve?** `dig name` — if not, it's [DNS](/learn/networking/dns/).
2. **Can I reach the IP?** `ping` the address or trace the route — if not, it's the
   network path.
3. **Is the port open?** `nc -vz host port` — if not, the service is down or a
   firewall is blocking it.
4. **Does the app respond?** `curl` the URL — if the port is open but the app errors,
   the fault is in the service, not the network. (More on this in
   [making requests with curl](/learn/networking/curl-and-http-tools/).)

The power is in reading the *combination* of results:

- **Works by IP but not by name** → DNS. The path and service are fine; the name
  just isn't resolving.
- **No ping but the port works** → ICMP is blocked, not the host. The machine is up
  and serving; it's simply ignoring pings.

Two facts together say more than either alone, which is why working the layers in
order — and noting what passes as well as what fails — beats poking at random.

<div class="knowledge-check" data-quiz data-correct-msg="Right — the host is reachable, so the name just isn't resolving to it: a DNS problem." markdown="0">
  <p class="knowledge-check__q">Quick check: a site loads when you use its IP address but not its domain name. The most likely culprit?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">The service on the host has crashed</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">DNS (name resolution)</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">A firewall is blocking the port</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **ping** tests reachability and latency with ICMP echoes — but a blocked ping is
  "can't confirm," not "host down."
- **traceroute** shows the hops to the destination, so you can see **where** the
  path breaks or slows.
- **dig** / **nslookup** confirm [DNS](/learn/networking/dns/) is answering with the
  right address — rule this out first for name failures.
- A **port check** (`nc`/`telnet`) confirms the service is listening, which is
  separate from the host being up.
- Diagnose in **layers** — name, route, port, app — and read results together:
  "IP works, name doesn't" → DNS; "no ping, port works" → ICMP blocked.

Next up: [making requests with curl](/learn/networking/curl-and-http-tools/)
