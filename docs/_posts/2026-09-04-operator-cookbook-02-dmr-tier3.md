---
title: "The Operator's Cookbook, Part 2: A DMR Tier III Network, End to End"
description: A complete GopherTrunk build for trunked DMR — one wideband dongle on a Tier III control channel, the LCN-to-frequency band plan problem (and letting GopherTrunk learn the plan off the air), plus the log lines, colour codes and timeslots that tell you it's working.
category: tutorials
keywords: dmr tier 3 scanner setup, dmr trunking decoder, dmr lcn frequency mapping, dmr band plan config, capacity max scanner sdr, dmr colour code, gophertrunk dmr config, dmr csbk grants, gophertrunk cookbook
tags: [operator-cookbook, dmr, tier-3, trunking, lcn, config]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "The Operator's Cookbook"
series_part: 2
---

*Part 2 of **The Operator's Cookbook**, a 14-part series of complete,
copy-paste GopherTrunk builds — one working rig per part, antenna to browser.
[Part 1]({{ '/blog/tutorials/operator-cookbook-01-forty-dollar-p25-rig/' | relative_url }})
built the $40 P25 starter and taught you the healthy-rig heartbeat: lock,
grant, recording. This part points the same hardware at trunked DMR — Tier III
— where one new problem changes the whole recipe: the control channel never
tells you a frequency. It hands out **logical channel numbers**, and until
those resolve to hertz, every voice grant on the system is a call you can hear
about but not hear.*

> **TL;DR:** Trunked DMR is `protocol: dmr` on a `role: wideband` dongle with
> `voice_taps`. A Tier III voice grant references its traffic channel by a
> 12-bit **LCN**, never a frequency, so the decoder needs a `dmr_band_plan`
> (linear `base_hz`/`spacing_hz`/`offset`, or an explicit table) — **or you
> omit the key and GopherTrunk learns the plan off the air** by correlating
> granted LCNs against which carriers key up in the IQ band
> (`dmrlcn: band plan learned (linear)`), then writes it back into your
> `config.yaml`. Healthy looks like `dmr cc locked freq=… cc=… sysid=…`
> followed by grants resolving to real frequencies; broken looks like grants
> logged and zero recordings, with `decode.error` `stage=no-bandplan`.

**Key takeaways**

- **DMR grants are indirect.** A [Tier III]({{ '/reference/dmr-tier-3/' | relative_url }})
  grant CSBK carries `(LCN, timeslot)`, and the LCN→frequency mapping is
  site-specific configuration the air interface never broadcasts in full. The
  band plan is the recipe's load-bearing ingredient.
- **You can let the rig learn the plan.** Omit `dmr_band_plan` on a wideband
  dongle and GopherTrunk watches grants, detects which carriers key up in
  response, and fits base + spacing for the whole LCN enumeration — then
  persists it.
- **A wrong plan fails quietly; a missing one fails loudly.** No plan drops
  grants with `stage=no-bandplan`. A plan with the wrong offset tunes taps to
  live carriers *the wrong calls* are on — the nastier failure this post's
  table covers.
- **Timeslots come for free here.** Each Tier III carrier is 2-slot TDMA and
  grants name their slot; the two-slot interleaved voice decoder is the
  default for DMR, so concurrent calls on one carrier resolve cleanly.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| System definition | trunked DMR control channel | `trunking.systems[]` with `protocol: dmr` |
| LCN → frequency | linear grid or explicit table | `dmr_band_plan.linear` (`base_hz`, `spacing_hz`, `offset`) or `dmr_band_plan.table` |
| Auto-learning | fit the plan from grants vs. keyed carriers | omit `dmr_band_plan` + `role: wideband`; [LCN correlation deep dive]({{ '/blog/deep-dives/the-hunt-08-dmr-lcn-correlation/' | relative_url }}) |
| Voice on one dongle | per-grant DDC taps in the IQ window | `sdr.devices[].voice_taps` |
| Colour code | per-site interference discriminator, decoded off the CC | [color code]({{ '/reference/color-code/' | relative_url }}) — no config key needed for Tier III |
| Talkgroup names | RadioReference-style CSV | `talkgroup_file` |
| Protocol background | how the CSBK chain works | [Protocol Decoders Part 5]({{ '/blog/deep-dives/protocol-decoders-05-dmr-tier-2-3/' | relative_url }}) |

## In this post

- **What you're building** — the Part 1 rig, retargeted at a Tier III network.
- **The LCN problem** — why DMR needs a band plan and P25 mostly didn't.
- **The config** — with the band plan omitted on purpose.
- **First run — what healthy looks like** — lock, learning, grants, calls.
- **When it doesn't work** — the wrong-offset trap and friends.
- **Variations** — explicit plans, tables, and multi-carrier realities.

## What you're building

Hardware is unchanged from [Part 1]({{ '/blog/tutorials/operator-cookbook-01-forty-dollar-p25-rig/' | relative_url }}):
one RTL-SDR, the whip, a computer. The target is different — a DMR Tier III
trunked network (Hytera and Motorola sell these as Capacity Max-class systems;
utilities, transit and commercial fleets run them everywhere). Like P25, one
control channel governs the system: radios affiliate, request calls, and get
granted a traffic channel. Unlike P25, the grant names that channel as a
**logical channel number** — an index into a numbering plan the system
operator chose — plus a **timeslot**, because every DMR carrier is two-slot
TDMA carrying two independent channels.

So the decode chain grows one box the P25 rig didn't have: an LCN resolver
between the grant and the voice tap. Everything downstream — the AMBE+2
vocoder, the recorder, the web console — is the machinery you already ran.

<figure class="lab-figure">
<svg viewBox="0 0 680 250" width="680" height="250" role="img" aria-label="Signal chain of the DMR Tier III rig: antenna and RTL-SDR feed a wideband IQ stream; the control-channel tap decodes CSBK grants carrying LCN and timeslot; a highlighted band-plan box resolves LCN 7 to a frequency, either from config or learned off the air by the LCN correlator; the resolved grant drives a voice tap into the AMBE+2 vocoder and out to recordings and the web console">
  <rect x="8" y="100" width="64" height="34" rx="4" fill="none" stroke="currentColor"/>
  <text x="40" y="121" text-anchor="middle" fill="currentColor" font-size="10">antenna</text>
  <line x1="72" y1="117" x2="100" y2="117" stroke="currentColor"/>
  <rect x="100" y="100" width="76" height="34" rx="4" fill="none" stroke="currentColor"/>
  <text x="138" y="121" text-anchor="middle" fill="currentColor" font-size="10">RTL-SDR</text>
  <line x1="176" y1="117" x2="204" y2="117" stroke="currentColor"/>
  <rect x="204" y="36" width="150" height="36" rx="4" fill="none" stroke="currentColor"/>
  <text x="279" y="51" text-anchor="middle" fill="currentColor" font-size="10">CC tap → CSBK decoder</text>
  <text x="279" y="64" text-anchor="middle" fill="var(--fg-muted)" font-size="9">grant: LCN 7, TS 2, TG 101</text>
  <rect x="204" y="100" width="150" height="46" rx="4" fill="none" stroke="var(--accent)" stroke-width="2"/>
  <text x="279" y="118" text-anchor="middle" fill="var(--accent)" font-size="10">band plan: LCN → Hz</text>
  <text x="279" y="132" text-anchor="middle" fill="var(--fg-muted)" font-size="9">config, or learned (dmrlcn)</text>
  <rect x="204" y="170" width="150" height="36" rx="4" fill="none" stroke="currentColor"/>
  <text x="279" y="185" text-anchor="middle" fill="currentColor" font-size="10">voice tap @ 465.287500</text>
  <text x="279" y="198" text-anchor="middle" fill="var(--fg-muted)" font-size="9">slot-filtered → AMBE+2</text>
  <line x1="279" y1="72" x2="279" y2="100" stroke="var(--accent)"/>
  <polygon points="275,94 279,102 283,94" fill="var(--accent)"/>
  <line x1="279" y1="146" x2="279" y2="170" stroke="currentColor"/>
  <polygon points="275,164 279,172 283,164" fill="currentColor"/>
  <line x1="354" y1="188" x2="420" y2="188" stroke="currentColor"/>
  <rect x="420" y="152" width="240" height="30" rx="4" fill="none" stroke="currentColor"/>
  <text x="540" y="171" text-anchor="middle" fill="currentColor" font-size="10">recordings/Regional-DMR-T3/101/*.wav</text>
  <rect x="420" y="192" width="240" height="30" rx="4" fill="none" stroke="var(--accent)"/>
  <text x="540" y="211" text-anchor="middle" fill="var(--accent)" font-size="10">web console — Active / History / CC</text>
  <line x1="420" y1="167" x2="380" y2="184" stroke="currentColor"/>
  <text x="470" y="60" fill="var(--fg-muted)" font-size="10">the box P25 didn't need:</text>
  <text x="470" y="74" fill="var(--fg-muted)" font-size="10">a grant is not a frequency</text>
  <text x="470" y="88" fill="var(--fg-muted)" font-size="10">until the plan says so</text>
  <text x="340" y="240" text-anchor="middle" fill="var(--fg-muted)" font-size="10">wrong plan ⇒ the tap tunes a real carrier carrying the wrong call — the failure that looks like success</text>
</svg>
<figcaption>The Tier III chain is the P25 chain plus one resolver: every grant passes through the LCN→frequency band plan, so the plan's correctness decides whether voice exists at all.</figcaption>
</figure>

## The LCN problem

Why doesn't the control channel just send frequencies? Because DMR's grant
CSBK has 12 bits for the channel field, and because the protocol was designed
so a site's channel lineup lives in radio codeplugs, not on the air. Every
radio on the system was programmed with the same LCN→frequency table; a
scanner has to reconstruct it. RadioReference sometimes lists LCNs for
documented systems — but the numbering is *per system*, `1`-indexed on some
and `0`-indexed on others, and nothing on the air validates your guess.

GopherTrunk's answer, when you don't know the plan: **watch the system prove
it**. On a wideband dongle, the daemon sees the whole IQ band, so when the
control channel grants LCN 7 and a carrier at 465.2875 MHz keys up 200 ms
later — and that correlation repeats across many grants — the plan fits
itself: base frequency, spacing, offset, with a confidence score. The
correlator, its onset detector, and the false-match defenses are the subject
of [The Hunt Part 8]({{ '/blog/deep-dives/the-hunt-08-dmr-lcn-correlation/' | relative_url }});
here we just use it.

## The config

Every key verified against `config.example.yaml`. Note what's *absent*:

```yaml
storage:
  path: "../data/calls.db"

recordings:
  dir: "../recordings"

sdr:
  sample_rate: 2_400_000
  devices:
    - serial: "00000001"
      role: wideband
      gain: "auto"
      center_freq_hz: 465_500_000   # middle of the site's carrier cluster
      voice_taps: 2
      channels:
        - frequency_hz: 465_237_500 # the Tier III control channel
          system: "Regional-DMR-T3"

trunking:
  systems:
    - name: "Regional-DMR-T3"
      protocol: dmr
      control_channels:
        - 465_237_500
      talkgroup_file: "../config/talkgroups-dmr.csv"   # optional
      # dmr_band_plan: deliberately omitted — learned off the air
```

If you *do* know the plan (from RadioReference or a codeplug), configure it
and learning is disabled — your plan is authoritative:

```yaml
      dmr_band_plan:
        linear:
          base_hz: 465_212_500   # LCN 1's frequency…
          spacing_hz: 12_500     # …12.5 kHz per LCN step
          offset: 1              # 1 ⇒ LCN 1 == base_hz; 0 for 0-indexed sites
```

For sites whose carriers aren't on a regular grid, `table:` takes explicit
`{ lcn, freq_hz }` rows instead. One thing you do **not** configure for
Tier III: the [colour code]({{ '/reference/color-code/' | relative_url }}).
It's decoded straight off the control channel (the `color_code` config key
exists only to pin *conventional* DMR, next part's topic).

## First run — what healthy looks like

Start the daemon and watch for the DMR lock line — it carries the colour code
and the decoded system ID:

```
INF dmr cc locked freq=465237500 cc=1 sysid=4242
```

With no band plan configured, the learner announces itself:

```
INF dmrlcn: learning DMR band plan center_hz=465500000 sample_rate_hz=2400000
```

Now wait for traffic. Every grant during the learning window is a data point;
on a moderately busy system the fit converges in minutes:

```
INF dmrlcn: band plan learned (linear) base_hz=465212500 spacing_hz=12500 offset=1 pairs=9 confidence=0.97 residual_hz=31
```

Watch what that line buys you: it applies **live** (grants that were being
dropped start following voice immediately), and because you started the daemon
with `-config`, the plan is **written back into your config.yaml** so the next
start skips the apprenticeship. From here the rhythm is Part 1's, with DMR
fields — at `log.level: debug` you'll see the raw grants
(`dmr/tier3: tv-grant … tg=101 src=2054 lcn=7 ts=2 freq_hz=465287500`), and at
`info` the familiar pair:

```
INF call started device=… grant=… priority=5
INF recorder: call started … tg=101 … vocoder=ambe2-dmr
```

In the web console, the **CC panel** streams CSBK activity, and **History**
fills with calls whose vocoder column reads `ambe2-dmr`. If DMR audio sounds
different from your P25 rig's — a bit bright, a bit synthetic — that's a known,
measured property of the software AMBE+2 path, not your antenna; the
[DMR voice quality page]({{ '/dmr-voice-quality.html' | relative_url }}) and
[Voice Coding Part 7]({{ '/blog/deep-dives/voice-coding-07-ambe-plus-2/' | relative_url }})
have the honest details.

## When it doesn't work

| Symptom | Likely cause | Fix |
|---|---|---|
| No `dmr cc locked` at all | wrong frequency, or it's a conventional (Tier II) repeater, not a trunked CC | Verify against RadioReference; run `gophertrunk hunt -candidates` to identify what's actually there. Tier II is [Part 3]({{ '/blog/tutorials/operator-cookbook-03-conventional-dmr-two-slots/' | relative_url }}) |
| CC locked, grants visible, **zero recordings**; `decode.error` events with `stage=no-bandplan` | no band plan and no wideband dongle to learn one from | Use `role: wideband` (learning needs the whole band in view), or configure `dmr_band_plan` explicitly |
| Learner runs but never prints `band plan learned` | quiet system (too few grants), or voice carriers fall outside the 2.4 MHz window | Be patient on quiet systems; re-center `center_freq_hz` on the carrier cluster, not the CC |
| Recordings exist but carry the **wrong conversations** for their talkgroup | right frequencies, wrong LCN order — usually `offset: 1` vs `offset: 0`, or a table sorted by frequency instead of LCN | Flip the offset, or delete the plan and let the correlator re-derive it from the air — the learned fit is empirical, not guessed |
| Two talkgroups' audio interleaved or ping-ponging in one file | interleaved two-slot decode forced off | `dmr_interleaved_voice` defaults **on** for DMR — remove any `false` override; see Part 3 for the slot-routing story |
| Calls decode but tear down late | nothing wrong — DMR teardown rides terminator + hangtime | Tune `trunking.voice_hangtime_ms` (default 3500) if you want tighter files |

The wrong-offset row deserves emphasis because it's this series' first
instance of a recurring villain: **a config that's self-consistently wrong
looks exactly like success**. Every tap tunes a real, keyed carrier; audio
decodes; files land on disk — and every file is mislabeled. The only check
that catches it is external validation: listen to a call against its talkgroup
name, or trust the learned fit over a transcribed one. The deep-dive series
tells the same story about test vectors —
[green synthetic ≠ on-air correct]({{ '/blog/solution-postmortem/from-the-issue-tracker-20-self-consistent-trap/' | relative_url }}) —
and it starts applying to *configs* here.

### How this recipe shapes operator practice

- **Prefer learned plans to transcribed ones.** The correlator's plan is
  validated by the system's own behaviour; a forum post's LCN table is not.
- **Center on the cluster, not the CC.** The control channel only needs to be
  *inside* the window; the voice carriers need to be too.
- **Debug level is your CSBK microscope.** `log.level: debug` shows every
  grant with its LCN, slot and resolved frequency — the first thing to read
  when recordings and reality disagree.

## Variations

- **Explicit table plan.** Non-grid sites: `dmr_band_plan.table` with one
  `{ lcn, freq_hz }` row per channel. Pin it from a learned run's output if
  the linear fit's `residual_hz` is high.
- **Second dongle for out-of-window voice.** Keep the wideband CC tap, add a
  `role: voice` device; grants the window can't serve spill over to it
  automatically.
- **Scan-list mode.** With a talkgroup CSV loaded, `scanner.scan_mode: list`
  follows only talkgroups marked `Scan` — the right posture on a huge fleet
  system where you care about ten groups out of hundreds.
- **Encrypted fleets.** DMR Enhanced Privacy (RC4) traffic can be decrypted
  *only* with operator-supplied keys via the system's `encryption_keys` list —
  GopherTrunk does no key recovery. Without keys, encrypted calls still log
  their metadata.

## Where this goes next

Tier III assumed a trunked network with a control channel to camp. Most DMR
in the wild is humbler: one conventional repeater, no trunking, two timeslots
carrying two independent conversations at once. [Part
3]({{ '/blog/tutorials/operator-cookbook-03-conventional-dmr-two-slots/' | relative_url }})
builds that rig — including the newly rebuilt two-slot decode path that turns
"one garbled call" into two clean ones, and the honest edge cases that ships
with.

## FAQ

**What is an LCN in DMR and why do I need a band plan?**
A Logical Channel Number is the index a Tier III grant uses to name a traffic
channel — the air interface never sends the frequency itself. The band plan is
the LCN→frequency mapping the system's radios were programmed with; without
it, a scanner can decode every grant and still tune nothing.

**Can GopherTrunk figure out the LCN order by itself?**
Yes — omit `dmr_band_plan` on a system monitored by a `role: wideband` dongle
and the daemon correlates granted LCNs against which carriers key up across
the band, fits base + spacing + offset, applies the plan live, and writes it
back to `config.yaml`. It needs real traffic and the voice carriers inside
the dongle's IQ window.

**Do I need to configure the DMR colour code?**
Not for Tier III — the decoder reads it off the control channel and logs it in
the `dmr cc locked` line (`cc=1`). The `color_code` config key exists to pin
conventional Tier II/Tier I channels, where it acts like a squelch
discriminator between co-channel systems.

**Why do my DMR recordings sound harsher than P25?**
The software AMBE+2 decoder for DMR's 3600×2450 mode currently reproduces
less high-band energy than reference decoders — a measured, localized
deficit, not an RF problem. `recordings.enhance.enabled: true` shapes the
audio louder and cleaner; the underlying vocoder work is tracked openly in
the [voice-coding series]({{ '/blog/deep-dives/voice-coding-07-ambe-plus-2/' | relative_url }}).

**Does one dongle really handle a whole Tier III site?**
If the site's carriers fit inside ~2.4 MHz — common for single-site systems —
yes: the CC tap and `voice_taps: 2` decode control plus two concurrent calls
from one capture. Multi-carrier sites spread wider need the spill-over voice
dongle, and multi-*system* rigs are [Part 7's]({{ '/blog/series/operator-cookbook/' | relative_url }}) territory.

## Series navigation

**Part 2 of 14** · ←
[Part 1: The $40 P25 Starter Rig]({{ '/blog/tutorials/operator-cookbook-01-forty-dollar-p25-rig/' | relative_url }})
· Next →
[Part 3: One Repeater, Two Conversations — Conventional DMR]({{ '/blog/tutorials/operator-cookbook-03-conventional-dmr-two-slots/' | relative_url }})
