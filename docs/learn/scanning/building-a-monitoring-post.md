---
slug: building-a-monitoring-post
title: Building an always-on monitoring post
description: Go from "switch it on when I'm home" to a setup that runs unattended 24/7 — the low-power hardware, headless software, and reliability habits that make a monitoring post keep logging and recording while you're not watching.
keywords: monitoring post, 24/7 scanner, unattended monitoring, headless SDR, always-on scanner, remote monitoring, low-power SDR server, scanner uptime, monitoring station, systemd scanner
level: advanced
status: full
prereq:
  - logging-and-recording
  - alerting-on-calls
faq:
  - q: What is a monitoring post?
    a: A receiver, antenna, and computer set up to run continuously and unattended — logging, recording, and alerting on radio traffic whether or not anyone is watching. Where a scanner on the desk is something you switch on, a monitoring post is something that's always on, so the interesting call at 3 a.m. is captured instead of missed.
  - q: What hardware does an always-on post need?
    a: Less than you'd think. A low-power computer like a Raspberry Pi or a small mini-PC, one or more SDR receivers, a good fixed antenna, and reliable power and network. The priorities shift from raw speed to running cool, quiet, and continuously — heat and power blips are the enemies of uptime, not CPU cycles.
  - q: How do I make it recover on its own?
    a: Run the software as a managed service that restarts on failure and starts on boot, so a crash or a power cut heals itself without you. On Linux that's typically a systemd unit. The goal is that after any hiccup — crash, reboot, network drop — the post comes back to full operation with no keyboard involved.
---

# Building an always-on monitoring post

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
A **monitoring post** is a receiver, antenna, and computer built to run
**unattended, around the clock** — so the interesting call at 3 a.m. is
[logged and recorded](/learn/scanning/logging-and-recording/) instead of missed. The
hardware is modest (a **low-power computer**, an SDR, a good fixed antenna); the real
work is **reliability** — running **headless**, as a **managed service** that
restarts on failure and boots on power-up, so any crash or outage heals itself. Build
it to survive the hiccups you won't be there to fix.
</div>

Everything so far — logging, recording, tagging, feeds, alerting — assumes the setup
is *running* when the traffic happens. A scanner on your desk only runs when you
switch it on, which means it misses the overnight incident, the weekend call-out, the
event you didn't know was coming. A **monitoring post** removes that gap: it's a
station built to be always on, capturing continuously so you review at your
convenience instead of listening in real time. This lesson is about crossing from
"on when I'm home" to "on, full stop."

## What changes when it's always on

The shift from an attended scanner to an unattended post is less about capability
than about **assumptions**. When you're sitting there, you're the error handler — you
notice the lock dropped, restart the crashed program, re-tune after a glitch. An
always-on post has no one in that chair, so every job you used to do by hand has to
happen by itself.

That reframes what "good" means. A desktop setup is judged on how well it decodes
right now; a monitoring post is judged on how many days it runs **without you
touching it**. Uptime, self-recovery, and unattended reliability become the headline
features, and raw performance takes a back seat to not-falling-over.

## Hardware for uptime, not speed

The hardware is more modest than newcomers expect, because the priorities are
different. You want gear that runs **cool, quiet, and continuously**:

- **A low-power computer.** A Raspberry Pi or a small fanless mini-PC is plenty for
  following a system or two, sips power, and makes no noise — ideal for something
  that runs in a closet forever.
- **One or more SDR receivers.** One SDR follows a system; add more to cover more
  systems or sites at once, which is where dedicated software earns its keep.
- **A good fixed antenna.** The single biggest factor in what you hear, and a post
  rewards a proper permanent install — revisit
  [antennas](/learn/scanning/antennas-for-scanning/) and
  [feedlines](/learn/scanning/feedlines-and-connectors/) for the real gains.
- **Reliable power and network.** Heat and power blips are the enemies of uptime, not
  CPU cycles. Good ventilation and, if you can, a small battery backup keep a brief
  outage from taking the post down.

Notice raw speed isn't on the list. Decoding a control channel and a few voice
channels is light work; the hard part is doing it for months without a stumble.

## Headless and hands-off

An always-on post usually runs **headless** — no monitor, no keyboard, tucked next to
the antenna feedline and reached over the network. That's a feature: it can live
wherever the antenna run is shortest, and you check on it from your laptop or phone.
It does mean the software has to be operable without a screen in front of it — a web
dashboard, a status page, remote logs — which is exactly how server software (and
GopherTrunk) is built to run.

Running headless also forces good hygiene. You can't rely on "I'll see it on the
screen," so status has to be *reported*: uptime, current locks, recent calls, disk
space. A post you can't check remotely is a post you're flying blind on.

## Reliability: make it heal itself

This is the heart of a monitoring post. Unattended means **self-recovering**: after
any hiccup — a crashed decoder, a rebooted machine, a dropped network link — the post
must return to full operation with no keyboard involved. The mechanism is to run the
software as a **managed service** rather than a program you launched in a terminal.

On Linux that's a service manager like **systemd**, covered in
[services & systemd](/learn/linux-cli/services-and-systemd/): you define the decoder
as a unit that **starts on boot** and **restarts on failure**, so a crash respawns in
seconds and a power cut brings the whole post back by itself when the lights return.
Layer on the basics — automatic disk pruning from your
[retention policy](/learn/scanning/logging-and-recording/), an
[alert](/learn/scanning/alerting-on-calls/) if the post goes quiet when it shouldn't —
and you have a station that looks after itself. The test of a good post is simple:
pull the power, plug it back in, walk away, and it's fully monitoring again before you
reach the door.

## Where to put it

A little planning on **siting** pays off for years. Keep the feedline run to the
antenna short (loss you spend once, forever), give the computer airflow, and put it
somewhere power and network are stable and you can reach it remotely. An attic, a
closet near the mast, or a shelf by the router are all common homes. The best
monitoring post is one you set up once, site well, and then mostly forget about —
because it just keeps running, and the archive just keeps filling.

<div class="knowledge-check" data-quiz data-correct-msg="Right — an always-on post must self-recover, so running the software as a managed service that restarts on failure and boots on power-up is the key." markdown="0">
  <p class="knowledge-check__q">Quick check: what most defines a good always-on monitoring post?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">The fastest possible CPU so it can decode more channels</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">It self-recovers — running as a managed service that restarts on failure and boots on power-up</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">A monitor and keyboard attached so you can watch it constantly</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- A **monitoring post** runs **unattended, 24/7**, so overnight and weekend traffic
  is captured instead of missed.
- The mindset shifts from raw performance to **uptime and self-recovery** — no one is
  there to handle errors by hand.
- The hardware is modest: a **low-power computer**, one or more SDRs, a good **fixed
  antenna**, and reliable power and network that run cool and continuously.
- Run it **headless** and report status remotely, since there's no screen in front of
  it.
- Make it **heal itself** — a **managed service** (systemd) that restarts on failure
  and boots on power-up, plus disk pruning and a quiet-post alert.

Next up: [scanning with SDR software](/learn/scanning/scanning-with-sdr-software/).
