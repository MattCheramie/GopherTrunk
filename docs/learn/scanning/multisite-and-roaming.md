---
slug: multisite-and-roaming
title: Multisite, simulcast & roaming
description: Big trunked systems have many sites — how to pick the right one to monitor from where you sit, what simulcast does to your received signal and why it can wreck a decode, and how to follow units as they roam from site to site.
keywords: multisite trunking, simulcast, simulcast distortion, roaming, trunked sites, wide-area system, site selection, control channel per site, directional antenna, GopherTrunk site
level: advanced
status: full
prereq:
  - following-a-call
faq:
  - q: Why does a big trunked system have multiple sites?
    a: To cover a wide area. A single transmitter site reaches only so far, so a county or statewide system chains many sites together, each covering its own patch, and radios roam between them. Each site has its own control channel and its own set of voice channels, so from a listener's seat a multisite system is really several local systems sharing one identity.
  - q: What is simulcast and why is it a problem for scanning?
    a: Simulcast is when several transmitters broadcast the same signal on the same frequency at once to blanket an area. Where their coverage overlaps, the copies arrive slightly out of step and interfere, distorting the signal. A strong meter with a fuzzy, un-lockable signal in a simulcast overlap zone is a classic decode killer.
  - q: How do I follow a unit that roams between sites?
    a: You generally monitor the site you can hear best, and you'll follow any talkgroup active on that site. A unit that physically moves to another site drops off yours and appears on the one it moved to — so following a roaming unit across a wide area can mean monitoring the site it's currently in range of, not chasing it site to site.
---

# Multisite, simulcast & roaming

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
Wide-area trunked systems chain many **sites** together, each with its own
[control channel](/learn/digital-trunking/the-control-channel/) and voice channels — so from your
chair a multisite system is really several local systems sharing one identity. Monitor the **site
you hear best**. **Simulcast** — several transmitters sending the same signal on the same frequency
to blanket an area — can **distort** the signal where their coverage overlaps, a classic decode
killer that a strong meter won't warn you about. And **roaming** units move between sites, so you
follow the one in range rather than chasing a unit across the map. The system view is in
[sites, simulcast &amp; roaming](/learn/digital-trunking/sites-simulcast-roaming/).
</div>

A small trunked system lives at one site, and following it is straightforward. Big systems — county,
regional, statewide — are different animals: dozens of sites, overlapping coverage, and units that
move between them. This lesson is about operating in that world from a fixed listening post: which
site to point at, why simulcast can defeat an otherwise strong signal, and what "following a
roaming unit" really means when you can't be everywhere at once.

## A system is really many sites

To cover more ground than one transmitter can reach, a wide-area system deploys many **sites**, each
blanketing its own area and handing off to the next. Crucially, **each site has its own control
channel and its own pool of voice channels**. The talkgroups and system identity are shared across
the whole system, but the frequencies are per-site.

For a listener this has a concrete consequence: you monitor **one site at a time** — whichever one
you can hear from where you sit. When you [programmed the system](/learn/scanning/programming-a-trunked-system/)
you may have entered several sites' control channels; in practice your receiver follows the site
whose control channel it's locked to. A call on a *different* site, out of your range, simply isn't
receivable from your location, no matter how it's listed.

## Picking the site to monitor

Site selection is mostly a question of signal. Point your receiver at the control channel of the
site that comes in strongest and cleanest — usually the nearest one — and follow that. If you're
between two sites and both are weak, a **directional antenna** aimed at one can lift it clear of the
other and give you a solid lock on the site you actually want.

The trade-off is coverage: the strongest site carries the traffic *in your area*, which is usually
what you want to hear anyway. Chasing a distant site's traffic from a weak signal rarely pays off —
better to monitor the site you're genuinely in range of and accept that a wide-area system's far
corners belong to listeners who live there.

## Simulcast — a strong signal that won't decode

**Simulcast** is a coverage technique where several transmitters send the **same signal on the same
frequency at the same time** to blanket a large area from multiple towers. Inside any one
transmitter's coverage it works beautifully. But where two transmitters' coverage **overlaps**, your
receiver hears two copies of the signal arriving *slightly out of step*, and they interfere — a
self-inflicted multipath that smears the digital constellation.

The signature is distinctive and counter-intuitive: a **strong signal meter** but a **fuzzy,
un-lockable signal**. Newcomers raise the gain, which doesn't help, because the problem isn't weak
signal — it's distortion. The fixes are about favouring **one** transmitter: a **directional
antenna** to reject the competing copy, sometimes **lowering gain**, and choosing a listening spot
that isn't deep in an overlap zone. This is exactly the simulcast entry on the
[troubleshooting checklist](/learn/digital-trunking/troubleshooting-a-decode/), and it's one of the
few decode problems that punishes a *strong* signal.

## Roaming — following units that move

Radios on a wide-area system **roam**: as a unit drives out of one site's coverage, it registers on
the next and its calls now appear there. From a single fixed post you can't follow a unit physically
across the whole system, because a call on a site you can't hear is unreceivable. What you *can* do
is monitor the site the unit is currently in range of — which, if it's the site near you, is often
exactly the traffic you care about.

For genuinely wide-area following, listeners sometimes monitor several sites (with multiple receivers,
or by choosing the busiest site), but the honest picture is that one post hears one site's worth of
the system at a time. Understanding this keeps expectations realistic: you're not losing calls to a
fault when a roaming unit "disappears" — it's simply moved to a site beyond your antenna.

## Operating a multisite system well

Put it together and a practical multisite strategy emerges. Identify the site you hear best and lock
its control channel. If simulcast distortion appears — strong meter, fuzzy lock — reach for a
directional antenna and back the gain down rather than up. Accept that you monitor one site's slice
of the system, and pick the site that carries the traffic you want. Note in your
[records](/learn/scanning/frequency-records/) which site works best from your location, because for a
fixed post that answer rarely changes — and it saves you rediscovering it every session.

<div class="knowledge-check" data-quiz data-correct-msg="Right — simulcast overlap makes a strong signal arrive as interfering copies, so it reads strong but won't lock; favour one transmitter with a directional antenna and less gain." markdown="0">
  <p class="knowledge-check__q">Quick check: your meter is strong but the signal is fuzzy and won't lock, in an area covered by several towers. Most likely?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">The signal is too weak — raise the gain</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Simulcast distortion from overlapping transmitters — favour one with a directional antenna</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">The talkgroup is encrypted</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- A wide-area system is **many sites**, each with its own **control channel and voice channels**;
  you monitor **one site at a time** — the one you hear best.
- **Pick the site** by signal — nearest and cleanest — and use a **directional antenna** when two
  sites compete.
- **Simulcast** distorts the signal where transmitter coverage overlaps, giving a **strong meter but
  a fuzzy, un-lockable signal** — favour one transmitter, don't raise gain.
- **Roaming** units move between sites, so from a fixed post you follow the site in range, not the
  unit across the whole map.
- Note the best site for your location in your **records** — for a fixed post the answer is stable.

Next up: [when decoding fails](/learn/scanning/when-decoding-fails/).
