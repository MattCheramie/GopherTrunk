---
slug: error-correction-and-framing
title: Error correction & framing
description: How raw symbols become trustworthy data — sync words, framing, interleaving, and forward error correction that fix bit errors before decoding.
keywords: error correction, framing, sync word, interleaving, forward error correction, fec, convolutional code, reed solomon, burst errors
level: advanced
status: full
prereq:
  - snr-evm-and-ber
faq:
  - q: What is a sync word and why is it needed?
    a: "A sync word is a fixed, known bit pattern the transmitter inserts at the start of each frame. The receiver, which has been reading a continuous stream of symbols with no idea where a frame begins, slides along looking for that pattern. When it matches, the receiver knows exactly where the frame — and every field inside it — starts. Without a sync word the bits would be a meaningless stream with no structure."
  - q: How does forward error correction fix errors without asking for a resend?
    a: "Forward error correction adds carefully computed redundant bits before transmission. Those extra bits let the receiver detect and reconstruct a bounded number of flipped bits entirely on its own, with no return channel and no retransmission. A convolutional code plus a Viterbi decoder, or a Reed-Solomon block code, are the classic schemes — essential for one-way broadcast radio where asking for a resend is impossible."
---

# Error correction & framing

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
Recovered bits are noisy and shapeless. **Framing** gives them structure — a **sync word**
marks where each frame begins. **Interleaving** scrambles bit order so a **burst** of
errors is spread thin. **Forward error correction (FEC)** — convolutional codes with
Viterbi decoding, or Reed–Solomon — adds redundancy that **repairs** flipped bits with no
retransmission. Together they turn a link with a nonzero [BER](/learn/dsp/snr-evm-and-ber/)
into trustworthy data.
</div>

Symbol recovery gave us bits; some are wrong. This lesson is the layer that makes those
bits *reliable* — the bridge from DSP into protocol decoding. It follows directly from
[SNR/EVM/BER](/learn/dsp/snr-evm-and-ber/) and hands off to the digital-trunking module's
[framing, FEC & interleaving](/learn/digital-trunking/framing-fec-interleaving/).

## Framing: finding structure in a stream

After symbol recovery you have an unbroken river of bits with no markers. **Framing**
imposes structure. The transmitter groups bits into **frames** of a fixed layout and
prefixes each with a **sync word** — a known pattern the receiver searches for by sliding
a comparison along the stream until it matches.

<figure class="figure" markdown="0">
<svg viewBox="0 0 520 90" role="img" aria-label="A bit stream divided into frames, each beginning with a highlighted sync word followed by header and payload fields." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <rect x="20" y="30" width="70" height="30" fill="currentColor" fill-opacity="0.25" stroke="currentColor"/><text x="55" y="49">sync</text>
    <rect x="90" y="30" width="60" height="30" fill="none" stroke="currentColor"/><text x="120" y="49">header</text>
    <rect x="150" y="30" width="110" height="30" fill="none" stroke="currentColor"/><text x="205" y="49">payload</text>
    <rect x="260" y="30" width="70" height="30" fill="currentColor" fill-opacity="0.25" stroke="currentColor"/><text x="295" y="49">sync</text>
    <rect x="330" y="30" width="60" height="30" fill="none" stroke="currentColor"/><text x="360" y="49">header</text>
    <rect x="390" y="30" width="110" height="30" fill="none" stroke="currentColor"/><text x="445" y="49">payload</text>
  </g>
  <text x="260" y="78" text-anchor="middle" font-size="9" fill="currentColor">one frame &#8594;| |&#8592; next frame</text>
</svg>
<figcaption>Framing: each frame opens with a known sync word so the receiver can lock onto frame boundaries and locate every field inside.</figcaption>
</figure>

Once aligned, the fixed layout tells the decoder where the header, addresses, and payload
sit — the boundary where DSP ends and [protocol decoding](/learn/digital-trunking/anatomy-of-a-call/)
begins.

## Interleaving: defeating burst errors

Radio errors rarely come one at a time. A fade or a burst of interference corrupts a run
of *consecutive* bits — and a long burst can overwhelm any error-correcting code, which
can only fix so many errors in one block. **Interleaving** is the clever countermeasure:
shuffle the bit order before transmission and unshuffle it at the receiver.

```text
sent order (interleaved):   1 5 9 2 6 10 3 7 11 4 8 12
a burst hits 3 in a row:        X X  X
de-interleaved back to order: 1 2 3 4 5 6 7 8 9 10 11 12
the 3 errors are now:             X     X        X   (spread out)
```

A tight burst becomes scattered single errors after de-interleaving — and *scattered*
single errors are exactly what FEC handles well.

## Forward error correction: repair without a resend

Broadcast radio has no back-channel; the receiver can't ask "say that again." So the
transmitter sends **redundancy up front** — **forward error correction**. Extra,
mathematically related bits let the receiver detect *and reconstruct* a bounded number of
flipped bits by itself. Two families dominate:

| Scheme | Style | Good at |
|--------|-------|---------|
| **Convolutional** (Viterbi-decoded) | continuous, over a sliding window | scattered random bit errors |
| **Reed–Solomon** | block, over groups of symbols | correcting whole bad symbols/short bursts |

Systems often **stack** them — Reed–Solomon over a convolutional inner code, with
interleaving between — so each mops up what the other misses. The payoff shows in the
metrics: FEC lets a raw [BER](/learn/dsp/snr-evm-and-ber/) of, say, one in a thousand
decode to *zero* errors, which is why a link can sound clean well below the SNR at which
the raw symbols are perfect.

## The pipeline in order

Putting it together, the receive side is: recover symbols → find the **sync word** →
**de-interleave** → **FEC decode** → trustworthy frame. That last output is what the
[digital trunking](/learn/digital-trunking/) module parses into calls, talkgroups, and
control messages.

<div class="knowledge-check" data-quiz data-correct-msg="Right — interleaving spreads a burst of errors into scattered single errors that FEC can fix." markdown="0">
  <p class="knowledge-check__q">Quick check: what does interleaving accomplish?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">It marks where each frame begins</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">It spreads a burst of consecutive errors into scattered single errors FEC can correct</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">It increases the raw symbol rate</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Framing** gives bits structure; a **sync word** marks where each frame begins.
- **Interleaving** shuffles bit order so a **burst** becomes scattered single errors.
- **FEC** (convolutional/Viterbi, Reed–Solomon) **repairs** flipped bits with no resend.
- The chain — sync → de-interleave → FEC — turns a nonzero **BER** into trustworthy frames.

Next up: making all of this run live — block processing, ring buffers, and latency.
