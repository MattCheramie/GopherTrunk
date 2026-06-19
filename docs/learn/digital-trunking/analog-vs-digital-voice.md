---
slug: analog-vs-digital-voice
title: Analog vs. digital voice
description: Analog FM voice fades gracefully into noise; digital voice stays clean then falls off the digital cliff at the BER threshold. The trade-offs that drove radio to digital.
keywords: analog vs digital voice, digital cliff, cliff effect, BER threshold, vocoder artifacts, graceful degradation, FM voice, digital trunking voice, robotic audio, underwater audio
level: intermediate
status: full
prereq:
  - the-digital-leap
  - digital-modulation
faq:
  - q: What is the digital cliff?
    a: The digital cliff is the abrupt way digital voice fails at the edge of coverage. As long as the signal stays above the bit-error-rate threshold the error correction can handle, audio is perfect. Drop below it and the audio doesn't fade gracefully like analog — it garbles and cuts out, as if walking off a cliff.
  - q: Why does analog FM degrade gracefully but digital does not?
    a: In analog FM the voice waveform rides the carrier directly, so as the signal weakens you simply hear more hiss but can still make out words. In digital, voice is a stream of bits protected by error correction. The decoder either recovers the bits or it doesn't, so quality stays flat then collapses once errors overwhelm the correction.
  - q: Why does digital voice sometimes sound robotic or underwater?
    a: Digital voice is reconstructed by a vocoder that models speech from a handful of parameters rather than reproducing the original sound. On clean signals it sounds crisp; on marginal signals or with unusual voices it can sound robotic, watery, or underwater because the model is guessing at detail it cannot fully capture.
  - q: If digital fails so abruptly, why did everyone switch to it?
    a: Digital buys capacity through trunking and narrow channels, privacy through encryption, and features like talkgroups and embedded radio IDs. It also keeps voice clean across most of the coverage area instead of fading into static. The hard failure edge is the price paid for those gains.
gophertrunk_links:
  - title: Voice calibration
    url: /voice-calibration.html
    note: tune GopherTrunk for the cleanest decoded audio.
  - title: Vocoders
    url: /vocoders.html
    note: the codecs that turn the recovered bits back into speech.
---

# Analog vs. digital voice

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
In **analog FM**, the voice waveform rides the carrier directly, so the signal
**degrades gracefully** — as it weakens you hear more hiss but can still copy words.
In **digital**, voice becomes **bits plus error correction**, so audio stays clean
and hiss-free right up to the **BER threshold**, then falls off the **digital cliff**
and cuts out abruptly. Digital also adds **vocoder artifacts** — a robotic or
"underwater" timbre — because speech is reconstructed from a model. Agencies accepted
that hard edge in exchange for **capacity, privacy, and features**, which is exactly
why the trunked systems GopherTrunk decodes are digital.
</div>

The [RF & SDR path covers this from the signal side](/learn/rf-sdr/digital-voice/);
here we frame it for trunking. Before you can follow a P25 or DMR call, it helps to
know *why* its audio behaves so differently from the analog scanner traffic you may
already know — and why that behaviour shapes everything about monitoring a digital
system.

## How analog voice carries speech

In an analog FM channel, the speech waveform **modulates the carrier directly**: as
your voice rises and falls, the carrier's frequency shifts in step, and the receiver
turns those shifts back into sound. There is no model in the middle — what comes out
is a direct (if imperfect) copy of what went in.

That directness is the source of analog's famous **graceful degradation**. As the
signal weakens, noise simply adds on top of the recovered audio. First a little hiss,
then more, then a lot — but a trained ear can pull words out of surprisingly deep
static. The signal *fades*; it doesn't suddenly vanish. One conversation occupies one
channel, and that channel is usable across a wide, soft-edged coverage area.

## How digital voice carries speech

Digital throws out the direct waveform. Speech is first squeezed by a
**[vocoder](/learn/rf-sdr/vocoders/)** into a few kilobits per second of parameters,
then wrapped in **[forward error correction](/learn/rf-sdr/demodulation-pipeline/)** and
sent as [digital modulation](/learn/rf-sdr/digital-modulation/). The receiver
demodulates the symbols back to bits, lets the error correction repair what it can,
and feeds the recovered parameters to the vocoder to **reconstruct** audio.

This indirection is what unlocks everything digital trunking is built on. Because the
voice is now just data, the same channel can also carry talkgroup numbers, radio IDs,
and signalling alongside it; the bits can be encrypted; and several calls can share a
channel through time-slotting. But it also means the audio is only as good as the
decoder's ability to recover bits — which leads straight to the cliff.

## The digital cliff

Error correction can fix a *limited* number of bit errors. As long as the channel's
**bit error rate (BER)** stays under the threshold the FEC can repair, every bit is
recovered and the audio is flawless — no hiss at all. Push the BER past that
threshold and the FEC is overwhelmed: the recovered bits are wrong, the vocoder is
fed garbage, and the audio breaks into burbles and then **cuts out**. There is almost
no middle ground. This abrupt, all-or-nothing failure is the **digital cliff**.

<figure class="figure" markdown="0">
<svg viewBox="0 0 440 160" role="img" aria-label="A graph of audio quality versus signal strength. The analog curve declines as a smooth downward slope. The digital curve stays flat and high, then drops off a vertical cliff at the bit-error-rate threshold." xmlns="http://www.w3.org/2000/svg">
  <line x1="40" y1="125" x2="420" y2="125" stroke="currentColor" stroke-opacity="0.4"/>
  <line x1="40" y1="20" x2="40" y2="125" stroke="currentColor" stroke-opacity="0.4"/>
  <text x="230" y="148" text-anchor="middle" font-size="10" fill="currentColor">weaker  ←  signal strength  →  stronger</text>
  <text x="20" y="75" font-size="10" fill="currentColor" transform="rotate(-90 20 75)">quality</text>
  <path d="M60 123 C170 108 300 64 410 38" fill="none" stroke="currentColor" stroke-width="1.5" stroke-dasharray="5 3"/>
  <text x="350" y="34" font-size="10" fill="currentColor">analog</text>
  <path d="M160 123 L160 48 L410 48" fill="none" stroke="currentColor" stroke-width="2"/>
  <text x="350" y="44" font-size="10" fill="currentColor">digital</text>
  <line x1="160" y1="48" x2="160" y2="123" stroke="currentColor" stroke-width="2"/>
  <text x="160" y="115" font-size="9" fill="currentColor" text-anchor="middle">BER cliff</text>
</svg>
<figcaption>Analog quality slides down a gradual slope; digital stays clear and hiss-free, then drops off a cliff once the bit error rate passes the threshold the error correction can repair.</figcaption>
</figure>

The practical upshot: a digital system is usually either **decoding well or not at
all**. There is little of the "weak but workable" zone analog gives you. Improving
[SNR](/learn/rf-sdr/decibels/) — a better antenna, placement, or
[gain](/learn/rf-sdr/gain-and-agc/) — doesn't make a marginal decode *prettier*; it
moves you back from the cliff edge so the decode succeeds at all.

## Vocoder artifacts

Even well above the cliff, digital voice has a distinctive sound. Because the
vocoder **models** speech rather than reproducing the waveform, it can introduce a
slightly **robotic** or **"underwater"** timbre, most noticeable on background noise,
music, sirens, or voices the model handles poorly. In good conditions it is crisp and
hiss-free — often *clearer* than analog — but it never sounds quite like an open
microphone. Recognising this timbre is useful: it tells you you're listening to a
reconstructed signal, and the next lesson explains exactly how that reconstruction
works.

## The trade-offs

| Aspect | Analog FM | Digital |
|--------|-----------|---------|
| Failure at the edge | Gradual fade into hiss | Abrupt cliff at BER threshold |
| Audio in good conditions | Slight hiss always present | Clean, hiss-free, slightly robotic |
| Capacity | One call per channel | Trunking + time-slots pack many more |
| Privacy | Anyone can listen | Optional strong encryption |
| Extra data | None | Talkgroup, radio ID, status embedded |
| Weak-signal copy | Often possible by ear | Usually all-or-nothing |

Neither is strictly "better" — they fail differently. Agencies chose digital because
the capacity, privacy, and feature gains outweighed the loss of graceful degradation,
and because regulators were pushing toward narrower channels that digital handles
well. For a monitor, the lesson is to stop expecting analog behaviour: with digital
you chase a *clean lock*, not a *readable signal*.

<div class="knowledge-check" data-quiz data-correct-msg="Correct — digital stays clean until the BER threshold, then falls off the cliff." markdown="0">
  <p class="knowledge-check__q">Quick check: how does digital voice behave as the signal weakens toward the edge of coverage?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">It gradually fades into hiss like analog</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">It stays clear, then garbles and cuts out abruptly</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">It slowly gets quieter but stays intelligible</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Analog FM** carries the voice waveform directly and **degrades gracefully** into
  hiss as the signal weakens.
- **Digital voice** is bits plus error correction, so it stays clean then falls off
  the **digital cliff** at the BER threshold.
- The vocoder's modelling introduces a **robotic or "underwater"** timbre even on
  strong signals.
- Digital won on **capacity, privacy, and features**; the hard failure edge is the
  price.
- For monitoring, raise [SNR](/learn/rf-sdr/decibels/) to back off the cliff — a
  marginal digital signal decodes well or not at all.

Next, we open up that vocoder: [from voice to bits](voice-to-bits-vocoders/), where
we meet IMBE, AMBE+2, and how a few kilobits become speech again.
