---
slug: following-a-system-end-to-end
title: Following a system end to end
description: Trace one trunked call through GopherTrunk — control channel decoded, talkgroup grant seen, voice channel tuned, vocoder run, audio out and recorded — the full pipeline that turns a lock into labelled audio.
keywords: trunked call pipeline, control channel grant, voice channel follow, vocoder, GopherTrunk pipeline, talkgroup grant, multiple simultaneous calls, radio IDs, web console, architecture
level: intermediate
status: full
prereq:
  - finding-the-control-channel
  - anatomy-of-a-call
faq:
  - q: What happens after GopherTrunk locks a control channel?
    a: It reads the control channel's grants in real time. When it sees a talkgroup granted a voice channel, it tunes a receiver to that channel, demodulates and decodes it, runs the vocoder frames through a speech decoder to produce audio, and plays and records the result tagged with the talkgroup and radio ID. Then it's immediately ready for the next grant.
  - q: Can GopherTrunk follow more than one call at once?
    a: Yes. Because it decodes the control channel continuously, it knows every voice-channel grant as it happens and can task receivers to capture multiple simultaneous calls across the system. It mixes or queues them for playback and recording according to the priorities you set, so a busy system doesn't drop calls.
  - q: How does GopherTrunk know which voice channel to tune?
    a: The control channel announces it. Each grant names the talkgroup and the voice channel it's been assigned, so GopherTrunk reads the grant, looks up the frequency, and tunes a receiver there. Periodic grant updates let it catch calls already in progress, so late entry is handled too.
  - q: Where can I watch this pipeline happen?
    a: The web console shows each stage live — the CC Activity panel for control-channel grants, the Radio IDs panel for who's transmitting, and the active-call view for audio. The architecture page is the engineering view of the same flow. Watching these together is the fastest way to confirm the whole chain is healthy.
gophertrunk_links:
  - title: CC Activity
    url: /cc-activity.html
    note: watch grants arrive and calls start in real time.
  - title: Radio IDs
    url: /radio-ids.html
    note: see which radio ID is transmitting on each call.
  - title: Web console
    url: /web.html
    note: where every stage of the pipeline is visible live.
---

# Following a system end to end

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
With the [control channel locked](/learn/digital-trunking/finding-the-control-channel/), GopherTrunk runs a
continuous loop. It **decodes the control channel**, sees a **talkgroup grant**, **tunes
a receiver** to the assigned **voice channel**, **demodulates and decodes** it, feeds the
**[vocoder](/learn/rf-sdr/vocoders/)** frames to a speech decoder, and **plays and
records** the audio tagged with the talkgroup and **[radio ID](/radio-ids.html)**.
Because it keeps reading the control channel in parallel, it can **follow several calls
at once**. Every stage is visible in the **[web console](/web.html)** — which makes
watching the pipeline the fastest way to confirm it's healthy.
</div>

This is the trunking equivalent of [From antenna to
audio](/learn/rf-sdr/antenna-to-audio/): the whole chain assembled, but with the
trunking logic in the middle. We'll follow a single dispatch call from a grant on the
control channel to a labelled recording on disk.

## Stage 1 — the control channel is decoded continuously

GopherTrunk is locked to the [control channel](/learn/digital-trunking/the-control-channel/) and reading its
[signalling](/learn/digital-trunking/control-channel-signaling/) stream every moment. Affiliations,
registrations, and — the part we care about — **channel grants** flow past. This
decoding never pauses; it's the running map of the system, and you can watch it scroll
in the **[CC Activity](/cc-activity.html)** panel.

## Stage 2 — a grant is seen

A user keys up. Their radio requests a channel, the system finds a free one, and the
control channel broadcasts a **grant**: *"talkgroup 101 → voice channel 3."* GopherTrunk
reads it, notes the talkgroup, the voice-channel frequency, and the transmitting
**[radio ID](/radio-ids.html)**. This is the moment a call is born — the
[anatomy-of-a-call](/learn/digital-trunking/anatomy-of-a-call/) request-and-grant arc, seen from the decoder's
side.

## Stage 3 — the voice channel is tuned and decoded

GopherTrunk tasks a receiver to tune to **voice channel 3** and runs the full receive
chain on it — [filter, demodulate, recover symbols](/learn/rf-sdr/demodulation-pipeline/),
decode. Because it's still reading the control channel in parallel, it can do this
*and* catch the next grant elsewhere, capturing **multiple simultaneous calls** across
the system. Periodic **grant updates** let it join calls already in progress (late
entry), so nothing slips by.

## Stage 4 — the vocoder reconstructs the voice

The voice channel doesn't carry audio — it carries **[vocoder](/learn/rf-sdr/vocoders/)**
frames, speech compressed to a few kilobits per second by a codec like **IMBE** (P25
Phase 1) or **AMBE+2** (DMR, P25 Phase 2). GopherTrunk feeds those frames into a matching
speech decoder, which **reconstructs an audible waveform**. This is the step that turns
abstract bits back into a human voice.

## Stage 5 — audio out and recording

The reconstructed audio is played live in the console and written to a file **tagged
with the talkgroup, radio ID, timestamp, and system**. From here GopherTrunk can stream
it onward to services like Broadcastify or RdioScanner. The call that began as a grant
on the control channel is now a labelled recording — and the engine is already following
the next one.

<figure class="figure" markdown="0">
<svg viewBox="0 0 540 120" role="img" aria-label="The trunking pipeline as a row of boxes: control channel, grant seen, tune voice channel, demodulate and decode, vocoder, audio out, with a loop arrow returning from audio back to control channel." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9.5" fill="currentColor" text-anchor="middle">
    <rect x="8"   y="44" width="74" height="34" rx="5" fill="currentColor" fill-opacity="0.15" stroke="currentColor" stroke-width="1.3"/><text x="45" y="58">control</text><text x="45" y="70">channel</text>
    <rect x="98"  y="44" width="64" height="34" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="130" y="58">grant</text><text x="130" y="70">seen</text>
    <rect x="178" y="44" width="64" height="34" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="210" y="58">tune</text><text x="210" y="70">voice ch</text>
    <rect x="258" y="44" width="64" height="34" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="290" y="58">demod/</text><text x="290" y="70">decode</text>
    <rect x="338" y="44" width="64" height="34" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="370" y="64">vocoder</text>
    <rect x="418" y="44" width="74" height="34" rx="5" fill="currentColor" fill-opacity="0.15" stroke="currentColor" stroke-width="1.3"/><text x="455" y="58">audio out</text><text x="455" y="70">+ record</text>
    <g stroke="currentColor" stroke-width="1.2">
      <line x1="82" y1="61" x2="97" y2="61"/><line x1="162" y1="61" x2="177" y2="61"/><line x1="242" y1="61" x2="257" y2="61"/>
      <line x1="322" y1="61" x2="337" y2="61"/><line x1="402" y1="61" x2="417" y2="61"/>
    </g>
    <path d="M455 44 C455 18, 45 18, 45 42" fill="none" stroke="currentColor" stroke-width="1.1" stroke-dasharray="4 3" marker-end="url(#lp)"/>
    <text x="250" y="14" font-size="9">control channel keeps reading — ready for the next grant ↺</text>
  </g>
  <defs><marker id="lp" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>The trunking loop: the control channel is decoded continuously, each grant spins up a voice-channel decode and vocoder, and the engine returns to wait for the next grant — following many calls at once.</figcaption>
</figure>

## Watching the whole thing live

The **[web console](/web.html)** lets you watch every stage at once, which is the
fastest way to confirm the pipeline is healthy:

| Stage | Where to watch | Healthy sign |
|-------|----------------|--------------|
| Control channel decoded | [CC Activity](/cc-activity.html) | Affiliations and grants scrolling steadily |
| Grant seen | [CC Activity](/cc-activity.html) | A `grant` row: TG ← radio @ frequency |
| Voice channel followed | Active-call view | A call appears with a live audio meter |
| Who's transmitting | [Radio IDs](/radio-ids.html) | The source radio ID labelled on the call |
| Audio out | Active-call view | Audible playback, a written recording |

If you want the engineering view of this same flow — the components and data paths
behind the panels — the **[architecture](/architecture.html)** page lays it out. When a
stage *doesn't* light up, this table doubles as a troubleshooting starting point.

<div class="knowledge-check" data-quiz data-correct-msg="Right — it reads the grant on the control channel, then tunes the named voice channel." markdown="0">
  <p class="knowledge-check__q">Quick check: how does GopherTrunk know which voice channel to tune for a new call?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">It scans every voice channel until it hears audio</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">The control-channel grant names the assigned voice channel</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">It uses the same frequency as the last call</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- The **control channel is decoded continuously** — the running map of every call.
- Each **grant** names a talkgroup and voice channel; GopherTrunk **tunes and decodes**
  that channel.
- The **[vocoder](/learn/rf-sdr/vocoders/)** reconstructs audio from the decoded frames.
- Audio is **played and recorded**, tagged with talkgroup, radio ID, and timestamp.
- Reading the control channel in parallel lets GopherTrunk **follow many calls at once**.
- The **[web console](/web.html)** shows every stage live, which doubles as a health check.

Next, we tackle the hardest real-world case: a system spread across multiple sites and
simulcast transmitters, in [multi-site & simulcast in
practice](/learn/digital-trunking/multisite-and-simulcast-in-practice/).
