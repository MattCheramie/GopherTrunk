---
layout: page
title: "Why Are Military Jets Flying Over My House?"
description: "Fighter jets low and loud over your neighborhood? The usual reasons — training routes, flyovers, intercepts, refueling tracks — how to identify military aircraft when trackers show nothing, and how to listen to military aviation legally."
keywords: military jets flying over my house, why are fighter jets flying over, low flying military aircraft today, military helicopters over my house, jets flying low and loud, military planes circling, military flyover today
permalink: /military-jets-flying-over-my-house/
affiliate: true
unlisted: true
search: false
faq:
  - q: "Why are military jets flying over my house?"
    a: "The routine explanations: a Military Training Route or Military Operations Area near you (low-level flying is scheduled, legal, and loud), traffic in and out of a base or Air National Guard field, air-refueling tracks (a tanker orbiting with receivers), an exercise, or a scheduled event flyover (game, memorial, air show). Rarely, a genuinely urgent one: an air-defense intercept of a small plane that violated restricted airspace."
  - q: "Why don't the jets show up on FlightRadar24?"
    a: "Tactical military aircraft usually don't broadcast the ADS-B signal consumer trackers rely on, or broadcast intermittently. ADS-B Exchange shows more military traffic than the commercial trackers (tankers, transports, and trainers often do broadcast), but an empty map does not mean an empty sky — fighters simply may not be transmitting."
  - q: "What was the extremely loud jet noise that shook the house?"
    a: "Either a low, fast pass — Military Training Routes are flown down to a few hundred feet, and the noise arrives suddenly — or afterburner departures from a nearby base, or in the extreme case a sonic boom from supersonic flight, which cracks like an explosion across a wide area. Repeated same-time-of-day noise points at scheduled base operations."
  - q: "Why do military jets keep circling the same area?"
    a: "Circling usually means an exercise in a Military Operations Area, a holding pattern, or air refueling — a big tanker flying a racetrack orbit while fighters join on it is the classic 'why are they circling for an hour' answer, and the tanker is often visible on ADS-B Exchange even when the fighters aren't."
  - q: "Can I listen to what military aircraft are saying?"
    a: "Often, yes, legally: military aviation uses UHF airband (225–400 MHz AM), which is unencrypted for routine traffic like refueling coordination and range control, plus civilian ATC frequencies when transiting. An RTL-SDR receives UHF milair; hobbyists pair it with ADS-B to identify and follow exercises. Encrypted tactical comms exist too — those stay closed, as always."
  - q: "Who do I complain to about constant military jet noise?"
    a: "Every base and Guard wing has a public affairs office with a noise-complaint process, and they do log them — noise abatement procedures exist and routes get adjusted. Include date, time, and direction. For persistent low-level activity, the FAA's regional office and your congressional office are the escalation path residents actually use."
---

# Why Are Military Jets Flying Over My House?

**Military jets over your neighborhood are almost always scheduled — training
routes, base traffic, refueling orbits, exercises, or an event flyover — and
the reason they're missing from your flight-tracker app is that fighters
often simply don't broadcast the signal those apps listen for.** Here's how
to identify what you can, infer what you can't, and — the part nobody
expects — how to legally listen in.

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Check [ADS-B Exchange](https://globe.adsbexchange.com/)** — it shows far
more military traffic than FlightRadar24 (tankers/transports usually
broadcast; fighters often don't — empty map ≠ empty sky). **Low + sudden +
repeated** = a Military Training Route or base pattern near you. **Circling
for an hour** = exercise or a refueling orbit — find the tanker's racetrack
on the tracker. **House-shaking crack** = possible
[sonic boom](/loud-boom-near-me/). **Listen legally:** military UHF airband
(225–400 MHz AM) is receivable on an RTL-SDR — routine traffic is
unencrypted. **Noise problem?** The base public affairs office logs
complaints, and it matters.
</div>

## The usual explanations

- **Military Training Routes (MTRs)** — charted low-level corridors flown
  fast and low on a schedule. If you're under one, you get the sudden
  ripping pass a few times a month, entirely routinely.
- **Base and Guard traffic** — pattern work, formation departures,
  same-time-of-day noise. Air National Guard fields hide inside many
  civilian airports.
- **Refueling tracks and MOAs** — the "circling forever" case: a tanker
  orbits a racetrack while fighters cycle on and off it. The tanker usually
  shows on ADS-B Exchange; its customers often don't.
- **Event flyovers** — stadium, memorial, air show; precisely timed, gone in
  a minute, announced locally beforehand.
- **The urgent rarity** — an air-defense intercept (fast jets converging on
  a small plane near restricted airspace). These make the news the same
  afternoon.

Helicopters at low level with no tracker return are frequently military
too — the same identification logic (and limits) as the
[helicopter page](/helicopter-circling-my-neighborhood/).

## Identify what's identifiable

1. **[ADS-B Exchange](https://globe.adsbexchange.com/)** — the unfiltered
   tracker: tankers, transports, trainers, and helicopters commonly
   broadcast; look for the racetrack orbit or the corridor-shaped track.
2. **Read absence correctly** — tactical jets may transmit nothing. A loud
   sky and an empty map *is itself* a military signature.
3. **The boom case** — a single house-shaking crack with no aircraft in
   sight belongs to the [loud boom checklist](/loud-boom-near-me/).
4. **News + base social media** — exercises and flyovers get announced;
   "military jets over [your metro] today" resolves the event-day cases.

## Find out if you live under military airspace

Ten minutes of lookup settles "is this going to keep happening":

- **Pull up a free VFR sectional chart** (skyvector.com shows the FAA
  charts) and find your house. **Grey lines labeled VR/IR** with numbers
  are Military Training Routes — IR routes run in any weather, and
  three-digit route numbers mean segments flown *below* 1,500 ft. A route
  over your roof means the low passes are scheduled and permanent.
- **Hatched boxes labeled "MOA"** are Military Operations Areas — the
  maneuvering/exercise airspace, usually starting a few thousand feet up:
  circling and formation work, less of the sudden low boom-and-gone.
- **Find the nearest base or Guard wing's public affairs page** — flying
  schedules, night-flying notices, and exercise announcements get posted
  there, and it's also where the noise-complaint process lives.
- **New-to-you noise with none of the above nearby** usually means a
  temporary exercise or a deployment cycle at a distant base — the
  announcements and local reporting catch up within days.

## The part hobbyists love: hearing it

Military aviation's routine coordination — refueling join-ups, range
control, transits through civilian airspace — runs on **UHF airband,
225–400 MHz AM, unencrypted**, plus regular civilian ATC. A cheap RTL-SDR
receives all of it (the same [dongles in the kit below](#kit)), and pairing
the audio with ADS-B Exchange turns "mystery jets" into a narrated exercise.
GopherTrunk's [ADS-B decoder](/adsb.html) runs your own local tracker off
the same hardware — no dependence on anyone's coverage or filtering. The
usual boundary applies here too: encrypted tactical comms are
[off-limits to everyone](/police-scanner-encryption/); the open traffic is
plenty.

{% include scanner-kit.html %}
