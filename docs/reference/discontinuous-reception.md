---
slug: discontinuous-reception
title: Discontinuous reception (DRX)
entry_type: concept
category: cellular
description: Discontinuous reception is a power-saving mechanism that lets an idle device sleep between scheduled paging occasions and wake briefly to check for a page, trading a little latency for large battery savings; extended DRX stretches the cycle to minutes for IoT.
keywords: DRX, discontinuous reception, eDRX, extended DRX, paging occasion, sleep cycle, battery saving, duty cycle, NB-IoT, LTE-M
aka: [DRX, Discontinuous reception, eDRX, extended DRX]
autolink: true
infobox:
  - { label: Type, value: Power-saving mechanism }
  - { label: Idea, value: Receiver duty-cycles between pages }
  - { label: Extended, value: eDRX (cycle up to minutes) }
see_also: [nb-iot, lte-m, 5g-nr, registration, lte]
cite_urls:
  - https://en.wikipedia.org/wiki/Discontinuous_reception
  - https://en.wikipedia.org/wiki/Paging_(telecommunications)
---

**Discontinuous reception** (**DRX**) is a power-saving mechanism in which an idle device's
receiver **sleeps** between scheduled *paging occasions* and **wakes** only briefly to check
whether the network is trying to reach it.[^wiki] By keeping the receiver off most of the
time, DRX trades a little latency — the network may have to wait for the next wake window to
deliver a page — for large battery savings. It is present in GSM, [LTE](/reference/lte/), and
[5G NR](/reference/5g-nr/), and its extended form (**eDRX**) stretches the cycle to minutes
for low-power IoT such as [NB-IoT](/reference/nb-iot/) and [LTE-M](/reference/lte-m/).

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 140" role="img" aria-label="A timeline of a receiver duty cycle showing long sleep stretches punctuated by short wake windows where the device listens for a paging message." xmlns="http://www.w3.org/2000/svg">
  <line x1="20" y1="100" x2="440" y2="100" stroke="currentColor" stroke-opacity="0.5"/>
  <text x="405" y="120" font-size="9" fill="currentColor">time →</text>
  <g stroke="currentColor" stroke-width="1.4" fill="none">
    <path d="M20 100 L60 100 L60 55 L78 55 L78 100 L170 100 L170 55 L188 55 L188 100 L280 100 L280 55 L298 55 L298 100 L390 100 L390 55 L408 55 L408 100 L440 100"/>
  </g>
  <g font-size="8" fill="currentColor" text-anchor="middle">
    <text x="115" y="115">sleep</text>
    <text x="225" y="115">sleep</text>
    <text x="335" y="115">sleep</text>
    <text x="69" y="48">wake</text>
    <text x="179" y="48">wake</text>
    <text x="289" y="48">wake</text>
    <text x="399" y="48">wake</text>
  </g>
  <text x="230" y="36" text-anchor="middle" font-size="9" fill="currentColor">short "listen for page" windows on a fixed cycle</text>
</svg>
<figcaption>DRX duty-cycles the receiver: it stays asleep through long stretches and powers up only at the agreed paging occasions to listen for a page, then sleeps again.</figcaption>
</figure>

## How it works

When a device registers with the network but has no active data, it enters an idle state and
negotiates a **DRX cycle** — a repeating period whose length both sides know. Within each
cycle the device computes, from its own identity, a specific *paging occasion*: a short
window in which it turns its receiver on and listens for a paging message addressed to it.
If nothing is there, it powers the receiver back down until the next cycle. Because the
network knows the device's schedule too, it holds any incoming page until that window and
transmits it then. The whole scheme rests on both ends agreeing, at
[registration](/reference/registration/) time, on the cycle length and how the paging
occasion is derived.

The trade-off is **latency for power**. A longer DRX cycle means the receiver is off a
greater fraction of the time — dramatically less energy — but an incoming call, message, or
data push can wait up to one cycle before the device notices it. Ordinary phones use cycles
of a fraction of a second to a couple of seconds, short enough that the added delay is
imperceptible while still cutting idle drain sharply. The receiver hardware, not the display,
is often the dominant idle consumer, so switching it off between pages is one of the largest
levers on standby battery life.

## Extended DRX and IoT

For devices that must run years on a small battery, standard DRX does not sleep long enough.
**Extended DRX (eDRX)** stretches the cycle far beyond the normal range — up to minutes, and
in some configurations longer — so an [NB-IoT](/reference/nb-iot/) or
[LTE-M](/reference/lte-m/) sensor can stay asleep for a very long interval and still be
reachable, at the cost of the network tolerating that much delay before a downlink reaches
it. A related mechanism, Power Saving Mode, goes further still by letting the device become
essentially unreachable between its own transmissions. Conceptually none of this is new:
it is the same **duty-cycling** that battery-powered paging receivers have always used —
listen briefly on a known schedule, sleep the rest of the time.

## Relevance to SDR

DRX is why a monitored idle cellular device is silent on the air most of the time and only
briefly active: its receiver, and any uplink it sends, follow the negotiated cycle rather
than a continuous stream. An observer sees traffic clustered around paging occasions rather
than spread evenly. GopherTrunk decodes land-mobile trunking rather than cellular paging, but
the same idea — a receiver that duty-cycles to catch a control-channel page and otherwise
stays idle — describes how battery-powered trunking subscribers conserve power while waiting
for a call, making DRX useful background for reasoning about intermittent RF activity.

## Sources

[^wiki]: [Discontinuous reception](https://en.wikipedia.org/wiki/Discontinuous_reception) — Wikipedia, for the DRX cycle, paging occasions, and the extended-DRX variant.
[^paging]: [Paging (telecommunications)](https://en.wikipedia.org/wiki/Paging_(telecommunications)) — Wikipedia, for how networks page idle devices and the duty-cycling that DRX builds on.
