---
slug: troubleshooting-a-decode
title: Troubleshooting a digital decode
description: A symptom-to-cause-to-fix checklist for trunked decoding — wrong control channel, gain too low or overloading, frequency and PPM error, simulcast distortion, encryption, weak signal, or the wrong system type configured.
keywords: troubleshoot trunked decode, control channel won't lock, gain overload, PPM error, simulcast distortion, encryption silent calls, weak signal, wrong system type, decode checklist, no audio
level: advanced
status: full
prereq:
  - multisite-and-simulcast-in-practice
  - encryption-and-authentication
faq:
  - q: My trunked system won't decode — where do I start?
    a: Work a checklist in order rather than guessing. Confirm you're on the right control channel, that gain isn't too low or overloading, that frequency/PPM is correct so the signal lands on channel, and that the configured system type matches. Each failure has a distinct symptom, so matching what you see to the checklist points you straight at the cause.
  - q: Calls are listed but play back silent — what's wrong?
    a: That's almost always encryption, not a fault. If the control channel decodes and grants appear but the audio is silent or garbled, the talkgroup is likely encrypted, which can't be decoded. Check whether other talkgroups produce audio; if they do, your hardware and config are fine and the silent one is encrypted.
  - q: How do I know if my gain is wrong?
    a: Too-low gain buries the signal in noise — a weak meter and a constellation lost in the floor. Too-high gain overloads the front end, smearing the spectrum and spraying ghost signals even on a strong signal. Aim for just enough gain to lift the signal cleanly above the noise without clipping the ADC.
  - q: Can GopherTrunk decode encrypted traffic?
    a: No. Encryption scrambles the voice payload with a key only authorised radios hold, and without that key the audio cannot be recovered by any receiver. GopherTrunk will still show the grant metadata — talkgroup, radio ID, the encryption flag — but the audio stays silent. This is a fundamental limit, not a bug.
gophertrunk_links:
  - title: Demod calibration
    url: /demod-calibration.html
    note: fix lock and constellation problems on the control channel.
  - title: Voice calibration
    url: /voice-calibration.html
    note: clean up decoded audio once the system locks.
  - title: Constellation
    url: /constellation.html
    note: read at a glance which stage is breaking.
---

# Troubleshooting a digital decode

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
When a trunked system won't decode, **work a checklist, don't guess**. The usual
culprits are: monitoring the **wrong control channel**, **gain too low** (buried in noise)
or **too high** (overload), a **frequency/PPM error** that lands the signal off channel,
**simulcast distortion**, **encryption** (silent calls — *not fixable*), a signal that's
simply **too weak**, or the **wrong system type** configured. Each has a distinct symptom,
so match what you see to its cause and apply the fix. The
[scopes](/learn/rf-sdr/tuning-with-scopes/) turn most of this from guesswork into a glance.
</div>

You've assembled the whole pipeline; now here's how to find the one stage that's broken
when a system stays stubbornly silent. This builds directly on the SDR-level [calibration
and troubleshooting](/learn/rf-sdr/calibration-troubleshooting/) lesson, applied to the
trunking-specific failures.

## The checklist — symptom → cause → fix

Read down the **Symptom** column until one matches what you're seeing, then apply the fix:

| Symptom | Likely cause | Fix |
|---------|--------------|-----|
| No lock; [CC Activity](/cc-activity.html) stays empty | **Wrong control channel** | Re-check the frequency against a database or [Hunt](/hunt.html); try the alternate control channel |
| Signal buried in noise, weak meter | **Gain too low** or **signal too weak** | Raise [gain](/learn/rf-sdr/gain-and-agc/); improve antenna/placement |
| Smeared spectrum, ghost signals, strong meter | **Gain too high — overload** | **Reduce gain**, add attenuation |
| Strong signal, **constellation slowly rotating** | **Frequency / PPM error** | Set the correct [PPM](/learn/rf-sdr/calibration-troubleshooting/) correction |
| Strong meter but **fuzzy constellation**, won't lock | **Simulcast distortion** | Directional antenna, favour one site, lower gain ([multi-site lesson](/learn/digital-trunking/multisite-and-simulcast-in-practice/)) |
| Locks but **no calls / no grants** | **Wrong system type** or parameters | Confirm the configured protocol matches; re-check system details |
| **Calls listed but silent or garbled** | **Encryption** | Expected — [encrypted](/learn/digital-trunking/encryption-and-authentication/) talkgroup, **not fixable** |
| Audio decodes but sounds wrong | **Voice level / clarity** | [Voice calibration](/voice-calibration.html) |

The trick is discipline: change **one thing at a time** and watch the
[constellation](/constellation.html) and [CC Activity](/cc-activity.html) react, so you
know which fix actually worked.

## Walking the common cases

A few of these deserve a closer look, because they're the ones newcomers misdiagnose.

**Wrong control channel.** The most common cause of "nothing decodes." If
[CC Activity](/cc-activity.html) is empty and the carrier *looks* like a control channel
(solid, never quiet), you may be on a voice channel of the system, an alternate control
channel that's currently inactive, or simply the wrong frequency. Re-confirm against a
database or re-run [Hunt](/hunt.html). Many systems publish more than one control-channel
frequency — try the alternates.

**Gain, both ways.** Too *little* gain leaves the signal in the [noise
floor](/learn/rf-sdr/decibels/); too *much* **overloads** the front end and sprays
distortion even on a strong signal. The overload tell is *ghost signals* — copies of a
carrier appearing where no transmitter exists — and a smeared spectrum. Back gain down
until the ghosts vanish, then nudge up only as far as a clean lock needs.

**Frequency / PPM error.** A cheap dongle's oscillator error lands the signal off the
channel centre. The classic symptom is a **strong signal whose constellation slowly
rotates** and never quite locks. The fix is [PPM
correction](/learn/rf-sdr/calibration-troubleshooting/), which shifts the radio back onto
frequency. [Demod calibration](/demod-calibration.html) is where you tune this in.

**Encryption — the one you can't fix.** If the control channel decodes, grants appear,
the [radio ID](/radio-ids.html) and talkgroup show, but the audio is silent or garbled,
the talkgroup is **encrypted**. No receiver can recover encrypted voice without the key.
Confirm by checking whether *other* talkgroups on the same system produce audio — if they
do, your chain is healthy and the silent one is simply encrypted. This is a
[fundamental limit](/learn/digital-trunking/encryption-and-authentication/), not a bug to chase.

<div class="knowledge-check" data-quiz data-correct-msg="Right — grants visible but silent audio, while other talkgroups work, means the talkgroup is encrypted." markdown="0">
  <p class="knowledge-check__q">Quick check: the control channel decodes, grants and radio IDs appear, but one talkgroup's calls play back silent while others work. Most likely cause?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Gain is set too low</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">That talkgroup is encrypted</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">The wrong control channel is configured</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Suspect hardware last

As with SDR-level [troubleshooting](/learn/rf-sdr/calibration-troubleshooting/), almost
every problem is **config or signal**, not hardware. Only after the checklist comes up
empty should you try a different USB port or cable, let the dongle warm up, or test a
known-good system to prove the chain. If a system that decodes elsewhere fails only on
your dongle, *then* consider the hardware — otherwise it's antenna, gain, PPM, control
channel, or system type, roughly in that order.

## Recap

- **Work the checklist in order** — each failure has a distinct symptom.
- **Wrong control channel** is the top cause of "nothing decodes" — re-confirm or
  [Hunt](/hunt.html) it, and try alternates.
- **Gain** fails both ways: too low buries the signal, too high **overloads** and sprays
  ghosts.
- A **rotating constellation** means a **PPM/frequency** error — calibrate it.
- **Calls listed but silent** is **encryption** — expected and not fixable.
- Suspect **hardware last**, after config and signal check out.

## You've finished the path — now go use it

That's the whole journey: from why radio went digital, through trunking's architecture
and the major protocols, to **operating GopherTrunk** against real systems — identifying
them, finding the control channel, following calls end to end, handling multi-site and
simulcast, and troubleshooting when it fights you. You now have a mental model of every
stage, which is exactly what turns settings-poking into reasoning.

Keep the **[glossary](/learn/digital-trunking/glossary/)** open whenever a term needs a refresher, and head to
**[getting started](/getting-started.html)** to put all of this into practice on a real
system. Well done — you've earned the lock.
