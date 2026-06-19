---
slug: voice-to-bits-vocoders
title: "From voice to bits: vocoders"
description: How a vocoder models speech to squeeze it into a few kbps — IMBE for P25 Phase 1, AMBE+2 for P25 Phase 2 and DMR, ACELP for TETRA, and how a frame becomes audio again.
keywords: vocoder, voice coder, IMBE, AMBE+2, ACELP, DVSI, P25 vocoder, DMR vocoder, TETRA codec, speech model, low bitrate codec, digital voice
level: advanced
status: full
prereq:
  - analog-vs-digital-voice
faq:
  - q: What is a vocoder?
    a: A vocoder, or voice coder, is a codec that compresses speech by modelling how the vocal tract produces sound rather than sampling the audio waveform. It extracts a small set of parameters per short frame — pitch, voicing, and spectral shape — and sends only those. The decoder runs the model in reverse to synthesise speech that resembles the talker, fitting voice into a few kilobits per second.
  - q: What vocoders do P25 and DMR use?
    a: P25 Phase 1 uses IMBE at roughly 7200 bps including its error correction, of which about 4400 bps is the actual voice description. P25 Phase 2 and DMR use AMBE+2, which carries useful speech at roughly half that net rate so it fits in a time slot. TETRA uses an ACELP-family codec instead.
  - q: Why are these vocoders proprietary?
    a: IMBE and the AMBE family are developed and licensed by Digital Voice Systems, Inc. (DVSI) and are protected by patents, so manufacturers and software must license them. That is why open decoders historically struggled with digital voice audio and why a built-in vocoder is a notable feature of a scanner.
  - q: Does GopherTrunk include a vocoder?
    a: Yes. GopherTrunk includes a vocoder so it can turn the recovered voice frames back into audio you can hear, rather than only showing metadata. Without a vocoder you would decode the control channel and see talkgroups but hear nothing.
gophertrunk_links:
  - title: Vocoders
    url: /vocoders.html
    note: see and configure the codec GopherTrunk uses to make audio.
  - title: Voice calibration
    url: /voice-calibration.html
    note: dial in the cleanest reconstructed speech.
---

# From voice to bits: vocoders

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
A **vocoder** ("voice coder") compresses speech by **modelling how it's produced**
rather than sampling the waveform, squeezing a voice into a few **kbps**. Different
systems use different vocoders: **IMBE** in P25 Phase 1 (~7200 bps including error
correction, ~4400 bps of net voice), **AMBE+2** in P25 Phase 2 and DMR (roughly half
that net rate, so it fits a time slot), and an **ACELP**-family codec in TETRA. The
IMBE and AMBE codecs are **proprietary (DVSI)**, which is why a decoder needs a
licensed implementation. **GopherTrunk includes a vocoder**, so it can reconstruct
audio from the recovered frames rather than only showing metadata.
</div>

The previous lesson said voice becomes "a few kilobits per second." This lesson is
about the device that performs that miracle — and why digital voice has the timbre it
does. See the [RF & SDR vocoder lesson](/learn/rf-sdr/vocoders/) for the companion
treatment; here we focus on the specific codecs the trunked systems use.

## What a vocoder actually does

An ordinary audio codec like MP3 starts from the *sound* and compresses it. A
**vocoder** starts somewhere stranger: it assumes the sound is **human speech** and
fits a model of the human vocal tract to it. Speech is produced by an excitation —
either a buzzing of the vocal cords (voiced sounds like vowels) or a turbulent hiss
(unvoiced sounds like "s") — passed through the resonant filter of the mouth and
throat. A vocoder measures that process a few dozen times a second.

For each short **frame** of speech (typically around 20 ms), the encoder extracts a
compact description:

- the **pitch** (how fast the vocal cords are vibrating),
- the **voicing** (which parts of the sound are buzz vs. hiss), and
- the **spectral shape** (the resonances of the vocal tract).

It sends only those parameters. There is no waveform on the air at all — just a recipe
for re-creating one. That is why the squeeze is so dramatic: full phone-quality audio
is about **64 kbps**, but the *description* of how to make similar-sounding speech is
a small fraction of that.

## The vocoders you'll meet

Digital trunking standardised on a short list of vocoders, and which one a system uses
is part of that system's identity.

| System | Vocoder | Approx. rate | Notes |
|--------|---------|--------------|-------|
| P25 Phase 1 | IMBE | ~7200 bps gross, ~4400 bps net voice | FDMA, one call per channel |
| P25 Phase 2 | AMBE+2 | roughly half IMBE's net rate | fits a TDMA time slot |
| DMR | AMBE+2 | low-rate variant per slot | two slots per channel |
| TETRA | ACELP family | low bitrate | different codec lineage |

**IMBE** (Improved Multi-Band Excitation) is the P25 Phase 1 codec. Of its roughly
**7200 bps**, a large share is error-correction overhead; the actual voice description
is around **4400 bps**. **AMBE+2** (Advanced Multi-Band Excitation) is the
successor used by P25 Phase 2 and DMR. It carries usable speech at roughly **half**
IMBE's net rate, which is what lets two voice paths share one channel via time-slots —
the subject of a later lesson. **TETRA** took a different path entirely, using an
**ACELP**-family codec (a code-excited linear-prediction design from the same family
as many cellphone codecs).

A consequence worth remembering: because the vocoder *is* part of the system
definition, you cannot decode P25 Phase 2 voice with an IMBE-only decoder, or vice
versa. The bits mean nothing without the matching model.

## Reconstructing a frame into audio

At the receiver, after [demodulation](/learn/rf-sdr/demodulation-pipeline/) and error
correction hand over a clean frame of vocoder parameters, the decoder runs the model
**in reverse**:

<figure class="figure" markdown="0">
<svg viewBox="0 0 520 150" role="img" aria-label="A pipeline showing a vocoder frame of parameters feeding an excitation generator and a vocal-tract filter, which combine to produce a 20-millisecond chunk of synthesised speech." xmlns="http://www.w3.org/2000/svg">
  <rect x="20" y="55" width="110" height="40" rx="6" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.5"/>
  <text x="75" y="72" text-anchor="middle" font-size="11" fill="currentColor" font-weight="600">vocoder frame</text>
  <text x="75" y="86" text-anchor="middle" font-size="9" fill="currentColor">pitch · voicing · shape</text>
  <rect x="180" y="20" width="120" height="38" rx="6" fill="none" stroke="currentColor" stroke-width="1.5"/>
  <text x="240" y="43" text-anchor="middle" font-size="10" fill="currentColor">excitation (buzz/hiss)</text>
  <rect x="180" y="92" width="120" height="38" rx="6" fill="none" stroke="currentColor" stroke-width="1.5"/>
  <text x="240" y="115" text-anchor="middle" font-size="10" fill="currentColor">vocal-tract filter</text>
  <rect x="350" y="55" width="150" height="40" rx="6" fill="currentColor" fill-opacity="0.18" stroke="currentColor" stroke-width="1.5"/>
  <text x="425" y="72" text-anchor="middle" font-size="11" fill="currentColor" font-weight="600">~20 ms of speech</text>
  <text x="425" y="86" text-anchor="middle" font-size="9" fill="currentColor">played back as audio</text>
  <line x1="130" y1="75" x2="178" y2="39" stroke="currentColor" stroke-width="1.5" marker-end="url(#a2)"/>
  <line x1="130" y1="75" x2="178" y2="111" stroke="currentColor" stroke-width="1.5" marker-end="url(#a2)"/>
  <line x1="300" y1="39" x2="348" y2="68" stroke="currentColor" stroke-width="1.5" marker-end="url(#a2)"/>
  <line x1="300" y1="111" x2="348" y2="82" stroke="currentColor" stroke-width="1.5" marker-end="url(#a2)"/>
  <defs><marker id="a2" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>The decoder builds an excitation from the pitch and voicing parameters, shapes it with the vocal-tract filter, and outputs a short chunk of synthesised speech. Stitch the chunks together and you have continuous audio.</figcaption>
</figure>

The decoder generates the **excitation** (a buzz at the coded pitch, or a hiss for
unvoiced sounds), passes it through a filter set to the coded **spectral shape**, and
produces about 20 ms of audio. Repeat ~50 times a second and the chunks blend into
continuous speech. Because every frame is *synthesised* from a model, the output sounds
like the talker but never exactly reproduces them — the source of the robotic or
"underwater" character from the last lesson, and the reason an unusual voice or heavy
background noise can confuse the model.

## Why this matters for a decoder

Two things follow from all this. First, the vocoders that matter here —
**IMBE** and **AMBE+2** — are **proprietary**, developed and licensed by **DVSI**.
Decoding their bits is only half the job; turning those bits into audio needs a
licensed implementation, which is why playable digital voice was scarce in open tools
for years. Second, a scanner that *only* decoded the control channel would show you
talkgroups and IDs but stay silent.

**GopherTrunk includes a vocoder**, so once it follows a call to its voice channel it
can reconstruct and play the audio — see the [Vocoders panel](/vocoders.html) to view
the active codec and [Voice calibration](/voice-calibration.html) to tune for the
cleanest result.

<div class="knowledge-check" data-quiz data-correct-msg="Right — a vocoder models how speech is produced and sends only the parameters." markdown="0">
  <p class="knowledge-check__q">Quick check: how does a vocoder compress speech into a few kbps?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">It samples the waveform at a very low rate</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">It models how speech is produced and sends only the parameters</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">It removes silence and sends the rest unchanged</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- A **vocoder** models how speech is produced and sends only **pitch, voicing, and
  spectral shape** per frame.
- **IMBE** carries P25 Phase 1 (~7200 bps gross, ~4400 bps net voice); **AMBE+2**
  carries P25 Phase 2 and DMR at roughly half the net rate.
- **TETRA** uses an **ACELP**-family codec from a different lineage.
- The IMBE/AMBE codecs are **proprietary (DVSI)**, so a decoder needs a licensed
  implementation.
- The decoder **synthesises** each frame, which is why digital voice sounds modelled —
  and **GopherTrunk includes a vocoder** to produce that audio.

Next, we move from the bits to how they ride the air:
[digital modulation for trunking](digital-modulation-for-trunking/) — C4FM, π/4-DQPSK,
and CQPSK.
