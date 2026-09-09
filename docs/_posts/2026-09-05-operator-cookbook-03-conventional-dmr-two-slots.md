---
title: "The Operator's Cookbook, Part 3: One Repeater, Two Conversations — Conventional DMR"
description: A single-dongle GopherTrunk build for a conventional DMR Tier II repeater — two independent talkgroups on TS1 and TS2 decoded as two simultaneous calls, with the interleaved two-slot voice path, embedded-LC routing, colour-code pinning, and the honest edge cases.
category: tutorials
keywords: dmr tier 2 scanner, conventional dmr decode, dmr timeslot ts1 ts2, dmr repeater sdr, dmr embedded link control, dmr colour code config, two slot dmr recording, mototrbo scanner sdr, gophertrunk cookbook
tags: [operator-cookbook, dmr, tier-2, tdma, timeslots, config]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "The Operator's Cookbook"
series_part: 3
---

*Part 3 of **The Operator's Cookbook**, a 14-part series of complete,
copy-paste GopherTrunk builds — one working rig per part, antenna to browser.
[Part 2]({{ '/blog/tutorials/operator-cookbook-02-dmr-tier3/' | relative_url }})
decoded a trunked Tier III network and its LCN band plan. This part drops the
trunking entirely: one conventional DMR repeater, no control channel handing
out anything — and yet still *two* simultaneous conversations, because every
DMR carrier is two-slot TDMA. Until recently GopherTrunk collapsed those two
slots into one garbled call; the rebuilt two-slot path decodes them as two.
This recipe is the cheapest complete build in the series: one dongle, no voice
radio, no band plan.*

> **TL;DR:** A DMR Tier II repeater carries **TS1 and TS2 interleaved on one
> carrier**, each able to hold its own talkgroup at the same time. Config is
> `protocol: dmr-tier2` with the repeater's output frequency in
> `control_channels` — one `role: control` dongle suffices, because
> conventional DMR voice rides the **same carrier** the decoder camps
> (GopherTrunk registers two same-carrier voice taps automatically). The
> interleaved two-slot decoder is the **default** for DMR: concurrent calls
> get their own recordings, routed by each superframe's embedded Link Control
> talkgroup, and a Terminator-with-LC ends only its own slot's call. Healthy
> looks like `dmr/tier2 cc locked` then, on a busy afternoon, two
> `recorder: call started` lines at once.

**Key takeaways**

- **Two conversations were always there.** [Tier II]({{ '/reference/dmr-tier-2/' | relative_url }})
  TDMA gives one 12.5 kHz repeater two logical channels; a decoder that treats
  the carrier as one stream produces "DJ scratchy" audio — both calls' AMBE
  frames spliced together (issue #644's signature).
- **The wire doesn't label slots — talkgroups do.** A base-station burst
  carries no reliable physical-slot number, so GopherTrunk assigns each
  concurrent call a *synthetic* timeslot identity and routes audio by the
  [embedded Link Control]({{ '/reference/dmr-embedded-lc/' | relative_url }})'s
  talkgroup, not by slot position.
- **This is the one-dongle-no-voice-radio build.** Voice lives on the decoded
  carrier itself, so the same-carrier taps make a second SDR pointless here.
- **It's new — verify it on your repeater.** The two-slot path is pinned by
  regression tests, but the on-air A/B on a real concurrent-traffic capture is
  still open. If your repeater misbehaves, a capture report is genuinely
  valuable.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| System definition | conventional carrier, camped not hunted | `protocol: dmr-tier2`, `control_channels: [<repeater Hz>]` |
| Two-slot decode | interleaved voice, per-slot calls | `dmr_interleaved_voice` (tri-state; **default on** for DMR) |
| Slot routing | which audio belongs to which call | embedded LC talkgroup — [dmr-embedded-lc]({{ '/reference/dmr-embedded-lc/' | relative_url }}) |
| Co-channel rejection | drop bursts from the wrong system | `color_code: 0..15` ([color code]({{ '/reference/color-code/' | relative_url }})) |
| Call teardown | Terminator-with-LC ends its own slot only | [dmr-full-link-control]({{ '/reference/dmr-full-link-control/' | relative_url }}); hangtime backstop `voice_hangtime_ms` |
| Protocol background | bursts, CACH, superframes | [Protocol Decoders Part 5]({{ '/blog/deep-dives/protocol-decoders-05-dmr-tier-2-3/' | relative_url }}), [dmr-voice-superframe]({{ '/reference/dmr-voice-superframe/' | relative_url }}) |

## In this post

- **What you're building** — a camped repeater that yields two calls at once.
- **The shopping list** — Part 1's, minus nothing, plus nothing.
- **How two slots become two calls** — the decode story, briefly.
- **The config** — the shortest one in this series.
- **First run — what healthy looks like** — beacons, locks, concurrent calls.
- **When it doesn't work** — scratchy audio, ping-pong, double records.

## What you're building

The target is the workhorse of commercial and amateur DMR: a single
conventional repeater — a warehouse fleet, a linked amateur repeater, a
municipal works channel. No trunking, no grants-to-elsewhere: radios transmit
on the repeater's input, the repeater retransmits on its output, and your rig
camps that output frequency full-time.

What makes it interesting is the TDMA. The repeater's carrier alternates
30 ms bursts between timeslot 1 and timeslot 2, and the two slots are fully
independent channels — dispatch on TS1 while two techs talk on TS2. A correct
decoder therefore has to *demultiplex before it decodes voice*: pull each
call's own slot cadence out of the interleaved stream, decode its AMBE+2
superframes separately, and keep two recordings open at once. That's exactly
what the rebuilt Tier II path does, and it's on by default.

<figure class="lab-figure">
<svg viewBox="0 0 680 235" width="680" height="235" role="img" aria-label="A conventional DMR carrier timeline showing alternating 30 millisecond bursts, timeslot 1 bursts accented and timeslot 2 bursts plain, interleaved on one carrier; below, the interleaved stream splits into two independent call chains routed by each superframe's embedded Link Control talkgroup, producing two simultaneous recordings, talkgroup 9 dispatch and talkgroup 12 operations">
  <text x="14" y="26" fill="currentColor" font-size="11" font-weight="bold">one carrier, alternating 30 ms bursts</text>
  <line x1="14" y1="62" x2="666" y2="62" stroke="var(--fg-muted)"/>
  <rect x="30" y="40" width="72" height="22" fill="none" stroke="var(--accent)" stroke-width="2"/>
  <text x="66" y="55" text-anchor="middle" fill="var(--accent)" font-size="10">TS1</text>
  <rect x="106" y="40" width="72" height="22" fill="none" stroke="currentColor"/>
  <text x="142" y="55" text-anchor="middle" fill="currentColor" font-size="10">TS2</text>
  <rect x="182" y="40" width="72" height="22" fill="none" stroke="var(--accent)" stroke-width="2"/>
  <text x="218" y="55" text-anchor="middle" fill="var(--accent)" font-size="10">TS1</text>
  <rect x="258" y="40" width="72" height="22" fill="none" stroke="currentColor"/>
  <text x="294" y="55" text-anchor="middle" fill="currentColor" font-size="10">TS2</text>
  <rect x="334" y="40" width="72" height="22" fill="none" stroke="var(--accent)" stroke-width="2"/>
  <text x="370" y="55" text-anchor="middle" fill="var(--accent)" font-size="10">TS1</text>
  <rect x="410" y="40" width="72" height="22" fill="none" stroke="currentColor"/>
  <text x="446" y="55" text-anchor="middle" fill="currentColor" font-size="10">TS2</text>
  <text x="560" y="55" fill="var(--fg-muted)" font-size="10">… 60 ms period</text>
  <line x1="218" y1="62" x2="180" y2="120" stroke="var(--accent)"/>
  <line x1="294" y1="62" x2="480" y2="120" stroke="currentColor"/>
  <rect x="60" y="120" width="240" height="52" rx="4" fill="none" stroke="var(--accent)"/>
  <text x="180" y="139" text-anchor="middle" fill="var(--accent)" font-size="10">call A — embedded LC says TG 9</text>
  <text x="180" y="154" text-anchor="middle" fill="var(--fg-muted)" font-size="9">own superframe cadence → AMBE+2</text>
  <text x="180" y="167" text-anchor="middle" fill="var(--fg-muted)" font-size="9">ends on its own Terminator-with-LC</text>
  <rect x="380" y="120" width="240" height="52" rx="4" fill="none" stroke="currentColor"/>
  <text x="500" y="139" text-anchor="middle" fill="currentColor" font-size="10">call B — embedded LC says TG 12</text>
  <text x="500" y="154" text-anchor="middle" fill="var(--fg-muted)" font-size="9">decoded in parallel, second tap</text>
  <text x="500" y="167" text-anchor="middle" fill="var(--fg-muted)" font-size="9">unaffected by TS1's teardown</text>
  <text x="180" y="196" text-anchor="middle" fill="var(--accent)" font-size="10">recordings/…/9/….wav</text>
  <text x="500" y="196" text-anchor="middle" fill="currentColor" font-size="10">recordings/…/12/….wav</text>
  <text x="340" y="224" text-anchor="middle" fill="var(--fg-muted)" font-size="10">the wire never labels a burst's physical slot — the talkgroup inside the embedded LC is the routing key</text>
</svg>
<figcaption>Two-slot demultiplex: the interleaved carrier splits into two independent call chains, each identified by its embedded-LC talkgroup — because base-station bursts carry no trustworthy physical slot number.</figcaption>
</figure>

## The shopping list

Identical to [Part 1]({{ '/blog/tutorials/operator-cookbook-01-forty-dollar-p25-rig/' | relative_url }}):
one ~$35 RTL-SDR, the kit whip, any computer. This recipe actually *uses less*
of it — no wideband window juggling, no voice taps to size. Conventional DMR
is the friendliest possible target for the
[starter checklist]({{ '/what-do-i-need-for-gophertrunk/' | relative_url }})
hardware, and most repeaters run enough power that the stock whip indoors is
plenty within a few miles.

## How two slots become two calls

Three facts about the decode path, because they explain both the config and
the troubleshooting table.

**First: interleaved decode is the default.** The `dmr_interleaved_voice`
key is a tri-state override; unset, DMR systems get the interleaved two-slot
decoder, which auto-detects each call's on-air cadence (with or without an
inter-burst [CACH]({{ '/reference/dmr-cach/' | relative_url }})) and pulls its
slot out of the stream. The single-slot decoder that used to splice both
slots into one call — the "DJ scratchy" audio of issue #644 — is now the
thing you'd have to opt *into*.

**Second: identity is per-destination, not per-slot.** The base-station wire
format shares its sync words across both slots and the slot-type field
carries only colour code + data type — so GopherTrunk assigns concurrent
calls a *synthetic* timeslot (1/2) as an engine identity token and routes
audio by the embedded Link Control's talkgroup. Two consecutive
transmissions on different talkgroups become two calls even if the repeater
juggles them across physical slots.

**Third: teardown respects the boundary.** A
Terminator-with-LC decodes its own destination, so a TS1 terminator releases
only the TS1 call. When a terminator's LC *doesn't* decode and two calls are
active, GopherTrunk deliberately releases nothing — a guess could tear down
the wrong conversation — and lets each call's own hangtime
(`trunking.voice_hangtime_ms`, default 3.5 s) close it. One active call with
an undecodable terminator still tears down promptly, keeping the snappy
single-call behaviour.

## The config

The shortest complete config in this series — every key verified against
`config.example.yaml`:

```yaml
storage:
  path: "../data/calls.db"

recordings:
  dir: "../recordings"

sdr:
  sample_rate: 2_400_000
  devices:
    - serial: "00000001"
      role: control
      gain: "auto"

trunking:
  systems:
    - name: "Ridge-Repeater"
      protocol: dmr-tier2
      control_channels:
        - 452_662_500      # the repeater's OUTPUT frequency
      talkgroup_file: "../config/talkgroups-dmr.csv"   # optional
      # color_code: 3      # optional: drop bursts from co-channel systems
      # dmr_interleaved_voice: false   # force single-slot (don't, normally)
```

Notes. `control_channels` here means "the carrier to camp" — the cc-hunt
supervisor treats a conventional system as a channel to sit on and wait, not
hunt. **No voice device appears anywhere**: because Tier II voice rides the
same carrier the state machine decodes, the daemon registers two
same-carrier voice taps for the system automatically — one per timeslot. And
`color_code` is worth setting once you know your repeater's
[colour code]({{ '/reference/color-code/' | relative_url }}): unset,
GopherTrunk accepts and reports whatever it reads off the air, which is right
for exploration and wrong on a shared frequency where a distant co-channel
system on a different colour code would pollute your call log.

## First run — what healthy looks like

Start the daemon. On a quiet repeater the first sign of life is the camp
announcement and, if the repeater beacons idle CSBKs, a periodic keepalive:

```
INF cchunt: camped on conventional channel — idle, waiting for traffic
INF dmr/tier2 site alive (beacon) freq=452662500 cc=3 csbk=... system=Ridge-Repeater
```

The lock line fires on the first decoded voice activity:

```
INF dmr/tier2 cc locked freq=452662500 cc=3 system=Ridge-Repeater
INF recorder: call started device=cc:same-carrier:1 wav=../recordings/Ridge-Repeater/9/... tg=9 provoice=false vocoder=ambe2-dmr
```

And the moment this build exists for — a busy afternoon, both slots keyed:

```
INF recorder: call started device=cc:same-carrier:1 ... tg=9 ...
INF recorder: call started device=cc:same-carrier:2 ... tg=12 ...
INF recorder: call ended ... duration=6.32s reason=released
```

Two `call started` lines with different talkgroups, overlapping in time, each
ending on its own terminator (`reason=released`) or hangtime
(`reason=timeout`). In the web console, **Active** shows both calls
simultaneously; at `log.level: debug` you can watch the raw grants
(`dmr/tier2: grant … dst=9 src=2054 individual=false`) and terminators
(`dmr/tier2: terminator dst=9 slot=1`) drive the lifecycle.

Honesty checkpoint, in the spirit this blog keeps repeating: the two-slot
path is pinned by failing-first regression tests against synthetic
interleaved carriers, but its **on-air A/B against a real concurrent-traffic
capture is still pending** — the only IQ grab contributed so far was
undecodable (~−75 dBFS). Green synthetic ≠ on-air correct is
[this project's most expensive lesson]({{ '/blog/solution-postmortem/from-the-issue-tracker-20-self-consistent-trap/' | relative_url }}),
so treat this recipe as *new in this release*: run it, and if your repeater
produces anything from the table below, a
[capture]({{ '/decoder-capture-needs.html' | relative_url }}) is the most
useful bug report there is.

## When it doesn't work

| Symptom | Likely cause | Fix |
|---|---|---|
| "DJ scratchy" audio — two voices chopped together in one file | both slots spliced into one call: the pre-rebuild behaviour, or `dmr_interleaved_voice: false` forced | Remove the override (default is interleaved-on); if it persists on defaults, capture IQ and report — that's the exact #644 signature the rebuild targets |
| One recording ping-pongs between two talkgroups | slot demux failing, calls being folded to one identity | Same as above — this is the second face of the same bug class; verify you're on a current build |
| Same transmission recorded **twice**, once per tap | embedded LC never decodes, so both slot routers fall back to phase parity and can bind the same phase | Known sharp edge (#644 family): weak signal usually underlies the LC failures — improve RF first, and report the capture |
| No lock, but the repeater is definitely transmitting | wrong frequency (input vs output), or a colour-code pin mismatch | Camp the repeater's **output**; unset `color_code` while diagnosing so GT reports what it actually reads (`cc=` in the lock line) |
| Bursts decode, calls log, but from a system 40 miles away | co-channel sharing — DMR reuses frequencies aggressively | Now set `color_code:` to your system's value; wrong-colour bursts are dropped before they grant or lock |
| Calls end several seconds after the voice clearly stopped | terminator LC undecodable with two calls active — teardown deferred to hangtime by design | Expected under weak signal; lower `voice_hangtime_ms` if the tail bothers you, at the cost of splitting long pauses |
| Constant `dmr/tier2: CSBK CRC mismatch (between-beacon noise)` at debug | normal — noise between transmissions probed and rejected | Nothing; that's the decoder declining to invent traffic |

### How this recipe shapes operator practice

- **Concurrent `call started` lines are the health check.** One busy hour
  with both slots active proves the whole demux; a repeater that never shows
  two concurrent calls might just be single-slot-provisioned — check with a
  local before suspecting the decoder.
- **Unset the colour code to diagnose, set it to operate.** GT reporting
  `cc=` off the air is your measurement; the pin is your filter.
- **New paths earn trust through your captures.** The regression suite proves
  the code against the bug it fixed; only the field proves it against radios.

## Variations

- **Several repeaters, one dongle.** If multiple conventional carriers fit in
  one 2.4 MHz window, use a `role: wideband` device with one `channels:`
  entry per repeater, each pointing at its own `protocol: dmr-tier2` system —
  every carrier gets its own state machine, decoded in parallel.
- **Tier I / direct mode.** `protocol: dmr-tier1` covers 446 MHz
  license-free DMR — same family, single-slot by design
  ([dmr-tier-1]({{ '/reference/dmr-tier-1/' | relative_url }})), and it stays
  on the single-slot voice path.
- **AMBE frame sidecars.** DMR calls always write a `.raw` vocoder-frame
  sidecar; add `recordings.mbe_files: true` for DSD-FME-playable `.amb`
  files if you want to A/B GopherTrunk's vocoder against mbelib — the
  workflow [vocoders.md]({{ '/vocoders.html' | relative_url }}) documents.
- **Names over numbers.** The same talkgroup CSV from Part 2 works here;
  conventional talkgroups especially benefit since the numbers are often
  fleet-internal.

## Where this goes next

Three parts, three protocols from the same 4-level-FSK family. [Part
4]({{ '/blog/tutorials/operator-cookbook-04-tetra-tmo/' | relative_url }})
leaves it entirely: TETRA — π/4-DQPSK, a continuously transmitting four-slot
downlink, a clean-room ACELP vocoder, and a control-channel equalizer that
ships enabled because real networks needed it. Different modulation, same
recipe skeleton.

## FAQ

**Can one SDR dongle decode both DMR timeslots at the same time?**
Yes — both slots live on one 12.5 kHz carrier, so one dongle captures
everything. The work is demultiplexing, not bandwidth: GopherTrunk's
interleaved decoder separates each call's burst cadence and decodes two AMBE+2
streams in parallel from the same capture.

**Why does my DMR recording contain two conversations chopped together?**
That's the classic single-slot-decoder-on-a-two-slot-carrier failure: both
timeslots' voice frames sliced into one superframe stream. In current
GopherTrunk the interleaved decoder is the DMR default, so if you hear it,
check that `dmr_interleaved_voice` isn't forced `false` — and if it happens
on defaults, report it with IQ.

**Do I need to know my repeater's colour code before configuring it?**
No — leave `color_code` unset and GopherTrunk decodes and logs the colour
code it hears (`cc=3` in the lock line). Once confirmed, pin it: on shared
frequencies the pin keeps a co-channel system on another colour code out of
your call log entirely.

**How does GopherTrunk know which timeslot a burst belongs to?**
Strictly speaking, it doesn't — a base-station burst carries no reliable
physical slot label. It separates calls by their decoded identity instead:
each concurrent destination gets its own call with a synthetic slot token,
and voice routes by the embedded Link Control's talkgroup. The physical slot
number is the one thing this rig never actually needs.

**Is conventional DMR the same as MOTOTRBO?**
MOTOTRBO is Motorola's product line built on the DMR standard; a MOTOTRBO
conventional repeater is exactly what this recipe decodes. IPSC-linked
multi-repeater MOTOTRBO systems work per-carrier too — each repeater is just
another `dmr-tier2` entry.

## Series navigation

**Part 3 of 14** · ←
[Part 2: A DMR Tier III Network, End to End]({{ '/blog/tutorials/operator-cookbook-02-dmr-tier3/' | relative_url }})
· Next →
[Part 4: A TETRA TMO Rig]({{ '/blog/tutorials/operator-cookbook-04-tetra-tmo/' | relative_url }})
