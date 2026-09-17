---
title: "The Field Notebook, Part 13: From Log to Issue — What to Paste, What to Capture"
description: How to turn a GopherTrunk debug.log into a bug report that gets fixed — the three blocks to paste, aligning a capture's clock to the log's, which IQ capture answers which question (gophertrunk capture, pre-combine diversity_capture, on_cc_sync_loss), and the verified-language policy that decides when an issue may close.
category: tutorials
keywords: sdr scanner bug report, what to include in a bug report log, iq capture for bug report, gophertrunk capture sidecar, pre-combine diversity capture, on_cc_sync_loss auto record, refs vs closes github issue, issue closing policy verified fix, gophertrunk field notebook
tags: [field-notebook, logs, issues, captures, methodology, tutorial]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "The Field Notebook"
series_part: 13
---

*Part 13 of **The Field Notebook**, a 14-part operator's tutorial that reads
GopherTrunk's `debug.log` one line family at a time — what each field
measures, what a healthy rig prints, which number is a noise meter and which
one means traffic, and which deep dive to open when a line goes wrong.
[Part 12]({{ '/blog/tutorials/field-notebook-12-web-symptoms-log-stories/' | relative_url }})
ended every web story the same way: the daemon line and the API answer for
the same second, pasted beside the symptom. This part makes that the method
— what to paste, which capture answers which question, and the words that
let an issue close as *verified*.*

> **TL;DR:** A report that gets fixed pastes three blocks — the **startup
> block** (banner to first `cc locked`, WARNs included), the protocol's
> **status-line family** across the event (`tetra: decode status`, the DMR
> activity line, the MRC health line), and **30 s around the event** with
> the `siglab: capture started/ended` bracket so the file maps onto log time. Then the capture that matches the
> question: `gophertrunk capture` (IQ + `.metadata.json`, with ppm and
> clipping verdicts), `baseband.auto_record.on_cc_sync_loss` for a lock that
> drops, `sdr.soapy_remote[].diversity_capture` for anything about MRC —
> every other tap is **post**-combine. IQ, never audio, honest rate. The
> close: only when a failing-first regression passes **and** the reporter
> confirms; `.claude/hooks/guard-issue-close.py` asks a human first, and PRs
> say `Refs #N`, not `Closes #N`, until then
> ([#764](https://github.com/MattCheramie/GopherTrunk/issues/764) /
> [#771](https://github.com/MattCheramie/GopherTrunk/issues/771)).

**Key takeaways**

- **Paste the family, not the line.** One WARN is a symptom; the status
  lines around it are the instrument reading.
- **Align clocks before anyone reasons.** `capture started` plus the file's
  `elapsed` turns "the deaf stretch" into a sample offset.
- **The right capture is taken at the right tap.** Post-combine cannot
  answer a combiner question; audio cannot answer a phase question.
- **"Fixed" and "verified" are different words on purpose.** A merged PR is
  a hypothesis; the issue closes when the symptom is gone.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| Startup block | proves config, driver and gain before any decode claim | banner → `daemon:` WARNs → `cc locked` ([Part 1]({{ '/blog/tutorials/field-notebook-01-startup-lines/' | relative_url }})) |
| Capture bracket | start/end lines with `elapsed`, `samples`, `recorded_seconds` | `internal/api/capture.go` |
| One-shot IQ + sidecar | writes `<stem>.metadata.json`; `-sigmf` with `-bundle`; clip / ppm verdicts | `cmd/gophertrunk/capture.go`, `capture_carrier.go` |
| Sync-loss capture | records the re-acquisition IQ when a lock drops | `baseband.auto_record.on_cc_sync_loss` |
| Pre-combine branches | `<prefix>.br0.cs16`, `.br1.cs16`, `.diversity.json` | `sdr.soapy_remote[].diversity_capture` |
| The close | failing-first test passes **and** reporter confirms | `CLAUDE.md`, `.claude/hooks/guard-issue-close.py` |

## In this post

- **What a report is telling the maintainer** — a claim and its evidence.
- **What to paste** — three blocks; a healthy and an unhealthy report.
- **Aligning a capture to the log** — the bracket fields.
- **Which capture to make** — from question to tap.
- **The words that close an issue** — the verified-language policy.

## What a report is telling the maintainer

An issue is a claim — *the rig did X when it should have done Y* — and
GopherTrunk's verification culture
([From Spec to Shipping Part 14]({{ '/blog/deep-dives/from-spec-to-shipping-14-definition-of-verified/' | relative_url }}))
accepts no claim above its evidence. The log turns the claim into a
measurement, and it was *designed* to be pasted:
[Part 13 of that series]({{ '/blog/deep-dives/from-spec-to-shipping-13-instruments-not-logs/' | relative_url }})
argued for branch-complete counters and persistent WARNs so a field log
answers the next question without a round trip. The log cannot supply the
**version**, or the **config** with secrets stripped
([Cookbook 14]({{ '/blog/tutorials/operator-cookbook-14-kitchen-sink-config/' | relative_url }})'s
hygiene rule).

## What to paste

Three blocks; each exists because a real thread stalled without it.

**1. The startup block** — banner to first lock, WARNs included. The 13 Sep
DMO operator's CC sat at −70 dBFS because `gain: 50` meant 5 dB, and the
daemon had said so:

```
WRN daemon: gain looks like dB, not tenths-of-dB — radio may be effectively deaf serial=32B0A4C configured=50 parsed_db=5 did_you_mean=500 hint="gain: is in TENTHS of a dB (\"320\" = 32 dB). ..."
```

Omit it and you invite a week of DSP theory
([Part 1]({{ '/blog/tutorials/field-notebook-01-startup-lines/' | relative_url }})).

**2. The status-line family across the event** — the protocol's periodic
line from minutes *before* the symptom to minutes after, so the trend shows:
`tetra: decode status`, `widebandt2: channel decode activity`, the MRC
health line, `tetra dmo: decode status`. These are DEBUG lines — run at
`log.level: debug` or the report describes a symptom nobody can measure.

**3. Thirty seconds around the event**, with the `siglab: capture
started/ended` bracket if you recorded IQ, and the nearest
`runtime: heartbeat` so CPU starvation can be ruled in or out
([Part 7]({{ '/blog/tutorials/field-notebook-07-overruns-host-drops/' | relative_url }})).

A report that answers, from a TETRA rig losing lock:

```
DBG tetra: decode status system=Metro-TETRA locked=true carrier_off_hz=-412.5 baud=18000 baud_dev_pct=0.01 sb_bursts=72 bsch_ok=72 bsch_fail=0 sysinfo=18 sch_pdus=340 sch_pdus_fail=6 frag_abandons=1 grants=3 colour_code=1
INF siglab: capture started serial=32B0A4C center_hz=467912500 sample_rate_hz=144000 bandwidth_hz=100000 requested_center_hz=467912500 tuner_center_hz=467700000 tuner_rate_hz=2000000 format=flac seconds=60 protocol=tetra path=../captures/tetra_467.9125MHz_144k.flac
DBG tetra: decode status system=Metro-TETRA locked=true carrier_off_hz=-418.0 baud=17998 baud_dev_pct=0.02 sb_bursts=70 bsch_ok=61 bsch_fail=9 sysinfo=14 sch_pdus=88 sch_pdus_fail=41 frag_abandons=4 grants=0 colour_code=1
DBG tetra: dsp resync (signal-time decode drought; reacquiring symbol timing from centre) system=Metro-TETRA
INF siglab: capture ended serial=32B0A4C center_hz=467912500 sample_rate_hz=144000 format=flac samples=8640000 elapsed=1m0.012s path=../captures/tetra_467.9125MHz_144k.flac recorded_seconds=60
```

`bsch_fail` climbing, `sch_pdus` collapsing, a resync, sixty seconds of IQ
around it. The unhealthy twin arrives more often:

```
WRN tetra: payload drought persists across resyncs (sync bursts decoding, no SCH payload) — declaring lock lost to force a re-hunt system=Metro-TETRA attempts=3
```

One WARN, no status lines, no capture. It says the lock was declared lost,
not whether the carrier faded, the AFC latched an alias bucket
([Part 4]({{ '/blog/tutorials/field-notebook-04-carrier-offset-warn/' | relative_url }}))
or the host starved. The next comment will ask for block 2.

## Aligning a capture to the log

A capture and a log are two clocks; the bracket fields map one onto the
other:

| Field | Measures | Healthy | Worry when |
|---|---|---|---|
| `center_hz` / `requested_center_hz` | slice centre vs what you typed | equal | different — the 15 Sep grab was carved at the *tuner* centre |
| `tuner_center_hz`, `tuner_rate_hz` | the stream the slice came from | your `sdr` block | the slice sits at the tuner's edge |
| `sample_rate_hz`, `format` | what the file actually is | matches the sidecar | an odd rate above 65535 Hz with `format=flac` (880029 Hz once aborted the encoder) |
| `seconds` vs `recorded_seconds` | asked vs delivered | equal | short — a `capture aborted` WARN follows |
| `samples`, `elapsed` | length in both units | `samples / sample_rate_hz ≈ elapsed` | mismatch — dropped chunks; check `host_drops` |
| `path` | where the sidecar sits | `<stem>.metadata.json` beside it | renamed — `DiscoverMetadata` finds it by stem |

The alignment is one subtraction: `capture started` is the file's sample
zero, so a log event at *T* sits `(T − started) × sample_rate_hz` samples
in. The 12 Sep IPSC investigation was decided this way — a 20-minute FLAC
aligned to the live log at offset 23:41:31.28 showed the Fire2 tap logging
`sync_hits=0` for six stretches of 180–210 s while the same seconds decoded
every transmission offline. That split "the capture is bad" from "the live
tap went deaf"; `dibits` and `healDeafTier2` followed from it. One trap: the log is local time while the API
and sidecars are UTC
([Part 12]({{ '/blog/tutorials/field-notebook-12-web-symptoms-log-stories/' | relative_url }})).

<figure class="lab-figure">
<svg viewBox="0 0 680 250" width="680" height="250" role="img" aria-label="A debug log timeline with three regions to paste — startup block, status-line family, and thirty seconds around the event bracketed by capture started and ended lines — and beneath it a capture file aligned to that bracket feeding a replay harness, a failing-first test, and a close that waits for the reporter's confirmation.">
  <line x1="20" y1="60" x2="660" y2="60" stroke="currentColor"/>
  <text x="20" y="48" fill="var(--fg-muted)" font-size="9">debug.log</text>
  <rect x="24" y="66" width="120" height="30" rx="4" fill="none" stroke="var(--accent)"/>
  <text x="84" y="85" text-anchor="middle" fill="var(--accent)" font-size="9">1. startup block</text>
  <rect x="170" y="66" width="230" height="30" rx="4" fill="none" stroke="var(--accent)"/>
  <text x="285" y="85" text-anchor="middle" fill="var(--accent)" font-size="9">2. status-line family</text>
  <rect x="426" y="66" width="230" height="30" rx="4" fill="none" stroke="var(--accent)"/>
  <text x="541" y="85" text-anchor="middle" fill="var(--accent)" font-size="9">3. 30 s around the event</text>
  <line x1="450" y1="56" x2="450" y2="130" stroke="currentColor" stroke-dasharray="3 3"/>
  <text x="450" y="140" text-anchor="middle" fill="currentColor" font-size="8">capture started</text>
  <line x1="640" y1="56" x2="640" y2="130" stroke="currentColor" stroke-dasharray="3 3"/>
  <text x="640" y="140" text-anchor="middle" fill="currentColor" font-size="8">capture ended</text>
  <rect x="450" y="150" width="190" height="22" rx="3" fill="none" stroke="currentColor"/>
  <text x="545" y="165" text-anchor="middle" fill="currentColor" font-size="8">IQ file + .metadata.json</text>
  <line x1="450" y1="161" x2="120" y2="161" stroke="var(--fg-muted)"/>
  <polygon points="122,157 114,161 122,165" fill="var(--fg-muted)"/>
  <rect x="20" y="150" width="94" height="22" rx="3" fill="none" stroke="currentColor"/>
  <text x="67" y="165" text-anchor="middle" fill="currentColor" font-size="8">replay harness</text>
  <line x1="67" y1="172" x2="67" y2="196" stroke="var(--fg-muted)"/><polygon points="63,194 67,202 71,194" fill="var(--fg-muted)"/>
  <rect x="20" y="202" width="94" height="22" rx="3" fill="none" stroke="currentColor"/>
  <text x="67" y="217" text-anchor="middle" fill="currentColor" font-size="8">failing-first test</text>
  <line x1="114" y1="213" x2="180" y2="213" stroke="var(--fg-muted)"/><polygon points="178,209 186,213 178,217" fill="var(--fg-muted)"/>
  <path d="M 240 199 L 296 213 L 240 227 L 184 213 Z" fill="none" stroke="var(--accent)"/>
  <text x="240" y="216" text-anchor="middle" fill="var(--accent)" font-size="8">reporter confirms?</text>
  <line x1="296" y1="213" x2="350" y2="213" stroke="var(--fg-muted)"/><polygon points="348,209 356,213 348,217" fill="var(--fg-muted)"/>
  <rect x="356" y="202" width="70" height="22" rx="3" fill="none" stroke="var(--accent)" stroke-width="2"/>
  <text x="391" y="217" text-anchor="middle" fill="var(--accent)" font-size="8">CLOSED</text>
  <text x="520" y="217" text-anchor="middle" fill="var(--fg-muted)" font-size="8">until then: Refs #N, issue open</text>
</svg>
<figcaption>Three pasted blocks, one capture aligned to the bracket, a harness, a failing-first test — and a close that waits for the reporter.</figcaption>
</figure>

## Which capture to make

The expensive mistakes happen before recording starts
([From Spec to Shipping 11]({{ '/blog/deep-dives/from-spec-to-shipping-11-capture-driven-development/' | relative_url }})):
wrong tap, wrong content, wrong physics.

| Question | Capture | Why this one |
|---|---|---|
| "It never locks / decodes garbage" | `gophertrunk capture -freq … -sample-rate … -seconds 60` | IQ + sidecar, two verdicts printed |
| "The lock drops every few minutes" | `baseband.auto_record.on_cc_sync_loss: true` (+ `tap: ddc`) | keeps the re-acquisition IQ ([TETRA Part 10]({{ '/blog/deep-dives/tetra-end-to-end-10-control-channel-sync-loss/' | relative_url }})) |
| "MRC hurts / never calibrates" | `sdr.soapy_remote[].diversity_capture: <prefix>` (`_seconds` 1..1200) | the **only pre-combine** tap — every other tap sits downstream of the combiner ([Cookbook 12]({{ '/blog/tutorials/operator-cookbook-12-diversity-mrc/' | relative_url }})) |
| "This call sounds wrong" | `baseband.voice_iq_debug` | the exact channelised IQ the voice chain consumed, per call |
| "P25 voice is weak / garbled" | a **voice-channel** capture of a *marginal* call | `samples/p25/README.md`'s priority ask ([P25 12]({{ '/blog/deep-dives/p25-end-to-end-12-weak-signal-gap/' | relative_url }})) |

`gophertrunk capture` heads the table because it *grades* its recording.
Samples pinned at the ADC rail print `capture: WARNING — N% of the recorded
samples are pinned at the ADC rail: the front end was overloaded … NO
receiver … can decode this file` plus the remedy. A carrier off centre prints `capture: WARNING — the recorded
carrier sits N kHz above/below centre (≈N ppm at this frequency)`; the probe
reads only the first ~11 ms, so a momentarily-strong neighbour can fool it
([#1143](https://github.com/MattCheramie/GopherTrunk/issues/1143)).

**Physics, then rate, then metadata.** `docs/decoder-capture-needs.md` puts
physics first: for anything carrying information in phase or 4-level
amplitude — TETRA, NXDN, YSF, DMR, M17, P25 — demodulated audio is useless.
IQ or nothing, [`.cfile`]({{ '/reference/cfile-format/' | relative_url }})
or [cs16]({{ '/reference/cs16-format/' | relative_url }}) — the
[#1187](https://github.com/MattCheramie/GopherTrunk/issues/1187) "RC4 audio"
files failed `discriminatorAudioSanity`. Rate next: TETRA wants ≥ 72 kHz; 48 kHz
cannot lock. Then the sidecar: neither format embeds a rate or centre, so
`<stem>.metadata.json` with `protocol`, `sample_rate_hz` and
`center_freq_hz` is mandatory — `gophertrunk test -capture` grades against
it, and `-sigmf` emits a [SigMF]({{ '/reference/sigmf/' | relative_url }}) twin
([Analog Edge 10]({{ '/blog/tutorials/analog-edge-10-capture-discipline/' | relative_url }})
is the full treatment). Content matters too: a 25 s *silent* DMO PTT was a
poor vector; the ask that worked was *known colour, actually talking*.

## The words that close an issue

The maintainer writes the last part, under the repo's strictest rule —
`CLAUDE.md`'s issue-closing policy:

- **Never close an issue as completed until the fix is verified**: a
  failing-first regression test passes **and** the reporter has confirmed
  it, or the original symptom was reproduced and shown resolved.
- **When you can't verify, leave it open** — post a concise status comment
  saying what was found and what is blocking.
- **Address the latest follow-up, not the original report.**
- **In PRs, prefer `Refs #N` over `Closes #N`** until verified, so a merge
  cannot auto-close. `not_planned` and `duplicate` are ungated.

It exists because #764 was closed twice on an unverified fix while the
symptom stayed live (#771) — the two-pipelines trap of
[Issue Tracker Part 22]({{ '/blog/solution-postmortem/from-the-issue-tracker-22-two-pipelines/' | relative_url }}) —
and a PreToolUse hook gives it teeth:

```python
# .claude/hooks/guard-issue-close.py (shape)
is_close_completed = (
    ti.get("method") == "update"
    and ti.get("state") == "closed"
    and ti.get("state_reason") in (None, "", "completed")
)
if not is_close_completed:
    return 0  # comments, labels, reopens, not_planned, duplicate pass
# else: emit an "ask" decision restating the bar
```

The PR template's last line is literally `Closes #NNN / Refs #NNN`; `Refs`
is the default. For a reporter this means **your confirmation is the last
gate**: when a maintainer asks you to run the fix, or to replay your capture
through a harness such as `TestTETRADMOReplay` or
`TestDiversityCombinerReplay`, that is the step nobody may skip on your
behalf. "Capture-verified, live run pending" is the same discipline, spoken
carefully.

| Symptom | Likely cause | Fix / read |
|---|---|---|
| Thread stalls after one WARN | no status-line family, no startup block | paste blocks 1–3 |
| Capture "looks strong", never locks | ADC clipping, wrong rate, audio not IQ | the clip verdict; [Analog Edge 4]({{ '/blog/tutorials/analog-edge-04-clipping-overload-intermod/' | relative_url }}) |
| MRC question, ordinary capture | every other tap is post-combine | `diversity_capture`; [Weak-Signal 12]({{ '/blog/deep-dives/weak-signal-engineering-12-proving-signal/' | relative_url }}) |
| Fix merged, issue still open | waiting for *your* confirmation | run it, or replay your capture |

### How this line shapes operator practice

- **Run at `debug` when something is wrong.** The status-line families are
  DEBUG; an INFO log has no counters to trend.
- **Bracket every capture.** Start it from the daemon so the bracket lands
  in the symptom's log.
- **Never reason from a lone WARN.** Find its family first.
- **Say what you verified, exactly.** Capture-verified and live-verified
  are different claims.

## Where this goes next

Thirteen parts have read the log one family at a time; the finale folds them
into one page. [Part 14]({{ '/blog/tutorials/field-notebook-14-field-card/' | relative_url }})
is the field card — one table per protocol family, the noise-meter list, and
the five lines to read first on any report.

## FAQ

**What should I include in a GopherTrunk bug report?**
The version, the config with secrets stripped, and three log blocks from a
`log.level: debug` run: the startup block through the first lock, the
protocol's periodic status line for minutes either side of the event, and
thirty seconds around the event with any capture bracket. Attach or link
IQ, not audio.

**How do I line up an IQ capture with the daemon log?**
The `siglab: capture started` timestamp is the file's sample zero, and
`capture ended` carries `samples`, `elapsed` and `recorded_seconds`; a log
event at time T sits `(T − started) × sample_rate_hz` samples in. The log is
local time; sidecars are UTC.

**Which GopherTrunk capture do I use for a diversity or MRC problem?**
Only `sdr.soapy_remote[].diversity_capture`. It taps both branches before
the combiner and writes `<prefix>.br0.cs16`, `.br1.cs16` and a
`.diversity.json` sidecar; every other recording path is downstream of the
combine and has one combiner baked in.

**Why won't GopherTrunk close my issue when the PR is merged?**
Because the policy separates fixed from verified: a close claims the symptom
is gone, earned by a failing-first regression passing *and* the reporter
confirming. PRs say `Refs #N` so a merge cannot auto-close, and a hook asks
a human before any close-as-completed.

**Can I send a WAV of the audio instead of IQ?**
For FM-audio formats — MPT 1327, and as a fallback for POCSAG/FLEX/DSC —
yes. For TETRA, DMR, P25, NXDN, YSF and M17, no: the information is in phase
or 4-level amplitude and demodulated audio has destroyed it. Record IQ with
a `.metadata.json` sidecar.

## Series navigation

**Part 13 of 14** · ←
[Part 12: Web Symptoms That Are Really Log Stories]({{ '/blog/tutorials/field-notebook-12-web-symptoms-log-stories/' | relative_url }})
· Next →
[Part 14: The One-Page Field Card]({{ '/blog/tutorials/field-notebook-14-field-card/' | relative_url }})
