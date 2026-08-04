---
slug: gophertrunk-as-a-scanner
title: GopherTrunk as a scanner
description: Using GopherTrunk to follow and decode a real trunked system — point it at an SDR, lock the control channel, follow the grants it hands out, and get tagged per-call audio out the other end.
keywords: GopherTrunk scanner, trunk tracking, control channel decode, follow grants, GopherTrunk config, P25 DMR NXDN TETRA, per-call audio, open source scanner, SDR decoder, GopherTrunk setup
level: advanced
status: full
prereq:
  - programming-a-trunked-system
  - scanning-with-sdr-software
faq:
  - q: What does GopherTrunk actually do?
    a: It takes raw samples from an SDR, locks a trunked system's control channel, reads the grants that control channel hands out, tunes to and decodes the voice channels those grants point to, and produces tagged per-call audio with logs and metadata. In short, it's the software that turns a bare SDR into a trunk-tracking scanner across P25, DMR, NXDN, TETRA, and more.
  - q: What do I need to give it to start?
    a: A supported SDR, an antenna, and one system's control-channel frequency and type — the same details you'd program into any trunking scanner. From that single frequency GopherTrunk discovers the rest of the system by decoding the control channel, or you can let its Hunt feature find control channels you don't have.
  - q: How is it different from a hardware trunking scanner?
    a: Same behaviour — lock control channel, follow calls — but open and exposed. It's multi-protocol, updatable, can follow more from one wideband capture, and lays every call's metadata, logs, and audio open for recording, alerting, and integration. It's the brain of the setup rather than a sealed appliance.
---

# GopherTrunk as a scanner

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
GopherTrunk is the software that turns a bare [SDR](/learn/rf-sdr/what-is-sdr/) into a
**trunk-tracking scanner**. Give it an SDR and **one system's control-channel
frequency and type**; it **locks the control channel**, reads the **grants** that
channel hands out, **follows** each to its voice channel, and produces **tagged
per-call audio** with logs and metadata — across P25, DMR, NXDN, TETRA, and more.
It's the same lock-and-follow behaviour as a hardware trunking scanner, but **open,
multi-protocol, and fully exposed** for recording, alerting, and integration.
</div>

You've seen where GopherTrunk sits among
[SDR scanning software](/learn/scanning/scanning-with-sdr-software/) — a
multi-protocol trunk-tracker on the top rung. This lesson puts it to work: what you
feed it, what it does with a real system, and what comes out. Everything you learned
about [programming a trunked system](/learn/scanning/programming-a-trunked-system/)
and [following a call](/learn/scanning/following-a-call/) is exactly what GopherTrunk
does automatically — here it is in the flesh. The [project home](/) and the
[architecture overview](/architecture.html) go deeper than one lesson can; this is the
scanner's-eye view.

## What you point it at

GopherTrunk needs the same three things any trunking scanner does, no more:

- **An SDR and antenna** — the [receiver](/learn/scanning/scanners-vs-sdr/) that hands
  it raw samples, fed by a [good antenna](/learn/scanning/antennas-for-scanning/).
- **One control-channel frequency** — the single frequency that unlocks the whole
  system, exactly as in
  [programming a trunked system](/learn/scanning/programming-a-trunked-system/).
- **The system type** — P25, DMR, NXDN, TETRA, and so on, so it knows which decoder
  to run.

From that one control-channel frequency it discovers the rest of the system by
listening. And when you *don't* have a control-channel frequency, its **Hunt** feature
sweeps a band, detects carriers, identifies the protocol, and reports which one is the
control channel — the [search-and-discovery](/learn/scanning/searching-and-discovery/)
job, automated.

## Locking the control channel

The moment GopherTrunk has the control-channel frequency and type, its first job is to
**lock** that channel — tune to it, demodulate it, and start decoding the steady
stream of [signalling](/learn/scanning/programming-a-trunked-system/) the control
channel never stops sending. A good lock is the foundation of everything else; if the
control channel isn't decoding cleanly, nothing downstream works, which is why
[when decoding fails](/learn/scanning/when-decoding-fails/) starts there.

Once locked, GopherTrunk is reading the system's live bookkeeping: affiliations,
registrations, and — the part that matters for scanning — the **grants** that assign a
talkgroup's call to a voice channel. It's the same control channel a
[hardware scanner](/learn/scanning/scanners-vs-sdr/) reads, decoded in the open.

## Following the grants

This is trunk-tracking, and it's the whole point. Each time the control channel grants
a call — "talkgroup 1201, go to this voice channel" — GopherTrunk **follows** it: it
tunes to that voice channel, decodes the digital voice, and plays or records the call,
then returns to watching the control channel for the next grant. That
control-channel-to-voice-and-back loop is precisely
[following a call across the system](/learn/scanning/following-a-call/), running
continuously and automatically.

Because it works from a wideband capture, GopherTrunk can follow more of a system at
once than switching a single tuner around — several concurrent calls decoded in
parallel from the same receiver, which is where software pulls ahead of a
one-call-at-a-time box.

## What comes out

The output is everything Unit 5 was about, produced for you:

- **Tagged per-call audio** — each call as its own
  [recording](/learn/scanning/logging-and-recording/), labelled with system,
  talkgroup, unit, time, and frequency straight from the control channel.
- **A live call log** and **metadata** — the searchable record from
  [metadata & tagging](/learn/scanning/metadata-and-tagging/), written as calls
  happen.
- **Hooks for the rest** — [alerting](/learn/scanning/alerting-on-calls/),
  [feeds](/learn/scanning/audio-feeds-and-streaming/), and dashboards all draw on the
  same exposed stream of calls and metadata.

In other words, GopherTrunk isn't just a decoder — it's the engine that drives the
whole logging, recording, and alerting apparatus you built earlier in the module.

## Running it for real

For a quick listen you run it on your desk; for anything lasting you run it as the
[always-on post](/learn/scanning/building-a-monitoring-post/) — headless, managed, and
self-recovering. Because GopherTrunk is server-shaped software, that's its natural
home, and the [deploying GopherTrunk](/learn/deployment/deploying-gophertrunk/) guide
walks through standing it up as a durable service rather than a program you babysit.
The [architecture overview](/architecture.html) shows how the pieces — receiver,
control-channel decoder, grant follower, voice decoders, recorders — fit together, and
the [project home](/) is where to start if you want to actually run it.

The next lesson assembles all of this — antenna, SDR, GopherTrunk, logging, and a
dashboard — into one complete, worked monitoring setup.

<div class="knowledge-check" data-quiz data-correct-msg="Right — from one control-channel frequency GopherTrunk locks the control channel, then follows every grant to its voice channel automatically." markdown="0">
  <p class="knowledge-check__q">Quick check: what does GopherTrunk do after it locks a system's control channel?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Nothing more — locking the control channel is the whole job</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">It reads the grants and follows each call to its voice channel, decoding the audio</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">It re-transmits the control channel to other scanners nearby</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- GopherTrunk turns a bare **SDR** into a **trunk-tracking scanner** across P25, DMR,
  NXDN, TETRA, and more.
- You give it an **SDR, one control-channel frequency, and the system type**; its
  **Hunt** feature can find control channels you don't have.
- It **locks the control channel**, reads the **grants**, and **follows** each call to
  its voice channel — trunk-tracking, automated.
- It outputs **tagged per-call audio**, a **live log and metadata**, and **hooks** for
  alerting, feeds, and dashboards.
- Run it briefly on the desk or, for real, as an
  [always-on post](/learn/scanning/building-a-monitoring-post/) — see the
  [architecture overview](/architecture.html) and
  [deployment guide](/learn/deployment/deploying-gophertrunk/).

Next up: [a worked end-to-end monitoring setup](/learn/scanning/a-worked-monitoring-setup/).
