---
layout: page
title: "SDR Filters: Fix Overload & Interference"
description: "How to fix RTL-SDR overload and interference with broadcast FM/AM notch filters, bandpass and SAW filters — when a filter beats an LNA for GopherTrunk decoding."
keywords: SDR filter, RTL-SDR overload, broadcast FM notch filter, AM band reject filter, SAW filter, bandpass filter SDR, FM interference, desensitization, GopherTrunk
permalink: /sdr-filters/
nav_group: Hardware
affiliate: true
faq:
  - q: "Why does my RTL-SDR hear noise or ghost signals everywhere?"
    a: "An 8-bit RTL-SDR has limited dynamic range. A strong nearby broadcast FM, AM, TV, or pager transmitter can overload its front end, raising the noise floor and creating false images across the band. A notch or bandpass filter removes the offending signal before it reaches the tuner and restores weak-signal reception."
  - q: "Do I need an FM notch or an AM notch filter?"
    a: "Use an FM broadcast notch (88–108 MHz) if you live near FM towers and scan VHF/UHF — it is the single most common fix. Add an AM broadcast notch (roughly below 1.7 MHz) if you monitor HF and a local AM station is swamping your upconverter or direct-sampling input."
  - q: "Should I buy a filter or an LNA first?"
    a: "If your problem is a raised noise floor, images, or intermod from strong local signals, buy the filter — an LNA would only amplify the interference and make overload worse. Buy an LNA only when a genuinely weak, distant signal needs a lift and the band is otherwise clean."
  - q: "What is a SAW filter?"
    a: "A surface-acoustic-wave (SAW) bandpass filter passes one narrow band (for example the 380–520 MHz public-safety range) and sharply rejects everything else. It is the strongest fix when you only care about one band and want maximum out-of-band rejection."
  - q: "Will a filter reduce my wanted signal?"
    a: "A little — every filter has some insertion loss in its passband, typically 1–3 dB. That is almost always a good trade: removing a strong interferer lowers the noise floor far more than the small loss costs you, so the net signal-to-noise ratio improves."
  - q: "Can a filter help with simulcast distortion?"
    a: "No. Simulcast distortion is multipath from multiple transmitters, not front-end overload. A filter cannot fix it. See our simulcast reference for what actually helps."
---

# SDR Filters: Fix Overload & Interference

**If your [RTL-SDR](/reference/rtl-sdr/) suddenly hears static, ghost signals, or a
raised noise floor across the whole band, the cause is almost always front-end
overload — and an [RF filter](/reference/rf-filter/) fixes it far better than any
amount of gain twiddling.** An 8-bit dongle has limited dynamic range, so one strong
local transmitter can swamp everything you actually want to decode with GopherTrunk.

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Most common fix:** a **[broadcast FM notch filter](https://www.amazon.com/dp/B07XKY8YKB?tag=gophertrunk-20)**
(~$25) kills 88–108 MHz overload. **On HF:** add an
**[AM broadcast notch](https://www.amazon.com/dp/B079CMB44V?tag=gophertrunk-20)** (~$20).
**Best out-of-band rejection:** a **[SAW bandpass filter](/reference/saw-filter/)** for
your one band. **Filter first, LNA later** — an [LNA](/best-sdr-lna/) amplifies
interference too. Filters cost 1–3 dB of insertion loss but can drop the noise floor
by 10 dB or more.
</div>

## Why strong signals wreck a cheap dongle

An [RTL-SDR](/reference/rtl-sdr/)'s 8-bit ADC gives it roughly 48 dB of usable
dynamic range. That is plenty when the band is quiet, but a
**broadcast FM station a few miles away can arrive 60–80 dB stronger** than the
distant P25 control channel you are trying to follow. When that happens the tuner's
amplifier is driven into compression — it stops behaving linearly — and three bad
things follow:

- **Desensitization.** The strong signal steals the front end's headroom, so weak
  signals across the *entire* band get quieter or vanish.
- **Intermodulation.** Two or more strong signals mix inside the overloaded tuner and
  produce phantom "signals" on frequencies where nothing is actually transmitting.
- **Images and a raised noise floor.** The whole spectrum display lifts, burying the
  low-level modulation your decoder needs.

None of this is a software problem, and none of it is fixed by more gain. The signal
you want is being drowned before it ever reaches the ADC. You have to remove the
offender in hardware, in front of the dongle — that is what a filter does.

## The three filter types

<div class="pick-cards" markdown="0">
<div class="pick-card pick-card--top">
<span class="pick-card__badge">Most common fix</span>
<h3>Broadcast FM notch</h3>
<p class="pick-card__price">around $25</p>
<p>Deep rejection of 88–108 MHz, passes everything else. The default fix for city dwellers scanning VHF/UHF near FM towers.</p>
<a class="btn btn--buy" href="https://www.amazon.com/dp/B07XKY8YKB?tag=gophertrunk-20" rel="nofollow sponsored noopener">FM notch on Amazon &rarr;</a>
<p class="pick-card__note"><a href="/reference/rf-filter/">RF filter details</a></p>
</div>
<div class="pick-card">
<span class="pick-card__badge">For HF users</span>
<h3>AM broadcast notch</h3>
<p class="pick-card__price">around $20</p>
<p>Rejects the AM broadcast band (below ~1.7 MHz) so a local mediumwave blowtorch stops swamping your HF input or upconverter.</p>
<a class="btn btn--buy" href="https://www.amazon.com/dp/B079CMB44V?tag=gophertrunk-20" rel="nofollow sponsored noopener">AM notch on Amazon &rarr;</a>
<p class="pick-card__note"><a href="/best-hf-sdr/">HF SDR guide</a></p>
</div>
<div class="pick-card">
<span class="pick-card__badge">Strongest rejection</span>
<h3>SAW bandpass</h3>
<p class="pick-card__price">around $25–40</p>
<p>Passes only one band (e.g. 380–520 MHz public safety) and rejects everything outside it. Best when you scan a single band.</p>
<a class="btn btn--buy" href="https://www.amazon.com/dp/B07G14Q6XX?tag=gophertrunk-20" rel="nofollow sponsored noopener">LNA + filter combos &rarr;</a>
<p class="pick-card__note"><a href="/reference/saw-filter/">SAW filter details</a></p>
</div>
</div>

**Notch (band-stop) filters** remove one band and pass everything else — ideal when a
single culprit (FM broadcast, AM broadcast, a nearby pager transmitter on 152/929 MHz)
is the problem but you still want wide coverage. **Bandpass filters** do the opposite:
they pass one band and reject the rest. **[SAW filters](/reference/saw-filter/)** are
bandpass filters built from a surface-acoustic-wave element, giving very steep skirts
and deep out-of-band rejection in a tiny package.

## When a filter beats an LNA

This is the decision people get wrong most often, so be honest about which problem you
have:

> **Filter vs LNA.** A **[filter](/reference/rf-filter/)** removes unwanted signal; an
> **[LNA](/best-sdr-lna/)** amplifies *all* signal. If your band is crowded with strong
> locals, an LNA makes overload *worse* — it pushes the interferer deeper into
> compression. Reach for the LNA only when the wanted signal is genuinely weak *and*
> the band is clean; reach for the filter whenever the noise floor is raised, you see
> images, or intermod products appear.

A useful order of operations for a stubborn setup:

1. **Turn gain down first.** Free. If lowering RTL-SDR gain cleans up the ghosts, you
   were overloading — a filter will let you run more gain cleanly.
2. **Add the right notch filter.** FM notch for VHF/UHF scanning; AM notch if you are
   on HF. This solves the large majority of city overload complaints.
3. **Add a SAW bandpass** if you only care about one band and want maximum rejection.
4. **Only then consider an LNA** — and put it *after* the filter (antenna → filter →
   LNA → dongle) so you never amplify the interference you just removed.

## Where filters live in the chain

Order matters. The general rule is **filter as early as possible, amplify as late as
possible**:

```
Antenna → (filter) → (LNA) → coax → RTL-SDR
```

Putting the filter ahead of any amplifier protects the whole chain from overload. If
you power an LNA from the dongle's [bias tee](/reference/bias-tee/), make sure the
filter you place before it passes DC or is placed on the antenna side of the LNA — a
DC-blocking filter between the bias tee and the LNA will starve it of power.

## Bottom line

If GopherTrunk was decoding fine and then went noisy — or never worked well in a city —
suspect **front-end overload before anything else**. A
**[broadcast FM notch filter](https://www.amazon.com/dp/B07XKY8YKB?tag=gophertrunk-20)**
is the ~$25 fix for the vast majority of VHF/UHF cases; add an
**[AM notch](https://www.amazon.com/dp/B079CMB44V?tag=gophertrunk-20)** if you work HF,
and a **[SAW bandpass](/reference/saw-filter/)** if you live on one band. Buy the filter
before the [LNA](/best-sdr-lna/): amplifying interference never helps. Then get the rest
of the kit right in our
[what-you-need checklist](/what-do-i-need-for-gophertrunk/) and pick the dongle in
[best SDR for GopherTrunk](/best-sdr-for-gophertrunk/).
