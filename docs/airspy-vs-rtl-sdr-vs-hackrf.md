---
layout: page
title: "Airspy vs RTL-SDR vs HackRF for Scanning"
description: "Airspy vs RTL-SDR vs HackRF compared for scanning P25/DMR/NXDN/TETRA with GopherTrunk — 8-bit vs 12-bit ADCs, capture bandwidth, dynamic range, TX, and which to buy by use case."
keywords: Airspy vs RTL-SDR, Airspy vs HackRF, RTL-SDR vs HackRF, best SDR for scanning, 12-bit SDR, Airspy R2, HackRF One, P25 SDR comparison
permalink: /airspy-vs-rtl-sdr-vs-hackrf/
nav_group: Hardware
affiliate: true
faq:
  - q: "Which is best for scanning: Airspy, RTL-SDR, or HackRF?"
    a: "For most people an RTL-SDR (around $35) is the best value and decodes P25/DMR/NXDN just as well as anything. Step up to a 12-bit Airspy (around $110–170) only when your RF is weak, congested, or you want to channelize many signals from one wideband capture. A HackRF is overkill for scanning — its extra reach and transmit are wasted on receive-only VHF/UHF."
  - q: "Is an Airspy really better than an RTL-SDR?"
    a: "In tough conditions, yes. The Airspy's 12-bit ADC has far more dynamic range than an 8-bit RTL-SDR, so it holds onto weak signals in the presence of strong ones without overloading, and its wider capture lets GopherTrunk channelize multiple control channels at once. On a clean, single-site system the RTL-SDR decodes the same voice."
  - q: "Should I buy a HackRF for GopherTrunk?"
    a: "Only if you also need its capabilities for other projects. The HackRF covers 1 MHz to 6 GHz and can transmit, but its 8-bit ADC has less dynamic range than an Airspy, and scanning uses none of the 6 GHz reach or TX. GopherTrunk uses it receive-only; it works but is more radio than scanning needs."
  - q: "Does more bits mean better audio?"
    a: "Not directly — decoded P25/DMR voice is either recovered or it is not. More ADC bits mean more dynamic range, so a 12-bit Airspy keeps decoding when a strong nearby signal would push an 8-bit RTL-SDR or HackRF into overload. In easy RF the audio is identical; the bits buy you resilience, not fidelity."
  - q: "Can any of them decode encrypted police?"
    a: "No. Airspy, RTL-SDR, and HackRF are all just receivers — none can decode AES-encrypted talkgroups, and neither can any scanner. GopherTrunk decodes clear P25/DMR/NXDN/TETRA only. Buy based on the systems still in the clear."
  - q: "Is the HackRF's transmit useful for scanning?"
    a: "No. Scanning is receive-only, and GopherTrunk never transmits. HackRF's TX is for other RF projects and legally needs a license. If you only want to scan, you are paying for a feature you will not use."
---

# Airspy vs RTL-SDR vs HackRF for Scanning

**For scanning with [GopherTrunk](/downloads.html), the honest ranking is:
[RTL-SDR](/reference/rtl-sdr/) for value, [Airspy](/reference/airspy/) for tough
RF, [HackRF](/reference/hackrf/) only if you also need its range or transmit.**
All three are USB [software-defined radios](/reference/software-defined-radio/)
GopherTrunk drives natively, but they are built for different jobs — and for
P25/DMR/NXDN trunk-tracking the cheapest one is usually right.

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**RTL-SDR (8-bit, ~$35):** the default — cheap, plenty for a clean single site.
**Airspy (12-bit, ~$110–170):** far more dynamic range and wider capture — the
pick for weak, busy, or multi-site RF. **HackRF (8-bit, 1 MHz–6 GHz + TX, ~$150):**
huge range and transmit, but **overkill for scanning**. **Bits buy dynamic range,
not audio fidelity.** **None decode [AES encryption](/police-scanner-encryption/).**
</div>

## Head-to-head

| | [RTL-SDR Blog V4](/reference/rtl-sdr/) | [Airspy R2 / Mini](/reference/airspy/) | [HackRF One](/reference/hackrf/) |
|---|---|---|---|
| ADC | 8-bit | **12-bit** | 8-bit |
| Frequency range | ~500 kHz–1.77 GHz | ~24 MHz–1.8 GHz | **1 MHz–6 GHz** |
| Capture bandwidth | ~2.4 MHz | ~6–10 MHz | **~20 MHz** |
| Dynamic range | Fair | **Best** | Fair |
| Transmit | No | No | **Yes** (licensed) |
| Best for | Value / single site | Tough / busy / multi-site | Range + TX projects |
| GopherTrunk uses | RX | RX | **RX only** |
| Approx price | ~$40 | ~$110–170 | ~$150 |

> **Bits are about dynamic range, not fidelity.** Decoded P25 voice is recovered
> or it is not — a 12-bit ADC does not make it "sound better." What the extra bits
> buy is headroom: an Airspy keeps decoding a weak control channel while a strong
> nearby pager or FM station would drive an 8-bit RTL-SDR or HackRF into
> [overload](/best-sdr-lna/). In easy RF, all three sound identical.

## When an RTL-SDR is all you need

Most people scanning one local [trunked system](/reference/trunked-radio/) never
need more than a [$35 RTL-SDR](/best-rtl-sdr/). One dongle follows one control
channel and its voice grants cleanly, it has a real TCXO so it stays locked, and
GopherTrunk decodes the same P25/DMR/NXDN/TETRA a radio ten times the price would.
If your target signal is reasonably strong and you are tracking a single site,
stop here — spend the savings on a better [antenna](/best-sdr-antenna/) instead.

## When to step up to an Airspy

The [Airspy](/reference/airspy/) earns its price in exactly two situations:

- **Weak or congested RF.** Its 12-bit front end pulls voice out of conditions
  that push an 8-bit dongle into overload — a distant control channel next to a
  strong local transmitter, or an intermod-heavy urban band.
- **Multi-site / wideband monitoring.** With up to ~10 MHz of capture (R2),
  GopherTrunk can channelize several control channels from one radio at once,
  instead of running a [pool of dongles](/multi-dongle-sdr-setup/).

If your problem is dynamic range or bandwidth, the Airspy is the right tool. If
it is simply low signal, an [LNA](/best-sdr-lna/) on any dongle may be the cheaper
fix first.

## When a HackRF makes sense (rarely, for scanning)

The [HackRF One](/reference/hackrf/) is a superb general-purpose SDR — 1 MHz to
6 GHz, ~20 MHz capture, and it can **transmit**. GopherTrunk drives it as a
receiver just fine. But for scanning specifically it is the wrong buy: its 8-bit
ADC gives it *less* dynamic range than an Airspy, its 6 GHz reach and transmit are
irrelevant to receive-only VHF/UHF decoding, and transmit legally needs a license.
Buy a HackRF if you already want it for RF experimentation, satellite work, or
protocol hacking — and enjoy that it scans too. Do not buy one *just* to scan.

> **GopherTrunk uses receive only.** Even on the HackRF, GopherTrunk never
> transmits. That is a feature you are paying for and will not use if scanning is
> your only goal.

## How to choose

- **Single local system, normal signal?** [RTL-SDR](/best-rtl-sdr/). Done.
- **Weak / busy / intermod-heavy RF?** [Airspy R2 or Mini](/reference/airspy/).
- **Many sites or control channels across a band?** [Airspy R2](/reference/airspy/)
  to channelize, or a [multi-dongle pool](/multi-dongle-sdr-setup/).
- **Want HF too?** An [Airspy HF+ Discovery](/reference/airspy-hf-plus/) or a
  V4's [upconverter](/reference/upconverter/).
- **Already own / want a HackRF?** GopherTrunk will use it — just know it is more
  radio than scanning needs.

## Where to buy

Airspy is sold through distributors rather than direct Amazon listings, so use a
search link; the HackRF and RTL-SDR have fixed listings.

<div class="pick-cards" markdown="0">
<div class="pick-card">
<span class="pick-card__badge">Best value</span>
<h3>RTL-SDR Blog V4</h3>
<p class="pick-card__price">around $40</p>
<a class="btn btn--buy" href="https://www.amazon.com/dp/B0CD745394?tag=gophertrunk-20" rel="nofollow sponsored noopener">RTL-SDR on Amazon &rarr;</a>
<p class="pick-card__note"><a href="/reference/rtl-sdr/">RTL-SDR details</a></p>
</div>
<div class="pick-card pick-card--top">
<span class="pick-card__badge">Best for tough RF</span>
<h3>Airspy R2 / Mini</h3>
<p class="pick-card__price">around $110–170</p>
<a class="btn btn--buy" href="https://www.amazon.com/s?k=Airspy+R2+SDR&tag=gophertrunk-20" rel="nofollow sponsored noopener">Airspy on Amazon &rarr;</a>
<p class="pick-card__note"><a href="/reference/airspy/">Airspy details</a></p>
</div>
<div class="pick-card">
<span class="pick-card__badge">Range + TX</span>
<h3>HackRF One (bundle)</h3>
<p class="pick-card__price">around $150</p>
<a class="btn btn--buy" href="https://www.amazon.com/dp/B0BKH7Z2NJ?tag=gophertrunk-20" rel="nofollow sponsored noopener">HackRF on Amazon &rarr;</a>
<p class="pick-card__note"><a href="/reference/hackrf/">HackRF details</a></p>
</div>
</div>

## Bottom line

Buy an **[RTL-SDR](/best-rtl-sdr/)** unless you have a reason not to — it decodes
the same digital voice for a third the price. Move to a **12-bit
[Airspy](/reference/airspy/)** when your RF is genuinely weak, busy, or
multi-site. Reach for a **[HackRF](/reference/hackrf/)** only if you also want its
6 GHz range or transmit for other projects. Whichever you pick, **no SDR decodes
[encryption](/police-scanner-encryption/)**, and the [software is free](/downloads.html).
For the full ranked lineup see the [best SDR for GopherTrunk](/best-sdr-for-gophertrunk/).
