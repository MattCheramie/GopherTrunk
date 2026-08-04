---
slug: choosing-a-scanner
title: Choosing a scanner or SDR
description: How to match a receiver to what you actually want to hear — the bands it must cover, the trunking and digital modes your local systems use, and your budget — so you buy the right scanner or SDR the first time instead of the wrong one twice.
keywords: choosing a scanner, best scanner for beginners, digital trunking scanner, which SDR to buy, RTL-SDR Airspy, scanner buying guide, P25 scanner, band coverage
level: beginner
status: full
prereq:
  - scanners-vs-sdr
faq:
  - q: How do I choose the right scanner?
    a: Work backwards from what you want to hear. Look up your local systems in a database, note which bands they use and whether they are conventional, trunked, or digital and in which mode. Then buy a receiver that covers those bands and supports those modes. The most common beginner mistake is buying on price or brand first and discovering the local system uses a mode the receiver cannot follow.
  - q: What's the cheapest way to start?
    a: An RTL-SDR dongle plus a computer you already own is the lowest-cost entry, and with software like GopherTrunk it can follow unencrypted digital trunked systems. If you would rather have a self-contained box, entry-level analog scanners are inexpensive but cannot follow digital trunking — check that your local traffic is analog before choosing one, or you will need to upgrade.
  - q: Do I need a digital scanner?
    a: Only if the systems you care about are digital. Many areas still have plenty of analog traffic, and an analog scanner covers it cheaply. But if your local public safety runs P25, DMR, NXDN, or TETRA, you need a digital-capable receiver — a digital trunk-tracking scanner or an SDR with software that decodes that mode. The database listing for your area tells you which.
---

# Choosing a scanner or SDR

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
Choose by **working backwards from what you want to hear**, not from price or brand.
Look up your local systems, note the **bands** they use and whether they are
**conventional, trunked, or digital** (and in which mode — P25, DMR, NXDN, TETRA), then
buy a receiver that **covers those bands and decodes those modes**. The classic mistake
is buying first and discovering the local system uses a mode your radio can't follow.
For SDR, [front-end quality](/learn/rf-sdr/front-end-and-overload/) matters as much as
raw specs.
</div>

The [previous lesson](/learn/scanning/scanners-vs-sdr/) framed the two roads — a
dedicated scanner or an SDR with software. This one helps you pick a specific receiver
without wasting money. The single most important idea is that the *right* choice is
defined by *your local spectrum*, so the process starts with research, not shopping.

## Start with what you want to hear

Before you look at a single product, answer this: **what systems do you actually want to
follow?** Open a database like [RadioReference](/learn/scanning/radioreference-database/)
for your county or region and read the entries for the agencies and services you care
about. For each, note three things:

1. **The band(s)** — VHF low, VHF high, UHF, 700/800 MHz, and so on.
2. **Conventional or trunked** — and if trunked, the **system type**.
3. **Analog or digital** — and if digital, the **mode** (P25 Phase 1/2, DMR, NXDN,
   TETRA).

Those three facts are your **requirements list**. Every candidate receiver either meets
them or it doesn't, and that turns a bewildering market into a simple filter. Buying
before you have this list is how people end up with a scanner that hears everything
except the one system they bought it for.

## Match the bands

A receiver only hears the frequency ranges it is built to cover. Most modern scanners and
general-coverage SDRs span the common scanning bands (roughly VHF through 800 MHz), but
**check** — some cheap or specialised receivers have gaps, and a few services live in
unusual ranges. If your target system is on 800 MHz, a receiver that stops at 512 MHz is
useless for it no matter how good it otherwise is. Match coverage to the bands on your
requirements list first; it is a hard yes/no.

## Match trunking and digital support

This is where most upgrades — and most regrets — happen. Two capabilities are separate,
and you may need both:

- **Trunk-tracking.** To follow a *trunked* system the receiver must decode its
  **[control channel](/learn/digital-trunking/the-control-channel/)** and chase grants.
  An analog-only or conventional-only scanner cannot do this, period.
- **Digital decoding.** To hear *digital voice* the receiver must decode the specific
  **mode**. "Digital" is not one thing — a P25 scanner does not automatically do DMR or
  NXDN, and a receiver that does P25 Phase 1 may not do Phase 2.

So a modern public-safety system that is *P25 Phase 2 trunked* demands a receiver that
does **both** P25 Phase 2 and trunk-tracking. On the SDR side, GopherTrunk supplies the
trunk-tracking and multi-protocol decoding in software, so a plain wideband dongle can
follow systems a fixed analog scanner never could — one of the strongest arguments for
the SDR road when your local systems are digital and trunked.

## For SDRs, mind the front end

If you go the SDR route, resist judging a receiver on headline numbers alone. The
**front end** — how well the receiver handles strong signals without distorting — matters
enormously in the real world, especially in a city crowded with broadcast, cellular, and
pager transmitters. A cheap dongle can **[overload](/learn/rf-sdr/front-end-and-overload/)**
in that environment, producing phantom signals and desensitisation. The very cheapest
RTL-SDR dongles are a superb, low-risk way to learn, and often enough; stepping up to a
better-filtered SDR buys headroom in tough RF conditions. Either way, factor a decent
**[antenna](/learn/scanning/antennas-for-scanning/)** into the budget — it does more for
what you hear than a receiver upgrade of the same money.

## Match the budget honestly

Money spent in the wrong place is the real waste, so spend it in this order:

- **Meet the requirements first.** A receiver that cannot decode your local mode is not a
  bargain at any price — it simply doesn't do the job.
- **Then invest in the antenna and feedline.** These set the ceiling on everything and are
  usually the highest-value dollars in the whole station.
- **Then chase nice-to-haves.** Better ergonomics, a nicer display, a stronger SDR front
  end, more memory — real improvements, but only after the essentials are met.

Rough shape of the market: entry-level **analog** scanners are cheap but analog-only;
**digital trunk-tracking** scanners cost considerably more; an **RTL-SDR dongle** is the
least-cost entry to digital trunk-tracking *if you already have a computer*, with
better-filtered SDRs available as you outgrow it. There is no single "best" — only the
cheapest thing that satisfies your requirements list.

## Don't forget the ecosystem

A receiver is easier to live with when it plays well with the rest of the hobby:
database-driven programming (so you can load your county in a few clicks), an active
community, and — for SDRs — well-supported software. GopherTrunk, for instance, is built
to turn a supported SDR into a trunk-tracking scanner, so choosing hardware GopherTrunk
handles well saves you friction later. A slightly cheaper receiver that fights you at
every step is no bargain.

<div class="knowledge-check" data-quiz data-correct-msg="Right — the receiver must cover the band AND decode the exact trunking and digital mode your local system uses." markdown="0">
  <p class="knowledge-check__q">Quick check: your local public safety runs a P25 Phase 2 trunked system. What must your receiver do?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Only cover the 800 MHz band — any scanner that reaches it will work</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Cover the band and both trunk-track and decode P25 Phase 2 — coverage alone isn't enough</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Support encryption keys so it can unscramble the digital voice</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Work backwards from what you want to hear**: list each local system's **band**,
  **conventional/trunked**, and **analog/digital mode** — that's your requirements list.
- **Match coverage** first (a hard yes/no) and then **trunking + digital support** —
  "digital" is not one mode, and trunk-tracking is separate from decoding.
- For **SDRs**, weigh **front-end quality**, not just headline specs — cheap dongles can
  [overload](/learn/rf-sdr/front-end-and-overload/) in a noisy city.
- **Spend in order**: meet requirements, then the **antenna and feedline**, then
  nice-to-haves.
- Favour receivers with a good **ecosystem** — database programming, community, and
  software support like GopherTrunk.

Next up: [Antennas for scanning](/learn/scanning/antennas-for-scanning/).
