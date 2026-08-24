---
layout: page
title: "What Happened Near Me Last Night? How to Look It Up"
description: "Woke up to helicopter noise, sirens, or police tape and nothing on the news? How to reconstruct what happened near you last night — scanner archives, CAD logs, crime maps, blotters, records requests — and how to have your own answer next time."
keywords: what happened near me last night, sirens last night, police in my neighborhood last night, helicopter last night, find out what happened, police blotter lookup, scanner archives, crime near me yesterday
permalink: /what-happened-near-me-last-night/
nav_group: Hardware
faq:
  - q: "How do I find out what happened in my neighborhood last night?"
    a: "Work backward through the records the incident left: Broadcastify's audio archives of your county's scanner feed (paid tier) let you replay the dispatch traffic from the exact time; the agency's calls-for-service or CAD log lists the call type and block; crime maps like CityProtect and SpotCrime plot it once the report posts; and the department's blotter or press release covers anything major."
  - q: "Why is there nothing in the news about what happened on my street?"
    a: "News outlets cover a tiny fraction of emergency calls — typically fatalities, fires with damage, and major crimes. A medical call, a crash without serious injury, a mental-health call, or a fled suspect generates lights, sirens, even a helicopter, and zero coverage. Absence from the news says nothing about whether it happened."
  - q: "Can I replay police scanner audio from last night?"
    a: "Yes, two ways. Broadcastify archives its volunteer feeds — a premium subscription lets you pick your county's feed and a 30-minute window from the past days and listen. Or run your own recording scanner: software like GopherTrunk with an RTL-SDR records every call on a system with timestamps and talkgroup labels, so you scroll to 2:47 a.m. and press play — no subscription, no feed dependency."
  - q: "How long until last night's incident shows up on crime maps?"
    a: "Typically one to three days, sometimes a week — maps like CityProtect, Community Crime Map, and SpotCrime ingest incident reports after they're written and approved, not live calls. Calls that never become an incident report (medical calls, unfounded alarms, assists) may never appear at all."
  - q: "Can I request the police report for something that happened near me?"
    a: "Yes — incident reports are public records in every state, requested through the agency's records portal or a written public-records request. Expect fees, response times from days to weeks, and redactions or denial while an investigation is open. For a quick answer, the CAD log entry or blotter line is usually enough."
  - q: "What was the helicopter doing over my house at 3 a.m.?"
    a: "Flight-tracking history answers this precisely: ADS-B Exchange lets you replay the aircraft's actual track for free — identify the operator and watch the orbit. Pair the track's timestamps with scanner-feed archives from the same window and you usually get the whole story: what was searched for, where, and how it ended."
---

# What Happened Near Me Last Night? How to Look It Up

**Last night's sirens left records — dispatch audio, CAD log entries, flight
tracks, incident reports — and most of them are public. You can usually
reconstruct what happened near you in about fifteen minutes,** even when the
news never mentions it (and for the overwhelming majority of calls, it never
will). Here's the lookup, in the order that pays off fastest.

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Replay the radio:** Broadcastify **archives** (premium) → your county's
feed at the exact time. **Check the log:** agency **calls-for-service / CAD**
page → call type and block. **Helicopter?** Replay its track free on
[ADS-B Exchange](https://globe.adsbexchange.com/). **Later:** crime maps
(1–3+ days), blotter, records request. **Next time, skip all of it:** a
recording scanner — [$30 SDR + free GopherTrunk](/cheap-sdr-scanner/) —
archives every call itself; scroll to the timestamp and press play.
</div>

## 1. Replay the dispatch audio

If your county has a [Broadcastify feed](/listen-to-police-scanner-online/),
Broadcastify **archives it**: a premium subscription (a few dollars a month)
lets you pick the feed, the date, and a 30-minute window, and listen to the
dispatch traffic from exactly when you heard the commotion. You'll hear the
call go out, the units arrive, and how it resolved — the complete story,
in the responders' own words.

Limits are the same as [live feeds](/listen-to-police-scanner-online/): no
volunteer feed → no archive, and only the channels that feed relays.

## 2. Pull the call log

Agencies that publish **calls-for-service / CAD pages** usually keep the last
24 hours to 30 days browsable. Find yesterday's date, scan for your street,
and you get the call type ("disturbance," "structure fire," "traffic
collision") and block-level address — often enough to settle it. Search
`"<county> sheriff calls for service"` if you've never found yours.

## 3. Identify last night's helicopter

[ADS-B Exchange](https://globe.adsbexchange.com/) replays historical aircraft
tracks free: set your location and last night's time, and watch the orbit —
tail number, operator, altitude, and exactly which blocks it circled. (Why
ADS-B Exchange and not FlightRadar24, and what the patterns mean:
[helicopter circling my neighborhood](/helicopter-circling-my-neighborhood/).)
Cross-reference the track's timestamps with the scanner archive from step 1
and you usually have the entire incident.

## 4. The paper trail (slower, official)

- **Crime maps** — [CityProtect](https://www.cityprotect.com/),
  [Community Crime Map](https://communitycrimemap.com/), SpotCrime — plot
  incidents once reports post, typically 1–3+ days later. Calls that never
  generate a report never appear.
- **Blotter / press releases** on the agency site or local paper, for
  anything major.
- **Public-records request** for the incident report itself — every state
  allows it; expect fees, a wait, and redactions on open cases.

## Next time: have your own archive

Every step above depends on someone else having recorded, logged, or filed
something. Scanner hobbyists skip the whole exercise: software like
**[GopherTrunk](/getting-started.html)** with a **~$30 RTL-SDR** monitors your
local trunked system continuously and **records every call with timestamps
and [talkgroup](/reference/talkgroup/) labels** to a browser console. Sirens
at 2:47 a.m.? Scroll to 2:47, press play, go back to bed. No subscription, no
feed volunteer, no gaps — subject to the one universal limit,
[encryption](/police-scanner-encryption/), which no device can decode.

Start with [what hardware you need](/what-do-i-need-for-gophertrunk/), the
[budget breakdown](/cheap-sdr-scanner/), or a
[dedicated recording scanner](/best-police-scanners/) if you'd rather skip
the computer. And when it's happening *right now* instead of last night,
start here: [why are there sirens near me?](/sirens-near-me/)
