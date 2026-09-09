---
title: "The Operator's Cookbook, Part 6: Analog FM & Tone-Out Paging"
description: A complete GopherTrunk recipe for conventional analog FM — a real scan list with squelch, CTCSS/DCS tone gating and per-channel hangtime, plus Quick Call II two-tone fire paging alerts, built on the scanner.conventional and tone_out config sections with every key verified.
category: tutorials
keywords: sdr analog fm scanner, fire dispatch scanner sdr, two tone paging decoder, quick call 2 tone out, ctcss squelch sdr scanner, conventional scan list config, marine vhf sdr monitor, rtl-sdr fire tones, gophertrunk cookbook
tags: [operator-cookbook, analog-fm, tone-out, ctcss, scanning, config]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "The Operator's Cookbook"
series_part: 6
---

*Part 6 of **The Operator's Cookbook**, a 14-part series of complete,
copy-paste GopherTrunk builds — one working rig per part, antenna to browser.
[Part 5]({{ '/blog/tutorials/operator-cookbook-05-tetra-dmo/' | relative_url }})
decoded TETRA radios talking directly to each other; this part drops the last
digital layer entirely. Plenty of the traffic worth hearing — volunteer fire
dispatch, marine VHF, GMRS repeaters, race crews — is still **plain analog
FM**, and GopherTrunk scans it the way a hardware scanner does: a channel
list, real squelch, sub-audible tone gating, and a two-tone detector that
fires an alert the instant your station's pager tones hit the air. One
honest reshaping up front: analog channels live in their own config section,
not under `trunking.systems` — this recipe is built on what actually exists.*

> **TL;DR:** Analog FM is a **scan list, not a trunked system**: channels go
> under `scanner.conventional` (label, `frequency_hz`, `mode: fm|nfm`,
> `squelch_dbfs`, `hangtime_ms`, per-channel CTCSS/DCS `tone:` gating), and
> the scanner drives a dedicated **`role: voice`** SDR — the last voice
> device in the pool. Each open-squelch dwell becomes a synthetic call
> (`synthetic call started` in the log), recorded like any digital call. On
> top of the decoded audio, `tone_out.profiles` runs a Goertzel two-tone
> detector for Quick Call II fire paging and logs
> `toneout: profile matched`. Pin a `talkgroup_id` per channel or your call
> log's IDs shift when you reorder the list (#1105).

**Key takeaways**

- **Conventional analog is its own config section.** `scanner.conventional`
  is a fixed-frequency scan list with per-channel squelch, priority and
  hangtime — deliberately shaped like the scan list in a hardware scanner,
  not like a trunked system.
- **The scanner needs a voice SDR to own.** It takes the *last* `role: voice`
  device in the pool; with none configured the daemon logs a WARN and skips
  the scanner entirely — the most common "why is my scan list dead" cause.
- **Tone gating is what makes shared frequencies usable.** With
  [CTCSS]({{ '/reference/ctcss/' | relative_url }}) or
  [DCS]({{ '/reference/dcs/' | relative_url }}) configured, the scanner
  dwells only when carrier *and* the right tone are present, so a
  co-channel system two counties over can't hold your scanner hostage.
- **Tone-out listens to audio, not RF.** The two-tone detector runs on
  decoded PCM from any voice device — so it catches pages on your analog
  scan list *and* on digital talkgroups alike, scoped by profile.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| The scan list | fixed analog channels, scanned in order | `scanner.conventional[]` |
| Squelch | carrier gate in dBFS, per channel | `squelch_dbfs` ([squelch]({{ '/reference/squelch/' | relative_url }})) |
| Tone gate | require CTCSS/DCS before dwelling | `tone: {mode, ctcss_hz, dcs_code}` |
| Stable call-log IDs | pin the channel's talkgroup number | `talkgroup_id` (else synthetic, shifts on reorder — #1105) |
| Fire paging alerts | Quick Call II two-tone sequential detect | `tone_out.profiles[]` |
| Live audio | hear dwells on the host speakers | `audio.enabled: true` |
| Where the frequencies are | US service-by-service listings | [scanner frequencies guide]({{ '/scanner-frequencies/' | relative_url }}) |

## In this post

- **What you're building** — a software scanner with a pager bolted on.
- **The shopping list** — one dongle, one band-appropriate antenna.
- **The config** — scan list + tone-out profiles, every key verified.
- **First run — what healthy looks like** — dwells, synthetic calls, a matched page.
- **When it doesn't work** — symptom → cause → fix.
- **Variations** — priority channels, manual VFO, digital-side tone-out.

## What you're building

The finished rig scans a list of conventional analog channels — say a
volunteer fire dispatch frequency, the county EMS channel, and
[marine VHF 16]({{ '/reference/marine-vhf/' | relative_url }}) — the way a
$150 hardware scanner does: tune, measure squelch, move on; when a carrier
(and, if configured, its tone) appears, **dwell**, demodulate FM, and record
until the carrier drops plus a hangtime. Every dwell becomes a call in the
History panel with a WAV on disk, indistinguishable from the digital calls
of Parts 1–5 except for its protocol tag (`fm-conv`).

Under the hood this path is simpler than anything else in the series — FM
discriminator to PCM, no vocoder — but it earned its own postmortem: the
conventional channel used to be
[its own voice channel]({{ '/blog/solution-postmortem/from-the-issue-tracker-16-conventional-fm-broker/' | relative_url }})
in a way the IQ plumbing didn't expect, and the broker fix from that story
is why live monitoring, recording and the tone-out detector all listen to
the same dwell today. The wider conventional architecture is
[Protocol Decoders Part 9]({{ '/blog/deep-dives/protocol-decoders-09-conventional-wideband/' | relative_url }})'s
territory.

The second half of the build is **tone-out**: US fire/EMS dispatch still
pages stations with Motorola Quick Call II — an A tone (~1 s) then a longer
B tone. GopherTrunk runs a Goertzel detector across decoded voice PCM and
fires a `KindToneAlert` event when a profile matches — surfaced in the web
**Tones** panel and the log. (The encoder side of this audio machinery
lives in
[Voice Coding Part 11]({{ '/blog/deep-dives/voice-coding-11-recording-encoding/' | relative_url }}).)

<figure class="lab-figure">
<svg viewBox="0 0 680 240" width="680" height="240" role="img" aria-label="Signal chain of the analog FM and tone-out rig: an antenna feeds one voice-role SDR owned by the conventional scanner, whose loop tunes each listed channel, measures squelch and checks the CTCSS or DCS tone gate; an open gate starts a dwell that runs FM demodulation to PCM, feeding three consumers in parallel — the WAV recorder as a synthetic call, the Goertzel two-tone detector that raises tone alerts, and live audio in the web console">
  <rect x="8" y="90" width="60" height="34" rx="4" fill="none" stroke="currentColor"/>
  <text x="38" y="111" text-anchor="middle" fill="currentColor" font-size="10">antenna</text>
  <line x1="68" y1="107" x2="98" y2="107" stroke="currentColor"/>
  <rect x="98" y="90" width="88" height="34" rx="4" fill="none" stroke="currentColor"/>
  <text x="142" y="105" text-anchor="middle" fill="currentColor" font-size="10">SDR</text>
  <text x="142" y="118" text-anchor="middle" fill="var(--fg-muted)" font-size="9">role: voice</text>
  <line x1="186" y1="107" x2="216" y2="107" stroke="currentColor"/>
  <rect x="216" y="16" width="212" height="208" rx="6" fill="none" stroke="var(--fg-muted)" stroke-dasharray="4 3"/>
  <text x="322" y="32" text-anchor="middle" fill="var(--fg-muted)" font-size="10">scanner.conventional</text>
  <rect x="230" y="44" width="184" height="32" rx="4" fill="none" stroke="currentColor"/>
  <text x="322" y="58" text-anchor="middle" fill="currentColor" font-size="10">scan loop: tune next channel</text>
  <text x="322" y="70" text-anchor="middle" fill="var(--fg-muted)" font-size="9">ch1 → ch2 → ch3 → …</text>
  <rect x="230" y="90" width="184" height="32" rx="4" fill="none" stroke="var(--accent)"/>
  <text x="322" y="104" text-anchor="middle" fill="var(--accent)" font-size="10">squelch_dbfs + tone gate</text>
  <text x="322" y="116" text-anchor="middle" fill="var(--fg-muted)" font-size="9">carrier AND CTCSS/DCS</text>
  <rect x="230" y="136" width="184" height="32" rx="4" fill="none" stroke="currentColor"/>
  <text x="322" y="150" text-anchor="middle" fill="currentColor" font-size="10">dwell: FM demod → PCM</text>
  <text x="322" y="162" text-anchor="middle" fill="var(--fg-muted)" font-size="9">until carrier drops + hangtime_ms</text>
  <line x1="322" y1="76" x2="322" y2="90" stroke="currentColor"/>
  <line x1="322" y1="122" x2="322" y2="136" stroke="var(--accent)"/>
  <line x1="414" y1="152" x2="452" y2="60" stroke="currentColor"/>
  <line x1="414" y1="152" x2="452" y2="152" stroke="currentColor"/>
  <line x1="414" y1="152" x2="452" y2="204" stroke="currentColor"/>
  <rect x="452" y="42" width="218" height="34" rx="4" fill="none" stroke="currentColor"/>
  <text x="561" y="56" text-anchor="middle" fill="currentColor" font-size="10">recorder: synthetic call → WAV</text>
  <text x="561" y="69" text-anchor="middle" fill="var(--fg-muted)" font-size="9">History row, tg from talkgroup_id</text>
  <rect x="452" y="134" width="218" height="34" rx="4" fill="none" stroke="var(--accent)"/>
  <text x="561" y="148" text-anchor="middle" fill="var(--accent)" font-size="10">tone_out: Goertzel A/B match</text>
  <text x="561" y="161" text-anchor="middle" fill="var(--fg-muted)" font-size="9">KindToneAlert → Tones panel</text>
  <rect x="452" y="188" width="218" height="32" rx="4" fill="none" stroke="currentColor"/>
  <text x="561" y="208" text-anchor="middle" fill="currentColor" font-size="10">live audio (web / speakers)</text>
</svg>
<figcaption>One voice SDR, three consumers of the same dwell: the recorder files it as a call, the Goertzel detector watches the audio for pager tones, and the web console streams it live.</figcaption>
</figure>

## The shopping list

| Item | Price (rough) | Notes |
|---|---|---|
| RTL-SDR Blog V3/V4 | ~$35 | analog FM is the least demanding decode in this series |
| VHF-capable antenna | $0–$30 | fire dispatch and marine live at 150–162 MHz — extend the kit whip fully, or a quarter-wave for the band |
| Computer | $0 | anything; this path is the lightest on CPU too |

Frequencies come free: the
[scanner frequencies guide]({{ '/scanner-frequencies/' | relative_url }})
lists US police/fire/EMS, marine, rail and GMRS allocations by service, and
the [fire & EMS guide]({{ '/fire-ems-scanner/' | relative_url }}) covers
which fire traffic tends to stay analog. Tone pairs for tone-out come from
your agency's dispatch records or from listening to a few pages.

## The config

```yaml
log:
  level: info

storage:
  path: "../data/calls.db"

recordings:
  dir: "../recordings"

sdr:
  sample_rate: 2_400_000
  devices:
    - serial: "00000001"      # from `gophertrunk sdr list`
      role: voice             # the conventional scanner drives a Voice SDR
      gain: "auto"

scanner:
  conventional:
    - label: "County Fire Dispatch"
      frequency_hz: 154_190_000
      mode: fm
      squelch_dbfs: -48
      hangtime_ms: 1500
      priority: 3
      talkgroup_id: 2147500001    # pin it — see below
      tone:
        mode: ctcss
        ctcss_hz: 110.9
    - label: "Marine VHF 16"
      frequency_hz: 156_800_000
      mode: nfm
      squelch_dbfs: -50
      hangtime_ms: 1500
      priority: 5
      talkgroup_id: 2147500002

audio:
  enabled: true               # hear dwells on the host speakers

tone_out:
  profiles:
    - name: "station-1-engine"
      alpha_tag: "Station 1 Engine"
      cooldown: "30s"
      tones:
        - frequency_hz: 1042.2
          min_duration: "250ms"
          max_duration: "1500ms"
        - frequency_hz: 1297.4
          min_duration: "2.5s"
          max_duration: "5s"
```

Decisions worth understanding. **`role: voice` is mandatory** — the
conventional scanner steals the *last* voice device in the pool; on a
pure-analog rig like this one, that's your only dongle, which is fine.
**`mode: nfm` vs `fm`** narrows the post-demod audio path for
12.5 kHz-channelized services (most modern licensing); when audio sounds
muffled or clipped, you've likely mismatched the channel's
[deviation]({{ '/reference/fm-deviation/' | relative_url }}) class.
**`squelch_dbfs`** is an absolute IQ-power gate — start around −48 and
tune per channel while watching dwell behavior. **`talkgroup_id`** deserves
its comment: an unset channel gets a positional synthetic ID
(`0x80000000 | list index`) that **shifts when you reorder, insert or remove
channels**, silently orphaning your call-log history and talkgroup-file rows
— issue #1105's lesson. Pin explicit IDs from day one. And the **`tone:`**
block makes the gate require carrier *and* tone: without it, any carrier
above squelch holds the scanner, including a distant system sharing the
frequency.

The tone-out profile is Quick Call II shaped: tone A ~1 s, tone B ~3 s,
with `tolerance_hz` (default 15), `magnitude_threshold` (default 0.05) and
`max_gap` (default 200 ms) available when the defaults misbehave, and a
`cooldown` to stop one long page re-firing the alert.

## First run — what healthy looks like

```sh
gophertrunk run -config config.yaml
```

The scan loop starts silently — an idle list is just the tuner stepping
channels a few times a second. The moment a carrier with the right tone
appears, the dwell shows up as a synthetic call:

```
INF synthetic call started device=00000001 grant=...
INF recorder: call started device=00000001 wav=../recordings/.../2147500001/... tg=2147500001 provoice=false
INF recorder: call ended device=00000001 wav=... duration=8.32s reason=...
INF synthetic call ended device=00000001 reason=...
```

Open the web console: the dwell plays live in **Active** and lands in
**History** under the pinned talkgroup ID. When a page matches a profile:

```
INF toneout: profile matched profile=station-1-engine device=00000001 tones=[1042.2 1297.4]
```

— and the **Tones** panel logs the alert with its alpha tag. If the scanner
has no device to run on, the daemon tells you at startup instead of failing
quietly:

```
WRN daemon: scanner.conventional / manual_tune_enabled configured but no Voice SDRs in the pool; skipping
```

That WARN is the first thing to grep for when the scan list seems dead.

## When it doesn't work

| Symptom | Likely cause | Fix |
|---|---|---|
| Scan list configured, nothing ever dwells, WARN above at startup | no `role: voice` device in the pool | Give the scanner a voice SDR. On a combined trunked+analog rig, add a *second* voice dongle — the scanner takes the last one |
| Scanner parks forever on one channel | squelch open on noise or a constant carrier (data link, remote base) | Raise that channel's `squelch_dbfs`; better, add a `tone:` gate so only your agency's traffic opens it |
| Dwells on someone else's traffic on a shared frequency | carrier squelch alone can't tell users apart | Configure the channel's [CTCSS]({{ '/reference/ctcss/' | relative_url }}) tone or [DCS]({{ '/reference/dcs/' | relative_url }}) code — the gate then needs carrier AND tone |
| `conv: CTCSS detector failed to initialise; tone gate disabled — every signal passes the gate` | bad tone parameters for this channel/rate | Check `ctcss_hz` is a standard EIA tone (67.0–254.1) / `dcs_code` a 3-digit octal; the scanner fails open, so fix it or you're back to carrier squelch |
| Audio muffled, or loud voices clip | deviation/bandwidth mismatch | Flip `mode:` between `fm` and `nfm` to match the service's channelization |
| History IDs changed after editing the list | synthetic positional IDs shifted (#1105) | Pin `talkgroup_id` on every channel — one-time fix, durable forever |
| Pages audible but no `toneout: profile matched` | tone frequencies off, or the B tone shorter than `min_duration` | Verify the pair against dispatch records; widen `tolerance_hz`, relax durations. The detector needs the whole A→B sequence within `max_gap` |
| Alert fires twice per page | long pages re-trigger | Lengthen the profile's `cooldown` |

One inherited lesson applies unchanged from the digital parts:
[gain staging]({{ '/blog/tutorials/analog-edge-03-gain-staging/' | relative_url }})
still decides everything downstream. A squelch threshold tuned against a
badly staged front end is a number about your gain knob, not about the
channel.

### How this recipe shapes operator practice

- **Pin identity early.** `talkgroup_id` and `label` cost nothing on day one
  and save your call history the first time you touch the list.
- **Prefer tone gates to tight squelch.** A tone answers "is it *my*
  agency?"; squelch only answers "is it loud?".
- **Let the WARNs work.** This path fails loudly and open — the startup WARN
  and the tone-gate-disabled WARNs are designed to be grepped, not ignored.

## Variations

- **Manual VFO.** `scanner.manual_tune_enabled: true` builds the scanner
  even with no channel list, so the TUI's `f` key (or
  `POST /api/v1/scanner/manual_tune`) can tune anything ad hoc — a
  software VFO on a spare voice dongle.
- **Priority channels.** Per-channel `priority` is honored by the engine's
  preemption against other calls — set your dispatch channel higher than
  routine traffic (more in [Part 7]({{ '/blog/tutorials/operator-cookbook-07-multi-system-pool/' | relative_url }})).
- **Digital-side tone-out.** Scope a profile with `system` and `group_id`
  to catch pages that go out over a trunked talkgroup.
- **Actual pagers.** For POCSAG/FLEX paging *protocols* (not voice tones),
  the separate `paging:` section dedicates or shares an SDR per paging
  frequency — same rig, different decoder.

## Where this goes next

You now have five kinds of rig and only ever one system per box. [Part
7]({{ '/blog/tutorials/operator-cookbook-07-multi-system-pool/' | relative_url }})
scales *out*: several dongles, several trunked systems, one daemon — device
serials and roles, sizing the shared voice pool against concurrent calls,
and the priority/preemption rules that decide who gets a radio when calls
outnumber tuners.

## FAQ

**Can GopherTrunk scan regular analog FM channels like a police scanner?**
Yes — `scanner.conventional` is a real scan list: fixed frequencies, carrier
squelch in dBFS, optional CTCSS/DCS gating, per-channel hangtime and
priority, with each dwell recorded and logged as a call. It needs one
`role: voice` SDR to drive.

**How do I decode two-tone fire pager tones with an SDR?**
Give GopherTrunk the A/B tone pair in a `tone_out.profiles` entry — the
Goertzel detector watches decoded audio from your channels and fires an
alert (log line, event, web Tones panel) when the sequence matches. Tone
pairs come from your agency's dispatch documentation or from measuring a
recorded page.

**What's the difference between squelch_dbfs and a CTCSS tone gate?**
`squelch_dbfs` opens on any carrier above a power threshold; a CTCSS/DCS
gate additionally requires the sub-audible tone your agency transmits. On a
shared or noisy frequency, tone gating is the difference between a usable
scan list and a scanner held open by strangers.

**Why do my conventional channels show weird talkgroup numbers?**
Unpinned channels get a synthetic ID built from their list position
(`0x80000000 | index`), which changes when the list changes. Set an explicit
`talkgroup_id` per channel to keep History rows and talkgroup-file aliases
stable — issue #1105 exists because reordering silently broke both.

**Can tone-out watch digital talkgroups too?**
Yes — the detector runs on decoded PCM regardless of protocol, and a
profile scoped with `system` and `group_id` matches pages sent over a
trunked talkgroup as readily as over analog dispatch.

## Series navigation

**Part 6 of 14** · ←
[Part 5: TETRA Direct Mode — Scanning DMO]({{ '/blog/tutorials/operator-cookbook-05-tetra-dmo/' | relative_url }})
· Next →
[Part 7: Many Systems, One Box — The SDR Pool]({{ '/blog/tutorials/operator-cookbook-07-multi-system-pool/' | relative_url }})
