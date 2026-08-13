---
title: "From the Issue Tracker, Part 4: The Dongle That Heard Nothing — One Line in a Register Table"
description: An RTL-SDR Blog V4 that only ever received noise survived four confident wrong diagnoses, a crystal set wrong and then wrong again, and a missing input-routing block — until the reporter read the register tables and found the one value that had to be 1.
category: solution-postmortem
keywords: rtl-sdr blog v4, r828d, r820t2, vco power reference, r82xx, crystal frequency, input routing, pll nint overflow, spectrum inversion, dmr sync polarity, gophertrunk postmortem
tags: [from-the-issue-tracker, rtl-sdr, hardware, drivers, dmr, debugging, postmortem]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "From the Issue Tracker"
series_part: 4
---

*Part 4 of **From the Issue Tracker**, postmortems of GopherTrunk bugs that fought
back. [Part 3]({{ '/blog/solution-postmortem/from-the-issue-tracker-03-phase2-encryption-metadata/' | relative_url }})
peeled four software layers off a single symptom. This one is a hardware
bring-up story with the same shape and a better ending: after four confident,
merged, wrong diagnoses, the person who fixed the RTL-SDR Blog V4 was the
reporter — by reading a register table and finding the one constant the port had
inherited wrong.*

> **TL;DR:** An RTL-SDR Blog V4 (R828D tuner) went through every stage of not
> working: I²C errors at init, a PLL overflow on tune, then — worst of all —
> apparent operation while receiving nothing but noise
> ([#264](https://github.com/MattCheramie/GopherTrunk/issues/264)). Four
> plausible root causes shipped and failed: PPM correction not reaching the tuner
> LO, spectrum-inverted IQ, a mixer-AGC gain polarity bug, and input-bank
> misrouting. Raw IQ captures kept the investigation honest (pure white noise
> means the LO is *elsewhere*, not attenuated), but the fix came from the
> reporter reading `r82xx_tables.go`: the VCO power reference must be **1** on
> the R828D, not the 2 the osmocom-derived port hardcoded. Along the way: a
> crystal constant reversed twice, a missing V4 input-routing block, and a latent
> PLL cap bug that only a correct crystal could expose.

## The symptom ladder

The report opened simply: GopherTrunk 0.1.5 doesn't detect the Blog V4; every
other program does.

```
rtlsdr: tuner init: r82xx init: burst write: rtl2832u: I2CWrite addr=0x74: broken pipe
```

That first rung was already understood — it's the same init-burst EPIPE as
[#248](https://github.com/MattCheramie/GopherTrunk/issues/248), at the R828D's
I²C address (`0x74`) instead of the R820T2's (`0x34`), and the layered defenses
shipped for that issue (chunked bursts, chip-settle delay, EPIPE retry) covered
it. Each fix revealed the next rung:

| Rung | Symptom | What it turned out to be |
|---|---|---|
| 1 | `I2CWrite addr=0x74: broken pipe` at init | The #248 init-burst timing family, fixed by existing defenses |
| 2 | `r82xx setPLL: nint=78 overflows` on tune | A latent PLL cap bug, exposed by a crystal change (below) |
| 3 | All three of the reporter's dongles claimed by the daemon; `ppm: -4` "not adopted" | `Pool.Open` ignored the config allowlist — a NooElec auto-claimed the control role, so the PPM landed on the wrong stick |
| 4 | V4 decodes nothing on DMR; a NooElec R820T2 on the **same antenna** decodes fine; "color code changes constantly" | The long tail — everything below |

Rung 4 is the interesting one, because from here the V4 *looked* alive. It
enumerated, tuned without errors, and produced samples. It just never produced
signal.

## Four confident wrong diagnoses

Each of these shipped as a fix, with a mechanism that genuinely explained the
symptom. None of them fixed it.

**1. PPM correction never reached the tuner LO.** True bug: `SetPPM` only retimed
the RTL2832U's resampler clock and never re-tuned the LO, where librtlsdr does
both. Merged. The V4 stayed deaf — a few ppm was never going to matter against
what was actually wrong.

**2. Spectrum-inverted IQ.** The reporter asked the right-shaped question — "Are
I and Q reversed?" — and the theory had teeth: a conjugated stream flips the FM
discriminator's sign, mapping every C4FM symbol to its negative, which is a
`(dibit+2) mod 4` flip. The genuinely interesting fact unearthed here: **DMR's
nine sync words are closed under that polarity flip** — an inverted *data* sync is
byte-identical to a clean *voice* sync, so the sync correlator alone structurally
cannot detect spectrum inversion; only the FEC-protected payload can. The decoder
gained dual-polarity burst decoding gated by the FEC chain. Merged. Still deaf.

**3. Mixer-AGC gain polarity.** `SetGainMode` programmed the mixer's AGC-enable
bit with the same polarity as the LNA's, where librtlsdr uses opposite polarities,
and never wrote the VGA in AGC mode — an under-gained front end, with a matching
fingerprint (low level, intermod forest). Merged, with an honest caveat attached
that if the V4 stayed deaf the theory was wrong. The reporter's reply: "The gain
fix also keep my v4 silent."

**4. Input-bank misrouting via the Lite variant flag.** Forcing `blog_v4_lite`
routed a VHF tune through the V4's UHF input bank — this guess actively made the
capture worse.

## The captures that kept everyone honest

What kept this from being an endless guessing game was ground truth: the reporter
supplied raw IQ captures at every stage, and the analysis of each one constrained
the next theory.

The first V4 capture was **pure complex white noise** — I/Q correlation ≈ 0.0001,
flat power spectral density, no carrier anywhere in the ±1.1 MHz passband. That
single measurement carries a hard conclusion: a blocked or under-gained front end
still shows a *weak* copy of a strong carrier. Total absence means **the LO is
tuned somewhere else entirely** — not attenuation, mistuning.

That insight drove the crystal work, which reversed itself twice. The first
change set *every* R828D to the generic 16 MHz reference crystal, per librtlsdr's
`R828D_XTAL_FREQ`. But the rtlsdr-blog fork — the V4's actual driver — keeps the
**V4 on 28.8 MHz** and applies 16 MHz only to non-V4 R828D dongles. A wrong
crystal is fatal arithmetic: every LO lands at 28.8/16 = 1.8× the requested
frequency, listening near 276 MHz when asked for 153. The follow-up (PR
[#506](https://github.com/MattCheramie/GopherTrunk/pull/506)) restored 28.8 MHz
for the V4 and added the second missing piece: the V4's **switched HF/VHF/UHF
input bank**. Stock R828D init leaves both Cable-1 and Air-In switched *off* — the
V4 routes no RF at all without per-band switching on registers `0x05`/`0x06`, the
notch on `0x17`, and the GPIO5 upconverter relay for HF.

After #506 the captures changed character: real signal energy, a 28 dB
peak-to-floor. Progress — the front end was finally connected to the antenna. But
side-by-side captures against the NooElec ground truth still showed the one
carrier that mattered missing:

| | NooElec R820T2 (works) | Blog V4 (fails) |
|---|---|---|
| DMR carrier at 153.139 MHz | +41.6 dB over floor, decodes | absent — at the noise floor |
| Overall level | +1.1 dB | ~17 dB lower, intermod forest |
| `replay -protocol dmr` | BS-Voice sync, valid FEC frames | 0 sync, 0 frames |

The "color code changes constantly" symptom, meanwhile, was never a signal
property at all — it was the decoder false-locking on noise, the slot-type
Hamming code dutifully "correcting" garbage to an ever-changing color code.

## The reporter reads the table

With detection confirmed working (`blog_v4=true ref_xtal_hz=28800000` in the boot
log) and the gain theory dead, the reporter went into the source:

> I found the solution. In line 56 of `internal/sdr/rtlsdr/tuners/r82xx_tables.go`,
> `r82xxVCOPowerRef` must be set to `1` for the Blog V4; that makes it work.

GopherTrunk's `setPLL` was a port of **osmocom** librtlsdr, which hardcodes
`VCO_POWER_REF = 2` for every chip. The **rtlsdr-blog** fork overrides it:

```c
uint8_t vco_power_ref = 2;
...
if (priv->cfg->rafael_chip == CHIP_R828D ||
    rtlsdr_check_dongle_model(priv->rtl_dev, "RTLSDRBlog", "Blog V4L"))
    vco_power_ref = 1;
```

With the wrong reference, the VCO fine-tune step nudged the mixer divider the
wrong way and mistuned the LO — the wanted carrier landed out of band, and the V4
heard only noise. Exactly what every capture had been saying. The fix is a
per-chip `vcoPowerRef()` — 1 for R828D, 2 for R820T/R820T2 — leaving the working
NooElec path byte-for-byte unchanged, plus a regression test pinning the per-chip
threshold. The reporter rebuilt: "After rebuild with latest main the V4 works
with gophertrunk."

## The latent bug the correct crystal exposed

One rung deserves its own note, because it's a pattern worth naming. When the
crystal first moved to 16 MHz, tuning 153.5875 MHz started failing with
`setPLL: nint=78 overflows`. The overflow guard was:

```go
if nint > 0x3F+13 { // = 76 — derived from ni's 6-bit field alone
```

But the real register encoding is `nint = 13 + 4*ni + si`, with `si` living in
two more bits of the same register — the true cap is **268**. With the 28.8 MHz
crystal the VCO range keeps `nint` near 67, so the wrong guard was *latent* for
every dongle ever tested. Halving the reference roughly doubled `nint`, and 78 —
perfectly encodable — tripped a limit that had never been exercised. A correct
change exposed a wrong guard: the bug was always there, waiting for the first
configuration that reached it.

## What we keep

- **Pure white noise is a diagnosis, not an absence of one.** Uncorrelated I/Q
  and a flat PSD mean the LO is elsewhere — mistuned, not attenuated. The
  distinction between "weak" and "absent" signatures is cataloged in
  [signal signatures]({{ '/reference/signal-signatures/' | relative_url }}).
- **Port against the fork the hardware actually uses.** Three of the V4's
  failures (crystal, input routing, VCO power reference) were divergences between
  osmocom librtlsdr and the rtlsdr-blog fork. A port inherits its upstream's
  blind spots.
- **A confident mechanism is not a confirmed root cause.** Four theories shipped
  with explanations that fit the symptom; the disciplined side-by-side capture
  against a known-good dongle is what falsified each one. That workflow is in the
  [diagnostic playbook]({{ '/reference/diagnostic-playbook/' | relative_url }}).
- **Correct fixes uncover latent guards.** The `nint` cap was wrong for years and
  harmless until the crystal fix doubled the operating range. Derive limits from
  the register encoding, not from one field of it.
- **Some inversions are structurally invisible.** DMR's sync words are closed
  under the polarity flip — detection has to come from the FEC-protected payload.
  Bring-up ladders like this one, from EPIPE to working RF, are traced in
  [RTL-SDR USB recovery]({{ '/reference/rtlsdr-usb-recovery/' | relative_url }}).
- **Reporters fix bugs.** The decisive read of `r82xx_tables.go` came from the
  person with the hardware on their desk. Make the source approachable; it pays
  for itself.
