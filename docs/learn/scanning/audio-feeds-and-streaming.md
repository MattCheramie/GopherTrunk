---
slug: audio-feeds-and-streaming
title: Audio feeds & streaming
description: How live scanner feeds like Broadcastify put local radio traffic on the internet — the encoder-to-server chain that streams your audio, and the legal, ethical, and reliability duties of running a feed responsibly.
keywords: scanner feed, Broadcastify, audio streaming, live scanner audio, feed provider, streaming encoder, Icecast, scanner stream, online scanner, audio feed setup
level: intermediate
status: full
prereq:
  - logging-and-recording
faq:
  - q: How does a live scanner feed work?
    a: Your receiver decodes audio as usual, an encoder on your computer compresses it into a streaming format and sends it to a feed provider's server, and the provider relays that stream to everyone listening on the web or an app. It's the same chain as any internet radio station — a source encoder pushing to a streaming server that fans the audio out to many listeners.
  - q: What is Broadcastify?
    a: Broadcastify is the largest public directory of live scanner audio feeds, an outgrowth of the RadioReference community. Volunteers run receivers and stream their local audio to Broadcastify's servers, which host thousands of feeds that anyone can listen to on the web or a phone app. It's how most people hear scanner traffic without owning a scanner.
  - q: Is streaming a scanner feed legal?
    a: In many places receiving and sharing unencrypted radio is legal, but the rules vary by country and even by state, and legality of receiving is not the same as legality of every use. Feed providers also impose their own rules — commonly no encrypted traffic and delays on sensitive channels. Check the law where you are and the provider's terms before you stream, the same care the legal-and-ethical lesson stresses.
---

# Audio feeds & streaming

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
A **feed** puts your local scanner audio on the internet for anyone to hear. The
chain is simple: your receiver decodes audio, an **encoder** compresses and pushes
it to a **feed provider** like **Broadcastify**, and the provider relays it to every
listener. Running a feed is a small broadcasting responsibility — keep it **reliable
and identified**, and stay inside the **law and the provider's rules** (no encrypted
traffic, delays where required). It's the same care the
[legal & ethical](/learn/scanning/scanning-legal-and-ethical/) lesson stresses,
applied to *sharing* rather than just listening.
</div>

Most people who have "listened to a scanner" have never owned one — they heard it on
a phone app, streamed by a volunteer somewhere with a receiver and an internet
connection. Those volunteers run **feeds**, and it's one of the most social things
you can do in the hobby: your rooftop antenna becomes something the whole world can
tune to. This lesson explains how a feed works and what running one responsibly asks
of you.

## The streaming chain

A live feed is ordinary internet radio, and it has three links:

1. **Source** — your receiver produces decoded audio, exactly as if you were
   listening. For a trunked system that's the followed voice channel; for a
   conventional channel it's whatever the squelch opens on.
2. **Encoder** — software on your computer compresses that audio into a streaming
   codec (MP3 or AAC, at a modest bitrate) and pushes it to the provider's server.
   This is the same job an internet-radio broadcaster's encoder does.
3. **Provider / server** — a service like Broadcastify receives your single stream
   and **fans it out** to everyone listening, so you upload once no matter how many
   people tune in. The provider also hosts the directory, the web player, and the
   phone apps.

You supply the first two links and a stable internet connection; the provider
supplies the third. Your upstream bandwidth only ever carries **one** copy of the
stream, which is why even a modest home connection can feed thousands of listeners.

## Broadcastify and feed providers

**Broadcastify** is the dominant public feed directory, grown out of the same
[RadioReference](/learn/scanning/radioreference-database/) community that maintains
the system database. Volunteers stream their local audio to its servers, and it hosts
thousands of feeds that anyone can browse by location and listen to free. When you
apply to run a feed, you typically pick the system or area it covers, get stream
credentials, point your encoder at their server, and appear in the directory once
it's live.

Other providers and self-hosted options exist — you can run your own **Icecast**
server and publish a stream yourself — but for reach and discoverability, a shared
directory is where the listeners already are. The trade-off is that a directory
imposes its own rules on top of the law, which is the next thing to get right.

## Doing it responsibly

Putting audio on the air, even relayed, is a small broadcasting responsibility, and
feeds live or die on trust. A few duties come with it:

- **Respect the law first.** Whether you may receive and re-share a given
  transmission depends on where you are; receiving unencrypted traffic is broadly
  permitted in many places, but not everywhere, and legality of *receiving* is not
  legality of every *use*. Revisit
  [legal & ethical scanning](/learn/scanning/scanning-legal-and-ethical/) before you
  publish anything.
- **Never stream encrypted traffic** even if you could — providers forbid it and it
  serves no one, since it decodes to noise anyway.
- **Follow the provider's channel rules.** Many require a broadcast delay or exclude
  sensitive channels (tactical talkgroups, phone patches) to avoid interfering with
  operations or exposing personal information.
- **Keep it clean and identified.** A good feed is a stable, well-labelled stream of
  the system it claims to carry — not a mix of half-tuned channels drifting in and
  out.

The guiding idea is the hobby's oldest: **receive freely, act responsibly.** A feed
amplifies both — reaching more people and carrying more responsibility for what those
people hear.

## Reliability is the real work

The technical setup is a one-evening job; keeping the feed *up* is the ongoing one.
Listeners rely on it, and a feed that drops out during the one big incident of the
month is the feed nobody trusts again. Reliability means the boring things: a
receiver and computer that run unattended, an encoder that reconnects automatically
when the network hiccups, and a machine that comes back on its own after a power
blip.

That's precisely the always-on discipline the
[monitoring-post](/learn/scanning/building-a-monitoring-post/) lesson is about, and
a public feed is one of the best reasons to build a post properly. If people are
counting on your audio, "I'll restart it when I get home" isn't good enough.

<div class="knowledge-check" data-quiz data-correct-msg="Right — you upload one stream to the provider, and the provider's server fans it out to every listener." markdown="0">
  <p class="knowledge-check__q">Quick check: when a thousand people listen to your feed, how many copies of the stream does your connection upload?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">One per listener — a thousand copies</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">One — the provider's server fans that single stream out to everyone</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">None — listeners connect directly to your receiver</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- A **feed** streams your local scanner audio to the internet through a
  source → **encoder** → **provider** chain, the same as internet radio.
- The provider (commonly **Broadcastify**) **fans out** your single uploaded stream
  to every listener, so your bandwidth carries just one copy.
- Streaming is **sharing**, which carries duties: obey the law, never stream
  encrypted traffic, follow the provider's channel rules and delays.
- Keep the feed **clean, labelled, and identified** — feeds run on the trust of the
  people listening.
- **Reliability** is the ongoing job; a public feed is a strong reason to build a
  proper [monitoring post](/learn/scanning/building-a-monitoring-post/).

Next up: [alerting on the calls you care about](/learn/scanning/alerting-on-calls/).
