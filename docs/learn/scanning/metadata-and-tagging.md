---
slug: metadata-and-tagging
title: Metadata, tagging & searchability
description: A recording you can't find is a recording you don't have — tag every call with talkgroup, time, system, and unit so a month of scanner audio stays searchable, and build the aliases that turn raw IDs into names.
keywords: call metadata, tagging recordings, talkgroup alias, searchable archive, scanner metadata, unit ID alias, call tagging, audio metadata, filename convention, monitoring database
level: intermediate
status: full
prereq:
  - logging-and-recording
faq:
  - q: What metadata should every recorded call carry?
    a: At minimum a timestamp, the system, the talkgroup (both its numeric ID and a human alias), the source unit ID if the system sends one, the frequency the call landed on, and the duration. That set lets you find any call later by who, when, or where — and it's exactly what your setup already knows from the control channel, so capturing it costs nothing extra.
  - q: What is a talkgroup alias?
    a: A friendly name mapped to a numeric talkgroup ID — turning "talkgroup 1201" into "County Fire Dispatch." Aliases live in a lookup table you build or import from a database like RadioReference. They make logs and recordings readable at a glance instead of a wall of numbers, and they're the single biggest quality-of-life upgrade to an archive.
  - q: Should I encode metadata in the filename or a database?
    a: Both, ideally. A structured filename (system, talkgroup, timestamp) makes files self-describing even if the database is lost, while a database or index makes bulk search and filtering fast. The filename is your durable fallback; the index is your fast path.
---

# Metadata, tagging & searchability

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
A [recording](/learn/scanning/logging-and-recording/) is only worth keeping if you
can **find it again**. Every call should carry **metadata** — timestamp, system,
talkgroup, source unit, frequency, duration — captured automatically from the
control channel at no extra cost. **Aliases** turn raw numbers ("talkgroup 1201")
into names ("County Fire Dispatch"), which is the single biggest readability upgrade
an archive gets. Store that metadata in **both the filename and an index** so a
month of audio stays searchable by who, when, or where.
</div>

Capturing calls is half the job; being able to *find* one later is the other half.
An unlabelled folder of ten thousand audio clips is nearly useless — the call you
want is in there, but so is everything else, and you have no way to reach it. Tagging
is what turns a pile of recordings into a searchable archive. The good news: your
setup already knows everything it needs to tag each call.

## The metadata that matters

When the control channel grants a call, it hands your setup a small bundle of facts.
Capture all of them with the recording:

- **Timestamp** — when the call started, ideally to the second and in a consistent
  time zone (UTC saves you grief across daylight-saving changes).
- **System** — which trunked system or site the call belongs to, so a multi-system
  archive doesn't blur together.
- **Talkgroup** — the numeric ID *and* its alias. Keep both: the ID is authoritative,
  the alias is readable.
- **Source unit ID** — which radio keyed up, if the system sends it. Following one
  unit across talkgroups is a powerful way to track an event.
- **Frequency** — the voice channel the call landed on, useful for troubleshooting
  and for confirming multisite behaviour.
- **Duration** — how long the call ran, which lets you spot the long incident calls
  amid the short routine ones.

None of this is extra work at capture time — it's the same data that drives
[following a call](/learn/scanning/following-a-call/). You're simply writing it down
next to the audio instead of letting it scroll off the screen.

## Aliases: from numbers to names

Raw scanning is a wall of numbers. Talkgroup 1201, unit 4471, system 0x2A. Nobody
remembers those, and a log full of them is barely more searchable than the audio it
describes. **Aliases** fix this by mapping each ID to a human name in a lookup table:
talkgroup 1201 becomes "County Fire Dispatch," unit 4471 becomes "Engine 12."

You build alias tables two ways, usually together. You **import** them from a
community database like [RadioReference](/learn/scanning/radioreference-database/),
which has aliases for tens of thousands of documented systems, and you **add your
own** as you identify talkgroups by listening — the local knowledge no database has.
Feed those aliases into your logs and recordings and the whole archive becomes
readable at a glance. This is the highest-value, lowest-effort improvement you can
make; do it early.

## Filenames and folders that describe themselves

The simplest tag lives in the filename. A convention like

```
SystemName/2026-08-04/1201_CountyFireDispatch_143207_unit4471.wav
```

makes a file self-describing: you can read the system, date, talkgroup, time, and
unit without opening anything or consulting a database. Group files into folders by
system and day and the filesystem itself becomes a coarse search tool — you can
navigate to "County Fire, last Tuesday" with nothing but a file browser.

A structured filename is also your **durable fallback**. Databases get corrupted,
software changes, indexes get rebuilt — but a well-named file explains itself years
later on any machine. Never rely on an external index as the *only* place the
metadata lives.

## An index for fast search

Filenames are durable but slow to search in bulk. For that you want an **index**: a
small database, or even a well-structured log file, holding one row per call with all
its metadata and a pointer to the audio. With an index you can ask real questions —
"every County Fire call over two minutes long between 6 and 9 p.m. last week" — and
get an answer in a moment instead of grepping ten thousand filenames.

The two aren't rivals; they're layers. The **filename is the durable copy** of the
metadata, the **index is the fast path** to it, and if the index is ever lost you can
rebuild it by walking the self-describing filenames. Keep both and your archive stays
both searchable and recoverable. This searchable base is exactly what the next
lessons on [feeds](/learn/scanning/audio-feeds-and-streaming/) and
[alerting](/learn/scanning/alerting-on-calls/) build on top of.

## Consistency is the whole game

Whatever scheme you pick, **apply it the same way every time**. The value of metadata
comes from uniformity: if half your calls use UTC and half use local time, or one
system spells a talkgroup two ways, search silently misses things and you learn not
to trust it. Decide the fields, the time zone, the alias source, and the filename
format once, automate them, and don't drift. An archive you can trust is worth ten
you have to second-guess.

<div class="knowledge-check" data-quiz data-correct-msg="Right — an alias maps a numeric talkgroup or unit ID to a human name, making logs and recordings readable at a glance." markdown="0">
  <p class="knowledge-check__q">Quick check: what does a talkgroup alias do?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">It encrypts the talkgroup ID so it can't be read from the log</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">It maps a numeric ID to a human name, like 1201 to "County Fire Dispatch"</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">It merges several talkgroups into one recording</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- Every call should carry **metadata** — timestamp, system, talkgroup, unit,
  frequency, duration — captured automatically from the control channel.
- **Aliases** map raw IDs to human names and are the single biggest readability
  upgrade an archive gets; import them and add your own.
- A structured **filename** makes each file self-describing and is your durable,
  recoverable copy of the metadata.
- An **index** or small database makes bulk search fast; it's the fast path, the
  filename is the fallback.
- **Consistency** — same fields, time zone, and format every time — is what makes an
  archive you can actually trust.

Next up: [audio feeds & streaming](/learn/scanning/audio-feeds-and-streaming/).
