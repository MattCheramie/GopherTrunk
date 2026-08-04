---
slug: logging-and-recording
title: Logging & recording calls
description: Turn live scanner traffic into a reviewable archive — call logging that writes down who talked and when, per-call recording that captures the audio, and the storage a busy trunked system quickly demands.
keywords: call logging, scanner recording, per-call recording, call log, audio archive, trunk recorder, recording storage, talkgroup log, scanner history, monitoring records
level: intermediate
status: full
prereq:
  - following-a-call
faq:
  - q: What is the difference between logging and recording?
    a: A log is text — a timestamped list of which talkgroup or unit was active, when, and for how long. A recording is the actual audio of the call. Logging tells you a call happened; recording lets you hear it. Most modern setups do both at once, writing one log row per call alongside the audio file it points to.
  - q: What is per-call recording?
    a: Instead of one long continuous audio file, a per-call recorder saves each transmission (or each call) as its own short file, tagged with the talkgroup, time, and system. That is what makes an archive searchable — you can jump straight to a single call instead of scrubbing through hours of mostly-silence.
  - q: How much storage does recording use?
    a: Far less than you would guess, because decoded voice is low-bitrate and calls are short. A busy public-safety system might produce a few hundred megabytes to a couple of gigabytes of compressed per-call audio a day. The metadata log is tiny by comparison. The real question is retention — how many days or weeks you keep before old audio is pruned.
---

# Logging & recording calls

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
Listening live means you hear a call **once**; logging and recording let you review
it later. A **call log** is the timestamped text record — which **talkgroup** or unit
was active, when, and for how long — while a **recording** is the audio itself.
Modern setups do **per-call recording**: each call becomes its own short, tagged
file, which is what makes an archive searchable rather than one endless tape. Plan
for **storage and retention** up front — a busy system fills a disk steadily, so
decide how many days you keep before old audio is pruned.
</div>

You can only pay attention to so much in real time, and the interesting call is
always the one you half-heard while doing something else. Logging and recording fix
that: they turn the fleeting live stream into an archive you can search, replay, and
share. This lesson covers what to capture, how per-call recording works, and the
storage that a busy system quietly demands.

## Logging: the text record

A **call log** is the cheapest, most durable thing you can keep. Every time the
control channel grants a call, your setup writes one row: a timestamp, the talkgroup
(and its alias, if you have one), the source unit ID if the system sends it, the
frequency the call landed on, and the duration. That's it — a few dozen bytes per
call.

That humble list is more useful than it looks. It shows you the **rhythm** of a
system: which talkgroups are busy at 3 a.m. versus rush hour, which units check in
where, when a big incident lights up half the fleet at once. You can keep months of
logs in a text file or a small database and never notice the space. Even if you
never record a second of audio, the log alone answers "was anything happening on
that channel last Tuesday?"

## Recording: capturing the audio

A log tells you a call happened; a **recording** lets you hear it. Once you follow a
[call across the system](/learn/scanning/following-a-call/), the decoded voice is
just audio, and audio can be written to a file as easily as it's played to a speaker.

There are two ways to do it. The old way is **continuous recording** — one long file
per channel or per session, running whether or not anyone is talking. It's simple,
but it leaves you scrubbing through hours of dead air to find thirty seconds of
traffic, and it wastes space on silence. The modern way, described next, is far
better for scanning.

## Per-call recording

**Per-call recording** saves each call as its own short file, created the moment the
call starts and closed when it ends. Because your setup already knows the call's
metadata from the control channel, it can name and tag each file automatically:
system, talkgroup, timestamp, source unit, frequency. A morning's monitoring becomes
a folder of hundreds of small, labelled clips instead of one giant tape.

This is the format that makes an archive genuinely useful:

- **You jump straight to a call** — no scrubbing. Find the row in the log, open the
  file it points to.
- **Silence costs nothing** — the recorder only writes while a call is active, so a
  quiet night produces a quiet folder.
- **Each file carries its own context** — the talkgroup and time travel with the
  audio, which is exactly what the next lesson on
  [metadata & tagging](/learn/scanning/metadata-and-tagging/) builds on.

Per-call recording pairs naturally with trunk tracking, because the recorder and the
[grant follower](/learn/scanning/following-a-call/) are reading the same control
channel. The log row and the audio file are written together, pointing at each other.

## What to record — and what not to

You rarely want *everything*. A busy metro system can grant thousands of calls a day
across dozens of talkgroups, most of which you don't care about. Lean on your
[talkgroups and scan lists](/learn/scanning/talkgroups-and-scan-lists/): record the
talkgroups you're monitoring and let the rest scroll past in the log only. That keeps
the archive focused and the disk usage sane.

Two things you can't usefully record: **encrypted** calls, which decode to noise, and
calls your receiver never followed because it was busy on another one. Neither is a
fault in your recorder — the first is by design on the system's end, and the second
is the nature of following one call at a time, a limit the
[when-decoding-fails](/learn/scanning/when-decoding-fails/) lesson returns to.

## Storage and retention

Decoded voice is low-bitrate and calls are short, so recordings are smaller than
newcomers expect — a busy public-safety system might produce anywhere from a few
hundred megabytes to a couple of gigabytes of compressed per-call audio a day, and
the text log is a rounding error beside it. Still, "a couple of gigabytes a day"
fills a disk in a month or two, so the real decision is **retention**: how long you
keep audio before old files are pruned.

Pick a policy and automate it. Common patterns are a rolling window (keep the last N
days, delete the rest), a size cap (prune oldest when the folder passes a limit), or
keeping the log forever while ageing out only the heavy audio. Whatever you choose,
decide it now — an archive with no retention policy is a disk that fills up at the
worst possible moment.

<div class="knowledge-check" data-quiz data-correct-msg="Right — per-call recording writes each call as its own tagged file, so you can jump straight to it instead of scrubbing a long tape." markdown="0">
  <p class="knowledge-check__q">Quick check: what makes per-call recording more useful than one continuous file?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">It records encrypted calls that continuous recording can't</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Each call is its own short, tagged file, so you jump straight to it</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">It uses more disk space, which means higher audio quality</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- A **call log** is the timestamped text record of which talkgroup or unit was
  active, when, and for how long — cheap, durable, and useful on its own.
- A **recording** captures the audio; logging and recording usually run together,
  one log row per call.
- **Per-call recording** saves each call as its own tagged file, which is what makes
  an archive searchable instead of an endless tape.
- Record the talkgroups you care about, not everything — encrypted calls decode to
  noise and can't be usefully saved.
- Decoded voice is small, but a busy system still fills a disk over weeks, so set a
  **retention policy** up front.

Next up: [metadata, tagging & searchability](/learn/scanning/metadata-and-tagging/).
