---
slug: symbols-and-baud
title: Symbols, baud & bitrate
description: The difference between baud (symbol rate) and bitrate explained simply — why a 4800-baud 4FSK signal like P25 carries 9600 bits per second, how bits-per-symbol works, and what symbol rate means for SDR capture and SNR.
keywords: baud rate, symbol rate, bitrate, bits per symbol, baud vs bitrate, 4FSK 4800 baud, P25 9600 bps, symbol rate explained
level: intermediate
status: full
prereq:
  - digital-modulation
faq:
  - q: What is the difference between baud and bitrate?
    a: Baud (the symbol rate) is how many symbols are sent per second. Bitrate is how many bits per second. They're equal only when each symbol carries one bit. When a symbol carries multiple bits — as in 4-level modulation — the bitrate is higher than the baud. The formula is bitrate = baud × bits per symbol.
  - q: Why does a 4800-baud P25 signal carry 9600 bits per second?
    a: P25 Phase 1 uses C4FM, a four-level modulation, so each symbol represents 2 bits (four states = 2 bits). At 4800 symbols per second, that's 4800 × 2 = 9600 bits per second. The symbol rate is 4800 baud but the bitrate is 9600 bps.
  - q: Does a higher symbol rate need more bandwidth?
    a: Yes. Roughly, the faster you send symbols, the more bandwidth the signal occupies. To carry more data you can either raise the symbol rate (more bandwidth) or pack more bits into each symbol (more states, which needs a higher SNR to tell the states apart). It's always a trade-off between bandwidth, data rate, and required signal quality.
  - q: Why does symbol rate matter for capturing a signal with an SDR?
    a: Your SDR has to sample fast enough and capture enough bandwidth to represent every symbol cleanly, and the signal needs enough SNR for the demodulator to distinguish the states. Knowing a system's symbol rate tells you the minimum bandwidth to capture and helps you reason about why a marginal signal drops symbols.
gophertrunk_links:
  - title: Symbol scope
    url: /symbol-scope.html
    note: watch recovered symbols stream past at the system's symbol rate.
  - title: Status
    url: /status.html
    note: the protocols (and their rates) GopherTrunk decodes.
---

# Symbols, baud & bitrate

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Baud** is the **symbol rate** — symbols per second. **Bitrate** is **bits per
second**. They're the same only when each symbol carries one bit; with multi-level
modulation a symbol carries several bits, so **bitrate = baud × bits-per-symbol**.
P25's C4FM runs at **4800 baud** with **4 states (2 bits each) = 9600 bps**. Raising
the symbol rate widens the signal's bandwidth; packing more bits per symbol needs more
[SNR](/learn/decibels/). Knowing a system's symbol rate tells you how much bandwidth to
capture.
</div>

This is a short, sharp distinction that clears up a surprising amount of confusion —
and it's the reason "4800" and "9600" both get quoted for the same P25 signal.

## Symbols vs. bits, revisited

From [digital modulation](/learn/digital-modulation/): a **bit** is a single 0 or 1; a
**symbol** is one state the transmitter actually puts on the air, and a symbol can
encode more than one bit. The number of bits per symbol depends on how many states the
modulation has:

| States | Bits per symbol | Example |
|--------|-----------------|---------|
| 2 | 1 | 2FSK, BPSK |
| 4 | 2 | 4FSK / C4FM (P25, DMR), QPSK |
| 8 | 3 | 8PSK |
| 16 | 4 | 16-QAM |

More states pack more bits into each symbol — but the states sit closer together, so
the receiver needs a cleaner signal (more SNR) to tell them apart.

## Baud (symbol rate) defined

**Baud** is simply how many symbols are transmitted each second. A signal "at
4800 baud" sends 4800 distinct states per second, whatever each state means. Baud is a
property of the *channel timing* — it's the metronome the
[symbol scope](/symbol-scope.html) is ticking to.

## The bitrate formula

Put the two together and you get the only formula you need here:

> **bitrate (bps) = baud (symbols/s) × bits-per-symbol**

So the same signal can be described two ways without contradiction: its **symbol rate**
(baud) and its **bitrate** (bps). They diverge whenever there's more than one bit per
symbol.

## Worked example: P25 and DMR

P25 Phase 1 uses **C4FM**, a four-level [FSK](/learn/digital-modulation/):

- 4 states → **2 bits per symbol**
- Symbol rate → **4800 baud**
- Bitrate → 4800 × 2 = **9600 bps**

That's why you'll see both "4800 symbols/s" and "9600 bps" quoted for P25 — they're
the symbol rate and the bitrate of the very same signal. DMR likewise uses 4-level FSK
at 4800 baud, organised into two time slots. When someone says a system is "9600,"
they usually mean its bitrate; "4800" is its baud.

## Why symbol rate matters for capture

Two practical consequences for SDR work:

1. **Bandwidth.** Faster symbols spread the signal wider. The symbol rate sets a floor
   on how much [bandwidth](/learn/sample-rate-nyquist/) you must capture to represent
   every symbol cleanly — capture too little and you mangle the edges of the signal.
2. **SNR and reliability.** The demodulator has to land a decision on *each* symbol. As
   the signal weakens or smears (multipath, low SNR), it starts mis-judging symbols —
   first a few errors that forward error correction hides, then audible drops. A
   four-level signal is harder to keep clean than a two-level one because its states are
   closer together.

Knowing the rate, in other words, tells you both the *minimum bandwidth* to set and
*why* a marginal signal degrades the way it does — groundwork for
[tuning a clean lock](/learn/tuning-with-scopes/).

<div class="knowledge-check" data-quiz data-correct-msg="Correct — 4 states = 2 bits/symbol, so 9600 × ... no: 4800 baud × 2 = 9600 bps." markdown="0">
  <p class="knowledge-check__q">Quick check: a 4-level signal runs at 3600 baud. What's its bitrate?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">1800 bps</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">3600 bps</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">7200 bps</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Baud** = symbols/second; **bitrate** = bits/second.
- They differ when a symbol carries more than one bit: **bitrate = baud × bits/symbol**.
- P25/DMR: **4800 baud × 2 bits = 9600 bps**.
- Faster symbols → more **bandwidth**; more bits/symbol → more **SNR** needed.
- The symbol rate tells you the minimum capture bandwidth and explains degradation.

That completes Module 2. Next module opens up the SDR itself — starting with what
software-defined radio is and how the receiver turns waves into samples.
