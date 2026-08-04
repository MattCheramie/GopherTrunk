---
slug: what-you-can-hear
title: What you can (and can't) hear today
description: A realistic tour of the scannable spectrum — the analog and unencrypted digital traffic still wide open, and the encrypted systems that have gone dark. What's listenable in aviation, marine, rail, weather, business, amateur, and public safety today.
keywords: what can you hear on a scanner, encrypted police radio, scannable frequencies, analog vs digital scanning, aviation scanning, marine radio, railroad scanning, unencrypted traffic
level: beginner
status: full
prereq:
  - conventional-vs-trunked-recap
faq:
  - q: Can you still hear police on a scanner?
    a: It depends entirely on your area. Many agencies still run unencrypted digital or analog dispatch you can follow with a trunk-tracking scanner or SDR. A growing number, though, have encrypted some or all of their traffic, and encrypted voice is not decodable by any legal scanner or software — it simply plays as noise. Check a database like RadioReference for your county to see what is open before assuming either way.
  - q: What is always listenable regardless of encryption trends?
    a: "Large parts of the spectrum are effectively always open because encryption there is rare or impractical: civil aviation (air-band AM), marine VHF, railroad road channels, NOAA and other weather broadcasts, amateur radio, FRS/GMRS and business itinerant channels, and much utility and public-works traffic. These make a rich hobby on their own even where public safety has gone dark."
  - q: Why do some systems play as noise?
    a: Two reasons. Either the traffic is encrypted — scrambled so only radios with the key can recover the audio, which no scanner can defeat — or it is a digital mode your receiver cannot decode. Encryption is a wall; an unsupported-but-unencrypted digital mode is just a gap in your decoder that better software or hardware may close.
---

# What you can (and can't) hear today

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
Plenty of the spectrum is **wide open** — civil **aviation**, **marine**, **rail**,
**weather**, **amateur**, **business**, and much utility and even public-safety traffic
is analog or **unencrypted digital** you can follow. What you **cannot** hear is
**encrypted** voice: it plays as noise and no legal scanner or software defeats it. The
map varies enormously **by region and agency**, so check a
[database](/learn/scanning/radioreference-database/) for your area rather than assuming.
Encryption is a wall; an *unencrypted* mode your gear can't yet decode is just a gap.
</div>

You now know the [difference](/learn/scanning/conventional-vs-trunked-recap/) between a
channel you park on and a system you follow. The next honest question is: with all that
capability, what is actually *there* to listen to today? The answer is a
lot — but not everything, and the mix has shifted. This lesson is a realistic tour of
what is open, what has gone dark, and why.

## The two ways a signal can be closed to you

Before the tour, it helps to separate two very different reasons a signal might be
unlistenable, because people conflate them constantly:

- **Encryption.** The traffic is deliberately scrambled so only radios holding the key
  can recover the audio. To everyone else it is noise. **No scanner and no software —
  including GopherTrunk — can decode encrypted voice**, and attempting to defeat
  encryption is both futile and, in many places, illegal. This is a hard wall.
- **An unsupported mode.** The traffic is *not* encrypted, but it uses a digital
  protocol your particular receiver or software does not decode yet. This is a soft gap:
  better software, a firmware update, or different hardware may open it.

The distinction matters. "I hear noise" is not automatically "it's encrypted." It might
be an unencrypted mode you simply cannot decode with what you have — and knowing which is
which tells you whether to upgrade or to move on.

## What's wide open

Large swaths of the spectrum are, in practice, always listenable — because encryption
there is rare, impractical, or forbidden by the service's own rules:

- **Civil aviation.** The air band (around 118–137 MHz) is **AM** voice and, by
  international convention, unencrypted — air-traffic control, tower, ground, and
  en-route are all in the clear. A perennial favourite and utterly reliable.
- **Marine VHF.** Ship-to-ship, ship-to-shore, and harbour operations on marine VHF are
  open analog voice.
- **Railroads.** Road, yard, and dispatcher channels are typically analog (or
  unencrypted NXDN in some regions) and heavily followed by railfans.
- **Weather.** NOAA Weather Radio and equivalents broadcast continuously in the clear.
- **Amateur radio.** By its rules, amateur (ham) traffic **may not be encrypted**, so
  repeaters and simplex are always open — and often the most technically interesting
  listening.
- **FRS / GMRS / MURS and business itinerant.** Consumer and low-power business radios
  are mostly analog and open.
- **Utilities and public works.** Power, water, gas, roads, and transit crews are often
  still analog or unencrypted digital.

Any one of these could sustain the hobby indefinitely, even in a region where public
safety has locked up.

## Public safety: the shifting frontier

Police, fire, and EMS are the traffic most people first think of, and here the picture
genuinely varies. A great deal of public-safety radio is still **unencrypted** — analog
in smaller jurisdictions, or unencrypted **P25** (and DMR, NXDN, or TETRA depending on
the region) on trunked systems — and a trunk-tracking scanner or GopherTrunk follows it
fine.

But the trend is toward encryption. Some agencies encrypt only sensitive channels
(surveillance, tactical, records checks) while leaving dispatch in the clear; others
have encrypted **everything**. This is a policy decision made agency by agency, so two
neighbouring counties can be night and day. There is no universal answer to "can I hear
the police" — only the answer *for your specific area*, which you get from a
[database](/learn/scanning/radioreference-database/) listing, not from a general
assumption.

## Trunked and digital, but still open

It is worth stressing that **digital does not mean encrypted, and trunked does not mean
encrypted.** A modern P25 trunked system can be entirely in the clear; the digital voice
is a *format*, not a lock. GopherTrunk and trunk-tracking scanners exist precisely to
decode these unencrypted digital systems and follow their calls. So a region that has
"gone digital" or "gone trunked" has not necessarily gone dark — it has simply moved to
systems that need a capable receiver. The [RF & SDR module's
finding-systems](/learn/rf-sdr/finding-systems/) lesson covers spotting these on the
air.

## Reading your own area honestly

The practical upshot is that **what you can hear is local**. Before you buy gear or
write off a service, look your county or region up in a database and read what is
listed: which systems are conventional, which are trunked, which protocol each uses, and
crucially which talkgroups are flagged **encrypted**. That listing is the ground truth
for your location, and it will tell you whether a basic analog scanner is plenty, whether
you need trunk-tracking, or whether the interesting traffic has been encrypted and your
attention is better spent on aviation, rail, or the amateur bands. We make that database
the subject of its own lesson in
[RadioReference & system databases](/learn/scanning/radioreference-database/).

<div class="knowledge-check" data-quiz data-correct-msg="Right — encrypted voice cannot be decoded by any scanner or software, no matter how capable." markdown="0">
  <p class="knowledge-check__q">Quick check: which of these can no scanner or software recover?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">An unencrypted P25 trunked system in a digital mode</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Encrypted voice traffic scrambled with a key you don't have</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Analog FM dispatch on a conventional channel</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- Two reasons a signal is closed: **encryption** (a hard wall no scanner defeats) and an
  **unsupported mode** (a soft gap better gear may close) — do not confuse them.
- **Wide open** almost everywhere: civil **aviation** (AM air band), **marine**,
  **rail**, **weather**, **amateur** (encryption forbidden), **FRS/GMRS**, and much
  **utility** traffic.
- **Public safety varies by agency** — much is still unencrypted analog or digital, but
  a growing share is encrypted, so the answer is always *local*.
- **Digital and trunked do not mean encrypted** — GopherTrunk and trunk-tracking
  scanners follow unencrypted digital systems fine.
- **Check a database for your area** to learn what is open before buying gear or giving
  up on a service.

Next up: [Legal & ethical scanning](/learn/scanning/scanning-legal-and-ethical/).
