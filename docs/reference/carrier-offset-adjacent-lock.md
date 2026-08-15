---
slug: carrier-offset-adjacent-lock
title: Carrier offset & adjacent-channel lock
entry_type: term
category: fn-diagnostics
description: "Adjacent-channel lock is the failure mode where GopherTrunk locks a compatible carrier inside the channel passband while reporting the configured frequency, so a carrier offset near the channel spacing means the decoded data belongs to an adjacent site."
keywords: carrier offset, adjacent channel, offset_hz, wrong site, 12.5 khz, carrier_offset_warn_hz, control channel, afc, decode quality, tsbk error rate
aka: [adjacent-channel lock, wrong-site lock, offset_hz fingerprint]
infobox:
  - { label: Type, value: Failure mode }
  - { label: Fingerprint, value: "offset_hz near the channel spacing — ±12.5 kHz for narrowband P25" }
  - { label: Detection, value: sdr.carrier_offset_warn_hz (default 4000 Hz) }
  - { label: Consequence, value: Decoded site identity belongs to the neighbour }
see_also: [signal-signatures, diagnostic-playbook, automatic-frequency-control, control-channel, c4fm, bit-error-rate, dbfs]
related_reading:
  - { title: "From the Issue Tracker, Part 22: Two Pipelines, One Symptom — When Parallel Code Paths Drift", url: /blog/solution-postmortem/from-the-issue-tracker-22-two-pipelines/ }
cite_urls:
  - https://github.com/MattCheramie/GopherTrunk/issues/815
  - https://github.com/MattCheramie/GopherTrunk/pull/866
  - https://github.com/MattCheramie/GopherTrunk/issues/858
---

**Adjacent-channel lock** is the failure mode where GopherTrunk decodes a
*neighbouring* channel's signal while reporting it under the *configured*
frequency. The receiver tunes its down-converter so the configured frequency sits
at 0 Hz, then locks whatever compatible [C4FM](/reference/c4fm/) carrier the
matched filter finds inside the roughly ±24 kHz channel passband — and every
downstream event, site record, and log line carries the configured frequency, not
the carrier actually decoded. At 12.5 kHz channel spacing, a strong neighbour one
step away sits comfortably inside that passband. The result is silent wrong-site
data: internally consistent, repeatable, and wrong.

## The fingerprint

The reported carrier offset (`offset_hz`) sitting near the channel spacing —
**±12.5 kHz for narrowband P25** — is the fingerprint. Crystal error on a
reasonable dongle is 1–3 kHz; a TCXO front end after autotune leaves a sub-kHz
residual. Nothing legitimate parks at exactly one channel step.

In [#815](https://github.com/MattCheramie/GopherTrunk/issues/815), a control
channel configured for Geelong at 420.0750 MHz confidently and repeatably decoded
RFSS 2 / Site 7 — Mt Anakie, whose control channel is 420.0875 MHz, one 12.5 kHz
step up. Off-pipeline `capture` + `spectrum` (rung 3 of the
[diagnostic playbook](/reference/diagnostic-playbook/)) showed **no carrier at
all** at 0 Hz offset and a dominant carrier at +12.2 kHz; once instrumented, the
live receiver reported `offset_hz` between 12,436 and 12,699 Hz. The receiver had
been measuring the true offset all along via its
[AFC](/reference/automatic-frequency-control/) — the value simply was never
inspected unless autotune was enabled.

## Detection and configuration

- **`sdr.carrier_offset_warn_hz`** — while a [control
  channel](/reference/control-channel/) is locked, GopherTrunk warns when the
  total carrier offset (autotune correction plus receiver residual) exceeds this
  magnitude. The default of 4000 Hz is chosen to sit in the gap: a good front end
  never trips it, a 1–3 kHz crystal error stays under it, and a 6.25 or 12.5 kHz
  adjacent-channel step always trips it. Raise it for a high-drift dongle. The
  warning is advisory only — it never changes tuning or decoding.
- **`control_channel_carrier_offset_hz`** on `GET /api/v1/sites` — the same total
  offset, published per site for dashboards and scripted checks.

A subtle trap from the follow-up
([#866](https://github.com/MattCheramie/GopherTrunk/pull/866)): the API field
was initially wired to the *residual* offset only, while the warning computed the
*total* (applied autotune correction + residual). With autotune enabled, the API
field would have hidden exactly the lock the warning flags. Both now report the
total. Autotune itself can never mask an adjacent lock either way — its
plausibility bound (~1.5 kHz at 420 MHz) is far below a 12.5 kHz step.

Two verification cautions:

- **The warning lives in the live daemon path only.** `gophertrunk replay` drives
  the receiver directly and does not construct the component that emits the
  warning, so replaying a capture of an adjacent lock stays silent. Judge the
  symptom in the path where it lives.
- **The decoder cannot know which site you *meant*.** The lock in
  [#815](https://github.com/MattCheramie/GopherTrunk/issues/815) was only caught
  because the user cross-referenced the decoded site identity against
  RadioReference and spectrum-licensing records and noticed it was the wrong
  site. When decoded identity matters, verify it against an external database —
  the fix is to tell the operator, not to change the DSP.

## Carrier offset is not a quality metric

The companion lesson from
[#858](https://github.com/MattCheramie/GopherTrunk/issues/858): a well-locked
carrier (near-zero offset) can still decode badly at range, so carrier offset is
a poor proxy for signal quality. Use the decode-quality fields on
`GET /api/v1/sites` instead:

| Field | Meaning |
|---|---|
| `control_channel_tsbk_error_rate` | Percentage of TSBK blocks that failed [Viterbi](/reference/viterbi-algorithm/) + [CRC](/reference/cyclic-redundancy-check/) — a **frame-error rate**, not a pre-FEC [bit-error rate](/reference/bit-error-rate/) |
| `control_channel_tsbk_count` | How many TSBKs the rate was measured over — a confidence weight; discard readings with `count < 100` |
| `control_channel_decode_quality` | The rate bucketed: `clean` ≤ 1%, `marginal` ≤ 5%, `poor` above |

All three are only populated once TSBKs actually decode. Read the two families of
fields together: a large `control_channel_carrier_offset_hz` says *you may be
decoding the wrong carrier*; a high `control_channel_tsbk_error_rate` says *the
carrier you are decoding is marginal*. Neither substitutes for the other — see
[signal-quality signatures](/reference/signal-signatures/) for the wider
symptom table.

## Provenance

- [#815](https://github.com/MattCheramie/GopherTrunk/issues/815) — adjacent-channel bleed-through at 12.5 kHz spacing; the offset fingerprint, `carrier_offset_warn_hz`, and the API field.
- [#866](https://github.com/MattCheramie/GopherTrunk/pull/866) — follow-up pull request aligning the published offset field with the warning's total-offset computation.
- [#858](https://github.com/MattCheramie/GopherTrunk/issues/858) — per-site decode-quality fields and why carrier offset is a bad quality proxy.
