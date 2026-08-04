---
slug: identifying-unknown-signals
title: Identifying an unknown signal
description: You found a signal — now what is it? — a practical method for naming a mystery transmission from its frequency, bandwidth, and modulation, using the waterfall's visual shape and the way it behaves on the air.
keywords: identify unknown signal, signal identification, waterfall shape, bandwidth, modulation, AM vs FM, digital signal, control channel signature, signal fingerprint, what is this frequency
level: intermediate
status: full
prereq:
  - searching-and-discovery
faq:
  - q: How do I tell what an unknown signal is?
    a: Gather clues rather than guessing. The frequency tells you the likely service via the band plan; the bandwidth tells you whether it's narrow voice or something wider; the modulation and the waterfall shape tell you AM, FM, or digital; and the behaviour — constant versus bursty — tells you data from voice. Together these usually narrow a mystery to one or two candidates.
  - q: Can the waterfall alone identify a signal?
    a: Not by itself, but it's the single most useful view. The width and shape of a trace on the waterfall reveal bandwidth and often the modulation at a glance, and the pattern over time — solid, bursty, or stepping — separates a control channel from a voice channel from noise. It narrows the field fast, then you confirm with the audio and the band plan.
  - q: What does a digital signal look like?
    a: Digital signals tend to show as a flat-topped block of fairly uniform width rather than the peaked shape of analog voice, and they sound like a harsh buzz or hiss instead of speech. A trunked control channel is the clearest case — a constant, unbroken block that never stops transmitting, unlike voice channels that come and go.
---

# Identifying an unknown signal

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
Finding a signal and *naming* it are two different skills. Identification is **detective work**:
combine the **frequency** (which service, via the [band plan](/learn/scanning/band-plans/)), the
**bandwidth** and **modulation** (read off the [waterfall](/learn/rf-sdr/fft-and-waterfall/)),
and the **behaviour** — constant data versus bursty voice — into a short list of candidates.
No single clue is proof, but together they usually settle it. The waterfall's **visual shape** is
your fastest tool; the audio and band plan confirm it.
</div>

You've caught a signal you can't name. Resist the urge to guess — identification is a process of
stacking clues until only one answer fits. Each property of a transmission narrows the field:
where it sits, how wide it is, how it's modulated, and how it behaves over time. This lesson is
that method, and it leans heavily on the [waterfall](/learn/rf-sdr/fft-and-waterfall/), because a
signal's picture reveals most of what you need before you've listened to a word.

## Clue 1 — frequency and the band plan

Start with where it lives. The [band plan](/learn/scanning/band-plans/) turns a frequency into a
short list of likely services: a signal at 156.8 MHz is almost certainly marine, one at 121 MHz
in AM is aviation, one at 855 MHz is a strong trunking candidate. This is the cheapest clue and
often the most powerful, because services are legally confined to their bands. Write the exact
frequency down first — everything else builds on it.

## Clue 2 — bandwidth and shape

How **wide** is the signal? On the waterfall, a trace's width is its bandwidth, and bandwidth is
one of the best discriminators there is. Narrowband voice channels are slim; a wideband data or
video signal sprawls. The **shape** helps too: analog FM voice tends to show a peaked, wandering
trace that moves with the speaker's voice, while many digital signals show a **flat-topped block**
of uniform width. The [signal anatomy](/learn/rf-sdr/signal-anatomy/) lesson breaks down what
each part of that picture means.

A rough field guide to widths:

| Rough bandwidth | Typical signal |
|-----------------|----------------|
| ~3 kHz | SSB / narrow utility voice |
| ~5–15 kHz | Analog FM / AM voice channel |
| ~6–12 kHz | Digital voice (P25, DMR, and kin) |
| Tens of kHz+ | Wideband data, paging, some trunking |

## Clue 3 — modulation and sound

Next, how is it **modulated**? AM and FM voice each have a characteristic sound — AM thin and
prone to fading, FM full and quieting to silence between words — while digital sounds like a harsh
buzz or hiss with no intelligible speech. On the waterfall, AM often shows a strong centre carrier
with sidebands, FM a fuller block that breathes with the audio, and digital a steady, textured
block. Switching your receiver's mode and hearing which one makes the signal intelligible is a
quick, decisive test for analog.

## Clue 4 — behaviour over time

Finally, watch what it **does**. A signal that keys up, carries a short exchange, and falls silent
behaves like a **voice channel**. A signal that transmits **continuously and never stops** behaves
like **data** — and a constant, unbroken block on the waterfall is the classic signature of a
[trunked control channel](/learn/digital-trunking/the-control-channel/). Something that bursts in
a regular rhythm might be telemetry or paging. Time-behaviour is exactly the clue that separates a
control channel from the voice channels around it, and it's only visible if you watch for a while.

## Putting the clues together

No single clue names a signal — the method is to **stack** them. Suppose you catch a carrier at
851.0125 MHz, a flat-topped block about 12 kHz wide, digital-sounding, transmitting without
pause. Frequency says 800 MHz public-safety band; width and sound say digital voice family;
constant behaviour says data — and the combination points hard at a **P25 control channel**. Any
one clue is weak; four pointing the same way is a confident identification.

When the clues *disagree* — an unexpected width in a band, an odd mode — that's a signal worth
extra attention, and often the start of the most interesting catches. If you're still stuck, note
everything and come back; identification gets faster with every mystery you solve.

## From identification to a decode

Once you believe you know what a signal is, the next step depends on the answer. Analog voice you
simply listen to. A trunked control channel you hand to a decoder — which is where the
[digital-trunking](/learn/digital-trunking/identifying-the-system/) side of the site picks up,
turning a named control channel into a followed system. Either way, an identified signal belongs
in your [frequency records](/learn/scanning/frequency-records/) so you never have to solve the
same mystery twice.

<div class="knowledge-check" data-quiz data-correct-msg="Right — a constant, unbroken block that never stops transmitting is the classic control-channel signature." markdown="0">
  <p class="knowledge-check__q">Quick check: on the waterfall you see a digital-sounding block in the 800 MHz band that transmits without ever pausing. Most likely?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">An analog FM voice channel</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">A trunked control channel</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">A marine voice channel</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- Identification is **detective work** — stack clues until only one candidate fits.
- **Frequency** plus the [band plan](/learn/scanning/band-plans/) gives the likely service; it's
  the cheapest and often strongest clue.
- **Bandwidth and shape** on the [waterfall](/learn/rf-sdr/fft-and-waterfall/) separate narrow
  voice from wide data and hint at the modulation.
- **Modulation and sound** distinguish AM, FM, and digital; switching mode is a fast test for
  analog.
- **Behaviour over time** separates bursty voice from constant data — and a never-stopping block
  is a **control channel**.
- No one clue is proof; a confident ID is several clues agreeing, then logged so you needn't solve
  it twice.

Next up: [building your frequency records](/learn/scanning/frequency-records/).
