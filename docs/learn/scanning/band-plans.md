---
slug: band-plans
title: Band plans & where services live
description: The radio spectrum has a floor plan — a band plan that tells you where public safety, business, aviation, marine, amateur, and other services live, so you know which slice to search instead of hunting the whole dial.
keywords: band plan, radio spectrum, VHF low band, VHF high band, UHF, 700 800 MHz, public safety spectrum, aviation band, marine VHF, frequency allocation, where to scan
level: intermediate
status: full
prereq:
  - radioreference-database
faq:
  - q: What is a band plan?
    a: A band plan is the agreed-upon layout of the radio spectrum — which ranges of frequency are set aside for which kinds of service. Regulators allocate bands to public safety, aviation, marine, business, amateur radio, and more, so a signal's frequency alone already tells you a lot about what it probably is.
  - q: Do I need to memorise every frequency?
    a: No. You need the shape of the map, not every street. Knowing that aircraft live around 108–137 MHz in AM, that marine sits near 156–162 MHz, and that modern public-safety trunking clusters in the 700/800 MHz range lets you point your search at the right neighbourhood. The exact channels you look up as you go.
  - q: Are band plans the same everywhere?
    a: The broad strokes are similar worldwide because international coordination keeps aviation and marine consistent, but the details differ by country. Public-safety allocations especially vary from one nation to the next, so confirm your own country's plan rather than assuming a list from elsewhere applies unchanged.
---

# Band plans & where services live

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
The spectrum isn't a random jumble — it has a **floor plan**. A **band plan** allocates ranges
of frequency to services: **aviation**, **marine**, **business**, **amateur**, and the
**public-safety** systems most scanning is about. Knowing the map means a signal's **frequency
alone** already narrows down what it is, and it tells you which slice to
[search](/learn/scanning/searching-and-discovery/) instead of hunting the whole dial. You don't
memorise every channel — you learn the neighbourhoods, then look up the addresses.
</div>

A newcomer often imagines scanning as sweeping the entire radio spectrum looking for anything.
That's a slow way to work, because the spectrum is already organised. Regulators divide it into
**bands**, each set aside for particular kinds of use, and once you know which band a service
lives in, you can point your search straight at it. This lesson is the map: the major bands, who
lives there, and how the frequency of a signal is itself a clue.

## Why the spectrum has a plan

Radio waves don't respect borders, and two transmitters on the same frequency in the same place
interfere. To keep the airwaves usable, national regulators — the FCC in the United States, and
its equivalents elsewhere — coordinated internationally allocate slices of spectrum to specific
**services**. Aviation gets one band, marine another, public safety several, and so on.

For a scanner this planning is a gift. Because a fire department can't legally transmit in the
aviation band, the frequency you hear something on already rules out most of what it could be.
Frequency is your first and cheapest identification clue — long before you analyse the
[signal's shape](/learn/rf-sdr/signal-anatomy/), its neighbourhood tells you the likely tenant.

## The major bands, roughly

Scanning mostly lives in the VHF and UHF ranges, with a few notable bands within them. The exact
edges vary by country, but the character of each is broadly consistent:

| Rough range | Common name | Who lives there |
|-------------|-------------|-----------------|
| 30–50 MHz | VHF low band | Older public safety, some utilities, long-range in odd conditions |
| 108–137 MHz | Aircraft band | Aviation voice, **AM** mode, air traffic control |
| 137–174 MHz | VHF high band | Public safety, business, marine (156–162), weather |
| 225–400 MHz | Military air | Military aviation, AM |
| 400–512 MHz | UHF | Business, public safety, some trunking |
| 700 / 800 MHz | Public-safety trunking | Modern P25 and other trunked systems |

A few landmarks are worth carrying in your head: **aircraft is AM** in the 118–137 MHz range,
**marine VHF** clusters near 156–162 MHz, **weather radio** sits around 162.4–162.55 MHz, and
**modern public-safety trunking** has largely migrated to the **700 and 800 MHz** bands. Those
few anchors let you place most of what you'll hear.

## Frequency as an identification clue

Combine the band plan with a signal's basic characteristics and you can often name a transmission
before you decode a word of it. A carrier at 121.5 MHz in **AM** with the tinny sound of an
aircraft cockpit is almost certainly aviation. A brief **FM** exchange at 154 MHz is very likely
public safety or business. A wall of continuous data at 851 MHz is a strong candidate for a
[trunked control channel](/learn/digital-trunking/the-control-channel/).

None of this is proof on its own, but it dramatically shrinks the search. When you reach the
[identifying an unknown signal](/learn/scanning/identifying-unknown-signals/) lesson, the band
is the first column of evidence you'll use, alongside bandwidth and modulation.

## Where the interesting traffic clusters

Because services are grouped, so is the traffic worth scanning. Public-safety dispatch on older
conventional systems tends to sit in the VHF high band and the 450–470 MHz UHF range. The big
county and state trunked systems dominate 700/800 MHz. Business and utility users spread across
UHF. Aviation and marine each have their own tidy band. Knowing this lets you build a
[search](/learn/scanning/searching-and-discovery/) plan that spends time where the action is,
rather than sweeping quiet spectrum.

It also tells you where *not* to look. Vast stretches of spectrum are allocated to services a
scanner can't usefully hear — broadcast, cellular, satellite, radar — and skipping them keeps
your search efficient.

## Band plans change

Allocations are not frozen. Spectrum gets **rebanded**, reallocated, and repurposed as needs
shift — the migration of public safety to 700/800 MHz is a decades-long example still playing
out. A band plan you learned years ago may have moved services around, and a database entry can
lag a change on the air. Keep a current reference for your own country, and when a band seems
oddly quiet or busy, check whether its allocation has shifted.

<div class="knowledge-check" data-quiz data-correct-msg="Right — aircraft voice lives around 108–137 MHz and uses AM, unlike the FM used by most land-mobile services." markdown="0">
  <p class="knowledge-check__q">Quick check: you hear an AM voice transmission near 125 MHz. Which service is it most likely?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Marine VHF</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Aviation / air traffic control</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">An 800 MHz trunked system</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- A **band plan** allocates ranges of spectrum to services, so a signal's **frequency alone**
  is an identification clue.
- Scanning lives mostly in **VHF and UHF**, with landmark bands for **aviation** (AM),
  **marine**, **weather**, and modern **700/800 MHz trunking**.
- The band is the **first column of evidence** when identifying an unknown signal, alongside
  bandwidth and mode.
- Traffic **clusters** by service, so a good search plan spends time where the interesting
  systems live and skips what a scanner can't hear.
- Allocations **change** over time, so keep a current plan for your own country.

Next up: [search, scan &amp; discovery](/learn/scanning/searching-and-discovery/).
