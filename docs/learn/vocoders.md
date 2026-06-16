---
slug: vocoders
title: Vocoders — IMBE & AMBE+2
description: How radio vocoders work — IMBE (P25) and AMBE+2 (DMR/NXDN) speech codecs explained, why modelling speech lets digital voice fit in a few kbps, and how GopherTrunk turns vocoder frames back into audio.
keywords: vocoder, IMBE, AMBE+2, AMBE, speech codec, P25 vocoder, DMR vocoder, half-rate full-rate, digital voice codec, MBE
level: advanced
status: full
prereq:
  - digital-voice
faq:
  - q: What is a vocoder in digital radio?
    a: A vocoder (voice coder) is a speech codec that compresses spoken audio into a very low bitrate by modelling how speech is produced rather than recording the waveform. It extracts parameters like pitch and the spectral shape of each short slice of speech, sends just those, and the receiver synthesises audio from them. This lets a voice fit in a few kilobits per second.
  - q: What vocoder does P25 use?
    a: P25 Phase 1 uses IMBE (Improved Multi-Band Excitation) at about 7.2 kbps including error correction (4.4 kbps of voice). P25 Phase 2 and many DMR systems use AMBE+2, a more efficient successor. Both come from the MBE family of codecs. GopherTrunk includes vocoder implementations to turn these frames back into audio.
  - q: What is the difference between IMBE and AMBE+2?
    a: Both are multi-band excitation vocoders from the same lineage. IMBE is the older codec used by P25 Phase 1. AMBE+2 is newer and more efficient, used by P25 Phase 2, DMR, and NXDN, and supports lower bitrates (half-rate) so systems can fit two voice streams where one used to go. AMBE+2 generally sounds better at a given bitrate.
  - q: Why does digital voice sometimes sound robotic?
    a: Because the vocoder reconstructs speech from a compact model of pitch and spectral shape rather than reproducing the original sound. When the model can't perfectly capture a voice — or when bit errors corrupt the parameters on a weak signal — the synthesised audio takes on a robotic, watery, or warbling quality.
gophertrunk_links:
  - title: Vocoders (reference)
    url: /vocoders.html
    note: GopherTrunk's vocoder implementations and details.
  - title: Voice calibration
    url: /voice-calibration.html
    note: tune decoded audio levels and clarity.
---

# Vocoders — IMBE & AMBE+2

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
A **vocoder** (voice coder) is the speech codec that makes [digital
voice](/learn/digital-voice/) possible. Instead of sending the speech *waveform*, it
**models how speech is produced** — pitch, energy, spectral shape — and sends just those
parameters, squeezing a voice into a few kbps. **IMBE** powers P25 Phase 1; **AMBE+2**
(its more efficient successor) powers P25 Phase 2, **DMR**, and **NXDN**. The receiver
**re-synthesises** audio from the parameters, which is why digital voice can sound
slightly robotic. GopherTrunk ships vocoder implementations to turn these frames back
into sound.
</div>

The [last lesson](/learn/digital-voice/) said a vocoder compresses speech into a few
kbps. This lesson explains *how* — and names the specific codecs you'll meet on the air.
It's the stage-7 detail of [antenna-to-audio](/learn/antenna-to-audio/).

## What a vocoder is

A **vocoder** is a codec specialised for *speech*. General audio compression (like MP3)
tries to reproduce any sound faithfully. A vocoder cheats brilliantly: it assumes the
sound is a human voice, so it only needs to capture the things that make speech
*intelligible* — and can throw away everything else. That assumption is what gets the
bitrate down to a few kbps, low enough to fit a narrow digital channel.

## Modelling speech instead of recording it

Human speech is produced by the vocal cords (a pitched buzz, or noisy hiss for sounds
like "s") shaped by the throat and mouth (a filter). A vocoder mirrors this **source +
filter** model. Many times a second, for each short slice of speech, it measures:

- **Pitch** — how fast the vocal cords vibrate (or whether the sound is unvoiced).
- **Voicing/energy** — how loud, and voiced vs. unvoiced across frequency bands.
- **Spectral shape** — the resonances the mouth and throat impose.

It transmits only these compact parameters as a **vocoder frame**. The receiver feeds
them into a matching synthesiser — *recreating* a voice that sounds like the talker
rather than replaying a recording. The codecs in P25 and DMR belong to the **MBE
(Multi-Band Excitation)** family, which is a refined version of this idea.

## IMBE and the P25 connection

**IMBE — Improved Multi-Band Excitation** — is the vocoder of **P25 Phase 1**. It runs
at about **7.2 kbps** over the air, of which roughly **4.4 kbps** is voice and the rest
is [forward error correction](/learn/demodulation-pipeline/) to protect the parameters
against bit errors (important, because a corrupted parameter sounds much worse than a
corrupted audio sample). IMBE is the classic public-safety digital-voice codec.

## AMBE+2 and DMR / NXDN

**AMBE+2** is IMBE's more efficient successor from the same family. It's used by **P25
Phase 2**, **DMR**, and **NXDN**, and it supports **lower (half-rate) bitrates**, which
is part of how DMR fits *two* voice timeslots in a channel and how P25 Phase 2 doubles
capacity. At a given bitrate AMBE+2 generally sounds cleaner than IMBE.

| Codec | Used by | Notes |
|-------|---------|-------|
| IMBE | P25 Phase 1 | ~7.2 kbps (incl. FEC); the original |
| AMBE+2 | P25 Phase 2, DMR, NXDN | More efficient; supports half-rate |

## Why it can sound robotic

Because the audio is **synthesised from a model**, not reproduced, it carries the
model's limitations: voices can sound slightly **robotic, watery, or warbling**,
especially when the speaker's voice doesn't fit the model well. And on a weak signal,
**bit errors corrupt the parameters** — a wrong pitch or spectral value makes a
distinctly digital "burble," the audible face of the [cliff
effect](/learn/digital-voice/).

## How GopherTrunk decodes vocoder frames to audio

In the [pipeline](/learn/demodulation-pipeline/), once a voice channel is demodulated and
decoded, the result is a stream of **vocoder frames**. GopherTrunk feeds them into its
built-in vocoder — the **matching** decoder for the system (IMBE for P25 Phase 1, AMBE+2
elsewhere) — which **synthesises the audio waveform**. That audio is then played live and
written to a [WAV file](/learn/antenna-to-audio/). The [Vocoders reference](/vocoders.html)
covers GopherTrunk's implementations, and [Voice calibration](/voice-calibration.html)
helps you dial in clean output.

<div class="knowledge-check" data-quiz data-correct-msg="Right — a vocoder sends a model of the speech, not the waveform." markdown="0">
  <p class="knowledge-check__q">Quick check: how does a vocoder fit a voice into a few kbps?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">By recording the waveform at very low quality</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">By sending parameters of a speech model the receiver re-synthesises</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">By speeding up the audio</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- A **vocoder** compresses speech by **modelling how it's produced**, not recording it.
- It sends **pitch, voicing, and spectral shape** as compact frames; the receiver
  re-synthesises audio.
- **IMBE** → P25 Phase 1; **AMBE+2** → P25 Phase 2, DMR, NXDN (more efficient, half-rate).
- Synthesis from a model is why digital voice can sound **robotic**, worse on weak signals.
- GopherTrunk runs the matching vocoder to turn frames into playable, recordable audio.

Next: the wider family of digital protocols and how to tell them apart.
