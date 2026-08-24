---
layout: page
title: "Why Is a Helicopter Circling My Neighborhood?"
description: "A helicopter circling overhead usually means a police search, a medevac, news coverage, or survey work. How to identify exactly whose helicopter it is in under a minute with free flight trackers — and how to hear what it's doing."
keywords: helicopter circling my neighborhood, why is a helicopter circling, police helicopter near me, helicopter over my house, helicopter circling at night, what is that helicopter doing, track police helicopter
permalink: /helicopter-circling-my-neighborhood/
nav_group: Hardware
affiliate: true
faq:
  - q: "Why is a helicopter circling my neighborhood?"
    a: "Tight, repeated circles usually mean a police helicopter supporting a ground search — often for a fleeing suspect or a missing person. Other common explanations: a news helicopter holding over an incident, a medevac orbiting before landing, powerline/pipeline patrol flying slow straight lines, or survey and photography work flying a grid."
  - q: "How can I find out whose helicopter is above me?"
    a: "Open a flight tracker and look at your location. ADS-B Exchange (adsbexchange.com) is best for this because it doesn't honor blocking requests, so it shows most police and government aircraft that FlightRadar24 and FlightAware hide. Tap the aircraft for its registration, operator, and flight path — a search pattern is obvious on the map."
  - q: "Why can't I find the helicopter on FlightRadar24?"
    a: "Many law-enforcement and government operators enroll in the FAA's LADD blocking program, which commercial trackers honor — so the aircraft exists but is hidden there. ADS-B Exchange uses unfiltered volunteer-received data and usually still shows it. A helicopter absent even there may be military, or flying without ADS-B."
  - q: "What does a police helicopter circling at night with a spotlight mean?"
    a: "A search on the ground — typically for a suspect who ran from a stop or a missing/endangered person. The orbit keeps the camera and light on the search area while ground units set a perimeter. It normally ends within an hour. A scanner tuned to the police dispatch or air-support channel tells you which it is in real time."
  - q: "Can I listen to what the helicopter is saying?"
    a: "Often, yes. Police air units talk on the agency's dispatch or tactical talkgroups — audible on a scanner where unencrypted. Medevac and news helicopters also talk to air traffic control on VHF airband (118–137 MHz AM), which is never encrypted and legal to hear; any scanner or SDR receives it."
  - q: "Why do helicopters keep flying low over my house every day?"
    a: "A repeating low, slow, straight-line pattern is almost always utility work — powerline or pipeline inspection — or aerial survey/mapping flying a back-and-forth grid. Check the operator's name on a flight tracker: utility patrol operators are usually named survey or helicopter-services companies, not police."
---

# Why Is a Helicopter Circling My Neighborhood?

**You can usually identify the exact helicopter over your house — operator,
registration, and flight path — in under a minute with a free flight tracker,
and the flight pattern itself tells you what it's doing.** Tight orbits mean a
police search or news coverage; straight low lines mean utility patrol; a grid
means survey work. Here's how to tell, and how to hear it.

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Identify it:** open **[ADS-B Exchange](https://globe.adsbexchange.com/)** —
unlike FlightRadar24/FlightAware it ignores FAA LADD blocking, so police and
government aircraft usually appear. **Read the pattern:** tight orbit = search
or news; straight lines = powerline/pipeline patrol; lawnmower grid = survey.
**Hear it:** police air units use the agency's
[talkgroups](/reference/talkgroup/) (scanner, where
[unencrypted](/police-scanner-encryption/)); all aircraft talk to ATC on VHF
airband, which any [scanner](/best-aviation-scanner/) or
[SDR + GopherTrunk](/cheap-sdr-scanner/) receives. GopherTrunk also [decodes
ADS-B itself](/adsb.html) — run your own tracker.
</div>

## Step 1: identify it on a flight tracker

1. Open **[ADS-B Exchange](https://globe.adsbexchange.com/)** and center on
   your location.
2. Tap the rotorcraft icon near you: you'll get the registration (tail
   number), operator, altitude, and a trail of its recent path.
3. Look up the tail number's owner via the FAA registry link if the operator
   field is blank.

Why ADS-B Exchange first: FlightRadar24 and FlightAware honor the FAA's
**LADD** privacy-blocking list, which many police and government operators
join. ADS-B Exchange is built from unfiltered volunteer receivers and shows
them anyway. If the aircraft appears on none of the trackers, it's likely
military or not transmitting ADS-B — rare for civil helicopters, which have
been required to carry it in most controlled airspace since 2020.

## Step 2: read the flight pattern

| Pattern | Most likely |
|---|---|
| Tight repeated circles, low, at night, spotlight | Police search (suspect or missing person) |
| Wider steady orbit, often daytime | News helicopter over an incident, or police overwatch |
| One pass, lands nearby (hospital, highway, field) | Medevac / air ambulance |
| Low, slow, straight lines following a corridor | Powerline or pipeline patrol |
| Back-and-forth "lawnmower" grid | Aerial survey, mapping, photography |
| High orbit lasting hours | Often military or law-enforcement surveillance |

A police search orbit typically resolves within the hour — the helicopter
leaves when the perimeter collapses or the person is found.

## Step 3: hear what it's doing

- **Police air support** coordinates with ground units on the agency's
  dispatch or tactical talkgroups. Where those are unencrypted, a
  [scanner](/how-police-scanners-work/) or a
  [$30 SDR running GopherTrunk](/cheap-sdr-scanner/) carries the whole
  search live. A [county scanner feed](/listen-to-police-scanner-online/)
  works too, if the feed relays those channels.
- **Every aircraft** — police, medevac, news — also talks to air traffic
  control on **VHF airband (118–137 MHz, AM)**. Airband is never encrypted
  and always [legal to monitor](/police-scanner-legal/); see the
  [aviation scanner guide](/best-aviation-scanner/) and the
  [frequency reference](/scanner-frequencies/).
- **Run your own tracker:** the same RTL-SDR dongle that scans dispatch also
  receives 1090 MHz transponder signals directly — GopherTrunk has a built-in
  [ADS-B decoder](/adsb.html) with a live map, so you're not depending on
  anyone's coverage or filtering.

## Sirens on the ground too?

Helicopter plus converging sirens means an active ground incident — work
through the fast sources in [why are there sirens near
me](/sirens-near-me/) and [police activity near
me](/police-activity-near-me/). If it's the middle of the night and you want
answers tomorrow, a recording scanner answers
[what happened last night](/what-happened-near-me-last-night/) on demand.

{% include scanner-kit.html %}
