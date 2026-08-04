---
slug: scanning-legal-and-ethical
title: Legal & ethical scanning
description: The rules and etiquette that keep the scanning hobby healthy — what is legal to receive, what is not legal to do with it, why encryption is off-limits, and the responsible-monitoring habits that separate a good listener from a nuisance.
keywords: scanning legal, is scanning legal, scanner laws, encrypted radio law, receive-only legal, scanner ethics, responsible monitoring, rebroadcasting scanner audio
level: beginner
status: full
prereq:
  - what-you-can-hear
faq:
  - q: Is it legal to listen to a scanner?
    a: "In most of the United States and many other countries, receiving unencrypted radio for personal use is legal — the airwaves are open and a receiver is passive. But the details vary by jurisdiction: some places restrict scanners in vehicles, some regulate reception of certain services, and what you may lawfully do with what you hear is a separate question from whether you may hear it. Always check the law where you are."
  - q: Can I share or rebroadcast what I hear?
    a: Receiving a transmission and rebroadcasting or publishing its contents are different acts with different rules. Many jurisdictions restrict divulging or using the contents of radio communications, and even where sharing is legal it may be harmful — publishing names, addresses, or medical details from a call can hurt real people. Treat rebroadcasting as a deliberate decision with its own responsibilities, not an automatic right.
  - q: Why can't I just decrypt encrypted traffic?
    a: "Encrypted traffic is scrambled so only authorized radios can recover it, and defeating that encryption is generally illegal as well as technically impractical — no scanner or software, GopherTrunk included, decodes it. The ethical and legal stance is the same: encrypted means not for you. Leave it, and spend your time on the large open portion of the spectrum."
---

# Legal & ethical scanning

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
Scanning is **receive-only**, and in most places **receiving unencrypted radio for
personal use is legal** — but *what you may do with what you hear* is a separate
question, and both vary by jurisdiction. **Never try to defeat encryption** (illegal
and impossible), be careful about **rebroadcasting or publishing** what you hear, and
never **act on traffic in ways that interfere or cause harm**. The
[RF & SDR module's legal & ethical lesson](/learn/rf-sdr/legal-ethical/) covers the same
ground for SDR; this is the scanner-listener's version. Good etiquette is what keeps the
hobby open.
</div>

The [last lesson](/learn/scanning/what-you-can-hear/) mapped what is open and what is
closed. This one is about staying on the right side of both the law and good conduct —
the ground rules that let the hobby continue. None of this is legal advice, and the
specifics differ by country and even by state, so treat it as a framework and verify the
details where you live.

## Receiving is passive — and mostly legal

The foundational fact is that a scanner is a **receiver**. It emits nothing; it simply
detects radio waves that are already passing through the space you occupy. Because
reception is passive and the airwaves are a shared public resource, **receiving
unencrypted radio for personal use is legal in most of the United States and many other
countries.** You are not tapping a line or breaking into anything — the signal came to
you.

That said, "mostly legal" is not "always legal everywhere":

- Some jurisdictions **restrict scanners in vehicles**, or tie legality to intent (e.g.
  using a scanner in the commission of a crime).
- Some countries **regulate reception** of particular services more tightly than the US
  does.
- Some places distinguish between **owning**, **using**, and **using for a particular
  purpose**.

The safe habit is to look up the rules for **your** jurisdiction before you assume, the
same way you would check what is on the air locally.

## Receiving vs. using — two different questions

Here is the distinction that trips up newcomers: **being allowed to hear something is
not the same as being allowed to do anything you like with it.** Many legal systems draw
a line between *receiving* a communication and *divulging or using its contents*. You may
lawfully hear a call and still be restricted in publishing it, acting on it, or profiting
from it.

Practically, keep two questions separate in your mind:

1. **May I receive this?** (Usually yes, if it is unencrypted and you are listening
   personally.)
2. **May I share, publish, or act on it?** (Often more restricted — and even when legal,
   frequently unwise.)

Answering the first "yes" tells you nothing about the second.

## Encryption is off-limits, full stop

If a system is **encrypted**, it is not for you — legally, technically, and ethically.
Encrypted voice is scrambled so only radios with the key can recover it; to everyone else
it is noise, and **no scanner or software, GopherTrunk included, can decode it.**
Attempting to defeat encryption is generally **illegal** on top of being impractical, and
there is no grey area to explore. The agency chose to close that traffic; respect the
wall and move on to the [large open portion of the
spectrum](/learn/scanning/what-you-can-hear/).

## Don't interfere, don't cause harm

The receive-only nature of scanning also carries a duty: **do no harm with what you
hear.** A few concrete lines the community holds to:

- **Never transmit on, jam, or interfere with** the systems you monitor. A scanner
  cannot transmit, but the principle extends to never using other equipment to disrupt
  the traffic you listen to.
- **Never act on live public-safety traffic** in a way that impedes responders — showing
  up at an active scene, interfering with an operation, or beating first responders to a
  call is dangerous and can be criminal.
- **Never use monitored information to commit or aid a crime.** This is exactly the kind
  of "use" that turns lawful reception into an offence.

The hobby's freedom rests on scanner listeners being harmless observers. One person
acting badly on what they heard is how restrictions get written.

## Rebroadcasting and privacy

Sharing what you hear — a live feed, a posted recording, a social-media clip — is its own
decision with its own weight. Even where rebroadcasting is legal, remember that the
traffic often concerns **real people on their worst day**: victims, patients, callers.
Broadcasting a name, an address, a licence plate, or a medical detail can cause genuine
harm and follows people long after the incident.

If you run a feed or publish recordings, the responsible practice is to think about
**what you are exposing and about whom**, to honour any delay or filtering norms in your
community, and to err toward the person in the traffic rather than the story. We return
to feeds specifically in
[audio feeds & streaming](/learn/scanning/audio-feeds-and-streaming/), where the
mechanics and the responsibilities meet.

## The etiquette that keeps the hobby healthy

Beyond the law there is simply being a good citizen of the airwaves. Listen out of
interest, not intrusion. Don't sensationalise. Don't dox. Share knowledge with newcomers,
contribute accurate frequencies back to the community databases, and treat the agencies
you monitor as neighbours rather than targets. The scanning hobby has stayed broadly
legal and welcome because most of its practitioners behave this way — quiet, curious, and
respectful. Keeping it that way is on all of us.

<div class="knowledge-check" data-quiz data-correct-msg="Right — being allowed to receive something is a separate question from what you may lawfully or ethically do with it." markdown="0">
  <p class="knowledge-check__q">Quick check: you can legally receive an unencrypted call. What does that tell you about sharing it?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">That you are automatically free to publish or rebroadcast it however you like</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Nothing on its own — using, sharing, or publishing the contents is a separate question with its own rules and risks</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">That the traffic must not have been sensitive, or it would have been encrypted</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- A scanner is **receive-only and passive**, and **receiving unencrypted radio for
  personal use is legal in most places** — but rules vary, so check your jurisdiction.
- **Receiving** something and **using or publishing** it are **separate questions**;
  "yes" to the first says nothing about the second.
- **Encryption is off-limits** — illegal to defeat and impossible to decode; leave it and
  enjoy the open spectrum.
- **Do no harm**: never interfere, never act on live traffic in ways that impede
  responders, never use what you hear to commit a crime.
- **Rebroadcasting carries responsibility** — real people are in that traffic, so weigh
  privacy before you publish.
- **Good etiquette keeps the hobby legal and welcome** — listen with respect and give
  back to the community.

Next up: [Hardware scanners vs. SDR](/learn/scanning/scanners-vs-sdr/).
