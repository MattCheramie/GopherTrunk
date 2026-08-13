---
title: "From the Issue Tracker, Part 7: The LSM Myth — When Your Own Docs Are the Bug"
description: GopherTrunk's docs, config comments, and UI labels all taught that simulcast P25 sites need the CQPSK demodulator. A user proved a genuine three-tower simulcast system decodes fine in C4FM — and that forcing CQPSK kills it. The real fault was a gain value, and the real fix was rewriting our own guidance across every surface it had leaked into.
category: solution-postmortem
keywords: p25 lsm, simulcast, cqpsk vs c4fm, demod mode, emission designator, sdr gain tenths of db, per-channel override, wideband p25, gophertrunk debugging
tags: [from-the-issue-tracker, p25, simulcast, configuration, documentation, debugging, postmortem]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "From the Issue Tracker"
series_part: 7
---

*Part 7 of **From the Issue Tracker**, postmortems of GopherTrunk bugs that
fought back. [Part 6]({{ '/blog/solution-postmortem/from-the-issue-tracker-06-cqpsk-four-acts/' | relative_url }})
spent four acts making the CQPSK path actually work. This part is about the
question that path immediately raised — *which sites should use it?* — and how
GopherTrunk's own documentation answered that question wrong, confidently, on
every surface we had. The bug in
[#935](https://github.com/MattCheramie/GopherTrunk/issues/935) wasn't in the
DSP. It was in what we had taught our users to believe.*

> **TL;DR:** A user running a large multi-site P25 network asked for per-site
> demod mode selection, reasoning from licensing data that their urban simulcast
> sites were "LSM/CQPSK" while rural single-transmitter sites were C4FM. We
> shipped the feature — keyed by control-channel frequency for a hard
> chicken-and-egg reason — and then the same user disproved the premise: the
> genuinely simulcast three-tower downtown site decodes reliably in **C4FM**,
> and forcing CQPSK **kills the decode entirely**. LSM is transmitter
> *coordination*, not a modulation; an emission designator can't tell you the
> mode; and the site's actual problem was a gain value mis-entered in
> GopherTrunk's tenths-of-a-dB format. The lasting fix was correcting our own
> docs, config comments, and UI labels — which had all taught "simulcast ⇒
> CQPSK."

## Cheat sheet

| | |
|---|---|
| Issue | [#935](https://github.com/MattCheramie/GopherTrunk/issues/935) — per-site P25 Phase 1 demod mode |
| Symptom | `control_channel_decode_quality: poor` for weeks on the network's strongest site, on an Airspy R2 and an RTL-SDR alike |
| Plausible theory | urban sites are simulcast ⇒ LSM ⇒ need the CQPSK demodulator (backed by ACMA emission designators) |
| Real cause | gain entered as dB in a tenths-of-a-dB field — a starving front end; the simulcast site transmits plain C4FM |
| Fix | `gain: "363"` (≈36 dB) locked the site; per-channel `p25_phase1_demod_mode` shipped anyway ([#942](https://github.com/MattCheramie/GopherTrunk/pull/942)); guidance corrected everywhere ([#958](https://github.com/MattCheramie/GopherTrunk/pull/958), [#971](https://github.com/MattCheramie/GopherTrunk/pull/971)) |
| Rule that survives | choose `cqpsk` empirically — never infer it from "the site is simulcast" or from licensing data |

## In this post

- **The request** — a well-sourced feature ask built on licensing data and a plausible theory.
- **Shipping the feature — and why it's keyed by frequency** — the chicken-and-egg that rules out site-ID keying.
- **The reversal** — the reporter disproves their own premise on air: simulcast ≠ CQPSK.
- **The real problem was gain, in tenths of a dB** — the unit convention behind weeks of "poor."
- **Unwinding our own documentation** — auditing every surface the myth had leaked into.
- **What we keep** — durable rules for demod selection, gain units, and load-bearing docs.

## The request

The feature request was a model of its kind. The reporter operates against a
statewide P25 network — Melbourne's metropolitan radio system — and had done
real homework: regulator (ACMA) license records showed different emission
designators across the network's sites. The urban simulcast sites (Melbourne
CBD, three transmit towers; Geelong) carried `10K1D7W`, read as "LSM/CQPSK,"
while the rural single-transmitter sites (Mt Anakie, Kinglake, and others)
carried `10K1F9W`, read as C4FM.

GopherTrunk's `p25_phase1_demod_mode` was a system-level setting, so covering
the whole network on one wideband device looked impossible: the rural sites
needed C4FM, the urban ones — by this reasoning — needed CQPSK. The report even
anticipated the voice-path implication: the setting routes voice grants through
the matching chain, so a wrong mode doesn't just degrade the control channel,
it times out every call.

And there was a symptom to explain. The CBD site had shown
`control_channel_decode_quality: poor` for weeks, on an Airspy R2 and an
RTL-SDR alike. The hypothesis: decoding an LSM site with the nonlinear C4FM
demodulator was the root cause. It was plausible, well-sourced, precedented
(other scanner software supports per-channel LSM), and wrong in a way nobody
involved yet knew.

## Shipping the feature — and why it's keyed by frequency

The per-site override was a good idea regardless, and it shipped in
[#942](https://github.com/MattCheramie/GopherTrunk/pull/942). But it deliberately
does *not* key on RFSS/site identity, and the reason is worth keeping.

The reporter's proposed config shape hung the override off the `sites:` list —
"RFSS 1, site 1: cqpsk." The problem is a chicken-and-egg: **the
symbol-recovery path has to be chosen before a control channel locks** — the
right demod is precisely what *lets* it lock — but a site's RFSS and site ID
are only known *after* GopherTrunk decodes that control channel. You would need
to already be decoding the site to know which demodulator to decode it with.

The one place a site's identity exists at configuration time is the wideband
device's `channels:` list — the operator is already writing down each site's
control-channel frequency there. So that's where the override lives:

```yaml
channels:
  - frequency_hz: 420_012_500     # simulcast site
    system: "MMR"
    p25_phase1_demod_mode: cqpsk  # per-channel override
  - frequency_hz: 420_087_500     # single-TX site — inherits the system default
    system: "MMR"
```

Wiring it up also flushed out a latent bug: the wideband decode path had never
stamped *any* demod mode onto the grants it emitted, so wideband P25 voice
always ran the C4FM chain no matter what the system setting said. The override
now drives both the control-channel decode and the voice chains its grants
spawn — see [wideband voice taps]({{ '/reference/wideband-voice-taps/' | relative_url }})
for how those taps are provisioned.

The feature was left open pending the real test: would CQPSK fix the CBD site?

## The reversal

Two days later the reporter came back with the answer, and it inverted the
whole premise:

> CBD (RFSS 1/1, 3 simulcast TX sites) decodes reliably in **C4FM** mode on
> both SDRTrunk and GT. Switching either to CQPSK mode kills the decode
> entirely.

Melbourne CBD is genuinely simulcast — three transmitter sites, timing- and
phase-coordinated. And it transmits C4FM. Both facts are true at once, because
**LSM is a simulcast *coordination* technique — aligning timing and phase
across multiple transmitters — not a baseband modulation.** CQPSK (π/4-DQPSK)
is an alternative P25 Phase 1 modulation that some vendors deployed on some
systems. Simulcast deployment and CQPSK modulation are orthogonal choices, and
this network chose C4FM baseband even on its simulcast sites.

The licensing evidence dissolved under the same light: the emission designator
`10K1D7W` **covers both C4FM and CQPSK**. It describes occupied bandwidth and
emission class, not which of two 4800-baud P25 modulations is on the air.
Regulator data cannot infer demod mode — nothing but the signal itself can.

## The real problem was gain, in tenths of a dB

So what was actually wrong with the CBD site — the strongest signal in the
system, `poor` on two different SDRs?

Gain. Specifically, a unit convention. GopherTrunk expresses SDR gain in
**tenths of a dB** — `"300"` means 30 dB, and a bare `"30"` means 3 dB. The
working reference configuration in other software ran the dongle at roughly
36 dB; translated correctly into GopherTrunk's format (`gain: "363"`) on a
known-good antenna port, the CBD control channel locked immediately and pumped
around two thousand grants per five minutes. Weeks of "poor decode quality,"
explained by a starving front end — the mirror image of
[Part 8's]({{ '/blog/solution-postmortem/from-the-issue-tracker-08-nineteen-dibits/' | relative_url }})
saturated one. Both failure modes and how to spot them live in
[SDR gain and overload]({{ '/reference/sdr-gain-overload/' | relative_url }}).

There's a quieter lesson in why the modulation theory survived so long: it was
unfalsifiable *from inside the theory*. Poor decode in C4FM? That's the LSM.
Tried CQPSK and it got worse? The feature wasn't per-site yet. Only when the
per-channel override existed could the hypothesis be cleanly tested — and
cleanly killed. Sometimes you have to build the experiment before you can lose
the argument.

## Unwinding our own documentation

Here's the uncomfortable part: the reporter didn't invent "simulcast ⇒ CQPSK."
**They could have learned it from us.** Once the correction landed, an audit
found the conflation woven through every guidance surface GopherTrunk has:

- The docs and `config.example.yaml` treated LSM/simulcast as implying CQPSK
  modulation — and used Melbourne CBD, of all things, as the worked "→ cqpsk"
  example.
- The opt-in-features documentation said, verbatim, "Operators on simulcast
  sites opt into the CQPSK path per system."
- The web config-builder labeled the demod option **"CQPSK / LSM (simulcast)"**
  and its help text described "mixes LSM simulcast and C4FM sites."
- A learn article called CQPSK "the linear cousin P25 simulcast transmitters
  use" and labeled example constellations "CQPSK simulcast."

Two PRs ([#958](https://github.com/MattCheramie/GopherTrunk/pull/958), then
[#971](https://github.com/MattCheramie/GopherTrunk/pull/971) when a re-read
found three more surfaces the first pass missed) rewrote all of it around one
empirical rule:

> **C4FM is correct for most systems, including most simulcast systems. Set
> `cqpsk` only when a strong, clean signal will not lock in C4FM — determined
> by trying it, never inferred from "the site is simulcast" or from an emission
> designator.**

The feature itself — the per-channel override, and the `lsm`/`linear` →
`cqpsk` alias — was untouched. It's the right tool for genuinely-CQPSK
systems; it just needed the right trigger attached to it. The reporter reviewed
the corrected wording against what they'd proven on air, and closed the issue
themselves.

## What we keep

- **LSM is not a modulation.** Simulcast is transmitter coordination; C4FM vs
  CQPSK is baseband modulation; a system can be either × either. The full
  decision procedure is in
  [P25 demod mode selection]({{ '/reference/p25-demod-mode-selection/' | relative_url }}).
- **Emission designators can't choose your demodulator.** `10K1D7W` covers both
  P25 Phase 1 modulations. The only authority on the mode is the signal —
  choose `cqpsk` empirically, when a strong clean carrier won't lock in C4FM.
- **Check the unit convention before the DSP.** GopherTrunk gain is tenths of a
  dB; `"36"` is 3.6 dB, not 36. A translated-not-converted gain value produced
  weeks of "poor decode quality" on the strongest site in the network. See
  [SDR gain and overload]({{ '/reference/sdr-gain-overload/' | relative_url }}).
- **Key config by what's knowable at config time.** Demod mode must be chosen
  before lock, but site identity is only decoded after lock — so the override
  keys on the control-channel frequency, the one site identifier that exists in
  the config file.
- **Docs are load-bearing.** A wrong sentence in a config comment propagates
  into user hypotheses, bug reports, and forced feature requests. When guidance
  is wrong, grep *every* surface it could have leaked into — docs, example
  configs, UI labels, help text, articles — because it will have drifted into
  all of them.

## FAQ

**Does a simulcast P25 site need the CQPSK demodulator?**
No. LSM/simulcast is transmitter coordination — timing and phase alignment
across towers — not a modulation. Melbourne CBD is genuinely three-tower
simulcast and transmits C4FM; forcing CQPSK there kills the decode entirely.
Some systems do transmit CQPSK, but simulcast deployment doesn't predict it.

**How do I decide between `c4fm` and `cqpsk` for a site?**
Empirically. Start in C4FM — correct for most systems, including most simulcast
ones. Switch that channel to `cqpsk` only when a strong, clean carrier refuses
to lock in C4FM, and confirm the switch actually fixes it. The full procedure is
in [P25 demod mode selection]({{ '/reference/p25-demod-mode-selection/' | relative_url }}).

**Can I read the modulation off a license database?**
No. The emission designator `10K1D7W` covers both C4FM and CQPSK — it describes
occupied bandwidth and emission class, not which of two 4800-baud P25
modulations is on the air. Only the signal itself can answer.

**Why is the demod override keyed by control-channel frequency instead of site ID?**
Because the demodulator must be chosen *before* a control channel can lock —
the right symbol-recovery path is what makes lock possible — while RFSS and
site ID are only decoded *after* lock. The wideband `channels:` list is the one
place a site's identity exists at configuration time.

**What does `gain: "300"` actually set?**
30 dB. GopherTrunk expresses SDR gain in tenths of a dB, so `"363"` is 36.3 dB
and a bare `"36"` is 3.6 dB — a translated-not-converted value starved this
network's strongest site for weeks. See
[SDR gain and overload]({{ '/reference/sdr-gain-overload/' | relative_url }}).

## Series navigation

**Part 7 of 22** · ←
[Part 6: CQPSK in Four Acts — Fixing the Linear Path One Layer at a Time]({{ '/blog/solution-postmortem/from-the-issue-tracker-06-cqpsk-four-acts/' | relative_url }})
· Next →
[Part 8: Nineteen Dibits — A Perfect Hypothesis Meets a Rail-Pinned ADC]({{ '/blog/solution-postmortem/from-the-issue-tracker-08-nineteen-dibits/' | relative_url }})
