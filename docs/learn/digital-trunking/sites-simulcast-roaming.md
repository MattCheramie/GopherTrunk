---
slug: sites-simulcast-roaming
title: "Sites, simulcast & roaming: multi-site systems"
description: How large trunked systems use many sites for coverage, why simulcast transmitters on the same frequency can distort reception, how voting and roaming work, and which control channel to monitor.
keywords: multi-site trunking, simulcast, voting receiver, roaming, registration, control channel per site, WACN, system ID, RFSS, site ID, P25, simulcast distortion
level: advanced
status: full
prereq:
  - control-channel-signaling
  - digital-modulation-for-trunking
faq:
  - q: What is simulcast?
    a: Simulcast is when several transmitters broadcast the same channel on the same frequency at the same time, carefully time-aligned, to blanket a wide area. It saves spectrum and gives seamless coverage. But a receiver located between two simulcast transmitters hears overlapping copies that can interfere, distorting the signal and making it hard to decode.
  - q: Why is simulcast hard to receive?
    a: Because a receiver midway between two simulcast transmitters gets two copies of the signal arriving at slightly different times and strengths. They combine into a distorted waveform whose constellation smears, even when both signals are strong. Moving the antenna, or using a linear modulation designed for simulcast, helps the copies combine more cleanly.
  - q: What is roaming on a trunked system?
    a: Roaming is a radio moving between sites in a multi-site system. As it travels, the radio re-registers and affiliates at each new site over that site's control channel, so the network knows where to deliver its calls. To a monitor, this means a radio may appear and disappear from different sites as it moves.
  - q: Which control channel should I monitor on a multi-site system?
    a: Each site has its own control channel, so you monitor the site whose coverage you're in and whose signal you receive most cleanly. The identifiers in the signaling — such as the site ID and system identifiers — confirm which site you're locked to. A neighbor that's stronger may be a better choice.
gophertrunk_links:
  - title: CC Activity
    url: /cc-activity.html
    note: read the site and system identifiers in the control-channel signaling.
  - title: Constellation
    url: /constellation.html
    note: spot the smeared constellation that simulcast distortion produces.
---

# Sites, simulcast & roaming: multi-site systems

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
Large trunked systems span many **sites** for coverage, and **each site has its own
control channel**. **Simulcast** broadcasts the same channel from several transmitters on
the **same frequency**, time-aligned — efficient, but a receiver *between* two simulcast
transmitters hears overlapping copies that **distort** the signal even when it's strong.
**Voting/receiver sites** handle the uplink; **roaming** radios **re-register** at each
site as they move. In P25, identifiers like **WACN**, **System ID**, **RFSS**, and **Site
ID** pin down exactly which site and system you're hearing — which guides *which control
channel to monitor*.

</div>

A real public-safety system rarely lives on one tower. It's a network of **sites** stitched
together, and that structure introduces three things you need to understand to monitor it
well: how sites relate, why **simulcast** can wreck reception, and how **roaming** works.

## Many sites, one system

Coverage over a county or a state requires more transmitters than a single site can
provide, so a large system is built from many **sites**, each covering an area. Critically,
**each site runs its own control channel**. A radio camps on the control channel of the
site it's currently in; as it moves, it switches to a neighbor's control channel.

For a monitor this has a direct consequence: **you choose a site**. You lock the control
channel of the site whose coverage you're in and whose signal you receive cleanly. The
[control-channel signaling](/learn/digital-trunking/control-channel-signaling/) you decode is *that site's* view of
the system, and its **adjacent-site broadcasts** tell you where the neighbors are.

## P25 identifiers: WACN, System ID, RFSS, Site

How do you know *which* site and system you've actually locked? The signaling carries a
hierarchy of identifiers. In P25 the key ones are:

| Identifier | Scope | Meaning |
|------------|-------|---------|
| WACN | Network | Wide Area Communications Network — the largest grouping |
| System ID | System | A specific system within the WACN |
| RFSS | Subsystem | RF Sub-System — a grouping of sites |
| Site ID | Site | The individual site (tower) you're hearing |

Together these uniquely place the control channel you're locked to within the larger
network. Reading them in **[CC Activity](/cc-activity.html)** is how you confirm you're on
the right system and site rather than a same-frequency neighbor.

## Simulcast and its distortion

Within a site, coverage is sometimes provided by **simulcast**: *several transmitters* all
broadcasting the **same channel on the same frequency at the same time**, precisely
time-aligned. This blankets a large area with one channel and is very spectrum-efficient.

The catch is for a receiver located **between** transmitters. It hears **overlapping
copies** of the signal arriving at slightly different times and strengths. Those copies
combine into a distorted waveform — even when every transmitter is strong, the
**constellation smears** and the decoder struggles. This is *simulcast distortion*, and
it's a signal-quality problem, not a weak-signal problem.

<figure class="figure" markdown="0">
<svg viewBox="0 0 540 200" role="img" aria-label="A diagram showing two simulcast transmitters broadcasting the same frequency, with a receiver located between them hearing two overlapping copies that combine into a distorted signal." xmlns="http://www.w3.org/2000/svg">
  <g font-size="10" fill="currentColor" text-anchor="middle">
    <rect x="40" y="40" width="90" height="30" rx="5" fill="currentColor" fill-opacity="0.18" stroke="currentColor" stroke-width="1.5"/>
    <text x="85" y="59">Site Tx A</text>
    <rect x="410" y="40" width="90" height="30" rx="5" fill="currentColor" fill-opacity="0.18" stroke="currentColor" stroke-width="1.5"/>
    <text x="455" y="59">Site Tx B</text>
    <text x="270" y="34" font-size="9">both on the same frequency, time-aligned</text>
    <circle cx="270" cy="150" r="10" fill="none" stroke="currentColor" stroke-width="1.5"/>
    <text x="270" y="180">receiver in the middle</text>
  </g>
  <line x1="120" y1="70" x2="262" y2="142" stroke="currentColor" stroke-width="1.5" stroke-dasharray="4 3"/>
  <line x1="420" y1="70" x2="278" y2="142" stroke="currentColor" stroke-width="1.5" stroke-dasharray="4 3"/>
  <text x="270" y="116" text-anchor="middle" font-size="9" fill="currentColor">two copies arrive slightly offset → distortion</text>
</svg>
<figcaption>Simulcast transmitters share a frequency to cover a wide area. A receiver between them hears overlapping copies that combine into a distorted waveform — the classic simulcast smear, visible on the constellation.</figcaption>
</figure>

This is exactly why systems sometimes use a **linear modulation** like
[CQPSK/LSM](/learn/digital-trunking/digital-modulation-for-trunking/) for simulcast: it lets the overlapping copies
combine more cleanly. When you can't lock a strong-but-smeared signal, the
**[Constellation panel](/constellation.html)** usually shows the simulcast fingerprint, and
moving the antenna even a little can help by changing which copy dominates.

## Voting and roaming

Two more pieces complete the multi-site picture.

The **uplink** (radios talking back to the system) is handled by **voting**: multiple
**receiver sites** listen for a radio, and the system "votes" for whichever copy is
cleanest, so a radio is heard well across the coverage area. You decode the **downlink**, so
voting mostly explains why the *infrastructure* hears mobiles so reliably.

**Roaming** is a radio moving between sites. As it travels out of one site's coverage and
into another's, it **re-registers and re-affiliates** on the new site's
[control channel](/learn/digital-trunking/control-channel-signaling/), so the network keeps routing its calls
correctly. To a monitor, a roaming radio appears to come and go across sites — and it's a
reminder that the *site you monitor* determines which radios and calls you see.

<div class="knowledge-check" data-quiz data-correct-msg="Right — overlapping same-frequency copies combine into a distorted waveform even when strong." markdown="0">
  <p class="knowledge-check__q">Quick check: why is a strong simulcast signal sometimes still hard to decode?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">The control channel is missing</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Overlapping same-frequency copies combine into a distorted waveform</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">The signal is too weak to detect</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- Large systems span many **sites**, and **each site has its own control channel** — you
  pick one to monitor.
- **Simulcast** shares a frequency across transmitters; a receiver between them hears
  **distorting** overlapping copies.
- A **linear modulation** (CQPSK/LSM) helps simulcast copies combine cleanly; the
  constellation reveals the smear.
- **Voting** receiver sites handle the uplink; **roaming** radios **re-register** at each
  new site.
- In P25, **WACN / System ID / RFSS / Site ID** pin down exactly which site and system
  you're hearing.

Next, the last piece of how trunking works: [encryption and
authentication](/learn/digital-trunking/encryption-and-authentication/) — and what a decoder can and cannot do
about them.
