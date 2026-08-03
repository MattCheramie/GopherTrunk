---
slug: p25-phase-1
title: "P25 Phase 1"
description: P25 Phase 1 explained in depth — C4FM modulation, the NAC, WACN and System ID, TSBKs on the control channel, LDU1/LDU2 voice frames, IMBE, and how GopherTrunk follows it.
keywords: P25, Project 25, APCO 25, TIA-102, C4FM, NAC, WACN, System ID, TSBK, LDU1, LDU2, IMBE, P25 Phase 1, trunking control channel
level: intermediate
status: full
prereq:
  - digital-modulation-for-trunking
  - control-channel-signaling
faq:
  - q: What is P25 Phase 1?
    a: P25 Phase 1 is the original digital mode of APCO Project 25, the open standard family (TIA-102) used by most North-American public-safety agencies. It uses C4FM four-level FSK in a 12.5 kHz FDMA channel, running 4800 symbols per second for a 9600 bps stream, and carries voice with the IMBE vocoder. It is the workhorse you meet most often on US police and fire systems.
  - q: What is a NAC in P25?
    a: The NAC, or Network Access Code, is a 12-bit value carried in every P25 frame. It works like a digital CTCSS tone — a radio ignores traffic whose NAC does not match the one it is programmed for, so several systems can share a frequency without hearing each other. GopherTrunk reads the NAC to confirm it is locked to the right system.
  - q: What are TSBKs?
    a: TSBKs (Trunking Signaling Blocks) are the data messages a P25 control channel broadcasts continuously. They announce group and unit call grants, channel assignments, affiliations, registrations and system parameters. By decoding the stream of TSBKs, GopherTrunk learns which talkgroup is starting a call and which voice channel it was assigned.
  - q: What is the difference between LDU1 and LDU2?
    a: LDU1 and LDU2 are the two halves of a P25 voice superframe. Each Logical Data Unit carries nine IMBE voice frames plus embedded signaling — LDU1 carries the Link Control word (talkgroup, source ID, service options) and LDU2 carries the encryption sync. Together they let a receiver decode audio while continuously confirming who is talking.
gophertrunk_links:
  - title: CC Activity
    url: /cc-activity.html
    note: watch the live stream of TSBKs the control channel broadcasts.
  - title: Constellation
    url: /constellation.html
    note: confirm a clean C4FM lock by the four symbol points.
  - title: Vocoders
    url: /vocoders.html
    note: see how GopherTrunk renders IMBE voice frames to audio.
---

# P25 Phase 1

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**P25 Phase 1** is the dominant North-American public-safety digital standard, defined
by **APCO Project 25 / TIA-102**. It is **FDMA** in a **12.5 kHz** channel using
**C4FM** four-level FSK — **4800 symbols/s × 2 bits = 9600 bps** — and carries voice
with the **IMBE** vocoder. Every frame carries a **NAC**, a digital CTCSS that keeps
systems apart, and the system is identified by its **WACN** and **System ID**. The
control channel broadcasts **TSBKs** that grant calls; voice rides in **LDU1/LDU2**
superframes carrying IMBE plus link control. **GopherTrunk decodes the control channel
first**, then follows each grant to the voice channel.
</div>

You met the [modulations of digital trunking](/learn/digital-trunking/digital-modulation-for-trunking/) and how
a [control channel signals](/learn/digital-trunking/control-channel-signaling/) calls. P25 Phase 1 is where all
of it comes together into the single most common digital system you will track in North
America. This lesson opens it up end to end.

## What P25 is, and who runs it

**Project 25** (P25, sometimes APCO-25) is an *open* suite of standards published as the
**TIA-102** series. "Open" matters: unlike a proprietary vendor system, the air
interface is documented, so radios from different manufacturers interoperate and
independent decoders can follow it. It was created for **public safety** — police, fire,
EMS — with a deliberate emphasis on coverage, reliability and graceful behaviour at the
edge of range, where a fire-ground radio still has to be intelligible.

**Phase 1** is the original digital mode. It is a straightforward **FDMA** design: each
call gets its own **12.5 kHz** frequency channel, exactly like the conventional channels
it replaced, but the voice and signaling are digital. That simplicity is part of why it
became the backbone of US public-safety radio.

## The physical layer: C4FM in 12.5 kHz

P25 Phase 1 transmits with **C4FM** — Continuous 4-level FM, a four-level
[FSK](/learn/rf-sdr/digital-modulation/). The carrier sits at one of four small
deviations per symbol, and because four levels carry **two bits** each (a *dibit*), the
**4800 symbols/s** symbol rate yields a **9600 bps** stream.

| Parameter | P25 Phase 1 |
|-----------|-------------|
| Standard | APCO Project 25 / TIA-102 |
| Access method | FDMA (one call per frequency) |
| Channel width | 12.5 kHz |
| Modulation | C4FM (also CQPSK/LSM for simulcast) |
| Symbol rate | 4800 symbols/s |
| Bit rate | 9600 bps |
| Vocoder | IMBE (7200 bps incl. FEC) |
| System access control | NAC (12-bit) |

Because C4FM is **constant-envelope** — only the frequency changes, never the amplitude —
it runs on the cheap, efficient nonlinear amplifiers in handheld radios. Infrastructure
that has to **simulcast** the same channel from many overlapping transmitters instead
uses **CQPSK/LSM**, a linear cousin engineered to land on the *same* four-point
constellation. On GopherTrunk's **[Constellation panel](/constellation.html)** a healthy
Phase 1 signal shows four tight points; smearing toward the centre is your cue that
[SNR](/learn/rf-sdr/decibels/), tuning, or clock recovery needs attention.

## How a P25 frame is addressed: NAC, WACN and System ID

A P25 system needs a way to keep itself separate from neighbours on the same frequency,
and to identify itself uniquely. Three identifiers do this work:

- **NAC (Network Access Code)** — a **12-bit** code in *every* frame, voice or control.
  It behaves like a **digital CTCSS**: a radio (or GopherTrunk) ignores any frame whose
  NAC does not match, so two systems can share a frequency without interfering with each
  other's squelch. It is the first thing a decoder checks to confirm it is on the right
  system.
- **WACN (Wide Area Communications Network)** — a 20-bit identifier for the *network*
  operator, unique enough to distinguish statewide systems from one another.
- **System ID** — a 12-bit number identifying the *system* within that WACN.

Together the **WACN + System ID** name a system globally, while the **NAC** is the quick
per-frame filter. GopherTrunk surfaces these in its activity panels so you can confirm
at a glance that you are locked to the system you intended.

## The control channel: a river of TSBKs

The control channel is a Phase 1 C4FM carrier that never carries voice — only a
continuous stream of data. Its unit of work is the **TSBK**, the **Trunking Signaling
Block**: a fixed-size message that announces something about the system. There are many
TSBK opcodes, but the ones a follower cares about most are:

- **Group Voice Channel Grant** — "talkgroup X is now on voice channel N." This is the
  message that tells GopherTrunk where to point a receiver.
- **Unit-to-Unit grants** — a private call between two individual radios.
- **Registration / affiliation / location** — radios announcing themselves, which keeps
  the [Radio IDs](/radio-ids.html) view populated.
- **System parameters** — the WACN, System ID, the channel-to-frequency mapping, and
  status broadcasts that let a newcomer learn the whole system from the control channel
  alone.

<figure class="figure" markdown="0">
<svg viewBox="0 0 540 220" role="img" aria-label="A P25 Phase 1 flow. A control channel at top emits TSBK messages. A grant message points down to a voice channel where an LDU1 and LDU2 superframe carry IMBE voice frames." xmlns="http://www.w3.org/2000/svg">
  <rect x="30" y="20" width="480" height="44" rx="6" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.5"/>
  <text x="270" y="40" text-anchor="middle" font-size="13" fill="currentColor" font-weight="600">Control channel — stream of TSBKs (C4FM, data only)</text>
  <text x="270" y="56" text-anchor="middle" font-size="10" fill="currentColor">“Group Voice Grant: TG 4521 → channel 9”</text>
  <line x1="270" y1="64" x2="270" y2="104" stroke="currentColor" stroke-width="1.5" stroke-dasharray="4 3" marker-end="url(#p25a)"/>
  <g font-size="10" fill="currentColor">
    <rect x="60" y="108" width="180" height="86" rx="5" fill="none" stroke="currentColor" stroke-width="1.5"/>
    <text x="150" y="126" text-anchor="middle" font-weight="600">LDU1</text>
    <text x="150" y="142" text-anchor="middle" font-size="9">9 × IMBE voice frames</text>
    <text x="150" y="156" text-anchor="middle" font-size="9">+ Link Control</text>
    <text x="150" y="170" text-anchor="middle" font-size="9">(talkgroup, source ID)</text>
    <text x="150" y="184" text-anchor="middle" font-size="9">+ low-speed data</text>
    <rect x="300" y="108" width="180" height="86" rx="5" fill="currentColor" fill-opacity="0.10" stroke="currentColor" stroke-width="1.5"/>
    <text x="390" y="126" text-anchor="middle" font-weight="600">LDU2</text>
    <text x="390" y="142" text-anchor="middle" font-size="9">9 × IMBE voice frames</text>
    <text x="390" y="156" text-anchor="middle" font-size="9">+ encryption sync</text>
    <text x="390" y="170" text-anchor="middle" font-size="9">(algorithm, key, MI)</text>
    <text x="390" y="184" text-anchor="middle" font-size="9">+ low-speed data</text>
  </g>
  <text x="270" y="212" text-anchor="middle" font-size="11" fill="currentColor">LDU1 + LDU2 = one voice superframe, repeating for the call's duration.</text>
  <defs><marker id="p25a" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>A TSBK grant on the control channel hands the call to a voice channel, where repeating LDU1/LDU2 superframes carry IMBE voice plus embedded link control and encryption sync.</figcaption>
</figure>

## Voice frames: LDU1 and LDU2

When a grant fires, the call lands on a voice channel as a repeating **voice superframe**
built from two **Logical Data Units**. Each LDU carries **nine IMBE voice frames** — so a
superframe is eighteen frames of audio — interleaved with embedded signaling:

- **LDU1** carries the **Link Control (LC)** word: the talkgroup, the source unit ID, and
  the service options (including the *encrypted* flag). This repeats so a receiver tuning
  in mid-call quickly learns who is talking.
- **LDU2** carries the **encryption sync** — the algorithm ID, key ID and Message
  Indicator needed to decrypt a protected call.

Both units also carry **low-speed data** and heavy forward error correction, so the call
keeps identifying itself and stays intelligible even as the signal fades. The voice
payload is **IMBE**, the [vocoder](/learn/rf-sdr/vocoders/) whose core patents have
expired — which is why GopherTrunk ships a pure-Go IMBE decoder and renders Phase 1 audio
end to end without external libraries (see [Vocoders](/vocoders.html)).

## Clear vs encrypted

A Phase 1 call is **clear** unless the LDU1 service options flag it encrypted. When it is,
the LDU2 encryption sync names the algorithm (commonly AES-256 or DES on public-safety
systems) and key. GopherTrunk reads the encrypted flag from the link control *before*
voice even starts, logs the call as encrypted, and still captures the protected frames —
but without the (operator-supplied, legally held) key the audio stays unintelligible.
This is the same known-key-only posture covered in [encryption and
authentication](/learn/digital-trunking/encryption-and-authentication/).

## How GopherTrunk follows a Phase 1 system

The procedure is exactly the trunking pattern, made concrete:

1. **Lock the control channel.** Acquire C4FM, recover the clock, and confirm the **NAC**,
   **WACN** and **System ID** match the intended system.
2. **Decode TSBKs.** Read the continuous stream on the
   **[CC Activity](/cc-activity.html)** panel, building a live map of talkgroups, units
   and the channel-to-frequency table.
3. **Follow a grant.** On a Group Voice Channel Grant, tune a receiver to the assigned
   voice channel.
4. **Decode voice.** Lock the LDU1/LDU2 superframes, extract IMBE frames, render them to
   audio, and read the link control to confirm the talkgroup and source ID.
5. **Release and repeat.** When the call ends, return to the control channel and follow
   the next grant.

<div class="knowledge-check" data-quiz data-correct-msg="Correct — the 12-bit NAC works like a digital CTCSS, filtering frames by system." markdown="0">
  <p class="knowledge-check__q">Quick check: what does the P25 NAC do?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Sets the voice channel frequency</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Acts like a digital CTCSS that keeps systems apart on a shared frequency</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Selects the vocoder</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **P25 Phase 1** is the open APCO/TIA-102 standard dominating North-American public
  safety: **FDMA, 12.5 kHz, C4FM, 9600 bps, IMBE**.
- The **NAC** is a per-frame digital CTCSS; **WACN + System ID** name the system globally.
- The control channel broadcasts **TSBKs**, of which **Group Voice Channel Grants** are
  what a follower acts on.
- Voice rides in **LDU1/LDU2** superframes — eighteen IMBE frames plus link control and
  encryption sync.
- GopherTrunk **locks the control channel, decodes TSBKs, follows grants, and renders
  IMBE** to audio.

Next, we see how P25 doubled its capacity by moving the traffic channels to TDMA:
[P25 Phase 2](/learn/digital-trunking/p25-phase-2/).
