---
slug: wideband-voice-taps
title: Wideband voice taps
entry_type: term
category: fn-config
description: "Wideband voice taps let a single SDR decode a trunked control channel and record its voice grants at once, with a usable-window rule, an init-ordering history, and several misleading log lines worth knowing."
keywords: wideband, voice_taps, role wideband, ddc tap, voice pool, no voice device available, trellis=0, per-channel demod mode, single sdr trunking
see_also: [p25-demod-mode-selection, encrypted-call-handling, digital-down-converter, channelizer, control-channel, channel-grant, sdr-gain-overload]
---

**Wideband voice taps** are per-call [digital down-converter](/reference/digital-down-converter/)
channels carved out of one SDR's IQ capture, so a single dongle can decode a trunked
[control channel](/reference/control-channel/) *and* record the voice
[grants](/reference/channel-grant/) it issues. On a trunked system the control channel and
each voice call sit on different frequencies, so one tuner cannot do both jobs by retuning —
either you add a second SDR with `role: voice`, or you channelize: give the dongle
`role: wideband`, point `center_freq_hz` at the middle of the site's frequency cluster, and
set `voice_taps: N`. Each grant whose frequency lands inside the capture window then gets its
own on-demand DDC tap on the same IQ stream, no extra hardware needed
([#379](https://github.com/MattCheramie/GopherTrunk/issues/379)).

## The window math

A tap can only be created where there is capture to tap. The validated rule is:

- Every tapped frequency must fall within `center_freq_hz ± sample_rate/2`, with a **5%
  guard band at each edge** (the [channelizer](/reference/channelizer/) and DDC filters
  need clean skirts).
- `voice_taps: 0` disables the feature; there is no hard upper bound, but CPU scales
  roughly linearly per tap and the daemon warns above 16.
- Grants that fall *outside* the window spill over to a physical `role: voice` SDR when one
  is configured, and otherwise drop with the standard `no voice device available` log.

So a 2.4 MS/s RTL-SDR capture gives a usable span of about ±1.1 MHz around center — plan the
center frequency around where the voice channels actually are, not just the control channel.

## Misleading messages and the bugs behind them

| Symptom | Looks like | Actually | Fix / check |
| --- | --- | --- | --- |
| `voice pool full but no actives` | Pool exhausted by traffic | The pool was **empty** — the message fired only when the daemon had *zero* voice devices (a control-only config), the exact opposite of "full" ([#379](https://github.com/MattCheramie/GopherTrunk/issues/379)) | Add `voice_taps` to the wideband entry or a `role: voice` SDR; current builds emit an actionable warning instead |
| Wideband-only setup drops every grant as "no voice SDR" | Grants out of window | Init ordering: the virtual voice tuners were created *after* the voice pool was built, so the pool had zero devices; a physical voice SDR would have masked it entirely ([#422](https://github.com/MattCheramie/GopherTrunk/issues/422)) | Fixed; on old builds, adding any physical voice SDR hid the bug |
| `composer: p25p2 voice chain started trellis=0` | Config problem on your side | The wideband path never forwarded the `p25_phase2_*` FEC options to the decoder, even though the config read back correctly through the API ([#882](https://github.com/MattCheramie/GopherTrunk/issues/882)) | Fixed; `trellis=1` in that line is the confirmation the options arrived |
| Wideband P25 voice garbled on a CQPSK site | Weak signal | The wideband path stamped no demod mode onto its grants, so voice always ran the C4FM chain regardless of the setting ([#882](https://github.com/MattCheramie/GopherTrunk/issues/882), [#935](https://github.com/MattCheramie/GopherTrunk/issues/935)) | Fixed; set the per-channel `p25_phase1_demod_mode` override where needed |

The middle two rows are the same lesson twice: the wideband engine is a second, parallel
pipeline, and "the config is accepted and echoed back by the API" does not prove the value
ever reached the decoder. The observable tell is always a decoder-side log line, not the
config surface.

## Per-channel demod mode

Each `channels:` entry on a wideband dongle can carry its own `p25_phase1_demod_mode:`
override, inheriting the system default when absent. The override is keyed by **frequency**,
not by RFSS/site identity, for a structural reason: the symbol-recovery path must be chosen
*before* the control channel locks (it is what lets it lock), but the RFSS and site numbers
are only known *after* that control channel decodes — a chicken-and-egg the config resolves
by using the channel frequency, the one place site identity exists at config time
([#935](https://github.com/MattCheramie/GopherTrunk/issues/935)). See
[P25 demod-mode selection](/reference/p25-demod-mode-selection/) for how to choose the value.

## Related knobs

- `signalling_taps: N` allocates signalling-only taps that harvest P25 Phase 2
  [talker aliases](/reference/p25-talker-alias/) off granted traffic channels without
  following the call as voice — useful where voice taps rarely win a grant.
- A multi-tap wideband dongle usually wants `gain: "auto"`; a single fixed gain cannot serve
  sites of differing strength, and overload has its own signature — see
  [SDR gain and overload](/reference/sdr-gain-overload/).

## Provenance

- [#379](https://github.com/MattCheramie/GopherTrunk/issues/379) — the backwards "voice pool full" message and the one-SDR-can't-do-both explanation that motivated `voice_taps`.
- [#422](https://github.com/MattCheramie/GopherTrunk/issues/422) — wideband-only topology dropped every grant because the virtual voice tuners were built after the pool.
- [#882](https://github.com/MattCheramie/GopherTrunk/issues/882) — wideband pipeline silently dropped the P25 Phase 2 FEC options; the `trellis=0` tell.
- [#935](https://github.com/MattCheramie/GopherTrunk/issues/935) — per-channel demod-mode override and why it is keyed by frequency rather than site.
