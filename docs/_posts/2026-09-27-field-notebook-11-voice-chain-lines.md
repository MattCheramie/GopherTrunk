---
title: "The Field Notebook, Part 11: Voice-Chain Lines — Frames, Colours & Teardown Reasons"
description: How to read GopherTrunk's composer voice-chain log — the voice-follow start and end lines, speech_frames and tch_frames, the bfi_count bad-frame counter, the TETRA DMO seed lines, and the undecoded_drops and concurrency_suppressed counters that look alarming but work exactly as designed.
category: tutorials
keywords: composer voice chain log, tetra voice follow, speech_frames tch_frames, bfi_count bad frame, undecoded_drops concurrency_suppressed, tetra dmo seed recovered, voice teardown reason timeout, gophertrunk field notebook, decode quality log
tags: [field-notebook, logs, composer, voice, tetra, tutorial]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "The Field Notebook"
series_part: 11
---

*Part 11 of **The Field Notebook**, a 14-part operator's tutorial that reads
GopherTrunk's `debug.log` one line family at a time — what each field measures,
what a healthy rig prints, and which deep dive to open when a line goes wrong.
[Part 10]({{ '/blog/tutorials/field-notebook-10-recorder-lines/' | relative_url }})
read the recorder's lines and, when a recording came out short, pointed
upstream. Upstream is the composer's per-call voice chain, and this part reads
its log: `composer: … voice follow started`, the `speech_frames` / `tch_frames`
that count real speech, the `bfi_count` bad frames, the DMO seed lines, and the
demux counters that look alarming but are working exactly as designed.*

> **TL;DR:** The composer spins one demod goroutine per call and logs
> `INF composer: … voice follow started` / `… ended`. On TETRA the ended line
> carries `bursts`, `speech_frames`/`tch_frames` (real speech), `bfi_count`
> (Bad Frame Indications — noise/encrypted/corrupt), `stch_bursts`, and
> `other_call_bursts`. The **shared same-carrier demux** adds `undecoded_drops`,
> `concurrency_suppressed`, `ownerless_drops` and `marker_collisions` — on a
> busy multi-slot carrier these run to tens of thousands (`undecoded_drops=24784`,
> `concurrency_suppressed=33125`) and are **by-design cross-slot-leak
> protection**, not defects. DMO logs a `seed` line: `seed_known`,
> `seed_verified`, `bursts_solved`, `solve_rejects`. A silent call ends
> `reason=timeout` at ~2× the hangtime (the 20 Aug DMO run: `speech_frames=0`,
> then a timeout teardown). P25 Phase 1 needs `minAutotuneLDUs`=5 to trust an
> autotune measurement.

**Key takeaways**

- **`speech_frames`/`tch_frames` mean traffic; `bfi_count` is bad frames.** A
  healthy TETRA follow ends with hundreds of speech frames and a bounded BFI
  count; all BFI and zero speech is a wrong seed, encryption, or a dead carrier.
- **`undecoded_drops` and `concurrency_suppressed` are noise meters, not
  faults.** On a busy multi-slot carrier they run to tens of thousands by design
  — cross-slot-leak protection dropping bursts that don't belong to this owner.
- **A DMO seed line is the descramble state.** `seed_verified=true` means the
  scramble seed was proven by a CRC-decoded solve; `seed_known` without verified
  is a fallback guess, and `solve_rejects` counts solves that decoded nothing.
- **`reason=timeout` on a voice chain is a silent call.** The boundary tracker
  ends a follow that never decoded a matching frame after ~2× the hangtime — the
  20 Aug DMO signature of a chain that fell back to the wrong colour.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| TETRA voice-follow lines | `speech_frames`, `tch_frames`, `bfi_count` | `internal/voice/composer/tetra_voice.go` |
| Shared-demux counters | `undecoded_drops`, `concurrency_suppressed` | `tetra_voice.go` (shared demux ended line) |
| DMO seed line | `seed`, `seed_known`, `seed_verified`, `solve_rejects` | `internal/voice/composer/tetra_dmo_voice.go` |
| P25 decode quality | `ldus`, `uncorrectable_ldus`, `gated_ldus` | `internal/voice/composer/p25p1_voice.go` |
| Autotune trust gate | `minAutotuneLDUs` = 5 before an offset is trusted | `p25p1_voice.go` |
| Boundary teardown | hangtime end; `reason=timeout` when nothing decodes | `internal/voice/composer/boundary.go` |
| The chain wiring | which chain runs for which protocol | [Voice Coding Part 9]({{ '/blog/deep-dives/voice-coding-09-the-composer/' | relative_url }}) |

## In this post

- **What the voice-chain lines are telling you** — start, end, the counters.
- **What healthy looks like** — a TETRA follow that decoded.
- **When it doesn't** — all BFI, wrong seed, and the timeout teardown.
- **The demux counters that look alarming** — why they are not faults.
- **Seeds and colours** — reading the DMO descramble state.

## What the voice-chain lines are telling you

When the engine grants a call and retunes a voice tap, the composer subscribes,
spins one demod goroutine, and logs the chain it chose. On TETRA:

```
INF composer: tetra voice follow started — TCH/S decode + ACELP vocoder serial=cc:same-carrier:2 group=61432 timeslot=2 usage_marker=3 colour_code=27 rate_hz=18000
```

`group` and `timeslot` identify the granted call; `usage_marker` and
`colour_code` are the TETRA descramble/routing keys; `rate_hz` is the symbol
rate. When the call ends, the ended line is the yield report:

```
INF composer: tetra voice follow ended serial=cc:same-carrier:2 bursts=214 speech_frames=196 tch_frames=196 stch_bursts=4 bfi_count=18 other_call_bursts=2
```

Read it as a funnel. `bursts` is every traffic burst the chain saw;
`speech_frames`/`tch_frames` are the ones that CRC-decoded to speech (the two
fields track the same count); `bfi_count` is Bad Frame Indications — bursts that
yielded no CRC-valid speech, whether from noise, corruption, or encryption;
`stch_bursts` counts stolen (STCH signalling) bursts; `other_call_bursts` are
bursts belonging to a different call on the same slot grid. A healthy follow has
`speech_frames` in the hundreds and a small `bfi_count`. When `speech_frames`
is near zero and `bfi_count` is nearly all of `bursts`, the chain saw traffic but
could not turn it into speech — a wrong descramble key, encryption, or a
signal too weak for the class-2 CRC.

<figure class="lab-figure">
<svg viewBox="0 0 680 250" width="680" height="250" role="img" aria-label="A funnel from traffic bursts to speech frames in a TETRA voice follow. Bursts enter at the top; most CRC-decode into speech frames, a small fraction become bad frame indications, a few are stolen signalling bursts, and a couple belong to another call. Below, a separate diagram shows the shared demux counters undecoded drops and concurrency suppressed as a by-design side channel that discards bursts not owned by this call, running to tens of thousands on a busy carrier without being a fault.">
  <rect x="20" y="30" width="150" height="34" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="95" y="51" text-anchor="middle" fill="var(--accent)" font-size="10">bursts=214</text>
  <line x1="95" y1="64" x2="95" y2="86" stroke="currentColor"/><polygon points="91,86 95,96 99,86" fill="currentColor"/>
  <rect x="20" y="96" width="150" height="34" rx="6" fill="none" stroke="currentColor"/>
  <text x="95" y="113" text-anchor="middle" fill="currentColor" font-size="10">speech_frames=196</text>
  <text x="95" y="125" text-anchor="middle" fill="var(--fg-muted)" font-size="8">CRC-valid → ACELP</text>
  <line x1="170" y1="47" x2="250" y2="47" stroke="var(--fg-muted)"/><polygon points="250,43 260,47 250,51" fill="var(--fg-muted)"/>
  <rect x="260" y="30" width="130" height="26" rx="5" fill="none" stroke="var(--fg-muted)"/>
  <text x="325" y="47" text-anchor="middle" fill="var(--fg-muted)" font-size="9">bfi_count=18</text>
  <line x1="170" y1="56" x2="250" y2="72" stroke="var(--fg-muted)"/><polygon points="248,68 258,74 246,76" fill="var(--fg-muted)"/>
  <rect x="260" y="62" width="130" height="24" rx="5" fill="none" stroke="var(--fg-muted)"/>
  <text x="325" y="78" text-anchor="middle" fill="var(--fg-muted)" font-size="9">stch_bursts=4</text>
  <line x1="170" y1="60" x2="250" y2="100" stroke="var(--fg-muted)"/><polygon points="247,96 257,102 245,105" fill="var(--fg-muted)"/>
  <rect x="260" y="92" width="130" height="24" rx="5" fill="none" stroke="var(--fg-muted)"/>
  <text x="325" y="108" text-anchor="middle" fill="var(--fg-muted)" font-size="9">other_call=2</text>
  <rect x="20" y="164" width="620" height="66" rx="6" fill="none" stroke="var(--fg-muted)" stroke-dasharray="4 3"/>
  <text x="330" y="182" text-anchor="middle" fill="var(--fg-muted)" font-size="10">shared same-carrier demux (busy multi-slot carrier)</text>
  <rect x="40" y="192" width="180" height="30" rx="5" fill="none" stroke="var(--accent)"/>
  <text x="130" y="211" text-anchor="middle" fill="var(--accent)" font-size="9">undecoded_drops=24784</text>
  <rect x="240" y="192" width="220" height="30" rx="5" fill="none" stroke="var(--accent)"/>
  <text x="350" y="211" text-anchor="middle" fill="var(--accent)" font-size="9">concurrency_suppressed=33125</text>
  <rect x="480" y="192" width="140" height="30" rx="5" fill="none" stroke="var(--fg-muted)"/>
  <text x="550" y="207" text-anchor="middle" fill="var(--fg-muted)" font-size="9">by design —</text>
  <text x="550" y="217" text-anchor="middle" fill="var(--fg-muted)" font-size="9">not a fault</text>
</svg>
<figcaption>The TETRA follow is a funnel: most bursts become speech frames, a few become BFIs or belong to another call. The shared demux's undecoded_drops and concurrency_suppressed are cross-slot-leak protection, not defects.</figcaption>
</figure>

## What healthy looks like

A TETRA voice follow that decoded looks like the funnel above: hundreds of
`speech_frames`, a small `bfi_count`, a handful of `other_call_bursts`. On a
shared same-carrier carrier the demux ended line is the aggregate, and the
29 Aug field session is the reference for what "busy and healthy" reads like:

```
INF composer: tetra shared voice demux ended key=… tch_frames=7052 stch_bursts=… bfi_count=… undecoded_drops=24784 ownerless_drops=… crc_fallbacks=… marker_collisions=… concurrency_suppressed=33125
```

`tch_frames=7052` is the traffic — thousands of CRC-valid speech frames across
the session — and `vocoder_drops=0` on the per-owner ended lines says nothing was
dropped for want of vocoder capacity. The large `undecoded_drops` and
`concurrency_suppressed` beside that healthy `tch_frames` are the point of the
next section: on a busy multi-slot carrier they are *supposed* to be large. This
is a fully-loaded TETRA site decoding several talkgroups at once.

### How this line shapes operator practice

- **Compare `speech_frames` to `bfi_count`, not to zero.** Some BFI on any real
  call is normal; the alarm is `bfi_count` swamping `speech_frames`, which means
  the chain saw traffic it could not decode.
- **Read the demux counters as ratios, not absolutes.** `undecoded_drops` and
  `concurrency_suppressed` in the tens of thousands next to a healthy
  `tch_frames` is a busy carrier working, not a fault.
- **`vocoder_drops` must be zero.** A non-zero count means the vocoder worker
  could not keep up — a CPU problem, distinct from the RF-driven `bfi_count`.

## When it doesn't look like that

**All BFI, no speech.** When a follow ends with `speech_frames=0` and
`bfi_count` accounting for nearly every burst, the chain locked onto traffic but
decoded none of it. On TETRA TMO the usual cause is a wrong `colour_code` or a
signal below the class-2 CRC threshold; on DMO it is a wrong scramble seed
(below). Either way the recorder then logs the short-recording or dead-key line
from [Part 10]({{ '/blog/tutorials/field-notebook-10-recorder-lines/' | relative_url }}),
and the call ends silent.

**The timeout teardown.** A voice chain that never decodes a matching frame is
torn down by the boundary tracker after its no-voice window — twice the hangtime
(`noVoiceStartupFactor` = 2) — with `reason=timeout`, so a phantom or mis-tuned
grant frees its tap instead of holding it for the engine's much longer watchdog.
This is exactly the 20 Aug DMO signature: the voice chain granted before the
control pipeline had recovered the colour, buffered its bursts, hit the buffer
cap, fell back to the wrong seed, and decoded every burst as BFI —
`speech_frames=0`, then a `reason=timeout` end at ~7 s because
`boundaryTracker.onVoice` was never called to refresh liveness. A DMO call that
*does* decode refreshes liveness on real speech and runs to its real tail; a
BFI-only one dies at the first hangtime. So a `reason=timeout` on a DMO follow
with `speech_frames=0` is the "bogus recording that ends at a timeout, not PTT
release" symptom — a decode failure, chased on the descramble side.

| Field | Measures | Healthy | Worry when |
|---|---|---|---|
| `speech_frames` / `tch_frames` | CRC-valid speech decoded | hundreds per over / thousands per session | near zero while `bursts` is high |
| `bfi_count` | bursts that yielded no speech | a small fraction of `bursts` | nearly all of `bursts` (wrong key / weak / encrypted) |
| `vocoder_drops` | frames the vocoder worker dropped | 0 | non-zero — CPU starvation |
| `undecoded_drops` | bursts not owned by this call, dropped | large on a busy carrier (by design) | — (a noise meter, not a fault) |
| `concurrency_suppressed` | cross-slot-leak bursts suppressed | large on a busy carrier (by design) | — (a noise meter, not a fault) |
| `reason` (teardown) | how the follow ended | hangtime `normal`/`released` | `timeout` with `speech_frames=0` |

## The demux counters that look alarming

The shared same-carrier demux routes each traffic burst to the call that owns
its slot, and counts everything it *doesn't* route. On a carrier running several
concurrent talkgroups, most bursts a given owner sees belong to *other* calls,
so the demux drops them — and counts the drops.
`undecoded_drops` is bursts that decoded to no owner; `concurrency_suppressed`
is bursts suppressed to stop one slot's audio leaking into another's recording;
`ownerless_drops`, `crc_fallbacks` and `marker_collisions` are the finer-grained
tallies of the same protection. On the 29 Aug session these read
`undecoded_drops=24784` and `concurrency_suppressed=33125` — tens of thousands —
next to a perfectly healthy `tch_frames=7052` and `vocoder_drops=0`. That is not
a fault; it is the cross-slot-leak protection doing its job on a busy multi-slot
carrier. The number to watch is `vocoder_drops`: zero means the decode kept up.
The mechanism — usage-marker routing and the ownership fences — is the subject of
[TETRA End to End Part 9]({{ '/blog/deep-dives/tetra-end-to-end-09-equalizer-voice-path/' | relative_url }})
and the composer design in
[Voice Coding Part 9]({{ '/blog/deep-dives/voice-coding-09-the-composer/' | relative_url }}).

## Seeds and colours

DMO adds one more line family, because its traffic descramble seed is recovered
off the air. The follow starts with a `seed_hint`, and the ended line reports the
descramble state:

```
INF composer: tetra DMO voice follow ended serial=… dnb_bursts=242 speech_frames=70 bfi_count=… seed=0x0001733855 seed_known=true seed_verified=true seed_from_pipeline=false bursts_solved=3 solve_rejects=1 exact_adopts=2 hint_adopts=1
```

`seed_known` says the chain has a working seed; `seed_verified` says that seed
was *proven* by a CRC-decoded solve rather than adopted as a fallback guess;
`bursts_solved` counts exact GF(2) seed solves; `solve_rejects` counts solves
that were rejected because they decoded nothing (a false solve from the
soft-assisted path); `seed_from_pipeline` says the chain adopted the control
pipeline's answer rather than solving its own. A healthy DMO follow ends
`seed_verified=true` with `speech_frames` well above zero. The honesty this line
enforces is the same lesson [instruments, not logs]({{ '/blog/deep-dives/from-spec-to-shipping-13-instruments-not-logs/' | relative_url }})
teaches: a `colour_recovered` flag was once set true even when the confidence
gate had not cleared, so a call that decoded nothing logged as "recovered" and
sent the investigation after the wrong thing — the flag now says only what the
code verified. When a DMO follow ends `seed_known=true` but `speech_frames=0`,
read `seed_verified`: if it is false, the seed was a fallback guess that decoded
nothing, and the recording is silent for the reason [Part 10]({{ '/blog/tutorials/field-notebook-10-recorder-lines/' | relative_url }})
described. The full DMO seed story is
[TETRA End to End Part 13]({{ '/blog/deep-dives/tetra-end-to-end-13-dmo-pipeline-grants/' | relative_url }}).

One P25 field to know: the Phase 1 chain logs a `decode quality` line with
`ldus`, `uncorrectable_ldus` and `gated_ldus`, and only trusts an autotune
carrier measurement once a call has decoded `minAutotuneLDUs` (5) LDUs at a low
uncorrectable rate. A `composer: p25p1 autotune measurement skipped` DEBUG line
with a low `ldus` is that gate firing, not a fault.

| Symptom | Likely cause | Fix / read |
|---|---|---|
| Follow ends `speech_frames=0`, `bfi_count` ≈ `bursts` | wrong descramble key, weak signal, or encryption | Check `colour_code`/seed; improve signal — [TETRA Part 9]({{ '/blog/deep-dives/tetra-end-to-end-09-equalizer-voice-path/' | relative_url }}) |
| DMO follow `seed_known=true` but `speech_frames=0` | seed was a fallback guess (`seed_verified=false`) | Let the pipeline recover the seed; [TETRA Part 13]({{ '/blog/deep-dives/tetra-end-to-end-13-dmo-pipeline-grants/' | relative_url }}) |
| `reason=timeout` with `speech_frames=0` | chain never decoded a matching frame (silent/phantom) | Fix the descramble/tuning; the timeout is a decode failure, not a short call |
| Huge `undecoded_drops` / `concurrency_suppressed` | busy multi-slot carrier (by design) | Nothing — read `vocoder_drops`/`tch_frames` instead |
| `vocoder_drops` non-zero | vocoder worker can't keep up (CPU) | Lower `sdr.sample_rate` or CPU load |
| `autotune measurement skipped` with low `ldus` | call too short to trust the offset (`minAutotuneLDUs`=5) | Expected on brief calls — not a fault ([Voice Coding Part 9]({{ '/blog/deep-dives/voice-coding-09-the-composer/' | relative_url }})) |

## Where this goes next

The recorder and the composer both point at symptoms that surface first in the
*web console* — a History table hours behind, an empty panel, a "symbol: poor"
badge — and each of those is really a log story once you know where to look.
[Part 12]({{ '/blog/tutorials/field-notebook-12-web-symptoms-log-stories/' | relative_url }})
reads the web symptoms that are really log stories, tracing each one back to the
line in `debug.log` that names it.

## FAQ

**What is bfi_count in a GopherTrunk TETRA voice log?**
It counts Bad Frame Indications — traffic bursts that yielded no CRC-valid speech,
whether from noise, corruption, or encryption. A small `bfi_count` next to
hundreds of `speech_frames` is a healthy call; `bfi_count` accounting for nearly
every burst means the chain saw traffic it could not decode.

**Are undecoded_drops and concurrency_suppressed errors?**
No. They are the shared same-carrier demux's cross-slot-leak protection, counting
bursts that belong to other calls on a busy multi-slot carrier. Tens of thousands
is normal next to a healthy `tch_frames`. Read `vocoder_drops` (should be zero)
and `tch_frames` to judge health, not these counters.

**Why did my DMO recording end at a timeout with no speech?**
The voice chain never decoded a matching frame, so the boundary tracker tore it
down after ~2× the hangtime with `reason=timeout`. On DMO that usually means a
wrong scramble seed (`seed_verified=false`) — the chain fell back to a guess that
decoded nothing. Let the control pipeline recover the seed.

**What does seed_verified mean on a DMO follow?**
It means the scramble seed was proven by a CRC-decoded solve, not merely adopted
as a fallback guess. `seed_known=true` with `seed_verified=false` is an unverified
fallback; if `speech_frames` is also zero, the recording is silent because the
descramble seed was wrong.

**Why does a short P25 call skip the autotune measurement?**
The Phase 1 chain only trusts a carrier-offset measurement after `minAutotuneLDUs`
(5) LDUs at a low uncorrectable rate — roughly a second of voice. A shorter call
logs `autotune measurement skipped` with a low `ldus`, which is the trust gate
working, not a decode fault.

## Series navigation

**Part 11 of 14** · ←
[Part 10: Recorder Lines — Spans, Segments & Dead Keys]({{ '/blog/tutorials/field-notebook-10-recorder-lines/' | relative_url }})
· Next →
[Part 12: Web Symptoms That Are Really Log Stories]({{ '/blog/tutorials/field-notebook-12-web-symptoms-log-stories/' | relative_url }})
