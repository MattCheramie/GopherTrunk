---
slug: digital-voice
title: Analog vs. digital voice
description: Analog versus digital voice radio explained — how analog FM voice works, why public safety moved to digital, the role of the vocoder, the digital "cliff effect" on range, and what it all means for monitoring with an SDR.
keywords: digital voice radio, analog vs digital, P25 vs analog, vocoder, cliff effect, digital radio quality, why public safety went digital, narrowband
level: intermediate
status: full
prereq:
  - digital-modulation
faq:
  - q: Why did public safety move from analog to digital voice?
    a: Digital voice lets more channels fit in the same spectrum, supports trunking and talkgroups, carries data like unit IDs alongside the audio, allows encryption, and keeps voice clear right out to the edge of coverage rather than fading into static. Regulators also pushed narrower channels, which digital handles well. The trade-offs are codec artefacts and an abrupt loss of audio when the signal gets too weak.
  - q: What is the cliff effect in digital radio?
    a: The cliff effect is digital radio's all-or-nothing behaviour at the edge of coverage. As long as the signal is decodable, audio is clear; once it drops below the threshold the error correction can handle, the audio doesn't gradually fade like analog — it breaks up and then cuts out abruptly, as if falling off a cliff.
  - q: Does digital voice sound better than analog?
    a: It sounds clearer in good-to-marginal conditions because there's no background hiss, but it can sound robotic or watery because the vocoder reconstructs speech from a compressed model rather than reproducing the original waveform. Analog degrades gracefully into static; digital stays crisp until it suddenly fails.
  - q: What does digital voice mean for monitoring?
    a: You need software that can demodulate the digital signal and run the matching vocoder to produce audio — which is exactly what GopherTrunk does. You also get metadata like talkgroups and radio IDs for free, but you'll hit the cliff effect on weak signals and you can't hear encrypted talkgroups at all.
gophertrunk_links:
  - title: Vocoders
    url: /vocoders.html
    note: the codecs that make digital voice possible.
  - title: Voice calibration
    url: /voice-calibration.html
    note: tune GopherTrunk for the cleanest decoded audio.
---

# Analog vs. digital voice

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Analog voice** (FM) sends the speech waveform directly and **fades gracefully** into
static as it weakens. **Digital voice** compresses speech with a **[vocoder](/learn/vocoders/)**
into a few kilobits per second, then sends it as [digital
modulation](/learn/digital-modulation/) — gaining spectrum efficiency, trunking,
talkgroups, embedded IDs, and optional encryption, at the cost of a robotic timbre and
the **cliff effect** (clear audio that *abruptly* cuts out at the coverage edge instead
of fading). For monitoring, you need software that demodulates *and* vocodes — which is
what GopherTrunk does.
</div>

Module 5 reaches the systems GopherTrunk was built for. Before the trunking protocols
themselves, it's worth understanding what changed when voice went from analog to digital
— because it shapes everything about how these systems sound and behave.

## How analog voice radio works

Traditional two-way radio uses **[FM](/learn/analog-modulation/)**: the audio directly
varies the carrier's frequency, and the receiver turns that back into sound. It's
simple, robust, and **degrades gracefully** — as the signal weakens you get more hiss,
then more, until the voice is buried, but you can often still make out words deep into
the noise. One conversation occupies one channel.

## Why systems moved to digital

Agencies and regulators pushed to digital for several concrete reasons:

- **Spectrum efficiency** — narrower channels and, with [trunking](/learn/what-is-trunking/),
  many groups sharing few frequencies.
- **Talkgroups and features** — virtual channels, priority, and selective calling.
- **Embedded data** — radio IDs, talkgroup numbers, and status ride alongside the voice.
- **Clear audio across coverage** — no hiss; voice stays crisp until the signal fails.
- **Encryption** — optional [privacy](/learn/encryption/) on sensitive talkgroups.

The catch is that digital introduces a codec (the vocoder) and a sharp failure edge,
both covered below.

## The role of the vocoder

You can't send full-quality audio in the tiny [bitrate](/learn/symbols-and-baud/) a
narrow digital channel provides. The solution is a **[vocoder](/learn/vocoders/)** (voice
coder): instead of transmitting the speech *waveform*, it transmits a compact
*description* of the speech — enough parameters for the receiver to **reconstruct**
something that sounds like the talker. This squeezes a voice into a few kbps, which is
the whole reason digital voice fits. The next lesson is devoted to how vocoders do this.

The scale of the squeeze is dramatic. Uncompressed phone-quality audio is about
**64 kbps**; an MP3 music stream, 128+ kbps. A P25 voice channel carries the actual
speech in roughly **4.4 kbps** — and that has to share the channel with error
correction and signalling. That's better than a **10:1** reduction versus a plain phone
call, achievable *only* because the vocoder models speech rather than storing sound.
Squeeze that hard and some fidelity is unavoidably lost — the source of digital voice's
characteristic timbre.

## Quality, range, and the cliff effect

Two perceptual differences fall out of going digital:

- **Timbre.** Because the vocoder *models* speech rather than reproducing it, digital
  voice can sound slightly robotic or "watery," especially on a marginal signal. In
  good conditions it's clean and hiss-free.
- **The cliff effect.** This is the big one. Analog fades smoothly; digital is
  **all-or-nothing**. As long as [error correction](/learn/demodulation-pipeline/) can
  fix the bit errors, audio is perfect. Once the signal drops below that threshold,
  audio doesn't gently fade — it **breaks into burbles and then cuts out abruptly**, as
  if walking off a cliff.

<figure class="figure" markdown="0">
<svg viewBox="0 0 440 150" role="img" aria-label="Two curves of audio quality versus signal strength. Analog declines as a smooth downward slope. Digital stays flat and high, then drops off a vertical cliff at the threshold." xmlns="http://www.w3.org/2000/svg">
  <line x1="40" y1="120" x2="420" y2="120" stroke="currentColor" stroke-opacity="0.4"/>
  <line x1="40" y1="20" x2="40" y2="120" stroke="currentColor" stroke-opacity="0.4"/>
  <text x="230" y="140" text-anchor="middle" font-size="10" fill="currentColor">signal strength →</text>
  <text x="20" y="70" font-size="10" fill="currentColor" transform="rotate(-90 20 70)">quality</text>
  <path d="M60 118 C160 100 300 60 410 35" fill="none" stroke="currentColor" stroke-width="1.5" stroke-dasharray="5 3"/>
  <text x="360" y="30" font-size="10" fill="currentColor">analog</text>
  <path d="M150 118 L150 45 L410 45" fill="none" stroke="currentColor" stroke-width="2"/>
  <text x="360" y="40" font-size="10" fill="currentColor">digital</text>
  <text x="150" y="100" font-size="9" fill="currentColor" text-anchor="middle">cliff</text>
</svg>
<figcaption>Analog quality slides down gradually; digital stays clear, then falls off a cliff once the signal can no longer be decoded.</figcaption>
</figure>

## What this means for monitoring

For SDR monitoring, the consequences are practical:

- You need software that can both **demodulate** the digital signal and run the matching
  **vocoder** to make sound — GopherTrunk does both (see
  [antenna-to-audio](/learn/antenna-to-audio/)).
- You get **metadata for free** — talkgroups, radio IDs — that analog never carried.
- The **cliff effect** means a marginal system is either decoding well or not at all;
  improving [SNR](/learn/decibels/) (antenna, placement, [gain](/learn/gain-and-agc/)) is
  what moves you back from the cliff edge.
- **Encrypted** talkgroups are silent no matter how strong the signal — that's the
  [next-but-one lesson](/learn/encryption/).

<div class="knowledge-check" data-quiz data-correct-msg="Right — digital is clear until it abruptly fails at the cliff." markdown="0">
  <p class="knowledge-check__q">Quick check: how does digital voice behave as the signal weakens toward the edge of coverage?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">It gradually fades into hiss like analog</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">It stays clear, then breaks up and cuts out abruptly</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">It gets louder</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Analog FM** sends the waveform and **fades gracefully**; **digital** sends vocoded
  speech and is **all-or-nothing**.
- Digital won for **spectrum efficiency, trunking, embedded data, and encryption**.
- The **[vocoder](/learn/vocoders/)** compresses speech into a few kbps.
- The **cliff effect** means marginal = clean or nothing; raise SNR to back off the edge.
- Monitoring needs demod **and** vocoder; you gain metadata but lose encrypted traffic.

Next: a closer look at the vocoders themselves — IMBE and AMBE+2.
