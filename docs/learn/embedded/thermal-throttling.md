---
slug: thermal-throttling
title: Thermal throttling
description: A hot board quietly slows itself down. Learn to see throttling happening with vcgencmd and get_throttled flags, understand soft limits and hard limits, and cool your way back to full speed.
keywords: thermal throttling, vcgencmd measure_temp, get_throttled, raspberry pi overheating, cpu frequency, soft temperature limit, cooling fix, sustained load
level: intermediate
status: full
prereq:
  - cases-and-cooling
faq:
  - q: How do I know if my Raspberry Pi is throttling?
    a: "Ask the firmware: vcgencmd get_throttled returns a bit field where 0x0 means never throttled, bit 2 (0x4) means currently throttled, and bit 18 (0x40000) means throttling has occurred since boot. Pair it with vcgencmd measure_temp — sustained readings in the 80 °C region mean the soft limit is active. Because the flags include \"since boot\" bits, you can detect throttling that happened while you weren't looking."
  - q: At what temperature does a Raspberry Pi throttle?
    a: Modern Pis begin backing off the clock at a soft limit (60 °C on some models, 80 °C on others depending on firmware) and throttle hard approaching 85 °C, where the SoC protects itself aggressively. The exact numbers vary by model, but the shape is universal — gentle clock reduction first, severe reduction near the ceiling. Cooling that keeps the SoC under about 70 °C at your real sustained load leaves margin for summer.
  - q: Is thermal throttling harmful?
    a: "No — it exists to prevent harm, and the SoC is protecting itself exactly as designed. The cost is performance: a throttled CPU may run at a fraction of its rated speed. That's an inconvenience on a desktop and a correctness problem for a real-time decoder, which falls behind the radio when the clock drops. The fix is never to fight the mechanism; it's to remove the heat."
---

# Thermal throttling

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
When an SoC nears its temperature limit it **reduces its own clock speed** —
silently. Nothing crashes, no log shouts; the board just gets slower, and a
real-time decoder starts falling behind. The board will tell you if you ask:
**`vcgencmd measure_temp`** for the temperature, **`vcgencmd get_throttled`** for
flags that include *has throttled since boot* — so you can catch it after the
fact. Diagnosis is a **measure → load → measure** experiment; the fix is always
the same family: better **heatsinking, airflow, or ambient** — cooling, not
software.
</div>

Unit 5 is about the enemies of 24/7 operation, and heat leads because it's the
sneakiest: a thermal problem doesn't announce itself, it just quietly taxes your
CPU — in summer, in the attic, when you're not looking. This lesson makes the
invisible visible.

## What actually happens as the chip heats up?

Under sustained load the SoC's temperature climbs until it crosses the firmware's
**soft limit**, where the governor starts stepping the clock down — shedding heat
by doing less work per second. If temperature keeps climbing toward the **hard
limit** (~85 °C), the clock drops severely. Cool it and the clock steps back up;
the process is continuous, automatic, and invisible from the outside.

For bursty workloads this design is perfect — the sprint finishes before the heat
arrives. Your appliance is the opposite case: decoding is a **sustained** load
that holds the SoC at its equilibrium temperature forever. If that equilibrium
sits above the soft limit, your board *permanently* runs slower than you paid
for — and [Tuning for small CPUs](/learn/embedded/tuning-for-small-cpus/) will
show the decoder budget assuming full speed.

## How do you see it happening?

Two firmware queries are the whole toolkit on a Pi:

```bash
$ vcgencmd measure_temp
temp=82.1'C
$ vcgencmd get_throttled
throttled=0x20002
```

`get_throttled` is the underrated one — a bit field with *current* and *since
boot* flags:

| Bit | Meaning |
|-----|---------|
| 0x1 | Undervoltage **now** |
| 0x4 | Throttled **now** |
| 0x8 | Soft temperature limit active **now** |
| 0x10000 | Undervoltage has occurred **since boot** |
| 0x40000 | Throttling has occurred **since boot** |

`0x0` is the answer you want. The *since boot* bits are the appliance feature:
log in on Monday and learn about Saturday afternoon's throttling. Note the same
register reports **undervoltage** — the two great silent gremlins share one
instrument, and a "hot" symptom is sometimes a
[power problem](/learn/embedded/power-supplies/) wearing a disguise. Watch
temperature and clock live while the decoder runs:

```bash
$ watch -n2 'vcgencmd measure_temp; vcgencmd measure_clock arm'
```

If temperature parks near the limit and the clock reading sags below the rated
frequency under load — you're watching throttling in real time.

## How do you run the diagnosis as an experiment?

Treat it as three measurements, not vibes:

1. **Idle baseline.** A cool idle (40–55 °C) says the case isn't broken at rest.
2. **Real load, to equilibrium.** Run the actual decoder (or a stress tool) for
   15–20 minutes — temperature rises then flattens; the flat is your equilibrium.
3. **Verdict.** Equilibrium under ~70 °C: healthy, with summer margin.
   Above the soft limit with `get_throttled` non-zero: you have your answer.

The same experiment *validates a fix*: change one thing — add the heatsink, open
the vents, move the box off the sunny shelf — and re-measure equilibrium. A good
cooling change moves it 10–20 °C, which you'll see in minutes.

> Rule of thumb: every fix for throttling is spelled "remove heat" — sink,
> airflow, ambient. Reducing the workload (fewer channels, lower sample rate) is
> legitimate too, but it's paying the heat tax, not repealing it.

## How does an appliance keep this handled forever?

The one-off experiment proves today; an appliance needs the *trend*. Temperature
belongs in your routine monitoring —
[Monitoring your board](/learn/embedded/monitoring-your-board/) will chart it and
alert on it, so August can't surprise a box tuned in April. Meanwhile the
physical checklist from [Cases &amp; cooling](/learn/embedded/cases-and-cooling/)
ages: dust blankets fins, fans slow and die, the closet gains a clutter of
warm gear. When a long-healthy board starts throttling, suspect the cooling has
*changed* — a stopped fan is the classic — before suspecting the software grew
hungrier.

<div class="knowledge-check" data-quiz data-correct-msg='Right — the "since boot" bits record past events, so you can catch throttling that happened while you were away.' markdown="0">
  <p class="knowledge-check__q">Quick check: what makes <code>vcgencmd get_throttled</code> especially useful for an unattended appliance?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">It permanently disables throttling when run as root</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Its "since boot" flags reveal throttling and undervoltage that happened while nobody was watching</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">It emails the temperature to the board's manufacturer for analysis</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- Near its limit an SoC **slows itself down silently** — no crash, no log, just a
  slower board and (for real-time decoding) dropped work.
- **Sustained** loads sit at an equilibrium temperature forever; if that's above
  the soft limit, you permanently lose speed you paid for.
- **`measure_temp`** + **`get_throttled`** (with its **since-boot** bits) are the
  instruments; the same register also reports **undervoltage**.
- Diagnose by experiment: **idle baseline → real load to equilibrium → verdict**;
  validate fixes the same way.
- Fixes are **cooling** (sink, airflow, ambient); for the appliance, put
  temperature in **monitoring** and re-suspect the cooling when an old board
  starts throttling.

Next up: [SD-card wear](/learn/embedded/sd-card-wear/).
