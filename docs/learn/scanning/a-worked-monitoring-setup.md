---
slug: a-worked-monitoring-setup
title: A worked end-to-end monitoring setup
description: Start to finish, one complete build — antenna, feedline, SDR, GopherTrunk, per-call recording, alerting, and a dashboard, assembled and running as an always-on monitoring station that follows a real trunked system.
keywords: monitoring setup, end-to-end scanner, GopherTrunk build, worked example, SDR monitoring station, complete scanner setup, trunk tracking build, always-on monitoring, scanner dashboard, full setup
level: advanced
status: full
prereq:
  - gophertrunk-as-a-scanner
  - building-a-monitoring-post
faq:
  - q: What does a complete monitoring setup include?
    a: The whole chain, in order — a good outdoor antenna, low-loss feedline into the shack, an SDR receiver, a low-power computer running GopherTrunk to lock the control channel and follow calls, per-call recording and logging to disk, alerting on the talkgroups you care about, and a dashboard to watch it all. Each piece is a lesson from this module; this one bolts them together.
  - q: In what order do you build it?
    a: Signal-path order, because each stage depends on the one before it. Antenna and feedline first (nothing downstream beats a bad antenna), then the SDR, then GopherTrunk locked on a control channel, then recording and logging, then alerting, then the dashboard and the always-on service wrapper last. Get each stage working before adding the next.
  - q: How do I know the whole thing is working?
    a: Trace one call end to end. A radio keys up, the control channel grants it, GopherTrunk follows it to a voice channel, you hear decoded audio, a tagged recording lands on disk with a matching log row, and any alert you set fires. If that full path works for one call, it works for all of them.
---

# A worked end-to-end monitoring setup

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
This is the whole module in one build. In **signal-path order** — antenna → feedline
→ **SDR** → **GopherTrunk** → recording & logging → alerting → dashboard, wrapped as
an **always-on service** — you assemble a complete station that locks a real trunked
system's control channel, follows its calls, and produces **tagged per-call audio**
you can search and get alerted on. Build **one stage at a time**, confirm each before
the next, and test the finished post by **tracing a single call end to end**.
</div>

Every lesson so far has been one piece. This one is the assembly: a single, concrete,
end-to-end monitoring setup, built in the order the signal actually flows, so you can
see how the parts you've met click together into a working station. Nothing here is
new — it's the module, wired up. Follow it as a template and adapt the specifics to
your own systems and gear. The [architecture overview](/architecture.html) shows the
same chain from GopherTrunk's side; the [project home](/) is where you get the
software.

## Build in signal-path order

Assemble the chain the way the signal travels, because every stage depends on the one
before it — a brilliant decoder can't rescue a bad antenna, and a flawless antenna is
wasted if the service keeps crashing. Get each stage working and verified before you
add the next:

1. **Antenna** — the biggest single factor in what you hear.
2. **Feedline & connectors** — deliver the signal indoors without throwing it away.
3. **SDR** — digitise the spectrum.
4. **GopherTrunk** — lock the control channel and follow calls.
5. **Recording & logging** — write tagged per-call audio and a searchable log.
6. **Alerting** — get told when the calls you care about happen.
7. **Dashboard & always-on service** — watch it, and keep it running unattended.

## Stage 1 — antenna and feedline

Start outdoors and up high. A wideband [antenna](/learn/scanning/antennas-for-scanning/)
like a discone, mounted as high and clear as you can manage, does more for your
results than any other single choice. Run it down to the shack with **low-loss
[feedline](/learn/scanning/feedlines-and-connectors/)** and good connectors, keeping
the run as short as practical — coax loss is signal you spend once and never get back.
Ground the install for safety while you're at it. This unglamorous stage sets the
ceiling on everything that follows.

## Stage 2 — the SDR

Plug the feedline into your [SDR](/learn/rf-sdr/what-is-sdr/) and the SDR into the
computer. Confirm the raw basics before decoding anything: the receiver is recognised,
you can see the [waterfall](/learn/scanning/identifying-unknown-signals/), and your
target system's control channel shows up as that solid, never-quiet carrier. If you
can see the control channel here, GopherTrunk can lock it; if you can't, fix reception
now, not later.

## Stage 3 — GopherTrunk locks and follows

Point [GopherTrunk](/learn/scanning/gophertrunk-as-a-scanner/) at the SDR and give it
the system's **control-channel frequency and type**. It locks the control channel,
starts reading grants, and follows each call to its voice channel — decoding P25, DMR,
NXDN, TETRA, or whatever the system runs. Confirm the lock the same way you always do:
within seconds you should see affiliations and voice grants scrolling, and hear
decoded audio when a call is granted. This is [following a
call](/learn/scanning/following-a-call/), running for real. If the lock won't hold,
[when decoding fails](/learn/scanning/when-decoding-fails/) is your checklist.

## Stage 4 — recording, logging, and alerting

Now capture what you're hearing. Turn on **per-call
[recording](/learn/scanning/logging-and-recording/)** so each call lands as its own
tagged file, with a matching row in the **log**. Load your
[talkgroup aliases](/learn/scanning/metadata-and-tagging/) so the log reads in names,
not numbers, and set a **[retention policy](/learn/scanning/logging-and-recording/)**
so the disk doesn't fill. Then add a couple of narrow
**[alerts](/learn/scanning/alerting-on-calls/)** on the talkgroups or units you most
care about — and only those, to keep them trustworthy. At this point the station is
producing a searchable archive on its own.

## Stage 5 — dashboard and always-on

Finally, make it watchable and durable. A **dashboard** gives you live status — locks,
recent calls, disk space — so you can check the post at a glance or from your phone,
which matters because it runs [headless](/learn/scanning/building-a-monitoring-post/).
Then wrap GopherTrunk as a **managed service** with
[systemd](/learn/linux-cli/services-and-systemd/) so it starts on boot and restarts on
failure, and follow the [deployment guide](/learn/deployment/deploying-gophertrunk/) to
stand it up properly. Now it's not a program you launched — it's a
[monitoring post](/learn/scanning/building-a-monitoring-post/) that looks after
itself.

## Prove it: trace one call

The whole build is verified by a single test. Wait for — or watch for — one call, and
follow it the entire way: a radio keys up, the control channel grants it, GopherTrunk
tunes to the voice channel, you hear clean decoded audio, a **tagged recording lands
on disk with a matching log row**, and any alert you set **fires**. If that complete
path works for one call, it works for all of them, and you have a real, end-to-end,
always-on monitoring station — every lesson in this module, running at once.

<div class="knowledge-check" data-quiz data-correct-msg="Right — build in signal-path order, because each stage depends on the one before it and a bad antenna can't be fixed downstream." markdown="0">
  <p class="knowledge-check__q">Quick check: why build the setup antenna-first, in signal-path order?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Because the antenna is the most expensive part and should be tested for defects</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Because each stage depends on the one before it — nothing downstream can fix a bad antenna</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Because GopherTrunk refuses to start until an antenna is detected</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- A complete setup is the whole chain: **antenna → feedline → SDR → GopherTrunk →
  recording & logging → alerting → dashboard**, wrapped as an always-on service.
- Build in **signal-path order**, verifying each stage before the next — nothing
  downstream fixes a bad antenna.
- GopherTrunk **locks the control channel and follows grants**; recording, logging,
  and alerting turn that into a searchable, watched archive.
- Make it durable with a **dashboard** and a **managed service** (systemd), per the
  [deployment guide](/learn/deployment/deploying-gophertrunk/).
- Verify the whole thing by **tracing one call end to end** — key-up to tagged
  recording, log row, and alert.

Next up: [the community & contributing data](/learn/scanning/contributing-and-community/).
