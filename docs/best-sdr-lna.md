---
layout: page
title: "Best LNA for SDR (and When You Need One)"
description: "When an SDR LNA helps and when it hurts — the RTL-SDR Blog Wideband LNA, bias-tee powering, recovering feedline loss, and why an amplifier can overload your receiver in strong-signal areas."
keywords: best LNA for SDR, SDR LNA, RTL-SDR LNA, low noise amplifier, bias tee LNA, mast-mounted preamp, SDR overload, when do I need an LNA
permalink: /best-sdr-lna/
nav_group: Hardware
affiliate: true
faq:
  - q: "What is the best LNA for an SDR?"
    a: "The RTL-SDR Blog Wideband LNA (around $30) is the standard pick — it is a low-noise, bias-tee-powered amplifier that mounts at the antenna and works across the VHF/UHF bands GopherTrunk scans. It powers over the coax from any SDR with a bias tee, so there is no separate power wire up the mast."
  - q: "Do I actually need an LNA?"
    a: "Only in specific cases: a weak signal, or a long feedline that loses signal before it reaches the receiver. If your control channel already decodes cleanly, an LNA adds nothing and can make things worse by overloading the front end. Fix the antenna and shorten the coax first."
  - q: "Can an LNA make reception worse?"
    a: "Yes. In a strong-signal area an LNA amplifies everything — including nearby broadcast FM, pagers, and cellular — and can drive the SDR into overload and intermodulation, burying the signal you want. That is the overload trap: more gain is not always better. Pair an LNA with a filter, or skip it."
  - q: "Where should an LNA go — at the antenna or the radio?"
    a: "At the antenna. An LNA works by setting a low noise figure before the feedline loss, so mounting it at the mast recovers coax loss that would otherwise be unrecoverable. Putting it at the radio end, after the loss, gives almost none of the benefit."
  - q: "What is a bias tee and do I need one to power the LNA?"
    a: "A bias tee injects DC power onto the coax so it reaches a mast-mounted LNA without a separate wire. Many SDRs — the RTL-SDR Blog V3/V4 and NESDR SMArTee — have a switchable or always-on bias tee built in. If yours does not, you need an external bias-tee injector."
  - q: "Will an LNA help me hear encrypted channels?"
    a: "No. An LNA only improves weak-signal reception — it cannot decode anything. No amplifier, SDR, or scanner can decode AES-encrypted talkgroups. GopherTrunk decodes clear P25/DMR/NXDN/TETRA only, and an LNA just helps a weak clear signal decode."
---

# Best LNA for SDR (and When You Need One)

**An [LNA](/reference/low-noise-amplifier/) is a scalpel, not a volume knob** — it
rescues a genuinely weak or feedline-starved signal, and it *wrecks* a setup that
is already awash in strong RF. Before you add gain to your
[SDR](/best-sdr-for-gophertrunk/), be honest about which situation you are in,
because for a lot of urban users the right amount of amplification is *none*.

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Best pick:** [RTL-SDR Blog Wideband LNA](/reference/low-noise-amplifier/) (~$30),
[bias-tee](/reference/bias-tee/) powered. **Use it for:** weak signals and long
feedlines. **Mount it at the antenna**, not the radio. **The overload trap:** in
strong-signal areas an LNA makes things *worse* — pair it with a
[filter](/sdr-filters/) or skip it. **Fix the [antenna](/best-sdr-antenna/) and
coax first.** **An LNA never decodes [encryption](/police-scanner-encryption/).**
</div>

## The pick

<div class="pick-cards" markdown="0">
<div class="pick-card pick-card--top">
<span class="pick-card__badge">Best overall</span>
<h3>RTL-SDR Blog Wideband LNA</h3>
<p class="pick-card__price">around $30</p>
<p>Low noise figure across VHF/UHF, powered up the coax by a bias tee — no separate mast wire. Mount it at the antenna to recover feedline loss cleanly.</p>
<a class="btn btn--buy" href="https://www.amazon.com/dp/B07G14Q6XX?tag=gophertrunk-20" rel="nofollow sponsored noopener">LNA on Amazon &rarr;</a>
<p class="pick-card__note"><a href="/reference/low-noise-amplifier/">LNA details</a> · <a href="/reference/bias-tee/">bias tee</a></p>
</div>
</div>

## What an LNA actually does

A low-noise amplifier sets the **noise figure** of your receive chain right at the
front, before the signal is degraded by cable and by the SDR's own noisier input.
Placed correctly, it lets a weak signal survive the trip down the coax and arrive
strong enough to decode. What it does **not** do is add information — it cannot
pull a signal out of thin air, and it amplifies noise and interference right along
with your target.

## When you genuinely need one

- **Long feedline.** Every foot of [coax](/reference/coaxial-cable/) loses signal,
  worse at UHF. A mast-mounted LNA recovers that loss *before* it happens, which
  is why placement matters (see below).
- **A truly weak signal.** A distant control channel that decodes intermittently,
  where the antenna is already as good and as high as you can get it, is the
  textbook case for a preamp.
- **Feeding a splitter or multiple radios.** Passive splits cost signal; an LNA
  ahead of the split restores headroom.

> **Mount it at the antenna, not the radio.** An LNA only delivers its low noise
> figure if it comes *before* the feedline loss. At the mast it recovers coax
> loss; bolted to the SDR at the bottom of a long run, after the loss, it buys you
> almost nothing.

## The overload trap

This is the part the marketing skips: **more gain is often worse.** An LNA
amplifies *everything* the antenna receives — including strong local broadcast FM,
pagers, cellular, and TV. In a signal-rich urban environment, that extra gain can
push the [SDR](/reference/rtl-sdr/)'s front end into **overload and
intermodulation**, generating spurious mixing products that bury the weak channel
you were trying to hear. An 8-bit RTL-SDR is especially prone to this because its
[dynamic range](/airspy-vs-rtl-sdr-vs-hackrf/) is limited to start with.

The symptom is telling: adding the LNA makes a marginal channel *disappear*, or
fills the waterfall with ghost signals that move when you change gain. If that
happens, the answer is not more amplification — it is **less**, plus a
[filter](/sdr-filters/) to knock down the strong out-of-band signals that are
causing the overload.

> **LNA + filter, not LNA alone.** In strong-signal areas the winning combination
> is a band or notch [filter](/sdr-filters/) *ahead of* the LNA, so the amplifier
> only boosts the band you care about. A bare LNA in a hot RF environment usually
> hurts.

## Powering it: the bias tee

The [RTL-SDR Blog LNA](/reference/low-noise-amplifier/) is powered by a
[bias tee](/reference/bias-tee/) — DC injected onto the coax so it reaches the
mast without a second wire. Many SDRs have this built in: the
[RTL-SDR Blog V3/V4](/rtl-sdr-blog-v3-vs-v4/) has a switchable bias tee, and the
[NESDR SMArTee v2](/reference/nesdr/) has an always-on 4.5 V one. If your dongle
lacks a bias tee, you add an external injector between the SDR and the coax. Turn
the bias tee **on** only when a device up the line actually needs the power —
feeding DC into a plain antenna does nothing useful and, on a short, could stress
the port.

## Do this before buying an LNA

Most "I need an amplifier" problems are really antenna or cable problems:

1. **Improve the [antenna](/best-sdr-antenna/) and its placement** — outdoors,
   higher, resonant on your band. This helps far more than a preamp.
2. **Shorten and upgrade the [coax](/sdr-cables-and-connectors/)** — a shorter,
   lower-loss run removes the very loss an LNA would be recovering.
3. **Set the SDR's own gain correctly** — too much gain overloads before any LNA
   is involved.

Only if the signal is *still* weak after all three, and your feedline is genuinely
long, does an LNA earn its place.

## Bottom line

Buy the **[RTL-SDR Blog Wideband LNA](/reference/low-noise-amplifier/)** if you
have a weak signal or a long feedline and you have already sorted the
[antenna](/best-sdr-antenna/) and [coax](/sdr-cables-and-connectors/). Mount it
**at the antenna**, power it over a [bias tee](/reference/bias-tee/), and in any
strong-signal area pair it with a [filter](/sdr-filters/) so you do not fall into
the overload trap. Used wrongly, an LNA makes reception *worse* — used in the
right situation, it is the difference between a weak channel that drops and one
that locks. And like everything else, it never decodes
[encryption](/police-scanner-encryption/). Then
[download GopherTrunk](/downloads.html) and get scanning.
